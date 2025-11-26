package executor

import (
	"context"
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
			name:       "with explicit session dir (ignored)",
			sessionDir: "/tmp/sessions",
			wantErr:    false,
		},
		{
			name:       "with empty session dir",
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
			// sessionDir parameter is kept for backward compatibility but no longer stored
		})
	}
}

func TestBehaviorCollector_LoadSessionMetrics(t *testing.T) {
	t.Run("session found in database", func(t *testing.T) {
		// Create store and collector
		store, err := learning.NewStore(":memory:")
		require.NoError(t, err)
		defer store.Close()

		ctx := context.Background()

		// Insert a session into the database
		sessionID, _, err := store.UpsertClaudeSession(ctx, "test-external-123", "/project/path", "golang-pro")
		require.NoError(t, err)

		// Update session aggregates to have some data
		err = store.UpdateSessionAggregates(ctx, sessionID, 1000, 60)
		require.NoError(t, err)

		collector := NewBehaviorCollector(store, "")

		// Load session metrics by external ID
		metrics, err := collector.LoadSessionMetrics("test-external-123")

		require.NoError(t, err)
		assert.NotNil(t, metrics)
		assert.Equal(t, 1, metrics.TotalSessions)
		assert.Equal(t, time.Duration(60)*time.Second, metrics.AverageDuration)
	})

	t.Run("session with agent tracking", func(t *testing.T) {
		store, err := learning.NewStore(":memory:")
		require.NoError(t, err)
		defer store.Close()

		ctx := context.Background()

		// Insert session with agent type
		_, _, err = store.UpsertClaudeSession(ctx, "test-agent-456", "/project", "rust-engineer")
		require.NoError(t, err)

		collector := NewBehaviorCollector(store, "")
		metrics, err := collector.LoadSessionMetrics("test-agent-456")

		require.NoError(t, err)
		assert.NotNil(t, metrics)
		assert.Contains(t, metrics.AgentPerformance, "rust-engineer")
	})
}

func TestBehaviorCollector_LoadSessionMetrics_NotFound(t *testing.T) {
	store, err := learning.NewStore(":memory:")
	require.NoError(t, err)
	defer store.Close()

	collector := NewBehaviorCollector(store, "")

	// Try to load non-existent session - should return empty metrics, not error
	metrics, err := collector.LoadSessionMetrics("nonexistent-session")
	require.NoError(t, err)
	assert.NotNil(t, metrics)
	assert.Equal(t, 0, metrics.TotalSessions) // Empty metrics indicate not ingested yet
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

	// Create collector (session directory parameter is now ignored)
	collector := NewBehaviorCollector(store, "/nonexistent/sessions")

	// Try to load session - should return empty metrics (not error) since we now use DB
	metrics, err := collector.LoadSessionMetrics("nonexistent-session")
	require.NoError(t, err)
	assert.NotNil(t, metrics)
	assert.Equal(t, 0, metrics.TotalSessions) // Empty metrics indicate not ingested yet
}
