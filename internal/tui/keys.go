package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// KeyHandler manages keyboard input and key bindings
type KeyHandler struct {
	bindings map[string]KeyBinding
}

// KeyBinding represents a keyboard shortcut and its action
type KeyBinding struct {
	Key         string
	Description string
	Action      KeyAction
	Context     KeyContext
}

// KeyAction represents the type of action a key performs
type KeyAction int

const (
	ActionQuit KeyAction = iota
	ActionNavigate
	ActionSelect
	ActionRefresh
	ActionCreate
	ActionDelete
	ActionEdit
	ActionToggle
	ActionMove
	ActionSearch
	ActionFilter
	ActionHelp
)

// KeyContext represents where a key binding is active
type KeyContext int

const (
	ContextGlobal KeyContext = iota
	ContextDashboard
	ContextSessions  
	ContextWorktrees
	ContextConfig
	ContextHelp
	ContextModal
)

// NewKeyHandler creates a new keyboard handler with default bindings
func NewKeyHandler() *KeyHandler {
	handler := &KeyHandler{
		bindings: make(map[string]KeyBinding),
	}
	
	handler.initializeDefaultBindings()
	return handler
}

// initializeDefaultBindings sets up the default keyboard shortcuts
func (h *KeyHandler) initializeDefaultBindings() {
	// Global bindings
	h.addBinding("q", "Quit", ActionQuit, ContextGlobal)
	h.addBinding("ctrl+c", "Quit", ActionQuit, ContextGlobal)
	h.addBinding("?", "Help", ActionHelp, ContextGlobal)
	h.addBinding("h", "Help", ActionHelp, ContextGlobal)
	h.addBinding("r", "Refresh", ActionRefresh, ContextGlobal)
	
	// Navigation bindings
	h.addBinding("1", "Dashboard", ActionNavigate, ContextGlobal)
	h.addBinding("2", "Sessions", ActionNavigate, ContextGlobal)
	h.addBinding("3", "Worktrees", ActionNavigate, ContextGlobal)
	h.addBinding("4", "Configuration", ActionNavigate, ContextGlobal)
	
	// Movement bindings (vi-style)
	h.addBinding("up", "Move up", ActionMove, ContextGlobal)
	h.addBinding("down", "Move down", ActionMove, ContextGlobal)
	h.addBinding("left", "Move left", ActionMove, ContextGlobal)
	h.addBinding("right", "Move right", ActionMove, ContextGlobal)
	h.addBinding("k", "Move up", ActionMove, ContextGlobal)
	h.addBinding("j", "Move down", ActionMove, ContextGlobal)
	h.addBinding("h", "Move left", ActionMove, ContextGlobal)
	h.addBinding("l", "Move right", ActionMove, ContextGlobal)
	
	// Action bindings
	h.addBinding("enter", "Select/Confirm", ActionSelect, ContextGlobal)
	h.addBinding(" ", "Select/Toggle", ActionToggle, ContextGlobal)
	h.addBinding("n", "New/Create", ActionCreate, ContextGlobal)
	h.addBinding("d", "Delete", ActionDelete, ContextGlobal)
	h.addBinding("e", "Edit", ActionEdit, ContextGlobal)
	h.addBinding("/", "Search", ActionSearch, ContextGlobal)
	h.addBinding("f", "Filter", ActionFilter, ContextGlobal)
	
	// Context-specific bindings
	
	// Dashboard context
	h.addBinding("s", "New Session", ActionCreate, ContextDashboard)
	h.addBinding("w", "New Worktree", ActionCreate, ContextDashboard)
	
	// Sessions context
	h.addBinding("a", "Attach Session", ActionSelect, ContextSessions)
	h.addBinding("k", "Kill Session", ActionDelete, ContextSessions)
	h.addBinding("r", "Rename Session", ActionEdit, ContextSessions)
	
	// Worktrees context
	h.addBinding("o", "Open Worktree", ActionSelect, ContextWorktrees)
	h.addBinding("p", "Prune Worktrees", ActionDelete, ContextWorktrees)
	h.addBinding("b", "Change Branch", ActionEdit, ContextWorktrees)
	
	// Config context
	h.addBinding("e", "Edit Config", ActionEdit, ContextConfig)
	h.addBinding("r", "Reload Config", ActionRefresh, ContextConfig)
	h.addBinding("s", "Save Config", ActionSelect, ContextConfig)
}

// addBinding adds a new key binding
func (h *KeyHandler) addBinding(key, description string, action KeyAction, context KeyContext) {
	h.bindings[key] = KeyBinding{
		Key:         key,
		Description: description,
		Action:      action,
		Context:     context,
	}
}

// GetBinding returns the key binding for a given key
func (h *KeyHandler) GetBinding(key string) (KeyBinding, bool) {
	binding, exists := h.bindings[key]
	return binding, exists
}

// GetBindingsForContext returns all key bindings for a specific context
func (h *KeyHandler) GetBindingsForContext(context KeyContext) []KeyBinding {
	var bindings []KeyBinding
	
	for _, binding := range h.bindings {
		if binding.Context == context || binding.Context == ContextGlobal {
			bindings = append(bindings, binding)
		}
	}
	
	return bindings
}

// HandleKeyPress processes a key press and returns the appropriate action
func (h *KeyHandler) HandleKeyPress(msg tea.KeyMsg, context KeyContext) (KeyAction, bool) {
	key := msg.String()
	
	// Check for context-specific binding first
	if binding, exists := h.bindings[key]; exists {
		if binding.Context == context || binding.Context == ContextGlobal {
			return binding.Action, true
		}
	}
	
	return ActionQuit, false // Default action if no binding found
}

// GetHelpText returns formatted help text for a given context
func (h *KeyHandler) GetHelpText(context KeyContext) []string {
	bindings := h.GetBindingsForContext(context)
	var helpLines []string
	
	// Group bindings by action type
	actionGroups := make(map[KeyAction][]KeyBinding)
	for _, binding := range bindings {
		actionGroups[binding.Action] = append(actionGroups[binding.Action], binding)
	}
	
	// Format help text by action groups
	if navBindings, exists := actionGroups[ActionNavigate]; exists {
		helpLines = append(helpLines, "Navigation:")
		for _, binding := range navBindings {
			helpLines = append(helpLines, fmt.Sprintf("  %s: %s", binding.Key, binding.Description))
		}
		helpLines = append(helpLines, "")
	}
	
	if moveBindings, exists := actionGroups[ActionMove]; exists {
		helpLines = append(helpLines, "Movement:")
		for _, binding := range moveBindings {
			helpLines = append(helpLines, fmt.Sprintf("  %s: %s", binding.Key, binding.Description))
		}
		helpLines = append(helpLines, "")
	}
	
	if actionBindings, exists := actionGroups[ActionSelect]; exists {
		helpLines = append(helpLines, "Actions:")
		for _, binding := range actionBindings {
			helpLines = append(helpLines, fmt.Sprintf("  %s: %s", binding.Key, binding.Description))
		}
		helpLines = append(helpLines, "")
	}
	
	return helpLines
}

// IsQuitKey checks if the pressed key is a quit key
func (h *KeyHandler) IsQuitKey(key string) bool {
	binding, exists := h.bindings[key]
	return exists && binding.Action == ActionQuit
}

// IsNavigationKey checks if the pressed key is a navigation key
func (h *KeyHandler) IsNavigationKey(key string) bool {
	binding, exists := h.bindings[key]
	return exists && binding.Action == ActionNavigate
}

// IsMovementKey checks if the pressed key is a movement key
func (h *KeyHandler) IsMovementKey(key string) bool {
	binding, exists := h.bindings[key]
	return exists && binding.Action == ActionMove
}

// GetNavigationTarget returns the screen number for navigation keys
func (h *KeyHandler) GetNavigationTarget(key string) AppScreen {
	switch key {
	case "1":
		return ScreenDashboard
	case "2":
		return ScreenSessions
	case "3":
		return ScreenWorktrees
	case "4":
		return ScreenConfig
	default:
		return ScreenDashboard
	}
}

// GetMovementDirection returns the direction for movement keys
func (h *KeyHandler) GetMovementDirection(key string) MovementDirection {
	switch key {
	case "up", "k":
		return DirectionUp
	case "down", "j":
		return DirectionDown
	case "left", "h":
		return DirectionLeft
	case "right", "l":
		return DirectionRight
	default:
		return DirectionNone
	}
}

// MovementDirection represents cursor movement directions
type MovementDirection int

const (
	DirectionNone MovementDirection = iota
	DirectionUp
	DirectionDown
	DirectionLeft
	DirectionRight
)

// String returns the string representation of MovementDirection
func (d MovementDirection) String() string {
	switch d {
	case DirectionUp:
		return "up"
	case DirectionDown:
		return "down"
	case DirectionLeft:
		return "left"
	case DirectionRight:
		return "right"
	default:
		return "none"
	}
}

// KeyBindingHelp provides helper methods for displaying key bindings
type KeyBindingHelp struct {
	handler *KeyHandler
}

// NewKeyBindingHelp creates a new key binding help utility
func NewKeyBindingHelp(handler *KeyHandler) *KeyBindingHelp {
	return &KeyBindingHelp{
		handler: handler,
	}
}

// FormatBindings returns formatted key bindings for display
func (h *KeyBindingHelp) FormatBindings(context KeyContext) []string {
	return h.handler.GetHelpText(context)
}

// GetQuickHelp returns a short list of the most important key bindings
func (h *KeyBindingHelp) GetQuickHelp(context KeyContext) []string {
	switch context {
	case ContextDashboard:
		return []string{"1-4: Navigate", "r: Refresh", "n: New session", "q: Quit"}
	case ContextSessions:
		return []string{"↑/↓: Select", "Enter: Attach", "n: New", "d: Delete"}
	case ContextWorktrees:
		return []string{"↑/↓: Select", "Enter: Open", "n: New", "d: Delete"}
	case ContextConfig:
		return []string{"e: Edit", "r: Reload", "s: Save", "q: Back"}
	case ContextHelp:
		return []string{"q: Back to previous screen"}
	default:
		return []string{"1-4: Navigate", "?: Help", "q: Quit"}
	}
}