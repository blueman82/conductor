package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/harrison/conductor/internal/agent"
	"github.com/harrison/conductor/internal/models"
)

func main() {
	registry := agent.NewRegistry("")
	_, err := registry.Discover()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Registry discovery failed: %v\n", err)
		os.Exit(1)
	}

	invoker := agent.NewInvokerWithRegistry(registry)

	task := models.Task{
		Number: "test",
		Name:   "Create test file",
		Agent:  "python-pro",
		Prompt: "Create a file at /tmp/agent_test_output.txt with the content 'Agent is working!' and nothing else. Use the Write tool.",
	}

	os.Remove("/tmp/agent_test_output.txt")

	fmt.Fprintf(os.Stderr, "\n=== HEADLESS AGENT TEST ===\n")
	fmt.Fprintf(os.Stderr, "Task: %s\n", task.Name)
	fmt.Fprintf(os.Stderr, "Agent: %s\n", task.Agent)
	fmt.Fprintf(os.Stderr, "Prompt: %s\n\n", task.Prompt)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	result, err := invoker.Invoke(ctx, task)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "\n=== FULL OUTPUT ===\n")
	fmt.Fprintf(os.Stderr, "%s\n", result.Output)

	fmt.Fprintf(os.Stderr, "\n=== VERIFICATION ===\n")
	if _, err := os.Stat("/tmp/agent_test_output.txt"); err == nil {
		content, _ := os.ReadFile("/tmp/agent_test_output.txt")
		fmt.Fprintf(os.Stderr, "✅ File created: %s\n", string(content))
	} else {
		fmt.Fprintf(os.Stderr, "❌ File NOT created\n")
	}
}
