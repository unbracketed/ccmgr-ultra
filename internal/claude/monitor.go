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
	"sync"
	"time"
)

// DefaultStateMonitor implements StateMonitor interface
type DefaultStateMonitor struct {
	config       *ProcessConfig
	detector     ProcessDetector
	logMonitors  map[string]*LogMonitor
	running      bool
	mutex        sync.RWMutex
	stopCh       chan struct{}
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewDefaultStateMonitor creates a new state monitor
func NewDefaultStateMonitor(config *ProcessConfig, detector ProcessDetector) *DefaultStateMonitor {
	return &DefaultStateMonitor{
		config:      config,
		detector:    detector,
		logMonitors: make(map[string]*LogMonitor),
		stopCh:      make(chan struct{}),
	}
}

// Start begins monitoring process states
func (m *DefaultStateMonitor) Start(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.running {
		return fmt.Errorf("monitor is already running")
	}

	m.ctx, m.cancel = context.WithCancel(ctx)
	m.running = true

	// Compile regex patterns for state detection
	if err := m.config.CompilePatterns(); err != nil {
		return fmt.Errorf("failed to compile state patterns: %w", err)
	}

	go m.monitorLoop()
	return nil
}

// Stop stops the monitoring process
func (m *DefaultStateMonitor) Stop() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.running {
		return nil
	}

	m.running = false
	if m.cancel != nil {
		m.cancel()
	}

	select {
	case <-m.stopCh:
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout waiting for monitor to stop")
	}
}

// MonitorState determines the current state of a process
func (m *DefaultStateMonitor) MonitorState(ctx context.Context, process *ProcessInfo) (ProcessState, error) {
	// Check if process still exists
	if detector, ok := m.detector.(*DefaultDetector); ok {
		exists, err := detector.processExists(process.PID)
		if err != nil {
			return StateUnknown, fmt.Errorf("failed to check process existence: %w", err)
		}
		if !exists {
			return StateStopped, nil
		}
	}

	// Try multiple detection methods in order of preference
	state := StateUnknown

	// 1. Resource-based detection
	if m.config.EnableResourceMonitoring {
		if resourceState, err := m.detectStateFromResources(process); err == nil && resourceState != StateUnknown {
			state = resourceState
		}
	}

	// 2. Log-based detection (if enabled and available)
	if m.config.EnableLogParsing && state == StateUnknown {
		if logState, err := m.detectStateFromLogs(process); err == nil && logState != StateUnknown {
			state = logState
		}
	}

	// 3. Process activity detection
	if state == StateUnknown {
		if activityState, err := m.detectStateFromActivity(process); err == nil && activityState != StateUnknown {
			state = activityState
		}
	}

	// 4. Tmux output detection (if tmux session is available)
	if state == StateUnknown && process.TmuxSession != "" {
		if tmuxState, err := m.detectStateFromTmux(process); err == nil && tmuxState != StateUnknown {
			state = tmuxState
		}
	}

	// Default to idle if we can't determine state but process exists
	if state == StateUnknown {
		state = StateIdle
	}

	return state, nil
}

// monitorLoop is the main monitoring loop
func (m *DefaultStateMonitor) monitorLoop() {
	defer func() {
		m.stopCh <- struct{}{}
	}()

	ticker := time.NewTicker(m.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			// This method would be called by the tracker
			// Individual process monitoring is handled by MonitorState
		}
	}
}

// detectStateFromResources analyzes CPU and memory usage to infer state
func (m *DefaultStateMonitor) detectStateFromResources(process *ProcessInfo) (ProcessState, error) {
	// Get current resource usage
	if detector, ok := m.detector.(*DefaultDetector); ok {
		cpuPercent, memoryMB, err := detector.getProcessResources(process.PID)
		if err != nil {
			return StateUnknown, err
		}

		// Update process stats
		process.UpdateStats(cpuPercent, memoryMB)

		// Analyze resource patterns
		// High CPU usage typically indicates busy state
		if cpuPercent > 5.0 {
			return StateBusy, nil
		}
		
		// Very low CPU with stable memory suggests idle
		if cpuPercent < 1.0 {
			return StateIdle, nil
		}
	}

	return StateUnknown, nil
}

// detectStateFromLogs analyzes log files for state indicators
func (m *DefaultStateMonitor) detectStateFromLogs(process *ProcessInfo) (ProcessState, error) {
	// Try to find log files for this process
	logPaths := m.findLogFiles(process)
	
	for _, logPath := range logPaths {
		state, err := m.analyzeLogFile(logPath, process.SessionID)
		if err != nil {
			continue // Try next log file
		}
		if state != StateUnknown {
			return state, nil
		}
	}

	return StateUnknown, nil
}

// detectStateFromActivity analyzes file system and network activity
func (m *DefaultStateMonitor) detectStateFromActivity(process *ProcessInfo) (ProcessState, error) {
	// Use lsof to check file descriptor activity
	cmd := exec.Command("lsof", "-p", strconv.Itoa(process.PID))
	output, err := cmd.Output()
	if err != nil {
		return StateUnknown, err
	}

	// Count open files and network connections
	lines := strings.Split(string(output), "\n")
	networkConnections := 0
	openFiles := 0

	for _, line := range lines {
		if strings.Contains(line, "TCP") || strings.Contains(line, "UDP") {
			networkConnections++
		}
		if strings.Contains(line, "REG") || strings.Contains(line, "DIR") {
			openFiles++
		}
	}

	// High network activity suggests busy state
	if networkConnections > 2 {
		return StateBusy, nil
	}

	// Moderate file activity suggests idle state
	if openFiles > 10 {
		return StateIdle, nil
	}

	return StateUnknown, nil
}

// detectStateFromTmux analyzes tmux session output
func (m *DefaultStateMonitor) detectStateFromTmux(process *ProcessInfo) (ProcessState, error) {
	// Capture tmux pane content
	cmd := exec.Command("tmux", "capture-pane", "-t", process.TmuxSession, "-p")
	output, err := cmd.Output()
	if err != nil {
		return StateUnknown, fmt.Errorf("failed to capture tmux pane: %w", err)
	}

	content := string(output)
	return m.analyzeTextContent(content), nil
}

// findLogFiles attempts to locate log files for a process
func (m *DefaultStateMonitor) findLogFiles(process *ProcessInfo) []string {
	var logPaths []string

	// Check configured log paths
	for _, pattern := range m.config.LogPaths {
		// Expand tilde and environment variables
		expandedPattern := os.ExpandEnv(pattern)
		if strings.HasPrefix(expandedPattern, "~/") {
			homeDir, err := os.UserHomeDir()
			if err == nil {
				expandedPattern = filepath.Join(homeDir, expandedPattern[2:])
			}
		}

		// Handle patterns with wildcards
		if strings.Contains(expandedPattern, "*") {
			matches, err := filepath.Glob(expandedPattern)
			if err == nil {
				logPaths = append(logPaths, matches...)
			}
		} else {
			if _, err := os.Stat(expandedPattern); err == nil {
				logPaths = append(logPaths, expandedPattern)
			}
		}
	}

	// Also check common locations relative to working directory
	if process.WorkingDir != "" {
		commonPaths := []string{
			filepath.Join(process.WorkingDir, ".claude"),
			filepath.Join(process.WorkingDir, "logs"),
			filepath.Join(process.WorkingDir, ".logs"),
		}
		
		for _, path := range commonPaths {
			if info, err := os.Stat(path); err == nil && info.IsDir() {
				// Look for log files in this directory
				entries, err := os.ReadDir(path)
				if err == nil {
					for _, entry := range entries {
						if strings.HasSuffix(entry.Name(), ".log") {
							logPaths = append(logPaths, filepath.Join(path, entry.Name()))
						}
					}
				}
			}
		}
	}

	return logPaths
}

// analyzeLogFile analyzes a log file for state indicators
func (m *DefaultStateMonitor) analyzeLogFile(logPath, sessionID string) (ProcessState, error) {
	// Get or create log monitor for this file
	monitor := m.getLogMonitor(logPath, sessionID)
	
	file, err := os.Open(logPath)
	if err != nil {
		return StateUnknown, err
	}
	defer file.Close()

	// Get file size and check if it has grown
	fileInfo, err := file.Stat()
	if err != nil {
		return StateUnknown, err
	}

	currentSize := fileInfo.Size()
	lastOffset := monitor.GetLastOffset()

	// If file hasn't grown, check timestamp
	if currentSize <= lastOffset {
		// File hasn't grown, might indicate idle state
		if time.Since(monitor.LastCheck) > m.config.StateTimeout {
			return StateIdle, nil
		}
		return StateUnknown, nil
	}

	// Seek to last read position
	if _, err := file.Seek(lastOffset, 0); err != nil {
		// If seek fails, start from beginning
		if _, err := file.Seek(0, 0); err != nil {
			return StateUnknown, err
		}
		lastOffset = 0
	}

	// Read new content
	scanner := bufio.NewScanner(file)
	var newContent strings.Builder
	bytesRead := int64(0)

	for scanner.Scan() {
		line := scanner.Text()
		newContent.WriteString(line)
		newContent.WriteString("\n")
		bytesRead += int64(len(line)) + 1
	}

	// Update monitor offset
	monitor.SetLastOffset(lastOffset + bytesRead)

	// Analyze the new content for state indicators
	return m.analyzeTextContent(newContent.String()), nil
}

// analyzeTextContent analyzes text content for state patterns
func (m *DefaultStateMonitor) analyzeTextContent(content string) ProcessState {
	// Check each state pattern
	states := []ProcessState{StateError, StateBusy, StateWaiting, StateIdle}
	
	for _, state := range states {
		pattern := m.config.GetCompiledPattern(state)
		if pattern != nil && pattern.MatchString(content) {
			return state
		}
	}

	return StateUnknown
}

// getLogMonitor gets or creates a log monitor for a file
func (m *DefaultStateMonitor) getLogMonitor(logPath, sessionID string) *LogMonitor {
	key := fmt.Sprintf("%s:%s", logPath, sessionID)
	
	if monitor, exists := m.logMonitors[key]; exists {
		return monitor
	}

	monitor := &LogMonitor{
		LogPath:   logPath,
		ProcessID: sessionID,
		StateRegex: make(map[ProcessState]*regexp.Regexp),
	}

	// Copy compiled patterns
	for state := range m.config.StatePatterns {
		stateMap := map[string]ProcessState{
			"busy":    StateBusy,
			"idle":    StateIdle,
			"waiting": StateWaiting,
			"error":   StateError,
		}
		if s, exists := stateMap[state]; exists {
			if pattern := m.config.GetCompiledPattern(s); pattern != nil {
				monitor.StateRegex[s] = pattern
			}
		}
	}

	m.logMonitors[key] = monitor
	return monitor
}

// CleanupLogMonitors removes unused log monitors
func (m *DefaultStateMonitor) CleanupLogMonitors() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Remove monitors that haven't been accessed recently
	cutoff := time.Now().Add(-m.config.CleanupInterval)
	for key, monitor := range m.logMonitors {
		if monitor.LastCheck.Before(cutoff) {
			delete(m.logMonitors, key)
		}
	}
}

// GetMonitorStats returns statistics about the monitor
func (m *DefaultStateMonitor) GetMonitorStats() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	stats := map[string]interface{}{
		"running":          m.running,
		"log_monitors":     len(m.logMonitors),
		"poll_interval":    m.config.PollInterval.String(),
		"log_parsing":      m.config.EnableLogParsing,
		"resource_monitoring": m.config.EnableResourceMonitoring,
	}

	return stats
}