package behavioral

import (
	"time"
)

// ExtractMetrics extracts behavioral metrics from parsed session data
// Processes all events and aggregates into actionable metrics
func ExtractMetrics(sessionData *SessionData) *BehavioralMetrics {
	if sessionData == nil {
		return &BehavioralMetrics{
			ToolExecutions: []ToolExecution{},
			BashCommands:   []BashCommand{},
			FileOperations: []FileOperation{},
		}
	}

	metrics := &BehavioralMetrics{
		TotalSessions:    1,
		ToolExecutions:   []ToolExecution{},
		BashCommands:     []BashCommand{},
		FileOperations:   []FileOperation{},
		AgentPerformance: make(map[string]int),
	}

	// Extract session-level metrics
	metrics.AverageDuration = sessionData.Session.GetDuration()
	if sessionData.Session.Success {
		metrics.SuccessRate = 1.0
		if sessionData.Session.AgentName != "" {
			metrics.AgentPerformance[sessionData.Session.AgentName] = 1
		}
	} else {
		metrics.SuccessRate = 0.0
	}
	metrics.TotalErrors = sessionData.Session.ErrorCount
	// ErrorRate is errors per session (can be > 1 if multiple errors per session)
	// For validation, we normalize it to 0-1 range based on average errors
	if metrics.TotalSessions > 0 {
		avgErrors := float64(metrics.TotalErrors) / float64(metrics.TotalSessions)
		// Normalize to 0-1 by capping at 1.0 (represents at least 1 error per session)
		if avgErrors > 1.0 {
			metrics.ErrorRate = 1.0
		} else {
			metrics.ErrorRate = avgErrors
		}
	}

	// Process events by type
	var toolCallEvents []*ToolCallEvent
	var bashCommandEvents []*BashCommandEvent
	var fileOperationEvents []*FileOperationEvent
	var tokenUsageEvents []*TokenUsageEvent

	for _, event := range sessionData.Events {
		switch e := event.(type) {
		case *ToolCallEvent:
			toolCallEvents = append(toolCallEvents, e)
		case *BashCommandEvent:
			bashCommandEvents = append(bashCommandEvents, e)
		case *FileOperationEvent:
			fileOperationEvents = append(fileOperationEvents, e)
		case *TokenUsageEvent:
			tokenUsageEvents = append(tokenUsageEvents, e)
		}
	}

	// Aggregate tool executions
	metrics.ToolExecutions = aggregateToolExecutions(toolCallEvents)

	// Aggregate bash commands
	metrics.BashCommands = aggregateBashCommands(bashCommandEvents)

	// Aggregate file operations
	metrics.FileOperations = aggregateFileOperations(fileOperationEvents)

	// Aggregate token usage
	metrics.TokenUsage = aggregateTokenUsage(tokenUsageEvents)

	// Calculate total cost
	metrics.TotalCost = metrics.CalculateTotalCost()

	return metrics
}

// aggregateToolExecutions groups tool calls by name and calculates metrics
func aggregateToolExecutions(events []*ToolCallEvent) []ToolExecution {
	toolMap := make(map[string]*ToolExecution)

	for _, event := range events {
		if event == nil {
			continue
		}

		tool, exists := toolMap[event.ToolName]
		if !exists {
			tool = &ToolExecution{
				Name:         event.ToolName,
				Count:        0,
				TotalSuccess: 0,
				TotalErrors:  0,
			}
			toolMap[event.ToolName] = tool
		}

		tool.Count++
		if event.Success {
			tool.TotalSuccess++
		} else {
			tool.TotalErrors++
		}

		// Update average duration
		if event.Duration > 0 {
			currentTotal := tool.AvgDuration * time.Duration(tool.Count-1)
			newDuration := time.Duration(event.Duration) * time.Millisecond
			tool.AvgDuration = (currentTotal + newDuration) / time.Duration(tool.Count)
		}
	}

	// Calculate rates and convert to slice
	result := make([]ToolExecution, 0, len(toolMap))
	for _, tool := range toolMap {
		tool.CalculateRates()
		result = append(result, *tool)
	}

	return result
}

// aggregateBashCommands converts bash command events to metrics
func aggregateBashCommands(events []*BashCommandEvent) []BashCommand {
	commands := make([]BashCommand, 0, len(events))

	for _, event := range events {
		if event == nil {
			continue
		}

		command := BashCommand{
			Command:      event.Command,
			ExitCode:     event.ExitCode,
			OutputLength: event.OutputLength,
			Duration:     time.Duration(event.Duration) * time.Millisecond,
			Success:      event.Success,
			Timestamp:    event.Timestamp,
		}

		commands = append(commands, command)
	}

	return commands
}

// aggregateFileOperations converts file operation events to metrics
func aggregateFileOperations(events []*FileOperationEvent) []FileOperation {
	operations := make([]FileOperation, 0, len(events))

	for _, event := range events {
		if event == nil {
			continue
		}

		operation := FileOperation{
			Type:      event.Operation,
			Path:      event.Path,
			SizeBytes: event.SizeBytes,
			Success:   event.Success,
			Timestamp: event.Timestamp,
			Duration:  event.Duration,
		}

		operations = append(operations, operation)
	}

	return operations
}

// aggregateTokenUsage sums up all token usage and calculates total cost
func aggregateTokenUsage(events []*TokenUsageEvent) TokenUsage {
	usage := TokenUsage{
		InputTokens:  0,
		OutputTokens: 0,
		CostUSD:      0,
		ModelName:    "",
	}

	totalCost := 0.0

	for _, event := range events {
		if event == nil {
			continue
		}

		usage.InputTokens += event.InputTokens
		usage.OutputTokens += event.OutputTokens
		totalCost += event.CostUSD

		// Use the most recent model name
		if event.ModelName != "" {
			usage.ModelName = event.ModelName
		}
	}

	// If cost was provided in events, use that; otherwise calculate
	if totalCost > 0 {
		usage.CostUSD = totalCost
	} else {
		usage.CalculateCost()
	}

	return usage
}

// calculateSuccessRate computes success rate from successes and total count
func calculateSuccessRate(successes, total int) float64 {
	if total == 0 {
		return 0.0
	}
	return float64(successes) / float64(total)
}

// calculateAverageDuration computes average duration from events
func calculateAverageDuration(durations []int64) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	var total int64
	for _, d := range durations {
		total += d
	}

	avgMs := total / int64(len(durations))
	return time.Duration(avgMs) * time.Millisecond
}

// calculateTokenCost calculates cost based on token count and model
// Uses standard Claude pricing: $3 per 1M input, $15 per 1M output
func calculateTokenCost(inputTokens, outputTokens int64, model string) float64 {
	const (
		inputCostPerMillion  = 3.0
		outputCostPerMillion = 15.0
	)

	inputCost := (float64(inputTokens) / 1_000_000) * inputCostPerMillion
	outputCost := (float64(outputTokens) / 1_000_000) * outputCostPerMillion

	return inputCost + outputCost
}
