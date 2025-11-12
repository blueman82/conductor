package learning

import (
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
			assert.Equal(t, 1, version)

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
			tables := []string{"task_executions", "approach_history", "schema_version"}
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
		name           string
		wantVersion    int
		wantErr        bool
	}{
		{
			name:        "returns version 1 for new database",
			wantVersion: 1,
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
		name       string
		dbPath     string
		wantErr    bool
		checkFile  bool
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
		assert.Equal(t, 1, version2)
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
			assert.Equal(t, 1, version)
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
	tests := []struct {
		name    string
		exec    *TaskExecution
		wantErr bool
	}{
		{
			name: "records execution successfully",
			exec: &TaskExecution{
				TaskNumber:   "Task 1",
				TaskName:     "Setup database",
				Agent:        "golang-pro",
				Prompt:       "Initialize PostgreSQL database",
				Success:      true,
				Output:       "Database initialized successfully",
				DurationSecs: 30,
			},
			wantErr: false,
		},
		{
			name: "records failed execution",
			exec: &TaskExecution{
				TaskNumber:   "Task 2",
				TaskName:     "Run tests",
				Agent:        "test-automator",
				Prompt:       "Run all unit tests",
				Success:      false,
				ErrorMessage: "3 tests failed",
				Output:       "FAIL: TestFoo",
				DurationSecs: 45,
			},
			wantErr: false,
		},
		{
			name: "handles empty optional fields",
			exec: &TaskExecution{
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
				TaskNumber:   "Task 4",
				TaskName:     "Generate report",
				Prompt:       "Generate large report",
				Success:      true,
				Output:       string(make([]byte, 10000)), // 10KB output
				DurationSecs: 60,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := setupTestStore(t)
			defer store.Close()

			err := store.RecordExecution(tt.exec)
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
		})
	}
}

func TestRecordExecution_Concurrent(t *testing.T) {
	t.Run("handles concurrent writes without corruption", func(t *testing.T) {
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
					TaskNumber:   fmt.Sprintf("Task %d", idx),
					TaskName:     fmt.Sprintf("Concurrent task %d", idx),
					Prompt:       fmt.Sprintf("Test prompt %d", idx),
					Success:      idx%2 == 0,
					DurationSecs: int64(idx * 10),
				}
				done <- store.RecordExecution(exec)
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
	tests := []struct {
		name       string
		setup      func(*Store)
		taskNumber string
		wantCount  int
		wantErr    bool
	}{
		{
			name: "returns history for task",
			setup: func(s *Store) {
				executions := []*TaskExecution{
					{TaskNumber: "Task 1", TaskName: "Test A", Prompt: "prompt", Success: true},
					{TaskNumber: "Task 1", TaskName: "Test A", Prompt: "prompt", Success: false},
					{TaskNumber: "Task 2", TaskName: "Test B", Prompt: "prompt", Success: true},
				}
				for _, exec := range executions {
					require.NoError(t, s.RecordExecution(exec))
				}
			},
			taskNumber: "Task 1",
			wantCount:  2,
			wantErr:    false,
		},
		{
			name: "returns empty for nonexistent task",
			setup: func(s *Store) {
				exec := &TaskExecution{TaskNumber: "Task 1", TaskName: "Test", Prompt: "prompt", Success: true}
				require.NoError(t, s.RecordExecution(exec))
			},
			taskNumber: "Task 99",
			wantCount:  0,
			wantErr:    false,
		},
		{
			name:       "handles empty database",
			setup:      func(s *Store) {},
			taskNumber: "Task 1",
			wantCount:  0,
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

			history, err := store.GetExecutionHistory(tt.taskNumber)
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
		store := setupTestStore(t)
		defer store.Close()

		// Insert 3 executions with slight delays
		for i := 1; i <= 3; i++ {
			exec := &TaskExecution{
				TaskNumber:   "Task 1",
				TaskName:     fmt.Sprintf("Attempt %d", i),
				Prompt:       "test",
				Success:      i == 3,
				DurationSecs: int64(i),
			}
			require.NoError(t, store.RecordExecution(exec))
		}

		history, err := store.GetExecutionHistory("Task 1")
		require.NoError(t, err)
		require.Len(t, history, 3)

		// Verify chronological order (newest first)
		assert.Equal(t, int64(3), history[0].DurationSecs)
		assert.Equal(t, int64(2), history[1].DurationSecs)
		assert.Equal(t, int64(1), history[2].DurationSecs)
	})
}

func TestRecordApproach(t *testing.T) {
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

			err := store.RecordApproach(tt.approach)
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
		store := setupTestStore(t)
		defer store.Close()

		count := store.GetRunCount("never-executed.md")
		assert.Equal(t, 0, count)
	})
}

func TestGetRunCount_MultipleRuns(t *testing.T) {
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
				require.NoError(t, s.RecordExecution(exec))
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
					require.NoError(t, s.RecordExecution(exec))
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
					require.NoError(t, s.RecordExecution(exec))
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
					require.NoError(t, s.RecordExecution(exec))
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
				require.NoError(t, s.RecordExecution(other))

				// Create executions for target plan file
				exec := &TaskExecution{
					PlanFile:   pf,
					RunNumber:  2,
					TaskNumber: "Task 1",
					TaskName:   "Test",
					Prompt:     "test",
					Success:    true,
				}
				require.NoError(t, s.RecordExecution(exec))
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

			count := store.GetRunCount(tt.planFile)
			assert.Equal(t, tt.expected, count)
		})
	}
}
