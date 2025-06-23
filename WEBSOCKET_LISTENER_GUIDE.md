# WebSocket Listener Service Guide

## Giới thiệu

WebSocket Listener Service là một service riêng biệt trong hệ thống Ethereum Raw Data Crawler, được thiết kế để lắng nghe các sự kiện real-time từ Ethereum blockchain thông qua WebSocket và ghi trực tiếp vào MongoDB.

## Tính năng

### 🔄 Real-time Data Streaming
- Lắng nghe new blocks theo thời gian thực
- Lắng nghe pending transactions (tùy chọn)
- Lắng nghe contract logs (tùy chọn)

### 📊 Optimized Performance
- Batch processing để tối ưu database writes
- Buffering với flush interval có thể cấu hình
- Connection pooling và retry logic
- Health monitoring và auto-reconnection

### 🔧 Flexible Configuration
- Có thể enable/disable từng loại subscription
- Cấu hình batch size và flush interval
- Timeout và retry settings
- NATS integration cho notifications (tùy chọn)

### 🏗️ Robust Architecture
- Graceful shutdown handling
- Connection health monitoring
- Automatic reconnection với backoff
- Comprehensive logging và metrics

## Kiến trúc

```
┌─────────────────┐    WebSocket    ┌──────────────────┐
│ Ethereum Node   │◄──────────────► │ WebSocket        │
│ (Infura/Alchemy)│                 │ Listener Service │
└─────────────────┘                 └──────────────────┘
                                             │
                                    ┌────────▼────────┐
                                    │ Data Processing │
                                    │ & Buffering     │
                                    └────────┬────────┘
                                             │
                              ┌──────────────▼──────────────┐
                              │         MongoDB             │
                              │ (Blocks & Transactions)     │
                              └─────────────────────────────┘
                                             │
                                    ┌────────▼────────┐
                                    │ NATS JetStream  │
                                    │ (Notifications) │
                                    └─────────────────┘
```

## Cài đặt và Chạy

### Prerequisites

- Go 1.21+
- Docker & Docker Compose
- Ethereum WebSocket endpoint (Infura, Alchemy, hoặc node riêng)

### 1. Cấu hình Environment Variables

Tạo file `.env` hoặc cấu hình các biến môi trường:

```bash
# Ethereum Configuration
ETHEREUM_WS_URL=wss://mainnet.infura.io/ws/v3/YOUR_PROJECT_ID
ETHEREUM_RPC_URL=https://mainnet.infura.io/v3/YOUR_PROJECT_ID

# MongoDB Configuration
MONGO_URI=mongodb://admin:password@localhost:27018/ethereum_raw_data?authSource=admin
MONGO_DATABASE=ethereum_raw_data

# WebSocket Configuration
WEBSOCKET_BATCH_SIZE=20
WEBSOCKET_FLUSH_INTERVAL=2s
WEBSOCKET_SUBSCRIBE_TO_BLOCKS=true
WEBSOCKET_SUBSCRIBE_TO_TXS=false
WEBSOCKET_SUBSCRIBE_TO_LOGS=false

# NATS Configuration (optional)
NATS_ENABLED=true
NATS_URL=nats://localhost:4222
```

### 2. Chạy với Docker Compose (Recommended)

```bash
# Start WebSocket Listener services
make websocket-up

# View logs
make websocket-logs

# Stop services
make websocket-down
```

### 3. Chạy Local Development

```bash
# Build service
make build-websocket

# Run locally
make run-websocket
```

### 4. Chạy với Docker

```bash
# Build Docker image
make docker-build-websocket

# Run với docker-compose
docker-compose -f docker-compose.websocket-listener.yml up -d
```

## Configuration

### WebSocket Settings

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `WEBSOCKET_RECONNECT_ATTEMPTS` | 10 | Max reconnection attempts |
| `WEBSOCKET_RECONNECT_DELAY` | 3s | Delay between reconnections |
| `WEBSOCKET_PING_INTERVAL` | 30s | Ping interval for health check |
| `WEBSOCKET_READ_TIMEOUT` | 60s | Read timeout |
| `WEBSOCKET_WRITE_TIMEOUT` | 10s | Write timeout |
| `WEBSOCKET_BUFFER_SIZE` | 500 | Message buffer size |
| `WEBSOCKET_BATCH_SIZE` | 20 | Database batch size |
| `WEBSOCKET_FLUSH_INTERVAL` | 2s | Buffer flush interval |
| `WEBSOCKET_MAX_RETRIES` | 5 | Max operation retries |
| `WEBSOCKET_RETRY_DELAY` | 1s | Retry delay |

### Subscription Settings

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `WEBSOCKET_SUBSCRIBE_TO_BLOCKS` | true | Subscribe to new blocks |
| `WEBSOCKET_SUBSCRIBE_TO_TXS` | false | Subscribe to pending transactions |
| `WEBSOCKET_SUBSCRIBE_TO_LOGS` | false | Subscribe to contract logs |

### Performance Tuning

#### High Volume Setup (>1000 TPS)
```bash
WEBSOCKET_BATCH_SIZE=50
WEBSOCKET_FLUSH_INTERVAL=1s
WEBSOCKET_BUFFER_SIZE=1000
MONGO_MAX_POOL_SIZE=50
```

#### Low Latency Setup
```bash
WEBSOCKET_BATCH_SIZE=5
WEBSOCKET_FLUSH_INTERVAL=500ms
WEBSOCKET_BUFFER_SIZE=100
```

#### Memory Constrained Setup
```bash
WEBSOCKET_BATCH_SIZE=10
WEBSOCKET_BUFFER_SIZE=200
MONGO_MAX_POOL_SIZE=10
```

## Monitoring và Debugging

### Health Checks

Service cung cấp health check endpoints và metrics:

```bash
# Check container health
docker ps

# View detailed logs
make websocket-logs

# Check service status
make websocket-status
```

### Log Levels

```bash
LOG_LEVEL=debug  # Detailed debugging
LOG_LEVEL=info   # Standard operation
LOG_LEVEL=warn   # Warnings only
LOG_LEVEL=error  # Errors only
```

### Common Issues

#### 1. WebSocket Connection Issues
```bash
# Check logs for connection errors
make websocket-logs | grep "websocket"

# Verify WebSocket URL
echo $ETHEREUM_WS_URL
```

#### 2. Database Connection Issues
```bash
# Check MongoDB connection
make websocket-mongo-shell

# Verify MongoDB status
docker exec ethereum-websocket-mongodb mongosh --eval "db.adminCommand('ping')"
```

#### 3. Memory Issues
```bash
# Check memory usage
docker stats ethereum-websocket-listener-app

# Reduce batch size if needed
WEBSOCKET_BATCH_SIZE=10
```

## NATS Integration

Service có thể publish notifications về các events thông qua NATS JetStream:

### Enable NATS
```bash
NATS_ENABLED=true
NATS_URL=nats://localhost:4222
```

### Message Topics
- `ethereum.realtime.blocks.new` - New block notifications
- `ethereum.realtime.metrics` - Service metrics

### Subscribe to Notifications

```bash
# Install NATS CLI
go install github.com/nats-io/natscli/nats@latest

# Subscribe to new blocks
nats sub "ethereum.realtime.blocks.new"

# View stream info
nats stream info ETHEREUM_REALTIME
```

## Deployment

### Production Deployment

1. **Resource Requirements**
   ```yaml
   resources:
     limits:
       memory: 1024M
       cpus: '1.0'
     reservations:
       memory: 512M
       cpus: '0.5'
   ```

2. **Network Configuration**
   - Ensure WebSocket endpoint is accessible
   - Configure firewall rules for MongoDB (27018)
   - Configure NATS ports if needed (4222, 8222)

3. **Scaling Considerations**
   - Chỉ chạy 1 instance để tránh duplicate data
   - Scale MongoDB nếu cần thiết
   - Monitor connection health

### High Availability Setup

```bash
# Monitor service health
docker-compose -f docker-compose.websocket-listener.yml up -d

# Setup health check alerts
# Use external monitoring tools (Prometheus, Grafana)
```

## Performance Benchmarks

### Test Results (Mainnet)

| Metric | Value |
|--------|-------|
| Blocks/second | ~0.2 |
| Transactions/block | ~150-300 |
| Memory usage | ~200-500MB |
| CPU usage | ~10-30% |
| Database writes/second | ~5-10 |

### Optimization Tips

1. **Batch Size Tuning**
   - Tăng batch size cho high throughput
   - Giảm batch size cho low latency

2. **Flush Interval**
   - Giảm flush interval cho real-time requirements
   - Tăng flush interval cho batch efficiency

3. **Connection Pool**
   - Tăng MongoDB connection pool cho high load
   - Monitor connection usage

## Troubleshooting

### Common Commands

```bash
# Restart service
make websocket-restart

# Check all service status
make status-all

# View live logs
make websocket-logs-all

# Connect to MongoDB
make websocket-mongo-shell

# Check NATS status
docker exec ethereum-websocket-nats nats server check

# Cleanup and restart
make websocket-down && make websocket-up
```

### Debug Mode

```bash
# Enable debug logging
LOG_LEVEL=debug make run-websocket

# Check specific component logs
make websocket-logs | grep "websocket-listener"
```

## Comparison với Scheduler

| Feature | WebSocket Listener | Scheduler |
|---------|-------------------|-----------|
| **Data Source** | WebSocket real-time | Polling RPC |
| **Latency** | ~1-2 seconds | ~5-15 seconds |
| **Use Case** | Real-time applications | Historical sync |
| **Resource Usage** | Lower | Higher |
| **Reliability** | Connection dependent | More robust |
| **Deployment** | Independent service | Independent service |

## Best Practices

1. **Monitoring**
   - Setup alerts cho connection failures
   - Monitor memory và CPU usage
   - Track database write performance

2. **Configuration**
   - Test cấu hình trên testnet trước
   - Tuning theo specific use case
   - Regular backup MongoDB data

3. **Security**
   - Secure WebSocket connections
   - MongoDB authentication
   - Network security rules

4. **Maintenance**
   - Regular log rotation
   - Monitor disk space
   - Update dependencies

## Support

Nếu gặp vấn đề, hãy:

1. Check logs: `make websocket-logs`
2. Verify configuration trong `.env`
3. Test connection manually
4. Check GitHub issues
5. Create detailed bug report với logs và configuration

---

**Note**: Service này được thiết kế để chạy độc lập với các service khác. Có thể chạy đồng thời với Scheduler để có cả real-time và historical data.