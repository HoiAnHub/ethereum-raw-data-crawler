# Ethereum Raw Data Crawler

A high-performance, scalable Ethereum blockchain data crawler built with Go, designed to extract and store raw blockchain data for downstream applications.

## Features

- **High Performance**: Concurrent block processing with configurable worker pools
- **Real-time Processing**: WebSocket-based scheduler for immediate block processing
- **Hybrid Scheduling**: Combines real-time WebSocket with polling fallback
- **Scalable Architecture**: Hexagonal architecture with dependency injection using uber-go/fx
- **Comprehensive Data Storage**: Stores blocks, transactions, and metadata in MongoDB
- **Real-time Monitoring**: GraphQL API for system health and metrics monitoring
- **Extensible Design**: Modular structure ready for multi-blockchain support
- **Production Ready**: Robust error handling, logging, and graceful shutdown

## Architecture

### Project Structure

```
ethereum-raw-data-crawler/
├── cmd/
│   ├── crawler/           # Main crawler application (batch processing)
│   ├── schedulers/        # Real-time block scheduler
│   └── api/               # GraphQL API server
├── internal/
│   ├── adapters/
│   │   ├── primary/       # HTTP handlers, GraphQL resolvers
│   │   └── secondary/     # Repository implementations
│   ├── application/
│   │   ├── service/       # Application services
│   │   └── usecase/       # Business use cases
│   ├── domain/
│   │   ├── entity/        # Domain entities
│   │   ├── repository/    # Repository interfaces
│   │   └── service/       # Domain service interfaces
│   └── infrastructure/
│       ├── config/        # Configuration management
│       ├── database/      # Database connections
│       ├── blockchain/    # Blockchain client
│       └── logger/        # Logging infrastructure
├── pkg/
│   ├── errors/           # Custom error types
│   └── utils/            # Utility functions
├── deployments/          # Docker, K8s configs
├── docs/                 # Documentation
└── scripts/              # Build and deployment scripts
```

### Key Components

1. **Domain Layer**: Core business logic and entities
2. **Application Layer**: Orchestrates business operations
3. **Infrastructure Layer**: External concerns (database, blockchain, etc.)
4. **Adapters**: Interface implementations

## Data Model

### Collections

#### Blocks Collection
- **Document Structure**: Complete Ethereum block data
- **Indexes**: block number, hash, timestamp, network, status
- **Purpose**: Store raw block data for analysis

#### Transactions Collection
- **Document Structure**: Detailed transaction data with receipts
- **Indexes**: hash, block_hash, from/to addresses, block_number
- **Purpose**: Store all transaction data for comprehensive analysis

#### Crawler Metrics Collection
- **Document Structure**: Performance and operational metrics
- **Indexes**: timestamp, network
- **Purpose**: Monitor crawler performance and health

#### System Health Collection
- **Document Structure**: System health status and component checks
- **Indexes**: timestamp, network, status
- **Purpose**: Track system health over time

## Configuration

Create a `.env` file (copy from `env.example`):

```bash
# Ethereum RPC Configuration
ETHEREUM_RPC_URL=https://mainnet.infura.io/v3/YOUR_PROJECT_ID
ETHEREUM_WS_URL=wss://mainnet.infura.io/v3/YOUR_PROJECT_ID
START_BLOCK_NUMBER=1

# MongoDB Configuration
MONGO_URI=mongodb://localhost:27017
MONGO_DATABASE=ethereum_raw_data
MONGO_CONNECT_TIMEOUT=10s
MONGO_MAX_POOL_SIZE=100

# Application Configuration
APP_PORT=8080
APP_ENV=development
LOG_LEVEL=info

# Crawler Configuration
BATCH_SIZE=100
CONCURRENT_WORKERS=10
RETRY_ATTEMPTS=3
RETRY_DELAY=5s

# GraphQL Configuration
GRAPHQL_ENDPOINT=/graphql
GRAPHQL_PLAYGROUND=true

# Monitoring Configuration
METRICS_ENABLED=true
HEALTH_CHECK_INTERVAL=30s
```

## Installation & Setup

### Prerequisites

- Go 1.21+
- MongoDB 5.0+
- Ethereum RPC access (Infura, Alchemy, or local node)

### Installation

1. **Clone the repository**:
```bash
git clone https://github.com/your-org/ethereum-raw-data-crawler.git
cd ethereum-raw-data-crawler
```

2. **Install dependencies**:
```bash
go mod download
```

3. **Setup environment**:
```bash
cp env.example .env
# Edit .env with your configuration
```

4. **Start MongoDB**:
```bash
# Using Docker
docker run -d -p 27017:27017 --name mongodb mongo:5.0

# Or use your existing MongoDB instance
```

5. **Run the crawler**:
```bash
# For batch processing (historical data)
go run cmd/crawler/main.go

# For real-time block processing
go run cmd/schedulers/main.go
```

## Usage

### Running the Applications

#### 1. Batch Crawler (Historical Data)
The main crawler processes blocks in batches for historical data:

```bash
go run cmd/crawler/main.go
```

The crawler automatically:
1. Connects to the Ethereum network
2. Initializes MongoDB indexes
3. Resumes from the last processed block
4. Processes blocks concurrently in batches
5. Stores data in MongoDB
6. Monitors system health

#### 2. Real-time Scheduler (Live Data)
The scheduler processes new blocks immediately as they are created:

```bash
# Using the helper script
./scripts/run-scheduler.sh dev

# Or directly
go run cmd/schedulers/main.go
```

The scheduler automatically:
1. Connects to Ethereum WebSocket
2. Listens for new block notifications
3. Processes blocks immediately upon creation
4. Falls back to polling if WebSocket fails
5. Provides real-time data processing

### Scheduler Configuration

Configure the scheduler mode in your `.env` file:

```bash
# Scheduler Configuration
SCHEDULER_MODE=hybrid                    # polling, realtime, hybrid
SCHEDULER_ENABLE_REALTIME=true          # Enable WebSocket
SCHEDULER_ENABLE_POLLING=true           # Enable polling fallback
SCHEDULER_POLLING_INTERVAL=3s           # Polling interval
SCHEDULER_FALLBACK_TIMEOUT=30s          # Fallback timeout
SCHEDULER_RECONNECT_ATTEMPTS=5          # WebSocket reconnection attempts
SCHEDULER_RECONNECT_DELAY=5s            # Reconnection delay

# Required for real-time mode
ETHEREUM_WS_URL=wss://mainnet.infura.io/ws/v3/YOUR_PROJECT_ID
```

**Scheduler Modes:**
- `realtime`: WebSocket only (fastest, requires stable connection)
- `polling`: Traditional polling (most reliable)
- `hybrid`: WebSocket with polling fallback (recommended)

### Monitoring

- **Logs**: Structured JSON logs with configurable levels
- **Metrics**: Real-time performance metrics stored in MongoDB
- **Health Checks**: Periodic system health validation

### Data Access

Access the raw data through MongoDB collections:

```javascript
// Get latest blocks
db.blocks.find().sort({number: -1}).limit(10)

// Get transactions for a specific address
db.transactions.find({
  $or: [
    {from: "0x742d35Cc6e56A0e24C1D887FC9b50f08a2B6F4bC"},
    {to: "0x742d35Cc6e56A0e24C1D887FC9b50f08a2B6F4bC"}
  ]
}).sort({block_number: -1})

// Get system metrics
db.crawler_metrics.find().sort({timestamp: -1}).limit(1)
```

## Performance Optimization

### Concurrent Processing
- Configurable worker pools for parallel block processing
- Batch operations for database writes
- Connection pooling for MongoDB

### Resource Management
- Memory usage monitoring
- Graceful shutdown handling
- Error recovery with exponential backoff

### Monitoring
- Real-time performance metrics
- Health check endpoints
- Comprehensive logging

## Extending for Other Blockchains

The architecture is designed for multi-blockchain support:

1. **Create new blockchain service**:
```go
// internal/infrastructure/blockchain/polygon_service.go
type PolygonService struct {
    // Implementation for Polygon
}
```

2. **Add configuration**:
```go
// internal/infrastructure/config/config.go
type PolygonConfig struct {
    RPCURL string
    // Polygon-specific config
}
```

3. **Register in DI container**:
```go
// cmd/crawler/main.go
fx.Provide(
    fx.Annotate(
        blockchain.NewPolygonService,
        fx.As(new(service.BlockchainService)),
    ),
)
```

## Development

### Building
```bash
go build -o bin/crawler cmd/crawler/main.go
```

### Testing
```bash
go test ./...
```

### Linting
```bash
golangci-lint run
```

## Production Deployment

### Docker
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o crawler cmd/crawler/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/crawler .
CMD ["./crawler"]
```

### Kubernetes
See `deployments/k8s/` for Kubernetes manifests.

## Best Practices

### Data Organization
- **Immutable Storage**: Raw data is never modified
- **Comprehensive Indexing**: Optimized for query patterns
- **Time-series Structure**: Efficient for temporal analysis
- **Network Separation**: Isolated by blockchain network

### Monitoring
- **Metrics Collection**: Performance and operational metrics
- **Health Checks**: System component validation
- **Alerting**: Critical error notifications
- **Log Aggregation**: Centralized logging

### Error Handling
- **Graceful Degradation**: Continue processing on non-critical errors
- **Retry Logic**: Exponential backoff for transient failures
- **Circuit Breaker**: Protect against cascading failures
- **Dead Letter Queue**: Handle permanently failed items

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

For support and questions:
- Create an issue on GitHub
- Check the documentation in `/docs`
- Review the example configurations