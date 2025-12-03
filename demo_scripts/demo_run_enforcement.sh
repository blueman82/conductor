#!/bin/bash
# Demo: conductor run enforcement-plan.yaml --verbose
# Shows v2.9+ runtime enforcement: dependency checks (silent on pass) and test commands
#
# Reality: Dependency checks run silently. Test commands execute and log in 🧪 Test Commands box.
# There is NO [Enforcement] or [Runtime] prefix in actual output - these are added by imagination.

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

echo -e "${BOLD}$ conductor run enforcement-plan.yaml --verbose${NC}"
echo ""
sleep 0.5

echo -e "${CYAN}Loading plan:${NC} enforcement-plan.yaml"
sleep 0.3

echo ""
echo "Plan Summary:"
echo "  Total tasks: 2"
echo "  Execution waves: 2"
echo "  Max concurrency: 2"
echo ""
sleep 0.5

echo "Starting execution..."
echo ""

# Wave 1
TS=$(timestamp)
echo -e "[${TS}] Starting Wave 1: 2 tasks"
sleep 0.3

TS=$(timestamp)
echo -e "[${TS}] ⏳ IN PROGRESS [1/2] Task 1 (Install Dependencies) (agent: python-pro)"
sleep 1.0

TS=$(timestamp)
echo -e "[${TS}] [INFO] Dependency checks: running (silent on pass)"
sleep 0.5

TS=$(timestamp)
echo -e "[${TS}] ✓ Task 1 (Install Dependencies): GREEN (2.3s, agent: python-pro, 1 files)"
sleep 0.3

# Dependency checks pass silently - no log output
# When checks FAIL, conductor logs error and stops task execution

echo ""
echo -e "┌────────────────────────────────────────────────────────────────────┐"
echo -e "│ 🧪 Test Commands                                                   │"
echo -e "├────────────────────────────────────────────────────────────────────┤"
echo -e "│ [1/2] python3 -m pip show requests                                 │"
echo -e "│       ✓ PASS (0.2s)                                                │"
echo -e "├────────────────────────────────────────────────────────────────────┤"
echo -e "│ [2/2] python3 -c \"import requests; print('OK')\"                    │"
echo -e "│       ✓ PASS (0.3s)                                                │"
echo -e "├────────────────────────────────────────────────────────────────────┤"
echo -e "│ Summary: All 2 passed ✓                                            │"
echo -e "└────────────────────────────────────────────────────────────────────┘"

echo ""
sleep 0.3

# Task 2
TS=$(timestamp)
echo -e "[${TS}] ⏳ IN PROGRESS [2/2] Task 2 (Create Server) (agent: python-pro)"
sleep 1.0

TS=$(timestamp)
echo -e "[${TS}] [INFO] Dependency checks: running (silent on pass)"
sleep 0.5

TS=$(timestamp)
echo -e "[${TS}] ✓ Task 2 (Create Server): GREEN (3.1s, agent: python-pro, 2 files)"
sleep 0.3

echo ""
echo -e "┌────────────────────────────────────────────────────────────────────┐"
echo -e "│ 🧪 Test Commands                                                   │"
echo -e "├────────────────────────────────────────────────────────────────────┤"
echo -e "│ [1/3] python3 -m py_compile server.py                              │"
echo -e "│       ✓ PASS (0.1s)                                                │"
echo -e "├────────────────────────────────────────────────────────────────────┤"
echo -e "│ [2/3] python3 server.py --check                                    │"
echo -e "│       ✓ PASS (0.2s)                                                │"
echo -e "├────────────────────────────────────────────────────────────────────┤"
echo -e "│ [3/3] grep -q 'def main' server.py                                 │"
echo -e "│       ✓ PASS (0.0s)                                                │"
echo -e "├────────────────────────────────────────────────────────────────────┤"
echo -e "│ Summary: All 3 passed ✓                                            │"
echo -e "└────────────────────────────────────────────────────────────────────┘"

echo ""
sleep 0.5

TS=$(timestamp)
echo -e "[${TS}] === Execution Summary ==="
echo "Total tasks: 2"
echo "Completed: 2"
echo "Failed: 0"
echo "Duration: 6.4s"
echo ""
echo "Status:"
echo "  GREEN: 2"
echo ""
echo "All tasks completed successfully ✓"
