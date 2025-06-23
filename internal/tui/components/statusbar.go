package components

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// StatusBarModel represents the status bar at the bottom of the screen
type StatusBarModel struct {
	theme  Theme
	width  int
	height int

	// Status information
	currentScreen string
	keyHelp       []string
	systemStatus  SystemStatus
	lastUpdate    time.Time

	// Styles
	barStyle    lipgloss.Style
	leftStyle   lipgloss.Style
	centerStyle lipgloss.Style
	rightStyle  lipgloss.Style
}

// Theme holds the color scheme for the status bar
type Theme struct {
	Primary    lipgloss.Color
	Secondary  lipgloss.Color
	Background lipgloss.Color
	Text       lipgloss.Color
	Muted      lipgloss.Color
	Success    lipgloss.Color
	Warning    lipgloss.Color
	Error      lipgloss.Color
}

// SystemStatus holds system-wide status information
type SystemStatus struct {
	ActiveProcesses  int
	ActiveSessions   int
	TrackedWorktrees int
	LastUpdate       time.Time
	IsHealthy        bool
	Errors           []string
}

// NewStatusBarModel creates a new status bar model
func NewStatusBarModel(theme Theme) StatusBarModel {
	barStyle := lipgloss.NewStyle().
		Background(theme.Background).
		Foreground(theme.Text).
		Padding(0, 1)

	leftStyle := lipgloss.NewStyle().
		Background(theme.Primary).
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(0, 1).
		Bold(true)

	centerStyle := lipgloss.NewStyle().
		Background(theme.Background).
		Foreground(theme.Text).
		Padding(0, 1)

	rightStyle := lipgloss.NewStyle().
		Background(theme.Secondary).
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(0, 1)

	return StatusBarModel{
		theme:       theme,
		barStyle:    barStyle,
		leftStyle:   leftStyle,
		centerStyle: centerStyle,
		rightStyle:  rightStyle,
		lastUpdate:  time.Now(),
	}
}

// Init implements the tea.Model interface
func (m StatusBarModel) Init() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// Update implements the tea.Model interface
func (m StatusBarModel) Update(msg tea.Msg) (StatusBarModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = 3 // Status bar is always 3 lines high

	case TickMsg:
		m.lastUpdate = time.Time(msg)
		return m, tea.Tick(time.Second, func(t time.Time) tea.Msg {
			return TickMsg(t)
		})

	case UpdateStatusMsg:
		m.currentScreen = msg.Screen
		m.keyHelp = msg.KeyHelp
		m.systemStatus = msg.SystemStatus

	case RefreshDataMsg:
		// Status bar will be updated via UpdateStatusMsg
	}

	return m, nil
}

// View implements the tea.Model interface
func (m StatusBarModel) View() string {
	if m.width == 0 {
		return ""
	}

	// Create the three sections of the status bar
	left := m.renderLeftSection()
	center := m.renderCenterSection()
	right := m.renderRightSection()

	// Calculate available widths
	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	centerWidth := m.width - leftWidth - rightWidth

	// Ensure center section fits
	if centerWidth < 0 {
		centerWidth = 0
		center = ""
	}

	// Apply styles with calculated widths
	leftStyled := m.leftStyle.Width(leftWidth).Render(left)
	centerStyled := m.centerStyle.Width(centerWidth).Render(center)
	rightStyled := m.rightStyle.Width(rightWidth).Render(right)

	// Combine sections
	topLine := lipgloss.JoinHorizontal(lipgloss.Left, leftStyled, centerStyled, rightStyled)

	// Add separator line
	separator := strings.Repeat("─", m.width)
	separatorStyled := lipgloss.NewStyle().
		Foreground(m.theme.Muted).
		Render(separator)

	// Add help line
	helpLine := m.renderHelpLine()

	return lipgloss.JoinVertical(lipgloss.Left, separatorStyled, topLine, helpLine)
}

// renderLeftSection creates the left section showing current screen and status
func (m StatusBarModel) renderLeftSection() string {
	screen := m.currentScreen
	if screen == "" {
		screen = "Dashboard"
	}

	// Add status indicator
	status := "●"
	if !m.systemStatus.IsHealthy {
		status = "⚠"
	}

	return fmt.Sprintf("%s %s", status, screen)
}

// renderCenterSection creates the center section showing system information
func (m StatusBarModel) renderCenterSection() string {
	if m.systemStatus.ActiveProcesses == 0 && m.systemStatus.ActiveSessions == 0 {
		return "No active sessions"
	}

	parts := []string{}

	if m.systemStatus.ActiveProcesses > 0 {
		parts = append(parts, fmt.Sprintf("Processes: %d", m.systemStatus.ActiveProcesses))
	}

	if m.systemStatus.ActiveSessions > 0 {
		parts = append(parts, fmt.Sprintf("Sessions: %d", m.systemStatus.ActiveSessions))
	}

	if m.systemStatus.TrackedWorktrees > 0 {
		parts = append(parts, fmt.Sprintf("Worktrees: %d", m.systemStatus.TrackedWorktrees))
	}

	return strings.Join(parts, " | ")
}

// renderRightSection creates the right section showing time and version
func (m StatusBarModel) renderRightSection() string {
	currentTime := m.lastUpdate.Format("15:04:05")
	return fmt.Sprintf("CCMGR Ultra | %s", currentTime)
}

// renderHelpLine creates the help line showing key bindings
func (m StatusBarModel) renderHelpLine() string {
	if len(m.keyHelp) == 0 {
		return m.centerStyle.Width(m.width).Render("Press ? for help")
	}

	// Join help items with separators
	helpText := strings.Join(m.keyHelp, " • ")

	// Truncate if too long
	if len(helpText) > m.width-4 {
		helpText = helpText[:m.width-7] + "..."
	}

	return m.centerStyle.Width(m.width).Render(helpText)
}

// UpdateStatus updates the status bar with new information
func (m StatusBarModel) UpdateStatus(screen string, keyHelp []string, systemStatus SystemStatus) StatusBarModel {
	m.currentScreen = screen
	m.keyHelp = keyHelp
	m.systemStatus = systemStatus
	return m
}

// SetTheme updates the status bar theme
func (m StatusBarModel) SetTheme(theme Theme) StatusBarModel {
	m.theme = theme

	// Update styles with new theme
	m.barStyle = m.barStyle.Background(theme.Background).Foreground(theme.Text)
	m.leftStyle = m.leftStyle.Background(theme.Primary)
	m.centerStyle = m.centerStyle.Background(theme.Background).Foreground(theme.Text)
	m.rightStyle = m.rightStyle.Background(theme.Secondary)

	return m
}

// UpdateStatusMsg is sent to update the status bar
type UpdateStatusMsg struct {
	Screen       string
	KeyHelp      []string
	SystemStatus SystemStatus
}

// RefreshDataMsg is sent when data should be refreshed
type RefreshDataMsg struct{}

// TickMsg is sent every second for time updates
type TickMsg time.Time

// NewUpdateStatusMsg creates a new status update message
func NewUpdateStatusMsg(screen string, keyHelp []string, systemStatus SystemStatus) UpdateStatusMsg {
	return UpdateStatusMsg{
		Screen:       screen,
		KeyHelp:      keyHelp,
		SystemStatus: systemStatus,
	}
}

// StatusBarHeight returns the height of the status bar
func StatusBarHeight() int {
	return 3
}

// DefaultSystemStatus returns a default system status
func DefaultSystemStatus() SystemStatus {
	return SystemStatus{
		ActiveProcesses:  0,
		ActiveSessions:   0,
		TrackedWorktrees: 0,
		LastUpdate:       time.Now(),
		IsHealthy:        true,
		Errors:           []string{},
	}
}
