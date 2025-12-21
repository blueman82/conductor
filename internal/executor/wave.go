package executor

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/harrison/conductor/internal/models"
)

// TaskExecutor defines the behavior required to execute individual tasks within a wave.
type TaskExecutor interface {
	Execute(ctx context.Context, task models.Task) (models.TaskResult, error)
}

// WaveExecutor coordinates sequential wave execution with bounded parallelism per wave.
type WaveExecutor struct {
	taskExecutor        TaskExecutor
	logger              Logger
	skipCompleted       bool          // Skip tasks that are already completed
	retryFailed         bool          // Retry tasks that have failed status
	packageGuard        *PackageGuard // Runtime package conflict guard (v2.9+)
	enforcePackageGuard bool          // Enable package guard enforcement
	guardProtocol       *GuardProtocol // GUARD Protocol for failure prediction
}

// NewWaveExecutor constructs a WaveExecutor with the provided task executor implementation.
// The logger parameter is optional and can be nil to disable logging.
func NewWaveExecutor(taskExecutor TaskExecutor, logger Logger) *WaveExecutor {
	return &WaveExecutor{
		taskExecutor:  taskExecutor,
		logger:        logger,
		skipCompleted: false, // Default: don't skip
		retryFailed:   false, // Default: don't retry
	}
}

// NewWaveExecutorWithConfig constructs a WaveExecutor with skip/retry configuration.
func NewWaveExecutorWithConfig(taskExecutor TaskExecutor, logger Logger, skipCompleted, retryFailed bool) *WaveExecutor {
	return &WaveExecutor{
		taskExecutor:  taskExecutor,
		logger:        logger,
		skipCompleted: skipCompleted,
		retryFailed:   retryFailed,
	}
}

// NewWaveExecutorWithPackageGuard constructs a WaveExecutor with package guard enforcement.
func NewWaveExecutorWithPackageGuard(taskExecutor TaskExecutor, logger Logger, skipCompleted, retryFailed, enforcePackageGuard bool) *WaveExecutor {
	return &WaveExecutor{
		taskExecutor:        taskExecutor,
		logger:              logger,
		skipCompleted:       skipCompleted,
		retryFailed:         retryFailed,
		packageGuard:        NewPackageGuard(),
		enforcePackageGuard: enforcePackageGuard,
	}
}

// SetPackageGuard enables or disables package guard enforcement.
func (w *WaveExecutor) SetPackageGuard(enabled bool) {
	w.enforcePackageGuard = enabled
	if enabled && w.packageGuard == nil {
		w.packageGuard = NewPackageGuard()
	}
}

// SetGuardProtocol sets the GUARD Protocol for failure prediction.
// This enables pre-wave risk analysis and adaptive blocking.
func (w *WaveExecutor) SetGuardProtocol(guard *GuardProtocol) {
	w.guardProtocol = guard
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

	// Wire plan reference into task executor for integration prompt builder
	if te, ok := w.taskExecutor.(*DefaultTaskExecutor); ok {
		te.Plan = plan
	}

	taskMap := make(map[string]models.Task, len(plan.Tasks))
	for _, task := range plan.Tasks {
		taskMap[task.Number] = task
	}

	var allResults []models.TaskResult
	var firstErr error

	for _, wave := range plan.Waves {
		waveResults, err := w.executeWave(ctx, wave, taskMap, plan.Tasks)
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

func (w *WaveExecutor) executeWave(ctx context.Context, wave models.Wave, taskMap map[string]models.Task, allTasks []models.Task) ([]models.TaskResult, error) {
	taskCount := len(wave.TaskNumbers)
	if taskCount == 0 {
		return []models.TaskResult{}, nil
	}

	// Separate tasks into skipped and to-execute based on CanSkip() logic
	var tasksToExecute []string
	var skippedResults []models.TaskResult

	for _, taskNumber := range wave.TaskNumbers {
		task, ok := taskMap[taskNumber]
		if !ok {
			return nil, NewTaskError(taskNumber, fmt.Sprintf("task not found in %s", wave.Name), nil)
		}

		// Check if we should skip this task
		if w.skipCompleted && task.CanSkip() {
			// Create synthetic GREEN result for skipped task
			skippedResult := models.TaskResult{
				Task:   task,
				Status: models.StatusGreen,
				Output: "Skipped",
			}
			skippedResults = append(skippedResults, skippedResult)

			// Log the skipped task result
			if w.logger != nil {
				if logErr := w.logger.LogTaskResult(skippedResult); logErr != nil {
					// Log error but don't fail execution
				}
			}
		} else {
			tasksToExecute = append(tasksToExecute, taskNumber)
		}
	}

	// If all tasks are skipped, return the skipped results
	if len(tasksToExecute) == 0 {
		return skippedResults, nil
	}

	// Track how many task goroutines actually started to prevent logging
	// wave start/completion for pre-cancelled contexts where no tasks launched.
	var tasksLaunched int32
	var waveLogged bool

	waveStartTime := time.Now()

	maxConcurrency := wave.MaxConcurrency
	execCount := len(tasksToExecute)
	if maxConcurrency <= 0 || maxConcurrency > execCount {
		maxConcurrency = execCount
	}
	if maxConcurrency == 0 {
		maxConcurrency = 1
	}

	semaphore := make(chan struct{}, maxConcurrency)
	resultsCh := make(chan taskExecutionResult, execCount)

	var wg sync.WaitGroup
	var launchErr error

	for _, taskNumber := range tasksToExecute {
		if err := ctx.Err(); err != nil {
			launchErr = err
			break
		}

		task := taskMap[taskNumber]

		// Check context again before acquiring semaphore to avoid blocking on a cancelled context
		select {
		case <-ctx.Done():
			launchErr = ctx.Err()
			goto launchComplete
		case semaphore <- struct{}{}:
			// Successfully acquired semaphore slot
		}

		wg.Add(1)

		// Log wave start only once, when the first task successfully acquires a semaphore slot
		if !waveLogged && w.logger != nil {
			w.logger.LogWaveStart(wave)
			waveLogged = true
		}

		go func(task models.Task) {
			atomic.AddInt32(&tasksLaunched, 1)
			defer wg.Done()
			defer func() { <-semaphore }()

			// Set SourceFile on executor before execution (for multi-file plans)
			if taskExec, ok := w.taskExecutor.(*DefaultTaskExecutor); ok {
				if task.SourceFile != "" {
					taskExec.SourceFile = task.SourceFile
				}
			}

			// Acquire package locks if guard is enabled (v2.9+)
			var releasePackages func()
			if w.enforcePackageGuard && w.packageGuard != nil {
				packages := GetTaskPackages(task)
				if len(packages) > 0 {
					var acquireErr error
					releasePackages, acquireErr = w.packageGuard.Acquire(ctx, task.Number, packages)
					if acquireErr != nil {
						// Failed to acquire - report as task failure
						result := models.TaskResult{
							Task:   task,
							Status: models.StatusFailed,
							Error:  fmt.Errorf("package guard: %w", acquireErr),
						}
						select {
						case resultsCh <- taskExecutionResult{taskNumber: task.Number, result: result, err: acquireErr}:
						case <-ctx.Done():
						}
						return
					}
				}
			}

			// Ensure packages are released after task execution
			if releasePackages != nil {
				defer releasePackages()
			}

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

launchComplete:
	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	resultMap := make(map[string]models.TaskResult, execCount)
	var execErr error

	for executionResult := range resultsCh {
		resultMap[executionResult.taskNumber] = executionResult.result
		if execErr == nil && executionResult.err != nil {
			execErr = executionResult.err
		}

		// Log individual task result
		if w.logger != nil {
			if logErr := w.logger.LogTaskResult(executionResult.result); logErr != nil {
				// Log error but don't fail execution
				// Task completed but logging failed - not a critical failure
			}
		}

		// Log progress with task results (pass results with actual duration data)
		if w.logger != nil {
			var resultsForProgress []models.TaskResult
			for _, taskNum := range wave.TaskNumbers {
				if result, ok := resultMap[taskNum]; ok {
					resultsForProgress = append(resultsForProgress, result)
				}
			}
			w.logger.LogProgress(resultsForProgress)
		}
	}

	waveResults := make([]models.TaskResult, 0, taskCount)

	// Build results map including skipped tasks for easy lookup
	allResults := make(map[string]models.TaskResult)
	for _, result := range skippedResults {
		allResults[result.Task.Number] = result
	}
	for taskNum, result := range resultMap {
		allResults[taskNum] = result
	}

	// Add results in original wave order
	for _, taskNumber := range wave.TaskNumbers {
		if result, ok := allResults[taskNumber]; ok {
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

	// Log wave completion only if at least one task was actually launched
	waveDuration := time.Since(waveStartTime)
	if atomic.LoadInt32(&tasksLaunched) > 0 && w.logger != nil {
		w.logger.LogWaveComplete(wave, waveDuration, waveResults)
	}

	return waveResults, execErr
}
