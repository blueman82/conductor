#!/bin/bash
# Demo: conductor run plan.md --verbose
# Simulates full execution with QC reviews

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

echo -e "${BOLD}$ conductor run plan.md --verbose${NC}"
echo ""
sleep 0.5

echo "Loading plan from plan.md..."
sleep 0.3
echo "Validating dependencies..."
sleep 0.3

echo ""
echo "Plan Summary:"
echo "  Total tasks: 6"
echo "  Execution waves: 3"
echo "  Timeout: 10h0m0s"
echo "  Max concurrency: 3"
echo ""
sleep 0.5

echo "Starting execution..."
echo ""

# Wave 1
TS=$(timestamp)
echo -e "[${TS}] Starting ${BOLD}Wave 1${NC}: 2 tasks"
sleep 0.3

TS=$(timestamp)
echo -e "[${TS}] ⏳ IN PROGRESS ${CYAN}[1/2]${NC} Task 1 (Initialize Project) ${MAGENTA}(agent: golang-pro)${NC}"
sleep 1.5
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}✓${NC} Task 1 (Initialize Project): ${GREEN}GREEN${NC} (2.3s, agent: golang-pro, 3 files)"

sleep 0.3
TS=$(timestamp)
echo -e "[${TS}] ⏳ IN PROGRESS ${CYAN}[2/2]${NC} Task 2 (Setup Database) ${MAGENTA}(agent: database-optimizer)${NC}"
sleep 1.8
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}✓${NC} Task 2 (Setup Database): ${GREEN}GREEN${NC} (3.1s, agent: database-optimizer, 2 files)"

sleep 0.3
TS=$(timestamp)
echo -e "[${TS}] ${BOLD}Wave 1${NC} ${GREEN}complete${NC} (5.4s) - 2/2 completed (${GREEN}2 GREEN${NC})"
echo ""

# Wave 2
sleep 0.5
TS=$(timestamp)
echo -e "[${TS}] Starting ${BOLD}Wave 2${NC}: 3 tasks"
sleep 0.3

TS=$(timestamp)
echo -e "[${TS}] ⏳ IN PROGRESS ${CYAN}[1/3]${NC} Task 3 (User Auth) ${MAGENTA}(agent: security-auditor)${NC}"
sleep 2
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Selected agents: ${MAGENTA}[code-reviewer, security-auditor]${NC} (mode: intelligent)"
sleep 0.5
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Final verdict: ${GREEN}GREEN${NC} (strictest-wins)"
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}✓${NC} Task 3 (User Auth): ${GREEN}GREEN${NC} (4.2s, agent: security-auditor, 5 files)"

sleep 0.3
TS=$(timestamp)
echo -e "[${TS}] ⏳ IN PROGRESS ${CYAN}[2/3]${NC} Task 4 (API Endpoints) ${MAGENTA}(agent: golang-pro)${NC}"
sleep 1.5
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}✓${NC} Task 4 (API Endpoints): ${GREEN}GREEN${NC} (2.8s, agent: golang-pro, 4 files)"

sleep 0.3
TS=$(timestamp)
echo -e "[${TS}] ⏳ IN PROGRESS ${CYAN}[3/3]${NC} Task 5 (Data Models) ${MAGENTA}(agent: golang-pro)${NC}"
sleep 2.2

# Simulate a YELLOW verdict
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Selected agents: ${MAGENTA}[code-reviewer]${NC} (mode: auto)"
sleep 0.3
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Final verdict: ${YELLOW}YELLOW${NC} (strictest-wins)"
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}⚠${NC} Task 5 (Data Models): ${YELLOW}YELLOW${NC} (3.5s, agent: golang-pro, 2 files)"

sleep 0.3
TS=$(timestamp)
echo -e "[${TS}] ${BOLD}Wave 2${NC} ${GREEN}complete${NC} (10.5s) - 3/3 completed (${GREEN}2 GREEN${NC}, ${YELLOW}1 YELLOW${NC})"
echo ""

# Wave 3
sleep 0.5
TS=$(timestamp)
echo -e "[${TS}] Starting ${BOLD}Wave 3${NC}: 1 tasks"
sleep 0.3

TS=$(timestamp)
echo -e "[${TS}] ⏳ IN PROGRESS ${CYAN}[1/1]${NC} Task 6 (Integration Tests) ${MAGENTA}(agent: golang-pro)${NC}"
sleep 2.5

# Simulate RED verdict with retry
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Selected agents: ${MAGENTA}[code-reviewer, golang-pro]${NC} (mode: intelligent)"
sleep 0.3
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Final verdict: ${RED}RED${NC} (strictest-wins)"
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}✗${NC} Task 6 (Integration Tests): ${RED}RED${NC} (4.1s)"
echo -e "[${TS}]   Feedback: Test coverage below 80% threshold. Missing edge case tests."

sleep 0.5
TS=$(timestamp)
echo -e "[${TS}] ${YELLOW}Retrying Task 6...${NC} (attempt 2/3)"
sleep 2
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Final verdict: ${GREEN}GREEN${NC} (strictest-wins)"
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}✓${NC} Task 6 (Integration Tests): ${GREEN}GREEN${NC} (3.8s, agent: golang-pro, 3 files)"

sleep 0.3
TS=$(timestamp)
echo -e "[${TS}] ${BOLD}Wave 3${NC} ${GREEN}complete${NC} (9.9s) - 1/1 completed (${GREEN}1 GREEN${NC})"
echo ""

# Summary
sleep 0.5
TS=$(timestamp)
echo -e "[${TS}] ${BOLD}=== Execution Summary ===${NC}"
echo -e "[${TS}] Total tasks: 6"
echo -e "[${TS}] ${GREEN}Completed: 6${NC}"
echo -e "[${TS}] Failed: 0"
echo -e "[${TS}] Duration: 25s"
echo -e "[${TS}] ${BOLD}Status Breakdown:${NC}"
echo -e "[${TS}]   ${GREEN}GREEN${NC}: 5"
echo -e "[${TS}]   ${YELLOW}YELLOW${NC}: 1"
echo -e "[${TS}] ${BOLD}Agent Usage:${NC}"
echo -e "[${TS}]   ${CYAN}golang-pro${NC}: 4 tasks"
echo -e "[${TS}]   ${CYAN}security-auditor${NC}: 1 tasks"
echo -e "[${TS}]   ${CYAN}database-optimizer${NC}: 1 tasks"
echo -e "[${TS}] Files Modified: ${GREEN}19${NC} files"
echo -e "[${TS}] Average Duration: 4.3s/task"
echo ""
echo "Execution completed successfully!"
echo "Logs written to: .conductor/logs"
