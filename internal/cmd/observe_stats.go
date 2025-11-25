package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/harrison/conductor/internal/behavioral"
	"github.com/harrison/conductor/internal/config"
	"github.com/harrison/conductor/internal/learning"
)

// DisplayStats displays summary statistics from the behavioral database
func DisplayStats(project string, limit int) error {
	// Load config to get DB path
	cfg, err := config.LoadConfigFromDir(".")
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Open learning store
	store, err := learning.NewStore(cfg.Learning.DBPath)
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

	// Token usage
	sb.WriteString(fmt.Sprintf("\nTotal Input Tokens:  %d\n", summary.TotalInputTokens))
	sb.WriteString(fmt.Sprintf("Total Output Tokens: %d\n", summary.TotalOutputTokens))
	sb.WriteString(fmt.Sprintf("Total Tokens:        %d\n", summary.TotalTokens))
	if summary.AvgTokensPerSession > 0 {
		sb.WriteString(fmt.Sprintf("Avg Tokens/Session:  %.0f\n", summary.AvgTokensPerSession))
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

		// Header
		sb.WriteString(fmt.Sprintf("%-30s %-12s %-12s %-12s %-12s %-10s\n",
			"Agent Type", "Sessions", "Success", "Failures", "Avg Duration", "Success%"))
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

			sb.WriteString(fmt.Sprintf("%-30s %-12d %-12d %-12d %-12s %-10.1f%%\n",
				truncateString(agent.AgentType, 29),
				agent.TotalSessions,
				agent.SuccessCount,
				agent.FailureCount,
				durationStr,
				successPct))
		}

		if limit > 0 && len(agents) > displayLimit {
			sb.WriteString(fmt.Sprintf("\n(Showing %d of %d agents. Use --limit flag to see more)\n", displayLimit, len(agents)))
		}
	}

	return sb.String()
}

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// StreamActivity streams real-time activity from the behavioral database, like `tail -f`
func StreamActivity(ctx context.Context, project string, pollInterval time.Duration, withIngest bool) error {
	// Load config to get DB path
	cfg, err := config.LoadConfigFromDir(".")
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Open learning store
	store, err := learning.NewStore(cfg.Learning.DBPath)
	if err != nil {
		return fmt.Errorf("open learning store: %w", err)
	}
	defer store.Close()

	// Optionally start ingestion daemon
	var engine *behavioral.IngestionEngine
	if withIngest {
		engine, err = startIngestionDaemon(ctx, store)
		if err != nil {
			return fmt.Errorf("start ingestion daemon: %w", err)
		}
		defer func() {
			if stopErr := engine.Stop(); stopErr != nil {
				fmt.Fprintf(os.Stderr, "Warning: error stopping ingestion engine: %v\n", stopErr)
			}
		}()
		fmt.Println("Ingestion daemon started for live JSONL processing")
	}

	// Get the maximum session ID to skip historical data
	lastSeenID, err := getMaxSessionID(store, project)
	if err != nil {
		return fmt.Errorf("get max session id: %w", err)
	}

	// Display ready message
	displayStreamHeader(lastSeenID)

	// Poll for new sessions
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			// Query for new sessions
			updates, err := watchNewSessions(store, project, lastSeenID)
			if err != nil {
				return fmt.Errorf("watch sessions: %w", err)
			}

			// Display new sessions
			for _, update := range updates {
				displaySessionUpdate(update)
				if update.ID > lastSeenID {
					lastSeenID = update.ID
				}
			}
		}
	}
}

// startIngestionDaemon initializes and starts the ingestion engine
func startIngestionDaemon(ctx context.Context, store *learning.Store) (*behavioral.IngestionEngine, error) {
	// Determine root directory for JSONL files
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home directory: %w", err)
	}
	rootDir := filepath.Join(homeDir, ".claude", "projects")

	// Verify root directory exists
	if _, err := os.Stat(rootDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("root directory does not exist: %s", rootDir)
	}

	// Configure ingestion engine
	cfg := behavioral.IngestionConfig{
		RootDir:      rootDir,
		Pattern:      "*.jsonl",
		BatchSize:    50,
		BatchTimeout: 500 * time.Millisecond,
	}

	// Create ingestion engine
	engine, err := behavioral.NewIngestionEngine(store, cfg)
	if err != nil {
		return nil, fmt.Errorf("create ingestion engine: %w", err)
	}

	// Start the engine in a goroutine (non-blocking)
	if err := engine.Start(ctx); err != nil {
		return nil, fmt.Errorf("start ingestion engine: %w", err)
	}

	return engine, nil
}

// getMaxSessionID returns the current maximum session ID to avoid showing historical data
func getMaxSessionID(store *learning.Store, project string) (int, error) {
	query := `SELECT COALESCE(MAX(bs.id), 0) FROM behavioral_sessions bs`

	args := []interface{}{}

	if project != "" {
		query += ` JOIN task_executions te ON bs.task_execution_id = te.id
		          WHERE te.plan_file LIKE ?`
		args = append(args, "%"+project+"%")
	}

	var maxID int
	rows, err := store.QueryRows(query, args...)
	if err != nil {
		return 0, fmt.Errorf("query max session id: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		if err := rows.Scan(&maxID); err != nil {
			return 0, fmt.Errorf("scan max id: %w", err)
		}
	}

	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterate rows: %w", err)
	}

	return maxID, nil
}

// displayStreamHeader shows the ready message when streaming starts
func displayStreamHeader(lastSeenID int) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("Real-time Activity Stream (like 'tail -f')")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("Ready - watching for new activity... (Ctrl+C to stop)")
	fmt.Println()
}

// SessionUpdate represents a new session detected
type SessionUpdate struct {
	ID              int
	TaskName        string
	Agent           string
	Timestamp       time.Time
	Success         bool
	Duration        time.Duration
	ToolCalls       int
	BashCommands    int
	FileOperations  int
}

// watchNewSessions queries for sessions added since lastSeenID in chronological order
func watchNewSessions(store *learning.Store, project string, lastSeenID int) ([]SessionUpdate, error) {
	query := `
		SELECT
			bs.id,
			te.task_name,
			te.agent,
			bs.session_start,
			te.success,
			bs.total_duration_seconds,
			bs.total_tool_calls,
			bs.total_bash_commands,
			bs.total_file_operations
		FROM behavioral_sessions bs
		JOIN task_executions te ON bs.task_execution_id = te.id
		WHERE bs.id > ?
	`

	args := []interface{}{lastSeenID}

	if project != "" {
		query += " AND te.plan_file LIKE ?"
		args = append(args, "%"+project+"%")
	}

	// Use chronological order (oldest new session first)
	query += " ORDER BY bs.session_start ASC LIMIT 10"

	rows, err := store.QueryRows(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query new sessions: %w", err)
	}
	defer rows.Close()

	var updates []SessionUpdate
	for rows.Next() {
		var update SessionUpdate
		var durationSecs int64
		var agent sql.NullString

		if err := rows.Scan(&update.ID, &update.TaskName, &agent, &update.Timestamp, &update.Success, &durationSecs,
			&update.ToolCalls, &update.BashCommands, &update.FileOperations); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}

		if agent.Valid {
			update.Agent = agent.String
		}
		update.Duration = time.Duration(durationSecs) * time.Second

		updates = append(updates, update)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sessions: %w", err)
	}

	return updates, nil
}

// displaySessionUpdate displays a single session update with rich formatting
func displaySessionUpdate(update SessionUpdate) {
	status := "SUCCESS"
	emoji := "✓"
	if !update.Success {
		status = "FAILED"
		emoji = "✗"
	}

	agent := update.Agent
	if agent == "" {
		agent = "default"
	}

	// Build activity description
	parts := []string{}
	if update.ToolCalls > 0 {
		parts = append(parts, fmt.Sprintf("%d tool%s", update.ToolCalls, pluralize(update.ToolCalls)))
	}
	if update.BashCommands > 0 {
		parts = append(parts, fmt.Sprintf("%d bash cmd%s", update.BashCommands, pluralize(update.BashCommands)))
	}
	if update.FileOperations > 0 {
		parts = append(parts, fmt.Sprintf("%d file op%s", update.FileOperations, pluralize(update.FileOperations)))
	}

	description := strings.Join(parts, ", ")
	if description == "" {
		description = "no activities"
	}

	fmt.Printf("[%s] %-20s %s %s | %s (%s)\n",
		update.Timestamp.Format("15:04:05"),
		agent,
		emoji,
		status,
		description,
		formatDuration(update.Duration))
}

// pluralize returns "s" if count != 1, otherwise ""
func pluralize(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

// collectBehavioralMetrics is a stub function for test compatibility
// This function is deprecated - use store.GetSummaryStats() instead
func collectBehavioralMetrics(store *learning.Store, project string) ([]interface{}, error) {
	// Return empty slice for compatibility
	return []interface{}{}, nil
}
