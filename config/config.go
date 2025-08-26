package config

import (
	"fmt"
	"os"
	"time"

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
	GlobalTimeout time.Duration   `yaml:"global_timeout"` // Global timeout for non-streaming requests
	Endpoints    []EndpointConfig `yaml:"endpoints"`
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