package cli

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// ConfirmationPrompt provides utilities for user confirmation prompts
type ConfirmationPrompt struct {
	defaultResponse bool
	timeout         time.Duration
	requireExplicit bool
	style           ConfirmationStyle
}

// ConfirmationStyle defines the visual style of confirmation prompts
type ConfirmationStyle struct {
	WarningColor string
	ErrorColor   string
	InfoColor    string
	PromptColor  string
	DefaultColor string
}

// ConfirmationOptions configures confirmation prompt behavior
type ConfirmationOptions struct {
	DefaultResponse bool
	Timeout         time.Duration
	RequireExplicit bool
	ShowImpact      bool
	Style           ConfirmationStyle
}

// DefaultConfirmationStyle returns the default confirmation style
func DefaultConfirmationStyle() ConfirmationStyle {
	return ConfirmationStyle{
		WarningColor: "\033[33m", // Yellow
		ErrorColor:   "\033[31m", // Red
		InfoColor:    "\033[36m", // Cyan
		PromptColor:  "\033[35m", // Magenta
		DefaultColor: "\033[0m",  // Reset
	}
}

// NewConfirmationPrompt creates a new confirmation prompt with options
func NewConfirmationPrompt(opts *ConfirmationOptions) *ConfirmationPrompt {
	if opts == nil {
		opts = &ConfirmationOptions{}
	}

	return &ConfirmationPrompt{
		defaultResponse: opts.DefaultResponse,
		timeout:         opts.Timeout,
		requireExplicit: opts.RequireExplicit,
		style:           opts.Style,
	}
}

// Confirm prompts the user for confirmation with optional impact assessment
func (cp *ConfirmationPrompt) Confirm(message string, impact *Impact) (bool, error) {
	return cp.ConfirmWithContext(ConfirmationContext{
		Message: message,
		Impact:  impact,
	})
}

// ConfirmationContext provides context for confirmation prompts
type ConfirmationContext struct {
	Message         string
	Impact          *Impact
	Details         []string
	AffectedItems   []string
	Recommendations []string
	Severity        Severity
}

// Severity indicates the severity level of an operation
type Severity int

const (
	SeverityInfo Severity = iota
	SeverityWarning
	SeverityError
	SeverityCritical
)

// String returns the string representation of severity
func (s Severity) String() string {
	switch s {
	case SeverityInfo:
		return "INFO"
	case SeverityWarning:
		return "WARNING"
	case SeverityError:
		return "ERROR"
	case SeverityCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// ConfirmWithContext prompts for confirmation with full context
func (cp *ConfirmationPrompt) ConfirmWithContext(ctx ConfirmationContext) (bool, error) {
	// Display the context
	cp.displayContext(ctx)

	// Get the prompt string
	promptStr := cp.buildPromptString(ctx)

	// Handle timeout if specified
	if cp.timeout > 0 {
		return cp.confirmWithTimeout(promptStr, cp.timeout)
	}

	return cp.confirmDirect(promptStr)
}

// displayContext displays the confirmation context to the user
func (cp *ConfirmationPrompt) displayContext(ctx ConfirmationContext) {
	style := cp.style

	// Display severity indicator
	if ctx.Severity >= SeverityWarning {
		color := style.WarningColor
		if ctx.Severity >= SeverityError {
			color = style.ErrorColor
		}
		fmt.Printf("%s[%s]%s ", color, ctx.Severity.String(), style.DefaultColor)
	}

	// Display main message
	fmt.Println(ctx.Message)

	// Display impact information
	if ctx.Impact != nil {
		cp.displayImpact(ctx.Impact)
	}

	// Display details
	if len(ctx.Details) > 0 {
		fmt.Printf("\n%sDetails:%s\n", style.InfoColor, style.DefaultColor)
		for _, detail := range ctx.Details {
			fmt.Printf("  • %s\n", detail)
		}
	}

	// Display affected items
	if len(ctx.AffectedItems) > 0 {
		fmt.Printf("\n%sAffected items:%s\n", style.InfoColor, style.DefaultColor)
		for _, item := range ctx.AffectedItems {
			fmt.Printf("  • %s\n", item)
		}
	}

	// Display recommendations
	if len(ctx.Recommendations) > 0 {
		fmt.Printf("\n%sRecommendations:%s\n", style.InfoColor, style.DefaultColor)
		for _, rec := range ctx.Recommendations {
			fmt.Printf("  • %s\n", rec)
		}
	}

	fmt.Println() // Add blank line before prompt
}

// displayImpact displays impact information with appropriate styling
func (cp *ConfirmationPrompt) displayImpact(impact *Impact) {
	style := cp.style

	if impact.Destructive {
		fmt.Printf("%s⚠️  DESTRUCTIVE OPERATION%s\n", style.ErrorColor, style.DefaultColor)
	}

	if impact.Description != "" {
		fmt.Printf("%sImpact:%s %s\n", style.InfoColor, style.DefaultColor, impact.Description)
	}

	if len(impact.AffectedItems) > 0 {
		fmt.Printf("%sWill affect:%s\n", style.InfoColor, style.DefaultColor)
		for _, item := range impact.AffectedItems {
			fmt.Printf("  • %s\n", item)
		}
	}

	if impact.Reversible {
		fmt.Printf("%s✓ This operation is reversible%s\n", style.InfoColor, style.DefaultColor)
	} else {
		fmt.Printf("%s⚠️  This operation is NOT reversible%s\n", style.WarningColor, style.DefaultColor)
	}
}

// buildPromptString builds the confirmation prompt string
func (cp *ConfirmationPrompt) buildPromptString(ctx ConfirmationContext) string {
	style := cp.style
	prompt := fmt.Sprintf("%sConfirm operation?%s", style.PromptColor, style.DefaultColor)

	if cp.requireExplicit {
		prompt += " (type 'yes' to confirm)"
	} else {
		if cp.defaultResponse {
			prompt += " [Y/n]"
		} else {
			prompt += " [y/N]"
		}
	}

	return prompt + ": "
}

// confirmDirect prompts for direct confirmation without timeout
func (cp *ConfirmationPrompt) confirmDirect(promptStr string) (bool, error) {
	fmt.Print(promptStr)

	var response string
	_, err := fmt.Scanln(&response)
	if err != nil {
		return false, NewErrorWithCause("failed to read response", err)
	}

	return cp.parseResponse(response), nil
}

// confirmWithTimeout prompts for confirmation with a timeout
func (cp *ConfirmationPrompt) confirmWithTimeout(promptStr string, timeout time.Duration) (bool, error) {
	fmt.Printf("%s (timeout: %s): ", promptStr, timeout)

	// Create a channel to receive the response
	responseChan := make(chan string, 1)
	errorChan := make(chan error, 1)

	// Start a goroutine to read input
	go func() {
		var response string
		_, err := fmt.Scanln(&response)
		if err != nil {
			errorChan <- err
			return
		}
		responseChan <- response
	}()

	// Wait for response or timeout
	select {
	case response := <-responseChan:
		return cp.parseResponse(response), nil
	case err := <-errorChan:
		return false, NewErrorWithCause("failed to read response", err)
	case <-time.After(timeout):
		fmt.Printf("\nTimeout reached. Using default response: %t\n", cp.defaultResponse)
		return cp.defaultResponse, nil
	}
}

// parseResponse parses the user's response and returns the confirmation result
func (cp *ConfirmationPrompt) parseResponse(response string) bool {
	response = strings.ToLower(strings.TrimSpace(response))

	if cp.requireExplicit {
		return response == "yes"
	}

	if response == "" {
		return cp.defaultResponse
	}

	switch response {
	case "y", "yes", "true", "1":
		return true
	case "n", "no", "false", "0":
		return false
	default:
		return cp.defaultResponse
	}
}

// ConfirmDestructive provides a specialized confirmation for destructive operations
func (cp *ConfirmationPrompt) ConfirmDestructive(operation string, targets []string) (bool, error) {
	impact := &Impact{
		Destructive:    true,
		Reversible:     false,
		Description:    fmt.Sprintf("This will permanently %s the specified items", operation),
		AffectedItems:  targets,
	}

	ctx := ConfirmationContext{
		Message:  fmt.Sprintf("About to %s %d item(s)", operation, len(targets)),
		Impact:   impact,
		Severity: SeverityError,
		Recommendations: []string{
			"Ensure you have backups if needed",
			"Double-check that you selected the correct items",
			"Consider using --dry-run first to preview changes",
		},
	}

	return cp.ConfirmWithContext(ctx)
}

// ConfirmBatch provides confirmation for batch operations
func (cp *ConfirmationPrompt) ConfirmBatch(operation string, count int, summary string) (bool, error) {
	ctx := ConfirmationContext{
		Message: fmt.Sprintf("About to %s %d items", operation, count),
		Details: []string{summary},
		Severity: SeverityWarning,
		Recommendations: []string{
			"Review the operation summary carefully",
			"Consider processing items in smaller batches",
		},
	}

	return cp.ConfirmWithContext(ctx)
}

// MultiStepConfirmation manages confirmation for multi-step operations
type MultiStepConfirmation struct {
	steps   []ConfirmationStep
	current int
	prompt  *ConfirmationPrompt
}

// ConfirmationStep represents a single step in a multi-step confirmation
type ConfirmationStep struct {
	Name        string
	Description string
	Context     ConfirmationContext
	Required    bool
	Completed   bool
}

// NewMultiStepConfirmation creates a new multi-step confirmation
func NewMultiStepConfirmation(opts *ConfirmationOptions) *MultiStepConfirmation {
	return &MultiStepConfirmation{
		steps:   make([]ConfirmationStep, 0),
		current: 0,
		prompt:  NewConfirmationPrompt(opts),
	}
}

// AddStep adds a confirmation step
func (msc *MultiStepConfirmation) AddStep(step ConfirmationStep) {
	msc.steps = append(msc.steps, step)
}

// ExecuteAll executes all confirmation steps
func (msc *MultiStepConfirmation) ExecuteAll() (bool, error) {
	fmt.Printf("Multi-step confirmation (%d steps)\n", len(msc.steps))
	fmt.Println(strings.Repeat("=", 50))

	for i, step := range msc.steps {
		msc.current = i

		fmt.Printf("\nStep %d/%d: %s\n", i+1, len(msc.steps), step.Name)
		if step.Description != "" {
			fmt.Printf("%s\n", step.Description)
		}

		confirmed, err := msc.prompt.ConfirmWithContext(step.Context)
		if err != nil {
			return false, err
		}

		if !confirmed {
			if step.Required {
				fmt.Printf("Step %d is required. Confirmation cancelled.\n", i+1)
				return false, nil
			} else {
				fmt.Printf("Step %d skipped.\n", i+1)
			}
		} else {
			msc.steps[i].Completed = true
			fmt.Printf("Step %d confirmed.\n", i+1)
		}
	}

	// Check if all required steps were completed
	for i, step := range msc.steps {
		if step.Required && !step.Completed {
			fmt.Printf("Required step %d (%s) was not completed.\n", i+1, step.Name)
			return false, nil
		}
	}

	fmt.Println("\nAll required steps confirmed.")
	return true, nil
}

// GetProgress returns the current progress of the multi-step confirmation
func (msc *MultiStepConfirmation) GetProgress() (int, int) {
	completed := 0
	for _, step := range msc.steps {
		if step.Completed {
			completed++
		}
	}
	return completed, len(msc.steps)
}

// IsInteractiveTerminal checks if the current terminal supports interactive input
func IsInteractiveTerminal() bool {
	// Check if stdout is a terminal
	if fileInfo, err := os.Stdout.Stat(); err == nil {
		return (fileInfo.Mode() & os.ModeCharDevice) != 0
	}
	return false
}

// SafeConfirm provides a safe confirmation that works in both interactive and non-interactive modes
func SafeConfirm(message string, defaultResponse bool) (bool, error) {
	if !IsInteractiveTerminal() {
		// Non-interactive mode - use default response
		fmt.Printf("%s (non-interactive mode, using default: %t)\n", message, defaultResponse)
		return defaultResponse, nil
	}

	prompt := NewConfirmationPrompt(&ConfirmationOptions{
		DefaultResponse: defaultResponse,
	})

	return prompt.Confirm(message, nil)
}