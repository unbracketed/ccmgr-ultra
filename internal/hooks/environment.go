package hooks

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// EnvironmentBuilder helps build environment variables for hook execution
type EnvironmentBuilder struct {
	variables map[string]string
}

// NewEnvironmentBuilder creates a new environment builder
func NewEnvironmentBuilder() *EnvironmentBuilder {
	return &EnvironmentBuilder{
		variables: make(map[string]string),
	}
}

// WithContext adds context variables to the environment
func (eb *EnvironmentBuilder) WithContext(ctx HookContext) *EnvironmentBuilder {
	if ctx.WorktreePath != "" {
		eb.variables["CCMGR_WORKTREE_PATH"] = ctx.WorktreePath
	}
	if ctx.WorktreeBranch != "" {
		eb.variables["CCMGR_WORKTREE_BRANCH"] = ctx.WorktreeBranch
	}
	if ctx.ProjectName != "" {
		eb.variables["CCMGR_PROJECT_NAME"] = ctx.ProjectName
	}
	if ctx.SessionID != "" {
		eb.variables["CCMGR_SESSION_ID"] = ctx.SessionID
	}
	if ctx.SessionType != "" {
		eb.variables["CCMGR_SESSION_TYPE"] = ctx.SessionType
	}
	if ctx.OldState != "" {
		eb.variables["CCMGR_OLD_STATE"] = ctx.OldState
	}
	if ctx.NewState != "" {
		eb.variables["CCMGR_NEW_STATE"] = ctx.NewState
	}
	
	// Add custom variables
	for key, value := range ctx.CustomVars {
		eb.variables[key] = value
	}
	
	return eb
}

// WithStatusHookVars adds status hook specific variables
func (eb *EnvironmentBuilder) WithStatusHookVars(hookType HookType, ctx HookContext) *EnvironmentBuilder {
	eb.WithContext(ctx)
	
	// Legacy environment variables for backward compatibility
	if ctx.WorktreePath != "" {
		eb.variables["CCMANAGER_WORKTREE"] = ctx.WorktreePath
	}
	if ctx.WorktreeBranch != "" {
		eb.variables["CCMANAGER_WORKTREE_BRANCH"] = ctx.WorktreeBranch
	}
	if ctx.NewState != "" {
		eb.variables["CCMANAGER_NEW_STATE"] = ctx.NewState
	}
	if ctx.SessionID != "" {
		eb.variables["CCMANAGER_SESSION_ID"] = ctx.SessionID
	}
	
	eb.variables["CCMANAGER_TIMESTAMP"] = time.Now().Format(time.RFC3339)
	
	return eb
}

// WithWorktreeCreationVars adds worktree creation specific variables
func (eb *EnvironmentBuilder) WithWorktreeCreationVars(ctx HookContext) *EnvironmentBuilder {
	eb.WithContext(ctx)
	
	eb.variables["CCMGR_WORKTREE_TYPE"] = "new"
	
	// Add parent path if available
	if parentPath, exists := ctx.CustomVars["CCMGR_PARENT_PATH"]; exists {
		eb.variables["CCMGR_PARENT_PATH"] = parentPath
	}
	
	return eb
}

// WithWorktreeActivationVars adds worktree activation specific variables
func (eb *EnvironmentBuilder) WithWorktreeActivationVars(ctx HookContext) *EnvironmentBuilder {
	eb.WithContext(ctx)
	
	if ctx.SessionType == "" {
		eb.variables["CCMGR_SESSION_TYPE"] = "new"
	}
	
	// Add previous state if available
	if prevState, exists := ctx.CustomVars["CCMGR_PREVIOUS_STATE"]; exists {
		eb.variables["CCMGR_PREVIOUS_STATE"] = prevState
	}
	
	return eb
}

// WithCustomVar adds a custom environment variable
func (eb *EnvironmentBuilder) WithCustomVar(key, value string) *EnvironmentBuilder {
	eb.variables[key] = value
	return eb
}

// Build returns the environment variables as a slice of strings
func (eb *EnvironmentBuilder) Build() []string {
	env := os.Environ()
	
	// Add timestamp
	env = append(env, fmt.Sprintf("CCMGR_TIMESTAMP=%s", time.Now().Format(time.RFC3339)))
	
	for key, value := range eb.variables {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}
	
	return env
}

// BuildMap returns the environment variables as a map
func (eb *EnvironmentBuilder) BuildMap() map[string]string {
	result := make(map[string]string)
	
	// Add timestamp
	result["CCMGR_TIMESTAMP"] = time.Now().Format(time.RFC3339)
	
	// Add custom variables
	for key, value := range eb.variables {
		result[key] = value
	}
	
	return result
}

// expandEnvironmentVariables expands environment variables in a string
func expandEnvironmentVariables(s string) string {
	return os.ExpandEnv(s)
}

// sanitizeEnvironmentValue sanitizes an environment variable value
func sanitizeEnvironmentValue(value string) string {
	// Remove any null bytes and control characters
	return strings.ReplaceAll(strings.ReplaceAll(value, "\x00", ""), "\n", "\\n")
}

// validateEnvironmentKey validates an environment variable key
func validateEnvironmentKey(key string) error {
	if key == "" {
		return fmt.Errorf("environment variable key cannot be empty")
	}
	
	if strings.Contains(key, "=") {
		return fmt.Errorf("environment variable key '%s' cannot contain '='", key)
	}
	
	// Check for valid characters (alphanumeric and underscore)
	for _, char := range key {
		if !((char >= 'A' && char <= 'Z') || 
			 (char >= 'a' && char <= 'z') || 
			 (char >= '0' && char <= '9') || 
			 char == '_') {
			return fmt.Errorf("environment variable key '%s' contains invalid character '%c'", key, char)
		}
	}
	
	return nil
}

// mergeEnvironmentMaps merges multiple environment maps, with later maps taking precedence
func mergeEnvironmentMaps(maps ...map[string]string) map[string]string {
	result := make(map[string]string)
	
	for _, m := range maps {
		for key, value := range m {
			result[key] = value
		}
	}
	
	return result
}