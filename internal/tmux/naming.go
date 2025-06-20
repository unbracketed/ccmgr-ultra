package tmux

import (
	"fmt"
	"regexp"
	"strings"
)

type NamingConvention struct {
	Pattern   string
	MaxLength int
}

const (
	sessionPrefix = "ccmgr"
	maxNameLength = 50
)

var (
	sessionNameRegex = regexp.MustCompile(`^ccmgr-([^-]+)-([^-]+)-(.+)$`)
	invalidChars     = regexp.MustCompile(`[^a-zA-Z0-9_]`)
)

func GenerateSessionName(project, worktree, branch string) string {
	sanitizedProject := SanitizeNameComponent(project)
	sanitizedWorktree := SanitizeNameComponent(worktree)
	sanitizedBranch := SanitizeNameComponent(branch)

	name := fmt.Sprintf("%s-%s-%s-%s", sessionPrefix, sanitizedProject, sanitizedWorktree, sanitizedBranch)

	if len(name) > maxNameLength {
		name = truncateSessionName(name, maxNameLength)
	}

	return name
}

func ParseSessionName(sessionName string) (project, worktree, branch string, err error) {
	if !ValidateSessionName(sessionName) {
		return "", "", "", fmt.Errorf("invalid session name format: %s", sessionName)
	}

	matches := sessionNameRegex.FindStringSubmatch(sessionName)
	if len(matches) != 4 {
		return "", "", "", fmt.Errorf("failed to parse session name: %s", sessionName)
	}

	project = matches[1]
	worktree = matches[2]
	branch = matches[3]

	return project, worktree, branch, nil
}

func ValidateSessionName(name string) bool {
	if !strings.HasPrefix(name, sessionPrefix+"-") {
		return false
	}

	if len(name) > maxNameLength {
		return false
	}

	return sessionNameRegex.MatchString(name)
}

func SanitizeNameComponent(component string) string {
	if component == "" {
		return "unnamed"
	}

	sanitized := invalidChars.ReplaceAllString(component, "_")
	
	sanitized = strings.Trim(sanitized, "_-")
	
	if sanitized == "" {
		return "unnamed"
	}

	if len(sanitized) > 20 {
		sanitized = sanitized[:20]
		sanitized = strings.TrimRight(sanitized, "_-")
	}

	return sanitized
}

func truncateSessionName(name string, maxLen int) string {
	if len(name) <= maxLen {
		return name
	}

	parts := strings.Split(name, "-")
	if len(parts) < 4 {
		return name[:maxLen]
	}

	prefix := parts[0]
	project := parts[1]
	worktree := parts[2]
	branch := strings.Join(parts[3:], "-")

	availableLength := maxLen - len(prefix) - 3

	projectLen := len(project)
	worktreeLen := len(worktree)
	branchLen := len(branch)

	totalLen := projectLen + worktreeLen + branchLen

	if totalLen <= availableLength {
		return name
	}

	targetLen := availableLength / 3

	if projectLen > targetLen {
		project = project[:targetLen-1] + "~"
	}
	if worktreeLen > targetLen {
		worktree = worktree[:targetLen-1] + "~"
	}
	if branchLen > targetLen {
		branch = branch[:targetLen-1] + "~"
	}

	remaining := availableLength - len(project) - len(worktree) - len(branch)
	if remaining > 0 && branchLen > targetLen {
		additionalBranchLen := min(remaining, branchLen-targetLen)
		branch = branch[:len(branch)-1+additionalBranchLen] + "~"
	}

	return fmt.Sprintf("%s-%s-%s-%s", prefix, project, worktree, branch)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}