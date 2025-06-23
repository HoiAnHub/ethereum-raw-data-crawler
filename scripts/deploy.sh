#!/bin/bash

# Ethereum Scheduler Deploy Helper
# Usage: ./scripts/deploy.sh [option]

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Default settings
FORCE_REBUILD=false
ENVIRONMENT="development"
ENV_FILE=".env"

print_usage() {
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  dev              Deploy for development (default)"
    echo "  prod             Deploy for production"
    echo "  fresh            Force fresh build (no cache)"
    echo "  clean            Clean all Docker artifacts and rebuild"
    echo "  check            Check environment and status"
    echo "  logs             Show recent logs"
    echo "  stop             Stop all services"
    echo "  help             Show this help"
    echo ""
    echo "Examples:"
    echo "  $0 dev           # Start development environment"
    echo "  $0 fresh         # Force rebuild and start"
    echo "  $0 prod          # Deploy to production"
    echo "  $0 clean         # Clean everything and rebuild"
}

log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_prerequisites() {
    log "Checking prerequisites..."

    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed"
        exit 1
    fi

    if ! command -v docker-compose &> /dev/null; then
        log_error "Docker Compose is not installed"
        exit 1
    fi

    log_success "Prerequisites check passed"
}

check_env_file() {
    log "Checking environment file: $ENV_FILE"

    if [ ! -f "$ENV_FILE" ]; then
        log_warning "Environment file $ENV_FILE not found"
        if [ -f "env.example" ]; then
            log "Creating $ENV_FILE from env.example"
            cp env.example "$ENV_FILE"
            log_warning "Please edit $ENV_FILE with your configuration"
        else
            log_error "No env.example found. Cannot create environment file"
            exit 1
        fi
    fi

    log_success "Environment file check passed"
}

deploy_development() {
    log "Deploying for development..."
    ENV_FILE=".env"

    check_env_file

    if [ "$FORCE_REBUILD" = true ]; then
        log "Force rebuilding..."
        docker-compose -f docker-compose.scheduler.yml down || true
        docker-compose -f docker-compose.scheduler.yml build --no-cache
    fi

    log "Starting services..."
    docker-compose -f docker-compose.scheduler.yml up -d

    log_success "Development environment started"
}

deploy_production() {
    log "Deploying for production..."
    ENV_FILE=".env.scheduler.production"

    if [ ! -f "$ENV_FILE" ]; then
        if [ -f ".env.scheduler.example" ]; then
            log "Creating $ENV_FILE from .env.scheduler.example"
            cp .env.scheduler.example "$ENV_FILE"
            log_warning "Please edit $ENV_FILE with your production configuration"
            read -p "Continue with deployment? (y/N): " -n 1 -r
            echo
            if [[ ! $REPLY =~ ^[Yy]$ ]]; then
                log "Deployment cancelled"
                exit 1
            fi
        else
            log_error "Production environment file not found"
            exit 1
        fi
    fi

    log "Production deployment - forcing fresh build"
    docker-compose -f docker-compose.scheduler.yml down || true
    docker system prune -f || true
    docker-compose -f docker-compose.scheduler.yml --env-file "$ENV_FILE" build --no-cache
    docker-compose -f docker-compose.scheduler.yml --env-file "$ENV_FILE" up -d

    log_success "Production deployment completed"
}

clean_deployment() {
    log "Cleaning Docker artifacts..."

    # Stop all containers
    docker-compose -f docker-compose.scheduler.yml down || true

    # Remove images
    docker rmi ethereum-raw-data-crawler-ethereum-scheduler:latest || true

    # Clean system
    docker system prune -f || true

    # Rebuild
    log "Rebuilding from scratch..."
    docker-compose -f docker-compose.scheduler.yml build --no-cache
    docker-compose -f docker-compose.scheduler.yml up -d

    log_success "Clean deployment completed"
}

check_status() {
    log "Checking deployment status..."

    echo ""
    echo -e "${YELLOW}Container Status:${NC}"
    docker ps --filter "name=ethereum-scheduler" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" || echo "No containers running"

    echo ""
    echo -e "${YELLOW}Environment Check:${NC}"
    if [ -f "scripts/quick-env-check.sh" ]; then
        chmod +x scripts/quick-env-check.sh
        ./scripts/quick-env-check.sh
    else
        log_warning "Environment check script not found"
    fi
}

show_logs() {
    log "Showing recent logs..."
    docker-compose -f docker-compose.scheduler.yml logs --tail=50 -f
}

stop_services() {
    log "Stopping all services..."
    docker-compose -f docker-compose.scheduler.yml down
    log_success "Services stopped"
}

# Main execution
main() {
    check_prerequisites

    case "${1:-dev}" in
        "dev"|"development")
            deploy_development
            ;;
        "prod"|"production")
            deploy_production
            ;;
        "fresh")
            FORCE_REBUILD=true
            deploy_development
            ;;
        "clean")
            clean_deployment
            ;;
        "check"|"status")
            check_status
            ;;
        "logs")
            show_logs
            ;;
        "stop")
            stop_services
            ;;
        "help"|"-h"|"--help")
            print_usage
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            print_usage
            exit 1
            ;;
    esac

    # Show status after deployment
    if [[ "$1" == "dev" || "$1" == "prod" || "$1" == "fresh" || "$1" == "clean" ]]; then
        echo ""
        log "Deployment completed. Checking status..."
        sleep 5
        check_status
    fi
}

main "$@"