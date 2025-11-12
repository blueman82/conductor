package learning

import (
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
