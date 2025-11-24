package behavioral

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Exporter defines the interface for exporting behavioral data
type Exporter interface {
	Export(metrics *BehavioralMetrics) (string, error)
}

// JSONExporter exports behavioral data in JSON format
type JSONExporter struct {
	Pretty bool // Enable pretty printing with indentation
}

// Export converts BehavioralMetrics to JSON string
func (je *JSONExporter) Export(metrics *BehavioralMetrics) (string, error) {
	if metrics == nil {
		return "", fmt.Errorf("metrics cannot be nil")
	}

	if err := metrics.Validate(); err != nil {
		return "", fmt.Errorf("invalid metrics: %w", err)
	}

	var data []byte
	var err error

	if je.Pretty {
		data, err = json.MarshalIndent(metrics, "", "  ")
	} else {
		data, err = json.Marshal(metrics)
	}

	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return string(data), nil
}

// MarkdownExporter exports behavioral data in Markdown format
type MarkdownExporter struct {
	IncludeTimestamp bool // Include export timestamp in header
}

// Export converts BehavioralMetrics to Markdown string
func (me *MarkdownExporter) Export(metrics *BehavioralMetrics) (string, error) {
	if metrics == nil {
		return "", fmt.Errorf("metrics cannot be nil")
	}

	if err := metrics.Validate(); err != nil {
		return "", fmt.Errorf("invalid metrics: %w", err)
	}

	var sb strings.Builder

	// Header
	sb.WriteString("# Behavioral Metrics Report\n\n")
	if me.IncludeTimestamp {
		sb.WriteString(fmt.Sprintf("**Generated**: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))
	}

	// Summary section
	sb.WriteString("## Summary\n\n")
	sb.WriteString(fmt.Sprintf("- **Total Sessions**: %d\n", metrics.TotalSessions))
	sb.WriteString(fmt.Sprintf("- **Success Rate**: %.2f%%\n", metrics.SuccessRate*100))
	sb.WriteString(fmt.Sprintf("- **Error Rate**: %.2f%%\n", metrics.ErrorRate*100))
	sb.WriteString(fmt.Sprintf("- **Total Errors**: %d\n", metrics.TotalErrors))
	sb.WriteString(fmt.Sprintf("- **Average Duration**: %s\n", metrics.AverageDuration))
	sb.WriteString(fmt.Sprintf("- **Total Cost**: $%.4f\n", metrics.TotalCost))
	sb.WriteString("\n")

	// Token usage section
	sb.WriteString("## Token Usage\n\n")
	sb.WriteString(fmt.Sprintf("- **Input Tokens**: %d\n", metrics.TokenUsage.InputTokens))
	sb.WriteString(fmt.Sprintf("- **Output Tokens**: %d\n", metrics.TokenUsage.OutputTokens))
	sb.WriteString(fmt.Sprintf("- **Total Tokens**: %d\n", metrics.TokenUsage.TotalTokens()))
	sb.WriteString(fmt.Sprintf("- **Model**: %s\n", metrics.TokenUsage.ModelName))
	sb.WriteString(fmt.Sprintf("- **Cost**: $%.4f\n", metrics.TokenUsage.CostUSD))
	sb.WriteString("\n")

	// Agent performance section
	if len(metrics.AgentPerformance) > 0 {
		sb.WriteString("## Agent Performance\n\n")
		sb.WriteString("| Agent Name | Success Count |\n")
		sb.WriteString("|------------|---------------|\n")
		for agent, count := range metrics.AgentPerformance {
			sb.WriteString(fmt.Sprintf("| %s | %d |\n", agent, count))
		}
		sb.WriteString("\n")
	}

	// Tool executions section
	if len(metrics.ToolExecutions) > 0 {
		sb.WriteString("## Tool Executions\n\n")
		sb.WriteString("| Tool Name | Count | Success Rate | Error Rate | Avg Duration |\n")
		sb.WriteString("|-----------|-------|--------------|------------|-------------|\n")
		for _, tool := range metrics.ToolExecutions {
			sb.WriteString(fmt.Sprintf("| %s | %d | %.2f%% | %.2f%% | %s |\n",
				tool.Name,
				tool.Count,
				tool.SuccessRate*100,
				tool.ErrorRate*100,
				tool.AvgDuration))
		}
		sb.WriteString("\n")
	}

	// Bash commands section
	if len(metrics.BashCommands) > 0 {
		sb.WriteString("## Bash Commands\n\n")
		sb.WriteString("| Command | Exit Code | Success | Duration | Output Size |\n")
		sb.WriteString("|---------|-----------|---------|----------|-------------|\n")
		for _, cmd := range metrics.BashCommands {
			successStr := "Yes"
			if !cmd.Success {
				successStr = "No"
			}
			sb.WriteString(fmt.Sprintf("| `%s` | %d | %s | %s | %d bytes |\n",
				truncateString(cmd.Command, 50),
				cmd.ExitCode,
				successStr,
				cmd.Duration,
				cmd.OutputLength))
		}
		sb.WriteString("\n")
	}

	// File operations section
	if len(metrics.FileOperations) > 0 {
		sb.WriteString("## File Operations\n\n")
		sb.WriteString("| Type | Path | Success | Size | Duration |\n")
		sb.WriteString("|------|------|---------|------|----------|\n")
		for _, file := range metrics.FileOperations {
			successStr := "Yes"
			if !file.Success {
				successStr = "No"
			}
			sb.WriteString(fmt.Sprintf("| %s | `%s` | %s | %d bytes | %dms |\n",
				file.Type,
				truncateString(file.Path, 40),
				successStr,
				file.SizeBytes,
				file.Duration))
		}
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// truncateString truncates a string to maxLen characters, adding "..." if truncated
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// CSVExporter exports behavioral data in CSV format
type CSVExporter struct{}

// Export converts BehavioralMetrics to CSV string
func (ce *CSVExporter) Export(metrics *BehavioralMetrics) (string, error) {
	if metrics == nil {
		return "", fmt.Errorf("metrics cannot be nil")
	}

	if err := metrics.Validate(); err != nil {
		return "", fmt.Errorf("invalid metrics: %w", err)
	}

	var sb strings.Builder

	// Header
	sb.WriteString("Type,Name,Value\n")

	// Summary metrics
	sb.WriteString(fmt.Sprintf("Summary,TotalSessions,%d\n", metrics.TotalSessions))
	sb.WriteString(fmt.Sprintf("Summary,SuccessRate,%.4f\n", metrics.SuccessRate))
	sb.WriteString(fmt.Sprintf("Summary,ErrorRate,%.4f\n", metrics.ErrorRate))
	sb.WriteString(fmt.Sprintf("Summary,TotalErrors,%d\n", metrics.TotalErrors))
	sb.WriteString(fmt.Sprintf("Summary,AverageDuration,%s\n", metrics.AverageDuration))
	sb.WriteString(fmt.Sprintf("Summary,TotalCost,%.4f\n", metrics.TotalCost))

	// Token usage
	sb.WriteString(fmt.Sprintf("TokenUsage,InputTokens,%d\n", metrics.TokenUsage.InputTokens))
	sb.WriteString(fmt.Sprintf("TokenUsage,OutputTokens,%d\n", metrics.TokenUsage.OutputTokens))
	sb.WriteString(fmt.Sprintf("TokenUsage,TotalTokens,%d\n", metrics.TokenUsage.TotalTokens()))
	sb.WriteString(fmt.Sprintf("TokenUsage,ModelName,%s\n", metrics.TokenUsage.ModelName))
	sb.WriteString(fmt.Sprintf("TokenUsage,CostUSD,%.4f\n", metrics.TokenUsage.CostUSD))

	// Agent performance
	for agent, count := range metrics.AgentPerformance {
		sb.WriteString(fmt.Sprintf("AgentPerformance,%s,%d\n", escapeCSV(agent), count))
	}

	// Tool executions
	for _, tool := range metrics.ToolExecutions {
		sb.WriteString(fmt.Sprintf("ToolExecution,%s,Count:%d|SuccessRate:%.4f|ErrorRate:%.4f|AvgDuration:%s\n",
			escapeCSV(tool.Name), tool.Count, tool.SuccessRate, tool.ErrorRate, tool.AvgDuration))
	}

	// Bash commands
	for _, cmd := range metrics.BashCommands {
		sb.WriteString(fmt.Sprintf("BashCommand,%s,ExitCode:%d|Success:%t|Duration:%s|OutputLength:%d\n",
			escapeCSV(cmd.Command), cmd.ExitCode, cmd.Success, cmd.Duration, cmd.OutputLength))
	}

	// File operations
	for _, file := range metrics.FileOperations {
		sb.WriteString(fmt.Sprintf("FileOperation,%s,Type:%s|Success:%t|SizeBytes:%d|Duration:%d\n",
			escapeCSV(file.Path), file.Type, file.Success, file.SizeBytes, file.Duration))
	}

	return sb.String(), nil
}

// escapeCSV escapes special characters in CSV fields
func escapeCSV(s string) string {
	if strings.ContainsAny(s, ",\"\n\r") {
		s = strings.ReplaceAll(s, "\"", "\"\"")
		return "\"" + s + "\""
	}
	return s
}

// ExportToFile exports metrics to a file in the specified format
// Supports format values: "json", "markdown", "md", "csv"
func ExportToFile(metrics *BehavioralMetrics, path string, format string) error {
	if metrics == nil {
		return fmt.Errorf("metrics cannot be nil")
	}

	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Normalize format
	format = strings.ToLower(format)
	if format == "md" {
		format = "markdown"
	}

	// Select appropriate exporter
	var exporter Exporter
	var content string
	var err error

	switch format {
	case "json":
		exporter = &JSONExporter{Pretty: true}
	case "markdown":
		exporter = &MarkdownExporter{IncludeTimestamp: true}
	case "csv":
		exporter = &CSVExporter{}
	default:
		return fmt.Errorf("unsupported format: %s (supported: json, markdown, csv)", format)
	}

	// Export to string
	content, err = exporter.Export(metrics)
	if err != nil {
		return fmt.Errorf("export failed: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write to temporary file first (atomic write pattern)
	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	// Rename temp file to target file (atomic on most filesystems)
	if err := os.Rename(tempPath, path); err != nil {
		// Clean up temp file on error
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}

// ExportToString exports metrics to a string in the specified format
func ExportToString(metrics *BehavioralMetrics, format string) (string, error) {
	if metrics == nil {
		return "", fmt.Errorf("metrics cannot be nil")
	}

	// Normalize format
	format = strings.ToLower(format)
	if format == "md" {
		format = "markdown"
	}

	// Select appropriate exporter
	var exporter Exporter

	switch format {
	case "json":
		exporter = &JSONExporter{Pretty: true}
	case "markdown":
		exporter = &MarkdownExporter{IncludeTimestamp: true}
	case "csv":
		exporter = &CSVExporter{}
	default:
		return "", fmt.Errorf("unsupported format: %s (supported: json, markdown, csv)", format)
	}

	return exporter.Export(metrics)
}
