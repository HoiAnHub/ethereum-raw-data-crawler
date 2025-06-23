# MongoDB Stability Guide for Ethereum Scheduler

This guide provides solutions for MongoDB connection stability issues when using self-hosted MongoDB on VPS.

## Problem Description

When using MongoDB Atlas (cloud), the crawler runs smoothly for extended periods. However, with self-hosted MongoDB on VPS, the crawler stops writing data after a few minutes due to connection issues.

## Root Causes

1. **Connection Pool Exhaustion**: Default MongoDB connection settings may not be optimal for VPS environments
2. **Network Timeouts**: VPS networks may have different timeout characteristics than cloud services
3. **Resource Constraints**: VPS may have limited memory/CPU affecting MongoDB performance
4. **Missing Connection Recovery**: Application may not properly handle connection drops

## Solutions Implemented

### 1. Enhanced MongoDB Configuration

**File**: `docker-compose.scheduler.yml`
- Increased connection limits (`--maxConns=1000`)
- Optimized WiredTiger cache size (`--wiredTigerCacheSizeGB=0.5`)
- Added proper logging and journaling
- Enhanced health checks with shorter intervals

**File**: `internal/infrastructure/database/mongodb.go`
- Added connection pooling with min/max pool sizes
- Implemented retry logic for connection failures
- Added heartbeat monitoring
- Enhanced timeout configurations

### 2. Application-Level Improvements

**File**: `internal/application/service/crawler_service.go`
- Comprehensive health checking for all components
- Automatic recovery mechanisms
- Connection failure tracking and alerting
- Enhanced error handling and logging

**Files**: `internal/adapters/secondary/*_repository_impl.go`
- Retry logic for database operations
- Connection error detection and handling
- Graceful degradation on connection issues

### 3. Monitoring and Diagnostics

**Scripts**:
- `scripts/monitor-mongodb.sh`: Real-time MongoDB monitoring
- `scripts/optimize-mongodb.sh`: MongoDB optimization automation
- `scripts/test-mongodb-stability.sh`: Connection stability testing
- `scripts/deploy-production.sh`: Production deployment automation

## Quick Start

### 1. Deploy with Optimized Configuration

```bash
# Copy production environment template
cp .env.scheduler.production .env.scheduler.local

# Edit configuration (set your Ethereum RPC URL)
nano .env.scheduler.local

# Deploy with optimizations
./scripts/deploy-production.sh
```

### 2. Monitor System Health

```bash
# Check service status
./scripts/deploy-production.sh status

# View logs
./scripts/deploy-production.sh logs

# Run stability test
./scripts/deploy-production.sh test
```

### 3. Troubleshooting

```bash
# Start MongoDB monitoring
./scripts/monitor-mongodb.sh

# Optimize MongoDB settings
./scripts/optimize-mongodb.sh

# Check MongoDB status
./scripts/optimize-mongodb.sh --status
```

## Configuration Options

### MongoDB Connection String

**Optimized for Self-Hosted**:
```
mongodb://admin:password@mongodb:27017/ethereum_raw_data?authSource=admin&maxPoolSize=50&minPoolSize=5&maxIdleTimeMS=30000&serverSelectionTimeoutMS=5000&socketTimeoutMS=30000&connectTimeoutMS=10000&heartbeatFrequencyMS=10000&retryWrites=true&retryReads=true
```

**Key Parameters**:
- `maxPoolSize=50`: Maximum connections in pool
- `minPoolSize=5`: Minimum connections maintained
- `maxIdleTimeMS=30000`: Close idle connections after 30s
- `serverSelectionTimeoutMS=5000`: Server selection timeout
- `socketTimeoutMS=30000`: Socket operation timeout
- `connectTimeoutMS=10000`: Initial connection timeout
- `heartbeatFrequencyMS=10000`: Health check frequency
- `retryWrites=true`: Enable write retries
- `retryReads=true`: Enable read retries

### Environment Variables

**Critical Settings**:
```bash
# MongoDB Configuration
MONGO_CONNECT_TIMEOUT=15s
MONGO_MAX_POOL_SIZE=50

# Crawler Configuration (Conservative)
BATCH_SIZE=1
CONCURRENT_WORKERS=1
RETRY_ATTEMPTS=5
RETRY_DELAY=5s

# Health Monitoring
HEALTH_CHECK_INTERVAL=15s
METRICS_ENABLED=true
```

## Monitoring and Alerts

### Health Check Components

1. **MongoDB Connection**: Tests database connectivity and response time
2. **Blockchain Connection**: Verifies Ethereum node connectivity
3. **Messaging Service**: Checks NATS connectivity (if enabled)
4. **Application Health**: Monitors crawler service status

### Automatic Recovery

- **Connection Failures**: Automatic reconnection with exponential backoff
- **Health Check Failures**: Recovery actions after 3 consecutive failures
- **Resource Monitoring**: Alerts on high memory/CPU usage

### Log Monitoring

```bash
# Monitor MongoDB logs
docker logs ethereum-scheduler-mongodb -f

# Monitor application logs
docker logs ethereum-scheduler-app -f

# Monitor system resources
docker stats
```

## Performance Tuning

### VPS Optimization

1. **Memory**: Allocate at least 1GB for MongoDB container
2. **CPU**: Reserve at least 1 core for MongoDB
3. **Disk**: Use SSD storage for better I/O performance
4. **Network**: Ensure stable network connectivity

### MongoDB Optimization

```bash
# Run optimization script
./scripts/optimize-mongodb.sh

# Key optimizations applied:
# - Connection pool settings
# - Write concern configuration
# - Index optimization
# - Cache size tuning
```

## Troubleshooting Common Issues

### Issue 1: Connection Timeouts

**Symptoms**: "server selection timeout" errors
**Solution**: Increase timeout values in connection string

### Issue 2: Connection Pool Exhaustion

**Symptoms**: "connection pool exhausted" errors
**Solution**: Optimize pool size and idle timeout settings

### Issue 3: Memory Issues

**Symptoms**: MongoDB container restarts, OOM errors
**Solution**: Increase container memory limits, optimize cache size

### Issue 4: Network Instability

**Symptoms**: Intermittent connection drops
**Solution**: Enable retry logic, increase heartbeat frequency

## Best Practices

1. **Always use connection pooling** with appropriate min/max sizes
2. **Implement retry logic** for all database operations
3. **Monitor health continuously** with automated alerts
4. **Use conservative batch sizes** for stability
5. **Enable comprehensive logging** for troubleshooting
6. **Test stability regularly** with automated tests
7. **Plan for graceful degradation** during outages

## Support

For additional support:
1. Check application logs for specific error messages
2. Run stability tests to identify connection patterns
3. Monitor MongoDB metrics for performance bottlenecks
4. Review VPS resource usage and network connectivity

## Files Modified

- `docker-compose.scheduler.yml`: Enhanced MongoDB container configuration
- `internal/infrastructure/database/mongodb.go`: Connection management improvements
- `internal/application/service/crawler_service.go`: Health checking and recovery
- `internal/adapters/secondary/*_repository_impl.go`: Retry logic implementation
- `.env.scheduler.production`: Optimized production configuration
- `scripts/`: Monitoring and deployment automation scripts
