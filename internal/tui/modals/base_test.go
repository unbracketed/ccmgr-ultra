package modals

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func TestModalManager(t *testing.T) {
	theme := Theme{
		Primary:      lipgloss.Color("#646CFF"),
		Secondary:    lipgloss.Color("#747BFF"),
		Accent:       lipgloss.Color("#42A5F5"),
		Background:   lipgloss.Color("#1E1E2E"),
		Text:         lipgloss.Color("#CDD6F4"),
		Muted:        lipgloss.Color("#6C7086"),
		Success:      lipgloss.Color("#A6E3A1"),
		Warning:      lipgloss.Color("#F9E2AF"),
		Error:        lipgloss.Color("#F38BA8"),
		BorderStyle:  lipgloss.RoundedBorder(),
		TitleStyle:   lipgloss.NewStyle().Bold(true),
		ContentStyle: lipgloss.NewStyle(),
		ButtonStyle:  lipgloss.NewStyle(),
		InputStyle:   lipgloss.NewStyle(),
	}

	manager := NewModalManager(theme)

	t.Run("Initial state", func(t *testing.T) {
		if manager.IsActive() {
			t.Error("Modal manager should not be active initially")
		}

		if result := manager.GetResult(); result != nil {
			t.Error("Modal manager should not have result initially")
		}
	})

	t.Run("Show modal", func(t *testing.T) {
		modal := NewInputModal(InputModalConfig{
			Title:  "Test Modal",
			Prompt: "Enter test input",
		})

		manager.ShowModal(modal)

		if !manager.IsActive() {
			t.Error("Modal manager should be active after showing modal")
		}
	})

	t.Run("Update with window size", func(t *testing.T) {
		manager.SetSize(80, 24)

		msg := tea.WindowSizeMsg{Width: 100, Height: 30}
		cmd := manager.Update(msg)

		if cmd != nil {
			t.Error("Window size update should not return command")
		}
	})

	t.Run("Cancel modal", func(t *testing.T) {
		manager.CancelModal()

		if manager.IsActive() {
			t.Error("Modal manager should not be active after canceling")
		}

		result := manager.GetResult()
		if result == nil {
			t.Error("Should have result after cancel")
		}

		if !result.Canceled {
			t.Error("Result should be marked as canceled")
		}
	})
}

func TestInputModal(t *testing.T) {
	modal := NewInputModal(InputModalConfig{
		Title:       "Test Input",
		Prompt:      "Enter value",
		Placeholder: "placeholder text",
		Required:    true,
	})

	t.Run("Initial state", func(t *testing.T) {
		if modal.IsComplete() {
			t.Error("Modal should not be complete initially")
		}

		if modal.GetResult() != nil {
			t.Error("Modal should not have result initially")
		}
	})

	t.Run("Handle character input", func(t *testing.T) {
		msg := tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune{'t', 'e', 's', 't'},
		}

		for _, char := range msg.Runes {
			keyMsg := tea.KeyMsg{
				Type:  tea.KeyRunes,
				Runes: []rune{char},
			}
			updatedModal, _ := modal.HandleKeyMsg(keyMsg)
			modal = updatedModal.(*InputModal)
		}

		// Should have "test" in the input now
		// We can't directly access the value, but we can test completion
	})

	t.Run("Submit with value", func(t *testing.T) {
		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updatedModal, _ := modal.HandleKeyMsg(msg)
		modal = updatedModal.(*InputModal)

		if !modal.IsComplete() {
			t.Error("Modal should be complete after submit with valid input")
		}

		result := modal.GetResult()
		if result == nil {
			t.Error("Modal should have result after submit")
		}
	})
}

func TestConfirmModal(t *testing.T) {
	modal := NewConfirmModal(ConfirmModalConfig{
		Title:   "Test Confirm",
		Message: "Are you sure?",
	})

	t.Run("Initial state", func(t *testing.T) {
		if modal.IsComplete() {
			t.Error("Modal should not be complete initially")
		}
	})

	t.Run("Confirm action", func(t *testing.T) {
		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updatedModal, _ := modal.HandleKeyMsg(msg)
		modal = updatedModal.(*ConfirmModal)

		if !modal.IsComplete() {
			t.Error("Modal should be complete after confirmation")
		}

		result := modal.GetResult()
		if result == nil {
			t.Error("Modal should have result after confirmation")
		}
	})
}

func TestProgressModal(t *testing.T) {
	modal := NewProgressModal(ProgressModalConfig{
		Title:         "Test Progress",
		Message:       "Processing...",
		Indeterminate: false,
	})

	t.Run("Initial state", func(t *testing.T) {
		if modal.IsComplete() {
			t.Error("Modal should not be complete initially")
		}
	})

	t.Run("Update progress", func(t *testing.T) {
		updateMsg := ProgressUpdateMsg{
			Progress: 0.5,
			Status:   "50% complete",
		}

		updatedModal, _ := modal.Update(updateMsg)
		modal = updatedModal.(*ProgressModal)

		// Progress should be updated internally
	})

	t.Run("Complete progress", func(t *testing.T) {
		updateMsg := ProgressUpdateMsg{
			Progress: 1.0,
			Status:   "Complete",
			Complete: true,
		}

		updatedModal, _ := modal.Update(updateMsg)
		modal = updatedModal.(*ProgressModal)

		if !modal.IsComplete() {
			t.Error("Modal should be complete when progress update marks it complete")
		}
	})
}

func TestErrorModal(t *testing.T) {
	modal := NewSimpleErrorModal("Test Error", "Something went wrong")

	t.Run("Initial state", func(t *testing.T) {
		if modal.IsComplete() {
			t.Error("Modal should not be complete initially")
		}
	})

	t.Run("Confirm error", func(t *testing.T) {
		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updatedModal, _ := modal.HandleKeyMsg(msg)
		modal = updatedModal.(*ErrorModal)

		if !modal.IsComplete() {
			t.Error("Modal should be complete after confirming error")
		}

		result := modal.GetResult()
		if result == nil {
			t.Error("Modal should have result after confirmation")
		}

		if result != "ok" {
			t.Errorf("Expected 'ok' result, got: %v", result)
		}
	})
}
