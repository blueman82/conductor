---
name: implementation-planner
description: Generate comprehensive implementation plans for features. Use when user requests "help me implement X", "create a plan for X", "break down feature X", "how should I build X", or asks for detailed implementation guidance. Activates for planning requests, not exploratory design discussions.
allowed-tools: Read, Bash, Glob, Grep, Write, TodoWrite, Task
---

# Implementation Planner v4.1 (Explorer Agents Mode)

Generate conductor-compatible YAML plans. **Do NOT activate for:** questions, debugging, code reviews.

## Workflow

1. **Discover** → Launch Explore agents (Task tool, `subagent_type: Explore`) for agents, codebase, patterns
2. **Design** → Break into tasks, map dependencies, build data flow registry
3. **Implement** → Write `implementation.key_points` FIRST for each task
4. **Criteria** → Derive `success_criteria` FROM key_points (same terminology)
5. **Classify** → CAPABILITY (unit) vs INTEGRATION (cross-component)
6. **Generate** → Output YAML with all required fields
7. **Validate** → `conductor validate <plan>.yaml`

---

## Critical Rules

| Rule | Rationale |
|------|-----------|
| **Key points → Criteria** | Every `success_criteria` item MUST trace to a `key_point` using IDENTICAL terms |
| **Data flow deps** | If task B uses function from task A, B must `depends_on: [A]` |
| **Package serialization** | Go: tasks modifying same package need sequential deps |
| **Verify before claiming** | `grep` to confirm existing behavior before writing key_points |
| **Verify file paths** | `find`/`ls` to confirm test files exist before adding to `test_commands` |
| **Code reuse first** | Search for existing implementations before creating new code |
| **No wrappers without value** | Direct usage preferred over unnecessary abstraction |

**Auto-append to ALL tasks' success_criteria:**
- No TODO comments in production code
- No placeholder structs (Type{})
- No unused variables (_ = x)
- All imports from deps resolve

---

## Phase 1-2: Discovery & Design

### Explorer Agents Mode (RECOMMENDED)

Launch up to 3 Explore agents IN PARALLEL using the Task tool with `subagent_type: Explore`:

```
┌──────────────────────────────────────────────────────────────────┐
│  PARALLEL EXPLORE AGENTS                                         │
├──────────────────────────────────────────────────────────────────┤
│  Agent 1: "Explore available agents in ~/.claude/agents"        │
│  Agent 2: "Explore codebase structure, stack, existing patterns"│
│  Agent 3: "Search for existing implementations of <feature>"    │
└──────────────────────────────────────────────────────────────────┘
```

**Benefits over direct bash:**
- Synthesized findings, not raw file lists
- Can follow references and trace dependencies
- Handles open-ended exploration efficiently

**Agent count guidance:**
- 1 agent: Isolated task, known files, small targeted change
- 2-3 agents: Uncertain scope, multiple areas, need pattern discovery

### Fallback: Direct Commands (when agents unavailable)

```bash
fd '\.md$' ~/.claude/agents --type f     # Available agents
ls -la && cat go.mod                      # Codebase structure
ls <expected/path> 2>/dev/null            # Verify paths exist
grep -r "pattern" internal/               # Find existing implementations
```

### Data Flow Registry (CRITICAL)

Build producer/consumer map to ensure correct dependencies:

```yaml
# Comment block at top of plan:
# PRODUCERS: Task 4 → ExtractMetrics, Task 5 → LoadSession
# CONSUMERS: Task 16 → [4, 5, 15]
# VALIDATION: All consumers depend_on producers ✓

# YAML field (validated by Conductor):
data_flow_registry:
  producers:
    FunctionName:
      - task: 4
        description: "Creates this function"
  consumers:
    FunctionName:
      - task: 16
        description: "Uses this function"
```

---

## Phase 3-4: Implementation → Criteria

**Write key_points FIRST, then derive criteria:**

```yaml
implementation:
  approach: |
    Strategy and architectural decisions.
  key_points:
    - point: "EnforcePackageIsolation with git diff"
      details: "Compare modified files against task.Files"
      reference: "internal/executor/package_guard.go"

# Derived using SAME terminology:
success_criteria:
  - "EnforcePackageIsolation runs git diff, compares against task.Files"
```

**Key point requirements:** Specific (names exact function/type), Verifiable (grep/test), Complete (all requirements).

---

## Phase 5: Classification

| Type | Scope | Criteria Field |
|------|-------|----------------|
| CAPABILITY | What component does alone | `success_criteria` |
| INTEGRATION | How components work together | `integration_criteria` |

**Route by keyword:**
- CLI flags, UI rendering, cross-component calls → `integration_criteria`
- Internal logic, data structures, algorithms → `success_criteria`

**RFC 2119 Routing:**

| Level | Route To | Behavior |
|-------|----------|----------|
| MUST | `test_commands` | Hard gate |
| SHOULD | `success_criteria` | QC reviews |
| MAY | `documentation_targets` | Informational |

---

## YAML Schema

### Root Structure

```yaml
conductor:
  default_agent: general-purpose
  worktree_groups:
    - group_id: "name"
      tasks: [1, 2, 3]
      rationale: "Why grouped"

planner_compliance:
  planner_version: "4.0.0"
  strict_enforcement: true
  required_features: [dependency_checks, test_commands, success_criteria, data_flow_registry, package_guard] # PACKAGE GUARD IS NOT NEEDED FOR NON GO-LANG PROJECTS

data_flow_registry:
  producers: {}
  consumers: {}

plan:
  metadata:
    feature_name: "Name"
    created: "YYYY-MM-DD"
    target: "Goal"
  context:
    framework: "Go"
    test_framework: "go test"
  tasks: []
```

### Task Structure

```yaml
- task_number: "1"
  name: "Task name"
  agent: "agent-name"
  files: ["path/to/file.go"]
  depends_on: []

  success_criteria:
    - "Criterion from key_point 1"
    - "No TODO/placeholder/unused patterns"

  test_commands:
    - "go test ./path -run TestName"

  runtime_metadata:
    dependency_checks:
      - command: "go build ./..."
        description: "Verify build"
    documentation_targets: []

  description: |
    <dependency_verification priority="execute_first">
      <commands># verify deps</commands>
    </dependency_verification>
    <task_description>What to implement.</task_description>

  implementation:
    approach: |
      Strategy.
    key_points:
      - point: "Name"
        details: "What and why"
        reference: "file.go"

  code_quality:
    go:
      full_quality_pipeline:
        command: "gofmt -w . && go test ./..."
        exit_on_failure: true

  commit:
    type: "feat"
    message: "description"
    files: ["path/**"]
```

### Cross-File Dependencies

```yaml
depends_on:
  - 4                                    # Same file
  - file: "plan-01-foundation.yaml"      # Different file
    task: 2
```

### Integration Tasks

```yaml
- task_number: "N"
  type: integration
  success_criteria: [...]      # Component-level
  integration_criteria: [...]  # Cross-component
```

---

## TDD Guidance

TDD is RECOMMENDED for component tasks (new logic). Not required for wiring/config/docs.

| Task Type | Testing Approach |
|-----------|------------------|
| New component with logic | TDD preferred - test in same task |
| Wiring/integration | Test-after or integration tests |
| Config/types only | Validation tests if applicable |
| Documentation | None |

When using TDD, include test file in task's `files[]` and test command in `test_commands[]`.

---

## Validation Checklist

```
□ Every key_point has corresponding success criterion (same terms)
□ Every success criterion traces to a key_point
□ Data flow: consumers depend_on all producers
□ Package conflicts serialized via depends_on
□ All file paths verified to exist (or will be created)
□ Integration tasks have both success_criteria AND integration_criteria
□ Every task has: implementation, success_criteria, test_commands, runtime_metadata
```

**Final step:**
```bash
conductor validate docs/plans/<plan>.yaml
```

---

## Common Failures

| Failure | Prevention |
|---------|------------|
| Agent implements wrong thing | Write ALL requirements in key_points |
| QC fails despite working code | Derive criteria FROM key_points |
| Missing dependency | Build producer registry |
| Scope leak | Classify criteria by type |
| Assumed behavior wrong | grep before claiming defaults |
| Multi-file plan: ~2000 lines max | Split at worktree group boundaries |
