# Phase 3.1 Implementation Plan: Main TUI Application

## Overview

Phase 3.1 implements the core Terminal User Interface (TUI) application structure using BubbleTea. This provides the foundation for the interactive user interface that will integrate with all backend systems implemented in Phases 2.1-2.5.

## Current State Analysis

### Completed Backend Systems (Phases 2.1-2.5) ✅
- **Configuration Management** (Phase 2.1): Hierarchical config with live reloading
- **Tmux Integration** (Phase 2.2): Session management and process monitoring  
- **Git Worktree Management** (Phase 2.3): Full worktree lifecycle
- **Claude Code Process Monitoring** (Phase 2.4): Process detection and tracking
- **Status Hooks System** (Phase 2.5): Event-driven hooks for state changes

### Current Gap
- **TUI Directory**: `internal/tui/` exists but is empty
- **Main Integration**: `cmd/ccmgr-ultra/main.go` has TODO for TUI application
- **User Interface**: No interactive interface for backend functionality

## Phase 3.1 Goals

Create the foundational TUI application structure that provides:
1. **Main Application Framework** - BubbleTea application structure
2. **Screen Management System** - Navigation between different screens
3. **Status Bar** - Real-time display of Claude Code states
4. **Keyboard Navigation** - Global and context-specific shortcuts
5. **Integration Layer** - Connect TUI to existing backend systems

## Implementation Steps

### Step 1: Add Dependencies
**File**: `go.mod`
```go
// Add BubbleTea ecosystem dependencies
github.com/charmbracelet/bubbletea v0.25.0
github.com/charmbracelet/lipgloss v0.9.1
github.com/charmbracelet/bubbles v0.17.1
```

### Step 2: Core Application Structure
**File**: `internal/tui/app.go`

Create the main BubbleTea application with:
- Application state management
- Screen stack for navigation
- Global message handling
- Integration with backend managers

**Key Components**:
- `AppModel` struct with current screen and global state
- `ScreenType` enum for different screens
- `GlobalState` struct containing backend manager references
- `Init()`, `Update()`, and `View()` methods

### Step 3: Screen Management System
**File**: `internal/tui/screens.go`

Implement screen management with:
- Screen interface for consistent behavior
- Screen factory for creating screen instances
- Navigation stack for push/pop operations
- Screen transition animations

**Key Interfaces**:
```go
type Screen interface {
    Init() tea.Cmd
    Update(tea.Msg) (Screen, tea.Cmd)
    View() string
    HandleKeyMsg(tea.KeyMsg) (Screen, tea.Cmd)
}
```

### Step 4: Status Bar Implementation
**File**: `internal/tui/components/statusbar.go`

Create status bar component with:
- Real-time Claude Code status display
- Current project and worktree information
- System status indicators
- Keyboard hint display

**Features**:
- Integration with `internal/claude` for process status
- Integration with `internal/tmux` for session info
- Integration with `internal/git` for repository status
- Configurable status update intervals

### Step 5: Keyboard Handling System
**File**: `internal/tui/keys.go`

Implement comprehensive keyboard handling:
- Global shortcuts (quit, help, navigation)
- Context-specific shortcuts per screen
- Configurable key bindings
- Help system integration

**Global Shortcuts**:
- `q` / `Ctrl+C`: Quit application
- `?` / `F1`: Show help
- `Tab` / `Shift+Tab`: Navigate screens
- `Esc`: Go back/cancel

### Step 6: Integration Layer
**File**: `internal/tui/integration.go`

Create integration layer for backend systems:
- Manager initialization and dependency injection
- Real-time updates from backend systems
- Command execution through existing managers
- Error handling and user feedback

**Integration Points**:
- Configuration management for TUI settings
- Tmux session monitoring for status updates
- Git worktree operations for repository management
- Claude Code process monitoring for status display
- Hooks system for event notifications

### Step 7: Main Application Entry Point
**File**: `cmd/ccmgr-ultra/main.go`

Update main function to:
- Initialize TUI application
- Set up backend manager dependencies
- Handle graceful shutdown
- Manage application lifecycle

## Implementation Details

### Application Architecture

```
TUI Application Layer
├── App (BubbleTea Root)
│   ├── Screen Stack Management
│   ├── Global State Management
│   └── Message Routing
├── Screens (Individual UI Screens)
│   ├── Dashboard Screen
│   ├── Worktree Management Screen
│   └── Settings Screen
├── Components (Reusable UI Components)
│   ├── Status Bar
│   ├── Navigation Menu
│   └── Input Forms
└── Integration Layer
    ├── Backend Manager Access
    ├── Real-time Updates
    └── Command Execution
```

### State Management

```go
type AppModel struct {
    screens     []Screen
    currentScreen int
    globalState *GlobalState
    statusBar   *StatusBar
    quitting    bool
}

type GlobalState struct {
    configManager  config.Manager
    sessionManager tmux.Manager
    gitManager     git.Manager
    claudeManager  claude.Manager
    hooksManager   hooks.Manager
}
```

### Screen Navigation Flow

```
Application Start
    ↓
Dashboard Screen (default)
    ├── Navigate to Worktree Management
    ├── Navigate to Session Management  
    ├── Navigate to Configuration
    └── Quit Application
```

## Validation Steps

### Unit Tests
**File**: `internal/tui/app_test.go`
- Test application initialization
- Test screen navigation
- Test message handling
- Test keyboard shortcuts
- Test integration layer

### Integration Tests
**File**: `internal/tui/integration_test.go`
- Test TUI with backend managers
- Test real-time status updates
- Test command execution through TUI
- Test error handling and recovery

### Manual Testing Checklist
- [ ] Application starts without errors
- [ ] Status bar displays current information
- [ ] Keyboard shortcuts work correctly
- [ ] Screen navigation functions properly
- [ ] Integration with backend systems works
- [ ] Error messages display appropriately
- [ ] Application exits gracefully

### Performance Validation
- [ ] UI responsive under normal load
- [ ] Real-time updates don't block interface
- [ ] Memory usage remains stable
- [ ] No resource leaks during operation

## Success Criteria

### Functional Requirements ✅
- [ ] BubbleTea application structure implemented
- [ ] Screen management system working
- [ ] Status bar showing real-time information
- [ ] Keyboard navigation fully functional
- [ ] Integration with all backend systems

### Non-Functional Requirements ✅
- [ ] Responsive UI (< 100ms response time)
- [ ] Clean, intuitive user interface
- [ ] Comprehensive error handling
- [ ] 80%+ test coverage
- [ ] Memory efficient operation

### Integration Requirements ✅
- [ ] Configuration system integration
- [ ] Tmux session status display
- [ ] Git worktree information display
- [ ] Claude Code process status
- [ ] Hooks system event handling

## Risks and Mitigations

### Technical Risks
1. **BubbleTea Learning Curve**
   - *Mitigation*: Start with simple examples, refer to official documentation
   
2. **Real-time Update Performance**
   - *Mitigation*: Implement efficient update intervals, use background goroutines
   
3. **Integration Complexity**
   - *Mitigation*: Use existing manager interfaces, implement gradual integration

### Implementation Risks
1. **Scope Creep**
   - *Mitigation*: Focus on core functionality, defer advanced features
   
2. **Testing Complexity**
   - *Mitigation*: Use BubbleTea testing utilities, mock backend systems

## Next Steps After Phase 3.1

After completing Phase 3.1, the following phases will build upon this foundation:
- **Phase 3.2**: Dashboard Screen Implementation
- **Phase 3.3**: Worktree Management Screen
- **Phase 3.4**: Session Management Screen
- **Phase 3.5**: Configuration Screen

## Files to Create/Modify

### New Files
- `internal/tui/app.go` - Main application structure
- `internal/tui/screens.go` - Screen management system
- `internal/tui/keys.go` - Keyboard handling
- `internal/tui/integration.go` - Backend integration layer
- `internal/tui/components/statusbar.go` - Status bar component
- `internal/tui/app_test.go` - Unit tests
- `internal/tui/integration_test.go` - Integration tests

### Modified Files
- `go.mod` - Add BubbleTea dependencies
- `cmd/ccmgr-ultra/main.go` - Initialize TUI application
- `internal/config/config.go` - Add TUI configuration options

## Estimated Effort

- **Implementation**: 3-4 days
- **Testing**: 1-2 days  
- **Documentation**: 0.5 days
- **Total**: 4.5-6.5 days

This implementation plan provides a solid foundation for the TUI system while maintaining integration with all existing backend functionality from Phases 2.1-2.5.