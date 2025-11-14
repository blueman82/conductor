package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestConfigIntegrationWithRunCommand tests config loading in a realistic scenario
func TestConfigIntegrationWithRunCommand(t *testing.T) {
	// Create temporary project directory
	tmpDir := t.TempDir()

	// Create .conductor directory
	conductorDir := filepath.Join(tmpDir, ".conductor")
	if err := os.MkdirAll(conductorDir, 0755); err != nil {
		t.Fatalf("failed to create .conductor dir: %v", err)
	}

	// Write config file
	configPath := filepath.Join(conductorDir, "config.yaml")
	configContent := `max_concurrency: 8
timeout: 2h30m
log_level: debug
log_dir: .conductor/logs
dry_run: false
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Load config from directory
	cfg, err := LoadConfigFromDir(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfigFromDir() error = %v", err)
	}

	// Verify loaded values
	if cfg.MaxConcurrency != 8 {
		t.Errorf("MaxConcurrency = %d, want 8", cfg.MaxConcurrency)
	}
	if cfg.Timeout != 2*time.Hour+30*time.Minute {
		t.Errorf("Timeout = %v, want 2h30m", cfg.Timeout)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "debug")
	}

	// Simulate CLI flag override
	maxConcurrency := 12
	timeout := 5 * time.Hour

	cfg.MergeWithFlags(&maxConcurrency, &timeout, nil, nil, nil, nil)

	// Verify flags override config
	if cfg.MaxConcurrency != 12 {
		t.Errorf("After merge: MaxConcurrency = %d, want 12", cfg.MaxConcurrency)
	}
	if cfg.Timeout != 5*time.Hour {
		t.Errorf("After merge: Timeout = %v, want 5h", cfg.Timeout)
	}

	// Verify non-overridden values preserved
	if cfg.LogLevel != "debug" {
		t.Errorf("After merge: LogLevel = %q, want %q (preserved)", cfg.LogLevel, "debug")
	}

	// Validate merged config
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() after merge error = %v", err)
	}
}

// TestConfigWithExampleFile tests loading the actual example config file
func TestConfigWithExampleFile(t *testing.T) {
	// Try to load the example config from project root
	examplePath := filepath.Join("..", "..", ".conductor", "config.yaml.example")

	// Check if example file exists
	if _, err := os.Stat(examplePath); os.IsNotExist(err) {
		t.Skip("Example config file not found, skipping")
	}

	// Load example config (will fall back to defaults since .example extension)
	cfg, err := LoadConfig(examplePath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Should return defaults since example file may not be valid YAML
	if cfg == nil {
		t.Fatal("LoadConfig() returned nil")
	}

	// Validate the config
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() error = %v", err)
	}
}

// TestConfigDefaultsWhenNoFileExists tests default behavior
func TestConfigDefaultsWhenNoFileExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Load config from empty directory
	cfg, err := LoadConfigFromDir(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfigFromDir() error = %v", err)
	}

	// Should return defaults
	defaults := DefaultConfig()

	if cfg.MaxConcurrency != defaults.MaxConcurrency {
		t.Errorf("MaxConcurrency = %d, want %d (default)", cfg.MaxConcurrency, defaults.MaxConcurrency)
	}
	if cfg.Timeout != defaults.Timeout {
		t.Errorf("Timeout = %v, want %v (default)", cfg.Timeout, defaults.Timeout)
	}
	if cfg.LogLevel != defaults.LogLevel {
		t.Errorf("LogLevel = %q, want %q (default)", cfg.LogLevel, defaults.LogLevel)
	}
	if cfg.LogDir != defaults.LogDir {
		t.Errorf("LogDir = %q, want %q (default)", cfg.LogDir, defaults.LogDir)
	}
	if cfg.DryRun != defaults.DryRun {
		t.Errorf("DryRun = %v, want %v (default)", cfg.DryRun, defaults.DryRun)
	}
}

// TestCompleteWorkflow tests the complete config workflow
func TestCompleteWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".conductor")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	// Write initial config
	configPath := filepath.Join(configDir, "config.yaml")
	initialConfig := `max_concurrency: 4
timeout: 1h
log_level: info
`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Load config using testable version with explicit root
	cfg, err := LoadConfigFromRootWithBuildTime(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfigFromDir() error = %v", err)
	}

	// Verify loaded
	if cfg.MaxConcurrency != 4 {
		t.Errorf("Initial MaxConcurrency = %d, want 4", cfg.MaxConcurrency)
	}

	// Simulate command: conductor run --max-concurrency 10 --dry-run plan.md
	maxConcurrency := 10
	dryRun := true

	cfg.MergeWithFlags(&maxConcurrency, nil, nil, &dryRun, nil, nil)

	// Verify merge
	if cfg.MaxConcurrency != 10 {
		t.Errorf("MaxConcurrency after merge = %d, want 10", cfg.MaxConcurrency)
	}
	if cfg.DryRun != true {
		t.Errorf("DryRun after merge = %v, want true", cfg.DryRun)
	}
	if cfg.Timeout != 1*time.Hour {
		t.Errorf("Timeout after merge = %v, want 1h (from config)", cfg.Timeout)
	}

	// Validate
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() error = %v", err)
	}
}
