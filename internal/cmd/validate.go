package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

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

// validatePlanDirectory validates a directory containing multiple plan files
// Returns error if validation fails, nil if plan is valid
// output parameter allows redirecting output for testing
func validatePlanDirectory(dirPath string, registry *agent.Registry, output io.Writer) error {
	var errors []string

	// 1. Check if directory exists
	info, err := os.Stat(dirPath)
	if err != nil {
		fmt.Fprintf(output, "✗ Failed to access directory %s\n", dirPath)
		fmt.Fprintf(output, "  Error: %v\n", err)
		return fmt.Errorf("directory access error: %w", err)
	}

	if !info.IsDir() {
		fmt.Fprintf(output, "✗ Path is not a directory: %s\n", dirPath)
		return fmt.Errorf("not a directory: %s", dirPath)
	}

	fmt.Fprintf(output, "✓ Validating multi-file plan from directory: %s\n", dirPath)

	// 2. Find all plan files in directory (.md and .yaml)
	planFiles := []string{}
	err = filepath.Walk(dirPath, func(path string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !fileInfo.IsDir() {
			ext := filepath.Ext(path)
			if ext == ".md" || ext == ".markdown" || ext == ".yaml" || ext == ".yml" {
				planFiles = append(planFiles, path)
			}
		}
		return nil
	})

	if err != nil {
		fmt.Fprintf(output, "✗ Failed to scan directory: %v\n", err)
		return fmt.Errorf("directory scan error: %w", err)
	}

	if len(planFiles) == 0 {
		fmt.Fprintf(output, "✗ No plan files found in directory\n")
		return fmt.Errorf("no plan files found in %s", dirPath)
	}

	fmt.Fprintf(output, "✓ Found %d plan file(s) in directory\n", len(planFiles))

	// 3. Parse all plan files and collect tasks
	allTasks := []models.Task{}
	groupsMap := make(map[string]*models.WorktreeGroup)

	for _, planFile := range planFiles {
		plan, err := parser.ParseFile(planFile)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Failed to parse %s: %v", filepath.Base(planFile), err))
			continue
		}

		allTasks = append(allTasks, plan.Tasks...)

		// Collect worktree groups from each file
		for _, group := range plan.WorktreeGroups {
			groupsMap[group.GroupID] = &group
		}
	}

	fmt.Fprintf(output, "✓ Parsed %d tasks from plan files\n", len(allTasks))

	// 4. Validate individual tasks
	for _, task := range allTasks {
		if err := task.Validate(); err != nil {
			errors = append(errors, fmt.Sprintf("Task %s: %v", task.Number, err))
		}
	}

	// 5. Validate worktree groups
	groupErrors := validateWorktreeGroups(&allTasks, groupsMap)
	if len(groupErrors) > 0 {
		errors = append(errors, groupErrors...)
	} else {
		fmt.Fprintf(output, "✓ All worktree groups are valid\n")
	}

	// 6. Validate cross-file dependencies
	depErrors := validateCrossFileDependencies(&allTasks)
	if len(depErrors) > 0 {
		errors = append(errors, depErrors...)
	} else {
		fmt.Fprintf(output, "✓ All cross-file dependencies are valid\n")
	}

	// 7. Check for circular dependencies
	graph := executor.BuildDependencyGraph(allTasks)
	if graph.HasCycle() {
		errors = append(errors, "Circular dependency detected in task dependencies")
		fmt.Fprintf(output, "✗ Circular dependency detected\n")
	} else {
		fmt.Fprintf(output, "✓ No circular dependencies detected\n")
	}

	// 8. Validate agents
	tempPlan := &models.Plan{
		Tasks:           allTasks,
		DefaultAgent:    "",
		QualityControl:  models.QualityControlConfig{},
		WorktreeGroups:  []models.WorktreeGroup{},
		FileToTaskMap:   map[string][]string{},
	}
	agentErrors := validateAgents(tempPlan, registry)
	if len(agentErrors) == 0 {
		fmt.Fprintf(output, "✓ All agents available\n")
	} else {
		errors = append(errors, agentErrors...)
	}

	// 9. Final validation check
	if len(errors) == 0 {
		fmt.Fprintf(output, "\n✓ Multi-file plan is valid!\n")
		return nil
	}

	// Report all validation errors
	fmt.Fprintf(output, "\n✗ Validation failed for multi-file plan\n")
	for _, errMsg := range errors {
		fmt.Fprintf(output, "  ✗ %s\n", errMsg)
	}
	fmt.Fprintf(output, "\nFound %d validation error(s)!\n", len(errors))

	return fmt.Errorf("validation failed with %d error(s)", len(errors))
}

// validateWorktreeGroups validates that all tasks with worktree_group have valid group assignments
func validateWorktreeGroups(tasks *[]models.Task, groupsMap map[string]*models.WorktreeGroup) []string {
	var errors []string

	for _, task := range *tasks {
		// Check if task has a worktree_group specified
		if task.WorktreeGroup != "" {
			// Verify the group exists
			if _, exists := groupsMap[task.WorktreeGroup]; !exists {
				errors = append(errors, fmt.Sprintf("Task %s: WorktreeGroup '%s' not found in plan configuration", task.Number, task.WorktreeGroup))
			}
		}
	}

	return errors
}

// validateCrossFileDependencies validates that all task dependencies reference valid tasks
func validateCrossFileDependencies(tasks *[]models.Task) []string {
	var errors []string

	// Build map of valid task numbers
	taskMap := make(map[string]bool)
	for _, task := range *tasks {
		taskMap[task.Number] = true
	}

	// Check each task's dependencies
	for _, task := range *tasks {
		for _, dep := range task.DependsOn {
			if !taskMap[dep] {
				errors = append(errors, fmt.Sprintf("Task %s: dependency '%s' not found in any plan file", task.Number, dep))
			}
		}
	}

	return errors
}
