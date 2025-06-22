# Technical Specification for Issue #1

## Issue Summary
- Title: Output of status is hard to read
- Description: The current status command output displays raw struct data in a difficult-to-read format with Go struct syntax
- Labels: None
- Priority: High

## Problem Statement
The `ccmgr-ultra status` command currently uses a basic `SimpleTableFormatter` that outputs raw struct data in Go's native format (e.g., `{false 2 1 1 1 0 0 0 false true 0s}`). This makes it extremely difficult for users to understand the status information at a glance. The output lacks proper formatting, labels, and structure that would make the data meaningful and actionable.

The project already has a sophisticated `TableFormatter` with borders, colors, and proper alignment capabilities, but the status command is not leveraging it. Instead, it uses the basic formatter that was likely intended as a fallback option.

## Technical Approach
Replace the `SimpleTableFormatter` usage in the status command with the existing `TableFormatter` to create properly formatted, readable output. The solution will:
1. Create structured table views for each status section (System, Worktrees, Sessions, Processes)
2. Use meaningful column headers and proper data formatting
3. Apply consistent styling with borders and colors for better visual hierarchy
4. Format data types appropriately (booleans as checkmarks, durations in human-readable format)
5. Maintain backward compatibility with JSON/YAML output formats

## Implementation Plan
1. Create a new `StatusTableFormatter` that uses the existing `TableFormatter` internally
2. Define table columns and formatting for each status section:
   - System overview table
   - Worktrees table with clean/dirty status indicators
   - Sessions table with activity indicators
   - Processes table with resource usage
3. Update `setupOutputFormatter` to return the new formatter for table format
4. Add formatting helpers for status-specific data (e.g., health indicators, state icons)
5. Update the simple formatter to act as a true fallback for non-terminal environments

## Test Plan
1. Unit Tests:
   - Test StatusTableFormatter with various status data scenarios
   - Test formatting of edge cases (empty data, nil values)
   - Test column width calculations with long values
2. Component Tests:
   - Test integration with status command
   - Test format switching between table/json/yaml
   - Test color output toggling based on terminal capabilities
3. Integration Tests:
   - Test full status command execution with real data
   - Test watch mode with formatted output
   - Test filtering with formatted output

## Files to Modify
- `/Users/brian/code/ccmgr-ultra/internal/cli/output.go`: Add StatusTableFormatter implementation
- `/Users/brian/code/ccmgr-ultra/cmd/ccmgr-ultra/common.go`: Update setupOutputFormatter to use new formatter
- `/Users/brian/code/ccmgr-ultra/cmd/ccmgr-ultra/status.go`: Minor adjustments for formatter integration

## Files to Create
- `/Users/brian/code/ccmgr-ultra/internal/cli/status_formatter.go`: New formatter specifically for status output
- `/Users/brian/code/ccmgr-ultra/internal/cli/status_formatter_test.go`: Unit tests for the new formatter

## Existing Utilities to Leverage
- `/Users/brian/code/ccmgr-ultra/internal/cli/table.go`: Comprehensive TableFormatter with all needed features
- Color and styling support already built into TableFormatter
- Terminal detection utilities for adaptive formatting
- Existing format validation and switching logic

## Success Criteria
- [ ] Status output displays structured tables with clear headers and borders
- [ ] Each data type is formatted appropriately (booleans as ✓/✗, durations as "2h 30m")
- [ ] System health is visually indicated with colors/icons
- [ ] Output is readable and immediately understandable
- [ ] JSON/YAML formats continue to work unchanged
- [ ] No regression in watch mode functionality
- [ ] Output adapts to terminal capabilities (colors, width)

## Out of Scope
- Changing the underlying data collection logic
- Adding new status information or metrics
- Modifying the command-line interface or flags
- Creating interactive features or real-time updates beyond existing watch mode