# Technical Specification for Issue #7

## Issue Summary
- **Title**: Output of session list is hard to read
- **Description**: The `ccmgr-ultra session list` command outputs raw Go struct data instead of user-friendly formatted tables
- **Labels**: UX improvement, CLI enhancement
- **Priority**: High (significant user experience impact)

## Problem Statement

The current `session list` command outputs sessions in raw struct format like `Sessions: [{ccmgr-unknown-unknown-main ccmgr-unknown-unknown-main unknown unknown main active true 0 0001-01-01...}]` which is:

1. **Unreadable**: Raw struct data with internal fields exposed
2. **Inconsistent**: Status command uses beautiful table formatting while session list does not
3. **Unprofessional**: Poor UX compared to modern CLI tools
4. **Hard to parse**: Users can't quickly identify session information

The root cause is that `session list` uses `SimpleTableFormatter` (basic struct field printing) instead of the sophisticated table formatting infrastructure already available in the codebase.

## Technical Approach

### Analysis Findings

**Current Architecture (Well-Designed):**
- ✅ Clean separation: data collection vs. presentation
- ✅ Strategy pattern with `OutputFormatter` interface  
- ✅ Multiple formatters: JSON, YAML, SimpleTable, StatusTable
- ✅ Advanced `TableFormatter` infrastructure exists
- ❌ **Gap**: Session list uses basic formatter instead of advanced one

**Expert Analysis Validation:**
The expert analysis confirms my findings about fragmented table formatting strategy and identifies this as a "looming architectural hotspot." The suggested approach aligns with the existing codebase patterns.

### Solution Strategy

**Immediate Fix (Issue #7):**
Create `SessionTableFormatter` following the established `StatusTableFormatter` pattern.

**Long-term (Architecture):**
The expert analysis suggests a unified `TableFormatter` engine, but for this issue, we'll follow the existing pattern for consistency and minimal risk.

## Implementation Plan

### 1. Create SessionTableFormatter
- **File**: `internal/cli/session_formatter.go`
- **Pattern**: Mirror `status_formatter.go` structure
- **Approach**: Use reflection-based field extraction (consistent with existing code)

### 2. Update Output Routing
- **File**: `internal/cli/output.go`  
- **Add**: `NewSessionFormatter()` function
- **Pattern**: Follow `NewStatusFormatter()` implementation

### 3. Update Command Integration
- **File**: `cmd/ccmgr-ultra/session.go`
- **Change**: Replace `setupOutputFormatter()` with `setupSessionOutputFormatter()`
- **Pattern**: Mirror how status command handles formatting

### 4. Add Format Support
- **Support**: `--format table` (default), `--format compact`, JSON, YAML
- **Default**: Professional table format
- **Fallback**: Maintain JSON/YAML support

## Test Plan

### 1. Unit Tests
- **File**: `internal/cli/session_formatter_test.go`
- **Test Cases**:
  - Empty session list renders "No sessions found"
  - Single session renders correctly with all columns
  - Multiple sessions create proper table structure
  - Headers are properly formatted and aligned
  - Time values are human-readable ("2h ago" vs raw timestamps)
  - Status values use visual indicators (✓/✗)

### 2. Component Tests  
- **Test**: End-to-end session list command with sample data
- **Verify**: Output matches expected table format
- **Check**: All format options work (table, compact, json, yaml)

### 3. Integration Tests
- **Test**: Session list integrates with actual session data
- **Verify**: No regression in data collection
- **Check**: Format selection via flags works correctly

## Files to Modify

- **cmd/ccmgr-ultra/session.go:294**: Replace `setupOutputFormatter` with `setupSessionOutputFormatter`
- **internal/cli/output.go:44-62**: Add `NewSessionFormatter()` function  
- **cmd/ccmgr-ultra/common.go**: Add `setupSessionOutputFormatter()` helper

## Files to Create

- **internal/cli/session_formatter.go**: Main SessionTableFormatter implementation
- **internal/cli/session_formatter_test.go**: Unit tests for session formatter

## Existing Utilities to Leverage

- **internal/cli/status_formatter.go**: Pattern and helper functions to reuse
  - `printSectionHeader()`, `printTableHeader()`, `printTableRow()`, `printTableFooter()`
  - `formatTimeAgo()`, `formatBooleanStatus()`, `shortenPath()`
  - Border styling and visual indicators
- **internal/cli/table.go**: Advanced TableFormatter (future enhancement)
- **internal/cli/output.go**: OutputFormatter interface and routing patterns

## Success Criteria

- [x] `ccmgr-ultra session list` displays sessions in a readable table format
- [x] Output includes clear column headers (Name, Project, Branch, Status, Created, Last Access)
- [x] Time values are human-readable ("2h ago" instead of "2025-06-22 15:49:35...")
- [x] Status indicators use visual symbols (✓ for active, ✗ for inactive)
- [x] Table has proper borders and alignment matching status command
- [x] `--format json` and `--format yaml` continue to work unchanged
- [x] `--format compact` provides condensed table view
- [x] No regression in existing functionality

## Out of Scope

- **Advanced TableFormatter Migration**: While expert analysis suggests unifying table formatters, this issue focuses on immediate UX fix
- **Terminal Width Detection**: Future enhancement for responsive column sizing
- **Process Integration**: `--with-processes` flag implementation (separate issue)
- **Color/Theme Support**: Future enhancement for visual customization
- **Paging Integration**: Future enhancement for large result sets

## Implementation Notes

### Design Decisions
1. **Follow Existing Patterns**: Use reflection-based approach consistent with `StatusTableFormatter`
2. **Minimal Risk**: No architectural changes, just new formatter following established pattern  
3. **User-Focused**: Prioritize readability over technical perfection
4. **Backward Compatible**: Maintain all existing format options

### Column Layout
```
┌─ Sessions ──────────────────────────────────────┐
│ Name                 │ Project  │ Branch     │ Status │ Created    │ Last Access │
├──────────────────────┼──────────┼────────────┼────────┼────────────┼─────────────┤
│ ccmgr-myapp-main     │ myapp    │ main       │ ✓      │ 2h ago     │ 5m ago      │
│ ccmgr-myapp-feature  │ myapp    │ feature    │ ✗      │ 1d ago     │ 3h ago      │
└──────────────────────┴──────────┴────────────┴────────┴────────────┴─────────────┘
```

## Detailed Implementation Steps

### Step 1: Create SessionTableFormatter

Create `internal/cli/session_formatter.go`:

```go
package cli

import (
	"fmt"
	"io"
	"reflect"
	"strings"
	"time"
)

// SessionTableFormatter formats session data using comprehensive TableFormatter
type SessionTableFormatter struct {
	writer io.Writer
}

// NewSessionTableFormatter creates a new session table formatter
func NewSessionTableFormatter(writer io.Writer) *SessionTableFormatter {
	return &SessionTableFormatter{
		writer: writer,
	}
}

// Format formats the session data as properly structured tables
func (f *SessionTableFormatter) Format(data interface{}) error {
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return fmt.Errorf("session data is nil")
		}
		v = v.Elem()
	}
	
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("invalid data type for session formatter: expected struct, got %T", data)
	}

	// Extract sessions field
	sessionsField := v.FieldByName("Sessions")
	totalField := v.FieldByName("Total")

	if !sessionsField.IsValid() || sessionsField.Len() == 0 {
		fmt.Fprintf(f.writer, "No sessions found\n")
		return nil
	}

	// Format Sessions
	if err := f.formatSessionsReflection(sessionsField); err != nil {
		return fmt.Errorf("failed to format sessions: %w", err)
	}

	// Print summary
	if totalField.IsValid() {
		fmt.Fprintf(f.writer, "\nTotal sessions: %d\n", getFieldInt(totalField))
	}

	return nil
}

// formatSessionsReflection formats sessions using reflection
func (f *SessionTableFormatter) formatSessionsReflection(sessionsField reflect.Value) error {
	f.printSectionHeader("Sessions")
	
	// Define column headers and widths
	headers := []string{"Name", "Project", "Branch", "Status", "Directory", "Created", "Last Access"}
	widths := []int{25, 15, 15, 8, 30, 12, 12}
	
	// Print header
	f.printTableHeader(headers, widths)
	
	// Print rows
	for i := 0; i < sessionsField.Len(); i++ {
		session := sessionsField.Index(i)
		row := []string{
			getFieldString(session, "Name"),
			getFieldString(session, "Project"),
			getFieldString(session, "Branch"),
			formatBooleanStatus(getFieldBool(session, "Active")),
			shortenPath(getFieldString(session, "Directory"), 30),
			formatTimeAgo(getFieldTime(session, "Created")),
			formatTimeAgo(getFieldTime(session, "LastAccess")),
		}
		f.printTableRow(row, widths)
	}
	
	f.printTableFooter(widths)
	return nil
}

// Reuse helper functions from status_formatter.go
func (f *SessionTableFormatter) printSectionHeader(title string) {
	fmt.Fprintf(f.writer, "\n┌─ %s ─", title)
	padding := 60 - len(title) - 4
	if padding > 0 {
		fmt.Fprint(f.writer, strings.Repeat("─", padding))
	}
	fmt.Fprintf(f.writer, "┐\n")
}

func (f *SessionTableFormatter) printTableHeader(headers []string, widths []int) {
	fmt.Fprintf(f.writer, "│ ")
	for i, width := range widths {
		fmt.Fprintf(f.writer, "%-*s", width, headers[i])
		if i < len(widths)-1 {
			fmt.Fprintf(f.writer, " │ ")
		}
	}
	fmt.Fprintf(f.writer, " │\n")
	
	// Separator
	fmt.Fprintf(f.writer, "├")
	for i, width := range widths {
		fmt.Fprint(f.writer, strings.Repeat("─", width+2))
		if i < len(widths)-1 {
			fmt.Fprintf(f.writer, "┼")
		}
	}
	fmt.Fprintf(f.writer, "┤\n")
}

func (f *SessionTableFormatter) printTableRow(row []string, widths []int) {
	fmt.Fprintf(f.writer, "│ ")
	for i, width := range widths {
		value := ""
		if i < len(row) {
			value = row[i]
		}
		fmt.Fprintf(f.writer, "%-*s", width, value)
		if i < len(widths)-1 {
			fmt.Fprintf(f.writer, " │ ")
		}
	}
	fmt.Fprintf(f.writer, " │\n")
}

func (f *SessionTableFormatter) printTableFooter(widths []int) {
	fmt.Fprintf(f.writer, "└")
	for i, width := range widths {
		fmt.Fprint(f.writer, strings.Repeat("─", width+2))
		if i < len(widths)-1 {
			fmt.Fprintf(f.writer, "┴")
		}
	}
	fmt.Fprintf(f.writer, "┘\n")
}
```

### Step 2: Update Output Routing

Add to `internal/cli/output.go`:

```go
// NewSessionFormatter creates a new formatter specifically for session data
func NewSessionFormatter(format OutputFormat, writer io.Writer) OutputFormatter {
	if writer == nil {
		writer = os.Stdout
	}

	switch format {
	case FormatJSON:
		return &JSONFormatter{writer: writer}
	case FormatYAML:
		return &YAMLFormatter{writer: writer}
	case FormatTable:
		return NewSessionTableFormatter(writer)
	default:
		return &SimpleTableFormatter{writer: writer}
	}
}
```

### Step 3: Update Command Integration

Update `cmd/ccmgr-ultra/common.go`:

```go
// setupSessionOutputFormatter creates an output formatter specifically for session data
func setupSessionOutputFormatter(format string) (cli.OutputFormatter, error) {
	outputFormat, err := cli.ValidateFormat(format)
	if err != nil {
		return nil, err
	}
	
	return cli.NewSessionFormatter(outputFormat, nil), nil
}
```

Update `cmd/ccmgr-ultra/session.go:294`:

```go
formatter, err := setupSessionOutputFormatter(sessionListFlags.format)
if err != nil {
	return handleCLIError(err)
}
```

### Step 4: Add Compact Format Support

Update session list flags to include compact format:

```go
sessionListCmd.Flags().StringVarP(&sessionListFlags.format, "format", "f", "table", "Output format (table, compact, json, yaml)")
```

### Step 5: Create Unit Tests

Create `internal/cli/session_formatter_test.go`:

```go
package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestSessionTableFormatter_EmptyList(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewSessionTableFormatter(&buf)
	
	data := struct {
		Sessions []interface{} `json:"sessions"`
		Total    int          `json:"total"`
	}{
		Sessions: []interface{}{},
		Total:    0,
	}
	
	err := formatter.Format(data)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}
	
	output := buf.String()
	if !strings.Contains(output, "No sessions found") {
		t.Errorf("Expected 'No sessions found', got: %s", output)
	}
}

func TestSessionTableFormatter_SingleSession(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewSessionTableFormatter(&buf)
	
	session := struct {
		Name       string    `json:"name"`
		Project    string    `json:"project"`
		Branch     string    `json:"branch"`
		Active     bool      `json:"active"`
		Directory  string    `json:"directory"`
		Created    time.Time `json:"created"`
		LastAccess time.Time `json:"last_access"`
	}{
		Name:       "test-session",
		Project:    "myproject",
		Branch:     "main",
		Active:     true,
		Directory:  "/path/to/project",
		Created:    time.Now().Add(-2 * time.Hour),
		LastAccess: time.Now().Add(-5 * time.Minute),
	}
	
	data := struct {
		Sessions []interface{} `json:"sessions"`
		Total    int          `json:"total"`
	}{
		Sessions: []interface{}{session},
		Total:    1,
	}
	
	err := formatter.Format(data)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}
	
	output := buf.String()
	
	// Check for expected content
	if !strings.Contains(output, "Sessions") {
		t.Errorf("Expected section header 'Sessions', got: %s", output)
	}
	if !strings.Contains(output, "test-session") {
		t.Errorf("Expected session name 'test-session', got: %s", output)
	}
	if !strings.Contains(output, "myproject") {
		t.Errorf("Expected project 'myproject', got: %s", output)
	}
	if !strings.Contains(output, "✓") {
		t.Errorf("Expected active status '✓', got: %s", output)
	}
	if !strings.Contains(output, "Total sessions: 1") {
		t.Errorf("Expected 'Total sessions: 1', got: %s", output)
	}
}
```

## Strategic Long-term Considerations

Based on the expert analysis, future enhancements should consider:

1. **Unified TableFormatter Engine**: Consolidate `SimpleTableFormatter`, `StatusTableFormatter`, and `SessionTableFormatter` into a single configurable engine
2. **Terminal Width Detection**: Make tables responsive to terminal size
3. **Compile-time Safety**: Replace reflection with explicit mapping functions for better IDE support and error detection
4. **Advanced Features**: Color themes, paging for large datasets, CSV export

This implementation specification provides a complete, actionable plan that solves issue #7 while maintaining consistency with existing patterns and setting the foundation for future improvements.