package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/behavioral"
)

func TestNewObserveLiveCmd(t *testing.T) {
	cmd := NewObserveLiveCmd()

	if cmd.Use != "live" {
		t.Errorf("expected Use='live', got %q", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("expected non-empty Short description")
	}

	// Check --raw flag exists
	flag := cmd.Flags().Lookup("raw")
	if flag == nil {
		t.Error("expected --raw flag to exist")
	}
}

func TestRunLiveWatch_ContextCancellation(t *testing.T) {
	// Create temp directory with test JSONL
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "testproject")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	// Write initial JSONL file
	jsonlPath := filepath.Join(projectDir, "test-session.jsonl")
	initialContent := `{"type":"assistant","timestamp":"2024-01-15T10:00:00Z","sessionId":"test123","message":{"role":"assistant","content":[{"type":"text","text":"Hello"}]}}
`
	if err := os.WriteFile(jsonlPath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("failed to write JSONL: %v", err)
	}

	// Create context that cancels after short delay
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Run with the temp directory as root - tests context cancellation
	err := runLiveWatchWithRoot(ctx, tmpDir, "testproject", 50*time.Millisecond, true)
	if err != nil && err != context.DeadlineExceeded {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunLiveWatch_EventFormatting(t *testing.T) {
	// Create temp directory with test JSONL
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "myproject")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	// Write JSONL with multiple event types
	jsonlPath := filepath.Join(projectDir, "session.jsonl")
	content := `{"type":"assistant","timestamp":"2024-01-15T10:00:00Z","sessionId":"sess1","message":{"role":"assistant","content":[{"type":"text","text":"Starting task"}]}}
{"type":"assistant","timestamp":"2024-01-15T10:00:01Z","sessionId":"sess1","message":{"role":"assistant","content":[{"type":"tool_use","id":"t1","name":"Bash","input":{"command":"ls"}}]}}
`
	if err := os.WriteFile(jsonlPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write JSONL: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	// Run in raw mode (no colors)
	err := runLiveWatchWithRoot(ctx, tmpDir, "myproject", 50*time.Millisecond, true)
	if err != nil && err != context.DeadlineExceeded {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunLiveWatch_AllProjects(t *testing.T) {
	// Create temp directory with multiple projects
	tmpDir := t.TempDir()

	project1 := filepath.Join(tmpDir, "project1")
	project2 := filepath.Join(tmpDir, "project2")
	if err := os.MkdirAll(project1, 0755); err != nil {
		t.Fatalf("failed to create project1: %v", err)
	}
	if err := os.MkdirAll(project2, 0755); err != nil {
		t.Fatalf("failed to create project2: %v", err)
	}

	// Write JSONL in both projects
	content := `{"type":"assistant","timestamp":"2024-01-15T10:00:00Z","sessionId":"test","message":{"role":"assistant","content":[{"type":"text","text":"Hello"}]}}
`
	if err := os.WriteFile(filepath.Join(project1, "s1.jsonl"), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write JSONL: %v", err)
	}
	if err := os.WriteFile(filepath.Join(project2, "s2.jsonl"), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write JSONL: %v", err)
	}

	// Run without project filter (all projects)
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	err := runLiveWatchWithRoot(ctx, tmpDir, "", 50*time.Millisecond, true)
	if err != nil && err != context.DeadlineExceeded {
		t.Errorf("unexpected error: %v", err)
	}
}

// runLiveWatchWithRoot is a test helper that uses a custom root directory
func runLiveWatchWithRoot(ctx context.Context, rootDir, project string, pollInterval time.Duration, raw bool) error {
	watcher := behavioral.NewLiveWatcher(rootDir, project)
	watcher.SetPollInterval(pollInterval)

	opts := behavioral.DefaultTranscriptOptions()
	if raw {
		opts.ColorOutput = false
	}

	errChan := make(chan error, 1)
	go func() {
		errChan <- watcher.Start(ctx)
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-errChan:
			if err != nil && err != ctx.Err() {
				return err
			}
			return nil
		case event, ok := <-watcher.Events():
			if !ok {
				return nil
			}
			// Format and discard (testing event processing)
			_ = behavioral.FormatTranscriptEntry(event, opts)
		}
	}
}
