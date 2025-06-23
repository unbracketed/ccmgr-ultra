package tmux

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bcdekker/ccmgr-ultra/internal/config"
)

type ProcessState int

const (
	StateUnknown ProcessState = iota
	StateIdle
	StateBusy
	StateWaiting
	StateError
)

func (s ProcessState) String() string {
	switch s {
	case StateUnknown:
		return "unknown"
	case StateIdle:
		return "idle"
	case StateBusy:
		return "busy"
	case StateWaiting:
		return "waiting"
	case StateError:
		return "error"
	default:
		return "unknown"
	}
}

type ProcessMonitor struct {
	sessions     map[string]*MonitoredSession
	stateHooks   []StateHook
	pollInterval time.Duration
	tmux         TmuxInterface
	mutex        sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
}

type MonitoredSession struct {
	SessionID       string
	ProcessPID      int
	CurrentState    ProcessState
	LastStateChange time.Time
	StateHistory    []StateChange
}

type StateChange struct {
	From      ProcessState
	To        ProcessState
	Timestamp time.Time
	Trigger   string
}

type StateHook interface {
	OnStateChange(sessionID string, from, to ProcessState) error
}

type StatePattern struct {
	Pattern    string
	State      ProcessState
	Confidence float64
	regex      *regexp.Regexp
}

var statePatterns = []StatePattern{
	{"claude>", StateIdle, 0.9, nil},
	{"Processing\\.\\.\\.", StateBusy, 0.8, nil},
	{"Waiting for input", StateWaiting, 0.9, nil},
	{"Error:", StateError, 0.95, nil},
	{"Exception:", StateError, 0.95, nil},
	{"Failed:", StateError, 0.9, nil},
	{"Loading\\.\\.\\.", StateBusy, 0.7, nil},
	{"Generating", StateBusy, 0.8, nil},
	{"Analyzing", StateBusy, 0.8, nil},
}

func init() {
	for i := range statePatterns {
		statePatterns[i].regex = regexp.MustCompile(statePatterns[i].Pattern)
	}
}

func NewProcessMonitor(config *config.Config) *ProcessMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	pollInterval := 2 * time.Second
	if config != nil && config.Tmux.MonitorInterval > 0 {
		pollInterval = config.Tmux.MonitorInterval
	}

	return &ProcessMonitor{
		sessions:     make(map[string]*MonitoredSession),
		stateHooks:   make([]StateHook, 0),
		pollInterval: pollInterval,
		tmux:         NewTmuxCmd(),
		ctx:          ctx,
		cancel:       cancel,
	}
}

func (pm *ProcessMonitor) StartMonitoring(sessionID string) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if _, exists := pm.sessions[sessionID]; exists {
		return fmt.Errorf("session %s is already being monitored", sessionID)
	}

	pid, err := pm.getProcessPID(sessionID)
	if err != nil {
		return fmt.Errorf("failed to get process PID for session %s: %w", sessionID, err)
	}

	session := &MonitoredSession{
		SessionID:       sessionID,
		ProcessPID:      pid,
		CurrentState:    StateUnknown,
		LastStateChange: time.Now(),
		StateHistory:    make([]StateChange, 0),
	}

	pm.sessions[sessionID] = session

	go pm.monitorSession(sessionID)

	return nil
}

func (pm *ProcessMonitor) StopMonitoring(sessionID string) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if _, exists := pm.sessions[sessionID]; !exists {
		return fmt.Errorf("session %s is not being monitored", sessionID)
	}

	delete(pm.sessions, sessionID)
	return nil
}

func (pm *ProcessMonitor) GetProcessState(sessionID string) (ProcessState, error) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	session, exists := pm.sessions[sessionID]
	if !exists {
		return StateUnknown, fmt.Errorf("session %s is not being monitored", sessionID)
	}

	return session.CurrentState, nil
}

func (pm *ProcessMonitor) GetProcessPID(sessionID string) (int, error) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	session, exists := pm.sessions[sessionID]
	if !exists {
		return 0, fmt.Errorf("session %s is not being monitored", sessionID)
	}

	return session.ProcessPID, nil
}

func (pm *ProcessMonitor) RegisterStateHook(hook StateHook) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.stateHooks = append(pm.stateHooks, hook)
}

func (pm *ProcessMonitor) DetectStateChange(sessionID string) (bool, ProcessState, error) {
	newState, err := pm.detectStateCombined(sessionID)
	if err != nil {
		return false, StateUnknown, fmt.Errorf("failed to detect state: %w", err)
	}

	pm.mutex.RLock()
	session, exists := pm.sessions[sessionID]
	pm.mutex.RUnlock()

	if !exists {
		return false, StateUnknown, fmt.Errorf("session %s is not being monitored", sessionID)
	}

	if session.CurrentState != newState {
		pm.updateSessionState(sessionID, newState, "state_detection")
		return true, newState, nil
	}

	return false, newState, nil
}

func (pm *ProcessMonitor) monitorSession(sessionID string) {
	ticker := time.NewTicker(pm.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-pm.ctx.Done():
			return
		case <-ticker.C:
			pm.mutex.RLock()
			_, exists := pm.sessions[sessionID]
			pm.mutex.RUnlock()

			if !exists {
				return
			}

			changed, newState, err := pm.DetectStateChange(sessionID)
			if err != nil {
				continue
			}

			if changed {
				pm.executeHooks(sessionID, pm.sessions[sessionID].CurrentState, newState)
			}
		}
	}
}

func (pm *ProcessMonitor) updateSessionState(sessionID string, newState ProcessState, trigger string) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	session, exists := pm.sessions[sessionID]
	if !exists {
		return
	}

	oldState := session.CurrentState

	stateChange := StateChange{
		From:      oldState,
		To:        newState,
		Timestamp: time.Now(),
		Trigger:   trigger,
	}

	session.CurrentState = newState
	session.LastStateChange = stateChange.Timestamp
	session.StateHistory = append(session.StateHistory, stateChange)

	if len(session.StateHistory) > 100 {
		session.StateHistory = session.StateHistory[1:]
	}
}

func (pm *ProcessMonitor) executeHooks(sessionID string, from, to ProcessState) {
	pm.mutex.RLock()
	hooks := make([]StateHook, len(pm.stateHooks))
	copy(hooks, pm.stateHooks)
	pm.mutex.RUnlock()

	for _, hook := range hooks {
		if err := hook.OnStateChange(sessionID, from, to); err != nil {
			continue
		}
	}
}

func (pm *ProcessMonitor) detectStateCombined(sessionID string) (ProcessState, error) {
	outputState, outputErr := pm.detectStateByOutput(sessionID)
	if outputErr == nil && outputState != StateUnknown {
		return outputState, nil
	}

	pid, err := pm.getProcessPID(sessionID)
	if err != nil {
		return StateUnknown, fmt.Errorf("failed to get PID: %w", err)
	}

	processState, processErr := pm.detectStateByProcess(pid)
	if processErr == nil && processState != StateUnknown {
		return processState, nil
	}

	if outputErr == nil {
		return outputState, nil
	}

	if processErr == nil {
		return processState, nil
	}

	return StateUnknown, fmt.Errorf("all detection methods failed: output=%v, process=%v", outputErr, processErr)
}

func (pm *ProcessMonitor) detectStateByProcess(pid int) (ProcessState, error) {
	if pid <= 0 {
		return StateUnknown, fmt.Errorf("invalid PID: %d", pid)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ps", "-p", strconv.Itoa(pid), "-o", "state=")
	output, err := cmd.Output()
	if err != nil {
		return StateUnknown, fmt.Errorf("failed to get process state: %w", err)
	}

	state := strings.TrimSpace(string(output))
	switch state {
	case "R", "R+":
		return StateBusy, nil
	case "S", "S+":
		return StateIdle, nil
	case "D":
		return StateWaiting, nil
	case "Z":
		return StateError, nil
	default:
		return StateUnknown, nil
	}
}

func (pm *ProcessMonitor) detectStateByOutput(sessionID string) (ProcessState, error) {
	panes, err := pm.tmux.GetSessionPanes(sessionID)
	if err != nil {
		return StateUnknown, fmt.Errorf("failed to get session panes: %w", err)
	}

	if len(panes) == 0 {
		return StateUnknown, fmt.Errorf("no panes found for session %s", sessionID)
	}

	output, err := pm.tmux.CapturePane(sessionID, panes[0])
	if err != nil {
		return StateUnknown, fmt.Errorf("failed to capture pane output: %w", err)
	}

	return pm.analyzeOutput(output), nil
}

func (pm *ProcessMonitor) analyzeOutput(output string) ProcessState {
	lines := strings.Split(output, "\n")

	recentLines := lines
	if len(lines) > 20 {
		recentLines = lines[len(lines)-20:]
	}

	recentOutput := strings.Join(recentLines, "\n")

	bestMatch := StateUnknown
	bestConfidence := 0.0

	for _, pattern := range statePatterns {
		if pattern.regex.MatchString(recentOutput) {
			if pattern.Confidence > bestConfidence {
				bestMatch = pattern.State
				bestConfidence = pattern.Confidence
			}
		}
	}

	return bestMatch
}

func (pm *ProcessMonitor) getProcessPID(sessionID string) (int, error) {
	panes, err := pm.tmux.GetSessionPanes(sessionID)
	if err != nil {
		return 0, fmt.Errorf("failed to get session panes: %w", err)
	}

	if len(panes) == 0 {
		return 0, fmt.Errorf("no panes found for session %s", sessionID)
	}

	return pm.tmux.GetPanePID(sessionID, panes[0])
}

func (pm *ProcessMonitor) Shutdown() {
	pm.cancel()
}
