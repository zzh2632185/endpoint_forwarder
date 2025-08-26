package config

import (
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