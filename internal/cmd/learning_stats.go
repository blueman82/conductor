package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/harrison/conductor/internal/config"
	"github.com/harrison/conductor/internal/learning"
	"github.com/spf13/cobra"
)

// NewStatsCommand creates the 'conductor learning stats' command
func NewStatsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stats <plan-file>",
		Short: "Show learning statistics for a plan",
		Long: `Display learning statistics for a plan file including:
  - Overall success rates
  - Agent performance metrics
  - Task-level statistics
  - Common failure patterns
  - Average execution durations`,
		Args: cobra.ExactArgs(1),
		RunE: runStats,
	}

	return cmd
}

// runStats executes the stats command
func runStats(cmd *cobra.Command, args []string) error {
	planFile := args[0]
	output := cmd.OutOrStdout()

	// Resolve plan file path
	absPath, err := filepath.Abs(planFile)
	if err != nil {
		return fmt.Errorf("resolve plan file path: %w", err)
	}

	// Check if plan file exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("plan file not found: %s", absPath)
	}

	// Use centralized conductor home database location
	dbPath, err := config.GetLearningDBPath()
	if err != nil {
		return fmt.Errorf("failed to get learning database path: %w", err)
	}

	// Check if database exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		fmt.Fprintf(output, "No execution data found for plan: %s\n", filepath.Base(absPath))
		fmt.Fprintf(output, "Database path: %s\n", dbPath)
		return nil
	}

	// Open learning store
	store, err := learning.NewStore(dbPath)
	if err != nil {
		return fmt.Errorf("open learning store: %w", err)
	}
	defer store.Close()

	// Get statistics
	stats, err := getStatistics(store, filepath.Base(absPath))
	if err != nil {
		return fmt.Errorf("get statistics: %w", err)
	}

	// Check if we have any data
	if stats.TotalExecutions == 0 {
		fmt.Fprintf(output, "No execution data found for plan: %s\n", filepath.Base(absPath))
		return nil
	}

	// Print statistics
	printStatistics(output, stats, filepath.Base(absPath))

	return nil
}

// Statistics contains aggregated learning statistics
type Statistics struct {
	TotalExecutions   int
	SuccessfulExecs   int
	FailedExecs       int
	SuccessRate       float64
	AgentPerformance  map[string]*AgentStats
	TaskMetrics       map[string]*TaskStats
	CommonFailures    map[string]int
	AverageDuration   float64
	TotalDurationSecs int64
}

// AgentStats tracks performance for a specific agent
type AgentStats struct {
	Name        string
	TotalExecs  int
	Successes   int
	Failures    int
	SuccessRate float64
	AvgDuration float64
}

// TaskStats tracks metrics for a specific task
type TaskStats struct {
	TaskNumber    string
	TaskName      string
	TotalAttempts int
	Successes     int
	Failures      int
	SuccessRate   float64
	AvgDuration   float64
}

// getStatistics queries the database and calculates statistics
func getStatistics(store *learning.Store, planFile string) (*Statistics, error) {
	stats := &Statistics{
		AgentPerformance: make(map[string]*AgentStats),
		TaskMetrics:      make(map[string]*TaskStats),
		CommonFailures:   make(map[string]int),
	}

	// Query all executions for this plan
	query := `SELECT task_number, task_name, agent, success, output, error_message, duration_seconds
		FROM task_executions
		WHERE plan_file = ?
		ORDER BY id DESC`

	rows, err := store.QueryRows(query, planFile)
	if err != nil {
		return nil, fmt.Errorf("query executions: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var taskNumber, taskName, agent, output, errorMsg string
		var success bool
		var duration int64

		if err := rows.Scan(&taskNumber, &taskName, &agent, &success, &output, &errorMsg, &duration); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}

		// Update overall stats
		stats.TotalExecutions++
		stats.TotalDurationSecs += duration
		if success {
			stats.SuccessfulExecs++
		} else {
			stats.FailedExecs++
			// Extract failure patterns
			extractFailurePatterns(output, errorMsg, stats.CommonFailures)
		}

		// Update agent performance
		if agent != "" {
			agentStats, exists := stats.AgentPerformance[agent]
			if !exists {
				agentStats = &AgentStats{Name: agent}
				stats.AgentPerformance[agent] = agentStats
			}
			agentStats.TotalExecs++
			if success {
				agentStats.Successes++
			} else {
				agentStats.Failures++
			}
			agentStats.AvgDuration = ((agentStats.AvgDuration * float64(agentStats.TotalExecs-1)) + float64(duration)) / float64(agentStats.TotalExecs)
		}

		// Update task metrics
		taskStats, exists := stats.TaskMetrics[taskNumber]
		if !exists {
			taskStats = &TaskStats{
				TaskNumber: taskNumber,
				TaskName:   taskName,
			}
			stats.TaskMetrics[taskNumber] = taskStats
		}
		taskStats.TotalAttempts++
		if success {
			taskStats.Successes++
		} else {
			taskStats.Failures++
		}
		taskStats.AvgDuration = ((taskStats.AvgDuration * float64(taskStats.TotalAttempts-1)) + float64(duration)) / float64(taskStats.TotalAttempts)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rows: %w", err)
	}

	// Calculate derived metrics
	if stats.TotalExecutions > 0 {
		stats.SuccessRate = (float64(stats.SuccessfulExecs) / float64(stats.TotalExecutions)) * 100
		stats.AverageDuration = float64(stats.TotalDurationSecs) / float64(stats.TotalExecutions)
	}

	// Calculate agent success rates
	for _, agentStats := range stats.AgentPerformance {
		if agentStats.TotalExecs > 0 {
			agentStats.SuccessRate = (float64(agentStats.Successes) / float64(agentStats.TotalExecs)) * 100
		}
	}

	// Calculate task success rates
	for _, taskStats := range stats.TaskMetrics {
		if taskStats.TotalAttempts > 0 {
			taskStats.SuccessRate = (float64(taskStats.Successes) / float64(taskStats.TotalAttempts)) * 100
		}
	}

	return stats, nil
}

// extractFailurePatterns extracts common failure patterns from error output
func extractFailurePatterns(output, errorMsg string, patterns map[string]int) {
	combined := strings.ToLower(output + " " + errorMsg)

	knownPatterns := []string{
		"compilation_error",
		"test_failure",
		"dependency_missing",
		"timeout",
		"syntax_error",
		"type_error",
		"import_error",
		"runtime_error",
		"permission_denied",
		"file_not_found",
	}

	for _, pattern := range knownPatterns {
		if strings.Contains(combined, pattern) || strings.Contains(combined, strings.ReplaceAll(pattern, "_", " ")) {
			patterns[pattern]++
		}
	}
}

// printStatistics formats and prints the statistics
func printStatistics(w io.Writer, stats *Statistics, planFile string) {
	// Colors
	cyan := color.New(color.FgCyan, color.Bold)
	green := color.New(color.FgGreen)
	red := color.New(color.FgRed)
	yellow := color.New(color.FgYellow)

	// Header
	cyan.Fprintf(w, "\n=== Learning Statistics for %s ===\n\n", planFile)

	// Overall Statistics
	cyan.Fprintf(w, "Overall Statistics:\n")
	fmt.Fprintf(w, "  Total executions: %d\n", stats.TotalExecutions)
	fmt.Fprintf(w, "  Successful: ")
	green.Fprintf(w, "%d\n", stats.SuccessfulExecs)
	fmt.Fprintf(w, "  Failed: ")
	red.Fprintf(w, "%d\n", stats.FailedExecs)
	fmt.Fprintf(w, "  Success rate: ")
	if stats.SuccessRate >= 70 {
		green.Fprintf(w, "%.1f%%\n", stats.SuccessRate)
	} else if stats.SuccessRate >= 40 {
		yellow.Fprintf(w, "%.1f%%\n", stats.SuccessRate)
	} else {
		red.Fprintf(w, "%.1f%%\n", stats.SuccessRate)
	}
	fmt.Fprintf(w, "  Average duration: %.1f seconds\n", stats.AverageDuration)

	// Agent Performance
	if len(stats.AgentPerformance) > 0 {
		fmt.Fprintf(w, "\n")
		cyan.Fprintf(w, "Agent Performance:\n")

		// Sort agents by success rate (descending)
		agents := make([]*AgentStats, 0, len(stats.AgentPerformance))
		for _, agentStats := range stats.AgentPerformance {
			agents = append(agents, agentStats)
		}
		sort.Slice(agents, func(i, j int) bool {
			return agents[i].SuccessRate > agents[j].SuccessRate
		})

		for _, agentStats := range agents {
			fmt.Fprintf(w, "\n  %s:\n", agentStats.Name)
			fmt.Fprintf(w, "    Executions: %d\n", agentStats.TotalExecs)
			fmt.Fprintf(w, "    Success rate: ")
			if agentStats.SuccessRate >= 70 {
				green.Fprintf(w, "%.1f%%", agentStats.SuccessRate)
			} else if agentStats.SuccessRate >= 40 {
				yellow.Fprintf(w, "%.1f%%", agentStats.SuccessRate)
			} else {
				red.Fprintf(w, "%.1f%%", agentStats.SuccessRate)
			}
			fmt.Fprintf(w, " (%d/%d)\n", agentStats.Successes, agentStats.TotalExecs)
			fmt.Fprintf(w, "    Avg duration: %.1f seconds\n", agentStats.AvgDuration)
		}
	}

	// Task Metrics
	if len(stats.TaskMetrics) > 0 {
		fmt.Fprintf(w, "\n")
		cyan.Fprintf(w, "Task Metrics:\n")

		// Sort tasks by task number
		tasks := make([]*TaskStats, 0, len(stats.TaskMetrics))
		for _, taskStats := range stats.TaskMetrics {
			tasks = append(tasks, taskStats)
		}
		sort.Slice(tasks, func(i, j int) bool {
			return tasks[i].TaskNumber < tasks[j].TaskNumber
		})

		for _, taskStats := range tasks {
			fmt.Fprintf(w, "\n  Task %s: %s\n", taskStats.TaskNumber, taskStats.TaskName)
			fmt.Fprintf(w, "    Attempts: %d\n", taskStats.TotalAttempts)
			fmt.Fprintf(w, "    Success rate: ")
			if taskStats.SuccessRate >= 70 {
				green.Fprintf(w, "%.1f%%", taskStats.SuccessRate)
			} else if taskStats.SuccessRate >= 40 {
				yellow.Fprintf(w, "%.1f%%", taskStats.SuccessRate)
			} else {
				red.Fprintf(w, "%.1f%%", taskStats.SuccessRate)
			}
			fmt.Fprintf(w, " (%d/%d)\n", taskStats.Successes, taskStats.TotalAttempts)
			fmt.Fprintf(w, "    Avg duration: %.1f seconds\n", taskStats.AvgDuration)
		}
	}

	// Common Failures
	if len(stats.CommonFailures) > 0 {
		fmt.Fprintf(w, "\n")
		cyan.Fprintf(w, "Common Failure Patterns:\n")

		// Sort by frequency (descending)
		type patternCount struct {
			pattern string
			count   int
		}
		patterns := make([]patternCount, 0, len(stats.CommonFailures))
		for pattern, count := range stats.CommonFailures {
			patterns = append(patterns, patternCount{pattern, count})
		}
		sort.Slice(patterns, func(i, j int) bool {
			return patterns[i].count > patterns[j].count
		})

		for _, pc := range patterns {
			fmt.Fprintf(w, "  - %s: ", strings.ReplaceAll(pc.pattern, "_", " "))
			red.Fprintf(w, "%d occurrences\n", pc.count)
		}
	}

	fmt.Fprintf(w, "\n")
}
