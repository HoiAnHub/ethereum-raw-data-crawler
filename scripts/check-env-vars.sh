#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default container name
CONTAINER_NAME="${1:-ethereum-scheduler-app}"

print_header() {
    echo -e "${BLUE}================================================${NC}"
    echo -e "${BLUE} Environment Variables Checker${NC}"
    echo -e "${BLUE}================================================${NC}"
    echo ""
}

print_section() {
    echo -e "${YELLOW}$1${NC}"
    echo "----------------------------------------"
}

check_container_exists() {
    if ! docker ps --format "table {{.Names}}" | grep -q "^${CONTAINER_NAME}$"; then
        echo -e "${RED}Error: Container '${CONTAINER_NAME}' not found or not running${NC}"
        echo ""
        echo "Available containers:"
        docker ps --format "table {{.Names}}\t{{.Status}}"
        exit 1
    fi
}

check_critical_vars() {
    print_section "🔍 Critical Environment Variables"

    local critical_vars=(
        "ETHEREUM_RPC_URL"
        "ETHEREUM_WS_URL"
        "MONGO_URI"
        "MONGO_DATABASE"
        "APP_ENV"
        "CRAWLER_USE_UPSERT"
        "CRAWLER_UPSERT_FALLBACK"
    )

    for var in "${critical_vars[@]}"; do
        local value=$(docker exec "$CONTAINER_NAME" sh -c "echo \$$var" 2>/dev/null)
        if [ -n "$value" ]; then
            # Mask sensitive data
            if [[ "$var" == *"URI"* ]] || [[ "$var" == *"URL"* ]]; then
                masked_value=$(echo "$value" | sed -E 's/(:\/\/[^:@]*:)[^@]*(@)/\1***\2/g')
                echo -e "${GREEN}✓${NC} $var = ${masked_value}"
            else
                echo -e "${GREEN}✓${NC} $var = $value"
            fi
        else
            echo -e "${RED}✗${NC} $var = ${RED}NOT SET${NC}"
        fi
    done
    echo ""
}

check_scheduler_vars() {
    print_section "⚡ Scheduler Configuration"

    local scheduler_vars=(
        "SCHEDULER_MODE"
        "SCHEDULER_ENABLE_REALTIME"
        "SCHEDULER_ENABLE_POLLING"
        "SCHEDULER_POLLING_INTERVAL"
        "SCHEDULER_FALLBACK_TIMEOUT"
        "SCHEDULER_RECONNECT_ATTEMPTS"
        "SCHEDULER_RECONNECT_DELAY"
    )

    for var in "${scheduler_vars[@]}"; do
        local value=$(docker exec "$CONTAINER_NAME" sh -c "echo \$$var" 2>/dev/null)
        if [ -n "$value" ]; then
            echo -e "${GREEN}✓${NC} $var = $value"
        else
            echo -e "${YELLOW}⚠${NC} $var = ${YELLOW}using default${NC}"
        fi
    done
    echo ""
}

check_performance_vars() {
    print_section "🚀 Performance Configuration"

    local perf_vars=(
        "BATCH_SIZE"
        "CONCURRENT_WORKERS"
        "RETRY_ATTEMPTS"
        "RETRY_DELAY"
        "ETHEREUM_RATE_LIMIT"
        "ETHEREUM_REQUEST_TIMEOUT"
        "MONGO_MAX_POOL_SIZE"
    )

    for var in "${perf_vars[@]}"; do
        local value=$(docker exec "$CONTAINER_NAME" sh -c "echo \$$var" 2>/dev/null)
        if [ -n "$value" ]; then
            echo -e "${GREEN}✓${NC} $var = $value"
        else
            echo -e "${YELLOW}⚠${NC} $var = ${YELLOW}using default${NC}"
        fi
    done
    echo ""
}

check_websocket_vars() {
    print_section "🔌 WebSocket Configuration"

    local ws_vars=(
        "WEBSOCKET_RECONNECT_ATTEMPTS"
        "WEBSOCKET_RECONNECT_DELAY"
        "WEBSOCKET_PING_INTERVAL"
    )

    for var in "${ws_vars[@]}"; do
        local value=$(docker exec "$CONTAINER_NAME" sh -c "echo \$$var" 2>/dev/null)
        if [ -n "$value" ]; then
            echo -e "${GREEN}✓${NC} $var = $value"
        else
            echo -e "${YELLOW}⚠${NC} $var = ${YELLOW}using default${NC}"
        fi
    done
    echo ""
}

test_database_connection() {
    print_section "💾 Database Connection Test"

    local mongo_uri=$(docker exec "$CONTAINER_NAME" sh -c "echo \$MONGO_URI" 2>/dev/null)
    if [ -n "$mongo_uri" ]; then
        echo "Testing MongoDB connection..."
        if docker exec "$CONTAINER_NAME" sh -c "timeout 10 mongosh \"$mongo_uri\" --eval 'db.adminCommand(\"ping\")' --quiet" >/dev/null 2>&1; then
            echo -e "${GREEN}✓${NC} MongoDB connection: ${GREEN}SUCCESS${NC}"
        else
            echo -e "${RED}✗${NC} MongoDB connection: ${RED}FAILED${NC}"
        fi
    else
        echo -e "${RED}✗${NC} MONGO_URI not set"
    fi
    echo ""
}

test_ethereum_connection() {
    print_section "⛓️ Ethereum Connection Test"

    local rpc_url=$(docker exec "$CONTAINER_NAME" sh -c "echo \$ETHEREUM_RPC_URL" 2>/dev/null)
    if [ -n "$rpc_url" ]; then
        echo "Testing Ethereum RPC connection..."
        if curl -s --max-time 10 -X POST -H "Content-Type: application/json" \
           --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
           "$rpc_url" | grep -q "result"; then
            echo -e "${GREEN}✓${NC} Ethereum RPC connection: ${GREEN}SUCCESS${NC}"
        else
            echo -e "${RED}✗${NC} Ethereum RPC connection: ${RED}FAILED${NC}"
        fi
    else
        echo -e "${RED}✗${NC} ETHEREUM_RPC_URL not set"
    fi
    echo ""
}

show_container_info() {
    print_section "📊 Container Information"

    echo "Container: $CONTAINER_NAME"
    echo "Status: $(docker inspect --format='{{.State.Status}}' "$CONTAINER_NAME")"
    echo "Started: $(docker inspect --format='{{.State.StartedAt}}' "$CONTAINER_NAME")"
    echo "Image: $(docker inspect --format='{{.Config.Image}}' "$CONTAINER_NAME")"
    echo ""
}

show_logs_summary() {
    print_section "📝 Recent Logs (Last 10 lines)"

    docker logs --tail=10 "$CONTAINER_NAME"
    echo ""
}

# Main execution
main() {
    print_header

    # Check if container exists
    check_container_exists

    # Show container info
    show_container_info

    # Check environment variables
    check_critical_vars
    check_scheduler_vars
    check_performance_vars
    check_websocket_vars

    # Test connections
    test_database_connection
    test_ethereum_connection

    # Show recent logs
    show_logs_summary

    echo -e "${BLUE}================================================${NC}"
    echo -e "${GREEN}Environment check completed!${NC}"
    echo -e "${BLUE}================================================${NC}"
}

# Show help
show_help() {
    echo "Usage: $0 [container_name]"
    echo ""
    echo "Check environment variables and configuration of Docker container"
    echo ""
    echo "Arguments:"
    echo "  container_name    Name of the container (default: ethereum-scheduler-app)"
    echo ""
    echo "Examples:"
    echo "  $0                                    # Check default container"
    echo "  $0 ethereum-scheduler-app             # Check specific container"
    echo "  $0 my-custom-container                # Check custom container"
}

# Handle arguments
case "${1:-}" in
    -h|--help)
        show_help
        exit 0
        ;;
    *)
        main
        ;;
esac