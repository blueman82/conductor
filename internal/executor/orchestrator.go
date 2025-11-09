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

// Logger defines the interface for logging orchestrator progress and results.
type Logger interface {
	LogWaveStart(wave models.Wave)
	LogWaveComplete(wave models.Wave, duration time.Duration)
	LogTaskStart(task models.Task)
	LogTaskComplete(result models.TaskResult)
	LogTaskFail(result models.TaskResult)
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
// The logger parameter is optional and can be nil.
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

	// Count completed and failed tasks
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

// isCompleted checks if a task result represents successful completion.
func isCompleted(result models.TaskResult) bool {
	// GREEN and YELLOW are considered successful completions
	return result.Status == "GREEN" || result.Status == "YELLOW"
}

// isFailed checks if a task result represents a failure.
func isFailed(result models.TaskResult) bool {
	// RED, FAILED status, or presence of an error indicates failure
	return result.Status == "RED" || result.Status == "FAILED" || result.Error != nil
}
