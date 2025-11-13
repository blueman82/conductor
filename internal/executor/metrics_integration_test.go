package executor

import (
	"context"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/agent"
	"github.com/harrison/conductor/internal/learning"
	"github.com/harrison/conductor/internal/models"
)

func TestMetricsIntegration_RecordsPatternDetections(t *testing.T) {
	// Setup: Create metrics collector
	metrics := learning.NewPatternMetrics()

	// Setup: Create task executor with metrics
	invoker := newStubInvoker(
		&agent.InvocationResult{
			Output:   `{"content":"compilation error: build failed"}`,
			ExitCode: 0,
			Duration: 100 * time.Millisecond,
		},
	)

	reviewer := &stubReviewer{
		results: []*ReviewResult{
			{Flag: models.StatusRed, Feedback: "compilation error detected"},
		},
		retryDecisions: map[int]bool{0: false}, // Don't retry
	}

	executor, err := NewTaskExecutor(invoker, reviewer, &recordingUpdater{}, TaskExecutorConfig{
		QualityControl: models.QualityControlConfig{
			Enabled:     true,
			RetryOnRed:  1,
			ReviewAgent: "test-reviewer",
		},
	})
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}

	// Inject metrics
	executor.metrics = metrics

	// Execute task
	task := models.Task{
		Number: "1",
		Name:   "Test Task",
		Prompt: "Do something",
	}

	_, _ = executor.Execute(context.Background(), task)

	// Verify metrics recorded execution
	if metrics.GetDetectionRate() == 0.0 {
		t.Error("expected metrics to record execution, got 0.0 detection rate")
	}

	// Verify pattern was detected
	stats := metrics.GetPatternStats("compilation_error")
	if stats == nil {
		t.Fatal("expected compilation_error pattern to be detected")
	}

	if stats.DetectionCount != 1 {
		t.Errorf("expected DetectionCount = 1, got %d", stats.DetectionCount)
	}

	// Verify overall execution count
	allPatterns := metrics.GetAllPatterns()
	if len(allPatterns) == 0 {
		t.Error("expected at least one pattern to be detected")
	}
}

func TestMetricsIntegration_HandlesMultiplePatterns(t *testing.T) {
	metrics := learning.NewPatternMetrics()

	// First task: compilation error
	invoker1 := newStubInvoker(
		&agent.InvocationResult{
			Output:   `{"content":"build failed: syntax error"}`,
			ExitCode: 0,
			Duration: 100 * time.Millisecond,
		},
	)

	reviewer1 := &stubReviewer{
		results: []*ReviewResult{
			{Flag: models.StatusRed, Feedback: "compilation error detected"},
		},
		retryDecisions: map[int]bool{0: false},
	}

	executor1, _ := NewTaskExecutor(invoker1, reviewer1, &recordingUpdater{}, TaskExecutorConfig{
		QualityControl: models.QualityControlConfig{Enabled: true, RetryOnRed: 1},
	})
	executor1.metrics = metrics

	task1 := models.Task{Number: "1", Name: "Task 1", Prompt: "Do something"}
	_, _ = executor1.Execute(context.Background(), task1)

	// Second task: test failure
	invoker2 := newStubInvoker(
		&agent.InvocationResult{
			Output:   `{"content":"tests failed: assertion error"}`,
			ExitCode: 0,
			Duration: 100 * time.Millisecond,
		},
	)

	reviewer2 := &stubReviewer{
		results: []*ReviewResult{
			{Flag: models.StatusRed, Feedback: "test failure detected"},
		},
		retryDecisions: map[int]bool{0: false},
	}

	executor2, _ := NewTaskExecutor(invoker2, reviewer2, &recordingUpdater{}, TaskExecutorConfig{
		QualityControl: models.QualityControlConfig{Enabled: true, RetryOnRed: 1},
	})
	executor2.metrics = metrics

	task2 := models.Task{Number: "2", Name: "Task 2", Prompt: "Do something"}
	_, _ = executor2.Execute(context.Background(), task2)

	// Verify both patterns detected
	// First task has "build failed: syntax error" which matches BOTH compilation_error AND syntax_error
	// Second task has "tests failed: assertion error" which matches test_failure
	// Total: 3 patterns across 2 executions
	allPatterns := metrics.GetAllPatterns()
	if len(allPatterns) != 3 {
		t.Errorf("expected 3 patterns (compilation_error, syntax_error, test_failure), got %d: %v", len(allPatterns), allPatterns)
	}

	compStats := metrics.GetPatternStats("compilation_error")
	if compStats == nil || compStats.DetectionCount != 1 {
		t.Error("expected compilation_error pattern to be detected once")
	}

	syntaxStats := metrics.GetPatternStats("syntax_error")
	if syntaxStats == nil || syntaxStats.DetectionCount != 1 {
		t.Error("expected syntax_error pattern to be detected once")
	}

	testStats := metrics.GetPatternStats("test_failure")
	if testStats == nil || testStats.DetectionCount != 1 {
		t.Error("expected test_failure pattern to be detected once")
	}

	// Verify detection rate (3 patterns across 2 executions = 150%)
	// This is expected because the first execution detected 2 patterns
	rate := metrics.GetDetectionRate()
	if rate != 1.5 {
		t.Errorf("expected detection rate = 1.5, got %.2f", rate)
	}
}

func TestMetricsIntegration_NilMetricsGracefulDegradation(t *testing.T) {
	// Create executor without metrics
	invoker := newStubInvoker(
		&agent.InvocationResult{
			Output:   `{"content":"compilation error: build failed"}`,
			ExitCode: 0,
			Duration: 100 * time.Millisecond,
		},
	)

	reviewer := &stubReviewer{
		results: []*ReviewResult{
			{Flag: models.StatusRed, Feedback: "compilation error detected"},
		},
		retryDecisions: map[int]bool{0: false},
	}

	executor, err := NewTaskExecutor(invoker, reviewer, &recordingUpdater{}, TaskExecutorConfig{
		QualityControl: models.QualityControlConfig{Enabled: true, RetryOnRed: 1},
	})
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}

	// executor.metrics is nil by default

	// Execute task - should not panic
	task := models.Task{Number: "1", Name: "Test Task", Prompt: "Do something"}
	_, err = executor.Execute(context.Background(), task)

	// Should complete without error (even though it failed QC)
	if err == nil {
		t.Error("expected QC failure error, got nil")
	}

	// Verify it's the expected QC error, not a panic
	if err != ErrQualityGateFailed {
		t.Errorf("expected ErrQualityGateFailed, got %v", err)
	}
}

func TestMetricsIntegration_GreenVerdictNoPatterns(t *testing.T) {
	metrics := learning.NewPatternMetrics()

	invoker := newStubInvoker(
		&agent.InvocationResult{
			Output:   `{"content":"task completed successfully"}`,
			ExitCode: 0,
			Duration: 100 * time.Millisecond,
		},
	)

	reviewer := &stubReviewer{
		results: []*ReviewResult{
			{Flag: models.StatusGreen, Feedback: "all good"},
		},
	}

	executor, _ := NewTaskExecutor(invoker, reviewer, &recordingUpdater{}, TaskExecutorConfig{
		QualityControl: models.QualityControlConfig{Enabled: true, RetryOnRed: 1},
	})
	executor.metrics = metrics

	task := models.Task{Number: "1", Name: "Test Task", Prompt: "Do something"}
	_, err := executor.Execute(context.Background(), task)

	if err != nil {
		t.Errorf("expected success, got error: %v", err)
	}

	// Verify execution recorded but no patterns (GREEN verdict)
	allPatterns := metrics.GetAllPatterns()
	if len(allPatterns) != 0 {
		t.Errorf("expected 0 patterns for GREEN verdict, got %d", len(allPatterns))
	}

	// Detection rate should be 0.0 (0 patterns / 1 execution)
	rate := metrics.GetDetectionRate()
	if rate != 0.0 {
		t.Errorf("expected detection rate = 0.0, got %.2f", rate)
	}
}
