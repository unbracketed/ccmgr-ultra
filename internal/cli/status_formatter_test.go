package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

// Mock status data structures for testing
type MockStatusData struct {
	System    MockSystemStatus    `json:"system" yaml:"system"`
	Worktrees []MockWorktreeStatus `json:"worktrees" yaml:"worktrees"`
	Sessions  []MockSessionStatus  `json:"sessions" yaml:"sessions"`
	Processes []MockProcessStatus  `json:"processes" yaml:"processes"`
	Hooks     MockHookStatus      `json:"hooks" yaml:"hooks"`
	Timestamp time.Time           `json:"timestamp" yaml:"timestamp"`
}

type MockSystemStatus struct {
	Healthy               bool          `json:"healthy" yaml:"healthy"`
	TotalWorktrees        int           `json:"total_worktrees" yaml:"total_worktrees"`
	CleanWorktrees        int           `json:"clean_worktrees" yaml:"clean_worktrees"`
	DirtyWorktrees        int           `json:"dirty_worktrees" yaml:"dirty_worktrees"`
	ActiveSessions        int           `json:"active_sessions" yaml:"active_sessions"`
	TotalProcesses        int           `json:"total_processes" yaml:"total_processes"`
	HealthyProcesses      int           `json:"healthy_processes" yaml:"healthy_processes"`
	UnhealthyProcesses    int           `json:"unhealthy_processes" yaml:"unhealthy_processes"`
	ProcessManagerRunning bool          `json:"process_manager_running" yaml:"process_manager_running"`
	HooksEnabled          bool          `json:"hooks_enabled" yaml:"hooks_enabled"`
	AverageUptime         time.Duration `json:"average_uptime" yaml:"average_uptime"`
}

type MockWorktreeStatus struct {
	Path           string    `json:"path" yaml:"path"`
	Branch         string    `json:"branch" yaml:"branch"`
	Head           string    `json:"head" yaml:"head"`
	IsClean        bool      `json:"is_clean" yaml:"is_clean"`
	HasUncommitted bool      `json:"has_uncommitted" yaml:"has_uncommitted"`
	TmuxSession    string    `json:"tmux_session" yaml:"tmux_session"`
	LastAccessed   time.Time `json:"last_accessed" yaml:"last_accessed"`
	ProcessCount   int       `json:"process_count" yaml:"process_count"`
}

type MockSessionStatus struct {
	ID         string    `json:"id" yaml:"id"`
	Name       string    `json:"name" yaml:"name"`
	Project    string    `json:"project" yaml:"project"`
	Worktree   string    `json:"worktree" yaml:"worktree"`
	Branch     string    `json:"branch" yaml:"branch"`
	Directory  string    `json:"directory" yaml:"directory"`
	Active     bool      `json:"active" yaml:"active"`
	Created    time.Time `json:"created" yaml:"created"`
	LastAccess time.Time `json:"last_access" yaml:"last_access"`
}

type MockProcessStatus struct {
	PID         int       `json:"pid" yaml:"pid"`
	SessionID   string    `json:"session_id" yaml:"session_id"`
	WorkingDir  string    `json:"working_dir" yaml:"working_dir"`
	State       string    `json:"state" yaml:"state"`
	StartTime   time.Time `json:"start_time" yaml:"start_time"`
	Uptime      string    `json:"uptime" yaml:"uptime"`
	TmuxSession string    `json:"tmux_session" yaml:"tmux_session"`
	WorktreeID  string    `json:"worktree_id" yaml:"worktree_id"`
	CPUPercent  float64   `json:"cpu_percent" yaml:"cpu_percent"`
	MemoryMB    int64     `json:"memory_mb" yaml:"memory_mb"`
}

type MockHookStatus struct {
	Enabled bool `json:"enabled" yaml:"enabled"`
}

func TestStatusTableFormatter_Format(t *testing.T) {
	tests := []struct {
		name     string
		data     interface{}
		wantErr  bool
		contains []string
	}{
		{
			name: "complete status data",
			data: &MockStatusData{
				System: MockSystemStatus{
					Healthy:               true,
					TotalWorktrees:        2,
					CleanWorktrees:        1,
					DirtyWorktrees:        1,
					ActiveSessions:        1,
					TotalProcesses:        1,
					HealthyProcesses:      1,
					UnhealthyProcesses:    0,
					ProcessManagerRunning: true,
					HooksEnabled:          true,
					AverageUptime:         2 * time.Hour,
				},
				Worktrees: []MockWorktreeStatus{
					{
						Path:           "/path/to/worktree",
						Branch:         "main",
						Head:           "abcd1234",
						IsClean:        true,
						HasUncommitted: false,
						TmuxSession:    "session-1",
						LastAccessed:   time.Now().Add(-1 * time.Hour),
						ProcessCount:   1,
					},
				},
				Sessions: []MockSessionStatus{
					{
						ID:         "sess-1",
						Name:       "test-session",
						Project:    "test-project",
						Branch:     "main",
						Active:     true,
						Created:    time.Now().Add(-2 * time.Hour),
						LastAccess: time.Now().Add(-30 * time.Minute),
					},
				},
				Processes: []MockProcessStatus{
					{
						PID:         12345,
						State:       "idle",
						TmuxSession: "session-1",
						Uptime:      "2h30m",
						CPUPercent:  5.5,
						MemoryMB:    256,
						WorkingDir:  "/path/to/work",
					},
				},
				Hooks: MockHookStatus{
					Enabled: true,
				},
			},
			wantErr: false,
			contains: []string{
				"System Overview",
				"✓ Healthy",
				"Total Worktrees",
				"Worktrees",
				"Sessions",
				"Claude Processes",
				"Hooks Status",
				"main",
				"abcd1234",
				"✓ Clean",
				"💤 Idle",
				"12345",
				"5.5",
				"256 MB",
			},
		},
		{
			name: "empty data",
			data: &MockStatusData{
				System: MockSystemStatus{
					Healthy: false,
				},
				Hooks: MockHookStatus{
					Enabled: false,
				},
			},
			wantErr: false,
			contains: []string{
				"System Overview",
				"✗ Unhealthy",
				"Hooks Status",
				"✗",
			},
		},
		{
			name:    "invalid data type",
			data:    "invalid",
			wantErr: true,
		},
		{
			name:    "nil data",
			data:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			formatter := NewStatusTableFormatter(&buf)

			err := formatter.Format(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("StatusTableFormatter.Format() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			output := buf.String()
			for _, want := range tt.contains {
				if !strings.Contains(output, want) {
					t.Errorf("StatusTableFormatter.Format() output missing expected content: %q\nOutput:\n%s", want, output)
				}
			}
		})
	}
}

func TestFormatHelpers(t *testing.T) {
	tests := []struct {
		name     string
		function func() string
		expected string
	}{
		{
			name:     "formatHealthStatus true",
			function: func() string { return formatHealthStatus(true) },
			expected: "✓ Healthy",
		},
		{
			name:     "formatHealthStatus false",
			function: func() string { return formatHealthStatus(false) },
			expected: "✗ Unhealthy",
		},
		{
			name:     "formatBooleanStatus true",
			function: func() string { return formatBooleanStatus(true) },
			expected: "✓",
		},
		{
			name:     "formatBooleanStatus false",
			function: func() string { return formatBooleanStatus(false) },
			expected: "✗",
		},
		{
			name:     "formatWorktreeStatus clean",
			function: func() string { return formatWorktreeStatus(true, false) },
			expected: "✓ Clean",
		},
		{
			name:     "formatWorktreeStatus dirty",
			function: func() string { return formatWorktreeStatus(false, true) },
			expected: "⚠ Dirty",
		},
		{
			name:     "formatProcessState idle",
			function: func() string { return formatProcessState("idle") },
			expected: "💤 Idle",
		},
		{
			name:     "formatProcessState busy",
			function: func() string { return formatProcessState("busy") },
			expected: "🔄 Busy",
		},
		{
			name:     "formatDuration zero",
			function: func() string { return formatDuration(0) },
			expected: "0s",
		},
		{
			name:     "formatDuration hours",
			function: func() string { return formatDuration(2*time.Hour + 30*time.Minute) },
			expected: "2h 30m",
		},
		{
			name:     "formatMemory MB",
			function: func() string { return formatMemory(512) },
			expected: "512 MB",
		},
		{
			name:     "formatMemory GB",
			function: func() string { return formatMemory(2048) },
			expected: "2.0 GB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.function()
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestShortenPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		maxLen  int
		want    string
	}{
		{
			name:   "short path",
			path:   "/short/path",
			maxLen: 20,
			want:   "/short/path",
		},
		{
			name:   "long path shortened",
			path:   "/very/long/path/to/some/file.txt",
			maxLen: 15,
			want:   ".../file.txt",
		},
		{
			name:   "just filename too long",
			path:   "/path/verylongfilename.txt",
			maxLen: 10,
			want:   "verylon...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shortenPath(tt.path, tt.maxLen); got != tt.want {
				t.Errorf("shortenPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatTimeAgo(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name string
		time time.Time
		want string
	}{
		{
			name: "zero time",
			time: time.Time{},
			want: "Never",
		},
		{
			name: "just now",
			time: now.Add(-30 * time.Second),
			want: "Just now",
		},
		{
			name: "minutes ago",
			time: now.Add(-30 * time.Minute),
			want: "30m ago",
		},
		{
			name: "hours ago",
			time: now.Add(-2 * time.Hour),
			want: "2h ago",
		},
		{
			name: "days ago",
			time: now.Add(-25 * time.Hour),
			want: "1d ago",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatTimeAgo(tt.time); got != tt.want {
				t.Errorf("formatTimeAgo() = %v, want %v", got, tt.want)
			}
		})
	}
}