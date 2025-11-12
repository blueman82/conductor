# Implementation Plan: Adaptive Learning System for Conductor v2.0.0 (Part 2)

**Created**: 2025-01-12
**Target**: Configuration, session management, integration testing, documentation, and release preparation for adaptive learning system
**Estimated Tasks**: 13 (Tasks 13-25 of 25 total)
**Previous file**: plan-01-foundation.md

## Context for the Engineer

This is Part 2 of the adaptive learning system implementation. You should have completed Tasks 1-12 from plan-01-foundation.md before starting these tasks.

**Current Focus**:
- Configuration schema updates
- Session ID generation and run number tracking
- Repository maintenance (gitignore, dependencies)
- Integration testing suite
- Documentation
- Final validation and release preparation

**You are expected to**:
- Write tests BEFORE implementation (TDD red-green-refactor)
- Commit after each completed task with conventional commit format
- Follow existing code patterns in internal/ packages
- Keep changes minimal and focused (YAGNI)
- Maintain 86%+ test coverage for all new code

## Worktree Groups (Continued)

**Group independent-13-config**: Task 13
- **Tasks**: Task 13 only
- **Branch**: `feature/adaptive-learning-v2/independent-13-config`
- **Execution**: Parallel-safe (no dependencies)
- **Isolation**: Separate worktree
- **Rationale**: Config changes are independent and can be developed in parallel with other chains

**Group chain-4-session-management**: Tasks 14→15
- **Tasks**: Task 14, Task 15 (executed sequentially)
- **Branch**: `feature/adaptive-learning-v2/chain-4-session`
- **Execution**: Sequential
- **Isolation**: Separate worktree
- **Rationale**: Session tracking depends on session ID generation logic

**Group independent-16-gitignore**: Task 16
- **Tasks**: Task 16 only
- **Branch**: `feature/adaptive-learning-v2/independent-16-gitignore`
- **Execution**: Parallel-safe
- **Isolation**: Separate worktree
- **Rationale**: Gitignore changes are independent documentation updates

**Group independent-17-dependencies**: Task 17
- **Tasks**: Task 17 only
- **Branch**: `feature/adaptive-learning-v2/independent-17-deps`
- **Execution**: Parallel-safe
- **Isolation**: Separate worktree
- **Rationale**: Dependency addition is independent and should be done early

**Group chain-5-integration-tests**: Tasks 18→19→20
- **Tasks**: Task 18, Task 19, Task 20 (executed sequentially)
- **Branch**: `feature/adaptive-learning-v2/chain-5-integration`
- **Execution**: Sequential
- **Isolation**: Separate worktree
- **Rationale**: Integration tests must be built sequentially and require all prior chains to be merged

**Group independent-21-docs**: Task 21
- **Tasks**: Task 21 only
- **Branch**: `feature/adaptive-learning-v2/independent-21-docs`
- **Execution**: Parallel-safe
- **Isolation**: Separate worktree
- **Rationale**: Documentation can be written independently as feature is being developed

**Group chain-6-final-integration**: Tasks 22→23→24→25
- **Tasks**: Task 22, Task 23, Task 24, Task 25 (executed sequentially)
- **Branch**: `feature/adaptive-learning-v2/chain-6-final`
- **Execution**: Sequential
- **Isolation**: Separate worktree
- **Rationale**: Final tasks must execute sequentially - end-to-end testing, validation, version updates, and release preparation

## Task 13: Update configuration schema with learning settings

**Agent**: golang-pro
**File(s)**: internal/config/config.go, internal/config/config_test.go, .conductor/config.yaml.example
**Depends on**: None
**WorktreeGroup**: independent-13-config
**Estimated time**: 45m

Add learning configuration fields to the config schema including enable/disable flags, database path, behavior settings, and database maintenance options.

### Test-First Approach

**Test file**: internal/config/config_test.go

**Test structure**:
- TestConfig_LearningDefaults
- TestConfig_LearningDisabled
- TestConfig_LearningCustomPath

**Fixtures**:
- Sample config YAML with learning section

**Assertions**:
- Learning enabled by default
- Default values correct
- Custom values override defaults

**Edge cases**:
- Learning section missing (use defaults)
- Invalid path specified

**Example test skeleton**:
```go
func TestConfig_LearningDefaults(t *testing.T) {
    config := &Config{}
    setDefaults(config)

    assert.True(t, config.Learning.Enabled)
    assert.Equal(t, ".conductor/learning", config.Learning.DBPath)
    assert.Equal(t, 2, config.Learning.MinFailuresBeforeAdapt)
}
```

### Implementation

**Approach**:
1. Add LearningConfig struct to config.go
2. Include in main Config struct
3. Set defaults in setDefaults function
4. Update config merging logic
5. Document all fields

**Code structure**:
```go
type LearningConfig struct {
    Enabled                bool   `yaml:"enabled"`
    DBPath                 string `yaml:"db_path"`
    AutoAdaptAgent         bool   `yaml:"auto_adapt_agent"`
    EnhancePrompts         bool   `yaml:"enhance_prompts"`
    MinFailuresBeforeAdapt int    `yaml:"min_failures_before_adapt"`
    KeepExecutionsDays     int    `yaml:"keep_executions_days"`
    MaxExecutionsPerTask   int    `yaml:"max_executions_per_task"`
}

type Config struct {
    // existing fields
    Learning LearningConfig `yaml:"learning"`
}
```

**Key points**:
- Default values: enabled: true, db_path: .conductor/learning, min_failures: 2
- Backward compatibility: Missing learning section uses defaults

**Integration points**:
- Error handling: Validate paths are not empty, validate numeric values are positive

### Verification

**Manual testing**:
1. Run config tests: `go test ./internal/config/ -v` (expected: All config tests pass)

**Automated tests**:
```bash
go test ./internal/config/ -v
```

**Success criteria**:
- Config loads with learning section
- Defaults work correctly

### Commit

```
feat: add learning configuration schema

Add learning settings to config:
- LearningConfig struct with all settings
- Default values (enabled: true)
- Config merging support
- Full test coverage
- Update example config
```

**Files to commit**:
- internal/config/config.go
- internal/config/config_test.go
- .conductor/config.yaml.example

## Task 14: Implement session ID generation with timestamp format

**Agent**: golang-pro
**File(s)**: internal/executor/session.go, internal/executor/session_test.go
**Depends on**: None
**WorktreeGroup**: chain-4-session-management
**Estimated time**: 30m

Create session management utilities for generating unique session IDs in the format session-YYYYMMDD-HHMM and handling session-related logic.

### Test-First Approach

**Test file**: internal/executor/session_test.go

**Test structure**:
- TestGenerateSessionID_Format
- TestGenerateSessionID_Unique

**Assertions**:
- Session ID matches format
- IDs are unique across calls

**Example test skeleton**:
```go
func TestGenerateSessionID_Format(t *testing.T) {
    sessionID := generateSessionID()

    assert.Regexp(t, `^session-\d{8}-\d{4}$`, sessionID)
}
```

### Implementation

**Approach**:
1. Create session.go in executor package
2. Implement generateSessionID using time.Now()
3. Format as session-YYYYMMDD-HHMM
4. Export for use by orchestrator

**Code structure**:
```go
func generateSessionID() string {
    now := time.Now()
    return fmt.Sprintf("session-%s", now.Format("20060102-1504"))
}
```

**Key points**:
- Time format: Use Go time format: 20060102-1504

**Integration points**:
- Imports: `time`, `fmt`

### Verification

**Manual testing**:
1. Run session tests: `go test ./internal/executor/session_test.go -v` (expected: Format tests pass)

**Automated tests**:
```bash
go test ./internal/executor/ -run Session -v
```

**Success criteria**:
- Session IDs generated correctly
- Format is consistent

### Commit

```
feat: add session ID generation and management

Implement session management:
- generateSessionID with timestamp format
- Format: session-YYYYMMDD-HHMM
- Test coverage for format validation
```

**Files to commit**:
- internal/executor/session.go
- internal/executor/session_test.go

## Task 15: Implement run number tracking per plan file

**Agent**: golang-pro
**File(s)**: internal/learning/store.go, internal/learning/store_test.go
**Depends on**: Task 2, Task 14
**WorktreeGroup**: chain-4-session-management
**Estimated time**: 30m

Add GetRunCount method to learning store to track how many times a plan file has been executed, enabling run number calculation.

### Test-First Approach

**Test file**: internal/learning/store_test.go

**Test structure**:
- TestGetRunCount_NoExecutions
- TestGetRunCount_MultipleRuns

**Assertions**:
- Returns 0 for new plans
- Returns correct count for executed plans

**Edge cases**:
- Plan never executed
- Multiple executions same run

**Example test skeleton**:
```go
func TestGetRunCount_MultipleRuns(t *testing.T) {
    store := setupTestStore(t)

    // Simulate 3 runs
    recordExecutionsForRuns(store, "plan.md", 3)

    count := store.GetRunCount("plan.md")
    assert.Equal(t, 3, count)
}
```

### Implementation

**Approach**:
1. Add GetRunCount method to Store
2. Query MAX(run_number) for plan file
3. Return 0 if no executions found
4. Use in orchestrator for run number calculation

**Code structure**:
```go
func (s *Store) GetRunCount(planFile string) int {
    var maxRun int
    query := `SELECT COALESCE(MAX(run_number), 0) FROM task_executions WHERE plan_file = ?`
    s.db.QueryRow(query, planFile).Scan(&maxRun)
    return maxRun
}
```

**Key points**:
- COALESCE for NULL handling: Returns 0 if no rows found

**Integration points**:
- Error handling: Return 0 on error

### Verification

**Manual testing**:
1. Run store tests: `go test ./internal/learning/ -run RunCount -v` (expected: Run counting works)

**Automated tests**:
```bash
go test ./internal/learning/ -v
```

**Success criteria**:
- Correct run counts returned
- Handles missing data gracefully

### Commit

```
feat: add run number tracking per plan file

Implement run count tracking:
- GetRunCount method in store
- Query MAX(run_number) per plan
- Handle NULL with COALESCE
- Test coverage
```

**Files to commit**:
- internal/learning/store.go
- internal/learning/store_test.go

## Task 16: Update .gitignore to exclude learning database

**Agent**: fullstack-developer
**File(s)**: .gitignore
**Depends on**: None
**WorktreeGroup**: independent-16-gitignore
**Estimated time**: 5m

Add .conductor/learning/ directory to .gitignore to prevent accidentally committing project-local learning databases to version control.

### Test-First Approach

N/A - simple file edit

### Implementation

**Approach**:
1. Open .gitignore
2. Add .conductor/learning/ entry
3. Add comment explaining purpose

**Code structure**:
```
# Conductor learning database (project-local)
.conductor/learning/
```

**Key points**:
- Directory path: Use trailing slash to ignore directory

### Verification

**Manual testing**:
1. Check gitignore: `git check-ignore .conductor/learning/test.db` (expected: Path is ignored)

**Success criteria**:
- Learning directory ignored
- Pattern works correctly

### Commit

```
chore: add learning directory to gitignore

Exclude learning database from git:
- Add .conductor/learning/ to .gitignore
- Prevent accidental commits of learning data
```

**Files to commit**:
- .gitignore

## Task 17: Add mattn/go-sqlite3 dependency to go.mod

**Agent**: golang-pro
**File(s)**: go.mod, go.sum
**Depends on**: None
**WorktreeGroup**: independent-17-dependencies
**Estimated time**: 5m

Add github.com/mattn/go-sqlite3 to project dependencies for SQLite database support.

### Test-First Approach

N/A - dependency management

### Implementation

**Approach**:
1. Run go get github.com/mattn/go-sqlite3
2. Verify go.mod and go.sum updated
3. Ensure CGO is available for build

**Code structure**:
```bash
# Command to run:
go get github.com/mattn/go-sqlite3
```

**Key points**:
- CGO requirement: mattn/go-sqlite3 requires CGO compiler

### Verification

**Manual testing**:
1. Add dependency: `go get github.com/mattn/go-sqlite3` (expected: Dependency added)
2. Build project: `go build ./cmd/conductor` (expected: Builds successfully)

**Automated tests**:
```bash
go mod verify
```

**Success criteria**:
- Dependency added to go.mod
- Project builds successfully

### Commit

```
chore: add mattn/go-sqlite3 dependency

Add SQLite database support:
- Add github.com/mattn/go-sqlite3
- Update go.mod and go.sum
- Required for learning system
```

**Files to commit**:
- go.mod
- go.sum

## Task 18: Create integration tests for full learning pipeline

**Agent**: test-automator
**File(s)**: internal/executor/learning_integration_test.go
**Depends on**: Task 15
**WorktreeGroup**: chain-5-integration-tests
**Estimated time**: 2h

Build comprehensive integration tests that validate the complete learning pipeline from task execution through recording to analysis and adaptation.

### Test-First Approach

**Test file**: internal/executor/learning_integration_test.go

**Test structure**:
- TestLearningPipeline_SingleTask
- TestLearningPipeline_FailureAdaptation
- TestLearningPipeline_PatternDetection
- TestLearningPipeline_RunNumberTracking

**Fixtures**:
- Test plan files
- In-memory database

**Assertions**:
- Full pipeline executes end-to-end
- Failures trigger adaptation
- Patterns detected and stored
- Run numbers increment correctly

**Edge cases**:
- Learning store unavailable
- First run vs subsequent runs

**Example test skeleton**:
```go
func TestLearningPipeline_FailureAdaptation(t *testing.T) {
    // Setup: Create plan, learning store
    store := learning.NewStore(":memory:")
    plan := createTestPlan()

    // Simulate 2 failures with backend-developer
    recordFailures(store, "plan.md", "Task 1", "backend-developer", 2)

    // Execute with learning
    orchestrator := NewOrchestrator(plan, Config{
        LearningStore: store,
    })

    result := orchestrator.Execute(context.Background())

    // Verify agent was switched
    assert.NotEqual(t, "backend-developer", result.AgentUsed)
}
```

### Implementation

**Approach**:
1. Create integration test file
2. Setup in-memory SQLite database
3. Test complete execution flow with learning
4. Verify all hooks execute correctly
5. Test cross-run learning scenarios
6. Validate database state after execution

**Code structure**:
```go
// Full pipeline test structure
func TestLearningPipeline_XXX(t *testing.T) {
    // 1. Setup
    store := setupTestStore()
    plan := createTestPlan()

    // 2. Execute
    orch := NewOrchestrator(plan, Config{...})
    result := orch.Execute(ctx)

    // 3. Verify
    assertLearningBehavior(t, store, result)

    // 4. Cleanup
    store.Close()
}
```

**Key points**:
- In-memory database: Use :memory: for fast isolated tests
- End-to-end validation: Test full flow not just units

**Integration points**:
- Error handling: Each test cleans up resources

### Verification

**Manual testing**:
1. Run integration tests: `go test ./internal/executor/ -run Integration -v` (expected: All integration tests pass)

**Automated tests**:
```bash
go test ./internal/executor/learning_integration_test.go -v
```

**Success criteria**:
- Full pipeline tested
- All scenarios covered
- Tests reliable and fast

### Commit

```
test: add integration tests for learning pipeline

Comprehensive integration tests:
- Full pipeline execution
- Failure adaptation scenarios
- Pattern detection validation
- Run number tracking
- Cross-run learning tests
```

**Files to commit**:
- internal/executor/learning_integration_test.go

## Task 19: Create cross-run learning integration tests

**Agent**: test-automator
**File(s)**: internal/executor/learning_integration_test.go
**Depends on**: Task 18
**WorktreeGroup**: chain-5-integration-tests
**Estimated time**: 1h

Add tests that specifically validate learning behavior across multiple conductor runs, ensuring persistence and adaptation work correctly.

### Test-First Approach

**Test file**: internal/executor/learning_integration_test.go

**Test structure**:
- TestCrossRun_LearningPersists
- TestCrossRun_AgentAdaptation
- TestCrossRun_RunNumberIncrement

**Assertions**:
- Data persists between runs
- Second run adapts based on first
- Run numbers increment correctly

**Edge cases**:
- Database deleted between runs
- Different session IDs

**Example test skeleton**:
```go
func TestCrossRun_AgentAdaptation(t *testing.T) {
    dbPath := filepath.Join(t.TempDir(), "test.db")

    // Run 1: Fail twice with agent A
    run1(dbPath, "agent-a", "RED")
    run1(dbPath, "agent-a", "RED")

    // Run 2: Should adapt to agent B
    result := run2(dbPath)

    assert.NotEqual(t, "agent-a", result.AgentUsed)
}
```

### Implementation

**Approach**:
1. Add cross-run test scenarios
2. Use persistent temp database
3. Simulate multiple conductor invocations
4. Verify learning persists and adapts
5. Test session and run number tracking

**Code structure**:
```go
func TestCrossRun_XXX(t *testing.T) {
    // Use temp file database (not :memory:)
    dbPath := filepath.Join(t.TempDir(), "test.db")

    // Run 1
    executeRun(dbPath, 1, config1)

    // Run 2 (new orchestrator, same database)
    executeRun(dbPath, 2, config2)

    // Verify cross-run behavior
    verifyLearning(dbPath)
}
```

**Key points**:
- Persistent database: Use temp file not :memory: for cross-run tests
- Session isolation: Each run gets unique session ID

### Verification

**Manual testing**:
1. Run cross-run tests: `go test ./internal/executor/ -run CrossRun -v` (expected: All cross-run tests pass)

**Automated tests**:
```bash
go test ./internal/executor/learning_integration_test.go -v
```

**Success criteria**:
- Cross-run learning verified
- Persistence works correctly

### Commit

```
test: add cross-run learning integration tests

Test learning across multiple runs:
- Data persistence validation
- Agent adaptation across runs
- Run number incrementing
- Session management
```

**Files to commit**:
- internal/executor/learning_integration_test.go

## Task 20: Create CLI command integration tests

**Agent**: test-automator
**File(s)**: internal/cmd/learning_integration_test.go
**Depends on**: Task 19
**WorktreeGroup**: chain-5-integration-tests
**Estimated time**: 1h

Build integration tests for all CLI learning commands to ensure they work correctly with the learning database end-to-end.

### Test-First Approach

**Test file**: internal/cmd/learning_integration_test.go

**Test structure**:
- TestCLI_StatsCommand
- TestCLI_ShowCommand
- TestCLI_ClearCommand
- TestCLI_ExportCommand

**Fixtures**:
- Test database with sample data

**Assertions**:
- Each command produces expected output
- Commands handle errors gracefully
- Output formatting correct

**Edge cases**:
- Empty database
- Missing plan file

**Example test skeleton**:
```go
func TestCLI_StatsCommand(t *testing.T) {
    // Setup database with data
    setupTestDatabase()

    // Execute command
    cmd := newStatsCommand()
    output := executeCommand(cmd, []string{"plan.md"})

    // Verify output
    assert.Contains(t, output, "Total executions:")
    assert.Contains(t, output, "Success rate:")
}
```

### Implementation

**Approach**:
1. Create CLI integration test file
2. Setup test database with sample data
3. Test each command end-to-end
4. Capture and verify output
5. Test error scenarios

**Code structure**:
```go
func TestCLI_XXXCommand(t *testing.T) {
    // Setup
    dbPath := setupTestDB(t)
    seedTestData(dbPath)

    // Execute
    cmd := newXXXCommand()
    output := captureOutput(func() {
        cmd.Execute()
    })

    // Verify
    assertOutputCorrect(t, output)
}
```

**Key points**:
- Output capture: Redirect stdout to buffer for verification
- Test data seeding: Create realistic test scenarios

### Verification

**Manual testing**:
1. Run CLI integration tests: `go test ./internal/cmd/learning_integration_test.go -v` (expected: All CLI tests pass)

**Automated tests**:
```bash
go test ./internal/cmd/ -run Integration -v
```

**Success criteria**:
- All commands tested end-to-end
- Output validation complete

### Commit

```
test: add CLI command integration tests

Test all learning CLI commands:
- Stats command output verification
- Show command functionality
- Clear command with confirmation
- Export command (JSON/CSV)
```

**Files to commit**:
- internal/cmd/learning_integration_test.go

## Task 21: Create comprehensive learning system documentation

**Agent**: technical-documentation-specialist
**File(s)**: docs/learning.md, README.md
**Depends on**: None
**WorktreeGroup**: independent-21-docs
**Estimated time**: 2h

Write comprehensive documentation for the learning system including architecture, usage examples, configuration options, CLI commands, and troubleshooting guide.

### Test-First Approach

N/A - documentation

### Implementation

**Approach**:
1. Create docs/learning.md with full documentation
2. Update README.md with learning features section
3. Include architecture diagrams (text-based)
4. Add usage examples for all CLI commands
5. Document configuration options
6. Add troubleshooting section
7. Include FAQ

**Documentation structure**:
```markdown
# Adaptive Learning System

## Overview
- What is it
- How it works
- Benefits

## Architecture
- Database schema
- Hook integration points
- Failure analysis algorithm
- Agent suggestion logic

## Configuration
- All config options
- Default values
- Examples

## CLI Commands
- stats
- show
- clear
- export

## Usage Examples
- Basic usage
- Advanced scenarios

## Troubleshooting
- Common issues
- Solutions

## FAQ
```

**Key points**:
- User-friendly: Write for engineers unfamiliar with the system
- Examples: Include real command examples with output
- Troubleshooting: Address common issues proactively

### Verification

**Manual testing**:
1. Review documentation: `cat docs/learning.md` (expected: Comprehensive and clear)
2. Check links: `markdown-link-check docs/learning.md` (expected: All links valid)

**Success criteria**:
- Documentation complete
- All features explained
- Examples provided

### Commit

```
docs: add comprehensive learning system documentation

Complete documentation for learning system:
- docs/learning.md with full details
- README.md updated with features
- Architecture explanations
- CLI command reference
- Configuration guide
- Troubleshooting section
- Usage examples
```

**Files to commit**:
- docs/learning.md
- README.md

## Task 22: End-to-end validation with real plan files

**Agent**: test-automator
**File(s)**: test/integration/learning_e2e_test.go
**Depends on**: Task 20
**WorktreeGroup**: chain-6-final-integration
**Estimated time**: 1h 30m

Create end-to-end validation tests using real plan files to ensure the complete learning system works in production scenarios.

### Test-First Approach

**Test file**: test/integration/learning_e2e_test.go

**Test structure**:
- TestE2E_LearningEnabled
- TestE2E_LearningDisabled
- TestE2E_FailureAdaptation

**Fixtures**:
- Real plan files in test/integration/fixtures/

**Assertions**:
- Complete workflow executes
- Learning adapts correctly
- CLI commands work

**Edge cases**:
- Very large plans
- Complex dependencies

**Example test skeleton**:
```go
func TestE2E_LearningEnabled(t *testing.T) {
    // Run conductor with real plan
    // Verify learning database created
    // Check execution recorded
    // Run again, verify adaptation
}
```

### Implementation

**Approach**:
1. Create E2E test file in test/integration/
2. Use real conductor binary
3. Execute with real plan files
4. Verify database created and populated
5. Test CLI commands against real database
6. Validate complete workflow

**Code structure**:
```go
func TestE2E_XXX(t *testing.T) {
    // Build conductor binary
    buildBinary(t)

    // Setup test directory
    tmpDir := t.TempDir()

    // Run conductor
    output := runConductor(tmpDir, "run", "plan.md")

    // Verify learning
    verifyLearningDatabase(tmpDir)
    verifyCLICommands(tmpDir)
}
```

**Key points**:
- Real binary: Build and test actual conductor executable
- Isolated environment: Each test in temp directory

### Verification

**Manual testing**:
1. Run E2E tests: `go test ./test/integration/ -run E2E -v` (expected: All E2E tests pass)

**Automated tests**:
```bash
go test ./test/integration/learning_e2e_test.go -v
```

**Success criteria**:
- Complete system validated
- Production scenarios tested

### Commit

```
test: add end-to-end validation suite

Comprehensive E2E validation:
- Real binary testing
- Production scenarios
- Complete workflow validation
- CLI command testing
```

**Files to commit**:
- test/integration/learning_e2e_test.go

## Task 23: Update example config with learning settings

**Agent**: fullstack-developer
**File(s)**: .conductor/config.yaml.example
**Depends on**: Task 22
**WorktreeGroup**: chain-6-final-integration
**Estimated time**: 15m

Update the example configuration file to include all learning settings with helpful comments and sensible defaults.

### Test-First Approach

N/A - documentation file

### Implementation

**Approach**:
1. Open .conductor/config.yaml.example
2. Add learning section with all options
3. Include helpful comments for each setting
4. Show default values
5. Provide usage examples

**Code structure**:
```yaml
# Learning system configuration
learning:
  # Enable/disable learning (default: true)
  enabled: true

  # Database path (default: .conductor/learning)
  db_path: .conductor/learning

  # Automatically adapt agent after failures (default: true)
  auto_adapt_agent: true

  # Enhance prompts with learning context (default: true)
  enhance_prompts: true

  # Minimum failures before adapting (default: 2)
  min_failures_before_adapt: 2

  # Keep execution data for N days, 0 = forever (default: 90)
  keep_executions_days: 90

  # Maximum executions to keep per task (default: 100)
  max_executions_per_task: 100
```

**Key points**:
- Comments: Explain each option clearly
- Defaults shown: Show what happens if not specified

### Verification

**Manual testing**:
1. Review example config: `cat .conductor/config.yaml.example` (expected: Learning section present and clear)

**Success criteria**:
- All options documented
- Examples helpful

### Commit

```
docs: update example config with learning settings

Add learning configuration to example:
- All learning options documented
- Helpful comments for each setting
- Default values shown
- Usage guidance included
```

**Files to commit**:
- .conductor/config.yaml.example

## Task 24: Bump version to 2.0.0 and update documentation

**Agent**: technical-documentation-specialist
**File(s)**: VERSION, CLAUDE.md
**Depends on**: Task 23
**WorktreeGroup**: chain-6-final-integration
**Estimated time**: 30m

Update VERSION file to 2.0.0 and update CLAUDE.md to document the new learning system architecture and features.

### Test-First Approach

N/A - version bump

### Implementation

**Approach**:
1. Update VERSION file to 2.0.0
2. Update CLAUDE.md with learning system info
3. Document new architecture components
4. Update feature list
5. Add learning system to "Current Status"

**Code structure**:
```
# VERSION file
2.0.0

# CLAUDE.md updates
- Add learning system to architecture overview
- Document new internal/learning/ package
- Add learning configuration section
- Update version in project overview
```

**Key points**:
- Major version: 2.0.0 indicates significant new feature
- Documentation completeness: Ensure CLAUDE.md fully describes learning system

### Verification

**Manual testing**:
1. Check version: `cat VERSION` (expected: 2.0.0)
2. Verify CLAUDE.md: `grep -i learning CLAUDE.md` (expected: Learning system documented)

**Automated tests**:
```bash
./conductor --version
```
Expected output: conductor version 2.0.0

**Success criteria**:
- Version updated to 2.0.0
- Documentation complete

### Commit

```
chore: bump version to 2.0.0

Release version 2.0.0 with learning system:
- Update VERSION to 2.0.0
- Update CLAUDE.md with learning architecture
- Document new features and components
- Major version for significant new capability
```

**Files to commit**:
- VERSION
- CLAUDE.md

## Task 25: Prepare 2.0.0 release with CHANGELOG and final validation

**Agent**: fullstack-developer
**File(s)**: CHANGELOG.md
**Depends on**: Task 24
**WorktreeGroup**: chain-6-final-integration
**Estimated time**: 45m

Create comprehensive CHANGELOG for 2.0.0 release, run final validation, and ensure all components are production-ready.

### Test-First Approach

N/A - release preparation

### Implementation

**Approach**:
1. Create/update CHANGELOG.md with 2.0.0 entry
2. List all new features
3. Document breaking changes (if any)
4. Run full test suite
5. Build binary and verify version
6. Test all CLI commands manually
7. Validate documentation completeness

**Code structure**:
```markdown
# CHANGELOG.md

## [2.0.0] - 2025-01-12

### Added
- Adaptive learning system with SQLite database
- Automatic agent adaptation after repeated failures
- Pattern detection (compilation, test, dependency errors)
- Session management with auto-generated IDs
- Run number tracking per plan file
- Four new CLI commands: stats, show, clear, export
- Learning configuration options
- Cross-run intelligence and prompt enhancement
- Graceful degradation on learning failures

### Changed
- Orchestrator now supports learning store integration
- TaskExecutor includes three hook points
- Configuration schema includes learning section

### Fixed
- N/A (no fixes in this release)

### Breaking Changes
- None (fully backward compatible)
```

**Key points**:
- Comprehensive changelog: List all features and changes
- Validation checklist: Verify everything works before release

### Verification

**Manual testing**:
1. Run full test suite: `go test ./... -v` (expected: All tests pass)
2. Build binary: `make build` (expected: Binary builds successfully)
3. Check version: `./conductor --version` (expected: conductor version 2.0.0)
4. Test learning commands: `./conductor learning --help` (expected: All subcommands listed)

**Automated tests**:
```bash
go test ./... -race -cover
```
Expected output: PASS, coverage >= 86%

**Success criteria**:
- CHANGELOG complete
- All tests pass
- Binary builds and runs
- Documentation complete
- Ready for release

### Commit

```
chore: prepare 2.0.0 release with learning system

Final release preparation for v2.0.0:
- Comprehensive CHANGELOG entry
- All features documented
- Full test suite passing
- Binary validated
- Production-ready adaptive learning system

This major release introduces intelligent learning
that adapts agent selection based on execution history,
enabling conductor to improve success rates over time.
```

**Files to commit**:
- CHANGELOG.md

---

## Testing Strategy

### Unit Tests
- **Location**: internal/**/*_test.go
- **Naming convention**: *_test.go files alongside source
- **Run command**: `go test ./...`
- **Coverage target**: 86%+
- **Coverage command**: `go test ./... -cover`

### Integration Tests
- **Location**: internal/executor/learning_integration_test.go, internal/cmd/learning_integration_test.go
- **What to test**:
  - Complete learning pipeline from execution to recording
  - Cross-run learning scenarios
  - CLI command end-to-end functionality
  - Agent adaptation after failures
- **Setup required**:
  - In-memory SQLite database for isolated tests
  - Temp directories for file-based tests
  - Mock task execution results
- **Run command**: `go test ./internal/executor/ -run Integration -v`

### E2E Tests
- **Enabled**: true
- **Location**: test/integration/learning_e2e_test.go
- **Critical flows**:
  - Full conductor execution with learning enabled
  - Multiple runs with failure adaptation
  - CLI commands against real database
- **Tools**: Go test framework with real binary
- **Run command**: `go test ./test/integration/ -run E2E -v`

### Test Design Principles

**Use these patterns**:
1. Table-driven tests:
```go
tests := []struct {
    name    string
    input   interface{}
    want    interface{}
    wantErr bool
}{
    {"case 1", input1, output1, false},
    {"case 2", input2, output2, true},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        // test logic
    })
}
```

2. TDD Red-Green-Refactor:
   - RED: Write failing test
   - GREEN: Minimal code to pass
   - REFACTOR: Improve code quality

**Avoid these anti-patterns**:
1. Testing implementation details (makes tests brittle - test behavior instead)
2. Shared mutable state between tests (creates flakiness - use t.TempDir() instead)

**Mocking guidelines**:
- Mock these: Claude CLI invocations (use InvokerInterface), file system operations, time-dependent operations
- Don't mock these: SQLite database (use :memory:), business logic, simple data transformations
- Project pattern: See internal/executor/task_test.go

## Commit Strategy

**Total commits**: 25

**Commit sequence**:
1. (Task 1) feat: Add learning database schema with SQLite
2. (Task 2) feat: Implement learning store with execution recording
3. (Task 3) feat: Add failure analysis and agent suggestion logic
4-7. (Tasks 4-7) feat: Executor integration (pre-task, QC-review, post-task hooks, orchestrator)
8-12. (Tasks 8-12) feat: CLI commands (stats, show, clear, export, root integration)
13-17. (Tasks 13-17) feat/chore: Configuration and dependencies
18-20. (Tasks 18-20) test: Integration tests
21. (Task 21) docs: Learning system documentation
22-25. (Tasks 22-25) test/chore: Final validation and release

**Message format**:
```
type: brief description in present tense

[Optional body explaining why]
```

**Examples from project history**:
- feat: add learning database schema with SQLite
- test: add integration tests for learning pipeline
- chore: bump version to 2.0.0

**Guidelines**:
- Keep commits atomic - one task per commit
- Write clear, descriptive messages in imperative mood
- Follow conventional commit format (feat, fix, test, docs, chore)
- Include test files in same commit as implementation
- Commit after each completed task

## Common Pitfalls & How to Avoid Them

1. **Forgetting CGO requirement for mattn/go-sqlite3**
   - Why: mattn/go-sqlite3 requires C compiler (CGO) to build
   - How to avoid: Ensure CGO_ENABLED=1 and C compiler available before building
   - Reference: go.mod

2. **Learning failures breaking task execution**
   - Why: Learning should enhance, not break, task execution
   - How to avoid: All learning operations wrapped in error handlers that log warnings but continue execution
   - Reference: internal/executor/task.go (graceful degradation pattern in hooks)

3. **Database path resolution issues**
   - Why: Relative paths can be confusing with worktrees
   - How to avoid: Always resolve database path relative to plan file location
   - Reference: internal/learning/store.go

4. **Race conditions in concurrent execution**
   - Why: Multiple tasks might try to write to database simultaneously
   - How to avoid: SQLite handles locking automatically, but test with -race flag
   - Command: `go test -race ./...`

5. **Forgetting to close database connections**
   - Why: Can lead to resource leaks
   - How to avoid: Always defer store.Close() after creating store
   - Reference: internal/executor/orchestrator.go

6. **Not handling :memory: vs file databases correctly in tests**
   - Why: In-memory databases don't persist, file databases do
   - How to avoid: Use :memory: for unit tests, temp files for integration tests
   - Reference: internal/learning/store_test.go

## Resources & References

### Existing Code to Reference
- Executor package structure: internal/executor/ (study task.go, orchestrator.go)
- Config package patterns: internal/config/ (config loading and merging)
- CLI command structure: internal/cmd/ (cobra patterns from run.go, validate.go)
- Test patterns: internal/executor/task_test.go (table-driven tests with mocks)
- Models package: internal/models/ (Task, Plan, Result structures)

### Documentation
- SQLite Go driver: https://github.com/mattn/go-sqlite3
- Cobra CLI framework: https://github.com/spf13/cobra
- Go testing: https://golang.org/pkg/testing/
- Go database/sql: https://golang.org/pkg/database/sql/
- Project architecture: CLAUDE.md

### External Resources
- SQLite query optimization: https://www.sqlite.org/queryplanner.html
- Go concurrency patterns: https://go.dev/blog/pipelines

### Validation Checklist
- [ ] All tests pass: `go test ./...`
- [ ] Race detector passes: `go test -race ./...`
- [ ] Test coverage >= 86%: `go test ./... -cover`
- [ ] Linter passes: `go vet ./...`
- [ ] Binary builds successfully: `go build ./cmd/conductor`
- [ ] Version displays correctly: `./conductor --version`
- [ ] Learning commands work: `./conductor learning --help`
- [ ] Documentation complete: docs/learning.md, README.md updated
- [ ] CHANGELOG updated: 2.0.0 entry with all features
- [ ] Example config updated: .conductor/config.yaml.example has learning section
- [ ] Worktree branches merged: All feature branches merged to main
- [ ] Git tag created: `git tag v2.0.0`

---

**Plan Complete**: Tasks 1-25 fully detailed across two files. Begin with plan-01-foundation.md Task 1.
