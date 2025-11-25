package cmd

import (
	"testing"
	"time"

	"github.com/harrison/conductor/internal/learning"
)

func TestFormatErrorAnalysisTable(t *testing.T) {
	tests := []struct {
		name     string
		patterns []learning.ErrorPattern
		limit    int
		verify   func(string) bool
	}{
		{
			name:     "empty patterns",
			patterns: []learning.ErrorPattern{},
			limit:    20,
			verify: func(s string) bool {
				return contains(s, "No error data available")
			},
		},
		{
			name: "single tool error",
			patterns: []learning.ErrorPattern{
				{
					ErrorType:    "tool",
					ErrorMessage: "permission denied",
					Tool:         "Read",
					Count:        5,
					LastOccurred: time.Now().Add(-1 * time.Hour),
				},
			},
			limit: 20,
			verify: func(s string) bool {
				return contains(s, "tool") &&
					contains(s, "Read") &&
					contains(s, "permission denied") &&
					contains(s, "Error Pattern Analysis")
			},
		},
		{
			name: "multiple errors with summary",
			patterns: []learning.ErrorPattern{
				{
					ErrorType:    "tool",
					ErrorMessage: "connection timeout",
					Tool:         "Read",
					Count:        10,
					LastOccurred: time.Now().Add(-2 * time.Hour),
				},
				{
					ErrorType:    "bash",
					ErrorMessage: "exit code 1",
					Command:      "go test ./...",
					Count:        8,
					LastOccurred: time.Now().Add(-1 * time.Hour),
				},
				{
					ErrorType:     "file",
					ErrorMessage:  "file not found",
					OperationType: "read",
					FilePath:      "/missing/file.txt",
					Count:         3,
					LastOccurred:  time.Now().Add(-30 * time.Minute),
				},
			},
			limit: 3,
			verify: func(s string) bool {
				return contains(s, "Error Summary") &&
					contains(s, "Total Errors:") &&
					contains(s, "Tool Errors:") &&
					contains(s, "Bash Errors:") &&
					contains(s, "File Errors:")
			},
		},
		{
			name: "multiple errors with limit",
			patterns: []learning.ErrorPattern{
				{
					ErrorType:    "tool",
					ErrorMessage: "error1",
					Tool:         "Tool1",
					Count:        10,
					LastOccurred: time.Now(),
				},
				{
					ErrorType:    "bash",
					ErrorMessage: "error2",
					Command:      "cmd1",
					Count:        8,
					LastOccurred: time.Now(),
				},
				{
					ErrorType:    "file",
					ErrorMessage: "error3",
					FilePath:     "/file.txt",
					Count:        5,
					LastOccurred: time.Now(),
				},
			},
			limit: 2,
			verify: func(s string) bool {
				return contains(s, "Showing 2 of 3 errors")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatErrorAnalysisTable(tt.patterns, tt.limit)
			if !tt.verify(result) {
				t.Errorf("formatErrorAnalysisTable output verification failed\nGot:\n%s", result)
			}
		})
	}
}

func TestDisplayErrorAnalysis_MissingDatabase(t *testing.T) {
	// This test verifies that the function properly handles a missing database
	// In actual usage, this would fail due to missing config, which is expected
	err := DisplayErrorAnalysis("", 20)
	if err == nil {
		t.Error("Expected error for missing database, got nil")
	}
}

func TestErrorPatternTypes(t *testing.T) {
	tests := []struct {
		name     string
		pattern  learning.ErrorPattern
		expected string
	}{
		{
			name:     "tool error type",
			pattern:  learning.ErrorPattern{ErrorType: "tool", Tool: "Read"},
			expected: "tool",
		},
		{
			name:     "bash error type",
			pattern:  learning.ErrorPattern{ErrorType: "bash", Command: "ls"},
			expected: "bash",
		},
		{
			name:     "file error type",
			pattern:  learning.ErrorPattern{ErrorType: "file", FilePath: "/tmp/file"},
			expected: "file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.pattern.ErrorType != tt.expected {
				t.Errorf("Expected error type %s, got %s", tt.expected, tt.pattern.ErrorType)
			}
		})
	}
}
