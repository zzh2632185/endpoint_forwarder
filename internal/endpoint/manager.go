package endpoint

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"sync"
	"time"

	"endpoint_forwarder/config"
)

// EndpointStatus represents the health status of an endpoint
type EndpointStatus struct {
	Healthy       bool
	LastCheck     time.Time
	ResponseTime  time.Duration
	ConsecutiveFails int
}

// Endpoint represents an endpoint with its configuration and status
type Endpoint struct {
	Config config.EndpointConfig
	Status EndpointStatus
	mutex  sync.RWMutex
}

// Manager manages endpoints and their health status
type Manager struct {
	endpoints  []*Endpoint
	config     *config.Config
	client     *http.Client
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	fastTester *FastTester
}

// NewManager creates a new endpoint manager
func NewManager(cfg *config.Config) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	
	manager := &Manager{
		config: cfg,
		client: &http.Client{
			Timeout: cfg.Health.Timeout,
		},
		ctx:        ctx,
		cancel:     cancel,
		fastTester: NewFastTester(cfg),
	}

	// Initialize endpoints
	for _, endpointCfg := range cfg.Endpoints {
		endpoint := &Endpoint{
			Config: endpointCfg,
			Status: EndpointStatus{
				Healthy:   true, // Start optimistic
				LastCheck: time.Now(),
			},
		}
		manager.endpoints = append(manager.endpoints, endpoint)
	}

	return manager
}

// Start starts the health checking routine
func (m *Manager) Start() {
	m.wg.Add(1)
	go m.healthCheckLoop()
}

// Stop stops the health checking routine
func (m *Manager) Stop() {
	m.cancel()
	m.wg.Wait()
}

// GetHealthyEndpoints returns a list of healthy endpoints based on strategy
func (m *Manager) GetHealthyEndpoints() []*Endpoint {
	var healthy []*Endpoint
	
	for _, endpoint := range m.endpoints {
		endpoint.mutex.RLock()
		if endpoint.Status.Healthy {
			healthy = append(healthy, endpoint)
		}
		endpoint.mutex.RUnlock()
	}

	// Sort based on strategy
	switch m.config.Strategy.Type {
	case "priority":
		sort.Slice(healthy, func(i, j int) bool {
			return healthy[i].Config.Priority < healthy[j].Config.Priority
		})
	case "fastest":
		// Log endpoint latencies for fastest strategy
		if len(healthy) > 1 {
			slog.Info("📊 [Fastest Strategy] 基于健康检查的端点延迟排序:")
			for _, ep := range healthy {
				ep.mutex.RLock()
				responseTime := ep.Status.ResponseTime
				ep.mutex.RUnlock()
				slog.Info(fmt.Sprintf("  ⏱️ %s - 延迟: %dms (来源: 定期健康检查)", 
					ep.Config.Name, responseTime.Milliseconds()))
			}
		}
		
		sort.Slice(healthy, func(i, j int) bool {
			healthy[i].mutex.RLock()
			healthy[j].mutex.RLock()
			defer healthy[i].mutex.RUnlock()
			defer healthy[j].mutex.RUnlock()
			return healthy[i].Status.ResponseTime < healthy[j].Status.ResponseTime
		})
	}

	return healthy
}

// GetFastestEndpointsWithRealTimeTest returns endpoints sorted by real-time testing
func (m *Manager) GetFastestEndpointsWithRealTimeTest(ctx context.Context) []*Endpoint {
	// First get healthy endpoints from regular health checks
	healthy := m.GetHealthyEndpoints()
	if len(healthy) == 0 {
		return healthy
	}

	// If not using fastest strategy or fast test disabled, return regular healthy endpoints
	if m.config.Strategy.Type != "fastest" || !m.config.Strategy.FastTestEnabled {
		return healthy
	}

	// Perform parallel fast testing
	testResults := m.fastTester.TestEndpointsParallel(ctx, healthy)
	
	// Log ALL test results first (including failures)
	if len(testResults) > 0 {
		slog.InfoContext(ctx, "🔍 [Fastest Response Mode] 端点性能测试结果:")
		successCount := 0
		for _, result := range testResults {
			if result.Success {
				successCount++
				slog.InfoContext(ctx, fmt.Sprintf("  ✅ 健康 %s - 响应时间: %dms", 
					result.Endpoint.Config.Name, 
					result.ResponseTime.Milliseconds()))
			} else {
				errorMsg := ""
				if result.Error != nil {
					errorMsg = fmt.Sprintf(" - 错误: %s", result.Error.Error())
				}
				slog.InfoContext(ctx, fmt.Sprintf("  ❌ 异常 %s - 响应时间: %dms%s", 
					result.Endpoint.Config.Name, 
					result.ResponseTime.Milliseconds(),
					errorMsg))
			}
		}
		
		slog.InfoContext(ctx, fmt.Sprintf("📊 [测试摘要] 总共测试: %d个端点, 健康: %d个, 异常: %d个",
			len(testResults), successCount, len(testResults)-successCount))
	}
	
	// Sort by response time (only successful results)
	sortedResults := SortByResponseTime(testResults)
	
	if len(sortedResults) == 0 {
		slog.WarnContext(ctx, "⚠️ [Fastest Response Mode] 所有端点测试失败，回退到健康检查模式")
		return healthy // Fall back to health check results if no fast tests succeeded
	}
	
	// Convert back to endpoint slice
	endpoints := make([]*Endpoint, 0, len(sortedResults))
	for _, result := range sortedResults {
		endpoints = append(endpoints, result.Endpoint)
	}

	// Log the successful endpoint ranking
	if len(endpoints) > 0 {
		// Show the fastest endpoint selection
		fastestEndpoint := endpoints[0]
		var fastestTime int64
		for _, result := range sortedResults {
			if result.Endpoint == fastestEndpoint {
				fastestTime = result.ResponseTime.Milliseconds()
				break
			}
		}
		
		slog.InfoContext(ctx, fmt.Sprintf("🚀 [Fastest Response Mode] 选择最快端点: %s (%dms)", 
			fastestEndpoint.Config.Name, fastestTime))
		
		// Show other available endpoints if there are more than one
		if len(endpoints) > 1 {
			slog.InfoContext(ctx, "📋 [备用端点] 其他可用端点:")
			for i := 1; i < len(endpoints); i++ {
				ep := endpoints[i]
				var responseTime int64
				for _, result := range sortedResults {
					if result.Endpoint == ep {
						responseTime = result.ResponseTime.Milliseconds()
						break
					}
				}
				slog.InfoContext(ctx, fmt.Sprintf("  🔄 备用 %s - 响应时间: %dms", 
					ep.Config.Name, responseTime))
			}
		}
	}

	return endpoints
}

// GetEndpointByName returns an endpoint by name
func (m *Manager) GetEndpointByName(name string) *Endpoint {
	for _, endpoint := range m.endpoints {
		if endpoint.Config.Name == name {
			return endpoint
		}
	}
	return nil
}

// GetAllEndpoints returns all endpoints
func (m *Manager) GetAllEndpoints() []*Endpoint {
	return m.endpoints
}

// GetConfig returns the manager's configuration
func (m *Manager) GetConfig() *config.Config {
	return m.config
}

// healthCheckLoop runs the health check routine
func (m *Manager) healthCheckLoop() {
	defer m.wg.Done()
	
	ticker := time.NewTicker(m.config.Health.CheckInterval)
	defer ticker.Stop()

	// Initial health check
	m.performHealthChecks()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.performHealthChecks()
		}
	}
}

// performHealthChecks performs health checks on all endpoints
func (m *Manager) performHealthChecks() {
	var wg sync.WaitGroup
	
	for _, endpoint := range m.endpoints {
		wg.Add(1)
		go func(ep *Endpoint) {
			defer wg.Done()
			m.checkEndpointHealth(ep)
		}(endpoint)
	}
	
	wg.Wait()
}

// checkEndpointHealth checks the health of a single endpoint
func (m *Manager) checkEndpointHealth(endpoint *Endpoint) {
	start := time.Now()
	
	healthURL := endpoint.Config.URL + m.config.Health.HealthPath
	req, err := http.NewRequestWithContext(m.ctx, "GET", healthURL, nil)
	if err != nil {
		m.updateEndpointStatus(endpoint, false, 0)
		return
	}

	// Add authorization header if token is configured
	if endpoint.Config.Token != "" {
		req.Header.Set("Authorization", "Bearer "+endpoint.Config.Token)
	}

	resp, err := m.client.Do(req)
	responseTime := time.Since(start)
	
	if err != nil {
		m.updateEndpointStatus(endpoint, false, responseTime)
		return
	}
	
	resp.Body.Close()
	
	// Consider 2xx and 40x as healthy for API endpoints
	// 2xx: Success responses
	// 40x: Client errors (like 401 Unauthorized, 403 Forbidden) indicate the endpoint is reachable
	healthy := (resp.StatusCode >= 200 && resp.StatusCode < 300) || 
			   (resp.StatusCode >= 400 && resp.StatusCode < 500)
	
	// Log health check results
	if healthy {
		slog.Debug("✅ Health check passed",
			"endpoint", endpoint.Config.Name,
			"url", endpoint.Config.URL,
			"status_code", resp.StatusCode,
			"response_time_ms", responseTime.Milliseconds())
	} else {
		slog.Warn("❌ Health check failed",
			"endpoint", endpoint.Config.Name,
			"url", endpoint.Config.URL,
			"status_code", resp.StatusCode,
			"response_time_ms", responseTime.Milliseconds())
	}
	
	m.updateEndpointStatus(endpoint, healthy, responseTime)
}

// updateEndpointStatus updates the health status of an endpoint
func (m *Manager) updateEndpointStatus(endpoint *Endpoint, healthy bool, responseTime time.Duration) {
	endpoint.mutex.Lock()
	defer endpoint.mutex.Unlock()
	
	endpoint.Status.LastCheck = time.Now()
	endpoint.Status.ResponseTime = responseTime
	
	if healthy {
		endpoint.Status.Healthy = true
		endpoint.Status.ConsecutiveFails = 0
	} else {
		endpoint.Status.ConsecutiveFails++
		// Mark as unhealthy after 2 consecutive failures
		if endpoint.Status.ConsecutiveFails >= 2 {
			endpoint.Status.Healthy = false
		}
	}
}

// IsHealthy returns the health status of an endpoint
func (e *Endpoint) IsHealthy() bool {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	return e.Status.Healthy
}

// GetResponseTime returns the last response time of an endpoint
func (e *Endpoint) GetResponseTime() time.Duration {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	return e.Status.ResponseTime
}

// GetStatus returns a copy of the endpoint status
func (e *Endpoint) GetStatus() EndpointStatus {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	return e.Status
}