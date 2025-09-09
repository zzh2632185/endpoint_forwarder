package proxy

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"time"

	"endpoint_forwarder/config"
	"endpoint_forwarder/internal/endpoint"
)

// RetryHandler handles retry logic with exponential backoff
type RetryHandler struct {
	config               *config.Config
	endpointManager      *endpoint.Manager
	monitoringMiddleware interface {
		RecordRetry(connID string, endpoint string)
	}
}

// NewRetryHandler creates a new retry handler
func NewRetryHandler(cfg *config.Config) *RetryHandler {
	return &RetryHandler{
		config: cfg,
	}
}

// SetEndpointManager sets the endpoint manager
func (rh *RetryHandler) SetEndpointManager(manager *endpoint.Manager) {
	rh.endpointManager = manager
}

// SetMonitoringMiddleware sets the monitoring middleware
func (rh *RetryHandler) SetMonitoringMiddleware(mm interface {
	RecordRetry(connID string, endpoint string)
}) {
	rh.monitoringMiddleware = mm
}

// Operation represents a function that can be retried, returns response and error
type Operation func(ep *endpoint.Endpoint, connID string) (*http.Response, error)

// RetryableError represents an error that can be retried with additional context
type RetryableError struct {
	Err         error
	StatusCode  int
	IsRetryable bool
	Reason      string
}

func (re *RetryableError) Error() string {
	if re.Err != nil {
		return re.Err.Error()
	}
	return fmt.Sprintf("HTTP %d", re.StatusCode)
}

// Execute executes an operation with retry and fallback logic
func (rh *RetryHandler) Execute(operation Operation, connID string) (*http.Response, error) {
	return rh.ExecuteWithContext(context.Background(), operation, connID)
}

// ExecuteWithContext executes an operation with context, retry and fallback logic with dynamic group management
func (rh *RetryHandler) ExecuteWithContext(ctx context.Context, operation Operation, connID string) (*http.Response, error) {
	var lastErr error
	var lastResp *http.Response
	var totalEndpointsAttempted int

	// Track groups that have been put into cooldown during this request
	groupsSetToCooldownThisRequest := make(map[string]bool)

	// Track groups that we've successfully processed in this request
	groupsProcessedThisRequest := make(map[string]bool)

	// Track initial configuration version to detect config changes
	initialConfigVersion := rh.endpointManager.GetConfigVersion()

	for {
	nextEndpointSelection:
		// Get healthy endpoints with real-time testing if enabled (dynamic refresh)
		var endpoints []*endpoint.Endpoint
		if rh.endpointManager.GetConfig().Strategy.Type == "fastest" && rh.endpointManager.GetConfig().Strategy.FastTestEnabled {
			endpoints = rh.endpointManager.GetFastestEndpointsWithRealTimeTest(ctx)
		} else {
			endpoints = rh.endpointManager.GetHealthyEndpoints()
		}

		if len(endpoints) == 0 {
			return nil, fmt.Errorf("no healthy endpoints available in active groups")
		}

		// Group endpoints by group name for failure tracking
		groupEndpoints := make(map[string][]*endpoint.Endpoint)
		for _, ep := range endpoints {
			groupName := ep.Config.Group
			if groupName == "" {
				groupName = "Default"
			}
			groupEndpoints[groupName] = append(groupEndpoints[groupName], ep)
		}

		// Track which groups failed completely in this iteration
		groupsFailedThisIteration := make(map[string]bool)
		endpointsTriedThisIteration := 0

		// Try each endpoint in current endpoint set
		for endpointIndex, ep := range endpoints {
			totalEndpointsAttempted++
			endpointsTriedThisIteration++

			// Add endpoint info to context for logging
			ctxWithEndpoint := context.WithValue(ctx, "selected_endpoint", ep.Config.Name)

			groupName := ep.Config.Group
			if groupName == "" {
				groupName = "Default"
			}

			slog.InfoContext(ctxWithEndpoint, fmt.Sprintf("🎯 [请求转发] 选择端点: %s (组: %s, 总尝试 %d)",
				ep.Config.Name, groupName, totalEndpointsAttempted))

			// Retry logic for current endpoint
			for attempt := 1; attempt <= rh.config.Retry.MaxAttempts; attempt++ {
				select {
				case <-ctx.Done():
					if lastResp != nil {
						lastResp.Body.Close()
					}
					return nil, ctx.Err()
				default:
				}

				// Execute operation
				resp, err := operation(ep, connID)
				if err == nil && resp != nil {
					// Check if response status code indicates success or should be retried
					retryDecision := rh.shouldRetryStatusCode(resp.StatusCode)

					if !retryDecision.IsRetryable {
						// Success or non-retryable error - return the response
						slog.InfoContext(ctxWithEndpoint, fmt.Sprintf("✅ [请求成功] 端点: %s (组: %s), 状态码: %d (总尝试 %d 个端点)",
							ep.Config.Name, groupName, resp.StatusCode, totalEndpointsAttempted))

						// Reset retry count for this group on success
						if !groupsProcessedThisRequest[groupName] {
							rh.endpointManager.GetGroupManager().ResetGroupRetry(groupName)
							groupsProcessedThisRequest[groupName] = true
						}

						return resp, nil
					}

					// Status code indicates we should retry
					slog.WarnContext(ctxWithEndpoint, fmt.Sprintf("🔄 [需要重试] 端点: %s (组: %s, 尝试 %d/%d) - 状态码: %d (%s)",
						ep.Config.Name, groupName, attempt, rh.config.Retry.MaxAttempts, resp.StatusCode, retryDecision.Reason))

					// Close the response body before retrying
					resp.Body.Close()
					lastErr = &RetryableError{
						StatusCode:  resp.StatusCode,
						IsRetryable: true,
						Reason:      retryDecision.Reason,
					}
				} else {
					// Network error or other failure
					lastErr = err
					if err != nil {
						slog.WarnContext(ctxWithEndpoint, fmt.Sprintf("❌ [网络错误] 端点: %s (组: %s, 尝试 %d/%d) - 错误: %s",
							ep.Config.Name, groupName, attempt, rh.config.Retry.MaxAttempts, err.Error()))
					}
				}

				// Don't wait after the last attempt on the current endpoint
				if attempt == rh.config.Retry.MaxAttempts {
					break
				}

				// Record retry (we're about to retry)
				if rh.monitoringMiddleware != nil && connID != "" {
					rh.monitoringMiddleware.RecordRetry(connID, ep.Config.Name)
				}

				// Calculate delay with exponential backoff
				delay := rh.calculateDelay(attempt)

				slog.InfoContext(ctxWithEndpoint, fmt.Sprintf("⏳ [等待重试] 端点: %s (组: %s) - %s后进行第%d次尝试",
					ep.Config.Name, groupName, delay.String(), attempt+1))

				// Wait before retry, but check for config updates during wait
				ticker := time.NewTicker(100 * time.Millisecond) // Check config every 100ms
				defer ticker.Stop()

				deadline := time.Now().Add(delay)
				for time.Now().Before(deadline) {
					select {
					case <-ctx.Done():
						if lastResp != nil {
							lastResp.Body.Close()
						}
						return nil, ctx.Err()
					case <-ticker.C:
						// Check if configuration has been updated
						currentConfigVersion := rh.endpointManager.GetConfigVersion()
						if currentConfigVersion != initialConfigVersion {
							slog.InfoContext(ctxWithEndpoint, fmt.Sprintf("🔄 [配置更新] 检测到配置变更，中断重试并重新选择端点"))
							// Break out of both the wait loop and the retry loop for this endpoint
							goto nextEndpointSelection
						}
					case <-time.After(time.Until(deadline)):
						// Wait completed normally
						goto waitCompleted
					}
				}
			waitCompleted:
			}

			slog.ErrorContext(ctxWithEndpoint, fmt.Sprintf("💥 [端点失败] 端点 %s (组: %s) 所有 %d 次尝试均失败",
				ep.Config.Name, groupName, rh.config.Retry.MaxAttempts))

			// Check if all endpoints in this group have been tried and failed in this iteration
			groupEndpointsCount := len(groupEndpoints[groupName])
			failedEndpointsInGroup := 0
			for _, groupEp := range groupEndpoints[groupName] {
				// Count endpoints in this group that we've already tried in this iteration
				for i := 0; i <= endpointIndex; i++ {
					if endpoints[i].Config.Name == groupEp.Config.Name {
						failedEndpointsInGroup++
						break
					}
				}
			}

			// If all endpoints in current group have failed in this iteration, mark group as failed
			if failedEndpointsInGroup == groupEndpointsCount {
				groupsFailedThisIteration[groupName] = true
			}
		}

		// After trying all endpoints in current iteration, handle failed groups
		for groupName := range groupsFailedThisIteration {
			if !groupsSetToCooldownThisRequest[groupName] {
				// Increment retry count for this group
				shouldCooldown := rh.endpointManager.GetGroupManager().IncrementGroupRetry(groupName)

				if shouldCooldown {
					slog.ErrorContext(ctx, fmt.Sprintf("❄️ [组重试超限] 组 %s 超过最大重试次数，进入冷却状态", groupName))
					rh.endpointManager.GetGroupManager().SetGroupCooldown(groupName)
					groupsSetToCooldownThisRequest[groupName] = true
				} else {
					slog.WarnContext(ctx, fmt.Sprintf("⚠️ [组失败] 组 %s 中所有端点均已失败，但未达到重试限制", groupName))
				}
			}
		}

		// Check if there are still active groups available after cooldown
		// Get fresh endpoint list to see if any new groups became active
		var newEndpoints []*endpoint.Endpoint
		if rh.endpointManager.GetConfig().Strategy.Type == "fastest" && rh.endpointManager.GetConfig().Strategy.FastTestEnabled {
			newEndpoints = rh.endpointManager.GetFastestEndpointsWithRealTimeTest(ctx)
		} else {
			newEndpoints = rh.endpointManager.GetHealthyEndpoints()
		}

		// If we have new endpoints available (from different groups), continue the retry loop
		if len(newEndpoints) > 0 && len(groupsFailedThisIteration) > 0 {
			// Check if the new endpoints are from different groups than what we just tried
			newGroupsAvailable := false
			newGroups := make(map[string]bool)
			for _, ep := range newEndpoints {
				groupName := ep.Config.Group
				if groupName == "" {
					groupName = "Default"
				}
				newGroups[groupName] = true
			}

			// Check if any new group is available that wasn't in the failed iteration
			for newGroup := range newGroups {
				if !groupsFailedThisIteration[newGroup] {
					newGroupsAvailable = true
					break
				}
			}

			if newGroupsAvailable {
				slog.InfoContext(ctx, fmt.Sprintf("🔄 [组切换] 检测到新的活跃组，继续重试 (已尝试 %d 个端点)", totalEndpointsAttempted))
				continue // Continue outer loop with fresh endpoint list
			}
		}

		// No more groups available, break the retry loop
		break
	}

	slog.ErrorContext(ctx, fmt.Sprintf("💥 [全部失败] 所有活跃组均不可用 - 总共尝试了 %d 个端点 - 最后错误: %v",
		totalEndpointsAttempted, lastErr))
	return nil, fmt.Errorf("all active groups exhausted after trying %d endpoints, last error: %w", totalEndpointsAttempted, lastErr)
}

// calculateDelay calculates the delay for exponential backoff
func (rh *RetryHandler) calculateDelay(attempt int) time.Duration {
	// Calculate exponential backoff: base_delay * (multiplier ^ (attempt - 1))
	multiplier := math.Pow(rh.config.Retry.Multiplier, float64(attempt-1))
	delay := time.Duration(float64(rh.config.Retry.BaseDelay) * multiplier)

	// Cap at maximum delay
	if delay > rh.config.Retry.MaxDelay {
		delay = rh.config.Retry.MaxDelay
	}

	return delay
}

// shouldRetryStatusCode determines if an HTTP status code should trigger a retry
func (rh *RetryHandler) shouldRetryStatusCode(statusCode int) *RetryableError {
	switch {
	case statusCode >= 200 && statusCode < 400:
		// 2xx Success and 3xx Redirects - don't retry
		return &RetryableError{
			StatusCode:  statusCode,
			IsRetryable: false,
			Reason:      "请求成功",
		}
	case statusCode == 400:
		// 400 Bad Request - should retry (could be temporary issue)
		return &RetryableError{
			StatusCode:  statusCode,
			IsRetryable: true,
			Reason:      "请求格式错误",
		}
	case statusCode == 401:
		// 401 Unauthorized - don't retry (auth issue)
		return &RetryableError{
			StatusCode:  statusCode,
			IsRetryable: false,
			Reason:      "身份验证失败，不重试",
		}
	case statusCode == 403:
		// 403 Forbidden - should retry (permission issue)
		return &RetryableError{
			StatusCode:  statusCode,
			IsRetryable: true,
			Reason:      "权限不足，重试中",
		}
	case statusCode == 404:
		// 404 Not Found - don't retry (resource doesn't exist)
		return &RetryableError{
			StatusCode:  statusCode,
			IsRetryable: false,
			Reason:      "资源不存在，不重试",
		}
	case statusCode == 429:
		// 429 Too Many Requests - should retry
		return &RetryableError{
			StatusCode:  statusCode,
			IsRetryable: true,
			Reason:      "请求频率过高",
		}
	case statusCode >= 400 && statusCode < 500:
		// Other 4xx Client Errors - don't retry by default
		return &RetryableError{
			StatusCode:  statusCode,
			IsRetryable: false,
			Reason:      "客户端错误，不重试",
		}
	case statusCode >= 500 && statusCode < 600:
		// 5xx Server Errors - should retry
		return &RetryableError{
			StatusCode:  statusCode,
			IsRetryable: true,
			Reason:      "服务器错误",
		}
	default:
		// Unknown status code - don't retry by default
		return &RetryableError{
			StatusCode:  statusCode,
			IsRetryable: false,
			Reason:      "未知状态码",
		}
	}
}

// IsRetryableError determines if an error should trigger a retry
func (rh *RetryHandler) IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Handle RetryableError type
	if retryErr, ok := err.(*RetryableError); ok {
		return retryErr.IsRetryable
	}

	// Add logic to determine which errors are retryable
	// For now, we retry all errors except context cancellation
	if err == context.Canceled || err == context.DeadlineExceeded {
		return false
	}

	// Network errors, timeout errors etc. should be retried
	errorStr := strings.ToLower(err.Error())
	if strings.Contains(errorStr, "timeout") ||
		strings.Contains(errorStr, "connection refused") ||
		strings.Contains(errorStr, "connection reset") ||
		strings.Contains(errorStr, "no such host") ||
		strings.Contains(errorStr, "network unreachable") {
		return true
	}

	return true
}

// UpdateConfig updates the retry handler configuration
func (rh *RetryHandler) UpdateConfig(cfg *config.Config) {
	rh.config = cfg
}
