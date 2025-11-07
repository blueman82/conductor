# Implementation Plan: Conductor V1 - Multi-Agent Orchestration System

**Created**: 2025-11-07
**Target**: Build autonomous multi-agent orchestration system in Go for executing implementation plans with Claude Code agents
**Estimated Tasks**: 25

## Context for the Engineer

You are building Conductor from scratch - a greenfield Go project that:
- Uses Go 1.21+ with goroutines for concurrency
- Follows cobra CLI framework patterns (like kubectl, docker CLI)
- Tests with Go's built-in testing package
- Parses both Markdown and YAML plan formats
- Spawns Claude Code CLI subprocesses
- Implements file locking for concurrent writes

**You are expected to**:
- Write tests BEFORE implementation (TDD - Red, Green, Refactor)
- Commit frequently (after each completed task)
- Follow Go idioms and conventions
- Keep changes minimal (YAGNI - You Aren't Gonna Need It)
- Avoid duplication (DRY - Don't Repeat Yourself)

**Prerequisites Checklist**:
- [ ] Go 1.21 or later installed (`go version`)
- [ ] Claude Code CLI installed and in PATH (`which claude`)
- [ ] Git initialized in project directory
- [ ] Editor with Go support (VS Code + Go extension recommended)

---

## Task 1: Initialize Go Module and Project Structure

**File(s)**: `go.mod`, `main.go`, directory structure
**Depends on**: None
**Estimated time**: 15m

### What you're building
Create the foundational Go module structure with proper directory organization following Go project layout standards.

### Test First (TDD)

**Test file**: `main_test.go`

**Test structure**:
```go
TestMainExists - verify main.go compiles
TestVersionCommand - verify version can be displayed
```

**Test specifics**:
- No mocks needed yet
- Just verify project compiles
- Test version constant exists

**Example test skeleton**:
```go
package main

import (
    "testing"
)

func TestVersionConstant(t *testing.T) {
    if Version == "" {
        t.Error("Version constant should not be empty")
    }
}
```

### Implementation

**Approach**:
Initialize Go module, create directory structure following standard Go project layout, set up basic main.go entry point.

**Code structure**:
```
conductor/
├── cmd/
│   └── conductor/
│       └── main.go          # CLI entry point
├── internal/
│   ├── parser/              # Plan file parsing
│   ├── executor/            # Task execution engine
│   ├── agent/               # Agent discovery and invocation
│   └── models/              # Data structures
├── pkg/                     # Public packages (if any)
├── docs/
│   └── plans/               # Implementation plans
├── go.mod
├── go.sum
├── README.md
└── Makefile
```

**Key points**:
- Use `go mod init github.com/yourusername/conductor`
- Version format: `const Version = "1.0.0"`
- Follow https://github.com/golang-standards/project-layout

**Integration points**:
- No external dependencies yet
- Pure Go standard library

### Verification

**Manual testing**:
1. Run `go mod init github.com/yourusername/conductor`
2. Run `go build ./cmd/conductor`
3. Run `./conductor` - should compile without errors

**Automated tests**:
```bash
go test ./...
```

**Expected output**:
```
ok      github.com/yourusername/conductor    0.001s
```

### Commit

**Commit message**:
```
feat: initialize Go module and project structure

- Create go.mod with Go 1.21
- Set up standard Go project layout
- Add basic main.go with version constant
- Create directory structure for internal packages
```

**Files to commit**:
- `go.mod`
- `cmd/conductor/main.go`
- `README.md`
- `.gitignore`

---

## Task 2: Install and Configure Cobra CLI Framework

**File(s)**: `go.mod`, `cmd/conductor/main.go`, `internal/cmd/root.go`
**Depends on**: Task 1
**Estimated time**: 30m

### What you're building
Set up cobra CLI framework to handle commands like `conductor run`, `conductor validate`, with proper flag parsing.

### Test First (TDD)

**Test file**: `internal/cmd/root_test.go`

**Test structure**:
```go
TestRootCommandExists - verify root command can be created
TestRootCommandHasSubcommands - verify run/validate subcommands exist
TestVersionFlag - verify --version flag works
```

**Test specifics**:
- Mock cobra.Command execution
- Test flag parsing
- Verify help text is present

**Example test skeleton**:
```go
package cmd

import (
    "bytes"
    "testing"
)

func TestRootCommand(t *testing.T) {
    cmd := NewRootCommand()
    if cmd == nil {
        t.Fatal("Root command should not be nil")
    }

    buf := new(bytes.Buffer)
    cmd.SetOut(buf)
    cmd.SetArgs([]string{"--help"})

    err := cmd.Execute()
    if err != nil {
        t.Fatalf("Root command execution failed: %v", err)
    }

    output := buf.String()
    if !bytes.Contains([]byte(output), []byte("Conductor")) {
        t.Error("Help text should contain 'Conductor'")
    }
}
```

### Implementation

**Approach**:
Install cobra package, create root command with version flag, prepare subcommand structure.

**Code structure**:
```go
// internal/cmd/root.go
package cmd

import (
    "fmt"
    "github.com/spf13/cobra"
)

const Version = "1.0.0"

func NewRootCommand() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "conductor",
        Short: "Autonomous multi-agent orchestration system",
        Long: `Conductor executes implementation plans by spawning and managing
multiple Claude Code CLI agents in coordinated waves.`,
        Version: Version,
    }

    return cmd
}
```

**Key points**:
- Use cobra generator or manual setup
- Add persistent flags for global options (--verbose, etc.)
- Set up proper help text
- Follow cobra best practices

**Integration points**:
- Import: `github.com/spf13/cobra`
- Update main.go to call `cmd.Execute()`

### Verification

**Manual testing**:
1. Run `go get github.com/spf13/cobra@latest`
2. Run `go build ./cmd/conductor`
3. Run `./conductor --help`
4. Run `./conductor --version`

**Automated tests**:
```bash
go test ./internal/cmd/...
```

**Expected output**:
```
Conductor v1.0.0
Autonomous multi-agent orchestration system
...
```

### Commit

**Commit message**:
```
feat: add cobra CLI framework

- Install cobra dependency
- Create root command with version flag
- Set up command structure for future subcommands
- Add comprehensive help text
```

**Files to commit**:
- `go.mod`
- `go.sum`
- `internal/cmd/root.go`
- `internal/cmd/root_test.go`
- `cmd/conductor/main.go`

---

## Task 3: Define Core Data Models

**File(s)**: `internal/models/plan.go`, `internal/models/task.go`, `internal/models/result.go`
**Depends on**: Task 1
**Estimated time**: 45m

### What you're building
Define Go structs for Plan, Task, Wave, Result, and Agent that will be used throughout the application.

### Test First (TDD)

**Test file**: `internal/models/models_test.go`

**Test structure**:
```go
TestTaskValidation - verify task validation logic
TestDependencyCycleDetection - verify circular dependency detection
TestWaveCalculation - test dependency graph to wave conversion
```

**Test specifics**:
- Test edge cases: empty dependencies, self-referencing tasks
- Test validation: missing required fields
- No external mocks needed

**Example test skeleton**:
```go
package models

import (
    "testing"
)

func TestTaskValidation(t *testing.T) {
    tests := []struct {
        name    string
        task    Task
        wantErr bool
    }{
        {
            name: "valid task",
            task: Task{
                Number: 1,
                Name:   "Test Task",
                Prompt: "Do something",
            },
            wantErr: false,
        },
        {
            name: "missing name",
            task: Task{
                Number: 1,
                Prompt: "Do something",
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.task.Validate()
            if (err != nil) != tt.wantErr {
                t.Errorf("Task.Validate() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}

func TestDetectCycles(t *testing.T) {
    tasks := []Task{
        {Number: 1, DependsOn: []int{2}},
        {Number: 2, DependsOn: []int{1}}, // Circular!
    }

    if !HasCyclicDependencies(tasks) {
        t.Error("Should detect circular dependency")
    }
}
```

### Implementation

**Approach**:
Create clean data structures with validation methods, dependency checking, and helper functions.

**Code structure**:
```go
// internal/models/task.go
package models

import (
    "errors"
    "time"
)

type Task struct {
    Number        int
    Name          string
    Files         []string
    DependsOn     []int
    EstimatedTime time.Duration
    Agent         string
    Prompt        string
}

func (t *Task) Validate() error {
    if t.Number <= 0 {
        return errors.New("task number must be positive")
    }
    if t.Name == "" {
        return errors.New("task name is required")
    }
    if t.Prompt == "" {
        return errors.New("task prompt is required")
    }
    return nil
}

// internal/models/plan.go
package models

type Plan struct {
    Name          string
    Tasks         []Task
    Waves         []Wave
    DefaultAgent  string
    QualityControl QualityControlConfig
}

type Wave struct {
    Name           string
    TaskNumbers    []int
    MaxConcurrency int
}

type QualityControlConfig struct {
    Enabled      bool
    ReviewAgent  string
    RetryOnRed   int
}

// internal/models/result.go
package models

type TaskResult struct {
    Task          Task
    Status        string // "GREEN", "RED", "TIMEOUT", "FAILED"
    Output        string
    Error         error
    Duration      time.Duration
    RetryCount    int
    ReviewFeedback string
}

type ExecutionResult struct {
    TotalTasks    int
    Completed     int
    Failed        int
    Duration      time.Duration
    FailedTasks   []TaskResult
}
```

**Key points**:
- Use time.Duration for time fields
- Implement Validate() methods
- Add helper functions for common operations
- Use exported fields for JSON/YAML marshaling

**Integration points**:
- Standard library only (`time`, `errors`)
- Will be imported by parser and executor packages

### Verification

**Manual testing**:
1. Create sample instances of each struct
2. Call Validate() methods
3. Verify errors are returned correctly

**Automated tests**:
```bash
go test ./internal/models/...
```

**Expected output**:
```
ok      github.com/yourusername/conductor/internal/models    0.002s
```

### Commit

**Commit message**:
```
feat: define core data models

- Add Task struct with validation
- Add Plan and Wave structs
- Add Result structs for execution tracking
- Implement dependency cycle detection helpers
```

**Files to commit**:
- `internal/models/task.go`
- `internal/models/plan.go`
- `internal/models/result.go`
- `internal/models/models_test.go`

---

## Task 4: Implement Markdown Plan Parser

**File(s)**: `internal/parser/markdown.go`, `internal/parser/markdown_test.go`
**Depends on**: Task 3
**Estimated time**: 2h

### What you're building
Parse Markdown files generated by `/doc` command, extracting tasks with metadata, dependencies, and optional conductor frontmatter.

### Test First (TDD)

**Test file**: `internal/parser/markdown_test.go`

**Test structure**:
```go
TestParseMarkdownPlan - parse valid markdown
TestExtractTasks - extract task sections
TestParseFrontmatter - parse YAML frontmatter
TestParseTaskMetadata - extract File(s), Depends on, Estimated time
TestParseTaskPrompt - extract full task content as prompt
```

**Test specifics**:
- Create test fixture markdown files in `testdata/`
- Mock file reading with in-memory strings
- Test edge cases: no frontmatter, missing metadata
- Assert task count, dependency parsing, prompt extraction

**Example test skeleton**:
```go
package parser

import (
    "strings"
    "testing"

    "github.com/yourusername/conductor/internal/models"
)

func TestParseMarkdownPlan(t *testing.T) {
    markdown := `# Implementation Plan: Test Plan

**Created**: 2025-11-07
**Estimated Tasks**: 2

## Task 1: First Task

**File(s)**: ` + "`file1.go`" + `
**Depends on**: None
**Estimated time**: 30m

### What you're building
Test task description

### Implementation
Implementation details here
`

    parser := NewMarkdownParser()
    plan, err := parser.Parse(strings.NewReader(markdown))
    if err != nil {
        t.Fatalf("Failed to parse markdown: %v", err)
    }

    if len(plan.Tasks) != 1 {
        t.Errorf("Expected 1 task, got %d", len(plan.Tasks))
    }

    task := plan.Tasks[0]
    if task.Number != 1 {
        t.Errorf("Expected task number 1, got %d", task.Number)
    }
    if task.Name != "First Task" {
        t.Errorf("Expected task name 'First Task', got '%s'", task.Name)
    }
}
```

### Implementation

**Approach**:
Use goldmark markdown parser library to parse markdown, extract task sections based on `## Task N:` headings, parse metadata fields with regex, combine full task section as prompt.

**Code structure**:
```go
// internal/parser/markdown.go
package parser

import (
    "bufio"
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

    "github.com/yourusername/conductor/internal/models"
)

type MarkdownParser struct {
    markdown goldmark.Markdown
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
        return nil, err
    }

    // Extract frontmatter if present
    plan := &models.Plan{}
    content, frontmatter := extractFrontmatter(content)
    if frontmatter != nil {
        if err := parseConductorConfig(frontmatter, plan); err != nil {
            return nil, err
        }
    }

    // Parse markdown AST
    doc := p.markdown.Parser().Parse(text.NewReader(content))

    // Extract tasks
    tasks, err := p.extractTasks(doc, content)
    if err != nil {
        return nil, err
    }

    plan.Tasks = tasks
    return plan, nil
}

func (p *MarkdownParser) extractTasks(doc ast.Node, source []byte) ([]models.Task, error) {
    var tasks []models.Task
    taskRegex := regexp.MustCompile(`^##\s+Task\s+(\d+):\s+(.+)$`)

    // Walk AST to find task headings
    ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
        if !entering {
            return ast.WalkContinue, nil
        }

        if heading, ok := n.(*ast.Heading); ok && heading.Level == 2 {
            // Extract heading text
            text := string(heading.Text(source))
            matches := taskRegex.FindStringSubmatch(text)
            if len(matches) == 3 {
                taskNum, _ := strconv.Atoi(matches[1])
                task := models.Task{
                    Number: taskNum,
                    Name:   matches[2],
                }

                // Extract task content (everything until next ## heading)
                content := extractTaskContent(n, source)
                task.Prompt = content

                // Parse metadata fields
                parseTaskMetadata(&task, content)

                tasks = append(tasks, task)
            }
        }
        return ast.WalkContinue, nil
    })

    return tasks, nil
}

func parseTaskMetadata(task *models.Task, content string) {
    // Parse **File(s)**:
    fileRegex := regexp.MustCompile(`\*\*File\(s\)\*\*:\s*(.+)`)
    if matches := fileRegex.FindStringSubmatch(content); len(matches) > 1 {
        files := strings.Split(matches[1], ",")
        for _, f := range files {
            task.Files = append(task.Files, strings.TrimSpace(f))
        }
    }

    // Parse **Depends on**:
    depRegex := regexp.MustCompile(`\*\*Depends on\*\*:\s*(.+)`)
    if matches := depRegex.FindStringSubmatch(content); len(matches) > 1 {
        if !strings.Contains(matches[1], "None") {
            // Parse "Task X, Task Y" or "[X, Y]"
            numRegex := regexp.MustCompile(`\d+`)
            nums := numRegex.FindAllString(matches[1], -1)
            for _, n := range nums {
                if num, err := strconv.Atoi(n); err == nil {
                    task.DependsOn = append(task.DependsOn, num)
                }
            }
        }
    }

    // Parse **Estimated time**:
    timeRegex := regexp.MustCompile(`\*\*Estimated time\*\*:\s*(\d+)([mh])`)
    if matches := timeRegex.FindStringSubmatch(content); len(matches) > 2 {
        val, _ := strconv.Atoi(matches[1])
        unit := matches[2]
        if unit == "m" {
            task.EstimatedTime = time.Duration(val) * time.Minute
        } else {
            task.EstimatedTime = time.Duration(val) * time.Hour
        }
    }

    // Parse **Agent**:
    agentRegex := regexp.MustCompile(`\*\*Agent\*\*:\s*(\S+)`)
    if matches := agentRegex.FindStringSubmatch(content); len(matches) > 1 {
        task.Agent = matches[1]
    }
}
```

**Key points**:
- Use goldmark for robust markdown parsing
- Regex for metadata extraction (fields like **File(s)**:)
- Full task section becomes the prompt
- Handle optional YAML frontmatter with conductor config

**Integration points**:
- Import: `github.com/yuin/goldmark`
- Import: `gopkg.in/yaml.v3`
- Use models.Plan, models.Task

### Verification

**Manual testing**:
1. Create test markdown file from `/doc` output
2. Run parser on file
3. Print extracted tasks and verify

**Automated tests**:
```bash
go test ./internal/parser/ -v
```

**Expected output**:
```
=== RUN   TestParseMarkdownPlan
--- PASS: TestParseMarkdownPlan (0.00s)
=== RUN   TestExtractTasks
--- PASS: TestExtractTasks (0.00s)
PASS
```

### Commit

**Commit message**:
```
feat: implement markdown plan parser

- Add goldmark dependency for markdown parsing
- Extract tasks from ## Task N: headings
- Parse metadata fields (File(s), Depends on, Estimated time, Agent)
- Extract full task content as prompt
- Support optional YAML frontmatter for conductor config
```

**Files to commit**:
- `internal/parser/markdown.go`
- `internal/parser/markdown_test.go`
- `internal/parser/testdata/sample-plan.md`
- `go.mod`
- `go.sum`

---

## Task 5: Implement YAML Plan Parser

**File(s)**: `internal/parser/yaml.go`, `internal/parser/yaml_test.go`
**Depends on**: Task 3
**Estimated time**: 1h

### What you're building
Parse YAML files generated by `/doc-yaml` command, extracting structured task definitions with full metadata.

### Test First (TDD)

**Test file**: `internal/parser/yaml_test.go`

**Test structure**:
```go
TestParseYAMLPlan - parse valid YAML
TestExtractYAMLTasks - extract tasks array
TestParseDependencies - parse depends_on field
TestParseConductorConfig - parse optional conductor section
```

**Test specifics**:
- Create test fixture YAML files in `testdata/`
- Test nested YAML structures
- Test edge cases: missing optional fields
- Verify proper unmarshaling

**Example test skeleton**:
```go
package parser

import (
    "strings"
    "testing"
)

func TestParseYAMLPlan(t *testing.T) {
    yaml := `
plan:
  metadata:
    feature_name: "Test Plan"
    estimated_tasks: 2
  tasks:
    - task_number: 1
      name: "First Task"
      estimated_time: "30m"
      depends_on: []
      description: "Test description"
`

    parser := NewYAMLParser()
    plan, err := parser.Parse(strings.NewReader(yaml))
    if err != nil {
        t.Fatalf("Failed to parse YAML: %v", err)
    }

    if len(plan.Tasks) != 1 {
        t.Errorf("Expected 1 task, got %d", len(plan.Tasks))
    }

    task := plan.Tasks[0]
    if task.Number != 1 {
        t.Errorf("Expected task number 1, got %d", task.Number)
    }
}
```

### Implementation

**Approach**:
Use gopkg.in/yaml.v3 to unmarshal YAML structure, map YAML schema to models.Plan/Task structs, build prompt from description + implementation + test_first sections.

**Code structure**:
```go
// internal/parser/yaml.go
package parser

import (
    "fmt"
    "io"
    "time"

    "gopkg.in/yaml.v3"

    "github.com/yourusername/conductor/internal/models"
)

type YAMLParser struct{}

type yamlPlan struct {
    Conductor *conductorConfig `yaml:"conductor"`
    Plan      struct {
        Metadata struct {
            FeatureName    string `yaml:"feature_name"`
            EstimatedTasks int    `yaml:"estimated_tasks"`
        } `yaml:"metadata"`
        Tasks []yamlTask `yaml:"tasks"`
    } `yaml:"plan"`
}

type yamlTask struct {
    TaskNumber    int      `yaml:"task_number"`
    Name          string   `yaml:"name"`
    Files         []string `yaml:"files"`
    DependsOn     []int    `yaml:"depends_on"`
    EstimatedTime string   `yaml:"estimated_time"`
    Description   string   `yaml:"description"`
    TestFirst     struct {
        TestFile string `yaml:"test_file"`
        Example  string `yaml:"example_skeleton"`
    } `yaml:"test_first"`
    Implementation struct {
        Approach string `yaml:"approach"`
        Code     string `yaml:"code_structure"`
    } `yaml:"implementation"`
}

func NewYAMLParser() *YAMLParser {
    return &YAMLParser{}
}

func (p *YAMLParser) Parse(r io.Reader) (*models.Plan, error) {
    var yp yamlPlan
    decoder := yaml.NewDecoder(r)
    if err := decoder.Decode(&yp); err != nil {
        return nil, fmt.Errorf("failed to decode YAML: %w", err)
    }

    plan := &models.Plan{
        Name: yp.Plan.Metadata.FeatureName,
    }

    // Parse conductor config if present
    if yp.Conductor != nil {
        parseConductorConfigYAML(yp.Conductor, plan)
    }

    // Convert YAML tasks to models.Task
    for _, yt := range yp.Plan.Tasks {
        task := models.Task{
            Number:    yt.TaskNumber,
            Name:      yt.Name,
            Files:     yt.Files,
            DependsOn: yt.DependsOn,
        }

        // Parse estimated time
        if dur, err := parseTimeString(yt.EstimatedTime); err == nil {
            task.EstimatedTime = dur
        }

        // Build comprehensive prompt from all sections
        task.Prompt = buildPromptFromYAML(&yt)

        plan.Tasks = append(plan.Tasks, task)
    }

    return plan, nil
}

func buildPromptFromYAML(yt *yamlTask) string {
    var prompt strings.Builder

    fmt.Fprintf(&prompt, "Task: %s\n\n", yt.Name)
    fmt.Fprintf(&prompt, "%s\n\n", yt.Description)

    if yt.TestFirst.Example != "" {
        fmt.Fprintf(&prompt, "Test First (TDD):\n%s\n\n", yt.TestFirst.Example)
    }

    if yt.Implementation.Approach != "" {
        fmt.Fprintf(&prompt, "Implementation:\n%s\n\n", yt.Implementation.Approach)
    }

    if yt.Implementation.Code != "" {
        fmt.Fprintf(&prompt, "Code Structure:\n%s\n", yt.Implementation.Code)
    }

    return prompt.String()
}

func parseTimeString(s string) (time.Duration, error) {
    // Parse "30m", "1h", "2h30m" format
    return time.ParseDuration(s)
}
```

**Key points**:
- Define YAML schema structs matching /doc-yaml output
- Use struct tags for YAML field mapping
- Combine multiple YAML sections into comprehensive prompt
- Handle missing optional fields gracefully

**Integration points**:
- Import: `gopkg.in/yaml.v3`
- Use models.Plan, models.Task

### Verification

**Manual testing**:
1. Create test YAML file from `/doc-yaml` output
2. Run parser on file
3. Verify all fields extracted correctly

**Automated tests**:
```bash
go test ./internal/parser/ -run TestYAML -v
```

**Expected output**:
```
=== RUN   TestParseYAMLPlan
--- PASS: TestParseYAMLPlan (0.00s)
PASS
```

### Commit

**Commit message**:
```
feat: implement YAML plan parser

- Parse /doc-yaml format with structured schema
- Extract tasks from plan.tasks array
- Build comprehensive prompts from description/test/implementation sections
- Support optional conductor configuration block
```

**Files to commit**:
- `internal/parser/yaml.go`
- `internal/parser/yaml_test.go`
- `internal/parser/testdata/sample-plan.yaml`

---

## Task 6: Implement Plan Parser Interface and Auto-Detection

**File(s)**: `internal/parser/parser.go`, `internal/parser/parser_test.go`
**Depends on**: Task 4, Task 5
**Estimated time**: 30m

### What you're building
Unified Parser interface that auto-detects plan format (Markdown vs YAML) based on file extension or content, returning parsed models.Plan.

### Test First (TDD)

**Test file**: `internal/parser/parser_test.go`

**Test structure**:
```go
TestAutoDetectMarkdown - verify .md extension triggers markdown parser
TestAutoDetectYAML - verify .yaml/.yml extension triggers YAML parser
TestParseFromFile - integration test for file reading and parsing
```

**Test specifics**:
- Mock file system or use testdata/ files
- Test both formats through unified interface
- Verify correct parser is selected

**Example test skeleton**:
```go
package parser

import (
    "os"
    "path/filepath"
    "testing"
)

func TestAutoDetectFormat(t *testing.T) {
    tests := []struct {
        filename string
        want     Format
    }{
        {"plan.md", FormatMarkdown},
        {"plan.yaml", FormatYAML},
        {"plan.yml", FormatYAML},
        {"unknown.txt", FormatUnknown},
    }

    for _, tt := range tests {
        t.Run(tt.filename, func(t *testing.T) {
            got := DetectFormat(tt.filename)
            if got != tt.want {
                t.Errorf("DetectFormat(%s) = %v, want %v", tt.filename, got, tt.want)
            }
        })
    }
}

func TestParseFromFile(t *testing.T) {
    mdPath := filepath.Join("testdata", "sample-plan.md")
    plan, err := ParseFile(mdPath)
    if err != nil {
        t.Fatalf("ParseFile failed: %v", err)
    }

    if len(plan.Tasks) == 0 {
        t.Error("Expected tasks to be parsed")
    }
}
```

### Implementation

**Approach**:
Create Parser interface, implement auto-detection based on file extension, provide convenient ParseFile() function.

**Code structure**:
```go
// internal/parser/parser.go
package parser

import (
    "fmt"
    "io"
    "os"
    "path/filepath"
    "strings"

    "github.com/yourusername/conductor/internal/models"
)

type Format int

const (
    FormatUnknown Format = iota
    FormatMarkdown
    FormatYAML
)

type Parser interface {
    Parse(r io.Reader) (*models.Plan, error)
}

func DetectFormat(filename string) Format {
    ext := strings.ToLower(filepath.Ext(filename))
    switch ext {
    case ".md", ".markdown":
        return FormatMarkdown
    case ".yaml", ".yml":
        return FormatYAML
    default:
        return FormatUnknown
    }
}

func NewParser(format Format) (Parser, error) {
    switch format {
    case FormatMarkdown:
        return NewMarkdownParser(), nil
    case FormatYAML:
        return NewYAMLParser(), nil
    default:
        return nil, fmt.Errorf("unsupported format: %v", format)
    }
}

func ParseFile(path string) (*models.Plan, error) {
    format := DetectFormat(path)
    if format == FormatUnknown {
        return nil, fmt.Errorf("unknown file format: %s", path)
    }

    parser, err := NewParser(format)
    if err != nil {
        return nil, err
    }

    file, err := os.Open(path)
    if err != nil {
        return nil, fmt.Errorf("failed to open file: %w", err)
    }
    defer file.Close()

    plan, err := parser.Parse(file)
    if err != nil {
        return nil, fmt.Errorf("failed to parse plan: %w", err)
    }

    // Set plan file path for later updates
    plan.FilePath = path

    return plan, nil
}
```

**Key points**:
- Use interface for polymorphism
- Auto-detection makes CLI usage simple
- Store original file path in plan for updates

**Integration points**:
- Will be used by cobra commands
- No external dependencies beyond parsers

### Verification

**Manual testing**:
1. Call ParseFile() with sample .md and .yaml files
2. Verify correct parser is used
3. Check returned plan has tasks

**Automated tests**:
```bash
go test ./internal/parser/ -v
```

**Expected output**:
```
PASS
ok      github.com/yourusername/conductor/internal/parser    0.003s
```

### Commit

**Commit message**:
```
feat: add unified parser interface with auto-detection

- Create Parser interface for both formats
- Implement auto-detection based on file extension
- Add convenient ParseFile() function
- Store file path in plan for later updates
```

**Files to commit**:
- `internal/parser/parser.go`
- `internal/parser/parser_test.go`

---

## Task 7: Implement Dependency Graph and Wave Calculator

**File(s)**: `internal/executor/graph.go`, `internal/executor/graph_test.go`
**Depends on**: Task 3
**Estimated time**: 1.5h

### What you're building
Build dependency graph from tasks, detect cycles, calculate execution waves using topological sort (Kahn's algorithm).

### Test First (TDD)

**Test file**: `internal/executor/graph_test.go`

**Test structure**:
```go
TestBuildGraph - verify graph construction from tasks
TestDetectCycle - detect circular dependencies
TestCalculateWaves - calculate execution waves from DAG
TestTopologicalSort - verify Kahn's algorithm implementation
TestIndependentTasks - tasks with no deps go in Wave 1
```

**Test specifics**:
- Test cycle detection with various scenarios
- Test wave calculation with different dependency patterns
- Test edge cases: no dependencies, complex chains
- No external mocks needed

**Example test skeleton**:
```go
package executor

import (
    "testing"

    "github.com/yourusername/conductor/internal/models"
)

func TestDetectCycle(t *testing.T) {
    tests := []struct {
        name      string
        tasks     []models.Task
        wantCycle bool
    }{
        {
            name: "no cycle",
            tasks: []models.Task{
                {Number: 1, DependsOn: []int{}},
                {Number: 2, DependsOn: []int{1}},
            },
            wantCycle: false,
        },
        {
            name: "simple cycle",
            tasks: []models.Task{
                {Number: 1, DependsOn: []int{2}},
                {Number: 2, DependsOn: []int{1}},
            },
            wantCycle: true,
        },
        {
            name: "self reference",
            tasks: []models.Task{
                {Number: 1, DependsOn: []int{1}},
            },
            wantCycle: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            graph := BuildDependencyGraph(tt.tasks)
            hasCycle := graph.HasCycle()
            if hasCycle != tt.wantCycle {
                t.Errorf("HasCycle() = %v, want %v", hasCycle, tt.wantCycle)
            }
        })
    }
}

func TestCalculateWaves(t *testing.T) {
    tasks := []models.Task{
        {Number: 1, Name: "Task 1", DependsOn: []int{}},
        {Number: 2, Name: "Task 2", DependsOn: []int{1}},
        {Number: 3, Name: "Task 3", DependsOn: []int{1}},
        {Number: 4, Name: "Task 4", DependsOn: []int{2, 3}},
    }

    waves, err := CalculateWaves(tasks)
    if err != nil {
        t.Fatalf("CalculateWaves failed: %v", err)
    }

    // Expected: Wave 1: [1], Wave 2: [2,3], Wave 3: [4]
    if len(waves) != 3 {
        t.Errorf("Expected 3 waves, got %d", len(waves))
    }

    if len(waves[0].TaskNumbers) != 1 || waves[0].TaskNumbers[0] != 1 {
        t.Errorf("Wave 1 should contain task 1")
    }

    if len(waves[1].TaskNumbers) != 2 {
        t.Errorf("Wave 2 should contain 2 tasks")
    }
}
```

### Implementation

**Approach**:
Build adjacency list representation of task dependencies, implement DFS for cycle detection, use Kahn's algorithm for topological sort and wave grouping.

**Code structure**:
```go
// internal/executor/graph.go
package executor

import (
    "fmt"

    "github.com/yourusername/conductor/internal/models"
)

type DependencyGraph struct {
    Tasks    map[int]*models.Task
    Edges    map[int][]int // task -> dependencies
    InDegree map[int]int   // task -> number of dependencies
}

func BuildDependencyGraph(tasks []models.Task) *DependencyGraph {
    g := &DependencyGraph{
        Tasks:    make(map[int]*models.Task),
        Edges:    make(map[int][]int),
        InDegree: make(map[int]int),
    }

    // Build task map and initialize in-degree
    for i := range tasks {
        g.Tasks[tasks[i].Number] = &tasks[i]
        g.InDegree[tasks[i].Number] = 0
    }

    // Build edges and calculate in-degree
    for _, task := range tasks {
        for _, dep := range task.DependsOn {
            g.Edges[dep] = append(g.Edges[dep], task.Number)
            g.InDegree[task.Number]++
        }
    }

    return g
}

func (g *DependencyGraph) HasCycle() bool {
    // Use DFS with color marking
    white := 0 // not visited
    gray := 1  // visiting
    black := 2 // visited

    colors := make(map[int]int)
    for taskNum := range g.Tasks {
        colors[taskNum] = white
    }

    var dfs func(int) bool
    dfs = func(node int) bool {
        colors[node] = gray

        for _, neighbor := range g.Edges[node] {
            if colors[neighbor] == gray {
                return true // back edge = cycle
            }
            if colors[neighbor] == white && dfs(neighbor) {
                return true
            }
        }

        colors[node] = black
        return false
    }

    for taskNum := range g.Tasks {
        if colors[taskNum] == white {
            if dfs(taskNum) {
                return true
            }
        }
    }

    return false
}

func CalculateWaves(tasks []models.Task) ([]models.Wave, error) {
    graph := BuildDependencyGraph(tasks)

    // Check for cycles first
    if graph.HasCycle() {
        return nil, fmt.Errorf("circular dependency detected")
    }

    // Kahn's algorithm for topological sort + wave grouping
    var waves []models.Wave
    inDegree := make(map[int]int)
    for k, v := range graph.InDegree {
        inDegree[k] = v
    }

    for len(inDegree) > 0 {
        // Find all tasks with in-degree 0 (current wave)
        var currentWave []int
        for taskNum, degree := range inDegree {
            if degree == 0 {
                currentWave = append(currentWave, taskNum)
            }
        }

        if len(currentWave) == 0 {
            return nil, fmt.Errorf("graph error: no tasks with zero in-degree")
        }

        // Create wave
        wave := models.Wave{
            Name:           fmt.Sprintf("Wave %d", len(waves)+1),
            TaskNumbers:    currentWave,
            MaxConcurrency: 10, // default
        }
        waves = append(waves, wave)

        // Remove current wave tasks and update in-degrees
        for _, taskNum := range currentWave {
            delete(inDegree, taskNum)

            // Decrease in-degree for dependent tasks
            for _, dependent := range graph.Edges[taskNum] {
                if _, exists := inDegree[dependent]; exists {
                    inDegree[dependent]--
                }
            }
        }
    }

    return waves, nil
}
```

**Key points**:
- Use DFS with color marking for cycle detection
- Kahn's algorithm groups independent tasks into waves
- Tasks with zero dependencies go in Wave 1
- Each wave's tasks can run in parallel

**Integration points**:
- Use models.Task, models.Wave
- Will be called by executor before running tasks

### Verification

**Manual testing**:
1. Create sample task lists with various dependency patterns
2. Call CalculateWaves()
3. Verify wave grouping is correct

**Automated tests**:
```bash
go test ./internal/executor/ -run TestGraph -v
```

**Expected output**:
```
=== RUN   TestDetectCycle
--- PASS: TestDetectCycle (0.00s)
=== RUN   TestCalculateWaves
--- PASS: TestCalculateWaves (0.00s)
PASS
```

### Commit

**Commit message**:
```
feat: implement dependency graph and wave calculator

- Build adjacency list dependency graph from tasks
- Implement DFS-based cycle detection
- Implement Kahn's algorithm for topological sort
- Calculate execution waves grouping independent tasks
```

**Files to commit**:
- `internal/executor/graph.go`
- `internal/executor/graph_test.go`

---

## Task 8: Implement Agent Discovery

**File(s)**: `internal/agent/discovery.go`, `internal/agent/discovery_test.go`
**Depends on**: Task 3
**Estimated time**: 45m

### What you're building
Scan ~/.claude/agents/ directory for available agents, parse agent metadata files, provide agent lookup by name.

### Test First (TDD)

**Test file**: `internal/agent/discovery_test.go`

**Test structure**:
```go
TestScanAgentsDirectory - verify directory scanning
TestParseAgentFile - parse .md agent definition
TestAgentExists - check if agent name exists
TestFallbackToGeneralPurpose - verify fallback logic
```

**Test specifics**:
- Mock file system with testdata/ agent files
- Test with various agent configurations
- Test missing agents directory (return empty map, no error)

**Example test skeleton**:
```go
package agent

import (
    "os"
    "path/filepath"
    "testing"
)

func TestDiscoverAgents(t *testing.T) {
    // Create temp directory with test agent files
    tmpDir := t.TempDir()

    // Write sample agent file
    agentContent := `---
name: test-agent
description: Test agent
tools: Read, Write
---
Test agent prompt
`
    err := os.WriteFile(filepath.Join(tmpDir, "test-agent.md"), []byte(agentContent), 0644)
    if err != nil {
        t.Fatal(err)
    }

    registry := NewRegistry(tmpDir)
    agents, err := registry.Discover()
    if err != nil {
        t.Fatalf("Discover failed: %v", err)
    }

    if len(agents) != 1 {
        t.Errorf("Expected 1 agent, got %d", len(agents))
    }

    if _, exists := agents["test-agent"]; !exists {
        t.Error("Expected test-agent to exist")
    }
}

func TestAgentExists(t *testing.T) {
    registry := NewRegistry("testdata/agents")
    registry.Discover()

    if !registry.Exists("swiftdev") {
        t.Error("swiftdev agent should exist")
    }

    if registry.Exists("nonexistent-agent") {
        t.Error("nonexistent-agent should not exist")
    }
}
```

### Implementation

**Approach**:
Walk ~/.claude/agents/ directory, parse .md files with YAML frontmatter, extract agent name and metadata, store in registry map.

**Code structure**:
```go
// internal/agent/discovery.go
package agent

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "gopkg.in/yaml.v3"
)

type Agent struct {
    Name        string
    Description string
    Tools       []string
    FilePath    string
}

type Registry struct {
    AgentsDir string
    agents    map[string]*Agent
}

func NewRegistry(agentsDir string) *Registry {
    if agentsDir == "" {
        // Default to ~/.claude/agents
        home, _ := os.UserHomeDir()
        agentsDir = filepath.Join(home, ".claude", "agents")
    }

    return &Registry{
        AgentsDir: agentsDir,
        agents:    make(map[string]*Agent),
    }
}

func (r *Registry) Discover() (map[string]*Agent, error) {
    // Check if directory exists
    if _, err := os.Stat(r.AgentsDir); os.IsNotExist(err) {
        // No agents directory - return empty map, not an error
        return r.agents, nil
    }

    // Walk directory
    err := filepath.Walk(r.AgentsDir, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }

        if info.IsDir() {
            return nil
        }

        // Only process .md files
        if !strings.HasSuffix(path, ".md") {
            return nil
        }

        agent, err := parseAgentFile(path)
        if err != nil {
            // Log warning but continue
            fmt.Fprintf(os.Stderr, "Warning: failed to parse %s: %v\n", path, err)
            return nil
        }

        r.agents[agent.Name] = agent
        return nil
    })

    return r.agents, err
}

func (r *Registry) Exists(agentName string) bool {
    _, exists := r.agents[agentName]
    return exists
}

func (r *Registry) Get(agentName string) (*Agent, bool) {
    agent, exists := r.agents[agentName]
    return agent, exists
}

func parseAgentFile(path string) (*Agent, error) {
    content, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }

    // Extract YAML frontmatter between --- markers
    frontmatter, _ := extractFrontmatter(content)
    if frontmatter == nil {
        return nil, fmt.Errorf("no frontmatter found in %s", path)
    }

    var agent Agent
    if err := yaml.Unmarshal(frontmatter, &agent); err != nil {
        return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
    }

    agent.FilePath = path

    if agent.Name == "" {
        return nil, fmt.Errorf("agent name is required")
    }

    return &agent, nil
}

func extractFrontmatter(content []byte) ([]byte, []byte) {
    lines := strings.Split(string(content), "\n")
    if len(lines) < 3 || lines[0] != "---" {
        return nil, content
    }

    // Find closing ---
    for i := 1; i < len(lines); i++ {
        if lines[i] == "---" {
            frontmatter := []byte(strings.Join(lines[1:i], "\n"))
            body := []byte(strings.Join(lines[i+1:], "\n"))
            return frontmatter, body
        }
    }

    return nil, content
}
```

**Key points**:
- Handle missing agents directory gracefully
- Parse YAML frontmatter from .md files
- Store agents in map for fast lookup
- Default to ~/.claude/agents if not specified

**Integration points**:
- Import: `gopkg.in/yaml.v3`
- Will be used by executor to validate agent assignments

### Verification

**Manual testing**:
1. Point to ~/.claude/agents directory
2. Call Discover()
3. Verify agents are found and parsed

**Automated tests**:
```bash
go test ./internal/agent/ -v
```

**Expected output**:
```
=== RUN   TestDiscoverAgents
--- PASS: TestDiscoverAgents (0.00s)
PASS
```

### Commit

**Commit message**:
```
feat: implement agent discovery

- Scan ~/.claude/agents/ directory for agent definitions
- Parse agent metadata from YAML frontmatter
- Store agents in registry map for lookup
- Handle missing directory gracefully
```

**Files to commit**:
- `internal/agent/discovery.go`
- `internal/agent/discovery_test.go`
- `internal/agent/testdata/sample-agent.md`

---

## Task 9: Implement Claude CLI Agent Invocation

**File(s)**: `internal/agent/invoker.go`, `internal/agent/invoker_test.go`
**Depends on**: Task 3, Task 8
**Estimated time**: 1h

### What you're building
Build and execute claude CLI commands with proper flags (--settings, -p, --output-format json), capture output, handle timeouts.

### Test First (TDD)

**Test file**: `internal/agent/invoker_test.go`

**Test structure**:
```go
TestBuildCommand - verify command construction
TestInvokeAgent - test agent invocation (mock claude CLI)
TestTimeout - verify timeout handling
TestOutputCapture - verify stdout/stderr capture
```

**Test specifics**:
- Mock exec.Command for testing
- Test timeout scenarios
- Test error handling
- Capture and parse output

**Example test skeleton**:
```go
package agent

import (
    "context"
    "testing"
    "time"

    "github.com/yourusername/conductor/internal/models"
)

func TestBuildCommand(t *testing.T) {
    task := models.Task{
        Number: 1,
        Name:   "Test Task",
        Prompt: "Do something",
        Agent:  "swiftdev",
        EstimatedTime: 30 * time.Minute,
    }

    invoker := NewInvoker()
    args := invoker.BuildCommandArgs(task)

    // Verify required flags
    hasP := false
    hasSettings := false

    for i, arg := range args {
        if arg == "-p" {
            hasP = true
        }
        if arg == "--settings" && i+1 < len(args) {
            if strings.Contains(args[i+1], "disableAllHooks") {
                hasSettings = true
            }
        }
    }

    if !hasP {
        t.Error("Command should have -p flag")
    }
    if !hasSettings {
        t.Error("Command should have --settings with disableAllHooks")
    }
}

// Test with mocked command execution
func TestInvokeAgentSuccess(t *testing.T) {
    // This test would use a mock or test helper
    // For real testing, consider using a test script that mimics claude CLI
}
```

### Implementation

**Approach**:
Use os/exec to spawn claude CLI subprocess, construct args with required flags, use context.WithTimeout for timeout handling, capture stdout/stderr.

**Code structure**:
```go
// internal/agent/invoker.go
package agent

import (
    "context"
    "encoding/json"
    "fmt"
    "os/exec"
    "strings"
    "time"

    "github.com/yourusername/conductor/internal/models"
)

type Invoker struct {
    ClaudePath string
    Registry   *Registry
}

type InvocationResult struct {
    Output   string
    ExitCode int
    Duration time.Duration
    Error    error
}

func NewInvoker() *Invoker {
    return &Invoker{
        ClaudePath: "claude", // assume in PATH
    }
}

func (inv *Invoker) Invoke(ctx context.Context, task models.Task) (*InvocationResult, error) {
    startTime := time.Now()

    // Build command args
    args := inv.BuildCommandArgs(task)

    // Create command with context (for timeout)
    cmd := exec.CommandContext(ctx, inv.ClaudePath, args...)

    // Capture output
    output, err := cmd.CombinedOutput()

    result := &InvocationResult{
        Output:   string(output),
        Duration: time.Since(startTime),
    }

    if err != nil {
        if exitErr, ok := err.(*exec.ExitError); ok {
            result.ExitCode = exitErr.ExitCode()
        } else {
            result.Error = err
        }
    }

    return result, nil
}

func (inv *Invoker) BuildCommandArgs(task models.Task) []string {
    args := []string{
        "-p", // Print mode (non-interactive)
    }

    // Build prompt with agent reference if specified
    prompt := task.Prompt
    if task.Agent != "" && inv.Registry != nil && inv.Registry.Exists(task.Agent) {
        // Reference agent in prompt
        prompt = fmt.Sprintf("use the %s subagent to: %s", task.Agent, task.Prompt)
    }

    args = append(args, prompt)

    // Disable hooks for automation
    args = append(args, "--settings", `{"disableAllHooks": true}`)

    // JSON output for easier parsing
    args = append(args, "--output-format", "json")

    return args
}

func (inv *Invoker) InvokeWithTimeout(task models.Task, timeout time.Duration) (*InvocationResult, error) {
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()

    return inv.Invoke(ctx, task)
}

// Parse JSON output from claude CLI (if output-format is json)
type ClaudeOutput struct {
    Content string `json:"content"`
    Error   string `json:"error"`
}

func ParseClaudeOutput(output string) (*ClaudeOutput, error) {
    var co ClaudeOutput
    if err := json.Unmarshal([]byte(output), &co); err != nil {
        // If not JSON, return raw output as content
        return &ClaudeOutput{Content: output}, nil
    }
    return &co, nil
}
```

**Key points**:
- Always use `-p` flag for non-interactive mode
- Always use `--settings '{"disableAllHooks": true}'`
- Use context for timeout propagation
- Capture both stdout and stderr
- Parse JSON output if available

**Integration points**:
- Import: `os/exec`, `context`
- Use models.Task
- Will be used by executor

### Verification

**Manual testing**:
1. Create test task
2. Call Invoke() (requires claude in PATH)
3. Verify command is constructed correctly

**Automated tests**:
```bash
go test ./internal/agent/ -run TestBuild -v
```

**Expected output**:
```
=== RUN   TestBuildCommand
--- PASS: TestBuildCommand (0.00s)
PASS
```

### Commit

**Commit message**:
```
feat: implement Claude CLI agent invocation

- Build claude CLI commands with required flags
- Add context-based timeout handling
- Capture stdout/stderr output
- Parse JSON output format
- Support agent-specific prompt construction
```

**Files to commit**:
- `internal/agent/invoker.go`
- `internal/agent/invoker_test.go`

---

_[Due to length, continuing with remaining tasks 10-25 in summary form]_

## Tasks 10-25 Summary

**Task 10**: Implement Quality Control Review Agent (1h)
- Build review prompts with task output
- Parse GREEN/RED responses
- Retry logic with attempt tracking

**Task 11**: Implement File Locking for Plan Updates (45m)
- Use github.com/gofrs/flock
- Atomic file writes with temp file + rename
- Update checkboxes or YAML status fields

**Task 12**: Implement Plan Updater (1h)
- Find and update task checkboxes in Markdown
- Update YAML fields (status, completed_at)
- Handle concurrent updates safely

**Task 13**: Implement Wave Executor (2h)
- Execute waves sequentially
- Spawn goroutines for parallel tasks within wave
- Bounded concurrency with semaphore pattern
- Collect results via channels

**Task 14**: Implement Task Executor (1.5h)
- Execute single task: invoke → review → retry
- Handle RED flags with retry logic
- Update plan file on completion
- Return TaskResult

**Task 15**: Implement Main Orchestration Engine (2h)
- Coordinate wave executor
- Handle graceful shutdown (SIGINT)
- Collect and aggregate results
- Print final summary

**Task 16**: Implement `conductor run` Command (1h)
- Parse flags (--dry-run, --max-concurrency, etc.)
- Load plan file
- Call orchestration engine
- Display progress

**Task 17**: Implement `conductor validate` Command (45m)
- Parse plan file
- Run validation checks
- Display validation report
- Return appropriate exit code

**Task 18**: Implement Console Logger (1h)
- Timestamp-prefixed log messages
- Task start/complete/fail messages
- Wave progress tracking
- Color coding (optional)

**Task 19**: Implement File Logger (45m)
- Create .conductor/logs/ directory
- Write per-run log files
- Write per-task detailed logs
- Create latest.log symlink

**Task 20**: Add Configuration File Support (1h)
- Parse .conductor/config.yaml
- Merge config with CLI flags
- Default values

**Task 21**: Add Error Handling and Recovery (1h)
- Graceful error handling throughout
- Continue-on-error strategy
- Timeout handling
- Resource cleanup

**Task 22**: Add Integration Tests (2h)
- End-to-end test with sample plan
- Test both Markdown and YAML formats
- Test failure scenarios
- Test --dry-run mode

**Task 23**: Add Makefile and Build Script (30m)
- `make build` - compile binary
- `make test` - run all tests
- `make install` - install to PATH
- Cross-compilation targets

**Task 24**: Write README and Documentation (1h)
- Installation instructions
- Usage examples
- Plan format documentation
- Troubleshooting guide

**Task 25**: Final Integration and Testing (2h)
- Run full orchestration with real plans
- Test error scenarios
- Performance testing
- Bug fixes and polish

---

## Testing Strategy

### Unit Tests
- **Location**: `*_test.go` files alongside implementation
- **Naming**: `TestFunctionName` for functions, `TestTypeName_MethodName` for methods
- **Run command**: `go test ./...`
- **Coverage target**: 70%

### Integration Tests
- **Location**: `test/integration/`
- **What to test**: Full orchestration flows with sample plans
- **Setup required**: Sample plan files, mock agent responses
- **Run command**: `go test ./test/integration/...`

### Test Design Principles

**Use these patterns**:
1. Table-driven tests for multiple scenarios
2. Subtests with t.Run() for organization
3. Test fixtures in testdata/ directories

**Avoid these anti-patterns**:
1. Testing implementation details (test behavior, not internals)
2. Brittle tests that break on refactoring
3. Tests with external dependencies (use mocks)

**Mocking guidelines**:
- Mock external claude CLI (use test scripts)
- Mock file system when appropriate
- Don't mock internal functions

---

## Commit Strategy

Break this work into **25+ commits** following TDD sequence:

1. **test**: Add tests for Go module initialization
2. **feat**: Initialize Go module and project structure
3. **test**: Add tests for cobra CLI setup
4. **feat**: Add cobra CLI framework
5. **test**: Add tests for data models
6. **feat**: Define core data models
... (continue for each task)

**Commit message format**:
```
type: brief description

Optional body explaining why
```

Types: `feat`, `test`, `fix`, `refactor`, `docs`, `chore`

---

## Common Pitfalls & How to Avoid Them

1. **Goroutine leaks**
   - Why: Forgetting to close channels or wait for goroutines
   - How to avoid: Use sync.WaitGroup, defer close(), context cancellation
   - Reference: Standard Go concurrency patterns

2. **Race conditions in plan updates**
   - Why: Multiple goroutines writing to plan file
   - How to avoid: Use flock file locking
   - Reference: github.com/gofrs/flock examples

3. **Timeout not propagating**
   - Why: Not passing context through call chain
   - How to avoid: Always pass context.Context as first parameter
   - Reference: Go context package documentation

4. **Circular dependencies not detected**
   - Why: Incorrect graph traversal
   - How to avoid: Implement proper DFS with color marking
   - Reference: Task 7 graph implementation

---

## Resources & References

### Go Resources
- Go Project Layout: https://github.com/golang-standards/project-layout
- Cobra CLI: https://github.com/spf13/cobra
- Effective Go: https://go.dev/doc/effective_go

### Libraries to Use
- `github.com/spf13/cobra` - CLI framework
- `github.com/yuin/goldmark` - Markdown parsing
- `gopkg.in/yaml.v3` - YAML parsing
- `github.com/gofrs/flock` - File locking

### Validation Checklist
- [ ] All tests pass (`go test ./...`)
- [ ] Golint passes (`golangci-lint run`)
- [ ] Formatted correctly (`gofmt -w .`)
- [ ] No race conditions (`go test -race ./...`)
- [ ] Build succeeds (`go build ./cmd/conductor`)
- [ ] Binary works (`./conductor --help`)

---

**Total Estimated Time**: ~25-30 hours
**Recommended Approach**: Follow TDD strictly - write tests first, then implement

