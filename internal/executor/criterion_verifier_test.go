package executor

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/models"
)

// === Criterion Verifier Tests ===

func TestRunCriterionVerifications_AllPass(t *testing.T) {
	runner := NewFakeCommandRunner()
	runner.SetOutput("grep -q 'func NewRouter'", "")
	runner.SetOutput("test -f router.go", "")

	task := models.Task{
		Number: "1",
		Name:   "Test Task",
		StructuredCriteria: []models.SuccessCriterion{
			{
				Criterion: "Router function exists",
				Verification: &models.CriterionVerification{
					Command:     "grep -q 'func NewRouter'",
					Description: "Check router exists",
				},
			},
			{
				Criterion: "Router file exists",
				Verification: &models.CriterionVerification{
					Command:     "test -f router.go",
					Description: "Check file exists",
				},
			},
		},
	}

	results, err := RunCriterionVerifications(context.Background(), runner, task)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	for i, r := range results {
		if !r.Passed {
			t.Errorf("result[%d] expected passed=true", i)
		}
	}
}

func TestRunCriterionVerifications_PartialFailure(t *testing.T) {
	runner := NewFakeCommandRunner()
	runner.SetOutput("check1", "ok")
	runner.SetOutput("check2", "not found")
	runner.SetError("check2", errors.New("exit status 1"))

	task := models.Task{
		Number: "1",
		Name:   "Test Task",
		StructuredCriteria: []models.SuccessCriterion{
			{
				Criterion: "First criterion",
				Verification: &models.CriterionVerification{
					Command:     "check1",
					Description: "First check",
				},
			},
			{
				Criterion: "Second criterion",
				Verification: &models.CriterionVerification{
					Command:     "check2",
					Description: "Second check",
				},
			},
		},
	}

	// Criterion verification failures should NOT return error - they feed to QC
	results, err := RunCriterionVerifications(context.Background(), runner, task)
	if err != nil {
		t.Errorf("expected no error (failures feed to QC), got %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	// First should pass, second should fail
	if !results[0].Passed {
		t.Error("expected first result to pass")
	}
	if results[1].Passed {
		t.Error("expected second result to fail")
	}

	// Both commands should have executed (does NOT stop on failure)
	cmds := runner.Commands()
	if len(cmds) != 2 {
		t.Errorf("expected 2 commands (continues on failure), got %d", len(cmds))
	}
}

func TestRunCriterionVerifications_MixedCriteria(t *testing.T) {
	runner := NewFakeCommandRunner()
	runner.SetOutput("verify cmd", "ok")

	task := models.Task{
		Number: "1",
		Name:   "Test Task",
		StructuredCriteria: []models.SuccessCriterion{
			{
				Criterion:    "Has no verification",
				Verification: nil, // No verification block
			},
			{
				Criterion: "Has verification",
				Verification: &models.CriterionVerification{
					Command:     "verify cmd",
					Description: "Check something",
				},
			},
			{
				Criterion:    "Another without verification",
				Verification: nil,
			},
		},
	}

	results, err := RunCriterionVerifications(context.Background(), runner, task)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Only 1 result (for the one with verification)
	if len(results) != 1 {
		t.Errorf("expected 1 result (only criteria with verification), got %d", len(results))
	}

	// Only 1 command should have been executed
	if len(runner.Commands()) != 1 {
		t.Errorf("expected 1 command, got %d", len(runner.Commands()))
	}
}

func TestRunCriterionVerifications_EmptyCriteria(t *testing.T) {
	runner := NewFakeCommandRunner()

	task := models.Task{
		Number:             "1",
		Name:               "Test Task",
		StructuredCriteria: []models.SuccessCriterion{},
	}

	results, err := RunCriterionVerifications(context.Background(), runner, task)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestRunCriterionVerifications_NilCriteria(t *testing.T) {
	runner := NewFakeCommandRunner()

	task := models.Task{
		Number:             "1",
		Name:               "Test Task",
		StructuredCriteria: nil,
	}

	results, err := RunCriterionVerifications(context.Background(), runner, task)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if results != nil && len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestRunCriterionVerifications_ContextCancellation(t *testing.T) {
	runner := NewFakeCommandRunner()

	task := models.Task{
		Number: "1",
		Name:   "Test Task",
		StructuredCriteria: []models.SuccessCriterion{
			{
				Criterion: "Some criterion",
				Verification: &models.CriterionVerification{
					Command: "some cmd",
				},
			},
		},
	}

	// Create already cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := RunCriterionVerifications(ctx, runner, task)
	if err == nil {
		t.Error("expected error due to context cancellation")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got %v", err)
	}
}

func TestRunCriterionVerifications_ExpectedOutput(t *testing.T) {
	runner := NewFakeCommandRunner()
	runner.SetOutput("echo 42", "42")
	runner.SetOutput("echo 100", "100")

	task := models.Task{
		Number: "1",
		Name:   "Test Task",
		StructuredCriteria: []models.SuccessCriterion{
			{
				Criterion: "Value is 42",
				Verification: &models.CriterionVerification{
					Command:  "echo 42",
					Expected: "42",
				},
			},
			{
				Criterion: "Value is 50 (will fail)",
				Verification: &models.CriterionVerification{
					Command:  "echo 100",
					Expected: "50", // Mismatch
				},
			},
		},
	}

	results, err := RunCriterionVerifications(context.Background(), runner, task)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	// First should pass (expected matches)
	if !results[0].Passed {
		t.Error("expected first result to pass (expected matches)")
	}

	// Second should fail (expected mismatch)
	if results[1].Passed {
		t.Error("expected second result to fail (expected mismatch)")
	}
}

func TestCriterionVerificationResult_Fields(t *testing.T) {
	result := &CriterionVerificationResult{
		Index:       0,
		Criterion:   "Test criterion",
		Command:     "test cmd",
		Output:      "output",
		Expected:    "expected",
		Error:       nil,
		Passed:      true,
		Duration:    50 * time.Millisecond,
		Description: "Test description",
	}

	if result.Index != 0 {
		t.Errorf("expected index 0, got %d", result.Index)
	}
	if result.Criterion != "Test criterion" {
		t.Errorf("expected criterion 'Test criterion', got %q", result.Criterion)
	}
	if !result.Passed {
		t.Error("expected passed=true")
	}
}

func TestFormatCriterionResults_AllPass(t *testing.T) {
	results := []CriterionVerificationResult{
		{Index: 0, Criterion: "Router exists", Passed: true},
		{Index: 1, Criterion: "Handler works", Passed: true},
	}

	formatted := FormatCriterionResults(results)

	if formatted == "" {
		t.Error("expected non-empty formatted string")
	}

	// Should contain pass markers
	if !containsString(formatted, "✅") && !containsString(formatted, "PASS") {
		t.Error("expected pass indicators in output")
	}
}

func TestFormatCriterionResults_WithFailure(t *testing.T) {
	results := []CriterionVerificationResult{
		{Index: 0, Criterion: "Router exists", Passed: true},
		{Index: 1, Criterion: "Handler works", Passed: false, Output: "error", Error: errors.New("failed")},
	}

	formatted := FormatCriterionResults(results)

	if formatted == "" {
		t.Error("expected non-empty formatted string")
	}

	// Should contain failure markers
	if !containsString(formatted, "❌") && !containsString(formatted, "FAIL") {
		t.Error("expected failure indicators in output")
	}
}

func TestFormatCriterionResults_Empty(t *testing.T) {
	results := []CriterionVerificationResult{}
	formatted := FormatCriterionResults(results)
	if formatted != "" {
		t.Errorf("expected empty string for empty results, got %q", formatted)
	}
}
