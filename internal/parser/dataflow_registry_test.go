package parser

import (
	"strings"
	"testing"

	"github.com/harrison/conductor/internal/models"
)

func TestParseDataFlowRegistry_FromYAMLBlock(t *testing.T) {
	yaml := `
conductor:
  default_agent: "golang-pro"

planner_compliance:
  planner_version: "1.2.0"
  strict_enforcement: true
  required_features:
    - dependency_checks

data_flow_registry:
  producers:
    PlannerComplianceSpec:
      - task: "1"
        description: "Defines the compliance spec struct"
    DataFlowRegistry:
      - task: "3"
        description: "Defines the data flow registry"
  consumers:
    PlannerComplianceSpec:
      - task: "2"
        description: "Uses compliance spec in validation"
      - task: "3"
        description: "References compliance for enforcement"

plan:
  metadata:
    feature_name: "Data Flow Test"
    created: "2025-12-01"
    estimated_tasks: 1
  tasks:
    - task_number: 1
      name: "Test Task"
      files:
        - "test.go"
      depends_on: []
      description: "Test task"
      runtime_metadata:
        dependency_checks:
          - command: "go build"
            description: "verify build"
        documentation_targets: []
        prompt_blocks: []
      success_criteria:
        - "Task compiles"
`

	parser := NewYAMLParser()
	plan, err := parser.Parse(strings.NewReader(yaml))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if plan.DataFlowRegistry == nil {
		t.Fatal("Expected DataFlowRegistry to be populated")
	}

	// Check producers
	if len(plan.DataFlowRegistry.Producers) != 2 {
		t.Errorf("Expected 2 producers, got %d", len(plan.DataFlowRegistry.Producers))
	}

	pcs := plan.DataFlowRegistry.Producers["PlannerComplianceSpec"]
	if len(pcs) != 1 {
		t.Errorf("Expected 1 producer for PlannerComplianceSpec, got %d", len(pcs))
	} else if pcs[0].TaskNumber != "1" {
		t.Errorf("Expected task 1 for PlannerComplianceSpec producer, got %s", pcs[0].TaskNumber)
	}

	// Check consumers
	if len(plan.DataFlowRegistry.Consumers) != 1 {
		t.Errorf("Expected 1 consumer key, got %d", len(plan.DataFlowRegistry.Consumers))
	}

	pcsConsumers := plan.DataFlowRegistry.Consumers["PlannerComplianceSpec"]
	if len(pcsConsumers) != 2 {
		t.Errorf("Expected 2 consumers for PlannerComplianceSpec, got %d", len(pcsConsumers))
	}
}

func TestParseDataFlowRegistry_MissingRegistryInStrictMode(t *testing.T) {
	yaml := `
conductor:
  default_agent: "golang-pro"

planner_compliance:
  planner_version: "1.2.0"
  strict_enforcement: true
  required_features:
    - data_flow_registry

plan:
  metadata:
    feature_name: "Missing Registry Test"
    created: "2025-12-01"
    estimated_tasks: 1
  tasks:
    - task_number: 1
      name: "Test Task"
      files:
        - "test.go"
      depends_on: []
      description: "Test task"
      runtime_metadata:
        dependency_checks:
          - command: "go build"
            description: "verify build"
        documentation_targets: []
        prompt_blocks: []
      success_criteria:
        - "Task compiles"
`

	parser := NewYAMLParser()
	_, err := parser.Parse(strings.NewReader(yaml))
	if err == nil {
		t.Fatal("Expected error for missing data_flow_registry when required_features includes it")
	}

	if !strings.Contains(err.Error(), "data_flow_registry") {
		t.Errorf("Expected error about data_flow_registry, got: %v", err)
	}
}

func TestParseDataFlowRegistry_EmptyRegistry(t *testing.T) {
	yaml := `
conductor:
  default_agent: "golang-pro"

planner_compliance:
  planner_version: "1.2.0"
  strict_enforcement: true
  required_features:
    - data_flow_registry

data_flow_registry:
  producers: {}
  consumers: {}

plan:
  metadata:
    feature_name: "Empty Registry Test"
    created: "2025-12-01"
    estimated_tasks: 1
  tasks:
    - task_number: 1
      name: "Test Task"
      files:
        - "test.go"
      depends_on: []
      description: "Test task"
      runtime_metadata:
        dependency_checks:
          - command: "go build"
            description: "verify build"
        documentation_targets: []
        prompt_blocks: []
      success_criteria:
        - "Task compiles"
`

	parser := NewYAMLParser()
	_, err := parser.Parse(strings.NewReader(yaml))
	if err == nil {
		t.Fatal("Expected error for empty data_flow_registry when required")
	}

	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("Expected error about empty registry, got: %v", err)
	}
}

func TestParseDataFlowRegistry_InvalidProducerFormat(t *testing.T) {
	yaml := `
conductor:
  default_agent: "golang-pro"

data_flow_registry:
  producers:
    InvalidSymbol:
      - not_a_valid_entry

plan:
  metadata:
    feature_name: "Invalid Format Test"
    created: "2025-12-01"
    estimated_tasks: 1
  tasks:
    - task_number: 1
      name: "Test Task"
      files:
        - "test.go"
      depends_on: []
      description: "Test task"
      success_criteria:
        - "Task compiles"
`

	parser := NewYAMLParser()
	_, err := parser.Parse(strings.NewReader(yaml))
	if err == nil {
		t.Fatal("Expected error for invalid producer format")
	}
}

func TestParseDataFlowRegistry_WithDocumentationTargets(t *testing.T) {
	yaml := `
conductor:
  default_agent: "golang-pro"

data_flow_registry:
  producers:
    SomeStruct:
      - task: "1"
        description: "Defines struct"
  consumers:
    SomeStruct:
      - task: "2"
        description: "Uses struct"
  documentation_targets:
    "1":
      - location: "docs/api.md"
        section: "Data Types"
      - location: "CHANGELOG.md"
        section: "Added"
    "2":
      - location: "docs/usage.md"
        section: "Examples"

plan:
  metadata:
    feature_name: "Doc Targets Test"
    created: "2025-12-01"
    estimated_tasks: 2
  tasks:
    - task_number: 1
      name: "Define struct"
      files:
        - "types.go"
      depends_on: []
      description: "Define the struct"
      success_criteria:
        - "Struct defined"
    - task_number: 2
      name: "Use struct"
      files:
        - "usage.go"
      depends_on: [1]
      description: "Use the struct"
      success_criteria:
        - "Struct used"
`

	parser := NewYAMLParser()
	plan, err := parser.Parse(strings.NewReader(yaml))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if plan.DataFlowRegistry == nil {
		t.Fatal("Expected DataFlowRegistry to be populated")
	}

	// Check documentation targets
	if len(plan.DataFlowRegistry.DocumentationTargets) != 2 {
		t.Errorf("Expected 2 documentation target entries, got %d", len(plan.DataFlowRegistry.DocumentationTargets))
	}

	task1Docs := plan.DataFlowRegistry.DocumentationTargets["1"]
	if len(task1Docs) != 2 {
		t.Errorf("Expected 2 doc targets for task 1, got %d", len(task1Docs))
	}

	if task1Docs[0].Location != "docs/api.md" {
		t.Errorf("Expected first doc location 'docs/api.md', got %s", task1Docs[0].Location)
	}
}

func TestParseDataFlowRegistry_Fixture(t *testing.T) {
	parser := NewYAMLParser()
	plan, err := parser.ParseFile("testdata/runtime_enforcement.yaml")
	if err != nil {
		t.Fatalf("Failed to parse fixture: %v", err)
	}

	// Verify basic parsing worked
	if plan.PlannerCompliance == nil {
		t.Fatal("Expected PlannerCompliance to be set")
	}

	// Verify data_flow_registry was parsed from fixture
	if plan.DataFlowRegistry == nil {
		t.Fatal("Expected DataFlowRegistry to be set from fixture")
	}

	// Verify producers
	if len(plan.DataFlowRegistry.Producers) != 2 {
		t.Errorf("Expected 2 producer symbols, got %d", len(plan.DataFlowRegistry.Producers))
	}

	rc := plan.DataFlowRegistry.Producers["RuntimeConfig"]
	if len(rc) != 1 || rc[0].TaskNumber != "1" {
		t.Errorf("Expected RuntimeConfig producer from task 1, got %v", rc)
	}

	// Verify consumers (RuntimeConfig has 2 consumers, EnforcementResult has 1)
	if len(plan.DataFlowRegistry.Consumers) != 2 {
		t.Errorf("Expected 2 consumer symbols, got %d", len(plan.DataFlowRegistry.Consumers))
	}

	// Verify documentation targets (3 tasks: "1", "2", "3")
	if len(plan.DataFlowRegistry.DocumentationTargets) != 3 {
		t.Errorf("Expected 3 documentation target entries, got %d", len(plan.DataFlowRegistry.DocumentationTargets))
	}

	task1Docs := plan.DataFlowRegistry.DocumentationTargets["1"]
	if len(task1Docs) != 2 {
		t.Errorf("Expected 2 doc targets for task 1, got %d", len(task1Docs))
	}
}

func TestParseDataFlowRegistry_MergedPlansDeduplicateProducers(t *testing.T) {
	// Test that when merging plans, producers are deduplicated
	registry1 := &models.DataFlowRegistry{
		Producers: map[string][]models.DataFlowEntry{
			"Symbol1": {
				{TaskNumber: "1", Description: "First producer"},
			},
		},
		Consumers: map[string][]models.DataFlowEntry{},
	}

	registry2 := &models.DataFlowRegistry{
		Producers: map[string][]models.DataFlowEntry{
			"Symbol1": {
				{TaskNumber: "2", Description: "Second producer"},
			},
			"Symbol2": {
				{TaskNumber: "3", Description: "Third producer"},
			},
		},
		Consumers: map[string][]models.DataFlowEntry{},
	}

	merged := MergeDataFlowRegistries(registry1, registry2)

	// Symbol1 should have both producers
	if len(merged.Producers["Symbol1"]) != 2 {
		t.Errorf("Expected 2 producers for Symbol1, got %d", len(merged.Producers["Symbol1"]))
	}

	// Symbol2 should have 1 producer
	if len(merged.Producers["Symbol2"]) != 1 {
		t.Errorf("Expected 1 producer for Symbol2, got %d", len(merged.Producers["Symbol2"]))
	}
}

func TestMergePlans_PreservesDataFlowRegistry(t *testing.T) {
	// Critical test: verify MergePlans() actually merges DataFlowRegistry and PlannerCompliance
	plan1 := &models.Plan{
		Name:     "Plan 1",
		FilePath: "/path/plan1.yaml",
		PlannerCompliance: &models.PlannerComplianceSpec{
			PlannerVersion:    "1.0.0",
			StrictEnforcement: true,
			RequiredFeatures:  []string{"data_flow_registry"},
		},
		DataFlowRegistry: &models.DataFlowRegistry{
			Producers: map[string][]models.DataFlowEntry{
				"Symbol1": {{TaskNumber: "1", Description: "Plan1 producer"}},
			},
			Consumers: map[string][]models.DataFlowEntry{},
			DocumentationTargets: map[string][]models.DocumentationTarget{
				"1": {{Location: "docs/plan1.md", Section: "API"}},
			},
		},
		Tasks: []models.Task{
			{Number: "1", Name: "Task 1"},
		},
	}

	plan2 := &models.Plan{
		Name:     "Plan 2",
		FilePath: "/path/plan2.yaml",
		DataFlowRegistry: &models.DataFlowRegistry{
			Producers: map[string][]models.DataFlowEntry{
				"Symbol1": {{TaskNumber: "2", Description: "Plan2 producer"}},
				"Symbol2": {{TaskNumber: "2", Description: "New symbol"}},
			},
			Consumers: map[string][]models.DataFlowEntry{
				"Symbol1": {{TaskNumber: "2", Description: "Plan2 consumes Symbol1"}},
			},
		},
		Tasks: []models.Task{
			{Number: "2", Name: "Task 2"},
		},
	}

	merged, err := MergePlans(plan1, plan2)
	if err != nil {
		t.Fatalf("MergePlans failed: %v", err)
	}

	// Verify PlannerCompliance was preserved (from first plan)
	if merged.PlannerCompliance == nil {
		t.Fatal("MergePlans should preserve PlannerCompliance from first plan")
	}
	if merged.PlannerCompliance.PlannerVersion != "1.0.0" {
		t.Errorf("Expected planner_version 1.0.0, got %s", merged.PlannerCompliance.PlannerVersion)
	}

	// Verify DataFlowRegistry was merged
	if merged.DataFlowRegistry == nil {
		t.Fatal("MergePlans should merge DataFlowRegistry from all plans")
	}

	// Symbol1 should have 2 producers (one from each plan)
	if len(merged.DataFlowRegistry.Producers["Symbol1"]) != 2 {
		t.Errorf("Expected 2 producers for Symbol1 after merge, got %d",
			len(merged.DataFlowRegistry.Producers["Symbol1"]))
	}

	// Symbol2 should have 1 producer (from plan2)
	if len(merged.DataFlowRegistry.Producers["Symbol2"]) != 1 {
		t.Errorf("Expected 1 producer for Symbol2, got %d",
			len(merged.DataFlowRegistry.Producers["Symbol2"]))
	}

	// Consumers should be merged
	if len(merged.DataFlowRegistry.Consumers["Symbol1"]) != 1 {
		t.Errorf("Expected 1 consumer for Symbol1, got %d",
			len(merged.DataFlowRegistry.Consumers["Symbol1"]))
	}

	// DocumentationTargets should be merged
	if len(merged.DataFlowRegistry.DocumentationTargets["1"]) != 1 {
		t.Errorf("Expected 1 doc target for task 1, got %d",
			len(merged.DataFlowRegistry.DocumentationTargets["1"]))
	}

	// Verify tasks were merged
	if len(merged.Tasks) != 2 {
		t.Errorf("Expected 2 tasks after merge, got %d", len(merged.Tasks))
	}
}

func TestValidateDataFlowRegistry(t *testing.T) {
	tests := []struct {
		name      string
		registry  *models.DataFlowRegistry
		required  bool
		wantError bool
		errorMsg  string
	}{
		{
			name:      "nil registry when not required",
			registry:  nil,
			required:  false,
			wantError: false,
		},
		{
			name:      "nil registry when required",
			registry:  nil,
			required:  true,
			wantError: true,
			errorMsg:  "data_flow_registry is required",
		},
		{
			name: "empty registry when required",
			registry: &models.DataFlowRegistry{
				Producers: map[string][]models.DataFlowEntry{},
				Consumers: map[string][]models.DataFlowEntry{},
			},
			required:  true,
			wantError: true,
			errorMsg:  "empty",
		},
		{
			name: "valid registry",
			registry: &models.DataFlowRegistry{
				Producers: map[string][]models.DataFlowEntry{
					"Symbol1": {{TaskNumber: "1"}},
				},
				Consumers: map[string][]models.DataFlowEntry{},
			},
			required:  true,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDataFlowRegistry(tt.registry, tt.required)
			if tt.wantError {
				if err == nil {
					t.Error("Expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
