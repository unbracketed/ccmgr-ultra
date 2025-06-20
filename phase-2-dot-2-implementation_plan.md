# Phase 2, Part 2 Implementation Plan: Tmux Integration

## Overview
This document provides a comprehensive implementation and validation plan for Phase 2, Part 2: Tmux Integration (internal/tmux/). This component is critical for session management and Claude Code process monitoring within tmux sessions.

## Requirements Analysis (from steps-to-implement.md)

### 2.2 Tmux Integration (internal/tmux/)
- Implement tmux session management:
  - Create new sessions with standardized naming (project-worktree-branch)
  - List existing sessions
  - Attach/detach from sessions
  - Kill sessions
- Monitor Claude Code processes within tmux sessions
- Handle session state persistence

## Implementation Strategy

### File Structure
```
internal/tmux/
├── session.go      # Core session management operations
├── monitor.go      # Claude Code process monitoring
├── naming.go       # Session naming conventions
└── state.go        # Session state persistence
```

## Detailed Implementation Plan

### 1. Session Management (internal/tmux/session.go)

#### 1.1 Core Session Operations
```go
type SessionManager struct {
    config *config.Config
    state  *SessionState
}

type Session struct {
    ID          string
    Name        string
    Project     string
    Worktree    string
    Branch      string
    Directory   string
    Created     time.Time
    LastAccess  time.Time
    Active      bool
}
```

**Functions to implement:**
- `NewSessionManager(config *config.Config) *SessionManager`
- `CreateSession(project, worktree, branch, directory string) (*Session, error)`
- `ListSessions() ([]*Session, error)`
- `GetSession(sessionID string) (*Session, error)`
- `AttachSession(sessionID string) error`
- `DetachSession(sessionID string) error`
- `KillSession(sessionID string) error`
- `IsSessionActive(sessionID string) (bool, error)`

#### 1.2 Session Naming Convention (internal/tmux/naming.go)
**Standard format:** `ccmgr-{project}-{worktree}-{branch}`

```go
type NamingConvention struct {
    Pattern string
    MaxLength int
}

func GenerateSessionName(project, worktree, branch string) string
func ParseSessionName(sessionName string) (project, worktree, branch string, err error)
func ValidateSessionName(name string) bool
func SanitizeNameComponent(component string) string
```

#### 1.3 Tmux Command Interface
```go
type TmuxCmd struct {
    executable string
}

func (t *TmuxCmd) NewSession(name, startDir string) error
func (t *TmuxCmd) ListSessions() ([]string, error)
func (t *TmuxCmd) HasSession(name string) (bool, error)
func (t *TmuxCmd) AttachSession(name string) error
func (t *TmuxCmd) KillSession(name string) error
func (t *TmuxCmd) SendKeys(session, keys string) error
func (t *TmuxCmd) GetSessionPanes(session string) ([]string, error)
```

### 2. Claude Code Process Monitoring (internal/tmux/monitor.go)

#### 2.1 Process State Detection
```go
type ProcessState int

const (
    StateUnknown ProcessState = iota
    StateIdle
    StateBusy
    StateWaiting
    StateError
)

type ProcessMonitor struct {
    sessions    map[string]*MonitoredSession
    stateHooks  []StateHook
    pollInterval time.Duration
}

type MonitoredSession struct {
    SessionID     string
    ProcessPID    int
    CurrentState  ProcessState
    LastStateChange time.Time
    StateHistory  []StateChange
}

type StateChange struct {
    From      ProcessState
    To        ProcessState
    Timestamp time.Time
    Trigger   string
}
```

**Functions to implement:**
- `NewProcessMonitor(config *config.Config) *ProcessMonitor`
- `StartMonitoring(sessionID string) error`
- `StopMonitoring(sessionID string) error`
- `GetProcessState(sessionID string) (ProcessState, error)`
- `GetProcessPID(sessionID string) (int, error)`
- `RegisterStateHook(hook StateHook)`
- `DetectStateChange(sessionID string) (bool, ProcessState, error)`

#### 2.2 State Detection Methods
```go
// Method 1: Process monitoring via ps/proc
func detectStateByProcess(pid int) (ProcessState, error)

// Method 2: Output parsing (parsing tmux capture-pane)
func detectStateByOutput(sessionID string) (ProcessState, error)

// Method 3: File monitoring (claude code status files)
func detectStateByStatusFile(sessionID string) (ProcessState, error)

// Method 4: Combined heuristic approach
func detectStateCombined(sessionID string, pid int) (ProcessState, error)
```

#### 2.3 State Patterns Recognition
```go
type StatePattern struct {
    Pattern     string
    State       ProcessState
    Confidence  float64
}

var statePatterns = []StatePattern{
    {"claude>", StateIdle, 0.9},
    {"Processing...", StateBusy, 0.8},
    {"Waiting for input", StateWaiting, 0.9},
    {"Error:", StateError, 0.95},
    // More patterns based on Claude Code output
}
```

### 3. Session State Persistence (internal/tmux/state.go)

#### 3.1 State Storage
```go
type SessionState struct {
    FilePath string
    Sessions map[string]*PersistedSession
    mutex    sync.RWMutex
}

type PersistedSession struct {
    ID            string            `json:"id"`
    Name          string            `json:"name"`
    Project       string            `json:"project"`
    Worktree      string            `json:"worktree"`
    Branch        string            `json:"branch"`
    Directory     string            `json:"directory"`
    Created       time.Time         `json:"created"`
    LastAccess    time.Time         `json:"last_access"`
    LastState     ProcessState      `json:"last_state"`
    Environment   map[string]string `json:"environment"`
    Metadata      map[string]interface{} `json:"metadata"`
}
```

**Functions to implement:**
- `LoadState(filePath string) (*SessionState, error)`
- `SaveState() error`
- `AddSession(session *PersistedSession) error`
- `RemoveSession(sessionID string) error`
- `UpdateSession(sessionID string, updates map[string]interface{}) error`
- `GetSession(sessionID string) (*PersistedSession, error)`
- `ListSessions() []*PersistedSession`
- `CleanupStaleEntries(maxAge time.Duration) error`

### 4. Integration Points

#### 4.1 Configuration Integration
```go
type TmuxConfig struct {
    SessionPrefix    string            `yaml:"session_prefix"`
    NamingPattern   string            `yaml:"naming_pattern"`
    MaxSessionName  int               `yaml:"max_session_name"`
    MonitorInterval time.Duration     `yaml:"monitor_interval"`
    StateFile       string            `yaml:"state_file"`
    DefaultEnv      map[string]string `yaml:"default_env"`
    AutoCleanup     bool              `yaml:"auto_cleanup"`
    CleanupAge      time.Duration     `yaml:"cleanup_age"`
}
```

#### 4.2 Hook System Integration
```go
type StateHook interface {
    OnStateChange(sessionID string, from, to ProcessState) error
}

func (pm *ProcessMonitor) executeHooks(sessionID string, from, to ProcessState) {
    env := map[string]string{
        "CCMANAGER_SESSION_ID": sessionID,
        "CCMANAGER_OLD_STATE":  from.String(),
        "CCMANAGER_NEW_STATE":  to.String(),
        "CCMANAGER_WORKTREE":   session.Worktree,
        "CCMANAGER_WORKTREE_BRANCH": session.Branch,
    }
    // Execute configured hooks
}
```

## Validation Plan

### 1. Unit Testing

#### 1.1 Session Management Tests
```go
func TestCreateSession(t *testing.T)
func TestListSessions(t *testing.T)
func TestAttachDetachSession(t *testing.T)
func TestKillSession(t *testing.T)
func TestSessionNaming(t *testing.T)
func TestSessionNameSanitization(t *testing.T)
```

#### 1.2 Process Monitoring Tests
```go
func TestProcessStateDetection(t *testing.T)
func TestStateChangeDetection(t *testing.T)
func TestStatePatternMatching(t *testing.T)
func TestMonitoringLifecycle(t *testing.T)
```

#### 1.3 State Persistence Tests
```go
func TestStateSaveLoad(t *testing.T)
func TestStateCleanup(t *testing.T)
func TestConcurrentStateAccess(t *testing.T)
func TestStateCorruption(t *testing.T)
```

### 2. Integration Testing

#### 2.1 Mock Tmux Testing
```go
type MockTmux struct {
    sessions map[string]bool
    outputs  map[string]string
}

func (m *MockTmux) NewSession(name, dir string) error
func (m *MockTmux) ListSessions() ([]string, error)
func (m *MockTmux) CapturePane(session string) (string, error)
```

#### 2.2 End-to-End Workflow Tests
```go
func TestFullSessionLifecycle(t *testing.T) {
    // Create session -> Monitor process -> Detect states -> Clean up
}

func TestMultipleSessionsManagement(t *testing.T) {
    // Create multiple sessions -> Monitor all -> Handle state changes
}

func TestSessionRecoveryAfterRestart(t *testing.T) {
    // Create sessions -> Simulate restart -> Recover state
}
```

### 3. Performance Testing

#### 3.1 Monitoring Performance
```go
func BenchmarkStateDetection(b *testing.B)
func BenchmarkMultiSessionMonitoring(b *testing.B)
func TestMonitoringResourceUsage(t *testing.T)
```

#### 3.2 State Persistence Performance
```go
func BenchmarkStateSave(b *testing.B)
func BenchmarkStateLoad(b *testing.B)
func TestLargeStateFile(t *testing.T)
```

## Error Handling Strategy

### 1. Tmux Availability
```go
func CheckTmuxAvailable() error {
    if _, err := exec.LookPath("tmux"); err != nil {
        return fmt.Errorf("tmux not found: %w", err)
    }
    return nil
}
```

### 2. Session Management Errors
- Session already exists
- Session not found
- Tmux server not running
- Permission issues
- Invalid session names

### 3. Monitoring Errors
- Process not found
- Output parsing failures
- State file corruption
- Hook execution failures

### 4. Graceful Degradation
```go
type FallbackMode int

const (
    FullFunctionality FallbackMode = iota
    MonitoringDisabled
    NoTmuxMode
)

func DetermineFallbackMode() FallbackMode
func HandleFallback(mode FallbackMode) error
```

## Success Criteria

### 1. Functional Requirements
- ✅ Create tmux sessions with standardized naming
- ✅ List, attach, detach, and kill sessions reliably
- ✅ Monitor Claude Code process states accurately
- ✅ Persist session state across application restarts
- ✅ Handle multiple concurrent sessions

### 2. Performance Requirements
- State detection latency < 500ms
- Support for 10+ concurrent sessions
- Memory usage < 50MB for monitoring
- State file operations < 100ms

### 3. Reliability Requirements
- 99% uptime for monitoring
- Graceful handling of tmux failures
- State consistency across restarts
- Error recovery mechanisms

### 4. Usability Requirements
- Clear error messages
- Intuitive session naming
- Responsive state updates
- Configurable behavior

## Implementation Timeline

### Week 1: Core Session Management
- Day 1-2: Basic tmux command interface
- Day 3-4: Session CRUD operations
- Day 5-7: Session naming and validation

### Week 2: Process Monitoring
- Day 1-3: State detection mechanisms
- Day 4-5: Monitoring lifecycle
- Day 6-7: Hook integration

### Week 3: State Persistence & Testing
- Day 1-2: State file management
- Day 3-4: Unit tests
- Day 5-7: Integration tests and debugging

## Dependencies

### External Dependencies
- tmux binary (runtime dependency)
- go-git library (for branch/worktree info)
- Configuration system (internal/config)

### Internal Dependencies
- Hook system (internal/hooks)
- Configuration management (internal/config)
- Logging system

## Risk Mitigation

### High Risks
1. **Tmux not available**: Implement fallback mode
2. **State detection accuracy**: Multiple detection methods
3. **Performance with many sessions**: Efficient polling and caching

### Medium Risks
1. **Session name conflicts**: Robust naming with collision handling
2. **State file corruption**: Backup and recovery mechanisms
3. **Process monitoring reliability**: Redundant detection methods

## Next Steps

1. Set up basic project structure for internal/tmux/
2. Implement core TmuxCmd interface with mock support
3. Create session management operations
4. Add comprehensive test suite
5. Integrate with existing configuration system
6. Performance optimization and monitoring