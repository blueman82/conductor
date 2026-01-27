---
name: implementation-planner
description: Generate comprehensive implementation plans for features. Use when user requests "help me implement X", "create a plan for X", "break down feature X", "how should I build X", or asks for detailed implementation guidance. Activates for planning requests, not exploratory design discussions.
allowed-tools: Read, Bash, Glob, Grep, Write, TodoWrite, Task
---

# Implementation Planner v4.2

Generate conductor-compatible YAML plans. **Do NOT activate for:** questions, debugging, code reviews.

## Workflow

1. **Discover** → Launch Explore agents for codebase, patterns, existing implementations
2. **Design** → Break into tasks, map dependencies, build data flow registry
3. **Detail** → Write SPECIFIC success_criteria and key_points (see examples below)
4. **Classify** → Simple task vs Complex task (determines what to embed)
5. **Generate** → Output YAML with all required fields
6. **Validate** → `conductor validate <plan>.yaml`

---

## Discovery Phase

Launch Explore agents IN PARALLEL (this is step 1!):
```
Agent 1: "Explore codebase structure, stack, existing patterns"
Agent 2: "Search for existing implementations of <feature>"
Agent 3: "Explore available agents in ~/.claude/agents"
```

### Data Flow Registry

Build producer/consumer map to ensure correct dependencies:

```yaml
# PRODUCERS: Task 2 → memories table, Task 3 → search methods
# CONSUMERS: Task 4 → [3], Task 8 → [3, 7]
# VALIDATION: All consumers depend_on producers ✓

data_flow_registry:
  producers:
    search_methods:
      - task: 3
        description: "Creates search_vector, search_fts, search_hybrid"
  consumers:
    search_methods:
      - task: 4
        description: "HybridStore uses search methods"
```

---

## THE CORE RULE: Specificity

**Agents only see task-level fields.** If context isn't in the task description, success_criteria, or key_points, the agent won't know it.

### Bad vs Good Success Criteria

| BAD (vague) | GOOD (specific) |
|-------------|-----------------|
| "Tables created" | "memories table created with columns: id, content, content_hash, memory_type, namespace, confidence, importance, source_file, chunk_index, start_line, end_line, created_at, last_accessed, access_count, tags" |
| "CRUD methods work" | "add_memory() inserts to memories + memories_vec + memories_fts in single transaction" |
| "Update types" | "Document dataclass field doc_type renamed to memory_type to match SQLite schema" |
| "Search works" | "search_vector(embedding, n_results, where) uses vec_distance_cosine" |
| "Tests pass" | "Tests cover CRUD: add_memory, get_memory, update_memory, delete_memory" |

### Bad vs Good Key_points

| BAD (abstract) | GOOD (actionable) |
|----------------|-------------------|
| "Update ValidationLoop" | "ValidationLoop uses SQLiteStore.update_memory() for confidence adjustments" |
| "Update tools" | "memory_tools.py: memory_store→add_memory(), memory_recall→search_hybrid(), memory_forget→delete_memory()" |
| "Handle errors" | "On migration failure, restore from .backup files created at start" |

### The Count Rule

**If success_criteria lists N items, key_points needs N entries.**

Example: If success_criteria says "Tests cover: add_memory, get_memory, update_memory, delete_memory, list_memories, count_memories, search_vector, search_fts, search_hybrid, cache operations, golden rules, edges" (12 items), then key_points needs 12 entries covering each.

---

## Task Complexity Classification

### Simple Tasks (NO `<mandatory_principles>` needed)

- Adding/removing dependencies in pyproject.toml
- Deleting files
- Updating __init__.py exports
- Removing fields from config classes
- Simple find-replace refactors

### Complex Tasks (NEED `<mandatory_principles>` + embedded context)

- New method implementations
- Schema changes with exact SQL
- Migration scripts with field mappings
- Refactoring with business logic

---

## Mandatory Principles Reference

Use these in `<mandatory_principles>` tags for complex tasks:

### Python
```
ENGINEERING: YAGNI, KISS, DRY, Fail Fast, Single Source of Truth, Law of Demeter.
PYTHONIC: PEP 8 naming (snake_case functions, PascalCase classes), full type hints (Pyright strict),
  EAFP over LBYL (try/except not if-checks), context managers for resources, f-strings,
  explicit imports (no star imports), PEP 257 docstrings for public APIs only.
CODE REDUCTION: No helpers for one-time ops, no premature abstractions, delete unused code completely.
```

### Go
```
ENGINEERING: YAGNI, KISS, DRY, Fail Fast, Single Source of Truth, Law of Demeter.
GO IDIOMS: Accept interfaces return structs, errors are values (handle or return),
  table-driven tests, short variable names in small scopes, package-level organization.
CODE REDUCTION: No empty structs (Type{}), no unused variables (_ = x), no TODO comments.
```

### TypeScript
```
ENGINEERING: YAGNI, KISS, DRY, Fail Fast, Single Source of Truth.
TS IDIOMS: Strict mode, explicit return types, discriminated unions over type assertions,
  const assertions, exhaustive switch checks, no any.
```

---

## Embedded Context by Task Type

### For Database Tasks
- Include exact SQL CREATE TABLE statements
- List all columns with types and constraints
- Include all indexes

### For Migration Tasks
- Include field mapping table (old name → new name)
- Include rollback plan
- Include verification queries

### For API/Method Tasks
- Include method signatures with full type hints
- Include example SQL queries if applicable
- Include error handling requirements

### For Test Tasks
- List each test case by name
- Specify test data (e.g., "[0.1] * 1024 for mock embeddings")
- Specify assertions

---

## Critical Rules

| Rule | Rationale |
|------|-----------|
| **Terminology match** | Conductor rubric validates that key_points terms appear in success_criteria |
| **Data flow deps** | If task B uses function from task A, B must `depends_on: [A]` |
| **Verify before claiming** | `grep` to confirm existing behavior before writing key_points |
| **Verify file paths** | `ls` to confirm files exist before adding to `files[]` |

**Auto-append to ALL tasks' success_criteria:**
- No TODO comments in production code
- No placeholder code

---

## Dependency Patterns

### Creation Dependencies (Data Flow)
If task B **uses** something task A creates → B `depends_on: [A]`

### Removal Dependencies (CRITICAL - often missed!)

| Removal Task | Must Depend On |
|--------------|----------------|
| **Delete a file** | ALL tasks that update imports from that file |
| **Remove a library** | ALL tasks that use that library (including migration scripts!) |
| **Remove a config field** | ALL tasks that reference that field |
| **Remove API endpoint** | ALL tasks that update consumers of that endpoint |

**Rule:** Before removing X, identify ALL consumers. Each consumer update becomes a prerequisite task. Use `grep` to find them.

**Example bug we caught:**
- Task 11 "Delete chroma_store.py" had `depends_on: [4, 8]`
- But ChromaStore was imported in 5 more files (validation/, __main__.py, tools/)
- Fixed to: `depends_on: [4, 5, 8, 9, 10]`

- Task 14 "Remove chromadb dependency" had `depends_on: [11, 13]`
- But migration script (Task 12) needs chromadb to read old data
- Fixed to: `depends_on: [11, 12, 13]`

---

## REFERENCE: YAML Schema

### Root Structure

```yaml
conductor:
  worktree_groups:
    - group_id: "foundation"
      tasks: [1, 2, 3]
      rationale: "Sequential dependency chain"

planner_compliance:
  planner_version: "4.0.0"
  strict_enforcement: true
  required_features: [dependency_checks, test_commands, success_criteria, data_flow_registry]

data_flow_registry:
  producers: {}
  consumers: {}

plan:
  metadata:
    feature_name: "Name"
    created: "YYYY-MM-DD"
    target: "Goal"
  context:
    framework: "Python 3.13"
    test_framework: "pytest"
  tasks: []
```

### Task Structure

```yaml
- task_number: "1"
  name: "Task name"
  agent: "python-pro"
  files: ["src/module/file.py"]
  depends_on: []

  success_criteria:
    - "Specific, verifiable criterion with exact names"
    - "Another specific criterion"
    - "No TODO comments"

  test_commands:
    - "cd /path && uv run pytest tests/test_file.py -v"

  runtime_metadata:
    dependency_checks:
      - command: "uv run python -c 'import module'"
        description: "Verify import works"
    documentation_targets: []

  description: |
    <mandatory_principles>
    ENGINEERING: YAGNI, KISS, DRY, Fail Fast, Single Source of Truth.
    PYTHONIC: Full type hints, EAFP (try/except), context managers.
    </mandatory_principles>

    <task_description>What to implement with full context.</task_description>

  implementation:
    approach: |
      Strategy and decisions.
    key_points:
      - point: "Exact function/method name"
        details: "What it does and how"
        reference: "src/file.py:method_name"

  code_quality:
    python:
      full_quality_pipeline:
        command: "cd /path && uv run black src/ && uv run isort src/ && uv run ruff check src/ --fix && uv run mypy src/"
        exit_on_failure: true

  commit:
    type: "feat"
    message: "Description of change"
    files: ["src/**"]
```

---

## REFERENCE: Code Quality Commands

### Python
```yaml
code_quality:
  python:
    full_quality_pipeline:
      command: "cd /path && uv run black . && uv run isort . && uv run ruff check . --fix && uv run mypy ."
      exit_on_failure: true
```

### Go
```yaml
code_quality:
  go:
    full_quality_pipeline:
      command: "gofmt -w . && go vet ./... && go test ./..."
      exit_on_failure: true
```

---

## Validation Checklist

Before running `conductor validate`:

```
□ Every success_criteria item is SPECIFIC (exact names, columns, methods)
□ Every key_point has corresponding success criterion (SAME terminology)
□ Key_points count matches success_criteria detail level
□ Complex tasks have <mandatory_principles> + embedded context
□ Simple tasks are lean (no unnecessary principles)
□ Data flow: consumers depend_on all producers
□ All file paths verified to exist
□ Code quality commands use project tooling
```

---

## Common Failures

| Failure | Prevention |
|---------|------------|
| Agent implements wrong thing | Be SPECIFIC in success_criteria |
| QC fails despite working code | Use IDENTICAL terms in key_points and success_criteria |
| Missing methods | Count items in success_criteria, ensure key_points matches |
| Context lost | Embed SQL/mappings/signatures in task description |
| Over-engineered simple tasks | Don't add principles to config/delete tasks |
| Plan too large | ~2000 lines max, split at worktree boundaries |
