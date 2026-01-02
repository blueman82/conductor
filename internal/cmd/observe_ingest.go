package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/harrison/conductor/internal/behavioral"
	"github.com/harrison/conductor/internal/config"
	"github.com/harrison/conductor/internal/learning"
	"github.com/spf13/cobra"
)

var (
	ingestWatch        bool
	ingestBatchSize    int
	ingestBatchTimeout time.Duration
	ingestRootDir      string
	ingestVerbose      bool
)

// NewObserveIngestCmd creates the 'conductor observe ingest' subcommand
func NewObserveIngestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ingest",
		Short: "Run JSONL ingestion daemon",
		Long: `Run the ingestion daemon to continuously import JSONL files from ~/.claude/projects/.

Without --watch: Imports all existing files and exits (one-time import).
With --watch: Runs as daemon until interrupted, watching for new files and changes.

The daemon:
- Uses batched writes for efficient database operations
- Tracks file offsets for incremental processing
- Handles graceful shutdown on SIGINT/SIGTERM
- Shows statistics on exit

Examples:
  conductor observe ingest                        # One-time import of all files
  conductor observe ingest --watch                # Run as daemon
  conductor observe ingest --batch-size 100       # Larger batches
  conductor observe ingest --batch-timeout 1s     # Flush every second
  conductor observe ingest --verbose              # Show detailed progress`,
		RunE: HandleIngestCommand,
	}

	cmd.Flags().BoolVar(&ingestWatch, "watch", false, "Run as daemon (default: one-time import)")
	cmd.Flags().IntVar(&ingestBatchSize, "batch-size", 50, "Events per batch (default: 50)")
	cmd.Flags().DurationVar(&ingestBatchTimeout, "batch-timeout", 500*time.Millisecond, "Batch flush interval (default: 500ms)")
	cmd.Flags().StringVar(&ingestRootDir, "root-dir", "", "Override Claude projects directory")
	cmd.Flags().BoolVarP(&ingestVerbose, "verbose", "v", false, "Show detailed progress")

	return cmd
}

// HandleIngestCommand processes ingest requests
func HandleIngestCommand(cmd *cobra.Command, args []string) error {
	// Set up context with signal handling for graceful shutdown
	ctx, cancel := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if ctx == nil {
		ctx = context.Background()
	}

	// Load config for base_dir
	configCfg, cfgErr := config.LoadConfigFromRootWithBuildTime(GetConductorRepoRoot())
	if cfgErr != nil {
		configCfg = &config.Config{
			AgentWatch: config.DefaultAgentWatchConfig(),
		}
	}

	// Get database path
	dbPath, err := config.GetLearningDBPath()
	if err != nil {
		return fmt.Errorf("failed to get learning database path: %w", err)
	}

	// Open database
	store, err := learning.NewStore(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open learning store: %w", err)
	}
	defer store.Close()

	// Determine root directory
	rootDir := ingestRootDir
	if rootDir == "" {
		rootDir = configCfg.AgentWatch.BaseDir
	}

	// Verify root directory exists
	if _, err := os.Stat(rootDir); os.IsNotExist(err) {
		return fmt.Errorf("root directory does not exist: %s", rootDir)
	}

	// Configure ingestion engine
	cfg := behavioral.IngestionConfig{
		RootDir:      rootDir,
		Pattern:      "*.jsonl",
		BatchSize:    ingestBatchSize,
		BatchTimeout: ingestBatchTimeout,
	}

	if ingestVerbose {
		fmt.Printf("Ingestion Configuration:\n")
		fmt.Printf("  Root Directory: %s\n", rootDir)
		fmt.Printf("  Pattern: %s\n", cfg.Pattern)
		fmt.Printf("  Batch Size: %d\n", cfg.BatchSize)
		fmt.Printf("  Batch Timeout: %s\n", cfg.BatchTimeout)
		fmt.Printf("  Watch Mode: %v\n", ingestWatch)
		fmt.Println()
	}

	// Create ingestion engine
	engine, err := behavioral.NewIngestionEngine(store, cfg)
	if err != nil {
		return fmt.Errorf("failed to create ingestion engine: %w", err)
	}

	// Start the engine
	if err := engine.Start(ctx); err != nil {
		return fmt.Errorf("failed to start ingestion engine: %w", err)
	}

	if ingestWatch {
		fmt.Printf("Ingestion daemon started, watching %s\n", rootDir)
		fmt.Println("Press Ctrl+C to stop...")
		fmt.Println()

		// Wait for shutdown signal
		<-ctx.Done()
		fmt.Println("\nShutting down...")
	} else {
		// One-time import: let the engine process existing files
		// Wait briefly for initial scan and processing
		fmt.Printf("Processing files in %s...\n", rootDir)

		// Give the engine time to discover and process files
		// In one-time mode, we poll until no more events are pending
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		timeout := time.After(30 * time.Second)
		stableCount := 0
		lastProcessed := int64(0)

		for {
			select {
			case <-ctx.Done():
				goto shutdown
			case <-timeout:
				if ingestVerbose {
					fmt.Println("Timeout reached, finishing...")
				}
				goto shutdown
			case <-ticker.C:
				stats := engine.Stats()
				if stats.EventsProcessed == lastProcessed && stats.EventsPending == 0 {
					stableCount++
					if stableCount >= 5 {
						// Stable for 500ms, done processing
						goto shutdown
					}
				} else {
					stableCount = 0
					lastProcessed = stats.EventsProcessed
				}
				if ingestVerbose && stats.EventsProcessed > 0 {
					fmt.Printf("  Processed: %d events\n", stats.EventsProcessed)
				}
			}
		}
	}

shutdown:
	// Stop the engine (flushes pending events)
	if err := engine.Stop(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: error stopping engine: %v\n", err)
	}

	// Display final statistics
	stats := engine.Stats()
	fmt.Println()
	fmt.Println("==================================================")
	fmt.Println("Ingestion Summary")
	fmt.Println("==================================================")
	fmt.Printf("Files Tracked:     %d\n", stats.FilesTracked)
	fmt.Printf("Events Processed:  %d\n", stats.EventsProcessed)
	fmt.Printf("Sessions Created:  %d\n", stats.SessionsCreated)
	fmt.Printf("Errors:            %d\n", stats.Errors)
	fmt.Printf("Uptime:            %s\n", stats.Uptime.Round(time.Millisecond))

	if stats.Errors > 0 {
		return fmt.Errorf("completed with %d errors", stats.Errors)
	}

	return nil
}
