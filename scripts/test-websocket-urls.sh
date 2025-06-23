#!/bin/bash

# Test different WebSocket URLs for stability

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

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# WebSocket URLs to test
declare -A WEBSOCKET_URLS=(
    ["Infura"]="wss://mainnet.infura.io/ws/v3/fc066db3e5254dd88e0890320478bc75"
    ["Alchemy"]="wss://eth-mainnet.ws.alchemyapi.io/ws/x6DMAG9Zx4vOWVlS3Dov4"
    ["QuickNode"]="wss://docs-demo.quiknode.pro/"
    ["Ankr"]="wss://rpc.ankr.com/eth/ws"
)

test_websocket_url() {
    local name="$1"
    local url="$2"
    local timeout=30
    
    print_info "Testing $name WebSocket: $url"
    
    # Create a temporary test script
    cat > /tmp/ws_test.js << EOF
const WebSocket = require('ws');

const ws = new WebSocket('$url');
let messageCount = 0;
let subscriptionId = null;

const timeout = setTimeout(() => {
    console.log('TIMEOUT: No response within $timeout seconds');
    process.exit(1);
}, ${timeout}000);

ws.on('open', function open() {
    console.log('CONNECTED: WebSocket connection established');
    
    // Subscribe to new heads
    const subscribeMsg = {
        jsonrpc: '2.0',
        id: 1,
        method: 'eth_subscribe',
        params: ['newHeads']
    };
    
    ws.send(JSON.stringify(subscribeMsg));
});

ws.on('message', function message(data) {
    messageCount++;
    const msg = JSON.parse(data);
    
    if (msg.result && !subscriptionId) {
        subscriptionId = msg.result;
        console.log('SUBSCRIBED: Subscription ID:', subscriptionId);
    } else if (msg.method === 'eth_subscription') {
        const blockNumber = msg.params.result.number;
        console.log('BLOCK_RECEIVED: Block', parseInt(blockNumber, 16));
        
        if (messageCount >= 3) {
            console.log('SUCCESS: Received', messageCount, 'messages');
            clearTimeout(timeout);
            process.exit(0);
        }
    }
});

ws.on('error', function error(err) {
    console.log('ERROR:', err.message);
    clearTimeout(timeout);
    process.exit(1);
});

ws.on('close', function close() {
    console.log('CLOSED: WebSocket connection closed');
    clearTimeout(timeout);
    process.exit(1);
});
EOF

    # Run the test
    if command -v node >/dev/null 2>&1; then
        if node /tmp/ws_test.js 2>&1; then
            print_success "$name WebSocket is working!"
            return 0
        else
            print_error "$name WebSocket failed!"
            return 1
        fi
    else
        print_warning "Node.js not found, skipping $name test"
        return 2
    fi
}

# Test with Go script instead
test_websocket_go() {
    local name="$1"
    local url="$2"
    
    print_info "Testing $name WebSocket with Go: $url"
    
    # Set the URL and run our Go test
    export ETHEREUM_WS_URL="$url"
    
    # Run the test with timeout
    if timeout 45s go run test-websocket.go 2>&1 | head -20; then
        print_success "$name WebSocket test completed"
        return 0
    else
        print_error "$name WebSocket test failed or timed out"
        return 1
    fi
}

main() {
    print_info "Testing WebSocket URLs for Ethereum block subscriptions"
    echo ""
    
    local best_url=""
    local best_name=""
    
    for name in "${!WEBSOCKET_URLS[@]}"; do
        url="${WEBSOCKET_URLS[$name]}"
        echo ""
        echo "========================================"
        
        if test_websocket_go "$name" "$url"; then
            if [ -z "$best_url" ]; then
                best_url="$url"
                best_name="$name"
            fi
        fi
        
        echo "========================================"
        sleep 2
    done
    
    echo ""
    if [ -n "$best_url" ]; then
        print_success "Recommended WebSocket URL: $best_name"
        print_info "URL: $best_url"
        echo ""
        print_info "To use this URL, update your .env file:"
        echo "ETHEREUM_WS_URL=$best_url"
    else
        print_error "No working WebSocket URLs found!"
        print_warning "Consider using polling mode instead:"
        echo "SCHEDULER_MODE=polling"
    fi
}

# Change to scripts directory
cd "$(dirname "$0")"

main "$@"
