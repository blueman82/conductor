#!/bin/bash
# Demo: conductor validate plan.md
# Simulates plan validation output

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Color

echo -e "${BOLD}$ conductor validate plan.md${NC}"
echo ""
sleep 0.5

echo -e "${GREEN}✓${NC} Validating plan from plan.md"
sleep 0.3
echo -e "${GREEN}✓${NC} Parsed 8 tasks successfully"
sleep 0.3
echo -e "${GREEN}✓${NC} No circular dependencies detected"
sleep 0.3
echo -e "${GREEN}✓${NC} No file overlaps in parallel tasks"
sleep 0.3
echo -e "${GREEN}✓${NC} All agents available"
sleep 0.3
echo -e "${GREEN}✓${NC} All task dependencies valid"
echo ""
echo -e "${GREEN}✓${NC} Plan is valid!"
echo ""

# Demo with validation errors
echo -e "${BOLD}───────────────────────────────────────────────────────────────${NC}"
echo ""
echo -e "${BOLD}$ conductor validate invalid-plan.md${NC}"
echo ""
sleep 0.5

echo -e "${GREEN}✓${NC} Validating plan from invalid-plan.md"
sleep 0.3
echo -e "${GREEN}✓${NC} Parsed 5 tasks successfully"
sleep 0.3
echo -e "${RED}✗${NC} Circular dependency detected"
sleep 0.3
echo ""
echo -e "${RED}✗${NC} Validation failed for plan from invalid-plan.md"
echo -e "  ${RED}✗${NC} Circular dependency detected in task dependencies"
echo ""
echo "Found 1 validation error(s)!"
echo ""

# Demo multi-file validation
echo -e "${BOLD}───────────────────────────────────────────────────────────────${NC}"
echo ""
echo -e "${BOLD}$ conductor validate docs/plans/backend/${NC}"
echo ""
sleep 0.5

echo "Validating plan files:"
echo -e "  ${CYAN}[1/3]${NC} plan-01-foundation.yaml"
sleep 0.3
echo -e "  ${CYAN}[2/3]${NC} plan-02-features.yaml"
sleep 0.3
echo -e "  ${CYAN}[3/3]${NC} plan-03-testing.yaml"
sleep 0.3
echo -e "${GREEN}✓${NC} Parsed 24 tasks from 3 plan files"
sleep 0.3
echo -e "${GREEN}✓${NC} No circular dependencies detected"
sleep 0.3
echo -e "${GREEN}✓${NC} No file overlaps in parallel tasks"
sleep 0.3
echo -e "${GREEN}✓${NC} All agents available"
sleep 0.3
echo -e "${GREEN}✓${NC} All task dependencies valid"
echo ""
echo -e "${GREEN}✓${NC} Plan is valid!"
