package behavioral

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseSessionFile(t *testing.T) {
	tests := []struct {
		name          string
		content       string
		wantErr       bool
		wantEventCnt  int
		expectedTypes []string
	}{
		{
			name: "valid mixed events",
			content: `{"type":"session_start","session_id":"test-123","project":"conductor","timestamp":"2025-01-15T09:00:00Z","status":"completed","agent_name":"general","duration":60000,"success":true,"error_count":0}
{"type":"tool_call","timestamp":"2025-01-15T10:00:00Z","tool_name":"Read","parameters":{"file_path":"/test.go"},"success":true,"duration":100}
{"type":"bash_command","timestamp":"2025-01-15T10:01:00Z","command":"go test","exit_code":0,"output":"PASS","output_length":4,"duration":500,"success":true}
{"type":"file_operation","timestamp":"2025-01-15T10:02:00Z","operation":"write","path":"/test.go","success":true,"size_bytes":1024,"duration":50}
{"type":"token_usage","timestamp":"2025-01-15T10:03:00Z","input_tokens":1000,"output_tokens":500,"cost_usd":0.05,"model_name":"claude-sonnet-4-5"}`,
			wantErr:       false,
			wantEventCnt:  4,
			expectedTypes: []string{"tool_call", "bash_command", "file_operation", "token_usage"},
		},
		{
			name: "malformed line skipped",
			content: `{"type":"session_metadata","id":"test-456","project":"conductor","timestamp":"2025-01-15T09:00:00Z","status":"completed","agent_name":"general","duration":30000,"success":true,"error_count":0}
{"type":"tool_call","timestamp":"2025-01-15T10:00:00Z","tool_name":"Read","parameters":{},"success":true,"duration":100}
{invalid json}
{"type":"bash_command","timestamp":"2025-01-15T10:01:00Z","command":"ls","exit_code":0,"output":"","output_length":0,"duration":10,"success":true}`,
			wantErr:       false,
			wantEventCnt:  2,
			expectedTypes: []string{"tool_call", "bash_command"},
		},
		{
			name:         "empty file",
			content:      "",
			wantErr:      false, // Empty file returns empty session data, not error
			wantEventCnt: 0,
		},
		{
			name:         "no valid events",
			content:      `{invalid}`,
			wantErr:      false, // Invalid content is skipped, returns empty session
			wantEventCnt: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.jsonl")
			if err := os.WriteFile(tmpFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}

			// Parse
			sessionData, err := ParseSessionFile(tmpFile)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseSessionFile() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseSessionFile() unexpected error: %v", err)
				return
			}

			if len(sessionData.Events) != tt.wantEventCnt {
				t.Errorf("ParseSessionFile() got %d events, want %d", len(sessionData.Events), tt.wantEventCnt)
			}

			// Verify event types
			for i, expectedType := range tt.expectedTypes {
				if i >= len(sessionData.Events) {
					break
				}
				if sessionData.Events[i].GetType() != expectedType {
					t.Errorf("Event %d: got type %s, want %s", i, sessionData.Events[i].GetType(), expectedType)
				}
			}
		})
	}
}

func TestToolCallEvent_Validate(t *testing.T) {
	tests := []struct {
		name    string
		event   *ToolCallEvent
		wantErr bool
	}{
		{
			name: "valid tool call",
			event: &ToolCallEvent{
				BaseEvent:  BaseEvent{Type: "tool_call", Timestamp: time.Now()},
				ToolName:   "Read",
				Parameters: map[string]interface{}{"file_path": "/test.go"},
				Success:    true,
				Duration:   100,
			},
			wantErr: false,
		},
		{
			name: "missing tool name",
			event: &ToolCallEvent{
				BaseEvent:  BaseEvent{Type: "tool_call", Timestamp: time.Now()},
				ToolName:   "",
				Parameters: map[string]interface{}{},
				Success:    true,
			},
			wantErr: true,
		},
		{
			name: "missing timestamp",
			event: &ToolCallEvent{
				BaseEvent:  BaseEvent{Type: "tool_call"},
				ToolName:   "Read",
				Parameters: map[string]interface{}{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.event.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ToolCallEvent.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBashCommandEvent_Validate(t *testing.T) {
	tests := []struct {
		name    string
		event   *BashCommandEvent
		wantErr bool
	}{
		{
			name: "valid bash command",
			event: &BashCommandEvent{
				BaseEvent:    BaseEvent{Type: "bash_command", Timestamp: time.Now()},
				Command:      "go test",
				ExitCode:     0,
				Output:       "PASS",
				OutputLength: 4,
				Duration:     500,
				Success:      true,
			},
			wantErr: false,
		},
		{
			name: "missing command",
			event: &BashCommandEvent{
				BaseEvent: BaseEvent{Type: "bash_command", Timestamp: time.Now()},
				Command:   "",
			},
			wantErr: true,
		},
		{
			name: "missing timestamp",
			event: &BashCommandEvent{
				BaseEvent: BaseEvent{Type: "bash_command"},
				Command:   "ls",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.event.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("BashCommandEvent.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFileOperationEvent_Validate(t *testing.T) {
	tests := []struct {
		name    string
		event   *FileOperationEvent
		wantErr bool
	}{
		{
			name: "valid file operation",
			event: &FileOperationEvent{
				BaseEvent: BaseEvent{Type: "file_operation", Timestamp: time.Now()},
				Operation: "write",
				Path:      "/test.go",
				Success:   true,
				SizeBytes: 1024,
				Duration:  50,
			},
			wantErr: false,
		},
		{
			name: "missing operation",
			event: &FileOperationEvent{
				BaseEvent: BaseEvent{Type: "file_operation", Timestamp: time.Now()},
				Operation: "",
				Path:      "/test.go",
			},
			wantErr: true,
		},
		{
			name: "missing file path",
			event: &FileOperationEvent{
				BaseEvent: BaseEvent{Type: "file_operation", Timestamp: time.Now()},
				Operation: "read",
				Path:      "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.event.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("FileOperationEvent.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTokenUsageEvent_Validate(t *testing.T) {
	tests := []struct {
		name    string
		event   *TokenUsageEvent
		wantErr bool
	}{
		{
			name: "valid token usage",
			event: &TokenUsageEvent{
				BaseEvent:    BaseEvent{Type: "token_usage", Timestamp: time.Now()},
				InputTokens:  1000,
				OutputTokens: 500,
				CostUSD:      0.05,
				ModelName:    "claude-sonnet-4-5",
			},
			wantErr: false,
		},
		{
			name: "negative input tokens",
			event: &TokenUsageEvent{
				BaseEvent:    BaseEvent{Type: "token_usage", Timestamp: time.Now()},
				InputTokens:  -100,
				OutputTokens: 500,
			},
			wantErr: true,
		},
		{
			name: "negative output tokens",
			event: &TokenUsageEvent{
				BaseEvent:    BaseEvent{Type: "token_usage", Timestamp: time.Now()},
				InputTokens:  1000,
				OutputTokens: -500,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.event.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("TokenUsageEvent.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTextEvent_Validate(t *testing.T) {
	tests := []struct {
		name    string
		event   *TextEvent
		wantErr bool
	}{
		{
			name: "valid text event with content",
			event: &TextEvent{
				BaseEvent: BaseEvent{Type: "text", Timestamp: time.Now()},
				Text:      "Hello, I'll help you with that.",
				Role:      "assistant",
			},
			wantErr: false,
		},
		{
			name: "valid text event empty text",
			event: &TextEvent{
				BaseEvent: BaseEvent{Type: "text", Timestamp: time.Now()},
				Text:      "",
				Role:      "assistant",
			},
			wantErr: false, // Text can be empty per Validate()
		},
		{
			name: "valid user text event",
			event: &TextEvent{
				BaseEvent: BaseEvent{Type: "text", Timestamp: time.Now()},
				Text:      "Please help me with this code",
				Role:      "user",
			},
			wantErr: false,
		},
		{
			name: "missing type",
			event: &TextEvent{
				BaseEvent: BaseEvent{Type: "", Timestamp: time.Now()},
				Text:      "Some text",
				Role:      "assistant",
			},
			wantErr: true,
		},
		{
			name: "missing timestamp",
			event: &TextEvent{
				BaseEvent: BaseEvent{Type: "text"},
				Text:      "Some text",
				Role:      "assistant",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.event.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("TextEvent.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExtractTextFromContent(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		role      string
		wantCount int
		wantTexts []string
	}{
		{
			name:      "single text block",
			content:   `[{"type":"text","text":"Hello world"}]`,
			role:      "assistant",
			wantCount: 1,
			wantTexts: []string{"Hello world"},
		},
		{
			name:      "multiple text blocks",
			content:   `[{"type":"text","text":"First message"},{"type":"text","text":"Second message"}]`,
			role:      "assistant",
			wantCount: 2,
			wantTexts: []string{"First message", "Second message"},
		},
		{
			name:      "mixed content with tool_use",
			content:   `[{"type":"text","text":"Let me read that file"},{"type":"tool_use","id":"123","name":"Read","input":{}}]`,
			role:      "assistant",
			wantCount: 1,
			wantTexts: []string{"Let me read that file"},
		},
		{
			name:      "empty text blocks skipped",
			content:   `[{"type":"text","text":""},{"type":"text","text":"Non-empty"}]`,
			role:      "assistant",
			wantCount: 1,
			wantTexts: []string{"Non-empty"},
		},
		{
			name:      "user role preserved",
			content:   `[{"type":"text","text":"User message"}]`,
			role:      "user",
			wantCount: 1,
			wantTexts: []string{"User message"},
		},
		{
			name:      "invalid json returns empty",
			content:   `{invalid}`,
			role:      "assistant",
			wantCount: 0,
		},
		{
			name:      "non-array returns empty",
			content:   `{"type":"text","text":"Not an array"}`,
			role:      "assistant",
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := time.Now()
			events := extractTextFromContent([]byte(tt.content), ts, tt.role)

			if len(events) != tt.wantCount {
				t.Errorf("extractTextFromContent() got %d events, want %d", len(events), tt.wantCount)
				return
			}

			for i, event := range events {
				textEvent, ok := event.(*TextEvent)
				if !ok {
					t.Errorf("Event %d: expected *TextEvent, got %T", i, event)
					continue
				}

				if textEvent.Role != tt.role {
					t.Errorf("Event %d: got role %q, want %q", i, textEvent.Role, tt.role)
				}

				if i < len(tt.wantTexts) && textEvent.Text != tt.wantTexts[i] {
					t.Errorf("Event %d: got text %q, want %q", i, textEvent.Text, tt.wantTexts[i])
				}

				if textEvent.GetType() != "text" {
					t.Errorf("Event %d: got type %q, want %q", i, textEvent.GetType(), "text")
				}
			}
		})
	}
}

func TestParseEventLine(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		wantType  string
		wantErr   bool
		validator func(Event) error
	}{
		{
			name:     "tool call event",
			line:     `{"type":"tool_call","timestamp":"2025-01-15T10:00:00Z","tool_name":"Read","parameters":{"file_path":"/test.go"},"success":true,"duration":100}`,
			wantType: "tool_call",
			wantErr:  false,
			validator: func(e Event) error {
				tc, ok := e.(*ToolCallEvent)
				if !ok {
					return errors.New("expected ToolCallEvent")
				}
				if tc.ToolName != "Read" {
					return errors.New("expected tool name 'Read'")
				}
				return nil
			},
		},
		{
			name:     "bash command event",
			line:     `{"type":"bash_command","timestamp":"2025-01-15T10:01:00Z","command":"go test","exit_code":0,"output":"PASS","duration":500,"background":false}`,
			wantType: "bash_command",
			wantErr:  false,
			validator: func(e Event) error {
				bc, ok := e.(*BashCommandEvent)
				if !ok {
					return errors.New("expected BashCommandEvent")
				}
				if bc.Command != "go test" {
					return errors.New("expected command 'go test'")
				}
				return nil
			},
		},
		{
			name:     "unknown event type",
			line:     `{"type":"unknown_type","timestamp":"2025-01-15T10:00:00Z"}`,
			wantType: "",
			wantErr:  true,
		},
		{
			name:     "invalid json",
			line:     `{invalid json}`,
			wantType: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events, err := parseEventLine(tt.line)

			if tt.wantErr {
				if err == nil {
					t.Errorf("parseEventLine() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("parseEventLine() unexpected error: %v", err)
				return
			}

			// parseEventLine now returns []Event, check first event
			if len(events) == 0 {
				t.Errorf("parseEventLine() returned no events")
				return
			}

			event := events[0]
			if event.GetType() != tt.wantType {
				t.Errorf("parseEventLine() got type %s, want %s", event.GetType(), tt.wantType)
			}

			if tt.validator != nil {
				if err := tt.validator(event); err != nil {
					t.Errorf("validator failed: %v", err)
				}
			}
		})
	}
}
