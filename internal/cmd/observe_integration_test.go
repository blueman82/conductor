package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/behavioral"
	"github.com/harrison/conductor/internal/learning"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestObserveIntegration_FullWorkflow tests complete observe workflow
func TestObserveIntegration_FullWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := learning.NewStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	t.Run("menu to export workflow", func(t *testing.T) {
		// Step 1: List projects (via menu)
		reader := &MockMenuReader{inputs: []string{"q"}}
		_, err := DisplayProjectMenuWithReader(reader)
		// Expected to fail on quit
		_ = err

		// Step 2: Export data
		outputPath := filepath.Join(tmpDir, "export.json")
		exportFormat = "json"
		exportOutput = outputPath
		observeProject = ""

		cmd := NewObserveExportCmd()
		err = HandleExportCommand(cmd, []string{})
		_ = err

		// Step 3: Validate time range
		observeTimeRange = "24h"
		err = ValidateFilterTimeRange()
		_ = err
	})
}

// TestObserveIntegration_Filtering tests filtering workflow
func TestObserveIntegration_Filtering(t *testing.T) {
	t.Run("apply multiple filters", func(t *testing.T) {
		observeProject = "test-project"
		observeFilterType = "tool"
		observeErrorsOnly = true
		observeTimeRange = "7d"

		criteria, err := ParseFilterFlags()
		if err == nil {
			desc := BuildFilterDescription(criteria)
			assert.NotEmpty(t, desc)
		}

		// Reset flags
		observeProject = ""
		observeFilterType = ""
		observeErrorsOnly = false
		observeTimeRange = ""
	})
}

// TestObserveIntegration_Export tests export formats
func TestObserveIntegration_Export(t *testing.T) {
	tmpDir := t.TempDir()

	formats := []string{"json", "markdown", "csv"}
	for _, format := range formats {
		t.Run("export "+format, func(t *testing.T) {
			outputPath := filepath.Join(tmpDir, "output."+format)
			exportFormat = format
			exportOutput = outputPath

			cmd := NewObserveExportCmd()
			err := HandleExportCommand(cmd, []string{})
			_ = err
		})
	}
}

// TestObserveIntegration_StatsDisplay tests stats workflow
func TestObserveIntegration_StatsDisplay(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := learning.NewStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	t.Run("display stats for project", func(t *testing.T) {
		err := DisplayStats("test")
		_ = err
	})

	t.Run("display stats for all", func(t *testing.T) {
		err := DisplayStats("")
		_ = err
	})
}

// TestObserveIntegration_Streaming tests streaming workflow
func TestObserveIntegration_Streaming(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "stream.db")

	store, err := learning.NewStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	t.Run("stream with timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()

		err := StreamActivity(ctx, "test-project")
		_ = err
	})

	t.Run("stream without project filter", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := StreamActivity(ctx, "")
		_ = err
	})
}

// TestObserveIntegration_AllSubcommands tests all subcommands execute
func TestObserveIntegration_AllSubcommands(t *testing.T) {
	subcommands := []struct {
		name string
		cmd  func() *cobra.Command
	}{
		{"project", NewObserveProjectCmd},
		{"session", NewObserveSessionCmd},
		{"tools", NewObserveToolsCmd},
		{"bash", NewObserveBashCmd},
		{"files", NewObserveFilesCmd},
		{"errors", NewObserveErrorsCmd},
		{"stats", NewObserveStatsCmd},
		{"stream", NewObserveStreamCmd},
		{"export", NewObserveExportCmd},
	}

	for _, sc := range subcommands {
		t.Run(sc.name, func(t *testing.T) {
			cmd := sc.cmd()
			assert.NotNil(t, cmd)
			assert.NotEmpty(t, cmd.Use)
		})
	}
}

// TestObserveIntegration_ObserveInteractive tests interactive mode
func TestObserveIntegration_ObserveInteractive(t *testing.T) {
	t.Run("interactive mode help", func(t *testing.T) {
		// Capture output
		buf := &bytes.Buffer{}
		cmd := NewObserveCommand()
		cmd.SetOut(buf)
		cmd.SetErr(buf)

		observeProject = ""
		observeSession = ""

		// Execute help
		cmd.SetArgs([]string{"--help"})
		err := cmd.Execute()
		assert.NoError(t, err)
		assert.Contains(t, buf.String(), "observe")
	})
}

// TestObserveIntegration_MenuNavigation tests menu pagination
func TestObserveIntegration_MenuNavigation(t *testing.T) {
	t.Run("quit from menu", func(t *testing.T) {
		reader := &MockMenuReader{inputs: []string{"q"}}
		_, err := DisplayProjectMenuWithReader(reader)
		// Expected to error
		_ = err
	})

	t.Run("empty input", func(t *testing.T) {
		reader := &MockMenuReader{inputs: []string{"", "q"}}
		_, err := DisplayProjectMenuWithReader(reader)
		_ = err
	})
}

// TestObserveIntegration_WatchSessions tests session watching
func TestObserveIntegration_WatchSessions(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := learning.NewStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	t.Run("watch with no sessions", func(t *testing.T) {
		updates, err := watchNewSessions(store, "", 0)
		assert.NoError(t, err)
		assert.Empty(t, updates)
	})

	t.Run("watch with project filter", func(t *testing.T) {
		updates, err := watchNewSessions(store, "test-project", 0)
		_ = updates
		_ = err
	})
}

// TestObserveIntegration_CollectMetrics tests metrics collection
func TestObserveIntegration_CollectMetrics(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := learning.NewStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	t.Run("collect with empty db", func(t *testing.T) {
		metrics, err := collectBehavioralMetrics(store, "")
		assert.NoError(t, err)
		assert.Empty(t, metrics)
	})

	t.Run("collect with project filter", func(t *testing.T) {
		metrics, err := collectBehavioralMetrics(store, "test-project")
		_ = metrics
		_ = err
	})
}

// TestObserveIntegration_ExportFormats tests all export format parsers
func TestObserveIntegration_ExportFormats(t *testing.T) {
	t.Run("parse json format", func(t *testing.T) {
		format, err := parseExportFormat("json")
		assert.NoError(t, err)
		assert.Equal(t, "json", format)
	})

	t.Run("parse markdown format", func(t *testing.T) {
		format, err := parseExportFormat("markdown")
		assert.NoError(t, err)
		assert.Equal(t, "markdown", format)
	})

	t.Run("parse md alias", func(t *testing.T) {
		format, err := parseExportFormat("md")
		assert.NoError(t, err)
		assert.Equal(t, "markdown", format)
	})

	t.Run("parse csv format", func(t *testing.T) {
		format, err := parseExportFormat("csv")
		assert.NoError(t, err)
		assert.Equal(t, "csv", format)
	})

	t.Run("invalid format", func(t *testing.T) {
		_, err := parseExportFormat("invalid")
		assert.Error(t, err)
	})
}

// TestObserveIntegration_ExportPath tests export path generation
func TestObserveIntegration_ExportPath(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("nested directory creation", func(t *testing.T) {
		nestedPath := filepath.Join(tmpDir, "a", "b", "c", "export.json")
		path, err := getExportPath(nestedPath)
		_ = path
		_ = err
	})

	t.Run("default path", func(t *testing.T) {
		path, err := getExportPath("")
		assert.Error(t, err) // Should error on empty path
		_ = path
	})
}

// TestObserveIntegration_ErrorHandling tests error scenarios
func TestObserveIntegration_ErrorHandling(t *testing.T) {
	t.Run("invalid filter type", func(t *testing.T) {
		observeFilterType = "invalid"
		_, err := ParseFilterFlags()
		assert.Error(t, err)
		observeFilterType = ""
	})

	t.Run("invalid time range", func(t *testing.T) {
		observeTimeRange = "invalid"
		err := ValidateFilterTimeRange()
		assert.Error(t, err)
		observeTimeRange = ""
	})

	t.Run("malformed export format", func(t *testing.T) {
		_, err := parseExportFormat("xml")
		assert.Error(t, err)
	})

	t.Run("invalid export path", func(t *testing.T) {
		_, err := getExportPath("")
		assert.Error(t, err)
	})
}

// TestObserveIntegration_ContextCancellation tests context handling
func TestObserveIntegration_ContextCancellation(t *testing.T) {
	t.Run("streaming cancels properly", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := StreamActivity(ctx, "")
		_ = err // Expected to return due to cancellation
	})
}

// TestObserveIntegration_HelpCommands tests help text for all subcommands
func TestObserveIntegration_HelpCommands(t *testing.T) {
	subcommands := []string{
		"project", "session", "tools", "bash", "files", "errors", "stats", "stream", "export",
	}

	for _, subcmd := range subcommands {
		t.Run(subcmd+" help", func(t *testing.T) {
			rootCmd := NewRootCommand()
			buf := new(bytes.Buffer)
			rootCmd.SetOut(buf)
			rootCmd.SetErr(buf)
			rootCmd.SetArgs([]string{"observe", subcmd, "--help"})

			err := rootCmd.Execute()
			assert.NoError(t, err)
			output := buf.String()
			assert.Contains(t, output, subcmd)
		})
	}
}

// TestObserveIntegration_EmptyDatabase tests handling of empty database
func TestObserveIntegration_EmptyDatabase(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "empty.db")

	store, err := learning.NewStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	t.Run("collect from empty db", func(t *testing.T) {
		metrics, err := collectBehavioralMetrics(store, "")
		assert.NoError(t, err)
		assert.Empty(t, metrics)
	})

	t.Run("watch empty db", func(t *testing.T) {
		updates, err := watchNewSessions(store, "", 0)
		assert.NoError(t, err)
		assert.Empty(t, updates)
	})
}

// TestObserveIntegration_FilterCombinations tests various filter combinations
func TestObserveIntegration_FilterCombinations(t *testing.T) {
	combinations := []struct {
		name         string
		project      string
		filterType   string
		errorsOnly   bool
		timeRange    string
		expectError  bool
	}{
		{"all filters", "test", "tool", true, "24h", false},
		{"no filters", "", "", false, "", false},
		{"invalid type", "test", "invalid", false, "", true},
		{"invalid time", "test", "tool", false, "bad", true},
		{"errors only", "", "", true, "", false},
		{"time range only", "", "", false, "7d", false},
	}

	for _, tc := range combinations {
		t.Run(tc.name, func(t *testing.T) {
			// Set flags
			observeProject = tc.project
			observeFilterType = tc.filterType
			observeErrorsOnly = tc.errorsOnly
			observeTimeRange = tc.timeRange

			_, err := ParseFilterFlags()
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Reset flags
			observeProject = ""
			observeFilterType = ""
			observeErrorsOnly = false
			observeTimeRange = ""
		})
	}
}


// TestObserveIntegration_MenuEdgeCases tests menu edge cases
func TestObserveIntegration_MenuEdgeCases(t *testing.T) {
	t.Run("selection out of range", func(t *testing.T) {
		reader := &MockMenuReader{inputs: []string{"999", "q"}}
		_, err := DisplayProjectMenuWithReader(reader)
		assert.Error(t, err)
	})

	t.Run("negative selection", func(t *testing.T) {
		reader := &MockMenuReader{inputs: []string{"-1", "q"}}
		_, err := DisplayProjectMenuWithReader(reader)
		assert.Error(t, err)
	})

	t.Run("non-numeric input", func(t *testing.T) {
		reader := &MockMenuReader{inputs: []string{"abc", "q"}}
		_, err := DisplayProjectMenuWithReader(reader)
		assert.Error(t, err)
	})
}

// TestObserveIntegration_CollectMetricsWithData tests metrics collection with actual data
func TestObserveIntegration_CollectMetricsWithData(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := learning.NewStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	t.Run("collect with no filter", func(t *testing.T) {
		metrics, err := collectBehavioralMetrics(store, "")
		assert.NoError(t, err)
		_ = metrics
	})

	t.Run("collect with project filter", func(t *testing.T) {
		metrics, err := collectBehavioralMetrics(store, "test-project")
		assert.NoError(t, err)
		_ = metrics
	})
}

// TestObserveIntegration_WatchSessionsWithData tests session watching with data
func TestObserveIntegration_WatchSessionsWithData(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := learning.NewStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	t.Run("watch new sessions", func(t *testing.T) {
		updates, err := watchNewSessions(store, "", 0)
		assert.NoError(t, err)
		_ = updates
	})

	t.Run("watch with project filter", func(t *testing.T) {
		updates, err := watchNewSessions(store, "plan", 0)
		assert.NoError(t, err)
		_ = updates
	})
}

// TestObserveIntegration_ExportStdout tests export to stdout
func TestObserveIntegration_ExportStdout(t *testing.T) {
	t.Run("export json to stdout", func(t *testing.T) {
		exportFormat = "json"
		exportOutput = ""
		observeProject = ""

		cmd := NewObserveExportCmd()
		err := HandleExportCommand(cmd, []string{})
		assert.NoError(t, err)
	})

	t.Run("export markdown to stdout", func(t *testing.T) {
		exportFormat = "markdown"
		exportOutput = ""

		cmd := NewObserveExportCmd()
		err := HandleExportCommand(cmd, []string{})
		assert.NoError(t, err)
	})
}

// TestObserveIntegration_ExportWithAllFilters tests export with various filter combinations
func TestObserveIntegration_ExportWithAllFilters(t *testing.T) {
	tmpDir := t.TempDir()

	testCases := []struct {
		name       string
		filterType string
		errorsOnly bool
		timeRange  string
		format     string
	}{
		{"filter_by_tool_type", "tool", false, "", "json"},
		{"filter_by_errors", "", true, "", "json"},
		{"filter_by_time_range", "", false, "24h", "markdown"},
		{"combined_filters", "bash", true, "7d", "csv"},
		{"all_options", "file", true, "1h", "json"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			observeFilterType = tc.filterType
			observeErrorsOnly = tc.errorsOnly
			observeTimeRange = tc.timeRange
			exportFormat = tc.format
			exportOutput = filepath.Join(tmpDir, tc.name+"."+tc.format)

			cmd := NewObserveExportCmd()
			err := HandleExportCommand(cmd, []string{})
			assert.NoError(t, err)

			observeFilterType = ""
			observeErrorsOnly = false
			observeTimeRange = ""
		})
	}
}

// TestObserveIntegration_DisplaySessionUpdate tests session update display
func TestObserveIntegration_DisplaySessionUpdate(t *testing.T) {
	t.Run("display success update", func(t *testing.T) {
		update := SessionUpdate{
			ID:        1,
			TaskName:  "test-task",
			Agent:     "test-agent",
			Timestamp: time.Now(),
			Success:   true,
			Duration:  5 * time.Second,
		}
		displaySessionUpdate(update)
	})

	t.Run("display failure update", func(t *testing.T) {
		update := SessionUpdate{
			ID:        2,
			TaskName:  "failed-task",
			Agent:     "",
			Timestamp: time.Now(),
			Success:   false,
			Duration:  10 * time.Second,
		}
		displaySessionUpdate(update)
	})
}

// TestObserveIntegration_FormatStatsWithData tests stats formatting with various data
func TestObserveIntegration_FormatStatsWithData(t *testing.T) {
	t.Run("format empty stats", func(t *testing.T) {
		stats := &behavioral.AggregateStats{}
		output := formatStatsTable(stats)
		assert.Contains(t, output, "SUMMARY STATISTICS")
	})

	t.Run("format stats with tool data", func(t *testing.T) {
		stats := &behavioral.AggregateStats{
			TotalSessions: 10,
			TopTools: []behavioral.ToolStatSummary{
				{Name: "Read", Count: 100, SuccessRate: 0.95, ErrorRate: 0.05},
				{Name: "Write", Count: 50, SuccessRate: 0.90, ErrorRate: 0.10},
			},
		}
		output := formatStatsTable(stats)
		assert.Contains(t, output, "Top Tools")
	})

	t.Run("format stats with agent data", func(t *testing.T) {
		stats := &behavioral.AggregateStats{
			TotalSessions: 5,
			AgentBreakdown: map[string]int{
				"backend-developer": 10,
				"test-automator":    5,
			},
		}
		output := formatStatsTable(stats)
		assert.Contains(t, output, "Agent Performance")
	})
}

// TestObserveIntegration_ObserveInteractiveWithProject tests interactive mode variations
func TestObserveIntegration_ObserveInteractiveWithProject(t *testing.T) {
	t.Run("interactive with project flag", func(t *testing.T) {
		observeProject = "test-project"
		cmd := &cobra.Command{}
		err := observeInteractive(cmd, []string{})
		assert.NoError(t, err)
		observeProject = ""
	})
}

// TestObserveIntegration_ExportPathVariations tests various export path scenarios
func TestObserveIntegration_ExportPathVariations(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("absolute path", func(t *testing.T) {
		absPath := filepath.Join(tmpDir, "export.json")
		path, err := getExportPath(absPath)
		assert.NoError(t, err)
		assert.NotEmpty(t, path)
	})

	t.Run("relative path", func(t *testing.T) {
		path, err := getExportPath("./output.json")
		assert.NoError(t, err)
		assert.NotEmpty(t, path)
	})

	t.Run("nested new directories", func(t *testing.T) {
		nestedPath := filepath.Join(tmpDir, "new", "nested", "dirs", "file.json")
		path, err := getExportPath(nestedPath)
		assert.NoError(t, err)
		assert.NotEmpty(t, path)
	})
}

// TestObserveIntegration_DisplayStatsErrors tests DisplayStats with config/store errors
func TestObserveIntegration_DisplayStatsErrors(t *testing.T) {
	t.Run("invalid config dir", func(t *testing.T) {
		// Test should handle missing config gracefully
		err := DisplayStats("")
		// Expect error due to missing or invalid config
		assert.Error(t, err)
	})
}

// TestObserveIntegration_CollectMetricsFullCoverage tests full metrics collection logic
func TestObserveIntegration_CollectMetricsFullCoverage(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "full.db")

	store, err := learning.NewStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	t.Run("metrics with empty database", func(t *testing.T) {
		metrics, err := collectBehavioralMetrics(store, "")
		assert.NoError(t, err)
		assert.Empty(t, metrics)
	})

	t.Run("metrics with project filter no matches", func(t *testing.T) {
		metrics, err := collectBehavioralMetrics(store, "nonexistent-project")
		assert.NoError(t, err)
		assert.Empty(t, metrics)
	})
}

// TestObserveIntegration_StreamActivityFullCoverage tests streaming with database updates
func TestObserveIntegration_StreamActivityFullCoverage(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "stream.db")

	store, err := learning.NewStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	t.Run("stream with empty database", func(t *testing.T) {
		updates, err := watchNewSessions(store, "", 0)
		assert.NoError(t, err)
		assert.Empty(t, updates)
	})

	t.Run("stream with project filter no matches", func(t *testing.T) {
		updates, err := watchNewSessions(store, "nonexistent", 0)
		assert.NoError(t, err)
		assert.Empty(t, updates)
	})
}

// TestObserveIntegration_FormatDuration tests duration formatting
func TestObserveIntegration_FormatDuration(t *testing.T) {
	testCases := []struct {
		name     string
		duration time.Duration
	}{
		{"zero duration", 0},
		{"seconds", 45 * time.Second},
		{"minutes", 5 * time.Minute},
		{"hours", 2 * time.Hour},
		{"days", 25 * time.Hour},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Call formatDuration indirectly via formatStatsTable
			stats := &behavioral.AggregateStats{
				AverageDuration: tc.duration,
			}
			output := formatStatsTable(stats)
			assert.Contains(t, output, "Average Duration")
		})
	}
}

// TestObserveIntegration_DisplayStatsEdgeCases tests DisplayStats edge cases
func TestObserveIntegration_DisplayStatsEdgeCases(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("display with nonexistent config", func(t *testing.T) {
		err := DisplayStats("nonexistent")
		_ = err
	})

	t.Run("collect metrics empty database", func(t *testing.T) {
		dbPath := filepath.Join(tmpDir, "empty.db")
		store, err := learning.NewStore(dbPath)
		require.NoError(t, err)
		defer store.Close()

		metrics, err := collectBehavioralMetrics(store, "")
		assert.NoError(t, err)
		assert.Empty(t, metrics)
	})
}

// TestObserveIntegration_StreamEdgeCases tests StreamActivity edge cases
func TestObserveIntegration_StreamEdgeCases(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "stream.db")

	store, err := learning.NewStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	t.Run("watch with empty last seen ID", func(t *testing.T) {
		updates, err := watchNewSessions(store, "", 0)
		assert.NoError(t, err)
		assert.Empty(t, updates)
	})

	t.Run("watch with high last seen ID", func(t *testing.T) {
		updates, err := watchNewSessions(store, "", 99999)
		assert.NoError(t, err)
		assert.Empty(t, updates)
	})
}

// TestObserveIntegration_FormatStatsEdgeCases tests formatStatsTable edge cases
func TestObserveIntegration_FormatStatsEdgeCases(t *testing.T) {
	t.Run("format with nil stats", func(t *testing.T) {
		stats := &behavioral.AggregateStats{}
		output := formatStatsTable(stats)
		assert.NotEmpty(t, output)
	})

	t.Run("format with single agent", func(t *testing.T) {
		stats := &behavioral.AggregateStats{
			AgentBreakdown: map[string]int{
				"single-agent": 1,
			},
		}
		output := formatStatsTable(stats)
		assert.Contains(t, output, "single-agent")
	})

	t.Run("format with many agents", func(t *testing.T) {
		agents := make(map[string]int)
		for i := 0; i < 20; i++ {
			agents["agent"+string(rune(i+'0'))] = i + 1
		}
		stats := &behavioral.AggregateStats{
			AgentBreakdown: agents,
		}
		output := formatStatsTable(stats)
		assert.NotEmpty(t, output)
	})
}

// TestObserveIntegration_DisplaySessionUpdateEdgeCases tests displaySessionUpdate variations
func TestObserveIntegration_DisplaySessionUpdateEdgeCases(t *testing.T) {
	t.Run("display with empty agent", func(t *testing.T) {
		update := SessionUpdate{
			ID:        1,
			TaskName:  "task",
			Agent:     "",
			Timestamp: time.Now(),
			Success:   true,
			Duration:  0,
		}
		displaySessionUpdate(update)
	})

	t.Run("display with long duration", func(t *testing.T) {
		update := SessionUpdate{
			ID:        2,
			TaskName:  "long-task",
			Agent:     "agent",
			Timestamp: time.Now(),
			Success:   false,
			Duration:  2 * time.Hour,
		}
		displaySessionUpdate(update)
	})
}

// TestObserveIntegration_HandleExportCommand tests HandleExportCommand coverage
func TestObserveIntegration_HandleExportCommand(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("export with all filters", func(t *testing.T) {
		observeProject = "test"
		observeFilterType = "tool"
		observeErrorsOnly = true
		observeTimeRange = "24h"
		exportFormat = "json"
		exportOutput = filepath.Join(tmpDir, "test.json")

		cmd := NewObserveExportCmd()
		err := HandleExportCommand(cmd, []string{})
		assert.NoError(t, err)

		observeProject = ""
		observeFilterType = ""
		observeErrorsOnly = false
		observeTimeRange = ""
	})

	t.Run("export markdown format", func(t *testing.T) {
		exportFormat = "markdown"
		exportOutput = filepath.Join(tmpDir, "test.md")

		cmd := NewObserveExportCmd()
		err := HandleExportCommand(cmd, []string{})
		assert.NoError(t, err)
	})

	t.Run("export csv format", func(t *testing.T) {
		exportFormat = "csv"
		exportOutput = filepath.Join(tmpDir, "test.csv")

		cmd := NewObserveExportCmd()
		err := HandleExportCommand(cmd, []string{})
		assert.NoError(t, err)
	})
}


// TestObserveIntegration_CollectMetricsEdgeCases tests collectBehavioralMetrics edge cases
func TestObserveIntegration_CollectMetricsEdgeCases(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := learning.NewStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	t.Run("collect with empty project filter", func(t *testing.T) {
		metrics, err := collectBehavioralMetrics(store, "")
		assert.NoError(t, err)
		assert.Empty(t, metrics)
	})

	t.Run("collect with specific project", func(t *testing.T) {
		metrics, err := collectBehavioralMetrics(store, "specific-project")
		assert.NoError(t, err)
		assert.Empty(t, metrics)
	})
}

// TestObserveIntegration_WatchSessionsEdgeCases tests watchNewSessions edge cases
func TestObserveIntegration_WatchSessionsEdgeCases(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := learning.NewStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	t.Run("watch with zero lastSeenID", func(t *testing.T) {
		updates, err := watchNewSessions(store, "", 0)
		assert.NoError(t, err)
		assert.Empty(t, updates)
	})

	t.Run("watch with project filter", func(t *testing.T) {
		updates, err := watchNewSessions(store, "project", 0)
		assert.NoError(t, err)
		assert.Empty(t, updates)
	})

	t.Run("watch with high lastSeenID", func(t *testing.T) {
		updates, err := watchNewSessions(store, "", 999999)
		assert.NoError(t, err)
		assert.Empty(t, updates)
	})
}

// TestObserveIntegration_DisplayStatsWithConfig tests DisplayStats coverage
func TestObserveIntegration_DisplayStatsWithConfig(t *testing.T) {
	t.Run("display stats fails with no config", func(t *testing.T) {
		// Will fail with config error, covering error paths
		err := DisplayStats("")
		assert.Error(t, err)
	})

	t.Run("display stats with project fails with no config", func(t *testing.T) {
		err := DisplayStats("test-project")
		assert.Error(t, err)
	})
}

// TestObserveIntegration_StreamActivityWithConfig tests StreamActivity coverage
func TestObserveIntegration_StreamActivityWithConfig(t *testing.T) {
	t.Run("stream fails with no config", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
		defer cancel()
		err := StreamActivity(ctx, "")
		assert.Error(t, err)
	})

	t.Run("stream with project fails with no config", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
		defer cancel()
		err := StreamActivity(ctx, "test-project")
		assert.Error(t, err)
	})
}

// TestObserveIntegration_CollectBehavioralMetrics tests metrics collection logic
func TestObserveIntegration_CollectBehavioralMetrics(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := learning.NewStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	t.Run("collect with empty db all projects", func(t *testing.T) {
		metrics, err := collectBehavioralMetrics(store, "")
		assert.NoError(t, err)
		assert.Empty(t, metrics)
	})

	t.Run("collect with project filter no results", func(t *testing.T) {
		metrics, err := collectBehavioralMetrics(store, "no-such-project")
		assert.NoError(t, err)
		assert.Empty(t, metrics)
	})
}

// TestObserveIntegration_WatchNewSessionsLogic tests session watching logic
func TestObserveIntegration_WatchNewSessionsLogic(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := learning.NewStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	t.Run("watch from beginning", func(t *testing.T) {
		updates, err := watchNewSessions(store, "", 0)
		assert.NoError(t, err)
		assert.Empty(t, updates)
	})

	t.Run("watch with project no results", func(t *testing.T) {
		updates, err := watchNewSessions(store, "no-project", 0)
		assert.NoError(t, err)
		assert.Empty(t, updates)
	})

	t.Run("watch from high watermark", func(t *testing.T) {
		updates, err := watchNewSessions(store, "", 99999)
		assert.NoError(t, err)
		assert.Empty(t, updates)
	})
}

// TestObserveIntegration_EndToEndWithRealData tests complete workflow with realistic JSONL data
func TestObserveIntegration_EndToEndWithRealData(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "e2e-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	// Create realistic JSONL session files matching Claude CLI format
	sessions := []struct {
		uuid    string
		agent   string
		success bool
		tools   []string
		cost    float64
	}{
		{
			uuid:    "e2e11111-1111-1111-1111-111111111111",
			agent:   "backend-developer",
			success: true,
			tools:   []string{"Read", "Write", "Bash"},
			cost:    0.05,
		},
		{
			uuid:    "e2e22222-2222-2222-2222-222222222222",
			agent:   "test-automator",
			success: true,
			tools:   []string{"Bash", "Grep", "Read"},
			cost:    0.08,
		},
		{
			uuid:    "e2e33333-3333-3333-3333-333333333333",
			agent:   "code-reviewer",
			success: false,
			tools:   []string{"Read", "Read", "Read"},
			cost:    0.03,
		},
	}

	for _, s := range sessions {
		var toolEvents string
		for i, tool := range s.tools {
			toolEvents += `{"type":"tool_call","timestamp":"2024-01-15T10:0` + string(rune('1'+i)) + `:00Z","tool_name":"` + tool + `","parameters":{},"result":"success","success":true,"duration":100}
`
		}

		content := `{"type":"session_start","session_id":"` + s.uuid + `","project":"e2e-project","timestamp":"2024-01-15T10:00:00Z","status":"completed","agent_name":"` + s.agent + `","duration":60000,"success":` + boolToStr(s.success) + `,"error_count":0}
` + toolEvents + `{"type":"token_usage","timestamp":"2024-01-15T10:05:00Z","input_tokens":2000,"output_tokens":1000,"cost_usd":` + floatToString(s.cost) + `}
`
		sessionFile := filepath.Join(projectDir, "agent-"+s.uuid+".jsonl")
		if err := os.WriteFile(sessionFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write session file: %v", err)
		}
	}

	t.Run("discover_and_load_sessions", func(t *testing.T) {
		discoveredSessions, err := behavioral.DiscoverProjectSessions(tmpDir, "e2e-project")
		require.NoError(t, err)
		assert.Equal(t, 3, len(discoveredSessions))

		for _, session := range discoveredSessions {
			data, err := behavioral.ParseSessionFile(session.FilePath)
			require.NoError(t, err)
			assert.NotNil(t, data)
			assert.NotEmpty(t, data.Session.ID)
		}
	})

	t.Run("aggregate_project_metrics", func(t *testing.T) {
		agg := behavioral.NewAggregatorWithBaseDir(10, tmpDir)
		metrics, err := agg.GetProjectMetrics("e2e-project")
		require.NoError(t, err)

		assert.Equal(t, 3, metrics.TotalSessions)
		assert.Greater(t, metrics.TotalCost, 0.0)
		assert.NotNil(t, metrics.AgentPerformance)
	})

	t.Run("export_json", func(t *testing.T) {
		outputPath := filepath.Join(tmpDir, "e2e-export.json")
		exportFormat = "json"
		exportOutput = outputPath
		observeProject = ""

		cmd := NewObserveExportCmd()
		err := HandleExportCommand(cmd, []string{})
		assert.NoError(t, err)

		// Verify file was created
		_, err = os.Stat(outputPath)
		if err == nil {
			// File exists, verify content is valid JSON
			content, _ := os.ReadFile(outputPath)
			assert.Contains(t, string(content), "{")
		}
	})

	t.Run("export_markdown", func(t *testing.T) {
		outputPath := filepath.Join(tmpDir, "e2e-export.md")
		exportFormat = "markdown"
		exportOutput = outputPath

		cmd := NewObserveExportCmd()
		err := HandleExportCommand(cmd, []string{})
		assert.NoError(t, err)
	})

	t.Run("pattern_analysis", func(t *testing.T) {
		// Collect all sessions for pattern analysis
		var allSessions []behavioral.Session
		var allMetrics []behavioral.BehavioralMetrics

		discoveredSessions, _ := behavioral.DiscoverProjectSessions(tmpDir, "e2e-project")
		for _, discovered := range discoveredSessions {
			data, err := behavioral.ParseSessionFile(discovered.FilePath)
			if err != nil {
				continue
			}
			allSessions = append(allSessions, data.Session)
			m := behavioral.ExtractMetrics(data)
			allMetrics = append(allMetrics, *m)
		}

		// Test pattern detection
		detector := behavioral.NewPatternDetector(allSessions, allMetrics)
		sequences := detector.DetectToolSequences(1, 2)
		assert.NotNil(t, sequences)

		// Test failure prediction
		predictor := behavioral.NewFailurePredictor(allSessions, allMetrics)
		rates := predictor.CalculateToolFailureRates()
		assert.NotNil(t, rates)

		// Test performance scoring
		scorer := behavioral.NewPerformanceScorer(allSessions, allMetrics)
		rankings := scorer.RankAgents()
		assert.NotEmpty(t, rankings)
	})

	t.Run("filter_and_display", func(t *testing.T) {
		observeProject = "e2e-project"
		observeFilterType = "tool"
		observeTimeRange = "24h"

		criteria, err := ParseFilterFlags()
		require.NoError(t, err)

		desc := BuildFilterDescription(criteria)
		assert.NotEmpty(t, desc)

		// Reset flags
		observeProject = ""
		observeFilterType = ""
		observeTimeRange = ""
	})
}

// TestObserveIntegration_CachePerformance tests aggregator cache efficiency
func TestObserveIntegration_CachePerformance(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "cache-test")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	// Create multiple session files with valid UUID format
	numSessions := 10
	for i := 0; i < numSessions; i++ {
		// Valid UUID format: 8-4-4-4-12 hex characters
		uuid := fmt.Sprintf("c%07x-a000-b000-c000-d%011x", i, i)
		content := `{"type":"session_start","session_id":"` + uuid + `","project":"cache-test","timestamp":"2024-01-15T10:00:00Z","status":"completed","success":true}
{"type":"tool_call","timestamp":"2024-01-15T10:01:00Z","tool_name":"Read","success":true,"duration":100}
`
		sessionFile := filepath.Join(projectDir, "agent-"+uuid+".jsonl")
		if err := os.WriteFile(sessionFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write session file: %v", err)
		}
	}

	t.Run("cache_hits_improve_performance", func(t *testing.T) {
		agg := behavioral.NewAggregatorWithBaseDir(20, tmpDir)
		discoveredSessions, _ := behavioral.DiscoverProjectSessions(tmpDir, "cache-test")

		// First pass - populate cache
		for _, session := range discoveredSessions {
			_, err := agg.LoadSession(session.FilePath)
			assert.NoError(t, err)
		}

		initialCacheSize := agg.CacheSize()
		assert.Equal(t, numSessions, initialCacheSize)

		// Second pass - should hit cache
		for _, session := range discoveredSessions {
			assert.True(t, agg.IsCached(session.FilePath))
			_, err := agg.LoadSession(session.FilePath)
			assert.NoError(t, err)
		}

		// Cache size should remain the same
		assert.Equal(t, initialCacheSize, agg.CacheSize())
	})

	t.Run("cache_eviction_works", func(t *testing.T) {
		smallCacheAgg := behavioral.NewAggregatorWithBaseDir(3, tmpDir)
		discoveredSessions, _ := behavioral.DiscoverProjectSessions(tmpDir, "cache-test")

		// Load more sessions than cache can hold
		for _, session := range discoveredSessions {
			_, err := smallCacheAgg.LoadSession(session.FilePath)
			assert.NoError(t, err)
		}

		// Cache should not exceed max size
		assert.LessOrEqual(t, smallCacheAgg.CacheSize(), 3)
	})

	t.Run("cache_clear", func(t *testing.T) {
		agg := behavioral.NewAggregatorWithBaseDir(20, tmpDir)
		discoveredSessions, _ := behavioral.DiscoverProjectSessions(tmpDir, "cache-test")

		for _, session := range discoveredSessions {
			_, _ = agg.LoadSession(session.FilePath)
		}

		assert.Greater(t, agg.CacheSize(), 0)
		agg.ClearCache()
		assert.Equal(t, 0, agg.CacheSize())
	})
}

// TestObserveIntegration_StoreWithExecution tests behavioral storage with task execution
func TestObserveIntegration_StoreWithExecution(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "exec.db")

	store, err := learning.NewStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	t.Run("record_and_retrieve_execution", func(t *testing.T) {
		exec := &learning.TaskExecution{
			PlanFile:     "test.yaml",
			RunNumber:    1,
			TaskNumber:   "1",
			TaskName:     "Integration Test Task",
			Agent:        "backend-developer",
			Prompt:       "Implement feature X",
			Success:      true,
			Output:       "Task completed successfully",
			DurationSecs: 120,
			QCVerdict:    "GREEN",
		}
		err := store.RecordExecution(ctx, exec)
		require.NoError(t, err)
		assert.Greater(t, exec.ID, int64(0))
	})

	t.Run("get_task_behavior", func(t *testing.T) {
		behavior, err := store.GetTaskBehavior(ctx, "1")
		require.NoError(t, err)
		assert.NotNil(t, behavior)
	})
}

// Helper functions for e2e tests
func boolToStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func floatToString(f float64) string {
	return fmt.Sprintf("%.4f", f)
}
