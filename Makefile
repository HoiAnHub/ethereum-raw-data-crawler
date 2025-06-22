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