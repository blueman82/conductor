#!/bin/bash
# Demo: conductor observe stats
# Simulates summary statistics output

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

echo -e "${BOLD}$ conductor observe stats${NC}"
echo ""
sleep 0.5

echo ""
echo -e "${BOLD}=== SUMMARY STATISTICS ===${NC}"
echo ""
echo "Total Sessions:      147"
echo "Total Agents:        12"
echo "Success Rate:        87.1%"
echo "Average Duration:    4m32s"
echo ""
echo "Total Input Tokens:  1,234,567"
echo "Total Output Tokens: 456,789"
echo "Total Tokens:        1,691,356"
echo "Avg Tokens/Session:  11,506"
echo "Estimated Cost:      \$24.87 (Sonnet pricing)"
echo ""
echo -e "${BOLD}--- Agent Performance (Top 10) ---${NC}"
printf "%-10s %-10s %-10s %-12s %-8s %s\n" "Sessions" "Success" "Failures" "Duration" "Rate" "Agent Type"
echo "──────────────────────────────────────────────────────────────────────────────────────────"
printf "%-10s %-10s %-10s %-12s %6s  %s\n" "45" "42" "3" "3m45s" "93.3%" "golang-pro"
printf "%-10s %-10s %-10s %-12s %6s  %s\n" "32" "28" "4" "5m12s" "87.5%" "code-reviewer"
printf "%-10s %-10s %-10s %-12s %6s  %s\n" "24" "22" "2" "4m08s" "91.7%" "security-auditor"
printf "%-10s %-10s %-10s %-12s %6s  %s\n" "18" "15" "3" "6m30s" "83.3%" "database-optimizer"
printf "%-10s %-10s %-10s %-12s %6s  %s\n" "12" "11" "1" "2m45s" "91.7%" "typescript-pro"
printf "%-10s %-10s %-10s %-12s %6s  %s\n" "8" "6" "2" "4m55s" "75.0%" "python-pro"
printf "%-10s %-10s %-10s %-12s %6s  %s\n" "5" "5" "0" "3m20s" "100.0%" "rust-pro"
printf "%-10s %-10s %-10s %-12s %6s  %s\n" "3" "2" "1" "7m10s" "66.7%" "ml-engineer"
echo ""
echo "(Showing 8 of 12 agents. Use --limit flag to see more)"
echo ""

# Demo with project filter
echo -e "${BOLD}───────────────────────────────────────────────────────────────${NC}"
echo ""
echo -e "${BOLD}$ conductor observe stats --project conductor${NC}"
echo ""
sleep 0.5

echo ""
echo -e "${BOLD}=== SUMMARY STATISTICS ===${NC}"
echo ""
echo "Total Sessions:      45"
echo "Total Agents:        6"
echo "Success Rate:        91.1%"
echo "Average Duration:    3m15s"
echo ""
echo "Total Input Tokens:  456,123"
echo "Total Output Tokens: 123,456"
echo "Total Tokens:        579,579"
echo "Avg Tokens/Session:  12,879"
echo "Estimated Cost:      \$8.45 (Sonnet pricing)"
