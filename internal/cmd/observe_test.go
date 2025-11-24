package cmd

import (
	"bytes"
	"strings"
	"testing"
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

	// Verify all 6 subcommands are registered
	subcommands := observeCmd.Commands()
	if len(subcommands) != 6 {
		t.Errorf("Expected 6 subcommands, got %d", len(subcommands))
	}

	// Verify specific subcommands exist
	expectedSubcommands := []string{"project", "session", "tools", "bash", "files", "errors"}
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
	// Note: Interactive mode uses fmt.Println which goes to stdout
	// In real execution, observeInteractive returns nil without error
	// We test that it executes without error, not the output
	rootCmd := NewRootCommand()
	if rootCmd == nil {
		t.Fatal("Root command should not be nil")
	}

	// Test running observe without subcommand (interactive mode)
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"observe"})

	err := rootCmd.Execute()
	if err != nil {
		t.Errorf("Expected no error from interactive mode, got: %v", err)
	}

	// Interactive mode should return nil (success)
	// Output verification would require capturing stdout which is out of scope for this test
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
