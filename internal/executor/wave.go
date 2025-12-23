package executor

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/harrison/conductor/internal/budget"
	"github.com/harrison/conductor/internal/config"
	"github.com/harrison/conductor/internal/models"
)

// TaskExecutor defines the behavior required to execute individual tasks within a wave.
type TaskExecutor interface {
	Execute(ctx context.Context, task models.Task) (models.TaskResult, error)
}

// ErrBudgetExceeded indicates the budget limit has been exceeded
var ErrBudgetExceeded = errors.New("budget exceeded")

// WaveExecutor coordinates sequential wave execution with bounded parallelism per wave.
type WaveExecutor struct {
	taskExecutor        TaskExecutor
	logger              Logger
	skipCompleted       bool                  // Skip tasks that are already completed
	retryFailed         bool                  // Retry tasks that have failed status
	packageGuard        *PackageGuard         // Runtime package conflict guard (v2.9+)
	enforcePackageGuard bool                  // Enable package guard enforcement
	guardProtocol       *GuardProtocol        // GUARD Protocol for failure prediction
	anomalyConfig       *AnomalyMonitorConfig // Real-time anomaly detection config (v2.18+)
	// Budget tracking (v2.19+)
	budgetTracker *budget.UsageTracker
	budgetConfig  *config.BudgetConfig
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

// SetAnomalyConfig sets the anomaly detection configuration.
// This enables real-time anomaly detection during wave execution.
func (w *WaveExecutor) SetAnomalyConfig(config *AnomalyMonitorConfig) {
	w.anomalyConfig = config
}

// SetBudgetTracking configures budget tracking for the wave executor.
// When enabled, checks budget before each wave and warns/stops if limits exceeded.
func (w *WaveExecutor) SetBudgetTracking(tracker *budget.UsageTracker, cfg *config.BudgetConfig) {
	w.budgetTracker = tracker
	w.budgetConfig = cfg
}

// checkBudget verifies budget limits before wave execution.
// Returns an error if budget is exceeded and execution should stop.
// Logs warnings if approaching budget threshold.
func (w *WaveExecutor) checkBudget() error {
	if w.budgetTracker == nil || w.budgetConfig == nil || !w.budgetConfig.Enabled {
		return nil
	}

	// Skip if check interval is "disabled"
	if w.budgetConfig.CheckInterval == "disabled" {
		return nil
	}

	status := w.budgetTracker.GetStatus()
	if status == nil || status.Block == nil {
		return nil
	}

	currentCost := status.Block.CostUSD

	// Check against MaxCostPerRun
	if w.budgetConfig.MaxCostPerRun > 0 {
		ratio := currentCost / w.budgetConfig.MaxCostPerRun

		// Stop if budget exceeded
		if ratio >= 1.0 {
			if w.logger != nil {
				w.logger.LogBudgetStatus(status)
			}
			return fmt.Errorf("budget exceeded: current cost $%.2f exceeds limit $%.2f",
				currentCost, w.budgetConfig.MaxCostPerRun)
		}

		// Warn if approaching threshold
		if ratio >= w.budgetConfig.WarnThreshold {
			if w.logger != nil {
				w.logger.LogBudgetWarning(ratio)
			}
		}
	}

	return nil
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

	// ========== GUARD PROTOCOL GATE ==========
	if w.guardProtocol != nil {
		// Build task slice for GUARD check
		var tasksForGuard []models.Task
		for _, taskNum := range tasksToExecute {
			if task, ok := taskMap[taskNum]; ok {
				tasksForGuard = append(tasksForGuard, task)
			}
		}

		// Run pre-wave failure prediction
		guardResults, err := w.guardProtocol.CheckWave(ctx, tasksForGuard)
		if err == nil && guardResults != nil {
			var blockedTasks []string

			for taskNum, guardResult := range guardResults {
				// Log prediction via logger if available
				if w.logger != nil {
					w.logger.LogGuardPrediction(taskNum, guardResult)
				}

				if guardResult.ShouldBlock {
					task := taskMap[taskNum]
					blockedTasks = append(blockedTasks, taskNum)

					// Create failed result for blocked task
					blockedResult := models.TaskResult{
						Task:   task,
						Status: models.StatusFailed,
						Error:  fmt.Errorf("GUARD blocked: %s", guardResult.BlockReason),
					}
					if guardResult.Prediction != nil {
						blockedResult.Output = fmt.Sprintf("Risk factors: %v\nRecommendations: %v",
							guardResult.Prediction.RiskFactors,
							guardResult.Recommendations)
					}
					skippedResults = append(skippedResults, blockedResult)
				} else if guardResult.SuggestedAgent != "" {
					// Predictive agent selection: swap to better agent before execution
					task := taskMap[taskNum]
					originalAgent := task.Agent
					task.Agent = guardResult.SuggestedAgent
					taskMap[taskNum] = task
					if w.logger != nil {
						w.logger.LogAgentSwap(taskNum, originalAgent, guardResult.SuggestedAgent)
					}
				}
			}

			// Remove blocked tasks from execution queue
			if len(blockedTasks) > 0 {
				tasksToExecute = filterOutTasks(tasksToExecute, blockedTasks)
			}
		}
		// Graceful degradation: if GUARD fails, continue with all tasks
	}
	// ========== END GUARD PROTOCOL GATE ==========

	// ========== BUDGET GATE ==========
	if w.budgetConfig != nil && w.budgetConfig.CheckInterval == "per_wave" {
		if err := w.checkBudget(); err != nil {
			return skippedResults, fmt.Errorf("budget gate: %w", err)
		}
	}
	// ========== END BUDGET GATE ==========

	// If all tasks were blocked by GUARD, return the blocked results
	if len(tasksToExecute) == 0 {
		return skippedResults, nil
	}

	// ========== ANOMALY MONITOR SETUP ==========
	var anomalyMonitor *AnomalyMonitor
	if w.anomalyConfig != nil && w.anomalyConfig.ConsecutiveFailureThreshold > 0 {
		anomalyMonitor = NewAnomalyMonitorWithConfig(wave.Name, *w.anomalyConfig)
	}
	// ========== END ANOMALY MONITOR SETUP ==========

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

		// ========== REAL-TIME ANOMALY DETECTION ==========
		if anomalyMonitor != nil {
			anomalies := anomalyMonitor.RecordResult(executionResult.result)
			for _, anomaly := range anomalies {
				if w.logger != nil {
					w.logger.LogAnomaly(anomaly)
				}
			}
		}
		// ========== END ANOMALY DETECTION ==========
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

// filterOutTasks removes excluded tasks from the task list.
func filterOutTasks(tasks []string, exclude []string) []string {
	excludeMap := make(map[string]bool)
	for _, t := range exclude {
		excludeMap[t] = true
	}
	result := make([]string, 0, len(tasks))
	for _, t := range tasks {
		if !excludeMap[t] {
			result = append(result, t)
		}
	}
	return result
}
