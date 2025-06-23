# Implementation Plan: Fix Session New Command Crash (Issue #12)

## Overview

This document outlines the implementation plan to fix the critical crash in the `ccmgr-ultra session new` command caused by a flag shorthand collision between the global `-n` flag and a local `-n` flag.

## Problem Analysis

### Root Cause
The `ccmgr-ultra session new` command crashes with a panic due to a flag shorthand collision in the Cobra CLI framework:

- **Global persistent flag**: `--non-interactive` uses `-n` shorthand (main.go:58)
- **Local session new flag**: `--name` uses `-n` shorthand (session.go:153)

When Cobra attempts to merge persistent flags with local flags during command initialization, it panics because it cannot redefine the same shorthand letter.

### Error Details
```
panic: unable to redefine 'n' shorthand in "new" flagset: it's already used for "name" flag
```

This occurs in `github.com/spf13/pflag.(*FlagSet).AddFlag()` when the flag merging process detects the duplicate shorthand usage.

## Technical Solution

### Approach
Change the shorthand for the `--name` flag in the session new command to eliminate the collision. The global `-n` for `--non-interactive` follows common CLI conventions and should be preserved.

### Shorthand Selection
**Recommended option**: Remove shorthand entirely (use long form only)
- Reasoning: The `--name` flag is optional and less frequently used
- Alternative options considered:
  - `-N` (uppercase N) - could be confusing with lowercase
  - `-s` (for session suffix) - less intuitive

## Implementation Steps

### Step 1: Update Flag Definition
**File**: `/Users/brian/code/ccmgr-ultra/cmd/ccmgr-ultra/session.go`
**Line**: 153

**Current code**:
```go
sessionNewCmd.Flags().StringVarP(&sessionNewFlags.name, "name", "n", "", "Custom session name suffix")
```

**Updated code**:
```go
sessionNewCmd.Flags().StringVar(&sessionNewFlags.name, "name", "", "Custom session name suffix")
```

### Step 2: Verify Fix
1. Build the application: `make build`
2. Test help output: `./build/ccmgr-ultra session new --help`
3. Test command execution: `./build/ccmgr-ultra session new test-worktree`
4. Test with name flag: `./build/ccmgr-ultra session new test-worktree --name custom-name`

### Step 3: Update Tests
Create or update tests to verify:
- Flag parsing works correctly
- Command executes without panic
- Backward compatibility with long form maintained

### Step 4: Documentation Review
- Verify help text displays correctly
- No documentation references the removed shorthand

## Testing Strategy

### Unit Tests
```go
func TestSessionNewFlagParsing(t *testing.T) {
    // Test that command can be created without panic
    cmd := sessionNewCmd
    assert.NotNil(t, cmd)
    
    // Test flag parsing
    cmd.SetArgs([]string{"test-worktree", "--name", "custom-name"})
    err := cmd.Execute()
    // Verify no flag parsing errors
}
```

### Integration Tests
1. **Command Help Test**:
   ```bash
   ./build/ccmgr-ultra session new --help
   # Should display help without panic
   ```

2. **Flag Usage Test**:
   ```bash
   ./build/ccmgr-ultra session new test-worktree --name custom-session
   # Should work with long form flag
   ```

3. **Global Flag Test**:
   ```bash
   ./build/ccmgr-ultra -n session new test-worktree
   # Should work with global non-interactive flag
   ```

## Risk Assessment

### Risks
- **Low**: Breaking change for users using `-n` shorthand (unlikely as command was crashing)
- **Low**: Confusion from removing shorthand

### Mitigation
- The command was completely broken, so removing shorthand doesn't reduce functionality
- Long form `--name` continues to work exactly as before
- Users can still use all functionality, just with slightly more typing

## Validation Criteria

### Pre-Implementation Validation
- [ ] Verify current crash occurs with `ccmgr-ultra session new`
- [ ] Confirm root cause is flag shorthand collision
- [ ] Identify all affected scenarios

### Post-Implementation Validation
- [ ] `ccmgr-ultra session new --help` executes without panic
- [ ] `ccmgr-ultra session new <worktree>` executes successfully  
- [ ] `ccmgr-ultra session new <worktree> --name <name>` works correctly
- [ ] Global `-n` flag continues to work: `ccmgr-ultra -n session list`
- [ ] No other commands are affected
- [ ] All existing tests pass
- [ ] Build process completes successfully

## Timeline

- **Estimated effort**: 15 minutes implementation + 15 minutes testing
- **Complexity**: Trivial (single line change)
- **Dependencies**: None

## Rollback Plan

If issues arise, rollback is simple:
1. Revert the single line change in session.go
2. Rebuild the application
3. The original crash behavior will return, but no new issues introduced

## Files Modified

### Primary Changes
- `cmd/ccmgr-ultra/session.go` - Line 153 flag definition

### Test Changes (if needed)
- Add test case to verify flag collision is resolved
- Update any existing tests that might reference the removed shorthand

## Success Metrics

- Zero crashes when running `ccmgr-ultra session new`
- 100% of existing session new functionality preserved
- Command help displays correctly
- All CI/CD tests pass

## Notes

This is a minimal, surgical fix that resolves a critical issue with the least possible risk. The solution follows the KISS principle and maintains backward compatibility for all documented functionality while making the broken command usable again.