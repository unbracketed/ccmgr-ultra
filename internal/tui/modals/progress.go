package modals

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ProgressModal represents a progress indicator modal
type ProgressModal struct {
	BaseModal
	message       string
	progress      float64 // 0.0 to 1.0
	indeterminate bool
	status        string
	spinner       int
	tickCounter   int
	cancelable    bool
	canceled      bool
	showETA       bool
	startTime     time.Time
	totalSteps    int
	currentStep   int
}

// ProgressModalConfig configures a progress modal
type ProgressModalConfig struct {
	Title         string
	Message       string
	Indeterminate bool // True for spinner, false for progress bar
	Cancelable    bool
	ShowETA       bool
	TotalSteps    int
}

// ProgressUpdate represents a progress update message
type ProgressUpdateMsg struct {
	Progress    float64 // 0.0 to 1.0
	Status      string
	CurrentStep int
	Complete    bool
}

// ProgressTickMsg is sent for spinner animation
type ProgressTickMsg time.Time

// NewProgressModal creates a new progress modal
func NewProgressModal(config ProgressModalConfig) *ProgressModal {
	return &ProgressModal{
		BaseModal:     NewBaseModal(config.Title, 50, 12),
		message:       config.Message,
		indeterminate: config.Indeterminate,
		cancelable:    config.Cancelable,
		showETA:       config.ShowETA,
		totalSteps:    config.TotalSteps,
		startTime:     time.Now(),
	}
}

// Init implements the tea.Model interface
func (m *ProgressModal) Init() tea.Cmd {
	if m.indeterminate {
		return m.tick()
	}
	return nil
}

// Update implements the tea.Model interface
func (m *ProgressModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.HandleKeyMsg(msg)
	case ProgressUpdateMsg:
		return m.handleProgressUpdate(msg)
	case ProgressTickMsg:
		return m.handleTick()
	}
	return m, nil
}

// HandleKeyMsg implements the Modal interface
func (m *ProgressModal) HandleKeyMsg(msg tea.KeyMsg) (Modal, tea.Cmd) {
	if !m.cancelable {
		return m, nil
	}

	switch msg.String() {
	case "ctrl+c", "esc":
		m.canceled = true
		m.MarkComplete(false)
		return m, nil
	}

	return m, nil
}

// handleProgressUpdate processes progress updates
func (m *ProgressModal) handleProgressUpdate(msg ProgressUpdateMsg) (Modal, tea.Cmd) {
	m.progress = msg.Progress
	if msg.Status != "" {
		m.status = msg.Status
	}
	if msg.CurrentStep > 0 {
		m.currentStep = msg.CurrentStep
	}

	if msg.Complete {
		m.MarkComplete(true)
		return m, nil
	}

	var cmd tea.Cmd
	if m.indeterminate {
		cmd = m.tick()
	}

	return m, cmd
}

// handleTick processes spinner animation ticks
func (m *ProgressModal) handleTick() (Modal, tea.Cmd) {
	m.tickCounter++
	m.spinner = (m.spinner + 1) % len(spinnerFrames)
	return m, m.tick()
}

// tick returns a command for the next spinner animation frame
func (m *ProgressModal) tick() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return ProgressTickMsg(t)
	})
}

// UpdateProgress updates the progress value and status
func (m *ProgressModal) UpdateProgress(progress float64, status string) {
	m.progress = progress
	if status != "" {
		m.status = status
	}
}

// MarkCompleted marks the operation as completed
func (m *ProgressModal) MarkCompleted() {
	m.MarkComplete(true)
}

// IsCanceled returns true if the operation was canceled
func (m *ProgressModal) IsCanceled() bool {
	return m.canceled
}

// View implements the tea.Model interface
func (m *ProgressModal) View() string {
	var elements []string

	// Message
	messageStyle := m.theme.ContentStyle.Copy().Bold(true)
	elements = append(elements, messageStyle.Render(m.message))

	if m.indeterminate {
		// Spinner view
		spinnerStyle := lipgloss.NewStyle().
			Foreground(m.theme.Accent).
			Bold(true)

		spinner := spinnerStyle.Render(spinnerFrames[m.spinner])
		statusText := m.status
		if statusText == "" {
			statusText = "Working..."
		}

		spinnerLine := lipgloss.JoinHorizontal(lipgloss.Left,
			spinner, " ", statusText)
		elements = append(elements, "", spinnerLine)
	} else {
		// Progress bar view
		progressBar := m.renderProgressBar()
		elements = append(elements, "", progressBar)

		// Step counter if available
		if m.totalSteps > 0 {
			stepText := fmt.Sprintf("Step %d of %d", m.currentStep, m.totalSteps)
			stepStyle := lipgloss.NewStyle().Foreground(m.theme.Muted)
			elements = append(elements, stepStyle.Render(stepText))
		}

		// Status text
		if m.status != "" {
			statusStyle := m.theme.ContentStyle
			elements = append(elements, statusStyle.Render(m.status))
		}

		// ETA if enabled
		if m.showETA && m.progress > 0 {
			eta := m.calculateETA()
			if eta != "" {
				etaStyle := lipgloss.NewStyle().
					Foreground(m.theme.Muted).
					Italic(true)
				elements = append(elements, etaStyle.Render("ETA: "+eta))
			}
		}
	}

	// Help text
	if m.cancelable {
		helpStyle := lipgloss.NewStyle().
			Foreground(m.theme.Muted).
			Italic(true)
		help := helpStyle.Render("Esc: Cancel")
		elements = append(elements, "", help)
	}

	content := strings.Join(elements, "\n")
	return m.RenderWithBorder(content)
}

// renderProgressBar creates a visual progress bar
func (m *ProgressModal) renderProgressBar() string {
	width := 40
	filled := int(m.progress * float64(width))

	_ = strings.Repeat("█", filled) + strings.Repeat("░", width-filled)

	barStyle := lipgloss.NewStyle().
		Foreground(m.theme.Success)

	emptyStyle := lipgloss.NewStyle().
		Foreground(m.theme.Muted)

	filledPart := barStyle.Render(strings.Repeat("█", filled))
	emptyPart := emptyStyle.Render(strings.Repeat("░", width-filled))

	percentage := fmt.Sprintf("%.1f%%", m.progress*100)

	return lipgloss.JoinHorizontal(lipgloss.Left,
		"[", filledPart, emptyPart, "] ", percentage)
}

// calculateETA estimates time remaining based on current progress
func (m *ProgressModal) calculateETA() string {
	if m.progress <= 0 {
		return ""
	}

	elapsed := time.Since(m.startTime)
	remaining := time.Duration(float64(elapsed) * (1.0/m.progress - 1.0))

	if remaining < time.Minute {
		return fmt.Sprintf("%ds", int(remaining.Seconds()))
	} else if remaining < time.Hour {
		return fmt.Sprintf("%dm %ds", int(remaining.Minutes()), int(remaining.Seconds())%60)
	} else {
		return fmt.Sprintf("%dh %dm", int(remaining.Hours()), int(remaining.Minutes())%60)
	}
}

// Spinner frames for indeterminate progress
var spinnerFrames = []string{
	"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏",
}
