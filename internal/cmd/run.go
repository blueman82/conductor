package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/harrison/conductor/internal/agent"
	"github.com/harrison/conductor/internal/config"
	"github.com/harrison/conductor/internal/display"
	"github.com/harrison/conductor/internal/executor"
	"github.com/harrison/conductor/internal/learning"
	"github.com/harrison/conductor/internal/logger"
	"github.com/harrison/conductor/internal/models"
	"github.com/harrison/conductor/internal/parser"
	"github.com/harrison/conductor/internal/tts"
	"github.com/spf13/cobra"
)

// consoleLogger implements executor.Logger interface for console output
type consoleLogger struct {
	writer  io.Writer
	verbose bool
}

// LogWaveStart logs the start of a wave execution
func (l *consoleLogger) LogWaveStart(wave models.Wave) {
	timestamp := time.Now().Format("15:04:05")
	fmt.Fprintf(l.writer, "[%s] Starting %s with %d task(s)\n", timestamp, wave.Name, len(wave.TaskNumbers))
}

// LogWaveComplete logs the completion of a wave
func (l *consoleLogger) LogWaveComplete(wave models.Wave, duration time.Duration, results []models.TaskResult) {
	timestamp := time.Now().Format("15:04:05")
	fmt.Fprintf(l.writer, "[%s] Completed %s in %s\n", timestamp, wave.Name, duration.Round(time.Second))
}

// LogTaskResult logs the completion of a task (no-op in this simple logger)
func (l *consoleLogger) LogTaskResult(result models.TaskResult) error {
	return nil
}

// LogProgress logs real-time progress (no-op in this simple logger)
func (l *consoleLogger) LogProgress(results []models.TaskResult) {
	// No-op: progress bars handled by main logger
}

// LogSummary logs the execution summary
func (l *consoleLogger) LogSummary(result models.ExecutionResult) {
	fmt.Fprintf(l.writer, "\n")
	fmt.Fprintf(l.writer, "Execution Summary:\n")
	fmt.Fprintf(l.writer, "  Total tasks: %d\n", result.TotalTasks)
	fmt.Fprintf(l.writer, "  Completed: %d\n", result.Completed)
	fmt.Fprintf(l.writer, "  Failed: %d\n", result.Failed)
	fmt.Fprintf(l.writer, "  Total duration: %s\n", result.Duration.Round(time.Second))

	if len(result.FailedTasks) > 0 {
		fmt.Fprintf(l.writer, "\nFailed Tasks:\n")
		for _, task := range result.FailedTasks {
			fmt.Fprintf(l.writer, "  - Task %s: %s (%s)\n", task.Task.Number, task.Task.Name, task.Status)
		}
	}
}

// NewRunCommand creates the run command
func NewRunCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run <plan-file-or-directory>...",
		Short: "Execute an implementation plan",
		Long: `Execute an implementation plan by orchestrating multiple Claude Code agents.

The run command parses the specified plan file(s) or directory (Markdown or YAML format),
calculates task dependencies, and executes tasks in parallel waves while
maintaining proper dependency ordering.

For multiple files, only files matching pattern plan-*.md or plan-*.yaml are loaded
and merged into a single unified plan with cross-file dependency resolution.

Configuration is loaded from .conductor/config.yaml if present.
CLI flags override configuration file settings.

Examples:
  # Single file execution
  conductor run plan.md

  # Directory execution (loads numbered files: 1-*.md, 2-*.yaml, etc.)
  conductor run docs/plans/adaptive-learning-system/

  # Multi-file execution (filters plan-*.md and plan-*.yaml files)
  conductor run plan-01-foundation.md plan-02-features.yaml

  # Multi-directory execution (filters plan-* files from each directory)
  conductor run docs/plans/backend/ docs/plans/frontend/

  # Glob patterns (shell expands before conductor sees them)
  conductor run docs/plans/*/plan-*.md --max-concurrency 5

  # Other options
  conductor run --dry-run plan.yaml        # Validate without executing
  conductor run --timeout 2h plan.md       # Set 2 hour timeout
  conductor run --verbose plan.md          # Show detailed progress
  conductor run --log-dir ./logs plan.md   # Use custom log directory
  conductor run --config custom.yaml plan.md  # Use custom config file
  conductor run --skip-completed plan.md   # Skip already completed tasks
  conductor run --retry-failed plan.md     # Retry failed tasks`,
		Args: cobra.MinimumNArgs(1),
		RunE: runCommand,
	}

	// Add flags
	cmd.Flags().String("config", "", "Path to config file (default: .conductor/config.yaml)")
	cmd.Flags().Bool("dry-run", false, "Validate the plan without executing tasks")
	cmd.Flags().Int("max-concurrency", -1, "Maximum number of concurrent tasks (0 = unlimited, -1 = use config)")
	cmd.Flags().String("timeout", "", "Maximum execution time (e.g., 30m, 2h, 1h30m)")
	cmd.Flags().Bool("verbose", false, "Show detailed execution information")
	cmd.Flags().String("log-dir", "", "Directory for log files")
	cmd.Flags().Bool("skip-completed", false, "Skip tasks that have already been completed")
	cmd.Flags().Bool("no-skip-completed", false, "Do not skip completed tasks (overrides config)")
	cmd.Flags().Bool("retry-failed", false, "Retry tasks that failed")
	cmd.Flags().Bool("no-retry-failed", false, "Do not retry failed tasks (overrides config)")

	// Multi-agent QC flags (v2.2+)
	cmd.Flags().String("qc-agents", "", "Override QC agents (comma-separated, e.g., golang-pro,code-reviewer)")
	cmd.Flags().String("qc-mode", "", "Override QC agent selection mode (auto, explicit, mixed)")

	// Runtime enforcement flags (v2.9+)
	cmd.Flags().Bool("no-enforce-dependency-checks", false, "Disable dependency check commands before task execution")
	cmd.Flags().Bool("no-enforce-test-commands", false, "Disable test command execution after agent output")
	cmd.Flags().Bool("no-verify-criteria", false, "Disable per-criterion verification commands")
	cmd.Flags().Bool("no-enforce-package-guard", false, "Disable Go package conflict guard")
	cmd.Flags().Bool("no-enforce-doc-targets", false, "Disable documentation target verification")

	return cmd
}

// generateSessionID generates a unique session ID for tracking task executions
func generateSessionID() string {
	return uuid.NewString()
}

// runCommand implements the run command logic
func runCommand(cmd *cobra.Command, args []string) error {
	// Load configuration from file
	configPath, _ := cmd.Flags().GetString("config")
	var cfg *config.Config
	var err error

	if configPath != "" {
		// Load from explicit config path
		cfg, err = config.LoadConfig(configPath)
		if err != nil {
			return fmt.Errorf("failed to load config from %s: %w", configPath, err)
		}
	} else {
		// Load from default .conductor/config.yaml in conductor repo root
		// Use build-time injected repo root to ensure config loads from conductor repo,
		// not from the current working directory where conductor is invoked from
		cfg, err = config.LoadConfigFromRootWithBuildTime(GetConductorRepoRoot())
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	// Get flag values
	dryRunFlag, _ := cmd.Flags().GetBool("dry-run")
	maxConcurrencyFlag, _ := cmd.Flags().GetInt("max-concurrency")
	timeoutStr, _ := cmd.Flags().GetString("timeout")
	logDirFlag, _ := cmd.Flags().GetString("log-dir")
	skipCompletedFlag, _ := cmd.Flags().GetBool("skip-completed")
	noSkipCompletedFlag, _ := cmd.Flags().GetBool("no-skip-completed")
	retryFailedFlag, _ := cmd.Flags().GetBool("retry-failed")
	noRetryFailedFlag, _ := cmd.Flags().GetBool("no-retry-failed")

	// Validate conflicting flags
	if cmd.Flags().Changed("skip-completed") && cmd.Flags().Changed("no-skip-completed") {
		return fmt.Errorf("cannot use both --skip-completed and --no-skip-completed")
	}
	if cmd.Flags().Changed("retry-failed") && cmd.Flags().Changed("no-retry-failed") {
		return fmt.Errorf("cannot use both --retry-failed and --no-retry-failed")
	}

	// Build flag pointers for merge (only non-default values)
	var maxConcurrencyPtr *int
	if cmd.Flags().Changed("max-concurrency") {
		maxConcurrencyPtr = &maxConcurrencyFlag
	}

	var timeoutPtr *time.Duration
	if cmd.Flags().Changed("timeout") {
		timeout, err := time.ParseDuration(timeoutStr)
		if err != nil {
			return fmt.Errorf("invalid timeout format %q: %w", timeoutStr, err)
		}
		timeoutPtr = &timeout
	}

	var logDirPtr *string
	if cmd.Flags().Changed("log-dir") {
		logDirPtr = &logDirFlag
	}

	var dryRunPtr *bool
	if cmd.Flags().Changed("dry-run") {
		dryRunPtr = &dryRunFlag
	}

	var skipCompletedPtr *bool
	if cmd.Flags().Changed("skip-completed") {
		skipCompletedPtr = &skipCompletedFlag
	} else if cmd.Flags().Changed("no-skip-completed") {
		skipCompletedPtr = &noSkipCompletedFlag
	}

	var retryFailedPtr *bool
	if cmd.Flags().Changed("retry-failed") {
		retryFailedPtr = &retryFailedFlag
	} else if cmd.Flags().Changed("no-retry-failed") {
		retryFailedPtr = &noRetryFailedFlag
	}

	// Merge CLI flags with config (flags take precedence)
	cfg.MergeWithFlags(maxConcurrencyPtr, timeoutPtr, logDirPtr, dryRunPtr, skipCompletedPtr, retryFailedPtr)

	// Process multi-agent QC flags (v2.2+)
	qcAgentsFlag, _ := cmd.Flags().GetString("qc-agents")
	qcModeFlag, _ := cmd.Flags().GetString("qc-mode")

	// Validate conflicting QC flags upfront
	if cmd.Flags().Changed("qc-agents") && cmd.Flags().Changed("qc-mode") {
		if qcModeFlag != "explicit" {
			return fmt.Errorf("--qc-agents implies explicit mode; cannot use with --qc-mode=%s (use --qc-mode=explicit or omit --qc-mode)", qcModeFlag)
		}
	}

	// Process --qc-mode first (if no conflict, it takes precedence as explicit config)
	if cmd.Flags().Changed("qc-mode") && qcModeFlag != "" {
		validModes := map[string]bool{"auto": true, "explicit": true, "mixed": true}
		if !validModes[qcModeFlag] {
			return fmt.Errorf("invalid --qc-mode %q: must be auto, explicit, or mixed", qcModeFlag)
		}
		cfg.QualityControl.Agents.Mode = qcModeFlag
	}

	// Process --qc-agents (this always forces explicit mode)
	if cmd.Flags().Changed("qc-agents") && qcAgentsFlag != "" {
		rawList := strings.Split(qcAgentsFlag, ",")
		agentsList := make([]string, 0, len(rawList))
		for _, agent := range rawList {
			trimmed := strings.TrimSpace(agent)
			if trimmed != "" {
				agentsList = append(agentsList, trimmed)
			}
		}

		if len(agentsList) == 0 {
			return fmt.Errorf("--qc-agents requires at least one non-empty agent name")
		}

		cfg.QualityControl.Agents.Mode = "explicit"
		cfg.QualityControl.Agents.ExplicitList = agentsList
	}

	// Process runtime enforcement flags (v2.9+)
	// These flags disable enforcement features (defaults are true in config)
	if cmd.Flags().Changed("no-enforce-dependency-checks") {
		cfg.Executor.EnforceDependencyChecks = false
	}
	if cmd.Flags().Changed("no-enforce-test-commands") {
		cfg.Executor.EnforceTestCommands = false
	}
	if cmd.Flags().Changed("no-verify-criteria") {
		cfg.Executor.VerifyCriteria = false
	}
	if cmd.Flags().Changed("no-enforce-package-guard") {
		cfg.Executor.EnforcePackageGuard = false
	}
	if cmd.Flags().Changed("no-enforce-doc-targets") {
		cfg.Executor.EnforceDocTargets = false
	}

	// Validate merged configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Initialize learning store if enabled
	var learningStore *learning.Store
	if cfg.Learning.Enabled {
		// Use centralized conductor home database location
		dbPath, err := config.GetLearningDBPath()
		if err != nil {
			return fmt.Errorf("failed to get learning database path: %w", err)
		}
		store, err := learning.NewStore(dbPath)
		if err != nil {
			return fmt.Errorf("failed to initialize learning store: %w", err)
		}
		learningStore = store
		defer store.Close()
	}

	// Use merged config values
	dryRun := cfg.DryRun
	maxConcurrency := cfg.MaxConcurrency
	timeout := cfg.Timeout
	verbose, _ := cmd.Flags().GetBool("verbose")
	logDir := cfg.LogDir

	// Load and parse plan file(s)
	var plan *models.Plan
	var planFile string

	// Check if we should use FilterPlanFiles or direct ParseFile
	// Use FilterPlanFiles for: multiple args, or single directory arg
	// Use ParseFile for: single file arg (maintains backward compatibility)
	useDirectParse := false
	if len(args) == 1 {
		// Check if single arg is a file (not directory)
		// If stat fails, assume it's a file path (error will be caught by ParseFile)
		info, err := os.Stat(args[0])
		if err != nil || !info.IsDir() {
			// Single file (or non-existent path) - use direct ParseFile (no filtering)
			useDirectParse = true
		}
	}

	if useDirectParse {
		// Single file specified directly - parse without filtering
		planFile = args[0]
		display.DisplaySingleFile(cmd.OutOrStdout(), planFile)
		plan, err = parser.ParseFile(planFile)
		if err != nil {
			return fmt.Errorf("failed to load plan file: %w", err)
		}
	} else {
		// Multiple args or directory - use FilterPlanFiles to get plan-* files
		planFiles, err := parser.FilterPlanFiles(args)
		if err != nil {
			return fmt.Errorf("failed to filter plan files: %w", err)
		}

		// Detect and warn about numbered files in directories
		numberedFiles, _ := display.FindNumberedFiles(args[0])
		if len(numberedFiles) > 0 {
			warning := display.Warning{
				Title:      fmt.Sprintf("Found numbered files (%s) in directory", strings.Join(numberedFiles, ", ")),
				Message:    "Conductor only processes plan-*.{md,yaml} files",
				Suggestion: "To use these files, rename them to: plan-01-setup.md, plan-02-api.yaml, etc.",
			}
			warning.Display(cmd.OutOrStdout())
		}

		if len(planFiles) == 1 {
			// Single plan file found after filtering
			planFile = planFiles[0]
			display.DisplaySingleFile(cmd.OutOrStdout(), planFile)
			plan, err = parser.ParseFile(planFile)
			if err != nil {
				return fmt.Errorf("failed to load plan file: %w", err)
			}
		} else {
			// Multiple plan files - merge them
			progress := display.NewProgressIndicator(cmd.OutOrStdout(), len(planFiles))
			progress.Start()

			var plans []*models.Plan
			for _, pf := range planFiles {
				progress.Step(pf)

				p, err := parser.ParseFile(pf)
				if err != nil {
					return fmt.Errorf("failed to parse %s: %w", pf, err)
				}
				plans = append(plans, p)
			}

			progress.Complete()

			// Merge all plans into a single unified plan
			plan, err = parser.MergePlans(plans...)
			if err != nil {
				return fmt.Errorf("failed to merge plans: %w", err)
			}

			// Build FileToTaskMap for multi-file tracking
			fileToTaskMap := make(map[string][]string)
			for _, p := range plans {
				for _, task := range p.Tasks {
					fileToTaskMap[p.FilePath] = append(fileToTaskMap[p.FilePath], task.Number)
				}
			}
			plan.FileToTaskMap = fileToTaskMap

			// For display/config, use comma-separated list of plan files
			planFile = strings.Join(planFiles, ", ")
		}
	}

	// Validate plan has tasks
	if len(plan.Tasks) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "Plan file is valid but contains no tasks.\n")
		return nil
	}

	if warnings, alignmentErrors := parser.ValidateKeyPointCriteriaAlignment(plan.Tasks, cfg.Validation.KeyPointCriteria); len(warnings) > 0 || len(alignmentErrors) > 0 {
		for _, warn := range warnings {
			fmt.Fprintf(cmd.OutOrStdout(), "Warning: %s\n", warn)
		}
		if len(alignmentErrors) > 0 {
			return fmt.Errorf("key_point alignment validation failed: %s", strings.Join(alignmentErrors, "; "))
		}
	}

	// Merge config QC settings into plan if plan doesn't explicitly set QC
	// Configuration priority: plan frontmatter (explicit) > config file > defaults
	// This ensures plans without QC frontmatter get sensible defaults from config
	if !plan.QualityControl.Enabled && cfg.QualityControl.Enabled {
		plan.QualityControl = models.QualityControlConfig{
			Enabled:     cfg.QualityControl.Enabled,
			ReviewAgent: cfg.QualityControl.ReviewAgent,
			Agents: models.QCAgentConfig{
				Mode:              cfg.QualityControl.Agents.Mode,
				ExplicitList:      cfg.QualityControl.Agents.ExplicitList,
				AdditionalAgents:  cfg.QualityControl.Agents.AdditionalAgents,
				BlockedAgents:     cfg.QualityControl.Agents.BlockedAgents,
				MaxAgents:         cfg.QualityControl.Agents.MaxAgents,
				CacheTTLSeconds:   cfg.QualityControl.Agents.CacheTTLSeconds,
				RequireCodeReview: cfg.QualityControl.Agents.RequireCodeReview,
			},
			RetryOnRed: cfg.QualityControl.RetryOnRed,
		}
	}

	// Apply retry_on_red fallback logic: plan value -> config value -> default 2
	// This ensures plans without explicit retry_on_red get sensible defaults
	parser.ApplyRetryOnRedFallback(plan, cfg.Learning.MinFailuresBeforeAdapt)

	// Build dependency graph and validate
	fmt.Fprintf(cmd.OutOrStdout(), "Validating dependencies...\n")
	graph := executor.BuildDependencyGraph(plan.Tasks)
	if graph.HasCycle() {
		return fmt.Errorf("circular dependency detected in task dependencies")
	}

	// Calculate execution waves
	waves, err := executor.CalculateWaves(plan.Tasks)
	if err != nil {
		return fmt.Errorf("failed to calculate execution waves: %w", err)
	}

	// Apply max concurrency to waves if specified
	if maxConcurrency > 0 {
		for i := range waves {
			waves[i].MaxConcurrency = maxConcurrency
		}
	}

	// Display plan summary
	fmt.Fprintf(cmd.OutOrStdout(), "\nPlan Summary:\n")
	fmt.Fprintf(cmd.OutOrStdout(), "  Total tasks: %d\n", len(plan.Tasks))
	fmt.Fprintf(cmd.OutOrStdout(), "  Execution waves: %d\n", len(waves))
	fmt.Fprintf(cmd.OutOrStdout(), "  Timeout: %s\n", timeout)
	if maxConcurrency > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "  Max concurrency: %d\n", maxConcurrency)
	}
	if configPath != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "  Config: %s\n", configPath)
	}

	// Dry-run mode: validate only
	if dryRun {
		fmt.Fprintf(cmd.OutOrStdout(), "\nDry-run mode: Plan is valid and ready for execution.\n")
		fmt.Fprintf(cmd.OutOrStdout(), "\nExecution waves:\n")
		for i, wave := range waves {
			// Filter tasks based on SkipCompleted config (same logic as wave executor)
			var tasksToDisplay []string
			for _, taskNum := range wave.TaskNumbers {
				task, ok := getTask(plan.Tasks, taskNum)
				if !ok {
					continue
				}
				// Apply skip-completed filter in dry-run mode
				if cfg.SkipCompleted && task.CanSkip() {
					continue // Skip completed/skipped tasks
				}
				tasksToDisplay = append(tasksToDisplay, taskNum)
			}

			// Display wave with filtered task count
			fmt.Fprintf(cmd.OutOrStdout(), "  Wave %d: %d task(s)\n", i+1, len(tasksToDisplay))
			if verbose {
				for _, taskNum := range tasksToDisplay {
					if task, ok := getTask(plan.Tasks, taskNum); ok {
						fmt.Fprintf(cmd.OutOrStdout(), "    - Task %s: %s\n", task.Number, task.Name)
					}
				}
			}
		}
		return nil
	}

	// Full execution mode: set up orchestrator and execute
	fmt.Fprintf(cmd.OutOrStdout(), "\nStarting execution...\n\n")

	// Determine log level: verbose flag overrides config
	logLevel := cfg.LogLevel
	if verbose {
		logLevel = "debug"
	}

	// Create console logger for real-time progress
	consoleLog := logger.NewConsoleLogger(os.Stdout, logLevel)

	// Create file logger for detailed logs (unless dry-run)
	var fileLog *logger.FileLogger
	if logDir != "" {
		// Use custom log directory with log level
		fileLog, err = logger.NewFileLoggerWithDirAndLevel(logDir, logLevel)
		if err != nil {
			return fmt.Errorf("failed to create file logger: %w", err)
		}
	} else {
		// Use default .conductor/logs directory - need to set log level
		fileLog, err = logger.NewFileLoggerWithDirAndLevel(".conductor/logs", logLevel)
		if err != nil {
			return fmt.Errorf("failed to create file logger: %w", err)
		}
	}
	defer fileLog.Close()

	// Create multi-logger that writes to both console and file
	multiLog := &multiLogger{
		loggers: []executor.Logger{consoleLog, fileLog},
	}

	// Wire optional TTS logger if enabled and available
	if cfg.TTS.Enabled {
		ttsClient := tts.NewClient(cfg.TTS)
		if ttsClient.IsAvailable() {
			announcer := tts.NewAnnouncer(ttsClient)
			ttsLogger := tts.NewTTSLogger(announcer)
			multiLog.loggers = append(multiLog.loggers, ttsLogger)
		}
	}

	// Create agent registry for agent discovery
	agentRegistry := agent.NewRegistry("")
	if _, err := agentRegistry.Discover(); err != nil {
		// Log warning but don't fail - auto-selection will still work with fallback
		fmt.Fprintf(cmd.OutOrStderr(), "Warning: agent discovery failed, QC auto-selection may be limited: %v\n", err)
	}

	// Validate agent references (v2.13+)
	// Check task agents
	if taskErrors := agent.ValidateTaskAgents(plan.Tasks, agentRegistry); len(taskErrors) > 0 {
		fmt.Fprintf(cmd.OutOrStderr(), "Agent validation failed:\n")
		for _, err := range taskErrors {
			fmt.Fprintf(cmd.OutOrStderr(), "  • %s\n", err.Error())
		}
		return fmt.Errorf("cannot execute: %d agent(s) not found in registry", len(taskErrors))
	}

	// Check QC agents if QC enabled
	if plan.QualityControl.Enabled && len(plan.QualityControl.Agents.ExplicitList) > 0 {
		if qcErrors := agent.ValidateQCAgents(plan.QualityControl.Agents.ExplicitList, agentRegistry); len(qcErrors) > 0 {
			fmt.Fprintf(cmd.OutOrStderr(), "Quality control agent validation failed:\n")
			for _, err := range qcErrors {
				fmt.Fprintf(cmd.OutOrStderr(), "  • %s\n", err.Error())
			}
			return fmt.Errorf("cannot execute: %d QC agent(s) not found in registry", len(qcErrors))
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ Agent validation passed\n\n")

	// Create invoker for Claude CLI calls WITH registry
	invoker := agent.NewInvokerWithRegistry(agentRegistry)

	// Create quality controller with learning integration
	qc := executor.NewQualityController(invoker)
	qc.Registry = agentRegistry                 // Wire registry for agent verification
	qc.LearningStore = learningStore            // Enable historical context loading
	qc.AgentConfig = plan.QualityControl.Agents // Apply plan-level QC agent configuration
	qc.Logger = multiLog                        // Wire logger for QC events

	// Create task executor config
	taskExecCfg := executor.TaskExecutorConfig{
		PlanPath:       planFile,
		DefaultAgent:   plan.DefaultAgent,
		QualityControl: plan.QualityControl,
	}

	// Create task executor with QC and updater
	taskExec, err := executor.NewTaskExecutor(invoker, qc, nil, taskExecCfg)
	if err != nil {
		return fmt.Errorf("failed to create task executor: %w", err)
	}

	// Wire learning system to task executor
	sessionID := generateSessionID()
	taskExec.LearningStore = learningStore
	taskExec.PlanFile = planFile
	taskExec.SessionID = sessionID
	taskExec.RunNumber = 1 // Increment per plan re-run
	taskExec.AutoAdaptAgent = cfg.Learning.AutoAdaptAgent
	taskExec.MinFailuresBeforeAdapt = cfg.Learning.MinFailuresBeforeAdapt
	taskExec.SwapDuringRetries = cfg.Learning.SwapDuringRetries
	taskExec.Logger = consoleLog      // Runtime enforcement logging
	taskExec.EventLogger = multiLog   // Event logging (TTS agent announcements, etc.)

	// Wire runtime enforcement flags (v2.9+)
	taskExec.EnforceTestCommands = cfg.Executor.EnforceTestCommands
	taskExec.VerifyCriteria = cfg.Executor.VerifyCriteria
	taskExec.EnableErrorPatternDetection = cfg.Executor.EnableErrorPatternDetection
	taskExec.EnableClaudeClassification = cfg.Executor.EnableClaudeClassification

	// Wire intelligent task agent selection (v2.15+)
	// Enable when either:
	// 1. executor.intelligent_agent_selection is true in config, OR
	// 2. quality_control.agents.mode is "intelligent" (backward compatibility)
	if agentRegistry != nil && (cfg.Executor.IntelligentAgentSelection || plan.QualityControl.Agents.Mode == "intelligent") {
		taskExec.TaskAgentSelector = executor.NewTaskAgentSelector(agentRegistry)
		taskExec.IntelligentAgentSelection = true
	}

	// Create wave executor with task executor and config
	waveExec := executor.NewWaveExecutorWithPackageGuard(taskExec, multiLog, cfg.SkipCompleted, cfg.RetryFailed, cfg.Executor.EnforcePackageGuard)

	// Create orchestrator with learning integration
	orch := executor.NewOrchestratorFromConfig(executor.OrchestratorConfig{
		WaveExecutor:  waveExec,
		Logger:        multiLog,
		LearningStore: learningStore,
		SessionID:     sessionID,
		RunNumber:     taskExec.RunNumber,
		PlanFile:      planFile,
		SkipCompleted: cfg.SkipCompleted,
		RetryFailed:   cfg.RetryFailed,
	})

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Execute the plan
	plan.Waves = waves
	result, err := orch.ExecutePlan(ctx, plan)

	// Log task results to file
	if result != nil {
		for _, taskResult := range result.FailedTasks {
			if logErr := fileLog.LogTaskResult(taskResult); logErr != nil {
				fmt.Fprintf(cmd.OutOrStderr(), "Warning: failed to log task result: %v\n", logErr)
			}
		}
	}

	// Check for errors
	if err != nil {
		return fmt.Errorf("execution failed: %w", err)
	}

	// Display completion message
	if result.Failed > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "\nExecution completed with %d failed task(s).\n", result.Failed)
		return fmt.Errorf("%d task(s) failed", result.Failed)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\nExecution completed successfully!\n")
	if logDir != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Logs written to: %s\n", logDir)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Logs written to: %s\n", filepath.Join(".conductor", "logs"))
	}

	return nil
}

// multiLogger implements executor.Logger by delegating to multiple loggers
type multiLogger struct {
	loggers []executor.Logger
}

// LogWaveStart forwards to all loggers
func (ml *multiLogger) LogWaveStart(wave models.Wave) {
	for _, logger := range ml.loggers {
		logger.LogWaveStart(wave)
	}
}

// LogWaveComplete forwards to all loggers
func (ml *multiLogger) LogWaveComplete(wave models.Wave, duration time.Duration, results []models.TaskResult) {
	for _, logger := range ml.loggers {
		logger.LogWaveComplete(wave, duration, results)
	}
}

// LogTaskResult forwards to all loggers
func (ml *multiLogger) LogTaskResult(result models.TaskResult) error {
	var lastErr error
	for _, logger := range ml.loggers {
		if err := logger.LogTaskResult(result); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// LogProgress forwards to all loggers
func (ml *multiLogger) LogProgress(results []models.TaskResult) {
	for _, logger := range ml.loggers {
		logger.LogProgress(results)
	}
}

// LogSummary forwards to all loggers
func (ml *multiLogger) LogSummary(result models.ExecutionResult) {
	for _, logger := range ml.loggers {
		logger.LogSummary(result)
	}
}

// LogTaskAgentInvoke forwards to all loggers
func (ml *multiLogger) LogTaskAgentInvoke(task models.Task) {
	for _, logger := range ml.loggers {
		logger.LogTaskAgentInvoke(task)
	}
}

// LogQCAgentSelection forwards to all loggers
func (ml *multiLogger) LogQCAgentSelection(agents []string, mode string) {
	for _, logger := range ml.loggers {
		logger.LogQCAgentSelection(agents, mode)
	}
}

// LogQCIndividualVerdicts forwards to all loggers
func (ml *multiLogger) LogQCIndividualVerdicts(verdicts map[string]string) {
	for _, logger := range ml.loggers {
		logger.LogQCIndividualVerdicts(verdicts)
	}
}

// LogQCAggregatedResult forwards to all loggers
func (ml *multiLogger) LogQCAggregatedResult(verdict string, strategy string) {
	for _, logger := range ml.loggers {
		logger.LogQCAggregatedResult(verdict, strategy)
	}
}

// LogQCCriteriaResults forwards to all loggers
func (ml *multiLogger) LogQCCriteriaResults(agentName string, results []models.CriterionResult) {
	for _, logger := range ml.loggers {
		logger.LogQCCriteriaResults(agentName, results)
	}
}

// LogQCIntelligentSelectionMetadata forwards to all loggers
func (ml *multiLogger) LogQCIntelligentSelectionMetadata(rationale string, fallback bool, fallbackReason string) {
	for _, logger := range ml.loggers {
		logger.LogQCIntelligentSelectionMetadata(rationale, fallback, fallbackReason)
	}
}

// getTask finds a task by number in a task list
func getTask(tasks []models.Task, number string) (*models.Task, bool) {
	for i := range tasks {
		if tasks[i].Number == number {
			return &tasks[i], true
		}
	}
	return nil, false
}
