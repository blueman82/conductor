// Package behavioral provides transcript formatting for session events
package behavioral

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
)

// TranscriptOptions configures transcript formatting behavior
type TranscriptOptions struct {
	ColorOutput     bool // Enable ANSI color codes
	TruncateLength  int  // Max length for tool inputs (0 = no truncation)
	ShowTimestamps  bool // Include timestamps in output
	ShowToolResults bool // Include tool result content
	CompactMode     bool // Minimize whitespace
}

// DefaultTranscriptOptions returns sensible defaults
func DefaultTranscriptOptions() TranscriptOptions {
	return TranscriptOptions{
		ColorOutput:     true,
		TruncateLength:  0, // No truncation - show full content
		ShowTimestamps:  true,
		ShowToolResults: false,
		CompactMode:     false,
	}
}

// FormatTranscript formats a slice of events as a human-readable transcript
func FormatTranscript(events []Event, opts TranscriptOptions) string {
	if len(events) == 0 {
		return "No events to display"
	}

	var sb strings.Builder
	for i, event := range events {
		entry := FormatTranscriptEntry(event, opts)
		if entry == "" {
			continue
		}
		sb.WriteString(entry)
		if !opts.CompactMode && i < len(events)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// FormatTranscriptEntry formats a single event as a transcript line
func FormatTranscriptEntry(event Event, opts TranscriptOptions) string {
	if event == nil {
		return ""
	}

	var line string
	timestamp := ""
	if opts.ShowTimestamps {
		timestamp = formatTimestamp(event.GetTimestamp()) + " "
	}

	switch e := event.(type) {
	case *SessionStartEvent:
		line = formatSessionStartEvent(e, opts)
	case *TextEvent:
		line = formatTextEvent(e, timestamp, opts)
	case *ToolCallEvent:
		line = formatToolCallEvent(e, timestamp, opts)
	case *BashCommandEvent:
		line = formatBashCommandEvent(e, timestamp, opts)
	case *FileOperationEvent:
		line = formatFileOperationEvent(e, timestamp, opts)
	case *TokenUsageEvent:
		// Skip token usage in transcript (metadata)
		return ""
	default:
		return ""
	}

	return line
}

// formatSessionStartEvent formats new session announcements
func formatSessionStartEvent(e *SessionStartEvent, opts TranscriptOptions) string {
	var sb strings.Builder

	// Header line
	header := "━━━ New Agent Started ━━━"
	agentLine := fmt.Sprintf("Agent: %s (%s)", e.SessionID, e.AgentType)

	if opts.ColorOutput {
		sb.WriteString(color.YellowString(header) + "\n")
		sb.WriteString(color.GreenString("Agent: ") + color.CyanString(e.SessionID) + color.HiBlackString(" ("+e.AgentType+")") + "\n")
	} else {
		sb.WriteString(header + "\n")
		sb.WriteString(agentLine + "\n")
	}

	sb.WriteString("\n")
	return sb.String()
}

// formatTextEvent formats assistant/user text with emoji
func formatTextEvent(e *TextEvent, timestamp string, opts TranscriptOptions) string {
	if e.Text == "" {
		return ""
	}

	emoji := "💬"
	roleLabel := "Assistant"
	if e.Role == "user" {
		emoji = "👤"
		roleLabel = "User"
	}

	text := e.Text
	if opts.TruncateLength > 0 && len(text) > opts.TruncateLength {
		text = text[:opts.TruncateLength] + "..."
	}

	line := fmt.Sprintf("%s%s [%s]: %s\n", timestamp, emoji, roleLabel, text)

	if opts.ColorOutput {
		if e.Role == "assistant" {
			return color.CyanString(line)
		}
		return color.WhiteString(line)
	}
	return line
}

// formatToolCallEvent formats tool invocations
func formatToolCallEvent(e *ToolCallEvent, timestamp string, opts TranscriptOptions) string {
	// Skip tool_result events (they're just completion tracking)
	if e.Type == "tool_result" {
		return ""
	}

	emoji := "🔧"
	if !e.Success {
		emoji = "❌"
	}

	// Special handling for Edit to show diff format
	if e.ToolName == "Edit" {
		return formatEditDiff(e, timestamp, emoji, opts)
	}

	// Build parameter summary for other tools
	paramSummary := formatParameters(e.Parameters, opts.TruncateLength)

	var line string
	if paramSummary != "" {
		line = fmt.Sprintf("%s%s %s(%s)\n", timestamp, emoji, e.ToolName, paramSummary)
	} else {
		line = fmt.Sprintf("%s%s %s\n", timestamp, emoji, e.ToolName)
	}

	if opts.ColorOutput {
		if !e.Success {
			return color.RedString(line)
		}
		return color.New(color.FgGreen, color.Faint).Sprint(line)
	}
	return line
}

// formatEditDiff formats Edit tool calls with full diff output
func formatEditDiff(e *ToolCallEvent, timestamp, emoji string, opts TranscriptOptions) string {
	var sb strings.Builder

	// Get file path
	filePath := ""
	if fp, ok := e.Parameters["file_path"].(string); ok {
		filePath = fp
	}

	// Header line
	header := fmt.Sprintf("%s%s Edit: %s\n", timestamp, emoji, filePath)
	if opts.ColorOutput {
		sb.WriteString(color.CyanString(header))
	} else {
		sb.WriteString(header)
	}

	// Old string (removed lines)
	if oldStr, ok := e.Parameters["old_string"].(string); ok && oldStr != "" {
		lines := strings.Split(oldStr, "\n")
		for _, line := range lines {
			if opts.ColorOutput {
				sb.WriteString(color.RedString("   - " + line + "\n"))
			} else {
				sb.WriteString("   - " + line + "\n")
			}
		}
	}

	// New string (added lines)
	if newStr, ok := e.Parameters["new_string"].(string); ok && newStr != "" {
		lines := strings.Split(newStr, "\n")
		for _, line := range lines {
			if opts.ColorOutput {
				sb.WriteString(color.GreenString("   + " + line + "\n"))
			} else {
				sb.WriteString("   + " + line + "\n")
			}
		}
	}

	return sb.String()
}

// formatBashCommandEvent formats bash commands
func formatBashCommandEvent(e *BashCommandEvent, timestamp string, opts TranscriptOptions) string {
	emoji := "🔧"
	if !e.Success {
		emoji = "❌"
	}

	cmd := e.Command
	if opts.TruncateLength > 0 && len(cmd) > opts.TruncateLength {
		cmd = cmd[:opts.TruncateLength] + "..."
	}

	line := fmt.Sprintf("%s%s Bash: %s\n", timestamp, emoji, cmd)

	if opts.ColorOutput {
		if !e.Success {
			return color.RedString(line)
		}
		return color.New(color.FgGreen, color.Faint).Sprint(line)
	}
	return line
}

// formatFileOperationEvent formats file operations
func formatFileOperationEvent(e *FileOperationEvent, timestamp string, opts TranscriptOptions) string {
	emoji := "🔧"
	if !e.Success {
		emoji = "❌"
	}

	path := e.Path
	if opts.TruncateLength > 0 && len(path) > opts.TruncateLength {
		// Keep the end of the path (more useful)
		path = "..." + path[len(path)-opts.TruncateLength+3:]
	}

	line := fmt.Sprintf("%s%s %s: %s\n", timestamp, emoji, e.Operation, path)

	if opts.ColorOutput {
		if !e.Success {
			return color.RedString(line)
		}
		return color.New(color.FgGreen, color.Faint).Sprint(line)
	}
	return line
}

// formatTimestamp formats a timestamp for transcript display
func formatTimestamp(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("15:04:05")
}

// formatParameters creates a summary of tool parameters
func formatParameters(params map[string]interface{}, maxLen int) string {
	if len(params) == 0 {
		return ""
	}

	var parts []string
	for key, val := range params {
		valStr := fmt.Sprintf("%v", val)
		if maxLen > 0 && len(valStr) > maxLen/2 {
			valStr = valStr[:maxLen/2] + "..."
		}
		parts = append(parts, fmt.Sprintf("%s=%s", key, valStr))
	}

	result := strings.Join(parts, ", ")
	if maxLen > 0 && len(result) > maxLen {
		result = result[:maxLen] + "..."
	}
	return result
}
