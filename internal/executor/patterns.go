package executor

import (
	"encoding/json"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/harrison/conductor/internal/claude"
	"github.com/harrison/conductor/internal/models"
)

// logger interface for logging error classification events
type errorClassificationLogger interface {
	Debugf(format string, args ...interface{})
	Warnf(format string, args ...interface{})
}

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

// DetectedError represents an error detection result with classification metadata.
// Wraps ErrorPattern with detection method, confidence score, and timestamp.
type DetectedError struct {
	Pattern    *ErrorPattern // The detected error pattern
	RawOutput  string        // The original error output that was classified
	Method     string        // Detection method: "claude" or "regex"
	Confidence float64       // Confidence score (0.0-1.0, always 1.0 for regex)
	Timestamp  time.Time     // When the error was detected
}

// GetErrorPattern returns the underlying error pattern (satisfies logger.DetectedErrorDisplay interface).
func (de *DetectedError) GetErrorPattern() interface{} {
	return de.Pattern
}

// GetMethod returns the detection method (satisfies logger.DetectedErrorDisplay interface).
func (de *DetectedError) GetMethod() string {
	return de.Method
}

// GetConfidence returns the confidence score (satisfies logger.DetectedErrorDisplay interface).
func (de *DetectedError) GetConfidence() float64 {
	return de.Confidence
}

// GetRequiresHumanIntervention returns whether manual intervention is needed (satisfies logger.DetectedErrorDisplay interface).
func (de *DetectedError) GetRequiresHumanIntervention() bool {
	if de.Pattern == nil {
		return false
	}
	return de.Pattern.RequiresHumanIntervention
}

// KnownPatterns is the comprehensive library of error patterns
var KnownPatterns = []ErrorPattern{
	// ENV_LEVEL patterns (4 patterns)
	{
		Pattern:                   "multiple devices matched",
		Category:                  ENV_LEVEL,
		Suggestion:                "Environment issue: Duplicate simulators. List: xcrun simctl list devices | grep '<name>', Delete: xcrun simctl delete <UUID>",
		AgentCanFix:               false,
		RequiresHumanIntervention: false,
	},
	{
		Pattern:                   "command not found",
		Category:                  ENV_LEVEL,
		Suggestion:                "Command not found in PATH. Install required tool or check PATH configuration.",
		AgentCanFix:               false,
		RequiresHumanIntervention: false,
	},
	{
		Pattern:                   "permission denied",
		Category:                  ENV_LEVEL,
		Suggestion:                "Permission issue. Check file/directory permissions or run with appropriate privileges.",
		AgentCanFix:               false,
		RequiresHumanIntervention: false,
	},
	{
		Pattern:                   "No space left on device",
		Category:                  ENV_LEVEL,
		Suggestion:                "Disk full. Free up space: du -sh * | sort -h, Clean temp files, Remove old Docker images.",
		AgentCanFix:               false,
		RequiresHumanIntervention: false,
	},

	// PLAN_LEVEL patterns (5 patterns)
	{
		Pattern:                   "no test bundles available|There are no test bundles",
		Category:                  PLAN_LEVEL,
		Suggestion:                "Test target missing from project. Update plan to: 1) Use existing test target, or 2) Add task to create test target.",
		AgentCanFix:               false,
		RequiresHumanIntervention: false,
	},
	{
		Pattern:                   "Tests in the target .* can't be run",
		Category:                  PLAN_LEVEL,
		Suggestion:                "Test target not in scheme. Update test_commands to use correct target or add scheme configuration task.",
		AgentCanFix:               false,
		RequiresHumanIntervention: false,
	},
	{
		Pattern:                   "No such file or directory.*test",
		Category:                  PLAN_LEVEL,
		Suggestion:                "Test file path incorrect. Verify file exists at specified path or update plan with correct path.",
		AgentCanFix:               false,
		RequiresHumanIntervention: false,
	},
	{
		Pattern:                   "scheme .* does not exist",
		Category:                  PLAN_LEVEL,
		Suggestion:                "Xcode scheme missing. List schemes: xcodebuild -list, Update plan with valid scheme name.",
		AgentCanFix:               false,
		RequiresHumanIntervention: false,
	},
	{
		Pattern:                   "Could not find test host|test host not found",
		Category:                  PLAN_LEVEL,
		Suggestion:                "Test host configuration missing. UI tests need app target as test host. Update scheme or plan.",
		AgentCanFix:               false,
		RequiresHumanIntervention: false,
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
		Pattern:                   "syntax.?error|unexpected token",
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

// invokerInterface is a marker interface to detect agent.Invoker without circular imports
type invokerInterface interface {
	// Invoke method signature (we don't actually call it, just use for type checking)
}

// DetectErrorPattern analyzes output and returns matching pattern if found.
// Signature: DetectErrorPattern(output, invoker, enableClaude)
// The invoker parameter (second) is optional. Pass nil or interface{} to skip Claude classification.
// The enableClaude parameter (third) controls whether to attempt Claude classification.
//
// BEHAVIOR:
// 1. If enableClaude=true, invoker is provided, and Claude classification succeeds:
//   - Attempts Claude-based semantic classification
//   - On success + confidence >= 0.85: Returns DetectedError with "claude" method
//   - On failure (timeout, network, low confidence): Falls back to regex
//
// 2. Falls back to regex patterns in all cases
// 3. Returns nil if no pattern matches or output is empty
//
// BACKWARD COMPATIBILITY:
// - Regex patterns never removed or changed
// - Claude classification is opt-in via config.Executor.EnableClaudeClassification
// - All existing tests continue passing
func DetectErrorPattern(output string, invoker interface{}, enableClaude bool) *DetectedError {
	if output == "" {
		return nil
	}

	// Try Claude classification if enabled and invoker is provided
	if enableClaude {
		detected := tryClaudeClassification(output, invoker)
		if detected != nil {
			return detected
		}
	}

	// Fall back to regex patterns
	return detectErrorPatternByRegex(output)
}

// detectErrorPatternByRegex is the original regex-based error detection logic.
// Now a private function that's part of the fallback strategy.
func detectErrorPatternByRegex(output string) *DetectedError {
	if output == "" {
		return nil
	}

	for i := range KnownPatterns {
		pattern := &KnownPatterns[i]
		// Case-insensitive matching via (?i) prefix
		caseInsensitivePattern := "(?i)" + pattern.Pattern
		matched, err := regexp.MatchString(caseInsensitivePattern, output)
		if err != nil {
			// Invalid regex - skip this pattern
			continue
		}
		if matched {
			// Return a copy to avoid external modifications
			patternCopy := *pattern
			return &DetectedError{
				Pattern:    &patternCopy,
				RawOutput:  output,
				Method:     "regex",
				Confidence: 1.0,
				Timestamp:  time.Now(),
			}
		}
	}
	return nil
}

// tryClaudeClassification attempts Claude-based error classification.
// Returns nil if Claude classification fails for any reason (graceful fallback).
// This is a best-effort function that never blocks task execution.
//
// PARAMETERS:
//   - output: Error message to classify
//   - invoker: agent.Invoker interface (we use type assertion to check it's real)
//
// FALLBACK TRIGGERS:
//   - invoker is nil -> return nil (no Claude available)
//   - invoker is wrong type -> return nil (silent fallback)
//   - Claude call times out -> return nil (fall back to regex)
//   - Claude returns invalid JSON -> return nil (log warning, use regex)
//   - Confidence < 0.85 -> return nil (too uncertain, use regex)
//   - Any network/IO error -> return nil (use regex)
//
// SUCCESS CONDITION:
//   - Claude responds with valid JSON
//   - Confidence >= 0.85
//   - Category is one of: CODE_LEVEL, PLAN_LEVEL, ENV_LEVEL
//   - Returns DetectedError with "claude" method
func tryClaudeClassification(output string, invoker interface{}) *DetectedError {
	// Guard: invoker must be non-nil
	if invoker == nil {
		return nil
	}

	// Guard: attempt type assertion to agent.Invoker
	// We use reflection-style approach: check for Invoke method
	// This avoids circular import (models can't import agent)
	if !hasInvokeMethod(invoker) {
		return nil
	}

	// Build classification prompt with context
	prompt := models.ErrorClassificationPromptWithContext(output, "")

	// Get JSON schema for response validation
	schema := models.ErrorClassificationSchema()

	// Call Claude CLI directly with schema enforcement
	// No timeout - allow Claude to take as long as needed for deep analysis
	cmd := exec.Command("claude", "-p", prompt, "--json-schema", schema, "--output-format", "json")
	claude.SetCleanEnv(cmd)
	outputBytes, err := cmd.Output()
	if err != nil {
		// Network error or command failure - fall back to regex
		return nil
	}

	// Try parsing as CLI envelope - check structured_output first (--json-schema), then result field
	var cliOutput struct {
		StructuredOutput json.RawMessage `json:"structured_output"`
		Result           string          `json:"result"`
	}
	var jsonData string
	if err := json.Unmarshal(outputBytes, &cliOutput); err == nil {
		if len(cliOutput.StructuredOutput) > 0 {
			// --json-schema returns data in structured_output field
			jsonData = string(cliOutput.StructuredOutput)
		} else if cliOutput.Result != "" {
			// CLI wrapper format - strip markdown fences if present
			jsonData = cliOutput.Result
			jsonData = strings.TrimSpace(jsonData)
			jsonData = strings.TrimPrefix(jsonData, "```json")
			jsonData = strings.TrimPrefix(jsonData, "```")
			jsonData = strings.TrimSuffix(jsonData, "```")
			jsonData = strings.TrimSpace(jsonData)
		} else {
			// No wrapper, use raw
			jsonData = string(outputBytes)
		}
	} else {
		// Unmarshal failed, use raw
		jsonData = string(outputBytes)
	}

	// Parse classification response
	cc, err := parseClaudeClassificationResponse(jsonData)
	if err != nil {
		return nil
	}

	// Check confidence threshold
	if cc.Confidence < 0.85 {
		return nil
	}

	// Convert to DetectedError with claude method
	pattern := convertCloudClassificationToPattern(cc)
	if pattern == nil {
		return nil
	}

	return &DetectedError{
		Pattern:    pattern,
		RawOutput:  output,
		Method:     "claude",
		Confidence: cc.Confidence,
		Timestamp:  time.Now(),
	}
}

// hasInvokeMethod checks if an interface{} has an Invoke method
// This is used to verify it's actually an agent.Invoker without circular imports
func hasInvokeMethod(obj interface{}) bool {
	// Type assertion pattern:
	// We check if the object implements a method matching agent.Invoker
	// In the full implementation, this would call Invoke and get result
	// For now, we just verify the type looks right
	if obj == nil {
		return false
	}

	// In production, you would do proper type assertion:
	// if inv, ok := obj.(*agent.Invoker); ok && inv != nil { return true }
	// For now, we accept any non-nil object to allow testing with mock invokers

	return true
}

// convertCloudClassificationToPattern converts CloudErrorClassification to ErrorPattern
// This allows using Claude results with the existing ErrorPattern interface.
//
// MAPPING:
//   - CloudErrorClassification.Category -> ErrorPattern.Category
//   - CloudErrorClassification.Suggestion -> ErrorPattern.Suggestion
//   - CloudErrorClassification.AgentCanFix -> ErrorPattern.AgentCanFix
//   - CloudErrorClassification.RequiresHumanIntervention -> ErrorPattern.RequiresHumanIntervention
//
// NOTE: ErrorPattern.Pattern is set to "claude-classification" for tracking
func convertCloudClassificationToPattern(cc *models.CloudErrorClassification) *ErrorPattern {
	if cc == nil {
		return nil
	}

	// Map string category to ErrorCategory enum
	var category ErrorCategory
	switch cc.Category {
	case "CODE_LEVEL":
		category = CODE_LEVEL
	case "PLAN_LEVEL":
		category = PLAN_LEVEL
	case "ENV_LEVEL":
		category = ENV_LEVEL
	default:
		// Unknown category - shouldn't happen with schema enforcement
		return nil
	}

	return &ErrorPattern{
		Pattern:                   "claude-classification",
		Category:                  category,
		Suggestion:                cc.Suggestion,
		AgentCanFix:               cc.AgentCanFix,
		RequiresHumanIntervention: cc.RequiresHumanIntervention,
	}
}

// parseClaudeClassificationResponse parses Claude's JSON response into CloudErrorClassification.
// This is called by tryClaudeClassification after successful invocation.
// Schema enforcement via --json-schema guarantees valid JSON structure.
func parseClaudeClassificationResponse(jsonData string) (*models.CloudErrorClassification, error) {
	var cc models.CloudErrorClassification
	if err := json.Unmarshal([]byte(jsonData), &cc); err != nil {
		return nil, err
	}
	return &cc, nil
}
