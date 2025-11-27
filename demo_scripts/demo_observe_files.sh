#!/bin/bash
# Demo: conductor observe files
# Simulates file operation pattern analysis

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

echo -e "${BOLD}$ conductor observe files${NC}"
echo ""
sleep 0.5

echo ""
echo -e "${BOLD}=== File Operation Analysis ===${NC}"
echo ""
printf "%-15s %-12s %-12s\n" "Operation" "Count" "Success Rate"
echo "────────────────────────────────────────────────"
printf "%-15s %-12s ${GREEN}%-12s${NC}\n" "Read" "1,234" "99.8%"
printf "%-15s %-12s ${GREEN}%-12s${NC}\n" "Edit" "987" "97.2%"
printf "%-15s %-12s ${GREEN}%-12s${NC}\n" "Write (new)" "456" "98.5%"
printf "%-15s %-12s ${GREEN}%-12s${NC}\n" "Delete" "23" "100.0%"
echo ""
echo "Total: 2,700 file operations"
echo ""
echo -e "${BOLD}--- Most Accessed Files ---${NC}"
printf "%-50s %-10s %-10s %-10s\n" "File Path" "Reads" "Edits" "Total"
echo "────────────────────────────────────────────────────────────────────────────────"
printf "%-50s %-10s %-10s %-10s\n" "internal/executor/task.go" "45" "23" "68"
printf "%-50s %-10s %-10s %-10s\n" "internal/models/task.go" "42" "18" "60"
printf "%-50s %-10s %-10s %-10s\n" "internal/parser/markdown.go" "38" "15" "53"
printf "%-50s %-10s %-10s %-10s\n" "internal/executor/qc.go" "35" "12" "47"
printf "%-50s %-10s %-10s %-10s\n" "internal/cmd/run.go" "32" "14" "46"
printf "%-50s %-10s %-10s %-10s\n" "go.mod" "28" "5" "33"
printf "%-50s %-10s %-10s %-10s\n" "internal/agent/invoker.go" "25" "8" "33"
printf "%-50s %-10s %-10s %-10s\n" "README.md" "22" "9" "31"
echo ""
echo -e "${BOLD}--- File Type Distribution ---${NC}"
printf "%-15s %-12s %-12s\n" "Extension" "Operations" "Percentage"
echo "────────────────────────────────────────────────"
printf "%-15s %-12s %-12s\n" ".go" "1,876" "69.5%"
printf "%-15s %-12s %-12s\n" ".md" "342" "12.7%"
printf "%-15s %-12s %-12s\n" ".yaml" "234" "8.7%"
printf "%-15s %-12s %-12s\n" ".json" "123" "4.6%"
printf "%-15s %-12s %-12s\n" ".ts" "87" "3.2%"
printf "%-15s %-12s %-12s\n" "other" "38" "1.4%"
echo ""
echo -e "${BOLD}--- Recently Modified Files ---${NC}"
printf "%-50s %-20s %-10s\n" "File" "Last Modified" "By Agent"
echo "────────────────────────────────────────────────────────────────────────────────────"
printf "%-50s %-20s %-10s\n" "internal/executor/task.go" "2024-01-15 14:28:17" "golang-pro"
printf "%-50s %-20s %-10s\n" "internal/cmd/run.go" "2024-01-15 14:15:42" "golang-pro"
printf "%-50s %-20s %-10s\n" "README.md" "2024-01-15 13:45:30" "code-reviewer"
printf "%-50s %-20s %-10s\n" "internal/parser/yaml.go" "2024-01-15 12:30:15" "golang-pro"
printf "%-50s %-20s %-10s\n" "go.mod" "2024-01-15 11:22:08" "golang-pro"
