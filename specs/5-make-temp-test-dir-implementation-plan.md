# Technical Specification for Issue #5

## Issue Summary
- Title: Add a make command to create a temp dir and init git repo for testing
- Description: Create a Makefile command for user acceptance testing that creates a temporary directory, initializes a git repository, and commits a basic README
- Labels: None
- Priority: Medium

## Problem Statement
During development of the CLI tool, developers need a quick way to set up isolated test environments for user acceptance testing. Currently, this process is manual and repetitive. The solution should automate the creation of a clean git repository in a temporary directory with minimal initial content, allowing developers to test CLI functionality without affecting their main project workspace.

## Technical Approach
Create a new Makefile target that automates the test environment setup process by:
1. Creating a dedicated `.testdirs/` directory at the project root for all test environments
2. Generating timestamped subdirectories to avoid conflicts between test sessions
3. Initializing a git repository with standard configuration
4. Creating and committing an initial README.md file
5. Providing clear instructions for the user to navigate to the test directory

The implementation will follow existing Makefile patterns including use of `.PHONY` declarations, `@echo` for user feedback, and proper help text documentation.

## Implementation Plan
1. Add `.testdirs/` to `.gitignore` to prevent test directories from being committed
2. Create `test-env` target in Makefile with the following steps:
   - Create `.testdirs` directory if it doesn't exist
   - Generate unique subdirectory using timestamp format
   - Initialize git repository in the subdirectory
   - Create README.md with "test README" content
   - Stage and commit the README file
   - Display cd command for user to change to the directory
3. Optionally add `test-env-clean` target to remove all test directories
4. Add appropriate help text following the existing `## Description` pattern

## Test Plan
1. Unit Tests:
   - Not applicable (Makefile target)
2. Component Tests:
   - Manual verification of directory creation
   - Verify git repository initialization
   - Confirm README.md creation and commit
3. Integration Tests:
   - Test multiple invocations to ensure unique directories
   - Verify proper error handling if permissions are restricted
   - Test cleanup functionality if implemented

## Files to Modify
- `/Users/brian/code/ccmgr-ultra/Makefile`: Add test-env target and optionally test-env-clean target
- `/Users/brian/code/ccmgr-ultra/.gitignore`: Add .testdirs/ entry

## Files to Create
- None (all changes are modifications to existing files)

## Existing Utilities to Leverage
- Standard Unix commands: mkdir, cd, echo
- Git commands: git init, git add, git commit
- Makefile patterns: .PHONY declarations, @ prefix for quiet commands, help text system

## Success Criteria
- [ ] Running `make test-env` creates a new timestamped directory under `.testdirs/`
- [ ] The created directory contains an initialized git repository
- [ ] A README.md file exists with "test README" content and is committed
- [ ] Clear instructions are displayed showing how to cd into the test directory
- [ ] `.testdirs/` is properly ignored by git
- [ ] The command appears in `make help` with appropriate description

## Out of Scope
- Automatic cleanup of old test directories
- Integration with the ccmgr-ultra application itself
- Pre-populated test data beyond the basic README
- Automatic shell navigation (user must manually cd)