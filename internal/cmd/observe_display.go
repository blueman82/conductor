package cmd

import (
	"fmt"
	"os"

	"github.com/harrison/conductor/internal/behavioral"
	"github.com/mattn/go-isatty"
)

// DisplayPaginatedResults displays paginated results with formatted tables
func DisplayPaginatedResults(items []interface{}, pageSize int) error {
	if len(items) == 0 {
		fmt.Println("No results found")
		return nil
	}

	// Detect if we're in a terminal (for color output)
	colorOutput := isatty.IsTerminal(os.Stdout.Fd())

	// Create paginator
	paginator := behavioral.NewPaginator(items, pageSize)

	// Display current page
	currentItems := paginator.GetCurrentPage()
	rows := behavioral.FormatTable(currentItems, colorOutput)

	// Print header
	printTableHeader()

	// Print rows
	for _, row := range rows {
		fmt.Println(row)
	}

	// Print navigation bar
	navBar := behavioral.PrintNavigationBar(
		paginator.GetCurrentPageNum(),
		paginator.GetTotalPages(),
		colorOutput,
	)
	if navBar != "" {
		fmt.Println(navBar)
	}

	return nil
}

// printTableHeader prints a generic table header
func printTableHeader() string {
	return "" // Header is included in FormatTable
}

// DisplayToolExecutions displays tool execution metrics with pagination
func DisplayToolExecutions(tools []behavioral.ToolExecution, pageSize int) error {
	items := make([]interface{}, len(tools))
	for i, tool := range tools {
		items[i] = tool
	}
	return DisplayPaginatedResults(items, pageSize)
}

// DisplayBashCommands displays bash command metrics with pagination
func DisplayBashCommands(commands []behavioral.BashCommand, pageSize int) error {
	items := make([]interface{}, len(commands))
	for i, cmd := range commands {
		items[i] = cmd
	}
	return DisplayPaginatedResults(items, pageSize)
}

// DisplayFileOperations displays file operation metrics with pagination
func DisplayFileOperations(ops []behavioral.FileOperation, pageSize int) error {
	items := make([]interface{}, len(ops))
	for i, op := range ops {
		items[i] = op
	}
	return DisplayPaginatedResults(items, pageSize)
}

// DisplaySessions displays session metrics with pagination
func DisplaySessions(sessions []behavioral.Session, pageSize int) error {
	items := make([]interface{}, len(sessions))
	for i, session := range sessions {
		items[i] = session
	}
	return DisplayPaginatedResults(items, pageSize)
}

// DisplayMetricsSummary displays a summary of behavioral metrics
func DisplayMetricsSummary(metrics *behavioral.BehavioralMetrics) error {
	colorOutput := isatty.IsTerminal(os.Stdout.Fd())

	fmt.Println("\n=== Behavioral Metrics Summary ===")
	fmt.Printf("Total Sessions: %d\n", metrics.TotalSessions)
	fmt.Printf("Success Rate: %.1f%%\n", metrics.SuccessRate*100)
	fmt.Printf("Average Duration: %s\n", metrics.AverageDuration)
	fmt.Printf("Total Cost: $%.4f\n", metrics.TotalCost)
	fmt.Printf("Error Rate: %.1f%%\n", metrics.ErrorRate*100)
	fmt.Printf("Total Errors: %d\n", metrics.TotalErrors)

	// Display tool executions
	if len(metrics.ToolExecutions) > 0 {
		fmt.Println("\n--- Tool Executions ---")
		err := DisplayToolExecutions(metrics.ToolExecutions, 50)
		if err != nil {
			return err
		}
	}

	// Display bash commands
	if len(metrics.BashCommands) > 0 {
		fmt.Println("\n--- Bash Commands ---")
		err := DisplayBashCommands(metrics.BashCommands, 50)
		if err != nil {
			return err
		}
	}

	// Display file operations
	if len(metrics.FileOperations) > 0 {
		fmt.Println("\n--- File Operations ---")
		err := DisplayFileOperations(metrics.FileOperations, 50)
		if err != nil {
			return err
		}
	}

	// Display agent performance
	if len(metrics.AgentPerformance) > 0 {
		fmt.Println("\n--- Agent Performance ---")
		for agent, count := range metrics.AgentPerformance {
			if colorOutput {
				fmt.Printf("  %s: %d successful tasks\n", agent, count)
			} else {
				fmt.Printf("  %s: %d successful tasks\n", agent, count)
			}
		}
	}

	return nil
}
