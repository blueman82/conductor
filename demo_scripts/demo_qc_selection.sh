#!/bin/bash
# Demo: conductor run plan.md --verbose --qc-mode intelligent --qc-agents code-reviewer,python-pro
# Demonstrates v2.4+ QC agent selection strategies
#
# Features:
# - auto mode: System selects appropriate agents automatically
# - explicit mode: Use only specified agents via --qc-agents
# - mixed mode: Auto-select + specified agents combined
# - intelligent mode: Claude analyzes task context and recommends agents

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m'

timestamp() {
    date +"%H:%M:%S"
}

echo -e "${BOLD}$ conductor run plan.md --verbose --qc-mode intelligent${NC}"
echo ""

echo "Plan Summary:"
echo "  Total tasks: 1"
echo "  QC mode: intelligent"
echo ""
sleep 0.5

echo "Starting execution..."
echo ""

# Task execution
TS=$(timestamp)
echo -e "[${TS}] Starting Wave 1: 1 tasks"
sleep 0.3

TS=$(timestamp)
echo -e "[${TS}] ⏳ IN PROGRESS [1/1] Task 1 (Implement User Auth) (agent: python-pro)"
sleep 1.5

TS=$(timestamp)
echo -e "[${TS}] ✓ Task output: auth.py created with login/logout functions"
sleep 0.3

# QC Agent Selection output (based on actual code)
echo ""
echo -e "${CYAN}[QC] Selected agents:${NC} [code-reviewer, python-pro, quality-control] (mode: intelligent)"
sleep 0.3

echo -e "${CYAN}[QC] Intelligent selection reasoning:${NC}"
echo -e "  - python-pro: Task uses Python. Validates language-specific best practices."
echo -e "  - code-reviewer: Ensures general code quality standards and refactoring opportunities."
echo -e "  - quality-control: Baseline validator for success criteria verification."
sleep 0.5

# Individual agent reviews
echo ""
echo -e "${CYAN}[QC] Running multi-agent review...${NC}"
sleep 1.0

TS=$(timestamp)
echo -e "[${TS}] [QC] code-reviewer verdict: GREEN - Code follows Python conventions with proper type hints"
sleep 0.3

TS=$(timestamp)
echo -e "[${TS}] [QC] python-pro verdict: GREEN - Secure password hashing (bcrypt), proper async login flow"
sleep 0.3

TS=$(timestamp)
echo -e "[${TS}] [QC] quality-control verdict: GREEN - All success criteria verified (auth functions, error handling)"
sleep 0.3

# Consensus
echo ""
echo -e "${CYAN}[QC] Consensus verdict: GREEN${NC} (all agents agree - 3/3 consensus)"
sleep 0.5

TS=$(timestamp)
echo -e "[${TS}] ✓ Task 1 (Implement User Auth): GREEN (4.2s, agent: python-pro, 1 files)"
sleep 0.5

echo ""
TS=$(timestamp)
echo -e "[${TS}] === Execution Summary ==="
echo "Total tasks: 1"
echo "Completed: 1"
echo "Failed: 0"
echo ""
echo "QC Summary:"
echo "  Mode: intelligent"
echo "  Agents: 3 (code-reviewer, python-pro, quality-control)"
echo "  Consensus: GREEN"
