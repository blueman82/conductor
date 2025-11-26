package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/harrison/conductor/internal/config"
	"github.com/harrison/conductor/internal/learning"
)

// DisplayStats displays summary statistics from the behavioral database
func DisplayStats(project string, limit int) error {
	// Get learning DB path (uses build-time injected root)
	dbPath, err := config.GetLearningDBPath()
	if err != nil {
		return fmt.Errorf("get learning db path: %w", err)
	}

	// Open learning store
	store, err := learning.NewStore(dbPath)
	if err != nil {
		return fmt.Errorf("open learning store: %w", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Get summary stats
	summaryStats, err := store.GetSummaryStats(ctx, project)
	if err != nil {
		return fmt.Errorf("get summary stats: %w", err)
	}

	// Get agent type stats with pagination
	agentStats, err := store.GetAgentTypeStats(ctx, project, limit, 0)
	if err != nil {
		return fmt.Errorf("get agent type stats: %w", err)
	}

	// Display formatted output
	fmt.Println(formatStatsTable(summaryStats, agentStats, limit))

	return nil
}

// formatStatsTable formats statistics as a readable table using new aggregated stats
func formatStatsTable(summary *learning.SummaryStats, agents []learning.AgentTypeStats, limit int) string {
	var sb strings.Builder

	sb.WriteString("\n=== SUMMARY STATISTICS ===\n\n")

	// Overall metrics
	sb.WriteString(fmt.Sprintf("Total Sessions:      %d\n", summary.TotalSessions))
	sb.WriteString(fmt.Sprintf("Total Agents:        %d\n", summary.TotalAgents))
	sb.WriteString(fmt.Sprintf("Success Rate:        %.1f%%\n", summary.SuccessRate*100))
	sb.WriteString(fmt.Sprintf("Average Duration:    %s\n", formatDuration(time.Duration(summary.AvgDurationSeconds)*time.Second)))

	// Token usage and cost
	sb.WriteString(fmt.Sprintf("\nTotal Input Tokens:  %d\n", summary.TotalInputTokens))
	sb.WriteString(fmt.Sprintf("Total Output Tokens: %d\n", summary.TotalOutputTokens))
	sb.WriteString(fmt.Sprintf("Total Tokens:        %d\n", summary.TotalTokens))
	if summary.AvgTokensPerSession > 0 {
		sb.WriteString(fmt.Sprintf("Avg Tokens/Session:  %.0f\n", summary.AvgTokensPerSession))
	}
	if summary.TotalCostUSD > 0 {
		sb.WriteString(fmt.Sprintf("Estimated Cost:      $%.2f (Sonnet pricing)\n", summary.TotalCostUSD))
	}

	// Agent breakdown
	if len(agents) > 0 {
		sb.WriteString("\n--- Agent Performance (Top")
		if limit > 0 && len(agents) > limit {
			sb.WriteString(fmt.Sprintf(" %d of %d", limit, len(agents)))
		} else if len(agents) == 1 {
			sb.WriteString(" 1")
		}
		sb.WriteString(") ---\n")

		displayLimit := limit
		if limit <= 0 || displayLimit > len(agents) {
			displayLimit = len(agents)
		}

		// Header - Agent Type at end for full display
		sb.WriteString(fmt.Sprintf("%-10s %-10s %-10s %-12s %-8s %s\n",
			"Sessions", "Success", "Failures", "Duration", "Rate", "Agent Type"))
		sb.WriteString(strings.Repeat("-", 90) + "\n")

		for i := 0; i < displayLimit; i++ {
			agent := agents[i]
			durationStr := "0s"
			if agent.AvgDurationSeconds > 0 {
				durationStr = formatDuration(time.Duration(agent.AvgDurationSeconds) * time.Second)
			}
			successPct := 0.0
			if agent.TotalSessions > 0 {
				successPct = agent.SuccessRate * 100
			}

			sb.WriteString(fmt.Sprintf("%-10d %-10d %-10d %-12s %6.1f%%  %s\n",
				agent.TotalSessions,
				agent.SuccessCount,
				agent.FailureCount,
				durationStr,
				successPct,
				agent.AgentType))
		}

		if limit > 0 && len(agents) > displayLimit {
			sb.WriteString(fmt.Sprintf("\n(Showing %d of %d agents. Use --limit flag to see more)\n", displayLimit, len(agents)))
		}
	}

	return sb.String()
}

// collectBehavioralMetrics is a stub function for test compatibility
// This function is deprecated - use store.GetSummaryStats() instead
func collectBehavioralMetrics(store *learning.Store, project string) ([]interface{}, error) {
	// Return empty slice for compatibility
	return []interface{}{}, nil
}

// DisplayRecentSessions displays a list of recent sessions
func DisplayRecentSessions(project string, limit int) error {
	// Get learning DB path (uses build-time injected root)
	dbPath, err := config.GetLearningDBPath()
	if err != nil {
		return fmt.Errorf("get learning db path: %w", err)
	}

	// Open learning store
	store, err := learning.NewStore(dbPath)
	if err != nil {
		return fmt.Errorf("open learning store: %w", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Get recent sessions
	sessions, err := store.GetRecentSessions(ctx, project, limit, 0)
	if err != nil {
		return fmt.Errorf("get recent sessions: %w", err)
	}

	// Display formatted output
	fmt.Println(formatRecentSessionsTable(sessions, limit))

	return nil
}

// formatRecentSessionsTable formats recent sessions as a readable table
func formatRecentSessionsTable(sessions []learning.RecentSession, limit int) string {
	var sb strings.Builder

	sb.WriteString("\n=== Recent Sessions ===\n\n")

	if len(sessions) == 0 {
		sb.WriteString("No sessions found\n")
		return sb.String()
	}

	// Header - Task and Agent at end for full display
	sb.WriteString(fmt.Sprintf("%-8s %-10s %-12s %-20s %-20s %s\n",
		"ID", "Status", "Duration", "Started", "Agent", "Task"))
	sb.WriteString(strings.Repeat("-", 100) + "\n")

	// Display rows
	for _, session := range sessions {
		status := "SUCCESS"
		if !session.Success {
			status = "FAILED"
		}
		durationStr := formatDuration(time.Duration(session.DurationSecs) * time.Second)

		sb.WriteString(fmt.Sprintf("%-8d %-10s %-12s %-20s %-20s %s\n",
			session.ID,
			status,
			durationStr,
			session.Timestamp.Format("2006-01-02 15:04:05"),
			session.Agent,
			session.TaskName))
	}

	sb.WriteString(fmt.Sprintf("\n(Showing %d sessions. Use --limit/-n to see more)\n", len(sessions)))
	sb.WriteString("Use 'conductor observe session <ID>' for detailed analysis.\n")

	return sb.String()
}
