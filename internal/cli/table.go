package cli

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// TableFormatter provides enhanced table formatting with styling and layout options
type TableFormatter struct {
	columns      []TableColumn
	rows         [][]string
	theme        TableTheme
	options      TableOptions
	maxWidth     int
	colorEnabled bool
}

// TableColumn defines a table column with formatting options
type TableColumn struct {
	Header    string
	Field     string
	Width     int
	MinWidth  int
	MaxWidth  int
	Alignment ColumnAlignment
	Format    ColumnFormat
	Visible   bool
	Sortable  bool
	Resizable bool
}

// ColumnAlignment defines text alignment within columns
type ColumnAlignment int

const (
	AlignLeft ColumnAlignment = iota
	AlignCenter
	AlignRight
)

// ColumnFormat defines formatting options for column data
type ColumnFormat int

const (
	FormatDefault ColumnFormat = iota
	FormatNumber
	FormatDateTime
	FormatDuration
	FormatFileSize
	FormatPercentage
	FormatBoolean
)

// TableTheme defines the visual styling for tables
type TableTheme struct {
	BorderStyle    BorderStyle
	HeaderStyle    CellStyle
	RowStyle       CellStyle
	AlternateStyle CellStyle
	SelectedStyle  CellStyle
	BorderColor    string
	HeaderColor    string
	DataColor      string
	AlternateColor string
	SelectedColor  string
}

// BorderStyle defines table border appearance
type BorderStyle struct {
	TopLeft     string
	TopRight    string
	BottomLeft  string
	BottomRight string
	Horizontal  string
	Vertical    string
	Cross       string
	TopTee      string
	BottomTee   string
	LeftTee     string
	RightTee    string
}

// CellStyle defines cell text styling
type CellStyle struct {
	Bold      bool
	Italic    bool
	Underline bool
	Color     string
	BgColor   string
}

// TableOptions configures table behavior and appearance
type TableOptions struct {
	ShowHeader     bool
	ShowBorder     bool
	ShowRowNumbers bool
	AlternateRows  bool
	CompactMode    bool
	WrapText       bool
	TruncateData   bool
	MaxCellWidth   int
	Padding        int
	SortColumn     string
	SortDescending bool
	FilterColumn   string
	FilterValue    string
}

// DefaultTableTheme returns a default table theme
func DefaultTableTheme() TableTheme {
	return TableTheme{
		BorderStyle: BorderStyle{
			TopLeft:     "┌",
			TopRight:    "┐",
			BottomLeft:  "└",
			BottomRight: "┘",
			Horizontal:  "─",
			Vertical:    "│",
			Cross:       "┼",
			TopTee:      "┬",
			BottomTee:   "┴",
			LeftTee:     "├",
			RightTee:    "┤",
		},
		HeaderStyle: CellStyle{
			Bold:  true,
			Color: "\033[1;36m", // Bold cyan
		},
		RowStyle: CellStyle{
			Color: "\033[0m", // Default
		},
		AlternateStyle: CellStyle{
			BgColor: "\033[48;5;236m", // Dark gray background
		},
		BorderColor: "\033[90m", // Dark gray
		DataColor:   "\033[0m",  // Default
	}
}

// CompactTableTheme returns a compact table theme without borders
func CompactTableTheme() TableTheme {
	theme := DefaultTableTheme()
	theme.BorderStyle = BorderStyle{
		Horizontal: " ",
		Vertical:   " ",
	}
	return theme
}

// NewTableFormatter creates a new table formatter with options
func NewTableFormatter(opts *TableOptions) *TableFormatter {
	if opts == nil {
		opts = &TableOptions{
			ShowHeader:    true,
			ShowBorder:    true,
			AlternateRows: false,
			Padding:       1,
			MaxCellWidth:  50,
		}
	}

	return &TableFormatter{
		columns:      make([]TableColumn, 0),
		rows:         make([][]string, 0),
		theme:        DefaultTableTheme(),
		options:      *opts,
		maxWidth:     120, // Default terminal width
		colorEnabled: true,
	}
}

// AddColumn adds a column to the table
func (tf *TableFormatter) AddColumn(column TableColumn) {
	if column.Width == 0 && column.MinWidth == 0 {
		column.MinWidth = len(column.Header)
	}
	if column.MaxWidth == 0 {
		column.MaxWidth = tf.options.MaxCellWidth
	}
	column.Visible = true
	tf.columns = append(tf.columns, column)
}

// AddSimpleColumn adds a simple column with just header and field
func (tf *TableFormatter) AddSimpleColumn(header, field string) {
	tf.AddColumn(TableColumn{
		Header:    header,
		Field:     field,
		Alignment: AlignLeft,
		Format:    FormatDefault,
	})
}

// SetData sets the table data from a slice of structs
func (tf *TableFormatter) SetData(data interface{}) error {
	tf.rows = make([][]string, 0)

	v := reflect.ValueOf(data)
	if v.Kind() != reflect.Slice {
		return NewError("data must be a slice")
	}

	for i := 0; i < v.Len(); i++ {
		item := v.Index(i)
		row, err := tf.extractRowData(item)
		if err != nil {
			return err
		}
		tf.rows = append(tf.rows, row)
	}

	return nil
}

// extractRowData extracts row data from a struct value
func (tf *TableFormatter) extractRowData(item reflect.Value) ([]string, error) {
	row := make([]string, len(tf.columns))

	// Handle pointer types
	if item.Kind() == reflect.Ptr {
		if item.IsNil() {
			return row, nil
		}
		item = item.Elem()
	}

	if item.Kind() != reflect.Struct {
		return nil, NewError("data items must be structs")
	}

	itemType := item.Type()

	for i, column := range tf.columns {
		var value string

		// Find field by name
		_, found := itemType.FieldByName(column.Field)
		if !found {
			value = ""
		} else {
			fieldValue := item.FieldByName(column.Field)
			value = tf.formatCellValue(fieldValue, column.Format)
		}

		row[i] = value
	}

	return row, nil
}

// formatCellValue formats a cell value based on the column format
func (tf *TableFormatter) formatCellValue(value reflect.Value, format ColumnFormat) string {
	if !value.IsValid() {
		return ""
	}

	switch format {
	case FormatDateTime:
		if t, ok := value.Interface().(time.Time); ok {
			return t.Format("2006-01-02 15:04:05")
		}
	case FormatDuration:
		if d, ok := value.Interface().(time.Duration); ok {
			return d.String()
		}
	case FormatFileSize:
		if size, ok := value.Interface().(int64); ok {
			return formatFileSize(size)
		}
	case FormatPercentage:
		if f, ok := value.Interface().(float64); ok {
			return fmt.Sprintf("%.1f%%", f)
		}
	case FormatBoolean:
		if b, ok := value.Interface().(bool); ok {
			if b {
				return "✓"
			}
			return "✗"
		}
	case FormatNumber:
		if n, ok := value.Interface().(int); ok {
			return strconv.Itoa(n)
		}
		if n, ok := value.Interface().(int64); ok {
			return strconv.FormatInt(n, 10)
		}
		if f, ok := value.Interface().(float64); ok {
			return strconv.FormatFloat(f, 'f', 2, 64)
		}
	}

	return fmt.Sprintf("%v", value.Interface())
}

// Render renders the table as a string
func (tf *TableFormatter) Render() string {
	if len(tf.columns) == 0 {
		return "No columns defined"
	}

	if len(tf.rows) == 0 {
		return "No data available"
	}

	// Calculate column widths
	tf.calculateColumnWidths()

	var result strings.Builder

	// Render header
	if tf.options.ShowHeader {
		if tf.options.ShowBorder {
			result.WriteString(tf.renderTopBorder())
			result.WriteString("\n")
		}
		result.WriteString(tf.renderHeaderRow())
		result.WriteString("\n")
		if tf.options.ShowBorder {
			result.WriteString(tf.renderSeparatorBorder())
			result.WriteString("\n")
		}
	}

	// Render data rows
	for i, row := range tf.rows {
		result.WriteString(tf.renderDataRow(row, i))
		result.WriteString("\n")
	}

	// Render bottom border
	if tf.options.ShowBorder {
		result.WriteString(tf.renderBottomBorder())
	}

	return result.String()
}

// calculateColumnWidths calculates optimal column widths
func (tf *TableFormatter) calculateColumnWidths() {
	for i := range tf.columns {
		column := &tf.columns[i]

		// Start with header width
		maxWidth := utf8.RuneCountInString(column.Header)

		// Check data width
		for _, row := range tf.rows {
			if i < len(row) {
				cellWidth := utf8.RuneCountInString(row[i])
				if cellWidth > maxWidth {
					maxWidth = cellWidth
				}
			}
		}

		// Apply constraints
		if column.MinWidth > 0 && maxWidth < column.MinWidth {
			maxWidth = column.MinWidth
		}
		if column.MaxWidth > 0 && maxWidth > column.MaxWidth {
			maxWidth = column.MaxWidth
		}

		column.Width = maxWidth
	}
}

// renderTopBorder renders the top border of the table
func (tf *TableFormatter) renderTopBorder() string {
	var result strings.Builder
	border := tf.theme.BorderStyle

	if tf.colorEnabled {
		result.WriteString(tf.theme.BorderColor)
	}

	result.WriteString(border.TopLeft)

	for i, column := range tf.columns {
		if !column.Visible {
			continue
		}

		result.WriteString(strings.Repeat(border.Horizontal, column.Width+2*tf.options.Padding))

		if i < len(tf.columns)-1 {
			result.WriteString(border.TopTee)
		}
	}

	result.WriteString(border.TopRight)

	if tf.colorEnabled {
		result.WriteString("\033[0m") // Reset color
	}

	return result.String()
}

// renderBottomBorder renders the bottom border of the table
func (tf *TableFormatter) renderBottomBorder() string {
	var result strings.Builder
	border := tf.theme.BorderStyle

	if tf.colorEnabled {
		result.WriteString(tf.theme.BorderColor)
	}

	result.WriteString(border.BottomLeft)

	for i, column := range tf.columns {
		if !column.Visible {
			continue
		}

		result.WriteString(strings.Repeat(border.Horizontal, column.Width+2*tf.options.Padding))

		if i < len(tf.columns)-1 {
			result.WriteString(border.BottomTee)
		}
	}

	result.WriteString(border.BottomRight)

	if tf.colorEnabled {
		result.WriteString("\033[0m") // Reset color
	}

	return result.String()
}

// renderSeparatorBorder renders a separator border between header and data
func (tf *TableFormatter) renderSeparatorBorder() string {
	var result strings.Builder
	border := tf.theme.BorderStyle

	if tf.colorEnabled {
		result.WriteString(tf.theme.BorderColor)
	}

	result.WriteString(border.LeftTee)

	for i, column := range tf.columns {
		if !column.Visible {
			continue
		}

		result.WriteString(strings.Repeat(border.Horizontal, column.Width+2*tf.options.Padding))

		if i < len(tf.columns)-1 {
			result.WriteString(border.Cross)
		}
	}

	result.WriteString(border.RightTee)

	if tf.colorEnabled {
		result.WriteString("\033[0m") // Reset color
	}

	return result.String()
}

// renderHeaderRow renders the header row
func (tf *TableFormatter) renderHeaderRow() string {
	var result strings.Builder
	border := tf.theme.BorderStyle

	if tf.options.ShowBorder {
		if tf.colorEnabled {
			result.WriteString(tf.theme.BorderColor)
		}
		result.WriteString(border.Vertical)
		if tf.colorEnabled {
			result.WriteString("\033[0m")
		}
	}

	for _, column := range tf.columns {
		if !column.Visible {
			continue
		}

		// Apply header styling
		if tf.colorEnabled {
			result.WriteString(tf.theme.HeaderStyle.Color)
			if tf.theme.HeaderStyle.Bold {
				result.WriteString("\033[1m")
			}
		}

		padding := strings.Repeat(" ", tf.options.Padding)
		result.WriteString(padding)

		// Format header text
		headerText := tf.alignText(column.Header, column.Width, column.Alignment)
		result.WriteString(headerText)

		result.WriteString(padding)

		if tf.colorEnabled {
			result.WriteString("\033[0m") // Reset color
		}

		// Add column separator
		if tf.options.ShowBorder {
			if tf.colorEnabled {
				result.WriteString(tf.theme.BorderColor)
			}
			result.WriteString(border.Vertical)
			if tf.colorEnabled {
				result.WriteString("\033[0m")
			}
		}
	}

	return result.String()
}

// renderDataRow renders a data row
func (tf *TableFormatter) renderDataRow(row []string, rowIndex int) string {
	var result strings.Builder
	border := tf.theme.BorderStyle

	if tf.options.ShowBorder {
		if tf.colorEnabled {
			result.WriteString(tf.theme.BorderColor)
		}
		result.WriteString(border.Vertical)
		if tf.colorEnabled {
			result.WriteString("\033[0m")
		}
	}

	// Determine row style
	style := tf.theme.RowStyle
	if tf.options.AlternateRows && rowIndex%2 == 1 {
		style = tf.theme.AlternateStyle
	}

	for i, column := range tf.columns {
		if !column.Visible {
			continue
		}

		// Apply cell styling
		if tf.colorEnabled {
			result.WriteString(style.Color)
			if style.BgColor != "" {
				result.WriteString(style.BgColor)
			}
		}

		padding := strings.Repeat(" ", tf.options.Padding)
		result.WriteString(padding)

		// Get cell value
		cellValue := ""
		if i < len(row) {
			cellValue = row[i]
		}

		// Format cell text
		cellText := tf.alignText(cellValue, column.Width, column.Alignment)
		result.WriteString(cellText)

		result.WriteString(padding)

		if tf.colorEnabled {
			result.WriteString("\033[0m") // Reset color
		}

		// Add column separator
		if tf.options.ShowBorder {
			if tf.colorEnabled {
				result.WriteString(tf.theme.BorderColor)
			}
			result.WriteString(border.Vertical)
			if tf.colorEnabled {
				result.WriteString("\033[0m")
			}
		}
	}

	return result.String()
}

// alignText aligns text within a specified width
func (tf *TableFormatter) alignText(text string, width int, alignment ColumnAlignment) string {
	textLen := utf8.RuneCountInString(text)

	// Truncate if necessary
	if textLen > width {
		if tf.options.TruncateData {
			// Truncate with ellipsis
			if width > 3 {
				runes := []rune(text)
				text = string(runes[:width-3]) + "..."
			} else {
				text = strings.Repeat(".", width)
			}
			textLen = width
		} else {
			// Keep original text (will overflow)
		}
	}

	if textLen >= width {
		return text
	}

	padding := width - textLen

	switch alignment {
	case AlignLeft:
		return text + strings.Repeat(" ", padding)
	case AlignRight:
		return strings.Repeat(" ", padding) + text
	case AlignCenter:
		leftPad := padding / 2
		rightPad := padding - leftPad
		return strings.Repeat(" ", leftPad) + text + strings.Repeat(" ", rightPad)
	default:
		return text + strings.Repeat(" ", padding)
	}
}

// SetMaxWidth sets the maximum table width
func (tf *TableFormatter) SetMaxWidth(width int) {
	tf.maxWidth = width
}

// SetColorEnabled enables or disables color output
func (tf *TableFormatter) SetColorEnabled(enabled bool) {
	tf.colorEnabled = enabled
}

// SetTheme sets the table theme
func (tf *TableFormatter) SetTheme(theme TableTheme) {
	tf.theme = theme
}

// formatFileSize formats a file size in bytes to human-readable format
func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	units := []string{"KB", "MB", "GB", "TB", "PB"}
	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), units[exp])
}

// RenderCompact renders the table in compact mode without borders
func (tf *TableFormatter) RenderCompact() string {
	originalShowBorder := tf.options.ShowBorder
	originalTheme := tf.theme

	tf.options.ShowBorder = false
	tf.theme = CompactTableTheme()

	defer func() {
		tf.options.ShowBorder = originalShowBorder
		tf.theme = originalTheme
	}()

	return tf.Render()
}
