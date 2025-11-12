package executor

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/harrison/conductor/internal/models"
)

const (
	// DefaultMaxConcurrency is the default maximum number of concurrent tasks per wave
	DefaultMaxConcurrency = 10
)

// parseTaskNumber extracts the numeric portion from task number strings.
// Handles formats like "1", "Task 1", "Task 10", etc.
// Returns a large number (999999) for unparseable strings so they sort last.
func parseTaskNumber(taskNum string) int {
	// Try parsing the string directly first
	if num, err := strconv.Atoi(taskNum); err == nil {
		return num
	}

	// Try extracting number from formats like "Task 1", "Task 10"
	fields := strings.Fields(taskNum)
	for _, field := range fields {
		if num, err := strconv.Atoi(field); err == nil {
			return num
		}
	}

	// Return large number for unparseable strings (they sort last)
	return 999999
}

// DependencyGraph represents a directed graph of task dependencies
type DependencyGraph struct {
	Tasks    map[string]*models.Task
	Edges    map[string][]string // task -> tasks that depend on it (adjacency list: prerequisite -> dependents)
	InDegree map[string]int      // task -> number of dependencies
	Groups   map[int]string      // task number (parsed as int from string) -> group ID
}

// ValidateTasks checks that all task dependencies are valid
func ValidateTasks(tasks []models.Task) error {
	taskMap := make(map[string]bool)
	for _, task := range tasks {
		if task.Number == "" {
			return fmt.Errorf("task has empty task number")
		}
		if taskMap[task.Number] {
			return fmt.Errorf("task %s: duplicate task number", task.Number)
		}
		taskMap[task.Number] = true
	}

	// Validate dependencies
	for _, task := range tasks {
		for _, dep := range task.DependsOn {
			if !taskMap[dep] {
				return fmt.Errorf("task %s (%s): depends on non-existent task %s", task.Number, task.Name, dep)
			}
		}
	}

	return nil
}

// ValidateFileOverlaps checks that tasks within the same wave do not modify
// the same files. Tasks in different waves are allowed to modify the same files.
// If any task has empty Files, validation is skipped for that wave with a warning.
//
// Parameters:
//   - waves: Wave grouping from topological sort
//   - tasks: Map of task number to task pointer for lookup
//
// Returns error if file overlaps detected in same wave, nil if valid.
func ValidateFileOverlaps(waves []models.Wave, tasks map[string]*models.Task) error {
	for _, wave := range waves {
		tasksInWave := make([]*models.Task, 0, len(wave.TaskNumbers))
		for _, taskNum := range wave.TaskNumbers {
			task, exists := tasks[taskNum]
			if !exists {
				return fmt.Errorf("wave %q: task %s not found in dependency graph", wave.Name, taskNum)
			}
			tasksInWave = append(tasksInWave, task)
		}

		var missingFiles []*models.Task
		for _, task := range tasksInWave {
			if len(task.Files) == 0 {
				missingFiles = append(missingFiles, task)
			}
		}

		if len(missingFiles) > 0 {
			switch len(missingFiles) {
			case 1:
				task := missingFiles[0]
				fmt.Fprintf(os.Stderr, "Warning: wave %q skipping file overlap validation because task %s (%s) has no files configured\n", wave.Name, task.Number, task.Name)
			default:
				entries := make([]string, 0, len(missingFiles))
				for _, task := range missingFiles {
					entries = append(entries, fmt.Sprintf("%s (%s)", task.Number, task.Name))
				}
				fmt.Fprintf(os.Stderr, "Warning: wave %q skipping file overlap validation because tasks %s have no files configured\n", wave.Name, strings.Join(entries, ", "))
			}
			continue
		}

		fileOwners := make(map[string]*models.Task)
		for _, task := range tasksInWave {
			for _, file := range task.Files {
				normalized := filepath.Clean(file)
				if owner, exists := fileOwners[normalized]; exists {
					if owner.Number == task.Number {
						continue
					}
					return fmt.Errorf("wave %q: file %q is assigned to multiple tasks (%s - %s and %s - %s). Move the conflicting tasks to separate waves or adjust each task's file list so only one task can modify the file per wave.", wave.Name, normalized, owner.Number, owner.Name, task.Number, task.Name)
				}
				fileOwners[normalized] = task
			}
		}
	}

	return nil
}

// BuildDependencyGraph constructs a dependency graph from a list of tasks
func BuildDependencyGraph(tasks []models.Task) *DependencyGraph {
	g := &DependencyGraph{
		Tasks:    make(map[string]*models.Task),
		Edges:    make(map[string][]string),
		InDegree: make(map[string]int),
		Groups:   make(map[int]string),
	}

	// Build task map and initialize in-degree
	for i := range tasks {
		g.Tasks[tasks[i].Number] = &tasks[i]
		g.InDegree[tasks[i].Number] = 0

		// Tag node with group info - convert task number to int
		if taskNum, err := strconv.Atoi(tasks[i].Number); err == nil {
			g.Groups[taskNum] = tasks[i].WorktreeGroup
		}
	}

	// Build edges and calculate in-degree
	// Only add edges for valid dependencies
	for _, task := range tasks {
		for _, dep := range task.DependsOn {
			// Validate that dependency exists
			if _, exists := g.Tasks[dep]; !exists {
				// Skip invalid dependencies - they will be caught by validation
				continue
			}
			// dep -> task (dep must complete before task)
			g.Edges[dep] = append(g.Edges[dep], task.Number)
			g.InDegree[task.Number]++
		}
	}

	return g
}

// HasCycle detects if the graph contains a cycle using DFS with color marking
func (g *DependencyGraph) HasCycle() bool {
	const (
		white = 0 // not visited
		gray  = 1 // visiting
		black = 2 // visited
	)

	colors := make(map[string]int)
	for taskNum := range g.Tasks {
		colors[taskNum] = white
	}

	var dfs func(string) bool
	dfs = func(node string) bool {
		colors[node] = gray

		for _, neighbor := range g.Edges[node] {
			if colors[neighbor] == gray {
				return true // back edge = cycle
			}
			if colors[neighbor] == white && dfs(neighbor) {
				return true
			}
		}

		colors[node] = black
		return false
	}

	// Check for self-referencing first
	for taskNum, task := range g.Tasks {
		for _, dep := range task.DependsOn {
			if dep == taskNum {
				return true // self-reference is a cycle
			}
		}
	}

	// Run DFS from all unvisited nodes
	for taskNum := range g.Tasks {
		if colors[taskNum] == white {
			if dfs(taskNum) {
				return true
			}
		}
	}

	return false
}

// CalculateWaves computes execution waves using Kahn's algorithm (topological sort)
// Tasks with no dependencies go in Wave 1, tasks depending only on Wave 1 go in Wave 2, etc.
func CalculateWaves(tasks []models.Task) ([]models.Wave, error) {
	// Validate tasks first
	if err := ValidateTasks(tasks); err != nil {
		return nil, err
	}

	// Handle empty task list
	if len(tasks) == 0 {
		return []models.Wave{}, nil
	}

	graph := BuildDependencyGraph(tasks)

	// Check for cycles
	if graph.HasCycle() {
		return nil, fmt.Errorf("circular dependency detected")
	}

	// Kahn's algorithm for topological sort + wave grouping
	var waves []models.Wave
	inDegree := make(map[string]int)
	for k, v := range graph.InDegree {
		inDegree[k] = v
	}

	for len(inDegree) > 0 {
		// Find all tasks with in-degree 0 (current wave)
		var currentWave []string
		for taskNum, degree := range inDegree {
			if degree == 0 {
				currentWave = append(currentWave, taskNum)
			}
		}

		if len(currentWave) == 0 {
			return nil, fmt.Errorf("graph error: no tasks with zero in-degree")
		}

		// Sort tasks numerically within the wave for better readability
		sort.Slice(currentWave, func(i, j int) bool {
			return parseTaskNumber(currentWave[i]) < parseTaskNumber(currentWave[j])
		})

		// Build group metadata for this wave
		groupInfo := make(map[string][]string)
		for _, taskNum := range currentWave {
			// Look up group from graph
			var groupID string
			if taskNumInt, err := strconv.Atoi(taskNum); err == nil {
				groupID = graph.Groups[taskNumInt]
			}

			// Add task to the group
			groupInfo[groupID] = append(groupInfo[groupID], taskNum)
		}

		// Create wave
		wave := models.Wave{
			Name:           fmt.Sprintf("Wave %d", len(waves)+1),
			TaskNumbers:    currentWave,
			MaxConcurrency: DefaultMaxConcurrency,
			GroupInfo:      groupInfo,
		}
		waves = append(waves, wave)

		// Remove current wave tasks and update in-degrees
		for _, taskNum := range currentWave {
			delete(inDegree, taskNum)

			// Decrease in-degree for dependent tasks
			for _, dependent := range graph.Edges[taskNum] {
				if _, exists := inDegree[dependent]; exists {
					inDegree[dependent]--
				}
			}
		}
	}

	// Validate file overlaps in waves
	taskMap := make(map[string]*models.Task)
	for i := range tasks {
		taskMap[tasks[i].Number] = &tasks[i]
	}
	if err := ValidateFileOverlaps(waves, taskMap); err != nil {
		return nil, err
	}

	return waves, nil
}
