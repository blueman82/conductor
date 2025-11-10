package cmd

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/harrison/conductor/internal/agent"
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

Examples:
  conductor run plan.md                    # Execute plan.md
  conductor run --dry-run plan.yaml        # Validate without executing
  conductor run --max-concurrency 5 plan.md  # Limit to 5 parallel tasks
  conductor run --timeout 2h plan.md       # Set 2 hour timeout
  conductor run --verbose plan.md          # Show detailed progress
  conductor run --log-dir ./logs plan.md   # Use custom log directory`,
		Args: cobra.ExactArgs(1),
		RunE: runCommand,
	}

	// Add flags
	cmd.Flags().Bool("dry-run", false, "Validate the plan without executing tasks")
	cmd.Flags().Int("max-concurrency", 0, "Maximum number of concurrent tasks (0 = unlimited)")
	cmd.Flags().String("timeout", "10h", "Maximum execution time (e.g., 30m, 2h, 1h30m)")
	cmd.Flags().Bool("verbose", false, "Show detailed execution information")
	cmd.Flags().String("log-dir", "", "Directory for log files (default: .conductor/logs)")

	return cmd
}

// runCommand implements the run command logic
func runCommand(cmd *cobra.Command, args []string) error {
	// Get flags
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	maxConcurrency, _ := cmd.Flags().GetInt("max-concurrency")
	timeoutStr, _ := cmd.Flags().GetString("timeout")
	verbose, _ := cmd.Flags().GetBool("verbose")
	logDir, _ := cmd.Flags().GetString("log-dir")

	// Validate timeout format
	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		return fmt.Errorf("invalid timeout format %q: %w", timeoutStr, err)
	}

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

	// Create console logger for real-time progress
	consoleLog := logger.NewConsoleLogger(cmd.OutOrStdout())

	// Create file logger for detailed logs (unless dry-run)
	var fileLog *logger.FileLogger
	if logDir != "" {
		// Use custom log directory
		fileLog, err = logger.NewFileLoggerWithDir(logDir)
		if err != nil {
			return fmt.Errorf("failed to create file logger: %w", err)
		}
	} else {
		// Use default .conductor/logs directory
		fileLog, err = logger.NewFileLogger()
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
