#!/bin/bash

# Ethereum Scheduler Update Script
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
LOG_FILE="/tmp/update-deployment-$(date +%Y%m%d_%H%M%S).log"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log() {
    echo -e "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

error_exit() {
    log "${RED}ERROR: $1${NC}"
    exit 1
}

# Function to check if service is running
check_service_running() {
    if docker ps | grep -q "ethereum-scheduler-app"; then
        return 0
    elif pgrep -f "./scheduler" > /dev/null; then
        return 0
    else
        return 1
    fi
}

# Main update function
main() {
    log "${BLUE}=== ETHEREUM SCHEDULER UPDATE PROCESS ===${NC}"

    cd "$PROJECT_DIR" || error_exit "Cannot access project directory"

    # 1. Pre-update checks
    log "${YELLOW}1. Pre-update checks...${NC}"
    git status || error_exit "Not a git repository"

    # 2. Backup current state
    log "${YELLOW}2. Backing up current state...${NC}"
    if [ -f "scheduler.log" ]; then
        cp scheduler.log "scheduler_backup_$(date +%Y%m%d_%H%M%S).log"
        log "${GREEN}✓ Log backup created${NC}"
    fi

    # 3. Stop current services
    log "${YELLOW}3. Stopping current services...${NC}"
    if check_service_running; then
        # Try Docker first
        if docker ps | grep -q "ethereum-scheduler"; then
            log "Stopping Docker containers..."
            ./scripts/run-scheduler.sh stop || docker-compose -f docker-compose.scheduler.yml down
        fi

        # Then check for binary process
        if pgrep -f "./scheduler" > /dev/null; then
            log "Stopping scheduler binary..."
            pkill -f "./scheduler" || true
            sleep 3
        fi

        log "${GREEN}✓ Services stopped${NC}"
    else
        log "${GREEN}✓ No services running${NC}"
    fi

    # 4. Update code
    log "${YELLOW}4. Updating code...${NC}"
    git fetch origin
    local_commit=$(git rev-parse HEAD)
    remote_commit=$(git rev-parse origin/main)

    if [ "$local_commit" != "$remote_commit" ]; then
        log "Pulling latest changes..."
        git pull origin main || error_exit "Git pull failed"
        log "${GREEN}✓ Code updated (${remote_commit:0:8})${NC}"
    else
        log "${GREEN}✓ Code already up to date${NC}"
    fi

    # 5. Check for dependency changes
    log "${YELLOW}5. Checking dependencies...${NC}"
    if git diff HEAD~1 go.mod go.sum | grep -q .; then
        log "Dependencies changed, updating..."
        go mod download && go mod tidy
        log "${GREEN}✓ Dependencies updated${NC}"
    else
        log "${GREEN}✓ No dependency changes${NC}"
    fi

    # 6. Build and deploy
    log "${YELLOW}6. Building and deploying...${NC}"

    # Choose deployment method
    if [ "${DEPLOY_METHOD:-docker}" == "binary" ]; then
        log "Building binary..."
        go build -o scheduler cmd/schedulers/main.go || error_exit "Build failed"

        log "Starting scheduler binary..."
        nohup ./scheduler > scheduler.log 2>&1 &
        SCHEDULER_PID=$!
        log "Scheduler started with PID: $SCHEDULER_PID"
    else
        log "Building and starting Docker containers..."
        docker-compose -f docker-compose.scheduler.yml build --no-cache || error_exit "Docker build failed"
        ./scripts/run-scheduler.sh docker || error_exit "Docker start failed"
    fi

    log "${GREEN}✓ Deployment completed${NC}"

    # 7. Post-deployment verification
    log "${YELLOW}7. Verifying deployment...${NC}"
    sleep 10

    if check_service_running; then
        log "${GREEN}✓ Service is running${NC}"

        # Check logs for errors
        if [ -f "scheduler.log" ]; then
            if grep -i "error\|panic\|fatal" scheduler.log | tail -1 | grep -q .; then
                log "${YELLOW}⚠ Recent errors found in logs${NC}"
                grep -i "error\|panic\|fatal" scheduler.log | tail -3
            else
                log "${GREEN}✓ No recent errors in logs${NC}"
            fi
        fi

        # Check block processing
        if [ -f "scheduler.log" ]; then
            if grep "Block processed successfully\|Started" scheduler.log | tail -1 | grep -q .; then
                log "${GREEN}✓ Scheduler appears to be processing correctly${NC}"
            else
                log "${YELLOW}⚠ No recent block processing activity${NC}"
            fi
        fi
    else
        error_exit "Service failed to start"
    fi

    log "${GREEN}=== UPDATE COMPLETED SUCCESSFULLY ===${NC}"
    log "${BLUE}Update log saved to: $LOG_FILE${NC}"
    log "${BLUE}Monitor logs with: tail -f scheduler.log${NC}"
}

# Handle command line arguments
case "${1:-}" in
    --binary)
        export DEPLOY_METHOD="binary"
        main
        ;;
    --docker)
        export DEPLOY_METHOD="docker"
        main
        ;;
    --help)
        echo "Usage: $0 [--binary|--docker|--help]"
        echo "  --binary  Deploy as standalone binary"
        echo "  --docker  Deploy using Docker (default)"
        echo "  --help    Show this help"
        ;;
    *)
        main
        ;;
esac