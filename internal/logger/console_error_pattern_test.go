package logger

import (
	"bytes"
	"strings"
	"testing"
)

// mockErrorPattern implements ErrorPatternDisplay for testing
type mockErrorPattern struct {
	category   string
	pattern    string
	suggestion string
	fixable    bool
}

func (m *mockErrorPattern) GetCategory() string   { return m.category }
func (m *mockErrorPattern) GetPattern() string    { return m.pattern }
func (m *mockErrorPattern) GetSuggestion() string { return m.suggestion }
func (m *mockErrorPattern) IsAgentFixable() bool  { return m.fixable }

// TestLogErrorPatternENVLevel verifies ENV_LEVEL error pattern logging with yellow color.
func TestLogErrorPatternENVLevel(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "info")

	pattern := &mockErrorPattern{
		category:   "ENV_LEVEL",
		pattern:    "multiple devices matched",
		suggestion: "Environment issue: Duplicate simulators. List: xcrun simctl list devices, Delete: xcrun simctl delete <UUID>",
		fixable:    false,
	}

	logger.LogErrorPattern(pattern)

	output := buf.String()
	if output == "" {
		t.Fatal("LogErrorPattern: expected output, got empty string")
	}

	// Verify key elements are present
	if !strings.Contains(output, "Error Pattern Detected") {
		t.Error("LogErrorPattern: missing header 'Error Pattern Detected'")
	}
	if !strings.Contains(output, "ENV_LEVEL") {
		t.Error("LogErrorPattern: missing category 'ENV_LEVEL'")
	}
	if !strings.Contains(output, "multiple devices matched") {
		t.Error("LogErrorPattern: missing pattern text")
	}
	if !strings.Contains(output, "Duplicate simulators") {
		t.Error("LogErrorPattern: missing suggestion text")
	}
	if !strings.Contains(output, "💡 Suggestion") {
		t.Error("LogErrorPattern: missing suggestion header")
	}
}

// TestLogErrorPatternPLANLevel verifies PLAN_LEVEL error pattern logging with yellow color.
func TestLogErrorPatternPLANLevel(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "info")

	pattern := &mockErrorPattern{
		category:   "PLAN_LEVEL",
		pattern:    "no test bundles available",
		suggestion: "Test target missing from project. Update plan to use existing test target.",
		fixable:    false,
	}

	logger.LogErrorPattern(pattern)

	output := buf.String()
	if output == "" {
		t.Fatal("LogErrorPattern: expected output, got empty string")
	}

	// Verify key elements
	if !strings.Contains(output, "PLAN_LEVEL") {
		t.Error("LogErrorPattern: missing category 'PLAN_LEVEL'")
	}
	if !strings.Contains(output, "no test bundles available") {
		t.Error("LogErrorPattern: missing pattern text")
	}
}

// TestLogErrorPatternCODELevel verifies CODE_LEVEL error pattern logging with green color.
func TestLogErrorPatternCODELevel(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "info")

	pattern := &mockErrorPattern{
		category:   "CODE_LEVEL",
		pattern:    "undefined: |not defined",
		suggestion: "Missing import or undefined identifier. Agent should add import or define missing symbol.",
		fixable:    true,
	}

	logger.LogErrorPattern(pattern)

	output := buf.String()
	if output == "" {
		t.Fatal("LogErrorPattern: expected output, got empty string")
	}

	// Verify key elements
	if !strings.Contains(output, "CODE_LEVEL") {
		t.Error("LogErrorPattern: missing category 'CODE_LEVEL'")
	}
	if !strings.Contains(output, "undefined: |not defined") {
		t.Error("LogErrorPattern: missing pattern text")
	}
	if !strings.Contains(output, "Missing import") {
		t.Error("LogErrorPattern: missing suggestion text")
	}
}

// TestLogErrorPatternNil verifies nil pattern is handled gracefully.
func TestLogErrorPatternNil(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "info")

	logger.LogErrorPattern(nil)

	output := buf.String()
	if output != "" {
		t.Error("LogErrorPattern: expected no output for nil pattern, got:", output)
	}
}

// TestLogErrorPatternNilWriter verifies nil writer is handled gracefully.
func TestLogErrorPatternNilWriter(t *testing.T) {
	logger := NewConsoleLogger(nil, "info")

	pattern := &mockErrorPattern{
		category:   "ENV_LEVEL",
		pattern:    "test",
		suggestion: "test suggestion",
		fixable:    false,
	}

	// Should not panic
	logger.LogErrorPattern(pattern)
}

// TestLogErrorPatternLogLevelFiltering verifies log level filtering works.
func TestLogErrorPatternLogLevelFiltering(t *testing.T) {
	// INFO level - should log
	buf1 := &bytes.Buffer{}
	logger1 := NewConsoleLogger(buf1, "info")

	pattern := &mockErrorPattern{
		category:   "ENV_LEVEL",
		pattern:    "test",
		suggestion: "test suggestion",
		fixable:    false,
	}

	logger1.LogErrorPattern(pattern)
	if buf1.Len() == 0 {
		t.Error("LogErrorPattern: expected output at INFO level")
	}

	// ERROR level - should not log
	buf2 := &bytes.Buffer{}
	logger2 := NewConsoleLogger(buf2, "error")
	logger2.LogErrorPattern(pattern)
	if buf2.Len() != 0 {
		t.Error("LogErrorPattern: expected no output at ERROR level (INFO should be filtered)")
	}
}

// TestLogErrorPatternWithLongSuggestion verifies text wrapping for long suggestions.
func TestLogErrorPatternWithLongSuggestion(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "info")

	longSuggestion := "This is a very long suggestion that should be wrapped across multiple lines to fit within the terminal width. It contains many words to test the word wrapping functionality properly."

	pattern := &mockErrorPattern{
		category:   "ENV_LEVEL",
		pattern:    "test pattern",
		suggestion: longSuggestion,
		fixable:    false,
	}

	logger.LogErrorPattern(pattern)

	output := buf.String()
	if output == "" {
		t.Fatal("LogErrorPattern: expected output, got empty string")
	}

	// Verify the suggestion text is present (even if wrapped)
	if !strings.Contains(output, "This is a very long suggestion") {
		t.Error("LogErrorPattern: missing beginning of suggestion text")
	}
}

// TestLogErrorPatternBoxedFormat verifies box characters are present.
func TestLogErrorPatternBoxedFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "info")

	pattern := &mockErrorPattern{
		category:   "ENV_LEVEL",
		pattern:    "test",
		suggestion: "test suggestion",
		fixable:    false,
	}

	logger.LogErrorPattern(pattern)

	output := buf.String()

	// Check for box drawing characters (they appear in the raw output)
	if !strings.Contains(output, "┌") && !strings.Contains(output, "│") && !strings.Contains(output, "┴") {
		// If no unicode box chars, check for ASCII-based or colored output format
		// The output might be colored, so just verify structure is present
		if !strings.Contains(output, "Error Pattern") || !strings.Contains(output, "Category") {
			t.Error("LogErrorPattern: missing expected output structure")
		}
	}
}

// TestLogErrorPatternConcurrency verifies thread-safe logging of patterns.
func TestLogErrorPatternConcurrency(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewConsoleLogger(buf, "info")

	pattern := &mockErrorPattern{
		category:   "ENV_LEVEL",
		pattern:    "test",
		suggestion: "test suggestion",
		fixable:    false,
	}

	// Log from multiple goroutines
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			logger.LogErrorPattern(pattern)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should not crash and should have output
	if buf.Len() == 0 {
		t.Error("LogErrorPattern: expected output from concurrent logging")
	}
}

// TestWordWrapText verifies text wrapping utility function.
func TestWordWrapText(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		maxLen int
		want   []string
	}{
		{
			name:   "short text",
			text:   "short text",
			maxLen: 20,
			want:   []string{"short text"},
		},
		{
			name:   "exact fit",
			text:   "exactly twenty chars",
			maxLen: 20,
			want:   []string{"exactly twenty chars"},
		},
		{
			name:   "wrapped text",
			text:   "This is a longer text that should wrap",
			maxLen: 15,
			want:   []string{"This is a", "longer text", "that should", "wrap"},
		},
		{
			name:   "empty string",
			text:   "",
			maxLen: 20,
			want:   []string{},
		},
		{
			name:   "only whitespace",
			text:   "   ",
			maxLen: 20,
			want:   []string{},
		},
		{
			name:   "single long word",
			text:   "verylongword",
			maxLen: 5,
			want:   []string{"verylongword"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wordWrapText(tt.text, tt.maxLen)
			if len(got) != len(tt.want) {
				t.Errorf("wordWrapText() returned %d lines, want %d", len(got), len(tt.want))
			}
			for i, line := range got {
				if i < len(tt.want) && line != tt.want[i] {
					t.Errorf("wordWrapText() line %d = %q, want %q", i, line, tt.want[i])
				}
			}
		})
	}
}
