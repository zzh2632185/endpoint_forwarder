package endpoint

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"endpoint_forwarder/config"
)

func TestHealthCheckWithAPIEndpoint(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
		expectHealthy bool
	}{
		{"Success 200", 200, true},
		{"Success 201", 201, true},
		{"Bad Request 400", 400, true},  // API reachable but invalid request
		{"Unauthorized 401", 401, true}, // API reachable but needs auth
		{"Forbidden 403", 403, true},    // API reachable but forbidden
		{"Not Found 404", 404, true},    // API reachable but endpoint not found
		{"Server Error 500", 500, false}, // API has issues
		{"Bad Gateway 502", 502, false},  // API unreachable
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a test server that returns the specified status code
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Add small delay to ensure response time is measurable
				time.Sleep(1 * time.Millisecond)
				
				// Verify it's checking the correct path
				if r.URL.Path != "/v1/models" {
					t.Errorf("Expected request to /v1/models, got %s", r.URL.Path)
				}
				// Verify Authorization header is present
				if r.Header.Get("Authorization") != "Bearer test-token" {
					t.Errorf("Expected Authorization header 'Bearer test-token', got '%s'", r.Header.Get("Authorization"))
				}
				w.WriteHeader(tc.statusCode)
			}))
			defer server.Close()

			// Create config with test server URL
			cfg := &config.Config{
				Health: config.HealthConfig{
					CheckInterval: 30 * time.Second,
					Timeout:       5 * time.Second,
					HealthPath:    "/v1/models",
				},
				Endpoints: []config.EndpointConfig{
					{
						Name:    "test-endpoint",
						URL:     server.URL,
						Token:   "test-token",
						Timeout: 30 * time.Second,
					},
				},
			}

			// Create manager and perform single health check
			manager := NewManager(cfg)
			endpoint := manager.GetAllEndpoints()[0]

			// Perform health check twice for endpoints that should be unhealthy
			// (due to 2-failure threshold)
			manager.checkEndpointHealth(endpoint)
			if !tc.expectHealthy {
				manager.checkEndpointHealth(endpoint) // Second check to trigger unhealthy status
			}

			// Check result
			if endpoint.IsHealthy() != tc.expectHealthy {
				t.Errorf("Expected healthy=%v for status %d, got %v", 
					tc.expectHealthy, tc.statusCode, endpoint.IsHealthy())
			}

			// Verify response time is recorded (should be > 0 for all HTTP responses)
			responseTime := endpoint.GetResponseTime()
			if responseTime <= 0 {
				t.Errorf("Expected response time to be recorded for status %d, got %v", tc.statusCode, responseTime)
			}
		})
	}
}

func TestFastestStrategyLogging(t *testing.T) {
	// Create multiple test servers with different response times
	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond) // Simulate slow response
		w.WriteHeader(200)
	}))
	defer slowServer.Close()

	fastServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond) // Simulate fast response
		w.WriteHeader(200)
	}))
	defer fastServer.Close()

	cfg := &config.Config{
		Strategy: config.StrategyConfig{
			Type: "fastest",
		},
		Health: config.HealthConfig{
			CheckInterval: 30 * time.Second,
			Timeout:       5 * time.Second,
			HealthPath:    "/v1/models",
		},
		Endpoints: []config.EndpointConfig{
			{
				Name:    "slow-endpoint",
				URL:     slowServer.URL,
				Priority: 1,
				Timeout: 30 * time.Second,
			},
			{
				Name:    "fast-endpoint", 
				URL:     fastServer.URL,
				Priority: 2,
				Timeout: 30 * time.Second,
			},
		},
	}

	manager := NewManager(cfg)
	
	// Perform health checks to populate response times
	for _, endpoint := range manager.GetAllEndpoints() {
		manager.checkEndpointHealth(endpoint)
	}

	// Get healthy endpoints (this should trigger logging for fastest strategy)
	healthy := manager.GetHealthyEndpoints()
	
	if len(healthy) != 2 {
		t.Errorf("Expected 2 healthy endpoints, got %d", len(healthy))
	}

	// Verify the fast endpoint comes first
	if healthy[0].Config.Name != "fast-endpoint" {
		t.Errorf("Expected fast-endpoint to be first in fastest strategy, got %s", healthy[0].Config.Name)
	}

	// Verify response times are different
	fastTime := healthy[0].GetResponseTime()
	slowTime := healthy[1].GetResponseTime()
	
	if fastTime >= slowTime {
		t.Errorf("Expected fast endpoint to have lower response time. Fast: %v, Slow: %v", fastTime, slowTime)
	}
}