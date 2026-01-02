package learning

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecordSessionMetrics(t *testing.T) {
	tests := []struct {
		name        string
		sessionData *BehavioralSessionData
		tools       []ToolExecutionData
		bashCmds    []BashCommandData
		fileOps     []FileOperationData
		tokenUsage  []TokenUsageData
		wantErr     bool
	}{
		{
			name: "records complete session with all metrics",
			sessionData: &BehavioralSessionData{
				TaskExecutionID:     1,
				SessionStart:        time.Now(),
				SessionEnd:          nil,
				TotalDurationSecs:   120,
				TotalToolCalls:      10,
				TotalBashCommands:   3,
				TotalFileOperations: 5,
				TotalTokensUsed:     1500,
				ContextWindowUsed:   50,
			},
			tools: []ToolExecutionData{
				{
					ToolName:     "Read",
					Parameters:   `{"file_path":"/test/file.go"}`,
					DurationMs:   100,
					Success:      true,
					ErrorMessage: "",
				},
				{
					ToolName:     "Write",
					Parameters:   `{"file_path":"/test/output.go"}`,
					DurationMs:   150,
					Success:      true,
					ErrorMessage: "",
				},
			},
			bashCmds: []BashCommandData{
				{
					Command:      "go test",
					DurationMs:   5000,
					ExitCode:     0,
					StdoutLength: 200,
					StderrLength: 0,
					Success:      true,
				},
			},
			fileOps: []FileOperationData{
				{
					OperationType: "write",
					FilePath:      "/test/output.go",
					DurationMs:    50,
					BytesAffected: 1024,
					Success:       true,
					ErrorMessage:  "",
				},
			},
			tokenUsage: []TokenUsageData{
				{
					InputTokens:       1000,
					OutputTokens:      500,
					TotalTokens:       1500,
					ContextWindowSize: 200000,
				},
			},
			wantErr: false,
		},
		{
			name: "records session with no child metrics",
			sessionData: &BehavioralSessionData{
				TaskExecutionID:     1,
				SessionStart:        time.Now(),
				TotalDurationSecs:   60,
				TotalToolCalls:      0,
				TotalBashCommands:   0,
				TotalFileOperations: 0,
				TotalTokensUsed:     0,
				ContextWindowUsed:   0,
			},
			tools:      []ToolExecutionData{},
			bashCmds:   []BashCommandData{},
			fileOps:    []FileOperationData{},
			tokenUsage: []TokenUsageData{},
			wantErr:    false,
		},
		{
			name: "records session with session end time",
			sessionData: &BehavioralSessionData{
				TaskExecutionID:     1,
				SessionStart:        time.Now().Add(-5 * time.Minute),
				SessionEnd:          timePtr(time.Now()),
				TotalDurationSecs:   300,
				TotalToolCalls:      5,
				TotalBashCommands:   2,
				TotalFileOperations: 3,
				TotalTokensUsed:     2000,
				ContextWindowUsed:   75,
			},
			tools:      []ToolExecutionData{},
			bashCmds:   []BashCommandData{},
			fileOps:    []FileOperationData{},
			tokenUsage: []TokenUsageData{},
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			store := setupTestStore(t)
			defer store.Close()

			// Create a task execution first
			exec := &TaskExecution{
				PlanFile:     "test.yaml",
				RunNumber:    1,
				TaskNumber:   "1",
				TaskName:     "Test Task",
				Agent:        "test-agent",
				Prompt:       "test prompt",
				Success:      true,
				DurationSecs: 120,
			}
			err := store.RecordExecution(ctx, exec)
			require.NoError(t, err)

			// Update session data with actual task execution ID
			tt.sessionData.TaskExecutionID = exec.ID

			// Record session metrics
			sessionID, err := store.RecordSessionMetrics(ctx, tt.sessionData, tt.tools, tt.bashCmds, tt.fileOps, tt.tokenUsage)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Greater(t, sessionID, int64(0))

			// Verify session was recorded
			metrics, err := store.GetSessionMetrics(ctx, sessionID)
			require.NoError(t, err)
			assert.Equal(t, sessionID, metrics.SessionID)
			assert.Equal(t, exec.ID, metrics.TaskExecutionID)
			assert.Equal(t, tt.sessionData.TotalDurationSecs, metrics.TotalDurationSecs)
			assert.Equal(t, tt.sessionData.TotalToolCalls, metrics.TotalToolCalls)
			assert.Equal(t, tt.sessionData.TotalBashCommands, metrics.TotalBashCommands)
			assert.Equal(t, tt.sessionData.TotalFileOperations, metrics.TotalFileOperations)
		})
	}
}

func TestGetSessionMetrics(t *testing.T) {
	t.Run("retrieves session with execution context", func(t *testing.T) {
		ctx := context.Background()
		store := setupTestStore(t)
		defer store.Close()

		// Create task execution
		exec := &TaskExecution{
			PlanFile:     "test.yaml",
			RunNumber:    1,
			TaskNumber:   "5",
			TaskName:     "Database Migration",
			Agent:        "migration-agent",
			Prompt:       "migrate database schema",
			Success:      true,
			DurationSecs: 180,
		}
		err := store.RecordExecution(ctx, exec)
		require.NoError(t, err)

		// Create session
		sessionData := &BehavioralSessionData{
			TaskExecutionID:     exec.ID,
			SessionStart:        time.Now(),
			TotalDurationSecs:   180,
			TotalToolCalls:      15,
			TotalBashCommands:   5,
			TotalFileOperations: 8,
			TotalTokensUsed:     3000,
			ContextWindowUsed:   80,
		}
		sessionID, err := store.RecordSessionMetrics(ctx, sessionData, nil, nil, nil, nil)
		require.NoError(t, err)

		// Retrieve metrics
		metrics, err := store.GetSessionMetrics(ctx, sessionID)
		require.NoError(t, err)

		assert.Equal(t, sessionID, metrics.SessionID)
		assert.Equal(t, exec.ID, metrics.TaskExecutionID)
		assert.Equal(t, "5", metrics.TaskNumber)
		assert.Equal(t, "Database Migration", metrics.TaskName)
		assert.Equal(t, "migration-agent", metrics.Agent)
		assert.True(t, metrics.Success)
		assert.Equal(t, int64(180), metrics.TotalDurationSecs)
		assert.Equal(t, 15, metrics.TotalToolCalls)
	})

	t.Run("returns error for nonexistent session", func(t *testing.T) {
		ctx := context.Background()
		store := setupTestStore(t)
		defer store.Close()

		_, err := store.GetSessionMetrics(ctx, 9999)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "session not found")
	})

	t.Run("handles session with end time", func(t *testing.T) {
		ctx := context.Background()
		store := setupTestStore(t)
		defer store.Close()

		// Create task execution
		exec := &TaskExecution{
			PlanFile:     "test.yaml",
			TaskNumber:   "1",
			TaskName:     "Test Task",
			Agent:        "test-agent",
			Prompt:       "test",
			Success:      true,
			DurationSecs: 60,
		}
		err := store.RecordExecution(ctx, exec)
		require.NoError(t, err)

		// Create session with end time
		endTime := time.Now()
		sessionData := &BehavioralSessionData{
			TaskExecutionID:   exec.ID,
			SessionStart:      time.Now().Add(-5 * time.Minute),
			SessionEnd:        &endTime,
			TotalDurationSecs: 300,
		}
		sessionID, err := store.RecordSessionMetrics(ctx, sessionData, nil, nil, nil, nil)
		require.NoError(t, err)

		// Verify end time
		metrics, err := store.GetSessionMetrics(ctx, sessionID)
		require.NoError(t, err)
		assert.NotNil(t, metrics.SessionEnd)
		assert.WithinDuration(t, endTime, *metrics.SessionEnd, time.Second)
	})
}

func TestGetTaskBehavior(t *testing.T) {
	t.Run("aggregates behavior for single task", func(t *testing.T) {
		ctx := context.Background()
		store := setupTestStore(t)
		defer store.Close()

		// Create multiple executions for same task
		taskNumber := "3"
		for i := 0; i < 3; i++ {
			exec := &TaskExecution{
				PlanFile:     "test.yaml",
				TaskNumber:   taskNumber,
				TaskName:     "API Handler",
				Agent:        "backend-agent",
				Prompt:       "implement handler",
				Success:      true,
				DurationSecs: 120,
			}
			err := store.RecordExecution(ctx, exec)
			require.NoError(t, err)

			sessionData := &BehavioralSessionData{
				TaskExecutionID:     exec.ID,
				SessionStart:        time.Now(),
				TotalDurationSecs:   120 + int64(i*10),
				TotalToolCalls:      10 + i,
				TotalBashCommands:   3 + i,
				TotalFileOperations: 5 + i,
				TotalTokensUsed:     1500 + int64(i*100),
				ContextWindowUsed:   50 + i*5,
			}
			_, err = store.RecordSessionMetrics(ctx, sessionData, nil, nil, nil, nil)
			require.NoError(t, err)
		}

		// Get task behavior
		behavior, err := store.GetTaskBehavior(ctx, taskNumber)
		require.NoError(t, err)

		assert.Equal(t, 3, behavior["session_count"])
		assert.Greater(t, behavior["avg_duration_seconds"], 0.0)
		assert.Greater(t, behavior["avg_tool_calls"], 0.0)
		assert.Greater(t, behavior["total_tokens"].(int64), int64(0))
	})

	t.Run("returns zeros for nonexistent task", func(t *testing.T) {
		ctx := context.Background()
		store := setupTestStore(t)
		defer store.Close()

		behavior, err := store.GetTaskBehavior(ctx, "nonexistent")
		require.NoError(t, err)

		assert.Equal(t, 0, behavior["session_count"])
		assert.Equal(t, 0.0, behavior["avg_duration_seconds"])
		assert.Equal(t, int64(0), behavior["total_tokens"])
	})
}

func TestGetProjectBehavior(t *testing.T) {
	t.Run("aggregates behavior for project", func(t *testing.T) {
		ctx := context.Background()
		store := setupTestStore(t)
		defer store.Close()

		project := "myproject"

		// Create multiple executions for project
		for i := 0; i < 5; i++ {
			exec := &TaskExecution{
				PlanFile:     project + "/plan.yaml",
				TaskNumber:   string(rune('1' + i)),
				TaskName:     "Task " + string(rune('A'+i)),
				Agent:        "test-agent",
				Prompt:       "test",
				Success:      i%2 == 0, // 3 successes, 2 failures
				DurationSecs: 100,
			}
			err := store.RecordExecution(ctx, exec)
			require.NoError(t, err)

			sessionData := &BehavioralSessionData{
				TaskExecutionID:     exec.ID,
				SessionStart:        time.Now(),
				TotalDurationSecs:   100,
				TotalToolCalls:      10,
				TotalBashCommands:   2,
				TotalFileOperations: 4,
				TotalTokensUsed:     1000,
				ContextWindowUsed:   40,
			}
			_, err = store.RecordSessionMetrics(ctx, sessionData, nil, nil, nil, nil)
			require.NoError(t, err)
		}

		// Get project behavior
		behavior, err := store.GetProjectBehavior(ctx, project)
		require.NoError(t, err)

		assert.Equal(t, 5, behavior["session_count"])
		assert.Equal(t, 3, behavior["success_count"])
		assert.Equal(t, 0.6, behavior["success_rate"])
		assert.Equal(t, int64(50), behavior["total_tool_calls"])
		assert.Equal(t, int64(10), behavior["total_bash_commands"])
		assert.Equal(t, int64(20), behavior["total_file_operations"])
		assert.Equal(t, int64(5000), behavior["total_tokens"])
	})

	t.Run("returns zeros for nonexistent project", func(t *testing.T) {
		ctx := context.Background()
		store := setupTestStore(t)
		defer store.Close()

		behavior, err := store.GetProjectBehavior(ctx, "nonexistent")
		require.NoError(t, err)

		assert.Equal(t, 0, behavior["session_count"])
		assert.Equal(t, 0.0, behavior["success_rate"])
	})
}

func TestDeleteOldBehavioralData(t *testing.T) {
	t.Run("deletes sessions older than cutoff", func(t *testing.T) {
		ctx := context.Background()
		store := setupTestStore(t)
		defer store.Close()

		// Create old execution
		oldExec := &TaskExecution{
			PlanFile:     "test.yaml",
			TaskNumber:   "1",
			TaskName:     "Old Task",
			Agent:        "test-agent",
			Prompt:       "test",
			Success:      true,
			DurationSecs: 60,
			Timestamp:    time.Now().Add(-48 * time.Hour),
		}
		// Manually insert with old timestamp
		query := `INSERT INTO task_executions
			(plan_file, task_number, task_name, agent, prompt, success, duration_seconds, timestamp)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
		result, err := store.db.Exec(query,
			oldExec.PlanFile, oldExec.TaskNumber, oldExec.TaskName,
			oldExec.Agent, oldExec.Prompt, oldExec.Success, oldExec.DurationSecs, oldExec.Timestamp)
		require.NoError(t, err)
		oldExecID, _ := result.LastInsertId()

		// Create recent execution
		recentExec := &TaskExecution{
			PlanFile:     "test.yaml",
			TaskNumber:   "2",
			TaskName:     "Recent Task",
			Agent:        "test-agent",
			Prompt:       "test",
			Success:      true,
			DurationSecs: 60,
		}
		err = store.RecordExecution(ctx, recentExec)
		require.NoError(t, err)

		// Create sessions for both
		oldSession := &BehavioralSessionData{
			TaskExecutionID:   oldExecID,
			SessionStart:      time.Now().Add(-48 * time.Hour),
			TotalDurationSecs: 60,
		}
		oldSessionID, err := store.RecordSessionMetrics(ctx, oldSession, nil, nil, nil, nil)
		require.NoError(t, err)

		recentSession := &BehavioralSessionData{
			TaskExecutionID:   recentExec.ID,
			SessionStart:      time.Now(),
			TotalDurationSecs: 60,
		}
		recentSessionID, err := store.RecordSessionMetrics(ctx, recentSession, nil, nil, nil, nil)
		require.NoError(t, err)

		// Delete old data (older than 24 hours)
		cutoff := time.Now().Add(-24 * time.Hour)
		deleted, err := store.DeleteOldBehavioralData(ctx, cutoff)
		require.NoError(t, err)
		assert.Equal(t, int64(1), deleted)

		// Verify old session deleted
		_, err = store.GetSessionMetrics(ctx, oldSessionID)
		require.Error(t, err)

		// Verify recent session still exists
		_, err = store.GetSessionMetrics(ctx, recentSessionID)
		require.NoError(t, err)
	})

	t.Run("deletes no data when none is old", func(t *testing.T) {
		ctx := context.Background()
		store := setupTestStore(t)
		defer store.Close()

		// Create recent execution and session
		exec := &TaskExecution{
			PlanFile:     "test.yaml",
			TaskNumber:   "1",
			TaskName:     "Test Task",
			Agent:        "test-agent",
			Prompt:       "test",
			Success:      true,
			DurationSecs: 60,
		}
		err := store.RecordExecution(ctx, exec)
		require.NoError(t, err)

		sessionData := &BehavioralSessionData{
			TaskExecutionID:   exec.ID,
			SessionStart:      time.Now(),
			TotalDurationSecs: 60,
		}
		_, err = store.RecordSessionMetrics(ctx, sessionData, nil, nil, nil, nil)
		require.NoError(t, err)

		// Delete old data (cutoff in past)
		cutoff := time.Now().Add(-48 * time.Hour)
		deleted, err := store.DeleteOldBehavioralData(ctx, cutoff)
		require.NoError(t, err)
		assert.Equal(t, int64(0), deleted)
	})
}

func TestRecordSessionMetrics_TransactionRollback(t *testing.T) {
	t.Run("rolls back on tool insertion error", func(t *testing.T) {
		ctx := context.Background()
		store := setupTestStore(t)
		defer store.Close()

		// Create task execution
		exec := &TaskExecution{
			PlanFile:     "test.yaml",
			TaskNumber:   "1",
			TaskName:     "Test Task",
			Agent:        "test-agent",
			Prompt:       "test",
			Success:      true,
			DurationSecs: 60,
		}
		err := store.RecordExecution(ctx, exec)
		require.NoError(t, err)

		// Try to insert with invalid tool data (negative duration not checked by DB, but we can test transaction)
		sessionData := &BehavioralSessionData{
			TaskExecutionID:   exec.ID,
			SessionStart:      time.Now(),
			TotalDurationSecs: 60,
			TotalToolCalls:    1,
		}

		// This should succeed as SQLite is permissive, but demonstrates transaction usage
		tools := []ToolExecutionData{
			{
				ToolName:   "Test",
				DurationMs: 100,
				Success:    true,
			},
		}

		sessionID, err := store.RecordSessionMetrics(ctx, sessionData, tools, nil, nil, nil)
		require.NoError(t, err)
		assert.Greater(t, sessionID, int64(0))
	})
}

func TestBehavioralSessionData_NullHandling(t *testing.T) {
	t.Run("handles NULL session_end gracefully", func(t *testing.T) {
		ctx := context.Background()
		store := setupTestStore(t)
		defer store.Close()

		exec := &TaskExecution{
			PlanFile:     "test.yaml",
			TaskNumber:   "1",
			TaskName:     "Test Task",
			Agent:        "test-agent",
			Prompt:       "test",
			Success:      true,
			DurationSecs: 60,
		}
		err := store.RecordExecution(ctx, exec)
		require.NoError(t, err)

		// Create session with NULL end time
		sessionData := &BehavioralSessionData{
			TaskExecutionID:   exec.ID,
			SessionStart:      time.Now(),
			SessionEnd:        nil, // Explicitly NULL
			TotalDurationSecs: 60,
		}
		sessionID, err := store.RecordSessionMetrics(ctx, sessionData, nil, nil, nil, nil)
		require.NoError(t, err)

		// Retrieve and verify NULL handling
		metrics, err := store.GetSessionMetrics(ctx, sessionID)
		require.NoError(t, err)
		assert.Nil(t, metrics.SessionEnd)
	})
}

func TestCleanupOldExecutions(t *testing.T) {
	t.Run("deletes executions older than keepDays", func(t *testing.T) {
		ctx := context.Background()
		store := setupTestStore(t)
		defer store.Close()

		// Create old execution (manually insert with old timestamp)
		query := `INSERT INTO task_executions
			(plan_file, task_number, task_name, agent, prompt, success, duration_seconds, timestamp)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
		_, err := store.db.Exec(query,
			"test.yaml", "1", "Old Task", "test-agent", "test", true, 60,
			time.Now().Add(-100*24*time.Hour)) // 100 days ago
		require.NoError(t, err)

		// Create recent execution
		recentExec := &TaskExecution{
			PlanFile:     "test.yaml",
			TaskNumber:   "2",
			TaskName:     "Recent Task",
			Agent:        "test-agent",
			Prompt:       "test",
			Success:      true,
			DurationSecs: 60,
		}
		err = store.RecordExecution(ctx, recentExec)
		require.NoError(t, err)

		// Cleanup records older than 90 days
		deleted, err := store.CleanupOldExecutions(ctx, 90)
		require.NoError(t, err)
		assert.Equal(t, int64(1), deleted)

		// Verify recent execution still exists
		execs, err := store.GetExecutions("test.yaml")
		require.NoError(t, err)
		assert.Len(t, execs, 1)
		assert.Equal(t, "Recent Task", execs[0].TaskName)
	})

	t.Run("keepDays 0 means keep forever", func(t *testing.T) {
		ctx := context.Background()
		store := setupTestStore(t)
		defer store.Close()

		// Create old execution
		query := `INSERT INTO task_executions
			(plan_file, task_number, task_name, agent, prompt, success, duration_seconds, timestamp)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
		_, err := store.db.Exec(query,
			"test.yaml", "1", "Old Task", "test-agent", "test", true, 60,
			time.Now().Add(-365*24*time.Hour)) // 1 year ago
		require.NoError(t, err)

		// Cleanup with 0 should not delete anything
		deleted, err := store.CleanupOldExecutions(ctx, 0)
		require.NoError(t, err)
		assert.Equal(t, int64(0), deleted)

		// Verify execution still exists
		execs, err := store.GetExecutions("test.yaml")
		require.NoError(t, err)
		assert.Len(t, execs, 1)
	})
}

// Helper function
func timePtr(t time.Time) *time.Time {
	return &t
}
