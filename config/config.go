package config

import (
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server       ServerConfig     `yaml:"server"`
	Strategy     StrategyConfig   `yaml:"strategy"`
	Retry        RetryConfig      `yaml:"retry"`
	Health       HealthConfig     `yaml:"health"`
	Logging      LoggingConfig    `yaml:"logging"`
	Streaming    StreamingConfig  `yaml:"streaming"`
	Proxy        ProxyConfig      `yaml:"proxy"`
	Auth         AuthConfig       `yaml:"auth"`
	TUI          TUIConfig        `yaml:"tui"`           // TUI configuration
	GlobalTimeout time.Duration   `yaml:"global_timeout"` // Global timeout for non-streaming requests
	Endpoints    []EndpointConfig `yaml:"endpoints"`
	
	// Runtime priority override (not serialized to YAML)
	PrimaryEndpoint string `yaml:"-"` // Primary endpoint name from command line
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type StrategyConfig struct {
	Type              string        `yaml:"type"` // "priority" or "fastest"
	FastTestEnabled   bool          `yaml:"fast_test_enabled"`   // Enable pre-request fast testing
	FastTestCacheTTL  time.Duration `yaml:"fast_test_cache_ttl"` // Cache TTL for fast test results
	FastTestTimeout   time.Duration `yaml:"fast_test_timeout"`   // Timeout for individual fast tests
	FastTestPath      string        `yaml:"fast_test_path"`      // Path for fast testing (default: health path)
}

type RetryConfig struct {
	MaxAttempts int           `yaml:"max_attempts"`
	BaseDelay   time.Duration `yaml:"base_delay"`
	MaxDelay    time.Duration `yaml:"max_delay"`
	Multiplier  float64       `yaml:"multiplier"`
}

type HealthConfig struct {
	CheckInterval time.Duration `yaml:"check_interval"`
	Timeout       time.Duration `yaml:"timeout"`
	HealthPath    string        `yaml:"health_path"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"` // "json" or "text"
}

type StreamingConfig struct {
	HeartbeatInterval time.Duration `yaml:"heartbeat_interval"`
	ReadTimeout       time.Duration `yaml:"read_timeout"`
	MaxIdleTime       time.Duration `yaml:"max_idle_time"`
}

type ProxyConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Type     string `yaml:"type"`     // "http", "https", "socks5"
	URL      string `yaml:"url"`      // Complete proxy URL
	Host     string `yaml:"host"`     // Proxy host
	Port     int    `yaml:"port"`     // Proxy port
	Username string `yaml:"username"` // Optional auth username
	Password string `yaml:"password"` // Optional auth password
}

type AuthConfig struct {
	Enabled bool   `yaml:"enabled"`                   // Enable authentication, default: false
	Token   string `yaml:"token,omitempty"`           // Bearer token for authentication
}

type TUIConfig struct {
	Enabled       bool          `yaml:"enabled"`        // Enable TUI interface, default: true
	UpdateInterval time.Duration `yaml:"update_interval"` // TUI refresh interval, default: 1s
}

type EndpointConfig struct {
	Name     string            `yaml:"name"`
	URL      string            `yaml:"url"`
	Priority int               `yaml:"priority"`
	Token    string            `yaml:"token,omitempty"`
	Timeout  time.Duration     `yaml:"timeout"`
	Headers  map[string]string `yaml:"headers,omitempty"`
}

// LoadConfig loads configuration from file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults
	config.setDefaults()

	// Validate configuration
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// setDefaults sets default values for configuration
func (c *Config) setDefaults() {
	if c.Server.Host == "" {
		c.Server.Host = "localhost"
	}
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.Strategy.Type == "" {
		c.Strategy.Type = "priority"
	}
	// Set fast test defaults
	if c.Strategy.FastTestCacheTTL == 0 {
		c.Strategy.FastTestCacheTTL = 3 * time.Second // Default 3 seconds cache
	}
	if c.Strategy.FastTestTimeout == 0 {
		c.Strategy.FastTestTimeout = 1 * time.Second // Default 1 second timeout for fast tests
	}
	if c.Strategy.FastTestPath == "" {
		c.Strategy.FastTestPath = c.Health.HealthPath // Default to health path
	}
	if c.Retry.MaxAttempts == 0 {
		c.Retry.MaxAttempts = 3
	}
	if c.Retry.BaseDelay == 0 {
		c.Retry.BaseDelay = time.Second
	}
	if c.Retry.MaxDelay == 0 {
		c.Retry.MaxDelay = 30 * time.Second
	}
	if c.Retry.Multiplier == 0 {
		c.Retry.Multiplier = 2.0
	}
	if c.Health.CheckInterval == 0 {
		c.Health.CheckInterval = 30 * time.Second
	}
	if c.Health.Timeout == 0 {
		c.Health.Timeout = 5 * time.Second
	}
	if c.Health.HealthPath == "" {
		c.Health.HealthPath = "/v1/models"
	}
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	if c.Logging.Format == "" {
		c.Logging.Format = "text"
	}
	if c.Streaming.HeartbeatInterval == 0 {
		c.Streaming.HeartbeatInterval = 30 * time.Second
	}
	if c.Streaming.ReadTimeout == 0 {
		c.Streaming.ReadTimeout = 10 * time.Second
	}
	if c.Streaming.MaxIdleTime == 0 {
		c.Streaming.MaxIdleTime = 120 * time.Second
	}

	// Set global timeout default
	if c.GlobalTimeout == 0 {
		c.GlobalTimeout = 300 * time.Second // Default 5 minutes for non-streaming requests
	}

	// Set TUI defaults
	if c.TUI.UpdateInterval == 0 {
		c.TUI.UpdateInterval = 2 * time.Second // Default 2 second refresh (reduced from 1s)
	}
	// TUI enabled defaults to true if not explicitly set in YAML
	// This will be handled by the application logic

	// Set default timeouts for endpoints and handle parameter inheritance
	var defaultEndpoint *EndpointConfig
	if len(c.Endpoints) > 0 {
		defaultEndpoint = &c.Endpoints[0]
	}

	for i := range c.Endpoints {
		// Set default timeout if not specified
		if c.Endpoints[i].Timeout == 0 {
			if defaultEndpoint != nil && defaultEndpoint.Timeout != 0 {
				// Inherit timeout from first endpoint
				c.Endpoints[i].Timeout = defaultEndpoint.Timeout
			} else {
				// Use global timeout setting
				c.Endpoints[i].Timeout = c.GlobalTimeout
			}
		}
		
		// Inherit token from first endpoint if not specified
		if c.Endpoints[i].Token == "" && defaultEndpoint != nil && defaultEndpoint.Token != "" {
			c.Endpoints[i].Token = defaultEndpoint.Token
		}
		
		// Inherit headers from first endpoint if not specified
		if len(c.Endpoints[i].Headers) == 0 && defaultEndpoint != nil && len(defaultEndpoint.Headers) > 0 {
			// Copy headers from first endpoint
			c.Endpoints[i].Headers = make(map[string]string)
			for key, value := range defaultEndpoint.Headers {
				c.Endpoints[i].Headers[key] = value
			}
		} else if len(c.Endpoints[i].Headers) > 0 && defaultEndpoint != nil && len(defaultEndpoint.Headers) > 0 {
			// Merge headers: inherit from first endpoint, but allow override
			mergedHeaders := make(map[string]string)
			
			// First, copy all headers from the first endpoint
			for key, value := range defaultEndpoint.Headers {
				mergedHeaders[key] = value
			}
			
			// Then, override with endpoint-specific headers
			for key, value := range c.Endpoints[i].Headers {
				mergedHeaders[key] = value
			}
			
			c.Endpoints[i].Headers = mergedHeaders
		}
	}
}

// ApplyPrimaryEndpoint applies primary endpoint override from command line
func (c *Config) ApplyPrimaryEndpoint() {
	if c.PrimaryEndpoint == "" {
		return
	}
	
	// Find the specified endpoint
	var primaryIndex = -1
	for i, endpoint := range c.Endpoints {
		if endpoint.Name == c.PrimaryEndpoint {
			primaryIndex = i
			break
		}
	}
	
	if primaryIndex == -1 {
		return // Endpoint not found, ignore silently
	}
	
	// Set the primary endpoint to priority 1 and adjust others
	c.Endpoints[primaryIndex].Priority = 1
	
	// Adjust other endpoints' priorities to be higher than 1
	for i := range c.Endpoints {
		if i != primaryIndex && c.Endpoints[i].Priority <= 1 {
			c.Endpoints[i].Priority = c.Endpoints[i].Priority + 10 // Push other endpoints down
		}
	}
}

// validate validates the configuration
func (c *Config) validate() error {
	if len(c.Endpoints) == 0 {
		return fmt.Errorf("at least one endpoint must be configured")
	}

	if c.Strategy.Type != "priority" && c.Strategy.Type != "fastest" {
		return fmt.Errorf("strategy type must be 'priority' or 'fastest'")
	}

	// Validate proxy configuration
	if c.Proxy.Enabled {
		if c.Proxy.Type == "" {
			return fmt.Errorf("proxy type is required when proxy is enabled")
		}
		if c.Proxy.Type != "http" && c.Proxy.Type != "https" && c.Proxy.Type != "socks5" {
			return fmt.Errorf("proxy type must be 'http', 'https', or 'socks5'")
		}
		if c.Proxy.URL == "" && (c.Proxy.Host == "" || c.Proxy.Port == 0) {
			return fmt.Errorf("proxy URL or host:port must be specified when proxy is enabled")
		}
	}

	for i, endpoint := range c.Endpoints {
		if endpoint.Name == "" {
			return fmt.Errorf("endpoint %d: name is required", i)
		}
		if endpoint.URL == "" {
			return fmt.Errorf("endpoint %s: URL is required", endpoint.Name)
		}
		if endpoint.Priority < 0 {
			return fmt.Errorf("endpoint %s: priority must be non-negative", endpoint.Name)
		}
	}

	return nil
}

// ConfigWatcher handles automatic configuration reloading
type ConfigWatcher struct {
	configPath    string
	config        *Config
	mutex         sync.RWMutex
	watcher       *fsnotify.Watcher
	logger        *slog.Logger
	callbacks     []func(*Config)
	lastModTime   time.Time
	debounceTimer *time.Timer
}

// NewConfigWatcher creates a new configuration watcher
func NewConfigWatcher(configPath string, logger *slog.Logger) (*ConfigWatcher, error) {
	// Load initial configuration
	config, err := LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load initial config: %w", err)
	}

	// Get initial modification time
	fileInfo, err := os.Stat(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Create file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	cw := &ConfigWatcher{
		configPath:  configPath,
		config:      config,
		watcher:     watcher,
		logger:      logger,
		callbacks:   make([]func(*Config), 0),
		lastModTime: fileInfo.ModTime(),
	}

	// Add config file to watcher
	if err := watcher.Add(configPath); err != nil {
		watcher.Close()
		return nil, fmt.Errorf("failed to watch config file: %w", err)
	}

	// Start watching in background
	go cw.watchLoop()

	return cw, nil
}

// GetConfig returns the current configuration (thread-safe)
func (cw *ConfigWatcher) GetConfig() *Config {
	cw.mutex.RLock()
	defer cw.mutex.RUnlock()
	return cw.config
}

// UpdateLogger updates the logger used by the config watcher
func (cw *ConfigWatcher) UpdateLogger(logger *slog.Logger) {
	cw.mutex.Lock()
	defer cw.mutex.Unlock()
	cw.logger = logger
}

// AddReloadCallback adds a callback function that will be called when config is reloaded
func (cw *ConfigWatcher) AddReloadCallback(callback func(*Config)) {
	cw.mutex.Lock()
	defer cw.mutex.Unlock()
	cw.callbacks = append(cw.callbacks, callback)
}

// watchLoop monitors the config file for changes
func (cw *ConfigWatcher) watchLoop() {
	for {
		select {
		case event, ok := <-cw.watcher.Events:
			if !ok {
				return
			}

			// Handle file write events
			if event.Has(fsnotify.Write) {
				// Check if file was actually modified by comparing modification time
				fileInfo, err := os.Stat(cw.configPath)
				if err != nil {
					cw.logger.Warn("⚠️ 无法获取配置文件信息", "error", err)
					continue
				}

				// Skip if modification time hasn't changed
				if !fileInfo.ModTime().After(cw.lastModTime) {
					continue
				}

				cw.lastModTime = fileInfo.ModTime()
				
				// Cancel any existing debounce timer
				if cw.debounceTimer != nil {
					cw.debounceTimer.Stop()
				}

				// Set up debounce timer to avoid multiple rapid reloads
				cw.debounceTimer = time.AfterFunc(500*time.Millisecond, func() {
					cw.logger.Info("🔄 检测到配置文件变更，正在重新加载...", "file", event.Name)
					if err := cw.reloadConfig(); err != nil {
						cw.logger.Error("❌ 配置文件重新加载失败", "error", err)
					} else {
						cw.logger.Info("✅ 配置文件重新加载成功")
					}
				})
			}

			// Handle file rename/remove events (some editors rename files during save)
			if event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
				// Re-add the file to watcher in case it was recreated
				time.Sleep(100 * time.Millisecond) // Give time for the file to be recreated
				if _, err := os.Stat(cw.configPath); err == nil {
					cw.watcher.Add(cw.configPath)
					cw.logger.Info("🔄 重新监听配置文件", "file", cw.configPath)
				}
			}

		case err, ok := <-cw.watcher.Errors:
			if !ok {
				return
			}
			cw.logger.Error("⚠️ 配置文件监听错误", "error", err)
		}
	}
}

// reloadConfig reloads the configuration from file
func (cw *ConfigWatcher) reloadConfig() error {
	newConfig, err := LoadConfig(cw.configPath)
	if err != nil {
		return err
	}

	cw.mutex.Lock()
	oldConfig := cw.config
	cw.config = newConfig
	callbacks := make([]func(*Config), len(cw.callbacks))
	copy(callbacks, cw.callbacks)
	cw.mutex.Unlock()

	// Call all registered callbacks
	for _, callback := range callbacks {
		callback(newConfig)
	}

	// Log configuration changes
	cw.logConfigChanges(oldConfig, newConfig)

	return nil
}

// logConfigChanges logs the key differences between old and new configurations
func (cw *ConfigWatcher) logConfigChanges(oldConfig, newConfig *Config) {
	if len(oldConfig.Endpoints) != len(newConfig.Endpoints) {
		cw.logger.Info("📡 端点数量变更",
			"old_count", len(oldConfig.Endpoints),
			"new_count", len(newConfig.Endpoints))
	}

	if oldConfig.Server.Port != newConfig.Server.Port {
		cw.logger.Info("🌐 服务器端口变更",
			"old_port", oldConfig.Server.Port,
			"new_port", newConfig.Server.Port)
	}

	if oldConfig.Strategy.Type != newConfig.Strategy.Type {
		cw.logger.Info("🎯 策略类型变更",
			"old_strategy", oldConfig.Strategy.Type,
			"new_strategy", newConfig.Strategy.Type)
	}

	if oldConfig.Auth.Enabled != newConfig.Auth.Enabled {
		cw.logger.Info("🔐 鉴权状态变更",
			"old_enabled", oldConfig.Auth.Enabled,
			"new_enabled", newConfig.Auth.Enabled)
	}
}

// Close stops the configuration watcher
func (cw *ConfigWatcher) Close() error {
	// Cancel any pending debounce timer
	if cw.debounceTimer != nil {
		cw.debounceTimer.Stop()
	}
	return cw.watcher.Close()
}