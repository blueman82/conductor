#!/bin/bash
# Demo: conductor run plan.md --verbose (with failures)
# Simulates execution with failed tasks

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
BOLD='\033[1m'
NC='\033[0m'

timestamp() {
    date +"%H:%M:%S"
}

echo -e "${BOLD}$ conductor run complex-plan.md --verbose${NC}"
echo ""
sleep 0.5

echo -e "${CYAN}Loading plan:${NC} complex-plan.md"
sleep 0.3
echo "Validating dependencies..."
sleep 0.3

echo ""
echo "Plan Summary:"
echo "  Total tasks: 4"
echo "  Execution waves: 2"
echo "  Timeout: 10h0m0s"
echo ""
sleep 0.5

echo -e "Starting execution..."
echo ""

# Wave 1
TS=$(timestamp)
echo -e "[${TS}] Starting ${BOLD}Wave 1${NC}: 2 tasks"
sleep 0.3

TS=$(timestamp)
echo -e "[${TS}] ⏳ IN PROGRESS ${CYAN}[1/2]${NC} Task 1 (Setup) ${MAGENTA}(agent: golang-pro)${NC}"
sleep 1.5
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}✓${NC} Task 1 (Setup): ${GREEN}GREEN${NC} (2.1s)"

sleep 0.3
TS=$(timestamp)
echo -e "[${TS}] ⏳ IN PROGRESS ${CYAN}[2/2]${NC} Task 2 (Database Migration) ${MAGENTA}(agent: database-optimizer)${NC}"
sleep 2

# Failed task with retries
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Final verdict: ${RED}RED${NC} (strictest-wins)"
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}✗${NC} Task 2 (Database Migration): ${RED}RED${NC} (3.2s)"
echo -e "[${TS}]   Feedback: Migration script has syntax error on line 45"

sleep 0.5
TS=$(timestamp)
echo -e "[${TS}] ${YELLOW}Retrying Task 2...${NC} (attempt 2/3)"
sleep 1.8
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Final verdict: ${RED}RED${NC} (strictest-wins)"
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}✗${NC} Task 2 (Database Migration): ${RED}RED${NC} (2.9s)"
echo -e "[${TS}]   Feedback: Foreign key constraint violation in users table"

sleep 0.5
TS=$(timestamp)
echo -e "[${TS}] ${YELLOW}Retrying Task 2...${NC} (attempt 3/3)"
sleep 2
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Final verdict: ${RED}RED${NC} (strictest-wins)"
TS=$(timestamp)
echo -e "[${TS}] ${RED}✗${NC} Task 2 (Database Migration): ${RED}RED${NC} (3.1s)"
echo -e "[${TS}]   Feedback: Max retries exceeded. Manual intervention required."

sleep 0.3
TS=$(timestamp)
echo -e "[${TS}] ${BOLD}Wave 1${NC} ${GREEN}complete${NC} (14.1s) - 2/2 completed (${GREEN}1 GREEN${NC}, ${RED}1 RED${NC})"
echo ""

# Wave 2 - blocked by failure
sleep 0.5
TS=$(timestamp)
echo -e "[${TS}] ${RED}ERROR:${NC} Task 3 blocked - dependency Task 2 failed"
echo -e "[${TS}] ${RED}ERROR:${NC} Task 4 blocked - dependency Task 2 failed"
echo ""

# Summary
sleep 0.5
TS=$(timestamp)
echo -e "[${TS}] ${BOLD}=== Execution Summary ===${NC}"
echo -e "[${TS}] Total tasks: 4"
echo -e "[${TS}] ${GREEN}Completed: 1${NC}"
echo -e "[${TS}] ${RED}Failed: 1${NC}"
echo -e "[${TS}] Blocked: 2"
echo -e "[${TS}] Duration: 14.1s"
echo -e "[${TS}] ${BOLD}Status Breakdown:${NC}"
echo -e "[${TS}]   ${GREEN}GREEN${NC}: 1"
echo -e "[${TS}]   ${RED}RED${NC}: 1"
echo -e "[${TS}] ${RED}Failed tasks:${NC}"
echo -e "[${TS}]   - ${RED}Task 2 (Database Migration)${NC}: RED"
echo -e "[${TS}]     Feedback: Max retries exceeded. Manual intervention required."
echo ""
echo -e "${RED}Execution completed with 1 failed task(s).${NC}"
echo "Logs written to: .conductor/logs"
exit 1
