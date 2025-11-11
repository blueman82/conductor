package cmd

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/harrison/conductor/internal/agent"
	"github.com/harrison/conductor/internal/config"
	"github.com/harrison/conductor/internal/executor"
	"github.com/harrison/conductor/internal/logger"
	"github.com/harrison/conductor/internal/models"
	"github.com/harrison/conductor/internal/parser"
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
func (l *consoleLogger) LogWaveComplete(wave models.Wave, duration time.Duration) {
	timestamp := time.Now().Format("15:04:05")
	fmt.Fprintf(l.writer, "[%s] Completed %s in %s\n", timestamp, wave.Name, duration.Round(time.Second))
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
		Use:   "run [plan-file]",
		Short: "Execute an implementation plan",
		Long: `Execute an implementation plan by orchestrating multiple Claude Code agents.

The run command parses the specified plan file (Markdown or YAML format),
calculates task dependencies, and executes tasks in parallel waves while
maintaining proper dependency ordering.

Configuration is loaded from .conductor/config.yaml if present.
CLI flags override configuration file settings.

Examples:
  conductor run plan.md                    # Execute plan.md
  conductor run --dry-run plan.yaml        # Validate without executing
  conductor run --max-concurrency 5 plan.md  # Limit to 5 parallel tasks
  conductor run --timeout 2h plan.md       # Set 2 hour timeout
  conductor run --verbose plan.md          # Show detailed progress
  conductor run --log-dir ./logs plan.md   # Use custom log directory
  conductor run --config custom.yaml plan.md  # Use custom config file
  conductor run --skip-completed plan.md   # Skip already completed tasks
  conductor run --retry-failed plan.md     # Retry failed tasks`,
		Args: cobra.ExactArgs(1),
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

	return cmd
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
		// Load from default .conductor/config.yaml
		cfg, err = config.LoadConfigFromDir(".")
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

	// Validate merged configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Use merged config values
	dryRun := cfg.DryRun
	maxConcurrency := cfg.MaxConcurrency
	timeout := cfg.Timeout
	verbose, _ := cmd.Flags().GetBool("verbose")
	logDir := cfg.LogDir

	// Get plan file path
	if len(args) == 0 {
		return fmt.Errorf("plan file argument required")
	}
	planFile := args[0]

	// Load and parse plan file
	fmt.Fprintf(cmd.OutOrStdout(), "Loading plan from %s...\n", planFile)
	plan, err := parser.ParseFile(planFile)
	if err != nil {
		return fmt.Errorf("failed to load plan file: %w", err)
	}

	// Validate plan has tasks
	if len(plan.Tasks) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "Plan file is valid but contains no tasks.\n")
		return nil
	}

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
			fmt.Fprintf(cmd.OutOrStdout(), "  Wave %d: %d task(s)\n", i+1, len(wave.TaskNumbers))
			if verbose {
				for _, taskNum := range wave.TaskNumbers {
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
	consoleLog := logger.NewConsoleLogger(cmd.OutOrStdout(), logLevel)

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

	// Create invoker for Claude CLI calls
	invoker := agent.NewInvoker()

	// Create quality controller
	qc := executor.NewQualityController(invoker)

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

	// Create wave executor with task executor
	waveExec := executor.NewWaveExecutor(taskExec, multiLog)

	// Create orchestrator with wave executor and logger
	orch := executor.NewOrchestrator(waveExec, multiLog)

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
func (ml *multiLogger) LogWaveComplete(wave models.Wave, duration time.Duration) {
	for _, logger := range ml.loggers {
		logger.LogWaveComplete(wave, duration)
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

// LogSummary forwards to all loggers
func (ml *multiLogger) LogSummary(result models.ExecutionResult) {
	for _, logger := range ml.loggers {
		logger.LogSummary(result)
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
