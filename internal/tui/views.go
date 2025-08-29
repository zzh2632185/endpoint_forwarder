package tui

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rivo/tview"
	
	"endpoint_forwarder/config"
	"endpoint_forwarder/internal/endpoint"
	"endpoint_forwarder/internal/middleware"
	"endpoint_forwarder/internal/monitor"
)

// OverviewView represents the overview tab
type OverviewView struct {
	container           *tview.Flex
	metricsBox          *tview.TextView
	chartBox            *tview.TextView
	endpointsBox        *tview.TextView
	systemBox           *tview.TextView
	monitoringMiddleware *middleware.MonitoringMiddleware
	endpointManager     *endpoint.Manager
	responseTimeHistory []time.Duration
	lastMetricsHash     string // Track metrics content changes
	lastEndpointsHash   string // Track endpoints content changes  
	lastSystemHash      string // Track system content changes
	startTime           time.Time // App start time for uptime calculation
}

// NewOverviewView creates a new overview view
func NewOverviewView(monitoringMiddleware *middleware.MonitoringMiddleware, endpointManager *endpoint.Manager, startTime time.Time) *OverviewView {
	view := &OverviewView{
		monitoringMiddleware: monitoringMiddleware,
		endpointManager:     endpointManager,
		responseTimeHistory: make([]time.Duration, 0, 60),
		startTime:           startTime,
	}
	view.setupUI()
	return view
}

func (v *OverviewView) setupUI() {
	v.metricsBox = tview.NewTextView().SetDynamicColors(true).SetScrollable(false)
	v.metricsBox.SetBorder(true).SetTitle(" 📊 Request Metrics ").SetTitleAlign(tview.AlignLeft)

	v.chartBox = tview.NewTextView().SetDynamicColors(true).SetScrollable(false)
	v.chartBox.SetBorder(true).SetTitle(" 🪙 Historical Token Usage ").SetTitleAlign(tview.AlignLeft)

	v.endpointsBox = tview.NewTextView().SetDynamicColors(true).SetScrollable(false)
	v.endpointsBox.SetBorder(true).SetTitle(" 🎯 Endpoints Status ").SetTitleAlign(tview.AlignLeft)

	v.systemBox = tview.NewTextView().SetDynamicColors(true).SetScrollable(false)
	v.systemBox.SetBorder(true).SetTitle(" 💻 System Info ").SetTitleAlign(tview.AlignLeft)

	topFlex := tview.NewFlex().
		AddItem(v.metricsBox, 0, 1, false).
		AddItem(v.chartBox, 0, 1, false)

	bottomFlex := tview.NewFlex().
		AddItem(v.endpointsBox, 0, 1, false).
		AddItem(v.systemBox, 0, 1, false)

	v.container = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(topFlex, 12, 0, false).   // Increased height for top section (Request Metrics + Historical Token Usage)  
		AddItem(bottomFlex, 0, 1, false)  // Remaining space for bottom (Endpoints Status + System Info)
}

func (v *OverviewView) GetPrimitive() tview.Primitive {
	return v.container
}

func (v *OverviewView) Update() {
	metrics := v.monitoringMiddleware.GetMetrics().GetMetrics()
	
	// Update metrics with content change detection
	avgTime := formatDurationShort(metrics.GetAverageResponseTime())
	successRate := metrics.GetSuccessRate()
	
	// Get token usage statistics with safety checks
	tokenStats := metrics.GetTotalTokenStats()
	totalTokens := tokenStats.InputTokens + tokenStats.OutputTokens
	
	// Ensure non-negative values
	if tokenStats.InputTokens < 0 { tokenStats.InputTokens = 0 }
	if tokenStats.OutputTokens < 0 { tokenStats.OutputTokens = 0 }
	if tokenStats.CacheCreationTokens < 0 { tokenStats.CacheCreationTokens = 0 }
	if tokenStats.CacheReadTokens < 0 { tokenStats.CacheReadTokens = 0 }
	totalTokens = tokenStats.InputTokens + tokenStats.OutputTokens
	
	metricsText := fmt.Sprintf(`[white::b]Total Requests:[white::-] [cyan]%8d[white]
[white::b]Successful:[white::-] [green]%8d[white] ([green]%5.1f%%[white])
[white::b]Failed:[white::-] [red]%8d[white] ([red]%5.1f%%[white])
[white::b]Avg Response Time:[white::-] [cyan]%8s[white]

[yellow::b]🪙 Token Usage[white::-]
[white::b]📥 Input Tokens:[white::-] [cyan]%8d[white]
[white::b]📤 Output Tokens:[white::-] [cyan]%8d[white]
[white::b]🆕 Cache Creation:[white::-] [cyan]%8d[white]
[white::b]📖 Cache Read:[white::-] [cyan]%8d[white]
[white::b]🔢 Total Tokens:[white::-] [magenta]%8d[white]`,
		metrics.TotalRequests,
		metrics.SuccessfulRequests, successRate,
		metrics.FailedRequests, 100-successRate,
		avgTime,
		tokenStats.InputTokens,
		tokenStats.OutputTokens,
		tokenStats.CacheCreationTokens,
		tokenStats.CacheReadTokens,
		totalTokens)

	// Only update metrics if content changed
	if metricsText != v.lastMetricsHash {
		v.lastMetricsHash = metricsText
		v.metricsBox.SetText(metricsText)
	}
	
	// Historical token usage from past connections
	connectionHistory := metrics.ConnectionHistory
	
	// Show token usage for the last 3 connections that have token data
	var chartText strings.Builder
	chartText.WriteString("[yellow::b]🪙 Historical Token Usage[white::-]\n")
	chartText.WriteString("[gray]Past connections with token consumption:[white]\n\n")
	
	// Filter connections that have token usage and get the most recent 3
	var connectionsWithTokens []*monitor.ConnectionInfo
	for i := len(connectionHistory) - 1; i >= 0 && len(connectionsWithTokens) < 3; i-- {
		conn := connectionHistory[i]
		totalTokens := conn.TokenUsage.InputTokens + conn.TokenUsage.OutputTokens + 
					   conn.TokenUsage.CacheCreationTokens + conn.TokenUsage.CacheReadTokens
		if totalTokens > 0 {
			connectionsWithTokens = append(connectionsWithTokens, conn)
		}
	}
	
	if len(connectionsWithTokens) > 0 {
		for i, conn := range connectionsWithTokens {
			totalTokens := conn.TokenUsage.InputTokens + conn.TokenUsage.OutputTokens
			totalCacheTokens := conn.TokenUsage.CacheCreationTokens + conn.TokenUsage.CacheReadTokens
			
			// Format connection info
			clientIP := truncateString(conn.ClientIP, 12)
			endpoint := truncateString(conn.Endpoint, 10)
			if endpoint == "" || endpoint == "unknown" {
				endpoint = "pending"
			}
			
			// Status color
			statusColor := "green"
			statusText := "✓"
			if conn.Status == "failed" {
				statusColor = "red" 
				statusText = "✗"
			}
			
			chartText.WriteString(fmt.Sprintf("%d. [%s]%s[white] [cyan]%-12s[white] -> [yellow]%-10s[white]\n",
				i+1, statusColor, statusText, clientIP, endpoint))
			chartText.WriteString(fmt.Sprintf("   📥[cyan]%4d[white] 📤[cyan]%4d[white] 🆕[cyan]%3d[white] 📖[cyan]%3d[white] 🔢[magenta]%5d[white]\n\n",
				conn.TokenUsage.InputTokens, conn.TokenUsage.OutputTokens,
				conn.TokenUsage.CacheCreationTokens, conn.TokenUsage.CacheReadTokens,
				totalTokens + totalCacheTokens))
		}

		// Fill remaining lines if fewer than 3 connections
		for i := len(connectionsWithTokens); i < 3; i++ {
			chartText.WriteString(fmt.Sprintf("%d. [gray]─[white]\n\n", i+1))
		}
	} else {
		chartText.WriteString("[gray]No connections with token usage yet...\n")
		chartText.WriteString("Token consumption history will appear here\n")
		chartText.WriteString("after processing Claude API requests.\n\n")
		for i := 0; i < 3; i++ {
			chartText.WriteString(fmt.Sprintf("%d. [gray]─[white]\n\n", i+1))
		}
	}
	
	v.chartBox.SetText(chartText.String())
	
	// Endpoints status - maintain consistent formatting
	endpoints := v.endpointManager.GetAllEndpoints()
	var statusText strings.Builder
	
	healthyCount := 0
	for _, ep := range endpoints {
		if ep.IsHealthy() {
			healthyCount++
		}
	}
	
	statusText.WriteString(fmt.Sprintf("[white::b]Total:[white::-] [cyan]%3d[white] | [white::b]Healthy:[white::-] [green]%3d[white]\n\n", len(endpoints), healthyCount))
	
	// Always show exactly 6 lines to maintain consistent height
	for i := 0; i < 6; i++ {
		if i < len(endpoints) {
			ep := endpoints[i]
			status := ep.GetStatus()
			healthIcon := "[red]●[white]"
			if status.Healthy {
				healthIcon = "[green]●[white]"
			}
			
			// Fixed width formatting to prevent jumping
			statusText.WriteString(fmt.Sprintf("%s [cyan]%-12s[white] (%3dms)\n",
				healthIcon,
				truncateString(ep.Config.Name, 12),
				status.ResponseTime.Milliseconds()))
		} else {
			// Fill empty lines to maintain height
			statusText.WriteString("\n")
		}
	}
	
	if len(endpoints) > 6 {
		statusText.WriteString("[gray]... and more[white]")
	}
	
	// Only update endpoints if content changed
	endpointsContent := statusText.String()
	if endpointsContent != v.lastEndpointsHash {
		v.lastEndpointsHash = endpointsContent
		v.endpointsBox.SetText(endpointsContent)
	}
	
	// System info - fixed width formatting
	uptime := time.Since(v.startTime)
	systemText := fmt.Sprintf(`[white::b]Active Connections:[white::-] [cyan]%6d[white]
[white::b]Total Connections:[white::-] [cyan]%7d[white]
[white::b]Uptime:[white::-] [cyan]%8s[white]`,
		len(metrics.ActiveConnections),
		len(metrics.ActiveConnections)+len(metrics.ConnectionHistory),
		formatUptimeShort(uptime))

	// Only update system info if content changed
	if systemText != v.lastSystemHash {
		v.lastSystemHash = systemText
		v.systemBox.SetText(systemText)
	}
}

// EndpointsView represents the endpoints tab
type EndpointsView struct {
	container           *tview.Flex
	table               *tview.Table
	detailBox           *tview.TextView
	monitoringMiddleware *middleware.MonitoringMiddleware
	endpointManager     *endpoint.Manager
	tuiApp              *TUIApp  // Reference to main TUI app for edit mode
	selectedRow         int
	lastDetailHash      string // Track detail content changes
}

func NewEndpointsView(monitoringMiddleware *middleware.MonitoringMiddleware, endpointManager *endpoint.Manager) *EndpointsView {
	view := &EndpointsView{
		monitoringMiddleware: monitoringMiddleware,
		endpointManager:     endpointManager,
	}
	view.setupUI()
	return view
}

func (v *EndpointsView) setupUI() {
	v.table = tview.NewTable().SetBorders(true).SetSelectable(true, false)
	
	// Update table title based on edit mode
	v.updateTableTitle()
	
	// Set up table selection change handler (auto-update on row change)
	v.table.SetSelectionChangedFunc(func(row, column int) {
		if row > 0 { // Skip header row
			v.selectedRow = row
			v.updateDetails()
		}
	})
	
	v.detailBox = tview.NewTextView().SetDynamicColors(true).SetScrollable(true)
	v.detailBox.SetBorder(true).SetTitle(" 📊 Details ").SetTitleAlign(tview.AlignLeft)

	v.container = tview.NewFlex().
		AddItem(v.table, 0, 2, true).
		AddItem(v.detailBox, 0, 1, false)

	// Setup headers
	headers := []string{"Status", "Name", "URL", "Priority", "Response Time", "Requests"}
	for col, header := range headers {
		v.table.SetCell(0, col, tview.NewTableCell(header).SetTextColor(tview.Styles.SecondaryTextColor).SetSelectable(false))
	}
}

// updateTableTitle updates the table title based on current mode
func (v *EndpointsView) updateTableTitle() {
	var title string
	if v.tuiApp != nil && v.tuiApp.IsInEditMode() {
		isDirty := ""
		if v.tuiApp.HasUnsavedChanges() {
			isDirty = " *"
		}
		title = fmt.Sprintf(" 🎯 Endpoints [编辑模式%s - ESC退出 Ctrl+S保存] ", isDirty)
	} else {
		title = " 🎯 Endpoints [Enter编辑 数字键选择优先级] "
	}
	v.table.SetBorder(true).SetTitle(title).SetTitleAlign(tview.AlignLeft)
}

func (v *EndpointsView) GetPrimitive() tview.Primitive {
	return v.container
}

// SetTUIApp sets the TUIApp reference for edit mode functionality
func (v *EndpointsView) SetTUIApp(app *TUIApp) {
	v.tuiApp = app
}

func (v *EndpointsView) Update() {
	// Update table title first
	v.updateTableTitle()
	
	v.updateTable()
	// Update details for currently selected row
	if v.selectedRow > 0 {
		v.updateDetails()
	} else {
		v.detailBox.SetText("[gray]Select an endpoint to view details[white]\n\n[yellow]Use arrow keys to navigate[white]")
	}
}

// updateTable updates the endpoints table efficiently
func (v *EndpointsView) updateTable() {
	endpoints := v.endpointManager.GetAllEndpoints()
	metrics := v.monitoringMiddleware.GetMetrics().GetMetrics()
	
	// Get current number of data rows (excluding header)
	currentRows := v.table.GetRowCount() - 1
	
	// Adjust table size only when necessary
	if currentRows < len(endpoints) {
		// Add missing rows
		for i := currentRows; i < len(endpoints); i++ {
			row := i + 1
			for col := 0; col < 6; col++ {
				v.table.SetCell(row, col, tview.NewTableCell(""))
			}
		}
	} else if currentRows > len(endpoints) {
		// Remove excess rows from the end
		for row := v.table.GetRowCount() - 1; row > len(endpoints); row-- {
			v.table.RemoveRow(row)
		}
	}
	
	// Update endpoint data without recreating cells
	for i, ep := range endpoints {
		row := i + 1 // Skip header row
		status := ep.GetStatus()
		
		statusIcon := "🔴"
		if status.Healthy {
			statusIcon = "🟢"
		}
		
		endpointStats := metrics.EndpointStats[ep.Config.Name]
		totalReqs := int64(0)
		if endpointStats != nil {
			totalReqs = endpointStats.TotalRequests
		}
		
		// Get effective priority (temp or config)
		effectivePriority := ep.Config.Priority
		if v.tuiApp != nil {
			effectivePriority = v.tuiApp.GetEffectivePriority(ep.Config.Name)
		}
		
		// Find if this is the highest priority endpoint
		isHighestPriority := false
		if v.tuiApp != nil {
			minPriority := 999
			for _, endpoint := range endpoints {
				priority := v.tuiApp.GetEffectivePriority(endpoint.Config.Name)
				if priority < minPriority {
					minPriority = priority
				}
			}
			isHighestPriority = effectivePriority == minPriority
		}
		
		// Add edit mode indicator and highlighting
		priorityText := fmt.Sprintf("%d", effectivePriority)
		if v.tuiApp != nil && v.tuiApp.IsInEditMode() {
			priorityText += " [编辑]"
			if isHighestPriority {
				priorityText = fmt.Sprintf("[red::b]%d [编辑][white::-]", effectivePriority)  // Highlight highest priority
			}
		} else if isHighestPriority {
			priorityText = fmt.Sprintf("[green::b]%d[white::-]", effectivePriority)  // Highlight highest priority in normal mode
		}
		
		// Safely update cells
		if row < v.table.GetRowCount() {
			if cell := v.table.GetCell(row, 0); cell != nil {
				cell.SetText(statusIcon)
			}
			if cell := v.table.GetCell(row, 1); cell != nil {
				cell.SetText(ep.Config.Name)
			}
			if cell := v.table.GetCell(row, 2); cell != nil {
				cell.SetText(truncateString(ep.Config.URL, 25))
			}
			if cell := v.table.GetCell(row, 3); cell != nil {
				cell.SetText(priorityText)
			}
			if cell := v.table.GetCell(row, 4); cell != nil {
				cell.SetText(fmt.Sprintf("%dms", status.ResponseTime.Milliseconds()))
			}
			if cell := v.table.GetCell(row, 5); cell != nil {
				cell.SetText(fmt.Sprintf("%d", totalReqs))
			}
		}
	}
	
	// Auto-select first row if no row is selected and endpoints exist
	if v.selectedRow == 0 && len(endpoints) > 0 {
		v.table.Select(1, 0) // Select first data row (row 1, column 0)
		v.selectedRow = 1
	}
}

// updateDetails updates the detail view for the selected endpoint
func (v *EndpointsView) updateDetails() {
	endpoints := v.endpointManager.GetAllEndpoints()
	metrics := v.monitoringMiddleware.GetMetrics().GetMetrics()
	
	// Check if selected row is valid
	if v.selectedRow <= 0 || v.selectedRow > len(endpoints) {
		return
	}
	
	endpoint := endpoints[v.selectedRow-1] // Subtract 1 for header row
	status := endpoint.GetStatus()
	
	var detailText strings.Builder
	detailText.WriteString(fmt.Sprintf("[blue::b]🎯 %s[white::-]\n\n", endpoint.Config.Name))
	
	// Basic Info
	detailText.WriteString("[yellow::b]Basic Information[white::-]\n")
	detailText.WriteString(fmt.Sprintf("URL: [cyan]%s[white]\n", endpoint.Config.URL))
	detailText.WriteString(fmt.Sprintf("Priority: [cyan]%d[white]\n", endpoint.Config.Priority))
	detailText.WriteString(fmt.Sprintf("Timeout: [cyan]%v[white]\n", endpoint.Config.Timeout))
	
	// Health Status
	detailText.WriteString("\n[yellow::b]Health Status[white::-]\n")
	healthStatus := "[red]Unhealthy[white]"
	healthIcon := "🔴"
	if status.Healthy {
		healthStatus = "[green]Healthy[white]"
		healthIcon = "🟢"
	}
	detailText.WriteString(fmt.Sprintf("Status: %s %s\n", healthIcon, healthStatus))
	detailText.WriteString(fmt.Sprintf("Response Time: [cyan]%dms[white]\n", status.ResponseTime.Milliseconds()))
	detailText.WriteString(fmt.Sprintf("Last Check: [cyan]%v[white]\n", status.LastCheck.Format("15:04:05")))
	detailText.WriteString(fmt.Sprintf("Consecutive Fails: [red]%d[white]\n", status.ConsecutiveFails))
	
	// Performance Metrics
	if endpointStats := metrics.EndpointStats[endpoint.Config.Name]; endpointStats != nil {
		detailText.WriteString("\n[yellow::b]Performance Metrics[white::-]\n")
		detailText.WriteString(fmt.Sprintf("Total Requests: [cyan]%d[white]\n", endpointStats.TotalRequests))
		detailText.WriteString(fmt.Sprintf("Successful: [green]%d[white]\n", endpointStats.SuccessfulRequests))
		detailText.WriteString(fmt.Sprintf("Failed: [red]%d[white]\n", endpointStats.FailedRequests))
		detailText.WriteString(fmt.Sprintf("Retries: [yellow]%d[white]\n", endpointStats.RetryCount))
		
		if endpointStats.TotalRequests > 0 {
			avgResponseTime := endpointStats.TotalResponseTime / time.Duration(endpointStats.TotalRequests)
			successRate := float64(endpointStats.SuccessfulRequests) / float64(endpointStats.TotalRequests) * 100
			
			detailText.WriteString(fmt.Sprintf("Success Rate: [cyan]%.1f%%[white]\n", successRate))
			detailText.WriteString(fmt.Sprintf("Avg Response: [cyan]%s[white]\n", formatDurationShort(avgResponseTime)))
			detailText.WriteString(fmt.Sprintf("Min Response: [cyan]%s[white]\n", formatDurationShort(endpointStats.MinResponseTime)))
			detailText.WriteString(fmt.Sprintf("Max Response: [cyan]%s[white]\n", formatDurationShort(endpointStats.MaxResponseTime)))
		}
		
		if !endpointStats.LastUsed.IsZero() {
			detailText.WriteString(fmt.Sprintf("Last Used: [cyan]%v[white]\n", endpointStats.LastUsed.Format("15:04:05")))
		}
		
		// Token Usage Metrics
		if endpointStats.TokenUsage.InputTokens > 0 || endpointStats.TokenUsage.OutputTokens > 0 {
			detailText.WriteString("\n[yellow::b]🪙 Token Usage[white::-]\n")
			detailText.WriteString(fmt.Sprintf("📥 Input Tokens: [cyan]%d[white]\n", endpointStats.TokenUsage.InputTokens))
			detailText.WriteString(fmt.Sprintf("📤 Output Tokens: [cyan]%d[white]\n", endpointStats.TokenUsage.OutputTokens))
			detailText.WriteString(fmt.Sprintf("🆕 Cache Creation: [cyan]%d[white]\n", endpointStats.TokenUsage.CacheCreationTokens))
			detailText.WriteString(fmt.Sprintf("📖 Cache Read: [cyan]%d[white]\n", endpointStats.TokenUsage.CacheReadTokens))
			totalTokens := endpointStats.TokenUsage.InputTokens + endpointStats.TokenUsage.OutputTokens
			detailText.WriteString(fmt.Sprintf("🔢 Total Tokens: [magenta]%d[white]\n", totalTokens))
		}
	} else {
		detailText.WriteString("\n[yellow::b]Performance Metrics[white::-]\n")
		detailText.WriteString("[gray]No requests processed yet[white]\n")
	}
	
	// Connection Info
	detailText.WriteString("\n[yellow::b]Connection Details[white::-]\n")
	activeConnections := 0
	for _, conn := range metrics.ActiveConnections {
		if conn.Endpoint == endpoint.Config.Name {
			activeConnections++
		}
	}
	detailText.WriteString(fmt.Sprintf("Active Connections: [cyan]%d[white]\n", activeConnections))
	
	// Only update if content changed
	newContent := detailText.String()
	if newContent != v.lastDetailHash {
		v.lastDetailHash = newContent
		v.detailBox.SetText(newContent)
	}
}

// ConnectionsView represents the connections tab
type ConnectionsView struct {
	container           *tview.Flex
	statsBox            *tview.TextView
	monitoringMiddleware *middleware.MonitoringMiddleware
	config              *config.Config
	lastDisplayHash     string // Track content changes to avoid unnecessary updates
	needsUpdate         bool   // Flag to indicate if data has changed since last display
}

func NewConnectionsView(monitoringMiddleware *middleware.MonitoringMiddleware, cfg *config.Config) *ConnectionsView {
	view := &ConnectionsView{
		monitoringMiddleware: monitoringMiddleware,
		config:              cfg,
	}
	view.setupUI()
	return view
}

func (v *ConnectionsView) setupUI() {
	v.statsBox = tview.NewTextView().SetDynamicColors(true).SetScrollable(true)
	v.statsBox.SetBorder(true).SetTitle(" 🔌 Active Connections ").SetTitleAlign(tview.AlignLeft)
	
	v.container = tview.NewFlex().AddItem(v.statsBox, 0, 1, true)
}

func (v *ConnectionsView) GetPrimitive() tview.Primitive {
	return v.container
}

func (v *ConnectionsView) Update() {
	metrics := v.monitoringMiddleware.GetMetrics().GetMetrics()
	
	// Build display text
	var stats strings.Builder
	stats.WriteString(fmt.Sprintf("[blue::b]📊 Connection Statistics[white::-]\n"))
	stats.WriteString(fmt.Sprintf("Active: [cyan]%3d[white] | Historical: [cyan]%4d[white]\n\n", 
		len(metrics.ActiveConnections), len(metrics.ConnectionHistory)))
	
	stats.WriteString("[blue::b]🔗 Active Connections[white::-]\n")
	
	// Convert map to slice for stable sorting
	connections := make([]*monitor.ConnectionInfo, 0, len(metrics.ActiveConnections))
	for _, conn := range metrics.ActiveConnections {
		connections = append(connections, conn)
	}
	
	// Sort connections by start time (newest first) for stable ordering
	sort.Slice(connections, func(i, j int) bool {
		return connections[i].StartTime.After(connections[j].StartTime)
	})
	
	// Always show exactly 15 lines to maintain consistent height
	connCount := 0
	for _, conn := range connections {
		if connCount >= 15 {
			break
		}
		duration := time.Since(conn.StartTime)
		
		// Display endpoint name and retry count
		endpointDisplay := conn.Endpoint
		if endpointDisplay == "" || endpointDisplay == "unknown" {
			endpointDisplay = "pending"
		}
		
		retryDisplay := ""
		if conn.RetryCount >= 0 {
			maxAttempts := v.config.Retry.MaxAttempts
			retryDisplay = fmt.Sprintf(" (%d/%d retry)", conn.RetryCount, maxAttempts)
		}
		
		stats.WriteString(fmt.Sprintf("  [cyan]%-15s[white] %-6s %-20s -> [yellow]%s[white]%s [gray](%8s)[white]\n",
			truncateString(conn.ClientIP, 15),
			conn.Method,
			truncateString(conn.Path, 20),
			truncateString(endpointDisplay, 8),
			retryDisplay,
			formatDurationShort(duration)))
		connCount++
	}
	
	// Fill remaining lines to maintain consistent height
	for connCount < 15 {
		if connCount == 0 {
			stats.WriteString("  [gray]No active connections[white]\n")
		} else {
			stats.WriteString("\n")
		}
		connCount++
	}
	
	// Only update if content has changed
	newContent := stats.String()
	if newContent != v.lastDisplayHash {
		v.lastDisplayHash = newContent
		v.statsBox.SetText(newContent)
	}
}

// LogEntry represents a log entry
type LogEntry struct {
	Timestamp time.Time
	Level     string
	Message   string
	Source    string
}

// LogsView represents the logs tab
type LogsView struct {
	container       *tview.Flex
	logText         *tview.TextView
	logs            []LogEntry
	mutex           sync.RWMutex
	maxLogs         int
	lastDisplayHash string // Track content changes to avoid unnecessary updates
	needsUpdate     bool   // Flag to indicate if logs have changed since last display
}

func NewLogsView() *LogsView {
	view := &LogsView{
		logs:    make([]LogEntry, 0),
		maxLogs: 500,
	}
	view.setupUI()
	return view
}

func (v *LogsView) setupUI() {
	v.logText = tview.NewTextView().SetDynamicColors(false).SetScrollable(true).SetWrap(true)
	v.logText.SetBorder(true).SetTitle(" System Logs ").SetTitleAlign(tview.AlignLeft)
	
	v.container = tview.NewFlex().AddItem(v.logText, 0, 1, true)
}

func (v *LogsView) GetPrimitive() tview.Primitive {
	return v.container
}

func (v *LogsView) Update() {
	v.refreshLogDisplay()
}

func (v *LogsView) ForceUpdate() {
	v.mutex.Lock()
	v.needsUpdate = true // Force update regardless of current state
	v.mutex.Unlock()
	v.refreshLogDisplay()
}

func (v *LogsView) AddLog(level, message, source string) {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Source:    source,
	}
	
	v.logs = append(v.logs, entry)
	if len(v.logs) > v.maxLogs {
		v.logs = v.logs[len(v.logs)-v.maxLogs:]
	}
	v.needsUpdate = true
}

func (v *LogsView) AddLogSilent(level, message, source string) {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Source:    source,
	}
	
	v.logs = append(v.logs, entry)
	if len(v.logs) > v.maxLogs {
		v.logs = v.logs[len(v.logs)-v.maxLogs:]
	}
	// Don't set needsUpdate=true to avoid triggering UI refresh
}

func (v *LogsView) refreshLogDisplay() {
	v.mutex.RLock()
	needsUpdate := v.needsUpdate
	v.mutex.RUnlock()
	
	// Only update if there are new logs
	if !needsUpdate {
		return
	}
	
	v.mutex.Lock()
	defer v.mutex.Unlock()
	
	v.needsUpdate = false
	
	// Build display text
	var displayText strings.Builder
	
	start := 0
	if len(v.logs) > 50 {
		start = len(v.logs) - 50
	}
	
	for i := start; i < len(v.logs); i++ {
		entry := v.logs[i]
		timeStr := entry.Timestamp.Format("15:04:05")
		
		// Simplified log display without emojis and complex formatting
		var levelStr string
		switch strings.ToUpper(entry.Level) {
		case "ERROR":
			levelStr = "[ERR]"
		case "WARN":
			levelStr = "[WRN]"
		case "INFO":
			levelStr = "[INF]"
		default:
			levelStr = "[LOG]"
		}
		
		displayText.WriteString(fmt.Sprintf("%s %s %s: %s\n",
			timeStr, levelStr, entry.Source, entry.Message))
	}
	
	// Only update if content has changed
	newContent := displayText.String()
	if newContent != v.lastDisplayHash {
		v.lastDisplayHash = newContent
		v.logText.SetText(newContent)
		// Scroll to end after setting new text
		v.logText.ScrollToEnd()
	}
}


// ConfigView represents the config tab
type ConfigView struct {
	container  *tview.Flex
	configText *tview.TextView
	cfg        *config.Config
}

func NewConfigView(cfg *config.Config) *ConfigView {
	view := &ConfigView{cfg: cfg}
	view.setupUI()
	return view
}

func (v *ConfigView) setupUI() {
	v.configText = tview.NewTextView().SetDynamicColors(true).SetScrollable(true)
	v.configText.SetBorder(true).SetTitle(" ⚙️ Configuration ").SetTitleAlign(tview.AlignLeft)
	
	v.container = tview.NewFlex().AddItem(v.configText, 0, 1, true)
}

func (v *ConfigView) GetPrimitive() tview.Primitive {
	return v.container
}

func (v *ConfigView) Update() {
	var details strings.Builder
	
	details.WriteString("[blue::b]🌐 Server[white::-]\n")
	details.WriteString(fmt.Sprintf("Host: [cyan]%s[white] | Port: [cyan]%d[white]\n\n", v.cfg.Server.Host, v.cfg.Server.Port))
	
	details.WriteString("[blue::b]🎯 Strategy[white::-]\n")
	details.WriteString(fmt.Sprintf("Type: [yellow]%s[white] | Fast Test: [yellow]%t[white]\n\n", 
		strings.Title(v.cfg.Strategy.Type), v.cfg.Strategy.FastTestEnabled))
	
	details.WriteString("[blue::b]🔐 Authentication[white::-]\n")
	if v.cfg.Auth.Enabled {
		details.WriteString("Status: [green]Enabled[white]\n")
	} else {
		details.WriteString("Status: [red]Disabled[white]\n")
	}
	details.WriteString("\n")
	
	details.WriteString("[blue::b]🖥️ TUI Settings[white::-]\n")
	details.WriteString(fmt.Sprintf("Update Interval: [cyan]%v[white]\n\n", v.cfg.TUI.UpdateInterval))
	
	details.WriteString("[blue::b]🎯 Endpoints[white::-]\n")
	details.WriteString(fmt.Sprintf("Total: [cyan]%d[white]\n", len(v.cfg.Endpoints)))
	for i, ep := range v.cfg.Endpoints {
		if i >= 8 {
			details.WriteString("[gray]... and more[white]\n")
			break
		}
		details.WriteString(fmt.Sprintf("  • [cyan]%s[white] ([yellow]%s[white]) P:%d\n",
			ep.Name, truncateString(ep.URL, 25), ep.Priority))
	}
	
	v.configText.SetText(details.String())
}

// Helper functions
func formatDurationShort(d time.Duration) string {
	if d == 0 {
		return "0ms"
	}
	if d < time.Millisecond {
		return fmt.Sprintf("%.0fμs", float64(d.Nanoseconds())/1000)
	}
	if d < time.Second {
		return fmt.Sprintf("%.0fms", float64(d.Nanoseconds())/1000000)
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func formatUptimeShort(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.0fm%.0fs", d.Minutes(), math.Mod(d.Seconds(), 60))
	} else if d < 24*time.Hour {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		return fmt.Sprintf("%dh%dm", hours, minutes)
	} else {
		days := int(d.Hours() / 24)
		hours := int(d.Hours()) % 24
		return fmt.Sprintf("%dd%dh", days, hours)
	}
}