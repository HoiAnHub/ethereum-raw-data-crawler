# 🚀 Quick Start Guide

## 📋 Overview

This guide covers:
1. **Basic Setup** - Getting the crawler running
2. **NATS JetStream Integration** - Real-time transaction events
3. **Development Workflow** - Code changes and testing
4. **Production Deployment** - Production-ready setup

## 🔧 Prerequisites

- Docker & Docker Compose
- Go 1.23+ (for development)
- Make utility

## 🔧 Setup Network

```bash
docker network create ethereum-raw-data-crawler_ethereum-network
```

## 1️⃣ Basic Setup

### Option A: Standard Setup (MongoDB Only)
```bash
# 1. Copy environment file
cp env.example .env

# 2. Edit configuration (MongoDB, Ethereum RPC, etc.)
nano .env

# 3. Start the crawler
make scheduler-up-fresh
```

### Option B: Full Stack with NATS JetStream
```bash
# 1. Copy environment file
cp env.example .env

# 2. Enable NATS in .env
echo "NATS_ENABLED=true" >> .env

# 3. Start NATS + Crawler (builds with latest code)
make -f Makefile.nats full-stack-up
```

**✅ Always builds with latest code!** The `full-stack-up` command includes `--build` flag to ensure your code changes are included.

## 2️⃣ NATS JetStream Integration 🆕

### Quick NATS Setup
```bash
# Setup complete development environment
make -f Makefile.nats setup-dev

# Start crawler with NATS enabled
make -f Makefile.nats crawler-start

# Run example consumer to see transaction events
make -f Makefile.nats run-consumer
```

### NATS Management Commands
```bash
# Monitor NATS stream and consumers
make -f Makefile.nats monitor

# Check health of all services
make -f Makefile.nats health-check

# View NATS logs
make -f Makefile.nats nats-logs

# Access NATS management shell
make -f Makefile.nats nats-shell

# Test publish/subscribe
make -f Makefile.nats publish-test
make -f Makefile.nats subscribe-test
```

### NATS Monitoring
- **NATS Web UI**: http://localhost:8222
- **NATS NUI (Advanced GUI)**:
  - Local access: http://localhost:31311
  - **Remote access (VPS)**: http://45.149.206.55:31311
- **Stream Name**: `TRANSACTIONS`
- **Subject**: `transactions.events`
- **Event Schema**: See [NATS_INTEGRATION.md](NATS_INTEGRATION.md)

**Open NATS NUI:**
```bash
# Local access
make -f Makefile.nats nats-ui

# Remote access (from any machine)
# Open browser and navigate to: http://45.149.206.55:31311
```

**🌐 Remote Access Setup:**
The NUI interface is configured to be accessible from outside the VPS. After running `make -f Makefile.nats full-stack-up`, you can access the NATS NUI from any machine using:
- **URL**: http://45.149.206.55:31311
- **No additional configuration needed** - the port is already bound to all interfaces

## 3️⃣ Development Workflow

### 🔥 IMPORTANT: Always Use Fresh Builds for Code Changes

When you make code changes, Docker cache can prevent your changes from being built.

### For Code Changes (WITHOUT NATS)
```bash
# 🔥 ALWAYS use this when you change code
make scheduler-up-fresh

# Check status
make env-check-container

# View logs
make scheduler-logs
```

### For Code Changes (WITH NATS)
```bash
# 1. Stop services
make -f Makefile.nats full-stack-down

# 2. Rebuild and start everything
make -f Makefile.nats full-stack-up

# 3. Check health
make -f Makefile.nats health-check

# 4. Monitor logs
make -f Makefile.nats crawler-logs
```

### Testing Your Changes
```bash
# Test transaction event publishing
make -f Makefile.nats run-consumer

# Verify MongoDB data
make env-check-full

# Check NATS stream statistics
make -f Makefile.nats monitor
```

## 4️⃣ Production Deployment

### Production with NATS
```bash
# 1. Setup production environment
cp .env.scheduler.example .env.production
nano .env.production

# 2. Enable NATS for production
echo "NATS_ENABLED=true" >> .env.production

# 3. Deploy production stack
ENV_FILE=.env.production make -f Makefile.nats full-stack-up

# 4. Verify deployment
make -f Makefile.nats health-check
```

### Production Monitoring
```bash
# Monitor stream performance
make -f Makefile.nats benchmark

# Check consumer lag
make -f Makefile.nats consumer-info

# Backup stream data
make -f Makefile.nats stream-backup
```

## 5️⃣ Verification & Testing

### Quick Health Check
```bash
# Standard services
make env-check-container

# Full stack with NATS
make -f Makefile.nats health-check
```

### Comprehensive Testing
```bash
# 1. Check MongoDB connectivity
make env-check-full

# 2. Verify NATS stream exists
make -f Makefile.nats stream-info

# 3. Test transaction event flow
make -f Makefile.nats publish-test

# 4. Monitor real transaction events
make -f Makefile.nats run-consumer
```

## 6️⃣ Common Workflows

### Development Workflow (Basic)
```bash
# 1. Make code changes
# 2. Force rebuild and start
make scheduler-up-fresh

# 3. Check environment
make env-check-container

# 4. View logs
make scheduler-logs
```

### Development Workflow (With NATS)
```bash
# 1. Make code changes
# 2. Restart full stack
make -f Makefile.nats full-stack-down
make -f Makefile.nats full-stack-up

# 3. Test transaction events
make -f Makefile.nats run-consumer

# 4. Monitor everything
make -f Makefile.nats health-check
```

### Production Workflow
```bash
# 1. Test in development first
make -f Makefile.nats setup-dev

# 2. Deploy to production
ENV_FILE=.env.production make -f Makefile.nats full-stack-up

# 3. Verify deployment
make -f Makefile.nats health-check

# 4. Setup monitoring
make -f Makefile.nats monitor
```

### Troubleshooting Workflow
```bash
# 1. Stop everything
make -f Makefile.nats full-stack-down

# 2. Clean Docker cache
make docker-clean-build

# 3. Reset NATS data (if needed)
make -f Makefile.nats reset-all

# 4. Start fresh
make -f Makefile.nats setup-dev
```

## 7️⃣ VPS Deployment with Remote Access 🌐

### Deploy to VPS with Remote NUI Access

**Prerequisites:**
- VPS with Docker and Docker Compose installed
- Port 31311 accessible from outside (firewall configured)
- Your VPS IP: `45.149.206.55`

### Step-by-Step VPS Deployment

```bash
# Option 1: Automated deployment (RECOMMENDED)
./scripts/vps-deploy.sh

# Option 2: Manual deployment
# 1. Clone repository on VPS
git clone <your-repo-url>
cd ethereum-raw-data-crawler

# 2. Setup environment
cp env.example .env
nano .env  # Configure your settings

# 3. Enable NATS for remote access
echo "NATS_ENABLED=true" >> .env

# 4. Deploy full stack with remote access
make -f Makefile.nats full-stack-up

# 5. Verify deployment
make -f Makefile.nats health-check
```

### Access NUI from Anywhere

After deployment, you can access the NATS NUI interface from any machine:

- **🌐 Remote Access URL**: http://45.149.206.55:31311
- **🔧 Local Access (on VPS)**: http://localhost:31311

### Firewall Configuration (if needed)

If you can't access the NUI from outside, ensure port 31311 is open:

```bash
# Ubuntu/Debian
sudo ufw allow 31311

# CentOS/RHEL
sudo firewall-cmd --permanent --add-port=31311/tcp
sudo firewall-cmd --reload

# Check if port is accessible
curl -I http://45.149.206.55:31311
```

### VPS Management Commands

```bash
# Start full stack with remote access
make -f Makefile.nats full-stack-up

# Stop all services
make -f Makefile.nats full-stack-down

# Check service health
make -f Makefile.nats health-check

# View logs
make -f Makefile.nats crawler-logs

# Monitor NATS metrics
make -f Makefile.nats monitor
```

### Security Considerations

- **Default Configuration**: NUI is accessible to anyone who can reach port 31311
- **Production Recommendation**: Consider using a reverse proxy with authentication
- **Network Security**: Ensure your VPS firewall is properly configured

## 8️⃣ Available Commands

### Basic Crawler Commands
```bash
make scheduler-up-fresh      # Force rebuild and start (RECOMMENDED)
make scheduler-down          # Stop crawler
make scheduler-logs          # View logs
make docker-clean-build      # Clean and rebuild everything
make env-check-container     # Check container environment
make env-check-full          # Full environment check
```

### NATS Management Commands
```bash
make -f Makefile.nats help          # Show all available commands
make -f Makefile.nats setup-dev     # Complete dev environment setup
make -f Makefile.nats full-stack-up # Start NATS + Crawler
make -f Makefile.nats health-check  # Check all services health
make -f Makefile.nats monitor       # Monitor stream metrics
make -f Makefile.nats run-consumer  # Run example consumer
make -f Makefile.nats teardown      # Stop everything
```

### Deploy Script Commands
```bash
./scripts/deploy.sh fresh    # Fresh development build
./scripts/deploy.sh prod     # Production deployment
./scripts/deploy.sh clean    # Clean everything and rebuild
./scripts/deploy.sh check    # Check status and environment
./scripts/deploy.sh logs     # Show logs
./scripts/deploy.sh stop     # Stop services
./scripts/vps-deploy.sh      # VPS deployment with remote NUI access
```

## 9️⃣ Environment Configuration

### Basic Configuration (.env)
```bash
# Ethereum Configuration
ETHEREUM_RPC_URL=https://mainnet.infura.io/v3/YOUR_KEY
ETHEREUM_WS_URL=wss://mainnet.infura.io/v3/YOUR_KEY
START_BLOCK_NUMBER=22759500

# MongoDB Configuration
MONGO_URI=mongodb://localhost:27017
MONGO_DATABASE=ethereum_raw_data

# NATS Configuration (NEW!)
NATS_ENABLED=true
NATS_URL=nats://localhost:4222
NATS_STREAM_NAME=TRANSACTIONS
NATS_SUBJECT_PREFIX=transactions
```

### Production Additions (.env.production)
```bash
# Additional production settings
NATS_RECONNECT_ATTEMPTS=10
NATS_MAX_PENDING_MESSAGES=10000
LOG_LEVEL=warn
BATCH_SIZE=10
CONCURRENT_WORKERS=5
```

## ⚠️ Common Mistakes to Avoid

### ❌ DON'T DO THIS:
```bash
# This might use cached/old code
make scheduler-up

# Starting NATS without creating stream
make -f Makefile.nats nats-up  # Missing stream setup
```

### ✅ DO THIS INSTEAD:
```bash
# This ensures latest code
make scheduler-up-fresh

# Complete NATS setup
make -f Makefile.nats setup-dev  # Includes stream creation
```

## 🔍 Monitoring & Debugging

### Real-time Monitoring
```bash
# Watch transaction events in real-time
make -f Makefile.nats run-consumer

# Monitor NATS metrics
make -f Makefile.nats monitor

# Watch crawler logs
make -f Makefile.nats crawler-logs
```

### Debug Issues
```bash
# Check all service health
make -f Makefile.nats health-check

# Verify NATS connection
make -f Makefile.nats nats-status

# Test message publishing
make -f Makefile.nats publish-test
```

## 🚨 Remember

- **Always use `fresh` commands when you change code**
- **Enable NATS with `NATS_ENABLED=true` for event streaming**
- **Use `make -f Makefile.nats setup-dev` for complete development setup**
- **Monitor transaction events with the consumer example**
- **Check health status before and after deployments**
- **For VPS deployment: Access NUI at http://45.149.206.55:31311**

## 🔗 Documentation Links

- **NATS Integration**: [NATS_INTEGRATION.md](NATS_INTEGRATION.md) - Complete NATS guide
- **Full Documentation**: [README.md](README.md) - Project overview
- **Environment Setup**: [env.example](env.example) - Configuration reference
- **Production Setup**: [.env.scheduler.example](.env.scheduler.example) - Production config

## 🆘 Need Help?

1. **Basic Issues**: Check [README.md](README.md)
2. **NATS Issues**: Check [NATS_INTEGRATION.md](NATS_INTEGRATION.md)
3. **Environment Issues**: Run `make env-check-full`
4. **NATS Issues**: Run `make -f Makefile.nats health-check`
5. **VPS Access Issues**: Check firewall and port 31311 accessibility