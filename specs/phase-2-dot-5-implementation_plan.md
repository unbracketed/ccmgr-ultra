# Phase 2.5 Implementation Plan: Status Hooks and Worktree Lifecycle Hooks

## Overview
This document provides a comprehensive implementation plan for Phase 2.5 of ccmgr-ultra, focusing on implementing both Claude Code status hooks and worktree lifecycle hooks. This phase establishes a flexible hook system that allows users to execute custom scripts during critical events in the application lifecycle.

## Phase 2.5 Requirements Analysis

From `steps-to-implement.md`, Phase 2.5 must implement:
- [ ] Execute configured scripts on state changes
- [ ] Pass environment variables (CCMANAGER_WORKTREE, CCMANAGER_WORKTREE_BRANCH, CCMANAGER_NEW_STATE, CCMANAGER_SESSION_ID)
- [ ] Handle hook script errors gracefully
- [ ] Support async hook execution

**Additional Requirements from User Feedback:**
- [ ] Worktree creation hooks for bootstrapping new environments
- [ ] Worktree activation hooks for session continuation/resumption
- [ ] Support for isolated execution environments per worktree
- [ ] File transfer capabilities for worktree-excluded files (e.g., .env)

## Hook Types and Use Cases

### 1. Status Hooks
Execute scripts when Claude Code process state changes:
- **Idle Hook**: Claude Code is idle, waiting for input
- **Busy Hook**: Claude Code is actively processing commands
- **Waiting Hook**: Claude Code is waiting for user confirmation

**Common Use Cases:**
- Update status tracking systems (e.g., using skate)
- Send notifications about long-running operations
- Log activity for time tracking
- Trigger automated workflows based on state

### 2. Worktree Lifecycle Hooks

#### Creation Hook
Executes after a new worktree is successfully created:
- **Use Cases:**
  - Initialize isolated development environment
  - Copy configuration files (.env, secrets)
  - Install project dependencies
  - Set up database schemas
  - Configure IDE settings

#### Activation Hook  
Executes when a worktree is activated (new session, continue, or resume):
- **Use Cases:**
  - Start development services (database, API servers)
  - Ensure environment variables are loaded
  - Validate dependencies are up to date
  - Restore previous session state
  - Update activity tracking

## Implementation Strategy

### Architecture Overview

```
internal/hooks/
├── executor.go       # Core hook execution logic
├── types.go         # Hook type definitions
├── status.go        # Status hook implementation
├── worktree.go      # Worktree hook implementation
├── config.go        # Hook configuration
├── environment.go   # Environment variable management
└── errors.go        # Error handling and recovery
```

### Core Components

#### 1. Hook Executor
```go
type HookExecutor interface {
    Execute(ctx context.Context, hook HookConfig, env Environment) error
    ExecuteAsync(hook HookConfig, env Environment) <-chan error
}

type Hook struct {
    Type        HookType
    Enabled     bool
    Script      string
    Timeout     time.Duration
    Async       bool
    Environment map[string]string
}
```

#### 2. Environment Management
```go
type Environment struct {
    // Common variables
    WorktreePath   string
    WorktreeBranch string
    ProjectName    string
    SessionID      string
    
    // Hook-specific variables
    Variables map[string]string
}
```

#### 3. Error Handling Strategy
- Graceful degradation when hooks fail
- Detailed error logging without interrupting main workflow
- Timeout enforcement to prevent hanging
- Recovery mechanisms for critical hooks

## Detailed Implementation Steps

### Step 1: Update Configuration Schema (2 days)

**File: `internal/config/schema.go`**

Add new configuration structures:
```go
// WorktreeHooksConfig defines worktree lifecycle hooks
type WorktreeHooksConfig struct {
    Enabled        bool       `yaml:"enabled" json:"enabled"`
    CreationHook   HookConfig `yaml:"creation" json:"creation"`
    ActivationHook HookConfig `yaml:"activation" json:"activation"`
}

// Update main Config struct
type Config struct {
    // ... existing fields ...
    WorktreeHooks WorktreeHooksConfig `yaml:"worktree_hooks" json:"worktree_hooks"`
}
```

**Validation Requirements:**
- Ensure script paths are valid
- Validate timeout ranges (1-300 seconds)
- Check for circular dependencies
- Verify environment variable names

### Step 2: Core Hook Executor Implementation (3 days)

**File: `internal/hooks/executor.go`**

Key Implementation Points:
1. **Process Execution**
   - Use `exec.CommandContext` for timeout support
   - Set working directory to worktree path
   - Capture stdout/stderr for debugging
   - Handle different shell types (bash, zsh, sh)

2. **Async Execution**
   - Implement goroutine-based async execution
   - Use channels for error reporting
   - Ensure proper cleanup on cancellation

3. **Environment Setup**
   - Merge hook-specific environment with system env
   - Expand shell variables in script paths
   - Set standard CCMGR_* variables

**File: `internal/hooks/types.go`**
```go
type HookType int

const (
    HookTypeStatusIdle HookType = iota
    HookTypeStatusBusy
    HookTypeStatusWaiting
    HookTypeWorktreeCreation
    HookTypeWorktreeActivation
)

type HookEvent struct {
    Type      HookType
    Timestamp time.Time
    Context   HookContext
}

type HookContext struct {
    WorktreePath   string
    WorktreeBranch string
    ProjectName    string
    SessionID      string
    SessionType    string // "new", "continue", "resume"
    OldState       string
    NewState       string
    CustomVars     map[string]string
}
```

### Step 3: Status Hooks Implementation (2 days)

**File: `internal/hooks/status.go`**

Integration with Claude monitoring system:
1. **State Change Detection**
   - Subscribe to ProcessTracker state changes
   - Map process states to hook types
   - Debounce rapid state changes

2. **Environment Variables**
   ```
   CCMANAGER_WORKTREE
   CCMANAGER_WORKTREE_BRANCH  
   CCMANAGER_NEW_STATE
   CCMANAGER_OLD_STATE
   CCMANAGER_SESSION_ID
   CCMANAGER_TIMESTAMP
   ```

3. **Example Hook Script**
   ```bash
   #!/bin/bash
   # Status tracking hook using skate
   
   skate set "${CCMANAGER_WORKTREE_BRANCH}@ccmgr-status" "${CCMANAGER_NEW_STATE}"
   skate set "${CCMANAGER_WORKTREE_BRANCH}@ccmgr-projects" "${CCMANAGER_WORKTREE#$HOME/code/}"
   skate set "${CCMANAGER_WORKTREE}@ccmgr-sessions" "${CCMANAGER_SESSION_ID}"
   ```

### Step 4: Worktree Hooks Implementation (3 days)

**File: `internal/hooks/worktree.go`**

#### Creation Hook Implementation
1. **Trigger Point**: After `WorktreeManager.CreateWorktree()` success
2. **Environment Variables**:
   ```
   CCMGR_WORKTREE_PATH
   CCMGR_WORKTREE_BRANCH
   CCMGR_PROJECT_NAME
   CCMGR_WORKTREE_TYPE=new
   CCMGR_PARENT_PATH
   CCMGR_TIMESTAMP
   ```

3. **Example Creation Hook**:
   ```bash
   #!/bin/bash
   # Bootstrap new worktree environment
   
   cd "$CCMGR_WORKTREE_PATH"
   
   # Copy environment files from parent
   if [ -f "$CCMGR_PARENT_PATH/.env" ]; then
       cp "$CCMGR_PARENT_PATH/.env" .
   fi
   
   # Install dependencies
   if [ -f "package.json" ]; then
       npm install
   elif [ -f "requirements.txt" ]; then
       python -m venv venv
       ./venv/bin/pip install -r requirements.txt
   fi
   
   # Initialize local services
   if [ -f "docker-compose.yml" ]; then
       docker-compose up -d
   fi
   ```

#### Activation Hook Implementation
1. **Trigger Points**:
   - `SessionManager.CreateSession()` - New session
   - `SessionManager.ContinueSession()` - Continue existing
   - `SessionManager.ResumeSession()` - Resume after break

2. **Environment Variables**:
   ```
   CCMGR_WORKTREE_PATH
   CCMGR_WORKTREE_BRANCH
   CCMGR_PROJECT_NAME
   CCMGR_SESSION_ID
   CCMGR_SESSION_TYPE (new/continue/resume)
   CCMGR_PREVIOUS_STATE
   CCMGR_TIMESTAMP
   ```

3. **Example Activation Hook**:
   ```bash
   #!/bin/bash
   # Ensure development environment is ready
   
   cd "$CCMGR_WORKTREE_PATH"
   
   # Start services based on session type
   case "$CCMGR_SESSION_TYPE" in
       "new")
           echo "Starting fresh development session..."
           # Start all services
           ;;
       "continue"|"resume")
           echo "Resuming development session..."
           # Check service health and restart if needed
           ;;
   esac
   
   # Ensure database is running
   if ! docker ps | grep -q postgres; then
       docker-compose up -d postgres
   fi
   
   # Load environment
   if [ -f ".env" ]; then
       export $(cat .env | xargs)
   fi
   ```

### Step 5: Integration Points (2 days)

#### 1. Worktree Manager Integration
**File: `internal/git/worktree.go`**

Update `CreateWorktree` method:
```go
func (wm *WorktreeManager) CreateWorktree(branch string, opts WorktreeOptions) (*WorktreeInfo, error) {
    // ... existing worktree creation logic ...
    
    // Execute creation hook if enabled
    if wm.hooks != nil && wm.config.WorktreeHooks.CreationHook.Enabled {
        hookCtx := hooks.HookContext{
            WorktreePath:   worktreeInfo.Path,
            WorktreeBranch: branch,
            ProjectName:    wm.getProjectName(),
            CustomVars: map[string]string{
                "CCMGR_PARENT_PATH": wm.repo.RootPath,
                "CCMGR_WORKTREE_TYPE": "new",
            },
        }
        
        if err := wm.hooks.ExecuteWorktreeCreationHook(hookCtx); err != nil {
            // Log error but don't fail worktree creation
            log.Printf("Warning: worktree creation hook failed: %v", err)
        }
    }
    
    return worktreeInfo, nil
}
```

#### 2. Tmux Session Integration
**File: `internal/tmux/session.go`**

Update session management methods:
```go
func (sm *SessionManager) CreateSession(opts SessionOptions) (*Session, error) {
    // ... existing session creation logic ...
    
    // Execute activation hook
    if sm.hooks != nil && sm.config.WorktreeHooks.ActivationHook.Enabled {
        hookCtx := hooks.HookContext{
            WorktreePath:   opts.WorkingDir,
            WorktreeBranch: opts.Branch,
            ProjectName:    opts.Project,
            SessionID:      session.ID,
            SessionType:    "new",
        }
        
        if err := sm.hooks.ExecuteWorktreeActivationHook(hookCtx); err != nil {
            log.Printf("Warning: worktree activation hook failed: %v", err)
        }
    }
    
    return session, nil
}
```

#### 3. Claude Process Monitor Integration
**File: `internal/claude/monitor.go`**

Subscribe to state changes and trigger status hooks:
```go
func (m *Monitor) onStateChange(event StateChangeEvent) {
    // ... existing state change logic ...
    
    // Execute status hook based on new state
    if m.hooks != nil && m.config.StatusHooks.Enabled {
        var hookType hooks.HookType
        switch event.NewState {
        case StateIdle:
            hookType = hooks.HookTypeStatusIdle
        case StateBusy:
            hookType = hooks.HookTypeStatusBusy
        case StateWaiting:
            hookType = hooks.HookTypeStatusWaiting
        default:
            return
        }
        
        hookCtx := hooks.HookContext{
            WorktreePath:   event.WorkingDir,
            WorktreeBranch: event.Branch,
            SessionID:      event.SessionID,
            OldState:       event.OldState.String(),
            NewState:       event.NewState.String(),
        }
        
        if err := m.hooks.ExecuteStatusHook(hookType, hookCtx); err != nil {
            log.Printf("Warning: status hook failed: %v", err)
        }
    }
}
```

### Step 6: Error Handling and Recovery (1 day)

**File: `internal/hooks/errors.go`**

Implement comprehensive error handling:

1. **Timeout Handling**
   ```go
   type TimeoutError struct {
       Hook    string
       Timeout time.Duration
   }
   
   func (e TimeoutError) Error() string {
       return fmt.Sprintf("hook %s timed out after %v", e.Hook, e.Timeout)
   }
   ```

2. **Script Errors**
   - Capture exit codes
   - Log stderr output
   - Provide helpful error messages

3. **Recovery Strategies**
   - Retry with exponential backoff for transient failures
   - Skip hook and continue for non-critical operations
   - Emergency fallback scripts

### Step 7: Testing Strategy (2 days)

#### Unit Tests
- Mock script execution
- Test timeout enforcement
- Verify environment variable passing
- Test error handling paths

#### Integration Tests
- Test with real worktree operations
- Verify tmux session integration
- Test Claude state monitoring
- End-to-end hook execution

#### Test Scenarios
1. **Happy Path**
   - Creation hook succeeds
   - Activation hook succeeds
   - Status hooks trigger correctly

2. **Error Cases**
   - Script not found
   - Script times out
   - Script returns non-zero exit
   - Invalid environment variables

3. **Edge Cases**
   - Rapid state changes
   - Concurrent hook execution
   - System resource constraints

## Configuration Examples

### Complete Hook Configuration
```yaml
# ~/.ccmgr-ultra/config.yaml
status_hooks:
  enabled: true
  idle:
    enabled: true
    script: "~/.config/ccmgr-ultra/hooks/status.sh"
    timeout: 30
    async: true
  busy:
    enabled: true
    script: "~/.config/ccmgr-ultra/hooks/status.sh"
    timeout: 30
    async: true
  waiting:
    enabled: true
    script: "~/.config/ccmgr-ultra/hooks/status.sh"
    timeout: 30
    async: true

worktree_hooks:
  enabled: true
  creation:
    enabled: true
    script: "~/.config/ccmgr-ultra/hooks/bootstrap.sh"
    timeout: 300  # 5 minutes for dependency installation
    async: false  # Wait for completion
  activation:
    enabled: true
    script: "~/.config/ccmgr-ultra/hooks/activate.sh"
    timeout: 60
    async: true   # Don't block session start
```

### Hook Script Templates

#### Bootstrap Script Template
```bash
#!/bin/bash
# ~/.config/ccmgr-ultra/hooks/bootstrap.sh
set -e

echo "Bootstrapping worktree: $CCMGR_WORKTREE_PATH"

# Copy configuration files
for file in .env .env.local .secrets; do
    if [ -f "$CCMGR_PARENT_PATH/$file" ]; then
        cp "$CCMGR_PARENT_PATH/$file" "$CCMGR_WORKTREE_PATH/"
        echo "Copied $file"
    fi
done

# Language-specific setup
cd "$CCMGR_WORKTREE_PATH"

# Node.js projects
if [ -f "package.json" ]; then
    echo "Installing npm dependencies..."
    npm install
fi

# Python projects
if [ -f "requirements.txt" ] || [ -f "pyproject.toml" ]; then
    echo "Setting up Python environment..."
    python -m venv .venv
    .venv/bin/pip install -r requirements.txt
fi

# Go projects
if [ -f "go.mod" ]; then
    echo "Downloading Go modules..."
    go mod download
fi

# Docker services
if [ -f "docker-compose.yml" ]; then
    echo "Starting Docker services..."
    docker-compose up -d
fi

echo "Bootstrap complete!"
```

#### Activation Script Template
```bash
#!/bin/bash
# ~/.config/ccmgr-ultra/hooks/activate.sh

echo "Activating worktree session: $CCMGR_SESSION_TYPE"

cd "$CCMGR_WORKTREE_PATH"

# Ensure services are running
if [ -f "docker-compose.yml" ]; then
    # Check if services are healthy
    if ! docker-compose ps | grep -q "Up"; then
        echo "Starting services..."
        docker-compose up -d
    fi
fi

# Session-specific setup
case "$CCMGR_SESSION_TYPE" in
    "new")
        echo "Welcome to $CCMGR_PROJECT_NAME!"
        # Show project info, todos, etc.
        ;;
    "continue"|"resume")
        echo "Resuming work on $CCMGR_WORKTREE_BRANCH"
        # Could show git status, last commits, etc.
        ;;
esac

# Update activity tracking
if command -v skate &> /dev/null; then
    skate set "ccmgr-last-active" "$CCMGR_WORKTREE_PATH"
fi
```

## Performance Considerations

1. **Hook Execution**
   - Use async execution for non-critical hooks
   - Implement proper timeout enforcement
   - Cache script validation results

2. **Resource Management**
   - Limit concurrent hook executions
   - Monitor resource usage during hooks
   - Implement circuit breakers for failing hooks

3. **Optimization Strategies**
   - Batch environment variable setup
   - Reuse shell processes where possible
   - Implement hook result caching

## Security Considerations

1. **Script Validation**
   - Verify script ownership and permissions
   - Prevent path traversal attacks
   - Sanitize environment variables

2. **Execution Isolation**
   - Run hooks with limited privileges
   - Use separate process groups
   - Implement resource limits

3. **Sensitive Data**
   - Never log sensitive environment variables
   - Secure storage for hook configurations
   - Audit hook executions

## Integration Points

### With Git Worktree Management (Phase 2.3)
- Trigger creation hooks after successful worktree creation
- Pass worktree metadata to hooks
- Handle hook failures gracefully

### With Tmux Integration (Phase 2.2)
- Execute activation hooks during session lifecycle
- Coordinate hook execution with session state
- Share session context with hooks

### With Claude Process Monitoring (Phase 2.4)
- Subscribe to state change events
- Execute status hooks asynchronously
- Maintain hook execution history

### With TUI (Phase 3)
- Show hook execution status
- Allow hook management from UI
- Display hook failure notifications

## Validation Criteria

### Functional Requirements
- [ ] All hook types execute correctly with proper environment
- [ ] Timeout enforcement works reliably
- [ ] Async execution doesn't block main operations
- [ ] Error handling prevents cascading failures
- [ ] Configuration validation catches common mistakes

### Performance Requirements
- [ ] Hook execution overhead <100ms for sync hooks
- [ ] Async hooks don't impact UI responsiveness
- [ ] Resource usage remains bounded
- [ ] No memory leaks during extended operation

### Reliability Requirements
- [ ] Graceful handling of missing scripts
- [ ] Recovery from hook failures
- [ ] Proper cleanup of hook processes
- [ ] Consistent behavior across platforms

## Timeline Estimate

- **Days 1-2**: Configuration schema updates and validation
- **Days 3-5**: Core hook executor implementation
- **Days 6-7**: Status hooks implementation
- **Days 8-10**: Worktree hooks implementation
- **Days 11-12**: Integration with existing components
- **Days 13-14**: Testing, documentation, and examples

**Total Estimated Duration: 14 days (2.8 weeks)**

## Dependencies

### Internal Dependencies
- Configuration system (Phase 2.1) - Must be completed
- Tmux integration (Phase 2.2) - For session hooks
- Git worktree management (Phase 2.3) - For worktree hooks
- Claude process monitoring (Phase 2.4) - For status hooks

### External Dependencies
- Go standard library (os/exec, context, time)
- Shell availability (bash/sh)
- Optional: skate for state management

## Risk Assessment

### High Risk
- Script execution security vulnerabilities
- Timeout handling complexity
- Cross-platform compatibility issues

### Medium Risk
- Hook script debugging difficulties
- Performance impact of synchronous hooks
- Environment variable conflicts

### Low Risk
- Configuration migration
- Documentation clarity
- Example script maintenance

## Success Metrics

1. **Functionality**: All hook types execute reliably
2. **Performance**: <1% overhead on operations
3. **Reliability**: <0.1% hook execution failures
4. **Usability**: Clear documentation and examples
5. **Adoption**: Users successfully implement custom workflows

## Future Enhancements

1. **Hook Marketplace**
   - Community-contributed hook scripts
   - Verified security reviews
   - Easy installation mechanism

2. **Advanced Features**
   - Conditional hook execution
   - Hook chaining and dependencies
   - Built-in hook templates

3. **Monitoring and Analytics**
   - Hook execution metrics
   - Performance profiling
   - Usage analytics

This comprehensive implementation plan ensures that ccmgr-ultra provides a flexible and powerful hook system that enables users to customize their development workflow while maintaining isolation between worktrees and sessions.