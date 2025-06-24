#!/bin/bash

# VPS Deployment Script for Ethereum Raw Data Crawler with NATS
# This script sets up the full stack with remote NUI access

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
VPS_IP="45.149.206.55"
NUI_PORT="31311"
NATS_PORT="4222"
MONITOR_PORT="8222"

echo -e "${BLUE}🚀 VPS Deployment Script for Ethereum Raw Data Crawler${NC}"
echo -e "${BLUE}===================================================${NC}"
echo ""

# Function to print colored output
print_status() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

# Check prerequisites
echo -e "${BLUE}📋 Checking prerequisites...${NC}"

if ! command -v docker &> /dev/null; then
    print_error "Docker is not installed. Please install Docker first."
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    print_error "Docker Compose is not installed. Please install Docker Compose first."
    exit 1
fi

if ! command -v make &> /dev/null; then
    print_error "Make is not installed. Please install Make first."
    exit 1
fi

print_status "All prerequisites are installed"

# Check if .env file exists
if [ ! -f ".env" ]; then
    print_warning ".env file not found. Creating from example..."
    if [ -f "env.example" ]; then
        cp env.example .env
        print_status ".env file created from example"
        print_warning "Please edit .env file with your configuration before continuing"
        echo "Press Enter to continue after editing .env file..."
        read
    else
        print_error "env.example file not found. Please create .env file manually."
        exit 1
    fi
fi

# Enable NATS in .env if not already enabled
if ! grep -q "NATS_ENABLED=true" .env; then
    print_info "Enabling NATS in .env file..."
    echo "NATS_ENABLED=true" >> .env
    print_status "NATS enabled in .env file"
fi

# Check firewall status
echo ""
echo -e "${BLUE}🔥 Checking firewall configuration...${NC}"

# Try to detect firewall type and check port status
if command -v ufw &> /dev/null; then
    if ufw status | grep -q "Status: active"; then
        print_info "UFW firewall is active"
        if ufw status | grep -q "$NUI_PORT"; then
            print_status "Port $NUI_PORT is already allowed in UFW"
        else
            print_warning "Port $NUI_PORT is not allowed in UFW"
            echo "Run: sudo ufw allow $NUI_PORT"
        fi
    else
        print_info "UFW firewall is inactive"
    fi
elif command -v firewall-cmd &> /dev/null; then
    if firewall-cmd --state | grep -q "running"; then
        print_info "firewalld is active"
        if firewall-cmd --list-ports | grep -q "$NUI_PORT"; then
            print_status "Port $NUI_PORT is already allowed in firewalld"
        else
            print_warning "Port $NUI_PORT is not allowed in firewalld"
            echo "Run: sudo firewall-cmd --permanent --add-port=$NUI_PORT/tcp && sudo firewall-cmd --reload"
        fi
    else
        print_info "firewalld is inactive"
    fi
else
    print_warning "No firewall detected. Please ensure port $NUI_PORT is accessible."
fi

# Deploy the stack
echo ""
echo -e "${BLUE}🚀 Deploying full stack...${NC}"

print_info "Starting NATS + Crawler with remote access..."
make -f Makefile.nats full-stack-up

# Wait for services to be ready
echo ""
print_info "Waiting for services to be ready..."
sleep 10

# Health check
echo ""
echo -e "${BLUE}🏥 Running health check...${NC}"
make -f Makefile.nats health-check

# Test NUI accessibility
echo ""
echo -e "${BLUE}🌐 Testing NUI accessibility...${NC}"

if curl -s -I "http://localhost:$NUI_PORT" | grep -q "HTTP"; then
    print_status "NUI is accessible locally"
else
    print_error "NUI is not accessible locally"
fi

# Test remote accessibility
if curl -s -I "http://$VPS_IP:$NUI_PORT" | grep -q "HTTP"; then
    print_status "NUI is accessible remotely at http://$VPS_IP:$NUI_PORT"
else
    print_warning "NUI is not accessible remotely. Check firewall configuration."
fi

# Final status
echo ""
echo -e "${GREEN}🎉 Deployment completed!${NC}"
echo ""
echo -e "${BLUE}📊 Service URLs:${NC}"
echo -e "  • NATS Monitor: ${GREEN}http://localhost:$MONITOR_PORT${NC}"
echo -e "  • NATS NUI (Local): ${GREEN}http://localhost:$NUI_PORT${NC}"
echo -e "  • NATS NUI (Remote): ${GREEN}http://$VPS_IP:$NUI_PORT${NC}"
echo -e "  • NATS Connection: ${GREEN}nats://localhost:$NATS_PORT${NC}"
echo ""
echo -e "${BLUE}🔧 Management Commands:${NC}"
echo -e "  • View logs: ${YELLOW}make -f Makefile.nats crawler-logs${NC}"
echo -e "  • Health check: ${YELLOW}make -f Makefile.nats health-check${NC}"
echo -e "  • Monitor NATS: ${YELLOW}make -f Makefile.nats monitor${NC}"
echo -e "  • Stop services: ${YELLOW}make -f Makefile.nats full-stack-down${NC}"
echo ""
echo -e "${BLUE}🧪 Testing Commands:${NC}"
echo -e "  • Test transaction events: ${YELLOW}make -f Makefile.nats run-consumer${NC}"
echo -e "  • Publish test message: ${YELLOW}make -f Makefile.nats publish-test${NC}"
echo ""
print_info "You can now access the NATS NUI from any machine using: http://$VPS_IP:$NUI_PORT"