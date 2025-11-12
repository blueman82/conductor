package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/harrison/conductor/internal/agent"
	"github.com/harrison/conductor/internal/display"
	"github.com/harrison/conductor/internal/executor"
	"github.com/harrison/conductor/internal/models"
	"github.com/harrison/conductor/internal/parser"
	"github.com/spf13/cobra"
)

// NewValidateCommand creates and returns the validate subcommand
func NewValidateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate <plan-file-or-directory>...",
		Short: "Validate one or more plan files or directories",
		Long: `Parse and validate plan files, checking for:
  - Task validation (names, prompts, etc.)
  - Circular dependencies
  - File overlaps in parallel tasks
  - Referenced agents exist
  - Task dependencies point to valid tasks

Supports multiple input modes:
  - Single file: conductor validate plan.md
  - Single directory: conductor validate docs/plans/ (filters plan-*.md and plan-*.yaml)
  - Multiple files: conductor validate plan-01.md plan-02.yaml
  - Shell globs: conductor validate docs/plans/*/plan-*.md

Exit code: 0 if valid, 1 if errors found`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return validatePlanFileWithOutput(args, cmd.OutOrStdout())
		},
		SilenceUsage: true,
	}

	return cmd
}

// validatePlanFile is the main entry point that uses default registry and stdout
// Supports single file, single directory, or multiple files
func validatePlanFile(paths []string) error {
	return validatePlanFileWithOutput(paths, os.Stdout)
}

// validatePlanFileWithOutput validates plan files with custom output writer (for testing)
func validatePlanFileWithOutput(paths []string, output io.Writer) error {
	registry := agent.NewRegistry("")
	registry.Discover()

	// Handle three cases:
	// 1. Single directory - use existing validatePlanDirectory
	// 2. Single file - use existing validatePlan
	// 3. Multiple files - filter, parse, merge, and validate

	if len(paths) == 1 {
		path := paths[0]
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("failed to access path: %w", err)
		}

		if info.IsDir() {
			// Single directory - filter plan-* files
			planFiles, err := filterPlanFiles([]string{path})
			if err != nil {
				return err
			}
			// Detect and warn about numbered files
			numberedFiles, _ := display.FindNumberedFiles(path)
			if len(numberedFiles) > 0 {
				warning := display.Warning{
					Title:      fmt.Sprintf("Found numbered files (%s) in directory", strings.Join(numberedFiles, ", ")),
					Message:    "Conductor only processes plan-*.{md,yaml} files",
					Suggestion: "To use these files, rename them to: plan-01-setup.md, plan-02-api.yaml, etc.",
				}
				warning.Display(output)
			}
			return validateMultipleFiles(planFiles, registry, output)
		}

		// Single file - use existing validatePlan
		return validatePlan(path, registry, output)
	}

	// Multiple paths provided - filter and validate together
	planFiles, err := filterPlanFiles(paths)
	if err != nil {
		return err
	}

	// Detect and warn about numbered files in directories
	var allNumberedFiles []string
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		if info.IsDir() {
			numberedFiles, _ := display.FindNumberedFiles(path)
			allNumberedFiles = append(allNumberedFiles, numberedFiles...)
		}
	}
	if len(allNumberedFiles) > 0 {
		warning := display.Warning{
			Title:      fmt.Sprintf("Found numbered files (%s) in directory", strings.Join(allNumberedFiles, ", ")),
			Message:    "Conductor only processes plan-*.{md,yaml} files",
			Suggestion: "To use these files, rename them to: plan-01-setup.md, plan-02-api.yaml, etc.",
		}
		warning.Display(output)
	}

	return validateMultipleFiles(planFiles, registry, output)
}

// filterPlanFiles filters paths to only include plan-*.md and plan-*.yaml files
// For directories, scans for matching files
// For files, only includes those matching the pattern
// Returns absolute paths to all matching plan files
func filterPlanFiles(paths []string) ([]string, error) {
	var planFiles []string
	seenFiles := make(map[string]bool) // Deduplicate files

	for _, path := range paths {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path for %s: %w", path, err)
		}

		info, err := os.Stat(absPath)
		if err != nil {
			return nil, fmt.Errorf("failed to access path %s: %w", absPath, err)
		}

		if info.IsDir() {
			// Scan directory for plan-*.md and plan-*.yaml files
			err := filepath.Walk(absPath, func(filePath string, fileInfo os.FileInfo, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}

				if fileInfo.IsDir() {
					return nil
				}

				fileName := filepath.Base(filePath)
				if isPlanFile(fileName) {
					absFilePath, err := filepath.Abs(filePath)
					if err != nil {
						return err
					}
					if !seenFiles[absFilePath] {
						planFiles = append(planFiles, absFilePath)
						seenFiles[absFilePath] = true
					}
				}

				return nil
			})

			if err != nil {
				return nil, fmt.Errorf("failed to scan directory %s: %w", absPath, err)
			}
		} else {
			// Single file - check if it matches plan-* pattern
			fileName := filepath.Base(absPath)
			if isPlanFile(fileName) {
				if !seenFiles[absPath] {
					planFiles = append(planFiles, absPath)
					seenFiles[absPath] = true
				}
			}
		}
	}

	if len(planFiles) == 0 {
		return nil, fmt.Errorf("no plan files matching pattern 'plan-*.md' or 'plan-*.yaml' found")
	}

	return planFiles, nil
}

// isPlanFile checks if a filename matches the plan-* pattern
func isPlanFile(filename string) bool {
	if !strings.HasPrefix(filename, "plan-") {
		return false
	}

	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".md" || ext == ".markdown" || ext == ".yaml" || ext == ".yml"
}

// validateMultipleFiles validates multiple plan files as a merged plan
func validateMultipleFiles(planFiles []string, registry *agent.Registry, output io.Writer) error {
	var errors []string

	// Parse all plan files and collect tasks with progress indicator
	progress := display.NewProgressIndicator(output, len(planFiles))
	fmt.Fprintf(output, "Validating plan files:\n")

	allTasks := []models.Task{}
	groupsMap := make(map[string]*models.WorktreeGroup)
	var defaultAgent string
	var qcConfig models.QualityControlConfig

	for _, planFile := range planFiles {
		progress.Step(planFile)

		plan, err := parser.ParseFile(planFile)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to parse %s: %v", filepath.Base(planFile), err)
			errors = append(errors, errMsg)
			fmt.Fprintf(output, "✗ %s\n", errMsg)
			continue
		}

		allTasks = append(allTasks, plan.Tasks...)

		// Collect worktree groups from each file
		for _, group := range plan.WorktreeGroups {
			groupsMap[group.GroupID] = &group
		}

		// Use first non-empty default agent
		if defaultAgent == "" && plan.DefaultAgent != "" {
			defaultAgent = plan.DefaultAgent
		}

		// Use first QC config that's enabled
		if !qcConfig.Enabled && plan.QualityControl.Enabled {
			qcConfig = plan.QualityControl
		}
	}

	// Show completion message in green
	fmt.Fprintf(output, "\x1b[32m✓\x1b[0m Parsed %d tasks from %d plan files\n", len(allTasks), len(planFiles))

	// Validate individual tasks
	for _, task := range allTasks {
		if err := task.Validate(); err != nil {
			errors = append(errors, fmt.Sprintf("Task %s: %v", task.Number, err))
		}
	}

	// Validate task dependencies (check all deps reference valid tasks)
	if err := executor.ValidateTasks(allTasks); err != nil {
		errors = append(errors, err.Error())
	}

	// Check for circular dependencies
	graph := executor.BuildDependencyGraph(allTasks)
	if graph.HasCycle() {
		errors = append(errors, "Circular dependency detected in task dependencies")
		fmt.Fprintf(output, "✗ Circular dependency detected\n")
	} else {
		fmt.Fprintf(output, "✓ No circular dependencies detected\n")
	}

	// Calculate waves and check file overlaps (only if no dependency errors)
	if len(errors) == 0 {
		waves, err := executor.CalculateWaves(allTasks)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Wave calculation failed: %v", err))
		} else {
			fmt.Fprintf(output, "✓ No file overlaps in parallel tasks\n")
		}
		_ = waves // waves calculated but not used for anything else
	}

	// Validate agents
	tempPlan := &models.Plan{
		Tasks:          allTasks,
		DefaultAgent:   defaultAgent,
		QualityControl: qcConfig,
	}
	agentErrors := validateAgents(tempPlan, registry)
	if len(agentErrors) == 0 {
		fmt.Fprintf(output, "✓ All agents available\n")
	} else {
		errors = append(errors, agentErrors...)
	}

	// Final validation check
	if len(errors) == 0 {
		fmt.Fprintf(output, "✓ All task dependencies valid\n")
		fmt.Fprintf(output, "\n✓ Plan is valid!\n")
		return nil
	}

	// Report all validation errors
	fmt.Fprintf(output, "\n✗ Validation failed\n")
	for _, errMsg := range errors {
		fmt.Fprintf(output, "  ✗ %s\n", errMsg)
	}
	fmt.Fprintf(output, "\nFound %d validation error(s)!\n", len(errors))

	return fmt.Errorf("validation failed with %d error(s)", len(errors))
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

	// 7. Validate worktree groups
	groupsMap := make(map[string]*models.WorktreeGroup)
	for _, group := range plan.WorktreeGroups {
		groupsMap[group.GroupID] = &group
	}
	groupErrors := validateWorktreeGroups(&plan.Tasks, groupsMap)
	if len(groupErrors) > 0 {
		errors = append(errors, groupErrors...)
	} else if len(plan.WorktreeGroups) > 0 || hasWorktreeGroupAssignments(plan.Tasks) {
		fmt.Fprintf(output, "✓ All worktree groups are valid\n")
	}

	// 8. Final validation check
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
		Tasks:          allTasks,
		DefaultAgent:   "",
		QualityControl: models.QualityControlConfig{},
		WorktreeGroups: []models.WorktreeGroup{},
		FileToTaskMap:  map[string][]string{},
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

// hasWorktreeGroupAssignments checks if any task has a worktree group assigned
func hasWorktreeGroupAssignments(tasks []models.Task) bool {
	for _, task := range tasks {
		if task.WorktreeGroup != "" {
			return true
		}
	}
	return false
}
