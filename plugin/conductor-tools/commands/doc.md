---
allowed-tools: Read, Bash(ls*), Glob, Grep, Bash(git status*), Bash(git log*), Bash(fd*), Write, TodoWrite, AskUserTool
argument-hint: "feature description"
description: Generate comprehensive implementation plan with detailed tasks, testing strategy, and commit points
---

# Doc - Comprehensive Implementation Plan Generator

Create a detailed, step-by-step implementation plan for $ARGUMENTS. The plan should enable a skilled engineer with zero context about this codebase to successfully implement the feature.

## Phase 1: Codebase Context Analysis

**First, discover available agents:**

**Execute agent discovery first:**

Run the command:
```bash
fd '\.md$' ~/.claude/agents --type f
```

Extract agent names by removing the path and `.md` extension. For example:
- `/Users/harrison/.claude/agents/golang-pro.md` → `golang-pro`
- `/Users/harrison/.claude/agents/test-automator.md` → `test-automator`

Store these agent names for assignment to tasks. If no agents are found, use `general-purpose` as the default.

**Then, determine if this is an existing project or greenfield:**

Use `Glob` tool with pattern `"*"` to check files in the current directory. If empty or only contains minimal files (README, .git), this is a **greenfield project** - proceed to Phase 1B.

If an existing codebase exists, thoroughly analyze it:

1. **Project Structure Analysis**
   - Use `Bash` with `ls -la` to understand directory structure
   - Use `Glob` tool to find specific file patterns (e.g., `"**/*.test.*"`, `"src/**/*.js"`)
   - Identify key directories: src/, tests/, docs/, config/, etc.
   - Locate test directories and understand testing patterns

2. **Technology Stack Discovery**
   - Use `Read` tool to examine package.json, requirements.txt, Gemfile, go.mod, or similar
   - Identify frameworks and libraries in use
   - Check tsconfig.json, .eslintrc, pytest.ini, or other config files

3. **Architecture Pattern Recognition**
   - Use `Grep` tool to search for existing similar features
   - Identify architectural patterns (MVC, Clean Architecture, etc.)
   - Find service layers, repositories, controllers, models
   - Understand dependency injection patterns if present

4. **Testing Infrastructure**
   - Use `Glob` tool to locate test files and understand naming conventions
   - Use `Read` tool to examine test files and identify test frameworks (Jest, PyTest, RSpec, etc.)
   - Find test utilities, fixtures, factories, mocks
   - Review existing test examples for patterns

5. **Code Quality Standards**
   - Use `Glob` tool to check for linters, formatters (.eslintrc, .prettierrc, etc.)
   - Use `Read` and `Grep` tools to review existing code for naming conventions
   - Identify error handling patterns
   - Note logging/observability patterns

6. **Git History Context**
   - Run `git log --oneline -20` to see recent commit patterns
   - Note commit message style and size
   - Use `Glob` or `Read` to check if there's a CONTRIBUTING.md or similar

## Phase 1B: Greenfield Project Setup (if applicable)

If no existing codebase detected, the plan must include initial project setup:

1. **Technology Stack Decisions**
   - Document language/framework choice and rationale
   - List required dependencies and their purposes
   - Specify project structure to create
   - Define testing framework and approach

2. **Initial Setup Tasks**
   - Directory structure to create (src/, tests/, docs/, etc.)
   - Config files needed (package.json, tsconfig.json, pytest.ini, etc.)
   - Testing framework setup and configuration
   - Linting/formatting setup (.eslintrc, .prettierrc, etc.)
   - CI/CD configuration (if applicable)

3. **Foundation Commits**
   - First commit: Initial project scaffold with basic directory structure
   - Second commit: Configuration files and tooling setup
   - Third commit: README and documentation structure with project overview

4. **Development Environment**
   - Required tools and versions to install
   - Environment variables needed
   - Setup verification steps
   - **CRITICAL**: Type checking configuration (mypy/tsc/go vet)
   - **CRITICAL**: Test framework configuration (pytest/jest/go test)
   - **CRITICAL**: Import path validation (PYTHONPATH, module structure)
   - Linting and formatting tools setup
   - Pre-commit hooks configuration

## Phase 1C: Analyze Task Dependency Structure

After identifying all tasks for the feature, analyze the dependency relationships to determine optimal worktree grouping:

1. **Build Dependency Graph**
   - For each task, note any "Depends on" relationships
   - Create a directed graph where edges represent dependencies (Task A → Task B means B depends on A)

2. **Identify Connected Components (Dependency Chains)**
   - Find groups of tasks where dependencies create a chain or network
   - Example: Task 1 → Task 2 → Task 3 forms a single connected component
   - Example: Task 4 → Task 5 ← Task 6 (where Task 5 depends on both 4 and 6) forms another component
   - **Each connected component will share ONE worktree** to maintain dependency order

3. **Identify Independent Tasks**
   - Find tasks with no "Depends on" field AND no other tasks that depend on them
   - These tasks can be executed in parallel without conflict
   - **Each independent task gets its OWN worktree** to enable maximum parallelism

4. **Assign Worktree Groups**
   - Connected components: `chain-1`, `chain-2`, `chain-3`, etc.
   - Independent tasks: `independent-1`, `independent-2`, `independent-3`, etc.
   - Each group will have its own branch: `feature/<feature-slug>/[group-id]`

**Worktree Strategy Benefits**:
- **Parallelism**: Independent tasks execute simultaneously in separate worktrees
- **Dependency Safety**: Tasks in a chain execute sequentially in shared worktree, preventing dependency violations
- **Resource Efficiency**: Minimize worktree count by grouping dependent tasks
- **Clear Isolation**: Each worktree group has its own branch, preventing merge conflicts during parallel development

## Phase 1D: Streaming Generation Strategy

**CRITICAL: You will generate this plan INCREMENTALLY, not all at once.**

Large plans (15+ tasks) will exceed token limits if generated as a single monolithic task. You MUST break the generation process itself into manageable chunks.

### Use TodoWrite to Break Down Generation

Before starting to write the plan, create a todo list that breaks generation into phases:

```
Example Todo List:
- [ ] Create docs/plans directory structure
- [ ] Write plan header, metadata, context, worktree groups, prerequisites (~200-400 lines)
- [ ] Generate tasks for first worktree group (monitor line count actively)
- [ ] Check line count - if >1900, complete file and create next file
- [ ] Generate tasks for next worktree group (continue monitoring)
- [ ] Generate testing strategy, commit strategy, pitfalls sections
- [ ] Generate resources and validation checklist
- [ ] Create index file (README.md) if multiple files exist
- [ ] Confirm completion to user with metrics
```

### Active Line Count Monitoring

**This is NOT a passive check - actively monitor as you generate:**

1. **Initialize tracking variables**:
   - `current_file_line_count = 0`
   - `current_file_number = 1`
   - `tasks_written = 0`
   - `total_tasks = [number identified in Phase 1C]`

2. **After writing EACH section** (header, each task, each support section):
   - Increment `current_file_line_count` by lines just written
   - Check: `if current_file_line_count > 1900 AND tasks_written < total_tasks`
   - If true: **STOP and split to new file NOW**

3. **When splitting**:
   - Complete current task fully (never split mid-task)
   - Write current buffer to: `docs/plans/<feature-slug>/plan-0N-<phase-name>.md`
   - Create new file: `plan-0(N+1)-<next-phase>.md`
   - Reset: `current_file_line_count = 0`
   - Continue with remaining content

### Write Incrementally, Don't Accumulate

**WRONG approach** ❌:
- Generate all tasks in memory → accumulate huge text → write one file → token limit exceeded

**RIGHT approach** ✅:
- Write header → check count → write task 1 → check count → write task 2 → check count
- When count > 1900: write current file → create next file → continue
- Each file is written as you go, not accumulated in memory

### Example: Generating a 25-Task Plan

**Step 1: Create todo list** (as shown above)

**Step 2: Execute with active monitoring**:
```
[Starting plan-01-foundation.md]
Writing header & context... (current: 250 lines)
Writing worktree groups... (current: 350 lines)
Writing prerequisites... (current: 450 lines)
Writing Task 1... (current: 620 lines)
Writing Task 2... (current: 790 lines)
Writing Task 3... (current: 960 lines)
Writing Task 4... (current: 1130 lines)
Writing Task 5... (current: 1300 lines)
Writing Task 6... (current: 1470 lines)
Writing Task 7... (current: 1640 lines)
Writing Task 8... (current: 1810 lines)
Writing Task 9... (current: 1980 lines) ⚠️ LIMIT APPROACHING

DECISION: Current file has 1980 lines, at worktree group boundary
→ Complete plan-01-foundation.md with tasks 1-9
→ Create plan-02-integration.md for remaining tasks

[Starting plan-02-integration.md]
Reset counter (current: 0 lines)
Writing Task 10... (current: 170 lines)
Writing Task 11... (current: 340 lines)
...continues until all tasks complete...
```

**Step 3: Create index** (if multiple files created)

### Key Principles

1. **Generation is multi-step** - Not a single "generate plan" task
2. **Active monitoring** - Check line count after every section
3. **Write as you go** - Don't accumulate in memory
4. **Split proactively** - At 1900 lines, not 3000 lines
5. **Use task boundaries** - Never split mid-task

## Phase 2: Plan Generation

Create a comprehensive implementation plan where `<feature-slug>` is derived from the feature description (e.g., "user-authentication", "payment-integration").

Output structure depends on task count:
- **1-15 tasks**: Single file `docs/plans/<feature-slug>.md`
- **16+ tasks**: Directory `docs/plans/<feature-slug>/` with split files

### Plan Format Alignment

The implementation plan follows the format specified in [Plan Format Guide](../docs/plan-format.md).

**Conductor-parsed metadata** (extracted by conductor's parsers):
- **Core fields**: Task number, name, files, depends_on, estimated_time, agent, status
- **Phase 2A fields**: worktree_group (for task organization)

**Human-developer sections** (included in task prompt for context):
- **test_first**: TDD guidance with test structure, mocks, fixtures, assertions
- **implementation**: Approach, code structure, key points, integration details
- **verification**: Manual and automated testing steps
- **commit**: Commit message and strategy

**Worktree groups**: Used for organizing tasks into logical execution groups for parallel development. This is organizational metadata - conductor tracks it but doesn't enforce isolation or execution order based on groups.

The plan MUST include these sections:

## Phase 2A: Objective Plan Splitting with Conductor Auto-Discovery

When generating task breakdowns, implement **metric-based plan splitting** to prevent files from becoming unwieldy while maintaining complete task details. Split files enable Conductor's multi-file orchestration.

**IMPORTANT**: This section describes HOW to split when you detect the need. Phase 1D describes the ACTIVE MONITORING process you must follow during generation.

### Line Count Tracking (Active, Not Passive)

**CONTINUOUSLY track** the number of lines as you write each section. This is NOT a one-time check at the end - it's ongoing monitoring during generation.

**Implementation (active monitoring):**
- Initialize `line_count = 0` when starting a new plan file
- **After writing each section** (header, task, support section): Increment `line_count`
- Track `tasks_detailed` (number of tasks fully written) vs `total_tasks` (total tasks identified)
- **Check IMMEDIATELY**: Is `line_count > 1900`? If yes, prepare to split

### Active Monitoring Logic (Not Passive Trigger)

This is NOT a trigger you check once - it's a loop you execute continuously:

```
WHILE generating plan content:
  Write next section (header, task, etc.)
  current_line_count += lines_just_written

  IF (current_line_count > 1900 AND tasks_remaining > 0 AND at_group_boundary):
    STOP writing to current file
    WRITE current file to disk
    CREATE next file: plan-0N-<phase>.md
    RESET current_line_count = 0
    CONTINUE with remaining content

  IF (all_tasks_written AND all_sections_complete):
    WRITE final file
    CREATE index if multiple files
    BREAK loop
```

**Critical rules:**
- **1900-2000 lines is the target range** - no subjective "feels too long" judgment
- **Never split mid-task** - always complete the current task's full detail before splitting
- **Split at worktree group boundaries** - natural organizational boundaries for splitting
- **Only split if work remains** - don't create empty files
- **Write files incrementally** - don't accumulate everything in memory first

### Split Strategy (Phase 2A Conductor Format)

When the trigger condition is met, create numbered plan files in Phase 2A format for Conductor auto-discovery:

```
docs/plans/feature-name/
├── plan-01-phase-name.md          (2000 lines, initial phase)
├── plan-02-phase-name.md          (1800 lines, next phase)
├── plan-03-phase-name.md          (1200 lines, final phase)
└── README.md                        (index with cross-references and overview)
```

**File naming convention (Phase 2A):**
- `plan-NN-<descriptive-phase-name>.md` where NN is 01, 02, 03, etc.
- Use descriptive phase names that indicate content (e.g., `database`, `api`, `testing`)
- This format enables Conductor's auto-discovery: `conductor run docs/plans/feature-name/`
- Example: `plan-01-database.md`, `plan-02-api-implementation.md`, `plan-03-testing.md`

**Why Phase 2A naming:**
- Conductor automatically discovers files matching `plan-*.md` pattern
- Sequential numbering (01, 02, 03) ensures proper load order
- Descriptive names make file purposes clear without worktree group IDs
- Cross-file dependencies work across all numbered plan files
- Index file (README.md) documents the multi-file orchestration

### Metrics Tracked During Generation

Track these objective metrics throughout plan generation:

1. **line_count**: Total lines written to current plan file
2. **tasks_completed**: Number of tasks fully detailed in current file
3. **total_tasks**: Total number of tasks identified in Phase 1C
4. **current_worktree_group**: Which group is currently being documented
5. **worktree_group_boundaries**: List of group transitions (when to potentially split)

### Decision Logic (Pure Metrics-Based, Executed Continuously)

**This logic runs DURING generation, not after:**

```
BEFORE starting generation:
  - Create todo list breaking generation into phases (use TodoWrite)
  - Mark first todo as in_progress

WHILE generating task breakdowns:
  - Write next section (header, task, support section) to current file
  - Increment line_count by lines just written
  - If section is a task: increment tasks_completed

  IMMEDIATELY AFTER each section:
    IF line_count > 1900 AND tasks_completed < total_tasks:
      - Check if current position is at worktree group boundary
      - IF yes (natural boundary exists):
        - WRITE current plan file to disk (plan-0N-<phase>.md)
        - UPDATE todo: Mark current generation phase as complete
        - CREATE new plan file (plan-0(N+1)-<next-phase>.md)
        - Reset line_count = 0 for new file
        - UPDATE todo: Mark next generation phase as in_progress
        - CONTINUE with remaining tasks in new file
      - IF no (mid-group):
        - CONTINUE in current file until group boundary reached
        - THEN split at next group boundary

    IF tasks_completed == total_tasks AND all_sections_written:
      - WRITE final plan file to disk
      - CREATE README.md index if multiple files exist
      - UPDATE todo: Mark final phase as complete
      - BREAK while loop
```

**Key principles:**
- No subjective judgment - only objective line counts and task boundaries
- Active monitoring - check after EACH section written
- Incremental writing - files written to disk as you go, not accumulated
- Todo tracking - update your todo list as you complete each generation phase

### Cross-Reference Index (README.md)

When plan splitting occurs, create a `docs/plans/<feature-name>/README.md` index file for Conductor orchestration:

```markdown
# Implementation Plan: [Feature Name]

**Created**: [Date]
**Total Tasks**: [N]
**Total Plan Files**: [M]

This implementation plan is split across multiple files for Conductor orchestration.

## Plan Files (In Execution Order)

Conductor auto-discovers and orchestrates these files:

1. **[plan-01-database.md](./plan-01-database.md)** (Tasks 1-8)
   - Phase: Database schema and migrations
   - Worktree group: chain-1
   - Execution: Sequential
   - Line count: ~2000
   - Dependencies: None (can start immediately)

2. **[plan-02-api.md](./plan-02-api.md)** (Tasks 9-14)
   - Phase: API implementation
   - Worktree group: chain-2
   - Execution: Sequential
   - Line count: ~1800
   - Dependencies: Requires plan-01 Task 8 completion

3. **[plan-03-testing.md](./plan-03-testing.md)** (Tasks 15-20)
   - Phase: Testing and deployment
   - Worktree group: independent/chain-3
   - Execution: Mixed (some parallel, some sequential)
   - Line count: ~1200
   - Dependencies: Requires plan-02 Task 14 completion

## Getting Started with Conductor

Conductor automatically orchestrates execution across all files:

```bash
# Validate the multi-file plan
conductor validate docs/plans/feature-name/

# Run with orchestration
conductor run docs/plans/feature-name/
```

This executes tasks in dependency order across all plan files, with cross-file dependencies automatically managed.

## Manual Execution (without Conductor)

If executing without Conductor:
1. Start with [plan-01-database.md](./plan-01-database.md) - Task 1
2. Complete all tasks in sequence as documented
3. Move to [plan-02-api.md](./plan-02-api.md) only after plan-01 dependencies met
4. Then proceed to [plan-03-testing.md](./plan-03-testing.md)

## Overview

[Brief 2-3 sentence summary of the entire feature and what the plan accomplishes]

## File Format

All files follow Markdown format compatible with:
- Conductor task orchestration
- Cross-file task references (Task 9 can depend on Task 6, even across files)
- Automatic task renumbering if conflicts exist
```

### Output Confirmation

When plan generation completes, report to the user:

**Single file output:**
```
Plan created: docs/plans/<feature-slug>.md
- Total tasks: 12
- Total lines: 1,543
- Worktree groups: 3 (chain-1, independent-1, independent-2)
- Format: Single Markdown file
- Conductor support: Direct (conductor run docs/plans/<feature-slug>.md)
```

**Split file output (Phase 2A format):**
```
Plan created in: docs/plans/<feature-name>/
- Total tasks: 25
- Plan files created: 3 (Phase 2A format)
  - plan-01-database.md (2,023 lines, tasks 1-10)
  - plan-02-api.md (1,987 lines, tasks 11-18)
  - plan-03-testing.md (1,156 lines, tasks 19-25)
- Index: docs/plans/<feature-name>/README.md
- Worktree groups: 5 (chain-1, chain-2, independent-1, independent-2, independent-3)
- Auto-discovery: Enabled (Conductor finds plan-*.md files)

Run with Conductor: conductor run docs/plans/<feature-name>/
Start with: docs/plans/<feature-name>/plan-01-database.md - Task 1
```

### Why This Approach Works

1. **Objective triggers**: No guessing - split happens at 2000 lines, period
2. **Complete tasks**: Never truncate task details across files
3. **Natural boundaries**: Worktree groups provide logical split points
4. **Maintainable**: Each file remains readable and navigable
5. **Discoverable**: README index makes multi-file plans easy to understand
6. **Consistent**: Metric-based approach produces repeatable results

### 1. Implementation Plan Header with Conductor Configuration

For Conductor orchestration, the plan MUST include YAML frontmatter before the Markdown content. This frontmatter specifies executor configuration and multi-agent QC settings.

```markdown
---
conductor:
  default_agent: general-purpose
  max_concurrency: 3
  quality_control:
    enabled: true
    review_agent: quality-control
    retry_on_red: 2
    agents:
      mode: auto
      explicit_list: []
      additional: []
      blocked: []
---

# Implementation Plan: [Feature Name]

**Created**: [Date]
**Target**: [Brief description of what we're building]
**Estimated Tasks**: [Number]

## Context for the Engineer

You are implementing this feature in a codebase that:
- Uses [framework/language]
- Follows [architecture pattern]
- Tests with [test framework]
- [Other critical context]

**You are expected to**:
- Write tests BEFORE implementation (TDD)
- Commit frequently (after each completed task)
- Follow existing code patterns
- Keep changes minimal (YAGNI - You Aren't Gonna Need It)
- Avoid duplication (DRY - Don't Repeat Yourself)
```

**Conductor Frontmatter Explanation:**

- `conductor.default_agent`: Default agent for tasks without explicit agent assignment (recommended: `general-purpose`)
- `conductor.max_concurrency`: Maximum number of parallel task waves (recommended: 2-4)
- `conductor.quality_control.enabled`: Enable QC review of task outputs (default: false, explicitly set to true here)
- `conductor.quality_control.review_agent`: Agent to perform quality control reviews (recommended: `quality-control`)
- `conductor.quality_control.retry_on_red`: Maximum retries when QC returns RED verdict (recommended: 2)
- `conductor.quality_control.agents.mode`: QC agent selection mode:
  - `auto` (RECOMMENDED): Automatically selects QC agents based on file types (optimal for most projects)
  - `explicit`: Uses ONLY agents specified in `explicit_list` (required field when mode is explicit)
  - `mixed`: Combines auto-selected agents with those in `additional` list
- `conductor.quality_control.agents.explicit_list`: Required when mode is `explicit` - list of agents to use (empty by default)
- `conductor.quality_control.agents.additional`: Extra agents to add when mode is `mixed` (added to auto-selected agents)
- `conductor.quality_control.agents.blocked`: List of agent names to exclude from QC selection (filters auto-selected agents)

### 2. Worktree Groups Section

Add this section immediately after the header and before "Prerequisites Checklist":

```markdown
## Worktree Groups

This plan identifies task groupings for parallel execution using git worktrees. Each group operates in isolation with its own branch.

**Group chain-[N]**: Dependency Chain [N]
- **Tasks**: Task X, Task Y, Task Z (executed sequentially in dependency order)
- **Branch**: `feature/[feature-slug]/chain-[N]`
- **Execution**: Sequential (dependencies enforced)
- **Isolation**: Separate worktree from other groups
- **Rationale**: These tasks have dependencies that require sequential execution

**Group independent-[N]**: Independent Task [N]
- **Tasks**: Task N only
- **Branch**: `feature/[feature-slug]/independent-[N]`
- **Execution**: Parallel-safe (no dependencies)
- **Isolation**: Separate worktree from other groups
- **Rationale**: This task has no dependencies and can run in parallel with other work

[Repeat for each worktree group identified in Phase 1C]

### Worktree Management

**Creating worktrees**:
```bash
# For each group, create a worktree
git worktree add ../[repo-name]-[group-id] -b feature/[feature-slug]/[group-id]
```

**Switching between worktrees**:
```bash
cd ../[repo-name]-[group-id]
```

**Cleaning up after merge**:
```bash
git worktree remove ../[repo-name]-[group-id]
git branch -d feature/[feature-slug]/[group-id]
```
```

### 3. Prerequisites Checklist

Before starting implementation tasks, complete this mandatory validation:

#### Task 0: Environment Validation (BLOCKING)

**This task MUST complete successfully before any implementation tasks can begin.**

##### Python Projects

**Type Checking Setup:**
```bash
# Verify mypy configuration exists
ls pyproject.toml mypy.ini

# Run type checker
python -m mypy src/
# Expected: "Success: no issues found in N source files"
```

**Configuration requirements** (pyproject.toml):
```toml
[tool.mypy]
python_version = "3.11"  # Match your runtime version
warn_return_any = true
warn_unused_configs = true
disallow_untyped_defs = true
mypy_path = "src"  # Ensures mypy finds your modules

[[tool.mypy.overrides]]
module = "tests.*"
disallow_untyped_defs = false
```

**Testing Framework Setup:**
```bash
# Verify pytest configuration
ls pyproject.toml pytest.ini

# Verify tests can be collected
python -m pytest --collect-only
# Expected: "collected N items"
```

**Configuration requirements** (pyproject.toml):
```toml
[tool.pytest.ini_options]
pythonpath = ["src"]  # CRITICAL: Enables correct imports
testpaths = ["tests"]
python_files = ["test_*.py"]
python_classes = ["Test*"]
python_functions = ["test_*"]
```

**Import Path Validation:**
```bash
# Test that imports work from src/
python -c "from bot.handlers import MessageHandler"

# Test that imports work from tests/
cd tests && python -c "from bot.handlers import MessageHandler"

# CRITICAL: Verify no 'from src.' imports exist
grep -r 'from src\.' . --include='*.py'
# Expected: No output (no matches found)
```

**Formatting and Linting:**
```bash
# Install and configure
pip install black ruff

# Verify formatter works
python -m black --check .

# Verify linter works
python -m ruff check .
```

##### TypeScript Projects

**Type Checking Setup:**
```bash
# Verify TypeScript configuration
ls tsconfig.json

# Run type checker
npx tsc --noEmit
# Expected: No errors
```

**Configuration requirements** (tsconfig.json):
```json
{
  "compilerOptions": {
    "target": "ES2020",
    "module": "commonjs",
    "strict": true,
    "noImplicitAny": true,
    "strictNullChecks": true,
    "esModuleInterop": true,
    "skipLibCheck": false,
    "outDir": "./dist",
    "rootDir": "./src"
  },
  "include": ["src/**/*"],
  "exclude": ["node_modules", "dist"]
}
```

**Testing Framework Setup:**
```bash
# Verify tests can be listed
npm test -- --listTests
# Expected: List of test files
```

**Linting and Formatting:**
```bash
# Verify ESLint configuration
ls .eslintrc.js eslint.config.js

# Run linter
npx eslint . --ext .ts,.tsx

# Verify Prettier
npx prettier --check .
```

##### Go Projects

**Module Setup:**
```bash
# Verify go.mod exists
ls go.mod

# Verify module path
go list -m
# Expected: github.com/user/project
```

**Type Checking and Vetting:**
```bash
# Run go vet
go vet ./...
# Expected: No output (success)
```

**Formatting:**
```bash
# Check formatting
gofmt -l .
# Expected: No output (all files formatted)

goimports -l .
# Expected: No output (all imports organized)
```

**Testing:**
```bash
# Compile tests without running
go test -run=^$ ./...
# Expected: No errors

# Run all tests
go test ./...
```

##### Blocking vs Non-Blocking Failures

**BLOCKING (must pass before starting tasks):**
- Configuration files missing (pyproject.toml, tsconfig.json, go.mod)
- Type checker not configured (mypy_path, tsconfig strict mode)
- Import paths broken (PYTHONPATH misconfigured)
- Tests cannot be collected (pytest.ini testpaths wrong)

**NON-BLOCKING (should fix but don't block task start):**
- Linter warnings (can be fixed incrementally)
- Code formatting issues (auto-fixable)
- Documentation missing (can be added later)

#### Additional Prerequisites

- Required tools installed (language runtime, package manager)
- Development environment setup (virtual env, node_modules, etc.)
- Access to necessary services/APIs
- Branch created from correct base

### 3A. Common Pitfalls Reference

Review this section BEFORE starting any tasks to avoid repeating common errors from previous implementations.

#### Python Pitfalls

**1. Using 'from src.' imports instead of proper module imports**

```python
# WRONG
from src.bot.handlers import MessageHandler

# RIGHT (when project root is in PYTHONPATH)
from bot.handlers import MessageHandler
```

**Why**: ModuleNotFoundError when running tests or scripts from different directories
**Detection**: `grep -r 'from src\.' .`
**Fix**: Update import statements and ensure PYTHONPATH or pytest.ini pythonpath is configured

**2. Incomplete async context managers (missing __aexit__)**

```python
# WRONG - only __aenter__
class DatabaseConnection:
    async def __aenter__(self):
        self.conn = await connect()
        return self
    # Missing __aexit__!

# RIGHT - both methods
class DatabaseConnection:
    async def __aenter__(self):
        self.conn = await connect()
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb):
        if self.conn:
            await self.conn.close()
```

**Why**: Runtime error when using 'async with' statement
**Detection**: `grep -A 5 '__aenter__' | grep -L '__aexit__'`
**Test pattern**:
```python
async def test_context_manager():
    async with DatabaseConnection() as conn:
        # Should enter and exit without errors
        assert conn is not None
```

**3. Missing bot_user_id initialization in Discord bots**

```python
# WRONG - bot_user_id never set
class DiscordBot:
    def __init__(self, token: str):
        self.token = token
        self.bot_user_id = None  # Never initialized!

# RIGHT - initialize in on_ready
class DiscordBot:
    def __init__(self, token: str):
        self.token = token
        self.bot_user_id: int | None = None

    async def on_ready(self):
        self.bot_user_id = self.user.id
        logger.info(f"Bot ready: {self.bot_user_id}")
```

**Why**: NoneType errors when bot tries to use its own ID
**Detection**: `grep -n 'bot_user_id' | grep '= None'` (Check if it's ever reassigned)
**Fix**: Initialize in on_ready or setup method, add assertion in critical paths

**4. No error handling in event handlers**

```python
# WRONG - crashes on any error
async def on_message(self, message):
    await self.process_message(message)

# RIGHT - graceful error handling
async def on_message(self, message):
    try:
        await self.process_message(message)
    except Exception as e:
        logger.error(f"Error processing message: {e}", exc_info=True)
        # Optionally notify user
        await message.channel.send("An error occurred processing your message.")
```

**Why**: Unhandled exceptions crash the entire bot
**Detection**: `grep -A 10 'async def on_' | grep -L 'try:'`
**Test pattern**:
```python
async def test_error_handling():
    with pytest.raises(RuntimeError):
        await handler.on_message(malformed_message)
    # Handler should log but not crash
```

**5. Using Dict/List instead of dict/list in type hints (Python 3.9+)**

```python
# WRONG (old style)
from typing import Dict, List
def process(data: Dict[str, List[int]]) -> None: ...

# RIGHT (Python 3.9+ builtin generics)
def process(data: dict[str, list[int]]) -> None: ...
```

**Why**: Deprecated in Python 3.9+, will be removed in future versions
**Detection**: `grep -r 'from typing import Dict\|List\|Tuple' .`
**Fix**: Use lowercase dict, list, tuple from builtins

**6. mypy not configured or not running**

**Why**: Type hints are useless without type checking
**Detection**:
```bash
# Check if mypy configuration exists
ls pyproject.toml setup.cfg mypy.ini

# Check if mypy is in dev dependencies
cat pyproject.toml | grep mypy
```

**Fix**:
```toml
# Add to pyproject.toml
[tool.mypy]
python_version = "3.11"
warn_return_any = true
warn_unused_configs = true
disallow_untyped_defs = true
mypy_path = "src"

# Run mypy
python -m mypy src/
```

#### TypeScript Pitfalls

**1. Missing null checks with optional properties**

```typescript
// WRONG
function getUsername(user: User): string {
    return user.name.toLowerCase(); // Crashes if name is undefined
}

// RIGHT
function getUsername(user: User): string {
    return user.name?.toLowerCase() ?? 'anonymous';
}
```

**Why**: Runtime errors when optional properties are undefined
**Detection**: eslint rule: `@typescript-eslint/no-non-null-assertion`

**2. Using 'any' type instead of proper types**

```typescript
// WRONG
function process(data: any): any { ... }

// RIGHT
function process(data: InputData): OutputData { ... }
```

**Why**: Defeats the purpose of TypeScript
**Detection**: eslint rule: `@typescript-eslint/no-explicit-any`

#### Go Pitfalls

**1. Not checking errors**

```go
// WRONG
file, _ := os.Open("config.json")

// RIGHT
file, err := os.Open("config.json")
if err != nil {
    return fmt.Errorf("failed to open config: %w", err)
}
```

**Why**: Silent failures and unexpected behavior
**Detection**: golangci-lint with errcheck linter enabled

**2. Goroutine leaks (not closing channels or contexts)**

```go
// WRONG
go func() {
    for {
        doWork() // Never exits!
    }
}()

// RIGHT
ctx, cancel := context.WithCancel(context.Background())
defer cancel()
go func() {
    for {
        select {
        case <-ctx.Done():
            return
        default:
            doWork()
        }
    }
}()
```

**Why**: Memory leaks and resource exhaustion
**Detection**: `go test -race ./...`

#### Cross-Language Pitfalls

**1. Not running formatters before commit**

- **Python**: `black . && ruff check .`
- **TypeScript**: `prettier --write . && eslint --fix .`
- **Go**: `gofmt -w . && goimports -w .`

**Why**: CI failures, inconsistent code style

**2. Tests not running in CI/CD**

**Detection**:
```bash
# Check if CI configuration exists
ls .github/workflows/ .gitlab-ci.yml

# Verify test commands are present
cat .github/workflows/*.yml | grep -i test
```

**Why**: Broken code gets merged

**3. Hardcoded paths or credentials**

**Detection**:
```bash
grep -r '/Users/' .
grep -r '/home/' .
grep -r 'password.*=' .
grep -r 'api_key.*=' .
```

**Why**: Works locally, fails in production or on other machines

### 4. Detailed Task Breakdown

For EACH task, provide:

```markdown
## Task [N]: [Task Name]

**Agent**: `agent-name` (required - reference to agent from ~/.claude/agents/)
**File(s)**: `path/to/file.ext`
**Depends on**: [Task specification - if applicable]
  - Single dependency: `Task 1`
  - Multiple dependencies: `Task 1, Task 2, Task 3` (comma-separated, list each individually)
  - **IMPORTANT**: Range notation (e.g., `Tasks 1-3`) is NOT supported - always list task numbers individually
  - Omit this field entirely if no dependencies
**WorktreeGroup**: [group-id] (e.g., "chain-1", "independent-3")
**Estimated time**: [5m/15m/30m/1h]

**Agent Assignment Guidelines:**

Based on the task type and technology stack, assign appropriate agents:
- Go code: `golang-pro`
- Python code: `python-pro`
- JavaScript/TypeScript: `javascript-pro`, `typescript-pro`
- Testing tasks: `test-automator`
- Documentation: `technical-documentation-specialist`
- Database work: `database-optimizer` or `database-admin`
- Performance: `performance-engineer`
- General tasks: `general-purpose`

Always verify the agent exists in the discovered agent list. If uncertain, use `general-purpose`.

### What you're building
[2-3 sentences explaining WHAT and WHY]

### Test First (TDD)

**Test file**: `path/to/test_file.ext`

**Test structure**:
```[language]
describe/test block for [feature]
  - test case 1: [specific behavior]
  - test case 2: [specific edge case]
  - test case 3: [error condition]
```

**Test specifics**:
- Mock these dependencies: [list]
- Use these fixtures/factories: [list]
- Assert these outcomes: [list]
- Edge cases to cover: [list]

**Example test skeleton**:
```[language]
[Provide actual test code structure following project patterns]
```

### Implementation

**Approach**:
[Detailed explanation of how to implement]

**Code structure**:
```[language]
[Provide skeleton/pseudocode showing structure]
```

**Key points**:
- Follow pattern from: `path/to/similar/file.ext:line`
- Use utility: `utilityName` from `path/to/utility.ext`
- Error handling: [specific approach]
- Validation: [what to validate and how]

**Integration points**:
- Imports needed: [list]
- Services to inject: [list]
- Config values: [list]

### Critical Patterns

**Complete, copy-paste-ready code templates for common patterns. Use these to prevent errors documented in Common Pitfalls Reference.**

#### Async Context Manager (Python)

Full implementation with BOTH `__aenter__` AND `__aexit__`:

```python
# src/bot/database.py
from typing import Optional
import logging

logger = logging.getLogger(__name__)

class DatabaseConnection:
    """Async context manager for database connections.

    CRITICAL: Must implement BOTH __aenter__ and __aexit__
    """
    def __init__(self, connection_string: str):
        self.connection_string = connection_string
        self.conn: Optional[Connection] = None
        self.is_connected = False

    async def __aenter__(self) -> "DatabaseConnection":
        """Enter context: establish connection."""
        logger.info("Establishing database connection")
        self.conn = await asyncpg.connect(self.connection_string)
        self.is_connected = True
        return self

    async def __aexit__(
        self,
        exc_type: Optional[type],
        exc_val: Optional[Exception],
        exc_tb: Optional[object]
    ) -> None:
        """Exit context: close connection.

        CRITICAL: This method is REQUIRED for context manager protocol.
        Without it, 'async with' will fail at runtime.
        """
        if self.conn:
            logger.info("Closing database connection")
            await self.conn.close()
            self.is_connected = False

        # Return None to propagate exceptions (standard behavior)
        # Return True to suppress exceptions (rarely needed)

# Usage:
async def query_data():
    async with DatabaseConnection(DB_URL) as db:
        result = await db.conn.fetch("SELECT * FROM users")
        return result
    # Connection automatically closed here
```

#### Initialization Order with Runtime Enforcement (Python)

```python
# src/bot/client.py
from typing import Optional
import logging

logger = logging.getLogger(__name__)

class DiscordBot:
    """Discord bot with enforced initialization order."""

    def __init__(self, token: str):
        self.token = token
        # Initialize as None with type hint
        self.bot_user_id: Optional[int] = None
        self._ready = False

    async def on_ready(self):
        """Called when bot connects. MUST run before processing messages."""
        # CRITICAL: Set bot_user_id here
        if self.user:
            self.bot_user_id = self.user.id
            self._ready = True
            logger.info(f"Bot ready with ID: {self.bot_user_id}")
        else:
            raise RuntimeError("Bot user not available in on_ready")

    def _ensure_initialized(self):
        """Enforce initialization before use."""
        if not self._ready or self.bot_user_id is None:
            raise RuntimeError(
                "Bot not initialized. Ensure on_ready() has been called "
                "before processing messages."
            )

    async def filter_own_messages(self, message):
        """Filter out bot's own messages.

        CRITICAL: Must check bot_user_id is initialized
        """
        # Enforce initialization
        self._ensure_initialized()

        # Now safe to use bot_user_id
        if message.author.id == self.bot_user_id:
            logger.debug("Ignoring own message")
            return True
        return False
```

#### Error Handling in Event Handlers (Python)

```python
# src/bot/handlers.py
import logging
from typing import Optional

logger = logging.getLogger(__name__)

class MessageHandler:
    """Message handler with comprehensive error handling."""

    async def on_message(self, message):
        """Handle incoming messages.

        CRITICAL: Must catch ALL exceptions to prevent bot crashes.
        Event handlers should NEVER allow exceptions to propagate.
        """
        try:
            # Pre-validation
            if not message or not message.content:
                logger.warning("Received invalid message")
                return

            # Main processing
            result = await self._process_message(message)

            if result.error:
                logger.error(f"Processing error: {result.error}")
                await self._notify_user(message, "An error occurred")

        except ValueError as e:
            # Specific error types
            logger.error(f"Validation error: {e}", exc_info=True)
            await self._notify_user(message, "Invalid input")

        except Exception as e:
            # Catch-all for unexpected errors
            logger.error(
                f"Unexpected error in on_message: {e}",
                exc_info=True,
                extra={"message_id": message.id}
            )
            # Optionally notify user
            try:
                await self._notify_user(message, "An unexpected error occurred")
            except Exception:
                # Even notification can fail - don't crash
                logger.error("Failed to notify user of error")

    async def _notify_user(self, message, text: str):
        """Safely notify user (with its own error handling)."""
        try:
            await message.channel.send(text)
        except Exception as e:
            logger.error(f"Failed to send notification: {e}")
```

#### Correct Import Patterns (Python)

```python
# CORRECT imports (assumes src/ in PYTHONPATH via pytest.ini or PYTHONPATH env var)

# In src/bot/handlers.py:
from bot.models import User, Message
from bot.database import DatabaseConnection
from utils.logging import setup_logger

# In src/api/routes.py:
from bot.handlers import MessageHandler
from api.middleware import authenticate

# In tests/test_handlers.py:
import pytest
from bot.handlers import MessageHandler  # Same imports as src files
from bot.models import Message

# WRONG - DO NOT USE:
# from src.bot.handlers import MessageHandler  # Don't include 'src' prefix
# import sys; sys.path.append('../src')  # Don't manipulate sys.path

# Configuration to make this work:
# 1. In pyproject.toml or pytest.ini:
#    [tool.pytest.ini_options]
#    pythonpath = ["src"]
#
# 2. Or set PYTHONPATH environment variable:
#    export PYTHONPATH="${PYTHONPATH}:${PWD}/src"
```

#### Modern Type Hints (Python 3.9+)

```python
# CORRECT - Python 3.9+ builtin generics
from typing import Optional, Protocol
from collections.abc import Callable, Awaitable

def process_data(
    data: dict[str, list[int]],
    callback: Optional[Callable[[int], str]] = None
) -> tuple[bool, str]:
    """Process data with modern type hints."""
    pass

async def fetch_users(
    user_ids: list[int]
) -> dict[int, dict[str, str | int]]:
    """Fetch users asynchronously."""
    pass

# WRONG - Old style (deprecated in Python 3.9+):
# from typing import Dict, List, Tuple
# def process_data(data: Dict[str, List[int]]) -> Tuple[bool, str]:
#     pass
```

#### TypeScript Null Safety

```typescript
// src/handlers/MessageHandler.ts

interface User {
  name?: string;
  email?: string;
  profile?: {
    avatar?: string;
    bio?: string;
  };
}

export class UserService {
  /**
   * Get username with proper null safety
   * CRITICAL: Always use optional chaining for optional properties
   */
  getUsername(user: User | null | undefined): string {
    // Optional chaining and nullish coalescing
    return user?.name?.toLowerCase() ?? 'anonymous';
  }

  getUserAvatar(user: User | null | undefined): string {
    // Deep optional chaining
    return user?.profile?.avatar ?? '/default-avatar.png';
  }

  /**
   * Process user data with comprehensive null checks
   */
  async processUser(userId: number): Promise<void> {
    const user = await this.fetchUser(userId);

    // Early return pattern for null checks
    if (!user) {
      console.warn(`User ${userId} not found`);
      return;
    }

    // Now user is guaranteed non-null
    const username = user.name ?? 'Unknown';
    console.log(`Processing user: ${username}`);
  }
}

// WRONG - crashes if name is undefined:
// return user.name.toLowerCase();
```

#### Go Error Handling

```go
// handlers/message.go
package handlers

import (
  "context"
  "fmt"
  "log"
)

type MessageHandler struct {
  db Database
}

// CORRECT: Always check errors, wrap with context
func (h *MessageHandler) ProcessMessage(ctx context.Context, msg *Message) error {
  // Check for nil
  if msg == nil {
    return fmt.Errorf("message cannot be nil")
  }

  // Call database with error checking
  user, err := h.db.GetUser(ctx, msg.AuthorID)
  if err != nil {
    // Wrap error with context using %w verb
    return fmt.Errorf("failed to get user %d: %w", msg.AuthorID, err)
  }

  // More operations with error checking
  if err := h.validateMessage(msg); err != nil {
    return fmt.Errorf("message validation failed: %w", err)
  }

  // Process message
  result, err := h.process(ctx, msg, user)
  if err != nil {
    return fmt.Errorf("failed to process message: %w", err)
  }

  log.Printf("Message processed successfully: %v", result)
  return nil
}

// WRONG - ignoring errors:
// user, _ := h.db.GetUser(ctx, msg.AuthorID)
```

#### Go Context Cancellation (Prevent Goroutine Leaks)

```go
// worker/processor.go
package worker

import (
  "context"
  "log"
  "time"
)

type Processor struct {
  workQueue chan Work
}

// CORRECT: Goroutine respects context cancellation
func (p *Processor) Start(ctx context.Context) {
  go func() {
    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()

    for {
      select {
      case <-ctx.Done():
        // Context cancelled - clean shutdown
        log.Println("Processor stopping due to context cancellation")
        return

      case work := <-p.workQueue:
        // Process work with context
        if err := p.processWork(ctx, work); err != nil {
          log.Printf("Error processing work: %v", err)
        }

      case <-ticker.C:
        // Periodic work
        p.healthCheck()
      }
    }
  }()
}

// WRONG - goroutine never exits (memory leak):
// go func() {
//   for {
//     work := <-p.workQueue
//     p.processWork(work)
//   }
// }()
```

### Verification

**Manual testing**:
1. [Step-by-step manual test]
2. [How to verify it works]

**Automated tests**:
```bash
[Exact command to run tests]
```

**Expected output**:
[What success looks like]

### Commit

**Commit message**:
```
[type]: [clear description]

[Optional body explaining why]
```

**Files to commit**:
- `path/to/implementation.ext`
- `path/to/test.ext`

### Code Quality Checklist

**Mandatory checks that MUST pass before considering a task complete. Run these commands before committing code.**

#### Python Quality Checks

**1. Formatter (BLOCKING)**
```bash
# Auto-format Python code to PEP 8 style
python -m black .
```
- **When to run**: Before every commit
- **Failure action**: Auto-fix by running the command, then review changes

**2. Type Checker (BLOCKING)**
```bash
# Static type checking for Python code
python -m mypy src/
```
- **When to run**: Before every commit, after adding/modifying type hints
- **Failure action**: Fix type errors - these indicate real bugs
- **Common issues**:
  - `error: Cannot find implementation or library stub for module` → Add `# type: ignore` comment or install type stubs (`pip install types-*`)
  - `error: Incompatible return value type` → Update return type hint or fix the return statement

**3. Test Runner (BLOCKING)**
```bash
# Run all tests to ensure functionality
python -m pytest
```
- **When to run**: Before every commit
- **Failure action**: Fix failing tests - do not commit broken code

**4. Linter (NON-BLOCKING)**
```bash
# Fast Python linter for code quality issues
python -m ruff check .
# Auto-fix: python -m ruff check --fix .
```
- **When to run**: Before every commit
- **Failure action**: Fix issues manually or run auto-fix

**5. Coverage Check (NON-BLOCKING)**
```bash
# Measure test coverage percentage
python -m pytest --cov=src --cov-report=term-missing
```
- **When to run**: After implementing new features
- **Failure action**: Add tests for uncovered code (aim for 80%+)

**Full Quality Pipeline (Python)**:
```bash
# Run all quality checks in sequence
python -m black . && \
python -m mypy src/ && \
python -m pytest --cov=src --cov-report=term-missing && \
python -m ruff check .
```

#### TypeScript Quality Checks

**1. Formatter (BLOCKING)**
```bash
# Auto-format TypeScript/JavaScript code
npx prettier --write .
```
- **When to run**: Before every commit
- **Failure action**: Auto-fix by running the command, then review changes

**2. Type Checker (BLOCKING)**
```bash
# TypeScript compiler type checking
npx tsc --noEmit
```
- **When to run**: Before every commit
- **Failure action**: Fix type errors - these indicate real bugs
- **Common issues**:
  - `Property 'x' does not exist on type 'Y'` → Add property to type definition or use type assertion
  - `Object is possibly 'undefined'` → Add null check or use optional chaining (`?.`)

**3. Test Runner (BLOCKING)**
```bash
# Run all tests
npm test
```
- **When to run**: Before every commit
- **Failure action**: Fix failing tests - do not commit broken code

**4. Linter (NON-BLOCKING)**
```bash
# Linter for TypeScript code quality
npx eslint . --ext .ts,.tsx
# Auto-fix: npx eslint --fix . --ext .ts,.tsx
```
- **When to run**: Before every commit
- **Failure action**: Fix issues manually or run auto-fix

**Full Quality Pipeline (TypeScript)**:
```bash
# Run all quality checks in sequence
npx prettier --write . && \
npx tsc --noEmit && \
npm test && \
npx eslint . --ext .ts,.tsx
```

#### Go Quality Checks

**1. Formatter (BLOCKING)**
```bash
# Auto-format Go code
gofmt -w .
```
- **When to run**: Before every commit
- **Failure action**: Auto-fix by running the command

**2. Import Organizer (BLOCKING)**
```bash
# Organize imports and format code
goimports -w .
```
- **When to run**: Before every commit
- **Failure action**: Auto-fix by running the command

**3. Go Vet (BLOCKING)**
```bash
# Examines Go source code and reports suspicious constructs
go vet ./...
```
- **When to run**: Before every commit
- **Failure action**: Fix issues - these indicate potential bugs

**4. Test Runner (BLOCKING)**
```bash
# Run all tests
go test ./...
```
- **When to run**: Before every commit
- **Failure action**: Fix failing tests - do not commit broken code

**5. Race Detector (BLOCKING for concurrent code)**
```bash
# Detect race conditions
go test -race ./...
```
- **When to run**: Before committing concurrent code
- **Failure action**: Fix race conditions using mutexes or channels

**6. Linter (NON-BLOCKING)**
```bash
# Comprehensive Go linter
golangci-lint run
```
- **When to run**: Before every commit
- **Failure action**: Fix issues or adjust `.golangci.yml` config

**Full Quality Pipeline (Go)**:
```bash
# Run all quality checks in sequence
gofmt -w . && \
goimports -w . && \
go vet ./... && \
go test -race ./... && \
golangci-lint run
```

#### Pre-Commit Checklist

**Run these in order**:

1. **Format code**
   - Python: `python -m black .`
   - TypeScript: `npx prettier --write .`
   - Go: `gofmt -w . && goimports -w .`

2. **Run type checker**
   - Python: `python -m mypy src/`
   - TypeScript: `npx tsc --noEmit`
   - Go: `go vet ./...`

3. **Run tests**
   - Python: `python -m pytest`
   - TypeScript: `npm test`
   - Go: `go test ./...`

4. **Run linter (optional but recommended)**
   - Python: `python -m ruff check .`
   - TypeScript: `npx eslint .`
   - Go: `golangci-lint run`

**Blocking vs Non-Blocking**:

**BLOCKING (must pass before commit)**:
- Python: black, mypy, pytest
- TypeScript: prettier, tsc, npm test
- Go: gofmt, goimports, go vet, go test, go test -race

**NON-BLOCKING (should pass but can be fixed incrementally)**:
- Python: ruff (linter), pytest --cov (coverage)
- TypeScript: eslint (linter)
- Go: golangci-lint (linter)
```

### 5. Testing Strategy Section

Include a dedicated section:

```markdown
## Testing Strategy

### Unit Tests
- **Location**: [directory]
- **Naming**: [convention]
- **Run command**: `[command]`
- **Coverage target**: [X%]

### Integration Tests
- **Location**: [directory]
- **What to test**: [integration points]
- **Setup required**: [fixtures/test data]

### E2E Tests (if applicable)
- **Location**: [directory]
- **Critical flows**: [list]

### Test Design Principles for This Feature

**Use these patterns**:
1. [Pattern 1 with example]
2. [Pattern 2 with example]

**Avoid these anti-patterns**:
1. [Anti-pattern 1 and why]
2. [Anti-pattern 2 and why]

**Mocking guidelines**:
- Mock external services: [list]
- Don't mock: [list]
- Use project's mocking pattern: [reference to existing test]
```

### 6. Commit Strategy

```markdown
## Commit Strategy

Break this work into [N] commits following this sequence:

1. **[Commit type]**: [Description]
   - Files: [list]
   - Why separate: [reason]

2. **[Commit type]**: [Description]
   - Files: [list]
   - Why separate: [reason]

[Continue...]

**Commit message format**:
Follow the pattern seen in recent commits:
[Show example from git log]
```

### 7. Common Pitfalls

```markdown
## Common Pitfalls & How to Avoid Them

1. **[Pitfall 1]**
   - Why it happens: [explanation]
   - How to avoid: [solution]
   - Reference: [similar code that does it right]

2. **[Pitfall 2]**
   - Why it happens: [explanation]
   - How to avoid: [solution]
   - Reference: [similar code that does it right]
```

### 8. Resources Section

```markdown
## Resources & References

### Existing Code to Reference
- Similar feature: `path/to/similar.ext`
- Utility functions: `path/to/utils.ext`
- Test examples: `path/to/test_example.ext`

### Documentation
- Framework docs: [link if applicable]
- Internal docs: [path to relevant docs]
- API specs: [path if applicable]

### Validation
- [ ] All tests pass
- [ ] Linter passes
- [ ] Formatted correctly
- [ ] No console.log/print statements left
- [ ] Error handling in place
- [ ] Edge cases covered
```

### 9. Enforcement Mechanisms

**Automated enforcement of code quality standards to prevent common errors from reaching production. These mechanisms should be set up at the start of the project and enforced consistently.**

#### Pre-Commit Hooks

Git pre-commit hooks run automatically before every commit, preventing commits that violate quality standards. These are local checks that provide immediate feedback to developers.

##### Python Pre-Commit Setup

**Tool**: pre-commit framework

**Installation**:
```bash
# Install pre-commit
pip install pre-commit

# Add pre-commit to requirements-dev.txt
echo "pre-commit>=3.5.0" >> requirements-dev.txt
```

**Configuration file**: `.pre-commit-config.yaml`

**Configuration example**:
```yaml
# .pre-commit-config.yaml
repos:
  # Code formatting with black
  - repo: https://github.com/psf/black
    rev: 23.10.0
    hooks:
      - id: black
        language_version: python3.11

  # Import sorting
  - repo: https://github.com/pycqa/isort
    rev: 5.12.0
    hooks:
      - id: isort
        args: ["--profile", "black"]

  # Linting with ruff
  - repo: https://github.com/astral-sh/ruff-pre-commit
    rev: v0.1.4
    hooks:
      - id: ruff
        args: ["--fix"]

  # Type checking with mypy
  - repo: https://github.com/pre-commit/mirrors-mypy
    rev: v1.6.0
    hooks:
      - id: mypy
        additional_dependencies: [types-all]
        args: [--config-file=pyproject.toml]

  # Check for 'from src.' imports (CRITICAL)
  - repo: local
    hooks:
      - id: check-src-imports
        name: Check for incorrect 'from src.' imports
        entry: bash -c 'if grep -r "from src\." . --include="*.py"; then echo "ERROR: Found \"from src.\" imports. Use proper module imports."; exit 1; fi'
        language: system
        pass_filenames: false

  # Check for missing __aexit__ in async context managers
  - repo: local
    hooks:
      - id: check-async-context-managers
        name: Check async context managers have both __aenter__ and __aexit__
        entry: bash -c 'files=$(grep -l "async def __aenter__" $(find . -name "*.py")); for f in $files; do if ! grep -q "__aexit__" "$f"; then echo "ERROR: $f has __aenter__ but missing __aexit__"; exit 1; fi; done'
        language: system
        pass_filenames: false
```

**Setup commands**:
```bash
# Install git hooks
pre-commit install

# Run against all files (one time)
pre-commit run --all-files

# Hooks now run automatically on every commit
```

**Bypass when needed** (RARE - only for emergency hotfixes):
```bash
git commit --no-verify -m "hotfix: critical security patch"
```

##### TypeScript Pre-Commit Setup

**Tool**: husky + lint-staged

**Installation**:
```bash
# Install husky and lint-staged
npm install --save-dev husky lint-staged

# Initialize husky
npx husky install

# Add postinstall script to package.json
npm pkg set scripts.prepare="husky install"
```

**Configuration file**: `package.json`

**Configuration example**:
```json
{
  "lint-staged": {
    "*.{ts,tsx}": [
      "prettier --write",
      "eslint --fix",
      "bash -c 'tsc --noEmit'"
    ],
    "*.{json,md,yml}": [
      "prettier --write"
    ]
  }
}
```

**Hook file**: `.husky/pre-commit`

**Hook content**:
```bash
#!/usr/bin/env sh
. "$(dirname -- "$0")/_/husky.sh"

# Run lint-staged
npx lint-staged

# Run tests if any test files changed
git diff --cached --name-only | grep -q '\.test\.ts$' && npm test
```

**Setup commands**:
```bash
# Create pre-commit hook
npx husky add .husky/pre-commit "npx lint-staged"

# Make executable
chmod +x .husky/pre-commit
```

##### Go Pre-Commit Setup

**Tool**: git pre-commit hook script

**Hook file**: `.git/hooks/pre-commit`

**Hook content**:
```bash
#!/bin/sh
# Go pre-commit hook

# Format code
echo "Running gofmt..."
gofmt -w .

echo "Running goimports..."
goimports -w .

# Re-add formatted files
git add -u

# Run go vet
echo "Running go vet..."
if ! go vet ./...; then
  echo "go vet failed. Fix issues before committing."
  exit 1
fi

# Run tests
echo "Running tests..."
if ! go test ./...; then
  echo "Tests failed. Fix tests before committing."
  exit 1
fi

# Run race detector
echo "Running race detector..."
if ! go test -race ./...; then
  echo "Race conditions detected. Fix before committing."
  exit 1
fi

echo "Pre-commit checks passed!"
```

**Setup commands**:
```bash
# Create hook file and paste content above
cat > .git/hooks/pre-commit << 'EOF'
[paste hook_content above]
EOF

# Make executable
chmod +x .git/hooks/pre-commit

# Test hook
./.git/hooks/pre-commit
```

#### CI/CD Gates

Continuous Integration checks that run on every push and pull request. These enforce quality standards at the repository level and prevent merging code that violates standards.

##### GitHub Actions - Python Workflow

**File**: `.github/workflows/python-quality.yml`

**Content**:
```yaml
name: Python Quality Checks
on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main, develop]

jobs:
  quality:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Set up Python
        uses: actions/setup-python@v4
        with:
          python-version: '3.11'

      - name: Install dependencies
        run: |
          pip install -r requirements.txt
          pip install -r requirements-dev.txt

      - name: Check formatting with black
        run: python -m black --check .

      - name: Check imports
        run: |
          if grep -r "from src\." . --include="*.py"; then
            echo "ERROR: Found 'from src.' imports"
            exit 1
          fi

      - name: Type check with mypy
        run: python -m mypy src/

      - name: Lint with ruff
        run: python -m ruff check .

      - name: Run tests with coverage
        run: python -m pytest --cov=src --cov-report=xml --cov-report=term

      - name: Upload coverage to Codecov
        uses: codecov/codecov-action@v3
        with:
          file: ./coverage.xml

      - name: Check coverage threshold
        run: |
          coverage report --fail-under=80
```

##### GitHub Actions - TypeScript Workflow

**File**: `.github/workflows/typescript-quality.yml`

**Content**:
```yaml
name: TypeScript Quality Checks
on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main, develop]

jobs:
  quality:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Setup Node.js
        uses: actions/setup-node@v3
        with:
          node-version: '18'
          cache: 'npm'

      - name: Install dependencies
        run: npm ci

      - name: Check formatting with prettier
        run: npx prettier --check .

      - name: Type check with tsc
        run: npx tsc --noEmit

      - name: Lint with eslint
        run: npx eslint . --ext .ts,.tsx

      - name: Run tests with coverage
        run: npm test -- --coverage

      - name: Upload coverage to Codecov
        uses: codecov/codecov-action@v3
```

##### GitHub Actions - Go Workflow

**File**: `.github/workflows/go-quality.yml`

**Content**:
```yaml
name: Go Quality Checks
on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main, develop]

jobs:
  quality:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Format check
        run: |
          if [ "$(gofmt -l . | wc -l)" -gt 0 ]; then
            echo "Code not formatted. Run 'gofmt -w .'"
            exit 1
          fi

      - name: Import check
        run: |
          if [ "$(goimports -l . | wc -l)" -gt 0 ]; then
            echo "Imports not organized. Run 'goimports -w .'"
            exit 1
          fi

      - name: Go vet
        run: go vet ./...

      - name: Run tests
        run: go test ./...

      - name: Race detector
        run: go test -race ./...

      - name: golangci-lint
        uses: golangci/golangci-lint-action@v3
        with:
          version: latest
```

## Phase 3: Plan Review

After generating the plan:

1. **Validate completeness**:
   - Every task has test-first approach
   - Every task has clear file paths
   - Every task has verification steps
   - Every task has worktree group assignment
   - Commit points are logical and frequent

2. **Ensure beginner-friendliness**:
   - No assumed knowledge about the codebase
   - Test design is explicitly explained
   - Patterns are referenced with file paths
   - Common mistakes are called out
   - Worktree grouping rationale is clear

3. **Verify DRY/YAGNI/TDD principles**:
   - Tests are written first for each task
   - No unnecessary abstractions suggested
   - Code reuse is explicitly called out
   - Frequent commits are mandated

4. **Validate worktree grouping**:
   - Dependency chains are correctly identified
   - Independent tasks are properly isolated
   - Group assignments enable maximum parallelism
   - Branch naming is consistent and clear

## Output Format

1. Write the complete plan to `docs/plans/<feature-slug>.md`
2. Confirm to the user:
   - Plan location
   - Number of tasks
   - Number of worktree groups (chains vs independent)
   - Parallelism opportunities identified
   - Estimated time
   - First task to start with

## Important Guidelines

- **Be specific**: Use actual file paths, not placeholders
- **Be concrete**: Provide code skeletons, not just descriptions
- **Be practical**: Reference existing code extensively
- **Be educational**: Explain the "why" behind decisions
- **Be thorough**: Assume zero codebase knowledge
- **Be test-focused**: TDD is mandatory, explain test design clearly
- **Be commit-focused**: Make commit strategy explicit
- **Be agent-aware**: Assign the most appropriate agent from the discovered list to each task based on the work type. Every task MUST have an agent assigned.
- **Be dependency-aware**: Analyze task dependencies and assign worktree groups correctly
- **Be parallelism-conscious**: Maximize concurrent execution while respecting dependencies

The goal is that an engineer can follow this plan mechanically and produce high-quality, well-tested code that fits the existing codebase perfectly, with the ability to work on independent tasks in parallel using separate worktrees.
