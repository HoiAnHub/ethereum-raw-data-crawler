#!/bin/bash

# MongoDB Optimization Script for Ethereum Scheduler
# This script applies optimizations to MongoDB for better stability

set -e

CONTAINER_NAME="ethereum-scheduler-mongodb"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to log with timestamp
log() {
    echo -e "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

# Function to check if container is running
check_container() {
    if docker ps --format "table {{.Names}}" | grep -q "$CONTAINER_NAME"; then
        return 0
    else
        return 1
    fi
}

# Function to execute MongoDB command
exec_mongo_cmd() {
    local cmd="$1"
    docker exec "$CONTAINER_NAME" mongosh --eval "$cmd" --quiet
}

# Function to optimize MongoDB settings
optimize_mongodb() {
    log "${YELLOW}Applying MongoDB optimizations...${NC}"
    
    # Set profiling level to log slow operations
    log "Setting profiling level for slow operations..."
    exec_mongo_cmd "db.setProfilingLevel(1, { slowms: 1000 })"
    
    # Set write concern for better durability
    log "Configuring write concern..."
    exec_mongo_cmd "db.adminCommand({setDefaultRWConcern: 1, defaultWriteConcern: {w: 1, j: true}})"
    
    # Configure connection pool settings
    log "Optimizing connection pool..."
    exec_mongo_cmd "db.adminCommand({setParameter: 1, maxConns: 1000})"
    
    # Set journal commit interval
    log "Setting journal commit interval..."
    exec_mongo_cmd "db.adminCommand({setParameter: 1, journalCommitInterval: 100})"
    
    # Configure WiredTiger cache
    log "Configuring WiredTiger cache..."
    exec_mongo_cmd "db.adminCommand({setParameter: 1, wiredTigerEngineRuntimeConfig: 'cache_size=512MB'})"
    
    log "${GREEN}MongoDB optimizations applied successfully${NC}"
}

# Function to create additional indexes for better performance
create_performance_indexes() {
    log "${YELLOW}Creating performance indexes...${NC}"
    
    # Switch to ethereum_raw_data database
    local db_cmd="db = db.getSiblingDB('ethereum_raw_data');"
    
    # Create compound indexes for better query performance
    log "Creating compound indexes for blocks..."
    exec_mongo_cmd "${db_cmd} db.blocks.createIndex({network: 1, status: 1, number: 1}, {background: true})"
    exec_mongo_cmd "${db_cmd} db.blocks.createIndex({timestamp: 1, network: 1}, {background: true})"
    
    log "Creating compound indexes for transactions..."
    exec_mongo_cmd "${db_cmd} db.transactions.createIndex({blockNumber: 1, status: 1}, {background: true})"
    exec_mongo_cmd "${db_cmd} db.transactions.createIndex({from: 1, blockNumber: 1}, {background: true})"
    exec_mongo_cmd "${db_cmd} db.transactions.createIndex({to: 1, blockNumber: 1}, {background: true})"
    
    log "Creating indexes for metrics collections..."
    exec_mongo_cmd "${db_cmd} db.crawler_metrics.createIndex({network: 1, timestamp: -1}, {background: true})"
    exec_mongo_cmd "${db_cmd} db.system_health.createIndex({network: 1, status: 1, timestamp: -1}, {background: true})"
    
    log "${GREEN}Performance indexes created successfully${NC}"
}

# Function to configure MongoDB for better connection handling
configure_connection_handling() {
    log "${YELLOW}Configuring connection handling...${NC}"
    
    # Set connection timeout settings
    exec_mongo_cmd "db.adminCommand({setParameter: 1, connPoolMaxShardedConnsPerHost: 200})"
    exec_mongo_cmd "db.adminCommand({setParameter: 1, connPoolMaxConnsPerHost: 200})"
    
    # Configure heartbeat settings
    exec_mongo_cmd "db.adminCommand({setParameter: 1, replMonitorMaxFailedChecks: 5})"
    
    log "${GREEN}Connection handling configured successfully${NC}"
}

# Function to set up MongoDB monitoring
setup_monitoring() {
    log "${YELLOW}Setting up MongoDB monitoring...${NC}"
    
    # Enable command monitoring
    exec_mongo_cmd "db.adminCommand({setParameter: 1, logLevel: 1})"
    
    # Create monitoring user (if needed)
    local create_user_cmd="
    db = db.getSiblingDB('admin');
    try {
        db.createUser({
            user: 'monitor',
            pwd: 'monitor123',
            roles: [
                { role: 'clusterMonitor', db: 'admin' },
                { role: 'read', db: 'ethereum_raw_data' }
            ]
        });
        print('Monitoring user created');
    } catch(e) {
        if (e.code === 51003) {
            print('Monitoring user already exists');
        } else {
            throw e;
        }
    }
    "
    exec_mongo_cmd "$create_user_cmd"
    
    log "${GREEN}MongoDB monitoring setup completed${NC}"
}

# Function to display current MongoDB status
show_status() {
    log "${YELLOW}Current MongoDB Status:${NC}"
    
    echo "Server Status:"
    exec_mongo_cmd "db.serverStatus().connections"
    
    echo -e "\nDatabase Stats:"
    exec_mongo_cmd "db = db.getSiblingDB('ethereum_raw_data'); db.stats()"
    
    echo -e "\nCollection Stats:"
    exec_mongo_cmd "db = db.getSiblingDB('ethereum_raw_data'); db.blocks.stats().count"
    exec_mongo_cmd "db = db.getSiblingDB('ethereum_raw_data'); db.transactions.stats().count"
}

# Main function
main() {
    log "Starting MongoDB optimization for $CONTAINER_NAME"
    
    if ! check_container; then
        log "${RED}MongoDB container is not running. Please start it first.${NC}"
        exit 1
    fi
    
    # Wait for MongoDB to be ready
    log "Waiting for MongoDB to be ready..."
    local retries=0
    while ! exec_mongo_cmd "db.adminCommand('ping').ok" >/dev/null 2>&1; do
        retries=$((retries + 1))
        if [ $retries -gt 30 ]; then
            log "${RED}MongoDB is not responding after 30 attempts${NC}"
            exit 1
        fi
        sleep 2
    done
    
    log "${GREEN}MongoDB is ready${NC}"
    
    # Apply optimizations
    optimize_mongodb
    create_performance_indexes
    configure_connection_handling
    setup_monitoring
    
    # Show final status
    show_status
    
    log "${GREEN}MongoDB optimization completed successfully${NC}"
}

# Handle command line arguments
case "${1:-}" in
    --status)
        if check_container; then
            show_status
        else
            log "${RED}MongoDB container is not running${NC}"
            exit 1
        fi
        ;;
    --help)
        echo "Usage: $0 [--status|--help]"
        echo "  --status  Show current MongoDB status"
        echo "  --help    Show this help message"
        echo "  (no args) Run full optimization"
        ;;
    *)
        main
        ;;
esac
