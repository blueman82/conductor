package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/harrison/conductor/internal/behavioral"
	"github.com/harrison/conductor/internal/config"
	"github.com/harrison/conductor/internal/learning"
)

// DisplayStats displays summary statistics from the behavioral database
func DisplayStats(project string) error {
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

	// Collect metrics
	metrics, err := collectBehavioralMetrics(store, project)
	if err != nil {
		return fmt.Errorf("collect metrics: %w", err)
	}

	// Calculate stats
	stats := behavioral.CalculateStats(metrics)

	// Display formatted output
	fmt.Println(formatStatsTable(stats))

	return nil
}

// collectBehavioralMetrics collects metrics from the database
func collectBehavioralMetrics(store *learning.Store, project string) ([]behavioral.BehavioralMetrics, error) {
	// Query behavioral_sessions table
	query := `
		SELECT
			bs.id,
			bs.total_duration_seconds,
			bs.total_tool_calls,
			bs.total_bash_commands,
			bs.total_file_operations,
			te.success,
			te.task_name,
			te.agent
		FROM behavioral_sessions bs
		JOIN task_executions te ON bs.task_execution_id = te.id
	`

	if project != "" {
		query += " WHERE te.plan_file LIKE ?"
		query = query + " ORDER BY bs.session_start DESC"
	} else {
		query += " ORDER BY bs.session_start DESC"
	}

	var rows *sql.Rows
	var err error
	if project != "" {
		rows, err = store.QueryRows(query, "%"+project+"%")
	} else {
		rows, err = store.QueryRows(query)
	}
	if err != nil {
		return nil, fmt.Errorf("query sessions: %w", err)
	}
	defer rows.Close()

	sessionMap := make(map[int]*behavioral.BehavioralMetrics)

	for rows.Next() {
		var sessionID int
		var duration int64
		var toolCalls, bashCommands, fileOps int
		var success bool
		var taskName, agent string

		if err := rows.Scan(&sessionID, &duration, &toolCalls, &bashCommands, &fileOps, &success, &taskName, &agent); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}

		// Create or update metrics for this session
		if _, exists := sessionMap[sessionID]; !exists {
			sessionMap[sessionID] = &behavioral.BehavioralMetrics{
				TotalSessions:    1,
				AgentPerformance: make(map[string]int),
			}
		}

		m := sessionMap[sessionID]
		m.AverageDuration = time.Duration(duration) * time.Second

		if success {
			m.SuccessRate = 1.0
			if agent != "" {
				m.AgentPerformance[agent]++
			}
		} else {
			m.ErrorRate = 1.0
			m.TotalErrors++
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rows: %w", err)
	}

	// Convert map to slice
	metrics := make([]behavioral.BehavioralMetrics, 0, len(sessionMap))
	for _, m := range sessionMap {
		metrics = append(metrics, *m)
	}

	return metrics, nil
}

// formatStatsTable formats statistics as a readable table
func formatStatsTable(stats *behavioral.AggregateStats) string {
	var sb strings.Builder

	sb.WriteString("\n=== SUMMARY STATISTICS ===\n\n")

	// Overall metrics
	sb.WriteString(fmt.Sprintf("Total Sessions:      %d\n", stats.TotalSessions))
	sb.WriteString(fmt.Sprintf("Total Agents:        %d\n", stats.TotalAgents))
	sb.WriteString(fmt.Sprintf("Total Operations:    %d\n", stats.TotalOperations))
	sb.WriteString(fmt.Sprintf("Success Rate:        %.1f%%\n", stats.SuccessRate*100))
	sb.WriteString(fmt.Sprintf("Error Rate:          %.1f%%\n", stats.ErrorRate*100))
	sb.WriteString(fmt.Sprintf("Average Duration:    %s\n", formatDuration(stats.AverageDuration)))
	sb.WriteString(fmt.Sprintf("Total Cost:          $%.4f\n", stats.TotalCost))

	// Token usage
	sb.WriteString(fmt.Sprintf("\nTotal Input Tokens:  %d\n", stats.TotalInputTokens))
	sb.WriteString(fmt.Sprintf("Total Output Tokens: %d\n", stats.TotalOutputTokens))
	sb.WriteString(fmt.Sprintf("Total Tokens:        %d\n", stats.TotalInputTokens+stats.TotalOutputTokens))

	// Top tools
	if len(stats.TopTools) > 0 {
		sb.WriteString("\n--- Top Tools (by usage) ---\n")
		topTools := stats.GetTopTools(10)
		for i, tool := range topTools {
			sb.WriteString(fmt.Sprintf("%2d. %-20s Count: %4d  Success: %5.1f%%  Errors: %5.1f%%\n",
				i+1, tool.Name, tool.Count, tool.SuccessRate*100, tool.ErrorRate*100))
		}
	}

	// Agent breakdown
	if len(stats.AgentBreakdown) > 0 {
		sb.WriteString("\n--- Agent Performance ---\n")

		// Sort agents by success count
		type agentStat struct {
			name  string
			count int
		}
		agents := make([]agentStat, 0, len(stats.AgentBreakdown))
		for name, count := range stats.AgentBreakdown {
			agents = append(agents, agentStat{name, count})
		}
		// Simple bubble sort for small lists
		for i := 0; i < len(agents)-1; i++ {
			for j := 0; j < len(agents)-i-1; j++ {
				if agents[j].count < agents[j+1].count {
					agents[j], agents[j+1] = agents[j+1], agents[j]
				}
			}
		}

		for i, agent := range agents {
			sb.WriteString(fmt.Sprintf("%2d. %-30s Success Count: %d\n", i+1, agent.name, agent.count))
		}
	}

	return sb.String()
}

// StreamActivity streams real-time activity from the behavioral database
func StreamActivity(ctx context.Context, project string) error {
	// Load config to get DB path
	cfg, err := config.LoadConfigFromDir(".")
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Check if streaming is enabled in config
	// For now, we'll assume it's gated by a future flag
	fmt.Println("Real-time streaming mode")
	fmt.Println("Watching for new sessions... (Press Ctrl+C to stop)")
	fmt.Println()

	// Open learning store
	store, err := learning.NewStore(cfg.Learning.DBPath)
	if err != nil {
		return fmt.Errorf("open learning store: %w", err)
	}
	defer store.Close()

	// Track last seen session ID
	lastSeenID := 0

	// Poll for new sessions
	ticker := time.NewTicker(2 * time.Second)
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

// SessionUpdate represents a new session detected
type SessionUpdate struct {
	ID        int
	TaskName  string
	Agent     string
	Timestamp time.Time
	Success   bool
	Duration  time.Duration
}

// watchNewSessions queries for sessions added since lastSeenID
func watchNewSessions(store *learning.Store, project string, lastSeenID int) ([]SessionUpdate, error) {
	query := `
		SELECT
			bs.id,
			te.task_name,
			te.agent,
			bs.session_start,
			te.success,
			bs.total_duration_seconds
		FROM behavioral_sessions bs
		JOIN task_executions te ON bs.task_execution_id = te.id
		WHERE bs.id > ?
	`

	args := []interface{}{lastSeenID}

	if project != "" {
		query += " AND te.plan_file LIKE ?"
		args = append(args, "%"+project+"%")
	}

	query += " ORDER BY bs.id ASC LIMIT 10"

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

		if err := rows.Scan(&update.ID, &update.TaskName, &agent, &update.Timestamp, &update.Success, &durationSecs); err != nil {
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

// displaySessionUpdate displays a single session update
func displaySessionUpdate(update SessionUpdate) {
	status := "SUCCESS"
	if !update.Success {
		status = "FAILED"
	}

	agent := update.Agent
	if agent == "" {
		agent = "default"
	}

	fmt.Printf("[%s] %s | Task: %s | Agent: %s | Duration: %s\n",
		update.Timestamp.Format("15:04:05"),
		status,
		update.TaskName,
		agent,
		formatDuration(update.Duration))
}
