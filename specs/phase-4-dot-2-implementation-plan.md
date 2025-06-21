# Phase 4.2 Implementation Plan: Advanced CLI Commands & Session Management

## Overview

Phase 4.2 builds upon Phase 4.1's foundation by implementing the remaining advanced CLI commands that were outlined but not yet developed. This phase focuses on **worktree management** and **session management** command groups, along with enhanced CLI tooling and user experience improvements.

Phase 4.1 successfully established the core CLI infrastructure with `init`, `continue`, and `status` commands. Phase 4.2 completes the CLI functionality by adding the sophisticated worktree and session management capabilities that make ccmgr-ultra a comprehensive tool for both interactive and automated workflows.

## Current State Analysis

### ✅ Already Implemented in Phase 4.1
- **Basic CLI Structure**: Cobra-based command structure with global flags
- **Core Commands**: `init`, `continue`, `status` commands fully functional
- **Configuration Integration**: Proper config loading with overrides via `--config` flag
- **CLI Utilities**: Comprehensive `internal/cli` package with output formatting, progress indicators, error handling
- **Signal Handling**: Graceful shutdown with context cancellation
- **Output Formats**: Support for table, JSON, and YAML output formats
- **Integration**: Seamless integration with existing TUI functionality

### 🎯 Phase 4.2 Targets
- **Worktree Command Group**: Complete git worktree lifecycle management
- **Session Command Group**: Comprehensive tmux session operations
- **Enhanced User Experience**: Interactive selection, batch operations, improved error handling
- **Advanced Integration**: PR creation, remote operations, session health monitoring
- **Performance Optimization**: Fast operations suitable for automation and scripting

## Implementation Scope

### 1. Worktree Command Group

#### 1.1 Primary Command Structure
```bash
ccmgr-ultra worktree <subcommand> [flags]
```

#### 1.2 Subcommands to Implement

##### A. `worktree list` Command
```bash
ccmgr-ultra worktree list [flags]
ccmgr-ultra worktree list --format=json --status=dirty
```

**Functionality**:
- List all git worktrees with comprehensive status information
- Show branch, HEAD commit, clean/dirty status, associated tmux sessions
- Display Claude Code process information per worktree
- Support filtering by status, branch pattern, or activity

**Flags**:
- `--format, -f`: Output format (table, json, yaml, compact)
- `--status, -s`: Filter by status (clean, dirty, active, stale)
- `--branch, -b`: Filter by branch name pattern
- `--with-processes`: Include Claude Code process information
- `--sort`: Sort by (name, last-accessed, created, status)

##### B. `worktree create` Command
```bash
ccmgr-ultra worktree create <branch> [flags]
ccmgr-ultra worktree create feature-auth --base=main --start-session
```

**Functionality**:
- Create new git worktree from specified or current branch
- Automatically generate worktree directory using configured pattern
- Optionally start tmux session and Claude Code process
- Support for remote branch tracking setup
- Handle worktree naming conflicts intelligently

**Flags**:
- `--base, -b`: Base branch for new worktree (default: current branch)
- `--directory, -d`: Custom worktree directory path
- `--start-session, -s`: Automatically start tmux session
- `--start-claude`: Automatically start Claude Code in new session
- `--remote, -r`: Track remote branch if exists
- `--force`: Overwrite existing worktree if present

##### C. `worktree delete` Command
```bash
ccmgr-ultra worktree delete <worktree> [flags]
ccmgr-ultra worktree delete feature-old --force --cleanup-sessions
```

**Functionality**:
- Delete specified worktree with safety checks
- Handle active tmux sessions and Claude Code processes
- Optionally clean up related sessions and processes
- Support batch deletion with pattern matching
- Provide confirmation prompts for destructive operations

**Flags**:
- `--force, -f`: Skip confirmation prompts
- `--cleanup-sessions`: Terminate related tmux sessions
- `--cleanup-processes`: Stop related Claude Code processes
- `--keep-branch`: Keep git branch after deleting worktree
- `--pattern`: Delete multiple worktrees matching pattern

##### D. `worktree merge` Command
```bash
ccmgr-ultra worktree merge <worktree> [flags]
ccmgr-ultra worktree merge feature-complete --delete-after --push-first
```

**Functionality**:
- Merge worktree changes back to main/target branch
- Handle merge conflicts with clear guidance
- Optionally push changes before merging
- Support for different merge strategies
- Clean up worktree after successful merge

**Flags**:
- `--target, -t`: Target branch for merge (default: main)
- `--strategy, -s`: Merge strategy (merge, squash, rebase)
- `--delete-after`: Delete worktree after successful merge
- `--push-first`: Push worktree branch before merging
- `--message, -m`: Custom merge commit message

##### E. `worktree push` Command
```bash
ccmgr-ultra worktree push <worktree> [flags]
ccmgr-ultra worktree push feature-ready --create-pr --pr-title="Add new feature"
```

**Functionality**:
- Push worktree branch to remote repository
- Optionally create pull request via GitHub CLI
- Handle remote tracking setup if needed
- Support for draft PRs and custom PR templates
- Integration with existing PR workflow tools

**Flags**:
- `--create-pr`: Create pull request after push
- `--pr-title`: Pull request title
- `--pr-body`: Pull request body
- `--draft`: Create draft pull request
- `--reviewer`: Add reviewers to pull request
- `--force`: Force push (use with caution)

#### 1.3 Interactive Features
- **Auto-completion**: Tab completion for worktree names and branches
- **Selection Mode**: Interactive selection when no worktree specified
- **Confirmation Prompts**: Safety prompts for destructive operations
- **Progress Feedback**: Real-time progress for multi-step operations

### 2. Session Command Group

#### 2.1 Primary Command Structure
```bash
ccmgr-ultra session <subcommand> [flags]
```

#### 2.2 Subcommands to Implement

##### A. `session list` Command
```bash
ccmgr-ultra session list [flags]
ccmgr-ultra session list --worktree=feature-branch --format=table
```

**Functionality**:
- List all active tmux sessions managed by ccmgr-ultra
- Show session details, associated worktrees, and process status
- Display session health and activity information
- Support filtering by worktree, project, or status

**Flags**:
- `--format, -f`: Output format (table, json, yaml, compact)
- `--worktree, -w`: Filter by worktree name
- `--project, -p`: Filter by project name
- `--status, -s`: Filter by status (active, idle, stale)
- `--with-processes`: Include Claude Code process details

##### B. `session new` Command
```bash
ccmgr-ultra session new <worktree> [flags]
ccmgr-ultra session new feature-branch --name=dev-session --start-claude
```

**Functionality**:
- Create new tmux session for specified worktree
- Follow ccmgr-ultra session naming conventions
- Optionally start Claude Code process in session
- Handle session configuration and environment setup
- Support for detached session creation

**Flags**:
- `--name, -n`: Custom session name suffix
- `--start-claude`: Automatically start Claude Code
- `--detached, -d`: Create session detached from terminal
- `--config, -c`: Custom Claude Code config for session
- `--inherit-config`: Inherit config from parent directory

##### C. `session resume` Command
```bash
ccmgr-ultra session resume <session-id> [flags]
ccmgr-ultra session resume ccmgr-main-feature-abc123 --attach
```

**Functionality**:
- Resume existing tmux session by ID or name
- Handle session state validation and recovery
- Support for attaching to local terminal
- Restart stopped Claude Code processes if needed
- Provide session status and health information

**Flags**:
- `--attach, -a`: Attach to session in current terminal
- `--detached, -d`: Resume session detached
- `--restart-claude`: Restart Claude Code if stopped
- `--force`: Force resume even if session appears unhealthy

##### D. `session kill` Command
```bash
ccmgr-ultra session kill <session-id> [flags]
ccmgr-ultra session kill --all-stale --force
```

**Functionality**:
- Terminate specified tmux session gracefully
- Handle Claude Code process shutdown properly
- Support batch termination with filters
- Provide confirmation for destructive operations
- Clean up associated resources and state

**Flags**:
- `--force, -f`: Skip confirmation prompts
- `--all-stale`: Kill all stale/orphaned sessions
- `--pattern`: Kill sessions matching pattern
- `--cleanup`: Clean up related processes and state
- `--timeout`: Timeout for graceful shutdown

##### E. `session clean` Command
```bash
ccmgr-ultra session clean [flags]
ccmgr-ultra session clean --dry-run --verbose
```

**Functionality**:
- Clean up stale, orphaned, or invalid sessions
- Detect and remove sessions with missing worktrees
- Handle sessions with stopped Claude Code processes
- Provide detailed cleanup report
- Support dry-run mode for safety

**Flags**:
- `--dry-run`: Show what would be cleaned without acting
- `--force, -f`: Skip confirmation prompts
- `--all`: Clean all eligible sessions, not just stale ones
- `--older-than`: Clean sessions older than specified duration
- `--verbose, -v`: Detailed cleanup information

#### 2.3 Session Management Features
- **Health Monitoring**: Continuous monitoring of session and process health
- **Auto-recovery**: Automatic recovery of failed Claude Code processes
- **State Persistence**: Session state tracking across restarts
- **Resource Management**: Efficient handling of session resources

### 3. Enhanced CLI Infrastructure

#### 3.1 New Files to Create
```
cmd/ccmgr-ultra/
├── worktree.go          # Worktree command group implementation
├── session.go           # Session command group implementation
└── completion.go        # Shell completion support

internal/cli/
├── interactive.go       # Interactive selection utilities
├── progress.go          # Enhanced progress indicators
├── batch.go            # Batch operation utilities
├── confirmation.go     # User confirmation prompts
└── table.go            # Enhanced table formatting
```

#### 3.2 Core Capabilities

##### A. Interactive Selection
- **Gum Integration**: Use gum for interactive selection when arguments omitted
- **Fuzzy Search**: Support fuzzy finding for worktrees and sessions
- **Multi-select**: Support for selecting multiple items for batch operations
- **Context-aware**: Show relevant information during selection

##### B. Batch Operations
- **Pattern Matching**: Support glob patterns for multi-item operations
- **Filtering**: Advanced filtering options for bulk operations
- **Progress Tracking**: Real-time progress for batch operations
- **Error Handling**: Graceful handling of partial failures in batch operations

##### C. Enhanced Validation
- **Pre-flight Checks**: Comprehensive validation before operations
- **Dependency Validation**: Check for required tools and dependencies
- **State Consistency**: Ensure consistent state across operations
- **Conflict Detection**: Early detection of potential conflicts

### 4. Integration Enhancements

#### 4.1 Git Integration
```go
// Enhanced git operations in internal/git package
type AdvancedWorktreeManager struct {
    baseManager *WorktreeManager
    remoteOps   *RemoteOperations
    validator   *StateValidator
}

func (m *AdvancedWorktreeManager) CreateWithTracking(branch, base string, opts CreateOptions) error
func (m *AdvancedWorktreeManager) MergeWithStrategy(worktree, target string, strategy MergeStrategy) error
func (m *AdvancedWorktreeManager) PushWithPR(worktree string, prOpts PROptions) error
```

**New Capabilities**:
- Remote push/pull operations with authentication handling
- Pull request creation via GitHub CLI integration
- Merge conflict detection and resolution guidance
- Branch tracking setup for new worktrees
- Validation of git state before operations

#### 4.2 Tmux Integration
```go
// Enhanced session management in internal/tmux package
type AdvancedSessionManager struct {
    baseManager *SessionManager
    healthMonitor *HealthMonitor
    lifecycle     *SessionLifecycle
}

func (m *AdvancedSessionManager) CreateWithConfig(worktree string, config SessionConfig) (*Session, error)
func (m *AdvancedSessionManager) HealthCheck(sessionID string) (*HealthStatus, error)
func (m *AdvancedSessionManager) CleanupStale(criteria CleanupCriteria) ([]string, error)
```

**New Capabilities**:
- Session health monitoring and reporting
- Automatic session recovery mechanisms
- Custom session configuration support
- Session lifecycle management with hooks
- Resource cleanup and optimization

#### 4.3 Claude Process Integration
```go
// Enhanced process management in internal/claude package
type AdvancedProcessManager struct {
    baseManager *ProcessManager
    lifecycle   *ProcessLifecycle
    monitor     *HealthMonitor
}

func (m *AdvancedProcessManager) StartWithSession(sessionID, worktree string) (*ProcessInfo, error)
func (m *AdvancedProcessManager) MonitorHealth(processID string) (*HealthReport, error)
func (m *AdvancedProcessManager) GracefulShutdown(processID string, timeout time.Duration) error
```

**New Capabilities**:
- Process-aware session operations
- State transition monitoring with hooks
- Graceful process shutdown and restart
- Integration with session lifecycle events
- Performance monitoring and reporting

### 5. User Experience Improvements

#### 5.1 Output Enhancements
```go
// Enhanced output formatting
type TableFormatter struct {
    colorEnabled bool
    maxWidth     int
    theme        Theme
}

func (f *TableFormatter) FormatWorktrees(worktrees []WorktreeStatus) string
func (f *TableFormatter) FormatSessions(sessions []SessionStatus) string
func (f *TableFormatter) FormatWithProgress(data interface{}) string
```

**Features**:
- Rich table formatting with color coding and icons
- Responsive layouts adapting to terminal width
- Machine-readable formats optimized for scripting
- Real-time status updates with watch mode
- Compact mode for CI/CD integration

#### 5.2 Interactive Features
```go
// Interactive utilities
type InteractiveSelector struct {
    gumClient *gum.Client
    theme     Theme
    options   SelectorOptions
}

func (s *InteractiveSelector) SelectWorktree(worktrees []WorktreeInfo) (*WorktreeInfo, error)
func (s *InteractiveSelector) SelectSessions(sessions []*Session) ([]*Session, error)
func (s *InteractiveSelector) ConfirmOperation(operation string, impact Impact) (bool, error)
```

**Features**:
- Auto-completion for worktree and session names
- Interactive confirmation prompts with impact assessment
- Progress bars and spinners for long operations
- Helpful error messages with suggested fixes
- Context-sensitive help and guidance

### 6. Error Handling and Recovery

#### 6.1 Comprehensive Error Handling
```go
// Advanced error handling
type OperationError struct {
    Operation  string
    Worktree   string
    Session    string
    Cause      error
    Suggestion string
    Recovery   func() error
}

func (e *OperationError) Error() string
func (e *OperationError) Suggest() string
func (e *OperationError) Recover() error
```

**Features**:
- Detailed error context with operation information
- Suggested fixes and recovery actions
- Automatic recovery for transient failures
- Error aggregation for batch operations
- User-friendly error presentation

#### 6.2 State Recovery
- **Consistency Checks**: Validate state before and after operations
- **Rollback Capability**: Undo partial operations on failure
- **Conflict Resolution**: Guide users through conflict resolution
- **Health Restoration**: Automatic restoration of unhealthy state

### 7. Implementation Timeline

#### Week 1: Worktree Commands Foundation
**Days 1-2: Core Worktree Operations**
- Implement `worktree list` with comprehensive status display
- Implement `worktree create` with automatic directory generation
- Add basic interactive selection for worktree operations
- Integrate with existing git package for worktree detection

**Days 3-4: Advanced Worktree Operations**
- Implement `worktree delete` with safety checks and cleanup
- Implement `worktree merge` with conflict detection
- Add batch operation support with pattern matching
- Implement confirmation prompts and validation

**Day 5: Worktree Push and PR Integration**
- Implement `worktree push` with remote operations
- Add GitHub CLI integration for PR creation
- Implement force push safety mechanisms
- Add comprehensive error handling and recovery

#### Week 2: Session Commands Foundation
**Days 1-2: Core Session Operations**
- Implement `session list` with detailed status information
- Implement `session new` with tmux integration
- Add session naming convention handling
- Integrate with existing tmux package for session management

**Days 3-4: Advanced Session Operations**
- Implement `session resume` with health validation
- Implement `session kill` with graceful shutdown
- Add batch session operations with filtering
- Implement session state persistence

**Day 5: Session Cleanup and Health**
- Implement `session clean` with stale detection
- Add session health monitoring integration
- Implement automatic recovery mechanisms
- Add comprehensive session lifecycle management

#### Week 3: Polish, Testing, and Integration
**Days 1-2: Enhanced User Experience**
- Implement enhanced output formatting with colors and themes
- Add interactive features with gum integration
- Implement comprehensive help and documentation
- Add shell completion support

**Days 3-4: Testing and Validation**
- Comprehensive unit testing for all new commands
- Integration testing with existing Phase 4.1 functionality
- End-to-end testing of complete workflows
- Performance testing and optimization

**Day 5: Documentation and Final Integration**
- Complete documentation for all new commands
- Integration testing with TUI functionality
- Final validation of success criteria
- Preparation for Phase 4.3 (if applicable)

### 8. File Structure and Components

#### 8.1 New Command Files
```
cmd/ccmgr-ultra/
├── worktree.go          # 500+ lines: Complete worktree command group
├── session.go           # 400+ lines: Complete session command group
└── completion.go        # 200+ lines: Shell completion support
```

#### 8.2 Enhanced CLI Infrastructure
```
internal/cli/
├── interactive.go       # 300+ lines: Gum-based interactive utilities
├── progress.go          # 200+ lines: Enhanced progress indicators
├── batch.go            # 250+ lines: Batch operation utilities
├── confirmation.go     # 150+ lines: User confirmation prompts
└── table.go            # 300+ lines: Enhanced table formatting
```

#### 8.3 Enhanced Integration Packages
```
internal/git/
├── advanced_worktree.go # 400+ lines: Advanced worktree operations
├── remote_advanced.go   # 300+ lines: Enhanced remote operations
└── pr_integration.go    # 200+ lines: Pull request integration

internal/tmux/
├── advanced_session.go  # 350+ lines: Advanced session management
├── health_monitor.go    # 250+ lines: Session health monitoring
└── lifecycle.go         # 200+ lines: Session lifecycle management

internal/claude/
├── advanced_process.go  # 300+ lines: Advanced process management
├── lifecycle.go         # 200+ lines: Process lifecycle integration
└── health_monitor.go    # 200+ lines: Process health monitoring
```

### 9. Testing Strategy

#### 9.1 Unit Testing
```go
// Example test structure
func TestWorktreeCommands(t *testing.T) {
    tests := []struct {
        name     string
        args     []string
        setup    func(*testing.T) string
        validate func(*testing.T, string, error)
    }{
        {
            name: "list_worktrees_table_format",
            args: []string{"worktree", "list", "--format=table"},
            setup: setupTestRepo,
            validate: validateTableOutput,
        },
        {
            name: "create_worktree_with_session",
            args: []string{"worktree", "create", "feature-test", "--start-session"},
            setup: setupTestRepo,
            validate: validateWorktreeCreation,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

**Coverage Areas**:
- Command-line argument parsing and validation
- Flag handling and default values
- Output formatting for all supported formats
- Error handling and recovery scenarios
- Interactive selection and confirmation flows

#### 9.2 Integration Testing
```go
// Example integration test
func TestWorkflowIntegration(t *testing.T) {
    // Test complete workflow: create -> work -> push -> merge -> cleanup
    testCases := []WorkflowTest{
        {
            name: "full_feature_workflow",
            steps: []WorkflowStep{
                {command: "worktree create feature-test --start-session"},
                {command: "session list --worktree=feature-test"},
                {command: "worktree push feature-test --create-pr"},
                {command: "worktree merge feature-test --delete-after"},
                {command: "session clean --dry-run"},
            },
            validate: validateCompleteWorkflow,
        },
    }
}
```

**Coverage Areas**:
- End-to-end command workflows
- Integration with existing Phase 4.1 commands
- Cross-component communication and state consistency
- Configuration loading and override behavior
- TUI and CLI integration points

#### 9.3 User Acceptance Testing
- **Interactive Scenarios**: Testing with real user workflows
- **Automation Scenarios**: Scripting and CI/CD integration testing
- **Error Recovery**: Testing recovery from various failure scenarios
- **Performance**: Testing with large numbers of worktrees and sessions
- **Cross-platform**: Testing on macOS and Linux environments

### 10. Success Criteria

#### 10.1 Functional Requirements ✅
- **Complete Command Coverage**: All specified worktree and session commands implemented and working
- **Integration Compatibility**: Seamless integration with existing Phase 4.1 functionality
- **Workflow Support**: Support for both interactive and automation workflows
- **Output Consistency**: Consistent output formatting across all commands
- **Error Handling**: Comprehensive error handling with recovery suggestions

#### 10.2 User Experience ✅
- **Intuitive Interface**: Command structure following Unix conventions and best practices
- **Fast Performance**: Sub-second response times for list and status operations
- **Clear Feedback**: Progress indication and helpful error messages
- **Interactive Features**: Smooth interactive selection and confirmation flows
- **Documentation**: Comprehensive help text and usage examples

#### 10.3 Technical Integration ✅
- **Package Integration**: Proper integration with all existing internal packages
- **Configuration Consistency**: Consistent configuration handling between CLI and TUI
- **State Management**: Reliable tmux session and git worktree state management
- **Process Management**: Proper Claude Code process lifecycle management
- **Resource Efficiency**: Efficient resource usage and cleanup

#### 10.4 Automation Support ✅
- **Scripting Friendly**: Machine-readable output formats for automation
- **Non-interactive Mode**: Full functionality without interactive prompts
- **Error Codes**: Consistent exit codes for script error handling
- **Batch Operations**: Efficient bulk operations for large-scale management
- **CI/CD Integration**: Suitable for integration in automated pipelines

### 11. Future Enhancements (Post Phase 4.2)

#### 11.1 Advanced Features
- **Shell Completion**: Comprehensive bash, zsh, and fish completion
- **Configuration Management**: Advanced configuration validation and migration
- **Monitoring Dashboard**: Real-time monitoring of all managed sessions
- **Remote Management**: Management of sessions on remote servers
- **Plugin System**: Extensible plugin architecture for custom workflows

#### 11.2 Automation Enhancements
- **Webhook Integration**: Git webhook handlers for automated workflows
- **Scheduled Operations**: Cron-like scheduling for maintenance tasks
- **API Server**: HTTP API for external tool integration
- **Metrics Collection**: Detailed metrics and analytics collection
- **Alerting System**: Configurable alerts for system health issues

#### 11.3 Enterprise Features
- **Multi-user Support**: Support for shared project environments
- **RBAC Integration**: Role-based access control for teams
- **Audit Logging**: Comprehensive audit trail for all operations
- **Backup/Restore**: State backup and restoration capabilities
- **High Availability**: Support for redundant session management

### 12. Risk Mitigation

#### 12.1 Identified Risks
- **Command Complexity**: Risk of CLI commands becoming too complex or confusing
- **Integration Challenges**: Potential conflicts with existing TUI functionality
- **Performance Issues**: Risk of slow performance with large numbers of worktrees/sessions
- **Cross-platform Compatibility**: Ensuring consistent behavior across different platforms
- **Data Loss**: Risk of accidental data loss during worktree or session operations

#### 12.2 Mitigation Strategies
- **Modular Design**: Keep commands focused and single-purpose with clear separation of concerns
- **Extensive Testing**: Comprehensive testing of CLI and TUI integration points
- **Performance Monitoring**: Profile command execution times and optimize bottlenecks
- **Platform Testing**: Regular testing on both macOS and Linux environments
- **Safety Mechanisms**: Multiple confirmation prompts and dry-run modes for destructive operations

#### 12.3 Quality Assurance
- **Code Review**: Thorough code review process for all new functionality
- **Automated Testing**: Comprehensive automated test suite with CI integration
- **User Testing**: Beta testing with real users and workflows
- **Documentation Review**: Thorough review of all documentation and help text
- **Security Review**: Security review of all new components and integrations

## Conclusion

Phase 4.2 represents the completion of the core CLI functionality for ccmgr-ultra, transforming it from a primarily TUI-focused tool into a comprehensive command-line application suitable for both interactive and automated workflows. The implementation builds upon the solid foundation established in Phase 4.1 while adding sophisticated worktree and session management capabilities.

The modular approach ensures that each command group can be implemented and tested independently, while the deep integration with existing packages maintains consistency and reduces code duplication. The focus on user experience, error handling, and automation support makes ccmgr-ultra suitable for a wide range of use cases, from individual developer productivity to enterprise-scale automation.

Upon completion of Phase 4.2, users will have access to a complete toolkit for managing Claude Code sessions and git worktrees through any interface they prefer, with the confidence that comes from comprehensive testing, excellent error handling, and thoughtful user experience design.