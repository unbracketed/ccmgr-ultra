package claude

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// LogEntry represents a parsed log entry
type LogEntry struct {
	Timestamp time.Time   `json:"timestamp"`
	Level     LogLevel    `json:"level"`
	Message   string      `json:"message"`
	Source    string      `json:"source"`
	ProcessID string      `json:"process_id,omitempty"`
	Raw       string      `json:"raw"`
}

// LogLevel represents the severity level of a log entry
type LogLevel int

const (
	LogLevelUnknown LogLevel = iota
	LogLevelTrace
	LogLevelDebug
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelFatal
)

// String returns the string representation of LogLevel
func (l LogLevel) String() string {
	switch l {
	case LogLevelTrace:
		return "TRACE"
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	case LogLevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// LogParser provides utilities for parsing Claude Code log files
type LogParser struct {
	timestampFormats []string
	levelPatterns    map[LogLevel]*regexp.Regexp
	statePatterns    map[ProcessState]*regexp.Regexp
	config           *ProcessConfig
}

// NewLogParser creates a new log parser
func NewLogParser(config *ProcessConfig) (*LogParser, error) {
	parser := &LogParser{
		config: config,
		timestampFormats: []string{
			"2006-01-02T15:04:05.000Z",
			"2006-01-02T15:04:05Z",
			"2006-01-02 15:04:05.000",
			"2006-01-02 15:04:05",
			"Jan 02 15:04:05",
			"15:04:05.000",
			"15:04:05",
		},
		levelPatterns: make(map[LogLevel]*regexp.Regexp),
		statePatterns: make(map[ProcessState]*regexp.Regexp),
	}

	// Compile level patterns
	levelRegexes := map[LogLevel]string{
		LogLevelTrace: `(?i)\b(trace|trc)\b`,
		LogLevelDebug: `(?i)\b(debug|dbg)\b`,
		LogLevelInfo:  `(?i)\b(info|inf)\b`,
		LogLevelWarn:  `(?i)\b(warn|warning|wrn)\b`,
		LogLevelError: `(?i)\b(error|err|exception)\b`,
		LogLevelFatal: `(?i)\b(fatal|panic|critical)\b`,
	}

	for level, pattern := range levelRegexes {
		compiled, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("failed to compile level pattern for %s: %w", level.String(), err)
		}
		parser.levelPatterns[level] = compiled
	}

	// Copy state patterns from config
	for state := range config.StatePatterns {
		stateEnum := stringToProcessState(state)
		if stateEnum != StateUnknown {
			pattern := config.GetCompiledPattern(stateEnum)
			if pattern != nil {
				parser.statePatterns[stateEnum] = pattern
			}
		}
	}

	return parser, nil
}

// ParseFile parses an entire log file
func (p *LogParser) ParseFile(filePath string) ([]*LogEntry, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	return p.ParseReader(file, filepath.Base(filePath))
}

// ParseReader parses log entries from a reader
func (p *LogParser) ParseReader(reader io.Reader, source string) ([]*LogEntry, error) {
	var entries []*LogEntry
	scanner := bufio.NewScanner(reader)

	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		
		if entry := p.ParseLine(line, source); entry != nil {
			entries = append(entries, entry)
		}
	}

	if err := scanner.Err(); err != nil {
		return entries, fmt.Errorf("error reading from source %s: %w", source, err)
	}

	return entries, nil
}

// ParseLine parses a single log line
func (p *LogParser) ParseLine(line, source string) *LogEntry {
	if strings.TrimSpace(line) == "" {
		return nil
	}

	entry := &LogEntry{
		Raw:    line,
		Source: source,
		Level:  LogLevelUnknown,
	}

	// Parse timestamp
	entry.Timestamp = p.extractTimestamp(line)

	// Parse level
	entry.Level = p.extractLevel(line)

	// Extract message (remove timestamp and level indicators)
	entry.Message = p.extractMessage(line)

	// Try to extract process ID if present
	entry.ProcessID = p.extractProcessID(line)

	return entry
}

// extractTimestamp attempts to extract a timestamp from the log line
func (p *LogParser) extractTimestamp(line string) time.Time {
	// Try each timestamp format
	for _, format := range p.timestampFormats {
		// Look for timestamp at the beginning of the line
		if len(line) >= len(format) {
			if ts, err := time.Parse(format, line[:len(format)]); err == nil {
				return ts
			}
		}

		// Look for timestamp anywhere in the line
		timeRegex := regexp.MustCompile(`\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}`)
		matches := timeRegex.FindStringSubmatch(line)
		if len(matches) > 0 {
			if ts, err := time.Parse(format, matches[0]); err == nil {
				return ts
			}
		}
	}

	// If no timestamp found, use current time
	return time.Now()
}

// extractLevel attempts to extract the log level from the line
func (p *LogParser) extractLevel(line string) LogLevel {
	for level, pattern := range p.levelPatterns {
		if pattern.MatchString(line) {
			return level
		}
	}
	return LogLevelUnknown
}

// extractMessage extracts the main message content from the line
func (p *LogParser) extractMessage(line string) string {
	// Remove common log prefixes (timestamp, level, etc.)
	message := line

	// Remove timestamp (common patterns)
	timeRegex := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}[.]\d{3}[Z]?\s*`)
	message = timeRegex.ReplaceAllString(message, "")

	// Remove level indicators
	levelRegex := regexp.MustCompile(`(?i)\[(TRACE|DEBUG|INFO|WARN|ERROR|FATAL)\]\s*`)
	message = levelRegex.ReplaceAllString(message, "")

	// Remove PID indicators
	pidRegex := regexp.MustCompile(`\[PID:\s*\d+\]\s*`)
	message = pidRegex.ReplaceAllString(message, "")

	return strings.TrimSpace(message)
}

// extractProcessID attempts to extract process ID from the line
func (p *LogParser) extractProcessID(line string) string {
	// Look for PID patterns
	pidRegex := regexp.MustCompile(`\[PID:\s*(\d+)\]`)
	matches := pidRegex.FindStringSubmatch(line)
	if len(matches) > 1 {
		return matches[1]
	}

	// Look for process indicators
	processRegex := regexp.MustCompile(`process[:\s]+(\w+)`)
	matches = processRegex.FindStringSubmatch(line)
	if len(matches) > 1 {
		return matches[1]
	}

	return ""
}

// DetectStateFromEntries analyzes log entries to detect process state
func (p *LogParser) DetectStateFromEntries(entries []*LogEntry) ProcessState {
	if len(entries) == 0 {
		return StateUnknown
	}

	// Analyze recent entries (last 10 or within last minute)
	recentEntries := p.getRecentEntries(entries, 10, time.Minute)
	
	// Check for error patterns first (highest priority)
	for _, entry := range recentEntries {
		if entry.Level >= LogLevelError {
			if p.matchesStatePattern(entry.Message, StateError) {
				return StateError
			}
		}
	}

	// Check for other state patterns
	stateScores := make(map[ProcessState]int)
	
	for _, entry := range recentEntries {
		for state, pattern := range p.statePatterns {
			if pattern.MatchString(entry.Message) {
				stateScores[state]++
			}
		}
	}

	// Return the state with the highest score
	maxScore := 0
	detectedState := StateUnknown
	
	for state, score := range stateScores {
		if score > maxScore {
			maxScore = score
			detectedState = state
		}
	}

	return detectedState
}

// getRecentEntries returns the most recent entries
func (p *LogParser) getRecentEntries(entries []*LogEntry, maxCount int, maxAge time.Duration) []*LogEntry {
	if len(entries) == 0 {
		return nil
	}

	// Start from the end and work backwards
	cutoff := time.Now().Add(-maxAge)
	var recent []*LogEntry
	
	for i := len(entries) - 1; i >= 0 && len(recent) < maxCount; i-- {
		entry := entries[i]
		if entry.Timestamp.After(cutoff) {
			recent = append([]*LogEntry{entry}, recent...) // Prepend to maintain order
		} else {
			break // Entries are assumed to be in chronological order
		}
	}

	return recent
}

// matchesStatePattern checks if a message matches a specific state pattern
func (p *LogParser) matchesStatePattern(message string, state ProcessState) bool {
	pattern := p.statePatterns[state]
	return pattern != nil && pattern.MatchString(message)
}

// FilterByLevel filters log entries by minimum level
func (p *LogParser) FilterByLevel(entries []*LogEntry, minLevel LogLevel) []*LogEntry {
	var filtered []*LogEntry
	for _, entry := range entries {
		if entry.Level >= minLevel {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

// FilterByTimeRange filters log entries by time range
func (p *LogParser) FilterByTimeRange(entries []*LogEntry, start, end time.Time) []*LogEntry {
	var filtered []*LogEntry
	for _, entry := range entries {
		if (entry.Timestamp.Equal(start) || entry.Timestamp.After(start)) &&
		   (entry.Timestamp.Equal(end) || entry.Timestamp.Before(end)) {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

// FilterByPattern filters log entries by regex pattern
func (p *LogParser) FilterByPattern(entries []*LogEntry, pattern *regexp.Regexp) []*LogEntry {
	var filtered []*LogEntry
	for _, entry := range entries {
		if pattern.MatchString(entry.Message) || pattern.MatchString(entry.Raw) {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

// GetStateSummary returns a summary of states detected in the log entries
func (p *LogParser) GetStateSummary(entries []*LogEntry) map[ProcessState]int {
	summary := make(map[ProcessState]int)
	
	for _, entry := range entries {
		for state := range p.statePatterns {
			if p.matchesStatePattern(entry.Message, state) {
				summary[state]++
			}
		}
	}
	
	return summary
}

// TailFile provides functionality to tail a log file for new entries
type LogTailer struct {
	filePath    string
	parser      *LogParser
	lastOffset  int64
	pollInterval time.Duration
}

// NewLogTailer creates a new log tailer
func NewLogTailer(filePath string, parser *LogParser, pollInterval time.Duration) *LogTailer {
	return &LogTailer{
		filePath:     filePath,
		parser:       parser,
		pollInterval: pollInterval,
	}
}

// GetNewEntries returns new log entries since the last check
func (t *LogTailer) GetNewEntries() ([]*LogEntry, error) {
	file, err := os.Open(t.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get current file size
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	currentSize := info.Size()
	if currentSize <= t.lastOffset {
		// No new data
		return nil, nil
	}

	// Seek to last read position
	if _, err := file.Seek(t.lastOffset, 0); err != nil {
		return nil, fmt.Errorf("failed to seek to offset %d: %w", t.lastOffset, err)
	}

	// Read new content
	entries, err := t.parser.ParseReader(file, filepath.Base(t.filePath))
	if err != nil {
		return nil, fmt.Errorf("failed to parse new content: %w", err)
	}

	// Update offset
	t.lastOffset = currentSize

	return entries, nil
}

// stringToProcessState converts a string to ProcessState
func stringToProcessState(s string) ProcessState {
	switch strings.ToLower(s) {
	case "busy":
		return StateBusy
	case "idle":
		return StateIdle
	case "waiting":
		return StateWaiting
	case "error":
		return StateError
	case "starting":
		return StateStarting
	case "stopped":
		return StateStopped
	default:
		return StateUnknown
	}
}