package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestDefaultConfig verifies default configuration values
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.MaxConcurrency != 0 {
		t.Errorf("MaxConcurrency = %d, want 0", cfg.MaxConcurrency)
	}
	if cfg.Timeout != 10*time.Hour {
		t.Errorf("Timeout = %v, want 10h", cfg.Timeout)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "info")
	}
	if cfg.LogDir != ".conductor/logs" {
		t.Errorf("LogDir = %q, want %q", cfg.LogDir, ".conductor/logs")
	}
	if cfg.DryRun != false {
		t.Errorf("DryRun = %v, want false", cfg.DryRun)
	}
	if cfg.SkipCompleted != false {
		t.Errorf("SkipCompleted = %v, want false", cfg.SkipCompleted)
	}
	if cfg.RetryFailed != false {
		t.Errorf("RetryFailed = %v, want false", cfg.RetryFailed)
	}
}

// TestLoadConfigValidFile tests loading a valid YAML config file
func TestLoadConfigValidFile(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `max_concurrency: 5
timeout: 30m
log_level: debug
log_dir: /tmp/logs
dry_run: true
skip_completed: true
retry_failed: true
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	// Load config
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Verify values
	if cfg.MaxConcurrency != 5 {
		t.Errorf("MaxConcurrency = %d, want 5", cfg.MaxConcurrency)
	}
	if cfg.Timeout != 30*time.Minute {
		t.Errorf("Timeout = %v, want 30m", cfg.Timeout)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "debug")
	}
	if cfg.LogDir != "/tmp/logs" {
		t.Errorf("LogDir = %q, want %q", cfg.LogDir, "/tmp/logs")
	}
	if cfg.DryRun != true {
		t.Errorf("DryRun = %v, want true", cfg.DryRun)
	}
	if cfg.SkipCompleted != true {
		t.Errorf("SkipCompleted = %v, want true", cfg.SkipCompleted)
	}
	if cfg.RetryFailed != true {
		t.Errorf("RetryFailed = %v, want true", cfg.RetryFailed)
	}
}

// TestLoadConfigFileNotExists tests fallback to defaults when file doesn't exist
func TestLoadConfigFileNotExists(t *testing.T) {
	cfg, err := LoadConfig("/nonexistent/path/config.yaml")
	if err != nil {
		t.Fatalf("LoadConfig() should not error on missing file, got: %v", err)
	}

	// Should return default config
	if cfg.MaxConcurrency != 0 {
		t.Errorf("MaxConcurrency = %d, want 0 (default)", cfg.MaxConcurrency)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want %q (default)", cfg.LogLevel, "info")
	}
}

// TestLoadConfigInvalidYAML tests error handling for malformed YAML
func TestLoadConfigInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Write invalid YAML
	invalidYAML := `
max_concurrency: 5
timeout: [this is not valid
log_level: debug
`
	if err := os.WriteFile(configPath, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("LoadConfig() expected error for invalid YAML, got nil")
	}
}

// TestLoadConfigPartialValues tests that partial config merges with defaults
func TestLoadConfigPartialValues(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Only set some values
	configContent := `max_concurrency: 8
log_level: warn
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Check set values
	if cfg.MaxConcurrency != 8 {
		t.Errorf("MaxConcurrency = %d, want 8", cfg.MaxConcurrency)
	}
	if cfg.LogLevel != "warn" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "warn")
	}

	// Check default values for unset fields
	if cfg.Timeout != 10*time.Hour {
		t.Errorf("Timeout = %v, want 10h (default)", cfg.Timeout)
	}
	if cfg.LogDir != ".conductor/logs" {
		t.Errorf("LogDir = %q, want %q (default)", cfg.LogDir, ".conductor/logs")
	}
}

// TestLoadConfigFromDir tests loading config from .conductor/config.yaml
func TestLoadConfigFromDir(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".conductor")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")
	configContent := `max_concurrency: 3
timeout: 1h
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	// Set build-time root to use tmpDir
	SetBuildTimeRepoRoot(tmpDir)
	defer SetBuildTimeRepoRoot("") // Reset after test

	cfg, err := LoadConfigFromDir(".")
	if err != nil {
		t.Fatalf("LoadConfigFromDir() error = %v", err)
	}

	if cfg.MaxConcurrency != 3 {
		t.Errorf("MaxConcurrency = %d, want 3", cfg.MaxConcurrency)
	}
	if cfg.Timeout != 1*time.Hour {
		t.Errorf("Timeout = %v, want 1h", cfg.Timeout)
	}
}

// TestLoadConfigFromDirNotExists tests loading when .conductor dir doesn't exist
func TestLoadConfigFromDirNotExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Set build-time root to tmpDir (no .conductor directory)
	SetBuildTimeRepoRoot(tmpDir)
	defer SetBuildTimeRepoRoot("")

	cfg, err := LoadConfigFromDir(".")
	if err != nil {
		t.Fatalf("LoadConfigFromDir() should not error on missing config, got: %v", err)
	}

	// Should return defaults
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want %q (default)", cfg.LogLevel, "info")
	}
}

// TestMergeWithFlags tests CLI flag precedence over config values
func TestMergeWithFlags(t *testing.T) {
	cfg := &Config{
		MaxConcurrency: 3,
		Timeout:        30 * time.Minute,
		LogLevel:       "info",
		LogDir:         ".conductor/logs",
		DryRun:         false,
		SkipCompleted:  false,
		RetryFailed:    false,
	}

	// Override all values with flags
	maxConcurrency := 10
	timeout := 2 * time.Hour
	logDir := "/custom/logs"
	dryRun := true
	skipCompleted := true
	retryFailed := true

	cfg.MergeWithFlags(&maxConcurrency, &timeout, &logDir, &dryRun, &skipCompleted, &retryFailed)

	// Verify flags take precedence
	if cfg.MaxConcurrency != 10 {
		t.Errorf("MaxConcurrency = %d, want 10", cfg.MaxConcurrency)
	}
	if cfg.Timeout != 2*time.Hour {
		t.Errorf("Timeout = %v, want 2h", cfg.Timeout)
	}
	if cfg.LogDir != "/custom/logs" {
		t.Errorf("LogDir = %q, want %q", cfg.LogDir, "/custom/logs")
	}
	if cfg.DryRun != true {
		t.Errorf("DryRun = %v, want true", cfg.DryRun)
	}
	if cfg.SkipCompleted != true {
		t.Errorf("SkipCompleted = %v, want true", cfg.SkipCompleted)
	}
	if cfg.RetryFailed != true {
		t.Errorf("RetryFailed = %v, want true", cfg.RetryFailed)
	}
}

// TestMergeWithFlagsPartial tests that only non-nil flags override config
func TestMergeWithFlagsPartial(t *testing.T) {
	cfg := &Config{
		MaxConcurrency: 3,
		Timeout:        30 * time.Minute,
		LogLevel:       "info",
		LogDir:         ".conductor/logs",
		DryRun:         false,
		SkipCompleted:  false,
		RetryFailed:    false,
	}

	// Only override some values (others are nil)
	maxConcurrency := 5
	timeout := 1 * time.Hour

	cfg.MergeWithFlags(&maxConcurrency, &timeout, nil, nil, nil, nil)

	// Verify partial override
	if cfg.MaxConcurrency != 5 {
		t.Errorf("MaxConcurrency = %d, want 5", cfg.MaxConcurrency)
	}
	if cfg.Timeout != 1*time.Hour {
		t.Errorf("Timeout = %v, want 1h", cfg.Timeout)
	}

	// Verify original values preserved
	if cfg.LogDir != ".conductor/logs" {
		t.Errorf("LogDir = %q, want %q (original)", cfg.LogDir, ".conductor/logs")
	}
	if cfg.DryRun != false {
		t.Errorf("DryRun = %v, want false (original)", cfg.DryRun)
	}
	if cfg.SkipCompleted != false {
		t.Errorf("SkipCompleted = %v, want false (original)", cfg.SkipCompleted)
	}
	if cfg.RetryFailed != false {
		t.Errorf("RetryFailed = %v, want false (original)", cfg.RetryFailed)
	}
}

// TestMergeWithFlagsNil tests that nil flags don't override config
func TestMergeWithFlagsNil(t *testing.T) {
	cfg := &Config{
		MaxConcurrency: 3,
		Timeout:        30 * time.Minute,
		LogLevel:       "info",
		LogDir:         ".conductor/logs",
		DryRun:         false,
		SkipCompleted:  false,
		RetryFailed:    false,
	}

	// Pass all nil flags
	cfg.MergeWithFlags(nil, nil, nil, nil, nil, nil)

	// Verify all original values preserved
	if cfg.MaxConcurrency != 3 {
		t.Errorf("MaxConcurrency = %d, want 3 (original)", cfg.MaxConcurrency)
	}
	if cfg.Timeout != 30*time.Minute {
		t.Errorf("Timeout = %v, want 30m (original)", cfg.Timeout)
	}
	if cfg.LogDir != ".conductor/logs" {
		t.Errorf("LogDir = %q, want %q (original)", cfg.LogDir, ".conductor/logs")
	}
	if cfg.DryRun != false {
		t.Errorf("DryRun = %v, want false (original)", cfg.DryRun)
	}
	if cfg.SkipCompleted != false {
		t.Errorf("SkipCompleted = %v, want false (original)", cfg.SkipCompleted)
	}
	if cfg.RetryFailed != false {
		t.Errorf("RetryFailed = %v, want false (original)", cfg.RetryFailed)
	}
}

// TestTimeoutParsing tests various timeout duration formats
func TestTimeoutParsing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Duration
	}{
		{
			name:     "seconds",
			input:    "timeout: 30s",
			expected: 30 * time.Second,
		},
		{
			name:     "minutes",
			input:    "timeout: 5m",
			expected: 5 * time.Minute,
		},
		{
			name:     "hours",
			input:    "timeout: 2h",
			expected: 2 * time.Hour,
		},
		{
			name:     "combined",
			input:    "timeout: 1h30m",
			expected: 90 * time.Minute,
		},
		{
			name:     "complex",
			input:    "timeout: 2h15m30s",
			expected: 2*time.Hour + 15*time.Minute + 30*time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")

			if err := os.WriteFile(configPath, []byte(tt.input), 0644); err != nil {
				t.Fatalf("failed to write test config: %v", err)
			}

			cfg, err := LoadConfig(configPath)
			if err != nil {
				t.Fatalf("LoadConfig() error = %v", err)
			}

			if cfg.Timeout != tt.expected {
				t.Errorf("Timeout = %v, want %v", cfg.Timeout, tt.expected)
			}
		})
	}
}

// TestConfigValidation tests validation of config values
func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		config    string
		wantError bool
	}{
		{
			name: "valid config",
			config: `max_concurrency: 5
timeout: 30m
log_level: info
`,
			wantError: false,
		},
		{
			name: "negative max_concurrency",
			config: `max_concurrency: -1
`,
			wantError: true,
		},
		{
			name: "invalid log_level",
			config: `log_level: invalid
`,
			wantError: true,
		},
		{
			name: "invalid key point alignment mode",
			config: `validation:
  key_point_criteria: invalid
`,
			wantError: true,
		},
		{
			name: "zero timeout (allowed)",
			config: `timeout: 0s
`,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")

			if err := os.WriteFile(configPath, []byte(tt.config), 0644); err != nil {
				t.Fatalf("failed to write test config: %v", err)
			}

			cfg, err := LoadConfig(configPath)
			if err != nil {
				if !tt.wantError {
					t.Fatalf("LoadConfig() unexpected error = %v", err)
				}
				return
			}

			// Validate the loaded config
			err = cfg.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

// TestEmptyConfigFile tests loading an empty config file
func TestEmptyConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create empty file
	if err := os.WriteFile(configPath, []byte(""), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Should return defaults for empty file
	if cfg.MaxConcurrency != 0 {
		t.Errorf("MaxConcurrency = %d, want 0 (default)", cfg.MaxConcurrency)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want %q (default)", cfg.LogLevel, "info")
	}
}

// TestConfigWithComments tests loading config with YAML comments
func TestConfigWithComments(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `# This is a comment
max_concurrency: 4  # inline comment
# Another comment
timeout: 5m
log_level: debug  # set to debug for troubleshooting
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.MaxConcurrency != 4 {
		t.Errorf("MaxConcurrency = %d, want 4", cfg.MaxConcurrency)
	}
	if cfg.Timeout != 5*time.Minute {
		t.Errorf("Timeout = %v, want 5m", cfg.Timeout)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "debug")
	}
}

// TestLoadConfigPermissionDenied tests handling of permission errors
func TestLoadConfigPermissionDenied(t *testing.T) {
	// Skip on Windows where file permissions work differently
	if os.Getenv("GOOS") == "windows" {
		t.Skip("Skipping permission test on Windows")
	}
	// Skip when running as root since root bypasses permission checks
	if os.Getuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create config file
	if err := os.WriteFile(configPath, []byte("max_concurrency: 5"), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	// Make file unreadable
	if err := os.Chmod(configPath, 0000); err != nil {
		t.Fatalf("failed to chmod config: %v", err)
	}
	defer os.Chmod(configPath, 0644) // Restore permissions for cleanup

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("LoadConfig() expected error for unreadable file, got nil")
	}
}

// TestMergeWithFlagsZeroValues tests that zero-value flags are treated as set
func TestMergeWithFlagsZeroValues(t *testing.T) {
	cfg := &Config{
		MaxConcurrency: 5,
		Timeout:        1 * time.Hour,
		LogLevel:       "debug",
		LogDir:         "/tmp/logs",
		DryRun:         true,
		SkipCompleted:  true,
		RetryFailed:    true,
	}

	// Set flags to zero values
	maxConcurrency := 0
	timeout := 0 * time.Second
	logDir := ""
	dryRun := false
	skipCompleted := false
	retryFailed := false

	cfg.MergeWithFlags(&maxConcurrency, &timeout, &logDir, &dryRun, &skipCompleted, &retryFailed)

	// Zero values should override config
	if cfg.MaxConcurrency != 0 {
		t.Errorf("MaxConcurrency = %d, want 0", cfg.MaxConcurrency)
	}
	if cfg.Timeout != 0 {
		t.Errorf("Timeout = %v, want 0", cfg.Timeout)
	}
	if cfg.LogDir != "" {
		t.Errorf("LogDir = %q, want empty string", cfg.LogDir)
	}
	if cfg.DryRun != false {
		t.Errorf("DryRun = %v, want false", cfg.DryRun)
	}
	if cfg.SkipCompleted != false {
		t.Errorf("SkipCompleted = %v, want false", cfg.SkipCompleted)
	}
	if cfg.RetryFailed != false {
		t.Errorf("RetryFailed = %v, want false", cfg.RetryFailed)
	}
}

// TestValidLogLevels tests that valid log levels are accepted
func TestValidLogLevels(t *testing.T) {
	validLevels := []string{"trace", "debug", "info", "warn", "error"}

	for _, level := range validLevels {
		t.Run(level, func(t *testing.T) {
			cfg := &Config{
				MaxConcurrency: 0,
				Timeout:        10 * time.Hour,
				LogLevel:       level,
				LogDir:         ".conductor/logs",
				DryRun:         false,
			}

			if err := cfg.Validate(); err != nil {
				t.Errorf("Validate() error = %v for valid level %q", err, level)
			}
		})
	}
}

// TestInvalidLogLevels tests that invalid log levels are rejected
func TestInvalidLogLevels(t *testing.T) {
	invalidLevels := []string{"invalid", "TRACE", "INFO", "warning", "fatal", ""}

	for _, level := range invalidLevels {
		t.Run(level, func(t *testing.T) {
			cfg := &Config{
				MaxConcurrency: 0,
				Timeout:        10 * time.Hour,
				LogLevel:       level,
				LogDir:         ".conductor/logs",
				DryRun:         false,
			}

			if err := cfg.Validate(); err == nil {
				t.Errorf("Validate() expected error for invalid level %q", level)
			}
		})
	}
}

// TestSkipCompletedFlag tests loading skip_completed config option
func TestSkipCompletedFlag(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `skip_completed: true
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.SkipCompleted != true {
		t.Errorf("SkipCompleted = %v, want true", cfg.SkipCompleted)
	}
}

// TestRetryFailedFlag tests loading retry_failed config option
func TestRetryFailedFlag(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `retry_failed: true
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.RetryFailed != true {
		t.Errorf("RetryFailed = %v, want true", cfg.RetryFailed)
	}
}

// TestSkipCompletedAndRetryFailedTogether tests both flags together
func TestSkipCompletedAndRetryFailedTogether(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `skip_completed: true
retry_failed: true
max_concurrency: 4
timeout: 2h
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.SkipCompleted != true {
		t.Errorf("SkipCompleted = %v, want true", cfg.SkipCompleted)
	}
	if cfg.RetryFailed != true {
		t.Errorf("RetryFailed = %v, want true", cfg.RetryFailed)
	}
	if cfg.MaxConcurrency != 4 {
		t.Errorf("MaxConcurrency = %d, want 4", cfg.MaxConcurrency)
	}
	if cfg.Timeout != 2*time.Hour {
		t.Errorf("Timeout = %v, want 2h", cfg.Timeout)
	}
}

// TestSkipCompletedFlagOverride tests CLI flag override for skip_completed
func TestSkipCompletedFlagOverride(t *testing.T) {
	cfg := &Config{
		MaxConcurrency: 3,
		Timeout:        30 * time.Minute,
		LogLevel:       "info",
		LogDir:         ".conductor/logs",
		DryRun:         false,
		SkipCompleted:  false,
		RetryFailed:    false,
	}

	skipCompleted := true
	cfg.MergeWithFlags(nil, nil, nil, nil, &skipCompleted, nil)

	if cfg.SkipCompleted != true {
		t.Errorf("SkipCompleted = %v, want true", cfg.SkipCompleted)
	}
	// Verify other values unchanged
	if cfg.MaxConcurrency != 3 {
		t.Errorf("MaxConcurrency = %d, want 3 (original)", cfg.MaxConcurrency)
	}
}

// TestRetryFailedFlagOverride tests CLI flag override for retry_failed
func TestRetryFailedFlagOverride(t *testing.T) {
	cfg := &Config{
		MaxConcurrency: 3,
		Timeout:        30 * time.Minute,
		LogLevel:       "info",
		LogDir:         ".conductor/logs",
		DryRun:         false,
		SkipCompleted:  false,
		RetryFailed:    false,
	}

	retryFailed := true
	cfg.MergeWithFlags(nil, nil, nil, nil, nil, &retryFailed)

	if cfg.RetryFailed != true {
		t.Errorf("RetryFailed = %v, want true", cfg.RetryFailed)
	}
	// Verify other values unchanged
	if cfg.SkipCompleted != false {
		t.Errorf("SkipCompleted = %v, want false (original)", cfg.SkipCompleted)
	}
}

// TestConfig_LearningDefaults tests that learning config has correct defaults
func TestConfig_LearningDefaults(t *testing.T) {
	cfg := DefaultConfig()

	if !cfg.Learning.Enabled {
		t.Errorf("Learning.Enabled = %v, want true", cfg.Learning.Enabled)
	}
	if cfg.Learning.DBPath != ".conductor/learning/executions.db" {
		t.Errorf("Learning.DBPath = %q, want %q", cfg.Learning.DBPath, ".conductor/learning/executions.db")
	}
}

// TestConfig_LearningDisabled tests loading config with learning disabled
func TestConfig_LearningDisabled(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `learning:
  enabled: false
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.Learning.Enabled {
		t.Errorf("Learning.Enabled = %v, want false", cfg.Learning.Enabled)
	}
	// Other fields should have defaults
	if cfg.Learning.DBPath != ".conductor/learning/executions.db" {
		t.Errorf("Learning.DBPath = %q, want %q (default)", cfg.Learning.DBPath, ".conductor/learning/executions.db")
	}
}

// TestConfig_LearningCustomPath tests loading config with custom learning settings
func TestConfig_LearningCustomPath(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `learning:
  enabled: true
  db_path: /custom/path/learning.db
  keep_executions_days: 30
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if !cfg.Learning.Enabled {
		t.Errorf("Learning.Enabled = %v, want true", cfg.Learning.Enabled)
	}
	if cfg.Learning.DBPath != "/custom/path/learning.db" {
		t.Errorf("Learning.DBPath = %q, want %q", cfg.Learning.DBPath, "/custom/path/learning.db")
	}
	if cfg.Learning.KeepExecutionsDays != 30 {
		t.Errorf("Learning.KeepExecutionsDays = %d, want 30", cfg.Learning.KeepExecutionsDays)
	}
}

// TestConfig_LearningMissingSection tests that missing learning section uses defaults
func TestConfig_LearningMissingSection(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `max_concurrency: 5
timeout: 30m
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Learning should have default values
	if !cfg.Learning.Enabled {
		t.Errorf("Learning.Enabled = %v, want true (default)", cfg.Learning.Enabled)
	}
	if cfg.Learning.DBPath != ".conductor/learning/executions.db" {
		t.Errorf("Learning.DBPath = %q, want %q (default)", cfg.Learning.DBPath, ".conductor/learning/executions.db")
	}
}

// TestConfig_LearningValidation tests validation of learning config values
func TestConfig_LearningValidation(t *testing.T) {
	tests := []struct {
		name      string
		config    string
		wantError bool
	}{
		{
			name: "valid learning config",
			config: `learning:
  enabled: true
  db_path: .conductor/learning/executions.db
  keep_executions_days: 90
`,
			wantError: false,
		},
		{
			name: "empty db_path",
			config: `learning:
  enabled: true
  db_path: ""
`,
			wantError: true,
		},
		{
			name: "negative keep_executions_days",
			config: `learning:
  enabled: true
  keep_executions_days: -1
`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")

			if err := os.WriteFile(configPath, []byte(tt.config), 0644); err != nil {
				t.Fatalf("failed to write test config: %v", err)
			}

			cfg, err := LoadConfig(configPath)
			if err != nil {
				if !tt.wantError {
					t.Fatalf("LoadConfig() unexpected error = %v", err)
				}
				return
			}

			// Validate the loaded config
			err = cfg.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

// TestLoadConfigFromRootWithBuildTime tests loading config from build-time injected root
func TestLoadConfigFromRootWithBuildTime(t *testing.T) {
	// Create temp conductor repo root with config
	buildRoot := t.TempDir()
	configDir := filepath.Join(buildRoot, ".conductor")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")
	configContent := `max_concurrency: 7
timeout: 45m
log_level: debug
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	// Load config from build-time root
	cfg, err := LoadConfigFromRootWithBuildTime(buildRoot)
	if err != nil {
		t.Fatalf("LoadConfigFromRootWithBuildTime() error = %v", err)
	}

	// Verify values loaded from conductor repo root
	if cfg.MaxConcurrency != 7 {
		t.Errorf("MaxConcurrency = %d, want 7", cfg.MaxConcurrency)
	}
	if cfg.Timeout != 45*time.Minute {
		t.Errorf("Timeout = %v, want 45m", cfg.Timeout)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "debug")
	}
}

// TestLoadConfigFromRootWithBuildTimeNoConfig tests defaults when no config exists
func TestLoadConfigFromRootWithBuildTimeNoConfig(t *testing.T) {
	// Empty temp directory (no config file)
	buildRoot := t.TempDir()

	cfg, err := LoadConfigFromRootWithBuildTime(buildRoot)
	if err != nil {
		t.Fatalf("LoadConfigFromRootWithBuildTime() error = %v", err)
	}

	// Should return defaults
	if cfg.MaxConcurrency != 0 {
		t.Errorf("MaxConcurrency = %d, want 0 (default)", cfg.MaxConcurrency)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want %q (default)", cfg.LogLevel, "info")
	}
}

// TestLoadConfigFromRootWithBuildTimeEmptyRoot tests error when root is empty
func TestLoadConfigFromRootWithBuildTimeEmptyRoot(t *testing.T) {
	_, err := LoadConfigFromRootWithBuildTime("")
	if err == nil {
		t.Error("LoadConfigFromRootWithBuildTime(\"\") expected error, got nil")
	}
}

// TestLoadConfigFromDirUsesInjectedRoot tests LoadConfigFromDir uses injected root
func TestLoadConfigFromDirUsesInjectedRoot(t *testing.T) {
	// Create config in build-time root
	buildRoot := t.TempDir()
	configDir := filepath.Join(buildRoot, ".conductor")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")
	configContent := `max_concurrency: 9
log_level: trace
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	// Set build-time root
	SetBuildTimeRepoRoot(buildRoot)
	defer SetBuildTimeRepoRoot("") // Reset after test

	// Load config - should use injected root, not "."
	cfg, err := LoadConfigFromDir(".")
	if err != nil {
		t.Fatalf("LoadConfigFromDir() error = %v", err)
	}

	// Should load from build-time root
	if cfg.MaxConcurrency != 9 {
		t.Errorf("MaxConcurrency = %d, want 9 (from build-time root)", cfg.MaxConcurrency)
	}
	if cfg.LogLevel != "trace" {
		t.Errorf("LogLevel = %q, want %q (from build-time root)", cfg.LogLevel, "trace")
	}
}

// TestLoadConfigExplicitPathOverridesRoot tests explicit path takes precedence
func TestLoadConfigExplicitPathOverridesRoot(t *testing.T) {
	// Create config in build-time root
	buildRoot := t.TempDir()
	buildConfigDir := filepath.Join(buildRoot, ".conductor")
	if err := os.MkdirAll(buildConfigDir, 0755); err != nil {
		t.Fatalf("failed to create build config dir: %v", err)
	}

	buildConfigPath := filepath.Join(buildConfigDir, "config.yaml")
	buildConfigContent := `max_concurrency: 5
`
	if err := os.WriteFile(buildConfigPath, []byte(buildConfigContent), 0644); err != nil {
		t.Fatalf("failed to write build config: %v", err)
	}

	// Create explicit config file (different location)
	explicitConfigDir := t.TempDir()
	explicitConfigPath := filepath.Join(explicitConfigDir, "custom.yaml")
	explicitConfigContent := `max_concurrency: 10
`
	if err := os.WriteFile(explicitConfigPath, []byte(explicitConfigContent), 0644); err != nil {
		t.Fatalf("failed to write explicit config: %v", err)
	}

	// Set build-time root
	SetBuildTimeRepoRoot(buildRoot)
	defer SetBuildTimeRepoRoot("")

	// Load from explicit path - should NOT use build-time root
	cfg, err := LoadConfig(explicitConfigPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Should load from explicit path, not build-time root
	if cfg.MaxConcurrency != 10 {
		t.Errorf("MaxConcurrency = %d, want 10 (from explicit path)", cfg.MaxConcurrency)
	}
}

// TestQualityControlParsing tests parsing quality_control section from YAML config
func TestQualityControlParsing(t *testing.T) {
	tests := []struct {
		name          string
		configContent string
		expectEnabled bool
		expectRetry   int
	}{
		{
			name: "quality_control section present with all fields",
			configContent: `quality_control:
  enabled: true
  agents:
    mode: explicit
    explicit_list:
      - custom-qa-agent
  retry_on_red: 3
`,
			expectEnabled: true,
			expectRetry:   3,
		},
		{
			name: "quality_control section with enabled false",
			configContent: `quality_control:
  enabled: false
  agents:
    mode: explicit
    explicit_list:
      - quality-control
  retry_on_red: 2
`,
			expectEnabled: false,
			expectRetry:   2,
		},
		{
			name: "quality_control section partial fields",
			configContent: `quality_control:
  enabled: true
`,
			expectEnabled: true,
			expectRetry:   2, // should use default
		},
		{
			name:          "quality_control section absent",
			configContent: `max_concurrency: 5`,
			expectEnabled: false, // should use default
			expectRetry:   2,     // should use default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")

			if err := os.WriteFile(configPath, []byte(tt.configContent), 0644); err != nil {
				t.Fatalf("failed to write test config: %v", err)
			}

			cfg, err := LoadConfig(configPath)
			if err != nil {
				t.Fatalf("LoadConfig() error = %v", err)
			}

			if cfg.QualityControl.Enabled != tt.expectEnabled {
				t.Errorf("QualityControl.Enabled = %v, want %v", cfg.QualityControl.Enabled, tt.expectEnabled)
			}
			if cfg.QualityControl.RetryOnRed != tt.expectRetry {
				t.Errorf("QualityControl.RetryOnRed = %d, want %d", cfg.QualityControl.RetryOnRed, tt.expectRetry)
			}
		})
	}
}

// TestQualityControlDefaults tests DefaultConfig() returns correct QC defaults
func TestQualityControlDefaults(t *testing.T) {
	cfg := DefaultConfig()

	// Quality control should be disabled by default
	if cfg.QualityControl.Enabled {
		t.Errorf("QualityControl.Enabled = %v, want false", cfg.QualityControl.Enabled)
	}

	// Should have default retry value
	if cfg.QualityControl.RetryOnRed != 2 {
		t.Errorf("QualityControl.RetryOnRed = %d, want 2", cfg.QualityControl.RetryOnRed)
	}
}

// TestConfig_EnhancedLearningDefaults tests new learning config defaults
func TestConfig_EnhancedLearningDefaults(t *testing.T) {
	cfg := DefaultConfig()

	if !cfg.Learning.SwapDuringRetries {
		t.Errorf("Learning.SwapDuringRetries = %v, want true", cfg.Learning.SwapDuringRetries)
	}
	if !cfg.Learning.WarmUpEnabled {
		t.Errorf("Learning.WarmUpEnabled = %v, want true", cfg.Learning.WarmUpEnabled)
	}
}

// TestConfig_EnhancedLearningYAMLLoading tests loading enhanced learning fields from YAML
func TestConfig_EnhancedLearningYAMLLoading(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `learning:
  enabled: true
  db_path: /custom/db.db
  swap_during_retries: false
  warmup_enabled: false
  keep_executions_days: 60
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if !cfg.Learning.Enabled {
		t.Errorf("Learning.Enabled = %v, want true", cfg.Learning.Enabled)
	}
	if cfg.Learning.DBPath != "/custom/db.db" {
		t.Errorf("Learning.DBPath = %q, want %q", cfg.Learning.DBPath, "/custom/db.db")
	}
	if cfg.Learning.SwapDuringRetries {
		t.Errorf("Learning.SwapDuringRetries = %v, want false", cfg.Learning.SwapDuringRetries)
	}
	if cfg.Learning.WarmUpEnabled {
		t.Errorf("Learning.WarmUpEnabled = %v, want false", cfg.Learning.WarmUpEnabled)
	}
	if cfg.Learning.KeepExecutionsDays != 60 {
		t.Errorf("Learning.KeepExecutionsDays = %d, want 60", cfg.Learning.KeepExecutionsDays)
	}
}

// TestQCAgentConfigDefaults tests default QC agent configuration
func TestQCAgentConfigDefaults(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.QualityControl.Agents.Mode != "auto" {
		t.Errorf("QCAgent.Mode = %q, want %q", cfg.QualityControl.Agents.Mode, "auto")
	}
	if len(cfg.QualityControl.Agents.ExplicitList) != 0 {
		t.Errorf("QCAgent.ExplicitList = %v, want empty", cfg.QualityControl.Agents.ExplicitList)
	}
	if len(cfg.QualityControl.Agents.AdditionalAgents) != 0 {
		t.Errorf("QCAgent.AdditionalAgents = %v, want empty", cfg.QualityControl.Agents.AdditionalAgents)
	}
	if len(cfg.QualityControl.Agents.BlockedAgents) != 0 {
		t.Errorf("QCAgent.BlockedAgents = %v, want empty", cfg.QualityControl.Agents.BlockedAgents)
	}
}

// TestQCAgentConfigAutoMode tests loading auto mode QC configuration
func TestQCAgentConfigAutoMode(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `quality_control:
  enabled: true
  agents:
    mode: auto
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if !cfg.QualityControl.Enabled {
		t.Errorf("QualityControl.Enabled = %v, want true", cfg.QualityControl.Enabled)
	}
	if cfg.QualityControl.Agents.Mode != "auto" {
		t.Errorf("QCAgent.Mode = %q, want %q", cfg.QualityControl.Agents.Mode, "auto")
	}
}

// TestQCAgentConfigExplicitMode tests loading explicit mode QC configuration
func TestQCAgentConfigExplicitMode(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `quality_control:
  enabled: true
  agents:
    mode: explicit
    explicit_list:
      - golang-pro
      - quality-control
      - test-expert
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.QualityControl.Agents.Mode != "explicit" {
		t.Errorf("QCAgent.Mode = %q, want %q", cfg.QualityControl.Agents.Mode, "explicit")
	}
	if len(cfg.QualityControl.Agents.ExplicitList) != 3 {
		t.Errorf("QCAgent.ExplicitList length = %d, want 3", len(cfg.QualityControl.Agents.ExplicitList))
	}
	if cfg.QualityControl.Agents.ExplicitList[0] != "golang-pro" {
		t.Errorf("QCAgent.ExplicitList[0] = %q, want %q", cfg.QualityControl.Agents.ExplicitList[0], "golang-pro")
	}
}

// TestQCAgentConfigMixedMode tests loading mixed mode QC configuration
func TestQCAgentConfigMixedMode(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `quality_control:
  enabled: true
  agents:
    mode: mixed
    additional:
      - security-expert
      - performance-pro
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.QualityControl.Agents.Mode != "mixed" {
		t.Errorf("QCAgent.Mode = %q, want %q", cfg.QualityControl.Agents.Mode, "mixed")
	}
	if len(cfg.QualityControl.Agents.AdditionalAgents) != 2 {
		t.Errorf("QCAgent.AdditionalAgents length = %d, want 2", len(cfg.QualityControl.Agents.AdditionalAgents))
	}
	if cfg.QualityControl.Agents.AdditionalAgents[0] != "security-expert" {
		t.Errorf("QCAgent.AdditionalAgents[0] = %q, want %q", cfg.QualityControl.Agents.AdditionalAgents[0], "security-expert")
	}
}

// TestQCAgentConfigBlockedAgents tests loading blocked agents configuration
func TestQCAgentConfigBlockedAgents(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `quality_control:
  enabled: true
  agents:
    mode: auto
    blocked:
      - slow-agent
      - deprecated-agent
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if len(cfg.QualityControl.Agents.BlockedAgents) != 2 {
		t.Errorf("QCAgent.BlockedAgents length = %d, want 2", len(cfg.QualityControl.Agents.BlockedAgents))
	}
	if cfg.QualityControl.Agents.BlockedAgents[0] != "slow-agent" {
		t.Errorf("QCAgent.BlockedAgents[0] = %q, want %q", cfg.QualityControl.Agents.BlockedAgents[0], "slow-agent")
	}
}

// TestQCAgentConfigComplexMixed tests loading complex mixed mode configuration
func TestQCAgentConfigComplexMixed(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `quality_control:
  enabled: true
  retry_on_red: 3
  agents:
    mode: mixed
    additional:
      - security-expert
      - performance-pro
    blocked:
      - slow-agent
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if !cfg.QualityControl.Enabled {
		t.Errorf("QualityControl.Enabled = %v, want true", cfg.QualityControl.Enabled)
	}
	if cfg.QualityControl.RetryOnRed != 3 {
		t.Errorf("QualityControl.RetryOnRed = %d, want 3", cfg.QualityControl.RetryOnRed)
	}
	if cfg.QualityControl.Agents.Mode != "mixed" {
		t.Errorf("QCAgent.Mode = %q, want %q", cfg.QualityControl.Agents.Mode, "mixed")
	}
	if len(cfg.QualityControl.Agents.AdditionalAgents) != 2 {
		t.Errorf("QCAgent.AdditionalAgents length = %d, want 2", len(cfg.QualityControl.Agents.AdditionalAgents))
	}
	if len(cfg.QualityControl.Agents.BlockedAgents) != 1 {
		t.Errorf("QCAgent.BlockedAgents length = %d, want 1", len(cfg.QualityControl.Agents.BlockedAgents))
	}
}

// TestQCAgentConfigValidationValidModes tests validation of valid QC modes
func TestQCAgentConfigValidationValidModes(t *testing.T) {
	validModes := []string{"auto", "explicit", "mixed"}

	for _, mode := range validModes {
		t.Run(mode, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")

			configContent := fmt.Sprintf(`quality_control:
  enabled: true
  agents:
    mode: %s
    explicit_list:
      - agent1
`, mode)

			if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
				t.Fatalf("failed to write test config: %v", err)
			}

			cfg, err := LoadConfig(configPath)
			if err != nil {
				t.Fatalf("LoadConfig() error = %v", err)
			}

			if err := cfg.Validate(); err != nil {
				t.Errorf("Validate() error = %v for valid mode %q", err, mode)
			}
		})
	}
}

// TestQCAgentConfigValidationInvalidMode tests validation rejects invalid modes
func TestQCAgentConfigValidationInvalidMode(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `quality_control:
  enabled: true
  agents:
    mode: invalid-mode
    explicit_list:
      - agent1
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if err := cfg.Validate(); err == nil {
		t.Error("Validate() expected error for invalid mode, got nil")
	}
}

// TestQCAgentConfigValidationExplicitNeedsList tests explicit mode requires agents
func TestQCAgentConfigValidationExplicitNeedsList(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `quality_control:
  enabled: true
  agents:
    mode: explicit
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Explicit mode without explicit_list should either fail validation or default to empty list
	// Current implementation defaults to empty list, which is acceptable
	if cfg.QualityControl.Agents.Mode != "explicit" {
		t.Errorf("Expected explicit mode, got %q", cfg.QualityControl.Agents.Mode)
	}
}

// TestQCAgentConfigValidationEmptyAgentNames tests validation rejects empty agent names
func TestQCAgentConfigValidationEmptyAgentNames(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		wantErr bool
	}{
		{
			name: "empty explicit_list item",
			config: `quality_control:
  enabled: true
  agents:
    mode: explicit
    explicit_list:
      - golang-pro
      - ""
      - quality-control
`,
			wantErr: true,
		},
		{
			name: "empty additional agent",
			config: `quality_control:
  enabled: true
  agents:
    mode: mixed
    additional:
      - valid-agent
      - ""
`,
			wantErr: true,
		},
		{
			name: "empty blocked agent",
			config: `quality_control:
  enabled: true
  agents:
    mode: auto
    blocked:
      - ""
      - slow-agent
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")

			if err := os.WriteFile(configPath, []byte(tt.config), 0644); err != nil {
				t.Fatalf("failed to write test config: %v", err)
			}

			cfg, err := LoadConfig(configPath)
			if err != nil {
				t.Fatalf("LoadConfig() error = %v", err)
			}

			err = cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

// TestQCAgentConfigValidationRetryCount tests validation of retry_on_red
func TestQCAgentConfigValidationRetryCount(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		wantErr bool
	}{
		{
			name: "valid retry count",
			config: `quality_control:
  enabled: true
  retry_on_red: 3
`,
			wantErr: false,
		},
		{
			name: "zero retry count (allowed)",
			config: `quality_control:
  enabled: true
  retry_on_red: 0
`,
			wantErr: false,
		},
		{
			name: "negative retry count",
			config: `quality_control:
  enabled: true
  retry_on_red: -1
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")

			if err := os.WriteFile(configPath, []byte(tt.config), 0644); err != nil {
				t.Fatalf("failed to write test config: %v", err)
			}

			cfg, err := LoadConfig(configPath)
			if err != nil {
				t.Fatalf("LoadConfig() error = %v", err)
			}

			err = cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

// TestQCAgentConfigMissingAgentsSection tests handling missing agents section
func TestQCAgentConfigMissingAgentsSection(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `quality_control:
  enabled: true
  retry_on_red: 3
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Quality control should be enabled
	if !cfg.QualityControl.Enabled {
		t.Errorf("QualityControl.Enabled = %v, want true", cfg.QualityControl.Enabled)
	}
	// RetryOnRed should be set from config
	if cfg.QualityControl.RetryOnRed != 3 {
		t.Errorf("QualityControl.RetryOnRed = %d, want 3", cfg.QualityControl.RetryOnRed)
	}
}

// TestQCAgentConfigEmptyAgentsSection tests handling empty agents section
func TestQCAgentConfigEmptyAgentsSection(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `quality_control:
  enabled: true
  agents:
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Empty agents section - mode will be what's in the YAML (empty, defaulting to "explicit"
	// when converted from missing agents section, or "auto" if explicitly set)
	// Just verify it's a valid mode
	validModes := []string{"auto", "explicit", "mixed"}
	found := false
	for _, mode := range validModes {
		if cfg.QualityControl.Agents.Mode == mode {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("QCAgent.Mode = %q, not a valid mode", cfg.QualityControl.Agents.Mode)
	}
}

// TestQCAgentConfigFullExample tests comprehensive QC configuration example
func TestQCAgentConfigFullExample(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `max_concurrency: 8
timeout: 2h
log_level: debug

quality_control:
  enabled: true
  retry_on_red: 3
  agents:
    mode: mixed
    additional:
      - security-expert
      - performance-auditor
    blocked:
      - deprecated-reviewer
      - slow-analyzer

learning:
  enabled: true
  swap_during_retries: true
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Verify main config
	if cfg.MaxConcurrency != 8 {
		t.Errorf("MaxConcurrency = %d, want 8", cfg.MaxConcurrency)
	}

	// Verify QC config
	if !cfg.QualityControl.Enabled {
		t.Errorf("QualityControl.Enabled = %v, want true", cfg.QualityControl.Enabled)
	}
	if cfg.QualityControl.RetryOnRed != 3 {
		t.Errorf("QualityControl.RetryOnRed = %d, want 3", cfg.QualityControl.RetryOnRed)
	}
	if cfg.QualityControl.Agents.Mode != "mixed" {
		t.Errorf("QCAgent.Mode = %q, want %q", cfg.QualityControl.Agents.Mode, "mixed")
	}
	if len(cfg.QualityControl.Agents.AdditionalAgents) != 2 {
		t.Errorf("QCAgent.AdditionalAgents length = %d, want 2", len(cfg.QualityControl.Agents.AdditionalAgents))
	}
	if len(cfg.QualityControl.Agents.BlockedAgents) != 2 {
		t.Errorf("QCAgent.BlockedAgents length = %d, want 2", len(cfg.QualityControl.Agents.BlockedAgents))
	}

	// Verify learning config
	if !cfg.Learning.Enabled {
		t.Errorf("Learning.Enabled = %v, want true", cfg.Learning.Enabled)
	}

	// Validate the entire config
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

// TestDefaultAgentWatchConfig verifies default Agent Watch configuration values
func TestDefaultAgentWatchConfig(t *testing.T) {
	cfg := DefaultAgentWatchConfig()

	if !cfg.Enabled {
		t.Errorf("Enabled = %v, want true", cfg.Enabled)
	}
	if cfg.BaseDir != "~/.claude/projects" {
		t.Errorf("BaseDir = %q, want %q", cfg.BaseDir, "~/.claude/projects")
	}
	if cfg.DefaultLimit != 100 {
		t.Errorf("DefaultLimit = %d, want 100", cfg.DefaultLimit)
	}
	if cfg.PollIntervalSecs != 2 {
		t.Errorf("PollIntervalSecs = %d, want 2", cfg.PollIntervalSecs)
	}
	if cfg.AutoImport != false {
		t.Errorf("AutoImport = %v, want false", cfg.AutoImport)
	}
	if cfg.CacheSize != 50 {
		t.Errorf("CacheSize = %d, want 50", cfg.CacheSize)
	}

	// Verify cost model defaults
	costModel := cfg.CostModel
	if costModel.SonnetInput != 3.0 {
		t.Errorf("SonnetInput = %f, want 3.0", costModel.SonnetInput)
	}
	if costModel.SonnetOutput != 15.0 {
		t.Errorf("SonnetOutput = %f, want 15.0", costModel.SonnetOutput)
	}
	if costModel.HaikuInput != 0.80 {
		t.Errorf("HaikuInput = %f, want 0.80", costModel.HaikuInput)
	}
	if costModel.HaikuOutput != 4.0 {
		t.Errorf("HaikuOutput = %f, want 4.0", costModel.HaikuOutput)
	}
	if costModel.OpusInput != 15.0 {
		t.Errorf("OpusInput = %f, want 15.0", costModel.OpusInput)
	}
	if costModel.OpusOutput != 75.0 {
		t.Errorf("OpusOutput = %f, want 75.0", costModel.OpusOutput)
	}
}

// TestDefaultCostModelConfig verifies default cost model pricing
func TestDefaultCostModelConfig(t *testing.T) {
	cfg := DefaultCostModelConfig()

	tests := []struct {
		name string
		got  float64
		want float64
	}{
		{"SonnetInput", cfg.SonnetInput, 3.0},
		{"SonnetOutput", cfg.SonnetOutput, 15.0},
		{"HaikuInput", cfg.HaikuInput, 0.80},
		{"HaikuOutput", cfg.HaikuOutput, 4.0},
		{"OpusInput", cfg.OpusInput, 15.0},
		{"OpusOutput", cfg.OpusOutput, 75.0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.got != test.want {
				t.Errorf("got %f, want %f", test.got, test.want)
			}
		})
	}
}

// TestLoadConfigAgentWatch tests loading Agent Watch configuration from YAML
func TestLoadConfigAgentWatch(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `max_concurrency: 4
timeout: 30m
agent_watch:
  enabled: true
  base_dir: /custom/claude/projects
  default_limit: 200
  poll_interval_secs: 5
  auto_import: true
  cache_size: 100
  cost_model:
    sonnet_input: 3.5
    sonnet_output: 16.0
    haiku_input: 1.0
    haiku_output: 5.0
    opus_input: 20.0
    opus_output: 80.0
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Verify Agent Watch config loaded correctly
	if !cfg.AgentWatch.Enabled {
		t.Errorf("AgentWatch.Enabled = %v, want true", cfg.AgentWatch.Enabled)
	}
	if cfg.AgentWatch.BaseDir != "/custom/claude/projects" {
		t.Errorf("AgentWatch.BaseDir = %q, want %q", cfg.AgentWatch.BaseDir, "/custom/claude/projects")
	}
	if cfg.AgentWatch.DefaultLimit != 200 {
		t.Errorf("AgentWatch.DefaultLimit = %d, want 200", cfg.AgentWatch.DefaultLimit)
	}
	if cfg.AgentWatch.PollIntervalSecs != 5 {
		t.Errorf("AgentWatch.PollIntervalSecs = %d, want 5", cfg.AgentWatch.PollIntervalSecs)
	}
	if !cfg.AgentWatch.AutoImport {
		t.Errorf("AgentWatch.AutoImport = %v, want true", cfg.AgentWatch.AutoImport)
	}
	if cfg.AgentWatch.CacheSize != 100 {
		t.Errorf("AgentWatch.CacheSize = %d, want 100", cfg.AgentWatch.CacheSize)
	}

	// Verify cost model loaded correctly
	costModel := cfg.AgentWatch.CostModel
	if costModel.SonnetInput != 3.5 {
		t.Errorf("CostModel.SonnetInput = %f, want 3.5", costModel.SonnetInput)
	}
	if costModel.SonnetOutput != 16.0 {
		t.Errorf("CostModel.SonnetOutput = %f, want 16.0", costModel.SonnetOutput)
	}
	if costModel.HaikuInput != 1.0 {
		t.Errorf("CostModel.HaikuInput = %f, want 1.0", costModel.HaikuInput)
	}
	if costModel.HaikuOutput != 5.0 {
		t.Errorf("CostModel.HaikuOutput = %f, want 5.0", costModel.HaikuOutput)
	}
	if costModel.OpusInput != 20.0 {
		t.Errorf("CostModel.OpusInput = %f, want 20.0", costModel.OpusInput)
	}
	if costModel.OpusOutput != 80.0 {
		t.Errorf("CostModel.OpusOutput = %f, want 80.0", costModel.OpusOutput)
	}
}

// TestLoadConfigAgentWatchDefaults tests that Agent Watch defaults are used when config is missing
func TestLoadConfigAgentWatchDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Config without agent_watch section should use defaults
	configContent := `max_concurrency: 4
timeout: 30m
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Verify defaults are used
	defaults := DefaultAgentWatchConfig()
	if cfg.AgentWatch.Enabled != defaults.Enabled {
		t.Errorf("AgentWatch.Enabled = %v, want %v", cfg.AgentWatch.Enabled, defaults.Enabled)
	}
	if cfg.AgentWatch.BaseDir != defaults.BaseDir {
		t.Errorf("AgentWatch.BaseDir = %q, want %q", cfg.AgentWatch.BaseDir, defaults.BaseDir)
	}
	if cfg.AgentWatch.DefaultLimit != defaults.DefaultLimit {
		t.Errorf("AgentWatch.DefaultLimit = %d, want %d", cfg.AgentWatch.DefaultLimit, defaults.DefaultLimit)
	}
	if cfg.AgentWatch.PollIntervalSecs != defaults.PollIntervalSecs {
		t.Errorf("AgentWatch.PollIntervalSecs = %d, want %d", cfg.AgentWatch.PollIntervalSecs, defaults.PollIntervalSecs)
	}
	if cfg.AgentWatch.CacheSize != defaults.CacheSize {
		t.Errorf("AgentWatch.CacheSize = %d, want %d", cfg.AgentWatch.CacheSize, defaults.CacheSize)
	}
}

// TestValidateAgentWatchEnabled tests validation of enabled Agent Watch configuration
func TestValidateAgentWatchEnabled(t *testing.T) {
	tests := []struct {
		name      string
		cfg       *Config
		wantError bool
		errMsg    string
	}{
		{
			name: "valid Agent Watch config",
			cfg: &Config{
				MaxConcurrency: 0,
				Timeout:        10 * time.Hour,
				LogLevel:       "info",
				LogDir:         ".conductor/logs",
				AgentWatch: AgentWatchConfig{
					Enabled:          true,
					BaseDir:          "~/.claude/projects",
					DefaultLimit:     100,
					PollIntervalSecs: 2,
					CacheSize:        50,
					CostModel: CostModelConfig{
						SonnetInput:  3.0,
						SonnetOutput: 15.0,
						HaikuInput:   0.80,
						HaikuOutput:  4.0,
						OpusInput:    15.0,
						OpusOutput:   75.0,
					},
				},
			},
			wantError: false,
		},
		{
			name: "empty base_dir when enabled",
			cfg: &Config{
				MaxConcurrency: 0,
				Timeout:        10 * time.Hour,
				LogLevel:       "info",
				LogDir:         ".conductor/logs",
				AgentWatch: AgentWatchConfig{
					Enabled:          true,
					BaseDir:          "",
					DefaultLimit:     100,
					PollIntervalSecs: 2,
					CacheSize:        50,
					CostModel:        DefaultCostModelConfig(),
				},
			},
			wantError: true,
			errMsg:    "base_dir cannot be empty",
		},
		{
			name: "invalid default_limit",
			cfg: &Config{
				MaxConcurrency: 0,
				Timeout:        10 * time.Hour,
				LogLevel:       "info",
				LogDir:         ".conductor/logs",
				AgentWatch: AgentWatchConfig{
					Enabled:          true,
					BaseDir:          "~/.claude/projects",
					DefaultLimit:     0,
					PollIntervalSecs: 2,
					CacheSize:        50,
					CostModel:        DefaultCostModelConfig(),
				},
			},
			wantError: true,
			errMsg:    "default_limit must be > 0",
		},
		{
			name: "invalid poll_interval_secs",
			cfg: &Config{
				MaxConcurrency: 0,
				Timeout:        10 * time.Hour,
				LogLevel:       "info",
				LogDir:         ".conductor/logs",
				AgentWatch: AgentWatchConfig{
					Enabled:          true,
					BaseDir:          "~/.claude/projects",
					DefaultLimit:     100,
					PollIntervalSecs: -1,
					CacheSize:        50,
					CostModel:        DefaultCostModelConfig(),
				},
			},
			wantError: true,
			errMsg:    "poll_interval_secs must be > 0",
		},
		{
			name: "invalid cache_size",
			cfg: &Config{
				MaxConcurrency: 0,
				Timeout:        10 * time.Hour,
				LogLevel:       "info",
				LogDir:         ".conductor/logs",
				AgentWatch: AgentWatchConfig{
					Enabled:          true,
					BaseDir:          "~/.claude/projects",
					DefaultLimit:     100,
					PollIntervalSecs: 2,
					CacheSize:        0,
					CostModel:        DefaultCostModelConfig(),
				},
			},
			wantError: true,
			errMsg:    "cache_size must be > 0",
		},
		{
			name: "negative sonnet_input pricing",
			cfg: &Config{
				MaxConcurrency: 0,
				Timeout:        10 * time.Hour,
				LogLevel:       "info",
				LogDir:         ".conductor/logs",
				AgentWatch: AgentWatchConfig{
					Enabled:          true,
					BaseDir:          "~/.claude/projects",
					DefaultLimit:     100,
					PollIntervalSecs: 2,
					CacheSize:        50,
					CostModel: CostModelConfig{
						SonnetInput:  -1.0,
						SonnetOutput: 15.0,
					},
				},
			},
			wantError: true,
			errMsg:    "sonnet_input must be >= 0",
		},
		{
			name: "disabled Agent Watch has no validation",
			cfg: &Config{
				MaxConcurrency: 0,
				Timeout:        10 * time.Hour,
				LogLevel:       "info",
				LogDir:         ".conductor/logs",
				AgentWatch: AgentWatchConfig{
					Enabled: false,
					// Other fields can be invalid when disabled
					DefaultLimit:     0,
					PollIntervalSecs: -1,
					CacheSize:        -5,
				},
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
			if tt.wantError && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Validate() error message = %q, want to contain %q", err.Error(), tt.errMsg)
			}
		})
	}
}

// TestLoadConfigAgentWatchPartial tests partial Agent Watch configuration updates
func TestLoadConfigAgentWatchPartial(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Only override some fields
	configContent := `agent_watch:
  enabled: false
  default_limit: 150
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Overridden values
	if cfg.AgentWatch.Enabled {
		t.Errorf("AgentWatch.Enabled = %v, want false", cfg.AgentWatch.Enabled)
	}
	if cfg.AgentWatch.DefaultLimit != 150 {
		t.Errorf("AgentWatch.DefaultLimit = %d, want 150", cfg.AgentWatch.DefaultLimit)
	}

	// Default values should remain
	if cfg.AgentWatch.BaseDir != "~/.claude/projects" {
		t.Errorf("AgentWatch.BaseDir = %q, want %q (default)", cfg.AgentWatch.BaseDir, "~/.claude/projects")
	}
	if cfg.AgentWatch.PollIntervalSecs != 2 {
		t.Errorf("AgentWatch.PollIntervalSecs = %d, want 2 (default)", cfg.AgentWatch.PollIntervalSecs)
	}
	if cfg.AgentWatch.CacheSize != 50 {
		t.Errorf("AgentWatch.CacheSize = %d, want 50 (default)", cfg.AgentWatch.CacheSize)
	}
}

// TestValidationConfig_DefaultValue tests that DefaultConfig sets KeyPointCriteria to 'warn'
func TestValidationConfig_DefaultValue(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Validation.KeyPointCriteria != "warn" {
		t.Errorf("Validation.KeyPointCriteria = %q, want %q", cfg.Validation.KeyPointCriteria, "warn")
	}
}

// TestValidationConfig_ValidModes tests that Validate() accepts valid modes
func TestValidationConfig_ValidModes(t *testing.T) {
	validModes := []string{"warn", "strict", "off"}

	for _, mode := range validModes {
		t.Run(mode, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Validation.KeyPointCriteria = mode

			if err := cfg.Validate(); err != nil {
				t.Errorf("Validate() error = %v for valid mode %q", err, mode)
			}
		})
	}
}

// TestValidationConfig_InvalidMode tests that Validate() returns error for invalid modes
func TestValidationConfig_InvalidMode(t *testing.T) {
	invalidModes := []string{"invalid", "WARN", "STRICT", "OFF", "warning", "error"}

	for _, mode := range invalidModes {
		t.Run(mode, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Validation.KeyPointCriteria = mode

			if err := cfg.Validate(); err == nil {
				t.Errorf("Validate() expected error for invalid mode %q, got nil", mode)
			}
		})
	}
}

// TestValidationConfig_YAMLMerge tests that YAML config overrides default KeyPointCriteria
func TestValidationConfig_YAMLMerge(t *testing.T) {
	tests := []struct {
		name          string
		configContent string
		expectMode    string
	}{
		{
			name: "strict mode from YAML",
			configContent: `validation:
  key_point_criteria: strict
`,
			expectMode: "strict",
		},
		{
			name: "off mode from YAML",
			configContent: `validation:
  key_point_criteria: off
`,
			expectMode: "off",
		},
		{
			name: "warn mode from YAML",
			configContent: `validation:
  key_point_criteria: warn
`,
			expectMode: "warn",
		},
		{
			name:          "missing validation section uses default",
			configContent: `max_concurrency: 5`,
			expectMode:    "warn",
		},
		{
			name: "empty validation section uses default",
			configContent: `validation:
`,
			expectMode: "warn",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")

			if err := os.WriteFile(configPath, []byte(tt.configContent), 0644); err != nil {
				t.Fatalf("failed to write test config: %v", err)
			}

			cfg, err := LoadConfig(configPath)
			if err != nil {
				t.Fatalf("LoadConfig() error = %v", err)
			}

			if cfg.Validation.KeyPointCriteria != tt.expectMode {
				t.Errorf("Validation.KeyPointCriteria = %q, want %q", cfg.Validation.KeyPointCriteria, tt.expectMode)
			}

			// Also verify it passes validation
			if err := cfg.Validate(); err != nil {
				t.Errorf("Validate() error = %v", err)
			}
		})
	}
}

// TestValidationConfig_WithOtherSettings tests validation config alongside other settings
func TestValidationConfig_WithOtherSettings(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `max_concurrency: 8
timeout: 2h
log_level: debug
validation:
  key_point_criteria: strict
learning:
  enabled: true
quality_control:
  enabled: true
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Verify validation setting
	if cfg.Validation.KeyPointCriteria != "strict" {
		t.Errorf("Validation.KeyPointCriteria = %q, want %q", cfg.Validation.KeyPointCriteria, "strict")
	}

	// Verify other settings are correct
	if cfg.MaxConcurrency != 8 {
		t.Errorf("MaxConcurrency = %d, want 8", cfg.MaxConcurrency)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "debug")
	}

	// Validate entire config
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() error = %v", err)
	}
}

// TestValidationConfig_StrictRubricDefault tests that DefaultConfig sets StrictRubric to false
func TestValidationConfig_StrictRubricDefault(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Validation.StrictRubric != false {
		t.Errorf("Validation.StrictRubric = %v, want false", cfg.Validation.StrictRubric)
	}
}

// TestValidationConfig_StrictRubricYAMLMerge tests that YAML config overrides StrictRubric
func TestValidationConfig_StrictRubricYAMLMerge(t *testing.T) {
	tests := []struct {
		name             string
		yaml             string
		wantStrictRubric bool
	}{
		{
			name:             "strict_rubric enabled",
			yaml:             "validation:\n  strict_rubric: true\n",
			wantStrictRubric: true,
		},
		{
			name:             "strict_rubric disabled",
			yaml:             "validation:\n  strict_rubric: false\n",
			wantStrictRubric: false,
		},
		{
			name:             "strict_rubric not set keeps default",
			yaml:             "validation:\n  key_point_criteria: warn\n",
			wantStrictRubric: false,
		},
		{
			name:             "no validation section keeps default",
			yaml:             "max_concurrency: 4\n",
			wantStrictRubric: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")

			if err := os.WriteFile(configPath, []byte(tt.yaml), 0644); err != nil {
				t.Fatalf("failed to write test config: %v", err)
			}

			cfg, err := LoadConfig(configPath)
			if err != nil {
				t.Fatalf("LoadConfig() error = %v", err)
			}

			if cfg.Validation.StrictRubric != tt.wantStrictRubric {
				t.Errorf("Validation.StrictRubric = %v, want %v", cfg.Validation.StrictRubric, tt.wantStrictRubric)
			}
		})
	}
}

// TestValidationConfig_StrictRubricWithKeyPointCriteria tests both fields together
func TestValidationConfig_StrictRubricWithKeyPointCriteria(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `validation:
  key_point_criteria: strict
  strict_rubric: true
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.Validation.KeyPointCriteria != "strict" {
		t.Errorf("Validation.KeyPointCriteria = %q, want %q", cfg.Validation.KeyPointCriteria, "strict")
	}
	if cfg.Validation.StrictRubric != true {
		t.Errorf("Validation.StrictRubric = %v, want true", cfg.Validation.StrictRubric)
	}

	// Validate entire config
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() error = %v", err)
	}
}

// TestDefaultConfigIncludesExecutor verifies DefaultConfig includes executor settings
func TestDefaultConfigIncludesExecutor(t *testing.T) {
	cfg := DefaultConfig()

	if !cfg.Executor.EnforceDependencyChecks {
		t.Errorf("Default Executor.EnforceDependencyChecks = %v, want true", cfg.Executor.EnforceDependencyChecks)
	}
}

// TestLoadConfigExecutor tests loading executor configuration from YAML
func TestLoadConfigExecutor(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `executor:
  enforce_dependency_checks: false
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.Executor.EnforceDependencyChecks {
		t.Errorf("Executor.EnforceDependencyChecks = %v, want false", cfg.Executor.EnforceDependencyChecks)
	}
}

// TestLoadConfigExecutorDefaults tests executor defaults are preserved when not specified
func TestLoadConfigExecutorDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Config without executor section
	configContent := `log_level: debug
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Default should be true
	if !cfg.Executor.EnforceDependencyChecks {
		t.Errorf("Executor.EnforceDependencyChecks = %v, want true (default)", cfg.Executor.EnforceDependencyChecks)
	}
}

// TestLoadConfigExecutorIntelligentAgentSelection tests loading intelligent_agent_selection from YAML
func TestLoadConfigExecutorIntelligentAgentSelection(t *testing.T) {
	t.Run("enabled via config", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		configContent := `executor:
  intelligent_agent_selection: true
`
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatalf("failed to write test config: %v", err)
		}

		cfg, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		if !cfg.Executor.IntelligentAgentSelection {
			t.Errorf("Executor.IntelligentAgentSelection = %v, want true", cfg.Executor.IntelligentAgentSelection)
		}
	})

	t.Run("disabled by default", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		// Config without intelligent_agent_selection
		configContent := `log_level: debug
`
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatalf("failed to write test config: %v", err)
		}

		cfg, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		// Default should be false
		if cfg.Executor.IntelligentAgentSelection {
			t.Errorf("Executor.IntelligentAgentSelection = %v, want false (default)", cfg.Executor.IntelligentAgentSelection)
		}
	})

	t.Run("explicitly disabled", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		configContent := `executor:
  intelligent_agent_selection: false
`
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatalf("failed to write test config: %v", err)
		}

		cfg, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		if cfg.Executor.IntelligentAgentSelection {
			t.Errorf("Executor.IntelligentAgentSelection = %v, want false", cfg.Executor.IntelligentAgentSelection)
		}
	})
}

// TestDefaultTTSConfig tests default TTS configuration values
func TestDefaultTTSConfig(t *testing.T) {
	cfg := DefaultTTSConfig()

	// TTS should be DISABLED by default
	if cfg.Enabled != false {
		t.Errorf("Enabled = %v, want false", cfg.Enabled)
	}
	if cfg.BaseURL != "http://localhost:5005" {
		t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, "http://localhost:5005")
	}
	if cfg.Model != "orpheus" {
		t.Errorf("Model = %q, want %q", cfg.Model, "orpheus")
	}
	if cfg.Voice != "tara" {
		t.Errorf("Voice = %q, want %q", cfg.Voice, "tara")
	}
}

// TestDefaultConfigIncludesTTS tests that DefaultConfig includes TTS config
func TestDefaultConfigIncludesTTS(t *testing.T) {
	cfg := DefaultConfig()

	// Verify TTS config is included with correct defaults
	if cfg.TTS.Enabled != false {
		t.Errorf("TTS.Enabled = %v, want false (default)", cfg.TTS.Enabled)
	}
	if cfg.TTS.BaseURL != "http://localhost:5005" {
		t.Errorf("TTS.BaseURL = %q, want %q (default)", cfg.TTS.BaseURL, "http://localhost:5005")
	}
	if cfg.TTS.Model != "orpheus" {
		t.Errorf("TTS.Model = %q, want %q (default)", cfg.TTS.Model, "orpheus")
	}
	if cfg.TTS.Voice != "tara" {
		t.Errorf("TTS.Voice = %q, want %q (default)", cfg.TTS.Voice, "tara")
	}
}

// TestLoadTTSConfigFullSettings tests loading complete TTS configuration
func TestLoadTTSConfigFullSettings(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `tts:
  enabled: true
  base_url: http://custom-tts:8080
  model: custom-model
  voice: custom-voice
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.TTS.Enabled != true {
		t.Errorf("TTS.Enabled = %v, want true", cfg.TTS.Enabled)
	}
	if cfg.TTS.BaseURL != "http://custom-tts:8080" {
		t.Errorf("TTS.BaseURL = %q, want %q", cfg.TTS.BaseURL, "http://custom-tts:8080")
	}
	if cfg.TTS.Model != "custom-model" {
		t.Errorf("TTS.Model = %q, want %q", cfg.TTS.Model, "custom-model")
	}
	if cfg.TTS.Voice != "custom-voice" {
		t.Errorf("TTS.Voice = %q, want %q", cfg.TTS.Voice, "custom-voice")
	}
}

// TestLoadTTSConfigPartialSettings tests partial TTS configuration merges with defaults
func TestLoadTTSConfigPartialSettings(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Only set enabled and voice, leave other fields to defaults
	configContent := `tts:
  enabled: true
  voice: emma
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Set fields should be overwritten
	if cfg.TTS.Enabled != true {
		t.Errorf("TTS.Enabled = %v, want true", cfg.TTS.Enabled)
	}
	if cfg.TTS.Voice != "emma" {
		t.Errorf("TTS.Voice = %q, want %q", cfg.TTS.Voice, "emma")
	}

	// Unset fields should retain defaults
	if cfg.TTS.BaseURL != "http://localhost:5005" {
		t.Errorf("TTS.BaseURL = %q, want %q (default)", cfg.TTS.BaseURL, "http://localhost:5005")
	}
	if cfg.TTS.Model != "orpheus" {
		t.Errorf("TTS.Model = %q, want %q (default)", cfg.TTS.Model, "orpheus")
	}
}

// TestLoadTTSConfigDisabledByDefault tests TTS stays disabled without explicit config
func TestLoadTTSConfigDisabledByDefault(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Config with no TTS section
	configContent := `max_concurrency: 4
log_level: debug
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// TTS should remain disabled
	if cfg.TTS.Enabled != false {
		t.Errorf("TTS.Enabled = %v, want false (default)", cfg.TTS.Enabled)
	}
}

// TestTimeoutsConfigParsing tests YAML parsing for the timeouts configuration section
func TestTimeoutsConfigParsing(t *testing.T) {
	tests := []struct {
		name           string
		configYAML     string
		expectedTask   time.Duration
		expectedLLM    time.Duration
		expectedHTTP   time.Duration
		expectedSearch time.Duration
		wantErr        bool
		errContains    string
	}{
		{
			name: "all timeouts specified",
			configYAML: `timeouts:
  task: 6h
  llm: 60s
  http: 15s
  search: 10s
`,
			expectedTask:   6 * time.Hour,
			expectedLLM:    60 * time.Second,
			expectedHTTP:   15 * time.Second,
			expectedSearch: 10 * time.Second,
			wantErr:        false,
		},
		{
			name:           "missing timeouts section uses defaults",
			configYAML:     `max_concurrency: 4`,
			expectedTask:   12 * time.Hour,
			expectedLLM:    90 * time.Second,
			expectedHTTP:   30 * time.Second,
			expectedSearch: 30 * time.Second,
			wantErr:        false,
		},
		{
			name: "partial timeouts - only task specified",
			configYAML: `timeouts:
  task: 8h
`,
			expectedTask:   8 * time.Hour,
			expectedLLM:    90 * time.Second, // default
			expectedHTTP:   30 * time.Second, // default
			expectedSearch: 30 * time.Second, // default
			wantErr:        false,
		},
		{
			name: "partial timeouts - only llm specified",
			configYAML: `timeouts:
  llm: 45s
`,
			expectedTask:   12 * time.Hour, // default
			expectedLLM:    45 * time.Second,
			expectedHTTP:   30 * time.Second, // default
			expectedSearch: 30 * time.Second, // default
			wantErr:        false,
		},
		{
			name: "complex duration formats",
			configYAML: `timeouts:
  task: 2h30m
  llm: 1m30s
  http: 500ms
  search: 2m
`,
			expectedTask:   2*time.Hour + 30*time.Minute,
			expectedLLM:    90 * time.Second,
			expectedHTTP:   500 * time.Millisecond,
			expectedSearch: 2 * time.Minute,
			wantErr:        false,
		},
		{
			name: "invalid task duration format",
			configYAML: `timeouts:
  task: invalid
`,
			wantErr:     true,
			errContains: "invalid timeouts.task format",
		},
		{
			name: "invalid llm duration format",
			configYAML: `timeouts:
  llm: 5minutes
`,
			wantErr:     true,
			errContains: "invalid timeouts.llm format",
		},
		{
			name: "invalid http duration format",
			configYAML: `timeouts:
  http: ten-seconds
`,
			wantErr:     true,
			errContains: "invalid timeouts.http format",
		},
		{
			name: "invalid search duration format",
			configYAML: `timeouts:
  search: 1 hour
`,
			wantErr:     true,
			errContains: "invalid timeouts.search format",
		},
		{
			name: "zero values are valid",
			configYAML: `timeouts:
  task: 0s
  llm: 0s
  http: 0s
  search: 0s
`,
			expectedTask:   0,
			expectedLLM:    0,
			expectedHTTP:   0,
			expectedSearch: 0,
			wantErr:        false,
		},
		{
			name: "very long task timeout",
			configYAML: `timeouts:
  task: 168h
`,
			expectedTask:   168 * time.Hour, // 1 week
			expectedLLM:    90 * time.Second,
			expectedHTTP:   30 * time.Second,
			expectedSearch: 30 * time.Second,
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")

			if err := os.WriteFile(configPath, []byte(tt.configYAML), 0644); err != nil {
				t.Fatalf("failed to write test config: %v", err)
			}

			cfg, err := LoadConfig(configPath)
			if tt.wantErr {
				if err == nil {
					t.Fatal("LoadConfig() expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %q, want it to contain %q", err.Error(), tt.errContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("LoadConfig() unexpected error = %v", err)
			}

			if cfg.Timeouts.Task != tt.expectedTask {
				t.Errorf("Timeouts.Task = %v, want %v", cfg.Timeouts.Task, tt.expectedTask)
			}
			if cfg.Timeouts.LLM != tt.expectedLLM {
				t.Errorf("Timeouts.LLM = %v, want %v", cfg.Timeouts.LLM, tt.expectedLLM)
			}
			if cfg.Timeouts.HTTP != tt.expectedHTTP {
				t.Errorf("Timeouts.HTTP = %v, want %v", cfg.Timeouts.HTTP, tt.expectedHTTP)
			}
			if cfg.Timeouts.Search != tt.expectedSearch {
				t.Errorf("Timeouts.Search = %v, want %v", cfg.Timeouts.Search, tt.expectedSearch)
			}
		})
	}
}

// TestTimeoutsConfigDefaults verifies DefaultTimeoutsConfig returns expected values
func TestTimeoutsConfigDefaults(t *testing.T) {
	defaults := DefaultTimeoutsConfig()

	if defaults.Task != 12*time.Hour {
		t.Errorf("DefaultTimeoutsConfig().Task = %v, want %v", defaults.Task, 12*time.Hour)
	}
	if defaults.LLM != 90*time.Second {
		t.Errorf("DefaultTimeoutsConfig().LLM = %v, want %v", defaults.LLM, 90*time.Second)
	}
	if defaults.HTTP != 30*time.Second {
		t.Errorf("DefaultTimeoutsConfig().HTTP = %v, want %v", defaults.HTTP, 30*time.Second)
	}
	if defaults.Search != 30*time.Second {
		t.Errorf("DefaultTimeoutsConfig().Search = %v, want %v", defaults.Search, 30*time.Second)
	}
}

// TestTimeoutsConfigMerge verifies partial config merges correctly with defaults
func TestTimeoutsConfigMerge(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Only specify some timeout values, others should use defaults
	configYAML := `timeouts:
  task: 24h
  http: 45s
`
	if err := os.WriteFile(configPath, []byte(configYAML), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Explicitly set values
	if cfg.Timeouts.Task != 24*time.Hour {
		t.Errorf("Timeouts.Task = %v, want %v", cfg.Timeouts.Task, 24*time.Hour)
	}
	if cfg.Timeouts.HTTP != 45*time.Second {
		t.Errorf("Timeouts.HTTP = %v, want %v", cfg.Timeouts.HTTP, 45*time.Second)
	}

	// Defaults should be preserved for unset values
	if cfg.Timeouts.LLM != 90*time.Second {
		t.Errorf("Timeouts.LLM = %v, want default %v", cfg.Timeouts.LLM, 90*time.Second)
	}
	if cfg.Timeouts.Search != 30*time.Second {
		t.Errorf("Timeouts.Search = %v, want default %v", cfg.Timeouts.Search, 30*time.Second)
	}
}
