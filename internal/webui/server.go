package webui

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"endpoint_forwarder/config"
	"endpoint_forwarder/internal/endpoint"
	"endpoint_forwarder/internal/middleware"
	"endpoint_forwarder/internal/monitor"
)

// WebUIServer represents the WebUI server
type WebUIServer struct {
	cfg                  *config.Config
	endpointManager      *endpoint.Manager
	monitoringMiddleware *middleware.MonitoringMiddleware
	startTime            time.Time
	server               *http.Server
	logger               *slog.Logger
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
		running:              false,
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

// handleLogs returns logs data (placeholder - logs would need to be collected)
func (w *WebUIServer) handleLogs(rw http.ResponseWriter, r *http.Request) {
	// For now, return empty logs since we don't have a centralized log collector
	// In a full implementation, you'd want to integrate with the logging system
	data := map[string]interface{}{
		"logs": []map[string]interface{}{
			{
				"timestamp": time.Now().Format("15:04:05"),
				"level":     "INFO",
				"source":    "webui",
				"message":   "WebUI服务器正在运行",
			},
		},
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
