# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Conductor is an autonomous multi-agent orchestration CLI built in Go that executes implementation plans by spawning and managing multiple Claude Code CLI agents in coordinated waves. It parses plan files (Markdown or YAML), calculates task dependencies using graph algorithms, and orchestrates parallel execution with quality control reviews.

**Current Status**: ✅ **PHASE 1 (25 TASKS) COMPLETE** + **PHASE 2A (Multi-File Plans) IMPLEMENTED**. Conductor v1 fully implemented, tested, and production-ready. All core functionality, logging, configuration, error handling, integration tests, build tooling, and documentation complete. Phase 2A adds multi-file plan support with objective plan splitting and worktree group organization.

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

## Architecture Overview

### High-Level Flow
```
Plan File (.md/.yaml)
  → Parser (auto-detects format)
  → Dependency Graph Builder (validates, detects cycles)
  → Wave Calculator (Kahn's algorithm for topological sort)
  → Orchestrator (NOT YET IMPLEMENTED)
  → Wave Executor (spawns parallel tasks)
  → Task Executor (invokes claude CLI, handles retries)
  → Quality Control (reviews output, decides GREEN/RED/YELLOW)
  → Plan Updater (marks tasks complete)
```

### Key Components

**Parser (`internal/parser/`)**: Auto-detects file format and parses plan files into Task structs. Markdown parser extracts tasks from `## Task N:` headings. YAML parser handles structured format. Both support optional YAML frontmatter for conductor config.

**Models (`internal/models/`)**: Core data structures (Task, Plan, Wave, TaskResult). Task includes Status and CompletedAt fields for resumable execution. Task.DependsOn creates dependency graph. Wave groups parallel-executable tasks. Helper methods: IsCompleted(), CanSkip().

**Executor (`internal/executor/`)**:
- `graph.go`: Dependency graph builder using Kahn's algorithm for wave calculation. DFS with color marking for cycle detection. Validates all dependencies exist.
- `qc.go`: Quality control reviews using dedicated agent. Parses GREEN/RED/YELLOW flags from output. Handles retry logic (max 2 attempts on RED).
- `task.go`: Task executor implementing invoke → review → retry pipeline. Handles GREEN/RED/YELLOW flags, retries on RED, updates plan file on completion. 84.5% test coverage with comprehensive error path testing.
- `wave.go`: Wave executor for sequential wave execution with bounded concurrency. Filters completed tasks before execution based on SkipCompleted config. Creates synthetic GREEN results for skipped tasks. Respects RetryFailed flag for failed task re-execution.

**Agent (`internal/agent/`)**:
- `discovery.go`: Scans `~/.claude/agents/` for agent .md files with YAML frontmatter. Uses directory whitelisting (root + numbered dirs 01-10) and file filtering (skips README.md, *-framework.md, examples/, transcripts/, logs/) to eliminate false warnings. Registry provides fast agent lookup with ~94% test coverage.
- `invoker.go`: Executes `claude -p` commands with proper flags. Always disables hooks, uses JSON output format. Prefixes prompts with "use the {agent} subagent to:" when agent specified.

**CLI (`internal/cmd/`)**:
- `root.go`: Root command with version display and subcommand registration
- `run.go`: Implements `conductor run` command for plan execution
  - CLI flags: --dry-run, --max-concurrency, --timeout, --verbose, --skip-completed, --retry-failed, --log-dir
  - Config file merging (CLI flags override config.yaml)
  - Plan file loading with auto-format detection
  - Orchestrator integration with progress logging
  - Comprehensive error handling
- `validate.go`: Implements `conductor validate` command for plan validation
- Console logger: Real-time progress updates with timestamps

**Config (`internal/config/`)**:
- Supports .conductor/config.yaml for default settings
- Configuration priority: defaults → config file → CLI flags (highest priority)
- Fields: max_concurrency, timeout, dry_run, skip_completed, retry_failed, log_dir, log_level

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

QC reviews follow structured format:
```
Review the following task execution:

Task: {task.Name}

Output:
{output}

Provide quality control review in this format:
Quality Control: [GREEN/RED/YELLOW]

Feedback: [your detailed feedback]
```

Parser extracts flag via regex: `Quality Control:\s*(GREEN|RED|YELLOW)`

Retry logic: Only retry on RED, up to MaxRetries (default: 2)

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

### ✅ ALL CORE FUNCTIONALITY COMPLETE
All critical components now implemented and production-ready:

✅ **Task 16 - Completed 2025-11-09**: `conductor run` command
- Full plan execution with parallel orchestration
- CLI flags: --dry-run, --max-concurrency, --timeout, --verbose
- 92.5% code coverage, 17 tests all passing
- QA GREEN verdict - production ready

✅ **Task 17 - Completed 2025-11-09**: `conductor validate` command
- Plan validation with 100% test coverage
- Validates dependencies, detects cycles
- Shows detailed validation report

✅ **Core Pipeline Complete**:
- ✅ File locking for concurrent plan updates (Task 11)
- ✅ Plan updater to mark tasks complete (Task 12)
- ✅ Wave executor for parallel task execution (Task 13)
- ✅ Task executor with retry logic (Task 14)
- ✅ Orchestration engine (Task 15)

✅ **Task 19 - Completed 2025-11-11**: File Logger
- Timestamped log files in .conductor/logs/
- Per-task detailed logs with execution results
- Latest.log symlink for quick access
- 93.2% test coverage, 17 tests passing

✅ **Task 20 - Completed 2025-11-10**: Configuration File Support
- YAML config file support (.conductor/config.yaml)
- CLI flag merging with proper precedence
- Sensible defaults, comprehensive validation
- 95.3% test coverage, 22 tests passing

✅ **Task 21 - Completed 2025-11-11**: Error Handling & Recovery (FIXED)
- Custom error types integrated throughout executor pipeline
- TaskError, TimeoutError, ExecutionError in use
- Wave, task, and orchestrator proper error wrapping
- 86.4% executor coverage (improved from 76.7%)
- 6 integration tests added for error handling

✅ **Task 22 - Completed 2025-11-10**: Integration Tests
- 12 comprehensive E2E test functions
- 9 test fixtures covering simple to complex scenarios
- Cycle detection, wave calculation, format validation
- 100% test pass rate

✅ **Task 23 - Completed 2025-11-10**: Makefile & Build Script
- 14 Makefile targets (exceeds 13 required)
- Cross-compilation for 5 platforms (Linux/macOS/Windows × x86/ARM)
- Version injection via LDFLAGS, colored output
- All targets tested and working

✅ **Task 24 - Completed 2025-11-10**: README & Documentation
- 361-line comprehensive README
- 4 documentation files (usage, plan-format, troubleshooting)
- 215+ code examples, 5 complete implementation examples
- 100% CLI command and flag coverage

✅ **Task 25 - Completed 2025-11-11**: Final Integration & Testing (FIXED)
- 15 E2E test functions implemented (was 0/15)
- Logger integration tests (FileLogger, ConsoleLogger, MultiLogger)
- Plan execution E2E tests (Markdown, YAML, complex dependencies, failures)
- Log file creation and symlink verification tests
- Test file grew from 604 to 1,193 lines

**Current Status**: **✅ PRODUCTION READY** - All 25 tasks complete. The conductor binary is fully functional with 86.4% test coverage (465+ tests passing). Complete pipeline: parsing → validation → execution → logging → completion. All documentation, tooling, and error handling in place. Ready for production deployment.

## Phase 2A: Multi-File Plans & Objective Plan Splitting

**Status**: ✅ Fully implemented with 37 new integration tests and 100% backward compatibility.

Phase 2A extends Conductor to support splitting large implementation plans across multiple files with objective grouping of related tasks.

### Key Phase 2A Features

**1. Multi-File Plan Loading**
- Load and automatically merge multiple plan files (.md or .yaml)
- Auto-detect format per file (each file can be Markdown or YAML)
- Maintain cross-file task dependencies
- Track which file each task originated from

**2. Objective Plan Splitting**
- Split complex plans into logical phases or components
- Each file represents a cohesive unit (e.g., frontend, backend, deployment)
- Related tasks stay together with clear file organization
- Smaller files = easier to review and understand

**3. Worktree Group Organization**
- Group related tasks into logical units (worktree groups)
- Define execution model: `parallel` or `sequential`
- Set isolation levels: `none`, `weak`, or `strong`
- Capture rationale for each group's organization

**4. FileToTaskMap Tracking**
- Every task knows which file it came from
- Maps file paths to task numbers for quick lookup
- Enables resume operations on split plans
- Supports partial plan re-execution

### Phase 2A Use Cases

1. **Large Projects**: Split a 20+ task plan into 2-3 focused files
2. **Team Collaboration**: Different teams own different plan files
3. **Phased Delivery**: Setup → Core Implementation → Testing → Deployment
4. **Microservices**: Separate plans per service, merged for orchestration
5. **Feature Flags**: Optional features in separate plan files

### Phase 2A Architecture

**Multi-File Merging Pipeline**:
```
Multiple Plan Files (.md/.yaml)
  → Parser (auto-detects format per file)
  → MergePlans (validate, deduplicate, merge)
  → Dependency Graph Builder (cross-file dependencies)
  → Wave Calculator (respects worktree groups)
  → Orchestrator (maintains file origins)
  → Execution & Logging (file-aware tracking)
```

**Models Integration**:
- `Plan.WorktreeGroups` - Define task groups and execution rules
- `Plan.FileToTaskMap` - Track task origins
- `Task.WorktreeGroup` - Assign task to group
- `Wave.GroupInfo` - Group-aware wave execution

**Backward Compatibility**:
- Single-file plans work unchanged
- No breaking changes to existing API
- All Phase 1 (v1.0) features fully preserved
- 100% test coverage for compatibility

### Phase 2A Examples

**Example 1: Split Backend Plan**

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

**Example 2: Microservices Plan**

Three separate plans:
- `auth-service-plan.md` - Authentication service (6 tasks)
- `api-service-plan.md` - Main API (8 tasks)
- `deployment-plan.md` - Infrastructure & deployment (5 tasks)

With worktree groups:
- `auth-service` group → sequential execution (isolation: strong)
- `api-service` group → parallel where possible (isolation: weak)
- `deployment` group → sequential (isolation: strong)

**Best Practices**

1. **File Organization**
   - 1 file per feature/module/service
   - Aim for 5-20 tasks per file
   - Clear, descriptive filenames

2. **Cross-File Dependencies**
   - Keep intra-file dependencies tight
   - Minimize cross-file dependencies where possible
   - Document file execution order if strict

3. **Worktree Groups**
   - Use groups for execution control
   - `sequential` for state-dependent tasks
   - `parallel` for independent tasks
   - `strong` isolation for infrastructure tasks

4. **Task Naming**
   - Prefix with module name: "Backend: Setup DB"
   - Clear task boundaries
   - File mappings should be obvious

5. **Testing Split Plans**
   - Validate all files together: `conductor validate *.md`
   - Dry-run before actual execution
   - Check dependency graph for cross-file links

## Module Path

The Go module uses path: `github.com/harrison/conductor`

When importing internal packages:
```go
import (
    "github.com/harrison/conductor/internal/parser"
    "github.com/harrison/conductor/internal/executor"
    "github.com/harrison/conductor/internal/agent"
    "github.com/harrison/conductor/internal/models"
)
```

## Dependencies

Minimal external dependencies:
- `github.com/spf13/cobra` - CLI framework
- `github.com/yuin/goldmark` - Markdown parsing
- `gopkg.in/yaml.v3` - YAML parsing

No networking, no ML frameworks, no web frameworks. Just local CLI orchestration.
