#!/bin/bash
# Demo: conductor observe tools
# Simulates tool usage analysis

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

echo -e "${BOLD}$ conductor observe tools${NC}"
echo ""
sleep 0.5

echo ""
echo -e "${BOLD}=== Tool Usage Analysis ===${NC}"
echo ""
printf "%-20s %-12s %-12s %-12s %-15s\n" "Tool" "Executions" "Success" "Errors" "Avg Duration"
echo "────────────────────────────────────────────────────────────────────────────"
printf "%-20s %-12s ${GREEN}%-12s${NC} %-12s %-15s\n" "Read" "1,234" "99.8%" "2" "45ms"
printf "%-20s %-12s ${GREEN}%-12s${NC} %-12s %-15s\n" "Edit" "987" "97.2%" "28" "123ms"
printf "%-20s %-12s ${GREEN}%-12s${NC} %-12s %-15s\n" "Grep" "876" "99.5%" "4" "89ms"
printf "%-20s %-12s ${GREEN}%-12s${NC} %-12s %-15s\n" "Write" "654" "96.8%" "21" "156ms"
printf "%-20s %-12s ${YELLOW}%-12s${NC} %-12s %-15s\n" "Bash" "543" "85.6%" "78" "2.3s"
printf "%-20s %-12s ${GREEN}%-12s${NC} %-12s %-15s\n" "Glob" "432" "100.0%" "0" "34ms"
printf "%-20s %-12s ${GREEN}%-12s${NC} %-12s %-15s\n" "LS" "321" "99.7%" "1" "28ms"
printf "%-20s %-12s ${GREEN}%-12s${NC} %-12s %-15s\n" "Task" "234" "92.3%" "18" "45s"
printf "%-20s %-12s ${GREEN}%-12s${NC} %-12s %-15s\n" "WebSearch" "123" "94.3%" "7" "1.2s"
printf "%-20s %-12s ${GREEN}%-12s${NC} %-12s %-15s\n" "FetchUrl" "98" "91.8%" "8" "890ms"
echo ""
echo "Total: 5,502 tool executions"
echo ""
echo -e "${BOLD}--- Common Tool Sequences ---${NC}"
echo "1. Read → Edit → Bash (156 occurrences)"
echo "2. Grep → Read → Edit (134 occurrences)"
echo "3. Read → Read → Edit (98 occurrences)"
echo "4. Glob → Read → Edit (87 occurrences)"
echo "5. Bash → Edit → Bash (65 occurrences)"
echo ""
echo -e "${BOLD}--- Tools with Highest Error Rates ---${NC}"
printf "%-20s %-12s %-12s %s\n" "Tool" "Errors" "Rate" "Common Error"
echo "────────────────────────────────────────────────────────────────────────────"
printf "%-20s %-12s ${RED}%-12s${NC} %s\n" "Bash" "78" "14.4%" "Non-zero exit code"
printf "%-20s %-12s ${YELLOW}%-12s${NC} %s\n" "FetchUrl" "8" "8.2%" "Connection timeout"
printf "%-20s %-12s ${YELLOW}%-12s${NC} %s\n" "Task" "18" "7.7%" "Agent timeout"
printf "%-20s %-12s %-12s %s\n" "WebSearch" "7" "5.7%" "Rate limit"
