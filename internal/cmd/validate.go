package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/harrison/conductor/internal/agent"
	"github.com/harrison/conductor/internal/executor"
	"github.com/harrison/conductor/internal/models"
	"github.com/harrison/conductor/internal/parser"
	"github.com/spf13/cobra"
)

// NewValidateCommand creates and returns the validate subcommand
func NewValidateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate <plan-file>",
		Short: "Validate a plan file",
		Long: `Parse and validate a plan file, checking for:
  - Task validation (names, prompts, etc.)
  - Circular dependencies
  - File overlaps in parallel tasks
  - Referenced agents exist
  - Task dependencies point to valid tasks

Exit code: 0 if valid, 1 if errors found`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return validatePlanFile(args[0])
		},
		SilenceUsage: true,
	}

	return cmd
}

// validatePlanFile is the main entry point that uses default registry and stdout
func validatePlanFile(filePath string) error {
	registry := agent.NewRegistry("")
	registry.Discover()
	return validatePlan(filePath, registry, os.Stdout)
}

// validatePlan performs comprehensive validation of a plan file
// Returns error if validation fails, nil if plan is valid
// output parameter allows redirecting output for testing
func validatePlan(filePath string, registry *agent.Registry, output io.Writer) error {
	var errors []string

	// 1. Parse the plan file
	plan, err := parser.ParseFile(filePath)
	if err != nil {
		fmt.Fprintf(output, "✗ Failed to parse plan from %s\n", filePath)
		fmt.Fprintf(output, "  Error: %v\n", err)
		return fmt.Errorf("parse error: %w", err)
	}

	fmt.Fprintf(output, "✓ Validating plan from %s\n", filePath)
	fmt.Fprintf(output, "✓ Parsed %d tasks successfully\n", len(plan.Tasks))

	// 2. Validate individual tasks
	for _, task := range plan.Tasks {
		if err := task.Validate(); err != nil {
			errors = append(errors, fmt.Sprintf("Task %s: %v", task.Number, err))
		}
	}

	// 3. Validate task dependencies (check all deps reference valid tasks)
	if err := executor.ValidateTasks(plan.Tasks); err != nil {
		errors = append(errors, err.Error())
	}

	// 4. Check for circular dependencies
	graph := executor.BuildDependencyGraph(plan.Tasks)
	if graph.HasCycle() {
		errors = append(errors, "Circular dependency detected in task dependencies")
		fmt.Fprintf(output, "✗ Circular dependency detected\n")
	} else {
		fmt.Fprintf(output, "✓ No circular dependencies detected\n")
	}

	// 5. Calculate waves and check file overlaps
	// Only do this if we don't have dependency errors
	if len(errors) == 0 {
		waves, err := executor.CalculateWaves(plan.Tasks)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Wave calculation failed: %v", err))
		} else {
			fmt.Fprintf(output, "✓ No file overlaps in parallel tasks\n")
			// Note: file overlap validation is done inside CalculateWaves
			// If it returned successfully, there are no overlaps
		}
		_ = waves // waves calculated but not used for anything else
	}

	// 6. Validate agents exist
	agentErrors := validateAgents(plan, registry)
	if len(agentErrors) == 0 {
		fmt.Fprintf(output, "✓ All agents available\n")
	} else {
		errors = append(errors, agentErrors...)
	}

	// 7. Final validation check
	if len(errors) == 0 {
		fmt.Fprintf(output, "✓ All task dependencies valid\n")
		fmt.Fprintf(output, "\n✓ Plan is valid!\n")
		return nil
	}

	// Report all validation errors
	fmt.Fprintf(output, "\n✗ Validation failed for plan from %s\n", filePath)
	for _, errMsg := range errors {
		fmt.Fprintf(output, "  ✗ %s\n", errMsg)
	}
	fmt.Fprintf(output, "\nFound %d validation error(s)!\n", len(errors))

	return fmt.Errorf("validation failed with %d error(s)", len(errors))
}

// validateAgents checks that all agents referenced in tasks exist in the registry
func validateAgents(plan *models.Plan, registry *agent.Registry) []string {
	var errors []string
	checkedAgents := make(map[string]bool)

	// Check default agent if specified
	if plan.DefaultAgent != "" && !checkedAgents[plan.DefaultAgent] {
		if !registry.Exists(plan.DefaultAgent) {
			errors = append(errors, fmt.Sprintf("Default agent '%s' not found in registry", plan.DefaultAgent))
		}
		checkedAgents[plan.DefaultAgent] = true
	}

	// Check QC review agent if specified
	if plan.QualityControl.Enabled && plan.QualityControl.ReviewAgent != "" {
		agentName := plan.QualityControl.ReviewAgent
		if !checkedAgents[agentName] {
			if !registry.Exists(agentName) {
				errors = append(errors, fmt.Sprintf("QC review agent '%s' not found in registry", agentName))
			}
			checkedAgents[agentName] = true
		}
	}

	// Check each task's agent
	for _, task := range plan.Tasks {
		if task.Agent != "" && !checkedAgents[task.Agent] {
			if !registry.Exists(task.Agent) {
				errors = append(errors, fmt.Sprintf("Task %s (%s): agent '%s' not found in registry", task.Number, task.Name, task.Agent))
			}
			checkedAgents[task.Agent] = true
		}
	}

	return errors
}
