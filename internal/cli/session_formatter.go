package cli

import (
	"fmt"
	"io"
	"reflect"
	"strings"
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
		fmt.Fprintf(f.writer, "\nTotal sessions: %d\n", int(totalField.Int()))
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
		sessionInterface := sessionsField.Index(i).Interface()
		session := reflect.ValueOf(sessionInterface)
		
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

// Helper printing functions (reused from status_formatter.go pattern)

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