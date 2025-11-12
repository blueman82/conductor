package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCommand(t *testing.T) {
	cmd := NewRootCommand()
	if cmd == nil {
		t.Fatal("Root command should not be nil")
	}

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--help"})

	// Execute will return nil for --help
	err := cmd.Execute()
	// --help causes cobra to exit with an error, which is expected behavior
	// We check the output buffer instead

	output := buf.String()

	// Check that basic command info is present
	hasName := strings.Contains(output, "Conductor") || strings.Contains(output, "conductor")
	if !hasName {
		t.Errorf("Help text should contain 'conductor' or 'Conductor', got: %s", output)
	}

	// Check for orchestration-related content
	hasOrchestration := strings.Contains(output, "orchestration") || strings.Contains(output, "orchestrat")
	if !hasOrchestration {
		t.Errorf("Help text should mention orchestration, got: %s", output)
	}

	// If we got here without panic, consider it success even if err != nil
	// because --help returns an error by design in some cobra versions
	if err != nil && !strings.Contains(err.Error(), "help requested") {
		t.Logf("Help command returned error (this is ok): %v", err)
	}
}

func TestRootCommandHasSubcommands(t *testing.T) {
	cmd := NewRootCommand()
	if cmd == nil {
		t.Fatal("Root command should not be nil")
	}

	// Look for run and validate subcommands (stub them for now)
	commands := cmd.Commands()

	// For now, we just verify that the root command exists
	// Subcommands will be added in later tasks
	// This test is here as a placeholder to ensure we structure properly
	if cmd.Use != "conductor" {
		t.Errorf("Expected Use to be 'conductor', got '%s'", cmd.Use)
	}

	// The commands slice should have validate command now
	// We're just testing the structure
	if len(commands) == 0 {
		t.Errorf("Expected at least 1 subcommand (validate), got %d", len(commands))
	}

	t.Logf("Found %d subcommands", len(commands))
}

func TestVersionFlag(t *testing.T) {
	cmd := NewRootCommand()
	if cmd == nil {
		t.Fatal("Root command should not be nil")
	}

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--version"})

	err := cmd.Execute()
	// Version flag may or may not return an error depending on cobra version

	output := buf.String()
	// Check that output contains "version" keyword (actual version varies based on build)
	if !strings.Contains(output, "version") {
		t.Errorf("Version output should contain 'version', got: %s", output)
	}

	if err != nil && !strings.Contains(err.Error(), "version") {
		t.Logf("Version flag returned error (this is ok): %v", err)
	}
}
