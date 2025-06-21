package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ConfigTextInput represents a text input field for configuration
type ConfigTextInput struct {
	label       string
	value       string
	placeholder string
	validator   func(string) error
	multiline   bool
	focused     bool
	err         error
	textInput   textinput.Model
	theme       Theme
	originalValue string
}

// NewConfigTextInput creates a new text input component
func NewConfigTextInput(label, value, placeholder string, theme Theme) *ConfigTextInput {
	ti := textinput.New()
	ti.SetValue(value)
	ti.Placeholder = placeholder
	ti.CharLimit = 256

	return &ConfigTextInput{
		label:         label,
		value:         value,
		originalValue: value,
		placeholder:   placeholder,
		textInput:     ti,
		theme:         theme,
	}
}

// SetValidator sets the validation function
func (c *ConfigTextInput) SetValidator(validator func(string) error) {
	c.validator = validator
}

// Focus sets the focus state
func (c *ConfigTextInput) Focus() {
	c.focused = true
	c.textInput.Focus()
}

// Blur removes focus
func (c *ConfigTextInput) Blur() {
	c.focused = false
	c.textInput.Blur()
}

// Update handles input events
func (c *ConfigTextInput) Update(msg tea.Msg) (*ConfigTextInput, tea.Cmd) {
	var cmd tea.Cmd
	c.textInput, cmd = c.textInput.Update(msg)
	c.value = c.textInput.Value()

	// Validate on change
	if c.validator != nil {
		c.err = c.validator(c.value)
	}

	return c, cmd
}

// View renders the text input
func (c *ConfigTextInput) View() string {
	labelStyle := c.theme.LabelStyle
	if c.focused {
		labelStyle = c.theme.FocusedStyle
	}

	label := labelStyle.Render(c.label + ":")
	
	inputView := c.textInput.View()
	
	// Show error if present
	if c.err != nil {
		errorMsg := c.theme.ErrorStyle.Render("  " + c.err.Error())
		return lipgloss.JoinVertical(lipgloss.Left,
			label,
			inputView,
			errorMsg,
		)
	}

	// Show if modified
	if c.value != c.originalValue {
		inputView += c.theme.WarningStyle.Render(" *")
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		label,
		inputView,
	)
}

// HasChanged returns true if the value has changed
func (c *ConfigTextInput) HasChanged() bool {
	return c.value != c.originalValue
}

// Reset resets to original value
func (c *ConfigTextInput) Reset() {
	c.value = c.originalValue
	c.textInput.SetValue(c.originalValue)
	c.err = nil
}

// Apply saves the current value as the original
func (c *ConfigTextInput) Apply() {
	c.originalValue = c.value
}

// ConfigToggle represents a boolean toggle switch
type ConfigToggle struct {
	label         string
	value         bool
	originalValue bool
	enabled       bool
	focused       bool
	theme         Theme
	description   string
}

// NewConfigToggle creates a new toggle component
func NewConfigToggle(label string, value bool, theme Theme) *ConfigToggle {
	return &ConfigToggle{
		label:         label,
		value:         value,
		originalValue: value,
		enabled:       true,
		theme:         theme,
	}
}

// SetDescription sets an optional description
func (c *ConfigToggle) SetDescription(desc string) {
	c.description = desc
}

// Focus sets the focus state
func (c *ConfigToggle) Focus() {
	c.focused = true
}

// Blur removes focus
func (c *ConfigToggle) Blur() {
	c.focused = false
}

// Toggle toggles the value
func (c *ConfigToggle) Toggle() {
	if c.enabled {
		c.value = !c.value
	}
}

// View renders the toggle
func (c *ConfigToggle) View() string {
	labelStyle := c.theme.LabelStyle
	if c.focused {
		labelStyle = c.theme.FocusedStyle
	}
	if !c.enabled {
		labelStyle = c.theme.MutedStyle
	}

	// Toggle indicator
	indicator := "[ ]"
	if c.value {
		indicator = "[✓]"
	}
	
	if c.focused {
		indicator = c.theme.FocusedStyle.Render(indicator)
	} else if !c.enabled {
		indicator = c.theme.MutedStyle.Render(indicator)
	}

	label := labelStyle.Render(c.label)
	
	// Show if modified
	modified := ""
	if c.value != c.originalValue {
		modified = c.theme.WarningStyle.Render(" *")
	}

	line := fmt.Sprintf("%s %s%s", indicator, label, modified)

	// Add description if present
	if c.description != "" {
		descStyle := c.theme.MutedStyle
		if c.focused {
			descStyle = c.theme.ContentStyle
		}
		desc := descStyle.Render("  " + c.description)
		return lipgloss.JoinVertical(lipgloss.Left, line, desc)
	}

	return line
}

// HasChanged returns true if the value has changed
func (c *ConfigToggle) HasChanged() bool {
	return c.value != c.originalValue
}

// Reset resets to original value
func (c *ConfigToggle) Reset() {
	c.value = c.originalValue
}

// Apply saves the current value as the original
func (c *ConfigToggle) Apply() {
	c.originalValue = c.value
}

// ConfigNumberInput represents a number input with bounds
type ConfigNumberInput struct {
	label         string
	value         int
	originalValue int
	min           int
	max           int
	step          int
	focused       bool
	theme         Theme
	textInput     textinput.Model
	err           error
}

// NewConfigNumberInput creates a new number input component
func NewConfigNumberInput(label string, value, min, max, step int, theme Theme) *ConfigNumberInput {
	ti := textinput.New()
	ti.SetValue(strconv.Itoa(value))
	ti.CharLimit = 10

	return &ConfigNumberInput{
		label:         label,
		value:         value,
		originalValue: value,
		min:           min,
		max:           max,
		step:          step,
		textInput:     ti,
		theme:         theme,
	}
}

// Focus sets the focus state
func (c *ConfigNumberInput) Focus() {
	c.focused = true
	c.textInput.Focus()
}

// Blur removes focus
func (c *ConfigNumberInput) Blur() {
	c.focused = false
	c.textInput.Blur()
}

// Increment increases the value by step
func (c *ConfigNumberInput) Increment() {
	newVal := c.value + c.step
	if newVal <= c.max {
		c.value = newVal
		c.textInput.SetValue(strconv.Itoa(c.value))
	}
}

// Decrement decreases the value by step
func (c *ConfigNumberInput) Decrement() {
	newVal := c.value - c.step
	if newVal >= c.min {
		c.value = newVal
		c.textInput.SetValue(strconv.Itoa(c.value))
	}
}

// Update handles input events
func (c *ConfigNumberInput) Update(msg tea.Msg) (*ConfigNumberInput, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if c.focused {
			switch msg.String() {
			case "up":
				c.Increment()
				return c, nil
			case "down":
				c.Decrement()
				return c, nil
			}
		}
	}

	c.textInput, cmd = c.textInput.Update(msg)
	
	// Validate and parse
	if val, err := strconv.Atoi(c.textInput.Value()); err == nil {
		if val < c.min {
			c.err = fmt.Errorf("value must be at least %d", c.min)
		} else if val > c.max {
			c.err = fmt.Errorf("value must be at most %d", c.max)
		} else {
			c.value = val
			c.err = nil
		}
	} else {
		c.err = fmt.Errorf("invalid number")
	}

	return c, cmd
}

// View renders the number input
func (c *ConfigNumberInput) View() string {
	labelStyle := c.theme.LabelStyle
	if c.focused {
		labelStyle = c.theme.FocusedStyle
	}

	label := labelStyle.Render(fmt.Sprintf("%s (%d-%d):", c.label, c.min, c.max))
	
	inputView := c.textInput.View()

	// Show if modified
	if c.value != c.originalValue {
		inputView += c.theme.WarningStyle.Render(" *")
	}

	// Show error if present
	if c.err != nil {
		errorMsg := c.theme.ErrorStyle.Render("  " + c.err.Error())
		return lipgloss.JoinVertical(lipgloss.Left,
			label,
			inputView,
			errorMsg,
		)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		label,
		inputView,
	)
}

// HasChanged returns true if the value has changed
func (c *ConfigNumberInput) HasChanged() bool {
	return c.value != c.originalValue
}

// Reset resets to original value
func (c *ConfigNumberInput) Reset() {
	c.value = c.originalValue
	c.textInput.SetValue(strconv.Itoa(c.originalValue))
	c.err = nil
}

// Apply saves the current value as the original
func (c *ConfigNumberInput) Apply() {
	c.originalValue = c.value
}

// ConfigListInput represents a list of editable items
type ConfigListInput struct {
	label         string
	items         []string
	originalItems []string
	cursor        int
	focused       bool
	editing       bool
	editIndex     int
	theme         Theme
	textInput     textinput.Model
	maxItems      int
}

// NewConfigListInput creates a new list input component
func NewConfigListInput(label string, items []string, theme Theme) *ConfigListInput {
	ti := textinput.New()
	ti.CharLimit = 256

	// Make a copy of items for original
	originalItems := make([]string, len(items))
	copy(originalItems, items)

	return &ConfigListInput{
		label:         label,
		items:         items,
		originalItems: originalItems,
		theme:         theme,
		textInput:     ti,
		maxItems:      50,
	}
}

// Focus sets the focus state
func (c *ConfigListInput) Focus() {
	c.focused = true
}

// Blur removes focus
func (c *ConfigListInput) Blur() {
	c.focused = false
	c.editing = false
	c.textInput.Blur()
}

// Update handles input events
func (c *ConfigListInput) Update(msg tea.Msg) (*ConfigListInput, tea.Cmd) {
	if c.editing {
		var cmd tea.Cmd
		c.textInput, cmd = c.textInput.Update(msg)
		
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				// Save the edit
				if c.editIndex >= 0 && c.editIndex < len(c.items) {
					c.items[c.editIndex] = c.textInput.Value()
				} else if c.editIndex == -1 {
					// Adding new item
					c.items = append(c.items, c.textInput.Value())
				}
				c.editing = false
				c.textInput.Blur()
			case "esc":
				// Cancel edit
				c.editing = false
				c.textInput.Blur()
			}
		}
		return c, cmd
	}

	// Normal navigation
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if c.focused {
			switch msg.String() {
			case "up", "k":
				if c.cursor > 0 {
					c.cursor--
				}
			case "down", "j":
				if c.cursor < len(c.items)-1 {
					c.cursor++
				}
			case "enter":
				// Edit current item
				if c.cursor < len(c.items) {
					c.editing = true
					c.editIndex = c.cursor
					c.textInput.SetValue(c.items[c.cursor])
					c.textInput.Focus()
				}
			case "n":
				// Add new item
				if len(c.items) < c.maxItems {
					c.editing = true
					c.editIndex = -1
					c.textInput.SetValue("")
					c.textInput.Focus()
				}
			case "d":
				// Delete current item
				if c.cursor < len(c.items) {
					c.items = append(c.items[:c.cursor], c.items[c.cursor+1:]...)
					if c.cursor >= len(c.items) && c.cursor > 0 {
						c.cursor--
					}
				}
			}
		}
	}

	return c, nil
}

// View renders the list input
func (c *ConfigListInput) View() string {
	labelStyle := c.theme.LabelStyle
	if c.focused {
		labelStyle = c.theme.FocusedStyle
	}

	label := labelStyle.Render(c.label + ":")
	
	// Show if modified
	if c.HasChanged() {
		label += c.theme.WarningStyle.Render(" *")
	}

	var lines []string
	lines = append(lines, label)

	if len(c.items) == 0 {
		lines = append(lines, c.theme.MutedStyle.Render("  (empty)"))
	} else {
		for i, item := range c.items {
			prefix := "  "
			if c.focused && i == c.cursor {
				prefix = "▶ "
			}
			
			itemText := item
			if c.editing && i == c.editIndex {
				itemText = c.textInput.View()
			}
			
			line := prefix + itemText
			if c.focused && i == c.cursor && !c.editing {
				line = c.theme.SelectedStyle.Render(line)
			}
			
			lines = append(lines, line)
		}
	}

	// Add new item input if editing new
	if c.editing && c.editIndex == -1 {
		lines = append(lines, "▶ "+c.textInput.View())
	}

	// Help text
	if c.focused && !c.editing {
		help := c.theme.MutedStyle.Render("  n:add d:delete enter:edit")
		lines = append(lines, help)
	}

	return strings.Join(lines, "\n")
}

// HasChanged returns true if the items have changed
func (c *ConfigListInput) HasChanged() bool {
	if len(c.items) != len(c.originalItems) {
		return true
	}
	for i, item := range c.items {
		if item != c.originalItems[i] {
			return true
		}
	}
	return false
}

// Reset resets to original items
func (c *ConfigListInput) Reset() {
	c.items = make([]string, len(c.originalItems))
	copy(c.items, c.originalItems)
	c.cursor = 0
	c.editing = false
}

// Apply saves the current items as the original
func (c *ConfigListInput) Apply() {
	c.originalItems = make([]string, len(c.items))
	copy(c.originalItems, c.items)
}

// ConfigSection represents a visual section separator
type ConfigSection struct {
	title string
	theme Theme
}

// NewConfigSection creates a new section separator
func NewConfigSection(title string, theme Theme) *ConfigSection {
	return &ConfigSection{
		title: title,
		theme: theme,
	}
}

// View renders the section
func (s *ConfigSection) View() string {
	return s.theme.TitleStyle.Render("─── " + s.title + " ───")
}

// ConfigHelp represents help text
type ConfigHelp struct {
	text  string
	theme Theme
}

// NewConfigHelp creates a new help text component
func NewConfigHelp(text string, theme Theme) *ConfigHelp {
	return &ConfigHelp{
		text:  text,
		theme: theme,
	}
}

// View renders the help text
func (h *ConfigHelp) View() string {
	return h.theme.MutedStyle.Render(h.text)
}