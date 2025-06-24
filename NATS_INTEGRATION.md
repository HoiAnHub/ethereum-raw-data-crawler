# NATS JetStream Integration

## Tổng quan

Dự án Ethereum Raw Data Crawler đã được tích hợp với NATS JetStream để publish transaction events sau khi save thành công vào MongoDB. Điều này cho phép các service khác consume transaction data real-time một cách đáng tin cậy.

## Kiến trúc

```
┌─────────────────┐    ┌──────────────┐    ┌─────────────────┐
│   Ethereum      │───▶│   Crawler    │───▶│    MongoDB      │
│   Blockchain    │    │   Service    │    │   Database      │
└─────────────────┘    └──────┬───────┘    └─────────────────┘
                              │
                              ▼ (sau khi save thành công)
                       ┌──────────────┐    ┌─────────────────┐
                       │ NATS         │───▶│   Consumer      │
                       │ JetStream    │    │   Applications  │
                       └──────────────┘    └─────────────────┘
```

## Cấu hình

### Environment Variables

```bash
# NATS JetStream Configuration
NATS_URL=nats://localhost:4222
NATS_STREAM_NAME=TRANSACTIONS
NATS_SUBJECT_PREFIX=transactions
NATS_CONNECT_TIMEOUT=10s
NATS_RECONNECT_ATTEMPTS=5
NATS_RECONNECT_DELAY=2s
NATS_MAX_PENDING_MESSAGES=1000
NATS_ENABLED=true
```

### Stream Configuration

- **Stream Name**: `TRANSACTIONS`
- **Subject**: `transactions.events`
- **Storage**: File Storage (Persistent)
- **Retention**: Work Queue Policy
- **Max Messages**: 1,000,000
- **Max Bytes**: 1GB
- **Max Age**: 24 hours
- **Duplicate Detection**: 5 minutes

## Transaction Event Schema

Mỗi transaction được publish với schema sau:

```json
{
  "hash": "0x1234...",
  "from": "0xabcd...",
  "to": "0xefgh...",
  "value": "1000000000000000000",
  "data": "0x",
  "block_number": "12345",
  "block_hash": "0x5678...",
  "timestamp": "2024-01-15T10:30:00Z",
  "gas_used": "21000",
  "gas_price": "20000000000",
  "network": "ethereum"
}
```

## Setup và Chạy

### 1. Khởi động NATS Server

```bash
# Sử dụng Docker Compose
docker-compose -f docker-compose.nats.yml up -d

# Hoặc chạy NATS trực tiếp
nats-server -js -m 8222 -DV --store_dir ./nats-data
```

### 2. Kiểm tra NATS Health

```bash
# Kiểm tra server status
curl http://localhost:8222/

# Kiểm tra streams (nếu có nats CLI)
nats stream ls
nats stream info TRANSACTIONS
```

### 3. Khởi động Crawler với NATS enabled

```bash
# Cập nhật environment
export NATS_ENABLED=true

# Chạy scheduler
docker-compose -f docker-compose.scheduler.yml up -d
```

## Testing & Development

### 1. Chạy Consumer Example

```bash
cd examples
go run nats_consumer.go
```

### 2. Monitor NATS

- **Web UI**: http://localhost:8222
- **NATS Box**: `docker exec -it ethereum-nats-box nats`

```bash
# Trong NATS Box container
nats stream info TRANSACTIONS
nats consumer ls TRANSACTIONS
nats pub transactions.events '{"test": "message"}'
```

### 3. NATS NUI - Advanced GUI Management

[NATS NUI](https://github.com/nats-nui/nui) là một GUI management tool miễn phí và mã nguồn mở cho NATS với nhiều tính năng nâng cao.

#### Tính năng chính:

- **Core NATS Pub/Sub**: Xem và gửi NATS messages
- **Request/Reply**: Gửi requests và xem responses
- **Multiple format visualization**: Text, JSON, hex và nhiều format khác
- **Streams management**: Xem, tạo và điều chỉnh stream configs
- **Stream messages**: Xem, filter và thao tác với stream messages
- **Stream operations**: Purge và xóa messages
- **KV Store management**: Quản lý KV buckets
- **KV entries**: Xem, filter và edit entries
- **Multiple connections**: Hỗ trợ nhiều connections song song

#### Cài đặt và sử dụng:

**Option 1: Docker (Khuyến nghị)**
```bash
# Thêm vào docker-compose.nats.yml
services:
  nats-nui:
    image: ghcr.io/nats-nui/nui:latest
    container_name: ethereum-nats-nui
    ports:
      - "31311:31311"
    environment:
      NATS_URL: nats://ethereum-nats:4222
    volumes:
      - nats_nui_data:/db
    depends_on:
      - nats
    networks:
      - ethereum-network
```

**Option 2: Desktop App**
- Download từ [GitHub Releases](https://github.com/nats-nui/nui/releases)
- Hỗ trợ macOS, Windows, Linux
- Live Demo: [natsnui.app](https://natsnui.app)

**Option 3: Build từ source**
```bash
# Prerequisites: Go 1.21+, Node 18+, Wails.io
git clone https://github.com/nats-nui/nui.git
cd nui

# Web app
npm install
make dev-web

# Desktop app
make dev
```

#### Kết nối với NATS server:

1. Mở NATS NUI (web hoặc desktop)
2. Thêm connection mới:
   - **Name**: Ethereum NATS
   - **URL**: `nats://localhost:4222` (local) hoặc `nats://ethereum-nats:4222` (Docker)
   - **Description**: Ethereum Raw Data Crawler NATS
3. Connect và bắt đầu monitoring

#### Monitoring với NATS NUI:

**Stream Management:**
- Xem TRANSACTIONS stream details
- Monitor message count, size, consumers
- Real-time message throughput
- Stream configuration tweaking

**Message Operations:**
- Browse transaction events real-time
- Filter messages by subject, time range
- View message payload trong multiple formats (JSON, hex, text)
- Replay hoặc forward messages

**Consumer Management:**
- Monitor consumer performance
- View pending messages, acknowledgment rates
- Consumer configuration adjustments
- Debug consumer issues

**Performance Metrics:**
- Connection statistics
- Throughput graphs
- Memory và storage usage
- Error rates và patterns

#### Quick Access Commands:

**Start NATS với NUI:**
```bash
# Start toàn bộ NATS stack với NUI
make -f Makefile.nats nats-up

# Mở NATS NUI trong browser
make -f Makefile.nats nats-ui
```

**Monitoring Commands:**
```bash
# Monitor stream và consumer metrics
make -f Makefile.nats monitor

# Check health của tất cả services
make -f Makefile.nats health-check

# Access NATS management shell
make -f Makefile.nats nats-shell
```

#### Truy cập NATS NUI:

1. **Web Interface**: http://localhost:31311
2. **Connection Settings**:
   - **Name**: Ethereum NATS
   - **URL**: `nats://localhost:4222`
   - **Description**: Ethereum Raw Data Crawler
3. **Available Features**:
   - Stream browser và message viewer
   - Real-time throughput monitoring
   - Consumer performance metrics
   - Message filtering và search
   - Manual message publish/subscribe testing

### 3. Testing với NATS CLI

```bash
# Subscribe to events
nats sub "transactions.events"

# Check stream stats
nats stream report TRANSACTIONS

# Monitor consumer
nats consumer info TRANSACTIONS example-consumer
```

## Production Considerations

### 1. Monitoring

```bash
# Stream metrics
nats stream info TRANSACTIONS --json | jq '.state'

# Consumer lag monitoring
nats consumer info TRANSACTIONS <consumer-name> --json | jq '.num_pending'
```

### 2. Scaling

- **Multiple Consumers**: Sử dụng different consumer names cho load balancing
- **Clustered NATS**: Setup NATS cluster cho high availability
- **Resource Limits**: Monitor memory và disk usage

### 3. Error Handling

- **Message Retry**: MaxDeliver = 3
- **Dead Letter**: Messages sau 3 lần retry sẽ bị discard
- **Ack Timeout**: 30 seconds
- **Consumer Health**: Monitor consumer connection status

### 4. Backup & Recovery

```bash
# Backup stream data
nats stream backup TRANSACTIONS ./backup/

# Restore stream
nats stream restore TRANSACTIONS ./backup/
```

## Troubleshooting

### Common Issues

1. **NATS Connection Failed**
   ```bash
   # Check NATS server status
   docker logs ethereum-nats

   # Check port availability
   netstat -tulpn | grep 4222
   ```

2. **Stream Not Found**
   ```bash
       # Manually create stream
        nats stream add TRANSACTIONS \
      --subjects "transactions.events" \
      --storage file \
      --retention work \
      --max-msgs 1000000 \
      --max-bytes 1GB \
      --max-age 24h \
      --dupe-window 5m \
      --defaults
   ```

3. **Consumer Lag**
   ```bash
   # Check consumer info
   nats consumer info TRANSACTIONS <consumer-name>

   # Reset consumer to latest
   nats consumer rm TRANSACTIONS <consumer-name>
   ```

4. **NATS NUI Browser Error (HTTP 500)**
   ```
   Error: multiple non-filtered consumers not allowed on workqueue stream
   ```
   **Solution**: WorkQueue streams chỉ cho phép 1 consumer. Xóa existing consumers:
   ```bash
   # List consumers
   make -f Makefile.nats monitor

   # Delete conflicting consumer
   make -f Makefile.nats consumer-delete CONSUMER_NAME=example-consumer

   # Verify no consumers remain
   docker exec ethereum-nats-box nats consumer ls TRANSACTIONS
   ```

   **Alternative**: Tạo consumer có filter:
   ```bash
   nats consumer add TRANSACTIONS filtered-consumer \
     --filter "transactions.events" \
     --ack explicit --pull --defaults
   ```

5. **NATS NUI WorkQueue Consumer Error**
   ```
   Error: consumer must be deliver all on workqueue stream
   ```
   **Solution**: WorkQueue streams yêu cầu consumer có `deliver=all` policy:
   ```bash
   # Tạo consumer phù hợp cho WorkQueue streams
   make -f Makefile.nats consumer-create-workqueue CONSUMER_NAME=nui-consumer

   # Hoặc tạo thủ công
   docker exec ethereum-nats-box nats consumer add TRANSACTIONS nui-consumer \
     --deliver=all \
     --filter "transactions.events" \
     --ack explicit \
     --max-deliver 3 \
     --wait 30s \
     --replay instant \
     --pull \
     --defaults
   ```

4. **Message Publishing Failed**
   - Check crawler logs for NATS connection status
   - Verify NATS_ENABLED=true
   - Check stream capacity (messages/bytes)

### Debugging

```bash
# Enable debug logging
export NATS_DEBUG=true

# Crawler service logs
docker logs ethereum-scheduler

# NATS server logs
docker logs ethereum-nats
```

## Performance Tuning

### Publisher (Crawler)

- **Batch Publishing**: Transactions được publish theo batch
- **Error Handling**: Continue crawler nếu NATS publish failed
- **Connection Pooling**: Reuse NATS connections

### Consumer

- **Pull Subscription**: Dùng pull thay vì push để control rate
- **Batch Processing**: Fetch multiple messages cùng lúc
- **Parallel Processing**: Multiple goroutines cho heavy processing

### NATS Server

```bash
# Increase file descriptor limits
ulimit -n 65536

# Optimize storage
--store_dir=/fast-ssd-storage

# Memory limits
--max_memory=2GB
```

## Monitoring Dashboard

### Metrics to Track

1. **Stream Metrics**
   - Messages per second
   - Stream size (bytes)
   - Consumer lag

2. **Consumer Metrics**
   - Processing rate
   - Error rate
   - Acknowledgment rate

3. **System Metrics**
   - NATS server CPU/Memory
   - Disk usage (JetStream storage)
   - Network I/O

### Alerting

```bash
# High consumer lag
nats consumer info TRANSACTIONS <consumer> --json | \
  jq '.num_pending > 1000'

# Stream capacity warnings
nats stream info TRANSACTIONS --json | \
  jq '.state.bytes > (.config.max_bytes * 0.8)'
```

## Advanced Features

### 1. Multiple Environments

```bash
# Development
NATS_STREAM_NAME=TRANSACTIONS_DEV
NATS_SUBJECT_PREFIX=dev.transactions

# Production
NATS_STREAM_NAME=TRANSACTIONS_PROD
NATS_SUBJECT_PREFIX=prod.transactions
```

### 2. Message Filtering

```bash
# Create filtered consumers
nats consumer add TRANSACTIONS high-value-consumer \
  --filter-subject "transactions.events" \
  --replay-policy instant
```

### 3. Cross-Network Setup

```bash
# Multi-network support
NATS_SUBJECT_PREFIX=ethereum.transactions  # cho Ethereum
NATS_SUBJECT_PREFIX=polygon.transactions   # cho Polygon
```

## Security

### 1. Authentication

```bash
# Enable NATS auth
nats-server -js -m 8222 --auth <auth-file>
```

### 2. TLS

```bash
# TLS connection
NATS_URL=nats://localhost:4222?tls=true
```

### 3. Access Control

```bash
# User permissions in NATS config
{
  "users": [
    {
      "user": "crawler",
      "password": "secret",
      "permissions": {
        "publish": ["transactions.>"],
        "subscribe": []
      }
    }
  ]
}
```