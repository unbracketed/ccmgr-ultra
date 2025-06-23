package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ValidateWorktreeName validates a worktree name according to git standards
func ValidateWorktreeName(name string) error {
	if name == "" {
		return NewError("worktree name cannot be empty")
	}

	// Git worktree names should not contain certain characters
	invalidChars := []string{" ", "\t", "\n", "\r", ":", "?", "*", "[", "]", "\\"}
	for _, char := range invalidChars {
		if strings.Contains(name, char) {
			return NewErrorWithSuggestion(
				fmt.Sprintf("worktree name contains invalid character: '%s'", char),
				"Use only alphanumeric characters, hyphens, and underscores",
			)
		}
	}

	// Check for reserved names
	reservedNames := []string{".", "..", "HEAD", "refs", "objects", "hooks"}
	for _, reserved := range reservedNames {
		if strings.EqualFold(name, reserved) {
			return NewErrorWithSuggestion(
				fmt.Sprintf("'%s' is a reserved name", name),
				"Choose a different name for your worktree",
			)
		}
	}

	return nil
}

// ValidateSessionName validates a tmux session name
func ValidateSessionName(name string) error {
	if name == "" {
		return NewError("session name cannot be empty")
	}

	// tmux session names have specific requirements
	// They cannot contain certain characters
	if strings.Contains(name, ":") {
		return NewErrorWithSuggestion(
			"session name cannot contain ':'",
			"Use hyphens or underscores instead",
		)
	}

	if strings.Contains(name, ".") && (strings.HasPrefix(name, ".") || strings.HasSuffix(name, ".")) {
		return NewErrorWithSuggestion(
			"session name cannot start or end with '.'",
			"Ensure dots are only used within the name",
		)
	}

	return nil
}

// ValidateBranchName validates a git branch name
func ValidateBranchName(name string) error {
	if name == "" {
		return NewError("branch name cannot be empty")
	}

	// Git branch naming rules
	if strings.HasPrefix(name, "-") {
		return NewError("branch name cannot start with '-'")
	}

	if strings.Contains(name, "..") {
		return NewError("branch name cannot contain '..'")
	}

	if strings.HasSuffix(name, "/") {
		return NewError("branch name cannot end with '/'")
	}

	if strings.HasSuffix(name, ".lock") {
		return NewError("branch name cannot end with '.lock'")
	}

	// Check for control characters and special chars
	controlChars := regexp.MustCompile(`[\x00-\x1f\x7f~^:?*[\]\\]`)
	if controlChars.MatchString(name) {
		return NewErrorWithSuggestion(
			"branch name contains invalid characters",
			"Use only alphanumeric characters, hyphens, underscores, and forward slashes",
		)
	}

	return nil
}

// ValidateFilePath validates that a file path exists and is accessible
func ValidateFilePath(path string) error {
	if path == "" {
		return NewError("file path cannot be empty")
	}

	// Expand home directory if needed
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return NewErrorWithCause("failed to get user home directory", err)
		}
		path = filepath.Join(home, path[2:])
	}

	// Check if file exists
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return NewErrorWithSuggestion(
				fmt.Sprintf("file does not exist: %s", path),
				"Check the file path and ensure the file exists",
			)
		}
		return NewErrorWithCause(fmt.Sprintf("cannot access file: %s", path), err)
	}

	return nil
}

// ValidateDirectoryPath validates that a directory path exists and is accessible
func ValidateDirectoryPath(path string) error {
	if path == "" {
		return NewError("directory path cannot be empty")
	}

	// Expand home directory if needed
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return NewErrorWithCause("failed to get user home directory", err)
		}
		path = filepath.Join(home, path[2:])
	}

	// Check if directory exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewErrorWithSuggestion(
				fmt.Sprintf("directory does not exist: %s", path),
				"Check the directory path and ensure it exists",
			)
		}
		return NewErrorWithCause(fmt.Sprintf("cannot access directory: %s", path), err)
	}

	if !info.IsDir() {
		return NewError(fmt.Sprintf("path is not a directory: %s", path))
	}

	return nil
}

// ValidateProjectName validates a project name for initialization
func ValidateProjectName(name string) error {
	if name == "" {
		return NewError("project name cannot be empty")
	}

	// Basic project name validation
	if len(name) > 100 {
		return NewError("project name is too long (max 100 characters)")
	}

	// Check for invalid characters in project names
	validName := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !validName.MatchString(name) {
		return NewErrorWithSuggestion(
			"project name contains invalid characters",
			"Use only alphanumeric characters, hyphens, and underscores",
		)
	}

	return nil
}

// ValidateOutputFormat validates that the output format is supported
func ValidateOutputFormat(format string) error {
	_, err := ValidateFormat(format)
	return err
}

// ValidatePositiveInteger validates that a string represents a positive integer
func ValidatePositiveInteger(value string, fieldName string) error {
	if value == "" {
		return NewError(fmt.Sprintf("%s cannot be empty", fieldName))
	}

	// Simple validation - could be enhanced with actual parsing
	validInt := regexp.MustCompile(`^\d+$`)
	if !validInt.MatchString(value) {
		return NewError(fmt.Sprintf("%s must be a positive integer", fieldName))
	}

	if value == "0" {
		return NewError(fmt.Sprintf("%s must be greater than 0", fieldName))
	}

	return nil
}
