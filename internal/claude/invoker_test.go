package claude

import (
	"testing"
)

func TestParseResponse(t *testing.T) {
	tests := []struct {
		name          string
		rawOutput     []byte
		wantContent   string
		wantSessionID string
		wantErr       bool
	}{
		{
			name:          "valid JSON with content field",
			rawOutput:     []byte(`{"content":"Hello World","error":"","session_id":"abc-123"}`),
			wantContent:   "Hello World",
			wantSessionID: "abc-123",
			wantErr:       false,
		},
		{
			name:          "valid JSON without session_id",
			rawOutput:     []byte(`{"content":"Task completed","error":""}`),
			wantContent:   "Task completed",
			wantSessionID: "",
			wantErr:       false,
		},
		{
			name:          "structured_output from --json-schema",
			rawOutput:     []byte(`{"type":"result","session_id":"test-123","structured_output":{"status":"success","summary":"Done"}}`),
			wantContent:   `{"status":"success","summary":"Done"}`,
			wantSessionID: "test-123",
			wantErr:       false,
		},
		{
			name:          "code-fenced JSON output - fallback extraction",
			rawOutput:     []byte("Here is the result:\n```json\n{\"status\":\"success\"}\n```\n"),
			wantContent:   `{"status":"success"}`,
			wantSessionID: "",
			wantErr:       false,
		},
		{
			name:          "mixed output with error prefix before JSON",
			rawOutput:     []byte("Error: some warning\n" + `{"content":"Result","session_id":"mixed-456"}`),
			wantContent:   "Result",
			wantSessionID: "mixed-456",
			wantErr:       false,
		},
		{
			name:          "plain text output without JSON",
			rawOutput:     []byte("Plain text output without JSON"),
			wantContent:   "",  // No JSON braces found, returns empty
			wantSessionID: "",
			wantErr:       false,
		},
		{
			name:          "empty output",
			rawOutput:     []byte(""),
			wantContent:   "",
			wantSessionID: "",
			wantErr:       false,
		},
		{
			name:          "raw JSON without wrapper - fallback extraction",
			rawOutput:     []byte(`{"status":"success","summary":"Task done","output":"Created file"}`),
			wantContent:   `{"status":"success","summary":"Task done","output":"Created file"}`,
			wantSessionID: "",
			wantErr:       false,
		},
		{
			name:          "JSON with prose before - fallback extraction",
			rawOutput:     []byte("Some prose before the JSON response\n{\"status\":\"success\"}"),
			wantContent:   `{"status":"success"}`,
			wantSessionID: "",
			wantErr:       false,
		},
		{
			name:          "structured_output null - falls through to content",
			rawOutput:     []byte(`{"type":"result","content":"Via content field","session_id":"test-789","structured_output":null}`),
			wantContent:   "Via content field",
			wantSessionID: "test-789",
			wantErr:       false,
		},
		{
			name:          "structured_output empty object - falls through to content",
			rawOutput:     []byte(`{"type":"result","content":"Via content field","session_id":"test-abc","structured_output":{}}`),
			wantContent:   "Via content field",
			wantSessionID: "test-abc",
			wantErr:       false,
		},
		{
			name:          "result field used by some agents",
			rawOutput:     []byte(`{"type":"result","result":"Agent response text","session_id":"result-123"}`),
			wantContent:   "Agent response text",
			wantSessionID: "result-123",
			wantErr:       false,
		},
		{
			name:          "malformed JSON without closing brace - returns empty",
			rawOutput:     []byte(`{"status":"success`),
			wantContent:   "",  // No closing brace, returns empty
			wantSessionID: "",
			wantErr:       false,
		},
		{
			name:          "only opening brace - no valid JSON",
			rawOutput:     []byte(`{`),
			wantContent:   "",
			wantSessionID: "",
			wantErr:       false,
		},
		{
			name:          "only closing brace - no valid JSON",
			rawOutput:     []byte(`}`),
			wantContent:   "",
			wantSessionID: "",
			wantErr:       false,
		},
		{
			name:          "nested JSON in content",
			rawOutput:     []byte(`{"content":"{\"nested\":\"value\"}","session_id":"nested-123"}`),
			wantContent:   `{"nested":"value"}`,
			wantSessionID: "nested-123",
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, sessionID, err := ParseResponse(tt.rawOutput)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if content != tt.wantContent {
				t.Errorf("ParseResponse() content = %q, want %q", content, tt.wantContent)
			}
			if sessionID != tt.wantSessionID {
				t.Errorf("ParseResponse() sessionID = %q, want %q", sessionID, tt.wantSessionID)
			}
		})
	}
}

func TestParseResponseWithResponseStruct(t *testing.T) {
	// Test using ParseResponse with the Response struct from Invoke
	tests := []struct {
		name          string
		response      *Response
		wantContent   string
		wantSessionID string
	}{
		{
			name: "parse Response.RawOutput",
			response: &Response{
				RawOutput: []byte(`{"content":"Task completed","session_id":"resp-123"}`),
			},
			wantContent:   "Task completed",
			wantSessionID: "resp-123",
		},
		{
			name: "parse Response with structured_output",
			response: &Response{
				RawOutput: []byte(`{"structured_output":{"status":"success","summary":"Done"},"session_id":"struct-456"}`),
			},
			wantContent:   `{"status":"success","summary":"Done"}`,
			wantSessionID: "struct-456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, sessionID, err := ParseResponse(tt.response.RawOutput)
			if err != nil {
				t.Errorf("ParseResponse() error = %v", err)
				return
			}
			if content != tt.wantContent {
				t.Errorf("ParseResponse() content = %q, want %q", content, tt.wantContent)
			}
			if sessionID != tt.wantSessionID {
				t.Errorf("ParseResponse() sessionID = %q, want %q", sessionID, tt.wantSessionID)
			}
		})
	}
}

func TestNewInvoker(t *testing.T) {
	inv := NewInvoker()
	if inv == nil {
		t.Fatal("NewInvoker() returned nil")
	}
	if inv.ClaudePath != "claude" {
		t.Errorf("ClaudePath = %s, want 'claude'", inv.ClaudePath)
	}
	if inv.SystemPrompt != DefaultSystemPrompt {
		t.Errorf("SystemPrompt not set to DefaultSystemPrompt")
	}
}

func TestDefaultSystemPrompt(t *testing.T) {
	// Verify DefaultSystemPrompt contains critical JSON enforcement
	if DefaultSystemPrompt == "" {
		t.Error("DefaultSystemPrompt should not be empty")
	}
	if !contains(DefaultSystemPrompt, "JSON") {
		t.Error("DefaultSystemPrompt should mention JSON")
	}
	if !contains(DefaultSystemPrompt, "No markdown") {
		t.Error("DefaultSystemPrompt should prohibit markdown")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
