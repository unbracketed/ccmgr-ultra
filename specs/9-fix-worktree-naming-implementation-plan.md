# Implementation Plan: Fix Worktree Template Variable Case Mismatch (Issue #9)

## Overview
Fix worktree creation failure caused by case mismatch between default configuration template variables and pattern validation logic.

## Issue Details
- **GitHub Issue**: #9
- **Title**: create worktree failed
- **Error**: `Template pattern error: failed to create worktree: cause: failed to generate worktree path: failed to apply pattern: invalid pattern: unknown template variable: {{.project}}`
- **Priority**: High (blocks core functionality)

## Root Cause Analysis
The system has an inconsistency between:
- **Default Configuration**: Uses lowercase variables `{{.project}}`, `{{.branch}}` (lines 563, 614 in `internal/config/schema.go`)
- **Pattern Validation**: Only accepts uppercase variables `{{.Project}}`, `{{.Branch}}` (lines 111-114 in `internal/git/patterns.go`)

## Technical Flow
1. User runs `ccmgr-ultra worktree create bra1`
2. `runWorktreeCreateCommand` calls `worktreeManager.CreateWorktree()`
3. `CreateWorktree` calls `wm.patternMgr.GenerateWorktreePath()` (worktree.go:77)
4. `GenerateWorktreePath` calls `pm.ApplyPattern()` with DirectoryPattern from config (patterns.go:183)
5. `ApplyPattern` calls `pm.ValidatePattern()` (patterns.go:58)
6. `ValidatePattern` fails because it only accepts uppercase variables (patterns.go:111-114)

## Solution Approach
Since this is early development and backwards compatibility isn't needed, we'll take the simple approach of fixing the default configurations to use uppercase variables that match the existing validation logic.

## Implementation Tasks

### Task 1: Fix Default Configuration Patterns
**File**: `internal/config/schema.go`

**Changes Required**:
- Line 563: Change `{{.project}}-{{.branch}}` to `{{.Project}}-{{.Branch}}`
- Line 614: Change `{{.project}}-{{.branch}}` to `{{.Project}}-{{.Branch}}`

**Details**:
```go
// WorktreeConfig.SetDefaults() - Line 563
if w.DirectoryPattern == "" {
    w.DirectoryPattern = "{{.Project}}-{{.Branch}}"  // Changed from lowercase
}

// GitConfig.SetDefaults() - Line 614  
if g.DirectoryPattern == "" {
    g.DirectoryPattern = "{{.Project}}-{{.Branch}}"  // Changed from lowercase
}
```

### Task 2: Add Regression Test
**File**: `cmd/ccmgr-ultra/worktree_default_config_test.go` (new file)

**Purpose**: Ensure worktree creation works with default configuration and prevent future regressions.

**Test Content**:
```go
package main

import (
    "os"
    "path/filepath"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/bcdekker/ccmgr-ultra/internal/config"
    "github.com/bcdekker/ccmgr-ultra/internal/git"
)

func TestWorktreeCreateWithDefaultConfig(t *testing.T) {
    // Setup test repository
    testDir := setupTestRepo(t)
    defer os.RemoveAll(testDir)
    
    // Create config with defaults
    cfg := &config.Config{}
    cfg.SetDefaults()
    
    // Test worktree creation
    gitCmd := git.NewGitCmd()
    repoManager := git.NewRepositoryManager(gitCmd)
    repo, err := repoManager.DetectRepository(testDir)
    require.NoError(t, err)
    
    worktreeManager := git.NewWorktreeManager(repo, cfg, gitCmd)
    
    // This should not fail with template variable error
    worktreeInfo, err := worktreeManager.CreateWorktree("test-branch", git.WorktreeOptions{
        CreateBranch: true,
        AutoName:     true,
    })
    
    assert.NoError(t, err)
    assert.NotNil(t, worktreeInfo)
    assert.Contains(t, worktreeInfo.Path, "test-branch")
}

func setupTestRepo(t *testing.T) string {
    testDir, err := os.MkdirTemp("", "ccmgr-test-*")
    require.NoError(t, err)
    
    // Initialize git repo
    gitCmd := git.NewGitCmd()
    _, err = gitCmd.Execute(testDir, "init")
    require.NoError(t, err)
    
    // Create initial commit
    _, err = gitCmd.Execute(testDir, "config", "user.email", "test@example.com")
    require.NoError(t, err)
    _, err = gitCmd.Execute(testDir, "config", "user.name", "Test User")
    require.NoError(t, err)
    
    readmeFile := filepath.Join(testDir, "README.md")
    err = os.WriteFile(readmeFile, []byte("# Test Repo"), 0644)
    require.NoError(t, err)
    
    _, err = gitCmd.Execute(testDir, "add", ".")
    require.NoError(t, err)
    _, err = gitCmd.Execute(testDir, "commit", "-m", "Initial commit")
    require.NoError(t, err)
    
    return testDir
}
```

### Task 3: Update Documentation Comments
**Files**: `internal/config/schema.go`

**Changes Required**:
Update the documentation comments to ensure all examples use uppercase variables consistently:

- Line 59: Update comment examples to use `{{.Project}}` instead of `{{.project}}`
- Line 121: Update comment examples to use `{{.Branch}}` instead of `{{.branch}}`

### Task 4: Verification Test
**File**: `internal/git/patterns_test.go` (add test)

**Purpose**: Verify that the default configuration patterns pass validation.

**Test Content**:
```go
func TestDefaultConfigurationPatternsValid(t *testing.T) {
    cfg := &config.Config{}
    cfg.SetDefaults()
    
    pm := NewPatternManager(&cfg.Worktree)
    
    // Test WorktreeConfig default pattern
    err := pm.ValidatePattern(cfg.Worktree.DirectoryPattern)
    assert.NoError(t, err, "Default WorktreeConfig pattern should be valid")
    
    // Test GitConfig default pattern  
    err = pm.ValidatePattern(cfg.Git.DirectoryPattern)
    assert.NoError(t, err, "Default GitConfig pattern should be valid")
}
```

## Testing Strategy

### Manual Testing
1. **Before Fix**: Run `ccmgr-ultra worktree create test-branch` - should fail with template variable error
2. **After Fix**: Run `ccmgr-ultra worktree create test-branch` - should succeed and create worktree

### Automated Testing
1. **Unit Tests**: Verify pattern validation accepts the new default patterns
2. **Integration Tests**: Test end-to-end worktree creation with default configuration
3. **Regression Tests**: Ensure the specific error from issue #9 doesn't occur

## Files Modified Summary
- `internal/config/schema.go`: Fix default configuration patterns (2 lines)
- `cmd/ccmgr-ultra/worktree_default_config_test.go`: Add regression test (new file)
- `internal/git/patterns_test.go`: Add validation test for default patterns

## Success Criteria
- [ ] `ccmgr-ultra worktree create <branch>` succeeds with default configuration
- [ ] Default configuration patterns use uppercase variables consistently
- [ ] All existing tests continue to pass
- [ ] New regression test prevents future occurrences of this issue
- [ ] Documentation comments use uppercase variable examples consistently

## Risk Assessment
- **Low Risk**: Simple configuration change with no logic modifications
- **No Breaking Changes**: Since this is early development, no backwards compatibility concerns
- **Isolated Impact**: Changes only affect default configuration values

## Rollback Plan
If issues arise, simply revert the configuration changes in `internal/config/schema.go` to restore the previous (broken) behavior while investigating.

## Implementation Order
1. Fix default configuration patterns in `schema.go`
2. Add regression test
3. Update documentation comments
4. Run full test suite to ensure no regressions
5. Manual verification of worktree creation

## Estimated Effort
- **Development Time**: 30 minutes
- **Testing Time**: 15 minutes  
- **Total**: 45 minutes

This is a straightforward configuration fix with minimal risk and high impact on user experience.