package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// LearningConfig represents learning system configuration
type LearningConfig struct {
	// Enabled enables the learning system
	Enabled bool `yaml:"enabled"`

	// DBPath is the path to the learning database
	DBPath string `yaml:"db_path"`

	// AutoAdaptAgent enables automatic agent adaptation based on learned patterns
	AutoAdaptAgent bool `yaml:"auto_adapt_agent"`

	// EnhancePrompts enables prompt enhancement based on learned patterns
	EnhancePrompts bool `yaml:"enhance_prompts"`

	// MinFailuresBeforeAdapt is the minimum number of failures before adapting
	MinFailuresBeforeAdapt int `yaml:"min_failures_before_adapt"`

	// KeepExecutionsDays is the number of days to keep execution history
	KeepExecutionsDays int `yaml:"keep_executions_days"`

	// MaxExecutionsPerTask is the maximum number of executions to keep per task
	MaxExecutionsPerTask int `yaml:"max_executions_per_task"`
}

// Config represents conductor configuration options
type Config struct {
	// MaxConcurrency is the maximum number of concurrent tasks (0 = unlimited)
	MaxConcurrency int `yaml:"max_concurrency"`

	// Timeout is the maximum execution time for tasks
	Timeout time.Duration `yaml:"timeout"`

	// LogLevel sets the logging verbosity (trace, debug, info, warn, error)
	LogLevel string `yaml:"log_level"`

	// LogDir is the directory where logs will be written
	LogDir string `yaml:"log_dir"`

	// DryRun enables validation-only mode without execution
	DryRun bool `yaml:"dry_run"`

	// SkipCompleted skips tasks that have already been completed
	SkipCompleted bool `yaml:"skip_completed"`

	// RetryFailed retries tasks that failed
	RetryFailed bool `yaml:"retry_failed"`

	// Learning contains learning system configuration
	Learning LearningConfig `yaml:"learning"`
}

// DefaultConfig returns a Config with sensible default values
func DefaultConfig() *Config {
	return &Config{
		MaxConcurrency: 0,              // Unlimited
		Timeout:        10 * time.Hour, // 10 hours
		LogLevel:       "info",
		LogDir:         ".conductor/logs",
		DryRun:         false,
		SkipCompleted:  false,
		RetryFailed:    false,
		Learning: LearningConfig{
			Enabled:                true,
			DBPath:                 ".conductor/learning",
			AutoAdaptAgent:         false,
			EnhancePrompts:         true,
			MinFailuresBeforeAdapt: 2,
			KeepExecutionsDays:     90,
			MaxExecutionsPerTask:   100,
		},
	}
}

// LoadConfig loads configuration from the specified file path
// If the file doesn't exist, returns default configuration without error
// If the file exists but is malformed, returns an error
func LoadConfig(path string) (*Config, error) {
	// Start with defaults
	cfg := DefaultConfig()

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// File doesn't exist, return defaults (not an error)
		return cfg, nil
	}

	// Read the file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	// Use a temporary struct to handle duration parsing
	type yamlConfig struct {
		MaxConcurrency int            `yaml:"max_concurrency"`
		Timeout        string         `yaml:"timeout"`
		LogLevel       string         `yaml:"log_level"`
		LogDir         string         `yaml:"log_dir"`
		DryRun         bool           `yaml:"dry_run"`
		SkipCompleted  bool           `yaml:"skip_completed"`
		RetryFailed    bool           `yaml:"retry_failed"`
		Learning       LearningConfig `yaml:"learning"`
	}

	var yamlCfg yamlConfig
	if err := yaml.Unmarshal(data, &yamlCfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply non-zero values from file (merging with defaults)
	if yamlCfg.MaxConcurrency != 0 {
		cfg.MaxConcurrency = yamlCfg.MaxConcurrency
	}
	if yamlCfg.Timeout != "" {
		timeout, err := time.ParseDuration(yamlCfg.Timeout)
		if err != nil {
			return nil, fmt.Errorf("invalid timeout format %q: %w", yamlCfg.Timeout, err)
		}
		cfg.Timeout = timeout
	}
	if yamlCfg.LogLevel != "" {
		cfg.LogLevel = yamlCfg.LogLevel
	}
	if yamlCfg.LogDir != "" {
		cfg.LogDir = yamlCfg.LogDir
	}
	// DryRun is explicitly set if present in YAML
	if yamlCfg.DryRun {
		cfg.DryRun = yamlCfg.DryRun
	}
	// SkipCompleted is explicitly set if present in YAML
	if yamlCfg.SkipCompleted {
		cfg.SkipCompleted = yamlCfg.SkipCompleted
	}
	// RetryFailed is explicitly set if present in YAML
	if yamlCfg.RetryFailed {
		cfg.RetryFailed = yamlCfg.RetryFailed
	}

	// Merge Learning config - need to check if the section was provided at all
	// We create a temporary unmarshal to detect if learning section exists
	var rawMap map[string]interface{}
	if err := yaml.Unmarshal(data, &rawMap); err == nil {
		if learningSection, exists := rawMap["learning"]; exists && learningSection != nil {
			// Learning section exists in YAML, merge it
			learning := yamlCfg.Learning

			// For nested struct, we need to check which fields were actually set
			// If any field is non-zero, we assume it was explicitly set
			learningMap, _ := learningSection.(map[string]interface{})

			if _, exists := learningMap["enabled"]; exists {
				cfg.Learning.Enabled = learning.Enabled
			}
			if _, exists := learningMap["db_path"]; exists {
				// Explicitly set db_path, even if empty string
				cfg.Learning.DBPath = learning.DBPath
			}
			if _, exists := learningMap["auto_adapt_agent"]; exists {
				cfg.Learning.AutoAdaptAgent = learning.AutoAdaptAgent
			}
			if _, exists := learningMap["enhance_prompts"]; exists {
				cfg.Learning.EnhancePrompts = learning.EnhancePrompts
			}
			if _, exists := learningMap["min_failures_before_adapt"]; exists {
				cfg.Learning.MinFailuresBeforeAdapt = learning.MinFailuresBeforeAdapt
			}
			if _, exists := learningMap["keep_executions_days"]; exists {
				cfg.Learning.KeepExecutionsDays = learning.KeepExecutionsDays
			}
			if _, exists := learningMap["max_executions_per_task"]; exists {
				cfg.Learning.MaxExecutionsPerTask = learning.MaxExecutionsPerTask
			}
		}
	}

	return cfg, nil
}

// LoadConfigFromDir loads configuration from .conductor/config.yaml in the specified directory
// If the directory or file doesn't exist, returns default configuration without error
func LoadConfigFromDir(dir string) (*Config, error) {
	configPath := filepath.Join(dir, ".conductor", "config.yaml")
	return LoadConfig(configPath)
}

// MergeWithFlags merges CLI flags into the configuration
// Non-nil flag values override configuration values
// This allows CLI flags to take precedence over config file settings
func (c *Config) MergeWithFlags(maxConcurrency *int, timeout *time.Duration, logDir *string, dryRun *bool, skipCompleted *bool, retryFailed *bool) {
	if maxConcurrency != nil {
		c.MaxConcurrency = *maxConcurrency
	}
	if timeout != nil {
		c.Timeout = *timeout
	}
	if logDir != nil {
		c.LogDir = *logDir
	}
	if dryRun != nil {
		c.DryRun = *dryRun
	}
	if skipCompleted != nil {
		c.SkipCompleted = *skipCompleted
	}
	if retryFailed != nil {
		c.RetryFailed = *retryFailed
	}
}

// Validate validates the configuration values
// Returns an error if any values are invalid
func (c *Config) Validate() error {
	// Validate max_concurrency
	if c.MaxConcurrency < 0 {
		return fmt.Errorf("max_concurrency must be >= 0, got %d", c.MaxConcurrency)
	}

	// Validate log_level
	validLevels := map[string]bool{
		"trace": true,
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLevels[c.LogLevel] {
		return fmt.Errorf("invalid log_level %q, must be one of: trace, debug, info, warn, error", c.LogLevel)
	}

	// Timeout can be 0 (no timeout) or positive, negative is invalid
	if c.Timeout < 0 {
		return fmt.Errorf("timeout must be >= 0, got %v", c.Timeout)
	}

	// Validate learning configuration
	if c.Learning.Enabled {
		if c.Learning.DBPath == "" {
			return fmt.Errorf("learning.db_path cannot be empty when learning is enabled")
		}
		if c.Learning.MinFailuresBeforeAdapt <= 0 {
			return fmt.Errorf("learning.min_failures_before_adapt must be > 0, got %d", c.Learning.MinFailuresBeforeAdapt)
		}
		if c.Learning.KeepExecutionsDays < 0 {
			return fmt.Errorf("learning.keep_executions_days must be >= 0, got %d", c.Learning.KeepExecutionsDays)
		}
		if c.Learning.MaxExecutionsPerTask < 0 {
			return fmt.Errorf("learning.max_executions_per_task must be >= 0, got %d", c.Learning.MaxExecutionsPerTask)
		}
	}

	return nil
}
