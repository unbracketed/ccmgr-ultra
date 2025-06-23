# Phase 3.3 Implementation Plan: Worktree Selection Screen

## Overview
Phase 3.3 focuses on implementing the Worktree Selection Screen with interactive selection options and Claude Code status integration, as outlined in the original implementation steps document. This phase builds upon the comprehensive TUI infrastructure, modal systems, and workflows established in phases 3.1 and 3.2.

## Objectives

Based on the original steps document (section 3.3), we need to implement:
- Display worktrees with status indicators
- Implement interactive selection with options:
  - 'n' - New session
  - 'c' - Continue session  
  - 'r' - Resume session
- Show Claude Code status for each worktree
- Handle worktree operations

## Key Features to Implement

### 1. Enhanced Worktree Selection Screen
- **Interactive worktree list** with visual status indicators
- **Multi-mode selection** with keyboard shortcuts:
  - `n` - New session for selected worktree
  - `c` - Continue existing session
  - `r` - Resume paused session
- **Claude Code status display** for each worktree (idle, busy, waiting)
- **Advanced filtering and sorting** options
- **Bulk operations** for multiple worktrees

### 2. Status Integration
- **Real-time Claude status updates** using existing monitoring system
- **Visual status indicators** (colors, icons, animations)
- **Status-based action availability** (disable actions for busy worktrees)
- **Session state tracking** per worktree

### 3. Interactive Selection Features
- **Multi-select mode** with checkbox-style selection
- **Search and filter** functionality
- **Sort options** (last accessed, branch name, status)
- **Quick actions** via context menus
- **Keyboard navigation** with vim-like shortcuts

### 4. Enhanced Worktree Operations
- **Smart session management** (detect existing sessions)
- **Session restoration** with proper state recovery
- **Directory validation** and cleanup
- **Git status integration** showing changes/sync status

## Implementation Steps

### Step 1: Enhance WorktreesModel in screens.go
**File**: `internal/tui/screens.go`

**Changes needed**:
- Add multi-selection state to WorktreesModel
- Implement selection mode toggle (single vs multi-select)
- Add keyboard handlers for n, c, r actions
- Integrate Claude status display
- Add filtering and sorting capabilities

**New fields for WorktreesModel**:
```go
type WorktreesModel struct {
    integration    *Integration
    theme          Theme
    width          int
    height         int
    cursor         int
    worktrees      []WorktreeInfo
    selectedItems  map[int]bool        // New: multi-selection
    selectionMode  bool                // New: toggle selection mode
    filterText     string              // New: search filter
    sortMode       WorktreeSortMode    // New: sorting
    claudeStatuses map[string]ClaudeStatus // New: status tracking
}
```

**New methods to implement**:
- `toggleSelection()` - Toggle selection mode
- `toggleItemSelection(index int)` - Select/deselect items
- `getSelectedWorktrees()` - Get selected worktrees
- `filterWorktrees(filter string)` - Apply search filter
- `sortWorktrees(mode SortMode)` - Sort worktree list
- `refreshClaudeStatuses()` - Update Claude status info

### Step 2: Extend Worktree Context Menus
**File**: `internal/tui/context/worktree_menu.go`

**Enhancements needed**:
- Add session-specific menu items to existing context menus
- Implement context-aware menu generation based on worktree state
- Add bulk operation menu for multi-selection mode

**New methods to add**:
- `CreateWorktreeSessionMenu(worktree WorktreeInfo, sessions []SessionInfo)` - Session-specific actions
- `CreateWorktreeSelectionMenu(selectedWorktrees []WorktreeInfo)` - Multi-selection menu
- Enhance existing `CreateWorktreeItemMenu()` with session awareness

### Step 3: Session-Worktree Integration
**Files**: `internal/tui/workflows/sessions.go`, `internal/tui/workflows/worktrees.go`

**New workflows to implement**:
- **Continue Session Workflow**: Find and attach to existing sessions for a worktree
- **Resume Session Workflow**: Restore paused/detached sessions
- **New Session from Worktree Workflow**: Create new session in selected worktree

**Integration points**:
- Extend existing session creation wizard to accept worktree source
- Add session detection and validation logic
- Implement session state restoration

### Step 4: Status Display System
**Files**: `internal/tui/screens.go`, `internal/tui/components/statusbar.go`

**Status display features**:
- Real-time Claude status updates in worktree list
- Visual status indicators (colors, icons, progress)
- Status-based action filtering (disable actions for busy worktrees)
- Session count and state display per worktree

**Status types to display**:
- Claude Code process status (idle, busy, waiting, error)
- Active session count
- Git status (clean, changes, conflicts)
- Last access time

### Step 5: Enhanced Keyboard Navigation
**File**: `internal/tui/screens.go` (WorktreesModel.Update method)

**New keyboard shortcuts**:
- `n` - New session for current/selected worktrees
- `c` - Continue session for current/selected worktrees  
- `r` - Resume session for current/selected worktrees
- `space` - Toggle selection of current item
- `a` - Select all / deselect all
- `/` - Enter search/filter mode
- `s` - Cycle through sort modes
- `tab` - Toggle selection mode

### Step 6: Testing and Polish
**Files**: New test files and existing test enhancements

**Testing requirements**:
- Unit tests for new WorktreesModel methods
- Integration tests for session-worktree workflows
- UI tests for keyboard navigation and selection
- Performance tests for large worktree lists

## Technical Implementation Details

### Enhanced WorktreesModel Structure

```go
type WorktreeInfo struct {
    Path           string
    Branch         string
    LastAccess     time.Time
    HasChanges     bool
    ActiveSessions []SessionSummary    // New
    ClaudeStatus   ClaudeStatus       // New
    GitStatus      GitWorktreeStatus  // New
}

type WorktreeSortMode int
const (
    SortByName WorktreeSortMode = iota
    SortByLastAccess
    SortByBranch
    SortByStatus
)

type ClaudeStatus struct {
    State       string    // idle, busy, waiting, error
    ProcessID   int
    LastUpdate  time.Time
    SessionID   string
}
```

### Keyboard Handler Enhancement

```go
func (m *WorktreesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "n":
            return m, m.createNewSessionForSelection()
        case "c":
            return m, m.continueSessionForSelection()
        case "r":
            return m, m.resumeSessionForSelection()
        case " ":
            m.toggleItemSelection(m.cursor)
        case "a":
            m.toggleSelectAll()
        case "/":
            return m, m.enterSearchMode()
        case "s":
            m.cycleSortMode()
        case "tab":
            m.toggleSelectionMode()
        // ... existing handlers
        }
    }
    return m, nil
}
```

### Status Integration

The existing Integration layer will be enhanced to provide:
- Real-time Claude process monitoring per worktree
- Session enumeration and state tracking
- Git status information for each worktree

## Validation Steps

### Step 1: Unit Testing
- **Test multi-selection functionality**: Verify selection state management
- **Test keyboard navigation**: Ensure all new shortcuts work correctly
- **Test filtering and sorting**: Validate search and sort operations
- **Test status integration**: Mock Claude status updates and verify display

### Step 2: Integration Testing  
- **Test session workflows**: Verify new/continue/resume session flows
- **Test context menu integration**: Ensure menus appear and function correctly
- **Test real-time updates**: Verify status updates reflect in UI immediately
- **Test large datasets**: Performance with 50+ worktrees

### Step 3: User Experience Testing
- **Navigation efficiency**: Time common workflows (should be < 3 keystrokes)
- **Visual clarity**: Status indicators should be immediately recognizable  
- **Error handling**: Graceful handling of missing sessions, invalid worktrees
- **Responsive feedback**: UI updates within 100ms of user action

### Step 4: Compatibility Testing
- **Existing functionality**: Ensure no regressions in current worktree operations
- **Theme consistency**: New elements match existing TUI theme
- **Modal integration**: New workflows work with existing modal system
- **Backend compatibility**: Integration layer handles new requirements

### Step 5: Performance Validation
- **Memory usage**: No memory leaks with selection state
- **Rendering performance**: Smooth scrolling with large lists
- **Status update efficiency**: Minimal CPU usage for real-time updates
- **Search performance**: Sub-100ms search response time

## Success Criteria

### Functional Requirements
- ✅ Worktree list displays with status indicators
- ✅ Interactive selection with n/c/r shortcuts working
- ✅ Claude Code status shown for each worktree
- ✅ Multi-select mode with bulk operations
- ✅ Search and filtering functionality
- ✅ Sort options (name, date, status)

### Performance Requirements  
- ✅ < 100ms UI response time for all interactions
- ✅ Handles 100+ worktrees without performance degradation
- ✅ Real-time status updates without blocking UI
- ✅ Memory efficient selection state management

### User Experience Requirements
- ✅ Intuitive keyboard navigation matching vim patterns
- ✅ Clear visual feedback for all actions
- ✅ Consistent with existing TUI design patterns
- ✅ Comprehensive error handling and recovery
- ✅ Context-sensitive help and shortcuts

## Dependencies

### Internal Dependencies
- Phase 3.1: Main TUI application framework
- Phase 3.2: Modal and workflow systems
- Existing Integration layer for backend communication
- Claude monitoring system from Phase 2.4
- Git worktree management from Phase 2.3

### External Dependencies
- BubbleTea framework for TUI updates
- Lipgloss for enhanced styling
- Existing tmux integration for session management

## Risk Mitigation

### Technical Risks
- **Complex state management**: Use existing modal patterns for consistency
- **Performance with large datasets**: Implement virtual scrolling if needed
- **Real-time update conflicts**: Use proper event sequencing and debouncing

### User Experience Risks  
- **Learning curve**: Provide contextual help and progressive disclosure
- **Feature overload**: Maintain simple default view with advanced features hidden
- **Existing workflow disruption**: Ensure backward compatibility

## Timeline Estimate

- **Step 1-2** (WorktreesModel + Context Menus): 2-3 days
- **Step 3** (Session Integration): 2-3 days  
- **Step 4** (Status Display): 1-2 days
- **Step 5** (Navigation Enhancement): 1 day
- **Step 6** (Testing and Polish): 2-3 days

**Total Estimated Time**: 8-12 days

## Future Enhancements

While not part of Phase 3.3, the following could be added later:
- **Saved selection sets**: Remember commonly used worktree groups
- **Custom status indicators**: User-defined status types and colors
- **Worktree templates**: Quick creation from predefined templates
- **Integration with external tools**: VS Code, IDE project opening
- **Advanced filtering**: Date ranges, regex patterns, custom criteria

## Conclusion

Phase 3.3 will complete the core TUI functionality by providing a sophisticated worktree selection interface that matches the original CCManager aesthetic while leveraging the advanced infrastructure built in previous phases. The implementation focuses on user efficiency and real-time status awareness, making worktree and session management intuitive and powerful.