# Ethereum Block Scheduler Makefile
# Simplified version focusing only on scheduler functionality

# Variables
DOCKER_TAG ?= latest
BUILD_DIR = bin
BINARY_NAME = scheduler

# Colors for output
RED = \033[0;31m
GREEN = \033[0;32m
YELLOW = \033[1;33m
BLUE = \033[0;34m
NC = \033[0m # No Color

.PHONY: help setup build clean test lint fmt vet deps
.PHONY: scheduler-build scheduler-run scheduler-up scheduler-down scheduler-logs scheduler-status
.PHONY: docker-build-scheduler

## Default target
all: help

## Show this help message
help:
	@echo "$(BLUE)Ethereum Block Scheduler - Available Commands:$(NC)"
	@echo ""
	@echo "$(YELLOW)Development:$(NC)"
	@echo "  setup                Setup development environment"
	@echo "  build                Build scheduler binary"
	@echo "  run                  Run scheduler locally"
	@echo "  test                 Run tests"
	@echo "  clean                Clean build artifacts"
	@echo ""
	@echo "$(YELLOW)Code Quality:$(NC)"
	@echo "  fmt                  Format Go code"
	@echo "  vet                  Vet Go code"
	@echo "  lint                 Lint Go code"
	@echo "  deps                 Update dependencies"
	@echo ""
	@echo "$(YELLOW)Docker & Deployment:$(NC)"
	@echo "  docker-build         Build Docker image"
	@echo "  docker-build-fresh   Build Docker image (no cache)"
	@echo "  docker-clean-build   Clean and rebuild Docker image"
	@echo "  scheduler-up         Start scheduler with Docker Compose"
	@echo "  scheduler-up-fresh   Force rebuild and start (latest code)"
	@echo "  scheduler-down       Stop scheduler services"
	@echo "  scheduler-logs       View scheduler logs"
	@echo "  scheduler-status     Show service status"
	@echo "  deploy-production    Deploy to production (fresh build)"
	@echo ""
	@echo "$(YELLOW)Environment & Debugging:$(NC)"
	@echo "  env-check            Check .env file exists"
	@echo "  env-check-container  Check container environment variables"
	@echo "  env-check-full       Full environment check with tests"
	@echo "  test-mongodb         Test MongoDB connection"
	@echo "  setup-mongodb        Setup MongoDB indexes"
	@echo ""
	@echo "$(YELLOW)🔥 Recommended deployment scripts:$(NC)"
	@echo "  ./scripts/deploy.sh fresh            # Force fresh build (latest code)"
	@echo "  ./scripts/deploy.sh prod             # Production deployment"
	@echo "  ./scripts/deploy.sh clean            # Clean build"
	@echo ""
	@echo "$(YELLOW)Legacy run-scheduler.sh script:$(NC)"
	@echo "  ./scripts/run-scheduler.sh dev       # Run in development mode"
	@echo "  ./scripts/run-scheduler.sh docker    # Run with Docker"
	@echo "  ./scripts/run-scheduler.sh build     # Build binary"
	@echo "  ./scripts/run-scheduler.sh logs      # View logs"

## Setup development environment
setup:
	@echo "$(BLUE)Setting up development environment...$(NC)"
	@command -v go >/dev/null 2>&1 || { echo "$(RED)Go is required but not installed$(NC)"; exit 1; }
	@command -v docker >/dev/null 2>&1 || { echo "$(RED)Docker is required but not installed$(NC)"; exit 1; }
	@command -v docker-compose >/dev/null 2>&1 || { echo "$(RED)Docker Compose is required but not installed$(NC)"; exit 1; }
	@go version
	@docker --version
	@docker-compose --version
	@echo "$(GREEN)✓ Environment setup completed$(NC)"
	@echo "$(YELLOW)Next steps:$(NC)"
	@echo "  1. Copy env.example to .env and configure your settings"
	@echo "  2. Run 'make build' to build the scheduler"
	@echo "  3. Run 'make scheduler-up' to start with Docker"

## Build scheduler binary
build:
	@echo "$(BLUE)Building scheduler binary...$(NC)"
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o $(BUILD_DIR)/$(BINARY_NAME) cmd/schedulers/main.go
	@echo "$(GREEN)✓ Binary built: $(BUILD_DIR)/$(BINARY_NAME)$(NC)"

## Run scheduler locally
run:
	@echo "$(BLUE)Running scheduler locally...$(NC)"
	@go run cmd/schedulers/main.go

## Clean build artifacts
clean:
	@echo "$(BLUE)Cleaning build artifacts...$(NC)"
	@rm -rf $(BUILD_DIR)
	@docker system prune -f 2>/dev/null || true
	@echo "$(GREEN)✓ Cleanup completed$(NC)"

## Run tests
test:
	@echo "$(BLUE)Running tests...$(NC)"
	@go test ./internal/application/service -v
	@go test ./internal/infrastructure/blockchain -v
	@echo "$(GREEN)✓ Tests completed$(NC)"

## Format code
fmt:
	@echo "$(BLUE)Formatting code...$(NC)"
	@go fmt ./...
	@echo "$(GREEN)✓ Code formatted$(NC)"

## Vet code
vet:
	@echo "$(BLUE)Vetting code...$(NC)"
	@go vet ./...
	@echo "$(GREEN)✓ Code vetted$(NC)"

## Lint code (requires golangci-lint)
lint:
	@echo "$(BLUE)Linting code...$(NC)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
		echo "$(GREEN)✓ Code linted$(NC)"; \
	else \
		echo "$(YELLOW)golangci-lint not installed, skipping...$(NC)"; \
	fi

## Update dependencies
deps:
	@echo "$(BLUE)Updating dependencies...$(NC)"
	@go get -u ./...
	@go mod tidy
	@echo "$(GREEN)✓ Dependencies updated$(NC)"

## Build Docker image
docker-build:
	@echo "$(BLUE)Building Docker image...$(NC)"
	@docker build -f Dockerfile.scheduler -t ethereum-scheduler:$(DOCKER_TAG) .
	@echo "$(GREEN)✓ Docker image built$(NC)"

## Build Docker image with no cache (force fresh build)
docker-build-fresh:
	@echo "$(BLUE)Building Docker image (no cache)...$(NC)"
	@docker-compose -f docker-compose.scheduler.yml build --no-cache
	@echo "$(GREEN)✓ Fresh Docker image built$(NC)"

## Clean Docker images and rebuild
docker-clean-build:
	@echo "$(BLUE)Cleaning Docker images and rebuilding...$(NC)"
	@docker-compose -f docker-compose.scheduler.yml down || true
	@docker rmi ethereum-raw-data-crawler-ethereum-scheduler:latest || true
	@docker system prune -f || true
	@docker-compose -f docker-compose.scheduler.yml build --no-cache
	@echo "$(GREEN)✓ Clean Docker build completed$(NC)"

## Start scheduler with Docker Compose
scheduler-up:
	@echo "$(BLUE)Starting scheduler services...$(NC)"
	@docker-compose -f docker-compose.scheduler.yml up -d
	@echo "$(GREEN)✓ Scheduler services started$(NC)"
	@echo "$(YELLOW)Use 'make scheduler-logs' to view logs$(NC)"

## Force rebuild and start scheduler (ensures latest code)
scheduler-up-fresh:
	@echo "$(BLUE)Force rebuilding and starting scheduler services...$(NC)"
	@docker-compose -f docker-compose.scheduler.yml down || true
	@docker-compose -f docker-compose.scheduler.yml build --no-cache
	@docker-compose -f docker-compose.scheduler.yml up -d
	@echo "$(GREEN)✓ Scheduler services started with fresh build$(NC)"
	@echo "$(YELLOW)Use 'make scheduler-logs' to view logs$(NC)"

## Stop scheduler services
scheduler-down:
	@echo "$(BLUE)Stopping scheduler services...$(NC)"
	@docker-compose -f docker-compose.scheduler.yml down
	@echo "$(GREEN)✓ Scheduler services stopped$(NC)"

## View scheduler logs
scheduler-logs:
	@echo "$(BLUE)Viewing scheduler logs...$(NC)"
	@docker-compose -f docker-compose.scheduler.yml logs -f ethereum-scheduler

## Check scheduler status
scheduler-status:
	@echo "$(BLUE)Scheduler Status:$(NC)"
	@echo "$(YELLOW)Docker Containers:$(NC)"
	@docker ps --filter "name=ethereum-scheduler" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" || echo "No containers running"

## Environment check
env-check:
	@echo "$(BLUE)Checking environment...$(NC)"
	@test -f .env || { echo "$(YELLOW)⚠ .env file not found. Copy from env.example$(NC)"; }
	@echo "$(GREEN)✓ Environment check completed$(NC)"

## Check environment variables in running container
env-check-container:
	@echo "$(BLUE)Checking container environment variables...$(NC)"
	@if [ -f scripts/quick-env-check.sh ]; then \
		chmod +x scripts/quick-env-check.sh; \
		./scripts/quick-env-check.sh; \
	else \
		echo "$(YELLOW)scripts/quick-env-check.sh not found$(NC)"; \
	fi

## Comprehensive environment check with connection tests
env-check-full:
	@echo "$(BLUE)Running comprehensive environment check...$(NC)"
	@if [ -f scripts/check-env-vars.sh ]; then \
		chmod +x scripts/check-env-vars.sh; \
		./scripts/check-env-vars.sh; \
	else \
		echo "$(YELLOW)scripts/check-env-vars.sh not found$(NC)"; \
	fi

## Test MongoDB connection
test-mongodb:
	@echo "$(BLUE)Testing MongoDB connection...$(NC)"
	@if [ -f scripts/test-mongodb-connection.sh ]; then \
		chmod +x scripts/test-mongodb-connection.sh; \
		./scripts/test-mongodb-connection.sh; \
	else \
		echo "$(YELLOW)scripts/test-mongodb-connection.sh not found$(NC)"; \
	fi

## Setup MongoDB indexes
setup-mongodb:
	@echo "$(BLUE)Setting up MongoDB indexes...$(NC)"
	@if [ -f scripts/setup-mongodb-indexes.sh ]; then \
		chmod +x scripts/setup-mongodb-indexes.sh; \
		./scripts/setup-mongodb-indexes.sh; \
	else \
		echo "$(YELLOW)scripts/setup-mongodb-indexes.sh not found$(NC)"; \
	fi

## Deploy to production (force fresh build)
deploy-production:
	@echo "$(BLUE)Deploying to production...$(NC)"
	@echo "$(YELLOW)This will rebuild everything from scratch$(NC)"
	@docker-compose -f docker-compose.scheduler.yml down || true
	@docker system prune -f || true
	@docker-compose -f docker-compose.scheduler.yml --env-file .env.scheduler.production build --no-cache
	@docker-compose -f docker-compose.scheduler.yml --env-file .env.scheduler.production up -d
	@echo "$(GREEN)✓ Production deployment completed$(NC)"
	@echo "$(YELLOW)Run 'make env-check-container' to verify deployment$(NC)"

## Development workflow
dev: deps fmt vet test build
	@echo "$(GREEN)✓ Development workflow completed$(NC)"

## CI pipeline
ci: deps fmt vet lint test
	@echo "$(GREEN)✓ CI pipeline completed$(NC)"