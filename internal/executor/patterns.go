package executor

import "regexp"

// ErrorCategory represents the type of error detected
type ErrorCategory int

const (
	CODE_LEVEL ErrorCategory = iota // Agent can fix with code changes
	PLAN_LEVEL                      // Plan file needs update
	ENV_LEVEL                       // Environment configuration issue
)

// String returns the string representation of ErrorCategory
func (ec ErrorCategory) String() string {
	switch ec {
	case CODE_LEVEL:
		return "CODE_LEVEL"
	case PLAN_LEVEL:
		return "PLAN_LEVEL"
	case ENV_LEVEL:
		return "ENV_LEVEL"
	default:
		return "UNKNOWN"
	}
}

// ErrorPattern defines a known error pattern with categorization
type ErrorPattern struct {
	Pattern                   string        // Regex pattern to match
	Category                  ErrorCategory // Category of error
	Suggestion                string        // Actionable suggestion for user
	AgentCanFix               bool          // Whether agent retry can fix
	RequiresHumanIntervention bool          // Whether manual intervention needed
}

// GetCategory returns the error category string (satisfies logger.ErrorPatternDisplay interface).
func (ep *ErrorPattern) GetCategory() string {
	return ep.Category.String()
}

// GetPattern returns the error pattern text (satisfies logger.ErrorPatternDisplay interface).
func (ep *ErrorPattern) GetPattern() string {
	return ep.Pattern
}

// GetSuggestion returns the actionable suggestion (satisfies logger.ErrorPatternDisplay interface).
func (ep *ErrorPattern) GetSuggestion() string {
	return ep.Suggestion
}

// IsAgentFixable returns whether the agent can fix this error (satisfies logger.ErrorPatternDisplay interface).
func (ep *ErrorPattern) IsAgentFixable() bool {
	return ep.AgentCanFix
}

// KnownPatterns is the comprehensive library of error patterns
var KnownPatterns = []ErrorPattern{
	// ENV_LEVEL patterns (4 patterns)
	{
		Pattern:                   "multiple devices matched",
		Category:                  ENV_LEVEL,
		Suggestion:                "Environment issue: Duplicate simulators. List: xcrun simctl list devices | grep '<name>', Delete: xcrun simctl delete <UUID>",
		AgentCanFix:               false,
		RequiresHumanIntervention: true,
	},
	{
		Pattern:                   "command not found",
		Category:                  ENV_LEVEL,
		Suggestion:                "Command not found in PATH. Install required tool or check PATH configuration.",
		AgentCanFix:               false,
		RequiresHumanIntervention: true,
	},
	{
		Pattern:                   "permission denied",
		Category:                  ENV_LEVEL,
		Suggestion:                "Permission issue. Check file/directory permissions or run with appropriate privileges.",
		AgentCanFix:               false,
		RequiresHumanIntervention: true,
	},
	{
		Pattern:                   "No space left on device",
		Category:                  ENV_LEVEL,
		Suggestion:                "Disk full. Free up space: du -sh * | sort -h, Clean temp files, Remove old Docker images.",
		AgentCanFix:               false,
		RequiresHumanIntervention: true,
	},

	// PLAN_LEVEL patterns (5 patterns)
	{
		Pattern:                   "no test bundles available|There are no test bundles",
		Category:                  PLAN_LEVEL,
		Suggestion:                "Test target missing from project. Update plan to: 1) Use existing test target, or 2) Add task to create test target.",
		AgentCanFix:               false,
		RequiresHumanIntervention: true,
	},
	{
		Pattern:                   "Tests in the target .* can't be run",
		Category:                  PLAN_LEVEL,
		Suggestion:                "Test target not in scheme. Update test_commands to use correct target or add scheme configuration task.",
		AgentCanFix:               false,
		RequiresHumanIntervention: true,
	},
	{
		Pattern:                   "No such file or directory.*test",
		Category:                  PLAN_LEVEL,
		Suggestion:                "Test file path incorrect. Verify file exists at specified path or update plan with correct path.",
		AgentCanFix:               false,
		RequiresHumanIntervention: true,
	},
	{
		Pattern:                   "scheme .* does not exist",
		Category:                  PLAN_LEVEL,
		Suggestion:                "Xcode scheme missing. List schemes: xcodebuild -list, Update plan with valid scheme name.",
		AgentCanFix:               false,
		RequiresHumanIntervention: true,
	},
	{
		Pattern:                   "Could not find test host|test host not found",
		Category:                  PLAN_LEVEL,
		Suggestion:                "Test host configuration missing. UI tests need app target as test host. Update scheme or plan.",
		AgentCanFix:               false,
		RequiresHumanIntervention: true,
	},

	// CODE_LEVEL patterns (4 patterns)
	{
		Pattern:                   "undefined: |not defined|cannot find symbol",
		Category:                  CODE_LEVEL,
		Suggestion:                "Missing import or undefined identifier. Agent should add import or define missing symbol.",
		AgentCanFix:               true,
		RequiresHumanIntervention: false,
	},
	{
		Pattern:                   "syntax error|unexpected token",
		Category:                  CODE_LEVEL,
		Suggestion:                "Syntax error in code. Agent should fix syntax.",
		AgentCanFix:               true,
		RequiresHumanIntervention: false,
	},
	{
		Pattern:                   "type mismatch|cannot convert",
		Category:                  CODE_LEVEL,
		Suggestion:                "Type error. Agent should fix type conversion or declaration.",
		AgentCanFix:               true,
		RequiresHumanIntervention: false,
	},
	{
		Pattern:                   "FAIL.*test.*failed",
		Category:                  CODE_LEVEL,
		Suggestion:                "Test assertion failed. Agent should fix implementation to pass test.",
		AgentCanFix:               true,
		RequiresHumanIntervention: false,
	},
}

// DetectErrorPattern analyzes output and returns matching pattern if found.
// Returns nil if no pattern matches or output is empty.
// Pattern priority: first match wins.
func DetectErrorPattern(output string) *ErrorPattern {
	if output == "" {
		return nil
	}

	for i := range KnownPatterns {
		pattern := &KnownPatterns[i]
		matched, err := regexp.MatchString(pattern.Pattern, output)
		if err != nil {
			// Invalid regex - skip this pattern
			continue
		}
		if matched {
			// Return a copy to avoid external modifications
			patternCopy := *pattern
			return &patternCopy
		}
	}
	return nil
}
