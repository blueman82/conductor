# Subagent Deployment Strategy
**Timeline**: 1 Day (24 hours)
**Target**: Complete all 43 tasks from implementation.md
**Strategy**: Maximum parallelization with overlapping deployment waves
**Generated**: 2025-11-06

---

## Executive Summary

To complete the implementation in 1 day, we deploy **117 specialized subagents** across 5 overlapping waves with a 20-agent red flag response team on continuous standby. Total deployment capacity: **137 concurrent subagents**.

**Key Metrics**:
- Total Tasks: 43
- Implementation Subagents: 59
- Review Subagents (quality-control): 58
- Standby Fix Team: 20
- Estimated Success Rate: 75% green on first pass, 25% requiring fixes

---

## Deployment Architecture

### Quality Assurance Workflow

```
┌─────────────────┐
│ Implementation  │
│    Subagent     │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ quality-control │ ◄─── Review work, check for:
│     Review      │      - Compilation errors
└────────┬────────┘      - Test failures
         │               - Warning messages
         │               - Code quality issues
         │
    ┌────┴────┐
    │         │
    ▼         ▼
┌───────┐ ┌───────┐
│ GREEN │ │  RED  │
│ PASS  │ │ FAIL  │
└───┬───┘ └───┬───┘
    │         │
    │         ▼
    │    ┌─────────────────┐
    │    │ Fix Subagent    │ ◄─── Same specialty as implementation
    │    │ (From Standby)  │
    │    └────────┬────────┘
    │             │
    │             ▼
    │    ┌─────────────────┐
    │    │ quality-control │ ◄─── Re-review
    │    │   Re-Review     │
    │    └────────┬────────┘
    │             │
    └─────────────┴─────────────┐
                                 ▼
                        ┌─────────────────┐
                        │ Update Plan Doc │
                        │ Mark Complete   │
                        └─────────────────┘
```

---

## Wave 1: Foundation Layer (Hour 0-4)
**Start Time**: Immediately
**Deploy**: 21 subagents in parallel

### Phase 0: Prerequisites
| Task | Implementation | Review | Specialty Required |
|------|----------------|--------|-------------------|
| Verify environment & create branch | 1x swiftdev | 1x quality-control | Swift, Git, macOS setup |

### Phase 1: Project Foundation (Tasks 1-3)
| Task | Implementation | Review | Specialty Required | Est. Time |
|------|----------------|--------|-------------------|-----------|
| Task 1: Swift Package Init | 1x swiftdev | 1x quality-control | Swift Package Manager, modular architecture | 30m |
| Task 2: Database Schema + Actor | 1x ios-developer | 1x quality-control | SQLite, FTS5, GRDB, Swift actors | 2h |
| Task 3: FSEvents Monitoring Actor | 1x swift-concurrency-expert | 1x quality-control | FSEvents C API, Swift concurrency, actors | 3h |

### Phase 2: Shadow Repo (Critical Path Start)
| Task | Implementation | Review | Specialty Required | Est. Time |
|------|----------------|--------|-------------------|-----------|
| Task 4: Shadow Repo Actor (Core) | 1x swiftdev | 1x quality-control | Git operations, Swift actors, isolated repos | 4h |

**Wave 1 Total**: 7 implementation + 7 review = **14 subagents**
**+ Preemptive standby**: 7 fix agents = **21 total**

---

## Wave 2: Core Services Layer (Hour 2-8)
**Start Time**: Hour 2 (overlaps Wave 1)
**Deploy**: 28 subagents in parallel

### Phase 2: Shadow Repository (Tasks 5-10)
| Task | Implementation | Review | Specialty Required | Est. Time |
|------|----------------|--------|-------------------|-----------|
| Task 5: Git commit creation logic | 1x swiftdev | 1x quality-control | Git libgit2/shell, process management | 2h |
| Task 6: File version tracking | 1x swiftdev | 1x quality-control | File I/O, content hashing, actor coordination | 2h |
| Task 7: Content deduplication integration | 1x database-optimizer | 1x quality-control | SHA-256, database optimization, BLOB storage | 2h |
| Task 8: Shadow repo coordinator | 1x swift-concurrency-expert | 1x quality-control | Actor orchestration, async/await patterns | 2h |
| Task 9: FSEvents → Shadow integration | 1x swiftdev | 1x quality-control | Event handling, debouncing, flow control | 2h |
| Task 10: Shadow commit metadata | 1x ios-developer | 1x quality-control | Data modeling, timestamp handling | 1.5h |

### Phase 3: Conversation Indexing (Tasks 11-16)
| Task | Implementation | Review | Specialty Required | Est. Time |
|------|----------------|--------|-------------------|-----------|
| Task 11: JSONL parser for transcripts | 1x ios-developer | 1x quality-control | JSON parsing, Swift Codable, file streaming | 2h |
| Task 12: Conversation extractor | 1x swiftdev | 1x quality-control | Data transformation, actor patterns | 2h |
| Task 13: Message extraction with tools | 1x ios-developer | 1x quality-control | JSON schema, complex parsing, tool detection | 2h |
| Task 14: Timestamp-based correlation | 1x swiftdev | 1x quality-control | Time window algorithms, scoring logic | 3h |
| Task 15: File overlap scoring | 1x swiftdev | 1x quality-control | Set operations, Jaccard similarity, scoring | 2.5h |
| Task 16: Indexing coordinator actor | 1x swift-concurrency-expert | 1x quality-control | Background processing, progress tracking | 2h |

### Database Optimization (Parallel)
| Task | Implementation | Review | Specialty Required | Est. Time |
|------|----------------|--------|-------------------|-----------|
| Optimize FTS5 indexes | 1x database-optimizer | 1x quality-control | SQLite performance tuning, index strategies | 1h |
| Query performance tuning | 1x database-optimizer | 1x quality-control | Query optimization, EXPLAIN QUERY PLAN | 1h |

**Wave 2 Total**: 14 implementation + 14 review = **28 subagents**

---

## Wave 3: Application Layer (Hour 4-12)
**Start Time**: Hour 4 (overlaps Wave 2)
**Deploy**: 32 subagents in parallel

### Phase 4: SwiftUI Application (Tasks 17-26)
| Task | Implementation | Review | Specialty Required | Est. Time |
|------|----------------|--------|-------------------|-----------|
| Task 17: Main app structure + App.swift | 1x ios-developer | 1x quality-control | SwiftUI App lifecycle, @MainActor | 1h |
| Task 18: Dashboard view | 1x ios-developer | 1x quality-control | SwiftUI views, metrics display, charts | 2h |
| Task 19: Search interface with FTS5 | 1x ios-developer | 1x quality-control | SwiftUI search, async data loading | 2h |
| Task 20: Project list view | 1x ios-developer | 1x quality-control | SwiftUI List, navigation, data binding | 1.5h |
| Task 21: Conversation detail view | 1x ios-developer | 1x quality-control | Detail views, message rendering | 2h |
| Task 22: Timeline visualization | 1x ios-developer | 1x quality-control | Custom SwiftUI shapes, timeline UI | 3h |
| Task 23: Settings view | 1x ios-developer | 1x quality-control | Forms, preferences, monitoring toggles | 1.5h |
| Task 24: Dashboard ViewModel | 1x ios-developer | 1x quality-control | MVVM pattern, @Observable, actor integration | 1.5h |
| Task 25: Search ViewModel | 1x ios-developer | 1x quality-control | Search state management, debouncing | 1.5h |
| Task 26: Settings ViewModel | 1x ios-developer | 1x quality-control | Preferences management, persistence | 1h |

### Phase 5: MCP Server (Tasks 27-34)
| Task | Implementation | Review | Specialty Required | Est. Time |
|------|----------------|--------|-------------------|-----------|
| Task 27: MCP server structure (Python) | 1x python-backend-tdd-agent | 1x quality-control | MCP protocol, Python async, FastMCP | 2h |
| Task 28: Search tool implementation | 1x python-backend-tdd-agent | 1x quality-control | FTS5 queries, result ranking, Python | 2h |
| Task 29: Restore tool implementation | 1x python-backend-tdd-agent | 1x quality-control | Git operations, safety checks, Python | 3h |
| Task 30: Query tool (metadata) | 1x python-backend-tdd-agent | 1x quality-control | SQL queries, data aggregation | 1.5h |
| Task 31: Database connection layer | 1x python-backend-tdd-agent | 1x quality-control | SQLite connections, connection pooling | 1.5h |
| Task 32: MCP protocol integration | 1x mcp-protocol-expert | 1x quality-control | MCP spec, tool definitions, JSON-RPC | 2h |
| Task 33: Pydantic schemas | 1x python-schema-architect | 1x quality-control | Data validation, schema design | 1.5h |
| Task 34: MCP server configuration | 1x python-integration-specialist | 1x quality-control | Config files, Claude.app integration | 1h |

**Wave 3 Total**: 18 implementation + 18 review = **36 subagents**
**Note**: Reduced to 32 after optimization (shared reviews)

---

## Wave 4: Testing & Integration (Hour 8-16)
**Start Time**: Hour 8 (overlaps Wave 3)
**Deploy**: 24 subagents in parallel

### Phase 6: Testing & Safety (Tasks 35-42)
| Task | Implementation | Review | Specialty Required | Est. Time |
|------|----------------|--------|-------------------|-----------|
| Task 35: Unit test suite completion | 1x test-automator | 1x quality-control | XCTest, Swift testing, TDD validation | 2h |
| Task 36: Integration tests (Swift) | 1x test-automator | 1x quality-control | End-to-end testing, FSEvents → DB | 2h |
| Task 37: MCP server tests (Python) | 1x python-backend-tdd-agent | 1x quality-control | Pytest, async testing, mocking | 2h |
| Task 38: Restore safety validation | 1x swiftdev | 1x quality-control | Git state checks, conflict detection | 2h |
| Task 39: Performance tests | 1x performance-engineer | 1x quality-control | Benchmarking, profiling, optimization | 2h |
| Task 40: UI responsiveness tests | 1x ios-developer | 1x quality-control | SwiftUI testing, UI performance | 1.5h |
| Task 41: Error handling audit | 1x error-detective | 1x quality-control | Error paths, exception handling | 1.5h |
| Task 42: Security audit | 1x quality-control | 1x quality-control | File permissions, data safety, sandboxing | 1.5h |

### Integration Fix Tasks (Parallel)
| Task | Implementation | Review | Specialty Required | Est. Time |
|------|----------------|--------|-------------------|-----------|
| Fix Wave 1 integration issues | 2x swiftdev | 2x quality-control | Debugging, cross-module integration | 2h |
| Fix Wave 2 integration issues | 2x swiftdev | 2x quality-control | Actor communication, data flow | 2h |

**Wave 4 Total**: 12 implementation + 12 review = **24 subagents**

---

## Wave 5: Final Validation (Hour 12-24)
**Start Time**: Hour 12
**Deploy**: 12 subagents in parallel

### Final Tasks (Tasks 43-46)
| Task | Implementation | Review | Specialty Required | Est. Time |
|------|----------------|--------|-------------------|-----------|
| Task 43: End-to-end workflow test | 1x test-automator | 1x quality-control | Full system testing, real scenarios | 3h |
| Task 44: Production readiness review | 1x production-readiness-validator | 1x quality-control | Deployment checklist, scalability | 2h |
| Task 45: Performance validation | 1x performance-engineer | 1x quality-control | Load testing, benchmarks vs targets | 2h |
| Task 46: Documentation generation | 1x technical-documentation-specialist | 1x quality-control | DocC, README, API docs | 2h |
| Task 47: Build & package for distribution | 1x swiftdev | 1x quality-control | Xcode build, signing, notarization | 2h |
| Task 48: Final integration smoke test | 1x quality-control | 1x quality-control | Full stack validation | 1h |

**Wave 5 Total**: 6 implementation + 6 review = **12 subagents**

---

## Continuous Red Flag Response Team
**Active**: Hours 0-24 (entire duration)
**Deploy**: 20 subagents on standby rotation

### Standby Specialists (Always Ready)
| Specialty | Count | Purpose |
|-----------|-------|---------|
| swift-build-master | 5 | Fix Swift compilation errors, orchestrate fixes |
| swiftdev | 5 | General Swift fixes, actor issues, logic bugs |
| python-backend-tdd-agent | 3 | Python MCP server fixes, test failures |
| error-detective | 3 | Root cause analysis, log investigation |
| quality-control | 4 | Emergency reviews, re-validation |

### Red Flag Response Protocol

1. **Detection** (quality-control review identifies issue)
   - Compilation errors
   - Test failures
   - Warnings
   - Performance regressions
   - Security issues

2. **Assignment** (automatic from standby pool)
   - Match specialty to issue type
   - Assign from available standby agent
   - Brief agent on context and error details

3. **Fix Execution** (standby agent activates)
   - Read relevant files
   - Apply fix
   - Run tests locally
   - Update implementation

4. **Re-Review** (quality-control validates fix)
   - GREEN: Mark complete, release standby agent
   - RED: Escalate to swift-build-master orchestrator

5. **Plan Update** (fix agent responsibility)
   - Update docs/plans/implementation.md
   - Mark task as completed
   - Document any deviations or learnings

---

## Timeout & Failure Handling

### Timeout Policy
- **Task timeout**: 2x estimated time (e.g., 2h task → 4h timeout)
- **Review timeout**: 30 minutes
- **Fix timeout**: 1.5x original task time

### Failure Response
```
Timeout/Failure Detected
         │
         ▼
┌─────────────────────┐
│ Kill existing agent │
│ Preserve context    │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│ Deploy replacement  │
│ from standby pool   │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│ Resume with context │
│ + error analysis    │
└─────────────────────┘
```

### Escalation Path
1. **Level 1**: Original specialist (1st attempt)
2. **Level 2**: Standby specialist (2nd attempt)
3. **Level 3**: swift-build-master orchestrator (coordinates multiple specialists)
4. **Level 4**: Human intervention (if 3 failures)

---

## Plan Update Protocol

### After Each Task Completion
Every subagent must update `docs/plans/implementation.md`:

1. **Mark task as complete**:
   ```markdown
   - [x] Task 1: Initialize Swift Package Structure ✅
   ```

2. **Update progress metrics**:
   ```markdown
   **Completed**: 5/43 tasks (11.6%)
   ```

3. **Document any changes**:
   - Deviations from spec
   - Additional tasks discovered
   - Time actuals vs estimates

4. **Commit update**:
   ```bash
   git add docs/plans/implementation.md
   git commit -m "docs: mark Task X complete"
   ```

---

## Success Metrics

### Definition of GREEN (Pass)
- ✅ All tests pass (no failures, no skips)
- ✅ No compilation errors
- ✅ No warnings (Swift: 0 warnings, Python: pylint score 10/10)
- ✅ Code coverage ≥ 80% for new code
- ✅ Performance benchmarks met (if applicable)
- ✅ DocC comments on all public APIs

### Definition of RED (Fail)
- ❌ Any test failures
- ❌ Compilation errors
- ❌ Warnings present
- ❌ Code coverage < 80%
- ❌ Performance regressions
- ❌ Missing documentation

### Phase Completion Criteria
Each phase considered complete when:
1. All tasks in phase marked ✅ GREEN
2. Integration between tasks validated
3. Phase documentation updated
4. Git commits pushed

---

## Timeline Visualization

```
Hour 0 ═══════════════════════════ Wave 1: Foundation (14+7 agents)
     │
Hour 2 ═══════════════════════════ Wave 2: Core Services (28 agents)
     │       │
Hour 4 ═══════════════════════════ Wave 3: Application (32 agents)
     │       │       │
Hour 8 ═══════════════════════════ Wave 4: Testing (24 agents)
     │       │       │       │
Hour 12 ════════════════════════ Wave 5: Validation (12 agents)
     │       │       │       │       │
Hour 24 ════════════════════════ Complete ✅

═══════════════════════════════ Red Flag Team (20 agents, continuous)
```

**Peak Concurrency**: Hour 8-12 (all waves active)
- Wave 2: ~10 agents still active
- Wave 3: ~25 agents active
- Wave 4: 24 agents active
- Standby: 20 agents
- **Total: ~79 concurrent agents**

---

## Resource Requirements

### Compute Resources
- **CPU**: 137 agent processes (assume 2 cores per agent = 274 cores)
- **Memory**: ~500MB per agent = 68.5GB RAM
- **Storage**: ~100GB for repos, databases, build artifacts
- **Network**: Minimal (local file operations)

### Human Oversight
- **Hour 0**: Initiate Wave 1
- **Hour 2**: Monitor Wave 1 completion, initiate Wave 2
- **Hour 4**: Monitor progress, initiate Wave 3
- **Hour 8**: Review integration issues, initiate Wave 4
- **Hour 12**: Initiate final validation
- **Hour 20**: Review final results
- **Hour 24**: Sign-off

---

## Risk Mitigation

### High-Risk Tasks (Likely to RED flag)
1. **Task 3** (FSEvents): C API complexity, memory management
2. **Task 4** (Shadow Repo): Git operations, isolation complexity
3. **Task 14** (Correlation): Algorithm complexity, scoring tuning
4. **Task 22** (Timeline UI): Custom rendering, performance
5. **Task 29** (Restore): Safety-critical, git state management

**Mitigation**: Allocate 2x standby agents for these tasks preemptively

### Dependency Risks
- **Database schema** (Task 2) blocks many downstream tasks
  - Mitigation: Priority execution, 2x specialist (ios-developer + database-optimizer)
- **FSEvents** (Task 3) blocks shadow repo integration
  - Mitigation: Parallel development with mocks, integrate later

### Integration Risks
- Swift ↔ Python communication (MCP server)
  - Mitigation: Integration tests in Wave 4 with dedicated fix team
- FSEvents → Database flow
  - Mitigation: End-to-end tests in Wave 4

---

## Subagent Specializations Summary

### Swift Specialists (59 total)
- **swiftdev**: 18 agents (general Swift, actors, logic)
- **ios-developer**: 15 agents (SwiftUI, views, MVVM)
- **swift-concurrency-expert**: 4 agents (FSEvents, actors, async/await)
- **swift-build-master**: 5 agents (compilation fixes, orchestration)
- **swift-syntax-cleaner**: 2 agents (standby, syntax errors)
- **swift-method-fixer**: 2 agents (standby, method errors)
- **swift-type-resolver**: 2 agents (standby, type errors)
- **swift-import-specialist**: 1 agent (standby, import errors)

### Python Specialists (14 total)
- **python-backend-tdd-agent**: 8 agents (MCP server, TDD)
- **python-schema-architect**: 1 agent (Pydantic schemas)
- **python-integration-specialist**: 1 agent (MCP config)
- **mcp-protocol-expert**: 1 agent (MCP protocol)

### Cross-Functional Specialists (24 total)
- **quality-control**: 58 agents (reviews, validation) - *shared across all waves*
- **database-optimizer**: 2 agents (SQLite, FTS5)
- **test-automator**: 3 agents (testing)
- **performance-engineer**: 2 agents (benchmarks)
- **error-detective**: 3 agents (debugging)
- **production-readiness-validator**: 1 agent (final validation)
- **technical-documentation-specialist**: 1 agent (docs)

---

## Execution Command

To initiate full deployment:

```bash
# Deploy all waves in parallel with monitoring
claude-code parallel-deploy \
  --plan docs/plans/implementation.md \
  --strategy docs/plans/subagent-deployment-strategy.md \
  --wave-1-start now \
  --wave-2-start "+2h" \
  --wave-3-start "+4h" \
  --wave-4-start "+8h" \
  --wave-5-start "+12h" \
  --standby-team continuous \
  --timeout-policy 2x \
  --max-retries 3 \
  --update-plan-after-task \
  --report-interval 1h \
  --target-completion 24h
```

---

## Monitoring Dashboard

### Real-Time Metrics
- Active agents by wave
- Tasks completed (X/43)
- GREEN vs RED ratio
- Standby team utilization
- Current blockers
- Estimated completion time

### Alerts
- Task timeout (>2x estimate)
- 3x RED flags on same task (escalate)
- Wave completion milestone
- Integration test failures
- Performance regression detected

---

## Conclusion

This aggressive 1-day deployment strategy requires:
- **137 specialized subagents** (59 implementation, 58 review, 20 standby)
- **5 overlapping waves** with peak concurrency of ~79 agents
- **Continuous monitoring** and rapid red flag response
- **Strict quality gates** (GREEN/RED validation after every task)
- **Automated plan updates** after each task completion

**Success depends on**:
1. Proper subagent specialization matching
2. Rapid red flag response (<15min fix cycle)
3. Parallel execution without blocking dependencies
4. Continuous integration testing starting Hour 8
5. Human oversight at key milestones

**Expected outcome**: Functional system with 80%+ code coverage, passing all tests, ready for production deployment by Hour 24.

---

**Document Status**: Ready for Execution
**Last Updated**: 2025-11-06
**Next Action**: Initiate Wave 1 deployment
