package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ShortcutsConfigModel represents the shortcuts configuration screen
type ShortcutsConfigModel struct {
	shortcuts      map[string]string
	original       map[string]string
	theme          Theme
	width          int
	height         int
	cursor         int
	adding         bool
	editing        bool
	deleting       bool
	editKey        string
	newKeyInput    textinput.Model
	newActionInput textinput.Model
	err            error
	sortedKeys     []string
}

// Predefined actions that can be assigned to shortcuts
var availableActions = map[string]string{
	"new_worktree":     "Create a new worktree",
	"merge_worktree":   "Merge current worktree",
	"delete_worktree":  "Delete selected worktree",
	"push_worktree":    "Push current worktree",
	"continue_session": "Continue Claude session",
	"resume_session":   "Resume Claude session",
	"new_session":      "Start new Claude session",
	"refresh":          "Refresh current view",
	"quit":             "Quit application",
	"help":             "Show help",
	"toggle_select":    "Toggle selection mode",
	"select_all":       "Select all items",
	"search":           "Search/filter items",
}

// NewShortcutsConfigModel creates a new shortcuts configuration model
func NewShortcutsConfigModel(shortcuts map[string]string, theme Theme) *ShortcutsConfigModel {
	// Create a copy of the original shortcuts
	original := make(map[string]string)
	for k, v := range shortcuts {
		original[k] = v
	}

	// Create text inputs
	keyInput := textinput.New()
	keyInput.CharLimit = 10
	keyInput.Placeholder = "key"

	actionInput := textinput.New()
	actionInput.CharLimit = 50
	actionInput.Placeholder = "action"

	m := &ShortcutsConfigModel{
		shortcuts:      shortcuts,
		original:       original,
		theme:          theme,
		newKeyInput:    keyInput,
		newActionInput: actionInput,
	}

	m.updateSortedKeys()
	return m
}

func (m *ShortcutsConfigModel) updateSortedKeys() {
	m.sortedKeys = make([]string, 0, len(m.shortcuts))
	for k := range m.shortcuts {
		m.sortedKeys = append(m.sortedKeys, k)
	}
	sort.Strings(m.sortedKeys)
}

func (m *ShortcutsConfigModel) Init() tea.Cmd {
	return nil
}

func (m *ShortcutsConfigModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		if m.adding || m.editing {
			return m.handleInputMode(msg)
		}

		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.sortedKeys)-1 {
				m.cursor++
			}
		case "n":
			// Add new shortcut
			m.adding = true
			m.newKeyInput.Focus()
			m.newKeyInput.SetValue("")
			m.newActionInput.SetValue("")
			return m, textinput.Blink
		case "enter", "e":
			// Edit current shortcut
			if m.cursor < len(m.sortedKeys) {
				m.editing = true
				m.editKey = m.sortedKeys[m.cursor]
				m.newKeyInput.SetValue(m.editKey)
				m.newActionInput.SetValue(m.shortcuts[m.editKey])
				m.newKeyInput.Focus()
				return m, textinput.Blink
			}
		case "d":
			// Delete current shortcut
			if m.cursor < len(m.sortedKeys) {
				key := m.sortedKeys[m.cursor]
				delete(m.shortcuts, key)
				m.updateSortedKeys()
				if m.cursor >= len(m.sortedKeys) && m.cursor > 0 {
					m.cursor--
				}
			}
		case "r":
			// Reset to defaults
			m.Reset()
		case "a":
			// Show available actions
			// TODO: Could implement a popup or side panel
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *ShortcutsConfigModel) handleInputMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg.String() {
	case "esc":
		// Cancel input
		m.adding = false
		m.editing = false
		m.newKeyInput.Blur()
		m.newActionInput.Blur()
		m.err = nil
		return m, nil

	case "tab":
		// Switch between key and action input
		if m.newKeyInput.Focused() {
			m.newKeyInput.Blur()
			m.newActionInput.Focus()
			return m, textinput.Blink
		} else {
			m.newActionInput.Blur()
			m.newKeyInput.Focus()
			return m, textinput.Blink
		}

	case "enter":
		// Save the shortcut
		key := strings.TrimSpace(m.newKeyInput.Value())
		action := strings.TrimSpace(m.newActionInput.Value())

		if err := m.validateShortcut(key, action); err != nil {
			m.err = err
			return m, nil
		}

		// If editing, remove the old key if it changed
		if m.editing && m.editKey != key {
			delete(m.shortcuts, m.editKey)
		}

		m.shortcuts[key] = action
		m.updateSortedKeys()

		// Find the new key in sorted list and set cursor
		for i, k := range m.sortedKeys {
			if k == key {
				m.cursor = i
				break
			}
		}

		m.adding = false
		m.editing = false
		m.newKeyInput.Blur()
		m.newActionInput.Blur()
		m.err = nil
		return m, nil
	}

	// Update the appropriate input
	var cmd tea.Cmd
	if m.newKeyInput.Focused() {
		m.newKeyInput, cmd = m.newKeyInput.Update(msg)
		cmds = append(cmds, cmd)
	} else {
		m.newActionInput, cmd = m.newActionInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *ShortcutsConfigModel) validateShortcut(key, action string) error {
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}

	if action == "" {
		return fmt.Errorf("action cannot be empty")
	}

	// Check for reserved keys
	reservedKeys := []string{"?", ":", "/", "q", "ctrl+c"}
	for _, reserved := range reservedKeys {
		if key == reserved {
			return fmt.Errorf("'%s' is a reserved key", key)
		}
	}

	// Check for duplicate key (unless editing the same key)
	if existingAction, exists := m.shortcuts[key]; exists && (!m.editing || m.editKey != key) {
		return fmt.Errorf("key '%s' is already assigned to '%s'", key, existingAction)
	}

	return nil
}

func (m *ShortcutsConfigModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	header := m.theme.HeaderStyle.Render("⚙️  Configuration > Keyboard Shortcuts")

	// Build content
	var lines []string

	// Instructions
	if !m.adding && !m.editing {
		lines = append(lines, m.theme.MutedStyle.Render("Customize keyboard shortcuts for common actions"))
		lines = append(lines, "")
	}

	// Error display
	if m.err != nil {
		lines = append(lines, m.theme.ErrorStyle.Render("Error: "+m.err.Error()))
		lines = append(lines, "")
	}

	// Shortcuts list or input form
	if m.adding || m.editing {
		lines = append(lines, m.renderInputForm())
	} else {
		lines = append(lines, m.renderShortcutsList())
	}

	content := strings.Join(lines, "\n")

	// Status bar
	statusText := ""
	if m.adding {
		statusText = "Adding shortcut | Tab: Switch fields | Enter: Save | Esc: Cancel"
	} else if m.editing {
		statusText = "Editing shortcut | Tab: Switch fields | Enter: Save | Esc: Cancel"
	} else {
		statusText = "Navigate: ↑↓ | Add: n | Edit: e/Enter | Delete: d | Reset: r | Back: Esc"
		if m.HasUnsavedChanges() {
			statusText = "⚠️  Unsaved Changes | " + statusText
		}
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

func (m *ShortcutsConfigModel) renderShortcutsList() string {
	if len(m.shortcuts) == 0 {
		return m.theme.MutedStyle.Render("No shortcuts defined. Press 'n' to add one.")
	}

	var lines []string

	// Header
	headerStyle := m.theme.TitleStyle
	lines = append(lines, fmt.Sprintf(
		"%s  %s  %s",
		headerStyle.Render(padRight("Key", 15)),
		headerStyle.Render(padRight("Action", 30)),
		headerStyle.Render("Description"),
	))
	lines = append(lines, strings.Repeat("─", m.width-10))

	// Shortcuts
	for i, key := range m.sortedKeys {
		action := m.shortcuts[key]
		description := ""
		if desc, ok := availableActions[action]; ok {
			description = desc
		}

		cursor := "  "
		if i == m.cursor {
			cursor = "▶ "
		}

		// Check if modified
		originalAction, existedBefore := m.original[key]
		isModified := !existedBefore || originalAction != action

		keyDisplay := padRight(key, 13)
		actionDisplay := padRight(action, 28)

		if isModified {
			keyDisplay += " *"
		}

		line := fmt.Sprintf("%s%s  %s  %s",
			cursor,
			keyDisplay,
			actionDisplay,
			description,
		)

		if i == m.cursor {
			line = m.theme.SelectedStyle.Render(line)
		}

		lines = append(lines, line)
	}

	// Show deleted shortcuts
	for key, action := range m.original {
		if _, exists := m.shortcuts[key]; !exists {
			line := fmt.Sprintf("  %s  %s  (deleted)",
				m.theme.ErrorStyle.Render(padRight(key, 13)+" ✗"),
				m.theme.MutedStyle.Render(padRight(action, 28)),
			)
			lines = append(lines, line)
		}
	}

	return strings.Join(lines, "\n")
}

func (m *ShortcutsConfigModel) renderInputForm() string {
	title := "Add New Shortcut"
	if m.editing {
		title = fmt.Sprintf("Edit Shortcut: %s", m.editKey)
	}

	var lines []string
	lines = append(lines, m.theme.TitleStyle.Render(title))
	lines = append(lines, "")

	// Key input
	keyLabel := "Key: "
	if m.newKeyInput.Focused() {
		keyLabel = m.theme.FocusedStyle.Render(keyLabel)
	}
	lines = append(lines, keyLabel+m.newKeyInput.View())

	// Action input
	actionLabel := "Action: "
	if m.newActionInput.Focused() {
		actionLabel = m.theme.FocusedStyle.Render(actionLabel)
	}
	lines = append(lines, actionLabel+m.newActionInput.View())

	// Available actions hint
	lines = append(lines, "")
	lines = append(lines, m.theme.MutedStyle.Render("Available actions:"))

	// Show first few available actions
	actionNames := make([]string, 0, len(availableActions))
	for name := range availableActions {
		actionNames = append(actionNames, name)
	}
	sort.Strings(actionNames)

	for i, name := range actionNames {
		if i >= 5 {
			lines = append(lines, m.theme.MutedStyle.Render("  ... and more"))
			break
		}
		lines = append(lines, m.theme.MutedStyle.Render(fmt.Sprintf("  • %s - %s", name, availableActions[name])))
	}

	return strings.Join(lines, "\n")
}

func (m *ShortcutsConfigModel) Title() string {
	return "Keyboard Shortcuts"
}

func (m *ShortcutsConfigModel) Help() []string {
	if m.adding || m.editing {
		return []string{
			"Tab: Switch fields",
			"Enter: Save",
			"Esc: Cancel",
		}
	}
	return []string{
		"↑/k, ↓/j: Navigate",
		"n: Add shortcut",
		"e/Enter: Edit",
		"d: Delete",
		"r: Reset defaults",
		"Esc: Back",
	}
}

func (m *ShortcutsConfigModel) HasUnsavedChanges() bool {
	// Check if any shortcuts were added or removed
	if len(m.shortcuts) != len(m.original) {
		return true
	}

	// Check if any shortcuts were modified
	for key, action := range m.shortcuts {
		if originalAction, exists := m.original[key]; !exists || originalAction != action {
			return true
		}
	}

	return false
}

func (m *ShortcutsConfigModel) Save() error {
	// Copy current to original
	m.original = make(map[string]string)
	for k, v := range m.shortcuts {
		m.original[k] = v
	}
	return nil
}

func (m *ShortcutsConfigModel) Cancel() {
	m.Reset()
}

func (m *ShortcutsConfigModel) Reset() {
	// Reset to original
	m.shortcuts = make(map[string]string)
	for k, v := range m.original {
		m.shortcuts[k] = v
	}
	m.updateSortedKeys()
	m.cursor = 0
	m.adding = false
	m.editing = false
	m.err = nil
}

func (m *ShortcutsConfigModel) GetConfig() interface{} {
	return m.shortcuts
}

// Helper function to pad strings
func padRight(s string, length int) string {
	if len(s) >= length {
		return s[:length]
	}
	return s + strings.Repeat(" ", length-len(s))
}
