# Phase 3.4 Implementation Plan: Configuration Screens

## Overview
Phase 3.4 implements comprehensive configuration screens for the ccmgr-ultra TUI application. This phase builds upon the completed foundations from Phases 3.1-3.3.

## Prerequisites Completed
- ✅ Phase 3.1: Main TUI application with BubbleTea framework
- ✅ Phase 3.2: Enhanced TUI workflows and interactive features
- ✅ Phase 3.3: Worktree Selection Screen with multi-select, filtering, and Claude status

## Current State Analysis

### Existing Infrastructure
1. **Configuration System** (`internal/config/`)
   - Well-defined schema in `schema.go`
   - Load/Save functionality in `config.go`
   - Validation methods for all config sections

2. **TUI Framework** (`internal/tui/`)
   - Screen interface established
   - Theme system implemented
   - Navigation patterns defined
   - Basic ConfigModel exists but needs expansion

3. **Integration Points**
   - Config package ready for use
   - TUI patterns established from WorktreesModel
   - BubbleTea components available

## Implementation Steps

### Step 1: Create Configuration Infrastructure

#### 1.1 Create `internal/tui/config.go`
```go
// Main configuration menu model
type ConfigMenuModel struct {
    config       *config.Config
    theme        Theme
    width        int
    height       int
    cursor       int
    menuItems    []ConfigMenuItem
    currentScreen ConfigScreen
    unsavedChanges bool
}

// Individual configuration screens
type ConfigScreen interface {
    Screen
    HasUnsavedChanges() bool
    Save() error
    Cancel()
    Reset()
}
```

#### 1.2 Define Menu Structure
```go
type ConfigMenuItem struct {
    Title       string
    Description string
    Icon        string
    Screen      ConfigScreen
}
```

### Step 2: Implement Main Configuration Menu

#### 2.1 ConfigMenuModel Implementation
- Display categorized configuration options
- Handle navigation between categories
- Track unsaved changes across all screens
- Implement breadcrumb navigation

#### 2.2 Menu Items
1. 🔔 Status Hooks - Configure Claude state change notifications
2. 🌳 Worktree Hooks - Lifecycle event scripts
3. ⌨️  Shortcuts - Keyboard shortcut customization
4. 📁 Worktree Settings - Directory patterns and defaults
5. 🖥️  Commands - External command configuration
6. 🎨 TUI Settings - Interface preferences
7. 🔧 Git Settings - Repository and branch configuration
8. 🪟 Tmux Settings - Session management options
9. 🤖 Claude Settings - Process monitoring configuration

### Step 3: Implement Individual Configuration Screens

#### 3.1 Status Hooks Configuration (`StatusHooksConfigModel`)
```go
type StatusHooksConfigModel struct {
    config    *config.StatusHooksConfig
    original  *config.StatusHooksConfig
    theme     Theme
    cursor    int
    editing   bool
    editField string
}
```

**Features:**
- Master enable/disable toggle
- Individual hook configuration:
  - Idle Hook (script path, timeout, async)
  - Busy Hook (script path, timeout, async)
  - Waiting Hook (script path, timeout, async)
- Script path validation
- Timeout input with bounds (1-300 seconds)
- Test hook execution

#### 3.2 Worktree Hooks Configuration (`WorktreeHooksConfigModel`)
**Features:**
- Master enable/disable toggle
- Creation hook configuration
- Activation hook configuration
- Script testing capability

#### 3.3 Shortcuts Configuration (`ShortcutsConfigModel`)
```go
type ShortcutsConfigModel struct {
    shortcuts  map[string]string
    original   map[string]string
    theme      Theme
    cursor     int
    adding     bool
    editing    bool
    newKey     string
    newAction  string
}
```

**Features:**
- List current shortcuts
- Add new shortcuts
- Edit existing shortcuts
- Delete shortcuts
- Validate key conflicts
- Action preview/description

#### 3.4 Worktree Settings (`WorktreeSettingsModel`)
**Features:**
- Auto-directory toggle
- Directory pattern editor
  - Template variable help
  - Live preview with example
- Default branch setting
- Cleanup on merge toggle

#### 3.5 Commands Configuration (`CommandsConfigModel`)
**Features:**
- Claude command path input
- Git command path input
- Tmux prefix configuration
- Environment variables editor
  - Add/edit/delete variables
  - Validate variable names

#### 3.6 TUI Settings (`TUISettingsModel`)
**Features:**
- Theme selector (with preview)
- Refresh interval slider (1-60 seconds)
- Mouse support toggle
- Default screen dropdown
- Status bar visibility
- Key help visibility
- Confirm quit toggle
- Auto-refresh toggle
- Debug mode toggle

#### 3.7 Git Settings (`GitSettingsModel`)
**Features:**
- Worktree configuration
  - Max worktrees limit
  - Cleanup age
- Branch settings
  - Default branch
  - Protected branches list
  - Force delete permission
- Remote settings
  - Default remote
  - Auto-push toggle
  - PR creation toggle
  - PR template editor
- Authentication tokens (masked input)

#### 3.8 Tmux Settings (`TmuxSettingsModel`)
**Features:**
- Session prefix
- Naming pattern editor
- Max session name length
- Monitor interval
- State file location
- Default environment variables
- Auto-cleanup toggle
- Cleanup age

#### 3.9 Claude Settings (`ClaudeSettingsModel`)
**Features:**
- Enable/disable monitoring
- Poll interval
- Max processes limit
- Cleanup interval
- State timeout
- Startup timeout
- Log paths configuration
- State patterns editor
- Integration toggles

### Step 4: Common UI Components

#### 4.1 Input Components
```go
// Text input with validation
type ConfigTextInput struct {
    label       string
    value       string
    placeholder string
    validator   func(string) error
    multiline   bool
}

// Toggle switch
type ConfigToggle struct {
    label   string
    value   bool
    enabled bool
}

// Number input with bounds
type ConfigNumberInput struct {
    label string
    value int
    min   int
    max   int
    step  int
}
```

#### 4.2 Layout Components
- Section headers
- Help text
- Error messages
- Validation indicators
- Save/Cancel buttons

### Step 5: State Management

#### 5.1 Change Tracking
```go
type ChangeTracker struct {
    original interface{}
    current  interface{}
    fields   map[string]bool // track which fields changed
}
```

#### 5.2 Validation System
- Real-time validation as user types
- Error message display
- Prevent saving invalid configuration

#### 5.3 Persistence
- Save to appropriate config files
- Handle merge of global/project configs
- Backup before save
- Atomic writes

### Step 6: Integration Tasks

#### 6.1 Navigation Integration
- Add configuration to main app screens
- Update help text
- Add keyboard shortcuts

#### 6.2 Hot Reload
- Watch config file for external changes
- Prompt user to reload

#### 6.3 Import/Export
- Export configuration to file
- Import from file with validation
- Share configuration templates

## Validation Steps

### 1. Unit Testing
- [ ] Test each configuration model independently
- [ ] Validate state management
- [ ] Test save/load functionality
- [ ] Verify validation rules

### 2. Integration Testing
- [ ] Test navigation between screens
- [ ] Verify configuration persistence
- [ ] Test configuration reload
- [ ] Validate error handling

### 3. User Experience Testing
- [ ] Keyboard navigation smooth
- [ ] Visual feedback clear
- [ ] Error messages helpful
- [ ] Changes tracked properly

### 4. Configuration Validation
- [ ] All fields validate correctly
- [ ] Defaults apply properly
- [ ] Merge logic works correctly
- [ ] File permissions handled

### 5. Edge Cases
- [ ] Handle missing config files
- [ ] Handle corrupted config
- [ ] Handle permission errors
- [ ] Handle concurrent edits

## Implementation Order

### Phase 1: Foundation (Day 1)
1. Create `internal/tui/config.go`
2. Implement `ConfigMenuModel`
3. Create base `ConfigScreen` interface
4. Implement navigation framework

### Phase 2: Core Screens (Days 2-3)
1. Status Hooks Configuration
2. Shortcuts Configuration
3. Worktree Settings
4. Commands Configuration

### Phase 3: Advanced Screens (Days 3-4)
1. TUI Settings
2. Git Settings
3. Tmux Settings
4. Claude Settings
5. Worktree Hooks Configuration

### Phase 4: Polish & Testing (Day 5)
1. Add validation throughout
2. Implement hot reload
3. Add import/export
4. Complete testing
5. Update documentation

## Success Criteria

### Functionality
- [ ] All configuration options accessible via TUI
- [ ] Changes persist correctly
- [ ] Validation prevents invalid configurations
- [ ] Unsaved changes tracked and warned

### User Experience
- [ ] Intuitive navigation
- [ ] Clear visual feedback
- [ ] Helpful error messages
- [ ] Responsive performance

### Code Quality
- [ ] Follows established patterns
- [ ] Well-documented
- [ ] Comprehensive tests
- [ ] Clean separation of concerns

## Dependencies

### External Packages
- `github.com/charmbracelet/bubbles/textinput`
- `github.com/charmbracelet/bubbles/textarea`
- `github.com/charmbracelet/bubbles/list`

### Internal Dependencies
- `internal/config` - Configuration schema and persistence
- `internal/tui` - Existing TUI framework
- `internal/services` - For testing hooks

## Risks & Mitigations

### Risk 1: Complex State Management
**Mitigation:** Use clear state tracking pattern, implement undo/redo

### Risk 2: Validation Complexity
**Mitigation:** Centralize validation logic, provide clear error messages

### Risk 3: Performance with Large Configs
**Mitigation:** Lazy load screens, optimize rendering

## Notes

### Design Principles
1. **Consistency**: Follow patterns from WorktreesModel
2. **Feedback**: Always show user what's happening
3. **Safety**: Prevent accidental data loss
4. **Discoverability**: Make options easy to find

### Future Enhancements
1. Configuration profiles
2. Cloud sync
3. Configuration templates
4. Bulk operations
5. Search within configuration

## Completion Checklist

- [ ] All screens implemented
- [ ] Navigation working smoothly
- [ ] Validation comprehensive
- [ ] Changes tracked properly
- [ ] Save/Cancel/Reset functional
- [ ] Tests passing
- [ ] Documentation updated
- [ ] Code reviewed
- [ ] Integration tested
- [ ] Performance acceptable