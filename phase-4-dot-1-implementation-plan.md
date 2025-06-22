# Phase 4.1 Implementation Plan: CLI Interface Enhancement

## Overview

Phase 4.1 focuses on expanding the CLI interface beyond the current basic TUI launcher to include comprehensive command-line functionality. Currently, the main.go only implements the basic TUI launch and version commands. This phase will add the full suite of CLI commands needed for automation, scripting, and non-interactive workflows.

## Current State Analysis

### ✅ Already Implemented
- **Basic CLI Structure**: Cobra-based command structure with root command
- **TUI Launch**: Default behavior launches the TUI application
- **Version Command**: `ccmgr-ultra version` displays version information
- **Configuration Loading**: Proper config loading with error handling
- **Signal Handling**: Graceful shutdown with context cancellation

### 🚧 Missing Components (Phase 4.1 Targets)
- **Init Command**: `ccmgr-ultra init` for new project initialization
- **Continue Command**: `ccmgr-ultra continue <worktree>` for session continuation
- **Status Command**: `ccmgr-ultra status` for current state information
- **Global Flags**: Non-interactive mode support
- **Worktree Commands**: Direct worktree management
- **Session Commands**: Direct session management

## Implementation Scope

### 1. Enhanced CLI Command Structure

#### 1.1 Root Command Enhancements
- **Global Flags**:
  - `--non-interactive, -n`: Skip TUI, use CLI-only mode
  - `--config, -c`: Specify custom config file path
  - `--verbose, -v`: Enable verbose output
  - `--quiet, -q`: Suppress non-essential output
  - `--dry-run`: Show what would be done without executing

#### 1.2 New Primary Commands

##### A. `init` Command
```bash
ccmgr-ultra init [flags]
ccmgr-ultra init --repo-name="my-project" --description="My awesome project"
```
**Functionality**:
- Detect if in empty/non-git directory
- Launch gum script for project initialization if interactive
- Accept flags for non-interactive initialization
- Create initial git repository structure
- Set up default ccmgr-ultra configuration
- Initialize first Claude Code session

**Flags**:
- `--repo-name, -r`: Repository name
- `--description, -d`: Project description  
- `--template, -t`: Project template to use
- `--no-claude`: Skip Claude Code session initialization
- `--branch, -b`: Initial branch name (default: main)

##### B. `continue` Command
```bash
ccmgr-ultra continue <worktree> [flags]
ccmgr-ultra continue feature-branch --new-session
```
**Functionality**:
- Continue existing session for specified worktree
- Create new session if none exists
- Handle tmux session attachment/creation
- Support both interactive and non-interactive modes

**Flags**:
- `--new-session, -n`: Force new session creation
- `--session-id, -s`: Specific session ID to continue
- `--detached, -d`: Start session detached from terminal

##### C. `status` Command
```bash
ccmgr-ultra status [flags]
ccmgr-ultra status --worktree=feature-branch --format=json
```
**Functionality**:
- Display current project status
- Show all worktrees and their states
- Display Claude Code process status
- Show active tmux sessions
- Report configuration status

**Flags**:
- `--worktree, -w`: Show status for specific worktree
- `--format, -f`: Output format (table, json, yaml)
- `--watch`: Continuously monitor and update status
- `--refresh-interval`: Status refresh interval for watch mode

#### 1.3 Worktree Management Commands

##### A. `worktree` Command Group
```bash
ccmgr-ultra worktree <subcommand> [flags]
```

**Subcommands**:
- `list`: List all worktrees with status
- `create <branch>`: Create new worktree
- `delete <worktree>`: Delete worktree
- `merge <worktree>`: Merge worktree changes
- `push <worktree>`: Push worktree and create PR

**Examples**:
```bash
ccmgr-ultra worktree list --format=table
ccmgr-ultra worktree create feature-auth --base=main
ccmgr-ultra worktree delete feature-old --force
ccmgr-ultra worktree merge feature-complete --delete-after
ccmgr-ultra worktree push feature-ready --create-pr
```

#### 1.4 Session Management Commands

##### A. `session` Command Group
```bash
ccmgr-ultra session <subcommand> [flags]
```

**Subcommands**:
- `list`: List all active sessions
- `new <worktree>`: Create new session
- `resume <session-id>`: Resume existing session
- `kill <session-id>`: Terminate session
- `clean`: Clean up stale sessions

**Examples**:
```bash
ccmgr-ultra session list --worktree=feature-branch
ccmgr-ultra session new feature-branch --name=dev-session
ccmgr-ultra session resume ccmgr-main-feature-abc123
ccmgr-ultra session kill --all-stale
```

### 2. Integration with Existing Components

#### 2.1 Configuration Integration
- Leverage existing `internal/config` package
- Support config file override via `--config` flag
- Validate configuration before command execution
- Provide config status in `status` command

#### 2.2 Git Integration
- Use existing `internal/git` package for worktree operations
- Implement command-line wrappers for git functionality
- Support both interactive and non-interactive git operations
- Handle git authentication and remote operations

#### 2.3 Tmux Integration
- Leverage `internal/tmux` package for session management
- Support detached session creation for automation
- Provide session status and monitoring
- Handle tmux session naming and organization

#### 2.4 Claude Process Integration
- Use `internal/claude` package for process monitoring
- Support Claude Code session lifecycle management
- Integrate with status hooks system
- Provide process status in CLI output

### 3. Output Formatting and User Experience

#### 3.1 Output Formats
- **Table**: Human-readable tabular output (default)
- **JSON**: Machine-readable JSON for scripting
- **YAML**: Configuration-friendly YAML format
- **Compact**: Single-line compact format for scripts

#### 3.2 Progress Indicators
- Spinner/progress bars for long operations
- Real-time status updates for monitoring commands
- Clear success/error messaging
- Verbose mode with detailed operation logs

#### 3.3 Error Handling
- Consistent error codes for scripting
- Helpful error messages with suggested fixes
- Graceful degradation when components unavailable
- Clear indication of interactive vs non-interactive limitations

### 4. File Structure and Implementation

#### 4.1 New Files to Create
```
cmd/ccmgr-ultra/
├── main.go                 # Enhanced root command
├── init.go                 # Init command implementation
├── continue.go             # Continue command implementation
├── status.go              # Status command implementation
├── worktree.go            # Worktree subcommands
├── session.go             # Session subcommands
└── common.go              # Shared CLI utilities
```

#### 4.2 Enhanced Existing Files
- `main.go`: Add new command registrations and global flags
- Enhanced error handling and output formatting

#### 4.3 Supporting Infrastructure
```
internal/cli/
├── output.go              # Output formatting utilities
├── spinner.go             # Progress indication
├── errors.go              # CLI-specific error handling
└── validation.go          # Input validation helpers
```

### 5. Implementation Priority and Sequence

#### Week 1: Foundation and Core Commands
1. **Day 1-2**: Enhance main.go with global flags and command structure
2. **Day 3-4**: Implement `init` command with gum script integration
3. **Day 5**: Implement `status` command with basic output formats

#### Week 2: Session and Worktree Commands
1. **Day 1-2**: Implement `continue` command with session management
2. **Day 3-4**: Implement `worktree` command group
3. **Day 5**: Implement `session` command group

#### Week 3: Polish and Testing
1. **Day 1-2**: Add output formatting and progress indicators
2. **Day 3-4**: Comprehensive testing and error handling
3. **Day 5**: Documentation and integration testing

### 6. Integration with Gum Scripts

#### 6.1 Enhanced Gum Script Integration
- Detect gum availability and provide fallbacks
- Support both interactive and non-interactive project initialization
- Pass configuration options to gum scripts
- Handle gum script errors gracefully

#### 6.2 Script Enhancement Requirements
- Update `scripts/gum/init-repo.sh` to accept command-line arguments
- Add validation for non-interactive inputs
- Integrate with ccmgr-ultra configuration system
- Support template-based project initialization

### 7. Testing Strategy

#### 7.1 Unit Tests
- Command-line argument parsing
- Flag validation and default handling
- Output formatting functions
- Error handling scenarios

#### 7.2 Integration Tests
- End-to-end command workflows
- Integration with existing components
- Configuration loading and validation
- Tmux and git integration

#### 7.3 User Acceptance Testing
- Interactive vs non-interactive mode compatibility
- Scripting and automation scenarios
- Cross-platform compatibility (macOS, Linux)
- Performance with large projects

### 8. Success Criteria

#### 8.1 Functional Requirements ✅
- All specified commands implemented and working
- Seamless integration with existing TUI functionality
- Support for both interactive and automation workflows
- Consistent output formatting across all commands

#### 8.2 User Experience ✅
- Intuitive command-line interface following Unix conventions
- Clear help text and usage examples
- Graceful error handling with helpful messages
- Fast response times for status and list operations

#### 8.3 Integration ✅
- Proper integration with all existing internal packages
- Configuration consistency between CLI and TUI
- Tmux session management working correctly
- Git worktree operations functioning as expected

### 9. Future Enhancements (Post Phase 4.1)

#### 9.1 Advanced Features
- Shell completion (bash, zsh, fish)
- Configuration validation and migration commands
- Bulk operations for multiple worktrees
- Integration with CI/CD pipelines

#### 9.2 Automation Support
- Webhook handlers for git events
- Scheduled operations and maintenance
- Remote management capabilities
- API server mode for external integration

### 10. Risk Mitigation

#### 10.1 Identified Risks
- **Command Complexity**: CLI commands might become too complex
- **TUI Integration**: Ensuring CLI doesn't break existing TUI functionality
- **Cross-Platform**: Ensuring commands work consistently across platforms
- **Performance**: CLI commands should be fast for automation use

#### 10.2 Mitigation Strategies
- **Modular Design**: Keep commands focused and single-purpose
- **Extensive Testing**: Comprehensive testing of CLI and TUI integration
- **Platform Testing**: Test on both macOS and Linux environments
- **Performance Monitoring**: Profile command execution times

## Conclusion

Phase 4.1 will transform ccmgr-ultra from primarily a TUI application to a comprehensive CLI tool that supports both interactive and automated workflows. The implementation will maintain full compatibility with existing functionality while adding powerful command-line capabilities for power users and automation scenarios.

The modular approach ensures that each command can be implemented and tested independently, while the integration with existing packages maintains consistency and reduces code duplication. Upon completion, users will have a complete toolkit for managing Claude Code sessions and git worktrees through any interface they prefer.