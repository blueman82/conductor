package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// PatternMode specifies the Pattern Intelligence operating mode
type PatternMode string

const (
	// PatternModeBlock fails tasks with high similarity to existing work
	PatternModeBlock PatternMode = "block"

	// PatternModeWarn logs a warning but allows task execution to proceed
	PatternModeWarn PatternMode = "warn"

	// PatternModeSuggest includes pattern analysis in agent prompt without blocking
	PatternModeSuggest PatternMode = "suggest"
)

// AnomalyDetectionConfig controls real-time anomaly detection during wave execution
type AnomalyDetectionConfig struct {
	// Enabled enables real-time anomaly detection during wave execution
	Enabled bool `yaml:"enabled"`

	// ConsecutiveFailureThreshold triggers alert after N consecutive task failures (default: 3)
	ConsecutiveFailureThreshold int `yaml:"consecutive_failure_threshold"`

	// ErrorRateThreshold triggers alert when wave error rate exceeds this percentage (0.0-1.0, default: 0.5)
	ErrorRateThreshold float64 `yaml:"error_rate_threshold"`

	// DurationDeviationThreshold triggers alert when task duration exceeds N times the estimate (default: 2.0)
	DurationDeviationThreshold float64 `yaml:"duration_deviation_threshold"`
}

// LearningConfig represents learning system configuration
type LearningConfig struct {
	// Enabled enables the learning system
	Enabled bool `yaml:"enabled"`

	// DBPath is the path to the learning database
	DBPath string `yaml:"db_path"`

	// SwapDuringRetries enables agent swapping during retry attempts using IntelligentAgentSwapper
	SwapDuringRetries bool `yaml:"swap_during_retries"`

	// WarmUpEnabled enables warm-up context injection for agent priming (v2.29+)
	// When true, agents receive similar successful task approaches, common pitfalls,
	// and file-specific patterns before task execution.
	WarmUpEnabled bool `yaml:"warmup_enabled"`

	// KeepExecutionsDays is the number of days to keep execution history
	KeepExecutionsDays int `yaml:"keep_executions_days"`

	// MinFailuresBeforeAdapt is the minimum consecutive failures before considering agent swap
	MinFailuresBeforeAdapt int `yaml:"min_failures_before_adapt"`
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

	// DefaultAgent is the default QC agent to use when mode is "auto" (configurable via env: CONDUCTOR_QC_DEFAULT_AGENT)
	DefaultAgent string `yaml:"default_agent" env:"CONDUCTOR_QC_DEFAULT_AGENT"`

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

	// Agents contains multi-agent QC configuration (v2.2+)
	Agents QCAgentConfig `yaml:"agents"`

	// RetryOnRed is the maximum number of retries when QC review is RED
	RetryOnRed int `yaml:"retry_on_red"`
}

// CostModelConfig represents Claude model token pricing configuration
type CostModelConfig struct {
	// SonnetInput is the cost per 1M input tokens for Claude Sonnet
	SonnetInput float64 `yaml:"sonnet_input"`

	// SonnetOutput is the cost per 1M output tokens for Claude Sonnet
	SonnetOutput float64 `yaml:"sonnet_output"`

	// HaikuInput is the cost per 1M input tokens for Claude Haiku
	HaikuInput float64 `yaml:"haiku_input"`

	// HaikuOutput is the cost per 1M output tokens for Claude Haiku
	HaikuOutput float64 `yaml:"haiku_output"`

	// OpusInput is the cost per 1M input tokens for Claude Opus
	OpusInput float64 `yaml:"opus_input"`

	// OpusOutput is the cost per 1M output tokens for Claude Opus
	OpusOutput float64 `yaml:"opus_output"`
}

// AgentWatchConfig represents Agent Watch behavioral analytics configuration
type AgentWatchConfig struct {
	// Enabled enables Agent Watch behavioral analytics
	Enabled bool `yaml:"enabled"`

	// BaseDir is the base directory for discovering Claude Code session files
	// Default: ~/.claude/projects
	BaseDir string `yaml:"base_dir"`

	// DefaultLimit is the default number of sessions/entries to display in observe commands
	DefaultLimit int `yaml:"default_limit"`

	// PollIntervalSecs is the poll interval in seconds for real-time activity streaming
	PollIntervalSecs int `yaml:"poll_interval_secs"`

	// AutoImport enables automatic import of new session data
	AutoImport bool `yaml:"auto_import"`

	// CacheSize is the maximum number of sessions to cache in memory
	CacheSize int `yaml:"cache_size"`

	// CostModel contains Claude model pricing configuration
	CostModel CostModelConfig `yaml:"cost_model"`
}

// ValidationConfig controls plan validation behavior
type ValidationConfig struct {
	// KeyPointCriteria sets how to handle key_point vs success_criteria misalignment:
	// "warn" (default) logs a warning, "strict" fails validation, "off" disables the check
	KeyPointCriteria string `yaml:"key_point_criteria"`

	// StrictRubric enables strict rubric validation when PlannerComplianceSpec is present.
	// When true, validates terminology alignment, task type constraints, and documentation targets.
	// Default: false (disabled)
	StrictRubric bool `yaml:"strict_rubric"`
}

// BudgetConfig controls usage budget tracking and rate limit handling
type BudgetConfig struct {
	// Enabled enables budget tracking (default: false for zero behavior change)
	Enabled bool `yaml:"enabled"`

	// Intelligent Rate Limit Auto-Resume (v2.20+)
	// AutoResume enables intelligent wait/exit on rate limit detection
	AutoResume bool `yaml:"auto_resume"`

	// MaxWaitDuration is the maximum time to wait for rate limit reset (default: 6h)
	MaxWaitDuration time.Duration `yaml:"max_wait_duration"`

	// AnnounceInterval is the countdown announcement interval (default: 15m)
	AnnounceInterval time.Duration `yaml:"announce_interval"`

	// SafetyBuffer is extra wait time after reset (default: 60s)
	SafetyBuffer time.Duration `yaml:"safety_buffer"`
}

// PatternConfig represents Pattern Intelligence configuration
type PatternConfig struct {
	// Enabled enables the Pattern Intelligence system
	Enabled bool `yaml:"enabled"`

	// Mode specifies the operating mode: "block", "warn", or "suggest"
	Mode PatternMode `yaml:"mode"`

	// SimilarityThreshold is the minimum similarity score to trigger action (0.0-1.0)
	// Tasks with similarity >= threshold will be flagged based on mode
	SimilarityThreshold float64 `yaml:"similarity_threshold"`

	// DuplicateThreshold is the minimum score to consider a task a duplicate (0.0-1.0)
	// Tasks with duplicate score >= threshold may be skipped in block mode
	DuplicateThreshold float64 `yaml:"duplicate_threshold"`

	// MinConfidence is the minimum confidence required before taking action (0.0-1.0)
	// Predictions with confidence < threshold are treated as suggestions only
	MinConfidence float64 `yaml:"min_confidence"`

	// EnableSTOP enables STOP protocol analysis (Search/Think/Outline/Prove)
	EnableSTOP bool `yaml:"enable_stop"`

	// EnableDuplicateDetection enables duplicate task detection
	EnableDuplicateDetection bool `yaml:"enable_duplicate_detection"`

	// InjectIntoPrompt includes pattern analysis in agent prompts
	InjectIntoPrompt bool `yaml:"inject_into_prompt"`

	// MaxPatternsPerTask limits patterns included in prompt injection
	MaxPatternsPerTask int `yaml:"max_patterns_per_task"`

	// MaxRelatedFiles limits related files included in analysis
	MaxRelatedFiles int `yaml:"max_related_files"`

	// LLM Enhancement (optional, requires Claude CLI)
	// LLMEnhancementEnabled enables Claude-based confidence refinement for uncertain cases
	LLMEnhancementEnabled bool `yaml:"llm_enhancement_enabled"`

	// RequireJustification requires implementing agent to justify custom implementations
	// when STOP protocol finds prior art (existing solutions, similar commits, related issues).
	// When true: QC agents will ask for justification; weak/missing justification → YELLOW.
	// Default: false (disabled for backward compatibility)
	RequireJustification bool `yaml:"require_justification"`
}

// MetricsConfig controls execution metrics collection (v3.4+)
type MetricsConfig struct {
	// LOCTracking enables lines-of-code tracking per task (default: true)
	LOCTracking bool `yaml:"loc_tracking"`

	// HumanEstimation enables human time estimation per task (default: false)
	// When enabled, uses Claude haiku to estimate how long a human developer would take.
	// Requires Claude CLI calls, so disabled by default.
	HumanEstimation bool `yaml:"human_estimation"`
}

// TimeoutsConfig controls timeout durations for different operation types
type TimeoutsConfig struct {
	// Task is the timeout for main agent task execution (default: 12h)
	// This is the maximum time allowed for an agent to complete a task.
	Task time.Duration `yaml:"task"`

	// LLM is the timeout for internal Claude CLI calls (default: 90s)
	// Used for agent swapping, QC selection, similarity checks, etc.
	// 90s provides headroom for complex operations while still working for simple ones.
	// Users can decrease to 60s or 30s for faster failure on simple operations.
	LLM time.Duration `yaml:"llm"`

	// HTTP is the timeout for external HTTP services (default: 30s)
	// Used for TTS, webhooks, and other external HTTP calls.
	HTTP time.Duration `yaml:"http"`

	// Search is the timeout for fast local CLI operations (default: 30s)
	// Used for git, grep, and other local search operations.
	Search time.Duration `yaml:"search"`
}

// TTSConfig controls text-to-speech functionality
type TTSConfig struct {
	// Enabled enables TTS functionality (default: false for zero behavior change)
	Enabled bool `yaml:"enabled"`

	// BaseURL is the TTS server endpoint
	BaseURL string `yaml:"base_url"`

	// Model is the TTS model to use
	Model string `yaml:"model"`

	// Voice is the TTS voice to use
	Voice string `yaml:"voice"`
}

// SetupConfig controls pre-wave setup phase functionality
type SetupConfig struct {
	// Enabled enables the setup phase (default: false for zero behavior change)
	Enabled bool `yaml:"enabled"`
}

// RollbackMode specifies when to perform automatic rollback
type RollbackMode string

const (
	// RollbackModeManual only rolls back when explicitly requested
	RollbackModeManual RollbackMode = "manual"

	// RollbackModeAutoOnRed automatically rolls back on RED verdict
	RollbackModeAutoOnRed RollbackMode = "auto_on_red"

	// RollbackModeAutoOnMaxRetries automatically rolls back after max retries exhausted
	RollbackModeAutoOnMaxRetries RollbackMode = "auto_on_max_retries"
)

// RollbackConfig controls git checkpoint and rollback functionality (v3.2+)
type RollbackConfig struct {
	// Enabled enables the checkpoint/rollback feature (default: false for zero behavior change)
	Enabled bool `yaml:"enabled"`

	// Mode specifies when to rollback: "manual", "auto_on_red", "auto_on_max_retries" (default: auto_on_max_retries)
	Mode RollbackMode `yaml:"mode"`

	// RequireCleanState blocks execution if uncommitted changes exist (default: true)
	RequireCleanState bool `yaml:"require_clean_state"`

	// ProtectedBranches are branches that trigger working branch creation (default: [main, master, develop])
	ProtectedBranches []string `yaml:"protected_branches"`

	// WorkingBranchPrefix is the prefix for working branches (default: "conductor-run/")
	WorkingBranchPrefix string `yaml:"working_branch_prefix"`

	// CheckpointPrefix is the prefix for checkpoint branches (default: "conductor-checkpoint-")
	CheckpointPrefix string `yaml:"checkpoint_prefix"`

	// KeepCheckpointDays is the number of days to keep old checkpoint branches (default: 7)
	KeepCheckpointDays int `yaml:"keep_checkpoint_days"`
}

// ExecutorConfig controls task execution behavior
type ExecutorConfig struct {
	// EnforceDependencyChecks enables running dependency check commands before task invocation.
	// When true (default for plans with RuntimeMetadata), dependency checks are executed
	// and task execution is aborted if any check fails.
	// Default: true
	EnforceDependencyChecks bool `yaml:"enforce_dependency_checks"`

	// EnforceTestCommands enables running test commands after agent output but before QC.
	// When true (default), test commands are executed and task fails immediately if any
	// command exits non-zero (blocks QC review).
	// Default: true
	EnforceTestCommands bool `yaml:"enforce_test_commands"`

	// VerifyCriteria enables running optional per-criterion verification commands.
	// When true (default), verification blocks on StructuredCriteria are executed
	// and results are injected into QC prompt for final judgment.
	// Default: true
	VerifyCriteria bool `yaml:"verify_criteria"`

	// EnforcePackageGuard enables runtime package conflict guard for Go packages.
	// When true, prevents concurrent task execution from modifying the same Go package.
	// Tasks must acquire package locks before execution; conflicts block until released.
	// Default: true
	EnforcePackageGuard bool `yaml:"enforce_package_guard"`

	// EnforceDocTargets enables documentation target verification for documentation tasks.
	// When true (default), documentation targets are verified before QC to ensure
	// agents edit the exact sections specified in the plan.
	// Default: true
	EnforceDocTargets bool `yaml:"enforce_doc_targets"`

	// EnableErrorPatternDetection enables error pattern detection on test failures.
	// When true (default), Conductor will analyze test command failures and categorize them
	// into CODE_LEVEL (agent can fix), PLAN_LEVEL (plan needs update), or ENV_LEVEL
	// (environment issue). Provides actionable suggestions for each category.
	// Default: true
	EnableErrorPatternDetection bool `yaml:"enable_error_pattern_detection"`

	// EnableClaudeClassification enables Claude-based error classification (v3.0+).
	// When true and EnableErrorPatternDetection is also true, uses Claude API to
	// semantically analyze test failures instead of regex patterns.
	// Falls back to regex patterns if Claude classification fails or confidence is low (<0.85).
	// This feature is opt-in and disabled by default for backward compatibility.
	// Default: false
	EnableClaudeClassification bool `yaml:"enable_claude_classification"`

	// IntelligentAgentSelection enables Claude-based agent selection for task execution (v2.15+).
	// When true and task.Agent is empty, uses Claude to analyze the task and select
	// the most appropriate agent from the registry based on task context, file types,
	// and success criteria. Falls back to general-purpose if selection fails.
	// Also automatically enabled when quality_control.agents.mode is "intelligent".
	// Default: false
	IntelligentAgentSelection bool `yaml:"intelligent_agent_selection"`
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

	// QualityControl contains quality control configuration
	QualityControl QualityControlConfig `yaml:"quality_control"`

	// AgentWatch contains Agent Watch behavioral analytics configuration
	AgentWatch AgentWatchConfig `yaml:"agent_watch"`

	// Validation controls plan validation strictness
	Validation ValidationConfig `yaml:"validation"`

	// Executor controls task execution behavior
	Executor ExecutorConfig `yaml:"executor"`

	// TTS controls text-to-speech functionality
	TTS TTSConfig `yaml:"tts"`

	// Setup controls pre-wave setup phase functionality
	Setup SetupConfig `yaml:"setup"`

	// Rollback controls git checkpoint and rollback functionality (v3.2+)
	Rollback RollbackConfig `yaml:"rollback"`

	// Budget controls usage budget tracking and enforcement
	Budget BudgetConfig `yaml:"budget"`

	// Pattern contains Pattern Intelligence configuration
	Pattern PatternConfig `yaml:"pattern"`

	// Architecture contains Architecture Checkpoint configuration (v2.27+)
	Architecture ArchitectureConfig `yaml:"architecture"`

	// Timeouts contains timeout configuration for different operation types
	Timeouts TimeoutsConfig `yaml:"timeouts"`

	// Metrics controls execution metrics collection (v3.4+)
	Metrics MetricsConfig `yaml:"metrics"`
}

// ArchitectureMode specifies the Architecture Checkpoint operating mode
type ArchitectureMode string

const (
	// ArchitectureModeBlock fails tasks flagged for architectural review
	ArchitectureModeBlock ArchitectureMode = "block"

	// ArchitectureModeEscalate logs warning, injects context, and continues
	ArchitectureModeEscalate ArchitectureMode = "escalate"
)

// ArchitectureConfig controls Architecture Checkpoint Gate behavior (v2.27+)
type ArchitectureConfig struct {
	// Enabled enables Architecture Checkpoint Gate
	Enabled bool `yaml:"enabled"`

	// Mode specifies operating mode: "block" or "escalate"
	Mode ArchitectureMode `yaml:"mode"`

	// RequireJustification requires task to justify architectural changes in output
	RequireJustification bool `yaml:"require_justification"`

	// EscalateOnUncertain escalates when Claude confidence < threshold
	EscalateOnUncertain bool `yaml:"escalate_on_uncertain"`

	// ConfidenceThreshold for escalation (default: 0.7)
	ConfidenceThreshold float64 `yaml:"confidence_threshold"`
}

// DefaultAnomalyDetectionConfig returns AnomalyDetectionConfig with sensible default values
func DefaultAnomalyDetectionConfig() AnomalyDetectionConfig {
	return AnomalyDetectionConfig{
		Enabled:                     true, // Enabled by default when GUARD is enabled
		ConsecutiveFailureThreshold: 3,
		ErrorRateThreshold:          0.5,
		DurationDeviationThreshold:  2.0,
	}
}

// DefaultCostModelConfig returns CostModelConfig with current Claude API pricing
// Prices are per 1M tokens as of the build date
func DefaultCostModelConfig() CostModelConfig {
	return CostModelConfig{
		SonnetInput:  3.0,
		SonnetOutput: 15.0,
		HaikuInput:   0.80,
		HaikuOutput:  4.0,
		OpusInput:    15.0,
		OpusOutput:   75.0,
	}
}

// DefaultTimeoutsConfig returns TimeoutsConfig with sensible default values.
// These defaults provide generous timeouts while still allowing timely failure detection.
func DefaultTimeoutsConfig() TimeoutsConfig {
	return TimeoutsConfig{
		Task:   12 * time.Hour,   // Main agent task execution
		LLM:    90 * time.Second, // Internal Claude CLI calls (agent swap, QC, similarity)
		HTTP:   30 * time.Second, // External HTTP services (TTS, webhooks)
		Search: 30 * time.Second, // Fast local CLI operations (git, grep)
	}
}

// DefaultTTSConfig returns TTSConfig with sensible default values
// TTS is DISABLED by default to ensure zero behavior change unless explicitly enabled
func DefaultTTSConfig() TTSConfig {
	return TTSConfig{
		Enabled: false,
		BaseURL: "http://localhost:5005",
		Model:   "orpheus",
		Voice:   "tara",
	}
}

// DefaultSetupConfig returns SetupConfig with sensible default values
// Setup is DISABLED by default to ensure zero behavior change unless explicitly enabled
func DefaultSetupConfig() SetupConfig {
	return SetupConfig{
		Enabled: false,
	}
}

// DefaultRollbackConfig returns RollbackConfig with sensible default values
// Rollback is DISABLED by default to ensure zero behavior change unless explicitly enabled
func DefaultRollbackConfig() RollbackConfig {
	return RollbackConfig{
		Enabled:             false,
		Mode:                RollbackModeAutoOnMaxRetries,
		RequireCleanState:   true,
		ProtectedBranches:   []string{"main", "master", "develop"},
		WorkingBranchPrefix: "conductor-run/",
		CheckpointPrefix:    "conductor-checkpoint-",
		KeepCheckpointDays:  7,
	}
}

// DefaultBudgetConfig returns BudgetConfig with sensible default values
// Budget is DISABLED by default to ensure zero behavior change unless explicitly enabled
func DefaultBudgetConfig() BudgetConfig {
	return BudgetConfig{
		Enabled:          false,
		AutoResume:       true,             // Enabled by default
		MaxWaitDuration:  6 * time.Hour,    // 6 hours
		AnnounceInterval: 15 * time.Minute, // 15 minutes for TTS announcements
		SafetyBuffer:     60 * time.Second, // 60 seconds
	}
}

// DefaultPatternConfig returns PatternConfig with sensible default values.
// Pattern Intelligence is DISABLED by default to ensure zero behavior change unless explicitly enabled.
func DefaultPatternConfig() PatternConfig {
	return PatternConfig{
		Enabled:                  false, // Disabled by default for zero behavior change
		Mode:                     PatternModeWarn,
		SimilarityThreshold:      0.8,   // 80% similarity triggers action
		DuplicateThreshold:       0.9,   // 90% similarity for duplicate detection
		MinConfidence:            0.7,   // 70% confidence required
		EnableSTOP:               true,  // STOP analysis enabled when system is enabled
		EnableDuplicateDetection: true,  // Duplicate detection enabled when system is enabled
		InjectIntoPrompt:         true,  // Include analysis in prompts by default
		MaxPatternsPerTask:       5,     // Limit patterns to avoid prompt bloat
		MaxRelatedFiles:          10,    // Limit related files
		LLMEnhancementEnabled:    false, // Disabled by default
	}
}

// DefaultMetricsConfig returns MetricsConfig with sensible default values.
// LOC tracking is ENABLED by default.
// Human estimation is DISABLED by default (requires Claude calls).
func DefaultMetricsConfig() MetricsConfig {
	return MetricsConfig{
		LOCTracking:     true,  // Enabled by default per requirements
		HumanEstimation: false, // Disabled by default (requires Claude calls)
	}
}

// DefaultArchitectureConfig returns ArchitectureConfig with sensible default values.
// Architecture Checkpoint is DISABLED by default to ensure zero behavior change.
func DefaultArchitectureConfig() ArchitectureConfig {
	return ArchitectureConfig{
		Enabled:              false,                    // Disabled by default
		Mode:                 ArchitectureModeEscalate, // Escalate mode when enabled
		RequireJustification: false,                    // Don't require justification by default
		EscalateOnUncertain:  false,                    // Don't escalate on low confidence
		ConfidenceThreshold:  0.7,                      // 70% confidence threshold
	}
}

// DefaultAgentWatchConfig returns AgentWatchConfig with sensible default values
func DefaultAgentWatchConfig() AgentWatchConfig {
	return AgentWatchConfig{
		Enabled:          true,
		BaseDir:          "~/.claude/projects",
		DefaultLimit:     100,
		PollIntervalSecs: 2,
		AutoImport:       false,
		CacheSize:        50,
		CostModel:        DefaultCostModelConfig(),
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
		Learning: LearningConfig{
			Enabled:           true,
			DBPath:            ".conductor/learning/executions.db",
			SwapDuringRetries: true,
			WarmUpEnabled:     true,
		},
		QualityControl: QualityControlConfig{
			Enabled: false,
			Agents: QCAgentConfig{
				Mode:              "auto",
				DefaultAgent:      "quality-control",
				ExplicitList:      []string{},
				AdditionalAgents:  []string{},
				BlockedAgents:     []string{},
				MaxAgents:         4,
				CacheTTLSeconds:   3600,
				RequireCodeReview: true,
			},
			RetryOnRed: 2,
		},
		AgentWatch: DefaultAgentWatchConfig(),
		Validation: ValidationConfig{
			KeyPointCriteria: "warn",
			StrictRubric:     false,
		},
		Executor: ExecutorConfig{
			EnforceDependencyChecks:     true,
			EnforceTestCommands:         true,
			VerifyCriteria:              true,
			EnforcePackageGuard:         true,
			EnforceDocTargets:           true,
			EnableErrorPatternDetection: true,
			EnableClaudeClassification:  false,
			IntelligentAgentSelection:   false, // Disabled by default, also enabled when QC mode is "intelligent"
		},
		TTS:          DefaultTTSConfig(),
		Setup:        DefaultSetupConfig(),
		Rollback:     DefaultRollbackConfig(),
		Budget:       DefaultBudgetConfig(),
		Pattern:      DefaultPatternConfig(),
		Architecture: DefaultArchitectureConfig(),
		Timeouts:     DefaultTimeoutsConfig(),
		Metrics:      DefaultMetricsConfig(),
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

// LoadConfig loads configuration from the specified file path
// If the file doesn't exist, returns default configuration without error
// If the file exists but is malformed, returns an error
func LoadConfig(path string) (*Config, error) {
	// Start with defaults
	cfg := DefaultConfig()

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// File doesn't exist, return defaults
		return cfg, nil
	}

	// Read the file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	// Use a temporary struct to handle duration parsing
	type yamlTTSConfig struct {
		Enabled bool   `yaml:"enabled"`
		BaseURL string `yaml:"base_url"`
		Model   string `yaml:"model"`
		Voice   string `yaml:"voice"`
		Timeout string `yaml:"timeout"`
	}
	type yamlTimeoutsConfig struct {
		Task   string `yaml:"task"`
		LLM    string `yaml:"llm"`
		HTTP   string `yaml:"http"`
		Search string `yaml:"search"`
	}
	type yamlBudgetConfig struct {
		Enabled          bool   `yaml:"enabled"`
		AutoResume       bool   `yaml:"auto_resume"`
		MaxWaitDuration  string `yaml:"max_wait_duration"`
		AnnounceInterval string `yaml:"announce_interval"`
		SafetyBuffer     string `yaml:"safety_buffer"`
	}
	type yamlConfig struct {
		MaxConcurrency int                  `yaml:"max_concurrency"`
		Timeout        string               `yaml:"timeout"`
		LogLevel       string               `yaml:"log_level"`
		LogDir         string               `yaml:"log_dir"`
		DryRun         bool                 `yaml:"dry_run"`
		SkipCompleted  bool                 `yaml:"skip_completed"`
		RetryFailed    bool                 `yaml:"retry_failed"`
		Learning       LearningConfig       `yaml:"learning"`
		QualityControl QualityControlConfig `yaml:"quality_control"`
		AgentWatch     AgentWatchConfig     `yaml:"agent_watch"`
		Validation     ValidationConfig     `yaml:"validation"`
		Executor       ExecutorConfig       `yaml:"executor"`
		TTS            yamlTTSConfig        `yaml:"tts"`
		Setup          SetupConfig          `yaml:"setup"`
		Rollback       RollbackConfig       `yaml:"rollback"`
		Budget         yamlBudgetConfig     `yaml:"budget"`
		Pattern        PatternConfig        `yaml:"pattern"`
		Architecture   ArchitectureConfig   `yaml:"architecture"`
		Timeouts       yamlTimeoutsConfig   `yaml:"timeouts"`
		Metrics        MetricsConfig        `yaml:"metrics"`
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

	// Merge nested configs - need to check if sections were provided at all
	// We create a temporary unmarshal to detect if sections exist
	var rawMap map[string]interface{}
	if err := yaml.Unmarshal(data, &rawMap); err == nil {
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
			if _, exists := learningMap["swap_during_retries"]; exists {
				cfg.Learning.SwapDuringRetries = learning.SwapDuringRetries
			}
			if _, exists := learningMap["warmup_enabled"]; exists {
				cfg.Learning.WarmUpEnabled = learning.WarmUpEnabled
			}
			if _, exists := learningMap["keep_executions_days"]; exists {
				cfg.Learning.KeepExecutionsDays = learning.KeepExecutionsDays
			}
			if _, exists := learningMap["min_failures_before_adapt"]; exists {
				cfg.Learning.MinFailuresBeforeAdapt = learning.MinFailuresBeforeAdapt
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
				if defaultAgent, exists := agentsMap["default_agent"]; exists {
					if agentStr, ok := defaultAgent.(string); ok {
						cfg.QualityControl.Agents.DefaultAgent = agentStr
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
			}
		}

		// Merge AgentWatch config
		if agentWatchSection, exists := rawMap["agent_watch"]; exists && agentWatchSection != nil {
			agentWatch := yamlCfg.AgentWatch
			agentWatchMap, _ := agentWatchSection.(map[string]interface{})

			if _, exists := agentWatchMap["enabled"]; exists {
				cfg.AgentWatch.Enabled = agentWatch.Enabled
			}
			if _, exists := agentWatchMap["base_dir"]; exists {
				cfg.AgentWatch.BaseDir = agentWatch.BaseDir
			}
			if _, exists := agentWatchMap["default_limit"]; exists {
				cfg.AgentWatch.DefaultLimit = agentWatch.DefaultLimit
			}
			if _, exists := agentWatchMap["poll_interval_secs"]; exists {
				cfg.AgentWatch.PollIntervalSecs = agentWatch.PollIntervalSecs
			}
			if _, exists := agentWatchMap["auto_import"]; exists {
				cfg.AgentWatch.AutoImport = agentWatch.AutoImport
			}
			if _, exists := agentWatchMap["cache_size"]; exists {
				cfg.AgentWatch.CacheSize = agentWatch.CacheSize
			}

			// Handle cost_model section
			if costModelSection, exists := agentWatchMap["cost_model"]; exists && costModelSection != nil {
				costModelMap, _ := costModelSection.(map[string]interface{})

				if sonnetInput, exists := costModelMap["sonnet_input"]; exists {
					if val, ok := sonnetInput.(float64); ok {
						cfg.AgentWatch.CostModel.SonnetInput = val
					}
				}
				if sonnetOutput, exists := costModelMap["sonnet_output"]; exists {
					if val, ok := sonnetOutput.(float64); ok {
						cfg.AgentWatch.CostModel.SonnetOutput = val
					}
				}
				if haikuInput, exists := costModelMap["haiku_input"]; exists {
					if val, ok := haikuInput.(float64); ok {
						cfg.AgentWatch.CostModel.HaikuInput = val
					}
				}
				if haikuOutput, exists := costModelMap["haiku_output"]; exists {
					if val, ok := haikuOutput.(float64); ok {
						cfg.AgentWatch.CostModel.HaikuOutput = val
					}
				}
				if opusInput, exists := costModelMap["opus_input"]; exists {
					if val, ok := opusInput.(float64); ok {
						cfg.AgentWatch.CostModel.OpusInput = val
					}
				}
				if opusOutput, exists := costModelMap["opus_output"]; exists {
					if val, ok := opusOutput.(float64); ok {
						cfg.AgentWatch.CostModel.OpusOutput = val
					}
				}
			}
		}

		// Merge Validation config
		if validationSection, exists := rawMap["validation"]; exists && validationSection != nil {
			validation := yamlCfg.Validation
			validationMap, _ := validationSection.(map[string]interface{})

			if _, exists := validationMap["key_point_criteria"]; exists {
				cfg.Validation.KeyPointCriteria = validation.KeyPointCriteria
			}
			if _, exists := validationMap["strict_rubric"]; exists {
				cfg.Validation.StrictRubric = validation.StrictRubric
			}
		}

		// Merge Executor config
		if executorSection, exists := rawMap["executor"]; exists && executorSection != nil {
			executor := yamlCfg.Executor
			executorMap, _ := executorSection.(map[string]interface{})

			if _, exists := executorMap["enforce_dependency_checks"]; exists {
				cfg.Executor.EnforceDependencyChecks = executor.EnforceDependencyChecks
			}
			if _, exists := executorMap["enforce_test_commands"]; exists {
				cfg.Executor.EnforceTestCommands = executor.EnforceTestCommands
			}
			if _, exists := executorMap["verify_criteria"]; exists {
				cfg.Executor.VerifyCriteria = executor.VerifyCriteria
			}
			if _, exists := executorMap["enforce_package_guard"]; exists {
				cfg.Executor.EnforcePackageGuard = executor.EnforcePackageGuard
			}
			if _, exists := executorMap["enforce_doc_targets"]; exists {
				cfg.Executor.EnforceDocTargets = executor.EnforceDocTargets
			}
			if _, exists := executorMap["enable_error_pattern_detection"]; exists {
				cfg.Executor.EnableErrorPatternDetection = executor.EnableErrorPatternDetection
			}
			if _, exists := executorMap["enable_claude_classification"]; exists {
				cfg.Executor.EnableClaudeClassification = executor.EnableClaudeClassification
			}
			if _, exists := executorMap["intelligent_agent_selection"]; exists {
				cfg.Executor.IntelligentAgentSelection = executor.IntelligentAgentSelection
			}
		}

		// Merge TTS config
		if ttsSection, exists := rawMap["tts"]; exists && ttsSection != nil {
			tts := yamlCfg.TTS
			ttsMap, _ := ttsSection.(map[string]interface{})

			if _, exists := ttsMap["enabled"]; exists {
				cfg.TTS.Enabled = tts.Enabled
			}
			if _, exists := ttsMap["base_url"]; exists {
				cfg.TTS.BaseURL = tts.BaseURL
			}
			if _, exists := ttsMap["model"]; exists {
				cfg.TTS.Model = tts.Model
			}
			if _, exists := ttsMap["voice"]; exists {
				cfg.TTS.Voice = tts.Voice
			}
		}

		// Merge Setup config
		if setupSection, exists := rawMap["setup"]; exists && setupSection != nil {
			setup := yamlCfg.Setup
			setupMap, _ := setupSection.(map[string]interface{})

			if _, exists := setupMap["enabled"]; exists {
				cfg.Setup.Enabled = setup.Enabled
			}
		}

		// Merge Rollback config
		if rollbackSection, exists := rawMap["rollback"]; exists && rollbackSection != nil {
			rollback := yamlCfg.Rollback
			rollbackMap, _ := rollbackSection.(map[string]interface{})

			if _, exists := rollbackMap["enabled"]; exists {
				cfg.Rollback.Enabled = rollback.Enabled
			}
			if _, exists := rollbackMap["mode"]; exists {
				cfg.Rollback.Mode = rollback.Mode
			}
			if _, exists := rollbackMap["require_clean_state"]; exists {
				cfg.Rollback.RequireCleanState = rollback.RequireCleanState
			}
			if protectedBranches, exists := rollbackMap["protected_branches"]; exists {
				if list, ok := protectedBranches.([]interface{}); ok {
					cfg.Rollback.ProtectedBranches = interfaceSliceToStringSlice(list)
				}
			}
			if _, exists := rollbackMap["working_branch_prefix"]; exists {
				cfg.Rollback.WorkingBranchPrefix = rollback.WorkingBranchPrefix
			}
			if _, exists := rollbackMap["checkpoint_prefix"]; exists {
				cfg.Rollback.CheckpointPrefix = rollback.CheckpointPrefix
			}
			if _, exists := rollbackMap["keep_checkpoint_days"]; exists {
				cfg.Rollback.KeepCheckpointDays = rollback.KeepCheckpointDays
			}
		}

		// Merge Budget config
		if budgetSection, exists := rawMap["budget"]; exists && budgetSection != nil {
			budget := yamlCfg.Budget
			budgetMap, _ := budgetSection.(map[string]interface{})

			if _, exists := budgetMap["enabled"]; exists {
				cfg.Budget.Enabled = budget.Enabled
			}
			if _, exists := budgetMap["auto_resume"]; exists {
				cfg.Budget.AutoResume = budget.AutoResume
			}
			if _, exists := budgetMap["max_wait_duration"]; exists && budget.MaxWaitDuration != "" {
				d, err := time.ParseDuration(budget.MaxWaitDuration)
				if err != nil {
					return nil, fmt.Errorf("invalid budget.max_wait_duration format %q: %w", budget.MaxWaitDuration, err)
				}
				cfg.Budget.MaxWaitDuration = d
			}
			if _, exists := budgetMap["announce_interval"]; exists && budget.AnnounceInterval != "" {
				d, err := time.ParseDuration(budget.AnnounceInterval)
				if err != nil {
					return nil, fmt.Errorf("invalid budget.announce_interval format %q: %w", budget.AnnounceInterval, err)
				}
				cfg.Budget.AnnounceInterval = d
			}
			if _, exists := budgetMap["safety_buffer"]; exists && budget.SafetyBuffer != "" {
				d, err := time.ParseDuration(budget.SafetyBuffer)
				if err != nil {
					return nil, fmt.Errorf("invalid budget.safety_buffer format %q: %w", budget.SafetyBuffer, err)
				}
				cfg.Budget.SafetyBuffer = d
			}
		}

		// Merge Pattern config
		if patternSection, exists := rawMap["pattern"]; exists && patternSection != nil {
			pattern := yamlCfg.Pattern
			patternMap, _ := patternSection.(map[string]interface{})

			if _, exists := patternMap["enabled"]; exists {
				cfg.Pattern.Enabled = pattern.Enabled
			}
			if _, exists := patternMap["mode"]; exists {
				cfg.Pattern.Mode = pattern.Mode
			}
			if _, exists := patternMap["similarity_threshold"]; exists {
				cfg.Pattern.SimilarityThreshold = pattern.SimilarityThreshold
			}
			if _, exists := patternMap["duplicate_threshold"]; exists {
				cfg.Pattern.DuplicateThreshold = pattern.DuplicateThreshold
			}
			if _, exists := patternMap["min_confidence"]; exists {
				cfg.Pattern.MinConfidence = pattern.MinConfidence
			}
			if _, exists := patternMap["enable_stop"]; exists {
				cfg.Pattern.EnableSTOP = pattern.EnableSTOP
			}
			if _, exists := patternMap["enable_duplicate_detection"]; exists {
				cfg.Pattern.EnableDuplicateDetection = pattern.EnableDuplicateDetection
			}
			if _, exists := patternMap["inject_into_prompt"]; exists {
				cfg.Pattern.InjectIntoPrompt = pattern.InjectIntoPrompt
			}
			if _, exists := patternMap["max_patterns_per_task"]; exists {
				cfg.Pattern.MaxPatternsPerTask = pattern.MaxPatternsPerTask
			}
			if _, exists := patternMap["max_related_files"]; exists {
				cfg.Pattern.MaxRelatedFiles = pattern.MaxRelatedFiles
			}
			if _, exists := patternMap["llm_enhancement_enabled"]; exists {
				cfg.Pattern.LLMEnhancementEnabled = pattern.LLMEnhancementEnabled
			}
			if _, exists := patternMap["require_justification"]; exists {
				cfg.Pattern.RequireJustification = pattern.RequireJustification
			}
		}

		// Merge Architecture config
		if architectureSection, exists := rawMap["architecture"]; exists && architectureSection != nil {
			arch := yamlCfg.Architecture
			archMap, _ := architectureSection.(map[string]interface{})

			if _, exists := archMap["enabled"]; exists {
				cfg.Architecture.Enabled = arch.Enabled
			}
			if _, exists := archMap["mode"]; exists {
				cfg.Architecture.Mode = arch.Mode
			}
			if _, exists := archMap["require_justification"]; exists {
				cfg.Architecture.RequireJustification = arch.RequireJustification
			}
			if _, exists := archMap["escalate_on_uncertain"]; exists {
				cfg.Architecture.EscalateOnUncertain = arch.EscalateOnUncertain
			}
			if _, exists := archMap["confidence_threshold"]; exists {
				cfg.Architecture.ConfidenceThreshold = arch.ConfidenceThreshold
			}
		}

		// Merge Timeouts config
		if timeoutsSection, exists := rawMap["timeouts"]; exists && timeoutsSection != nil {
			timeouts := yamlCfg.Timeouts
			timeoutsMap, _ := timeoutsSection.(map[string]interface{})

			if _, exists := timeoutsMap["task"]; exists && timeouts.Task != "" {
				d, err := time.ParseDuration(timeouts.Task)
				if err != nil {
					return nil, fmt.Errorf("invalid timeouts.task format %q: %w", timeouts.Task, err)
				}
				cfg.Timeouts.Task = d
			}
			if _, exists := timeoutsMap["llm"]; exists && timeouts.LLM != "" {
				d, err := time.ParseDuration(timeouts.LLM)
				if err != nil {
					return nil, fmt.Errorf("invalid timeouts.llm format %q: %w", timeouts.LLM, err)
				}
				cfg.Timeouts.LLM = d
			}
			if _, exists := timeoutsMap["http"]; exists && timeouts.HTTP != "" {
				d, err := time.ParseDuration(timeouts.HTTP)
				if err != nil {
					return nil, fmt.Errorf("invalid timeouts.http format %q: %w", timeouts.HTTP, err)
				}
				cfg.Timeouts.HTTP = d
			}
			if _, exists := timeoutsMap["search"]; exists && timeouts.Search != "" {
				d, err := time.ParseDuration(timeouts.Search)
				if err != nil {
					return nil, fmt.Errorf("invalid timeouts.search format %q: %w", timeouts.Search, err)
				}
				cfg.Timeouts.Search = d
			}
		}

		// Merge Metrics config
		if metricsSection, exists := rawMap["metrics"]; exists && metricsSection != nil {
			metrics := yamlCfg.Metrics
			metricsMap, _ := metricsSection.(map[string]interface{})

			if _, exists := metricsMap["loc_tracking"]; exists {
				cfg.Metrics.LOCTracking = metrics.LOCTracking
			}
			if _, exists := metricsMap["human_estimation"]; exists {
				cfg.Metrics.HumanEstimation = metrics.HumanEstimation
			}
		}

	}

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

// LoadConfigFromDir loads configuration from .conductor/config.yaml
// Priority order:
//  1. If dir is provided and config exists at {dir}/.conductor/config.yaml, use it
//  2. If "." is passed, use current working directory
//  3. Try build-time injected root if available
//  4. Return default configuration if no config file found
func LoadConfigFromDir(dir string) (*Config, error) {
	// Handle "." as current directory
	if dir == "." {
		cwd, err := os.Getwd()
		if err != nil {
			return DefaultConfig(), nil
		}
		dir = cwd
	}

	// Try config at provided/resolved directory
	if dir != "" {
		configPath := filepath.Join(dir, ".conductor", "config.yaml")
		if _, err := os.Stat(configPath); err == nil {
			return LoadConfig(configPath)
		}
	}

	// Try build-time root as fallback
	if buildTimeRepoRoot != "" {
		return LoadConfigFromRootWithBuildTime(buildTimeRepoRoot)
	}

	// Return defaults if no config found
	return DefaultConfig(), nil
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
		c.Timeouts.Task = *timeout // Also update centralized timeout (v2.33+)
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

	// Validate validation settings (supports warn, strict, off)
	if c.Validation.KeyPointCriteria == "" {
		c.Validation.KeyPointCriteria = "warn"
	}
	validCriteriaModes := map[string]bool{
		"warn":   true,
		"strict": true,
		"off":    true,
	}
	if !validCriteriaModes[c.Validation.KeyPointCriteria] {
		return fmt.Errorf("validation.key_point_criteria must be one of: warn, strict, off; got %q", c.Validation.KeyPointCriteria)
	}

	// Validate learning configuration
	if c.Learning.Enabled {
		if c.Learning.DBPath == "" {
			return fmt.Errorf("learning.db_path cannot be empty when learning is enabled")
		}
		if c.Learning.KeepExecutionsDays < 0 {
			return fmt.Errorf("learning.keep_executions_days must be >= 0, got %d", c.Learning.KeepExecutionsDays)
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
			// explicit mode requires at least one agent - this is a configuration error
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

	// Validate Agent Watch configuration
	if c.AgentWatch.Enabled {
		// Validate base_dir is not empty
		if c.AgentWatch.BaseDir == "" {
			return fmt.Errorf("agent_watch.base_dir cannot be empty when agent_watch is enabled")
		}

		// Validate default_limit is positive
		if c.AgentWatch.DefaultLimit <= 0 {
			return fmt.Errorf("agent_watch.default_limit must be > 0, got %d", c.AgentWatch.DefaultLimit)
		}

		// Validate poll_interval_secs is positive
		if c.AgentWatch.PollIntervalSecs <= 0 {
			return fmt.Errorf("agent_watch.poll_interval_secs must be > 0, got %d", c.AgentWatch.PollIntervalSecs)
		}

		// Validate cache_size is positive
		if c.AgentWatch.CacheSize <= 0 {
			return fmt.Errorf("agent_watch.cache_size must be > 0, got %d", c.AgentWatch.CacheSize)
		}

		// Validate cost model pricing is non-negative
		costModel := c.AgentWatch.CostModel
		if costModel.SonnetInput < 0 {
			return fmt.Errorf("agent_watch.cost_model.sonnet_input must be >= 0, got %f", costModel.SonnetInput)
		}
		if costModel.SonnetOutput < 0 {
			return fmt.Errorf("agent_watch.cost_model.sonnet_output must be >= 0, got %f", costModel.SonnetOutput)
		}
		if costModel.HaikuInput < 0 {
			return fmt.Errorf("agent_watch.cost_model.haiku_input must be >= 0, got %f", costModel.HaikuInput)
		}
		if costModel.HaikuOutput < 0 {
			return fmt.Errorf("agent_watch.cost_model.haiku_output must be >= 0, got %f", costModel.HaikuOutput)
		}
		if costModel.OpusInput < 0 {
			return fmt.Errorf("agent_watch.cost_model.opus_input must be >= 0, got %f", costModel.OpusInput)
		}
		if costModel.OpusOutput < 0 {
			return fmt.Errorf("agent_watch.cost_model.opus_output must be >= 0, got %f", costModel.OpusOutput)
		}
	}

	// Validate Budget configuration
	if c.Budget.Enabled {
		if c.Budget.MaxWaitDuration < 0 {
			return fmt.Errorf("budget.max_wait_duration must be >= 0, got %v", c.Budget.MaxWaitDuration)
		}
		if c.Budget.AnnounceInterval < 0 {
			return fmt.Errorf("budget.announce_interval must be >= 0, got %v", c.Budget.AnnounceInterval)
		}
		if c.Budget.SafetyBuffer < 0 {
			return fmt.Errorf("budget.safety_buffer must be >= 0, got %v", c.Budget.SafetyBuffer)
		}
	}

	// Validate Rollback configuration
	if c.Rollback.Enabled {
		// Validate mode
		validRollbackModes := map[RollbackMode]bool{
			RollbackModeManual:           true,
			RollbackModeAutoOnRed:        true,
			RollbackModeAutoOnMaxRetries: true,
		}
		if !validRollbackModes[c.Rollback.Mode] {
			return fmt.Errorf("rollback.mode must be one of: manual, auto_on_red, auto_on_max_retries; got %q", c.Rollback.Mode)
		}

		// Validate working_branch_prefix is not empty
		if c.Rollback.WorkingBranchPrefix == "" {
			return fmt.Errorf("rollback.working_branch_prefix cannot be empty when rollback is enabled")
		}

		// Validate checkpoint_prefix is not empty
		if c.Rollback.CheckpointPrefix == "" {
			return fmt.Errorf("rollback.checkpoint_prefix cannot be empty when rollback is enabled")
		}

		// Validate keep_checkpoint_days is non-negative
		if c.Rollback.KeepCheckpointDays < 0 {
			return fmt.Errorf("rollback.keep_checkpoint_days must be >= 0, got %d", c.Rollback.KeepCheckpointDays)
		}
	}

	// Validate Pattern configuration
	if c.Pattern.Enabled {
		// Validate mode
		validPatternModes := map[PatternMode]bool{
			PatternModeBlock:   true,
			PatternModeWarn:    true,
			PatternModeSuggest: true,
		}
		if !validPatternModes[c.Pattern.Mode] {
			return fmt.Errorf("pattern.mode must be one of: block, warn, suggest; got %q", c.Pattern.Mode)
		}

		// Validate similarity_threshold is between 0 and 1
		if c.Pattern.SimilarityThreshold < 0 || c.Pattern.SimilarityThreshold > 1 {
			return fmt.Errorf("pattern.similarity_threshold must be between 0 and 1, got %f", c.Pattern.SimilarityThreshold)
		}

		// Validate duplicate_threshold is between 0 and 1
		if c.Pattern.DuplicateThreshold < 0 || c.Pattern.DuplicateThreshold > 1 {
			return fmt.Errorf("pattern.duplicate_threshold must be between 0 and 1, got %f", c.Pattern.DuplicateThreshold)
		}

		// Validate min_confidence is between 0 and 1
		if c.Pattern.MinConfidence < 0 || c.Pattern.MinConfidence > 1 {
			return fmt.Errorf("pattern.min_confidence must be between 0 and 1, got %f", c.Pattern.MinConfidence)
		}

		// Validate max_patterns_per_task is non-negative
		if c.Pattern.MaxPatternsPerTask < 0 {
			return fmt.Errorf("pattern.max_patterns_per_task must be >= 0, got %d", c.Pattern.MaxPatternsPerTask)
		}

		// Validate max_related_files is non-negative
		if c.Pattern.MaxRelatedFiles < 0 {
			return fmt.Errorf("pattern.max_related_files must be >= 0, got %d", c.Pattern.MaxRelatedFiles)
		}
	}

	return nil
}
