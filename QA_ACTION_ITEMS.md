# QA Review - Action Items & Checklist

**Review Date**: 2025-11-23
**Test Suite**: QC Feedback Injection Fix Tests
**Overall Status**: GOOD - Production Ready (1 critical fix required)

---

## BLOCKER: Must Fix Before Merge

### Bug: File Path Mismatch in postTaskHook

**Issue ID**: QA-001
**Severity**: CRITICAL
**Impact**: Data integrity in multi-file plans
**Type**: Code bug (not test bug)

**Location**: `internal/executor/task.go` Line 441

**Current Code**:
```go
exec := &learning.TaskExecution{
    PlanFile: te.PlanFile,  // WRONG for multi-file plans
    ...
}
```

**Fixed Code**:
```go
exec := &learning.TaskExecution{
    PlanFile: fileToQuery,  // Use same file as query
    ...
}
```

**Why It's Broken**:
- Line 384-389: Correctly determines `fileToQuery` (uses `task.SourceFile` if present)
- Line 393: Correctly queries history from `fileToQuery`
- Line 441: **BUG** - Records execution to `te.PlanFile` instead of `fileToQuery`
- Result: History queries from one file, records to another (split records)

**Test Gap**:
- `TestRetry_CorrectFilePathForMultiFile` only verifies retry loop records (line 716)
- Does NOT verify postTaskHook record uses correct file
- Bug hidden because test doesn't check postTaskHook behavior

**How to Fix**:
1. Open `internal/executor/task.go`
2. Go to line 441
3. Change: `PlanFile: te.PlanFile,`
4. To: `PlanFile: fileToQuery,`
5. Save and test

**Verification**:
```bash
# After fix, run tests
go test ./internal/executor/... -v
go test ./... -cover
```

**Time Required**: 2 minutes
**Must Do**: YES - Before any merge

---

## HIGH PRIORITY: Recommended Before Merge

### Test: Database Error Graceful Degradation

**Issue ID**: QA-002
**Type**: Missing test coverage
**Priority**: HIGH
**Time**: 15 minutes

**Gap**: Code has graceful degradation for DB errors (lines 732-735, 457-461) but no test

**Test Outline**:
```go
func TestRetry_GracefulDegradationOnDBError(t *testing.T) {
    // Setup: Mock that returns error on RecordExecution
    mockStore := &MockLearningStore{
        RecordExecutionError: errors.New("connection timeout"),
    }

    executor.LearningStore = mockStore

    // Execute task with QC enabled (triggers immediate writes)
    result, err := executor.Execute(context.Background(), task)

    // Assert: Task completes successfully despite DB error
    if result.Status != models.StatusGreen {
        t.Fatalf("task should succeed despite DB error, got status %s", result.Status)
    }

    // Assert: No task execution error
    if err != nil {
        t.Fatalf("execute should not return error despite DB failure, got %v", err)
    }
}
```

**Where to Add**: `internal/executor/task_test.go` after line 777

**Why Important**: Validates resilience - DB failures shouldn't crash tasks

---

### Test Enhancement: postTaskHook Multi-File Verification

**Issue ID**: QA-003
**Type**: Test coverage gap
**Priority**: HIGH
**Time**: 10 minutes

**Gap**: `TestRetry_CorrectFilePathForMultiFile` only tests retry loop, not postTaskHook

**Enhancement**:
```go
func TestRetry_PostTaskHookUsesSourceFile(t *testing.T) {
    // Same setup as TestRetry_CorrectFilePathForMultiFile
    // BUT: Task succeeds on first attempt (GREEN immediately)
    // This forces postTaskHook to record the final verdict

    invoker := newStubInvoker(
        &agent.InvocationResult{Output: `{"content":"attempt1"}`, ExitCode: 0},
    )

    reviewer := &stubReviewer{
        results: []*ReviewResult{
            {Flag: models.StatusGreen, Feedback: "Pass"},
        },
        retryDecisions: map[int]bool{0: false},  // No retry needed
    }

    task := models.Task{
        Number: "1",
        SourceFile: "plans/module-a.yaml",  // Different from PlanFile
        Prompt: "Test",
        Agent: "test-agent",
    }

    result, _ := executor.Execute(context.Background(), task)

    // Assert: postTaskHook record uses SourceFile
    executions := mockStore.GetRecordedExecutions()
    if executions[0].PlanFile != "plans/module-a.yaml" {
        t.Errorf("postTaskHook should use SourceFile, got %q", executions[0].PlanFile)
    }
}
```

**Where to Add**: `internal/executor/task_test.go` after `TestRetry_CorrectFilePathForMultiFile`

**Why Important**: Validates the fix for QA-001

---

## MEDIUM PRIORITY: Nice to Have

### Mock Improvement: Reverse-Sort History

**Issue ID**: QA-004
**Type**: Mock fidelity
**Priority**: MEDIUM
**Time**: 5 minutes

**Current Issue**: MockLearningStore returns history in insertion order (FIFO)
Real databases return most recent first

**Current Code** (Lines 353-364):
```go
func (m *MockLearningStore) GetExecutionHistory(...) ([]*learning.TaskExecution, error) {
    var history []*learning.TaskExecution
    for _, exec := range m.RecordedExecutions {
        if exec.PlanFile == planFile && exec.TaskNumber == taskNumber {
            history = append(history, exec)  // Insertion order
        }
    }
    return history, nil
}
```

**Fixed Code**:
```go
func (m *MockLearningStore) GetExecutionHistory(...) ([]*learning.TaskExecution, error) {
    var history []*learning.TaskExecution
    for _, exec := range m.RecordedExecutions {
        if exec.PlanFile == planFile && exec.TaskNumber == taskNumber {
            history = append(history, exec)
        }
    }

    // Match real database behavior: most recent first
    sort.Slice(history, func(i, j int) bool {
        return history[i].Timestamp > history[j].Timestamp
    })

    return history, nil
}
```

**Note**: Requires `learning.TaskExecution` to have Timestamp field
Check if it exists before implementing

**Why**: Tests currently work with small datasets but would fail with 3+ records

---

### Test: Context Cancellation Handling

**Issue ID**: QA-005
**Type**: Missing edge case test
**Priority**: LOW
**Time**: 15 minutes

**Test Outline**:
```go
func TestRetry_ContextCancelledDuringDBWrite(t *testing.T) {
    ctx, cancel := context.WithCancel(context.Background())

    // Cancel immediately
    cancel()

    // Execute with cancelled context
    result, err := executor.Execute(ctx, task)

    // Assert: No panic, graceful handling
    // Allow context error but task shouldn't crash
    if result.Status == models.StatusFailed {
        // OK - context error causes failure
    } else if result.Status == models.StatusGreen {
        // OK - task completed before error
    }
}
```

**Why**: Edge case validation, ensure no panics

---

## VERIFICATION CHECKLIST

Before marking tests as complete, verify:

- [ ] All 6 existing tests still pass: `go test ./internal/executor/... -v -run "TestRetry_|TestPostTaskHook_"`
- [ ] Fix for QA-001 applied
- [ ] TestRetry_GracefulDegradationOnDBError added
- [ ] TestRetry_PostTaskHookUsesSourceFile added
- [ ] MockLearningStore reverse-sort optional but recommended
- [ ] Code coverage maintained: `go test ./... -cover` shows >= 82%
- [ ] No new test failures introduced
- [ ] All tests pass in under 1 second total

---

## SUMMARY TABLE

| ID | Item | Type | Priority | Time | Status |
|----|---------------------------------|--------|----------|------|--------|
| QA-001 | Fix postTaskHook file path | Bug Fix | CRITICAL | 2m | TODO |
| QA-002 | Add DB error graceful degradation test | Test | HIGH | 15m | TODO |
| QA-003 | Enhance TestRetry_CorrectFilePathForMultiFile | Test | HIGH | 10m | TODO |
| QA-004 | Mock reverse-sort GetExecutionHistory | Mock | MEDIUM | 5m | TODO |
| QA-005 | Context cancellation test | Test | LOW | 15m | OPTIONAL |

---

## COMPLETION CRITERIA

**Phase 1: CRITICAL (Must complete)**
- [ ] QA-001 fixed and verified
- [ ] All existing tests still pass
- [ ] No new test failures

**Phase 2: RECOMMENDED (Should complete this sprint)**
- [ ] QA-002 test added and passing
- [ ] QA-003 test added and passing
- [ ] Code coverage still >= 82%

**Phase 3: OPTIONAL (Can do later)**
- [ ] QA-004 mock improved
- [ ] QA-005 edge case test added

---

## SIGN-OFF

Once all CRITICAL and RECOMMENDED items are complete, this test suite is approved for merge:

- [ ] QA-001 fixed
- [ ] QA-002 added
- [ ] QA-003 added
- [ ] Tests passing: 8+ total
- [ ] Coverage: >= 82%
- [ ] Ready for merge

---

## RELATED DOCUMENTS

- `QA_TEST_REVIEW.md` - Detailed comprehensive analysis
- `QA_FINDINGS_SUMMARY.md` - Executive summary of findings
- Test files: `internal/executor/task_test.go`
- Source files: `internal/executor/task.go`

---

**Report Generated**: 2025-11-23
**Review Status**: In Progress
**Next Steps**: Apply QA-001 fix and add recommended tests
