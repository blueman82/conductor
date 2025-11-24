package logger

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
)

// colorScheme defines consistent colors for different metric types.
// Green: success/positive metrics
// Red: failure/error metrics
// Yellow: warning/threshold metrics
// Cyan: labels and identifiers
type colorScheme struct {
	success *color.Color
	fail    *color.Color
	warn    *color.Color
	label   *color.Color
	value   *color.Color
}

// newColorScheme creates the standard color scheme for metrics.
func newColorScheme() *colorScheme {
	return &colorScheme{
		success: color.New(color.FgGreen),
		fail:    color.New(color.FgRed),
		warn:    color.New(color.FgYellow),
		label:   color.New(color.FgCyan),
		value:   color.New(color.FgWhite),
	}
}

// formatColorizedMetric formats a single metric with colorized label and value.
// Label is colored cyan, value is colored based on the metric type and value.
// Format: "label: value"
func formatColorizedMetric(label string, value interface{}, scheme *colorScheme) string {
	labelColored := scheme.label.Sprint(label)
	valueColored := scheme.value.Sprintf("%v", value)
	return fmt.Sprintf("%s: %s", labelColored, valueColored)
}

// formatColorizedBehavioralMetrics formats behavioral metrics with color coding.
// Returns empty string if no relevant behavioral data is available.
// Format: "tools: N, bash: N, files: N, cost: $X.XX, errors: N"
// Success metrics (tools, bash, files) are colored green/cyan.
// Error metrics are colored red.
// Warning metrics (high cost) are colored yellow.
// Colors are automatically disabled when output is not a TTY via fatih/color's built-in detection.
func formatColorizedBehavioralMetrics(metadata map[string]interface{}) string {
	if metadata == nil {
		return ""
	}

	scheme := newColorScheme()
	var parts []string

	// Extract tool count (success - green)
	if toolCount, ok := metadata["tool_count"].(int); ok && toolCount > 0 {
		labelColored := scheme.success.Sprint("tools")
		valueColored := scheme.value.Sprintf("%d", toolCount)
		parts = append(parts, fmt.Sprintf("%s: %s", labelColored, valueColored))
	} else if toolCountFloat, ok := metadata["tool_count"].(float64); ok && toolCountFloat > 0 {
		labelColored := scheme.success.Sprint("tools")
		valueColored := scheme.value.Sprintf("%d", int(toolCountFloat))
		parts = append(parts, fmt.Sprintf("%s: %s", labelColored, valueColored))
	}

	// Extract bash command count (cyan label)
	if bashCount, ok := metadata["bash_count"].(int); ok && bashCount > 0 {
		parts = append(parts, formatColorizedMetric("bash", bashCount, scheme))
	} else if bashCountFloat, ok := metadata["bash_count"].(float64); ok && bashCountFloat > 0 {
		parts = append(parts, formatColorizedMetric("bash", int(bashCountFloat), scheme))
	}

	// Extract file operation count (success - green)
	if fileOps, ok := metadata["file_operations"].(int); ok && fileOps > 0 {
		labelColored := scheme.success.Sprint("files")
		valueColored := scheme.value.Sprintf("%d", fileOps)
		parts = append(parts, fmt.Sprintf("%s: %s", labelColored, valueColored))
	} else if fileOpsFloat, ok := metadata["file_operations"].(float64); ok && fileOpsFloat > 0 {
		labelColored := scheme.success.Sprint("files")
		valueColored := scheme.value.Sprintf("%d", int(fileOpsFloat))
		parts = append(parts, fmt.Sprintf("%s: %s", labelColored, valueColored))
	}

	// Extract cost (warning if high, cyan if low)
	if cost, ok := metadata["cost"].(float64); ok && cost > 0 {
		costStr := fmt.Sprintf("$%.4f", cost)
		if cost > 0.10 { // High cost threshold
			labelColored := scheme.warn.Sprint("cost")
			valueColored := scheme.warn.Sprint(costStr)
			parts = append(parts, fmt.Sprintf("%s: %s", labelColored, valueColored))
		} else {
			parts = append(parts, formatColorizedMetric("cost", costStr, scheme))
		}
	}

	// Extract error count (failures - red)
	if errorCount, ok := metadata["error_count"].(int); ok && errorCount > 0 {
		labelColored := scheme.fail.Sprint("errors")
		valueColored := scheme.fail.Sprintf("%d", errorCount)
		parts = append(parts, fmt.Sprintf("%s: %s", labelColored, valueColored))
	} else if errorCountFloat, ok := metadata["error_count"].(float64); ok && errorCountFloat > 0 {
		labelColored := scheme.fail.Sprint("errors")
		valueColored := scheme.fail.Sprintf("%d", int(errorCountFloat))
		parts = append(parts, fmt.Sprintf("%s: %s", labelColored, valueColored))
	}

	// Extract failed command count (failures - red)
	if failedCmds, ok := metadata["failed_commands"].(int); ok && failedCmds > 0 {
		labelColored := scheme.fail.Sprint("failed")
		valueColored := scheme.fail.Sprintf("%d", failedCmds)
		parts = append(parts, fmt.Sprintf("%s: %s", labelColored, valueColored))
	} else if failedCmdsFloat, ok := metadata["failed_commands"].(float64); ok && failedCmdsFloat > 0 {
		labelColored := scheme.fail.Sprint("failed")
		valueColored := scheme.fail.Sprintf("%d", int(failedCmdsFloat))
		parts = append(parts, fmt.Sprintf("%s: %s", labelColored, valueColored))
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, ", ")
}
