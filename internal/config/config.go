package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

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
}

// DefaultConfig returns a Config with sensible default values
func DefaultConfig() *Config {
	return &Config{
		MaxConcurrency: 0,              // Unlimited
		Timeout:        10 * time.Hour, // 10 hours
		LogLevel:       "info",
		LogDir:         ".conductor/logs",
		DryRun:         false,
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
		MaxConcurrency int    `yaml:"max_concurrency"`
		Timeout        string `yaml:"timeout"`
		LogLevel       string `yaml:"log_level"`
		LogDir         string `yaml:"log_dir"`
		DryRun         bool   `yaml:"dry_run"`
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
func (c *Config) MergeWithFlags(maxConcurrency *int, timeout *time.Duration, logDir *string, dryRun *bool) {
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

	return nil
}
