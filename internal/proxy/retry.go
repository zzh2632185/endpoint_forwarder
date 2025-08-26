package proxy

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"endpoint_forwarder/config"
	"endpoint_forwarder/internal/endpoint"
)

// RetryHandler handles retry logic with exponential backoff
type RetryHandler struct {
	config          *config.Config
	endpointManager *endpoint.Manager
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

// Operation represents a function that can be retried
type Operation func(ep *endpoint.Endpoint) error

// Execute executes an operation with retry and fallback logic
func (rh *RetryHandler) Execute(operation Operation) error {
	return rh.ExecuteWithContext(context.Background(), operation)
}

// ExecuteWithContext executes an operation with context, retry and fallback logic
func (rh *RetryHandler) ExecuteWithContext(ctx context.Context, operation Operation) error {
	// Get healthy endpoints with real-time testing if enabled
	var endpoints []*endpoint.Endpoint
	if rh.endpointManager.GetConfig().Strategy.Type == "fastest" && rh.endpointManager.GetConfig().Strategy.FastTestEnabled {
		endpoints = rh.endpointManager.GetFastestEndpointsWithRealTimeTest(ctx)
	} else {
		endpoints = rh.endpointManager.GetHealthyEndpoints()
	}
	
	if len(endpoints) == 0 {
		return fmt.Errorf("no healthy endpoints available")
	}

	var lastErr error
	
	// Try each endpoint
	for endpointIndex, ep := range endpoints {
		// Add endpoint info to context for logging
		ctxWithEndpoint := context.WithValue(ctx, "selected_endpoint", ep.Config.Name)
		
		slog.InfoContext(ctxWithEndpoint, fmt.Sprintf("🎯 [请求转发] 选择端点: %s (尝试 %d/%d)", 
			ep.Config.Name, endpointIndex+1, len(endpoints)))
		
		// Retry logic for current endpoint
		for attempt := 1; attempt <= rh.config.Retry.MaxAttempts; attempt++ {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			// Execute operation
			err := operation(ep)
			if err == nil {
				// Success
				if attempt > 1 || endpointIndex > 0 {
					slog.InfoContext(ctxWithEndpoint, fmt.Sprintf("✅ [请求成功] 端点: %s (重试 %d次后成功)", 
						ep.Config.Name, attempt-1))
				}
				return nil
			}

			lastErr = err
			slog.WarnContext(ctxWithEndpoint, fmt.Sprintf("❌ [请求失败] 端点: %s (尝试 %d/%d) - 错误: %s", 
				ep.Config.Name, attempt, rh.config.Retry.MaxAttempts, err.Error()))

			// Don't wait after the last attempt on the last endpoint
			if attempt == rh.config.Retry.MaxAttempts {
				break
			}

			// Calculate delay with exponential backoff
			delay := rh.calculateDelay(attempt)
			
			slog.DebugContext(ctxWithEndpoint, fmt.Sprintf("⏳ [等待重试] 端点: %s - %s后进行第%d次尝试", 
				ep.Config.Name, delay.String(), attempt+1))

			// Wait before retry
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				// Continue to next attempt
			}
		}

		slog.ErrorContext(ctxWithEndpoint, fmt.Sprintf("💥 [端点失败] 端点 %s 所有 %d 次尝试均失败", 
			ep.Config.Name, rh.config.Retry.MaxAttempts))

		// If this isn't the last endpoint, log fallback
		if endpointIndex < len(endpoints)-1 {
			slog.InfoContext(ctxWithEndpoint, fmt.Sprintf("🔄 [切换端点] 从 %s 切换到 %s", 
				ep.Config.Name, endpoints[endpointIndex+1].Config.Name))
		}
	}

	slog.ErrorContext(ctx, fmt.Sprintf("💥 [全部失败] 所有 %d 个端点均不可用 - 最后错误: %s", 
		len(endpoints), lastErr.Error()))
	return fmt.Errorf("all endpoints failed after retries, last error: %w", lastErr)
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

// IsRetryableError determines if an error should trigger a retry
func (rh *RetryHandler) IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Add logic to determine which errors are retryable
	// For now, we retry all errors except context cancellation
	if err == context.Canceled || err == context.DeadlineExceeded {
		return false
	}

	return true
}