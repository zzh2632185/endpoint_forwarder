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
	config          *config.Config
	endpointManager *endpoint.Manager
	monitoringMiddleware interface{
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
func (rh *RetryHandler) SetMonitoringMiddleware(mm interface{
	RecordRetry(connID string, endpoint string)
}) {
	rh.monitoringMiddleware = mm
}

// Operation represents a function that can be retried, returns response and error
type Operation func(ep *endpoint.Endpoint, connID string) (*http.Response, error)

// RetryableError represents an error that can be retried with additional context
type RetryableError struct {
	Err        error
	StatusCode int
	IsRetryable bool
	Reason     string
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

// ExecuteWithContext executes an operation with context, retry and fallback logic with group management
func (rh *RetryHandler) ExecuteWithContext(ctx context.Context, operation Operation, connID string) (*http.Response, error) {
	// Get healthy endpoints with real-time testing if enabled
	var endpoints []*endpoint.Endpoint
	if rh.endpointManager.GetConfig().Strategy.Type == "fastest" && rh.endpointManager.GetConfig().Strategy.FastTestEnabled {
		endpoints = rh.endpointManager.GetFastestEndpointsWithRealTimeTest(ctx)
	} else {
		endpoints = rh.endpointManager.GetHealthyEndpoints()
	}
	
	if len(endpoints) == 0 {
		return nil, fmt.Errorf("no healthy endpoints available in active groups")
	}

	var lastErr error
	var lastResp *http.Response
	
	// Group endpoints by group name for failure tracking
	groupEndpoints := make(map[string][]*endpoint.Endpoint)
	for _, ep := range endpoints {
		groupName := ep.Config.Group
		if groupName == "" {
			groupName = "Default"
		}
		groupEndpoints[groupName] = append(groupEndpoints[groupName], ep)
	}
	
	// Track failed groups
	failedGroups := make(map[string]bool)
	
	// Try each endpoint
	for endpointIndex, ep := range endpoints {
		// Add endpoint info to context for logging
		ctxWithEndpoint := context.WithValue(ctx, "selected_endpoint", ep.Config.Name)
		
		groupName := ep.Config.Group
		if groupName == "" {
			groupName = "Default"
		}
		
		slog.InfoContext(ctxWithEndpoint, fmt.Sprintf("🎯 [请求转发] 选择端点: %s (组: %s, 尝试 %d/%d)", 
			ep.Config.Name, groupName, endpointIndex+1, len(endpoints)))
		
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
					if attempt > 1 || endpointIndex > 0 {
						slog.InfoContext(ctxWithEndpoint, fmt.Sprintf("✅ [请求成功] 端点: %s (组: %s), 状态码: %d (重试 %d次后成功)", 
							ep.Config.Name, groupName, resp.StatusCode, attempt-1))
					} else {
						slog.InfoContext(ctxWithEndpoint, fmt.Sprintf("✅ [请求成功] 端点: %s (组: %s), 状态码: %d", 
							ep.Config.Name, groupName, resp.StatusCode))
					}
					return resp, nil
				}
				
				// Status code indicates we should retry
				slog.WarnContext(ctxWithEndpoint, fmt.Sprintf("🔄 [需要重试] 端点: %s (组: %s, 尝试 %d/%d) - 状态码: %d (%s)", 
					ep.Config.Name, groupName, attempt, rh.config.Retry.MaxAttempts, resp.StatusCode, retryDecision.Reason))
				
				// Close the response body before retrying
				resp.Body.Close()
				lastErr = &RetryableError{
					StatusCode: resp.StatusCode,
					IsRetryable: true,
					Reason: retryDecision.Reason,
				}
			} else {
				// Network error or other failure
				lastErr = err
				if err != nil {
					slog.WarnContext(ctxWithEndpoint, fmt.Sprintf("❌ [网络错误] 端点: %s (组: %s, 尝试 %d/%d) - 错误: %s", 
						ep.Config.Name, groupName, attempt, rh.config.Retry.MaxAttempts, err.Error()))
				}
			}

			// Don't wait after the last attempt on the last endpoint
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

			// Wait before retry
			select {
			case <-ctx.Done():
				if lastResp != nil {
					lastResp.Body.Close()
				}
				return nil, ctx.Err()
			case <-time.After(delay):
				// Continue to next attempt
			}
		}

		slog.ErrorContext(ctxWithEndpoint, fmt.Sprintf("💥 [端点失败] 端点 %s (组: %s) 所有 %d 次尝试均失败", 
			ep.Config.Name, groupName, rh.config.Retry.MaxAttempts))

		// Mark endpoint's group as failed
		failedGroups[groupName] = true
		
		// Check if all endpoints in this group have been tried and failed
		groupEndpointsCount := len(groupEndpoints[groupName])
		failedEndpointsInGroup := 0
		for _, groupEp := range groupEndpoints[groupName] {
			// Count endpoints in this group that we've already tried
			for i := 0; i <= endpointIndex; i++ {
				if endpoints[i].Config.Name == groupEp.Config.Name {
					failedEndpointsInGroup++
					break
				}
			}
		}
		
		// If all endpoints in current group have failed, put group in cooldown
		if failedEndpointsInGroup == groupEndpointsCount {
			slog.WarnContext(ctxWithEndpoint, fmt.Sprintf("❄️ [组失败] 组 %s 中所有端点均已失败，将组设置为冷却状态", groupName))
			rh.endpointManager.GetGroupManager().SetGroupCooldown(groupName)
		}

		// If this isn't the last endpoint, log fallback
		if endpointIndex < len(endpoints)-1 {
			nextGroupName := endpoints[endpointIndex+1].Config.Group
			if nextGroupName == "" {
				nextGroupName = "Default"
			}
			slog.InfoContext(ctxWithEndpoint, fmt.Sprintf("🔄 [切换端点] 从 %s (组: %s) 切换到 %s (组: %s)", 
				ep.Config.Name, groupName, endpoints[endpointIndex+1].Config.Name, nextGroupName))
		}
	}

	slog.ErrorContext(ctx, fmt.Sprintf("💥 [全部失败] 活跃组中所有 %d 个端点均不可用 - 最后错误: %v", 
		len(endpoints), lastErr))
	return nil, fmt.Errorf("all endpoints in active groups failed after retries, last error: %w", lastErr)
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
		// 403 Forbidden - don't retry (permission issue)
		return &RetryableError{
			StatusCode:  statusCode,
			IsRetryable: false,
			Reason:      "权限不足，不重试",
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