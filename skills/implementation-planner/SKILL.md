---
name: implementation-planner
description: Generate conductor-compatible YAML plans. Use when user requests "help me implement X", "create a plan for X", or asks for implementation guidance. NOT for questions, debugging, or code reviews.
allowed-tools: Read, Write, Edit, Bash, Glob, Grep, Task, mcp__mcp-exec__*
---

# Implementation Planner v5.1

Generate conductor-compatible YAML plans with **mandatory** tool integrations.

**Output location:** `docs/plans/<feature-name>.yaml`

---

## PHASE 1: Context Gathering (MANDATORY)

Execute ALL of these before designing:

### 1.1 Activate Serena + Recall Theo Context

```javascript
// EXECUTE via mcp__mcp-exec__execute_code_with_wrappers with wrappers: ["serena", "theo"]

// Activate project
await serena.activate_project({ project: "/path/to/project" });

// Get project memories
const memories = await theo.memory_recall({
  query: "architecture patterns decisions for <project>",
  namespace: "project:<name>",
  limit: 5
});
console.log("Project context:", memories.data?.memories?.map(m => m.content.slice(0, 200)));

// Check for ./check script
const hasCheck = await serena.find_file({ file_mask: "check", relative_path: "." });
console.log("Has ./check:", hasCheck);
```

### 1.2 Explore Codebase with Serena

```javascript
// EXECUTE via mcp__mcp-exec__execute_code_with_wrappers with wrappers: ["serena"]

// List structure
const structure = await serena.list_dir({ relative_path: ".", recursive: false });
console.log("Root:", structure);

// Find relevant files
const symbols = await serena.get_symbols_overview({ relative_path: "TARGET_FILE" });
console.log("Symbols:", symbols);

// Search for patterns
const patterns = await serena.search_for_pattern({ substring_pattern: "PATTERN" });
console.log("Found:", patterns);
```

### 1.3 Check Available Agents

```bash
ls ~/.claude/agents/*.md | head -20
```

---

## PHASE 2: Design

### 2.1 Data Flow Registry

Build producer/consumer map:
```yaml
data_flow_registry:
  producers:
    config: [{task: 1, description: "Creates X"}]
  consumers:
    config: [{task: 2, description: "Uses X"}]
```

**Rule:** If task B uses something task A creates → B `depends_on: [A]`

### 2.2 Task Breakdown

For each task identify:
- Files to modify
- Dependencies (data flow)
- Success criteria (SPECIFIC names, not vague)
- Test commands

---

## PHASE 3: Detail Tasks (CRITICAL)

### 3.0 Task Creation Loop (MANDATORY - BLOCKING)

For **EACH** task you create, you MUST follow this sequence:

```
1. IDENTIFY: List files this task will modify
2. READ: Read EVERY file NOW using Read tool
   - Verify the file exists
   - Verify APIs/methods you'll call exist
   - Note actual signatures (don't assume)
3. ONLY THEN: Write the task description
```

**BLOCKING RULE:** Do NOT write a task's `description` field until you have read ALL files in its `files` array in THIS session.

**Why this works:**
- Verification happens at point of need (when creating task)
- Cannot skip because it's part of the creation workflow
- Files are known at task creation time

**Example (CORRECT):**
```yaml
# Step 1: I identify files
files: ["src/main.py", "src/client.py"]

# Step 2: I READ both files NOW
# [Read src/main.py] → Found Agent created at line 142
# [Read src/client.py] → Found start() initializes at line 320

# Step 3: NOW I write the description with verified info
description: |
  Modify Agent initialization at main.py:142...
  Modify start() at client.py:320...
```

**Example (WRONG - causes broken plans):**
```yaml
# I write files and description together without reading
files: ["src/main.py", "src/client.py"]
description: |
  Modify Agent initialization...  # ASSUMED, not verified - WILL FAIL
```

---

### 3.1 The Specificity Rule

**Agents only see task-level fields.** Embed ALL context in description.

| BAD | GOOD |
|-----|------|
| "Tables created" | "users table with columns: id, email, created_at" |
| "Search works" | "search_vector(embedding, n_results) uses vec_distance_cosine" |
| "Tests pass" | "Tests: test_create, test_read, test_update, test_delete" |

### 3.2 Mandatory Principles (embed in EVERY complex task)

```
<mandatory_principles>
ENGINEERING: YAGNI, KISS, DRY, Fail Fast, SSOT, Law of Demeter.
PYTHON:
  - PEP 8: snake_case functions, PascalCase classes, UPPER_CASE constants
  - Full type hints on ALL signatures (Pyright strict)
  - Explicit imports (no star imports, no lazy imports, no TYPE_CHECKING)
  - EAFP over LBYL (try/except not if-checks)
  - Context managers for resources
  - f-strings, list comprehensions where readable
  - No mutable default arguments
GO:
  - Accept interfaces, return structs
  - Errors are values (handle or return, never ignore)
  - Table-driven tests
  - Early returns over nested if/else
TYPESCRIPT:
  - Strict mode, explicit return types
  - Discriminated unions over type assertions
  - Optional chaining (?.), nullish coalescing (??)
  - No any type
CODE REDUCTION:
  - No helpers for one-time ops
  - No premature abstractions (3 similar lines > abstraction)
  - Delete unused code completely (no _unused renames)
QUALITY: Run ./check --fix before committing (or project equivalent)
</mandatory_principles>
```

### 3.3 Task Classification

**Simple tasks** (config changes, deletions, imports): No principles needed
**Complex tasks** (new code, refactoring): MUST include `<mandatory_principles>`

### 3.4 Count Rule

If success_criteria has N items → key_points needs N matching entries.

---

## PHASE 4: Generate YAML

### 4.1 Required Structure

```yaml
conductor:
  worktree_groups:
    - group_id: "name"
      tasks: [1, 2]
      rationale: "Why grouped"

planner_compliance:
  planner_version: "5.0.0"
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
    project_root: "/absolute/path"
  context:
    framework: "Python 3.11, FastAPI"
    test_framework: "pytest"
    quality_tool: "./check --fix"
  tasks: []
```

### 4.2 Task Template

```yaml
- task_number: "1"
  name: "Task name"
  agent: "python-pro"  # or fastapi-pro, test-automator, etc.
  files: ["path/to/file.py"]
  depends_on: []

  success_criteria:
    - "Specific criterion with exact names"
    - "./check --fix passes"
    - "No TODO comments"

  test_commands:
    - "cd /path && ./check --fix"
    - "cd /path && uv run pytest tests/test_x.py -v"

  runtime_metadata:
    dependency_checks:
      - command: "grep 'pattern' file"
        description: "Verify X"
    documentation_targets: []

  description: |
    <mandatory_principles>
    [Full principles from 3.2 above]
    </mandatory_principles>

    <task_description>
    What to implement with EXACT code snippets.

    ```python
    def function_name(param: Type) -> ReturnType:
        """Docstring."""
        implementation
    ```

    Run ./check --fix after changes.
    </task_description>

  implementation:
    approach: "Strategy"
    key_points:
      - point: "function_name"
        details: "What it does"
        reference: "file.py:line"

  code_quality:
    python:
      full_quality_pipeline:
        command: "cd /path && ./check --fix"
        exit_on_failure: true

  commit:
    type: "feat"
    message: "description"
    files: ["path/**"]
```

---

## PHASE 5: Validate

```bash
cd /project && conductor validate docs/plans/<name>.yaml
```

Must pass:
- Rubric validation
- Data flow registry validation
- No circular dependencies
- All agents available
- All task dependencies valid

---

## Dependency Patterns

### Creation: B uses A's output → B depends_on A

### Removal (often missed!):
| Removing | Must depend on |
|----------|----------------|
| Delete file | ALL tasks updating imports from it |
| Remove library | ALL tasks using it (including migrations!) |
| Remove config | ALL tasks referencing it |

**Use grep to find all consumers before removal.**

---

## Common Failures

| Failure | Fix |
|---------|-----|
| Wrong API called | READ files during task creation (Phase 3.0), verify signatures |
| Integration point missing | READ files during task creation (Phase 3.0), confirm hooks exist |
| Agent does wrong thing | Be SPECIFIC in success_criteria |
| QC fails | Use IDENTICAL terms in key_points and success_criteria |
| Context lost | Embed SQL/signatures/code in description |
| Wrong location | Output to docs/plans/, NOT .conductor/ |
| Missing ./check | Add to EVERY task's test_commands AND description |

---

## Checklist Before Generating

```
□ Serena activated, codebase explored
□ Theo memories recalled for project
□ For EACH task: Read files BEFORE writing description (Phase 3.0)
□ Method signatures VERIFIED not assumed (Phase 3.0)
□ Integration points confirmed to exist (Phase 3.0)
□ ./check --fix in every Python task
□ Full <mandatory_principles> in complex tasks
□ success_criteria are SPECIFIC (exact names)
□ key_points count matches success_criteria count
□ Data flow: consumers depend_on producers
□ Output path: docs/plans/<name>.yaml
□ conductor validate passes
```
