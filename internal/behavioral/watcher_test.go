package behavioral

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewLiveWatcher(t *testing.T) {
	w := NewLiveWatcher("/test/root", "myproject")

	if w.rootDir != "/test/root" {
		t.Errorf("rootDir = %q, want %q", w.rootDir, "/test/root")
	}
	if w.project != "myproject" {
		t.Errorf("project = %q, want %q", w.project, "myproject")
	}
	if w.positions == nil {
		t.Error("positions map not initialized")
	}
	if w.seenFiles == nil {
		t.Error("seenFiles map not initialized")
	}
	if w.eventChan == nil {
		t.Error("eventChan not initialized")
	}
	if w.pollInterval != 500*time.Millisecond {
		t.Errorf("pollInterval = %v, want %v", w.pollInterval, 500*time.Millisecond)
	}
}

func TestLiveWatcher_SetPollInterval(t *testing.T) {
	w := NewLiveWatcher("/test", "proj")
	w.SetPollInterval(100 * time.Millisecond)

	if w.pollInterval != 100*time.Millisecond {
		t.Errorf("pollInterval = %v, want %v", w.pollInterval, 100*time.Millisecond)
	}
}

func TestLiveWatcher_Events(t *testing.T) {
	w := NewLiveWatcher("/test", "proj")
	ch := w.Events()

	if ch == nil {
		t.Error("Events() returned nil channel")
	}
}

func TestLiveWatcher_BuildPattern(t *testing.T) {
	tests := []struct {
		name     string
		rootDir  string
		project  string
		expected string
	}{
		{
			name:     "with project",
			rootDir:  "/home/user/.claude/projects",
			project:  "myapp",
			expected: "/home/user/.claude/projects/myapp/*.jsonl",
		},
		{
			name:     "without project",
			rootDir:  "/home/user/.claude/projects",
			project:  "",
			expected: "/home/user/.claude/projects/*/*.jsonl",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := NewLiveWatcher(tt.rootDir, tt.project)
			pattern := w.buildPattern()
			if pattern != tt.expected {
				t.Errorf("buildPattern() = %q, want %q", pattern, tt.expected)
			}
		})
	}
}

func TestLiveWatcher_ProcessFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test JSONL file with valid events
	content := `{"type":"session_start","session_id":"test-123","project":"test","timestamp":"2025-01-15T10:00:00Z","status":"active"}
{"type":"tool_call","timestamp":"2025-01-15T10:00:01Z","tool_name":"Read","parameters":{},"success":true,"duration":100}
{"type":"bash_command","timestamp":"2025-01-15T10:00:02Z","command":"ls","exit_code":0,"output":"","output_length":0,"duration":10,"success":true}
`
	testFile := filepath.Join(tmpDir, "test.jsonl")
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	w := NewLiveWatcher(tmpDir, "")
	w.positions[testFile] = 0
	w.seenFiles[testFile] = true

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := w.processFile(ctx, testFile)
	if err != nil {
		t.Errorf("processFile() error = %v", err)
	}

	// Verify position updated
	if w.positions[testFile] != int64(len(content)) {
		t.Errorf("position = %d, want %d", w.positions[testFile], len(content))
	}

	// Read events from channel
	eventCount := 0
	timeout := time.After(100 * time.Millisecond)
loop:
	for {
		select {
		case event, ok := <-w.eventChan:
			if !ok {
				break loop
			}
			if event != nil {
				eventCount++
			}
		case <-timeout:
			break loop
		}
	}

	if eventCount != 2 { // tool_call and bash_command (session_start is metadata)
		t.Errorf("received %d events, want 2", eventCount)
	}
}

func TestLiveWatcher_IncrementalRead(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.jsonl")

	// Initial content
	content1 := `{"type":"session_start","session_id":"test-123","project":"test","timestamp":"2025-01-15T10:00:00Z","status":"active"}
{"type":"tool_call","timestamp":"2025-01-15T10:00:01Z","tool_name":"Read","parameters":{},"success":true,"duration":100}
`
	if err := os.WriteFile(testFile, []byte(content1), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	w := NewLiveWatcher(tmpDir, "")
	w.positions[testFile] = 0
	w.seenFiles[testFile] = true

	ctx := context.Background()

	// First read
	if err := w.processFile(ctx, testFile); err != nil {
		t.Errorf("first processFile() error = %v", err)
	}

	// Drain events
	drained := 0
	timeout := time.After(100 * time.Millisecond)
drain1:
	for {
		select {
		case <-w.eventChan:
			drained++
		case <-timeout:
			break drain1
		}
	}

	// Append new content
	content2 := `{"type":"bash_command","timestamp":"2025-01-15T10:00:02Z","command":"pwd","exit_code":0,"output":"","output_length":0,"duration":5,"success":true}
`
	f, err := os.OpenFile(testFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("failed to open file for append: %v", err)
	}
	if _, err := f.WriteString(content2); err != nil {
		f.Close()
		t.Fatalf("failed to append to file: %v", err)
	}
	f.Close()

	// Second read (incremental)
	if err := w.processFile(ctx, testFile); err != nil {
		t.Errorf("second processFile() error = %v", err)
	}

	// Should only get the new event
	newEvents := 0
	timeout = time.After(100 * time.Millisecond)
drain2:
	for {
		select {
		case event := <-w.eventChan:
			if event != nil {
				newEvents++
				if event.GetType() != "bash_command" {
					t.Errorf("expected bash_command, got %s", event.GetType())
				}
			}
		case <-timeout:
			break drain2
		}
	}

	if newEvents != 1 {
		t.Errorf("received %d new events, want 1", newEvents)
	}
}

func TestLiveWatcher_Start_ContextCancel(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "testproj")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	// Create initial file
	testFile := filepath.Join(projectDir, "test.jsonl")
	content := `{"type":"session_start","session_id":"test-123","project":"test","timestamp":"2025-01-15T10:00:00Z","status":"active"}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	w := NewLiveWatcher(tmpDir, "testproj")
	w.SetPollInterval(10 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := w.Start(ctx)
	if err != context.DeadlineExceeded {
		t.Errorf("Start() error = %v, want context.DeadlineExceeded", err)
	}
}

func TestLiveWatcher_GetPositions(t *testing.T) {
	w := NewLiveWatcher("/test", "proj")
	w.positions["/test/file1.jsonl"] = 100
	w.positions["/test/file2.jsonl"] = 200

	positions := w.GetPositions()

	if len(positions) != 2 {
		t.Errorf("GetPositions() returned %d entries, want 2", len(positions))
	}
	if positions["/test/file1.jsonl"] != 100 {
		t.Errorf("file1 position = %d, want 100", positions["/test/file1.jsonl"])
	}

	// Verify it's a copy (modification doesn't affect original)
	positions["/test/file1.jsonl"] = 999
	if w.positions["/test/file1.jsonl"] != 100 {
		t.Error("GetPositions() did not return a copy")
	}
}

func TestLiveWatcher_GetSeenFiles(t *testing.T) {
	w := NewLiveWatcher("/test", "proj")
	w.seenFiles["/test/file1.jsonl"] = true
	w.seenFiles["/test/file2.jsonl"] = true

	seen := w.GetSeenFiles()

	if len(seen) != 2 {
		t.Errorf("GetSeenFiles() returned %d entries, want 2", len(seen))
	}
	if !seen["/test/file1.jsonl"] {
		t.Error("file1 not in seen files")
	}

	// Verify it's a copy
	seen["/test/new.jsonl"] = true
	if _, ok := w.seenFiles["/test/new.jsonl"]; ok {
		t.Error("GetSeenFiles() did not return a copy")
	}
}

func TestLiveWatcher_ParseAndEmit_MalformedLines(t *testing.T) {
	w := NewLiveWatcher("/test", "proj")

	ctx := context.Background()

	// Mix of valid and invalid lines
	data := []byte(`{"type":"tool_call","timestamp":"2025-01-15T10:00:00Z","tool_name":"Read","parameters":{},"success":true,"duration":100}
{invalid json}
{"type":"bash_command","timestamp":"2025-01-15T10:00:01Z","command":"ls","exit_code":0,"output":"","output_length":0,"duration":10,"success":true}
`)

	err := w.parseAndEmit(ctx, data)
	if err != nil {
		t.Errorf("parseAndEmit() error = %v", err)
	}

	// Should get 2 valid events (malformed skipped)
	eventCount := 0
	timeout := time.After(100 * time.Millisecond)
loop:
	for {
		select {
		case event := <-w.eventChan:
			if event != nil {
				eventCount++
			}
		case <-timeout:
			break loop
		}
	}

	if eventCount != 2 {
		t.Errorf("received %d events, want 2", eventCount)
	}
}

func TestLiveWatcher_ScanFiles_Discovery(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "myproject")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	// Create multiple JSONL files
	for _, name := range []string{"agent-001.jsonl", "agent-002.jsonl", "session.jsonl"} {
		content := `{"type":"session_start","session_id":"test-123","project":"test","timestamp":"2025-01-15T10:00:00Z","status":"active"}
`
		if err := os.WriteFile(filepath.Join(projectDir, name), []byte(content), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}

	w := NewLiveWatcher(tmpDir, "myproject")
	ctx := context.Background()

	if err := w.scanFiles(ctx); err != nil {
		t.Errorf("scanFiles() error = %v", err)
	}

	seen := w.GetSeenFiles()
	if len(seen) != 3 {
		t.Errorf("discovered %d files, want 3", len(seen))
	}
}
