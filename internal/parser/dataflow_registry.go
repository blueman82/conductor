package parser

import (
	"fmt"

	"github.com/harrison/conductor/internal/models"
)

// yamlDataFlowRegistry represents the YAML structure for data flow registry
type yamlDataFlowRegistry struct {
	Producers            map[string][]yamlDataFlowEntry       `yaml:"producers"`
	Consumers            map[string][]yamlDataFlowEntry       `yaml:"consumers"`
	DocumentationTargets map[string][]yamlDocumentationTarget `yaml:"documentation_targets"`
}

// yamlDataFlowEntry represents a single producer/consumer entry in YAML
type yamlDataFlowEntry struct {
	Task        interface{} `yaml:"task"` // Can be int or string
	Symbol      string      `yaml:"symbol,omitempty"`
	Description string      `yaml:"description,omitempty"`
}

// ParseDataFlowRegistry parses the data_flow_registry section from YAML
func ParseDataFlowRegistry(raw *yamlDataFlowRegistry) (*models.DataFlowRegistry, error) {
	if raw == nil {
		return nil, nil
	}

	registry := &models.DataFlowRegistry{
		Producers:            make(map[string][]models.DataFlowEntry),
		Consumers:            make(map[string][]models.DataFlowEntry),
		DocumentationTargets: make(map[string][]models.DocumentationTarget),
	}

	// Parse producers
	for symbol, entries := range raw.Producers {
		for i, entry := range entries {
			taskNum, err := convertToString(entry.Task)
			if err != nil {
				return nil, fmt.Errorf("data_flow_registry.producers[%s][%d]: invalid task: %w", symbol, i, err)
			}
			registry.Producers[symbol] = append(registry.Producers[symbol], models.DataFlowEntry{
				TaskNumber:  taskNum,
				Symbol:      entry.Symbol,
				Description: entry.Description,
			})
		}
	}

	// Parse consumers
	for symbol, entries := range raw.Consumers {
		for i, entry := range entries {
			taskNum, err := convertToString(entry.Task)
			if err != nil {
				return nil, fmt.Errorf("data_flow_registry.consumers[%s][%d]: invalid task: %w", symbol, i, err)
			}
			registry.Consumers[symbol] = append(registry.Consumers[symbol], models.DataFlowEntry{
				TaskNumber:  taskNum,
				Symbol:      entry.Symbol,
				Description: entry.Description,
			})
		}
	}

	// Parse documentation targets
	for taskNum, targets := range raw.DocumentationTargets {
		for _, target := range targets {
			registry.DocumentationTargets[taskNum] = append(registry.DocumentationTargets[taskNum], models.DocumentationTarget{
				Location: target.Location,
				Section:  target.Section,
			})
		}
	}

	return registry, nil
}

// ValidateDataFlowRegistry validates the data flow registry
// If required is true, returns error if registry is nil or empty
func ValidateDataFlowRegistry(registry *models.DataFlowRegistry, required bool) error {
	if registry == nil {
		if required {
			return fmt.Errorf("data_flow_registry is required when required_features includes 'data_flow_registry'")
		}
		return nil
	}

	if required {
		// Check if registry is effectively empty
		if len(registry.Producers) == 0 && len(registry.Consumers) == 0 {
			return fmt.Errorf("data_flow_registry cannot be empty when required")
		}
	}

	return nil
}

// MergeDataFlowRegistries combines multiple registries into one
// Producers and consumers for the same symbol are combined (not deduplicated)
func MergeDataFlowRegistries(registries ...*models.DataFlowRegistry) *models.DataFlowRegistry {
	merged := &models.DataFlowRegistry{
		Producers:            make(map[string][]models.DataFlowEntry),
		Consumers:            make(map[string][]models.DataFlowEntry),
		DocumentationTargets: make(map[string][]models.DocumentationTarget),
	}

	for _, reg := range registries {
		if reg == nil {
			continue
		}

		// Merge producers
		for symbol, entries := range reg.Producers {
			merged.Producers[symbol] = append(merged.Producers[symbol], entries...)
		}

		// Merge consumers
		for symbol, entries := range reg.Consumers {
			merged.Consumers[symbol] = append(merged.Consumers[symbol], entries...)
		}

		// Merge documentation targets
		for taskNum, targets := range reg.DocumentationTargets {
			merged.DocumentationTargets[taskNum] = append(merged.DocumentationTargets[taskNum], targets...)
		}
	}

	return merged
}

// IsDataFlowRegistryRequired checks if data_flow_registry is in required_features
func IsDataFlowRegistryRequired(compliance *models.PlannerComplianceSpec) bool {
	if compliance == nil {
		return false
	}

	for _, feature := range compliance.RequiredFeatures {
		if feature == "data_flow_registry" {
			return true
		}
	}
	return false
}
