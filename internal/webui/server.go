package webui

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"endpoint_forwarder/config"
	"endpoint_forwarder/internal/endpoint"
	"endpoint_forwarder/internal/middleware"
	"endpoint_forwarder/internal/monitor"
)

// LogEntry represents a log entry for WebUI
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Source    string `json:"source"`
	Message   string `json:"message"`
}

// LogCollector collects and manages logs for WebUI display
type LogCollector struct {
	logs        []LogEntry
	maxLogs     int
	mutex       sync.RWMutex
	subscribers []chan LogEntry
}

// NewLogCollector creates a new log collector
func NewLogCollector(maxLogs int) *LogCollector {
	return &LogCollector{
		logs:        make([]LogEntry, 0, maxLogs),
		maxLogs:     maxLogs,
		subscribers: make([]chan LogEntry, 0),
	}
}

// AddLog adds a new log entry and notifies subscribers
func (lc *LogCollector) AddLog(level, message, source string) {
	lc.mutex.Lock()
	defer lc.mutex.Unlock()

	entry := LogEntry{
		Timestamp: time.Now().Format("15:04:05"),
		Level:     level,
		Source:    source,
		Message:   message,
	}

	// Add to logs buffer
	lc.logs = append(lc.logs, entry)
	
	// Keep only the latest maxLogs entries
	if len(lc.logs) > lc.maxLogs {
		lc.logs = lc.logs[1:]
	}

	// Notify all subscribers
	for _, subscriber := range lc.subscribers {
		select {
		case subscriber <- entry:
		default:
			// Skip if subscriber's channel is full
		}
	}
}

// GetLogs returns all current logs
func (lc *LogCollector) GetLogs() []LogEntry {
	lc.mutex.RLock()
	defer lc.mutex.RUnlock()
	
	// Return a copy of logs
	result := make([]LogEntry, len(lc.logs))
	copy(result, lc.logs)
	return result
}

// Subscribe adds a new subscriber for real-time log updates
func (lc *LogCollector) Subscribe() chan LogEntry {
	lc.mutex.Lock()
	defer lc.mutex.Unlock()
	
	subscriber := make(chan LogEntry, 100) // Buffer for 100 log entries
	lc.subscribers = append(lc.subscribers, subscriber)
	return subscriber
}

// Unsubscribe removes a subscriber
func (lc *LogCollector) Unsubscribe(subscriber chan LogEntry) {
	lc.mutex.Lock()
	defer lc.mutex.Unlock()
	
	for i, sub := range lc.subscribers {
		if sub == subscriber {
			lc.subscribers = append(lc.subscribers[:i], lc.subscribers[i+1:]...)
			close(subscriber)
			break
		}
	}
}

// WebUIServer represents the WebUI server
type WebUIServer struct {
	cfg                  *config.Config
	endpointManager      *endpoint.Manager
	monitoringMiddleware *middleware.MonitoringMiddleware
	startTime            time.Time
	server               *http.Server
	logger               *slog.Logger
	logCollector         *LogCollector
	running              bool
}

// NewWebUIServer creates a new WebUI server
func NewWebUIServer(cfg *config.Config, endpointManager *endpoint.Manager, monitoringMiddleware *middleware.MonitoringMiddleware, startTime time.Time, logger *slog.Logger) *WebUIServer {
	return &WebUIServer{
		cfg:                  cfg,
		endpointManager:      endpointManager,
		monitoringMiddleware: monitoringMiddleware,
		startTime:            startTime,
		logger:               logger,
		logCollector:         NewLogCollector(500), // Keep consistent with TUI (500 logs)
		running:              false,
	}
}

// AddLog allows external systems to add logs to the collector
func (w *WebUIServer) AddLog(level, message, source string) {
	if w.logCollector != nil {
		w.logCollector.AddLog(level, message, source)
	}
}

// Start starts the WebUI server
func (w *WebUIServer) Start() error {
	if !w.cfg.WebUI.Enabled {
		return nil
	}

	mux := http.NewServeMux()

	// Serve static files (HTML, CSS, JS)
	mux.HandleFunc("/", w.handleIndex)
	mux.HandleFunc("/static/", w.handleStatic)

	// API endpoints
	mux.HandleFunc("/api/overview", w.handleOverview)
	mux.HandleFunc("/api/endpoints", w.handleEndpoints)
	mux.HandleFunc("/api/connections", w.handleConnections)
	mux.HandleFunc("/api/logs", w.handleLogs)
	mux.HandleFunc("/api/config", w.handleConfig)

	// Server-Sent Events for real-time updates
	mux.HandleFunc("/api/events", w.handleEvents)
	
	// Server-Sent Events for real-time log updates  
	mux.HandleFunc("/api/log-stream", w.handleLogStream)

	w.server = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", w.cfg.WebUI.Host, w.cfg.WebUI.Port),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	w.running = true
	w.logger.Info("🌐 WebUI服务器启动中...", "address", w.server.Addr)

	go func() {
		if err := w.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			w.logger.Error("WebUI服务器错误", "error", err)
		}
	}()

	w.logger.Info("✅ WebUI服务器启动成功！", "url", fmt.Sprintf("http://%s", w.server.Addr))
	return nil
}

// Stop stops the WebUI server
func (w *WebUIServer) Stop() error {
	if w.server == nil || !w.running {
		return nil
	}

	w.running = false
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	w.logger.Info("🛑 正在关闭WebUI服务器...")
	return w.server.Shutdown(ctx)
}

// IsRunning returns whether the WebUI server is running
func (w *WebUIServer) IsRunning() bool {
	return w.running
}

// handleIndex serves the main HTML page
func (w *WebUIServer) handleIndex(rw http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(rw, r)
		return
	}

	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	rw.Write([]byte(indexHTML))
}

// handleStatic serves static files
func (w *WebUIServer) handleStatic(rw http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch path {
	case "/static/style.css":
		rw.Header().Set("Content-Type", "text/css")
		rw.Write([]byte(styleCSS))
	case "/static/app.js":
		rw.Header().Set("Content-Type", "application/javascript")
		rw.Write([]byte(appJS))
	default:
		http.NotFound(rw, r)
	}
}

// handleOverview returns overview data
func (w *WebUIServer) handleOverview(rw http.ResponseWriter, r *http.Request) {
	metrics := w.monitoringMiddleware.GetMetrics().GetMetrics()
	endpoints := w.endpointManager.GetAllEndpoints()

	// Calculate uptime
	uptime := time.Since(w.startTime)

	// Get endpoint status
	healthyCount := 0
	endpointStatuses := make([]map[string]interface{}, 0, len(endpoints))
	for _, ep := range endpoints {
		status := ep.GetStatus()
		if status.Healthy {
			healthyCount++
		}

		endpointStatuses = append(endpointStatuses, map[string]interface{}{
			"name":         ep.Config.Name,
			"healthy":      status.Healthy,
			"responseTime": status.ResponseTime.Milliseconds(),
		})
	}

	// Get token usage statistics
	tokenStats := metrics.GetTotalTokenStats()
	totalTokens := tokenStats.InputTokens + tokenStats.OutputTokens

	data := map[string]interface{}{
		"metrics": map[string]interface{}{
			"totalRequests":       metrics.TotalRequests,
			"successfulRequests":  metrics.SuccessfulRequests,
			"failedRequests":      metrics.FailedRequests,
			"successRate":         metrics.GetSuccessRate(),
			"averageResponseTime": metrics.GetAverageResponseTime().Milliseconds(),
		},
		"tokens": map[string]interface{}{
			"inputTokens":         tokenStats.InputTokens,
			"outputTokens":        tokenStats.OutputTokens,
			"cacheCreationTokens": tokenStats.CacheCreationTokens,
			"cacheReadTokens":     tokenStats.CacheReadTokens,
			"totalTokens":         totalTokens,
		},
		"endpoints": map[string]interface{}{
			"total":    len(endpoints),
			"healthy":  healthyCount,
			"statuses": endpointStatuses,
		},
		"system": map[string]interface{}{
			"activeConnections": len(metrics.ActiveConnections),
			"totalConnections":  len(metrics.ActiveConnections) + len(metrics.ConnectionHistory),
			"uptime":            uptime.Seconds(),
		},
		"connectionHistory": w.getRecentConnectionHistory(metrics.ConnectionHistory, 3),
	}

	w.writeJSON(rw, data)
}

// handleEndpoints returns endpoints data
func (w *WebUIServer) handleEndpoints(rw http.ResponseWriter, r *http.Request) {
	endpoints := w.endpointManager.GetAllEndpoints()
	metrics := w.monitoringMiddleware.GetMetrics().GetMetrics()

	endpointData := make([]map[string]interface{}, 0, len(endpoints))

	for _, ep := range endpoints {
		status := ep.GetStatus()
		endpointStats := metrics.EndpointStats[ep.Config.Name]

		data := map[string]interface{}{
			"name":             ep.Config.Name,
			"url":              ep.Config.URL,
			"priority":         ep.Config.Priority,
			"timeout":          ep.Config.Timeout.String(),
			"healthy":          status.Healthy,
			"responseTime":     status.ResponseTime.Milliseconds(),
			"consecutiveFails": status.ConsecutiveFails,
			"lastCheck":        status.LastCheck.Format("15:04:05"),
		}

		if endpointStats != nil {
			successRate := float64(0)
			if endpointStats.TotalRequests > 0 {
				successRate = float64(endpointStats.SuccessfulRequests) / float64(endpointStats.TotalRequests) * 100
			}

			avgResponseTime := time.Duration(0)
			if endpointStats.TotalRequests > 0 {
				avgResponseTime = endpointStats.TotalResponseTime / time.Duration(endpointStats.TotalRequests)
			}

			data["stats"] = map[string]interface{}{
				"totalRequests":      endpointStats.TotalRequests,
				"successfulRequests": endpointStats.SuccessfulRequests,
				"successRate":        successRate,
				"retryCount":         endpointStats.RetryCount,
				"avgResponseTime":    avgResponseTime.Milliseconds(),
				"minResponseTime":    endpointStats.MinResponseTime.Milliseconds(),
				"maxResponseTime":    endpointStats.MaxResponseTime.Milliseconds(),
				"lastUsed":           endpointStats.LastUsed.Format("15:04:05"),
				"tokenUsage": map[string]interface{}{
					"inputTokens":         endpointStats.TokenUsage.InputTokens,
					"outputTokens":        endpointStats.TokenUsage.OutputTokens,
					"cacheCreationTokens": endpointStats.TokenUsage.CacheCreationTokens,
					"cacheReadTokens":     endpointStats.TokenUsage.CacheReadTokens,
				},
			}
		}

		endpointData = append(endpointData, data)
	}

	w.writeJSON(rw, map[string]interface{}{
		"endpoints": endpointData,
	})
}

// handleConnections returns connections data
func (w *WebUIServer) handleConnections(rw http.ResponseWriter, r *http.Request) {
	metrics := w.monitoringMiddleware.GetMetrics().GetMetrics()

	// Convert active connections to JSON-friendly format
	activeConnections := make([]map[string]interface{}, 0, len(metrics.ActiveConnections))
	for _, conn := range metrics.ActiveConnections {
		duration := time.Since(conn.StartTime)
		endpoint := conn.Endpoint
		if endpoint == "" || endpoint == "unknown" {
			endpoint = "pending"
		}

		retryInfo := ""
		if conn.RetryCount >= 0 {
			retryInfo = fmt.Sprintf("(%d/%d retry)", conn.RetryCount, w.cfg.Retry.MaxAttempts)
		}

		activeConnections = append(activeConnections, map[string]interface{}{
			"clientIP":  conn.ClientIP,
			"method":    conn.Method,
			"path":      conn.Path,
			"endpoint":  endpoint,
			"retryInfo": retryInfo,
			"duration":  duration.Seconds(),
			"startTime": conn.StartTime.Format("15:04:05"),
		})
	}

	data := map[string]interface{}{
		"activeCount":       len(metrics.ActiveConnections),
		"historicalCount":   len(metrics.ConnectionHistory),
		"activeConnections": activeConnections,
	}

	w.writeJSON(rw, data)
}

// handleLogs returns logs data
func (w *WebUIServer) handleLogs(rw http.ResponseWriter, r *http.Request) {
	logs := w.logCollector.GetLogs()
	
	// Convert LogEntry to the format expected by the frontend
	logData := make([]map[string]interface{}, 0, len(logs))
	for _, log := range logs {
		logData = append(logData, map[string]interface{}{
			"timestamp": log.Timestamp,
			"level":     log.Level,
			"source":    log.Source,
			"message":   log.Message,
		})
	}

	data := map[string]interface{}{
		"logs": logData,
	}

	w.writeJSON(rw, data)
}

// handleConfig returns configuration data
func (w *WebUIServer) handleConfig(rw http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"server": map[string]interface{}{
			"host": w.cfg.Server.Host,
			"port": w.cfg.Server.Port,
		},
		"strategy": map[string]interface{}{
			"type":            w.cfg.Strategy.Type,
			"fastTestEnabled": w.cfg.Strategy.FastTestEnabled,
		},
		"auth": map[string]interface{}{
			"enabled": w.cfg.Auth.Enabled,
		},
		"tui": map[string]interface{}{
			"updateInterval": w.cfg.TUI.UpdateInterval.String(),
		},
		"webui": map[string]interface{}{
			"enabled": w.cfg.WebUI.Enabled,
			"host":    w.cfg.WebUI.Host,
			"port":    w.cfg.WebUI.Port,
		},
		"endpoints": func() []map[string]interface{} {
			endpoints := make([]map[string]interface{}, 0, len(w.cfg.Endpoints))
			for _, ep := range w.cfg.Endpoints {
				endpoints = append(endpoints, map[string]interface{}{
					"name":     ep.Name,
					"url":      ep.URL,
					"priority": ep.Priority,
					"timeout":  ep.Timeout.String(),
				})
			}
			return endpoints
		}(),
	}

	w.writeJSON(rw, data)
}

// handleEvents provides Server-Sent Events for real-time updates
func (w *WebUIServer) handleEvents(rw http.ResponseWriter, r *http.Request) {
	// Set headers for SSE
	rw.Header().Set("Content-Type", "text/event-stream")
	rw.Header().Set("Cache-Control", "no-cache")
	rw.Header().Set("Connection", "keep-alive")
	rw.Header().Set("Access-Control-Allow-Origin", "*")

	// Create a channel to signal when the client disconnects
	clientGone := r.Context().Done()

	// Send periodic updates
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-clientGone:
			return
		case <-ticker.C:
			// Send overview data as SSE
			metrics := w.monitoringMiddleware.GetMetrics().GetMetrics()

			data := map[string]interface{}{
				"totalRequests":     metrics.TotalRequests,
				"successRate":       metrics.GetSuccessRate(),
				"activeConnections": len(metrics.ActiveConnections),
				"timestamp":         time.Now().Unix(),
			}

			jsonData, _ := json.Marshal(data)
			fmt.Fprintf(rw, "data: %s\n\n", jsonData)

			if flusher, ok := rw.(http.Flusher); ok {
				flusher.Flush()
			}
		}
	}
}

// handleLogStream provides Server-Sent Events for real-time log updates
func (w *WebUIServer) handleLogStream(rw http.ResponseWriter, r *http.Request) {
	// Set headers for SSE
	rw.Header().Set("Content-Type", "text/event-stream")
	rw.Header().Set("Cache-Control", "no-cache")
	rw.Header().Set("Connection", "keep-alive")
	rw.Header().Set("Access-Control-Allow-Origin", "*")

	// Create a channel to signal when the client disconnects
	clientGone := r.Context().Done()
	
	// Subscribe to log updates
	logChannel := w.logCollector.Subscribe()
	defer w.logCollector.Unsubscribe(logChannel)

	// Send initial logs
	initialLogs := w.logCollector.GetLogs()
	for _, log := range initialLogs {
		jsonData, _ := json.Marshal(log)
		fmt.Fprintf(rw, "data: %s\n\n", jsonData)
		if flusher, ok := rw.(http.Flusher); ok {
			flusher.Flush()
		}
	}

	// Stream real-time log updates
	for {
		select {
		case <-clientGone:
			return
		case logEntry := <-logChannel:
			jsonData, _ := json.Marshal(logEntry)
			fmt.Fprintf(rw, "data: %s\n\n", jsonData)
			
			if flusher, ok := rw.(http.Flusher); ok {
				flusher.Flush()
			}
		}
	}
}

// writeJSON writes JSON response
func (w *WebUIServer) writeJSON(rw http.ResponseWriter, data interface{}) {
	rw.Header().Set("Content-Type", "application/json")
	rw.Header().Set("Access-Control-Allow-Origin", "*")

	if err := json.NewEncoder(rw).Encode(data); err != nil {
		w.logger.Error("Failed to encode JSON response", "error", err)
		http.Error(rw, "Internal Server Error", http.StatusInternalServerError)
	}
}

// getRecentConnectionHistory returns recent connection history with token data
func (w *WebUIServer) getRecentConnectionHistory(history []*monitor.ConnectionInfo, limit int) []map[string]interface{} {
	// Filter connections that have token usage and get the most recent ones
	var connectionsWithTokens []map[string]interface{}

	for i := len(history) - 1; i >= 0 && len(connectionsWithTokens) < limit; i-- {
		conn := history[i]
		totalTokens := conn.TokenUsage.InputTokens + conn.TokenUsage.OutputTokens +
			conn.TokenUsage.CacheCreationTokens + conn.TokenUsage.CacheReadTokens

		if totalTokens > 0 {
			endpoint := conn.Endpoint
			if endpoint == "" || endpoint == "unknown" {
				endpoint = "pending"
			}

			status := "success"
			if conn.Status == "failed" {
				status = "failed"
			}

			connectionsWithTokens = append(connectionsWithTokens, map[string]interface{}{
				"clientIP": conn.ClientIP,
				"endpoint": endpoint,
				"status":   status,
				"tokenUsage": map[string]interface{}{
					"inputTokens":         conn.TokenUsage.InputTokens,
					"outputTokens":        conn.TokenUsage.OutputTokens,
					"cacheCreationTokens": conn.TokenUsage.CacheCreationTokens,
					"cacheReadTokens":     conn.TokenUsage.CacheReadTokens,
					"totalTokens":         totalTokens,
				},
			})
		}
	}

	return connectionsWithTokens
}
