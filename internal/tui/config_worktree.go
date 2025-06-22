package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/bcdekker/ccmgr-ultra/internal/config"
)

// WorktreeSettingsModel represents the worktree settings configuration screen
type WorktreeSettingsModel struct {
	config     *config.WorktreeConfig
	original   *config.WorktreeConfig
	theme      Theme
	width      int
	height     int
	cursor     int
	components []interface{}
	focusedIndex int
}

// NewWorktreeSettingsModel creates a new worktree settings configuration model
func NewWorktreeSettingsModel(cfg *config.WorktreeConfig, theme Theme) *WorktreeSettingsModel {
	// Create a copy of the original config
	original := &config.WorktreeConfig{
		AutoDirectory:    cfg.AutoDirectory,
		DirectoryPattern: cfg.DirectoryPattern,
		DefaultBranch:    cfg.DefaultBranch,
		CleanupOnMerge:   cfg.CleanupOnMerge,
	}

	m := &WorktreeSettingsModel{
		config:   cfg,
		original: original,
		theme:    theme,
	}

	m.initComponents()
	return m
}

func (m *WorktreeSettingsModel) initComponents() {
	m.components = []interface{}{
		NewConfigSection("Worktree Settings", m.theme),
		NewConfigHelp("Configure default behavior for worktree creation and management", m.theme),

		// Auto-directory setting
		NewConfigSection("Directory Management", m.theme),
		NewConfigToggle("Auto-create directories", m.config.AutoDirectory, m.theme),
		
		// Directory pattern
		m.createPatternInput(),
		m.createPatternHelp(),
		
		// Default branch
		NewConfigSection("Branch Settings", m.theme),
		m.createDefaultBranchInput(),
		
		// Cleanup settings
		NewConfigSection("Cleanup Options", m.theme),
		NewConfigToggle("Cleanup on merge", m.config.CleanupOnMerge, m.theme),
	}
	
	// Set descriptions for toggles
	if toggle, ok := m.components[3].(*ConfigToggle); ok {
		toggle.SetDescription("Automatically create directory structure when creating worktrees")
	}
	if toggle, ok := m.components[9].(*ConfigToggle); ok {
		toggle.SetDescription("Automatically delete worktree when branch is merged")
	}
}

func (m *WorktreeSettingsModel) createPatternInput() *ConfigTextInput {
	input := NewConfigTextInput(
		"Directory pattern",
		m.config.DirectoryPattern,
		"{{.project}}-{{.branch}}",
		m.theme,
	)
	input.SetValidator(m.validateDirectoryPattern)
	return input
}

func (m *WorktreeSettingsModel) createPatternHelp() *ConfigHelp {
	helpText := `Template variables:
  {{.project}} - Project/repository name
  {{.branch}}  - Branch name (sanitized)
  {{.user}}    - Current user
  {{.date}}    - Current date (YYYY-MM-DD)
  
Examples:
  {{.project}}-{{.branch}}     → myapp-feature-auth
  work/{{.project}}/{{.branch}} → work/myapp/feature-auth
  {{.user}}/{{.project}}-{{.branch}} → john/myapp-feature-auth`
	
	return NewConfigHelp(helpText, m.theme)
}

func (m *WorktreeSettingsModel) createDefaultBranchInput() *ConfigTextInput {
	input := NewConfigTextInput(
		"Default branch",
		m.config.DefaultBranch,
		"main",
		m.theme,
	)
	input.SetValidator(m.validateBranchName)
	return input
}

func (m *WorktreeSettingsModel) validateDirectoryPattern(pattern string) error {
	if pattern == "" {
		return nil // Empty is valid when auto-directory is disabled
	}

	// Check if pattern contains required variables
	if !strings.Contains(pattern, "{{") || !strings.Contains(pattern, "}}") {
		return fmt.Errorf("pattern must contain template variables like {{.project}} or {{.branch}}")
	}

	// Check for forbidden characters
	forbidden := []string{"..", "//", "\\"}
	for _, f := range forbidden {
		if strings.Contains(pattern, f) {
			return fmt.Errorf("pattern cannot contain '%s'", f)
		}
	}

	return nil
}

func (m *WorktreeSettingsModel) validateBranchName(branch string) error {
	if branch == "" {
		return fmt.Errorf("default branch cannot be empty")
	}

	// Check for invalid characters
	invalid := []string{" ", "..", "~", "^", ":", "?", "*", "[", "\\"}
	for _, inv := range invalid {
		if strings.Contains(branch, inv) {
			return fmt.Errorf("branch name cannot contain '%s'", inv)
		}
	}

	return nil
}

func (m *WorktreeSettingsModel) Init() tea.Cmd {
	return nil
}

func (m *WorktreeSettingsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k", "shift+tab":
			m.navigateUp()
		case "down", "j", "tab":
			m.navigateDown()
		case "enter", " ":
			return m.handleEnter()
		case "s":
			// Save changes
			return m, m.save()
		case "r":
			// Reset changes
			m.Reset()
			return m, nil
		case "p":
			// Preview directory pattern
			return m, m.previewPattern()
		}
	}

	// Update focused component
	if m.focusedIndex < len(m.components) {
		switch comp := m.components[m.focusedIndex].(type) {
		case *ConfigTextInput:
			newComp, cmd := comp.Update(msg)
			m.components[m.focusedIndex] = newComp
			m.syncConfigFromComponents()
			return m, cmd
		}
	}

	return m, nil
}

func (m *WorktreeSettingsModel) navigateUp() {
	m.blurCurrent()
	for i := m.focusedIndex - 1; i >= 0; i-- {
		if m.isFocusable(m.components[i]) {
			m.focusedIndex = i
			m.focusCurrent()
			break
		}
	}
}

func (m *WorktreeSettingsModel) navigateDown() {
	m.blurCurrent()
	for i := m.focusedIndex + 1; i < len(m.components); i++ {
		if m.isFocusable(m.components[i]) {
			m.focusedIndex = i
			m.focusCurrent()
			break
		}
	}
}

func (m *WorktreeSettingsModel) blurCurrent() {
	if m.focusedIndex < len(m.components) {
		switch comp := m.components[m.focusedIndex].(type) {
		case *ConfigToggle:
			comp.Blur()
		case *ConfigTextInput:
			comp.Blur()
		}
	}
}

func (m *WorktreeSettingsModel) focusCurrent() {
	if m.focusedIndex < len(m.components) {
		switch comp := m.components[m.focusedIndex].(type) {
		case *ConfigToggle:
			comp.Focus()
		case *ConfigTextInput:
			comp.Focus()
		}
	}
}

func (m *WorktreeSettingsModel) isFocusable(component interface{}) bool {
	switch component.(type) {
	case *ConfigToggle, *ConfigTextInput:
		return true
	default:
		return false
	}
}

func (m *WorktreeSettingsModel) handleEnter() (tea.Model, tea.Cmd) {
	if m.focusedIndex < len(m.components) {
		switch comp := m.components[m.focusedIndex].(type) {
		case *ConfigToggle:
			comp.Toggle()
			m.syncConfigFromComponents()
		}
	}
	return m, nil
}

func (m *WorktreeSettingsModel) syncConfigFromComponents() {
	// Auto-directory toggle
	if toggle, ok := m.components[3].(*ConfigToggle); ok {
		m.config.AutoDirectory = toggle.value
	}

	// Directory pattern
	if input, ok := m.components[4].(*ConfigTextInput); ok {
		m.config.DirectoryPattern = input.value
	}

	// Default branch
	if input, ok := m.components[7].(*ConfigTextInput); ok {
		m.config.DefaultBranch = input.value
	}

	// Cleanup on merge toggle
	if toggle, ok := m.components[9].(*ConfigToggle); ok {
		m.config.CleanupOnMerge = toggle.value
	}
}

func (m *WorktreeSettingsModel) previewPattern() tea.Cmd {
	return func() tea.Msg {
		// Generate preview of directory pattern
		pattern := m.config.DirectoryPattern
		if pattern == "" {
			return WorktreePatternPreviewMsg{
				Pattern: pattern,
				Preview: "(no pattern - directories created manually)",
			}
		}

		// Replace template variables with example values
		preview := pattern
		preview = strings.ReplaceAll(preview, "{{.project}}", "myapp")
		preview = strings.ReplaceAll(preview, "{{.branch}}", "feature-auth")
		preview = strings.ReplaceAll(preview, "{{.user}}", "john")
		preview = strings.ReplaceAll(preview, "{{.date}}", "2024-01-15")

		return WorktreePatternPreviewMsg{
			Pattern: pattern,
			Preview: preview,
		}
	}
}

func (m *WorktreeSettingsModel) save() tea.Cmd {
	return func() tea.Msg {
		// Apply all component changes
		for _, comp := range m.components {
			switch c := comp.(type) {
			case *ConfigToggle:
				c.Apply()
			case *ConfigTextInput:
				c.Apply()
			}
		}

		// Update original to match current
		*m.original = *m.config

		return ConfigSavedMsg{}
	}
}

func (m *WorktreeSettingsModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	header := m.theme.HeaderStyle.Render("⚙️  Configuration > Worktree Settings")

	// Build component views
	var lines []string
	for _, comp := range m.components {
		switch c := comp.(type) {
		case *ConfigSection:
			if len(lines) > 0 {
				lines = append(lines, "") // Add spacing before sections
			}
			lines = append(lines, c.View())
		case *ConfigHelp:
			// Split multi-line help text
			helpLines := strings.Split(c.View(), "\n")
			lines = append(lines, helpLines...)
		case *ConfigToggle:
			lines = append(lines, c.View())
		case *ConfigTextInput:
			inputLines := strings.Split(c.View(), "\n")
			lines = append(lines, inputLines...)
		}
	}

	content := strings.Join(lines, "\n")

	// Status bar
	statusText := "Navigate: ↑↓/Tab | Toggle: Space/Enter | Preview: p | Save: s | Reset: r | Back: Esc"
	if m.HasUnsavedChanges() {
		statusText = "⚠️  Unsaved Changes | " + statusText
	}
	statusBar := m.theme.StatusStyle.Render(statusText)

	// Compose view
	mainContent := lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		content,
	)

	// Fixed height layout
	contentHeight := m.height - 3
	if contentHeight < 0 {
		contentHeight = 0
	}

	contentBox := lipgloss.NewStyle().
		Width(m.width).
		Height(contentHeight).
		Render(mainContent)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		contentBox,
		statusBar,
	)
}

func (m *WorktreeSettingsModel) Title() string {
	return "Worktree Settings"
}

func (m *WorktreeSettingsModel) Help() []string {
	return []string{
		"↑/k, ↓/j: Navigate",
		"Space/Enter: Toggle",
		"p: Preview pattern",
		"s: Save",
		"r: Reset",
		"Esc: Back",
	}
}

func (m *WorktreeSettingsModel) HasUnsavedChanges() bool {
	return m.config.AutoDirectory != m.original.AutoDirectory ||
		m.config.DirectoryPattern != m.original.DirectoryPattern ||
		m.config.DefaultBranch != m.original.DefaultBranch ||
		m.config.CleanupOnMerge != m.original.CleanupOnMerge
}

func (m *WorktreeSettingsModel) Save() error {
	// Validate before saving
	if err := m.config.Validate(); err != nil {
		return err
	}

	// Update original to match current
	*m.original = *m.config

	// Mark all components as saved
	for _, comp := range m.components {
		switch c := comp.(type) {
		case *ConfigToggle:
			c.Apply()
		case *ConfigTextInput:
			c.Apply()
		}
	}

	return nil
}

func (m *WorktreeSettingsModel) Cancel() {
	m.Reset()
}

func (m *WorktreeSettingsModel) Reset() {
	// Reset config to original
	*m.config = *m.original

	// Reset all components
	for _, comp := range m.components {
		switch c := comp.(type) {
		case *ConfigToggle:
			c.Reset()
		case *ConfigTextInput:
			c.Reset()
		}
	}

	// Re-init to sync component values
	m.initComponents()
}

func (m *WorktreeSettingsModel) GetConfig() interface{} {
	return m.config
}

// WorktreePatternPreviewMsg is sent when pattern preview is requested
type WorktreePatternPreviewMsg struct {
	Pattern string
	Preview string
}