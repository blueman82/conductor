#!/bin/bash
# Demo: conductor run integration-plan.md --verbose
# Demonstrates v2.5+ integration tasks with dual criteria validation
# Shows component tasks vs integration tasks with success_criteria and integration_criteria

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
BOLD='\033[1m'
NC='\033[0m'

timestamp() {
    date +"%H:%M:%S"
}

echo -e "${BOLD}$ conductor run integration-plan.md --verbose${NC}"
echo ""
sleep 0.5

echo "Loading plan from integration-plan.md..."
sleep 0.3
echo "Validating dependencies..."
sleep 0.3

echo ""
echo "Plan Summary:"
echo "  Total tasks: 3"
echo "  Execution waves: 2"
echo "  Timeout: 10h0m0s"
echo "  Max concurrency: 3"
echo ""
sleep 0.5

echo "Starting execution..."
echo ""

# Wave 1 - Component tasks
TS=$(timestamp)
echo -e "[${TS}] Starting ${BOLD}Wave 1${NC}: 2 component tasks"
sleep 0.3

# Task 1: Router Component
TS=$(timestamp)
echo -e "[${TS}] ⏳ IN PROGRESS ${CYAN}[1/2]${NC} Task 1 (Router Implementation) ${MAGENTA}[COMPONENT]${NC} ${MAGENTA}(agent: golang-pro)${NC}"
sleep 1.5

TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Selected agents: ${MAGENTA}[code-reviewer]${NC} (mode: auto)"
sleep 0.3

TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Validating success_criteria..."
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}]   [PASS] Criterion: Routes defined with correct HTTP methods"
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}]   [PASS] Criterion: All endpoints return correct response structure"
sleep 0.2

TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Final verdict: ${GREEN}GREEN${NC} (strictest-wins)"
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}✓${NC} Task 1 (Router Implementation): ${GREEN}GREEN${NC} (2.8s, agent: golang-pro, 3 files)"

sleep 0.3

# Task 2: Auth Middleware Component
TS=$(timestamp)
echo -e "[${TS}] ⏳ IN PROGRESS ${CYAN}[2/2]${NC} Task 2 (Auth Middleware) ${MAGENTA}[COMPONENT]${NC} ${MAGENTA}(agent: security-auditor)${NC}"
sleep 1.8

TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Selected agents: ${MAGENTA}[code-reviewer, security-auditor]${NC} (mode: intelligent)"
sleep 0.3

TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Validating success_criteria..."
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}]   [PASS] Criterion: Middleware validates JWT tokens correctly"
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}]   [PASS] Criterion: Invalid tokens are rejected with 401"
sleep 0.2

TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Final verdict: ${GREEN}GREEN${NC} (strictest-wins)"
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}✓${NC} Task 2 (Auth Middleware): ${GREEN}GREEN${NC} (3.2s, agent: security-auditor, 4 files)"

sleep 0.3
TS=$(timestamp)
echo -e "[${TS}] ${BOLD}Wave 1${NC} ${GREEN}complete${NC} (6.0s) - 2/2 completed (${GREEN}2 GREEN${NC})"
echo ""

# Wave 2 - Integration task
sleep 0.5
TS=$(timestamp)
echo -e "[${TS}] Starting ${BOLD}Wave 2${NC}: 1 integration task"
sleep 0.3

# Task 3: Wire components together - INTEGRATION TASK
TS=$(timestamp)
echo -e "[${TS}] ⏳ IN PROGRESS ${CYAN}[1/1]${NC} Task 3 (Wire Auth to Router) ${MAGENTA}[INTEGRATION]${NC} ${MAGENTA}(agent: golang-pro)${NC}"
echo -e "[${TS}]   Dependencies: Task 1 (Router), Task 2 (Auth Middleware)"
sleep 2.0

TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Selected agents: ${MAGENTA}[code-reviewer, golang-pro]${NC} (mode: intelligent)"
sleep 0.3

TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Validating success_criteria (component-level)..."
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}]   [PASS] Criterion: Auth middleware integrated into router"
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}]   [PASS] Criterion: Protected endpoints require authentication"
sleep 0.2

echo ""

TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Validating integration_criteria (cross-component)..."
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}]   [PASS] Criterion: Auth middleware executes before request handlers"
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}]   [PASS] Criterion: Valid tokens are validated end-to-end through both components"
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}]   [PASS] Criterion: Authentication errors propagate correctly from middleware to router"
sleep 0.2

TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Final verdict: ${GREEN}GREEN${NC} (all criteria unanimous)"
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}✓${NC} Task 3 (Wire Auth to Router): ${GREEN}GREEN${NC} (4.1s, agent: golang-pro, 5 files)"

sleep 0.3
TS=$(timestamp)
echo -e "[${TS}] ${BOLD}Wave 2${NC} ${GREEN}complete${NC} (6.1s) - 1/1 completed (${GREEN}1 GREEN${NC})"
echo ""

# Summary
sleep 0.5
TS=$(timestamp)
echo -e "[${TS}] ${BOLD}=== Execution Summary ===${NC}"
TS=$(timestamp)
echo -e "[${TS}] Total tasks: 3 (2 component, 1 integration)"
TS=$(timestamp)
echo -e "[${TS}] ${GREEN}Completed: 3${NC}"
TS=$(timestamp)
echo -e "[${TS}] Failed: 0"
TS=$(timestamp)
echo -e "[${TS}] Duration: 12.1s"
TS=$(timestamp)
echo -e "[${TS}] ${BOLD}Status Breakdown:${NC}"
TS=$(timestamp)
echo -e "[${TS}]   ${GREEN}GREEN${NC}: 3"
TS=$(timestamp)
echo -e "[${TS}] ${BOLD}Criteria Results:${NC}"
TS=$(timestamp)
echo -e "[${TS}]   Task 1 - success_criteria: 2/2 PASS"
TS=$(timestamp)
echo -e "[${TS}]   Task 2 - success_criteria: 2/2 PASS"
TS=$(timestamp)
echo -e "[${TS}]   Task 3 - success_criteria: 2/2 PASS"
TS=$(timestamp)
echo -e "[${TS}]   Task 3 - integration_criteria: 3/3 PASS"
TS=$(timestamp)
echo -e "[${TS}] ${BOLD}Agent Usage:${NC}"
TS=$(timestamp)
echo -e "[${TS}]   ${CYAN}golang-pro${NC}: 2 tasks"
TS=$(timestamp)
echo -e "[${TS}]   ${CYAN}security-auditor${NC}: 1 task"
TS=$(timestamp)
echo -e "[${TS}] Files Modified: ${GREEN}12${NC} files"
TS=$(timestamp)
echo -e "[${TS}] Average Duration: 4.0s/task"
echo ""
echo "Execution completed successfully!"
echo "Logs written to: .conductor/logs"
