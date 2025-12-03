#!/bin/bash
# Demo: Intelligent QC Agent Selection (v2.4+)
# Demonstrates --qc-mode and --qc-agents flags with different selection strategies
#
# Features demonstrated:
# - auto mode: Automatic agent selection based on task type
# - explicit mode: Use only specified agents
# - mixed mode: Auto-select + explicit agents combined
# - intelligent mode: Claude-based context-aware selection with reasoning

# Colors
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

echo -e "${BOLD}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${BOLD}Conductor v2.4+ - Intelligent QC Agent Selection Demo${NC}"
echo -e "${BOLD}═══════════════════════════════════════════════════════════════${NC}"
echo ""
sleep 1

# ============================================================================
# SCENARIO 1: AUTO MODE
# ============================================================================
echo -e "${BOLD}SCENARIO 1: AUTO MODE${NC}"
echo -e "${CYAN}(Let conductor auto-select QC agents based on task type)${NC}"
echo ""
sleep 0.5

echo -e "${BOLD}$ conductor run complex-plan.md --verbose${NC}"
echo ""
sleep 0.3

echo "Loading plan from complex-plan.md..."
sleep 0.2
echo "Validating dependencies..."
sleep 0.2

echo ""
echo "Plan Summary:"
echo "  Total tasks: 3"
echo "  Execution waves: 2"
echo ""
sleep 0.5

echo "Starting execution..."
echo ""

# Task 1 - API Implementation
TS=$(timestamp)
echo -e "[${TS}] Starting ${BOLD}Wave 1${NC}: 2 tasks"
sleep 0.3

TS=$(timestamp)
echo -e "[${TS}] ⏳ IN PROGRESS ${CYAN}[1/2]${NC} Task 1 (Implement API Endpoints) ${MAGENTA}(agent: golang-pro)${NC}"
sleep 2

TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Mode: ${MAGENTA}auto${NC}"
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Selected agents: ${MAGENTA}[code-reviewer]${NC}"
sleep 0.3
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Selection strategy: Task is code implementation, using general-purpose code-reviewer"
sleep 0.3
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Verdict from code-reviewer: ${GREEN}GREEN${NC}"
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}✓${NC} Task 1 (Implement API Endpoints): ${GREEN}GREEN${NC} (3.2s, 4 files)"

sleep 0.4

# Task 2 - Database Schema
TS=$(timestamp)
echo -e "[${TS}] ⏳ IN PROGRESS ${CYAN}[2/2]${NC} Task 2 (Database Schema) ${MAGENTA}(agent: database-optimizer)${NC}"
sleep 2

TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Mode: ${MAGENTA}auto${NC}"
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Selected agents: ${MAGENTA}[database-optimizer]${NC}"
sleep 0.3
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Selection strategy: Task involves database work, using domain-specific database-optimizer"
sleep 0.3
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Verdict from database-optimizer: ${GREEN}GREEN${NC}"
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}✓${NC} Task 2 (Database Schema): ${GREEN}GREEN${NC} (2.8s, 2 files)"

sleep 0.3
TS=$(timestamp)
echo -e "[${TS}] ${BOLD}Wave 1${NC} ${GREEN}complete${NC} (6.0s) - 2/2 completed (${GREEN}2 GREEN${NC})"
echo ""

# ============================================================================
# SCENARIO 2: EXPLICIT MODE
# ============================================================================
echo ""
echo -e "${BOLD}SCENARIO 2: EXPLICIT MODE${NC}"
echo -e "${CYAN}(Use only specified agents via --qc-agents)${NC}"
echo ""
sleep 0.5

echo -e "${BOLD}$ conductor run security-plan.md --verbose --qc-agents security-auditor,code-reviewer${NC}"
echo ""
sleep 0.3

echo "Loading plan from security-plan.md..."
sleep 0.2
echo "Validating dependencies..."
sleep 0.2

echo ""
echo "Plan Summary:"
echo "  Total tasks: 2"
echo "  Execution waves: 1"
echo ""
sleep 0.5

echo "Starting execution..."
echo ""

# Task 1 - Auth Implementation
TS=$(timestamp)
echo -e "[${TS}] Starting ${BOLD}Wave 1${NC}: 2 tasks"
sleep 0.3

TS=$(timestamp)
echo -e "[${TS}] ⏳ IN PROGRESS ${CYAN}[1/2]${NC} Task 1 (Implement Authentication) ${MAGENTA}(agent: security-auditor)${NC}"
sleep 2.5

TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Mode: ${MAGENTA}explicit${NC}"
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Selected agents: ${MAGENTA}[security-auditor, code-reviewer]${NC}"
sleep 0.3
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Selection strategy: Using explicitly configured agents (--qc-agents override)"
sleep 0.3
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Verdict from security-auditor: ${GREEN}GREEN${NC}"
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Verdict from code-reviewer: ${GREEN}GREEN${NC}"
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Final verdict (strictest-wins): ${GREEN}GREEN${NC}"
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}✓${NC} Task 1 (Implement Authentication): ${GREEN}GREEN${NC} (4.1s, 6 files)"

sleep 0.4

# Task 2 - Access Control
TS=$(timestamp)
echo -e "[${TS}] ⏳ IN PROGRESS ${CYAN}[2/2]${NC} Task 2 (Access Control & Permissions) ${MAGENTA}(agent: security-auditor)${NC}"
sleep 2.3

TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Mode: ${MAGENTA}explicit${NC}"
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Selected agents: ${MAGENTA}[security-auditor, code-reviewer]${NC}"
sleep 0.3
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Verdict from security-auditor: ${GREEN}GREEN${NC}"
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Verdict from code-reviewer: ${GREEN}GREEN${NC}"
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Final verdict (strictest-wins): ${GREEN}GREEN${NC}"
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}✓${NC} Task 2 (Access Control & Permissions): ${GREEN}GREEN${NC} (3.9s, 5 files)"

sleep 0.3
TS=$(timestamp)
echo -e "[${TS}] ${BOLD}Wave 1${NC} ${GREEN}complete${NC} (8.2s) - 2/2 completed (${GREEN}2 GREEN${NC})"
echo ""

# ============================================================================
# SCENARIO 3: MIXED MODE
# ============================================================================
echo ""
echo -e "${BOLD}SCENARIO 3: MIXED MODE${NC}"
echo -e "${CYAN}(Combine auto-selected agents with explicit overrides)${NC}"
echo ""
sleep 0.5

echo -e "${BOLD}$ conductor run integration-plan.md --verbose --qc-mode mixed --qc-agents security-auditor${NC}"
echo ""
sleep 0.3

echo "Loading plan from integration-plan.md..."
sleep 0.2
echo "Validating dependencies..."
sleep 0.2

echo ""
echo "Plan Summary:"
echo "  Total tasks: 2"
echo "  Execution waves: 1"
echo ""
sleep 0.5

echo "Starting execution..."
echo ""

# Task 1 - API + Database integration
TS=$(timestamp)
echo -e "[${TS}] Starting ${BOLD}Wave 1${NC}: 1 task"
sleep 0.3

TS=$(timestamp)
echo -e "[${TS}] ⏳ IN PROGRESS ${CYAN}[1/1]${NC} Task 1 (Integrate API with Database) ${MAGENTA}(agent: golang-pro)${NC}"
sleep 2.5

TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Mode: ${MAGENTA}mixed${NC}"
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Selected agents: ${MAGENTA}[code-reviewer, database-optimizer, security-auditor]${NC}"
sleep 0.3
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Selection strategy:"
sleep 0.1
echo -e "[${TS}]   - Auto-selected: code-reviewer (general code), database-optimizer (database logic)"
sleep 0.1
echo -e "[${TS}]   - Explicit addition: security-auditor (--qc-agents override)"
sleep 0.3
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Verdict from code-reviewer: ${GREEN}GREEN${NC}"
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Verdict from database-optimizer: ${GREEN}GREEN${NC}"
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Verdict from security-auditor: ${YELLOW}YELLOW${NC} (SQL injection risk in query)"
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Final verdict (strictest-wins): ${YELLOW}YELLOW${NC}"
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}⚠${NC} Task 1 (Integrate API with Database): ${YELLOW}YELLOW${NC} (3.7s)"
echo -e "[${TS}]   Issues:"
echo -e "[${TS}]   - SQL injection vulnerability in query construction"
echo -e "[${TS}]   - Missing parameterized query usage"
echo -e "[${TS}]   Recommendation: Use prepared statements for all database queries"

sleep 0.5
TS=$(timestamp)
echo -e "[${TS}] ${YELLOW}Retrying Task 1...${NC} (attempt 2/3)"
sleep 2.2

TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Mode: ${MAGENTA}mixed${NC}"
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Selected agents: ${MAGENTA}[code-reviewer, database-optimizer, security-auditor]${NC}"
sleep 0.3
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Verdict from code-reviewer: ${GREEN}GREEN${NC}"
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Verdict from database-optimizer: ${GREEN}GREEN${NC}"
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Verdict from security-auditor: ${GREEN}GREEN${NC}"
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Final verdict (strictest-wins): ${GREEN}GREEN${NC}"
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}✓${NC} Task 1 (Integrate API with Database): ${GREEN}GREEN${NC} (3.4s, 5 files)"

sleep 0.3
TS=$(timestamp)
echo -e "[${TS}] ${BOLD}Wave 1${NC} ${GREEN}complete${NC} (5.9s) - 1/1 completed (${GREEN}1 GREEN${NC})"
echo ""

# ============================================================================
# SCENARIO 4: INTELLIGENT MODE
# ============================================================================
echo ""
echo -e "${BOLD}SCENARIO 4: INTELLIGENT MODE${NC}"
echo -e "${CYAN}(Claude-based context-aware selection with detailed reasoning)${NC}"
echo ""
sleep 0.5

echo -e "${BOLD}$ conductor run error-handling-plan.md --verbose --qc-mode intelligent${NC}"
echo ""
sleep 0.3

echo "Loading plan from error-handling-plan.md..."
sleep 0.2
echo "Validating dependencies..."
sleep 0.2

echo ""
echo "Plan Summary:"
echo "  Total tasks: 3"
echo "  Execution waves: 2"
echo ""
sleep 0.5

echo "Starting execution..."
echo ""

# Task 1 - Error Recovery
TS=$(timestamp)
echo -e "[${TS}] Starting ${BOLD}Wave 1${NC}: 2 tasks"
sleep 0.3

TS=$(timestamp)
echo -e "[${TS}] ⏳ IN PROGRESS ${CYAN}[1/2]${NC} Task 1 (Error Recovery Handlers) ${MAGENTA}(agent: golang-pro)${NC}"
sleep 2.1

TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Mode: ${MAGENTA}intelligent${NC}"
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Analyzing task context for intelligent selection..."
sleep 0.4
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Selected agents: ${MAGENTA}[code-reviewer]${NC}"
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Selection reasoning:"
sleep 0.1
echo -e "[${TS}]   Task: Error Recovery Handlers"
sleep 0.1
echo -e "[${TS}]   Executing agent: golang-pro (suitable for Go error handling)"
sleep 0.1
echo -e "[${TS}]   Error pattern: No historical failures detected"
sleep 0.1
echo -e "[${TS}]   Recommendation: Use code-reviewer for general validation"
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Verdict from code-reviewer: ${GREEN}GREEN${NC}"
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}✓${NC} Task 1 (Error Recovery Handlers): ${GREEN}GREEN${NC} (2.8s, 3 files)"

sleep 0.4

# Task 2 - Error Logging (has error patterns)
TS=$(timestamp)
echo -e "[${TS}] ⏳ IN PROGRESS ${CYAN}[2/2]${NC} Task 2 (Error Logging & Monitoring) ${MAGENTA}(agent: backend-developer)${NC}"
sleep 2.3

TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Mode: ${MAGENTA}intelligent${NC}"
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Analyzing task context for intelligent selection..."
sleep 0.4
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Selected agents: ${MAGENTA}[code-reviewer, security-auditor]${NC}"
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Selection reasoning:"
sleep 0.1
echo -e "[${TS}]   Task: Error Logging & Monitoring"
sleep 0.1
echo -e "[${TS}]   Executing agent: backend-developer (multi-agent context)"
sleep 0.1
echo -e "[${TS}]   Error pattern: Logging tasks have security implications (PII exposure)"
sleep 0.1
echo -e "[${TS}]   Historical failures: 2 previous RED verdicts due to sensitive data leaks"
sleep 0.1
echo -e "[${TS}]   Recommendation: Add security-auditor for sensitive data validation"
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Verdict from code-reviewer: ${GREEN}GREEN${NC}"
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Verdict from security-auditor: ${RED}RED${NC}"
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Final verdict (strictest-wins): ${RED}RED${NC}"
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}✗${NC} Task 2 (Error Logging & Monitoring): ${RED}RED${NC} (3.1s)"
echo -e "[${TS}]   Issues:"
echo -e "[${TS}]   - User credentials logged in debug output"
echo -e "[${TS}]   - Database connection strings exposed in logs"
echo -e "[${TS}]   - Missing log redaction for sensitive fields"
echo -e "[${TS}]   Suggested agent for retry: security-auditor (patterns show PII issues)"

sleep 0.5
TS=$(timestamp)
echo -e "[${TS}] ${YELLOW}Retrying Task 2...${NC} (attempt 2/3, agent swap: backend-developer → security-auditor)"
sleep 2.4

TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Mode: ${MAGENTA}intelligent${NC}"
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Selected agents: ${MAGENTA}[code-reviewer, security-auditor]${NC}"
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Verdict from code-reviewer: ${GREEN}GREEN${NC}"
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Verdict from security-auditor: ${GREEN}GREEN${NC}"
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Final verdict (strictest-wins): ${GREEN}GREEN${NC}"
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}✓${NC} Task 2 (Error Logging & Monitoring): ${GREEN}GREEN${NC} (3.0s, 4 files)"

sleep 0.3
TS=$(timestamp)
echo -e "[${TS}] ${BOLD}Wave 1${NC} ${GREEN}complete${NC} (7.4s) - 2/2 completed (${GREEN}1 GREEN${NC}, ${RED}1 RED${NC} w/ retry)"
echo ""

# Wave 2
sleep 0.5
TS=$(timestamp)
echo -e "[${TS}] Starting ${BOLD}Wave 2${NC}: 1 task"
sleep 0.3

TS=$(timestamp)
echo -e "[${TS}] ⏳ IN PROGRESS ${CYAN}[1/1]${NC} Task 3 (Error Response Standards) ${MAGENTA}(agent: golang-pro)${NC}"
sleep 2

TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Mode: ${MAGENTA}intelligent${NC}"
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Analyzing task context for intelligent selection..."
sleep 0.4
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Selected agents: ${MAGENTA}[code-reviewer, security-auditor]${NC}"
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Selection reasoning:"
sleep 0.1
echo -e "[${TS}]   Task: Error Response Standards"
sleep 0.1
echo -e "[${TS}]   Executing agent: golang-pro"
sleep 0.1
echo -e "[${TS}]   Cross-task analysis: Related to error logging which had security issues"
sleep 0.1
echo -e "[${TS}]   Recommendation: Include security-auditor for consistency"
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Verdict from code-reviewer: ${GREEN}GREEN${NC}"
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Verdict from security-auditor: ${GREEN}GREEN${NC}"
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}[QC]${NC} Final verdict (strictest-wins): ${GREEN}GREEN${NC}"
sleep 0.2
TS=$(timestamp)
echo -e "[${TS}] ${CYAN}✓${NC} Task 3 (Error Response Standards): ${GREEN}GREEN${NC} (2.9s, 2 files)"

sleep 0.3
TS=$(timestamp)
echo -e "[${TS}] ${BOLD}Wave 2${NC} ${GREEN}complete${NC} (2.9s) - 1/1 completed (${GREEN}1 GREEN${NC})"
echo ""

# ============================================================================
# SUMMARY
# ============================================================================
sleep 0.5
echo -e "${BOLD}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${BOLD}Summary: QC Agent Selection Modes${NC}"
echo -e "${BOLD}═══════════════════════════════════════════════════════════════${NC}"
echo ""

echo -e "${BOLD}1. AUTO MODE${NC}"
echo -e "   Command: ${CYAN}conductor run plan.md${NC}"
echo -e "   Behavior: Conducts auto-selects agents based on task type/domain"
echo -e "   Use case: Default behavior, general-purpose execution"
echo ""

echo -e "${BOLD}2. EXPLICIT MODE${NC}"
echo -e "   Command: ${CYAN}conductor run plan.md --qc-agents agent1,agent2${NC}"
echo -e "   Behavior: Uses ONLY specified agents (no auto-selection)"
echo -e "   Use case: Strict quality gates, specific reviewer requirements"
echo ""

echo -e "${BOLD}3. MIXED MODE${NC}"
echo -e "   Command: ${CYAN}conductor run plan.md --qc-mode mixed --qc-agents agent1${NC}"
echo -e "   Behavior: Auto-selects + explicit agents combined"
echo -e "   Use case: Baseline reviewers (auto) + domain specialists (explicit)"
echo ""

echo -e "${BOLD}4. INTELLIGENT MODE${NC}"
echo -e "   Command: ${CYAN}conductor run plan.md --qc-mode intelligent${NC}"
echo -e "   Behavior: Claude analyzes task context and error patterns for selection"
echo -e "   Features:"
echo -e "     • Historical error analysis"
echo -e "     • Domain-specific agent recommendations"
echo -e "     • Cross-task dependency awareness"
echo -e "     • Detailed selection reasoning"
echo -e "   Use case: Adaptive quality control, complex multi-agent scenarios"
echo ""

echo -e "${BOLD}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${BOLD}Key Features:${NC}"
echo -e "${BOLD}═══════════════════════════════════════════════════════════════${NC}"
echo ""

echo -e "${CYAN}[QC] Mode:${NC} Displays current selection mode"
echo -e "${CYAN}[QC] Selected agents:${NC} Lists chosen QC agents"
echo -e "${CYAN}[QC] Selection strategy:${NC} Explains why agents were selected (auto/mixed)"
echo -e "${CYAN}[QC] Selection reasoning:${NC} Detailed Claude analysis (intelligent mode)"
echo -e "${CYAN}[QC] Verdict:${NC} Per-agent verdicts and final strictest-wins result"
echo ""

echo -e "${BOLD}═══════════════════════════════════════════════════════════════${NC}"
echo "Demo completed!"
echo ""
