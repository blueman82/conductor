// Package display provides terminal UI utilities for displaying progress, warnings, and status messages.
//
// This package centralizes all terminal output formatting, ANSI color codes, and user-facing display logic
// for the Conductor CLI. It provides three main categories of functionality:
//
// # Progress Indicators
//
// Use ProgressIndicator for multi-step operations:
//
//	progress := display.NewProgressIndicator(os.Stdout, len(files))
//	progress.Start()
//	for _, file := range files {
//	    progress.Step(file)
//	    // ... process file ...
//	}
//	progress.Complete()
//
// For single file operations:
//
//	display.DisplaySingleFile(os.Stdout, filename)
//
// # Warning Messages
//
// Display warnings with optional components:
//
//	warning := display.Warning{
//	    Title:      "Configuration Issue",
//	    Message:    "Setting 'max_parallel' is deprecated",
//	    Files:      []string{"config.yaml"},
//	    Suggestion: "Use 'max_concurrency' instead",
//	}
//	warning.Display(os.Stderr)
//
// Or use the convenience factory for numbered file warnings:
//
//	numberedFiles, _ := display.FindNumberedFiles(dir)
//	if len(numberedFiles) > 0 {
//	    warning := display.WarnNumberedFiles("Numbered Plan Files Detected", numberedFiles)
//	    warning.Display(os.Stdout)
//	}
//
// # File Utilities
//
// Check if a filename matches the numbered file pattern (e.g., "1-setup.md"):
//
//	if display.IsNumberedFile(filename) {
//	    // Handle numbered file
//	}
//
// Scan a directory for numbered files:
//
//	files, err := display.FindNumberedFiles(directory)
//	if err != nil {
//	    // Handle error
//	}
//
// # ANSI Colors
//
// The package uses ANSI escape codes for terminal colors:
//   - Blue (\x1b[34m) for progress indicators
//   - Green (\x1b[32m) for success messages
//   - Yellow (\x1b[33m) for warnings
//   - Reset (\x1b[0m) after each colored section
//
// All functions accept io.Writer interfaces for testability and flexibility.
//
// # Design Principles
//
//   - Single source of truth for all display logic
//   - Consistent formatting across all commands
//   - Testable via io.Writer abstraction
//   - No global state or side effects
//   - Minimal dependencies (standard library only)
package display
