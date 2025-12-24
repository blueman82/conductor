// Package logger provides logging implementations for Conductor execution.
//
// The logger package offers structured logging of execution progress at the
// wave and summary levels. Implementations are thread-safe and support various
// output destinations (console, file, etc.).
package logger

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/harrison/conductor/internal/models"
	"github.com/mattn/go-isatty"
	"github.com/mattn/go-runewidth"
	"golang.org/x/term"
)

// Log level constants for filtering
const (
	levelTrace int = 0
	levelDebug int = 1
	levelInfo  int = 2
	levelWarn  int = 3
	levelError int = 4
)

// ErrorPatternDisplay is an interface for objects that represent detected error patterns.
// It's used to decouple the logger from specific error pattern implementations.
type ErrorPatternDisplay interface {
	// GetCategory returns the error category string (e.g., "ENV_LEVEL", "PLAN_LEVEL", "CODE_LEVEL")
	GetCategory() string
	// GetPattern returns the error pattern text
	GetPattern() string
	// GetSuggestion returns the actionable suggestion for resolving the error
	GetSuggestion() string
	// IsAgentFixable returns whether the agent can fix this error with a retry
	IsAgentFixable() bool
}

// DetectedErrorDisplay is an interface for objects that represent detected errors with metadata.
// It extends ErrorPatternDisplay with detection method and confidence information.
type DetectedErrorDisplay interface {
	// GetErrorPattern returns the underlying error pattern (as interface{}, caller must assert to ErrorPatternDisplay)
	GetErrorPattern() interface{}
	// GetMethod returns the detection method ("claude", "regex", etc.)
	GetMethod() string
	// GetConfidence returns the confidence score (0.0-1.0)
	GetConfidence() float64
	// GetRequiresHumanIntervention returns whether manual intervention is needed
	GetRequiresHumanIntervention() bool
}

// GuardResultDisplay is an interface for objects that represent GUARD prediction results.
// It's used to decouple the logger from executor.GuardResult to avoid import cycles.
type GuardResultDisplay interface {
	GetTaskNumber() string
	GetProbability() float64
	GetConfidence() float64
	GetRiskLevel() string
	GetShouldBlock() bool
	GetBlockReason() string
	GetRecommendations() []string
}

// ConsoleLogger logs execution progress to a writer with timestamps and thread safety.
// All output is prefixed with [HH:MM:SS] timestamps for tracking execution flow.
// It supports log level filtering to control message verbosity.
// Color output is automatically enabled for terminal output (os.Stdout/os.Stderr).
// Verbose mode extends task result output to multi-line format with detailed information.
type ConsoleLogger struct {
	writer      io.Writer
	logLevel    string
	mutex       sync.Mutex
	colorOutput bool
	verbose     bool
}

// NewConsoleLogger creates a ConsoleLogger that writes to the provided io.Writer.
// If writer is nil, messages are silently discarded.
// logLevel determines the minimum log level for messages to be output.
// Valid levels: trace, debug, info, warn, error (case-insensitive).
// If logLevel is empty or invalid, defaults to "info".
// Color output is automatically enabled when writing to os.Stdout or os.Stderr with TTY support.
func NewConsoleLogger(writer io.Writer, logLevel string) *ConsoleLogger {
	// Normalize and validate log level
	normalizedLevel := normalizeLogLevel(logLevel)

	// Detect if we should use color output
	useColor := isTerminal(writer)

	return &ConsoleLogger{
		writer:      writer,
		logLevel:    normalizedLevel,
		mutex:       sync.Mutex{},
		colorOutput: useColor,
	}
}

// isTerminal checks if the writer is a terminal that supports colors.
// Returns true for os.Stdout and os.Stderr when they are TTYs.
// Does not check NO_COLOR environment variable - color control should be
// handled via config.yaml settings.
func isTerminal(w io.Writer) bool {
	if w == nil {
		return false
	}

	// Check if writer is os.Stdout and is a TTY
	if w == os.Stdout {
		return isatty.IsTerminal(os.Stdout.Fd())
	}

	// Check if writer is os.Stderr and is a TTY
	if w == os.Stderr {
		return isatty.IsTerminal(os.Stderr.Fd())
	}

	return false
}

// SetVerbose sets the verbose mode for task result logging.
// When true, LogTaskResult() outputs multi-line detailed format.
// When false, LogTaskResult() outputs compact single-line format.
func (cl *ConsoleLogger) SetVerbose(verbose bool) {
	cl.mutex.Lock()
	defer cl.mutex.Unlock()
	cl.verbose = verbose
}

// IsVerbose returns whether verbose mode is enabled for task result logging.
func (cl *ConsoleLogger) IsVerbose() bool {
	cl.mutex.Lock()
	defer cl.mutex.Unlock()
	return cl.verbose
}

// normalizeLogLevel converts a log level string to lowercase and validates it.
// Returns "info" as default for empty or invalid levels.
func normalizeLogLevel(level string) string {
	normalized := strings.ToLower(strings.TrimSpace(level))

	validLevels := map[string]bool{
		"trace": true,
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}

	if validLevels[normalized] {
		return normalized
	}

	return "info" // Default level
}

// shouldLog checks if a message at the given level should be logged.
// Returns true if messageLevel >= configured logLevel.
func (cl *ConsoleLogger) shouldLog(messageLevel string) bool {
	configuredLevel := logLevelToInt(cl.logLevel)
	msgLevel := logLevelToInt(messageLevel)
	return msgLevel >= configuredLevel
}

// logLevelToInt converts a log level string to its numeric value.
func logLevelToInt(level string) int {
	switch level {
	case "trace":
		return levelTrace
	case "debug":
		return levelDebug
	case "info":
		return levelInfo
	case "warn":
		return levelWarn
	case "error":
		return levelError
	default:
		return levelInfo // Default to info if unknown
	}
}

// LogTrace logs a trace-level message (most verbose).
// Format: "[HH:MM:SS] [TRACE] <message>"
func (cl *ConsoleLogger) LogTrace(message string) {
	cl.logWithLevel("TRACE", message)
}

// LogDebug logs a debug-level message.
// Format: "[HH:MM:SS] [DEBUG] <message>"
func (cl *ConsoleLogger) LogDebug(message string) {
	cl.logWithLevel("DEBUG", message)
}

// LogInfo logs an info-level message.
// Format: "[HH:MM:SS] [INFO] <message>"
func (cl *ConsoleLogger) LogInfo(message string) {
	cl.logWithLevel("INFO", message)
}

// LogWarn logs a warning-level message.
// Format: "[HH:MM:SS] [WARN] <message>"
func (cl *ConsoleLogger) LogWarn(message string) {
	cl.logWithLevel("WARN", message)
}

// LogError logs an error-level message.
// Format: "[HH:MM:SS] [ERROR] <message>"
func (cl *ConsoleLogger) LogError(message string) {
	cl.logWithLevel("ERROR", message)
}

// Info logs an info-level message (alias for LogInfo, satisfies RuntimeEnforcementLogger).
func (cl *ConsoleLogger) Info(message string) {
	cl.LogInfo(message)
}

// Infof logs a formatted info-level message.
func (cl *ConsoleLogger) Infof(format string, args ...interface{}) {
	cl.LogInfo(fmt.Sprintf(format, args...))
}

// Warnf logs a formatted warning-level message.
func (cl *ConsoleLogger) Warnf(format string, args ...interface{}) {
	cl.LogWarn(fmt.Sprintf(format, args...))
}

// logWithLevel is a helper that logs a message at the specified level if filtering allows it.
func (cl *ConsoleLogger) logWithLevel(level string, message string) {
	if cl.writer == nil {
		return
	}

	// Check if this level should be logged
	if !cl.shouldLog(strings.ToLower(level)) {
		return
	}

	cl.mutex.Lock()
	defer cl.mutex.Unlock()

	ts := timestamp()
	var formatted string

	if cl.colorOutput {
		// Format with colors
		formatted = cl.formatWithColor(ts, level, message)
	} else {
		// Plain text format
		formatted = fmt.Sprintf("[%s] [%s] %s\n", ts, level, message)
	}

	cl.writer.Write([]byte(formatted))
}

// formatWithColor formats a log message with ANSI color codes.
func (cl *ConsoleLogger) formatWithColor(ts, level, message string) string {
	var coloredLevel string

	switch strings.ToUpper(level) {
	case "TRACE":
		coloredLevel = color.New(color.FgHiBlack).Sprint(level)
	case "DEBUG":
		coloredLevel = color.New(color.FgCyan).Sprint(level)
	case "INFO":
		coloredLevel = color.New(color.FgBlue).Sprint(level)
	case "WARN":
		coloredLevel = color.New(color.FgYellow).Sprint(level)
	case "ERROR":
		coloredLevel = color.New(color.FgRed).Sprint(level)
	default:
		coloredLevel = level
	}

	return fmt.Sprintf("[%s] [%s] %s\n", ts, coloredLevel, message)
}

// LogWaveStart logs the start of a wave execution at INFO level.
// Format: "[HH:MM:SS] Starting <name>: <count> tasks"
func (cl *ConsoleLogger) LogWaveStart(wave models.Wave) {
	if cl.writer == nil {
		return
	}

	// Wave logging is at INFO level
	if !cl.shouldLog("info") {
		return
	}

	cl.mutex.Lock()
	defer cl.mutex.Unlock()

	ts := timestamp()
	taskCount := len(wave.TaskNumbers)

	var message string
	if cl.colorOutput {
		// Bold/bright for wave headers
		waveName := color.New(color.Bold).Sprint(wave.Name)
		message = fmt.Sprintf("[%s] Starting %s: %d tasks\n", ts, waveName, taskCount)
	} else {
		message = fmt.Sprintf("[%s] Starting %s: %d tasks\n", ts, wave.Name, taskCount)
	}

	cl.writer.Write([]byte(message))
}

// LogWaveComplete logs the completion of a wave execution at INFO level.
// Format: "[HH:MM:SS] <name> complete (<duration>) - X/X completed (X GREEN, X YELLOW, X RED)"
// Shows detailed status breakdown for each wave including task counts and QC status distribution.
func (cl *ConsoleLogger) LogWaveComplete(wave models.Wave, duration time.Duration, results []models.TaskResult) {
	if cl.writer == nil {
		return
	}

	// Wave logging is at INFO level
	if !cl.shouldLog("info") {
		return
	}

	cl.mutex.Lock()
	defer cl.mutex.Unlock()

	ts := timestamp()
	durationStr := formatDurationWithDecimal(duration)

	// Calculate task count
	taskCount := len(wave.TaskNumbers)

	// Count status breakdown from results
	statusCounts := make(map[string]int)
	for _, result := range results {
		if result.Status != "" {
			statusCounts[result.Status]++
		}
	}

	// Build the completion message with task count
	var statusBreakdown string
	if taskCount > 0 {
		// Build status breakdown string (only show non-zero statuses)
		var statusParts []string

		// Check each status in order: GREEN, YELLOW, RED
		for _, status := range []string{models.StatusGreen, models.StatusYellow, models.StatusRed} {
			count := statusCounts[status]
			if count > 0 {
				var statusText string
				if cl.colorOutput {
					// Color-code each status with symbols
					switch status {
					case models.StatusGreen:
						statusText = color.New(color.FgGreen).Sprintf("%d ✓ %s", count, status)
					case models.StatusYellow:
						statusText = color.New(color.FgYellow).Sprintf("%d ⚠ %s", count, status)
					case models.StatusRed:
						statusText = color.New(color.FgRed).Sprintf("%d ✗ %s", count, status)
					}
				} else {
					statusText = fmt.Sprintf("%d %s", count, status)
				}
				statusParts = append(statusParts, statusText)
			}
		}

		// Format: "X/X completed (X GREEN, X YELLOW, X RED)"
		if len(statusParts) > 0 {
			statusBreakdown = fmt.Sprintf(" - %d/%d completed (%s)", taskCount, taskCount, strings.Join(statusParts, ", "))
		} else {
			// No status data, just show task count
			statusBreakdown = fmt.Sprintf(" - %d/%d completed", taskCount, taskCount)
		}
	} else {
		// Show 0/0 for empty waves
		statusBreakdown = " - 0/0 completed"
	}

	var message string
	if cl.colorOutput {
		// Green for successful completion
		waveName := color.New(color.Bold).Sprint(wave.Name)
		completeText := color.New(color.FgGreen).Sprint("complete")
		message = fmt.Sprintf("[%s] %s %s (%s)%s\n", ts, waveName, completeText, durationStr, statusBreakdown)
	} else {
		message = fmt.Sprintf("[%s] %s complete (%s)%s\n", ts, wave.Name, durationStr, statusBreakdown)
	}

	cl.writer.Write([]byte(message))
}

// LogTaskResult logs the completion of a task at DEBUG level.
// Supports two output modes:
// - Default (single-line): "[HH:MM:SS] ✓/⚠/✗ Task <number> (<name>): STATUS (duration, agent: X, Y files)"
// - Verbose (multi-line): Header with indented Duration, Agent, Files, and Feedback sections
// Returns nil for successful logging, or an error if logging failed.
func (cl *ConsoleLogger) LogTaskResult(result models.TaskResult) error {
	if cl.writer == nil {
		return nil
	}

	// Task result logging is at DEBUG level
	if !cl.shouldLog("debug") {
		return nil
	}

	cl.mutex.Lock()
	defer cl.mutex.Unlock()

	// Check verbose mode and call appropriate logger
	if cl.verbose {
		return cl.logTaskResultVerbose(result)
	}
	return cl.logTaskResultDefault(result)
}

// logTaskResultDefault logs a task result in single-line format.
func (cl *ConsoleLogger) logTaskResultDefault(result models.TaskResult) error {
	ts := timestamp()

	// Build status icon and status text
	statusIcon := getStatusIcon(result.Status)
	statusText := result.Status

	// Build task identifier
	taskID := fmt.Sprintf("Task %s (%s)", result.Task.Number, result.Task.Name)

	// Build duration string
	durationStr := formatDurationWithDecimal(result.Duration)

	// Build details (agent and file count)
	details := buildTaskDetails(result.Task)

	var message string
	if cl.colorOutput {
		// Color code based on status
		var statusColor *color.Color
		switch result.Status {
		case models.StatusGreen:
			statusColor = color.New(color.FgGreen)
		case models.StatusRed, models.StatusFailed:
			statusColor = color.New(color.FgRed)
		case models.StatusYellow:
			statusColor = color.New(color.FgYellow)
		default:
			statusColor = color.New(color.FgWhite)
		}

		// Cyan for status icon
		iconColored := color.New(color.FgCyan).Sprint(statusIcon)
		statusColored := statusColor.Sprint(statusText)

		if details != "" {
			message = fmt.Sprintf("[%s] %s %s: %s (%s, %s)\n", ts, iconColored, taskID, statusColored, durationStr, details)
		} else {
			message = fmt.Sprintf("[%s] %s %s: %s (%s)\n", ts, iconColored, taskID, statusColored, durationStr)
		}
	} else {
		// Plain text format
		if details != "" {
			message = fmt.Sprintf("[%s] %s %s: %s (%s, %s)\n", ts, statusIcon, taskID, statusText, durationStr, details)
		} else {
			message = fmt.Sprintf("[%s] %s %s: %s (%s)\n", ts, statusIcon, taskID, statusText, durationStr)
		}
	}

	_, err := cl.writer.Write([]byte(message))
	return err
}

// logTaskResultVerbose logs a task result in multi-line verbose format.
// Format:
// [HH:MM:SS] ✓/⚠/✗ Task N (name): STATUS
//
//	Duration: X.XsX
//	Agent: agent-name
//	Files: list of files (with operation types)
//	Feedback: QC review feedback
//	Behavioral: tools, bash, cost (if available)
func (cl *ConsoleLogger) logTaskResultVerbose(result models.TaskResult) error {
	ts := timestamp()

	// Build status icon and status text
	statusIcon := getStatusIcon(result.Status)
	statusText := result.Status

	// Build task identifier
	taskID := fmt.Sprintf("Task %s (%s)", result.Task.Number, result.Task.Name)

	// Build duration string
	durationStr := formatDurationWithDecimal(result.Duration)

	// Build the header line
	var headerLine string
	if cl.colorOutput {
		// Color code based on status
		var statusColor *color.Color
		switch result.Status {
		case models.StatusGreen:
			statusColor = color.New(color.FgGreen)
		case models.StatusRed, models.StatusFailed:
			statusColor = color.New(color.FgRed)
		case models.StatusYellow:
			statusColor = color.New(color.FgYellow)
		default:
			statusColor = color.New(color.FgWhite)
		}

		// Cyan for status icon
		iconColored := color.New(color.FgCyan).Sprint(statusIcon)
		statusColored := statusColor.Sprint(statusText)

		headerLine = fmt.Sprintf("[%s] %s %s: %s\n", ts, iconColored, taskID, statusColored)
	} else {
		// Plain text format
		headerLine = fmt.Sprintf("[%s] %s %s: %s\n", ts, statusIcon, taskID, statusText)
	}

	// Build detailed sections (with proper indentation)
	var output strings.Builder
	output.WriteString(headerLine)

	// Duration section
	output.WriteString(fmt.Sprintf("[%s]   Duration: %s\n", ts, durationStr))

	// Agent section
	if result.Task.Agent != "" {
		if cl.colorOutput {
			agentColored := color.New(color.FgMagenta).Sprint(result.Task.Agent)
			output.WriteString(fmt.Sprintf("[%s]   Agent: %s\n", ts, agentColored))
		} else {
			output.WriteString(fmt.Sprintf("[%s]   Agent: %s\n", ts, result.Task.Agent))
		}
	}

	// Files section (only if files exist)
	totalFiles := result.Task.FilesModified + result.Task.FilesCreated + result.Task.FilesDeleted
	if totalFiles > 0 {
		filesList := buildVerboseFilesList(result.Task)
		output.WriteString(fmt.Sprintf("[%s]   Files: %s\n", ts, filesList))
	}

	// Behavioral metrics section (only if available)
	if result.Task.Metadata != nil {
		var metrics string
		if cl.colorOutput {
			metrics = formatColorizedBehavioralMetrics(result.Task.Metadata)
		} else {
			metrics = formatBehavioralMetrics(result.Task.Metadata)
		}
		if metrics != "" {
			output.WriteString(fmt.Sprintf("[%s]   Behavioral: %s\n", ts, metrics))
		}
	}

	// QC Feedback section (only if feedback exists)
	if result.ReviewFeedback != "" {
		output.WriteString(fmt.Sprintf("[%s]   Feedback: %s\n", ts, result.ReviewFeedback))
	}

	_, err := cl.writer.Write([]byte(output.String()))
	return err
}

// getStatusIcon returns a visual icon for the task status.
func getStatusIcon(status string) string {
	switch status {
	case models.StatusGreen:
		return "✓"
	case models.StatusRed, models.StatusFailed:
		return "✗"
	case models.StatusYellow:
		return "⚠"
	default:
		return "•"
	}
}

// buildTaskDetails constructs the details string (agent and file count).
// Returns empty string if no agent and no files.
func buildTaskDetails(task models.Task) string {
	var parts []string

	// Add agent info (only if agent is specified)
	if task.Agent != "" {
		parts = append(parts, fmt.Sprintf("agent: %s", task.Agent))
	}

	// Add file count (only if files were modified)
	totalFiles := task.FilesModified + task.FilesCreated + task.FilesDeleted
	if totalFiles > 0 {
		fileLabel := "file"
		if totalFiles > 1 {
			fileLabel = "files"
		}
		parts = append(parts, fmt.Sprintf("%d %s", totalFiles, fileLabel))
	}

	// Join parts with ", "
	return strings.Join(parts, ", ")
}

// buildVerboseFilesList constructs a verbose files list with operation types.
// Returns a string showing file counts by operation type (modified, created, deleted).
// Example: "2 modified, 1 created, 1 deleted"
func buildVerboseFilesList(task models.Task) string {
	var parts []string

	if task.FilesModified > 0 {
		fileLabel := "file"
		if task.FilesModified > 1 {
			fileLabel = "files"
		}
		parts = append(parts, fmt.Sprintf("%d %s modified", task.FilesModified, fileLabel))
	}

	if task.FilesCreated > 0 {
		fileLabel := "file"
		if task.FilesCreated > 1 {
			fileLabel = "files"
		}
		parts = append(parts, fmt.Sprintf("%d %s created", task.FilesCreated, fileLabel))
	}

	if task.FilesDeleted > 0 {
		fileLabel := "file"
		if task.FilesDeleted > 1 {
			fileLabel = "files"
		}
		parts = append(parts, fmt.Sprintf("%d %s deleted", task.FilesDeleted, fileLabel))
	}

	// If no files, return empty (caller should handle)
	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, ", ")
}

// LogSummary logs the execution summary with comprehensive statistics at INFO level.
// Includes status breakdown, agent usage, file modifications, failed task details, and average duration.
func (cl *ConsoleLogger) LogSummary(result models.ExecutionResult) {
	if cl.writer == nil {
		return
	}

	// Summary logging is at INFO level
	if !cl.shouldLog("info") {
		return
	}

	cl.mutex.Lock()
	defer cl.mutex.Unlock()

	ts := timestamp()
	durationStr := formatDuration(result.Duration)

	var output string

	if cl.colorOutput {
		// Colorized summary with enhanced statistics
		header := color.New(color.Bold).Sprint("=== Execution Summary ===")
		output = fmt.Sprintf("[%s] %s\n", ts, header)
		output += fmt.Sprintf("[%s] Total tasks: %d\n", ts, result.TotalTasks)

		// Green for completed tasks
		completedText := color.New(color.FgGreen).Sprintf("Completed: %d", result.Completed)
		output += fmt.Sprintf("[%s] %s\n", ts, completedText)

		// Red for failed tasks if any, otherwise show in default color
		if result.Failed > 0 {
			failedText := color.New(color.FgRed).Sprintf("Failed: %d", result.Failed)
			output += fmt.Sprintf("[%s] %s\n", ts, failedText)
		} else {
			output += fmt.Sprintf("[%s] Failed: %d\n", ts, result.Failed)
		}

		output += fmt.Sprintf("[%s] Duration: %s\n", ts, durationStr)

		// Status Breakdown section
		if len(result.StatusBreakdown) > 0 {
			breakdownHeader := color.New(color.Bold).Sprint("Status Breakdown:")
			output += fmt.Sprintf("[%s] %s\n", ts, breakdownHeader)

			// Display in consistent order: GREEN, YELLOW, RED with symbols
			for _, status := range []string{models.StatusGreen, models.StatusYellow, models.StatusRed} {
				count := result.StatusBreakdown[status]
				if count > 0 || status == models.StatusGreen { // Always show GREEN (even if 0)
					var statusColored string
					switch status {
					case models.StatusGreen:
						statusColored = color.New(color.FgGreen).Sprint("✓ " + status)
					case models.StatusYellow:
						statusColored = color.New(color.FgYellow).Sprint("⚠ " + status)
					case models.StatusRed:
						statusColored = color.New(color.FgRed).Sprint("✗ " + status)
					}
					output += fmt.Sprintf("[%s]   %s: %d\n", ts, statusColored, count)
				}
			}
		}

		// Agent Usage section (sorted by count, descending)
		if len(result.AgentUsage) > 0 {
			agentHeader := color.New(color.Bold).Sprint("Agent Usage:")
			output += fmt.Sprintf("[%s] %s\n", ts, agentHeader)

			// Sort agents by count (descending)
			agents := sortAgentsByCount(result.AgentUsage)
			for _, agent := range agents {
				agentColored := color.New(color.FgCyan).Sprint(agent.Name)
				output += fmt.Sprintf("[%s]   %s: %d tasks\n", ts, agentColored, agent.Count)
			}
		}

		// Files Modified section
		if result.TotalFiles > 0 {
			filesStr := fmt.Sprintf("%d", result.TotalFiles)
			filesColored := color.New(color.FgGreen).Sprint(filesStr)
			output += fmt.Sprintf("[%s] Files Modified: %s files\n", ts, filesColored)
		}

		// Average Duration section
		if result.AvgTaskDuration > 0 {
			avgStr := formatDurationWithDecimal(result.AvgTaskDuration)
			output += fmt.Sprintf("[%s] Average Duration: %s/task\n", ts, avgStr)
		}

		// Failed tasks section with enhanced details
		if result.Failed > 0 && len(result.FailedTasks) > 0 {
			failedHeader := color.New(color.FgRed).Sprint("Failed tasks:")
			output += fmt.Sprintf("[%s] %s\n", ts, failedHeader)
			for _, failedTask := range result.FailedTasks {
				taskHeader := fmt.Sprintf("Task %s (%s)", failedTask.Task.Number, failedTask.Task.Name)
				taskColored := color.New(color.FgRed).Sprint(taskHeader)
				output += fmt.Sprintf("[%s]   - %s: %s\n", ts, taskColored, failedTask.Status)

				// Include QC feedback if available
				if failedTask.ReviewFeedback != "" {
					output += fmt.Sprintf("[%s]     Feedback: %s\n", ts, failedTask.ReviewFeedback)
				}
			}
		}
	} else {
		// Plain text summary
		output = fmt.Sprintf("[%s] === Execution Summary ===\n", ts)
		output += fmt.Sprintf("[%s] Total tasks: %d\n", ts, result.TotalTasks)
		output += fmt.Sprintf("[%s] Completed: %d\n", ts, result.Completed)
		output += fmt.Sprintf("[%s] Failed: %d\n", ts, result.Failed)
		output += fmt.Sprintf("[%s] Duration: %s\n", ts, durationStr)

		// Status Breakdown section
		if len(result.StatusBreakdown) > 0 {
			output += fmt.Sprintf("[%s] Status Breakdown:\n", ts)
			for _, status := range []string{models.StatusGreen, models.StatusYellow, models.StatusRed} {
				count := result.StatusBreakdown[status]
				if count > 0 || status == models.StatusGreen {
					output += fmt.Sprintf("[%s]   %s: %d\n", ts, status, count)
				}
			}
		}

		// Agent Usage section
		if len(result.AgentUsage) > 0 {
			output += fmt.Sprintf("[%s] Agent Usage:\n", ts)
			agents := sortAgentsByCount(result.AgentUsage)
			for _, agent := range agents {
				output += fmt.Sprintf("[%s]   %s: %d tasks\n", ts, agent.Name, agent.Count)
			}
		}

		// Files Modified section
		if result.TotalFiles > 0 {
			output += fmt.Sprintf("[%s] Files Modified: %d files\n", ts, result.TotalFiles)
		}

		// Average Duration section
		if result.AvgTaskDuration > 0 {
			avgStr := formatDurationWithDecimal(result.AvgTaskDuration)
			output += fmt.Sprintf("[%s] Average Duration: %s/task\n", ts, avgStr)
		}

		// Failed tasks section with enhanced details
		if result.Failed > 0 && len(result.FailedTasks) > 0 {
			output += fmt.Sprintf("[%s] Failed tasks:\n", ts)
			for _, failedTask := range result.FailedTasks {
				output += fmt.Sprintf("[%s]   - Task %s (%s): %s\n", ts, failedTask.Task.Number, failedTask.Task.Name, failedTask.Status)

				// Include QC feedback if available
				if failedTask.ReviewFeedback != "" {
					output += fmt.Sprintf("[%s]     Feedback: %s\n", ts, failedTask.ReviewFeedback)
				}
			}
		}
	}

	cl.writer.Write([]byte(output))
}

// AgentCount represents an agent and its task count
type AgentCount struct {
	Name  string
	Count int
}

// sortAgentsByCount sorts agents by task count in descending order
func sortAgentsByCount(agentUsage map[string]int) []AgentCount {
	var agents []AgentCount

	for name, count := range agentUsage {
		// Skip empty agent names
		if name == "" {
			continue
		}
		agents = append(agents, AgentCount{Name: name, Count: count})
	}

	// Sort by count descending, then by name for consistency
	for i := 0; i < len(agents); i++ {
		for j := i + 1; j < len(agents); j++ {
			if agents[j].Count > agents[i].Count ||
				(agents[j].Count == agents[i].Count && agents[j].Name < agents[i].Name) {
				agents[i], agents[j] = agents[j], agents[i]
			}
		}
	}

	return agents
}

// timestamp returns the current time formatted as "15:04:05" (HH:MM:SS).
func timestamp() string {
	return time.Now().Format("15:04:05")
}

// LogTaskStart logs the start of a task execution at INFO level.
// Format: "[HH:MM:SS] ⏳ IN PROGRESS [current/total] Task number (name) (agent: agent-name)"
// Displays task start with timestamp, progress counter, status indicator, task identifier, and agent info.
// Agent info is only included if agent is not empty.
// Thread-safe with mutex protection.
func (cl *ConsoleLogger) LogTaskStart(task models.Task, current int, total int) {
	if cl.writer == nil {
		return
	}

	// Task logging is at INFO level
	if !cl.shouldLog("info") {
		return
	}

	cl.mutex.Lock()
	defer cl.mutex.Unlock()

	ts := timestamp()

	// Build task identifier
	taskID := fmt.Sprintf("Task %s (%s)", task.Number, task.Name)

	// Build progress counter
	progress := fmt.Sprintf("[%d/%d]", current, total)

	// Build agent info (only if agent is specified)
	var agentInfo string
	if task.Agent != "" {
		agentInfo = fmt.Sprintf(" (agent: %s)", task.Agent)
	}

	var message string
	if cl.colorOutput {
		// Cyan for progress counter
		progressColored := color.New(color.FgCyan).Sprint(progress)
		// White/default for status
		statusColored := color.New(color.FgWhite).Sprint("⏳ IN PROGRESS")
		// Magenta for agent info if present
		var agentColored string
		if agentInfo != "" {
			agentColored = color.New(color.FgMagenta).Sprint(agentInfo)
		}

		message = fmt.Sprintf("[%s] %s %s %s %s%s\n", ts, statusColored, progressColored, taskID, agentColored, "")
		// Clean up the message to avoid extra space at the end
		if agentInfo == "" {
			message = fmt.Sprintf("[%s] %s %s %s\n", ts, statusColored, progressColored, taskID)
		}
	} else {
		// Plain text format
		message = fmt.Sprintf("[%s] ⏳ IN PROGRESS %s %s%s\n", ts, progress, taskID, agentInfo)
	}

	cl.writer.Write([]byte(message))
}

// LogTaskAgentInvoke is called when a task agent is about to be invoked.
// This is a no-op for console logger since the invoker already displays agent info.
func (cl *ConsoleLogger) LogTaskAgentInvoke(task models.Task) {
	// No-op: agent invocation display is handled by the invoker's displayAgentInfo
}

// LogQCAgentSelection logs which QC agents were selected for review.
// Format: "[HH:MM:SS] [QC] Selected agents: [agent1, agent2] (mode: auto)"
// Thread-safe with mutex protection.
func (cl *ConsoleLogger) LogQCAgentSelection(agents []string, mode string) {
	if cl.writer == nil {
		return
	}

	// QC logging is at INFO level
	if !cl.shouldLog("info") {
		return
	}

	cl.mutex.Lock()
	defer cl.mutex.Unlock()

	ts := timestamp()

	// Format agents as comma-separated list
	agentsList := fmt.Sprintf("[%s]", strings.Join(agents, ", "))

	var message string
	if cl.colorOutput {
		// Cyan for [QC] prefix
		qcPrefix := color.New(color.FgCyan).Sprint("[QC]")
		// Magenta for agent names
		agentColored := color.New(color.FgMagenta).Sprint(agentsList)
		message = fmt.Sprintf("[%s] %s Selected agents: %s (mode: %s)\n", ts, qcPrefix, agentColored, mode)
	} else {
		// Plain text format
		message = fmt.Sprintf("[%s] [QC] Selected agents: %s (mode: %s)\n", ts, agentsList, mode)
	}

	cl.writer.Write([]byte(message))
}

// LogQCIndividualVerdicts logs verdicts from each QC agent.
// Format: "[HH:MM:SS] [QC] Individual verdicts: agent1=GREEN, agent2=RED"
// Thread-safe with mutex protection. Verdicts are color-coded if color output is enabled.
func (cl *ConsoleLogger) LogQCIndividualVerdicts(verdicts map[string]string) {
	if cl.writer == nil {
		return
	}

	// QC logging is at DEBUG level (more detailed)
	if !cl.shouldLog("debug") {
		return
	}

	cl.mutex.Lock()
	defer cl.mutex.Unlock()

	ts := timestamp()

	// Build verdict strings sorted by agent name for consistent output
	var agentNames []string
	for agent := range verdicts {
		agentNames = append(agentNames, agent)
	}

	// Sort for consistent output
	for i := 0; i < len(agentNames); i++ {
		for j := i + 1; j < len(agentNames); j++ {
			if agentNames[j] < agentNames[i] {
				agentNames[i], agentNames[j] = agentNames[j], agentNames[i]
			}
		}
	}

	var verdictsStrs []string
	for _, agent := range agentNames {
		verdict := verdicts[agent]
		verdictsStrs = append(verdictsStrs, fmt.Sprintf("%s=%s", agent, verdict))
	}

	verdictsStr := strings.Join(verdictsStrs, ", ")

	var message string
	if cl.colorOutput {
		// Cyan for [QC] prefix
		qcPrefix := color.New(color.FgCyan).Sprint("[QC]")
		// Color-code individual verdicts
		coloredVerdicts := cl.colorizeVerdicts(verdicts, agentNames)
		message = fmt.Sprintf("[%s] %s Individual verdicts: %s\n", ts, qcPrefix, coloredVerdicts)
	} else {
		// Plain text format
		message = fmt.Sprintf("[%s] [QC] Individual verdicts: %s\n", ts, verdictsStr)
	}

	cl.writer.Write([]byte(message))
}

// colorizeVerdicts creates a color-coded verdict string where each verdict is colored
// according to its status with symbols: ✓ GREEN, ⚠ YELLOW, ✗ RED.
func (cl *ConsoleLogger) colorizeVerdicts(verdicts map[string]string, agentNames []string) string {
	var parts []string
	for _, agent := range agentNames {
		verdict := verdicts[agent]
		var coloredVerdict string

		switch verdict {
		case models.StatusGreen:
			coloredVerdict = color.New(color.FgGreen).Sprint("✓ " + verdict)
		case models.StatusYellow:
			coloredVerdict = color.New(color.FgYellow).Sprint("⚠ " + verdict)
		case models.StatusRed:
			coloredVerdict = color.New(color.FgRed).Sprint("✗ " + verdict)
		default:
			coloredVerdict = verdict
		}

		parts = append(parts, fmt.Sprintf("%s=%s", agent, coloredVerdict))
	}

	return strings.Join(parts, ", ")
}

// LogQCCriteriaResults logs the per-criterion verification results from a QC agent.
// Format: "[HH:MM:SS] [QC] agent-name criteria: [✓, ✓, ✗]" or "PASS" for single criterion
// Thread-safe with mutex protection. Uses checkmarks for visual clarity.
func (cl *ConsoleLogger) LogQCCriteriaResults(agentName string, results []models.CriterionResult) {
	if cl.writer == nil {
		return
	}

	// Criteria logging is at DEBUG level
	if !cl.shouldLog("debug") {
		return
	}

	if len(results) == 0 {
		return
	}

	cl.mutex.Lock()
	defer cl.mutex.Unlock()

	ts := timestamp()

	// Build criteria status display: ✓ PASS / ✗ FAIL for color, plain text otherwise
	var parts []string
	for _, cr := range results {
		if cr.Passed {
			if cl.colorOutput {
				parts = append(parts, color.New(color.FgGreen).Sprint("✓ PASS"))
			} else {
				parts = append(parts, "PASS")
			}
		} else {
			if cl.colorOutput {
				parts = append(parts, color.New(color.FgRed).Sprint("✗ FAIL"))
			} else {
				parts = append(parts, "FAIL")
			}
		}
	}
	criteriaStr := strings.Join(parts, " ")

	var message string
	if cl.colorOutput {
		// Cyan for [QC] prefix
		qcPrefix := color.New(color.FgCyan).Sprint("[QC]")
		// Magenta for agent name
		agentColored := color.New(color.FgMagenta).Sprint(agentName)
		message = fmt.Sprintf("[%s] %s %s criteria: %s\n", ts, qcPrefix, agentColored, criteriaStr)
	} else {
		// Plain text format
		message = fmt.Sprintf("[%s] [QC] %s criteria: %s\n", ts, agentName, criteriaStr)
	}

	cl.writer.Write([]byte(message))
}

// LogQCAggregatedResult logs the final aggregated QC verdict.
// Format: "[HH:MM:SS] [QC] Final verdict: GREEN (strictest-wins)"
// The verdict is color-coded (GREEN=green, YELLOW=yellow, RED=red).
// Thread-safe with mutex protection.
func (cl *ConsoleLogger) LogQCAggregatedResult(verdict string, strategy string) {
	if cl.writer == nil {
		return
	}

	// QC logging is at INFO level
	if !cl.shouldLog("info") {
		return
	}

	cl.mutex.Lock()
	defer cl.mutex.Unlock()

	ts := timestamp()

	var message string
	if cl.colorOutput {
		// Cyan for [QC] prefix
		qcPrefix := color.New(color.FgCyan).Sprint("[QC]")
		// Color-code the verdict
		var verdictColored string
		switch verdict {
		case models.StatusGreen:
			verdictColored = color.New(color.FgGreen).Sprint(verdict)
		case models.StatusYellow:
			verdictColored = color.New(color.FgYellow).Sprint(verdict)
		case models.StatusRed:
			verdictColored = color.New(color.FgRed).Sprint(verdict)
		default:
			verdictColored = verdict
		}
		message = fmt.Sprintf("[%s] %s Final verdict: %s (%s)\n", ts, qcPrefix, verdictColored, strategy)
	} else {
		// Plain text format
		message = fmt.Sprintf("[%s] [QC] Final verdict: %s (%s)\n", ts, verdict, strategy)
	}

	cl.writer.Write([]byte(message))
}

// LogQCIntelligentSelectionMetadata logs details about intelligent agent selection.
// Logs rationale when successful, or fallback reason if intelligent selection failed.
// Format: "[HH:MM:SS] [QC] Intelligent selection: <rationale>" or
// "[HH:MM:SS] [QC] WARN: Fallback to auto mode: <reason>"
// Thread-safe with mutex protection.
func (cl *ConsoleLogger) LogQCIntelligentSelectionMetadata(rationale string, fallback bool, fallbackReason string) {
	if cl.writer == nil {
		return
	}

	// QC logging is at INFO level
	if !cl.shouldLog("info") {
		return
	}

	cl.mutex.Lock()
	defer cl.mutex.Unlock()

	ts := timestamp()

	var message string
	if fallback {
		// Log fallback warning
		if cl.colorOutput {
			qcPrefix := color.New(color.FgCyan).Sprint("[QC]")
			warnPrefix := color.New(color.FgYellow).Sprint("WARN:")
			reason := fallbackReason
			if reason == "" {
				reason = "unknown"
			}
			message = fmt.Sprintf("[%s] %s %s Fallback to auto mode: %s\n", ts, qcPrefix, warnPrefix, reason)
		} else {
			reason := fallbackReason
			if reason == "" {
				reason = "unknown"
			}
			message = fmt.Sprintf("[%s] [QC] WARN: Fallback to auto mode: %s\n", ts, reason)
		}
	} else if rationale != "" {
		// Log successful intelligent selection rationale
		if cl.colorOutput {
			qcPrefix := color.New(color.FgCyan).Sprint("[QC]")
			message = fmt.Sprintf("[%s] %s Intelligent selection: %s\n", ts, qcPrefix, rationale)
		} else {
			message = fmt.Sprintf("[%s] [QC] Intelligent selection: %s\n", ts, rationale)
		}
	} else {
		// No metadata to log
		return
	}

	cl.writer.Write([]byte(message))
}

// LogGuardPrediction logs GUARD protocol prediction results at INFO level.
// Format depends on result: blocked tasks show full details with recommendations,
// high/medium risk tasks show summary, low risk tasks are not logged (reduce noise).
// Thread-safe with mutex protection. Uses risk level emojis: 🟢 low, 🟡 medium, 🔴 high.
func (cl *ConsoleLogger) LogGuardPrediction(taskNumber string, result interface{}) {
	if cl.writer == nil || result == nil {
		return
	}

	// GUARD logging is at INFO level
	if !cl.shouldLog("info") {
		return
	}

	// Type assert to GuardResultDisplay interface
	guard, ok := result.(GuardResultDisplay)
	if !ok {
		return
	}

	cl.mutex.Lock()
	defer cl.mutex.Unlock()

	ts := timestamp()

	// Map risk levels to emojis
	riskEmoji := map[string]string{"low": "🟢", "medium": "🟡", "high": "🔴"}
	emoji := riskEmoji[guard.GetRiskLevel()]
	if emoji == "" {
		emoji = "⚪"
	}

	var message string

	if guard.GetShouldBlock() {
		// Blocked task: show full details
		if cl.colorOutput {
			guardPrefix := color.New(color.FgCyan).Sprint("[GUARD]")
			blockedText := color.New(color.FgRed, color.Bold).Sprint("BLOCKED")
			message = fmt.Sprintf("[%s] %s %s Task %s %s: %s (probability: %.1f%%, confidence: %.1f%%)\n",
				ts, guardPrefix, emoji, taskNumber, blockedText, guard.GetBlockReason(),
				guard.GetProbability()*100, guard.GetConfidence()*100)
		} else {
			message = fmt.Sprintf("[%s] [GUARD] %s Task %s BLOCKED: %s (probability: %.1f%%, confidence: %.1f%%)\n",
				ts, emoji, taskNumber, guard.GetBlockReason(),
				guard.GetProbability()*100, guard.GetConfidence()*100)
		}

		// Add recommendations
		for _, rec := range guard.GetRecommendations() {
			if cl.colorOutput {
				recColored := color.New(color.FgYellow).Sprint(rec)
				message += fmt.Sprintf("[%s]          → %s\n", ts, recColored)
			} else {
				message += fmt.Sprintf("[%s]          → %s\n", ts, rec)
			}
		}
	} else {
		// All other risk levels (high, medium, low): show summary
		if cl.colorOutput {
			guardPrefix := color.New(color.FgCyan).Sprint("[GUARD]")
			var riskColored string
			switch guard.GetRiskLevel() {
			case "high":
				riskColored = color.New(color.FgRed).Sprint(guard.GetRiskLevel())
			case "medium":
				riskColored = color.New(color.FgYellow).Sprint(guard.GetRiskLevel())
			default:
				riskColored = color.New(color.FgGreen).Sprint(guard.GetRiskLevel())
			}
			message = fmt.Sprintf("[%s] %s %s Task %s: %s risk (%.1f%% failure probability)\n",
				ts, guardPrefix, emoji, taskNumber, riskColored, guard.GetProbability()*100)
		} else {
			message = fmt.Sprintf("[%s] [GUARD] %s Task %s: %s risk (%.1f%% failure probability)\n",
				ts, emoji, taskNumber, guard.GetRiskLevel(), guard.GetProbability()*100)
		}
	}

	if message != "" {
		cl.writer.Write([]byte(message))
	}
}

// LogAgentSwap logs when GUARD predictive selection swaps to a better-performing agent.
// Thread-safe with mutex protection. Uses 🔄 emoji for swap indication.
func (cl *ConsoleLogger) LogAgentSwap(taskNumber string, fromAgent string, toAgent string) {
	if cl.writer == nil {
		return
	}

	// Agent swap is at INFO level
	if !cl.shouldLog("info") {
		return
	}

	cl.mutex.Lock()
	defer cl.mutex.Unlock()

	ts := timestamp()

	var message string
	if cl.colorOutput {
		guardPrefix := color.New(color.FgCyan).Sprint("[GUARD]")
		fromColored := color.New(color.FgYellow).Sprint(fromAgent)
		toColored := color.New(color.FgGreen).Sprint(toAgent)
		message = fmt.Sprintf("[%s] %s 🔄 Task %s: Swapping agent %s → %s (predictive selection)\n",
			ts, guardPrefix, taskNumber, fromColored, toColored)
	} else {
		message = fmt.Sprintf("[%s] [GUARD] 🔄 Task %s: Swapping agent %s → %s (predictive selection)\n",
			ts, taskNumber, fromAgent, toAgent)
	}

	cl.writer.Write([]byte(message))
}

// WaveAnomalyDisplay is an interface for objects that represent wave anomalies.
// It's used to decouple the logger from executor.WaveAnomaly to avoid import cycles.
type WaveAnomalyDisplay interface {
	GetType() string
	GetDescription() string
	GetSeverity() string
	GetTaskNumber() string
	GetWaveName() string
}

// LogAnomaly logs real-time anomaly detection results during wave execution.
// Format: "[HH:MM:SS] [ANOMALY] ⚠️ type: description (task N, severity)"
// Thread-safe with mutex protection. Uses warning colors for visibility.
func (cl *ConsoleLogger) LogAnomaly(anomaly interface{}) {
	if cl.writer == nil || anomaly == nil {
		return
	}

	// Anomaly logging is at WARN level
	if !cl.shouldLog("warn") {
		return
	}

	// Type assert to WaveAnomalyDisplay interface
	a, ok := anomaly.(WaveAnomalyDisplay)
	if !ok {
		return
	}

	cl.mutex.Lock()
	defer cl.mutex.Unlock()

	ts := timestamp()

	// Map severity to emoji
	severityEmoji := map[string]string{"low": "🟡", "medium": "🟠", "high": "🔴"}
	emoji := severityEmoji[a.GetSeverity()]
	if emoji == "" {
		emoji = "⚠️"
	}

	var message string
	taskInfo := ""
	if a.GetTaskNumber() != "" {
		taskInfo = fmt.Sprintf(" (Task %s)", a.GetTaskNumber())
	}

	if cl.colorOutput {
		anomalyPrefix := color.New(color.FgYellow, color.Bold).Sprint("[ANOMALY]")
		typeColored := color.New(color.FgYellow).Sprint(a.GetType())
		var severityColored string
		switch a.GetSeverity() {
		case "high":
			severityColored = color.New(color.FgRed).Sprint(a.GetSeverity())
		case "medium":
			severityColored = color.New(color.FgYellow).Sprint(a.GetSeverity())
		default:
			severityColored = color.New(color.FgWhite).Sprint(a.GetSeverity())
		}
		message = fmt.Sprintf("[%s] %s %s %s: %s%s [%s]\n",
			ts, anomalyPrefix, emoji, typeColored, a.GetDescription(), taskInfo, severityColored)
	} else {
		message = fmt.Sprintf("[%s] [ANOMALY] %s %s: %s%s [%s]\n",
			ts, emoji, a.GetType(), a.GetDescription(), taskInfo, a.GetSeverity())
	}

	cl.writer.Write([]byte(message))
}

// ==================== Budget Tracking Methods (v2.19+) ====================

func (cl *ConsoleLogger) LogBudgetStatus(status interface{}) {
	if cl.writer == nil || status == nil {
		return
	}
	if !cl.shouldLog("info") {
		return
	}
	cl.mutex.Lock()
	defer cl.mutex.Unlock()
	ts := timestamp()
	message := fmt.Sprintf("[%s] [BUDGET] %v\n", ts, status)
	cl.writer.Write([]byte(message))
}

func (cl *ConsoleLogger) LogBudgetWarning(percentUsed float64) {
	if cl.writer == nil {
		return
	}
	if !cl.shouldLog("warn") {
		return
	}
	cl.mutex.Lock()
	defer cl.mutex.Unlock()
	ts := timestamp()
	percentage := int(percentUsed * 100)
	message := fmt.Sprintf("[%s] [BUDGET WARNING] ⚠️  %d%%\n", ts, percentage)
	cl.writer.Write([]byte(message))
}

func (cl *ConsoleLogger) LogRateLimitPause(delay time.Duration) {
	if cl.writer == nil {
		return
	}
	if !cl.shouldLog("info") {
		return
	}
	cl.mutex.Lock()
	defer cl.mutex.Unlock()
	ts := timestamp()
	message := fmt.Sprintf("[%s] [RATE LIMIT] ⏸️  Pausing %s\n", ts, delay.Round(time.Second))
	cl.writer.Write([]byte(message))
}

func (cl *ConsoleLogger) LogRateLimitResume() {
	if cl.writer == nil {
		return
	}
	if !cl.shouldLog("info") {
		return
	}
	cl.mutex.Lock()
	defer cl.mutex.Unlock()
	ts := timestamp()
	// Print newline first to preserve the countdown line, then the resume message
	message := fmt.Sprintf("\n[%s] [RATE LIMIT] ▶️  Resuming\n", ts)
	cl.writer.Write([]byte(message))
}

func (cl *ConsoleLogger) LogRateLimitCountdown(remaining, total time.Duration) {
	if cl.writer == nil {
		return
	}
	if !cl.shouldLog("info") {
		return
	}
	cl.mutex.Lock()
	defer cl.mutex.Unlock()
	ts := timestamp()

	// Calculate percentage complete
	pct := 100.0 * (1.0 - remaining.Seconds()/total.Seconds())

	var message string
	if cl.colorOutput {
		// Yellow for countdown to indicate waiting
		prefix := color.New(color.FgYellow).Sprint("[RATE LIMIT]")
		message = fmt.Sprintf("\r[%s] %s ⏳ %s remaining (%.0f%% complete)    ",
			ts, prefix, remaining.Round(time.Second), pct)
	} else {
		message = fmt.Sprintf("\r[%s] [RATE LIMIT] ⏳ %s remaining (%.0f%% complete)    ",
			ts, remaining.Round(time.Second), pct)
	}
	cl.writer.Write([]byte(message))
}

// Box drawing characters for rich output formatting
const (
	boxTopLeft     = "┌"
	boxTopRight    = "┐"
	boxBottomLeft  = "└"
	boxBottomRight = "┘"
	boxHorizontal  = "─"
	boxVertical    = "│"
	boxTeeLeft     = "├"
	boxTeeRight    = "┤"
)

// getTerminalWidth returns the current terminal width with sensible bounds.
// Returns width capped between 60 (minimum readable) and 120 (max for readability).
// Falls back to 80 if detection fails.
func getTerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width < 60 {
		return 80 // fallback
	}
	if width > 120 {
		return 120 // cap for readability
	}
	return width
}

// ANSI color codes for box drawing
const (
	cyanColor  = "\033[36m"
	resetColor = "\033[0m"
)

// drawBoxTop draws the top border of a box (colored cyan)
func drawBoxTop(width int) string {
	return cyanColor + boxTopLeft + strings.Repeat(boxHorizontal, width-2) + boxTopRight + resetColor
}

// drawBoxBottom draws the bottom border of a box (colored cyan)
func drawBoxBottom(width int) string {
	return cyanColor + boxBottomLeft + strings.Repeat(boxHorizontal, width-2) + boxBottomRight + resetColor
}

// drawBoxDivider draws a horizontal divider within a box (colored cyan)
func drawBoxDivider(width int) string {
	return cyanColor + boxTeeLeft + strings.Repeat(boxHorizontal, width-2) + boxTeeRight + resetColor
}

// drawBoxLine draws a line of content within a box, padding to width (borders colored cyan)
func drawBoxLine(content string, width int) string {
	// Account for visible width (strip ANSI codes for length calculation)
	visibleLen := visibleLength(content)
	padding := width - 4 - visibleLen // -4 for "│ " and " │"
	if padding < 0 {
		padding = 0
		// Truncate content if too long
		content = truncateToVisibleWidth(content, width-4)
	}
	return cyanColor + boxVertical + resetColor + " " + content + strings.Repeat(" ", padding) + " " + cyanColor + boxVertical + resetColor
}

// visibleLength returns the visible terminal width of a string (excluding ANSI codes).
// Uses runewidth to properly handle wide characters like emojis (🧪 = 2 cols, ✓ = 1 col).
func visibleLength(s string) int {
	// Strip ANSI escape sequences
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	clean := ansiRegex.ReplaceAllString(s, "")
	return runewidth.StringWidth(clean)
}

// truncateToVisibleWidth truncates a string to a visible width, preserving ANSI codes
func truncateToVisibleWidth(s string, maxWidth int) string {
	if visibleLength(s) <= maxWidth {
		return s
	}
	// Simple truncation - strip codes, truncate using runewidth, lose colors
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	clean := ansiRegex.ReplaceAllString(s, "")
	return runewidth.Truncate(clean, maxWidth-3, "...")
}

// LogTestCommands logs test command execution results with boxed colorized output.
// Format: Box with header, individual results per command, and summary
func (cl *ConsoleLogger) LogTestCommands(results []models.TestCommandResult) {
	if cl.writer == nil || len(results) == 0 {
		return
	}

	if !cl.shouldLog("info") {
		return
	}

	cl.mutex.Lock()
	defer cl.mutex.Unlock()

	var output strings.Builder
	w := getTerminalWidth()

	// Box top and header
	output.WriteString(drawBoxTop(w) + "\n")

	var headerText string
	if cl.colorOutput {
		headerText = color.New(color.FgCyan, color.Bold).Sprint("🧪 Test Commands")
	} else {
		headerText = "🧪 Test Commands"
	}
	output.WriteString(drawBoxLine(headerText, w) + "\n")
	output.WriteString(drawBoxDivider(w) + "\n")

	// Individual results
	passCount := 0
	for i, entry := range results {
		if entry.Passed {
			passCount++
		}

		// Command line: [1/2] go test ./internal/config -run TestFoo
		cmdLine := fmt.Sprintf("[%d/%d] %s", i+1, len(results), truncateCommand(entry.Command, w-16))
		output.WriteString(drawBoxLine(cmdLine, w) + "\n")

		// Status line: ✓ PASS (1.2s) or ✗ FAIL (0.5s)
		durationStr := formatDurationWithDecimal(entry.Duration)
		var statusLine string
		if cl.colorOutput {
			if entry.Passed {
				statusLine = "      " + color.New(color.FgGreen).Sprint("✓ PASS") + fmt.Sprintf(" (%s)", durationStr)
			} else {
				statusLine = "      " + color.New(color.FgRed).Sprint("✗ FAIL") + fmt.Sprintf(" (%s)", durationStr)
			}
		} else {
			if entry.Passed {
				statusLine = fmt.Sprintf("      ✓ PASS (%s)", durationStr)
			} else {
				statusLine = fmt.Sprintf("      ✗ FAIL (%s)", durationStr)
			}
		}
		output.WriteString(drawBoxLine(statusLine, w) + "\n")

		// Error line for failures
		if !entry.Passed && entry.Error != nil {
			errLine := strings.TrimSpace(entry.Error.Error())
			if len(errLine) > w-12 {
				errLine = errLine[:w-15] + "..."
			}
			if cl.colorOutput {
				errLine = color.New(color.FgRed).Sprint("      " + errLine)
			} else {
				errLine = "      " + errLine
			}
			output.WriteString(drawBoxLine(errLine, w) + "\n")
		}

		// Divider between commands (not after last)
		if i < len(results)-1 {
			output.WriteString(drawBoxDivider(w) + "\n")
		}
	}

	// Summary divider and line
	output.WriteString(drawBoxDivider(w) + "\n")

	var summaryText string
	if passCount == len(results) {
		if cl.colorOutput {
			summaryText = color.New(color.FgGreen, color.Bold).Sprintf("Summary: All %d passed ✓", len(results))
		} else {
			summaryText = fmt.Sprintf("Summary: All %d passed ✓", len(results))
		}
	} else {
		if cl.colorOutput {
			summaryText = color.New(color.FgYellow, color.Bold).Sprintf("Summary: %d/%d passed", passCount, len(results))
		} else {
			summaryText = fmt.Sprintf("Summary: %d/%d passed", passCount, len(results))
		}
	}
	output.WriteString(drawBoxLine(summaryText, w) + "\n")

	// Box bottom
	output.WriteString(drawBoxBottom(w) + "\n")

	cl.writer.Write([]byte(output.String()))
}

// LogDetectedError logs a detected error with classification context.
// Format: Boxed output with category, method, confidence, pattern, and actionable suggestion.
// Shows detection method (Claude vs regex) and confidence score for transparency.
// Accepts DetectedErrorDisplay interface for decoupling from specific implementations.
func (cl *ConsoleLogger) LogDetectedError(detected interface{}) {
	if cl.writer == nil || detected == nil {
		return
	}

	// Type assert to DetectedErrorDisplay interface
	detectedErr, ok := detected.(DetectedErrorDisplay)
	if !ok {
		// If not DetectedErrorDisplay, log error
		cl.LogError(fmt.Sprintf("LogDetectedError received invalid type: %T", detected))
		return
	}

	patternRaw := detectedErr.GetErrorPattern()
	if patternRaw == nil {
		return
	}

	// Type assert to ErrorPatternDisplay
	pattern, ok := patternRaw.(ErrorPatternDisplay)
	if !ok {
		return // Not the right type
	}

	if !cl.shouldLog("info") {
		return
	}

	cl.mutex.Lock()
	defer cl.mutex.Unlock()

	var output strings.Builder
	w := getTerminalWidth()

	// Box top and header
	output.WriteString(drawBoxTop(w) + "\n")

	var headerText string
	if cl.colorOutput {
		headerText = color.New(color.FgCyan, color.Bold).Sprint("🔍 Error Pattern Detected")
	} else {
		headerText = "🔍 Error Pattern Detected"
	}
	output.WriteString(drawBoxLine(headerText, w) + "\n")
	output.WriteString(drawBoxDivider(w) + "\n")

	// Category line with color based on category
	var categoryLine string
	categoryStr := pattern.GetCategory()
	switch categoryStr {
	case "ENV_LEVEL":
		if cl.colorOutput {
			categoryLine = "Category: " + color.New(color.FgYellow).Sprint(categoryStr)
		} else {
			categoryLine = "Category: " + categoryStr
		}
	case "PLAN_LEVEL":
		if cl.colorOutput {
			categoryLine = "Category: " + color.New(color.FgYellow).Sprint(categoryStr)
		} else {
			categoryLine = "Category: " + categoryStr
		}
	case "CODE_LEVEL":
		if cl.colorOutput {
			categoryLine = "Category: " + color.New(color.FgGreen).Sprint(categoryStr)
		} else {
			categoryLine = "Category: " + categoryStr
		}
	default:
		categoryLine = "Category: " + categoryStr
	}
	output.WriteString(drawBoxLine(categoryLine, w) + "\n")

	// Method line with confidence
	var methodLine string
	method := detectedErr.GetMethod()
	if method == "claude" {
		if cl.colorOutput {
			methodLine = fmt.Sprintf("Method: %s (%.0f%% confidence)",
				color.New(color.FgMagenta).Sprint("Claude"),
				detectedErr.GetConfidence()*100)
		} else {
			methodLine = fmt.Sprintf("Method: Claude (%.0f%% confidence)", detectedErr.GetConfidence()*100)
		}
	} else {
		if cl.colorOutput {
			methodLine = "Method: " + color.New(color.FgCyan).Sprint("Regex pattern match")
		} else {
			methodLine = "Method: Regex pattern match"
		}
	}
	output.WriteString(drawBoxLine(methodLine, w) + "\n")

	// Pattern line
	patternText := truncateCommand(pattern.GetPattern(), w-12)
	patternLine := "Pattern: " + patternText
	output.WriteString(drawBoxLine(patternLine, w) + "\n")

	// Empty line for spacing
	output.WriteString(drawBoxLine("", w) + "\n")

	// Suggestion header and wrapped suggestion text
	suggestionHeader := "💡 Suggestion:"
	output.WriteString(drawBoxLine(suggestionHeader, w) + "\n")

	// Word-wrap the suggestion text
	wrappedSuggestion := wordWrapText(pattern.GetSuggestion(), w-8)
	for _, line := range wrappedSuggestion {
		output.WriteString(drawBoxLine(line, w) + "\n")
	}

	// RequiresHumanIntervention flag
	if detectedErr.GetRequiresHumanIntervention() {
		output.WriteString(drawBoxLine("", w) + "\n")
		var warningLine string
		if cl.colorOutput {
			warningLine = color.New(color.FgRed, color.Bold).Sprint("⚠️  Requires Human Intervention")
		} else {
			warningLine = "⚠️  Requires Human Intervention"
		}
		output.WriteString(drawBoxLine(warningLine, w) + "\n")
	}

	// Box bottom
	output.WriteString(drawBoxBottom(w) + "\n")

	cl.writer.Write([]byte(output.String()))
}

// LogErrorPattern logs a detected error pattern with categorization and suggestion.
// DEPRECATED: Use LogDetectedError for new code. Kept for backward compatibility.
// Format: Boxed output with category, pattern, and actionable suggestion.
// Accepts interface{} for flexibility with different pattern types (e.g., ErrorPattern from executor).
func (cl *ConsoleLogger) LogErrorPattern(pattern interface{}) {
	if cl.writer == nil || pattern == nil {
		return
	}

	// Type assert to ErrorPatternDisplay
	p, ok := pattern.(ErrorPatternDisplay)
	if !ok {
		return // Not the right type, skip
	}

	// Fallback to inline implementation for backward compatibility
	if !cl.shouldLog("info") {
		return
	}

	cl.mutex.Lock()
	defer cl.mutex.Unlock()

	var output strings.Builder
	w := getTerminalWidth()

	// Box top and header
	output.WriteString(drawBoxTop(w) + "\n")

	var headerText string
	if cl.colorOutput {
		headerText = color.New(color.FgCyan, color.Bold).Sprint("🔍 Error Pattern Detected")
	} else {
		headerText = "🔍 Error Pattern Detected"
	}
	output.WriteString(drawBoxLine(headerText, w) + "\n")
	output.WriteString(drawBoxDivider(w) + "\n")

	// Category line with color based on category
	var categoryLine string
	categoryStr := p.GetCategory()
	switch categoryStr {
	case "ENV_LEVEL":
		if cl.colorOutput {
			categoryLine = "Category: " + color.New(color.FgYellow).Sprint(categoryStr)
		} else {
			categoryLine = "Category: " + categoryStr
		}
	case "PLAN_LEVEL":
		if cl.colorOutput {
			categoryLine = "Category: " + color.New(color.FgYellow).Sprint(categoryStr)
		} else {
			categoryLine = "Category: " + categoryStr
		}
	case "CODE_LEVEL":
		if cl.colorOutput {
			categoryLine = "Category: " + color.New(color.FgGreen).Sprint(categoryStr)
		} else {
			categoryLine = "Category: " + categoryStr
		}
	default:
		categoryLine = "Category: " + categoryStr
	}
	output.WriteString(drawBoxLine(categoryLine, w) + "\n")

	// Pattern line
	patternText := truncateCommand(p.GetPattern(), w-12)
	patternLine := "Pattern: " + patternText
	output.WriteString(drawBoxLine(patternLine, w) + "\n")

	// Empty line for spacing
	output.WriteString(drawBoxLine("", w) + "\n")

	// Suggestion header and wrapped suggestion text
	suggestionHeader := "💡 Suggestion:"
	output.WriteString(drawBoxLine(suggestionHeader, w) + "\n")

	// Word-wrap the suggestion text
	wrappedSuggestion := wordWrapText(p.GetSuggestion(), w-8)
	for _, line := range wrappedSuggestion {
		output.WriteString(drawBoxLine(line, w) + "\n")
	}

	// Box bottom
	output.WriteString(drawBoxBottom(w) + "\n")

	cl.writer.Write([]byte(output.String()))
}

// LogCriterionVerifications logs criterion verification results with boxed output.
// Format: Box with header, individual criteria results, and summary
func (cl *ConsoleLogger) LogCriterionVerifications(results []models.CriterionVerificationResult) {
	if cl.writer == nil || len(results) == 0 {
		return
	}

	if !cl.shouldLog("info") {
		return
	}

	cl.mutex.Lock()
	defer cl.mutex.Unlock()

	var output strings.Builder
	w := getTerminalWidth()

	// Box top and header
	output.WriteString(drawBoxTop(w) + "\n")

	var headerText string
	if cl.colorOutput {
		headerText = color.New(color.FgCyan, color.Bold).Sprint("🔍 Criterion Verification") + " (soft signal → QC)"
	} else {
		headerText = "🔍 Criterion Verification (soft signal → QC)"
	}
	output.WriteString(drawBoxLine(headerText, w) + "\n")
	output.WriteString(drawBoxDivider(w) + "\n")

	// Individual results
	passCount := 0
	for i, entry := range results {
		if entry.Passed {
			passCount++
		}

		// Criterion line: [0] "All new fields serialize via YAML round-trip"
		criterionText := truncateCommand(entry.Criterion, w-16)
		criterionLine := fmt.Sprintf("[%d] \"%s\"", entry.Index, criterionText)
		output.WriteString(drawBoxLine(criterionLine, w) + "\n")

		// Command line if present
		if entry.Command != "" {
			cmdLine := "    cmd: " + truncateCommand(entry.Command, w-16)
			output.WriteString(drawBoxLine(cmdLine, w) + "\n")
		}

		// Status line: ✓ PASS (1.2s) or ○ FAIL (0.5s)
		durationStr := formatDurationWithDecimal(entry.Duration)
		var statusLine string
		if cl.colorOutput {
			if entry.Passed {
				statusLine = "    " + color.New(color.FgGreen).Sprint("✓ PASS") + fmt.Sprintf(" (%s)", durationStr)
			} else {
				statusLine = "    " + color.New(color.FgYellow).Sprint("○ FAIL") + fmt.Sprintf(" (%s)", durationStr)
			}
		} else {
			if entry.Passed {
				statusLine = fmt.Sprintf("    ✓ PASS (%s)", durationStr)
			} else {
				statusLine = fmt.Sprintf("    ○ FAIL (%s)", durationStr)
			}
		}
		output.WriteString(drawBoxLine(statusLine, w) + "\n")

		// Divider between criteria (not after last)
		if i < len(results)-1 {
			output.WriteString(drawBoxDivider(w) + "\n")
		}
	}

	// Summary divider and line
	output.WriteString(drawBoxDivider(w) + "\n")

	var summaryText string
	if passCount == len(results) {
		if cl.colorOutput {
			summaryText = color.New(color.FgGreen, color.Bold).Sprintf("Summary: All %d passed → QC", len(results))
		} else {
			summaryText = fmt.Sprintf("Summary: All %d passed → QC", len(results))
		}
	} else {
		if cl.colorOutput {
			summaryText = color.New(color.FgYellow, color.Bold).Sprintf("Summary: %d/%d passed → QC (failures inform review)", passCount, len(results))
		} else {
			summaryText = fmt.Sprintf("Summary: %d/%d passed → QC (failures inform review)", passCount, len(results))
		}
	}
	output.WriteString(drawBoxLine(summaryText, w) + "\n")

	// Box bottom
	output.WriteString(drawBoxBottom(w) + "\n")

	cl.writer.Write([]byte(output.String()))
}

// LogDocTargetVerifications logs documentation target verification results with boxed output.
// Format: Box with header, individual target results, and summary
func (cl *ConsoleLogger) LogDocTargetVerifications(results []models.DocTargetResult) {
	if cl.writer == nil || len(results) == 0 {
		return
	}

	if !cl.shouldLog("info") {
		return
	}

	cl.mutex.Lock()
	defer cl.mutex.Unlock()

	var output strings.Builder
	w := getTerminalWidth()

	// Box top and header
	output.WriteString(drawBoxTop(w) + "\n")

	var headerText string
	if cl.colorOutput {
		headerText = color.New(color.FgCyan, color.Bold).Sprint("📄 Doc Target Verification") + " (soft signal → QC)"
	} else {
		headerText = "📄 Doc Target Verification (soft signal → QC)"
	}
	output.WriteString(drawBoxLine(headerText, w) + "\n")
	output.WriteString(drawBoxDivider(w) + "\n")

	// Individual results
	passCount := 0
	for i, entry := range results {
		if entry.Passed {
			passCount++
		}

		// Target line: [1] docs/runtime.md → "# Configuration"
		target := fmt.Sprintf("[%d] %s → \"%s\"", i+1, entry.Location, entry.Section)
		if len(target) > w-6 {
			target = target[:w-9] + "..."
		}
		output.WriteString(drawBoxLine(target, w) + "\n")

		// Status line: ✓ FOUND or ○ MISSING
		var statusLine string
		if cl.colorOutput {
			if entry.Passed {
				statusLine = "    " + color.New(color.FgGreen).Sprint("✓ FOUND")
			} else {
				statusLine = "    " + color.New(color.FgYellow).Sprint("○ MISSING")
			}
		} else {
			if entry.Passed {
				statusLine = "    ✓ FOUND"
			} else {
				statusLine = "    ○ MISSING"
			}
		}
		output.WriteString(drawBoxLine(statusLine, w) + "\n")

		// Divider between targets (not after last)
		if i < len(results)-1 {
			output.WriteString(drawBoxDivider(w) + "\n")
		}
	}

	// Summary divider and line
	output.WriteString(drawBoxDivider(w) + "\n")

	var summaryText string
	if passCount == len(results) {
		if cl.colorOutput {
			summaryText = color.New(color.FgGreen, color.Bold).Sprintf("Summary: All %d found → QC", len(results))
		} else {
			summaryText = fmt.Sprintf("Summary: All %d found → QC", len(results))
		}
	} else {
		if cl.colorOutput {
			summaryText = color.New(color.FgYellow, color.Bold).Sprintf("Summary: %d/%d found → QC (missing inform review)", passCount, len(results))
		} else {
			summaryText = fmt.Sprintf("Summary: %d/%d found → QC (missing inform review)", passCount, len(results))
		}
	}
	output.WriteString(drawBoxLine(summaryText, w) + "\n")

	// Box bottom
	output.WriteString(drawBoxBottom(w) + "\n")

	cl.writer.Write([]byte(output.String()))
}

// truncateCommand shortens a command string for display.
func truncateCommand(cmd string, maxLen int) string {
	cmd = strings.TrimSpace(cmd)
	if len(cmd) <= maxLen {
		return cmd
	}
	return cmd[:maxLen-3] + "..."
}

// wordWrapText wraps text to fit within maxLen characters per line.
// Returns a slice of strings, each fitting within the specified width.
func wordWrapText(text string, maxLen int) []string {
	if maxLen <= 0 {
		maxLen = 60
	}

	text = strings.TrimSpace(text)
	if text == "" {
		return []string{}
	}

	// If text fits on one line, return it as-is
	if len(text) <= maxLen {
		return []string{text}
	}

	var lines []string
	words := strings.Fields(text)
	var currentLine string

	for _, word := range words {
		if currentLine == "" {
			currentLine = word
		} else if len(currentLine)+1+len(word) <= maxLen {
			currentLine += " " + word
		} else {
			lines = append(lines, currentLine)
			currentLine = word
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines
}

// LogProgress logs real-time progress of task execution with percentage, counts, and average duration.
// Format: "[HH:MM:SS] Progress: [████░░░░░░] 50% (4/8 tasks) - Avg: 3.2s/task"
// Handles edge cases: zero tasks, all completed, no duration data.
func (cl *ConsoleLogger) LogProgress(results []models.TaskResult) {
	if cl.writer == nil {
		return
	}

	// Progress logging is at INFO level
	if !cl.shouldLog("info") {
		return
	}

	cl.mutex.Lock()
	defer cl.mutex.Unlock()

	ts := timestamp()

	// Count completed tasks and sum durations from results
	completed := 0
	totalDuration := time.Duration(0)

	for _, result := range results {
		if result.Status == "GREEN" {
			completed++
			// Use the Duration field from TaskResult
			totalDuration += result.Duration
		}
	}

	total := len(results)

	// Calculate percentage
	var percentage int
	if total == 0 {
		percentage = 0
	} else {
		percentage = (completed * 100) / total
		if percentage > 100 {
			percentage = 100
		}
	}

	// Create progress bar using ProgressBar component
	pb := NewProgressBar(total, 10, cl.colorOutput)
	pb.Update(completed)
	pbRender := pb.Render()

	// Calculate average duration per task
	var avgDurationStr string
	if completed > 0 {
		avgDuration := totalDuration / time.Duration(completed)
		avgDurationStr = fmt.Sprintf(" - Avg: %s/task", formatDurationWithDecimal(avgDuration))
	}

	// Build progress message
	progressMsg := fmt.Sprintf("Progress: %s (%d/%d tasks)%s", pbRender, completed, total, avgDurationStr)

	var output string
	if cl.colorOutput {
		// Always use cyan for progress (doesn't turn green even when 100% since RED/YELLOW tasks are counted)
		progressMsg = color.New(color.FgCyan).Sprint(progressMsg)
		output = fmt.Sprintf("[%s] %s\n", ts, progressMsg)
	} else {
		output = fmt.Sprintf("[%s] %s\n", ts, progressMsg)
	}

	cl.writer.Write([]byte(output))
}

// formatDuration converts a time.Duration to a human-readable string.
// Examples: "5s", "1m30s", "2h15m"
func formatDuration(d time.Duration) string {
	switch {
	case d >= time.Hour:
		hours := d / time.Hour
		remainder := d % time.Hour
		if remainder == 0 {
			return fmt.Sprintf("%dh", hours)
		}
		minutes := remainder / time.Minute
		remainder = remainder % time.Minute
		if remainder == 0 {
			return fmt.Sprintf("%dh%dm", hours, minutes)
		}
		seconds := remainder / time.Second
		return fmt.Sprintf("%dh%dm%ds", hours, minutes, seconds)
	case d >= time.Minute:
		minutes := d / time.Minute
		remainder := d % time.Minute
		if remainder == 0 {
			return fmt.Sprintf("%dm", minutes)
		}
		seconds := remainder / time.Second
		return fmt.Sprintf("%dm%ds", minutes, seconds)
	default:
		return fmt.Sprintf("%ds", int64(d.Seconds()))
	}
}

// formatDurationWithDecimal converts a time.Duration to a human-readable string with decimal precision.
// For sub-minute durations, shows decimal precision (e.g., "12.5s", "0.5s").
// For longer durations, uses standard format (e.g., "1m30s", "2h15m").
func formatDurationWithDecimal(d time.Duration) string {
	switch {
	case d >= time.Hour:
		// Use standard format for hours and above
		hours := d / time.Hour
		remainder := d % time.Hour
		if remainder == 0 {
			return fmt.Sprintf("%dh", hours)
		}
		minutes := remainder / time.Minute
		remainder = remainder % time.Minute
		if remainder == 0 {
			return fmt.Sprintf("%dh%dm", hours, minutes)
		}
		seconds := remainder / time.Second
		return fmt.Sprintf("%dh%dm%ds", hours, minutes, seconds)
	case d >= time.Minute:
		// Use standard format for minutes and above
		minutes := d / time.Minute
		remainder := d % time.Minute
		if remainder == 0 {
			return fmt.Sprintf("%dm", minutes)
		}
		seconds := remainder / time.Second
		return fmt.Sprintf("%dm%ds", minutes, seconds)
	default:
		// Use decimal precision for seconds
		seconds := d.Seconds()
		return fmt.Sprintf("%.1f%s", seconds, "s")
	}
}

// formatBehavioralMetrics formats behavioral metrics from task metadata into a readable summary.
// Returns empty string if no relevant behavioral data is available.
// Format: "tools: N, bash: N, cost: $X.XX"
func formatBehavioralMetrics(metadata map[string]interface{}) string {
	if metadata == nil {
		return ""
	}

	var parts []string

	// Extract tool count
	if toolCount, ok := metadata["tool_count"].(int); ok && toolCount > 0 {
		parts = append(parts, fmt.Sprintf("tools: %d", toolCount))
	} else if toolCountFloat, ok := metadata["tool_count"].(float64); ok && toolCountFloat > 0 {
		parts = append(parts, fmt.Sprintf("tools: %d", int(toolCountFloat)))
	}

	// Extract bash command count
	if bashCount, ok := metadata["bash_count"].(int); ok && bashCount > 0 {
		parts = append(parts, fmt.Sprintf("bash: %d", bashCount))
	} else if bashCountFloat, ok := metadata["bash_count"].(float64); ok && bashCountFloat > 0 {
		parts = append(parts, fmt.Sprintf("bash: %d", int(bashCountFloat)))
	}

	// Extract file operation count
	if fileOps, ok := metadata["file_operations"].(int); ok && fileOps > 0 {
		parts = append(parts, fmt.Sprintf("files: %d", fileOps))
	} else if fileOpsFloat, ok := metadata["file_operations"].(float64); ok && fileOpsFloat > 0 {
		parts = append(parts, fmt.Sprintf("files: %d", int(fileOpsFloat)))
	}

	// Extract cost (in dollars)
	if cost, ok := metadata["cost"].(float64); ok && cost > 0 {
		parts = append(parts, fmt.Sprintf("cost: $%.4f", cost))
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, ", ")
}

// NoOpLogger is a Logger implementation that discards all log messages.
// Useful for testing or when logging is disabled.
type NoOpLogger struct{}

// NewNoOpLogger creates a NoOpLogger instance.
func NewNoOpLogger() *NoOpLogger {
	return &NoOpLogger{}
}

// LogWaveStart is a no-op implementation.
func (n *NoOpLogger) LogWaveStart(wave models.Wave) {
}

// LogWaveComplete is a no-op implementation.
func (n *NoOpLogger) LogWaveComplete(wave models.Wave, duration time.Duration, results []models.TaskResult) {
}

// LogTaskResult is a no-op implementation.
func (n *NoOpLogger) LogTaskResult(result models.TaskResult) error {
	return nil
}

// LogProgress is a no-op implementation.
func (n *NoOpLogger) LogProgress(results []models.TaskResult) {
}

// LogSummary is a no-op implementation.
func (n *NoOpLogger) LogSummary(result models.ExecutionResult) {
}

// LogTaskAgentInvoke is a no-op implementation.
func (n *NoOpLogger) LogTaskAgentInvoke(task models.Task) {
}

// LogQCAgentSelection is a no-op implementation.
func (n *NoOpLogger) LogQCAgentSelection(agents []string, mode string) {
}

// LogQCIndividualVerdicts is a no-op implementation.
func (n *NoOpLogger) LogQCIndividualVerdicts(verdicts map[string]string) {
}

// LogQCAggregatedResult is a no-op implementation.
func (n *NoOpLogger) LogQCAggregatedResult(verdict string, strategy string) {
}

// LogQCCriteriaResults is a no-op implementation.
func (n *NoOpLogger) LogQCCriteriaResults(agentName string, results []models.CriterionResult) {
}

// LogQCIntelligentSelectionMetadata is a no-op implementation.
func (n *NoOpLogger) LogQCIntelligentSelectionMetadata(rationale string, fallback bool, fallbackReason string) {
}

// LogGuardPrediction is a no-op implementation.
func (n *NoOpLogger) LogGuardPrediction(taskNumber string, result interface{}) {
}

// LogAgentSwap is a no-op implementation.
func (n *NoOpLogger) LogAgentSwap(taskNumber string, fromAgent string, toAgent string) {
}

// LogAnomaly is a no-op implementation.
func (n *NoOpLogger) LogAnomaly(anomaly interface{}) {
}

// LogTestCommands is a no-op implementation.
func (n *NoOpLogger) LogTestCommands(entries []models.TestCommandResult) {
}

// LogCriterionVerifications is a no-op implementation.
func (n *NoOpLogger) LogCriterionVerifications(entries []models.CriterionVerificationResult) {
}

// LogDocTargetVerifications is a no-op implementation.
func (n *NoOpLogger) LogDocTargetVerifications(entries []models.DocTargetResult) {
}

// Budget tracking methods (v2.19+)
func (n *NoOpLogger) LogBudgetStatus(status interface{})                   {}
func (n *NoOpLogger) LogBudgetWarning(percentUsed float64)                 {}
func (n *NoOpLogger) LogRateLimitPause(delay time.Duration)                {}
func (n *NoOpLogger) LogRateLimitResume()                                  {}
func (n *NoOpLogger) LogRateLimitCountdown(remaining, total time.Duration) {}
