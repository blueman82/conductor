package config

import (
	"os"
	"path/filepath"
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

	cfg, err := LoadConfigFromDir(tmpDir)
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

	cfg, err := LoadConfigFromDir(tmpDir)
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
	if cfg.Learning.DBPath != ".conductor/learning" {
		t.Errorf("Learning.DBPath = %q, want %q", cfg.Learning.DBPath, ".conductor/learning")
	}
	if cfg.Learning.AutoAdaptAgent {
		t.Errorf("Learning.AutoAdaptAgent = %v, want false", cfg.Learning.AutoAdaptAgent)
	}
	if !cfg.Learning.EnhancePrompts {
		t.Errorf("Learning.EnhancePrompts = %v, want true", cfg.Learning.EnhancePrompts)
	}
	if cfg.Learning.MinFailuresBeforeAdapt != 2 {
		t.Errorf("Learning.MinFailuresBeforeAdapt = %d, want 2", cfg.Learning.MinFailuresBeforeAdapt)
	}
	if cfg.Learning.KeepExecutionsDays != 90 {
		t.Errorf("Learning.KeepExecutionsDays = %d, want 90", cfg.Learning.KeepExecutionsDays)
	}
	if cfg.Learning.MaxExecutionsPerTask != 100 {
		t.Errorf("Learning.MaxExecutionsPerTask = %d, want 100", cfg.Learning.MaxExecutionsPerTask)
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
	if cfg.Learning.DBPath != ".conductor/learning" {
		t.Errorf("Learning.DBPath = %q, want %q (default)", cfg.Learning.DBPath, ".conductor/learning")
	}
}

// TestConfig_LearningCustomPath tests loading config with custom learning settings
func TestConfig_LearningCustomPath(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `learning:
  enabled: true
  db_path: /custom/path/learning.db
  auto_adapt_agent: true
  enhance_prompts: false
  min_failures_before_adapt: 5
  keep_executions_days: 30
  max_executions_per_task: 50
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
	if !cfg.Learning.AutoAdaptAgent {
		t.Errorf("Learning.AutoAdaptAgent = %v, want true", cfg.Learning.AutoAdaptAgent)
	}
	if cfg.Learning.EnhancePrompts {
		t.Errorf("Learning.EnhancePrompts = %v, want false", cfg.Learning.EnhancePrompts)
	}
	if cfg.Learning.MinFailuresBeforeAdapt != 5 {
		t.Errorf("Learning.MinFailuresBeforeAdapt = %d, want 5", cfg.Learning.MinFailuresBeforeAdapt)
	}
	if cfg.Learning.KeepExecutionsDays != 30 {
		t.Errorf("Learning.KeepExecutionsDays = %d, want 30", cfg.Learning.KeepExecutionsDays)
	}
	if cfg.Learning.MaxExecutionsPerTask != 50 {
		t.Errorf("Learning.MaxExecutionsPerTask = %d, want 50", cfg.Learning.MaxExecutionsPerTask)
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
	if cfg.Learning.DBPath != ".conductor/learning" {
		t.Errorf("Learning.DBPath = %q, want %q (default)", cfg.Learning.DBPath, ".conductor/learning")
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
  db_path: .conductor/learning
  min_failures_before_adapt: 2
  keep_executions_days: 90
  max_executions_per_task: 100
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
			name: "negative min_failures_before_adapt",
			config: `learning:
  enabled: true
  min_failures_before_adapt: -1
`,
			wantError: true,
		},
		{
			name: "zero min_failures_before_adapt",
			config: `learning:
  enabled: true
  min_failures_before_adapt: 0
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
		{
			name: "negative max_executions_per_task",
			config: `learning:
  enabled: true
  max_executions_per_task: -1
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
