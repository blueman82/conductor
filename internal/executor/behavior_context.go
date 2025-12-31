package executor

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/harrison/conductor/internal/agent"
	"github.com/harrison/conductor/internal/behavioral"
)

// FormatBehaviorContext formats behavioral metrics as human-readable context for QC prompts
func FormatBehaviorContext(metrics *behavioral.BehavioralMetrics) string {
	if metrics == nil {
		return ""
	}

	var sb strings.Builder

	sb.WriteString("<behavioral_analysis>\n")

	// Session summary
	sb.WriteString(fmt.Sprintf("<session_summary sessions=\"%d\" success_rate=\"%.1f%%\" avg_duration=\"%s\"/>\n",
		metrics.TotalSessions,
		metrics.SuccessRate*100,
		formatDuration(metrics.AverageDuration)))

	// Cost summary
	if costSummary := SummarizeCost(metrics); costSummary != "" {
		sb.WriteString(costSummary)
		sb.WriteString("\n")
	}

	// Tool usage summary
	if toolSummary := SummarizeToolUsage(metrics); toolSummary != "" {
		sb.WriteString(toolSummary)
		sb.WriteString("\n")
	}

	// Anomalies
	if anomalies := IdentifyAnomalies(metrics); len(anomalies) > 0 {
		sb.WriteString("<anomalies>\n")
		for _, anomaly := range anomalies {
			sb.WriteString(agent.XMLTag("item", anomaly))
			sb.WriteString("\n")
		}
		sb.WriteString("</anomalies>\n")
	}

	sb.WriteString("</behavioral_analysis>\n")
	return sb.String()
}

// IdentifyAnomalies flags unusual patterns in behavioral metrics
func IdentifyAnomalies(metrics *behavioral.BehavioralMetrics) []string {
	if metrics == nil {
		return nil
	}

	var anomalies []string

	// High error rate (>20%)
	if metrics.ErrorRate > 0.2 {
		anomalies = append(anomalies, fmt.Sprintf("High error rate: %.1f%% (threshold: 20%%)", metrics.ErrorRate*100))
	}

	// High total errors
	if metrics.TotalErrors > 5 {
		anomalies = append(anomalies, fmt.Sprintf("Elevated error count: %d errors", metrics.TotalErrors))
	}

	// Low success rate (<70%)
	if metrics.TotalSessions > 0 && metrics.SuccessRate < 0.7 {
		anomalies = append(anomalies, fmt.Sprintf("Low success rate: %.1f%% (threshold: 70%%)", metrics.SuccessRate*100))
	}

	// High cost (>$1)
	if metrics.TotalCost > 1.0 {
		anomalies = append(anomalies, fmt.Sprintf("High cost: $%.2f (threshold: $1.00)", metrics.TotalCost))
	}

	// Long duration (>5 minutes)
	if metrics.AverageDuration > 5*time.Minute {
		anomalies = append(anomalies, fmt.Sprintf("Long execution: %s (threshold: 5m)", formatDuration(metrics.AverageDuration)))
	}

	// Tool-specific anomalies
	for _, tool := range metrics.ToolExecutions {
		// High tool error rate (>30%)
		if tool.ErrorRate > 0.3 && tool.Count >= 3 {
			anomalies = append(anomalies, fmt.Sprintf("Tool '%s' high error rate: %.1f%% (%d/%d failed)",
				tool.Name, tool.ErrorRate*100, tool.TotalErrors, tool.Count))
		}
	}

	// Bash command failures
	bashFailures := 0
	for _, cmd := range metrics.BashCommands {
		if !cmd.Success {
			bashFailures++
		}
	}
	if bashFailures > 3 {
		anomalies = append(anomalies, fmt.Sprintf("Multiple bash failures: %d commands failed", bashFailures))
	}

	// File operation failures
	fileFailures := 0
	for _, op := range metrics.FileOperations {
		if !op.Success {
			fileFailures++
		}
	}
	if fileFailures > 2 {
		anomalies = append(anomalies, fmt.Sprintf("File operation issues: %d operations failed", fileFailures))
	}

	return anomalies
}

// SummarizeCost formats cost metrics as readable string
func SummarizeCost(metrics *behavioral.BehavioralMetrics) string {
	if metrics == nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("<cost_summary>\n")

	totalTokens := metrics.TokenUsage.TotalTokens()
	if totalTokens > 0 {
		sb.WriteString(fmt.Sprintf("<tokens input=\"%s\" output=\"%s\" total=\"%s\"/>\n",
			formatTokenCount(metrics.TokenUsage.InputTokens),
			formatTokenCount(metrics.TokenUsage.OutputTokens),
			formatTokenCount(totalTokens)))
	}

	if metrics.TokenUsage.CostUSD > 0 {
		sb.WriteString(fmt.Sprintf("<cost>$%.4f</cost>\n", metrics.TokenUsage.CostUSD))
	} else if metrics.TotalCost > 0 {
		sb.WriteString(fmt.Sprintf("<cost>$%.4f</cost>\n", metrics.TotalCost))
	}

	if metrics.TokenUsage.ModelName != "" {
		sb.WriteString(agent.XMLTag("model", metrics.TokenUsage.ModelName))
		sb.WriteString("\n")
	}

	sb.WriteString("</cost_summary>")
	return sb.String()
}

// SummarizeToolUsage formats tool execution metrics as readable string
func SummarizeToolUsage(metrics *behavioral.BehavioralMetrics) string {
	if metrics == nil || len(metrics.ToolExecutions) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("<tool_usage>\n")

	// Sort tools by usage count (descending)
	tools := make([]behavioral.ToolExecution, len(metrics.ToolExecutions))
	copy(tools, metrics.ToolExecutions)
	sort.Slice(tools, func(i, j int) bool {
		return tools[i].Count > tools[j].Count
	})

	// Show top tools (max 5)
	maxTools := 5
	if len(tools) < maxTools {
		maxTools = len(tools)
	}

	for i := 0; i < maxTools; i++ {
		tool := tools[i]
		status := "ok"
		if tool.Count > 0 {
			if tool.SuccessRate >= 0.9 {
				status = "ok"
			} else if tool.SuccessRate < 0.7 {
				status = "issues"
			}
		}
		sb.WriteString(fmt.Sprintf("<tool name=\"%s\" calls=\"%d\" success_rate=\"%.0f%%\" avg_duration=\"%s\" status=\"%s\"/>\n",
			tool.Name, tool.Count, tool.SuccessRate*100, formatDuration(tool.AvgDuration), status))
	}

	if len(tools) > maxTools {
		sb.WriteString(fmt.Sprintf("<more_tools count=\"%d\"/>\n", len(tools)-maxTools))
	}

	sb.WriteString("</tool_usage>")
	return sb.String()
}

// BuildBehaviorPromptSection creates a QC prompt section for behavioral context
func BuildBehaviorPromptSection(metrics *behavioral.BehavioralMetrics) string {
	if metrics == nil {
		return ""
	}

	context := FormatBehaviorContext(metrics)
	if context == "" {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(context)
	sb.WriteString("Consider these behavioral patterns when assessing task completion quality.\n")

	return sb.String()
}

// formatDuration formats duration as human-readable string
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}

// formatTokenCount formats token count with K/M suffixes
func formatTokenCount(tokens int64) string {
	if tokens < 1000 {
		return fmt.Sprintf("%d", tokens)
	}
	if tokens < 1000000 {
		return fmt.Sprintf("%.1fK", float64(tokens)/1000)
	}
	return fmt.Sprintf("%.2fM", float64(tokens)/1000000)
}
