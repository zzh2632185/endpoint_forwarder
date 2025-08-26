package endpoint

import (
	"context"
	"endpoint_forwarder/config"
	"testing"
	"time"
)

func TestFastTester(t *testing.T) {
	// Create test config
	cfg := &config.Config{
		Strategy: config.StrategyConfig{
			Type:             "fastest",
			FastTestEnabled:  true,
			FastTestCacheTTL: 2 * time.Second,
			FastTestTimeout:  1 * time.Second,
			FastTestPath:     "/test",
		},
		Health: config.HealthConfig{
			HealthPath: "/health",
		},
	}

	// Create fast tester
	tester := NewFastTester(cfg)

	// Create test endpoints
	endpoints := []*Endpoint{
		{
			Config: config.EndpointConfig{
				Name: "test1",
				URL:  "https://httpbin.org",
			},
			Status: EndpointStatus{
				Healthy: true,
			},
		},
		{
			Config: config.EndpointConfig{
				Name: "test2", 
				URL:  "https://example.com",
			},
			Status: EndpointStatus{
				Healthy: true,
			},
		},
	}

	// Test parallel testing
	ctx := context.Background()
	results := tester.TestEndpointsParallel(ctx, endpoints)

	if len(results) != len(endpoints) {
		t.Errorf("Expected %d results, got %d", len(endpoints), len(results))
	}

	// Test caching
	results2 := tester.TestEndpointsParallel(ctx, endpoints)
	if len(results2) != len(endpoints) {
		t.Errorf("Expected %d cached results, got %d", len(endpoints), len(results2))
	}
}

func TestFastTesterDisabled(t *testing.T) {
	// Create config with fast testing disabled
	cfg := &config.Config{
		Strategy: config.StrategyConfig{
			Type:            "fastest",
			FastTestEnabled: false,
		},
	}

	// Create fast tester
	tester := NewFastTester(cfg)

	// Create test endpoints
	endpoints := []*Endpoint{
		{
			Config: config.EndpointConfig{
				Name: "test1",
				URL:  "https://httpbin.org",
			},
			Status: EndpointStatus{
				Healthy:      true,
				ResponseTime: 100 * time.Millisecond,
			},
		},
	}

	// Test with disabled fast testing
	ctx := context.Background()
	results := tester.TestEndpointsParallel(ctx, endpoints)

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	if !results[0].Success {
		t.Error("Expected artificial success result when fast testing is disabled")
	}
}