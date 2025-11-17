package integration

import (
	"context"
	"errors"
	"path/filepath"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/executor"
	"github.com/harrison/conductor/internal/models"
	"github.com/harrison/conductor/internal/parser"
)

func TestEndToEndExecution_HappyPath(t *testing.T) {
	result, _, err := runE2EScenario(t, map[string]scriptedOutcome{
		"1": {status: models.StatusGreen, output: "bootstrap", duration: 10 * time.Millisecond},
		"2": {status: models.StatusGreen, output: "service", duration: 20 * time.Millisecond},
		"3": {status: models.StatusGreen, output: "validation", duration: 15 * time.Millisecond},
	})

	if err != nil {
		t.Fatalf("ExecutePlan() unexpected error: %v", err)
	}

	if result.TotalTasks != 3 {
		t.Fatalf("TotalTasks = %d, want 3", result.TotalTasks)
	}
	if result.Completed != 3 || result.Failed != 0 {
		t.Fatalf("Completed/Failed mismatch: got %d/%d, want 3/0", result.Completed, result.Failed)
	}
	if result.StatusBreakdown[models.StatusGreen] != 3 {
		t.Fatalf("GREEN count = %d, want 3", result.StatusBreakdown[models.StatusGreen])
	}
}

func TestEndToEndExecution_FailedPath(t *testing.T) {
	result, _, err := runE2EScenario(t, map[string]scriptedOutcome{
		"1": {status: models.StatusGreen, output: "bootstrap"},
		"2": {status: models.StatusGreen, output: "service"},
		"3": {status: models.StatusRed, output: "qc failure", err: executor.ErrQualityGateFailed},
	})

	if !errors.Is(err, executor.ErrQualityGateFailed) {
		t.Fatalf("ExecutePlan() error = %v, want ErrQualityGateFailed", err)
	}

	if result.TotalTasks != 3 {
		t.Fatalf("TotalTasks = %d, want 3", result.TotalTasks)
	}
	if result.Completed != 2 || result.Failed != 1 {
		t.Fatalf("Completed/Failed mismatch: got %d/%d, want 2/1", result.Completed, result.Failed)
	}
	if result.StatusBreakdown[models.StatusRed] != 1 {
		t.Fatalf("RED count = %d, want 1", result.StatusBreakdown[models.StatusRed])
	}
}

func TestEndToEndExecution_MixedVerdicts(t *testing.T) {
	result, _, err := runE2EScenario(t, map[string]scriptedOutcome{
		"1": {status: models.StatusGreen, output: "bootstrap"},
		"2": {status: models.StatusYellow, output: "service warning"},
		"3": {status: models.StatusRed, output: "qc failure", err: executor.ErrQualityGateFailed},
	})

	if !errors.Is(err, executor.ErrQualityGateFailed) {
		t.Fatalf("ExecutePlan() error = %v, want ErrQualityGateFailed", err)
	}

	if result.Completed != 2 || result.Failed != 1 {
		t.Fatalf("Completed/Failed mismatch: got %d/%d, want 2/1", result.Completed, result.Failed)
	}
	if result.StatusBreakdown[models.StatusGreen] != 1 {
		t.Fatalf("GREEN count = %d, want 1", result.StatusBreakdown[models.StatusGreen])
	}
	if result.StatusBreakdown[models.StatusYellow] != 1 {
		t.Fatalf("YELLOW count = %d, want 1", result.StatusBreakdown[models.StatusYellow])
	}
	if result.StatusBreakdown[models.StatusRed] != 1 {
		t.Fatalf("RED count = %d, want 1", result.StatusBreakdown[models.StatusRed])
	}
}

func TestEndToEndExecution_SkipResumeFlow(t *testing.T) {
	script := map[string]scriptedOutcome{
		"2": {status: models.StatusGreen, output: "service retry"},
		"3": {status: models.StatusGreen, output: "validation"},
	}

	result, exec, err := runE2EScenario(t, script,
		withPlan(filepath.Join("fixtures", "e2e-skip-plan.md")),
		withSkipCompleted(true),
		withRetryFailed(true),
	)

	if err != nil {
		t.Fatalf("ExecutePlan() error = %v", err)
	}

	if result.StatusBreakdown[models.StatusGreen] != 3 {
		t.Fatalf("expected all tasks GREEN, got %+v", result.StatusBreakdown)
	}

	executed := exec.ExecutedTasks()
	sort.Strings(executed)
	expected := []string{"2", "3"}
	if len(executed) != len(expected) {
		t.Fatalf("executed tasks = %v, want %v", executed, expected)
	}
	for i, taskNum := range expected {
		if executed[i] != taskNum {
			t.Fatalf("executed tasks = %v, want %v", executed, expected)
		}
	}
}

type scriptedOutcome struct {
	status   string
	output   string
	duration time.Duration
	err      error
}

type scriptedTaskExecutor struct {
	script   map[string]scriptedOutcome
	mu       sync.Mutex
	executed []string
}

func newScriptedTaskExecutor(script map[string]scriptedOutcome) *scriptedTaskExecutor {
	return &scriptedTaskExecutor{script: script}
}

func (s *scriptedTaskExecutor) Execute(ctx context.Context, task models.Task) (models.TaskResult, error) {
	outcome, ok := s.script[task.Number]
	if !ok {
		outcome = scriptedOutcome{status: models.StatusGreen, output: "default"}
	}

	s.mu.Lock()
	s.executed = append(s.executed, task.Number)
	s.mu.Unlock()

	result := models.TaskResult{
		Task:     task,
		Status:   outcome.status,
		Output:   outcome.output,
		Duration: outcome.duration,
	}

	if outcome.err != nil {
		result.Error = outcome.err
		return result, outcome.err
	}

	if result.Status == "" {
		result.Status = models.StatusGreen
	}

	return result, nil
}

func (s *scriptedTaskExecutor) ExecutedTasks() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.executed...)
}

type scenarioOptions struct {
	planPath      string
	skipCompleted bool
	retryFailed   bool
}

type scenarioOption func(*scenarioOptions)

func withPlan(path string) scenarioOption {
	return func(opts *scenarioOptions) {
		opts.planPath = path
	}
}

func withSkipCompleted(enabled bool) scenarioOption {
	return func(opts *scenarioOptions) {
		opts.skipCompleted = enabled
	}
}

func withRetryFailed(enabled bool) scenarioOption {
	return func(opts *scenarioOptions) {
		opts.retryFailed = enabled
	}
}

func runE2EScenario(t *testing.T, script map[string]scriptedOutcome, opts ...scenarioOption) (*models.ExecutionResult, *scriptedTaskExecutor, error) {
	t.Helper()

	config := scenarioOptions{
		planPath: filepath.Join("fixtures", "e2e-plan.md"),
	}
	for _, opt := range opts {
		opt(&config)
	}

	plan := loadPlanFromPath(t, config.planPath)
	taskExec := newScriptedTaskExecutor(script)
	var waveExec *executor.WaveExecutor
	if config.skipCompleted || config.retryFailed {
		waveExec = executor.NewWaveExecutorWithConfig(taskExec, nil, config.skipCompleted, config.retryFailed)
	} else {
		waveExec = executor.NewWaveExecutor(taskExec, nil)
	}
	orch := executor.NewOrchestratorWithConfig(waveExec, nil, config.skipCompleted, config.retryFailed)

	result, err := orch.ExecutePlan(context.Background(), plan)
	if result == nil {
		t.Fatal("ExecutePlan() returned nil result")
	}
	return result, taskExec, err
}

func loadPlanFromPath(t *testing.T, planPath string) *models.Plan {
	t.Helper()
	plan, err := parser.ParseFile(planPath)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}

	waves, err := executor.CalculateWaves(plan.Tasks)
	if err != nil {
		t.Fatalf("CalculateWaves() error = %v", err)
	}
	plan.Waves = waves
	return plan
}
