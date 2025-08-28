package monitor

import (
	"fmt"
	"sync"
	"time"
)

// TokenUsage represents token usage statistics
type TokenUsage struct {
	InputTokens            int64
	OutputTokens           int64
	CacheCreationTokens    int64
	CacheReadTokens        int64
}

// Metrics contains all monitoring metrics
type Metrics struct {
	mu sync.RWMutex

	// Request metrics
	TotalRequests     int64
	SuccessfulRequests int64
	FailedRequests    int64
	
	// Token usage metrics
	TotalTokenUsage   TokenUsage
	
	// Response time metrics
	ResponseTimes     []time.Duration
	TotalResponseTime time.Duration
	MinResponseTime   time.Duration
	MaxResponseTime   time.Duration
	
	// Endpoint metrics
	EndpointStats map[string]*EndpointMetrics
	
	// Connection metrics  
	ActiveConnections map[string]*ConnectionInfo
	ConnectionHistory []*ConnectionInfo
	
	// System metrics
	StartTime time.Time
	
	// Historical data (circular buffer)
	RequestHistory    []RequestDataPoint
	ResponseHistory   []ResponseTimePoint
	TokenHistory      []TokenHistoryPoint
	MaxHistoryPoints  int
}

// EndpointMetrics tracks metrics for a specific endpoint
type EndpointMetrics struct {
	Name             string
	URL              string
	TotalRequests    int64
	SuccessfulRequests int64
	FailedRequests   int64
	TotalResponseTime time.Duration
	MinResponseTime  time.Duration
	MaxResponseTime  time.Duration
	LastUsed         time.Time
	RetryCount       int64
	Priority         int
	Healthy          bool
	TokenUsage       TokenUsage
}

// ConnectionInfo represents an active connection
type ConnectionInfo struct {
	ID             string
	ClientIP       string
	UserAgent      string
	StartTime      time.Time
	LastActivity   time.Time
	Method         string
	Path           string
	Endpoint       string
	Port           string
	RetryCount     int
	Status         string // "active", "completed", "failed", "timeout"
	BytesReceived  int64
	BytesSent      int64
	IsStreaming    bool
	TokenUsage     TokenUsage  // Token usage for this connection
}

// RequestDataPoint represents a point in time for request metrics
type RequestDataPoint struct {
	Timestamp  time.Time
	Total      int64
	Successful int64
	Failed     int64
}

// ResponseTimePoint represents response time at a point in time
type ResponseTimePoint struct {
	Timestamp    time.Time
	AverageTime  time.Duration
	MinTime      time.Duration
	MaxTime      time.Duration
}

// TokenHistoryPoint represents token usage at a point in time
type TokenHistoryPoint struct {
	Timestamp           time.Time
	InputTokens         int64
	OutputTokens        int64
	CacheCreationTokens int64
	CacheReadTokens     int64
	TotalTokens         int64
}

// NewMetrics creates a new metrics instance
func NewMetrics() *Metrics {
	return &Metrics{
		EndpointStats:     make(map[string]*EndpointMetrics),
		ActiveConnections: make(map[string]*ConnectionInfo),
		ConnectionHistory: make([]*ConnectionInfo, 0),
		StartTime:         time.Now(),
		RequestHistory:    make([]RequestDataPoint, 0),
		ResponseHistory:   make([]ResponseTimePoint, 0),
		TokenHistory:      make([]TokenHistoryPoint, 0),
		MaxHistoryPoints:  300, // 5 minutes of data at 1-second intervals
		MinResponseTime:   time.Duration(0),
		MaxResponseTime:   time.Duration(0),
	}
}

// RecordRequest records a new request
func (m *Metrics) RecordRequest(endpoint, clientIP, userAgent, method, path string) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalRequests++
	
	// Update endpoint stats
	if m.EndpointStats[endpoint] == nil {
		m.EndpointStats[endpoint] = &EndpointMetrics{
			Name:            endpoint,
			MinResponseTime: time.Duration(0),
			MaxResponseTime: time.Duration(0),
		}
	}
	m.EndpointStats[endpoint].TotalRequests++
	m.EndpointStats[endpoint].LastUsed = time.Now()

	// Generate connection ID
	connID := generateConnectionID()
	
	// Create connection info
	conn := &ConnectionInfo{
		ID:           connID,
		ClientIP:     clientIP,
		UserAgent:    userAgent,
		StartTime:    time.Now(),
		LastActivity: time.Now(),
		Method:       method,
		Path:         path,
		Endpoint:     endpoint,
		Status:       "active",
		RetryCount:   0,
		BytesReceived: 0,
		BytesSent:    0,
	}
	
	m.ActiveConnections[connID] = conn
	
	return connID
}

// RecordResponse records a response
func (m *Metrics) RecordResponse(connID string, statusCode int, responseTime time.Duration, bytesSent int64, endpoint string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Update overall metrics
	m.TotalResponseTime += responseTime
	m.ResponseTimes = append(m.ResponseTimes, responseTime)
	
	// Update min/max response times
	if m.MinResponseTime == 0 || responseTime < m.MinResponseTime {
		m.MinResponseTime = responseTime
	}
	if responseTime > m.MaxResponseTime {
		m.MaxResponseTime = responseTime
	}

	// Track success/failure
	if statusCode >= 200 && statusCode < 400 {
		m.SuccessfulRequests++
		// Ensure endpoint stats exist
		if m.EndpointStats[endpoint] == nil && endpoint != "unknown" {
			m.EndpointStats[endpoint] = &EndpointMetrics{
				Name:            endpoint,
				MinResponseTime: time.Duration(0),
				MaxResponseTime: time.Duration(0),
			}
		}
		if endpoint != "unknown" && m.EndpointStats[endpoint] != nil {
			m.EndpointStats[endpoint].SuccessfulRequests++
			m.EndpointStats[endpoint].TotalRequests++
		}
	} else {
		m.FailedRequests++
		// Ensure endpoint stats exist
		if m.EndpointStats[endpoint] == nil && endpoint != "unknown" {
			m.EndpointStats[endpoint] = &EndpointMetrics{
				Name:            endpoint,
				MinResponseTime: time.Duration(0),
				MaxResponseTime: time.Duration(0),
			}
		}
		if endpoint != "unknown" && m.EndpointStats[endpoint] != nil {
			m.EndpointStats[endpoint].FailedRequests++
			m.EndpointStats[endpoint].TotalRequests++
		}
	}

	// Update endpoint metrics
	if endpoint != "unknown" && m.EndpointStats[endpoint] != nil {
		endpointMetrics := m.EndpointStats[endpoint]
		endpointMetrics.TotalResponseTime += responseTime
		endpointMetrics.LastUsed = time.Now()
		if endpointMetrics.MinResponseTime == 0 || responseTime < endpointMetrics.MinResponseTime {
			endpointMetrics.MinResponseTime = responseTime
		}
		if responseTime > endpointMetrics.MaxResponseTime {
			endpointMetrics.MaxResponseTime = responseTime
		}
	}

	// Update connection
	if conn, exists := m.ActiveConnections[connID]; exists {
		conn.LastActivity = time.Now()
		conn.BytesSent = bytesSent
		
		if statusCode >= 200 && statusCode < 400 {
			conn.Status = "completed"
		} else {
			conn.Status = "failed"
		}

		// Move to history and remove from active
		m.ConnectionHistory = append(m.ConnectionHistory, conn)
		delete(m.ActiveConnections, connID)
		
		// Limit history size
		if len(m.ConnectionHistory) > 1000 {
			m.ConnectionHistory = m.ConnectionHistory[len(m.ConnectionHistory)-1000:]
		}
	}

	// Limit response times history
	if len(m.ResponseTimes) > 1000 {
		m.ResponseTimes = m.ResponseTimes[len(m.ResponseTimes)-1000:]
	}
}

// RecordRetry records a retry attempt
func (m *Metrics) RecordRetry(connID string, endpoint string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if conn, exists := m.ActiveConnections[connID]; exists {
		conn.RetryCount++
		conn.LastActivity = time.Now()
		// Debug log to verify retry recording
		fmt.Printf("DEBUG: Recorded retry %d for connection %s on endpoint %s\n", conn.RetryCount, connID, endpoint)
	} else {
		fmt.Printf("DEBUG: Connection %s not found for retry recording\n", connID)
	}

	if endpointMetrics := m.EndpointStats[endpoint]; endpointMetrics != nil {
		endpointMetrics.RetryCount++
	}
}

// UpdateEndpointHealth updates endpoint health status
func (m *Metrics) UpdateEndpointHealth(endpoint, url string, healthy bool, priority int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.EndpointStats[endpoint] == nil {
		m.EndpointStats[endpoint] = &EndpointMetrics{
			Name:            endpoint,
			URL:             url,
			Priority:        priority,
			MinResponseTime: time.Duration(0),
			MaxResponseTime: time.Duration(0),
		}
	}
	
	m.EndpointStats[endpoint].Healthy = healthy
	m.EndpointStats[endpoint].URL = url
	m.EndpointStats[endpoint].Priority = priority
}

// UpdateConnectionEndpoint updates the endpoint name for an active connection
func (m *Metrics) UpdateConnectionEndpoint(connID, endpoint string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if conn, exists := m.ActiveConnections[connID]; exists {
		conn.Endpoint = endpoint
		conn.LastActivity = time.Now()
	}
}

// MarkStreamingConnection marks a connection as streaming
func (m *Metrics) MarkStreamingConnection(connID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if conn, exists := m.ActiveConnections[connID]; exists {
		conn.IsStreaming = true
		conn.LastActivity = time.Now()
	}
}

// GetMetrics returns a snapshot of current metrics
func (m *Metrics) GetMetrics() *Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Create a copy of metrics
	snapshot := &Metrics{
		TotalRequests:      m.TotalRequests,
		SuccessfulRequests: m.SuccessfulRequests,
		FailedRequests:     m.FailedRequests,
		TotalTokenUsage:    m.TotalTokenUsage,
		TotalResponseTime:  m.TotalResponseTime,
		MinResponseTime:    m.MinResponseTime,
		MaxResponseTime:    m.MaxResponseTime,
		StartTime:          m.StartTime,
		EndpointStats:      make(map[string]*EndpointMetrics),
		ActiveConnections:  make(map[string]*ConnectionInfo),
		ConnectionHistory:  make([]*ConnectionInfo, len(m.ConnectionHistory)),
	}

	// Copy endpoint stats
	for k, v := range m.EndpointStats {
		snapshot.EndpointStats[k] = &EndpointMetrics{
			Name:               v.Name,
			URL:                v.URL,
			TotalRequests:      v.TotalRequests,
			SuccessfulRequests: v.SuccessfulRequests,
			FailedRequests:     v.FailedRequests,
			TotalResponseTime:  v.TotalResponseTime,
			MinResponseTime:    v.MinResponseTime,
			MaxResponseTime:    v.MaxResponseTime,
			LastUsed:           v.LastUsed,
			RetryCount:         v.RetryCount,
			Priority:           v.Priority,
			Healthy:            v.Healthy,
			TokenUsage:         v.TokenUsage,
		}
	}

	// Copy active connections
	for k, v := range m.ActiveConnections {
		snapshot.ActiveConnections[k] = &ConnectionInfo{
			ID:            v.ID,
			ClientIP:      v.ClientIP,
			UserAgent:     v.UserAgent,
			StartTime:     v.StartTime,
			LastActivity:  v.LastActivity,
			Method:        v.Method,
			Path:          v.Path,
			Endpoint:      v.Endpoint,
			Port:          v.Port,
			RetryCount:    v.RetryCount,
			Status:        v.Status,
			BytesReceived: v.BytesReceived,
			BytesSent:     v.BytesSent,
			IsStreaming:   v.IsStreaming,
			TokenUsage:    v.TokenUsage,
		}
	}

	// Copy connection history
	for i, v := range m.ConnectionHistory {
		snapshot.ConnectionHistory[i] = &ConnectionInfo{
			ID:            v.ID,
			ClientIP:      v.ClientIP,
			UserAgent:     v.UserAgent,
			StartTime:     v.StartTime,
			LastActivity:  v.LastActivity,
			Method:        v.Method,
			Path:          v.Path,
			Endpoint:      v.Endpoint,
			Port:          v.Port,
			RetryCount:    v.RetryCount,
			Status:        v.Status,
			BytesReceived: v.BytesReceived,
			BytesSent:     v.BytesSent,
			IsStreaming:   v.IsStreaming,
			TokenUsage:    v.TokenUsage,
		}
	}

	// Copy response times (last 100)
	if len(m.ResponseTimes) > 0 {
		start := 0
		if len(m.ResponseTimes) > 100 {
			start = len(m.ResponseTimes) - 100
		}
		snapshot.ResponseTimes = make([]time.Duration, len(m.ResponseTimes[start:]))
		copy(snapshot.ResponseTimes, m.ResponseTimes[start:])
	}

	return snapshot
}

// GetAverageResponseTime calculates average response time
func (m *Metrics) GetAverageResponseTime() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.TotalRequests == 0 {
		return 0
	}
	return m.TotalResponseTime / time.Duration(m.TotalRequests)
}

// GetSuccessRate calculates success rate as percentage
func (m *Metrics) GetSuccessRate() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.TotalRequests == 0 {
		return 0
	}
	return float64(m.SuccessfulRequests) / float64(m.TotalRequests) * 100
}

// GetP95ResponseTime calculates 95th percentile response time
func (m *Metrics) GetP95ResponseTime() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.ResponseTimes) == 0 {
		return 0
	}

	// Simple approximation for P95
	index := int(float64(len(m.ResponseTimes)) * 0.95)
	if index >= len(m.ResponseTimes) {
		index = len(m.ResponseTimes) - 1
	}
	
	// For a proper implementation, we'd sort the slice
	// For now, return max as approximation
	return m.MaxResponseTime
}

// RecordTokenUsage records token usage for a specific request
func (m *Metrics) RecordTokenUsage(connID string, endpoint string, tokens *TokenUsage) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Update overall token metrics
	m.TotalTokenUsage.InputTokens += tokens.InputTokens
	m.TotalTokenUsage.OutputTokens += tokens.OutputTokens
	m.TotalTokenUsage.CacheCreationTokens += tokens.CacheCreationTokens
	m.TotalTokenUsage.CacheReadTokens += tokens.CacheReadTokens

	// Record token history point
	historyPoint := TokenHistoryPoint{
		Timestamp:           time.Now(),
		InputTokens:         m.TotalTokenUsage.InputTokens,
		OutputTokens:        m.TotalTokenUsage.OutputTokens,
		CacheCreationTokens: m.TotalTokenUsage.CacheCreationTokens,
		CacheReadTokens:     m.TotalTokenUsage.CacheReadTokens,
		TotalTokens:         m.TotalTokenUsage.InputTokens + m.TotalTokenUsage.OutputTokens,
	}
	
	m.TokenHistory = append(m.TokenHistory, historyPoint)
	
	// Limit token history size
	if len(m.TokenHistory) > m.MaxHistoryPoints {
		m.TokenHistory = m.TokenHistory[len(m.TokenHistory)-m.MaxHistoryPoints:]
	}

	// Update endpoint-specific token metrics
	if endpoint != "unknown" && m.EndpointStats[endpoint] != nil {
		m.EndpointStats[endpoint].TokenUsage.InputTokens += tokens.InputTokens
		m.EndpointStats[endpoint].TokenUsage.OutputTokens += tokens.OutputTokens
		m.EndpointStats[endpoint].TokenUsage.CacheCreationTokens += tokens.CacheCreationTokens
		m.EndpointStats[endpoint].TokenUsage.CacheReadTokens += tokens.CacheReadTokens
	}

	// Update connection info if available
	if conn, exists := m.ActiveConnections[connID]; exists {
		// Update token usage for this connection
		conn.TokenUsage.InputTokens += tokens.InputTokens
		conn.TokenUsage.OutputTokens += tokens.OutputTokens
		conn.TokenUsage.CacheCreationTokens += tokens.CacheCreationTokens
		conn.TokenUsage.CacheReadTokens += tokens.CacheReadTokens
		conn.LastActivity = time.Now()
	}
}

// GetTotalTokenStats returns total token usage statistics
func (m *Metrics) GetTotalTokenStats() TokenUsage {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.TotalTokenUsage
}

// GetTokenHistory returns the token usage history
func (m *Metrics) GetTokenHistory() []TokenHistoryPoint {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy of the token history
	history := make([]TokenHistoryPoint, len(m.TokenHistory))
	copy(history, m.TokenHistory)
	return history
}

// generateConnectionID generates a unique connection ID
func generateConnectionID() string {
	return time.Now().Format("20060102150405.000000")
}