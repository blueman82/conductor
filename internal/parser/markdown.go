package parser

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"gopkg.in/yaml.v3"

	"github.com/harrison/conductor/internal/models"
)

type MarkdownParser struct {
	markdown goldmark.Markdown
}

// injectFilesIntoPrompt prepends the target files section to the prompt content
// This ensures agents know exactly which files they MUST create/modify
func injectFilesIntoPrompt(content string, files []string) string {
	if len(files) == 0 {
		return content
	}

	var sb strings.Builder
	sb.WriteString("## Target Files (REQUIRED)\n\n")
	sb.WriteString("**You MUST create/modify these exact files:**\n")
	for _, file := range files {
		fmt.Fprintf(&sb, "- `%s`\n", file)
	}
	sb.WriteString("\n⚠️ Do NOT create files with different names or paths. Use the exact paths listed above.\n\n")
	sb.WriteString(content)
	return sb.String()
}

// conductorConfig represents the optional conductor configuration in frontmatter
type conductorConfig struct {
	DefaultAgent   string              `yaml:"default_agent"`
	QualityControl *qualityControlYAML `yaml:"quality_control"`
}

type qualityControlYAML struct {
	Enabled     bool        `yaml:"enabled"`
	ReviewAgent string      `yaml:"review_agent"`
	RetryOnRed  int         `yaml:"retry_on_red"`
	Agents      *agentsYAML `yaml:"agents"`
}

type agentsYAML struct {
	Mode         string   `yaml:"mode"`
	ExplicitList []string `yaml:"explicit_list"`
	Additional   []string `yaml:"additional"`
	Blocked      []string `yaml:"blocked"`
}

func NewMarkdownParser() *MarkdownParser {
	return &MarkdownParser{
		markdown: goldmark.New(),
	}
}

func (p *MarkdownParser) Parse(r io.Reader) (*models.Plan, error) {
	// Read full content
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read content: %w", err)
	}

	// Extract frontmatter if present
	plan := &models.Plan{}
	content, frontmatter := extractFrontmatter(content)
	if frontmatter != nil {
		if err := parseConductorConfig(frontmatter, plan); err != nil {
			return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
		}
	}

	// Parse markdown AST
	doc := p.markdown.Parser().Parse(text.NewReader(content))

	// Extract tasks
	tasks, err := p.extractTasks(doc, content)
	if err != nil {
		return nil, fmt.Errorf("failed to extract tasks: %w", err)
	}

	plan.Tasks = tasks
	return plan, nil
}

func (p *MarkdownParser) extractTasks(doc ast.Node, source []byte) ([]models.Task, error) {
	var tasks []models.Task
	taskRegex := regexp.MustCompile(`^Task\s+(\d+):\s+(.+)$`)

	// We need to walk through headings and collect task sections
	var currentTask *models.Task
	var taskContent strings.Builder
	var inTask bool

	err := ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		// Check if this is a level 2 heading (## Task N:)
		if heading, ok := n.(*ast.Heading); ok && heading.Level == 2 {
			// If we were processing a task, save it
			if inTask && currentTask != nil {
				// Parse metadata and set prompt
				content := taskContent.String()
				parseTaskMetadata(currentTask, content)
				currentTask.Prompt = content
				tasks = append(tasks, *currentTask)
			}

			// Extract heading text
			headingText := extractText(heading, source)
			matches := taskRegex.FindStringSubmatch(headingText)

			if len(matches) == 3 {
				// Start new task
				currentTask = &models.Task{
					Number: matches[1], // Keep as string
					Name:   matches[2],
				}
				taskContent.Reset()
				inTask = true
			} else {
				// Not a task heading, stop current task
				inTask = false
			}

			return ast.WalkContinue, nil
		}

		return ast.WalkContinue, nil
	})

	if err != nil {
		return nil, err
	}

	// Don't forget the last task
	if inTask && currentTask != nil {
		content := taskContent.String()
		parseTaskMetadata(currentTask, content)
		currentTask.Prompt = content
		tasks = append(tasks, *currentTask)
	}

	// Alternative approach: parse line by line to extract task sections
	// This is more reliable for our use case
	return extractTasksLineByLine(source)
}

// extractTasksLineByLine extracts tasks by parsing markdown line by line
// This is more reliable than walking the AST for our specific format
func extractTasksLineByLine(content []byte) ([]models.Task, error) {
	var tasks []models.Task
	taskRegex := regexp.MustCompile(`^##\s+Task\s+(\d+):\s+(.+)$`)
	codeBlockRegex := regexp.MustCompile(`^` + "```")

	lines := strings.Split(string(content), "\n")
	var currentTask *models.Task
	var taskContent strings.Builder
	inCodeBlock := false

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Track code block state (triple backticks)
		if codeBlockRegex.MatchString(line) {
			inCodeBlock = !inCodeBlock
			// If in task, accumulate the code block markers and content
			if currentTask != nil {
				taskContent.WriteString(line)
				taskContent.WriteString("\n")
			}
			continue
		}

		// Skip task extraction if we're inside a code block
		if inCodeBlock {
			// Still accumulate content if we're in a task
			if currentTask != nil {
				taskContent.WriteString(line)
				taskContent.WriteString("\n")
			}
			continue
		}

		// Check if this is a task heading (only if NOT in code block)
		matches := taskRegex.FindStringSubmatch(line)
		if len(matches) == 3 {
			// Save previous task if exists
			if currentTask != nil {
				content := taskContent.String()
				parseTaskMetadata(currentTask, content)
				currentTask.Prompt = injectFilesIntoPrompt(content, currentTask.Files)
				tasks = append(tasks, *currentTask)
			}

			// Start new task
			currentTask = &models.Task{
				Number: matches[1], // Keep as string
				Name:   strings.TrimSpace(matches[2]),
			}
			taskContent.Reset()
			continue
		}

		// If we're in a task, accumulate content until next ## heading
		if currentTask != nil {
			// Stop at next level 2 heading (but not level 3)
			if strings.HasPrefix(line, "## ") && !strings.HasPrefix(line, "### ") {
				// This is another section, stop current task
				content := taskContent.String()
				parseTaskMetadata(currentTask, content)
				currentTask.Prompt = injectFilesIntoPrompt(content, currentTask.Files)
				tasks = append(tasks, *currentTask)
				currentTask = nil
				taskContent.Reset()
				continue
			}

			// Accumulate task content
			taskContent.WriteString(line)
			taskContent.WriteString("\n")
		}
	}

	// Don't forget the last task
	if currentTask != nil {
		content := taskContent.String()
		parseTaskMetadata(currentTask, content)
		currentTask.Prompt = injectFilesIntoPrompt(content, currentTask.Files)
		tasks = append(tasks, *currentTask)
	}

	return tasks, nil
}

// extractText extracts plain text from an AST node
func extractText(n ast.Node, source []byte) string {
	var buf bytes.Buffer
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if text, ok := c.(*ast.Text); ok {
			buf.Write(text.Segment.Value(source))
		}
	}
	return buf.String()
}

// parseDependenciesFromMarkdown parses dependency strings from Markdown format
// Supports multiple formats:
// - Numeric: "1, 2, 3"
// - Cross-file: "file:plan-01.yaml/task:2" or "file:plan-01.yaml:task:2"
// - Cross-file with spaces: "file:plan-01.yaml / task:2" or "file:plan-01.yaml : task:2"
// - Mixed: "1, file:plan-01.yaml/task:2, 3"
// - Task notation: "Task 1, file:plan-01.yaml/task:2"
func parseDependenciesFromMarkdown(task *models.Task, depStr string) {
	// Split by comma to handle multiple dependencies
	parts := strings.Split(depStr, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)

		// Skip empty parts
		if part == "" {
			continue
		}

		// Check for cross-file dependency pattern (with optional spaces around separators)
		// Patterns: file:NAME/task:ID or file:NAME:task:ID or file:NAME / task:ID or file:NAME : task:ID
		crossFilePatterns := []*regexp.Regexp{
			regexp.MustCompile(`^file:([^/:]+)\s*/\s*task:(.+)$`), // file:name / task:id
			regexp.MustCompile(`^file:([^/:]+)\s*:\s*task:(.+)$`), // file:name : task:id
			regexp.MustCompile(`^file:([^/:]+)/task:(.+)$`),       // file:name/task:id
			regexp.MustCompile(`^file:([^/]+):task:(.+)$`),        // file:name:task:id
		}

		matched := false
		for _, pat := range crossFilePatterns {
			if matches := pat.FindStringSubmatch(part); len(matches) == 3 {
				filename := strings.TrimSpace(matches[1])
				taskID := strings.TrimSpace(matches[2])
				depStr := fmt.Sprintf("file:%s:task:%s", filename, taskID)
				task.DependsOn = append(task.DependsOn, depStr)
				matched = true
				break
			}
		}
		if matched {
			continue
		}

		// Check for "Task N" prefix (e.g., "Task 1", "Task 2.5", "Task integration-1")
		taskPrefixPat := regexp.MustCompile(`^Task\s+(.+)$`)
		if matches := taskPrefixPat.FindStringSubmatch(part); len(matches) == 2 {
			taskNum := strings.TrimSpace(matches[1])
			task.DependsOn = append(task.DependsOn, taskNum)
			continue
		}

		// Check for numeric pattern (int, float, alphanumeric)
		numPat := regexp.MustCompile(`^[\d.]+[a-zA-Z-]*$|^\d+$`)
		if numPat.MatchString(part) {
			task.DependsOn = append(task.DependsOn, part)
			continue
		}

		// If it doesn't match any pattern, try to extract a numeric portion
		numRegex := regexp.MustCompile(`[\d.]+[a-zA-Z-]*|\d+`)
		if matches := numRegex.FindStringSubmatch(part); len(matches) > 0 {
			task.DependsOn = append(task.DependsOn, strings.TrimSpace(matches[0]))
		}
	}
}

// removeCodeBlocks strips code blocks from content to prevent false positives
// in metadata extraction. Code blocks are marked by triple backticks (```).
func removeCodeBlocks(content string) string {
	lines := strings.Split(content, "\n")
	var result strings.Builder
	inCodeBlock := false

	for _, line := range lines {
		// Toggle code block state on triple backticks
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inCodeBlock = !inCodeBlock
			continue
		}

		// Only include lines outside code blocks
		if !inCodeBlock {
			result.WriteString(line)
			result.WriteString("\n")
		}
	}

	return result.String()
}

// parseTaskMetadata extracts metadata fields from task content
func parseTaskMetadata(task *models.Task, content string) {
	// Strip code blocks to prevent extracting metadata from code examples
	contentWithoutCode := removeCodeBlocks(content)

	// Parse **Status**: inline annotation (takes precedence)
	statusRegex := regexp.MustCompile(`\*\*Status\*\*:\s*(\w+)`)
	if matches := statusRegex.FindStringSubmatch(contentWithoutCode); len(matches) > 1 {
		task.Status = strings.TrimSpace(matches[1])
	} else {
		// Parse checkbox [x] or [ ] or [X] as fallback
		checkboxRegex := regexp.MustCompile(`\[([xX ])\]`)
		if matches := checkboxRegex.FindStringSubmatch(contentWithoutCode); len(matches) > 1 {
			checkbox := strings.ToLower(matches[1])
			if checkbox == "x" {
				task.Status = "completed"
			} else {
				task.Status = "pending"
			}
		} else {
			// Default to pending if no status found
			task.Status = "pending"
		}
	}

	// Parse **File(s)**:
	fileRegex := regexp.MustCompile(`\*\*File\(s\)\*\*:\s*(.+)`)
	if matches := fileRegex.FindStringSubmatch(contentWithoutCode); len(matches) > 1 {
		// Extract files from backticks or comma-separated list
		filesStr := matches[1]
		// Split by comma or backtick pairs
		backtickRegex := regexp.MustCompile("`([^`]+)`")
		backtickMatches := backtickRegex.FindAllStringSubmatch(filesStr, -1)

		if len(backtickMatches) > 0 {
			// Use backtick-enclosed filenames
			for _, match := range backtickMatches {
				if len(match) > 1 {
					task.Files = append(task.Files, strings.TrimSpace(match[1]))
				}
			}
		} else {
			// Fall back to comma-separated
			files := strings.Split(filesStr, ",")
			for _, f := range files {
				trimmed := strings.TrimSpace(f)
				if trimmed != "" && trimmed != "None" {
					task.Files = append(task.Files, trimmed)
				}
			}
		}
	}

	// Parse **Depends on**: supports multiple formats
	// - Numeric: "1, 2, 3"
	// - Cross-file: "file:plan-01.yaml/task:2"
	// - Cross-file alt: "file:plan-01-foundation.yaml:task:1"
	// - Mixed: "1, file:plan-01.yaml/task:2, 3"
	depRegex := regexp.MustCompile(`\*\*Depends on\*\*:\s*(.+)`)
	if matches := depRegex.FindStringSubmatch(contentWithoutCode); len(matches) > 1 {
		depStr := strings.TrimSpace(matches[1])
		if !strings.Contains(strings.ToLower(depStr), "none") {
			parseDependenciesFromMarkdown(task, depStr)
		}
	}

	// Parse **Estimated time**:
	// Support formats: "30m", "1h", "2h30m"
	timeRegex := regexp.MustCompile(`\*\*Estimated time\*\*:\s*(.+)`)
	if matches := timeRegex.FindStringSubmatch(contentWithoutCode); len(matches) > 1 {
		timeStr := strings.TrimSpace(matches[1])
		// Try parsing as duration first (handles "2h30m" format)
		if dur, err := parseDuration(timeStr); err == nil {
			task.EstimatedTime = dur
		}
	}

	// Parse **Agent**:
	agentRegex := regexp.MustCompile(`\*\*Agent\*\*:\s*(\S+)`)
	if matches := agentRegex.FindStringSubmatch(contentWithoutCode); len(matches) > 1 {
		task.Agent = strings.TrimSpace(matches[1])
	}

	// Parse **WorktreeGroup**:
	worktreeGroupRegex := regexp.MustCompile(`\*\*WorktreeGroup\*\*:\s*(\S+)`)
	if matches := worktreeGroupRegex.FindStringSubmatch(contentWithoutCode); len(matches) > 1 {
		task.WorktreeGroup = strings.TrimSpace(matches[1])
	}
}

// parseDuration parses time strings like "30m", "1h", "2h30m"
func parseDuration(s string) (time.Duration, error) {
	// Remove any trailing newline or whitespace
	s = strings.TrimSpace(s)

	// Simple patterns: "30m", "1h", "2h"
	simpleRegex := regexp.MustCompile(`^(\d+)([mh])$`)
	if matches := simpleRegex.FindStringSubmatch(s); len(matches) > 2 {
		val, _ := strconv.Atoi(matches[1])
		unit := matches[2]
		if unit == "m" {
			return time.Duration(val) * time.Minute, nil
		}
		return time.Duration(val) * time.Hour, nil
	}

	// Complex pattern: "2h30m"
	complexRegex := regexp.MustCompile(`^(\d+)h(\d+)m$`)
	if matches := complexRegex.FindStringSubmatch(s); len(matches) > 2 {
		hours, _ := strconv.Atoi(matches[1])
		minutes, _ := strconv.Atoi(matches[2])
		return time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute, nil
	}

	// Try standard Go duration parsing
	return time.ParseDuration(s)
}

// extractFrontmatter extracts YAML frontmatter from markdown content
// Returns the content without frontmatter and the frontmatter bytes
func extractFrontmatter(content []byte) ([]byte, []byte) {
	lines := bytes.Split(content, []byte("\n"))

	// Check if starts with ---
	if len(lines) < 3 || !bytes.Equal(bytes.TrimSpace(lines[0]), []byte("---")) {
		return content, nil
	}

	// Find closing ---
	for i := 1; i < len(lines); i++ {
		if bytes.Equal(bytes.TrimSpace(lines[i]), []byte("---")) {
			// Found closing delimiter
			frontmatter := bytes.Join(lines[1:i], []byte("\n"))
			body := bytes.Join(lines[i+1:], []byte("\n"))
			return body, frontmatter
		}
	}

	// No closing delimiter found
	return content, nil
}

// parseConductorConfig parses conductor configuration from frontmatter
func parseConductorConfig(frontmatter []byte, plan *models.Plan) error {
	var config struct {
		Conductor *conductorConfig `yaml:"conductor"`
	}

	if err := yaml.Unmarshal(frontmatter, &config); err != nil {
		return err
	}

	if config.Conductor != nil {
		plan.DefaultAgent = config.Conductor.DefaultAgent

		if config.Conductor.QualityControl != nil {
			plan.QualityControl.Enabled = config.Conductor.QualityControl.Enabled
			plan.QualityControl.ReviewAgent = config.Conductor.QualityControl.ReviewAgent
			plan.QualityControl.RetryOnRed = config.Conductor.QualityControl.RetryOnRed

			// Parse QC agent configuration if present
			if config.Conductor.QualityControl.Agents != nil {
				agents := config.Conductor.QualityControl.Agents

				// Normalize and validate mode
				mode := strings.ToLower(strings.TrimSpace(agents.Mode))
				validModes := map[string]bool{"auto": true, "explicit": true, "mixed": true, "intelligent": true, "": true}
				if !validModes[mode] {
					return fmt.Errorf("invalid QC agents mode: %q", mode)
				}

				// Explicit mode requires explicit_list
				if mode == "explicit" && len(agents.ExplicitList) == 0 {
					return fmt.Errorf("explicit mode requires non-empty explicit_list")
				}

				plan.QualityControl.Agents = models.QCAgentConfig{
					Mode:             mode,
					ExplicitList:     agents.ExplicitList,
					AdditionalAgents: agents.Additional,
					BlockedAgents:    agents.Blocked,
				}
			}
		}
	}

	return nil
}
