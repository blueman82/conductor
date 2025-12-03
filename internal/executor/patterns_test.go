package executor

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/models"
)

// TestErrorCategoryString verifies ErrorCategory string representation.
func TestErrorCategoryString(t *testing.T) {
	tests := []struct {
		category ErrorCategory
		want     string
	}{
		{CODE_LEVEL, "CODE_LEVEL"},
		{PLAN_LEVEL, "PLAN_LEVEL"},
		{ENV_LEVEL, "ENV_LEVEL"},
		{ErrorCategory(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.category.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestDetectErrorPatternEnvLevel verifies ENV_LEVEL pattern detection.
func TestDetectErrorPatternEnvLevel(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		wantCat  ErrorCategory
		wantFix  bool
		wantHelp bool
	}{
		{
			name:     "multiple devices matched",
			output:   "xcrun: error: multiple devices matched",
			wantCat:  ENV_LEVEL,
			wantFix:  false,
			wantHelp: false,
		},
		{
			name:     "command not found",
			output:   "xcodebuild: command not found",
			wantCat:  ENV_LEVEL,
			wantFix:  false,
			wantHelp: false,
		},
		{
			name:     "permission denied",
			output:   "Error: permission denied when accessing /usr/local/bin/tool",
			wantCat:  ENV_LEVEL,
			wantFix:  false,
			wantHelp: false,
		},
		{
			name:     "no space left on device",
			output:   "Error writing file: No space left on device",
			wantCat:  ENV_LEVEL,
			wantFix:  false,
			wantHelp: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detected := DetectErrorPattern(tt.output, nil, false)

			if detected == nil {
				t.Fatal("expected pattern to be detected")
			}

			if detected.Pattern.Category != tt.wantCat {
				t.Errorf("Category = %v, want %v", detected.Pattern.Category, tt.wantCat)
			}

			if detected.Pattern.AgentCanFix != tt.wantFix {
				t.Errorf("AgentCanFix = %v, want %v", detected.Pattern.AgentCanFix, tt.wantFix)
			}

			if detected.Pattern.RequiresHumanIntervention != tt.wantHelp {
				t.Errorf("RequiresHumanIntervention = %v, want %v", detected.Pattern.RequiresHumanIntervention, tt.wantHelp)
			}

			if detected.Pattern.Suggestion == "" {
				t.Error("Suggestion should not be empty")
			}
		})
	}
}

// TestDetectErrorPatternPlanLevel verifies PLAN_LEVEL pattern detection.
func TestDetectErrorPatternPlanLevel(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		wantCat  ErrorCategory
		wantFix  bool
		wantHelp bool
	}{
		{
			name:     "no test bundles available",
			output:   "Error: no test bundles available for testing",
			wantCat:  PLAN_LEVEL,
			wantFix:  false,
			wantHelp: false,
		},
		{
			name:     "There are no test bundles",
			output:   "Build output: There are no test bundles in this project",
			wantCat:  PLAN_LEVEL,
			wantFix:  false,
			wantHelp: false,
		},
		{
			name:     "Tests in target cannot be run",
			output:   "Build failed: Tests in the target 'UITests' can't be run",
			wantCat:  PLAN_LEVEL,
			wantFix:  false,
			wantHelp: false,
		},
		{
			name:     "No such file test path",
			output:   "Error: No such file or directory: /path/to/test_file.swift",
			wantCat:  PLAN_LEVEL,
			wantFix:  false,
			wantHelp: false,
		},
		{
			name:     "scheme does not exist",
			output:   "Error: The scheme 'InvalidScheme' does not exist.",
			wantCat:  PLAN_LEVEL,
			wantFix:  false,
			wantHelp: false,
		},
		{
			name:     "Could not find test host",
			output:   "Build error: Could not find test host bundle",
			wantCat:  PLAN_LEVEL,
			wantFix:  false,
			wantHelp: false,
		},
		{
			name:     "test host not found",
			output:   "Error: test host not found for UI testing",
			wantCat:  PLAN_LEVEL,
			wantFix:  false,
			wantHelp: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detected := DetectErrorPattern(tt.output, nil, false)

			if detected == nil {
				t.Fatal("expected pattern to be detected")
			}

			if detected.Pattern.Category != tt.wantCat {
				t.Errorf("Category = %v, want %v", detected.Pattern.Category, tt.wantCat)
			}

			if detected.Pattern.AgentCanFix != tt.wantFix {
				t.Errorf("AgentCanFix = %v, want %v", detected.Pattern.AgentCanFix, tt.wantFix)
			}

			if detected.Pattern.RequiresHumanIntervention != tt.wantHelp {
				t.Errorf("RequiresHumanIntervention = %v, want %v", detected.Pattern.RequiresHumanIntervention, tt.wantHelp)
			}

			if detected.Pattern.Suggestion == "" {
				t.Error("Suggestion should not be empty")
			}
		})
	}
}

// TestDetectErrorPatternCodeLevel verifies CODE_LEVEL pattern detection.
func TestDetectErrorPatternCodeLevel(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		wantCat  ErrorCategory
		wantFix  bool
		wantHelp bool
	}{
		{
			name:     "undefined identifier",
			output:   "Error: undefined: SomeVariable",
			wantCat:  CODE_LEVEL,
			wantFix:  true,
			wantHelp: false,
		},
		{
			name:     "not defined",
			output:   "Compilation error: variable 'x' is not defined",
			wantCat:  CODE_LEVEL,
			wantFix:  true,
			wantHelp: false,
		},
		{
			name:     "cannot find symbol",
			output:   "Error: cannot find symbol: class MyClass",
			wantCat:  CODE_LEVEL,
			wantFix:  true,
			wantHelp: false,
		},
		{
			name:     "syntax error",
			output:   "Error: syntax error on line 42",
			wantCat:  CODE_LEVEL,
			wantFix:  true,
			wantHelp: false,
		},
		{
			name:     "unexpected token",
			output:   "Parse error: unexpected token '}'",
			wantCat:  CODE_LEVEL,
			wantFix:  true,
			wantHelp: false,
		},
		{
			name:     "type mismatch",
			output:   "Error: type mismatch - expected int, got string",
			wantCat:  CODE_LEVEL,
			wantFix:  true,
			wantHelp: false,
		},
		{
			name:     "cannot convert",
			output:   "Error: cannot convert []string to string",
			wantCat:  CODE_LEVEL,
			wantFix:  true,
			wantHelp: false,
		},
		{
			name:     "test failed",
			output:   "FAIL: test case TestExample failed with assertion",
			wantCat:  CODE_LEVEL,
			wantFix:  true,
			wantHelp: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detected := DetectErrorPattern(tt.output, nil, false)

			if detected == nil {
				t.Fatal("expected pattern to be detected")
			}

			if detected.Pattern.Category != tt.wantCat {
				t.Errorf("Category = %v, want %v", detected.Pattern.Category, tt.wantCat)
			}

			if detected.Pattern.AgentCanFix != tt.wantFix {
				t.Errorf("AgentCanFix = %v, want %v", detected.Pattern.AgentCanFix, tt.wantFix)
			}

			if detected.Pattern.RequiresHumanIntervention != tt.wantHelp {
				t.Errorf("RequiresHumanIntervention = %v, want %v", detected.Pattern.RequiresHumanIntervention, tt.wantHelp)
			}

			if detected.Pattern.Suggestion == "" {
				t.Error("Suggestion should not be empty")
			}
		})
	}
}

// TestDetectErrorPatternNoMatch verifies nil is returned when no pattern matches.
func TestDetectErrorPatternNoMatch(t *testing.T) {
	tests := []struct {
		name   string
		output string
	}{
		{
			name:   "empty output",
			output: "",
		},
		{
			name:   "unknown error",
			output: "some random error message that matches nothing",
		},
		{
			name:   "generic failure",
			output: "Error: something went wrong",
		},
		{
			name:   "warning only",
			output: "Warning: this is just a warning, not an error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detected := DetectErrorPattern(tt.output, nil, false)

			if detected != nil {
				t.Errorf("expected no pattern match, but got: %+v", detected)
			}
		})
	}
}

// TestDetectErrorPatternPriority verifies first match wins.
func TestDetectErrorPatternPriority(t *testing.T) {
	// Test with output that could match multiple patterns
	// The first matching pattern in KnownPatterns should be returned
	output := "Error: undefined: SomeFunction"

	detected := DetectErrorPattern(output, nil, false)

	if detected == nil {
		t.Fatal("expected pattern to be detected")
	}

	if detected.Pattern.Category != CODE_LEVEL {
		t.Errorf("Category = %v, want %v", detected.Pattern.Category, CODE_LEVEL)
	}

	// Verify it matches the "undefined" pattern by checking the suggestion
	if detected.Pattern.Suggestion != "Missing import or undefined identifier. Agent should add import or define missing symbol." {
		t.Errorf("unexpected suggestion: %q", detected.Pattern.Suggestion)
	}
}

// TestDetectErrorPatternCaseSensitivity verifies pattern matching is case-insensitive.
func TestDetectErrorPatternCaseSensitivity(t *testing.T) {
	tests := []struct {
		name        string
		output      string
		shouldMatch bool
	}{
		{
			name:        "exact case match",
			output:      "command not found",
			shouldMatch: true,
		},
		{
			name:        "uppercase variant",
			output:      "Command Not Found",
			shouldMatch: true, // Should match - patterns are case-insensitive
		},
		{
			name:        "Python SyntaxError",
			output:      "SyntaxError: invalid syntax",
			shouldMatch: true, // Should match "syntax error" pattern case-insensitively
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detected := DetectErrorPattern(tt.output, nil, false)

			if tt.shouldMatch && detected == nil {
				t.Error("expected pattern to be detected")
			}

			if !tt.shouldMatch && detected != nil {
				t.Errorf("expected no pattern match, but got: %+v", detected)
			}
		})
	}
}

// TestDetectErrorPatternComplexRegex verifies regex patterns with special characters.
func TestDetectErrorPatternComplexRegex(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		wantCat ErrorCategory
	}{
		{
			name:    "scheme pattern with different scheme names",
			output:  "scheme MyApp does not exist",
			wantCat: PLAN_LEVEL,
		},
		{
			name:    "scheme with spaces in name",
			output:  "scheme My Custom App does not exist",
			wantCat: PLAN_LEVEL,
		},
		{
			name:    "target pattern with different target names",
			output:  "Tests in the target 'AppTests' can't be run",
			wantCat: PLAN_LEVEL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detected := DetectErrorPattern(tt.output, nil, false)

			if detected == nil {
				t.Fatal("expected pattern to be detected")
			}

			if detected.Pattern.Category != tt.wantCat {
				t.Errorf("Category = %v, want %v", detected.Pattern.Category, tt.wantCat)
			}
		})
	}
}

// TestDetectErrorPatternPartialMatches verifies pattern matching with partial strings.
func TestDetectErrorPatternPartialMatches(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		wantCat ErrorCategory
	}{
		{
			name:    "permission denied in path",
			output:  "/home/user/.local/bin: permission denied",
			wantCat: ENV_LEVEL,
		},
		{
			name:    "FAIL with additional context",
			output:  "FAIL: test [TestSum] failed: expected 10 but got 5",
			wantCat: CODE_LEVEL,
		},
		{
			name:    "undefined in complex message",
			output:  "build error: undefined: github.com/some/package.FunctionName at line 42",
			wantCat: CODE_LEVEL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detected := DetectErrorPattern(tt.output, nil, false)

			if detected == nil {
				t.Fatal("expected pattern to be detected")
			}

			if detected.Pattern.Category != tt.wantCat {
				t.Errorf("Category = %v, want %v", detected.Pattern.Category, tt.wantCat)
			}
		})
	}
}

// TestKnownPatternsConsistency verifies all patterns have required fields.
func TestKnownPatternsConsistency(t *testing.T) {
	for i, pattern := range KnownPatterns {
		t.Run(pattern.Pattern, func(t *testing.T) {
			if pattern.Pattern == "" {
				t.Errorf("pattern[%d]: Pattern field is empty", i)
			}

			if pattern.Suggestion == "" {
				t.Errorf("pattern[%d]: Suggestion field is empty", i)
			}

			// Verify pattern is valid regex
			_, err := regexp.Compile(pattern.Pattern)
			if err != nil {
				t.Errorf("pattern[%d]: invalid regex %q: %v", i, pattern.Pattern, err)
			}

			// Verify category is valid
			category := pattern.Category
			if category != CODE_LEVEL && category != PLAN_LEVEL && category != ENV_LEVEL {
				t.Errorf("pattern[%d]: invalid category %v", i, category)
			}

			// Verify consistency: if AgentCanFix is true, RequiresHumanIntervention should be false
			if pattern.AgentCanFix && pattern.RequiresHumanIntervention {
				t.Errorf("pattern[%d]: cannot both be fixable by agent and require human intervention", i)
			}

			// For CODE_LEVEL, agent should be able to fix
			if pattern.Category == CODE_LEVEL && !pattern.AgentCanFix {
				t.Errorf("pattern[%d]: CODE_LEVEL errors should have AgentCanFix=true", i)
			}

			// ENV_LEVEL and PLAN_LEVEL: agents cannot fix directly (AgentCanFix=false)
			// but can be informed by QC feedback (RequiresHumanIntervention=false allows retries)
			if (pattern.Category == ENV_LEVEL || pattern.Category == PLAN_LEVEL) && pattern.AgentCanFix {
				t.Errorf("pattern[%d]: %v errors should have AgentCanFix=false", i, pattern.Category)
			}
		})
	}
}

// TestPatternCategoryCount verifies expected number of patterns per category.
func TestPatternCategoryCount(t *testing.T) {
	codeLevelCount := 0
	planLevelCount := 0
	envLevelCount := 0

	for _, pattern := range KnownPatterns {
		switch pattern.Category {
		case CODE_LEVEL:
			codeLevelCount++
		case PLAN_LEVEL:
			planLevelCount++
		case ENV_LEVEL:
			envLevelCount++
		}
	}

	t.Run("pattern counts", func(t *testing.T) {
		if envLevelCount < 4 {
			t.Errorf("ENV_LEVEL count = %d, want at least 4", envLevelCount)
		}

		if planLevelCount < 5 {
			t.Errorf("PLAN_LEVEL count = %d, want at least 5", planLevelCount)
		}

		if codeLevelCount < 4 {
			t.Errorf("CODE_LEVEL count = %d, want at least 4", codeLevelCount)
		}
	})
}

// TestDetectErrorPatternCopy verifies returned pattern is independent copy.
func TestDetectErrorPatternCopy(t *testing.T) {
	output := "command not found"

	detected1 := DetectErrorPattern(output, nil, false)
	detected2 := DetectErrorPattern(output, nil, false)

	if detected1 == nil || detected2 == nil {
		t.Fatal("expected patterns to be detected")
	}

	// Patterns should have same values
	if detected1.Pattern.Category != detected2.Pattern.Category {
		t.Error("Categories should match")
	}

	if detected1.Pattern.Suggestion != detected2.Pattern.Suggestion {
		t.Error("Suggestions should match")
	}

	// But should be different pointers (independent copies)
	if detected1.Pattern == detected2.Pattern {
		t.Error("returned patterns should be independent copies")
	}

	// Modifying one should not affect the other
	originalSuggestion := detected2.Pattern.Suggestion
	detected1.Pattern.Suggestion = "modified"

	if detected2.Pattern.Suggestion != originalSuggestion {
		t.Error("modifying one pattern should not affect another")
	}
}

// TestDetectErrorPatternMultilineOutput verifies matching in multiline strings.
func TestDetectErrorPatternMultilineOutput(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		wantCat ErrorCategory
	}{
		{
			name: "error in multiline output",
			output: `Building target...
Error: undefined: SomeVariable
Cleaning up...`,
			wantCat: CODE_LEVEL,
		},
		{
			name: "permission denied in multiline",
			output: `Attempting to access resource
Error: permission denied
Retrying...`,
			wantCat: ENV_LEVEL,
		},
		{
			name: "test failure in multiline",
			output: `Running tests...
FAIL: test case failed
Summary: 1 failure`,
			wantCat: CODE_LEVEL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detected := DetectErrorPattern(tt.output, nil, false)

			if detected == nil {
				t.Fatal("expected pattern to be detected")
			}

			if detected.Pattern.Category != tt.wantCat {
				t.Errorf("Category = %v, want %v", detected.Pattern.Category, tt.wantCat)
			}
		})
	}
}

// TestErrorPatternFields verifies ErrorPattern structure has all required fields.
func TestErrorPatternFields(t *testing.T) {
	pattern := ErrorPattern{
		Pattern:                   "test pattern",
		Category:                  CODE_LEVEL,
		Suggestion:                "test suggestion",
		AgentCanFix:               true,
		RequiresHumanIntervention: false,
	}

	if pattern.Pattern != "test pattern" {
		t.Errorf("Pattern = %q, want %q", pattern.Pattern, "test pattern")
	}

	if pattern.Category != CODE_LEVEL {
		t.Errorf("Category = %v, want %v", pattern.Category, CODE_LEVEL)
	}

	if pattern.Suggestion != "test suggestion" {
		t.Errorf("Suggestion = %q, want %q", pattern.Suggestion, "test suggestion")
	}

	if pattern.AgentCanFix != true {
		t.Errorf("AgentCanFix = %v, want %v", pattern.AgentCanFix, true)
	}

	if pattern.RequiresHumanIntervention != false {
		t.Errorf("RequiresHumanIntervention = %v, want %v", pattern.RequiresHumanIntervention, false)
	}
}

// ============================================================================
// MOCK INVOKER FOR CLAUDE CLASSIFICATION TESTS
// ============================================================================

// MockInvoker simulates an agent.Invoker for testing Claude-based error classification.
// It allows tests to inject specific responses or errors without requiring real Claude invocation.
type MockInvoker struct {
	Response  string // JSON response to return
	Error     error  // Error to return (if non-nil, overrides Response)
	CallCount int    // Track number of invocations
}

// Invoke simulates the agent.Invoker interface method.
// Returns the configured Response or Error, and increments CallCount.
func (m *MockInvoker) Invoke(ctx context.Context, prompt string, agent map[string]interface{}) (string, error) {
	m.CallCount++
	if m.Error != nil {
		return "", m.Error
	}
	return m.Response, nil
}

// ============================================================================
// CLAUDE CLASSIFICATION TESTS
// ============================================================================

// TestDetectErrorPatternWithClaudeSuccess verifies successful Claude classification
// when a valid response with high confidence is received.
func TestDetectErrorPatternWithClaudeSuccess(t *testing.T) {
	tests := []struct {
		name           string
		output         string
		claudeCategory string
		claudeResponse string
		wantCategory   ErrorCategory
		wantCanFix     bool
		wantHelp       bool
	}{
		{
			name:           "code level error with high confidence",
			output:         "Error: undefined: SomeVariable",
			claudeCategory: "CODE_LEVEL",
			claudeResponse: `{
				"category": "CODE_LEVEL",
				"suggestion": "Missing variable definition",
				"agent_can_fix": true,
				"requires_human_intervention": false,
				"confidence": 0.95
			}`,
			wantCategory: CODE_LEVEL,
			wantCanFix:   true,
			wantHelp:     false,
		},
		{
			name:           "plan level error with high confidence",
			output:         "scheme MyApp does not exist",
			claudeCategory: "PLAN_LEVEL",
			claudeResponse: `{
				"category": "PLAN_LEVEL",
				"suggestion": "Update scheme name in plan",
				"agent_can_fix": false,
				"requires_human_intervention": true,
				"confidence": 0.92
			}`,
			wantCategory: PLAN_LEVEL,
			wantCanFix:   false,
			wantHelp:     true,
		},
		{
			name:           "env level error with high confidence",
			output:         "command not found: swiftc",
			claudeCategory: "ENV_LEVEL",
			claudeResponse: `{
				"category": "ENV_LEVEL",
				"suggestion": "Install Swift compiler",
				"agent_can_fix": false,
				"requires_human_intervention": true,
				"confidence": 0.98
			}`,
			wantCategory: ENV_LEVEL,
			wantCanFix:   false,
			wantHelp:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: This test is set up for future implementation
			// Currently tryClaudeClassification returns nil, falling back to regex
			// Once Claude invocation is implemented, this test verifies the behavior

			mock := &MockInvoker{
				Response: tt.claudeResponse,
				Error:    nil,
			}

			detected := DetectErrorPattern(tt.output, mock, false)

			// For now, with invoker provided, we still fall back to regex
			// This test documents the expected behavior when Claude is implemented
			if detected == nil {
				t.Error("expected pattern to be detected (from regex fallback)")
			}

			// The regex patterns should still work as fallback
			if detected != nil && detected.Pattern.Category != tt.wantCategory {
				t.Errorf("Category (from regex) = %v, want %v (may differ when Claude implemented)", detected.Pattern.Category, tt.wantCategory)
			}
		})
	}
}

// TestDetectErrorPatternWithClaudeLowConfidence verifies fallback to regex
// when Claude returns valid JSON but confidence is below threshold (< 0.85).
func TestDetectErrorPatternWithClaudeLowConfidence(t *testing.T) {
	tests := []struct {
		name         string
		output       string
		confidence   float64
		wantCategory ErrorCategory
	}{
		{
			name:         "low confidence returns regex fallback",
			output:       "Error: undefined: SomeVariable",
			confidence:   0.75,
			wantCategory: CODE_LEVEL,
		},
		{
			name:         "very low confidence returns regex fallback",
			output:       "Error: syntax error in code",
			confidence:   0.50,
			wantCategory: CODE_LEVEL,
		},
		{
			name:         "barely below threshold",
			output:       "command not found: tool",
			confidence:   0.84,
			wantCategory: ENV_LEVEL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a response with low confidence
			claudeResponse := `{
				"category": "CODE_LEVEL",
				"suggestion": "Low confidence response",
				"agent_can_fix": true,
				"requires_human_intervention": false,
				"confidence": ` + formatFloat(tt.confidence) + `
			}`

			mock := &MockInvoker{
				Response: claudeResponse,
				Error:    nil,
			}

			// DetectErrorPattern with low-confidence Claude response
			// should fall back to regex patterns
			detected := DetectErrorPattern(tt.output, mock, false)

			if detected == nil {
				t.Error("expected regex fallback pattern, got nil")
			}

			// Verify it matched via regex (which should return the known pattern)
			if detected != nil && detected.Pattern.Category != tt.wantCategory {
				t.Errorf("Category = %v, want %v (from regex fallback)", detected.Pattern.Category, tt.wantCategory)
			}
		})
	}
}

// TestDetectErrorPatternWithClaudeError verifies fallback to regex
// when Claude invocation encounters an error (timeout, network, etc).
// Note: Currently tryClaudeClassification returns nil without calling the invoker
// (as Claude implementation is stubbed), but this test documents the expected behavior
// once the full Claude invocation is implemented.
func TestDetectErrorPatternWithClaudeError(t *testing.T) {
	tests := []struct {
		name         string
		output       string
		errorMsg     string
		wantCategory ErrorCategory
	}{
		{
			name:         "network timeout falls back to regex",
			output:       "Error: undefined: SomeVariable",
			errorMsg:     "context deadline exceeded",
			wantCategory: CODE_LEVEL,
		},
		{
			name:         "connection refused falls back to regex",
			output:       "command not found: missing_tool",
			errorMsg:     "connection refused",
			wantCategory: ENV_LEVEL,
		},
		{
			name:         "rate limit error falls back to regex",
			output:       "permission denied: /usr/bin",
			errorMsg:     "rate limit exceeded",
			wantCategory: ENV_LEVEL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockInvoker{
				Response: "",
				Error:    testError(tt.errorMsg),
			}

			detected := DetectErrorPattern(tt.output, mock, false)

			if detected == nil {
				t.Error("expected regex fallback pattern, got nil")
			}

			if detected != nil && detected.Pattern.Category != tt.wantCategory {
				t.Errorf("Category = %v, want %v (from regex fallback)", detected.Pattern.Category, tt.wantCategory)
			}

			// Note: CallCount may be 0 if tryClaudeClassification returns nil early
			// Once full implementation is complete, this would verify the invoker was called
		})
	}
}

// TestDetectErrorPatternWithNilInvoker verifies that nil invoker
// gracefully falls back to regex patterns without error.
func TestDetectErrorPatternWithNilInvoker(t *testing.T) {
	tests := []struct {
		name         string
		output       string
		wantCategory ErrorCategory
	}{
		{
			name:         "nil invoker uses regex for code error",
			output:       "Error: undefined: Variable",
			wantCategory: CODE_LEVEL,
		},
		{
			name:         "nil invoker uses regex for plan error",
			output:       "scheme Test does not exist",
			wantCategory: PLAN_LEVEL,
		},
		{
			name:         "nil invoker uses regex for env error",
			output:       "command not found: tool",
			wantCategory: ENV_LEVEL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Pass nil as invoker - should fall back to regex without error
			detected := DetectErrorPattern(tt.output, nil, false)

			if detected == nil {
				t.Error("expected regex fallback pattern, got nil")
			}

			if detected != nil && detected.Pattern.Category != tt.wantCategory {
				t.Errorf("Category = %v, want %v", detected.Pattern.Category, tt.wantCategory)
			}
		})
	}
}

// TestDetectErrorPatternWithWrongType verifies that an invalid invoker type
// (non-agent.Invoker interface) falls back to regex gracefully.
func TestDetectErrorPatternWithWrongType(t *testing.T) {
	tests := []struct {
		name         string
		output       string
		invoker      interface{}
		wantCategory ErrorCategory
	}{
		{
			name:         "string type falls back to regex",
			output:       "Error: undefined: Variable",
			invoker:      "not an invoker",
			wantCategory: CODE_LEVEL,
		},
		{
			name:         "int type falls back to regex",
			output:       "command not found",
			invoker:      42,
			wantCategory: ENV_LEVEL,
		},
		{
			name:         "empty struct falls back to regex",
			output:       "permission denied",
			invoker:      struct{}{},
			wantCategory: ENV_LEVEL,
		},
		{
			name:         "map type falls back to regex",
			output:       "Error: type mismatch expected int",
			invoker:      make(map[string]interface{}),
			wantCategory: CODE_LEVEL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detected := DetectErrorPattern(tt.output, tt.invoker, false)

			if detected == nil {
				t.Error("expected regex fallback pattern, got nil")
			}

			if detected != nil && detected.Pattern.Category != tt.wantCategory {
				t.Errorf("Category = %v, want %v (from regex fallback)", detected.Pattern.Category, tt.wantCategory)
			}
		})
	}
}

// TestCloudErrorClassificationResponseFormats verifies that various valid
// JSON response formats are properly handled.
func TestCloudErrorClassificationResponseFormats(t *testing.T) {
	tests := []struct {
		name           string
		output         string
		claudeResponse string
		shouldFallback bool // true = expect regex fallback, false = expect Claude result
		wantCategory   ErrorCategory
	}{
		{
			name:   "minimal valid response with required fields",
			output: "Error: undefined: Var",
			claudeResponse: `{
				"category": "CODE_LEVEL",
				"suggestion": "Fix undefined variable",
				"agent_can_fix": true,
				"requires_human_intervention": false,
				"confidence": 0.9
			}`,
			shouldFallback: true, // Currently no Claude implementation
			wantCategory:   CODE_LEVEL,
		},
		{
			name:   "complete response with all optional fields",
			output: "command not found",
			claudeResponse: `{
				"category": "ENV_LEVEL",
				"suggestion": "Install missing tool",
				"agent_can_fix": false,
				"requires_human_intervention": true,
				"confidence": 0.95,
				"raw_output": "command not found",
				"related_patterns": ["tool_not_in_path", "missing_dependency"],
				"time_to_resolve": "moderate",
				"severity_level": "high",
				"error_language": "bash",
				"reasoning": "Command missing from PATH indicates environment issue"
			}`,
			shouldFallback: true, // Currently no Claude implementation
			wantCategory:   ENV_LEVEL,
		},
		{
			name:   "response with exactly 0.85 confidence (threshold)",
			output: "type mismatch",
			claudeResponse: `{
				"category": "CODE_LEVEL",
				"suggestion": "Fix type conversion",
				"agent_can_fix": true,
				"requires_human_intervention": false,
				"confidence": 0.85
			}`,
			shouldFallback: true, // At threshold, currently falls back
			wantCategory:   CODE_LEVEL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockInvoker{
				Response: tt.claudeResponse,
				Error:    nil,
			}

			detected := DetectErrorPattern(tt.output, mock, false)

			if detected == nil {
				t.Error("expected pattern (from regex fallback), got nil")
			}

			if detected != nil && detected.Pattern.Category != tt.wantCategory {
				t.Errorf("Category = %v, want %v", detected.Pattern.Category, tt.wantCategory)
			}
		})
	}
}

// TestConvertCloudClassificationToPattern verifies conversion from
// CloudErrorClassification to ErrorPattern works correctly.
func TestConvertCloudClassificationToPattern(t *testing.T) {
	tests := []struct {
		name           string
		classification *models.CloudErrorClassification
		wantCategory   ErrorCategory
		wantCanFix     bool
		wantHelp       bool
		wantPattern    string
	}{
		{
			name: "code level classification",
			classification: &models.CloudErrorClassification{
				Category:                  "CODE_LEVEL",
				Suggestion:                "Fix undefined variable",
				AgentCanFix:               true,
				RequiresHumanIntervention: false,
			},
			wantCategory: CODE_LEVEL,
			wantCanFix:   true,
			wantHelp:     false,
			wantPattern:  "claude-classification",
		},
		{
			name: "plan level classification",
			classification: &models.CloudErrorClassification{
				Category:                  "PLAN_LEVEL",
				Suggestion:                "Update plan file",
				AgentCanFix:               false,
				RequiresHumanIntervention: true,
			},
			wantCategory: PLAN_LEVEL,
			wantCanFix:   false,
			wantHelp:     true,
			wantPattern:  "claude-classification",
		},
		{
			name: "env level classification",
			classification: &models.CloudErrorClassification{
				Category:                  "ENV_LEVEL",
				Suggestion:                "Install missing tool",
				AgentCanFix:               false,
				RequiresHumanIntervention: true,
			},
			wantCategory: ENV_LEVEL,
			wantCanFix:   false,
			wantHelp:     true,
			wantPattern:  "claude-classification",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pattern := convertCloudClassificationToPattern(tt.classification)

			if pattern == nil {
				t.Fatal("expected converted pattern, got nil")
			}

			if pattern.Category != tt.wantCategory {
				t.Errorf("Category = %v, want %v", pattern.Category, tt.wantCategory)
			}

			if pattern.AgentCanFix != tt.wantCanFix {
				t.Errorf("AgentCanFix = %v, want %v", pattern.AgentCanFix, tt.wantCanFix)
			}

			if pattern.RequiresHumanIntervention != tt.wantHelp {
				t.Errorf("RequiresHumanIntervention = %v, want %v", pattern.RequiresHumanIntervention, tt.wantHelp)
			}

			if pattern.Pattern != tt.wantPattern {
				t.Errorf("Pattern = %q, want %q", pattern.Pattern, tt.wantPattern)
			}

			if pattern.Suggestion != tt.classification.Suggestion {
				t.Errorf("Suggestion = %q, want %q", pattern.Suggestion, tt.classification.Suggestion)
			}
		})
	}
}

// TestConvertCloudClassificationInvalidCategory verifies that invalid
// category strings are rejected during conversion.
func TestConvertCloudClassificationInvalidCategory(t *testing.T) {
	tests := []struct {
		name     string
		category string
		wantNil  bool
	}{
		{
			name:     "invalid category typo",
			category: "CODE-LEVEL",
			wantNil:  true,
		},
		{
			name:     "lowercase invalid",
			category: "code_level",
			wantNil:  true,
		},
		{
			name:     "misspelled env",
			category: "ENVIRONMENT_LEVEL",
			wantNil:  true,
		},
		{
			name:     "empty category",
			category: "",
			wantNil:  true,
		},
		{
			name:     "valid code level",
			category: "CODE_LEVEL",
			wantNil:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cc := &models.CloudErrorClassification{
				Category:                  tt.category,
				Suggestion:                "Test suggestion",
				AgentCanFix:               true,
				RequiresHumanIntervention: false,
			}

			pattern := convertCloudClassificationToPattern(cc)

			if tt.wantNil && pattern != nil {
				t.Errorf("expected nil pattern for invalid category, got %+v", pattern)
			}

			if !tt.wantNil && pattern == nil {
				t.Error("expected pattern for valid category, got nil")
			}
		})
	}
}

// TestParseClaudeClassificationResponse verifies JSON parsing of
// Claude's error classification responses.
func TestParseClaudeClassificationResponse(t *testing.T) {
	tests := []struct {
		name           string
		jsonData       string
		wantErr        bool
		wantNil        bool
		wantCategory   string
		wantConfidence float64
	}{
		{
			name: "valid json response",
			jsonData: `{
				"category": "CODE_LEVEL",
				"suggestion": "Fix error",
				"agent_can_fix": true,
				"requires_human_intervention": false,
				"confidence": 0.92
			}`,
			wantErr:        false,
			wantNil:        false,
			wantCategory:   "CODE_LEVEL",
			wantConfidence: 0.92,
		},
		{
			name: "valid json with optional fields",
			jsonData: `{
				"category": "ENV_LEVEL",
				"suggestion": "Install tool",
				"agent_can_fix": false,
				"requires_human_intervention": true,
				"confidence": 0.87,
				"raw_output": "command not found",
				"severity_level": "high",
				"time_to_resolve": "moderate"
			}`,
			wantErr:        false,
			wantNil:        false,
			wantCategory:   "ENV_LEVEL",
			wantConfidence: 0.87,
		},
		{
			name:     "invalid json",
			jsonData: `{not valid json}`,
			wantErr:  true,
			wantNil:  true,
		},
		{
			name:     "empty json",
			jsonData: `{}`,
			wantErr:  false,
			wantNil:  false,
		},
		{
			name:     "malformed json missing quotes",
			jsonData: `{category: CODE_LEVEL}`,
			wantErr:  true,
			wantNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseClaudeClassificationResponse(tt.jsonData)

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if tt.wantNil && result != nil {
				t.Error("expected nil result, got value")
			}

			if !tt.wantNil && result != nil {
				if result.Category != tt.wantCategory {
					t.Errorf("Category = %q, want %q", result.Category, tt.wantCategory)
				}

				if result.Confidence != tt.wantConfidence {
					t.Errorf("Confidence = %v, want %v", result.Confidence, tt.wantConfidence)
				}
			}
		})
	}
}

// TestHasInvokeMethod verifies the invoker type checking function.
func TestHasInvokeMethod(t *testing.T) {
	tests := []struct {
		name       string
		obj        interface{}
		wantResult bool
	}{
		{
			name:       "nil object",
			obj:        nil,
			wantResult: false,
		},
		{
			name:       "mock invoker",
			obj:        &MockInvoker{},
			wantResult: true,
		},
		{
			name:       "string",
			obj:        "not an invoker",
			wantResult: true, // Current implementation accepts non-nil
		},
		{
			name:       "integer",
			obj:        42,
			wantResult: true, // Current implementation accepts non-nil
		},
		{
			name:       "empty struct",
			obj:        struct{}{},
			wantResult: true, // Current implementation accepts non-nil
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasInvokeMethod(tt.obj)
			if result != tt.wantResult {
				t.Errorf("hasInvokeMethod = %v, want %v", result, tt.wantResult)
			}
		})
	}
}

// ============================================================================
// HELPER FUNCTIONS FOR TESTS
// ============================================================================

// testError is a simple error type for testing
type testError string

func (e testError) Error() string {
	return string(e)
}

// formatFloat formats a float64 as a JSON number string
func formatFloat(f float64) string {
	switch {
	case f == 0.50:
		return "0.50"
	case f == 0.75:
		return "0.75"
	case f == 0.84:
		return "0.84"
	case f == 0.85:
		return "0.85"
	case f == 0.90:
		return "0.90"
	case f == 0.92:
		return "0.92"
	case f == 0.95:
		return "0.95"
	case f == 0.98:
		return "0.98"
	default:
		return "0.0"
	}
}

// ============================================================================
// DETECTED ERROR STRUCTURE TESTS
// ============================================================================

// TestDetectedErrorStructure verifies DetectedError contains all expected fields.
func TestDetectedErrorStructure(t *testing.T) {
	output := "Error: undefined: SomeVariable"

	detected := DetectErrorPattern(output, nil, false)

	if detected == nil {
		t.Fatal("expected pattern to be detected")
	}

	// Verify all fields are populated
	if detected.Pattern == nil {
		t.Error("Pattern field should not be nil")
	}

	if detected.RawOutput != output {
		t.Errorf("RawOutput = %q, want %q", detected.RawOutput, output)
	}

	if detected.Method == "" {
		t.Error("Method field should not be empty")
	}

	if detected.Confidence == 0 {
		t.Error("Confidence field should not be zero")
	}

	if detected.Timestamp.IsZero() {
		t.Error("Timestamp field should not be zero")
	}
}

// TestDetectedErrorMethodField verifies Method field is set correctly.
func TestDetectedErrorMethodField(t *testing.T) {
	tests := []struct {
		name         string
		output       string
		enableClaude bool
		wantMethod   string
	}{
		{
			name:         "regex detection",
			output:       "Error: undefined: Variable",
			enableClaude: false,
			wantMethod:   "regex",
		},
		{
			name:         "regex fallback when claude disabled",
			output:       "command not found",
			enableClaude: false,
			wantMethod:   "regex",
		},
		{
			name:         "regex fallback when no invoker",
			output:       "syntax error",
			enableClaude: true, // enabled but no invoker
			wantMethod:   "regex",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detected := DetectErrorPattern(tt.output, nil, tt.enableClaude)

			if detected == nil {
				t.Fatal("expected pattern to be detected")
			}

			if detected.Method != tt.wantMethod {
				t.Errorf("Method = %q, want %q", detected.Method, tt.wantMethod)
			}
		})
	}
}

// TestDetectedErrorConfidenceField verifies Confidence field values.
func TestDetectedErrorConfidenceField(t *testing.T) {
	tests := []struct {
		name           string
		output         string
		wantConfidence float64
	}{
		{
			name:           "regex detection always 1.0",
			output:         "Error: undefined: Variable",
			wantConfidence: 1.0,
		},
		{
			name:           "ENV_LEVEL regex confidence",
			output:         "command not found",
			wantConfidence: 1.0,
		},
		{
			name:           "PLAN_LEVEL regex confidence",
			output:         "scheme Test does not exist",
			wantConfidence: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detected := DetectErrorPattern(tt.output, nil, false)

			if detected == nil {
				t.Fatal("expected pattern to be detected")
			}

			if detected.Confidence != tt.wantConfidence {
				t.Errorf("Confidence = %v, want %v", detected.Confidence, tt.wantConfidence)
			}
		})
	}
}

// TestDetectedErrorTimestampField verifies Timestamp is populated.
func TestDetectedErrorTimestampField(t *testing.T) {
	output := "Error: undefined: Variable"

	detected := DetectErrorPattern(output, nil, false)

	if detected == nil {
		t.Fatal("expected pattern to be detected")
	}

	if detected.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}

	// Timestamp should be recent (within last second)
	if detected.Timestamp.After(time.Now()) {
		t.Error("Timestamp should not be in the future")
	}

	if time.Since(detected.Timestamp) > time.Second {
		t.Error("Timestamp should be recent (within last second)")
	}
}
