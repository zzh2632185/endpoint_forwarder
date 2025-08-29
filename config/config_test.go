package config

import (
	"log/slog"
	"os"
	"testing"
	"time"
)

func TestFullParameterInheritance(t *testing.T) {
	configContent := `
server:
  host: "localhost"
  port: 8080

strategy:
  type: "priority"

endpoints:
  - name: "primary"
    url: "https://api1.example.com"
    priority: 1
    timeout: "45s"
    token: "primary-token"
    headers:
      X-API-Version: "v1"
      User-Agent: "Claude-Forwarder/1.0"
      Authorization-Fallback: "Bearer fallback"

  - name: "inherits-all"
    url: "https://api2.example.com"
    priority: 2
    # Should inherit timeout, token, and all headers

  - name: "partial-override" 
    url: "https://api3.example.com"
    priority: 3
    timeout: "60s"
    headers:
      X-API-Version: "v2"
      X-Custom: "custom-value"

  - name: "full-custom"
    url: "https://api4.example.com"
    priority: 4
    timeout: "30s"
    token: "custom-token"
    headers:
      X-Different: "different-value"
`

	tmpFile, err := os.CreateTemp("", "test-config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	tmpFile.Close()

	config, err := LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Test full inheritance endpoint
	inheritsAll := config.Endpoints[1]
	if inheritsAll.Timeout != 45*time.Second {
		t.Errorf("Expected inherited timeout 45s, got %v", inheritsAll.Timeout)
	}
	if inheritsAll.Token != "primary-token" {
		t.Errorf("Expected inherited token 'primary-token', got '%s'", inheritsAll.Token)
	}
	if len(inheritsAll.Headers) != 3 {
		t.Errorf("Expected 3 inherited headers, got %d", len(inheritsAll.Headers))
	}
	if inheritsAll.Headers["X-API-Version"] != "v1" {
		t.Errorf("Expected inherited header X-API-Version=v1, got '%s'", inheritsAll.Headers["X-API-Version"])
	}

	// Test partial override endpoint
	partialOverride := config.Endpoints[2]
	if partialOverride.Timeout != 60*time.Second {
		t.Errorf("Expected override timeout 60s, got %v", partialOverride.Timeout)
	}
	if partialOverride.Token != "primary-token" {
		t.Errorf("Expected inherited token 'primary-token', got '%s'", partialOverride.Token)
	}
	if partialOverride.Headers["X-API-Version"] != "v2" {
		t.Errorf("Expected override header X-API-Version=v2, got '%s'", partialOverride.Headers["X-API-Version"])
	}
	if partialOverride.Headers["User-Agent"] != "Claude-Forwarder/1.0" {
		t.Errorf("Expected inherited User-Agent, got '%s'", partialOverride.Headers["User-Agent"])
	}
	if partialOverride.Headers["X-Custom"] != "custom-value" {
		t.Errorf("Expected new header X-Custom=custom-value, got '%s'", partialOverride.Headers["X-Custom"])
	}

	// Test full custom with merging
	fullCustom := config.Endpoints[3]
	if fullCustom.Timeout != 30*time.Second {
		t.Errorf("Expected custom timeout 30s, got %v", fullCustom.Timeout)
	}
	if fullCustom.Token != "custom-token" {
		t.Errorf("Expected custom token 'custom-token', got '%s'", fullCustom.Token)
	}
	if fullCustom.Headers["X-Different"] != "different-value" {
		t.Errorf("Expected custom header X-Different=different-value, got '%s'", fullCustom.Headers["X-Different"])
	}
	if fullCustom.Headers["User-Agent"] != "Claude-Forwarder/1.0" {
		t.Errorf("Expected inherited User-Agent, got '%s'", fullCustom.Headers["User-Agent"])
	}
}

func TestNoTokenInheritance(t *testing.T) {
	// Test case where first endpoint has no token
	configContent := `
server:
  host: "localhost"
  port: 8080

strategy:
  type: "priority"

endpoints:
  - name: "first"
    url: "https://api1.example.com"
    priority: 1
    # No token specified

  - name: "second"
    url: "https://api2.example.com"
    priority: 2
    # No token specified

  - name: "third"
    url: "https://api3.example.com"
    priority: 3
    token: "third-token"
`

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "test-config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	tmpFile.Close()

	// Load configuration
	config, err := LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify no token inheritance when first endpoint has no token
	if config.Endpoints[0].Token != "" {
		t.Errorf("First endpoint token: expected empty, got '%s'", config.Endpoints[0].Token)
	}

	if config.Endpoints[1].Token != "" {
		t.Errorf("Second endpoint token: expected empty, got '%s'", config.Endpoints[1].Token)
	}

	if config.Endpoints[2].Token != "third-token" {
		t.Errorf("Third endpoint token: expected 'third-token', got '%s'", config.Endpoints[2].Token)
	}
}

func TestApplyPrimaryEndpoint(t *testing.T) {
	tests := []struct {
		name             string
		config           *Config
		primaryEndpoint  string
		expectError      bool
		expectedPriority int
		expectedAdjusted int
	}{
		{
			name: "Valid endpoint name",
			config: &Config{
				Endpoints: []EndpointConfig{
					{Name: "endpoint1", Priority: 5},
					{Name: "endpoint2", Priority: 1},
					{Name: "endpoint3", Priority: 3},
				},
			},
			primaryEndpoint:  "endpoint3",
			expectError:      false,
			expectedPriority: 1,
			expectedAdjusted: 1, // only endpoint2 has priority <= 1
		},
		{
			name: "Invalid endpoint name",
			config: &Config{
				Endpoints: []EndpointConfig{
					{Name: "endpoint1", Priority: 1},
					{Name: "endpoint2", Priority: 2},
				},
			},
			primaryEndpoint: "nonexistent",
			expectError:     true,
		},
		{
			name: "Empty primary endpoint",
			config: &Config{
				Endpoints: []EndpointConfig{
					{Name: "endpoint1", Priority: 1},
				},
			},
			primaryEndpoint: "",
			expectError:     false,
		},
		{
			name: "Multiple endpoints need adjustment",
			config: &Config{
				Endpoints: []EndpointConfig{
					{Name: "endpoint1", Priority: 1},
					{Name: "endpoint2", Priority: 0},
					{Name: "endpoint3", Priority: -1},
					{Name: "endpoint4", Priority: 5},
				},
			},
			primaryEndpoint:  "endpoint4",
			expectError:      false,
			expectedPriority: 1,
			expectedAdjusted: 3, // endpoints 1, 2, 3 all have priority <= 1
		},
		{
			name: "No adjustments needed",
			config: &Config{
				Endpoints: []EndpointConfig{
					{Name: "endpoint1", Priority: 5},
					{Name: "endpoint2", Priority: 3},
					{Name: "endpoint3", Priority: 2},
				},
			},
			primaryEndpoint:  "endpoint2",
			expectError:      false,
			expectedPriority: 1,
			expectedAdjusted: 0, // no endpoints have priority <= 1 except the primary
		},
	}

	// Create a null logger for testing
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up the primary endpoint
			tt.config.PrimaryEndpoint = tt.primaryEndpoint

			// Apply the primary endpoint
			err := tt.config.ApplyPrimaryEndpoint(logger)

			// Check error expectation
			if tt.expectError && err == nil {
				t.Errorf("Expected an error, but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error, but got: %v", err)
			}

			// If no error expected, check the results
			if !tt.expectError && tt.primaryEndpoint != "" {
				// Find the primary endpoint and check its priority
				primaryIndex := tt.config.findEndpointIndex(tt.primaryEndpoint)
				if primaryIndex != -1 {
					if tt.config.Endpoints[primaryIndex].Priority != tt.expectedPriority {
						t.Errorf("Expected primary endpoint priority %d, got %d",
							tt.expectedPriority, tt.config.Endpoints[primaryIndex].Priority)
					}
				}

				// Count endpoints that were adjusted (now have priority > original priority)
				adjustedCount := 0
				for i := range tt.config.Endpoints {
					if i != primaryIndex {
						// Check if this endpoint was adjusted (has priority > 1)
						switch tt.name {
						case "Valid endpoint name":
							if i == 1 && tt.config.Endpoints[i].Priority == 3 { // endpoint2: 1 + 2 = 3
								adjustedCount++
							}
						case "Multiple endpoints need adjustment":
							if (i == 0 && tt.config.Endpoints[i].Priority == 3) || // endpoint1: 1 + 2 = 3
								(i == 1 && tt.config.Endpoints[i].Priority == 2) || // endpoint2: 0 + 2 = 2
								(i == 2 && tt.config.Endpoints[i].Priority == 1) { // endpoint3: -1 + 2 = 1
								adjustedCount++
							}
						}
					}
				}

				if adjustedCount != tt.expectedAdjusted {
					t.Errorf("Expected %d endpoints to be adjusted, got %d",
						tt.expectedAdjusted, adjustedCount)
				}
			}
		})
	}
}

func TestFindEndpointIndex(t *testing.T) {
	config := &Config{
		Endpoints: []EndpointConfig{
			{Name: "endpoint1"},
			{Name: "endpoint2"},
			{Name: "endpoint3"},
		},
	}

	tests := []struct {
		name     string
		endpoint string
		expected int
	}{
		{"First endpoint", "endpoint1", 0},
		{"Middle endpoint", "endpoint2", 1},
		{"Last endpoint", "endpoint3", 2},
		{"Non-existent endpoint", "nonexistent", -1},
		{"Empty string", "", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.findEndpointIndex(tt.endpoint)
			if result != tt.expected {
				t.Errorf("Expected index %d, got %d", tt.expected, result)
			}
		})
	}
}