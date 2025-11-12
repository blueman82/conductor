package executor

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/harrison/conductor/internal/agent"
	"github.com/harrison/conductor/internal/models"
	"github.com/harrison/conductor/internal/updater"
)

// Task status constants for plan updates.
const (
	StatusInProgress = "in-progress"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
)

// ErrQualityGateFailed indicates that a task failed quality control after exhausting retries.
var ErrQualityGateFailed = errors.New("quality control failed after maximum retries")

// Reviewer is implemented by quality control components capable of reviewing task output.
type Reviewer interface {
	Review(ctx context.Context, task models.Task, output string) (*ReviewResult, error)
	ShouldRetry(result *ReviewResult, currentAttempt int) bool
}

// PlanUpdater abstracts plan status updates to allow fakes in tests.
type PlanUpdater interface {
	Update(planPath string, taskNumber string, status string, completedAt *time.Time) error
}

type planUpdaterFunc func(planPath string, taskNumber string, status string, completedAt *time.Time) error

func (f planUpdaterFunc) Update(planPath string, taskNumber string, status string, completedAt *time.Time) error {
	if f == nil {
		return nil
	}
	return f(planPath, taskNumber, status, completedAt)
}

// FileLockManager manages per-file locking to prevent concurrent modifications.
// Each file has its own lock, allowing concurrent updates to different files
// while serializing updates to the same file.
type FileLockManager interface {
	// Lock acquires an exclusive lock for the given file path.
	// Returns a function that should be called to release the lock.
	Lock(filePath string) func()
}

// DefaultFileLockManager implements per-file locking using sync.Mutex.
type DefaultFileLockManager struct {
	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

// NewFileLockManager creates a new file lock manager.
func NewFileLockManager() *DefaultFileLockManager {
	return &DefaultFileLockManager{
		locks: make(map[string]*sync.Mutex),
	}
}

// Lock acquires an exclusive lock for the given file path.
func (dm *DefaultFileLockManager) Lock(filePath string) func() {
	dm.mu.Lock()
	// Get or create lock for this file
	fileLock, exists := dm.locks[filePath]
	if !exists {
		fileLock = &sync.Mutex{}
		dm.locks[filePath] = fileLock
	}
	dm.mu.Unlock()

	// Acquire the file-specific lock
	fileLock.Lock()

	// Return unlock function
	return func() {
		fileLock.Unlock()
	}
}

// TaskExecutorConfig configures TaskExecutor behaviour for a specific plan.
type TaskExecutorConfig struct {
	PlanPath       string
	DefaultAgent   string
	QualityControl models.QualityControlConfig
}

// DefaultTaskExecutor executes individual tasks, applying QC review and plan updates.
type DefaultTaskExecutor struct {
	invoker         InvokerInterface
	reviewer        Reviewer
	planUpdater     PlanUpdater
	cfg             TaskExecutorConfig
	clock           func() time.Time
	qcEnabled       bool
	retryLimit      int
	SourceFile      string          // Track which file this task comes from
	FileLockManager FileLockManager // Per-file locking strategy
}

// NewTaskExecutor constructs a TaskExecutor implementation.
func NewTaskExecutor(invoker InvokerInterface, reviewer Reviewer, planUpdater PlanUpdater, cfg TaskExecutorConfig) (*DefaultTaskExecutor, error) {
	if invoker == nil {
		return nil, fmt.Errorf("task executor requires an invoker")
	}

	if planUpdater == nil {
		planUpdater = planUpdaterFunc(func(path string, taskNumber string, status string, completedAt *time.Time) error {
			if path == "" {
				return nil
			}
			return updater.UpdateTaskStatus(path, taskNumber, status, completedAt)
		})
	}

	te := &DefaultTaskExecutor{
		invoker:         invoker,
		reviewer:        reviewer,
		planUpdater:     planUpdater,
		cfg:             cfg,
		clock:           time.Now,
		FileLockManager: NewFileLockManager(),
	}

	if cfg.QualityControl.Enabled {
		te.qcEnabled = true
		te.retryLimit = cfg.QualityControl.RetryOnRed
		if te.retryLimit < 0 {
			te.retryLimit = 0
		}

		if te.reviewer == nil {
			qc := NewQualityController(invoker)
			if cfg.QualityControl.ReviewAgent != "" {
				qc.ReviewAgent = cfg.QualityControl.ReviewAgent
			}
			qc.MaxRetries = te.retryLimit
			te.reviewer = qc
		} else if qc, ok := te.reviewer.(*QualityController); ok {
			if cfg.QualityControl.ReviewAgent != "" {
				qc.ReviewAgent = cfg.QualityControl.ReviewAgent
			}
			qc.MaxRetries = te.retryLimit
		}
	}

	return te, nil
}

// Execute runs an individual task, handling agent invocation, quality control, and plan updates.
func (te *DefaultTaskExecutor) Execute(ctx context.Context, task models.Task) (models.TaskResult, error) {
	result := models.TaskResult{Task: task}

	// Determine which file to lock and update - priority order:
	// 1. task.SourceFile (set for multi-file plans)
	// 2. te.SourceFile (set by legacy code or single-file plans)
	// 3. te.cfg.PlanPath (fallback)
	fileToLock := te.cfg.PlanPath
	if task.SourceFile != "" {
		fileToLock = task.SourceFile
	} else if te.SourceFile != "" {
		fileToLock = te.SourceFile
	}

	// Acquire per-file lock to ensure only one task updates this file at a time
	unlock := te.FileLockManager.Lock(fileToLock)
	defer unlock()

	// Apply default agent if provided.
	if task.Agent == "" && te.cfg.DefaultAgent != "" {
		task.Agent = te.cfg.DefaultAgent
		result.Task.Agent = te.cfg.DefaultAgent
	}

	if err := te.updatePlanStatus(task, StatusInProgress, false); err != nil {
		result.Status = models.StatusFailed
		result.Error = err
		return result, err
	}

	maxAttempt := te.retryLimit
	if !te.qcEnabled || te.reviewer == nil {
		maxAttempt = 0
	}

	var totalDuration time.Duration
	var lastErr error

	for attempt := 0; attempt <= maxAttempt; attempt++ {
		if err := ctx.Err(); err != nil {
			// Wrap context errors with TimeoutError for better error handling
			if errors.Is(err, context.DeadlineExceeded) {
				timeoutErr := NewTimeoutError(task.Number, 0)
				timeoutErr.Context = "task execution timeout"
				result.Status = models.StatusFailed
				result.Error = timeoutErr
				_ = te.updatePlanStatus(task, StatusFailed, false)
				return result, timeoutErr
			}
			result.Status = models.StatusFailed
			result.Error = err
			_ = te.updatePlanStatus(task, StatusFailed, false)
			return result, err
		}

		invocation, err := te.invoker.Invoke(ctx, task)
		if err != nil {
			// Wrap invocation errors with TimeoutError if it's a timeout
			if errors.Is(err, context.DeadlineExceeded) {
				timeoutErr := NewTimeoutError(task.Number, 0)
				timeoutErr.Context = "invoker timeout"
				result.Status = models.StatusFailed
				result.Error = timeoutErr
				_ = te.updatePlanStatus(task, StatusFailed, false)
				return result, timeoutErr
			}
			result.Status = models.StatusFailed
			result.Error = err
			_ = te.updatePlanStatus(task, StatusFailed, false)
			return result, err
		}

		totalDuration += invocation.Duration

		if invocation.Error != nil {
			taskErr := NewTaskError(task.Number, "task invocation failed", invocation.Error)
			result.Status = models.StatusFailed
			result.Error = taskErr
			_ = te.updatePlanStatus(task, StatusFailed, false)
			return result, taskErr
		}
		if invocation.ExitCode != 0 {
			taskErr := NewTaskError(task.Number, fmt.Sprintf("task exited with code %d", invocation.ExitCode), nil)
			result.Status = models.StatusFailed
			result.Error = taskErr
			_ = te.updatePlanStatus(task, StatusFailed, false)
			return result, taskErr
		}

		parsedOutput, _ := agent.ParseClaudeOutput(invocation.Output)
		output := invocation.Output
		if parsedOutput != nil {
			if parsedOutput.Content != "" {
				output = parsedOutput.Content
			} else if parsedOutput.Error != "" {
				output = parsedOutput.Error
			} else {
				output = ""
			}
		}

		result.Output = output
		result.Duration = totalDuration

		if !te.qcEnabled || te.reviewer == nil {
			result.Status = models.StatusGreen
			result.RetryCount = attempt
			if err := te.updatePlanStatus(task, StatusCompleted, true); err != nil {
				result.Status = models.StatusFailed
				result.Error = err
				return result, err
			}
			return result, nil
		}

		review, reviewErr := te.reviewer.Review(ctx, task, output)
		if reviewErr != nil {
			result.Status = models.StatusFailed
			result.Error = reviewErr
			_ = te.updatePlanStatus(task, StatusFailed, false)
			return result, reviewErr
		}

		if review != nil {
			result.ReviewFeedback = review.Feedback
		}

		switch {
		case review == nil || review.Flag == "":
			lastErr = NewTaskError(task.Number, "quality control did not return a valid flag", nil)
		case review.Flag == models.StatusGreen:
			result.Status = models.StatusGreen
			result.RetryCount = attempt
			if err := te.updatePlanStatus(task, StatusCompleted, true); err != nil {
				result.Status = models.StatusFailed
				result.Error = err
				return result, err
			}
			return result, nil
		case review.Flag == models.StatusYellow:
			result.Status = models.StatusYellow
			result.RetryCount = attempt
			if err := te.updatePlanStatus(task, StatusCompleted, true); err != nil {
				result.Status = models.StatusFailed
				result.Error = err
				return result, err
			}
			return result, nil
		case review.Flag == models.StatusRed:
			lastErr = ErrQualityGateFailed
		default:
			lastErr = NewTaskError(task.Number, fmt.Sprintf("quality control returned unsupported flag %q", review.Flag), nil)
		}

		if lastErr == nil {
			lastErr = ErrQualityGateFailed
		}

		// Determine whether to retry.
		if attempt >= te.retryLimit || !te.reviewer.ShouldRetry(review, attempt) {
			result.Status = models.StatusRed
			result.RetryCount = attempt
			result.Error = lastErr
			_ = te.updatePlanStatus(task, StatusFailed, false)
			return result, lastErr
		}
	}

	// Should not reach here; treat as failure.
	if lastErr == nil {
		lastErr = ErrQualityGateFailed
	}
	result.Status = models.StatusRed
	result.RetryCount = te.retryLimit
	result.Error = lastErr
	_ = te.updatePlanStatus(task, StatusFailed, false)
	return result, lastErr
}

func (te *DefaultTaskExecutor) updatePlanStatus(task models.Task, status string, markComplete bool) error {
	if te.planUpdater == nil {
		return nil
	}

	var completedAt *time.Time
	if markComplete {
		ts := te.clock().UTC()
		completedAt = &ts
	}

	// Determine which file to update - priority order:
	// 1. task.SourceFile (set for multi-file plans)
	// 2. te.SourceFile (set by legacy code or single-file plans)
	// 3. te.cfg.PlanPath (fallback)
	fileToUpdate := te.cfg.PlanPath
	if task.SourceFile != "" {
		fileToUpdate = task.SourceFile
	} else if te.SourceFile != "" {
		fileToUpdate = te.SourceFile
	}

	return te.planUpdater.Update(fileToUpdate, task.Number, status, completedAt)
}
