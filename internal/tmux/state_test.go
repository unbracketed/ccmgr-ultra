package tmux

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadStateNewFile(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "state.json")
	
	state, err := LoadState(stateFile)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if state == nil {
		t.Error("Expected state to be initialized")
	}
	
	if state.FilePath != stateFile {
		t.Errorf("Expected file path %s, got %s", stateFile, state.FilePath)
	}
	
	if state.Sessions == nil {
		t.Error("Expected sessions map to be initialized")
	}
	
	if len(state.Sessions) != 0 {
		t.Errorf("Expected empty sessions map, got %d entries", len(state.Sessions))
	}
	
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		t.Error("Expected state file to be created")
	}
}

func TestLoadStateExistingFile(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "state.json")
	
	testData := `{
		"test-session": {
			"id": "test-session",
			"name": "test-session",
			"project": "test-project",
			"worktree": "main",
			"branch": "feature",
			"directory": "/tmp",
			"created": "2023-01-01T00:00:00Z",
			"last_access": "2023-01-01T01:00:00Z",
			"last_state": 1,
			"environment": {"KEY": "value"},
			"metadata": {"key": "value"}
		}
	}`
	
	err := os.WriteFile(stateFile, []byte(testData), 0644)
	if err != nil {
		t.Fatalf("Failed to write test data: %v", err)
	}
	
	state, err := LoadState(stateFile)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if len(state.Sessions) != 1 {
		t.Errorf("Expected 1 session, got %d", len(state.Sessions))
	}
	
	session := state.Sessions["test-session"]
	if session == nil {
		t.Error("Expected test-session to exist")
	}
	
	if session.ID != "test-session" {
		t.Errorf("Expected ID test-session, got %s", session.ID)
	}
	
	if session.Project != "test-project" {
		t.Errorf("Expected project test-project, got %s", session.Project)
	}
}

func TestLoadStateCorruptedFile(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "state.json")
	
	err := os.WriteFile(stateFile, []byte("invalid json"), 0644)
	if err != nil {
		t.Fatalf("Failed to write test data: %v", err)
	}
	
	state, err := LoadState(stateFile)
	if err != nil {
		t.Errorf("Expected no error for corrupted file, got %v", err)
	}
	
	if len(state.Sessions) != 0 {
		t.Errorf("Expected empty sessions for corrupted file, got %d", len(state.Sessions))
	}
	
	backupFiles, _ := filepath.Glob(stateFile + ".backup.*")
	if len(backupFiles) == 0 {
		t.Error("Expected backup file to be created")
	}
}

func TestSaveState(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "state.json")
	
	state := &SessionState{
		FilePath: stateFile,
		Sessions: make(map[string]*PersistedSession),
	}
	
	session := &PersistedSession{
		ID:          "test-session",
		Name:        "test-session",
		Project:     "test-project",
		Worktree:    "main",
		Branch:      "feature",
		Directory:   "/tmp",
		Created:     time.Now(),
		LastAccess:  time.Now(),
		LastState:   StateIdle,
		Environment: map[string]string{"KEY": "value"},
		Metadata:    map[string]interface{}{"key": "value"},
	}
	
	state.Sessions["test-session"] = session
	
	err := state.SaveState()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		t.Error("Expected state file to exist after save")
	}
	
	data, err := os.ReadFile(stateFile)
	if err != nil {
		t.Errorf("Failed to read saved file: %v", err)
	}
	
	if len(data) == 0 {
		t.Error("Expected non-empty state file")
	}
}

func TestAddSession(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "state.json")
	
	state, _ := LoadState(stateFile)
	
	session := &PersistedSession{
		ID:          "test-session",
		Name:        "test-session",
		Project:     "test-project",
		Worktree:    "main",
		Branch:      "feature",
		Directory:   "/tmp",
		Created:     time.Now(),
		LastAccess:  time.Now(),
		LastState:   StateIdle,
		Environment: nil,
		Metadata:    nil,
	}
	
	err := state.AddSession(session)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if len(state.Sessions) != 1 {
		t.Errorf("Expected 1 session, got %d", len(state.Sessions))
	}
	
	savedSession := state.Sessions["test-session"]
	if savedSession == nil {
		t.Error("Expected session to be saved")
	}
	
	if savedSession.Environment == nil {
		t.Error("Expected Environment to be initialized")
	}
	
	if savedSession.Metadata == nil {
		t.Error("Expected Metadata to be initialized")
	}
}

func TestAddSessionEmptyID(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "state.json")
	
	state, _ := LoadState(stateFile)
	
	session := &PersistedSession{
		ID: "",
	}
	
	err := state.AddSession(session)
	if err == nil {
		t.Error("Expected error for empty session ID")
	}
}

func TestRemoveSession(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "state.json")
	
	state, _ := LoadState(stateFile)
	
	session := &PersistedSession{
		ID:          "test-session",
		Environment: make(map[string]string),
		Metadata:    make(map[string]interface{}),
	}
	
	state.Sessions["test-session"] = session
	
	err := state.RemoveSession("test-session")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if len(state.Sessions) != 0 {
		t.Errorf("Expected 0 sessions, got %d", len(state.Sessions))
	}
}

func TestRemoveSessionNotFound(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "state.json")
	
	state, _ := LoadState(stateFile)
	
	err := state.RemoveSession("non-existent")
	if err == nil {
		t.Error("Expected error for non-existent session")
	}
}

func TestUpdateSession(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "state.json")
	
	state, _ := LoadState(stateFile)
	
	session := &PersistedSession{
		ID:          "test-session",
		LastAccess:  time.Unix(0, 0),
		LastState:   StateUnknown,
		Directory:   "/old",
		Branch:      "old-branch",
		Environment: make(map[string]string),
		Metadata:    make(map[string]interface{}),
	}
	
	state.Sessions["test-session"] = session
	
	newTime := time.Now()
	updates := map[string]interface{}{
		"last_access": newTime,
		"last_state":  StateIdle,
		"directory":   "/new",
		"branch":      "new-branch",
		"custom_key":  "custom_value",
	}
	
	err := state.UpdateSession("test-session", updates)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	updated := state.Sessions["test-session"]
	if !updated.LastAccess.Equal(newTime) {
		t.Errorf("Expected LastAccess to be updated")
	}
	
	if updated.LastState != StateIdle {
		t.Errorf("Expected LastState to be StateIdle, got %v", updated.LastState)
	}
	
	if updated.Directory != "/new" {
		t.Errorf("Expected Directory to be /new, got %s", updated.Directory)
	}
	
	if updated.Branch != "new-branch" {
		t.Errorf("Expected Branch to be new-branch, got %s", updated.Branch)
	}
	
	if updated.Metadata["custom_key"] != "custom_value" {
		t.Errorf("Expected custom metadata to be set")
	}
}

func TestGetSession(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "state.json")
	
	state, _ := LoadState(stateFile)
	
	originalSession := &PersistedSession{
		ID:          "test-session",
		Name:        "test-session",
		Environment: map[string]string{"KEY": "value"},
		Metadata:    map[string]interface{}{"key": "value"},
	}
	
	state.Sessions["test-session"] = originalSession
	
	session, err := state.GetSession("test-session")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if session == nil {
		t.Error("Expected session to be returned")
	}
	
	if session == originalSession {
		t.Error("Expected session copy, not original reference")
	}
	
	if session.ID != originalSession.ID {
		t.Errorf("Expected ID %s, got %s", originalSession.ID, session.ID)
	}
	
	session.Environment["NEW_KEY"] = "new_value"
	if originalSession.Environment["NEW_KEY"] == "new_value" {
		t.Error("Expected environment to be copied, not shared")
	}
}

func TestGetSessionNotFound(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "state.json")
	
	state, _ := LoadState(stateFile)
	
	_, err := state.GetSession("non-existent")
	if err == nil {
		t.Error("Expected error for non-existent session")
	}
}

func TestListSessions(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "state.json")
	
	state, _ := LoadState(stateFile)
	
	session1 := &PersistedSession{
		ID:          "session1",
		Environment: make(map[string]string),
		Metadata:    make(map[string]interface{}),
	}
	
	session2 := &PersistedSession{
		ID:          "session2",
		Environment: make(map[string]string),
		Metadata:    make(map[string]interface{}),
	}
	
	state.Sessions["session1"] = session1
	state.Sessions["session2"] = session2
	
	sessions := state.ListSessions()
	
	if len(sessions) != 2 {
		t.Errorf("Expected 2 sessions, got %d", len(sessions))
	}
	
	foundSession1 := false
	foundSession2 := false
	
	for _, session := range sessions {
		if session.ID == "session1" {
			foundSession1 = true
		}
		if session.ID == "session2" {
			foundSession2 = true
		}
	}
	
	if !foundSession1 || !foundSession2 {
		t.Error("Expected both sessions to be returned")
	}
}

func TestGetSessionCount(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "state.json")
	
	state, _ := LoadState(stateFile)
	
	if state.GetSessionCount() != 0 {
		t.Errorf("Expected 0 sessions, got %d", state.GetSessionCount())
	}
	
	session := &PersistedSession{
		ID:          "test-session",
		Environment: make(map[string]string),
		Metadata:    make(map[string]interface{}),
	}
	
	state.Sessions["test-session"] = session
	
	if state.GetSessionCount() != 1 {
		t.Errorf("Expected 1 session, got %d", state.GetSessionCount())
	}
}

func TestGetSessionsByProject(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "state.json")
	
	state, _ := LoadState(stateFile)
	
	session1 := &PersistedSession{
		ID:          "session1",
		Project:     "project1",
		Environment: make(map[string]string),
		Metadata:    make(map[string]interface{}),
	}
	
	session2 := &PersistedSession{
		ID:          "session2",
		Project:     "project1",
		Environment: make(map[string]string),
		Metadata:    make(map[string]interface{}),
	}
	
	session3 := &PersistedSession{
		ID:          "session3",
		Project:     "project2",
		Environment: make(map[string]string),
		Metadata:    make(map[string]interface{}),
	}
	
	state.Sessions["session1"] = session1
	state.Sessions["session2"] = session2
	state.Sessions["session3"] = session3
	
	project1Sessions := state.GetSessionsByProject("project1")
	if len(project1Sessions) != 2 {
		t.Errorf("Expected 2 sessions for project1, got %d", len(project1Sessions))
	}
	
	project2Sessions := state.GetSessionsByProject("project2")
	if len(project2Sessions) != 1 {
		t.Errorf("Expected 1 session for project2, got %d", len(project2Sessions))
	}
	
	nonExistentSessions := state.GetSessionsByProject("non-existent")
	if len(nonExistentSessions) != 0 {
		t.Errorf("Expected 0 sessions for non-existent project, got %d", len(nonExistentSessions))
	}
}

func TestGetSessionsByWorktree(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "state.json")
	
	state, _ := LoadState(stateFile)
	
	session1 := &PersistedSession{
		ID:          "session1",
		Worktree:    "main",
		Environment: make(map[string]string),
		Metadata:    make(map[string]interface{}),
	}
	
	session2 := &PersistedSession{
		ID:          "session2",
		Worktree:    "main",
		Environment: make(map[string]string),
		Metadata:    make(map[string]interface{}),
	}
	
	session3 := &PersistedSession{
		ID:          "session3",
		Worktree:    "dev",
		Environment: make(map[string]string),
		Metadata:    make(map[string]interface{}),
	}
	
	state.Sessions["session1"] = session1
	state.Sessions["session2"] = session2
	state.Sessions["session3"] = session3
	
	mainSessions := state.GetSessionsByWorktree("main")
	if len(mainSessions) != 2 {
		t.Errorf("Expected 2 sessions for main worktree, got %d", len(mainSessions))
	}
	
	devSessions := state.GetSessionsByWorktree("dev")
	if len(devSessions) != 1 {
		t.Errorf("Expected 1 session for dev worktree, got %d", len(devSessions))
	}
}