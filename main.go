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
	"endpoint_forwarder/internal/transport"
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

	// Setup initial logger (will be updated when config is loaded)
	logger := setupLogger(config.LoggingConfig{Level: "info", Format: "text"})
	slog.SetDefault(logger)

	// Create configuration watcher
	configWatcher, err := config.NewConfigWatcher(*configPath, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create configuration watcher: %v\n", err)
		os.Exit(1)
	}
	defer configWatcher.Close()

	// Get initial configuration
	cfg := configWatcher.GetConfig()

	// Update logger with config settings
	logger = setupLogger(cfg.Logging)
	slog.SetDefault(logger)

	logger.Info("🚀 Claude Request Forwarder 启动中...",
		"version", version,
		"commit", commit,
		"build_date", date,
		"config_file", *configPath,
		"endpoints_count", len(cfg.Endpoints),
		"strategy", cfg.Strategy.Type)

	// Display proxy configuration
	if cfg.Proxy.Enabled {
		proxyInfo := transport.GetProxyInfo(cfg)
		logger.Info("🔗 " + proxyInfo)
	} else {
		logger.Info("🔗 代理未启用，将直接连接目标端点")
	}

	// Display security information during startup
	if cfg.Auth.Enabled {
		logger.Info("🔐 鉴权已启用，访问需要Bearer Token验证")
	} else {
		logger.Info("🔓 鉴权已禁用，所有请求将直接转发")
		// Pre-warn about non-localhost binding without auth
		if cfg.Server.Host != "127.0.0.1" && cfg.Server.Host != "localhost" && cfg.Server.Host != "::1" {
			logger.Warn("⚠️  注意：将在非本地地址启动但未启用鉴权，请确保网络环境安全")
		}
	}

	// Create endpoint manager
	endpointManager := endpoint.NewManager(cfg)
	endpointManager.Start()
	defer endpointManager.Stop()

	// Create proxy handler
	proxyHandler := proxy.NewHandler(endpointManager, cfg)

	// Create middleware
	loggingMiddleware := middleware.NewLoggingMiddleware(logger)
	monitoringMiddleware := middleware.NewMonitoringMiddleware(endpointManager)
	authMiddleware := middleware.NewAuthMiddleware(cfg.Auth)

	// Setup configuration reload callback to update components
	configWatcher.AddReloadCallback(func(newCfg *config.Config) {
		// Update logger
		newLogger := setupLogger(newCfg.Logging)
		slog.SetDefault(newLogger)
		
		// Update endpoint manager
		endpointManager.UpdateConfig(newCfg)
		
		// Update proxy handler
		proxyHandler.UpdateConfig(newCfg)
		
		// Update auth middleware
		authMiddleware.UpdateConfig(newCfg.Auth)
		
		newLogger.Info("🔄 所有组件已更新为新配置")
	})

	logger.Info("🔄 配置文件自动重载已启用")

	// Setup HTTP server
	mux := http.NewServeMux()

	// Register monitoring endpoints
	monitoringMiddleware.RegisterHealthEndpoint(mux)

	// Register proxy handler for all other requests with middleware chain
	mux.Handle("/", loggingMiddleware.Wrap(authMiddleware.Wrap(proxyHandler)))

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
		
		// Security warning for non-localhost addresses
		if cfg.Server.Host != "127.0.0.1" && cfg.Server.Host != "localhost" && cfg.Server.Host != "::1" {
			if !cfg.Auth.Enabled {
				logger.Warn("⚠️  安全警告：服务器绑定到非本地地址但未启用鉴权！")
				logger.Warn("🔒 强烈建议启用鉴权以保护您的端点访问")
				logger.Warn("📝 在配置文件中设置 auth.enabled: true 和 auth.token 来启用鉴权")
			} else {
				logger.Info("🔒 已启用鉴权保护，服务器可安全对外开放")
			}
		}
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