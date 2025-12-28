package learning

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStore(t *testing.T) {
	tests := []struct {
		name    string
		dbPath  string
		wantErr bool
	}{
		{
			name:    "creates database successfully",
			dbPath:  filepath.Join(t.TempDir(), "test.db"),
			wantErr: false,
		},
		{
			name:    "handles in-memory database",
			dbPath:  ":memory:",
			wantErr: false,
		},
		{
			name:    "returns error for invalid path",
			dbPath:  "/invalid/nonexistent/deep/path/db.db",
			wantErr: true,
		},
		{
			name:    "creates parent directories if needed",
			dbPath:  filepath.Join(t.TempDir(), "nested", "dir", "test.db"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := NewStore(tt.dbPath)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, store)
			defer store.Close()

			// Verify schema initialized
			version, err := store.getSchemaVersion()
			require.NoError(t, err)
			assert.Equal(t, len(migrations), version)

			// Verify database path set correctly
			assert.Equal(t, tt.dbPath, store.dbPath)
		})
	}
}

func TestInitSchema(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "initializes schema successfully",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbPath := filepath.Join(t.TempDir(), "schema_test.db")
			store, err := NewStore(dbPath)
			require.NoError(t, err)
			defer store.Close()

			// Verify all tables exist
			tables := []string{
				"task_executions",
				"approach_history",
				"schema_version",
				"behavioral_sessions",
				"tool_executions",
				"bash_commands",
				"file_operations",
				"token_usage",
			}
			for _, table := range tables {
				exists, err := store.tableExists(table)
				require.NoError(t, err)
				assert.True(t, exists, "table %s should exist", table)
			}

			// Verify indexes exist
			indexes := []string{
				"idx_task_executions_task",
				"idx_task_executions_timestamp",
				"idx_approach_history_task",
				"idx_behavioral_sessions_task_id",
				"idx_tool_executions_session_id",
				"idx_bash_commands_session_id",
				"idx_file_operations_session_id",
				"idx_token_usage_session_id",
			}
			for _, index := range indexes {
				exists, err := store.indexExists(index)
				require.NoError(t, err)
				assert.True(t, exists, "index %s should exist", index)
			}
		})
	}
}

func TestStoreConnection(t *testing.T) {
	tests := []struct {
		name    string
		dbPath  string
		wantErr bool
	}{
		{
			name:    "can open and close connection",
			dbPath:  filepath.Join(t.TempDir(), "connection_test.db"),
			wantErr: false,
		},
		{
			name:    "can reopen closed store",
			dbPath:  filepath.Join(t.TempDir(), "reopen_test.db"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := NewStore(tt.dbPath)
			require.NoError(t, err)
			require.NotNil(t, store)

			// Close connection
			err = store.Close()
			require.NoError(t, err)

			// Verify can close multiple times without error
			err = store.Close()
			require.NoError(t, err)
		})
	}
}

func TestSchemaVersion(t *testing.T) {
	tests := []struct {
		name        string
		wantVersion int
		wantErr     bool
	}{
		{
			name:        "returns latest version for new database",
			wantVersion: len(migrations),
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbPath := filepath.Join(t.TempDir(), "version_test.db")
			store, err := NewStore(dbPath)
			require.NoError(t, err)
			defer store.Close()

			version, err := store.getSchemaVersion()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantVersion, version)
		})
	}
}

func TestDatabasePath(t *testing.T) {
	tests := []struct {
		name      string
		dbPath    string
		wantErr   bool
		checkFile bool
	}{
		{
			name:      "handles absolute path",
			dbPath:    filepath.Join(t.TempDir(), "absolute", "test.db"),
			wantErr:   false,
			checkFile: true,
		},
		{
			name:      "handles in-memory path",
			dbPath:    ":memory:",
			wantErr:   false,
			checkFile: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := NewStore(tt.dbPath)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			defer store.Close()

			// Check file exists if not in-memory
			if tt.checkFile && tt.dbPath != ":memory:" {
				_, err := os.Stat(tt.dbPath)
				require.NoError(t, err, "database file should exist")
			}
		})
	}
}

func TestSchemaIdempotency(t *testing.T) {
	t.Run("schema initialization is idempotent", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "idempotent_test.db")

		// Initialize first time
		store1, err := NewStore(dbPath)
		require.NoError(t, err)
		version1, err := store1.getSchemaVersion()
		require.NoError(t, err)
		store1.Close()

		// Initialize second time with existing database
		store2, err := NewStore(dbPath)
		require.NoError(t, err)
		defer store2.Close()

		version2, err := store2.getSchemaVersion()
		require.NoError(t, err)

		// Version should remain the same
		assert.Equal(t, version1, version2)
		assert.Equal(t, len(migrations), version2)
	})
}

func TestConcurrentInitialization(t *testing.T) {
	t.Run("handles concurrent initialization attempts", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "concurrent_test.db")

		// Create multiple stores concurrently
		stores := make([]*Store, 3)
		errs := make([]error, 3)

		done := make(chan bool)
		for i := 0; i < 3; i++ {
			go func(idx int) {
				stores[idx], errs[idx] = NewStore(dbPath)
				done <- true
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < 3; i++ {
			<-done
		}

		// All should succeed
		for i := 0; i < 3; i++ {
			require.NoError(t, errs[i])
			require.NotNil(t, stores[i])
			defer stores[i].Close()

			version, err := stores[i].getSchemaVersion()
			require.NoError(t, err)
			assert.Equal(t, len(migrations), version)
		}
	})
}

// setupTestStore creates a test store with in-memory database
func setupTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := NewStore(":memory:")
	require.NoError(t, err)
	return store
}

func TestRecordExecution(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		exec    *TaskExecution
		wantErr bool
	}{
		{
			name: "records execution successfully",
			exec: &TaskExecution{
				PlanFile:     "test.md",
				TaskNumber:   "Task 1",
				TaskName:     "Setup database",
				Agent:        "golang-pro",
				Prompt:       "Initialize PostgreSQL database",
				Success:      true,
				Output:       "Database initialized successfully",
				DurationSecs: 30,
				QCVerdict:    "GREEN",
			},
			wantErr: false,
		},
		{
			name: "records failed execution with QC feedback",
			exec: &TaskExecution{
				PlanFile:     "test.md",
				TaskNumber:   "Task 2",
				TaskName:     "Run tests",
				Agent:        "test-automator",
				Prompt:       "Run all unit tests",
				Success:      false,
				ErrorMessage: "3 tests failed",
				Output:       "FAIL: TestFoo",
				DurationSecs: 45,
				QCVerdict:    "RED",
				QCFeedback:   "Tests are failing due to missing imports",
			},
			wantErr: false,
		},
		{
			name: "handles empty optional fields",
			exec: &TaskExecution{
				PlanFile:   "test.md",
				TaskNumber: "Task 3",
				TaskName:   "Build project",
				Prompt:     "Build the project",
				Success:    true,
			},
			wantErr: false,
		},
		{
			name: "handles very long output",
			exec: &TaskExecution{
				PlanFile:     "test.md",
				TaskNumber:   "Task 4",
				TaskName:     "Generate report",
				Prompt:       "Generate large report",
				Success:      true,
				Output:       string(make([]byte, 10000)), // 10KB output
				DurationSecs: 60,
			},
			wantErr: false,
		},
		{
			name: "records execution with failure patterns",
			exec: &TaskExecution{
				PlanFile:        "test.md",
				TaskNumber:      "Task 5",
				TaskName:        "Fix compilation",
				Agent:           "golang-pro",
				Prompt:          "Fix compilation errors",
				Success:         false,
				QCVerdict:       "RED",
				FailurePatterns: []string{"compilation_error", "missing_import"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := setupTestStore(t)
			defer store.Close()

			err := store.RecordExecution(ctx, tt.exec)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Verify record exists by querying
			var count int
			err = store.db.QueryRow("SELECT COUNT(*) FROM task_executions WHERE task_number = ?", tt.exec.TaskNumber).Scan(&count)
			require.NoError(t, err)
			assert.Equal(t, 1, count)

			// Verify all fields were stored correctly
			history, err := store.GetExecutionHistory(ctx, tt.exec.PlanFile, tt.exec.TaskNumber)
			require.NoError(t, err)
			require.Len(t, history, 1)
			assert.Equal(t, tt.exec.QCVerdict, history[0].QCVerdict)
			assert.Equal(t, tt.exec.QCFeedback, history[0].QCFeedback)
			// Note: nil FailurePatterns becomes empty slice after JSON round-trip
			if tt.exec.FailurePatterns == nil {
				assert.Empty(t, history[0].FailurePatterns)
			} else {
				assert.Equal(t, tt.exec.FailurePatterns, history[0].FailurePatterns)
			}
		})
	}
}

func TestRecordExecution_Concurrent(t *testing.T) {
	t.Run("handles concurrent writes without corruption", func(t *testing.T) {
		ctx := context.Background()
		// Use file-based DB for better concurrent access support
		dbPath := filepath.Join(t.TempDir(), "concurrent.db")
		store, err := NewStore(dbPath)
		require.NoError(t, err)
		defer store.Close()

		// Verify table exists before starting concurrent operations
		var count int
		err = store.db.QueryRow("SELECT COUNT(*) FROM task_executions").Scan(&count)
		require.NoError(t, err)

		// Record 10 executions concurrently
		done := make(chan error, 10)
		for i := 0; i < 10; i++ {
			go func(idx int) {
				exec := &TaskExecution{
					PlanFile:     "test.md",
					TaskNumber:   fmt.Sprintf("Task %d", idx),
					TaskName:     fmt.Sprintf("Concurrent task %d", idx),
					Prompt:       fmt.Sprintf("Test prompt %d", idx),
					Success:      idx%2 == 0,
					DurationSecs: int64(idx * 10),
				}
				done <- store.RecordExecution(ctx, exec)
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			err := <-done
			require.NoError(t, err)
		}

		// Verify all records exist
		err = store.db.QueryRow("SELECT COUNT(*) FROM task_executions").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 10, count)
	})
}

func TestGetExecutionHistory(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name       string
		setup      func(*Store)
		planFile   string
		taskNumber string
		wantCount  int
		wantErr    bool
	}{
		{
			name: "returns history for task",
			setup: func(s *Store) {
				executions := []*TaskExecution{
					{PlanFile: "test.md", TaskNumber: "Task 1", TaskName: "Test A", Prompt: "prompt", Success: true},
					{PlanFile: "test.md", TaskNumber: "Task 1", TaskName: "Test A", Prompt: "prompt", Success: false},
					{PlanFile: "test.md", TaskNumber: "Task 2", TaskName: "Test B", Prompt: "prompt", Success: true},
				}
				for _, exec := range executions {
					require.NoError(t, s.RecordExecution(ctx, exec))
				}
			},
			planFile:   "test.md",
			taskNumber: "Task 1",
			wantCount:  2,
			wantErr:    false,
		},
		{
			name: "returns empty for nonexistent task",
			setup: func(s *Store) {
				exec := &TaskExecution{PlanFile: "test.md", TaskNumber: "Task 1", TaskName: "Test", Prompt: "prompt", Success: true}
				require.NoError(t, s.RecordExecution(ctx, exec))
			},
			planFile:   "test.md",
			taskNumber: "Task 99",
			wantCount:  0,
			wantErr:    false,
		},
		{
			name:       "handles empty database",
			setup:      func(s *Store) {},
			planFile:   "test.md",
			taskNumber: "Task 1",
			wantCount:  0,
			wantErr:    false,
		},
		{
			name: "filters by plan file correctly",
			setup: func(s *Store) {
				executions := []*TaskExecution{
					{PlanFile: "plan1.md", TaskNumber: "Task 1", TaskName: "Test", Prompt: "prompt", Success: true},
					{PlanFile: "plan2.md", TaskNumber: "Task 1", TaskName: "Test", Prompt: "prompt", Success: true},
					{PlanFile: "plan1.md", TaskNumber: "Task 1", TaskName: "Test", Prompt: "prompt", Success: false},
				}
				for _, exec := range executions {
					require.NoError(t, s.RecordExecution(ctx, exec))
				}
			},
			planFile:   "plan1.md",
			taskNumber: "Task 1",
			wantCount:  2,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := setupTestStore(t)
			defer store.Close()

			if tt.setup != nil {
				tt.setup(store)
			}

			history, err := store.GetExecutionHistory(ctx, tt.planFile, tt.taskNumber)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, history, tt.wantCount)
		})
	}
}

func TestGetExecutionHistory_Ordering(t *testing.T) {
	t.Run("returns history in chronological order", func(t *testing.T) {
		ctx := context.Background()
		store := setupTestStore(t)
		defer store.Close()

		// Insert 3 executions with slight delays
		for i := 1; i <= 3; i++ {
			exec := &TaskExecution{
				PlanFile:     "test.md",
				TaskNumber:   "Task 1",
				TaskName:     fmt.Sprintf("Attempt %d", i),
				Prompt:       "test",
				Success:      i == 3,
				DurationSecs: int64(i),
			}
			require.NoError(t, store.RecordExecution(ctx, exec))
		}

		history, err := store.GetExecutionHistory(ctx, "test.md", "Task 1")
		require.NoError(t, err)
		require.Len(t, history, 3)

		// Verify chronological order (newest first)
		assert.Equal(t, int64(3), history[0].DurationSecs)
		assert.Equal(t, int64(2), history[1].DurationSecs)
		assert.Equal(t, int64(1), history[2].DurationSecs)
	})
}

func TestRecordApproach(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name     string
		approach *ApproachHistory
		wantErr  bool
	}{
		{
			name: "records new approach",
			approach: &ApproachHistory{
				TaskPattern:         "test-fix",
				ApproachDescription: "Use table-driven tests",
				SuccessCount:        1,
				FailureCount:        0,
			},
			wantErr: false,
		},
		{
			name: "records failed approach",
			approach: &ApproachHistory{
				TaskPattern:         "build-error",
				ApproachDescription: "Update dependencies",
				SuccessCount:        0,
				FailureCount:        1,
			},
			wantErr: false,
		},
		{
			name: "handles empty metadata",
			approach: &ApproachHistory{
				TaskPattern:         "refactor",
				ApproachDescription: "Extract function",
				SuccessCount:        1,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := setupTestStore(t)
			defer store.Close()

			err := store.RecordApproach(ctx, tt.approach)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Verify record exists
			var count int
			err = store.db.QueryRow("SELECT COUNT(*) FROM approach_history WHERE task_pattern = ?", tt.approach.TaskPattern).Scan(&count)
			require.NoError(t, err)
			assert.Equal(t, 1, count)
		})
	}
}

func TestGetRunCount_NoExecutions(t *testing.T) {
	t.Run("returns 0 for plan never executed", func(t *testing.T) {
		ctx := context.Background()
		store := setupTestStore(t)
		defer store.Close()

		count, err := store.GetRunCount(ctx, "never-executed.md")
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

func TestGetRunCount_MultipleRuns(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name     string
		planFile string
		setup    func(*Store, string)
		expected int
	}{
		{
			name:     "single run with one task",
			planFile: "plan.md",
			setup: func(s *Store, pf string) {
				exec := &TaskExecution{
					PlanFile:   pf,
					RunNumber:  1,
					TaskNumber: "Task 1",
					TaskName:   "Test task",
					Prompt:     "test",
					Success:    true,
				}
				require.NoError(t, s.RecordExecution(ctx, exec))
			},
			expected: 1,
		},
		{
			name:     "multiple runs with multiple tasks",
			planFile: "plan2.md",
			setup: func(s *Store, pf string) {
				// Run 1
				for i := 1; i <= 3; i++ {
					exec := &TaskExecution{
						PlanFile:   pf,
						RunNumber:  1,
						TaskNumber: fmt.Sprintf("Task %d", i),
						TaskName:   fmt.Sprintf("Test task %d", i),
						Prompt:     "test",
						Success:    true,
					}
					require.NoError(t, s.RecordExecution(ctx, exec))
				}
				// Run 2
				for i := 1; i <= 3; i++ {
					exec := &TaskExecution{
						PlanFile:   pf,
						RunNumber:  2,
						TaskNumber: fmt.Sprintf("Task %d", i),
						TaskName:   fmt.Sprintf("Test task %d", i),
						Prompt:     "test",
						Success:    true,
					}
					require.NoError(t, s.RecordExecution(ctx, exec))
				}
				// Run 3
				for i := 1; i <= 3; i++ {
					exec := &TaskExecution{
						PlanFile:   pf,
						RunNumber:  3,
						TaskNumber: fmt.Sprintf("Task %d", i),
						TaskName:   fmt.Sprintf("Test task %d", i),
						Prompt:     "test",
						Success:    true,
					}
					require.NoError(t, s.RecordExecution(ctx, exec))
				}
			},
			expected: 3,
		},
		{
			name:     "different plan files don't interfere",
			planFile: "plan3.md",
			setup: func(s *Store, pf string) {
				// Create executions for different plan file
				other := &TaskExecution{
					PlanFile:   "other.md",
					RunNumber:  5,
					TaskNumber: "Task 1",
					TaskName:   "Test",
					Prompt:     "test",
					Success:    true,
				}
				require.NoError(t, s.RecordExecution(ctx, other))

				// Create executions for target plan file
				exec := &TaskExecution{
					PlanFile:   pf,
					RunNumber:  2,
					TaskNumber: "Task 1",
					TaskName:   "Test",
					Prompt:     "test",
					Success:    true,
				}
				require.NoError(t, s.RecordExecution(ctx, exec))
			},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := setupTestStore(t)
			defer store.Close()

			if tt.setup != nil {
				tt.setup(store, tt.planFile)
			}

			count, err := store.GetRunCount(ctx, tt.planFile)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, count)
		})
	}
}

func TestGetExecutions(t *testing.T) {
	t.Run("retrieves all executions with QC fields", func(t *testing.T) {
		store := setupTestStore(t)
		defer store.Close()

		ctx := context.Background()
		planFile := "test-plan.yaml"

		// Record executions with QC fields
		exec1 := &TaskExecution{
			PlanFile:        planFile,
			RunNumber:       1,
			TaskNumber:      "Task 1",
			TaskName:        "First Task",
			Agent:           "test-agent",
			Prompt:          "Do something",
			Success:         true,
			Output:          "Success output",
			DurationSecs:    120,
			QCVerdict:       "GREEN",
			QCFeedback:      "Looks good!",
			FailurePatterns: []string{"pattern1", "pattern2"},
		}
		require.NoError(t, store.RecordExecution(ctx, exec1))

		exec2 := &TaskExecution{
			PlanFile:        planFile,
			RunNumber:       1,
			TaskNumber:      "Task 2",
			TaskName:        "Second Task",
			Agent:           "test-agent-2",
			Prompt:          "Do something else",
			Success:         false,
			Output:          "Failed output",
			ErrorMessage:    "Something went wrong",
			DurationSecs:    60,
			QCVerdict:       "RED",
			QCFeedback:      "Needs improvement",
			FailurePatterns: []string{"compilation_error"},
		}
		require.NoError(t, store.RecordExecution(ctx, exec2))

		// Retrieve executions
		executions, err := store.GetExecutions(planFile)
		require.NoError(t, err)
		require.Len(t, executions, 2)

		// Verify first execution (most recent, so exec2)
		assert.Equal(t, "Task 2", executions[0].TaskNumber)
		assert.Equal(t, "Second Task", executions[0].TaskName)
		assert.Equal(t, "test-agent-2", executions[0].Agent)
		assert.Equal(t, "RED", executions[0].QCVerdict)
		assert.Equal(t, "Needs improvement", executions[0].QCFeedback)
		assert.Equal(t, []string{"compilation_error"}, executions[0].FailurePatterns)
		assert.False(t, executions[0].Success)

		// Verify second execution (older, so exec1)
		assert.Equal(t, "Task 1", executions[1].TaskNumber)
		assert.Equal(t, "First Task", executions[1].TaskName)
		assert.Equal(t, "test-agent", executions[1].Agent)
		assert.Equal(t, "GREEN", executions[1].QCVerdict)
		assert.Equal(t, "Looks good!", executions[1].QCFeedback)
		assert.Equal(t, []string{"pattern1", "pattern2"}, executions[1].FailurePatterns)
		assert.True(t, executions[1].Success)
	})

	t.Run("returns empty for nonexistent plan file", func(t *testing.T) {
		store := setupTestStore(t)
		defer store.Close()

		executions, err := store.GetExecutions("nonexistent.yaml")
		require.NoError(t, err)
		assert.Empty(t, executions)
	})

	t.Run("handles executions without QC fields", func(t *testing.T) {
		store := setupTestStore(t)
		defer store.Close()

		ctx := context.Background()
		planFile := "minimal-plan.yaml"

		// Record execution without QC fields
		exec := &TaskExecution{
			PlanFile:     planFile,
			RunNumber:    1,
			TaskNumber:   "Task 1",
			TaskName:     "Minimal Task",
			Prompt:       "Do something",
			Success:      true,
			DurationSecs: 30,
		}
		require.NoError(t, store.RecordExecution(ctx, exec))

		// Retrieve executions
		executions, err := store.GetExecutions(planFile)
		require.NoError(t, err)
		require.Len(t, executions, 1)

		// Verify fields are empty (failure patterns will be empty slice, not nil)
		assert.Equal(t, "", executions[0].QCVerdict)
		assert.Equal(t, "", executions[0].QCFeedback)
		assert.Empty(t, executions[0].FailurePatterns)
	})

	t.Run("filters by plan file correctly", func(t *testing.T) {
		store := setupTestStore(t)
		defer store.Close()

		ctx := context.Background()

		// Record executions for different plan files
		exec1 := &TaskExecution{
			PlanFile:   "plan-a.yaml",
			RunNumber:  1,
			TaskNumber: "Task 1",
			TaskName:   "Task A",
			Prompt:     "test",
			Success:    true,
			QCVerdict:  "GREEN",
		}
		require.NoError(t, store.RecordExecution(ctx, exec1))

		exec2 := &TaskExecution{
			PlanFile:   "plan-b.yaml",
			RunNumber:  1,
			TaskNumber: "Task 1",
			TaskName:   "Task B",
			Prompt:     "test",
			Success:    false,
			QCVerdict:  "RED",
		}
		require.NoError(t, store.RecordExecution(ctx, exec2))

		// Retrieve executions for plan-a only
		executions, err := store.GetExecutions("plan-a.yaml")
		require.NoError(t, err)
		require.Len(t, executions, 1)
		assert.Equal(t, "plan-a.yaml", executions[0].PlanFile)
		assert.Equal(t, "GREEN", executions[0].QCVerdict)

		// Retrieve executions for plan-b only
		executions, err = store.GetExecutions("plan-b.yaml")
		require.NoError(t, err)
		require.Len(t, executions, 1)
		assert.Equal(t, "plan-b.yaml", executions[0].PlanFile)
		assert.Equal(t, "RED", executions[0].QCVerdict)
	})
}

// TestGetFileOffset tests the GetFileOffset method
func TestGetFileOffset(t *testing.T) {
	ctx := context.Background()

	t.Run("returns nil for non-existent file", func(t *testing.T) {
		store := setupTestStore(t)
		defer store.Close()

		offset, err := store.GetFileOffset(ctx, "/path/to/nonexistent.jsonl")
		require.NoError(t, err)
		assert.Nil(t, offset)
	})

	t.Run("returns offset for tracked file", func(t *testing.T) {
		store := setupTestStore(t)
		defer store.Close()

		// First set an offset
		setOffset := &IngestOffset{
			FilePath:     "/path/to/test.jsonl",
			ByteOffset:   12345,
			Inode:        98765,
			LastLineHash: "abc123def456",
		}
		require.NoError(t, store.SetFileOffset(ctx, setOffset))

		// Now retrieve it
		offset, err := store.GetFileOffset(ctx, "/path/to/test.jsonl")
		require.NoError(t, err)
		require.NotNil(t, offset)
		assert.Equal(t, "/path/to/test.jsonl", offset.FilePath)
		assert.Equal(t, int64(12345), offset.ByteOffset)
		assert.Equal(t, uint64(98765), offset.Inode)
		assert.Equal(t, "abc123def456", offset.LastLineHash)
		assert.False(t, offset.UpdatedAt.IsZero())
	})
}

// TestSetFileOffset tests the SetFileOffset method
func TestSetFileOffset(t *testing.T) {
	ctx := context.Background()

	t.Run("creates new offset", func(t *testing.T) {
		store := setupTestStore(t)
		defer store.Close()

		offset := &IngestOffset{
			FilePath:     "/path/to/new.jsonl",
			ByteOffset:   5000,
			Inode:        12345,
			LastLineHash: "hash123",
		}
		err := store.SetFileOffset(ctx, offset)
		require.NoError(t, err)
		assert.Greater(t, offset.ID, int64(0))

		// Verify by retrieving
		retrieved, err := store.GetFileOffset(ctx, "/path/to/new.jsonl")
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		assert.Equal(t, int64(5000), retrieved.ByteOffset)
	})

	t.Run("updates existing offset (upsert)", func(t *testing.T) {
		store := setupTestStore(t)
		defer store.Close()

		// Create initial offset
		offset := &IngestOffset{
			FilePath:     "/path/to/update.jsonl",
			ByteOffset:   1000,
			Inode:        111,
			LastLineHash: "initial",
		}
		require.NoError(t, store.SetFileOffset(ctx, offset))

		// Update the offset
		offset.ByteOffset = 9999
		offset.LastLineHash = "updated"
		require.NoError(t, store.SetFileOffset(ctx, offset))

		// Verify update
		retrieved, err := store.GetFileOffset(ctx, "/path/to/update.jsonl")
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		assert.Equal(t, int64(9999), retrieved.ByteOffset)
		assert.Equal(t, "updated", retrieved.LastLineHash)
	})

	t.Run("handles zero values", func(t *testing.T) {
		store := setupTestStore(t)
		defer store.Close()

		offset := &IngestOffset{
			FilePath:   "/path/to/zero.jsonl",
			ByteOffset: 0,
			Inode:      0,
		}
		err := store.SetFileOffset(ctx, offset)
		require.NoError(t, err)

		retrieved, err := store.GetFileOffset(ctx, "/path/to/zero.jsonl")
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		assert.Equal(t, int64(0), retrieved.ByteOffset)
		assert.Equal(t, uint64(0), retrieved.Inode)
	})
}

// TestDeleteFileOffset tests the DeleteFileOffset method
func TestDeleteFileOffset(t *testing.T) {
	ctx := context.Background()

	t.Run("deletes existing offset", func(t *testing.T) {
		store := setupTestStore(t)
		defer store.Close()

		// Create offset
		offset := &IngestOffset{
			FilePath:   "/path/to/delete.jsonl",
			ByteOffset: 1000,
		}
		require.NoError(t, store.SetFileOffset(ctx, offset))

		// Verify it exists
		retrieved, err := store.GetFileOffset(ctx, "/path/to/delete.jsonl")
		require.NoError(t, err)
		require.NotNil(t, retrieved)

		// Delete it
		err = store.DeleteFileOffset(ctx, "/path/to/delete.jsonl")
		require.NoError(t, err)

		// Verify it's gone
		retrieved, err = store.GetFileOffset(ctx, "/path/to/delete.jsonl")
		require.NoError(t, err)
		assert.Nil(t, retrieved)
	})

	t.Run("no error when deleting non-existent file", func(t *testing.T) {
		store := setupTestStore(t)
		defer store.Close()

		err := store.DeleteFileOffset(ctx, "/path/to/nonexistent.jsonl")
		require.NoError(t, err)
	})
}

// TestListFileOffsets tests the ListFileOffsets method
func TestListFileOffsets(t *testing.T) {
	ctx := context.Background()

	t.Run("returns empty list when no offsets", func(t *testing.T) {
		store := setupTestStore(t)
		defer store.Close()

		offsets, err := store.ListFileOffsets(ctx)
		require.NoError(t, err)
		assert.Empty(t, offsets)
	})

	t.Run("returns all offsets sorted by path", func(t *testing.T) {
		store := setupTestStore(t)
		defer store.Close()

		// Create multiple offsets in non-alphabetical order
		paths := []string{"/z/file.jsonl", "/a/file.jsonl", "/m/file.jsonl"}
		for i, path := range paths {
			offset := &IngestOffset{
				FilePath:   path,
				ByteOffset: int64(i * 1000),
				Inode:      uint64(i + 1),
			}
			require.NoError(t, store.SetFileOffset(ctx, offset))
		}

		offsets, err := store.ListFileOffsets(ctx)
		require.NoError(t, err)
		require.Len(t, offsets, 3)

		// Verify sorted order
		assert.Equal(t, "/a/file.jsonl", offsets[0].FilePath)
		assert.Equal(t, "/m/file.jsonl", offsets[1].FilePath)
		assert.Equal(t, "/z/file.jsonl", offsets[2].FilePath)
	})

	t.Run("returns correct data for each offset", func(t *testing.T) {
		store := setupTestStore(t)
		defer store.Close()

		offset := &IngestOffset{
			FilePath:     "/test/data.jsonl",
			ByteOffset:   54321,
			Inode:        11111,
			LastLineHash: "testhash",
		}
		require.NoError(t, store.SetFileOffset(ctx, offset))

		offsets, err := store.ListFileOffsets(ctx)
		require.NoError(t, err)
		require.Len(t, offsets, 1)

		assert.Equal(t, "/test/data.jsonl", offsets[0].FilePath)
		assert.Equal(t, int64(54321), offsets[0].ByteOffset)
		assert.Equal(t, uint64(11111), offsets[0].Inode)
		assert.Equal(t, "testhash", offsets[0].LastLineHash)
		assert.False(t, offsets[0].UpdatedAt.IsZero())
	})
}

// TestUpsertClaudeSession tests the UpsertClaudeSession method
func TestUpsertClaudeSession(t *testing.T) {
	ctx := context.Background()

	t.Run("creates new session", func(t *testing.T) {
		store := setupTestStore(t)
		defer store.Close()

		sessionID, isNew, err := store.UpsertClaudeSession(ctx,
			"claude-session-123",
			"/path/to/project",
			"golang-pro",
		)
		require.NoError(t, err)
		assert.True(t, isNew)
		assert.Greater(t, sessionID, int64(0))
	})

	t.Run("returns existing session on duplicate", func(t *testing.T) {
		store := setupTestStore(t)
		defer store.Close()

		// First call creates session
		sessionID1, isNew1, err := store.UpsertClaudeSession(ctx,
			"claude-session-456",
			"/path/to/project",
			"golang-pro",
		)
		require.NoError(t, err)
		assert.True(t, isNew1)

		// Second call returns existing session
		sessionID2, isNew2, err := store.UpsertClaudeSession(ctx,
			"claude-session-456",
			"/path/to/project",
			"golang-pro",
		)
		require.NoError(t, err)
		assert.False(t, isNew2)
		assert.Equal(t, sessionID1, sessionID2)
	})

	t.Run("handles different external IDs", func(t *testing.T) {
		store := setupTestStore(t)
		defer store.Close()

		id1, _, err := store.UpsertClaudeSession(ctx, "session-a", "/proj", "agent1")
		require.NoError(t, err)

		id2, _, err := store.UpsertClaudeSession(ctx, "session-b", "/proj", "agent2")
		require.NoError(t, err)

		assert.NotEqual(t, id1, id2)
	})

	t.Run("handles empty optional fields", func(t *testing.T) {
		store := setupTestStore(t)
		defer store.Close()

		sessionID, isNew, err := store.UpsertClaudeSession(ctx, "session-minimal", "", "")
		require.NoError(t, err)
		assert.True(t, isNew)
		assert.Greater(t, sessionID, int64(0))
	})
}

// TestAppendToolExecution tests the AppendToolExecution method
func TestAppendToolExecution(t *testing.T) {
	ctx := context.Background()

	t.Run("appends tool execution to session", func(t *testing.T) {
		store := setupTestStore(t)
		defer store.Close()

		// Create session first
		sessionID, _, err := store.UpsertClaudeSession(ctx, "tool-session", "/proj", "agent")
		require.NoError(t, err)

		// Append tool execution
		tool := ToolExecutionData{
			ToolName:     "Read",
			Parameters:   `{"file_path": "/test.go"}`,
			DurationMs:   150,
			Success:      true,
			ErrorMessage: "",
		}
		err = store.AppendToolExecution(ctx, sessionID, tool)
		require.NoError(t, err)

		// Verify by querying
		var count int
		err = store.db.QueryRow("SELECT COUNT(*) FROM tool_executions WHERE session_id = ?", sessionID).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("appends multiple tool executions", func(t *testing.T) {
		store := setupTestStore(t)
		defer store.Close()

		sessionID, _, err := store.UpsertClaudeSession(ctx, "multi-tool-session", "/proj", "agent")
		require.NoError(t, err)

		for i := 0; i < 5; i++ {
			tool := ToolExecutionData{
				ToolName:   fmt.Sprintf("Tool%d", i),
				DurationMs: int64(i * 100),
				Success:    true,
			}
			err = store.AppendToolExecution(ctx, sessionID, tool)
			require.NoError(t, err)
		}

		var count int
		err = store.db.QueryRow("SELECT COUNT(*) FROM tool_executions WHERE session_id = ?", sessionID).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 5, count)
	})

	t.Run("records failed tool execution", func(t *testing.T) {
		store := setupTestStore(t)
		defer store.Close()

		sessionID, _, err := store.UpsertClaudeSession(ctx, "failed-tool-session", "/proj", "agent")
		require.NoError(t, err)

		tool := ToolExecutionData{
			ToolName:     "Bash",
			Parameters:   `{"command": "invalid"}`,
			DurationMs:   50,
			Success:      false,
			ErrorMessage: "command not found",
		}
		err = store.AppendToolExecution(ctx, sessionID, tool)
		require.NoError(t, err)

		var errorMsg string
		err = store.db.QueryRow("SELECT error_message FROM tool_executions WHERE session_id = ?", sessionID).Scan(&errorMsg)
		require.NoError(t, err)
		assert.Equal(t, "command not found", errorMsg)
	})
}

// TestAppendBashCommand tests the AppendBashCommand method
func TestAppendBashCommand(t *testing.T) {
	ctx := context.Background()

	t.Run("appends bash command to session", func(t *testing.T) {
		store := setupTestStore(t)
		defer store.Close()

		sessionID, _, err := store.UpsertClaudeSession(ctx, "bash-session", "/proj", "agent")
		require.NoError(t, err)

		bash := BashCommandData{
			Command:      "go test ./...",
			DurationMs:   5000,
			ExitCode:     0,
			StdoutLength: 1234,
			StderrLength: 0,
			Success:      true,
		}
		err = store.AppendBashCommand(ctx, sessionID, bash)
		require.NoError(t, err)

		var count int
		err = store.db.QueryRow("SELECT COUNT(*) FROM bash_commands WHERE session_id = ?", sessionID).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("records failed bash command with exit code", func(t *testing.T) {
		store := setupTestStore(t)
		defer store.Close()

		sessionID, _, err := store.UpsertClaudeSession(ctx, "failed-bash-session", "/proj", "agent")
		require.NoError(t, err)

		bash := BashCommandData{
			Command:      "exit 1",
			DurationMs:   10,
			ExitCode:     1,
			StdoutLength: 0,
			StderrLength: 100,
			Success:      false,
		}
		err = store.AppendBashCommand(ctx, sessionID, bash)
		require.NoError(t, err)

		var exitCode int
		var success bool
		err = store.db.QueryRow("SELECT exit_code, success FROM bash_commands WHERE session_id = ?", sessionID).Scan(&exitCode, &success)
		require.NoError(t, err)
		assert.Equal(t, 1, exitCode)
		assert.False(t, success)
	})
}

// TestAppendFileOperation tests the AppendFileOperation method
func TestAppendFileOperation(t *testing.T) {
	ctx := context.Background()

	t.Run("appends file operation to session", func(t *testing.T) {
		store := setupTestStore(t)
		defer store.Close()

		sessionID, _, err := store.UpsertClaudeSession(ctx, "file-session", "/proj", "agent")
		require.NoError(t, err)

		fileOp := FileOperationData{
			OperationType: "write",
			FilePath:      "/path/to/file.go",
			DurationMs:    25,
			BytesAffected: 1024,
			Success:       true,
			ErrorMessage:  "",
		}
		err = store.AppendFileOperation(ctx, sessionID, fileOp)
		require.NoError(t, err)

		var count int
		err = store.db.QueryRow("SELECT COUNT(*) FROM file_operations WHERE session_id = ?", sessionID).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("records failed file operation", func(t *testing.T) {
		store := setupTestStore(t)
		defer store.Close()

		sessionID, _, err := store.UpsertClaudeSession(ctx, "failed-file-session", "/proj", "agent")
		require.NoError(t, err)

		fileOp := FileOperationData{
			OperationType: "read",
			FilePath:      "/nonexistent/file.go",
			DurationMs:    5,
			BytesAffected: 0,
			Success:       false,
			ErrorMessage:  "file not found",
		}
		err = store.AppendFileOperation(ctx, sessionID, fileOp)
		require.NoError(t, err)

		var errorMsg string
		err = store.db.QueryRow("SELECT error_message FROM file_operations WHERE session_id = ?", sessionID).Scan(&errorMsg)
		require.NoError(t, err)
		assert.Equal(t, "file not found", errorMsg)
	})
}

// TestUpdateSessionAggregates tests the UpdateSessionAggregates method
func TestUpdateSessionAggregates(t *testing.T) {
	ctx := context.Background()

	t.Run("updates session aggregates", func(t *testing.T) {
		store := setupTestStore(t)
		defer store.Close()

		sessionID, _, err := store.UpsertClaudeSession(ctx, "agg-session", "/proj", "agent")
		require.NoError(t, err)

		err = store.UpdateSessionAggregates(ctx, sessionID, 1000, 60)
		require.NoError(t, err)

		var tokens, duration int64
		err = store.db.QueryRow("SELECT total_tokens_used, total_duration_seconds FROM behavioral_sessions WHERE id = ?", sessionID).Scan(&tokens, &duration)
		require.NoError(t, err)
		assert.Equal(t, int64(1000), tokens)
		assert.Equal(t, int64(60), duration)
	})

	t.Run("increments aggregates cumulatively", func(t *testing.T) {
		store := setupTestStore(t)
		defer store.Close()

		sessionID, _, err := store.UpsertClaudeSession(ctx, "cumulative-session", "/proj", "agent")
		require.NoError(t, err)

		// First update
		err = store.UpdateSessionAggregates(ctx, sessionID, 500, 30)
		require.NoError(t, err)

		// Second update
		err = store.UpdateSessionAggregates(ctx, sessionID, 700, 45)
		require.NoError(t, err)

		var tokens, duration int64
		err = store.db.QueryRow("SELECT total_tokens_used, total_duration_seconds FROM behavioral_sessions WHERE id = ?", sessionID).Scan(&tokens, &duration)
		require.NoError(t, err)
		assert.Equal(t, int64(1200), tokens)
		assert.Equal(t, int64(75), duration)
	})

	t.Run("returns error for nonexistent session", func(t *testing.T) {
		store := setupTestStore(t)
		defer store.Close()

		err := store.UpdateSessionAggregates(ctx, 99999, 100, 10)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "session not found")
	})
}

// TestIncrementalSessionWorkflow tests the complete incremental session workflow
func TestIncrementalSessionWorkflow(t *testing.T) {
	t.Run("complete incremental session workflow", func(t *testing.T) {
		ctx := context.Background()
		store := setupTestStore(t)
		defer store.Close()

		// 1. Create session
		sessionID, isNew, err := store.UpsertClaudeSession(ctx,
			"workflow-session-123",
			"/Users/test/project",
			"golang-pro",
		)
		require.NoError(t, err)
		assert.True(t, isNew)

		// 2. Append tool executions
		tools := []ToolExecutionData{
			{ToolName: "Read", Parameters: `{"file": "main.go"}`, DurationMs: 50, Success: true},
			{ToolName: "Edit", Parameters: `{"file": "main.go"}`, DurationMs: 100, Success: true},
			{ToolName: "Grep", Parameters: `{"pattern": "TODO"}`, DurationMs: 75, Success: true},
		}
		for _, tool := range tools {
			require.NoError(t, store.AppendToolExecution(ctx, sessionID, tool))
		}

		// 3. Append bash commands
		bashCmds := []BashCommandData{
			{Command: "go build", DurationMs: 2000, ExitCode: 0, Success: true},
			{Command: "go test", DurationMs: 5000, ExitCode: 0, Success: true},
		}
		for _, bash := range bashCmds {
			require.NoError(t, store.AppendBashCommand(ctx, sessionID, bash))
		}

		// 4. Append file operations
		fileOps := []FileOperationData{
			{OperationType: "read", FilePath: "/main.go", DurationMs: 10, BytesAffected: 500, Success: true},
			{OperationType: "write", FilePath: "/main.go", DurationMs: 20, BytesAffected: 600, Success: true},
		}
		for _, fileOp := range fileOps {
			require.NoError(t, store.AppendFileOperation(ctx, sessionID, fileOp))
		}

		// 5. Update aggregates
		require.NoError(t, store.UpdateSessionAggregates(ctx, sessionID, 2500, 120))

		// 6. Verify all data was recorded
		var toolCount, bashCount, fileCount int
		store.db.QueryRow("SELECT COUNT(*) FROM tool_executions WHERE session_id = ?", sessionID).Scan(&toolCount)
		store.db.QueryRow("SELECT COUNT(*) FROM bash_commands WHERE session_id = ?", sessionID).Scan(&bashCount)
		store.db.QueryRow("SELECT COUNT(*) FROM file_operations WHERE session_id = ?", sessionID).Scan(&fileCount)

		assert.Equal(t, 3, toolCount)
		assert.Equal(t, 2, bashCount)
		assert.Equal(t, 2, fileCount)

		// 7. Verify aggregates
		var tokens, duration int64
		store.db.QueryRow("SELECT total_tokens_used, total_duration_seconds FROM behavioral_sessions WHERE id = ?", sessionID).Scan(&tokens, &duration)
		assert.Equal(t, int64(2500), tokens)
		assert.Equal(t, int64(120), duration)

		// 8. Verify idempotent upsert returns same session
		sessionID2, isNew2, err := store.UpsertClaudeSession(ctx,
			"workflow-session-123",
			"/Users/test/project",
			"golang-pro",
		)
		require.NoError(t, err)
		assert.False(t, isNew2)
		assert.Equal(t, sessionID, sessionID2)
	})
}

// TestIncrementalSessionConcurrency tests concurrent access to incremental session APIs
func TestIncrementalSessionConcurrency(t *testing.T) {
	t.Run("handles concurrent appends to same session", func(t *testing.T) {
		ctx := context.Background()
		dbPath := filepath.Join(t.TempDir(), "concurrent_session.db")
		store, err := NewStore(dbPath)
		require.NoError(t, err)
		defer store.Close()

		sessionID, _, err := store.UpsertClaudeSession(ctx, "concurrent-session", "/proj", "agent")
		require.NoError(t, err)

		// Concurrent appends
		done := make(chan error, 30)

		// 10 tool executions
		for i := 0; i < 10; i++ {
			go func(idx int) {
				tool := ToolExecutionData{
					ToolName:   fmt.Sprintf("Tool%d", idx),
					DurationMs: int64(idx * 10),
					Success:    true,
				}
				done <- store.AppendToolExecution(ctx, sessionID, tool)
			}(i)
		}

		// 10 bash commands
		for i := 0; i < 10; i++ {
			go func(idx int) {
				bash := BashCommandData{
					Command:    fmt.Sprintf("cmd%d", idx),
					DurationMs: int64(idx * 5),
					ExitCode:   0,
					Success:    true,
				}
				done <- store.AppendBashCommand(ctx, sessionID, bash)
			}(i)
		}

		// 10 file operations
		for i := 0; i < 10; i++ {
			go func(idx int) {
				fileOp := FileOperationData{
					OperationType: "write",
					FilePath:      fmt.Sprintf("/file%d.go", idx),
					DurationMs:    int64(idx * 2),
					BytesAffected: int64(idx * 100),
					Success:       true,
				}
				done <- store.AppendFileOperation(ctx, sessionID, fileOp)
			}(i)
		}

		// Wait for all
		for i := 0; i < 30; i++ {
			err := <-done
			require.NoError(t, err)
		}

		// Verify all recorded
		var toolCount, bashCount, fileCount int
		store.db.QueryRow("SELECT COUNT(*) FROM tool_executions WHERE session_id = ?", sessionID).Scan(&toolCount)
		store.db.QueryRow("SELECT COUNT(*) FROM bash_commands WHERE session_id = ?", sessionID).Scan(&bashCount)
		store.db.QueryRow("SELECT COUNT(*) FROM file_operations WHERE session_id = ?", sessionID).Scan(&fileCount)

		assert.Equal(t, 10, toolCount)
		assert.Equal(t, 10, bashCount)
		assert.Equal(t, 10, fileCount)
	})
}

// TestIngestOffsetConcurrency tests concurrent access to offset APIs
func TestIngestOffsetConcurrency(t *testing.T) {
	t.Run("handles concurrent updates", func(t *testing.T) {
		ctx := context.Background()
		// Use file-based DB for better concurrent access support
		dbPath := filepath.Join(t.TempDir(), "offset_concurrent.db")
		store, err := NewStore(dbPath)
		require.NoError(t, err)
		defer store.Close()

		// Concurrently update different files
		done := make(chan error, 10)
		for i := 0; i < 10; i++ {
			go func(idx int) {
				offset := &IngestOffset{
					FilePath:   fmt.Sprintf("/path/file%d.jsonl", idx),
					ByteOffset: int64(idx * 100),
					Inode:      uint64(idx),
				}
				done <- store.SetFileOffset(ctx, offset)
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			err := <-done
			require.NoError(t, err)
		}

		// Verify all offsets exist
		offsets, err := store.ListFileOffsets(ctx)
		require.NoError(t, err)
		assert.Len(t, offsets, 10)
	})

	t.Run("handles concurrent updates to same file", func(t *testing.T) {
		ctx := context.Background()
		dbPath := filepath.Join(t.TempDir(), "offset_same_file.db")
		store, err := NewStore(dbPath)
		require.NoError(t, err)
		defer store.Close()

		// Concurrently update the same file
		done := make(chan error, 5)
		for i := 0; i < 5; i++ {
			go func(idx int) {
				offset := &IngestOffset{
					FilePath:   "/shared/file.jsonl",
					ByteOffset: int64(idx * 1000),
				}
				done <- store.SetFileOffset(ctx, offset)
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < 5; i++ {
			err := <-done
			require.NoError(t, err)
		}

		// Verify only one offset exists (upsert behavior)
		offsets, err := store.ListFileOffsets(ctx)
		require.NoError(t, err)
		assert.Len(t, offsets, 1)
	})
}

func TestGetRecentSessions(t *testing.T) {
	store, err := NewStore(":memory:")
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Insert test task execution
	exec := &TaskExecution{
		PlanFile:   "test-project/plan.md",
		TaskNumber: "1",
		TaskName:   "Test Task",
		Agent:      "test-agent",
		Success:    true,
	}
	err = store.RecordExecution(ctx, exec)
	require.NoError(t, err)

	// Record behavioral session using RecordSessionMetrics
	sessionData := &BehavioralSessionData{
		TaskExecutionID:     exec.ID,
		SessionStart:        exec.Timestamp,
		TotalDurationSecs:   120,
		TotalToolCalls:      5,
		TotalBashCommands:   3,
		TotalFileOperations: 2,
	}
	_, err = store.RecordSessionMetrics(ctx, sessionData, nil, nil, nil, nil)
	require.NoError(t, err)

	t.Run("returns recent sessions", func(t *testing.T) {
		sessions, err := store.GetRecentSessions(ctx, "", 10, 0)
		require.NoError(t, err)
		require.Len(t, sessions, 1)

		assert.Equal(t, "Test Task", sessions[0].TaskName)
		assert.Equal(t, "test-agent", sessions[0].Agent)
		assert.True(t, sessions[0].Success)
		assert.Equal(t, int64(120), sessions[0].DurationSecs)
	})

	t.Run("filters by project", func(t *testing.T) {
		sessions, err := store.GetRecentSessions(ctx, "test-project", 10, 0)
		require.NoError(t, err)
		require.Len(t, sessions, 1)

		sessions, err = store.GetRecentSessions(ctx, "nonexistent", 10, 0)
		require.NoError(t, err)
		assert.Len(t, sessions, 0)
	})

	t.Run("respects limit and offset", func(t *testing.T) {
		// Add more sessions
		for i := 2; i <= 5; i++ {
			exec := &TaskExecution{
				PlanFile:   "test-project/plan.md",
				TaskNumber: fmt.Sprintf("%d", i),
				TaskName:   fmt.Sprintf("Test Task %d", i),
				Agent:      "test-agent",
				Success:    true,
			}
			err = store.RecordExecution(ctx, exec)
			require.NoError(t, err)

			sessionData := &BehavioralSessionData{
				TaskExecutionID:   exec.ID,
				SessionStart:      exec.Timestamp,
				TotalDurationSecs: int64(60 * i),
			}
			_, err = store.RecordSessionMetrics(ctx, sessionData, nil, nil, nil, nil)
			require.NoError(t, err)
		}

		// Test limit
		sessions, err := store.GetRecentSessions(ctx, "", 2, 0)
		require.NoError(t, err)
		assert.Len(t, sessions, 2)

		// Test offset
		sessions, err = store.GetRecentSessions(ctx, "", 2, 2)
		require.NoError(t, err)
		assert.Len(t, sessions, 2)
	})
}

// TestUpdateCommitVerification tests the UpdateCommitVerification method.
func TestUpdateCommitVerification(t *testing.T) {
	ctx := context.Background()

	t.Run("updates commit verification for latest execution", func(t *testing.T) {
		store := setupTestStore(t)
		defer store.Close()

		// Record a task execution first
		exec := &TaskExecution{
			PlanFile:   "test-plan.yaml",
			TaskNumber: "1",
			TaskName:   "Test Task",
			Agent:      "golang-pro",
			Prompt:     "Do something",
			Success:    true,
			Output:     "Done",
			RunNumber:  1,
		}

		err := store.RecordExecution(ctx, exec)
		require.NoError(t, err)

		// Update commit verification
		err = store.UpdateCommitVerification(ctx, "test-plan.yaml", "1", 1, true, "abc1234")
		require.NoError(t, err)

		// Verify the update by querying directly
		var verified bool
		var hash string
		err = store.db.QueryRowContext(ctx,
			"SELECT commit_verified, commit_hash FROM task_executions WHERE plan_file = ? AND task_number = ?",
			"test-plan.yaml", "1").Scan(&verified, &hash)
		require.NoError(t, err)
		assert.True(t, verified)
		assert.Equal(t, "abc1234", hash)
	})

	t.Run("handles not found commit", func(t *testing.T) {
		store := setupTestStore(t)
		defer store.Close()

		// Record a task execution first
		exec := &TaskExecution{
			PlanFile:   "test-plan.yaml",
			TaskNumber: "2",
			TaskName:   "Test Task 2",
			Agent:      "golang-pro",
			Prompt:     "Do something",
			Success:    true,
			Output:     "Done",
			RunNumber:  1,
		}

		err := store.RecordExecution(ctx, exec)
		require.NoError(t, err)

		// Update commit verification with not found
		err = store.UpdateCommitVerification(ctx, "test-plan.yaml", "2", 1, false, "")
		require.NoError(t, err)

		// Verify the update
		var verified bool
		var hash sql.NullString
		err = store.db.QueryRowContext(ctx,
			"SELECT commit_verified, commit_hash FROM task_executions WHERE plan_file = ? AND task_number = ?",
			"test-plan.yaml", "2").Scan(&verified, &hash)
		require.NoError(t, err)
		assert.False(t, verified)
		assert.Equal(t, "", hash.String)
	})

	t.Run("updates only latest execution for same task", func(t *testing.T) {
		store := setupTestStore(t)
		defer store.Close()

		// Record multiple executions for the same task
		for i := 1; i <= 3; i++ {
			exec := &TaskExecution{
				PlanFile:   "test-plan.yaml",
				TaskNumber: "3",
				TaskName:   "Test Task 3",
				Agent:      "golang-pro",
				Prompt:     "Do something",
				Success:    i == 3, // Only last one succeeds
				Output:     fmt.Sprintf("Attempt %d", i),
				RunNumber:  1,
			}
			err := store.RecordExecution(ctx, exec)
			require.NoError(t, err)
		}

		// Update commit verification (should only update the latest)
		err := store.UpdateCommitVerification(ctx, "test-plan.yaml", "3", 1, true, "xyz7890")
		require.NoError(t, err)

		// Count how many have commit_verified set
		var count int
		err = store.db.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM task_executions WHERE plan_file = ? AND task_number = ? AND commit_verified = 1",
			"test-plan.yaml", "3").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count, "only latest execution should be updated")
	})
}
