package parser

import (
	"fmt"
	"strings"

	"github.com/harrison/conductor/internal/models"
)

// ValidateTaskType validates the task type field
func ValidateTaskType(task *models.Task) error {
	if task.Type == "" {
		return nil // Type is optional
	}

	// Normalize and validate
	task.Type = strings.ToLower(strings.TrimSpace(task.Type))
	validTypes := map[string]bool{
		"regular":     true,
		"integration": true,
		"component":   true,
	}

	if !validTypes[task.Type] {
		return fmt.Errorf("invalid task type %q: must be 'regular', 'integration', or 'component'", task.Type)
	}

	return nil
}

// ValidateIntegrationTask validates integration-specific requirements
func ValidateIntegrationTask(task *models.Task) error {
	if task.Type != "integration" {
		return nil
	}

	// Integration tasks should have dependencies
	if len(task.DependsOn) == 0 {
		return fmt.Errorf("integration task %s must have dependencies", task.Number)
	}

	return nil
}
