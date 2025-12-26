package pattern

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

	// CacheTTLSeconds is how long to cache pattern analysis results (default: 3600)
	CacheTTLSeconds int `yaml:"cache_ttl_seconds"`

	// LLM Enhancement (optional, requires Claude CLI)
	// LLMEnhancementEnabled enables Claude-based confidence refinement
	LLMEnhancementEnabled bool `yaml:"llm_enhancement_enabled"`

	// LLMMinConfidence is the minimum confidence to trigger LLM enhancement (default: 0.3)
	LLMMinConfidence float64 `yaml:"llm_min_confidence"`

	// LLMMaxConfidence is the maximum confidence to trigger LLM enhancement (default: 0.7)
	LLMMaxConfidence float64 `yaml:"llm_max_confidence"`

	// LLMTimeoutSeconds is the timeout for Claude CLI response (default: 30)
	LLMTimeoutSeconds int `yaml:"llm_timeout_seconds"`
}

// DefaultPatternConfig returns PatternConfig with sensible default values.
// Pattern Intelligence is DISABLED by default to ensure zero behavior change unless explicitly enabled.
func DefaultPatternConfig() PatternConfig {
	return PatternConfig{
		Enabled:                  false, // Disabled by default for zero behavior change
		Mode:                     PatternModeWarn,
		SimilarityThreshold:      0.8,  // 80% similarity triggers action
		DuplicateThreshold:       0.9,  // 90% similarity for duplicate detection
		MinConfidence:            0.7,  // 70% confidence required
		EnableSTOP:               true, // STOP analysis enabled when system is enabled
		EnableDuplicateDetection: true, // Duplicate detection enabled when system is enabled
		InjectIntoPrompt:         true, // Include analysis in prompts by default
		MaxPatternsPerTask:       5,     // Limit patterns to avoid prompt bloat
		MaxRelatedFiles:          10,    // Limit related files
		CacheTTLSeconds:          3600,  // 1 hour cache
		LLMEnhancementEnabled:    false, // Disabled by default
		LLMMinConfidence:         0.3,   // Enhance when >= 0.3
		LLMMaxConfidence:         0.7,   // Enhance when <= 0.7
		LLMTimeoutSeconds:        30,    // 30 second timeout
		LLMMaxWaitSeconds:        60,    // 60 second max wait for rate limit
	}
}

// Validate validates the PatternConfig values.
// Returns an error if any values are invalid.
func (c *PatternConfig) Validate() error {
	// Skip validation if disabled
	if !c.Enabled {
		return nil
	}

	// Mode validation is handled by the config loader
	// Threshold validations
	if c.SimilarityThreshold < 0 || c.SimilarityThreshold > 1 {
		return &ConfigError{
			Field:   "similarity_threshold",
			Message: "must be between 0 and 1",
			Value:   c.SimilarityThreshold,
		}
	}

	if c.DuplicateThreshold < 0 || c.DuplicateThreshold > 1 {
		return &ConfigError{
			Field:   "duplicate_threshold",
			Message: "must be between 0 and 1",
			Value:   c.DuplicateThreshold,
		}
	}

	if c.MinConfidence < 0 || c.MinConfidence > 1 {
		return &ConfigError{
			Field:   "min_confidence",
			Message: "must be between 0 and 1",
			Value:   c.MinConfidence,
		}
	}

	if c.MaxPatternsPerTask < 0 {
		return &ConfigError{
			Field:   "max_patterns_per_task",
			Message: "must be >= 0",
			Value:   c.MaxPatternsPerTask,
		}
	}

	if c.MaxRelatedFiles < 0 {
		return &ConfigError{
			Field:   "max_related_files",
			Message: "must be >= 0",
			Value:   c.MaxRelatedFiles,
		}
	}

	if c.CacheTTLSeconds < 0 {
		return &ConfigError{
			Field:   "cache_ttl_seconds",
			Message: "must be >= 0",
			Value:   c.CacheTTLSeconds,
		}
	}

	// LLM enhancement validation
	if c.LLMEnhancementEnabled {
		if c.LLMMinConfidence < 0 || c.LLMMinConfidence > 1 {
			return &ConfigError{
				Field:   "llm_min_confidence",
				Message: "must be between 0 and 1",
				Value:   c.LLMMinConfidence,
			}
		}
		if c.LLMMaxConfidence < 0 || c.LLMMaxConfidence > 1 {
			return &ConfigError{
				Field:   "llm_max_confidence",
				Message: "must be between 0 and 1",
				Value:   c.LLMMaxConfidence,
			}
		}
		if c.LLMMinConfidence > c.LLMMaxConfidence {
			return &ConfigError{
				Field:   "llm_min_confidence",
				Message: "must be <= llm_max_confidence",
				Value:   c.LLMMinConfidence,
			}
		}
		if c.LLMTimeoutSeconds <= 0 {
			return &ConfigError{
				Field:   "llm_timeout_seconds",
				Message: "must be > 0",
				Value:   c.LLMTimeoutSeconds,
			}
		}
		if c.LLMMaxWaitSeconds <= 0 {
			return &ConfigError{
				Field:   "llm_max_wait_seconds",
				Message: "must be > 0",
				Value:   c.LLMMaxWaitSeconds,
			}
		}
	}

	return nil
}

// ConfigError represents a configuration validation error
type ConfigError struct {
	Field   string
	Message string
	Value   interface{}
}

func (e *ConfigError) Error() string {
	return "pattern." + e.Field + ": " + e.Message
}

// ShouldBlock returns true if the mode is set to block
func (c *PatternConfig) ShouldBlock() bool {
	return c.Enabled && c.Mode == PatternModeBlock
}

// ShouldWarn returns true if warnings should be logged
func (c *PatternConfig) ShouldWarn() bool {
	return c.Enabled && (c.Mode == PatternModeWarn || c.Mode == PatternModeBlock)
}

// ShouldInject returns true if analysis should be injected into prompts
func (c *PatternConfig) ShouldInject() bool {
	return c.Enabled && c.InjectIntoPrompt
}
