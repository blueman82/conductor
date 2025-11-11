# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Conductor is an autonomous multi-agent orchestration CLI built in Go that executes implementation plans by spawning and managing multiple Claude Code CLI agents in coordinated waves. It parses plan files (Markdown or YAML), calculates task dependencies using graph algorithms, and orchestrates parallel execution with quality control reviews.

**Current Status**: ✅ **ALL 25 TASKS COMPLETE**. Conductor is fully implemented, tested, and production-ready. All core functionality, logging, configuration, error handling, integration tests, build tooling, and documentation complete. Latest updates: Task 21 (error types integrated into executor) and Task 25 (15 E2E tests implemented).

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

**Current Status**: **✅ PRODUCTION READY** - All 25 tasks complete. The conductor binary is fully functional with 86.4% test coverage (465+ tests passing). Complete pipeline: parsing → validation → execution → logging → completion. All documentation, tooling, and error handling in place. Ready for production deployment.**

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
