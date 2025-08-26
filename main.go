package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"endpoint_forwarder/config"
	"endpoint_forwarder/internal/endpoint"
	"endpoint_forwarder/internal/middleware"
	"endpoint_forwarder/internal/proxy"
)

var (
	configPath = flag.String("config", "config/example.yaml", "Path to configuration file")
	showVersion = flag.Bool("version", false, "Show version information")
	
	// Build-time variables (set via ldflags)
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	flag.Parse()

	// Handle version flag
	if *showVersion {
		fmt.Printf("Claude Request Forwarder\n")
		fmt.Printf("Version: %s\n", version)
		fmt.Printf("Commit: %s\n", commit)
		fmt.Printf("Built: %s\n", date)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Setup logger
	logger := setupLogger(cfg.Logging)
	slog.SetDefault(logger)

	logger.Info("🚀 Claude Request Forwarder 启动中...",
		"version", version,
		"commit", commit,
		"build_date", date,
		"config_file", *configPath,
		"endpoints_count", len(cfg.Endpoints),
		"strategy", cfg.Strategy.Type)

	// Create endpoint manager
	endpointManager := endpoint.NewManager(cfg)
	endpointManager.Start()
	defer endpointManager.Stop()

	// Create proxy handler
	proxyHandler := proxy.NewHandler(endpointManager, cfg)

	// Create middleware
	loggingMiddleware := middleware.NewLoggingMiddleware(logger)
	monitoringMiddleware := middleware.NewMonitoringMiddleware(endpointManager)

	// Setup HTTP server
	mux := http.NewServeMux()

	// Register monitoring endpoints
	monitoringMiddleware.RegisterHealthEndpoint(mux)

	// Register proxy handler for all other requests
	mux.Handle("/", loggingMiddleware.Wrap(proxyHandler))

	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      mux,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 0, // No write timeout for streaming
		IdleTimeout:  120 * time.Second,
	}

	// Start server in a goroutine
	serverErr := make(chan error, 1)
	go func() {
		logger.Info("🌐 HTTP 服务器启动中...",
			"address", server.Addr,
			"endpoints_count", len(cfg.Endpoints))

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()
	
	// Give server a moment to start
	time.Sleep(100 * time.Millisecond)
	
	// Check if server started successfully
	select {
	case err := <-serverErr:
		logger.Error("❌ 服务器启动失败", "error", err)
		os.Exit(1)
	default:
		// Server started successfully
		baseURL := fmt.Sprintf("http://%s:%d", cfg.Server.Host, cfg.Server.Port)
		logger.Info("✅ 服务器启动成功！")
		logger.Info("📋 配置说明：请在 Claude Code 的 settings.json 中设置")
		logger.Info("🔧 ANTHROPIC_BASE_URL: " + baseURL)
		logger.Info("📡 服务器地址: " + baseURL)
	}

	// Wait for interrupt signal
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	// Block until we receive a signal or server error
	select {
	case err := <-serverErr:
		logger.Error("❌ 服务器运行时错误", "error", err)
		os.Exit(1)
	case sig := <-interrupt:
		logger.Info("📡 收到终止信号，开始优雅关闭...", "signal", sig)
	}

	// Graceful shutdown
	logger.Info("🛑 正在关闭服务器...")
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("❌ 服务器关闭失败", "error", err)
		os.Exit(1)
	}

	logger.Info("✅ 服务器已安全关闭")
}

// setupLogger configures the structured logger
func setupLogger(cfg config.LoggingConfig) *slog.Logger {
	var level slog.Level
	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	var handler slog.Handler
	// Create a custom handler that only outputs the message
	handler = &SimpleHandler{level: level}

	return slog.New(handler)
}

// SimpleHandler only outputs the log message without any metadata
type SimpleHandler struct {
	level slog.Level
}

func (h *SimpleHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *SimpleHandler) Handle(_ context.Context, r slog.Record) error {
	// Only output the message
	fmt.Println(r.Message)
	return nil
}

func (h *SimpleHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// Return the same handler since we don't use attributes
	return h
}

func (h *SimpleHandler) WithGroup(name string) slog.Handler {
	// Return the same handler since we don't use groups
	return h
}