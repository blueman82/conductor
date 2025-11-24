package executor

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/behavioral"
	"github.com/harrison/conductor/internal/learning"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBehaviorCollector(t *testing.T) {
	tests := []struct {
		name       string
		sessionDir string
		wantErr    bool
	}{
		{
			name:       "with explicit session dir",
			sessionDir: "/tmp/sessions",
			wantErr:    false,
		},
		{
			name:       "with empty session dir defaults to ~/.claude/sessions",
			sessionDir: "",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := learning.NewStore(":memory:")
			require.NoError(t, err)
			defer store.Close()

			collector := NewBehaviorCollector(store, tt.sessionDir)
			assert.NotNil(t, collector)
			assert.NotNil(t, collector.store)
			if tt.sessionDir != "" {
				assert.Equal(t, tt.sessionDir, collector.sessionDir)
			}
		})
	}
}

func TestBehaviorCollector_LoadSessionMetrics(t *testing.T) {
	tests := []struct {
		name             string
		sessionContent   string
		expectedError    bool
		expectedSessions int
		expectedTools    int
	}{
		{
			name: "valid session with tool calls",
			sessionContent: `{"type":"session_start","session_id":"test-123","project":"conductor","timestamp":"2025-01-24T10:00:00Z","status":"completed","agent_name":"golang-pro","duration":5000,"success":true,"error_count":0}
{"type":"tool_call","timestamp":"2025-01-24T10:00:01Z","tool_name":"Read","parameters":{"file_path":"test.go"},"result":"success","success":true,"duration":100}
{"type":"tool_call","timestamp":"2025-01-24T10:00:02Z","tool_name":"Write","parameters":{"file_path":"test.go"},"result":"success","success":true,"duration":200}
`,
			expectedError:    false,
			expectedSessions: 1,
			expectedTools:    2,
		},
		{
			name: "session with bash commands",
			sessionContent: `{"type":"session_metadata","session_id":"test-456","project":"conductor","timestamp":"2025-01-24T10:00:00Z","status":"completed","agent_name":"golang-pro","duration":3000,"success":true,"error_count":0}
{"type":"bash_command","timestamp":"2025-01-24T10:00:01Z","command":"go test","exit_code":0,"output":"PASS","output_length":100,"duration":1500,"success":true}
`,
			expectedError:    false,
			expectedSessions: 1,
			expectedTools:    0,
		},
		{
			name: "session with file operations",
			sessionContent: `{"type":"session_start","id":"test-789","project":"conductor","timestamp":"2025-01-24T10:00:00Z","status":"completed","agent_name":"golang-pro","duration":2000,"success":true,"error_count":0}
{"type":"file_operation","timestamp":"2025-01-24T10:00:01Z","operation":"write","path":"test.go","success":true,"size_bytes":1024,"duration":50}
`,
			expectedError:    false,
			expectedSessions: 1,
			expectedTools:    0,
		},
		{
			name: "session with token usage",
			sessionContent: `{"type":"session_start","session_id":"test-abc","project":"conductor","timestamp":"2025-01-24T10:00:00Z","status":"completed","agent_name":"golang-pro","duration":4000,"success":true,"error_count":0}
{"type":"token_usage","timestamp":"2025-01-24T10:00:01Z","input_tokens":1000,"output_tokens":500,"cost_usd":0.05,"model_name":"claude-sonnet-4-5"}
`,
			expectedError:    false,
			expectedSessions: 1,
			expectedTools:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary session file
			tmpDir := t.TempDir()
			sessionFile := filepath.Join(tmpDir, "test-123.jsonl")
			err := os.WriteFile(sessionFile, []byte(tt.sessionContent), 0644)
			require.NoError(t, err)

			// Create store and collector
			store, err := learning.NewStore(":memory:")
			require.NoError(t, err)
			defer store.Close()

			collector := NewBehaviorCollector(store, tmpDir)

			// Load session metrics
			metrics, err := collector.LoadSessionMetrics("test-123")

			if tt.expectedError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, metrics)
			assert.Equal(t, tt.expectedSessions, metrics.TotalSessions)
			assert.Equal(t, tt.expectedTools, len(metrics.ToolExecutions))
		})
	}
}

func TestBehaviorCollector_LoadSessionMetrics_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := learning.NewStore(":memory:")
	require.NoError(t, err)
	defer store.Close()

	collector := NewBehaviorCollector(store, tmpDir)

	// Try to load non-existent session
	_, err = collector.LoadSessionMetrics("nonexistent-session")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session file not found")
}

func TestBehaviorCollector_AggregateMetrics(t *testing.T) {
	tests := []struct {
		name        string
		sessionData *behavioral.SessionData
		wantTools   int
		wantBash    int
		wantFiles   int
		wantTokens  int64
	}{
		{
			name: "aggregate tool executions",
			sessionData: &behavioral.SessionData{
				Session: behavioral.Session{
					ID:         "test-1",
					Project:    "conductor",
					Timestamp:  time.Now(),
					Status:     "completed",
					AgentName:  "golang-pro",
					Duration:   5000,
					Success:    true,
					ErrorCount: 0,
				},
				Events: []behavioral.Event{
					&behavioral.ToolCallEvent{
						BaseEvent:  behavioral.BaseEvent{Type: "tool_call", Timestamp: time.Now()},
						ToolName:   "Read",
						Parameters: map[string]interface{}{"file": "test.go"},
						Result:     "success",
						Success:    true,
						Duration:   100,
					},
					&behavioral.ToolCallEvent{
						BaseEvent:  behavioral.BaseEvent{Type: "tool_call", Timestamp: time.Now()},
						ToolName:   "Write",
						Parameters: map[string]interface{}{"file": "test.go"},
						Result:     "success",
						Success:    true,
						Duration:   200,
					},
				},
			},
			wantTools:  2,
			wantBash:   0,
			wantFiles:  0,
			wantTokens: 0,
		},
		{
			name: "aggregate bash commands",
			sessionData: &behavioral.SessionData{
				Session: behavioral.Session{
					ID:         "test-2",
					Project:    "conductor",
					Timestamp:  time.Now(),
					Status:     "completed",
					AgentName:  "golang-pro",
					Duration:   3000,
					Success:    true,
					ErrorCount: 0,
				},
				Events: []behavioral.Event{
					&behavioral.BashCommandEvent{
						BaseEvent:    behavioral.BaseEvent{Type: "bash_command", Timestamp: time.Now()},
						Command:      "go test",
						ExitCode:     0,
						Output:       "PASS",
						OutputLength: 100,
						Duration:     1500,
						Success:      true,
					},
				},
			},
			wantTools:  0,
			wantBash:   1,
			wantFiles:  0,
			wantTokens: 0,
		},
		{
			name: "aggregate file operations",
			sessionData: &behavioral.SessionData{
				Session: behavioral.Session{
					ID:         "test-3",
					Project:    "conductor",
					Timestamp:  time.Now(),
					Status:     "completed",
					AgentName:  "golang-pro",
					Duration:   2000,
					Success:    true,
					ErrorCount: 0,
				},
				Events: []behavioral.Event{
					&behavioral.FileOperationEvent{
						BaseEvent: behavioral.BaseEvent{Type: "file_operation", Timestamp: time.Now()},
						Operation: "write",
						Path:      "test.go",
						Success:   true,
						SizeBytes: 1024,
						Duration:  50,
					},
				},
			},
			wantTools:  0,
			wantBash:   0,
			wantFiles:  1,
			wantTokens: 0,
		},
		{
			name: "aggregate token usage",
			sessionData: &behavioral.SessionData{
				Session: behavioral.Session{
					ID:         "test-4",
					Project:    "conductor",
					Timestamp:  time.Now(),
					Status:     "completed",
					AgentName:  "golang-pro",
					Duration:   4000,
					Success:    true,
					ErrorCount: 0,
				},
				Events: []behavioral.Event{
					&behavioral.TokenUsageEvent{
						BaseEvent:    behavioral.BaseEvent{Type: "token_usage", Timestamp: time.Now()},
						InputTokens:  1000,
						OutputTokens: 500,
						CostUSD:      0.05,
						ModelName:    "claude-sonnet-4-5",
					},
				},
			},
			wantTools:  0,
			wantBash:   0,
			wantFiles:  0,
			wantTokens: 1500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := learning.NewStore(":memory:")
			require.NoError(t, err)
			defer store.Close()

			collector := NewBehaviorCollector(store, "")
			metrics := collector.AggregateMetrics(tt.sessionData)

			assert.NotNil(t, metrics)
			assert.Equal(t, 1, metrics.TotalSessions)
			assert.Equal(t, tt.wantTools, len(metrics.ToolExecutions))
			assert.Equal(t, tt.wantBash, len(metrics.BashCommands))
			assert.Equal(t, tt.wantFiles, len(metrics.FileOperations))
			assert.Equal(t, tt.wantTokens, metrics.TokenUsage.TotalTokens())
		})
	}
}

func TestBehaviorCollector_StoreSessionMetrics(t *testing.T) {
	ctx := context.Background()

	// Create test metrics
	metrics := &behavioral.BehavioralMetrics{
		TotalSessions:   1,
		SuccessRate:     1.0,
		AverageDuration: 5 * time.Second,
		TotalCost:       0.05,
		ToolExecutions: []behavioral.ToolExecution{
			{
				Name:         "Read",
				Count:        2,
				SuccessRate:  1.0,
				AvgDuration:  100 * time.Millisecond,
				TotalSuccess: 2,
				TotalErrors:  0,
			},
		},
		BashCommands: []behavioral.BashCommand{
			{
				Command:      "go test",
				ExitCode:     0,
				OutputLength: 100,
				Duration:     1500 * time.Millisecond,
				Success:      true,
				Timestamp:    time.Now(),
			},
		},
		FileOperations: []behavioral.FileOperation{
			{
				Type:      "write",
				Path:      "test.go",
				SizeBytes: 1024,
				Success:   true,
				Timestamp: time.Now(),
				Duration:  50,
			},
		},
		TokenUsage: behavioral.TokenUsage{
			InputTokens:  1000,
			OutputTokens: 500,
			CostUSD:      0.05,
			ModelName:    "claude-sonnet-4-5",
		},
		AgentPerformance: map[string]int{
			"golang-pro": 1,
		},
	}

	// Create store and collector
	store, err := learning.NewStore(":memory:")
	require.NoError(t, err)
	defer store.Close()

	collector := NewBehaviorCollector(store, "")

	// First record a task execution to get task_execution_id
	taskExec := &learning.TaskExecution{
		PlanFile:     "test-plan.yaml",
		RunNumber:    1,
		TaskNumber:   "1",
		TaskName:     "Test Task",
		Agent:        "golang-pro",
		Prompt:       "Test prompt",
		Success:      true,
		Output:       "Test output",
		DurationSecs: 5,
		QCVerdict:    "GREEN",
	}
	err = store.RecordExecution(ctx, taskExec)
	require.NoError(t, err)

	// Store behavioral metrics
	err = collector.StoreSessionMetrics(ctx, taskExec.ID, metrics)
	require.NoError(t, err)

	// Verify metrics were stored
	sessionMetrics, err := store.GetSessionMetrics(ctx, 1)
	require.NoError(t, err)
	assert.NotNil(t, sessionMetrics)
	assert.Equal(t, taskExec.ID, sessionMetrics.TaskExecutionID)
}

func TestBehaviorCollector_GracefulDegradation(t *testing.T) {
	ctx := context.Background()

	// Create store
	store, err := learning.NewStore(":memory:")
	require.NoError(t, err)
	defer store.Close()

	// Create task execution record
	taskExec := &learning.TaskExecution{
		PlanFile:     "test-plan.yaml",
		RunNumber:    1,
		TaskNumber:   "1",
		TaskName:     "Test Task",
		Agent:        "golang-pro",
		Prompt:       "Test prompt",
		Success:      true,
		DurationSecs: 5,
	}
	err = store.RecordExecution(ctx, taskExec)
	require.NoError(t, err)

	// Create collector with non-existent session directory
	collector := NewBehaviorCollector(store, "/nonexistent/sessions")

	// Try to load session - should return error but not panic
	_, err = collector.LoadSessionMetrics("nonexistent-session")
	assert.Error(t, err)
}
