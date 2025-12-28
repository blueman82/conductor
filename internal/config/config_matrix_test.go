package config

import (
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

// TestLoadConfig_FullMatrixCoversAllFields ensures that every configuration field
// can be overridden via YAML and that nested sections are respected.
func TestLoadConfig_FullMatrixCoversAllFields(t *testing.T) {
	clearConsoleEnv(t)

	cfg, err := LoadConfig(filepath.Join("testdata", "full-config.yaml"))
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	assertEqual(t, "MaxConcurrency", cfg.MaxConcurrency, 7)
	assertEqual(t, "Timeout", cfg.Timeout, 45*time.Minute)
	assertEqual(t, "LogLevel", cfg.LogLevel, "debug")
	assertEqual(t, "LogDir", cfg.LogDir, "/tmp/conductor/logs")
	assertEqual(t, "DryRun", cfg.DryRun, true)
	assertEqual(t, "SkipCompleted", cfg.SkipCompleted, true)
	assertEqual(t, "RetryFailed", cfg.RetryFailed, true)

	t.Run("Console", func(t *testing.T) {
		assertEqual(t, "EnableColor", cfg.Console.EnableColor, false)
		assertEqual(t, "EnableProgressBar", cfg.Console.EnableProgressBar, false)
		assertEqual(t, "EnableTaskDetails", cfg.Console.EnableTaskDetails, false)
		assertEqual(t, "EnableQCFeedback", cfg.Console.EnableQCFeedback, false)
		assertEqual(t, "CompactMode", cfg.Console.CompactMode, true)
		assertEqual(t, "ShowAgentNames", cfg.Console.ShowAgentNames, false)
		assertEqual(t, "ShowFileCounts", cfg.Console.ShowFileCounts, false)
		assertEqual(t, "ShowDurations", cfg.Console.ShowDurations, false)
	})

	t.Run("Feedback", func(t *testing.T) {
		assertEqual(t, "StoreInPlanFile", cfg.Feedback.StoreInPlanFile, false)
		assertEqual(t, "StoreInDatabase", cfg.Feedback.StoreInDatabase, false)
		assertEqual(t, "Format", cfg.Feedback.Format, "plain")
		assertEqual(t, "StoreOnGreen", cfg.Feedback.StoreOnGreen, false)
		assertEqual(t, "StoreOnRed", cfg.Feedback.StoreOnRed, true)
		assertEqual(t, "StoreOnYellow", cfg.Feedback.StoreOnYellow, false)
	})

	t.Run("Learning", func(t *testing.T) {
		assertEqual(t, "Enabled", cfg.Learning.Enabled, true)
		assertEqual(t, "DBPath", cfg.Learning.DBPath, "/tmp/learning.db")
		assertEqual(t, "SwapDuringRetries", cfg.Learning.SwapDuringRetries, false)
		assertEqual(t, "EnhancePrompts", cfg.Learning.EnhancePrompts, false)
		assertEqual(t, "QCReadsPlanContext", cfg.Learning.QCReadsPlanContext, false)
		assertEqual(t, "QCReadsDBContext", cfg.Learning.QCReadsDBContext, false)
		assertEqual(t, "MaxContextEntries", cfg.Learning.MaxContextEntries, 25)
		assertEqual(t, "KeepExecutionsDays", cfg.Learning.KeepExecutionsDays, 45)
		assertEqual(t, "MaxExecutionsPerTask", cfg.Learning.MaxExecutionsPerTask, 150)
	})

	t.Run("QualityControl", func(t *testing.T) {
		assertEqual(t, "Enabled", cfg.QualityControl.Enabled, true)
		assertEqual(t, "ReviewAgent", cfg.QualityControl.ReviewAgent, "custom-qc")
		assertEqual(t, "RetryOnRed", cfg.QualityControl.RetryOnRed, 3)

		assertEqual(t, "Agents.Mode", cfg.QualityControl.Agents.Mode, "explicit")
		assertDeepEqual(t, "Agents.ExplicitList", cfg.QualityControl.Agents.ExplicitList, []string{"qc-a", "qc-b"})
		assertDeepEqual(t, "Agents.Additional", cfg.QualityControl.Agents.AdditionalAgents, []string{"extra-agent"})
		assertDeepEqual(t, "Agents.Blocked", cfg.QualityControl.Agents.BlockedAgents, []string{"blocked-agent"})
	})
}

func clearConsoleEnv(t *testing.T) {
	t.Helper()
	envs := []string{
		"CONDUCTOR_CONSOLE_COLOR",
		"CONDUCTOR_CONSOLE_PROGRESS_BAR",
		"CONDUCTOR_CONSOLE_TASK_DETAILS",
		"CONDUCTOR_CONSOLE_QC_FEEDBACK",
		"CONDUCTOR_CONSOLE_COMPACT",
		"CONDUCTOR_CONSOLE_AGENT_NAMES",
		"CONDUCTOR_CONSOLE_FILE_COUNTS",
		"CONDUCTOR_CONSOLE_DURATIONS",
	}
	for _, key := range envs {
		t.Setenv(key, "")
	}
}

func assertEqual[T comparable](t *testing.T, field string, got, want T) {
	t.Helper()
	if got != want {
		t.Fatalf("%s = %v, want %v", field, got, want)
	}
}

func assertDeepEqual(t *testing.T, field string, got, want interface{}) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s mismatch: got %#v, want %#v", field, got, want)
	}
}
