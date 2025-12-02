# Package-Level Conflict Detection & Resolution

**Purpose:** Prevent race conditions during parallel task execution by detecting when multiple tasks modify the same Go package.

## The Problem: Test Contamination

When tasks modifying the same package execute in parallel:

```
Task 1: Creates internal/behavioral/models.go + models_test.go
Task 22: Creates internal/behavioral/patterns.go + patterns_test.go

Both running simultaneously:
├─ Task 1 QC runs: go test ./internal/behavioral/...
│  └─ Picks up Task 22's incomplete tests → FAILS
└─ Task 22 still writing patterns_test.go
```

**Result:** Task 1 gets RED verdict despite correct implementation.

## Detection Algorithm

### Step 1: Extract Package Paths

For each task, parse `files:` field to extract Go package paths:

```yaml
files: ["internal/behavioral/models.go", "internal/behavioral/doc.go"]
# → Package: internal/behavioral
```

**Algorithm:**
```
for each file in task.files:
  if file ends with .go:
    package_path = directory_path(file)
    add to task.packages
```

### Step 2: Group Tasks by Package

```
Package Map:
  internal/behavioral:
    - Task 1 (models.go, doc.go)
    - Task 22 (patterns.go, patterns_test.go)

  internal/parser:
    - Task 5 (yaml.go)
    - Task 6 (markdown.go)
```

### Step 3: Detect Conflicts

For each package with 2+ tasks:
```
if tasks[i].depends_on does NOT contain tasks[j]:
  AND tasks[j].depends_on does NOT contain tasks[i]:
    → CONFLICT DETECTED
```

## Resolution Strategies

### Strategy 1: Serialize Execution (Default)

Add dependency to later task:

```yaml
# BEFORE (conflict)
tasks:
  - task_number: 1
    files: ["internal/behavioral/models.go"]
    depends_on: []

  - task_number: 22
    files: ["internal/behavioral/patterns.go"]
    depends_on: []  # ← Will run in parallel with Task 1

# AFTER (serialized)
tasks:
  - task_number: 1
    files: ["internal/behavioral/models.go"]
    depends_on: []

  - task_number: 22
    files: ["internal/behavioral/patterns.go"]
    depends_on: [1]  # ← Added to prevent parallel execution
    # metadata:
    #   conflict_resolution: "Serialized with Task 1 (same package: internal/behavioral)"
```

**Pros:**
- Simple, guaranteed to work
- No test scoping required
- Clear execution order

**Cons:**
- Reduces parallelism
- May increase total execution time

### Strategy 2: Scope Test Commands

Keep parallel execution but isolate tests:

```yaml
# Task 1
test_commands:
  - "go test ./internal/behavioral/models_test.go -v"
  - "go test -run TestSession ./internal/behavioral/"
  - "go test -run TestBehavioralMetrics ./internal/behavioral/"
# Only tests Task 1's code, ignores patterns_test.go

# Task 22
test_commands:
  - "go test ./internal/behavioral/patterns_test.go -v"
  - "go test -run TestPatternDetector ./internal/behavioral/"
  - "go test -run TestIdentifyAnomalies ./internal/behavioral/"
# Only tests Task 22's code, ignores models_test.go
```

**Pros:**
- Preserves parallelism
- Faster overall execution

**Cons:**
- More complex test commands
- Requires knowing test names in advance
- Integration issues may be missed

### Strategy 3: Isolate with Subdirectories

Move tasks to separate subdirs temporarily:

```yaml
# Task 1
files:
  - "internal/behavioral/models/models.go"
  - "internal/behavioral/models/models_test.go"
test_commands:
  - "go test ./internal/behavioral/models/..."

# Task 22
files:
  - "internal/behavioral/patterns/patterns.go"
  - "internal/behavioral/patterns/patterns_test.go"
test_commands:
  - "go test ./internal/behavioral/patterns/..."

# Task 25 (Integration)
files: ["internal/behavioral/behavioral.go"]
implementation:
  approach: "Merge models/ and patterns/ into parent package"
depends_on: [1, 22]
```

**Pros:**
- Complete isolation
- Clean package-level tests
- Natural integration task

**Cons:**
- Extra integration task needed
- More complex file structure
- May require import path updates

## Decision Matrix

| Scenario | Recommended Strategy | Reason |
|----------|---------------------|--------|
| 2-3 tasks in package | Strategy 1 (Serialize) | Simple, low overhead |
| 4+ tasks in package | Strategy 2 (Scope tests) | Better parallelism |
| Complex dependencies | Strategy 1 (Serialize) | Clearer execution order |
| Independent modules | Strategy 3 (Subdirs) | Natural package boundaries |

## Implementation in Skill

### During Phase 2: Dependency Graph Analysis

```
1. Extract packages from all tasks
2. Group tasks by package
3. For each package group:
   a. Check for existing dependencies
   b. If no deps AND 2+ tasks:
      - Apply Strategy 1 (default)
      - Add comment explaining resolution
      - Update depends_on field
   c. Log conflict resolution decision
```

### Example Output

```yaml
tasks:
  - task_number: 1
    name: "Create behavioral models"
    files: ["internal/behavioral/models.go", "internal/behavioral/models_test.go"]
    depends_on: []

  - task_number: 22
    name: "Implement pattern detection"
    files: ["internal/behavioral/patterns.go", "internal/behavioral/patterns_test.go"]
    depends_on: [1]  # ← Auto-added by conflict detection
    # Package conflict resolution: Serialized with Task 1
    # Both tasks modify package: internal/behavioral
```

## Validation Checklist

During Phase 6 validation:

- [ ] All same-package tasks have explicit dependencies OR scoped tests
- [ ] Test commands in parallel tasks won't collide
- [ ] Conflict resolution documented in comments or metadata
- [ ] Parallelism optimized where safe
- [ ] Integration tasks added if using subdir strategy

## When to Skip Conflict Detection

**Don't apply for:**
- Single-task packages (no conflict possible)
- Tasks with existing dependencies (already serialized)
- Non-Go projects (different test isolation model)
- Test-only tasks (no production code conflicts)

## Language-Specific Notes

### Go
- Package-level test execution: `go test ./path/...`
- File-specific: `go test ./path/file_test.go`
- Function-specific: `go test -run TestName`

### Python
- Less common: Python imports are module-based
- Use `-k` flag for test scoping: `pytest -k test_models`

### TypeScript/JavaScript
- Jest: `jest models.test.ts` or `jest -t "test name"`
- Similar package-level concerns

### Rust
- Cargo tests are crate-level
- Use `--test test_name` for specific test files

## Real-World Example

From Agent Watch integration (conductor v2.5.2):

```yaml
# Initial plan (CAUSED RACE CONDITION)
plan:
  tasks:
    - task_number: 1
      files: ["internal/behavioral/models.go", "internal/behavioral/doc.go"]
      test_commands: ["go test ./internal/behavioral/..."]

    - task_number: 22
      files: ["internal/behavioral/patterns.go"]
      test_commands: ["go test ./internal/behavioral/..."]

# Result:
# Task 1 QC (14:14:27) → Saw Task 22's failing tests → RED verdict
# Task 22 completion (14:15:45) → All tests pass → GREEN verdict
# Task 1's code was actually correct, RED was false positive

# CORRECTED plan (with conflict detection)
plan:
  tasks:
    - task_number: 1
      files: ["internal/behavioral/models.go", "internal/behavioral/doc.go"]
      test_commands: ["go test ./internal/behavioral/..."]
      depends_on: []

    - task_number: 22
      files: ["internal/behavioral/patterns.go"]
      test_commands: ["go test ./internal/behavioral/..."]
      depends_on: [1]  # ← Auto-added by conflict detection
      # Prevents parallel execution, eliminates race condition
```

## Summary

Package conflict detection is a **proactive quality gate** that:

1. ✅ Prevents false-negative QC verdicts
2. ✅ Eliminates test contamination during parallel execution
3. ✅ Preserves execution determinism
4. ✅ Documents conflict resolution decisions
5. ✅ Optimizes parallelism within safety constraints

**Default to Strategy 1 (serialization)** unless you have specific reasons to use scoped tests or subdirectories.
