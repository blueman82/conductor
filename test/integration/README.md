# Conductor Integration Tests

This directory contains end-to-end integration tests for the Conductor multi-agent orchestration CLI.

## Overview

The integration test suite validates the complete Conductor workflow from plan parsing through execution, including:

- Plan parsing (Markdown and YAML formats)
- Dependency graph construction and validation
- Wave calculation using Kahn's algorithm
- Multi-wave orchestration with parallel execution
- Error handling and recovery
- Timeout enforcement
- Graceful shutdown via context cancellation

## Test Structure

### Test File
- `conductor_test.go` (703 lines, 13 test functions)

### Test Fixtures
Located in `fixtures/` directory:

1. **simple-plan.md** - Basic 2-task Markdown plan with simple linear dependency
2. **simple-plan.yaml** - Same as above in YAML format (tests YAML parser)
3. **with-failure.md** - 3-task plan with intentional failure to test error handling
4. **multi-wave.md** - 5-task plan with complex dependencies across 3 waves
5. **cyclic.md** - Plan with circular dependencies (validation test)
6. **invalid.md** - Malformed plan with missing fields and invalid dependencies
7. **complex-dependencies.md** - 7-task plan with complex branching dependency graph
8. **large-plan.md** - 10-task plan for performance testing with parallel execution

## Test Cases

### Core Workflow Tests

#### TestE2E_SimpleMarkdownPlan
Tests complete workflow with Markdown plan:
- Parses 2-task plan
- Calculates 2 execution waves
- Executes tasks successfully
- Verifies orchestration results
- Validates logging calls

#### TestE2E_SimpleYamlPlan
Tests complete workflow with YAML plan:
- Parses YAML format with conductor config
- Validates configuration parsing (default_agent, max_concurrency)
- Executes 2-task plan
- Verifies successful completion

#### TestE2E_ComplexDependencies
Tests execution with complex 7-task dependency graph:
- Validates wave structure (4 waves)
- Verifies correct task grouping:
  - Wave 1: Task 1 (foundation)
  - Wave 2: Tasks 2, 3 (parallel branches)
  - Wave 3: Tasks 4, 5, 6 (dependent on wave 2)
  - Wave 4: Task 7 (final integration)
- Confirms all 7 tasks complete successfully

#### TestE2E_MultipleWaveExecution
Tests parallel execution across 3 waves:
- Validates 5-task plan structure
- Tracks execution order
- Verifies wave-based parallelism:
  - Wave 1: Tasks 1, 2 (parallel)
  - Wave 2: Task 3 (depends on 1, 2)
  - Wave 3: Tasks 4, 5 (parallel, depend on 3)
- Confirms all 5 tasks execute

### Error Handling Tests

#### TestE2E_FailureHandling
Tests task failure scenarios:
- Simulates Task 2 failing with RED status
- Verifies error is propagated
- Checks failed task appears in FailedTasks slice
- Confirms execution stops after failure

#### TestE2E_InvalidPlans
Table-driven test for malformed plans:
- Tests plan with missing fields
- Tests plan with invalid dependencies (non-existent tasks)
- Tests plan with circular dependencies
- Verifies appropriate error messages

#### TestE2E_CyclicDependencies
Dedicated test for cycle detection:
- Loads plan with circular dependency (1→3→2→1)
- Verifies CalculateWaves returns "circular dependency detected" error
- Confirms cycle detection prevents execution

### Operational Tests

#### TestE2E_DryRunMode
Tests validation without execution (dry-run):
- Parses plan successfully
- Validates task structure
- Calculates waves (3 waves for 5-task plan)
- Verifies wave 1 has 2 parallel tasks
- Does NOT execute orchestrator (validation only)

#### TestE2E_TimeoutHandling
Tests timeout enforcement via context:
- Simulates long-running tasks (5 second delay)
- Sets context timeout of 100ms
- Verifies context.DeadlineExceeded error
- Confirms result is still returned on timeout

#### TestE2E_GracefulShutdown
Tests signal handling and cancellation:
- Starts execution of large plan (10 tasks, 2s each)
- Cancels context after 50ms (simulates Ctrl+C)
- Verifies cancellation error propagates
- Confirms result returned even on cancellation
- Validates graceful cleanup

#### TestE2E_ResourceCleanup
Tests proper cleanup after execution:
- Executes simple plan
- Verifies result is non-nil
- Confirms FailedTasks slice is initialized (not nil)
- Validates no panics or resource leaks

#### TestE2E_LargePlan
Performance test with 10-task plan:
- Parses and executes 10-task plan
- Measures total execution time
- Verifies parallelism (should be faster than sequential)
- Expects completion under 500ms (actual: ~28ms)
- Confirms all 10 tasks complete

#### TestE2E_ConcurrentExecutions
Tests running multiple plans in parallel:
- Spawns 3 concurrent plan executions
- Each executes the same simple plan independently
- Verifies all 3 complete without errors
- Tests thread safety and isolation

## Running Tests

```bash
# Run all integration tests
go test ./test/integration/ -v

# Run with timeout (recommended for long-running tests)
go test ./test/integration/ -v -timeout 10m

# Run specific test
go test ./test/integration/ -run TestE2E_SimpleMarkdownPlan -v

# Run with coverage (note: integration tests cover external packages)
go test ./test/integration/ -v -cover

# Run with race detection
go test ./test/integration/ -v -race
```

## Test Results

**Total Tests**: 13 functions (15 test cases including subtests)
**Status**: All tests passing
**Execution Time**: ~0.4-0.6 seconds
**Coverage**: Integration tests exercise:
- `internal/parser` (Markdown and YAML parsers)
- `internal/executor` (graph, wave, orchestrator)
- `internal/models` (Task, Plan, Wave, Result)

## Mock Components

### mockTaskExecutor
Simulates task execution without invoking real Claude CLI:
- Configurable via `executeFunc` callback
- Default: returns StatusGreen for all tasks
- Can simulate failures, timeouts, specific behavior per task

### mockLogger
Captures logging calls for verification:
- Tracks `LogWaveStart` calls
- Tracks `LogWaveComplete` calls with duration
- Tracks `LogSummary` calls
- Enables assertion on logging behavior

## Test Coverage Philosophy

Integration tests focus on:
1. **End-to-end workflows** - Complete user journeys from parse to execution
2. **Component integration** - How parser, graph, executor, orchestrator work together
3. **Error paths** - Failures, timeouts, cancellations, invalid inputs
4. **Parallelism** - Multi-wave concurrent execution, race conditions
5. **Performance** - Large plans, execution timing

Integration tests do NOT:
- Test individual unit behavior (covered by unit tests)
- Invoke real Claude CLI (mocked for deterministic testing)
- Modify real files on disk (test fixtures are read-only)
- Test UI/CLI output formatting (covered by cmd tests)

## Future Enhancements

Potential additions for Task 25 (Final Integration):
- Real Claude CLI invocation tests (with test agents)
- File system integration (plan updates, locking)
- Configuration file loading tests
- Log file generation and verification
- Signal handling tests (actual SIGINT/SIGTERM)
- Network error simulation
- Resource exhaustion scenarios

## Notes

- All fixtures use `test-agent` to avoid dependency on real agent files
- Tests are deterministic and can run in any order
- No external dependencies required (no network, no real agents)
- Fast execution suitable for CI/CD pipelines
- Thread-safe for parallel test execution
