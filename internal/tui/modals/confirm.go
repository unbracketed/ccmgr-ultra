package modals

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ConfirmModal represents a confirmation dialog
type ConfirmModal struct {
	BaseModal
	message        string
	confirmText    string
	cancelText     string
	defaultConfirm bool
	selected       bool
	dangerMode     bool
}

// ConfirmModalConfig configures a confirmation modal
type ConfirmModalConfig struct {
	Title          string
	Message        string
	ConfirmText    string
	CancelText     string
	DefaultConfirm bool
	DangerMode     bool // Use danger styling for destructive actions
}

// NewConfirmModal creates a new confirmation modal
func NewConfirmModal(config ConfirmModalConfig) *ConfirmModal {
	confirmText := config.ConfirmText
	if confirmText == "" {
		confirmText = "Yes"
	}

	cancelText := config.CancelText
	if cancelText == "" {
		cancelText = "No"
	}

	return &ConfirmModal{
		BaseModal:      NewBaseModal(config.Title, 40, 10),
		message:        config.Message,
		confirmText:    confirmText,
		cancelText:     cancelText,
		defaultConfirm: config.DefaultConfirm,
		selected:       config.DefaultConfirm,
		dangerMode:     config.DangerMode,
	}
}

// Init implements the tea.Model interface
func (m *ConfirmModal) Init() tea.Cmd {
	return nil
}

// Update implements the tea.Model interface
func (m *ConfirmModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.HandleKeyMsg(msg)
	}
	return m, nil
}

// HandleKeyMsg implements the Modal interface
func (m *ConfirmModal) HandleKeyMsg(msg tea.KeyMsg) (Modal, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc":
		m.MarkComplete(false)
		return m, nil

	case "enter", " ":
		m.MarkComplete(m.selected)
		return m, nil

	case "y", "Y":
		m.MarkComplete(true)
		return m, nil

	case "n", "N":
		m.MarkComplete(false)
		return m, nil

	case "left", "h":
		m.selected = false

	case "right", "l":
		m.selected = true

	case "tab":
		m.selected = !m.selected
	}

	return m, nil
}

// View implements the tea.Model interface
func (m *ConfirmModal) View() string {
	// Message styling
	messageStyle := m.theme.ContentStyle.Copy()
	if m.dangerMode {
		messageStyle = messageStyle.Foreground(m.theme.Warning)
	}

	// Wrap message text
	messageLines := strings.Split(m.message, "\n")
	message := messageStyle.Render(strings.Join(messageLines, "\n"))

	// Button styling
	confirmButtonStyle := lipgloss.NewStyle().
		Padding(0, 3).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Accent)

	cancelButtonStyle := lipgloss.NewStyle().
		Padding(0, 3).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Muted)

	// Apply selection styling
	var confirmButton, cancelButton string
	if m.selected {
		// Confirm button selected
		confirmStyle := confirmButtonStyle.Copy()
		if m.dangerMode {
			confirmStyle = confirmStyle.
				Background(m.theme.Error).
				BorderForeground(m.theme.Error).
				Foreground(lipgloss.Color("#FFFFFF")).
				Bold(true)
		} else {
			confirmStyle = confirmStyle.
				Background(m.theme.Success).
				BorderForeground(m.theme.Success).
				Foreground(lipgloss.Color("#FFFFFF")).
				Bold(true)
		}
		confirmButton = confirmStyle.Render(m.confirmText)

		cancelStyle := cancelButtonStyle.Copy().
			Foreground(m.theme.Muted)
		cancelButton = cancelStyle.Render(m.cancelText)
	} else {
		// Cancel button selected
		confirmStyle := confirmButtonStyle.Copy().
			Foreground(m.theme.Muted)
		confirmButton = confirmStyle.Render(m.confirmText)

		cancelStyle := cancelButtonStyle.Copy().
			Background(m.theme.Accent).
			BorderForeground(m.theme.Accent).
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true)
		cancelButton = cancelStyle.Render(m.cancelText)
	}

	// Button layout
	buttons := lipgloss.JoinHorizontal(lipgloss.Center,
		confirmButton,
		"   ",
		cancelButton,
	)

	// Help text
	helpStyle := lipgloss.NewStyle().
		Foreground(m.theme.Muted).
		Italic(true)

	help := helpStyle.Render("← → Tab: Select • Enter/Space: Confirm • Y/N: Quick choice • Esc: Cancel")

	// Combine all elements
	content := lipgloss.JoinVertical(lipgloss.Center,
		message,
		"",
		"",
		buttons,
		"",
		help,
	)

	return m.RenderWithBorder(content)
}
