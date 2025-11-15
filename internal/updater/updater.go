// Package updater provides atomic, lock-coordinated updates for Conductor plan
// files. It supports both Markdown checkbox updates and YAML status field
// mutations while ensuring cross-process safety via file locking.
//
// Example:
//
//	err := UpdateTaskStatus("plan.md", 12, "completed", timePtr,
//	    WithTimeout(2*time.Second),
//	    WithMonitor(func(metrics UpdateMetrics) { log.Printf("%+v", metrics) }))
//
// UpdateTaskStatus automatically detects the plan format, acquires a file lock,
// applies the update, and writes the result atomically. Optional functional
// parameters expose timeout and metrics monitoring hooks for production usage.
package updater

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/harrison/conductor/internal/filelock"
	"github.com/harrison/conductor/internal/parser"
)

const (
	markdownCheckboxPattern = `^(?P<prefix>\s*[-*+]\s+\[)(?P<mark>[ xX])(?P<suffix>\]\s+Task\s+%s\b.*)$`
)

// ExecutionAttempt captures details of a single task execution attempt.
type ExecutionAttempt struct {
	AttemptNumber int
	Agent         string
	Verdict       string
	AgentOutput   string
	QCFeedback    string
	Timestamp     time.Time
}

var (
	markdownStatusPattern = regexp.MustCompile(`(?i)status\s*:\s*([^)|]+)`)

	// ErrUnsupportedFormat indicates the plan file uses an unsupported format.
	ErrUnsupportedFormat = errors.New("updater: unsupported plan format")
	// ErrTaskNotFound indicates the requested task number cannot be located.
	ErrTaskNotFound = errors.New("updater: task not found")
	// ErrInvalidPlan indicates the plan structure is invalid or cannot be parsed.
	ErrInvalidPlan = errors.New("updater: invalid plan structure")
)

// UpdateMonitor receives metrics describing each plan update attempt.
type UpdateMonitor func(UpdateMetrics)

// UpdateMetrics captures contextual data about a plan update.
type UpdateMetrics struct {
	Path         string
	Format       parser.Format
	TaskNumber   string
	OldStatus    string
	NewStatus    string
	Duration     time.Duration
	BytesRead    int
	BytesWritten int
	Err          error
}

type options struct {
	timeout time.Duration
	monitor UpdateMonitor
}

// Option configures behaviour of UpdateTaskStatus.
type Option func(*options)

// WithTimeout configures how long UpdateTaskStatus should wait when acquiring
// the underlying file lock. A non-positive duration falls back to blocking.
func WithTimeout(d time.Duration) Option {
	return func(o *options) {
		o.timeout = d
	}
}

// WithMonitor registers a callback that receives metrics after each update.
func WithMonitor(m UpdateMonitor) Option {
	return func(o *options) {
		o.monitor = m
	}
}

// UpdateTaskStatus updates the status of a task in the given plan file.
// The plan format (Markdown or YAML) is auto-detected from the file extension.
// For YAML plans, completed tasks get their completed_date/at updated (or
// created) using the provided timestamp. When status is not "completed",
// completion fields are removed. For Markdown plans, task checkboxes are
// toggled and inline status annotations updated. Optional functional options
// expose timeout handling and monitoring hooks.
func UpdateTaskStatus(planPath string, taskNumber string, status string, completedAt *time.Time, opts ...Option) error {
	config := options{}
	for _, opt := range opts {
		if opt != nil {
			opt(&config)
		}
	}

	metrics := UpdateMetrics{
		Path:       planPath,
		TaskNumber: taskNumber,
		NewStatus:  status,
	}
	start := time.Now()
	defer func() {
		metrics.Duration = time.Since(start)
		if config.monitor != nil {
			config.monitor(metrics)
		}
	}()

	format := parser.DetectFormat(planPath)
	metrics.Format = format
	if format == parser.FormatUnknown {
		err := fmt.Errorf("%w: %s", ErrUnsupportedFormat, filepath.Ext(planPath))
		metrics.Err = err
		return err
	}

	lockPath := planPath + ".lock"
	lock := filelock.NewFileLock(lockPath)
	var lockErr error
	if config.timeout > 0 {
		lockErr = lock.LockWithTimeout(config.timeout)
	} else {
		lockErr = lock.Lock()
	}
	if lockErr != nil {
		metrics.Err = lockErr
		return lockErr
	}
	defer func() {
		lock.Unlock()
		os.Remove(lockPath)
	}()

	content, err := os.ReadFile(planPath)
	if err != nil {
		metrics.Err = err
		return err
	}
	metrics.BytesRead = len(content)

	var (
		updated   []byte
		oldStatus string
	)

	switch format {
	case parser.FormatMarkdown:
		updated, oldStatus, err = updateMarkdownPlan(content, taskNumber, status)
	case parser.FormatYAML:
		updated, oldStatus, err = updateYAMLPlan(content, taskNumber, status, completedAt)
	}
	metrics.OldStatus = oldStatus
	if err != nil {
		metrics.Err = err
		return err
	}

	if err := filelock.AtomicWrite(planPath, updated); err != nil {
		metrics.Err = err
		return err
	}

	metrics.BytesWritten = len(updated)
	metrics.Err = nil
	return nil
}

func updateMarkdownPlan(content []byte, taskNumber string, status string) ([]byte, string, error) {
	lines := strings.Split(string(content), "\n")
	re := regexp.MustCompile(fmt.Sprintf(markdownCheckboxPattern, regexp.QuoteMeta(taskNumber)))

	desiredMark := " "
	if strings.EqualFold(status, "completed") {
		desiredMark = "x"
	}

	for i, line := range lines {
		if matches := re.FindStringSubmatch(line); matches != nil {
			prefix := matches[1]
			suffix := matches[3]
			oldStatus := ""
			if sub := markdownStatusPattern.FindStringSubmatch(line); len(sub) > 1 {
				oldStatus = strings.TrimSpace(sub[1])
			}

			lines[i] = prefix + desiredMark + suffix

			if markdownStatusPattern.MatchString(lines[i]) {
				lines[i] = markdownStatusPattern.ReplaceAllString(lines[i], fmt.Sprintf("status: %s", status))
			} else if status != "" {
				lines[i] = strings.TrimRight(lines[i], " ") + fmt.Sprintf(" (status: %s)", status)
			}

			return []byte(strings.Join(lines, "\n")), oldStatus, nil
		}
	}

	return nil, "", fmt.Errorf("%w: task %s not found in markdown plan", ErrTaskNotFound, taskNumber)
}

func updateYAMLPlan(content []byte, taskNumber string, status string, completedAt *time.Time) ([]byte, string, error) {
	var doc yaml.Node

	decoder := yaml.NewDecoder(bytes.NewReader(content))
	if err := decoder.Decode(&doc); err != nil {
		return nil, "", fmt.Errorf("%w: %v", ErrInvalidPlan, err)
	}

	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return nil, "", fmt.Errorf("%w: missing document node", ErrInvalidPlan)
	}

	root := doc.Content[0]
	planNode := findMapValue(root, "plan")
	if planNode == nil {
		return nil, "", fmt.Errorf("%w: plan section not found", ErrInvalidPlan)
	}

	tasksNode := findMapValue(planNode, "tasks")
	if tasksNode == nil || tasksNode.Kind != yaml.SequenceNode {
		return nil, "", fmt.Errorf("%w: tasks sequence not found", ErrInvalidPlan)
	}

	taskNode := findTaskNode(tasksNode, taskNumber)
	if taskNode == nil {
		return nil, "", fmt.Errorf("%w: task %s not found in YAML plan", ErrTaskNotFound, taskNumber)
	}

	oldStatus := getMapScalar(taskNode, "status")
	setMapScalar(taskNode, "status", status)

	if strings.EqualFold(status, "completed") {
		completedKey := detectCompletedKey(taskNode)
		if completedKey == "" {
			completedKey = "completed_date"
		}

		when := time.Now().UTC()
		if completedAt != nil {
			when = completedAt.UTC()
		}

		setMapScalar(taskNode, completedKey, when.Format("2006-01-02"))
	} else {
		removeMapKey(taskNode, "completed_date")
		removeMapKey(taskNode, "completed_at")
	}

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(&doc); err != nil {
		return nil, "", fmt.Errorf("failed to encode YAML plan: %w", err)
	}

	return buf.Bytes(), oldStatus, nil
}

func findMapValue(mapping *yaml.Node, key string) *yaml.Node {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return nil
	}

	for i := 0; i < len(mapping.Content); i += 2 {
		k := mapping.Content[i]
		v := mapping.Content[i+1]
		if k.Value == key {
			return v
		}
	}

	return nil
}

func findTaskNode(tasks *yaml.Node, taskNumber string) *yaml.Node {
	for _, item := range tasks.Content {
		if item.Kind != yaml.MappingNode {
			continue
		}

		numNode := findMapValue(item, "task_number")
		if numNode == nil {
			continue
		}

		// Compare as strings, handling both integer and float/alphanumeric task numbers
		if numNode.Value == taskNumber {
			return item
		}

		// Also try numeric comparison for backward compatibility
		if num, err := strconv.Atoi(numNode.Value); err == nil {
			if fmt.Sprintf("%d", num) == taskNumber {
				return item
			}
		}
	}

	return nil
}

func setMapScalar(mapping *yaml.Node, key, value string) {
	for i := 0; i < len(mapping.Content); i += 2 {
		k := mapping.Content[i]
		if k.Value == key {
			v := mapping.Content[i+1]
			v.Kind = yaml.ScalarNode
			v.Tag = "!!str"
			v.Style = yaml.DoubleQuotedStyle
			v.Value = value
			return
		}
	}

	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}
	valueNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Style: yaml.DoubleQuotedStyle, Value: value}
	mapping.Content = append(mapping.Content, keyNode, valueNode)
}

func getMapScalar(mapping *yaml.Node, key string) string {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return ""
	}

	for i := 0; i < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1].Value
		}
	}

	return ""
}

func removeMapKey(mapping *yaml.Node, key string) {
	for i := 0; i < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			mapping.Content = append(mapping.Content[:i], mapping.Content[i+2:]...)
			return
		}
	}
}

func detectCompletedKey(taskNode *yaml.Node) string {
	for i := 0; i < len(taskNode.Content); i += 2 {
		keyNode := taskNode.Content[i]
		if keyNode.Value == "completed_date" || keyNode.Value == "completed_at" {
			return keyNode.Value
		}
	}
	return ""
}

// UpdateTaskFeedback appends execution history to a task in the plan file.
// Supports both Markdown and YAML plans. Uses file locking to ensure atomic
// read-modify-write operations in concurrent scenarios.
func UpdateTaskFeedback(planPath string, taskNumber string, attempt *ExecutionAttempt) error {
	format := parser.DetectFormat(planPath)
	if format == parser.FormatUnknown {
		return fmt.Errorf("%w: %s", ErrUnsupportedFormat, filepath.Ext(planPath))
	}

	lockPath := planPath + ".lock"
	lock := filelock.NewFileLock(lockPath)

	// Acquire lock - only one goroutine/process can proceed from here
	if err := lock.Lock(); err != nil {
		return err
	}

	// Ensure lock is always released and cleanup happens
	defer func() {
		_ = lock.Unlock()
		_ = os.Remove(lockPath)
	}()

	// Step 1: Read the current file content (under lock)
	content, err := os.ReadFile(planPath)
	if err != nil {
		return err
	}

	// Step 2: Modify the content in memory (under lock)
	var updated []byte
	switch format {
	case parser.FormatMarkdown:
		updated, err = updateMarkdownFeedbackWithLock(content, taskNumber, attempt)
	case parser.FormatYAML:
		updated, err = updateYAMLFeedback(content, taskNumber, attempt)
	default:
		return fmt.Errorf("%w: unsupported format", ErrUnsupportedFormat)
	}

	if err != nil {
		return err
	}

	// Step 3: Write the modified content atomically (under lock)
	// This ensures that the entire read-modify-write cycle is atomic from the perspective
	// of other goroutines/processes attempting to access the same file.
	if err := filelock.AtomicWrite(planPath, updated); err != nil {
		return err
	}

	// Sync directory to ensure filesystem coherency on macOS/APFS.
	// AtomicWrite already syncs the temp file, but directory entry may be cached.
	if dir, err := os.Open(filepath.Dir(planPath)); err == nil {
		_ = dir.Sync()
		_ = dir.Close()
	}

	return nil
}

// updateMarkdownFeedbackWithLock is an internal wrapper that ensures proper locking semantics.
// It's called while the file lock is already held by the caller.
func updateMarkdownFeedbackWithLock(content []byte, taskNumber string, attempt *ExecutionAttempt) ([]byte, error) {
	return updateMarkdownFeedback(content, taskNumber, attempt)
}

func updateMarkdownFeedback(content []byte, taskNumber string, attempt *ExecutionAttempt) ([]byte, error) {
	lines := strings.Split(string(content), "\n")

	// Find task line
	taskPattern := regexp.MustCompile(fmt.Sprintf(`^(?:\s*[-*+]\s+\[[xX ]\]\s+)?Task\s+%s\b`, regexp.QuoteMeta(taskNumber)))
	taskIdx := -1
	for i, line := range lines {
		if taskPattern.MatchString(line) {
			taskIdx = i
			break
		}
	}

	if taskIdx == -1 {
		return nil, fmt.Errorf("%w: task %s not found in markdown plan", ErrTaskNotFound, taskNumber)
	}

	// Find next task or end of file
	nextTaskPattern := regexp.MustCompile(`^(?:\s*[-*+]\s+\[[xX ]\]\s+)?Task\s+\d+\b`)
	endIdx := len(lines)
	for i := taskIdx + 1; i < len(lines); i++ {
		if nextTaskPattern.MatchString(lines[i]) {
			endIdx = i
			break
		}
	}

	// Find or create Execution History section
	historyIdx := -1
	for i := taskIdx + 1; i < endIdx; i++ {
		if strings.TrimSpace(lines[i]) == "### Execution History" {
			historyIdx = i
			break
		}
	}

	// Format attempt with proper spacing
	timestamp := attempt.Timestamp.Format("2006-01-02 15:04:05")
	attemptLines := []string{
		fmt.Sprintf("#### Attempt %d (%s)", attempt.AttemptNumber, timestamp),
		fmt.Sprintf("Agent: %s", attempt.Agent),
		fmt.Sprintf("Verdict: %s", attempt.Verdict),
		"",
		"Agent Output:",
		attempt.AgentOutput,
		"",
		"QC Feedback:",
		attempt.QCFeedback,
		"",
	}

	if historyIdx == -1 {
		// Create new history section at end of task
		newLines := make([]string, 0, len(lines)+len(attemptLines)+4)
		newLines = append(newLines, lines[:endIdx]...)
		newLines = append(newLines, "")
		newLines = append(newLines, "### Execution History")
		newLines = append(newLines, "")
		newLines = append(newLines, attemptLines...)
		newLines = append(newLines, lines[endIdx:]...)
		return []byte(strings.Join(newLines, "\n")), nil
	}

	// Append to existing history section (at the end, before the next task or end of file)
	// endIdx points to the next task (or end of file), which is where we want to insert
	newLines := make([]string, 0, len(lines)+len(attemptLines)+1)
	newLines = append(newLines, lines[:endIdx]...)
	newLines = append(newLines, attemptLines...)
	newLines = append(newLines, lines[endIdx:]...)
	return []byte(strings.Join(newLines, "\n")), nil
}

func updateYAMLFeedback(content []byte, taskNumber string, attempt *ExecutionAttempt) ([]byte, error) {
	var doc yaml.Node

	decoder := yaml.NewDecoder(bytes.NewReader(content))
	if err := decoder.Decode(&doc); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidPlan, err)
	}

	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return nil, fmt.Errorf("%w: missing document node", ErrInvalidPlan)
	}

	root := doc.Content[0]
	planNode := findMapValue(root, "plan")
	if planNode == nil {
		return nil, fmt.Errorf("%w: plan section not found", ErrInvalidPlan)
	}

	tasksNode := findMapValue(planNode, "tasks")
	if tasksNode == nil || tasksNode.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("%w: tasks sequence not found", ErrInvalidPlan)
	}

	taskNode := findTaskNode(tasksNode, taskNumber)
	if taskNode == nil {
		return nil, fmt.Errorf("%w: task %s not found in YAML plan", ErrTaskNotFound, taskNumber)
	}

	// Find or create execution_history array
	historyNode := findMapValue(taskNode, "execution_history")
	if historyNode == nil {
		// Create new sequence
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "execution_history"}
		historyNode = &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		taskNode.Content = append(taskNode.Content, keyNode, historyNode)
	}

	// Create attempt node
	attemptNode := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	addKeyValue := func(key, value string) {
		attemptNode.Content = append(attemptNode.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value})
	}

	addKeyValue("attempt_number", fmt.Sprintf("%d", attempt.AttemptNumber))
	addKeyValue("agent", attempt.Agent)
	addKeyValue("verdict", attempt.Verdict)
	addKeyValue("agent_output", attempt.AgentOutput)
	addKeyValue("qc_feedback", attempt.QCFeedback)
	addKeyValue("timestamp", attempt.Timestamp.Format(time.RFC3339))

	// Append to history
	historyNode.Content = append(historyNode.Content, attemptNode)

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(&doc); err != nil {
		return nil, fmt.Errorf("failed to encode YAML plan: %w", err)
	}

	return buf.Bytes(), nil
}
