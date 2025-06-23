# Ethereum Block Scheduler

Đây là một ứng dụng scheduler riêng biệt để lắng nghe và xử lý các block mới từ Ethereum blockchain theo thời gian thực.

## Tính năng

- **Real-time Block Listening**: Sử dụng WebSocket để lắng nghe block mới ngay khi chúng được tạo ra
- **Hybrid Mode**: Kết hợp WebSocket real-time với polling fallback để đảm bảo không bỏ lỡ block
- **Configurable**: Có thể cấu hình các mode khác nhau (polling, realtime, hybrid)
- **Fault Tolerant**: Tự động reconnect WebSocket và fallback sang polling khi cần

## Các Mode Hoạt động

### 1. Realtime Mode (`SCHEDULER_MODE=realtime`)
- Chỉ sử dụng WebSocket để lắng nghe block mới
- Xử lý block ngay lập tức khi nhận được thông báo
- Tốc độ xử lý nhanh nhất nhưng phụ thuộc vào kết nối WebSocket

### 2. Polling Mode (`SCHEDULER_MODE=polling`)
- Sử dụng polling truyền thống, kiểm tra block mới theo interval
- Ổn định hơn nhưng có thể chậm hơn
- Phù hợp khi WebSocket không khả dụng

### 3. Hybrid Mode (`SCHEDULER_MODE=hybrid`) - **Khuyến nghị**
- Kết hợp cả WebSocket và polling
- Ưu tiên WebSocket, tự động fallback sang polling khi cần
- Cân bằng giữa tốc độ và độ tin cậy

## Cấu hình

Cấu hình scheduler thông qua các biến môi trường trong file `.env`:

```bash
# Scheduler Configuration
SCHEDULER_MODE=hybrid                    # polling, realtime, hybrid
SCHEDULER_ENABLE_REALTIME=true          # Bật/tắt WebSocket
SCHEDULER_ENABLE_POLLING=true           # Bật/tắt polling fallback
SCHEDULER_POLLING_INTERVAL=3s           # Interval cho polling
SCHEDULER_FALLBACK_TIMEOUT=30s          # Thời gian chờ trước khi fallback
SCHEDULER_RECONNECT_ATTEMPTS=5          # Số lần thử reconnect WebSocket
SCHEDULER_RECONNECT_DELAY=5s            # Delay giữa các lần reconnect

# Ethereum WebSocket URL (bắt buộc cho realtime mode)
ETHEREUM_WS_URL=wss://mainnet.infura.io/ws/v3/YOUR_PROJECT_ID
```

## Cách chạy

### 1. Development
```bash
# Từ thư mục root của project
go run cmd/schedulers/main.go
```

### 2. Build và chạy
```bash
# Build
go build -o bin/scheduler cmd/schedulers/main.go

# Chạy
./bin/scheduler
```

### 3. Docker
```bash
# Build Docker image
docker build -t ethereum-scheduler -f Dockerfile.scheduler .

# Chạy container
docker run --env-file .env ethereum-scheduler
```

## Logs và Monitoring

Scheduler sẽ log các thông tin quan trọng:

- Kết nối WebSocket
- Block mới được nhận
- Fallback sang polling
- Reconnection attempts
- Lỗi xử lý

Ví dụ log:
```
2025-06-23T10:30:15.123+0700	INFO	Starting Ethereum Block Scheduler	{"mode": "hybrid"}
2025-06-23T10:30:15.456+0700	INFO	Successfully connected to WebSocket
2025-06-23T10:30:16.789+0700	INFO	Received new block notification	{"block_number": "22759501"}
```

## So sánh với Crawler chính

| Tính năng | Main Crawler | Block Scheduler |
|-----------|--------------|-----------------|
| Mục đích | Crawl lịch sử blocks | Real-time block processing |
| Mode | Polling only | Realtime + Polling |
| Tốc độ | Batch processing | Immediate processing |
| Sử dụng | Historical data | Live data |

## Troubleshooting

### WebSocket connection failed
- Kiểm tra `ETHEREUM_WS_URL` có đúng không
- Đảm bảo API key có quyền WebSocket
- Thử chuyển sang `SCHEDULER_MODE=polling`

### Missing blocks
- Tăng `SCHEDULER_FALLBACK_TIMEOUT`
- Kiểm tra network connectivity
- Xem log để tìm lỗi reconnection

### High resource usage
- Giảm `SCHEDULER_POLLING_INTERVAL`
- Tăng `SCHEDULER_RECONNECT_DELAY`
- Sử dụng `SCHEDULER_MODE=realtime` nếu WebSocket ổn định
