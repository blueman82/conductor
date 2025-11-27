#!/bin/bash
# Demo: conductor observe export
# Simulates exporting data to various formats

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

echo -e "${BOLD}$ conductor observe export --format json --output metrics.json${NC}"
echo ""
sleep 0.5

echo "Collecting metrics..."
sleep 0.5
echo "Exporting to JSON format..."
sleep 0.3
echo -e "${GREEN}✓${NC} Exported 147 sessions to metrics.json"
echo ""
echo "File size: 245 KB"
echo ""

# Show sample of exported JSON
echo -e "${BOLD}Sample output (metrics.json):${NC}"
echo '{'
echo '  "summary": {'
echo '    "total_sessions": 147,'
echo '    "success_rate": 0.871,'
echo '    "total_tokens": 1691356,'
echo '    "estimated_cost_usd": 24.87'
echo '  },'
echo '  "sessions": ['
echo '    {'
echo '      "id": "abc123",'
echo '      "project": "conductor",'
echo '      "agent": "golang-pro",'
echo '      "status": "success",'
echo '      "duration_seconds": 272,'
echo '      "tokens": 16348'
echo '    },'
echo '    ...'
echo '  ]'
echo '}'
echo ""

# Markdown export
echo -e "${BOLD}───────────────────────────────────────────────────────────────${NC}"
echo ""
echo -e "${BOLD}$ conductor observe export --format markdown --output report.md${NC}"
echo ""
sleep 0.5

echo "Collecting metrics..."
sleep 0.5
echo "Generating Markdown report..."
sleep 0.3
echo -e "${GREEN}✓${NC} Exported report to report.md"
echo ""
echo "File size: 18 KB"
echo ""

echo -e "${BOLD}Sample output (report.md):${NC}"
echo '# Agent Watch Report'
echo ''
echo '## Summary'
echo '- Total Sessions: 147'
echo '- Success Rate: 87.1%'
echo '- Total Tokens: 1,691,356'
echo '- Estimated Cost: $24.87'
echo ''
echo '## Agent Performance'
echo '| Agent | Sessions | Success Rate |'
echo '|-------|----------|--------------|'
echo '| golang-pro | 45 | 93.3% |'
echo '| code-reviewer | 32 | 87.5% |'
echo '...'
echo ""

# CSV export
echo -e "${BOLD}───────────────────────────────────────────────────────────────${NC}"
echo ""
echo -e "${BOLD}$ conductor observe export --format csv --output data.csv${NC}"
echo ""
sleep 0.5

echo "Collecting metrics..."
sleep 0.5
echo "Exporting to CSV format..."
sleep 0.3
echo -e "${GREEN}✓${NC} Exported 147 rows to data.csv"
echo ""
echo "File size: 32 KB"
echo ""

echo -e "${BOLD}Sample output (data.csv):${NC}"
echo 'session_id,project,agent,status,duration_s,tokens,cost_usd'
echo 'abc123,conductor,golang-pro,success,272,16348,0.24'
echo 'def456,conductor,code-reviewer,success,135,8234,0.12'
echo 'ghi789,conductor,database-optimizer,failed,408,12456,0.18'
echo '...'
