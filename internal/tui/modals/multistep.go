package modals

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MultiStepModal represents a multi-step wizard modal
type MultiStepModal struct {
	BaseModal
	steps        []Step
	currentStep  int
	stepData     map[string]interface{}
	canGoBack    bool
	canGoNext    bool
	showProgress bool
}

// Step represents a single step in a multi-step wizard
type Step interface {
	Title() string
	Description() string
	Render(theme Theme, width int, data map[string]interface{}) string
	HandleKey(msg tea.KeyMsg, data map[string]interface{}) (map[string]interface{}, tea.Cmd, error)
	Validate(data map[string]interface{}) error
	IsComplete(data map[string]interface{}) bool
}

// MultiStepModalConfig configures a multi-step modal
type MultiStepModalConfig struct {
	Title        string
	Steps        []Step
	ShowProgress bool
}

// NewMultiStepModal creates a new multi-step modal
func NewMultiStepModal(config MultiStepModalConfig) *MultiStepModal {
	return &MultiStepModal{
		BaseModal:    NewBaseModal(config.Title, 60, 16),
		steps:        config.Steps,
		stepData:     make(map[string]interface{}),
		showProgress: config.ShowProgress,
		canGoNext:    len(config.Steps) > 0,
	}
}

// Init implements the tea.Model interface
func (m *MultiStepModal) Init() tea.Cmd {
	return nil
}

// Update implements the tea.Model interface
func (m *MultiStepModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.HandleKeyMsg(msg)
	}
	return m, nil
}

// HandleKeyMsg implements the Modal interface
func (m *MultiStepModal) HandleKeyMsg(msg tea.KeyMsg) (Modal, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc":
		m.MarkComplete(nil)
		return m, nil
		
	case "ctrl+n", "f1":
		return m.nextStep()
		
	case "ctrl+p", "f2":
		return m.previousStep()
		
	case "ctrl+enter":
		// Complete wizard
		if m.canFinish() {
			m.MarkComplete(m.stepData)
			return m, nil
		}
		return m.nextStep()
	}
	
	// Pass key to current step
	if m.currentStep < len(m.steps) {
		step := m.steps[m.currentStep]
		data, cmd, err := step.HandleKey(msg, m.stepData)
		if err != nil {
			m.MarkError(err)
			return m, cmd
		}
		
		// Update step data
		for k, v := range data {
			m.stepData[k] = v
		}
		
		// Update navigation state
		m.updateNavigationState()
		
		return m, cmd
	}
	
	return m, nil
}

// nextStep moves to the next step
func (m *MultiStepModal) nextStep() (Modal, tea.Cmd) {
	if !m.canGoNext {
		return m, nil
	}
	
	// Validate current step
	if m.currentStep < len(m.steps) {
		step := m.steps[m.currentStep]
		if err := step.Validate(m.stepData); err != nil {
			m.MarkError(err)
			return m, nil
		}
	}
	
	if m.currentStep < len(m.steps)-1 {
		m.currentStep++
		m.updateNavigationState()
	} else if m.canFinish() {
		m.MarkComplete(m.stepData)
		return m, nil
	}
	
	return m, nil
}

// previousStep moves to the previous step
func (m *MultiStepModal) previousStep() (Modal, tea.Cmd) {
	if !m.canGoBack {
		return m, nil
	}
	
	if m.currentStep > 0 {
		m.currentStep--
		m.updateNavigationState()
	}
	
	return m, nil
}

// updateNavigationState updates the canGoBack/canGoNext flags
func (m *MultiStepModal) updateNavigationState() {
	m.canGoBack = m.currentStep > 0
	m.canGoNext = m.currentStep < len(m.steps)-1
	
	// Check if current step is complete for next navigation
	if m.currentStep < len(m.steps) {
		step := m.steps[m.currentStep]
		if !step.IsComplete(m.stepData) {
			m.canGoNext = false
		}
	}
}

// canFinish checks if the wizard can be completed
func (m *MultiStepModal) canFinish() bool {
	if m.currentStep != len(m.steps)-1 {
		return false
	}
	
	// All steps must be complete
	for _, step := range m.steps {
		if !step.IsComplete(m.stepData) {
			return false
		}
	}
	
	return true
}

// View implements the tea.Model interface
func (m *MultiStepModal) View() string {
	if len(m.steps) == 0 {
		return m.RenderWithBorder("No steps defined")
	}
	
	var elements []string
	
	// Progress indicator
	if m.showProgress {
		progress := m.renderProgress()
		elements = append(elements, progress)
		elements = append(elements, "")
	}
	
	// Current step
	if m.currentStep < len(m.steps) {
		step := m.steps[m.currentStep]
		
		// Step title
		titleStyle := m.theme.TitleStyle.Copy().
			Foreground(m.theme.Accent)
		title := titleStyle.Render(step.Title())
		elements = append(elements, title)
		
		// Step description
		if desc := step.Description(); desc != "" {
			descStyle := lipgloss.NewStyle().
				Foreground(m.theme.Muted).
				Italic(true)
			elements = append(elements, descStyle.Render(desc))
		}
		
		elements = append(elements, "")
		
		// Step content
		stepContent := step.Render(m.theme, m.width-8, m.stepData)
		elements = append(elements, stepContent)
	}
	
	elements = append(elements, "")
	
	// Navigation buttons
	navigation := m.renderNavigation()
	elements = append(elements, navigation)
	
	// Help text
	help := m.renderHelp()
	elements = append(elements, "", help)
	
	content := strings.Join(elements, "\n")
	return m.RenderWithBorder(content)
}

// renderProgress creates a visual progress indicator
func (m *MultiStepModal) renderProgress() string {
	if len(m.steps) == 0 {
		return ""
	}
	
	var indicators []string
	for i, step := range m.steps {
		indicator := ""
		
		if i < m.currentStep {
			// Completed step
			indicator = lipgloss.NewStyle().
				Foreground(m.theme.Success).
				Bold(true).
				Render("✓")
		} else if i == m.currentStep {
			// Current step
			indicator = lipgloss.NewStyle().
				Foreground(m.theme.Accent).
				Bold(true).
				Render("●")
		} else {
			// Future step
			indicator = lipgloss.NewStyle().
				Foreground(m.theme.Muted).
				Render("○")
		}
		
		// Add step title
		stepTitle := lipgloss.NewStyle().
			Width(12).
			Align(lipgloss.Center)
		
		if i == m.currentStep {
			stepTitle = stepTitle.Foreground(m.theme.Accent).Bold(true)
		} else {
			stepTitle = stepTitle.Foreground(m.theme.Muted)
		}
		
		stepText := stepTitle.Render(step.Title())
		
		stepIndicator := lipgloss.JoinVertical(lipgloss.Center,
			indicator,
			stepText,
		)
		
		indicators = append(indicators, stepIndicator)
	}
	
	progress := lipgloss.JoinHorizontal(lipgloss.Center, indicators...)
	
	// Add step counter
	counter := fmt.Sprintf("Step %d of %d", m.currentStep+1, len(m.steps))
	counterStyle := lipgloss.NewStyle().
		Foreground(m.theme.Muted).
		Italic(true)
	
	return lipgloss.JoinVertical(lipgloss.Center,
		progress,
		"",
		counterStyle.Render(counter),
	)
}

// renderNavigation creates navigation buttons
func (m *MultiStepModal) renderNavigation() string {
	var buttons []string
	
	// Back button
	backStyle := lipgloss.NewStyle().
		Padding(0, 2).
		Border(lipgloss.RoundedBorder())
	
	if m.canGoBack {
		backStyle = backStyle.
			BorderForeground(m.theme.Accent).
			Foreground(m.theme.Text)
	} else {
		backStyle = backStyle.
			BorderForeground(m.theme.Muted).
			Foreground(m.theme.Muted)
	}
	
	backButton := backStyle.Render("◀ Back")
	buttons = append(buttons, backButton)
	
	// Next/Finish button
	nextStyle := lipgloss.NewStyle().
		Padding(0, 2).
		Border(lipgloss.RoundedBorder())
	
	var nextLabel string
	if m.currentStep == len(m.steps)-1 {
		nextLabel = "Finish"
		if m.canFinish() {
			nextStyle = nextStyle.
				Background(m.theme.Success).
				BorderForeground(m.theme.Success).
				Foreground(lipgloss.Color("#FFFFFF")).
				Bold(true)
		} else {
			nextStyle = nextStyle.
				BorderForeground(m.theme.Muted).
				Foreground(m.theme.Muted)
		}
	} else {
		nextLabel = "Next ▶"
		if m.canGoNext {
			nextStyle = nextStyle.
				Background(m.theme.Accent).
				BorderForeground(m.theme.Accent).
				Foreground(lipgloss.Color("#FFFFFF")).
				Bold(true)
		} else {
			nextStyle = nextStyle.
				BorderForeground(m.theme.Muted).
				Foreground(m.theme.Muted)
		}
	}
	
	nextButton := nextStyle.Render(nextLabel)
	buttons = append(buttons, nextButton)
	
	return lipgloss.JoinHorizontal(lipgloss.Center,
		buttons[0], "   ", buttons[1])
}

// renderHelp creates help text
func (m *MultiStepModal) renderHelp() string {
	helpLines := []string{
		"Ctrl+N/F1: Next",
		"Ctrl+P/F2: Back",
		"Ctrl+Enter: Finish",
		"Esc: Cancel",
	}
	
	helpStyle := lipgloss.NewStyle().
		Foreground(m.theme.Muted).
		Italic(true)
	
	return helpStyle.Render(strings.Join(helpLines, " • "))
}