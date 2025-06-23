package modals

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ErrorModal represents an error dialog with recovery options
type ErrorModal struct {
	BaseModal
	message        string
	errorDetails   string
	showDetails    bool
	actions        []ErrorAction
	selectedAction int
}

// ErrorAction represents a recovery action for an error
type ErrorAction struct {
	Label   string
	Action  string
	Primary bool
}

// ErrorModalConfig configures an error modal
type ErrorModalConfig struct {
	Title        string
	Message      string
	ErrorDetails string
	Actions      []ErrorAction
}

// NewErrorModal creates a new error modal
func NewErrorModal(config ErrorModalConfig) *ErrorModal {
	actions := config.Actions
	if len(actions) == 0 {
		actions = []ErrorAction{
			{Label: "OK", Action: "ok", Primary: true},
		}
	}

	return &ErrorModal{
		BaseModal:    NewBaseModal(config.Title, 50, 12),
		message:      config.Message,
		errorDetails: config.ErrorDetails,
		actions:      actions,
	}
}

// Init implements the tea.Model interface
func (m *ErrorModal) Init() tea.Cmd {
	return nil
}

// Update implements the tea.Model interface
func (m *ErrorModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.HandleKeyMsg(msg)
	}
	return m, nil
}

// HandleKeyMsg implements the Modal interface
func (m *ErrorModal) HandleKeyMsg(msg tea.KeyMsg) (Modal, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc":
		m.MarkComplete("cancel")
		return m, nil

	case "enter", " ":
		if m.selectedAction < len(m.actions) {
			action := m.actions[m.selectedAction]
			m.MarkComplete(action.Action)
		} else {
			m.MarkComplete("ok")
		}
		return m, nil

	case "left", "h":
		if m.selectedAction > 0 {
			m.selectedAction--
		}

	case "right", "l":
		if m.selectedAction < len(m.actions)-1 {
			m.selectedAction++
		}

	case "tab":
		m.selectedAction = (m.selectedAction + 1) % len(m.actions)

	case "d":
		if m.errorDetails != "" {
			m.showDetails = !m.showDetails
		}

	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		// Quick action selection
		idx := int(msg.Runes[0] - '1')
		if idx < len(m.actions) {
			action := m.actions[idx]
			m.MarkComplete(action.Action)
			return m, nil
		}
	}

	return m, nil
}

// View implements the tea.Model interface
func (m *ErrorModal) View() string {
	var elements []string

	// Error icon and message
	errorStyle := lipgloss.NewStyle().
		Foreground(m.theme.Error).
		Bold(true)

	messageStyle := m.theme.ContentStyle.Copy()

	icon := errorStyle.Render("❌")
	message := messageStyle.Render(m.message)

	errorLine := lipgloss.JoinHorizontal(lipgloss.Left, icon, " ", message)
	elements = append(elements, errorLine)

	// Error details if available and shown
	if m.errorDetails != "" && m.showDetails {
		detailsStyle := lipgloss.NewStyle().
			Foreground(m.theme.Muted).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(m.theme.Muted).
			Padding(1, 2).
			Width(m.width - 12).
			Height(6)

		details := detailsStyle.Render(m.errorDetails)
		elements = append(elements, "", details)
	}

	// Action buttons
	var buttons []string
	for i, action := range m.actions {
		buttonStyle := lipgloss.NewStyle().
			Padding(0, 2).
			Border(lipgloss.RoundedBorder())

		if i == m.selectedAction {
			if action.Primary {
				buttonStyle = buttonStyle.
					Background(m.theme.Error).
					BorderForeground(m.theme.Error).
					Foreground(lipgloss.Color("#FFFFFF")).
					Bold(true)
			} else {
				buttonStyle = buttonStyle.
					Background(m.theme.Accent).
					BorderForeground(m.theme.Accent).
					Foreground(lipgloss.Color("#FFFFFF")).
					Bold(true)
			}
		} else {
			buttonStyle = buttonStyle.
				BorderForeground(m.theme.Muted).
				Foreground(m.theme.Muted)
		}

		label := action.Label
		if i < 9 {
			label = lipgloss.JoinHorizontal(lipgloss.Left,
				lipgloss.NewStyle().Foreground(m.theme.Accent).Render(string(rune('1'+i))),
				". ", action.Label)
		}

		buttons = append(buttons, buttonStyle.Render(label))
	}

	buttonRow := lipgloss.JoinHorizontal(lipgloss.Center, buttons...)
	elements = append(elements, "", buttonRow)

	// Help text
	helpLines := []string{
		"← → Tab: Select",
		"Enter/Space: Confirm",
		"1-9: Quick select",
		"Esc: Cancel",
	}

	if m.errorDetails != "" {
		helpLines = append(helpLines, "D: Toggle details")
	}

	helpStyle := lipgloss.NewStyle().
		Foreground(m.theme.Muted).
		Italic(true)

	help := helpStyle.Render(strings.Join(helpLines, " • "))
	elements = append(elements, "", help)

	content := strings.Join(elements, "\n")
	return m.RenderWithBorder(content)
}

// Common error modal constructors

// NewSimpleErrorModal creates a basic error modal with just an OK button
func NewSimpleErrorModal(title, message string) *ErrorModal {
	return NewErrorModal(ErrorModalConfig{
		Title:   title,
		Message: message,
		Actions: []ErrorAction{
			{Label: "OK", Action: "ok", Primary: true},
		},
	})
}

// NewRetryErrorModal creates an error modal with retry and cancel options
func NewRetryErrorModal(title, message string) *ErrorModal {
	return NewErrorModal(ErrorModalConfig{
		Title:   title,
		Message: message,
		Actions: []ErrorAction{
			{Label: "Retry", Action: "retry", Primary: true},
			{Label: "Cancel", Action: "cancel", Primary: false},
		},
	})
}

// NewDetailedErrorModal creates an error modal with detailed error information
func NewDetailedErrorModal(title, message, details string) *ErrorModal {
	return NewErrorModal(ErrorModalConfig{
		Title:        title,
		Message:      message,
		ErrorDetails: details,
		Actions: []ErrorAction{
			{Label: "OK", Action: "ok", Primary: true},
		},
	})
}
