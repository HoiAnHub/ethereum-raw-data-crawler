# Quick Start with External MongoDB

This guide helps you set up the Ethereum Block Scheduler with an external MongoDB instance.

## 🚀 Prerequisites

- **External MongoDB Instance**: MongoDB Atlas, self-hosted MongoDB, or any MongoDB service
- **Docker & Docker Compose**: For containerized deployment
- **Ethereum RPC Endpoint**: Infura, Alchemy, or your own Ethereum node

## 📋 Setup Steps

### 1. Clone and Setup

```bash
git clone <repository-url>
cd ethereum-raw-data-crawler
make setup
```

### 2. Configure Environment

```bash
# Copy environment template
cp env.example .env

# Edit .env with your settings
nano .env
```

**Required MongoDB Configuration:**
```bash
# Replace with your external MongoDB connection string
MONGO_URI=mongodb+srv://username:password@your-cluster.mongodb.net/ethereum_raw_data?retryWrites=true&w=majority
MONGO_DATABASE=ethereum_raw_data
MONGO_CONNECT_TIMEOUT=15s
MONGO_MAX_POOL_SIZE=10

# Ethereum RPC endpoints
ETHEREUM_RPC_URL=https://mainnet.infura.io/v3/YOUR_PROJECT_ID
ETHEREUM_WS_URL=wss://mainnet.infura.io/ws/v3/YOUR_PROJECT_ID
```

### 3. Test MongoDB Connection

```bash
# Test connection to external MongoDB
make test-mongodb
```

### 4. Setup MongoDB Indexes

```bash
# Create necessary indexes for optimal performance
make setup-mongodb
```

### 5. Start the Scheduler

```bash
# Start with Docker (recommended)
make scheduler-up

# Or use the deployment script
./scripts/deploy.sh fresh
```

## 🔧 Configuration Options

### MongoDB Connection String Examples

**MongoDB Atlas:**
```
MONGO_URI=mongodb+srv://username:password@cluster.mongodb.net/ethereum_raw_data?retryWrites=true&w=majority
```

**Self-hosted MongoDB:**
```
MONGO_URI=mongodb://username:password@localhost:27017/ethereum_raw_data?authSource=admin
```

**MongoDB with SSL:**
```
MONGO_URI=mongodb://username:password@localhost:27017/ethereum_raw_data?ssl=true&sslVerifyCertificate=false
```

### Performance Tuning

**For High-Volume Production:**
```bash
MONGO_MAX_POOL_SIZE=50
MONGO_CONNECT_TIMEOUT=30s
BATCH_SIZE=10
CONCURRENT_WORKERS=5
```

**For Conservative/Development:**
```bash
MONGO_MAX_POOL_SIZE=10
MONGO_CONNECT_TIMEOUT=15s
BATCH_SIZE=1
CONCURRENT_WORKERS=1
```

## 🐛 Troubleshooting

### Connection Issues

1. **Test MongoDB Connection:**
   ```bash
   make test-mongodb
   ```

2. **Check Network Connectivity:**
   ```bash
   # If using MongoDB Atlas, ensure your IP is whitelisted
   # If self-hosted, check firewall settings
   ```

3. **Verify Connection String:**
   - Ensure username/password are correct
   - Check if database name is included in URI
   - Verify authentication database (`authSource`)

### Performance Issues

1. **Check MongoDB Indexes:**
   ```bash
   make setup-mongodb
   ```

2. **Monitor Connection Pool:**
   ```bash
   # Check container logs
   make scheduler-logs
   ```

3. **Adjust Pool Settings:**
   - Increase `MONGO_MAX_POOL_SIZE` for high load
   - Adjust `MONGO_CONNECT_TIMEOUT` for slow networks

## 📊 Monitoring

### Health Checks

```bash
# Check service status
make scheduler-status

# View logs
make scheduler-logs

# Test environment
make env-check-full
```

### MongoDB Monitoring

The application includes built-in MongoDB health checks:
- Connection monitoring
- Automatic reconnection
- Performance metrics
- Error logging

## 🔄 Deployment Workflow

### Development
```bash
# 1. Test connection
make test-mongodb

# 2. Setup indexes
make setup-mongodb

# 3. Start with fresh build
./scripts/deploy.sh fresh

# 4. Monitor logs
make scheduler-logs
```

### Production
```bash
# 1. Use production environment
cp env.example.production .env.production
# Edit .env.production with production settings

# 2. Deploy to production
docker-compose -f docker-compose.scheduler.yml --env-file .env.production up -d

# 3. Verify deployment
make scheduler-status
make env-check-container
```

## 📝 Environment Variables Reference

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `MONGO_URI` | MongoDB connection string | - | ✅ |
| `MONGO_DATABASE` | Database name | `ethereum_raw_data` | ❌ |
| `MONGO_CONNECT_TIMEOUT` | Connection timeout | `15s` | ❌ |
| `MONGO_MAX_POOL_SIZE` | Connection pool size | `10` | ❌ |
| `ETHEREUM_RPC_URL` | Ethereum RPC endpoint | - | ✅ |
| `ETHEREUM_WS_URL` | Ethereum WebSocket endpoint | - | ✅ |

## 🆘 Support

If you encounter issues:

1. **Check logs:** `make scheduler-logs`
2. **Test connection:** `make test-mongodb`
3. **Verify environment:** `make env-check-full`
4. **Review configuration:** Check `.env` file settings

For MongoDB-specific issues, refer to your MongoDB provider's documentation.