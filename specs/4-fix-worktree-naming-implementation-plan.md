# Implementation Plan: Fix Worktree Naming (Issue #4)

## Overview

This document provides the complete implementation plan for fixing the worktree naming issue where Go template variables like `{{.project}}-{{.branch}}` are not being replaced with actual values, resulting in literal directory names.

## Issue Analysis

### Root Cause
The CLI command `worktree create` bypasses the properly designed `PatternManager` architecture and uses a legacy `generateWorktreeDirectory()` function that only supports basic string replacement patterns (`%s`, `{branch}`) instead of Go template syntax (`{{.Project}}`, `{{.Branch}}`).

### Technical Details
- **Affected File**: `cmd/ccmgr-ultra/worktree.go:358`
- **Problem Function**: `generateWorktreeDirectory()` (lines 753-774)
- **Architecture Issue**: CLI directly calls legacy function instead of using `WorktreeManager.CreateWorktree()`
- **Configuration Mismatch**: Default config uses Go template syntax but CLI cannot process it

### Impact
- Users cannot create worktrees with configured template patterns
- Configuration appears broken despite correct syntax
- Inconsistent behavior between CLI and programmatic usage

## Implementation Strategy

### High-Level Approach
1. **Align CLI with Architecture**: Modify CLI to use existing `WorktreeManager.CreateWorktree()` method
2. **Enable Auto-Naming**: Use `AutoName: true` to trigger `PatternManager` usage
3. **Remove Legacy Code**: Eliminate deprecated `generateWorktreeDirectory()` function
4. **Maintain Compatibility**: Ensure existing functionality remains intact

### Architecture Benefits
- Leverages existing, well-tested `PatternManager` system
- Eliminates code duplication and technical debt
- Provides consistent pattern processing across all components
- Maintains security and validation features

## Detailed Implementation Plan

### Phase 1: Core CLI Modification

#### File: `cmd/ccmgr-ultra/worktree.go`

**Current Code (lines 355-362):**
```go
// Determine worktree directory
worktreeDir := worktreeCreateFlags.directory
if worktreeDir == "" {
    worktreeDir, err = generateWorktreeDirectory(cfg, branchName)
    if err != nil {
        return handleCLIError(err)
    }
}
```

**New Code:**
```go
// Determine worktree directory
worktreeDir := worktreeCreateFlags.directory
useAutoName := worktreeDir == ""
```

**Current Code (lines 378-390):**
```go
// Create the worktree
opts := git.WorktreeOptions{
    Path:         worktreeDir,
    Branch:       branchName,
    CreateBranch: true,
    Force:        worktreeCreateFlags.force,
    Checkout:     true,
    TrackRemote:  worktreeCreateFlags.remote,
}
_, err = worktreeManager.CreateWorktree(branchName, opts)
```

**New Code:**
```go
// Create the worktree
opts := git.WorktreeOptions{
    Path:         worktreeDir,
    Branch:       branchName,
    CreateBranch: true,
    Force:        worktreeCreateFlags.force,
    Checkout:     true,
    TrackRemote:  worktreeCreateFlags.remote,
    AutoName:     useAutoName,
}
worktreeInfo, err := worktreeManager.CreateWorktree(branchName, opts)
```

**Update Success Message (lines 433-446):**
```go
if spinner != nil {
    // Use actual path from worktree info
    actualPath := worktreeDir
    if useAutoName && worktreeInfo != nil {
        actualPath = worktreeInfo.Path
    }
    spinner.StopWithMessage(fmt.Sprintf("Worktree '%s' created successfully at %s", branchName, actualPath))
}

if !isQuiet() {
    fmt.Printf("\nWorktree created:\n")
    fmt.Printf("  Branch: %s\n", branchName)
    if useAutoName && worktreeInfo != nil {
        fmt.Printf("  Path: %s\n", worktreeInfo.Path)
    } else {
        fmt.Printf("  Path: %s\n", worktreeDir)
    }
    // ... rest of output
}
```

### Phase 2: Remove Legacy Functions

#### Remove from `cmd/ccmgr-ultra/worktree.go`:

**Functions to Remove:**
- `generateWorktreeDirectory()` (lines 753-774)
- `generateSessionName()` (lines 776-789) - if not used elsewhere
- `getCurrentProjectName()` (lines 791-797) - if not used elsewhere

**Before Removal - Verify Usage:**
```bash
# Search for usage of these functions
grep -r "generateWorktreeDirectory" cmd/
grep -r "generateSessionName" cmd/
grep -r "getCurrentProjectName" cmd/
```

### Phase 3: Configuration Updates

#### Update Default Configuration Comments

**File: `internal/config/schema.go`**

Add documentation comment above `DirectoryPattern`:
```go
// DirectoryPattern defines the template for worktree directory names.
// Supports Go template syntax with variables:
// - {{.Project}}: Project/repository name (sanitized)
// - {{.Branch}}: Git branch name (sanitized)  
// - {{.Worktree}}: Unique worktree identifier
// - {{.Timestamp}}: Current timestamp (YYYYMMDD-HHMMSS)
// - {{.UserName}}: Git user name or system user (sanitized)
// - {{.Prefix}}: Configured prefix value
// - {{.Suffix}}: Configured suffix value
// 
// Template functions available: lower, upper, title, replace, trim, sanitize, truncate
// Example: "{{.Project}}-{{.Branch}}" or "{{.Project | upper}}-{{.Branch | lower}}"
DirectoryPattern string `yaml:"directory_pattern" json:"directory_pattern" default:"{{.Project}}-{{.Branch}}"`
```

### Phase 4: Testing Implementation

#### Unit Tests

**File: `cmd/ccmgr-ultra/worktree_test.go` (create if doesn't exist)**

```go
func TestWorktreeCreate_AutoNaming(t *testing.T) {
    // Test that CLI uses PatternManager for auto-naming
    // Mock WorktreeManager.CreateWorktree to verify AutoName flag
}

func TestWorktreeCreate_CustomDirectory(t *testing.T) {
    // Test that custom directory bypasses auto-naming
    // Verify AutoName is false when directory is specified
}

func TestWorktreeCreate_PatternProcessing(t *testing.T) {
    // Test various template patterns work correctly
    // Verify template variables are replaced properly
}
```

#### Integration Tests

**File: `internal/git/worktree_test.go` (extend existing)**

```go
func TestCreateWorktree_AutoNameWithTemplates(t *testing.T) {
    // Test WorktreeManager with AutoName: true
    // Verify PatternManager.GenerateWorktreePath is called
    // Test various template patterns
}
```

### Phase 5: Error Handling Enhancement

#### Improve Error Messages

**Add to CLI error handling:**
```go
func handlePatternError(err error) error {
    if strings.Contains(err.Error(), "template") {
        return cli.NewErrorWithSuggestion(
            fmt.Sprintf("Template pattern error: %v", err),
            "Check your directory_pattern in config. Use Go template syntax like {{.Project}}-{{.Branch}}",
        )
    }
    return handleCLIError(err)
}
```

## Implementation Checklist

### Pre-Implementation
- [ ] Verify current test coverage for affected functions
- [ ] Backup current implementation 
- [ ] Review dependencies on functions to be removed
- [ ] Confirm PatternManager behavior with various inputs

### Implementation Steps
- [ ] **Step 1**: Modify CLI to use `AutoName` flag
- [ ] **Step 2**: Update worktree creation logic 
- [ ] **Step 3**: Fix success message handling
- [ ] **Step 4**: Remove legacy `generateWorktreeDirectory()` function
- [ ] **Step 5**: Remove unused helper functions
- [ ] **Step 6**: Add configuration documentation
- [ ] **Step 7**: Implement enhanced error handling

### Testing Steps
- [ ] **Unit Tests**: Test CLI with various patterns
- [ ] **Integration Tests**: Test end-to-end worktree creation
- [ ] **Manual Testing**: Verify template patterns work
- [ ] **Regression Testing**: Ensure existing functionality intact
- [ ] **Edge Case Testing**: Test error conditions and validation

### Validation Steps
- [ ] Test with pattern: `{{.Project}}-{{.Branch}}`
- [ ] Test with pattern: `{{.Prefix}}-{{.Project}}-{{.Branch}}-{{.Timestamp}}`
- [ ] Test with functions: `{{.Project | upper}}-{{.Branch | lower}}`
- [ ] Test custom directory override still works
- [ ] Verify no literal template syntax in directory names
- [ ] Confirm tmux session naming remains functional

## Risk Assessment

### Low Risk Changes
- Using existing `WorktreeManager.CreateWorktree()` method
- Enabling `AutoName` flag (already supported)
- Removing unused legacy functions

### Medium Risk Changes  
- Modifying CLI success message logic
- Changing how worktree paths are determined

### Mitigation Strategies
- Comprehensive testing before removing legacy code
- Staged rollout of changes
- Maintain backward compatibility for custom directory specification
- Clear error messages for configuration issues

## Success Criteria

### Functional Requirements
1. **Template Processing**: `{{.Project}}-{{.Branch}}` creates `myproject-feature-auth`
2. **Variable Replacement**: All template variables replaced with actual values
3. **Function Support**: Template functions like `{{.Branch | lower}}` work correctly
4. **Custom Directory**: Manual directory specification still works
5. **Error Handling**: Clear messages for template syntax errors

### Non-Functional Requirements
1. **Performance**: No degradation in worktree creation time
2. **Compatibility**: Existing workflows remain functional
3. **Maintainability**: Reduced code duplication and technical debt
4. **Testability**: Comprehensive test coverage for new behavior

### Acceptance Tests
```bash
# Test 1: Basic template pattern
ccmgr-ultra worktree create feature-auth
# Expected: Creates directory like "myproject-feature-auth"

# Test 2: Complex template pattern  
# Config: directory_pattern: "{{.Prefix}}-{{.Project}}-{{.Branch | lower}}"
ccmgr-ultra worktree create Feature-AUTH
# Expected: Creates directory like "main-myproject-feature-auth"

# Test 3: Custom directory (should bypass templates)
ccmgr-ultra worktree create feature-auth --directory /custom/path
# Expected: Creates directory at "/custom/path"

# Test 4: Invalid template pattern
# Config: directory_pattern: "{{.Invalid}}"  
ccmgr-ultra worktree create feature-auth
# Expected: Clear error message about invalid template variable
```

## Timeline

### Phase 1: Core Implementation (1-2 days)
- Modify CLI command logic
- Update worktree creation flow
- Basic testing

### Phase 2: Cleanup and Enhancement (1 day)
- Remove legacy functions
- Add documentation
- Enhance error handling

### Phase 3: Testing and Validation (1-2 days)
- Comprehensive testing
- Edge case validation
- Documentation updates

**Total Estimated Time: 3-5 days**

## Notes

### Architecture Benefits
- Aligns CLI with designed architecture
- Eliminates duplicate pattern processing logic
- Leverages existing security and validation features
- Maintains consistency across all components

### Backward Compatibility
- Users with custom directory specifications unaffected
- Users with basic patterns (`%s`, `{branch}`) need to update configuration
- Clear migration path to Go template syntax

### Future Enhancements
- Configuration validation to catch template syntax errors early
- Interactive pattern builder for complex templates
- Pattern preview functionality in CLI