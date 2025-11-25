package cmd

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestObserveCommand_Registered(t *testing.T) {
	rootCmd := NewRootCommand()
	if rootCmd == nil {
		t.Fatal("Root command should not be nil")
	}

	// Find the observe command
	observeCmd := findCommand(rootCmd, "observe")
	if observeCmd == nil {
		t.Fatal("Observe command should be registered with root command")
	}

	// Verify basic properties
	if observeCmd.Use != "observe" {
		t.Errorf("Expected Use to be 'observe', got '%s'", observeCmd.Use)
	}

	if observeCmd.Short == "" {
		t.Error("Observe command should have Short description")
	}

	if observeCmd.Long == "" {
		t.Error("Observe command should have Long description")
	}
}

func TestObserveCommand_Subcommands(t *testing.T) {
	rootCmd := NewRootCommand()
	if rootCmd == nil {
		t.Fatal("Root command should not be nil")
	}

	observeCmd := findCommand(rootCmd, "observe")
	if observeCmd == nil {
		t.Fatal("Observe command should be registered")
	}

	// Verify all 11 subcommands are registered
	subcommands := observeCmd.Commands()
	if len(subcommands) != 11 {
		t.Errorf("Expected 11 subcommands, got %d", len(subcommands))
	}

	// Verify specific subcommands exist
	expectedSubcommands := []string{"import", "ingest", "project", "session", "tools", "bash", "files", "errors", "stats", "stream", "export"}
	for _, expectedName := range expectedSubcommands {
		found := false
		for _, subcmd := range subcommands {
			if subcmd.Name() == expectedName {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected subcommand '%s' not found", expectedName)
		}
	}
}

func TestObserveCommand_HelpText(t *testing.T) {
	rootCmd := NewRootCommand()
	if rootCmd == nil {
		t.Fatal("Root command should not be nil")
	}

	observeCmd := findCommand(rootCmd, "observe")
	if observeCmd == nil {
		t.Fatal("Observe command should be registered")
	}

	// Verify help text mentions observability or agent watch
	longLower := strings.ToLower(observeCmd.Long)
	if !strings.Contains(longLower, "observ") && !strings.Contains(longLower, "agent") {
		t.Error("Observe command Long description should mention 'observ' or 'agent'")
	}

	// Test help output
	testRootCmd := NewRootCommand()
	buf := new(bytes.Buffer)
	testRootCmd.SetOut(buf)
	testRootCmd.SetErr(buf)
	testRootCmd.SetArgs([]string{"observe", "--help"})

	_ = testRootCmd.Execute()
	output := buf.String()

	// Verify help output contains subcommand names
	for _, subcmd := range []string{"project", "session", "tools", "bash", "files", "errors"} {
		if !strings.Contains(output, subcmd) {
			t.Errorf("Help output should mention '%s' subcommand, got: %s", subcmd, output)
		}
	}
}

func TestObserveCommand_GlobalFlags(t *testing.T) {
	rootCmd := NewRootCommand()
	if rootCmd == nil {
		t.Fatal("Root command should not be nil")
	}

	observeCmd := findCommand(rootCmd, "observe")
	if observeCmd == nil {
		t.Fatal("Observe command should be registered")
	}

	// Test that global flags are registered
	expectedFlags := []string{"project", "session", "filter-type", "errors-only", "time-range"}
	for _, flagName := range expectedFlags {
		flag := observeCmd.PersistentFlags().Lookup(flagName)
		if flag == nil {
			t.Errorf("Expected persistent flag '%s' not found", flagName)
		}
	}
}

func TestObserveCommand_InteractiveMode(t *testing.T) {
	// Note: Interactive mode reads from stdin, so we need to provide project flag
	// to bypass the interactive menu, or it will fail with EOF
	rootCmd := NewRootCommand()
	if rootCmd == nil {
		t.Fatal("Root command should not be nil")
	}

	// Test running observe with project flag (bypasses interactive menu)
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"observe", "--project", "test-project"})

	err := rootCmd.Execute()
	if err != nil {
		// May fail if project doesn't exist, but command should be properly structured
		// Just verify it doesn't panic
		t.Logf("Command executed with expected behavior: %v", err)
	}

	// Verify command executes without panic
}

func TestObserveProjectCmd_Help(t *testing.T) {
	rootCmd := NewRootCommand()
	if rootCmd == nil {
		t.Fatal("Root command should not be nil")
	}

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"observe", "project", "--help"})

	_ = rootCmd.Execute()
	output := buf.String()

	// Verify help text mentions project-level metrics
	if !strings.Contains(output, "project") {
		t.Errorf("Project command help should mention 'project', got: %s", output)
	}
}

func TestObserveSessionCmd_Help(t *testing.T) {
	rootCmd := NewRootCommand()
	if rootCmd == nil {
		t.Fatal("Root command should not be nil")
	}

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"observe", "session", "--help"})

	_ = rootCmd.Execute()
	output := buf.String()

	// Verify help text mentions session analysis
	if !strings.Contains(output, "session") {
		t.Errorf("Session command help should mention 'session', got: %s", output)
	}
}

func TestObserveToolsCmd_Help(t *testing.T) {
	rootCmd := NewRootCommand()
	if rootCmd == nil {
		t.Fatal("Root command should not be nil")
	}

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"observe", "tools", "--help"})

	_ = rootCmd.Execute()
	output := buf.String()

	// Verify help text mentions tool usage
	if !strings.Contains(output, "tool") {
		t.Errorf("Tools command help should mention 'tool', got: %s", output)
	}
}

func TestObserveStreamCmd_WithIngestFlag(t *testing.T) {
	rootCmd := NewRootCommand()
	if rootCmd == nil {
		t.Fatal("Root command should not be nil")
	}

	observeCmd := findCommand(rootCmd, "observe")
	if observeCmd == nil {
		t.Fatal("Observe command should be registered")
	}

	streamCmd := findCommand(observeCmd, "stream")
	if streamCmd == nil {
		t.Fatal("Stream subcommand should be registered")
	}

	// Verify --with-ingest flag is registered
	flag := streamCmd.Flags().Lookup("with-ingest")
	if flag == nil {
		t.Fatal("Expected --with-ingest flag not found")
	}

	// Verify flag properties
	if flag.DefValue != "false" {
		t.Errorf("Expected --with-ingest default to be 'false', got '%s'", flag.DefValue)
	}

	if flag.Usage == "" {
		t.Error("Expected --with-ingest flag to have usage description")
	}

	// Verify help text mentions --with-ingest
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"observe", "stream", "--help"})

	_ = rootCmd.Execute()
	output := buf.String()

	if !strings.Contains(output, "with-ingest") {
		t.Errorf("Stream command help should mention 'with-ingest', got: %s", output)
	}

	if !strings.Contains(output, "ingestion daemon") {
		t.Errorf("Stream command help should mention 'ingestion daemon', got: %s", output)
	}
}

func TestInteractiveSessionSelection_QuitImmediately(t *testing.T) {
	// Mock reader that returns 'q' immediately
	reader := &MockMenuReader{inputs: []string{"q\n"}}

	err := runInteractiveSessionSelectionWithReader("nonexistent-project", reader)
	// Should exit gracefully (no error for quit)
	if err != nil && !strings.Contains(err.Error(), "failed to list sessions") {
		t.Errorf("Expected no error or 'failed to list sessions', got: %v", err)
	}
}

func TestInteractiveSessionSelection_InvalidInput(t *testing.T) {
	// Mock reader that returns invalid input, then 'q'
	reader := &MockMenuReader{inputs: []string{"invalid\n", "q\n"}}

	err := runInteractiveSessionSelectionWithReader("nonexistent-project", reader)
	// Should handle gracefully
	if err != nil && !strings.Contains(err.Error(), "failed to list sessions") {
		t.Errorf("Expected no error or 'failed to list sessions', got: %v", err)
	}
}

func TestFormatTimeAgo(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		time     time.Time
		contains string
	}{
		{"just_now", now.Add(-30 * time.Second), "just now"},
		{"minutes_ago", now.Add(-5 * time.Minute), "m ago"},
		{"hours_ago", now.Add(-3 * time.Hour), "h ago"},
		{"days_ago", now.Add(-2 * 24 * time.Hour), "d ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatTimeAgo(tt.time)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("formatTimeAgo(%v) = %q, expected to contain %q", tt.time, result, tt.contains)
			}
		})
	}
}

func TestEstimateDuration(t *testing.T) {
	tests := []struct {
		name     string
		fileSize int64
		expected string
	}{
		{"small_file", 30 * 1024, "30s"},
		{"medium_file", 90 * 1024, "1m 30s"},
		{"large_file", 3700 * 1024, "1h 1m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := estimateDuration(tt.fileSize)
			if result != tt.expected {
				t.Errorf("estimateDuration(%d) = %q, expected %q", tt.fileSize, result, tt.expected)
			}
		})
	}
}
