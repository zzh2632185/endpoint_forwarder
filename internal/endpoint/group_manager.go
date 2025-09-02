package endpoint

import (
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"

	"endpoint_forwarder/config"
)

// GroupInfo represents information about an endpoint group
type GroupInfo struct {
	Name         string
	Priority     int
	IsActive     bool
	CooldownUntil time.Time
	Endpoints    []*Endpoint
}

// GroupManager manages endpoint groups and their cooldown states
type GroupManager struct {
	groups        map[string]*GroupInfo
	config        *config.Config
	mutex         sync.RWMutex
	cooldownDuration time.Duration
}

// NewGroupManager creates a new group manager
func NewGroupManager(cfg *config.Config) *GroupManager {
	return &GroupManager{
		groups:        make(map[string]*GroupInfo),
		config:        cfg,
		cooldownDuration: cfg.Group.Cooldown,
	}
}

// UpdateConfig updates the group manager configuration
func (gm *GroupManager) UpdateConfig(cfg *config.Config) {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()
	
	gm.config = cfg
	gm.cooldownDuration = cfg.Group.Cooldown
}

// UpdateGroups rebuilds group information from endpoints
func (gm *GroupManager) UpdateGroups(endpoints []*Endpoint) {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()
	
	// Clear existing groups but preserve cooldown states
	oldGroups := make(map[string]*GroupInfo)
	for name, group := range gm.groups {
		if !group.CooldownUntil.IsZero() && time.Now().Before(group.CooldownUntil) {
			// Preserve cooldown state
			oldGroups[name] = &GroupInfo{
				Name:         group.Name,
				Priority:     group.Priority,
				IsActive:     false,
				CooldownUntil: group.CooldownUntil,
				Endpoints:    nil, // Will be updated
			}
		}
	}
	
	// Rebuild groups from current endpoints
	newGroups := make(map[string]*GroupInfo)
	
	for _, ep := range endpoints {
		groupName := ep.Config.Group
		if groupName == "" {
			groupName = "Default"
		}
		
		if _, exists := newGroups[groupName]; !exists {
			// Check if this group was in cooldown
			var cooldownUntil time.Time
			if oldGroup, hadCooldown := oldGroups[groupName]; hadCooldown {
				cooldownUntil = oldGroup.CooldownUntil
			}
			
			newGroups[groupName] = &GroupInfo{
				Name:         groupName,
				Priority:     ep.Config.GroupPriority,
				IsActive:     cooldownUntil.IsZero() || time.Now().After(cooldownUntil),
				CooldownUntil: cooldownUntil,
				Endpoints:    make([]*Endpoint, 0),
			}
		}
		
		newGroups[groupName].Endpoints = append(newGroups[groupName].Endpoints, ep)
	}
	
	gm.groups = newGroups
	
	// Update active status based on cooldown timers
	gm.updateActiveGroups()
}

// updateActiveGroups updates which groups are currently active
func (gm *GroupManager) updateActiveGroups() {
	now := time.Now()
	
	// First, check cooldown timers and update active status
	for _, group := range gm.groups {
		if !group.CooldownUntil.IsZero() && now.After(group.CooldownUntil) {
			// Cooldown expired, group can be active again
			group.IsActive = true
			group.CooldownUntil = time.Time{}
			slog.Info(fmt.Sprintf("🔄 [组管理] 组冷却结束，重新激活: %s (优先级: %d)", 
				group.Name, group.Priority))
		} else if !group.CooldownUntil.IsZero() && now.Before(group.CooldownUntil) {
			// Still in cooldown
			group.IsActive = false
		}
	}
	
	// Determine which groups should be active based on priority
	// Get all groups sorted by priority
	sortedGroups := gm.getSortedGroups()
	
	// Find the highest priority group that's not in cooldown
	activeGroupFound := false
	for _, group := range sortedGroups {
		if group.CooldownUntil.IsZero() || now.After(group.CooldownUntil) {
			if !activeGroupFound {
				group.IsActive = true
				activeGroupFound = true
			} else {
				group.IsActive = false // Only one group can be active at a time
			}
		}
	}
}

// getSortedGroups returns groups sorted by priority (lower number = higher priority)
func (gm *GroupManager) getSortedGroups() []*GroupInfo {
	groups := make([]*GroupInfo, 0, len(gm.groups))
	for _, group := range gm.groups {
		groups = append(groups, group)
	}
	
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Priority < groups[j].Priority
	})
	
	return groups
}

// GetActiveGroups returns currently active groups
func (gm *GroupManager) GetActiveGroups() []*GroupInfo {
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()
	
	gm.updateActiveGroups()
	
	var active []*GroupInfo
	for _, group := range gm.groups {
		if group.IsActive {
			active = append(active, group)
		}
	}
	
	// Sort by priority
	sort.Slice(active, func(i, j int) bool {
		return active[i].Priority < active[j].Priority
	})
	
	return active
}

// GetAllGroups returns all groups
func (gm *GroupManager) GetAllGroups() []*GroupInfo {
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()
	
	gm.updateActiveGroups()
	
	groups := make([]*GroupInfo, 0, len(gm.groups))
	for _, group := range gm.groups {
		groups = append(groups, group)
	}
	
	// Sort by priority
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Priority < groups[j].Priority
	})
	
	return groups
}

// SetGroupCooldown sets a group into cooldown mode
func (gm *GroupManager) SetGroupCooldown(groupName string) {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()
	
	if group, exists := gm.groups[groupName]; exists {
		now := time.Now()
		group.CooldownUntil = now.Add(gm.cooldownDuration)
		group.IsActive = false
		
		slog.Warn(fmt.Sprintf("❄️ [组管理] 组进入冷却状态: %s (冷却时长: %v, 恢复时间: %s)", 
			groupName, gm.cooldownDuration, group.CooldownUntil.Format("15:04:05")))
		
		// Update active groups after cooldown change
		gm.updateActiveGroups()
		
		// Log next active group if any
		for _, g := range gm.getSortedGroups() {
			if g.IsActive {
				slog.Info(fmt.Sprintf("🔄 [组管理] 切换到下一优先级组: %s (优先级: %d)", 
					g.Name, g.Priority))
				break
			}
		}
	}
}

// IsGroupInCooldown checks if a group is currently in cooldown
func (gm *GroupManager) IsGroupInCooldown(groupName string) bool {
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()
	
	if group, exists := gm.groups[groupName]; exists {
		return !group.CooldownUntil.IsZero() && time.Now().Before(group.CooldownUntil)
	}
	
	return false
}

// GetGroupCooldownRemaining returns remaining cooldown time for a group
func (gm *GroupManager) GetGroupCooldownRemaining(groupName string) time.Duration {
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()
	
	if group, exists := gm.groups[groupName]; exists {
		if !group.CooldownUntil.IsZero() && time.Now().Before(group.CooldownUntil) {
			return group.CooldownUntil.Sub(time.Now())
		}
	}
	
	return 0
}

// FilterEndpointsByActiveGroups filters endpoints to only include those in active groups
func (gm *GroupManager) FilterEndpointsByActiveGroups(endpoints []*Endpoint) []*Endpoint {
	activeGroups := gm.GetActiveGroups()
	if len(activeGroups) == 0 {
		return nil
	}
	
	// Create a map of active group names for quick lookup
	activeGroupNames := make(map[string]bool)
	for _, group := range activeGroups {
		activeGroupNames[group.Name] = true
	}
	
	// Filter endpoints
	var filtered []*Endpoint
	for _, ep := range endpoints {
		groupName := ep.Config.Group
		if groupName == "" {
			groupName = "Default"
		}
		
		if activeGroupNames[groupName] {
			filtered = append(filtered, ep)
		}
	}
	
	return filtered
}