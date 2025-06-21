package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/your-username/ccmgr-ultra/internal/config"
)

// Screen interface that all screens must implement
type Screen interface {
	tea.Model
	Title() string
	Help() []string
}

// DashboardModel represents the main dashboard screen
type DashboardModel struct {
	integration *Integration
	theme       Theme
	width       int
	height      int
}

// NewDashboardModel creates a new dashboard model
func NewDashboardModel(integration *Integration, theme Theme) *DashboardModel {
	return &DashboardModel{
		integration: integration,
		theme:       theme,
	}
}

func (m *DashboardModel) Init() tea.Cmd {
	return nil
}

func (m *DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case RefreshDataMsg:
		// Refresh dashboard data
		return m, nil
	}
	return m, nil
}

func (m *DashboardModel) View() string {
	if m.width == 0 {
		return "Loading dashboard..."
	}

	// Get system status
	status := m.integration.GetSystemStatus()
	
	// Create dashboard sections
	header := m.theme.HeaderStyle.Render("🚀 CCMGR Ultra Dashboard")
	
	// System overview
	overview := m.renderSystemOverview(status)
	
	// Active sessions
	sessions := m.renderActiveSessions()
	
	// Recent worktrees
	worktrees := m.renderRecentWorktrees()
	
	// Quick actions
	actions := m.renderQuickActions()

	// Layout sections
	sections := lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		overview,
		"",
		sessions,
		"",
		worktrees,
		"",
		actions,
	)

	return m.theme.ContentStyle.Render(sections)
}

func (m *DashboardModel) Title() string {
	return "Dashboard"
}

func (m *DashboardModel) Help() []string {
	return []string{
		"1-4: Switch screens",
		"?: Help",
		"q: Quit",
	}
}

func (m *DashboardModel) renderSystemOverview(status SystemStatus) string {
	title := m.theme.TitleStyle.Render("📊 System Overview")
	
	content := fmt.Sprintf(
		"Claude Processes: %d active\n"+
		"Tmux Sessions: %d running\n"+
		"Git Worktrees: %d tracked\n"+
		"Last Updated: %s",
		status.ActiveProcesses,
		status.ActiveSessions,
		status.TrackedWorktrees,
		status.LastUpdate.Format("15:04:05"),
	)
	
	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		m.theme.ContentStyle.Render(content),
	)
}

func (m *DashboardModel) renderActiveSessions() string {
	title := m.theme.TitleStyle.Render("🖥️  Active Sessions")
	
	sessions := m.integration.GetActiveSessions()
	if len(sessions) == 0 {
		return lipgloss.JoinVertical(lipgloss.Left,
			title,
			m.theme.ContentStyle.Render("No active sessions"),
		)
	}
	
	var sessionLines []string
	for _, session := range sessions {
		line := fmt.Sprintf("• %s (%s) - %s",
			session.Name,
			session.Project,
			session.Branch,
		)
		sessionLines = append(sessionLines, line)
	}
	
	content := strings.Join(sessionLines, "\n")
	
	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		m.theme.ContentStyle.Render(content),
	)
}

func (m *DashboardModel) renderRecentWorktrees() string {
	title := m.theme.TitleStyle.Render("🌳 Recent Worktrees")
	
	worktrees := m.integration.GetRecentWorktrees()
	if len(worktrees) == 0 {
		return lipgloss.JoinVertical(lipgloss.Left,
			title,
			m.theme.ContentStyle.Render("No recent worktrees"),
		)
	}
	
	var worktreeLines []string
	for _, wt := range worktrees {
		line := fmt.Sprintf("• %s (%s) - %s",
			wt.Path,
			wt.Branch,
			wt.LastAccess.Format("Jan 2 15:04"),
		)
		worktreeLines = append(worktreeLines, line)
	}
	
	content := strings.Join(worktreeLines, "\n")
	
	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		m.theme.ContentStyle.Render(content),
	)
}

func (m *DashboardModel) renderQuickActions() string {
	title := m.theme.TitleStyle.Render("⚡ Quick Actions")
	
	actions := []string{
		"n: New session",
		"w: New worktree",  
		"r: Refresh data",
		"c: Configuration",
	}
	
	content := strings.Join(actions, "\n")
	
	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		m.theme.ContentStyle.Render(content),
	)
}

// SessionsModel represents the sessions management screen
type SessionsModel struct {
	integration *Integration
	theme       Theme
	width       int
	height      int
	cursor      int
	sessions    []SessionInfo
}

func NewSessionsModel(integration *Integration, theme Theme) *SessionsModel {
	return &SessionsModel{
		integration: integration,
		theme:       theme,
	}
}

func (m *SessionsModel) Init() tea.Cmd {
	return nil
}

func (m *SessionsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.sessions)-1 {
				m.cursor++
			}
		case "enter":
			// Attach to selected session
			if m.cursor < len(m.sessions) {
				session := m.sessions[m.cursor]
				return m, m.integration.AttachSession(session.ID)
			}
		}
	case RefreshDataMsg:
		m.sessions = m.integration.GetAllSessions()
	}
	return m, nil
}

func (m *SessionsModel) View() string {
	if m.width == 0 {
		return "Loading sessions..."
	}

	header := m.theme.HeaderStyle.Render("🖥️  Session Management")
	
	if len(m.sessions) == 0 {
		return lipgloss.JoinVertical(lipgloss.Left,
			header,
			"",
			m.theme.ContentStyle.Render("No sessions found"),
		)
	}

	var sessionLines []string
	for i, session := range m.sessions {
		cursor := " "
		if i == m.cursor {
			cursor = ">"
		}
		
		status := "●"
		statusColor := m.theme.Success
		if !session.Active {
			status = "○"
			statusColor = m.theme.Muted
		}
		
		line := fmt.Sprintf("%s %s %s (%s) - %s - %s",
			cursor,
			lipgloss.NewStyle().Foreground(statusColor).Render(status),
			session.Name,
			session.Project,
			session.Branch,
			session.Directory,
		)
		sessionLines = append(sessionLines, line)
	}
	
	content := strings.Join(sessionLines, "\n")
	
	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		m.theme.ContentStyle.Render(content),
	)
}

func (m *SessionsModel) Title() string {
	return "Sessions"
}

func (m *SessionsModel) Help() []string {
	return []string{
		"↑/k: Move up",
		"↓/j: Move down",
		"Enter: Attach session",
		"n: New session",
	}
}

// WorktreesModel represents the worktrees management screen
type WorktreesModel struct {
	integration *Integration
	theme       Theme
	width       int
	height      int
	cursor      int
	worktrees   []WorktreeInfo
}

func NewWorktreesModel(integration *Integration, theme Theme) *WorktreesModel {
	return &WorktreesModel{
		integration: integration,
		theme:       theme,
	}
}

func (m *WorktreesModel) Init() tea.Cmd {
	return nil
}

func (m *WorktreesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.worktrees)-1 {
				m.cursor++
			}
		case "enter":
			// Open selected worktree
			if m.cursor < len(m.worktrees) {
				worktree := m.worktrees[m.cursor]
				return m, m.integration.OpenWorktree(worktree.Path)
			}
		}
	case RefreshDataMsg:
		m.worktrees = m.integration.GetAllWorktrees()
	}
	return m, nil
}

func (m *WorktreesModel) View() string {
	if m.width == 0 {
		return "Loading worktrees..."
	}

	header := m.theme.HeaderStyle.Render("🌳 Worktree Management")
	
	if len(m.worktrees) == 0 {
		return lipgloss.JoinVertical(lipgloss.Left,
			header,
			"",
			m.theme.ContentStyle.Render("No worktrees found"),
		)
	}

	var worktreeLines []string
	for i, wt := range m.worktrees {
		cursor := " "
		if i == m.cursor {
			cursor = ">"
		}
		
		line := fmt.Sprintf("%s %s (%s) - %s",
			cursor,
			wt.Path,
			wt.Branch,
			wt.LastAccess.Format("Jan 2 15:04"),
		)
		worktreeLines = append(worktreeLines, line)
	}
	
	content := strings.Join(worktreeLines, "\n")
	
	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		m.theme.ContentStyle.Render(content),
	)
}

func (m *WorktreesModel) Title() string {
	return "Worktrees"
}

func (m *WorktreesModel) Help() []string {
	return []string{
		"↑/k: Move up",
		"↓/j: Move down",
		"Enter: Open worktree",
		"n: New worktree",
	}
}

// ConfigModel represents the configuration screen
type ConfigModel struct {
	config *config.Config
	theme  Theme
	width  int
	height int
}

func NewConfigModel(config *config.Config, theme Theme) *ConfigModel {
	return &ConfigModel{
		config: config,
		theme:  theme,
	}
}

func (m *ConfigModel) Init() tea.Cmd {
	return nil
}

func (m *ConfigModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m *ConfigModel) View() string {
	if m.width == 0 {
		return "Loading configuration..."
	}

	header := m.theme.HeaderStyle.Render("⚙️  Configuration")
	
	content := fmt.Sprintf(
		"Config File: %s\n"+
		"Log Level: %s\n"+
		"Claude Enabled: %t\n"+
		"TUI Theme: %s\n"+
		"Auto Refresh: %ds",
		m.config.ConfigFile,
		m.config.LogLevel,
		m.config.Claude.Enabled,
		m.config.TUI.Theme,
		m.config.RefreshInterval,
	)
	
	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		m.theme.ContentStyle.Render(content),
	)
}

func (m *ConfigModel) Title() string {
	return "Configuration"
}

func (m *ConfigModel) Help() []string {
	return []string{
		"e: Edit config",
		"r: Reload config",
	}
}

// HelpModel represents the help screen
type HelpModel struct {
	theme  Theme
	width  int
	height int
}

func NewHelpModel(theme Theme) *HelpModel {
	return &HelpModel{
		theme: theme,
	}
}

func (m *HelpModel) Init() tea.Cmd {
	return nil
}

func (m *HelpModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m *HelpModel) View() string {
	if m.width == 0 {
		return "Loading help..."
	}

	header := m.theme.HeaderStyle.Render("❓ Help & Keyboard Shortcuts")
	
	sections := []string{
		m.theme.TitleStyle.Render("Global Navigation:"),
		"1: Dashboard",
		"2: Sessions",
		"3: Worktrees", 
		"4: Configuration",
		"?: Help (this screen)",
		"q, Ctrl+C: Quit",
		"",
		m.theme.TitleStyle.Render("Dashboard:"),
		"r: Refresh data",
		"n: New session",
		"w: New worktree",
		"",
		m.theme.TitleStyle.Render("Sessions:"),
		"↑/k, ↓/j: Navigate",
		"Enter: Attach to session",
		"n: Create new session",
		"d: Delete session", 
		"",
		m.theme.TitleStyle.Render("Worktrees:"),
		"↑/k, ↓/j: Navigate",
		"Enter: Open worktree",
		"n: Create new worktree",
		"d: Delete worktree",
	}
	
	content := strings.Join(sections, "\n")
	
	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		m.theme.ContentStyle.Render(content),
	)
}

func (m *HelpModel) Title() string {
	return "Help"
}

func (m *HelpModel) Help() []string {
	return []string{
		"Navigate with numbers 1-4",
		"q: Return to previous screen",
	}
}