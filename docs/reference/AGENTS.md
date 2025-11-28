# AGENTS.md - Agent Architecture in Conductor

## Overview

Conductor is a Go-based agent orchestration system that invokes Claude Code CLI agents to execute implementation tasks. The system manages task execution, dependency resolution, parallel execution waves, and quality control validation.

## Core Agent System

### Agent Definition

Agents in Conductor are defined as YAML frontmatter in Markdown files following this structure:

```markdown
---
name: agent-name
description: When to invoke this agent
tools:
  - Read
  - Grep
  - Edit
---

# Agent System Prompt

Detailed instructions and context for the agent...
```

**Required Fields:**
- `name`: Unique identifier (lowercase, hyphen-separated)
- `description`: Natural language description for the agent
- `prompt`: System instructions (Markdown body after frontmatter)

**Optional Fields:**
- `tools`: List of available tools for the agent (inherited if omitted)

### Agent Discovery

Agents are discovered from `.md` files in configurable directories:

1. **Project-Level Agents**: `.claude/agents/*.md` (highest priority)
2. **User-Level Agents**: `~/.claude/agents/*.md` (default fallback)

The `Registry` component handles agent discovery and storage:

```go
type Registry struct {
    AgentsDir string              // Directory to scan for agent files
    agents    map[string]*Agent   // Discovered agents by name
}

type Agent struct {
    Name        string   // Agent identifier
    Description string   // Usage description
    Tools       []string // Available tools
    FilePath    string   // Location of agent file
}
```

#### Discovery Process (Strategy B - Directory Whitelisting)

The discovery process uses directory whitelisting to reduce false warnings from documentation files:

1. **Directory Whitelisting**:
   - Include root-level `.md` files (agent definitions)
   - Include numbered subdirectories: `01-*`, `02-*`, ..., `10-*` (categorized agents)
   - Skip special directories: `examples/`, `transcripts/`, `logs/` (documentation/metadata)

2. **File Filtering**:
   - Process only `.md` files
   - Skip `README.md` (category documentation)
   - Skip `*-framework.md` (methodology documentation)

3. **Parsing**:
   - Extract YAML frontmatter (between `---` delimiters)
   - Skip invalid/incomplete agent definitions (warning logged, no error)

4. **Results**:
   - Return map of agent name → Agent struct
   - Return empty map if directory doesn't exist (not an error)

## Agent Invocation

### Invoker Component

The `Invoker` executes Claude Code CLI commands to invoke agents:

```go
type Invoker struct {
    ClaudePath string    // Path to claude CLI binary (default: "claude")
    Registry   *Registry // Optional agent registry for agent references
}

type InvocationResult struct {
    Output   string        // Captured output from agent
    ExitCode int           // Exit code (0 = success)
    Duration time.Duration // Execution time
    Error    error         // Any system errors
}
```

### Command Building

The invoker builds Claude Code CLI arguments programmatically:

```
claude -p "prompt text" --settings '{"disableAllHooks": true}' --output-format json
```

**Key Flags:**
- `-p` (print mode): Execute non-interactively and exit
- `--settings '{"disableAllHooks": true}'`: Disable hooks for automation
- `--output-format json`: Parse output as JSON

### Agent Reference in Prompts

When a task specifies an agent and the registry has that agent:

```
"use the {agent-name} subagent to: {original-prompt}"
```

This allows the main Claude instance to delegate to a specialized subagent. If the agent isn't in the registry, the prompt is used as-is.

### Invocation Methods

```go
// With timeout
result, err := invoker.InvokeWithTimeout(task, 5*time.Minute)

// With context
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()
result, err := invoker.Invoke(ctx, task)
```

**Exit Code Handling:**
- Exit code 0: Success
- Non-zero exit code: Error (captured in `result.ExitCode`)
- System error (binary not found): Captured in `result.Error`

### Output Parsing

```go
type ClaudeOutput struct {
    Content string `json:"content"`
    Error   string `json:"error"`
}

// Parse JSON output, fall back to raw text if not JSON
parsed, err := ParseClaudeOutput(jsonString)
```

Fallback behavior: If JSON parsing fails, the raw output is returned as `Content`.

## Task Model

Tasks are the unit of work executed by agents:

```go
type Task struct {
    Number        int           // Unique identifier
    Name          string        // Human-readable name (required)
    Files         []string      // Files affected by task
    DependsOn     []int         // Task numbers this depends on
    EstimatedTime time.Duration // Time estimate
    Agent         string        // Optional agent to use (references registry)
    Prompt        string        // Full task description (required)
}
```

**Task Validation:**
- Must have `Number > 0`
- Must have `Name` (non-empty)
- Must have `Prompt` (non-empty)

### Task Results

```go
type TaskResult struct {
    Task           Task          // The executed task
    Status         string        // "GREEN", "RED", "TIMEOUT", "FAILED"
    Output         string        // Agent output
    Error          error         // Execution error
    Duration       time.Duration // Actual execution time
    RetryCount     int           // Number of retries attempted
    ReviewFeedback string        // QC feedback if reviewed
}
```

## Plan Model

Plans organize tasks into execution units:

```go
type Plan struct {
    Name           string               // Plan identifier
    Tasks          []Task               // All tasks in plan
    Waves          []Wave               // Execution waves (task groupings)
    DefaultAgent   string               // Default agent if task.Agent is empty
    QualityControl QualityControlConfig // QC settings
    FilePath       string               // Original file path
}

type Wave struct {
    Name           string // Wave identifier
    TaskNumbers    []int  // Task IDs in this wave
    MaxConcurrency int    // Max parallel tasks
}

type QualityControlConfig struct {
    Enabled     bool   // Enable QC review
    ReviewAgent string // Agent to use for reviews
    RetryOnRed  int    // Retry count on RED status
}
```

### Dependency Resolution

Plans support task dependencies for sequencing:

```
Task 1: Initialize
Task 2: (depends on 1) Build schema
Task 3: (depends on 2) Load data
Task 4: (depends on 2,3) Validate
```

**Cycle Detection:**
- Tasks cannot depend on themselves
- Circular dependencies are detected and rejected
- Implemented via DFS during plan validation

### Wave Calculation

Waves are calculated by grouping tasks that:
1. Are independent of each other (no blocking dependencies)
2. Can execute in parallel
3. Respect max concurrency limits

**Topological Sort Approach:**
- Build dependency graph
- Find all tasks with no dependencies → Wave 1
- Repeat with completed tasks filtered out → Wave N

## Quality Control System

Quality control validates task execution through agent review:

```go
type QualityController struct {
    Invoker     InvokerInterface // Agent executor
    ReviewAgent string           // Agent name for reviews (e.g., "quality-control")
    MaxRetries  int              // Retry attempts for RED status
}

type ReviewResult struct {
    Flag     string // "GREEN", "RED", or "YELLOW"
    Feedback string // Detailed review feedback
}
```

### Review Workflow

1. **Build Review Prompt**: Task name + output → review prompt
2. **Invoke QC Agent**: Call ReviewAgent with review task
3. **Parse Response**: Extract flag (GREEN/RED/YELLOW) and feedback
4. **Determine Action**:
   - **GREEN**: Task passes, mark complete
   - **RED**: Task fails, retry if attempts remaining
   - **YELLOW**: Minor issues, pass with notes

### Review Prompt Format

```
Review the following task execution:

Task: {task.Name}

Output:
{agent-output}

Provide quality control review in this format:
Quality Control: [GREEN/RED/YELLOW]

Feedback: [your detailed feedback]
```

### Retry Logic

```go
func (qc *QualityController) ShouldRetry(result *ReviewResult, attempt int) bool {
    if result.Flag != "RED" {
        return false
    }
    return attempt < qc.MaxRetries
}
```

Retries only trigger on RED status, respecting `MaxRetries` limit.

## Parser System

Plans can be defined in Markdown or YAML format:

### Markdown Format

```markdown
# Plan Name

## Task 1: Initialize
**Estimated time**: 30m
**Agent**: general

Description of task 1...

## Task 2: Build Features
**Depends on**: 1
**Estimated time**: 2h
**Agent**: swiftdev

Description of task 2...
```

### YAML Format

```yaml
name: Example Plan
tasks:
  - number: 1
    name: Initialize
    estimated_time: 30m
    agent: general
    prompt: |
      Description of task 1
  - number: 2
    name: Build Features
    depends_on: [1]
    estimated_time: 2h
    agent: swiftdev
    prompt: |
      Description of task 2
```

### Format Detection

Parser auto-detects format by:
1. YAML frontmatter (`---`) → Markdown
2. YAML extension (`.yaml`, `.yml`) → YAML
3. Default → Markdown

## Execution Flow

### High-Level Execution

```
1. Parse plan (Markdown/YAML)
2. Validate tasks & dependencies
3. Calculate execution waves
4. For each wave:
   a. Execute tasks in parallel (respecting max concurrency)
   b. Invoke agent for each task
   c. If QC enabled, review output with QC agent
   d. On RED: retry task (if attempts remaining)
   e. Track results
5. Return execution summary
```

### Error Handling

**Task Failures:**
- Timeout: Mark RED, retry if QC enabled
- Agent error: Mark FAILED, don't retry
- QC rejection: Mark RED, retry if attempts remaining

**System Errors:**
- Invalid plan: Fail immediately
- Cyclic dependencies: Fail immediately
- Missing agents: Continue (use default agent)

## Integration Points

### Claude Code CLI Integration

Tasks are executed by invoking Claude Code CLI in print mode (`-p`):

```bash
claude -p "use the swiftdev subagent to: implement feature X" \
  --settings '{"disableAllHooks": true}' \
  --output-format json
```

**Agent Delegation:**
- Main Claude instance can delegate to subagents
- Subagent context is isolated from main conversation
- Subagents report results back to main thread

### File-Based Configuration

Plans can reference agent configurations from disk:

```
~/.claude/agents/quality-control.md     # QC agent definition
~/.claude/agents/swiftdev.md           # Swift developer agent
.claude/agents/project-specific.md      # Project-level agents
```

Agent files are discovered at runtime and used to enhance task prompts.

## Data Flow Diagram

```
┌─────────────┐
│ Plan File   │
│ (MD/YAML)   │
└──────┬──────┘
       │
       ▼
┌─────────────────┐
│ Parser          │ ──► Detect format
├─────────────────┤     Parse tasks
│ - Markdown      │     Extract metadata
│ - YAML          │
└──────┬──────────┘
       │
       ▼
┌─────────────────────┐
│ Plan Model          │
├─────────────────────┤
│ - Tasks             │
│ - Dependencies      │
│ - QC Config         │
└──────┬──────────────┘
       │
       ▼
┌──────────────────────┐
│ Dependency Graph     │
├──────────────────────┤
│ - Cycle detection    │
│ - Wave calculation   │
│ - Topological sort   │
└──────┬───────────────┘
       │
       ▼
┌─────────────────────────────┐
│ Agent Registry              │
├─────────────────────────────┤
│ - Discover agents from FS   │
│ - Parse agent definitions   │
│ - Store in memory map       │
└──────┬──────────────────────┘
       │
       ▼
┌─────────────────────────────┐
│ Executor                    │
├─────────────────────────────┤
│ For each wave:              │
│  - Invoker.Invoke(task)     │
│  - QC.Review(output)        │
│  - Retry on RED             │
│  - Track results            │
└──────┬──────────────────────┘
       │
       ▼
┌──────────────────────┐
│ Execution Result     │
├──────────────────────┤
│ - Total tasks        │
│ - Completed count    │
│ - Failed tasks       │
│ - Duration           │
└──────────────────────┘
```

## Usage Example

```go
package main

import (
    "github.com/harrison/conductor/internal/agent"
    "github.com/harrison/conductor/internal/models"
    "github.com/harrison/conductor/internal/parser"
)

func main() {
    // 1. Discover agents
    registry := agent.NewRegistry("") // ~/.claude/agents
    registry.Discover()
    
    // 2. Parse plan
    p := parser.NewParser()
    plan, err := p.ParseFile("implementation.md")
    
    // 3. Create invoker with registry
    invoker := agent.NewInvokerWithRegistry(registry)
    
    // 4. Execute task
    task := plan.Tasks[0]
    result, err := invoker.InvokeWithTimeout(task, 5*time.Minute)
    
    // 5. Review with QC
    qc := executor.NewQualityController(invoker)
    review, err := qc.Review(context.Background(), task, result.Output)
    
    // 6. Check status
    if review.Flag == "GREEN" {
        // Task passed
    } else if review.Flag == "RED" {
        // Retry task
    }
}
```

## Implementation Status

**Fully Implemented (v1.0):**
- ✅ Agent discovery and registry
- ✅ Task model with validation
- ✅ Agent invoker (Claude Code CLI integration)
- ✅ Plan parsing (Markdown and YAML)
- ✅ Dependency graph and wave calculation
- ✅ Quality control review system
- ✅ Comprehensive test suite (451 tests, 78.3% coverage)
- ✅ Parallel wave execution (Task 13)
- ✅ Retry logic integration (Task 14)
- ✅ Execution result aggregation (Task 15)
- ✅ CLI commands for execution (Tasks 16-17)
- ✅ File locking for concurrent updates (Task 11)
- ✅ Plan updater (Task 12)
- ✅ Comprehensive documentation (Task 24)

**Current Status:**
- Production-ready v1.0 with all core features
- 24/25 tasks completed (only Task 25 remaining)
- Ready for end-to-end execution with real plans
- Awaiting final integration testing (Task 25)

## Testing

Conductor uses comprehensive unit tests following TDD principles:

```bash
# Run all tests
go test ./...

# Run specific package tests
go test ./internal/agent/...

# Run with coverage
go test -cover ./...
```

**Test Coverage Areas:**
- Agent discovery (multiple agents, missing dirs, invalid YAML)
- Invoker (command building, output parsing, exit codes, timeouts)
- Task validation (required fields, dependency cycles)
- QC flow (GREEN/RED/YELLOW parsing, retry logic)
- Plan parsing (Markdown, YAML, format detection)
- Dependency resolution (cycle detection, topological sort)

## References

- [Claude Code CLI Documentation](./docs/plans/claude_code_agents_invocation.md)
- [Subagent Deployment Strategy](./docs/plans/subagent-deployment-strategy.md)
- [Implementation Plan](./docs/plans/conductor-v1-implementation.md)
