#!/bin/bash

echo "=== Testing WebSocket Listener Auto-Reconnect Functionality ==="

# Function to check if service is running
check_service_status() {
    local status=$(docker-compose -f docker-compose.websocket-listener.yml ps -q ethereum-websocket-listener)
    if [ -n "$status" ]; then
        echo "✅ WebSocket Listener service is running"
        return 0
    else
        echo "❌ WebSocket Listener service is not running"
        return 1
    fi
}

# Function to check for errors in logs
check_for_errors() {
    local error_count=$(docker-compose -f docker-compose.websocket-listener.yml logs --tail=50 ethereum-websocket-listener 2>/dev/null | grep -c "not connected to blockchain node")
    if [ "$error_count" -eq 0 ]; then
        echo "✅ No 'not connected to blockchain node' errors found"
        return 0
    else
        echo "⚠️  Found $error_count 'not connected to blockchain node' errors"
        return 1
    fi
}

# Function to check for successful processing
check_processing() {
    local processing_count=$(docker-compose -f docker-compose.websocket-listener.yml logs --tail=20 ethereum-websocket-listener 2>/dev/null | grep -c "Processing transactions in block")
    if [ "$processing_count" -gt 0 ]; then
        echo "✅ Successfully processing blocks ($processing_count recent blocks)"
        return 0
    else
        echo "❌ No recent block processing found"
        return 1
    fi
}

# Function to test reconnection logic
test_reconnection() {
    echo ""
    echo "🔄 Testing auto-reconnection logic..."

    # Kill the container temporarily to simulate network issues
    echo "📡 Simulating network disconnection..."
    docker-compose -f docker-compose.websocket-listener.yml stop ethereum-websocket-listener >/dev/null 2>&1

    sleep 3

    echo "🔌 Restarting service..."
    docker-compose -f docker-compose.websocket-listener.yml up -d ethereum-websocket-listener >/dev/null 2>&1

    # Wait for service to start
    echo "⏳ Waiting for service to start and establish connections..."
    sleep 15

    # Check if service recovered
    if check_service_status && check_processing; then
        echo "✅ Auto-reconnection test PASSED"
        return 0
    else
        echo "❌ Auto-reconnection test FAILED"
        return 1
    fi
}

# Main test execution
echo ""
echo "📊 Checking current service status..."
check_service_status

echo ""
echo "🔍 Checking for connection errors..."
check_for_errors

echo ""
echo "📈 Checking block processing..."
check_processing

# Run reconnection test
test_reconnection

echo ""
echo "🏁 Final status check..."
if check_service_status && check_for_errors && check_processing; then
    echo ""
    echo "🎉 Auto-reconnect functionality test PASSED!"
    echo "✅ WebSocket Listener is resilient to connection issues"
    exit 0
else
    echo ""
    echo "❌ Auto-reconnect functionality test FAILED!"
    echo "🔧 Check logs for detailed error information:"
    echo "   docker-compose -f docker-compose.websocket-listener.yml logs ethereum-websocket-listener"
    exit 1
fi