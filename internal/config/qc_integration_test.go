package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/harrison/conductor/internal/models"
	"github.com/harrison/conductor/internal/parser"
)

// TestQualityControlConfigIntegration tests the full integration of QC config from config.yaml
// This test verifies:
// 1. Config struct has QualityControl field
// 2. LoadConfig() properly parses quality_control from YAML
// 3. Plan without QC frontmatter receives config QC settings
// 4. TaskExecutor receives merged QC config
func TestQualityControlConfigIntegration(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := t.TempDir()

	// Step 1: Create config.yaml with quality_control enabled
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `max_concurrency: 4
timeout: 30m
quality_control:
  enabled: true
  agents:
    mode: explicit
    explicit_list:
      - custom-qa-agent
  retry_on_red: 3
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config.yaml: %v", err)
	}

	// Step 2: Load config and verify QC settings are parsed
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Verify QualityControl field exists and is populated
	if !cfg.QualityControl.Enabled {
		t.Errorf("Config.QualityControl.Enabled = %v, want true", cfg.QualityControl.Enabled)
	}
	if cfg.QualityControl.Agents.Mode != "explicit" {
		t.Errorf("Config.QualityControl.Agents.Mode = %q, want %q", cfg.QualityControl.Agents.Mode, "explicit")
	}
	if len(cfg.QualityControl.Agents.ExplicitList) != 1 || cfg.QualityControl.Agents.ExplicitList[0] != "custom-qa-agent" {
		t.Errorf("Config.QualityControl.Agents.ExplicitList = %v, want [custom-qa-agent]", cfg.QualityControl.Agents.ExplicitList)
	}
	if cfg.QualityControl.RetryOnRed != 3 {
		t.Errorf("Config.QualityControl.RetryOnRed = %d, want 3", cfg.QualityControl.RetryOnRed)
	}

	// Step 3: Create plan file WITHOUT QC frontmatter
	planPath := filepath.Join(tmpDir, "test-plan.md")
	planContent := `# Test Plan

## Task 1: Setup database
**Files**: db/schema.sql
**Depends on**: None
**Estimated time**: 30m

Initialize the database schema.

## Task 2: Create API
**Files**: api/server.go
**Depends on**: Task 1
**Estimated time**: 1h

Set up REST API server.
`
	if err := os.WriteFile(planPath, []byte(planContent), 0644); err != nil {
		t.Fatalf("failed to write test plan: %v", err)
	}

	// Step 4: Parse plan (should NOT have QC settings)
	plan, err := parser.ParseFile(planPath)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}

	// Verify plan has no QC config initially
	if plan.QualityControl.Enabled {
		t.Errorf("Initial Plan.QualityControl.Enabled = %v, want false (no frontmatter)", plan.QualityControl.Enabled)
	}

	// Step 5: Simulate merge logic from run.go (lines 341-350)
	// This is the critical path that merges config QC into plan
	if !plan.QualityControl.Enabled && cfg.QualityControl.Enabled {
		plan.QualityControl = models.QualityControlConfig{
			Enabled:    cfg.QualityControl.Enabled,
			Agents:     cfg.QualityControl.Agents,
			RetryOnRed: cfg.QualityControl.RetryOnRed,
		}
	}

	// Step 6: Verify plan now has merged QC config
	if !plan.QualityControl.Enabled {
		t.Errorf("After merge, Plan.QualityControl.Enabled = %v, want true", plan.QualityControl.Enabled)
	}
	if plan.QualityControl.Agents.Mode != "explicit" {
		t.Errorf("After merge, Plan.QualityControl.Agents.Mode = %q, want %q", plan.QualityControl.Agents.Mode, "explicit")
	}
	if len(plan.QualityControl.Agents.ExplicitList) != 1 || plan.QualityControl.Agents.ExplicitList[0] != "custom-qa-agent" {
		t.Errorf("After merge, Plan.QualityControl.Agents.ExplicitList = %v, want [custom-qa-agent]", plan.QualityControl.Agents.ExplicitList)
	}
	if plan.QualityControl.RetryOnRed != 3 {
		t.Errorf("After merge, Plan.QualityControl.RetryOnRed = %d, want 3", plan.QualityControl.RetryOnRed)
	}
}

// TestQualityControlConfigPlanOverridesConfig tests that plan frontmatter takes precedence
func TestQualityControlConfigPlanOverridesConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Create config.yaml with QC enabled
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `quality_control:
  enabled: true
  agents:
    mode: explicit
    explicit_list:
      - config-agent
  retry_on_red: 2
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config.yaml: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Create plan WITH QC in conductor section (YAML format)
	planPath := filepath.Join(tmpDir, "test-plan.yaml")
	planContent := `conductor:
  quality_control:
    enabled: true
    agents:
      mode: explicit
      explicit_list:
        - plan-agent
    retry_on_red: 5
plan:
  metadata:
    feature_name: Test Feature
  tasks:
    - task_number: 1
      name: Test task
      files:
        - test.go
      description: Test task description
`
	if err := os.WriteFile(planPath, []byte(planContent), 0644); err != nil {
		t.Fatalf("failed to write test plan: %v", err)
	}

	plan, err := parser.ParseFile(planPath)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}

	// Plan already has QC config from conductor section
	if !plan.QualityControl.Enabled {
		t.Errorf("Plan.QualityControl.Enabled = %v, want true", plan.QualityControl.Enabled)
	}

	// Simulate merge logic from run.go
	// Config should NOT override plan frontmatter
	if !plan.QualityControl.Enabled && cfg.QualityControl.Enabled {
		plan.QualityControl = models.QualityControlConfig{
			Enabled:    cfg.QualityControl.Enabled,
			Agents:     cfg.QualityControl.Agents,
			RetryOnRed: cfg.QualityControl.RetryOnRed,
		}
	}

	// Verify plan settings are preserved (not overridden by config)
	if len(plan.QualityControl.Agents.ExplicitList) != 1 || plan.QualityControl.Agents.ExplicitList[0] != "plan-agent" {
		t.Errorf("Plan.QualityControl.Agents.ExplicitList = %v, want [plan-agent] (plan takes precedence)", plan.QualityControl.Agents.ExplicitList)
	}
	if plan.QualityControl.RetryOnRed != 5 {
		t.Errorf("Plan.QualityControl.RetryOnRed = %d, want 5 (plan takes precedence)", plan.QualityControl.RetryOnRed)
	}
}

// TestQualityControlConfigDisabledInConfig tests that disabled config doesn't override plan
func TestQualityControlConfigDisabledInConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Create config.yaml with QC disabled
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `quality_control:
  enabled: false
  agents:
    mode: auto
  retry_on_red: 2
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config.yaml: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Create plan without QC frontmatter
	planPath := filepath.Join(tmpDir, "test-plan.md")
	planContent := `# Test Plan

## Task 1: Test
**Files**: test.go

Test task.
`
	if err := os.WriteFile(planPath, []byte(planContent), 0644); err != nil {
		t.Fatalf("failed to write test plan: %v", err)
	}

	plan, err := parser.ParseFile(planPath)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}

	// Simulate merge logic from run.go
	if !plan.QualityControl.Enabled && cfg.QualityControl.Enabled {
		plan.QualityControl = models.QualityControlConfig{
			Enabled:     cfg.QualityControl.Enabled,
			ReviewAgent: cfg.QualityControl.ReviewAgent,
			RetryOnRed:  cfg.QualityControl.RetryOnRed,
		}
	}

	// Plan should remain disabled (config is disabled)
	if plan.QualityControl.Enabled {
		t.Errorf("Plan.QualityControl.Enabled = %v, want false (config disabled)", plan.QualityControl.Enabled)
	}
}

// TestQualityControlConfigValidation tests validation of QC config values
func TestQualityControlConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		config    string
		wantError bool
	}{
		{
			name: "valid QC config",
			config: `quality_control:
  enabled: true
  review_agent: my-agent
  retry_on_red: 3
`,
			wantError: false,
		},
		{
			name: "QC enabled but empty review_agent",
			config: `quality_control:
  enabled: true
  review_agent: ""
  retry_on_red: 2
`,
			wantError: false, // v2.2+: empty review_agent is valid with auto mode (auto-selects agents)
		},
		{
			name: "QC enabled but negative retry_on_red",
			config: `quality_control:
  enabled: true
  review_agent: my-agent
  retry_on_red: -1
`,
			wantError: true,
		},
		{
			name: "QC enabled but missing review_agent field",
			config: `quality_control:
  enabled: true
  retry_on_red: 2
`,
			wantError: false, // Should use default "quality-control"
		},
		{
			name: "QC disabled with invalid values",
			config: `quality_control:
  enabled: false
  review_agent: ""
  retry_on_red: -1
`,
			wantError: false, // Validation only applies when enabled
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

// TestQualityControlConfigMergeOrder tests configuration priority order
func TestQualityControlConfigMergeOrder(t *testing.T) {
	// Test priority: plan frontmatter (explicit) > config file > defaults
	// This test documents the expected merge behavior

	tests := []struct {
		name          string
		configContent string
		planFormat    string // "yaml" or "markdown"
		planHasQC     bool
		planQCConfig  string
		expectEnabled bool
		expectAgent   string
		expectRetry   int
		description   string
	}{
		{
			name:          "YAML plan with conductor QC overrides config",
			configContent: "quality_control:\n  enabled: true\n  review_agent: config-agent\n  retry_on_red: 2\n",
			planFormat:    "yaml",
			planHasQC:     true,
			planQCConfig: `conductor:
  quality_control:
    enabled: true
    review_agent: plan-agent
    retry_on_red: 5
`,
			expectEnabled: true,
			expectAgent:   "plan-agent",
			expectRetry:   5,
			description:   "YAML plan conductor section takes highest precedence",
		},
		{
			name:          "config fills in when plan has no QC",
			configContent: "quality_control:\n  enabled: true\n  review_agent: config-agent\n  retry_on_red: 3\n",
			planFormat:    "markdown",
			planHasQC:     false,
			expectEnabled: true,
			expectAgent:   "config-agent",
			expectRetry:   3,
			description:   "config provides defaults when plan has no QC",
		},
		{
			name:          "defaults used when config and plan both missing QC",
			configContent: "max_concurrency: 5\n",
			planFormat:    "markdown",
			planHasQC:     false,
			expectEnabled: false,
			expectAgent:   "quality-control",
			expectRetry:   2,
			description:   "system defaults used when neither config nor plan specify QC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Create config file
			configPath := filepath.Join(tmpDir, "config.yaml")
			if err := os.WriteFile(configPath, []byte(tt.configContent), 0644); err != nil {
				t.Fatalf("failed to write config: %v", err)
			}

			cfg, err := LoadConfig(configPath)
			if err != nil {
				t.Fatalf("LoadConfig() error = %v", err)
			}

			// Create plan file based on format
			var planPath string
			var planContent string

			if tt.planFormat == "yaml" {
				planPath = filepath.Join(tmpDir, "plan.yaml")
				if tt.planHasQC {
					planContent = tt.planQCConfig + `plan:
  metadata:
    feature_name: Test Feature
  tasks:
    - task_number: 1
      name: Test
      files: [test.go]
      description: Test task
`
				} else {
					planContent = `plan:
  metadata:
    feature_name: Test Feature
  tasks:
    - task_number: 1
      name: Test
      files: [test.go]
      description: Test task
`
				}
			} else {
				planPath = filepath.Join(tmpDir, "plan.md")
				planContent = `# Test Plan

## Task 1: Test
**Files**: test.go

Test task.
`
			}

			if err := os.WriteFile(planPath, []byte(planContent), 0644); err != nil {
				t.Fatalf("failed to write plan: %v", err)
			}

			plan, err := parser.ParseFile(planPath)
			if err != nil {
				t.Fatalf("ParseFile() error = %v", err)
			}

			// Apply merge logic (from run.go lines 341-350)
			if !plan.QualityControl.Enabled && cfg.QualityControl.Enabled {
				plan.QualityControl = models.QualityControlConfig{
					Enabled:     cfg.QualityControl.Enabled,
					ReviewAgent: cfg.QualityControl.ReviewAgent,
					RetryOnRed:  cfg.QualityControl.RetryOnRed,
				}
			}

			// Verify final merged config
			if plan.QualityControl.Enabled != tt.expectEnabled {
				t.Errorf("%s: Plan.QualityControl.Enabled = %v, want %v", tt.description, plan.QualityControl.Enabled, tt.expectEnabled)
			}

			// Only check agent and retry if QC is enabled
			if plan.QualityControl.Enabled {
				if plan.QualityControl.ReviewAgent != tt.expectAgent {
					t.Errorf("%s: Plan.QualityControl.ReviewAgent = %q, want %q", tt.description, plan.QualityControl.ReviewAgent, tt.expectAgent)
				}
				if plan.QualityControl.RetryOnRed != tt.expectRetry {
					t.Errorf("%s: Plan.QualityControl.RetryOnRed = %d, want %d", tt.description, plan.QualityControl.RetryOnRed, tt.expectRetry)
				}
			}
		})
	}
}
