package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ConsoleConfig controls terminal output formatting and features
type ConsoleConfig struct {
	// EnableColor enables colored output
	EnableColor bool `yaml:"enable_color"`

	// EnableProgressBar enables progress bar display
	EnableProgressBar bool `yaml:"enable_progress_bar"`

	// EnableTaskDetails enables detailed task information
	EnableTaskDetails bool `yaml:"enable_task_details"`

	// EnableQCFeedback enables quality control feedback display
	EnableQCFeedback bool `yaml:"enable_qc_feedback"`

	// CompactMode enables compact output format
	CompactMode bool `yaml:"compact_mode"`

	// ShowAgentNames shows agent names in output
	ShowAgentNames bool `yaml:"show_agent_names"`

	// ShowFileCounts shows file counts in output
	ShowFileCounts bool `yaml:"show_file_counts"`

	// ShowDurations shows task durations in output
	ShowDurations bool `yaml:"show_durations"`
}

// FeedbackConfig represents feedback storage configuration
type FeedbackConfig struct {
	// StoreInPlanFile stores feedback in plan file
	StoreInPlanFile bool `yaml:"store_in_plan_file"`

	// StoreInDatabase stores feedback in database
	StoreInDatabase bool `yaml:"store_in_database"`

	// Format specifies feedback format (json or plain)
	Format string `yaml:"format"`

	// StoreOnGreen stores feedback on GREEN verdict
	StoreOnGreen bool `yaml:"store_on_green"`

	// StoreOnRed stores feedback on RED verdict
	StoreOnRed bool `yaml:"store_on_red"`

	// StoreOnYellow stores feedback on YELLOW verdict
	StoreOnYellow bool `yaml:"store_on_yellow"`
}

// LearningConfig represents learning system configuration
type LearningConfig struct {
	// Enabled enables the learning system
	Enabled bool `yaml:"enabled"`

	// DBPath is the path to the learning database
	DBPath string `yaml:"db_path"`

	// AutoAdaptAgent enables automatic agent adaptation based on learned patterns
	AutoAdaptAgent bool `yaml:"auto_adapt_agent"`

	// SwapDuringRetries enables agent swapping during retry attempts
	SwapDuringRetries bool `yaml:"swap_during_retries"`

	// EnhancePrompts enables prompt enhancement based on learned patterns
	EnhancePrompts bool `yaml:"enhance_prompts"`

	// QCReadsPlanContext enables QC agent to read plan context
	QCReadsPlanContext bool `yaml:"qc_reads_plan_context"`

	// QCReadsDBContext enables QC agent to read database context
	QCReadsDBContext bool `yaml:"qc_reads_db_context"`

	// MaxContextEntries limits context entries loaded from DB
	MaxContextEntries int `yaml:"max_context_entries"`

	// MinFailuresBeforeAdapt is the minimum number of failures before adapting
	MinFailuresBeforeAdapt int `yaml:"min_failures_before_adapt"`

	// KeepExecutionsDays is the number of days to keep execution history
	KeepExecutionsDays int `yaml:"keep_executions_days"`

	// MaxExecutionsPerTask is the maximum number of executions to keep per task
	MaxExecutionsPerTask int `yaml:"max_executions_per_task"`
}

// QCAgentConfig represents multi-agent QC configuration
type QCAgentConfig struct {
	// Mode specifies agent selection mode: "auto", "explicit", "mixed", or "intelligent"
	Mode string `yaml:"mode"`

	// ExplicitList is the list of agents to use when mode is "explicit"
	ExplicitList []string `yaml:"explicit_list"`

	// AdditionalAgents are extra agents added to auto-selection when mode is "mixed"
	AdditionalAgents []string `yaml:"additional"`

	// BlockedAgents are agents that should never be used (for auto/mixed modes)
	BlockedAgents []string `yaml:"blocked"`

	// Intelligent selection settings (v2.4+)
	// MaxAgents limits the number of agents selected (default: 4)
	MaxAgents int `yaml:"max_agents"`

	// CacheTTLSeconds is how long to cache intelligent selection results (default: 3600)
	CacheTTLSeconds int `yaml:"cache_ttl_seconds"`

	// RequireCodeReview ensures code-reviewer is always included as baseline (default: true)
	RequireCodeReview bool `yaml:"require_code_review"`
}

// QualityControlConfig represents quality control configuration
type QualityControlConfig struct {
	// Enabled enables the quality control system
	Enabled bool `yaml:"enabled"`

	// ReviewAgent is the name of the agent used for quality control reviews
	// DEPRECATED: Use Agents.ExplicitList instead. Kept for backward compatibility.
	ReviewAgent string `yaml:"review_agent"`

	// Agents contains multi-agent QC configuration (v2.2+)
	Agents QCAgentConfig `yaml:"agents"`

	// RetryOnRed is the maximum number of retries when QC review is RED
	RetryOnRed int `yaml:"retry_on_red"`
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

	// Console contains console output configuration
	Console ConsoleConfig `yaml:"console"`

	// Feedback contains feedback storage configuration
	Feedback FeedbackConfig `yaml:"feedback"`

	// Learning contains learning system configuration
	Learning LearningConfig `yaml:"learning"`

	// QualityControl contains quality control configuration
	QualityControl QualityControlConfig `yaml:"quality_control"`
}

// DefaultConsoleConfig returns ConsoleConfig with sensible default values
func DefaultConsoleConfig() ConsoleConfig {
	return ConsoleConfig{
		EnableColor:       true,
		EnableProgressBar: true,
		EnableTaskDetails: true,
		EnableQCFeedback:  true,
		CompactMode:       false,
		ShowAgentNames:    true,
		ShowFileCounts:    true,
		ShowDurations:     true,
	}
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
		Console:        DefaultConsoleConfig(),
		Feedback: FeedbackConfig{
			StoreInPlanFile: true,
			StoreInDatabase: true,
			Format:          "json",
			StoreOnGreen:    true,
			StoreOnRed:      true,
			StoreOnYellow:   true,
		},
		Learning: LearningConfig{
			Enabled:                true,
			DBPath:                 ".conductor/learning/executions.db",
			AutoAdaptAgent:         false,
			SwapDuringRetries:      true,
			EnhancePrompts:         true,
			QCReadsPlanContext:     true,
			QCReadsDBContext:       true,
			MaxContextEntries:      10,
			MinFailuresBeforeAdapt: 2,
			KeepExecutionsDays:     90,
			MaxExecutionsPerTask:   100,
		},
		QualityControl: QualityControlConfig{
			Enabled:     false,
			ReviewAgent: "quality-control", // Deprecated, kept for backward compat
			Agents: QCAgentConfig{
				Mode:              "auto",
				ExplicitList:      []string{},
				AdditionalAgents:  []string{},
				BlockedAgents:     []string{},
				MaxAgents:         4,
				CacheTTLSeconds:   3600,
				RequireCodeReview: true,
			},
			RetryOnRed: 2,
		},
	}
}

// interfaceSliceToStringSlice converts []interface{} to []string
func interfaceSliceToStringSlice(slice []interface{}) []string {
	result := make([]string, 0, len(slice))
	for _, item := range slice {
		if str, ok := item.(string); ok {
			result = append(result, str)
		}
	}
	return result
}

// applyConsoleEnvOverrides applies environment variable overrides to console configuration
// Environment variables take precedence over config file values
// Recognized variables:
//   - CONDUCTOR_CONSOLE_COLOR (enable_color)
//   - CONDUCTOR_CONSOLE_PROGRESS_BAR (enable_progress_bar)
//   - CONDUCTOR_CONSOLE_TASK_DETAILS (enable_task_details)
//   - CONDUCTOR_CONSOLE_QC_FEEDBACK (enable_qc_feedback)
//   - CONDUCTOR_CONSOLE_COMPACT (compact_mode)
//   - CONDUCTOR_CONSOLE_AGENT_NAMES (show_agent_names)
//   - CONDUCTOR_CONSOLE_FILE_COUNTS (show_file_counts)
//   - CONDUCTOR_CONSOLE_DURATIONS (show_durations)
//
// Only "true" (lowercase) or "1" are recognized as true; all other values are false
func applyConsoleEnvOverrides(cfg *ConsoleConfig) {
	if val := os.Getenv("CONDUCTOR_CONSOLE_COLOR"); val != "" {
		cfg.EnableColor = val == "true" || val == "1"
	}
	if val := os.Getenv("CONDUCTOR_CONSOLE_PROGRESS_BAR"); val != "" {
		cfg.EnableProgressBar = val == "true" || val == "1"
	}
	if val := os.Getenv("CONDUCTOR_CONSOLE_TASK_DETAILS"); val != "" {
		cfg.EnableTaskDetails = val == "true" || val == "1"
	}
	if val := os.Getenv("CONDUCTOR_CONSOLE_QC_FEEDBACK"); val != "" {
		cfg.EnableQCFeedback = val == "true" || val == "1"
	}
	if val := os.Getenv("CONDUCTOR_CONSOLE_COMPACT"); val != "" {
		cfg.CompactMode = val == "true" || val == "1"
	}
	if val := os.Getenv("CONDUCTOR_CONSOLE_AGENT_NAMES"); val != "" {
		cfg.ShowAgentNames = val == "true" || val == "1"
	}
	if val := os.Getenv("CONDUCTOR_CONSOLE_FILE_COUNTS"); val != "" {
		cfg.ShowFileCounts = val == "true" || val == "1"
	}
	if val := os.Getenv("CONDUCTOR_CONSOLE_DURATIONS"); val != "" {
		cfg.ShowDurations = val == "true" || val == "1"
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
		// File doesn't exist, return defaults with env overrides applied
		applyConsoleEnvOverrides(&cfg.Console)
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
		MaxConcurrency int                  `yaml:"max_concurrency"`
		Timeout        string               `yaml:"timeout"`
		LogLevel       string               `yaml:"log_level"`
		LogDir         string               `yaml:"log_dir"`
		DryRun         bool                 `yaml:"dry_run"`
		SkipCompleted  bool                 `yaml:"skip_completed"`
		RetryFailed    bool                 `yaml:"retry_failed"`
		Console        ConsoleConfig        `yaml:"console"`
		Feedback       FeedbackConfig       `yaml:"feedback"`
		Learning       LearningConfig       `yaml:"learning"`
		QualityControl QualityControlConfig `yaml:"quality_control"`
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

	// Merge Console, Feedback, and Learning configs - need to check if sections were provided at all
	// We create a temporary unmarshal to detect if sections exist
	var rawMap map[string]interface{}
	if err := yaml.Unmarshal(data, &rawMap); err == nil {
		// Merge Console config if section exists
		if consoleSection, exists := rawMap["console"]; exists && consoleSection != nil {
			console := yamlCfg.Console
			consoleMap, _ := consoleSection.(map[string]interface{})

			if _, exists := consoleMap["enable_color"]; exists {
				cfg.Console.EnableColor = console.EnableColor
			}
			if _, exists := consoleMap["enable_progress_bar"]; exists {
				cfg.Console.EnableProgressBar = console.EnableProgressBar
			}
			if _, exists := consoleMap["enable_task_details"]; exists {
				cfg.Console.EnableTaskDetails = console.EnableTaskDetails
			}
			if _, exists := consoleMap["enable_qc_feedback"]; exists {
				cfg.Console.EnableQCFeedback = console.EnableQCFeedback
			}
			if _, exists := consoleMap["compact_mode"]; exists {
				cfg.Console.CompactMode = console.CompactMode
			}
			if _, exists := consoleMap["show_agent_names"]; exists {
				cfg.Console.ShowAgentNames = console.ShowAgentNames
			}
			if _, exists := consoleMap["show_file_counts"]; exists {
				cfg.Console.ShowFileCounts = console.ShowFileCounts
			}
			if _, exists := consoleMap["show_durations"]; exists {
				cfg.Console.ShowDurations = console.ShowDurations
			}
		}

		// Merge Feedback config
		if feedbackSection, exists := rawMap["feedback"]; exists && feedbackSection != nil {
			feedback := yamlCfg.Feedback
			feedbackMap, _ := feedbackSection.(map[string]interface{})

			if _, exists := feedbackMap["store_in_plan_file"]; exists {
				cfg.Feedback.StoreInPlanFile = feedback.StoreInPlanFile
			}
			if _, exists := feedbackMap["store_in_database"]; exists {
				cfg.Feedback.StoreInDatabase = feedback.StoreInDatabase
			}
			if _, exists := feedbackMap["format"]; exists {
				cfg.Feedback.Format = feedback.Format
			}
			if _, exists := feedbackMap["store_on_green"]; exists {
				cfg.Feedback.StoreOnGreen = feedback.StoreOnGreen
			}
			if _, exists := feedbackMap["store_on_red"]; exists {
				cfg.Feedback.StoreOnRed = feedback.StoreOnRed
			}
			if _, exists := feedbackMap["store_on_yellow"]; exists {
				cfg.Feedback.StoreOnYellow = feedback.StoreOnYellow
			}
		}

		// Merge Learning config
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
			if _, exists := learningMap["swap_during_retries"]; exists {
				cfg.Learning.SwapDuringRetries = learning.SwapDuringRetries
			}
			if _, exists := learningMap["enhance_prompts"]; exists {
				cfg.Learning.EnhancePrompts = learning.EnhancePrompts
			}
			if _, exists := learningMap["qc_reads_plan_context"]; exists {
				cfg.Learning.QCReadsPlanContext = learning.QCReadsPlanContext
			}
			if _, exists := learningMap["qc_reads_db_context"]; exists {
				cfg.Learning.QCReadsDBContext = learning.QCReadsDBContext
			}
			if _, exists := learningMap["max_context_entries"]; exists {
				cfg.Learning.MaxContextEntries = learning.MaxContextEntries
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

		// Merge QualityControl config
		if qcSection, exists := rawMap["quality_control"]; exists && qcSection != nil {
			// QualityControl section exists in YAML, merge it
			qc := yamlCfg.QualityControl
			qcMap, _ := qcSection.(map[string]interface{})

			if _, exists := qcMap["enabled"]; exists {
				cfg.QualityControl.Enabled = qc.Enabled
			}
			if _, exists := qcMap["review_agent"]; exists {
				cfg.QualityControl.ReviewAgent = qc.ReviewAgent
			}
			if _, exists := qcMap["retry_on_red"]; exists {
				cfg.QualityControl.RetryOnRed = qc.RetryOnRed
			}

			// Handle new multi-agent QC configuration (v2.2+)
			if agentsSection, exists := qcMap["agents"]; exists && agentsSection != nil {
				agentsMap, _ := agentsSection.(map[string]interface{})

				if mode, exists := agentsMap["mode"]; exists {
					if modeStr, ok := mode.(string); ok {
						cfg.QualityControl.Agents.Mode = modeStr
					}
				}
				if explicitList, exists := agentsMap["explicit_list"]; exists {
					if list, ok := explicitList.([]interface{}); ok {
						cfg.QualityControl.Agents.ExplicitList = interfaceSliceToStringSlice(list)
					}
				}
				if additional, exists := agentsMap["additional"]; exists {
					if list, ok := additional.([]interface{}); ok {
						cfg.QualityControl.Agents.AdditionalAgents = interfaceSliceToStringSlice(list)
					}
				}
				if blocked, exists := agentsMap["blocked"]; exists {
					if list, ok := blocked.([]interface{}); ok {
						cfg.QualityControl.Agents.BlockedAgents = interfaceSliceToStringSlice(list)
					}
				}
			} else if cfg.QualityControl.ReviewAgent != "" && cfg.QualityControl.Agents.Mode == "auto" {
				// Backward compatibility: if no agents section but review_agent is set,
				// convert to explicit mode with single agent
				cfg.QualityControl.Agents.Mode = "explicit"
				cfg.QualityControl.Agents.ExplicitList = []string{cfg.QualityControl.ReviewAgent}
			}
		}
	}

	// Apply environment variable overrides (highest priority)
	applyConsoleEnvOverrides(&cfg.Console)

	return cfg, nil
}

// LoadConfigFromRootWithBuildTime loads configuration from conductor repo root
// This is the testable version that accepts the build-time injected root
// Priority order:
//  1. Config at {root}/.conductor/config.yaml
//  2. Default configuration
//
// Returns error if root is empty
func LoadConfigFromRootWithBuildTime(buildTimeRoot string) (*Config, error) {
	if buildTimeRoot == "" {
		return nil, fmt.Errorf("conductor repo root not configured: rebuild with conductor repo path injected")
	}

	configPath := filepath.Join(buildTimeRoot, ".conductor", "config.yaml")
	return LoadConfig(configPath)
}

// LoadConfigFromDir loads configuration from .conductor/config.yaml in the conductor repo root
// Uses the build-time injected root (set via SetBuildTimeRepoRoot)
// The dir parameter is IGNORED - kept for backward compatibility only
// If the directory or file doesn't exist, returns default configuration without error
func LoadConfigFromDir(dir string) (*Config, error) {
	return LoadConfigFromRootWithBuildTime(buildTimeRepoRoot)
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

	// Console configuration is all boolean flags with sensible defaults,
	// no additional validation needed

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

	// Validate quality control configuration
	if c.QualityControl.Enabled {
		// Validate retry count
		if c.QualityControl.RetryOnRed < 0 {
			return fmt.Errorf("quality_control.retry_on_red must be >= 0, got %d", c.QualityControl.RetryOnRed)
		}

		// Validate multi-agent QC configuration
		validModes := map[string]bool{
			"auto":        true,
			"explicit":    true,
			"mixed":       true,
			"intelligent": true,
		}
		if !validModes[c.QualityControl.Agents.Mode] {
			return fmt.Errorf("quality_control.agents.mode must be one of: auto, explicit, mixed, intelligent; got %q", c.QualityControl.Agents.Mode)
		}

		// If mode is explicit, must have at least one agent
		if c.QualityControl.Agents.Mode == "explicit" && len(c.QualityControl.Agents.ExplicitList) == 0 {
			// No fallback to deprecated ReviewAgent - this is a configuration error
			return fmt.Errorf("quality_control.agents.explicit_list cannot be empty when mode is 'explicit'. Provide at least one agent name, or switch to mode 'auto'")
		}

		// Validate agent names are non-empty strings
		for i, agent := range c.QualityControl.Agents.ExplicitList {
			trimmed := strings.TrimSpace(agent)
			if trimmed == "" {
				return fmt.Errorf("quality_control.agents.explicit_list[%d] cannot be empty", i)
			}
		}
		for i, agent := range c.QualityControl.Agents.AdditionalAgents {
			trimmed := strings.TrimSpace(agent)
			if trimmed == "" {
				return fmt.Errorf("quality_control.agents.additional[%d] cannot be empty", i)
			}
		}
		for i, agent := range c.QualityControl.Agents.BlockedAgents {
			trimmed := strings.TrimSpace(agent)
			if trimmed == "" {
				return fmt.Errorf("quality_control.agents.blocked[%d] cannot be empty", i)
			}
		}
	}

	return nil
}
