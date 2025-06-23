# Technical Specification for Issue #8

## Issue Summary
- **Title**: make test-env should change dir to the temp dir
- **Description**: Enhancement to automatically change to the test directory instead of requiring manual `cd` command
- **Labels**: None
- **Priority**: Medium

## Problem Statement
The current `make test-env` target (implemented in Issue #5) creates a temporary git repository for testing but leaves the user in the original shell, requiring them to manually execute a `cd` command. This creates friction in the developer workflow by adding an unnecessary manual step. The enhancement should automatically place the user in the newly created test directory while maintaining all existing functionality.

## Technical Approach
Replace the current instruction-printing approach with an interactive shell spawning solution that:

1. **Preserves all existing functionality** from the current `test-env` target
2. **Spawns an interactive shell** in the test directory using `bash -i` 
3. **Provides clear visual feedback** about being in a test environment
4. **Maintains user environment** and shell settings
5. **Offers easy exit mechanism** (type `exit` to return)

The solution leverages bash's interactive shell capabilities to create a non-destructive temporary environment that doesn't replace the original shell process.

## Implementation Plan
1. **Modify Makefile target** (lines 121-134):
   - Keep all existing directory creation and git initialization logic
   - Replace the final `echo` instructions with interactive shell spawning
   - Add informational messages about the test environment

2. **Enhanced user experience**:
   - Display clear entry message indicating test environment activation
   - Show the test directory path
   - Provide exit instructions
   - Maintain shell prompt customization if possible

3. **Backward compatibility**:
   - No breaking changes to existing functionality
   - All test environment setup remains identical
   - Directory structure and naming unchanged

## Test Plan
1. **Unit Tests**:
   - Verify directory creation works as before
   - Confirm git repository initialization
   - Test initial commit creation

2. **Component Tests**:
   - Manual verification of interactive shell spawning
   - Confirm user lands in correct directory
   - Test exit functionality returns to original location
   - Verify shell environment preservation

3. **Integration Tests**:
   - Test multiple sequential invocations
   - Verify unique directory creation still works
   - Test behavior with different shell environments
   - Confirm cleanup target still functions

## Files to Modify
- **`/Users/brian/code/ccmgr-ultra/Makefile`**: Update `test-env` target (lines 124-134)
  - Replace final echo statements with interactive shell invocation
  - Add informational messages for better UX

## Files to Create
- None (enhancement to existing functionality)

## Existing Utilities to Leverage
- **Bash interactive shell**: `bash -i` for spawning user-friendly shell
- **Shell variable preservation**: `${SHELL:-bash}` for respecting user preferences  
- **Make variable handling**: Existing `$$TESTDIR` pattern for directory reference
- **Unix process management**: Standard shell process spawning and cleanup

## Success Criteria
- [x] Running `make test-env` creates test directory (existing functionality)
- [x] Git repository is initialized with initial commit (existing functionality)
- [ ] **NEW**: User is automatically placed in the test directory
- [ ] **NEW**: Clear visual feedback indicates test environment status
- [ ] **NEW**: User can type `exit` to return to original directory
- [ ] **NEW**: Shell environment and settings are preserved
- [ ] All existing functionality continues to work unchanged
- [ ] Help text accurately reflects the new behavior

## Out of Scope
- Automatic cleanup of old test directories (already handled by existing `test-env-clean`)
- Integration with tmux or other terminal multiplexers
- Persistent environment variables or shell customization
- Cross-platform shell compatibility beyond bash/zsh
- Automated testing of interactive shell behavior

**Implementation Complexity**: Low
**User Impact**: High (eliminates manual step, improves developer experience)
**Risk Level**: Very Low (non-breaking enhancement to existing functionality)