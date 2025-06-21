package modals

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Modal interface that all modal dialogs must implement
type Modal interface {
	tea.Model
	HandleKeyMsg(tea.KeyMsg) (Modal, tea.Cmd)
	IsComplete() bool
	GetResult() interface{}
	GetError() error
	SetTheme(Theme)
	SetSize(width, height int)
}

// ModalType represents different types of modal dialogs
type ModalType int

const (
	ModalInput ModalType = iota
	ModalConfirm
	ModalProgress
	ModalError
	ModalMultiStep
)

// ModalResult contains the result of a modal interaction
type ModalResult struct {
	Type     ModalType
	Action   string
	Data     interface{}
	Error    error
	Canceled bool
}

// Theme defines the styling for modal dialogs
type Theme struct {
	Primary      lipgloss.Color
	Secondary    lipgloss.Color
	Accent       lipgloss.Color
	Background   lipgloss.Color
	Text         lipgloss.Color
	Muted        lipgloss.Color
	Success      lipgloss.Color
	Warning      lipgloss.Color
	Error        lipgloss.Color
	BorderStyle  lipgloss.Border
	TitleStyle   lipgloss.Style
	ContentStyle lipgloss.Style
	ButtonStyle  lipgloss.Style
	InputStyle   lipgloss.Style
}

// ModalManager manages the modal dialog stack and state
type ModalManager struct {
	activeModal Modal
	modalStack  []Modal
	backdrop    bool
	theme       Theme
	width       int
	height      int
	result      *ModalResult
}

// NewModalManager creates a new modal manager
func NewModalManager(theme Theme) *ModalManager {
	return &ModalManager{
		backdrop: true,
		theme:    theme,
	}
}

// ShowModal displays a modal dialog
func (m *ModalManager) ShowModal(modal Modal) {
	if m.activeModal != nil {
		m.modalStack = append(m.modalStack, m.activeModal)
	}
	
	modal.SetTheme(m.theme)
	modal.SetSize(m.width, m.height)
	m.activeModal = modal
	m.result = nil
}

// CloseModal closes the current modal and returns to the previous one
func (m *ModalManager) CloseModal() {
	if m.activeModal != nil && m.activeModal.IsComplete() {
		// Store result before clearing modal
		m.result = &ModalResult{
			Data:     m.activeModal.GetResult(),
			Error:    m.activeModal.GetError(),
			Canceled: false,
		}
	}
	
	if len(m.modalStack) > 0 {
		m.activeModal = m.modalStack[len(m.modalStack)-1]
		m.modalStack = m.modalStack[:len(m.modalStack)-1]
	} else {
		m.activeModal = nil
	}
}

// CancelModal cancels the current modal
func (m *ModalManager) CancelModal() {
	m.result = &ModalResult{
		Canceled: true,
	}
	
	if len(m.modalStack) > 0 {
		m.activeModal = m.modalStack[len(m.modalStack)-1]
		m.modalStack = m.modalStack[:len(m.modalStack)-1]
	} else {
		m.activeModal = nil
	}
}

// IsActive returns true if there's an active modal
func (m *ModalManager) IsActive() bool {
	return m.activeModal != nil
}

// GetResult returns the result of the last completed modal
func (m *ModalManager) GetResult() *ModalResult {
	result := m.result
	m.result = nil // Clear result after retrieval
	return result
}

// SetSize updates the size for all modals
func (m *ModalManager) SetSize(width, height int) {
	m.width = width
	m.height = height
	
	if m.activeModal != nil {
		m.activeModal.SetSize(width, height)
	}
}

// SetTheme updates the theme for all modals
func (m *ModalManager) SetTheme(theme Theme) {
	m.theme = theme
	
	if m.activeModal != nil {
		m.activeModal.SetTheme(theme)
	}
}

// Update handles tea.Msg updates for the modal manager
func (m *ModalManager) Update(msg tea.Msg) tea.Cmd {
	if !m.IsActive() {
		return nil
	}
	
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
		
	case tea.KeyMsg:
		// Handle global modal keys
		switch msg.String() {
		case "esc":
			m.CancelModal()
			return nil
		}
		
		// Pass to active modal
		if m.activeModal != nil {
			updatedModal, cmd := m.activeModal.HandleKeyMsg(msg)
			m.activeModal = updatedModal
			
			// Check if modal completed
			if m.activeModal.IsComplete() {
				m.CloseModal()
			}
			
			return cmd
		}
	}
	
	// Pass other messages to active modal
	if m.activeModal != nil {
		updatedModal, cmd := m.activeModal.Update(msg)
		m.activeModal = updatedModal.(Modal)
		
		// Check if modal completed
		if m.activeModal.IsComplete() {
			m.CloseModal()
		}
		
		return cmd
	}
	
	return nil
}

// View renders the active modal with backdrop
func (m *ModalManager) View() string {
	if !m.IsActive() {
		return ""
	}
	
	modalContent := m.activeModal.View()
	
	if !m.backdrop {
		return modalContent
	}
	
	// Create backdrop (unused for now)
	_ = lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Background(lipgloss.Color("#000000")).
		Foreground(lipgloss.Color("#FFFFFF")).
		Render("")
	
	// Center the modal content
	centeredModal := lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		modalContent,
	)
	
	// Overlay modal on backdrop
	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Render(centeredModal)
}

// BaseModal provides common functionality for all modal implementations
type BaseModal struct {
	title     string
	width     int
	height    int
	theme     Theme
	complete  bool
	result    interface{}
	error     error
	minWidth  int
	minHeight int
}

// NewBaseModal creates a new base modal
func NewBaseModal(title string, minWidth, minHeight int) BaseModal {
	return BaseModal{
		title:     title,
		minWidth:  minWidth,
		minHeight: minHeight,
	}
}

// SetTheme implements the Modal interface
func (m *BaseModal) SetTheme(theme Theme) {
	m.theme = theme
}

// SetSize implements the Modal interface
func (m *BaseModal) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// IsComplete implements the Modal interface
func (m *BaseModal) IsComplete() bool {
	return m.complete
}

// GetResult implements the Modal interface
func (m *BaseModal) GetResult() interface{} {
	return m.result
}

// GetError implements the Modal interface
func (m *BaseModal) GetError() error {
	return m.error
}

// MarkComplete marks the modal as completed with a result
func (m *BaseModal) MarkComplete(result interface{}) {
	m.complete = true
	m.result = result
}

// MarkError marks the modal as completed with an error
func (m *BaseModal) MarkError(err error) {
	m.complete = true
	m.error = err
}

// RenderWithBorder renders content with a styled border and title
func (m *BaseModal) RenderWithBorder(content string) string {
	contentWidth := max(m.minWidth, 40)
	contentHeight := max(m.minHeight, 8)
	
	// Ensure modal fits in available space
	if m.width > 0 && contentWidth > m.width-4 {
		contentWidth = m.width - 4
	}
	if m.height > 0 && contentHeight > m.height-4 {
		contentHeight = m.height - 4
	}
	
	borderStyle := lipgloss.NewStyle().
		Border(m.theme.BorderStyle).
		BorderForeground(m.theme.Primary).
		Width(contentWidth).
		Height(contentHeight).
		Padding(1, 2)
	
	titleStyle := m.theme.TitleStyle.
		Width(contentWidth - 4).
		Align(lipgloss.Center)
	
	title := titleStyle.Render(m.title)
	
	return borderStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			title,
			"",
			content,
		),
	)
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}