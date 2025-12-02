package executor

import (
	"regexp"
	"testing"
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
			wantHelp: true,
		},
		{
			name:     "command not found",
			output:   "xcodebuild: command not found",
			wantCat:  ENV_LEVEL,
			wantFix:  false,
			wantHelp: true,
		},
		{
			name:     "permission denied",
			output:   "Error: permission denied when accessing /usr/local/bin/tool",
			wantCat:  ENV_LEVEL,
			wantFix:  false,
			wantHelp: true,
		},
		{
			name:     "no space left on device",
			output:   "Error writing file: No space left on device",
			wantCat:  ENV_LEVEL,
			wantFix:  false,
			wantHelp: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pattern := DetectErrorPattern(tt.output)

			if pattern == nil {
				t.Fatal("expected pattern to be detected")
			}

			if pattern.Category != tt.wantCat {
				t.Errorf("Category = %v, want %v", pattern.Category, tt.wantCat)
			}

			if pattern.AgentCanFix != tt.wantFix {
				t.Errorf("AgentCanFix = %v, want %v", pattern.AgentCanFix, tt.wantFix)
			}

			if pattern.RequiresHumanIntervention != tt.wantHelp {
				t.Errorf("RequiresHumanIntervention = %v, want %v", pattern.RequiresHumanIntervention, tt.wantHelp)
			}

			if pattern.Suggestion == "" {
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
			wantHelp: true,
		},
		{
			name:     "There are no test bundles",
			output:   "Build output: There are no test bundles in this project",
			wantCat:  PLAN_LEVEL,
			wantFix:  false,
			wantHelp: true,
		},
		{
			name:     "Tests in target cannot be run",
			output:   "Build failed: Tests in the target 'UITests' can't be run",
			wantCat:  PLAN_LEVEL,
			wantFix:  false,
			wantHelp: true,
		},
		{
			name:     "No such file test path",
			output:   "Error: No such file or directory: /path/to/test_file.swift",
			wantCat:  PLAN_LEVEL,
			wantFix:  false,
			wantHelp: true,
		},
		{
			name:     "scheme does not exist",
			output:   "Error: The scheme 'InvalidScheme' does not exist.",
			wantCat:  PLAN_LEVEL,
			wantFix:  false,
			wantHelp: true,
		},
		{
			name:     "Could not find test host",
			output:   "Build error: Could not find test host bundle",
			wantCat:  PLAN_LEVEL,
			wantFix:  false,
			wantHelp: true,
		},
		{
			name:     "test host not found",
			output:   "Error: test host not found for UI testing",
			wantCat:  PLAN_LEVEL,
			wantFix:  false,
			wantHelp: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pattern := DetectErrorPattern(tt.output)

			if pattern == nil {
				t.Fatal("expected pattern to be detected")
			}

			if pattern.Category != tt.wantCat {
				t.Errorf("Category = %v, want %v", pattern.Category, tt.wantCat)
			}

			if pattern.AgentCanFix != tt.wantFix {
				t.Errorf("AgentCanFix = %v, want %v", pattern.AgentCanFix, tt.wantFix)
			}

			if pattern.RequiresHumanIntervention != tt.wantHelp {
				t.Errorf("RequiresHumanIntervention = %v, want %v", pattern.RequiresHumanIntervention, tt.wantHelp)
			}

			if pattern.Suggestion == "" {
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
			pattern := DetectErrorPattern(tt.output)

			if pattern == nil {
				t.Fatal("expected pattern to be detected")
			}

			if pattern.Category != tt.wantCat {
				t.Errorf("Category = %v, want %v", pattern.Category, tt.wantCat)
			}

			if pattern.AgentCanFix != tt.wantFix {
				t.Errorf("AgentCanFix = %v, want %v", pattern.AgentCanFix, tt.wantFix)
			}

			if pattern.RequiresHumanIntervention != tt.wantHelp {
				t.Errorf("RequiresHumanIntervention = %v, want %v", pattern.RequiresHumanIntervention, tt.wantHelp)
			}

			if pattern.Suggestion == "" {
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
			pattern := DetectErrorPattern(tt.output)

			if pattern != nil {
				t.Errorf("expected no pattern match, but got: %+v", pattern)
			}
		})
	}
}

// TestDetectErrorPatternPriority verifies first match wins.
func TestDetectErrorPatternPriority(t *testing.T) {
	// Test with output that could match multiple patterns
	// The first matching pattern in KnownPatterns should be returned
	output := "Error: undefined: SomeFunction"

	pattern := DetectErrorPattern(output)

	if pattern == nil {
		t.Fatal("expected pattern to be detected")
	}

	if pattern.Category != CODE_LEVEL {
		t.Errorf("Category = %v, want %v", pattern.Category, CODE_LEVEL)
	}

	// Verify it matches the "undefined" pattern by checking the suggestion
	if pattern.Suggestion != "Missing import or undefined identifier. Agent should add import or define missing symbol." {
		t.Errorf("unexpected suggestion: %q", pattern.Suggestion)
	}
}

// TestDetectErrorPatternCaseSensitivity verifies pattern matching is case-sensitive.
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
			shouldMatch: false, // Should not match - patterns are case-sensitive
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pattern := DetectErrorPattern(tt.output)

			if tt.shouldMatch && pattern == nil {
				t.Error("expected pattern to be detected")
			}

			if !tt.shouldMatch && pattern != nil {
				t.Errorf("expected no pattern match, but got: %+v", pattern)
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
			pattern := DetectErrorPattern(tt.output)

			if pattern == nil {
				t.Fatal("expected pattern to be detected")
			}

			if pattern.Category != tt.wantCat {
				t.Errorf("Category = %v, want %v", pattern.Category, tt.wantCat)
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
			pattern := DetectErrorPattern(tt.output)

			if pattern == nil {
				t.Fatal("expected pattern to be detected")
			}

			if pattern.Category != tt.wantCat {
				t.Errorf("Category = %v, want %v", pattern.Category, tt.wantCat)
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

			// For ENV_LEVEL and PLAN_LEVEL, require human intervention
			if (pattern.Category == ENV_LEVEL || pattern.Category == PLAN_LEVEL) && !pattern.RequiresHumanIntervention {
				t.Errorf("pattern[%d]: %v errors should have RequiresHumanIntervention=true", i, pattern.Category)
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

	pattern1 := DetectErrorPattern(output)
	pattern2 := DetectErrorPattern(output)

	if pattern1 == nil || pattern2 == nil {
		t.Fatal("expected patterns to be detected")
	}

	// Patterns should have same values
	if pattern1.Category != pattern2.Category {
		t.Error("Categories should match")
	}

	if pattern1.Suggestion != pattern2.Suggestion {
		t.Error("Suggestions should match")
	}

	// But should be different pointers (independent copies)
	if pattern1 == pattern2 {
		t.Error("returned patterns should be independent copies")
	}

	// Modifying one should not affect the other
	originalSuggestion := pattern2.Suggestion
	pattern1.Suggestion = "modified"

	if pattern2.Suggestion != originalSuggestion {
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
			pattern := DetectErrorPattern(tt.output)

			if pattern == nil {
				t.Fatal("expected pattern to be detected")
			}

			if pattern.Category != tt.wantCat {
				t.Errorf("Category = %v, want %v", pattern.Category, tt.wantCat)
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
