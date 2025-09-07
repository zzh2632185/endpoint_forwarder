package webui

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"endpoint_forwarder/config"
	"endpoint_forwarder/internal/endpoint"
	"endpoint_forwarder/internal/middleware"
	"endpoint_forwarder/internal/monitor"

	yaml "gopkg.in/yaml.v3"
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
	authMiddleware       *AuthMiddleware
	running              bool
	configRegistry       *config.ConfigRegistry
	configDir            string
	registryPath         string
	configWatcher        *config.ConfigWatcher
}

// NewWebUIServer creates a new WebUI server
func NewWebUIServer(cfg *config.Config, endpointManager *endpoint.Manager, monitoringMiddleware *middleware.MonitoringMiddleware, startTime time.Time, logger *slog.Logger) *WebUIServer {
	// Initialize config management
	configDir := "config"
	// Normalize to absolute to avoid systemd working dir issues
	if abs, err := filepath.Abs(configDir); err == nil {
		configDir = abs
	}
	registryPath := filepath.Join(configDir, "registry.yaml")

	// Try to initialize config registry
	configRegistry, err := config.ScanAndInitializeRegistry(configDir, registryPath, "")
	if err != nil {
		logger.Warn("Failed to initialize config registry", "error", err)
		configRegistry = config.NewConfigRegistry()
	}

	return &WebUIServer{
		cfg:                  cfg,
		endpointManager:      endpointManager,
		monitoringMiddleware: monitoringMiddleware,
		startTime:            startTime,
		logger:               logger,
		logCollector:         NewLogCollector(500), // Keep consistent with TUI (500 logs)
		authMiddleware:       NewAuthMiddleware(cfg.WebUI.Password),
		running:              false,
		configRegistry:       configRegistry,
		configDir:            configDir,
		registryPath:         registryPath,
	}
}

// SetConfigWatcher sets the config watcher reference
func (w *WebUIServer) SetConfigWatcher(configWatcher *config.ConfigWatcher) {
	w.configWatcher = configWatcher
	// Update registry from config watcher
	w.configRegistry = configWatcher.GetRegistry()
}

// UpdateConfig updates the WebUI server configuration
func (w *WebUIServer) UpdateConfig(cfg *config.Config) {
	w.cfg = cfg
	// Update auth middleware with new config
	w.authMiddleware.UpdateConfig(cfg.WebUI.Password)
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

	// Authentication endpoints (no auth required)
	mux.HandleFunc("/login", w.authMiddleware.HandleLogin)
	mux.HandleFunc("/logout", w.authMiddleware.HandleLogout)

	// Protected endpoints (require authentication if password is set)
	mux.HandleFunc("/", w.authMiddleware.RequireAuth(w.handleIndex))
	mux.HandleFunc("/static/", w.authMiddleware.RequireAuth(w.handleStatic))

	// Protected API endpoints
	mux.HandleFunc("/api/overview", w.authMiddleware.RequireAuth(w.handleOverview))
	mux.HandleFunc("/api/endpoints", w.authMiddleware.RequireAuth(w.handleEndpoints))
	mux.HandleFunc("/api/connections", w.authMiddleware.RequireAuth(w.handleConnections))
	mux.HandleFunc("/api/logs", w.authMiddleware.RequireAuth(w.handleLogs))
	mux.HandleFunc("/api/config", w.authMiddleware.RequireAuth(w.handleConfig))

	// Protected Server-Sent Events for real-time updates
	mux.HandleFunc("/api/events", w.authMiddleware.RequireAuth(w.handleEvents))

	// Protected Server-Sent Events for real-time log updates
	mux.HandleFunc("/api/log-stream", w.authMiddleware.RequireAuth(w.handleLogStream))

	// Protected Configuration editing endpoints (WebUI TUI-like functionality)
	mux.HandleFunc("/api/endpoints/priority", w.authMiddleware.RequireAuth(w.handleEndpointPriority))
	mux.HandleFunc("/api/config/save", w.authMiddleware.RequireAuth(w.handleConfigSave))
	mux.HandleFunc("/api/endpoints/details", w.authMiddleware.RequireAuth(w.handleEndpointDetails))
	mux.HandleFunc("/api/overview/token-history", w.authMiddleware.RequireAuth(w.handleTokenHistory))

	// Protected Configuration management endpoints
	mux.HandleFunc("/api/configs", w.authMiddleware.RequireAuth(w.handleConfigs))
	mux.HandleFunc("/api/configs/import", w.authMiddleware.RequireAuth(w.handleConfigImport))
	mux.HandleFunc("/api/configs/switch", w.authMiddleware.RequireAuth(w.handleConfigSwitch))
	mux.HandleFunc("/api/configs/delete", w.authMiddleware.RequireAuth(w.handleConfigDelete))
	mux.HandleFunc("/api/configs/rename", w.authMiddleware.RequireAuth(w.handleConfigRename))
	mux.HandleFunc("/api/configs/active", w.authMiddleware.RequireAuth(w.handleActiveConfig))
	// New: config file content + export endpoints
	mux.HandleFunc("/api/configs/content", w.authMiddleware.RequireAuth(w.handleConfigContent))
	mux.HandleFunc("/api/configs/export", w.authMiddleware.RequireAuth(w.handleConfigExport))
	mux.HandleFunc("/api/configs/export-all", w.authMiddleware.RequireAuth(w.handleConfigExportAll))

	w.server = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", w.cfg.WebUI.Host, w.cfg.WebUI.Port),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	w.running = true
	w.logger.Info("🌐 WebUI服务器启动中...", "address", w.server.Addr)

	// Start server in a goroutine
	serverErr := make(chan error, 1)
	go func() {
		w.logger.Debug("WebUI服务器开始监听...", "address", w.server.Addr)
		if err := w.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			w.logger.Error("WebUI服务器监听失败", "error", err, "address", w.server.Addr)
			serverErr <- err
		} else {
			w.logger.Debug("WebUI服务器监听结束", "address", w.server.Addr)
		}
	}()

	// Give server a moment to start
	time.Sleep(200 * time.Millisecond)

	// Check if server started successfully
	select {
	case err := <-serverErr:
		w.running = false
		w.logger.Error("WebUI服务器启动失败", "error", err, "address", w.server.Addr)
		return fmt.Errorf("WebUI服务器启动失败: %w", err)
	default:
		w.logger.Info("✅ WebUI服务器启动成功！", "url", fmt.Sprintf("http://%s", w.server.Addr))
		return nil
	}
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

		// Get failed requests count (consistent with TUI implementation)
		failedRequests := int64(0)
		if endpointStats != nil {
			failedRequests = endpointStats.FailedRequests
		}

		data := map[string]interface{}{
			"name":             ep.Config.Name,
			"url":              ep.Config.URL,
			"priority":         ep.Config.Priority,
			"timeout":          ep.Config.Timeout.String(),
			"healthy":          status.Healthy,
			"responseTime":     status.ResponseTime.Milliseconds(),
			"consecutiveFails": status.ConsecutiveFails, // Keep for backward compatibility
			"failedRequests":   failedRequests,          // Add actual failed requests count
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

// handleEndpointPriority handles endpoint priority modification requests
func (w *WebUIServer) handleEndpointPriority(rw http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		EndpointName string `json:"endpointName"`
		Priority     int    `json:"priority"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(rw, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Find and update the endpoint priority in config
	found := false
	for i := range w.cfg.Endpoints {
		if w.cfg.Endpoints[i].Name == request.EndpointName {
			w.cfg.Endpoints[i].Priority = request.Priority
			found = true
			break
		}
	}

	if !found {
		http.Error(rw, "Endpoint not found", http.StatusNotFound)
		return
	}

	// Update endpoint manager with new config (same as TUI)
	w.endpointManager.UpdateConfig(w.cfg)

	w.logger.Info("WebUI: 端点优先级已更新", "endpoint", request.EndpointName, "priority", request.Priority)

	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(map[string]interface{}{
		"success": true,
		"message": "Priority updated successfully",
	})
}

// handleConfigSave handles configuration save requests
func (w *WebUIServer) handleConfigSave(rw http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		ConfigPath string `json:"configPath,omitempty"`
	}

	// Parse request body (configPath is optional)
	json.NewDecoder(r.Body).Decode(&request)

	// Use default config path if not provided
	configPath := request.ConfigPath
	if configPath == "" {
		// Try to get config path from environment or use default
		// This should match the path used when starting the application
		configPath = "config/config.yaml" // Default path
	}

	// Check if saving is enabled (same logic as TUI)
	if w.cfg.TUI.SavePriorityEdits {
		// Save to config file (preserve comments) - reuse TUI logic
		if err := config.SavePriorityConfigWithComments(w.cfg, configPath); err != nil {
			w.logger.Error("WebUI: 保存配置文件失败", "error", err)
			http.Error(rw, fmt.Sprintf("Failed to save config: %v", err), http.StatusInternalServerError)
			return
		}
		w.logger.Info("WebUI: 配置已保存到文件并同步到路由系统，优先级更改已生效")
	} else {
		w.logger.Info("WebUI: 优先级更改已应用到内存（配置文件保存已禁用）")
	}

	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(map[string]interface{}{
		"success":     true,
		"message":     "Configuration saved successfully",
		"savedToFile": w.cfg.TUI.SavePriorityEdits,
	})
}

// handleEndpointDetails returns detailed endpoint information (similar to TUI details panel)
func (w *WebUIServer) handleEndpointDetails(rw http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	endpointName := r.URL.Query().Get("name")
	if endpointName == "" {
		http.Error(rw, "Endpoint name is required", http.StatusBadRequest)
		return
	}

	endpoints := w.endpointManager.GetAllEndpoints()
	metrics := w.monitoringMiddleware.GetMetrics().GetMetrics()

	var targetEndpoint *endpoint.Endpoint
	for _, ep := range endpoints {
		if ep.Config.Name == endpointName {
			targetEndpoint = ep
			break
		}
	}

	if targetEndpoint == nil {
		http.Error(rw, "Endpoint not found", http.StatusNotFound)
		return
	}

	status := targetEndpoint.GetStatus()
	endpointStats := metrics.EndpointStats[targetEndpoint.Config.Name]

	// Build detailed response similar to TUI details panel
	details := map[string]interface{}{
		"name":          targetEndpoint.Config.Name,
		"url":           targetEndpoint.Config.URL,
		"priority":      targetEndpoint.Config.Priority,
		"group":         targetEndpoint.Config.Group,
		"groupPriority": targetEndpoint.Config.GroupPriority,
		"timeout":       targetEndpoint.Config.Timeout.String(),
		"healthy":       status.Healthy,
		"lastCheck":     status.LastCheck.Format("15:04:05"),
		"responseTime":  status.ResponseTime.Milliseconds(),
		"headers":       targetEndpoint.Config.Headers,
	}

	if endpointStats != nil {
		// Calculate average response time
		var avgResponseTime int64 = 0
		if endpointStats.TotalRequests > 0 {
			avgResponseTime = endpointStats.TotalResponseTime.Milliseconds() / endpointStats.TotalRequests
		}

		details["stats"] = map[string]interface{}{
			"totalRequests":       endpointStats.TotalRequests,
			"successfulRequests":  endpointStats.SuccessfulRequests,
			"failedRequests":      endpointStats.FailedRequests,
			"averageResponseTime": avgResponseTime,
			"minResponseTime":     endpointStats.MinResponseTime.Milliseconds(),
			"maxResponseTime":     endpointStats.MaxResponseTime.Milliseconds(),
			"tokenUsage": map[string]interface{}{
				"inputTokens":         endpointStats.TokenUsage.InputTokens,
				"outputTokens":        endpointStats.TokenUsage.OutputTokens,
				"cacheCreationTokens": endpointStats.TokenUsage.CacheCreationTokens,
				"cacheReadTokens":     endpointStats.TokenUsage.CacheReadTokens,
			},
		}
	}

	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(details)
}

// handleTokenHistory returns historical token usage data (similar to TUI chart)
func (w *WebUIServer) handleTokenHistory(rw http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metrics := w.monitoringMiddleware.GetMetrics().GetMetrics()

	// Get token history from metrics (similar to TUI chart data)
	tokenHistory := make([]map[string]interface{}, 0)

	// If we have token history points, use them
	if len(metrics.TokenHistory) > 0 {
		for _, point := range metrics.TokenHistory {
			tokenHistory = append(tokenHistory, map[string]interface{}{
				"timestamp":           point.Timestamp.Format("15:04:05"),
				"inputTokens":         point.InputTokens,
				"outputTokens":        point.OutputTokens,
				"cacheCreationTokens": point.CacheCreationTokens,
				"cacheReadTokens":     point.CacheReadTokens,
				"totalTokens":         point.TotalTokens,
			})
		}
	} else {
		// If no history, provide current totals as single point
		totalTokens := metrics.TotalTokenUsage.InputTokens + metrics.TotalTokenUsage.OutputTokens
		tokenHistory = append(tokenHistory, map[string]interface{}{
			"timestamp":           time.Now().Format("15:04:05"),
			"inputTokens":         metrics.TotalTokenUsage.InputTokens,
			"outputTokens":        metrics.TotalTokenUsage.OutputTokens,
			"cacheCreationTokens": metrics.TotalTokenUsage.CacheCreationTokens,
			"cacheReadTokens":     metrics.TotalTokenUsage.CacheReadTokens,
			"totalTokens":         totalTokens,
		})
	}

	response := map[string]interface{}{
		"history": tokenHistory,
		"current": map[string]interface{}{
			"inputTokens":         metrics.TotalTokenUsage.InputTokens,
			"outputTokens":        metrics.TotalTokenUsage.OutputTokens,
			"cacheCreationTokens": metrics.TotalTokenUsage.CacheCreationTokens,
			"cacheReadTokens":     metrics.TotalTokenUsage.CacheReadTokens,
			"totalTokens":         metrics.TotalTokenUsage.InputTokens + metrics.TotalTokenUsage.OutputTokens,
		},
	}

	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(response)
}

// handleConfigs returns all available configurations
func (w *WebUIServer) handleConfigs(rw http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	configs := w.configRegistry.GetAllConfigs()

	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(map[string]interface{}{
		"success": true,
		"configs": configs,
	})
}

// handleActiveConfig returns the current active configuration
func (w *WebUIServer) handleActiveConfig(rw http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	activeConfig := w.configRegistry.GetActiveConfig()

	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(map[string]interface{}{
		"success":      true,
		"activeConfig": activeConfig,
	})
}

// handleConfigImport handles configuration file import
func (w *WebUIServer) handleConfigImport(rw http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form
	err := r.ParseMultipartForm(10 << 20) // 10MB max
	if err != nil {
		http.Error(rw, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Get config name
	configName := r.FormValue("configName")
	if configName == "" {
		http.Error(rw, "Config name is required", http.StatusBadRequest)
		return
	}

	// Get uploaded file
	file, _, err := r.FormFile("configFile")
	if err != nil {
		http.Error(rw, "Failed to get uploaded file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Read file content
	configData, err := io.ReadAll(file)
	if err != nil {
		http.Error(rw, "Failed to read file content", http.StatusInternalServerError)
		return
	}

	// Import configuration
	filePath, err := config.ImportConfigFile(w.configDir, configName, configData, w.configRegistry)
	if err != nil {
		w.logger.Error("Failed to import config", "error", err, "name", configName)
		http.Error(rw, fmt.Sprintf("Failed to import config: %v", err), http.StatusBadRequest)
		return
	}

	// Save registry
	if err := w.configRegistry.Save(w.registryPath); err != nil {
		w.logger.Error("Failed to save registry", "error", err)
		http.Error(rw, "Failed to save registry", http.StatusInternalServerError)
		return
	}

	w.logger.Info("Config imported successfully", "name", configName, "path", filePath)

	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(map[string]interface{}{
		"success":  true,
		"message":  "Configuration imported successfully",
		"filePath": filePath,
	})
}

// handleConfigSwitch handles configuration switching
func (w *WebUIServer) handleConfigSwitch(rw http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		ConfigName string `json:"configName"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(rw, "Invalid request body", http.StatusBadRequest)
		return
	}

	if request.ConfigName == "" {
		http.Error(rw, "Config name is required", http.StatusBadRequest)
		return
	}

	// Check if config watcher is available
	if w.configWatcher == nil {
		http.Error(rw, "Configuration switching not available", http.StatusServiceUnavailable)
		return
	}

	// Perform actual config switch
	if err := w.configWatcher.SwitchConfig(request.ConfigName); err != nil {
		w.logger.Error("Failed to switch config", "error", err, "name", request.ConfigName)
		http.Error(rw, fmt.Sprintf("Failed to switch config: %v", err), http.StatusInternalServerError)
		return
	}

	w.logger.Info("Config switched successfully", "name", request.ConfigName)

	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Configuration switched to: %s", request.ConfigName),
	})
}

// handleConfigDelete handles configuration deletion
func (w *WebUIServer) handleConfigDelete(rw http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		ConfigName string `json:"configName"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(rw, "Invalid request body", http.StatusBadRequest)
		return
	}

	if request.ConfigName == "" {
		http.Error(rw, "Config name is required", http.StatusBadRequest)
		return
	}

	// Get config metadata before deletion
	configMeta, err := w.configRegistry.GetConfig(request.ConfigName)
	if err != nil {
		http.Error(rw, fmt.Sprintf("Configuration not found: %s", request.ConfigName), http.StatusNotFound)
		return
	}

	// Remove from registry
	if err := w.configRegistry.RemoveConfig(request.ConfigName); err != nil {
		http.Error(rw, fmt.Sprintf("Failed to remove config: %v", err), http.StatusBadRequest)
		return
	}

	// Delete config file
	if err := os.Remove(configMeta.FilePath); err != nil {
		w.logger.Warn("Failed to delete config file", "error", err, "path", configMeta.FilePath)
		// Continue anyway, registry is already updated
	}

	// Save registry
	if err := w.configRegistry.Save(w.registryPath); err != nil {
		w.logger.Error("Failed to save registry", "error", err)
		http.Error(rw, "Failed to save registry", http.StatusInternalServerError)
		return
	}

	w.logger.Info("Config deleted successfully", "name", request.ConfigName)

	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Configuration deleted: %s", request.ConfigName),
	})
}

// handleConfigRename handles configuration renaming
func (w *WebUIServer) handleConfigRename(rw http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		OldName string `json:"oldName"`
		NewName string `json:"newName"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(rw, "Invalid request body", http.StatusBadRequest)
		return
	}

	if request.OldName == "" || request.NewName == "" {
		http.Error(rw, "Both old name and new name are required", http.StatusBadRequest)
		return
	}

	// Get config metadata before renaming
	configMeta, err := w.configRegistry.GetConfig(request.OldName)
	if err != nil {
		http.Error(rw, fmt.Sprintf("Configuration not found: %s", request.OldName), http.StatusNotFound)
		return
	}

	// Rename in registry
	if err := w.configRegistry.RenameConfig(request.OldName, request.NewName); err != nil {
		http.Error(rw, fmt.Sprintf("Failed to rename config: %v", err), http.StatusBadRequest)
		return
	}

	// Generate new file path
	newFileName := fmt.Sprintf("config_%s.yaml", request.NewName)
	newFilePath := filepath.Join(w.configDir, newFileName)

	// Rename config file
	if err := os.Rename(configMeta.FilePath, newFilePath); err != nil {
		// Rollback registry change
		w.configRegistry.RenameConfig(request.NewName, request.OldName)
		http.Error(rw, fmt.Sprintf("Failed to rename config file: %v", err), http.StatusInternalServerError)
		return
	}

	// Update file path in registry
	updatedMeta, _ := w.configRegistry.GetConfig(request.NewName)
	updatedMeta.FilePath = newFilePath
	w.configRegistry.AddConfig(*updatedMeta)

	// Save registry
	if err := w.configRegistry.Save(w.registryPath); err != nil {
		w.logger.Error("Failed to save registry", "error", err)
		http.Error(rw, "Failed to save registry", http.StatusInternalServerError)
		return
	}

	w.logger.Info("Config renamed successfully", "oldName", request.OldName, "newName", request.NewName)

	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Configuration renamed from '%s' to '%s'", request.OldName, request.NewName),
	})
}

// handleConfigContent supports getting and updating raw YAML of a configuration
// GET  /api/configs/content?name={configName} -> { success, name, content }
// PUT  /api/configs/content { name, content } -> { success }
func (w *WebUIServer) handleConfigContent(rw http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		name := r.URL.Query().Get("name")
		if name == "" {
			http.Error(rw, "Config name is required", http.StatusBadRequest)
			return
		}

		meta, err := w.configRegistry.GetConfig(name)
		if err != nil {
			http.Error(rw, fmt.Sprintf("Configuration not found: %s", name), http.StatusNotFound)
			return
		}

		data, err := os.ReadFile(meta.FilePath)
		if err != nil {
			w.logger.Error("Failed to read config file", "error", err, "path", meta.FilePath)
			http.Error(rw, "Failed to read config", http.StatusInternalServerError)
			return
		}

		rw.Header().Set("Content-Type", "application/json")
		json.NewEncoder(rw).Encode(map[string]any{
			"success": true,
			"name":    name,
			"content": string(data),
		})
		return

	case http.MethodPut:
		var req struct {
			Name    string `json:"name"`
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(rw, "Invalid request body", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.Name) == "" {
			http.Error(rw, "Config name is required", http.StatusBadRequest)
			return
		}

		meta, err := w.configRegistry.GetConfig(req.Name)
		if err != nil {
			http.Error(rw, fmt.Sprintf("Configuration not found: %s", req.Name), http.StatusNotFound)
			return
		}

		// Validate YAML syntax by unmarshalling
		var syntaxCheck any
		if err := yaml.Unmarshal([]byte(req.Content), &syntaxCheck); err != nil {
			http.Error(rw, fmt.Sprintf("Invalid YAML: %v", err), http.StatusBadRequest)
			return
		}

		// Ensure directory exists
		if err := os.MkdirAll(filepath.Dir(meta.FilePath), 0o755); err != nil {
			w.logger.Error("Failed to create config directory", "error", err, "path", filepath.Dir(meta.FilePath))
			http.Error(rw, fmt.Sprintf("Failed to prepare directory: %v", err), http.StatusInternalServerError)
			return
		}

		// Write back to file (create if not exists)
		f, err := os.OpenFile(meta.FilePath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0o644)
		if err != nil {
			w.logger.Error("Failed to open config file for write", "error", err, "path", meta.FilePath)
			status := http.StatusInternalServerError
			if os.IsPermission(err) {
				status = http.StatusForbidden
			}
			http.Error(rw, fmt.Sprintf("Failed to write config file: %v (path: %s)", err, meta.FilePath), status)
			return
		}
		if _, err := f.Write([]byte(req.Content)); err != nil {
			f.Close()
			w.logger.Error("Failed to write config content", "error", err, "path", meta.FilePath)
			http.Error(rw, fmt.Sprintf("Failed to save config content: %v", err), http.StatusInternalServerError)
			return
		}
		if err := f.Close(); err != nil {
			w.logger.Warn("Error closing config file after write", "error", err)
		}

		// Update registry metadata (UpdatedAt)
		meta.UpdatedAt = time.Now()
		w.configRegistry.AddConfig(*meta)
		if err := w.configRegistry.Save(w.registryPath); err != nil {
			w.logger.Warn("Failed to save registry after edit", "error", err)
		}

		// If this is the active config, the file watcher will reload automatically
		rw.Header().Set("Content-Type", "application/json")
		json.NewEncoder(rw).Encode(map[string]any{
			"success": true,
			"message": "Configuration saved",
			"active":  meta.IsActive,
		})
		return

	default:
		http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
}

// handleConfigExport streams a single YAML config file to the client
func (w *WebUIServer) handleConfigExport(rw http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(rw, "Config name is required", http.StatusBadRequest)
		return
	}

	meta, err := w.configRegistry.GetConfig(name)
	if err != nil {
		http.Error(rw, fmt.Sprintf("Configuration not found: %s", name), http.StatusNotFound)
		return
	}

	data, err := os.ReadFile(meta.FilePath)
	if err != nil {
		w.logger.Error("Failed to read config for export", "error", err, "path", meta.FilePath)
		http.Error(rw, "Failed to read config", http.StatusInternalServerError)
		return
	}

	fileName := fmt.Sprintf("%s.yaml", strings.TrimSpace(name))
	rw.Header().Set("Content-Type", "application/x-yaml")
	rw.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	rw.WriteHeader(http.StatusOK)
	rw.Write(data)
}

// handleConfigExportAll exports all known configuration YAMLs in a single ZIP
func (w *WebUIServer) handleConfigExportAll(rw http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	configs := w.configRegistry.GetAllConfigs()
	if len(configs) == 0 {
		http.Error(rw, "No configurations to export", http.StatusNotFound)
		return
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	for _, meta := range configs {
		// Skip files that don't exist
		if _, err := os.Stat(meta.FilePath); err != nil {
			continue
		}

		data, err := os.ReadFile(meta.FilePath)
		if err != nil {
			w.logger.Warn("Skip config in export due to read error", "name", meta.Name, "error", err)
			continue
		}

		entryName := fmt.Sprintf("config_%s.yaml", strings.TrimSpace(meta.Name))
		f, err := zw.Create(entryName)
		if err != nil {
			w.logger.Warn("Failed to add file to zip", "name", meta.Name, "error", err)
			continue
		}
		if _, err := f.Write(data); err != nil {
			w.logger.Warn("Failed writing file to zip", "name", meta.Name, "error", err)
			continue
		}
	}

	// Optionally include registry to retain metadata
	if _, err := os.Stat(w.registryPath); err == nil {
		regData, err := os.ReadFile(w.registryPath)
		if err == nil {
			if f, err := zw.Create("registry.yaml"); err == nil {
				f.Write(regData)
			}
		}
	}

	_ = zw.Close()

	fileName := fmt.Sprintf("configs_%d.zip", time.Now().Unix())
	rw.Header().Set("Content-Type", "application/zip")
	rw.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	rw.WriteHeader(http.StatusOK)
	rw.Write(buf.Bytes())
}
