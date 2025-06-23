package context

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ContextMenu represents a context-sensitive action menu
type ContextMenu struct {
	title         string
	items         []ContextMenuItem
	selectedIndex int
	width         int
	height        int
	x             int
	y             int
	visible       bool
	theme         Theme
}

// ContextMenuItem represents an item in a context menu
type ContextMenuItem struct {
	Label    string
	Key      string
	Action   string
	Enabled  bool
	Divider  bool
	Icon     string
	Shortcut string
	Submenu  *ContextMenu
}

// Theme defines the styling for context menus
type Theme struct {
	Primary     lipgloss.Color
	Secondary   lipgloss.Color
	Accent      lipgloss.Color
	Background  lipgloss.Color
	Text        lipgloss.Color
	Muted       lipgloss.Color
	Success     lipgloss.Color
	Warning     lipgloss.Color
	Error       lipgloss.Color
	BorderStyle lipgloss.Border
}

// ContextMenuConfig configures a context menu
type ContextMenuConfig struct {
	Title string
	Items []ContextMenuItem
	X     int
	Y     int
}

// NewContextMenu creates a new context menu
func NewContextMenu(config ContextMenuConfig, theme Theme) *ContextMenu {
	return &ContextMenu{
		title:   config.Title,
		items:   config.Items,
		x:       config.X,
		y:       config.Y,
		visible: true,
		theme:   theme,
		width:   calculateMenuWidth(config.Items),
		height:  calculateMenuHeight(config.Items),
	}
}

// calculateMenuWidth determines the optimal width for the menu
func calculateMenuWidth(items []ContextMenuItem) int {
	maxWidth := 20
	for _, item := range items {
		if item.Divider {
			continue
		}

		itemWidth := len(item.Label)
		if item.Icon != "" {
			itemWidth += 2
		}
		if item.Shortcut != "" {
			itemWidth += len(item.Shortcut) + 2
		}

		if itemWidth > maxWidth {
			maxWidth = itemWidth
		}
	}

	return maxWidth + 4 // Add padding
}

// calculateMenuHeight determines the height for the menu
func calculateMenuHeight(items []ContextMenuItem) int {
	return len(items) + 2 // Items plus top/bottom padding
}

// Show displays the context menu at the specified position
func (m *ContextMenu) Show(x, y int) {
	m.x = x
	m.y = y
	m.visible = true
	m.selectedIndex = 0

	// Find first enabled item
	for i, item := range m.items {
		if item.Enabled && !item.Divider {
			m.selectedIndex = i
			break
		}
	}
}

// Hide conceals the context menu
func (m *ContextMenu) Hide() {
	m.visible = false
}

// IsVisible returns whether the menu is currently visible
func (m *ContextMenu) IsVisible() bool {
	return m.visible
}

// GetPosition returns the menu's current position
func (m *ContextMenu) GetPosition() (int, int) {
	return m.x, m.y
}

// GetSize returns the menu's dimensions
func (m *ContextMenu) GetSize() (int, int) {
	return m.width, m.height
}

// Update handles input messages for the context menu
func (m *ContextMenu) Update(msg tea.Msg) (*ContextMenu, tea.Cmd) {
	if !m.visible {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	}

	return m, nil
}

// handleKeyMsg processes keyboard input
func (m *ContextMenu) handleKeyMsg(msg tea.KeyMsg) (*ContextMenu, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.Hide()
		return m, nil

	case "up", "k":
		m.movePrevious()

	case "down", "j":
		m.moveNext()

	case "enter", " ":
		return m.activateSelected()

	case "left", "h":
		// Handle submenu navigation
		if m.hasSelectedSubmenu() {
			// Close submenu
			return m, nil
		}
		m.Hide()
		return m, nil

	case "right", "l":
		// Handle submenu navigation
		if m.hasSelectedSubmenu() {
			// Open submenu
			return m, m.openSubmenu()
		}

	default:
		// Handle shortcut keys
		for i, item := range m.items {
			if item.Enabled && !item.Divider && item.Key == msg.String() {
				m.selectedIndex = i
				return m.activateSelected()
			}
		}
	}

	return m, nil
}

// movePrevious moves selection to the previous enabled item
func (m *ContextMenu) movePrevious() {
	if len(m.items) == 0 {
		return
	}

	start := m.selectedIndex
	for {
		m.selectedIndex--
		if m.selectedIndex < 0 {
			m.selectedIndex = len(m.items) - 1
		}

		if m.selectedIndex == start {
			break // Avoid infinite loop
		}

		if m.items[m.selectedIndex].Enabled && !m.items[m.selectedIndex].Divider {
			break
		}
	}
}

// moveNext moves selection to the next enabled item
func (m *ContextMenu) moveNext() {
	if len(m.items) == 0 {
		return
	}

	start := m.selectedIndex
	for {
		m.selectedIndex++
		if m.selectedIndex >= len(m.items) {
			m.selectedIndex = 0
		}

		if m.selectedIndex == start {
			break // Avoid infinite loop
		}

		if m.items[m.selectedIndex].Enabled && !m.items[m.selectedIndex].Divider {
			break
		}
	}
}

// activateSelected triggers the action for the selected item
func (m *ContextMenu) activateSelected() (*ContextMenu, tea.Cmd) {
	if m.selectedIndex < 0 || m.selectedIndex >= len(m.items) {
		return m, nil
	}

	item := m.items[m.selectedIndex]
	if !item.Enabled || item.Divider {
		return m, nil
	}

	m.Hide()

	// Return a command that indicates the selected action
	return m, func() tea.Msg {
		return ContextMenuActionMsg{
			Action: item.Action,
			Item:   item,
		}
	}
}

// hasSelectedSubmenu checks if the selected item has a submenu
func (m *ContextMenu) hasSelectedSubmenu() bool {
	if m.selectedIndex < 0 || m.selectedIndex >= len(m.items) {
		return false
	}
	return m.items[m.selectedIndex].Submenu != nil
}

// openSubmenu opens the submenu for the selected item
func (m *ContextMenu) openSubmenu() tea.Cmd {
	if !m.hasSelectedSubmenu() {
		return nil
	}

	submenu := m.items[m.selectedIndex].Submenu
	submenu.Show(m.x+m.width, m.y+m.selectedIndex)

	return func() tea.Msg {
		return ContextMenuSubmenuMsg{
			Submenu: submenu,
		}
	}
}

// View renders the context menu
func (m *ContextMenu) View() string {
	if !m.visible {
		return ""
	}

	var elements []string

	// Title (if present)
	if m.title != "" {
		titleStyle := lipgloss.NewStyle().
			Foreground(m.theme.Accent).
			Bold(true).
			Width(m.width - 2).
			Align(lipgloss.Center)
		elements = append(elements, titleStyle.Render(m.title))

		// Title separator
		separatorStyle := lipgloss.NewStyle().
			Foreground(m.theme.Muted).
			Width(m.width - 2)
		elements = append(elements, separatorStyle.Render(strings.Repeat("─", m.width-2)))
	}

	// Menu items
	for i, item := range m.items {
		if item.Divider {
			elements = append(elements, m.renderDivider())
		} else {
			elements = append(elements, m.renderItem(item, i == m.selectedIndex))
		}
	}

	// Combine content
	content := strings.Join(elements, "\n")

	// Apply border and styling
	menuStyle := lipgloss.NewStyle().
		Border(m.theme.BorderStyle).
		BorderForeground(m.theme.Primary).
		Background(m.theme.Background).
		Foreground(m.theme.Text).
		Width(m.width).
		Padding(0, 1)

	return menuStyle.Render(content)
}

// renderItem renders a single menu item
func (m *ContextMenu) renderItem(item ContextMenuItem, selected bool) string {
	var parts []string

	// Icon
	if item.Icon != "" {
		iconStyle := lipgloss.NewStyle().Width(2)
		if selected {
			iconStyle = iconStyle.Foreground(m.theme.Accent)
		} else {
			iconStyle = iconStyle.Foreground(m.theme.Muted)
		}
		parts = append(parts, iconStyle.Render(item.Icon))
	} else {
		parts = append(parts, "  ")
	}

	// Label
	labelStyle := lipgloss.NewStyle()
	if !item.Enabled {
		labelStyle = labelStyle.Foreground(m.theme.Muted)
	} else if selected {
		labelStyle = labelStyle.
			Foreground(m.theme.Background).
			Background(m.theme.Accent).
			Bold(true)
	} else {
		labelStyle = labelStyle.Foreground(m.theme.Text)
	}

	labelWidth := m.width - 4 // Account for icon and padding
	if item.Shortcut != "" {
		labelWidth -= len(item.Shortcut) + 2
	}
	if item.Submenu != nil {
		labelWidth -= 2 // Account for submenu arrow
	}

	label := labelStyle.Width(labelWidth).Render(item.Label)
	parts = append(parts, label)

	// Shortcut
	if item.Shortcut != "" {
		shortcutStyle := lipgloss.NewStyle().
			Foreground(m.theme.Muted).
			Width(len(item.Shortcut))
		parts = append(parts, shortcutStyle.Render(item.Shortcut))
	}

	// Submenu indicator
	if item.Submenu != nil {
		arrowStyle := lipgloss.NewStyle().
			Foreground(m.theme.Muted).
			Width(2)
		if selected {
			arrowStyle = arrowStyle.Foreground(m.theme.Background)
		}
		parts = append(parts, arrowStyle.Render("▶"))
	}

	return strings.Join(parts, "")
}

// renderDivider renders a separator line
func (m *ContextMenu) renderDivider() string {
	style := lipgloss.NewStyle().
		Foreground(m.theme.Muted).
		Width(m.width - 2)
	return style.Render(strings.Repeat("─", m.width-2))
}

// ContextMenuActionMsg is sent when a menu item is selected
type ContextMenuActionMsg struct {
	Action string
	Item   ContextMenuItem
}

// ContextMenuSubmenuMsg is sent when a submenu is opened
type ContextMenuSubmenuMsg struct {
	Submenu *ContextMenu
}

// Common menu item constructors

// NewMenuItem creates a basic menu item
func NewMenuItem(label, action, key string) ContextMenuItem {
	return ContextMenuItem{
		Label:   label,
		Action:  action,
		Key:     key,
		Enabled: true,
	}
}

// NewMenuItemWithIcon creates a menu item with an icon
func NewMenuItemWithIcon(label, action, key, icon string) ContextMenuItem {
	return ContextMenuItem{
		Label:   label,
		Action:  action,
		Key:     key,
		Icon:    icon,
		Enabled: true,
	}
}

// NewMenuItemWithShortcut creates a menu item with a keyboard shortcut
func NewMenuItemWithShortcut(label, action, key, shortcut string) ContextMenuItem {
	return ContextMenuItem{
		Label:    label,
		Action:   action,
		Key:      key,
		Shortcut: shortcut,
		Enabled:  true,
	}
}

// NewMenuDivider creates a separator item
func NewMenuDivider() ContextMenuItem {
	return ContextMenuItem{
		Divider: true,
		Enabled: false,
	}
}

// NewDisabledMenuItem creates a disabled menu item
func NewDisabledMenuItem(label string) ContextMenuItem {
	return ContextMenuItem{
		Label:   label,
		Enabled: false,
	}
}
