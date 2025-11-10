# Plan with Intentional Failure

## Task 1: Successful Task

**File(s)**: `success.go`
**Depends on**: None
**Estimated time**: 5m
**Agent**: test-agent

This task should succeed.

## Task 2: Failing Task

**File(s)**: `failure.go`
**Depends on**: Task 1
**Estimated time**: 5m
**Agent**: failing-agent

This task will fail intentionally to test error handling.

## Task 3: Blocked Task

**File(s)**: `blocked.go`
**Depends on**: Task 2
**Estimated time**: 5m
**Agent**: test-agent

This task should be blocked by Task 2's failure.
