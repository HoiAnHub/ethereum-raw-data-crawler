#!/bin/bash

# Ethereum Block Scheduler Monitor Script
# This script monitors the scheduler and provides real-time statistics

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Function to print colored output
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

# Function to get latest block from Ethereum
get_latest_block() {
    local rpc_url="${ETHEREUM_RPC_URL:-https://mainnet.infura.io/v3/fc066db3e5254dd88e0890320478bc75}"
    
    curl -s -X POST \
        -H "Content-Type: application/json" \
        -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
        "$rpc_url" | \
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
"
}

# Function to get processed blocks from MongoDB
get_processed_blocks() {
    local mongo_uri="${MONGO_URI:-mongodb+srv://haitranwang:eURhdPjFc10NGyDR@cluster0.kzyty5l.mongodb.net/ethereum_raw_data}"
    
    # Use mongosh to get the latest processed block
    echo 'db.blocks.find().sort({number: -1}).limit(1).forEach(function(doc) { print(doc.number); })' | \
    mongosh "$mongo_uri" --quiet 2>/dev/null | tail -1 || echo "0"
}

# Function to monitor scheduler process
monitor_scheduler() {
    local scheduler_pid=""
    local start_time=$(date +%s)
    local last_block_check=0
    local last_processed_block=0
    local blocks_processed=0
    
    print_info "Starting Ethereum Block Scheduler Monitor"
    print_info "Press Ctrl+C to stop monitoring"
    echo ""
    
    # Start scheduler in background
    print_info "Starting scheduler..."
    go run cmd/schedulers/main.go > scheduler.log 2>&1 &
    scheduler_pid=$!
    
    print_success "Scheduler started with PID: $scheduler_pid"
    echo ""
    
    # Monitor loop
    while true; do
        # Check if scheduler is still running
        if ! kill -0 $scheduler_pid 2>/dev/null; then
            print_error "Scheduler process died! Restarting..."
            go run cmd/schedulers/main.go > scheduler.log 2>&1 &
            scheduler_pid=$!
            print_success "Scheduler restarted with PID: $scheduler_pid"
        fi
        
        # Get current stats
        local current_time=$(date +%s)
        local uptime=$((current_time - start_time))
        local latest_block=$(get_latest_block)
        local processed_block=$(get_processed_blocks)
        
        # Calculate blocks processed
        if [ "$processed_block" != "0" ] && [ "$processed_block" -gt "$last_processed_block" ]; then
            blocks_processed=$((blocks_processed + processed_block - last_processed_block))
            last_processed_block=$processed_block
        fi
        
        # Calculate lag
        local block_lag=0
        if [ "$latest_block" != "0" ] && [ "$processed_block" != "0" ]; then
            block_lag=$((latest_block - processed_block))
        fi
        
        # Display stats
        clear
        echo -e "${CYAN}╔══════════════════════════════════════════════════════════════╗${NC}"
        echo -e "${CYAN}║                 ETHEREUM BLOCK SCHEDULER MONITOR            ║${NC}"
        echo -e "${CYAN}╠══════════════════════════════════════════════════════════════╣${NC}"
        echo -e "${CYAN}║${NC} Scheduler PID: ${GREEN}$scheduler_pid${NC}"
        echo -e "${CYAN}║${NC} Uptime: ${GREEN}$(date -u -d @$uptime +%H:%M:%S)${NC}"
        echo -e "${CYAN}║${NC} Current Time: ${GREEN}$(date '+%Y-%m-%d %H:%M:%S')${NC}"
        echo -e "${CYAN}╠══════════════════════════════════════════════════════════════╣${NC}"
        echo -e "${CYAN}║${NC} Latest Block (Network): ${YELLOW}$latest_block${NC}"
        echo -e "${CYAN}║${NC} Processed Block: ${GREEN}$processed_block${NC}"
        echo -e "${CYAN}║${NC} Block Lag: ${RED}$block_lag${NC} blocks"
        echo -e "${CYAN}║${NC} Blocks Processed: ${PURPLE}$blocks_processed${NC}"
        echo -e "${CYAN}╠══════════════════════════════════════════════════════════════╣${NC}"
        
        # Show recent log entries
        echo -e "${CYAN}║${NC} Recent Log Entries:"
        if [ -f "scheduler.log" ]; then
            tail -5 scheduler.log | while IFS= read -r line; do
                # Truncate long lines
                if [ ${#line} -gt 58 ]; then
                    line="${line:0:55}..."
                fi
                echo -e "${CYAN}║${NC} ${line}"
            done
        fi
        
        echo -e "${CYAN}╚══════════════════════════════════════════════════════════════╝${NC}"
        
        # Check for issues
        if [ "$block_lag" -gt 10 ]; then
            print_warning "High block lag detected: $block_lag blocks"
        fi
        
        if [ -f "scheduler.log" ]; then
            local error_count=$(grep -c "ERROR" scheduler.log 2>/dev/null || echo "0")
            if [ "$error_count" -gt 0 ]; then
                print_warning "Found $error_count errors in log"
            fi
        fi
        
        # Wait before next check
        sleep 5
    done
}

# Function to show scheduler logs
show_logs() {
    if [ -f "scheduler.log" ]; then
        print_info "Showing scheduler logs (press Ctrl+C to exit):"
        tail -f scheduler.log
    else
        print_error "No scheduler log file found"
    fi
}

# Function to show usage
show_usage() {
    echo "Usage: $0 [COMMAND]"
    echo ""
    echo "Commands:"
    echo "  monitor     Monitor scheduler with real-time stats (default)"
    echo "  logs        Show scheduler logs"
    echo "  stats       Show current statistics"
    echo "  help        Show this help message"
}

# Function to show current stats
show_stats() {
    print_info "Getting current statistics..."
    
    local latest_block=$(get_latest_block)
    local processed_block=$(get_processed_blocks)
    local block_lag=0
    
    if [ "$latest_block" != "0" ] && [ "$processed_block" != "0" ]; then
        block_lag=$((latest_block - processed_block))
    fi
    
    echo ""
    print_stat "Latest Block (Network): $latest_block"
    print_stat "Processed Block: $processed_block"
    print_stat "Block Lag: $block_lag blocks"
    
    if [ "$block_lag" -eq 0 ]; then
        print_success "Scheduler is up to date!"
    elif [ "$block_lag" -lt 5 ]; then
        print_info "Scheduler is slightly behind"
    else
        print_warning "Scheduler is significantly behind"
    fi
}

# Trap Ctrl+C
trap 'echo -e "\n${YELLOW}Monitoring stopped${NC}"; exit 0' INT

# Main script logic
case "${1:-monitor}" in
    "monitor")
        monitor_scheduler
        ;;
    "logs")
        show_logs
        ;;
    "stats")
        show_stats
        ;;
    "help"|"--help"|"-h")
        show_usage
        ;;
    *)
        print_error "Unknown command: $1"
        show_usage
        exit 1
        ;;
esac
