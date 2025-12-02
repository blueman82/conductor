package models

import (
	"encoding/json"
)

// ============================================================================
// ERROR CLASSIFICATION ARCHITECTURE v3.0
// ============================================================================
//
// OVERVIEW:
// This module defines the Claude-based error pattern classification system
// that replaces the regex-based detection in internal/executor/patterns.go.
//
// The refactor addresses the brittleness of regex patterns by leveraging Claude's
// semantic understanding to classify errors into actionable categories:
// - CODE_LEVEL: Errors agents can fix with code changes
// - PLAN_LEVEL: Issues requiring plan file updates
// - ENV_LEVEL: Environment configuration problems
//
// ARCHITECTURAL DECISIONS:
//
// 1. DUAL STORAGE PATTERN (Backward Compatibility)
//    - Keep existing ErrorPattern struct in patterns.go for fallback/comparison
//    - Add CloudErrorClassification with extended fields (confidence, raw_output)
//    - Executor can use both simultaneously during transition period (v2.11 → v3.0)
//    - Design allows A/B testing and gradual rollout
//
// 2. INTERFACE-BASED INVOKER ACCEPTANCE (Dependency Inversion)
//    - DetectErrorPattern() accepts agent.Invoker interface, not concrete type
//    - Decouples error classification from specific invoker implementation
//    - Enables testing with mock invokers, future invoker variants
//    - Config controls whether Claude classification is enabled
//
// 3. FALLBACK STRATEGY (Graceful Degradation)
//    - On Claude invocation failure: fall back to regex patterns
//    - On network error: use cached classification results (TTL-based)
//    - On timeout: use lightweight regex as fallback
//    - Never block task execution due to classification failure
//
// 4. PERFORMANCE CONSIDERATIONS
//    - Cache classifications by error text hash (SHA256)
//    - TTL: 24h for same error message across different tasks
//    - Batch classification: collect 5 errors, classify in parallel
//    - Early exit: if error matches high-confidence regex, skip Claude invocation
//
// 5. CONFIGURATION (config.yaml)
//    executor:
//      error_classification:
//        enabled: true                              # Master toggle
//        use_claude: true                           # Use Claude vs regex
//        fallback_to_regex: true                    # Fallback strategy
//        cache_enabled: true                        # Cache classifications
//        cache_ttl_hours: 24                        # Cache expiration
//        confidence_threshold: 0.85                 # Min acceptable confidence
//        timeout_ms: 5000                           # Claude invocation timeout
//        batch_size: 5                              # Errors per batch classification
//        enable_fallback_metrics: true              # Track fallback usage
//
// ============================================================================

// ErrorClassificationRequest represents the input to Claude for error analysis
type ErrorClassificationRequest struct {
	ErrorOutput string   `json:"error_output"`      // Raw error text from test output
	Context     string   `json:"context,omitempty"` // Optional: task context, file types
	FileTypes   []string `json:"file_types,omitempty"` // File extensions being tested
	RecentErrors []string `json:"recent_errors,omitempty"` // Previous errors in session (for pattern)
}

// CloudErrorClassification represents Claude's structured error categorization
//
// DESIGN NOTES:
// - Extends ErrorPattern with confidence and raw_output for traceability
// - Confidence (0.0-1.0) enables threshold-based filtering
// - AgentCanFix and RequiresHumanIntervention are booleans (not enums) for clarity
// - RawOutput preserved for debugging and cache validation
// - RelatedPatterns array enables cross-pattern analysis in learning system
// - TimeToResolve provides agents with severity indicators
type CloudErrorClassification struct {
	// Core categorization (matches ErrorPattern interface)
	Category                  string   `json:"category"`                    // CODE_LEVEL, PLAN_LEVEL, ENV_LEVEL
	Suggestion                string   `json:"suggestion"`                  // Actionable guidance (1-2 sentences)
	AgentCanFix               bool     `json:"agent_can_fix"`               // Whether agent retry can resolve
	RequiresHumanIntervention bool     `json:"requires_human_intervention"` // If human action needed

	// Classification quality metrics
	Confidence float64 `json:"confidence"` // 0.0-1.0: Claude's certainty in classification

	// Traceability
	RawOutput string `json:"raw_output"` // The error text analyzed (for cache validation)

	// Extended analysis for smarter retry strategies
	RelatedPatterns []string `json:"related_patterns,omitempty"` // Similar known patterns
	TimeToResolve   string   `json:"time_to_resolve,omitempty"` // Estimated fix time: "quick", "moderate", "long"
	SeverityLevel   string   `json:"severity_level,omitempty"`   // "critical", "high", "medium", "low"

	// Multi-language support (emerging requirement)
	ErrorLanguage string `json:"error_language,omitempty"` // Detected: "english", "python", "go", "swift", etc.

	// LLM reasoning (for transparency)
	Reasoning string `json:"reasoning,omitempty"` // Brief explanation of classification logic
}

// ============================================================================
// RESPONSE SCHEMA FOR CLAUDE CLASSIFICATION
// ============================================================================

// ErrorClassificationSchema returns the JSON Schema that enforces Claude's
// error classification response structure. This schema is passed to claude CLI
// via --json-schema flag to guarantee valid, parseable responses.
//
// SCHEMA DESIGN:
// - Required fields: category, suggestion, agent_can_fix, requires_human_intervention, confidence
// - Optional fields: raw_output, related_patterns, time_to_resolve, severity_level
// - Enum constraints: category (3 values), severity_level (4 values), time_to_resolve (3 values)
// - Ranges: confidence (0.0-1.0), description length (max 500 chars)
//
// Schema enforcement eliminates:
// - Invalid category values (no typos like "CODE-LEVEL" or "codeLvl")
// - Missing required fields (category always present)
// - Type mismatches (confidence is number, not string)
// - Hallucinated fields (additionalProperties: false)
func ErrorClassificationSchema() string {
	schema := map[string]interface{}{
		"$schema":     "http://json-schema.org/draft-07/schema#",
		"title":       "Error Classification",
		"description": "Claude's structured analysis of error output for automated categorization and fixing",
		"type":        "object",
		"required":    []string{"category", "suggestion", "agent_can_fix", "requires_human_intervention", "confidence"},
		"properties": map[string]interface{}{
			"category": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"CODE_LEVEL", "PLAN_LEVEL", "ENV_LEVEL"},
				"description": "Error category: CODE_LEVEL (agent-fixable code changes), PLAN_LEVEL (plan file updates), ENV_LEVEL (environment config)",
			},
			"suggestion": map[string]interface{}{
				"type":        "string",
				"maxLength":   500,
				"description": "Actionable guidance for fixing this error (1-2 sentences, concrete steps)",
			},
			"agent_can_fix": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether the executing agent can fix this error with code changes and retry",
			},
			"requires_human_intervention": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether human/operator intervention is needed (e.g., ENV_LEVEL or complex PLAN_LEVEL issues)",
			},
			"confidence": map[string]interface{}{
				"type":        "number",
				"minimum":     0.0,
				"maximum":     1.0,
				"description": "Classification confidence score (0.0=uncertain, 1.0=certain)",
			},
			"raw_output": map[string]interface{}{
				"type":        "string",
				"description": "The error output text analyzed (for cache validation and traceability)",
			},
			"related_patterns": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "string",
				},
				"description": "Similar known error patterns (enables learning system analysis)",
			},
			"time_to_resolve": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"quick", "moderate", "long"},
				"description": "Estimated effort to resolve: quick (<5 min), moderate (5-30 min), long (>30 min)",
			},
			"severity_level": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"critical", "high", "medium", "low"},
				"description": "Error severity: critical (blocks all progress), high (blocks feature), medium (degraded), low (cosmetic)",
			},
			"error_language": map[string]interface{}{
				"type":        "string",
				"description": "Detected error source language: python, go, swift, typescript, java, etc.",
			},
			"reasoning": map[string]interface{}{
				"type":        "string",
				"maxLength":   300,
				"description": "Brief explanation of why this error was classified this way (for debugging/transparency)",
			},
		},
		"additionalProperties": false,
	}

	jsonBytes, _ := json.Marshal(schema)
	return string(jsonBytes)
}

// ============================================================================
// CLAUDE CLASSIFICATION AGENT PROMPT
// ============================================================================

// ErrorClassificationPrompt returns the system prompt for Claude when
// performing error pattern classification. This prompt:
// - Explains the three error categories with semantic clarity
// - Provides 5+ examples per category to anchor Claude's understanding
// - Includes edge cases and ambiguity resolution
// - Instructs Claude to respond in the structured JSON format
//
// PROMPT DESIGN PRINCIPLES:
// 1. Category definitions focus on OUTCOME (who fixes it) not CAUSE (what broke)
// 2. Examples span multiple languages/domains to improve generalization
// 3. Edge cases clarify boundaries between categories
// 4. JSON instructions are minimal (schema enforcement handles structure)
// 5. Examples show confidence scores to train Claude's calibration
//
// CONTENT STRUCTURE:
// 1. Role/context (what Claude is doing)
// 2. Three category definitions with intent and signals
// 3. Per-category examples (5 each, with diverse domains)
// 4. Edge cases and disambiguation rules
// 5. Instructions for JSON response format
// 6. Notes on confidence scoring
func ErrorClassificationPrompt() string {
	return `You are an expert error classifier for automated software testing and task execution.
Your role is to analyze error messages and classify them into actionable categories that help
determine how to fix the error and who should fix it.

## ERROR CATEGORIES

### 1. CODE_LEVEL
Errors that the executing agent can fix by modifying source code and retrying.
- Agent has source code available
- Fix requires code changes (logic, syntax, imports, types)
- Retry strategy: agent writes fix → reruns tests → validation

SIGNALS:
- Syntax errors, undefined identifiers, import errors
- Type mismatches, assertion failures
- Logic bugs (test failures with clear expected vs actual)
- Missing methods/functions agent can implement
- Compilation/build errors
- No external tool/permission/config changes needed

CODE_LEVEL EXAMPLES:
1. "undefined: parsedConfig" in Go code → Missing import or type definition
2. "FAIL: expected 42, got 41" in test output → Test assertion failure, logic bug
3. "SyntaxError: Unexpected token }" in TypeScript → Bracket mismatch
4. "cannot find symbol" in Java → Missing import or undefined variable
5. "NameError: name 'user' is not defined" in Python → Missing variable definition

---

### 2. PLAN_LEVEL
Errors that require updating the implementation plan file (e.g., YAML/MD) but not environment config.
- Issue is plan/configuration mismatch, not code quality
- Fix requires changing task dependencies, test commands, file paths, or scheme names
- Agent cannot fix because the issue is in the plan, not the code
- Requires human review of plan correctness

SIGNALS:
- Wrong test target, incorrect file paths in test commands
- Missing scheme/configuration referenced in plan
- Test host mismatch, build target unavailable
- File paths don't exist (but could if plan changed)
- Wrong test command format for the framework

PLAN_LEVEL EXAMPLES:
1. "scheme 'MyApp' does not exist" → Scheme name in plan is wrong, need to correct it
2. "No such file or directory: test/specs/missing_test.swift" → Path in test command is wrong
3. "Tests in target 'UnitTests' can't be run" → Test target not in scheme, plan needs update
4. "Could not find test host 'MyAppTests'" → Test host setup missing from plan
5. "no test bundles available" → Plan references wrong test target

---

### 3. ENV_LEVEL
Errors caused by environment configuration, system resources, or tool availability.
- NOT a code or plan problem
- Agent cannot fix without external intervention (install tools, free disk space, permissions)
- Requires human operator action or environment setup
- Retry without environment fix will fail identically

SIGNALS:
- Tool not found (command not found in PATH)
- Permission denied (file access, directory write)
- Resource exhaustion (disk full, memory exceeded)
- Duplicate/conflicting system state (multiple simulators, port already in use)
- Missing external dependencies, SDKs, or system libraries
- Network unavailability (if API is needed)

ENV_LEVEL EXAMPLES:
1. "command not found: swiftc" → Swift compiler not installed, need env setup
2. "permission denied: /Users/.../.git" → File permissions incorrect, need chmod
3. "No space left on device" → Disk full, need to clean up
4. "multiple devices matched the query" → Two simulators with same name, need to delete
5. "EACCES: permission denied, open '/tmp/test.log'" → Write permission issue, need sudo or change umask

---

## EDGE CASES & DISAMBIGUATION

### Ambiguous Errors
If an error could belong to multiple categories, use these tiebreakers:

1. **"Test target not found" (Could be PLAN or ENV)**
   - PLAN_LEVEL if: plan references non-existent target name, fix plan
   - ENV_LEVEL if: project file corrupted, Xcode not installed, fix env
   - Context: If error mentions scheme or xcodebuild, likely PLAN (project file config)

2. **"Compilation error" (Could be CODE or ENV)**
   - CODE_LEVEL if: clear syntax issue or missing import (agent can fix)
   - ENV_LEVEL if: SDK missing, wrong Xcode version, incompatible compiler
   - Context: If error mentions SDK path or compiler flags, likely ENV

3. **"File not found" (Could be CODE, PLAN, or ENV)**
   - CODE_LEVEL if: code has wrong path, agent should fix path in code
   - PLAN_LEVEL if: plan has wrong path in test_commands, fix plan
   - ENV_LEVEL if: dependency missing from system, file should exist in /usr/local/bin
   - Context: Look at context of file (test data file = PLAN, binary = ENV)

### Multiple Errors in Output
If output contains multiple different errors:
- Classify the FIRST blocking error (highest priority to fix)
- Mention related patterns in "related_patterns" array
- Suggest fixing primary error first

### Locale/Language Variations
- Some errors are translated (e.g., Python's locale-specific messages)
- Classify based on semantic meaning, not exact text
- Handle both "SyntaxError" and "Syntax error" formats

---

## CONFIDENCE SCORING

Provide a confidence score (0.0-1.0) reflecting your certainty:
- 1.0: Crystal clear (e.g., "SyntaxError: unexpected token")
- 0.9: Very confident (e.g., "command not found: xyz")
- 0.8: Confident (e.g., test failure with assertion output)
- 0.7: Moderately confident (e.g., ambiguous but leans toward one category)
- 0.6: Uncertain (e.g., context-dependent, could be 2 categories equally)
- <0.6: Don't classify if confidence is too low; suggest human review

---

## JSON RESPONSE FORMAT

Respond with ONLY valid JSON (no explanation, no markdown, no fences).

Example response:
{
  "category": "CODE_LEVEL",
  "suggestion": "Syntax error: missing closing brace on line 42. Add '}' before the semicolon.",
  "agent_can_fix": true,
  "requires_human_intervention": false,
  "confidence": 0.95,
  "raw_output": "SyntaxError: Unexpected token '}' on line 42",
  "severity_level": "high",
  "error_language": "typescript",
  "time_to_resolve": "quick",
  "reasoning": "Clear syntax error with line number. Agent can immediately fix."
}

---

## NOW CLASSIFY THIS ERROR:

[ERROR_OUTPUT_PLACEHOLDER]

Context (if available):
[CONTEXT_PLACEHOLDER]
`
}

// ErrorClassificationPromptWithContext builds the classification prompt with specific
// error output and task context injected. This is called by the executor before
// invoking Claude.
//
// PARAMETERS:
//   - errorOutput: Raw error text from test/build output
//   - context: Optional context (task name, file types, recent history)
//
// RETURNS:
//   - Complete prompt ready for Claude invocation
func ErrorClassificationPromptWithContext(errorOutput, context string) string {
	if context == "" {
		context = "(No additional context provided)"
	}
	prompt := ErrorClassificationPrompt()
	prompt = strings.ReplaceAll(prompt, "[ERROR_OUTPUT_PLACEHOLDER]", errorOutput)
	prompt = strings.ReplaceAll(prompt, "[CONTEXT_PLACEHOLDER]", context)
	return prompt
}

// ============================================================================
// AGENT INVOKER INTEGRATION INTERFACE
// ============================================================================

// CloudClassifier defines the interface for Claude-based error classification.
// This interface decouples error classification from specific invoker implementations,
// enabling:
// - Testing with mock classifiers
// - Alternative implementations (caching, batching, fallback)
// - Configuration-driven behavior
//
// DESIGN NOTES:
// - Accepts agent.Invoker interface (from internal/agent/invoker.go)
// - Returns both CloudErrorClassification and fallback ErrorPattern
// - Handles errors gracefully (never blocks task execution)
// - Supports caching and batching for performance
type CloudClassifier interface {
	// ClassifyError analyzes an error and returns Claude-based classification
	// with fallback to regex patterns if Claude invocation fails.
	//
	// PARAMETERS:
	//   - errorOutput: Raw error text from test output
	//   - context: Optional context (task name, files, history)
	//   - invoker: Agent invoker (from internal/agent/invoker.go)
	//
	// RETURNS:
	//   - classification: Structured error analysis from Claude
	//   - fallback: Regex-based classification (if Claude failed)
	//   - err: Non-critical errors (don't block execution)
	//
	// ERROR HANDLING STRATEGY:
	//   - Network error: use cached result (if available) or regex fallback
	//   - Timeout: immediately fall back to regex
	//   - Invalid response: fall back to regex, log warning
	//   - Empty confidence: reject classification, use regex
	//
	// PERFORMANCE:
	//   - First call: invoke Claude (5s timeout)
	//   - Cache hit: return cached result (24h TTL)
	//   - Fallback: regex patterns (<10ms)
	ClassifyError(errorOutput, context string, invoker interface{}) (*CloudErrorClassification, error)

	// ClassifyErrorWithFallback is a convenience method that always returns a result
	// (never errors). It falls back to regex patterns if Claude fails.
	//
	// USAGE:
	//   classification, _ := classifier.ClassifyErrorWithFallback(output, ctx, invoker)
	//   // Now classification is guaranteed non-nil
	ClassifyErrorWithFallback(errorOutput, context string, invoker interface{}) *CloudErrorClassification
}

// ============================================================================
// ARCHITECTURAL NOTES FOR INTEGRATION
// ============================================================================

/*

INTEGRATION POINTS:

1. EXECUTOR TASK.GO (Lines 858-882, current patterns.go location)
   CURRENT (v2.11):
     pattern := executor.DetectErrorPattern(output)
     if pattern != nil {
       logger.LogErrorPattern(pattern)
     }

   REFACTORED (v3.0, backward compatible):
     // Try Claude classification if enabled
     if config.ErrorClassification.UseClaude && classifier != nil {
       classification, _ := classifier.ClassifyErrorWithFallback(output, ctx, invoker)
       if classification.Confidence >= config.ErrorClassification.ConfidenceThreshold {
         logger.LogCloudClassification(classification)
         task.Metadata["error_classification"] = classification
         break // Use Claude result
       }
     }
     // Fallback to regex
     pattern := executor.DetectErrorPattern(output)
     if pattern != nil {
       logger.LogErrorPattern(pattern)
       task.Metadata["error_classification"] = pattern
     }

2. CONFIG (config.yaml) ADDITIONS:
   executor:
     error_classification:
       enabled: true
       use_claude: true
       fallback_to_regex: true
       cache_enabled: true
       cache_ttl_hours: 24
       confidence_threshold: 0.85
       timeout_ms: 5000

3. LEARNING SYSTEM INTEGRATION:
   - Store both regex and Claude classifications for A/B analysis
   - Track fallback frequency to identify regex limitations
   - Use confidence scores to improve agent selection
   - Analyze which classification method led to successful retry

4. TESTING STRATEGY:
   - Mock agent.Invoker for unit tests
   - Capture Claude responses in golden fixtures
   - Compare regex vs Claude classifications on real task failures
   - Gradual rollout: Enable for subset of agents/tasks first
   - Metrics: accuracy, latency, fallback rate, cost

*/

// Import statement reminder (add to file top):
// import "strings"
