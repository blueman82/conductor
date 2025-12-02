// Package rubric provides strict validation of implementation plans against PlannerComplianceSpec rules.
// It enforces terminology alignment between key_points and success_criteria,
// validates task type constraints (CAPABILITY vs INTEGRATION), and ensures
// documentation targets are specified for documentation tasks.
package rubric

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/harrison/conductor/internal/models"
)

// ValidationError represents a single validation failure with context
type ValidationError struct {
	TaskNumber string
	TaskName   string
	Field      string
	Message    string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("task %s (%s): %s - %s", e.TaskNumber, e.TaskName, e.Field, e.Message)
}

// ValidationResult contains all validation errors from a plan
type ValidationResult struct {
	Errors []ValidationError
}

// Error returns aggregated error message
func (r *ValidationResult) Error() string {
	if len(r.Errors) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("rubric validation failed with %d error(s):\n", len(r.Errors)))
	for _, err := range r.Errors {
		sb.WriteString(fmt.Sprintf("  - %s\n", err.Error()))
	}
	return sb.String()
}

// HasErrors returns true if validation found errors
func (r *ValidationResult) HasErrors() bool {
	return len(r.Errors) > 0
}

// ValidatePlan validates tasks against PlannerComplianceSpec rules.
// Returns nil if validation passes or compliance spec is nil/disabled.
// Returns aggregated error with all violations when strict enforcement is enabled.
func ValidatePlan(tasks []models.Task, spec *models.PlannerComplianceSpec) error {
	// No compliance spec means legacy mode - skip validation
	if spec == nil {
		return nil
	}

	result := &ValidationResult{}

	// Validate planner_version format
	if err := validatePlannerVersion(spec.PlannerVersion); err != nil {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "planner_version",
			Message: err.Error(),
		})
	}

	// If strict enforcement is disabled, only validate version and return
	if !spec.StrictEnforcement {
		if result.HasErrors() {
			return result
		}
		return nil
	}

	// Build required features map for quick lookup
	requiredFeatures := make(map[string]bool)
	for _, f := range spec.RequiredFeatures {
		requiredFeatures[f] = true
	}

	// Validate each task
	for _, task := range tasks {
		validateTask(&task, spec, requiredFeatures, result)
	}

	if result.HasErrors() {
		return result
	}
	return nil
}

// validatePlannerVersion validates the planner_version format (semver)
func validatePlannerVersion(version string) error {
	if version == "" {
		return fmt.Errorf("planner_version is required")
	}

	// Strip leading 'v' if present
	v := strings.TrimPrefix(version, "v")

	// Match semver pattern: MAJOR.MINOR.PATCH
	semverRegex := regexp.MustCompile(`^\d+\.\d+\.\d+$`)
	if !semverRegex.MatchString(v) {
		return fmt.Errorf("invalid planner_version format %q: must be semver (e.g., 1.0.0 or v1.0.0)", version)
	}

	return nil
}

// validateTask validates a single task against compliance rules
func validateTask(task *models.Task, spec *models.PlannerComplianceSpec, requiredFeatures map[string]bool, result *ValidationResult) {
	// Under strict enforcement, runtime_metadata is required
	if task.RuntimeMetadata == nil {
		result.Errors = append(result.Errors, ValidationError{
			TaskNumber: task.Number,
			TaskName:   task.Name,
			Field:      "runtime_metadata",
			Message:    "runtime_metadata is required under strict enforcement",
		})
		return // Can't validate further without metadata
	}

	// Validate key_points are present when success_criteria feature is required
	if requiredFeatures["success_criteria"] {
		if len(task.KeyPoints) == 0 {
			result.Errors = append(result.Errors, ValidationError{
				TaskNumber: task.Number,
				TaskName:   task.Name,
				Field:      "key_points",
				Message:    "key_points are required when success_criteria feature is enforced",
			})
		} else {
			// Validate terminology alignment between key_points and success_criteria
			validateTerminologyAlignment(task, result)
		}
	}

	// Validate task type constraints (CAPABILITY vs INTEGRATION)
	validateTaskTypeConstraints(task, result)

	// Validate documentation_targets for doc tasks
	if requiredFeatures["documentation_targets"] {
		validateDocumentationTargets(task, result)
	}

	// Validate dependency_checks if required
	if requiredFeatures["dependency_checks"] {
		validateDependencyChecks(task, result)
	}
}

// validateTerminologyAlignment checks that key_points terminology appears in success_criteria
func validateTerminologyAlignment(task *models.Task, result *ValidationResult) {
	if len(task.KeyPoints) == 0 || len(task.SuccessCriteria) == 0 {
		return
	}

	// Extract significant terms from key_points
	keyTerms := extractSignificantTerms(task.KeyPoints)
	if len(keyTerms) == 0 {
		return
	}

	// Check if at least some key terms appear in success_criteria
	criteriaText := strings.ToLower(strings.Join(task.SuccessCriteria, " "))
	matchedTerms := 0
	for term := range keyTerms {
		if strings.Contains(criteriaText, strings.ToLower(term)) {
			matchedTerms++
		}
	}

	// Require at least 30% of key terms to appear in criteria
	minRequired := len(keyTerms) * 30 / 100
	if minRequired < 1 {
		minRequired = 1
	}

	if matchedTerms < minRequired {
		result.Errors = append(result.Errors, ValidationError{
			TaskNumber: task.Number,
			TaskName:   task.Name,
			Field:      "key_points/success_criteria",
			Message:    fmt.Sprintf("terminology mismatch: key_points reference terms not found in success_criteria (found %d/%d required terms)", matchedTerms, minRequired),
		})
	}
}

// extractSignificantTerms extracts meaningful terms from key_points for comparison
func extractSignificantTerms(keyPoints []models.KeyPoint) map[string]bool {
	terms := make(map[string]bool)

	// Common stop words to ignore
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"is": true, "are": true, "was": true, "were": true, "be": true,
		"to": true, "of": true, "in": true, "for": true, "on": true,
		"with": true, "as": true, "at": true, "by": true, "from": true,
		"that": true, "this": true, "it": true, "its": true,
		"function": true, "method": true, "class": true, "type": true,
	}

	// Extract significant words from each key point
	for _, kp := range keyPoints {
		// Look for specific patterns like "package.Function" or significant nouns
		words := strings.Fields(kp.Point)
		for _, word := range words {
			// Clean punctuation
			cleaned := strings.Trim(word, ".,;:()[]{}\"'`")
			cleaned = strings.ToLower(cleaned)

			// Skip short words, stop words, and common words
			if len(cleaned) < 4 || stopWords[cleaned] {
				continue
			}

			// Look for compound terms (package.Function, snake_case)
			if strings.Contains(word, ".") || strings.Contains(word, "_") {
				terms[cleaned] = true
				continue
			}

			// Include significant words (longer than 5 chars or capitalized in original)
			if len(cleaned) > 5 {
				terms[cleaned] = true
			}
		}
	}

	return terms
}

// validateTaskTypeConstraints validates CAPABILITY vs INTEGRATION task type rules
func validateTaskTypeConstraints(task *models.Task, result *ValidationResult) {
	taskType := strings.ToLower(task.Type)

	switch taskType {
	case "component", "regular", "":
		// CAPABILITY tasks (component/regular) should NOT have integration_criteria
		if len(task.IntegrationCriteria) > 0 {
			result.Errors = append(result.Errors, ValidationError{
				TaskNumber: task.Number,
				TaskName:   task.Name,
				Field:      "integration_criteria",
				Message:    "component task must not have integration_criteria - these belong only in integration tasks",
			})
		}

	case "integration":
		// INTEGRATION tasks MUST have integration_criteria
		if len(task.IntegrationCriteria) == 0 {
			result.Errors = append(result.Errors, ValidationError{
				TaskNumber: task.Number,
				TaskName:   task.Name,
				Field:      "integration_criteria",
				Message:    "integration task must have integration_criteria defining cross-component validation",
			})
		}
	}
}

// validateDocumentationTargets validates that documentation tasks have proper targets
func validateDocumentationTargets(task *models.Task, result *ValidationResult) {
	// Check if task is documentation-related
	if !isDocumentationTask(task) {
		return
	}

	// Doc tasks must have documentation_targets
	if task.RuntimeMetadata == nil || len(task.RuntimeMetadata.DocumentationTargets) == 0 {
		result.Errors = append(result.Errors, ValidationError{
			TaskNumber: task.Number,
			TaskName:   task.Name,
			Field:      "documentation_targets",
			Message:    "documentation task requires documentation_targets specifying locations to update",
		})
	}
}

// isDocumentationTask determines if a task is documentation-related
func isDocumentationTask(task *models.Task) bool {
	// Check name and prompt for documentation keywords
	docKeywords := []string{
		"document", "documentation", "readme", "docs", "changelog",
		"update doc", "write doc", "add doc", "cli doc", "api doc",
	}

	nameAndPrompt := strings.ToLower(task.Name + " " + task.Prompt)
	for _, keyword := range docKeywords {
		if strings.Contains(nameAndPrompt, keyword) {
			return true
		}
	}

	// Check if any target files are documentation files
	for _, file := range task.Files {
		lower := strings.ToLower(file)
		if strings.HasSuffix(lower, ".md") ||
			strings.Contains(lower, "readme") ||
			strings.Contains(lower, "changelog") ||
			strings.Contains(lower, "docs/") {
			return true
		}
	}

	return false
}

// validateDependencyChecks validates that tasks with dependencies have proper checks
func validateDependencyChecks(task *models.Task, result *ValidationResult) {
	// If task has dependencies, it should have dependency_checks
	// This is a soft validation - we only warn if task explicitly references deps
	if len(task.DependsOn) > 0 && len(task.RuntimeMetadata.DependencyChecks) == 0 {
		// Check if prompt mentions dependency verification
		promptLower := strings.ToLower(task.Prompt)
		if strings.Contains(promptLower, "depend") || strings.Contains(promptLower, "prerequisite") {
			result.Errors = append(result.Errors, ValidationError{
				TaskNumber: task.Number,
				TaskName:   task.Name,
				Field:      "dependency_checks",
				Message:    "task with dependencies should have dependency_checks for prerequisite verification",
			})
		}
	}
}

// =============================================================================
// Data Flow Registry Validation (v2.9+)
// =============================================================================

// ValidateRegistryBindings validates that consumer tasks depend on producer tasks
// for each symbol in the DataFlowRegistry. This ensures data flow contracts are
// enforced at validation time rather than runtime.
//
// Validation rules:
// 1. Every consumed symbol must have at least one producer
// 2. Consumer tasks must depend (directly or transitively) on at least one producer
// 3. If data_flow_registry is in required_features and registry is nil/empty, fail
func ValidateRegistryBindings(plan *models.Plan) error {
	// Check if registry is required
	registryRequired := isRegistryRequired(plan.PlannerCompliance)

	if plan.DataFlowRegistry == nil {
		if registryRequired {
			return fmt.Errorf("data_flow_registry is required when required_features includes 'data_flow_registry'")
		}
		return nil // Not required and not present - OK
	}

	// Build task dependency graph for transitive checks
	taskDeps := buildTransitiveDependencies(plan.Tasks)

	var errors []string

	// For each consumed symbol, verify:
	// 1. There is at least one producer
	// 2. Consumer task depends on at least one producer (directly or transitively)
	for symbol, consumers := range plan.DataFlowRegistry.Consumers {
		producers, hasProducer := plan.DataFlowRegistry.Producers[symbol]
		if !hasProducer || len(producers) == 0 {
			errors = append(errors, fmt.Sprintf("symbol %q is consumed but has no producer", symbol))
			continue
		}

		// Build set of producer task numbers
		producerTasks := make(map[string]bool)
		for _, p := range producers {
			producerTasks[p.TaskNumber] = true
		}

		// Check each consumer depends on at least one producer
		for _, consumer := range consumers {
			consumerTaskNum := consumer.TaskNumber
			consumerDeps := taskDeps[consumerTaskNum]

			dependsOnProducer := false
			for prodTask := range producerTasks {
				if consumerDeps[prodTask] || consumerTaskNum == prodTask {
					dependsOnProducer = true
					break
				}
			}

			if !dependsOnProducer {
				// Find task name for better error message
				taskName := findTaskName(plan.Tasks, consumerTaskNum)
				errors = append(errors, fmt.Sprintf(
					"task %s (%s) consumes %q but does not depend on any producer task",
					consumerTaskNum, taskName, symbol))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("data flow registry validation failed:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}

// buildTransitiveDependencies builds a map of task -> all tasks it depends on (transitively)
func buildTransitiveDependencies(tasks []models.Task) map[string]map[string]bool {
	// Build direct dependency map
	directDeps := make(map[string][]string)
	for _, task := range tasks {
		deps := make([]string, 0, len(task.DependsOn))
		for _, dep := range task.DependsOn {
			// Resolve cross-file dependencies to task ID
			depTaskID := resolveDepToTaskID(dep)
			deps = append(deps, depTaskID)
		}
		directDeps[task.Number] = deps
	}

	// Compute transitive closure for each task
	transitive := make(map[string]map[string]bool)
	for _, task := range tasks {
		visited := make(map[string]bool)
		collectTransitiveDeps(task.Number, directDeps, visited)
		transitive[task.Number] = visited
	}

	return transitive
}

// collectTransitiveDeps recursively collects all transitive dependencies
func collectTransitiveDeps(taskNum string, directDeps map[string][]string, visited map[string]bool) {
	for _, dep := range directDeps[taskNum] {
		if visited[dep] {
			continue
		}
		visited[dep] = true
		collectTransitiveDeps(dep, directDeps, visited)
	}
}

// resolveDepToTaskID extracts the task ID from a dependency string
// Handles both local dependencies and cross-file dependencies
func resolveDepToTaskID(dep string) string {
	if models.IsCrossFileDep(dep) {
		if cfd, err := models.ParseCrossFileDep(dep); err == nil {
			return cfd.TaskID
		}
	}
	return dep
}

// findTaskName finds the task name by number
func findTaskName(tasks []models.Task, taskNum string) string {
	for _, t := range tasks {
		if t.Number == taskNum {
			return t.Name
		}
	}
	return "unknown"
}

// isRegistryRequired checks if data_flow_registry is in required_features
func isRegistryRequired(compliance *models.PlannerComplianceSpec) bool {
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

// =============================================================================
// Documentation Target Validation (v2.9+)
// =============================================================================

// ValidateDocumentationTargets validates that documentation targets point to
// existing files/sections. If basePath is empty, file existence checks are skipped.
func ValidateDocumentationTargets(plan *models.Plan, basePath string) error {
	if plan.DataFlowRegistry == nil {
		return nil
	}

	if len(plan.DataFlowRegistry.DocumentationTargets) == 0 {
		return nil
	}

	// Skip file existence checks if no basePath provided
	if basePath == "" {
		return nil
	}

	var errors []string

	for taskNum, targets := range plan.DataFlowRegistry.DocumentationTargets {
		for _, target := range targets {
			// Check if file exists
			filePath := filepath.Join(basePath, target.Location)
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				errors = append(errors, fmt.Sprintf(
					"task %s: documentation target file %q does not exist",
					taskNum, target.Location))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("documentation target validation failed:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}
