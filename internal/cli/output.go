package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"
)

// OutputFormatter defines the interface for formatting output data
type OutputFormatter interface {
	Format(data interface{}) error
}

// OutputFormat represents the supported output formats
type OutputFormat string

const (
	FormatTable OutputFormat = "table"
	FormatJSON  OutputFormat = "json"
	FormatYAML  OutputFormat = "yaml"
)

// NewFormatter creates a new formatter based on the specified format
func NewFormatter(format OutputFormat, writer io.Writer) OutputFormatter {
	if writer == nil {
		writer = os.Stdout
	}

	switch format {
	case FormatJSON:
		return &JSONFormatter{writer: writer}
	case FormatYAML:
		return &YAMLFormatter{writer: writer}
	default:
		return &SimpleTableFormatter{writer: writer}
	}
}

// NewStatusFormatter creates a new formatter specifically for status data
func NewStatusFormatter(format OutputFormat, writer io.Writer) OutputFormatter {
	if writer == nil {
		writer = os.Stdout
	}

	switch format {
	case FormatJSON:
		return &JSONFormatter{writer: writer}
	case FormatYAML:
		return &YAMLFormatter{writer: writer}
	case FormatTable:
		// Use the sophisticated TableFormatter for status data
		return NewStatusTableFormatter(writer)
	default:
		// Fallback to simple formatter for unknown formats
		return &SimpleTableFormatter{writer: writer}
	}
}

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

// SimpleTableFormatter formats output as a simple table (for backward compatibility)
type SimpleTableFormatter struct {
	writer io.Writer
}

func (f *SimpleTableFormatter) Format(data interface{}) error {
	if data == nil {
		return nil
	}

	// Handle slice/array data
	if reflect.TypeOf(data).Kind() == reflect.Slice {
		return f.formatSlice(data)
	}

	// Handle single items
	return f.formatSingle(data)
}

func (f *SimpleTableFormatter) formatSlice(data interface{}) error {
	v := reflect.ValueOf(data)
	if v.Len() == 0 {
		fmt.Fprintln(f.writer, "No data available")
		return nil
	}

	// For now, print each item on a separate line
	// This can be enhanced to create proper table formatting
	for i := 0; i < v.Len(); i++ {
		item := v.Index(i).Interface()
		if err := f.formatSingle(item); err != nil {
			return err
		}
		if i < v.Len()-1 {
			fmt.Fprintln(f.writer)
		}
	}
	return nil
}

func (f *SimpleTableFormatter) formatSingle(data interface{}) error {
	// Simple key-value formatting for structs
	v := reflect.ValueOf(data)
	t := reflect.TypeOf(data)

	if t.Kind() == reflect.Ptr {
		if v.IsNil() {
			fmt.Fprintln(f.writer, "nil")
			return nil
		}
		v = v.Elem()
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			field := t.Field(i)
			value := v.Field(i)

			// Skip unexported fields
			if !field.IsExported() {
				continue
			}

			fmt.Fprintf(f.writer, "%-20s: %v\n", field.Name, value.Interface())
		}
	default:
		fmt.Fprintln(f.writer, data)
	}
	return nil
}

// JSONFormatter formats output as JSON
type JSONFormatter struct {
	writer io.Writer
}

func (f *JSONFormatter) Format(data interface{}) error {
	encoder := json.NewEncoder(f.writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// YAMLFormatter formats output as YAML
type YAMLFormatter struct {
	writer io.Writer
}

func (f *YAMLFormatter) Format(data interface{}) error {
	encoder := yaml.NewEncoder(f.writer)
	defer encoder.Close()
	return encoder.Encode(data)
}

// ValidateFormat checks if the given format string is valid
func ValidateFormat(format string) (OutputFormat, error) {
	switch strings.ToLower(format) {
	case "table", "t":
		return FormatTable, nil
	case "json", "j":
		return FormatJSON, nil
	case "yaml", "yml", "y":
		return FormatYAML, nil
	default:
		return FormatTable, fmt.Errorf("unsupported output format: %s (supported: table, json, yaml)", format)
	}
}
