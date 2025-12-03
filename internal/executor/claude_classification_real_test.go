package executor

import (
	"testing"
)

// TestClaudeClassificationRealCall tests the actual Claude classification with a real API call
// This test requires ANTHROPIC_API_KEY to be set and will make an actual Claude API call
// Skip in CI with: go test -short (this test won't run with -short flag)
func TestClaudeClassificationRealCall(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real Claude API test in short mode")
	}

	tests := []struct {
		name             string
		errorOutput      string
		expectedCategory ErrorCategory
	}{
		{
			name:             "python_syntax_error",
			errorOutput:      "File \"test.py\", line 5\n    def calculate(x y):\n                    ^\nSyntaxError: invalid syntax",
			expectedCategory: CODE_LEVEL,
		},
		{
			name:             "go_undefined_identifier",
			errorOutput:      "# github.com/harrison/conductor/internal/auth\ninternal/auth/handler.go:45:15: undefined: jwt\nFAIL    github.com/harrison/conductor/internal/auth [build failed]",
			expectedCategory: CODE_LEVEL,
		},
		{
			name:             "xcode_duplicate_simulators",
			errorOutput:      "xcodebuild: error: Unable to find a destination matching the provided destination specifier:\n  { platform:iOS Simulator, name:iPhone 15 Pro }\n\n  There are multiple devices matched: 'iPhone 15 Pro'\n    - iPhone 15 Pro (UDID: 12345678-ABCD-1234-ABCD-123456789ABC)\n    - iPhone 15 Pro (UDID: 87654321-DCBA-4321-DCBA-CBA987654321)",
			expectedCategory: ENV_LEVEL,
		},
		{
			name:             "xcode_scheme_missing",
			errorOutput:      "xcodebuild: error: Unable to find a scheme named 'IntegrationTests'\nAvailable schemes:\n  - AppScheme\n  - UnitTests\n  - E2ETests",
			expectedCategory: PLAN_LEVEL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock invoker (any non-nil object will pass hasInvokeMethod check)
			mockInvoker := &struct{}{}

			// Call DetectErrorPattern with real Claude invocation
			detected := DetectErrorPattern(tt.errorOutput, mockInvoker, true)

			if detected == nil {
				t.Fatal("Expected pattern from Claude classification, got nil (check ANTHROPIC_API_KEY and network)")
			}

			// Verify category matches expected
			if detected.Pattern.Category != tt.expectedCategory {
				t.Errorf("Category mismatch: got %v, want %v", detected.Pattern.Category, tt.expectedCategory)
			}

			// Verify we got Claude classification (not regex fallback)
			if detected.Pattern.Pattern != "claude-classification" {
				t.Errorf("Expected Claude classification, got regex pattern: %s", detected.Pattern.Pattern)
			}

			// Verify suggestion is non-empty
			if detected.Pattern.Suggestion == "" {
				t.Error("Expected non-empty suggestion from Claude")
			}

			// Log the result for inspection
			t.Logf("Claude Classification Result:")
			t.Logf("  Category: %v", detected.Pattern.Category)
			t.Logf("  Suggestion: %s", detected.Pattern.Suggestion)
			t.Logf("  AgentCanFix: %v", detected.Pattern.AgentCanFix)
			t.Logf("  RequiresHumanIntervention: %v", detected.Pattern.RequiresHumanIntervention)
		})
	}
}

// TestClaudeClassificationFallback verifies fallback to regex when Claude is unavailable
func TestClaudeClassificationFallback(t *testing.T) {
	// Pass nil invoker to trigger immediate fallback
	errorOutput := "SyntaxError: invalid syntax"
	detected := DetectErrorPattern(errorOutput, nil, false)

	if detected == nil {
		t.Fatal("Expected regex fallback pattern, got nil")
	}

	// Should be regex pattern, not Claude classification
	if detected.Pattern.Pattern == "claude-classification" {
		t.Error("Expected regex pattern, got Claude classification (should have fallen back)")
	}

	// Should still match CODE_LEVEL via regex
	if detected.Pattern.Category != CODE_LEVEL {
		t.Errorf("Expected CODE_LEVEL from regex fallback, got %v", detected.Pattern.Category)
	}
}

// TestClaudeClassificationTimeout verifies graceful fallback on timeout
// Note: This test manipulates timeout but is hard to test deterministically
// In production, 5s timeout in tryClaudeClassification should handle slow responses
func TestClaudeClassificationGracefulFallback(t *testing.T) {
	// Test that invalid invoker type falls back gracefully
	invalidInvoker := "not a real invoker"
	errorOutput := "undefined: jwt"

	detected := DetectErrorPattern(errorOutput, invalidInvoker, false)

	// Should fall back to regex
	if detected == nil {
		t.Fatal("Expected regex fallback, got nil")
	}

	if detected.Pattern.Pattern == "claude-classification" {
		t.Error("Expected regex fallback, got Claude classification")
	}
}
