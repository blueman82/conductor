# Dependency Graphing & Execution Control

How conductor calculates task execution order and what worktree groups actually do.

## Actual Execution Control (Conductor)

**Conductor uses `depends_on` fields ONLY** - not worktree groups.

### Wave Calculation (Kahn's Algorithm)

```
1. Build dependency graph from task.depends_on fields
2. Calculate in-degree for each task (number of dependencies)
3. Wave 1: All tasks with in-degree 0
4. Execute Wave 1 tasks in parallel (up to max_concurrency)
5. After wave completes, decrease in-degree of dependent tasks
6. Wave 2: Tasks that now have in-degree 0
7. Repeat until all tasks processed
```

**Example:**
```yaml
tasks:
  - task_number: 1
    depends_on: []        # Wave 1 (in-degree 0)

  - task_number: 2
    depends_on: [1]       # Wave 2 (in-degree 1, becomes 0 after Wave 1)

  - task_number: 3
    depends_on: []        # Wave 1 (in-degree 0, parallel with Task 1)

  - task_number: 4
    depends_on: [2, 3]    # Wave 3 (in-degree 2, waits for both)
```

**Execution:**
- Wave 1: Tasks 1 and 3 (parallel)
- Wave 2: Task 2
- Wave 3: Task 4

## Worktree Groups (Metadata Only)

**What they ARE:**
- Organizational labels for related tasks
- Human-readable grouping for git workflow
- Logged in `Wave.GroupInfo` for tracking
- Documentation for engineers using git worktrees

**What they are NOT:**
- ❌ Execution control mechanism
- ❌ Isolation enforcement
- ❌ Parallel/sequential directives
- ❌ Used by conductor's scheduler

### Code Evidence

```go
// internal/models/plan.go
type WorktreeGroup struct {
    GroupID     string // Organizational ID
    Description string // Human description
    ExecutionModel string // DEPRECATED: Not used
    Isolation      string // DEPRECATED: Not used
    Rationale      string // Why tasks grouped
}

// internal/executor/graph.go:153
g.Groups[taskNum] = tasks[i].WorktreeGroup  // Only for logging

// Wave execution ignores groups entirely
```

### Why Include Worktree Groups?

**For git workflow documentation:**

```yaml
worktree_groups:
  - group_id: "auth-chain"
    description: "Tasks 1→2→4 (auth flow)"
    rationale: "Related auth tasks for git worktree organization"
    setup_commands: |
      # Human workflow suggestion (not enforced)
      git worktree add ../wt-auth -b feature/auth
      # Implement tasks 1, 2, 4 here
      git merge feature/auth
```

Engineers can follow these git workflows, but conductor doesn't enforce them.

## Dependency Analysis for Plan Generation

When generating plans, analyze dependencies to create meaningful groups:

### 1. Build Dependency Graph

```
Task dependencies:
  1 → 2 → 4  (chain)
  3 (independent)
  5 depends on [2, 3]
```

### 2. Identify Patterns

**Dependency chains:**
- Tasks with sequential dependencies (1→2→4)
- Group for documentation: "auth-chain"

**Independent tasks:**
- No dependencies, no dependents (Task 3)
- Group for documentation: "docs-update"

**Integration points:**
- Multiple dependencies (Task 5 depends on 2 and 3)
- Group for documentation: "integration"

### 3. Document Groups (Not Control)

```yaml
worktree_groups:
  - group_id: "auth-chain"
    description: "Tasks 1→2→4 implement auth flow"
    rationale: "Sequential dependency chain"
    # Suggestion for git workflow:
    setup_commands: |
      git worktree add ../wt-auth -b feature/auth
      # Work on tasks 1, 2, 4 sequentially
      git merge feature/auth

  - group_id: "docs-update"
    description: "Task 3 updates documentation"
    rationale: "Independent of auth implementation"
    # Can work in parallel with auth-chain:
    setup_commands: |
      git worktree add ../wt-docs -b feature/docs
      # Work on task 3
      git merge feature/docs
```

**Conductor will still execute based on `depends_on`:**
- Wave 1: Tasks 1, 3 (parallel - both have in-degree 0)
- Wave 2: Task 2 (depends on 1)
- Wave 3: Task 4, 5 (4 depends on 2, 5 depends on 2 and 3)

## Best Practices for Groups

### Do Use Groups For:

1. **Documentation:**
   ```yaml
   description: "Tasks 1→2→4 implement authentication flow"
   ```

2. **Git workflow suggestions:**
   ```yaml
   setup_commands: |
     git worktree add ../wt-auth -b feature/auth
   ```

3. **Human organization:**
   ```yaml
   rationale: "Related database setup tasks grouped for clarity"
   ```

4. **Logging/tracking:**
   - Conductor logs which group each task belongs to
   - Helpful for debugging and understanding execution

### Don't Use Groups For:

1. **Execution control** - Use `depends_on` instead
2. **Forcing sequential execution** - Use dependency chains
3. **Isolation enforcement** - Conductor doesn't enforce
4. **Parallel execution** - Automatic based on dependencies

## Example: Correct Usage

```yaml
conductor:
  worktree_groups:
    - group_id: "setup"
      description: "Tasks 1-3 setup infrastructure"
      rationale: "Foundation tasks engineers should understand as related"
      setup_commands: |
        # Suggestion for git workflow
        git worktree add ../wt-setup -b feature/setup

    - group_id: "features"
      description: "Tasks 4-6 implement features"
      rationale: "Feature tasks that build on setup"
      setup_commands: |
        # Suggestion for git workflow
        git worktree add ../wt-features -b feature/implementation

plan:
  tasks:
    - task_number: 1
      worktree_group: "setup"  # Organizational label
      depends_on: []           # Wave 1

    - task_number: 2
      worktree_group: "setup"
      depends_on: [1]          # Wave 2 (actual execution control)

    - task_number: 3
      worktree_group: "setup"
      depends_on: []           # Wave 1 (parallel with Task 1)

    - task_number: 4
      worktree_group: "features"
      depends_on: [2]          # Wave 3 (waits for Task 2)
```

**Conductor execution:**
- Wave 1: Tasks 1, 3 (both setup group, but parallel via depends_on)
- Wave 2: Task 2 (setup group, sequential via depends_on)
- Wave 3: Task 4 (features group, waits via depends_on)

Groups label tasks, dependencies control execution.

## Validation

Conductor validates:
- ✅ Every task's `worktree_group` exists in `worktree_groups` list
- ✅ Dependency graph has no cycles
- ❌ Does NOT validate execution_model or isolation (ignored)

## Summary

**Execution control:** `depends_on` fields → Kahn's algorithm → waves

**Worktree groups:** Human organization + git workflow documentation

When generating plans:
1. Calculate dependencies correctly (critical for execution)
2. Add worktree groups for documentation (helpful for humans)
3. Never claim groups control execution (that's depends_on)
