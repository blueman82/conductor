#!/bin/bash
# Demo: conductor observe project
# Simulates project listing and project analysis

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

echo -e "${BOLD}$ conductor observe project${NC}"
echo ""
sleep 0.5

echo ""
echo -e "${BOLD}=== Available Projects ===${NC}"
echo ""
printf "%-30s %-12s %-12s %-15s\n" "Project" "Sessions" "Success" "Last Activity"
echo "──────────────────────────────────────────────────────────────────────────"
printf "%-30s %-12s %-12s %-15s\n" "conductor" "45" "91.1%" "2h ago"
printf "%-30s %-12s %-12s %-15s\n" "my-webapp" "32" "84.4%" "5h ago"
printf "%-30s %-12s %-12s %-15s\n" "api-service" "28" "89.3%" "1d ago"
printf "%-30s %-12s %-12s %-15s\n" "data-pipeline" "22" "86.4%" "3d ago"
printf "%-30s %-12s %-12s %-15s\n" "mobile-app" "15" "80.0%" "1w ago"
printf "%-30s %-12s %-12s %-15s\n" "infra-scripts" "5" "100.0%" "2w ago"
echo ""
echo "Total: 6 projects, 147 sessions"
echo ""

# Project-specific analysis
echo -e "${BOLD}───────────────────────────────────────────────────────────────${NC}"
echo ""
echo -e "${BOLD}$ conductor observe project conductor${NC}"
echo ""
sleep 0.5

echo ""
echo -e "${BOLD}=== Project: conductor ===${NC}"
echo ""
echo "Sessions:        45"
echo "Success Rate:    91.1%"
echo "Avg Duration:    3m15s"
echo "Last Activity:   2h ago"
echo ""
echo -e "${BOLD}--- Tool Usage ---${NC}"
printf "%-20s %-10s %-10s\n" "Tool" "Count" "Success"
echo "────────────────────────────────────────────"
printf "%-20s %-10s %-10s\n" "Edit" "234" "98.3%"
printf "%-20s %-10s %-10s\n" "Read" "189" "100.0%"
printf "%-20s %-10s %-10s\n" "Grep" "156" "99.4%"
printf "%-20s %-10s %-10s\n" "Write" "98" "97.9%"
printf "%-20s %-10s %-10s\n" "Bash" "67" "88.1%"
printf "%-20s %-10s %-10s\n" "Glob" "45" "100.0%"
echo ""
echo -e "${BOLD}--- Recent Sessions ---${NC}"
printf "%-10s %-12s %-10s %-20s\n" "ID" "Duration" "Status" "Agent"
echo "────────────────────────────────────────────────────────"
printf "%-10s %-12s ${GREEN}%-10s${NC} %-20s\n" "abc123" "4m32s" "success" "golang-pro"
printf "%-10s %-12s ${GREEN}%-10s${NC} %-20s\n" "def456" "2m15s" "success" "code-reviewer"
printf "%-10s %-12s ${RED}%-10s${NC} %-20s\n" "ghi789" "6m48s" "failed" "database-optimizer"
printf "%-10s %-12s ${GREEN}%-10s${NC} %-20s\n" "jkl012" "3m22s" "success" "golang-pro"
printf "%-10s %-12s ${GREEN}%-10s${NC} %-20s\n" "mno345" "1m58s" "success" "typescript-pro"
