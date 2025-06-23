package context

// ConfigContextMenu creates context menus for configuration operations
type ConfigContextMenu struct {
	theme Theme
}

// NewConfigContextMenu creates a new configuration context menu provider
func NewConfigContextMenu(theme Theme) *ConfigContextMenu {
	return &ConfigContextMenu{
		theme: theme,
	}
}

// ConfigSection represents a configuration section
type ConfigSection struct {
	Name        string
	Key         string
	Description string
	Type        string
	Valid       bool
	Modified    bool
	Default     bool
}

// CreateConfigMainMenu creates the main configuration screen context menu
func (c *ConfigContextMenu) CreateConfigMainMenu() *ContextMenu {
	items := []ContextMenuItem{
		NewMenuItemWithIcon("Edit Configuration", "config_edit", "e", "✏️"),
		NewMenuItemWithIcon("Reload Config", "config_reload", "r", "🔄"),
		NewMenuDivider(),
		NewMenuItemWithIcon("Import Config", "config_import", "i", "📥"),
		NewMenuItemWithIcon("Export Config", "config_export", "x", "📤"),
		NewMenuDivider(),
		NewMenuItemWithIcon("Validate Config", "config_validate", "v", "✅"),
		NewMenuItemWithIcon("Show Schema", "config_schema", "s", "📋"),
		NewMenuDivider(),
		NewMenuItemWithIcon("Reset to Defaults", "config_reset", "d", "🔄"),
		NewMenuItemWithIcon("Backup Config", "config_backup", "b", "💾"),
	}

	return NewContextMenu(ContextMenuConfig{
		Title: "Configuration",
		Items: items,
	}, c.theme)
}

// CreateConfigSectionMenu creates a context menu for a specific configuration section
func (c *ConfigContextMenu) CreateConfigSectionMenu(section ConfigSection) *ContextMenu {
	var items []ContextMenuItem

	// Basic operations
	items = append(items,
		NewMenuItemWithIcon("Edit Section", "config_edit_section", "e", "✏️"),
		NewMenuItemWithIcon("View Details", "config_view_section", "v", "👁️"),
		NewMenuDivider(),
	)

	// Validation status
	if !section.Valid {
		items = append(items,
			NewMenuItemWithIcon("Show Errors", "config_show_errors", "r", "❌"),
			NewMenuItemWithIcon("Fix Errors", "config_fix_errors", "f", "🔧"),
			NewMenuDivider(),
		)
	}

	// Reset options
	if section.Modified {
		items = append(items,
			NewMenuItemWithIcon("Reset Section", "config_reset_section", "r", "🔄"),
			NewMenuItemWithIcon("Show Changes", "config_show_changes", "c", "📝"),
			NewMenuDivider(),
		)
	}

	// Advanced operations based on section type
	switch section.Type {
	case "tui":
		tuiSubmenu := c.createTUISubmenu()
		items = append(items, ContextMenuItem{
			Label:   "TUI Settings",
			Action:  "",
			Icon:    "🖥️",
			Enabled: true,
			Submenu: tuiSubmenu,
		})

	case "claude":
		claudeSubmenu := c.createClaudeSubmenu()
		items = append(items, ContextMenuItem{
			Label:   "Claude Settings",
			Action:  "",
			Icon:    "🤖",
			Enabled: true,
			Submenu: claudeSubmenu,
		})

	case "git":
		gitSubmenu := c.createGitConfigSubmenu()
		items = append(items, ContextMenuItem{
			Label:   "Git Settings",
			Action:  "",
			Icon:    "📊",
			Enabled: true,
			Submenu: gitSubmenu,
		})

	case "tmux":
		tmuxSubmenu := c.createTmuxSubmenu()
		items = append(items, ContextMenuItem{
			Label:   "Tmux Settings",
			Action:  "",
			Icon:    "🖥️",
			Enabled: true,
			Submenu: tmuxSubmenu,
		})
	}

	// Help and documentation
	items = append(items,
		NewMenuDivider(),
		NewMenuItemWithIcon("Show Help", "config_help_section", "h", "❓"),
		NewMenuItemWithIcon("Show Examples", "config_examples", "x", "📖"),
	)

	return NewContextMenu(ContextMenuConfig{
		Title: section.Name,
		Items: items,
	}, c.theme)
}

// createTUISubmenu creates a submenu for TUI-specific settings
func (c *ConfigContextMenu) createTUISubmenu() *ContextMenu {
	items := []ContextMenuItem{
		NewMenuItemWithIcon("Theme Settings", "config_tui_theme", "t", "🎨"),
		NewMenuItemWithIcon("Keyboard Shortcuts", "config_tui_keys", "k", "⌨️"),
		NewMenuItemWithIcon("Layout Options", "config_tui_layout", "l", "📐"),
		NewMenuDivider(),
		NewMenuItemWithIcon("Refresh Interval", "config_tui_refresh", "r", "🔄"),
		NewMenuItemWithIcon("Animation Settings", "config_tui_animation", "a", "✨"),
		NewMenuDivider(),
		NewMenuItemWithIcon("Reset TUI Settings", "config_tui_reset", "x", "🔄"),
	}

	return &ContextMenu{
		title: "TUI Settings",
		items: items,
		theme: c.theme,
	}
}

// createClaudeSubmenu creates a submenu for Claude-specific settings
func (c *ConfigContextMenu) createClaudeSubmenu() *ContextMenu {
	items := []ContextMenuItem{
		NewMenuItemWithIcon("API Configuration", "config_claude_api", "a", "🔑"),
		NewMenuItemWithIcon("Model Settings", "config_claude_model", "m", "🧠"),
		NewMenuItemWithIcon("MCP Servers", "config_claude_mcp", "c", "🔌"),
		NewMenuDivider(),
		NewMenuItemWithIcon("Permissions", "config_claude_permissions", "p", "🛡️"),
		NewMenuItemWithIcon("Rate Limits", "config_claude_limits", "l", "⏱️"),
		NewMenuDivider(),
		NewMenuItemWithIcon("Test Connection", "config_claude_test", "t", "🔍"),
		NewMenuItemWithIcon("Reset Claude Config", "config_claude_reset", "r", "🔄"),
	}

	return &ContextMenu{
		title: "Claude Settings",
		items: items,
		theme: c.theme,
	}
}

// createGitConfigSubmenu creates a submenu for Git-specific settings
func (c *ConfigContextMenu) createGitConfigSubmenu() *ContextMenu {
	items := []ContextMenuItem{
		NewMenuItemWithIcon("Worktree Patterns", "config_git_patterns", "p", "🌳"),
		NewMenuItemWithIcon("Branch Settings", "config_git_branches", "b", "🌿"),
		NewMenuItemWithIcon("Remote Settings", "config_git_remotes", "r", "🌐"),
		NewMenuDivider(),
		NewMenuItemWithIcon("Hooks Configuration", "config_git_hooks", "h", "🪝"),
		NewMenuItemWithIcon("Ignore Patterns", "config_git_ignore", "i", "🚫"),
		NewMenuDivider(),
		NewMenuItemWithIcon("Validate Git Config", "config_git_validate", "v", "✅"),
		NewMenuItemWithIcon("Reset Git Settings", "config_git_reset", "x", "🔄"),
	}

	return &ContextMenu{
		title: "Git Settings",
		items: items,
		theme: c.theme,
	}
}

// createTmuxSubmenu creates a submenu for Tmux-specific settings
func (c *ConfigContextMenu) createTmuxSubmenu() *ContextMenu {
	items := []ContextMenuItem{
		NewMenuItemWithIcon("Session Templates", "config_tmux_templates", "t", "📋"),
		NewMenuItemWithIcon("Window Settings", "config_tmux_windows", "w", "🪟"),
		NewMenuItemWithIcon("Pane Settings", "config_tmux_panes", "p", "📱"),
		NewMenuDivider(),
		NewMenuItemWithIcon("Key Bindings", "config_tmux_keys", "k", "⌨️"),
		NewMenuItemWithIcon("Status Bar", "config_tmux_status", "s", "📊"),
		NewMenuDivider(),
		NewMenuItemWithIcon("Test Tmux Config", "config_tmux_test", "e", "🔍"),
		NewMenuItemWithIcon("Reset Tmux Settings", "config_tmux_reset", "r", "🔄"),
	}

	return &ContextMenu{
		title: "Tmux Settings",
		items: items,
		theme: c.theme,
	}
}

// CreateConfigEditorMenu creates a context menu for the configuration editor
func (c *ConfigContextMenu) CreateConfigEditorMenu() *ContextMenu {
	items := []ContextMenuItem{
		NewMenuItemWithIcon("Save Changes", "config_save", "s", "💾"),
		NewMenuItemWithIcon("Discard Changes", "config_discard", "d", "❌"),
		NewMenuDivider(),
		NewMenuItemWithIcon("Validate", "config_validate_editor", "v", "✅"),
		NewMenuItemWithIcon("Format", "config_format", "f", "📐"),
		NewMenuDivider(),
		NewMenuItemWithIcon("Find/Replace", "config_find", "/", "🔍"),
		NewMenuItemWithIcon("Go to Line", "config_goto", "g", "➡️"),
		NewMenuDivider(),
		NewMenuItemWithIcon("Show Schema", "config_schema_editor", "h", "📋"),
		NewMenuItemWithIcon("Insert Template", "config_template", "t", "📄"),
	}

	return NewContextMenu(ContextMenuConfig{
		Title: "Editor",
		Items: items,
	}, c.theme)
}

// CreateConfigValidationMenu creates a context menu for validation results
func (c *ConfigContextMenu) CreateConfigValidationMenu() *ContextMenu {
	items := []ContextMenuItem{
		NewMenuItemWithIcon("Show All Errors", "config_show_all_errors", "e", "❌"),
		NewMenuItemWithIcon("Show Warnings", "config_show_warnings", "w", "⚠️"),
		NewMenuDivider(),
		NewMenuItemWithIcon("Auto-fix Issues", "config_autofix", "f", "🔧"),
		NewMenuItemWithIcon("Ignore Warnings", "config_ignore_warnings", "i", "🙈"),
		NewMenuDivider(),
		NewMenuItemWithIcon("Validate Again", "config_revalidate", "r", "🔄"),
		NewMenuItemWithIcon("Export Report", "config_export_report", "x", "📤"),
	}

	return NewContextMenu(ContextMenuConfig{
		Title: "Validation",
		Items: items,
	}, c.theme)
}

// CreateConfigBackupMenu creates a context menu for configuration backup operations
func (c *ConfigContextMenu) CreateConfigBackupMenu() *ContextMenu {
	items := []ContextMenuItem{
		NewMenuItemWithIcon("Create Backup", "config_backup_create", "c", "💾"),
		NewMenuItemWithIcon("Restore Backup", "config_backup_restore", "r", "📥"),
		NewMenuDivider(),
		NewMenuItemWithIcon("List Backups", "config_backup_list", "l", "📋"),
		NewMenuItemWithIcon("Delete Backup", "config_backup_delete", "d", "🗑️"),
		NewMenuDivider(),
		NewMenuItemWithIcon("Auto-backup Settings", "config_backup_auto", "a", "⚙️"),
		NewMenuItemWithIcon("Backup Location", "config_backup_location", "p", "📁"),
	}

	return NewContextMenu(ContextMenuConfig{
		Title: "Backup",
		Items: items,
	}, c.theme)
}
