package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/bcdekker/ccmgr-ultra/internal/config"
)

// ConfigScreen interface for all configuration screens
type ConfigScreen interface {
	Screen
	HasUnsavedChanges() bool
	Save() error
	Cancel()
	Reset()
	GetConfig() interface{}
}

// ConfigMenuItem represents a menu item in the configuration menu
type ConfigMenuItem struct {
	Title       string
	Description string
	Icon        string
	Screen      ConfigScreen
}

// ConfigMenuModel represents the main configuration menu
type ConfigMenuModel struct {
	config         *config.Config
	theme          Theme
	width          int
	height         int
	cursor         int
	menuItems      []ConfigMenuItem
	currentScreen  ConfigScreen
	unsavedChanges bool
	showingScreen  bool
	err            error
}

// NewConfigMenuModel creates a new configuration menu model
func NewConfigMenuModel(cfg *config.Config, theme Theme) *ConfigMenuModel {
	m := &ConfigMenuModel{
		config:        cfg,
		theme:         theme,
		cursor:        0,
		showingScreen: false,
	}

	// Initialize menu items
	m.menuItems = []ConfigMenuItem{
		{
			Title:       "Status Hooks",
			Description: "Configure Claude state change notifications",
			Icon:        "🔔",
			Screen:      NewStatusHooksConfigModel(&cfg.StatusHooks, theme),
		},
		{
			Title:       "Worktree Hooks",
			Description: "Lifecycle event scripts",
			Icon:        "🌳",
			Screen:      NewWorktreeHooksConfigModel(&cfg.WorktreeHooks, theme),
		},
		{
			Title:       "Shortcuts",
			Description: "Keyboard shortcut customization",
			Icon:        "⌨️",
			Screen:      NewShortcutsConfigModel(cfg.Shortcuts, theme),
		},
		{
			Title:       "Worktree Settings",
			Description: "Directory patterns and defaults",
			Icon:        "📁",
			Screen:      NewWorktreeSettingsModel(&cfg.Worktree, theme),
		},
		{
			Title:       "Commands",
			Description: "External command configuration",
			Icon:        "🖥️",
			Screen:      NewCommandsConfigModel(&cfg.Commands, theme),
		},
		{
			Title:       "TUI Settings",
			Description: "Interface preferences",
			Icon:        "🎨",
			Screen:      NewTUISettingsModel(&cfg.TUI, theme),
		},
		{
			Title:       "Git Settings",
			Description: "Repository and branch configuration",
			Icon:        "🔧",
			Screen:      NewGitSettingsModel(&cfg.Git, theme),
		},
		{
			Title:       "Tmux Settings",
			Description: "Session management options",
			Icon:        "🪟",
			Screen:      NewTmuxSettingsModel(&cfg.Tmux, theme),
		},
		{
			Title:       "Claude Settings",
			Description: "Process monitoring configuration",
			Icon:        "🤖",
			Screen:      NewClaudeSettingsModel(&cfg.Claude, theme),
		},
	}

	return m
}

func (m *ConfigMenuModel) Init() tea.Cmd {
	return nil
}

func (m *ConfigMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// If we're showing a sub-screen, delegate to it
	if m.showingScreen && m.currentScreen != nil {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.String() == "esc" {
				// Check for unsaved changes
				if m.currentScreen.HasUnsavedChanges() {
					// TODO: Show confirmation dialog
					m.currentScreen.Cancel()
				}
				m.showingScreen = false
				m.currentScreen = nil
				return m, nil
			}
		}

		// Delegate to the current screen
		newScreen, cmd := m.currentScreen.Update(msg)
		m.currentScreen = newScreen.(ConfigScreen)
		return m, cmd
	}

	// Handle menu navigation
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Update all screens with new size
		for _, item := range m.menuItems {
			item.Screen.Update(msg)
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.menuItems)-1 {
				m.cursor++
			}
		case "enter":
			if m.cursor < len(m.menuItems) {
				m.currentScreen = m.menuItems[m.cursor].Screen
				m.showingScreen = true
			}
		case "s":
			// Save all changes
			return m, m.saveAll()
		case "r":
			// Reset all changes
			return m, m.resetAll()
		}
	}

	return m, nil
}

func (m *ConfigMenuModel) View() string {
	if m.width == 0 {
		return "Loading configuration..."
	}

	// If showing a sub-screen, display it
	if m.showingScreen && m.currentScreen != nil {
		return m.currentScreen.View()
	}

	// Build the main menu
	header := m.theme.HeaderStyle.Render("⚙️  Configuration Menu")

	// Create breadcrumb
	breadcrumb := m.theme.MutedStyle.Render("Configuration")

	// Menu description
	description := m.theme.ContentStyle.Render(
		"Select a configuration category to modify settings. " +
			"Press Enter to edit, s to save all changes, r to reset all.",
	)

	// Build menu items
	var menuLines []string
	for i, item := range m.menuItems {
		cursor := "  "
		if i == m.cursor {
			cursor = "▶ "
		}

		// Check if this item has unsaved changes
		changeIndicator := ""
		if item.Screen.HasUnsavedChanges() {
			changeIndicator = " *"
		}

		line := fmt.Sprintf("%s%s %s%s",
			cursor,
			item.Icon,
			item.Title,
			changeIndicator,
		)

		if i == m.cursor {
			// Add description for selected item
			line = m.theme.SelectedStyle.Render(line)
			menuLines = append(menuLines, line)
			menuLines = append(menuLines, m.theme.MutedStyle.Render(
				"    "+item.Description,
			))
		} else {
			menuLines = append(menuLines, line)
		}
	}

	menuContent := strings.Join(menuLines, "\n")

	// Error display
	errorMsg := ""
	if m.err != nil {
		errorMsg = m.theme.ErrorStyle.Render("Error: " + m.err.Error())
	}

	// Status bar
	statusText := "Navigate: ↑↓/jk | Select: Enter | Save All: s | Reset All: r | Back: Esc"
	if m.hasAnyUnsavedChanges() {
		statusText = "⚠️  Unsaved Changes | " + statusText
	}
	statusBar := m.theme.StatusStyle.Render(statusText)

	// Compose the view
	sections := []string{
		header,
		breadcrumb,
		"",
		description,
		"",
		menuContent,
	}

	if errorMsg != "" {
		sections = append(sections, "", errorMsg)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)

	// Calculate available height for content
	contentHeight := m.height - 3 // Account for status bar
	if contentHeight < 0 {
		contentHeight = 0
	}

	// Create a fixed-height box for content
	contentBox := lipgloss.NewStyle().
		Width(m.width).
		Height(contentHeight).
		Render(content)

	// Add status bar at bottom
	return lipgloss.JoinVertical(
		lipgloss.Left,
		contentBox,
		statusBar,
	)
}

func (m *ConfigMenuModel) Title() string {
	if m.showingScreen && m.currentScreen != nil {
		return "Configuration > " + m.currentScreen.Title()
	}
	return "Configuration"
}

func (m *ConfigMenuModel) Help() []string {
	if m.showingScreen && m.currentScreen != nil {
		return m.currentScreen.Help()
	}
	return []string{
		"↑/k: Move up",
		"↓/j: Move down",
		"Enter: Select category",
		"s: Save all changes",
		"r: Reset all changes",
		"Esc: Back to main",
	}
}

// Helper methods

func (m *ConfigMenuModel) hasAnyUnsavedChanges() bool {
	for _, item := range m.menuItems {
		if item.Screen.HasUnsavedChanges() {
			return true
		}
	}
	return false
}

func (m *ConfigMenuModel) saveAll() tea.Cmd {
	return func() tea.Msg {
		// Save all screens with changes
		for _, item := range m.menuItems {
			if item.Screen.HasUnsavedChanges() {
				if err := item.Screen.Save(); err != nil {
					return ErrorMsg{Error: err}
				}
			}
		}

		// TODO: Save the config file - implement Save method in config package
		// if err := m.config.Save(); err != nil {
		//	return ConfigErrorMsg{Err: err}
		// }

		return ConfigSavedMsg{}
	}
}

func (m *ConfigMenuModel) resetAll() tea.Cmd {
	return func() tea.Msg {
		// Reset all screens
		for _, item := range m.menuItems {
			item.Screen.Reset()
		}
		return ConfigResetMsg{}
	}
}

// Messages
type ConfigSavedMsg struct{}
type ConfigResetMsg struct{}
type ConfigErrorMsg struct {
	Error error
}