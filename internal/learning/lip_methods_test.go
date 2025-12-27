package learning

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLIPRecordEvent(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		event     *LIPEvent
		wantErr   bool
		errContains string
	}{
		{
			name: "records test_pass event successfully",
			event: &LIPEvent{
				TaskExecutionID: 1,
				TaskNumber:      "1",
				EventType:       LIPEventTestPass,
				Details:         "TestFoo passed",
				Confidence:      0.95,
			},
			wantErr: false,
		},
		{
			name: "records test_fail event successfully",
			event: &LIPEvent{
				TaskExecutionID: 1,
				TaskNumber:      "2",
				EventType:       LIPEventTestFail,
				Details:         "TestBar failed: assertion error",
				Confidence:      1.0,
			},
			wantErr: false,
		},
		{
			name: "records build_success event successfully",
			event: &LIPEvent{
				TaskExecutionID: 1,
				TaskNumber:      "3",
				EventType:       LIPEventBuildSuccess,
				Details:         "go build succeeded",
				Confidence:      1.0,
			},
			wantErr: false,
		},
		{
			name: "records build_fail event successfully",
			event: &LIPEvent{
				TaskExecutionID: 1,
				TaskNumber:      "4",
				EventType:       LIPEventBuildFail,
				Details:         "compilation error at line 42",
				Confidence:      0.8,
			},
			wantErr: false,
		},
		{
			name:      "returns error for nil event",
			event:     nil,
			wantErr:   true,
			errContains: "event cannot be nil",
		},
		{
			name: "returns error for invalid event type",
			event: &LIPEvent{
				TaskExecutionID: 1,
				TaskNumber:      "1",
				EventType:       LIPEventType("invalid_type"),
			},
			wantErr:   true,
			errContains: "invalid event type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, cleanup := setupLIPTestStore(t)
			defer cleanup()

			// Create a task execution to reference
			if tt.event != nil && tt.event.TaskExecutionID > 0 {
				exec := &TaskExecution{
					TaskNumber: tt.event.TaskNumber,
					TaskName:   "Test Task",
					Prompt:     "test prompt",
					Success:    true,
				}
				err := store.RecordExecution(ctx, exec)
				require.NoError(t, err)
				tt.event.TaskExecutionID = exec.ID
			}

			err := store.RecordEvent(ctx, tt.event)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}
			require.NoError(t, err)

			// Verify event was stored with ID assigned
			assert.Greater(t, tt.event.ID, int64(0))
			assert.False(t, tt.event.Timestamp.IsZero())
		})
	}
}

func TestLIPGetEvents(t *testing.T) {
	ctx := context.Background()
	store, cleanup := setupLIPTestStore(t)
	defer cleanup()

	// Create task execution
	exec := &TaskExecution{
		TaskNumber: "1",
		TaskName:   "Test Task",
		Prompt:     "test prompt",
		Success:    true,
	}
	err := store.RecordExecution(ctx, exec)
	require.NoError(t, err)

	// Record multiple events
	events := []*LIPEvent{
		{
			TaskExecutionID: exec.ID,
			TaskNumber:      "1",
			EventType:       LIPEventTestPass,
			Details:         "Test 1 passed",
			Confidence:      1.0,
		},
		{
			TaskExecutionID: exec.ID,
			TaskNumber:      "1",
			EventType:       LIPEventTestFail,
			Details:         "Test 2 failed",
			Confidence:      0.9,
		},
		{
			TaskExecutionID: exec.ID,
			TaskNumber:      "1",
			EventType:       LIPEventBuildSuccess,
			Details:         "Build succeeded",
			Confidence:      1.0,
		},
	}

	for _, event := range events {
		err := store.RecordEvent(ctx, event)
		require.NoError(t, err)
	}

	t.Run("retrieves all events for task execution", func(t *testing.T) {
		retrieved, err := store.GetEvents(ctx, &LIPFilter{
			TaskExecutionID: exec.ID,
		})
		require.NoError(t, err)
		assert.Len(t, retrieved, 3)
	})

	t.Run("filters by task number", func(t *testing.T) {
		retrieved, err := store.GetEvents(ctx, &LIPFilter{
			TaskNumber: "1",
		})
		require.NoError(t, err)
		assert.Len(t, retrieved, 3)
	})

	t.Run("filters by event type", func(t *testing.T) {
		retrieved, err := store.GetEvents(ctx, &LIPFilter{
			TaskExecutionID: exec.ID,
			EventTypes:      []LIPEventType{LIPEventTestPass},
		})
		require.NoError(t, err)
		assert.Len(t, retrieved, 1)
		assert.Equal(t, LIPEventTestPass, retrieved[0].EventType)
	})

	t.Run("filters by minimum confidence", func(t *testing.T) {
		retrieved, err := store.GetEvents(ctx, &LIPFilter{
			TaskExecutionID: exec.ID,
			MinConfidence:   0.95,
		})
		require.NoError(t, err)
		assert.Len(t, retrieved, 2) // Only events with confidence >= 0.95
	})

	t.Run("applies limit", func(t *testing.T) {
		retrieved, err := store.GetEvents(ctx, &LIPFilter{
			TaskExecutionID: exec.ID,
			Limit:           2,
		})
		require.NoError(t, err)
		assert.Len(t, retrieved, 2)
	})

	t.Run("handles empty filter", func(t *testing.T) {
		retrieved, err := store.GetEvents(ctx, nil)
		require.NoError(t, err)
		assert.Len(t, retrieved, 3)
	})
}

func TestLIPCalculateProgress(t *testing.T) {
	ctx := context.Background()

	t.Run("returns error for invalid task execution ID", func(t *testing.T) {
		store, cleanup := setupLIPTestStore(t)
		defer cleanup()

		score, err := store.CalculateProgress(ctx, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid task execution ID")
		assert.Equal(t, ProgressNone, score)

		score, err = store.CalculateProgress(ctx, -1)
		require.Error(t, err)
		assert.Equal(t, ProgressNone, score)
	})

	t.Run("returns zero for task with no events", func(t *testing.T) {
		store, cleanup := setupLIPTestStore(t)
		defer cleanup()

		// Create task execution with no events
		exec := &TaskExecution{
			TaskNumber: "1",
			TaskName:   "Empty Task",
			Prompt:     "test",
			Success:    false,
		}
		err := store.RecordExecution(ctx, exec)
		require.NoError(t, err)

		score, err := store.CalculateProgress(ctx, exec.ID)
		require.NoError(t, err)
		assert.Equal(t, ProgressNone, score)
	})

	t.Run("calculates score from LIP events only", func(t *testing.T) {
		store, cleanup := setupLIPTestStore(t)
		defer cleanup()

		exec := &TaskExecution{
			TaskNumber: "1",
			TaskName:   "LIP Task",
			Prompt:     "test",
			Success:    true,
		}
		err := store.RecordExecution(ctx, exec)
		require.NoError(t, err)

		// Record a test pass event (weight: 0.3)
		err = store.RecordEvent(ctx, &LIPEvent{
			TaskExecutionID: exec.ID,
			TaskNumber:      "1",
			EventType:       LIPEventTestPass,
			Confidence:      1.0,
		})
		require.NoError(t, err)

		score, err := store.CalculateProgress(ctx, exec.ID)
		require.NoError(t, err)
		assert.InDelta(t, 0.3, float64(score), 0.01) // test_pass weight
	})

	t.Run("calculates combined score from LIP and behavioral data", func(t *testing.T) {
		store, cleanup := setupLIPTestStore(t)
		defer cleanup()

		exec := &TaskExecution{
			TaskNumber: "1",
			TaskName:   "Combined Task",
			Prompt:     "test",
			Success:    true,
		}
		err := store.RecordExecution(ctx, exec)
		require.NoError(t, err)

		// Create behavioral session
		sessionData := &BehavioralSessionData{
			TaskExecutionID:   exec.ID,
			SessionStart:      time.Now(),
			TotalToolCalls:    2,
			TotalFileOperations: 1,
		}
		sessionID, err := store.RecordSessionMetrics(ctx, sessionData,
			[]ToolExecutionData{
				{ToolName: "Read", Success: true},
				{ToolName: "Write", Success: true},
			},
			nil,
			[]FileOperationData{
				{OperationType: "write", FilePath: "/test.go", Success: true},
			},
			nil,
		)
		require.NoError(t, err)
		require.Greater(t, sessionID, int64(0))

		// Record LIP event
		err = store.RecordEvent(ctx, &LIPEvent{
			TaskExecutionID: exec.ID,
			TaskNumber:      "1",
			EventType:       LIPEventBuildSuccess,
			Confidence:      1.0,
		})
		require.NoError(t, err)

		score, err := store.CalculateProgress(ctx, exec.ID)
		require.NoError(t, err)

		// Expected: build_success (0.3) + tool (0.1 * 1.0) + file (0.2 * 1.0) = 0.6
		assert.InDelta(t, 0.6, float64(score), 0.01)
	})

	t.Run("reaches maximum achievable score", func(t *testing.T) {
		store, cleanup := setupLIPTestStore(t)
		defer cleanup()

		exec := &TaskExecution{
			TaskNumber: "1",
			TaskName:   "Full Score Task",
			Prompt:     "test",
			Success:    true,
		}
		err := store.RecordExecution(ctx, exec)
		require.NoError(t, err)

		// Record all positive LIP event types (score uses AVG per type, so multiple of same type averages)
		for _, et := range []LIPEventType{
			LIPEventTestPass,   // 0.3
			LIPEventBuildSuccess, // 0.3
		} {
			err = store.RecordEvent(ctx, &LIPEvent{
				TaskExecutionID: exec.ID,
				TaskNumber:      "1",
				EventType:       et,
				Confidence:      1.0,
			})
			require.NoError(t, err)
		}

		// Create behavioral session with all successful operations
		sessionData := &BehavioralSessionData{
			TaskExecutionID:     exec.ID,
			SessionStart:        time.Now(),
			TotalToolCalls:      5,
			TotalFileOperations: 3,
		}
		_, err = store.RecordSessionMetrics(ctx, sessionData,
			[]ToolExecutionData{
				{ToolName: "Read", Success: true},
				{ToolName: "Write", Success: true},
				{ToolName: "Bash", Success: true},
				{ToolName: "Grep", Success: true},
				{ToolName: "Glob", Success: true},
			},
			nil,
			[]FileOperationData{
				{OperationType: "write", FilePath: "/a.go", Success: true},
				{OperationType: "write", FilePath: "/b.go", Success: true},
				{OperationType: "read", FilePath: "/c.go", Success: true},
			},
			nil,
		)
		require.NoError(t, err)

		score, err := store.CalculateProgress(ctx, exec.ID)
		require.NoError(t, err)
		// Max achievable: test_pass(0.3) + build_success(0.3) + tool(0.1) + file(0.2) = 0.9
		// The algorithm caps at 1.0 but max weights sum to 0.9
		assert.InDelta(t, 0.9, float64(score), 0.01)
	})
}

func TestLIPConvenienceMethods(t *testing.T) {
	ctx := context.Background()
	store, cleanup := setupLIPTestStore(t)
	defer cleanup()

	exec := &TaskExecution{
		TaskNumber: "1",
		TaskName:   "Convenience Test",
		Prompt:     "test",
		Success:    true,
	}
	err := store.RecordExecution(ctx, exec)
	require.NoError(t, err)

	t.Run("RecordTestResult - pass", func(t *testing.T) {
		err := store.RecordTestResult(ctx, exec.ID, "1", true, "TestFoo passed")
		require.NoError(t, err)

		events, err := store.GetLIPEventsByExecution(ctx, exec.ID)
		require.NoError(t, err)
		require.Len(t, events, 1)
		assert.Equal(t, LIPEventTestPass, events[0].EventType)
	})

	t.Run("RecordTestResult - fail", func(t *testing.T) {
		err := store.RecordTestResult(ctx, exec.ID, "1", false, "TestBar failed")
		require.NoError(t, err)

		events, err := store.GetEvents(ctx, &LIPFilter{
			TaskExecutionID: exec.ID,
			EventTypes:      []LIPEventType{LIPEventTestFail},
		})
		require.NoError(t, err)
		require.Len(t, events, 1)
		assert.Equal(t, LIPEventTestFail, events[0].EventType)
	})

	t.Run("RecordBuildResult - success", func(t *testing.T) {
		err := store.RecordBuildResult(ctx, exec.ID, "1", true, "Build OK")
		require.NoError(t, err)

		events, err := store.GetEvents(ctx, &LIPFilter{
			TaskExecutionID: exec.ID,
			EventTypes:      []LIPEventType{LIPEventBuildSuccess},
		})
		require.NoError(t, err)
		require.Len(t, events, 1)
		assert.Equal(t, LIPEventBuildSuccess, events[0].EventType)
	})

	t.Run("RecordBuildResult - fail", func(t *testing.T) {
		err := store.RecordBuildResult(ctx, exec.ID, "1", false, "Compilation error")
		require.NoError(t, err)

		events, err := store.GetEvents(ctx, &LIPFilter{
			TaskExecutionID: exec.ID,
			EventTypes:      []LIPEventType{LIPEventBuildFail},
		})
		require.NoError(t, err)
		require.Len(t, events, 1)
		assert.Equal(t, LIPEventBuildFail, events[0].EventType)
	})

	t.Run("GetLIPEventsByTask", func(t *testing.T) {
		events, err := store.GetLIPEventsByTask(ctx, "1")
		require.NoError(t, err)
		assert.Len(t, events, 4) // All 4 events from above
	})
}

func TestLIPMigration(t *testing.T) {
	t.Run("lip_events table is created after migration", func(t *testing.T) {
		store, cleanup := setupLIPTestStore(t)
		defer cleanup()

		exists, err := store.tableExists("lip_events")
		require.NoError(t, err)
		assert.True(t, exists, "lip_events table should exist after migration")
	})

	t.Run("lip_events indexes are created", func(t *testing.T) {
		store, cleanup := setupLIPTestStore(t)
		defer cleanup()

		indexes := []string{
			"idx_lip_events_task_execution",
			"idx_lip_events_task_number",
			"idx_lip_events_event_type",
			"idx_lip_events_timestamp",
		}

		for _, idx := range indexes {
			exists, err := store.indexExists(idx)
			require.NoError(t, err)
			assert.True(t, exists, "index %s should exist", idx)
		}
	})
}

func TestLIPIntegrationWithExistingBehavioralData(t *testing.T) {
	ctx := context.Background()
	store, cleanup := setupLIPTestStore(t)
	defer cleanup()

	// Create a task execution with behavioral data but no LIP events
	exec := &TaskExecution{
		TaskNumber: "1",
		TaskName:   "Behavioral Only Task",
		Prompt:     "test",
		Success:    false, // Failed task with partial progress
	}
	err := store.RecordExecution(ctx, exec)
	require.NoError(t, err)

	// Record behavioral session
	sessionData := &BehavioralSessionData{
		TaskExecutionID: exec.ID,
		SessionStart:    time.Now(),
	}
	_, err = store.RecordSessionMetrics(ctx, sessionData,
		[]ToolExecutionData{
			{ToolName: "Read", Success: true},
			{ToolName: "Write", Success: false}, // One failed
		},
		nil,
		[]FileOperationData{
			{OperationType: "write", FilePath: "/partial.go", Success: true},
		},
		nil,
	)
	require.NoError(t, err)

	t.Run("calculates progress from behavioral data alone", func(t *testing.T) {
		score, err := store.CalculateProgress(ctx, exec.ID)
		require.NoError(t, err)

		// Expected: tool (0.1 * 0.5) + file (0.2 * 1.0) = 0.05 + 0.2 = 0.25
		assert.InDelta(t, 0.25, float64(score), 0.01)
	})

	t.Run("progress updates when LIP events are added", func(t *testing.T) {
		// Add a test pass event
		err := store.RecordEvent(ctx, &LIPEvent{
			TaskExecutionID: exec.ID,
			TaskNumber:      "1",
			EventType:       LIPEventTestPass,
			Confidence:      1.0,
		})
		require.NoError(t, err)

		score, err := store.CalculateProgress(ctx, exec.ID)
		require.NoError(t, err)

		// Expected: test_pass (0.3) + tool (0.05) + file (0.2) = 0.55
		assert.InDelta(t, 0.55, float64(score), 0.01)
	})
}

func TestStoreImplementsLIPCollector(t *testing.T) {
	// Compile-time check that Store implements LIPCollector
	var _ LIPCollector = (*Store)(nil)

	store, cleanup := setupLIPTestStore(t)
	defer cleanup()

	// Verify the interface methods are callable
	ctx := context.Background()

	// Create test execution
	exec := &TaskExecution{
		TaskNumber: "1",
		TaskName:   "Interface Test",
		Prompt:     "test",
		Success:    true,
	}
	err := store.RecordExecution(ctx, exec)
	require.NoError(t, err)

	// Test RecordEvent (LIPCollector interface method)
	event := &LIPEvent{
		TaskExecutionID: exec.ID,
		TaskNumber:      "1",
		EventType:       LIPEventTestPass,
	}
	err = store.RecordEvent(ctx, event)
	require.NoError(t, err)

	// Test GetEvents (LIPCollector interface method)
	events, err := store.GetEvents(ctx, &LIPFilter{TaskExecutionID: exec.ID})
	require.NoError(t, err)
	assert.Len(t, events, 1)

	// Test CalculateProgress (LIPCollector interface method)
	score, err := store.CalculateProgress(ctx, exec.ID)
	require.NoError(t, err)
	assert.Greater(t, float64(score), 0.0)
}

// setupLIPTestStore creates a test store with a temporary database for LIP tests
func setupLIPTestStore(t *testing.T) (*Store, func()) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test_lip.db")
	store, err := NewStore(dbPath)
	require.NoError(t, err)

	cleanup := func() {
		store.Close()
	}

	return store, cleanup
}
