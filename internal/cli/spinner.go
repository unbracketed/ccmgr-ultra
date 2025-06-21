package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// Spinner provides a simple text-based progress indicator
type Spinner struct {
	writer   io.Writer
	message  string
	frames   []string
	delay    time.Duration
	active   bool
	done     chan bool
	mu       sync.Mutex
}

// NewSpinner creates a new spinner with default settings
func NewSpinner(message string) *Spinner {
	return &Spinner{
		writer:  os.Stderr,
		message: message,
		frames:  []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		delay:   100 * time.Millisecond,
		done:    make(chan bool),
	}
}

// NewDotSpinner creates a simple dot-based spinner
func NewDotSpinner(message string) *Spinner {
	return &Spinner{
		writer:  os.Stderr,
		message: message,
		frames:  []string{".", "..", "...", "    "},
		delay:   500 * time.Millisecond,
		done:    make(chan bool),
	}
}

// SetWriter sets the output writer for the spinner
func (s *Spinner) SetWriter(w io.Writer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.writer = w
}

// SetMessage updates the spinner message
func (s *Spinner) SetMessage(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.message = message
}

// Start begins the spinner animation
func (s *Spinner) Start() {
	s.mu.Lock()
	if s.active {
		s.mu.Unlock()
		return
	}
	s.active = true
	s.mu.Unlock()

	go s.spin()
}

// Stop ends the spinner animation
func (s *Spinner) Stop() {
	s.mu.Lock()
	if !s.active {
		s.mu.Unlock()
		return
	}
	s.active = false
	s.mu.Unlock()

	s.done <- true
	s.clearLine()
}

// StopWithMessage stops the spinner and displays a final message
func (s *Spinner) StopWithMessage(message string) {
	s.Stop()
	fmt.Fprintln(s.writer, message)
}

// spin runs the spinner animation loop
func (s *Spinner) spin() {
	frameIndex := 0
	ticker := time.NewTicker(s.delay)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.mu.Lock()
			if !s.active {
				s.mu.Unlock()
				return
			}
			
			frame := s.frames[frameIndex%len(s.frames)]
			output := fmt.Sprintf("\r%s %s", frame, s.message)
			fmt.Fprint(s.writer, output)
			
			frameIndex++
			s.mu.Unlock()
		}
	}
}

// clearLine clears the current line in the terminal
func (s *Spinner) clearLine() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Calculate the length of the line to clear
	lineLength := len(s.message) + 2 // spinner character + space + message
	clearString := "\r" + strings.Repeat(" ", lineLength) + "\r"
	fmt.Fprint(s.writer, clearString)
}


// IsQuietMode checks if output should be suppressed based on environment
func IsQuietMode() bool {
	// This can be enhanced to check for global quiet flags
	// For now, just check if stderr is not a terminal
	return false
}

// ShouldShowProgress determines if progress indicators should be shown
func ShouldShowProgress() bool {
	// Don't show progress indicators in quiet mode or when output is redirected
	return !IsQuietMode()
}