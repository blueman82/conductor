# Success Criteria Classification Reference

**Purpose:** Prevent cross-task scope leakage by properly classifying success criteria as CAPABILITY (component-scoped) or INTEGRATION (cross-component).

## The Problem

When success criteria describe behavior that spans multiple components, but the task only owns one component's files, the task becomes impossible to complete or verify.

**Real-world failure (Weather Oracle Task 14):**
```yaml
# Task 14: Cache Layer
files:
  - "packages/core/src/cache/index.ts"
  - "packages/core/src/cache/file-cache.ts"

success_criteria:
  - "Caches API responses with configurable TTL"      # ✅ Can verify with cache files
  - "Cache can be bypassed with --no-cache flag"      # ❌ Requires CLI program.ts!
```

The agent successfully implemented the cache with an `enabled` option, but QC failed because `--no-cache` CLI flag was never added to `program.ts` (owned by Task 10).

## Classification Framework

### CAPABILITY Criteria

**Definition:** Describes what a component CAN do in isolation.

**Characteristics:**
- Testable using ONLY the task's `files` list
- Describes internal behavior/state
- No references to external UI, CLI, or other components
- Can be verified with unit tests

**Examples:**
```yaml
# Good CAPABILITY criteria:
- "CacheManager constructor accepts { enabled: boolean } option"
- "When enabled=false, get() returns null without disk access"
- "set() respects maxEntries limit"
- "Expired entries are evicted on cleanup()"
- "invalidate(pattern) removes matching keys"
```

### INTEGRATION Criteria

**Definition:** Describes how multiple components WORK TOGETHER.

**Characteristics:**
- Requires files from multiple tasks/packages
- Describes end-to-end user-facing behavior
- References external systems (CLI, UI, API endpoints)
- Requires integration or E2E tests

**Examples:**
```yaml
# INTEGRATION criteria (require multiple components):
- "Cache can be bypassed with --no-cache flag"        # cache + CLI
- "Forecast shows loading spinner during fetch"       # engine + UI
- "API errors display user-friendly messages"         # API + errors + CLI
- "Clicking refresh triggers new model fetch"         # UI + API + engine
```

## Detection Keywords

Use these keywords to identify likely INTEGRATION criteria:

| Category | Keywords |
|----------|----------|
| **CLI** | `flag`, `--`, `command`, `argument`, `CLI`, `terminal`, `option` |
| **UI** | `displays`, `shows`, `renders`, `UI`, `screen`, `page`, `component`, `button` |
| **Cross-component** | `when X then Y`, `triggers`, `calls`, `invokes`, `after`, `before` |
| **User-facing** | `user can`, `user sees`, `allows user to` |

## Resolution Strategies

When you find an INTEGRATION criterion in a component task:

### Strategy 1: Split the Criterion

Separate the internal capability from the external integration.

**Before:**
```yaml
# Task 14 (Cache)
success_criteria:
  - "Cache can be bypassed with --no-cache flag"
```

**After:**
```yaml
# Task 14 (Cache) - CAPABILITY only
success_criteria:
  - "CacheManager constructor accepts { enabled: boolean } option"
  - "When enabled=false, get() always returns null"
  - "When enabled=false, set() is a no-op"

# Task 10 or 11 (CLI) - INTEGRATION
success_criteria:
  - "CLI --no-cache flag instantiates CacheManager with enabled=false"
```

### Strategy 2: Create Integration Task

When wiring is complex enough to warrant its own task.

```yaml
- task_number: 20
  type: integration
  name: "Wire Cache to CLI"
  files:
    - "packages/cli/src/program.ts"
    - "packages/cli/src/commands/forecast.ts"
  depends_on: [10, 14]  # CLI setup + Cache

  success_criteria:
    - "program.ts defines --no-cache global flag"

  integration_criteria:
    - "Forecast command respects --no-cache flag"
    - "Compare command respects --no-cache flag"
    - "Cache disabled when flag present"
```

### Strategy 3: Expand Task Scope

Add the external files to the task (use sparingly).

```yaml
- task_number: 14
  name: "Cache Layer with CLI Integration"
  files:
    - "packages/core/src/cache/index.ts"
    - "packages/core/src/cache/file-cache.ts"
    - "packages/cli/src/program.ts"           # Added
  depends_on: [1, 3, 10]                       # Added Task 10 dependency

  success_criteria:
    - "CacheManager accepts enabled option"
    - "CLI --no-cache flag sets enabled=false"  # Now achievable
```

**Warning:** This can lead to large, unfocused tasks. Prefer Strategy 1 or 2.

## Behavioral vs Implementation-Specific

**Critical Rule:** Success criteria describe WHAT, not HOW.

**BAD - Implementation-specific:**
```yaml
- "RenderTier enum exported with 5 levels (RICH=4, FULL=3, STANDARD=2, COMPAT=1, PLAIN=0)"
- "RGB interface exported with readonly number fields"
- "tempToColor returns chalk.rgb() for true-color terminals"
```

**GOOD - Behavioral:**
```yaml
- "RenderTier enum has 5 distinct capability tiers"
- "RGB interface has r, g, b number properties"
- "tempToColor returns function that applies color styling to text"
```

The implementation section tells the agent HOW. Success criteria tell QC WHAT to verify. If criteria duplicate implementation details, any reasonable deviation fails QC.

**Test:** Can the agent achieve this criterion with a different but equally valid implementation? If no, it's too specific.

## Validation Checklist

For each non-integration task, verify:

- [ ] Every success criterion is testable with ONLY the task's files
- [ ] No criteria reference CLI flags, UI elements, or other packages
- [ ] No criteria use integration keywords (see table above)
- [ ] All cross-component behavior is in integration tasks
- [ ] No criteria specify implementation details (enum values, exact chars, specific libraries)

## Plan Header

Add this validation marker to plan header:

```yaml
# ═══════════════════════════════════════════════════════════════════════════
# SUCCESS CRITERIA CLASSIFICATION
# ═══════════════════════════════════════════════════════════════════════════
# All component tasks have CAPABILITY-only criteria ✓
# Integration criteria isolated to integration tasks ✓
# ═══════════════════════════════════════════════════════════════════════════
```

## Common Mistakes

### Mistake 1: CLI flags in library tasks
```yaml
# BAD - Library task with CLI criterion
- task: "Implement Logger"
  files: ["src/logger.ts"]
  success_criteria:
    - "Logs include timestamps"
    - "--verbose flag enables debug logs"  # ❌ CLI concern!
```

### Mistake 2: UI behavior in API tasks
```yaml
# BAD - API task with UI criterion
- task: "Weather API Client"
  files: ["src/api/weather.ts"]
  success_criteria:
    - "Fetches forecast data"
    - "Shows spinner during fetch"  # ❌ UI concern!
```

### Mistake 3: End-to-end flows in component tasks
```yaml
# BAD - Component task with E2E criterion
- task: "User Authentication"
  files: ["src/auth/jwt.ts"]
  success_criteria:
    - "Validates JWT tokens"
    - "User can login and see dashboard"  # ❌ Full flow!
```

## Related Phases

- **Phase 2.5 (Data Flow)** - Tracks function/type dependencies
- **Phase 2.7 (This phase)** - Tracks success criteria scope
- **Phase 3b (Integration Detection)** - Creates integration tasks when needed
