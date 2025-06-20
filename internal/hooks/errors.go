package hooks

import (
	"fmt"
	"time"
)

// HookError represents a hook execution error
type HookError struct {
	HookType HookType
	Script   string
	Err      error
}

func (e *HookError) Error() string {
	return fmt.Sprintf("hook %s failed: %v", e.HookType.String(), e.Err)
}

func (e *HookError) Unwrap() error {
	return e.Err
}

// TimeoutError represents a hook timeout error
type TimeoutError struct {
	Hook    string
	Timeout time.Duration
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("hook %s timed out after %v", e.Hook, e.Timeout)
}

// ScriptNotFoundError represents a script not found error
type ScriptNotFoundError struct {
	Script string
}

func (e *ScriptNotFoundError) Error() string {
	return fmt.Sprintf("hook script not found: %s", e.Script)
}

// ScriptPermissionError represents a script permission error
type ScriptPermissionError struct {
	Script string
	Err    error
}

func (e *ScriptPermissionError) Error() string {
	return fmt.Sprintf("insufficient permissions to execute hook script %s: %v", e.Script, e.Err)
}

func (e *ScriptPermissionError) Unwrap() error {
	return e.Err
}

// ScriptExecutionError represents a script execution error
type ScriptExecutionError struct {
	Script   string
	ExitCode int
	Stderr   string
	Err      error
}

func (e *ScriptExecutionError) Error() string {
	if e.Stderr != "" {
		return fmt.Sprintf("hook script %s failed with exit code %d: %s", e.Script, e.ExitCode, e.Stderr)
	}
	return fmt.Sprintf("hook script %s failed with exit code %d", e.Script, e.ExitCode)
}

func (e *ScriptExecutionError) Unwrap() error {
	return e.Err
}

// ValidationError represents a hook validation error
type ValidationError struct {
	Field   string
	Value   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for %s='%s': %s", e.Field, e.Value, e.Message)
}

// ConfigurationError represents a hook configuration error
type ConfigurationError struct {
	HookType HookType
	Message  string
}

func (e *ConfigurationError) Error() string {
	return fmt.Sprintf("configuration error for %s hook: %s", e.HookType.String(), e.Message)
}

// EnvironmentError represents an environment setup error
type EnvironmentError struct {
	Variable string
	Value    string
	Err      error
}

func (e *EnvironmentError) Error() string {
	return fmt.Sprintf("environment error for variable %s='%s': %v", e.Variable, e.Value, e.Err)
}

func (e *EnvironmentError) Unwrap() error {
	return e.Err
}

// isRetryableError determines if an error is retryable
func isRetryableError(err error) bool {
	switch err.(type) {
	case *TimeoutError:
		return false // Timeouts are not retryable
	case *ScriptNotFoundError:
		return false // Missing scripts are not retryable
	case *ScriptPermissionError:
		return false // Permission errors are not retryable
	case *ValidationError:
		return false // Validation errors are not retryable
	case *ConfigurationError:
		return false // Configuration errors are not retryable
	default:
		return true // Other errors might be transient
	}
}

// shouldFailSilently determines if a hook error should fail silently
func shouldFailSilently(hookType HookType, err error) bool {
	// Status hooks should generally fail silently to not interrupt the main workflow
	switch hookType {
	case HookTypeStatusIdle, HookTypeStatusBusy, HookTypeStatusWaiting:
		return true
	default:
		// Worktree hooks are more critical but still shouldn't block operations
		return false
	}
}