#!/bin/bash
# Demo: conductor run integration-plan.md --verbose
# Demonstrates v2.5+ integration tasks with dual criteria validation
# Shows component tasks vs integration tasks with success_criteria and integration_criteria

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

echo -e "${BOLD}$ conductor run integration-plan.md --verbose${NC}"
echo ""

echo "Plan Summary:"
echo "  Total tasks: 3"
echo "  Task types: 2 component, 1 integration"
echo ""
sleep 0.5

echo "Starting execution..."
echo ""

# Wave 1: Component tasks
TS=$(timestamp)
echo -e "[${TS}] Starting Wave 1: 2 tasks"
sleep 0.3

# Task 1: Create API
TS=$(timestamp)
echo -e "[${TS}] ⏳ IN PROGRESS [1/2] Task 1 (Create API) (agent: golang-pro)"
sleep 1.0

TS=$(timestamp)
echo -e "[${TS}] ✓ Task 1 (Create API): GREEN (1.8s, agent: golang-pro, 2 files)"
sleep 0.3

# Task 2: Create Database
TS=$(timestamp)
echo -e "[${TS}] ⏳ IN PROGRESS [2/2] Task 2 (Create Database) (agent: database-optimizer)"
sleep 1.0

TS=$(timestamp)
echo -e "[${TS}] ✓ Task 2 (Create Database): GREEN (1.5s, agent: database-optimizer, 1 files)"
sleep 0.5

# Wave 2: Integration task
echo ""
TS=$(timestamp)
echo -e "[${TS}] Starting Wave 2: 1 tasks"
sleep 0.3

TS=$(timestamp)
echo -e "[${TS}] ⏳ IN PROGRESS [1/1] Task 3 (Integrate API + Database) (agent: backend-architect)"
sleep 1.5

TS=$(timestamp)
echo -e "[${TS}] [INFO] Integration context loaded from Task 1, Task 2"
sleep 0.3

TS=$(timestamp)
echo -e "[${TS}] ✓ Task 3 (Integrate API + Database): GREEN (2.3s, agent: backend-architect, 1 files)"

echo ""
echo -e "┌────────────────────────────────────────────────────────────────────┐"
echo -e "│ 🧪 Test Commands                                                   │"
echo -e "├────────────────────────────────────────────────────────────────────┤"
echo -e "│ [1/3] go test ./api/... -race                                      │"
echo -e "│       ✓ PASS (0.5s)                                                │"
echo -e "├────────────────────────────────────────────────────────────────────┤"
echo -e "│ [2/3] psql -c \"SELECT 1\" db_test                                  │"
echo -e "│       ✓ PASS (0.2s)                                                │"
echo -e "├────────────────────────────────────────────────────────────────────┤"
echo -e "│ [3/3] curl -s http://localhost:8080/health | jq .status          │"
echo -e "│       ✓ PASS (0.3s)                                                │"
echo -e "├────────────────────────────────────────────────────────────────────┤"
echo -e "│ Summary: All 3 passed ✓                                            │"
echo -e "└────────────────────────────────────────────────────────────────────┘"

echo ""
sleep 0.5

TS=$(timestamp)
echo -e "[${TS}] === Execution Summary ==="
echo "Total tasks: 3"
echo "Completed: 3"
echo "Failed: 0"
echo "Duration: 5.1s"
echo ""
echo "Task Types:"
echo "  Component tasks: 2 (GREEN)"
echo "  Integration tasks: 1 (GREEN)"
echo ""
echo "All tasks completed - API and Database integrated successfully ✓"
