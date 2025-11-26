package executor

import (
	"context"
	"fmt"
	"time"

	"github.com/harrison/conductor/internal/behavioral"
	"github.com/harrison/conductor/internal/learning"
)

// BehaviorCollector collects and processes behavioral metrics from agent sessions
type BehaviorCollector struct {
	store *learning.Store
}

// NewBehaviorCollector creates a new behavioral metrics collector
func NewBehaviorCollector(store *learning.Store, sessionDir string) *BehaviorCollector {
	// sessionDir parameter kept for backward compatibility but no longer used
	// Sessions are now looked up from the database by external session ID
	return &BehaviorCollector{
		store: store,
	}
}

// collectBehavioralMetrics is defined in task.go as a method on DefaultTaskExecutor
// to access private executor fields. This file contains only helper types.

// LoadSessionMetrics loads metrics for a session by its external Claude session ID.
// First checks the database (populated by ingestion daemon), then falls back to
// returning empty metrics if not found (session not yet ingested).
func (bc *BehaviorCollector) LoadSessionMetrics(sessionID string) (*behavioral.BehavioralMetrics, error) {
	ctx := context.Background()

	// Look up session in database by external session ID
	dbMetrics, err := bc.store.GetSessionByExternalID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("database lookup failed: %w", err)
	}

	if dbMetrics != nil {
		// Found in database - convert to BehavioralMetrics
		return bc.convertDBMetrics(dbMetrics), nil
	}

	// Session not in database yet - return empty metrics with note
	// This happens when ingestion daemon hasn't processed the session yet
	return &behavioral.BehavioralMetrics{
		TotalSessions:    0,
		ToolExecutions:   []behavioral.ToolExecution{},
		BashCommands:     []behavioral.BashCommand{},
		FileOperations:   []behavioral.FileOperation{},
		TokenUsage:       behavioral.TokenUsage{},
		AgentPerformance: make(map[string]int),
	}, nil
}

// convertDBMetrics converts database session metrics to BehavioralMetrics
func (bc *BehaviorCollector) convertDBMetrics(dbMetrics *learning.BehavioralSessionMetrics) *behavioral.BehavioralMetrics {
	metrics := &behavioral.BehavioralMetrics{
		TotalSessions:    1,
		AverageDuration:  time.Duration(dbMetrics.TotalDurationSecs) * time.Second,
		ToolExecutions:   []behavioral.ToolExecution{},
		BashCommands:     []behavioral.BashCommand{},
		FileOperations:   []behavioral.FileOperation{},
		TokenUsage:       behavioral.TokenUsage{},
		AgentPerformance: make(map[string]int),
	}

	// Set success rate based on task success if available
	if dbMetrics.Success {
		metrics.SuccessRate = 1.0
	}

	// Track agent performance
	if dbMetrics.Agent != "" {
		if dbMetrics.Success {
			metrics.AgentPerformance[dbMetrics.Agent] = 1
		} else {
			metrics.AgentPerformance[dbMetrics.Agent] = 0
		}
	}

	// Note: Detailed tool/bash/file data would require additional queries
	// For now we return aggregate counts from the session record

	return metrics
}

// AggregateMetrics processes session data and computes aggregated metrics
func (bc *BehaviorCollector) AggregateMetrics(sessionData *behavioral.SessionData) *behavioral.BehavioralMetrics {
	metrics := &behavioral.BehavioralMetrics{
		TotalSessions:    1,
		ToolExecutions:   []behavioral.ToolExecution{},
		BashCommands:     []behavioral.BashCommand{},
		FileOperations:   []behavioral.FileOperation{},
		TokenUsage:       behavioral.TokenUsage{},
		AgentPerformance: make(map[string]int),
	}

	// Track tool counts for aggregation
	toolStats := make(map[string]*toolStats)

	// Process events
	for _, event := range sessionData.Events {
		switch e := event.(type) {
		case *behavioral.ToolCallEvent:
			// Track tool execution
			stats := getOrCreateToolStats(toolStats, e.ToolName)
			stats.count++
			stats.totalDuration += e.Duration
			if e.Success {
				stats.successCount++
			} else {
				stats.errorCount++
			}

		case *behavioral.BashCommandEvent:
			// Track bash command
			metrics.BashCommands = append(metrics.BashCommands, behavioral.BashCommand{
				Command:      e.Command,
				ExitCode:     e.ExitCode,
				OutputLength: e.OutputLength,
				Duration:     time.Duration(e.Duration) * time.Millisecond,
				Success:      e.Success,
				Timestamp:    e.Timestamp,
			})

		case *behavioral.FileOperationEvent:
			// Track file operation
			metrics.FileOperations = append(metrics.FileOperations, behavioral.FileOperation{
				Type:      e.Operation,
				Path:      e.Path,
				SizeBytes: e.SizeBytes,
				Success:   e.Success,
				Timestamp: e.Timestamp,
				Duration:  e.Duration,
			})

		case *behavioral.TokenUsageEvent:
			// Accumulate token usage
			metrics.TokenUsage.InputTokens += e.InputTokens
			metrics.TokenUsage.OutputTokens += e.OutputTokens
			metrics.TokenUsage.CostUSD += e.CostUSD
			if e.ModelName != "" {
				metrics.TokenUsage.ModelName = e.ModelName
			}
		}
	}

	// Convert tool stats to ToolExecution records
	for name, stats := range toolStats {
		te := behavioral.ToolExecution{
			Name:         name,
			Count:        stats.count,
			TotalSuccess: stats.successCount,
			TotalErrors:  stats.errorCount,
			AvgDuration:  time.Duration(stats.totalDuration/int64(stats.count)) * time.Millisecond,
		}
		te.CalculateRates()
		metrics.ToolExecutions = append(metrics.ToolExecutions, te)
	}

	// Aggregate session-level metrics
	if sessionData.Session.Success {
		metrics.SuccessRate = 1.0
		if sessionData.Session.AgentName != "" {
			metrics.AgentPerformance[sessionData.Session.AgentName] = 1
		}
	}
	metrics.AverageDuration = time.Duration(sessionData.Session.Duration) * time.Millisecond
	metrics.TotalErrors = sessionData.Session.ErrorCount
	if sessionData.Session.ErrorCount > 0 {
		metrics.ErrorRate = float64(sessionData.Session.ErrorCount) / float64(len(sessionData.Events))
	}

	return metrics
}

// toolStats tracks metrics for a specific tool
type toolStats struct {
	count         int
	successCount  int
	errorCount    int
	totalDuration int64 // milliseconds
}

func getOrCreateToolStats(statsMap map[string]*toolStats, toolName string) *toolStats {
	if stats, exists := statsMap[toolName]; exists {
		return stats
	}
	stats := &toolStats{}
	statsMap[toolName] = stats
	return stats
}

// StoreSessionMetrics stores behavioral metrics in the database
func (bc *BehaviorCollector) StoreSessionMetrics(ctx context.Context, taskExecutionID int64, metrics *behavioral.BehavioralMetrics) error {
	// Build session data record
	sessionData := &learning.BehavioralSessionData{
		TaskExecutionID:     taskExecutionID,
		SessionStart:        time.Now().Add(-metrics.AverageDuration), // Approximate start time
		TotalDurationSecs:   int64(metrics.AverageDuration.Seconds()),
		TotalToolCalls:      len(metrics.ToolExecutions),
		TotalBashCommands:   len(metrics.BashCommands),
		TotalFileOperations: len(metrics.FileOperations),
		TotalTokensUsed:     metrics.TokenUsage.TotalTokens(),
		ContextWindowUsed:   0, // Not available from JSONL
	}

	// If session completed, set end time
	if metrics.AverageDuration > 0 {
		endTime := time.Now()
		sessionData.SessionEnd = &endTime
	}

	// Convert tool executions to database format
	var toolExecs []learning.ToolExecutionData
	for _, te := range metrics.ToolExecutions {
		toolExecs = append(toolExecs, learning.ToolExecutionData{
			ToolName:     te.Name,
			Parameters:   "{}", // Not available from aggregated data
			DurationMs:   int64(te.AvgDuration.Milliseconds()),
			Success:      te.SuccessRate > 0.5,
			ErrorMessage: "",
		})
	}

	// Convert bash commands to database format
	var bashCmds []learning.BashCommandData
	for _, bc := range metrics.BashCommands {
		bashCmds = append(bashCmds, learning.BashCommandData{
			Command:      bc.Command,
			DurationMs:   int64(bc.Duration.Milliseconds()),
			ExitCode:     bc.ExitCode,
			StdoutLength: bc.OutputLength,
			StderrLength: 0, // Not tracked separately
			Success:      bc.Success,
		})
	}

	// Convert file operations to database format
	var fileOps []learning.FileOperationData
	for _, fo := range metrics.FileOperations {
		fileOps = append(fileOps, learning.FileOperationData{
			OperationType: fo.Type,
			FilePath:      fo.Path,
			DurationMs:    fo.Duration,
			BytesAffected: fo.SizeBytes,
			Success:       fo.Success,
			ErrorMessage:  "",
		})
	}

	// Convert token usage to database format
	var tokenUsage []learning.TokenUsageData
	if metrics.TokenUsage.TotalTokens() > 0 {
		tokenUsage = append(tokenUsage, learning.TokenUsageData{
			InputTokens:       metrics.TokenUsage.InputTokens,
			OutputTokens:      metrics.TokenUsage.OutputTokens,
			TotalTokens:       metrics.TokenUsage.TotalTokens(),
			ContextWindowSize: 0, // Not available
		})
	}

	// Record in database
	_, err := bc.store.RecordSessionMetrics(ctx, sessionData, toolExecs, bashCmds, fileOps, tokenUsage)
	return err
}
