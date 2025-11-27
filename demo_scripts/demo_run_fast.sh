#!/bin/bash
# Fast version of demo_run.sh for GIF recording
# Reduced sleep times for quicker playback

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
BOLD='\033[1m'
NC='\033[0m'

S=0.1  # Base sleep time (adjust for speed)

echo -e "${BOLD}$ conductor run plan.md --verbose${NC}"
echo ""
sleep $S

echo "Loading plan from plan.md..."
sleep $S
echo "Validating dependencies..."
sleep $S

echo ""
echo "Plan Summary:"
echo "  Total tasks: 6"
echo "  Execution waves: 3"
echo "  Timeout: 10h0m0s"
echo "  Max concurrency: 3"
echo ""
sleep $S

echo "Starting execution..."
echo ""

# Wave 1
echo -e "[14:23:45] Starting ${BOLD}Wave 1${NC}: 2 tasks"
sleep $S
echo -e "[14:23:45] ⏳ IN PROGRESS ${CYAN}[1/2]${NC} Task 1 (Initialize Project) ${MAGENTA}(agent: golang-pro)${NC}"
sleep 0.3
echo -e "[14:23:47] ${CYAN}✓${NC} Task 1 (Initialize Project): ${GREEN}GREEN${NC} (2.3s, agent: golang-pro, 3 files)"
sleep $S
echo -e "[14:23:47] ⏳ IN PROGRESS ${CYAN}[2/2]${NC} Task 2 (Setup Database) ${MAGENTA}(agent: database-optimizer)${NC}"
sleep 0.3
echo -e "[14:23:50] ${CYAN}✓${NC} Task 2 (Setup Database): ${GREEN}GREEN${NC} (3.1s, agent: database-optimizer, 2 files)"
sleep $S
echo -e "[14:23:50] ${BOLD}Wave 1${NC} ${GREEN}complete${NC} (5.4s) - 2/2 completed (${GREEN}2 GREEN${NC})"
echo ""

# Wave 2
sleep $S
echo -e "[14:23:50] Starting ${BOLD}Wave 2${NC}: 3 tasks"
sleep $S
echo -e "[14:23:50] ⏳ IN PROGRESS ${CYAN}[1/3]${NC} Task 3 (User Auth) ${MAGENTA}(agent: security-auditor)${NC}"
sleep 0.3
echo -e "[14:23:52] ${CYAN}[QC]${NC} Selected agents: ${MAGENTA}[code-reviewer, security-auditor]${NC} (mode: intelligent)"
sleep $S
echo -e "[14:23:52] ${CYAN}[QC]${NC} Final verdict: ${GREEN}GREEN${NC} (strictest-wins)"
echo -e "[14:23:52] ${CYAN}✓${NC} Task 3 (User Auth): ${GREEN}GREEN${NC} (4.2s, agent: security-auditor, 5 files)"
sleep $S
echo -e "[14:23:52] ⏳ IN PROGRESS ${CYAN}[2/3]${NC} Task 4 (API Endpoints) ${MAGENTA}(agent: golang-pro)${NC}"
sleep 0.3
echo -e "[14:23:54] ${CYAN}✓${NC} Task 4 (API Endpoints): ${GREEN}GREEN${NC} (2.8s, agent: golang-pro, 4 files)"
sleep $S
echo -e "[14:23:54] ⏳ IN PROGRESS ${CYAN}[3/3]${NC} Task 5 (Data Models) ${MAGENTA}(agent: golang-pro)${NC}"
sleep 0.3
echo -e "[14:23:56] ${CYAN}[QC]${NC} Selected agents: ${MAGENTA}[code-reviewer]${NC} (mode: auto)"
sleep $S
echo -e "[14:23:56] ${CYAN}[QC]${NC} Final verdict: ${YELLOW}YELLOW${NC} (strictest-wins)"
echo -e "[14:23:56] ${CYAN}⚠${NC} Task 5 (Data Models): ${YELLOW}YELLOW${NC} (3.5s, agent: golang-pro, 2 files)"
sleep $S
echo -e "[14:23:56] ${BOLD}Wave 2${NC} ${GREEN}complete${NC} (10.5s) - 3/3 completed (${GREEN}2 GREEN${NC}, ${YELLOW}1 YELLOW${NC})"
echo ""

# Wave 3 with retry
sleep $S
echo -e "[14:23:56] Starting ${BOLD}Wave 3${NC}: 1 tasks"
sleep $S
echo -e "[14:23:56] ⏳ IN PROGRESS ${CYAN}[1/1]${NC} Task 6 (Integration Tests) ${MAGENTA}(agent: golang-pro)${NC}"
sleep 0.3
echo -e "[14:23:58] ${CYAN}[QC]${NC} Selected agents: ${MAGENTA}[code-reviewer, golang-pro]${NC} (mode: intelligent)"
sleep $S
echo -e "[14:23:58] ${CYAN}[QC]${NC} Final verdict: ${RED}RED${NC} (strictest-wins)"
echo -e "[14:23:58] ${CYAN}✗${NC} Task 6 (Integration Tests): ${RED}RED${NC} (4.1s)"
echo -e "[14:23:58]   Feedback: Test coverage below 80% threshold."
sleep 0.2
echo -e "[14:23:58] ${YELLOW}Retrying Task 6...${NC} (attempt 2/3)"
sleep 0.3
echo -e "[14:24:00] ${CYAN}[QC]${NC} Final verdict: ${GREEN}GREEN${NC} (strictest-wins)"
echo -e "[14:24:00] ${CYAN}✓${NC} Task 6 (Integration Tests): ${GREEN}GREEN${NC} (3.8s, agent: golang-pro, 3 files)"
sleep $S
echo -e "[14:24:00] ${BOLD}Wave 3${NC} ${GREEN}complete${NC} (9.9s) - 1/1 completed (${GREEN}1 GREEN${NC})"
echo ""

# Summary
sleep $S
echo -e "[14:24:00] ${BOLD}=== Execution Summary ===${NC}"
echo -e "[14:24:00] Total tasks: 6"
echo -e "[14:24:00] ${GREEN}Completed: 6${NC}"
echo -e "[14:24:00] Failed: 0"
echo -e "[14:24:00] Duration: 25s"
echo -e "[14:24:00] ${BOLD}Status Breakdown:${NC}"
echo -e "[14:24:00]   ${GREEN}GREEN${NC}: 5"
echo -e "[14:24:00]   ${YELLOW}YELLOW${NC}: 1"
echo -e "[14:24:00] ${BOLD}Agent Usage:${NC}"
echo -e "[14:24:00]   ${CYAN}golang-pro${NC}: 4 tasks"
echo -e "[14:24:00]   ${CYAN}security-auditor${NC}: 1 tasks"
echo -e "[14:24:00]   ${CYAN}database-optimizer${NC}: 1 tasks"
echo -e "[14:24:00] Files Modified: ${GREEN}19${NC} files"
echo -e "[14:24:00] Average Duration: 4.3s/task"
echo ""
echo "Execution completed successfully!"
echo "Logs written to: .conductor/logs"
sleep 0.5
