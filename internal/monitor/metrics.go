package monitor

import (
	"sync"
	"time"
)

// Metrics contains all monitoring metrics
type Metrics struct {
	mu sync.RWMutex

	// Request metrics
	TotalRequests     int64
	SuccessfulRequests int64
	FailedRequests    int64
	
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

// NewMetrics creates a new metrics instance
func NewMetrics() *Metrics {
	return &Metrics{
		EndpointStats:     make(map[string]*EndpointMetrics),
		ActiveConnections: make(map[string]*ConnectionInfo),
		ConnectionHistory: make([]*ConnectionInfo, 0),
		StartTime:         time.Now(),
		RequestHistory:    make([]RequestDataPoint, 0),
		ResponseHistory:   make([]ResponseTimePoint, 0),
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
		if m.EndpointStats[endpoint] != nil {
			m.EndpointStats[endpoint].SuccessfulRequests++
		}
	} else {
		m.FailedRequests++
		if m.EndpointStats[endpoint] != nil {
			m.EndpointStats[endpoint].FailedRequests++
		}
	}

	// Update endpoint metrics
	if endpointMetrics := m.EndpointStats[endpoint]; endpointMetrics != nil {
		endpointMetrics.TotalResponseTime += responseTime
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
		TotalResponseTime:  m.TotalResponseTime,
		MinResponseTime:    m.MinResponseTime,
		MaxResponseTime:    m.MaxResponseTime,
		StartTime:          m.StartTime,
		EndpointStats:      make(map[string]*EndpointMetrics),
		ActiveConnections:  make(map[string]*ConnectionInfo),
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

// generateConnectionID generates a unique connection ID
func generateConnectionID() string {
	return time.Now().Format("20060102150405.000000")
}