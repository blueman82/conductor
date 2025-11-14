# Tasks 7-12: ConsoleLogger Enhancement

## Task 7: Enhance LogWaveComplete with Status Breakdown

**Agent**: `golang-pro`
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

**Agent**: `golang-pro`
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

**Agent**: `golang-pro`
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

**Agent**: `golang-pro`
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

**Agent**: `golang-pro`
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

**Agent**: `golang-pro`
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
