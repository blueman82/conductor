// Package executor provides task execution orchestration for Conductor.
//
// The executor package coordinates multi-wave task execution with quality control,
// graceful shutdown, and result aggregation. It manages the lifecycle of plan
// execution from parsing through completion.
//
// Key components:
//   - Orchestrator: Coordinates plan execution with signal handling and result aggregation
//   - WaveExecutor: Executes waves sequentially with bounded parallelism within each wave
//   - TaskExecutor: Executes individual tasks with quality control review and retry logic
//   - QualityController: Reviews task output and determines GREEN/RED/YELLOW status
//   - DependencyGraph: Calculates execution waves using Kahn's algorithm
//
// The execution flow is:
//   Plan → Graph → Waves → Orchestrator → WaveExecutor → TaskExecutor → QC → Results
package executor

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/harrison/conductor/internal/models"
)

// Logger interface for orchestrator progress reporting.
// Task-level logging is deferred to CLI implementation (Task 16/18).
// The orchestrator operates at wave/summary granularity.
// Implementations can log to console, file, or other destinations.
type Logger interface {
	LogWaveStart(wave models.Wave)
	LogWaveComplete(wave models.Wave, duration time.Duration)
	LogSummary(result models.ExecutionResult)
}

// WaveExecutorInterface defines the behavior required to execute waves.
type WaveExecutorInterface interface {
	ExecutePlan(ctx context.Context, plan *models.Plan) ([]models.TaskResult, error)
}

// Orchestrator coordinates plan execution, handles graceful shutdown, and aggregates results.
type Orchestrator struct {
	waveExecutor WaveExecutorInterface
	logger       Logger
}

// NewOrchestrator creates a new Orchestrator instance.
// The logger parameter is optional and can be nil to disable logging.
// The waveExecutor is required and must not be nil.
func NewOrchestrator(waveExecutor WaveExecutorInterface, logger Logger) *Orchestrator {
	if waveExecutor == nil {
		panic("wave executor cannot be nil")
	}

	return &Orchestrator{
		waveExecutor: waveExecutor,
		logger:       logger,
	}
}

// ExecutePlan orchestrates the execution of a plan with graceful shutdown support.
// It handles SIGINT/SIGTERM signals, coordinates wave execution, logs progress,
// and aggregates results into an ExecutionResult.
func (o *Orchestrator) ExecutePlan(ctx context.Context, plan *models.Plan) (*models.ExecutionResult, error) {
	if plan == nil {
		return nil, fmt.Errorf("plan cannot be nil")
	}

	// Set up context with cancellation for signal handling
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	// Start signal handler in background
	go func() {
		select {
		case <-sigChan:
			if o.logger != nil {
				// Note: We can't easily test this log message, but it's here for production use
				fmt.Println("\nReceived interrupt signal, shutting down gracefully...")
			}
			cancel()
		case <-ctx.Done():
			// Context already canceled
		}
	}()

	startTime := time.Now()

	// Execute the plan through the wave executor
	results, err := o.waveExecutor.ExecutePlan(ctx, plan)

	duration := time.Since(startTime)

	// Aggregate results
	executionResult := o.aggregateResults(plan, results, duration)

	// Log summary
	if o.logger != nil {
		o.logger.LogSummary(*executionResult)
	}

	return executionResult, err
}

// aggregateResults processes task results and creates an ExecutionResult summary.
func (o *Orchestrator) aggregateResults(plan *models.Plan, results []models.TaskResult, duration time.Duration) *models.ExecutionResult {
	executionResult := &models.ExecutionResult{
		TotalTasks:  len(plan.Tasks),
		Completed:   0,
		Failed:      0,
		Duration:    duration,
		FailedTasks: []models.TaskResult{},
	}

	// Count completed and failed tasks (handles empty results slice gracefully)
	for _, result := range results {
		if isCompleted(result) {
			executionResult.Completed++
		} else if isFailed(result) {
			executionResult.Failed++
			executionResult.FailedTasks = append(executionResult.FailedTasks, result)
		}
	}

	return executionResult
}

// isCompleted returns true if the task result indicates successful completion.
// Both GREEN (perfect) and YELLOW (acceptable with warnings) are considered completed.
func isCompleted(result models.TaskResult) bool {
	return result.Status == models.StatusGreen || result.Status == models.StatusYellow
}

// isFailed returns true if the task result indicates failure.
// This includes RED (quality control failed), FAILED (execution error), or presence of Error field.
func isFailed(result models.TaskResult) bool {
	return result.Status == models.StatusRed || result.Status == models.StatusFailed || result.Error != nil
}
