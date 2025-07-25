package tui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/unbracketed/ccmgr-ultra/internal/config"
	"github.com/unbracketed/ccmgr-ultra/internal/tui/components"
	contextmenu "github.com/unbracketed/ccmgr-ultra/internal/tui/context"
	"github.com/unbracketed/ccmgr-ultra/internal/tui/modals"
	"github.com/unbracketed/ccmgr-ultra/internal/tui/workflows"
)

// AppScreen represents different screens in the TUI
type AppScreen int

const (
	ScreenDashboard AppScreen = iota
	ScreenSessions
	ScreenWorktrees
	ScreenConfig
	ScreenHelp
)

// AppModel represents the main application state
type AppModel struct {
	ctx         context.Context
	config      *config.Config
	integration *Integration
	keyHandler  *KeyHandler

	// Screen management
	currentScreen AppScreen
	screens       map[AppScreen]tea.Model

	// Modal and context menu management
	modalManager *modals.ModalManager
	contextMenu  *contextmenu.ContextMenu

	// Workflow management
	workflowFactory    *WorkflowFactory
	integrationAdapter *IntegrationAdapter
	sessionWizard      *workflows.SessionCreationWizard
	worktreeWizard     *workflows.WorktreeCreationWizard

	// Application state
	width     int
	height    int
	statusBar components.StatusBarModel
	ready     bool
	quitting  bool

	// Styles
	theme Theme
}

// Theme holds the color scheme and styles for the TUI
type Theme struct {
	Primary    lipgloss.Color
	Secondary  lipgloss.Color
	Accent     lipgloss.Color
	Background lipgloss.Color
	Text       lipgloss.Color
	Muted      lipgloss.Color
	Success    lipgloss.Color
	Warning    lipgloss.Color
	Error      lipgloss.Color
	Info       lipgloss.Color // New: for info status indicators

	BorderStyle   lipgloss.Border
	TitleStyle    lipgloss.Style
	HeaderStyle   lipgloss.Style
	ContentStyle  lipgloss.Style
	FooterStyle   lipgloss.Style
	SelectedStyle lipgloss.Style // New: for highlighting selected items
	StatusStyle   lipgloss.Style // New: for status/help bar
	LabelStyle    lipgloss.Style // New: for form labels
	FocusedStyle  lipgloss.Style // New: for focused form elements
	MutedStyle    lipgloss.Style // New: for muted text
	SuccessStyle  lipgloss.Style // New: for success messages
	ErrorStyle    lipgloss.Style // New: for error messages
	WarningStyle  lipgloss.Style // New: for warning messages
}

// DefaultTheme returns the default color theme
func DefaultTheme() Theme {
	return Theme{
		Primary:    lipgloss.Color("#646CFF"),
		Secondary:  lipgloss.Color("#747BFF"),
		Accent:     lipgloss.Color("#42A5F5"),
		Background: lipgloss.Color("#1E1E2E"),
		Text:       lipgloss.Color("#CDD6F4"),
		Muted:      lipgloss.Color("#6C7086"),
		Success:    lipgloss.Color("#A6E3A1"),
		Warning:    lipgloss.Color("#F9E2AF"),
		Error:      lipgloss.Color("#F38BA8"),
		Info:       lipgloss.Color("#89B4FA"), // New: blue for info status

		BorderStyle: lipgloss.RoundedBorder(),

		TitleStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#646CFF")).
			Bold(true).
			Padding(0, 1),

		HeaderStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#CDD6F4")).
			Bold(true).
			Padding(0, 1),

		ContentStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#CDD6F4")).
			Padding(1),

		FooterStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6C7086")).
			Padding(0, 1),

		SelectedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#1E1E2E")).
			Background(lipgloss.Color("#646CFF")).
			Bold(true),

		StatusStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#CDD6F4")).
			Background(lipgloss.Color("#313244")).
			Padding(0, 1),

		LabelStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#CDD6F4")).
			Bold(false),

		FocusedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#646CFF")).
			Bold(true),

		MutedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6C7086")),

		SuccessStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A6E3A1")).
			Bold(true),

		ErrorStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F38BA8")).
			Bold(true),

		WarningStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F9E2AF")).
			Bold(true),
	}
}

// NewAppModel creates a new application model
func NewAppModel(ctx context.Context, config *config.Config) (*AppModel, error) {
	// Initialize integration layer
	integration, err := NewIntegration(config)
	if err != nil {
		return nil, err
	}

	// Create key handler
	keyHandler := NewKeyHandler()

	// Initialize theme
	theme := DefaultTheme()

	// Convert theme for modal and context systems
	modalTheme := modals.Theme{
		Primary:      theme.Primary,
		Secondary:    theme.Secondary,
		Accent:       theme.Accent,
		Background:   theme.Background,
		Text:         theme.Text,
		Muted:        theme.Muted,
		Success:      theme.Success,
		Warning:      theme.Warning,
		Error:        theme.Error,
		BorderStyle:  theme.BorderStyle,
		TitleStyle:   theme.TitleStyle,
		ContentStyle: theme.ContentStyle,
		ButtonStyle:  theme.HeaderStyle,
		InputStyle:   theme.ContentStyle,
	}

	// TODO: Use contextTheme when implementing context menus
	_ = contextmenu.Theme{
		Primary:     theme.Primary,
		Secondary:   theme.Secondary,
		Accent:      theme.Accent,
		Background:  theme.Background,
		Text:        theme.Text,
		Muted:       theme.Muted,
		Success:     theme.Success,
		Warning:     theme.Warning,
		Error:       theme.Error,
		BorderStyle: theme.BorderStyle,
	}

	// Create app model
	app := &AppModel{
		ctx:           ctx,
		config:        config,
		integration:   integration,
		keyHandler:    keyHandler,
		currentScreen: ScreenDashboard,
		screens:       make(map[AppScreen]tea.Model),
		modalManager:  modals.NewModalManager(modalTheme),
		statusBar: components.NewStatusBarModel(components.Theme{
			Primary:    theme.Primary,
			Secondary:  theme.Secondary,
			Background: theme.Background,
			Text:       theme.Text,
			Muted:      theme.Muted,
			Success:    theme.Success,
			Warning:    theme.Warning,
			Error:      theme.Error,
		}),
		theme: theme,
	}

	// Create integration adapter for workflows
	integrationAdapter := NewIntegrationAdapter(integration, config)

	// Initialize workflow factory
	workflowFactory := NewWorkflowFactory(integrationAdapter, modalTheme)

	// Store workflow components in app model
	app.integrationAdapter = integrationAdapter
	app.workflowFactory = workflowFactory

	// Initialize wizards with adapter
	app.sessionWizard = workflowFactory.CreateSessionWizard()
	app.worktreeWizard = workflowFactory.CreateWorktreeWizard()

	// Initialize screens
	app.initializeScreens()

	return app, nil
}

// initializeScreens creates all screen models
func (m *AppModel) initializeScreens() {
	m.screens[ScreenDashboard] = NewDashboardModel(m.integration, m.theme)
	m.screens[ScreenSessions] = NewSessionsModel(m.integration, m.theme)
	m.screens[ScreenWorktrees] = NewWorktreesModel(m.integration, m.theme)
	m.screens[ScreenConfig] = NewConfigMenuModel(m.config, m.theme)
	m.screens[ScreenHelp] = NewHelpModel(m.theme)

	// Initialize all screens to ensure proper state
	for _, screen := range m.screens {
		screen.Init()
	}
}

// Init implements the tea.Model interface
func (m *AppModel) Init() tea.Cmd {
	// Start background data refresh
	return tea.Batch(
		m.integration.StartPeriodicRefresh(),
		tea.WindowSize(), // Get initial window size
	)
}

// Update implements the tea.Model interface
func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		// Update modal manager size
		m.modalManager.SetSize(msg.Width, msg.Height)

		// Update status bar size
		m.statusBar, cmd = m.statusBar.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		// Update current screen size
		if screen, exists := m.screens[m.currentScreen]; exists {
			screen, cmd = screen.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			m.screens[m.currentScreen] = screen
		}

	case tea.KeyMsg:
		// Handle modal input first if modal is active
		if m.modalManager.IsActive() {
			cmd = m.modalManager.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}

			// Check for modal completion
			if result := m.modalManager.GetResult(); result != nil {
				cmds = append(cmds, m.handleModalResult(result))
			}
			return m, tea.Batch(cmds...)
		}

		// Handle context menu input if active
		if m.contextMenu != nil && m.contextMenu.IsVisible() {
			m.contextMenu, cmd = m.contextMenu.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)
		}

		// Handle global key bindings
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit

		case "1":
			return m.switchScreen(ScreenDashboard)
		case "2":
			return m.switchScreen(ScreenSessions)
		case "3":
			return m.switchScreen(ScreenWorktrees)
		case "4":
			return m.switchScreen(ScreenConfig)
		case "?", "h":
			return m.switchScreen(ScreenHelp)

		// Workflow shortcuts
		case "ctrl+n":
			// New session wizard - TODO: Implement
			modal := modals.NewSimpleErrorModal("Not Implemented", "Session wizard not yet implemented")
			m.modalManager.ShowModal(modal)
			return m, nil

		case "ctrl+w":
			// New worktree wizard - TODO: Implement
			modal := modals.NewSimpleErrorModal("Not Implemented", "Worktree wizard not yet implemented")
			m.modalManager.ShowModal(modal)
			return m, nil

		// Dashboard quick actions (only when on dashboard screen)
		case "n":
			if m.currentScreen == ScreenDashboard {
				// New session - switch to worktrees screen
				return m.switchScreen(ScreenWorktrees)
			}
		case "w":
			if m.currentScreen == ScreenDashboard {
				// New worktree - switch to worktrees screen
				return m.switchScreen(ScreenWorktrees)
			}
		case "c":
			if m.currentScreen == ScreenDashboard {
				// Configuration - switch to config screen
				return m.switchScreen(ScreenConfig)
			}
		}

		// Pass key to current screen
		if screen, exists := m.screens[m.currentScreen]; exists {
			screen, cmd = screen.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			m.screens[m.currentScreen] = screen
		}

	case RefreshDataMsg:
		// Update all screens with new data
		for screenType, screen := range m.screens {
			screen, cmd = screen.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			m.screens[screenType] = screen
		}

		// Update status bar
		m.statusBar, cmd = m.statusBar.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case contextmenu.ContextMenuActionMsg:
		// Handle context menu action
		cmds = append(cmds, m.handleContextMenuAction(msg))

	case contextmenu.ContextMenuSubmenuMsg:
		// Show submenu
		m.contextMenu = msg.Submenu

	case NewSessionRequestedMsg:
		// Handle new session request from worktree selection
		return m.handleNewSessionRequest(msg)

	case ContinueSessionRequestedMsg:
		// Handle continue session request
		return m.handleContinueSessionRequest(msg)

	case ResumeSessionRequestedMsg:
		// Handle resume session request
		return m.handleResumeSessionRequest(msg)

	default:
		// Update modal manager
		if m.modalManager.IsActive() {
			cmd = m.modalManager.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}

			// Check for modal completion
			if result := m.modalManager.GetResult(); result != nil {
				cmds = append(cmds, m.handleModalResult(result))
			}
		}

		// Update context menu
		if m.contextMenu != nil && m.contextMenu.IsVisible() {
			m.contextMenu, cmd = m.contextMenu.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

		// Pass message to current screen
		if screen, exists := m.screens[m.currentScreen]; exists {
			screen, cmd = screen.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			m.screens[m.currentScreen] = screen
		}

		// Update status bar
		m.statusBar, cmd = m.statusBar.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// switchScreen changes the current screen and returns the updated model
func (m *AppModel) switchScreen(screen AppScreen) (tea.Model, tea.Cmd) {
	m.currentScreen = screen

	// Refresh screen data if needed
	if screenModel, exists := m.screens[screen]; exists {
		// Ensure screen has window size before refreshing data
		if m.width > 0 && m.height > 0 {
			// Send window size to screen first
			screenModel, _ = screenModel.Update(tea.WindowSizeMsg{
				Width:  m.width,
				Height: m.height,
			})
		}
		screenModel, cmd := screenModel.Update(RefreshDataMsg{})
		m.screens[screen] = screenModel
		return m, cmd
	}

	return m, nil
}

// View implements the tea.Model interface
func (m *AppModel) View() string {
	if !m.ready {
		return "Initializing CCMGR Ultra..."
	}

	if m.quitting {
		return "Thanks for using CCMGR Ultra! 👋\n"
	}

	// Get current screen content
	var content string
	if screen, exists := m.screens[m.currentScreen]; exists {
		content = screen.View()
	} else {
		content = "Screen not found"
	}

	// Calculate available height for content (subtract status bar)
	statusBarHeight := 3 // Status bar takes 3 lines
	contentHeight := m.height - statusBarHeight

	// Style the main content area
	contentArea := lipgloss.NewStyle().
		Width(m.width).
		Height(contentHeight).
		Render(content)

	// Render status bar
	statusBar := m.statusBar.View()

	// Combine content and status bar
	baseView := lipgloss.JoinVertical(lipgloss.Left, contentArea, statusBar)

	// Overlay modal if active
	if m.modalManager.IsActive() {
		modalView := m.modalManager.View()
		if modalView != "" {
			return modalView
		}
	}

	// Overlay context menu if visible
	if m.contextMenu != nil && m.contextMenu.IsVisible() {
		contextView := m.contextMenu.View()
		if contextView != "" {
			// TODO: Properly overlay context menu at correct position
			// For now, just return the base view with menu overlaid
			return baseView + "\n" + contextView
		}
	}

	return baseView
}

// GetCurrentScreen returns the current screen type
func (m *AppModel) GetCurrentScreen() AppScreen {
	return m.currentScreen
}

// GetIntegration returns the integration layer for external access
func (m *AppModel) GetIntegration() *Integration {
	return m.integration
}

// RefreshDataMsg is sent when data should be refreshed
type RefreshDataMsg struct{}

// TickMsg is sent periodically for animations or time-based updates
type TickMsg time.Time

// handleModalResult processes the result of a completed modal
func (m *AppModel) handleModalResult(result *modals.ModalResult) tea.Cmd {
	if result.Canceled {
		return nil
	}

	if result.Error != nil {
		// Show error modal
		errorModal := modals.NewSimpleErrorModal("Error", result.Error.Error())
		m.modalManager.ShowModal(errorModal)
		return nil
	}

	// Handle successful results based on data type
	switch data := result.Data.(type) {
	case map[string]interface{}:
		// Check if this is a session creation result
		if sessionName, ok := data["session_name"].(string); ok {
			return m.handleSessionCreation(data, sessionName)
		}

		// Check if this is a worktree creation result
		if worktreePath, ok := data["worktree_path"].(string); ok {
			return m.handleWorktreeCreation(data, worktreePath)
		}

	case string:
		// Handle simple string results
		return m.handleStringResult(data)
	}

	return nil
}

// handleContextMenuAction processes context menu actions
func (m *AppModel) handleContextMenuAction(msg contextmenu.ContextMenuActionMsg) tea.Cmd {
	// Hide context menu first
	if m.contextMenu != nil {
		m.contextMenu.Hide()
	}

	switch msg.Action {
	case "session_new":
		// TODO: Implement session wizard
		modal := modals.NewSimpleErrorModal("Not Implemented", "Session wizard not yet implemented")
		m.modalManager.ShowModal(modal)

	case "worktree_new":
		// TODO: Implement worktree wizard
		modal := modals.NewSimpleErrorModal("Not Implemented", "Worktree wizard not yet implemented")
		m.modalManager.ShowModal(modal)

	case "session_attach", "session_kill", "session_delete":
		return m.handleSessionAction(msg.Action)

	case "worktree_open", "worktree_remove":
		return m.handleWorktreeAction(msg.Action)

	case "config_edit", "config_reload", "config_validate":
		return m.handleConfigAction(msg.Action)

	default:
		// Show not implemented message
		modal := modals.NewSimpleErrorModal("Not Implemented",
			"Action '"+msg.Action+"' is not yet implemented")
		m.modalManager.ShowModal(modal)
	}

	return nil
}

// handleSessionCreation processes session creation results
func (m *AppModel) handleSessionCreation(data map[string]interface{}, sessionName string) tea.Cmd {
	// TODO: Implement actual session creation via integration
	// For now, show success message
	modal := modals.NewSimpleErrorModal("Success",
		"Session '"+sessionName+"' created successfully")
	m.modalManager.ShowModal(modal)

	// Refresh sessions screen
	return func() tea.Msg {
		return RefreshDataMsg{}
	}
}

// handleWorktreeCreation processes worktree creation results
func (m *AppModel) handleWorktreeCreation(data map[string]interface{}, worktreePath string) tea.Cmd {
	// TODO: Implement actual worktree creation via integration
	// For now, show success message
	modal := modals.NewSimpleErrorModal("Success",
		"Worktree created at '"+worktreePath+"'")
	m.modalManager.ShowModal(modal)

	// Refresh worktrees screen
	return func() tea.Msg {
		return RefreshDataMsg{}
	}
}

// handleStringResult processes simple string results
func (m *AppModel) handleStringResult(result string) tea.Cmd {
	// Show result in a modal
	modal := modals.NewSimpleErrorModal("Result", result)
	m.modalManager.ShowModal(modal)
	return nil
}

// handleSessionAction processes session-related actions
func (m *AppModel) handleSessionAction(action string) tea.Cmd {
	// TODO: Implement session actions via integration
	modal := modals.NewSimpleErrorModal("Not Implemented",
		"Session action '"+action+"' is not yet implemented")
	m.modalManager.ShowModal(modal)
	return nil
}

// handleWorktreeAction processes worktree-related actions
func (m *AppModel) handleWorktreeAction(action string) tea.Cmd {
	// TODO: Implement worktree actions via integration
	modal := modals.NewSimpleErrorModal("Not Implemented",
		"Worktree action '"+action+"' is not yet implemented")
	m.modalManager.ShowModal(modal)
	return nil
}

// handleConfigAction processes configuration-related actions
func (m *AppModel) handleConfigAction(action string) tea.Cmd {
	// TODO: Implement config actions
	modal := modals.NewSimpleErrorModal("Not Implemented",
		"Config action '"+action+"' is not yet implemented")
	m.modalManager.ShowModal(modal)
	return nil
}

// handleNewSessionRequest launches the session creation wizard
func (m *AppModel) handleNewSessionRequest(msg NewSessionRequestedMsg) (tea.Model, tea.Cmd) {
	if len(msg.Worktrees) == 1 {
		// Single worktree session creation
		modal := m.workflowFactory.CreateSingleWorktreeSessionWizard(msg.Worktrees[0])
		m.modalManager.ShowModal(modal)
	} else if len(msg.Worktrees) > 1 {
		// Bulk session creation
		modal := m.workflowFactory.CreateBulkWorktreeSessionWizard(msg.Worktrees)
		m.modalManager.ShowModal(modal)
	} else {
		// General session creation
		modal := m.workflowFactory.CreateGeneralSessionWizard()
		m.modalManager.ShowModal(modal)
	}
	return m, nil
}

// handleContinueSessionRequest finds and attaches to existing session
func (m *AppModel) handleContinueSessionRequest(msg ContinueSessionRequestedMsg) (tea.Model, tea.Cmd) {
	return m, func() tea.Msg {
		// Use workflow factory to handle continue operation
		err := m.workflowFactory.HandleContinueOperation(msg.Worktrees)
		if err != nil {
			return ErrorMsg{Error: err}
		}

		// For now, find existing sessions manually and attach to the first one
		for _, wt := range msg.Worktrees {
			sessions := m.integration.GetActiveSessionsForWorktree(wt.Path)
			if len(sessions) > 0 {
				// Attach to most recent session
				return m.integration.AttachSession(sessions[0].ID)
			}
		}

		// No sessions found
		return ErrorMsg{Error: fmt.Errorf("no existing sessions found for selected worktrees")}
	}
}

// handleResumeSessionRequest restores paused sessions
func (m *AppModel) handleResumeSessionRequest(msg ResumeSessionRequestedMsg) (tea.Model, tea.Cmd) {
	return m, func() tea.Msg {
		// Use workflow factory to handle resume operation
		err := m.workflowFactory.HandleResumeOperation(msg.Worktrees)
		if err != nil {
			return ErrorMsg{Error: err}
		}

		// Implementation for resuming paused sessions
		// This would integrate with tmux session restoration
		return SessionResumedMsg{Success: true}
	}
}
