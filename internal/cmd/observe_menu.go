package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/harrison/conductor/internal/behavioral"
)

const (
	menuPageSize = 15 // Number of projects per page
)

// MenuReader defines interface for reading user input (for testing)
type MenuReader interface {
	ReadString(delim byte) (string, error)
}

// DefaultMenuReader wraps bufio.Reader
type DefaultMenuReader struct {
	reader *bufio.Reader
}

func (d *DefaultMenuReader) ReadString(delim byte) (string, error) {
	return d.reader.ReadString(delim)
}

// DisplayProjectMenu shows interactive project selection menu
// Returns selected project name or error
func DisplayProjectMenu() (string, error) {
	return DisplayProjectMenuWithReader(&DefaultMenuReader{
		reader: bufio.NewReader(os.Stdin),
	})
}

// DisplayProjectMenuWithReader allows injection of reader for testing
func DisplayProjectMenuWithReader(reader MenuReader) (string, error) {
	projects, err := behavioral.ListProjects()
	if err != nil {
		return "", fmt.Errorf("failed to list projects: %w", err)
	}

	if len(projects) == 0 {
		return "", fmt.Errorf("no projects found in ~/.claude/projects")
	}

	// Single page if projects fit
	if len(projects) <= menuPageSize {
		return displaySinglePage(projects, reader)
	}

	// Multi-page navigation
	return displayPaginated(projects, reader)
}

// displaySinglePage shows all projects without pagination
func displaySinglePage(projects []behavioral.ProjectInfo, reader MenuReader) (string, error) {
	bold := color.New(color.Bold)
	cyan := color.New(color.FgCyan)

	bold.Println("\nAvailable Projects:")
	fmt.Println(strings.Repeat("-", 70))

	for i, project := range projects {
		line := formatMenuLine(project, i+1)
		fmt.Println(line)
	}

	fmt.Println(strings.Repeat("-", 70))
	cyan.Print("\nSelect project (1-", len(projects), ") or 'q' to quit: ")

	return readSelection(reader, len(projects), projects)
}

// displayPaginated shows projects with page navigation
func displayPaginated(projects []behavioral.ProjectInfo, reader MenuReader) (string, error) {
	currentPage := 0
	totalPages := (len(projects) + menuPageSize - 1) / menuPageSize

	for {
		start := currentPage * menuPageSize
		end := start + menuPageSize
		if end > len(projects) {
			end = len(projects)
		}

		page := projects[start:end]

		bold := color.New(color.Bold)
		cyan := color.New(color.FgCyan)

		// Clear screen (basic)
		fmt.Print("\033[H\033[2J")

		bold.Printf("\nAvailable Projects (Page %d/%d):\n", currentPage+1, totalPages)
		fmt.Println(strings.Repeat("-", 70))

		for i, project := range page {
			globalIndex := start + i + 1
			line := formatMenuLine(project, globalIndex)
			fmt.Println(line)
		}

		fmt.Println(strings.Repeat("-", 70))
		cyan.Print("\nSelect project (number), 'n' for next, 'p' for prev, 'q' to quit: ")

		input, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("failed to read input: %w", err)
		}

		input = strings.TrimSpace(strings.ToLower(input))

		// Handle navigation
		if input == "q" {
			return "", fmt.Errorf("selection cancelled")
		}
		if input == "n" && currentPage < totalPages-1 {
			currentPage++
			continue
		}
		if input == "p" && currentPage > 0 {
			currentPage--
			continue
		}

		// Try to parse as project number
		var selection int
		_, scanErr := fmt.Sscanf(input, "%d", &selection)
		if scanErr == nil && selection >= 1 && selection <= len(projects) {
			return projects[selection-1].Name, nil
		}

		// Invalid input, stay on current page
		color.Red("Invalid selection. Please try again.")
		fmt.Print("Press Enter to continue...")
		reader.ReadString('\n')
	}
}

// formatMenuLine formats a single menu line with project info
func formatMenuLine(project behavioral.ProjectInfo, index int) string {
	yellow := color.New(color.FgYellow)
	green := color.New(color.FgGreen)

	// Format: [1] project-name (5 sessions, 1.2 MB)
	sizeMB := float64(project.TotalSize) / (1024 * 1024)

	return fmt.Sprintf("  %s %-30s %s",
		yellow.Sprintf("[%d]", index),
		project.Name,
		green.Sprintf("(%d sessions, %.2f MB)", project.SessionCount, sizeMB),
	)
}

// readSelection reads and validates user selection
func readSelection(reader MenuReader, maxSelection int, projects []behavioral.ProjectInfo) (string, error) {
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.TrimSpace(strings.ToLower(input))

	if input == "q" {
		return "", fmt.Errorf("selection cancelled")
	}

	var selection int
	_, err = fmt.Sscanf(input, "%d", &selection)
	if err != nil || selection < 1 || selection > maxSelection {
		return "", fmt.Errorf("invalid selection: must be between 1 and %d", maxSelection)
	}

	return projects[selection-1].Name, nil
}
