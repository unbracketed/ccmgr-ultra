package hooks

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/unbracketed/ccmgr-ultra/internal/config"
)

// DefaultExecutor implements the HookExecutor interface
type DefaultExecutor struct {
	config        *config.Config
	mu            sync.RWMutex
	activeHooks   map[string]context.CancelFunc
	maxConcurrent int
	semaphore     chan struct{}
}

// NewDefaultExecutor creates a new default hook executor
func NewDefaultExecutor(cfg *config.Config) *DefaultExecutor {
	maxConcurrent := 5 // Default max concurrent hooks
	return &DefaultExecutor{
		config:        cfg,
		activeHooks:   make(map[string]context.CancelFunc),
		maxConcurrent: maxConcurrent,
		semaphore:     make(chan struct{}, maxConcurrent),
	}
}

// Execute executes a hook synchronously
func (e *DefaultExecutor) Execute(ctx context.Context, hookType HookType, hookCtx HookContext) error {
	hook, err := e.getHookConfig(hookType)
	if err != nil {
		return err
	}

	if !hook.Enabled {
		return nil // Hook is disabled
	}

	// Acquire semaphore for concurrency control
	select {
	case e.semaphore <- struct{}{}:
		defer func() { <-e.semaphore }()
	case <-ctx.Done():
		return ctx.Err()
	}

	return e.executeHook(ctx, hook, hookCtx)
}

// ExecuteAsync executes a hook asynchronously
func (e *DefaultExecutor) ExecuteAsync(hookType HookType, hookCtx HookContext) <-chan error {
	errChan := make(chan error, 1)

	go func() {
		defer close(errChan)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		err := e.Execute(ctx, hookType, hookCtx)
		if err != nil && !shouldFailSilently(hookType, err) {
			errChan <- err
		}
	}()

	return errChan
}

// ExecuteStatusHook executes a status hook
func (e *DefaultExecutor) ExecuteStatusHook(hookType HookType, hookCtx HookContext) error {
	if !e.config.StatusHooks.Enabled {
		return nil
	}

	var hookConfig config.HookConfig
	switch hookType {
	case HookTypeStatusIdle:
		hookConfig = e.config.StatusHooks.IdleHook
	case HookTypeStatusBusy:
		hookConfig = e.config.StatusHooks.BusyHook
	case HookTypeStatusWaiting:
		hookConfig = e.config.StatusHooks.WaitingHook
	default:
		return fmt.Errorf("unsupported status hook type: %s", hookType.String())
	}

	if !hookConfig.Enabled {
		return nil
	}

	hook := Hook{
		Type:    hookType,
		Enabled: hookConfig.Enabled,
		Script:  hookConfig.Script,
		Timeout: time.Duration(hookConfig.Timeout) * time.Second,
		Async:   hookConfig.Async,
	}

	if hook.Async {
		errChan := e.ExecuteAsync(hookType, hookCtx)
		go func() {
			if err := <-errChan; err != nil {
				log.Printf("Status hook %s failed: %v", hookType.String(), err)
			}
		}()
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), hook.Timeout)
	defer cancel()

	return e.executeHook(ctx, hook, hookCtx)
}

// ExecuteWorktreeCreationHook executes a worktree creation hook
func (e *DefaultExecutor) ExecuteWorktreeCreationHook(hookCtx HookContext) error {
	if !e.config.WorktreeHooks.Enabled || !e.config.WorktreeHooks.CreationHook.Enabled {
		return nil
	}

	hookConfig := e.config.WorktreeHooks.CreationHook
	hook := Hook{
		Type:    HookTypeWorktreeCreation,
		Enabled: hookConfig.Enabled,
		Script:  hookConfig.Script,
		Timeout: time.Duration(hookConfig.Timeout) * time.Second,
		Async:   hookConfig.Async,
	}

	if hook.Async {
		errChan := e.ExecuteAsync(HookTypeWorktreeCreation, hookCtx)
		go func() {
			if err := <-errChan; err != nil {
				log.Printf("Worktree creation hook failed: %v", err)
			}
		}()
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), hook.Timeout)
	defer cancel()

	return e.executeHook(ctx, hook, hookCtx)
}

// ExecuteWorktreeActivationHook executes a worktree activation hook
func (e *DefaultExecutor) ExecuteWorktreeActivationHook(hookCtx HookContext) error {
	if !e.config.WorktreeHooks.Enabled || !e.config.WorktreeHooks.ActivationHook.Enabled {
		return nil
	}

	hookConfig := e.config.WorktreeHooks.ActivationHook
	hook := Hook{
		Type:    HookTypeWorktreeActivation,
		Enabled: hookConfig.Enabled,
		Script:  hookConfig.Script,
		Timeout: time.Duration(hookConfig.Timeout) * time.Second,
		Async:   hookConfig.Async,
	}

	if hook.Async {
		errChan := e.ExecuteAsync(HookTypeWorktreeActivation, hookCtx)
		go func() {
			if err := <-errChan; err != nil {
				log.Printf("Worktree activation hook failed: %v", err)
			}
		}()
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), hook.Timeout)
	defer cancel()

	return e.executeHook(ctx, hook, hookCtx)
}

// executeHook executes a single hook
func (e *DefaultExecutor) executeHook(ctx context.Context, hook Hook, hookCtx HookContext) error {
	// Expand script path
	scriptPath := expandPath(hook.Script)

	// Validate script exists and is executable
	if err := e.validateScript(scriptPath); err != nil {
		return err
	}

	// Build environment
	env := e.buildEnvironment(hook.Type, hookCtx)

	// Determine shell
	shell := e.getShell()

	// Create command
	var cmd *exec.Cmd
	if shell != "" {
		cmd = exec.CommandContext(ctx, shell, scriptPath)
	} else {
		cmd = exec.CommandContext(ctx, scriptPath)
	}

	// Set working directory to worktree path if available
	if hookCtx.WorktreePath != "" && isValidPath(hookCtx.WorktreePath) {
		cmd.Dir = hookCtx.WorktreePath
	}

	// Set environment
	cmd.Env = env

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute command
	startTime := time.Now()
	err := cmd.Run()
	duration := time.Since(startTime)

	// Handle result
	result := HookResult{
		HookType:  hook.Type,
		Success:   err == nil,
		Duration:  duration,
		Output:    stdout.String(),
		Timestamp: startTime,
	}

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
		}

		hookErr := &HookError{
			HookType: hook.Type,
			Script:   scriptPath,
			Err:      err,
		}

		// Check for specific error types
		if ctx.Err() == context.DeadlineExceeded {
			return &TimeoutError{
				Hook:    scriptPath,
				Timeout: hook.Timeout,
			}
		}

		if result.ExitCode != 0 {
			return &ScriptExecutionError{
				Script:   scriptPath,
				ExitCode: result.ExitCode,
				Stderr:   stderr.String(),
				Err:      err,
			}
		}

		return hookErr
	}

	return nil
}

// getHookConfig gets the hook configuration for a given hook type
func (e *DefaultExecutor) getHookConfig(hookType HookType) (Hook, error) {
	switch hookType {
	case HookTypeStatusIdle:
		hc := e.config.StatusHooks.IdleHook
		return Hook{
			Type:    hookType,
			Enabled: hc.Enabled,
			Script:  hc.Script,
			Timeout: time.Duration(hc.Timeout) * time.Second,
			Async:   hc.Async,
		}, nil
	case HookTypeStatusBusy:
		hc := e.config.StatusHooks.BusyHook
		return Hook{
			Type:    hookType,
			Enabled: hc.Enabled,
			Script:  hc.Script,
			Timeout: time.Duration(hc.Timeout) * time.Second,
			Async:   hc.Async,
		}, nil
	case HookTypeStatusWaiting:
		hc := e.config.StatusHooks.WaitingHook
		return Hook{
			Type:    hookType,
			Enabled: hc.Enabled,
			Script:  hc.Script,
			Timeout: time.Duration(hc.Timeout) * time.Second,
			Async:   hc.Async,
		}, nil
	case HookTypeWorktreeCreation:
		hc := e.config.WorktreeHooks.CreationHook
		return Hook{
			Type:    hookType,
			Enabled: hc.Enabled,
			Script:  hc.Script,
			Timeout: time.Duration(hc.Timeout) * time.Second,
			Async:   hc.Async,
		}, nil
	case HookTypeWorktreeActivation:
		hc := e.config.WorktreeHooks.ActivationHook
		return Hook{
			Type:    hookType,
			Enabled: hc.Enabled,
			Script:  hc.Script,
			Timeout: time.Duration(hc.Timeout) * time.Second,
			Async:   hc.Async,
		}, nil
	default:
		return Hook{}, fmt.Errorf("unsupported hook type: %s", hookType.String())
	}
}

// buildEnvironment builds the environment for hook execution
func (e *DefaultExecutor) buildEnvironment(hookType HookType, hookCtx HookContext) []string {
	builder := NewEnvironmentBuilder()

	switch hookType {
	case HookTypeStatusIdle, HookTypeStatusBusy, HookTypeStatusWaiting:
		builder.WithStatusHookVars(hookType, hookCtx)
	case HookTypeWorktreeCreation:
		builder.WithWorktreeCreationVars(hookCtx)
	case HookTypeWorktreeActivation:
		builder.WithWorktreeActivationVars(hookCtx)
	default:
		builder.WithContext(hookCtx)
	}

	return builder.Build()
}

// validateScript validates that the script exists and is executable
func (e *DefaultExecutor) validateScript(scriptPath string) error {
	// Check if script exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return &ScriptNotFoundError{Script: scriptPath}
	}

	// Check if script is executable
	if err := checkExecutable(scriptPath); err != nil {
		return &ScriptPermissionError{Script: scriptPath, Err: err}
	}

	return nil
}

// getShell determines the shell to use for script execution
func (e *DefaultExecutor) getShell() string {
	// Try to detect shell from script shebang or use system default
	if shell := os.Getenv("SHELL"); shell != "" {
		return shell
	}

	// Default shells by platform
	return "/bin/bash"
}

// expandPath expands ~ and environment variables in path
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			path = filepath.Join(home, path[2:])
		}
	}
	return os.ExpandEnv(path)
}

// isValidPath checks if a path exists and is a directory
func isValidPath(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// checkExecutable checks if a file is executable
func checkExecutable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	mode := info.Mode()
	if mode&0111 == 0 {
		return fmt.Errorf("file is not executable")
	}

	return nil
}
