#!/bin/bash

# Quick environment variables check for Docker containers

CONTAINER="${1:-ethereum-scheduler-app}"

echo "🔍 Quick Environment Check for: $CONTAINER"
echo "============================================"

# Check if container is running
if ! docker ps --format "{{.Names}}" | grep -q "^$CONTAINER$"; then
    echo "❌ Container '$CONTAINER' is not running"
    echo "Available containers:"
    docker ps --format "table {{.Names}}\t{{.Status}}"
    exit 1
fi

echo "✅ Container is running"
echo ""

# Critical variables to check
echo "📋 Critical Variables:"
echo "----------------------"

# Function to check and display env var
check_var() {
    local var_name="$1"
    local value=$(docker exec "$CONTAINER" sh -c "echo \$$var_name" 2>/dev/null)

    if [ -n "$value" ]; then
        # Mask sensitive URLs
        if [[ "$var_name" == *"URL"* ]] || [[ "$var_name" == *"URI"* ]]; then
            masked_value=$(echo "$value" | sed -E 's/(\/\/[^:@]*:)[^@]*(@)/\1***\2/g' | sed -E 's/(\/v3\/)[^?&]*/\1***/')
            echo "✅ $var_name = $masked_value"
        else
            echo "✅ $var_name = $value"
        fi
    else
        echo "❌ $var_name = NOT SET"
    fi
}

# Check critical variables
check_var "ETHEREUM_RPC_URL"
check_var "ETHEREUM_WS_URL"
check_var "MONGO_URI"
check_var "APP_ENV"
check_var "CRAWLER_USE_UPSERT"
check_var "CRAWLER_UPSERT_FALLBACK"
check_var "SCHEDULER_MODE"

echo ""
echo "🔧 Performance Settings:"
echo "------------------------"
check_var "BATCH_SIZE"
check_var "CONCURRENT_WORKERS"
check_var "ETHEREUM_RATE_LIMIT"

echo ""
echo "📊 Container Status:"
echo "-------------------"
echo "Status: $(docker inspect --format='{{.State.Status}}' "$CONTAINER")"
echo "Health: $(docker inspect --format='{{.State.Health.Status}}' "$CONTAINER" 2>/dev/null || echo "No health check")"
echo "Uptime: $(docker inspect --format='{{.State.StartedAt}}' "$CONTAINER")"

echo ""
echo "📝 Latest Logs (Last 5 lines):"
echo "------------------------------"
docker logs --tail=5 "$CONTAINER" 2>/dev/null || echo "Cannot read logs"

echo ""
echo "✨ Quick check completed!"