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
