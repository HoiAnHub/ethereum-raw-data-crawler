#!/bin/bash

# MongoDB Monitoring Script for Ethereum Scheduler
# This script monitors MongoDB health and connection status

set -e

CONTAINER_NAME="ethereum-scheduler-mongodb"
LOG_FILE="/tmp/mongodb-monitor.log"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to log with timestamp
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

# Function to check if container is running
check_container() {
    if docker ps --format "table {{.Names}}" | grep -q "$CONTAINER_NAME"; then
        return 0
    else
        return 1
    fi
}

# Function to check MongoDB health
check_mongodb_health() {
    local result
    result=$(docker exec "$CONTAINER_NAME" mongosh --eval "db.adminCommand('ping').ok" --quiet 2>/dev/null || echo "0")
    if [ "$result" = "1" ]; then
        return 0
    else
        return 1
    fi
}

# Function to get MongoDB connection count
get_connection_count() {
    docker exec "$CONTAINER_NAME" mongosh --eval "db.serverStatus().connections" --quiet 2>/dev/null || echo "N/A"
}

# Function to get MongoDB memory usage
get_memory_usage() {
    docker exec "$CONTAINER_NAME" mongosh --eval "db.serverStatus().mem" --quiet 2>/dev/null || echo "N/A"
}

# Function to check MongoDB logs for errors
check_mongodb_logs() {
    local error_count
    error_count=$(docker logs "$CONTAINER_NAME" --since=1m 2>&1 | grep -i "error\|exception\|failed" | wc -l)
    echo "$error_count"
}

# Function to restart MongoDB container if needed
restart_mongodb() {
    log "${YELLOW}Restarting MongoDB container...${NC}"
    docker restart "$CONTAINER_NAME"
    sleep 10
    
    if check_container && check_mongodb_health; then
        log "${GREEN}MongoDB container restarted successfully${NC}"
        return 0
    else
        log "${RED}Failed to restart MongoDB container${NC}"
        return 1
    fi
}

# Main monitoring loop
main() {
    log "Starting MongoDB monitoring for $CONTAINER_NAME"
    
    local consecutive_failures=0
    local max_failures=3
    
    while true; do
        if ! check_container; then
            log "${RED}MongoDB container is not running${NC}"
            consecutive_failures=$((consecutive_failures + 1))
        elif ! check_mongodb_health; then
            log "${RED}MongoDB health check failed${NC}"
            consecutive_failures=$((consecutive_failures + 1))
        else
            # Container is healthy
            if [ $consecutive_failures -gt 0 ]; then
                log "${GREEN}MongoDB is healthy again after $consecutive_failures failures${NC}"
            fi
            consecutive_failures=0
            
            # Log status information
            local connections
            local memory
            local error_count
            
            connections=$(get_connection_count)
            memory=$(get_memory_usage)
            error_count=$(check_mongodb_logs)
            
            log "${GREEN}MongoDB Status: Healthy${NC}"
            log "Connections: $connections"
            log "Memory: $memory"
            log "Recent errors (last 1min): $error_count"
        fi
        
        # Check if we need to restart
        if [ $consecutive_failures -ge $max_failures ]; then
            log "${YELLOW}MongoDB has failed $consecutive_failures consecutive health checks${NC}"
            if restart_mongodb; then
                consecutive_failures=0
            else
                log "${RED}Failed to recover MongoDB, will try again in next cycle${NC}"
            fi
        fi
        
        # Wait before next check
        sleep 30
    done
}

# Handle script termination
cleanup() {
    log "MongoDB monitoring stopped"
    exit 0
}

trap cleanup SIGINT SIGTERM

# Check if running in background
if [ "$1" = "--daemon" ]; then
    main > "$LOG_FILE" 2>&1 &
    echo "MongoDB monitoring started in background. Log file: $LOG_FILE"
    echo "PID: $!"
else
    main
fi
