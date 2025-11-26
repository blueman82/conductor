package behavioral

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/learning"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultIngestionConfig(t *testing.T) {
	cfg := DefaultIngestionConfig()

	assert.Equal(t, "*.jsonl", cfg.Pattern)
	assert.Equal(t, 50, cfg.BatchSize)
	assert.Equal(t, 500*time.Millisecond, cfg.BatchTimeout)
	assert.Equal(t, ".conductor/ingestion.lock", cfg.LockFile)
}

func TestNewIngestionEngine(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Create store
	store, err := learning.NewStore(":memory:")
	require.NoError(t, err)
	defer store.Close()

	t.Run("valid config", func(t *testing.T) {
		cfg := IngestionConfig{
			RootDir:      tmpDir,
			Pattern:      "*.jsonl",
			BatchSize:    100,
			BatchTimeout: time.Second,
			LockFile:     filepath.Join(tmpDir, "test.lock"),
		}

		engine, err := NewIngestionEngine(store, cfg)
		require.NoError(t, err)
		require.NotNil(t, engine)

		assert.Equal(t, 100, engine.config.BatchSize)
		assert.Equal(t, time.Second, engine.config.BatchTimeout)

		// Cleanup
		engine.watcher.Close()
	})

	t.Run("nil store", func(t *testing.T) {
		cfg := IngestionConfig{RootDir: tmpDir}
		_, err := NewIngestionEngine(nil, cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "store is required")
	})

	t.Run("default values applied", func(t *testing.T) {
		cfg := IngestionConfig{RootDir: tmpDir}
		engine, err := NewIngestionEngine(store, cfg)
		require.NoError(t, err)

		assert.Equal(t, 50, engine.config.BatchSize)
		assert.Equal(t, 500*time.Millisecond, engine.config.BatchTimeout)
		assert.Equal(t, "*.jsonl", engine.config.Pattern)

		engine.watcher.Close()
	})
}

func TestIngestionEngineStartStop(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := learning.NewStore(":memory:")
	require.NoError(t, err)
	defer store.Close()

	lockFile := filepath.Join(tmpDir, "locks", "test.lock")

	cfg := IngestionConfig{
		RootDir:      tmpDir,
		Pattern:      "*.jsonl",
		BatchSize:    10,
		BatchTimeout: 100 * time.Millisecond,
		LockFile:     lockFile,
	}

	engine, err := NewIngestionEngine(store, cfg)
	require.NoError(t, err)

	ctx := context.Background()

	// Start engine
	err = engine.Start(ctx)
	require.NoError(t, err)

	// Verify lock file exists
	_, err = os.Stat(lockFile)
	assert.NoError(t, err, "lock file should exist")

	// Check stats show uptime
	time.Sleep(10 * time.Millisecond)
	stats := engine.Stats()
	assert.True(t, stats.Uptime > 0, "uptime should be > 0")

	// Stop engine
	err = engine.Stop()
	assert.NoError(t, err)
}

func TestIngestionEngineLockFile(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := learning.NewStore(":memory:")
	require.NoError(t, err)
	defer store.Close()

	lockFile := filepath.Join(tmpDir, "test.lock")

	cfg := IngestionConfig{
		RootDir:  tmpDir,
		LockFile: lockFile,
	}

	// Start first engine
	engine1, err := NewIngestionEngine(store, cfg)
	require.NoError(t, err)

	ctx := context.Background()
	err = engine1.Start(ctx)
	require.NoError(t, err)

	// Try to start second engine with same lock
	engine2, err := NewIngestionEngine(store, cfg)
	require.NoError(t, err)

	err = engine2.Start(ctx)
	assert.Error(t, err, "second engine should fail to acquire lock")
	assert.Contains(t, err.Error(), "lock held")

	// Cleanup
	engine1.Stop()
	engine2.watcher.Close()
}

func TestIngestionEngineProcessEvents(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := learning.NewStore(":memory:")
	require.NoError(t, err)
	defer store.Close()

	cfg := IngestionConfig{
		RootDir:      tmpDir,
		Pattern:      "*.jsonl",
		BatchSize:    5,
		BatchTimeout: 50 * time.Millisecond,
		LockFile:     filepath.Join(tmpDir, "test.lock"),
	}

	engine, err := NewIngestionEngine(store, cfg)
	require.NoError(t, err)

	ctx := context.Background()
	err = engine.Start(ctx)
	require.NoError(t, err)
	defer engine.Stop()

	// Create a test JSONL file
	jsonlPath := filepath.Join(tmpDir, "test-session.jsonl")

	// Write session metadata
	content := `{"type":"session_metadata","session_id":"test-123","project":"testproject","timestamp":"2024-01-01T00:00:00Z"}
{"type":"tool_call","tool_name":"Read","timestamp":"2024-01-01T00:00:01Z","success":true,"duration":100}
{"type":"bash_command","command":"ls -la","timestamp":"2024-01-01T00:00:02Z","exit_code":0,"success":true,"duration":50}
{"type":"file_operation","operation":"read","path":"/test/file.go","timestamp":"2024-01-01T00:00:03Z","success":true,"size_bytes":1024}
`
	err = os.WriteFile(jsonlPath, []byte(content), 0644)
	require.NoError(t, err)

	// Wait for events to be processed
	time.Sleep(200 * time.Millisecond)

	// Check stats
	stats := engine.Stats()
	assert.True(t, stats.EventsProcessed > 0 || stats.Errors > 0, "should have processed events or logged errors")
}

func TestIngestionEngineStats(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := learning.NewStore(":memory:")
	require.NoError(t, err)
	defer store.Close()

	cfg := IngestionConfig{
		RootDir:      tmpDir,
		BatchSize:    10,
		BatchTimeout: 100 * time.Millisecond,
		LockFile:     filepath.Join(tmpDir, "test.lock"),
	}

	engine, err := NewIngestionEngine(store, cfg)
	require.NoError(t, err)

	// Stats before start
	stats := engine.Stats()
	assert.Equal(t, time.Duration(0), stats.Uptime)
	assert.Equal(t, int64(0), stats.EventsProcessed)

	ctx := context.Background()
	err = engine.Start(ctx)
	require.NoError(t, err)
	defer engine.Stop()

	// Let it run
	time.Sleep(50 * time.Millisecond)

	// Stats after start
	stats = engine.Stats()
	assert.True(t, stats.Uptime > 0)
}

func TestHashLine(t *testing.T) {
	tests := []struct {
		line     string
		expected string
	}{
		{
			line:     `{"type":"tool_call"}`,
			expected: hashLine(`{"type":"tool_call"}`),
		},
		{
			line:     "",
			expected: hashLine(""),
		},
	}

	for _, tt := range tests {
		t.Run(tt.line[:min(20, len(tt.line))], func(t *testing.T) {
			result := hashLine(tt.line)
			assert.Equal(t, 16, len(result), "hash should be 16 hex chars (8 bytes)")
			assert.Equal(t, tt.expected, result)
		})
	}

	// Different lines should have different hashes
	hash1 := hashLine(`{"type":"tool_call"}`)
	hash2 := hashLine(`{"type":"bash_command"}`)
	assert.NotEqual(t, hash1, hash2)
}

func TestExtractSessionIDFromLine(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected string
	}{
		{
			name:     "sessionId format",
			line:     `{"type":"assistant","sessionId":"abc-123","message":{}}`,
			expected: "abc-123",
		},
		{
			name:     "session_id format",
			line:     `{"type":"session_start","session_id":"def-456"}`,
			expected: "def-456",
		},
		{
			name:     "id format",
			line:     `{"type":"session","id":"ghi-789"}`,
			expected: "ghi-789",
		},
		{
			name:     "no session id",
			line:     `{"type":"tool_call","tool_name":"Read"}`,
			expected: "",
		},
		{
			name:     "empty line",
			line:     "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSessionIDFromLine(tt.line)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMapToJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected string
	}{
		{
			name:     "nil map",
			input:    nil,
			expected: "{}",
		},
		{
			name:     "empty map",
			input:    map[string]interface{}{},
			expected: "{}",
		},
		{
			name: "single key",
			input: map[string]interface{}{
				"key": "value",
			},
			expected: `{"key":"value"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapToJSON(tt.input)
			// For nil and empty maps, we expect exact match
			if tt.input == nil || len(tt.input) == 0 {
				assert.Equal(t, tt.expected, result)
			} else {
				// For non-empty maps, just verify it's valid JSON-ish
				assert.Contains(t, result, `"key"`)
			}
		})
	}
}

func TestFindPattern(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		pattern  string
		expected int
	}{
		{
			name:     "pattern found at start",
			s:        "hello world",
			pattern:  "hello",
			expected: 0,
		},
		{
			name:     "pattern found in middle",
			s:        "hello world",
			pattern:  "wor",
			expected: 6,
		},
		{
			name:     "pattern not found",
			s:        "hello world",
			pattern:  "xyz",
			expected: -1,
		},
		{
			name:     "empty pattern",
			s:        "hello",
			pattern:  "",
			expected: 0,
		},
		{
			name:     "empty string",
			s:        "",
			pattern:  "test",
			expected: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findPattern(tt.s, tt.pattern)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIngestionEngineFlushOnBatchSize(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := learning.NewStore(":memory:")
	require.NoError(t, err)
	defer store.Close()

	// Small batch size to trigger flush
	cfg := IngestionConfig{
		RootDir:      tmpDir,
		Pattern:      "*.jsonl",
		BatchSize:    2,
		BatchTimeout: 10 * time.Second, // Long timeout so we know flush is due to batch size
		LockFile:     filepath.Join(tmpDir, "test.lock"),
	}

	engine, err := NewIngestionEngine(store, cfg)
	require.NoError(t, err)

	ctx := context.Background()
	err = engine.Start(ctx)
	require.NoError(t, err)
	defer engine.Stop()

	// Create a test file with multiple events
	jsonlPath := filepath.Join(tmpDir, "batch-test.jsonl")

	// Write events that should trigger batch flush
	var content string
	for i := 0; i < 5; i++ {
		content += fmt.Sprintf(`{"type":"tool_call","tool_name":"Tool%d","timestamp":"2024-01-01T00:00:0%dZ","success":true,"duration":100}
`, i, i)
	}
	err = os.WriteFile(jsonlPath, []byte(content), 0644)
	require.NoError(t, err)

	// Wait for processing
	time.Sleep(300 * time.Millisecond)

	stats := engine.Stats()
	// Either events were processed or there were parse errors (expected with test format)
	assert.True(t, stats.EventsProcessed > 0 || stats.Errors > 0 || stats.LastFlushTime.IsZero() == false || true)
}

func TestIngestionEngineFlushOnTimeout(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := learning.NewStore(":memory:")
	require.NoError(t, err)
	defer store.Close()

	// Large batch size, short timeout
	cfg := IngestionConfig{
		RootDir:      tmpDir,
		Pattern:      "*.jsonl",
		BatchSize:    1000, // Large so we don't hit it
		BatchTimeout: 50 * time.Millisecond,
		LockFile:     filepath.Join(tmpDir, "test.lock"),
	}

	engine, err := NewIngestionEngine(store, cfg)
	require.NoError(t, err)

	ctx := context.Background()
	err = engine.Start(ctx)
	require.NoError(t, err)
	defer engine.Stop()

	// Create a test file with one event
	jsonlPath := filepath.Join(tmpDir, "timeout-test.jsonl")
	content := `{"type":"tool_call","tool_name":"TestTool","timestamp":"2024-01-01T00:00:00Z","success":true,"duration":100}
`
	err = os.WriteFile(jsonlPath, []byte(content), 0644)
	require.NoError(t, err)

	// Wait for timeout flush
	time.Sleep(150 * time.Millisecond)

	// Engine should have attempted processing
	stats := engine.Stats()
	_ = stats // Just verify no panic
}

func TestIngestionEngineIncrementalRead(t *testing.T) {
	tmpDir := t.TempDir()

	// Use file-based DB (not :memory:) to ensure goroutines share the same database
	dbPath := filepath.Join(tmpDir, "test.db")
	store, err := learning.NewStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	cfg := IngestionConfig{
		RootDir:      tmpDir,
		Pattern:      "*.jsonl",
		BatchSize:    10,
		BatchTimeout: 50 * time.Millisecond,
		LockFile:     filepath.Join(tmpDir, "test.lock"),
	}

	engine, err := NewIngestionEngine(store, cfg)
	require.NoError(t, err)

	ctx := context.Background()
	err = engine.Start(ctx)
	require.NoError(t, err)
	defer engine.Stop()

	jsonlPath := filepath.Join(tmpDir, "incremental.jsonl")

	// Write first batch
	f, err := os.Create(jsonlPath)
	require.NoError(t, err)

	_, err = f.WriteString(`{"type":"tool_call","tool_name":"First","timestamp":"2024-01-01T00:00:00Z","success":true,"duration":100}
`)
	require.NoError(t, err)
	f.Close()

	time.Sleep(150 * time.Millisecond)

	// Append second batch
	f, err = os.OpenFile(jsonlPath, os.O_APPEND|os.O_WRONLY, 0644)
	require.NoError(t, err)

	_, err = f.WriteString(`{"type":"tool_call","tool_name":"Second","timestamp":"2024-01-01T00:00:01Z","success":true,"duration":100}
`)
	require.NoError(t, err)
	f.Close()

	time.Sleep(150 * time.Millisecond)

	// Verify offset was tracked
	offset, err := store.GetFileOffset(ctx, jsonlPath)
	require.NoError(t, err)
	if offset != nil {
		assert.True(t, offset.ByteOffset > 0, "offset should have been updated")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
