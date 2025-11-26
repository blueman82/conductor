package learning

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyMigrations(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "applies all migrations successfully",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			store := setupTestStore(t)
			defer store.Close()

			err := store.ApplyMigrations(ctx)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Verify all migrations applied
			versions, err := store.GetAppliedVersions()
			require.NoError(t, err)
			assert.Len(t, versions, len(migrations))

			// Verify version order
			for i, v := range versions {
				assert.Equal(t, migrations[i].Version, v.Version)
			}
		})
	}
}

func TestApplyMigrations_Idempotency(t *testing.T) {
	t.Run("applying migrations multiple times is safe", func(t *testing.T) {
		ctx := context.Background()
		store := setupTestStore(t)
		defer store.Close()

		// Apply migrations first time
		err := store.ApplyMigrations(ctx)
		require.NoError(t, err)

		versionsFirst, err := store.GetAppliedVersions()
		require.NoError(t, err)

		// Apply migrations second time
		err = store.ApplyMigrations(ctx)
		require.NoError(t, err)

		versionsSecond, err := store.GetAppliedVersions()
		require.NoError(t, err)

		// Should have same number of versions
		assert.Equal(t, len(versionsFirst), len(versionsSecond))
	})
}

func TestApplyMigrations_IncrementalApplication(t *testing.T) {
	t.Run("applies only pending migrations", func(t *testing.T) {
		ctx := context.Background()
		dbPath := filepath.Join(t.TempDir(), "incremental.db")

		// Create store and apply first migration
		store1, err := NewStore(dbPath)
		require.NoError(t, err)

		// Manually apply only version 1
		_, err = store1.db.Exec(migrations[0].SQL)
		require.NoError(t, err)
		err = store1.recordMigration(ctx, 1)
		require.NoError(t, err)
		store1.Close()

		// Reopen and apply remaining migrations
		store2, err := NewStore(dbPath)
		require.NoError(t, err)
		defer store2.Close()

		// Version 1 should already be applied
		applied, err := store2.IsMigrationApplied(1)
		require.NoError(t, err)
		assert.True(t, applied)

		// All migrations should now be applied
		versions, err := store2.GetAppliedVersions()
		require.NoError(t, err)
		assert.Len(t, versions, len(migrations))
	})
}

func TestGetAppliedVersions(t *testing.T) {
	t.Run("returns empty for fresh database with only schema_version table", func(t *testing.T) {
		// Create a raw store without migrations
		dbPath := filepath.Join(t.TempDir(), "fresh.db")
		db, err := sql.Open("sqlite3", dbPath)
		require.NoError(t, err)
		defer db.Close()

		store := &Store{db: db, dbPath: dbPath}
		err = store.ensureSchemaVersionTable()
		require.NoError(t, err)

		versions, err := store.GetAppliedVersions()
		require.NoError(t, err)
		assert.Len(t, versions, 0)
	})

	t.Run("returns applied versions after migrations", func(t *testing.T) {
		store := setupTestStore(t)
		defer store.Close()

		versions, err := store.GetAppliedVersions()
		require.NoError(t, err)
		assert.Len(t, versions, len(migrations))
	})
}

func TestIsMigrationApplied(t *testing.T) {
	t.Run("returns false for unapplied migration", func(t *testing.T) {
		// Create a raw store without migrations
		dbPath := filepath.Join(t.TempDir(), "unapplied.db")
		db, err := sql.Open("sqlite3", dbPath)
		require.NoError(t, err)
		defer db.Close()

		store := &Store{db: db, dbPath: dbPath}
		err = store.ensureSchemaVersionTable()
		require.NoError(t, err)

		applied, err := store.IsMigrationApplied(1)
		require.NoError(t, err)
		assert.False(t, applied)
	})

	t.Run("returns true for applied migration", func(t *testing.T) {
		store := setupTestStore(t)
		defer store.Close()

		applied, err := store.IsMigrationApplied(1)
		require.NoError(t, err)
		assert.True(t, applied)
	})

	t.Run("returns false for nonexistent migration version", func(t *testing.T) {
		store := setupTestStore(t)
		defer store.Close()

		applied, err := store.IsMigrationApplied(999)
		require.NoError(t, err)
		assert.False(t, applied)
	})
}

func TestGetLatestVersion(t *testing.T) {
	t.Run("returns 0 for fresh database with only schema_version table", func(t *testing.T) {
		// Create a raw store without migrations
		dbPath := filepath.Join(t.TempDir(), "fresh_version.db")
		db, err := sql.Open("sqlite3", dbPath)
		require.NoError(t, err)
		defer db.Close()

		store := &Store{db: db, dbPath: dbPath}
		err = store.ensureSchemaVersionTable()
		require.NoError(t, err)

		version, err := store.GetLatestVersion()
		require.NoError(t, err)
		assert.Equal(t, 0, version)
	})

	t.Run("returns latest version after migrations", func(t *testing.T) {
		store := setupTestStore(t)
		defer store.Close()

		version, err := store.GetLatestVersion()
		require.NoError(t, err)
		assert.Equal(t, len(migrations), version)
	})
}

func TestEnsureSchemaVersionTable(t *testing.T) {
	t.Run("creates schema_version table", func(t *testing.T) {
		store := setupTestStore(t)
		defer store.Close()

		err := store.ensureSchemaVersionTable()
		require.NoError(t, err)

		// Verify table exists
		exists, err := store.tableExists("schema_version")
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("is idempotent", func(t *testing.T) {
		store := setupTestStore(t)
		defer store.Close()

		// Call multiple times
		err := store.ensureSchemaVersionTable()
		require.NoError(t, err)

		err = store.ensureSchemaVersionTable()
		require.NoError(t, err)

		err = store.ensureSchemaVersionTable()
		require.NoError(t, err)

		// Verify table still exists
		exists, err := store.tableExists("schema_version")
		require.NoError(t, err)
		assert.True(t, exists)
	})
}

func TestRecordMigration(t *testing.T) {
	tests := []struct {
		name    string
		version int
		wantErr bool
	}{
		{
			name:    "records migration version",
			version: 1,
			wantErr: false,
		},
		{
			name:    "handles duplicate version gracefully",
			version: 1,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			store := setupTestStore(t)
			defer store.Close()

			err := store.ensureSchemaVersionTable()
			require.NoError(t, err)

			// Record migration
			err = store.recordMigration(ctx, tt.version)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Verify recorded
			applied, err := store.IsMigrationApplied(tt.version)
			require.NoError(t, err)
			assert.True(t, applied)

			// Record again (should be idempotent)
			err = store.recordMigration(ctx, tt.version)
			require.NoError(t, err)
		})
	}
}

func TestMigrations_TableCreation(t *testing.T) {
	t.Run("migration 1 creates base tables", func(t *testing.T) {
		ctx := context.Background()
		store := setupTestStore(t)
		defer store.Close()

		// Apply only migration 1
		err := store.ensureSchemaVersionTable()
		require.NoError(t, err)

		_, err = store.db.ExecContext(ctx, migrations[0].SQL)
		require.NoError(t, err)

		err = store.recordMigration(ctx, 1)
		require.NoError(t, err)

		// Verify base tables exist
		tables := []string{"task_executions", "approach_history"}
		for _, table := range tables {
			exists, err := store.tableExists(table)
			require.NoError(t, err, "table %s should exist", table)
			assert.True(t, exists, "table %s should exist", table)
		}
	})

	t.Run("migration 2 creates behavioral tables", func(t *testing.T) {
		ctx := context.Background()
		store := setupTestStore(t)
		defer store.Close()

		// Apply all migrations
		err := store.ApplyMigrations(ctx)
		require.NoError(t, err)

		// Verify behavioral tables exist
		behavioralTables := []string{
			"behavioral_sessions",
			"tool_executions",
			"bash_commands",
			"file_operations",
			"token_usage",
		}
		for _, table := range behavioralTables {
			exists, err := store.tableExists(table)
			require.NoError(t, err, "table %s should exist", table)
			assert.True(t, exists, "table %s should exist", table)
		}
	})

	t.Run("migration 3 creates ingest_offsets table", func(t *testing.T) {
		ctx := context.Background()
		store := setupTestStore(t)
		defer store.Close()

		// Apply all migrations
		err := store.ApplyMigrations(ctx)
		require.NoError(t, err)

		// Verify ingest_offsets table exists
		exists, err := store.tableExists("ingest_offsets")
		require.NoError(t, err)
		assert.True(t, exists, "ingest_offsets table should exist")

		// Verify index exists
		indexExists, err := store.indexExists("idx_ingest_offsets_path")
		require.NoError(t, err)
		assert.True(t, indexExists, "idx_ingest_offsets_path index should exist")

		// Verify table structure by inserting test data
		_, err = store.db.ExecContext(ctx, `
			INSERT INTO ingest_offsets (file_path, byte_offset, inode, last_line_hash)
			VALUES (?, ?, ?, ?)
		`, "/test/file.jsonl", 1024, 12345, "abc123")
		require.NoError(t, err)

		// Verify unique constraint on file_path
		_, err = store.db.ExecContext(ctx, `
			INSERT INTO ingest_offsets (file_path, byte_offset)
			VALUES (?, ?)
		`, "/test/file.jsonl", 2048)
		require.Error(t, err, "should fail on duplicate file_path")
	})
}

func TestMigrations_IndexCreation(t *testing.T) {
	t.Run("creates all required indexes", func(t *testing.T) {
		ctx := context.Background()
		store := setupTestStore(t)
		defer store.Close()

		err := store.ApplyMigrations(ctx)
		require.NoError(t, err)

		// Verify indexes for behavioral tables
		indexes := []string{
			"idx_behavioral_sessions_task_id",
			"idx_behavioral_sessions_start",
			"idx_tool_executions_session_id",
			"idx_tool_executions_tool_name",
			"idx_tool_executions_success",
			"idx_bash_commands_session_id",
			"idx_bash_commands_success",
			"idx_bash_commands_exit_code",
			"idx_file_operations_session_id",
			"idx_file_operations_type",
			"idx_file_operations_path",
			"idx_token_usage_session_id",
			"idx_token_usage_time",
			"idx_ingest_offsets_path",
		}

		for _, index := range indexes {
			exists, err := store.indexExists(index)
			require.NoError(t, err, "index %s check failed", index)
			assert.True(t, exists, "index %s should exist", index)
		}
	})
}

func TestNewStore_AppliesMigrations(t *testing.T) {
	t.Run("new store automatically applies all migrations", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "auto_migrate.db")
		store, err := NewStore(dbPath)
		require.NoError(t, err)
		defer store.Close()

		// Verify all migrations applied
		version, err := store.GetLatestVersion()
		require.NoError(t, err)
		assert.Equal(t, len(migrations), version)

		// Verify behavioral tables exist
		exists, err := store.tableExists("behavioral_sessions")
		require.NoError(t, err)
		assert.True(t, exists)
	})
}

func TestMigrations_FreshVsExisting(t *testing.T) {
	t.Run("fresh database gets all migrations", func(t *testing.T) {
		store := setupTestStore(t)
		defer store.Close()

		version, err := store.GetLatestVersion()
		require.NoError(t, err)
		assert.Equal(t, len(migrations), version)
	})

	t.Run("existing database gets incremental migrations", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "existing.db")

		// Create first store with only migration 1
		store1, err := NewStore(dbPath)
		require.NoError(t, err)

		// Manually reset to version 1
		ctx := context.Background()
		_, err = store1.db.ExecContext(ctx, "DELETE FROM schema_version WHERE version > 1")
		require.NoError(t, err)

		// Verify only version 1
		version, err := store1.GetLatestVersion()
		require.NoError(t, err)
		assert.Equal(t, 1, version)

		store1.Close()

		// Reopen and verify migration 2 gets applied
		store2, err := NewStore(dbPath)
		require.NoError(t, err)
		defer store2.Close()

		version, err = store2.GetLatestVersion()
		require.NoError(t, err)
		assert.Equal(t, len(migrations), version)

		// Verify behavioral tables now exist
		exists, err := store2.tableExists("behavioral_sessions")
		require.NoError(t, err)
		assert.True(t, exists)
	})
}

func TestWALMode_ConcurrentAccess(t *testing.T) {
	t.Run("concurrent reads and writes succeed with WAL mode", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "wal_concurrent.db")
		store, err := NewStore(dbPath)
		require.NoError(t, err)
		defer store.Close()

		// Verify WAL mode is enabled
		var journalMode string
		err = store.db.QueryRow("PRAGMA journal_mode").Scan(&journalMode)
		require.NoError(t, err)
		assert.Equal(t, "wal", journalMode)

		// Verify busy_timeout is set
		var busyTimeout int
		err = store.db.QueryRow("PRAGMA busy_timeout").Scan(&busyTimeout)
		require.NoError(t, err)
		assert.Equal(t, 5000, busyTimeout)

		ctx := context.Background()
		const iterations = 100
		errCh := make(chan error, iterations*2)

		// Writer goroutine
		go func() {
			for i := 0; i < iterations; i++ {
				_, err := store.db.ExecContext(ctx, `
					INSERT INTO task_executions (task_number, task_name, prompt, success)
					VALUES (?, ?, ?, ?)
				`, fmt.Sprintf("task_%d", i), "concurrent_test", "test prompt", true)
				if err != nil {
					errCh <- fmt.Errorf("write %d: %w", i, err)
					return
				}
			}
			errCh <- nil
		}()

		// Reader goroutine
		go func() {
			for i := 0; i < iterations; i++ {
				var count int
				err := store.db.QueryRowContext(ctx, `
					SELECT COUNT(*) FROM task_executions WHERE task_name = ?
				`, "concurrent_test").Scan(&count)
				if err != nil {
					errCh <- fmt.Errorf("read %d: %w", i, err)
					return
				}
			}
			errCh <- nil
		}()

		// Wait for both goroutines
		for i := 0; i < 2; i++ {
			err := <-errCh
			require.NoError(t, err)
		}
	})
}
