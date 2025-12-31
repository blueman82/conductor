// Package main provides the CLI entry point for the conductor application.
//
// # Dependency Wiring Architecture (v2.32+)
//
// ClaudeSimilarity is a shared service providing Claude-based semantic similarity.
// It is created once during command initialization and injected into both:
//   - PatternIntelligence: for duplicate detection and STOP protocol analysis
//   - WarmUpProvider: for finding similar historical tasks to prime agents
//
// This shared instance ensures:
//   - Consistent configuration across both subsystems
//   - Unified rate limit handling with TTS feedback
//   - Single point of initialization and lifecycle management
//
// The actual wiring is performed in internal/cmd/run.go during runCommand execution.
// The OrchestratorConfig.Similarity field carries the instance through to the orchestrator.
package main

import (
	"fmt"
	"os"

	"github.com/harrison/conductor/internal/cmd"
)

// Version is the current version of the conductor application
const Version = "1.0.0"

func main() {
	rootCmd := cmd.NewRootCommand()

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
