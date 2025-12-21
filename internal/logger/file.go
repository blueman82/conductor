package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/harrison/conductor/internal/models"
)

// FileLogger logs orchestrator events to files in .conductor/logs/ directory.
// It creates timestamped per-run log files, per-task detailed logs,
// and maintains a latest.log symlink pointing to the most recent run.
// It is thread-safe and implements the executor.Logger interface.
// It supports log level filtering to control message verbosity.
type FileLogger struct {
	logDir   string
	runLog   *os.File
	runFile  string
	tasksDir string
	logLevel string
	mu       sync.Mutex
}

// NewFileLogger creates a new FileLogger that writes to .conductor/logs/.
// It creates the log directory if it doesn't exist, opens a timestamped
// run log file, and creates/updates the latest.log symlink.
// Uses default log level "info".
func NewFileLogger() (*FileLogger, error) {
	// Default log directory is .conductor/logs/ in current working directory
	logDir := filepath.Join(".conductor", "logs")
	return NewFileLoggerWithDirAndLevel(logDir, "info")
}

// NewFileLoggerWithDir creates a new FileLogger with a custom log directory.
// This is useful for testing or custom deployments.
// Uses default log level "info".
func NewFileLoggerWithDir(logDir string) (*FileLogger, error) {
	return NewFileLoggerWithDirAndLevel(logDir, "info")
}

// NewFileLoggerWithDirAndLevel creates a new FileLogger with a custom log directory and log level.
// This is useful for testing or custom deployments.
func NewFileLoggerWithDirAndLevel(logDir string, logLevel string) (*FileLogger, error) {
	// Create log directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create tasks subdirectory
	tasksDir := filepath.Join(logDir, "tasks")
	if err := os.MkdirAll(tasksDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create tasks directory: %w", err)
	}

	// Generate timestamped filename: run-YYYYMMDD-HHMMSS.log
	timestamp := time.Now().Format("20060102-150405")
	runFile := filepath.Join(logDir, fmt.Sprintf("run-%s.log", timestamp))

	// Open run log file for writing
	file, err := os.OpenFile(runFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create run log file: %w", err)
	}

	// Create/update latest.log symlink
	symlinkPath := filepath.Join(logDir, "latest.log")

	// Remove existing symlink if it exists
	if _, err := os.Lstat(symlinkPath); err == nil {
		if err := os.Remove(symlinkPath); err != nil {
			file.Close()
			return nil, fmt.Errorf("failed to remove old symlink: %w", err)
		}
	}

	// Create new symlink pointing to current run log
	if err := os.Symlink(filepath.Base(runFile), symlinkPath); err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to create symlink: %w", err)
	}

	// Normalize and validate log level
	normalizedLevel := normalizeLogLevel(logLevel)

	logger := &FileLogger{
		logDir:   logDir,
		runLog:   file,
		runFile:  runFile,
		tasksDir: tasksDir,
		logLevel: normalizedLevel,
		mu:       sync.Mutex{},
	}

	// Write header to run log
	logger.writeRunLog("=== Conductor Run Log ===\n")
	logger.writeRunLog(fmt.Sprintf("Started at: %s\n\n", time.Now().Format(time.RFC3339)))

	return logger, nil
}

// shouldLog checks if a message at the given level should be logged.
// Returns true if messageLevel >= configured logLevel.
func (fl *FileLogger) shouldLog(messageLevel string) bool {
	configuredLevel := logLevelToInt(fl.logLevel)
	msgLevel := logLevelToInt(messageLevel)
	return msgLevel >= configuredLevel
}

// LogTrace logs a trace-level message (most verbose).
func (fl *FileLogger) LogTrace(message string) {
	fl.logWithLevel("TRACE", message)
}

// LogDebug logs a debug-level message.
func (fl *FileLogger) LogDebug(message string) {
	fl.logWithLevel("DEBUG", message)
}

// LogInfo logs an info-level message.
func (fl *FileLogger) LogInfo(message string) {
	fl.logWithLevel("INFO", message)
}

// LogWarn logs a warning-level message.
func (fl *FileLogger) LogWarn(message string) {
	fl.logWithLevel("WARN", message)
}

// LogError logs an error-level message.
func (fl *FileLogger) LogError(message string) {
	fl.logWithLevel("ERROR", message)
}

// logWithLevel is a helper that logs a message at the specified level if filtering allows it.
func (fl *FileLogger) logWithLevel(level string, message string) {
	// Check if this level should be logged
	levelLower := strings.ToLower(level)
	if !fl.shouldLog(levelLower) {
		return
	}

	formatted := fmt.Sprintf("[%s] [%s] %s\n", time.Now().Format("15:04:05"), level, message)
	fl.writeRunLog(formatted)
}

// LogWaveStart logs the start of a wave execution at INFO level.
// It displays the wave name, number of tasks, and max concurrency.
func (fl *FileLogger) LogWaveStart(wave models.Wave) {
	// Wave logging is at INFO level
	if !fl.shouldLog("info") {
		return
	}

	taskCount := len(wave.TaskNumbers)
	taskLabel := "task"
	if taskCount != 1 {
		taskLabel = "tasks"
	}

	message := fmt.Sprintf(
		"[%s] Starting %s: %d %s (max concurrency: %d)\n",
		time.Now().Format("15:04:05"),
		wave.Name,
		taskCount,
		taskLabel,
		wave.MaxConcurrency,
	)

	fl.writeRunLog(message)
}

// LogWaveComplete logs the completion of a wave execution at INFO level.
// It displays the wave name and duration.
func (fl *FileLogger) LogWaveComplete(wave models.Wave, duration time.Duration, results []models.TaskResult) {
	// Wave logging is at INFO level
	if !fl.shouldLog("info") {
		return
	}

	message := fmt.Sprintf(
		"[%s] %s complete: duration %.1fs\n",
		time.Now().Format("15:04:05"),
		wave.Name,
		duration.Seconds(),
	)

	fl.writeRunLog(message)
}

// LogSummary logs the execution summary with final statistics at INFO level.
// It displays total tasks, completed, failed, duration, and overall status.
func (fl *FileLogger) LogSummary(result models.ExecutionResult) {
	// Summary logging is at INFO level
	if !fl.shouldLog("info") {
		return
	}

	timestamp := time.Now().Format("15:04:05")

	// Determine status
	status := "SUCCESS"
	if result.Failed > 0 {
		if result.Completed == 0 {
			status = "FAILED"
		} else {
			status = "PARTIAL"
		}
	}

	// Build summary output
	message := fmt.Sprintf(
		"\n[%s] === EXECUTION SUMMARY ===\n"+
			"[%s] Total tasks:  %d\n"+
			"[%s] Completed:    %d\n"+
			"[%s] Failed:       %d\n"+
			"[%s] Total time:   %.1fs\n"+
			"[%s] Status:       %s (%d/%d tasks passed)\n"+
			"[%s] Completed at: %s\n",
		timestamp,
		timestamp,
		result.TotalTasks,
		timestamp,
		result.Completed,
		timestamp,
		result.Failed,
		timestamp,
		result.Duration.Seconds(),
		timestamp,
		status,
		result.Completed,
		result.TotalTasks,
		timestamp,
		time.Now().Format(time.RFC3339),
	)

	fl.writeRunLog(message)
}

// LogTaskResult logs detailed information about a task execution.
// It creates a separate log file for each task in the tasks/ subdirectory.
func (fl *FileLogger) LogTaskResult(result models.TaskResult) error {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	// Create task log file: tasks/task-N.log
	taskLogPath := filepath.Join(fl.tasksDir, fmt.Sprintf("task-%s.log", result.Task.Number))

	file, err := os.OpenFile(taskLogPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create task log file: %w", err)
	}
	defer file.Close()

	// Write task details
	content := fmt.Sprintf("=== Task %s: %s ===\n", result.Task.Number, result.Task.Name)
	content += fmt.Sprintf("Status: %s\n", result.Status)
	content += fmt.Sprintf("Duration: %.1fs\n", result.Duration.Seconds())
	content += fmt.Sprintf("Retry Count: %d\n", result.RetryCount)
	content += "\n"

	if result.Task.Prompt != "" {
		content += fmt.Sprintf("Prompt:\n%s\n\n", result.Task.Prompt)
	}

	// Log execution history if available (for retry tracking)
	if len(result.ExecutionHistory) > 0 {
		content += "=== Execution History ===\n\n"
		for _, attempt := range result.ExecutionHistory {
			content += fmt.Sprintf("#### Attempt %d (Agent: %s) - %s\n", attempt.Attempt, attempt.Agent, attempt.Verdict)
			content += fmt.Sprintf("Duration: %.1fs\n\n", attempt.Duration.Seconds())

			if attempt.AgentOutput != "" {
				content += fmt.Sprintf("Agent Output (JSON):\n%s\n\n", attempt.AgentOutput)
			}

			if attempt.QCFeedback != "" {
				content += fmt.Sprintf("QC Review (JSON):\n%s\n\n", attempt.QCFeedback)
			}
		}
	}

	// Legacy fields (for compatibility with single-attempt tasks)
	if result.Output != "" {
		content += fmt.Sprintf("Output:\n%s\n\n", result.Output)
	}

	if result.ReviewFeedback != "" {
		content += fmt.Sprintf("QC Feedback:\n%s\n\n", result.ReviewFeedback)
	}

	if result.Error != nil {
		content += fmt.Sprintf("Error:\n%v\n\n", result.Error)
	}

	content += fmt.Sprintf("Completed at: %s\n", time.Now().Format(time.RFC3339))

	_, err = file.WriteString(content)
	if err != nil {
		return fmt.Errorf("failed to write task log: %w", err)
	}

	return nil
}

// LogProgress logs the current execution progress (no-op for file logger).
// Progress is displayed on console but not written to log files.
func (fl *FileLogger) LogProgress(results []models.TaskResult) {
	// No-op: progress bars are console-only for now
}

// LogTaskAgentInvoke is called when a task agent is about to be invoked.
// This is a no-op for file logger.
func (fl *FileLogger) LogTaskAgentInvoke(task models.Task) {
	// No-op: agent invocation details are logged elsewhere
}

// LogQCAgentSelection logs which QC agents were selected for review.
// Format: "[HH:MM:SS] [QC] Selected agents: [agent1, agent2] (mode: auto)"
func (fl *FileLogger) LogQCAgentSelection(agents []string, mode string) {
	// QC logging is at INFO level
	if !fl.shouldLog("info") {
		return
	}

	// Format agents as comma-separated list
	agentsList := fmt.Sprintf("[%s]", strings.Join(agents, ", "))
	message := fmt.Sprintf("[%s] [QC] Selected agents: %s (mode: %s)\n", time.Now().Format("15:04:05"), agentsList, mode)
	fl.writeRunLog(message)
}

// LogQCIndividualVerdicts logs verdicts from each QC agent.
// Format: "[HH:MM:SS] [QC] Individual verdicts: agent1=GREEN, agent2=RED"
func (fl *FileLogger) LogQCIndividualVerdicts(verdicts map[string]string) {
	// QC logging is at DEBUG level (more detailed)
	if !fl.shouldLog("debug") {
		return
	}

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
	message := fmt.Sprintf("[%s] [QC] Individual verdicts: %s\n", time.Now().Format("15:04:05"), verdictsStr)
	fl.writeRunLog(message)
}

// LogQCAggregatedResult logs the final aggregated QC verdict.
// Format: "[HH:MM:SS] [QC] Final verdict: GREEN (strictest-wins)"
func (fl *FileLogger) LogQCAggregatedResult(verdict string, strategy string) {
	// QC logging is at INFO level
	if !fl.shouldLog("info") {
		return
	}

	message := fmt.Sprintf("[%s] [QC] Final verdict: %s (%s)\n", time.Now().Format("15:04:05"), verdict, strategy)
	fl.writeRunLog(message)
}

// LogQCCriteriaResults logs the per-criterion verification results from a QC agent.
// Format: "[HH:MM:SS] [QC] agent-name criteria: PASS" or "[PASS, PASS, FAIL]" for multiple
func (fl *FileLogger) LogQCCriteriaResults(agentName string, results []models.CriterionResult) {
	// Criteria logging is at DEBUG level
	if !fl.shouldLog("debug") {
		return
	}

	if len(results) == 0 {
		return
	}

	// Build criteria status display
	var criteriaStr string
	if len(results) == 1 {
		// Single criterion: just show PASS or FAIL
		if results[0].Passed {
			criteriaStr = "PASS"
		} else {
			criteriaStr = "FAIL"
		}
	} else {
		// Multiple criteria: show [PASS, PASS, FAIL]
		var parts []string
		for _, cr := range results {
			if cr.Passed {
				parts = append(parts, "PASS")
			} else {
				parts = append(parts, "FAIL")
			}
		}
		criteriaStr = fmt.Sprintf("[%s]", strings.Join(parts, ", "))
	}

	message := fmt.Sprintf("[%s] [QC] %s criteria: %s\n", time.Now().Format("15:04:05"), agentName, criteriaStr)
	fl.writeRunLog(message)
}

// LogQCIntelligentSelectionMetadata logs details about intelligent agent selection.
// Format: "[HH:MM:SS] [QC] Intelligent selection: <rationale>" or fallback warning
func (fl *FileLogger) LogQCIntelligentSelectionMetadata(rationale string, fallback bool, fallbackReason string) {
	// QC logging is at INFO level
	if !fl.shouldLog("info") {
		return
	}

	var message string
	ts := time.Now().Format("15:04:05")

	if fallback {
		reason := fallbackReason
		if reason == "" {
			reason = "unknown"
		}
		message = fmt.Sprintf("[%s] [QC] WARN: Fallback to auto mode: %s\n", ts, reason)
	} else if rationale != "" {
		message = fmt.Sprintf("[%s] [QC] Intelligent selection: %s\n", ts, rationale)
	} else {
		return
	}

	fl.writeRunLog(message)
}

// LogGuardPrediction logs GUARD protocol prediction results at INFO level.
// Format: "[HH:MM:SS] [GUARD] Task N: risk_level (probability: X%, confidence: Y%)"
// Only logs blocked tasks and high/medium risk tasks to reduce noise.
func (fl *FileLogger) LogGuardPrediction(taskNumber string, result interface{}) {
	// GUARD logging is at INFO level
	if !fl.shouldLog("info") {
		return
	}

	if result == nil {
		return
	}

	// Type assert to GuardResultDisplay interface (defined in console.go)
	type guardResultDisplay interface {
		GetTaskNumber() string
		GetProbability() float64
		GetConfidence() float64
		GetRiskLevel() string
		ShouldBlock() bool
		GetBlockReason() string
		GetRecommendations() []string
	}

	guard, ok := result.(guardResultDisplay)
	if !ok {
		return
	}

	ts := time.Now().Format("15:04:05")
	var message string

	if guard.ShouldBlock() {
		// Blocked task: show full details
		message = fmt.Sprintf("[%s] [GUARD] Task %s BLOCKED: %s (probability: %.1f%%, confidence: %.1f%%)\n",
			ts, taskNumber, guard.GetBlockReason(), guard.GetProbability()*100, guard.GetConfidence()*100)

		// Add recommendations
		for _, rec := range guard.GetRecommendations() {
			message += fmt.Sprintf("[%s]          → %s\n", ts, rec)
		}
	} else if guard.GetRiskLevel() == "high" || guard.GetRiskLevel() == "medium" {
		// High/medium risk but not blocked: show summary
		message = fmt.Sprintf("[%s] [GUARD] Task %s: %s risk (%.1f%% probability)\n",
			ts, taskNumber, guard.GetRiskLevel(), guard.GetProbability()*100)
	}
	// Low risk tasks are not logged to reduce noise

	if message != "" {
		fl.writeRunLog(message)
	}
}

// Close flushes and closes the run log file.
// It should be called when the logger is no longer needed.
func (fl *FileLogger) Close() error {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	if fl.runLog != nil {
		if err := fl.runLog.Sync(); err != nil {
			return fmt.Errorf("failed to sync run log: %w", err)
		}
		if err := fl.runLog.Close(); err != nil {
			return fmt.Errorf("failed to close run log: %w", err)
		}
		fl.runLog = nil
	}

	return nil
}

// writeRunLog is a thread-safe helper to write to the run log file.
func (fl *FileLogger) writeRunLog(message string) {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	if fl.runLog != nil {
		fl.runLog.WriteString(message)
		// Flush after each write for real-time logging
		fl.runLog.Sync()
	}
}
