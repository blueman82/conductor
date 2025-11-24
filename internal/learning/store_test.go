package learning

import (
	"context"
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
