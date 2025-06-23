#!/bin/bash

# Ethereum Block Scheduler Runner Script
# This script provides easy commands to run the scheduler in different modes

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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

# Function to check if .env file exists
check_env_file() {
    if [ ! -f ".env" ]; then
        print_error ".env file not found!"
        print_info "Please create .env file with required configuration."
        print_info "See .env.example for reference."
        exit 1
    fi
}

# Function to check if required environment variables are set
check_required_vars() {
    local required_vars=("ETHEREUM_RPC_URL" "ETHEREUM_WS_URL" "MONGO_URI")
    local missing_vars=()
    
    for var in "${required_vars[@]}"; do
        if [ -z "${!var}" ]; then
            missing_vars+=("$var")
        fi
    done
    
    if [ ${#missing_vars[@]} -ne 0 ]; then
        print_error "Missing required environment variables:"
        for var in "${missing_vars[@]}"; do
            echo "  - $var"
        done
        exit 1
    fi
}

# Function to show usage
show_usage() {
    echo "Usage: $0 [COMMAND] [OPTIONS]"
    echo ""
    echo "Commands:"
    echo "  dev         Run scheduler in development mode"
    echo "  build       Build scheduler binary"
    echo "  run         Run built scheduler binary"
    echo "  docker      Run scheduler with Docker Compose"
    echo "  docker-dev  Run scheduler with Docker Compose in development mode"
    echo "  stop        Stop Docker Compose services"
    echo "  logs        Show Docker Compose logs"
    echo "  clean       Clean up Docker resources"
    echo "  test        Run tests"
    echo "  help        Show this help message"
    echo ""
    echo "Options:"
    echo "  --mode MODE     Set scheduler mode (polling, realtime, hybrid)"
    echo "  --follow        Follow logs (for logs command)"
    echo ""
    echo "Examples:"
    echo "  $0 dev --mode realtime"
    echo "  $0 docker"
    echo "  $0 logs --follow"
}

# Function to run in development mode
run_dev() {
    print_info "Starting Ethereum Block Scheduler in development mode..."
    check_env_file
    source .env
    check_required_vars
    
    # Set mode if provided
    if [ ! -z "$1" ]; then
        export SCHEDULER_MODE="$1"
        print_info "Scheduler mode set to: $1"
    fi
    
    print_info "Running: go run cmd/schedulers/main.go"
    go run cmd/schedulers/main.go
}

# Function to build binary
build_binary() {
    print_info "Building scheduler binary..."
    mkdir -p bin
    go build -o bin/scheduler cmd/schedulers/main.go
    print_success "Binary built successfully: bin/scheduler"
}

# Function to run binary
run_binary() {
    print_info "Running scheduler binary..."
    check_env_file
    
    if [ ! -f "bin/scheduler" ]; then
        print_error "Binary not found. Run 'build' command first."
        exit 1
    fi
    
    source .env
    check_required_vars
    
    # Set mode if provided
    if [ ! -z "$1" ]; then
        export SCHEDULER_MODE="$1"
        print_info "Scheduler mode set to: $1"
    fi
    
    ./bin/scheduler
}

# Function to run with Docker
run_docker() {
    print_info "Starting Ethereum Block Scheduler with Docker Compose..."
    check_env_file
    
    # Set mode if provided
    if [ ! -z "$1" ]; then
        export SCHEDULER_MODE="$1"
        print_info "Scheduler mode set to: $1"
    fi
    
    docker-compose -f docker-compose.scheduler.yml up -d
    print_success "Scheduler started with Docker Compose"
    print_info "Use '$0 logs' to view logs"
}

# Function to run Docker in development mode
run_docker_dev() {
    print_info "Starting Ethereum Block Scheduler with Docker Compose (development mode)..."
    check_env_file
    
    # Set mode if provided
    if [ ! -z "$1" ]; then
        export SCHEDULER_MODE="$1"
        print_info "Scheduler mode set to: $1"
    fi
    
    docker-compose -f docker-compose.scheduler.yml up --build
}

# Function to stop Docker services
stop_docker() {
    print_info "Stopping Docker Compose services..."
    docker-compose -f docker-compose.scheduler.yml down
    print_success "Services stopped"
}

# Function to show logs
show_logs() {
    local follow_flag=""
    if [ "$1" = "--follow" ]; then
        follow_flag="-f"
    fi
    
    docker-compose -f docker-compose.scheduler.yml logs $follow_flag
}

# Function to clean Docker resources
clean_docker() {
    print_warning "This will remove all Docker containers, images, and volumes related to the scheduler."
    read -p "Are you sure? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        print_info "Cleaning Docker resources..."
        docker-compose -f docker-compose.scheduler.yml down -v --rmi all
        print_success "Docker resources cleaned"
    else
        print_info "Cancelled"
    fi
}

# Function to run tests
run_tests() {
    print_info "Running tests..."
    go test ./internal/application/service -v
    go test ./internal/infrastructure/blockchain -v
    print_success "Tests completed"
}

# Main script logic
case "$1" in
    "dev")
        run_dev "$2"
        ;;
    "build")
        build_binary
        ;;
    "run")
        run_binary "$2"
        ;;
    "docker")
        run_docker "$2"
        ;;
    "docker-dev")
        run_docker_dev "$2"
        ;;
    "stop")
        stop_docker
        ;;
    "logs")
        show_logs "$2"
        ;;
    "clean")
        clean_docker
        ;;
    "test")
        run_tests
        ;;
    "help"|"--help"|"-h")
        show_usage
        ;;
    "")
        print_error "No command specified"
        show_usage
        exit 1
        ;;
    *)
        print_error "Unknown command: $1"
        show_usage
        exit 1
        ;;
esac
