package executor

import (
	"fmt"
	"strings"

	"github.com/harrison/conductor/internal/agent"
	"github.com/harrison/conductor/internal/models"
)

// buildIntegrationPrompt enhances integration task prompts with dependency file context
func buildIntegrationPrompt(task models.Task, plan *models.Plan) string {
	// If no dependencies, return original prompt
	if len(task.DependsOn) == 0 {
		return task.Prompt
	}

	var builder strings.Builder

	// Build header
	builder.WriteString("# INTEGRATION TASK CONTEXT\n\n")
	builder.WriteString("Before implementing, you MUST read these dependency files:\n\n")

	// For each dependency
	for _, depNum := range task.DependsOn {
		depTask := findTaskByNumber(plan, depNum)
		if depTask == nil {
			continue
		}

		// Dependency section using XML format with agent helpers
		builder.WriteString(fmt.Sprintf("<dependency task=\"%s\" name=\"%s\">\n", depTask.Number, depTask.Name))
		builder.WriteString("<files_to_read>\n")
		for _, file := range depTask.Files {
			builder.WriteString(agent.XMLTag("file", file))
			builder.WriteString("\n")
		}
		builder.WriteString("</files_to_read>\n")
		builder.WriteString(agent.XMLSection("justification", generateReadJustification(depTask)))
		builder.WriteString("\n</dependency>\n\n")
	}

	// Append original task prompt
	builder.WriteString("---\n\n# YOUR INTEGRATION TASK\n\n")
	builder.WriteString(task.Prompt)

	return builder.String()
}

// findTaskByNumber finds a task in the plan by its number
func findTaskByNumber(plan *models.Plan, taskNum string) *models.Task {
	for i := range plan.Tasks {
		if plan.Tasks[i].Number == taskNum {
			return &plan.Tasks[i]
		}
	}
	return nil
}

// generateReadJustification generates the justification for reading dependency files
func generateReadJustification(task *models.Task) string {
	return fmt.Sprintf("You need to understand the implementation of %s to properly integrate it. "+
		"Read these files to see:\n"+
		"- Exported functions and their signatures\n"+
		"- Data structures and types\n"+
		"- Error handling patterns\n"+
		"- Integration interfaces\n\n"+
		"Use the Read tool to examine these files before proceeding.",
		task.Name)
}
