package cli

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

// BatchOperation represents a single operation in a batch
type BatchOperation struct {
	ID           string
	Description  string
	Target       string
	Operation    func(ctx context.Context) error
	Priority     int
	Dependencies []string
}

// BatchExecutor manages and executes batch operations
type BatchExecutor struct {
	operations      []*BatchOperation
	maxConcurrency  int
	continueOnError bool
	progressTracker *BatchProgressTracker
	mutex           sync.RWMutex
}

// BatchExecutorOptions configures batch execution behavior
type BatchExecutorOptions struct {
	MaxConcurrency  int
	ContinueOnError bool
	ShowProgress    bool
	Timeout         time.Duration
}

// BatchResult represents the result of a batch execution
type BatchResult struct {
	TotalOperations int
	SuccessCount    int
	FailureCount    int
	Errors          map[string]error
	Duration        time.Duration
	ExecutionOrder  []string
}

// NewBatchExecutor creates a new batch executor with the specified options
func NewBatchExecutor(opts *BatchExecutorOptions) *BatchExecutor {
	if opts == nil {
		opts = &BatchExecutorOptions{}
	}

	// Set defaults
	if opts.MaxConcurrency <= 0 {
		opts.MaxConcurrency = 4
	}

	return &BatchExecutor{
		operations:      make([]*BatchOperation, 0),
		maxConcurrency:  opts.MaxConcurrency,
		continueOnError: opts.ContinueOnError,
	}
}

// AddOperation adds an operation to the batch
func (be *BatchExecutor) AddOperation(op *BatchOperation) {
	be.mutex.Lock()
	defer be.mutex.Unlock()
	be.operations = append(be.operations, op)
}

// AddSimpleOperation adds a simple operation with auto-generated ID
func (be *BatchExecutor) AddSimpleOperation(description string, target string, operation func(ctx context.Context) error) {
	id := fmt.Sprintf("op_%d", len(be.operations)+1)
	be.AddOperation(&BatchOperation{
		ID:          id,
		Description: description,
		Target:      target,
		Operation:   operation,
		Priority:    0,
	})
}

// Execute runs all batch operations
func (be *BatchExecutor) Execute(ctx context.Context) (*BatchResult, error) {
	be.mutex.Lock()
	operations := make([]*BatchOperation, len(be.operations))
	copy(operations, be.operations)
	be.mutex.Unlock()

	if len(operations) == 0 {
		return &BatchResult{}, nil
	}

	// Initialize progress tracking
	be.progressTracker = NewBatchProgressTracker(len(operations))

	startTime := time.Now()
	result := &BatchResult{
		TotalOperations: len(operations),
		Errors:          make(map[string]error),
		ExecutionOrder:  make([]string, 0),
	}

	// Sort operations by priority and dependencies
	sortedOps := be.sortOperations(operations)

	// Execute operations with concurrency control
	err := be.executeWithConcurrency(ctx, sortedOps, result)

	result.Duration = time.Since(startTime)

	// Get final stats
	stats := be.progressTracker.GetStats()
	result.SuccessCount = stats.Completed
	result.FailureCount = stats.Failed

	return result, err
}

// executeWithConcurrency executes operations with controlled concurrency
func (be *BatchExecutor) executeWithConcurrency(ctx context.Context, operations []*BatchOperation, result *BatchResult) error {
	semaphore := make(chan struct{}, be.maxConcurrency)
	var wg sync.WaitGroup
	var mutex sync.Mutex

	for _, op := range operations {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		wg.Add(1)
		go func(operation *BatchOperation) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Execute operation
			err := operation.Operation(ctx)

			// Record result
			mutex.Lock()
			result.ExecutionOrder = append(result.ExecutionOrder, operation.ID)
			if err != nil {
				result.Errors[operation.ID] = err
				be.progressTracker.RecordFailure(err)
			} else {
				be.progressTracker.RecordSuccess()
			}
			mutex.Unlock()

			// Check if we should continue on error
			if err != nil && !be.continueOnError {
				// Cancel context to stop other operations
				// Note: This would require a cancellable context
			}
		}(op)
	}

	wg.Wait()
	return nil
}

// sortOperations sorts operations by priority and resolves dependencies
func (be *BatchExecutor) sortOperations(operations []*BatchOperation) []*BatchOperation {
	// Simple priority-based sorting for now
	// Real implementation would handle dependency resolution
	sorted := make([]*BatchOperation, len(operations))
	copy(sorted, operations)

	// Sort by priority (higher priority first)
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i].Priority < sorted[j].Priority {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	return sorted
}

// GetProgress returns the current batch progress
func (be *BatchExecutor) GetProgress() *BatchProgressTracker {
	return be.progressTracker
}

// PatternMatcher provides utilities for pattern-based filtering
type PatternMatcher struct {
	patterns []string
	compiled []*regexp.Regexp
}

// NewPatternMatcher creates a new pattern matcher with the specified patterns
func NewPatternMatcher(patterns []string) (*PatternMatcher, error) {
	pm := &PatternMatcher{
		patterns: patterns,
		compiled: make([]*regexp.Regexp, 0, len(patterns)),
	}

	for _, pattern := range patterns {
		// Convert glob pattern to regex
		regex, err := globToRegex(pattern)
		if err != nil {
			return nil, NewErrorWithCause(fmt.Sprintf("invalid pattern: %s", pattern), err)
		}

		compiled, err := regexp.Compile(regex)
		if err != nil {
			return nil, NewErrorWithCause(fmt.Sprintf("failed to compile pattern: %s", pattern), err)
		}

		pm.compiled = append(pm.compiled, compiled)
	}

	return pm, nil
}

// Match returns true if the target matches any of the patterns
func (pm *PatternMatcher) Match(target string) bool {
	if len(pm.compiled) == 0 {
		return true // No patterns means match all
	}

	for _, regex := range pm.compiled {
		if regex.MatchString(target) {
			return true
		}
	}

	return false
}

// Filter filters a slice of strings based on the patterns
func (pm *PatternMatcher) Filter(targets []string) []string {
	if len(pm.compiled) == 0 {
		return targets // No patterns means return all
	}

	filtered := make([]string, 0)
	for _, target := range targets {
		if pm.Match(target) {
			filtered = append(filtered, target)
		}
	}

	return filtered
}

// globToRegex converts a glob pattern to a regular expression
func globToRegex(pattern string) (string, error) {
	// Escape regex special characters except * and ?
	escaped := regexp.QuoteMeta(pattern)

	// Replace escaped glob characters with regex equivalents
	escaped = strings.ReplaceAll(escaped, "\\*", ".*")
	escaped = strings.ReplaceAll(escaped, "\\?", ".")

	// Anchor the pattern
	return "^" + escaped + "$", nil
}

// BatchFilter provides filtering capabilities for batch operations
type BatchFilter struct {
	namePattern    *PatternMatcher
	targetPattern  *PatternMatcher
	minPriority    int
	maxPriority    int
	includeTargets []string
	excludeTargets []string
}

// BatchFilterOptions configures batch filtering
type BatchFilterOptions struct {
	NamePatterns   []string
	TargetPatterns []string
	MinPriority    int
	MaxPriority    int
	IncludeTargets []string
	ExcludeTargets []string
}

// NewBatchFilter creates a new batch filter with the specified options
func NewBatchFilter(opts *BatchFilterOptions) (*BatchFilter, error) {
	if opts == nil {
		return &BatchFilter{}, nil
	}

	bf := &BatchFilter{
		minPriority:    opts.MinPriority,
		maxPriority:    opts.MaxPriority,
		includeTargets: opts.IncludeTargets,
		excludeTargets: opts.ExcludeTargets,
	}

	// Set up name pattern matcher
	if len(opts.NamePatterns) > 0 {
		matcher, err := NewPatternMatcher(opts.NamePatterns)
		if err != nil {
			return nil, NewErrorWithCause("failed to create name pattern matcher", err)
		}
		bf.namePattern = matcher
	}

	// Set up target pattern matcher
	if len(opts.TargetPatterns) > 0 {
		matcher, err := NewPatternMatcher(opts.TargetPatterns)
		if err != nil {
			return nil, NewErrorWithCause("failed to create target pattern matcher", err)
		}
		bf.targetPattern = matcher
	}

	return bf, nil
}

// Filter filters operations based on the configured criteria
func (bf *BatchFilter) Filter(operations []*BatchOperation) []*BatchOperation {
	if bf == nil {
		return operations
	}

	filtered := make([]*BatchOperation, 0)

	for _, op := range operations {
		if bf.shouldInclude(op) {
			filtered = append(filtered, op)
		}
	}

	return filtered
}

// shouldInclude determines if an operation should be included based on filter criteria
func (bf *BatchFilter) shouldInclude(op *BatchOperation) bool {
	// Check priority range
	if bf.maxPriority > 0 && op.Priority > bf.maxPriority {
		return false
	}
	if bf.minPriority > 0 && op.Priority < bf.minPriority {
		return false
	}

	// Check name patterns
	if bf.namePattern != nil && !bf.namePattern.Match(op.ID) {
		return false
	}

	// Check target patterns
	if bf.targetPattern != nil && !bf.targetPattern.Match(op.Target) {
		return false
	}

	// Check explicit includes
	if len(bf.includeTargets) > 0 {
		found := false
		for _, target := range bf.includeTargets {
			if op.Target == target {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check explicit excludes
	for _, target := range bf.excludeTargets {
		if op.Target == target {
			return false
		}
	}

	return true
}

// BatchValidator validates operations before execution
type BatchValidator struct {
	checks []ValidationCheck
}

// ValidationCheck represents a single validation check
type ValidationCheck struct {
	Name        string
	Description string
	Check       func(op *BatchOperation) error
	Critical    bool
}

// NewBatchValidator creates a new batch validator
func NewBatchValidator() *BatchValidator {
	return &BatchValidator{
		checks: make([]ValidationCheck, 0),
	}
}

// AddCheck adds a validation check
func (bv *BatchValidator) AddCheck(check ValidationCheck) {
	bv.checks = append(bv.checks, check)
}

// Validate validates all operations in a batch
func (bv *BatchValidator) Validate(operations []*BatchOperation) []ValidationError {
	var errors []ValidationError

	for _, op := range operations {
		for _, check := range bv.checks {
			if err := check.Check(op); err != nil {
				errors = append(errors, ValidationError{
					OperationID: op.ID,
					CheckName:   check.Name,
					Error:       err,
					Critical:    check.Critical,
				})
			}
		}
	}

	return errors
}

// ValidationError represents a validation error
type ValidationError struct {
	OperationID string
	CheckName   string
	Error       error
	Critical    bool
}

// String returns a string representation of the validation error
func (ve ValidationError) String() string {
	criticality := "WARNING"
	if ve.Critical {
		criticality = "ERROR"
	}
	return fmt.Sprintf("[%s] %s (%s): %s", criticality, ve.OperationID, ve.CheckName, ve.Error.Error())
}

// HasCriticalErrors returns true if there are any critical validation errors
func HasCriticalErrors(errors []ValidationError) bool {
	for _, err := range errors {
		if err.Critical {
			return true
		}
	}
	return false
}

// GroupValidationErrors groups validation errors by operation ID
func GroupValidationErrors(errors []ValidationError) map[string][]ValidationError {
	grouped := make(map[string][]ValidationError)

	for _, err := range errors {
		grouped[err.OperationID] = append(grouped[err.OperationID], err)
	}

	return grouped
}
