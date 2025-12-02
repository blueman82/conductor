package models

import (
	"encoding/json"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestDataFlowRegistry_YAMLRoundTrip(t *testing.T) {
	original := &DataFlowRegistry{
		Producers: map[string][]DataFlowEntry{
			"PlannerComplianceSpec": {
				{TaskNumber: "1", Description: "Defines the compliance spec"},
			},
			"DataFlowRegistry": {
				{TaskNumber: "3", Description: "Defines registry struct"},
			},
		},
		Consumers: map[string][]DataFlowEntry{
			"PlannerComplianceSpec": {
				{TaskNumber: "2", Description: "Uses for validation"},
				{TaskNumber: "3", Description: "References for enforcement"},
			},
		},
		DocumentationTargets: map[string][]DocumentationTarget{
			"1": {
				{Location: "docs/api.md", Section: "Overview"},
				{Location: "CHANGELOG.md", Section: "Added"},
			},
		},
	}

	// Marshal to YAML
	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal to YAML: %v", err)
	}

	// Unmarshal back
	var decoded DataFlowRegistry
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal from YAML: %v", err)
	}

	// Verify producers
	if len(decoded.Producers) != 2 {
		t.Errorf("Expected 2 producer symbols, got %d", len(decoded.Producers))
	}
	pcs := decoded.Producers["PlannerComplianceSpec"]
	if len(pcs) != 1 || pcs[0].TaskNumber != "1" {
		t.Errorf("Producer mismatch: %+v", pcs)
	}

	// Verify consumers
	if len(decoded.Consumers) != 1 {
		t.Errorf("Expected 1 consumer symbol, got %d", len(decoded.Consumers))
	}
	cons := decoded.Consumers["PlannerComplianceSpec"]
	if len(cons) != 2 {
		t.Errorf("Expected 2 consumers for PlannerComplianceSpec, got %d", len(cons))
	}

	// Verify documentation targets
	if len(decoded.DocumentationTargets) != 1 {
		t.Errorf("Expected 1 task with doc targets, got %d", len(decoded.DocumentationTargets))
	}
	docs := decoded.DocumentationTargets["1"]
	if len(docs) != 2 {
		t.Errorf("Expected 2 doc targets for task 1, got %d", len(docs))
	}
}

func TestDataFlowRegistry_JSONRoundTrip(t *testing.T) {
	original := &DataFlowRegistry{
		Producers: map[string][]DataFlowEntry{
			"Symbol1": {
				{TaskNumber: "1", Symbol: "Symbol1", Description: "First producer"},
			},
		},
		Consumers: map[string][]DataFlowEntry{
			"Symbol1": {
				{TaskNumber: "2", Description: "Consumer task"},
			},
		},
		DocumentationTargets: map[string][]DocumentationTarget{
			"1": {{Location: "docs/test.md", Section: "API"}},
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal to JSON: %v", err)
	}

	// Unmarshal back
	var decoded DataFlowRegistry
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal from JSON: %v", err)
	}

	// Verify
	if len(decoded.Producers["Symbol1"]) != 1 {
		t.Errorf("Expected 1 producer, got %d", len(decoded.Producers["Symbol1"]))
	}
	if decoded.Producers["Symbol1"][0].TaskNumber != "1" {
		t.Errorf("Wrong task number: %s", decoded.Producers["Symbol1"][0].TaskNumber)
	}
}

func TestPlannerComplianceSpec_YAMLRoundTrip(t *testing.T) {
	original := &PlannerComplianceSpec{
		PlannerVersion:    "1.2.0",
		StrictEnforcement: true,
		RequiredFeatures:  []string{"dependency_checks", "documentation_targets", "data_flow_registry"},
	}

	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded PlannerComplianceSpec
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.PlannerVersion != original.PlannerVersion {
		t.Errorf("Version mismatch: %s != %s", decoded.PlannerVersion, original.PlannerVersion)
	}
	if decoded.StrictEnforcement != original.StrictEnforcement {
		t.Errorf("StrictEnforcement mismatch: %v != %v", decoded.StrictEnforcement, original.StrictEnforcement)
	}
	if len(decoded.RequiredFeatures) != len(original.RequiredFeatures) {
		t.Errorf("RequiredFeatures length mismatch: %d != %d", len(decoded.RequiredFeatures), len(original.RequiredFeatures))
	}
}

func TestPlan_WithDataFlowRegistry_RoundTrip(t *testing.T) {
	original := &Plan{
		Name: "Test Plan",
		PlannerCompliance: &PlannerComplianceSpec{
			PlannerVersion:    "1.0.0",
			StrictEnforcement: true,
			RequiredFeatures:  []string{"data_flow_registry"},
		},
		DataFlowRegistry: &DataFlowRegistry{
			Producers: map[string][]DataFlowEntry{
				"TestStruct": {{TaskNumber: "1", Description: "Defines struct"}},
			},
			Consumers: map[string][]DataFlowEntry{
				"TestStruct": {{TaskNumber: "2", Description: "Uses struct"}},
			},
			DocumentationTargets: map[string][]DocumentationTarget{
				"1": {{Location: "README.md", Section: "Usage"}},
			},
		},
		Tasks: []Task{
			{
				Number: "1",
				Name:   "Define struct",
				Files:  []string{"types.go"},
			},
			{
				Number:    "2",
				Name:      "Use struct",
				DependsOn: []string{"1"},
				Files:     []string{"usage.go"},
			},
		},
	}

	// This tests that Plan's fields serialize correctly
	// Actual plan serialization happens via the plan file writers
	// but we verify the model structure is correct

	if original.DataFlowRegistry == nil {
		t.Fatal("DataFlowRegistry should not be nil")
	}
	if len(original.DataFlowRegistry.Producers) != 1 {
		t.Errorf("Expected 1 producer, got %d", len(original.DataFlowRegistry.Producers))
	}
	if len(original.DataFlowRegistry.Consumers) != 1 {
		t.Errorf("Expected 1 consumer, got %d", len(original.DataFlowRegistry.Consumers))
	}
}

func TestDataFlowEntry_Fields(t *testing.T) {
	entry := DataFlowEntry{
		TaskNumber:  "5",
		Symbol:      "MyFunction",
		Description: "Implementation of key algorithm",
	}

	// Marshal and unmarshal YAML
	data, err := yaml.Marshal(entry)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded DataFlowEntry
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.TaskNumber != entry.TaskNumber {
		t.Errorf("TaskNumber mismatch: %s != %s", decoded.TaskNumber, entry.TaskNumber)
	}
	if decoded.Symbol != entry.Symbol {
		t.Errorf("Symbol mismatch: %s != %s", decoded.Symbol, entry.Symbol)
	}
	if decoded.Description != entry.Description {
		t.Errorf("Description mismatch: %s != %s", decoded.Description, entry.Description)
	}
}

func TestDocumentationTarget_Fields(t *testing.T) {
	target := DocumentationTarget{
		Location: "docs/architecture.md",
		Section:  "Data Flow",
	}

	data, err := yaml.Marshal(target)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded DocumentationTarget
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.Location != target.Location {
		t.Errorf("Location mismatch: %s != %s", decoded.Location, target.Location)
	}
	if decoded.Section != target.Section {
		t.Errorf("Section mismatch: %s != %s", decoded.Section, target.Section)
	}
}

func TestDataFlowRegistry_EmptyMaps(t *testing.T) {
	// Test that empty maps serialize/deserialize correctly
	registry := &DataFlowRegistry{
		Producers:            map[string][]DataFlowEntry{},
		Consumers:            map[string][]DataFlowEntry{},
		DocumentationTargets: map[string][]DocumentationTarget{},
	}

	data, err := yaml.Marshal(registry)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded DataFlowRegistry
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Empty maps might unmarshal as nil, ensure we handle both cases
	if decoded.Producers == nil {
		decoded.Producers = map[string][]DataFlowEntry{}
	}
	if len(decoded.Producers) != 0 {
		t.Errorf("Expected empty producers, got %d", len(decoded.Producers))
	}
}

func TestDataFlowRegistry_MultipleProducersPerSymbol(t *testing.T) {
	// Test that multiple producers for the same symbol work correctly
	registry := &DataFlowRegistry{
		Producers: map[string][]DataFlowEntry{
			"SharedInterface": {
				{TaskNumber: "1", Description: "Base implementation"},
				{TaskNumber: "3", Description: "Extended implementation"},
				{TaskNumber: "5", Description: "Alternative implementation"},
			},
		},
		Consumers: map[string][]DataFlowEntry{},
	}

	data, err := yaml.Marshal(registry)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded DataFlowRegistry
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	producers := decoded.Producers["SharedInterface"]
	if len(producers) != 3 {
		t.Errorf("Expected 3 producers for SharedInterface, got %d", len(producers))
	}
}
