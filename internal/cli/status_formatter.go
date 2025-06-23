package cli

import (
	"fmt"
	"io"
	"reflect"
	"strings"
	"time"
)

// StatusTableFormatter formats status data using the comprehensive TableFormatter
type StatusTableFormatter struct {
	writer io.Writer
}

// NewStatusTableFormatter creates a new status table formatter
func NewStatusTableFormatter(writer io.Writer) *StatusTableFormatter {
	return &StatusTableFormatter{
		writer: writer,
	}
}

// Format formats the status data as properly structured tables
func (f *StatusTableFormatter) Format(data interface{}) error {
	// Use reflection to extract the fields since the types are in different packages
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return fmt.Errorf("status data is nil")
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return fmt.Errorf("invalid data type for status formatter: expected struct, got %T", data)
	}

	// Extract fields using reflection
	systemField := v.FieldByName("System")
	worktreesField := v.FieldByName("Worktrees")
	sessionsField := v.FieldByName("Sessions")
	processesField := v.FieldByName("Processes")
	hooksField := v.FieldByName("Hooks")

	// Format System Status
	if systemField.IsValid() {
		if err := f.formatSystemStatusReflection(systemField); err != nil {
			return fmt.Errorf("failed to format system status: %w", err)
		}
	}

	// Format Worktrees
	if worktreesField.IsValid() && worktreesField.Len() > 0 {
		fmt.Fprintf(f.writer, "\n")
		if err := f.formatWorktreesReflection(worktreesField); err != nil {
			return fmt.Errorf("failed to format worktrees: %w", err)
		}
	}

	// Format Sessions
	if sessionsField.IsValid() && sessionsField.Len() > 0 {
		fmt.Fprintf(f.writer, "\n")
		if err := f.formatSessionsReflection(sessionsField); err != nil {
			return fmt.Errorf("failed to format sessions: %w", err)
		}
	}

	// Format Processes
	if processesField.IsValid() && processesField.Len() > 0 {
		fmt.Fprintf(f.writer, "\n")
		if err := f.formatProcessesReflection(processesField); err != nil {
			return fmt.Errorf("failed to format processes: %w", err)
		}
	}

	// Format Hooks
	if hooksField.IsValid() {
		fmt.Fprintf(f.writer, "\n")
		if err := f.formatHooksStatusReflection(hooksField); err != nil {
			return fmt.Errorf("failed to format hooks status: %w", err)
		}
	}

	return nil
}

// formatSystemStatusReflection formats the system overview using reflection
func (f *StatusTableFormatter) formatSystemStatusReflection(systemField reflect.Value) error {
	f.printSectionHeader("System Overview")

	maxKeyWidth := 25
	data := [][]string{
		{"Overall Health", formatHealthStatus(getFieldBool(systemField, "Healthy"))},
		{"Total Worktrees", fmt.Sprintf("%d", getFieldInt(systemField, "TotalWorktrees"))},
		{"Clean Worktrees", fmt.Sprintf("%d", getFieldInt(systemField, "CleanWorktrees"))},
		{"Dirty Worktrees", fmt.Sprintf("%d", getFieldInt(systemField, "DirtyWorktrees"))},
		{"Active Sessions", fmt.Sprintf("%d", getFieldInt(systemField, "ActiveSessions"))},
		{"Total Processes", fmt.Sprintf("%d", getFieldInt(systemField, "TotalProcesses"))},
		{"Healthy Processes", fmt.Sprintf("%d", getFieldInt(systemField, "HealthyProcesses"))},
		{"Unhealthy Processes", fmt.Sprintf("%d", getFieldInt(systemField, "UnhealthyProcesses"))},
		{"Process Manager", formatBooleanStatus(getFieldBool(systemField, "ProcessManagerRunning"))},
		{"Hooks Enabled", formatBooleanStatus(getFieldBool(systemField, "HooksEnabled"))},
		{"Average Uptime", formatDuration(getFieldDuration(systemField, "AverageUptime"))},
	}

	return f.printKeyValueTable(data, maxKeyWidth)
}

// formatWorktreesReflection formats worktrees using reflection
func (f *StatusTableFormatter) formatWorktreesReflection(worktreesField reflect.Value) error {
	f.printSectionHeader("Worktrees")

	// Define column headers and widths
	headers := []string{"Path", "Branch", "Head", "Status", "Session", "Procs", "Last Accessed"}
	widths := []int{30, 20, 10, 10, 15, 6, 15}

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
			shortenPath(getFieldString(wt, "Path"), 30),
			getFieldString(wt, "Branch"),
			head,
			formatWorktreeStatus(getFieldBool(wt, "IsClean"), getFieldBool(wt, "HasUncommitted")),
			getFieldString(wt, "TmuxSession"),
			fmt.Sprintf("%d", getFieldInt(wt, "ProcessCount")),
			formatTimeAgo(getFieldTime(wt, "LastAccessed")),
		}
		f.printTableRow(row, widths)
	}

	f.printTableFooter(widths)
	return nil
}

// formatSessionsReflection formats sessions using reflection
func (f *StatusTableFormatter) formatSessionsReflection(sessionsField reflect.Value) error {
	f.printSectionHeader("Sessions")

	// Define column headers and widths
	headers := []string{"Name", "Project", "Branch", "Status", "Created", "Last Access"}
	widths := []int{20, 15, 15, 8, 12, 12}

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
			formatTimeAgo(getFieldTime(session, "Created")),
			formatTimeAgo(getFieldTime(session, "LastAccess")),
		}
		f.printTableRow(row, widths)
	}

	f.printTableFooter(widths)
	return nil
}

// formatProcessesReflection formats processes using reflection
func (f *StatusTableFormatter) formatProcessesReflection(processesField reflect.Value) error {
	f.printSectionHeader("Claude Processes")

	// Define column headers and widths
	headers := []string{"PID", "State", "Session", "Uptime", "CPU%", "Memory", "Directory"}
	widths := []int{8, 10, 15, 10, 8, 10, 25}

	// Print header
	f.printTableHeader(headers, widths)

	// Print rows
	for i := 0; i < processesField.Len(); i++ {
		proc := processesField.Index(i)
		row := []string{
			fmt.Sprintf("%d", getFieldInt(proc, "PID")),
			formatProcessState(getFieldString(proc, "State")),
			getFieldString(proc, "TmuxSession"),
			getFieldString(proc, "Uptime"),
			fmt.Sprintf("%.1f", getFieldFloat64(proc, "CPUPercent")),
			formatMemory(getFieldInt64(proc, "MemoryMB")),
			shortenPath(getFieldString(proc, "WorkingDir"), 25),
		}
		f.printTableRow(row, widths)
	}

	f.printTableFooter(widths)
	return nil
}

// formatHooksStatusReflection formats hooks status using reflection
func (f *StatusTableFormatter) formatHooksStatusReflection(hooksField reflect.Value) error {
	f.printSectionHeader("Hooks Status")

	data := [][]string{
		{"Hooks System", formatBooleanStatus(getFieldBool(hooksField, "Enabled"))},
	}

	return f.printKeyValueTable(data, 20)
}

// Helper functions for reflection access

func getFieldString(v reflect.Value, fieldName string) string {
	field := v.FieldByName(fieldName)
	if !field.IsValid() {
		return ""
	}
	return field.String()
}

func getFieldInt(v reflect.Value, fieldName string) int {
	field := v.FieldByName(fieldName)
	if !field.IsValid() {
		return 0
	}
	return int(field.Int())
}

func getFieldInt64(v reflect.Value, fieldName string) int64 {
	field := v.FieldByName(fieldName)
	if !field.IsValid() {
		return 0
	}
	return field.Int()
}

func getFieldFloat64(v reflect.Value, fieldName string) float64 {
	field := v.FieldByName(fieldName)
	if !field.IsValid() {
		return 0
	}
	return field.Float()
}

func getFieldBool(v reflect.Value, fieldName string) bool {
	field := v.FieldByName(fieldName)
	if !field.IsValid() {
		return false
	}
	return field.Bool()
}

func getFieldTime(v reflect.Value, fieldName string) time.Time {
	field := v.FieldByName(fieldName)
	if !field.IsValid() {
		return time.Time{}
	}
	if t, ok := field.Interface().(time.Time); ok {
		return t
	}
	return time.Time{}
}

func getFieldDuration(v reflect.Value, fieldName string) time.Duration {
	field := v.FieldByName(fieldName)
	if !field.IsValid() {
		return 0
	}
	if d, ok := field.Interface().(time.Duration); ok {
		return d
	}
	return 0
}

// Helper printing functions

// printSectionHeader prints a section header with decorative styling
func (f *StatusTableFormatter) printSectionHeader(title string) {
	fmt.Fprintf(f.writer, "\n┌─ %s ─", title)
	padding := 60 - len(title) - 4 // Adjust based on desired width
	if padding > 0 {
		fmt.Fprint(f.writer, strings.Repeat("─", padding))
	}
	fmt.Fprintf(f.writer, "┐\n")
}

// printKeyValueTable prints a simple key-value table
func (f *StatusTableFormatter) printKeyValueTable(data [][]string, keyWidth int) error {
	for _, row := range data {
		if len(row) >= 2 {
			fmt.Fprintf(f.writer, "│ %-*s │ %s\n", keyWidth, row[0], row[1])
		}
	}
	fmt.Fprintf(f.writer, "└")
	fmt.Fprint(f.writer, strings.Repeat("─", keyWidth+35)) // Adjust total width
	fmt.Fprintf(f.writer, "┘\n")
	return nil
}

// printTableHeader prints a table header with borders
func (f *StatusTableFormatter) printTableHeader(headers []string, widths []int) {
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
func (f *StatusTableFormatter) printTableRow(row []string, widths []int) {
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
func (f *StatusTableFormatter) printTableFooter(widths []int) {
	fmt.Fprintf(f.writer, "└")
	for i, width := range widths {
		fmt.Fprint(f.writer, strings.Repeat("─", width+2))
		if i < len(widths)-1 {
			fmt.Fprintf(f.writer, "┴")
		}
	}
	fmt.Fprintf(f.writer, "┘\n")
}

// Helper functions for formatting

// formatHealthStatus formats a boolean health status with color
func formatHealthStatus(healthy bool) string {
	if healthy {
		return "✓ Healthy"
	}
	return "✗ Unhealthy"
}

// formatBooleanStatus formats a boolean as a checkmark or X
func formatBooleanStatus(status bool) string {
	if status {
		return "✓"
	}
	return "✗"
}

// formatWorktreeStatus formats worktree status with visual indicators
func formatWorktreeStatus(isClean, hasUncommitted bool) string {
	if isClean && !hasUncommitted {
		return "✓ Clean"
	}
	if hasUncommitted {
		return "⚠ Dirty"
	}
	return "? Unknown"
}

// formatProcessState formats process state with visual indicators
func formatProcessState(state string) string {
	switch strings.ToLower(state) {
	case "idle":
		return "💤 Idle"
	case "busy":
		return "🔄 Busy"
	case "waiting":
		return "⏳ Wait"
	case "error":
		return "❌ Error"
	default:
		return state
	}
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d == 0 {
		return "0s"
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

// formatTimeAgo formats a time as "time ago"
func formatTimeAgo(t time.Time) string {
	if t.IsZero() {
		return "Never"
	}

	now := time.Now()
	diff := now.Sub(t)

	if diff < time.Minute {
		return "Just now"
	}
	if diff < time.Hour {
		return fmt.Sprintf("%dm ago", int(diff.Minutes()))
	}
	if diff < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(diff.Hours()))
	}
	return fmt.Sprintf("%dd ago", int(diff.Hours()/24))
}

// formatMemory formats memory usage in MB
func formatMemory(mb int64) string {
	if mb < 1024 {
		return fmt.Sprintf("%d MB", mb)
	}
	return fmt.Sprintf("%.1f GB", float64(mb)/1024)
}

// shortenPath shortens a path to fit within the specified length
func shortenPath(path string, maxLen int) string {
	if len(path) <= maxLen {
		return path
	}

	// Try to keep the filename and some parent directories
	parts := strings.Split(path, "/")
	if len(parts) <= 1 {
		return path[:maxLen-3] + "..."
	}

	filename := parts[len(parts)-1]
	if len(filename) > maxLen-3 {
		return filename[:maxLen-3] + "..."
	}

	// Build path from the end
	result := filename
	for i := len(parts) - 2; i >= 0; i-- {
		candidate := parts[i] + "/" + result
		if len(candidate) > maxLen-3 {
			return ".../" + result
		}
		result = candidate
	}

	return result
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
