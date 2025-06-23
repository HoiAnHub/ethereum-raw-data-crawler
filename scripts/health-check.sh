#!/bin/bash

# Quick health check after deployment
echo "=== SCHEDULER HEALTH CHECK ==="

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_status() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓ $2${NC}"
    else
        echo -e "${RED}✗ $2${NC}"
    fi
}

# Check process
echo -e "\n${BLUE}1. Process Status:${NC}"
if ps aux | grep -E "(scheduler|ethereum-scheduler-app)" | grep -v grep; then
    print_status 0 "Process running"
else
    print_status 1 "Process not found"
fi

# Check recent logs
echo -e "\n${BLUE}2. Recent Log Entries:${NC}"
if [ -f "scheduler.log" ]; then
    echo "Last 5 log entries:"
    tail -5 scheduler.log | while read line; do
        echo "  $line"
    done

    # Check for errors
    error_count=$(grep -i "error\|panic\|fatal" scheduler.log | wc -l)
    warn_count=$(grep -i "warn" scheduler.log | wc -l)
    echo -e "\n${BLUE}Log Analysis:${NC}"
    echo "  Total errors: $error_count"
    echo "  Total warnings: $warn_count"

    # Recent errors
    recent_errors=$(grep -i "error\|panic\|fatal" scheduler.log | tail -3)
    if [ -n "$recent_errors" ]; then
        echo -e "\n${YELLOW}Recent errors:${NC}"
        echo "$recent_errors" | while read line; do
            echo "  $line"
        done
    fi
else
    print_status 1 "Log file not found"
fi

# Check MongoDB connection
echo -e "\n${BLUE}3. MongoDB Connection:${NC}"
if docker exec ethereum-scheduler-mongodb mongosh --eval "db.adminCommand('ping')" --quiet 2>/dev/null; then
    print_status 0 "MongoDB connection OK"

    # Check database stats
    block_count=$(docker exec ethereum-scheduler-mongodb mongosh ethereum_raw_data --eval "db.blocks.countDocuments()" --quiet 2>/dev/null | tail -1)
    tx_count=$(docker exec ethereum-scheduler-mongodb mongosh ethereum_raw_data --eval "db.transactions.countDocuments()" --quiet 2>/dev/null | tail -1)
    echo "  Blocks in DB: $block_count"
    echo "  Transactions in DB: $tx_count"
else
    print_status 1 "MongoDB connection failed"
fi

# Check memory usage
echo -e "\n${BLUE}4. Memory Usage:${NC}"
if ps -o pid,rss,vsz,pmem,etime,command -C scheduler 2>/dev/null; then
    print_status 0 "Memory stats available"
else
    echo "  No scheduler process found for memory check"
fi

# Check disk space
echo -e "\n${BLUE}5. Disk Space:${NC}"
df -h . | tail -1 | while read filesystem size used available percent mountpoint; do
    echo "  Available: $available ($percent used)"
    if [ "${percent%\%}" -gt 90 ]; then
        print_status 1 "Disk space critical"
    else
        print_status 0 "Disk space OK"
    fi
done

# Check Docker containers (if applicable)
echo -e "\n${BLUE}6. Docker Containers:${NC}"
if docker ps | grep -E "(ethereum-scheduler|mongodb)" > /dev/null; then
    docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" | grep -E "(ethereum-scheduler|mongodb)"
    print_status 0 "Docker containers running"
else
    echo "  No Docker containers found"
fi

# Check network connectivity
echo -e "\n${BLUE}7. Network Connectivity:${NC}"
if command -v curl > /dev/null; then
    if curl -s --max-time 5 https://mainnet.infura.io > /dev/null; then
        print_status 0 "Internet connectivity OK"
    else
        print_status 1 "Internet connectivity issues"
    fi
else
    echo "  curl not available for network check"
fi

# Final summary
echo -e "\n${BLUE}=== HEALTH CHECK SUMMARY ===${NC}"
echo "Timestamp: $(date)"

if ps aux | grep -E "(scheduler|ethereum-scheduler-app)" | grep -v grep > /dev/null; then
    if [ -f "scheduler.log" ]; then
        recent_success=$(grep "Block processed successfully\|Started" scheduler.log | tail -1)
        if [ -n "$recent_success" ]; then
            echo -e "${GREEN}✓ System appears healthy${NC}"
        else
            echo -e "${YELLOW}⚠ System running but no recent block processing${NC}"
        fi
    else
        echo -e "${YELLOW}⚠ Process running but no log file${NC}"
    fi
else
    echo -e "${RED}✗ System not running${NC}"
fi