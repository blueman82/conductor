package budget

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// ExecutionStatus represents the state of a paused execution
type ExecutionStatus string

const (
	StatusPaused   ExecutionStatus = "paused"
	StatusReady    ExecutionStatus = "ready"   // Reset time has passed
	StatusResumed  ExecutionStatus = "resumed" // Already resumed
	StatusExpired  ExecutionStatus = "expired" // Too old to resume
)

// ExecutionState represents a paused execution that can be resumed
type ExecutionState struct {
	SessionID      string          `json:"session_id"`
	PlanFile       string          `json:"plan_file"`
	RateLimitInfo  *RateLimitInfo  `json:"rate_limit_info,omitempty"`
	CompletedTasks []string        `json:"completed_tasks"`
	CurrentWave    int             `json:"current_wave"`
	PausedAt       time.Time       `json:"paused_at"`
	ResumeAt       time.Time       `json:"resume_at"`
	Status         ExecutionStatus `json:"status"`
}

// StateManager handles saving/loading execution state
type StateManager struct {
	stateDir string // .conductor/state/
}

// NewStateManager creates a manager with the given state directory
func NewStateManager(stateDir string) *StateManager {
	return &StateManager{
		stateDir: stateDir,
	}
}

// Save persists an execution state to disk
// Creates state directory if needed
// File: {stateDir}/{sessionID}.json
func (sm *StateManager) Save(state *ExecutionState) error {
	// Create state directory if needed
	if err := os.MkdirAll(sm.stateDir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	// Marshal to JSON with indentation for human readability
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Write to file
	path := filepath.Join(sm.stateDir, state.SessionID+".json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// Load retrieves a specific execution state by session ID
func (sm *StateManager) Load(sessionID string) (*ExecutionState, error) {
	path := filepath.Join(sm.stateDir, sessionID+".json")

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no state found for session %s", sessionID)
		}
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state ExecutionState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	// Update status dynamically
	state.Status = sm.calculateStatus(&state)

	return &state, nil
}

// GetPausedStates returns all paused execution states
// Updates status to "ready" if reset time has passed
func (sm *StateManager) GetPausedStates() ([]*ExecutionState, error) {
	// Check if state directory exists
	if _, err := os.Stat(sm.stateDir); os.IsNotExist(err) {
		return []*ExecutionState{}, nil // No states yet
	}

	entries, err := os.ReadDir(sm.stateDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read state directory: %w", err)
	}

	var states []*ExecutionState

	for _, entry := range entries {
		// Skip non-JSON files
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(sm.stateDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			// Log warning but continue processing other files
			continue
		}

		var state ExecutionState
		if err := json.Unmarshal(data, &state); err != nil {
			// Skip corrupt files
			continue
		}

		// Update status dynamically
		state.Status = sm.calculateStatus(&state)

		states = append(states, &state)
	}

	// Sort by resume time (soonest first)
	sort.Slice(states, func(i, j int) bool {
		return states[i].ResumeAt.Before(states[j].ResumeAt)
	})

	return states, nil
}

// GetReadyStates returns only states ready to resume (reset time passed)
func (sm *StateManager) GetReadyStates() ([]*ExecutionState, error) {
	allStates, err := sm.GetPausedStates()
	if err != nil {
		return nil, err
	}

	var readyStates []*ExecutionState
	for _, state := range allStates {
		if state.Status == StatusReady {
			readyStates = append(readyStates, state)
		}
	}

	return readyStates, nil
}

// Delete removes a state file after successful resume
func (sm *StateManager) Delete(sessionID string) error {
	path := filepath.Join(sm.stateDir, sessionID+".json")

	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return nil // Already deleted
		}
		return fmt.Errorf("failed to delete state file: %w", err)
	}

	return nil
}

// calculateStatus determines the current status of an execution state
func (sm *StateManager) calculateStatus(state *ExecutionState) ExecutionStatus {
	now := time.Now()

	// Check if already resumed
	if state.Status == StatusResumed {
		return StatusResumed
	}

	// Check if expired (older than 7 days)
	const maxAge = 7 * 24 * time.Hour
	if now.Sub(state.PausedAt) > maxAge {
		return StatusExpired
	}

	// Check if ready to resume
	if state.ResumeAt.Before(now) || state.ResumeAt.Equal(now) {
		return StatusReady
	}

	return StatusPaused
}

// GenerateSessionID creates a unique session ID for new executions
// Format: exec-{timestamp}-{random}
// Example: exec-20231223-a1b2c3
func GenerateSessionID() string {
	// Format: exec-YYYYMMDD-HHMMSS-{random}
	timestamp := time.Now().Format("20060102-150405")

	// Generate 6 random hex characters
	randomBytes := make([]byte, 3)
	if _, err := rand.Read(randomBytes); err != nil {
		// Fallback to timestamp-based random if crypto/rand fails
		randomBytes = []byte{
			byte(time.Now().UnixNano() % 256),
			byte((time.Now().UnixNano() >> 8) % 256),
			byte((time.Now().UnixNano() >> 16) % 256),
		}
	}
	randomStr := hex.EncodeToString(randomBytes)

	return fmt.Sprintf("exec-%s-%s", timestamp, randomStr)
}
