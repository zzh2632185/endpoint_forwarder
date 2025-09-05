package endpoint

import (
	"testing"
	"time"

	"endpoint_forwarder/config"
)

func TestGroupRetryFunctionality(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		Group: config.GroupConfig{
			Cooldown:   60 * time.Second, // 1 minute for testing
			MaxRetries: 2,                // Max 2 retries before cooldown
		},
	}

	// Create group manager
	gm := NewGroupManager(cfg)

	// Create test endpoints
	endpoints := []*Endpoint{
		{
			Config: config.EndpointConfig{
				Name:         "endpoint1",
				Group:        "testgroup",
				GroupPriority: 1,
			},
		},
		{
			Config: config.EndpointConfig{
				Name:         "endpoint2",
				Group:        "testgroup",
				GroupPriority: 1,
			},
		},
	}

	// Update groups
	gm.UpdateGroups(endpoints)

	// Test 1: Initial retry count should be 0
	retryCount := gm.GetGroupRetryCount("testgroup")
	if retryCount != 0 {
		t.Errorf("Expected initial retry count to be 0, got %d", retryCount)
	}

	// Test 2: Increment retry count - first time should not trigger cooldown
	shouldCooldown := gm.IncrementGroupRetry("testgroup")
	if shouldCooldown {
		t.Error("First increment should not trigger cooldown")
	}

	// Check retry count after first increment
	retryCount = gm.GetGroupRetryCount("testgroup")
	if retryCount != 1 {
		t.Errorf("Expected retry count to be 1 after first increment, got %d", retryCount)
	}

	// Test 3: Increment retry count - second time should not trigger cooldown (max is 2)
	shouldCooldown = gm.IncrementGroupRetry("testgroup")
	if shouldCooldown {
		t.Error("Second increment should not trigger cooldown (max is 2)")
	}

	// Check retry count after second increment
	retryCount = gm.GetGroupRetryCount("testgroup")
	if retryCount != 2 {
		t.Errorf("Expected retry count to be 2 after second increment, got %d", retryCount)
	}

	// Test 4: Increment retry count - third time should trigger cooldown
	shouldCooldown = gm.IncrementGroupRetry("testgroup")
	if !shouldCooldown {
		t.Error("Third increment should trigger cooldown")
	}

	// Check retry count after third increment
	retryCount = gm.GetGroupRetryCount("testgroup")
	if retryCount != 3 {
		t.Errorf("Expected retry count to be 3 after third increment, got %d", retryCount)
	}

	// Test 5: Reset retry count
	gm.ResetGroupRetry("testgroup")
	retryCount = gm.GetGroupRetryCount("testgroup")
	if retryCount != 0 {
		t.Errorf("Expected retry count to be 0 after reset, got %d", retryCount)
	}

	// Test 6: Get max retries
	maxRetries := gm.GetGroupMaxRetries("testgroup")
	if maxRetries != 2 {
		t.Errorf("Expected max retries to be 2, got %d", maxRetries)
	}
}

func TestGroupRetryWithNonExistentGroup(t *testing.T) {
	cfg := &config.Config{
		Group: config.GroupConfig{
			MaxRetries: 3,
		},
	}

	gm := NewGroupManager(cfg)

	// Test with non-existent group
	retryCount := gm.GetGroupRetryCount("nonexistent")
	if retryCount != 0 {
		t.Errorf("Expected retry count to be 0 for non-existent group, got %d", retryCount)
	}

	maxRetries := gm.GetGroupMaxRetries("nonexistent")
	if maxRetries != 3 {
		t.Errorf("Expected max retries to be 3 for non-existent group, got %d", maxRetries)
	}

	shouldCooldown := gm.IncrementGroupRetry("nonexistent")
	if shouldCooldown {
		t.Error("Increment should not trigger cooldown for non-existent group")
	}
}