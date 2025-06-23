# Implementation Plan: Issue #13 - Improve Worktree List Output

## Issue Summary
- **GitHub Issue**: #13
- **Title**: Improve the output of `worktree list`
- **Description**: The current worktree list command shows raw Go struct data instead of user-friendly formatted tables
- **Priority**: High (poor user experience, basic functionality issue)

## Problem Analysis

### Current State
The `ccmgr-ultra worktree list` command outputs raw Go struct data:
```
Worktrees           : [{ccmgr-ultra /Users/brian/code/ccmgr-ultra master 7fc4d50... dirty false {{.Prefix}}-{{.Project}}-{{.Worktree}}-{{.Branch}} 0 2025-06-22...}]
Total               : 4
Timestamp           : 2025-06-22 17:05:56.382126 -0700 PDT
```

### Root Cause
- Worktree list uses `setupOutputFormatter()` which returns `SimpleTableFormatter`
- `SimpleTableFormatter` prints raw Go struct data via reflection
- Status and session commands have specialized formatters (`StatusTableFormatter`, `SessionTableFormatter`)
- Missing `WorktreeTableFormatter` implementation

### Target State
Clean, bordered table format matching other commands:
```
┌─ Worktrees ──────────────────────────────────────────────────────┐
│ Name                         │ Branch               │ Head     │ Status  │ Session         │ Last Access │
├──────────────────────────────┼──────────────────────┼──────────┼─────────┼─────────────────┼─────────────┤
│ ccmgr-ultra                  │ master               │ 7fc4d50  │ ⚠ Dirty │ ccmgr-master    │ 2h ago      │
│ feature-1-improve-status     │ feature/1-improve... │ a007bb2  │ ✓ Clean │ ccmgr-feature-1 │ 4h ago      │
└──────────────────────────────┴──────────────────────┴──────────┴─────────┴─────────────────┴─────────────┘

Total worktrees: 4
```

## Technical Approach

### Architecture Pattern
Follow the established formatter pattern:
1. **Specialized Formatter**: Create `WorktreeTableFormatter` following `StatusTableFormatter` pattern
2. **Factory Function**: Add `NewWorktreeFormatter` to `output.go`
3. **Reflection-based**: Use reflection to extract fields from structs (established pattern)
4. **Consistent Styling**: Apply same visual design as status/session formatters

### Data Structures
**WorktreeListData** (cmd/ccmgr-ultra/worktree.go:18-23):
```go
type WorktreeListData struct {
    Worktrees []WorktreeListItem `json:"worktrees"`
    Total     int                `json:"total"`
    Timestamp time.Time          `json:"timestamp"`
}
```

**WorktreeInfo** (internal/git/repository.go):
```go
type WorktreeInfo struct {
    Path         string
    Branch       string
    Head         string
    IsClean      bool
    HasUncommitted bool
    LastCommit   CommitInfo
    TmuxSession  string
    Created      time.Time
    LastAccessed time.Time
}
```

## Implementation Plan

### Step 1: Create WorktreeTableFormatter
**File**: `internal/cli/worktree_formatter.go`

```go
package cli

import (
    "fmt"
    "io"
    "reflect"
    "strings"
)

// WorktreeTableFormatter formats worktree data using comprehensive TableFormatter
type WorktreeTableFormatter struct {
    writer io.Writer
}

// NewWorktreeTableFormatter creates a new worktree table formatter
func NewWorktreeTableFormatter(writer io.Writer) *WorktreeTableFormatter {
    return &WorktreeTableFormatter{
        writer: writer,
    }
}

// Format formats the worktree data as properly structured tables
func (f *WorktreeTableFormatter) Format(data interface{}) error {
    v := reflect.ValueOf(data)
    if v.Kind() == reflect.Ptr {
        if v.IsNil() {
            return fmt.Errorf("worktree data is nil")
        }
        v = v.Elem()
    }
    
    if v.Kind() != reflect.Struct {
        return fmt.Errorf("invalid data type for worktree formatter: expected struct, got %T", data)
    }

    // Extract fields using reflection
    worktreesField := v.FieldByName("Worktrees")
    totalField := v.FieldByName("Total")

    if !worktreesField.IsValid() || worktreesField.Len() == 0 {
        fmt.Fprintf(f.writer, "No worktrees found\n")
        return nil
    }

    // Format Worktrees
    if err := f.formatWorktreesReflection(worktreesField); err != nil {
        return fmt.Errorf("failed to format worktrees: %w", err)
    }

    // Print summary
    if totalField.IsValid() {
        fmt.Fprintf(f.writer, "\nTotal worktrees: %d\n", int(totalField.Int()))
    }

    return nil
}

// formatWorktreesReflection formats worktrees using reflection
func (f *WorktreeTableFormatter) formatWorktreesReflection(worktreesField reflect.Value) error {
    f.printSectionHeader("Worktrees")
    
    // Define column headers and widths
    headers := []string{"Name", "Branch", "Head", "Status", "Session", "Last Access"}
    widths := []int{25, 20, 10, 10, 15, 12}
    
    // Print header
    f.printTableHeader(headers, widths)
    
    // Print rows
    for i := 0; i < worktreesField.Len(); i++ {
        wt := worktreesField.Index(i)
        head := getFieldString(wt, "Head")
        if len(head) > 8 {
            head = head[:8]
        }
        
        row := []string{
            shortenPath(getFieldString(wt, "Name"), 25),
            shortenPath(getFieldString(wt, "Branch"), 20),
            head,
            formatWorktreeStatusFromFields(getFieldBool(wt, "IsClean")),
            getFieldString(wt, "TmuxSession"),
            formatTimeAgo(getFieldTime(wt, "LastAccessed")),
        }
        f.printTableRow(row, widths)
    }
    
    f.printTableFooter(widths)
    return nil
}

// formatWorktreeStatusFromFields formats worktree status from IsClean field
func formatWorktreeStatusFromFields(isClean bool) string {
    if isClean {
        return "✓ Clean"
    }
    return "⚠ Dirty"
}

// Helper printing functions (reuse from status_formatter.go pattern)
// ... [implementation of printSectionHeader, printTableHeader, etc.]
```

### Step 2: Update output.go
**File**: `internal/cli/output.go`

Add after line 80:
```go
// NewWorktreeFormatter creates a new formatter specifically for worktree data
func NewWorktreeFormatter(format OutputFormat, writer io.Writer) OutputFormatter {
    if writer == nil {
        writer = os.Stdout
    }

    switch format {
    case FormatJSON:
        return &JSONFormatter{writer: writer}
    case FormatYAML:
        return &YAMLFormatter{writer: writer}
    case FormatTable:
        return NewWorktreeTableFormatter(writer)
    default:
        return &SimpleTableFormatter{writer: writer}
    }
}
```

### Step 3: Update common.go
**File**: `cmd/ccmgr-ultra/common.go`

Add after line 69:
```go
// setupWorktreeOutputFormatter creates an output formatter specifically for worktree data
func setupWorktreeOutputFormatter(format string) (cli.OutputFormatter, error) {
    outputFormat, err := cli.ValidateFormat(format)
    if err != nil {
        return nil, err
    }
    
    return cli.NewWorktreeFormatter(outputFormat, nil), nil
}
```

### Step 4: Update worktree.go
**File**: `cmd/ccmgr-ultra/worktree.go`

Replace line 308-313:
```go
// OLD:
formatter, err := setupOutputFormatter(worktreeListFlags.format)
if err != nil {
    return handleCLIError(err)
}

return formatter.Format(listData)

// NEW:
formatter, err := setupWorktreeOutputFormatter(worktreeListFlags.format)
if err != nil {
    return handleCLIError(err)
}

return formatter.Format(listData)
```

## Testing Strategy

### Unit Tests
**File**: `internal/cli/worktree_formatter_test.go`

```go
func TestWorktreeTableFormatter_Format(t *testing.T) {
    tests := []struct {
        name     string
        data     interface{}
        expected string
        wantErr  bool
    }{
        {
            name: "valid worktree data",
            data: WorktreeListData{
                Worktrees: []WorktreeListItem{
                    {
                        Name: "test-worktree",
                        Branch: "feature/test",
                        Head: "abc1234",
                        IsClean: true,
                        TmuxSession: "ccmgr-test",
                        LastAccessed: time.Now().Add(-2 * time.Hour),
                    },
                },
                Total: 1,
            },
            expected: "test-worktree",
            wantErr: false,
        },
        // ... more test cases
    }
    // ... test implementation
}
```

### Integration Tests
Test the complete command flow:
```bash
# Test table format (default)
./ccmgr-ultra worktree list

# Test JSON format
./ccmgr-ultra worktree list --format json

# Test YAML format  
./ccmgr-ultra worktree list --format yaml

# Test with filters
./ccmgr-ultra worktree list --status clean
./ccmgr-ultra worktree list --branch feature
```

## Implementation Details

### Visual Indicators
- ✓ Clean worktree
- ⚠ Dirty worktree  
- 💤 Inactive session
- 🔄 Active session

### Field Formatting
- **Name**: Shortened to 25 chars with "..." if needed
- **Branch**: Shortened to 20 chars with "..." if needed  
- **Head**: Shortened to 8 chars (commit hash)
- **Status**: Visual indicator + text
- **Session**: Session name or empty
- **Last Access**: Human-readable time ago format

### Helper Function Reuse
Leverage existing functions from `status_formatter.go`:
- `formatTimeAgo()`: Convert timestamps to "2h ago" format
- `shortenPath()`: Truncate long paths intelligently
- `getFieldString()`, `getFieldBool()`, `getFieldTime()`: Reflection helpers
- Table rendering functions: headers, rows, footers

## File Structure

```
internal/cli/
├── worktree_formatter.go      # NEW - Main formatter implementation
├── worktree_formatter_test.go # NEW - Unit tests
├── output.go                  # MODIFIED - Add NewWorktreeFormatter
├── status_formatter.go        # REFERENCE - Pattern to follow
└── session_formatter.go       # REFERENCE - Pattern to follow

cmd/ccmgr-ultra/
├── common.go                  # MODIFIED - Add setupWorktreeOutputFormatter
└── worktree.go               # MODIFIED - Use new formatter
```

## Success Criteria

- [ ] `worktree list` displays clean bordered table format
- [ ] Visual indicators show worktree status (✓ Clean, ⚠ Dirty)
- [ ] Table headers and alignment match status/session commands  
- [ ] JSON and YAML formats continue working
- [ ] Paths shortened appropriately for readability
- [ ] Times show as human-readable "time ago" format
- [ ] Consistent styling with other commands
- [ ] All tests pass
- [ ] Code follows project conventions (300-line limit, KISS principle)

## Rollback Plan

If issues arise:
1. Revert `worktree.go` to use `setupOutputFormatter`
2. Remove new files: `worktree_formatter.go`, `worktree_formatter_test.go`
3. Revert changes to `output.go` and `common.go`
4. Original raw output functionality remains intact

## Dependencies

- No external dependencies required
- Leverages existing reflection utilities
- Uses established table rendering patterns
- Compatible with current Go version and dependencies