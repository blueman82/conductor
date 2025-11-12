# Implementation Plan: Adaptive Learning System (Part 2 - Integration & CLI)

**Continuation of**: plan-01-database-learning-foundation.md
**Tasks**: 9-25 (Hook Integration, CLI Commands, Orchestrator Integration, Testing)

## Tasks 9-12: Hook Integration Chain (chain-3)

These tasks integrate learning into the task executor through three hook points: pre-task (adapt agent/prompt), QC-review (capture patterns), and post-task (record execution).

---

## Task 9: Implement Pattern Extraction for QC Feedback

**Agent**: golang-pro
**File(s)**: `internal/learning/patterns.go`
**Depends on**: None
**Worktree Group**: chain-3
**Estimated time**: 45m

### What you're building

Pattern extraction logic that analyzes QC feedback and task output to automatically identify failure types (compilation_error, test_failure, dependency_missing, etc.). This runs during the QC-review hook to tag executions with patterns before storing them.

### Test First (TDD)

**Test file**: `internal/learning/patterns_test.go`

**Test structure**:
```go
// Test compilation error detection
func TestExtractPatterns_CompilationError(t *testing.T)
  - Input: "compilation failed: syntax error on line 42"
  - Expect: []string{"compilation_error"}

// Test multiple patterns
func TestExtractPatterns_MultiplePatterns(t *testing.T)
  - Input: "compilation failed, tests also failed"
  - Expect: []string{"compilation_error", "test_failure"}

// Test no patterns
func TestExtractPatterns_NoPatterns(t *testing.T)
  - Input: "task completed successfully"
  - Expect: []string{}

// Test case insensitivity
func TestExtractPatterns_CaseInsensitive(t *testing.T)
  - Input: "COMPILATION FAILED"
  - Expect: []string{"compilation_error"}
```

**Test specifics**:
- Mock: None (pure function)
- Use: Table-driven tests
- Assert: Correct patterns detected
- Edge cases: Empty input, multiple occurrences, overlapping patterns

**Example test skeleton**:
```go
package learning

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractPatterns(t *testing.T) {
	tests := []struct {
		name     string
		verdict  string
		feedback string
		output   string
		want     []string
	}{
		{
			name:     "compilation error",
			verdict:  "RED",
			feedback: "Code has compilation errors",
			output:   "syntax error on line 42",
			want:     []string{"compilation_error"},
		},
		{
			name:     "test failure",
			verdict:  "RED",
			feedback: "Tests failed",
			output:   "assertion failed: expected 5, got 3",
			want:     []string{"test_failure"},
		},
		{
			name:     "multiple patterns",
			verdict:  "RED",
			feedback: "compilation failed and tests failed",
			output:   "",
			want:     []string{"compilation_error", "test_failure"},
		},
		{
			name:     "no patterns",
			verdict:  "GREEN",
			feedback: "Task completed successfully",
			output:   "All tests passed",
			want:     []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractFailurePatterns(tt.verdict, tt.feedback, tt.output)
			assert.Equal(t, tt.want, got)
		})
	}
}
```

### Implementation

**Approach**:
Create pattern extraction with keyword matching. Use map of patterns to keywords for easy extension.

**File: internal/learning/patterns.go**:
```go
package learning

import "strings"

// Pattern keywords mapping
var patternKeywords = map[string][]string{
	"compilation_error": {
		"compilation failed",
		"syntax error",
		"cannot find symbol",
		"undefined reference",
		"parse error",
	},
	"test_failure": {
		"test failed",
		"assertion failed",
		"expected",
		"FAILED",
		"tests failed",
	},
	"dependency_missing": {
		"cannot import",
		"module not found",
		"package not found",
		"dependency not installed",
		"no such file or directory",
	},
	"permission_error": {
		"permission denied",
		"access denied",
		"forbidden",
	},
	"timeout": {
		"timeout",
		"timed out",
		"deadline exceeded",
	},
	"runtime_error": {
		"panic:",
		"exception",
		"segmentation fault",
		"null pointer",
		"nil pointer",
	},
}

// ExtractFailurePatterns analyzes QC feedback and output to identify failure types.
// Returns a list of detected patterns (e.g., "compilation_error", "test_failure").
func ExtractFailurePatterns(verdict, feedback, output string) []string {
	if verdict != "RED" && verdict != "YELLOW" {
		return []string{}
	}

	// Combine feedback and output for analysis
	searchText := strings.ToLower(feedback + " " + output)

	patterns := make([]string, 0)
	seen := make(map[string]bool)

	// Check each pattern's keywords
	for pattern, keywords := range patternKeywords {
		for _, keyword := range keywords {
			if strings.Contains(searchText, strings.ToLower(keyword)) {
				if !seen[pattern] {
					patterns = append(patterns, pattern)
					seen[pattern] = true
				}
				break // Found this pattern, check next pattern
			}
		}
	}

	return patterns
}
```

**Key points**:
- Pure function (no dependencies, easy to test)
- Case-insensitive matching
- Deduplication with `seen` map
- Only extracts patterns for RED/YELLOW verdicts
- Extensible via patternKeywords map

**Integration points**:
- Imports: `strings`
- Called by: QC-review hook
- Returns: String slice of patterns

### Verification

**Manual testing**:
1. Run tests: `go test ./internal/learning/ -v -run TestExtractPatterns`
2. Test manually:
```go
patterns := ExtractFailurePatterns("RED", "compilation failed", "syntax error")
fmt.Println(patterns) // Should print: [compilation_error]
```

**Automated tests**:
```bash
go test ./internal/learning/ -v -run TestExtractPatterns
```

**Expected output**:
```
=== RUN   TestExtractPatterns
=== RUN   TestExtractPatterns/compilation_error
=== RUN   TestExtractPatterns/test_failure
=== RUN   TestExtractPatterns/multiple_patterns
=== RUN   TestExtractPatterns/no_patterns
--- PASS: TestExtractPatterns (0.00s)
PASS
```

### Commit

**Commit message**:
```
feat(learning): add automatic pattern extraction

Implement ExtractFailurePatterns for QC output analysis:
- Keyword-based pattern detection
- Case-insensitive matching
- Detects 6 common failure types
- Pure function design for easy testing

Patterns: compilation_error, test_failure, dependency_missing,
permission_error, timeout, runtime_error.

Related to #adaptive-learning-system
```

**Files to commit**:
- `internal/learning/patterns.go`
- `internal/learning/patterns_test.go`

---

## Task 10: Add Pre-Task Hook for Agent Adaptation

**Agent**: golang-pro
**File(s)**: `internal/executor/task.go` (extend)
**Depends on**: Task 9
**Worktree Group**: chain-3
**Estimated time**: 1h 15m

### What you're building

The pre-task hook that queries the learning store, analyzes failures, and adapts the agent and prompt if patterns are detected. This runs before task execution and modifies the task based on learning. Includes prompt enhancement with learning context.

### Test First (TDD)

**Test file**: `internal/executor/task_test.go` (extend)

**Test structure**:
```go
// Test pre-task hook with no learning data
func TestTaskExecutor_PreTaskHook_NoLearning(t *testing.T)
  - Execute task with learning store but no prior executions
  - Verify agent unchanged
  - Verify prompt unchanged

// Test pre-task hook adapts agent after 2 failures
func TestTaskExecutor_PreTaskHook_AdaptsAgent(t *testing.T)
  - Mock learning store with 2 RED executions
  - Execute task with original agent "backend-developer"
  - Verify agent changed to "golang-pro" (suggested agent)
  - Verify prompt includes learning context

// Test pre-task hook enhances prompt
func TestTaskExecutor_PreTaskHook_EnhancesPrompt(t *testing.T)
  - Mock learning store with failure patterns
  - Execute task
  - Verify prompt contains "Learning from past attempts"
  - Verify prompt mentions common patterns

// Test pre-task hook respects nil learning store
func TestTaskExecutor_PreTaskHook_NilStore(t *testing.T)
  - Execute task with nil learning store
  - Verify no panics, task proceeds normally
```

**Test specifics**:
- Mock: LearningStore interface (create mock)
- Use: Existing task executor test patterns
- Assert: Agent adapted, prompt enhanced, no panics
- Edge cases: Nil store, failed analysis, no suggestions

**Example test skeleton**:
```go
// Create mock learning store interface
type mockLearningStore struct {
	analysisResult *learning.FailureAnalysis
	analysisError  error
}

func (m *mockLearningStore) AnalyzeFailures(ctx context.Context, planFile, taskNumber string) (*learning.FailureAnalysis, error) {
	if m.analysisError != nil {
		return nil, m.analysisError
	}
	if m.analysisResult == nil {
		return &learning.FailureAnalysis{}, nil
	}
	return m.analysisResult, nil
}

func (m *mockLearningStore) RecordExecution(ctx context.Context, exec *learning.TaskExecution) error {
	return nil
}

func TestTaskExecutor_PreTaskHook_AdaptsAgent(t *testing.T) {
	mockStore := &mockLearningStore{
		analysisResult: &learning.FailureAnalysis{
			TotalAttempts:           2,
			FailedAttempts:          2,
			TriedAgents:             []string{"backend-developer"},
			CommonPatterns:          []string{"compilation_error"},
			ShouldTryDifferentAgent: true,
			SuggestedAgent:          "golang-pro",
			SuggestedApproach:       "Focus on syntax validation",
		},
	}

	// Create task executor with learning
	executor := createTestExecutorWithLearning(t, mockStore)

	task := models.Task{
		Number: "Task 1",
		Name:   "Test Task",
		Agent:  "backend-developer",
		Prompt: "Implement feature",
	}

	result, err := executor.Execute(context.Background(), task)
	require.NoError(t, err)

	// Verify agent was adapted
	assert.Equal(t, "golang-pro", result.Task.Agent)

	// Verify prompt was enhanced
	assert.Contains(t, result.Task.Prompt, "Learning from past attempts")
	assert.Contains(t, result.Task.Prompt, "compilation_error")
}
```

### Implementation

**Approach**:
Add LearningStore field to TaskExecutor config, implement pre-task hook method, integrate into Execute flow. Follow existing hook pattern from QC review.

**Code structure** (modifications to internal/executor/task.go):
```go
// Add to TaskExecutorConfig
type TaskExecutorConfig struct {
	PlanPath       string
	DefaultAgent   string
	QualityControl models.QualityControlConfig
	LearningStore  LearningStore  // NEW
	SessionID      string          // NEW
	RunNumber      int             // NEW
}

// LearningStore interface for testing
type LearningStore interface {
	AnalyzeFailures(ctx context.Context, planFile, taskNumber string) (*learning.FailureAnalysis, error)
	RecordExecution(ctx context.Context, exec *learning.TaskExecution) error
}

// Add to DefaultTaskExecutor
type DefaultTaskExecutor struct {
	// ... existing fields ...
	learningStore LearningStore
	sessionID     string
	runNumber     int
}

// Update Execute method to call pre-task hook
func (te *DefaultTaskExecutor) Execute(ctx context.Context, task models.Task) (models.TaskResult, error) {
	result := models.TaskResult{Task: task}

	// ... existing locking code ...

	// PRE-TASK HOOK: Analyze learning and adapt
	if err := te.preTaskHook(ctx, &task); err != nil {
		// Log warning but don't fail execution
		// Learning failures shouldn't break conductor
		log.Warn("Pre-task hook failed: %v", err)
	}

	// ... rest of existing Execute logic ...
}

// preTaskHook analyzes past failures and adapts agent/prompt if needed
func (te *DefaultTaskExecutor) preTaskHook(ctx context.Context, task *models.Task) error {
	if te.learningStore == nil {
		return nil // No learning configured
	}

	analysis, err := te.learningStore.AnalyzeFailures(ctx, te.cfg.PlanPath, task.Number)
	if err != nil {
		return fmt.Errorf("analyze failures: %w", err)
	}

	// No failures, proceed normally
	if analysis.FailedAttempts == 0 {
		return nil
	}

	// Log learning insights
	log.Info("📚 Learning: Task %s has %d previous failed attempts", task.Number, analysis.FailedAttempts)

	if len(analysis.TriedAgents) > 0 {
		log.Warn("⚠️  Previous agents tried: %v", analysis.TriedAgents)
	}

	if len(analysis.CommonPatterns) > 0 {
		log.Warn("⚠️  Common patterns: %v", analysis.CommonPatterns)
	}

	// Adapt agent if threshold met
	if analysis.ShouldTryDifferentAgent {
		log.Info("💡 Suggested agent: %s", analysis.SuggestedAgent)
		log.Info("💡 Approach: %s", analysis.SuggestedApproach)

		// Override agent if specified agent has failed or no agent specified
		if task.Agent == "" || contains(analysis.TriedAgents, task.Agent) {
			oldAgent := task.Agent
			task.Agent = analysis.SuggestedAgent
			log.Info("🔄 Switching agent: %s → %s", oldAgent, task.Agent)
		}

		// Enhance prompt with learning context
		task.Prompt = enhancePromptWithLearning(task.Prompt, analysis)
	}

	return nil
}

// enhancePromptWithLearning adds learning context to task prompt
func enhancePromptWithLearning(originalPrompt string, analysis *learning.FailureAnalysis) string {
	enhanced := originalPrompt

	enhanced += "\n\n---\n**IMPORTANT - Learning from past attempts:**\n"
	enhanced += fmt.Sprintf("- This task has failed %d times previously\n", analysis.FailedAttempts)
	enhanced += fmt.Sprintf("- Previous agents tried: %v\n", analysis.TriedAgents)

	if len(analysis.CommonPatterns) > 0 {
		enhanced += "- Common failure patterns detected:\n"
		for _, pattern := range analysis.CommonPatterns {
			enhanced += fmt.Sprintf("  - %s\n", pattern)
		}
	}

	if analysis.SuggestedApproach != "" {
		enhanced += fmt.Sprintf("\n**Recommended approach:** %s\n", analysis.SuggestedApproach)
	}

	enhanced += "\nPlease take a different approach than what was tried before.\n"
	enhanced += "---\n"

	return enhanced
}

// Helper function
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
```

**Key points**:
- LearningStore is optional (nil check)
- Pre-task hook logs insights but doesn't fail execution
- Agent adaptation respects user intent (only overrides if agent failed before)
- Prompt enhancement is clear and structured
- Follows existing executor patterns

**Integration points**:
- Imports: `github.com/harrison/conductor/internal/learning`
- Config changes: Add fields to TaskExecutorConfig
- Logger: Use existing logger

### Verification

**Manual testing**:
1. Run tests: `go test ./internal/executor/ -v -run TestTaskExecutor_PreTaskHook`
2. Test with real learning store (integration test later)

**Automated tests**:
```bash
go test ./internal/executor/ -v -run TestTaskExecutor_PreTaskHook
```

**Expected output**:
```
=== RUN   TestTaskExecutor_PreTaskHook_NoLearning
--- PASS: TestTaskExecutor_PreTaskHook_NoLearning (0.00s)
=== RUN   TestTaskExecutor_PreTaskHook_AdaptsAgent
--- PASS: TestTaskExecutor_PreTaskHook_AdaptsAgent (0.01s)
PASS
```

### Commit

**Commit message**:
```
feat(executor): add pre-task learning hook for agent adaptation

Implement pre-task hook that adapts based on learning:
- Query failure history before task execution
- Switch agent if current agent has failed 2+ times
- Enhance prompt with learning context
- Log insights without breaking execution

LearningStore interface enables testing with mocks.

Related to #adaptive-learning-system
```

**Files to commit**:
- `internal/executor/task.go`
- `internal/executor/task_test.go`

---

*[Plan continues with remaining tasks 11-25, testing strategy, commit strategy, pitfalls, resources, and validation checklist in subsequent messages due to length]*

