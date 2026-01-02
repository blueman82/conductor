package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/harrison/conductor/internal/behavioral"
	"github.com/harrison/conductor/internal/config"
	"github.com/spf13/cobra"
)

var observeLiveRaw bool

// NewObserveLiveCmd creates the 'conductor observe live' subcommand
func NewObserveLiveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "live",
		Short: "Watch JSONL files and display events in real-time",
		Long: `Watch JSONL session files in ~/.claude/projects/ for new content
and display events as a live transcript.

Events are displayed as they occur with timestamps and color formatting.
Press Ctrl+C to stop watching.

Examples:
  conductor observe live                    # Watch all projects
  conductor observe live -p myproject       # Watch specific project
  conductor observe live --raw              # Plain text output`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			// Get poll interval: CLI flag > config > default (2s)
			pollInterval := 2 * time.Second

			// Check if CLI flag was explicitly set
			if cmd.Flags().Changed("poll-interval") {
				parsed, err := parsePollInterval(observePollInterval)
				if err != nil {
					return fmt.Errorf("invalid poll-interval: %w", err)
				}
				pollInterval = parsed
			} else {
				// Try to load from config
				cfg, err := config.LoadConfigFromRootWithBuildTime(GetConductorRepoRoot())
				if err == nil && cfg.AgentWatch.PollIntervalSecs > 0 {
					pollInterval = time.Duration(cfg.AgentWatch.PollIntervalSecs) * time.Second
				}
			}

			return RunLiveWatch(ctx, observeProject, pollInterval, observeLiveRaw)
		},
	}

	cmd.Flags().BoolVar(&observeLiveRaw, "raw", false, "Plain text output (no colors/emojis)")

	return cmd
}

// RunLiveWatch starts the live file watcher and displays events as transcript
func RunLiveWatch(ctx context.Context, project string, pollInterval time.Duration, raw bool) error {
	// Disable color if raw mode
	if raw {
		color.NoColor = true
	}

	// Load config for base_dir
	cfg, err := config.LoadConfigFromRootWithBuildTime(GetConductorRepoRoot())
	if err != nil {
		cfg = &config.Config{
			AgentWatch: config.DefaultAgentWatchConfig(),
		}
	}

	// Get Claude projects directory from config
	rootDir := cfg.AgentWatch.BaseDir

	// Create watcher
	watcher := behavioral.NewLiveWatcher(rootDir, project)
	watcher.SetPollInterval(pollInterval)

	// Print header
	cyan := color.New(color.FgCyan)
	if project != "" {
		cyan.Printf("Watching project: %s\n", project)
	} else {
		cyan.Println("Watching all projects")
	}
	cyan.Printf("Poll interval: %s\n", pollInterval)
	cyan.Println("Press Ctrl+C to stop")
	fmt.Println()

	// Configure transcript options
	opts := behavioral.DefaultTranscriptOptions()
	if raw {
		opts.ColorOutput = false
	}

	// Start watcher in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- watcher.Start(ctx)
	}()

	// Process events from channel
	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-errChan:
			if err != nil && err != ctx.Err() {
				return fmt.Errorf("watcher error: %w", err)
			}
			return nil
		case event, ok := <-watcher.Events():
			if !ok {
				return nil
			}
			// Format and print the event
			entry := behavioral.FormatTranscriptEntry(event, opts)
			if entry != "" {
				fmt.Print(entry)
			}
		}
	}
}
