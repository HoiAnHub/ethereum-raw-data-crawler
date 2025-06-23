# Batch Upsert for Transactions

## Overview

The Ethereum Raw Data Crawler now supports batch upsert operations for transactions, providing better performance and handling of duplicate transactions when processing blocks.

## Features

### 1. Batch Upsert Operations
- Uses MongoDB's `BulkWrite` with upsert operations
- Processes all transactions in a block as a single batch operation
- Handles duplicate transactions gracefully without errors
- Significantly improves performance compared to individual inserts

### 2. Configurable Fallback Mechanism
- Configurable fallback to traditional batch insert if upsert fails
- Provides resilience against potential upsert issues
- Maintains backward compatibility

### 3. Enhanced Logging and Monitoring
- Detailed logging of batch operations with timing information
- Performance metrics for both upsert and fallback operations
- Error tracking for troubleshooting

## Configuration

### Environment Variables

Add these to your `.env` file:

```bash
# Enable batch upsert (default: true)
CRAWLER_USE_UPSERT=true

# Enable fallback to insert if upsert fails (default: true)
CRAWLER_UPSERT_FALLBACK=true
```

### Configuration Options

| Variable | Default | Description |
|----------|---------|-------------|
| `CRAWLER_USE_UPSERT` | `true` | Enable batch upsert instead of batch insert |
| `CRAWLER_UPSERT_FALLBACK` | `true` | Fallback to insert if upsert fails |

## How It Works

### 1. Upsert Mode (Default)
```
Block Processing → Get Transactions → Batch Upsert → Success
                                   ↓ (if fails and fallback enabled)
                                   Batch Insert → Success/Failure
```

### 2. Insert Mode (Legacy)
```
Block Processing → Get Transactions → Batch Insert → Success/Failure
```

## Benefits

### Performance Improvements
- **Reduced Database Round Trips**: Single bulk operation per block instead of individual operations
- **Better Concurrency**: MongoDB handles conflicts internally during upsert
- **Faster Processing**: Eliminates duplicate key error handling at application level

### Reliability Improvements
- **Duplicate Handling**: Gracefully handles re-processing of blocks
- **Fault Tolerance**: Fallback mechanism ensures data is saved even if upsert fails
- **Idempotent Operations**: Safe to re-run block processing without data corruption

## Monitoring

### Log Messages

The system provides detailed logging for batch operations:

```
DEBUG: Attempting batch upsert transaction_count=150
DEBUG: Batch upsert succeeded transaction_count=150 duration=45ms

WARN:  Batch upsert failed transaction_count=150 duration=120ms error="..."
INFO:  Falling back to batch insert transaction_count=150
INFO:  Fallback batch insert succeeded transaction_count=150 fallback_duration=80ms total_duration=200ms
```

### Performance Metrics

Monitor these metrics to track performance:
- **Batch Operation Duration**: Time taken for each batch operation
- **Fallback Rate**: Frequency of fallback to insert operations
- **Transaction Throughput**: Transactions processed per second

## Troubleshooting

### Common Issues

1. **High Fallback Rate**
   - Check MongoDB connection stability
   - Verify index configuration
   - Monitor MongoDB logs for errors

2. **Slow Upsert Performance**
   - Ensure proper indexing on `hash` field
   - Check MongoDB server resources
   - Consider adjusting batch size

3. **Duplicate Key Errors in Fallback**
   - Usually indicates concurrent processing of same block
   - Enable proper coordination between crawler instances
   - Check block processing logic

### Debugging

Enable debug logging to see detailed operation information:

```bash
LOG_LEVEL=debug
```

## Migration

### From Previous Versions

The batch upsert feature is backward compatible. Existing installations will:
1. Automatically use upsert mode with fallback enabled
2. Continue working with existing data
3. Benefit from improved performance immediately

### Disabling Batch Upsert

To revert to traditional insert mode:

```bash
CRAWLER_USE_UPSERT=false
```

## Best Practices

1. **Keep Upsert Enabled**: Provides better performance and duplicate handling
2. **Enable Fallback**: Ensures reliability during edge cases
3. **Monitor Performance**: Track batch operation metrics
4. **Proper Indexing**: Ensure `hash` field is properly indexed for optimal upsert performance
5. **Batch Size Tuning**: Adjust `BATCH_SIZE` based on your system's performance characteristics

## Technical Details

### MongoDB Operations

The upsert operation uses MongoDB's `BulkWrite` with `UpdateOne` operations:

```go
filter := bson.M{"hash": tx.Hash}
update := bson.M{"$set": tx}
upsertOp := mongo.NewUpdateOneModel()
upsertOp.SetFilter(filter)
upsertOp.SetUpdate(update)
upsertOp.SetUpsert(true)
```

### Index Requirements

Ensure the following indexes exist for optimal performance:

```javascript
db.transactions.createIndex({ "hash": 1 }, { unique: true })
db.transactions.createIndex({ "block_number": 1 })
db.transactions.createIndex({ "block_hash": 1 })
```

These indexes are automatically created by the application during startup.
