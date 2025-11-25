package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/harrison/conductor/internal/behavioral"
	"github.com/harrison/conductor/internal/config"
	"github.com/harrison/conductor/internal/learning"
)

// DisplayProjectAnalysis displays comprehensive project-level metrics
func DisplayProjectAnalysis(project string, limit int) error {
	if project == "" {
		return fmt.Errorf("project name is required (use --project flag or provide as argument)")
	}

	// Load config to get DB path
	cfg, err := config.LoadConfigFromDir(".")
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Open learning store for DB-backed stats
	store, err := learning.NewStore(cfg.Learning.DBPath)
	if err != nil {
		return fmt.Errorf("open learning store: %w", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Try to get summary stats from DB
	summaryStats, err := store.GetSummaryStats(ctx, project)
	if err != nil {
		return fmt.Errorf("get summary stats: %w", err)
	}

	// Try to get basic project stats from JSONL files
	projectStats, projectStatsErr := behavioral.GetProjectStats(project)

	// Try aggregator for richer JSONL-based metrics
	aggregator := behavioral.NewAggregator(50)
	projectMetrics, projectMetricsErr := aggregator.GetProjectMetrics(project)

	// Get agent type stats for breakdown
	agentStats, agentErr := store.GetAgentTypeStats(ctx, project, limit, 0)

	// Get tool stats
	toolStats, toolErr := store.GetToolStats(ctx, project, limit, 0)

	// Get recent sessions
	recentSessions, sessionsErr := store.GetRecentSessions(ctx, project, 10, 0)

	// Display output
	fmt.Println(formatProjectAnalysis(
		project,
		summaryStats,
		projectStats, projectStatsErr,
		projectMetrics, projectMetricsErr,
		agentStats, agentErr,
		toolStats, toolErr,
		recentSessions, sessionsErr,
		limit,
	))

	return nil
}

// formatProjectAnalysis formats the project analysis as a readable output
func formatProjectAnalysis(
	project string,
	summary *learning.SummaryStats,
	projectStats *behavioral.ProjectStats, projectStatsErr error,
	projectMetrics *behavioral.AggregateProjectMetrics, projectMetricsErr error,
	agentStats []learning.AgentTypeStats, agentErr error,
	toolStats []learning.ToolStats, toolErr error,
	recentSessions []learning.RecentSession, sessionsErr error,
	limit int,
) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("\n=== Project: %s ===\n\n", project))

	// Summary section - prefer DB stats, supplement with JSONL stats
	sb.WriteString("--- Overview ---\n")
	sb.WriteString(fmt.Sprintf("Sessions:        %d\n", summary.TotalSessions))
	sb.WriteString(fmt.Sprintf("Success Rate:    %.1f%%\n", summary.SuccessRate*100))
	sb.WriteString(fmt.Sprintf("Average Duration: %s\n", formatDuration(time.Duration(summary.AvgDurationSeconds)*time.Second)))

	// Add JSONL-based info if available
	if projectStatsErr == nil && projectStats != nil {
		if projectStats.LastModified != "" {
			sb.WriteString(fmt.Sprintf("Last Activity:   %s\n", projectStats.LastModified))
		}
		if projectStats.TotalSize > 0 {
			sb.WriteString(fmt.Sprintf("Data Size:       %s\n", formatBytes(projectStats.TotalSize)))
		}
	}

	// Token usage if available
	if summary.TotalTokens > 0 {
		sb.WriteString(fmt.Sprintf("\nTotal Tokens:    %d\n", summary.TotalTokens))
		if summary.AvgTokensPerSession > 0 {
			sb.WriteString(fmt.Sprintf("Avg/Session:     %.0f\n", summary.AvgTokensPerSession))
		}
	}

	// Aggregator metrics if available (JSONL-based)
	if projectMetricsErr == nil && projectMetrics != nil {
		if projectMetrics.TotalCost > 0 {
			sb.WriteString(fmt.Sprintf("Estimated Cost:  $%.4f\n", projectMetrics.TotalCost))
		}
		if projectMetrics.TotalErrors > 0 {
			sb.WriteString(fmt.Sprintf("Total Errors:    %d\n", projectMetrics.TotalErrors))
		}
	}

	// Tool usage section
	if toolErr == nil && len(toolStats) > 0 {
		sb.WriteString("\n--- Tool Usage ---\n")
		sb.WriteString(fmt.Sprintf("%-20s %10s %10s\n", "Tool", "Calls", "Success%"))
		sb.WriteString(strings.Repeat("-", 42) + "\n")

		displayCount := limit
		if displayCount <= 0 || displayCount > len(toolStats) {
			displayCount = len(toolStats)
		}

		for i := 0; i < displayCount; i++ {
			tool := toolStats[i]
			sb.WriteString(fmt.Sprintf("%-20s %10d %9.1f%%\n",
				truncateString(tool.ToolName, 19),
				tool.CallCount,
				tool.SuccessRate*100))
		}

		if limit > 0 && len(toolStats) > displayCount {
			sb.WriteString(fmt.Sprintf("(Showing %d of %d tools)\n", displayCount, len(toolStats)))
		}
	}

	// Agent performance section
	if agentErr == nil && len(agentStats) > 0 {
		sb.WriteString("\n--- Agent Performance ---\n")
		sb.WriteString(fmt.Sprintf("%-25s %10s %10s %12s\n", "Agent", "Sessions", "Success%", "Avg Duration"))
		sb.WriteString(strings.Repeat("-", 60) + "\n")

		displayCount := limit
		if displayCount <= 0 || displayCount > len(agentStats) {
			displayCount = len(agentStats)
		}

		for i := 0; i < displayCount; i++ {
			agent := agentStats[i]
			durationStr := "0s"
			if agent.AvgDurationSeconds > 0 {
				durationStr = formatDuration(time.Duration(agent.AvgDurationSeconds) * time.Second)
			}
			sb.WriteString(fmt.Sprintf("%-25s %10d %9.1f%% %12s\n",
				truncateString(agent.AgentType, 24),
				agent.TotalSessions,
				agent.SuccessRate*100,
				durationStr))
		}

		if limit > 0 && len(agentStats) > displayCount {
			sb.WriteString(fmt.Sprintf("(Showing %d of %d agents)\n", displayCount, len(agentStats)))
		}
	}

	// Recent sessions section
	if sessionsErr == nil && len(recentSessions) > 0 {
		sb.WriteString("\n--- Recent Sessions ---\n")
		sb.WriteString(fmt.Sprintf("%-20s %12s %8s %s\n", "Agent", "Duration", "Status", "Task"))
		sb.WriteString(strings.Repeat("-", 70) + "\n")

		for _, session := range recentSessions {
			status := "success"
			if !session.Success {
				status = "failed"
			}
			agent := session.Agent
			if agent == "" {
				agent = "default"
			}
			taskName := session.TaskName
			if len(taskName) > 25 {
				taskName = taskName[:22] + "..."
			}
			sb.WriteString(fmt.Sprintf("%-20s %12s %8s %s\n",
				truncateString(agent, 19),
				formatDuration(time.Duration(session.DurationSecs)*time.Second),
				status,
				taskName))
		}
	}

	return sb.String()
}

// formatBytes formats bytes as human-readable string
func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
