#!/bin/bash
# Demo: conductor learning commands
# Simulates learning system CLI commands

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
BOLD='\033[1m'
NC='\033[0m'

echo -e "${BOLD}$ conductor learning stats plan.md${NC}"
echo ""
sleep 0.5

echo ""
echo -e "${BOLD}=== Learning Statistics for plan.md ===${NC}"
echo ""
echo "Total Executions:     45"
echo "Unique Tasks:         8"
echo "Success Rate:         88.9%"
echo "Avg Retries/Task:     0.3"
echo ""
echo -e "${BOLD}--- Task Performance ---${NC}"
printf "%-30s %-10s %-10s %-12s\n" "Task" "Runs" "Success" "Avg Time"
echo "────────────────────────────────────────────────────────────────"
printf "%-30s %-10s ${GREEN}%-10s${NC} %-12s\n" "Initialize Project" "6" "100.0%" "2.1s"
printf "%-30s %-10s ${GREEN}%-10s${NC} %-12s\n" "Setup Database" "6" "100.0%" "3.4s"
printf "%-30s %-10s ${GREEN}%-10s${NC} %-12s\n" "User Authentication" "6" "83.3%" "5.2s"
printf "%-30s %-10s ${GREEN}%-10s${NC} %-12s\n" "API Endpoints" "6" "100.0%" "2.8s"
printf "%-30s %-10s ${YELLOW}%-10s${NC} %-12s\n" "Data Models" "6" "66.7%" "4.1s"
printf "%-30s %-10s ${GREEN}%-10s${NC} %-12s\n" "Testing Framework" "6" "100.0%" "3.5s"
printf "%-30s %-10s ${GREEN}%-10s${NC} %-12s\n" "Integration Tests" "5" "80.0%" "6.8s"
printf "%-30s %-10s ${GREEN}%-10s${NC} %-12s\n" "Documentation" "4" "100.0%" "1.9s"
echo ""
echo -e "${BOLD}--- Agent Effectiveness ---${NC}"
printf "%-20s %-10s %-12s\n" "Agent" "Tasks" "Success Rate"
echo "────────────────────────────────────────────────"
printf "%-20s %-10s ${GREEN}%-12s${NC}\n" "golang-pro" "28" "92.9%"
printf "%-20s %-10s ${GREEN}%-12s${NC}\n" "security-auditor" "8" "87.5%"
printf "%-20s %-10s ${GREEN}%-12s${NC}\n" "database-optimizer" "6" "83.3%"
printf "%-20s %-10s ${YELLOW}%-12s${NC}\n" "code-reviewer" "3" "66.7%"
echo ""

# Show specific task history
echo -e "${BOLD}───────────────────────────────────────────────────────────────${NC}"
echo ""
echo -e "${BOLD}$ conductor learning show plan.md \"Data Models\"${NC}"
echo ""
sleep 0.5

echo ""
echo -e "${BOLD}=== Execution History: Data Models ===${NC}"
echo ""
printf "%-20s %-15s %-10s %-10s %s\n" "Timestamp" "Agent" "Verdict" "Duration" "Notes"
echo "────────────────────────────────────────────────────────────────────────────────"
printf "%-20s %-15s ${GREEN}%-10s${NC} %-10s %s\n" "2024-01-15 14:30" "golang-pro" "GREEN" "3.8s" ""
printf "%-20s %-15s ${RED}%-10s${NC} %-10s %s\n" "2024-01-14 10:15" "golang-pro" "RED" "4.2s" "Missing validation"
printf "%-20s %-15s ${GREEN}%-10s${NC} %-10s %s\n" "2024-01-14 10:18" "golang-pro" "GREEN" "3.5s" "(retry)"
printf "%-20s %-15s ${GREEN}%-10s${NC} %-10s %s\n" "2024-01-13 16:45" "golang-pro" "GREEN" "4.1s" ""
printf "%-20s %-15s ${RED}%-10s${NC} %-10s %s\n" "2024-01-12 09:30" "database-opt" "RED" "5.2s" "Schema mismatch"
printf "%-20s %-15s ${YELLOW}%-10s${NC} %-10s %s\n" "2024-01-12 09:35" "golang-pro" "YELLOW" "4.8s" "(agent swap)"
echo ""
echo -e "${BOLD}Patterns Detected:${NC}"
echo "  • Schema validation issues occur when database-optimizer is used"
echo "  • golang-pro recovers well after failed attempts"
echo "  • Average retry count: 0.5"
echo ""

# Export learning data
echo -e "${BOLD}───────────────────────────────────────────────────────────────${NC}"
echo ""
echo -e "${BOLD}$ conductor learning export learning-data.json${NC}"
echo ""
sleep 0.5

echo "Exporting learning data..."
sleep 0.3
echo -e "${GREEN}✓${NC} Exported 45 execution records to learning-data.json"
echo ""

# Clear learning data
echo -e "${BOLD}───────────────────────────────────────────────────────────────${NC}"
echo ""
echo -e "${BOLD}$ conductor learning clear${NC}"
echo ""
sleep 0.5

echo -e "${YELLOW}Warning:${NC} This will permanently delete all learning data."
echo ""
echo -n "Are you sure you want to continue? [y/N] "
sleep 1
echo "n"
echo ""
echo "Operation cancelled."
