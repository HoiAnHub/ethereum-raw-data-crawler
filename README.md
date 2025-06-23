# Ethereum Block Scheduler

A robust Ethereum block scheduler service that efficiently monitors and processes Ethereum blockchain data in real-time or polling mode.

## ✨ Features

- **Multi-Mode Operation**: Supports realtime (WebSocket), polling, and hybrid modes
- **Real-time Block Processing**: WebSocket-based real-time block monitoring
- **Fallback Polling**: Automatic fallback to polling when WebSocket fails
- **Configurable Scheduling**: Flexible configuration for different use cases
- **Docker Support**: Full Docker and Docker Compose support
- **Database Integration**: MongoDB integration for data persistence
- **Health Monitoring**: Built-in health checks and monitoring
- **Error Recovery**: Automatic reconnection and error recovery

## 🚀 Quick Start

### Prerequisites

- Go 1.21 or higher
- Docker and Docker Compose
- MongoDB
- Ethereum RPC endpoint (Infura/Alchemy recommended)

### Setup

1. **Clone and setup environment:**
   ```bash
   git clone <repository-url>
   cd ethereum-raw-data-crawler
   make setup
   ```

2. **Configure environment:**
   ```bash
   cp env.example .env
   # Edit .env with your Ethereum RPC URLs and MongoDB settings
   ```

3. **Start the scheduler:**
   ```bash
   # Using Docker (recommended)
   make scheduler-up

   # Or using the dedicated script
   ./scripts/run-scheduler.sh docker

   # Or build and run locally
   make build && make run
   ```

## 📋 Usage

### 🔥 Using Deploy Script (Recommended)

The `./scripts/deploy.sh` script ensures latest code is always used:

```bash
# Development with fresh build (RECOMMENDED FOR CODE CHANGES)
./scripts/deploy.sh fresh

# Production deployment
./scripts/deploy.sh prod

# Clean everything and rebuild (if Docker cache issues)
./scripts/deploy.sh clean

# Check status and environment variables
./scripts/deploy.sh check

# Show logs
./scripts/deploy.sh logs

# Stop all services
./scripts/deploy.sh stop
```

### Using run-scheduler.sh Script (Legacy)

The `./scripts/run-scheduler.sh` script provides a convenient interface:

```bash
# Development mode
./scripts/run-scheduler.sh dev

# Docker mode (detached)
./scripts/run-scheduler.sh docker

# Docker development mode (with logs)
./scripts/run-scheduler.sh docker-dev

# Build binary
./scripts/run-scheduler.sh build

# Run built binary
./scripts/run-scheduler.sh run

# View logs
./scripts/run-scheduler.sh logs --follow

# Stop services
./scripts/run-scheduler.sh stop

# Clean up Docker resources
./scripts/run-scheduler.sh clean
```

### Using Makefile

```bash
# Build scheduler
make build

# Run locally
make run

# Start with Docker Compose
make scheduler-up

# 🔥 IMPORTANT: Force rebuild with latest code
make scheduler-up-fresh

# View logs
make scheduler-logs

# Stop services
make scheduler-down

# Check status
make scheduler-status

# Check environment variables in container
make env-check-container

# Run tests
make test
```

## ⚙️ Configuration

Key environment variables in `.env`:

```bash
# Ethereum Configuration
ETHEREUM_RPC_URL=https://mainnet.infura.io/v3/YOUR_PROJECT_ID
ETHEREUM_WS_URL=wss://mainnet.infura.io/ws/v3/YOUR_PROJECT_ID

# MongoDB Configuration
MONGO_URI=mongodb://admin:password@localhost:27017/ethereum_raw_data?authSource=admin

# Scheduler Configuration
SCHEDULER_MODE=hybrid                    # realtime, polling, or hybrid
SCHEDULER_ENABLE_REALTIME=true
SCHEDULER_ENABLE_POLLING=true
SCHEDULER_POLLING_INTERVAL=3s
SCHEDULER_FALLBACK_TIMEOUT=30s

# Rate Limiting (for free tier APIs)
ETHEREUM_RATE_LIMIT=1s
ETHEREUM_REQUEST_TIMEOUT=120s
ETHEREUM_SKIP_RECEIPTS=true

# Application Configuration
APP_ENV=production
LOG_LEVEL=info
```

## 🏗️ Architecture

The scheduler operates in three modes:

### 1. Realtime Mode
- Uses WebSocket connections to receive new blocks immediately
- Minimal latency but requires stable WebSocket connection
- Best for real-time applications

### 2. Polling Mode
- Periodically polls for new blocks via RPC
- More reliable but higher latency
- Better for rate-limited APIs

### 3. Hybrid Mode (Recommended)
- Combines both realtime and polling
- Uses WebSocket as primary with polling fallback
- Automatically switches between modes based on connection health

## 📊 Monitoring

### Health Checks

The scheduler includes built-in health monitoring:

```bash
# Check container health
docker ps | grep ethereum-scheduler

# View detailed logs
make scheduler-logs

# Check service status
make scheduler-status
```

### Key Metrics

- Block processing rate
- Connection status (WebSocket/RPC)
- Error rates and recovery
- Database write performance

## 🛠️ Development

### Project Structure

```
.
├── cmd/schedulers/          # Scheduler entry point
├── internal/
│   ├── application/service/ # Application logic
│   ├── domain/             # Domain entities and interfaces
│   ├── infrastructure/     # Infrastructure implementations
│   └── adapters/           # Database adapters
├── scripts/                # Utility scripts
├── docker-compose.scheduler.yml
├── Dockerfile.scheduler
└── Makefile
```

### Building from Source

```bash
# Install dependencies
make deps

# Format and lint code
make fmt vet lint

# Run tests
make test

# Build binary
make build

# Development workflow
make dev
```

### Testing

```bash
# Run all tests
make test

# Run specific test
go test ./internal/application/service -v

# Test with coverage
go test -cover ./...
```

## 🐳 Docker Deployment

### Docker Compose (Recommended)

```bash
# Start all services
make scheduler-up

# 🔥 Force rebuild with latest code (recommended for development)
make scheduler-up-fresh

# Clean build (if Docker cache issues)
make docker-clean-build
make scheduler-up

# View logs
make scheduler-logs

# Stop services
make scheduler-down
```

### ⚠️ Ensuring Latest Code

When you make code changes, Docker may use cached layers. Always use:

```bash
# For development - forces fresh build
make scheduler-up-fresh

# For production deployment
make deploy-production

# If having Docker cache issues
make docker-clean-build && make scheduler-up
```

### 🔍 Environment Verification

After deployment, verify your environment:

```bash
# Quick check of critical variables
make env-check-container

# Comprehensive check with connection tests
make env-check-full
```

### Manual Docker

```bash
# Build image
make docker-build

# Run container
docker run -d \
  --name ethereum-scheduler \
  -e ETHEREUM_RPC_URL=your_rpc_url \
  -e MONGO_URI=your_mongo_uri \
  ethereum-scheduler:latest
```

## 🔧 Configuration Reference

### Scheduler Modes

| Mode | Description | Use Case |
|------|-------------|----------|
| `realtime` | WebSocket only | Real-time applications |
| `polling` | RPC polling only | Rate-limited APIs |
| `hybrid` | WebSocket + fallback | Production (recommended) |

### Performance Tuning

For high-throughput scenarios:

```bash
# Increase worker concurrency
CONCURRENT_WORKERS=5

# Adjust batch sizes
BATCH_SIZE=10

# Optimize MongoDB connection
MONGO_MAX_POOL_SIZE=20
```

For rate-limited APIs:

```bash
# Conservative settings
ETHEREUM_RATE_LIMIT=2s
ETHEREUM_SKIP_RECEIPTS=true
CONCURRENT_WORKERS=1
BATCH_SIZE=1
```

## 📝 Logging

Logs are structured JSON format with different levels:

```bash
# Set log level
LOG_LEVEL=debug  # debug, info, warn, error

# View logs in real-time
make scheduler-logs

# Filter logs
docker logs ethereum-scheduler-app | grep "level\":\"error\""
```

## 🆘 Troubleshooting

### Common Issues

1. **WebSocket Connection Fails**
   ```bash
   # Check WebSocket URL
   curl -H "Upgrade: websocket" $ETHEREUM_WS_URL

   # Fallback to polling mode
   export SCHEDULER_MODE=polling
   ```

2. **MongoDB Connection Issues**
   ```bash
   # Test MongoDB connection
   mongosh $MONGO_URI

   # Check container health
   docker exec ethereum-scheduler-mongodb mongosh --eval "db.adminCommand('ping')"
   ```

3. **Rate Limiting**
   ```bash
   # Increase delay between requests
   export ETHEREUM_RATE_LIMIT=5s

   # Skip transaction receipts
   export ETHEREUM_SKIP_RECEIPTS=true
   ```

### Debug Mode

Enable debug logging for detailed troubleshooting:

```bash
export LOG_LEVEL=debug
./scripts/run-scheduler.sh dev
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Make your changes and add tests
4. Run the test suite: `make test`
5. Format code: `make fmt`
6. Submit a pull request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🔗 Links

- [Docker Hub](https://hub.docker.com/)
- [Ethereum Documentation](https://ethereum.org/developers/)
- [MongoDB Documentation](https://docs.mongodb.com/)
- [Go Documentation](https://golang.org/doc/)

---

**Made with ❤️ for the Ethereum community**