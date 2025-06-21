package cli

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// ProgressBar represents a progress bar with customizable styling
type ProgressBar struct {
	total       int
	current     int
	width       int
	prefix      string
	suffix      string
	fill        string
	empty       string
	showPercent bool
	showRate    bool
	startTime   time.Time
	mutex       sync.RWMutex
}

// ProgressBarOptions configures progress bar behavior and appearance
type ProgressBarOptions struct {
	Width       int
	Prefix      string
	Suffix      string
	Fill        string
	Empty       string
	ShowPercent bool
	ShowRate    bool
}

// NewProgressBar creates a new progress bar with the specified total and options
func NewProgressBar(total int, opts *ProgressBarOptions) *ProgressBar {
	if opts == nil {
		opts = &ProgressBarOptions{}
	}

	// Set defaults
	if opts.Width == 0 {
		opts.Width = 40
	}
	if opts.Fill == "" {
		opts.Fill = "█"
	}
	if opts.Empty == "" {
		opts.Empty = "░"
	}

	return &ProgressBar{
		total:       total,
		current:     0,
		width:       opts.Width,
		prefix:      opts.Prefix,
		suffix:      opts.Suffix,
		fill:        opts.Fill,
		empty:       opts.Empty,
		showPercent: opts.ShowPercent,
		showRate:    opts.ShowRate,
		startTime:   time.Now(),
	}
}

// Increment advances the progress bar by one unit
func (pb *ProgressBar) Increment() {
	pb.Add(1)
}

// Add advances the progress bar by the specified amount
func (pb *ProgressBar) Add(amount int) {
	pb.mutex.Lock()
	defer pb.mutex.Unlock()
	
	pb.current += amount
	if pb.current > pb.total {
		pb.current = pb.total
	}
}

// Set sets the current progress to the specified value
func (pb *ProgressBar) Set(current int) {
	pb.mutex.Lock()
	defer pb.mutex.Unlock()
	
	pb.current = current
	if pb.current > pb.total {
		pb.current = pb.total
	}
	if pb.current < 0 {
		pb.current = 0
	}
}

// SetPrefix updates the progress bar prefix
func (pb *ProgressBar) SetPrefix(prefix string) {
	pb.mutex.Lock()
	defer pb.mutex.Unlock()
	pb.prefix = prefix
}

// SetSuffix updates the progress bar suffix
func (pb *ProgressBar) SetSuffix(suffix string) {
	pb.mutex.Lock()
	defer pb.mutex.Unlock()
	pb.suffix = suffix
}

// Render returns the current progress bar as a string
func (pb *ProgressBar) Render() string {
	pb.mutex.RLock()
	defer pb.mutex.RUnlock()
	
	percentage := float64(pb.current) / float64(pb.total) * 100
	if pb.total == 0 {
		percentage = 0
	}
	
	filled := int(float64(pb.width) * float64(pb.current) / float64(pb.total))
	if filled > pb.width {
		filled = pb.width
	}
	
	bar := strings.Repeat(pb.fill, filled) + strings.Repeat(pb.empty, pb.width-filled)
	
	result := ""
	
	if pb.prefix != "" {
		result += pb.prefix + " "
	}
	
	result += fmt.Sprintf("[%s]", bar)
	
	if pb.showPercent {
		result += fmt.Sprintf(" %.1f%%", percentage)
	}
	
	if pb.showRate {
		elapsed := time.Since(pb.startTime)
		if elapsed > 0 {
			rate := float64(pb.current) / elapsed.Seconds()
			result += fmt.Sprintf(" (%.1f/s)", rate)
		}
	}
	
	result += fmt.Sprintf(" %d/%d", pb.current, pb.total)
	
	if pb.suffix != "" {
		result += " " + pb.suffix
	}
	
	return result
}

// IsComplete returns true if the progress bar has reached 100%
func (pb *ProgressBar) IsComplete() bool {
	pb.mutex.RLock()
	defer pb.mutex.RUnlock()
	return pb.current >= pb.total
}

// GetProgress returns the current progress as a percentage (0-100)
func (pb *ProgressBar) GetProgress() float64 {
	pb.mutex.RLock()
	defer pb.mutex.RUnlock()
	
	if pb.total == 0 {
		return 0
	}
	return float64(pb.current) / float64(pb.total) * 100
}

// MultiProgressBar manages multiple progress bars simultaneously
type MultiProgressBar struct {
	bars    map[string]*ProgressBar
	labels  map[string]string
	order   []string
	mutex   sync.RWMutex
}

// NewMultiProgressBar creates a new multi-progress bar manager
func NewMultiProgressBar() *MultiProgressBar {
	return &MultiProgressBar{
		bars:   make(map[string]*ProgressBar),
		labels: make(map[string]string),
		order:  make([]string, 0),
	}
}

// AddBar adds a new progress bar with the specified ID and options
func (mpb *MultiProgressBar) AddBar(id string, label string, total int, opts *ProgressBarOptions) {
	mpb.mutex.Lock()
	defer mpb.mutex.Unlock()
	
	mpb.bars[id] = NewProgressBar(total, opts)
	mpb.labels[id] = label
	mpb.order = append(mpb.order, id)
}

// GetBar returns the progress bar with the specified ID
func (mpb *MultiProgressBar) GetBar(id string) *ProgressBar {
	mpb.mutex.RLock()
	defer mpb.mutex.RUnlock()
	
	return mpb.bars[id]
}

// RemoveBar removes the progress bar with the specified ID
func (mpb *MultiProgressBar) RemoveBar(id string) {
	mpb.mutex.Lock()
	defer mpb.mutex.Unlock()
	
	delete(mpb.bars, id)
	delete(mpb.labels, id)
	
	// Remove from order slice
	for i, orderID := range mpb.order {
		if orderID == id {
			mpb.order = append(mpb.order[:i], mpb.order[i+1:]...)
			break
		}
	}
}

// Render returns all progress bars as a multi-line string
func (mpb *MultiProgressBar) Render() string {
	mpb.mutex.RLock()
	defer mpb.mutex.RUnlock()
	
	var lines []string
	
	for _, id := range mpb.order {
		bar := mpb.bars[id]
		label := mpb.labels[id]
		
		if bar != nil {
			line := label + ": " + bar.Render()
			lines = append(lines, line)
		}
	}
	
	return strings.Join(lines, "\n")
}

// AllComplete returns true if all progress bars are complete
func (mpb *MultiProgressBar) AllComplete() bool {
	mpb.mutex.RLock()
	defer mpb.mutex.RUnlock()
	
	for _, bar := range mpb.bars {
		if !bar.IsComplete() {
			return false
		}
	}
	
	return true
}

// BatchProgressTracker tracks progress for batch operations
type BatchProgressTracker struct {
	total     int
	completed int
	failed    int
	errors    []error
	startTime time.Time
	mutex     sync.RWMutex
}

// NewBatchProgressTracker creates a new batch progress tracker
func NewBatchProgressTracker(total int) *BatchProgressTracker {
	return &BatchProgressTracker{
		total:     total,
		completed: 0,
		failed:    0,
		errors:    make([]error, 0),
		startTime: time.Now(),
	}
}

// RecordSuccess records a successful operation
func (bpt *BatchProgressTracker) RecordSuccess() {
	bpt.mutex.Lock()
	defer bpt.mutex.Unlock()
	bpt.completed++
}

// RecordFailure records a failed operation with optional error
func (bpt *BatchProgressTracker) RecordFailure(err error) {
	bpt.mutex.Lock()
	defer bpt.mutex.Unlock()
	bpt.failed++
	if err != nil {
		bpt.errors = append(bpt.errors, err)
	}
}

// GetStats returns current batch operation statistics
func (bpt *BatchProgressTracker) GetStats() BatchStats {
	bpt.mutex.RLock()
	defer bpt.mutex.RUnlock()
	
	return BatchStats{
		Total:     bpt.total,
		Completed: bpt.completed,
		Failed:    bpt.failed,
		Remaining: bpt.total - bpt.completed - bpt.failed,
		Errors:    append([]error{}, bpt.errors...), // Copy slice
		Duration:  time.Since(bpt.startTime),
	}
}

// BatchStats represents statistics for a batch operation
type BatchStats struct {
	Total     int
	Completed int
	Failed    int
	Remaining int
	Errors    []error
	Duration  time.Duration
}

// GetSuccessRate returns the success rate as a percentage
func (bs *BatchStats) GetSuccessRate() float64 {
	processed := bs.Completed + bs.Failed
	if processed == 0 {
		return 0
	}
	return float64(bs.Completed) / float64(processed) * 100
}

// IsComplete returns true if all operations are finished
func (bs *BatchStats) IsComplete() bool {
	return bs.Remaining == 0
}

// RenderSummary returns a human-readable summary of the batch stats
func (bs *BatchStats) RenderSummary() string {
	var parts []string
	
	parts = append(parts, fmt.Sprintf("Total: %d", bs.Total))
	parts = append(parts, fmt.Sprintf("Completed: %d", bs.Completed))
	
	if bs.Failed > 0 {
		parts = append(parts, fmt.Sprintf("Failed: %d", bs.Failed))
	}
	
	if bs.Remaining > 0 {
		parts = append(parts, fmt.Sprintf("Remaining: %d", bs.Remaining))
	}
	
	if bs.Completed+bs.Failed > 0 {
		parts = append(parts, fmt.Sprintf("Success Rate: %.1f%%", bs.GetSuccessRate()))
	}
	
	if bs.Duration > 0 {
		parts = append(parts, fmt.Sprintf("Duration: %s", bs.Duration.Truncate(time.Second)))
	}
	
	return strings.Join(parts, " | ")
}

// SteppedProgress provides progress tracking for multi-step operations
type SteppedProgress struct {
	steps       []string
	currentStep int
	stepPrefix  string
	mutex       sync.RWMutex
}

// NewSteppedProgress creates a new stepped progress tracker
func NewSteppedProgress(steps []string) *SteppedProgress {
	return &SteppedProgress{
		steps:       steps,
		currentStep: 0,
		stepPrefix:  "Step",
	}
}

// SetStepPrefix sets the prefix used for step numbering
func (sp *SteppedProgress) SetStepPrefix(prefix string) {
	sp.mutex.Lock()
	defer sp.mutex.Unlock()
	sp.stepPrefix = prefix
}

// NextStep advances to the next step
func (sp *SteppedProgress) NextStep() bool {
	sp.mutex.Lock()
	defer sp.mutex.Unlock()
	
	if sp.currentStep < len(sp.steps)-1 {
		sp.currentStep++
		return true
	}
	return false
}

// GetCurrentStep returns the current step information
func (sp *SteppedProgress) GetCurrentStep() (int, string, bool) {
	sp.mutex.RLock()
	defer sp.mutex.RUnlock()
	
	if sp.currentStep >= len(sp.steps) {
		return sp.currentStep, "", false
	}
	
	return sp.currentStep + 1, sp.steps[sp.currentStep], true
}

// GetProgress returns the current progress as a percentage
func (sp *SteppedProgress) GetProgress() float64 {
	sp.mutex.RLock()
	defer sp.mutex.RUnlock()
	
	if len(sp.steps) == 0 {
		return 100
	}
	
	return float64(sp.currentStep) / float64(len(sp.steps)) * 100
}

// Render returns a string representation of the current progress
func (sp *SteppedProgress) Render() string {
	stepNum, stepDesc, valid := sp.GetCurrentStep()
	if !valid {
		return "Completed"
	}
	
	progress := sp.GetProgress()
	return fmt.Sprintf("%s %d/%d (%.1f%%): %s", 
		sp.stepPrefix, stepNum, len(sp.steps), progress, stepDesc)
}

// IsComplete returns true if all steps are complete
func (sp *SteppedProgress) IsComplete() bool {
	sp.mutex.RLock()
	defer sp.mutex.RUnlock()
	
	return sp.currentStep >= len(sp.steps)
}