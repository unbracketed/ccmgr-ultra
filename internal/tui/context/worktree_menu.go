package context

// WorktreeContextMenu creates context menus for worktree-related operations
type WorktreeContextMenu struct {
	theme Theme
}

// NewWorktreeContextMenu creates a new worktree context menu provider
func NewWorktreeContextMenu(theme Theme) *WorktreeContextMenu {
	return &WorktreeContextMenu{
		theme: theme,
	}
}

// WorktreeInfo represents worktree information for context menu generation
type WorktreeInfo struct {
	Path         string
	Branch       string
	ProjectName  string
	HasChanges   bool
	UpstreamSync bool
	LastAccess   string
	IsMain       bool
	ConflictState string
}

// CreateWorktreeListMenu creates a context menu for the worktree list
func (w *WorktreeContextMenu) CreateWorktreeListMenu() *ContextMenu {
	items := []ContextMenuItem{
		NewMenuItemWithIcon("New Worktree", "worktree_new", "n", "🌱"),
		NewMenuItemWithIcon("Refresh List", "worktree_refresh", "r", "🔄"),
		NewMenuDivider(),
		NewMenuItemWithIcon("Sync All", "worktree_sync_all", "s", "🔄"),
		NewMenuItemWithIcon("Check Status", "worktree_status_all", "t", "📊"),
		NewMenuDivider(),
		NewMenuItemWithIcon("Prune Worktrees", "worktree_prune", "p", "🧹"),
		NewMenuItemWithIcon("Repair Worktrees", "worktree_repair", "x", "🔧"),
	}
	
	return NewContextMenu(ContextMenuConfig{
		Title: "Worktree Actions",
		Items: items,
	}, w.theme)
}

// CreateWorktreeItemMenu creates a context menu for a specific worktree
func (w *WorktreeContextMenu) CreateWorktreeItemMenu(worktree WorktreeInfo) *ContextMenu {
	var items []ContextMenuItem
	
	// Navigation actions
	items = append(items,
		NewMenuItemWithIcon("Open", "worktree_open", "o", "📁"),
		NewMenuItemWithIcon("Open in Editor", "worktree_open_editor", "e", "📝"),
		NewMenuItemWithIcon("Open Terminal", "worktree_open_terminal", "t", "🖥️"),
		NewMenuDivider(),
	)
	
	// Session management
	sessionSubmenu := &ContextMenu{
		title: "Session",
		items: []ContextMenuItem{
			NewMenuItemWithIcon("Create Session", "worktree_create_session", "c", "➕"),
			NewMenuItemWithIcon("Attach Session", "worktree_attach_session", "a", "🔗"),
			NewMenuItemWithIcon("Find Sessions", "worktree_find_sessions", "f", "🔍"),
		},
		theme: w.theme,
	}
	
	items = append(items, ContextMenuItem{
		Label:   "Session Actions",
		Action:  "",
		Icon:    "🖥️",
		Enabled: true,
		Submenu: sessionSubmenu,
	})
	
	// Git operations
	gitSubmenu := w.createGitSubmenu(worktree)
	items = append(items, ContextMenuItem{
		Label:   "Git Operations",
		Action:  "",
		Icon:    "📊",
		Enabled: true,
		Submenu: gitSubmenu,
	})
	
	// Branch operations
	branchSubmenu := w.createBranchSubmenu(worktree)
	items = append(items, ContextMenuItem{
		Label:   "Branch Operations",
		Action:  "",
		Icon:    "🌿",
		Enabled: true,
		Submenu: branchSubmenu,
	})
	
	items = append(items, NewMenuDivider())
	
	// File operations
	fileSubmenu := &ContextMenu{
		title: "Files",
		items: []ContextMenuItem{
			NewMenuItemWithIcon("Show Changes", "worktree_show_changes", "c", "📝"),
			NewMenuItemWithIcon("Stash Changes", "worktree_stash", "s", "📦"),
			NewMenuItemWithIcon("Discard Changes", "worktree_discard", "d", "🗑️"),
			NewMenuDivider(),
			NewMenuItemWithIcon("Add All", "worktree_add_all", "a", "➕"),
			NewMenuItemWithIcon("Commit", "worktree_commit", "m", "💾"),
		},
		theme: w.theme,
	}
	
	items = append(items, ContextMenuItem{
		Label:   "File Operations",
		Action:  "",
		Icon:    "📄",
		Enabled: true,
		Submenu: fileSubmenu,
	})
	
	// Maintenance operations
	items = append(items,
		NewMenuDivider(),
		NewMenuItemWithIcon("Sync Worktree", "worktree_sync", "y", "🔄"),
		NewMenuItemWithIcon("Repair", "worktree_repair_single", "r", "🔧"),
		NewMenuItemWithIcon("Show Info", "worktree_info", "i", "ℹ️"),
	)
	
	// Danger zone - only show delete if not main worktree
	if !worktree.IsMain {
		items = append(items,
			NewMenuDivider(),
			ContextMenuItem{
				Label:   "Remove Worktree",
				Action:  "worktree_remove",
				Key:     "x",
				Icon:    "🗑️",
				Enabled: true,
			},
		)
	}
	
	return NewContextMenu(ContextMenuConfig{
		Title: worktree.Branch,
		Items: items,
	}, w.theme)
}

// createGitSubmenu creates a submenu for Git operations
func (w *WorktreeContextMenu) createGitSubmenu(worktree WorktreeInfo) *ContextMenu {
	items := []ContextMenuItem{
		NewMenuItemWithIcon("Status", "git_status", "s", "📊"),
		NewMenuItemWithIcon("Log", "git_log", "l", "📋"),
		NewMenuItemWithIcon("Diff", "git_diff", "d", "🔍"),
		NewMenuDivider(),
		NewMenuItemWithIcon("Fetch", "git_fetch", "f", "⬇️"),
		NewMenuItemWithIcon("Pull", "git_pull", "p", "⬇️"),
		NewMenuItemWithIcon("Push", "git_push", "u", "⬆️"),
		NewMenuDivider(),
	}
	
	// Add conflict resolution if in conflict state
	if worktree.ConflictState != "" {
		items = append(items,
			NewMenuItemWithIcon("Resolve Conflicts", "git_resolve", "r", "⚔️"),
			NewMenuItemWithIcon("Abort Merge", "git_abort", "a", "❌"),
			NewMenuDivider(),
		)
	}
	
	items = append(items,
		NewMenuItemWithIcon("Reset Hard", "git_reset_hard", "h", "🔄"),
		NewMenuItemWithIcon("Clean", "git_clean", "c", "🧹"),
	)
	
	return &ContextMenu{
		title: "Git Operations",
		items: items,
		theme: w.theme,
	}
}

// createBranchSubmenu creates a submenu for branch operations
func (w *WorktreeContextMenu) createBranchSubmenu(worktree WorktreeInfo) *ContextMenu {
	items := []ContextMenuItem{
		NewMenuItemWithIcon("Create Branch", "branch_create", "c", "🌱"),
		NewMenuItemWithIcon("Switch Branch", "branch_switch", "s", "🔄"),
		NewMenuDivider(),
		NewMenuItemWithIcon("Merge Into", "branch_merge_into", "m", "🔀"),
		NewMenuItemWithIcon("Rebase Onto", "branch_rebase", "r", "📈"),
		NewMenuDivider(),
		NewMenuItemWithIcon("Compare Branches", "branch_compare", "d", "🔍"),
		NewMenuItemWithIcon("Show Upstream", "branch_upstream", "u", "⬆️"),
		NewMenuDivider(),
	}
	
	// Only allow branch deletion if not main worktree
	if !worktree.IsMain {
		items = append(items,
			ContextMenuItem{
				Label:   "Delete Branch",
				Action:  "branch_delete",
				Key:     "x",
				Icon:    "🗑️",
				Enabled: true,
			},
		)
	}
	
	return &ContextMenu{
		title: "Branch Operations",
		items: items,
		theme: w.theme,
	}
}

// CreateNewWorktreeMenu creates a context menu for new worktree options
func (w *WorktreeContextMenu) CreateNewWorktreeMenu() *ContextMenu {
	items := []ContextMenuItem{
		NewMenuItemWithIcon("From Existing Branch", "worktree_new_existing", "e", "🌿"),
		NewMenuItemWithIcon("From New Branch", "worktree_new_branch", "n", "🌱"),
		NewMenuDivider(),
		NewMenuItemWithIcon("From Remote Branch", "worktree_new_remote", "r", "🌐"),
		NewMenuItemWithIcon("From Tag", "worktree_new_tag", "t", "🏷️"),
		NewMenuDivider(),
		NewMenuItemWithIcon("Setup Wizard", "worktree_new_wizard", "w", "🧙"),
	}
	
	return NewContextMenu(ContextMenuConfig{
		Title: "New Worktree",
		Items: items,
	}, w.theme)
}

// CreateWorktreeBulkMenu creates a context menu for bulk worktree operations
func (w *WorktreeContextMenu) CreateWorktreeBulkMenu() *ContextMenu {
	items := []ContextMenuItem{
		NewMenuItemWithIcon("Sync All", "worktree_bulk_sync", "s", "🔄"),
		NewMenuItemWithIcon("Fetch All", "worktree_bulk_fetch", "f", "⬇️"),
		NewMenuDivider(),
		NewMenuItemWithIcon("Status All", "worktree_bulk_status", "t", "📊"),
		NewMenuItemWithIcon("Clean All", "worktree_bulk_clean", "c", "🧹"),
		NewMenuDivider(),
		NewMenuItemWithIcon("Prune Worktrees", "worktree_bulk_prune", "p", "✂️"),
		NewMenuItemWithIcon("Repair All", "worktree_bulk_repair", "r", "🔧"),
		NewMenuDivider(),
		ContextMenuItem{
			Label:   "Remove Selected",
			Action:  "worktree_bulk_remove",
			Key:     "x",
			Icon:    "🗑️",
			Enabled: true,
		},
	}
	
	return NewContextMenu(ContextMenuConfig{
		Title: "Bulk Operations",
		Items: items,
	}, w.theme)
}