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
//
//	Plan → Graph → Waves → Orchestrator → WaveExecutor → TaskExecutor → QC → Results
package executor

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/harrison/conductor/internal/claude"
	"github.com/harrison/conductor/internal/models"
	"github.com/harrison/conductor/internal/similarity"
)

// Logger interface for orchestrator progress reporting.
// Supports wave-level, task-level, summary, and quality control logging.
// Implementations can log to console, file, or other destinations.
type Logger interface {
	LogWaveStart(wave models.Wave)
	LogWaveComplete(wave models.Wave, duration time.Duration, results []models.TaskResult)
	LogTaskResult(result models.TaskResult) error
	LogProgress(results []models.TaskResult)
	LogSummary(result models.ExecutionResult)
	LogTaskAgentInvoke(task models.Task)
	LogQCAgentSelection(agents []string, mode string)
	LogQCIndividualVerdicts(verdicts map[string]string)
	LogQCAggregatedResult(verdict string, strategy string)
	LogQCCriteriaResults(agentName string, results []models.CriterionResult)
	LogQCIntelligentSelectionMetadata(rationale string, fallback bool, fallbackReason string)
	LogAnomaly(anomaly interface{})

	// Budget tracking methods (v2.19+)
	LogBudgetStatus(status interface{})                   // Log current budget status (block info, usage, burn rate)
	LogBudgetWarning(percentUsed float64)                 // Log warning when approaching budget limit
	LogRateLimitPause(delay time.Duration)                // Log when pausing due to rate limit
	LogRateLimitResume()                                  // Log when resuming after rate limit pause
	LogRateLimitCountdown(remaining, total time.Duration) // Log live countdown (every 1s, console only)
	LogRateLimitAnnounce(remaining, total time.Duration)  // Log TTS announcement (at announce_interval)
}

// WaveExecutorInterface defines the behavior required to execute waves.
type WaveExecutorInterface interface {
	ExecutePlan(ctx context.Context, plan *models.Plan) ([]models.TaskResult, error)
}

// LearningStoreInterface defines the behavior required for adaptive learning storage.
type LearningStoreInterface interface {
	GetRunCount(ctx context.Context, planFile string) (int, error)
}

// OrchestratorConfig holds configuration for creating a new Orchestrator.
type OrchestratorConfig struct {
	WaveExecutor      WaveExecutorInterface
	Logger            Logger
	LearningStore     LearningStoreInterface
	SessionID         string
	RunNumber         int
	PlanFile          string
	SkipCompleted     bool
	RetryFailed       bool
	FileToTaskMapping map[string]string
	// Pattern Intelligence hook (v2.23+)
	PatternHook *PatternIntelligenceHook
	// Setup hook for pre-wave project introspection (v3.0+)
	SetupHook *SetupHook
	// Branch guard hook for plan-level branch protection (v3.2+)
	// Runs BEFORE SetupHook to ensure branch safety before any other operations
	BranchGuardHook *BranchGuardHook
	// TargetTask filters execution to a single task (v2.27+)
	// Empty string means run all tasks
	TargetTask string
	// Similarity provides Claude-based semantic similarity (v2.32+)
	// Shared instance used by PatternIntelligence and WarmUpProvider.
	// Created once at startup and injected into both subsystems for
	// consistent configuration and rate limit handling.
	Similarity *similarity.ClaudeSimilarity
	// ClaudeInvoker is the shared Claude CLI invoker instance (v3.1+)
	// Created once at startup with cfg.Timeouts.LLM and multiLog.
	// Components that need Claude CLI invocation receive this shared instance
	// for consistent configuration, rate limit handling, and logging.
	// This centralizes Claude CLI configuration across all components:
	// SetupIntrospector, ClaudeSimilarity, ClaudeEnhancer, IntelligentSelector,
	// TaskAgentSelector, Assessor, and IntelligentAgentSwapper.
	ClaudeInvoker *claude.Invoker
}

// Orchestrator coordinates plan execution, handles graceful shutdown, and aggregates results.
type Orchestrator struct {
	waveExecutor      WaveExecutorInterface
	logger            Logger
	learningStore     LearningStoreInterface
	sessionID         string
	runNumber         int
	planFile          string
	FileToTaskMapping map[string]string // task number -> file path mapping
	skipCompleted     bool              // Skip tasks that are already completed
	retryFailed       bool              // Retry tasks that have failed status
	// Pattern Intelligence hook (v2.23+)
	patternHook *PatternIntelligenceHook
	// Setup hook for pre-wave project introspection (v3.0+)
	setupHook *SetupHook
	// Branch guard hook for plan-level branch protection (v3.2+)
	branchGuardHook *BranchGuardHook
	// targetTask filters execution to a single task (v2.27+)
	// Empty string means run all tasks
	targetTask string
	// similarity provides Claude-based semantic similarity (v2.32+)
	// Shared instance for PatternIntelligence and WarmUpProvider
	similarity *similarity.ClaudeSimilarity
	// claudeInvoker is the shared Claude CLI invoker (v3.1+)
	// Centralized instance for all components that need Claude CLI
	claudeInvoker *claude.Invoker
}

// NewOrchestrator creates a new Orchestrator instance.
// The logger parameter is optional and can be nil to disable logging.
// The waveExecutor is required and must not be nil.
func NewOrchestrator(waveExecutor WaveExecutorInterface, logger Logger) *Orchestrator {
	if waveExecutor == nil {
		panic("wave executor cannot be nil")
	}

	return &Orchestrator{
		waveExecutor:      waveExecutor,
		logger:            logger,
		FileToTaskMapping: make(map[string]string),
	}
}

// NewOrchestratorWithConfig creates a new Orchestrator instance with skip/retry configuration.
// The logger parameter is optional and can be nil to disable logging.
// The waveExecutor is required and must not be nil.
func NewOrchestratorWithConfig(waveExecutor WaveExecutorInterface, logger Logger, skipCompleted, retryFailed bool) *Orchestrator {
	if waveExecutor == nil {
		panic("wave executor cannot be nil")
	}

	return &Orchestrator{
		waveExecutor:      waveExecutor,
		logger:            logger,
		skipCompleted:     skipCompleted,
		retryFailed:       retryFailed,
		FileToTaskMapping: make(map[string]string),
	}
}

// NewOrchestratorFromConfig creates a new Orchestrator instance from a configuration object.
// This constructor supports the full set of orchestrator features including learning integration.
// The waveExecutor is required and must not be nil.
func NewOrchestratorFromConfig(config OrchestratorConfig) *Orchestrator {
	if config.WaveExecutor == nil {
		panic("wave executor cannot be nil")
	}

	// Initialize FileToTaskMapping if nil
	if config.FileToTaskMapping == nil {
		config.FileToTaskMapping = make(map[string]string)
	}

	// Generate session ID if not provided
	sessionID := config.SessionID
	if sessionID == "" {
		sessionID = generateSessionID()
	}

	// Calculate run number from learning store if learning is enabled
	// Only calculate run number if:
	// 1. Learning store is provided (not nil)
	// 2. Run number is not manually specified (is 0)
	// 3. Plan file is specified (not empty)
	runNumber := config.RunNumber
	if config.LearningStore != nil && runNumber == 0 && config.PlanFile != "" {
		count, err := config.LearningStore.GetRunCount(context.Background(), config.PlanFile)
		if err != nil {
			// Graceful degradation - log but don't fail
			count = 0
		}
		runNumber = count + 1
	}

	o := &Orchestrator{
		waveExecutor:      config.WaveExecutor,
		logger:            config.Logger,
		learningStore:     config.LearningStore,
		sessionID:         sessionID,
		runNumber:         runNumber,
		planFile:          config.PlanFile,
		skipCompleted:     config.SkipCompleted,
		retryFailed:       config.RetryFailed,
		FileToTaskMapping: config.FileToTaskMapping,
		patternHook:       config.PatternHook,
		setupHook:         config.SetupHook,
		branchGuardHook:   config.BranchGuardHook,
		targetTask:        config.TargetTask,
		similarity:        config.Similarity,
		claudeInvoker:     config.ClaudeInvoker,
	}

	// Wire Pattern Intelligence hook to WaveExecutor if provided (v2.23+)
	if config.PatternHook != nil {
		if waveExec, ok := config.WaveExecutor.(*WaveExecutor); ok {
			waveExec.SetPatternHook(config.PatternHook)
		}
	}

	return o
}

// ExecutePlan orchestrates the execution of one or more plans with graceful shutdown support.
// It merges multiple plans, handles SIGINT/SIGTERM signals, coordinates wave execution,
// logs progress, and aggregates results into an ExecutionResult.
// For file-to-task mapping, it populates the FileToTaskMapping field from the merged plan.
func (o *Orchestrator) ExecutePlan(ctx context.Context, plans ...*models.Plan) (*models.ExecutionResult, error) {
	if len(plans) == 0 {
		return nil, fmt.Errorf("at least one plan is required")
	}

	// Merge all plans
	mergedPlan, err := MergePlans(plans...)
	if err != nil {
		return nil, fmt.Errorf("failed to merge plans: %w", err)
	}

	// Populate FileToTaskMapping from merged plan
	if mergedPlan.FileToTaskMap != nil {
		for filePath, taskNumbers := range mergedPlan.FileToTaskMap {
			for _, taskNum := range taskNumbers {
				o.FileToTaskMapping[taskNum] = filePath
			}
		}
	}

	// Filter tasks if targetTask is set (--task flag)
	if o.targetTask != "" {
		var filteredTasks []models.Task
		var targetFound bool
		for _, task := range mergedPlan.Tasks {
			if task.Number == o.targetTask {
				filteredTasks = append(filteredTasks, task)
				targetFound = true
			} else {
				// Log skipped tasks
				if o.logger != nil {
					fmt.Printf("Skipping task %s (--task filter)\n", task.Number)
				}
			}
		}
		if !targetFound {
			return nil, fmt.Errorf("task %s not found in plan", o.targetTask)
		}
		mergedPlan.Tasks = filteredTasks
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

	// Run branch guard hook FIRST, before any other operations (v3.2+)
	// BranchGuardHook ensures branch safety, creates checkpoints, and switches to working branch if needed
	if o.branchGuardHook != nil {
		result, err := o.branchGuardHook.Guard(ctx)
		if err != nil {
			// Return error to block execution (e.g., dirty state with require_clean_state: true)
			return nil, fmt.Errorf("branch guard failed: %w", err)
		}
		// Log result if WasProtected (orchestrator-level visibility)
		if result != nil && result.WasProtected && o.logger != nil {
			fmt.Printf("Branch Guard: Switched from protected branch '%s' to working branch '%s'\n",
				result.OriginalBranch, result.WorkingBranch)
		}
	}

	// Run setup hook before wave execution (v3.0+)
	// SetupHook introspects the project and runs required setup commands
	if o.setupHook != nil {
		if err := o.setupHook.Setup(ctx); err != nil {
			// Log but don't fail - graceful degradation
			if o.logger != nil {
				fmt.Printf("Setup hook warning: %v\n", err)
			}
		}
	}

	// Execute the plan through the wave executor
	results, err := o.waveExecutor.ExecutePlan(ctx, mergedPlan)

	duration := time.Since(startTime)

	// Aggregate results
	executionResult := o.aggregateResults(mergedPlan, results, duration)

	// Log summary
	if o.logger != nil {
		o.logger.LogSummary(*executionResult)
	}

	return executionResult, err
}

// Similarity returns the shared ClaudeSimilarity instance (v2.32+).
// This can be used to access the similarity service for downstream components.
// Returns nil if no similarity was configured.
func (o *Orchestrator) Similarity() *similarity.ClaudeSimilarity {
	return o.similarity
}

// ClaudeInvoker returns the shared Claude CLI invoker instance (v3.1+).
// This centralized invoker provides consistent configuration and rate limit
// handling for all components that need Claude CLI invocation.
// Returns nil if no invoker was configured.
func (o *Orchestrator) ClaudeInvoker() *claude.Invoker {
	return o.claudeInvoker
}

// aggregateResults processes task results and creates an ExecutionResult summary.
func (o *Orchestrator) aggregateResults(plan *models.Plan, results []models.TaskResult, duration time.Duration) *models.ExecutionResult {
	// Use NewExecutionResult to leverage consolidated metric calculation
	executionResult := models.NewExecutionResult(results, true, duration)

	// Override TotalTasks to reflect plan size (not just executed tasks)
	// This is important when some tasks are skipped or execution is interrupted
	executionResult.TotalTasks = len(plan.Tasks)

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

// MergePlans combines multiple plan files into a single plan with validation.
// It merges tasks, waves, and tracks file-to-task ownership.
// Returns an error if there are conflicting task numbers or invalid merged state.
func MergePlans(plans ...*models.Plan) (*models.Plan, error) {
	if len(plans) == 0 {
		return nil, fmt.Errorf("no plans provided")
	}

	// Single plan: still need to set SourceFile on tasks
	if len(plans) == 1 {
		plan := plans[0]
		if plan != nil && plan.FilePath != "" {
			// Set SourceFile on each task (same as multi-plan case)
			for i := range plan.Tasks {
				plan.Tasks[i].SourceFile = plan.FilePath
			}
		}
		return plan, nil
	}

	// Track seen task numbers to detect conflicts
	seenTasks := make(map[string]bool)

	// Merged result
	merged := &models.Plan{
		Name:           "Merged Plan",
		Tasks:          []models.Task{},
		Waves:          []models.Wave{},
		FileToTaskMap:  make(map[string][]string), // file path -> task numbers
		WorktreeGroups: []models.WorktreeGroup{},
	}

	// Merge all plans
	for _, plan := range plans {
		if plan == nil {
			continue
		}

		// Check for conflicting task numbers
		for _, task := range plan.Tasks {
			if seenTasks[task.Number] {
				return nil, fmt.Errorf("conflicting task number %q from %s", task.Number, plan.FilePath)
			}
			seenTasks[task.Number] = true
		}

		// Add tasks with SourceFile populated for multi-file tracking
		for _, task := range plan.Tasks {
			// Set SourceFile to track which plan file this task comes from
			task.SourceFile = plan.FilePath
			merged.Tasks = append(merged.Tasks, task)
		}

		// Track file-to-task mapping
		if plan.FilePath != "" {
			for _, task := range plan.Tasks {
				merged.FileToTaskMap[plan.FilePath] = append(merged.FileToTaskMap[plan.FilePath], task.Number)
			}
		}

		// Merge waves
		merged.Waves = append(merged.Waves, plan.Waves...)

		// Merge worktree groups (dedup by GroupID)
		groupMap := make(map[string]models.WorktreeGroup)
		for _, group := range merged.WorktreeGroups {
			groupMap[group.GroupID] = group
		}
		for _, group := range plan.WorktreeGroups {
			groupMap[group.GroupID] = group
		}
		merged.WorktreeGroups = make([]models.WorktreeGroup, 0, len(groupMap))
		for _, group := range groupMap {
			merged.WorktreeGroups = append(merged.WorktreeGroups, group)
		}

		// Inherit first plan's defaults
		if merged.DefaultAgent == "" && plan.DefaultAgent != "" {
			merged.DefaultAgent = plan.DefaultAgent
		}
		if !merged.QualityControl.Enabled && plan.QualityControl.Enabled {
			merged.QualityControl = plan.QualityControl
		}
	}

	// Validate merged plan
	if len(merged.Tasks) == 0 {
		return nil, fmt.Errorf("merged plan has no tasks")
	}

	// Detect cycles in merged plan
	if models.HasCyclicDependencies(merged.Tasks) {
		return nil, fmt.Errorf("merged plan contains circular dependencies")
	}

	return merged, nil
}
