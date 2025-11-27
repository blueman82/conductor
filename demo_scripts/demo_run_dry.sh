#!/bin/bash
# Demo: conductor run plan.md --dry-run --verbose
# Simulates dry-run execution output

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
BOLD='\033[1m'
NC='\033[0m'

echo -e "${BOLD}$ conductor run plan.md --dry-run --verbose${NC}"
echo ""
sleep 0.5

echo "Loading plan from plan.md..."
sleep 0.3
echo "Validating dependencies..."
sleep 0.3

echo ""
echo "Plan Summary:"
echo "  Total tasks: 8"
echo "  Execution waves: 3"
echo "  Timeout: 10h0m0s"
echo "  Max concurrency: 3"
echo ""
sleep 0.5

echo "Dry-run mode: Plan is valid and ready for execution."
echo ""
echo "Execution waves:"
echo "  Wave 1: 2 task(s)"
echo "    - Task 1: Initialize Project Structure"
echo "    - Task 2: Setup Database Schema"
sleep 0.3
echo "  Wave 2: 4 task(s)"
echo "    - Task 3: Implement User Authentication"
echo "    - Task 4: Create API Endpoints"
echo "    - Task 5: Build Data Models"
echo "    - Task 6: Setup Testing Framework"
sleep 0.3
echo "  Wave 3: 2 task(s)"
echo "    - Task 7: Integration Tests"
echo "    - Task 8: Documentation"
echo ""

# Demo with skip-completed
echo -e "${BOLD}───────────────────────────────────────────────────────────────${NC}"
echo ""
echo -e "${BOLD}$ conductor run plan.md --dry-run --verbose --skip-completed${NC}"
echo ""
sleep 0.5

echo "Loading plan from plan.md..."
sleep 0.3
echo "Validating dependencies..."
sleep 0.3

echo ""
echo "Plan Summary:"
echo "  Total tasks: 8"
echo "  Execution waves: 3"
echo "  Timeout: 10h0m0s"
echo ""
sleep 0.5

echo "Dry-run mode: Plan is valid and ready for execution."
echo ""
echo "Execution waves:"
echo "  Wave 1: 0 task(s)"
sleep 0.3
echo "  Wave 2: 2 task(s)"
echo "    - Task 5: Build Data Models"
echo "    - Task 6: Setup Testing Framework"
sleep 0.3
echo "  Wave 3: 2 task(s)"
echo "    - Task 7: Integration Tests"
echo "    - Task 8: Documentation"
