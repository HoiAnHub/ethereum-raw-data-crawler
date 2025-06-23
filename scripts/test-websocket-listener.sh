#!/bin/bash

# Test script for WebSocket Listener Service
set -e

echo "🚀 Testing WebSocket Listener Service"
echo "======================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Check if .env file exists
if [ ! -f ".env" ]; then
    echo -e "${YELLOW}⚠️  .env file not found. Creating from example...${NC}"
    cp env.websocket-listener.example .env.websocket-test
    echo -e "${YELLOW}📝 Please edit .env.websocket-test with your API keys${NC}"
    exit 1
fi

# Function to check if service is healthy
check_service_health() {
    local service_name=$1
    local max_attempts=30
    local attempt=1

    echo -e "${BLUE}🔍 Checking $service_name health...${NC}"

    while [ $attempt -le $max_attempts ]; do
        if docker ps --filter "name=$service_name" --filter "status=running" | grep -q "$service_name"; then
            echo -e "${GREEN}✅ $service_name is running${NC}"
            return 0
        fi

        echo "Attempt $attempt/$max_attempts: Waiting for $service_name..."
        sleep 2
        ((attempt++))
    done

    echo -e "${RED}❌ $service_name failed to start properly${NC}"
    return 1
}

# Function to check logs for errors
check_logs_for_errors() {
    local service_name=$1
    echo -e "${BLUE}🔍 Checking $service_name logs for errors...${NC}"

    # Get last 50 lines of logs and check for critical errors
    local error_count=$(docker-compose -f docker-compose.websocket-listener.yml logs --tail=50 $service_name 2>/dev/null | grep -i "error\|fatal\|panic" | wc -l)

    if [ $error_count -gt 0 ]; then
        echo -e "${YELLOW}⚠️  Found $error_count potential errors in $service_name logs${NC}"
        docker-compose -f docker-compose.websocket-listener.yml logs --tail=10 $service_name | grep -i "error\|fatal\|panic" || true
    else
        echo -e "${GREEN}✅ No critical errors found in $service_name logs${NC}"
    fi
}

# Function to test WebSocket connection
test_websocket_connection() {
    echo -e "${BLUE}🔗 Testing WebSocket connection...${NC}"

    # Check if WebSocket listener is connecting successfully
    local connection_logs=$(docker-compose -f docker-compose.websocket-listener.yml logs --tail=20 ethereum-websocket-listener 2>/dev/null | grep -i "websocket\|connected" || true)

    if echo "$connection_logs" | grep -q "Successfully connected"; then
        echo -e "${GREEN}✅ WebSocket connection established${NC}"
    else
        echo -e "${YELLOW}⚠️  WebSocket connection status unclear${NC}"
        echo "Recent logs:"
        docker-compose -f docker-compose.websocket-listener.yml logs --tail=5 ethereum-websocket-listener || true
    fi
}

# Function to test MongoDB connection
test_mongodb_connection() {
    echo -e "${BLUE}🔗 Testing MongoDB connection...${NC}"

    if docker exec ethereum-websocket-mongodb mongosh --eval "db.adminCommand('ping')" --quiet >/dev/null 2>&1; then
        echo -e "${GREEN}✅ MongoDB connection successful${NC}"
    else
        echo -e "${RED}❌ MongoDB connection failed${NC}"
        return 1
    fi
}

# Function to show service stats
show_service_stats() {
    echo -e "${BLUE}📊 Service Statistics${NC}"
    echo "===================="

    echo -e "${YELLOW}Docker Containers:${NC}"
    docker ps --filter "name=ethereum-websocket" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

    echo -e "\n${YELLOW}Memory Usage:${NC}"
    docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}" $(docker ps --filter "name=ethereum-websocket" --format "{{.Names}}")

    echo -e "\n${YELLOW}Recent Logs (last 5 lines):${NC}"
    docker-compose -f docker-compose.websocket-listener.yml logs --tail=5 ethereum-websocket-listener || true
}

# Main test flow
main() {
    echo -e "${BLUE}🏗️  Building WebSocket Listener image...${NC}"
    make docker-build-websocket

    echo -e "\n${BLUE}🚀 Starting WebSocket Listener services...${NC}"
    make websocket-up

    echo -e "\n${BLUE}⏳ Waiting for services to initialize...${NC}"
    sleep 10

    # Check service health
    check_service_health "ethereum-websocket-mongodb"
    check_service_health "ethereum-websocket-nats"
    check_service_health "ethereum-websocket-listener-app"

    # Test connections
    test_mongodb_connection
    test_websocket_connection

    # Check logs for errors
    check_logs_for_errors "ethereum-websocket-listener"
    check_logs_for_errors "mongodb"
    check_logs_for_errors "nats"

    # Show service stats
    show_service_stats

    echo -e "\n${GREEN}🎉 WebSocket Listener Service test completed!${NC}"
    echo -e "${YELLOW}📋 Next steps:${NC}"
    echo "  1. Check logs: make websocket-logs"
    echo "  2. View status: make websocket-status"
    echo "  3. Connect to MongoDB: make websocket-mongo-shell"
    echo "  4. Stop services: make websocket-down"

    echo -e "\n${BLUE}🔗 Service URLs:${NC}"
    echo "  - MongoDB: mongodb://admin:password@localhost:27018/ethereum_raw_data"
    echo "  - NATS Management: http://localhost:8222"
    echo "  - NATS Client: nats://localhost:4222"
}

# Handle cleanup on exit
cleanup() {
    echo -e "\n${YELLOW}🧹 Cleaning up...${NC}"
}

trap cleanup EXIT

# Run main function
main "$@"