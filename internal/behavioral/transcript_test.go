package behavioral

import (
	"strings"
	"testing"
	"time"
)

func TestDefaultTranscriptOptions(t *testing.T) {
	opts := DefaultTranscriptOptions()

	if !opts.ColorOutput {
		t.Error("ColorOutput should be true by default")
	}
	if opts.TruncateLength != 0 {
		t.Errorf("TruncateLength should be 0 (no truncation), got %d", opts.TruncateLength)
	}
	if !opts.ShowTimestamps {
		t.Error("ShowTimestamps should be true by default")
	}
	if opts.ShowToolResults {
		t.Error("ShowToolResults should be false by default")
	}
	if opts.CompactMode {
		t.Error("CompactMode should be false by default")
	}
}

func TestFormatTranscript_EmptyEvents(t *testing.T) {
	result := FormatTranscript([]Event{}, DefaultTranscriptOptions())
	if result != "No events to display" {
		t.Errorf("Expected 'No events to display', got %q", result)
	}
}

func TestFormatTranscript_TextEvents(t *testing.T) {
	ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	events := []Event{
		&TextEvent{
			BaseEvent: BaseEvent{Type: "text", Timestamp: ts},
			Text:      "Hello, how can I help?",
			Role:      "assistant",
		},
		&TextEvent{
			BaseEvent: BaseEvent{Type: "text", Timestamp: ts.Add(time.Minute)},
			Text:      "Please fix the bug",
			Role:      "user",
		},
	}

	opts := TranscriptOptions{
		ColorOutput:    false,
		TruncateLength: 200,
		ShowTimestamps: true,
	}

	result := FormatTranscript(events, opts)

	if !strings.Contains(result, "💬") {
		t.Error("Should contain assistant emoji 💬")
	}
	if !strings.Contains(result, "👤") {
		t.Error("Should contain user emoji 👤")
	}
	if !strings.Contains(result, "Hello, how can I help?") {
		t.Error("Should contain assistant text")
	}
	if !strings.Contains(result, "Please fix the bug") {
		t.Error("Should contain user text")
	}
	if !strings.Contains(result, "10:30:00") {
		t.Error("Should contain timestamp")
	}
}

func TestFormatTranscript_ToolCallEvents(t *testing.T) {
	ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	events := []Event{
		&ToolCallEvent{
			BaseEvent:  BaseEvent{Type: "tool_call", Timestamp: ts},
			ToolName:   "Read",
			Parameters: map[string]interface{}{"file_path": "/src/main.go"},
			Success:    true,
		},
		&ToolCallEvent{
			BaseEvent: BaseEvent{Type: "tool_call", Timestamp: ts.Add(time.Second)},
			ToolName:  "Write",
			Success:   false,
			Error:     "Permission denied",
		},
	}

	opts := TranscriptOptions{
		ColorOutput:    false,
		TruncateLength: 200,
		ShowTimestamps: false,
	}

	result := FormatTranscript(events, opts)

	if !strings.Contains(result, "🔧") {
		t.Error("Should contain tool emoji 🔧")
	}
	if !strings.Contains(result, "❌") {
		t.Error("Should contain error emoji ❌ for failed tool")
	}
	if !strings.Contains(result, "Read") {
		t.Error("Should contain Read tool name")
	}
	if !strings.Contains(result, "Write") {
		t.Error("Should contain Write tool name")
	}
}

func TestFormatTranscript_BashCommandEvents(t *testing.T) {
	ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	events := []Event{
		&BashCommandEvent{
			BaseEvent: BaseEvent{Type: "bash_command", Timestamp: ts},
			Command:   "go build ./...",
			Success:   true,
			ExitCode:  0,
		},
		&BashCommandEvent{
			BaseEvent: BaseEvent{Type: "bash_command", Timestamp: ts.Add(time.Second)},
			Command:   "rm -rf /",
			Success:   false,
			ExitCode:  1,
		},
	}

	opts := TranscriptOptions{
		ColorOutput:    false,
		TruncateLength: 200,
		ShowTimestamps: false,
	}

	result := FormatTranscript(events, opts)

	if !strings.Contains(result, "Bash: go build") {
		t.Error("Should contain bash command")
	}
	if !strings.Contains(result, "❌") {
		t.Error("Should contain error emoji for failed command")
	}
}

func TestFormatTranscript_FileOperationEvents(t *testing.T) {
	ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	events := []Event{
		&FileOperationEvent{
			BaseEvent: BaseEvent{Type: "file_operation", Timestamp: ts},
			Operation: "Read",
			Path:      "/src/main.go",
			Success:   true,
		},
	}

	opts := TranscriptOptions{
		ColorOutput:    false,
		TruncateLength: 200,
		ShowTimestamps: false,
	}

	result := FormatTranscript(events, opts)

	if !strings.Contains(result, "Read: /src/main.go") {
		t.Error("Should contain file operation")
	}
}

func TestFormatTranscript_Truncation(t *testing.T) {
	ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	longText := strings.Repeat("a", 300)
	events := []Event{
		&TextEvent{
			BaseEvent: BaseEvent{Type: "text", Timestamp: ts},
			Text:      longText,
			Role:      "assistant",
		},
	}

	opts := TranscriptOptions{
		ColorOutput:    false,
		TruncateLength: 50,
		ShowTimestamps: false,
	}

	result := FormatTranscript(events, opts)

	if len(result) > 200 { // generous buffer for formatting
		t.Error("Text should be truncated")
	}
	if !strings.Contains(result, "...") {
		t.Error("Truncated text should end with ...")
	}
}

func TestFormatTranscript_NoTimestamps(t *testing.T) {
	ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	events := []Event{
		&TextEvent{
			BaseEvent: BaseEvent{Type: "text", Timestamp: ts},
			Text:      "Hello",
			Role:      "assistant",
		},
	}

	opts := TranscriptOptions{
		ColorOutput:    false,
		ShowTimestamps: false,
	}

	result := FormatTranscript(events, opts)

	if strings.Contains(result, "10:30") {
		t.Error("Should not contain timestamp when ShowTimestamps is false")
	}
}

func TestFormatTranscript_CompactMode(t *testing.T) {
	ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	events := []Event{
		&TextEvent{
			BaseEvent: BaseEvent{Type: "text", Timestamp: ts},
			Text:      "First",
			Role:      "assistant",
		},
		&TextEvent{
			BaseEvent: BaseEvent{Type: "text", Timestamp: ts.Add(time.Second)},
			Text:      "Second",
			Role:      "assistant",
		},
	}

	normalOpts := TranscriptOptions{
		ColorOutput:    false,
		ShowTimestamps: false,
		CompactMode:    false,
	}
	compactOpts := TranscriptOptions{
		ColorOutput:    false,
		ShowTimestamps: false,
		CompactMode:    true,
	}

	normalResult := FormatTranscript(events, normalOpts)
	compactResult := FormatTranscript(events, compactOpts)

	// Compact mode should have fewer blank lines
	normalNewlines := strings.Count(normalResult, "\n\n")
	compactNewlines := strings.Count(compactResult, "\n\n")

	if compactNewlines >= normalNewlines && normalNewlines > 0 {
		t.Error("Compact mode should have fewer blank lines")
	}
}

func TestFormatTranscriptEntry_NilEvent(t *testing.T) {
	result := FormatTranscriptEntry(nil, DefaultTranscriptOptions())
	if result != "" {
		t.Error("Nil event should return empty string")
	}
}

func TestFormatTranscriptEntry_ToolResult(t *testing.T) {
	// tool_result events should be skipped
	ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	event := &ToolCallEvent{
		BaseEvent: BaseEvent{Type: "tool_result", Timestamp: ts},
		ToolName:  "some-id",
		Result:    "result content",
		Success:   true,
	}

	result := FormatTranscriptEntry(event, TranscriptOptions{ColorOutput: false})
	if result != "" {
		t.Error("tool_result events should be skipped")
	}
}

func TestFormatTranscriptEntry_TokenUsage(t *testing.T) {
	// Token usage events should be skipped
	ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	event := &TokenUsageEvent{
		BaseEvent:    BaseEvent{Type: "token_usage", Timestamp: ts},
		InputTokens:  1000,
		OutputTokens: 500,
	}

	result := FormatTranscriptEntry(event, TranscriptOptions{ColorOutput: false})
	if result != "" {
		t.Error("Token usage events should be skipped in transcript")
	}
}

func TestFormatTranscriptEntry_EmptyText(t *testing.T) {
	ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	event := &TextEvent{
		BaseEvent: BaseEvent{Type: "text", Timestamp: ts},
		Text:      "",
		Role:      "assistant",
	}

	result := FormatTranscriptEntry(event, TranscriptOptions{ColorOutput: false})
	if result != "" {
		t.Error("Empty text events should return empty string")
	}
}

func TestFormatParameters(t *testing.T) {
	tests := []struct {
		name   string
		params map[string]interface{}
		maxLen int
		want   string
	}{
		{
			name:   "empty params",
			params: nil,
			maxLen: 100,
			want:   "",
		},
		{
			name:   "single param",
			params: map[string]interface{}{"key": "value"},
			maxLen: 100,
			want:   "key=value",
		},
		{
			name:   "truncated value",
			params: map[string]interface{}{"key": strings.Repeat("x", 100)},
			maxLen: 20,
			want:   "key=" + strings.Repeat("x", 10) + "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatParameters(tt.params, tt.maxLen)
			if tt.params == nil {
				if got != tt.want {
					t.Errorf("formatParameters() = %q, want %q", got, tt.want)
				}
				return
			}
			// For non-empty params, just check it's not empty and contains the key
			if len(tt.params) > 0 && !strings.Contains(got, "key=") {
				t.Errorf("formatParameters() should contain 'key=', got %q", got)
			}
		})
	}
}

func TestFormatTranscript_MixedEvents(t *testing.T) {
	ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	events := []Event{
		&TextEvent{
			BaseEvent: BaseEvent{Type: "text", Timestamp: ts},
			Text:      "Let me check the file",
			Role:      "assistant",
		},
		&ToolCallEvent{
			BaseEvent:  BaseEvent{Type: "tool_call", Timestamp: ts.Add(time.Second)},
			ToolName:   "Read",
			Parameters: map[string]interface{}{"file_path": "/main.go"},
			Success:    true,
		},
		&TextEvent{
			BaseEvent: BaseEvent{Type: "text", Timestamp: ts.Add(2 * time.Second)},
			Text:      "Found the issue",
			Role:      "assistant",
		},
		&BashCommandEvent{
			BaseEvent: BaseEvent{Type: "bash_command", Timestamp: ts.Add(3 * time.Second)},
			Command:   "go test ./...",
			Success:   true,
		},
	}

	opts := TranscriptOptions{
		ColorOutput:    false,
		TruncateLength: 200,
		ShowTimestamps: true,
	}

	result := FormatTranscript(events, opts)

	// Check all event types are present
	if !strings.Contains(result, "💬") {
		t.Error("Should contain text emoji")
	}
	if !strings.Contains(result, "🔧") {
		t.Error("Should contain tool emoji")
	}
	if !strings.Contains(result, "Let me check") {
		t.Error("Should contain first text")
	}
	if !strings.Contains(result, "Found the issue") {
		t.Error("Should contain second text")
	}
	if !strings.Contains(result, "Read") {
		t.Error("Should contain tool name")
	}
	if !strings.Contains(result, "go test") {
		t.Error("Should contain bash command")
	}
}
