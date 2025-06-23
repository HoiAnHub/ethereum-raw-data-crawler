#!/bin/bash

# Production Deployment Script for Ethereum Scheduler
# This script deploys the scheduler with optimized MongoDB configuration

set -e

# Configuration
COMPOSE_FILE="docker-compose.scheduler.yml"
ENV_FILE=".env.scheduler.production"
PROJECT_NAME="ethereum-scheduler"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to log with timestamp
log() {
    echo -e "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

# Function to check if required files exist
check_prerequisites() {
    log "${YELLOW}Checking prerequisites...${NC}"
    
    local missing_files=()
    
    if [ ! -f "$COMPOSE_FILE" ]; then
        missing_files+=("$COMPOSE_FILE")
    fi
    
    if [ ! -f "$ENV_FILE" ]; then
        log "${YELLOW}Environment file $ENV_FILE not found, creating from template...${NC}"
        if [ -f ".env.scheduler.example" ]; then
            cp ".env.scheduler.example" "$ENV_FILE"
            log "${GREEN}Created $ENV_FILE from template${NC}"
        else
            missing_files+=("$ENV_FILE")
        fi
    fi
    
    if [ ${#missing_files[@]} -gt 0 ]; then
        log "${RED}Missing required files:${NC}"
        for file in "${missing_files[@]}"; do
            log "${RED}  - $file${NC}"
        done
        exit 1
    fi
    
    log "${GREEN}Prerequisites check passed${NC}"
}

# Function to validate environment configuration
validate_environment() {
    log "${YELLOW}Validating environment configuration...${NC}"
    
    # Check if critical environment variables are set
    source "$ENV_FILE"
    
    local missing_vars=()
    
    if [ -z "$ETHEREUM_RPC_URL" ] || [ "$ETHEREUM_RPC_URL" = "YOUR_PROJECT_ID" ]; then
        missing_vars+=("ETHEREUM_RPC_URL")
    fi
    
    if [ -z "$MONGO_URI" ]; then
        missing_vars+=("MONGO_URI")
    fi
    
    if [ ${#missing_vars[@]} -gt 0 ]; then
        log "${RED}Missing or invalid environment variables:${NC}"
        for var in "${missing_vars[@]}"; do
            log "${RED}  - $var${NC}"
        done
        log "${YELLOW}Please update $ENV_FILE with correct values${NC}"
        exit 1
    fi
    
    log "${GREEN}Environment validation passed${NC}"
}

# Function to build and start services
deploy_services() {
    log "${YELLOW}Deploying services...${NC}"
    
    # Stop existing services
    log "Stopping existing services..."
    docker-compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" -p "$PROJECT_NAME" down || true
    
    # Build and start services
    log "Building and starting services..."
    docker-compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" -p "$PROJECT_NAME" up -d --build
    
    log "${GREEN}Services deployed successfully${NC}"
}

# Function to wait for services to be ready
wait_for_services() {
    log "${YELLOW}Waiting for services to be ready...${NC}"
    
    local max_wait=300  # 5 minutes
    local wait_time=0
    local interval=10
    
    while [ $wait_time -lt $max_wait ]; do
        if docker-compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" -p "$PROJECT_NAME" ps | grep -q "Up"; then
            log "${GREEN}Services are running${NC}"
            break
        fi
        
        log "Waiting for services... (${wait_time}s/${max_wait}s)"
        sleep $interval
        wait_time=$((wait_time + interval))
    done
    
    if [ $wait_time -ge $max_wait ]; then
        log "${RED}Services failed to start within ${max_wait} seconds${NC}"
        show_logs
        exit 1
    fi
    
    # Additional wait for MongoDB to be fully ready
    log "Waiting for MongoDB to be fully ready..."
    sleep 30
}

# Function to optimize MongoDB
optimize_mongodb() {
    log "${YELLOW}Optimizing MongoDB configuration...${NC}"
    
    if [ -f "scripts/optimize-mongodb.sh" ]; then
        chmod +x scripts/optimize-mongodb.sh
        ./scripts/optimize-mongodb.sh
        log "${GREEN}MongoDB optimization completed${NC}"
    else
        log "${YELLOW}MongoDB optimization script not found, skipping...${NC}"
    fi
}

# Function to run stability test
run_stability_test() {
    log "${YELLOW}Running MongoDB stability test...${NC}"
    
    if [ -f "scripts/test-mongodb-stability.sh" ]; then
        chmod +x scripts/test-mongodb-stability.sh
        
        # Run a short stability test (60 seconds)
        TEST_DURATION=60 TEST_INTERVAL=5 ./scripts/test-mongodb-stability.sh
        
        if [ $? -eq 0 ]; then
            log "${GREEN}Stability test passed${NC}"
        else
            log "${YELLOW}Stability test had some issues, but continuing deployment${NC}"
        fi
    else
        log "${YELLOW}Stability test script not found, skipping...${NC}"
    fi
}

# Function to start monitoring
start_monitoring() {
    log "${YELLOW}Starting monitoring...${NC}"
    
    if [ -f "scripts/monitor-mongodb.sh" ]; then
        chmod +x scripts/monitor-mongodb.sh
        
        # Start monitoring in background
        ./scripts/monitor-mongodb.sh --daemon
        
        log "${GREEN}MongoDB monitoring started${NC}"
    else
        log "${YELLOW}MongoDB monitoring script not found, skipping...${NC}"
    fi
}

# Function to show service status
show_status() {
    log "${YELLOW}Service Status:${NC}"
    docker-compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" -p "$PROJECT_NAME" ps
    
    echo ""
    log "${YELLOW}Resource Usage:${NC}"
    docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}" \
        $(docker-compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" -p "$PROJECT_NAME" ps -q) 2>/dev/null || true
}

# Function to show logs
show_logs() {
    log "${YELLOW}Recent logs:${NC}"
    docker-compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" -p "$PROJECT_NAME" logs --tail=50
}

# Function to show help
show_help() {
    echo "Usage: $0 [command]"
    echo ""
    echo "Commands:"
    echo "  deploy     Deploy the services (default)"
    echo "  status     Show service status"
    echo "  logs       Show recent logs"
    echo "  stop       Stop all services"
    echo "  restart    Restart all services"
    echo "  optimize   Run MongoDB optimization"
    echo "  test       Run stability test"
    echo "  monitor    Start monitoring"
    echo "  help       Show this help message"
    echo ""
    echo "Environment:"
    echo "  ENV_FILE   Environment file to use (default: $ENV_FILE)"
}

# Function to stop services
stop_services() {
    log "${YELLOW}Stopping services...${NC}"
    docker-compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" -p "$PROJECT_NAME" down
    log "${GREEN}Services stopped${NC}"
}

# Function to restart services
restart_services() {
    log "${YELLOW}Restarting services...${NC}"
    stop_services
    deploy_services
    wait_for_services
    log "${GREEN}Services restarted${NC}"
}

# Main deployment function
main_deploy() {
    log "${BLUE}Starting production deployment...${NC}"
    
    check_prerequisites
    validate_environment
    deploy_services
    wait_for_services
    optimize_mongodb
    run_stability_test
    start_monitoring
    show_status
    
    log "${GREEN}Production deployment completed successfully!${NC}"
    log "${YELLOW}Monitor the logs with: $0 logs${NC}"
    log "${YELLOW}Check status with: $0 status${NC}"
}

# Handle command line arguments
case "${1:-deploy}" in
    deploy)
        main_deploy
        ;;
    status)
        show_status
        ;;
    logs)
        show_logs
        ;;
    stop)
        stop_services
        ;;
    restart)
        restart_services
        ;;
    optimize)
        optimize_mongodb
        ;;
    test)
        run_stability_test
        ;;
    monitor)
        start_monitoring
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        log "${RED}Unknown command: $1${NC}"
        show_help
        exit 1
        ;;
esac
