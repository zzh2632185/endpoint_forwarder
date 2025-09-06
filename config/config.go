package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"
)

// ConfigMetadata represents metadata for a configuration
type ConfigMetadata struct {
	Name        string    `json:"name" yaml:"name"`               // 配置名称
	FilePath    string    `json:"filePath" yaml:"file_path"`      // 配置文件路径
	Description string    `json:"description" yaml:"description"` // 配置描述
	CreatedAt   time.Time `json:"createdAt" yaml:"created_at"`    // 创建时间
	UpdatedAt   time.Time `json:"updatedAt" yaml:"updated_at"`    // 更新时间
	IsActive    bool      `json:"isActive" yaml:"is_active"`      // 是否为当前活动配置
}

// ConfigRegistry manages multiple configurations
type ConfigRegistry struct {
	Configs      []ConfigMetadata `json:"configs" yaml:"configs"`
	ActiveConfig string           `json:"activeConfig" yaml:"active_config"`
	LastUpdated  time.Time        `json:"lastUpdated" yaml:"last_updated"`
	mutex        sync.RWMutex     `json:"-" yaml:"-"`
}

type Config struct {
	Server        ServerConfig     `yaml:"server"`
	Strategy      StrategyConfig   `yaml:"strategy"`
	Retry         RetryConfig      `yaml:"retry"`
	Health        HealthConfig     `yaml:"health"`
	Logging       LoggingConfig    `yaml:"logging"`
	Streaming     StreamingConfig  `yaml:"streaming"`
	Group         GroupConfig      `yaml:"group"` // Group configuration
	Proxy         ProxyConfig      `yaml:"proxy"`
	Auth          AuthConfig       `yaml:"auth"`
	TUI           TUIConfig        `yaml:"tui"`            // TUI configuration
	WebUI         WebUIConfig      `yaml:"webui"`          // WebUI configuration
	GlobalTimeout time.Duration    `yaml:"global_timeout"` // Global timeout for non-streaming requests
	Endpoints     []EndpointConfig `yaml:"endpoints"`
	// Runtime priority override (not serialized to YAML)
	PrimaryEndpoint string `yaml:"-"` // Primary endpoint name from command line
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type StrategyConfig struct {
	Type             string        `yaml:"type"`                // "priority" or "fastest"
	FastTestEnabled  bool          `yaml:"fast_test_enabled"`   // Enable pre-request fast testing
	FastTestCacheTTL time.Duration `yaml:"fast_test_cache_ttl"` // Cache TTL for fast test results
	FastTestTimeout  time.Duration `yaml:"fast_test_timeout"`   // Timeout for individual fast tests
	FastTestPath     string        `yaml:"fast_test_path"`      // Path for fast testing (default: health path)
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
	Level                string `yaml:"level"`
	Format               string `yaml:"format"`                 // "json" or "text"
	FileEnabled          bool   `yaml:"file_enabled"`           // Enable file logging
	FilePath             string `yaml:"file_path"`              // Log file path
	MaxFileSize          string `yaml:"max_file_size"`          // Max file size (e.g., "100MB")
	MaxFiles             int    `yaml:"max_files"`              // Max number of rotated files to keep
	CompressRotated      bool   `yaml:"compress_rotated"`       // Compress rotated log files
	DisableResponseLimit bool   `yaml:"disable_response_limit"` // Disable response content output limit when file logging is enabled
}

type StreamingConfig struct {
	HeartbeatInterval time.Duration `yaml:"heartbeat_interval"`
	ReadTimeout       time.Duration `yaml:"read_timeout"`
	MaxIdleTime       time.Duration `yaml:"max_idle_time"`
}

type GroupConfig struct {
	Cooldown       time.Duration `yaml:"cooldown"` // Cooldown duration for groups when all endpoints fail
	MaxRetries     int           `yaml:"max_retries"` // Maximum retry attempts per group before cooldown
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
	Enabled bool   `yaml:"enabled"`         // Enable authentication, default: false
	Token   string `yaml:"token,omitempty"` // Bearer token for authentication
}

type TUIConfig struct {
	Enabled           bool          `yaml:"enabled"`             // Enable TUI interface, default: true
	UpdateInterval    time.Duration `yaml:"update_interval"`     // TUI refresh interval, default: 1s
	SavePriorityEdits bool          `yaml:"save_priority_edits"` // Save priority edits to config file, default: false
}

type WebUIConfig struct {
	Enabled  bool   `yaml:"enabled"`  // Enable WebUI interface, default: false
	Host     string `yaml:"host"`     // WebUI host, default: "127.0.0.1"
	Port     int    `yaml:"port"`     // WebUI port, default: 8003
	Password string `yaml:"password"` // WebUI access password, if empty no authentication required
}

type EndpointConfig struct {
	Name          string            `yaml:"name"`
	URL           string            `yaml:"url"`
	Priority      int               `yaml:"priority"`
	Group         string            `yaml:"group,omitempty"`
	GroupPriority int               `yaml:"group-priority,omitempty"`
	Token         string            `yaml:"token,omitempty"`
	ApiKey        string            `yaml:"api-key,omitempty"`
	Timeout       time.Duration     `yaml:"timeout"`
	Headers       map[string]string `yaml:"headers,omitempty"`
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
	// Set file logging defaults
	if c.Logging.FileEnabled && c.Logging.FilePath == "" {
		c.Logging.FilePath = "logs/app.log"
	}
	if c.Logging.FileEnabled && c.Logging.MaxFileSize == "" {
		c.Logging.MaxFileSize = "100MB"
	}
	if c.Logging.FileEnabled && c.Logging.MaxFiles == 0 {
		c.Logging.MaxFiles = 10
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

	// Set group defaults
	if c.Group.Cooldown == 0 {
		c.Group.Cooldown = 600 * time.Second // Default 10 minutes cooldown for groups
	}
	if c.Group.MaxRetries == 0 {
		c.Group.MaxRetries = 3 // Default 3 retry attempts per group
	}

	// Set TUI defaults
	if c.TUI.UpdateInterval == 0 {
		c.TUI.UpdateInterval = 2 * time.Second // Default 2 second refresh (reduced from 1s)
	}
	// TUI enabled defaults to true if not explicitly set in YAML
	// This will be handled by the application logic
	// Save priority edits defaults to false for safety
	// Note: We don't set a default here since the zero value (false) is what we want

	// Set WebUI defaults
	if c.WebUI.Host == "" {
		c.WebUI.Host = "127.0.0.1"
	}
	if c.WebUI.Port == 0 {
		c.WebUI.Port = 8003
	}
	// WebUI enabled defaults to false if not explicitly set in YAML

	// Set default timeouts for endpoints and handle parameter inheritance (except tokens)
	var defaultEndpoint *EndpointConfig
	if len(c.Endpoints) > 0 {
		defaultEndpoint = &c.Endpoints[0]
	}

	// Handle group inheritance - endpoints inherit group settings from previous endpoint
	var currentGroup string = "Default" // Default group name
	var currentGroupPriority int = 1    // Default group priority

	for i := range c.Endpoints {
		// Handle group inheritance - check if this endpoint defines a new group
		if c.Endpoints[i].Group != "" {
			// Endpoint specifies a group, use it and update current group
			currentGroup = c.Endpoints[i].Group
			if c.Endpoints[i].GroupPriority != 0 {
				currentGroupPriority = c.Endpoints[i].GroupPriority
			}
		} else {
			// Endpoint doesn't specify group, inherit from previous
			c.Endpoints[i].Group = currentGroup
			c.Endpoints[i].GroupPriority = currentGroupPriority
		}

		// If GroupPriority is still 0 after inheritance, set default
		if c.Endpoints[i].GroupPriority == 0 {
			c.Endpoints[i].GroupPriority = currentGroupPriority
		}

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

		// NOTE: We do NOT inherit tokens here - tokens will be resolved dynamically at runtime
		// This allows for proper group-based token switching when groups fail

		// Inherit api-key from first endpoint if not specified
		if c.Endpoints[i].ApiKey == "" && defaultEndpoint != nil && defaultEndpoint.ApiKey != "" {
			c.Endpoints[i].ApiKey = defaultEndpoint.ApiKey
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
// Returns error if the specified endpoint is not found
func (c *Config) ApplyPrimaryEndpoint(logger *slog.Logger) error {
	if c.PrimaryEndpoint == "" {
		return nil
	}

	// Find the specified endpoint
	primaryIndex := c.findEndpointIndex(c.PrimaryEndpoint)
	if primaryIndex == -1 {
		// Create list of available endpoints for better error message
		var availableEndpoints []string
		for _, endpoint := range c.Endpoints {
			availableEndpoints = append(availableEndpoints, endpoint.Name)
		}

		err := fmt.Errorf("指定的主端点 '%s' 未找到，可用端点: %v", c.PrimaryEndpoint, availableEndpoints)
		if logger != nil {
			logger.Error(fmt.Sprintf("❌ 主端点设置失败 - 端点: %s, 可用端点: %v",
				c.PrimaryEndpoint, availableEndpoints))
		}
		return err
	}

	// Store original priority for logging
	originalPriority := c.Endpoints[primaryIndex].Priority

	// Set the primary endpoint to priority 1
	c.Endpoints[primaryIndex].Priority = 1

	// Adjust other endpoints' priorities to ensure they are lower than primary
	adjustedCount := 0
	for i := range c.Endpoints {
		if i != primaryIndex && c.Endpoints[i].Priority <= 1 {
			c.Endpoints[i].Priority = c.Endpoints[i].Priority + 2 // Use consistent increment
			adjustedCount++
		}
	}

	if logger != nil {
		logger.Info(fmt.Sprintf("✅ 主端点优先级设置成功 - 端点: %s, 原优先级: %d → 新优先级: %d, 调整了%d个其他端点",
			c.PrimaryEndpoint, originalPriority, 1, adjustedCount))
	}

	return nil
}

// findEndpointIndex finds the index of an endpoint by name
func (c *Config) findEndpointIndex(name string) int {
	for i, endpoint := range c.Endpoints {
		if endpoint.Name == name {
			return i
		}
	}
	return -1
}

// validate validates the configuration
func (c *Config) validate() error {
	if len(c.Endpoints) == 0 {
		return fmt.Errorf("at least one endpoint must be configured")
	}

	if c.Strategy.Type != "priority" && c.Strategy.Type != "fastest" && c.Strategy.Type != "round-robin" {
		return fmt.Errorf("strategy type must be 'priority', 'fastest', or 'round-robin'")
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
	registry      *ConfigRegistry
	registryPath  string
}

// NewConfigWatcher creates a new configuration watcher
func NewConfigWatcher(configPath string, logger *slog.Logger) (*ConfigWatcher, error) {
    // Normalize to absolute path for watcher reliability
    if abs, err := filepath.Abs(configPath); err == nil {
        configPath = abs
    }

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

	// Initialize config registry
	configDir := filepath.Dir(configPath)
	registryPath := filepath.Join(configDir, "registry.yaml")
	registry, err := ScanAndInitializeRegistry(configDir, registryPath, configPath)
	if err != nil {
		logger.Warn("Failed to initialize config registry", "error", err)
		registry = NewConfigRegistry()
	}

	cw := &ConfigWatcher{
		configPath:   configPath,
		config:       config,
		watcher:      watcher,
		logger:       logger,
		callbacks:    make([]func(*Config), 0),
		lastModTime:  fileInfo.ModTime(),
		registry:     registry,
		registryPath: registryPath,
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
					cw.logger.Warn(fmt.Sprintf("⚠️ 无法获取配置文件信息: %v", err))
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
					cw.logger.Info(fmt.Sprintf("🔄 检测到配置文件变更，正在重新加载... - 文件: %s", event.Name))
					if err := cw.reloadConfig(); err != nil {
						cw.logger.Error(fmt.Sprintf("❌ 配置文件重新加载失败: %v", err))
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
					cw.logger.Info(fmt.Sprintf("🔄 重新监听配置文件: %s", cw.configPath))
				}
			}

		case err, ok := <-cw.watcher.Errors:
			if !ok {
				return
			}
			cw.logger.Error(fmt.Sprintf("⚠️ 配置文件监听错误: %v", err))
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

// SaveConfig saves configuration to file
func SaveConfig(config *Config, path string) error {
	// Marshal config to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// SaveConfigWithComments saves configuration to file while preserving all comments
func SavePriorityConfigWithComments(config *Config, path string) error {
	// Read existing file to preserve comments
	yamlFile, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read existing config file: %w", err)
	}

	var rootNode yaml.Node
	if len(yamlFile) > 0 {
		// Decode existing YAML to preserve structure and comments
		if err := yaml.Unmarshal(yamlFile, &rootNode); err != nil {
			return fmt.Errorf("failed to decode existing YAML: %w", err)
		}
	} else {
		// Create new YAML structure if file doesn't exist
		rootNode = yaml.Node{}
		if err := rootNode.Encode(config); err != nil {
			return fmt.Errorf("failed to create new YAML structure: %w", err)
		}
	}

	// Update endpoint priorities in the YAML node tree
	if len(rootNode.Content) > 0 {
		mappingNode := rootNode.Content[0]

		// Find endpoints section
		for i := 0; i < len(mappingNode.Content); i += 2 {
			keyNode := mappingNode.Content[i]
			valueNode := mappingNode.Content[i+1]

			if keyNode.Value == "endpoints" {
				// Update each endpoint's priority
				for _, endpointNode := range valueNode.Content {
					var endpointName string
					var priorityNode *yaml.Node

					// Find name and priority nodes for this endpoint
					for j := 0; j < len(endpointNode.Content); j += 2 {
						fieldKey := endpointNode.Content[j]
						fieldValue := endpointNode.Content[j+1]

						if fieldKey.Value == "name" {
							endpointName = fieldValue.Value
						} else if fieldKey.Value == "priority" {
							priorityNode = fieldValue
						}
					}

					// Find the corresponding endpoint in config and update priority
					if endpointName != "" && priorityNode != nil {
						for _, endpoint := range config.Endpoints {
							if endpoint.Name == endpointName {
								priorityNode.Value = fmt.Sprintf("%d", endpoint.Priority)
								break
							}
						}
					}
				}
				break
			}
		}
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Directly write to the original file
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	// Encode with comments
	encoder := yaml.NewEncoder(file)
	encoder.SetIndent(2)
	if err := encoder.Encode(&rootNode); err != nil {
		return fmt.Errorf("failed to encode YAML: %w", err)
	}

	return nil
}

// NewConfigRegistry creates a new configuration registry
func NewConfigRegistry() *ConfigRegistry {
	return &ConfigRegistry{
		Configs:      make([]ConfigMetadata, 0),
		ActiveConfig: "",
		LastUpdated:  time.Now(),
	}
}

// LoadConfigRegistry loads the configuration registry from file
func LoadConfigRegistry(registryPath string) (*ConfigRegistry, error) {
	if _, err := os.Stat(registryPath); os.IsNotExist(err) {
		// Create new registry if file doesn't exist
		registry := NewConfigRegistry()
		if err := registry.Save(registryPath); err != nil {
			return nil, fmt.Errorf("failed to create registry file: %w", err)
		}
		return registry, nil
	}

	data, err := os.ReadFile(registryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read registry file: %w", err)
	}

	var registry ConfigRegistry
	if err := yaml.Unmarshal(data, &registry); err != nil {
		return nil, fmt.Errorf("failed to parse registry file: %w", err)
	}

	return &registry, nil
}

// Save saves the configuration registry to file
func (cr *ConfigRegistry) Save(registryPath string) error {
	cr.mutex.Lock()
	defer cr.mutex.Unlock()

	cr.LastUpdated = time.Now()

	data, err := yaml.Marshal(cr)
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(registryPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(registryPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write registry file: %w", err)
	}

	return nil
}

// AddConfig adds a new configuration to the registry
func (cr *ConfigRegistry) AddConfig(metadata ConfigMetadata) error {
	cr.mutex.Lock()
	defer cr.mutex.Unlock()

	// Check if config with same name already exists
	for i, config := range cr.Configs {
		if config.Name == metadata.Name {
			// Update existing config
			metadata.CreatedAt = config.CreatedAt // Preserve creation time
			metadata.UpdatedAt = time.Now()
			cr.Configs[i] = metadata
			return nil
		}
	}

	// Add new config
	metadata.CreatedAt = time.Now()
	metadata.UpdatedAt = time.Now()
	cr.Configs = append(cr.Configs, metadata)
	return nil
}

// RemoveConfig removes a configuration from the registry
func (cr *ConfigRegistry) RemoveConfig(name string) error {
	cr.mutex.Lock()
	defer cr.mutex.Unlock()

	// Don't allow removing active config
	if cr.ActiveConfig == name {
		return fmt.Errorf("cannot remove active configuration")
	}

	for i, config := range cr.Configs {
		if config.Name == name {
			cr.Configs = append(cr.Configs[:i], cr.Configs[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("configuration not found: %s", name)
}

// GetConfig returns a configuration by name
func (cr *ConfigRegistry) GetConfig(name string) (*ConfigMetadata, error) {
	cr.mutex.RLock()
	defer cr.mutex.RUnlock()

	for _, config := range cr.Configs {
		if config.Name == name {
			return &config, nil
		}
	}

	return nil, fmt.Errorf("configuration not found: %s", name)
}

// GetAllConfigs returns all configurations
func (cr *ConfigRegistry) GetAllConfigs() []ConfigMetadata {
	cr.mutex.RLock()
	defer cr.mutex.RUnlock()

	configs := make([]ConfigMetadata, len(cr.Configs))
	copy(configs, cr.Configs)
	return configs
}

// SetActiveConfig sets the active configuration
func (cr *ConfigRegistry) SetActiveConfig(name string) error {
	cr.mutex.Lock()
	defer cr.mutex.Unlock()

	// Verify config exists
	found := false
	for i := range cr.Configs {
		if cr.Configs[i].Name == name {
			cr.Configs[i].IsActive = true
			found = true
		} else {
			cr.Configs[i].IsActive = false
		}
	}

	if !found {
		return fmt.Errorf("configuration not found: %s", name)
	}

	cr.ActiveConfig = name
	return nil
}

// GetActiveConfig returns the active configuration metadata
func (cr *ConfigRegistry) GetActiveConfig() *ConfigMetadata {
	cr.mutex.RLock()
	defer cr.mutex.RUnlock()

	for _, config := range cr.Configs {
		if config.IsActive {
			return &config
		}
	}

	return nil
}

// RenameConfig renames a configuration
func (cr *ConfigRegistry) RenameConfig(oldName, newName string) error {
	cr.mutex.Lock()
	defer cr.mutex.Unlock()

	// Check if new name already exists
	for _, config := range cr.Configs {
		if config.Name == newName {
			return fmt.Errorf("configuration with name '%s' already exists", newName)
		}
	}

	// Find and rename the config
	for i := range cr.Configs {
		if cr.Configs[i].Name == oldName {
			cr.Configs[i].Name = newName
			cr.Configs[i].UpdatedAt = time.Now()

			// Update active config name if necessary
			if cr.ActiveConfig == oldName {
				cr.ActiveConfig = newName
			}

			return nil
		}
	}

	return fmt.Errorf("configuration not found: %s", oldName)
}

// ScanAndInitializeRegistry scans the config directory and initializes the registry
func ScanAndInitializeRegistry(configDir, registryPath, currentConfigPath string) (*ConfigRegistry, error) {
	registry, err := LoadConfigRegistry(registryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load registry: %w", err)
	}

	// Scan config directory for YAML files
	files, err := filepath.Glob(filepath.Join(configDir, "*.yaml"))
	if err != nil {
		return nil, fmt.Errorf("failed to scan config directory: %w", err)
	}

	yamlFiles, err := filepath.Glob(filepath.Join(configDir, "*.yml"))
	if err != nil {
		return nil, fmt.Errorf("failed to scan config directory: %w", err)
	}
	files = append(files, yamlFiles...)

	// Get current config name from path
	currentConfigName := ""
	if currentConfigPath != "" {
		currentConfigName = getConfigNameFromPath(currentConfigPath)
	}

	// Process each config file
	for _, filePath := range files {
		// Skip registry file
		if filepath.Base(filePath) == "registry.yaml" {
			continue
		}

		// Skip test files
		if strings.Contains(filepath.Base(filePath), "test") {
			continue
		}

		// Try to load config to validate it
		_, err := LoadConfig(filePath)
		if err != nil {
			// Skip invalid config files
			continue
		}

		// Extract config name from filename
		configName := getConfigNameFromPath(filePath)
		if configName == "" {
			continue
		}

        // Normalize to absolute path
        if abs, errAbs := filepath.Abs(filePath); errAbs == nil {
            filePath = abs
        }

        // Create metadata
        _, err = os.Stat(filePath)
		if err != nil {
			continue
		}

        metadata := ConfigMetadata{
            Name:        configName,
            FilePath:    filePath,
            Description: fmt.Sprintf("Configuration: %s", configName),
            IsActive:    configName == currentConfigName,
        }

		// Add to registry
		registry.AddConfig(metadata)
	}

	// Set active config
	if currentConfigName != "" {
		registry.SetActiveConfig(currentConfigName)
	}

	// Save updated registry
	if err := registry.Save(registryPath); err != nil {
		return nil, fmt.Errorf("failed to save registry: %w", err)
	}

	return registry, nil
}

// getConfigNameFromPath extracts config name from file path
func getConfigNameFromPath(filePath string) string {
	filename := filepath.Base(filePath)
	ext := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, ext)

	// Remove common prefixes
	if strings.HasPrefix(name, "config_") {
		return strings.TrimPrefix(name, "config_")
	}
	if strings.HasPrefix(name, "config") && name != "config" {
		return strings.TrimPrefix(name, "config")
	}

	return name
}

// ImportConfigFile imports a configuration file with given name
func ImportConfigFile(configDir, configName string, configData []byte, registry *ConfigRegistry) (string, error) {
	// Validate config data
	var testConfig Config
	if err := yaml.Unmarshal(configData, &testConfig); err != nil {
		return "", fmt.Errorf("invalid configuration format: %w", err)
	}

	// Set defaults and validate
	testConfig.setDefaults()
	if err := testConfig.validate(); err != nil {
		return "", fmt.Errorf("invalid configuration: %w", err)
	}

	// Generate file path
    fileName := fmt.Sprintf("config_%s.yaml", configName)
    filePath := filepath.Join(configDir, fileName)
    if abs, errAbs := filepath.Abs(filePath); errAbs == nil {
        filePath = abs
    }

	// Write config file
	if err := os.WriteFile(filePath, configData, 0644); err != nil {
		return "", fmt.Errorf("failed to write config file: %w", err)
	}

	// Add to registry
	metadata := ConfigMetadata{
		Name:        configName,
		FilePath:    filePath,
		Description: fmt.Sprintf("Imported configuration: %s", configName),
		IsActive:    false,
	}

	if err := registry.AddConfig(metadata); err != nil {
		return "", fmt.Errorf("failed to add to registry: %w", err)
	}

	return filePath, nil
}

// SwitchConfig switches to a different configuration file
func (cw *ConfigWatcher) SwitchConfig(configName string) error {
	cw.mutex.Lock()
	defer cw.mutex.Unlock()

	// Get config metadata from registry
	configMeta, err := cw.registry.GetConfig(configName)
	if err != nil {
		return fmt.Errorf("configuration not found: %w", err)
	}

	// Check if file exists
	if _, err := os.Stat(configMeta.FilePath); os.IsNotExist(err) {
		return fmt.Errorf("configuration file not found: %s", configMeta.FilePath)
	}

	// Load new configuration
	newConfig, err := LoadConfig(configMeta.FilePath)
	if err != nil {
		return fmt.Errorf("failed to load new config: %w", err)
	}

	// Stop watching old file
	if err := cw.watcher.Remove(cw.configPath); err != nil {
		cw.logger.Warn("Failed to remove old config from watcher", "error", err)
	}

	// Update config path and config
	oldConfigPath := cw.configPath
	cw.configPath = configMeta.FilePath
	cw.config = newConfig

	// Get new file modification time
	fileInfo, err := os.Stat(configMeta.FilePath)
	if err != nil {
		cw.logger.Warn("Failed to get new config file info", "error", err)
	} else {
		cw.lastModTime = fileInfo.ModTime()
	}

	// Start watching new file
	if err := cw.watcher.Add(configMeta.FilePath); err != nil {
		cw.logger.Error("Failed to watch new config file", "error", err)
		// Try to restore old config
		cw.configPath = oldConfigPath
		cw.watcher.Add(oldConfigPath)
		return fmt.Errorf("failed to watch new config file: %w", err)
	}

	// Update registry active config
	if err := cw.registry.SetActiveConfig(configName); err != nil {
		cw.logger.Warn("Failed to update active config in registry", "error", err)
	}

	// Save registry
	if err := cw.registry.Save(cw.registryPath); err != nil {
		cw.logger.Warn("Failed to save registry", "error", err)
	}

	// Call all registered callbacks with new config
	callbacks := make([]func(*Config), len(cw.callbacks))
	copy(callbacks, cw.callbacks)

	// Release lock before calling callbacks to avoid deadlock
	cw.mutex.Unlock()
	for _, callback := range callbacks {
		callback(newConfig)
	}
	cw.mutex.Lock()

	cw.logger.Info("🔄 配置已切换", "from", oldConfigPath, "to", configMeta.FilePath, "name", configName)

	return nil
}

// GetRegistry returns the configuration registry
func (cw *ConfigWatcher) GetRegistry() *ConfigRegistry {
	cw.mutex.RLock()
	defer cw.mutex.RUnlock()
	return cw.registry
}
