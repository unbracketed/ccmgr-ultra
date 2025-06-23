package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestSessionTableFormatter_EmptyList(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewSessionTableFormatter(&buf)

	data := struct {
		Sessions []interface{} `json:"sessions"`
		Total    int           `json:"total"`
	}{
		Sessions: []interface{}{},
		Total:    0,
	}

	err := formatter.Format(data)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No sessions found") {
		t.Errorf("Expected 'No sessions found', got: %s", output)
	}
}

func TestSessionTableFormatter_SingleSession(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewSessionTableFormatter(&buf)

	session := struct {
		Name       string    `json:"name"`
		Project    string    `json:"project"`
		Branch     string    `json:"branch"`
		Active     bool      `json:"active"`
		Directory  string    `json:"directory"`
		Created    time.Time `json:"created"`
		LastAccess time.Time `json:"last_access"`
	}{
		Name:       "test-session",
		Project:    "myproject",
		Branch:     "main",
		Active:     true,
		Directory:  "/path/to/project",
		Created:    time.Now().Add(-2 * time.Hour),
		LastAccess: time.Now().Add(-5 * time.Minute),
	}

	data := struct {
		Sessions []interface{} `json:"sessions"`
		Total    int           `json:"total"`
	}{
		Sessions: []interface{}{session},
		Total:    1,
	}

	err := formatter.Format(data)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()

	// Check for expected content
	if !strings.Contains(output, "Sessions") {
		t.Errorf("Expected section header 'Sessions', got: %s", output)
	}
	if !strings.Contains(output, "test-session") {
		t.Errorf("Expected session name 'test-session', got: %s", output)
	}
	if !strings.Contains(output, "myproject") {
		t.Errorf("Expected project 'myproject', got: %s", output)
	}
	if !strings.Contains(output, "✓") {
		t.Errorf("Expected active status '✓', got: %s", output)
	}
	if !strings.Contains(output, "Total sessions: 1") {
		t.Errorf("Expected 'Total sessions: 1', got: %s", output)
	}
}

func TestSessionTableFormatter_MultipleSessions(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewSessionTableFormatter(&buf)

	sessions := []interface{}{
		struct {
			Name       string    `json:"name"`
			Project    string    `json:"project"`
			Branch     string    `json:"branch"`
			Active     bool      `json:"active"`
			Directory  string    `json:"directory"`
			Created    time.Time `json:"created"`
			LastAccess time.Time `json:"last_access"`
		}{
			Name:       "session-1",
			Project:    "project1",
			Branch:     "main",
			Active:     true,
			Directory:  "/path/to/project1",
			Created:    time.Now().Add(-1 * time.Hour),
			LastAccess: time.Now().Add(-10 * time.Minute),
		},
		struct {
			Name       string    `json:"name"`
			Project    string    `json:"project"`
			Branch     string    `json:"branch"`
			Active     bool      `json:"active"`
			Directory  string    `json:"directory"`
			Created    time.Time `json:"created"`
			LastAccess time.Time `json:"last_access"`
		}{
			Name:       "session-2",
			Project:    "project2",
			Branch:     "feature",
			Active:     false,
			Directory:  "/path/to/project2",
			Created:    time.Now().Add(-24 * time.Hour),
			LastAccess: time.Now().Add(-3 * time.Hour),
		},
	}

	data := struct {
		Sessions []interface{} `json:"sessions"`
		Total    int           `json:"total"`
	}{
		Sessions: sessions,
		Total:    2,
	}

	err := formatter.Format(data)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()

	// Check for table structure
	if !strings.Contains(output, "┌─ Sessions ─") {
		t.Errorf("Expected table header, got: %s", output)
	}
	if !strings.Contains(output, "│ Name") {
		t.Errorf("Expected column headers, got: %s", output)
	}
	if !strings.Contains(output, "session-1") {
		t.Errorf("Expected first session, got: %s", output)
	}
	if !strings.Contains(output, "session-2") {
		t.Errorf("Expected second session, got: %s", output)
	}
	if !strings.Contains(output, "Total sessions: 2") {
		t.Errorf("Expected 'Total sessions: 2', got: %s", output)
	}

	// Check for status formatting
	activeStatusCount := strings.Count(output, "✓")
	inactiveStatusCount := strings.Count(output, "✗")
	if activeStatusCount < 1 {
		t.Errorf("Expected at least one active status (✓), got: %s", output)
	}
	if inactiveStatusCount < 1 {
		t.Errorf("Expected at least one inactive status (✗), got: %s", output)
	}
}

func TestSessionTableFormatter_NilData(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewSessionTableFormatter(&buf)

	err := formatter.Format(nil)
	if err == nil {
		t.Error("Expected error for nil data, got nil")
	}

	expectedError := "invalid data type for session formatter"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error containing '%s', got: %v", expectedError, err)
	}
}

func TestSessionTableFormatter_InvalidDataType(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewSessionTableFormatter(&buf)

	err := formatter.Format("invalid data")
	if err == nil {
		t.Error("Expected error for invalid data type, got nil")
	}

	expectedError := "invalid data type for session formatter"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error containing '%s', got: %v", expectedError, err)
	}
}

func TestSessionTableFormatter_TimeFormatting(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewSessionTableFormatter(&buf)

	now := time.Now()
	session := struct {
		Name       string    `json:"name"`
		Project    string    `json:"project"`
		Branch     string    `json:"branch"`
		Active     bool      `json:"active"`
		Directory  string    `json:"directory"`
		Created    time.Time `json:"created"`
		LastAccess time.Time `json:"last_access"`
	}{
		Name:       "test-session",
		Project:    "test",
		Branch:     "main",
		Active:     true,
		Directory:  "/test",
		Created:    now.Add(-25 * time.Hour),   // Should show as "1d ago"
		LastAccess: now.Add(-90 * time.Minute), // Should show as "1h ago"
	}

	data := struct {
		Sessions []interface{} `json:"sessions"`
		Total    int           `json:"total"`
	}{
		Sessions: []interface{}{session},
		Total:    1,
	}

	err := formatter.Format(data)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()

	// Check for human-readable time formatting
	if !strings.Contains(output, "d ago") && !strings.Contains(output, "h ago") {
		t.Errorf("Expected human-readable time format, got: %s", output)
	}
}

func TestSessionTableFormatter_LongPathShortening(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewSessionTableFormatter(&buf)

	longPath := "/very/long/path/to/some/deeply/nested/project/directory/that/exceeds/normal/length"
	session := struct {
		Name       string    `json:"name"`
		Project    string    `json:"project"`
		Branch     string    `json:"branch"`
		Active     bool      `json:"active"`
		Directory  string    `json:"directory"`
		Created    time.Time `json:"created"`
		LastAccess time.Time `json:"last_access"`
	}{
		Name:       "test-session",
		Project:    "test",
		Branch:     "main",
		Active:     true,
		Directory:  longPath,
		Created:    time.Now(),
		LastAccess: time.Now(),
	}

	data := struct {
		Sessions []interface{} `json:"sessions"`
		Total    int           `json:"total"`
	}{
		Sessions: []interface{}{session},
		Total:    1,
	}

	err := formatter.Format(data)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()

	// Check that the long path is shortened (should contain "..." or be truncated)
	if strings.Contains(output, longPath) {
		t.Errorf("Expected long path to be shortened, but found full path in output: %s", output)
	}
}
