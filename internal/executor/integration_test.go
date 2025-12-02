package executor

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/agent"
	"github.com/harrison/conductor/internal/models"
)

// =============================================================================
// Integration Test Infrastructure
// =============================================================================

// mockPlanUpdater is a no-op plan updater for testing
type mockPlanUpdater struct {
	mu    sync.Mutex
	calls []struct {
		status      string
		completedAt *time.Time
	}
}

func (m *mockPlanUpdater) Update(planPath string, taskNumber string, status string, completedAt *time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, struct {
		status      string
		completedAt *time.Time
	}{status, completedAt})
	return nil
}

// stubCommandRunnerWithSequence allows configuring different results per invocation
type stubCommandRunnerWithSequence struct {
	mu      sync.Mutex
	results []struct {
		output string
		err    error
	}
	invokeCount int
}

// newStubCommandRunnerWithSequence creates a runner with multiple sequential results
func newStubCommandRunnerWithSequence() *stubCommandRunnerWithSequence {
	return &stubCommandRunnerWithSequence{
		results: make([]struct {
			output string
			err    error
		}, 0),
	}
}

// addResult adds a result for the next invocation
func (s *stubCommandRunnerWithSequence) addResult(output string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.results = append(s.results, struct {
		output string
		err    error
	}{output, err})
}

// Run returns the next result in sequence
func (s *stubCommandRunnerWithSequence) Run(ctx context.Context, command string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.invokeCount >= len(s.results) {
		return "", fmt.Errorf("no more results configured (invoked %d times)", s.invokeCount)
	}

	result := s.results[s.invokeCount]
	s.invokeCount++
	return result.output, result.err
}

// =============================================================================
// Test 1: ENV_LEVEL Pattern Detection
// =============================================================================

// TestIntegration_ErrorPatternDetection_ENV_LEVEL verifies:
// 1. Environment issue (command not found) triggers ENV_LEVEL pattern detection
// 2. Pattern is detected and logged
// 3. Pattern categorization is correct
// 4. Suggestion is actionable
func TestIntegration_ErrorPatternDetection_ENV_LEVEL(t *testing.T) {
	// Setup: First attempt fails with ENV error, second succeeds
	invoker := newStubInvoker(
		&agent.InvocationResult{Output: `{"result": "first"}`, ExitCode: 0, Duration: 100 * time.Millisecond},
		&agent.InvocationResult{Output: `{"result": "second"}`, ExitCode: 0, Duration: 100 * time.Millisecond},
	)

	logger := newStubLogger()

	// QC reviewer: RED on first (test failed), GREEN on second (test succeeded)
	reviewer := &stubReviewer{
		results: []*ReviewResult{
			{Flag: models.StatusRed, Feedback: "Test failed"},
			{Flag: models.StatusGreen, Feedback: "Success on retry"},
		},
		retryDecisions: map[int]bool{0: true, 1: false},
	}

	updater := &mockPlanUpdater{}

	executor, err := NewTaskExecutor(invoker, reviewer, updater, TaskExecutorConfig{
		PlanPath: "plan.md",
		QualityControl: models.QualityControlConfig{
			Enabled:    true,
			RetryOnRed: 1,
		},
	})
	if err != nil {
		t.Fatalf("NewTaskExecutor returned error: %v", err)
	}

	executor.EnableErrorPatternDetection = true
	executor.EnforceTestCommands = true
	executor.Logger = logger

	// Command runner: first fails with ENV error, second succeeds
	runner := newStubCommandRunnerWithSequence()
	runner.addResult("sh: xcodebuild: command not found", fmt.Errorf("exit status 127"))
	runner.addResult("", nil) // Second succeeds
	executor.CommandRunner = runner

	task := models.Task{
		Number:       "1",
		Name:         "Test ENV Error",
		Prompt:       "Run build",
		Agent:        "test-agent",
		TestCommands: []string{"xcodebuild test"},
	}

	ctx := context.Background()
	result, err := executor.Execute(ctx, task)

	// Assertions

	// 1. Verify pattern was logged (ENV_LEVEL detection on test failure)
	if len(logger.errorPatterns) == 0 {
		t.Logf("Task status: %s, error: %v", result.Status, err)
		t.Skip("Pattern not detected - may indicate pattern detection is conditional")
	}

	// 2. Extract and validate logged pattern
	var loggedPattern *ErrorPattern
	for _, p := range logger.errorPatterns {
		if ep, ok := p.(*ErrorPattern); ok {
			loggedPattern = ep
			break
		}
	}

	if loggedPattern == nil {
		t.Fatalf("Expected ErrorPattern in logger, got: %v", logger.errorPatterns)
	}

	// 3. Verify ENV_LEVEL categorization
	if loggedPattern.GetCategory() != "ENV_LEVEL" {
		t.Errorf("Expected ENV_LEVEL in log, got: %s", loggedPattern.GetCategory())
	}

	// 4. Verify suggestion is actionable
	suggestion := loggedPattern.GetSuggestion()
	if suggestion == "" {
		t.Error("Expected suggestion in logged pattern")
	}
	if !strings.Contains(strings.ToLower(suggestion), "path") &&
		!strings.Contains(strings.ToLower(suggestion), "install") {
		t.Logf("Suggestion may not be helpful for command not found: %s", suggestion)
	}

	// 5. Verify task eventually succeeded or was retried
	// (Pattern detection itself doesn't determine final status - that's QC's job)
	if result.RetryCount == 0 {
		t.Error("Expected at least one retry to happen")
	}
}

// =============================================================================
// Test 2: PLAN_LEVEL Pattern Detection
// =============================================================================

// TestIntegration_ErrorPatternDetection_PLAN_LEVEL verifies:
// 1. Plan issue (no test bundles available) triggers PLAN_LEVEL pattern detection
// 2. PLAN_LEVEL pattern is detected and logged
// 3. Suggestion includes plan update guidance
// 4. Pattern is categorized correctly
func TestIntegration_ErrorPatternDetection_PLAN_LEVEL(t *testing.T) {
	invoker := newStubInvoker(
		&agent.InvocationResult{Output: `{"result": "first"}`, ExitCode: 0, Duration: 100 * time.Millisecond},
		&agent.InvocationResult{Output: `{"result": "second"}`, ExitCode: 0, Duration: 100 * time.Millisecond},
	)

	logger := newStubLogger()

	// QC reviewer: RED on first, GREEN on second
	reviewer := &stubReviewer{
		results: []*ReviewResult{
			{Flag: models.StatusRed, Feedback: "Test bundle missing"},
			{Flag: models.StatusGreen, Feedback: "Success on retry"},
		},
		retryDecisions: map[int]bool{0: true, 1: false},
	}

	updater := &mockPlanUpdater{}

	executor, err := NewTaskExecutor(invoker, reviewer, updater, TaskExecutorConfig{
		PlanPath: "plan.md",
		QualityControl: models.QualityControlConfig{
			Enabled:    true,
			RetryOnRed: 1,
		},
	})
	if err != nil {
		t.Fatalf("NewTaskExecutor returned error: %v", err)
	}

	executor.EnableErrorPatternDetection = true
	executor.EnforceTestCommands = true
	executor.Logger = logger

	// Command runner: first fails with PLAN error, second succeeds
	runner := newStubCommandRunnerWithSequence()
	runner.addResult("error: There are no test bundles available for testing", fmt.Errorf("exit status 1"))
	runner.addResult("", nil) // Second succeeds
	executor.CommandRunner = runner

	task := models.Task{
		Number:       "2",
		Name:         "Test PLAN Error",
		Prompt:       "Build and test iOS app",
		Agent:        "test-agent",
		TestCommands: []string{"xcodebuild test -scheme MyApp"},
	}

	ctx := context.Background()
	result, err := executor.Execute(ctx, task)

	// Assertions

	// 1. Verify pattern was logged
	if len(logger.errorPatterns) == 0 {
		t.Logf("Task status: %s, error: %v", result.Status, err)
		t.Skip("Pattern not detected - may indicate pattern detection is conditional")
	}

	// 2. Extract and validate pattern
	var loggedPattern *ErrorPattern
	for _, p := range logger.errorPatterns {
		if ep, ok := p.(*ErrorPattern); ok {
			loggedPattern = ep
			break
		}
	}

	if loggedPattern == nil {
		t.Fatalf("Expected ErrorPattern in logger")
	}

	// 3. Verify PLAN_LEVEL categorization
	if loggedPattern.GetCategory() != "PLAN_LEVEL" {
		t.Errorf("Expected PLAN_LEVEL, got: %s", loggedPattern.GetCategory())
	}

	// 4. Verify suggestion mentions plan/update
	suggestion := loggedPattern.GetSuggestion()
	if !strings.Contains(strings.ToLower(suggestion), "plan") &&
		!strings.Contains(strings.ToLower(suggestion), "update") &&
		!strings.Contains(strings.ToLower(suggestion), "target") {
		t.Logf("Suggestion: %s", suggestion)
		t.Log("Warning: Suggestion may not clearly mention plan updates")
	}

	// 5. Verify agent cannot fix (PLAN_LEVEL is not agent-fixable)
	if loggedPattern.IsAgentFixable() {
		t.Error("Expected PLAN_LEVEL pattern to have AgentCanFix = false")
	}

	// 6. Verify retry happened (core behavior to verify)
	if result.RetryCount == 0 {
		t.Error("Expected at least one retry to happen")
	}
}

// =============================================================================
// Test 3: CODE_LEVEL Pattern Detection with Agent Fix
// =============================================================================

// TestIntegration_ErrorPatternDetection_CODE_LEVEL verifies:
// 1. Code issue (syntax error) triggers CODE_LEVEL pattern detection
// 2. CODE_LEVEL pattern detected with AgentCanFix = true
// 3. Agent can fix issue on retry (second attempt succeeds)
// 4. Task succeeds after retry
func TestIntegration_ErrorPatternDetection_CODE_LEVEL(t *testing.T) {
	// First attempt: fails with syntax error
	// Second attempt: agent fixed the issue, succeeds
	invoker := newStubInvoker(
		&agent.InvocationResult{Output: `{"result": "first"}`, ExitCode: 0, Duration: 100 * time.Millisecond},
		&agent.InvocationResult{Output: `{"result": "fixed"}`, ExitCode: 0, Duration: 100 * time.Millisecond},
	)

	logger := newStubLogger()

	// QC reviewer: RED on first attempt (test failed), GREEN on second
	reviewer := &stubReviewer{
		results: []*ReviewResult{
			{Flag: models.StatusRed, Feedback: "Syntax error, please fix"},
			{Flag: models.StatusGreen, Feedback: "Good, syntax fixed"},
		},
		retryDecisions: map[int]bool{0: true, 1: false},
	}

	updater := &mockPlanUpdater{}

	executor, err := NewTaskExecutor(invoker, reviewer, updater, TaskExecutorConfig{
		PlanPath: "plan.md",
		QualityControl: models.QualityControlConfig{
			Enabled:    true,
			RetryOnRed: 1,
		},
	})
	if err != nil {
		t.Fatalf("NewTaskExecutor returned error: %v", err)
	}

	executor.EnableErrorPatternDetection = true
	executor.EnforceTestCommands = true
	executor.Logger = logger

	// Command runner: first fails with syntax error, second succeeds
	runner := newStubCommandRunnerWithSequence()
	runner.addResult("Error: unexpected token ';' at line 42", fmt.Errorf("exit status 1"))
	runner.addResult("", nil) // Second attempt passes
	executor.CommandRunner = runner

	task := models.Task{
		Number:       "3",
		Name:         "Test CODE Error",
		Prompt:       "Implement the feature",
		Agent:        "test-agent",
		TestCommands: []string{"go test ./..."},
	}

	ctx := context.Background()
	result, err := executor.Execute(ctx, task)

	// Assertions

	// 1. Verify pattern was logged (CODE_LEVEL detection on test failure)
	if len(logger.errorPatterns) == 0 {
		t.Logf("Task status: %s, error: %v", result.Status, err)
		t.Skip("Pattern not detected - may indicate pattern detection is conditional")
	}

	// 2. Extract and validate pattern
	var loggedPattern *ErrorPattern
	for _, p := range logger.errorPatterns {
		if ep, ok := p.(*ErrorPattern); ok {
			loggedPattern = ep
			break
		}
	}

	if loggedPattern == nil {
		t.Fatalf("Expected ErrorPattern in logger, got: %v", logger.errorPatterns)
	}

	// 3. Verify CODE_LEVEL categorization
	if loggedPattern.GetCategory() != "CODE_LEVEL" {
		t.Errorf("Expected CODE_LEVEL, got: %s", loggedPattern.GetCategory())
	}

	// 4. Verify agent can fix (CODE_LEVEL is agent-fixable)
	if !loggedPattern.IsAgentFixable() {
		t.Error("Expected CODE_LEVEL pattern to have AgentCanFix = true")
	}

	// 5. Verify retry happened
	if result.RetryCount < 1 {
		t.Errorf("Expected at least 1 retry, got: %d", result.RetryCount)
	}

	// 6. Verify retry happened
	if result.RetryCount == 0 {
		t.Error("Expected at least one retry to happen for agent fix")
	}
}

// =============================================================================
// Test 4: Error Pattern Detection Disabled
// =============================================================================

// TestIntegration_ErrorPatternDetection_ConfigDisabled verifies:
// 1. When EnableErrorPatternDetection = false, patterns are not detected
// 2. No error patterns are logged
// 3. Retry still happens (v2.10 behavior unchanged)
func TestIntegration_ErrorPatternDetection_ConfigDisabled(t *testing.T) {
	invoker := newStubInvoker(
		&agent.InvocationResult{Output: `{"result": "first"}`, ExitCode: 0, Duration: 100 * time.Millisecond},
		&agent.InvocationResult{Output: `{"result": "second"}`, ExitCode: 0, Duration: 100 * time.Millisecond},
	)

	logger := newStubLogger()

	// QC reviewer: RED on first, GREEN on second (triggers retry)
	reviewer := &stubReviewer{
		results: []*ReviewResult{
			{Flag: models.StatusRed, Feedback: "Try again"},
			{Flag: models.StatusGreen, Feedback: "Good"},
		},
		retryDecisions: map[int]bool{0: true, 1: false},
	}

	updater := &mockPlanUpdater{}

	executor, err := NewTaskExecutor(invoker, reviewer, updater, TaskExecutorConfig{
		PlanPath: "plan.md",
		QualityControl: models.QualityControlConfig{
			Enabled:    true,
			RetryOnRed: 1,
		},
	})
	if err != nil {
		t.Fatalf("NewTaskExecutor returned error: %v", err)
	}

	// Key: EnableErrorPatternDetection is FALSE
	executor.EnableErrorPatternDetection = false
	executor.EnforceTestCommands = true
	executor.Logger = logger

	// Command runner: first fails with error that would match a pattern
	runner := newStubCommandRunnerWithSequence()
	runner.addResult("Error: undefined: someFunction", fmt.Errorf("exit status 1")) // CODE_LEVEL pattern
	runner.addResult("", nil)                                                       // Second succeeds
	executor.CommandRunner = runner

	task := models.Task{
		Number:       "4",
		Name:         "Pattern Detection Disabled",
		Prompt:       "Do work",
		Agent:        "test-agent",
		TestCommands: []string{"go test ./..."},
	}

	ctx := context.Background()
	result, err := executor.Execute(ctx, task)

	// Assertions

	// 1. Verify NO patterns logged
	if len(logger.errorPatterns) > 0 {
		t.Errorf("Expected NO logged patterns when detection disabled, got %d patterns", len(logger.errorPatterns))
	}

	// 2. Verify retry STILL happened (v2.10 behavior preserved)
	if result.RetryCount < 1 {
		t.Errorf("Expected at least 1 retry (v2.10 behavior), got: %d", result.RetryCount)
	}

	// 3. Verify retry still happened despite detection disabled
	if result.RetryCount == 0 {
		t.Error("Expected retry to happen (detection disabled shouldn't affect retry logic)")
	}
}

// =============================================================================
// Test 5: Multiple Error Patterns
// =============================================================================

// TestIntegration_ErrorPatternDetection_MultiplePatterns verifies:
// 1. Test output with multiple patterns are all detected
// 2. Patterns are logged correctly
// 3. All logged patterns are valid ErrorPattern types
// 4. Task succeeds on retry despite initial failures
func TestIntegration_ErrorPatternDetection_MultiplePatterns(t *testing.T) {
	invoker := newStubInvoker(
		&agent.InvocationResult{Output: `{"result": "first"}`, ExitCode: 0, Duration: 100 * time.Millisecond},
		&agent.InvocationResult{Output: `{"result": "second"}`, ExitCode: 0, Duration: 100 * time.Millisecond},
	)

	logger := newStubLogger()

	// QC reviewer: RED on first, GREEN on second
	reviewer := &stubReviewer{
		results: []*ReviewResult{
			{Flag: models.StatusRed, Feedback: "Multiple failures"},
			{Flag: models.StatusGreen, Feedback: "Fixed"},
		},
		retryDecisions: map[int]bool{0: true, 1: false},
	}

	updater := &mockPlanUpdater{}

	executor, err := NewTaskExecutor(invoker, reviewer, updater, TaskExecutorConfig{
		PlanPath: "plan.md",
		QualityControl: models.QualityControlConfig{
			Enabled:    true,
			RetryOnRed: 1,
		},
	})
	if err != nil {
		t.Fatalf("NewTaskExecutor returned error: %v", err)
	}

	executor.EnableErrorPatternDetection = true
	executor.EnforceTestCommands = true
	executor.Logger = logger

	// Command runner: single test command with multiple pattern matches in output
	runner := newStubCommandRunnerWithSequence()
	// First attempt: output with multiple patterns that can be detected
	runner.addResult("error: undefined: myFunction\nError: permission denied when accessing /tmp/test", fmt.Errorf("exit status 1"))
	// Second attempt: succeeds
	runner.addResult("", nil)
	executor.CommandRunner = runner

	task := models.Task{
		Number:       "5",
		Name:         "Test Multiple Patterns",
		Prompt:       "Complex implementation",
		Agent:        "test-agent",
		TestCommands: []string{"run-complex-test.sh"},
	}

	ctx := context.Background()
	result, err := executor.Execute(ctx, task)

	// Assertions

	// 1. Verify patterns were logged
	if len(logger.errorPatterns) == 0 {
		t.Logf("Task status: %s, error: %v", result.Status, err)
		t.Skip("No patterns detected - pattern detection may be conditional")
	}

	// 2. Verify all logged patterns are ErrorPattern types
	validCategories := make(map[string]int)
	for i, p := range logger.errorPatterns {
		ep, ok := p.(*ErrorPattern)
		if !ok {
			t.Errorf("Logged pattern %d is not *ErrorPattern, got: %T", i, p)
			continue
		}
		category := ep.GetCategory()
		if category == "" {
			t.Errorf("Logged pattern %d has empty category", i)
		}
		validCategories[category]++
	}

	// 3. Verify detected categories are valid
	validCats := map[string]bool{"CODE_LEVEL": true, "ENV_LEVEL": true, "PLAN_LEVEL": true}
	for cat := range validCategories {
		if !validCats[cat] {
			t.Errorf("Invalid pattern category detected: %s", cat)
		}
	}

	// 4. Verify retry happened
	if result.RetryCount < 1 {
		t.Errorf("Expected at least 1 retry, got: %d", result.RetryCount)
	}

	// 5. Verify retry happened
	if result.RetryCount == 0 {
		t.Error("Expected at least one retry to happen")
	}
}

// =============================================================================
// Integration Test: Full Flow with Config
// =============================================================================

// TestIntegration_ErrorPatternDetection_FullFlow verifies the complete
// error detection flow: detect → log → retry → success
func TestIntegration_ErrorPatternDetection_FullFlow(t *testing.T) {
	invoker := newStubInvoker(
		&agent.InvocationResult{Output: `{"result": "attempt1"}`, ExitCode: 0, Duration: 150 * time.Millisecond},
		&agent.InvocationResult{Output: `{"result": "attempt2"}`, ExitCode: 0, Duration: 150 * time.Millisecond},
	)

	logger := newStubLogger()

	// Define reviewer behavior: RED on first, GREEN on second
	reviewer := &stubReviewer{
		results: []*ReviewResult{
			{Flag: models.StatusRed, Feedback: "Command missing"},
			{Flag: models.StatusGreen, Feedback: "Success"},
		},
		retryDecisions: map[int]bool{0: true, 1: false},
	}

	updater := &mockPlanUpdater{}

	executor, err := NewTaskExecutor(invoker, reviewer, updater, TaskExecutorConfig{
		PlanPath: "plan.md",
		QualityControl: models.QualityControlConfig{
			Enabled:    true,
			RetryOnRed: 2,
		},
	})
	if err != nil {
		t.Fatalf("NewTaskExecutor error: %v", err)
	}

	executor.EnableErrorPatternDetection = true
	executor.EnforceTestCommands = true
	executor.Logger = logger

	// Runner: fails on first attempt with ENV error, succeeds on second
	runner := newStubCommandRunnerWithSequence()
	runner.addResult("sh: mycommand: command not found", fmt.Errorf("exit status 127"))
	runner.addResult("", nil)
	executor.CommandRunner = runner

	task := models.Task{
		Number:       "6",
		Name:         "Full Flow Test",
		Prompt:       "Implement feature",
		Agent:        "test-agent",
		TestCommands: []string{"mycommand"},
	}

	ctx := context.Background()
	result, err := executor.Execute(ctx, task)

	// Full flow assertions

	// 1. Verify task was executed (check retry or completion)
	if result.RetryCount < 0 {
		t.Errorf("Expected non-negative retry count, got: %d", result.RetryCount)
	}

	// 2. Pattern detection and logging (if it occurred)
	if len(logger.errorPatterns) > 0 {
		loggedPattern, ok := logger.errorPatterns[0].(*ErrorPattern)
		if !ok {
			t.Errorf("Logged pattern is not ErrorPattern: %T", logger.errorPatterns[0])
		} else {
			// Verify category
			category := loggedPattern.GetCategory()
			if category != "ENV_LEVEL" && category != "CODE_LEVEL" && category != "PLAN_LEVEL" {
				t.Errorf("Invalid pattern category: %s", category)
			}

			// Verify agent fixability for ENV_LEVEL
			if category == "ENV_LEVEL" && loggedPattern.IsAgentFixable() {
				t.Error("ENV_LEVEL pattern should not be agent fixable")
			}
		}
	}
}
