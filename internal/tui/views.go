package tui

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/rivo/tview"
	
	"endpoint_forwarder/config"
	"endpoint_forwarder/internal/endpoint"
	"endpoint_forwarder/internal/middleware"
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
}

// NewOverviewView creates a new overview view
func NewOverviewView(monitoringMiddleware *middleware.MonitoringMiddleware, endpointManager *endpoint.Manager) *OverviewView {
	view := &OverviewView{
		monitoringMiddleware: monitoringMiddleware,
		endpointManager:     endpointManager,
		responseTimeHistory: make([]time.Duration, 0, 60),
	}
	view.setupUI()
	return view
}

func (v *OverviewView) setupUI() {
	v.metricsBox = tview.NewTextView().SetDynamicColors(true).SetScrollable(false)
	v.metricsBox.SetBorder(true).SetTitle(" 📊 Request Metrics ").SetTitleAlign(tview.AlignLeft)

	v.chartBox = tview.NewTextView().SetDynamicColors(true).SetScrollable(false)
	v.chartBox.SetBorder(true).SetTitle(" 📈 Response Time Trend ").SetTitleAlign(tview.AlignLeft)

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
		AddItem(topFlex, 8, 0, false).    // Fixed height for top section
		AddItem(bottomFlex, 0, 1, false)  // Remaining space for bottom
}

func (v *OverviewView) GetPrimitive() tview.Primitive {
	return v.container
}

func (v *OverviewView) Update() {
	metrics := v.monitoringMiddleware.GetMetrics().GetMetrics()
	
	// Update metrics
	avgTime := formatDurationShort(metrics.GetAverageResponseTime())
	successRate := metrics.GetSuccessRate()
	
	metricsText := fmt.Sprintf(`[white::b]Total Requests:[white::-] [cyan]%8d[white]
[white::b]Successful:[white::-] [green]%8d[white] ([green]%5.1f%%[white])
[white::b]Failed:[white::-] [red]%8d[white] ([red]%5.1f%%[white])
[white::b]Avg Response Time:[white::-] [cyan]%8s[white]`,
		metrics.TotalRequests,
		metrics.SuccessfulRequests, successRate,
		metrics.FailedRequests, 100-successRate,
		avgTime)

	v.metricsBox.SetText(metricsText)
	
	// Simple chart placeholder
	v.chartBox.SetText("[gray]Response time trending...\n[green]████████░░[white] Low\n[yellow]██████░░░░[white] Med\n[red]███░░░░░░░[white] High")
	
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
	
	v.endpointsBox.SetText(statusText.String())
	
	// System info - fixed width formatting (removed uptime component)
	systemText := fmt.Sprintf(`[white::b]Active Connections:[white::-] [cyan]%6d[white]
[white::b]Total Connections:[white::-] [cyan]%7d[white]`,
		len(metrics.ActiveConnections),
		len(metrics.ActiveConnections)+len(metrics.ConnectionHistory))

	v.systemBox.SetText(systemText)
}

// EndpointsView represents the endpoints tab
type EndpointsView struct {
	container           *tview.Flex
	table               *tview.Table
	detailBox           *tview.TextView
	monitoringMiddleware *middleware.MonitoringMiddleware
	endpointManager     *endpoint.Manager
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
	v.table.SetBorder(true).SetTitle(" 🎯 Endpoints ").SetTitleAlign(tview.AlignLeft)
	
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

func (v *EndpointsView) GetPrimitive() tview.Primitive {
	return v.container
}

func (v *EndpointsView) Update() {
	v.updateTable()
	v.detailBox.SetText("[gray]Select an endpoint to view details[white]")
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
				cell.SetText(fmt.Sprintf("%d", ep.Config.Priority))
			}
			if cell := v.table.GetCell(row, 4); cell != nil {
				cell.SetText(fmt.Sprintf("%dms", status.ResponseTime.Milliseconds()))
			}
			if cell := v.table.GetCell(row, 5); cell != nil {
				cell.SetText(fmt.Sprintf("%d", totalReqs))
			}
		}
	}
}

// ConnectionsView represents the connections tab
type ConnectionsView struct {
	container           *tview.Flex
	statsBox            *tview.TextView
	monitoringMiddleware *middleware.MonitoringMiddleware
}

func NewConnectionsView(monitoringMiddleware *middleware.MonitoringMiddleware) *ConnectionsView {
	view := &ConnectionsView{
		monitoringMiddleware: monitoringMiddleware,
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
	
	var stats strings.Builder
	stats.WriteString(fmt.Sprintf("[blue::b]📊 Connection Statistics[white::-]\n"))
	stats.WriteString(fmt.Sprintf("Active: [cyan]%3d[white] | Historical: [cyan]%4d[white]\n\n", 
		len(metrics.ActiveConnections), len(metrics.ConnectionHistory)))
	
	stats.WriteString("[blue::b]🔗 Active Connections[white::-]\n")
	
	// Always show exactly 15 lines to maintain consistent height
	connCount := 0
	for _, conn := range metrics.ActiveConnections {
		if connCount >= 15 {
			break
		}
		duration := time.Since(conn.StartTime)
		stats.WriteString(fmt.Sprintf("  [cyan]%-15s[white] %-6s %-25s [gray](%8s)[white]\n",
			truncateString(conn.ClientIP, 15),
			conn.Method,
			truncateString(conn.Path, 25),
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
	
	v.statsBox.SetText(stats.String())
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
}

func NewLogsView() *LogsView {
	view := &LogsView{
		logs:    make([]LogEntry, 0),
		maxLogs: 500,
	}
	view.setupUI()
	view.startLogSimulation()
	return view
}

func (v *LogsView) setupUI() {
	v.logText = tview.NewTextView().SetDynamicColors(true).SetScrollable(true).SetChangedFunc(func() {
		v.logText.ScrollToEnd()
	})
	v.logText.SetBorder(true).SetTitle(" 📜 System Logs ").SetTitleAlign(tview.AlignLeft)
	
	v.container = tview.NewFlex().AddItem(v.logText, 0, 1, true)
}

func (v *LogsView) GetPrimitive() tview.Primitive {
	return v.container
}

func (v *LogsView) Update() {
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
}

func (v *LogsView) refreshLogDisplay() {
	v.mutex.RLock()
	defer v.mutex.RUnlock()
	
	// Build display text
	var displayText strings.Builder
	
	start := 0
	if len(v.logs) > 50 {
		start = len(v.logs) - 50
	}
	
	for i := start; i < len(v.logs); i++ {
		entry := v.logs[i]
		timeStr := entry.Timestamp.Format("15:04:05")
		
		var levelColor, levelIcon string
		switch strings.ToUpper(entry.Level) {
		case "ERROR":
			levelColor = "[red]"
			levelIcon = "❌"
		case "WARN":
			levelColor = "[yellow]"
			levelIcon = "⚠️"
		case "INFO":
			levelColor = "[blue]"
			levelIcon = "ℹ️"
		default:
			levelColor = "[white]"
			levelIcon = "📝"
		}
		
		displayText.WriteString(fmt.Sprintf("[gray]%s[white] %s%s[white] [cyan]%s[white]: %s\n",
			timeStr, levelColor, levelIcon, entry.Source, entry.Message))
	}
	
	// Only update if content has changed
	newContent := displayText.String()
	if newContent != v.lastDisplayHash {
		v.lastDisplayHash = newContent
		v.logText.SetText(newContent)
	}
}

func (v *LogsView) startLogSimulation() {
	go func() {
		ticker := time.NewTicker(4 * time.Second)
		defer ticker.Stop()
		
		sampleLogs := []struct {
			level   string
			message string
			source  string
		}{
			{"INFO", "Request processed successfully", "proxy"},
			{"INFO", "Health check completed", "health"},
			{"WARN", "High response time detected", "monitor"},
			{"ERROR", "Connection timeout", "network"},
			{"INFO", "Configuration reloaded", "config"},
		}
		
		logIndex := 0
		for range ticker.C {
			log := sampleLogs[logIndex%len(sampleLogs)]
			v.AddLog(log.level, log.message, log.source)
			logIndex++
		}
	}()
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