package modals

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// InputModal represents an input dialog for collecting user text input
type InputModal struct {
	BaseModal
	prompt      string
	placeholder string
	value       string
	cursor      int
	validator   func(string) error
	multiline   bool
	maxLength   int
	required    bool
	password    bool
}

// InputModalConfig configures an input modal
type InputModalConfig struct {
	Title       string
	Prompt      string
	Placeholder string
	DefaultValue string
	Validator   func(string) error
	Multiline   bool
	MaxLength   int
	Required    bool
	Password    bool
}

// NewInputModal creates a new input modal
func NewInputModal(config InputModalConfig) *InputModal {
	maxLength := config.MaxLength
	if maxLength == 0 {
		maxLength = 200 // Default max length
	}
	
	minWidth := 50
	minHeight := 8
	if config.Multiline {
		minHeight = 12
	}
	
	return &InputModal{
		BaseModal:   NewBaseModal(config.Title, minWidth, minHeight),
		prompt:      config.Prompt,
		placeholder: config.Placeholder,
		value:       config.DefaultValue,
		cursor:      len(config.DefaultValue),
		validator:   config.Validator,
		multiline:   config.Multiline,
		maxLength:   maxLength,
		required:    config.Required,
		password:    config.Password,
	}
}

// Init implements the tea.Model interface
func (m *InputModal) Init() tea.Cmd {
	return nil
}

// Update implements the tea.Model interface
func (m *InputModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.HandleKeyMsg(msg)
	}
	return m, nil
}

// HandleKeyMsg implements the Modal interface
func (m *InputModal) HandleKeyMsg(msg tea.KeyMsg) (Modal, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc":
		return m, nil // Let modal manager handle cancellation
		
	case "enter":
		if !m.multiline {
			return m.submit()
		}
		// For multiline, enter adds a newline
		if len(m.value) < m.maxLength {
			m.value = m.value[:m.cursor] + "\n" + m.value[m.cursor:]
			m.cursor++
		}
		
	case "ctrl+enter":
		if m.multiline {
			return m.submit()
		}
		
	case "left":
		if m.cursor > 0 {
			m.cursor--
		}
		
	case "right":
		if m.cursor < len(m.value) {
			m.cursor++
		}
		
	case "home":
		m.cursor = 0
		
	case "end":
		m.cursor = len(m.value)
		
	case "backspace":
		if m.cursor > 0 {
			m.value = m.value[:m.cursor-1] + m.value[m.cursor:]
			m.cursor--
		}
		
	case "delete":
		if m.cursor < len(m.value) {
			m.value = m.value[:m.cursor] + m.value[m.cursor+1:]
		}
		
	case "ctrl+a":
		m.cursor = 0
		
	case "ctrl+e":
		m.cursor = len(m.value)
		
	case "ctrl+k":
		m.value = m.value[:m.cursor]
		
	case "ctrl+u":
		m.value = m.value[m.cursor:]
		m.cursor = 0
		
	default:
		// Handle regular character input
		if len(msg.Runes) > 0 && len(m.value) < m.maxLength {
			char := string(msg.Runes[0])
			m.value = m.value[:m.cursor] + char + m.value[m.cursor:]
			m.cursor++
		}
	}
	
	return m, nil
}

// submit validates and submits the input
func (m *InputModal) submit() (Modal, tea.Cmd) {
	// Check if required field is empty
	if m.required && strings.TrimSpace(m.value) == "" {
		m.MarkError(fmt.Errorf("field is required"))
		return m, nil
	}
	
	// Run validator if provided
	if m.validator != nil {
		if err := m.validator(m.value); err != nil {
			m.MarkError(err)
			return m, nil
		}
	}
	
	m.MarkComplete(m.value)
	return m, nil
}

// View implements the tea.Model interface
func (m *InputModal) View() string {
	// Build prompt section
	promptStyle := m.theme.ContentStyle.Copy().Bold(true)
	prompt := promptStyle.Render(m.prompt)
	
	// Build input field
	displayValue := m.value
	if m.password {
		displayValue = strings.Repeat("*", len(m.value))
	}
	
	// Add cursor
	if m.cursor <= len(displayValue) {
		displayValue = displayValue[:m.cursor] + "│" + displayValue[m.cursor:]
	}
	
	// Style input field
	inputStyle := lipgloss.NewStyle().
		Width(m.width - 8).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Accent).
		Padding(0, 1)
	
	input := inputStyle.Render(displayValue)
	
	// Add placeholder if value is empty
	if m.value == "" && m.placeholder != "" {
		placeholderStyle := lipgloss.NewStyle().
			Width(m.width - 8).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(m.theme.Muted).
			Foreground(m.theme.Muted).
			Padding(0, 1)
		input = placeholderStyle.Render(m.placeholder + "│")
	}
	
	// Build help text
	helpLines := []string{}
	if m.multiline {
		helpLines = append(helpLines, "Ctrl+Enter: Submit")
		helpLines = append(helpLines, "Enter: New line")
	} else {
		helpLines = append(helpLines, "Enter: Submit")
	}
	helpLines = append(helpLines, "Esc: Cancel")
	
	helpStyle := lipgloss.NewStyle().
		Foreground(m.theme.Muted).
		Italic(true)
	help := helpStyle.Render(strings.Join(helpLines, " • "))
	
	// Show validation error if any
	var errorText string
	if m.error != nil {
		errorStyle := lipgloss.NewStyle().
			Foreground(m.theme.Error).
			Bold(true)
		errorText = errorStyle.Render("Error: " + m.error.Error())
	}
	
	// Combine all elements
	content := lipgloss.JoinVertical(lipgloss.Left,
		prompt,
		"",
		input,
		"",
		help,
	)
	
	if errorText != "" {
		content = lipgloss.JoinVertical(lipgloss.Left,
			content,
			"",
			errorText,
		)
	}
	
	return m.RenderWithBorder(content)
}