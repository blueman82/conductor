# Complex Dependency Graph

## Task 1: Foundation

**File(s)**: `foundation.go`
**Depends on**: None
**Estimated time**: 10m
**Agent**: test-agent

Base task with no dependencies.

## Task 2: Layer 1A

**File(s)**: `layer1a.go`
**Depends on**: Task 1
**Estimated time**: 10m
**Agent**: test-agent

First layer, first branch.

## Task 3: Layer 1B

**File(s)**: `layer1b.go`
**Depends on**: Task 1
**Estimated time**: 10m
**Agent**: test-agent

First layer, second branch.

## Task 4: Layer 2A

**File(s)**: `layer2a.go`
**Depends on**: Task 2
**Estimated time**: 10m
**Agent**: test-agent

Second layer, depends on 1A.

## Task 5: Layer 2B

**File(s)**: `layer2b.go`
**Depends on**: Task 2, Task 3
**Estimated time**: 10m
**Agent**: test-agent

Second layer, depends on both 1A and 1B.

## Task 6: Layer 2C

**File(s)**: `layer2c.go`
**Depends on**: Task 3
**Estimated time**: 10m
**Agent**: test-agent

Second layer, depends on 1B.

## Task 7: Final Integration

**File(s)**: `final.go`
**Depends on**: Task 4, Task 5, Task 6
**Estimated time**: 15m
**Agent**: test-agent

Final task depends on all layer 2 tasks.
