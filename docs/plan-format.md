# Plan Format Guide

Complete specification for Conductor implementation plan formats.

## Table of Contents

- [Overview](#overview)
- [Markdown Format](#markdown-format)
- [YAML Format](#yaml-format)
- [Task Metadata](#task-metadata)
- [Dependencies](#dependencies)
- [Validation Rules](#validation-rules)
- [Best Practices](#best-practices)
- [Examples](#examples)

## Overview

Conductor supports two plan formats:

1. **Markdown Format** (`.md`, `.markdown`) - Human-readable, easy to write
2. **YAML Format** (`.yaml`, `.yml`) - Structured, machine-friendly

Both formats support the same features and are automatically detected by file extension.

## Markdown Format

### Basic Structure

```markdown
# Plan Title

Optional plan description.

## Task N: Task Name
**File(s)**: file1.go, file2.go
**Depends on**: Task 1, Task 2
**Estimated time**: 10 minutes
**Agent**: agent-name

Task description and detailed requirements.

## Task N+1: Next Task
...
```

### Format Specification

#### Plan Header
```markdown
# Plan Title

Optional description paragraph.
Can span multiple lines.
```

- **Title**: First H1 heading (`#`) in file
- **Description**: Optional text between title and first task
- **Frontmatter**: Optional YAML configuration

#### Task Definition
```markdown
## Task N: Task Name
**File(s)**: file1.go, file2.go
**Depends on**: Task 1, Task 2
**Estimated time**: 10 minutes
**Agent**: agent-name

Task description and requirements.
Multiple paragraphs allowed.

Code examples also allowed:
```go
func example() {}
```
```

- **Task Header**: H2 heading (`##`) with `Task N:` prefix
- **Metadata**: Bold key-value pairs
- **Description**: Markdown content after metadata

#### Metadata Format

Metadata uses bold keys followed by colons:

```markdown
**File(s)**: value
**Depends on**: value
**Estimated time**: value
**Agent**: value
**WorktreeGroup**: value
**Status**: value
```

## YAML Format

### Basic Structure

```yaml
plan:
  name: Plan Title
  description: Optional plan description
  tasks:
    - id: 1
      name: Task Name
      files: [file1.go, file2.go]
      depends_on: []
      estimated_time: 10 minutes
      agent: agent-name
      description: Task description.
```

### Format Specification

#### Plan Object
```yaml
plan:
  name: string           # Required: Plan title
  description: string    # Optional: Plan description
  tasks: []             # Required: List of tasks
```

#### Task Object
```yaml
- id: int                    # Required: Task number
  name: string               # Required: Task name
  files: [string]            # Optional: List of files
  depends_on: [string]       # Optional: Task dependencies
  estimated_time: string     # Optional: Estimated duration
  agent: string              # Optional: Agent name
  worktree_group: string     # Optional: Worktree group name (Phase 2A)
  status: string             # Optional: completed|failed|in-progress
  description: string        # Optional: Task description
```

## Task Metadata

### File(s) / files

**Purpose**: Specify files modified by task

**Format:**
- Markdown: `**File(s)**: file1.go, file2.go`
- YAML: `files: [file1.go, file2.go]`

**Rules:**
- Multiple files separated by commas (Markdown) or array (YAML)
- Relative or absolute paths allowed
- Optional but recommended
- Used for file overlap detection

### Depends on / depends_on

**Purpose**: Specify task dependencies

**Format:**
- Markdown: `**Depends on**: Task 1, Task 2`
- YAML: `depends_on: [Task 1, Task 2]`

**Rules:**
- Reference tasks by "Task N" format
- Multiple dependencies separated by commas or array
- Dependencies must exist in plan
- Cannot create circular dependencies
- "None" or empty list for no dependencies

### Estimated time / estimated_time

**Purpose**: Indicate expected task duration

**Format:**
- Markdown: `**Estimated time**: 10 minutes`
- YAML: `estimated_time: 10 minutes`

**Rules:**
- Freeform text field
- No validation or parsing
- Used for planning only
- Optional but recommended

### Agent / agent

**Purpose**: Specify Claude agent for task execution

**Format:**
- Markdown: `**Agent**: agent-name`
- YAML: `agent: agent-name`

**Rules:**
- Agent must exist in `~/.claude/agents/`
- Optional (uses default if not specified)
- Agent discovery scans numbered directories and root

### Status / status

**Purpose**: Track task completion status for resumable execution

**Format:**
- Markdown: `**Status**: completed` or `[x]` checkbox
- YAML: `status: completed`

**Valid Values:**
- `completed` - Task successfully completed
- `failed` - Task failed in previous execution
- `in-progress` - Task is currently running
- (empty) - Task not yet executed

**Rules:**
- Optional field (defaults to empty/pending)
- Set manually or automatically by conductor after execution
- Used with `--skip-completed` flag to resume plans
- Skipped tasks create synthetic GREEN results

**Examples:**

Markdown with explicit status:
```markdown
## Task 1: Already Done
**Status**: completed

This task was already completed and will be skipped.
```

Markdown with checkbox (shorthand):
```markdown
## Task 1: Already Done
- [x] This task is marked as completed
```

YAML format:
```yaml
- id: 1
  name: Already Done
  status: completed
  description: This task was already completed.
```

**Resume Examples:**

```bash
# First run: marks completed tasks in plan file
conductor run plan.md

# Resume later: skip completed tasks
conductor run plan.md --skip-completed

# Retry failed tasks on resume
conductor run plan.md --skip-completed --retry-failed
```

### WorktreeGroup / worktree_group

**Purpose**: Assign task to worktree group for organizational purposes

**Format:**
- Markdown: `**WorktreeGroup**: backend-core`
- YAML: `worktree_group: backend-core`

**Rules:**
- Optional field (defaults to empty if not specified)
- Parsed by conductor (Phase 2A feature)
- Used for task organization and grouping in multi-file plans
- Group names should use hyphens for multi-word names (no spaces)
- Informational metadata - not enforced by conductor execution engine
- Groups can be defined in plan configuration for validation

**Examples:**

Markdown:
```markdown
## Task 2: API Implementation
**File(s)**: api/routes.go
**Depends on**: Task 1
**WorktreeGroup**: backend-core

Implement REST API endpoints.
```

YAML:
```yaml
- id: 2
  name: API Implementation
  files: [api/routes.go]
  depends_on: [1]
  worktree_group: backend-core
  description: Implement REST API endpoints.
```

**See Also:**
- [Phase 2A Guide](phase-2a-guide.md) - Multi-file plans and worktree groups
- [Worktree Best Practices](worktree-best-practices.md) - Using worktree groups effectively

## Dependencies

### Dependency Syntax

Dependencies reference other tasks by their task number:

**Correct:**
```markdown
**Depends on**: Task 1
**Depends on**: Task 1, Task 2
```

```yaml
depends_on: [Task 1]
depends_on: [Task 1, Task 2]
```

### Dependency Rules

1. **Must Exist**: All referenced tasks must exist in plan
2. **No Cycles**: Cannot create circular dependencies
3. **Forward References**: Can depend on tasks defined later in plan
4. **Wave Grouping**: Dependencies determine execution waves

### Wave Examples

**Simple Dependency:**
```markdown
## Task 1: Setup
**Depends on**: None

## Task 2: Implementation
**Depends on**: Task 1
```
Execution: Task 1 (Wave 1) → Task 2 (Wave 2)

**Parallel Dependencies:**
```markdown
## Task 1: Setup
**Depends on**: None

## Task 2: Database
**Depends on**: Task 1

## Task 3: API
**Depends on**: Task 1

## Task 4: Tests
**Depends on**: Task 2, Task 3
```
Execution:
- Wave 1: Task 1
- Wave 2: Task 2, Task 3 (parallel)
- Wave 3: Task 4

## Validation Rules

Conductor validates plans before execution:

### 1. Format Validation

**Markdown:**
- At least one H1 heading (plan title)
- Tasks must use H2 headings with "Task N:" prefix
- Task numbers must be sequential integers
- Metadata must follow task heading

**YAML:**
- Valid YAML syntax
- Required fields: `plan.name`, `plan.tasks`
- Each task must have `id` and `name`
- Task IDs must be unique integers

### 2. Dependency Validation

- All dependencies must reference existing tasks
- No circular dependencies (checked via DFS)
- Dependencies use "Task N" format

**Valid:**
```markdown
## Task 1: Setup
**Depends on**: None

## Task 2: Implementation
**Depends on**: Task 1
```

**Invalid - Circular:**
```markdown
## Task 1: A
**Depends on**: Task 2

## Task 2: B
**Depends on**: Task 1
```
Error: Circular dependency detected

### 3. File Validation

- File paths should not overlap across tasks
- Same file modified by multiple tasks may cause conflicts

**Warning Example:**
```markdown
## Task 1: Setup
**File(s)**: main.go

## Task 2: Implementation
**File(s)**: main.go
```
Warning: File main.go modified by multiple tasks

### 4. Agent Validation

- Agent must exist in `~/.claude/agents/`
- Agent discovery checks numbered directories (01-10) and root

**Valid:**
```markdown
**Agent**: code-implementation
```
(Assumes `~/.claude/agents/code-implementation.md` exists)

## Best Practices

### Plan Design

1. **Start with Foundation**: Place setup/infrastructure tasks first
2. **Maximize Parallelism**: Minimize dependencies where possible
3. **Logical Dependencies**: Only depend on what you actually need
4. **Balance Task Size**: 5-15 minute tasks work best
5. **Clear Task Names**: Use descriptive, actionable names

### Task Definition

1. **Specify Files**: Always include `File(s)` metadata
2. **Estimate Time**: Provide realistic time estimates
3. **Use Agents**: Specify appropriate agent for task type
4. **Detailed Descriptions**: Include enough detail for autonomous execution
5. **Code Examples**: Add code snippets for complex requirements

### Dependencies

1. **Minimal Dependencies**: Only depend on what's necessary
2. **Group Related**: Tasks in same wave should be related
3. **Avoid Chains**: Long dependency chains reduce parallelism
4. **Test Independence**: Make test tasks independent when possible
5. **Documentation Last**: Docs usually depend on implementation

### Format Choice

**Use Markdown when:**
- Writing plans manually
- Need rich descriptions with code examples
- Want human-readable format

**Use YAML when:**
- Generating plans programmatically
- Need strict structure
- Integrating with other tools

## Examples

### Example 1: Simple Feature

**Markdown:**
```markdown
# Simple Feature

## Task 1: Create File
**File(s)**: feature.go
**Estimated time**: 5 minutes

Create new feature file.

## Task 2: Add Tests
**File(s)**: feature_test.go
**Depends on**: Task 1
**Estimated time**: 5 minutes

Add unit tests.

## Task 3: Update Docs
**File(s)**: README.md
**Depends on**: Task 2
**Estimated time**: 3 minutes

Document the feature.
```

### Example 2: Parallel Components

**Markdown:**
```markdown
# Multi-Component Feature

## Task 1: Foundation
**File(s)**: base.go
**Estimated time**: 10 minutes
**WorktreeGroup**: foundation

Create foundation.

## Task 2: Component A
**File(s)**: component_a.go
**Depends on**: Task 1
**Estimated time**: 15 minutes
**WorktreeGroup**: components

Implement component A.

## Task 3: Component B
**File(s)**: component_b.go
**Depends on**: Task 1
**Estimated time**: 15 minutes
**WorktreeGroup**: components

Implement component B.

## Task 4: Integration
**File(s)**: integration.go
**Depends on**: Task 2, Task 3
**Estimated time**: 10 minutes
**WorktreeGroup**: integration

Integrate components.
```

Wave execution:
- Wave 1: Task 1 (foundation group)
- Wave 2: Task 2, Task 3 (components group, parallel)
- Wave 3: Task 4 (integration group)

### Example 3: Full-Stack

**Markdown:**
```markdown
# Full-Stack Implementation

## Task 1: Database Schema
**File(s)**: migrations/001_schema.sql
**Estimated time**: 10 minutes
**WorktreeGroup**: backend

Create database schema.

## Task 2: Backend Model
**File(s)**: models/entity.go
**Depends on**: Task 1
**Estimated time**: 15 minutes
**WorktreeGroup**: backend

Implement backend model.

## Task 3: Backend API
**File(s)**: handlers/api.go
**Depends on**: Task 2
**Estimated time**: 20 minutes
**WorktreeGroup**: backend

Implement REST API.

## Task 4: Frontend Component
**File(s)**: components/Feature.tsx
**Estimated time**: 20 minutes
**WorktreeGroup**: frontend

Create frontend component.

## Task 5: Frontend Integration
**File(s)**: pages/feature.tsx
**Depends on**: Task 3, Task 4
**Estimated time**: 15 minutes
**WorktreeGroup**: frontend

Integrate frontend with API.

## Task 6: Tests
**File(s)**: tests/e2e_test.go
**Depends on**: Task 5
**Estimated time**: 20 minutes
**WorktreeGroup**: testing

Write end-to-end tests.

## Task 7: Documentation
**File(s)**: docs/feature.md
**Depends on**: Task 6
**Estimated time**: 15 minutes
**WorktreeGroup**: docs

Document feature.
```

## See Also

- [Usage Guide](usage.md) - CLI commands and execution
- [Troubleshooting Guide](troubleshooting.md) - Common issues and solutions
- [README](../README.md) - Project overview
