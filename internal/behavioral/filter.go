package behavioral

import (
	"fmt"
	"strings"
	"time"
)

// FilterCriteria defines dimensions for filtering behavioral metrics
type FilterCriteria struct {
	Search     string    // Case-insensitive search across relevant fields
	EventType  string    // Filter by type: tool, bash, file
	ErrorsOnly bool      // Show only items with errors
	Since      time.Time // Start of time range (inclusive)
	Until      time.Time // End of time range (inclusive)
}

// Validate checks if the filter criteria are valid
func (fc *FilterCriteria) Validate() error {
	if !fc.Since.IsZero() && !fc.Until.IsZero() && fc.Since.After(fc.Until) {
		return fmt.Errorf("since time (%v) cannot be after until time (%v)", fc.Since, fc.Until)
	}

	if fc.EventType != "" {
		validTypes := map[string]bool{"tool": true, "bash": true, "file": true}
		if !validTypes[fc.EventType] {
			return fmt.Errorf("invalid event type '%s': must be one of: tool, bash, file", fc.EventType)
		}
	}

	return nil
}

// ApplyFiltersToSessions filters sessions based on criteria (AND logic)
func ApplyFiltersToSessions(sessions []Session, criteria FilterCriteria) []Session {
	if err := criteria.Validate(); err != nil {
		return sessions // Return unfiltered on invalid criteria
	}

	var filtered []Session
	for _, session := range sessions {
		if matchesSessionFilter(session, criteria) {
			filtered = append(filtered, session)
		}
	}
	return filtered
}

// ApplyFiltersToToolExecutions filters tool executions based on criteria
func ApplyFiltersToToolExecutions(tools []ToolExecution, criteria FilterCriteria) []ToolExecution {
	if err := criteria.Validate(); err != nil {
		return tools
	}

	// Check event type filter - only include if type is "tool" or empty
	if criteria.EventType != "" && criteria.EventType != "tool" {
		return []ToolExecution{}
	}

	var filtered []ToolExecution
	for _, tool := range tools {
		if matchesToolFilter(tool, criteria) {
			filtered = append(filtered, tool)
		}
	}
	return filtered
}

// ApplyFiltersToBashCommands filters bash commands based on criteria
func ApplyFiltersToBashCommands(commands []BashCommand, criteria FilterCriteria) []BashCommand {
	if err := criteria.Validate(); err != nil {
		return commands
	}

	// Check event type filter - only include if type is "bash" or empty
	if criteria.EventType != "" && criteria.EventType != "bash" {
		return []BashCommand{}
	}

	var filtered []BashCommand
	for _, cmd := range commands {
		if matchesBashFilter(cmd, criteria) {
			filtered = append(filtered, cmd)
		}
	}
	return filtered
}

// ApplyFiltersToFileOperations filters file operations based on criteria
func ApplyFiltersToFileOperations(files []FileOperation, criteria FilterCriteria) []FileOperation {
	if err := criteria.Validate(); err != nil {
		return files
	}

	// Check event type filter - only include if type is "file" or empty
	if criteria.EventType != "" && criteria.EventType != "file" {
		return []FileOperation{}
	}

	var filtered []FileOperation
	for _, file := range files {
		if matchesFileFilter(file, criteria) {
			filtered = append(filtered, file)
		}
	}
	return filtered
}

// matchesSessionFilter checks if a session matches all filter criteria (AND logic)
func matchesSessionFilter(session Session, criteria FilterCriteria) bool {
	// Search filter - search in project, agent name, and status
	if criteria.Search != "" {
		searchLower := strings.ToLower(criteria.Search)
		if !strings.Contains(strings.ToLower(session.Project), searchLower) &&
			!strings.Contains(strings.ToLower(session.AgentName), searchLower) &&
			!strings.Contains(strings.ToLower(session.Status), searchLower) &&
			!strings.Contains(strings.ToLower(session.ID), searchLower) {
			return false
		}
	}

	// Errors only filter
	if criteria.ErrorsOnly && session.ErrorCount == 0 {
		return false
	}

	// Time range filters
	if !criteria.Since.IsZero() && session.Timestamp.Before(criteria.Since) {
		return false
	}
	if !criteria.Until.IsZero() && session.Timestamp.After(criteria.Until) {
		return false
	}

	return true
}

// matchesToolFilter checks if a tool execution matches filter criteria
func matchesToolFilter(tool ToolExecution, criteria FilterCriteria) bool {
	// Search filter - search in tool name
	if criteria.Search != "" {
		searchLower := strings.ToLower(criteria.Search)
		if !strings.Contains(strings.ToLower(tool.Name), searchLower) {
			return false
		}
	}

	// Errors only filter
	if criteria.ErrorsOnly && tool.TotalErrors == 0 {
		return false
	}

	return true
}

// matchesBashFilter checks if a bash command matches filter criteria
func matchesBashFilter(cmd BashCommand, criteria FilterCriteria) bool {
	// Search filter - search in command string
	if criteria.Search != "" {
		searchLower := strings.ToLower(criteria.Search)
		if !strings.Contains(strings.ToLower(cmd.Command), searchLower) {
			return false
		}
	}

	// Errors only filter
	if criteria.ErrorsOnly && cmd.Success {
		return false
	}

	// Time range filters
	if !criteria.Since.IsZero() && cmd.Timestamp.Before(criteria.Since) {
		return false
	}
	if !criteria.Until.IsZero() && cmd.Timestamp.After(criteria.Until) {
		return false
	}

	return true
}

// matchesFileFilter checks if a file operation matches filter criteria
func matchesFileFilter(file FileOperation, criteria FilterCriteria) bool {
	// Search filter - search in file path and type
	if criteria.Search != "" {
		searchLower := strings.ToLower(criteria.Search)
		if !strings.Contains(strings.ToLower(file.Path), searchLower) &&
			!strings.Contains(strings.ToLower(file.Type), searchLower) {
			return false
		}
	}

	// Errors only filter
	if criteria.ErrorsOnly && file.Success {
		return false
	}

	// Time range filters
	if !criteria.Since.IsZero() && file.Timestamp.Before(criteria.Since) {
		return false
	}
	if !criteria.Until.IsZero() && file.Timestamp.After(criteria.Until) {
		return false
	}

	return true
}

// ParseTimeRange parses time range string into time.Time
// Supports formats:
// - Relative: "1h", "24h", "7d", "30d" (from now)
// - Keywords: "today", "yesterday"
// - ISO dates: "2025-01-15", "2025-01-15T14:30:00Z"
func ParseTimeRange(timeRange string) (time.Time, error) {
	if timeRange == "" {
		return time.Time{}, nil
	}

	now := time.Now()

	// Handle keywords
	switch strings.ToLower(timeRange) {
	case "today":
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()), nil
	case "yesterday":
		yesterday := now.AddDate(0, 0, -1)
		return time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, now.Location()), nil
	}

	// Handle relative durations (e.g., "1h", "24h", "7d")
	if len(timeRange) >= 2 {
		unit := timeRange[len(timeRange)-1]
		valueStr := timeRange[:len(timeRange)-1]

		var duration time.Duration
		switch unit {
		case 'h':
			// Parse hours
			var hours int
			if _, err := fmt.Sscanf(valueStr, "%d", &hours); err == nil {
				duration = time.Duration(hours) * time.Hour
				return now.Add(-duration), nil
			}
		case 'd':
			// Parse days
			var days int
			if _, err := fmt.Sscanf(valueStr, "%d", &days); err == nil {
				return now.AddDate(0, 0, -days), nil
			}
		case 'm':
			// Parse minutes
			var minutes int
			if _, err := fmt.Sscanf(valueStr, "%d", &minutes); err == nil {
				duration = time.Duration(minutes) * time.Minute
				return now.Add(-duration), nil
			}
		}
	}

	// Try parsing as ISO 8601 date/datetime
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timeRange); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("invalid time range format: %s", timeRange)
}

// ValidateTimeRange checks if a time range string is valid
func ValidateTimeRange(timeRange string) error {
	if timeRange == "" {
		return nil
	}
	_, err := ParseTimeRange(timeRange)
	return err
}
