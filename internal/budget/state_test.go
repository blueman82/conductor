package budget

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewStateManager(t *testing.T) {
	sm := NewStateManager("/tmp/test-state")
	if sm == nil {
		t.Fatal("NewStateManager returned nil")
	}
	if sm.stateDir != "/tmp/test-state" {
		t.Errorf("expected stateDir=/tmp/test-state, got %s", sm.stateDir)
	}
}

func TestStateManager_SaveAndLoad(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "conductor-state-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sm := NewStateManager(tempDir)

	// Create test state
	now := time.Now()
	resumeAt := now.Add(2 * time.Hour)
	state := &ExecutionState{
		SessionID:      "test-session-123",
		PlanFile:       "/path/to/plan.yaml",
		CompletedTasks: []string{"task1", "task2", "task3"},
		CurrentWave:    2,
		PausedAt:       now,
		ResumeAt:       resumeAt,
		Status:         StatusPaused,
		RateLimitInfo: &RateLimitInfo{
			DetectedAt:  now,
			ResetAt:     resumeAt,
			WaitSeconds: 7200,
			LimitType:   LimitTypeSession,
			RawMessage:  "rate limit reached",
			Source:      "output",
		},
	}

	// Test Save
	if err := sm.Save(state); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file exists
	expectedPath := filepath.Join(tempDir, "test-session-123.json")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("state file not created at %s", expectedPath)
	}

	// Test Load
	loaded, err := sm.Load("test-session-123")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify loaded data
	if loaded.SessionID != state.SessionID {
		t.Errorf("SessionID mismatch: expected %s, got %s", state.SessionID, loaded.SessionID)
	}
	if loaded.PlanFile != state.PlanFile {
		t.Errorf("PlanFile mismatch: expected %s, got %s", state.PlanFile, loaded.PlanFile)
	}
	if len(loaded.CompletedTasks) != 3 {
		t.Errorf("CompletedTasks length: expected 3, got %d", len(loaded.CompletedTasks))
	}
	if loaded.CurrentWave != 2 {
		t.Errorf("CurrentWave: expected 2, got %d", loaded.CurrentWave)
	}
	if loaded.Status != StatusPaused {
		t.Errorf("Status: expected %s, got %s", StatusPaused, loaded.Status)
	}
	if loaded.RateLimitInfo == nil {
		t.Error("RateLimitInfo is nil")
	} else {
		if loaded.RateLimitInfo.LimitType != LimitTypeSession {
			t.Errorf("LimitType: expected %s, got %s", LimitTypeSession, loaded.RateLimitInfo.LimitType)
		}
		if loaded.RateLimitInfo.WaitSeconds != 7200 {
			t.Errorf("WaitSeconds: expected 7200, got %d", loaded.RateLimitInfo.WaitSeconds)
		}
	}
}

func TestStateManager_LoadNonExistent(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "conductor-state-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sm := NewStateManager(tempDir)

	_, err = sm.Load("nonexistent-session")
	if err == nil {
		t.Error("expected error loading nonexistent session, got nil")
	}
}

func TestStateManager_GetPausedStates(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "conductor-state-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sm := NewStateManager(tempDir)

	// Create multiple states with different resume times
	now := time.Now()
	states := []*ExecutionState{
		{
			SessionID:      "session1",
			PlanFile:       "plan1.yaml",
			CompletedTasks: []string{"task1"},
			CurrentWave:    1,
			PausedAt:       now,
			ResumeAt:       now.Add(1 * time.Hour),
			Status:         StatusPaused,
		},
		{
			SessionID:      "session2",
			PlanFile:       "plan2.yaml",
			CompletedTasks: []string{"task1", "task2"},
			CurrentWave:    2,
			PausedAt:       now,
			ResumeAt:       now.Add(3 * time.Hour),
			Status:         StatusPaused,
		},
		{
			SessionID:      "session3",
			PlanFile:       "plan3.yaml",
			CompletedTasks: []string{},
			CurrentWave:    0,
			PausedAt:       now,
			ResumeAt:       now.Add(30 * time.Minute),
			Status:         StatusPaused,
		},
	}

	// Save all states
	for _, state := range states {
		if err := sm.Save(state); err != nil {
			t.Fatalf("failed to save state %s: %v", state.SessionID, err)
		}
	}

	// Get paused states
	loaded, err := sm.GetPausedStates()
	if err != nil {
		t.Fatalf("GetPausedStates failed: %v", err)
	}

	// Verify count
	if len(loaded) != 3 {
		t.Errorf("expected 3 states, got %d", len(loaded))
	}

	// Verify sorting (soonest resume time first)
	if len(loaded) >= 3 {
		if loaded[0].SessionID != "session3" {
			t.Errorf("first state should be session3 (30min), got %s", loaded[0].SessionID)
		}
		if loaded[1].SessionID != "session1" {
			t.Errorf("second state should be session1 (1h), got %s", loaded[1].SessionID)
		}
		if loaded[2].SessionID != "session2" {
			t.Errorf("third state should be session2 (3h), got %s", loaded[2].SessionID)
		}
	}
}

func TestStateManager_GetPausedStates_EmptyDirectory(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "conductor-state-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sm := NewStateManager(tempDir)

	states, err := sm.GetPausedStates()
	if err != nil {
		t.Fatalf("GetPausedStates failed on empty dir: %v", err)
	}
	if len(states) != 0 {
		t.Errorf("expected 0 states, got %d", len(states))
	}
}

func TestStateManager_GetPausedStates_NonExistentDirectory(t *testing.T) {
	sm := NewStateManager("/nonexistent/path/that/does/not/exist")

	states, err := sm.GetPausedStates()
	if err != nil {
		t.Fatalf("GetPausedStates should handle nonexistent dir gracefully: %v", err)
	}
	if len(states) != 0 {
		t.Errorf("expected 0 states, got %d", len(states))
	}
}

func TestStateManager_GetReadyStates(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "conductor-state-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sm := NewStateManager(tempDir)

	now := time.Now()
	states := []*ExecutionState{
		{
			SessionID:      "ready1",
			PlanFile:       "plan1.yaml",
			CompletedTasks: []string{"task1"},
			CurrentWave:    1,
			PausedAt:       now.Add(-2 * time.Hour),
			ResumeAt:       now.Add(-1 * time.Hour), // In the past (ready)
			Status:         StatusPaused,
		},
		{
			SessionID:      "paused1",
			PlanFile:       "plan2.yaml",
			CompletedTasks: []string{"task1", "task2"},
			CurrentWave:    2,
			PausedAt:       now,
			ResumeAt:       now.Add(2 * time.Hour), // In the future (not ready)
			Status:         StatusPaused,
		},
		{
			SessionID:      "ready2",
			PlanFile:       "plan3.yaml",
			CompletedTasks: []string{},
			CurrentWave:    0,
			PausedAt:       now.Add(-1 * time.Hour),
			ResumeAt:       now.Add(-5 * time.Minute), // In the past (ready)
			Status:         StatusPaused,
		},
	}

	// Save all states
	for _, state := range states {
		if err := sm.Save(state); err != nil {
			t.Fatalf("failed to save state %s: %v", state.SessionID, err)
		}
	}

	// Get ready states
	ready, err := sm.GetReadyStates()
	if err != nil {
		t.Fatalf("GetReadyStates failed: %v", err)
	}

	// Should only return states with resume time in the past
	if len(ready) != 2 {
		t.Errorf("expected 2 ready states, got %d", len(ready))
	}

	// Verify correct states are returned
	readyIDs := make(map[string]bool)
	for _, state := range ready {
		readyIDs[state.SessionID] = true
	}

	if !readyIDs["ready1"] {
		t.Error("ready1 should be in ready states")
	}
	if !readyIDs["ready2"] {
		t.Error("ready2 should be in ready states")
	}
	if readyIDs["paused1"] {
		t.Error("paused1 should not be in ready states")
	}
}

func TestStateManager_Delete(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "conductor-state-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sm := NewStateManager(tempDir)

	// Create and save a state
	state := &ExecutionState{
		SessionID:      "delete-test",
		PlanFile:       "plan.yaml",
		CompletedTasks: []string{"task1"},
		CurrentWave:    1,
		PausedAt:       time.Now(),
		ResumeAt:       time.Now().Add(1 * time.Hour),
		Status:         StatusPaused,
	}

	if err := sm.Save(state); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file exists
	path := filepath.Join(tempDir, "delete-test.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("state file was not created")
	}

	// Delete the state
	if err := sm.Delete("delete-test"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify file is gone
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("state file still exists after delete")
	}

	// Delete again should not error (idempotent)
	if err := sm.Delete("delete-test"); err != nil {
		t.Errorf("second Delete should not error: %v", err)
	}
}

func TestStateManager_calculateStatus(t *testing.T) {
	sm := NewStateManager("/tmp/test")
	now := time.Now()

	tests := []struct {
		name     string
		state    *ExecutionState
		expected ExecutionStatus
	}{
		{
			name: "paused state with future resume time",
			state: &ExecutionState{
				SessionID: "test1",
				PausedAt:  now,
				ResumeAt:  now.Add(2 * time.Hour),
				Status:    StatusPaused,
			},
			expected: StatusPaused,
		},
		{
			name: "ready state with past resume time",
			state: &ExecutionState{
				SessionID: "test2",
				PausedAt:  now.Add(-2 * time.Hour),
				ResumeAt:  now.Add(-1 * time.Hour),
				Status:    StatusPaused,
			},
			expected: StatusReady,
		},
		{
			name: "expired state older than 7 days",
			state: &ExecutionState{
				SessionID: "test3",
				PausedAt:  now.Add(-8 * 24 * time.Hour),
				ResumeAt:  now.Add(-7*24*time.Hour - 1*time.Hour),
				Status:    StatusPaused,
			},
			expected: StatusExpired,
		},
		{
			name: "resumed state stays resumed",
			state: &ExecutionState{
				SessionID: "test4",
				PausedAt:  now.Add(-1 * time.Hour),
				ResumeAt:  now.Add(-30 * time.Minute),
				Status:    StatusResumed,
			},
			expected: StatusResumed,
		},
		{
			name: "ready at exact resume time",
			state: &ExecutionState{
				SessionID: "test5",
				PausedAt:  now.Add(-1 * time.Hour),
				ResumeAt:  now,
				Status:    StatusPaused,
			},
			expected: StatusReady,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := sm.calculateStatus(tt.state)
			if status != tt.expected {
				t.Errorf("expected status %s, got %s", tt.expected, status)
			}
		})
	}
}

func TestGenerateSessionID(t *testing.T) {
	// Generate multiple IDs
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := GenerateSessionID()

		// Check format: exec-{timestamp}-{random}
		if len(id) < 20 {
			t.Errorf("session ID too short: %s", id)
		}

		if id[:5] != "exec-" {
			t.Errorf("session ID should start with 'exec-', got: %s", id)
		}

		// Check uniqueness
		if ids[id] {
			t.Errorf("duplicate session ID generated: %s", id)
		}
		ids[id] = true
	}

	// Verify all IDs are unique
	if len(ids) != 100 {
		t.Errorf("expected 100 unique IDs, got %d", len(ids))
	}
}

func TestStateManager_SaveCreatesDirectory(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "conductor-state-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Use a subdirectory that doesn't exist yet
	stateDir := filepath.Join(tempDir, "nested", "state", "dir")
	sm := NewStateManager(stateDir)

	state := &ExecutionState{
		SessionID:      "test-create-dir",
		PlanFile:       "plan.yaml",
		CompletedTasks: []string{},
		CurrentWave:    0,
		PausedAt:       time.Now(),
		ResumeAt:       time.Now().Add(1 * time.Hour),
		Status:         StatusPaused,
	}

	// Save should create the directory
	if err := sm.Save(state); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(stateDir); os.IsNotExist(err) {
		t.Error("state directory was not created")
	}

	// Verify file exists
	path := filepath.Join(stateDir, "test-create-dir.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("state file was not created")
	}
}

func TestStateManager_GetPausedStates_SkipsCorruptFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "conductor-state-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sm := NewStateManager(tempDir)

	// Create a valid state
	validState := &ExecutionState{
		SessionID:      "valid",
		PlanFile:       "plan.yaml",
		CompletedTasks: []string{"task1"},
		CurrentWave:    1,
		PausedAt:       time.Now(),
		ResumeAt:       time.Now().Add(1 * time.Hour),
		Status:         StatusPaused,
	}
	if err := sm.Save(validState); err != nil {
		t.Fatalf("failed to save valid state: %v", err)
	}

	// Create a corrupt JSON file
	corruptPath := filepath.Join(tempDir, "corrupt.json")
	if err := os.WriteFile(corruptPath, []byte("{invalid json"), 0644); err != nil {
		t.Fatalf("failed to create corrupt file: %v", err)
	}

	// Create a non-JSON file (should be skipped)
	txtPath := filepath.Join(tempDir, "readme.txt")
	if err := os.WriteFile(txtPath, []byte("some text"), 0644); err != nil {
		t.Fatalf("failed to create txt file: %v", err)
	}

	// GetPausedStates should handle corrupt files gracefully
	states, err := sm.GetPausedStates()
	if err != nil {
		t.Fatalf("GetPausedStates should handle corrupt files: %v", err)
	}

	// Should only return the valid state
	if len(states) != 1 {
		t.Errorf("expected 1 valid state, got %d", len(states))
	}
	if len(states) > 0 && states[0].SessionID != "valid" {
		t.Errorf("expected valid state, got %s", states[0].SessionID)
	}
}

func TestExecutionState_WithoutRateLimitInfo(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "conductor-state-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sm := NewStateManager(tempDir)

	// State without RateLimitInfo (nil)
	state := &ExecutionState{
		SessionID:      "no-ratelimit",
		PlanFile:       "plan.yaml",
		CompletedTasks: []string{"task1"},
		CurrentWave:    1,
		PausedAt:       time.Now(),
		ResumeAt:       time.Now().Add(1 * time.Hour),
		Status:         StatusPaused,
		RateLimitInfo:  nil, // Explicitly nil
	}

	if err := sm.Save(state); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := sm.Load("no-ratelimit")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.RateLimitInfo != nil {
		t.Error("RateLimitInfo should be nil")
	}
}
