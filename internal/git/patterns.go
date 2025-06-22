package git

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"
	"unicode"

	"github.com/bcdekker/ccmgr-ultra/internal/config"
)

// PatternManager handles directory naming patterns
type PatternManager struct {
	config *config.WorktreeConfig
}

// PatternContext provides variables for pattern substitution
type PatternContext struct {
	Project   string `json:"project"`
	Branch    string `json:"branch"`
	Worktree  string `json:"worktree"`
	Timestamp string `json:"timestamp"`
	UserName  string `json:"user"`
	Prefix    string `json:"prefix"`
	Suffix    string `json:"suffix"`
}

// DirectoryPattern represents a naming pattern configuration
type DirectoryPattern struct {
	Template   string
	AutoCreate bool
	Sanitize   bool
	MaxLength  int
	Prefix     string
	Suffix     string
}

// NewPatternManager creates a new PatternManager
func NewPatternManager(cfg *config.WorktreeConfig) *PatternManager {
	if cfg == nil {
		cfg = &config.WorktreeConfig{}
		cfg.SetDefaults()
	}
	return &PatternManager{config: cfg}
}

// ApplyPattern applies a naming pattern with the given context
func (pm *PatternManager) ApplyPattern(pattern string, context PatternContext) (string, error) {
	if pattern == "" {
		pattern = pm.config.DirectoryPattern
	}

	// Validate the pattern first
	if err := pm.ValidatePattern(pattern); err != nil {
		return "", fmt.Errorf("invalid pattern: %w", err)
	}

	// Resolve pattern variables
	resolved, err := pm.ResolvePatternVariables(pattern, context)
	if err != nil {
		return "", fmt.Errorf("failed to resolve pattern variables: %w", err)
	}

	// Sanitize the path if needed
	sanitized := pm.SanitizePath(resolved)

	// Apply length restrictions
	if pm.config.AutoDirectory && len(sanitized) > 100 { // reasonable default
		sanitized = pm.truncatePath(sanitized, 100)
	}

	return sanitized, nil
}

// ValidatePattern validates a directory naming pattern
func (pm *PatternManager) ValidatePattern(pattern string) error {
	if pattern == "" {
		return fmt.Errorf("pattern cannot be empty")
	}

	// Check for template syntax
	_, err := createPatternTemplate(pattern)
	if err != nil {
		return fmt.Errorf("invalid template syntax: %w", err)
	}

	// Check for required variables
	if !strings.Contains(pattern, "{{") || !strings.Contains(pattern, "}}") {
		return fmt.Errorf("pattern must contain at least one template variable")
	}

	// Check for dangerous patterns
	dangerousPatterns := []string{
		"..", // parent directory traversal
		"~",  // home directory
		"/",  // absolute paths
		"\\", // Windows paths
	}

	for _, dangerous := range dangerousPatterns {
		if strings.Contains(pattern, dangerous) {
			return fmt.Errorf("pattern contains dangerous sequence: %s", dangerous)
		}
	}

	// Validate against known variables
	validVars := []string{
		"{{.Project}}", "{{.Branch}}", "{{.Worktree}}", 
		"{{.Timestamp}}", "{{.UserName}}", "{{.Prefix}}", "{{.Suffix}}",
	}

	// Extract variables from pattern
	varRegex := regexp.MustCompile(`\{\{\.[\w]+\}\}`)
	foundVars := varRegex.FindAllString(pattern, -1)

	for _, foundVar := range foundVars {
		valid := false
		for _, validVar := range validVars {
			if foundVar == validVar {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("unknown template variable: %s", foundVar)
		}
	}

	return nil
}

// SanitizePath sanitizes a path to be safe for filesystem use
func (pm *PatternManager) SanitizePath(path string) string {
	if path == "" {
		return path
	}

	// Replace unsafe characters
	unsafe := []string{
		"<", ">", ":", "\"", "|", "?", "*", 
		"\x00", "\x01", "\x02", "\x03", "\x04", "\x05", "\x06", "\x07",
		"\x08", "\x09", "\x0a", "\x0b", "\x0c", "\x0d", "\x0e", "\x0f",
		"\x10", "\x11", "\x12", "\x13", "\x14", "\x15", "\x16", "\x17",
		"\x18", "\x19", "\x1a", "\x1b", "\x1c", "\x1d", "\x1e", "\x1f",
	}

	sanitized := path
	for _, char := range unsafe {
		sanitized = strings.ReplaceAll(sanitized, char, "-")
	}

	// Replace multiple consecutive separators with single separator
	sanitized = regexp.MustCompile(`[-_\s]+`).ReplaceAllString(sanitized, "-")

	// Remove leading/trailing separators
	sanitized = strings.Trim(sanitized, "-_")

	// Ensure it's not empty after sanitization
	if sanitized == "" {
		sanitized = "worktree"
	}

	return sanitized
}

// GenerateWorktreePath generates a full worktree path based on configuration
func (pm *PatternManager) GenerateWorktreePath(branch, project string) (string, error) {
	context := PatternContext{
		Project:   pm.sanitizeComponent(project),
		Branch:    pm.sanitizeComponent(branch),
		Worktree:  pm.generateWorktreeID(branch),
		Timestamp: time.Now().Format("20060102-150405"),
		UserName:  pm.getUserName(),
		Prefix:    pm.config.DefaultBranch, // Use default branch as prefix
		Suffix:    "",
	}

	// Apply the pattern
	dirName, err := pm.ApplyPattern(pm.config.DirectoryPattern, context)
	if err != nil {
		return "", fmt.Errorf("failed to apply pattern: %w", err)
	}

	// Get current working directory as base
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	// Create .worktrees base directory if it doesn't exist
	worktreeBaseDir := filepath.Join(cwd, ".worktrees")
	if err := os.MkdirAll(worktreeBaseDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create .worktrees directory: %w", err)
	}

	// Create full path within .worktrees
	fullPath := filepath.Join(worktreeBaseDir, dirName)
	
	// Clean the path
	fullPath = filepath.Clean(fullPath)

	return fullPath, nil
}

// ResolvePatternVariables resolves template variables in a pattern
func (pm *PatternManager) ResolvePatternVariables(template string, context PatternContext) (string, error) {
	// Create a template
	tmpl, err := createPatternTemplate(template)
	if err != nil {
		return "", fmt.Errorf("failed to create template: %w", err)
	}

	// Execute template with context
	var buf strings.Builder
	if err := tmpl.Execute(&buf, context); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// createPatternTemplate creates a template with custom functions
func createPatternTemplate(pattern string) (*template.Template, error) {
	funcMap := template.FuncMap{
		"lower":     strings.ToLower,
		"upper":     strings.ToUpper,
		"title":     strings.Title,
		"replace":   replaceString,
		"trim":      strings.TrimSpace,
		"sanitize":  sanitizeForFilesystem,
		"truncate":  truncateString,
	}

	return template.New("pattern").Funcs(funcMap).Parse(pattern)
}

// sanitizeComponent sanitizes individual components like branch or project names
func (pm *PatternManager) sanitizeComponent(component string) string {
	if component == "" {
		return component
	}

	// Convert to lowercase for consistency
	sanitized := strings.ToLower(component)

	// Replace common separators
	sanitized = strings.ReplaceAll(sanitized, "/", "-")
	sanitized = strings.ReplaceAll(sanitized, "\\", "-")
	sanitized = strings.ReplaceAll(sanitized, " ", "-")
	sanitized = strings.ReplaceAll(sanitized, "_", "-")

	// Remove special characters
	var result strings.Builder
	for _, r := range sanitized {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '.' {
			result.WriteRune(r)
		}
	}

	sanitized = result.String()

	// Remove multiple consecutive dashes
	sanitized = regexp.MustCompile(`-+`).ReplaceAllString(sanitized, "-")

	// Remove leading/trailing dashes
	sanitized = strings.Trim(sanitized, "-")

	// Ensure it's not empty
	if sanitized == "" {
		sanitized = "unnamed"
	}

	return sanitized
}

// generateWorktreeID generates a unique identifier for the worktree
func (pm *PatternManager) generateWorktreeID(branch string) string {
	sanitized := pm.sanitizeComponent(branch)
	timestamp := time.Now().Format("0102-1504") // MMDD-HHMM
	return fmt.Sprintf("%s-%s", sanitized, timestamp)
}

// getUserName gets the git user name or system user name
func (pm *PatternManager) getUserName() string {
	// Try to get git user name first
	gitCmd := NewGitCmd()
	if name, err := gitCmd.Execute("", "config", "--get", "user.name"); err == nil && name != "" {
		return pm.sanitizeComponent(name)
	}

	// Fall back to system user
	if user := os.Getenv("USER"); user != "" {
		return pm.sanitizeComponent(user)
	}

	if user := os.Getenv("USERNAME"); user != "" {
		return pm.sanitizeComponent(user)
	}

	return "user"
}

// truncatePath truncates a path to the specified maximum length
func (pm *PatternManager) truncatePath(path string, maxLength int) string {
	if len(path) <= maxLength {
		return path
	}

	// Try to truncate at word boundaries (dashes)
	parts := strings.Split(path, "-")
	if len(parts) > 1 {
		var result strings.Builder
		for i, part := range parts {
			if result.Len()+len(part)+1 > maxLength { // +1 for dash
				break
			}
			if i > 0 {
				result.WriteString("-")
			}
			result.WriteString(part)
		}
		if result.Len() > 0 {
			return result.String()
		}
	}

	// Simple truncation with ellipsis
	if maxLength > 3 {
		return path[:maxLength-3] + "..."
	}

	return path[:maxLength]
}

// CheckPathAvailable checks if a path is available for use
func (pm *PatternManager) CheckPathAvailable(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Check if path already exists
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("path already exists: %s", path)
	}

	// Check if parent directory exists and is writable
	parentDir := filepath.Dir(path)
	if parentDir != "." {
		if stat, err := os.Stat(parentDir); err != nil {
			return fmt.Errorf("parent directory does not exist: %s", parentDir)
		} else if !stat.IsDir() {
			return fmt.Errorf("parent path is not a directory: %s", parentDir)
		}

		// Check write permission
		testFile := filepath.Join(parentDir, ".ccmgr-write-test")
		if file, err := os.Create(testFile); err != nil {
			return fmt.Errorf("parent directory is not writable: %s", parentDir)
		} else {
			file.Close()
			os.Remove(testFile)
		}
	}

	return nil
}

// CreateDirectory creates a directory with the appropriate permissions
func (pm *PatternManager) CreateDirectory(path string) error {
	if err := pm.CheckPathAvailable(path); err != nil {
		return err
	}

	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return nil
}

// GetPatternVariables returns all available pattern variables with descriptions
func (pm *PatternManager) GetPatternVariables() map[string]string {
	return map[string]string{
		"{{.Project}}":   "Project/repository name (sanitized)",
		"{{.Branch}}":    "Git branch name (sanitized)",
		"{{.Worktree}}":  "Unique worktree identifier",
		"{{.Timestamp}}": "Current timestamp (YYYYMMDD-HHMMSS)",
		"{{.UserName}}":  "Git user name or system user (sanitized)",
		"{{.Prefix}}":    "Configured prefix value",
		"{{.Suffix}}":    "Configured suffix value",
	}
}

// GetPatternFunctions returns all available template functions with descriptions
func (pm *PatternManager) GetPatternFunctions() map[string]string {
	return map[string]string{
		"lower":     "Convert to lowercase: {{.Branch | lower}}",
		"upper":     "Convert to uppercase: {{.Branch | upper}}",
		"title":     "Convert to title case: {{.Branch | title}}",
		"replace":   "Replace text: {{.Branch | replace \"/\" \"-\"}}",
		"trim":      "Trim whitespace: {{.Branch | trim}}",
		"sanitize":  "Sanitize for filesystem: {{.Branch | sanitize}}",
		"truncate":  "Truncate to length: {{.Branch | truncate 10}}",
	}
}

// Template function implementations

// sanitizeForFilesystem sanitizes a string for filesystem use
func sanitizeForFilesystem(s string) string {
	pm := &PatternManager{}
	return pm.SanitizePath(s)
}

// truncateString truncates a string to the specified length
func truncateString(s string, length int) string {
	if len(s) <= length {
		return s
	}
	if length <= 3 {
		return s[:length]
	}
	return s[:length-3] + "..."
}

// replaceString replaces all occurrences of old with new in the string
// For template usage: {{.field | replace "old" "new"}}
func replaceString(old, new, s string) string {
	return strings.ReplaceAll(s, old, new)
}

// ValidatePatternResult validates the result of pattern application
func (pm *PatternManager) ValidatePatternResult(result string) error {
	if result == "" {
		return fmt.Errorf("pattern result cannot be empty")
	}

	// Check for absolute paths
	if filepath.IsAbs(result) {
		return fmt.Errorf("pattern result cannot be an absolute path: %s", result)
	}

	// Check for parent directory traversal
	clean := filepath.Clean(result)
	if strings.Contains(clean, "..") {
		return fmt.Errorf("pattern result contains parent directory traversal: %s", result)
	}

	// Check for reserved names (Windows)
	reserved := []string{
		"CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9",
	}

	upper := strings.ToUpper(result)
	for _, res := range reserved {
		if upper == res || strings.HasPrefix(upper, res+".") {
			return fmt.Errorf("pattern result uses reserved name: %s", result)
		}
	}

	// Check length
	if len(result) > 255 {
		return fmt.Errorf("pattern result too long (%d chars, max 255): %s", len(result), result)
	}

	return nil
}

// GenerateExamplePaths generates example paths for testing patterns
func (pm *PatternManager) GenerateExamplePaths(pattern string) ([]string, error) {
	examples := []PatternContext{
		{
			Project:   "my-project",
			Branch:    "feature/user-auth",
			Worktree:  "feature-user-auth-0102-1430",
			Timestamp: "20240102-143045",
			UserName:  "john-doe",
			Prefix:    "main",
			Suffix:    "dev",
		},
		{
			Project:   "api-server",
			Branch:    "bugfix/memory-leak",
			Worktree:  "bugfix-memory-leak-0103-0915",
			Timestamp: "20240103-091530",
			UserName:  "jane-smith",
			Prefix:    "master",
			Suffix:    "fix",
		},
		{
			Project:   "frontend-app",
			Branch:    "main",
			Worktree:  "main-0103-1020",
			Timestamp: "20240103-102015",
			UserName:  "dev-user",
			Prefix:    "main",
			Suffix:    "",
		},
	}

	var results []string
	for _, context := range examples {
		result, err := pm.ApplyPattern(pattern, context)
		if err != nil {
			return nil, fmt.Errorf("failed to apply pattern with context %+v: %w", context, err)
		}
		results = append(results, result)
	}

	return results, nil
}