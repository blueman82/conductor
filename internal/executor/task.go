package executor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/harrison/conductor/internal/agent"
	"github.com/harrison/conductor/internal/architecture"
	"github.com/harrison/conductor/internal/budget"
	"github.com/harrison/conductor/internal/config"
	"github.com/harrison/conductor/internal/learning"
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

// ErrRateLimitExit indicates task stopped due to rate limit with save-and-exit
type ErrRateLimitExit struct {
	ResumeAt time.Time
	StateID  string
}

func (e *ErrRateLimitExit) Error() string {
	return fmt.Sprintf("rate limited: saved state for resume at %s (ID: %s)", e.ResumeAt.Format(time.RFC3339), e.StateID)
}

// Reviewer is implemented by quality control components capable of reviewing task output.
type Reviewer interface {
	Review(ctx context.Context, task models.Task, output string) (*ReviewResult, error)
	ShouldRetry(result *ReviewResult, currentAttempt int) bool
}

// LearningStore defines the interface for adaptive learning storage.
type LearningStore interface {
	AnalyzeFailures(ctx context.Context, planFile, taskNumber string, minFailures int) (*learning.FailureAnalysis, error)
	RecordExecution(ctx context.Context, exec *learning.TaskExecution) error
	GetExecutionHistory(ctx context.Context, planFile, taskNumber string) ([]*learning.TaskExecution, error)
}

// RuntimeEnforcementLogger logs runtime enforcement events (test commands, criterion verification).
// LogErrorPattern accepts an error pattern display (ErrorPattern implements logger.ErrorPatternDisplay).
// LogDetectedError accepts a detected error display (DetectedError implements logger.DetectedErrorDisplay).
type RuntimeEnforcementLogger interface {
	LogTestCommands(entries []models.TestCommandResult)
	LogCriterionVerifications(entries []models.CriterionVerificationResult)
	LogDocTargetVerifications(entries []models.DocTargetResult)
	LogErrorPattern(pattern interface{})      // Flexible interface for testing (deprecated, use LogDetectedError)
	LogDetectedError(detected interface{})    // Flexible interface for testing (v2.12+)
	Warnf(format string, args ...interface{}) // Warning messages
	Info(message string)                      // Info messages
	Infof(format string, args ...interface{}) // Formatted info messages
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
	invoker                     InvokerInterface
	reviewer                    Reviewer
	planUpdater                 PlanUpdater
	cfg                         TaskExecutorConfig
	clock                       func() time.Time
	qcEnabled                   bool
	retryLimit                  int
	SourceFile                  string                   // Track which file this task comes from
	FileLockManager             FileLockManager          // Per-file locking strategy
	LearningStore               LearningStore            // Adaptive learning store (optional)
	PlanFile                    string                   // Plan file path for learning queries
	SessionID                   string                   // Session ID for learning tracking
	RunNumber                   int                      // Run number for learning tracking
	metrics                     *learning.PatternMetrics // Pattern detection metrics (optional)
	AutoAdaptAgent              bool                     // Enable automatic agent adaptation
	MinFailuresBeforeAdapt      int                      // Minimum failures before adapting agent
	SwapDuringRetries           bool                     // Enable inter-retry agent swapping
	Plan                        *models.Plan             // Plan reference for integration prompt builder
	EnforceDependencyChecks     bool                     // Run dependency checks before task invocation
	CommandRunner               CommandRunner            // Command runner for dependency checks (optional)
	WorkDir                     string                   // Working directory for dependency check commands
	EnforceTestCommands         bool                     // Run test commands after agent output (v2.9+)
	VerifyCriteria              bool                     // Run optional per-criterion verifications (v2.9+)
	EnforceDocTargets           bool                     // Run documentation target verification for doc tasks (v2.9+)
	EnableErrorPatternDetection bool                     // Enable error pattern detection on test failures (v2.11+)
	EnableClaudeClassification  bool                     // Enable Claude-based error classification (v2.11+)
	Logger                      RuntimeEnforcementLogger // Logger for runtime enforcement output (optional)
	EventLogger                 Logger                   // Logger for execution events (task agent invoke, etc.)
	TaskAgentSelector           *TaskAgentSelector       // Intelligent agent selector for task execution (v2.15+)
	IntelligentAgentSelection   bool                     // Enable intelligent agent selection when task.Agent is empty
	BudgetConfig                *config.BudgetConfig     // Budget tracking configuration (v2.19+)

	// Intelligent rate limit handling (v2.20+)
	Waiter       *budget.RateLimitWaiter // Smart wait with countdown
	StateManager *budget.StateManager    // Execution state persistence

	// Pattern Intelligence integration (v2.23+)
	PatternHook *PatternIntelligenceHook // Pattern Intelligence hook for STOP protocol and duplicate detection

	// Architecture Checkpoint integration (v2.27+)
	ArchitectureHook *ArchitectureCheckpointHook // Architecture checkpoint hook for 6-question assessment

	// LIP Collection integration (v2.29+)
	LIPCollectorHook *LIPCollectorHook // LIP event collector for test/build results and knowledge graph

	// Warm-Up Context integration (v2.29+)
	WarmUpHook *WarmUpHook // Warm-up context hook for agent priming with historical patterns

	// Intelligent Agent Swap integration (v2.30+)
	IntelligentAgentSwapper *learning.IntelligentAgentSwapper // Claude-powered agent selection for retries (optional)

	// Runtime state for passing to QC
	lastTestResults      []TestCommandResult            // Populated after RunTestCommands
	lastCriterionResults []CriterionVerificationResult  // Populated after RunCriterionVerifications
	lastDocTargetResults []DocTargetResult              // Populated after VerifyDocumentationTargets
	lastPatternResult    *PreTaskCheckResult            // Populated after Pattern Intelligence check (v2.24+)
	lastArchResult       *architecture.CheckpointResult // Populated after Architecture checkpoint (v2.27+)
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
		invoker:                invoker,
		reviewer:               reviewer,
		planUpdater:            planUpdater,
		cfg:                    cfg,
		clock:                  time.Now,
		FileLockManager:        NewFileLockManager(),
		MinFailuresBeforeAdapt: 2, // Default threshold
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
			// Wire up multi-agent QC configuration (v2.2+)
			if cfg.QualityControl.Agents.Mode != "" {
				qc.AgentConfig = cfg.QualityControl.Agents
			}
			qc.MaxRetries = te.retryLimit
			te.reviewer = qc
		} else if qc, ok := te.reviewer.(*QualityController); ok {
			if cfg.QualityControl.ReviewAgent != "" {
				qc.ReviewAgent = cfg.QualityControl.ReviewAgent
			}
			// Wire up multi-agent QC configuration (v2.2+)
			if cfg.QualityControl.Agents.Mode != "" {
				qc.AgentConfig = cfg.QualityControl.Agents
			}
			qc.MaxRetries = te.retryLimit
		}
	}

	return te, nil
}

// preTaskHook queries the learning database and adapts agent/prompt before execution.
// This hook enables adaptive learning from past failures and warm-up context injection.
func (te *DefaultTaskExecutor) preTaskHook(ctx context.Context, task *models.Task) error {
	// Learning disabled - no-op
	if te.LearningStore == nil {
		return nil
	}

	// Query learning store for failure analysis with configurable threshold
	analysis, err := te.LearningStore.AnalyzeFailures(ctx, te.PlanFile, task.Number, te.MinFailuresBeforeAdapt)
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

	// Adapt agent if auto-adaptation is enabled and recommended
	if te.AutoAdaptAgent && analysis.ShouldTryDifferentAgent && analysis.SuggestedAgent != "" {
		// Only switch if different from current agent
		if task.Agent != analysis.SuggestedAgent {
			// Store original agent for logging
			originalAgent := task.Agent
			if originalAgent == "" {
				originalAgent = "default"
			}

			// Log agent switch for observability
			// In production: log.Info("Switching agent: %s → %s based on failure patterns", originalAgent, analysis.SuggestedAgent)
			task.Agent = analysis.SuggestedAgent
		}
	}

	// Enhance prompt with learning context if there are past failures
	if analysis.FailedAttempts > 0 {
		task.Prompt = enhancePromptWithLearning(task.Prompt, analysis)
	}

	// Warm-up context injection (v2.29+)
	// Injects similar successful task approaches, common pitfalls, and file-specific patterns
	// This happens AFTER failure analysis to avoid duplicating failure context
	if te.WarmUpHook != nil {
		modifiedTask, warmUpErr := te.WarmUpHook.InjectContext(ctx, *task)
		if warmUpErr != nil {
			// Graceful degradation - log but don't fail
			if te.Logger != nil {
				te.Logger.Warnf("WarmUp: failed to inject context for task %s: %v", task.Number, warmUpErr)
			}
		} else {
			// Apply the modified task (with injected prompt)
			*task = modifiedTask
		}
	}

	return nil
}

// enhancePromptWithLearning adds learning context to the task prompt.
func enhancePromptWithLearning(originalPrompt string, analysis *learning.FailureAnalysis) string {
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
			"compilation error", "compilation fail",
			"build fail", "build error", "parse error",
			"code won't compile", "unable to build",
			"compilation failed",
		},
		"test_failure": {
			"test fail", "tests fail", "test failure",
			"assertion fail", "verification fail",
			"check fail", "validation fail",
		},
		"dependency_missing": {
			"dependency", "package not found", "module not found",
			"unable to locate", "missing package",
			"import error", "cannot find module",
		},
		"permission_error": {
			"permission", "access denied", "forbidden",
			"unauthorized",
		},
		"timeout": {
			"timeout", "deadline", "timed out",
			"request timeout", "execution timeout",
			"deadline exceeded",
		},
		"runtime_error": {
			"runtime error", "panic", "segfault", "nil pointer",
			"null reference", "stack overflow",
			"segmentation fault",
		},
		"syntax_error": {
			"syntax error", "syntax_error",
			"syntax fail",
		},
		"type_error": {
			"type error", "type_error",
			"type mismatch", "type fail",
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

	// Record metrics if metrics collector available
	if te.metrics != nil {
		te.metrics.RecordExecution()
		for _, pattern := range patterns {
			te.metrics.RecordPatternDetection(pattern, []string{})
		}
	}

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
	if te.LearningStore == nil {
		return
	}

	// Determine which file to query for deduplication
	fileToQuery := te.PlanFile
	if task.SourceFile != "" {
		fileToQuery = task.SourceFile
	} else if te.SourceFile != "" {
		fileToQuery = te.SourceFile
	}

	// Check if this final outcome was already recorded during retry loop
	// Query most recent execution for this task
	history, err := te.LearningStore.GetExecutionHistory(ctx, fileToQuery, task.Number)
	if err == nil && len(history) > 0 {
		lastExec := history[0]
		// Check if last execution matches this final state (same verdict + run number)
		// If so, it was already recorded during retry - skip duplicate
		if lastExec.QCVerdict == verdict && lastExec.RunNumber == te.RunNumber {
			// Already recorded during retry loop - skip to avoid duplicate
			return
		}
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
	exec := &learning.TaskExecution{
		PlanFile:        fileToQuery,
		RunNumber:       te.RunNumber,
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
	if err := te.LearningStore.RecordExecution(ctx, exec); err != nil {
		// Log warning but don't fail task (graceful degradation)
		// In production: log.Warn("failed to record execution: %v", err)
		// For now, silently continue
	}

	// LIP Collection: Create knowledge graph edges in post-task hook (v2.29+)
	// Records: task→succeeded_with→agent (or used_by for failures)
	if te.LIPCollectorHook != nil {
		te.LIPCollectorHook.PostTaskHook(ctx, *task, result, success)
	}
}

// collectBehavioralMetrics extracts metrics from a session JSONL file and stores in database
// Returns nil if session file not found (graceful degradation)
func (te *DefaultTaskExecutor) collectBehavioralMetrics(ctx context.Context, sessionID string, task models.Task, taskExecutionID int64) error {
	// Skip if no learning store
	if te.LearningStore == nil {
		return nil
	}

	// Convert to concrete type to access database
	store, ok := te.LearningStore.(*learning.Store)
	if !ok {
		return nil // Graceful degradation
	}

	// Create behavior collector
	collector := NewBehaviorCollector(store, "")

	// Load session metrics
	metrics, err := collector.LoadSessionMetrics(sessionID)
	if err != nil {
		// Log warning but don't fail task - graceful degradation
		fmt.Fprintf(os.Stderr, "Warning: failed to load behavioral metrics for session %s: %v\n", sessionID, err)
		return nil
	}

	// Store metrics in database
	if err := collector.StoreSessionMetrics(ctx, taskExecutionID, metrics); err != nil {
		// Log warning but don't fail task - graceful degradation
		fmt.Fprintf(os.Stderr, "Warning: failed to store behavioral metrics: %v\n", err)
		return nil
	}

	return nil
}

// updateFeedback stores execution attempt feedback to the plan file.
// This is called ONLY after QC review completes, when we have the full context
// (agent output + verdict + QC feedback). We do NOT call this before QC review
// to avoid creating duplicate execution history entries.
func (te *DefaultTaskExecutor) updateFeedback(task models.Task, attempt int, agentOutput, qcFeedback, verdict string) error {
	// Only update if we have a plan file configured
	if te.cfg.PlanPath == "" {
		return nil
	}

	// Determine which file to update (same logic as updatePlanStatus)
	fileToUpdate := te.cfg.PlanPath
	if task.SourceFile != "" {
		fileToUpdate = task.SourceFile
	} else if te.SourceFile != "" {
		fileToUpdate = te.SourceFile
	}

	// Create execution attempt record
	execAttempt := &updater.ExecutionAttempt{
		AttemptNumber: attempt,
		Agent:         task.Agent,
		Verdict:       verdict,
		AgentOutput:   agentOutput,
		QCFeedback:    qcFeedback,
		Timestamp:     time.Now().UTC(),
	}

	// Update plan file with execution history (graceful degradation on error)
	if err := updater.UpdateTaskFeedback(fileToUpdate, task.Number, execAttempt); err != nil {
		// Log warning but don't fail task execution
		// In production: log.Warnf("failed to update plan feedback: %v", err)
		return nil // Non-fatal error
	}

	return nil
}

// isRateLimitError checks if an error indicates a Claude API rate limit.
// Matches common rate limit error patterns from Claude CLI.
func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}

	// Check for ErrRateLimit type from agent package (v2.20.1+)
	if strings.HasPrefix(err.Error(), "rate limit:") {
		return true
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "rate limit") ||
		strings.Contains(msg, "out of") && strings.Contains(msg, "usage") ||
		strings.Contains(msg, "429") ||
		strings.Contains(msg, "too many requests") ||
		strings.Contains(msg, "quota exceeded") ||
		strings.Contains(msg, "rate_limit_error")
}

// executeWithRateLimitRecovery wraps task execution with intelligent rate limit handling.
// If rate limited:
//   - Parses actual reset time from CLI output
//   - If wait <= MaxWaitDuration: waits with countdown announcements
//   - If wait > MaxWaitDuration: saves state and returns ErrRateLimitExit
//   - On retry: uses --resume with session ID if available, or injects git diff context
func (te *DefaultTaskExecutor) executeWithRateLimitRecovery(ctx context.Context, task models.Task) (models.TaskResult, error) {
	var lastSessionID string // Track session ID from previous attempt for --resume
	isRetry := false         // Track if this is a retry after rate limit

	for {
		// Prepare task for execution (potentially with resume context)
		taskToExecute := task
		if isRetry {
			// Only inject context on retry, not first execution
			if lastSessionID != "" {
				// Primary: Resume previous session via Claude CLI --resume flag
				taskToExecute.ResumeSessionID = lastSessionID
				if te.Logger != nil {
					te.Logger.Infof("Resuming session %s after rate limit", lastSessionID)
				}
			} else if te.hasGitChanges() {
				// Fallback: Inject git diff context if no session ID but partial work exists
				diffContext := te.getGitDiffContext()
				if diffContext != "" {
					taskToExecute.Prompt = task.Prompt + diffContext
					if te.Logger != nil {
						te.Logger.Info("Injecting git diff context for rate limit retry (no session ID)")
					}
				}
			}
		}

		result, err := te.executeTask(ctx, taskToExecute)

		// Capture session ID for potential retry (from InvocationResult via TaskResult)
		if result.SessionID != "" {
			lastSessionID = result.SessionID
		}

		// Check for rate limit error
		if !isRateLimitError(err) {
			return result, err
		}

		// Parse actual reset time from error message
		info := budget.ParseRateLimitFromOutput(err.Error())
		if info == nil {
			// Fallback: use inferred reset time (5-hour window)
			info = &budget.RateLimitInfo{
				DetectedAt: time.Now(),
				ResetAt:    budget.InferResetTime(),
				LimitType:  budget.LimitTypeSession,
				RawMessage: err.Error(),
				Source:     "error",
			}
		}

		// Check if we have a waiter configured
		if te.Waiter == nil {
			// No waiter - fail immediately (backward compat)
			return result, err
		}

		// Check if we should wait or save-and-exit
		if !te.Waiter.ShouldWait(info) {
			// Wait too long - save state and exit
			if te.StateManager != nil {
				state := &budget.ExecutionState{
					SessionID:     budget.GenerateSessionID(),
					PlanFile:      te.PlanFile,
					RateLimitInfo: info,
					CurrentWave:   0, // Will be set by wave executor
					PausedAt:      time.Now(),
					ResumeAt:      info.ResetAt,
					Status:        budget.StatusPaused,
				}
				if saveErr := te.StateManager.Save(state); saveErr != nil {
					// Log warning but still return the rate limit error
					if te.Logger != nil {
						te.Logger.Warnf("Failed to save execution state: %v", saveErr)
					}
				}
				return result, &ErrRateLimitExit{
					ResumeAt: info.ResetAt,
					StateID:  state.SessionID,
				}
			}
			return result, err
		}

		// Log the wait
		if te.EventLogger != nil {
			te.EventLogger.LogRateLimitPause(info.TimeUntilReset())
		}

		// Wait for reset with countdown announcements
		if waitErr := te.Waiter.WaitForReset(ctx, info); waitErr != nil {
			// Context cancelled during wait
			result.Status = models.StatusFailed
			result.Error = waitErr
			return result, waitErr
		}

		// Log resume
		if te.EventLogger != nil {
			te.EventLogger.LogRateLimitResume()
		}

		// Mark as retry so context injection happens on next iteration
		isRetry = true
		// Retry the task (loop continues with lastSessionID set)
	}
}

// hasGitChanges checks if there are uncommitted changes in the working directory
func (te *DefaultTaskExecutor) hasGitChanges() bool {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = te.WorkDir
	output, err := cmd.Output()
	return err == nil && len(output) > 0
}

// getGitDiffContext returns a formatted git diff summary for injection into retry prompts
func (te *DefaultTaskExecutor) getGitDiffContext() string {
	cmd := exec.Command("git", "diff", "--stat", "HEAD")
	cmd.Dir = te.WorkDir
	output, err := cmd.Output()
	if err != nil || len(output) == 0 {
		return ""
	}

	// Truncate if too large (keep under 2KB to avoid bloating prompt)
	diffStr := string(output)
	if len(diffStr) > 2048 {
		diffStr = diffStr[:2048] + "\n... (truncated)"
	}

	return fmt.Sprintf(`

---
## RATE LIMIT RECOVERY - PARTIAL PROGRESS DETECTED
The following files were modified before the rate limit. DO NOT re-apply changes that already exist.
Review the current state and continue from where you left off.

%s
---`, diffStr)
}

// Execute runs an individual task, handling agent invocation, quality control, and plan updates.
func (te *DefaultTaskExecutor) Execute(ctx context.Context, task models.Task) (models.TaskResult, error) {
	// If budget auto-resume is enabled, use intelligent recovery wrapper
	if te.BudgetConfig != nil && te.BudgetConfig.Enabled && te.BudgetConfig.AutoResume {
		return te.executeWithRateLimitRecovery(ctx, task)
	}

	// Standard execution path
	return te.executeTask(ctx, task)
}

// executeTask is the core task execution logic (refactored from original Execute).
func (te *DefaultTaskExecutor) executeTask(ctx context.Context, task models.Task) (models.TaskResult, error) {
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
	// Sync LearningStore with QualityController for historical context loading
	if te.LearningStore != nil && te.reviewer != nil {
		if qc, ok := te.reviewer.(*QualityController); ok {
			if store, ok := te.LearningStore.(*learning.Store); ok {
				qc.LearningStore = store
			}
		}
	}

	// Pre-task hook: Query learning database and adapt agent/prompt
	if err := te.preTaskHook(ctx, &task); err != nil {
		// Hook errors are non-fatal but should be logged
		// For now, continue without learning adaptation
	}

	// Pattern Intelligence pre-task hook: STOP protocol analysis and duplicate detection (v2.23+)
	te.lastPatternResult = nil // Reset for new task
	if te.PatternHook != nil {
		patternResult, patternErr := te.PatternHook.CheckTask(ctx, task)
		if patternErr != nil {
			// Graceful degradation: log but don't block
			if te.Logger != nil {
				te.Logger.Warnf("Pattern Intelligence check failed for task %s: %v", task.Number, patternErr)
			}
		} else if patternResult != nil {
			// Store for QC integration (v2.24+)
			te.lastPatternResult = patternResult

			// Handle block mode - task should not proceed
			if patternResult.ShouldBlock {
				result.Status = models.StatusFailed
				result.Error = fmt.Errorf("pattern intelligence blocked: %s", patternResult.BlockReason)
				_ = te.updatePlanStatus(task, StatusFailed, false)
				return result, result.Error
			}

			// Apply prompt injection (warn/suggest modes)
			if patternResult.PromptInjection != "" {
				task = ApplyPromptInjection(task, patternResult)
			}
		}
	}

	// Architecture Checkpoint pre-task hook: 6-question assessment (v2.27+)
	te.lastArchResult = nil // Reset for new task
	if te.ArchitectureHook != nil {
		archResult, archErr := te.ArchitectureHook.CheckTask(ctx, task)
		if archErr != nil {
			// Graceful degradation: log but don't block
			if te.Logger != nil {
				te.Logger.Warnf("Architecture checkpoint failed for task %s: %v", task.Number, archErr)
			}
		} else if archResult != nil {
			// Store for QC integration (v2.27+)
			te.lastArchResult = archResult

			// Handle block mode - task should not proceed
			if archResult.ShouldBlock {
				result.Status = models.StatusFailed
				result.Error = fmt.Errorf("architecture checkpoint blocked: %s", archResult.BlockReason)
				_ = te.updatePlanStatus(task, StatusFailed, false)
				return result, result.Error
			}

			// Handle escalate mode - for now, log and continue (user prompt TBD)
			if archResult.ShouldEscalate {
				if te.Logger != nil {
					te.Logger.Warnf("Architecture checkpoint requires escalation for task %s", task.Number)
				}
			}

			// Apply prompt injection (warn/escalate modes)
			if archResult.PromptInjection != "" {
				task.Prompt = task.Prompt + archResult.PromptInjection
			}
		}
	}

	// Run dependency checks before agent invocation (v2.9+)
	if te.EnforceDependencyChecks && task.RuntimeMetadata != nil && len(task.RuntimeMetadata.DependencyChecks) > 0 {
		runner := te.CommandRunner
		if runner == nil {
			runner = NewShellCommandRunner(te.WorkDir)
		}
		if err := RunDependencyChecks(ctx, runner, task); err != nil {
			result.Status = models.StatusFailed
			result.Error = fmt.Errorf("preflight dependency check failed: %w", err)
			_ = te.updatePlanStatus(task, StatusFailed, false)
			return result, result.Error
		}
	}

	// Apply integration context to integration tasks AND any task with dependencies
	// This helps all dependent tasks understand their dependencies, not just explicit integration tasks
	// Must be done BEFORE agent invocation to inject file context
	if te.Plan != nil && (task.Type == "integration" || len(task.DependsOn) > 0) {
		task.Prompt = buildIntegrationPrompt(task, te.Plan)
	}

	// Apply intelligent agent selection if enabled and task has no agent
	if task.Agent == "" && te.IntelligentAgentSelection && te.TaskAgentSelector != nil {
		selResult, err := te.TaskAgentSelector.SelectAgent(ctx, task)
		if err == nil && selResult.Agent != "" {
			task.Agent = selResult.Agent
			result.Task.Agent = selResult.Agent
			// Log the selection rationale
			if te.Logger != nil {
				te.Logger.Infof("[Task] Intelligent agent selection: %s - %s", selResult.Agent, selResult.Rationale)
			}
		}
		// If intelligent selection fails, fall through to default agent
	}

	// Apply default agent if provided and task still has no agent.
	if task.Agent == "" && te.cfg.DefaultAgent != "" {
		task.Agent = te.cfg.DefaultAgent
		result.Task.Agent = te.cfg.DefaultAgent
	}

	// Update result task to reflect any changes from hook
	result.Task = task

	// Announce agent deployment (after any agent swaps/defaults have been applied)
	if te.EventLogger != nil {
		te.EventLogger.LogTaskAgentInvoke(task)
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

	// Track execution history for all attempts
	var executionHistory []models.ExecutionAttempt

	// Track test failure state for retry injection (v2.10+)
	var testFailureErr error

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
			} else if parsedOutput.Content == "" && parsedOutput.Error == "" {
				// Both fields are empty strings - this means JSON was parsed successfully
				// but both fields were empty. Use empty string instead of raw JSON.
				output = ""
			}
			// If parsed output is nil or JSON parsing failed, fall back to original output for QC review
			// This ensures QC always has something to review
		}

		result.Output = output
		result.Duration = totalDuration
		result.SessionID = invocation.SessionID // Capture for rate limit recovery

		// Track task execution ID for LIP event collection (v2.29+)
		var currentTaskExecutionID int64

		// Collect behavioral metrics post-invocation (before QC)
		// This captures the session_id from the invocation and loads JSONL metrics
		if invocation.SessionID != "" {
			// We need the task_execution_id to link behavioral data
			// Store attempt execution first to get the ID
			if te.LearningStore != nil {
				fileToRecord := te.PlanFile
				if task.SourceFile != "" {
					fileToRecord = task.SourceFile
				} else if te.SourceFile != "" {
					fileToRecord = te.SourceFile
				}

				// Create preliminary execution record (before QC)
				prelimExec := &learning.TaskExecution{
					PlanFile:     fileToRecord,
					RunNumber:    te.RunNumber,
					TaskNumber:   task.Number,
					TaskName:     task.Name,
					Agent:        task.Agent,
					Prompt:       task.Prompt,
					Success:      false,  // Will be updated after QC
					Output:       output, // Store parsed agent content (not raw CLI wrapper)
					ErrorMessage: "",
					DurationSecs: int64(invocation.Duration.Seconds()),
					QCVerdict:    "", // Not yet determined
					QCFeedback:   "",
				}

				// Record to get task_execution_id
				if err := te.LearningStore.RecordExecution(ctx, prelimExec); err == nil {
					currentTaskExecutionID = prelimExec.ID // Store for LIP event collection
					// Collect behavioral metrics with the task_execution_id
					_ = te.collectBehavioralMetrics(ctx, invocation.SessionID, task, prelimExec.ID)
				}
			}
		}

		// Initialize execution attempt record
		execAttempt := models.ExecutionAttempt{
			Attempt:     attempt + 1, // 1-indexed
			Agent:       task.Agent,
			AgentOutput: output, // Store parsed agent content (not raw CLI wrapper)
			Duration:    invocation.Duration,
		}

		// Clear test failure from previous attempt
		testFailureErr = nil

		// Run test commands after agent output but BEFORE QC (v2.9+)
		// Test command failure is tracked but doesn't return immediately (v2.10+)
		if te.EnforceTestCommands && len(task.TestCommands) > 0 {
			runner := te.CommandRunner
			if runner == nil {
				runner = NewShellCommandRunner(te.WorkDir)
			}
			testResults, testErr := RunTestCommands(ctx, runner, task)
			te.lastTestResults = testResults // Store for QC prompt injection
			if te.Logger != nil {
				te.Logger.LogTestCommands(testResults)
			}

			// LIP Collection: Record test results as LIP events (v2.29+)
			// This is a NEW event type - tool/file events are already tracked via RecordSessionMetrics
			if te.LIPCollectorHook != nil && currentTaskExecutionID > 0 {
				_ = te.LIPCollectorHook.RecordTestResults(ctx, currentTaskExecutionID, task.Number, testResults)
			}

			if testErr != nil {
				// Track failure but continue to allow retry (v2.10+)
				testFailureErr = testErr
			}
		}

		// Run optional per-criterion verifications (v2.9+)
		// Verification failures do NOT block - they feed into QC prompt
		if te.VerifyCriteria && len(task.StructuredCriteria) > 0 {
			runner := te.CommandRunner
			if runner == nil {
				runner = NewShellCommandRunner(te.WorkDir)
			}
			criterionResults, verifyErr := RunCriterionVerifications(ctx, runner, task)
			te.lastCriterionResults = criterionResults // Store for QC prompt injection
			if te.Logger != nil {
				te.Logger.LogCriterionVerifications(criterionResults)
			}
			if verifyErr != nil && errors.Is(verifyErr, context.Canceled) {
				// Only fail on context cancellation, not on verification failures
				result.Status = models.StatusFailed
				result.Error = verifyErr
				_ = te.updatePlanStatus(task, StatusFailed, false)
				return result, verifyErr
			}
		}

		// Run documentation target verification for documentation tasks (v2.9+)
		// Verification failures do NOT block - they feed into QC prompt
		if te.EnforceDocTargets {
			// Documentation tasks without targets MUST fail validation (criterion #3)
			if IsDocumentationTask(task) && !HasDocumentationTargets(task) {
				result.Status = models.StatusFailed
				result.Error = fmt.Errorf("documentation task %s missing required documentation_targets metadata", task.Number)
				_ = te.updatePlanStatus(task, StatusFailed, false)
				return result, result.Error
			}

			if HasDocumentationTargets(task) {
				docResults, docErr := VerifyDocumentationTargets(ctx, task)
				te.lastDocTargetResults = docResults // Store for QC prompt injection
				if te.Logger != nil {
					te.Logger.LogDocTargetVerifications(docResults)
				}
				if docErr != nil && errors.Is(docErr, context.Canceled) {
					// Only fail on context cancellation, not on verification failures
					result.Status = models.StatusFailed
					result.Error = docErr
					_ = te.updatePlanStatus(task, StatusFailed, false)
					return result, docErr
				}
			}
		}

		// Handle test failures (v2.10+)
		// Test failures are treated like QC RED verdicts - retry with feedback injection
		if testFailureErr != nil {
			if !te.qcEnabled || te.reviewer == nil {
				// No QC enabled - fail immediately with no retry (backward compat: hard gate)
				result.Status = models.StatusFailed
				result.Error = fmt.Errorf("test command failed: %w", testFailureErr)
				_ = te.updatePlanStatus(task, StatusFailed, false)
				return result, result.Error
			}

			// QC enabled - treat test failure like RED verdict
			lastErr = fmt.Errorf("test command failed: %w", testFailureErr)
			result.RetryCount = attempt

			// Check retry budget
			if attempt >= te.retryLimit {
				result.Status = models.StatusRed
				result.Error = lastErr
				_ = te.updatePlanStatus(task, StatusFailed, false)
				te.postTaskHook(ctx, &task, &result, models.StatusRed)
				return result, lastErr
			}

			// Detect error patterns before injecting feedback (v2.11+)
			if te.EnableErrorPatternDetection {
				for _, result := range te.lastTestResults {
					if !result.Passed {
						// Pass invoker + enableClaude flag for Claude classification (v2.11+)
						detected := DetectErrorPattern(result.Output, te.invoker, te.EnableClaudeClassification)
						if detected != nil {
							// Store full DetectedError (v2.12+)
							if task.Metadata == nil {
								task.Metadata = make(map[string]interface{})
							}
							if task.Metadata["detected_errors"] == nil {
								task.Metadata["detected_errors"] = []*DetectedError{}
							}
							task.Metadata["detected_errors"] = append(
								task.Metadata["detected_errors"].([]*DetectedError),
								detected,
							)

							// Log with new method (v2.12+)
							if te.Logger != nil {
								te.Logger.LogDetectedError(detected)
							}
						}
					}
				}
			}

			// Check if retry is futile based on error classification (v2.12+)
			// CRITICAL: Check BEFORE building retry prompt to avoid wasted agent invocation
			if te.EnableErrorPatternDetection {
				detectedErrors := getDetectedErrors(&task)

				// Check if ALL detected errors require human intervention
				allRequireHuman := len(detectedErrors) > 0
				for _, detected := range detectedErrors {
					if detected != nil && detected.Pattern != nil && !detected.Pattern.RequiresHumanIntervention {
						allRequireHuman = false
						break
					}
				}

				// If all errors require human intervention, skip retry immediately
				if allRequireHuman {
					if te.Logger != nil {
						te.Logger.Warnf("Task %s: All detected errors require human intervention - skipping retry", task.Number)
						te.Logger.Info("Suggestions:")
						for i, detected := range detectedErrors {
							if detected != nil && detected.Pattern != nil {
								te.Logger.Infof("  %d. [%s] %s", i+1, detected.Pattern.Category.String(), detected.Pattern.Suggestion)
							}
						}
					}
					result.Status = models.StatusRed
					result.RetryCount = attempt
					result.Error = fmt.Errorf("human intervention required: all detected errors cannot be fixed by agent")
					_ = te.updatePlanStatus(task, StatusFailed, false)
					te.postTaskHook(ctx, &task, &result, models.StatusRed)
					return result, result.Error
				}
			}

			// Inject test failure feedback for retry (mirrors QC pattern)
			testFeedback := FormatTestResults(te.lastTestResults)
			if testFeedback != "" {
				// Build classification context from stored detected errors
				classificationContext := ""
				if task.Metadata != nil {
					if detectedErrors, ok := task.Metadata["detected_errors"].([]*DetectedError); ok && len(detectedErrors) > 0 {
						classificationContext = formatClassificationForRetry(detectedErrors)
					}
				}

				task.Prompt = fmt.Sprintf("%s\n\n---\n## PREVIOUS ATTEMPT FAILED - TEST COMMANDS FAILED (MUST FIX):\n%s\n%s\n---\n\nFix ALL test failures listed above before completing the task.",
					task.Prompt, testFeedback, classificationContext)
			}

			// Continue to next retry iteration
			continue
		}

		if !te.qcEnabled || te.reviewer == nil {
			// No QC - mark as GREEN and store in history
			execAttempt.Verdict = models.StatusGreen
			execAttempt.QCFeedback = "" // No QC feedback
			executionHistory = append(executionHistory, execAttempt)
			result.ExecutionHistory = executionHistory

			result.Status = models.StatusGreen
			result.RetryCount = attempt
			if err := te.updatePlanStatus(task, StatusCompleted, true); err != nil {
				result.Status = models.StatusFailed
				result.Error = err
				te.postTaskHook(ctx, &task, &result, models.StatusFailed)
				return result, err
			}
			te.postTaskHook(ctx, &task, &result, models.StatusGreen)

			// Pattern Intelligence post-task hook: Record successful pattern (v2.23+)
			if te.PatternHook != nil {
				_ = te.PatternHook.RecordSuccess(ctx, task, task.Agent)
			}

			return result, nil
		}

		// Pass test/verification results to QC before review (v2.9+)
		if qc, ok := te.reviewer.(*QualityController); ok {
			qc.TestCommandResults = te.lastTestResults
			qc.CriterionVerifyResults = te.lastCriterionResults
			qc.DocTargetResults = te.lastDocTargetResults

			// Wire STOP protocol context to QC (v2.24+)
			// This enables QC to request justification for custom implementations when prior art exists
			if te.lastPatternResult != nil && te.lastPatternResult.STOPResult != nil {
				qc.STOPSummary = BuildSTOPSummaryFromSTOPResult(te.lastPatternResult.STOPResult)
				qc.RequireJustification = te.PatternHook.RequireJustification()
			}
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
			// Store QC feedback in execution attempt
			execAttempt.QCFeedback = review.Feedback
			execAttempt.Verdict = review.Flag

			// Store QC feedback to plan file for this attempt (after QC review completes)
			// This is the ONLY call to updateFeedback - we skip the pre-QC call to avoid duplicates
			verdict := review.Flag
			qcFeedback := review.Feedback
			if err := te.updateFeedback(task, attempt+1, output, qcFeedback, verdict); err != nil {
				// Log warning but continue (non-fatal)
			}

			// Call QC review hook to extract patterns and store metadata
			te.qcReviewHook(ctx, &task, review.Flag, review.Feedback, output)
			// Update result task to include metadata
			result.Task = task

			// Record attempt to database immediately for next retry's QC context
			// This enables QC agents to see previous attempt feedback during retries
			if te.LearningStore != nil && review != nil {
				// Determine which file to use (multi-file plan support)
				fileToRecord := te.PlanFile
				if task.SourceFile != "" {
					fileToRecord = task.SourceFile
				} else if te.SourceFile != "" {
					fileToRecord = te.SourceFile
				}

				// Extract failure patterns from metadata (populated by qcReviewHook)
				var failurePatterns []string
				if task.Metadata != nil {
					if patterns, ok := task.Metadata["failure_patterns"].([]string); ok {
						failurePatterns = patterns
					}
				}

				// Determine success based on verdict
				attemptSuccess := review.Flag == models.StatusGreen || review.Flag == models.StatusYellow

				// Build execution record for this attempt
				attemptExec := &learning.TaskExecution{
					PlanFile:        fileToRecord,
					RunNumber:       te.RunNumber,
					TaskNumber:      task.Number,
					TaskName:        task.Name,
					Agent:           task.Agent,
					Prompt:          task.Prompt,
					Success:         attemptSuccess,
					Output:          output, // Store parsed agent content (not raw CLI wrapper)
					ErrorMessage:    "",     // No error if we reached QC
					DurationSecs:    int64(invocation.Duration.Seconds()),
					QCVerdict:       review.Flag,
					QCFeedback:      review.Feedback,
					FailurePatterns: failurePatterns,
				}

				// Record to database (graceful degradation on error)
				if err := te.LearningStore.RecordExecution(ctx, attemptExec); err != nil {
					// Log warning but don't fail task execution
					// In production: log.Warn("failed to record attempt to DB: %v", err)
				}
			}
		}

		// Append attempt to history
		executionHistory = append(executionHistory, execAttempt)
		result.ExecutionHistory = executionHistory

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

			// Pattern Intelligence post-task hook: Record successful pattern (v2.23+)
			if te.PatternHook != nil {
				_ = te.PatternHook.RecordSuccess(ctx, task, task.Agent)
			}

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

			// Track RED failures for agent swap
			redCount := attempt + 1

			// Agent swap during retries: if enabled and threshold reached
			if te.SwapDuringRetries && redCount >= te.MinFailuresBeforeAdapt {
				var newAgent string
				var swapReason string
				swapped := false

				// Use IntelligentAgentSwapper for context-aware agent selection (v2.29+)
				if te.IntelligentAgentSwapper != nil {
					swapCtx := &learning.SwapContext{
						TaskNumber:      task.Number,
						TaskName:        task.Name,
						TaskDescription: task.Prompt,
						Files:           extractTaskFiles(task),
						CurrentAgent:    task.Agent,
						ErrorContext:    review.Feedback,
						AttemptNumber:   attempt + 1,
					}
					recommendation, err := te.IntelligentAgentSwapper.SelectAgent(ctx, swapCtx)
					if err == nil && recommendation != nil && recommendation.RecommendedAgent != "" && recommendation.RecommendedAgent != task.Agent {
						newAgent = recommendation.RecommendedAgent
						swapReason = fmt.Sprintf("%s (%.0f%% confidence)", recommendation.Rationale, recommendation.Confidence*100)
						swapped = true
						if te.Logger != nil {
							te.Logger.Infof("[Task %s] Agent swap: %s → %s (%s)", task.Number, task.Agent, newAgent, swapReason)
						}
					} else if err != nil && te.Logger != nil {
						te.Logger.Warnf("[Task %s] Intelligent agent swap failed: %v", task.Number, err)
					}
				}

				// Apply the agent swap
				if swapped && newAgent != "" {
					task.Agent = newAgent
					result.Task.Agent = newAgent
				}
			}
		default:
			lastErr = NewTaskError(task.Number, fmt.Sprintf("quality control returned unsupported flag %q", review.Flag), nil)
		}

		if lastErr == nil {
			lastErr = ErrQualityGateFailed
		}

		// Adaptive check moved to BEFORE retry (task.go:885-916) to avoid wasted agent invocation

		// Determine whether to retry.
		if attempt >= te.retryLimit || !te.reviewer.ShouldRetry(review, attempt) {
			result.Status = models.StatusRed
			result.RetryCount = attempt
			result.Error = lastErr
			_ = te.updatePlanStatus(task, StatusFailed, false)
			te.postTaskHook(ctx, &task, &result, models.StatusRed)
			return result, lastErr
		}

		// Inject QC feedback into prompt for next retry (v2.9+)
		// This ensures the agent sees what failed and can address specific issues
		if review.Feedback != "" || len(review.CriteriaResults) > 0 {
			// Build classification context from stored detected errors
			classificationContext := ""
			if task.Metadata != nil {
				if detectedErrors, ok := task.Metadata["detected_errors"].([]*DetectedError); ok && len(detectedErrors) > 0 {
					classificationContext = formatClassificationForRetry(detectedErrors)
				}
			}

			// Format failed criteria for explicit feedback (v2.16+)
			failedCriteriaFeedback := formatFailedCriteria(review.CriteriaResults)

			task.Prompt = fmt.Sprintf("%s\n\n---\n## PREVIOUS ATTEMPT FAILED - QC FEEDBACK (MUST ADDRESS):\n%s%s%s\n---\n\nFix ALL issues listed above before completing the task.",
				task.Prompt, review.Feedback, failedCriteriaFeedback, classificationContext)
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

// getDetectedErrorPatterns extracts ErrorPattern objects from task metadata.
// Returns a slice of patterns that were detected and stored during test command execution.
func getDetectedErrorPatterns(task *models.Task) []*ErrorPattern {
	if task.Metadata == nil {
		return nil
	}

	// Extract DetectedError objects from metadata (v2.12+)
	detectedErrors, ok := task.Metadata["detected_errors"].([]*DetectedError)
	if !ok || len(detectedErrors) == 0 {
		return nil
	}

	// Extract ErrorPattern from each DetectedError
	var patterns []*ErrorPattern
	for _, detected := range detectedErrors {
		if detected != nil && detected.Pattern != nil {
			patterns = append(patterns, detected.Pattern)
		}
	}

	return patterns
}

// getDetectedErrors extracts DetectedError list from task metadata.
// Returns nil if no detected errors are stored (v2.12+).
func getDetectedErrors(task *models.Task) []*DetectedError {
	if task == nil || task.Metadata == nil {
		return nil
	}
	if detected, ok := task.Metadata["detected_errors"].([]*DetectedError); ok {
		return detected
	}
	return nil
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

// formatFailedCriteria extracts failed criteria from QC review results and formats them
// as actionable feedback for retry agents. Returns empty string if no criteria failed.
func formatFailedCriteria(results []models.CriterionResult) string {
	if len(results) == 0 {
		return ""
	}

	var failed []models.CriterionResult
	for _, r := range results {
		if !r.Passed {
			failed = append(failed, r)
		}
	}

	if len(failed) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n### Failed Success Criteria (MUST FIX)\n\n")

	for _, f := range failed {
		criterion := f.Criterion
		if criterion == "" {
			criterion = fmt.Sprintf("Criterion %d", f.Index+1)
		}
		sb.WriteString(fmt.Sprintf("- **[%d]** %s\n", f.Index+1, criterion))
		if f.FailReason != "" {
			sb.WriteString(fmt.Sprintf("  - Reason: %s\n", f.FailReason))
		}
	}

	return sb.String()
}

// extractTaskFiles extracts file paths from task metadata for intelligent agent swap.
// Used to provide file context to IntelligentAgentSwapper.
func extractTaskFiles(task models.Task) []string {
	var files []string

	// Extract from Metadata if available
	if task.Metadata != nil {
		if targetFiles, ok := task.Metadata["target_files"].([]string); ok {
			files = append(files, targetFiles...)
		}
		if metaFiles, ok := task.Metadata["files"].([]string); ok {
			files = append(files, metaFiles...)
		}
		// Handle []interface{} case (from YAML unmarshalling)
		if targetFiles, ok := task.Metadata["target_files"].([]interface{}); ok {
			for _, f := range targetFiles {
				if s, ok := f.(string); ok {
					files = append(files, s)
				}
			}
		}
		if metaFiles, ok := task.Metadata["files"].([]interface{}); ok {
			for _, f := range metaFiles {
				if s, ok := f.(string); ok {
					files = append(files, s)
				}
			}
		}
	}

	// Deduplicate
	seen := make(map[string]bool)
	unique := make([]string, 0, len(files))
	for _, f := range files {
		if !seen[f] {
			seen[f] = true
			unique = append(unique, f)
		}
	}

	return unique
}

// formatClassificationForRetry formats classification suggestions for retry agent
func formatClassificationForRetry(errors []*DetectedError) string {
	if len(errors) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n### Error Classification Guidance\n\n")

	for i, err := range errors {
		if err == nil || err.Pattern == nil {
			continue
		}

		category := err.Pattern.Category.String()
		suggestion := err.Pattern.Suggestion

		sb.WriteString(fmt.Sprintf("**Error %d** (%s", i+1, category))
		if err.Method == "claude" {
			sb.WriteString(fmt.Sprintf(", %.0f%% confidence", err.Confidence*100))
		}
		sb.WriteString("):\n")
		sb.WriteString(fmt.Sprintf("- %s\n\n", suggestion))
	}

	return sb.String()
}
