---
name: implementation-planner
description: Generate comprehensive implementation plans for features. Use when user requests "help me implement X", "create a plan for X", "break down feature X", "how should I build X", or asks for detailed implementation guidance. Activates for planning requests, not exploratory design discussions.
allowed-tools: Read, Bash, Glob, Grep, Write, TodoWrite
---

# Implementation Planner Skill

**Version:** 3.0.0
**Purpose:** Generate conductor-compatible YAML implementation plans with built-in validation.

## Activation

**Activate for:**
- "Help me implement [feature]"
- "Create a plan for [feature]"
- "Break down [feature] into tasks"

**Do NOT activate for:** Questions, debugging, code reviews, exploratory discussions.

---

## Phase 1: Discovery

### 1.1 Discover Agents
```bash
fd '\.md$' ~/.claude/agents --type f
```
Extract names (remove path/extension). Default: `general-purpose`.

### 1.2 Analyze Codebase
```bash
ls -la                           # Structure
cat go.mod || cat package.json   # Stack
```

**Document:** Framework, test framework, architecture pattern, existing patterns for similar features.

**CRITICAL:** Verify file organization before specifying paths:
```bash
ls internal/learning/migrations/ 2>/dev/null  # Does dir exist?
grep -r "CREATE TABLE" internal/learning/     # Where does SQL live?
```

---

## Phase 2: Task Design

### 2.1 Break Feature into Tasks
- Each task = one focused unit of work
- Identify dependencies between tasks
- Assign appropriate agent per task

### 2.2 Build Dependency Graph
- Map `depends_on` relationships
- Detect package conflicts (Go: tasks modifying same package need serialization)
- Create worktree groups (organizational only, NOT execution control)

### 2.3 Data Flow Analysis (CRITICAL)

**Problem:** Feature-chain thinking produces wrong dependencies.

**Process:**
1. Extract function/type references from task descriptions
2. Build producer registry: `{function: task_that_creates_it}`
3. For each task, identify what it consumes
4. Validate `depends_on` includes ALL producers

```yaml
# Add to plan header:
# DATA FLOW REGISTRY
# PRODUCERS: Task 4 → ExtractMetrics, Task 5 → LoadSession
# CONSUMERS: Task 16 → [4, 5, 15]
# VALIDATION: All consumers depend_on their producers ✓
```

---

## Phase 3: Write Implementation Section

For each task, write `implementation:` FIRST, then derive criteria.

### 3.1 Implementation Structure

```yaml
implementation:
  approach: |
    Strategy and architectural decisions.
  key_points:
    - point: "Descriptive name"
      details: "What this accomplishes and why"
      reference: "path/to/file.go"
    - point: "Another key point"
      details: "Details here"
      reference: "path/to/other.go"
  integration:  # Only for tasks with depends_on
    imports: ["package/from/dep"]
    config_values: ["setting.name"]
```

### 3.2 Key Points Rules

Each key_point must be:
- **Specific**: Names exact function, type, or behavior
- **Verifiable**: Can be checked with grep/test
- **Complete**: Covers ALL requirements for the task

---

## Phase 4: Derive Success Criteria (CRITICAL)

**RULE:** Success criteria MUST be derived directly from key_points using SAME terminology.

### 4.1 Derivation Process

```
For each key_point:
  → Write criterion that verifies THIS specific point
  → Use EXACT same terms as the key_point
  → Criterion = testable assertion of key_point
```

### 4.2 Example

```yaml
# WRITE key_points FIRST:
key_points:
  - point: "EnforcePackageIsolation with git diff"
    details: "Run git diff --name-only, compare against task.Files, fail if outside scope"
    reference: "internal/executor/package_guard.go"

# THEN derive success_criteria using same words:
success_criteria:
  - "EnforcePackageIsolation runs git diff --name-only before test commands, compares against task.Files, fails with remediation message if files modified outside declared scope."
```

### 4.3 Anti-Pattern: Misaligned Criteria

```yaml
# BAD - criteria uses different terms than key_points:
key_points:
  - point: "Runtime package locks"
    details: "Mutex prevents concurrent modifications"

success_criteria:
  - "EnforcePackageIsolation validates file scope with git diff"  # WRONG - not in key_points!

# GOOD - criteria matches key_points:
key_points:
  - point: "Runtime package locks"
    details: "Mutex prevents concurrent modifications"
  - point: "EnforcePackageIsolation with git diff"
    details: "Validate file scope before tests"

success_criteria:
  - "Runtime package locks via mutex prevent concurrent modifications to same Go package."
  - "EnforcePackageIsolation runs git diff --name-only, validates against task.Files."
```

### 4.4 Auto-Append Anti-Pattern Criteria

Add to ALL tasks:
```yaml
success_criteria:
  # Task-specific (derived from key_points)
  - "..."
  # Auto-appended:
  - "No TODO comments in production code paths."
  - "No placeholder empty structs (e.g., Type{})."
  - "No unused variables (_ = x pattern)."
  - "All imports from dependency tasks resolve."
```

---

## Phase 5: Criteria Classification

### 5.1 CAPABILITY vs INTEGRATION

| Type | Definition | Test Method |
|------|------------|-------------|
| CAPABILITY | What component CAN do | Unit test with task's files only |
| INTEGRATION | How components WORK TOGETHER | E2E across components |

### 5.2 Rules

- **Component tasks** (`type: component` or no type): ONLY capability criteria
- **Integration tasks** (`type: integration`): BOTH `success_criteria` AND `integration_criteria`

### 5.3 Integration Indicator Keywords

Move criterion to integration task if it contains:
- CLI: "flag", "--", "command", "argument"
- UI: "displays", "shows", "renders"
- Cross-component: "when X then Y", "triggers", "calls [other component]"

```yaml
# BAD - CLI criterion in cache component task:
success_criteria:
  - "Cache can be bypassed with --no-cache flag"  # Requires CLI!

# GOOD - Split:
# Cache task:
success_criteria:
  - "CacheManager accepts enabled: boolean option"
  - "When enabled=false, get() returns null"

# CLI task or integration task:
integration_criteria:
  - "CLI --no-cache flag passes enabled=false to CacheManager"
```

### 5.4 RFC 2119 Requirement Levels

Conductor's plan structure maps to RFC 2119 requirement levels. Route criteria to the appropriate field:

| Level | Route To | Behavior |
|-------|----------|----------|
| **MUST** | `test_commands` | Hard gate - task fails if not met |
| **MUST** | `dependency_checks` | Preflight gate - blocks task start |
| **SHOULD** | `success_criteria` | Soft signal - QC reviews and judges |
| **MAY** | `documentation_targets` | Informational - YELLOW max |

**Routing Rules:**
- Absolute requirements with verifiable commands → `test_commands`
- Subjective quality criteria → `success_criteria`
- Nice-to-have enhancements → `documentation_targets` or omit

**Embedding Levels in Criterion Text:**

When criteria need explicit severity signaling to QC agents, prefix with RFC 2119 keywords:

```yaml
success_criteria:
  # MUST-level in success_criteria (QC treats as hard fail)
  - "MUST: Function validates all input before processing"
  - "MUST NOT: Function must not expose internal errors to users"

  # SHOULD-level (QC may issue YELLOW)
  - "SHOULD: Error messages include file paths for debugging"
  - "SHOULD NOT: Avoid blocking the main thread"

  # MAY-level (informational)
  - "MAY: Support custom exclusion patterns via environment variable"
```

**Classification Heuristics:**

| Pattern in Criterion | Level | Route |
|---------------------|-------|-------|
| "validates", "rejects invalid", "prevents", "fails if" | MUST | `test_commands` if verifiable |
| "returns error when", "must not expose" | MUST | `success_criteria` with prefix |
| "handles gracefully", "includes", "logs" | SHOULD | `success_criteria` |
| "supports optional", "can be configured" | MAY | `documentation_targets` |

---

## Phase 6: Validation (MANDATORY)

### 6.1 Key Points ↔ Success Criteria Alignment

**For EACH task, verify:**

```
□ Every key_point has a corresponding success criterion
□ Every success criterion traces to a key_point
□ Same terminology used in both
□ No orphan criteria (criteria without key_point source)
```

### 6.2 Behavioral Fact Verification

Before writing key_points that claim existing behavior:
```bash
# Verify defaults
grep -n "??" <file> | grep <option>

# Verify option existence
grep -n "option\|flag\|--" <file>

# Verify function behavior
grep -A5 "func <name>" <file>
```

### 6.3 Dependency Completeness

```
□ All numeric deps exist (same file)
□ All cross-file references point to real files/tasks
□ No circular dependencies
□ Data flow producers included in depends_on
```

### 6.4 Structure Completeness

```
□ Every task has implementation section with approach + key_points
□ Every task has success_criteria (derived from key_points)
□ Every task has test_commands
□ Every task has code_quality pipeline
□ Integration tasks have BOTH success_criteria AND integration_criteria
□ Files are flat lists (not nested)
```

### 6.5 Runtime Enforcement (v2.9+)

Conductor enforces quality gates at runtime:

| Field | Type | Behavior |
|-------|------|----------|
| `test_commands` | **Hard gate** | Must pass or task fails |
| `key_points` | **Soft signal** | Verified, results sent to QC |
| `documentation_targets` | **Soft signal** | Checked, results sent to QC |

```yaml
# Hard gate - blocks task if fails:
test_commands:
  - "go test ./internal/executor -run TestFoo"
  - "go build ./..."

# Soft signal - verified before QC:
implementation:
  key_points:
    - point: "Function name"
      details: "What it does"
      reference: "path/to/file.go"  # Verified to exist

# Soft signal - for doc tasks:
documentation_targets:
  - file: "docs/README.md"
    section: "## Installation"
    action: "update"
```

---

## Phase 7: YAML Generation

### 7.1 Root Structure

```yaml
# ═══════════════════════════════════════════════════════════════
# DATA FLOW REGISTRY
# ═══════════════════════════════════════════════════════════════
# PRODUCERS: Task N → Function/Type
# CONSUMERS: Task M → [deps]
# VALIDATION: All consumers depend_on producers ✓
# ═══════════════════════════════════════════════════════════════
# SUCCESS CRITERIA VALIDATION
# ═══════════════════════════════════════════════════════════════
# All criteria derived from key_points ✓
# Same terminology in key_points and criteria ✓
# Component tasks have CAPABILITY-only criteria ✓
# Integration tasks have dual criteria ✓
# ═══════════════════════════════════════════════════════════════

conductor:
  default_agent: general-purpose
  # quality_control: Omit to inherit from .conductor/config.yaml
  worktree_groups:
    - group_id: "group-name"
      description: "Purpose"
      tasks: [1, 2, 3]
      rationale: "Why grouped"

# Enables strict validation when strict_rubric: true in .conductor/config.yaml
# NOTE: strict_enforcement: true requires runtime_metadata on EVERY task
planner_compliance:
  planner_version: "3.0.0"
  strict_enforcement: false  # Set true only if all tasks have runtime_metadata
  required_features:
    - dependency_checks
    - test_commands
    - documentation_targets
    - success_criteria
    - data_flow_registry
    # Go projects only:
    # - package_guard

plan:
  metadata:
    feature_name: "Feature Name"
    created: "YYYY-MM-DD"
    target: "What this achieves"

  context:
    framework: "Framework"
    architecture: "Pattern"
    test_framework: "Test framework"

  tasks:
    - task_number: "1"
      name: "Task name"
      agent: "agent-name"
      files:
        - "path/to/file.go"
      depends_on: []
      estimated_time: "30m"

      success_criteria:
        - "Criterion derived from key_point 1"
        - "Criterion derived from key_point 2"
        - "No TODO comments in production code paths."
        - "No placeholder empty structs."
        - "No unused variables."
        - "All imports from dependency tasks resolve."

      test_commands:
        - "go test ./path -run TestName"

      # Required when strict_enforcement: true
      runtime_metadata:
        dependency_checks:
          - command: "go build ./..."
            description: "Verify build succeeds"
        documentation_targets: []  # Optional: doc sections to verify
        prompt_blocks: []          # Optional: extra agent context

      description: |
        ## PHASE 0: DEPENDENCY VERIFICATION (EXECUTE FIRST)
        ```bash
        # Verify dependencies exist
        ```

        ## TASK DESCRIPTION
        What to implement.

      implementation:
        approach: |
          Strategy here.
        key_points:
          - point: "Key point 1"
            details: "Details"
            reference: "file.go"
          - point: "Key point 2"
            details: "Details"
            reference: "file.go"
        integration: {}

      verification:
        automated_tests:
          command: "go test ./..."
          expected_output: "Tests pass"

      code_quality:
        go:
          full_quality_pipeline:
            command: |
              gofmt -w . && golangci-lint run ./... && go test ./...
            exit_on_failure: true

      commit:
        type: "feat"
        message: "description"
        files:
          - "path/**"
```

### 7.2 Cross-File Dependencies

```yaml
depends_on:
  - 4                              # Same file
  - file: "plan-01-foundation.yaml"  # Different file
    task: 2
```

### 7.3 Integration Task Structure

```yaml
- task_number: "N"
  name: "Wire X to Y"
  type: integration
  files:
    - "component1/file.go"
    - "component2/file.go"
  depends_on: [component1_task, component2_task]

  success_criteria:      # Component-level
    - "Function signatures correct"

  integration_criteria:  # Cross-component
    - "X calls Y in correct sequence"
    - "Error propagates end-to-end"
```

### 7.4 Multi-File Plans

Split at ~2000 lines at worktree group boundaries:
```
docs/plans/feature-name/
├── plan-01-foundation.yaml
├── plan-02-execution.yaml
└── plan-03-integration.yaml
```

---

## Phase 8: Final Validation

Run before outputting:
```bash
conductor validate docs/plans/<plan>.yaml
```

**Output confirmation:**
```
YAML plan: docs/plans/<slug>.yaml
- Total tasks: N
- Validation: PASSED
- Key points ↔ Criteria: ALIGNED ✓

Run: conductor run docs/plans/<slug>.yaml
```

---

## Golden Rules (NON-NEGOTIABLE)

1. **Code Reuse First**: Before creating new code, search for existing implementations (`grep -r "pattern" internal/`). Use directly, extend, or implement existing interfaces. New utilities require justification in `implementation.approach`.

2. **No Wrappers Without Value**: Only create adapter layers when they add real functionality (interface adaptation, lifecycle management). Direct usage preferred over unnecessary abstraction.

---

## Quick Reference: Common Failures

| Failure | Cause | Prevention |
|---------|-------|------------|
| Agent implements wrong thing | key_points incomplete | Write ALL requirements in key_points |
| QC fails despite working code | Criteria not in key_points | Derive criteria FROM key_points |
| Missing dependency | Data flow not traced | Build producer registry |
| Scope leak | Integration criterion in component | Classify criteria by type |
| Assumed behavior wrong | Didn't verify codebase | grep before claiming defaults |

---

## Version History

### v3.0.0 (2025-12-01)
- **Consolidated from 5500 lines to ~450 lines**
- **Added mandatory key_points ↔ success_criteria validation**
- Removed templates (LLMs understand structure)
- Removed reference files (inlined critical rules only)
- Streamlined phases with clear validation checkpoints

### v2.6.0 (2025-11-27)
- Added criteria derivation from key_points requirement
- Added behavioral fact verification

### v2.5.0 (2025-11-26)
- Added success criteria classification (CAPABILITY vs INTEGRATION)

### v2.3.0 (2025-11-24)
- Added data flow analysis phase
- Added prompt templates with dependency verification
- Added anti-pattern criteria auto-append
