# CLAUDE.md

This file provides guidance to Claude Code when working with the ccmgr-ultra project.

## Project Overview

ccmgr-ultra is a comprehensive CLI tool for managing Claude Code sessions across multiple projects and git worktrees. It combines the best features of CCManager and Claude Squad to provide:

- **Tmux Session Management**: Seamless creation and management of tmux sessions for Claude Code
- **Git Worktree Support**: Easy creation, merging, and management of git worktrees
- **Status Monitoring**: Real-time monitoring of Claude Code process states (idle, busy, waiting)
- **Hook System**: Execute custom scripts on state changes for automation
- **Beautiful TUI**: Intuitive terminal user interface built with Charm's Bubble Tea
- **Multi-Project Support**: Manage multiple projects and sessions from a single interface

## Technology Stack

- **Language**: Go 1.24.4
- **UI Framework**: Charm's Bubble Tea (TUI framework)
- **Database**: SQLite (for session/analytics storage)
- **Configuration**: Viper + YAML
- **CLI Framework**: Cobra
- **Testing**: testify
- **External Dependencies**: tmux, git, gum (for interactive scripts)

## Project Structure

```
ccmgr-ultra/
├── cmd/ccmgr-ultra/        # CLI entry point and commands
│   ├── main.go            # Main entry point
│   ├── common.go          # Shared command utilities
│   ├── init.go            # Project initialization
│   ├── session.go         # Session management commands
│   ├── worktree.go        # Worktree management commands
│   ├── status.go          # Status monitoring
│   ├── continue.go        # Continue/resume functionality
│   └── completion.go      # Shell completion
├── internal/              # Internal packages
│   ├── analytics/        # Usage analytics and metrics
│   ├── claude/          # Claude Code process monitoring
│   ├── cli/             # CLI utilities (spinners, tables, etc.)
│   ├── config/          # Configuration management
│   ├── git/             # Git operations and worktree management
│   ├── hooks/           # Hook execution system
│   ├── storage/         # Database abstraction layer
│   ├── tmux/            # Tmux session management
│   └── tui/             # Terminal UI implementation
├── pkg/ccmgr/           # Public API package
│   ├── interfaces.go    # Core interfaces
│   ├── api.go          # Public API implementation
│   └── *_manager.go    # Manager implementations
├── scripts/             # Supporting scripts
│   ├── gum/            # Gum-based interactive scripts
│   └── hooks/          # Example hook scripts
└── docs/               # Documentation

```

## Key Design Patterns

### 1. Manager Pattern
The project uses manager interfaces for different subsystems:
- `SessionManager`: Handles tmux session lifecycle
- `WorktreeManager`: Manages git worktrees
- `SystemManager`: Provides overall system status

### 2. State Machine
Claude process monitoring uses a state machine pattern (`internal/claude/state_machine.go`) to track process states:
- Idle
- Busy  
- Waiting
- Error

### 3. Hook System
Status changes trigger configurable hooks with environment variables:
- `CCMGR_SESSION_ID`
- `CCMGR_WORKTREE_ID`
- `CCMGR_WORKING_DIR`
- `CCMGR_NEW_STATE`
- `CCMGR_OLD_STATE`

### 4. TUI Architecture
The TUI follows the Elm Architecture pattern from Bubble Tea:
- Model: Application state
- Update: Handle messages and update state
- View: Render the UI

## Development Guidelines

### Building and Testing

```bash
# Build the project
make build

# Run tests
make test

# Run with coverage
make test-coverage

# Install locally
make install

# Run linting
make lint
```

### Adding New Features

1. **New Commands**: Add to `cmd/ccmgr-ultra/` following the existing pattern
2. **New TUI Screens**: Add to `internal/tui/screens.go` and update navigation
3. **New Workflows**: Add to `internal/tui/workflows/`
4. **Configuration**: Update schema in `internal/config/schema.go`

### Database Migrations

Migrations are handled automatically using embedded SQL files:
- Location: `internal/storage/sqlite/migrations/`
- Naming: `XXX_description.sql` (e.g., `001_initial.sql`)

### Error Handling

- Use wrapped errors with context: `fmt.Errorf("failed to X: %w", err)`
- Define specific error types in relevant packages
- Handle errors gracefully in the TUI without crashing

### Testing Approach

- Unit tests for individual components
- Integration tests for subsystems (marked with `// +build integration`)
- Mock interfaces for testing
- Table-driven tests where appropriate

## Configuration

The project uses a layered configuration system:
1. Default values (hardcoded)
2. Config file (`~/.config/ccmgr-ultra/config.yaml`)
3. Environment variables (`CCMGR_*`)
4. Command-line flags

Example configuration structure is in `example-claude-config.yaml`.

## Important Implementation Notes

### Claude Process Detection
The system monitors Claude Code processes through:
1. Process name matching
2. Log file parsing
3. Resource usage patterns
4. Tmux pane content analysis

### Worktree Naming Convention
Default pattern: `.git/{branch-name}` where slashes are replaced with dashes
Example: `feature/new-auth` → `.git/feature-new-auth`

### Session Naming
Tmux sessions follow the pattern: `ccmgr-{project}-{branch}`
Example: `ccmgr-myapp-feature-auth`

### State Persistence
- Sessions are tracked in SQLite database
- Analytics data is stored for usage patterns
- Configuration is persisted in YAML

## Common Development Tasks

### Adding a New Command
1. Create new file in `cmd/ccmgr-ultra/`
2. Define cobra.Command
3. Add to rootCmd in init()
4. Implement business logic using managers

### Extending the TUI
1. Add new screen type to `internal/tui/screens.go`
2. Implement Init(), Update(), View() methods
3. Add navigation logic
4. Update help text

### Adding Analytics
1. Define new event type in `internal/analytics/types.go`
2. Add collection logic in relevant manager
3. Create query in `internal/analytics/queries.go`

## Debugging Tips

1. **Verbose Mode**: Use `-v` flag for detailed output
2. **Dry Run**: Use `--dry-run` to preview actions
3. **TUI Debugging**: Set `BUBBLETEA_LOG=debug`
4. **Database**: SQLite database is at `~/.local/share/ccmgr-ultra/ccmgr.db`

## Future Enhancements (from purpose.md)

- Claude Code configuration inheritance for worktrees
- MCP (Model Context Protocol) settings synchronization
- Enhanced project templates
- Multi-agent coordination features