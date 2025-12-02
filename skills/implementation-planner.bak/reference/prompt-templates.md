# Prompt Templates for Robust Task Execution

Templates ensure agents verify dependencies before coding and avoid common anti-patterns.

## Prompt Structure

Every task prompt should include these sections IN ORDER:

```
1. PHASE 0: DEPENDENCY VERIFICATION (Mandatory pre-checks)
2. IMPLEMENTATION REQUIREMENTS (What to call, what's prohibited)
3. TASK DESCRIPTION (The actual work)
4. SUCCESS CRITERIA (Verification checklist)
```

## Phase 0: Dependency Verification Template

**Add to every task that has `depends_on`:**

```yaml
description: |
  # Task N: [Task Name]

  ## PHASE 0: DEPENDENCY VERIFICATION (EXECUTE FIRST)

  **You MUST verify these dependencies exist before implementing:**

  ```bash
  # Verify Task 4 complete (ExtractMetrics function exists)
  grep -q "func ExtractMetrics" internal/behavioral/extractor.go && \
    echo "✓ Task 4: ExtractMetrics ready" || \
    echo "❌ STOP: Task 4 incomplete - ExtractMetrics missing"

  # Verify Task 5 complete (Aggregator exists)
  grep -q "type Aggregator struct" internal/behavioral/aggregator.go && \
    echo "✓ Task 5: Aggregator ready" || \
    echo "❌ STOP: Task 5 incomplete - Aggregator missing"
  ```

  **If ANY check fails:** Report "Dependency Task X incomplete" and STOP.
  Do NOT proceed with placeholder implementations.

  ---

  ## IMPLEMENTATION REQUIREMENTS
  [...]

  ## TASK DESCRIPTION
  [...]
```

## Implementation Requirements Template

**Standard requirements block:**

```yaml
  ## IMPLEMENTATION REQUIREMENTS

  **You MUST call these functions (from upstream tasks):**
  - `extractor.ExtractMetrics(sessionData)` - from Task 4
  - `aggregator.LoadSession(sessionFile)` - from Task 5
  - `filter.ApplyFilters(metrics, criteria)` - from Task 15

  **PROHIBITED (will cause QC RED):**
  - ❌ TODO comments in production code paths
  - ❌ Placeholder empty structs: `metrics := &BehavioralMetrics{}`
  - ❌ Unused variables: `_ = criteria`
  - ❌ Mocking upstream functions instead of calling them
  - ❌ Hardcoded test data instead of real function calls

  **REQUIRED (for QC GREEN):**
  - ✓ All upstream function calls resolve at compile time
  - ✓ Real data flows through the implementation
  - ✓ Error handling for all function calls
  - ✓ Tests use real (or properly mocked) dependencies
```

## Anti-Pattern Detection Criteria

**Auto-append to EVERY task's success_criteria:**

```yaml
success_criteria:
  # Task-specific criteria (from description)
  - "Export to JSON format"
  - "Export to Markdown format"
  - "Export to CSV format"

  # AUTO-APPENDED: Anti-pattern criteria
  - "No TODO comments in production code paths"
  - "No placeholder empty structs (e.g., Type{})"
  - "No unused variables (_ = x pattern)"
  - "All imports from dependency tasks resolve"
  - "All upstream function calls compile successfully"
```

## Language-Specific Verification Commands

### Go

```yaml
## PHASE 0: DEPENDENCY VERIFICATION

```bash
# Function existence
grep -q "func FunctionName" path/to/file.go && echo "✓ Ready" || echo "❌ Missing"

# Type existence
grep -q "type TypeName struct" path/to/file.go && echo "✓ Ready" || echo "❌ Missing"

# Interface existence
grep -q "type InterfaceName interface" path/to/file.go && echo "✓ Ready" || echo "❌ Missing"

# Method existence
grep -q "func.*TypeName.*MethodName" path/to/file.go && echo "✓ Ready" || echo "❌ Missing"

# Build verification (most thorough)
go build ./path/to/package && echo "✓ Compiles" || echo "❌ Build fails"
```
```

### TypeScript/JavaScript

```yaml
## PHASE 0: DEPENDENCY VERIFICATION

```bash
# Export existence
grep -q "export.*functionName" path/to/file.ts && echo "✓ Ready" || echo "❌ Missing"

# Class existence
grep -q "export class ClassName" path/to/file.ts && echo "✓ Ready" || echo "❌ Missing"

# Interface existence
grep -q "export interface InterfaceName" path/to/file.ts && echo "✓ Ready" || echo "❌ Missing"

# Type check (most thorough)
npx tsc --noEmit && echo "✓ Types valid" || echo "❌ Type errors"
```
```

### Python

```yaml
## PHASE 0: DEPENDENCY VERIFICATION

```bash
# Function existence
grep -q "def function_name" path/to/file.py && echo "✓ Ready" || echo "❌ Missing"

# Class existence
grep -q "class ClassName" path/to/file.py && echo "✓ Ready" || echo "❌ Missing"

# Import check
python -c "from module import function" && echo "✓ Importable" || echo "❌ Import fails"
```
```

## Full Task Prompt Example

```yaml
- task_number: 16
  name: "Add export functionality (JSON, Markdown, CSV)"
  agent: "golang-pro"
  depends_on: [4, 5, 13, 15]

  # Data flow documentation
  calls:
    - function: "ExtractMetrics"
      from_task: 4
    - function: "LoadSession"
      from_task: 5
    - function: "ApplyFilters"
      from_task: 15

  success_criteria:
    # Functional criteria
    - "Export to JSON format with proper structure"
    - "Export to Markdown format with tables"
    - "Export to CSV format with headers"
    - "Filters applied to exported data"

    # Data integration criteria (auto-appended)
    - "Calls extractor.ExtractMetrics() with real SessionData"
    - "Calls aggregator.LoadSession() for session files"
    - "Calls filter.ApplyFilters() when criteria specified"

    # Anti-pattern criteria (auto-appended)
    - "No TODO comments in observe_export.go"
    - "No placeholder empty structs"
    - "No unused variables"
    - "All imports resolve"

  description: |
    # Task 16: Add export functionality (JSON, Markdown, CSV)

    ## PHASE 0: DEPENDENCY VERIFICATION (EXECUTE FIRST)

    **You MUST verify these dependencies exist before implementing:**

    ```bash
    # Verify Task 4: ExtractMetrics function
    grep -q "func ExtractMetrics" internal/behavioral/extractor.go && \
      echo "✓ Task 4: ExtractMetrics ready" || \
      echo "❌ STOP: Task 4 incomplete"

    # Verify Task 5: Aggregator and LoadSession
    grep -q "type Aggregator struct" internal/behavioral/aggregator.go && \
      echo "✓ Task 5: Aggregator ready" || \
      echo "❌ STOP: Task 5 incomplete"

    # Verify Task 15: ApplyFilters function
    grep -q "func ApplyFilters" internal/behavioral/filter.go && \
      echo "✓ Task 15: ApplyFilters ready" || \
      echo "❌ STOP: Task 15 incomplete"

    # Verify compilation
    go build ./internal/behavioral/... && \
      echo "✓ Dependencies compile" || \
      echo "❌ STOP: Dependency compilation failed"
    ```

    **If ANY check fails:** Report "Dependency Task X incomplete" and STOP.

    ---

    ## IMPLEMENTATION REQUIREMENTS

    **You MUST call these functions:**
    - `extractor.ExtractMetrics(sessionData)` - from Task 4
    - `aggregator.LoadSession(sessionFile)` - from Task 5
    - `filter.ApplyFilters(metrics, criteria)` - from Task 15

    **PROHIBITED:**
    - ❌ TODO comments in production paths
    - ❌ Placeholder structs: `metrics := &BehavioralMetrics{}`
    - ❌ Unused variables: `_ = criteria`
    - ❌ Mocking instead of calling real functions

    ---

    ## TASK DESCRIPTION

    Implement export functionality for behavioral metrics:

    1. **JSON Export**: Structured JSON with metrics, sessions, patterns
    2. **Markdown Export**: Human-readable report with tables
    3. **CSV Export**: Flat data suitable for spreadsheets

    Use the extractor (Task 4) to get metrics, aggregator (Task 5) for
    session loading, and filter (Task 15) to apply user-specified criteria.

    ---

    ## VERIFICATION CHECKLIST

    After implementation, verify:
    - [ ] `go build ./internal/cmd/` succeeds
    - [ ] `grep TODO internal/cmd/observe_export.go` returns no matches
    - [ ] `go test ./internal/cmd/ -run Export -v` passes
```

## Criteria Categories

### Functional Criteria

What the task should accomplish:

```yaml
success_criteria:
  - "Export to JSON format"
  - "Handle empty metrics gracefully"
  - "Support file output and stdout"
```

### Data Integration Criteria

How the task uses upstream dependencies:

```yaml
success_criteria:
  - "Calls extractor.ExtractMetrics() with real SessionData"
  - "Calls aggregator.LoadSession() for cached sessions"
  - "Uses filter.ApplyFilters() when criteria provided"
```

### Anti-Pattern Criteria

What the task must NOT do:

```yaml
success_criteria:
  - "No TODO comments in production code"
  - "No placeholder empty structs"
  - "No unused variables (_ = x)"
  - "No hardcoded test data in production paths"
```

### Build/Compile Criteria

Verification that code works:

```yaml
success_criteria:
  - "go build ./... succeeds"
  - "All imports resolve"
  - "go vet ./... passes"
```

## Quick Reference

**For tasks WITH dependencies:**

```
1. Add PHASE 0 verification commands
2. Add IMPLEMENTATION REQUIREMENTS with:
   - Functions that MUST be called
   - PROHIBITED patterns
3. Add data integration criteria
4. Add anti-pattern criteria
```

**For tasks WITHOUT dependencies (foundation):**

```
1. Skip PHASE 0 (no dependencies to verify)
2. Add anti-pattern criteria
3. Document what this task PRODUCES
```

**Standard anti-patterns to ALWAYS include:**

```yaml
- "No TODO comments in production code"
- "No placeholder empty structs"
- "No unused variables"
- "All imports resolve"
```
