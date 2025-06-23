# Hướng Dẫn Triển Khai Ethereum Block Scheduler

Tài liệu này hướng dẫn chi tiết cách triển khai Ethereum Block Scheduler lên VPS và kết nối MongoDB với các ứng dụng bên ngoài.

## 1. Chuẩn Bị VPS

### 1.1. Yêu Cầu Hệ Thống
- Ubuntu 20.04 LTS hoặc mới hơn
- Tối thiểu 2GB RAM, 2 CPU cores
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
git clone https://github.com/your-org/ethereum-raw-data-crawler.git
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

## 3. Kết Nối MongoDB Với Ứng Dụng Bên Ngoài

### 3.1. Cấu Hình Firewall
```bash
# Mở port MongoDB
sudo ufw allow 27017/tcp

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

## 4. Bảo Mật MongoDB

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

### 4.2. Giới Hạn IP Truy Cập
Sử dụng iptables để giới hạn IP có thể kết nối:

```bash
# Chỉ cho phép IP cụ thể kết nối đến port 27017
sudo iptables -A INPUT -p tcp -s TRUSTED_IP_ADDRESS --dport 27017 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 27017 -j DROP

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

## 7. Tối Ưu Hóa Hiệu Suất

### 7.1. Cấu Hình MongoDB Cho Hiệu Suất Cao
Thêm vào phần `command` của MongoDB trong `docker-compose.scheduler.yml`:

```yaml
command:
  - "--bind_ip_all"
  - "--wiredTigerCacheSizeGB=1"  # Điều chỉnh theo RAM có sẵn
  - "--setParameter=maxTransactionLockRequestTimeoutMillis=5000"
```

### 7.2. Cấu Hình Scheduler Cho Hiệu Suất Cao
Điều chỉnh các biến môi trường trong `.env`:

```
BATCH_SIZE=10
CONCURRENT_WORKERS=5
SCHEDULER_POLLING_INTERVAL=5s
```

## 8. Kết Luận

Bạn đã hoàn tất việc triển khai Ethereum Block Scheduler lên VPS và cấu hình MongoDB để có thể kết nối từ các ứng dụng bên ngoài. Hệ thống này sẽ liên tục thu thập dữ liệu blockchain Ethereum và lưu trữ vào MongoDB, cho phép các ứng dụng khác truy cập và phân tích dữ liệu này.

Để đảm bảo hệ thống hoạt động ổn định, hãy thường xuyên kiểm tra logs và giám sát hiệu suất của cả scheduler và MongoDB.