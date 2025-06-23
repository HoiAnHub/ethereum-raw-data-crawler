# Ethereum Raw Data Crawler Makefile

# Variables
BINARY_NAME=crawler
DOCKER_IMAGE=ethereum-crawler
DOCKER_TAG=latest
BUILD_DIR=bin
GO_VERSION=1.21

# Default target
.DEFAULT_GOAL := help

# Colors for pretty output
BLUE=\033[0;34m
GREEN=\033[0;32m
YELLOW=\033[1;33m
RED=\033[0;31m
NC=\033[0m # No Color

.PHONY: help build clean test lint run dev docker-build docker-run docker-compose-up docker-compose-down install deps fmt vet

## Show this help message
help:
	@echo "$(BLUE)Ethereum Raw Data Crawler$(NC)"
	@echo "$(YELLOW)Available commands:$(NC)"
	@awk 'BEGIN {FS = ":.*##"; printf "\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  $(GREEN)%-15s$(NC) %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

## Install dependencies
install:
	@echo "$(BLUE)Installing dependencies...$(NC)"
	go mod download
	go mod tidy

## Install development dependencies
deps:
	@echo "$(BLUE)Installing development dependencies...$(NC)"
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/air-verse/air@latest

## Format code
fmt:
	@echo "$(BLUE)Formatting code...$(NC)"
	go fmt ./...

## Vet code
vet:
	@echo "$(BLUE)Vetting code...$(NC)"
	go vet ./...

## Run linter
lint:
	@echo "$(BLUE)Running linter...$(NC)"
	golangci-lint run

## Run tests
test:
	@echo "$(BLUE)Running tests...$(NC)"
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

## Run tests with coverage
test-coverage:
	@echo "$(BLUE)Running tests with coverage...$(NC)"
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -func coverage.out

## Build the application
build:
	@echo "$(BLUE)Building application...$(NC)"
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o $(BUILD_DIR)/$(BINARY_NAME) cmd/crawler/main.go

## Build for multiple platforms
build-all:
	@echo "$(BLUE)Building for multiple platforms...$(NC)"
	mkdir -p $(BUILD_DIR)
	# Linux
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 cmd/crawler/main.go
	# macOS
	GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 cmd/crawler/main.go
	GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 cmd/crawler/main.go
	# Windows
	GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe cmd/crawler/main.go

## Run the application locally
run:
	@echo "$(BLUE)Running application...$(NC)"
	go run cmd/crawler/main.go

## Run the application in development mode with hot reload
dev:
	@echo "$(BLUE)Starting development server with hot reload...$(NC)"
	air -c .air.toml

## Clean build artifacts
clean:
	@echo "$(BLUE)Cleaning build artifacts...$(NC)"
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

## Build Docker image
docker-build:
	@echo "$(BLUE)Building Docker image...$(NC)"
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

## Run Docker container
docker-run:
	@echo "$(BLUE)Running Docker container...$(NC)"
	docker run --rm --name ethereum-crawler \
		-e ETHEREUM_RPC_URL=${ETHEREUM_RPC_URL} \
		-e MONGO_URI=${MONGO_URI} \
		-e START_BLOCK_NUMBER=${START_BLOCK_NUMBER} \
		$(DOCKER_IMAGE):$(DOCKER_TAG)

## Start services with docker-compose
docker-compose-up:
	@echo "$(BLUE)Starting services with docker-compose...$(NC)"
	docker-compose up -d

## Stop services with docker-compose
docker-compose-down:
	@echo "$(BLUE)Stopping services with docker-compose...$(NC)"
	docker-compose down

## Start development environment
start-dev: docker-compose-up
	@echo "$(GREEN)Development environment started!$(NC)"
	@echo "$(YELLOW)Services:$(NC)"
	@echo "  - MongoDB: mongodb://localhost:27017"
	@echo "  - Mongo Express: http://localhost:8081 (admin/password)"
	@echo "  - Prometheus: http://localhost:9090"

## Stop development environment
stop-dev: docker-compose-down
	@echo "$(GREEN)Development environment stopped!$(NC)"

## Setup environment for first time
setup:
	@echo "$(BLUE)Setting up environment...$(NC)"
	cp env.example .env
	@echo "$(YELLOW)Please edit .env file with your configuration$(NC)"
	make install
	make deps

## Check if everything is ready to run
check:
	@echo "$(BLUE)Checking environment...$(NC)"
	@command -v go >/dev/null 2>&1 || { echo "$(RED)Go is not installed$(NC)"; exit 1; }
	@echo "$(GREEN)✓ Go is installed$(NC)"
	@command -v docker >/dev/null 2>&1 || { echo "$(RED)Docker is not installed$(NC)"; exit 1; }
	@echo "$(GREEN)✓ Docker is installed$(NC)"
	@command -v docker-compose >/dev/null 2>&1 || { echo "$(RED)Docker Compose is not installed$(NC)"; exit 1; }
	@echo "$(GREEN)✓ Docker Compose is installed$(NC)"
	@test -f .env || { echo "$(YELLOW)⚠ .env file not found. Run 'make setup' first$(NC)"; }
	@echo "$(GREEN)✓ Environment check completed$(NC)"

## Show project status
status:
	@echo "$(BLUE)Project Status:$(NC)"
	@echo "$(YELLOW)Docker Containers:$(NC)"
	@docker ps --filter "name=ethereum-crawler" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" || echo "No containers running"

## View logs
logs:
	@echo "$(BLUE)Viewing application logs...$(NC)"
	docker-compose logs -f ethereum-crawler

## View all logs
logs-all:
	@echo "$(BLUE)Viewing all service logs...$(NC)"
	docker-compose logs -f

## Connect to MongoDB shell
mongo-shell:
	@echo "$(BLUE)Connecting to MongoDB shell...$(NC)"
	docker exec -it ethereum-crawler-mongodb mongosh ethereum_raw_data

## Backup MongoDB data
backup:
	@echo "$(BLUE)Backing up MongoDB data...$(NC)"
	mkdir -p backups
	docker exec ethereum-crawler-mongodb mongodump --db ethereum_raw_data --out /tmp/backup
	docker cp ethereum-crawler-mongodb:/tmp/backup ./backups/backup-$(shell date +%Y%m%d-%H%M%S)

## Restore MongoDB data from backup
restore:
	@echo "$(BLUE)Restoring MongoDB data...$(NC)"
	@echo "$(YELLOW)Usage: make restore BACKUP_DIR=backups/backup-YYYYMMDD-HHMMSS$(NC)"
	@test -n "$(BACKUP_DIR)" || { echo "$(RED)BACKUP_DIR is required$(NC)"; exit 1; }
	docker cp $(BACKUP_DIR) ethereum-crawler-mongodb:/tmp/restore
	docker exec ethereum-crawler-mongodb mongorestore --db ethereum_raw_data /tmp/restore/ethereum_raw_data

## Update dependencies
update:
	@echo "$(BLUE)Updating dependencies...$(NC)"
	go get -u ./...
	go mod tidy

## Security audit
audit:
	@echo "$(BLUE)Running security audit...$(NC)"
	go list -json -m all | nancy sleuth

## Performance benchmark
benchmark:
	@echo "$(BLUE)Running benchmarks...$(NC)"
	go test -bench=. -benchmem ./...

## Generate documentation
docs:
	@echo "$(BLUE)Generating documentation...$(NC)"
	godoc -http=:6060 &
	@echo "$(GREEN)Documentation server started at http://localhost:6060$(NC)"

## Release build
release: clean test lint build-all
	@echo "$(GREEN)Release build completed!$(NC)"

## Full CI pipeline
ci: deps fmt vet lint test
	@echo "$(GREEN)CI pipeline completed successfully!$(NC)"

# ===== WebSocket Listener Service =====

## Build WebSocket Listener
build-websocket:
	@echo "$(BLUE)Building WebSocket Listener...$(NC)"
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o $(BUILD_DIR)/websocket-listener cmd/websocket-listener/main.go

## Run WebSocket Listener locally
run-websocket:
	@echo "$(BLUE)Running WebSocket Listener...$(NC)"
	go run cmd/websocket-listener/main.go

## Build WebSocket Listener Docker image
docker-build-websocket:
	@echo "$(BLUE)Building WebSocket Listener Docker image...$(NC)"
	docker build -f Dockerfile.websocket-listener -t ethereum-websocket-listener:$(DOCKER_TAG) .

## Start WebSocket Listener services with docker-compose
websocket-up:
	@echo "$(BLUE)Starting WebSocket Listener services...$(NC)"
	docker-compose -f docker-compose.websocket-listener.yml up -d

## Stop WebSocket Listener services
websocket-down:
	@echo "$(BLUE)Stopping WebSocket Listener services...$(NC)"
	docker-compose -f docker-compose.websocket-listener.yml down

## View WebSocket Listener logs
websocket-logs:
	@echo "$(BLUE)Viewing WebSocket Listener logs...$(NC)"
	docker-compose -f docker-compose.websocket-listener.yml logs -f ethereum-websocket-listener

## View all WebSocket Listener service logs
websocket-logs-all:
	@echo "$(BLUE)Viewing all WebSocket Listener service logs...$(NC)"
	docker-compose -f docker-compose.websocket-listener.yml logs -f

## Connect to WebSocket Listener MongoDB
websocket-mongo-shell:
	@echo "$(BLUE)Connecting to WebSocket Listener MongoDB...$(NC)"
	docker exec -it ethereum-websocket-mongodb mongosh ethereum_raw_data

## Status of WebSocket Listener services
websocket-status:
	@echo "$(BLUE)WebSocket Listener Service Status:$(NC)"
	@echo "$(YELLOW)Docker Containers:$(NC)"
	@docker ps --filter "name=ethereum-websocket" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" || echo "No containers running"

## Restart WebSocket Listener
websocket-restart: websocket-down websocket-up
	@echo "$(GREEN)WebSocket Listener restarted!$(NC)"

## Setup WebSocket Listener environment
websocket-setup:
	@echo "$(BLUE)Setting up WebSocket Listener environment...$(NC)"
	@echo "$(YELLOW)WebSocket Listener Environment Variables:$(NC)"
	@echo "  ETHEREUM_WS_URL - WebSocket URL for Ethereum node"
	@echo "  WEBSOCKET_BATCH_SIZE - Batch size for database writes"
	@echo "  WEBSOCKET_SUBSCRIBE_TO_BLOCKS - Subscribe to new blocks (true/false)"
	@echo "  WEBSOCKET_SUBSCRIBE_TO_TXS - Subscribe to pending transactions (true/false)"
	@echo "  WEBSOCKET_SUBSCRIBE_TO_LOGS - Subscribe to contract logs (true/false)"
	@echo "$(GREEN)Add these to your .env file for local development$(NC)"

# ===== Scheduler Service =====

## Build Scheduler
build-scheduler:
	@echo "$(BLUE)Building Scheduler...$(NC)"
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o $(BUILD_DIR)/scheduler cmd/schedulers/main.go

## Run Scheduler locally
run-scheduler:
	@echo "$(BLUE)Running Scheduler...$(NC)"
	go run cmd/schedulers/main.go

## Start Scheduler services with docker-compose
scheduler-up:
	@echo "$(BLUE)Starting Scheduler services...$(NC)"
	docker-compose -f docker-compose.scheduler.yml up -d

## Stop Scheduler services
scheduler-down:
	@echo "$(BLUE)Stopping Scheduler services...$(NC)"
	docker-compose -f docker-compose.scheduler.yml down

## View Scheduler logs
scheduler-logs:
	@echo "$(BLUE)Viewing Scheduler logs...$(NC)"
	docker-compose -f docker-compose.scheduler.yml logs -f ethereum-scheduler

## Status of Scheduler services
scheduler-status:
	@echo "$(BLUE)Scheduler Service Status:$(NC)"
	@echo "$(YELLOW)Docker Containers:$(NC)"
	@docker ps --filter "name=ethereum-scheduler" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" || echo "No containers running"

# ===== All Services Management =====

## Start all services (Crawler, Scheduler, WebSocket Listener)
start-all:
	@echo "$(BLUE)Starting all services...$(NC)"
	make docker-compose-up
	make scheduler-up
	make websocket-up
	@echo "$(GREEN)All services started!$(NC)"
	@echo "$(YELLOW)Services running:$(NC)"
	@echo "  - Crawler: docker-compose.yml"
	@echo "  - Scheduler: docker-compose.scheduler.yml"
	@echo "  - WebSocket Listener: docker-compose.websocket-listener.yml"

## Stop all services
stop-all:
	@echo "$(BLUE)Stopping all services...$(NC)"
	make docker-compose-down
	make scheduler-down
	make websocket-down
	@echo "$(GREEN)All services stopped!$(NC)"

## Show status of all services
status-all:
	@echo "$(BLUE)All Services Status:$(NC)"
	make status
	make scheduler-status
	make websocket-status