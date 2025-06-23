package context

// SessionContextMenu creates context menus for session-related operations
type SessionContextMenu struct {
	theme Theme
}

// NewSessionContextMenu creates a new session context menu provider
func NewSessionContextMenu(theme Theme) *SessionContextMenu {
	return &SessionContextMenu{
		theme: theme,
	}
}

// SessionInfo represents session information for context menu generation
type SessionInfo struct {
	ID         string
	Name       string
	Active     bool
	Project    string
	Branch     string
	Directory  string
	HasChanges bool
	LastAccess string
}

// CreateSessionListMenu creates a context menu for the session list
func (s *SessionContextMenu) CreateSessionListMenu() *ContextMenu {
	items := []ContextMenuItem{
		NewMenuItemWithIcon("New Session", "session_new", "n", "➕"),
		NewMenuItemWithIcon("Refresh List", "session_refresh", "r", "🔄"),
		NewMenuDivider(),
		NewMenuItemWithIcon("Import Session", "session_import", "i", "📥"),
		NewMenuItemWithIcon("Export Sessions", "session_export", "e", "📤"),
		NewMenuDivider(),
		NewMenuItemWithIcon("Session Templates", "session_templates", "t", "📋"),
		NewMenuItemWithIcon("Bulk Operations", "session_bulk", "b", "⚙️"),
	}

	return NewContextMenu(ContextMenuConfig{
		Title: "Session Actions",
		Items: items,
	}, s.theme)
}

// CreateSessionItemMenu creates a context menu for a specific session
func (s *SessionContextMenu) CreateSessionItemMenu(session SessionInfo) *ContextMenu {
	var items []ContextMenuItem

	if session.Active {
		items = append(items,
			NewMenuItemWithIcon("Attach", "session_attach", "a", "🔗"),
			NewMenuItemWithIcon("Detach", "session_detach", "d", "🔓"),
			NewMenuDivider(),
		)
	} else {
		items = append(items,
			NewMenuItemWithIcon("Start", "session_start", "s", "▶️"),
			NewMenuDivider(),
		)
	}

	// Common actions
	items = append(items,
		NewMenuItemWithIcon("Rename", "session_rename", "r", "✏️"),
		NewMenuItemWithIcon("Clone", "session_clone", "c", "📋"),
		NewMenuDivider(),
		NewMenuItemWithIcon("Show Info", "session_info", "i", "ℹ️"),
		NewMenuItemWithIcon("Show Logs", "session_logs", "l", "📄"),
		NewMenuDivider(),
	)

	// Directory operations
	directorySubmenu := &ContextMenu{
		title: "Directory",
		items: []ContextMenuItem{
			NewMenuItemWithIcon("Open in Editor", "session_open_editor", "e", "📝"),
			NewMenuItemWithIcon("Open in Terminal", "session_open_terminal", "t", "🖥️"),
			NewMenuItemWithIcon("Open in Finder", "session_open_finder", "f", "📁"),
			NewMenuDivider(),
			NewMenuItemWithIcon("Show Git Status", "session_git_status", "g", "📊"),
		},
		theme: s.theme,
	}

	items = append(items, ContextMenuItem{
		Label:   "Directory Actions",
		Action:  "",
		Icon:    "📁",
		Enabled: true,
		Submenu: directorySubmenu,
	})

	// Danger zone
	items = append(items,
		NewMenuDivider(),
		ContextMenuItem{
			Label:   "Kill Session",
			Action:  "session_kill",
			Key:     "k",
			Icon:    "💀",
			Enabled: true,
		},
		ContextMenuItem{
			Label:   "Delete Session",
			Action:  "session_delete",
			Key:     "x",
			Icon:    "🗑️",
			Enabled: true,
		},
	)

	return NewContextMenu(ContextMenuConfig{
		Title: session.Name,
		Items: items,
	}, s.theme)
}

// CreateNewSessionMenu creates a context menu for new session options
func (s *SessionContextMenu) CreateNewSessionMenu() *ContextMenu {
	items := []ContextMenuItem{
		NewMenuItemWithIcon("From Current Directory", "session_new_current", "c", "📁"),
		NewMenuItemWithIcon("From Project", "session_new_project", "p", "📦"),
		NewMenuItemWithIcon("From Worktree", "session_new_worktree", "w", "🌳"),
		NewMenuDivider(),
		NewMenuItemWithIcon("From Template", "session_new_template", "t", "📋"),
		NewMenuItemWithIcon("Import from File", "session_new_import", "i", "📥"),
		NewMenuDivider(),
		NewMenuItemWithIcon("Custom Setup Wizard", "session_new_wizard", "z", "🧙"),
	}

	return NewContextMenu(ContextMenuConfig{
		Title: "New Session",
		Items: items,
	}, s.theme)
}

// CreateBulkSessionMenu creates a context menu for bulk session operations
func (s *SessionContextMenu) CreateBulkSessionMenu() *ContextMenu {
	items := []ContextMenuItem{
		NewMenuItemWithIcon("Start All Inactive", "session_bulk_start", "s", "▶️"),
		NewMenuItemWithIcon("Stop All Active", "session_bulk_stop", "x", "⏹️"),
		NewMenuDivider(),
		NewMenuItemWithIcon("Export Selected", "session_bulk_export", "e", "📤"),
		NewMenuItemWithIcon("Clone Selected", "session_bulk_clone", "c", "📋"),
		NewMenuDivider(),
		NewMenuItemWithIcon("Update All Configs", "session_bulk_update", "u", "🔄"),
		NewMenuItemWithIcon("Cleanup Orphaned", "session_bulk_cleanup", "l", "🧹"),
		NewMenuDivider(),
		ContextMenuItem{
			Label:   "Delete Selected",
			Action:  "session_bulk_delete",
			Key:     "d",
			Icon:    "🗑️",
			Enabled: true,
		},
	}

	return NewContextMenu(ContextMenuConfig{
		Title: "Bulk Operations",
		Items: items,
	}, s.theme)
}

// CreateSessionTemplateMenu creates a context menu for session templates
func (s *SessionContextMenu) CreateSessionTemplateMenu() *ContextMenu {
	items := []ContextMenuItem{
		NewMenuItemWithIcon("Create Template", "template_create", "c", "➕"),
		NewMenuItemWithIcon("Edit Template", "template_edit", "e", "✏️"),
		NewMenuDivider(),
		NewMenuItemWithIcon("Import Template", "template_import", "i", "📥"),
		NewMenuItemWithIcon("Export Template", "template_export", "x", "📤"),
		NewMenuDivider(),
		NewMenuItemWithIcon("Apply Template", "template_apply", "a", "🎯"),
		ContextMenuItem{
			Label:   "Delete Template",
			Action:  "template_delete",
			Key:     "d",
			Icon:    "🗑️",
			Enabled: true,
		},
	}

	return NewContextMenu(ContextMenuConfig{
		Title: "Templates",
		Items: items,
	}, s.theme)
}
