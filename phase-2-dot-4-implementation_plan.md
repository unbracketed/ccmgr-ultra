# Phase 2.4 Implementation Plan: Claude Code Process Monitoring

## Overview
This document provides a comprehensive implementation plan for Phase 2.4 of ccmgr-ultra, focusing on Claude Code process monitoring capabilities. This phase establishes the foundation for detecting and tracking Claude Code process states (busy, idle, waiting) across multiple instances and sessions.

## Phase 2.4 Requirements Analysis

From `steps-to-implement.md`, Phase 2.4 must implement:
- [ ] Detect Claude Code process state (busy, idle, waiting)
- [ ] Implement state change detection
- [ ] Create process tracking mechanism
- [ ] Handle multiple Claude Code instances

## Implementation Strategy

### 2.4.1 Process Detection and Identification

**Files to create/modify:**
- `internal/claude/process.go` - Core process management
- `internal/claude/detector.go` - Process detection logic
- `internal/claude/types.go` - Type definitions

**Implementation Steps:**

1. **Process Discovery**
   ```go
   type ProcessInfo struct {
       PID         int
       SessionID   string
       WorkingDir  string
       Command     []string
       StartTime   time.Time
       State       ProcessState
   }
   ```

2. **Detection Methods**
   - Use `ps` command to find Claude Code processes
   - Parse process arguments to identify working directories
   - Map processes to tmux sessions when available
   - Handle both direct and tmux-wrapped Claude Code instances

3. **Process Identification Logic**
   - Scan for processes matching "claude" or specific Claude Code patterns
   - Extract working directory from process environment
   - Associate with git worktrees when possible

### 2.4.2 State Detection Mechanism

**Core States to Monitor:**
- `Idle` - Claude Code is running but not actively processing
- `Busy` - Currently executing commands or processing requests
- `Waiting` - Waiting for user input or confirmation
- `Starting` - Process is initializing
- `Error` - Process encountered an error state

**Implementation Approach:**

1. **Log File Monitoring**
   ```go
   type LogMonitor struct {
       logPath     string
       lastOffset  int64
       stateRegex  map[ProcessState]*regexp.Regexp
   }
   ```

2. **Process Resource Monitoring**
   - Monitor CPU usage patterns
   - Track file descriptor activity
   - Monitor network connections for API calls

3. **Output Stream Analysis**
   - Parse stdout/stderr when accessible
   - Look for specific Claude Code state indicators
   - Handle different output formats and versions

### 2.4.3 State Change Detection

**Event System:**
```go
type StateChangeEvent struct {
    ProcessID   string
    OldState    ProcessState
    NewState    ProcessState
    Timestamp   time.Time
    SessionID   string
    WorktreeID  string
}

type StateChangeHandler interface {
    OnStateChange(event StateChangeEvent) error
}
```

**Detection Methods:**

1. **Polling Strategy**
   - Configurable polling intervals (default: 2-5 seconds)
   - Efficient diff-based state comparison
   - Batch processing for multiple instances

2. **Event-Driven Detection**
   - File system watchers for log files
   - Process signal monitoring
   - Integration with tmux capture-pane for real-time output

3. **Hybrid Approach**
   - Fast polling for active processes
   - Slower polling for idle processes
   - Event-driven updates when available

### 2.4.4 Multi-Instance Process Tracking

**Process Registry:**
```go
type ProcessTracker struct {
    processes   map[string]*ProcessInfo
    subscribers []StateChangeHandler
    config      *ProcessConfig
    mutex       sync.RWMutex
    stopCh      chan struct{}
}
```

**Key Features:**

1. **Instance Management**
   - Unique identification per Claude Code instance
   - Process lifecycle tracking (start/stop/crash)
   - Orphaned process cleanup

2. **Session Association**
   - Link processes to tmux sessions
   - Associate with git worktrees
   - Track multiple sessions per worktree

3. **Resource Management**
   - Efficient polling with backoff strategies
   - Memory-conscious process tracking
   - Graceful handling of process termination

## Implementation Files Structure

```
internal/claude/
├── process.go          # Main process management logic
├── detector.go         # Process detection and discovery
├── monitor.go          # State monitoring and polling
├── tracker.go          # Multi-instance tracking
├── types.go           # Type definitions and interfaces
├── log_parser.go      # Log file parsing utilities
├── state_machine.go   # State transition logic
└── config.go          # Configuration for monitoring
```

## Detailed Implementation Steps

### Step 1: Core Types and Interfaces (1-2 days)

**File: `internal/claude/types.go`**
```go
type ProcessState int

const (
    StateUnknown ProcessState = iota
    StateStarting
    StateIdle
    StateBusy
    StateWaiting
    StateError
    StateStopped
)

type ProcessConfig struct {
    PollInterval     time.Duration
    LogPaths         []string
    StatePatterns    map[ProcessState]string
    MaxProcesses     int
    CleanupInterval  time.Duration
}
```

### Step 2: Process Detection (2-3 days)

**File: `internal/claude/detector.go`**
- Implement cross-platform process discovery
- Handle different Claude Code installation methods
- Extract working directory and session information
- Validate process accessibility and permissions

### Step 3: State Monitoring (3-4 days)

**File: `internal/claude/monitor.go`**
- Implement log file parsing for state detection
- Create resource-based state inference
- Handle different Claude Code versions and output formats
- Implement efficient polling with smart intervals

### Step 4: Process Tracking (2-3 days)

**File: `internal/claude/tracker.go`**
- Build multi-instance registry
- Implement state change event system
- Create process lifecycle management
- Add subscription mechanism for state updates

### Step 5: Integration Layer (1-2 days)

**File: `internal/claude/process.go`**
- Create unified API for process management
- Implement configuration loading
- Add error handling and logging
- Create shutdown procedures

## Testing Strategy

### Unit Tests
- Process detection accuracy
- State transition logic
- Event system functionality
- Configuration parsing

### Integration Tests
- Multi-instance tracking
- Tmux session integration
- Git worktree association
- Resource cleanup

### End-to-End Tests
- Full process lifecycle
- State change propagation
- Error recovery scenarios
- Performance under load

## Configuration Integration

**Example configuration (`~/.ccmgr-ultra/config.yaml`):**
```yaml
claude:
  monitoring:
    poll_interval: "3s"
    max_processes: 10
    cleanup_interval: "5m"
    log_paths:
      - "~/.claude/logs"
      - "/tmp/claude-*"
    state_patterns:
      busy: "Processing|Executing|Running"
      idle: "Waiting for input|Ready"
      error: "Error|Failed|Exception"
```

## Error Handling and Edge Cases

1. **Process Permission Issues**
   - Graceful degradation when process details unavailable
   - Clear error messages for permission problems
   - Fallback to basic process detection

2. **Resource Constraints**
   - Memory limits for process tracking
   - CPU usage throttling during heavy monitoring
   - Configurable limits and timeouts

3. **Platform Differences**
   - macOS vs Linux process detection differences
   - Different tmux behaviors across platforms
   - Path resolution variations

## Performance Considerations

1. **Efficient Polling**
   - Adaptive polling intervals based on activity
   - Batch processing for multiple processes
   - Early termination for unchanged states

2. **Memory Management**
   - Bounded process history
   - Efficient data structures
   - Regular cleanup of stale data

3. **CPU Usage**
   - Configurable monitoring intensity
   - Background processing with low priority
   - Intelligent caching of process information

## Integration Points

### With Tmux Integration (Phase 2.2)
- Associate processes with tmux sessions
- Use tmux capture-pane for state detection
- Coordinate session lifecycle with process monitoring

### With Status Hooks (Phase 2.5)
- Trigger hooks on state changes
- Provide process context to hook scripts
- Handle hook execution errors gracefully

### With TUI (Phase 3)
- Real-time status updates in interface
- Process state indicators
- Interactive process management

## Validation Criteria

### Functional Requirements
- [ ] Accurately detect Claude Code processes across all supported platforms
- [ ] Correctly identify process states with <5% false positives
- [ ] Handle at least 10 concurrent Claude Code instances
- [ ] State changes detected within 5 seconds of occurrence
- [ ] Process tracking survives system sleep/wake cycles

### Performance Requirements
- [ ] CPU usage <2% during normal monitoring
- [ ] Memory usage <50MB for tracking 10 processes
- [ ] Startup time <1 second for monitoring initialization
- [ ] State change propagation <500ms

### Reliability Requirements
- [ ] Graceful handling of process crashes
- [ ] Recovery from monitoring service interruption
- [ ] No memory leaks during extended operation
- [ ] Proper cleanup of resources on shutdown

## Timeline Estimate

- **Days 1-2**: Core types and interfaces
- **Days 3-5**: Process detection implementation
- **Days 6-9**: State monitoring and parsing
- **Days 10-12**: Multi-instance tracking
- **Days 13-14**: Integration and polish
- **Days 15-16**: Testing and validation

**Total Estimated Duration: 16 days (3.2 weeks)**

## Dependencies

### Internal Dependencies
- Configuration system (Phase 2.1) - Must be completed first
- Tmux integration (Phase 2.2) - Parallel development, integration needed

### External Dependencies
- Go standard library (os, exec, time, regexp)
- Third-party process utilities (if needed)
- Platform-specific process APIs

## Risk Assessment

### High Risk
- Cross-platform process detection differences
- Claude Code output format changes
- Permission issues in corporate environments

### Medium Risk
- Performance impact of continuous monitoring
- State detection accuracy across different usage patterns
- Integration complexity with existing systems

### Low Risk
- Configuration parsing and validation
- Event system implementation
- Basic process lifecycle management

## Success Metrics

1. **Accuracy**: >95% correct state detection
2. **Performance**: <3% system resource usage
3. **Reliability**: <1 failure per 1000 state changes
4. **Coverage**: Support for all major Claude Code usage patterns
5. **Integration**: Seamless operation with tmux and git workflows

This implementation plan provides a solid foundation for Phase 2.4, ensuring robust Claude Code process monitoring that integrates seamlessly with the overall ccmgr-ultra architecture.