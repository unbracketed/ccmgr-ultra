package context

import "fmt"

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
	Path          string
	Branch        string
	ProjectName   string
	HasChanges    bool
	UpstreamSync  bool
	LastAccess    string
	IsMain        bool
	ConflictState string
	
	// New session-related fields
	ActiveSessions []SessionSummary
	ClaudeStatus   string  // idle, busy, waiting, error
	HasSessions    bool
}

// SessionSummary provides session info for context menus
type SessionSummary struct {
	ID       string
	Name     string
	State    string
	LastUsed string
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
	
	// Session management - now context-aware
	sessionSubmenu := w.createSessionSubmenu(worktree)
	
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

// createSessionSubmenu creates a context-aware session management submenu
func (w *WorktreeContextMenu) createSessionSubmenu(worktree WorktreeInfo) *ContextMenu {
	var items []ContextMenuItem
	
	// Show existing sessions if any
	if len(worktree.ActiveSessions) > 0 {
		items = append(items, 
			NewMenuItemWithIcon("Active Sessions", "", "", "📋"),
			NewMenuDivider(),
		)
		
		// List each active session with actions
		for _, session := range worktree.ActiveSessions {
			sessionIcon := "🖥️"
			if session.State == "active" {
				sessionIcon = "🟢"
			} else if session.State == "paused" {
				sessionIcon = "⏸️"
			}
			
			items = append(items, ContextMenuItem{
				Label:   fmt.Sprintf("%s (%s)", session.Name, session.State),
				Action:  fmt.Sprintf("session_attach_%s", session.ID),
				Key:     "",
				Icon:    sessionIcon,
				Enabled: true,
			})
		}
		
		items = append(items, NewMenuDivider())
	}
	
	// Session creation actions
	items = append(items,
		NewMenuItemWithIcon("New Session", "session_new", "n", "➕"),
	)
	
	// Context-aware session actions based on worktree state
	if len(worktree.ActiveSessions) > 0 {
		items = append(items,
			NewMenuItemWithIcon("Continue Session", "session_continue", "c", "▶️"),
			NewMenuItemWithIcon("Resume Session", "session_resume", "r", "🔄"),
		)
		
		// Claude-specific actions based on status
		if worktree.ClaudeStatus == "busy" {
			items = append(items,
				NewMenuItemWithIcon("Pause Claude", "claude_pause", "p", "⏸️"),
			)
		} else if worktree.ClaudeStatus == "error" {
			items = append(items,
				NewMenuItemWithIcon("Restart Claude", "claude_restart", "x", "🔄"),
			)
		}
	} else {
		// No active sessions
		items = append(items,
			NewMenuItemWithIcon("Find Sessions", "session_find", "f", "🔍"),
			NewMenuItemWithIcon("Restore Session", "session_restore", "r", "📂"),
		)
	}
	
	items = append(items, NewMenuDivider())
	
	// Session management actions
	sessionManagementItems := []ContextMenuItem{
		NewMenuItemWithIcon("List All Sessions", "session_list_all", "l", "📋"),
		NewMenuItemWithIcon("Session History", "session_history", "h", "📜"),
	}
	
	if len(worktree.ActiveSessions) > 0 {
		sessionManagementItems = append(sessionManagementItems,
			NewMenuItemWithIcon("Kill All Sessions", "session_kill_all", "k", "💀"),
		)
	}
	
	items = append(items, sessionManagementItems...)
	
	return &ContextMenu{
		title: "Session Actions",
		items: items,
		theme: w.theme,
	}
}

// CreateWorktreeSessionMenu creates a session-specific menu for a worktree with session context
func (w *WorktreeContextMenu) CreateWorktreeSessionMenu(worktree WorktreeInfo, sessions []SessionSummary) *ContextMenu {
	var items []ContextMenuItem
	
	// Header showing context
	items = append(items,
		ContextMenuItem{
			Label:   fmt.Sprintf("Sessions for %s", worktree.Branch),
			Action:  "",
			Key:     "",
			Icon:    "🌿",
			Enabled: false,
		},
		NewMenuDivider(),
	)
	
	// Session actions based on current state
	if len(sessions) == 0 {
		// No sessions available
		items = append(items,
			NewMenuItemWithIcon("Create New Session", "session_new_here", "n", "➕"),
			NewMenuItemWithIcon("Import Session", "session_import", "i", "📥"),
			NewMenuDivider(),
			NewMenuItemWithIcon("Session Wizard", "session_wizard", "w", "🧙"),
		)
	} else {
		// Sessions available
		for i, session := range sessions {
			var sessionActions []ContextMenuItem
			
			// Session-specific actions
			if session.State == "active" {
				sessionActions = []ContextMenuItem{
					NewMenuItemWithIcon("Attach", fmt.Sprintf("session_attach_%s", session.ID), "a", "🔗"),
					NewMenuItemWithIcon("Detach", fmt.Sprintf("session_detach_%s", session.ID), "d", "📎"),
					NewMenuItemWithIcon("Pause", fmt.Sprintf("session_pause_%s", session.ID), "p", "⏸️"),
				}
			} else if session.State == "paused" {
				sessionActions = []ContextMenuItem{
					NewMenuItemWithIcon("Resume", fmt.Sprintf("session_resume_%s", session.ID), "r", "▶️"),
					NewMenuItemWithIcon("Attach", fmt.Sprintf("session_attach_%s", session.ID), "a", "🔗"),
					NewMenuItemWithIcon("Kill", fmt.Sprintf("session_kill_%s", session.ID), "k", "💀"),
				}
			} else {
				sessionActions = []ContextMenuItem{
					NewMenuItemWithIcon("Start", fmt.Sprintf("session_start_%s", session.ID), "s", "▶️"),
					NewMenuItemWithIcon("Remove", fmt.Sprintf("session_remove_%s", session.ID), "x", "🗑️"),
				}
			}
			
			// Create submenu for each session
			sessionSubmenu := &ContextMenu{
				title: session.Name,
				items: sessionActions,
				theme: w.theme,
			}
			
			// Session indicator
			sessionIcon := "🖥️"
			switch session.State {
			case "active":
				sessionIcon = "🟢"
			case "paused":
				sessionIcon = "⏸️"
			case "stopped":
				sessionIcon = "⏹️"
			}
			
			items = append(items, ContextMenuItem{
				Label:   fmt.Sprintf("%s (%s) - %s", session.Name, session.State, session.LastUsed),
				Action:  "",
				Key:     fmt.Sprintf("%d", i+1),
				Icon:    sessionIcon,
				Enabled: true,
				Submenu: sessionSubmenu,
			})
		}
		
		items = append(items, 
			NewMenuDivider(),
			NewMenuItemWithIcon("New Session", "session_new_additional", "n", "➕"),
		)
	}
	
	return NewContextMenu(ContextMenuConfig{
		Title: "Session Management",
		Items: items,
	}, w.theme)
}

// CreateWorktreeSelectionMenu creates a menu for multi-selected worktrees
func (w *WorktreeContextMenu) CreateWorktreeSelectionMenu(selectedWorktrees []WorktreeInfo) *ContextMenu {
	var items []ContextMenuItem
	
	// Header
	items = append(items,
		ContextMenuItem{
			Label:   fmt.Sprintf("Bulk Actions (%d selected)", len(selectedWorktrees)),
			Action:  "",
			Key:     "",
			Icon:    "📦",
			Enabled: false,
		},
		NewMenuDivider(),
	)
	
	// Session bulk actions
	sessionItems := []ContextMenuItem{
		NewMenuItemWithIcon("Create Sessions", "bulk_session_create", "n", "➕"),
		NewMenuItemWithIcon("Continue Sessions", "bulk_session_continue", "c", "▶️"),
		NewMenuItemWithIcon("Resume Sessions", "bulk_session_resume", "r", "🔄"),
		NewMenuDivider(),
		NewMenuItemWithIcon("Kill All Sessions", "bulk_session_kill", "k", "💀"),
		NewMenuItemWithIcon("Pause All Sessions", "bulk_session_pause", "p", "⏸️"),
	}
	
	sessionSubmenu := &ContextMenu{
		title: "Bulk Session Actions",
		items: sessionItems,
		theme: w.theme,
	}
	
	items = append(items, ContextMenuItem{
		Label:   "Session Actions",
		Action:  "",
		Icon:    "🖥️",
		Enabled: true,
		Submenu: sessionSubmenu,
	})
	
	// Git bulk actions
	gitItems := []ContextMenuItem{
		NewMenuItemWithIcon("Sync All", "bulk_git_sync", "s", "🔄"),
		NewMenuItemWithIcon("Fetch All", "bulk_git_fetch", "f", "⬇️"),
		NewMenuItemWithIcon("Pull All", "bulk_git_pull", "p", "⬇️"),
		NewMenuDivider(),
		NewMenuItemWithIcon("Status All", "bulk_git_status", "t", "📊"),
		NewMenuItemWithIcon("Clean All", "bulk_git_clean", "c", "🧹"),
	}
	
	gitSubmenu := &ContextMenu{
		title: "Bulk Git Actions",
		items: gitItems,
		theme: w.theme,
	}
	
	items = append(items, ContextMenuItem{
		Label:   "Git Actions",
		Action:  "",
		Icon:    "📊",
		Enabled: true,
		Submenu: gitSubmenu,
	})
	
	// Maintenance actions
	items = append(items,
		NewMenuDivider(),
		NewMenuItemWithIcon("Repair All", "bulk_worktree_repair", "x", "🔧"),
		NewMenuItemWithIcon("Refresh All", "bulk_worktree_refresh", "r", "🔄"),
		NewMenuDivider(),
	)
	
	// Danger zone
	canDelete := true
	for _, wt := range selectedWorktrees {
		if wt.IsMain {
			canDelete = false
			break
		}
	}
	
	if canDelete {
		items = append(items,
			ContextMenuItem{
				Label:   "Remove Selected",
				Action:  "bulk_worktree_remove",
				Key:     "del",
				Icon:    "🗑️",
				Enabled: true,
			},
		)
	}
	
	return NewContextMenu(ContextMenuConfig{
		Title: "Multi-Selection Actions",
		Items: items,
	}, w.theme)
}