package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/harrison/conductor/internal/budget"
	"github.com/harrison/conductor/internal/config"
	"github.com/spf13/cobra"
)

var (
	budgetJSON    bool
	budgetBaseDir string
)

// NewBudgetCommand creates the budget command with subcommands
func NewBudgetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "budget",
		Short: "Show Claude Code usage budget status",
		Long: `Display current 5-hour billing window usage, burn rate, and projections.

Examples:
  conductor budget              # Show current block status
  conductor budget status       # Same as above
  conductor budget report       # Show recent blocks (last 3 days)
  conductor budget --json       # JSON output for scripting`,
		Run: runBudgetStatus,
	}

	// Add subcommands
	cmd.AddCommand(newBudgetStatusCmd())
	cmd.AddCommand(newBudgetReportCmd())
	cmd.AddCommand(newBudgetListPausedCmd())
	cmd.AddCommand(newBudgetResumeCmd())

	// Persistent flags
	cmd.PersistentFlags().BoolVar(&budgetJSON, "json", false, "Output in JSON format")
	cmd.PersistentFlags().StringVar(&budgetBaseDir, "base-dir", "", "Override base directory (default: ~/.claude/projects)")

	return cmd
}

func newBudgetStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current 5-hour block usage",
		Run:   runBudgetStatus,
	}
}

func newBudgetReportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "report",
		Short: "Show recent usage blocks (last 3 days)",
		Run:   runBudgetReport,
	}
}

func newBudgetListPausedCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list-paused",
		Short: "List paused executions waiting for rate limit reset",
		Long:  "Shows all executions that were paused due to rate limits and their resume status",
		RunE:  runBudgetListPaused,
	}
}

func newBudgetResumeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "resume [session-id]",
		Short: "Resume paused execution(s)",
		Long:  "Resumes executions that were paused due to rate limits. If no session ID is provided, resumes all executions that are ready.",
		RunE:  runBudgetResume,
	}
}

// runBudgetStatus shows the current 5-hour block status
func runBudgetStatus(cmd *cobra.Command, args []string) {
	tracker, err := createUsageTracker()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	status := tracker.GetStatus()

	if budgetJSON {
		printStatusJSON(status)
	} else {
		printStatusPretty(status)
	}
}

// runBudgetReport shows recent blocks (last 3 days)
func runBudgetReport(cmd *cobra.Command, args []string) {
	tracker, err := createUsageTracker()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	blocks := tracker.GetBlocks()

	// Filter to last 3 days
	cutoff := time.Now().Add(-72 * time.Hour)
	recentBlocks := make([]budget.UsageBlock, 0)
	for _, block := range blocks {
		if block.StartTime.After(cutoff) {
			recentBlocks = append(recentBlocks, block)
		}
	}

	if budgetJSON {
		printReportJSON(recentBlocks)
	} else {
		printReportPretty(recentBlocks)
	}
}

// runBudgetListPaused lists all paused executions
func runBudgetListPaused(cmd *cobra.Command, args []string) error {
	// Create state manager
	stateDir := filepath.Join(".conductor", "state")
	sm := budget.NewStateManager(stateDir)

	// Get all paused states
	states, err := sm.GetPausedStates()
	if err != nil {
		return fmt.Errorf("failed to get paused states: %w", err)
	}

	if len(states) == 0 {
		fmt.Println("No paused executions found.")
		return nil
	}

	fmt.Printf("Found %d paused execution(s):\n\n", len(states))

	for _, state := range states {
		status := string(state.Status)
		if state.Status == budget.StatusReady {
			status = "READY TO RESUME"
		}

		fmt.Printf("  ID: %s\n", state.SessionID)
		fmt.Printf("  Plan: %s\n", state.PlanFile)
		fmt.Printf("  Status: %s\n", status)
		fmt.Printf("  Paused: %s\n", state.PausedAt.Format(time.RFC3339))
		fmt.Printf("  Resume At: %s\n", state.ResumeAt.Format(time.RFC3339))
		if state.RateLimitInfo != nil {
			fmt.Printf("  Limit Type: %s\n", state.RateLimitInfo.LimitType)
		}
		fmt.Println()
	}

	return nil
}

// runBudgetResume resumes paused execution(s)
func runBudgetResume(cmd *cobra.Command, args []string) error {
	stateDir := filepath.Join(".conductor", "state")
	sm := budget.NewStateManager(stateDir)

	if len(args) > 0 {
		// Resume specific session
		sessionID := args[0]
		state, err := sm.Load(sessionID)
		if err != nil {
			return fmt.Errorf("failed to load session %s: %w", sessionID, err)
		}

		if state.Status != budget.StatusReady {
			remaining := time.Until(state.ResumeAt)
			if remaining > 0 {
				return fmt.Errorf("session %s not ready yet - %s remaining", sessionID, remaining.Round(time.Second))
			}
		}

		fmt.Printf("Resuming session %s (plan: %s)...\n", sessionID, state.PlanFile)
		fmt.Printf("Run: conductor run %s --skip-completed\n", state.PlanFile)

		// Delete state file after showing resume command
		if err := sm.Delete(sessionID); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to delete state file: %v\n", err)
		}

		return nil
	}

	// Resume all ready sessions
	states, err := sm.GetReadyStates()
	if err != nil {
		return fmt.Errorf("failed to get ready states: %w", err)
	}

	if len(states) == 0 {
		fmt.Println("No executions ready to resume.")
		return nil
	}

	fmt.Printf("Found %d execution(s) ready to resume:\n\n", len(states))

	for _, state := range states {
		fmt.Printf("  conductor run %s --skip-completed  # Session: %s\n", state.PlanFile, state.SessionID)

		// Delete state file
		if err := sm.Delete(state.SessionID); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to delete state file for %s: %v\n", state.SessionID, err)
		}
	}

	return nil
}

// createUsageTracker creates a UsageTracker with the appropriate base directory
func createUsageTracker() (*budget.UsageTracker, error) {
	// Determine base directory
	baseDir := budgetBaseDir
	if baseDir == "" {
		// Load from config
		cfg, err := config.LoadConfigFromDir(".")
		if err != nil {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}
		baseDir = cfg.AgentWatch.BaseDir
	}

	// Expand home directory
	baseDir = expandPath(baseDir)

	// Create tracker
	tracker := budget.NewUsageTracker(baseDir, budget.DefaultCostModel())
	return tracker, nil
}

// expandPath expands ~ to home directory
func expandPath(path string) string {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[1:])
	}
	return path
}

// printStatusJSON prints status in JSON format
func printStatusJSON(status *budget.BlockStatus) {
	if status == nil {
		fmt.Println(`{"active": false, "message": "No active block"}`)
		return
	}

	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(data))
}

// printStatusPretty prints status in human-readable format
func printStatusPretty(status *budget.BlockStatus) {
	if status == nil {
		fmt.Println("No active usage block found.")
		fmt.Println("Start a Claude Code session to begin tracking usage.")
		return
	}

	block := status.Block
	width := 55

	// Header
	fmt.Println(strings.Repeat("─", width))
	fmt.Printf("%-*s\n", width, centerText("USAGE BUDGET STATUS", width))
	fmt.Println(strings.Repeat("─", width))

	// Session section
	fmt.Println("SESSION")

	// Progress bar
	barWidth := width - 10
	pct := status.PercentElapsed / 100.0
	filled := int(pct * float64(barWidth))
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
	fmt.Printf("%s %.1f%%\n", bar, status.PercentElapsed)

	// Time info
	elapsed := time.Duration(status.ElapsedMinutes) * time.Minute
	remaining := time.Duration(status.RemainingMinutes) * time.Minute
	fmt.Printf("Started: %s  Elapsed: %s  Remaining: %s\n",
		block.StartTime.Format("3:04 PM"),
		formatBudgetDuration(elapsed),
		formatBudgetDuration(remaining))
	fmt.Printf("Ends: %s\n", block.EndTime.Format("3:04 PM"))

	fmt.Println(strings.Repeat("─", width))

	// Usage section
	fmt.Println("USAGE")
	fmt.Printf("Tokens:  %s input / %s output\n",
		formatTokens(block.InputTokens),
		formatTokens(block.OutputTokens))
	fmt.Printf("Cost:    %s\n", formatCost(block.CostUSD))

	// Models
	if len(block.Models) > 0 {
		fmt.Printf("Models:  %s\n", strings.Join(block.Models, ", "))
	}

	fmt.Println(strings.Repeat("─", width))

	// Burn rate section
	if status.BurnRate != nil {
		fmt.Println("BURN RATE")
		fmt.Printf("%.0f tokens/min\n", status.BurnRate.TokensPerMinute)
		fmt.Printf("%s/hour\n", formatCost(status.BurnRate.CostPerHour))
		fmt.Println(strings.Repeat("─", width))
	}

	// Projection section
	if status.Projection != nil {
		fmt.Println("PROJECTION (at current rate)")
		fmt.Printf("End-of-block: ~%s tokens, ~%s\n",
			formatTokens(status.Projection.TotalTokens),
			formatCost(status.Projection.TotalCost))
		fmt.Println(strings.Repeat("─", width))
	}
}

// printReportJSON prints blocks in JSON format
func printReportJSON(blocks []budget.UsageBlock) {
	type reportData struct {
		Count  int                 `json:"count"`
		Blocks []budget.UsageBlock `json:"blocks"`
	}

	data, err := json.MarshalIndent(reportData{
		Count:  len(blocks),
		Blocks: blocks,
	}, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(data))
}

// printReportPretty prints blocks in human-readable format
func printReportPretty(blocks []budget.UsageBlock) {
	if len(blocks) == 0 {
		fmt.Println("No usage blocks found in the last 3 days.")
		return
	}

	fmt.Printf("Recent Usage Blocks (Last 3 Days): %d blocks\n\n", len(blocks))

	for i := len(blocks) - 1; i >= 0; i-- {
		block := blocks[i]

		// Calculate duration
		duration := block.ActualEndTime.Sub(block.StartTime)
		if block.ActualEndTime.IsZero() {
			duration = time.Now().Sub(block.StartTime)
		}

		// Print block summary
		fmt.Printf("Block %s\n", block.StartTime.Format("Jan 02, 3:04 PM"))
		fmt.Printf("  Duration:  %s\n", formatBudgetDuration(duration))
		fmt.Printf("  Tokens:    %s (%s input, %s output)\n",
			formatTokens(block.TotalTokens),
			formatTokens(block.InputTokens),
			formatTokens(block.OutputTokens))
		fmt.Printf("  Cost:      %s\n", formatCost(block.CostUSD))
		fmt.Printf("  Entries:   %d API calls\n", len(block.Entries))
		if len(block.Models) > 0 {
			fmt.Printf("  Models:    %s\n", strings.Join(block.Models, ", "))
		}
		fmt.Println()
	}

	// Summary
	totalTokens := int64(0)
	totalCost := 0.0
	for _, block := range blocks {
		totalTokens += block.TotalTokens
		totalCost += block.CostUSD
	}

	fmt.Println(strings.Repeat("─", 55))
	fmt.Printf("Total: %s tokens, %s across %d blocks\n",
		formatTokens(totalTokens),
		formatCost(totalCost),
		len(blocks))
}

// formatTokens formats token count with commas and abbreviation
func formatTokens(n int64) string {
	if n >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(n)/1000000.0)
	}
	if n >= 1000 {
		// Add commas
		str := fmt.Sprintf("%d", n)
		result := ""
		for i, c := range str {
			if i > 0 && (len(str)-i)%3 == 0 {
				result += ","
			}
			result += string(c)
		}
		return result
	}
	return fmt.Sprintf("%d", n)
}

// formatCost formats cost in USD
func formatCost(cost float64) string {
	return fmt.Sprintf("$%.2f", cost)
}

// formatDuration formats duration in human-readable form
func formatBudgetDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	if minutes == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh %dm", hours, minutes)
}

// centerText centers text within a given width
func centerText(text string, width int) string {
	padding := (width - len(text)) / 2
	if padding < 0 {
		padding = 0
	}
	return strings.Repeat(" ", padding) + text
}
