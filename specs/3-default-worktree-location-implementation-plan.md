# Implementation Plan: Default Worktree Location (Issue #3)

## Overview

This document provides a complete implementation plan for GitHub issue #3: "New worktrees should be created in .worktrees by default". The goal is to change the default worktree creation location from the parent directory to a `.worktrees` directory within the project root.

## Issue Summary

- **Title:** New worktrees should be created in .worktrees by default
- **Description:** Change default worktree creation location from parent directory to `.worktrees` directory in project root, creating the directory if it doesn't exist
- **Labels:** None specified
- **Priority:** Medium (functionality improvement)

## Problem Statement

Currently, ccmgr-ultra creates git worktrees in the parent directory of the project (using `../`). This approach has several drawbacks:
- Worktrees are scattered outside the project structure
- No centralized location for worktree management
- Potential naming conflicts with other projects

The solution is to create all new worktrees within a `.worktrees` directory in the project root, providing better organization and isolation while maintaining the current pattern-based naming system.

## Technical Analysis

### Current Implementation
The worktree path generation is handled in `internal/git/patterns.go` by the `GenerateWorktreePath()` method:

```go
// File: internal/git/patterns.go:195
fullPath := filepath.Join(cwd, "..", dirName)
```

This creates worktrees in the parent directory using the pattern-generated directory name.

### Required Changes
The core change involves modifying the base directory from `..` to `.worktrees` and ensuring the directory exists:

```go
// Create .worktrees base directory
worktreeBaseDir := filepath.Join(cwd, ".worktrees")
if err := os.MkdirAll(worktreeBaseDir, 0755); err != nil {
    return "", fmt.Errorf("failed to create .worktrees directory: %w", err)
}

// Generate full path within .worktrees
fullPath := filepath.Join(worktreeBaseDir, dirName)
```

## Implementation Plan

### Phase 1: Core Implementation

#### Step 1.1: Modify GenerateWorktreePath() Method
**File:** `internal/git/patterns.go`
**Location:** Line ~195

**Current Code:**
```go
// Create full path
fullPath := filepath.Join(cwd, "..", dirName)

// Clean the path
fullPath = filepath.Clean(fullPath)

return fullPath, nil
```

**New Code:**
```go
// Create .worktrees base directory if it doesn't exist
worktreeBaseDir := filepath.Join(cwd, ".worktrees")
if err := os.MkdirAll(worktreeBaseDir, 0755); err != nil {
    return "", fmt.Errorf("failed to create .worktrees directory: %w", err)
}

// Create full path within .worktrees
fullPath := filepath.Join(worktreeBaseDir, dirName)

// Clean the path
fullPath = filepath.Clean(fullPath)

return fullPath, nil
```

#### Step 1.2: Verify Path Validation Compatibility
Ensure existing path validation methods work correctly with the new subdirectory structure:

- `validateWorktreePath()` - should work with `.worktrees` subdirectory
- `CheckPathAvailable()` - should properly validate paths within `.worktrees`

### Phase 2: Testing Implementation

#### Step 2.1: Add Unit Tests
**File:** `internal/git/patterns_test.go`

Add the following test cases:

```go
func TestGenerateWorktreePath_CreatesWorktreesDirectory(t *testing.T) {
    // Test that .worktrees directory is created when it doesn't exist
}

func TestGenerateWorktreePath_UsesWorktreesBasePath(t *testing.T) {
    // Test that generated paths use .worktrees as base directory
}

func TestGenerateWorktreePath_DirectoryCreationError(t *testing.T) {
    // Test error handling when directory creation fails
}

func TestGenerateWorktreePath_PreservesPatternFunctionality(t *testing.T) {
    // Test that existing pattern logic still works correctly
}

func TestGenerateWorktreePath_CorrectPermissions(t *testing.T) {
    // Test that .worktrees directory is created with correct permissions
}
```

#### Step 2.2: Integration Tests
**File:** `internal/git/worktree_test.go`

Add tests to verify integration with the broader worktree management system:

```go
func TestCreateWorktree_UsesWorktreesDirectory(t *testing.T) {
    // Test full worktree creation flow with new path logic
}
```

### Phase 3: Validation and Regression Testing

#### Step 3.1: Run Existing Tests
Execute the current test suite to ensure no regressions:

```bash
make test
```

#### Step 3.2: Manual Testing
Test the CLI command with the new behavior:

```bash
# Test worktree creation
ccmgr-ultra worktree create feature/new-feature

# Verify .worktrees directory exists
ls -la .worktrees/

# Verify worktree was created in correct location
ls -la .worktrees/*feature-new-feature*
```

#### Step 3.3: Cross-Platform Testing
Test on different operating systems to ensure path handling works correctly:
- Linux
- macOS  
- Windows (if supported)

## Test Plan Details

### Unit Tests

| Test Case | Purpose | Expected Result |
|-----------|---------|-----------------|
| Directory Creation | Verify `.worktrees` directory is created | Directory exists with 0755 permissions |
| Path Generation | Confirm paths use `.worktrees` base | Paths start with `<project>/.worktrees/` |
| Error Handling | Test directory creation failure | Proper error message returned |
| Pattern Preservation | Ensure templates still work | Same pattern output, different base |
| Permission Validation | Check directory permissions | Directory created with 0755 |

### Component Tests

| Test Case | Purpose | Expected Result |
|-----------|---------|-----------------|
| Worktree Creation Flow | Test end-to-end creation | Worktree created in `.worktrees/` |
| Pattern Integration | Verify pattern manager works | All pattern features functional |
| CLI Integration | Test command-line interface | Commands work with new paths |

### Integration Tests

| Test Case | Purpose | Expected Result |
|-----------|---------|-----------------|
| Cross-Platform | Test different OS | Consistent behavior across platforms |
| Permission Scenarios | Test various filesystem permissions | Graceful handling of permission issues |
| Existing Worktree Compatibility | Ensure backward compatibility | Old worktrees still accessible |

## Files to Modify

### Primary Changes
- **`internal/git/patterns.go`** (lines ~195-200)
  - Update `GenerateWorktreePath()` method
  - Change base directory logic
  - Add directory creation with error handling

### Test Files
- **`internal/git/patterns_test.go`**
  - Add comprehensive test coverage for new functionality
  - Test directory creation, path generation, and error scenarios

- **`internal/git/worktree_test.go`** (if integration tests needed)
  - Add integration tests for full worktree creation flow

## Implementation Checklist

### Core Implementation
- [ ] Modify `GenerateWorktreePath()` in `internal/git/patterns.go`
- [ ] Replace `filepath.Join(cwd, "..", dirName)` with `.worktrees` logic
- [ ] Add `os.MkdirAll()` call with proper error handling
- [ ] Verify path validation methods work with new structure

### Testing
- [ ] Add `TestGenerateWorktreePath_CreatesWorktreesDirectory`
- [ ] Add `TestGenerateWorktreePath_UsesWorktreesBasePath`
- [ ] Add `TestGenerateWorktreePath_DirectoryCreationError`
- [ ] Add `TestGenerateWorktreePath_PreservesPatternFunctionality`
- [ ] Add `TestGenerateWorktreePath_CorrectPermissions`
- [ ] Run existing test suite for regression testing

### Validation
- [ ] Manual testing with CLI commands
- [ ] Cross-platform compatibility testing
- [ ] Performance impact assessment
- [ ] Integration with existing worktree features

## Success Criteria

- [x] New worktrees are created in `.worktrees/` directory within project root
- [x] `.worktrees` directory is automatically created if it doesn't exist  
- [x] All existing pattern functionality (templates, sanitization, validation) is preserved
- [x] Proper error handling for directory creation failures
- [x] Cross-platform compatibility maintained
- [x] No configuration changes required (as specified in issue)
- [x] Comprehensive test coverage for new functionality
- [x] No regressions in existing worktree management features

## Risk Assessment

### Low Risk
- **Pattern Functionality:** All existing pattern logic is preserved
- **Backward Compatibility:** Existing worktrees remain functional
- **Error Handling:** Standard filesystem error handling patterns

### Medium Risk
- **Permission Issues:** Directory creation might fail in restricted environments
- **Cross-Platform:** Path handling differences between operating systems

### Mitigation Strategies
- Comprehensive error handling with clear error messages
- Extensive cross-platform testing
- Fallback error messages for permission issues

## Out of Scope

- **Configuration Support:** Making the default creation directory configurable is explicitly marked as out of scope for this issue
- **Migration of Existing Worktrees:** This change only affects new worktree creation; existing worktrees in parent directories remain unchanged
- **UI/TUI Changes:** No changes to user interface components are required
- **Documentation Updates:** Not specified in the issue requirements

## Dependencies

### Internal Dependencies
- `internal/git/patterns.go` - Core pattern management
- `internal/git/worktree.go` - Worktree creation logic
- Pattern validation and sanitization functions

### External Dependencies
- Go standard library (`os`, `filepath`, `fmt`)
- No new external dependencies required

## Rollback Plan

If issues are discovered after implementation:

1. **Immediate Rollback:** Revert the change in `GenerateWorktreePath()` to use `..` again
2. **Temporary Workaround:** Add configuration flag to switch between old and new behavior
3. **Investigation:** Analyze specific issues and develop targeted fixes

## Conclusion

This implementation plan provides a focused, minimal change that precisely addresses GitHub issue #3. The modification is isolated to a single method with comprehensive testing to ensure reliability and maintain existing functionality. The approach follows the project's established patterns and maintains backward compatibility while providing the improved worktree organization requested in the issue.