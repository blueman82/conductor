package agent

import (
	"fmt"
	"strings"

	"github.com/harrison/conductor/internal/models"
)

// ValidationError represents an agent validation failure
type ValidationError struct {
	AgentName  string   // Name of the missing agent
	UsageType  string   // "task" or "quality_control"
	TaskNumber int      // For task agents only (0 if N/A)
	Available  []string // Available agent names
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("Agent '%s' not found in registry", e.AgentName))

	if e.UsageType == "task" {
		msg.WriteString(fmt.Sprintf(" (referenced in task %d)", e.TaskNumber))
	} else if e.UsageType == "quality_control" {
		msg.WriteString(" (referenced in quality_control.agents config)")
	}

	if len(e.Available) > 0 {
		msg.WriteString(fmt.Sprintf("\n\nAvailable agents: %s", strings.Join(e.Available, ", ")))
	} else {
		msg.WriteString("\n\nNo agents found in registry. Check ~/.claude/agents/")
	}

	return msg.String()
}

// ValidateTaskAgents checks all task agent references exist in registry
// Returns a slice of ValidationErrors for each missing agent
// Empty agent names are skipped (tasks don't require an agent)
func ValidateTaskAgents(tasks []models.Task, registry *Registry) []ValidationError {
	if registry == nil {
		return nil // No registry, no validation
	}

	var errors []ValidationError
	available := registry.ListNames()

	for _, task := range tasks {
		if task.Agent == "" {
			continue // No agent specified - OK
		}

		if !registry.Exists(task.Agent) {
			// Convert Number string to int for error reporting
			taskNum := 0
			if task.Number != "" {
				// Try to parse as int for error message
				fmt.Sscanf(task.Number, "%d", &taskNum)
			}

			errors = append(errors, ValidationError{
				AgentName:  task.Agent,
				UsageType:  "task",
				TaskNumber: taskNum,
				Available:  available,
			})
		}
	}

	return errors
}

// ValidateQCAgents checks QC agent references exist in registry
// Returns a slice of ValidationErrors for each missing agent
// Empty registry results in errors for all agents (no agents available)
func ValidateQCAgents(qcAgents []string, registry *Registry) []ValidationError {
	if registry == nil {
		return nil // No registry, no validation
	}

	var errors []ValidationError
	available := registry.ListNames()

	for _, agentName := range qcAgents {
		if agentName == "" {
			continue // Skip empty agent names
		}

		if !registry.Exists(agentName) {
			errors = append(errors, ValidationError{
				AgentName: agentName,
				UsageType: "quality_control",
				Available: available,
			})
		}
	}

	return errors
}
