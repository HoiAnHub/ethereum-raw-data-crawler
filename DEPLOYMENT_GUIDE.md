# Hướng Dẫn Triển Khai Ethereum Block Scheduler

Tài liệu này hướng dẫn chi tiết cách triển khai Ethereum Block Scheduler lên VPS và kết nối MongoDB với các ứng dụng bên ngoài.

## 1. Chuẩn Bị VPS

### 1.1. Yêu Cầu Hệ Thống
- Ubuntu 20.04 LTS hoặc mới hơn
- Tối thiểu 4GB RAM, 2 CPU cores (tăng từ 2GB do thêm NATS)
- 50GB SSD
- Docker và Docker Compose

### 1.2. Cài Đặt Docker và Docker Compose
```bash
# Cập nhật hệ thống
sudo apt update && sudo apt upgrade -y

# Cài đặt các gói cần thiết
sudo apt install -y apt-transport-https ca-certificates curl software-properties-common

# Thêm Docker GPG key
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -

# Thêm Docker repository
sudo add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable"

# Cài đặt Docker
sudo apt update
sudo apt install -y docker-ce docker-ce-cli containerd.io

# Cài đặt Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/download/v2.20.3/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose

# Thêm user hiện tại vào nhóm docker
sudo usermod -aG docker $USER
```

Đăng xuất và đăng nhập lại để áp dụng thay đổi nhóm.

## 2. Triển Khai Dự Án

### 2.1. Clone Repository
```bash
# Clone repository
git clone https://github.com/HoiAnHub/ethereum-raw-data-crawler.git
cd ethereum-raw-data-crawler
```

### 2.2. Cấu Hình Môi Trường
```bash
# Tạo file .env từ mẫu
cp env.example .env

# Chỉnh sửa file .env
nano .env
```

Cấu hình các thông số quan trọng trong file `.env`:
```
# Ethereum RPC Configuration
ETHEREUM_RPC_URL=https://mainnet.infura.io/v3/YOUR_PROJECT_ID
ETHEREUM_WS_URL=wss://mainnet.infura.io/ws/v3/YOUR_PROJECT_ID
START_BLOCK_NUMBER=latest

# MongoDB Configuration
MONGO_URI=mongodb://admin:password@mongodb:27017/ethereum_raw_data?authSource=admin
MONGO_DATABASE=ethereum_raw_data

# Scheduler Configuration
SCHEDULER_MODE=hybrid
SCHEDULER_ENABLE_REALTIME=true
SCHEDULER_ENABLE_POLLING=true
SCHEDULER_POLLING_INTERVAL=3s
LOG_LEVEL=info

# NATS JetStream Configuration
NATS_URL=nats://nats:4222
NATS_STREAM_NAME=TRANSACTIONS
NATS_SUBJECT_PREFIX=transactions
NATS_CONNECT_TIMEOUT=10s
NATS_RECONNECT_ATTEMPTS=5
NATS_RECONNECT_DELAY=2s
NATS_MAX_PENDING_MESSAGES=1000
NATS_ENABLED=true
```

### 2.3. Cấu Hình MongoDB Cho Truy Cập Bên Ngoài
Chỉnh sửa file `docker-compose.scheduler.yml` để cho phép kết nối từ bên ngoài:

```bash
nano docker-compose.scheduler.yml
```

Đảm bảo phần cấu hình MongoDB có các thiết lập sau:
```yaml
mongodb:
  # ... các cấu hình khác ...
  ports:
    - "27017:27017"  # Map port ra host
  command:
    - "--bind_ip_all"  # Cho phép kết nối từ tất cả IP
  # ... các cấu hình khác ...
```

### 2.4. Triển Khai Với Docker Compose
```bash
# Khởi động dịch vụ
./scripts/run-scheduler.sh docker
```

Kiểm tra trạng thái:
```bash
docker ps
```

Xem logs:
```bash
./scripts/run-scheduler.sh logs --follow
```

### 2.5. Kiểm tra các biến môi trường
```bash
# Kiểm tra các biến môi trường trong container
docker exec ethereum-scheduler-app env
```

## 3. Kết Nối Với Các Dịch Vụ Bên Ngoài

### 3.0. Tổng Quan Các Dịch Vụ
Hệ thống cung cấp hai cách để truy cập dữ liệu:
- **MongoDB**: Truy cập trực tiếp vào database để query dữ liệu lịch sử
- **NATS JetStream**: Subscribe real-time events cho transaction mới

## 3.1. Kết Nối MongoDB Với Ứng Dụng Bên Ngoài

### 3.1. Cấu Hình Firewall
```bash
# Mở port MongoDB
sudo ufw allow 27017/tcp

# Mở port NATS (nếu cần truy cập từ bên ngoài)
sudo ufw allow 4222/tcp   # NATS client port
sudo ufw allow 8222/tcp   # NATS monitoring port

# Kích hoạt firewall nếu chưa kích hoạt
sudo ufw enable

# Kiểm tra trạng thái
sudo ufw status
```

### 3.2. Chuỗi Kết Nối MongoDB

#### Từ Máy Cùng Mạng LAN
```
mongodb://admin:password@VPS_IP_ADDRESS:27017/ethereum_raw_data?authSource=admin
```

#### Từ Internet (Cần Bảo Mật)
Nếu cần truy cập từ internet, nên thiết lập VPN hoặc SSH tunnel để bảo mật:

**SSH Tunnel:**
```bash
# Trên máy local
ssh -L 27017:localhost:27017 user@VPS_IP_ADDRESS
```

Sau đó kết nối qua:
```
mongodb://admin:password@localhost:27017/ethereum_raw_data?authSource=admin
```

### 3.3. Kiểm Tra Kết Nối

#### Sử Dụng MongoDB Compass
1. Tải và cài đặt [MongoDB Compass](https://www.mongodb.com/try/download/compass)
2. Kết nối với URI: `mongodb://admin:password@VPS_IP_ADDRESS:27017/ethereum_raw_data?authSource=admin`

#### Sử Dụng mongosh
```bash
mongosh "mongodb://admin:password@VPS_IP_ADDRESS:27017/ethereum_raw_data?authSource=admin"
```

### 3.4. Ví Dụ Kết Nối Từ Các Ngôn Ngữ Lập Trình

#### Node.js
```javascript
const { MongoClient } = require('mongodb');

const uri = "mongodb://admin:password@VPS_IP_ADDRESS:27017/ethereum_raw_data?authSource=admin";
const client = new MongoClient(uri);

async function main() {
  try {
    await client.connect();
    const db = client.db("ethereum_raw_data");
    const blocks = await db.collection("blocks").find().limit(10).toArray();
    console.log(blocks);
  } finally {
    await client.close();
  }
}

main().catch(console.error);
```

#### Python
```python
from pymongo import MongoClient

uri = "mongodb://admin:password@VPS_IP_ADDRESS:27017/ethereum_raw_data?authSource=admin"
client = MongoClient(uri)
db = client["ethereum_raw_data"]
blocks = list(db.blocks.find().limit(10))
print(blocks)
```

#### Go
```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
    uri := "mongodb://admin:password@VPS_IP_ADDRESS:27017/ethereum_raw_data?authSource=admin"
    client, err := mongo.NewClient(options.Client().ApplyURI(uri))
    if err != nil {
        log.Fatal(err)
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    err = client.Connect(ctx)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Disconnect(ctx)

    db := client.Database("ethereum_raw_data")
    cursor, err := db.Collection("blocks").Find(ctx, bson.M{})
    if err != nil {
        log.Fatal(err)
    }

    var blocks []bson.M
    if err = cursor.All(ctx, &blocks); err != nil {
        log.Fatal(err)
    }

    fmt.Println(blocks)
}
```

## 3.2. Kết Nối NATS JetStream

### 3.2.1. Chuỗi Kết Nối NATS

#### Từ Máy Cùng Mạng LAN
```
nats://VPS_IP_ADDRESS:4222
```

#### Monitoring Dashboard
Truy cập NATS monitoring dashboard tại:
```
http://VPS_IP_ADDRESS:8222
```

### 3.2.2. Ví Dụ Subscribe Transaction Events

#### Go
```go
package main

import (
    "encoding/json"
    "log"
    "github.com/nats-io/nats.go"
)

type TransactionEvent struct {
    Hash        string `json:"hash"`
    From        string `json:"from"`
    To          string `json:"to"`
    Value       string `json:"value"`
    BlockNumber string `json:"block_number"`
    Timestamp   string `json:"timestamp"`
    Network     string `json:"network"`
}

func main() {
    // Kết nối NATS
    nc, err := nats.Connect("nats://VPS_IP_ADDRESS:4222")
    if err != nil {
        log.Fatal(err)
    }
    defer nc.Close()

    // Tạo JetStream context
    js, err := nc.JetStream()
    if err != nil {
        log.Fatal(err)
    }

    // Subscribe transaction events
    sub, err := js.Subscribe("transactions.events", func(msg *nats.Msg) {
        var txEvent TransactionEvent
        if err := json.Unmarshal(msg.Data, &txEvent); err != nil {
            log.Printf("Error unmarshaling: %v", err)
            return
        }

        log.Printf("New transaction: %s from %s to %s",
            txEvent.Hash, txEvent.From, txEvent.To)

        // Acknowledge message
        msg.Ack()
    })
    if err != nil {
        log.Fatal(err)
    }
    defer sub.Unsubscribe()

    // Keep running
    select {}
}
```

#### Node.js
```javascript
const { connect, StringCodec } = require('nats');

async function main() {
    // Kết nối NATS
    const nc = await connect({ servers: 'nats://VPS_IP_ADDRESS:4222' });

    // Tạo JetStream context
    const js = nc.jetstream();

    // Subscribe transaction events
    const sub = await js.subscribe('transactions.events');

    console.log('Listening for transaction events...');

    for await (const msg of sub) {
        const txEvent = JSON.parse(StringCodec().decode(msg.data));
        console.log('New transaction:', txEvent.hash);
        msg.ack();
    }
}

main().catch(console.error);
```

#### Python
```python
import asyncio
import json
from nats.aio.client import Client as NATS
from nats.js.api import ConsumerConfig

async def main():
    nc = NATS()
    await nc.connect("nats://VPS_IP_ADDRESS:4222")

    js = nc.jetstream()

    async def message_handler(msg):
        tx_event = json.loads(msg.data.decode())
        print(f"New transaction: {tx_event['hash']}")
        await msg.ack()

    await js.subscribe("transactions.events", cb=message_handler)

    # Keep running
    await asyncio.sleep(3600)

if __name__ == '__main__':
    asyncio.run(main())
```

### 3.2.3. Kiểm Tra NATS Stream
```bash
# Cài đặt NATS CLI
curl -sf https://binaries.nats.dev/nats-io/natscli/nats@latest | sh

# Kiểm tra stream info
nats --server=nats://VPS_IP_ADDRESS:4222 stream info TRANSACTIONS

# Xem messages trong stream
nats --server=nats://VPS_IP_ADDRESS:4222 stream view TRANSACTIONS
```

## 4. Bảo Mật MongoDB và NATS

### 4.1. Tạo User Chỉ Đọc Cho Ứng Dụng Bên Ngoài
Kết nối vào MongoDB và tạo user chỉ đọc:

```javascript
db.createUser({
  user: "readonly_user",
  pwd: "secure_password",
  roles: [
    { role: "read", db: "ethereum_raw_data" }
  ]
})
```

Sau đó sử dụng user này để kết nối từ ứng dụng bên ngoài:
```
mongodb://readonly_user:secure_password@VPS_IP_ADDRESS:27017/ethereum_raw_data?authSource=admin
```

### 4.2. Bảo Mật NATS JetStream

#### 4.2.1. Cấu Hình Authentication (Tùy Chọn)
Để bảo mật NATS, có thể thêm authentication vào docker-compose.yml:

```yaml
nats:
  # ... cấu hình khác ...
  command: [
    "--jetstream",
    "--store_dir=/data",
    "--http_port=8222",
    "--port=4222",
    "--user=nats_user",
    "--pass=secure_password"
  ]
```

Sau đó cập nhật connection string:
```
nats://nats_user:secure_password@VPS_IP_ADDRESS:4222
```

### 4.3. Giới Hạn IP Truy Cập
Sử dụng iptables để giới hạn IP có thể kết nối:

```bash
# Chỉ cho phép IP cụ thể kết nối đến MongoDB
sudo iptables -A INPUT -p tcp -s TRUSTED_IP_ADDRESS --dport 27017 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 27017 -j DROP

# Chỉ cho phép IP cụ thể kết nối đến NATS
sudo iptables -A INPUT -p tcp -s TRUSTED_IP_ADDRESS --dport 4222 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 4222 -j DROP

# Lưu cấu hình
sudo apt install -y iptables-persistent
sudo netfilter-persistent save
```

## 5. Giám Sát và Bảo Trì

### 5.1. Giám Sát Logs
```bash
# Xem logs của scheduler
./scripts/run-scheduler.sh logs

# Xem logs của MongoDB
docker logs ethereum-scheduler-mongodb

# Xem logs của NATS
docker logs ethereum-crawler-nats

# Xem logs tất cả services
docker-compose logs -f
```

### 5.2. Sao Lưu MongoDB
```bash
# Tạo thư mục backup
mkdir -p ~/mongodb_backups

# Sao lưu database
docker exec ethereum-scheduler-mongodb mongodump --authenticationDatabase admin -u admin -p password --db ethereum_raw_data --out /data/db/backup

# Copy backup từ container ra host
docker cp ethereum-scheduler-mongodb:/data/db/backup ~/mongodb_backups/$(date +%Y%m%d)
```

### 5.3. Khởi Động Lại Dịch Vụ
```bash
# Dừng dịch vụ
./scripts/run-scheduler.sh stop

# Khởi động lại
./scripts/run-scheduler.sh docker
```

## 6. Xử Lý Sự Cố

### 6.1. Kiểm Tra Trạng Thái Container
```bash
docker ps -a
```

### 6.2. Kiểm Tra Logs Chi Tiết
```bash
docker logs ethereum-scheduler-app
```

### 6.3. Kiểm Tra Kết Nối MongoDB
```bash
docker exec -it ethereum-scheduler-mongodb mongosh -u admin -p password --authenticationDatabase admin
```

### 6.4. Kiểm Tra Kết Nối Ethereum RPC
```bash
curl -X POST -H "Content-Type: application/json" --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' $ETHEREUM_RPC_URL
```

### 6.5. Kiểm Tra NATS JetStream
```bash
# Kiểm tra NATS server status
curl http://VPS_IP_ADDRESS:8222/varz

# Kiểm tra JetStream info
curl http://VPS_IP_ADDRESS:8222/jsz

# Kiểm tra streams
curl http://VPS_IP_ADDRESS:8222/jsz?streams=1
```

## 7. Deployment và Update Code Mới Nhất

### 7.1. 🚀 Quy Trình Update Code Standard

Sau mỗi lần có thay đổi code, thực hiện các bước sau để deployment:

#### **7.1.1. Backup Log và Data Hiện Tại**
```bash
# 1. Backup logs hiện tại
cp scheduler.log scheduler_backup_$(date +%Y%m%d_%H%M%S).log

# 2. Backup database (tùy chọn cho update quan trọng)
mkdir -p ~/backups
docker exec ethereum-scheduler-mongodb mongodump --authenticationDatabase admin \
  -u admin -p password --db ethereum_raw_data --out /data/db/backup_$(date +%Y%m%d_%H%M%S)
```

#### **7.1.2. Graceful Stop Services**
```bash
# 1. Kiểm tra process hiện tại
ps aux | grep -E "(scheduler|crawler)" | grep -v grep

# 2. Stop scheduler process (nếu chạy binary)
kill $(ps aux | grep './scheduler' | grep -v grep | awk '{print $2}')

# 3. Hoặc stop Docker containers (nếu chạy Docker)
./scripts/run-scheduler.sh stop
# Hoặc
docker-compose -f docker-compose.scheduler.yml down
```

#### **7.1.3. Update Code**
```bash
# 1. Pull latest code
git pull origin main

# 2. Kiểm tra changes
git log --oneline -5

# 3. Kiểm tra dependencies có thay đổi không
git diff HEAD~1 go.mod go.sum
```

#### **7.1.4. Build và Deploy**

**Option A: Deploy Binary (Standalone)**
```bash
# 1. Build scheduler mới
go build -o scheduler cmd/schedulers/main.go

# 2. Verify binary
./scheduler --help || echo "Build successful"

# 3. Run scheduler
nohup ./scheduler > scheduler.log 2>&1 &

# 4. Verify deployment
ps aux | grep scheduler | grep -v grep
tail -f scheduler.log | head -20
```

**Option B: Deploy Docker (Recommended)**
```bash
# 1. Rebuild Docker images
docker-compose -f docker-compose.scheduler.yml build --no-cache

# 2. Start services
./scripts/run-scheduler.sh docker

# 3. Verify deployment
docker ps | grep -E "(scheduler|mongodb)"
./scripts/run-scheduler.sh logs --follow | head -20
```

#### **7.1.5. Post-Deployment Verification**
```bash
# 1. Kiểm tra service health
curl -f http://localhost:8080/health || echo "Health check endpoint not available"

# 2. Kiểm tra MongoDB connection
docker exec ethereum-scheduler-mongodb mongosh --eval "db.adminCommand('ping')" --quiet

# 3. Kiểm tra log không có errors
grep -i "error\|panic\|fatal" scheduler.log | tail -5

# 4. Kiểm tra block processing
grep "Block processed successfully" scheduler.log | tail -3

# 5. Monitor memory usage
ps -o pid,etime,rss,vsz,pmem,command -p $(pgrep scheduler)
```

### 7.2. 🔧 Update Scripts Tự Động

#### **7.2.1. Tạo Update Script**
Tạo file `scripts/update-deployment.sh`:

```bash
#!/bin/bash

# Ethereum Scheduler Update Script
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
LOG_FILE="/tmp/update-deployment-$(date +%Y%m%d_%H%M%S).log"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log() {
    echo -e "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

error_exit() {
    log "${RED}ERROR: $1${NC}"
    exit 1
}

# Function to check if service is running
check_service_running() {
    if docker ps | grep -q "ethereum-scheduler-app"; then
        return 0
    elif pgrep -f "./scheduler" > /dev/null; then
        return 0
    else
        return 1
    fi
}

# Main update function
main() {
    log "${BLUE}=== ETHEREUM SCHEDULER UPDATE PROCESS ===${NC}"

    cd "$PROJECT_DIR" || error_exit "Cannot access project directory"

    # 1. Pre-update checks
    log "${YELLOW}1. Pre-update checks...${NC}"
    git status || error_exit "Not a git repository"

    # 2. Backup current state
    log "${YELLOW}2. Backing up current state...${NC}"
    if [ -f "scheduler.log" ]; then
        cp scheduler.log "scheduler_backup_$(date +%Y%m%d_%H%M%S).log"
        log "${GREEN}✓ Log backup created${NC}"
    fi

    # 3. Stop current services
    log "${YELLOW}3. Stopping current services...${NC}"
    if check_service_running; then
        # Try Docker first
        if docker ps | grep -q "ethereum-scheduler"; then
            log "Stopping Docker containers..."
            ./scripts/run-scheduler.sh stop || docker-compose -f docker-compose.scheduler.yml down
        fi

        # Then check for binary process
        if pgrep -f "./scheduler" > /dev/null; then
            log "Stopping scheduler binary..."
            pkill -f "./scheduler" || true
            sleep 3
        fi

        log "${GREEN}✓ Services stopped${NC}"
    else
        log "${GREEN}✓ No services running${NC}"
    fi

    # 4. Update code
    log "${YELLOW}4. Updating code...${NC}"
    git fetch origin
    local_commit=$(git rev-parse HEAD)
    remote_commit=$(git rev-parse origin/main)

    if [ "$local_commit" != "$remote_commit" ]; then
        log "Pulling latest changes..."
        git pull origin main || error_exit "Git pull failed"
        log "${GREEN}✓ Code updated (${remote_commit:0:8})${NC}"
    else
        log "${GREEN}✓ Code already up to date${NC}"
    fi

    # 5. Check for dependency changes
    log "${YELLOW}5. Checking dependencies...${NC}"
    if git diff HEAD~1 go.mod go.sum | grep -q .; then
        log "Dependencies changed, updating..."
        go mod download && go mod tidy
        log "${GREEN}✓ Dependencies updated${NC}"
    else
        log "${GREEN}✓ No dependency changes${NC}"
    fi

    # 6. Build and deploy
    log "${YELLOW}6. Building and deploying...${NC}"

    # Choose deployment method
    if [ "${DEPLOY_METHOD:-docker}" == "binary" ]; then
        log "Building binary..."
        go build -o scheduler cmd/schedulers/main.go || error_exit "Build failed"

        log "Starting scheduler binary..."
        nohup ./scheduler > scheduler.log 2>&1 &
        SCHEDULER_PID=$!
        log "Scheduler started with PID: $SCHEDULER_PID"
    else
        log "Building and starting Docker containers..."
        docker-compose -f docker-compose.scheduler.yml build --no-cache || error_exit "Docker build failed"
        ./scripts/run-scheduler.sh docker || error_exit "Docker start failed"
    fi

    log "${GREEN}✓ Deployment completed${NC}"

    # 7. Post-deployment verification
    log "${YELLOW}7. Verifying deployment...${NC}"
    sleep 10

    if check_service_running; then
        log "${GREEN}✓ Service is running${NC}"

        # Check logs for errors
        if [ -f "scheduler.log" ]; then
            if grep -i "error\|panic\|fatal" scheduler.log | tail -1 | grep -q .; then
                log "${YELLOW}⚠ Recent errors found in logs${NC}"
                grep -i "error\|panic\|fatal" scheduler.log | tail -3
            else
                log "${GREEN}✓ No recent errors in logs${NC}"
            fi
        fi

        # Check block processing
        if [ -f "scheduler.log" ]; then
            if grep "Block processed successfully\|Started" scheduler.log | tail -1 | grep -q .; then
                log "${GREEN}✓ Scheduler appears to be processing correctly${NC}"
            else
                log "${YELLOW}⚠ No recent block processing activity${NC}"
            fi
        fi
    else
        error_exit "Service failed to start"
    fi

    log "${GREEN}=== UPDATE COMPLETED SUCCESSFULLY ===${NC}"
    log "${BLUE}Update log saved to: $LOG_FILE${NC}"
    log "${BLUE}Monitor logs with: tail -f scheduler.log${NC}"
}

# Handle command line arguments
case "${1:-}" in
    --binary)
        export DEPLOY_METHOD="binary"
        main
        ;;
    --docker)
        export DEPLOY_METHOD="docker"
        main
        ;;
    --help)
        echo "Usage: $0 [--binary|--docker|--help]"
        echo "  --binary  Deploy as standalone binary"
        echo "  --docker  Deploy using Docker (default)"
        echo "  --help    Show this help"
        ;;
    *)
        main
        ;;
esac
```

#### **7.2.2. Sử Dụng Update Script**
```bash
# Cấp quyền execute
chmod +x scripts/update-deployment.sh

# Update với Docker (recommended)
./scripts/update-deployment.sh --docker

# Hoặc update với binary
./scripts/update-deployment.sh --binary

# Xem help
./scripts/update-deployment.sh --help
```

### 7.3. 🔄 Rollback Strategy

#### **7.3.1. Quick Rollback**
```bash
# 1. Rollback code
git log --oneline -5
git reset --hard HEAD~1  # Rollback 1 commit

# 2. Redeploy
./scripts/update-deployment.sh --docker
```

#### **7.3.2. Restore from Backup**
```bash
# 1. Stop services
./scripts/run-scheduler.sh stop

# 2. Restore log backup (if needed)
cp scheduler_backup_YYYYMMDD_HHMMSS.log scheduler.log

# 3. Restore database backup (if needed)
docker exec ethereum-scheduler-mongodb mongorestore \
  --authenticationDatabase admin -u admin -p password \
  --db ethereum_raw_data /data/db/backup_YYYYMMDD_HHMMSS/ethereum_raw_data

# 4. Restart services
./scripts/run-scheduler.sh docker
```

### 7.4. 📊 Monitoring After Deployment

#### **7.4.1. Health Check Script**
Tạo file `scripts/health-check.sh`:

```bash
#!/bin/bash

# Quick health check after deployment
echo "=== SCHEDULER HEALTH CHECK ==="

# Check process
if ps aux | grep -E "(scheduler|ethereum-scheduler-app)" | grep -v grep; then
    echo "✓ Process running"
else
    echo "✗ Process not found"
fi

# Check recent logs
if [ -f "scheduler.log" ]; then
    echo "Recent log entries:"
    tail -5 scheduler.log | while read line; do
        echo "  $line"
    done

    # Check for errors
    error_count=$(grep -i "error\|panic\|fatal" scheduler.log | wc -l)
    echo "Total errors in log: $error_count"
else
    echo "✗ Log file not found"
fi

# Check MongoDB connection
if docker exec ethereum-scheduler-mongodb mongosh --eval "db.adminCommand('ping')" --quiet 2>/dev/null; then
    echo "✓ MongoDB connection OK"
else
    echo "✗ MongoDB connection failed"
fi

# Check memory usage
echo "Memory usage:"
ps -o pid,rss,vsz,pmem,command -C scheduler 2>/dev/null || echo "  No scheduler process found"
```

#### **7.4.2. Monitoring Commands**
```bash
# Quick health check
./scripts/health-check.sh

# Real-time log monitoring
tail -f scheduler.log | grep -E "(ERROR|WARN|Block processed|Started)"

# Memory monitoring
watch 'ps -o pid,rss,vsz,pmem,etime,command -C scheduler'

# Docker monitoring (if using Docker)
docker stats ethereum-scheduler-app ethereum-scheduler-mongodb
```

### 7.5. 🚨 Emergency Procedures

#### **7.5.1. Emergency Stop**
```bash
# Force stop all related processes
pkill -f scheduler
docker-compose -f docker-compose.scheduler.yml down --remove-orphans
docker system prune -f
```

#### **7.5.2. Emergency Recovery**
```bash
# 1. Clean restart
./scripts/run-scheduler.sh stop
docker system prune -f

# 2. Fresh deployment
git reset --hard origin/main
./scripts/update-deployment.sh --docker

# 3. If still failing, restore from backup
git reset --hard LAST_KNOWN_GOOD_COMMIT
./scripts/update-deployment.sh --docker
```

## 8. Tối Ưu Hóa Hiệu Suất

### 8.1. Cấu Hình MongoDB Cho Hiệu Suất Cao
Thêm vào phần `command` của MongoDB trong `docker-compose.scheduler.yml`:

```yaml
command:
  - "--bind_ip_all"
  - "--wiredTigerCacheSizeGB=1"  # Điều chỉnh theo RAM có sẵn
  - "--setParameter=maxTransactionLockRequestTimeoutMillis=5000"
```

### 8.2. Cấu Hình Scheduler Cho Hiệu Suất Cao
Điều chỉnh các biến môi trường trong `.env`:

```
BATCH_SIZE=10
CONCURRENT_WORKERS=5
SCHEDULER_POLLING_INTERVAL=5s
```

## 9. Kết Luận

Bạn đã hoàn tất việc triển khai Ethereum Raw Data Crawler lên VPS với đầy đủ các thành phần:

- **Ethereum Crawler**: Thu thập dữ liệu blockchain Ethereum
- **MongoDB**: Lưu trữ dữ liệu lịch sử
- **NATS JetStream**: Stream real-time transaction events
- **Monitoring**: Giám sát hệ thống
- **Auto Deployment**: Scripts tự động update và deployment

### 🚀 **Quy Trình Deployment Chuẩn:**

```bash
# 1. Pull và update code mới nhất
git pull origin main

# 2. Auto deployment với Docker (recommended)
./scripts/update-deployment.sh --docker

# 3. Kiểm tra health sau deployment
./scripts/health-check.sh

# 4. Monitor logs
tail -f scheduler.log | grep -E "(ERROR|WARN|Block processed|Started)"
```

### 🛠 **Công Cụ Quản Lý:**

1. **Update Script**: `./scripts/update-deployment.sh`
   - Tự động backup, stop, update, build, deploy
   - Hỗ trợ cả Docker và binary deployment
   - Verification tự động sau deployment

2. **Health Check**: `./scripts/health-check.sh`
   - Kiểm tra process, logs, MongoDB, memory
   - Tóm tắt tình trạng hệ thống
   - Phát hiện sớm các vấn đề

3. **Monitoring Scripts**: Trong thư mục `scripts/`
   - `run-scheduler.sh`: Quản lý Docker containers
   - `monitor-mongodb.sh`: Monitor MongoDB health
   - `check-data-flow.sh`: Kiểm tra data flow

### 📊 **Hai Cách Truy Cập Dữ Liệu:**

1. **Database Access**: Truy vấn dữ liệu lịch sử qua MongoDB
2. **Real-time Events**: Subscribe transaction events qua NATS JetStream

### 🔒 **Best Practices:**

- **Backup thường xuyên**: Sử dụng scripts backup tự động
- **Monitor logs**: Kiểm tra errors và warnings định kỳ
- **Health checks**: Chạy health check sau mỗi deployment
- **Update security**: Cập nhật dependencies và security patches
- **Resource monitoring**: Theo dõi CPU, memory, disk space
- **Network monitoring**: Kiểm tra connectivity với Ethereum nodes

### 🆘 **Troubleshooting:**

Nếu gặp vấn đề, thực hiện theo thứ tự:

1. **Check health**: `./scripts/health-check.sh`
2. **Check logs**: `tail -100 scheduler.log | grep -i error`
3. **Emergency restart**: `./scripts/update-deployment.sh --docker`
4. **Rollback**: `git reset --hard HEAD~1 && ./scripts/update-deployment.sh`

**Với hệ thống này, bạn có thể dễ dàng maintain và scale Ethereum data crawler một cách chuyên nghiệp!** 🎉