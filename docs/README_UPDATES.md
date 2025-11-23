# README.md Updates: Cross-File Dependencies (v2.5)

This document shows the updates needed for README.md to document cross-file dependency support.

## Section 1: Key Features Update

### CURRENT (Lines 59-100)

```markdown
- **Wave-Based Execution**: Tasks execute in parallel within waves, sequentially between waves
- **Dependency Management**: Automatic dependency graph calculation with cycle detection
- **Quality Control**: Automated review of task outputs (GREEN/RED/YELLOW verdicts)
- **Retry Logic**: Automatic retry on failures (up to 2 attempts per task)
- **Skip Completed Tasks**: Resume interrupted plans by skipping already-completed tasks
- **File Locking**: Safe concurrent updates to plan files
- **Agent Discovery**: Automatic detection of available Claude agents
- **Dual Format Support**: Both Markdown and YAML plan formats
- **Dry Run Mode**: Test execution without actually running tasks
- **Progress Logging**: Real-time console output and file-based logs
- **Adaptive Learning** (v2.0+): AI-powered learning system that improves over time
  - Tracks execution history and failure patterns
  - Automatically adapts agent selection after failures
  - Inter-retry agent swapping based on QC suggestions
  - Dual feedback storage (plain text + structured JSON)
  - Learns from past successes to optimize future runs
  - CLI commands for statistics and insights
- **Structured QC Responses** (v2.1+): Quality control returns JSON with verdicts, issues, and recommendations
  - Automatic parsing of Claude CLI JSON envelopes and markdown code fences
  - Agent suggestions for improved retry success rates
  - Enhanced context loading from plan files and database
- **Structured Success Criteria** (v2.3+): Per-criterion verification with multi-agent consensus
  - Define explicit success criteria per task
  - QC agents verify each criterion individually (PASS/FAIL)
  - Multi-agent unanimous consensus required (all agents must agree)
  - Backward compatible with legacy blob review
- **Intelligent QC Agent Selection** (v2.4+): Claude-based agent recommendations
  - Analyzes task context (name, files, description) + executing agent
  - Recommends domain specialists (security-auditor, database-optimizer)
  - Deterministic guardrails (max agents cap, code-reviewer baseline)
  - Caches results to minimize API calls during retries
  - **Critical fix**: Respects agent RED verdicts even when criteria pass
  - Domain-specific review criteria (Go/SQL/TypeScript/Python)
  - File path verification in QC prompts
- **Integration Tasks with Dual Criteria** (v2.5+): Component and cross-component validation
  - Define component tasks (`type: "component"`) vs integration tasks (`type: "integration"`)
  - Dual-level validation: `success_criteria` for component checks, `integration_criteria` for cross-component
  - QC system verifies both criteria types for comprehensive quality control
  - Automatic dependency context injection for tasks with dependencies
  - Better task organization and clearer separation of concerns
```

### UPDATED

```markdown
- **Wave-Based Execution**: Tasks execute in parallel within waves, sequentially between waves
- **Dependency Management**: Automatic dependency graph calculation with cycle detection
  - **Local dependencies** (same file): `depends_on: [1, 2]`
  - **Cross-file dependencies** (v2.5+): `file: foundation.yaml, task: 1`
  - Works with any file order - dependencies are explicit
- **Quality Control**: Automated review of task outputs (GREEN/RED/YELLOW verdicts)
- **Retry Logic**: Automatic retry on failures (up to 2 attempts per task)
- **Skip Completed Tasks**: Resume interrupted plans by skipping already-completed tasks
- **File Locking**: Safe concurrent updates to plan files
- **Agent Discovery**: Automatic detection of available Claude agents
- **Dual Format Support**: Both Markdown and YAML plan formats
- **Dry Run Mode**: Test execution without actually running tasks
- **Progress Logging**: Real-time console output and file-based logs
- **Adaptive Learning** (v2.0+): AI-powered learning system that improves over time
  - Tracks execution history and failure patterns
  - Automatically adapts agent selection after failures
  - Inter-retry agent swapping based on QC suggestions
  - Dual feedback storage (plain text + structured JSON)
  - Learns from past successes to optimize future runs
  - CLI commands for statistics and insights
- **Structured QC Responses** (v2.1+): Quality control returns JSON with verdicts, issues, and recommendations
  - Automatic parsing of Claude CLI JSON envelopes and markdown code fences
  - Agent suggestions for improved retry success rates
  - Enhanced context loading from plan files and database
- **Structured Success Criteria** (v2.3+): Per-criterion verification with multi-agent consensus
  - Define explicit success criteria per task
  - QC agents verify each criterion individually (PASS/FAIL)
  - Multi-agent unanimous consensus required (all agents must agree)
  - Backward compatible with legacy blob review
- **Intelligent QC Agent Selection** (v2.4+): Claude-based agent recommendations
  - Analyzes task context (name, files, description) + executing agent
  - Recommends domain specialists (security-auditor, database-optimizer)
  - Deterministic guardrails (max agents cap, code-reviewer baseline)
  - Caches results to minimize API calls during retries
  - **Critical fix**: Respects agent RED verdicts even when criteria pass
  - Domain-specific review criteria (Go/SQL/TypeScript/Python)
  - File path verification in QC prompts
- **Cross-File Dependencies** (v2.5+): Explicit dependency notation for multi-file plans
  - Tasks can reference other files: `file: foundation.yaml, task: 1`
  - Works in any file order - dependencies determine execution sequence
  - Clear error messages if files or tasks are missing
  - Resumable execution correctly tracks across files
  - Fully backward compatible with implicit file ordering
- **Integration Tasks with Dual Criteria** (v2.5+): Component and cross-component validation
  - Define component tasks (`type: "component"`) vs integration tasks (`type: "integration"`)
  - Dual-level validation: `success_criteria` for component checks, `integration_criteria` for cross-component
  - QC system verifies both criteria types for comprehensive quality control
  - Automatic dependency context injection for tasks with dependencies
  - Better task organization and clearer separation of concerns
```

---

## Section 2: Project Status Update

### CURRENT (Lines 820-875)

```markdown
**Current Status**: Production-ready v2.5.0

Conductor is feature-complete with:
- Complete implementation with 86%+ test coverage
- `conductor validate` and `conductor run` commands
- Wave-based parallel execution with dependency management
- Quality control reviews with automated retries
- Multi-file plan loading and merging
- Worktree group organization with isolation levels
- Auto-incrementing version management (VERSION file)
- File locking for concurrent updates
- Agent discovery system
- **Adaptive learning system** (v2.0)
  - SQLite-based execution history
  - Automatic agent adaptation
  - Pattern detection and analysis
  - Four CLI learning commands
- **Structured QC responses** (v2.1)
  - JSON parsing with nested envelope extraction
  - Markdown code fence stripping
  - Detailed issues and recommendations
- **Inter-retry agent swapping** (v2.1)
  - QC suggests alternative agents on failures
  - Automatic agent swap during retry loop
  - Configurable swap behavior
- **Dual feedback storage** (v2.1)
  - Plan file storage (human-readable, git-trackable)
  - Database storage (long-term learning)
  - No duplicate entries
- **Structured success criteria** (v2.3)
  - Per-criterion QC verification
  - Multi-agent unanimous consensus
  - Backward compatible with legacy review
- **Intelligent QC agent selection** (v2.4)
  - Claude-based agent recommendations
  - Task context + executing agent analysis
  - Deterministic guardrails and caching
  - Critical fix: Respects agent RED verdicts even when criteria pass
  - Domain-specific review criteria (Go/SQL/TypeScript/Python)
  - File path verification in QC prompts
- **Integration tasks with dual criteria** (v2.5)
  - Component vs integration task type distinction
  - Dual-level validation (success_criteria + integration_criteria)
  - Automatic dependency context injection
  - Comprehensive cross-component validation
  - Clearer task organization and separation of concerns
- Comprehensive documentation
```

### UPDATED

```markdown
**Current Status**: Production-ready v2.5.0

Conductor is feature-complete with:
- Complete implementation with 86%+ test coverage
- `conductor validate` and `conductor run` commands
- Wave-based parallel execution with dependency management
- Quality control reviews with automated retries
- Multi-file plan loading and merging
- **Cross-file dependencies** (v2.5)
  - Explicit dependency notation: `file: foundation.yaml, task: 1`
  - Local dependencies still supported: `depends_on: [1, 2]`
  - Works in any file order (order-independent)
  - Fully backward compatible with implicit ordering
  - Clear validation and error messages
  - Resumable execution across files
- Worktree group organization with isolation levels
- Auto-incrementing version management (VERSION file)
- File locking for concurrent updates
- Agent discovery system
- **Adaptive learning system** (v2.0)
  - SQLite-based execution history
  - Automatic agent adaptation
  - Pattern detection and analysis
  - Four CLI learning commands
- **Structured QC responses** (v2.1)
  - JSON parsing with nested envelope extraction
  - Markdown code fence stripping
  - Detailed issues and recommendations
- **Inter-retry agent swapping** (v2.1)
  - QC suggests alternative agents on failures
  - Automatic agent swap during retry loop
  - Configurable swap behavior
- **Dual feedback storage** (v2.1)
  - Plan file storage (human-readable, git-trackable)
  - Database storage (long-term learning)
  - No duplicate entries
- **Structured success criteria** (v2.3)
  - Per-criterion QC verification
  - Multi-agent unanimous consensus
  - Backward compatible with legacy review
- **Intelligent QC agent selection** (v2.4)
  - Claude-based agent recommendations
  - Task context + executing agent analysis
  - Deterministic guardrails and caching
  - Critical fix: Respects agent RED verdicts even when criteria pass
  - Domain-specific review criteria (Go/SQL/TypeScript/Python)
  - File path verification in QC prompts
- **Integration tasks with dual criteria** (v2.5)
  - Component vs integration task type distinction
  - Dual-level validation (success_criteria + integration_criteria)
  - Automatic dependency context injection
  - Comprehensive cross-component validation
  - Clearer task organization and separation of concerns
- Comprehensive documentation
```

---

## Section 3: Multi-File Plans Section Update

### CURRENT (Lines 794-814)

```markdown
## Multi-File Plans

Conductor supports splitting large implementation plans across multiple files with automatic merging and dependency management:

```bash
# Load and execute multiple plan files
conductor run setup.md features.md deployment.md

# Validate split plans before execution
conductor validate *.md
```

**Features:**
- Multi-file plan loading with auto-format detection
- Objective plan splitting by feature/component/service
- Cross-file dependency management
- Worktree groups for execution control
- File-to-task mapping for resume operations
- 100% backward compatible with single-file plans

See [Multi-File Plans Guide](docs/conductor.md#multi-file-plans--objective-splitting) for detailed documentation and examples.
```

### UPDATED

```markdown
## Multi-File Plans

Conductor supports splitting large implementation plans across multiple files with automatic merging and cross-file dependency management:

### Quick Start

```bash
# Load and execute multiple plan files
conductor run foundation.yaml services.yaml integration.yaml

# Validate split plans before execution
conductor validate *.yaml
```

### Key Features

- **Multi-file plan loading** with auto-format detection
- **Cross-file dependencies** (v2.5+): Explicit `file:` notation
  - Local: `depends_on: [1, 2]`
  - Cross-file: `file: foundation.yaml, task: 1`
- **Order-independent execution**: Works in any file order when using explicit dependencies
- **Objective plan splitting** by feature/component/service
- **Worktree groups** for execution control and organization
- **File-to-task mapping** for resumable operations
- **100% backward compatible** with single-file plans and implicit ordering

### Basic Example

**foundation.yaml:**
```yaml
plan:
  name: Foundation
  tasks:
    - id: 1
      name: Database Setup
      depends_on: []
```

**services.yaml:**
```yaml
plan:
  name: Services
  tasks:
    - id: 2
      name: Auth Service
      depends_on:
        - file: foundation.yaml
          task: 1  # Explicit dependency!
```

Execute together:
```bash
conductor run foundation.yaml services.yaml
# Or in reverse order - dependencies are explicit:
conductor run services.yaml foundation.yaml
```

### When to Use Cross-File Dependencies

**Use explicit cross-file dependencies when:**
- You have 3+ plan files
- Dependencies between files matter
- You want self-documenting plans
- You want robust resumable execution

**Simple implicit ordering works when:**
- Only 2 files
- Clear natural ordering
- Less critical documentation
- Prefer simplicity

### Documentation

- **[Cross-File Dependencies Reference](docs/CROSS_FILE_DEPENDENCIES.md)** - Complete guide with syntax, examples, and best practices
- **[Migration Guide](docs/MIGRATION_CROSS_FILE_DEPS.md)** - Convert existing plans to explicit dependencies
- **[Multi-File Plans Guide](docs/conductor.md#multi-file-plans--objective-splitting)** - Detailed documentation and architecture

[⬆ back to top](#table-of-contents)
```

---

## Section 4: Dependency Management Documentation Update

### ADD NEW SECTION after "Command-Line Flags"

```markdown
### Cross-File Dependency Syntax (v2.5+)

Conductor supports explicit cross-file dependencies for multi-file plans.

**Local Dependencies** (same file):
```yaml
tasks:
  - id: 2
    depends_on: [1]  # Reference by task number
```

**Cross-File Dependencies** (different files):
```yaml
tasks:
  - id: 2
    depends_on:
      - file: foundation.yaml
        task: 1  # Reference file and task
```

**Mixed Dependencies:**
```yaml
tasks:
  - id: 5
    depends_on:
      - 1                    # Local
      - 2                    # Local
      - file: foundation.yaml
        task: 1              # Cross-file
```

**How It Works:**
1. Conductor loads all plan files
2. Validates all cross-file references exist
3. Merges into unified task graph
4. Calculates execution waves respecting all dependencies
5. Executes tasks in correct order

**Benefits:**
- **Self-documenting**: Dependencies are explicit
- **Order-independent**: Works in any file order
- **Clear errors**: Shows which file/task is missing
- **Resumable**: Correctly tracks across files
- **Backward compatible**: Old plans still work

**Example:**
```bash
# These two commands produce the same result with explicit cross-file deps:
conductor run foundation.yaml services.yaml integration.yaml
conductor run integration.yaml foundation.yaml services.yaml
```

See [Cross-File Dependencies Reference](docs/CROSS_FILE_DEPENDENCIES.md) for detailed syntax, patterns, and troubleshooting.
```

---

## Section 5: Table of Contents Update

### CURRENT (Lines 7-40)

```markdown
## Table of Contents

- [What It Does](#what-it-does)
- [Key Features](#key-features)
- [Quick Start](#quick-start)
  - [Prerequisites](#prerequisites)
  - [Installation](#installation)
  - [Your First Execution](#your-first-execution)
- [Basic Usage](#basic-usage)
  - [Validate a Plan](#validate-a-plan)
  - [Run a Plan](#run-a-plan)
  - [Resume Interrupted Plans](#resume-interrupted-plans)
  - [Command-Line Flags](#command-line-flags)
- [Configuration](#configuration)
  - [Config File Location](#config-file-location)
  - [Setup](#setup)
  - [Configuration Priority](#configuration-priority)
  - [Example Configuration](#example-configuration)
  - [Build-Time Configuration](#build-time-configuration)
- [Adaptive Learning System](#adaptive-learning-system-v20)
- [Plan Format](#plan-format)
  - [Markdown Format](#markdown-format)
  - [YAML Format](#yaml-format)
- [Conductor Plugin](#conductor-plugin-included)
- [Documentation](#documentation)
- [Architecture Overview](#architecture-overview)
- [Development](#development)
- [Multi-File Plans](#multi-file-plans)
- [Project Status](#project-status)
- [Dependencies](#dependencies)
- [Contributing](#contributing)
- [License](#license)
- [Support](#support)
- [Acknowledgments](#acknowledgments)
```

### UPDATED

```markdown
## Table of Contents

- [What It Does](#what-it-does)
- [Key Features](#key-features)
- [Quick Start](#quick-start)
  - [Prerequisites](#prerequisites)
  - [Installation](#installation)
  - [Your First Execution](#your-first-execution)
- [Basic Usage](#basic-usage)
  - [Validate a Plan](#validate-a-plan)
  - [Run a Plan](#run-a-plan)
  - [Resume Interrupted Plans](#resume-interrupted-plans)
  - [Command-Line Flags](#command-line-flags)
  - [Cross-File Dependency Syntax](#cross-file-dependency-syntax-v25)
- [Configuration](#configuration)
  - [Config File Location](#config-file-location)
  - [Setup](#setup)
  - [Configuration Priority](#configuration-priority)
  - [Example Configuration](#example-configuration)
  - [Build-Time Configuration](#build-time-configuration)
- [Adaptive Learning System](#adaptive-learning-system-v20)
- [Plan Format](#plan-format)
  - [Markdown Format](#markdown-format)
  - [YAML Format](#yaml-format)
- [Conductor Plugin](#conductor-plugin-included)
- [Documentation](#documentation)
- [Architecture Overview](#architecture-overview)
- [Development](#development)
- [Multi-File Plans](#multi-file-plans)
- [Project Status](#project-status)
- [Dependencies](#dependencies)
- [Contributing](#contributing)
- [License](#license)
- [Support](#support)
- [Acknowledgments](#acknowledgments)
```

---

## Section 6: Plan Format Examples

### ADD new YAML example for cross-file dependencies

Insert after line 558 (after YAML Format with Success Criteria example):

```markdown
### YAML Format with Cross-File Dependencies (v2.5+)

```yaml
# foundation.yaml
plan:
  name: Foundation
  tasks:
    - id: 1
      name: Initialize Database
      files: [infrastructure/db.tf]
      depends_on: []
      estimated_time: 10 minutes

# services.yaml
plan:
  name: Services
  tasks:
    - id: 2
      name: Auth Service
      files: [internal/auth/auth.go]
      # Cross-file dependency: wait for database setup from foundation.yaml
      depends_on:
        - file: foundation.yaml
          task: 1
      estimated_time: 15 minutes

    - id: 3
      name: API Service
      files: [internal/api/api.go]
      # Mix of local and cross-file dependencies
      depends_on:
        - 2                      # Local task (same file)
        - file: foundation.yaml
          task: 1                # Cross-file task
      estimated_time: 20 minutes
```

**Execute together:**
```bash
conductor run foundation.yaml services.yaml
# Works in any order:
conductor run services.yaml foundation.yaml
```

**Validation:**
```bash
conductor validate foundation.yaml services.yaml
# Shows all cross-file references and execution plan
```
```

---

## Section 7: Add Documentation Reference

### APPEND to Documentation section (after line 684)

```markdown
- **[Cross-File Dependencies Reference](docs/CROSS_FILE_DEPENDENCIES.md)** - Complete syntax guide, examples, and patterns for cross-file dependencies (v2.5+)
- **[Migration Guide](docs/MIGRATION_CROSS_FILE_DEPS.md)** - Step-by-step guide to migrating existing plans to explicit cross-file dependencies
```

---

## Section 8: Architecture Overview Enhancement

### UPDATE Multi-File Plans section in Architecture (Lines 704-713)

**Before:**
```markdown
**Multi-File Plans:**
```
Multiple Plan Files (.md/.yaml)
  → Multi-File Loader (auto-detects format per file)
  → Plan Merger (validates, deduplicates, merges)
  → Dependency Graph (cross-file dependencies)
  → Wave Calculator (respects worktree groups)
  → [Rest of pipeline as above]
  → Plan Updater (file-aware task tracking)
```
```

**After:**
```markdown
**Multi-File Plans (with explicit cross-file dependencies, v2.5+):**
```
Multiple Plan Files (.md/.yaml)
  → Multi-File Loader (auto-detects format per file)
  → Cross-File Reference Validator (checks file/task existence)
  → Plan Merger (resolves cross-file refs, deduplicates, merges)
  → Unified Dependency Graph (all tasks with resolved dependencies)
  → Wave Calculator (respects worktree groups, processes all files)
  → [Rest of pipeline as above]
  → Plan Updater (file-aware task tracking, updates correct file)
```

**Key Components:**

- **Parser**: Auto-detects and parses Markdown/YAML plan files
- **Multi-File Loader**: Loads and merges multiple plans with cross-file validation
- **Cross-File Reference Validator** (v2.5+): Verifies file and task existence before processing
- **Graph Builder**: Calculates dependencies using Kahn's algorithm, resolves cross-file references
- **Orchestrator**: Coordinates wave-based execution with bounded concurrency
- **Task Executor**: Spawns Claude CLI agents with timeout and retry logic
- **Quality Control**: Reviews task outputs using dedicated QC agent
- **Plan Updater**: Thread-safe updates to correct plan files with file locking
- **Worktree Groups**: Organize tasks into execution groups with isolation levels
```

---

## Section 9: Quick Start Enhancement

### UPDATE "Your First Execution" section (after line 172)

Add a note about cross-file plans:

```markdown
### Multi-File Plans

To split plans across multiple files:

```bash
# Create multiple files with explicit cross-file dependencies
# foundation.yaml - infrastructure tasks
# services.yaml - service implementations

# Execute all files together
conductor run foundation.yaml services.yaml

# Validate before execution
conductor validate foundation.yaml services.yaml
```

For detailed examples and migration guide, see [Cross-File Dependencies Reference](docs/CROSS_FILE_DEPENDENCIES.md).
```

---

## Summary of Changes

| Section | Change | Type |
|---------|--------|------|
| Key Features | Added cross-file dependencies feature | Enhancement |
| Project Status | Added v2.5 cross-file capabilities | Update |
| Multi-File Plans | Expanded with explicit dependency examples | Enhancement |
| Basic Usage | Added cross-file syntax section | Addition |
| Plan Format | Added cross-file YAML example | Addition |
| Documentation | Added cross-file reference links | Addition |
| Architecture | Enhanced multi-file flow diagram | Enhancement |
| Table of Contents | Added cross-file dependency section | Addition |

All changes are **backward compatible** - existing documentation remains valid.

---

## Implementation Priority

1. Update Project Status (v2.5.0)
2. Update Key Features (add cross-file dependencies)
3. Enhance Multi-File Plans section
4. Add cross-file dependency syntax section
5. Add YAML format example with cross-file dependencies
6. Update architecture overview
7. Add documentation references
8. Update table of contents

---

## Related Documentation Files

These updates reference new documentation files:
- `docs/CROSS_FILE_DEPENDENCIES.md` - Complete reference
- `docs/MIGRATION_CROSS_FILE_DEPS.md` - Migration guide
