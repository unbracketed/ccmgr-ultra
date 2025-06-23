package cli

import (
	"fmt"
	"os"
	"strings"
)

// ExitCode represents standard CLI exit codes
type ExitCode int

const (
	ExitSuccess ExitCode = 0
	ExitError   ExitCode = 1
	ExitUsage   ExitCode = 2
	ExitConfig  ExitCode = 3
	ExitTimeout ExitCode = 4
)

// CLIError represents a CLI-specific error with additional context
type CLIError struct {
	Message    string
	Suggestion string
	ExitCode   ExitCode
	Cause      error
}

func (e *CLIError) Error() string {
	var parts []string

	if e.Message != "" {
		parts = append(parts, e.Message)
	}

	if e.Cause != nil {
		parts = append(parts, fmt.Sprintf("cause: %v", e.Cause))
	}

	return strings.Join(parts, ": ")
}

// NewError creates a new CLI error
func NewError(message string) *CLIError {
	return &CLIError{
		Message:  message,
		ExitCode: ExitError,
	}
}

// NewErrorWithSuggestion creates a new CLI error with an actionable suggestion
func NewErrorWithSuggestion(message, suggestion string) *CLIError {
	return &CLIError{
		Message:    message,
		Suggestion: suggestion,
		ExitCode:   ExitError,
	}
}

// NewErrorWithCause creates a new CLI error wrapping an underlying error
func NewErrorWithCause(message string, cause error) *CLIError {
	return &CLIError{
		Message:  message,
		Cause:    cause,
		ExitCode: ExitError,
	}
}

// WithSuggestion adds a suggestion to an existing error
func (e *CLIError) WithSuggestion(suggestion string) *CLIError {
	e.Suggestion = suggestion
	return e
}

// WithExitCode sets the exit code for an error
func (e *CLIError) WithExitCode(code ExitCode) *CLIError {
	e.ExitCode = code
	return e
}

// HandleCLIError processes a CLI error and provides consistent error output
func HandleCLIError(err error) error {
	if err == nil {
		return nil
	}

	var cliErr *CLIError
	switch e := err.(type) {
	case *CLIError:
		cliErr = e
	default:
		cliErr = NewErrorWithCause("command failed", err)
	}

	// Print error message to stderr
	fmt.Fprintf(os.Stderr, "Error: %s\n", cliErr.Message)

	if cliErr.Cause != nil && cliErr.Cause.Error() != cliErr.Message {
		fmt.Fprintf(os.Stderr, "Cause: %v\n", cliErr.Cause)
	}

	if cliErr.Suggestion != "" {
		fmt.Fprintf(os.Stderr, "Suggestion: %s\n", cliErr.Suggestion)
	}

	return cliErr
}

// ExitWithError handles an error and exits with the appropriate code
func ExitWithError(err error) {
	if err == nil {
		os.Exit(int(ExitSuccess))
		return
	}

	HandleCLIError(err)

	if cliErr, ok := err.(*CLIError); ok {
		os.Exit(int(cliErr.ExitCode))
	}

	os.Exit(int(ExitError))
}

// Common error helpers

// ErrorConfigNotFound creates a standard configuration not found error
func ErrorConfigNotFound(path string) *CLIError {
	return NewErrorWithSuggestion(
		fmt.Sprintf("configuration file not found: %s", path),
		"Run 'ccmgr-ultra init' to create a new configuration or specify a different config file with --config",
	).WithExitCode(ExitConfig)
}

// ErrorInvalidWorktree creates a standard invalid worktree error
func ErrorInvalidWorktree(name string) *CLIError {
	return NewErrorWithSuggestion(
		fmt.Sprintf("worktree '%s' not found", name),
		"Use 'ccmgr-ultra worktree list' to see available worktrees",
	)
}

// ErrorNotInRepository creates a standard "not in repository" error
func ErrorNotInRepository() *CLIError {
	return NewErrorWithSuggestion(
		"not in a git repository",
		"Run this command from within a git repository or use 'ccmgr-ultra init' to create one",
	)
}

// ErrorPermissionDenied creates a standard permission denied error
func ErrorPermissionDenied(resource string) *CLIError {
	return NewErrorWithSuggestion(
		fmt.Sprintf("permission denied accessing: %s", resource),
		"Check file permissions or run with appropriate privileges",
	)
}
