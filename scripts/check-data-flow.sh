#!/bin/bash

# Data Flow Monitoring Script
# This script checks if data is being written to MongoDB continuously

set -e

# Configuration
MONGO_URI="mongodb://admin:password@localhost:27017/ethereum_raw_data?authSource=admin"
CHECK_INTERVAL=60  # seconds
ALERT_THRESHOLD=300  # seconds (5 minutes without new data)

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to log with timestamp
log() {
    echo -e "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

# Function to get latest block from MongoDB
get_latest_block_from_db() {
    docker exec ethereum-scheduler-mongodb mongosh "$MONGO_URI" --eval "
        db.blocks.findOne({}, {number: 1, timestamp: 1, _id: 0}, {sort: {number: -1}})
    " --quiet 2>/dev/null | grep -E "number|timestamp" | head -2
}

# Function to get latest transaction from MongoDB
get_latest_transaction_from_db() {
    docker exec ethereum-scheduler-mongodb mongosh "$MONGO_URI" --eval "
        db.transactions.findOne({}, {blockNumber: 1, timestamp: 1, _id: 0}, {sort: {blockNumber: -1}})
    " --quiet 2>/dev/null | grep -E "blockNumber|timestamp" | head -2
}

# Function to get total counts
get_total_counts() {
    local block_count
    local tx_count
    
    block_count=$(docker exec ethereum-scheduler-mongodb mongosh "$MONGO_URI" --eval "db.blocks.countDocuments()" --quiet 2>/dev/null || echo "0")
    tx_count=$(docker exec ethereum-scheduler-mongodb mongosh "$MONGO_URI" --eval "db.transactions.countDocuments()" --quiet 2>/dev/null || echo "0")
    
    echo "Blocks: $block_count, Transactions: $tx_count"
}

# Function to check recent activity
check_recent_activity() {
    local recent_blocks
    local recent_txs
    local current_time=$(date +%s)
    local five_minutes_ago=$((current_time - 300))
    
    # Check for blocks added in last 5 minutes
    recent_blocks=$(docker exec ethereum-scheduler-mongodb mongosh "$MONGO_URI" --eval "
        db.blocks.countDocuments({createdAt: {\$gte: new Date($five_minutes_ago * 1000)}})
    " --quiet 2>/dev/null || echo "0")
    
    # Check for transactions added in last 5 minutes
    recent_txs=$(docker exec ethereum-scheduler-mongodb mongosh "$MONGO_URI" --eval "
        db.transactions.countDocuments({createdAt: {\$gte: new Date($five_minutes_ago * 1000)}})
    " --quiet 2>/dev/null || echo "0")
    
    echo "Recent (5min): Blocks: $recent_blocks, Transactions: $recent_txs"
    
    # Return 0 if there's recent activity, 1 if not
    if [ "$recent_blocks" -gt 0 ] || [ "$recent_txs" -gt 0 ]; then
        return 0
    else
        return 1
    fi
}

# Function to check application logs for recent processing
check_app_processing() {
    local recent_processing
    recent_processing=$(docker logs ethereum-scheduler-app --since=5m 2>/dev/null | grep -c "Block processed successfully" || echo "0")
    
    echo "Recent processing (5min): $recent_processing blocks"
    
    if [ "$recent_processing" -gt 0 ]; then
        return 0
    else
        return 1
    fi
}

# Function to show detailed status
show_detailed_status() {
    log "${BLUE}=== Data Flow Status ===${NC}"
    
    # Container status
    if docker ps --format "{{.Names}}" | grep -q "ethereum-scheduler-mongodb"; then
        log "${GREEN}✓ MongoDB container: Running${NC}"
    else
        log "${RED}✗ MongoDB container: Not running${NC}"
        return 1
    fi
    
    if docker ps --format "{{.Names}}" | grep -q "ethereum-scheduler-app"; then
        log "${GREEN}✓ App container: Running${NC}"
    else
        log "${RED}✗ App container: Not running${NC}"
        return 1
    fi
    
    # Database connectivity
    if docker exec ethereum-scheduler-mongodb mongosh "$MONGO_URI" --eval "db.adminCommand('ping').ok" --quiet >/dev/null 2>&1; then
        log "${GREEN}✓ MongoDB connection: OK${NC}"
    else
        log "${RED}✗ MongoDB connection: Failed${NC}"
        return 1
    fi
    
    # Data counts
    local counts
    counts=$(get_total_counts)
    log "${BLUE}Total data: $counts${NC}"
    
    # Latest data
    log "${BLUE}Latest block in DB:${NC}"
    get_latest_block_from_db | sed 's/^/  /'
    
    log "${BLUE}Latest transaction in DB:${NC}"
    get_latest_transaction_from_db | sed 's/^/  /'
    
    # Recent activity
    local recent_status
    recent_status=$(check_recent_activity)
    log "${BLUE}$recent_status${NC}"
    
    # App processing
    local processing_status
    processing_status=$(check_app_processing)
    log "${BLUE}$processing_status${NC}"
    
    # Check if data flow is healthy
    if check_recent_activity && check_app_processing; then
        log "${GREEN}✓ Data flow: Healthy${NC}"
        return 0
    else
        log "${YELLOW}⚠ Data flow: Stalled${NC}"
        return 1
    fi
}

# Function to send alert (placeholder for future notification system)
send_alert() {
    local message="$1"
    log "${RED}ALERT: $message${NC}"
    
    # Future: Send to Slack, email, etc.
    # For now, just log to a separate alert file
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] ALERT: $message" >> /tmp/ethereum-alerts.log
}

# Function to check and alert on data flow issues
monitor_data_flow() {
    local consecutive_failures=0
    local max_failures=3
    
    while true; do
        log "${YELLOW}Checking data flow...${NC}"
        
        if show_detailed_status; then
            if [ $consecutive_failures -gt 0 ]; then
                log "${GREEN}Data flow recovered after $consecutive_failures failures${NC}"
                send_alert "Data flow recovered after $consecutive_failures failures"
            fi
            consecutive_failures=0
        else
            consecutive_failures=$((consecutive_failures + 1))
            log "${RED}Data flow check failed ($consecutive_failures/$max_failures)${NC}"
            
            if [ $consecutive_failures -ge $max_failures ]; then
                send_alert "Data flow has been stalled for $((consecutive_failures * CHECK_INTERVAL)) seconds"
                
                # Try to restart containers
                log "${YELLOW}Attempting to restart containers...${NC}"
                if [ -f "./scripts/monitor-and-restart.sh" ]; then
                    ./scripts/monitor-and-restart.sh --restart-stack
                else
                    docker-compose -f docker-compose.scheduler.yml restart
                fi
                
                # Reset counter after restart attempt
                consecutive_failures=0
                
                # Wait longer after restart
                sleep 120
                continue
            fi
        fi
        
        echo ""
        sleep $CHECK_INTERVAL
    done
}

# Function to run a single check
run_single_check() {
    show_detailed_status
    exit $?
}

# Function to show help
show_help() {
    echo "Usage: $0 [command]"
    echo ""
    echo "Commands:"
    echo "  monitor    Start continuous monitoring (default)"
    echo "  check      Run single check and exit"
    echo "  status     Show detailed status"
    echo "  help       Show this help message"
    echo ""
    echo "Environment variables:"
    echo "  CHECK_INTERVAL     Check interval in seconds (default: 60)"
    echo "  ALERT_THRESHOLD    Alert threshold in seconds (default: 300)"
}

# Main function
main() {
    log "${BLUE}Starting Data Flow Monitor${NC}"
    log "Check interval: ${CHECK_INTERVAL}s"
    log "Alert threshold: ${ALERT_THRESHOLD}s"
    echo ""
    
    monitor_data_flow
}

# Handle script termination
cleanup() {
    log "${YELLOW}Data flow monitor stopped${NC}"
    exit 0
}

trap cleanup SIGINT SIGTERM

# Handle command line arguments
case "${1:-monitor}" in
    monitor)
        main
        ;;
    check|status)
        run_single_check
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        log "${RED}Unknown command: $1${NC}"
        show_help
        exit 1
        ;;
esac
