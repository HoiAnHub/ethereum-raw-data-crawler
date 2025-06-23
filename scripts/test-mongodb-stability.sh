#!/bin/bash

# MongoDB Connection Stability Test
# This script tests MongoDB connection stability over time

set -e

# Configuration
MONGO_URI="${MONGO_URI:-mongodb://admin:password@localhost:27017/ethereum_raw_data?authSource=admin}"
TEST_DURATION="${TEST_DURATION:-300}" # 5 minutes default
TEST_INTERVAL="${TEST_INTERVAL:-5}"   # 5 seconds default
CONTAINER_NAME="ethereum-scheduler-mongodb"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Counters
total_tests=0
successful_tests=0
failed_tests=0
start_time=$(date +%s)

# Function to log with timestamp
log() {
    echo -e "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

# Function to test MongoDB connection
test_connection() {
    local test_num=$1
    local result
    
    log "${BLUE}Test #$test_num: Testing MongoDB connection...${NC}"
    
    # Test basic connection
    if result=$(docker exec "$CONTAINER_NAME" mongosh "$MONGO_URI" --eval "db.adminCommand('ping').ok" --quiet 2>&1); then
        if [ "$result" = "1" ]; then
            log "${GREEN}✓ Connection test passed${NC}"
            return 0
        else
            log "${RED}✗ Connection test failed: Invalid response ($result)${NC}"
            return 1
        fi
    else
        log "${RED}✗ Connection test failed: $result${NC}"
        return 1
    fi
}

# Function to test database operations
test_database_operations() {
    local test_num=$1
    local test_doc="{\"test_id\": $test_num, \"timestamp\": new Date(), \"data\": \"stability_test\"}"
    
    log "${BLUE}Test #$test_num: Testing database operations...${NC}"
    
    # Test write operation
    if docker exec "$CONTAINER_NAME" mongosh "$MONGO_URI" --eval "db.stability_test.insertOne($test_doc)" --quiet >/dev/null 2>&1; then
        log "${GREEN}✓ Write operation successful${NC}"
    else
        log "${RED}✗ Write operation failed${NC}"
        return 1
    fi
    
    # Test read operation
    if docker exec "$CONTAINER_NAME" mongosh "$MONGO_URI" --eval "db.stability_test.findOne({test_id: $test_num})" --quiet >/dev/null 2>&1; then
        log "${GREEN}✓ Read operation successful${NC}"
    else
        log "${RED}✗ Read operation failed${NC}"
        return 1
    fi
    
    # Test delete operation
    if docker exec "$CONTAINER_NAME" mongosh "$MONGO_URI" --eval "db.stability_test.deleteOne({test_id: $test_num})" --quiet >/dev/null 2>&1; then
        log "${GREEN}✓ Delete operation successful${NC}"
    else
        log "${RED}✗ Delete operation failed${NC}"
        return 1
    fi
    
    return 0
}

# Function to test connection pool
test_connection_pool() {
    local test_num=$1
    
    log "${BLUE}Test #$test_num: Testing connection pool...${NC}"
    
    # Get current connection count
    local conn_count
    if conn_count=$(docker exec "$CONTAINER_NAME" mongosh "$MONGO_URI" --eval "db.serverStatus().connections.current" --quiet 2>/dev/null); then
        log "${GREEN}✓ Connection pool test passed (current connections: $conn_count)${NC}"
        return 0
    else
        log "${RED}✗ Connection pool test failed${NC}"
        return 1
    fi
}

# Function to run comprehensive test
run_test() {
    local test_num=$1
    total_tests=$((total_tests + 1))
    
    log "${YELLOW}=== Running Test #$test_num ===${NC}"
    
    local test_start=$(date +%s)
    local test_passed=true
    
    # Run individual tests
    if ! test_connection "$test_num"; then
        test_passed=false
    fi
    
    if ! test_database_operations "$test_num"; then
        test_passed=false
    fi
    
    if ! test_connection_pool "$test_num"; then
        test_passed=false
    fi
    
    local test_end=$(date +%s)
    local test_duration=$((test_end - test_start))
    
    if [ "$test_passed" = true ]; then
        successful_tests=$((successful_tests + 1))
        log "${GREEN}✓ Test #$test_num completed successfully (${test_duration}s)${NC}"
    else
        failed_tests=$((failed_tests + 1))
        log "${RED}✗ Test #$test_num failed (${test_duration}s)${NC}"
    fi
    
    echo ""
}

# Function to show statistics
show_statistics() {
    local current_time=$(date +%s)
    local elapsed_time=$((current_time - start_time))
    local success_rate=0
    
    if [ $total_tests -gt 0 ]; then
        success_rate=$((successful_tests * 100 / total_tests))
    fi
    
    log "${YELLOW}=== Test Statistics ===${NC}"
    log "Elapsed time: ${elapsed_time}s"
    log "Total tests: $total_tests"
    log "Successful tests: ${GREEN}$successful_tests${NC}"
    log "Failed tests: ${RED}$failed_tests${NC}"
    log "Success rate: ${success_rate}%"
    
    # Show MongoDB container stats
    if docker ps --format "table {{.Names}}\t{{.Status}}" | grep -q "$CONTAINER_NAME"; then
        local container_status
        container_status=$(docker ps --format "{{.Status}}" --filter "name=$CONTAINER_NAME")
        log "Container status: ${GREEN}$container_status${NC}"
        
        # Show resource usage
        local stats
        stats=$(docker stats "$CONTAINER_NAME" --no-stream --format "table {{.CPUPerc}}\t{{.MemUsage}}" | tail -n 1)
        log "Resource usage: $stats"
    else
        log "Container status: ${RED}Not running${NC}"
    fi
}

# Function to cleanup test data
cleanup() {
    log "${YELLOW}Cleaning up test data...${NC}"
    docker exec "$CONTAINER_NAME" mongosh "$MONGO_URI" --eval "db.stability_test.drop()" --quiet >/dev/null 2>&1 || true
    log "${GREEN}Cleanup completed${NC}"
}

# Main function
main() {
    log "${YELLOW}Starting MongoDB stability test${NC}"
    log "Test duration: ${TEST_DURATION}s"
    log "Test interval: ${TEST_INTERVAL}s"
    log "MongoDB URI: $MONGO_URI"
    echo ""
    
    # Check if container is running
    if ! docker ps --format "table {{.Names}}" | grep -q "$CONTAINER_NAME"; then
        log "${RED}MongoDB container is not running. Please start it first.${NC}"
        exit 1
    fi
    
    # Wait for MongoDB to be ready
    log "Waiting for MongoDB to be ready..."
    local retries=0
    while ! docker exec "$CONTAINER_NAME" mongosh "$MONGO_URI" --eval "db.adminCommand('ping').ok" --quiet >/dev/null 2>&1; do
        retries=$((retries + 1))
        if [ $retries -gt 30 ]; then
            log "${RED}MongoDB is not responding after 30 attempts${NC}"
            exit 1
        fi
        sleep 2
    done
    
    log "${GREEN}MongoDB is ready, starting tests...${NC}"
    echo ""
    
    # Run tests
    local test_count=1
    local end_time=$((start_time + TEST_DURATION))
    
    while [ $(date +%s) -lt $end_time ]; do
        run_test $test_count
        test_count=$((test_count + 1))
        
        # Show intermediate statistics every 10 tests
        if [ $((test_count % 10)) -eq 1 ] && [ $test_count -gt 1 ]; then
            show_statistics
            echo ""
        fi
        
        sleep "$TEST_INTERVAL"
    done
    
    # Final statistics
    show_statistics
    
    # Cleanup
    cleanup
    
    # Exit with appropriate code
    if [ $failed_tests -eq 0 ]; then
        log "${GREEN}All tests passed successfully!${NC}"
        exit 0
    else
        log "${RED}Some tests failed. Check the logs above.${NC}"
        exit 1
    fi
}

# Handle script termination
cleanup_on_exit() {
    log "${YELLOW}Test interrupted, cleaning up...${NC}"
    cleanup
    show_statistics
    exit 1
}

trap cleanup_on_exit SIGINT SIGTERM

# Handle command line arguments
case "${1:-}" in
    --help)
        echo "Usage: $0 [options]"
        echo "Options:"
        echo "  --help              Show this help message"
        echo "Environment variables:"
        echo "  MONGO_URI          MongoDB connection URI (default: mongodb://admin:password@localhost:27017/ethereum_raw_data?authSource=admin)"
        echo "  TEST_DURATION      Test duration in seconds (default: 300)"
        echo "  TEST_INTERVAL      Interval between tests in seconds (default: 5)"
        ;;
    *)
        main
        ;;
esac
