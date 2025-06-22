# ccmgr-ultra Implementation Steps

## Overview
This document outlines the step-by-step implementation plan for ccmgr-ultra, a Claude Multi-Project Multi-Session Manager that combines the best features of CCManager and Claude Squad.

## Phase 1: Project Setup and Foundation

### 1.1 Initialize Go Project
```bash
go mod init github.com/bcdekker/ccmgr-ultra
```

### 1.2 Install Dependencies
```bash
go get github.com/charmbracelet/bubbletea
go get github.com/charmbracelet/bubbles
go get github.com/charmbracelet/lipgloss
go get github.com/spf13/cobra
go get github.com/spf13/viper
go get github.com/go-git/go-git/v5
```

### 1.3 Create Project Structure
```
ccmgr-ultra/
├── cmd/
│   └── ccmgr-ultra/
│       └── main.go
├── internal/
│   ├── config/
│   │   ├── config.go
│   │   └── schema.go
│   ├── git/
│   │   ├── worktree.go
│   │   └── operations.go
│   ├── tmux/
│   │   ├── session.go
│   │   └── monitor.go
│   ├── claude/
│   │   ├── process.go
│   │   └── status.go
│   ├── hooks/
│   │   └── hooks.go
│   └── tui/
│       ├── app.go
│       ├── mainmenu.go
│       ├── worktree.go
│       └── config.go
├── scripts/
│   └── gum/
│       └── init-repo.sh
├── go.mod
├── go.sum
└── README.md
```

## Phase 2: Core Components Implementation

### 2.1 Configuration Management (internal/config/)
- [ ] Define configuration schema for:
  - Status hooks (idle, busy, waiting)
  - Worktree settings (auto-directory, pattern)
  - Shortcuts
  - Command settings
- [ ] Implement config file loading/saving using viper
- [ ] Create default configuration template
- [ ] Handle config migration for future versions

### 2.2 Tmux Integration (internal/tmux/)
- [ ] Implement tmux session management:
  - Create new sessions with standardized naming (project-worktree-branch)
  - List existing sessions
  - Attach/detach from sessions
  - Kill sessions
- [ ] Monitor Claude Code processes within tmux sessions
- [ ] Handle session state persistence

### 2.3 Git Worktree Management (internal/git/)
- [ ] Implement worktree operations:
  - List existing worktrees
  - Create new worktree with branch
  - Delete worktree
  - Merge worktree changes
  - Push worktree to remote
- [ ] Handle worktree directory naming patterns
- [ ] Implement PR creation functionality (GitHub/GitLab API)

### 2.4 Claude Code Process Monitoring (internal/claude/)
- [ ] Detect Claude Code process state (busy, idle, waiting)
- [ ] Implement state change detection
- [ ] Create process tracking mechanism
- [ ] Handle multiple Claude Code instances

### 2.5 Status Hooks System (internal/hooks/)
- [ ] Execute configured scripts on state changes
- [ ] Pass environment variables:
  - CCMANAGER_WORKTREE
  - CCMANAGER_WORKTREE_BRANCH
  - CCMANAGER_NEW_STATE
  - CCMANAGER_SESSION_ID
- [ ] Handle hook script errors gracefully
- [ ] Support async hook execution

## Phase 3: TUI Implementation

### 3.1 Main Application (internal/tui/app.go)
- [ ] Create bubbletea application structure
- [ ] Implement navigation between screens
- [ ] Handle keyboard shortcuts
- [ ] Add status bar with Claude Code states

### 3.2 Main Menu (internal/tui/mainmenu.go)
- [ ] Detect if in git repo or empty directory
- [ ] Show appropriate menu based on context:
  - New project options
  - Existing project worktree list
- [ ] Implement menu items:
  - New Worktree
  - Merge Worktree
  - Delete Worktree
  - Push Worktree (with PR creation)
  - Configuration
  - Exit

### 3.3 Worktree Selection Screen
- [ ] Display worktrees with status indicators
- [ ] Implement interactive selection with options:
  - 'n' - New session
  - 'c' - Continue session
  - 'r' - Resume session
- [ ] Show Claude Code status for each worktree
- [ ] Handle worktree operations

### 3.4 Configuration Screens (internal/tui/config.go)
- [ ] Main configuration menu
- [ ] Shortcuts configuration
- [ ] Status hooks configuration with enable/disable toggles
- [ ] Worktree settings (auto-directory, pattern)
- [ ] Command configuration
- [ ] Save/cancel functionality

## Phase 4: CLI and Scripts

### 4.1 CLI Interface (cmd/ccmgr-ultra/main.go)
- [ ] Implement cobra commands:
  - `ccmgr-ultra` - Launch TUI
  - `ccmgr-ultra init` - Initialize new project
  - `ccmgr-ultra continue <worktree>` - Continue session
  - `ccmgr-ultra status` - Show status
- [ ] Add global flags for non-interactive mode

### 4.2 Gum Scripts (scripts/gum/)
- [ ] Create init-repo.sh script for new projects:
  - Collect repo name
  - Collect project name
  - Collect description
  - Initialize git repo
  - Create initial commit

## Phase 5: Features Implementation

### 5.1 New Project Flow
- [ ] Detect non-git or empty directory
- [ ] Run gum script for initialization
- [ ] Create initial Claude Code session
- [ ] Set up default configuration

### 5.2 Session Management
- [ ] Implement continue vs resume logic
- [ ] Handle multiple sessions per worktree
- [ ] Track session IDs
- [ ] Clean up stale sessions

### 5.3 Push Worktree Feature
- [ ] Push branch to origin
- [ ] Detect git host (GitHub, GitLab, etc.)
- [ ] Create pull request via API
- [ ] Handle authentication (token/SSH)

### 5.4 Status Monitoring
- [ ] Background process for monitoring
- [ ] Efficient polling mechanism
- [ ] Status persistence across restarts
- [ ] Integration with skate or similar tools

## Phase 6: Testing and Documentation

### 6.1 Testing
- [ ] Unit tests for core components
- [ ] Integration tests for git operations
- [ ] TUI component tests
- [ ] End-to-end workflow tests
- [ ] Mock tmux for testing

### 6.2 Documentation
- [ ] Installation guide
- [ ] Configuration reference
- [ ] Usage examples
- [ ] Hook script examples
- [ ] Troubleshooting guide

## Phase 7: Advanced Features (Future)

### 7.1 Configuration Inheritance
- [ ] Implement settings sync for new worktrees
- [ ] Copy MCPs and permissions from parent
- [ ] Handle .claude directory management
- [ ] Provide override options

### 7.2 Multi-Project Support
- [ ] Global project registry
- [ ] Quick switch between projects
- [ ] Project templates
- [ ] Shared configuration profiles

## Implementation Order

1. **Week 1-2**: Project setup, core configuration, and basic tmux integration
2. **Week 3-4**: Git worktree management and Claude Code monitoring
3. **Week 5-6**: TUI implementation with main menu and worktree selection
4. **Week 7-8**: Configuration screens and status hooks
5. **Week 9-10**: Session management and push worktree feature
6. **Week 11-12**: Testing, documentation, and polish

## Technical Decisions

### Language: Go
- Excellent for CLI tools
- Great concurrency support for monitoring
- Strong ecosystem with charm.sh libraries
- Single binary distribution

### State Management
- Use local JSON/YAML files for configuration
- Consider SQLite for session/project state if needed
- Integrate with skate for distributed state

### Error Handling
- Graceful degradation when tmux not available
- Clear error messages for git operations
- Fallback modes for different environments

## Success Criteria

1. Seamless tmux session management for Claude Code
2. Intuitive TUI matching the CCManager aesthetic
3. Reliable status monitoring and hook execution
4. Easy worktree creation and management
5. Configuration that persists across sessions
6. Cross-platform support (macOS, Linux)

## Notes

- Prioritize user experience and reliability
- Keep the TUI responsive even during long operations
- Ensure backward compatibility with CCManager hooks
- Design for extensibility and future features