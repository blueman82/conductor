package executor

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/harrison/conductor/internal/models"
)

// TaskExecutor defines the behavior required to execute individual tasks within a wave.
type TaskExecutor interface {
	Execute(ctx context.Context, task models.Task) (models.TaskResult, error)
}

// WaveExecutor coordinates sequential wave execution with bounded parallelism per wave.
type WaveExecutor struct {
	taskExecutor TaskExecutor
}

// NewWaveExecutor constructs a WaveExecutor with the provided task executor implementation.
func NewWaveExecutor(taskExecutor TaskExecutor) *WaveExecutor {
	return &WaveExecutor{taskExecutor: taskExecutor}
}

// ExecutePlan runs the plan's waves sequentially while executing tasks within each wave in parallel.
// It returns all collected task results and the first error encountered, if any.
func (w *WaveExecutor) ExecutePlan(ctx context.Context, plan *models.Plan) ([]models.TaskResult, error) {
	if w == nil {
		return nil, fmt.Errorf("wave executor is nil")
	}
	if plan == nil {
		return nil, fmt.Errorf("plan cannot be nil")
	}
	if w.taskExecutor == nil {
		return nil, fmt.Errorf("task executor is required")
	}

	taskMap := make(map[string]models.Task, len(plan.Tasks))
	for _, task := range plan.Tasks {
		taskMap[task.Number] = task
	}

	var allResults []models.TaskResult
	var firstErr error

	for _, wave := range plan.Waves {
		waveResults, err := w.executeWave(ctx, wave, taskMap)
		allResults = append(allResults, waveResults...)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			// Stop executing subsequent waves once an error is encountered.
			break
		}
	}

	return allResults, firstErr
}

type taskExecutionResult struct {
	taskNumber string
	result     models.TaskResult
	err        error
}

func (w *WaveExecutor) executeWave(ctx context.Context, wave models.Wave, taskMap map[string]models.Task) ([]models.TaskResult, error) {
	taskCount := len(wave.TaskNumbers)
	if taskCount == 0 {
		return []models.TaskResult{}, nil
	}

	maxConcurrency := wave.MaxConcurrency
	if maxConcurrency <= 0 || maxConcurrency > taskCount {
		maxConcurrency = taskCount
	}
	if maxConcurrency == 0 {
		maxConcurrency = 1
	}

	semaphore := make(chan struct{}, maxConcurrency)
	resultsCh := make(chan taskExecutionResult, taskCount)

	var wg sync.WaitGroup
	var launchErr error

	for _, taskNumber := range wave.TaskNumbers {
		if err := ctx.Err(); err != nil {
			launchErr = err
			break
		}

		task, ok := taskMap[taskNumber]
		if !ok {
			launchErr = fmt.Errorf("%s: task %d not found", wave.Name, taskNumber)
			break
		}

		semaphore <- struct{}{}
		wg.Add(1)
		go func(task models.Task) {
			defer wg.Done()
			defer func() { <-semaphore }()

			result, err := w.taskExecutor.Execute(ctx, task)
			if result.Task.Number == "" {
				result.Task = task
			}
			if err != nil && result.Error == nil {
				result.Error = err
			}
			if result.Status == "" && err != nil {
				result.Status = models.StatusFailed
			}

			select {
			case resultsCh <- taskExecutionResult{taskNumber: task.Number, result: result, err: err}:
			case <-ctx.Done():
			}
		}(task)
	}

	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	resultMap := make(map[string]models.TaskResult, taskCount)
	var execErr error

	for executionResult := range resultsCh {
		resultMap[executionResult.taskNumber] = executionResult.result
		if execErr == nil && executionResult.err != nil {
			execErr = executionResult.err
		}
	}

	waveResults := make([]models.TaskResult, 0, len(resultMap))
	for _, taskNumber := range wave.TaskNumbers {
		if result, ok := resultMap[taskNumber]; ok {
			waveResults = append(waveResults, result)
		}
	}

	if launchErr != nil {
		if execErr == nil {
			execErr = launchErr
		}
	} else if execErr != nil && errors.Is(execErr, context.Canceled) {
		// Propagate context cancellation explicitly
		execErr = context.Canceled
	}

	return waveResults, execErr
}
