package claude

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// DefaultDetector implements ProcessDetector interface
type DefaultDetector struct {
	config      *ProcessConfig
	claudeRegex *regexp.Regexp
	knownPaths  []string
}

// NewDefaultDetector creates a new process detector
func NewDefaultDetector(config *ProcessConfig) (*DefaultDetector, error) {
	// Compile regex for identifying Claude Code processes
	claudeRegex, err := regexp.Compile(`(?i)(claude|claude-code|claude_code)`)
	if err != nil {
		return nil, fmt.Errorf("failed to compile claude regex: %w", err)
	}

	detector := &DefaultDetector{
		config:      config,
		claudeRegex: claudeRegex,
		knownPaths: []string{
			"/usr/local/bin/claude",
			"/opt/homebrew/bin/claude",
			"/usr/bin/claude",
			"~/.local/bin/claude",
			"./claude",
		},
	}

	return detector, nil
}

// DetectProcesses finds all Claude Code processes currently running
func (d *DefaultDetector) DetectProcesses(ctx context.Context) ([]*ProcessInfo, error) {
	processes := make([]*ProcessInfo, 0)

	// Get all processes using ps command
	pids, err := d.getAllProcessPIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get process PIDs: %w", err)
	}

	for _, pid := range pids {
		select {
		case <-ctx.Done():
			return processes, ctx.Err()
		default:
		}

		isClaudeProcess, err := d.IsClaudeProcess(pid)
		if err != nil {
			continue // Skip processes we can't access
		}

		if isClaudeProcess {
			processInfo, err := d.GetProcessInfo(pid)
			if err != nil {
				continue // Skip processes we can't get info for
			}
			processes = append(processes, processInfo)
		}
	}

	return processes, nil
}

// IsClaudeProcess checks if a given PID is a Claude Code process
func (d *DefaultDetector) IsClaudeProcess(pid int) (bool, error) {
	// Get process command line
	cmdline, err := d.getProcessCommand(pid)
	if err != nil {
		return false, err
	}

	// Check if command matches Claude Code patterns
	for _, arg := range cmdline {
		if d.claudeRegex.MatchString(arg) {
			return true, nil
		}
		// Also check for common Claude Code execution patterns
		if strings.Contains(arg, "anthropic") ||
			strings.Contains(arg, "claude") ||
			strings.Contains(strings.ToLower(arg), "claude") {
			return true, nil
		}
	}

	// Check if binary path matches known Claude Code paths
	if len(cmdline) > 0 {
		binaryPath := cmdline[0]
		baseName := filepath.Base(binaryPath)
		if d.claudeRegex.MatchString(baseName) {
			return true, nil
		}
	}

	return false, nil
}

// GetProcessInfo retrieves detailed information about a process
func (d *DefaultDetector) GetProcessInfo(pid int) (*ProcessInfo, error) {
	// Get basic process info
	cmdline, err := d.getProcessCommand(pid)
	if err != nil {
		return nil, fmt.Errorf("failed to get command for PID %d: %w", pid, err)
	}

	if len(cmdline) == 0 {
		return nil, fmt.Errorf("empty command line for PID %d", pid)
	}

	// Get working directory
	workingDir, err := d.getProcessWorkingDir(pid)
	if err != nil {
		workingDir = "" // Not fatal, continue without working dir
	}

	// Get start time
	startTime, err := d.getProcessStartTime(pid)
	if err != nil {
		startTime = time.Now() // Default to current time if can't determine
	}

	// Get resource usage
	cpuPercent, memoryMB, err := d.getProcessResources(pid)
	if err != nil {
		cpuPercent, memoryMB = 0, 0 // Default to 0 if can't determine
	}

	// Generate session ID based on PID and start time
	sessionID := fmt.Sprintf("claude-%d-%d", pid, startTime.Unix())

	// Try to determine associated tmux session
	tmuxSession := d.getTmuxSession(pid)

	// Try to determine worktree ID from working directory
	worktreeID := d.getWorktreeID(workingDir)

	processInfo := &ProcessInfo{
		PID:         pid,
		SessionID:   sessionID,
		WorkingDir:  workingDir,
		Command:     cmdline,
		StartTime:   startTime,
		State:       StateStarting, // Default to starting state
		LastUpdate:  time.Now(),
		TmuxSession: tmuxSession,
		WorktreeID:  worktreeID,
		CPUPercent:  cpuPercent,
		MemoryMB:    memoryMB,
	}

	return processInfo, nil
}

// getAllProcessPIDs returns all process PIDs on the system
func (d *DefaultDetector) getAllProcessPIDs(ctx context.Context) ([]int, error) {
	cmd := exec.CommandContext(ctx, "ps", "-A", "-o", "pid=")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute ps command: %w", err)
	}

	var pids []int
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		pid, err := strconv.Atoi(line)
		if err != nil {
			continue // Skip invalid PIDs
		}
		pids = append(pids, pid)
	}

	return pids, nil
}

// getProcessCommand returns the command line arguments for a process
func (d *DefaultDetector) getProcessCommand(pid int) ([]string, error) {
	cmd := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "args=")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get command for PID %d: %w", pid, err)
	}

	cmdline := strings.TrimSpace(string(output))
	if cmdline == "" {
		return nil, fmt.Errorf("empty command line for PID %d", pid)
	}

	// Split command line into arguments
	// This is a simplified split - in production, we might want more sophisticated parsing
	args := strings.Fields(cmdline)
	return args, nil
}

// getProcessWorkingDir attempts to get the working directory of a process
func (d *DefaultDetector) getProcessWorkingDir(pid int) (string, error) {
	// Try to read from /proc (Linux) or use lsof (macOS)
	procPath := fmt.Sprintf("/proc/%d/cwd", pid)
	if link, err := os.Readlink(procPath); err == nil {
		return link, nil
	}

	// Fallback to lsof for macOS and other systems
	cmd := exec.Command("lsof", "-p", strconv.Itoa(pid), "-d", "cwd", "-Fn")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory for PID %d: %w", pid, err)
	}

	// Parse lsof output to extract working directory
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "n") && len(line) > 1 {
			return line[1:], nil
		}
	}

	return "", fmt.Errorf("could not determine working directory for PID %d", pid)
}

// getProcessStartTime attempts to get the start time of a process
func (d *DefaultDetector) getProcessStartTime(pid int) (time.Time, error) {
	cmd := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "lstart=")
	output, err := cmd.Output()
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get start time for PID %d: %w", pid, err)
	}

	startTimeStr := strings.TrimSpace(string(output))
	if startTimeStr == "" {
		return time.Time{}, fmt.Errorf("empty start time for PID %d", pid)
	}

	// Parse the start time - format varies by system
	// Try common formats
	formats := []string{
		"Mon Jan 2 15:04:05 2006",
		"Mon Jan 2 15:04:05 MST 2006",
		"2006-01-02 15:04:05",
		"Jan 2 15:04:05",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, startTimeStr); err == nil {
			// If year is missing, assume current year
			if t.Year() == 0 {
				now := time.Now()
				t = t.AddDate(now.Year(), 0, 0)
			}
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("could not parse start time '%s' for PID %d", startTimeStr, pid)
}

// getProcessResources gets CPU and memory usage for a process
func (d *DefaultDetector) getProcessResources(pid int) (float64, int64, error) {
	cmd := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "pcpu=,rss=")
	output, err := cmd.Output()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get resources for PID %d: %w", pid, err)
	}

	fields := strings.Fields(strings.TrimSpace(string(output)))
	if len(fields) < 2 {
		return 0, 0, fmt.Errorf("insufficient resource data for PID %d", pid)
	}

	cpuPercent, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		cpuPercent = 0
	}

	rssKB, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		rssKB = 0
	}

	// Convert RSS from KB to MB
	memoryMB := rssKB / 1024

	return cpuPercent, memoryMB, nil
}

// getTmuxSession attempts to find the tmux session associated with a process
func (d *DefaultDetector) getTmuxSession(pid int) string {
	// Check if the process itself is running in tmux
	cmd := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "ppid=")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	ppidStr := strings.TrimSpace(string(output))
	ppid, err := strconv.Atoi(ppidStr)
	if err != nil {
		return ""
	}

	// Get parent process command to check if it's tmux
	parentCmd, err := d.getProcessCommand(ppid)
	if err != nil {
		return ""
	}

	if len(parentCmd) > 0 {
		if strings.Contains(parentCmd[0], "tmux") {
			// Try to extract session name from tmux process
			for _, arg := range parentCmd {
				if strings.HasPrefix(arg, "ccmgr-") {
					return arg
				}
			}
		}
	}

	// Alternative: check if TMUX environment variable is set
	// This would require reading process environment, which is more complex
	return ""
}

// getWorktreeID attempts to determine the git worktree ID from working directory
func (d *DefaultDetector) getWorktreeID(workingDir string) string {
	if workingDir == "" {
		return ""
	}

	// Check if this is a git repository
	gitDir := filepath.Join(workingDir, ".git")
	if info, err := os.Stat(gitDir); err == nil {
		// For git worktrees, the .git file contains the path to the actual git dir
		// We can extract the worktree name from this
		if !info.IsDir() {
			// Read .git file to get actual git directory
			content, err := os.ReadFile(gitDir)
			if err == nil {
				gitDirPath := strings.TrimSpace(strings.TrimPrefix(string(content), "gitdir: "))
				// Extract worktree name from path like: /path/to/repo/.git/worktrees/branch-name
				if strings.Contains(gitDirPath, "/worktrees/") {
					parts := strings.Split(gitDirPath, "/worktrees/")
					if len(parts) > 1 {
						return parts[1]
					}
				}
			}
		}
	}

	// Fallback: use the directory name
	return filepath.Base(workingDir)
}

// RefreshProcess updates process information for an existing process
func (d *DefaultDetector) RefreshProcess(process *ProcessInfo) error {
	// Check if process still exists
	exists, err := d.processExists(process.PID)
	if err != nil {
		return fmt.Errorf("failed to check if process exists: %w", err)
	}

	if !exists {
		process.SetState(StateStopped)
		return nil
	}

	// Update resource usage
	cpuPercent, memoryMB, err := d.getProcessResources(process.PID)
	if err == nil {
		process.UpdateStats(cpuPercent, memoryMB)
	}

	return nil
}

// processExists checks if a process with the given PID exists
func (d *DefaultDetector) processExists(pid int) (bool, error) {
	_, err := os.Stat(fmt.Sprintf("/proc/%d", pid))
	if err == nil {
		return true, nil
	}

	// Fallback for systems without /proc
	cmd := exec.Command("ps", "-p", strconv.Itoa(pid))
	err = cmd.Run()
	return err == nil, nil
}
