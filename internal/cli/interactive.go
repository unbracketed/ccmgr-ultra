package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// InteractiveSelector provides utilities for interactive selection using gum
type InteractiveSelector struct {
	theme   Theme
	options SelectorOptions
}

// SelectorOptions configures the behavior of interactive selectors
type SelectorOptions struct {
	Height        int
	EnableFilter  bool
	EnableConfirm bool
	Placeholder   string
	Header        string
}

// Theme defines the visual styling for interactive elements
type Theme struct {
	Primary   string
	Secondary string
	Success   string
	Warning   string
	Error     string
	Info      string
	Muted     string
}

// DefaultTheme returns the default color theme
func DefaultTheme() Theme {
	return Theme{
		Primary:   "#7C3AED", // Purple
		Secondary: "#6B7280", // Gray
		Success:   "#10B981", // Green
		Warning:   "#F59E0B", // Yellow
		Error:     "#EF4444", // Red
		Info:      "#3B82F6", // Blue
		Muted:     "#9CA3AF", // Light gray
	}
}

// NewInteractiveSelector creates a new interactive selector with default options
func NewInteractiveSelector() *InteractiveSelector {
	return &InteractiveSelector{
		theme: DefaultTheme(),
		options: SelectorOptions{
			Height:        10,
			EnableFilter:  true,
			EnableConfirm: true,
		},
	}
}

// SelectWorktree provides interactive selection of a worktree
func (s *InteractiveSelector) SelectWorktree(worktrees []string) (string, error) {
	if len(worktrees) == 0 {
		return "", NewError("no worktrees available for selection")
	}

	if len(worktrees) == 1 {
		return worktrees[0], nil
	}

	return s.selectFromOptions("Select worktree:", worktrees)
}

// SelectSession provides interactive selection of a session
func (s *InteractiveSelector) SelectSession(sessions []string) (string, error) {
	if len(sessions) == 0 {
		return "", NewError("no sessions available for selection")
	}

	if len(sessions) == 1 {
		return sessions[0], nil
	}

	return s.selectFromOptions("Select session:", sessions)
}

// SelectMultiple provides interactive multi-selection
func (s *InteractiveSelector) SelectMultiple(header string, options []string) ([]string, error) {
	if len(options) == 0 {
		return nil, NewError("no options available for selection")
	}

	return s.selectMultipleFromOptions(header, options)
}

// ConfirmOperation asks for user confirmation with impact assessment
func (s *InteractiveSelector) ConfirmOperation(operation string, impact Impact) (bool, error) {
	message := fmt.Sprintf("Confirm %s?", operation)

	if impact.Destructive {
		message = fmt.Sprintf("⚠️  %s (DESTRUCTIVE OPERATION)", message)
	}

	if impact.Description != "" {
		message = fmt.Sprintf("%s\n%s", message, impact.Description)
	}

	return s.confirm(message)
}

// Impact describes the impact of an operation for confirmation prompts
type Impact struct {
	Destructive   bool
	Reversible    bool
	Description   string
	AffectedItems []string
}

// selectFromOptions uses gum to select from a list of options
func (s *InteractiveSelector) selectFromOptions(header string, options []string) (string, error) {
	// Check if gum is available
	if !s.isGumAvailable() {
		return s.fallbackSelect(header, options)
	}

	args := []string{"choose"}

	if header != "" {
		args = append(args, "--header", header)
	}

	if s.options.Height > 0 {
		args = append(args, "--height", fmt.Sprintf("%d", s.options.Height))
	}

	args = append(args, options...)

	cmd := exec.Command("gum", args...)
	cmd.Stderr = os.Stderr

	output, err := cmd.Output()
	if err != nil {
		return s.fallbackSelect(header, options)
	}

	selection := strings.TrimSpace(string(output))
	if selection == "" {
		return "", NewError("no selection made")
	}

	return selection, nil
}

// selectMultipleFromOptions uses gum to select multiple items
func (s *InteractiveSelector) selectMultipleFromOptions(header string, options []string) ([]string, error) {
	// Check if gum is available
	if !s.isGumAvailable() {
		return s.fallbackSelectMultiple(header, options)
	}

	args := []string{"choose", "--no-limit"}

	if header != "" {
		args = append(args, "--header", header)
	}

	if s.options.Height > 0 {
		args = append(args, "--height", fmt.Sprintf("%d", s.options.Height))
	}

	args = append(args, options...)

	cmd := exec.Command("gum", args...)
	cmd.Stderr = os.Stderr

	output, err := cmd.Output()
	if err != nil {
		return s.fallbackSelectMultiple(header, options)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return []string{}, nil
	}

	return lines, nil
}

// confirm uses gum to get user confirmation
func (s *InteractiveSelector) confirm(message string) (bool, error) {
	// Check if gum is available
	if !s.isGumAvailable() {
		return s.fallbackConfirm(message)
	}

	cmd := exec.Command("gum", "confirm", message)
	err := cmd.Run()

	// gum confirm returns exit code 0 for "yes", 1 for "no"
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if exitError.ExitCode() == 1 {
				return false, nil // User said no
			}
		}
		// Other error - fallback to simple confirmation
		return s.fallbackConfirm(message)
	}

	return true, nil
}

// isGumAvailable checks if gum command is available
func (s *InteractiveSelector) isGumAvailable() bool {
	_, err := exec.LookPath("gum")
	return err == nil
}

// fallbackSelect provides simple text-based selection when gum is not available
func (s *InteractiveSelector) fallbackSelect(header string, options []string) (string, error) {
	if header != "" {
		fmt.Println(header)
	}

	fmt.Println("\nOptions:")
	for i, option := range options {
		fmt.Printf("%d) %s\n", i+1, option)
	}

	fmt.Print("\nEnter selection number: ")
	var selection int
	_, err := fmt.Scanln(&selection)
	if err != nil {
		return "", NewErrorWithCause("failed to read selection", err)
	}

	if selection < 1 || selection > len(options) {
		return "", NewError("invalid selection")
	}

	return options[selection-1], nil
}

// fallbackSelectMultiple provides simple text-based multi-selection
func (s *InteractiveSelector) fallbackSelectMultiple(header string, options []string) ([]string, error) {
	if header != "" {
		fmt.Println(header)
	}

	fmt.Println("\nOptions:")
	for i, option := range options {
		fmt.Printf("%d) %s\n", i+1, option)
	}

	fmt.Print("\nEnter selection numbers (comma-separated): ")
	var input string
	_, err := fmt.Scanln(&input)
	if err != nil {
		return nil, NewErrorWithCause("failed to read selections", err)
	}

	if input == "" {
		return []string{}, nil
	}

	parts := strings.Split(input, ",")
	var selected []string

	for _, part := range parts {
		var num int
		_, err := fmt.Sscanf(strings.TrimSpace(part), "%d", &num)
		if err != nil {
			continue
		}

		if num >= 1 && num <= len(options) {
			selected = append(selected, options[num-1])
		}
	}

	return selected, nil
}

// fallbackConfirm provides simple text-based confirmation
func (s *InteractiveSelector) fallbackConfirm(message string) (bool, error) {
	fmt.Printf("%s [y/N]: ", message)
	var response string
	_, err := fmt.Scanln(&response)
	if err != nil {
		return false, NewErrorWithCause("failed to read response", err)
	}

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes", nil
}

// Input provides an interactive input prompt
func (s *InteractiveSelector) Input(prompt string, placeholder string) (string, error) {
	if s.isGumAvailable() {
		args := []string{"input"}

		if prompt != "" {
			args = append(args, "--prompt", prompt)
		}

		if placeholder != "" {
			args = append(args, "--placeholder", placeholder)
		}

		cmd := exec.Command("gum", args...)
		cmd.Stderr = os.Stderr

		output, err := cmd.Output()
		if err != nil {
			return s.fallbackInput(prompt)
		}

		return strings.TrimSpace(string(output)), nil
	}

	return s.fallbackInput(prompt)
}

// fallbackInput provides simple text-based input
func (s *InteractiveSelector) fallbackInput(prompt string) (string, error) {
	if prompt != "" {
		fmt.Print(prompt)
	}

	var input string
	_, err := fmt.Scanln(&input)
	if err != nil {
		return "", NewErrorWithCause("failed to read input", err)
	}

	return strings.TrimSpace(input), nil
}

// Password provides an interactive password prompt
func (s *InteractiveSelector) Password(prompt string) (string, error) {
	if s.isGumAvailable() {
		args := []string{"input", "--password"}

		if prompt != "" {
			args = append(args, "--prompt", prompt)
		}

		cmd := exec.Command("gum", args...)
		cmd.Stderr = os.Stderr

		output, err := cmd.Output()
		if err != nil {
			return "", NewErrorWithCause("failed to read password", err)
		}

		return strings.TrimSpace(string(output)), nil
	}

	// Fallback for password input would need termios or similar
	// For now, just use regular input with a warning
	fmt.Print("Warning: Password will be visible. ")
	return s.fallbackInput(prompt)
}

// ProgressWithSpinner shows a spinner with progress updates
func (s *InteractiveSelector) ProgressWithSpinner(message string) *InteractiveSpinner {
	return &InteractiveSpinner{
		message: message,
		active:  false,
	}
}

// InteractiveSpinner provides a simple spinner for progress indication
type InteractiveSpinner struct {
	message string
	active  bool
}

// Start begins showing the spinner
func (s *InteractiveSpinner) Start() {
	s.active = true
	fmt.Printf("%s ", s.message)
}

// Update changes the spinner message
func (s *InteractiveSpinner) Update(message string) {
	if s.active {
		fmt.Printf("\r%s ", message)
		s.message = message
	}
}

// Stop terminates the spinner
func (s *InteractiveSpinner) Stop() {
	if s.active {
		fmt.Print("\r")
		s.active = false
	}
}

// StopWithMessage terminates the spinner and shows a final message
func (s *InteractiveSpinner) StopWithMessage(message string) {
	if s.active {
		fmt.Printf("\r%s\n", message)
		s.active = false
	}
}
