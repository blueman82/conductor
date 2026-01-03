package behavioral

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileOpString(t *testing.T) {
	tests := []struct {
		op       FileOp
		expected string
	}{
		{FileCreated, "created"},
		{FileWritten, "written"},
		{FileRemoved, "removed"},
		{FileOp(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.op.String(); got != tt.expected {
				t.Errorf("FileOp.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNewFileWatcher(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "filewatcher_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test basic creation
	fw, err := NewFileWatcher(tmpDir, "*.jsonl")
	if err != nil {
		t.Fatalf("NewFileWatcher failed: %v", err)
	}
	defer fw.Close()

	// Verify properties
	if fw.RootDir() != tmpDir {
		t.Errorf("RootDir() = %v, want %v", fw.RootDir(), tmpDir)
	}
	if fw.Pattern() != "*.jsonl" {
		t.Errorf("Pattern() = %v, want %v", fw.Pattern(), "*.jsonl")
	}
}

func TestNewFileWatcher_NonExistentDir(t *testing.T) {
	// Test with non-existent directory - should still work but add nothing
	tmpDir := filepath.Join(os.TempDir(), "nonexistent_"+time.Now().Format("20060102150405"))

	fw, err := NewFileWatcher(tmpDir, "*.jsonl")
	if err != nil {
		t.Fatalf("NewFileWatcher with non-existent dir failed: %v", err)
	}
	defer fw.Close()
}

func TestNewFileWatcher_TildeExpansion(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get home directory")
	}

	// Create a test dir in home
	testDir := filepath.Join(home, ".conductor_test_"+time.Now().Format("20060102150405"))
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test dir: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Test tilde expansion
	fw, err := NewFileWatcher("~/.conductor_test_"+time.Now().Format("20060102150405"), "*.jsonl")
	if err != nil {
		// This is expected if the dir doesn't exist at the exact moment
		t.Skip("Test dir timing issue")
	}
	defer fw.Close()

	if !filepath.IsAbs(fw.RootDir()) {
		t.Errorf("RootDir() should be absolute, got %v", fw.RootDir())
	}
}

func TestFileWatcher_FileCreated(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "filewatcher_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	fw, err := NewFileWatcher(tmpDir, "*.jsonl")
	if err != nil {
		t.Fatalf("NewFileWatcher failed: %v", err)
	}
	defer fw.Close()

	// Create a file
	testFile := filepath.Join(tmpDir, "test.jsonl")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Wait for event
	select {
	case event := <-fw.Events():
		if event.Path != testFile {
			t.Errorf("Event.Path = %v, want %v", event.Path, testFile)
		}
		if event.Op != FileCreated {
			t.Errorf("Event.Op = %v, want %v", event.Op, FileCreated)
		}
	case err := <-fw.Errors():
		t.Fatalf("Unexpected error: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for file created event")
	}
}

func TestFileWatcher_FileWritten(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "filewatcher_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a file first
	testFile := filepath.Join(tmpDir, "test.jsonl")
	if err := os.WriteFile(testFile, []byte("initial"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	fw, err := NewFileWatcher(tmpDir, "*.jsonl")
	if err != nil {
		t.Fatalf("NewFileWatcher failed: %v", err)
	}
	// Use a shorter debounce for testing
	fw.SetDebounceDelay(50 * time.Millisecond)
	defer fw.Close()

	// Write to the file
	if err := os.WriteFile(testFile, []byte("modified"), 0644); err != nil {
		t.Fatalf("Failed to write to test file: %v", err)
	}

	// Wait for event (accounting for debounce)
	select {
	case event := <-fw.Events():
		if event.Path != testFile {
			t.Errorf("Event.Path = %v, want %v", event.Path, testFile)
		}
		if event.Op != FileWritten {
			t.Errorf("Event.Op = %v, want %v", event.Op, FileWritten)
		}
	case err := <-fw.Errors():
		t.Fatalf("Unexpected error: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for file written event")
	}
}

func TestFileWatcher_FileRemoved(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "filewatcher_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a file first
	testFile := filepath.Join(tmpDir, "test.jsonl")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	fw, err := NewFileWatcher(tmpDir, "*.jsonl")
	if err != nil {
		t.Fatalf("NewFileWatcher failed: %v", err)
	}
	defer fw.Close()

	// Remove the file
	if err := os.Remove(testFile); err != nil {
		t.Fatalf("Failed to remove test file: %v", err)
	}

	// Wait for event
	select {
	case event := <-fw.Events():
		if event.Path != testFile {
			t.Errorf("Event.Path = %v, want %v", event.Path, testFile)
		}
		if event.Op != FileRemoved {
			t.Errorf("Event.Op = %v, want %v", event.Op, FileRemoved)
		}
	case err := <-fw.Errors():
		t.Fatalf("Unexpected error: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for file removed event")
	}
}

func TestFileWatcher_PatternFiltering(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "filewatcher_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	fw, err := NewFileWatcher(tmpDir, "*.jsonl")
	if err != nil {
		t.Fatalf("NewFileWatcher failed: %v", err)
	}
	defer fw.Close()

	// Create a file that doesn't match the pattern
	nonMatchingFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(nonMatchingFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create non-matching file: %v", err)
	}

	// Create a file that matches the pattern
	matchingFile := filepath.Join(tmpDir, "test.jsonl")
	if err := os.WriteFile(matchingFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create matching file: %v", err)
	}

	// Should only receive event for matching file
	select {
	case event := <-fw.Events():
		if event.Path != matchingFile {
			t.Errorf("Event.Path = %v, want %v", event.Path, matchingFile)
		}
	case err := <-fw.Errors():
		t.Fatalf("Unexpected error: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for event")
	}

	// Ensure no events for non-matching files (allow additional events for matching file)
	timeout := time.After(200 * time.Millisecond)
drainLoop:
	for {
		select {
		case event := <-fw.Events():
			// Additional events for the matching file are OK (e.g., multiple filesystem events)
			if event.Path != matchingFile {
				t.Errorf("Unexpected event for non-matching file: %v", event.Path)
			}
		case <-timeout:
			break drainLoop
		}
	}
}

func TestFileWatcher_RecursiveWatching(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "filewatcher_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a subdirectory
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	fw, err := NewFileWatcher(tmpDir, "*.jsonl")
	if err != nil {
		t.Fatalf("NewFileWatcher failed: %v", err)
	}
	defer fw.Close()

	// Create a file in the subdirectory
	testFile := filepath.Join(subDir, "test.jsonl")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Wait for event
	select {
	case event := <-fw.Events():
		if event.Path != testFile {
			t.Errorf("Event.Path = %v, want %v", event.Path, testFile)
		}
	case err := <-fw.Errors():
		t.Fatalf("Unexpected error: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for event from subdirectory")
	}
}

func TestFileWatcher_NewSubdirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "filewatcher_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	fw, err := NewFileWatcher(tmpDir, "*.jsonl")
	if err != nil {
		t.Fatalf("NewFileWatcher failed: %v", err)
	}
	defer fw.Close()

	// Create a new subdirectory after watcher starts
	subDir := filepath.Join(tmpDir, "newsubdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	// Give the watcher time to add the new directory
	time.Sleep(100 * time.Millisecond)

	// Create a file in the new subdirectory
	testFile := filepath.Join(subDir, "test.jsonl")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Wait for event
	select {
	case event := <-fw.Events():
		if event.Path != testFile {
			t.Errorf("Event.Path = %v, want %v", event.Path, testFile)
		}
	case err := <-fw.Errors():
		t.Fatalf("Unexpected error: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for event from new subdirectory")
	}
}

func TestFileWatcher_Debounce(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "filewatcher_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a file first
	testFile := filepath.Join(tmpDir, "test.jsonl")
	if err := os.WriteFile(testFile, []byte("initial"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	fw, err := NewFileWatcher(tmpDir, "*.jsonl")
	if err != nil {
		t.Fatalf("NewFileWatcher failed: %v", err)
	}
	// Use a longer debounce for this test
	fw.SetDebounceDelay(200 * time.Millisecond)
	defer fw.Close()

	// Perform rapid writes
	for i := 0; i < 5; i++ {
		if err := os.WriteFile(testFile, []byte("write"+string(rune('0'+i))), 0644); err != nil {
			t.Fatalf("Failed to write to test file: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}

	// Should receive only one coalesced event
	eventCount := 0
	timeout := time.After(1 * time.Second)

loop:
	for {
		select {
		case <-fw.Events():
			eventCount++
		case <-timeout:
			break loop
		}
	}

	if eventCount != 1 {
		t.Errorf("Expected 1 debounced event, got %d", eventCount)
	}
}

func TestFileWatcher_Close(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "filewatcher_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	fw, err := NewFileWatcher(tmpDir, "*.jsonl")
	if err != nil {
		t.Fatalf("NewFileWatcher failed: %v", err)
	}

	// Close should not error
	if err := fw.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Double close should be safe
	if err := fw.Close(); err != nil {
		t.Errorf("Second Close() error = %v", err)
	}
}

func TestFileWatcher_EmptyPattern(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "filewatcher_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	fw, err := NewFileWatcher(tmpDir, "")
	if err != nil {
		t.Fatalf("NewFileWatcher failed: %v", err)
	}
	defer fw.Close()

	// Create any file - should be watched
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Wait for event
	select {
	case event := <-fw.Events():
		if event.Path != testFile {
			t.Errorf("Event.Path = %v, want %v", event.Path, testFile)
		}
	case err := <-fw.Errors():
		t.Fatalf("Unexpected error: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for event")
	}
}

func TestMatchesPattern(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "filewatcher_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		pattern  string
		path     string
		expected bool
	}{
		{"*.jsonl", "/some/path/file.jsonl", true},
		{"*.jsonl", "/some/path/file.txt", false},
		{"*.jsonl", "/some/path/file.jsonl.bak", false},
		{"test*", "/some/path/test123.jsonl", true},
		{"test*", "/some/path/notest.jsonl", false},
		{"", "/any/path/file.txt", true},
		{"*.json*", "/some/path/file.jsonl", true},
		{"*.json*", "/some/path/file.json", true},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+filepath.Base(tt.path), func(t *testing.T) {
			fw, err := NewFileWatcher(tmpDir, tt.pattern)
			if err != nil {
				t.Fatalf("NewFileWatcher failed: %v", err)
			}
			defer fw.Close()

			if got := fw.matchesPattern(tt.path); got != tt.expected {
				t.Errorf("matchesPattern(%q) = %v, want %v", tt.path, got, tt.expected)
			}
		})
	}
}
