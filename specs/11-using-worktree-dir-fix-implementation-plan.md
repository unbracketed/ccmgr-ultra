# Implementation Plan: Fix Worktree Directory Issue #11

## Overview

This document provides a complete implementation plan for resolving GitHub issue #11, where ccmgr-ultra fails to create worktrees in `.worktrees` directory due to git's constraint that worktree paths cannot be inside the repository.

## Problem Analysis

### Root Cause
The current implementation in `internal/git/patterns.go:194-206` creates worktree paths within the repository:
```go
worktreeBaseDir := filepath.Join(cwd, ".worktrees")
```

This violates git's constraint enforced in `internal/git/worktree.go:320-335`:
```go
if strings.HasPrefix(absPath, repoPath) {
    return fmt.Errorf("worktree path cannot be inside repository: %s", path)
}
```

### Current Behavior
- User runs: `ccmgr-ultra worktree create feature/new-test`
- Generated path: `/repo/.worktrees/project-feature-new-test-timestamp`
- Git validation fails: "worktree path cannot be inside repository"

### Target Behavior
- User runs: `ccmgr-ultra worktree create feature/new-test`
- Generated path: `/parent/.worktrees/repo-name/project-feature-new-test-timestamp`
- Git validation succeeds: worktree created outside repository

## Implementation Strategy

### Approach: Sibling Directory Pattern
Move `.worktrees` outside the repository as a sibling directory, organized by repository name for multi-repo support.

**Directory Structure:**
```
parent-directory/
├── my-repo/                    # Original repository
│   ├── .git/
│   └── source-files...
└── .worktrees/                 # Worktrees base directory
    └── my-repo/                # Per-repository worktrees
        ├── feature-auth-123/
        ├── bugfix-issue-456/
        └── main-backup/
```

## Detailed Implementation Plan

### Phase 1: Core Configuration Changes

#### 1.1 Update WorktreeConfig Schema
**File:** `internal/config/schema.go`

**Changes:**
```go
// WorktreeConfig defines worktree configuration
type WorktreeConfig struct {
    AutoDirectory    bool   `yaml:"auto_directory" json:"auto_directory"`
    DirectoryPattern string `yaml:"directory_pattern" json:"directory_pattern"`
    DefaultBranch    string `yaml:"default_branch" json:"default_branch"`
    CleanupOnMerge   bool   `yaml:"cleanup_on_merge" json:"cleanup_on_merge"`
    
    // NEW: Base directory for worktrees (relative to repository parent or absolute)
    BaseDirectory    string `yaml:"base_directory" json:"base_directory"`
}
```

**Update SetDefaults method:**
```go
func (w *WorktreeConfig) SetDefaults() {
    if w.DirectoryPattern == "" {
        w.DirectoryPattern = "{{.Branch}}"  // Simplified since repo name is in path
    }
    if w.DefaultBranch == "" {
        w.DefaultBranch = "main"
    }
    if w.BaseDirectory == "" {
        w.BaseDirectory = "../.worktrees/{{.Project}}"  // Default sibling pattern
    }
}
```

#### 1.2 Update Pattern Generation Logic
**File:** `internal/git/patterns.go`

**Modify GenerateWorktreePath function:**
```go
// GenerateWorktreePath generates a full worktree path based on configuration
func (pm *PatternManager) GenerateWorktreePath(branch, project string) (string, error) {
    context := PatternContext{
        Project:   pm.sanitizeComponent(project),
        Branch:    pm.sanitizeComponent(branch),
        Worktree:  pm.generateWorktreeID(branch),
        Timestamp: time.Now().Format("20060102-150405"),
        UserName:  pm.getUserName(),
        Prefix:    pm.config.DefaultBranch,
        Suffix:    "",
    }

    // Resolve base directory pattern first
    baseDir, err := pm.ResolvePatternVariables(pm.config.BaseDirectory, context)
    if err != nil {
        return "", fmt.Errorf("failed to resolve base directory pattern: %w", err)
    }

    // Apply the naming pattern for the worktree directory
    dirName, err := pm.ApplyPattern(pm.config.DirectoryPattern, context)
    if err != nil {
        return "", fmt.Errorf("failed to apply pattern: %w", err)
    }

    // Get current working directory as reference
    cwd, err := os.Getwd()
    if err != nil {
        return "", fmt.Errorf("failed to get current directory: %w", err)
    }

    // Resolve base directory (can be relative or absolute)
    var fullBaseDir string
    if filepath.IsAbs(baseDir) {
        fullBaseDir = baseDir
    } else {
        fullBaseDir = filepath.Join(cwd, baseDir)
    }

    // Create base directory if it doesn't exist
    if err := os.MkdirAll(fullBaseDir, 0755); err != nil {
        return "", fmt.Errorf("failed to create base directory: %w", err)
    }

    // Create full path
    fullPath := filepath.Join(fullBaseDir, dirName)
    
    // Clean the path
    fullPath = filepath.Clean(fullPath)

    return fullPath, nil
}
```

#### 1.3 Add Base Directory Validation
**File:** `internal/git/patterns.go`

**Add new validation function:**
```go
// ValidateBaseDirectory validates the base directory configuration
func (pm *PatternManager) ValidateBaseDirectory(baseDir string, repoPath string) error {
    if baseDir == "" {
        return fmt.Errorf("base directory cannot be empty")
    }

    // Resolve relative paths
    var absBaseDir string
    if filepath.IsAbs(baseDir) {
        absBaseDir = baseDir
    } else {
        cwd, err := os.Getwd()
        if err != nil {
            return fmt.Errorf("failed to get current directory: %w", err)
        }
        absBaseDir = filepath.Join(cwd, baseDir)
    }
    
    absBaseDir = filepath.Clean(absBaseDir)
    
    // Get absolute repository path
    absRepoPath, err := filepath.Abs(repoPath)
    if err != nil {
        return fmt.Errorf("failed to get absolute repository path: %w", err)
    }

    // Check if base directory is inside repository
    if strings.HasPrefix(absBaseDir, absRepoPath) {
        return fmt.Errorf("base directory cannot be inside repository: %s", baseDir)
    }

    return nil
}
```

### Phase 2: Integration Updates

#### 2.1 Update WorktreeManager
**File:** `internal/git/worktree.go`

**Modify CreateWorktree to validate base directory:**
```go
func (wm *WorktreeManager) CreateWorktree(branch string, opts WorktreeOptions) (*WorktreeInfo, error) {
    // ... existing validation code ...

    // NEW: Validate base directory configuration
    if err := wm.patternMgr.ValidateBaseDirectory(wm.config.Worktree.BaseDirectory, wm.repo.RootPath); err != nil {
        return nil, fmt.Errorf("invalid base directory configuration: %w", err)
    }

    // ... rest of existing code unchanged ...
}
```

#### 2.2 Add Configuration Migration Support
**File:** `internal/config/migration.go`

**Add migration logic for existing configurations:**
```go
// MigrateWorktreeConfig migrates legacy worktree configurations
func (c *Config) MigrateWorktreeConfig() bool {
    changed := false
    
    // If BaseDirectory is not set and we're using legacy pattern, migrate
    if c.Worktree.BaseDirectory == "" {
        c.Worktree.BaseDirectory = "../.worktrees/{{.Project}}"
        changed = true
    }
    
    // Update pattern if it's still the old default and doesn't include project context
    if c.Worktree.DirectoryPattern == "{{.Project}}-{{.Branch}}" {
        c.Worktree.DirectoryPattern = "{{.Branch}}"
        changed = true
    }
    
    return changed
}
```

**Update main migration function:**
```go
func (c *Config) Migrate() error {
    changed := false
    
    // ... existing migrations ...
    
    // NEW: Worktree configuration migration
    if c.MigrateWorktreeConfig() {
        changed = true
    }
    
    if changed {
        if err := c.Save(); err != nil {
            return fmt.Errorf("failed to save migrated config: %w", err)
        }
    }
    
    return nil
}
```

### Phase 3: Testing and Validation

#### 3.1 Unit Tests
**File:** `internal/git/patterns_test.go`

**Add comprehensive test cases:**
```go
func TestGenerateWorktreePath_SiblingDirectory(t *testing.T) {
    tests := []struct {
        name        string
        baseDir     string
        pattern     string
        branch      string
        project     string
        expected    string
        shouldError bool
    }{
        {
            name:     "default sibling pattern",
            baseDir:  "../.worktrees/{{.Project}}",
            pattern:  "{{.Branch}}",
            branch:   "feature/auth",
            project:  "myapp",
            expected: "../.worktrees/myapp/feature-auth",
        },
        {
            name:     "absolute base directory",
            baseDir:  "/tmp/worktrees/{{.Project}}",
            pattern:  "{{.Branch}}",
            branch:   "main",
            project:  "testapp",
            expected: "/tmp/worktrees/testapp/main",
        },
        {
            name:        "base directory inside repo should error in validation",
            baseDir:     ".worktrees",
            pattern:     "{{.Branch}}",
            branch:      "test",
            project:     "app",
            shouldError: false, // This test is for path generation, not validation
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            config := &config.WorktreeConfig{
                BaseDirectory:    tt.baseDir,
                DirectoryPattern: tt.pattern,
            }
            pm := NewPatternManager(config)
            
            path, err := pm.GenerateWorktreePath(tt.branch, tt.project)
            
            if tt.shouldError {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Contains(t, path, tt.expected)
            }
        })
    }
}

func TestValidateBaseDirectory(t *testing.T) {
    tests := []struct {
        name        string
        baseDir     string
        repoPath    string
        shouldError bool
        errorMsg    string
    }{
        {
            name:        "sibling directory should pass",
            baseDir:     "../.worktrees/myproject",
            repoPath:    "/home/user/repos/myproject",
            shouldError: false,
        },
        {
            name:        "absolute path outside repo should pass",
            baseDir:     "/tmp/worktrees",
            repoPath:    "/home/user/repos/myproject",
            shouldError: false,
        },
        {
            name:        "directory inside repo should fail",
            baseDir:     ".worktrees",
            repoPath:    "/home/user/repos/myproject",
            shouldError: true,
            errorMsg:    "cannot be inside repository",
        },
        {
            name:        "empty base directory should fail",
            baseDir:     "",
            repoPath:    "/home/user/repos/myproject",
            shouldError: true,
            errorMsg:    "cannot be empty",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            pm := &PatternManager{}
            err := pm.ValidateBaseDirectory(tt.baseDir, tt.repoPath)
            
            if tt.shouldError {
                assert.Error(t, err)
                if tt.errorMsg != "" {
                    assert.Contains(t, err.Error(), tt.errorMsg)
                }
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

#### 3.2 Integration Tests
**File:** `internal/git/worktree_integration_test.go`

**Add end-to-end test:**
```go
func TestWorktreeCreation_SiblingDirectory(t *testing.T) {
    // Setup temporary repository
    tempDir := t.TempDir()
    repoDir := filepath.Join(tempDir, "test-repo")
    
    // Initialize git repository
    gitCmd := NewGitCmd()
    _, err := gitCmd.Execute(tempDir, "init", "test-repo")
    require.NoError(t, err)
    
    // Create initial commit
    _, err = gitCmd.Execute(repoDir, "commit", "--allow-empty", "-m", "initial commit")
    require.NoError(t, err)
    
    // Setup config with sibling worktree directory
    config := &config.Config{
        Worktree: config.WorktreeConfig{
            BaseDirectory:    "../.worktrees/{{.Project}}",
            DirectoryPattern: "{{.Branch}}",
            AutoDirectory:    true,
        },
    }
    
    // Create repository and worktree manager
    repo, err := NewRepository(repoDir)
    require.NoError(t, err)
    
    wm := NewWorktreeManager(repo, config, gitCmd)
    
    // Create worktree
    opts := WorktreeOptions{
        CreateBranch: true,
        Checkout:     true,
    }
    
    worktreeInfo, err := wm.CreateWorktree("feature/test", opts)
    
    // Verify success
    assert.NoError(t, err)
    assert.NotNil(t, worktreeInfo)
    
    // Verify path is outside repository
    expectedPath := filepath.Join(tempDir, ".worktrees", "test-repo", "feature-test")
    assert.Equal(t, expectedPath, worktreeInfo.Path)
    
    // Verify directory was created
    assert.DirExists(t, worktreeInfo.Path)
    
    // Verify it's a valid git worktree
    _, err = gitCmd.Execute(worktreeInfo.Path, "status")
    assert.NoError(t, err)
}
```

### Phase 4: Documentation and Migration

#### 4.1 Update Configuration Documentation
**File:** `docs/configuration.md`

**Add section about worktree configuration:**
```markdown
## Worktree Configuration

### Base Directory

The `base_directory` setting controls where worktrees are created. It supports template variables and can be either relative or absolute:

```yaml
worktree:
  # Relative path (recommended) - creates worktrees as siblings to repository
  base_directory: "../.worktrees/{{.Project}}"
  
  # Absolute path - fixed location for all projects
  base_directory: "/home/user/worktrees/{{.Project}}"
  
  # Simple relative path
  base_directory: "../my-worktrees"
```

### Migration from v1.x

If you're upgrading from a previous version that created worktrees inside the repository (`.worktrees/`), your configuration will be automatically migrated to use sibling directories. Existing worktrees will continue to work but new ones will be created in the new location.

To manually clean up old worktrees:
```bash
# List existing worktrees
ccmgr-ultra worktree list

# Remove old worktrees (will not delete the directories)
git worktree remove .worktrees/old-worktree-name

# Manually delete old .worktrees directory if empty
rmdir .worktrees
```
```

#### 4.2 Add Migration Guide
**File:** `docs/migration-v2.md`

**Create migration guide:**
```markdown
# Migration Guide: Worktree Directory Changes

## Overview

Version 2.0 changes how worktree directories are organized to comply with git's constraints. Worktrees are now created outside the repository directory by default.

## Changes

### Before (v1.x)
```
my-project/
├── .git/
├── .worktrees/
│   ├── feature-auth/
│   └── bugfix-123/
└── source-files...
```

### After (v2.x)
```
parent-directory/
├── my-project/           # Original repository
│   ├── .git/
│   └── source-files...
└── .worktrees/           # Worktrees moved outside
    └── my-project/
        ├── feature-auth/
        └── bugfix-123/
```

## Automatic Migration

Your configuration will be automatically updated when you first run ccmgr-ultra v2.0:

- `base_directory` will be set to `"../.worktrees/{{.Project}}"`
- `directory_pattern` will be simplified to `"{{.Branch}}"`

## Manual Steps (Optional)

1. **List existing worktrees:**
   ```bash
   git worktree list
   ```

2. **Remove old worktrees** (if you want to recreate them in new location):
   ```bash
   git worktree remove .worktrees/old-branch-name
   ```

3. **Delete old .worktrees directory** (if empty):
   ```bash
   rmdir .worktrees
   ```

4. **Recreate worktrees** in new location:
   ```bash
   ccmgr-ultra worktree create branch-name
   ```

## Rollback

To revert to the old behavior (not recommended):

```yaml
worktree:
  base_directory: ".worktrees"
```

**Note:** This will cause git worktree creation to fail unless you add `.worktrees/` to `.gitignore`.
```

### Phase 5: Configuration Updates

#### 5.1 Update Example Configuration
**File:** `example-claude-config.yaml`

**Update worktree section:**
```yaml
worktree:
  auto_directory: true
  # Base directory for worktrees (outside repository)
  base_directory: "../.worktrees/{{.Project}}"
  # Pattern for individual worktree directories
  directory_pattern: "{{.Branch}}"
  default_branch: "main"
  cleanup_on_merge: true
```

#### 5.2 Update .gitignore
**File:** `.gitignore`

**Add worktrees directory pattern:**
```gitignore
# Worktrees (for any legacy setups)
.worktrees/
```

## Implementation Checklist

### Core Changes
- [ ] Update `WorktreeConfig` schema with `BaseDirectory` field
- [ ] Update `SetDefaults()` method for new default values
- [ ] Modify `GenerateWorktreePath()` function for sibling directory pattern
- [ ] Add `ValidateBaseDirectory()` function
- [ ] Update `CreateWorktree()` to validate base directory

### Configuration & Migration
- [ ] Add configuration migration logic
- [ ] Update example configuration file
- [ ] Add .gitignore entry for legacy support

### Testing
- [ ] Add unit tests for path generation
- [ ] Add unit tests for base directory validation
- [ ] Add integration tests for worktree creation
- [ ] Add migration tests

### Documentation
- [ ] Update configuration documentation
- [ ] Create migration guide
- [ ] Update README with new behavior
- [ ] Add troubleshooting section

### Validation
- [ ] Test with various repository layouts
- [ ] Test absolute vs relative base directories
- [ ] Test template variable resolution
- [ ] Verify backward compatibility
- [ ] Test error handling and edge cases

## Risk Assessment

### Low Risk
- Configuration changes (backward compatible)
- Path generation logic (well-isolated)
- Documentation updates

### Medium Risk
- Migration logic (needs thorough testing)
- Integration with existing worktree workflows

### Mitigation Strategies
- Comprehensive test coverage
- Gradual rollout with feature flag option
- Clear migration documentation
- Rollback procedures documented

## Success Metrics

1. **Functional Success:**
   - Worktree creation succeeds with default configuration
   - No git constraint violations
   - Existing configurations work without modification

2. **Performance Success:**
   - No significant performance impact on worktree operations
   - Path resolution remains fast

3. **User Experience Success:**
   - Seamless upgrade experience
   - Clear documentation for new users
   - Troubleshooting guide for edge cases

## Future Enhancements

### Phase 6 (Future)
- GUI configuration interface for worktree settings
- Advanced template functions for path generation
- Integration with external worktree management tools
- Worktree cleanup automation based on age/usage

### Potential Extensions
- Cloud storage integration for worktrees
- Shared worktree pools for team environments
- Advanced conflict resolution for worktree paths