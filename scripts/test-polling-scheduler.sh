#!/bin/bash

# Quick test for polling scheduler
set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Go to project root (find ethereum-raw-data-crawler directory)
while [[ ! -f "cmd/schedulers/main.go" && "$(pwd)" != "/" ]]; do
    cd ..
done

if [[ ! -f "cmd/schedulers/main.go" ]]; then
    print_error "Could not find cmd/schedulers/main.go - please run from project directory"
    exit 1
fi

print_info "Testing Polling Scheduler..."
print_info "Current directory: $(pwd)"

# Ensure we're in polling mode
export SCHEDULER_MODE=polling
export LOG_LEVEL=info
export SCHEDULER_POLLING_INTERVAL=5s

print_info "Configuration:"
echo "  SCHEDULER_MODE: $SCHEDULER_MODE"
echo "  LOG_LEVEL: $LOG_LEVEL"
echo "  SCHEDULER_POLLING_INTERVAL: $SCHEDULER_POLLING_INTERVAL"
echo ""

# Get current latest block
print_info "Getting current latest block from network..."
LATEST_BLOCK=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
    "https://mainnet.infura.io/v3/fc066db3e5254dd88e0890320478bc75" | \
    python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    if 'result' in data:
        print(int(data['result'], 16))
    else:
        print('0')
except:
    print('0')
" 2>/dev/null || echo "unknown")

print_info "Latest block on network: $LATEST_BLOCK"
echo ""

# Start scheduler and monitor for 60 seconds
print_info "Starting scheduler for 60 seconds..."
print_info "Press Ctrl+C to stop early"
echo ""

# Create a temporary log file
LOG_FILE="/tmp/scheduler_test.log"

# Start scheduler in background
go run cmd/schedulers/main.go > "$LOG_FILE" 2>&1 &
SCHEDULER_PID=$!

print_success "Scheduler started with PID: $SCHEDULER_PID"

# Monitor for 60 seconds
SECONDS=0
BLOCKS_PROCESSED=0
LAST_PROCESSED_BLOCK=""

while [ $SECONDS -lt 60 ]; do
    sleep 5

    # Check if scheduler is still running
    if ! kill -0 $SCHEDULER_PID 2>/dev/null; then
        print_error "Scheduler process died!"
        break
    fi

    # Count blocks processed
    if [ -f "$LOG_FILE" ]; then
        NEW_BLOCKS=$(grep -c "Block processed successfully" "$LOG_FILE" 2>/dev/null || echo "0")
        if [ "$NEW_BLOCKS" -gt "$BLOCKS_PROCESSED" ]; then
            BLOCKS_PROCESSED=$NEW_BLOCKS
            # Get the latest processed block
            LAST_PROCESSED_BLOCK=$(grep "Block processed successfully" "$LOG_FILE" | tail -1 | grep -o 'block_number":[0-9]*' | cut -d':' -f2 || echo "unknown")
            print_success "Blocks processed so far: $BLOCKS_PROCESSED (latest: $LAST_PROCESSED_BLOCK)"
        fi

        # Check for errors
        ERROR_COUNT=$(grep -c "ERROR" "$LOG_FILE" 2>/dev/null || echo "0")
        if [ "$ERROR_COUNT" -gt 0 ]; then
            print_error "Found $ERROR_COUNT errors in log"
        fi
    fi

    echo -n "."
done

echo ""
print_info "Test completed after 60 seconds"

# Stop scheduler
if kill -0 $SCHEDULER_PID 2>/dev/null; then
    print_info "Stopping scheduler..."
    kill $SCHEDULER_PID
    sleep 2
    if kill -0 $SCHEDULER_PID 2>/dev/null; then
        kill -9 $SCHEDULER_PID
    fi
fi

# Final results
echo ""
print_info "=== TEST RESULTS ==="
print_info "Total blocks processed: $BLOCKS_PROCESSED"
print_info "Latest processed block: $LAST_PROCESSED_BLOCK"
print_info "Network latest block: $LATEST_BLOCK"

if [ "$BLOCKS_PROCESSED" -gt 0 ]; then
    print_success "✅ Scheduler is working! Processed $BLOCKS_PROCESSED blocks"

    if [ "$LAST_PROCESSED_BLOCK" != "unknown" ] && [ "$LATEST_BLOCK" != "unknown" ] && [ "$LATEST_BLOCK" != "0" ]; then
        BLOCK_LAG=$((LATEST_BLOCK - LAST_PROCESSED_BLOCK))
        print_info "Block lag: $BLOCK_LAG blocks"

        if [ "$BLOCK_LAG" -lt 10 ]; then
            print_success "✅ Low lag - scheduler is keeping up well!"
        elif [ "$BLOCK_LAG" -lt 50 ]; then
            print_info "⚠️  Moderate lag - scheduler is working but slightly behind"
        else
            print_error "❌ High lag - scheduler may need optimization"
        fi
    fi
else
    print_error "❌ No blocks processed - there may be an issue"
fi

# Show recent log entries
if [ -f "$LOG_FILE" ]; then
    echo ""
    print_info "Recent log entries:"
    tail -10 "$LOG_FILE"
fi

# Cleanup
rm -f "$LOG_FILE"

echo ""
print_info "Test completed. If scheduler is working well, you can run it normally with:"
echo "  go run cmd/schedulers/main.go"
