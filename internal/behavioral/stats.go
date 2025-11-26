package behavioral

import (
	"sort"
	"time"
)

// AggregateStats contains summary statistics from behavioral metrics
type AggregateStats struct {
	TotalAgents       int               `json:"total_agents"`
	TotalOperations   int               `json:"total_operations"`
	TotalCost         float64           `json:"total_cost"`
	TotalSessions     int               `json:"total_sessions"`
	SuccessRate       float64           `json:"success_rate"`
	ErrorRate         float64           `json:"error_rate"`
	AverageDuration   time.Duration     `json:"average_duration"`
	TotalInputTokens  int64             `json:"total_input_tokens"`
	TotalOutputTokens int64             `json:"total_output_tokens"`
	TopTools          []ToolStatSummary `json:"top_tools"`
	AgentBreakdown    map[string]int    `json:"agent_breakdown"`
}

// ToolStatSummary contains summary statistics for a tool
type ToolStatSummary struct {
	Name        string  `json:"name"`
	Count       int     `json:"count"`
	SuccessRate float64 `json:"success_rate"`
	ErrorRate   float64 `json:"error_rate"`
}

// CalculateStats calculates aggregate statistics from a list of behavioral metrics
func CalculateStats(metrics []BehavioralMetrics) *AggregateStats {
	if len(metrics) == 0 {
		return &AggregateStats{
			AgentBreakdown: make(map[string]int),
			TopTools:       []ToolStatSummary{},
		}
	}

	stats := &AggregateStats{
		AgentBreakdown: make(map[string]int),
		TopTools:       []ToolStatSummary{},
	}

	toolMap := make(map[string]*ToolExecution)
	var totalDuration time.Duration
	var successCount int

	for _, m := range metrics {
		stats.TotalSessions += m.TotalSessions
		stats.TotalCost += m.CalculateTotalCost()
		stats.TotalInputTokens += m.TokenUsage.InputTokens
		stats.TotalOutputTokens += m.TokenUsage.OutputTokens
		totalDuration += m.AverageDuration * time.Duration(m.TotalSessions)

		// Count successful sessions
		successCount += int(m.SuccessRate * float64(m.TotalSessions))

		// Aggregate agent breakdown
		for agent, count := range m.AgentPerformance {
			stats.AgentBreakdown[agent] += count
		}

		// Aggregate tool usage
		for _, tool := range m.ToolExecutions {
			if existing, ok := toolMap[tool.Name]; ok {
				existing.Count += tool.Count
				existing.TotalSuccess += tool.TotalSuccess
				existing.TotalErrors += tool.TotalErrors
			} else {
				toolCopy := tool
				toolMap[tool.Name] = &toolCopy
			}
		}

		// Count total operations
		stats.TotalOperations += len(m.ToolExecutions) + len(m.BashCommands) + len(m.FileOperations)
	}

	// Calculate success and error rates
	if stats.TotalSessions > 0 {
		stats.SuccessRate = float64(successCount) / float64(stats.TotalSessions)
		stats.ErrorRate = 1.0 - stats.SuccessRate
		stats.AverageDuration = totalDuration / time.Duration(stats.TotalSessions)
	}

	// Count unique agents
	stats.TotalAgents = len(stats.AgentBreakdown)

	// Build top tools list
	stats.TopTools = buildToolSummaries(toolMap)

	return stats
}

// buildToolSummaries converts tool map to sorted summary list
func buildToolSummaries(toolMap map[string]*ToolExecution) []ToolStatSummary {
	summaries := make([]ToolStatSummary, 0, len(toolMap))

	for name, tool := range toolMap {
		summary := ToolStatSummary{
			Name:  name,
			Count: tool.Count,
		}

		if tool.Count > 0 {
			summary.SuccessRate = float64(tool.TotalSuccess) / float64(tool.Count)
			summary.ErrorRate = float64(tool.TotalErrors) / float64(tool.Count)
		}

		summaries = append(summaries, summary)
	}

	// Sort by count descending
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Count > summaries[j].Count
	})

	return summaries
}

// GetTopTools returns the top N tools by usage count
func (as *AggregateStats) GetTopTools(limit int) []ToolStatSummary {
	if limit <= 0 || limit > len(as.TopTools) {
		return as.TopTools
	}
	return as.TopTools[:limit]
}

// GetAgentTypeBreakdown returns a map of agent names to success counts
func (as *AggregateStats) GetAgentTypeBreakdown() map[string]int {
	return as.AgentBreakdown
}
