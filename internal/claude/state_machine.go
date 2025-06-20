package claude

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// StateTransition represents a state transition
type StateTransition struct {
	From      ProcessState `json:"from"`
	To        ProcessState `json:"to"`
	Trigger   string       `json:"trigger"`
	Timestamp time.Time    `json:"timestamp"`
	ProcessID string       `json:"process_id"`
}

// StateTransitionRule defines rules for valid state transitions
type StateTransitionRule struct {
	From        ProcessState  `json:"from"`
	To          ProcessState  `json:"to"`
	Conditions  []string      `json:"conditions"`
	MinDuration time.Duration `json:"min_duration"`
	MaxDuration time.Duration `json:"max_duration"`
}

// StateMachine manages process state transitions and validation
type StateMachine struct {
	rules       []StateTransitionRule
	transitions map[string][]StateTransition // processID -> transitions
	validators  []StateValidator
	mutex       sync.RWMutex
	config      *ProcessConfig
}

// StateValidator defines an interface for validating state transitions
type StateValidator interface {
	ValidateTransition(ctx context.Context, from, to ProcessState, process *ProcessInfo) error
}

// NewStateMachine creates a new state machine with default rules
func NewStateMachine(config *ProcessConfig) *StateMachine {
	sm := &StateMachine{
		rules:       getDefaultTransitionRules(),
		transitions: make(map[string][]StateTransition),
		validators:  make([]StateValidator, 0),
		config:      config,
	}

	// Add default validators
	sm.validators = append(sm.validators, &DefaultStateValidator{config: config})

	return sm
}

// ValidateTransition checks if a state transition is valid
func (sm *StateMachine) ValidateTransition(ctx context.Context, processID string, from, to ProcessState, process *ProcessInfo) error {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	// Check if transition is allowed by rules
	if !sm.isTransitionAllowed(from, to) {
		return fmt.Errorf("transition from %s to %s is not allowed", from.String(), to.String())
	}

	// Check minimum duration constraints
	if err := sm.checkMinimumDuration(processID, from, to); err != nil {
		return fmt.Errorf("duration constraint violation: %w", err)
	}

	// Run all validators
	for _, validator := range sm.validators {
		if err := validator.ValidateTransition(ctx, from, to, process); err != nil {
			return fmt.Errorf("validation failed: %w", err)
		}
	}

	return nil
}

// RecordTransition records a successful state transition
func (sm *StateMachine) RecordTransition(processID string, from, to ProcessState, trigger string) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	transition := StateTransition{
		From:      from,
		To:        to,
		Trigger:   trigger,
		Timestamp: time.Now(),
		ProcessID: processID,
	}

	if sm.transitions[processID] == nil {
		sm.transitions[processID] = make([]StateTransition, 0)
	}

	sm.transitions[processID] = append(sm.transitions[processID], transition)

	// Keep only recent transitions (last 100 per process)
	if len(sm.transitions[processID]) > 100 {
		sm.transitions[processID] = sm.transitions[processID][len(sm.transitions[processID])-100:]
	}
}

// GetTransitionHistory returns the transition history for a process
func (sm *StateMachine) GetTransitionHistory(processID string) []StateTransition {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	if transitions, exists := sm.transitions[processID]; exists {
		// Return a copy to prevent external modification
		result := make([]StateTransition, len(transitions))
		copy(result, transitions)
		return result
	}

	return nil
}

// GetRecentTransitions returns transitions within a time window
func (sm *StateMachine) GetRecentTransitions(processID string, since time.Time) []StateTransition {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	var recent []StateTransition
	if transitions, exists := sm.transitions[processID]; exists {
		for _, transition := range transitions {
			if transition.Timestamp.After(since) {
				recent = append(recent, transition)
			}
		}
	}

	return recent
}

// GetLastTransition returns the most recent transition for a process
func (sm *StateMachine) GetLastTransition(processID string) *StateTransition {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	if transitions, exists := sm.transitions[processID]; exists && len(transitions) > 0 {
		return &transitions[len(transitions)-1]
	}

	return nil
}

// GetStateMetrics returns metrics about state transitions
func (sm *StateMachine) GetStateMetrics(processID string) *StateMetrics {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	metrics := &StateMetrics{
		ProcessID:        processID,
		TotalTransitions: 0,
		StateDistribution: make(map[ProcessState]int),
		TransitionCounts:  make(map[string]int),
		AverageStateDuration: make(map[ProcessState]time.Duration),
	}

	transitions, exists := sm.transitions[processID]
	if !exists || len(transitions) == 0 {
		return metrics
	}

	metrics.TotalTransitions = len(transitions)
	
	// Calculate state distribution and transition counts
	stateDurations := make(map[ProcessState]time.Duration)
	stateCount := make(map[ProcessState]int)

	for i, transition := range transitions {
		// Count transitions
		transitionKey := fmt.Sprintf("%s->%s", transition.From.String(), transition.To.String())
		metrics.TransitionCounts[transitionKey]++

		// Calculate state duration (time spent in 'from' state)
		if i > 0 {
			duration := transition.Timestamp.Sub(transitions[i-1].Timestamp)
			stateDurations[transition.From] += duration
			stateCount[transition.From]++
		}

		metrics.StateDistribution[transition.To]++
	}

	// Calculate average durations
	for state, totalDuration := range stateDurations {
		if count := stateCount[state]; count > 0 {
			metrics.AverageStateDuration[state] = totalDuration / time.Duration(count)
		}
	}

	// Set first and last transition times
	if len(transitions) > 0 {
		metrics.FirstTransition = transitions[0].Timestamp
		metrics.LastTransition = transitions[len(transitions)-1].Timestamp
	}

	return metrics
}

// isTransitionAllowed checks if a transition is allowed by the rules
func (sm *StateMachine) isTransitionAllowed(from, to ProcessState) bool {
	// Same state is always allowed
	if from == to {
		return true
	}

	for _, rule := range sm.rules {
		if rule.From == from && rule.To == to {
			return true
		}
	}

	return false
}

// checkMinimumDuration ensures minimum time constraints are met
func (sm *StateMachine) checkMinimumDuration(processID string, from, to ProcessState) error {
	if from == to {
		return nil // Same state transitions don't have duration constraints
	}

	// Find the rule for this transition
	var rule *StateTransitionRule
	for _, r := range sm.rules {
		if r.From == from && r.To == to {
			rule = &r
			break
		}
	}

	if rule == nil || rule.MinDuration == 0 {
		return nil // No rule or no minimum duration constraint
	}

	// Get the last transition to this state
	lastTransition := sm.GetLastTransition(processID)
	if lastTransition == nil {
		return nil // No previous transitions
	}

	// Check if enough time has passed
	timeSinceLastTransition := time.Since(lastTransition.Timestamp)
	if timeSinceLastTransition < rule.MinDuration {
		return fmt.Errorf("minimum duration not met: %s < %s", 
			timeSinceLastTransition, rule.MinDuration)
	}

	return nil
}

// AddValidator adds a custom state validator
func (sm *StateMachine) AddValidator(validator StateValidator) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.validators = append(sm.validators, validator)
}

// AddTransitionRule adds a custom transition rule
func (sm *StateMachine) AddTransitionRule(rule StateTransitionRule) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.rules = append(sm.rules, rule)
}

// CleanupOldTransitions removes old transition records
func (sm *StateMachine) CleanupOldTransitions(maxAge time.Duration) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	cutoff := time.Now().Add(-maxAge)
	
	for processID, transitions := range sm.transitions {
		var filtered []StateTransition
		for _, transition := range transitions {
			if transition.Timestamp.After(cutoff) {
				filtered = append(filtered, transition)
			}
		}
		
		if len(filtered) == 0 {
			delete(sm.transitions, processID)
		} else {
			sm.transitions[processID] = filtered
		}
	}
}

// StateMetrics contains metrics about state transitions for a process
type StateMetrics struct {
	ProcessID            string                      `json:"process_id"`
	TotalTransitions     int                         `json:"total_transitions"`
	StateDistribution    map[ProcessState]int        `json:"state_distribution"`
	TransitionCounts     map[string]int              `json:"transition_counts"`
	AverageStateDuration map[ProcessState]time.Duration `json:"average_state_duration"`
	FirstTransition      time.Time                   `json:"first_transition"`
	LastTransition       time.Time                   `json:"last_transition"`
}

// DefaultStateValidator provides basic state transition validation
type DefaultStateValidator struct {
	config *ProcessConfig
}

// ValidateTransition implements StateValidator interface
func (v *DefaultStateValidator) ValidateTransition(ctx context.Context, from, to ProcessState, process *ProcessInfo) error {
	// Validate based on process age
	if from == StateStarting && to != StateIdle && to != StateBusy && to != StateError {
		if time.Since(process.StartTime) < v.config.StartupTimeout {
			return fmt.Errorf("process is still starting up")
		}
	}

	// Validate based on last update time
	if time.Since(process.LastUpdate) > v.config.StateTimeout {
		if to != StateError && to != StateStopped {
			return fmt.Errorf("process hasn't been updated recently, may be unresponsive")
		}
	}

	// Validate resource-based transitions
	if to == StateBusy && process.CPUPercent < 1.0 {
		return fmt.Errorf("CPU usage too low for busy state: %.2f%%", process.CPUPercent)
	}

	return nil
}

// getDefaultTransitionRules returns the default state transition rules
func getDefaultTransitionRules() []StateTransitionRule {
	return []StateTransitionRule{
		// From Unknown
		{From: StateUnknown, To: StateStarting, MinDuration: 0},
		{From: StateUnknown, To: StateIdle, MinDuration: 0},
		{From: StateUnknown, To: StateBusy, MinDuration: 0},
		{From: StateUnknown, To: StateError, MinDuration: 0},
		{From: StateUnknown, To: StateStopped, MinDuration: 0},

		// From Starting
		{From: StateStarting, To: StateIdle, MinDuration: time.Second},
		{From: StateStarting, To: StateBusy, MinDuration: time.Second},
		{From: StateStarting, To: StateWaiting, MinDuration: time.Second},
		{From: StateStarting, To: StateError, MinDuration: 0},
		{From: StateStarting, To: StateStopped, MinDuration: 0},

		// From Idle
		{From: StateIdle, To: StateBusy, MinDuration: 0},
		{From: StateIdle, To: StateWaiting, MinDuration: 0},
		{From: StateIdle, To: StateError, MinDuration: 0},
		{From: StateIdle, To: StateStopped, MinDuration: 0},

		// From Busy
		{From: StateBusy, To: StateIdle, MinDuration: time.Second},
		{From: StateBusy, To: StateWaiting, MinDuration: 0},
		{From: StateBusy, To: StateError, MinDuration: 0},
		{From: StateBusy, To: StateStopped, MinDuration: 0},

		// From Waiting
		{From: StateWaiting, To: StateIdle, MinDuration: 0},
		{From: StateWaiting, To: StateBusy, MinDuration: 0},
		{From: StateWaiting, To: StateError, MinDuration: 0},
		{From: StateWaiting, To: StateStopped, MinDuration: 0},

		// From Error
		{From: StateError, To: StateIdle, MinDuration: 2 * time.Second},
		{From: StateError, To: StateStarting, MinDuration: time.Second},
		{From: StateError, To: StateStopped, MinDuration: 0},

		// From Stopped - only allowed to go back to starting or stay stopped
		{From: StateStopped, To: StateStarting, MinDuration: 0},
	}
}