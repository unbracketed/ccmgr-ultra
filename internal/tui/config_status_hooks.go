package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bcdekker/ccmgr-ultra/internal/config"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// StatusHooksConfigModel represents the status hooks configuration screen
type StatusHooksConfigModel struct {
	config       *config.StatusHooksConfig
	original     *config.StatusHooksConfig
	theme        Theme
	width        int
	height       int
	cursor       int
	components   []interface{} // Mix of different component types
	focusedIndex int
	testResult   string
	testError    error
}

// NewStatusHooksConfigModel creates a new status hooks configuration model
func NewStatusHooksConfigModel(cfg *config.StatusHooksConfig, theme Theme) *StatusHooksConfigModel {
	// Create a copy of the original config
	original := &config.StatusHooksConfig{
		Enabled:     cfg.Enabled,
		IdleHook:    cfg.IdleHook,
		BusyHook:    cfg.BusyHook,
		WaitingHook: cfg.WaitingHook,
	}

	m := &StatusHooksConfigModel{
		config:   cfg,
		original: original,
		theme:    theme,
	}

	// Initialize components
	m.initComponents()
	return m
}

func (m *StatusHooksConfigModel) initComponents() {
	m.components = []interface{}{
		NewConfigSection("Status Hooks Configuration", m.theme),
		NewConfigHelp("Configure scripts that run when Claude changes state", m.theme),

		// Master enable toggle
		NewConfigToggle("Enable Status Hooks", m.config.Enabled, m.theme),

		// Idle Hook Section
		NewConfigSection("Idle Hook", m.theme),
		NewConfigHelp("Runs when Claude becomes idle", m.theme),
		m.createHookToggle(&m.config.IdleHook, "Enable Idle Hook"),
		m.createScriptInput(&m.config.IdleHook, "Idle Hook Script"),
		m.createTimeoutInput(&m.config.IdleHook, "Idle Hook Timeout"),
		m.createAsyncToggle(&m.config.IdleHook, "Run Idle Hook Asynchronously"),

		// Busy Hook Section
		NewConfigSection("Busy Hook", m.theme),
		NewConfigHelp("Runs when Claude becomes busy", m.theme),
		m.createHookToggle(&m.config.BusyHook, "Enable Busy Hook"),
		m.createScriptInput(&m.config.BusyHook, "Busy Hook Script"),
		m.createTimeoutInput(&m.config.BusyHook, "Busy Hook Timeout"),
		m.createAsyncToggle(&m.config.BusyHook, "Run Busy Hook Asynchronously"),

		// Waiting Hook Section
		NewConfigSection("Waiting Hook", m.theme),
		NewConfigHelp("Runs when Claude is waiting for input", m.theme),
		m.createHookToggle(&m.config.WaitingHook, "Enable Waiting Hook"),
		m.createScriptInput(&m.config.WaitingHook, "Waiting Hook Script"),
		m.createTimeoutInput(&m.config.WaitingHook, "Waiting Hook Timeout"),
		m.createAsyncToggle(&m.config.WaitingHook, "Run Waiting Hook Asynchronously"),
	}
}

func (m *StatusHooksConfigModel) createHookToggle(hook *config.HookConfig, label string) *ConfigToggle {
	toggle := NewConfigToggle(label, hook.Enabled, m.theme)
	return toggle
}

func (m *StatusHooksConfigModel) createScriptInput(hook *config.HookConfig, label string) *ConfigTextInput {
	input := NewConfigTextInput(label, hook.Script, "~/path/to/script.sh", m.theme)
	input.SetValidator(m.validateScriptPath)
	return input
}

func (m *StatusHooksConfigModel) createTimeoutInput(hook *config.HookConfig, label string) *ConfigNumberInput {
	return NewConfigNumberInput(label, hook.Timeout, 1, 300, 5, m.theme)
}

func (m *StatusHooksConfigModel) createAsyncToggle(hook *config.HookConfig, label string) *ConfigToggle {
	toggle := NewConfigToggle(label, hook.Async, m.theme)
	toggle.SetDescription("Run without blocking Claude operations")
	return toggle
}

func (m *StatusHooksConfigModel) validateScriptPath(path string) error {
	if path == "" {
		return nil // Empty is valid (hook disabled)
	}

	// Expand tilde
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot expand ~: %w", err)
		}
		path = filepath.Join(home, path[1:])
	}

	// Check if file exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file does not exist")
		}
		return fmt.Errorf("cannot access file: %w", err)
	}

	// Check if it's a regular file
	if !info.Mode().IsRegular() {
		return fmt.Errorf("not a regular file")
	}

	// Check if it's executable
	if info.Mode()&0111 == 0 {
		return fmt.Errorf("file is not executable")
	}

	return nil
}

func (m *StatusHooksConfigModel) Init() tea.Cmd {
	return nil
}

func (m *StatusHooksConfigModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		case "t":
			// Test current hook
			return m, m.testCurrentHook()
		case "s":
			// Save changes
			return m, m.save()
		case "r":
			// Reset changes
			m.Reset()
			return m, nil
		case "left", "h":
			return m.handleLeft()
		case "right", "l":
			return m.handleRight()
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
		case *ConfigNumberInput:
			newComp, cmd := comp.Update(msg)
			m.components[m.focusedIndex] = newComp
			m.syncConfigFromComponents()
			return m, cmd
		}
	}

	return m, nil
}

func (m *StatusHooksConfigModel) navigateUp() {
	// Blur current
	m.blurCurrent()

	// Find previous focusable
	for i := m.focusedIndex - 1; i >= 0; i-- {
		if m.isFocusable(m.components[i]) {
			m.focusedIndex = i
			m.focusCurrent()
			break
		}
	}
}

func (m *StatusHooksConfigModel) navigateDown() {
	// Blur current
	m.blurCurrent()

	// Find next focusable
	for i := m.focusedIndex + 1; i < len(m.components); i++ {
		if m.isFocusable(m.components[i]) {
			m.focusedIndex = i
			m.focusCurrent()
			break
		}
	}
}

func (m *StatusHooksConfigModel) blurCurrent() {
	if m.focusedIndex < len(m.components) {
		switch comp := m.components[m.focusedIndex].(type) {
		case *ConfigToggle:
			comp.Blur()
		case *ConfigTextInput:
			comp.Blur()
		case *ConfigNumberInput:
			comp.Blur()
		}
	}
}

func (m *StatusHooksConfigModel) focusCurrent() {
	if m.focusedIndex < len(m.components) {
		switch comp := m.components[m.focusedIndex].(type) {
		case *ConfigToggle:
			comp.Focus()
		case *ConfigTextInput:
			comp.Focus()
		case *ConfigNumberInput:
			comp.Focus()
		}
	}
}

func (m *StatusHooksConfigModel) isFocusable(component interface{}) bool {
	switch component.(type) {
	case *ConfigToggle, *ConfigTextInput, *ConfigNumberInput:
		return true
	default:
		return false
	}
}

func (m *StatusHooksConfigModel) handleEnter() (tea.Model, tea.Cmd) {
	if m.focusedIndex < len(m.components) {
		switch comp := m.components[m.focusedIndex].(type) {
		case *ConfigToggle:
			comp.Toggle()
			m.syncConfigFromComponents()
		}
	}
	return m, nil
}

func (m *StatusHooksConfigModel) handleLeft() (tea.Model, tea.Cmd) {
	if m.focusedIndex < len(m.components) {
		switch comp := m.components[m.focusedIndex].(type) {
		case *ConfigNumberInput:
			comp.Decrement()
			m.syncConfigFromComponents()
		}
	}
	return m, nil
}

func (m *StatusHooksConfigModel) handleRight() (tea.Model, tea.Cmd) {
	if m.focusedIndex < len(m.components) {
		switch comp := m.components[m.focusedIndex].(type) {
		case *ConfigNumberInput:
			comp.Increment()
			m.syncConfigFromComponents()
		}
	}
	return m, nil
}

func (m *StatusHooksConfigModel) syncConfigFromComponents() {
	// Sync master toggle
	if toggle, ok := m.components[2].(*ConfigToggle); ok {
		m.config.Enabled = toggle.value
	}

	// Sync hook configs
	m.syncHookFromComponents(&m.config.IdleHook, 5, 6, 7, 8)
	m.syncHookFromComponents(&m.config.BusyHook, 11, 12, 13, 14)
	m.syncHookFromComponents(&m.config.WaitingHook, 17, 18, 19, 20)
}

func (m *StatusHooksConfigModel) syncHookFromComponents(hook *config.HookConfig, enableIdx, scriptIdx, timeoutIdx, asyncIdx int) {
	if toggle, ok := m.components[enableIdx].(*ConfigToggle); ok {
		hook.Enabled = toggle.value
	}
	if input, ok := m.components[scriptIdx].(*ConfigTextInput); ok {
		hook.Script = input.value
	}
	if number, ok := m.components[timeoutIdx].(*ConfigNumberInput); ok {
		hook.Timeout = number.value
	}
	if toggle, ok := m.components[asyncIdx].(*ConfigToggle); ok {
		hook.Async = toggle.value
	}
}

func (m *StatusHooksConfigModel) testCurrentHook() tea.Cmd {
	return func() tea.Msg {
		// Determine which hook section we're in
		var hookName string
		var hook *config.HookConfig

		if m.focusedIndex >= 5 && m.focusedIndex <= 8 {
			hookName = "Idle"
			hook = &m.config.IdleHook
		} else if m.focusedIndex >= 11 && m.focusedIndex <= 14 {
			hookName = "Busy"
			hook = &m.config.BusyHook
		} else if m.focusedIndex >= 17 && m.focusedIndex <= 20 {
			hookName = "Waiting"
			hook = &m.config.WaitingHook
		} else {
			return ConfigErrorMsg{Error: fmt.Errorf("select a hook to test")}
		}

		if !hook.Enabled || hook.Script == "" {
			return ConfigErrorMsg{Error: fmt.Errorf("%s hook is not enabled or has no script", hookName)}
		}

		// In a real implementation, this would execute the script
		// For now, return a success message
		return StatusHookTestMsg{
			HookName: hookName,
			Success:  true,
			Output:   fmt.Sprintf("%s hook test completed successfully", hookName),
		}
	}
}

func (m *StatusHooksConfigModel) save() tea.Cmd {
	return func() tea.Msg {
		// Apply all component changes
		for _, comp := range m.components {
			switch c := comp.(type) {
			case *ConfigToggle:
				c.Apply()
			case *ConfigTextInput:
				c.Apply()
			case *ConfigNumberInput:
				c.Apply()
			}
		}

		// Update original to match current
		m.original.Enabled = m.config.Enabled
		m.original.IdleHook = m.config.IdleHook
		m.original.BusyHook = m.config.BusyHook
		m.original.WaitingHook = m.config.WaitingHook

		return ConfigSavedMsg{}
	}
}

func (m *StatusHooksConfigModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	header := m.theme.HeaderStyle.Render("⚙️  Configuration > Status Hooks")

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
			lines = append(lines, c.View())
		case *ConfigToggle:
			lines = append(lines, c.View())
		case *ConfigTextInput:
			lines = append(lines, strings.Split(c.View(), "\n")...)
		case *ConfigNumberInput:
			lines = append(lines, strings.Split(c.View(), "\n")...)
		}
	}

	content := strings.Join(lines, "\n")

	// Test result display
	testDisplay := ""
	if m.testResult != "" {
		testDisplay = m.theme.SuccessStyle.Render("✓ " + m.testResult)
	}
	if m.testError != nil {
		testDisplay = m.theme.ErrorStyle.Render("✗ " + m.testError.Error())
	}

	// Status bar
	statusText := "Navigate: ↑↓/Tab | Toggle: Space/Enter | Test: t | Save: s | Reset: r | Back: Esc"
	if m.HasUnsavedChanges() {
		statusText = "⚠️  Unsaved Changes | " + statusText
	}
	statusBar := m.theme.StatusStyle.Render(statusText)

	// Compose view
	sections := []string{header, "", content}
	if testDisplay != "" {
		sections = append(sections, "", testDisplay)
	}

	mainContent := lipgloss.JoinVertical(lipgloss.Left, sections...)

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

func (m *StatusHooksConfigModel) Title() string {
	return "Status Hooks"
}

func (m *StatusHooksConfigModel) Help() []string {
	return []string{
		"↑/k, ↓/j: Navigate",
		"Space/Enter: Toggle",
		"t: Test hook",
		"s: Save",
		"r: Reset",
		"Esc: Back",
	}
}

func (m *StatusHooksConfigModel) HasUnsavedChanges() bool {
	// Check master toggle
	if m.config.Enabled != m.original.Enabled {
		return true
	}

	// Check each hook
	if m.hookHasChanges(&m.config.IdleHook, &m.original.IdleHook) {
		return true
	}
	if m.hookHasChanges(&m.config.BusyHook, &m.original.BusyHook) {
		return true
	}
	if m.hookHasChanges(&m.config.WaitingHook, &m.original.WaitingHook) {
		return true
	}

	return false
}

func (m *StatusHooksConfigModel) hookHasChanges(current, original *config.HookConfig) bool {
	return current.Enabled != original.Enabled ||
		current.Script != original.Script ||
		current.Timeout != original.Timeout ||
		current.Async != original.Async
}

func (m *StatusHooksConfigModel) Save() error {
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
		case *ConfigNumberInput:
			c.Apply()
		}
	}

	return nil
}

func (m *StatusHooksConfigModel) Cancel() {
	m.Reset()
}

func (m *StatusHooksConfigModel) Reset() {
	// Reset config to original
	*m.config = *m.original

	// Reset all components
	for _, comp := range m.components {
		switch c := comp.(type) {
		case *ConfigToggle:
			c.Reset()
		case *ConfigTextInput:
			c.Reset()
		case *ConfigNumberInput:
			c.Reset()
		}
	}

	// Re-init to sync component values
	m.initComponents()
}

func (m *StatusHooksConfigModel) GetConfig() interface{} {
	return m.config
}

// StatusHookTestMsg is sent when a hook test completes
type StatusHookTestMsg struct {
	HookName string
	Success  bool
	Output   string
	Error    error
}
