package cmd

import (
	"bytes"
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

// Silence unused import warnings for tests that don't use all imports
var (
	_ = bytes.Buffer{}
)

func TestNewObserveIngestCmd(t *testing.T) {
	cmd := NewObserveIngestCmd()

	assert.Equal(t, "ingest", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotNil(t, cmd.RunE)

	// Check flags exist
	flags := []string{"watch", "batch-size", "batch-timeout", "root-dir", "verbose"}
	for _, flag := range flags {
		f := cmd.Flags().Lookup(flag)
		assert.NotNil(t, f, "flag %s should exist", flag)
	}
}

func TestIngestCmdFlags(t *testing.T) {
	cmd := NewObserveIngestCmd()

	// Check default values
	watchFlag := cmd.Flags().Lookup("watch")
	assert.Equal(t, "false", watchFlag.DefValue)

	batchSizeFlag := cmd.Flags().Lookup("batch-size")
	assert.Equal(t, "50", batchSizeFlag.DefValue)

	batchTimeoutFlag := cmd.Flags().Lookup("batch-timeout")
	assert.Equal(t, "500ms", batchTimeoutFlag.DefValue)

	rootDirFlag := cmd.Flags().Lookup("root-dir")
	assert.Equal(t, "", rootDirFlag.DefValue)

	verboseFlag := cmd.Flags().Lookup("verbose")
	assert.Equal(t, "false", verboseFlag.DefValue)
}

func TestIngestCmdHelp(t *testing.T) {
	cmd := NewObserveIngestCmd()

	// Capture help output
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "ingest")
	assert.Contains(t, output, "--watch")
	assert.Contains(t, output, "--batch-size")
	assert.Contains(t, output, "--batch-timeout")
	assert.Contains(t, output, "--root-dir")
	assert.Contains(t, output, "--verbose")
}

func TestIngestCmdNilStore(t *testing.T) {
	// Test with nil store
	cfg := behavioral.IngestionConfig{
		RootDir: t.TempDir(),
	}

	_, err := behavioral.NewIngestionEngine(nil, cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "store is required")
}

func TestIngestCmdWithEmptyDir(t *testing.T) {
	// Create temp directories
	tmpDir := t.TempDir()
	rootDir := filepath.Join(tmpDir, "projects")
	dbPath := filepath.Join(tmpDir, "test.db")

	require.NoError(t, os.MkdirAll(rootDir, 0755))

	// Create store directly
	store, err := learning.NewStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	cfg := behavioral.IngestionConfig{
		RootDir:      rootDir,
		BatchSize:    50,
		BatchTimeout: 100 * time.Millisecond,
		LockFile:     filepath.Join(tmpDir, "test.lock"),
	}

	engine, err := behavioral.NewIngestionEngine(store, cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Start should succeed with empty directory
	err = engine.Start(ctx)
	require.NoError(t, err)

	// Let it run briefly
	time.Sleep(50 * time.Millisecond)

	// Stop should complete
	err = engine.Stop()
	assert.NoError(t, err)

	// Stats should show zero events
	stats := engine.Stats()
	assert.Equal(t, int64(0), stats.EventsProcessed)
}

func TestIngestEngineStats(t *testing.T) {
	// Create temp directories
	tmpDir := t.TempDir()
	rootDir := filepath.Join(tmpDir, "projects")
	dbPath := filepath.Join(tmpDir, "test.db")

	require.NoError(t, os.MkdirAll(rootDir, 0755))

	// Create store
	store, err := learning.NewStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	cfg := behavioral.IngestionConfig{
		RootDir:      rootDir,
		BatchSize:    100,
		BatchTimeout: 1 * time.Second,
		LockFile:     filepath.Join(tmpDir, "test.lock"),
	}

	engine, err := behavioral.NewIngestionEngine(store, cfg)
	require.NoError(t, err)

	// Stats before start should have zero uptime
	stats := engine.Stats()
	assert.Equal(t, time.Duration(0), stats.Uptime)

	ctx := context.Background()
	err = engine.Start(ctx)
	require.NoError(t, err)

	// Let it run
	time.Sleep(50 * time.Millisecond)

	// Stats should show uptime
	stats = engine.Stats()
	assert.True(t, stats.Uptime > 0)

	err = engine.Stop()
	assert.NoError(t, err)
}
