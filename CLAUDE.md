# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Conductor is an autonomous multi-agent orchestration CLI built in Go that executes implementation plans by spawning and managing multiple Claude Code CLI agents in coordinated waves. It parses plan files (Markdown or YAML), calculates task dependencies using graph algorithms, and orchestrates parallel execution with quality control reviews and adaptive learning.

**Current Status**: Production-ready v2.3.0 with comprehensive multi-agent orchestration, multi-file plan support, quality control reviews, adaptive learning system, inter-retry agent swapping, structured success criteria with per-criterion verification, and auto-incrementing version management.

## Development Commands

### Build & Run
```bash
# Build the binary
go build ./cmd/conductor

# Run from source
go run ./cmd/conductor

# Show version
./conductor --version

# Show help
./conductor --help

# Learning commands
./conductor learning stats        # View learning statistics
./conductor learning export       # Export learning data
./conductor learning clear        # Clear learning history
```

### Testing
```bash
# Run all tests
go test ./...

# Run all tests with coverage
go test ./... -cover

# Run tests for specific package
go test ./internal/parser/ -v
go test ./internal/executor/ -v
go test ./internal/agent/ -v
go test ./internal/learning/ -v

# Run specific test
go test ./internal/parser/ -run TestMarkdownParser -v

# Run with race detection
go test -race ./...

# Generate coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Code Quality
```bash
# Format code
go fmt ./...

# Run linter (if installed)
golangci-lint run

# Check for issues
go vet ./...
```

### Version Management
```bash
# Current version (stored in VERSION file)
cat VERSION

# Build with current version
make build

# Auto-increment patch version (1.1.0 → 1.1.1) and build
make build-patch

# Auto-increment minor version (1.1.0 → 1.2.0) and build
make build-minor

# Auto-increment major version (1.1.0 → 2.0.0) and build
make build-major
```

**Versioning**: Conductor uses semantic versioning with auto-increment targets. The VERSION file is the single source of truth, automatically injected into the binary at build time via LDFLAGS.

## Architecture Overview

### High-Level Flow

**Single-File Plans:**
```
Plan File (.md/.yaml)
  → Parser (auto-detects format)
  → Dependency Graph Builder (validates, detects cycles)
  → Wave Calculator (Kahn's algorithm for topological sort)
  → Orchestrator (coordinates execution)
  → Wave Executor (spawns parallel tasks)
  → Pre-Task Hook (loads failure history, injects context)
  → Task Executor (invokes claude CLI, handles retries)
  → Quality Control (reviews output, decides GREEN/RED/YELLOW)
  → Post-Task Hook (stores success/failure patterns)
  → Plan Updater (marks tasks complete)
  → Learning System (analyzes patterns, generates insights)
```

**Multi-File Plans:**
```
Multiple Plan Files (.md/.yaml)
  → Parser (auto-detects format per file)
  → Plan Merger (validates, deduplicates, merges)
  → Dependency Graph Builder (cross-file dependencies)
  → Wave Calculator (respects worktree groups)
  → Orchestrator (maintains file origins)
  → Pre-Task Hook (loads failure history, injects context)
  → Execution & Logging (file-aware tracking)
  → Post-Task Hook (stores success/failure patterns)
  → Learning System (cross-run pattern analysis)
```

### Key Components

**Parser (`internal/parser/`)**: Auto-detects file format and parses plan files into Task structs. Markdown parser extracts tasks from `## Task N:` headings. YAML parser handles structured format. Both support optional YAML frontmatter for conductor config.

**Plan Merger** (multi-file support):
- Loads and merges multiple plan files (.md or .yaml)
- Auto-detects format per file
- Validates cross-file dependencies
- Tracks file-to-task mapping for resume operations
- Maintains worktree group organization

**Models (`internal/models/`)**: Core data structures (Task, Plan, Wave, TaskResult). Task includes Status and CompletedAt fields for resumable execution. Task.DependsOn creates dependency graph. Wave groups parallel-executable tasks. Plan includes WorktreeGroups and FileToTaskMap for multi-file coordination. Helper methods: IsCompleted(), CanSkip().

**Executor (`internal/executor/`)**:
- `graph.go`: Dependency graph builder using Kahn's algorithm for wave calculation. DFS with color marking for cycle detection. Validates all dependencies exist.
- `qc.go`: Quality control reviews using dedicated agent. Parses GREEN/RED/YELLOW flags from JSON or legacy text output. Uses `parseQCJSON()` with Claude CLI envelope extraction. Handles retry logic (max 2 attempts on RED).
- `task.go`: Task executor implementing invoke → review → retry pipeline. Handles GREEN/RED/YELLOW flags, retries on RED, updates plan file on completion. Inter-retry agent swapping via `learning.SelectBetterAgent()`. Single `updateFeedback()` call after QC review (not before). 84.5% test coverage with comprehensive error path testing.
- `wave.go`: Wave executor for sequential wave execution with bounded concurrency. Filters completed tasks before execution based on SkipCompleted config. Creates synthetic GREEN results for skipped tasks. Respects RetryFailed flag for failed task re-execution.

**Agent (`internal/agent/`)**:
- `discovery.go`: Scans `~/.claude/agents/` for agent .md files with YAML frontmatter. Uses directory whitelisting (root + numbered dirs 01-10) and file filtering (skips README.md, *-framework.md, examples/, transcripts/, logs/) to eliminate false warnings. Registry provides fast agent lookup with ~94% test coverage.
- `invoker.go`: Executes `claude -p` commands with proper flags. Always disables hooks, uses JSON output format. Prefixes prompts with "use the {agent} subagent to:" when agent specified.

**Learning (`internal/learning/`)**:
- `store.go`: Persistent storage for task execution history. Stores success/failure outcomes, error messages, file changes, and timestamps. JSON-based storage with file-per-task organization in `.conductor/learning/` directory.
- `analyzer.go`: Pattern recognition engine that analyzes historical task executions. Identifies common failure patterns, success strategies, and file-based correlations. Generates actionable insights for future task execution.
- `hooks.go`: Pre-task and post-task hooks that integrate learning into the execution pipeline. Pre-task hook loads relevant failure history and injects context into task prompts. Post-task hook stores execution outcomes for future analysis.
- Architecture: Learning data stored in `.conductor/learning/{task-name}-{timestamp}.json`. Each entry contains task metadata, execution outcome, duration, files modified, and detailed error information for failures.

**CLI (`internal/cmd/`)**:
- `root.go`: Root command with version display and subcommand registration
- `run.go`: Implements `conductor run` command for plan execution
  - CLI flags: --dry-run, --max-concurrency, --timeout, --verbose, --skip-completed, --retry-failed, --log-dir
  - Config file merging (CLI flags override config.yaml)
  - Plan file loading with auto-format detection
  - Orchestrator integration with progress logging
  - Comprehensive error handling
- `validate.go`: Implements `conductor validate` command for plan validation
- `learning.go`: Implements `conductor learning` command group
  - `learning export`: Export learning data to JSON for analysis
  - `learning stats`: Display learning statistics and insights
  - `learning clear`: Clear learning history (with confirmation)
- Console logger: Real-time progress updates with timestamps

**Config (`internal/config/`)**:
- Supports .conductor/config.yaml for default settings
- Configuration priority: defaults → config file → CLI flags (highest priority)
- Fields: max_concurrency, timeout, dry_run, skip_completed, retry_failed, log_dir, log_level
- Learning configuration: learning_enabled (default: true), learning_dir (default: .conductor/learning), max_history_entries (default: 100)

### Dependency Graph Algorithm

Uses **Kahn's algorithm** for topological sort with wave grouping:
1. Calculate in-degree for each task (number of dependencies)
2. Wave 1: All tasks with in-degree 0 (no dependencies)
3. Process wave, decrease in-degree of dependent tasks
4. Wave N: Tasks that now have in-degree 0
5. Repeat until all tasks processed

Cycle detection uses **DFS with color marking**:
- White (0): Not visited
- Gray (1): Currently visiting (on recursion stack)
- Black (2): Fully visited
- Back edge (gray → gray) = cycle detected

### Claude CLI Invocation Pattern

All agent invocations follow this structure:
```go
args := []string{
    "-p",  // Non-interactive print mode
    prompt,
    "--settings", `{"disableAllHooks": true}`,  // No user hooks
    "--output-format", "json",  // Structured output
}
cmd := exec.CommandContext(ctx, "claude", args...)
```

Agent references are added by prefixing prompt:
```go
prompt = fmt.Sprintf("use the %s subagent to: %s", agent, task.Prompt)
```

### Quality Control Pattern

QC reviews use structured JSON output format with envelope extraction:

**JSON Response Schema (`internal/models/response.go`):**
```go
type QCResponse struct {
    Verdict         string   `json:"verdict"`          // "GREEN", "RED", "YELLOW"
    Feedback        string   `json:"feedback"`         // Detailed review feedback
    Issues          []Issue  `json:"issues"`           // Specific issues found
    Recommendations []string `json:"recommendations"`  // Suggested improvements
    ShouldRetry     bool     `json:"should_retry"`     // Whether to retry
    SuggestedAgent  string   `json:"suggested_agent"`  // Alternative agent suggestion
}

type Issue struct {
    Severity    string `json:"severity"`    // "critical", "warning", "info"
    Description string `json:"description"` // Issue description
    Location    string `json:"location"`    // File:line or component
}
```

**QC JSON Parsing (`internal/executor/qc.go`):**
```go
// parseQCJSON extracts and validates QC response from Claude CLI output
func parseQCJSON(output string) (*models.QCResponse, error) {
    // Step 1: Extract "result" field from Claude CLI envelope
    claudeOut, _ := agent.ParseClaudeOutput(output)
    actualOutput := claudeOut.Content

    // Step 2: Strip markdown code fences if present (```json...```)
    if strings.HasPrefix(strings.TrimSpace(actualOutput), "```") {
        actualOutput = extractJSONFromCodeFence(actualOutput)
    }

    // Step 3: Parse and validate JSON
    var resp models.QCResponse
    json.Unmarshal([]byte(actualOutput), &resp)
    resp.Validate()
    return &resp, nil
}
```

Claude CLI returns wrapped JSON: `{"type":"result","result":"..."}`. The `agent.ParseClaudeOutput()` function extracts the inner `result` field, then `extractJSONFromCodeFence()` strips any markdown wrappers before JSON parsing.

**Dual Feedback Storage (`internal/executor/task.go`):**
- `updateFeedback()` called ONLY after QC review completes (line ~647)
- Stores: attempt number, agent, verdict, agent output, QC feedback, timestamp
- Uses `updater.UpdateTaskFeedback()` to write execution history to plan files
- Eliminates duplicate entries by avoiding pre-QC storage

```go
// Called once per attempt, after QC review is complete
te.updateFeedback(task, attempt+1, invocation.Output, qcFeedback, verdict)
```

**Inter-Retry Agent Swapping (v2.1):**
Quality control can suggest alternative agents via `SuggestedAgent` field. When QC returns RED verdict with a suggested agent, conductor uses `learning.SelectBetterAgent()` to determine the optimal agent for retry:

```go
// After RED verdict, check for agent swap opportunity
if newAgent, reason := learning.SelectBetterAgent(task.Agent, history, review.SuggestedAgent);
   newAgent != "" && newAgent != task.Agent {
    task.Agent = newAgent  // Swap agent for next retry
}
```

Flow: Task fails (RED) → QC suggests better agent → `SelectBetterAgent()` validates → Auto-swap on retry → Enhanced success rate

**Retry Logic:**
- Only retry on RED verdict
- Up to MaxRetries (default: 2)
- Agent swapping occurs between retries when suggested
- Controlled by `learning.swap_during_retries` config (default: true)
- Respects `min_failures_before_adapt` threshold from config

### Structured Success Criteria (v2.3+)

Tasks can define explicit success criteria for per-criterion QC verification:

```yaml
tasks:
  - id: 5
    name: "Add JWT Authentication"
    files: [internal/auth/jwt.go]
    success_criteria:
      - "JWT validation function implemented"
      - "Supports HS256 algorithm"
      - "Unit tests achieve 90% coverage"
    test_commands:
      - "go test ./internal/auth/ -v"
```

**QC Verification Process:**
1. Criteria numbered in prompt (0, 1, 2...)
2. QC agents return per-criterion PASS/FAIL with evidence
3. Multi-agent consensus: criterion passes only if ALL agents agree
4. Any failed criterion = RED verdict (unanimous consensus required)

**Response Schema (`internal/models/response.go`):**
```go
type QCResponse struct {
    Verdict         string             `json:"verdict"`
    Feedback        string             `json:"feedback"`
    CriteriaResults []CriterionResult  `json:"criteria_results"`  // Per-criterion verdicts
    Issues          []Issue            `json:"issues"`
    Recommendations []string           `json:"recommendations"`
    ShouldRetry     bool               `json:"should_retry"`
    SuggestedAgent  string             `json:"suggested_agent"`
}

type CriterionResult struct {
    Index    int    `json:"index"`     // Criterion index (0-based)
    Passed   bool   `json:"passed"`    // PASS/FAIL
    Evidence string `json:"evidence"`  // Evidence or reason
}
```

**Multi-Agent QC (`internal/executor/qc.go`):**
```go
// MergeQCResponses combines results from multiple QC agents
// using unanimous consensus (all must agree for PASS)
func MergeQCResponses(responses []*models.QCResponse) *models.QCResponse {
    // Unanimous consensus: criterion passes only if ALL agents mark PASS
    // Any disagreement = FAIL
}
```

**Backward Compatible**: Tasks without `success_criteria` use legacy blob review (unstructured output assessment).

### Adaptive Learning System

Conductor learns from task execution history to improve future runs:

**Learning Flow:**
1. **Pre-Task Hook**: Before executing a task, load historical failures for similar tasks
2. **Context Injection**: Augment task prompt with relevant failure patterns and solutions
3. **Task Execution**: Execute task with enhanced context
4. **Post-Task Hook**: Store execution outcome (success/failure, duration, files, errors)
5. **Pattern Analysis**: Analyze stored history to identify trends and insights

**Learning Data Structure:**
```json
{
  "task_name": "Task 5: Implement error handling",
  "task_number": "5",
  "timestamp": "2025-01-12T10:30:00Z",
  "outcome": "failed",
  "duration_seconds": 45.2,
  "files_modified": ["internal/executor/task.go"],
  "error_message": "compilation error: undefined variable",
  "retry_count": 1
}
```

**Pattern Recognition:**
- Failure clustering: Group similar failures by error message, files, task type
- Success strategies: Identify patterns in successful task executions
- File correlations: Track which files commonly fail together
- Time analysis: Detect tasks that consistently take longer than estimated

**Learning Commands:**
```bash
# Export all learning data for external analysis
conductor learning export > analysis.json

# View statistics and insights
conductor learning stats

# Clear learning history (with confirmation)
conductor learning clear
```

**Configuration:**
Learning can be configured in `.conductor/config.yaml`:
```yaml
learning:
  enabled: true                    # Enable/disable learning system
  db_path: .conductor/learning/executions.db  # SQLite database path
  auto_adapt_agent: true          # Enable agent adaptation based on patterns
  swap_during_retries: true       # v2.1: swap agents within same run (not just between runs)
  enhance_prompts: true           # Add learned context to prompts
  qc_reads_plan_context: true     # v2.1: QC loads execution history from plan file
  qc_reads_db_context: true       # v2.1: QC loads execution history from database
  max_context_entries: 10         # v2.1: limit context to last N attempts
  min_failures_before_adapt: 2    # Failure threshold before adapting
  max_executions_per_task: 100    # Max entries per task in database
  keep_executions_days: 90        # Data retention period
```

**Hook Integration:**
Hooks are automatically invoked during task execution:
- Pre-task: `PreTaskHook(task, store)` - injects failure context
- Post-task: `PostTaskHook(task, result, store)` - stores outcome

**QC Context Loading (`internal/executor/qc.go`):**
```go
// LoadContext loads execution history from database for the task
func (qc *QualityController) LoadContext(ctx context.Context, task models.Task, store *learning.Store) (string, error) {
    history, err := store.GetExecutionHistory(ctx, task.SourceFile, task.Number)
    // Formats historical attempts with verdicts, feedback, and errors
    // for QC agent to make informed retry recommendations
}
```

## Test-Driven Development

**This project follows strict TDD**:
1. **Red**: Write failing test first
2. **Green**: Implement minimal code to pass
3. **Refactor**: Improve code quality
4. **Commit**: Commit after each completed task

### Test Organization
- Test files live alongside source: `foo.go` → `foo_test.go`
- Test fixtures in `internal/{package}/testdata/`
- Table-driven tests using `t.Run()` for subtests
- Interfaces enable mocking (e.g., `InvokerInterface` for QC tests)

### Coverage Targets
- Critical paths (graph algorithms, QC logic): 90%+
- Overall target: 70%+ (currently 86.4% - exceeds target)

## Important Implementation Details

### File Format Detection
Extension-based auto-detection in `parser.DetectFormat()`:
- `.md`, `.markdown` → Markdown parser
- `.yaml`, `.yml` → YAML parser
- Convenience function `ParseFile()` handles detection + parsing

### Task Metadata Parsing

Conductor's parsers (Markdown and YAML) extract these fields from plan files:

**Core fields** (always parsed):
- `Number` - Task identifier (string)
- `Name` - Task title
- `Files` - List of files the task modifies
- `DependsOn` - Task dependencies for wave calculation
- `EstimatedTime` - Duration estimate (parsed as time.Duration)
- `Agent` - Claude agent to execute the task
- `Status` - Task completion status (completed, failed, in-progress, pending)
- `CompletedAt` - Timestamp when task was completed
- `SuccessCriteria` - List of explicit success criteria for QC verification (v2.3+)
- `TestCommands` - Commands to run for verification (v2.3+)

**Multi-file plan fields**:
- `WorktreeGroup` - Group assignment for task organization
  - Parsed by both Markdown parser (`**WorktreeGroup**: value`) and YAML parser (`worktree_group: value`)
  - Stored in Task.WorktreeGroup field
  - Used for organizational purposes and multi-file plan coordination
  - **Not used for execution control** - conductor doesn't enforce isolation or execution order based on groups
  - Groups are tracked in Wave.GroupInfo for logging and analysis

**Extended task sections** (for human developers):
- `test_first`, `implementation`, `verification`, `commit` sections in YAML
- These sections are included in Task.Prompt as markdown but not parsed as structured fields
- Provide detailed guidance for engineers implementing the tasks
- Generated by `/doc` and `/doc-yaml` commands for comprehensive task specifications

### Agent Discovery
Registry scans `~/.claude/agents/` on initialization:
```go
registry := agent.NewRegistry("") // Uses ~/.claude/agents default
agents, _ := registry.Discover()
if registry.Exists("quality-control") {
    // Agent available
}
```

### Context-Based Timeouts
All agent invocations accept context for cancellation:
```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()
result, err := invoker.Invoke(ctx, task)
```

## Production Status

Conductor v2.1.0 is production-ready with 86.4% test coverage (465+ tests passing). Complete pipeline: parsing → validation → dependency analysis → orchestration → execution → quality control → adaptive learning → logging. All core features, multi-file plan support, adaptive learning system, inter-retry agent swapping, and auto-incrementing version management implemented and tested.

**Latest coverage run (2025-11-17, `go test -cover ./...`):**

| Package | Coverage |
| --- | --- |
| `internal/config` | 86.3% |
| `internal/executor` | 90.0% |
| `internal/agent` | 91.3% |
| `internal/learning` | 90.6% |
| `internal/logger` | 71.6% |
| `internal/models` | 100.0% |
| `internal/parser` | 74.6% |
| `internal/cmd` | 76.9% |
| `internal/filelock` | 83.1% |
| `internal/fileutil` | 90.9% |
| `internal/display` | 97.0% |
| `internal/updater` | 89.8% |
| `cmd/conductor` | 0.0% (wrapper delegates to internal packages) |
| `test/integration` | No statements (harness package used for E2E tests) |

This run also verified the newly added configuration matrix tests and end-to-end scenarios (happy/failed/mixed verdicts plus skip/resume and structured success-criteria flows).

**Major Features:**
- Multi-agent orchestration with dependency resolution
- Quality control reviews with structured JSON responses
- Inter-retry agent swapping based on QC recommendations
- Adaptive learning from execution history
- Multi-file plan support with cross-file dependencies
- Resumable execution with state tracking
- Dual feedback storage (plan files + database)
- Comprehensive logging and error handling

**v2.1 Enhancements:**
- QC JSON parsing with Claude CLI envelope extraction
- `extractJSONFromCodeFence()` strips markdown wrappers
- Single `updateFeedback()` call eliminates duplicate history entries
- `learning.SelectBetterAgent()` for intelligent agent swapping
- QC loads historical context via `LoadContext()` for informed retry decisions

## Multi-File Plan Examples

### Example 1: Split Backend Plan

Part 1 (setup.md):
```markdown
## Task 1: Database Setup
**Files**: infrastructure/db.tf
**Depends on**: None

Initialize PostgreSQL and migrations.

## Task 2: API Server Setup
**Files**: cmd/api/main.go
**Depends on**: Task 1
**WorktreeGroup**: backend-core

Set up web server framework.
```

Part 2 (features.md):
```markdown
## Task 3: Auth Service
**Files**: internal/auth/auth.go
**Depends on**: Task 1, Task 2
**WorktreeGroup**: backend-features

Implement JWT authentication.

## Task 4: User API
**Files**: internal/api/users.go
**Depends on**: Task 3
**WorktreeGroup**: backend-features

Create user CRUD endpoints.
```

Execute together:
```bash
conductor run setup.md features.md --max-concurrency 3
```

Or validate first:
```bash
conductor validate setup.md features.md
```

### Example 2: Microservices Plan

Three separate plans:
- `auth-service-plan.md` - Authentication service (6 tasks)
- `api-service-plan.md` - Main API (8 tasks)
- `deployment-plan.md` - Infrastructure & deployment (5 tasks)

With worktree groups:
- `auth-service` group → sequential execution (isolation: strong)
- `api-service` group → parallel where possible (isolation: weak)
- `deployment` group → sequential (isolation: strong)

## Module Path

The Go module uses path: `github.com/harrison/conductor`

When importing internal packages:
```go
import (
    "github.com/harrison/conductor/internal/parser"
    "github.com/harrison/conductor/internal/executor"
    "github.com/harrison/conductor/internal/agent"
    "github.com/harrison/conductor/internal/models"
    "github.com/harrison/conductor/internal/learning"
)
```

## Dependencies

Minimal external dependencies:
- `github.com/spf13/cobra` - CLI framework
- `github.com/yuin/goldmark` - Markdown parsing
- `gopkg.in/yaml.v3` - YAML parsing

No networking, no ML frameworks, no web frameworks. Just local CLI orchestration.

## Best Practices

### Single-File Plans
- Keep tasks focused and well-scoped
- Use clear dependency declarations
- Test with --dry-run first
- Leverage --skip-completed for resume

### Multi-File Plans

**File Organization**
- 1 file per feature/module/service
- Aim for 5-20 tasks per file
- Clear, descriptive filenames

**Cross-File Dependencies**
- Keep intra-file dependencies tight
- Minimize cross-file dependencies where possible
- Document file execution order if strict

**Worktree Groups**
- Use groups for execution control
- `sequential` for state-dependent tasks
- `parallel` for independent tasks
- `strong` isolation for infrastructure tasks

**Task Naming**
- Prefix with module name: "Backend: Setup DB"
- Clear task boundaries
- File mappings should be obvious

**Testing Split Plans**
- Validate all files together: `conductor validate *.md`
- Dry-run before actual execution
- Check dependency graph for cross-file links
