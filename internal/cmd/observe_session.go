package cmd

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/harrison/conductor/internal/behavioral"
	"github.com/harrison/conductor/internal/config"
	"github.com/harrison/conductor/internal/learning"
)

// DisplaySessionAnalysis displays detailed analysis for a specific session
func DisplaySessionAnalysis(sessionID, project string) error {
	if sessionID == "" {
		return fmt.Errorf("session ID is required")
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

	// Try aggregator for JSONL-based metrics first
	aggregator := behavioral.NewAggregator(50)

	var metrics *behavioral.BehavioralMetrics
	var sessionData *behavioral.SessionData
	var sessionInfo *behavioral.SessionInfo

	// If project specified, use it directly
	if project != "" {
		metrics, err = aggregator.GetSessionMetrics(project, sessionID)
		if err == nil {
			// Also get session info
			sessions, _ := aggregator.ListSessions(project)
			for _, s := range sessions {
				if s.SessionID == sessionID {
					sessionInfo = &s
					break
				}
			}
			// Load raw session data for timeline
			if sessionInfo != nil {
				sessionData, _ = behavioral.ParseSessionFile(sessionInfo.FilePath)
			}
		}
	} else {
		// Search across all projects for the session
		sessionInfo, metrics, sessionData, err = findSessionAcrossProjects(aggregator, sessionID)
		if err != nil {
			// Fall back to DB-based search by session ID number
			return displayDBSessionAnalysis(ctx, store, sessionID)
		}
	}

	if metrics == nil && sessionInfo == nil {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Display output
	fmt.Println(formatSessionAnalysis(sessionID, project, sessionInfo, metrics, sessionData))

	return nil
}

// findSessionAcrossProjects searches all projects for a session ID
func findSessionAcrossProjects(aggregator *behavioral.Aggregator, sessionID string) (*behavioral.SessionInfo, *behavioral.BehavioralMetrics, *behavioral.SessionData, error) {
	// Discover all sessions
	sessions, err := behavioral.DiscoverSessions("~/.claude/projects")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("discover sessions: %w", err)
	}

	// Search for matching session ID (partial match supported)
	for _, session := range sessions {
		if session.SessionID == sessionID || strings.Contains(session.SessionID, sessionID) {
			metrics, err := aggregator.LoadSession(session.FilePath)
			if err != nil {
				continue
			}
			sessionData, _ := behavioral.ParseSessionFile(session.FilePath)
			return &session, metrics, sessionData, nil
		}
	}

	return nil, nil, nil, fmt.Errorf("session not found: %s", sessionID)
}

// displayDBSessionAnalysis displays session analysis from database
func displayDBSessionAnalysis(ctx context.Context, store *learning.Store, sessionID string) error {
	// Try to parse as numeric ID
	var numericID int64
	_, err := fmt.Sscanf(sessionID, "%d", &numericID)
	if err != nil {
		return fmt.Errorf("session not found: %s (not a valid numeric ID)", sessionID)
	}

	metrics, err := store.GetSessionMetrics(ctx, numericID)
	if err != nil {
		return fmt.Errorf("session not found: %w", err)
	}

	fmt.Println(formatDBSessionAnalysis(metrics))
	return nil
}

// formatSessionAnalysis formats the session analysis as a readable output
func formatSessionAnalysis(sessionID, project string, info *behavioral.SessionInfo, metrics *behavioral.BehavioralMetrics, sessionData *behavioral.SessionData) string {
	var sb strings.Builder

	// Header
	sb.WriteString(fmt.Sprintf("\n=== Session: %s ===\n\n", sessionID))

	// Metadata section
	if info != nil {
		sb.WriteString(fmt.Sprintf("Project:         %s\n", info.Project))
		sb.WriteString(fmt.Sprintf("File:            %s\n", info.Filename))
		sb.WriteString(fmt.Sprintf("Size:            %s\n", formatBytes(info.FileSize)))
		sb.WriteString(fmt.Sprintf("Created:         %s\n", info.CreatedAt.Format("2006-01-02 15:04:05")))
	} else if project != "" {
		sb.WriteString(fmt.Sprintf("Project:         %s\n", project))
	}

	if metrics != nil {
		// Duration and status
		sb.WriteString(fmt.Sprintf("Duration:        %s\n", formatDuration(metrics.AverageDuration)))

		status := "success"
		if metrics.SuccessRate < 0.5 {
			status = "failed"
		}
		sb.WriteString(fmt.Sprintf("Status:          %s\n", status))
		sb.WriteString(fmt.Sprintf("Success Rate:    %.1f%%\n", metrics.SuccessRate*100))

		// Token usage section
		if metrics.TokenUsage.InputTokens > 0 || metrics.TokenUsage.OutputTokens > 0 {
			sb.WriteString("\n--- Token Usage ---\n")
			sb.WriteString(fmt.Sprintf("Input Tokens:    %s\n", formatNumber(metrics.TokenUsage.InputTokens)))
			sb.WriteString(fmt.Sprintf("Output Tokens:   %s\n", formatNumber(metrics.TokenUsage.OutputTokens)))
			sb.WriteString(fmt.Sprintf("Total:           %s\n", formatNumber(metrics.TokenUsage.TotalTokens())))
			if metrics.TokenUsage.CostUSD > 0 {
				sb.WriteString(fmt.Sprintf("Cost:            $%.4f\n", metrics.TokenUsage.CostUSD))
			}
		}
	}

	// Tool timeline from session data
	if sessionData != nil && len(sessionData.Events) > 0 {
		sb.WriteString("\n--- Tool Timeline ---\n")

		// Sort events by timestamp
		events := sessionData.Events
		sort.Slice(events, func(i, j int) bool {
			return events[i].GetTimestamp().Before(events[j].GetTimestamp())
		})

		// Collect errors for summary
		var errors []errorEntry

		// Display timeline
		for _, event := range events {
			ts := event.GetTimestamp().Format("15:04:05")

			switch e := event.(type) {
			case *behavioral.ToolCallEvent:
				if e.Type == "tool_result" {
					continue // Skip tool results in timeline (they're processed with errors)
				}
				status := "success"
				if !e.Success {
					status = "failed"
					errors = append(errors, errorEntry{
						timestamp: e.Timestamp,
						errType:   e.ToolName,
						message:   e.Error,
					})
				}
				durationStr := formatMillis(e.Duration)
				// Truncate tool name for display
				toolName := truncateString(e.ToolName, 12)
				sb.WriteString(fmt.Sprintf("%s  %-12s %8s  %s\n", ts, toolName, status, durationStr))

			case *behavioral.BashCommandEvent:
				status := "success"
				if !e.Success {
					status = "failed"
					errors = append(errors, errorEntry{
						timestamp: e.Timestamp,
						errType:   "Bash",
						message:   fmt.Sprintf("exit code %d", e.ExitCode),
					})
				}
				durationStr := formatMillis(e.Duration)
				cmd := truncateString(e.Command, 30)
				sb.WriteString(fmt.Sprintf("%s  %-12s %8s  %s  %s\n", ts, "Bash", status, durationStr, cmd))

			case *behavioral.FileOperationEvent:
				status := "success"
				if !e.Success {
					status = "failed"
					errors = append(errors, errorEntry{
						timestamp: e.Timestamp,
						errType:   e.Operation,
						message:   e.Error,
					})
				}
				durationStr := formatMillis(e.Duration)
				path := truncateString(e.Path, 30)
				sb.WriteString(fmt.Sprintf("%s  %-12s %8s  %s  %s\n", ts, e.Operation, status, durationStr, path))
			}
		}

		// Errors summary
		if len(errors) > 0 {
			sb.WriteString(fmt.Sprintf("\n--- Errors (%d) ---\n", len(errors)))
			for i, err := range errors {
				ts := err.timestamp.Format("15:04:05")
				sb.WriteString(fmt.Sprintf("%d. [%s] %s: %s\n", i+1, ts, err.errType, err.message))
			}
		}
	} else if metrics != nil {
		// Fallback: show tool usage from metrics if no timeline
		if len(metrics.ToolExecutions) > 0 {
			sb.WriteString("\n--- Tool Usage ---\n")
			sb.WriteString(fmt.Sprintf("%-20s %10s %10s\n", "Tool", "Count", "Success%"))
			sb.WriteString(strings.Repeat("-", 42) + "\n")

			for _, tool := range metrics.ToolExecutions {
				sb.WriteString(fmt.Sprintf("%-20s %10d %9.1f%%\n",
					truncateString(tool.Name, 19),
					tool.Count,
					tool.SuccessRate*100))
			}
		}

		// Show errors if any
		if metrics.TotalErrors > 0 {
			sb.WriteString(fmt.Sprintf("\n--- Errors (%d) ---\n", metrics.TotalErrors))
			sb.WriteString(fmt.Sprintf("Error Rate: %.1f%%\n", metrics.ErrorRate*100))
		}
	}

	return sb.String()
}

// formatDBSessionAnalysis formats DB-based session metrics
func formatDBSessionAnalysis(metrics *learning.BehavioralSessionMetrics) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("\n=== Session: %d ===\n\n", metrics.SessionID))

	sb.WriteString(fmt.Sprintf("Task Number:     %s\n", metrics.TaskNumber))
	sb.WriteString(fmt.Sprintf("Task Name:       %s\n", metrics.TaskName))
	sb.WriteString(fmt.Sprintf("Agent:           %s\n", metrics.Agent))

	status := "success"
	if !metrics.Success {
		status = "failed"
	}
	sb.WriteString(fmt.Sprintf("Status:          %s\n", status))

	sb.WriteString(fmt.Sprintf("Started:         %s\n", metrics.SessionStart.Format("2006-01-02 15:04:05")))
	if metrics.SessionEnd != nil {
		sb.WriteString(fmt.Sprintf("Ended:           %s\n", metrics.SessionEnd.Format("2006-01-02 15:04:05")))
	}
	sb.WriteString(fmt.Sprintf("Duration:        %s\n", formatDuration(time.Duration(metrics.TotalDurationSecs)*time.Second)))

	sb.WriteString("\n--- Activity ---\n")
	sb.WriteString(fmt.Sprintf("Tool Calls:      %d\n", metrics.TotalToolCalls))
	sb.WriteString(fmt.Sprintf("Bash Commands:   %d\n", metrics.TotalBashCommands))
	sb.WriteString(fmt.Sprintf("File Operations: %d\n", metrics.TotalFileOperations))

	if metrics.TotalTokensUsed > 0 {
		sb.WriteString("\n--- Token Usage ---\n")
		sb.WriteString(fmt.Sprintf("Total Tokens:    %s\n", formatNumber(metrics.TotalTokensUsed)))
		if metrics.ContextWindowUsed > 0 {
			sb.WriteString(fmt.Sprintf("Context Window:  %d%%\n", metrics.ContextWindowUsed))
		}
	}

	return sb.String()
}

// errorEntry holds error info for display
type errorEntry struct {
	timestamp time.Time
	errType   string
	message   string
}

// formatNumber formats large numbers with commas
func formatNumber(n int64) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1000000 {
		return fmt.Sprintf("%d,%03d", n/1000, n%1000)
	}
	return fmt.Sprintf("%d,%03d,%03d", n/1000000, (n/1000)%1000, n%1000)
}

// formatMillis formats milliseconds as human-readable duration
func formatMillis(ms int64) string {
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	if ms < 60000 {
		return fmt.Sprintf("%.1fs", float64(ms)/1000)
	}
	return fmt.Sprintf("%.1fm", float64(ms)/60000)
}
