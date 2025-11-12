---
allowed-tools: Read, Bash(ls*), Glob, Grep, Bash(git status*), Bash(git log*), Bash(fd*), Write
argument-hint: "feature description"
description: Generate comprehensive implementation plan in YAML format with detailed tasks, testing strategy, and commit points
---

# Doc YAML - Comprehensive Implementation Plan Generator (YAML Format)

Create a detailed, step-by-step implementation plan for $ARGUMENTS in **YAML format**. The plan should enable a skilled engineer with zero context about this codebase to successfully implement the feature.

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

## Phase 2: YAML Plan Generation

Create a comprehensive implementation plan in `docs/plans/<feature-slug>.yaml` where `<feature-slug>` is derived from the feature description (e.g., "user-authentication", "payment-integration").

## Phase 2A: Objective Plan Splitting (YAML-Specific)

When generating YAML task breakdowns, implement **metric-based plan splitting** to keep YAML files manageable while preserving complete task details and valid YAML structure.

### Line Count Tracking for YAML

Continuously track the number of lines written to the current YAML plan file as you generate task details. This is a precise, objective metric.

**Implementation:**
- Initialize `line_count = 0` when starting a new YAML plan file
- Increment `line_count` for each line written (including YAML structure lines)
- Track `tasks_detailed` (number of tasks fully written) vs `total_tasks` (total tasks identified)
- Account for YAML syntax overhead (indentation, list markers, object keys)

### Trigger Condition (Objective, Metric-Based)

Split the YAML plan into a new file when ALL of these conditions are met:

```
IF (
  line_count > 2000
  AND tasks_detailed < total_tasks
  AND next_worktree_group_exists
):
  THEN split_yaml_plan_to_new_file(next_group)
```

**Critical rules:**
- **2000 lines is the hard limit** - no subjective judgment
- **Never split mid-task** - complete the current task's YAML structure before splitting
- **Maintain valid YAML** - each split file must be independently parseable
- **Split at worktree group boundaries** - natural organizational boundaries
- **Only split if work remains** - don't create empty YAML files

### Split Strategy for YAML Files

When the trigger condition is met, create numbered YAML plan files organized by worktree group:

```
docs/plans/feature-name/
├── plan-01-chain-1.yaml        (2000 lines, tasks in chain-1 worktree group)
├── plan-02-chain-2.yaml        (1800 lines, tasks in chain-2 worktree group)
├── plan-03-independent.yaml    (1200 lines, independent tasks)
└── index.yaml                  (metadata and cross-references to all plan files)
```

**File naming convention:**
- `plan-NN-[worktree-group-id].yaml` where NN is the zero-padded sequence number (01, 02, 03, etc.)
- The worktree group ID identifies which tasks are in which file
- Example: `plan-01-chain-1.yaml`, `plan-02-independent-tasks.yaml`, `plan-03-chain-2.yaml`

### YAML-Specific Considerations

When splitting YAML plans, ensure each file:

1. **Is valid YAML independently** - complete document structure in each file
2. **Has proper root structure** - each file should have a `plan:` root object
3. **Contains complete task objects** - never split a task's YAML structure across files
4. **Includes necessary metadata** - each file has enough context to be understood
5. **Uses proper references** - if tasks reference other tasks, include notes about cross-file dependencies

**Example valid split file structure:**

```yaml
# docs/plans/feature-name/plan-01-chain-1.yaml
plan:
  metadata:
    feature_name: "User Authentication"
    file_sequence: 1
    total_files: 3
    worktree_group: "chain-1"
    tasks_in_file: [1, 2, 3, 4, 5]
    next_file: "plan-02-chain-2.yaml"

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

### Decision Logic (Pure Metrics-Based)

```
WHILE generating YAML task breakdowns:
  - Write task YAML to current plan file
  - Increment line_count for each line written
  - When complete task object is written, increment tasks_completed

  IF line_count > 2000 AND tasks_completed < total_tasks:
    - Check if current task is last in current worktree group
    - IF yes (natural boundary exists):
      - COMPLETE current YAML structure (close all objects/lists)
      - VALIDATE YAML is parseable
      - Create new YAML plan file for next worktree group
      - Reset line_count = 0 for new file
      - Start new file with proper YAML structure
      - CONTINUE with remaining tasks in new file
    - IF no (mid-group):
      - CONTINUE in current file until group boundary reached
      - THEN split at next group boundary

  IF tasks_completed == total_tasks:
    - COMPLETE current YAML file structure
    - VALIDATE final YAML is parseable
    - Create index.yaml if multiple files exist
    - BREAK while loop
```

**Key principle:** No subjective judgment - only objective line counts and task boundaries, with valid YAML guaranteed.

### Cross-Reference Index (index.yaml)

When YAML plan splitting occurs, create a `docs/plans/<feature-name>/index.yaml` index file:

```yaml
plan_index:
  metadata:
    feature_name: "User Authentication System"
    created: "2025-01-09"
    total_tasks: 25
    total_files: 3

  files:
    - sequence: 1
      filename: "plan-01-chain-1.yaml"
      worktree_group: "chain-1"
      tasks: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
      line_count: 2023
      description: "Core authentication flow implementation"

    - sequence: 2
      filename: "plan-02-chain-2.yaml"
      worktree_group: "chain-2"
      tasks: [11, 12, 13, 14, 15, 16, 17, 18]
      line_count: 1987
      description: "API integration layer and middleware"

    - sequence: 3
      filename: "plan-03-independent.yaml"
      worktree_group: "independent"
      tasks: [19, 20, 21, 22, 23, 24, 25]
      line_count: 1156
      description: "Documentation, tests, and independent cleanup tasks"

  getting_started:
    first_file: "plan-01-chain-1.yaml"
    first_task: 1
    instructions: |
      1. Start with plan-01-chain-1.yaml - Task 1
      2. Follow task dependencies as documented in each file
      3. Refer to worktree_group field for parallel execution guidance
      4. Tasks may reference tasks in other files - check depends_on field

  overview: |
    Implementation plan for User Authentication System.
    This feature adds JWT-based authentication with OAuth2 support,
    including registration, login, password reset, and session management.
```

### Output Confirmation

When YAML plan generation completes, report to the user:

**Single file output:**
```
YAML plan created: docs/plans/<feature-slug>.yaml
- Total tasks: 12
- Total lines: 1,543
- Worktree groups: 3 (chain-1, independent-1, independent-2)
- YAML validation: PASSED
```

**Split file output:**
```
YAML plan created in: docs/plans/<feature-name>/
- Total tasks: 25
- Plan files created: 3
  - plan-01-chain-1.yaml (2,023 lines, tasks 1-10)
  - plan-02-chain-2.yaml (1,987 lines, tasks 11-18)
  - plan-03-independent.yaml (1,156 lines, tasks 19-25)
- Index: docs/plans/<feature-name>/index.yaml
- Worktree groups: 5 (chain-1, chain-2, independent-1, independent-2, independent-3)
- YAML validation: ALL FILES PASSED

Start with: docs/plans/<feature-name>/plan-01-chain-1.yaml - Task 1
```

### YAML Validation Requirements

Before finalizing any YAML file (especially after splitting):

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
6. **Cross-file references**: index.yaml provides navigation between files
7. **Automation-friendly**: Tools can programmatically process split plans
8. **Maintainable**: Each YAML file remains readable and properly structured

The YAML plan MUST follow this exact structure:

```yaml
plan:
  # Metadata about the implementation plan
  metadata:
    feature_name: "Descriptive Feature Name"
    created: "YYYY-MM-DD"
    target: "Brief description of what we're building"
    estimated_tasks: <number>

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

  # Detailed task breakdown
  tasks:
    - task_number: 1
      name: "Descriptive Task Name"
      agent: "agent-name"  # REQUIRED: Reference to agent from ~/.claude/agents/
      worktree_group: "chain-1"
      files:
        - "path/to/file1.ext"
        - "path/to/file2.ext"
      depends_on: []  # Empty if no dependencies, otherwise [2, 3] for tasks 2 and 3
      estimated_time: "30m"  # Options: 5m, 15m, 30m, 1h, 2h

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
          // Provide actual test code structure following project patterns
          describe('Feature', () => {
            it('should handle specific behavior', () => {
              // Arrange
              const mock = createMock();

              // Act
              const result = functionUnderTest(mock);

              // Assert
              expect(result).toBe(expected);
            });
          });

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

        integration:
          imports:
            - "import { SomeClass } from './path/to/module'"
            - "import type { SomeType } from './types'"

          services_to_inject:
            - name: "DatabaseService"
              from: "services/database"
            - name: "LoggerService"
              from: "services/logger"

          config_values:
            - name: "API_TIMEOUT"
              source: "config/constants"

          error_handling:
            - "Throw ValidationError for invalid input"
            - "Catch and log DatabaseError, return null"

      verification:
        manual_testing:
          - step: "Start the development server"
            command: "npm run dev"
          - step: "Navigate to /feature-path"
            expected: "Should see feature working"
          - step: "Test edge case X"
            expected: "Should handle gracefully"

        automated_tests:
          command: "npm test path/to/test.ext"
          expected_output: |
            All tests passing:
            ✓ Feature test 1
            ✓ Feature test 2
            ✓ Edge case test

        success_criteria:
          - "All unit tests pass"
          - "No TypeScript errors"
          - "Linter passes"

      commit:
        # NOTE: This 'commit' object is for PLANNING the commit you intend to make.
        # After task completion, conductor will add a separate 'git_commit' field
        # containing the actual git hash. These serve different purposes:
        #   - commit: (object) Planning - what commit WILL be made
        #   - git_commit: (string) Tracking - what commit WAS made
        type: "feat"  # Options: feat, fix, test, refactor, docs, chore
        message: "add user authentication flow"
        body: |
          Optional commit body explaining why this change was made.
          Can include breaking changes, references to issues, etc.

        files:
          - "path/to/implementation.ext"
          - "path/to/test.ext"
          - "path/to/types.ext"

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
```

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
   - Every task has verification steps
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
