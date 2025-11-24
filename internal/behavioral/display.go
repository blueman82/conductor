// Package behavioral provides display and pagination utilities for behavioral data
package behavioral

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
)

// Paginator manages pagination state for displaying items
type Paginator struct {
	items       []interface{}
	pageSize    int
	currentPage int
	totalPages  int
}

// NewPaginator creates a paginator with the given items and page size
func NewPaginator(items []interface{}, pageSize int) *Paginator {
	if pageSize <= 0 {
		pageSize = 50
	}

	totalPages := len(items) / pageSize
	if len(items)%pageSize != 0 {
		totalPages++
	}

	return &Paginator{
		items:       items,
		pageSize:    pageSize,
		currentPage: 1,
		totalPages:  totalPages,
	}
}

// NextPage advances to the next page if available
func (p *Paginator) NextPage() bool {
	if p.currentPage < p.totalPages {
		p.currentPage++
		return true
	}
	return false
}

// PrevPage goes back to the previous page if available
func (p *Paginator) PrevPage() bool {
	if p.currentPage > 1 {
		p.currentPage--
		return true
	}
	return false
}

// GetCurrentPage returns the items for the current page
func (p *Paginator) GetCurrentPage() []interface{} {
	if len(p.items) == 0 {
		return []interface{}{}
	}

	start := (p.currentPage - 1) * p.pageSize
	end := start + p.pageSize

	if start >= len(p.items) {
		return []interface{}{}
	}

	if end > len(p.items) {
		end = len(p.items)
	}

	return p.items[start:end]
}

// GetCurrentPageNum returns the current page number
func (p *Paginator) GetCurrentPageNum() int {
	return p.currentPage
}

// GetTotalPages returns the total number of pages
func (p *Paginator) GetTotalPages() int {
	return p.totalPages
}

// HasNextPage returns true if there is a next page
func (p *Paginator) HasNextPage() bool {
	return p.currentPage < p.totalPages
}

// HasPrevPage returns true if there is a previous page
func (p *Paginator) HasPrevPage() bool {
	return p.currentPage > 1
}

// FormatTable formats behavioral metrics as a table
func FormatTable(items interface{}, colorOutput bool) []string {
	var rows []string

	// Handle []interface{} slice and convert to concrete types
	if itemSlice, ok := items.([]interface{}); ok {
		if len(itemSlice) == 0 {
			return []string{"No items found"}
		}

		// Detect type from first item
		switch itemSlice[0].(type) {
		case ToolExecution:
			tools := make([]ToolExecution, len(itemSlice))
			for i, item := range itemSlice {
				tools[i] = item.(ToolExecution)
			}
			return formatToolExecutions(tools, colorOutput)
		case BashCommand:
			cmds := make([]BashCommand, len(itemSlice))
			for i, item := range itemSlice {
				cmds[i] = item.(BashCommand)
			}
			return formatBashCommands(cmds, colorOutput)
		case FileOperation:
			ops := make([]FileOperation, len(itemSlice))
			for i, item := range itemSlice {
				ops[i] = item.(FileOperation)
			}
			return formatFileOperations(ops, colorOutput)
		case Session:
			sessions := make([]Session, len(itemSlice))
			for i, item := range itemSlice {
				sessions[i] = item.(Session)
			}
			return formatSessions(sessions, colorOutput)
		default:
			return []string{"Unsupported metric type"}
		}
	}

	// Handle concrete types
	switch m := items.(type) {
	case []ToolExecution:
		rows = formatToolExecutions(m, colorOutput)
	case []BashCommand:
		rows = formatBashCommands(m, colorOutput)
	case []FileOperation:
		rows = formatFileOperations(m, colorOutput)
	case []Session:
		rows = formatSessions(m, colorOutput)
	default:
		rows = []string{"Unsupported metric type"}
	}

	return rows
}

// formatToolExecutions formats tool execution metrics as table rows
func formatToolExecutions(tools []ToolExecution, colorOutput bool) []string {
	if len(tools) == 0 {
		return []string{"No tool executions found"}
	}

	// Calculate column widths
	widths := map[string]int{
		"name":     4, // "Name"
		"count":    5, // "Count"
		"success":  7, // "Success"
		"errors":   6, // "Errors"
		"duration": 8, // "Avg Duration"
	}

	for _, tool := range tools {
		if len(tool.Name) > widths["name"] {
			widths["name"] = len(tool.Name)
		}
	}

	// Build header
	header := fmt.Sprintf("%-*s  %-*s  %-*s  %-*s  %-*s",
		widths["name"], "Name",
		widths["count"], "Count",
		widths["success"], "Success",
		widths["errors"], "Errors",
		widths["duration"], "Avg Duration")

	rows := []string{header, strings.Repeat("-", len(header))}

	// Build rows
	for _, tool := range tools {
		successRate := fmt.Sprintf("%.1f%%", tool.SuccessRate*100)
		avgDuration := formatDurationShort(tool.AvgDuration)

		row := fmt.Sprintf("%-*s  %-*d  %-*s  %-*d  %-*s",
			widths["name"], tool.Name,
			widths["count"], tool.Count,
			widths["success"], successRate,
			widths["errors"], tool.TotalErrors,
			widths["duration"], avgDuration)

		if colorOutput {
			// Color based on success rate
			if tool.SuccessRate >= 0.9 {
				row = color.GreenString(row)
			} else if tool.SuccessRate < 0.5 {
				row = color.RedString(row)
			} else {
				row = color.YellowString(row)
			}
		}

		rows = append(rows, row)
	}

	return rows
}

// formatBashCommands formats bash command metrics as table rows
func formatBashCommands(commands []BashCommand, colorOutput bool) []string {
	if len(commands) == 0 {
		return []string{"No bash commands found"}
	}

	// Calculate column widths
	widths := map[string]int{
		"command":  40,
		"exitcode": 8,
		"duration": 8,
		"output":   10,
	}

	// Build header
	header := fmt.Sprintf("%-*s  %-*s  %-*s  %-*s",
		widths["command"], "Command",
		widths["exitcode"], "ExitCode",
		widths["duration"], "Duration",
		widths["output"], "Output")

	rows := []string{header, strings.Repeat("-", len(header))}

	// Build rows
	for _, cmd := range commands {
		// Truncate command if too long
		cmdStr := cmd.Command
		if len(cmdStr) > widths["command"] {
			cmdStr = cmdStr[:widths["command"]-3] + "..."
		}

		duration := formatDurationShort(cmd.Duration)
		outputSize := fmt.Sprintf("%d bytes", cmd.OutputLength)

		row := fmt.Sprintf("%-*s  %-*d  %-*s  %-*s",
			widths["command"], cmdStr,
			widths["exitcode"], cmd.ExitCode,
			widths["duration"], duration,
			widths["output"], outputSize)

		if colorOutput {
			// Color based on success
			if cmd.Success {
				row = color.GreenString(row)
			} else {
				row = color.RedString(row)
			}
		}

		rows = append(rows, row)
	}

	return rows
}

// formatFileOperations formats file operation metrics as table rows
func formatFileOperations(ops []FileOperation, colorOutput bool) []string {
	if len(ops) == 0 {
		return []string{"No file operations found"}
	}

	// Calculate column widths
	widths := map[string]int{
		"type":     6,
		"path":     40,
		"size":     10,
		"duration": 8,
	}

	// Build header
	header := fmt.Sprintf("%-*s  %-*s  %-*s  %-*s",
		widths["type"], "Type",
		widths["path"], "Path",
		widths["size"], "Size",
		widths["duration"], "Duration")

	rows := []string{header, strings.Repeat("-", len(header))}

	// Build rows
	for _, op := range ops {
		// Truncate path if too long
		pathStr := op.Path
		if len(pathStr) > widths["path"] {
			pathStr = "..." + pathStr[len(pathStr)-widths["path"]+3:]
		}

		sizeStr := formatBytes(op.SizeBytes)
		duration := formatDurationShort(op.GetDuration())

		row := fmt.Sprintf("%-*s  %-*s  %-*s  %-*s",
			widths["type"], op.Type,
			widths["path"], pathStr,
			widths["size"], sizeStr,
			widths["duration"], duration)

		if colorOutput {
			// Color based on success
			if op.Success {
				row = color.GreenString(row)
			} else {
				row = color.RedString(row)
			}
		}

		rows = append(rows, row)
	}

	return rows
}

// formatSessions formats session metrics as table rows
func formatSessions(sessions []Session, colorOutput bool) []string {
	if len(sessions) == 0 {
		return []string{"No sessions found"}
	}

	// Calculate column widths
	widths := map[string]int{
		"id":       12,
		"project":  20,
		"status":   10,
		"duration": 10,
		"errors":   6,
	}

	// Build header
	header := fmt.Sprintf("%-*s  %-*s  %-*s  %-*s  %-*s",
		widths["id"], "Session ID",
		widths["project"], "Project",
		widths["status"], "Status",
		widths["duration"], "Duration",
		widths["errors"], "Errors")

	rows := []string{header, strings.Repeat("-", len(header))}

	// Build rows
	for _, session := range sessions {
		// Truncate session ID
		idStr := session.ID
		if len(idStr) > widths["id"] {
			idStr = idStr[:widths["id"]]
		}

		// Truncate project name
		projectStr := session.Project
		if len(projectStr) > widths["project"] {
			projectStr = projectStr[:widths["project"]-3] + "..."
		}

		duration := formatDurationShort(session.GetDuration())

		row := fmt.Sprintf("%-*s  %-*s  %-*s  %-*s  %-*d",
			widths["id"], idStr,
			widths["project"], projectStr,
			widths["status"], session.Status,
			widths["duration"], duration,
			widths["errors"], session.ErrorCount)

		if colorOutput {
			// Color based on success
			if session.Success {
				row = color.GreenString(row)
			} else {
				row = color.RedString(row)
			}
		}

		rows = append(rows, row)
	}

	return rows
}

// formatDurationShort formats a duration in short form
func formatDurationShort(d interface{}) string {
	// Handle time.Duration
	if dur, ok := d.(interface{ String() string }); ok {
		s := dur.String()
		// Simplify common patterns
		s = strings.TrimSuffix(s, "0s")
		s = strings.TrimSuffix(s, "0m")
		return s
	}
	return "0s"
}

// formatBytes formats bytes in human-readable form
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// PrintNavigationBar prints the pagination navigation bar
func PrintNavigationBar(currentPage, totalPages int, colorOutput bool) string {
	if totalPages <= 1 {
		return ""
	}

	var nav string
	if colorOutput {
		nav = color.CyanString(fmt.Sprintf("\nPage %d of %d", currentPage, totalPages))
	} else {
		nav = fmt.Sprintf("\nPage %d of %d", currentPage, totalPages)
	}

	// Add navigation hints
	var hints []string
	if currentPage > 1 {
		hints = append(hints, "← Previous")
	}
	if currentPage < totalPages {
		hints = append(hints, "Next →")
	}

	if len(hints) > 0 {
		var sb strings.Builder
		sb.WriteString(nav)
		sb.WriteString(" | ")
		for i, hint := range hints {
			if i > 0 {
				sb.WriteString(" | ")
			}
			sb.WriteString(hint)
		}
		nav = sb.String()
	}

	return nav
}
