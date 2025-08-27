package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	
	"endpoint_forwarder/config"
	"endpoint_forwarder/internal/endpoint"
	"endpoint_forwarder/internal/middleware"
)

// TUIApp represents the main TUI application
type TUIApp struct {
	app                  *tview.Application
	cfg                  *config.Config
	endpointManager      *endpoint.Manager
	monitoringMiddleware *middleware.MonitoringMiddleware
	startTime            time.Time
	
	// UI components
	pages     *tview.Pages
	tabBar    *tview.TextView
	statusBar *tview.TextView
	
	// Views
	overviewView    *OverviewView
	endpointsView   *EndpointsView
	connectionsView *ConnectionsView
	logsView        *LogsView
	configView      *ConfigView
	
	// State
	currentTab int
	tabs       []Tab
	ctx        context.Context
	cancel     context.CancelFunc
	running    bool
}

// Tab represents a tab in the TUI
type Tab struct {
	Name string
	View tview.Primitive
}

// NewTUIApp creates a new TUI application
func NewTUIApp(cfg *config.Config, endpointManager *endpoint.Manager, monitoringMiddleware *middleware.MonitoringMiddleware, startTime time.Time) *TUIApp {
	app := tview.NewApplication()
	ctx, cancel := context.WithCancel(context.Background())
	
	tuiApp := &TUIApp{
		app:                  app,
		cfg:                  cfg,
		endpointManager:      endpointManager,
		monitoringMiddleware: monitoringMiddleware,
		startTime:            startTime,
		ctx:                  ctx,
		cancel:               cancel,
		currentTab:           0,
		running:              false,
	}

	// Create UI components
	tuiApp.setupUI()
	
	return tuiApp
}

// setupUI creates and configures all UI components
func (t *TUIApp) setupUI() {
	// Create main pages container
	t.pages = tview.NewPages()

	// Create views
	t.overviewView = NewOverviewView(t.monitoringMiddleware, t.endpointManager, t.startTime)
	t.endpointsView = NewEndpointsView(t.monitoringMiddleware, t.endpointManager)
	t.connectionsView = NewConnectionsView(t.monitoringMiddleware, t.cfg)
	t.logsView = NewLogsView()
	t.configView = NewConfigView(t.cfg)

	// Define tabs
	t.tabs = []Tab{
		{"Overview", t.overviewView.GetPrimitive()},
		{"Endpoints", t.endpointsView.GetPrimitive()},
		{"Connections", t.connectionsView.GetPrimitive()},
		{"Logs", t.logsView.GetPrimitive()},
		{"Config", t.configView.GetPrimitive()},
	}

	// Create tab bar
	t.tabBar = tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWrap(false)
	t.tabBar.SetBorder(true).SetTitle(" Navigation ").SetTitleAlign(tview.AlignLeft)
	t.updateTabBar()

	// Create status bar
	t.statusBar = tview.NewTextView().
		SetDynamicColors(false).
		SetWrap(false).
		SetTextAlign(tview.AlignLeft)
	t.statusBar.SetBorder(true).SetTitle(" Status ").SetTitleAlign(tview.AlignLeft)
	t.updateStatusBar()

	// Create main layout
	mainFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(t.createHeaderFlex(), 3, 1, false).
		AddItem(t.tabBar, 3, 1, false).
		AddItem(t.pages, 0, 1, true).
		AddItem(t.statusBar, 3, 1, false)

	// Add pages to the container
	for i, tab := range t.tabs {
		t.pages.AddPage(tab.Name, tab.View, true, i == 0)
	}

	// Set input capture for tab navigation
	t.app.SetInputCapture(t.handleInput)

	// Set root and focus
	t.app.SetRoot(mainFlex, true).SetFocus(t.pages)
}

// createHeaderFlex creates the header section
func (t *TUIApp) createHeaderFlex() *tview.Flex {
	title := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText(fmt.Sprintf("[blue::b]🚀 Claude EndPoints Forwarder TUI[white::-]"))

	headerFlex := tview.NewFlex().
		AddItem(tview.NewTextView(), 1, 1, false).
		AddItem(title, 0, 1, false).
		AddItem(tview.NewTextView(), 1, 1, false)
	
	headerFlex.SetBorder(true).SetTitle(" Claude EndPoints Forwarder TUI ").SetTitleAlign(tview.AlignCenter)
	
	return headerFlex
}

// handleInput handles keyboard input for navigation
func (t *TUIApp) handleInput(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyTab:
		// Next tab
		t.currentTab = (t.currentTab + 1) % len(t.tabs)
		t.switchToTab(t.currentTab)
		t.updateTabBar()
		return nil
	case tcell.KeyBacktab:
		// Previous tab
		t.currentTab = (t.currentTab - 1 + len(t.tabs)) % len(t.tabs)
		t.switchToTab(t.currentTab)
		t.updateTabBar()
		return nil
	case tcell.KeyCtrlC:
		// Quit application
		t.Stop()
		return nil
	case tcell.KeyF1:
		t.showHelp()
		return nil
	}

	// Handle number keys for direct tab access
	if event.Rune() >= '1' && event.Rune() <= '9' {
		tabIndex := int(event.Rune() - '1')
		if tabIndex < len(t.tabs) {
			t.currentTab = tabIndex
			t.switchToTab(t.currentTab)
			t.updateTabBar()
		}
		return nil
	}

	return event
}

// switchToTab switches to the specified tab
func (t *TUIApp) switchToTab(tabIndex int) {
	if tabIndex >= 0 && tabIndex < len(t.tabs) {
		t.pages.SwitchToPage(t.tabs[tabIndex].Name)
		
		// Force update logs view when switching to it to show any missed logs
		if tabIndex == 3 && t.logsView != nil {
			t.logsView.ForceUpdate()
		}
	}
}

// updateTabBar updates the tab bar display
func (t *TUIApp) updateTabBar() {
	var tabText string
	for i, tab := range t.tabs {
		if i == t.currentTab {
			tabText += fmt.Sprintf(`[black:blue:b] %d: %s [white:black:-] `, i+1, tab.Name)
		} else {
			tabText += fmt.Sprintf(` [gray]%d: %s[white] `, i+1, tab.Name)
		}
	}
	tabText += `   [gray]Tab/Shift+Tab: Navigate  F1: Help  Ctrl+C: Quit[white]`
	t.tabBar.SetText(tabText)
}

// updateStatusBar updates the status bar
func (t *TUIApp) updateStatusBar() {
	metrics := t.monitoringMiddleware.GetMetrics().GetMetrics()
	
	// Very simple status text for testing
	statusText := fmt.Sprintf("Requests: %d | Success: %.1f%% | Connections: %d",
		metrics.TotalRequests,
		metrics.GetSuccessRate(),
		len(metrics.ActiveConnections),
	)
	t.statusBar.SetText(statusText)
}

// showHelp displays help information
func (t *TUIApp) showHelp() {
	helpText := `[blue::b]Claude Request Forwarder TUI - Help[white::-]

[yellow::b]Navigation:[white::-]
• Tab / Shift+Tab: Navigate between tabs
• 1-5: Jump directly to tab (1=Overview, 2=Endpoints, etc.)
• Arrow Keys: Navigate within views
• Enter: Select/Activate items
• F1: Show this help
• Ctrl+C: Quit application

[yellow::b]Tabs:[white::-]
• [green]Overview[white]: Real-time metrics and system status
• [green]Endpoints[white]: Endpoint health and performance details  
• [green]Connections[white]: Active connections and traffic info
• [green]Logs[white]: Real-time application logs
• [green]Config[white]: Current configuration display

[yellow::b]Features:[white::-]
• Real-time monitoring with 1-second refresh rate
• Color-coded status indicators (Green=Good, Yellow=Warning, Red=Error)
• Historical trending data and performance metrics
• Connection tracking and traffic analysis

Press any key to close this help.`

	modal := tview.NewModal().
		SetText(helpText).
		AddButtons([]string{"Close"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			t.app.SetRoot(t.app.GetFocus(), true)
		})
	
	t.app.SetRoot(modal, false).SetFocus(modal)
}

// Run starts the TUI application
func (t *TUIApp) Run() error {
	t.running = true
	
	// Start background refresh routine
	go t.refreshLoop()
	
	// Start the UI
	return t.app.Run()
}

// refreshLoop runs the background refresh routine
func (t *TUIApp) refreshLoop() {
	ticker := time.NewTicker(t.cfg.TUI.UpdateInterval)
	defer ticker.Stop()
	
	// Add a flag to prevent overlapping updates
	updating := false

	for t.running {
		select {
		case <-t.ctx.Done():
			return
		case <-ticker.C:
			if !t.running || updating {
				continue
			}
			
			updating = true
			// Use QueueUpdateDraw to ensure thread-safe UI updates
			t.app.QueueUpdateDraw(func() {
				defer func() { 
					updating = false 
					t.app.Sync()
				}()
				
				// Check if app is still running before updating
				if !t.running {
					return
				}
				
				// Update endpoint health in metrics first
				t.monitoringMiddleware.UpdateEndpointHealthStatus()
				
				// Update status bar
				t.updateStatusBar()
				
				// Update only the currently active view to reduce UI conflicts
				if t.currentTab >= 0 && t.currentTab < len(t.tabs) {
					switch t.currentTab {
					case 0:
						if t.overviewView != nil {
							t.overviewView.Update()
						}
					case 1:
						if t.endpointsView != nil {
							t.endpointsView.Update()
						}
					case 2:
						if t.connectionsView != nil {
							t.connectionsView.Update()
						}
					case 3:
						// Only update logs view when it's the active tab
						if t.logsView != nil {
							t.logsView.Update()
						}
					case 4:
						if t.configView != nil {
							t.configView.Update()
						}
					}
				}
			})
		}
	}
}

// AddLog adds a log entry to the logs view (thread-safe)
func (t *TUIApp) AddLog(level, message, source string) {
	if t.logsView != nil {
		// Only add log if logs tab is currently active to avoid unnecessary UI updates
		if t.currentTab == 3 {
			t.logsView.AddLog(level, message, source)
		} else {
			// Still add log to buffer but don't trigger UI update
			t.logsView.AddLogSilent(level, message, source)
		}
	}
}

// GetLogsView returns the logs view instance
func (t *TUIApp) GetLogsView() *LogsView {
	return t.logsView
}

// Stop gracefully stops the TUI application
func (t *TUIApp) Stop() {
	t.running = false
	t.cancel()
	t.app.Stop()
}

// IsRunning returns whether the TUI is currently running
func (t *TUIApp) IsRunning() bool {
	return t.running
}