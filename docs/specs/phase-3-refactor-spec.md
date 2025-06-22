# Phase 3 Refactor: Connect Menu Interactions to Workflow Commands

## Executive Summary

The Phase 3 menu interaction system in CCMGR Ultra has sophisticated keyboard shortcuts (n/c/r for new/continue/resume sessions) and comprehensive multi-step workflow wizards, but they are currently disconnected. Users can trigger the shortcuts, but they display "Not Implemented" modals instead of launching the carefully designed session creation and management workflows.

This specification outlines the technical refactoring required to connect the menu interactions to their corresponding workflow commands, enabling full Phase 3 functionality as originally intended.

## Current State Analysis

### What Works
- **Keyboard Shortcuts**: The WorktreesModel correctly implements n/c/r key handling
- **Workflow Wizards**: Sophisticated multi-step session and worktree creation wizards exist
- **Modal System**: Complete modal and multi-step wizard framework
- **Backend Integration**: Comprehensive tmux, git, and Claude process management

### What's Broken
- **Message Routing**: Workflow request messages are not handled in the main app
- **Interface Compatibility**: Workflow interfaces don't match actual integration methods
- **Wizard Initialization**: Session and worktree wizards are commented out due to interface issues
- **Placeholder Operations**: Continue/resume operations return dummy messages

### Root Cause Analysis

#### 1. Missing Message Handlers
**Location**: `internal/tui/app.go:255-406` (Update method)
**Issue**: The following messages are not handled:
- `NewSessionRequestedMsg`
- `ContinueSessionRequestedMsg` 
- `ResumeSessionRequestedMsg`

**Current Code**:
```go
// In screens.go WorktreesModel
case "n":
    return m, m.createNewSessionForSelection()  // Returns NewSessionRequestedMsg
case "c":
    return m, m.continueSessionForSelection()   // Returns ContinueSessionRequestedMsg
case "r":
    return m, m.resumeSessionForSelection()     // Returns ResumeSessionRequestedMsg
```

**Problem**: These messages reach app.go but fall through to the default case with no handling.

#### 2. Interface Incompatibility
**Location**: `internal/tui/workflows/sessions.go:20-27`
**Issue**: Workflow Integration interface expects methods that don't exist:

```go
type Integration interface {
    GetAvailableProjects() ([]ProjectInfo, error)        // MISSING
    GetAvailableWorktrees() ([]WorktreeInfo, error)      // MISSING  
    GetDefaultClaudeConfig(projectPath string) (ClaudeConfig, error)  // MISSING
    CreateSession(config SessionConfig) error            // MISSING
    ValidateSessionName(name string) error               // MISSING
    ValidateProjectPath(path string) error               // MISSING
}
```

**Actual Integration** (`internal/tui/integration.go`):
```go
type Integration struct {
    // Has methods like GetAllWorktrees(), GetAllSessions(), etc.
    // But not the workflow-specific interface methods
}
```

#### 3. Commented-out Wizard Initialization
**Location**: `internal/tui/app.go:223-224`
**Issue**: Wizards are disabled due to interface incompatibility:

```go
// Initialize wizards - TODO: Fix interface compatibility
// app.sessionWizard = workflows.NewSessionCreationWizard(integration, modalTheme)
// app.worktreeWizard = workflows.NewWorktreeCreationWizard(integration, modalTheme)
```

#### 4. Placeholder Implementations
**Location**: `internal/tui/workflows/sessions.go:686-709`
**Issue**: Operations return dummy messages instead of real functionality:

```go
func (w *WorktreeSessionIntegration) ContinueSessionInWorktree(worktree WorktreeInfo) tea.Cmd {
    return func() tea.Msg {
        // This would find existing sessions for the worktree
        // For now, return a placeholder message
        return SessionContinueMsg{
            WorktreePath: worktree.Path,
            Success:      true,
            Message:      fmt.Sprintf("Continuing session in %s", worktree.Branch),
        }
    }
}
```

## Technical Requirements

### Functional Requirements
1. **Menu Shortcut Integration**: Keyboard shortcuts (n/c/r) must launch appropriate workflow wizards
2. **Session Discovery**: Continue/resume operations must find and attach to real tmux sessions
3. **Multi-Worktree Support**: Bulk operations must work with selected worktrees
4. **Error Handling**: Proper validation and error reporting throughout workflows
5. **State Management**: UI must reflect current session and worktree states

### Non-Functional Requirements
1. **Performance**: Wizard launching must be responsive (<200ms)
2. **Reliability**: Session operations must handle tmux/git failures gracefully
3. **Usability**: Clear feedback and progress indication during workflows
4. **Maintainability**: Clean interfaces and separation of concerns

### Interface Requirements

#### Integration Adapter Interface
```go
type WorkflowIntegration interface {
    // Project and Worktree Discovery
    GetAvailableProjects() ([]ProjectInfo, error)
    GetAvailableWorktrees() ([]WorktreeInfo, error)
    
    // Session Management
    CreateSession(config SessionConfig) error
    FindSessionsForWorktree(worktreePath string) ([]SessionInfo, error)
    AttachToSession(sessionID string) error
    
    // Validation
    ValidateSessionName(name string) error
    ValidateProjectPath(path string) error
    ValidateWorktreePath(path string) error
    
    // Configuration
    GetDefaultClaudeConfig(projectPath string) (ClaudeConfig, error)
    GetDefaultWorktreeDir(repoPath string) (string, error)
}
```

## Implementation Specification

### Step 1: Create Integration Adapter

**File**: `internal/tui/integration_adapter.go` (NEW)

**Purpose**: Bridge the gap between workflow interface requirements and actual Integration methods.

**Implementation**:
```go
package tui

import (
    "path/filepath"
    "strings"
    "github.com/bcdekker/ccmgr-ultra/internal/tui/workflows"
)

// IntegrationAdapter adapts the TUI Integration to workflow interfaces
type IntegrationAdapter struct {
    integration *Integration
    config      *config.Config
}

// NewIntegrationAdapter creates a new integration adapter
func NewIntegrationAdapter(integration *Integration, config *config.Config) *IntegrationAdapter {
    return &IntegrationAdapter{
        integration: integration,
        config:      config,
    }
}

// GetAvailableProjects derives project list from worktrees
func (a *IntegrationAdapter) GetAvailableProjects() ([]workflows.ProjectInfo, error) {
    worktrees := a.integration.GetAllWorktrees()
    projectMap := make(map[string]workflows.ProjectInfo)
    
    for _, wt := range worktrees {
        if _, exists := projectMap[wt.Repository]; !exists {
            projectMap[wt.Repository] = workflows.ProjectInfo{
                Name:        wt.Repository,
                Path:        filepath.Dir(wt.Path),
                Description: fmt.Sprintf("Repository with %d worktrees", 1),
                HasClaude:   len(wt.ActiveSessions) > 0,
                LastUsed:    wt.LastAccess.Format("2006-01-02 15:04"),
            }
        }
    }
    
    var projects []workflows.ProjectInfo
    for _, project := range projectMap {
        projects = append(projects, project)
    }
    
    return projects, nil
}

// GetAvailableWorktrees wraps existing method with interface conversion
func (a *IntegrationAdapter) GetAvailableWorktrees() ([]workflows.WorktreeInfo, error) {
    worktrees := a.integration.GetAllWorktrees()
    var result []workflows.WorktreeInfo
    
    for _, wt := range worktrees {
        result = append(result, workflows.WorktreeInfo{
            Path:         wt.Path,
            Branch:       wt.Branch,
            ProjectName:  wt.Repository,
            LastAccess:   wt.LastAccess.Format("2006-01-02 15:04"),
            HasChanges:   wt.HasChanges,
        })
    }
    
    return result, nil
}

// Additional adapter methods...
```

### Step 2: Add Workflow Message Handlers

**File**: `internal/tui/app.go`
**Location**: `Update()` method, `tea.KeyMsg` switch statement

**Add after line 342**:
```go
case NewSessionRequestedMsg:
    // Handle new session request from worktree selection
    return m.handleNewSessionRequest(msg)
    
case ContinueSessionRequestedMsg:
    // Handle continue session request
    return m.handleContinueSessionRequest(msg)
    
case ResumeSessionRequestedMsg:
    // Handle resume session request  
    return m.handleResumeSessionRequest(msg)
```

**Handler Methods**:
```go
// handleNewSessionRequest launches the session creation wizard
func (m *AppModel) handleNewSessionRequest(msg NewSessionRequestedMsg) (tea.Model, tea.Cmd) {
    if len(msg.Worktrees) == 1 {
        // Single worktree session creation
        modal := m.workflowFactory.CreateSingleWorktreeSessionWizard(msg.Worktrees[0])
        m.modalManager.ShowModal(modal)
    } else if len(msg.Worktrees) > 1 {
        // Bulk session creation
        modal := m.workflowFactory.CreateBulkWorktreeSessionWizard(msg.Worktrees)
        m.modalManager.ShowModal(modal)
    } else {
        // General session creation
        modal := m.workflowFactory.CreateGeneralSessionWizard()
        m.modalManager.ShowModal(modal)
    }
    return m, nil
}

// handleContinueSessionRequest finds and attaches to existing session
func (m *AppModel) handleContinueSessionRequest(msg ContinueSessionRequestedMsg) (tea.Model, tea.Cmd) {
    return m, func() tea.Msg {
        // Find existing sessions for worktrees
        for _, wt := range msg.Worktrees {
            sessions := m.integration.GetActiveSessionsForWorktree(wt.Path)
            if len(sessions) > 0 {
                // Attach to most recent session
                return m.integration.AttachSession(sessions[0].ID)
            }
        }
        // No sessions found
        return ErrorMsg{Error: fmt.Errorf("no existing sessions found for selected worktrees")}
    }
}

// handleResumeSessionRequest restores paused sessions
func (m *AppModel) handleResumeSessionRequest(msg ResumeSessionRequestedMsg) (tea.Model, tea.Cmd) {
    return m, func() tea.Msg {
        // Implementation for resuming paused sessions
        // This would integrate with tmux session restoration
        return SessionResumedMsg{Success: true}
    }
}
```

### Step 3: Fix Wizard Initialization

**File**: `internal/tui/app.go`
**Location**: `NewAppModel()` function

**Replace lines 223-224**:
```go
// Create integration adapter for workflows
integrationAdapter := NewIntegrationAdapter(integration, config)

// Initialize workflow factory
app.workflowFactory = NewWorkflowFactory(integrationAdapter, modalTheme)

// Initialize wizards with adapter
app.sessionWizard = app.workflowFactory.CreateSessionWizard()
app.worktreeWizard = app.workflowFactory.CreateWorktreeWizard()
```

### Step 4: Create Workflow Factory

**File**: `internal/tui/workflow_factory.go` (NEW)

**Purpose**: Centralized creation and management of workflow wizards.

**Implementation**:
```go
package tui

import (
    "github.com/bcdekker/ccmgr-ultra/internal/tui/modals"
    "github.com/bcdekker/ccmgr-ultra/internal/tui/workflows"
)

// WorkflowFactory creates and manages workflow wizards
type WorkflowFactory struct {
    integration workflows.Integration
    theme       modals.Theme
}

// NewWorkflowFactory creates a new workflow factory
func NewWorkflowFactory(integration workflows.Integration, theme modals.Theme) *WorkflowFactory {
    return &WorkflowFactory{
        integration: integration,
        theme:       theme,
    }
}

// CreateSessionWizard creates a general session creation wizard
func (f *WorkflowFactory) CreateSessionWizard() *workflows.SessionCreationWizard {
    return workflows.NewSessionCreationWizard(f.integration, f.theme)
}

// CreateWorktreeWizard creates a worktree creation wizard
func (f *WorkflowFactory) CreateWorktreeWizard() *workflows.WorktreeCreationWizard {
    return workflows.NewWorktreeCreationWizard(f.integration, f.theme)
}

// CreateSingleWorktreeSessionWizard creates session wizard for specific worktree
func (f *WorkflowFactory) CreateSingleWorktreeSessionWizard(worktree WorktreeInfo) *modals.MultiStepModal {
    integration := workflows.NewWorktreeSessionIntegration(f.integration, f.theme)
    return integration.CreateNewSessionForWorktree(workflows.WorktreeInfo{
        Path:        worktree.Path,
        Branch:      worktree.Branch,
        ProjectName: worktree.Repository,
        LastAccess:  worktree.LastAccess.Format("2006-01-02 15:04"),
        HasChanges:  worktree.HasChanges,
    })
}

// CreateBulkWorktreeSessionWizard creates bulk session wizard
func (f *WorkflowFactory) CreateBulkWorktreeSessionWizard(worktrees []WorktreeInfo) *modals.MultiStepModal {
    integration := workflows.NewWorktreeSessionIntegration(f.integration, f.theme)
    
    var workflowWorktrees []workflows.WorktreeInfo
    for _, wt := range worktrees {
        workflowWorktrees = append(workflowWorktrees, workflows.WorktreeInfo{
            Path:        wt.Path,
            Branch:      wt.Branch,
            ProjectName: wt.Repository,
            LastAccess:  wt.LastAccess.Format("2006-01-02 15:04"),
            HasChanges:  wt.HasChanges,
        })
    }
    
    return integration.CreateBulkSessionsForWorktrees(workflowWorktrees)
}

// CreateGeneralSessionWizard creates a general session wizard
func (f *WorkflowFactory) CreateGeneralSessionWizard() *modals.MultiStepModal {
    wizard := f.CreateSessionWizard()
    return wizard.CreateWizard()
}
```

### Step 5: Implement Real Session Operations

**File**: `internal/tui/integration.go`

**Add methods**:
```go
// FindSessionsForWorktree finds existing sessions for a worktree
func (i *Integration) FindSessionsForWorktree(worktreePath string) ([]SessionInfo, error) {
    i.mu.RLock()
    defer i.mu.RUnlock()
    
    var result []SessionInfo
    for _, session := range i.sessions {
        if session.Directory == worktreePath {
            result = append(result, session)
        }
    }
    
    return result, nil
}

// AttachToExistingSession attaches to an existing tmux session
func (i *Integration) AttachToExistingSession(sessionID string) tea.Cmd {
    return func() tea.Msg {
        err := i.tmuxMgr.AttachSession(sessionID)
        if err != nil {
            return ErrorMsg{Error: err}
        }
        return SessionAttachedMsg{SessionID: sessionID}
    }
}

// ResumeSession resumes a paused session
func (i *Integration) ResumeSession(sessionID string) tea.Cmd {
    return func() tea.Msg {
        // Implementation would restore session state
        // For now, treat as attach operation
        err := i.tmuxMgr.AttachSession(sessionID)
        if err != nil {
            return ErrorMsg{Error: err}
        }
        return SessionResumedMsg{SessionID: sessionID}
    }
}
```

**Update workflow methods**:
```go
// Update ContinueSessionInWorktree to use real operations
func (w *WorktreeSessionIntegration) ContinueSessionInWorktree(worktree WorktreeInfo) tea.Cmd {
    return func() tea.Msg {
        // Find existing sessions for the worktree
        sessions, err := w.integration.FindSessionsForWorktree(worktree.Path)
        if err != nil {
            return SessionContinueMsg{
                WorktreePath: worktree.Path,
                Success:      false,
                Message:      fmt.Sprintf("Error finding sessions: %v", err),
            }
        }
        
        if len(sessions) == 0 {
            return SessionContinueMsg{
                WorktreePath: worktree.Path,
                Success:      false,
                Message:      "No existing sessions found for this worktree",
            }
        }
        
        // Attach to most recent session
        return w.integration.AttachToExistingSession(sessions[0].ID)
    }
}
```

### Step 6: Update App Model Structure

**File**: `internal/tui/app.go`

**Add to AppModel struct**:
```go
type AppModel struct {
    // ... existing fields ...
    
    // Workflow management
    workflowFactory    *WorkflowFactory        // NEW
    integrationAdapter *IntegrationAdapter    // NEW
    sessionWizard      *workflows.SessionCreationWizard
    worktreeWizard     *workflows.WorktreeCreationWizard
    
    // ... rest of fields ...
}
```

## Data Flow Architecture

### Message Flow
```
User Input (n/c/r) → WorktreesModel 
  ↓
Workflow Message (NewSessionRequestedMsg/etc.) → AppModel.Update()
  ↓
Message Handler → WorkflowFactory
  ↓
Wizard Creation → Modal Display
  ↓
User Completion → Integration Operations
  ↓
Backend Action (tmux/git) → Success/Error Response
```

### Component Relationships
```
WorktreesModel ←→ AppModel ←→ WorkflowFactory
                     ↓
               IntegrationAdapter ←→ Integration
                     ↓                    ↓
            SessionWizard          BackendManagers
            WorktreeWizard         (tmux/git/claude)
```

## Testing Strategy

### Unit Tests
1. **IntegrationAdapter Tests**: Verify interface conversion and method delegation
2. **WorkflowFactory Tests**: Validate wizard creation with different parameters
3. **Message Handler Tests**: Ensure proper routing and error handling
4. **Session Operation Tests**: Test session discovery, attachment, and resumption

### Integration Tests
1. **End-to-End Workflow Tests**: Complete keyboard shortcut → wizard → backend operation flows
2. **Multi-Worktree Tests**: Bulk operations with various worktree selections
3. **Error Scenario Tests**: Network failures, missing sessions, invalid configurations
4. **State Management Tests**: UI state consistency during workflow execution

### Manual Testing Checklist
- [ ] Press 'n' on worktree → Session creation wizard launches
- [ ] Press 'c' on worktree with existing session → Attaches to session
- [ ] Press 'r' on worktree with paused session → Resumes session
- [ ] Multi-select worktrees + 'n' → Bulk session creation wizard
- [ ] Error handling displays appropriate messages
- [ ] Wizard completion creates real sessions
- [ ] Backend integration works with tmux/git

## Risk Assessment

### High-Risk Areas
1. **Interface Compatibility**: Changes to workflow interfaces could break existing code
2. **Session Management**: Tmux operations could fail or interfere with existing sessions
3. **State Synchronization**: UI state and backend state could become inconsistent

### Mitigation Strategies
1. **Adapter Pattern**: Isolates interface changes from core functionality
2. **Comprehensive Testing**: Both unit and integration tests prevent regressions
3. **Graceful Degradation**: Error handling ensures UI remains functional during failures
4. **Incremental Implementation**: Step-by-step approach allows validation at each stage

### Rollback Plan
1. **Git Branching**: All changes implemented in feature branch
2. **Feature Flags**: Ability to disable new functionality if issues arise
3. **Configuration**: Option to fall back to placeholder implementations

## Success Criteria

### Functional Criteria
- [ ] All keyboard shortcuts (n/c/r) launch appropriate workflows
- [ ] Session creation wizard creates functional tmux sessions
- [ ] Continue operation finds and attaches to existing sessions
- [ ] Resume operation restores paused sessions
- [ ] Multi-worktree operations work correctly
- [ ] Error handling provides clear user feedback

### Quality Criteria
- [ ] No performance degradation in UI responsiveness
- [ ] Clean separation of concerns between UI and backend
- [ ] Comprehensive test coverage (>80%)
- [ ] Documentation updated for new functionality
- [ ] Code review approval from team

### User Experience Criteria
- [ ] Workflows feel smooth and intuitive
- [ ] Error messages are helpful and actionable
- [ ] Progress indication during long operations
- [ ] Consistent behavior across different scenarios

## Implementation Timeline

### Phase 1: Foundation (Days 1-2)
- Create IntegrationAdapter
- Implement WorkflowFactory
- Add basic message handlers

### Phase 2: Core Functionality (Days 3-4)
- Fix wizard initialization
- Implement real session operations
- Update workflow methods

### Phase 3: Testing & Refinement (Days 5-6)
- Write comprehensive tests
- Manual testing and bug fixes
- Performance optimization

### Phase 4: Documentation & Review (Day 7)
- Update documentation
- Code review and refinements
- Final testing and validation

## Conclusion

This refactor will transform the Phase 3 menu interactions from placeholder implementations to fully functional workflow commands. The approach prioritizes maintainability through the adapter pattern, reliability through comprehensive testing, and user experience through proper error handling and feedback.

The implementation connects sophisticated UI components that already exist to robust backend systems, enabling the full Phase 3 functionality as originally envisioned in the CCMGR Ultra design.