package cmd

import (
	"os"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/behavioral"
)

func TestNewObserveTranscriptCmd(t *testing.T) {
	cmd := NewObserveTranscriptCmd()

	if cmd.Use != "transcript <session-id>" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}

	if cmd.Short != "Display session transcript" {
		t.Errorf("unexpected Short: %s", cmd.Short)
	}

	// Check --raw flag exists
	flag := cmd.Flags().Lookup("raw")
	if flag == nil {
		t.Error("expected --raw flag to exist")
	}
}

func TestDisplaySessionTranscript_NoSessionID(t *testing.T) {
	err := DisplaySessionTranscript("", "", false)
	if err == nil {
		t.Error("DisplaySessionTranscript() should error when session ID is empty")
	}
	if !containsStr(err.Error(), "session ID is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDisplaySessionTranscript_SessionNotFound(t *testing.T) {
	// Set up temp directory
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(tmpDir)
	os.MkdirAll(".conductor", 0755)
	os.WriteFile(".conductor/config.yaml", []byte("learning:\n  db_path: .conductor/learning/test.db\n"), 0644)

	err := DisplaySessionTranscript("nonexistent-session-id-12345", "", false)
	if err == nil {
		t.Error("DisplaySessionTranscript() should error when session not found")
	}
	if !containsStr(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestObserveTranscriptCommand_ExactArgs(t *testing.T) {
	cmd := NewObserveTranscriptCmd()

	// No args should fail
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when no session ID provided")
	}

	// Too many args should fail
	cmd = NewObserveTranscriptCmd()
	cmd.SetArgs([]string{"session1", "session2"})
	err = cmd.Execute()
	if err == nil {
		t.Error("expected error when too many args provided")
	}
}

func TestFormatTranscript_WithEvents(t *testing.T) {
	ts := time.Date(2025, 11, 25, 10, 30, 0, 0, time.UTC)

	events := []behavioral.Event{
		&behavioral.TextEvent{
			BaseEvent: behavioral.BaseEvent{
				Type:      "text",
				Timestamp: ts,
			},
			Text: "Hello, I will help you.",
			Role: "assistant",
		},
		&behavioral.ToolCallEvent{
			BaseEvent: behavioral.BaseEvent{
				Type:      "tool_call",
				Timestamp: ts.Add(time.Second),
			},
			ToolName: "Read",
			Success:  true,
		},
		&behavioral.BashCommandEvent{
			BaseEvent: behavioral.BaseEvent{
				Type:      "bash_command",
				Timestamp: ts.Add(2 * time.Second),
			},
			Command: "go test ./...",
			Success: false,
		},
	}

	opts := behavioral.DefaultTranscriptOptions()
	opts.ColorOutput = false // Disable for test comparison

	result := behavioral.FormatTranscript(events, opts)

	wantContains := []string{
		"[Assistant]",
		"Hello, I will help you.",
		"Read",
		"Bash",
		"go test ./...",
	}

	for _, want := range wantContains {
		if !containsStr(result, want) {
			t.Errorf("FormatTranscript() missing %q in output:\n%s", want, result)
		}
	}
}

func TestFormatTranscript_RawMode(t *testing.T) {
	ts := time.Date(2025, 11, 25, 10, 30, 0, 0, time.UTC)

	events := []behavioral.Event{
		&behavioral.TextEvent{
			BaseEvent: behavioral.BaseEvent{
				Type:      "text",
				Timestamp: ts,
			},
			Text: "Test message",
			Role: "assistant",
		},
	}

	// Raw mode
	opts := behavioral.TranscriptOptions{
		ColorOutput:    false,
		TruncateLength: 500,
		ShowTimestamps: true,
	}

	result := behavioral.FormatTranscript(events, opts)

	// Should contain text but no ANSI codes
	if !containsStr(result, "Test message") {
		t.Errorf("FormatTranscript() should contain message text")
	}

	// Check no ANSI escape codes (raw mode)
	if containsStr(result, "\x1b[") {
		t.Errorf("FormatTranscript() should not contain ANSI codes in raw mode")
	}
}

func TestFormatTranscript_EmptyEvents(t *testing.T) {
	opts := behavioral.DefaultTranscriptOptions()
	result := behavioral.FormatTranscript([]behavioral.Event{}, opts)

	if result != "No events to display" {
		t.Errorf("FormatTranscript() with empty events = %q, want 'No events to display'", result)
	}
}
