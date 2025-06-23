package context

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func TestContextMenu(t *testing.T) {
	theme := Theme{
		Primary:     lipgloss.Color("#646CFF"),
		Secondary:   lipgloss.Color("#747BFF"),
		Accent:      lipgloss.Color("#42A5F5"),
		Background:  lipgloss.Color("#1E1E2E"),
		Text:        lipgloss.Color("#CDD6F4"),
		Muted:       lipgloss.Color("#6C7086"),
		Success:     lipgloss.Color("#A6E3A1"),
		Warning:     lipgloss.Color("#F9E2AF"),
		Error:       lipgloss.Color("#F38BA8"),
		BorderStyle: lipgloss.RoundedBorder(),
	}

	items := []ContextMenuItem{
		NewMenuItem("Item 1", "action1", "1"),
		NewMenuItem("Item 2", "action2", "2"),
		NewMenuDivider(),
		NewMenuItem("Item 3", "action3", "3"),
	}

	menu := NewContextMenu(ContextMenuConfig{
		Title: "Test Menu",
		Items: items,
		X:     10,
		Y:     5,
	}, theme)

	t.Run("Initial state", func(t *testing.T) {
		if !menu.IsVisible() {
			t.Error("Menu should be visible initially")
		}

		x, y := menu.GetPosition()
		if x != 10 || y != 5 {
			t.Errorf("Expected position (10, 5), got (%d, %d)", x, y)
		}
	})

	t.Run("Navigation", func(t *testing.T) {
		// Test down navigation
		menu, _ = menu.Update(tea.KeyMsg{Type: tea.KeyDown})

		// Should skip divider and select item 3
		menu, _ = menu.Update(tea.KeyMsg{Type: tea.KeyDown})
		menu, _ = menu.Update(tea.KeyMsg{Type: tea.KeyDown})
	})

	t.Run("Selection", func(t *testing.T) {
		_, cmd := menu.Update(tea.KeyMsg{Type: tea.KeyEnter})

		if cmd == nil {
			t.Error("Selection should return a command")
		}

		// Execute command to get the message
		msg := cmd()
		actionMsg, ok := msg.(ContextMenuActionMsg)
		if !ok {
			t.Error("Command should return ContextMenuActionMsg")
		}

		if actionMsg.Action == "" {
			t.Error("Action should not be empty")
		}
	})

	t.Run("Hide menu", func(t *testing.T) {
		menu.Hide()

		if menu.IsVisible() {
			t.Error("Menu should not be visible after hiding")
		}
	})

	t.Run("Show menu", func(t *testing.T) {
		menu.Show(20, 10)

		if !menu.IsVisible() {
			t.Error("Menu should be visible after showing")
		}

		x, y := menu.GetPosition()
		if x != 20 || y != 10 {
			t.Errorf("Expected position (20, 10), got (%d, %d)", x, y)
		}
	})

	t.Run("Escape key", func(t *testing.T) {
		menu, _ = menu.Update(tea.KeyMsg{Type: tea.KeyEsc})

		if menu.IsVisible() {
			t.Error("Menu should be hidden after escape key")
		}
	})
}

func TestContextMenuItem(t *testing.T) {
	t.Run("NewMenuItem", func(t *testing.T) {
		item := NewMenuItem("Test Item", "test_action", "t")

		if item.Label != "Test Item" {
			t.Errorf("Expected label 'Test Item', got '%s'", item.Label)
		}

		if item.Action != "test_action" {
			t.Errorf("Expected action 'test_action', got '%s'", item.Action)
		}

		if item.Key != "t" {
			t.Errorf("Expected key 't', got '%s'", item.Key)
		}

		if !item.Enabled {
			t.Error("Item should be enabled by default")
		}

		if item.Divider {
			t.Error("Item should not be a divider")
		}
	})

	t.Run("NewMenuItemWithIcon", func(t *testing.T) {
		item := NewMenuItemWithIcon("Test Item", "test_action", "t", "🔧")

		if item.Icon != "🔧" {
			t.Errorf("Expected icon '🔧', got '%s'", item.Icon)
		}
	})

	t.Run("NewMenuItemWithShortcut", func(t *testing.T) {
		item := NewMenuItemWithShortcut("Test Item", "test_action", "t", "Ctrl+T")

		if item.Shortcut != "Ctrl+T" {
			t.Errorf("Expected shortcut 'Ctrl+T', got '%s'", item.Shortcut)
		}
	})

	t.Run("NewMenuDivider", func(t *testing.T) {
		item := NewMenuDivider()

		if !item.Divider {
			t.Error("Item should be a divider")
		}

		if item.Enabled {
			t.Error("Divider should not be enabled")
		}
	})

	t.Run("NewDisabledMenuItem", func(t *testing.T) {
		item := NewDisabledMenuItem("Disabled Item")

		if item.Enabled {
			t.Error("Item should be disabled")
		}

		if item.Label != "Disabled Item" {
			t.Errorf("Expected label 'Disabled Item', got '%s'", item.Label)
		}
	})
}

func TestMenuSizing(t *testing.T) {
	items := []ContextMenuItem{
		NewMenuItem("Short", "action1", "1"),
		NewMenuItem("Much Longer Item Name", "action2", "2"),
		NewMenuItem("Medium", "action3", "3"),
	}

	width := calculateMenuWidth(items)
	height := calculateMenuHeight(items)

	if width < 20 {
		t.Errorf("Menu width should accommodate longest item, got %d", width)
	}

	if height != len(items)+2 {
		t.Errorf("Expected height %d, got %d", len(items)+2, height)
	}
}

func TestMenuWithSubmenu(t *testing.T) {
	theme := Theme{
		Primary:     lipgloss.Color("#646CFF"),
		BorderStyle: lipgloss.RoundedBorder(),
	}

	submenu := &ContextMenu{
		title: "Submenu",
		items: []ContextMenuItem{
			NewMenuItem("Sub Item 1", "sub_action1", "1"),
			NewMenuItem("Sub Item 2", "sub_action2", "2"),
		},
		theme: theme,
	}

	items := []ContextMenuItem{
		NewMenuItem("Item 1", "action1", "1"),
		{
			Label:   "Item with Submenu",
			Action:  "",
			Enabled: true,
			Submenu: submenu,
		},
	}

	menu := NewContextMenu(ContextMenuConfig{
		Title: "Test Menu",
		Items: items,
	}, theme)

	t.Run("Has submenu", func(t *testing.T) {
		// Navigate to item with submenu
		menu, _ = menu.Update(tea.KeyMsg{Type: tea.KeyDown})

		if !menu.hasSelectedSubmenu() {
			t.Error("Should detect submenu on selected item")
		}
	})

	t.Run("Open submenu", func(t *testing.T) {
		_, cmd := menu.Update(tea.KeyMsg{Type: tea.KeyRight})

		if cmd == nil {
			t.Error("Opening submenu should return a command")
		}

		// Execute command to get the message
		msg := cmd()
		submenuMsg, ok := msg.(ContextMenuSubmenuMsg)
		if !ok {
			t.Error("Command should return ContextMenuSubmenuMsg")
		}

		if submenuMsg.Submenu == nil {
			t.Error("Submenu message should contain submenu")
		}
	})
}
