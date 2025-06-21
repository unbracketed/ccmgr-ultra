package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/your-username/ccmgr-ultra/internal/config"
	"github.com/your-username/ccmgr-ultra/internal/tui/components"
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
	ctx          context.Context
	config       *config.Config
	integration  *Integration
	keyHandler   *KeyHandler
	
	// Screen management
	currentScreen AppScreen
	screens       map[AppScreen]tea.Model
	
	// Application state
	width          int
	height         int
	statusBar      components.StatusBarModel
	ready          bool
	quitting       bool
	
	// Styles
	theme Theme
}

// Theme holds the color scheme and styles for the TUI
type Theme struct {
	Primary     lipgloss.Color
	Secondary   lipgloss.Color
	Accent      lipgloss.Color
	Background  lipgloss.Color
	Text        lipgloss.Color
	Muted       lipgloss.Color
	Success     lipgloss.Color
	Warning     lipgloss.Color
	Error       lipgloss.Color
	
	BorderStyle  lipgloss.Border
	TitleStyle   lipgloss.Style
	HeaderStyle  lipgloss.Style
	ContentStyle lipgloss.Style
	FooterStyle  lipgloss.Style
}

// DefaultTheme returns the default color theme
func DefaultTheme() Theme {
	return Theme{
		Primary:     lipgloss.Color("#646CFF"),
		Secondary:   lipgloss.Color("#747BFF"),
		Accent:      lipgloss.Color("#42A5F5"),
		Background:  lipgloss.Color("#1E1E2E"),
		Text:        lipgloss.Color("#CDD6F4"),
		Muted:       lipgloss.Color("#6C7086"),
		Success:     lipgloss.Color("#A6E3A1"),
		Warning:     lipgloss.Color("#F9E2AF"),
		Error:       lipgloss.Color("#F38BA8"),
		
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

	// Create app model
	app := &AppModel{
		ctx:           ctx,
		config:        config,
		integration:   integration,
		keyHandler:    keyHandler,
		currentScreen: ScreenDashboard,
		screens:       make(map[AppScreen]tea.Model),
		statusBar:     components.NewStatusBarModel(components.Theme{
			Primary:     theme.Primary,
			Secondary:   theme.Secondary,
			Background:  theme.Background,
			Text:        theme.Text,
			Muted:       theme.Muted,
			Success:     theme.Success,
			Warning:     theme.Warning,
			Error:       theme.Error,
		}),
		theme:         theme,
	}

	// Initialize screens
	app.initializeScreens()

	return app, nil
}

// initializeScreens creates all screen models
func (m *AppModel) initializeScreens() {
	m.screens[ScreenDashboard] = NewDashboardModel(m.integration, m.theme)
	m.screens[ScreenSessions] = NewSessionsModel(m.integration, m.theme)
	m.screens[ScreenWorktrees] = NewWorktreesModel(m.integration, m.theme)
	m.screens[ScreenConfig] = NewConfigModel(m.config, m.theme)
	m.screens[ScreenHelp] = NewHelpModel(m.theme)
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
		// Handle global key bindings first
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

	default:
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
	return lipgloss.JoinVertical(lipgloss.Left, contentArea, statusBar)
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