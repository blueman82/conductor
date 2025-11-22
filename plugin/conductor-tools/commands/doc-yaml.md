---
allowed-tools: Read, Bash(ls*), Glob, Grep, Bash(git status*), Bash(git log*), Bash(fd*), Write, TodoWrite, AskUserTool
argument-hint: "feature description"
description: Generate comprehensive implementation plan in YAML format with detailed tasks, testing strategy, and commit points
---

# Doc YAML - Comprehensive Implementation Plan Generator (YAML Format)

Create a detailed, step-by-step implementation plan for $ARGUMENTS in **YAML format**. The plan should enable a skilled engineer with zero context about this codebase to successfully implement the feature. You MUST break down this task into micro tasks to better track line counts and ensure the plan is manageable.

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

## Phase 1C: Dependency Graph Analysis

After breaking down the feature into tasks, analyze task dependencies to determine optimal worktree grouping:

1. **Build the dependency graph**:
   - Examine each task's `depends_on` field
   - Create a graph where tasks are nodes and dependencies are edges
   - Identify all dependency relationships (both forward and backward)

2. **Identify connected components** (dependency chains):
   - Find groups of tasks where dependencies exist (e.g., Task 1→2→3)
   - Each connected component represents tasks that MUST execute sequentially
   - These groups should share ONE worktree to maintain dependency order
   - Label each group with a descriptive ID (e.g., "chain-1", "auth-flow")

3. **Identify independent tasks**:
   - Find tasks with no dependencies (empty `depends_on` array)
   - Find tasks that are not dependencies of other tasks
   - Each independent task gets its OWN worktree for parallel execution
   - Label each with a descriptive ID (e.g., "independent-3", "docs-task")

4. **Determine worktree grouping strategy**:
   - **ONE worktree per dependency chain**: All tasks in a chain execute sequentially in the same worktree
   - **ONE worktree per independent task**: Each independent task executes in isolation
   - This strategy enables maximum parallelism while preserving dependency ordering
   - Branch naming: `feature/<feature-slug>/chain-1`, `feature/<feature-slug>/independent-3`

5. **Document the groups**:
   - For each group, list the tasks it contains
   - Specify whether execution is sequential (chain) or parallel (independent)
   - Provide setup commands for creating and using each worktree
   - Include the merge strategy (independent branches merged separately to main)

**Example dependency analysis:**
- Tasks 1, 2, 4 form a chain: 1→2→4 (one worktree for sequential execution)
- Task 3 is independent (its own worktree for parallel execution)
- Task 5 depends on both 2 and 3 (forms its own chain after prerequisites)

This analysis produces the `worktree_groups` section in the YAML output, enabling engineers to work on multiple task groups in parallel while respecting dependencies.

## Phase 1D: Integration Task Generation

**CRITICAL: Detect Integration Points**

When generating implementation plans, you MUST detect integration points and generate explicit integration tasks to wire components together. Conductor automatically enhances integration task prompts with dependency file context to help agents understand implementation requirements.

### When to Generate Integration Tasks

Integration tasks are explicit, focused tasks that wire together previously completed components. Generate them when:

1. **Multi-component plans** (3+ independent modules/features)
   - Auth system connects to API router
   - Database client integrates with service layer
   - Cache wrapper integrates with data access layer
   - Multiple microservices need coordination

2. **After component boundaries are clear**
   - All component tasks in one group are defined
   - Component interfaces and APIs are specified
   - Integration points are documented

3. **Between dependent worktree groups**
   - Group 1: Component A implementation
   - Group 2: Component B implementation
   - **Integration task**: Wire A to B (depends on both groups)

4. **For cross-cutting concerns**
   - Security: Wire authentication to all endpoints
   - Logging: Inject logging into existing components
   - Monitoring: Add instrumentation to service layer

### When NOT to Generate Integration Tasks

- Single-component tasks (simple feature within one module)
- Tasks that are inherently part of component implementation
- Greenfield projects with sequential feature-based tasks
- When wiring is explicitly included in component task descriptions

### Integration Task Structure

Integration tasks include special metadata that Conductor uses to enhance prompts with dependency context:

```yaml
- task_number: 5
  name: "Wire auth module to router"
  type: integration  # REQUIRED - marks this as integration task
  agent: "fullstack-developer"
  files:
    - internal/auth/jwt.go
    - internal/api/router.go
    - cmd/main.go
  depends_on: [1, 2, 3, 4]  # Dependencies - all components being wired
  estimated_time: "45m"

  # Integration tasks have BOTH success_criteria AND integration_criteria
  success_criteria:
    - "Component-level check"
  integration_criteria:
    - "Router imports internal/auth package"
    - "auth.Middleware() registered in router.Use()"
    - "main.go initializes auth before router"
    - "Integration test passes"

  description: |
    Wire the JWT auth module to the API router.

    This task takes the completed auth module and connects it
    to the API router, ensuring all authentication middleware
    is properly integrated before route handling.

    ## What You Need to Know Before Starting

    You have access to completed components:
    - Task 1: JWT auth implementation (internal/auth/jwt.go)
    - Task 2: API router setup (internal/api/router.go)
    - Task 3: Server bootstrap (cmd/main.go)

    ## Integration Steps
    1. Read completed auth module interfaces
    2. Read router initialization patterns
    3. Import auth package in router.go
    4. Register middleware in router initialization
    5. Update main.go initialization order
    6. Add integration test verifying wired components work together
```

### Conductor's Automatic Prompt Enhancement

When Conductor executes an integration task (type: "integration" or with dependencies), it automatically enhances the prompt with dependency file context:

**Automatically added to task prompt:**
```
# INTEGRATION TASK CONTEXT

Before implementing, you MUST read these dependency files:

## Dependency: Task 1 - JWT Auth Implementation
**Files to read**:
- internal/auth/jwt.go

**WHY YOU MUST READ THESE**:
You need to understand the implementation of JWT Auth Implementation
to properly integrate it. Read these files to see:
- Exported functions and their signatures
- Data structures and types
- Error handling patterns
- Integration interfaces
```

This automatic enhancement ensures agents understand:
- What components they're integrating
- Which files contain the implementations
- Why understanding those files is critical
- How to discover integration interfaces

**The enhanced prompt is transparent** - agents see the full context including the original task description, so you don't need to repeat dependency details in the description.

### Integration Task Patterns

**Pattern 1: Component Wiring**
```yaml
- task_number: 8
  name: "Wire database layer to repository"
  type: integration
  depends_on: [5, 6, 7]  # DB client, transaction manager, connection pool
  files: [internal/repository/user.go, internal/db/connection.go]
  description: |
    Connect the database client to the repository layer,
    ensuring transactions and connection pooling are properly integrated.
```

**Pattern 2: Security Integration**
```yaml
- task_number: 12
  name: "Integrate auth middleware into API handlers"
  type: integration
  depends_on: [1, 3, 7, 9, 11]  # Auth, router, handlers
  files: [internal/api/middleware.go, internal/api/handlers.go]
  description: |
    Add authentication middleware to all protected API endpoints,
    ensuring proper token validation and authorization checks.
```

**Pattern 3: Multi-Service Coordination**
```yaml
- task_number: 20
  name: "Configure service mesh and inter-service communication"
  type: integration
  depends_on: [14, 15, 16, 17, 18, 19]  # All microservices
  files: [config/service-mesh.yaml, internal/discovery/resolver.go]
  description: |
    Set up service discovery and configure inter-service communication,
    enabling all microservices to locate and communicate with each other.
```

### Decision Tree: Should This Be an Integration Task?

```
Is this task wiring multiple components together?
├─ NO (single module, feature, or implementation)
│  └─ Use regular task
└─ YES (connecting multiple previously-completed components)
   ├─ Does the task require understanding completed implementations?
   │  └─ YES → Integration task (Conductor adds context automatically)
   └─ Is it truly independent work (no dependencies)?
      └─ YES → Regular task
      └─ NO (has dependencies) → Integration task
```

### Integration vs Regular Tasks: Quick Reference

| Aspect | Regular Task | Integration Task |
|--------|--------------|------------------|
| **Scope** | Implement one feature/module | Wire multiple features together |
| **Dependencies** | May have 0-2 | Typically 3+ |
| **Type field** | (not set) | "integration" |
| **Criteria fields** | success_criteria only | success_criteria + integration_criteria |
| **Prompt enhancement** | None | Automatic (Conductor adds dependency context) |
| **Agent capability** | Feature-focused | Full-stack/architecture-aware preferred |
| **File focus** | New files mostly | Mix of new + existing files |
| **Example** | "Implement JWT auth" | "Wire auth module to router" |

**IMPORTANT:** If you lack sufficient context to identify integration points, ASK THE USER for clarification before proceeding. Do not guess integration task boundaries.

## Phase 1E: Streaming Generation Strategy

**CRITICAL: You will generate this YAML plan INCREMENTALLY, not all at once.**

Large plans (15+ tasks) will exceed token limits if generated as a single monolithic task. You MUST break the generation process itself into manageable chunks.

### Use TodoWrite to Break Down Generation

Before starting to write the plan, create a todo list that breaks generation into phases:

```
Example Todo List:
- [ ] Create docs/plans directory structure
- [ ] Write YAML header, metadata, context, worktree groups, prerequisites (~200-400 lines)
- [ ] Generate tasks for first worktree group (monitor line count actively)
- [ ] Check line count - if >1900, complete file and create next file
- [ ] Generate tasks for next worktree group (continue monitoring)
- [ ] Generate testing strategy, commit strategy, pitfalls sections
- [ ] Generate resources and validation checklist
- [ ] Validate all YAML files are syntactically correct
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
   - Increment `current_file_line_count` by lines just written (including YAML structure)
   - Check: `if current_file_line_count > 1900 AND tasks_written < total_tasks`
   - If true: **STOP and split to new file NOW**

3. **When splitting**:
   - Complete current task fully (never split mid-task)
   - Ensure YAML structure is valid and complete
   - Write current buffer to: `docs/plans/<feature-slug>/plan-0N-<phase-name>.yaml`
   - Create new file: `plan-0(N+1)-<next-phase>.yaml`
   - Reset: `current_file_line_count = 0`
   - Continue with remaining content

### Write Incrementally, Don't Accumulate

**WRONG approach** ❌:
- Generate all tasks in memory → accumulate huge YAML text → write one file → token limit exceeded

**RIGHT approach** ✅:
- Write YAML header → check count → write task 1 → check count → write task 2 → check count
- When count > 1900: write current file → validate YAML → create next file → continue
- Each file is written as you go, not accumulated in memory

### Example: Generating a 25-Task YAML Plan

**Step 1: Create todo list** (as shown above)

**Step 2: Execute with active monitoring**:
```
[Starting plan-01-foundation.yaml]
Writing YAML header & metadata... (current: 50 lines)
Writing context section... (current: 150 lines)
Writing worktree_groups... (current: 300 lines)
Writing prerequisites... (current: 400 lines)
Writing Task 1 YAML... (current: 570 lines)
Writing Task 2 YAML... (current: 740 lines)
Writing Task 3 YAML... (current: 910 lines)
Writing Task 4 YAML... (current: 1080 lines)
Writing Task 5 YAML... (current: 1250 lines)
Writing Task 6 YAML... (current: 1420 lines)
Writing Task 7 YAML... (current: 1590 lines)
Writing Task 8 YAML... (current: 1760 lines)
Writing Task 9 YAML... (current: 1930 lines) ⚠️ LIMIT APPROACHING

DECISION: Current file has 1930 lines, at worktree group boundary
→ Complete YAML structure for plan-01-foundation.yaml
→ Validate YAML syntax (must be parseable)
→ Create plan-02-integration.yaml for remaining tasks

[Starting plan-02-integration.yaml]
Reset counter (current: 0 lines)
Writing plan: metadata... (current: 30 lines)
Writing Task 10 YAML... (current: 200 lines)
Writing Task 11 YAML... (current: 370 lines)
...continues until all tasks complete...
```

**Step 3: Validate YAML** (each file must be independently parseable)

### Key Principles

1. **Generation is multi-step** - Not a single "generate plan" task
2. **Active monitoring** - Check line count after every section
3. **Write as you go** - Don't accumulate in memory
4. **Split proactively** - At 1900 lines, not 3000 lines
5. **Use task boundaries** - Never split mid-task
6. **Validate YAML** - Each file must be syntactically correct

## Phase 2: YAML Plan Generation

Create a comprehensive implementation plan where `<feature-slug>` is derived from the feature description (e.g., "user-authentication", "payment-integration").

Output structure depends on task count:
- **1-15 tasks**: Single file `docs/plans/<feature-slug>.yaml`
- **16+ tasks**: Directory `docs/plans/<feature-slug>/` with split files

## Phase 2A: Objective Plan Splitting with Conductor Auto-Discovery (YAML-Specific)

When generating YAML task breakdowns, implement **metric-based plan splitting** to keep YAML files manageable while preserving complete task details and valid YAML structure. Split files enable Conductor's multi-file orchestration.

**IMPORTANT**: This section describes HOW to split when you detect the need. Phase 1D describes the ACTIVE MONITORING process you must follow during generation.

### Line Count Tracking for YAML (Active, Not Passive)

**CONTINUOUSLY track** the number of lines as you write each YAML section. This is NOT a one-time check at the end - it's ongoing monitoring during generation.

**Implementation (active monitoring):**
- Initialize `line_count = 0` when starting a new YAML plan file
- **After writing each YAML section** (header, task, support section): Increment `line_count`
- Track `tasks_detailed` (number of tasks fully written) vs `total_tasks` (total tasks identified)
- Account for YAML syntax overhead (indentation, list markers, object keys)
- **Check IMMEDIATELY**: Is `line_count > 1900`? If yes, prepare to split

### Active Monitoring Logic (Not Passive Trigger)

This is NOT a trigger you check once - it's a loop you execute continuously:

```
WHILE generating YAML plan content:
  Write next YAML section (header, task object, etc.)
  current_line_count += lines_just_written

  IF (current_line_count > 1900 AND tasks_remaining > 0 AND at_group_boundary):
    STOP writing to current file
    COMPLETE current YAML structure (close objects/lists)
    VALIDATE YAML is parseable
    WRITE current file to disk
    CREATE next file: plan-0N-<phase>.yaml
    START new YAML structure
    RESET current_line_count = 0
    CONTINUE with remaining content

  IF (all_tasks_written AND all_sections_complete):
    COMPLETE final YAML structure
    VALIDATE final YAML is parseable
    WRITE final file
    BREAK loop
```

**Critical rules:**
- **1900-2000 lines is the target range** - no subjective judgment
- **Never split mid-task** - complete the current task's YAML structure before splitting
- **Maintain valid YAML** - each split file must be independently parseable
- **Split at worktree group boundaries** - natural organizational boundaries
- **Only split if work remains** - don't create empty YAML files
- **Write files incrementally** - don't accumulate everything in memory first
- **Validate each file** - ensure YAML syntax is correct before moving to next file

### Split Strategy for YAML Files (Phase 2A Conductor Format)

When the trigger condition is met, create numbered YAML plan files in Phase 2A format for Conductor auto-discovery:

```
docs/plans/feature-name/
├── plan-01-phase-name.yaml        (2000 lines, initial phase)
├── plan-02-phase-name.yaml        (1800 lines, next phase)
└── plan-03-phase-name.yaml        (1200 lines, final phase)
```

**File naming convention (Phase 2A):**
- `plan-NN-<descriptive-phase-name>.yaml` where NN is 01, 02, 03, etc.
- Use descriptive phase names that indicate content (e.g., `database`, `api`, `testing`)
- This format enables Conductor's auto-discovery: `conductor run docs/plans/feature-name/`
- Example: `plan-01-database.yaml`, `plan-02-api-implementation.yaml`, `plan-03-testing.yaml`

**Why Phase 2A naming:**
- Conductor automatically discovers files matching `plan-*.yaml` pattern
- Sequential numbering (01, 02, 03) ensures proper load order
- Descriptive names make file purposes clear without worktree group IDs
- Cross-file dependencies work across all numbered plan files

### YAML-Specific Considerations

When splitting YAML plans, ensure each file:

1. **Is valid YAML independently** - complete document structure in each file
2. **Has proper root structure** - each file should have a `plan:` root object
3. **Contains complete task objects** - never split a task's YAML structure across files
4. **Includes necessary metadata** - each file has enough context to be understood
5. **Uses proper references** - if tasks reference other tasks, include notes about cross-file dependencies

**Example valid split file structure:**

```yaml
# docs/plans/feature-name/1-chain-1.yaml
plan:
  metadata:
    feature_name: "User Authentication"
    file_sequence: 1
    total_files: 3
    worktree_group: "chain-1"
    tasks_in_file: [1, 2, 3, 4, 5]
    next_file: "2-chain-2.yaml"

  tasks:
    - task_number: 1
      # ... full task details

    - task_number: 2
      # ... full task details

    # ... more tasks in this worktree group
```

### Metrics Tracked During YAML Generation

Track these objective metrics throughout YAML plan generation:

1. **line_count**: Total lines written to current YAML file (including structure)
2. **tasks_completed**: Number of tasks fully detailed in current file
3. **total_tasks**: Total number of tasks identified in Phase 1C
4. **current_worktree_group**: Which group is currently being documented
5. **yaml_depth**: Current indentation depth (to ensure proper structure)
6. **worktree_group_boundaries**: List of group transitions (when to potentially split)

### Decision Logic (Pure Metrics-Based, Executed Continuously)

**This logic runs DURING generation, not after:**

```
BEFORE starting generation:
  - Create todo list breaking generation into phases (use TodoWrite)
  - Mark first todo as in_progress

WHILE generating YAML task breakdowns:
  - Write next YAML section (header, task object, support section) to current file
  - Increment line_count by lines just written (including YAML structure)
  - If section is a task: increment tasks_completed

  IMMEDIATELY AFTER each section:
    IF line_count > 1900 AND tasks_completed < total_tasks:
      - Check if current position is at worktree group boundary
      - IF yes (natural boundary exists):
        - COMPLETE current YAML structure (close all objects/lists)
        - VALIDATE YAML is parseable (critical!)
        - WRITE current plan file to disk (plan-0N-<phase>.yaml)
        - UPDATE todo: Mark current generation phase as complete
        - CREATE new YAML plan file (plan-0(N+1)-<next-phase>.yaml)
        - START new file with proper YAML structure (plan: metadata: etc.)
        - Reset line_count = 0 for new file
        - UPDATE todo: Mark next generation phase as in_progress
        - CONTINUE with remaining tasks in new file
      - IF no (mid-group):
        - CONTINUE in current file until group boundary reached
        - THEN split at next group boundary

    IF tasks_completed == total_tasks AND all_sections_written:
      - COMPLETE current YAML file structure
      - VALIDATE final YAML is parseable (critical!)
      - WRITE final plan file to disk
      - UPDATE todo: Mark final phase as complete
      - BREAK while loop
```

**Key principles:**
- No subjective judgment - only objective line counts and task boundaries
- Active monitoring - check after EACH section written
- Incremental writing - files written to disk as you go, not accumulated
- YAML validation - each file must be independently parseable
- Todo tracking - update your todo list as you complete each generation phase

### Output Confirmation

When YAML plan generation completes, report to the user:

**Single file output:**
```
YAML plan created: docs/plans/<feature-slug>.yaml
- Total tasks: 12
- Total lines: 1,543
- Worktree groups: 3 (chain-1, independent-1, independent-2)
- Format: Single YAML file
- YAML validation: PASSED
- Conductor support: Direct (conductor run docs/plans/<feature-slug>.yaml)
```

**Split file output (Phase 2A format):**
```
YAML plan created in: docs/plans/<feature-name>/
- Total tasks: 25
- Plan files created: 3 (Phase 2A format)
  - plan-01-database.yaml (2,023 lines, tasks 1-10)
  - plan-02-api.yaml (1,987 lines, tasks 11-18)
  - plan-03-testing.yaml (1,156 lines, tasks 19-25)
- Worktree groups: 5 (chain-1, chain-2, independent-1, independent-2, independent-3)
- Auto-discovery: Enabled (Conductor finds plan-*.yaml files)
- YAML validation: ALL FILES PASSED

Run with Conductor: conductor run docs/plans/<feature-name>/
Start with: docs/plans/<feature-name>/plan-01-database.yaml - Task 1
```

### YAML Validation Requirements

Before finalizing any YAML file (especially after splitting) use `conductor validate command`:

1. **Parse the YAML** - ensure it's syntactically valid
2. **Check structure** - verify all required top-level keys exist
3. **Validate references** - ensure cross-task references make sense
4. **Confirm completeness** - no truncated task objects
5. **Verify indentation** - consistent 2-space indentation throughout

**If validation fails:** Do not proceed with splitting. Fix the YAML structure and retry.

### Why This YAML-Specific Approach Works

1. **Objective triggers**: 2000-line limit is clear and measurable
2. **Valid YAML guaranteed**: Each file is independently parseable
3. **Complete task objects**: Never break YAML structure across files
4. **Natural boundaries**: Worktree groups provide logical split points
5. **Machine-readable**: YAML parsers can consume each file independently
6. **Automation-friendly**: Tools can programmatically process split plans
7. **Maintainable**: Each YAML file remains readable and properly structured

The YAML plan MUST follow this exact structure:

```yaml
conductor:
  # Default agent for tasks without explicit agent assignment
  default_agent: general-purpose

  # NOTE: Concurrency is controlled via CLI flags or config file, NOT plan files:
  #   - CLI: conductor run plan.yaml --max-concurrency 5
  #   - Config: .conductor/config.yaml (max_concurrency: 5)

  # Quality control configuration with multi-agent support
  quality_control:
    enabled: true
    review_agent: quality-control
    retry_on_red: 2

    # Multi-agent QC configuration
    agents:
      # mode: auto - Automatically selects QC agents based on file types (RECOMMENDED)
      # mode: explicit - Uses ONLY agents specified in explicit_list
      # mode: mixed - Combines auto-selected agents with those in additional list
      mode: auto

      # explicit_list: Required when mode is explicit - list of agents to use
      # Leave empty by default unless using explicit mode
      # Example: ["typescript-pro", "python-pro"] when mode is explicit
      explicit_list: []

      # additional: Extra agents to add when mode is mixed
      # Leave empty by default to let auto-selection handle agent matching
      # Added to auto-selected agents when mode is mixed
      # Example: ["custom-agent"] for mixed mode
      additional: []

      # blocked: List of agent names to exclude from QC selection
      # Leave empty by default, no exclusions
      # Filters auto-selected agents in auto and mixed modes
      # Example: ["slow-agent"] to prevent certain agents from reviewing
      blocked: []

  # Worktree groups for parallel execution while respecting dependencies
  worktree_groups:
    - group_id: "chain-1"
      description: "Tasks 1→2→4 (core authentication flow)"
      tasks: [1, 2, 4]
      branch: "feature/<feature-slug>/chain-1"
      execution_model: "sequential"
      isolation: "separate-worktree"
      rationale: "These tasks form a dependency chain and must execute in order"
      setup_commands: |
        # Create worktree for this dependency chain
        git worktree add ../wt-chain-1 -b feature/<feature-slug>/chain-1
        cd ../wt-chain-1

        # Task 1: [Task name]
        # [Implement and test Task 1]
        git add .
        git commit -m "type: task 1 commit message"

        # Task 2: [Task name] (depends on Task 1)
        # [Implement and test Task 2]
        git add .
        git commit -m "type: task 2 commit message"

        # Task 4: [Task name] (depends on Task 2)
        # [Implement and test Task 4]
        git add .
        git commit -m "type: task 4 commit message"

        # When ready, merge to main
        git checkout main
        git merge feature/<feature-slug>/chain-1
        git push origin main

    - group_id: "independent-3"
      description: "Task 3 (documentation updates - no dependencies)"
      tasks: [3]
      branch: "feature/<feature-slug>/independent-3"
      execution_model: "parallel"
      isolation: "separate-worktree"
      rationale: "Independent task that can execute in parallel with chain-1"
      setup_commands: |
        # Create worktree for this independent task
        git worktree add ../wt-independent-3 -b feature/<feature-slug>/independent-3
        cd ../wt-independent-3

        # Task 3: [Task name]
        # [Implement and test Task 3]
        git add .
        git commit -m "type: task 3 commit message"

        # When ready, merge to main
        git checkout main
        git merge feature/<feature-slug>/independent-3
        git push origin main

    - group_id: "chain-5"
      description: "Task 5 (integration task requiring Tasks 2 and 3)"
      tasks: [5]
      branch: "feature/<feature-slug>/chain-5"
      execution_model: "sequential"
      isolation: "separate-worktree"
      rationale: "Depends on Tasks 2 and 3; must wait for both to complete"
      setup_commands: |
        # Wait for prerequisite tasks to complete and merge to main
        # Then create worktree starting from updated main
        git checkout main
        git pull origin main
        git worktree add ../wt-chain-5 -b feature/<feature-slug>/chain-5
        cd ../wt-chain-5

        # Task 5: [Task name] (depends on Tasks 2 and 3)
        # [Implement and test Task 5]
        git add .
        git commit -m "type: task 5 commit message"

        # When ready, merge to main
        git checkout main
        git merge feature/<feature-slug>/chain-5
        git push origin main

plan:
  # Metadata about the implementation plan
  metadata:
    feature_name: "Descriptive Feature Name"
    created: "YYYY-MM-DD"
    target: "Brief description of what we're building"
    estimated_tasks: <number>

  # Context for the engineer implementing this
  context:
    framework: "Framework/Language name"
    architecture: "Architecture pattern (e.g., Clean Architecture, MVC)"
    test_framework: "Test framework name (e.g., Jest, PyTest)"
    other_context:
      - "Additional context point 1"
      - "Additional context point 2"
    expectations:
      - "Write tests BEFORE implementation (TDD)"
      - "Commit frequently (after each completed task)"
      - "Follow existing code patterns"
      - "Keep changes minimal (YAGNI - You Aren't Gonna Need It)"
      - "Avoid duplication (DRY - Don't Repeat Yourself)"
      - "Use worktrees for parallel development when tasks are independent"

  # Prerequisites checklist
  prerequisites:
    - item: "Required tools installed"
      details: "List of tools: Node.js 18+, npm, git worktree support"
      verified: false
    - item: "Development environment setup"
      details: "Environment variables configured, database running"
      verified: false
    - item: "Access to necessary services/APIs"
      details: "API keys, service credentials"
      verified: false
    - item: "Branch created from correct base"
      details: "feature/<feature-name> from main"
      verified: false
    - item: "Git worktrees understood"
      details: "Familiarity with git worktree commands for parallel development"
      verified: false
    - item: "Task 0: Development environment validation (BLOCKING)"
      details: |
        MUST be completed before any implementation tasks begin.
        For Python projects:
          - mypy_path configured in pyproject.toml or mypy.ini
          - pytest pythonpath configured (PYTHONPATH or pytest.ini)
          - conftest.py with shared fixtures exists
          - Run: mypy --config-file=pyproject.toml src/
        For TypeScript projects:
          - tsconfig.json with strict mode enabled
          - ESLint configuration present
          - Run: tsc --noEmit && eslint .
        For Go projects:
          - go.mod present with correct module path
          - golangci-lint configured
          - Run: go vet ./... && golangci-lint run
      blocking: true
      commands:
        python:
          - "python -m mypy --version"
          - "python -m pytest --collect-only"
          - "cat pyproject.toml | grep mypy_path"
        typescript:
          - "npx tsc --version"
          - "npx eslint --version"
          - "cat tsconfig.json | grep strict"
        go:
          - "go version"
          - "golangci-lint --version"
          - "go list -m"
      verified: false

  # Common pitfalls reference (review BEFORE starting tasks)
  common_pitfalls_reference:
    purpose: |
      Document common mistakes that have occurred in previous implementations
      using this template. Review this section BEFORE starting any tasks to
      avoid repeating these errors.

    python_pitfalls:
      - pitfall: "Using 'from src.' imports instead of proper module imports"
        error_example: |
          # WRONG
          from src.bot.handlers import MessageHandler

          # RIGHT (when project root is in PYTHONPATH)
          from bot.handlers import MessageHandler
        why: "ModuleNotFoundError when running tests or scripts from different directories"
        detection: "grep -r 'from src\\.' ."
        fix: "Update import statements and ensure PYTHONPATH or pytest.ini pythonpath is configured"

      - pitfall: "Incomplete async context managers (missing __aexit__)"
        error_example: |
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
        why: "Runtime error when using 'async with' statement"
        detection: "grep -A 5 '__aenter__' | grep -L '__aexit__'"
        test_pattern: |
          async def test_context_manager():
              async with DatabaseConnection() as conn:
                  # Should enter and exit without errors
                  assert conn is not None

      - pitfall: "Missing bot_user_id initialization in Discord bots"
        error_example: |
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
        why: "NoneType errors when bot tries to use its own ID"
        detection: "grep -n 'bot_user_id' | grep '= None' # Check if it's ever reassigned"
        fix: "Initialize in on_ready or setup method, add assertion in critical paths"

      - pitfall: "No error handling in event handlers"
        error_example: |
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
        why: "Unhandled exceptions crash the entire bot"
        detection: "grep -A 10 'async def on_' | grep -L 'try:'"
        test_pattern: |
          async def test_error_handling():
              with pytest.raises(RuntimeError):
                  await handler.on_message(malformed_message)
              # Handler should log but not crash

      - pitfall: "Using Dict/List instead of dict/list in type hints (Python 3.9+)"
        error_example: |
          # WRONG (old style)
          from typing import Dict, List
          def process(data: Dict[str, List[int]]) -> None: ...

          # RIGHT (Python 3.9+ builtin generics)
          def process(data: dict[str, list[int]]) -> None: ...
        why: "Deprecated in Python 3.9+, will be removed in future versions"
        detection: "grep -r 'from typing import Dict\\|List\\|Tuple' ."
        fix: "Use lowercase dict, list, tuple from builtins"

      - pitfall: "mypy not configured or not running"
        error_example: |
          # Project has type hints but mypy never configured
          # Type errors exist but are never caught
        why: "Type hints are useless without type checking"
        detection: |
          # Check if mypy configuration exists
          ls pyproject.toml setup.cfg mypy.ini

          # Check if mypy is in dev dependencies
          cat pyproject.toml | grep mypy
        fix: |
          # Add to pyproject.toml
          [tool.mypy]
          python_version = "3.11"
          warn_return_any = true
          warn_unused_configs = true
          disallow_untyped_defs = true
          mypy_path = "src"

          # Run mypy
          python -m mypy src/

    typescript_pitfalls:
      - pitfall: "Missing null checks with optional properties"
        error_example: |
          // WRONG
          function getUsername(user: User): string {
              return user.name.toLowerCase(); // Crashes if name is undefined
          }

          // RIGHT
          function getUsername(user: User): string {
              return user.name?.toLowerCase() ?? 'anonymous';
          }
        why: "Runtime errors when optional properties are undefined"
        detection: "eslint rule: @typescript-eslint/no-non-null-assertion"

      - pitfall: "Using 'any' type instead of proper types"
        error_example: |
          // WRONG
          function process(data: any): any { ... }

          // RIGHT
          function process(data: InputData): OutputData { ... }
        why: "Defeats the purpose of TypeScript"
        detection: "eslint rule: @typescript-eslint/no-explicit-any"

    go_pitfalls:
      - pitfall: "Not checking errors"
        error_example: |
          // WRONG
          file, _ := os.Open("config.json")

          // RIGHT
          file, err := os.Open("config.json")
          if err != nil {
              return fmt.Errorf("failed to open config: %w", err)
          }
        why: "Silent failures and unexpected behavior"
        detection: "golangci-lint with errcheck linter enabled"

      - pitfall: "Goroutine leaks (not closing channels or contexts)"
        error_example: |
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
        why: "Memory leaks and resource exhaustion"
        detection: "go test -race ./..."

    cross_language_pitfalls:
      - pitfall: "Not running formatters before commit"
        languages:
          python: "black . && ruff check ."
          typescript: "prettier --write . && eslint --fix ."
          go: "gofmt -w . && goimports -w ."
        why: "CI failures, inconsistent code style"

      - pitfall: "Tests not running in CI/CD"
        detection: |
          # Check if CI configuration exists
          ls .github/workflows/ .gitlab-ci.yml

          # Verify test commands are present
          cat .github/workflows/*.yml | grep -i test
        why: "Broken code gets merged"

      - pitfall: "Hardcoded paths or credentials"
        detection: |
          grep -r '/Users/' .
          grep -r '/home/' .
          grep -r 'password.*=' .
          grep -r 'api_key.*=' .
        why: "Works locally, fails in production or on other machines"

  # Development environment configuration
  development_environment:
    purpose: |
      Validate that the development environment is correctly configured
      BEFORE starting implementation tasks. This prevents common setup issues.

    setup_reference: |
      **IMPORTANT**: Before starting tasks, validate your environment is configured correctly.

      See [Development Setup Guide](ai_docs/development-setup.md) for comprehensive validation instructions including:
      - Python: mypy, pytest, black, ruff configuration
      - TypeScript: tsc, eslint, prettier configuration
      - Go: go vet, gofmt, golangci-lint configuration

      Quick validation commands are in prerequisites section (lines 883-896).

  # Detailed task breakdown
  tasks:
    - task_number: 1
      name: "Descriptive Task Name"
      agent: "agent-name"  # REQUIRED: Reference to agent from ~/.claude/agents/
      worktree_group: "chain-1"
      files:  # REQUIRED: Flat list of file paths (simple strings only)
        - "path/to/file1.ext"
        - "path/to/file2.ext"
      # IMPORTANT: files MUST be a simple flat list of strings
      # WRONG (nested structure):
      #   files:
      #     source: ["file1.go"]
      #     test: ["file1_test.go"]
      # CORRECT (flat list):
      #   files: ["file1.go", "file1_test.go"]
      depends_on: []  # Empty if no dependencies, otherwise [2, 3] for tasks 2 and 3
      estimated_time: "30m"  # Options: 5m, 15m, 30m, 1h, 2h

      # SUCCESS CRITERIA - TASK-LEVEL FIELDS FOR CONDUCTOR QC
      # These fields are parsed DIRECTLY by conductor for per-criterion verification
      success_criteria:  # TASK-LEVEL field (NOT nested under verification)
        - "First success criterion - specific, measurable outcome"
        - "Second success criterion - what must be true for task to pass"
        - "Third success criterion - verifiable condition"
      test_commands:  # TASK-LEVEL field for automated verification
        - "go test ./path/to/package/ -v"
        - "npm test -- --grep 'feature'"
        # Commands that verify the success criteria are met

      # IMPORTANT: SUCCESS_CRITERIA FIELD PLACEMENT
      # =============================================
      # Conductor's YAML parser ONLY reads success_criteria and test_commands when they
      # are DIRECT task-level fields (at the same indentation as name, files, depends_on).
      #
      # WRONG - parser ignores this:
      #   verification:
      #     success_criteria:      # NESTED - NOT parsed by conductor
      #       - "criterion"
      #
      # CORRECT - parser reads this:
      #   success_criteria:        # TASK-LEVEL - parsed by conductor
      #     - "criterion"
      #   test_commands:           # TASK-LEVEL - parsed by conductor
      #     - "go test ./..."
      #   verification:            # Keep for human documentation
      #     automated_tests: ...
      #
      # The verification section below still serves as human-readable documentation
      # for quality gates, but the task-level fields are what conductor actually uses
      # for per-criterion QC verification.

      # FOR INTEGRATION TASKS: Add BOTH criteria types
      # Component tasks (no deps) → success_criteria only
      # Integration tasks (type: "integration") → BOTH
      #
      #   success_criteria:        # Component works standalone
      #     - "Function returns correct value"
      #   integration_criteria:    # Component integrates correctly
      #     - "Calls auth middleware"

      # Agent Assignment Guidelines:
      # Based on task type and technology stack, assign appropriate agents:
      #   - Go code: golang-pro
      #   - Python code: python-pro
      #   - JavaScript/TypeScript: javascript-pro, typescript-pro
      #   - Testing tasks: test-automator
      #   - Documentation: technical-documentation-specialist
      #   - Database work: database-optimizer or database-admin
      #   - Performance: performance-engineer
      #   - General tasks: general-purpose
      #
      # Always verify the agent exists in the discovered agent list.
      # Every task MUST have an agent assigned.

      description: |
        2-3 sentences explaining WHAT you're building and WHY.
        This should provide clear context for the engineer.

      test_first:
        test_file: "path/to/test_file.ext"

        structure:
          - "describe/test block for [feature]"
          - "test case 1: [specific behavior]"
          - "test case 2: [specific edge case]"
          - "test case 3: [error condition]"

        mocks:
          - "dependency1"
          - "dependency2"

        fixtures:
          - "fixture1"
          - "factory1"

        assertions:
          - "outcome1"
          - "outcome2"

        edge_cases:
          - "edge case 1"
          - "edge case 2"

        example_skeleton: |
          # COMPLETE test patterns with all essential elements

          # Python example (pytest with async support)
          # tests/test_handlers.py
          import pytest
          from bot.handlers import MessageHandler
          from bot.models import Message

          # CORRECT import pattern (no 'from src.')
          # Assumes src/ is in PYTHONPATH via pytest.ini

          @pytest.mark.asyncio
          async def test_message_handler_success():
              """Test message handler processes valid messages."""
              # Arrange
              handler = MessageHandler()
              message = Message(content="test", author_id=123)

              # Act
              result = await handler.process(message)

              # Assert
              assert result is not None
              assert result.status == "success"

          @pytest.mark.asyncio
          async def test_message_handler_error_handling():
              """Test handler gracefully handles errors."""
              # Arrange
              handler = MessageHandler()
              invalid_message = None

              # Act & Assert - should NOT raise
              try:
                  result = await handler.process(invalid_message)
                  assert result.status == "error"
              except Exception as e:
                  pytest.fail(f"Handler should catch errors, not raise: {e}")

          # COMPLETE async context manager test
          @pytest.mark.asyncio
          async def test_database_context_manager():
              """Test database connection context manager lifecycle."""
              from bot.database import DatabaseConnection

              # Act - use async context manager
              async with DatabaseConnection() as conn:
                  # Assert - connection established
                  assert conn is not None
                  assert conn.is_connected

              # Assert - connection properly closed after exit
              # This ensures __aexit__ was implemented correctly
              assert not conn.is_connected

          # RuntimeError enforcement test
          @pytest.mark.asyncio
          async def test_bot_user_id_required():
              """Test that operations fail without bot_user_id initialization."""
              from bot.client import DiscordBot

              bot = DiscordBot(token="fake_token")
              # bot_user_id not set (on_ready not called)

              # Should raise RuntimeError if bot_user_id not initialized
              with pytest.raises(RuntimeError, match="bot_user_id not initialized"):
                  await bot.filter_own_messages(message)

          # TypeScript/JavaScript example (Jest)
          // tests/handlers.test.ts
          import { MessageHandler } from '../src/handlers/MessageHandler';
          import { Message } from '../src/models/Message';

          describe('MessageHandler', () => {
            let handler: MessageHandler;

            beforeEach(() => {
              // Reset state before each test
              handler = new MessageHandler();
            });

            it('should process valid messages successfully', async () => {
              // Arrange
              const message: Message = {
                content: 'test message',
                authorId: 123,
                timestamp: new Date()
              };

              // Act
              const result = await handler.process(message);

              // Assert
              expect(result).toBeDefined();
              expect(result.status).toBe('success');
              expect(result.processedAt).toBeInstanceOf(Date);
            });

            it('should handle null messages gracefully', async () => {
              // Arrange
              const invalidMessage = null as unknown as Message;

              // Act
              const result = await handler.process(invalidMessage);

              // Assert - should not throw, should return error status
              expect(result.status).toBe('error');
              expect(result.error).toContain('Invalid message');
            });

            it('should enforce required initialization', () => {
              // Arrange
              const handler = new MessageHandler();
              // Don't call initialize()

              // Act & Assert
              expect(() => handler.process(message))
                .rejects
                .toThrow('Handler not initialized');
            });
          });

          # Go example (standard testing package)
          // handlers_test.go
          package handlers

          import (
            "context"
            "testing"
            "time"
          )

          func TestMessageHandler_ProcessSuccess(t *testing.T) {
            // Arrange
            handler := NewMessageHandler()
            ctx := context.Background()
            message := &Message{
              Content: "test message",
              AuthorID: 123,
            }

            // Act
            result, err := handler.Process(ctx, message)

            // Assert
            if err != nil {
              t.Fatalf("expected no error, got: %v", err)
            }
            if result == nil {
              t.Fatal("expected result, got nil")
            }
            if result.Status != "success" {
              t.Errorf("expected status 'success', got: %s", result.Status)
            }
          }

          func TestMessageHandler_ErrorHandling(t *testing.T) {
            // Arrange
            handler := NewMessageHandler()
            ctx := context.Background()
            var nilMessage *Message = nil

            // Act
            result, err := handler.Process(ctx, nilMessage)

            // Assert - should return error, not panic
            if err == nil {
              t.Error("expected error for nil message, got nil")
            }
            if result != nil {
              t.Error("expected nil result on error")
            }
          }

          func TestMessageHandler_ContextCancellation(t *testing.T) {
            // Arrange
            handler := NewMessageHandler()
            ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
            defer cancel()

            message := &Message{Content: "test"}
            time.Sleep(2 * time.Millisecond) // Ensure context expires

            // Act
            _, err := handler.Process(ctx, message)

            // Assert - should respect context cancellation
            if err == nil || err != context.DeadlineExceeded {
              t.Errorf("expected context.DeadlineExceeded, got: %v", err)
            }
          }

      implementation:
        approach: |
          Detailed explanation of how to implement this task.
          Include architectural decisions and reasoning.

        code_structure: |
          // Provide skeleton/pseudocode showing structure
          class FeatureName {
            constructor(dependencies) {
              this.deps = dependencies;
            }

            async mainMethod(params) {
              // Validation
              // Business logic
              // Return result
            }
          }

        key_points:
          - point: "Follow pattern from similar implementation"
            reference: "path/to/similar/file.ext:123"
          - point: "Use existing utility function"
            reference: "utilityName from path/to/utility.ext"
          - point: "Error handling approach"
            details: "Use try-catch with custom error types"
          - point: "Validation requirements"
            details: "Validate input using Zod schema"

        critical_patterns:
          purpose: |
            Complete, copy-paste-ready code templates for common patterns.
            These prevent the errors documented in common_pitfalls_reference.

          async_context_manager_python:
            description: "Full async context manager implementation (both __aenter__ AND __aexit__)"
            complete_code: |
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

          initialization_order_python:
            description: "Proper initialization order with runtime enforcement"
            complete_code: |
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

          error_handling_event_handlers_python:
            description: "Error handling in async event handlers"
            complete_code: |
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

          correct_imports_python:
            description: "Correct import patterns (no 'from src.')"
            complete_code: |
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

          type_hints_modern_python:
            description: "Modern type hints (Python 3.9+, no typing.Dict/List)"
            complete_code: |
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

          typescript_null_safety:
            description: "TypeScript null safety with optional chaining"
            complete_code: |
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

          go_error_handling:
            description: "Go error handling with error wrapping"
            complete_code: |
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

          go_context_cancellation:
            description: "Go goroutine with context cancellation (prevent leaks)"
            complete_code: |
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

        integration:
          imports:
            - "import { SomeClass } from './path/to/module'"
            - "import type { SomeType } from './types'"

          services_to_inject:
            - "DatabaseService from services/database"
            - "LoggerService from services/logger"

          config_values:
            - "API_TIMEOUT from config/constants"

          # For empty/placeholder values, use empty arrays:
          # services_to_inject: []
          # config_values: []

          error_handling:
            - "Throw ValidationError for invalid input"
            - "Catch and log DatabaseError, return null"

      verification:  # Human-readable documentation for quality gates (supplements task-level fields)
        # NOTE: This section provides detailed documentation for engineers.
        # The TASK-LEVEL success_criteria and test_commands fields above are what
        # conductor's QC system actually parses for per-criterion verification.
        automated_tests:
          command: "language-specific test command (pytest, npm test, go test ./...)"
          expected_output: |
            All tests passing for this task's test file

        success_criteria:  # Duplicates task-level field for human readability
          - "All tests pass"
          - "Type checker passes (mypy, tsc, go vet)"
          - "Code compiles/runs without errors"

      code_quality:  # REQUIRED - language-specific quality pipeline
        # Choose the appropriate section based on your task's language
        # Include ONLY the language section that applies to your task

        python:
          full_quality_pipeline:
            command: |
              # Run all quality checks in sequence
              python -m black . && \
              python -m mypy src/ && \
              python -m pytest
            description: "Complete quality check pipeline for Python"
            exit_on_failure: true

        typescript:
          full_quality_pipeline:
            command: |
              # Run all quality checks in sequence
              npx prettier --write . && \
              npx tsc --noEmit && \
              npm test
            description: "Complete quality check pipeline for TypeScript"
            exit_on_failure: true

        go:
          full_quality_pipeline:
            command: |
              # Run all quality checks in sequence
              gofmt -w . && \
              go vet ./... && \
              go test ./...
            description: "Complete quality check pipeline for Go"
            exit_on_failure: true

      commit:
        # NOTE: This 'commit' object is for PLANNING the commit you intend to make.
        type: "feat"  # Options: feat, fix, test, refactor, docs, chore
        message: "descriptive commit message"
        body: |
          Optional commit body explaining why this change was made.

        files:
          - "path/to/implementation.ext"
          - "path/to/test.ext"

  # Advanced Examples: Detailed Quality Control (Expansion of Required Fields)
  #
  # NOTE: The examples below show advanced usage of verification and code_quality sections.
  # Basic verification and code_quality sections are REQUIRED for every task (see template above).
  # Use these examples as reference for comprehensive quality gates when needed.

  ## Advanced code_quality Configuration

  # For detailed tool-by-tool configuration beyond the basic full_quality_pipeline,
  # see the Quality Gates Reference guide.

  advanced_code_quality_reference: |
    For comprehensive tool-by-tool configuration details, see [Quality Gates Reference](ai_docs/quality-gates-reference.yaml) which includes:
    - Per-tool blocking/non-blocking configuration
    - Common issues and fixes for mypy, tsc, go vet
    - Coverage checking setup
    - Pre-commit checklist by language

  advanced_code_quality_examples_reference: |
    For detailed quality gate definitions with must_pass flags, auto_fix commands, and execution pipelines,
    see [Quality Gates Reference](ai_docs/quality-gates-reference.yaml).

  ## Advanced verification Configuration

  # For advanced verification configuration details, see the Quality Gates Reference guide.

  advanced_verification_reference: |
    For comprehensive quality gate definitions with manual testing, automated tests, and success criteria,
    see [Quality Gates Reference](ai_docs/quality-gates-reference.yaml).

  ## Task Lifecycle: From Plan to Completion

  # Tasks evolve as they move through execution. Here's the complete lifecycle:
  #
  # Initial state (when plan is first created):
  # ```yaml
  # - task_number: 1
  #   name: "Add user model"
  #   depends_on: []
  #   estimated_time: "2h"
  #
  #   commit:
  #     type: "feat"
  #     message: "add user model"
  #     body: "Initial user data structure"
  #     files: ["models/user.go", "models/user_test.go"]
  # ```
  #
  # After execution (conductor adds tracking fields):
  # ```yaml
  # - task_number: 1
  #   name: "Add user model"
  #   depends_on: []
  #   estimated_time: "2h"
  #   status: "completed"              # NEW - added by conductor
  #   completed_at: "2025-11-10"       # NEW - added by conductor
  #   git_commit: "abc123def456..."    # NEW - added by conductor
  #
  #   commit:
  #     type: "feat"
  #     message: "add user model"
  #     body: "Initial user data structure"
  #     files: ["models/user.go", "models/user_test.go"]
  # ```
  #
  # The 'commit' object remains unchanged - it documents what commit was made and why.
  # The new fields track when the task was completed and which commit resulted.

    # Task 2 example (depends on Task 1)
    - task_number: 2
      name: "Next Task Name"
      agent: "another-agent"  # REQUIRED: Different agent for this task based on work type
      worktree_group: "chain-1"
      files:
        - "path/to/another/file.ext"
      depends_on: [1]  # Depends on task 1 - same worktree
      estimated_time: "15m"
      # ... (same structure as task 1)

    # Task 3 example (independent)
    - task_number: 3
      name: "Independent Task Name"
      agent: "docs-agent"
      worktree_group: "independent-3"
      files:
        - "docs/feature.md"
      depends_on: []  # No dependencies - can run in parallel
      estimated_time: "20m"
      # ... (same structure as task 1)

    # Task 4 example (continues chain-1)
    - task_number: 4
      name: "Build on Task 2"
      agent: "agent-name"
      worktree_group: "chain-1"
      files:
        - "path/to/file3.ext"
      depends_on: [2]  # Depends on task 2 - same worktree as tasks 1 and 2
      estimated_time: "45m"
      # ... (same structure as task 1)

    # Task 5 example (depends on multiple tasks from different groups)
    - task_number: 5
      name: "Integration Task"
      agent: "integration-agent"
      worktree_group: "chain-5"
      files:
        - "path/to/integration.ext"
      depends_on: [2, 3]  # Depends on tasks from different worktrees
      estimated_time: "1h"
      description: |
        This task integrates work from both chain-1 (task 2) and independent-3 (task 3).
        Must wait for both prerequisite branches to merge to main before starting.
      # ... (same structure as task 1)

  ## Task Status Tracking (Runtime Fields)

  # The following fields are automatically managed by Conductor during execution.
  # DO NOT manually define these in your initial plan - they will be added/updated automatically.
  #
  # Reserved runtime fields added by Conductor:
  #   - status: "pending" | "in_progress" | "completed"
  #   - completed_at: "YYYY-MM-DD" (or completed_date for variations)
  #   - git_commit: "full-git-hash" (actual commit hash after task completion)
  #
  # Example of a task AFTER completion:
  # ```yaml
  # - task_number: 3
  #   name: "Implement authentication"
  #   status: "completed"              # Added by conductor
  #   completed_at: "2025-11-10"       # Added by conductor
  #   git_commit: "07b195b..."         # Added by conductor (actual commit hash)
  #
  #   # Original planning fields:
  #   commit:
  #     type: "feat"
  #     message: "add authentication"
  #     files: ["auth.go"]
  # ```
  #
  # IMPORTANT: The 'commit' object is for PLANNING (what commit you intend to make).
  #            The 'git_commit' field is for TRACKING (the actual commit hash after completion).
  #            Both should exist in completed tasks - they serve different purposes.

  # Testing strategy
  testing_strategy:
    unit_tests:
      location: "src/**/*.test.ts"
      naming_convention: "*.test.ts or *.spec.ts"
      run_command: "npm test"
      coverage_target: "80%"
      coverage_command: "npm run test:coverage"

    integration_tests:
      location: "tests/integration"
      what_to_test:
        - "API endpoint interactions"
        - "Database operations"
        - "Service integrations"
      setup_required:
        - "Test database with seed data"
        - "Mock external services"
        - "Test fixtures and factories"
      run_command: "npm run test:integration"

    e2e_tests:
      enabled: true  # or false if not applicable
      location: "tests/e2e"
      critical_flows:
        - "User registration and login flow"
        - "Main feature workflow"
      tools: "Playwright, Cypress, etc."
      run_command: "npm run test:e2e"

    test_design_principles:
      patterns_to_use:
        - pattern: "Arrange-Act-Assert pattern"
          example: |
            // Arrange: Set up test data
            const user = createTestUser();
            // Act: Execute the function
            const result = authenticateUser(user);
            // Assert: Verify the outcome
            expect(result).toBeDefined();

        - pattern: "Factory pattern for test data"
          example: |
            const user = UserFactory.create({ role: 'admin' });

      anti_patterns_to_avoid:
        - pattern: "Testing implementation details"
          why: "Makes tests brittle and coupled to implementation"
          instead: "Test behavior and outcomes"

        - pattern: "Shared mutable state between tests"
          why: "Creates test interdependencies and flakiness"
          instead: "Use beforeEach to reset state"

      mocking_guidelines:
        mock_these:
          - "External API calls"
          - "Database connections"
          - "File system operations"
          - "Third-party services"

        dont_mock_these:
          - "Business logic functions"
          - "Internal utilities"
          - "Simple data transformations"

        project_mocking_pattern:
          reference: "tests/helpers/mocks.ts"
          example: |
            jest.mock('./services/api', () => ({
              fetchUser: jest.fn().mockResolvedValue(mockUser)
            }));

  # Commit strategy
  commit_strategy:
    total_commits: 5

    commits:
      - sequence: 1
        type: "test"
        description: "Add tests for feature X core functionality"
        files:
          - "tests/feature-x.test.ts"
        why_separate: "TDD - tests must come first"

      - sequence: 2
        type: "feat"
        description: "Implement feature X core functionality"
        files:
          - "src/features/feature-x.ts"
          - "src/types/feature-x.types.ts"
        why_separate: "Core implementation separate from extensions"

      - sequence: 3
        type: "test"
        description: "Add integration tests for feature X"
        files:
          - "tests/integration/feature-x.integration.test.ts"
        why_separate: "Integration tests separate from unit tests"

      - sequence: 4
        type: "feat"
        description: "Add feature X API endpoints"
        files:
          - "src/api/feature-x.routes.ts"
          - "src/api/feature-x.controller.ts"
        why_separate: "API layer separate from business logic"

      - sequence: 5
        type: "docs"
        description: "Add documentation for feature X"
        files:
          - "docs/features/feature-x.md"
          - "README.md"
        why_separate: "Documentation as final step after implementation"

    message_format:
      pattern: "type: brief description in present tense"
      examples:
        - "feat: add user authentication with JWT"
        - "fix: resolve race condition in async handler"
        - "test: add edge case coverage for validation"
        - "refactor: extract common logic to utility"

      example_from_history: |
        Based on recent git history:
        feat: implement OAuth2 flow for third-party login

    commit_guidelines:
      - "Keep commits atomic - one logical change per commit"
      - "Write clear, descriptive messages in imperative mood"
      - "Include tests in the same commit as implementation (after test-first commit)"
      - "Commit early and often - easier to squash than to split"

    worktree_commit_workflow:
      - "Each worktree maintains its own commit history on its branch"
      - "Commits in one worktree don't affect other worktrees"
      - "Merge worktree branches to main independently when ready"
      - "For tasks depending on multiple groups, merge prerequisites first"

  # Common pitfalls
  common_pitfalls:
    - pitfall: "Forgetting to validate user input"
      why: "Can lead to runtime errors and security vulnerabilities"
      how_to_avoid: "Always validate at API boundaries using schema validation (e.g., Zod, Joi)"
      reference:
        file: "src/api/validators/user.validator.ts"
        pattern: "Follow this validation pattern"

    - pitfall: "Not handling async errors properly"
      why: "Unhandled promise rejections can crash the application"
      how_to_avoid: "Use try-catch in async functions, add error boundaries"
      reference:
        file: "src/middleware/error-handler.ts"
        pattern: "Use centralized error handling"

    - pitfall: "Creating tightly coupled code"
      why: "Makes testing difficult and reduces maintainability"
      how_to_avoid: "Use dependency injection, program to interfaces"
      reference:
        file: "src/services/user.service.ts"
        pattern: "Note how dependencies are injected"

    - pitfall: "Ignoring existing patterns"
      why: "Creates inconsistency in the codebase"
      how_to_avoid: "Search for similar features and follow their patterns"
      reference:
        file: "src/features/authentication/"
        pattern: "Study this feature's structure"

    - pitfall: "Working in wrong worktree or mixing task groups"
      why: "Can break dependency ordering or create merge conflicts"
      how_to_avoid: "Always check worktree_group field before starting a task"
      reference:
        command: "git worktree list"
        pattern: "Verify you're in the correct worktree for the task"

    - pitfall: "Merging worktree branches out of order"
      why: "Tasks depending on other tasks may fail if prerequisites not merged"
      how_to_avoid: "Check depends_on field and ensure prerequisite tasks are merged to main first"
      reference:
        pattern: "For Task 5 depending on Tasks 2 and 3, merge both chain-1 and independent-3 branches first"

  # Resources and references
  resources:
    existing_code:
      - type: "Similar feature implementation"
        path: "src/features/similar-feature/"
        note: "Study the structure and patterns used here"

      - type: "Utility functions"
        path: "src/utils/"
        note: "Reusable helpers for validation, formatting, etc."

      - type: "Test examples"
        path: "tests/features/authentication.test.ts"
        note: "Well-structured test examples to follow"

      - type: "API patterns"
        path: "src/api/routes/user.routes.ts"
        note: "Standard API route structure"

    documentation:
      - type: "Framework documentation"
        link: "https://framework-docs-url.com"
        relevance: "Core framework concepts"

      - type: "Internal architecture docs"
        path: "docs/architecture/README.md"
        relevance: "Project-specific patterns and decisions"

      - type: "API specifications"
        path: "docs/api/openapi.yaml"
        relevance: "API contract definitions"

      - type: "Testing guide"
        path: "docs/testing-guide.md"
        relevance: "Testing standards and practices"

      - type: "Git worktree guide"
        link: "https://git-scm.com/docs/git-worktree"
        relevance: "Understanding worktree commands and workflows"

    external_resources:
      - title: "Relevant external guide or tutorial"
        url: "https://example.com/guide"
        why: "Helpful for understanding concept X"

    validation_checklist:
      - item: "All tests pass"
        command: "npm test"
        checked: false

      - item: "Linter passes"
        command: "npm run lint"
        checked: false

      - item: "Type checker passes"
        command: "npm run type-check"
        checked: false

      - item: "Code formatted correctly"
        command: "npm run format:check"
        checked: false

      - item: "No debug statements left"
        search_for: "console.log, debugger, print()"
        checked: false

      - item: "Error handling in place"
        verify: "All async operations have try-catch"
        checked: false

      - item: "Edge cases covered in tests"
        verify: "Test coverage includes null, undefined, empty, boundary values"
        checked: false

      - item: "Documentation updated"
        files: "README.md, relevant docs updated"
        checked: false

      - item: "Worktree branches merged in correct order"
        verify: "Dependencies resolved before merging dependent tasks"
        checked: false

      - item: "Worktrees cleaned up after merge"
        command: "git worktree remove <worktree-path>"
        checked: false

  # Enforcement mechanisms
  enforcement_mechanisms_reference: |
    See [Enforcement Guide](ai_docs/enforcement-guide.md) for comprehensive setup of:
    - Pre-commit hooks (Python pre-commit framework, TypeScript husky+lint-staged, Go git hooks)
    - CI/CD workflows (GitHub Actions for Python, TypeScript, Go)
    - Branch protection rules
    - Blocking vs warning check philosophy


## Phase 3: Plan Review & Validation

After generating the YAML plan:

1. **Validate YAML structure**:
   - Ensure valid YAML syntax (proper indentation, no tabs)
   - All required fields are present
   - Arrays and objects are properly formatted
   - Multiline strings use `|` or `>` appropriately

2. **Validate completeness**:
   - Every task has test-first approach defined
   - Every task has clear file paths (no placeholders)
   - **Every task has TASK-LEVEL success_criteria field (direct field, NOT nested under verification)**
   - **Every task has TASK-LEVEL test_commands field (direct field for conductor QC)**
   - **Every task has verification section with automated_tests and success_criteria (for human documentation)**
   - **Every task has code_quality section with full_quality_pipeline for the appropriate language (REQUIRED)**
   - **Integration tasks must have BOTH success_criteria AND integration_criteria fields**
   - Every task has a worktree_group assignment
   - Every task has an agent assigned (singular field, not array)
   - All assigned agents exist in the discovered agent list
   - Commit points are logical and frequent
   - All sections from the template are filled
   - Worktree groups correctly reflect dependency analysis

3. **Validate dependency graph and worktree groupings**:
   - All dependency chains correctly identified in worktree_groups
   - Independent tasks properly isolated in separate worktrees
   - Each task's worktree_group matches a defined group
   - Setup commands include correct task sequence for chains
   - Tasks with multiple dependencies wait for prerequisite merges

4. **Ensure beginner-friendliness**:
   - No assumed knowledge about the codebase
   - Test design is explicitly explained with examples
   - Patterns are referenced with specific file paths and line numbers
   - Common mistakes are called out with solutions
   - Worktree workflow is clearly explained with commands

5. **Verify DRY/YAGNI/TDD principles**:
   - Tests are written first for each task (separate test commit, then implementation)
   - No unnecessary abstractions suggested
   - Code reuse is explicitly called out with references
   - Frequent, atomic commits are mandated

6. **Check YAML field integrity and avoid conflicts**:
   - Proper use of lists vs. objects
   - Consistent indentation (2 spaces recommended)
   - String quoting where necessary
   - **No duplicate keys** - avoid using these reserved field names:
     - `status` - will be managed by conductor
     - `completed_at` / `completed_date` - will be managed by conductor
     - `git_commit` - will be managed by conductor (different from `commit` object)
   - Use distinct names: `commit` for planning, `git_commit` for tracking actual results
   - Comments are helpful but not excessive

## Output Format

1. **Create the docs/plans directory** if it doesn't exist
2. **Write the complete YAML plan** to `docs/plans/<feature-slug>.yaml`
3. **Validate the YAML** is syntactically correct
4. **Confirm to the user**:
   - Plan location: `docs/plans/<feature-slug>.yaml`
   - Number of tasks defined
   - Number of worktree groups identified
   - Estimated total time
   - Parallel execution opportunities (independent worktrees)
   - First task to start with
   - Command to view the plan (e.g., `cat docs/plans/<feature-slug>.yaml`)

## Important Guidelines

- **Be specific**: Use actual file paths from the codebase, not placeholders like `path/to/file`
- **Be concrete**: Provide code skeletons and examples in the YAML structure
- **Be practical**: Reference existing code extensively with specific file paths and line numbers
- **Be educational**: Explain the "why" behind decisions in description fields
- **Be thorough**: Assume zero codebase knowledge from the engineer
- **Be test-focused**: TDD is mandatory, explain test design clearly with examples
- **Be quality-focused**: Every task MUST include TASK-LEVEL success_criteria and test_commands fields (for conductor QC parsing), plus verification section with automated_tests (for human documentation) and code_quality (full_quality_pipeline) sections - these are not optional
- **Be commit-focused**: Make commit strategy explicit with clear sequence
- **Be agent-aware**: Assign the most appropriate agent from the discovered list to each task based on the work type. Every task MUST have an agent assigned (singular field, not array).
- **Be YAML-compliant**: Ensure proper syntax, indentation, and structure
- **Be machine-readable**: YAML format allows for automation and tooling integration
- **Be dependency-aware**: Accurately analyze task dependencies and create worktree groups
- **Be parallelism-optimized**: Identify opportunities for parallel execution via independent worktrees

## YAML Best Practices

1. **Indentation**: Use 2 spaces consistently (never tabs)
2. **Multiline strings**: Use `|` for literal blocks (preserves newlines) or `>` for folded blocks
3. **Quoting**: Quote strings containing special characters (`:`, `#`, `@`, etc.)
4. **Lists**: Use `-` for array items, consistent indentation
5. **Objects**: Use key-value pairs with proper nesting
6. **Comments**: Use `#` sparingly, prefer descriptive keys
7. **Anchors**: Avoid unless necessary for true reuse
8. **Validation**: Plan must be parseable by standard YAML parsers

## Worktree Strategy Benefits

The worktree grouping strategy provides:

1. **Parallelism**: Independent tasks can be developed simultaneously in separate worktrees
2. **Dependency ordering**: Tasks in dependency chains execute sequentially within their worktree
3. **Isolation**: Each worktree has its own working directory and branch, preventing conflicts
4. **Flexibility**: Engineers can switch between worktrees without stashing or committing incomplete work
5. **Visibility**: Clear documentation of which tasks can run in parallel vs. must run sequentially

**Example workflow:**
- Engineer A works on chain-1 worktree (tasks 1→2→4 sequentially)
- Engineer B simultaneously works on independent-3 worktree (task 3)
- Both merge to main independently when ready
- Engineer C then starts chain-5 worktree (task 5) after prerequisites are merged

The goal is that an engineer can follow this YAML plan mechanically (or even parse it programmatically) and produce high-quality, well-tested code that fits the existing codebase perfectly. The YAML format enables automation and tooling integration, while the worktree groups enable optimal parallel execution.
