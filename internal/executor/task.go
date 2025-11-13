package executor

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

// LearningStore defines the interface for adaptive learning storage.
type LearningStore interface {
	AnalyzeFailures(ctx context.Context, planFile, taskNumber string) (*FailureAnalysis, error)
	RecordExecution(ctx context.Context, exec *TaskExecution) error
}

// TaskExecution represents a single task execution record for learning storage.
type TaskExecution struct {
	PlanFile        string
	RunNumber       int
	TaskNumber      string
	TaskName        string
	Agent           string
	Prompt          string
	Success         bool
	Output          string
	ErrorMessage    string
	DurationSecs    int64
	QCVerdict       string
	QCFeedback      string
	FailurePatterns []string
}

// FailureAnalysis contains metrics and recommendations from analyzing task execution history.
type FailureAnalysis struct {
	TotalAttempts           int
	FailedAttempts          int
	TriedAgents             []string
	CommonPatterns          []string
	SuggestedAgent          string
	SuggestedApproach       string
	ShouldTryDifferentAgent bool
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
	learningStore   LearningStore   // Adaptive learning store (optional)
	planFile        string          // Plan file path for learning queries
	sessionID       string          // Session ID for learning tracking
	runNumber       int             // Run number for learning tracking
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

// preTaskHook queries the learning database and adapts agent/prompt before execution.
// This hook enables adaptive learning from past failures.
func (te *DefaultTaskExecutor) preTaskHook(ctx context.Context, task *models.Task) error {
	// Learning disabled - no-op
	if te.learningStore == nil {
		return nil
	}

	// Query learning store for failure analysis
	analysis, err := te.learningStore.AnalyzeFailures(ctx, te.planFile, task.Number)
	if err != nil {
		// Log warning but don't break execution (graceful degradation)
		// In production, this would use a proper logger
		// For now, just continue without learning
		return nil
	}

	// No analysis available
	if analysis == nil {
		return nil
	}

	// Adapt agent if recommended and suggestion available
	if analysis.ShouldTryDifferentAgent && analysis.SuggestedAgent != "" {
		// Only switch if different from current agent
		if task.Agent != analysis.SuggestedAgent {
			// Log agent switch for observability
			// In production: log.Info("Switching agent: %s → %s", task.Agent, analysis.SuggestedAgent)
			task.Agent = analysis.SuggestedAgent
		}
	}

	// Enhance prompt with learning context if there are past failures
	if analysis.FailedAttempts > 0 {
		task.Prompt = enhancePromptWithLearning(task.Prompt, analysis)
	}

	return nil
}

// enhancePromptWithLearning adds learning context to the task prompt.
func enhancePromptWithLearning(originalPrompt string, analysis *FailureAnalysis) string {
	if analysis == nil || analysis.FailedAttempts == 0 {
		return originalPrompt
	}

	// Build learning context
	learningContext := fmt.Sprintf("\n\nNote: This task has %d past failures. ", analysis.FailedAttempts)

	if len(analysis.TriedAgents) > 0 {
		learningContext += fmt.Sprintf("Previously tried agents: %s. ", strings.Join(analysis.TriedAgents, ", "))
	}

	if len(analysis.CommonPatterns) > 0 {
		learningContext += fmt.Sprintf("Common issues: %s. ", strings.Join(analysis.CommonPatterns, ", "))
	}

	if analysis.SuggestedApproach != "" {
		learningContext += fmt.Sprintf("\n\nRecommended approach: %s", analysis.SuggestedApproach)
	}

	return originalPrompt + learningContext
}

// extractFailurePatterns identifies common failure patterns from QC output using keyword matching.
// Only extracts patterns for RED verdicts. Returns empty slice for GREEN/YELLOW verdicts.
func extractFailurePatterns(verdict, feedback, output string) []string {
	// Only extract patterns for RED verdicts
	if verdict != models.StatusRed {
		return []string{}
	}

	patterns := []string{}
	patternKeywords := map[string][]string{
		"compilation_error": {
			"compilation error", "compilation fail", "syntax error", // Original (more specific)
			"build fail", "build error", "parse error", // NEW
			"code won't compile", "unable to build", // NEW
			"compilation failed", // NEW - needed for test cases
		},
		"test_failure": {
			"test fail", "tests fail", "test failure", // Original
			"assertion fail", "verification fail", // NEW
			"check fail", "validation fail", // NEW
		},
		"dependency_missing": {
			"dependency", "package not found", "module not found", // Original
			"unable to locate", "missing package", // NEW
			"import error", "cannot find module", // NEW
		},
		"permission_error": {
			"permission", "access denied", "forbidden", // Original
			"unauthorized", // NEW
		},
		"timeout": {
			"timeout", "deadline", "timed out", // Original
			"request timeout", "execution timeout", // NEW
			"deadline exceeded", // NEW
		},
		"runtime_error": {
			"runtime error", "panic", "segfault", "nil pointer", // Original
			"null reference", "stack overflow", // NEW
			"segmentation fault", // NEW
		},
	}

	// Combine all text sources for pattern matching (case insensitive)
	combinedText := strings.ToLower(feedback + " " + output)

	// Check each pattern
	for pattern, keywords := range patternKeywords {
		for _, keyword := range keywords {
			if strings.Contains(combinedText, strings.ToLower(keyword)) {
				patterns = append(patterns, pattern)
				break // Only add pattern once
			}
		}
	}

	return patterns
}

// qcReviewHook runs after QC review to extract patterns and store metadata.
// This hook captures failure patterns for the post-task hook to use.
func (te *DefaultTaskExecutor) qcReviewHook(ctx context.Context, task *models.Task, verdict, feedback, output string) {
	// Extract failure patterns from QC output
	patterns := extractFailurePatterns(verdict, feedback, output)

	// Initialize metadata map if needed
	if task.Metadata == nil {
		task.Metadata = make(map[string]interface{})
	}

	// Store QC verdict and patterns in task metadata
	task.Metadata["qc_verdict"] = verdict
	task.Metadata["failure_patterns"] = patterns

	// Log patterns at debug level (would use proper logger in production)
	// For now, this is a no-op, but in production we'd log:
	// log.Debug("QC verdict: %s, patterns: %v", verdict, patterns)
}

// postTaskHook records complete execution history to the learning database.
// Called after task completion (success or failure) with all execution details.
func (te *DefaultTaskExecutor) postTaskHook(ctx context.Context, task *models.Task, result *models.TaskResult, verdict string) {
	// Learning disabled - no-op
	if te.learningStore == nil {
		return
	}

	// Extract failure patterns from metadata if present
	var failurePatterns []string
	if task.Metadata != nil {
		if patterns, ok := task.Metadata["failure_patterns"].([]string); ok {
			failurePatterns = patterns
		}
	}

	// Extract QC feedback if present
	qcFeedback := ""
	if result != nil {
		qcFeedback = result.ReviewFeedback
	}

	// Determine success status
	success := verdict == models.StatusGreen || verdict == models.StatusYellow

	// Extract error message if task failed
	errorMessage := ""
	if result != nil && result.Error != nil {
		errorMessage = result.Error.Error()
	}

	// Calculate duration in seconds
	durationSecs := int64(0)
	if result != nil {
		durationSecs = int64(result.Duration.Seconds())
	}

	// Extract output
	output := ""
	if result != nil {
		output = result.Output
	}

	// Build TaskExecution record
	exec := &TaskExecution{
		PlanFile:        te.planFile,
		RunNumber:       te.runNumber,
		TaskNumber:      task.Number,
		TaskName:        task.Name,
		Agent:           task.Agent,
		Prompt:          task.Prompt,
		Success:         success,
		Output:          output,
		ErrorMessage:    errorMessage,
		DurationSecs:    durationSecs,
		QCVerdict:       verdict,
		QCFeedback:      qcFeedback,
		FailurePatterns: failurePatterns,
	}

	// Record execution (graceful degradation on error)
	if err := te.learningStore.RecordExecution(ctx, exec); err != nil {
		// Log warning but don't fail task (graceful degradation)
		// In production: log.Warn("failed to record execution: %v", err)
		// For now, silently continue
	}
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

	// Pre-task hook: Query learning database and adapt agent/prompt
	if err := te.preTaskHook(ctx, &task); err != nil {
		// Hook errors are non-fatal but should be logged
		// For now, continue without learning adaptation
	}

	// Apply default agent if provided and task still has no agent.
	if task.Agent == "" && te.cfg.DefaultAgent != "" {
		task.Agent = te.cfg.DefaultAgent
		result.Task.Agent = te.cfg.DefaultAgent
	}

	// Update result task to reflect any changes from hook
	result.Task = task

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
				te.postTaskHook(ctx, &task, &result, models.StatusFailed)
				return result, err
			}
			te.postTaskHook(ctx, &task, &result, models.StatusGreen)
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
			// Call QC review hook to extract patterns and store metadata
			te.qcReviewHook(ctx, &task, review.Flag, review.Feedback, output)
			// Update result task to include metadata
			result.Task = task
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
				te.postTaskHook(ctx, &task, &result, models.StatusFailed)
				return result, err
			}
			te.postTaskHook(ctx, &task, &result, models.StatusGreen)
			return result, nil
		case review.Flag == models.StatusYellow:
			result.Status = models.StatusYellow
			result.RetryCount = attempt
			if err := te.updatePlanStatus(task, StatusCompleted, true); err != nil {
				result.Status = models.StatusFailed
				result.Error = err
				te.postTaskHook(ctx, &task, &result, models.StatusFailed)
				return result, err
			}
			te.postTaskHook(ctx, &task, &result, models.StatusYellow)
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
			te.postTaskHook(ctx, &task, &result, models.StatusRed)
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
	te.postTaskHook(ctx, &task, &result, models.StatusRed)
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
