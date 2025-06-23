#!/bin/bash

# Stable Scheduler Runner with Real-time Monitoring
# This script runs the scheduler and provides live statistics

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
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

print_stat() {
    echo -e "${PURPLE}[STAT]${NC} $1"
}

# Get latest block from network
get_latest_block() {
    curl -s -X POST \
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
" 2>/dev/null || echo "0"
}

# Monitor scheduler
monitor_scheduler() {
    local log_file="scheduler_$(date +%Y%m%d_%H%M%S).log"
    local start_time=$(date +%s)
    local blocks_processed=0
    local last_processed_block=""
    local scheduler_pid=""
    
    print_info "Starting Ethereum Block Scheduler with monitoring..."
    print_info "Log file: $log_file"
    print_info "Press Ctrl+C to stop"
    echo ""
    
    # Start scheduler
    print_info "Starting scheduler..."
    go run cmd/schedulers/main.go > "$log_file" 2>&1 &
    scheduler_pid=$!
    
    print_success "Scheduler started with PID: $scheduler_pid"
    echo ""
    
    # Wait a bit for startup
    sleep 5
    
    # Monitor loop
    while true; do
        # Check if scheduler is still running
        if ! kill -0 $scheduler_pid 2>/dev/null; then
            print_error "Scheduler process died!"
            print_info "Check log file: $log_file"
            break
        fi
        
        # Get current stats
        local current_time=$(date +%s)
        local uptime=$((current_time - start_time))
        local latest_block=$(get_latest_block)
        
        # Count blocks processed from log
        if [ -f "$log_file" ]; then
            local new_blocks=$(grep -c "Block processed successfully" "$log_file" 2>/dev/null || echo "0")
            if [ "$new_blocks" -gt "$blocks_processed" ]; then
                blocks_processed=$new_blocks
                # Get the latest processed block
                last_processed_block=$(grep "Block processed successfully" "$log_file" | tail -1 | grep -o 'block_number":[0-9]*' | cut -d':' -f2 2>/dev/null || echo "unknown")
            fi
            
            # Check for errors
            local error_count=$(grep -c "ERROR" "$log_file" 2>/dev/null || echo "0")
            local warning_count=$(grep -c "WARN" "$log_file" 2>/dev/null || echo "0")
        fi
        
        # Calculate lag
        local block_lag="unknown"
        if [ "$latest_block" != "0" ] && [ "$last_processed_block" != "unknown" ] && [ "$last_processed_block" != "" ]; then
            block_lag=$((latest_block - last_processed_block))
        fi
        
        # Calculate processing rate
        local blocks_per_hour=0
        if [ "$uptime" -gt 0 ] && [ "$blocks_processed" -gt 0 ]; then
            blocks_per_hour=$(echo "scale=1; $blocks_processed * 3600 / $uptime" | bc 2>/dev/null || echo "0")
        fi
        
        # Display stats
        clear
        echo -e "${CYAN}╔══════════════════════════════════════════════════════════════╗${NC}"
        echo -e "${CYAN}║               ETHEREUM BLOCK SCHEDULER - LIVE               ║${NC}"
        echo -e "${CYAN}╠══════════════════════════════════════════════════════════════╣${NC}"
        echo -e "${CYAN}║${NC} Status: ${GREEN}RUNNING${NC} (PID: $scheduler_pid)"
        echo -e "${CYAN}║${NC} Uptime: ${GREEN}$(date -u -d @$uptime +%H:%M:%S)${NC}"
        echo -e "${CYAN}║${NC} Time: ${GREEN}$(date '+%Y-%m-%d %H:%M:%S')${NC}"
        echo -e "${CYAN}╠══════════════════════════════════════════════════════════════╣${NC}"
        echo -e "${CYAN}║${NC} Network Block: ${YELLOW}$latest_block${NC}"
        echo -e "${CYAN}║${NC} Processed Block: ${GREEN}$last_processed_block${NC}"
        echo -e "${CYAN}║${NC} Block Lag: ${RED}$block_lag${NC} blocks"
        echo -e "${CYAN}║${NC} Blocks Processed: ${PURPLE}$blocks_processed${NC}"
        echo -e "${CYAN}║${NC} Processing Rate: ${BLUE}$blocks_per_hour${NC} blocks/hour"
        echo -e "${CYAN}╠══════════════════════════════════════════════════════════════╣${NC}"
        echo -e "${CYAN}║${NC} Errors: ${RED}$error_count${NC} | Warnings: ${YELLOW}$warning_count${NC}"
        echo -e "${CYAN}║${NC} Log File: $log_file"
        echo -e "${CYAN}╠══════════════════════════════════════════════════════════════╣${NC}"
        
        # Show recent activity
        echo -e "${CYAN}║${NC} Recent Activity:"
        if [ -f "$log_file" ]; then
            tail -5 "$log_file" | grep -E "(Block processed successfully|ERROR|WARN)" | tail -3 | while IFS= read -r line; do
                # Extract timestamp and message
                local timestamp=$(echo "$line" | cut -d$'\t' -f1 | cut -d'T' -f2 | cut -d'+' -f1)
                local message=$(echo "$line" | cut -d$'\t' -f3- | head -c 50)
                if [ ${#message} -eq 50 ]; then
                    message="${message}..."
                fi
                echo -e "${CYAN}║${NC} ${timestamp}: ${message}"
            done
        fi
        
        echo -e "${CYAN}╚══════════════════════════════════════════════════════════════╝${NC}"
        
        # Status indicators
        if [ "$block_lag" != "unknown" ] && [ "$block_lag" -gt 50 ]; then
            print_warning "High block lag detected!"
        fi
        
        if [ "$error_count" -gt 0 ]; then
            print_error "Errors detected in log!"
        fi
        
        if [ "$blocks_processed" -gt 0 ]; then
            print_success "Scheduler is processing blocks successfully!"
        fi
        
        # Wait before next update
        sleep 10
    done
    
    # Cleanup
    if [ -n "$scheduler_pid" ] && kill -0 $scheduler_pid 2>/dev/null; then
        print_info "Stopping scheduler..."
        kill $scheduler_pid
        sleep 2
        if kill -0 $scheduler_pid 2>/dev/null; then
            kill -9 $scheduler_pid
        fi
    fi
    
    print_info "Scheduler monitoring stopped"
    print_info "Log file saved: $log_file"
}

# Trap Ctrl+C
trap 'echo -e "\n${YELLOW}Stopping monitoring...${NC}"; exit 0' INT

# Check if we're in the right directory
if [ ! -f "cmd/schedulers/main.go" ]; then
    print_error "Please run this script from the project root directory"
    exit 1
fi

# Check dependencies
if ! command -v bc >/dev/null 2>&1; then
    print_warning "bc not found - processing rate calculation will be disabled"
fi

# Set optimal configuration
export SCHEDULER_MODE=polling
export LOG_LEVEL=info
export SCHEDULER_POLLING_INTERVAL=3s
export BATCH_SIZE=1
export CONCURRENT_WORKERS=2
export ETHEREUM_RATE_LIMIT=1s

print_info "Configuration:"
echo "  SCHEDULER_MODE: $SCHEDULER_MODE"
echo "  SCHEDULER_POLLING_INTERVAL: $SCHEDULER_POLLING_INTERVAL"
echo "  BATCH_SIZE: $BATCH_SIZE"
echo "  CONCURRENT_WORKERS: $CONCURRENT_WORKERS"
echo ""

# Start monitoring
monitor_scheduler
