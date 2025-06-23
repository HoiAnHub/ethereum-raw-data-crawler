#!/bin/bash

# Monitor and Auto-Restart Script for Ethereum Scheduler
# This script monitors containers and automatically restarts them if they fail

set -e

# Configuration
COMPOSE_FILE="docker-compose.scheduler.yml"
PROJECT_NAME="ethereum-raw-data-crawler"
CHECK_INTERVAL=30  # seconds
MAX_RESTART_ATTEMPTS=3
RESTART_COOLDOWN=60  # seconds

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Counters
mongodb_restart_count=0
app_restart_count=0
last_restart_time=0

# Function to log with timestamp
log() {
    echo -e "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

# Function to check if container is running
is_container_running() {
    local container_name="$1"
    docker ps --format "{{.Names}}" | grep -q "^${container_name}$"
}

# Function to check if container is healthy
is_container_healthy() {
    local container_name="$1"
    local health_status
    health_status=$(docker inspect --format='{{.State.Health.Status}}' "$container_name" 2>/dev/null || echo "no-health-check")

    if [ "$health_status" = "healthy" ] || [ "$health_status" = "no-health-check" ]; then
        return 0
    else
        return 1
    fi
}

# Function to check MongoDB connection
check_mongodb_connection() {
    docker exec ethereum-scheduler-mongodb mongosh --eval "db.adminCommand('ping').ok" --quiet >/dev/null 2>&1
}

# Function to check if app is processing blocks
check_app_processing() {
    local recent_logs
    recent_logs=$(docker logs ethereum-scheduler-app --since=2m 2>/dev/null | grep -c "Block processed successfully" || echo "0")

    # Clean up any whitespace/newlines
    recent_logs=$(echo "$recent_logs" | tr -d '\n\r ')

    # If we see recent block processing, app is working
    if [ "$recent_logs" -gt 0 ] 2>/dev/null; then
        return 0
    fi

    # Check if app is at least running and connected
    recent_logs=$(docker logs ethereum-scheduler-app --since=1m 2>/dev/null | grep -c "WebSocket scheduler\|Successfully connected" || echo "0")
    recent_logs=$(echo "$recent_logs" | tr -d '\n\r ')

    if [ "$recent_logs" -gt 0 ] 2>/dev/null; then
        return 0
    fi

    return 1
}

# Function to restart container with cooldown
restart_container() {
    local container_name="$1"
    local current_time=$(date +%s)

    # Check cooldown period
    if [ $((current_time - last_restart_time)) -lt $RESTART_COOLDOWN ]; then
        log "${YELLOW}Restart cooldown active, skipping restart of $container_name${NC}"
        return 1
    fi

    log "${YELLOW}Restarting container: $container_name${NC}"

    if [ "$container_name" = "ethereum-scheduler-mongodb" ]; then
        mongodb_restart_count=$((mongodb_restart_count + 1))
        if [ $mongodb_restart_count -gt $MAX_RESTART_ATTEMPTS ]; then
            log "${RED}MongoDB has been restarted too many times, stopping monitoring${NC}"
            return 1
        fi
    elif [ "$container_name" = "ethereum-scheduler-app" ]; then
        app_restart_count=$((app_restart_count + 1))
        if [ $app_restart_count -gt $MAX_RESTART_ATTEMPTS ]; then
            log "${RED}App has been restarted too many times, stopping monitoring${NC}"
            return 1
        fi
    fi

    # Restart the container
    docker restart "$container_name" || {
        log "${RED}Failed to restart $container_name${NC}"
        return 1
    }

    last_restart_time=$current_time
    log "${GREEN}Successfully restarted $container_name${NC}"

    # Wait for container to be ready
    sleep 30
    return 0
}

# Function to restart entire stack
restart_stack() {
    log "${YELLOW}Restarting entire stack...${NC}"

    docker-compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" down
    sleep 10
    docker-compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" up -d

    log "${GREEN}Stack restarted${NC}"

    # Reset counters
    mongodb_restart_count=0
    app_restart_count=0
    last_restart_time=$(date +%s)

    # Wait for services to be ready
    sleep 60
}

# Function to show status
show_status() {
    log "${BLUE}=== Container Status ===${NC}"

    if is_container_running "ethereum-scheduler-mongodb"; then
        if is_container_healthy "ethereum-scheduler-mongodb"; then
            log "${GREEN}✓ MongoDB: Running and Healthy${NC}"
        else
            log "${YELLOW}⚠ MongoDB: Running but Unhealthy${NC}"
        fi
    else
        log "${RED}✗ MongoDB: Not Running${NC}"
    fi

    if is_container_running "ethereum-scheduler-app"; then
        if is_container_healthy "ethereum-scheduler-app"; then
            log "${GREEN}✓ App: Running and Healthy${NC}"
        else
            log "${YELLOW}⚠ App: Running but Unhealthy${NC}"
        fi
    else
        log "${RED}✗ App: Not Running${NC}"
    fi

    # Show resource usage
    if is_container_running "ethereum-scheduler-mongodb" || is_container_running "ethereum-scheduler-app"; then
        log "${BLUE}=== Resource Usage ===${NC}"
        docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}" \
            ethereum-scheduler-mongodb ethereum-scheduler-app 2>/dev/null || true
    fi

    log "MongoDB restarts: $mongodb_restart_count/$MAX_RESTART_ATTEMPTS"
    log "App restarts: $app_restart_count/$MAX_RESTART_ATTEMPTS"
}

# Main monitoring loop
main() {
    log "${BLUE}Starting Ethereum Scheduler Monitor${NC}"
    log "Check interval: ${CHECK_INTERVAL}s"
    log "Max restart attempts: $MAX_RESTART_ATTEMPTS"
    log "Restart cooldown: ${RESTART_COOLDOWN}s"
    echo ""

    while true; do
        local mongodb_ok=true
        local app_ok=true
        local issues=()

        # Check MongoDB
        if ! is_container_running "ethereum-scheduler-mongodb"; then
            mongodb_ok=false
            issues+=("MongoDB container not running")
        elif ! is_container_healthy "ethereum-scheduler-mongodb"; then
            mongodb_ok=false
            issues+=("MongoDB container unhealthy")
        elif ! check_mongodb_connection; then
            mongodb_ok=false
            issues+=("MongoDB connection failed")
        fi

        # Check App
        if ! is_container_running "ethereum-scheduler-app"; then
            app_ok=false
            issues+=("App container not running")
        elif ! is_container_healthy "ethereum-scheduler-app"; then
            app_ok=false
            issues+=("App container unhealthy")
        elif ! check_app_processing; then
            app_ok=false
            issues+=("App not processing blocks")
        fi

        # Handle issues
        if [ ${#issues[@]} -gt 0 ]; then
            log "${RED}Issues detected:${NC}"
            for issue in "${issues[@]}"; do
                log "${RED}  - $issue${NC}"
            done

            # Decide restart strategy
            if [ "$mongodb_ok" = false ] && [ "$app_ok" = false ]; then
                restart_stack
            elif [ "$mongodb_ok" = false ]; then
                restart_container "ethereum-scheduler-mongodb"
            elif [ "$app_ok" = false ]; then
                restart_container "ethereum-scheduler-app"
            fi
        else
            log "${GREEN}All services healthy${NC}"
            # Reset restart counters on successful check
            if [ $mongodb_restart_count -gt 0 ] || [ $app_restart_count -gt 0 ]; then
                log "${GREEN}Resetting restart counters${NC}"
                mongodb_restart_count=0
                app_restart_count=0
            fi
        fi

        # Show status every 5 checks
        if [ $(($(date +%s) % 150)) -lt $CHECK_INTERVAL ]; then
            show_status
            echo ""
        fi

        sleep $CHECK_INTERVAL
    done
}

# Handle script termination
cleanup() {
    log "${YELLOW}Monitor stopped${NC}"
    exit 0
}

trap cleanup SIGINT SIGTERM

# Handle command line arguments
case "${1:-}" in
    --status)
        show_status
        ;;
    --restart-stack)
        restart_stack
        ;;
    --help)
        echo "Usage: $0 [--status|--restart-stack|--help]"
        echo "  --status        Show current status"
        echo "  --restart-stack Restart entire stack"
        echo "  --help          Show this help"
        echo "  (no args)       Start monitoring"
        ;;
    *)
        main
        ;;
esac
