# Skip Test Plan: Markdown with Completed Tasks

**Created**: 2025-11-11
**Status**: Partial completion for skip testing

## Task 1: Already Completed Task

**File(s)**: `main.go`
**Depends on**: None
**Estimated time**: 10m
**Agent**: general-purpose
**Status**: completed

### What you're building
This task has already been completed and should be skipped during execution.

### Implementation
This task will be skipped when --skip-completed is enabled.

---

## Task 2: Pending Task

**File(s)**: `handler.go`
**Depends on**: 1
**Estimated time**: 10m
**Agent**: general-purpose

### What you're building
This task is pending and must be executed.

### Implementation
This task must be executed normally.

---

## Task 3: Another Completed Task

**File(s)**: `utils.go`
**Depends on**: 2
**Estimated time**: 10m
**Agent**: general-purpose
**Status**: completed

### What you're building
Another completed task that should be skipped.

### Implementation
This will be skipped when skip is enabled.

---

## Task 4: Final Pending Task

**File(s)**: `test.go`
**Depends on**: 3
**Estimated time**: 10m
**Agent**: general-purpose

### What you're building
Final task that must execute.

### Implementation
This task must execute normally.
