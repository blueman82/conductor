package models

import (
	"errors"
	"time"
)

// Task represents a single task in an implementation plan
type Task struct {
	Number        int           // Task number/identifier
	Name          string        // Task name/title
	Files         []string      // Files to be modified/created
	DependsOn     []int         // Task numbers this task depends on
	EstimatedTime time.Duration // Estimated time to complete
	Agent         string        // Agent to use (optional)
	Prompt        string        // Full task description/prompt
}

// Validate checks if the task has all required fields
func (t *Task) Validate() error {
	if t.Number <= 0 {
		return errors.New("task number must be positive")
	}
	if t.Name == "" {
		return errors.New("task name is required")
	}
	if t.Prompt == "" {
		return errors.New("task prompt is required")
	}
	return nil
}

// HasCyclicDependencies detects circular dependencies in a list of tasks
// using DFS with color marking (white=unvisited, gray=visiting, black=visited)
func HasCyclicDependencies(tasks []Task) bool {
	// Build adjacency list: task number -> list of dependent task numbers
	graph := make(map[int][]int)
	taskMap := make(map[int]bool)

	// Initialize graph and track valid task numbers
	for _, task := range tasks {
		taskMap[task.Number] = true
		graph[task.Number] = []int{}
	}

	// Build edges: if task A depends on B, then B -> A
	for _, task := range tasks {
		for _, dep := range task.DependsOn {
			// Check for self-reference
			if dep == task.Number {
				return true
			}
			// Only add edge if dependency exists
			if taskMap[dep] {
				graph[dep] = append(graph[dep], task.Number)
			}
		}
	}

	// DFS with color marking
	const (
		white = 0 // not visited
		gray  = 1 // currently visiting
		black = 2 // visited
	)

	colors := make(map[int]int)
	for taskNum := range taskMap {
		colors[taskNum] = white
	}

	// DFS function to detect back edges (cycles)
	var dfs func(int) bool
	dfs = func(node int) bool {
		colors[node] = gray

		for _, neighbor := range graph[node] {
			if colors[neighbor] == gray {
				// Back edge found - cycle detected
				return true
			}
			if colors[neighbor] == white && dfs(neighbor) {
				return true
			}
		}

		colors[node] = black
		return false
	}

	// Run DFS from each unvisited node
	for taskNum := range taskMap {
		if colors[taskNum] == white {
			if dfs(taskNum) {
				return true
			}
		}
	}

	return false
}
