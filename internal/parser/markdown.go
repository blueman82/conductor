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

// markdownPlannerCompliance represents planner compliance in frontmatter (v2.9+)
type markdownPlannerCompliance struct {
	PlannerVersion    string   `yaml:"planner_version"`
	StrictEnforcement bool     `yaml:"strict_enforcement"`
	RequiredFeatures  []string `yaml:"required_features"`
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

// parseType extracts the Type field from task content (component or integration)
// Returns empty string if Type field not found (backward compatible - defaults to component)
func parseType(content string) string {
	// Regex pattern: \*\*Type\*\*:\s*(.+)
	// Matches **Type**: followed by whitespace and value, capturing value group
	typeRegex := regexp.MustCompile(`\*\*Type\*\*:\s*(.+)`)
	if matches := typeRegex.FindStringSubmatch(content); len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	// Return empty string if Type field not found (backward compatible)
	return ""
}

// parseSuccessCriteria: markdown variant (invoked from parseTaskMetadata function)
// Extracts Success Criteria bullet list from markdown task content
// Finds **Success Criteria**: heading and extracts following bullet list items
// Returns []string with all criteria, empty slice if section not found (backward compatible)
func parseSuccessCriteriaMarkdown(content string) []string {
	// Find the **Success Criteria**: heading
	headingRegex := regexp.MustCompile(`(?m)^\*\*Success Criteria\*\*:\s*$`)
	headingMatch := headingRegex.FindStringIndex(content)

	if headingMatch == nil {
		// Section not found, return empty slice (backward compatible)
		return []string{}
	}

	// Extract content after the heading
	startPos := headingMatch[1]
	remainingContent := content[startPos:]

	// Find lines until next heading (## or **) or double newline
	lines := strings.Split(remainingContent, "\n")
	var criteria []string
	var currentCriterion strings.Builder

	for _, line := range lines {
		// Stop at next section heading (## or **)
		if strings.HasPrefix(strings.TrimSpace(line), "##") ||
		   (strings.HasPrefix(strings.TrimSpace(line), "**") && strings.Contains(line, ":")) {
			break
		}

		// Stop at double newline (empty line after content)
		if strings.TrimSpace(line) == "" && currentCriterion.Len() > 0 {
			// Finish current criterion if any
			criterion := strings.TrimSpace(currentCriterion.String())
			if criterion != "" {
				criteria = append(criteria, criterion)
			}
			currentCriterion.Reset()
			// Check if this is truly end of section (another empty line follows)
			continue
		}

		// Match bullet point: line starts with optional whitespace, then dash, then content
		bulletRegex := regexp.MustCompile(`^\s*-\s+(.+)$`)
		if matches := bulletRegex.FindStringSubmatch(line); len(matches) > 1 {
			// Save previous criterion if exists
			if currentCriterion.Len() > 0 {
				criterion := strings.TrimSpace(currentCriterion.String())
				if criterion != "" {
					criteria = append(criteria, criterion)
				}
			}
			// Start new criterion
			currentCriterion.Reset()
			currentCriterion.WriteString(matches[1])
		} else if strings.HasPrefix(line, "  ") && currentCriterion.Len() > 0 {
			// Continuation line (indented with 2+ spaces)
			currentCriterion.WriteString(" ")
			currentCriterion.WriteString(strings.TrimLeft(line, " \t"))
		} else if strings.TrimSpace(line) == "" && currentCriterion.Len() > 0 {
			// Empty line might mark end of section
			continue
		}
	}

	// Don't forget the last criterion
	if currentCriterion.Len() > 0 {
		criterion := strings.TrimSpace(currentCriterion.String())
		if criterion != "" {
			criteria = append(criteria, criterion)
		}
	}

	return criteria
}

// parseTaskMetadata extracts metadata fields from task content
func parseTaskMetadata(task *models.Task, content string) {
	// Strip code blocks to prevent extracting metadata from code examples
	contentWithoutCode := removeCodeBlocks(content)

	// Parse **Type**: inline annotation (component or integration)
	task.Type = parseType(contentWithoutCode)

	// Parse **Success Criteria**: bullet list (called from parseTaskMetadata)
	task.SuccessCriteria = parseSuccessCriteriaMarkdown(contentWithoutCode) // parseSuccessCriteria extracted

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

	// Parse **Test Commands**: (supports both bullet list and code block formats)
	task.TestCommands = parseTestCommands(content)
}

// parseTestCommands extracts test commands from markdown task content
// Supports two formats:
// 1. Code block: **Test Commands**: \n```bash\ncommand1\ncommand2\n```
// 2. Bullet list: **Test Commands**: \n- command1\n- command2
// Returns []string with all commands, empty slice if section not found (backward compatible)
func parseTestCommands(content string) []string {
	// Find the **Test Commands**: heading
	headingRegex := regexp.MustCompile(`(?m)^\*\*Test Commands\*\*:\s*$`)
	headingMatch := headingRegex.FindStringIndex(content)

	if headingMatch == nil {
		// Section not found, return empty slice (backward compatible)
		return []string{}
	}

	// Extract content after the heading
	startPos := headingMatch[1]
	remainingContent := content[startPos:]

	// Check if next non-blank line is a code block or bullet list
	lines := strings.Split(remainingContent, "\n")

	// Skip empty lines to find the first non-empty line
	var firstContentLine string
	var firstContentIdx int
	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			firstContentLine = strings.TrimSpace(line)
			firstContentIdx = i
			break
		}
	}

	// If no content found, return empty slice
	if firstContentLine == "" {
		return []string{}
	}

	// Check if it's a code block (starts with ```)
	if strings.HasPrefix(firstContentLine, "```") {
		return parseTestCommandsCodeBlock(lines, firstContentIdx)
	}

	// Otherwise, assume it's a bullet list
	return parseTestCommandsBulletList(lines, firstContentIdx)
}

// parseTestCommandsCodeBlock extracts commands from a code block
// Starting from the line with opening ``` (at startIdx)
func parseTestCommandsCodeBlock(lines []string, startIdx int) []string {
	var commands []string

	// startIdx points to the ```bash line, so skip it
	for i := startIdx + 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		// Stop at closing ```
		if line == "```" {
			break
		}

		// Skip empty lines, but collect non-empty lines as commands
		if line != "" {
			commands = append(commands, line)
		}
	}

	return commands
}

// parseTestCommandsBulletList extracts commands from a bullet list
// Starting from the line with first bullet (at startIdx)
func parseTestCommandsBulletList(lines []string, startIdx int) []string {
	var commands []string

	for i := startIdx; i < len(lines); i++ {
		line := lines[i]

		// Stop at next section heading (## or **) or code block
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, "##") ||
			(strings.HasPrefix(trimmedLine, "**") && strings.Contains(trimmedLine, ":")) ||
			strings.HasPrefix(trimmedLine, "```") {
			break
		}

		// Match bullet point: line starts with optional whitespace, then dash, then content
		bulletRegex := regexp.MustCompile(`^\s*-\s+(.+)$`)
		if matches := bulletRegex.FindStringSubmatch(line); len(matches) > 1 {
			command := strings.TrimSpace(matches[1])
			if command != "" {
				commands = append(commands, command)
			}
		} else if strings.HasPrefix(line, "  ") && len(commands) > 0 {
			// Continuation line (indented with 2+ spaces) - append to last command
			continuation := strings.TrimLeft(line, " \t")
			if continuation != "" {
				commands[len(commands)-1] = commands[len(commands)-1] + " " + continuation
			}
		} else if strings.TrimSpace(line) == "" {
			// Empty line might mark end of section, but continue checking for more bullets
			continue
		}
	}

	return commands
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

// markdownDataFlowRegistry represents data flow registry in frontmatter (v2.9+)
type markdownDataFlowRegistry struct {
	Producers            map[string][]markdownDataFlowEntry       `yaml:"producers"`
	Consumers            map[string][]markdownDataFlowEntry       `yaml:"consumers"`
	DocumentationTargets map[string][]markdownDocumentationTarget `yaml:"documentation_targets"`
}

// markdownDataFlowEntry represents a single producer/consumer entry
type markdownDataFlowEntry struct {
	Task        interface{} `yaml:"task"`
	Symbol      string      `yaml:"symbol,omitempty"`
	Description string      `yaml:"description,omitempty"`
}

// markdownDocumentationTarget represents a documentation location in frontmatter
type markdownDocumentationTarget struct {
	Location string `yaml:"location"`
	Section  string `yaml:"section,omitempty"`
}

// parseConductorConfig parses conductor configuration from frontmatter
func parseConductorConfig(frontmatter []byte, plan *models.Plan) error {
	var config struct {
		Conductor         *conductorConfig           `yaml:"conductor"`
		PlannerCompliance *markdownPlannerCompliance `yaml:"planner_compliance"`
		DataFlowRegistry  *markdownDataFlowRegistry  `yaml:"data_flow_registry"`
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

	// Parse planner compliance if present (v2.9+)
	if config.PlannerCompliance != nil {
		if config.PlannerCompliance.PlannerVersion == "" {
			return fmt.Errorf("planner_compliance: planner_version is required")
		}
		plan.PlannerCompliance = &models.PlannerComplianceSpec{
			PlannerVersion:    config.PlannerCompliance.PlannerVersion,
			StrictEnforcement: config.PlannerCompliance.StrictEnforcement,
			RequiredFeatures:  config.PlannerCompliance.RequiredFeatures,
		}
	}

	// Parse data flow registry if present (v2.9+)
	if config.DataFlowRegistry != nil {
		registry, err := parseMarkdownDataFlowRegistry(config.DataFlowRegistry)
		if err != nil {
			return fmt.Errorf("failed to parse data_flow_registry: %w", err)
		}
		plan.DataFlowRegistry = registry
	}

	// Validate data flow registry requirements
	if IsDataFlowRegistryRequired(plan.PlannerCompliance) {
		if err := ValidateDataFlowRegistry(plan.DataFlowRegistry, true); err != nil {
			return err
		}
	}

	return nil
}

// parseMarkdownDataFlowRegistry converts markdown frontmatter registry to models
func parseMarkdownDataFlowRegistry(raw *markdownDataFlowRegistry) (*models.DataFlowRegistry, error) {
	if raw == nil {
		return nil, nil
	}

	registry := &models.DataFlowRegistry{
		Producers:            make(map[string][]models.DataFlowEntry),
		Consumers:            make(map[string][]models.DataFlowEntry),
		DocumentationTargets: make(map[string][]models.DocumentationTarget),
	}

	// Parse producers
	for symbol, entries := range raw.Producers {
		for i, entry := range entries {
			taskNum, err := convertMarkdownTaskNum(entry.Task)
			if err != nil {
				return nil, fmt.Errorf("data_flow_registry.producers[%s][%d]: invalid task: %w", symbol, i, err)
			}
			registry.Producers[symbol] = append(registry.Producers[symbol], models.DataFlowEntry{
				TaskNumber:  taskNum,
				Symbol:      entry.Symbol,
				Description: entry.Description,
			})
		}
	}

	// Parse consumers
	for symbol, entries := range raw.Consumers {
		for i, entry := range entries {
			taskNum, err := convertMarkdownTaskNum(entry.Task)
			if err != nil {
				return nil, fmt.Errorf("data_flow_registry.consumers[%s][%d]: invalid task: %w", symbol, i, err)
			}
			registry.Consumers[symbol] = append(registry.Consumers[symbol], models.DataFlowEntry{
				TaskNumber:  taskNum,
				Symbol:      entry.Symbol,
				Description: entry.Description,
			})
		}
	}

	// Parse documentation targets
	for taskNum, targets := range raw.DocumentationTargets {
		for _, target := range targets {
			registry.DocumentationTargets[taskNum] = append(registry.DocumentationTargets[taskNum], models.DocumentationTarget{
				Location: target.Location,
				Section:  target.Section,
			})
		}
	}

	return registry, nil
}

// convertMarkdownTaskNum converts task number from interface{} to string
func convertMarkdownTaskNum(val interface{}) (string, error) {
	if val == nil {
		return "", fmt.Errorf("task number is nil")
	}

	switch v := val.(type) {
	case string:
		return v, nil
	case int:
		return fmt.Sprintf("%d", v), nil
	case float64:
		if v == float64(int(v)) {
			return fmt.Sprintf("%d", int(v)), nil
		}
		return fmt.Sprintf("%g", v), nil
	default:
		return "", fmt.Errorf("unsupported type: %T", val)
	}
}
