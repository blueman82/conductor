# Implementation Plan: Enhanced Console Output System

**Created**: 2025-01-13
**Target**: Add rich terminal output with progress tracking, detailed task information, inline QC feedback, and comprehensive execution summaries
**Estimated Tasks**: 18

## Context for the Engineer

You are implementing this feature in a codebase that:
- Uses **Go 1.25.4** with clean architecture patterns
- Follows strict **TDD methodology** (red-green-refactor cycle)
- Tests with `github.com/stretchr/testify` (86.4% coverage target)
- Uses `github.com/fatih/color` for ANSI terminal colors
- Logs to console via `internal/logger/console.go` (thread-safe with mutex)
- Tracks execution via `internal/models/result.go` structs
- Configures via `.conductor/config.yaml` with YAML unmarshaling

**You are expected to**:
- Write tests BEFORE implementation (TDD - mandatory)
- Commit frequently (after each completed task)
- Follow existing code patterns (see references in each task)
- Keep changes minimal (YAGNI - You Aren't Gonna Need It)
- Avoid duplication (DRY - Don't Repeat Yourself)
- Maintain 90%+ test coverage for critical paths

## Worktree Groups

This plan identifies task groupings for parallel execution using git worktrees. Each group operates in isolation with its own branch.

**Group chain-1**: Configuration & Models Foundation
- **Tasks**: Task 1, Task 2, Task 3 (executed sequentially in dependency order)
- **Branch**: `feature/enhanced-console-output/chain-1`
- **Execution**: Sequential (dependencies enforced)
- **Isolation**: Separate worktree from other groups
- **Rationale**: Configuration must exist before models can reference it. Models must be updated before logger can use new fields.

**Group chain-2**: Progress Bar Component
- **Tasks**: Task 4, Task 5, Task 6 (executed sequentially in dependency order)
- **Branch**: `feature/enhanced-console-output/chain-2`
- **Execution**: Sequential (TDD cycle: test → implement → integrate)
- **Isolation**: Separate worktree from chain-1
- **Rationale**: Progress bar tests drive implementation, which must complete before logger integration.

**Group chain-3**: ConsoleLogger Enhancement
- **Tasks**: Task 7, Task 8, Task 9, Task 10, Task 11, Task 12 (executed sequentially)
- **Branch**: `feature/enhanced-console-output/chain-3`
- **Execution**: Sequential (each method builds on previous)
- **Isolation**: Separate worktree from chain-2
- **Rationale**: LogTaskStart → LogTaskResult → LogProgress → LogWaveComplete build incrementally

**Group chain-4**: Orchestrator Integration
- **Tasks**: Task 13, Task 14, Task 15 (executed sequentially)
- **Branch**: `feature/enhanced-console-output/chain-4`
- **Execution**: Sequential (depends on chain-3 completion)
- **Isolation**: Separate worktree from chain-3
- **Rationale**: Orchestrator must integrate all logger enhancements in correct call order

**Group independent-1**: Integration Testing
- **Tasks**: Task 16 only
- **Branch**: `feature/enhanced-console-output/independent-1`
- **Execution**: Parallel-safe (no dependencies on other groups)
- **Isolation**: Separate worktree from all chains
- **Rationale**: End-to-end tests can run in parallel with documentation updates

**Group independent-2**: Configuration Documentation
- **Tasks**: Task 17 only
- **Branch**: `feature/enhanced-console-output/independent-2`
- **Execution**: Parallel-safe (no code dependencies)
- **Isolation**: Separate worktree from all groups
- **Rationale**: config.yaml.example updates are independent of code changes

**Group independent-3**: User Documentation
- **Tasks**: Task 18 only
- **Branch**: `feature/enhanced-console-output/independent-3`
- **Execution**: Parallel-safe (no code dependencies)
- **Isolation**: Separate worktree from all groups
- **Rationale**: README updates can happen independently

### Worktree Management

**Creating worktrees**:
```bash
# For each group, create a worktree
git worktree add ../conductor-chain-1 -b feature/enhanced-console-output/chain-1
git worktree add ../conductor-chain-2 -b feature/enhanced-console-output/chain-2
git worktree add ../conductor-chain-3 -b feature/enhanced-console-output/chain-3
git worktree add ../conductor-chain-4 -b feature/enhanced-console-output/chain-4
git worktree add ../conductor-independent-1 -b feature/enhanced-console-output/independent-1
git worktree add ../conductor-independent-2 -b feature/enhanced-console-output/independent-2
git worktree add ../conductor-independent-3 -b feature/enhanced-console-output/independent-3
```

**Switching between worktrees**:
```bash
cd ../conductor-chain-1    # Work on config & models
cd ../conductor-chain-2    # Work on progress bar
# etc.
```

**Cleaning up after merge**:
```bash
git worktree remove ../conductor-chain-1
git branch -d feature/enhanced-console-output/chain-1
# Repeat for all worktrees
```

## Prerequisites Checklist

Before starting implementation, verify:

- [ ] **Go 1.25.4+** installed (`go version`)
- [ ] **Development environment** set up (can run `go test ./...` successfully)
- [ ] **Current working branch** is main/master
- [ ] **All existing tests pass** (`go test ./...` shows no failures)
- [ ] **Color library available** (`github.com/fatih/color` in go.mod)
- [ ] **Testify available** (`github.com/stretchr/testify` in go.mod)
- [ ] **Read CLAUDE.md** for TDD workflow and project conventions
- [ ] **Worktrees created** (7 worktrees as shown above, optional for serial execution)

## Task 1: Add Console Configuration to Config Struct

**Agent**: golang-pro
**File(s)**: `internal/config/config.go`, `internal/config/config_test.go`
**Depends on**: None
**WorktreeGroup**: chain-1
**Estimated time**: 15m

### What you're building
Add a new `Console` configuration section to the existing Config struct to control terminal output behavior (colors, verbose mode, progress bars, QC feedback display). This provides a unified configuration source replacing environment variable checks.

### Test First (TDD)

**Test file**: `internal/config/config_test.go`

**Test structure**:
```go
func TestDefaultConfig_ConsoleDefaults(t *testing.T)
  - test that Console.ColorsEnabled defaults to "auto"
  - test that Console.Verbose defaults to false
  - test that Console.ProgressBar defaults to true
  - test that Console.ShowQCFeedback defaults to true

func TestLoadConfig_ConsoleSection(t *testing.T)
  - create temp config.yaml with console section
  - load config and verify console fields parsed correctly
  - test colors_enabled: "true", verbose: true, etc.

func TestLoadConfig_ConsoleColorsValidation(t *testing.T)
  - test valid values: "auto", "true", "false"
  - test invalid value defaults to "auto"
```

**Test specifics**:
- Mock these dependencies: File I/O (use temp files)
- Use these fixtures/factories: Create temp YAML config files in test
- Assert these outcomes: Console struct fields match expected values
- Edge cases to cover: Missing console section, invalid colors_enabled value

**Example test skeleton**:
```go
func TestDefaultConfig_ConsoleDefaults(t *testing.T) {
	cfg := DefaultConfig()
	
	assert.Equal(t, "auto", cfg.Console.ColorsEnabled)
	assert.False(t, cfg.Console.Verbose)
	assert.True(t, cfg.Console.ProgressBar)
	assert.True(t, cfg.Console.ShowQCFeedback)
}

func TestLoadConfig_ConsoleSection(t *testing.T) {
	// Create temp config with console section
	yamlContent := `
console:
  colors_enabled: "false"
  verbose: true
  progress_bar: false
  show_qc_feedback: true
`
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	
	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	tmpFile.Close()
	
	cfg, err := LoadConfig(tmpFile.Name())
	require.NoError(t, err)
	
	assert.Equal(t, "false", cfg.Console.ColorsEnabled)
	assert.True(t, cfg.Console.Verbose)
	assert.False(t, cfg.Console.ProgressBar)
	assert.True(t, cfg.Console.ShowQCFeedback)
}
```

### Implementation

**Approach**:
Add a `ConsoleConfig` struct and embed it in the existing `Config` struct. Update `DefaultConfig()` to populate console defaults. The YAML unmarshaling will automatically handle the `console:` section.

**Code structure**:
```go
// Add after LearningConfig definition (around line 35)
type ConsoleConfig struct {
	// ColorsEnabled controls color output: "auto", "true", "false"
	ColorsEnabled string `yaml:"colors_enabled"`
	
	// Verbose enables detailed multi-line task output
	Verbose bool `yaml:"verbose"`
	
	// ProgressBar shows ASCII progress bar after each task
	ProgressBar bool `yaml:"progress_bar"`
	
	// ShowQCFeedback displays QC feedback inline in console
	ShowQCFeedback bool `yaml:"show_qc_feedback"`
}

// Add Console field to Config struct (around line 61, after Learning field)
type Config struct {
	// ... existing fields ...
	Learning LearningConfig `yaml:"learning"`
	
	// Console controls terminal output behavior
	Console ConsoleConfig `yaml:"console"`
}
```

**Key points**:
- Follow pattern from: `internal/config/config.go:13-34` (LearningConfig struct)
- Place ConsoleConfig definition right after LearningConfig (line 35)
- Add Console field after Learning field in Config struct (line 61)
- Update DefaultConfig() function (line 73-82) to include Console defaults
- No validation logic needed yet - just data structures

**Integration points**:
- Imports needed: None (yaml.v3 already imported)
- Services to inject: None
- Config values: 
  - `colors_enabled` defaults to "auto"
  - `verbose` defaults to false
  - `progress_bar` defaults to true
  - `show_qc_feedback` defaults to true

### Verification

**Manual testing**:
1. Run `go build ./cmd/conductor` - should compile without errors
2. Create test config file with console section, verify it loads
3. Run with missing console section, verify defaults apply

**Automated tests**:
```bash
go test ./internal/config/ -v -run TestDefaultConfig_Console
go test ./internal/config/ -v -run TestLoadConfig_Console
```

**Expected output**:
All console config tests pass with default values correctly set.

### Commit

**Commit message**:
```
feat(config): add console output configuration

Add ConsoleConfig struct with fields for colors_enabled,
verbose, progress_bar, and show_qc_feedback. Integrate
into Config struct with sensible defaults.

Supports unified config control replacing env var checks.
```

**Files to commit**:
- `internal/config/config.go`
- `internal/config/config_test.go`

---

## Task 2: Extend ExecutionResult with Enhanced Statistics

**Agent**: golang-pro
**File(s)**: `internal/models/result.go`, `internal/models/models_test.go`
**Depends on**: Task 1
**WorktreeGroup**: chain-1
**Estimated time**: 15m

### What you're building
Add new fields to ExecutionResult struct to track status breakdown (GREEN/YELLOW/RED counts), agent usage statistics, total files modified, and average task duration. These enable the enhanced execution summary display.

### Test First (TDD)

**Test file**: `internal/models/models_test.go`

**Test structure**:
```go
func TestExecutionResult_StatusBreakdownTracking(t *testing.T)
  - create ExecutionResult with StatusBreakdown map
  - populate with status counts
  - verify map contains expected keys and values

func TestExecutionResult_AgentUsageTracking(t *testing.T)
  - create ExecutionResult with AgentUsage map
  - populate with agent names and counts
  - verify map accumulates agent usage correctly

func TestExecutionResult_FileCountTracking(t *testing.T)
  - test TotalFiles field stores file count
  - verify zero value and positive values

func TestExecutionResult_AvgDurationCalculation(t *testing.T)
  - test AvgTaskDuration field stores duration
  - verify time.Duration type handling
```

**Test specifics**:
- Mock these dependencies: None (pure data structure tests)
- Use these fixtures/factories: Create ExecutionResult with populated fields
- Assert these outcomes: New fields exist and store correct types
- Edge cases to cover: Empty maps, zero durations, nil handling

**Example test skeleton**:
```go
func TestExecutionResult_EnhancedFields(t *testing.T) {
	result := models.ExecutionResult{
		TotalTasks:  10,
		Completed:   8,
		Failed:      2,
		Duration:    5 * time.Minute,
		FailedTasks: []models.TaskResult{},
		
		// New fields
		StatusBreakdown: map[string]int{
			models.StatusGreen:  6,
			models.StatusYellow: 2,
			models.StatusRed:    1,
		},
		AgentUsage: map[string]int{
			"golang-pro":     5,
			"test-automator": 3,
		},
		TotalFiles:      25,
		AvgTaskDuration: 30 * time.Second,
	}
	
	assert.Equal(t, 6, result.StatusBreakdown[models.StatusGreen])
	assert.Equal(t, 5, result.AgentUsage["golang-pro"])
	assert.Equal(t, 25, result.TotalFiles)
	assert.Equal(t, 30*time.Second, result.AvgTaskDuration)
}
```

### Implementation

**Approach**:
Add four new fields to the existing ExecutionResult struct. Use map[string]int for status and agent tracking to enable flexible accumulation during execution.

**Code structure**:
```go
// Update ExecutionResult struct in internal/models/result.go (around line 24-31)
type ExecutionResult struct {
	TotalTasks  int           // Total number of tasks
	Completed   int           // Number of completed tasks
	Failed      int           // Number of failed tasks
	Duration    time.Duration // Total execution time
	FailedTasks []TaskResult  // Details of failed tasks
	
	// NEW: Enhanced statistics for summary display
	StatusBreakdown map[string]int // Count per status (GREEN: 6, YELLOW: 1, etc.)
	AgentUsage      map[string]int // Count per agent (golang-pro: 5, etc.)
	TotalFiles      int            // Total files modified across all tasks
	AvgTaskDuration time.Duration  // Average duration per task
}
```

**Key points**:
- Follow pattern from: Existing ExecutionResult struct fields (line 24-31)
- Add fields after FailedTasks field
- Use map[string]int for flexible key-value tracking
- Add comments explaining each new field's purpose
- No initialization logic needed yet (orchestrator will populate)

**Integration points**:
- Imports needed: None (time already imported)
- Services to inject: None
- Config values: None

### Verification

**Manual testing**:
1. Run `go build ./internal/models/` - should compile
2. Create ExecutionResult in Go playground, verify new fields accessible

**Automated tests**:
```bash
go test ./internal/models/ -v -run TestExecutionResult
```

**Expected output**:
Tests pass showing new fields can be set and retrieved.

### Commit

**Commit message**:
```
feat(models): add enhanced execution statistics fields

Extend ExecutionResult with StatusBreakdown, AgentUsage,
TotalFiles, and AvgTaskDuration for comprehensive summary
display. Maps enable flexible accumulation during execution.
```

**Files to commit**:
- `internal/models/result.go`
- `internal/models/models_test.go`

---

## Task 3: Create Progress Bar Component with Tests

**Agent**: golang-pro
**File(s)**: `internal/logger/progress_bar.go`, `internal/logger/progress_bar_test.go`
**Depends on**: Task 1, Task 2
**WorktreeGroup**: chain-1
**Estimated time**: 30m

### What you're building
Create a reusable ProgressBar component that renders ASCII progress bars with Unicode block characters (█ for filled, ░ for empty), supports color output, calculates percentages, and displays task counts. This component will be used by ConsoleLogger.

### Test First (TDD)

**Test file**: `internal/logger/progress_bar_test.go`

**Test structure**:
```go
func TestProgressBar_RenderBasic(t *testing.T)
  - test 0% progress (all empty blocks)
  - test 50% progress (half filled, half empty)
  - test 100% progress (all filled blocks)
  - verify format: [blocks] percentage (current/total)

func TestProgressBar_RenderWithColors(t *testing.T)
  - test colorOutput=true applies ANSI codes
  - verify hi-green for filled blocks
  - verify hi-black/gray for empty blocks
  - verify hi-cyan for percentage

func TestProgressBar_RenderWithoutColors(t *testing.T)
  - test colorOutput=false produces plain text
  - verify no ANSI escape codes in output

func TestProgressBar_PercentageCalculation(t *testing.T)
  - test edge cases: 0/10 = 0%, 5/10 = 50%, 10/10 = 100%
  - test rounding: 1/3 = 33%, 2/3 = 67%

func TestProgressBar_BarWidthScaling(t *testing.T)
  - test different bar widths (10, 20, 30 chars)
  - verify filled blocks scale correctly
```

**Test specifics**:
- Mock these dependencies: None (pure rendering logic)
- Use these fixtures/factories: None
- Assert these outcomes: 
  - Correct number of filled/empty Unicode blocks
  - ANSI color codes present when colorOutput=true
  - Percentage matches calculation
- Edge cases to cover: 0 total, 0 completed, equal completed and total

**Example test skeleton**:
```go
func TestProgressBar_RenderBasic(t *testing.T) {
	tests := []struct{
		name      string
		completed int
		total     int
		expected  string
	}{
		{"empty", 0, 10, "[░░░░░░░░░░] 0% (0/10)"},
		{"half", 5, 10, "[█████░░░░░] 50% (5/10)"},
		{"full", 10, 10, "[██████████] 100% (10/10)"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pb := NewProgressBar(tt.total, 10, false)
			result := pb.Render(tt.completed)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProgressBar_RenderWithColors(t *testing.T) {
	pb := NewProgressBar(10, 10, true)
	result := pb.Render(5)
	
	// Verify ANSI color codes present
	assert.Contains(t, result, "\x1b[92m") // hi-green
	assert.Contains(t, result, "\x1b[90m") // hi-black
	assert.Contains(t, result, "\x1b[96m") // hi-cyan
}
```

### Implementation

**Approach**:
Create a ProgressBar struct with total, width, and colorOutput fields. Implement Render() method that calculates filled/empty block counts, applies colors if enabled, and formats the complete progress string.

**Code structure**:
```go
// internal/logger/progress_bar.go
package logger

import (
	"fmt"
	"strings"
	
	"github.com/fatih/color"
)

// ProgressBar renders ASCII progress bars with Unicode block characters
type ProgressBar struct {
	total       int  // Total number of items
	width       int  // Width of bar in characters
	colorOutput bool // Whether to apply ANSI colors
}

// NewProgressBar creates a progress bar renderer
// width is the number of block characters in the bar (default: 20)
func NewProgressBar(total int, width int, colorOutput bool) *ProgressBar {
	if width <= 0 {
		width = 20
	}
	return &ProgressBar{
		total:       total,
		width:       width,
		colorOutput: colorOutput,
	}
}

// Render produces a formatted progress bar string
// Format: [████████░░] 80% (8/10)
func (pb *ProgressBar) Render(completed int) string {
	if pb.total == 0 {
		return "[          ] 0% (0/0)"
	}
	
	// Calculate percentage
	percentage := (float64(completed) / float64(pb.total)) * 100
	
	// Calculate filled blocks
	filledCount := int((float64(completed) / float64(pb.total)) * float64(pb.width))
	emptyCount := pb.width - filledCount
	
	// Build bar string with Unicode blocks
	filled := strings.Repeat("█", filledCount)
	empty := strings.Repeat("░", emptyCount)
	
	// Apply colors if enabled
	if pb.colorOutput {
		filled = color.New(color.FgHiGreen).Sprint(filled)
		empty = color.New(color.FgHiBlack).Sprint(empty)
		percentStr := color.New(color.FgHiCyan).Sprintf("%.0f%%", percentage)
		return fmt.Sprintf("[%s%s] %s (%d/%d)", filled, empty, percentStr, completed, pb.total)
	}
	
	// Plain text output
	return fmt.Sprintf("[%s%s] %.0f%% (%d/%d)", filled, empty, percentage, completed, pb.total)
}
```

**Key points**:
- Follow pattern from: `internal/logger/console.go` for color application
- Use Unicode blocks: U+2588 (█ full block), U+2591 (░ light shade)
- Color codes: `color.FgHiGreen` (92), `color.FgHiBlack` (90), `color.FgHiCyan` (96)
- Handle edge case: total=0 should not panic
- Width parameter allows customization (default 20 blocks)

**Integration points**:
- Imports needed: `github.com/fatih/color` (already in project)
- Services to inject: None (pure rendering)
- Config values: None (colorOutput passed from logger)

### Verification

**Manual testing**:
1. Create test program that renders progress bars at various percentages
2. Run in terminal to verify Unicode blocks display correctly
3. Test with NO_COLOR=1 to verify plain text output

**Automated tests**:
```bash
go test ./internal/logger/ -v -run TestProgressBar
```

**Expected output**:
All progress bar tests pass with correct block counts and color codes.

### Commit

**Commit message**:
```
feat(logger): add progress bar rendering component

Create ProgressBar struct with Render() method for ASCII
progress bars. Uses Unicode blocks (█/░) with color support
(hi-green/hi-black/hi-cyan). Handles edge cases and custom widths.
```

**Files to commit**:
- `internal/logger/progress_bar.go`
- `internal/logger/progress_bar_test.go`


## Task 4: Add LogTaskStart Method to ConsoleLogger

**Agent**: golang-pro
**File(s)**: `internal/logger/console.go`, `internal/logger/console_test.go`
**Depends on**: Task 3
**WorktreeGroup**: chain-2
**Estimated time**: 20m

### What you're building
Add a new LogTaskStart() method to ConsoleLogger that displays an in-progress indicator when a task begins execution. This provides real-time visibility into currently executing tasks with progress counter, agent name, and status indicator.

### Test First (TDD)

**Test file**: `internal/logger/console_test.go`

**Test structure**:
```go
func TestLogTaskStart_Format(t *testing.T)
  - test basic format: [HH:MM:SS] [N/M] ⏳ Task N (name): IN PROGRESS
  - verify timestamp present
  - verify progress counter [N/M]
  - verify hourglass emoji ⏳
  - verify "IN PROGRESS" text

func TestLogTaskStart_WithAgent(t *testing.T)
  - test agent name appears in output
  - format: (agent: agent-name)
  - verify agent colored in magenta when colors enabled

func TestLogTaskStart_ColorOutput(t *testing.T)
  - test colorOutput=true applies cyan to "IN PROGRESS"
  - test colorOutput=true applies cyan to progress counter
  - test colorOutput=true applies magenta to agent name

func TestLogTaskStart_NoColors(t *testing.T)
  - test colorOutput=false produces plain text
  - verify no ANSI escape codes

func TestLogTaskStart_ThreadSafety(t *testing.T)
  - launch multiple goroutines calling LogTaskStart
  - verify no race conditions (mutex protection)
  - verify all messages written successfully
```

**Test specifics**:
- Mock these dependencies: None (use bytes.Buffer for output)
- Use these fixtures/factories: Create sample Task structs
- Assert these outcomes: Correct format, colors applied, thread-safe
- Edge cases to cover: Empty agent name, task number 0, total 0

**Example test skeleton**:
```go
func TestLogTaskStart_Format(t *testing.T) {
	var buf bytes.Buffer
	logger := NewConsoleLogger(&buf, "info")
	
	task := models.Task{
		Number: "5",
		Name:   "Setup Database",
		Agent:  "database-architect",
	}
	
	logger.LogTaskStart(task, 5, 10)
	
	output := buf.String()
	assert.Contains(t, output, "[5/10]")
	assert.Contains(t, output, "⏳")
	assert.Contains(t, output, "Task 5 (Setup Database)")
	assert.Contains(t, output, "IN PROGRESS")
	assert.Contains(t, output, "agent: database-architect")
}

func TestLogTaskStart_ColorOutput(t *testing.T) {
	var buf bytes.Buffer
	logger := &ConsoleLogger{
		writer:      &buf,
		logLevel:    "info",
		colorOutput: true,
	}
	
	task := models.Task{
		Number: "1",
		Name:   "Test",
		Agent:  "golang-pro",
	}
	
	logger.LogTaskStart(task, 1, 5)
	
	output := buf.String()
	// Verify cyan color code for IN PROGRESS
	assert.Contains(t, output, "\x1b[36m")
	// Verify magenta color code for agent
	assert.Contains(t, output, "\x1b[35m")
}
```

### Implementation

**Approach**:
Add LogTaskStart() method following the same pattern as LogTaskResult(). Use mutex for thread safety, format timestamp, apply colors conditionally, and include progress counter and agent information.

**Code structure**:
```go
// Add to internal/logger/console.go after LogTaskResult method (around line 307)

// LogTaskStart logs when a task begins execution at INFO level.
// Shows in-progress indicator with progress counter and agent.
// Format: "[HH:MM:SS] [N/M] ⏳ Task <number> (<name>): IN PROGRESS (agent: <agent>)"
func (cl *ConsoleLogger) LogTaskStart(task models.Task, current int, total int) error {
	if cl.writer == nil {
		return nil
	}
	
	// Task start logging is at INFO level
	if !cl.shouldLog("info") {
		return nil
	}
	
	cl.mutex.Lock()
	defer cl.mutex.Unlock()
	
	ts := timestamp()
	taskInfo := fmt.Sprintf("Task %s (%s)", task.Number, task.Name)
	progressCounter := fmt.Sprintf("[%d/%d]", current, total)
	
	var message string
	if cl.colorOutput {
		// Apply colors: cyan for progress/status, magenta for agent
		counterColored := color.New(color.FgCyan).Sprint(progressCounter)
		statusColored := color.New(color.FgCyan).Sprint("IN PROGRESS")
		agentColored := ""
		if task.Agent != "" {
			agentColored = fmt.Sprintf(" (agent: %s)", 
				color.New(color.FgMagenta).Sprint(task.Agent))
		}
		message = fmt.Sprintf("[%s] %s ⏳ %s: %s%s\n", 
			ts, counterColored, taskInfo, statusColored, agentColored)
	} else {
		agentInfo := ""
		if task.Agent != "" {
			agentInfo = fmt.Sprintf(" (agent: %s)", task.Agent)
		}
		message = fmt.Sprintf("[%s] %s ⏳ %s: IN PROGRESS%s\n", 
			ts, progressCounter, taskInfo, agentInfo)
	}
	
	_, err := cl.writer.Write([]byte(message))
	return err
}
```

**Key points**:
- Follow pattern from: `internal/logger/console.go:268-307` (LogTaskResult)
- Use hourglass emoji (⏳) for visual in-progress indicator
- Apply cyan (`color.FgCyan`) to progress counter and status
- Apply magenta (`color.FgMagenta`) to agent name
- Check colorOutput flag before applying colors
- Protect with mutex.Lock() for thread safety

**Integration points**:
- Imports needed: None (color already imported)
- Services to inject: None
- Config values: None (uses existing colorOutput field)

### Verification

**Manual testing**:
1. Create test that calls LogTaskStart and prints to console
2. Verify hourglass emoji renders correctly in terminal
3. Test with NO_COLOR=1 to verify plain text

**Automated tests**:
```bash
go test ./internal/logger/ -v -run TestLogTaskStart
go test ./internal/logger/ -race -run TestLogTaskStart  # Check thread safety
```

**Expected output**:
All LogTaskStart tests pass, including color and thread-safety tests.

### Commit

**Commit message**:
```
feat(logger): add LogTaskStart for in-progress indicators

Implement LogTaskStart() method that displays real-time task
execution with progress counter [N/M], hourglass emoji, and
agent name. Applies cyan/magenta colors and thread-safe.
```

**Files to commit**:
- `internal/logger/console.go`
- `internal/logger/console_test.go`

---

## Task 5: Enhance LogTaskResult with Progress Counter and Details

**Agent**: golang-pro
**File(s)**: `internal/logger/console.go`, `internal/logger/console_test.go`
**Depends on**: Task 4
**WorktreeGroup**: chain-2
**Estimated time**: 30m

### What you're building
Modify the existing LogTaskResult() method to add progress counter [N/M], duration, agent name, and file count in default mode. Also add support for verbose mode with multi-line detailed output including full file list and QC feedback.

### Test First (TDD)

**Test file**: `internal/logger/console_test.go`

**Test structure**:
```go
func TestLogTaskResult_DefaultModeWithDetails(t *testing.T)
  - test format includes [N/M] progress counter
  - test duration displayed (e.g., "2.3s")
  - test agent name displayed
  - test file count displayed (e.g., "3 files")
  - verify single-line format

func TestLogTaskResult_VerboseMode(t *testing.T)
  - set verbose mode enabled
  - verify multi-line output with indentation
  - verify "Duration:", "Agent:", "Files:", "QC Feedback:" labels
  - verify file list expanded (comma-separated)

func TestLogTaskResult_QCFeedbackDisplay(t *testing.T)
  - test GREEN status shows positive feedback
  - test YELLOW status shows warning with ⚠️
  - test RED status shows error with ❌
  - verify feedback text included

func TestLogTaskResult_ColorCoding(t *testing.T)
  - test duration in gray (hi-black)
  - test agent in magenta
  - test progress counter in cyan
  - test status colors unchanged (green/yellow/red)

func TestLogTaskResult_BackwardCompatibility(t *testing.T)
  - test with nil/empty optional fields
  - verify doesn't crash with missing data
  - verify graceful degradation
```

**Test specifics**:
- Mock these dependencies: None (use bytes.Buffer)
- Use these fixtures/factories: Create TaskResult with all fields populated
- Assert these outcomes: Correct inline format, verbose multi-line format, colors
- Edge cases to cover: Zero duration, no agent, empty files list, no QC feedback

**Example test skeleton**:
```go
func TestLogTaskResult_DefaultModeWithDetails(t *testing.T) {
	var buf bytes.Buffer
	logger := NewConsoleLogger(&buf, "debug")
	
	result := models.TaskResult{
		Task: models.Task{
			Number: "3",
			Name:   "Setup DB",
			Agent:  "database-architect",
			Files:  []string{"db.go", "schema.sql", "migrations.go"},
		},
		Status:   models.StatusGreen,
		Duration: 2300 * time.Millisecond,
	}
	
	logger.LogTaskResult(result, 3, 10)
	
	output := buf.String()
	assert.Contains(t, output, "[3/10]")
	assert.Contains(t, output, "2.3s")
	assert.Contains(t, output, "agent: database-architect")
	assert.Contains(t, output, "3 files")
	assert.Contains(t, output, "GREEN")
}

func TestLogTaskResult_VerboseMode(t *testing.T) {
	var buf bytes.Buffer
	logger := &ConsoleLogger{
		writer:      &buf,
		logLevel:    "debug",
		colorOutput: false,
		verbose:     true,  // NEW field needed
	}
	
	result := models.TaskResult{
		Task: models.Task{
			Number: "1",
			Name:   "Test",
			Agent:  "golang-pro",
			Files:  []string{"test.go"},
		},
		Status:         models.StatusGreen,
		Duration:       5 * time.Second,
		ReviewFeedback: "All tests pass",
	}
	
	logger.LogTaskResult(result, 1, 5)
	
	output := buf.String()
	// Verify multi-line format with indentation
	assert.Contains(t, output, "Duration: 5s")
	assert.Contains(t, output, "Agent: golang-pro")
	assert.Contains(t, output, "Files: test.go")
	assert.Contains(t, output, "QC Feedback: All tests pass")
}
```

### Implementation

**Approach**:
Modify existing LogTaskResult() to check for verbose mode. In default mode, add inline details (duration, agent, file count). In verbose mode, output multi-line format with full details. Add verbose field to ConsoleLogger struct.

**Code structure**:
```go
// 1. Add verbose field to ConsoleLogger struct (around line 36)
type ConsoleLogger struct {
	writer      io.Writer
	logLevel    string
	mutex       sync.Mutex
	colorOutput bool
	verbose     bool  // NEW: Enable detailed multi-line output
}

// 2. Update NewConsoleLogger to accept verbose parameter (around line 46)
func NewConsoleLogger(writer io.Writer, logLevel string, verbose bool) *ConsoleLogger {
	normalizedLevel := normalizeLogLevel(logLevel)
	useColor := isTerminal(writer)
	
	return &ConsoleLogger{
		writer:      writer,
		logLevel:    normalizedLevel,
		mutex:       sync.Mutex{},
		colorOutput: useColor,
		verbose:     verbose,  // NEW
	}
}

// 3. Modify LogTaskResult method (replace existing implementation around line 268-307)
func (cl *ConsoleLogger) LogTaskResult(result models.TaskResult, current int, total int) error {
	if cl.writer == nil {
		return nil
	}
	
	if !cl.shouldLog("debug") {
		return nil
	}
	
	cl.mutex.Lock()
	defer cl.mutex.Unlock()
	
	ts := timestamp()
	taskInfo := fmt.Sprintf("Task %s (%s)", result.Task.Number, result.Task.Name)
	progressCounter := fmt.Sprintf("[%d/%d]", current, total)
	
	// Format duration
	durationStr := formatDuration(result.Duration)
	
	// Count files
	fileCount := len(result.Task.Files)
	fileCountStr := fmt.Sprintf("%d files", fileCount)
	if fileCount == 1 {
		fileCountStr = "1 file"
	}
	
	// Determine status color
	var statusText string
	if cl.colorOutput {
		switch result.Status {
		case models.StatusGreen:
			statusText = color.New(color.FgHiGreen).Sprint("GREEN")
		case models.StatusRed:
			statusText = color.New(color.FgHiRed).Sprint("RED")
		case models.StatusYellow:
			statusText = color.New(color.FgHiYellow).Sprint("YELLOW")
		case models.StatusFailed:
			statusText = color.New(color.FgHiRed).Sprint("FAILED")
		default:
			statusText = string(result.Status)
		}
	} else {
		statusText = string(result.Status)
	}
	
	// Check if verbose mode
	if cl.verbose {
		// Multi-line detailed format
		var message strings.Builder
		
		// First line: basic info with status
		if cl.colorOutput {
			counterColored := color.New(color.FgCyan).Sprint(progressCounter)
			message.WriteString(fmt.Sprintf("[%s] %s %s: %s\n", ts, counterColored, taskInfo, statusText))
		} else {
			message.WriteString(fmt.Sprintf("[%s] %s %s: %s\n", ts, progressCounter, taskInfo, statusText))
		}
		
		// Indented details
		if cl.colorOutput {
			durationColored := color.New(color.FgHiBlack).Sprint(durationStr)
			agentColored := color.New(color.FgMagenta).Sprint(result.Task.Agent)
			message.WriteString(fmt.Sprintf("           Duration: %s\n", durationColored))
			if result.Task.Agent != "" {
				message.WriteString(fmt.Sprintf("           Agent: %s\n", agentColored))
			}
		} else {
			message.WriteString(fmt.Sprintf("           Duration: %s\n", durationStr))
			if result.Task.Agent != "" {
				message.WriteString(fmt.Sprintf("           Agent: %s\n", result.Task.Agent))
			}
		}
		
		// Files list
		if len(result.Task.Files) > 0 {
			filesStr := strings.Join(result.Task.Files, ", ")
			message.WriteString(fmt.Sprintf("           Files: %s\n", filesStr))
		}
		
		// QC Feedback (if present)
		if result.ReviewFeedback != "" {
			icon := ""
			switch result.Status {
			case models.StatusGreen:
				icon = "✓"
			case models.StatusYellow:
				icon = "⚠️"
			case models.StatusRed:
				icon = "❌"
			}
			message.WriteString(fmt.Sprintf("           QC Feedback: %s %s\n", icon, result.ReviewFeedback))
		}
		
		_, err := cl.writer.Write([]byte(message.String()))
		return err
	}
	
	// Default mode: single-line format with inline details
	var message string
	if cl.colorOutput {
		counterColored := color.New(color.FgCyan).Sprint(progressCounter)
		durationColored := color.New(color.FgHiBlack).Sprint(durationStr)
		agentColored := color.New(color.FgMagenta).Sprint(result.Task.Agent)
		fileCountColored := color.New(color.FgHiBlack).Sprint(fileCountStr)
		
		message = fmt.Sprintf("[%s] %s %s: %s (%s, agent: %s, %s)\n",
			ts, counterColored, taskInfo, statusText, durationColored, agentColored, fileCountColored)
	} else {
		message = fmt.Sprintf("[%s] %s %s: %s (%s, agent: %s, %s)\n",
			ts, progressCounter, taskInfo, statusText, durationStr, result.Task.Agent, fileCountStr)
	}
	
	_, err := cl.writer.Write([]byte(message))
	return err
}
```

**Key points**:
- Add `verbose bool` field to ConsoleLogger struct
- Update NewConsoleLogger signature to accept verbose parameter
- Check `cl.verbose` to determine output format
- In verbose mode: multi-line with indentation (11 spaces to align)
- In default mode: single-line with inline details
- Use existing formatDuration() function (already exists in console.go:381-409)
- Apply colors: cyan (progress), gray/hi-black (duration/file count), magenta (agent)

**Integration points**:
- Imports needed: `strings` (for strings.Builder and strings.Join)
- Services to inject: None
- Config values: verbose flag passed to NewConsoleLogger

### Verification

**Manual testing**:
1. Run with verbose=false, verify single-line output
2. Run with verbose=true, verify multi-line indented output
3. Test with missing fields (no agent, no files), verify graceful handling

**Automated tests**:
```bash
go test ./internal/logger/ -v -run TestLogTaskResult
```

**Expected output**:
All LogTaskResult tests pass for both default and verbose modes.

### Commit

**Commit message**:
```
feat(logger): enhance LogTaskResult with progress and details

Add progress counter [N/M], duration, agent, and file count
to task result logging. Support verbose mode with multi-line
detailed output including QC feedback. Maintains backward
compatibility with single-line default mode.
```

**Files to commit**:
- `internal/logger/console.go`
- `internal/logger/console_test.go`

---

## Task 6: Add LogProgress Method with Progress Bar

**Agent**: golang-pro
**File(s)**: `internal/logger/console.go`, `internal/logger/console_test.go`
**Depends on**: Task 5
**WorktreeGroup**: chain-2
**Estimated time**: 20m

### What you're building
Add LogProgress() method to ConsoleLogger that displays an ASCII progress bar after each task completion, showing overall progress percentage, task counts, and average task duration. Integrates the ProgressBar component created in Task 3.

### Test First (TDD)

**Test file**: `internal/logger/console_test.go`

**Test structure**:
```go
func TestLogProgress_Format(t *testing.T)
  - test format: "Progress: [blocks] percentage (current/total tasks) - Avg: duration/task"
  - verify progress bar rendered correctly
  - verify percentage matches completed/total
  - verify average duration displayed

func TestLogProgress_ProgressBarIntegration(t *testing.T)
  - test ProgressBar component called correctly
  - verify Unicode blocks in output (█ and ░)
  - test at various completion levels (0%, 50%, 100%)

func TestLogProgress_ColorOutput(t *testing.T)
  - test colorOutput=true applies to progress bar
  - verify hi-green for filled blocks
  - verify hi-cyan for percentage

func TestLogProgress_LogLevelFiltering(t *testing.T)
  - test only logs at INFO level or higher
  - verify respects shouldLog check

func TestLogProgress_ZeroTasks(t *testing.T)
  - test edge case: total=0
  - verify doesn't panic
  - verify reasonable output or skip
```

**Test specifics**:
- Mock these dependencies: None (use bytes.Buffer)
- Use these fixtures/factories: None
- Assert these outcomes: Progress bar present, percentage correct, avg duration shown
- Edge cases to cover: 0 total tasks, 0 average duration

**Example test skeleton**:
```go
func TestLogProgress_Format(t *testing.T) {
	var buf bytes.Buffer
	logger := NewConsoleLogger(&buf, "info", false)
	
	avgDuration := 3 * time.Second
	logger.LogProgress(5, 10, avgDuration)
	
	output := buf.String()
	assert.Contains(t, output, "Progress:")
	assert.Contains(t, output, "50%")
	assert.Contains(t, output, "(5/10 tasks)")
	assert.Contains(t, output, "Avg: 3s/task")
	// Verify progress bar characters present
	assert.Contains(t, output, "█")
	assert.Contains(t, output, "░")
}

func TestLogProgress_ProgressBarIntegration(t *testing.T) {
	var buf bytes.Buffer
	logger := &ConsoleLogger{
		writer:      &buf,
		logLevel:    "info",
		colorOutput: false,
	}
	
	logger.LogProgress(7, 10, 2*time.Second)
	
	output := buf.String()
	// At 70%, should have more filled than empty blocks
	filledCount := strings.Count(output, "█")
	emptyCount := strings.Count(output, "░")
	assert.Greater(t, filledCount, emptyCount)
}
```

### Implementation

**Approach**:
Create LogProgress() method that instantiates a ProgressBar with the total task count, renders it with current progress, formats average duration, and writes the complete progress line to output.

**Code structure**:
```go
// Add to internal/logger/console.go after LogTaskResult (around line 350+)

// LogProgress displays an overall progress bar with completion statistics at INFO level.
// Shows ASCII progress bar, percentage, task counts, and average duration per task.
// Format: "Progress: [████████░░] 80% (8/10 tasks) - Avg: 3.2s/task"
func (cl *ConsoleLogger) LogProgress(completed int, total int, avgDuration time.Duration) error {
	if cl.writer == nil {
		return nil
	}
	
	// Progress logging is at INFO level
	if !cl.shouldLog("info") {
		return nil
	}
	
	cl.mutex.Lock()
	defer cl.mutex.Unlock()
	
	// Handle edge case: no tasks
	if total == 0 {
		return nil
	}
	
	// Create progress bar (width: 20 blocks)
	progressBar := NewProgressBar(total, 20, cl.colorOutput)
	barStr := progressBar.Render(completed)
	
	// Format average duration
	avgStr := formatDuration(avgDuration)
	
	// Build complete progress message
	var message string
	if cl.colorOutput {
		// "Progress:" label in default color
		taskCountStr := color.New(color.FgCyan).Sprintf("(%d/%d tasks)", completed, total)
		avgLabelStr := color.New(color.FgHiBlack).Sprintf("Avg: %s/task", avgStr)
		message = fmt.Sprintf("Progress: %s %s - %s\n", barStr, taskCountStr, avgLabelStr)
	} else {
		message = fmt.Sprintf("Progress: %s (%d/%d tasks) - Avg: %s/task\n", 
			barStr, completed, total, avgStr)
	}
	
	_, err := cl.writer.Write([]byte(message))
	return err
}
```

**Key points**:
- Use NewProgressBar(total, 20, cl.colorOutput) from Task 3
- Progress bar width: 20 characters (standard)
- Apply cyan to task count, hi-black/gray to average duration label
- Handle total=0 gracefully (return early, don't panic)
- Use existing formatDuration() function
- Log at INFO level (respects shouldLog check)

**Integration points**:
- Imports needed: None (ProgressBar in same package)
- Services to inject: None
- Config values: Uses existing colorOutput field

### Verification

**Manual testing**:
1. Create test that calls LogProgress at various percentages
2. Run in terminal to verify progress bar renders correctly
3. Verify average duration displays in human-readable format

**Automated tests**:
```bash
go test ./internal/logger/ -v -run TestLogProgress
```

**Expected output**:
All LogProgress tests pass with correct progress bar rendering.

### Commit

**Commit message**:
```
feat(logger): add LogProgress with progress bar display

Implement LogProgress() method that shows ASCII progress bar,
percentage completion, task counts, and average task duration.
Integrates ProgressBar component with color support.
```

**Files to commit**:
- `internal/logger/console.go`
- `internal/logger/console_test.go`

# Tasks 7-12: ConsoleLogger Enhancement

## Task 7: Enhance LogWaveComplete with Status Breakdown

**Agent**: golang-pro
**File(s)**: `internal/logger/console.go`, `internal/logger/console_test.go`
**Depends on**: Task 6
**WorktreeGroup**: chain-3
**Estimated time**: 20m

### What you're building
Enhance the existing LogWaveComplete() method to display status breakdown counts after wave completion. Instead of just showing "Wave 1 complete (13s)", it will show "Wave 1 complete (13s) - 3/3 completed (2 GREEN, 1 YELLOW)" with color-coded status counts. This provides immediate visibility into quality control results.

### Test First (TDD)

**Test file**: `internal/logger/console_test.go`

**Test structure**:
```go
func TestLogWaveComplete_WithStatusBreakdown(t *testing.T)
  - test format includes status counts in parentheses
  - verify "completed (2 GREEN, 1 YELLOW)" format
  - verify task count matches (completed/total)
  - verify duration still displayed

func TestLogWaveComplete_AllGreen(t *testing.T)
  - test all tasks GREEN shows "3/3 completed (3 GREEN)"
  - verify no YELLOW or RED displayed when zero count

func TestLogWaveComplete_MixedStatuses(t *testing.T)
  - test multiple statuses: GREEN, YELLOW, RED
  - verify all non-zero counts displayed
  - verify order: GREEN, YELLOW, RED

func TestLogWaveComplete_ColorCoding(t *testing.T)
  - test colorOutput=true applies hi-green to GREEN count
  - test hi-yellow to YELLOW count
  - test hi-red to RED count
  - test "complete" word in green as before

func TestLogWaveComplete_BackwardCompatibility(t *testing.T)
  - test with empty statusCounts map
  - test with nil statusCounts map
  - verify doesn't crash, shows basic completion
```

**Test specifics**:
- Mock these dependencies: None (use bytes.Buffer)
- Use these fixtures/factories: Create Wave and statusCounts map
- Assert these outcomes: Status counts formatted correctly, colors applied
- Edge cases to cover: Empty map, nil map, single status, all statuses

**Example test skeleton**:
```go
func TestLogWaveComplete_WithStatusBreakdown(t *testing.T) {
	var buf bytes.Buffer
	logger := NewConsoleLogger(&buf, "info", false)

	wave := models.Wave{
		Name:        "Wave 1",
		TaskNumbers: []string{"1", "2", "3"},
	}

	statusCounts := map[string]int{
		models.StatusGreen:  2,
		models.StatusYellow: 1,
	}

	logger.LogWaveComplete(wave, 13*time.Second, statusCounts)

	output := buf.String()
	assert.Contains(t, output, "Wave 1 complete (13s)")
	assert.Contains(t, output, "3/3 completed")
	assert.Contains(t, output, "2 GREEN")
	assert.Contains(t, output, "1 YELLOW")
	assert.NotContains(t, output, "RED") // Zero count not shown
}

func TestLogWaveComplete_ColorCoding(t *testing.T) {
	var buf bytes.Buffer
	logger := &ConsoleLogger{
		writer:      &buf,
		logLevel:    "info",
		colorOutput: true,
	}

	wave := models.Wave{
		Name:        "Wave 1",
		TaskNumbers: []string{"1", "2"},
	}

	statusCounts := map[string]int{
		models.StatusGreen:  1,
		models.StatusYellow: 1,
	}

	logger.LogWaveComplete(wave, 5*time.Second, statusCounts)

	output := buf.String()
	// Verify hi-green for GREEN count
	assert.Contains(t, output, "\x1b[92m")
	// Verify hi-yellow for YELLOW count
	assert.Contains(t, output, "\x1b[93m")
}
```

### Implementation

**Approach**:
Modify the existing LogWaveComplete() method signature to accept a `statusCounts map[string]int` parameter. After the basic completion message, build a status breakdown string that shows completed count and individual status counts (only non-zero). Apply colors to each status count.

**Code structure**:
```go
// Modify existing LogWaveComplete in internal/logger/console.go (around line 236-263)

// LogWaveComplete logs the completion of a wave execution at INFO level.
// Displays status breakdown showing GREEN/YELLOW/RED counts from QC reviews.
// Format: "[HH:MM:SS] <name> complete (<duration>) - <completed>/<total> completed (<status counts>)"
func (cl *ConsoleLogger) LogWaveComplete(wave models.Wave, duration time.Duration, statusCounts map[string]int) {
	if cl.writer == nil {
		return
	}

	// Wave logging is at INFO level
	if !cl.shouldLog("info") {
		return
	}

	cl.mutex.Lock()
	defer cl.mutex.Unlock()

	ts := timestamp()
	durationStr := formatDuration(duration)

	// Calculate total completed from status counts
	totalCompleted := 0
	if statusCounts != nil {
		for _, count := range statusCounts {
			totalCompleted += count
		}
	}
	totalTasks := len(wave.TaskNumbers)

	// Build status breakdown string
	var statusParts []string
	if statusCounts != nil {
		// Order: GREEN, YELLOW, RED
		if count := statusCounts[models.StatusGreen]; count > 0 {
			if cl.colorOutput {
				statusParts = append(statusParts,
					color.New(color.FgHiGreen).Sprintf("%d GREEN", count))
			} else {
				statusParts = append(statusParts, fmt.Sprintf("%d GREEN", count))
			}
		}
		if count := statusCounts[models.StatusYellow]; count > 0 {
			if cl.colorOutput {
				statusParts = append(statusParts,
					color.New(color.FgHiYellow).Sprintf("%d YELLOW", count))
			} else {
				statusParts = append(statusParts, fmt.Sprintf("%d YELLOW", count))
			}
		}
		if count := statusCounts[models.StatusRed]; count > 0 {
			if cl.colorOutput {
				statusParts = append(statusParts,
					color.New(color.FgHiRed).Sprintf("%d RED", count))
			} else {
				statusParts = append(statusParts, fmt.Sprintf("%d RED", count))
			}
		}
	}

	// Build complete message
	var message string
	if cl.colorOutput {
		waveName := color.New(color.Bold).Sprint(wave.Name)
		completeText := color.New(color.FgGreen).Sprint("complete")

		if len(statusParts) > 0 {
			statusBreakdown := strings.Join(statusParts, ", ")
			message = fmt.Sprintf("[%s] %s %s (%s) - %d/%d completed (%s)\n",
				ts, waveName, completeText, durationStr, totalCompleted, totalTasks, statusBreakdown)
		} else {
			message = fmt.Sprintf("[%s] %s %s (%s)\n", ts, waveName, completeText, durationStr)
		}
	} else {
		if len(statusParts) > 0 {
			statusBreakdown := strings.Join(statusParts, ", ")
			message = fmt.Sprintf("[%s] %s complete (%s) - %d/%d completed (%s)\n",
				ts, wave.Name, durationStr, totalCompleted, totalTasks, statusBreakdown)
		} else {
			message = fmt.Sprintf("[%s] %s complete (%s)\n", ts, wave.Name, durationStr)
		}
	}

	cl.writer.Write([]byte(message))
}
```

**Key points**:
- Follow pattern from: Existing LogWaveComplete method (line 236-263)
- Add `statusCounts map[string]int` parameter to method signature
- Calculate totalCompleted by summing all status counts
- Only show statuses with count > 0 (omit zero counts)
- Order statuses: GREEN, YELLOW, RED (success → warning → failure)
- Use `strings.Join()` to combine status parts with ", " separator
- Apply colors: hi-green (92), hi-yellow (93), hi-red (91)
- Handle nil statusCounts gracefully (backward compatibility)

**Integration points**:
- Imports needed: `strings` (for strings.Join - may already be imported from Task 5)
- Services to inject: None
- Config values: Uses existing colorOutput field

### Verification

**Manual testing**:
1. Create test that logs wave completion with mixed statuses
2. Run in terminal to verify colors display correctly
3. Test with nil statusCounts to verify backward compatibility

**Automated tests**:
```bash
go test ./internal/logger/ -v -run TestLogWaveComplete
```

**Expected output**:
All LogWaveComplete tests pass with status breakdown correctly formatted and colored.

### Commit

**Commit message**:
```
feat(logger): add status breakdown to LogWaveComplete

Enhance LogWaveComplete to show QC status counts after wave
completion. Displays completed count with GREEN/YELLOW/RED
breakdown. Color-codes each status count and maintains
backward compatibility with optional statusCounts parameter.
```

**Files to commit**:
- `internal/logger/console.go`
- `internal/logger/console_test.go`

---

## Task 8: Enhance LogSummary with Extended Statistics

**Agent**: golang-pro
**File(s)**: `internal/logger/console.go`, `internal/logger/console_test.go`
**Depends on**: Task 7
**WorktreeGroup**: chain-3
**Estimated time**: 30m

### What you're building
Extend the LogSummary() method to display comprehensive execution statistics including status breakdown (GREEN/YELLOW/RED counts), agent usage statistics (which agents ran how many tasks), total files modified, average task duration, and enhanced failed task details with duration, agent, and QC feedback. This transforms the basic summary into an actionable execution report.

### Test First (TDD)

**Test file**: `internal/logger/console_test.go`

**Test structure**:
```go
func TestLogSummary_StatusBreakdown(t *testing.T)
  - test StatusBreakdown map displayed correctly
  - verify format: "Completed: 7 (6 GREEN, 1 YELLOW)"
  - verify only non-zero statuses shown
  - verify colors applied to each status

func TestLogSummary_AgentUsage(t *testing.T)
  - test AgentUsage map displayed as bulleted list
  - verify format: "  - golang-pro: 5 tasks"
  - verify multiple agents sorted or in deterministic order
  - verify colors applied to agent names (magenta)

func TestLogSummary_TotalFilesAndAvgDuration(t *testing.T)
  - test TotalFiles displayed: "Total files modified: 25"
  - test AvgTaskDuration displayed: "Average task duration: 3.2s"
  - verify formatDuration() used for duration
  - verify colors applied (hi-black/gray)

func TestLogSummary_EnhancedFailedTaskDetails(t *testing.T)
  - test failed task shows duration
  - test failed task shows agent name
  - test failed task shows QC feedback
  - verify format includes all new fields

func TestLogSummary_EmptyStatistics(t *testing.T)
  - test with nil/empty StatusBreakdown
  - test with nil/empty AgentUsage
  - test with zero TotalFiles
  - verify graceful handling, no crashes

func TestLogSummary_ColorOutput(t *testing.T)
  - verify status counts colored (green/yellow/red)
  - verify agent names colored (magenta)
  - verify file count and avg duration colored (hi-black)
  - verify section headers bold
```

**Test specifics**:
- Mock these dependencies: None (use bytes.Buffer)
- Use these fixtures/factories: Create ExecutionResult with all new fields populated
- Assert these outcomes: All statistics displayed, formatted correctly, colored
- Edge cases to cover: Empty maps, zero values, nil values, no failed tasks

**Example test skeleton**:
```go
func TestLogSummary_ExtendedStatistics(t *testing.T) {
	var buf bytes.Buffer
	logger := NewConsoleLogger(&buf, "info", false)

	result := models.ExecutionResult{
		TotalTasks:  10,
		Completed:   8,
		Failed:      2,
		Duration:    5 * time.Minute,
		FailedTasks: []models.TaskResult{
			{
				Task: models.Task{
					Number: "5",
					Name:   "Failed Task",
					Agent:  "golang-pro",
				},
				Status:         models.StatusRed,
				Duration:       30 * time.Second,
				ReviewFeedback: "Tests failed",
			},
		},

		// Extended statistics
		StatusBreakdown: map[string]int{
			models.StatusGreen:  6,
			models.StatusYellow: 2,
		},
		AgentUsage: map[string]int{
			"golang-pro":     5,
			"test-automator": 3,
		},
		TotalFiles:      25,
		AvgTaskDuration: 30 * time.Second,
	}

	logger.LogSummary(result)

	output := buf.String()

	// Verify status breakdown
	assert.Contains(t, output, "Completed: 8 (6 GREEN, 2 YELLOW)")

	// Verify agent usage
	assert.Contains(t, output, "Agent usage:")
	assert.Contains(t, output, "- golang-pro: 5 tasks")
	assert.Contains(t, output, "- test-automator: 3 tasks")

	// Verify files and avg duration
	assert.Contains(t, output, "Total files modified: 25")
	assert.Contains(t, output, "Average task duration: 30s")

	// Verify enhanced failed task details
	assert.Contains(t, output, "Duration: 30s")
	assert.Contains(t, output, "Agent: golang-pro")
	assert.Contains(t, output, "QC Feedback: Tests failed")
}

func TestLogSummary_StatusBreakdownColored(t *testing.T) {
	var buf bytes.Buffer
	logger := &ConsoleLogger{
		writer:      &buf,
		logLevel:    "info",
		colorOutput: true,
	}

	result := models.ExecutionResult{
		TotalTasks: 5,
		Completed:  5,
		Failed:     0,
		Duration:   1 * time.Minute,
		StatusBreakdown: map[string]int{
			models.StatusGreen:  3,
			models.StatusYellow: 2,
		},
		AgentUsage:      map[string]int{},
		TotalFiles:      10,
		AvgTaskDuration: 12 * time.Second,
	}

	logger.LogSummary(result)

	output := buf.String()
	// Verify hi-green for GREEN count
	assert.Contains(t, output, "\x1b[92m")
	// Verify hi-yellow for YELLOW count
	assert.Contains(t, output, "\x1b[93m")
}
```

### Implementation

**Approach**:
Modify the existing LogSummary() method to check for the new ExecutionResult fields (StatusBreakdown, AgentUsage, TotalFiles, AvgTaskDuration). Add sections to display status breakdown within the "Completed" line, agent usage as a bulleted list, and file/duration statistics. Enhance failed task details to include duration, agent, and QC feedback.

**Code structure**:
```go
// Modify existing LogSummary in internal/logger/console.go (around line 309-374)

// LogSummary logs the execution summary with comprehensive statistics at INFO level.
// Displays status breakdown, agent usage, file counts, and enhanced failed task details.
func (cl *ConsoleLogger) LogSummary(result models.ExecutionResult) {
	if cl.writer == nil {
		return
	}

	// Summary logging is at INFO level
	if !cl.shouldLog("info") {
		return
	}

	cl.mutex.Lock()
	defer cl.mutex.Unlock()

	ts := timestamp()
	durationStr := formatDuration(result.Duration)

	var output strings.Builder

	// Header
	if cl.colorOutput {
		header := color.New(color.Bold).Sprint("=== Execution Summary ===")
		output.WriteString(fmt.Sprintf("[%s] %s\n", ts, header))
	} else {
		output.WriteString(fmt.Sprintf("[%s] === Execution Summary ===\n", ts))
	}

	output.WriteString(fmt.Sprintf("[%s] Total tasks: %d\n", ts, result.TotalTasks))

	// Completed with status breakdown
	if result.StatusBreakdown != nil && len(result.StatusBreakdown) > 0 {
		var statusParts []string

		// Build status breakdown: GREEN, YELLOW, RED order
		if count := result.StatusBreakdown[models.StatusGreen]; count > 0 {
			if cl.colorOutput {
				statusParts = append(statusParts,
					color.New(color.FgHiGreen).Sprintf("%d GREEN", count))
			} else {
				statusParts = append(statusParts, fmt.Sprintf("%d GREEN", count))
			}
		}
		if count := result.StatusBreakdown[models.StatusYellow]; count > 0 {
			if cl.colorOutput {
				statusParts = append(statusParts,
					color.New(color.FgHiYellow).Sprintf("%d YELLOW", count))
			} else {
				statusParts = append(statusParts, fmt.Sprintf("%d YELLOW", count))
			}
		}
		if count := result.StatusBreakdown[models.StatusRed]; count > 0 {
			if cl.colorOutput {
				statusParts = append(statusParts,
					color.New(color.FgHiRed).Sprintf("%d RED", count))
			} else {
				statusParts = append(statusParts, fmt.Sprintf("%d RED", count))
			}
		}

		statusBreakdown := strings.Join(statusParts, ", ")

		if cl.colorOutput {
			completedText := color.New(color.FgGreen).Sprintf("Completed: %d (%s)",
				result.Completed, statusBreakdown)
			output.WriteString(fmt.Sprintf("[%s] %s\n", ts, completedText))
		} else {
			output.WriteString(fmt.Sprintf("[%s] Completed: %d (%s)\n",
				ts, result.Completed, statusBreakdown))
		}
	} else {
		// Fallback to simple completed count (backward compatibility)
		if cl.colorOutput {
			completedText := color.New(color.FgGreen).Sprintf("Completed: %d", result.Completed)
			output.WriteString(fmt.Sprintf("[%s] %s\n", ts, completedText))
		} else {
			output.WriteString(fmt.Sprintf("[%s] Completed: %d\n", ts, result.Completed))
		}
	}

	// Failed count (existing logic)
	if result.Failed > 0 {
		if cl.colorOutput {
			failedText := color.New(color.FgRed).Sprintf("Failed: %d", result.Failed)
			output.WriteString(fmt.Sprintf("[%s] %s\n", ts, failedText))
		} else {
			output.WriteString(fmt.Sprintf("[%s] Failed: %d\n", ts, result.Failed))
		}
	} else {
		output.WriteString(fmt.Sprintf("[%s] Failed: %d\n", ts, result.Failed))
	}

	output.WriteString(fmt.Sprintf("[%s] Duration: %s\n", ts, durationStr))

	// Total files modified (NEW)
	if result.TotalFiles > 0 {
		if cl.colorOutput {
			filesText := color.New(color.FgHiBlack).Sprintf("%d", result.TotalFiles)
			output.WriteString(fmt.Sprintf("[%s] Total files modified: %s\n", ts, filesText))
		} else {
			output.WriteString(fmt.Sprintf("[%s] Total files modified: %d\n", ts, result.TotalFiles))
		}
	}

	// Average task duration (NEW)
	if result.AvgTaskDuration > 0 {
		avgStr := formatDuration(result.AvgTaskDuration)
		if cl.colorOutput {
			avgText := color.New(color.FgHiBlack).Sprint(avgStr)
			output.WriteString(fmt.Sprintf("[%s] Average task duration: %s\n", ts, avgText))
		} else {
			output.WriteString(fmt.Sprintf("[%s] Average task duration: %s\n", ts, avgStr))
		}
	}

	// Agent usage statistics (NEW)
	if result.AgentUsage != nil && len(result.AgentUsage) > 0 {
		if cl.colorOutput {
			header := color.New(color.Bold).Sprint("Agent usage:")
			output.WriteString(fmt.Sprintf("[%s] %s\n", ts, header))
		} else {
			output.WriteString(fmt.Sprintf("[%s] Agent usage:\n", ts))
		}

		// Sort agent names for deterministic output
		agentNames := make([]string, 0, len(result.AgentUsage))
		for agent := range result.AgentUsage {
			agentNames = append(agentNames, agent)
		}
		sort.Strings(agentNames)

		for _, agent := range agentNames {
			count := result.AgentUsage[agent]
			taskWord := "tasks"
			if count == 1 {
				taskWord = "task"
			}

			if cl.colorOutput {
				agentColored := color.New(color.FgMagenta).Sprint(agent)
				output.WriteString(fmt.Sprintf("[%s]   - %s: %d %s\n",
					ts, agentColored, count, taskWord))
			} else {
				output.WriteString(fmt.Sprintf("[%s]   - %s: %d %s\n",
					ts, agent, count, taskWord))
			}
		}
	}

	// Enhanced failed task details
	if result.Failed > 0 && len(result.FailedTasks) > 0 {
		if cl.colorOutput {
			failedHeader := color.New(color.FgRed).Sprint("Failed tasks:")
			output.WriteString(fmt.Sprintf("[%s] %s\n", ts, failedHeader))
		} else {
			output.WriteString(fmt.Sprintf("[%s] Failed tasks:\n", ts))
		}

		for _, failedTask := range result.FailedTasks {
			if cl.colorOutput {
				taskName := color.New(color.FgRed).Sprint(failedTask.Task.Name)
				output.WriteString(fmt.Sprintf("[%s]   - Task %s: %s\n",
					ts, taskName, failedTask.Status))
			} else {
				output.WriteString(fmt.Sprintf("[%s]   - Task %s: %s\n",
					ts, failedTask.Task.Name, failedTask.Status))
			}

			// Enhanced details: duration, agent, QC feedback
			if failedTask.Duration > 0 {
				durationStr := formatDuration(failedTask.Duration)
				if cl.colorOutput {
					durationColored := color.New(color.FgHiBlack).Sprint(durationStr)
					output.WriteString(fmt.Sprintf("[%s]     Duration: %s\n", ts, durationColored))
				} else {
					output.WriteString(fmt.Sprintf("[%s]     Duration: %s\n", ts, durationStr))
				}
			}

			if failedTask.Task.Agent != "" {
				if cl.colorOutput {
					agentColored := color.New(color.FgMagenta).Sprint(failedTask.Task.Agent)
					output.WriteString(fmt.Sprintf("[%s]     Agent: %s\n", ts, agentColored))
				} else {
					output.WriteString(fmt.Sprintf("[%s]     Agent: %s\n", ts, failedTask.Task.Agent))
				}
			}

			if failedTask.ReviewFeedback != "" {
				output.WriteString(fmt.Sprintf("[%s]     QC Feedback: %s\n",
					ts, failedTask.ReviewFeedback))
			}
		}
	}

	cl.writer.Write([]byte(output.String()))
}
```

**Key points**:
- Follow pattern from: Existing LogSummary method (line 309-374)
- Use `strings.Builder` for efficient string concatenation
- Check for nil maps before accessing (StatusBreakdown, AgentUsage)
- Sort agent names with `sort.Strings()` for deterministic output
- Build status breakdown inline with "Completed: N (X GREEN, Y YELLOW)"
- Display agent usage as indented bulleted list with proper pluralization
- Show enhanced failed task details with indentation (5 spaces: "     ")
- Apply colors: green (completed), red (failed), magenta (agents), hi-black (durations/files)
- Maintain backward compatibility when new fields are nil/empty

**Integration points**:
- Imports needed: `sort` (for sort.Strings), `strings` (for strings.Builder)
- Services to inject: None
- Config values: Uses existing colorOutput field

### Verification

**Manual testing**:
1. Create ExecutionResult with all new fields populated
2. Run LogSummary and verify all statistics display correctly
3. Test with empty/nil maps to verify graceful handling
4. Verify colors in terminal output

**Automated tests**:
```bash
go test ./internal/logger/ -v -run TestLogSummary
```

**Expected output**:
All LogSummary tests pass with extended statistics correctly displayed and colored.

### Commit

**Commit message**:
```
feat(logger): extend LogSummary with comprehensive statistics

Add status breakdown (GREEN/YELLOW/RED counts), agent usage
bulleted list, total files modified, and average task duration
to execution summary. Enhance failed task details with duration,
agent, and QC feedback. Sort agents for deterministic output.
```

**Files to commit**:
- `internal/logger/console.go`
- `internal/logger/console_test.go`

---

## Task 9: Add ShowQCFeedback Field and Conditional Feedback Display

**Agent**: golang-pro
**File(s)**: `internal/logger/console.go`, `internal/logger/console_test.go`
**Depends on**: Task 8
**WorktreeGroup**: chain-3
**Estimated time**: 15m

### What you're building
Add a `showQCFeedback` field to ConsoleLogger and modify LogTaskResult's verbose mode to conditionally display QC feedback based on this flag. This allows users to control whether inline QC feedback is shown during execution, configured via the Console.ShowQCFeedback config option.

### Test First (TDD)

**Test file**: `internal/logger/console_test.go`

**Test structure**:
```go
func TestLogTaskResult_VerboseModeWithQCFeedbackEnabled(t *testing.T)
  - set verbose=true, showQCFeedback=true
  - verify QC feedback section appears in output
  - verify feedback text displayed with icon

func TestLogTaskResult_VerboseModeWithQCFeedbackDisabled(t *testing.T)
  - set verbose=true, showQCFeedback=false
  - verify QC feedback section NOT in output
  - verify other verbose details still shown (duration, agent, files)

func TestLogTaskResult_DefaultModeIgnoresQCFeedbackFlag(t *testing.T)
  - set verbose=false, showQCFeedback=true
  - verify single-line format (no QC feedback shown anyway)
  - verify flag doesn't affect default mode

func TestNewConsoleLogger_ShowQCFeedbackParameter(t *testing.T)
  - test NewConsoleLogger accepts showQCFeedback parameter
  - verify field stored correctly in struct
  - test both true and false values
```

**Test specifics**:
- Mock these dependencies: None (use bytes.Buffer)
- Use these fixtures/factories: Create TaskResult with ReviewFeedback populated
- Assert these outcomes: QC feedback conditionally displayed based on flag
- Edge cases to cover: Empty feedback with flag enabled, flag in default mode

**Example test skeleton**:
```go
func TestLogTaskResult_QCFeedbackConditional(t *testing.T) {
	tests := []struct{
		name             string
		verbose          bool
		showQCFeedback   bool
		reviewFeedback   string
		expectFeedback   bool
	}{
		{"verbose with feedback enabled", true, true, "All good", true},
		{"verbose with feedback disabled", true, false, "All good", false},
		{"default mode ignores flag", false, true, "All good", false},
		{"no feedback text", true, true, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := &ConsoleLogger{
				writer:         &buf,
				logLevel:       "debug",
				colorOutput:    false,
				verbose:        tt.verbose,
				showQCFeedback: tt.showQCFeedback,
			}

			result := models.TaskResult{
				Task: models.Task{
					Number: "1",
					Name:   "Test",
					Agent:  "golang-pro",
					Files:  []string{"test.go"},
				},
				Status:         models.StatusGreen,
				Duration:       5 * time.Second,
				ReviewFeedback: tt.reviewFeedback,
			}

			logger.LogTaskResult(result, 1, 5)

			output := buf.String()
			if tt.expectFeedback {
				assert.Contains(t, output, "QC Feedback:")
				assert.Contains(t, output, tt.reviewFeedback)
			} else {
				assert.NotContains(t, output, "QC Feedback:")
			}
		})
	}
}

func TestNewConsoleLogger_ShowQCFeedbackParameter(t *testing.T) {
	logger := NewConsoleLogger(os.Stdout, "info", false, true)
	assert.True(t, logger.showQCFeedback)

	logger2 := NewConsoleLogger(os.Stdout, "info", false, false)
	assert.False(t, logger2.showQCFeedback)
}
```

### Implementation

**Approach**:
Add `showQCFeedback bool` field to ConsoleLogger struct. Update NewConsoleLogger signature to accept this parameter. In LogTaskResult's verbose mode section, wrap the QC feedback display in a conditional check: `if cl.showQCFeedback && result.ReviewFeedback != ""`.

**Code structure**:
```go
// 1. Add showQCFeedback field to ConsoleLogger struct (around line 33-38)
type ConsoleLogger struct {
	writer         io.Writer
	logLevel       string
	mutex          sync.Mutex
	colorOutput    bool
	verbose        bool
	showQCFeedback bool  // NEW: Control QC feedback display in verbose mode
}

// 2. Update NewConsoleLogger signature (around line 46)
func NewConsoleLogger(writer io.Writer, logLevel string, verbose bool, showQCFeedback bool) *ConsoleLogger {
	normalizedLevel := normalizeLogLevel(logLevel)
	useColor := isTerminal(writer)

	return &ConsoleLogger{
		writer:         writer,
		logLevel:       normalizedLevel,
		mutex:          sync.Mutex{},
		colorOutput:    useColor,
		verbose:        verbose,
		showQCFeedback: showQCFeedback,  // NEW
	}
}

// 3. Modify LogTaskResult verbose mode QC feedback section (around line 1036-1047 from Task 5)
// Find this section in verbose mode:
		// QC Feedback (if present)
		if cl.showQCFeedback && result.ReviewFeedback != "" {  // ADD showQCFeedback check
			icon := ""
			switch result.Status {
			case models.StatusGreen:
				icon = "✓"
			case models.StatusYellow:
				icon = "⚠️"
			case models.StatusRed:
				icon = "❌"
			}
			message.WriteString(fmt.Sprintf("           QC Feedback: %s %s\n", icon, result.ReviewFeedback))
		}
```

**Key points**:
- Add `showQCFeedback bool` field to ConsoleLogger struct (line ~37)
- Update NewConsoleLogger signature to accept showQCFeedback parameter (line ~46)
- Modify QC feedback conditional: `if cl.showQCFeedback && result.ReviewFeedback != ""`
- Only affects verbose mode (default mode doesn't show QC feedback anyway)
- Maintains backward compatibility by making it a constructor parameter

**Integration points**:
- Imports needed: None
- Services to inject: None
- Config values: Will be passed from Console.ShowQCFeedback config (next tasks)

### Verification

**Manual testing**:
1. Create logger with showQCFeedback=true and verbose=true, verify feedback shown
2. Create logger with showQCFeedback=false and verbose=true, verify feedback hidden
3. Test in default mode, verify flag doesn't affect output

**Automated tests**:
```bash
go test ./internal/logger/ -v -run TestLogTaskResult
go test ./internal/logger/ -v -run TestNewConsoleLogger
```

**Expected output**:
All tests pass with QC feedback conditionally displayed based on flag.

### Commit

**Commit message**:
```
feat(logger): add conditional QC feedback display

Add showQCFeedback field to ConsoleLogger to control inline
QC feedback display in verbose mode. Update NewConsoleLogger
signature and modify LogTaskResult to conditionally show
feedback based on flag. Prepares for config integration.
```

**Files to commit**:
- `internal/logger/console.go`
- `internal/logger/console_test.go`

---

## Task 10: Update Wave Struct with GroupInfo Field

**Agent**: golang-pro
**File(s)**: `internal/models/wave.go`, `internal/models/wave_test.go`
**Depends on**: Task 9
**WorktreeGroup**: chain-3
**Estimated time**: 15m

### What you're building
Add a `GroupInfo map[string]int` field to the Wave struct to track task counts per worktree group within each wave. This enables enhanced wave logging that shows group distribution (e.g., "Wave 1 (backend-core: 2, backend-api: 1)") and supports multi-file plan coordination visibility.

### Test First (TDD)

**Test file**: `internal/models/wave_test.go`

**Test structure**:
```go
func TestWave_GroupInfoTracking(t *testing.T)
  - create Wave with GroupInfo populated
  - verify map stores group names and task counts
  - test multiple groups with different counts

func TestWave_GroupInfoEmpty(t *testing.T)
  - create Wave with empty GroupInfo
  - create Wave with nil GroupInfo
  - verify doesn't crash when accessing

func TestWave_GroupInfoSingleGroup(t *testing.T)
  - test wave with tasks from single group
  - verify GroupInfo contains one entry

func TestWave_GroupInfoMultipleGroups(t *testing.T)
  - test wave with tasks from 3+ groups
  - verify GroupInfo contains all groups
  - verify counts sum to total tasks in wave
```

**Test specifics**:
- Mock these dependencies: None (pure data structure test)
- Use these fixtures/factories: Create Wave structs with GroupInfo populated
- Assert these outcomes: GroupInfo field exists, stores correct counts
- Edge cases to cover: Empty map, nil map, single group, many groups

**Example test skeleton**:
```go
func TestWave_GroupInfoTracking(t *testing.T) {
	wave := models.Wave{
		Name:        "Wave 1",
		TaskNumbers: []string{"1", "2", "3"},
		GroupInfo: map[string]int{
			"backend-core": 2,
			"backend-api":  1,
		},
	}

	assert.Equal(t, 2, wave.GroupInfo["backend-core"])
	assert.Equal(t, 1, wave.GroupInfo["backend-api"])
	assert.Equal(t, 3, len(wave.TaskNumbers))
}

func TestWave_GroupInfoNil(t *testing.T) {
	wave := models.Wave{
		Name:        "Wave 1",
		TaskNumbers: []string{"1"},
		GroupInfo:   nil,
	}

	// Verify accessing nil map doesn't panic
	count := wave.GroupInfo["nonexistent"]
	assert.Equal(t, 0, count)
}

func TestWave_GroupInfoSumsToTotal(t *testing.T) {
	wave := models.Wave{
		Name:        "Wave 2",
		TaskNumbers: []string{"4", "5", "6", "7", "8"},
		GroupInfo: map[string]int{
			"group-a": 3,
			"group-b": 2,
		},
	}

	totalFromGroups := 0
	for _, count := range wave.GroupInfo {
		totalFromGroups += count
	}

	assert.Equal(t, len(wave.TaskNumbers), totalFromGroups)
}
```

### Implementation

**Approach**:
Add a `GroupInfo map[string]int` field to the existing Wave struct. No additional methods needed - this is a pure data structure addition. The field will be populated by the wave calculator/orchestrator in future tasks.

**Code structure**:
```go
// Modify Wave struct in internal/models/wave.go

package models

// Wave represents a set of tasks that can be executed in parallel
type Wave struct {
	Name        string   // Wave name (e.g., "Wave 1", "Wave 2")
	TaskNumbers []string // Task numbers in this wave

	// GroupInfo tracks task counts per worktree group in this wave
	// Key: group name, Value: number of tasks from that group
	// Used for enhanced logging: "Wave 1 (backend-core: 2, api: 1)"
	// May be nil for single-file plans or waves without group distribution
	GroupInfo map[string]int  // NEW
}
```

**Key points**:
- Add GroupInfo field after TaskNumbers field
- Use `map[string]int` to map group name → task count
- Add descriptive comment explaining purpose and usage
- Field is optional (can be nil) for backward compatibility
- No initialization logic needed (will be set by wave calculator)

**Integration points**:
- Imports needed: None
- Services to inject: None
- Config values: None

### Verification

**Manual testing**:
1. Create Wave struct with GroupInfo populated in Go playground
2. Verify map operations work correctly (read, write, nil access)

**Automated tests**:
```bash
go test ./internal/models/ -v -run TestWave
```

**Expected output**:
All Wave tests pass with GroupInfo field accessible and functional.

### Commit

**Commit message**:
```
feat(models): add GroupInfo to Wave for group tracking

Add GroupInfo map[string]int field to Wave struct to track
task counts per worktree group within each wave. Enables
enhanced wave logging showing group distribution. Optional
field maintains backward compatibility.
```

**Files to commit**:
- `internal/models/wave.go`
- `internal/models/wave_test.go`

---

## Task 11: Enhance LogWaveStart with Group Distribution Display

**Agent**: golang-pro
**File(s)**: `internal/logger/console.go`, `internal/logger/console_test.go`
**Depends on**: Task 10
**WorktreeGroup**: chain-3
**Estimated time**: 20m

### What you're building
Modify LogWaveStart() to display worktree group distribution when GroupInfo is populated in the Wave. Format: "Starting Wave 1: 5 tasks (backend-core: 3, api: 2)". This provides visibility into multi-file plan coordination and shows which groups are executing in each wave.

### Test First (TDD)

**Test file**: `internal/logger/console_test.go`

**Test structure**:
```go
func TestLogWaveStart_WithGroupInfo(t *testing.T)
  - test format includes group distribution in parentheses
  - verify "5 tasks (backend-core: 3, api: 2)" format
  - verify multiple groups displayed correctly

func TestLogWaveStart_WithoutGroupInfo(t *testing.T)
  - test with nil GroupInfo
  - test with empty GroupInfo map
  - verify basic format: "Starting Wave 1: 5 tasks" (no groups)

func TestLogWaveStart_SingleGroup(t *testing.T)
  - test wave with tasks from single group
  - verify format: "3 tasks (backend-core: 3)"

func TestLogWaveStart_GroupInfoColorCoding(t *testing.T)
  - test colorOutput=true colors group names
  - verify cyan color applied to group distribution

func TestLogWaveStart_GroupsSorted(t *testing.T)
  - test groups displayed in deterministic order (sorted)
  - verify consistent output across runs
```

**Test specifics**:
- Mock these dependencies: None (use bytes.Buffer)
- Use these fixtures/factories: Create Wave with GroupInfo populated
- Assert these outcomes: Group distribution formatted correctly, sorted
- Edge cases to cover: No groups, single group, many groups, nil map

**Example test skeleton**:
```go
func TestLogWaveStart_WithGroupInfo(t *testing.T) {
	var buf bytes.Buffer
	logger := NewConsoleLogger(&buf, "info", false, false)

	wave := models.Wave{
		Name:        "Wave 1",
		TaskNumbers: []string{"1", "2", "3", "4", "5"},
		GroupInfo: map[string]int{
			"backend-core": 3,
			"backend-api":  2,
		},
	}

	logger.LogWaveStart(wave)

	output := buf.String()
	assert.Contains(t, output, "Starting Wave 1: 5 tasks")
	// Groups should be sorted alphabetically
	assert.Contains(t, output, "(backend-api: 2, backend-core: 3)")
}

func TestLogWaveStart_WithoutGroupInfo(t *testing.T) {
	var buf bytes.Buffer
	logger := NewConsoleLogger(&buf, "info", false, false)

	wave := models.Wave{
		Name:        "Wave 2",
		TaskNumbers: []string{"6", "7"},
		GroupInfo:   nil,  // No group info
	}

	logger.LogWaveStart(wave)

	output := buf.String()
	assert.Contains(t, output, "Starting Wave 2: 2 tasks")
	assert.NotContains(t, output, "(")  // No group distribution
}

func TestLogWaveStart_GroupInfoColorCoding(t *testing.T) {
	var buf bytes.Buffer
	logger := &ConsoleLogger{
		writer:      &buf,
		logLevel:    "info",
		colorOutput: true,
	}

	wave := models.Wave{
		Name:        "Wave 1",
		TaskNumbers: []string{"1", "2"},
		GroupInfo: map[string]int{
			"backend": 2,
		},
	}

	logger.LogWaveStart(wave)

	output := buf.String()
	// Verify cyan color for group distribution
	assert.Contains(t, output, "\x1b[36m")
}
```

### Implementation

**Approach**:
Modify LogWaveStart() to check if wave.GroupInfo is populated. If so, build a group distribution string by sorting group names and formatting as "group1: N, group2: M". Append to the task count in parentheses. Apply cyan color to group distribution when colors enabled.

**Code structure**:
```go
// Modify LogWaveStart in internal/logger/console.go (around line 206-232)

// LogWaveStart logs the start of a wave execution at INFO level.
// Shows task count and group distribution if GroupInfo is populated.
// Format: "[HH:MM:SS] Starting <name>: <count> tasks (group1: N, group2: M)"
func (cl *ConsoleLogger) LogWaveStart(wave models.Wave) {
	if cl.writer == nil {
		return
	}

	// Wave logging is at INFO level
	if !cl.shouldLog("info") {
		return
	}

	cl.mutex.Lock()
	defer cl.mutex.Unlock()

	ts := timestamp()
	taskCount := len(wave.TaskNumbers)

	// Build group distribution string if GroupInfo exists
	var groupDistribution string
	if wave.GroupInfo != nil && len(wave.GroupInfo) > 0 {
		// Sort group names for deterministic output
		groupNames := make([]string, 0, len(wave.GroupInfo))
		for group := range wave.GroupInfo {
			groupNames = append(groupNames, group)
		}
		sort.Strings(groupNames)

		// Build "group1: N, group2: M" string
		groupParts := make([]string, 0, len(groupNames))
		for _, group := range groupNames {
			count := wave.GroupInfo[group]
			groupParts = append(groupParts, fmt.Sprintf("%s: %d", group, count))
		}
		groupDistribution = " (" + strings.Join(groupParts, ", ") + ")"
	}

	var message string
	if cl.colorOutput {
		// Bold/bright for wave headers
		waveName := color.New(color.Bold).Sprint(wave.Name)

		// Color the group distribution if present
		if groupDistribution != "" {
			groupDistribution = color.New(color.FgCyan).Sprint(groupDistribution)
		}

		message = fmt.Sprintf("[%s] Starting %s: %d tasks%s\n",
			ts, waveName, taskCount, groupDistribution)
	} else {
		message = fmt.Sprintf("[%s] Starting %s: %d tasks%s\n",
			ts, wave.Name, taskCount, groupDistribution)
	}

	cl.writer.Write([]byte(message))
}
```

**Key points**:
- Check `wave.GroupInfo != nil && len(wave.GroupInfo) > 0` before building distribution
- Sort group names with `sort.Strings()` for deterministic output
- Format each group as "groupName: count"
- Join with ", " separator and wrap in parentheses: " (group1: N, group2: M)"
- Apply cyan color (`color.FgCyan`) to entire group distribution string
- Gracefully handle nil/empty GroupInfo (backward compatibility)

**Integration points**:
- Imports needed: `sort` (for sort.Strings), `strings` (for strings.Join)
- Services to inject: None
- Config values: Uses existing colorOutput field

### Verification

**Manual testing**:
1. Create Wave with GroupInfo, log wave start, verify group distribution shown
2. Create Wave without GroupInfo, verify basic format displayed
3. Test with colors enabled, verify cyan color applied

**Automated tests**:
```bash
go test ./internal/logger/ -v -run TestLogWaveStart
```

**Expected output**:
All LogWaveStart tests pass with group distribution correctly formatted and sorted.

### Commit

**Commit message**:
```
feat(logger): add group distribution to LogWaveStart

Enhance LogWaveStart to display worktree group distribution
when GroupInfo is populated. Format: "5 tasks (backend: 3, api: 2)".
Sort groups alphabetically for deterministic output. Apply cyan
color to group distribution. Maintains backward compatibility.
```

**Files to commit**:
- `internal/logger/console.go`
- `internal/logger/console_test.go`

---

## Task 12: Update NoOpLogger Interface Compatibility

**Agent**: golang-pro
**File(s)**: `internal/logger/console.go`, `internal/logger/console_test.go`
**Depends on**: Task 11
**WorktreeGroup**: chain-3
**Estimated time**: 10m

### What you're building
Update the NoOpLogger implementations to match the modified ConsoleLogger method signatures from Tasks 4-11. Specifically, update LogWaveComplete to accept statusCounts parameter, and add stub implementations for LogTaskStart and LogProgress. This ensures NoOpLogger maintains interface compatibility.

### Test First (TDD)

**Test file**: `internal/logger/console_test.go`

**Test structure**:
```go
func TestNoOpLogger_LogTaskStart(t *testing.T)
  - verify LogTaskStart method exists
  - verify accepts correct parameters (task, current, total)
  - verify returns error (should be nil)
  - verify no panic when called

func TestNoOpLogger_LogProgress(t *testing.T)
  - verify LogProgress method exists
  - verify accepts correct parameters (completed, total, avgDuration)
  - verify returns error (should be nil)
  - verify no panic when called

func TestNoOpLogger_LogWaveComplete(t *testing.T)
  - verify accepts new statusCounts parameter
  - verify no panic when called with map
  - verify no panic when called with nil map

func TestNoOpLogger_AllMethodsNoOp(t *testing.T)
  - call all NoOpLogger methods
  - verify no output, no errors, no panics
  - verify truly no-op behavior
```

**Test specifics**:
- Mock these dependencies: None
- Use these fixtures/factories: Create sample Task, Wave, TaskResult
- Assert these outcomes: Methods exist, accept correct params, no panics, no output
- Edge cases to cover: Nil parameters, empty parameters

**Example test skeleton**:
```go
func TestNoOpLogger_NewMethods(t *testing.T) {
	logger := NewNoOpLogger()

	task := models.Task{
		Number: "1",
		Name:   "Test",
		Agent:  "golang-pro",
	}

	// Test LogTaskStart
	err := logger.LogTaskStart(task, 1, 10)
	assert.NoError(t, err)

	// Test LogProgress
	err = logger.LogProgress(5, 10, 30*time.Second)
	assert.NoError(t, err)

	// Test LogWaveComplete with statusCounts
	wave := models.Wave{
		Name:        "Wave 1",
		TaskNumbers: []string{"1", "2", "3"},
	}
	statusCounts := map[string]int{
		models.StatusGreen: 3,
	}
	logger.LogWaveComplete(wave, 10*time.Second, statusCounts)

	// No assertions needed - just verify no panic
}

func TestNoOpLogger_NilParameters(t *testing.T) {
	logger := NewNoOpLogger()

	wave := models.Wave{Name: "Wave 1"}

	// Test with nil statusCounts
	assert.NotPanics(t, func() {
		logger.LogWaveComplete(wave, 5*time.Second, nil)
	})
}
```

### Implementation

**Approach**:
Update NoOpLogger struct methods to match the ConsoleLogger interface. Add LogTaskStart and LogProgress stub methods that do nothing and return nil. Update LogWaveComplete signature to accept statusCounts parameter (ignored).

**Code structure**:
```go
// Modify NoOpLogger in internal/logger/console.go (around line 411-436)

// NoOpLogger is a Logger implementation that discards all log messages.
// Useful for testing or when logging is disabled.
type NoOpLogger struct{}

// NewNoOpLogger creates a NoOpLogger instance.
func NewNoOpLogger() *NoOpLogger {
	return &NoOpLogger{}
}

// LogWaveStart is a no-op implementation.
func (n *NoOpLogger) LogWaveStart(wave models.Wave) {
}

// LogWaveComplete is a no-op implementation.
// Accepts statusCounts parameter for interface compatibility but ignores it.
func (n *NoOpLogger) LogWaveComplete(wave models.Wave, duration time.Duration, statusCounts map[string]int) {
}

// LogTaskStart is a no-op implementation.
func (n *NoOpLogger) LogTaskStart(task models.Task, current int, total int) error {
	return nil
}

// LogTaskResult is a no-op implementation.
// Accepts current and total parameters for interface compatibility but ignores them.
func (n *NoOpLogger) LogTaskResult(result models.TaskResult, current int, total int) error {
	return nil
}

// LogProgress is a no-op implementation.
func (n *NoOpLogger) LogProgress(completed int, total int, avgDuration time.Duration) error {
	return nil
}

// LogSummary is a no-op implementation.
func (n *NoOpLogger) LogSummary(result models.ExecutionResult) {
}
```

**Key points**:
- Add statusCounts parameter to LogWaveComplete (ignored)
- Add current, total parameters to LogTaskResult (already existed, verify present)
- Add new LogTaskStart method (returns error for consistency with LogTaskResult)
- Add new LogProgress method (returns error for consistency)
- All methods remain no-op (empty implementations)
- Return nil for error types
- No actual logic needed - just interface compatibility

**Integration points**:
- Imports needed: None (models.Task, models.Wave already imported)
- Services to inject: None
- Config values: None

### Verification

**Manual testing**:
1. Create NoOpLogger and call all methods
2. Verify no output produced
3. Verify no panics or errors

**Automated tests**:
```bash
go test ./internal/logger/ -v -run TestNoOpLogger
```

**Expected output**:
All NoOpLogger tests pass with no panics or output.

### Commit

**Commit message**:
```
feat(logger): update NoOpLogger interface compatibility

Update NoOpLogger to match ConsoleLogger interface changes.
Add LogTaskStart and LogProgress stub methods. Update
LogWaveComplete signature to accept statusCounts parameter.
All methods remain no-op for testing/disabled logging.
```

**Files to commit**:
- `internal/logger/console.go`
- `internal/logger/console_test.go`

---
