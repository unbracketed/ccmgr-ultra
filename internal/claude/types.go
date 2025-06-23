package claude

import (
	"context"
	"regexp"
	"sync"
	"time"
)

// ProcessState represents the current state of a Claude Code process
type ProcessState int

const (
	StateUnknown ProcessState = iota
	StateStarting
	StateIdle
	StateBusy
	StateWaiting
	StateError
	StateStopped
)

// String returns the string representation of ProcessState
func (s ProcessState) String() string {
	switch s {
	case StateUnknown:
		return "unknown"
	case StateStarting:
		return "starting"
	case StateIdle:
		return "idle"
	case StateBusy:
		return "busy"
	case StateWaiting:
		return "waiting"
	case StateError:
		return "error"
	case StateStopped:
		return "stopped"
	default:
		return "unknown"
	}
}

// ProcessInfo holds information about a Claude Code process
type ProcessInfo struct {
	PID         int          `json:"pid"`
	SessionID   string       `json:"session_id"`
	WorkingDir  string       `json:"working_dir"`
	Command     []string     `json:"command"`
	StartTime   time.Time    `json:"start_time"`
	State       ProcessState `json:"state"`
	LastUpdate  time.Time    `json:"last_update"`
	TmuxSession string       `json:"tmux_session,omitempty"`
	WorktreeID  string       `json:"worktree_id,omitempty"`
	CPUPercent  float64      `json:"cpu_percent"`
	MemoryMB    int64        `json:"memory_mb"`
	mutex       sync.RWMutex `json:"-"`
}

// GetState safely returns the current state
func (p *ProcessInfo) GetState() ProcessState {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.State
}

// SetState safely updates the process state
func (p *ProcessInfo) SetState(state ProcessState) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.State = state
	p.LastUpdate = time.Now()
}

// UpdateStats safely updates process statistics
func (p *ProcessInfo) UpdateStats(cpu float64, memory int64) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.CPUPercent = cpu
	p.MemoryMB = memory
	p.LastUpdate = time.Now()
}

// StateChangeEvent represents a process state change
type StateChangeEvent struct {
	ProcessID   string       `json:"process_id"`
	PID         int          `json:"pid"`
	OldState    ProcessState `json:"old_state"`
	NewState    ProcessState `json:"new_state"`
	Timestamp   time.Time    `json:"timestamp"`
	SessionID   string       `json:"session_id"`
	WorktreeID  string       `json:"worktree_id,omitempty"`
	TmuxSession string       `json:"tmux_session,omitempty"`
	WorkingDir  string       `json:"working_dir"`
}

// ProcessConfig holds configuration for process monitoring
type ProcessConfig struct {
	PollInterval             time.Duration                   `yaml:"poll_interval" json:"poll_interval"`
	LogPaths                 []string                        `yaml:"log_paths" json:"log_paths"`
	StatePatterns            map[string]string               `yaml:"state_patterns" json:"state_patterns"`
	MaxProcesses             int                             `yaml:"max_processes" json:"max_processes"`
	CleanupInterval          time.Duration                   `yaml:"cleanup_interval" json:"cleanup_interval"`
	EnableLogParsing         bool                            `yaml:"enable_log_parsing" json:"enable_log_parsing"`
	EnableResourceMonitoring bool                            `yaml:"enable_resource_monitoring" json:"enable_resource_monitoring"`
	StateTimeout             time.Duration                   `yaml:"state_timeout" json:"state_timeout"`
	StartupTimeout           time.Duration                   `yaml:"startup_timeout" json:"startup_timeout"`
	compiledPatterns         map[ProcessState]*regexp.Regexp `yaml:"-" json:"-"`
	mutex                    sync.RWMutex                    `yaml:"-" json:"-"`
}

// SetDefaults sets default values for ProcessConfig
func (c *ProcessConfig) SetDefaults() {
	if c.PollInterval == 0 {
		c.PollInterval = 3 * time.Second
	}
	if c.MaxProcesses == 0 {
		c.MaxProcesses = 10
	}
	if c.CleanupInterval == 0 {
		c.CleanupInterval = 5 * time.Minute
	}
	if c.StateTimeout == 0 {
		c.StateTimeout = 30 * time.Second
	}
	if c.StartupTimeout == 0 {
		c.StartupTimeout = 10 * time.Second
	}
	if len(c.LogPaths) == 0 {
		c.LogPaths = []string{
			"~/.claude/logs",
			"/tmp/claude-*",
		}
	}
	if len(c.StatePatterns) == 0 {
		c.StatePatterns = map[string]string{
			"busy":    `(?i)(Processing|Executing|Running|Working on|Analyzing|Generating)`,
			"idle":    `(?i)(Waiting for input|Ready|Idle|Available)`,
			"waiting": `(?i)(Waiting for confirmation|Press any key|Continue\?|Y/n)`,
			"error":   `(?i)(Error|Failed|Exception|Panic|Fatal)`,
		}
	}
	c.EnableLogParsing = true
	c.EnableResourceMonitoring = true
}

// CompilePatterns compiles regex patterns for state detection
func (c *ProcessConfig) CompilePatterns() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.compiledPatterns == nil {
		c.compiledPatterns = make(map[ProcessState]*regexp.Regexp)
	}

	stateMap := map[string]ProcessState{
		"busy":    StateBusy,
		"idle":    StateIdle,
		"waiting": StateWaiting,
		"error":   StateError,
	}

	for patternName, pattern := range c.StatePatterns {
		if state, exists := stateMap[patternName]; exists {
			compiled, err := regexp.Compile(pattern)
			if err != nil {
				return err
			}
			c.compiledPatterns[state] = compiled
		}
	}

	return nil
}

// GetCompiledPattern safely returns a compiled regex pattern
func (c *ProcessConfig) GetCompiledPattern(state ProcessState) *regexp.Regexp {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.compiledPatterns[state]
}

// StateChangeHandler defines the interface for handling state changes
type StateChangeHandler interface {
	OnStateChange(ctx context.Context, event StateChangeEvent) error
}

// ProcessDetector defines the interface for detecting Claude Code processes
type ProcessDetector interface {
	DetectProcesses(ctx context.Context) ([]*ProcessInfo, error)
	IsClaudeProcess(pid int) (bool, error)
	GetProcessInfo(pid int) (*ProcessInfo, error)
}

// StateMonitor defines the interface for monitoring process states
type StateMonitor interface {
	MonitorState(ctx context.Context, process *ProcessInfo) (ProcessState, error)
	Start(ctx context.Context) error
	Stop() error
}

// ProcessTracker defines the interface for tracking multiple processes
type ProcessTracker interface {
	AddProcess(process *ProcessInfo) error
	RemoveProcess(processID string) error
	GetProcess(processID string) (*ProcessInfo, bool)
	GetAllProcesses() []*ProcessInfo
	GetProcessesByState(state ProcessState) []*ProcessInfo
	GetProcessesByWorktree(worktreeID string) []*ProcessInfo
	Subscribe(handler StateChangeHandler) error
	Unsubscribe(handler StateChangeHandler) error
	Start(ctx context.Context) error
	Stop() error
}

// LogMonitor represents log file monitoring for state detection
type LogMonitor struct {
	LogPath    string                          `json:"log_path"`
	LastOffset int64                           `json:"last_offset"`
	LastCheck  time.Time                       `json:"last_check"`
	StateRegex map[ProcessState]*regexp.Regexp `json:"-"`
	ProcessID  string                          `json:"process_id"`
	mutex      sync.RWMutex                    `json:"-"`
}

// GetLastOffset safely returns the last read offset
func (l *LogMonitor) GetLastOffset() int64 {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	return l.LastOffset
}

// SetLastOffset safely updates the last read offset
func (l *LogMonitor) SetLastOffset(offset int64) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.LastOffset = offset
	l.LastCheck = time.Now()
}

// ProcessRegistry holds all tracked processes
type ProcessRegistry struct {
	processes   map[string]*ProcessInfo `json:"processes"`
	subscribers []StateChangeHandler    `json:"-"`
	config      *ProcessConfig          `json:"-"`
	mutex       sync.RWMutex            `json:"-"`
	stopCh      chan struct{}           `json:"-"`
	ctx         context.Context         `json:"-"`
	cancel      context.CancelFunc      `json:"-"`
}

// NewProcessRegistry creates a new process registry
func NewProcessRegistry(config *ProcessConfig) *ProcessRegistry {
	return &ProcessRegistry{
		processes:   make(map[string]*ProcessInfo),
		subscribers: make([]StateChangeHandler, 0),
		config:      config,
		stopCh:      make(chan struct{}),
	}
}

// ProcessStats holds statistics about monitored processes
type ProcessStats struct {
	TotalProcesses    int                  `json:"total_processes"`
	StateDistribution map[ProcessState]int `json:"state_distribution"`
	AverageUptime     time.Duration        `json:"average_uptime"`
	LastUpdated       time.Time            `json:"last_updated"`
}

// GetStats returns current statistics
func (r *ProcessRegistry) GetStats() ProcessStats {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	stats := ProcessStats{
		TotalProcesses:    len(r.processes),
		StateDistribution: make(map[ProcessState]int),
		LastUpdated:       time.Now(),
	}

	var totalUptime time.Duration
	for _, process := range r.processes {
		state := process.GetState()
		stats.StateDistribution[state]++
		totalUptime += time.Since(process.StartTime)
	}

	if len(r.processes) > 0 {
		stats.AverageUptime = totalUptime / time.Duration(len(r.processes))
	}

	return stats
}
