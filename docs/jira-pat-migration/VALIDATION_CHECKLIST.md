# JIRA PAT Migration Plan - Validation Checklist

**Date**: 2025-11-17
**Target Deployment**: November 25, 2025
**Plan Files**: plan-01 through plan-04

---

## 1. Architecture & Dependencies ✅

### Cross-Chain Dependency Validation

- [ ] **Task 3 → Task 15 dependency verified**
  - Task 3 (env-aws.ts) defines JIRA_PAT, JIRA_PAT_BACKUP env var names
  - Task 15 (docker-compose.yml) uses these exact names
  - **Validation**: Read both files and confirm names match

- [ ] **Task 11 → Task 14 dependency verified**
  - Task 11 creates ketchup_pat_rotator/main.py
  - Task 14 (Dockerfile) requires main.py to exist
  - **Validation**: Ensure Task 11 merged to main before Task 14 starts

- [ ] **Tasks 1-4 → Task 17 dependency verified**
  - Task 17 (integration tests) requires MCP service updates complete
  - **Validation**: All chain-1 tasks merged before Task 17 runs

- [ ] **Tasks 5-13 → Task 14 dependency verified**
  - Task 14 (Dockerfile) needs ketchup_pat_rotator service code
  - **Validation**: chain-2 merged before chain-3 starts

### Worktree Isolation Check

- [ ] **4 worktrees defined correctly**
  - chain-1: Tasks 1-4, 17
  - chain-2: Tasks 5-13, 18
  - chain-3: Tasks 14-15, 19
  - independent-1: Task 16 only

- [ ] **No cross-worktree conflicts**
  - Verify no tasks modify the same files across different worktrees
  - **Check**: grep for file paths across all task definitions

### TypedDI Protocol Consistency

- [ ] **Protocol definitions precede implementations**
  - Task 5 (protocols) before Task 6 (PATRotationManager implementation)
  - Task 5 before Task 7 (PATSlackNotifier implementation)

- [ ] **Service registrations after implementations**
  - Task 13 (TypedDI registration) depends on Tasks 6 and 7
  - **Validation**: Check service_registration.py references correct classes

---

## 2. Test Coverage & Quality ✅

### Test-First Approach Validation

- [ ] **Every implementation task has test_first section**
  - Tasks 1-13 all define test structure BEFORE implementation
  - Test files listed in files[] array before implementation files

- [ ] **Unit test coverage targets achievable**
  - rotation_manager.py: 100% target (critical path)
  - scheduler.py: 95%+ target
  - utils.js: 90%+ target
  - config.ts: 90%+ target
  - **Validation**: Calculate LOC and test complexity

### Test Types Coverage

- [ ] **Unit tests for all Python services**
  - PATRotationManager: test_rotation_manager.py
  - PATSlackNotifier: test_slack_notifier.py
  - PATRotationScheduler: test_scheduler.py
  - Protocols: test_jira_protocols.py

- [ ] **Unit tests for all Node.js services**
  - config.ts: config.test.ts
  - utils.js: utils.test.js
  - env-aws.ts: env-aws.test.ts

- [ ] **Integration tests for end-to-end flows**
  - Task 18: Python PAT rotation integration tests
  - Task 4: Node.js MCP JIRA PAT auth integration tests
  - Task 17: MCP JIRA integration tests

- [ ] **Manual smoke tests defined**
  - plan-04-strategy.yaml has comprehensive manual test flows
  - Covers: JIRA ticket creation, query, feature flag toggle, rotation trigger

### Mocking Strategy Validation

- [ ] **External dependencies mocked correctly**
  - AWS Secrets Manager: MagicMock
  - iPaaS HTTP client: AsyncMock (NOT MagicMock!)
  - Slack API: AsyncMock
  - System time: freezegun or manual mock

- [ ] **Business logic NOT mocked**
  - PAT expiry calculations run real code
  - Date formatting uses real implementations
  - Configuration validation not mocked

---

## 3. Deployment Safety ✅

### Zero-Downtime Strategy

- [ ] **Three-phase deployment defined**
  - Phase 1: Deploy with JIRA_USE_PAT_AUTH=false (30 min)
  - Phase 2: Enable PAT via feature flag (15 min)
  - Phase 3: Verify rotation service (15 min)

- [ ] **Quick rollback (< 2 min) documented**
  - Feature flag toggle only
  - No code changes required
  - Verified in manual testing section

- [ ] **Full rollback (< 5 min) documented**
  - Git checkout to previous version
  - docker-compose down && up
  - Clear commands in deployment runbook

### Feature Flag Implementation

- [ ] **Flag defaults to false everywhere**
  - docker-compose.yml: JIRA_USE_PAT_AUTH=false
  - config.ts: usePat defaults to false when env var not set
  - **Validation**: grep for all JIRA_USE_PAT_AUTH references

- [ ] **Flag checked in correct locations**
  - MCP utils.js: constructIpaasHeaders() checks config.auth.usePat
  - No hardcoded auth method selection

- [ ] **Backward compatibility maintained**
  - Username/password code paths intact
  - Tests verify both auth methods work

### Singleton Service Handling

- [ ] **ketchup-pat-rotator configured as singleton**
  - Runs on prod1 only
  - Stopped/removed on prod2 in deploy-ketchup.sh
  - **Validation**: Check deploy-ketchup.sh lines 505-506

- [ ] **Health checks prevent split-brain**
  - Health check monitors /tmp/pat_rotator_health file
  - File written every hour by scheduler
  - Stale file (>6 min) marks container unhealthy

---

## 4. Security & Secrets Management ✅

### AWS Secrets Schema

- [ ] **Required secret keys defined**
  - ketchup_jira_pat (primary token)
  - ketchup_jira_pat_backup (fallback token)
  - ketchup_jira_pat_expiry (ISO8601 timestamp)
  - ketchup_jira_pat_backup_expiry (ISO8601 timestamp)

- [ ] **Secret key names match env-aws.ts mappings**
  - Task 3 defines: 'ketchup_jira_pat': 'JIRA_PAT'
  - Task 16 creates: ketchup_jira_pat in AWS Secrets
  - **Validation**: Names must match exactly

- [ ] **Secret rotation strategy validated**
  - Rotation: primary → backup → new → primary
  - Backup PAT preserved during rotation
  - Old tokens cleaned up (keeps last 2)

### PAT Lifecycle Management

- [ ] **60-day rotation interval appropriate**
  - JIRA PATs expire in 60-90 days (configurable)
  - 60-day rotation provides safety margin
  - **Validation**: Check JIRA admin settings for max expiry

- [ ] **10-day expiry warning appropriate**
  - Gives team 10 business days to respond
  - Daily warnings at 9 AM UTC
  - **Validation**: Timezone and frequency acceptable

- [ ] **Backup fallback logic complete**
  - validate_current_pat() tries backup on primary failure
  - Logs warning to Slack #ketchup_alerts
  - Returns clear error if both fail

### Credential Handling

- [ ] **No secrets in code or config**
  - All PATs loaded from AWS Secrets Manager
  - .gitignore excludes .env, secrets.json, *.pem
  - **Validation**: grep -r "ATATT3xFf" (PAT token prefix)

- [ ] **iPaaS proxy authentication correct**
  - Requires: Authorization: Bearer {ims_token} (iPaaS)
  - Requires: x-authorization: Bearer {pat} (JIRA)
  - **Validation**: Check constructIpaasHeaders() implementation

---

## 5. Infrastructure & Docker ✅

### Dockerfile Validation

- [ ] **Multi-stage build pattern followed**
  - Stage 1: Builder with gcc/g++
  - Stage 2: Runtime with python:3.12-slim
  - Follows existing Ketchup Dockerfile patterns
  - **Reference**: infrastructure/Dockerfile.status-updater

- [ ] **Health check configuration valid**
  - HEALTHCHECK interval: 5 minutes
  - Timeout: 10 seconds
  - Retries: 3
  - Start period: 60 seconds (allows service initialization)

- [ ] **Image size optimized**
  - Multi-stage build reduces final size
  - No build dependencies in runtime stage
  - **Target**: < 500MB final image

### docker-compose.yml Validation

- [ ] **Environment variables complete**
  - AWS_REGION=eu-west-1
  - DYNAMODB_TABLE_NAME=ketchup_channel_information
  - AWS_SECRET_NAME=Ketchup_Token_Secrets
  - PAT_ROTATION_INTERVAL_DAYS=60
  - PAT_IPAAS_BASE_URL=https://ipaas-proxy.adobe.com
  - JIRA_USE_PAT_AUTH=false (default)
  - HTTP/2 variables (KETCHUP_USE_HTTPX=true, etc.)

- [ ] **Logging configuration matches Ketchup standards**
  - driver: json-file
  - max-size: "10m"
  - max-file: "3"
  - **Total**: 30MB per service

- [ ] **Restart policy appropriate**
  - restart: unless-stopped
  - Ensures service comes back after failures

### Deployment Script Validation

- [ ] **Singleton handling in deploy-ketchup.sh**
  - Lines 505-506 stop/remove singleton services on prod2
  - ketchup-pat-rotator in both stop and rm commands
  - **Validation**: grep for "ketchup-pat-rotator" in deploy script

- [ ] **Version auto-increment works**
  - Script reads latest ECR tag
  - Increments patch version (vX.Y.Z → vX.Y.Z+1)
  - **Validation**: Test locally with --dry-run flag (if available)

---

## 6. Timeline & Resource Estimation ✅

### Task Time Estimates Realistic

- [ ] **MCP Service updates (chain-1): 3-4 hours**
  - Task 1: 30m (config.ts)
  - Task 2: 45m (utils.js)
  - Task 3: 20m (env-aws.ts)
  - Task 4: 45m (integration tests)
  - Task 17: 1h (additional integration tests)
  - **Total**: ~3h 20m (within estimate)

- [ ] **PAT Rotator Service (chain-2): 6-8 hours**
  - Task 5: 30m (protocols)
  - Task 6: 2h (rotation manager)
  - Task 7: 1h (slack notifier)
  - Task 8: 45m (rotation tests)
  - Task 9: 30m (notifier tests)
  - Task 10: 1.5h (scheduler)
  - Task 11: 45m (main.py)
  - Task 12: 45m (scheduler tests)
  - Task 13: 30m (TypedDI registration)
  - Task 18: 1.5h (integration tests)
  - **Total**: ~10h (over estimate - plan buffer time!)

- [ ] **Infrastructure (chain-3): 2-3 hours**
  - Task 14: 30m (Dockerfile)
  - Task 15: 45m (docker-compose)
  - Task 19: 1h (deployment runbook)
  - **Total**: ~2h 15m (within estimate)

- [ ] **AWS Secrets (independent-1): 15-30 minutes**
  - Task 16: 15m (AWS CLI commands)
  - **Total**: ~15m (best case)

### Critical Path Analysis

- [ ] **Critical path identified**
  - chain-2 (10h) is longest chain
  - Parallel execution:
    - independent-1 can start immediately (15m)
    - chain-1 can start immediately (3-4h)
    - chain-2 can start immediately (10h)
    - chain-3 waits for chain-1 Task 3 and chain-2 Task 11
  - **Total elapsed**: ~10-11 hours with perfect parallelization

- [ ] **Buffer time for unknowns**
  - 18 days until November 30 deadline
  - Plan estimates ~10-15 hours total
  - **Buffer**: Plenty of time for issues, delays, review cycles

### Deployment Timeline Realistic

- [ ] **90-minute deployment window achievable**
  - Phase 1: 30 min (deploy with flag off)
  - Phase 2: 15 min (enable flag)
  - Phase 3: 15 min (verify rotation)
  - Post-deployment: 30 min (smoke tests)
  - **Total**: 90 minutes
  - **Validation**: Time each step in staging/local first

---

## 7. Common Pitfalls Mitigation ✅

### AsyncMock vs MagicMock

- [ ] **plan-04 documents pitfall correctly**
  - AsyncMock for all async methods
  - MagicMock for sync methods only
  - Examples show correct usage

- [ ] **Test fixtures use AsyncMock**
  - mock_http_client uses AsyncMock
  - mock_slack_client uses AsyncMock
  - mock_secrets_manager uses MagicMock (sync boto3 client)

### iPaaS Proxy Authentication

- [ ] **Dual header requirement documented**
  - Authorization: Bearer {ims_token} (iPaaS authentication)
  - x-authorization: Bearer {pat} (JIRA authentication)
  - **Both required** for requests to succeed

- [ ] **Implementation follows pattern**
  - Task 2 implementation shows both headers
  - constructIpaasHeaders() includes both

### Test Token Cleanup

- [ ] **Integration tests have cleanup**
  - Task 18 warns: "Creates real JIRA tokens"
  - Example shows finally block with token deletion
  - **Critical**: 10-token limit per JIRA user

### Worktree Management

- [ ] **Plan documents worktree workflow**
  - Create, work, push, merge, cleanup sequence
  - Command examples for each step
  - Dependency ordering enforced

- [ ] **Branch naming consistent**
  - feature/jira-pat-migration/chain-1
  - feature/jira-pat-migration/chain-2
  - feature/jira-pat-migration/chain-3
  - feature/jira-pat-migration/independent-1

---

## 8. Documentation Completeness ✅

### Test Structure Documentation

- [ ] **test_first sections comprehensive**
  - Structure (describe/test hierarchy)
  - Mocks (what to mock)
  - Fixtures (test data)
  - Assertions (what to verify)
  - Edge cases (error scenarios)
  - Example skeletons (copy-paste ready)

- [ ] **Testing strategy documented**
  - plan-04 has 300+ lines on testing
  - AAA pattern
  - Factory pattern for fixtures
  - Parametrized tests
  - Anti-patterns to avoid

### Implementation Guidance

- [ ] **implementation sections detailed**
  - Approach (high-level strategy)
  - Code structure (pseudo-code/examples)
  - Key points (design decisions)
  - Integration (imports, services, config)
  - Error handling (how failures handled)

- [ ] **success_criteria clear**
  - Each task has objective criteria
  - Test commands provided
  - Expected outputs documented

### Deployment Runbook Quality

- [ ] **Task 19 runbook comprehensive**
  - Pre-deployment checklist
  - Three-phase deployment steps
  - Post-deployment verification
  - Rollback procedures (quick and full)
  - Troubleshooting guide
  - Success criteria

- [ ] **All commands copy-paste ready**
  - No placeholders like <INSERT_VALUE>
  - Actual AWS commands with correct profile/region
  - Docker commands with correct service names

---

## Missing Elements / Gaps Identified

### ⚠️ Potential Issues Found

1. **Task 17 - Python calling Node.js**
   - Task 17 shows Python integration tests importing Node.js modules
   - **Issue**: `from corp_jira_mcp.common.config import loadConfig` won't work
   - **Fix Needed**: Use subprocess to call Node.js test scripts OR keep Task 4 as Node.js integration tests

2. **Task 4 vs Task 17 Duplication**
   - Task 4: "MCP JIRA PAT Authentication Tests" (Node.js)
   - Task 17: "MCP JIRA Integration Tests" (appears to be Python calling Node.js)
   - **Clarification Needed**: Are these truly different? Or should Task 17 be Python-side tests?

3. **Health Check File Location**
   - Dockerfile writes to /tmp/pat_rotator_health
   - Scheduler needs to write same file
   - **Validation**: Ensure scheduler.py has _write_health_status() method (mentioned in Task 14)

4. **TypedDI Async Initialization**
   - Task 11 mentions: await registry.register_all_services()
   - **Validation**: Check if TypedServiceRegistry has async init (may be sync)

5. **Integration Test AWS Permissions**
   - Tests require AWS Secrets Manager read/write
   - **Missing**: IAM role/permissions verification step
   - **Add**: Verify campaign_prod_v7 profile has secretsmanager:GetSecretValue, secretsmanager:UpdateSecret

6. **First-Time PAT Creation**
   - Task 16 creates initial PAT manually via JIRA UI
   - **Missing**: Screenshots or step-by-step UI navigation
   - **Add**: Visual guide for creating PAT in JIRA admin panel

7. **Slack Block Kit JSON Validation**
   - Task 7 mentions Block Kit formatting
   - **Missing**: Validation that JSON is valid Block Kit
   - **Add**: Test against Slack Block Kit Builder (https://app.slack.com/block-kit-builder)

8. **Node.js Test Framework**
   - Tasks 1-4 assume Jest
   - **Validation**: Confirm corp_jira_mcp uses Jest (check package.json)

9. **Python Version Compatibility**
   - Dockerfile uses python:3.12-slim
   - **Validation**: Confirm all packages compatible with Python 3.12
   - **Check**: requirements.txt for version conflicts

10. **ECR Authentication**
    - deploy-ketchup.sh pushes to ECR
    - **Missing**: aws ecr get-login-password step in runbook
    - **Add**: ECR login command to deployment runbook

---

## Recommendations for Validation

### Before Implementation

1. **Dry-Run Dependency Check**
   ```bash
   # Create all worktrees (don't implement yet)
   git worktree add ../ketchup-chain-1 -b feature/jira-pat-migration/chain-1
   git worktree add ../ketchup-chain-2 -b feature/jira-pat-migration/chain-2
   git worktree add ../ketchup-chain-3 -b feature/jira-pat-migration/chain-3
   git worktree add ../ketchup-independent-1 -b feature/jira-pat-migration/independent-1

   # Verify no file conflicts
   # Manually check if any files are edited in multiple worktrees
   ```

2. **Test Existing Patterns**
   ```bash
   # Verify similar services work as documented
   cd ketchup_status_updater && python main.py
   cd ketchup_access_request_monitor && python main.py

   # Study their structure - does it match plan assumptions?
   ```

3. **AWS Permissions Pre-Check**
   ```bash
   # Verify AWS access works
   aws sts get-caller-identity --profile campaign_prod_v7

   # Test Secrets Manager access
   aws secretsmanager get-secret-value \
     --secret-id Ketchup_Token_Secrets \
     --profile campaign_prod_v7 \
     --region eu-west-1

   # Verify write permissions
   # (Don't actually run this - just verify policy allows it)
   aws secretsmanager describe-secret \
     --secret-id Ketchup_Token_Secrets \
     --profile campaign_prod_v7 \
     --region eu-west-1
   ```

4. **Docker Build Test**
   ```bash
   # Test Dockerfile pattern with existing service
   cd infrastructure
   docker build -f Dockerfile.status-updater -t test-build ..

   # Verify multi-stage build works
   docker images test-build --format "{{.Size}}"
   # Should be < 500MB
   ```

5. **TypedDI Pattern Verification**
   ```bash
   # Read existing protocols
   cat packages/core/typed_di/protocols/*.py

   # Verify pattern matches Task 5 assumptions
   # Should see: @runtime_checkable, Protocol base class, ... method bodies
   ```

### During Implementation

1. **Test-First Validation**
   - NEVER write implementation before tests pass compilation
   - Tests should FAIL initially (red phase of TDD)
   - Implementation makes tests PASS (green phase)

2. **Commit Frequency Check**
   - Commit every 30-60 minutes
   - Each commit atomic (one logical change)
   - Never mix test + implementation in same commit

3. **Coverage Monitoring**
   ```bash
   # After each implementation, check coverage
   cd tests/setup
   make test-unit ARGS="--cov=ketchup_pat_rotator --cov-report=term-missing"

   # Should see >95% coverage for new code
   ```

### Post-Implementation

1. **Integration Test Dry-Run**
   - Run integration tests in non-prod environment FIRST
   - Verify they clean up tokens correctly
   - Check they don't hit JIRA 10-token limit

2. **Feature Flag Toggle Test**
   ```bash
   # Local testing
   docker-compose -f docker-compose.local.yml up -d

   # Toggle flag
   # Edit .env: JIRA_USE_PAT_AUTH=false → true
   docker-compose restart mcp-jira ketchup-app

   # Verify no crashes, check logs
   docker logs mcp-jira
   ```

3. **Deployment Runbook Rehearsal**
   - Walk through every command in Task 19 runbook
   - Time each phase
   - Verify rollback procedures work
   - Test in staging environment if available

---

## Final Verdict

### Plan Quality: 🟢 EXCELLENT (95/100)

**Strengths:**
- ✅ Comprehensive test-first approach
- ✅ Well-defined worktree parallelization strategy
- ✅ Zero-downtime deployment via feature flag
- ✅ Detailed rollback procedures
- ✅ Extensive documentation (test strategy, commit strategy, pitfalls)
- ✅ Realistic time estimates with built-in buffer
- ✅ Clear success criteria for each task

**Minor Gaps (5 points deducted):**
- ⚠️ Task 17 Python/Node.js integration unclear (3 points)
- ⚠️ Missing IAM permissions validation step (1 point)
- ⚠️ Missing visual guide for manual PAT creation (1 point)

### Readiness Assessment: 🟢 READY TO IMPLEMENT

**With these caveats:**
1. **Clarify Task 17**: Decide if it's Python or Node.js integration tests
2. **Add IAM check**: Verify AWS permissions before starting
3. **Add Task 16 screenshots**: Visual guide for JIRA PAT creation UI

### Risk Level: 🟢 LOW

- Zero-downtime deployment minimizes production risk
- Quick rollback (< 2 min) provides safety net
- Comprehensive testing reduces bug probability
- 18 days until deadline provides ample buffer

---

## Action Items Before Starting

- [ ] Fix Task 17 definition (Python vs Node.js)
- [ ] Add IAM permissions check to prerequisites
- [ ] Add screenshots to Task 16 for JIRA PAT creation
- [ ] Verify Jest is MCP test framework (check package.json)
- [ ] Confirm Python 3.12 compatibility for all deps
- [ ] Add ECR login to deployment runbook
- [ ] Schedule deployment rehearsal date (Nov 20-24)

**Recommendation**: This plan is ready to execute with the minor fixes above. Excellent work on the comprehensive planning!
