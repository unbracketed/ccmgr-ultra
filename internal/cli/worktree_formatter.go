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

// printSectionHeader prints a section header with decorative styling
func (f *WorktreeTableFormatter) printSectionHeader(title string) {
	fmt.Fprintf(f.writer, "┌─ %s ─", title)
	padding := 60 - len(title) - 4 // Adjust based on desired width
	if padding > 0 {
		fmt.Fprint(f.writer, strings.Repeat("─", padding))
	}
	fmt.Fprintf(f.writer, "┐\n")
}

// printTableHeader prints a table header with borders
func (f *WorktreeTableFormatter) printTableHeader(headers []string, widths []int) {
	// Top border
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

// printTableRow prints a table data row
func (f *WorktreeTableFormatter) printTableRow(row []string, widths []int) {
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

// printTableFooter prints a table footer border
func (f *WorktreeTableFormatter) printTableFooter(widths []int) {
	fmt.Fprintf(f.writer, "└")
	for i, width := range widths {
		fmt.Fprint(f.writer, strings.Repeat("─", width+2))
		if i < len(widths)-1 {
			fmt.Fprintf(f.writer, "┴")
		}
	}
	fmt.Fprintf(f.writer, "┘\n")
}
