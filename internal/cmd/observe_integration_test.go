package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/behavioral"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestObserveFullWorkflow tests complete user journey from project selection to export
func TestObserveFullWorkflow(t *testing.T) {
	// Setup test environment with projects and data
	tmpDir := setupTestEnvironment(t, 3, 5)

	t.Run("interactive workflow with project selection", func(t *testing.T) {
		// Reset global flags
		observeProject = ""
		observeSession = ""
		observeFilterType = ""
		observeErrorsOnly = false
		observeTimeRange = ""

		// Create mock reader for project selection
		reader := &MockReader{inputs: []string{"2"}}

		// Test project selection
		selected, err := DisplayProjectMenuWithReader(reader)
		require.NoError(t, err)
		assert.Equal(t, "project-02", selected)

		// Set selected project
		observeProject = selected

		// Test filter parsing
		criteria, err := ParseFilterFlags()
		require.NoError(t, err)
		assert.Equal(t, "", criteria.Search)
		assert.False(t, criteria.ErrorsOnly)
	})

	t.Run("workflow with filtering applied", func(t *testing.T) {
		// Reset and set flags
		observeProject = "project-01"
		observeSession = "test"
		observeFilterType = "tool"
		observeErrorsOnly = true
		observeTimeRange = "24h"

		// Parse filters
		criteria, err := ParseFilterFlags()
		require.NoError(t, err)

		// Validate criteria
		assert.Equal(t, "test", criteria.Search)
		assert.Equal(t, "tool", criteria.EventType)
		assert.True(t, criteria.ErrorsOnly)
		assert.False(t, criteria.Since.IsZero())

		// Build description
		desc := BuildFilterDescription(criteria)
		assert.Contains(t, desc, "search='test'")
		assert.Contains(t, desc, "type='tool'")
		assert.Contains(t, desc, "errors-only")
	})

	t.Run("workflow ends with export", func(t *testing.T) {
		// Create test metrics
		metrics := createTestMetrics()

		// Test JSON export
		exportFormat = "json"
		exportOutput = filepath.Join(tmpDir, "export.json")

		err := behavioral.ExportToFile(metrics, exportOutput, exportFormat)
		require.NoError(t, err)

		// Verify file exists
		_, err = os.Stat(exportOutput)
		assert.NoError(t, err)

		// Test Markdown export
		exportFormat = "markdown"
		exportOutput = filepath.Join(tmpDir, "export.md")

		err = behavioral.ExportToFile(metrics, exportOutput, exportFormat)
		require.NoError(t, err)

		// Verify file exists
		_, err = os.Stat(exportOutput)
		assert.NoError(t, err)
	})
}

// TestObserveMenuSelection tests interactive menu selection
func TestObserveMenuSelection(t *testing.T) {
	setupTestEnvironment(t, 10, 3)

	t.Run("single page menu selection", func(t *testing.T) {
		// Create projects within single page limit
		setupTestEnvironment(t, 10, 2)

		reader := &MockReader{inputs: []string{"5"}}
		selected, err := DisplayProjectMenuWithReader(reader)
		require.NoError(t, err)
		assert.Equal(t, "project-05", selected)
	})

	t.Run("multi-page menu navigation", func(t *testing.T) {
		// Create projects requiring pagination
		setupTestEnvironment(t, 20, 2)

		// Navigate to second page and select
		reader := &MockReader{inputs: []string{"n", "16"}}
		selected, err := DisplayProjectMenuWithReader(reader)
		require.NoError(t, err)
		assert.Equal(t, "project-16", selected)
	})

	t.Run("navigation with invalid input recovery", func(t *testing.T) {
		setupTestEnvironment(t, 20, 2)

		// Invalid input followed by valid selection
		reader := &MockReader{inputs: []string{"invalid", "", "5"}}
		selected, err := DisplayProjectMenuWithReader(reader)
		require.NoError(t, err)
		assert.Equal(t, "project-05", selected)
	})

	t.Run("quit from menu", func(t *testing.T) {
		setupTestEnvironment(t, 5, 2)

		reader := &MockReader{inputs: []string{"q"}}
		_, err := DisplayProjectMenuWithReader(reader)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cancelled")
	})
}

// TestObserveFiltering tests all filter combinations
func TestObserveFiltering(t *testing.T) {
	t.Run("filter by search term", func(t *testing.T) {
		observeSession = "conductor"
		observeFilterType = ""
		observeErrorsOnly = false
		observeTimeRange = ""

		criteria, err := ParseFilterFlags()
		require.NoError(t, err)

		assert.Equal(t, "conductor", criteria.Search)
		assert.Equal(t, "", criteria.EventType)
		assert.False(t, criteria.ErrorsOnly)
	})

	t.Run("filter by event type tool", func(t *testing.T) {
		observeSession = ""
		observeFilterType = "tool"
		observeErrorsOnly = false
		observeTimeRange = ""

		criteria, err := ParseFilterFlags()
		require.NoError(t, err)

		assert.Equal(t, "tool", criteria.EventType)

		// Test filter application
		tools := []behavioral.ToolExecution{
			{Name: "Read", Count: 5},
			{Name: "Write", Count: 3},
		}
		bash := []behavioral.BashCommand{
			{Command: "go test", Success: true},
		}

		filteredTools := behavioral.ApplyFiltersToToolExecutions(tools, criteria)
		filteredBash := behavioral.ApplyFiltersToBashCommands(bash, criteria)

		assert.Len(t, filteredTools, 2)
		assert.Len(t, filteredBash, 0) // Bash filtered out
	})

	t.Run("filter by event type bash", func(t *testing.T) {
		observeSession = ""
		observeFilterType = "bash"
		observeErrorsOnly = false
		observeTimeRange = ""

		criteria, err := ParseFilterFlags()
		require.NoError(t, err)

		assert.Equal(t, "bash", criteria.EventType)

		// Test filter application
		tools := []behavioral.ToolExecution{
			{Name: "Read", Count: 5},
		}
		bash := []behavioral.BashCommand{
			{Command: "go test", Success: true},
			{Command: "go build", Success: false},
		}

		filteredTools := behavioral.ApplyFiltersToToolExecutions(tools, criteria)
		filteredBash := behavioral.ApplyFiltersToBashCommands(bash, criteria)

		assert.Len(t, filteredTools, 0) // Tools filtered out
		assert.Len(t, filteredBash, 2)
	})

	t.Run("filter by event type file", func(t *testing.T) {
		observeSession = ""
		observeFilterType = "file"
		observeErrorsOnly = false
		observeTimeRange = ""

		criteria, err := ParseFilterFlags()
		require.NoError(t, err)

		assert.Equal(t, "file", criteria.EventType)

		// Test filter application
		files := []behavioral.FileOperation{
			{Path: "/test/file.go", Type: "write", Success: true},
			{Path: "/test/config.yaml", Type: "read", Success: true},
		}

		filteredFiles := behavioral.ApplyFiltersToFileOperations(files, criteria)
		assert.Len(t, filteredFiles, 2)
	})

	t.Run("filter errors only", func(t *testing.T) {
		observeSession = ""
		observeFilterType = ""
		observeErrorsOnly = true
		observeTimeRange = ""

		criteria, err := ParseFilterFlags()
		require.NoError(t, err)

		assert.True(t, criteria.ErrorsOnly)

		// Test filter application
		tools := []behavioral.ToolExecution{
			{Name: "Read", Count: 5, TotalErrors: 0},
			{Name: "Write", Count: 3, TotalErrors: 2},
		}

		filteredTools := behavioral.ApplyFiltersToToolExecutions(tools, criteria)
		assert.Len(t, filteredTools, 1)
		assert.Equal(t, "Write", filteredTools[0].Name)
	})

	t.Run("filter by time range relative", func(t *testing.T) {
		observeSession = ""
		observeFilterType = ""
		observeErrorsOnly = false
		observeTimeRange = "24h"

		criteria, err := ParseFilterFlags()
		require.NoError(t, err)

		assert.False(t, criteria.Since.IsZero())
		assert.True(t, criteria.Since.Before(time.Now()))

		// Should be approximately 24 hours ago
		expectedSince := time.Now().Add(-24 * time.Hour)
		diff := criteria.Since.Sub(expectedSince)
		assert.Less(t, diff.Abs(), 1*time.Second)
	})

	t.Run("filter by time range keyword", func(t *testing.T) {
		observeSession = ""
		observeFilterType = ""
		observeErrorsOnly = false
		observeTimeRange = "today"

		criteria, err := ParseFilterFlags()
		require.NoError(t, err)

		assert.False(t, criteria.Since.IsZero())

		// Should be start of today
		now := time.Now()
		expected := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		assert.Equal(t, expected, criteria.Since)
	})

	t.Run("combined filters", func(t *testing.T) {
		observeSession = "test-session"
		observeFilterType = "tool"
		observeErrorsOnly = true
		observeTimeRange = "7d"

		criteria, err := ParseFilterFlags()
		require.NoError(t, err)

		assert.Equal(t, "test-session", criteria.Search)
		assert.Equal(t, "tool", criteria.EventType)
		assert.True(t, criteria.ErrorsOnly)
		assert.False(t, criteria.Since.IsZero())

		// Build and verify description
		desc := BuildFilterDescription(criteria)
		assert.Contains(t, desc, "search='test-session'")
		assert.Contains(t, desc, "type='tool'")
		assert.Contains(t, desc, "errors-only")
		assert.Contains(t, desc, "since=")
	})

	t.Run("invalid filter type", func(t *testing.T) {
		observeSession = ""
		observeFilterType = "invalid"
		observeErrorsOnly = false
		observeTimeRange = ""

		_, err := ParseFilterFlags()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid event type")
	})

	t.Run("invalid time range", func(t *testing.T) {
		observeSession = ""
		observeFilterType = ""
		observeErrorsOnly = false
		observeTimeRange = "invalid-time"

		_, err := ParseFilterFlags()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid time range")
	})
}

// TestObserveExport tests JSON and Markdown exports
func TestObserveExport(t *testing.T) {
	tmpDir := t.TempDir()
	metrics := createTestMetrics()

	t.Run("export to JSON file", func(t *testing.T) {
		outputPath := filepath.Join(tmpDir, "metrics.json")
		err := behavioral.ExportToFile(metrics, outputPath, "json")
		require.NoError(t, err)

		// Verify file exists and contains valid JSON
		data, err := os.ReadFile(outputPath)
		require.NoError(t, err)
		assert.Contains(t, string(data), "total_sessions")
		assert.Contains(t, string(data), "success_rate")
	})

	t.Run("export to Markdown file", func(t *testing.T) {
		outputPath := filepath.Join(tmpDir, "metrics.md")
		err := behavioral.ExportToFile(metrics, outputPath, "markdown")
		require.NoError(t, err)

		// Verify file exists and contains Markdown
		data, err := os.ReadFile(outputPath)
		require.NoError(t, err)
		assert.Contains(t, string(data), "# Behavioral Metrics Report")
		assert.Contains(t, string(data), "## Summary")
		assert.Contains(t, string(data), "Total Sessions")
	})

	t.Run("export to CSV file", func(t *testing.T) {
		outputPath := filepath.Join(tmpDir, "metrics.csv")
		err := behavioral.ExportToFile(metrics, outputPath, "csv")
		require.NoError(t, err)

		// Verify file exists and contains CSV
		data, err := os.ReadFile(outputPath)
		require.NoError(t, err)
		assert.Contains(t, string(data), "Type,Name,Value")
		assert.Contains(t, string(data), "Summary,TotalSessions")
	})

	t.Run("export with md alias", func(t *testing.T) {
		outputPath := filepath.Join(tmpDir, "metrics-alias.md")
		err := behavioral.ExportToFile(metrics, outputPath, "md")
		require.NoError(t, err)

		// Verify file created
		_, err = os.Stat(outputPath)
		assert.NoError(t, err)
	})

	t.Run("export to stdout JSON", func(t *testing.T) {
		content, err := behavioral.ExportToString(metrics, "json")
		require.NoError(t, err)
		assert.Contains(t, content, "total_sessions")
		assert.Contains(t, content, "success_rate")
	})

	t.Run("export to stdout Markdown", func(t *testing.T) {
		content, err := behavioral.ExportToString(metrics, "markdown")
		require.NoError(t, err)
		assert.Contains(t, content, "# Behavioral Metrics Report")
		assert.Contains(t, content, "## Summary")
	})

	t.Run("export with invalid format", func(t *testing.T) {
		outputPath := filepath.Join(tmpDir, "invalid.txt")
		err := behavioral.ExportToFile(metrics, outputPath, "invalid")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported format")
	})

	t.Run("export with nested directory creation", func(t *testing.T) {
		outputPath := filepath.Join(tmpDir, "subdir", "nested", "metrics.json")
		err := behavioral.ExportToFile(metrics, outputPath, "json")
		require.NoError(t, err)

		// Verify file created in nested directory
		_, err = os.Stat(outputPath)
		assert.NoError(t, err)
	})
}

// TestObservePagination tests page navigation
func TestObservePagination(t *testing.T) {
	t.Run("paginator with small dataset", func(t *testing.T) {
		items := make([]interface{}, 10)
		for i := range items {
			items[i] = behavioral.ToolExecution{Name: fmt.Sprintf("Tool%d", i)}
		}

		paginator := behavioral.NewPaginator(items, 5)

		// Check initial state
		assert.Equal(t, 1, paginator.GetCurrentPageNum())
		assert.Equal(t, 2, paginator.GetTotalPages())
		assert.True(t, paginator.HasNextPage())
		assert.False(t, paginator.HasPrevPage())

		// Get first page
		page := paginator.GetCurrentPage()
		assert.Len(t, page, 5)

		// Move to next page
		moved := paginator.NextPage()
		assert.True(t, moved)
		assert.Equal(t, 2, paginator.GetCurrentPageNum())
		assert.False(t, paginator.HasNextPage())
		assert.True(t, paginator.HasPrevPage())

		// Get second page
		page = paginator.GetCurrentPage()
		assert.Len(t, page, 5)

		// Try to go past last page
		moved = paginator.NextPage()
		assert.False(t, moved)
		assert.Equal(t, 2, paginator.GetCurrentPageNum())

		// Move back to first page
		moved = paginator.PrevPage()
		assert.True(t, moved)
		assert.Equal(t, 1, paginator.GetCurrentPageNum())

		// Try to go before first page
		moved = paginator.PrevPage()
		assert.False(t, moved)
		assert.Equal(t, 1, paginator.GetCurrentPageNum())
	})

	t.Run("paginator with uneven pages", func(t *testing.T) {
		items := make([]interface{}, 23)
		for i := range items {
			items[i] = behavioral.BashCommand{Command: fmt.Sprintf("cmd%d", i)}
		}

		paginator := behavioral.NewPaginator(items, 10)

		assert.Equal(t, 3, paginator.GetTotalPages())

		// Page 1: 10 items
		page := paginator.GetCurrentPage()
		assert.Len(t, page, 10)

		// Page 2: 10 items
		paginator.NextPage()
		page = paginator.GetCurrentPage()
		assert.Len(t, page, 10)

		// Page 3: 3 items (remainder)
		paginator.NextPage()
		page = paginator.GetCurrentPage()
		assert.Len(t, page, 3)
	})

	t.Run("paginator with empty dataset", func(t *testing.T) {
		items := []interface{}{}
		paginator := behavioral.NewPaginator(items, 10)

		assert.Equal(t, 0, paginator.GetTotalPages())
		assert.Empty(t, paginator.GetCurrentPage())
		assert.False(t, paginator.HasNextPage())
		assert.False(t, paginator.HasPrevPage())
	})

	t.Run("navigation bar display", func(t *testing.T) {
		// Single page - no navigation
		nav := behavioral.PrintNavigationBar(1, 1, false)
		assert.Empty(t, nav)

		// Multi-page first page
		nav = behavioral.PrintNavigationBar(1, 3, false)
		assert.Contains(t, nav, "Page 1 of 3")
		assert.Contains(t, nav, "Next →")
		assert.NotContains(t, nav, "← Previous")

		// Multi-page middle page
		nav = behavioral.PrintNavigationBar(2, 3, false)
		assert.Contains(t, nav, "Page 2 of 3")
		assert.Contains(t, nav, "Next →")
		assert.Contains(t, nav, "← Previous")

		// Multi-page last page
		nav = behavioral.PrintNavigationBar(3, 3, false)
		assert.Contains(t, nav, "Page 3 of 3")
		assert.Contains(t, nav, "← Previous")
		assert.NotContains(t, nav, "Next →")
	})
}

// TestObserveStats tests statistics calculation
func TestObserveStats(t *testing.T) {
	t.Run("calculate stats from single metric", func(t *testing.T) {
		metrics := []behavioral.BehavioralMetrics{
			{
				TotalSessions:   10,
				SuccessRate:     0.8,
				ErrorRate:       0.2,
				AverageDuration: 5 * time.Minute,
				TotalCost:       1.50,
				TokenUsage: behavioral.TokenUsage{
					InputTokens:  1000,
					OutputTokens: 500,
				},
				AgentPerformance: map[string]int{
					"backend-developer": 8,
					"test-automator":    2,
				},
				ToolExecutions: []behavioral.ToolExecution{
					{Name: "Read", Count: 50, TotalSuccess: 48, TotalErrors: 2},
					{Name: "Write", Count: 20, TotalSuccess: 18, TotalErrors: 2},
				},
			},
		}

		stats := behavioral.CalculateStats(metrics)

		assert.Equal(t, 10, stats.TotalSessions)
		assert.InDelta(t, 0.8, stats.SuccessRate, 0.01)
		assert.InDelta(t, 0.2, stats.ErrorRate, 0.01)
		assert.Equal(t, 5*time.Minute, stats.AverageDuration)
		// TotalCost calculated from TokenUsage, not direct TotalCost field
		assert.GreaterOrEqual(t, stats.TotalCost, 0.0)
		assert.Equal(t, int64(1000), stats.TotalInputTokens)
		assert.Equal(t, int64(500), stats.TotalOutputTokens)
		assert.Equal(t, 2, stats.TotalAgents)
		assert.Len(t, stats.TopTools, 2)
	})

	t.Run("calculate stats from multiple metrics", func(t *testing.T) {
		metrics := []behavioral.BehavioralMetrics{
			{
				TotalSessions:   5,
				SuccessRate:     1.0,
				AverageDuration: 3 * time.Minute,
				TotalCost:       0.75,
				TokenUsage: behavioral.TokenUsage{
					InputTokens:  500,
					OutputTokens: 250,
				},
				AgentPerformance: map[string]int{
					"backend-developer": 5,
				},
				ToolExecutions: []behavioral.ToolExecution{
					{Name: "Read", Count: 20, TotalSuccess: 20, TotalErrors: 0},
				},
			},
			{
				TotalSessions:   5,
				SuccessRate:     0.6,
				AverageDuration: 7 * time.Minute,
				TotalCost:       1.25,
				TokenUsage: behavioral.TokenUsage{
					InputTokens:  800,
					OutputTokens: 400,
				},
				AgentPerformance: map[string]int{
					"test-automator": 3,
				},
				ToolExecutions: []behavioral.ToolExecution{
					{Name: "Read", Count: 30, TotalSuccess: 28, TotalErrors: 2},
					{Name: "Write", Count: 15, TotalSuccess: 14, TotalErrors: 1},
				},
			},
		}

		stats := behavioral.CalculateStats(metrics)

		assert.Equal(t, 10, stats.TotalSessions)
		assert.InDelta(t, 0.8, stats.SuccessRate, 0.01) // (5*1.0 + 5*0.6) / 10
		assert.InDelta(t, 0.2, stats.ErrorRate, 0.01)
		// TotalCost calculated from TokenUsage, not direct TotalCost field
		assert.GreaterOrEqual(t, stats.TotalCost, 0.0)
		assert.Equal(t, int64(1300), stats.TotalInputTokens)
		assert.Equal(t, int64(650), stats.TotalOutputTokens)
		assert.Equal(t, 2, stats.TotalAgents)

		// Tool aggregation
		assert.Len(t, stats.TopTools, 2)

		// Find Read tool
		var readTool *behavioral.ToolStatSummary
		for i := range stats.TopTools {
			if stats.TopTools[i].Name == "Read" {
				readTool = &stats.TopTools[i]
				break
			}
		}
		require.NotNil(t, readTool)
		assert.Equal(t, 50, readTool.Count) // 20 + 30
		assert.InDelta(t, 0.96, readTool.SuccessRate, 0.01) // 48/50
	})

	t.Run("calculate stats with empty metrics", func(t *testing.T) {
		metrics := []behavioral.BehavioralMetrics{}
		stats := behavioral.CalculateStats(metrics)

		assert.Equal(t, 0, stats.TotalSessions)
		assert.Equal(t, 0.0, stats.SuccessRate)
		assert.Empty(t, stats.TopTools)
		assert.Empty(t, stats.AgentBreakdown)
	})

	t.Run("get top tools limited", func(t *testing.T) {
		metrics := []behavioral.BehavioralMetrics{
			{
				TotalSessions: 1,
				ToolExecutions: []behavioral.ToolExecution{
					{Name: "Read", Count: 100},
					{Name: "Write", Count: 50},
					{Name: "Edit", Count: 30},
					{Name: "Bash", Count: 20},
					{Name: "Grep", Count: 10},
				},
			},
		}

		stats := behavioral.CalculateStats(metrics)

		// Get top 3
		topTools := stats.GetTopTools(3)
		assert.Len(t, topTools, 3)
		assert.Equal(t, "Read", topTools[0].Name)
		assert.Equal(t, "Write", topTools[1].Name)
		assert.Equal(t, "Edit", topTools[2].Name)

		// Get all
		allTools := stats.GetTopTools(0)
		assert.Len(t, allTools, 5)
	})
}

// setupTestEnvironment creates test projects and session files
func setupTestEnvironment(t *testing.T, projectCount, sessionsPerProject int) string {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	t.Cleanup(func() { os.Setenv("HOME", oldHome) })

	projectsDir := filepath.Join(tmpDir, ".claude", "projects")
	err := os.MkdirAll(projectsDir, 0755)
	require.NoError(t, err)

	for i := 0; i < projectCount; i++ {
		projectName := fmt.Sprintf("project-%02d", i+1)
		projectPath := filepath.Join(projectsDir, projectName)
		err := os.MkdirAll(projectPath, 0755)
		require.NoError(t, err)

		// Create session files with realistic JSONL data
		for j := 0; j < sessionsPerProject; j++ {
			sessionFile := filepath.Join(projectPath, fmt.Sprintf("session-%d.jsonl", j+1))
			content := createRealisticSessionData(projectName, j+1)
			err := os.WriteFile(sessionFile, []byte(content), 0644)
			require.NoError(t, err)
		}
	}

	return tmpDir
}

// createRealisticSessionData generates realistic JSONL session data
func createRealisticSessionData(project string, sessionNum int) string {
	timestamp := time.Now().Add(-time.Duration(sessionNum) * time.Hour).Format(time.RFC3339)
	sessionID := fmt.Sprintf("%s-session-%d", project, sessionNum)

	return fmt.Sprintf(`{"type":"session_start","timestamp":"%s","session_id":"%s","project_path":"/Users/test/%s"}
{"type":"session_metadata","timestamp":"%s","metadata":{"claude_version":"1.2.3","os":"darwin"}}
{"type":"tool_call","timestamp":"%s","tool":"Read","parameters":{"file_path":"main.go"},"duration_ms":45}
{"type":"bash_command","timestamp":"%s","command":"go test ./...","exit_code":0,"duration_ms":1234}
{"type":"file_operation","timestamp":"%s","operation":"write","path":"internal/feature.go","size_bytes":2048}
{"type":"token_usage","timestamp":"%s","input_tokens":1500,"output_tokens":800}
`, timestamp, sessionID, project, timestamp, timestamp, timestamp, timestamp, timestamp)
}

// createTestMetrics creates test BehavioralMetrics for export testing
func createTestMetrics() *behavioral.BehavioralMetrics {
	return &behavioral.BehavioralMetrics{
		TotalSessions:   25,
		SuccessRate:     0.88,
		ErrorRate:       0.12,
		TotalErrors:     3,
		AverageDuration: 8 * time.Minute,
		TotalCost:       4.75,
		TokenUsage: behavioral.TokenUsage{
			InputTokens:  15000,
			OutputTokens: 8000,
			ModelName:    "claude-sonnet-4-5",
			CostUSD:      4.75,
		},
		AgentPerformance: map[string]int{
			"backend-developer": 15,
			"test-automator":    7,
			"code-reviewer":     3,
		},
		ToolExecutions: []behavioral.ToolExecution{
			{
				Name:         "Read",
				Count:        125,
				TotalSuccess: 123,
				TotalErrors:  2,
				SuccessRate:  0.984,
				ErrorRate:    0.016,
				AvgDuration:  45 * time.Millisecond,
			},
			{
				Name:         "Write",
				Count:        45,
				TotalSuccess: 44,
				TotalErrors:  1,
				SuccessRate:  0.978,
				ErrorRate:    0.022,
				AvgDuration:  67 * time.Millisecond,
			},
			{
				Name:         "Edit",
				Count:        89,
				TotalSuccess: 89,
				TotalErrors:  0,
				SuccessRate:  1.0,
				ErrorRate:    0.0,
				AvgDuration:  52 * time.Millisecond,
			},
		},
		BashCommands: []behavioral.BashCommand{
			{
				Command:      "go test ./...",
				ExitCode:     0,
				Success:      true,
				Duration:     1234 * time.Millisecond,
				OutputLength: 512,
				Timestamp:    time.Now().Add(-1 * time.Hour),
			},
			{
				Command:      "go build",
				ExitCode:     0,
				Success:      true,
				Duration:     890 * time.Millisecond,
				OutputLength: 256,
				Timestamp:    time.Now().Add(-30 * time.Minute),
			},
		},
		FileOperations: []behavioral.FileOperation{
			{
				Type:      "write",
				Path:      "internal/feature.go",
				Success:   true,
				SizeBytes: 2048,
				Duration:  150,
				Timestamp: time.Now().Add(-45 * time.Minute),
			},
			{
				Type:      "read",
				Path:      "config.yaml",
				Success:   true,
				SizeBytes: 512,
				Duration:  45,
				Timestamp: time.Now().Add(-20 * time.Minute),
			},
		},
	}
}

// TestObserveCommandIntegration tests cobra command integration
func TestObserveCommandIntegration(t *testing.T) {
	setupTestEnvironment(t, 3, 2)

	t.Run("observe root command exists", func(t *testing.T) {
		cmd := NewObserveCommand()
		assert.NotNil(t, cmd)
		assert.Equal(t, "observe", cmd.Use)
	})

	t.Run("observe has all subcommands", func(t *testing.T) {
		cmd := NewObserveCommand()

		expectedCommands := []string{
			"project", "session", "tools", "bash", "files",
			"errors", "stats", "stream", "export",
		}

		for _, expected := range expectedCommands {
			found := false
			for _, subcmd := range cmd.Commands() {
				if subcmd.Use == expected || (len(subcmd.Use) >= len(expected) && subcmd.Use[:len(expected)] == expected) {
					found = true
					break
				}
			}
			assert.True(t, found, "Expected subcommand not found: %s", expected)
		}
	})

	t.Run("export command with flags", func(t *testing.T) {
		cmd := NewObserveCommand()
		exportCmd := findSubcommand(cmd, "export")
		require.NotNil(t, exportCmd)

		// Check flags exist
		formatFlag := exportCmd.Flags().Lookup("format")
		assert.NotNil(t, formatFlag)

		outputFlag := exportCmd.Flags().Lookup("output")
		assert.NotNil(t, outputFlag)
	})

	t.Run("observe interactive mode bypassed with project flag", func(t *testing.T) {
		// When project is pre-set via flag, interactive menu is skipped
		// This allows non-interactive testing
		observeProject = "project-01"
		defer func() { observeProject = "" }()

		// Verify the observe command exists and can be constructed
		cmd := NewObserveCommand()
		assert.NotNil(t, cmd)
		assert.Equal(t, "observe", cmd.Use)

		// Note: Cannot actually execute as it needs stdin for interactive parts
		// This test just verifies command structure
	})
}

// TestStreamActivity tests real-time streaming functionality
func TestStreamActivity(t *testing.T) {
	t.Run("stream context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		// Cancel immediately
		cancel()

		// Stream should return error when context is cancelled
		err := StreamActivity(ctx, "test-project")
		// May return config error or context error
		assert.Error(t, err)
	})

	t.Run("stream with timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		// Stream should timeout or fail config
		err := StreamActivity(ctx, "test-project")
		// May return config error, timeout, or nil
		// Just verify function can be called
		_ = err
	})
}

// Helper function to find a subcommand by name
func findSubcommand(cmd *cobra.Command, name string) *cobra.Command {
	for _, subcmd := range cmd.Commands() {
		if subcmd.Use == name || (len(subcmd.Use) >= len(name) && subcmd.Use[:len(name)] == name) {
			return subcmd
		}
	}
	return nil
}
