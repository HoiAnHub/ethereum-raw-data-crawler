#!/bin/bash

# Test Docker Rate Limiting Configuration
# This script helps verify that Docker mode uses the same rate limiting as dev mode

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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

# Function to check environment variables
check_env_vars() {
    print_info "Checking environment variables from .env file..."
    
    if [ ! -f ".env" ]; then
        print_error ".env file not found!"
        exit 1
    fi
    
    source .env
    
    echo "Current .env configuration:"
    echo "  ETHEREUM_RATE_LIMIT: ${ETHEREUM_RATE_LIMIT:-not set}"
    echo "  CONCURRENT_WORKERS: ${CONCURRENT_WORKERS:-not set}"
    echo "  BATCH_SIZE: ${BATCH_SIZE:-not set}"
    echo "  RETRY_DELAY: ${RETRY_DELAY:-not set}"
    echo ""
}

# Function to show Docker configuration
show_docker_config() {
    print_info "Docker Compose configuration for scheduler:"
    echo "Environment variables that will be used in Docker:"
    echo "  ETHEREUM_RATE_LIMIT: ${ETHEREUM_RATE_LIMIT:-1s} (from .env or default)"
    echo "  ETHEREUM_REQUEST_TIMEOUT: ${ETHEREUM_REQUEST_TIMEOUT:-120s} (from .env or default)"
    echo "  ETHEREUM_SKIP_RECEIPTS: ${ETHEREUM_SKIP_RECEIPTS:-true} (from .env or default)"
    echo "  CONCURRENT_WORKERS: 2 (hardcoded in docker-compose.scheduler.yml)"
    echo "  BATCH_SIZE: 1 (hardcoded in docker-compose.scheduler.yml)"
    echo "  RETRY_DELAY: 3s (hardcoded in docker-compose.scheduler.yml)"
    echo ""
}

# Function to calculate request rate
calculate_rate() {
    local workers=$1
    local rate_limit=$2
    
    print_info "Calculating request rate:"
    echo "  Workers: $workers"
    echo "  Rate limit: $rate_limit"
    
    # Convert rate limit to seconds for calculation
    if [[ $rate_limit == *"ms" ]]; then
        local ms=${rate_limit%ms}
        local seconds=$(echo "scale=3; $ms / 1000" | bc)
    elif [[ $rate_limit == *"s" ]]; then
        local seconds=${rate_limit%s}
    else
        local seconds=$rate_limit
    fi
    
    local requests_per_second=$(echo "scale=2; $workers / $seconds" | bc)
    echo "  Estimated requests per second: $requests_per_second"
    
    if (( $(echo "$requests_per_second > 8" | bc -l) )); then
        print_warning "Request rate may be too high for Infura free tier (10 req/s limit)"
        print_warning "Consider reducing CONCURRENT_WORKERS or increasing ETHEREUM_RATE_LIMIT"
    else
        print_success "Request rate should be safe for Infura free tier"
    fi
    echo ""
}

# Function to test Docker build
test_docker_build() {
    print_info "Testing Docker build..."
    if docker-compose -f docker-compose.scheduler.yml build --no-cache; then
        print_success "Docker build successful"
    else
        print_error "Docker build failed"
        exit 1
    fi
    echo ""
}

# Function to show comparison
show_comparison() {
    print_info "Configuration comparison:"
    echo ""
    echo "DEV MODE (.env file):"
    echo "  CONCURRENT_WORKERS: ${CONCURRENT_WORKERS:-2}"
    echo "  ETHEREUM_RATE_LIMIT: ${ETHEREUM_RATE_LIMIT:-1s}"
    echo "  BATCH_SIZE: ${BATCH_SIZE:-1}"
    echo "  RETRY_DELAY: ${RETRY_DELAY:-3s}"
    echo ""
    echo "DOCKER MODE (docker-compose.scheduler.yml):"
    echo "  CONCURRENT_WORKERS: 2 (fixed)"
    echo "  ETHEREUM_RATE_LIMIT: ${ETHEREUM_RATE_LIMIT:-1s} (from .env)"
    echo "  BATCH_SIZE: 1 (fixed)"
    echo "  RETRY_DELAY: 3s (fixed)"
    echo ""
    
    calculate_rate 2 "${ETHEREUM_RATE_LIMIT:-1s}"
}

# Main execution
main() {
    print_info "Docker Rate Limiting Configuration Test"
    echo "========================================"
    echo ""
    
    check_env_vars
    show_docker_config
    show_comparison
    
    print_info "To test the configuration:"
    echo "1. Run: ./scripts/run-scheduler.sh docker"
    echo "2. Monitor logs: ./scripts/run-scheduler.sh logs --follow"
    echo "3. Watch for 429 errors in the logs"
    echo ""
    
    print_info "If you still get 429 errors, try:"
    echo "1. Increase ETHEREUM_RATE_LIMIT in .env (e.g., 2s or 3s)"
    echo "2. Or reduce CONCURRENT_WORKERS in docker-compose.scheduler.yml"
    echo ""
}

# Check if bc is available for calculations
if ! command -v bc &> /dev/null; then
    print_warning "bc command not found. Install it for rate calculations: brew install bc"
fi

main "$@"
