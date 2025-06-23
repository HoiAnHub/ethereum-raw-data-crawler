// MongoDB initialization script for Ethereum Raw Data Crawler
// This script sets up the database and collections with proper indexes

// Switch to the ethereum_raw_data database
db = db.getSiblingDB('ethereum_raw_data');

// Create collections with proper indexes
print('Creating collections and indexes...');

// Blocks collection
db.createCollection('blocks');
db.blocks.createIndex({ "number": 1 }, { unique: true });
db.blocks.createIndex({ "hash": 1 }, { unique: true });
db.blocks.createIndex({ "timestamp": 1 });
db.blocks.createIndex({ "miner": 1 });

// Transactions collection
db.createCollection('transactions');
db.transactions.createIndex({ "hash": 1 }, { unique: true });
db.transactions.createIndex({ "blockNumber": 1 });
db.transactions.createIndex({ "from": 1 });
db.transactions.createIndex({ "to": 1 });
db.transactions.createIndex({ "blockNumber": 1, "transactionIndex": 1 });

// Transaction receipts collection
db.createCollection('transaction_receipts');
db.transaction_receipts.createIndex({ "transactionHash": 1 }, { unique: true });
db.transaction_receipts.createIndex({ "blockNumber": 1 });
db.transaction_receipts.createIndex({ "contractAddress": 1 });

// Logs collection
db.createCollection('logs');
db.logs.createIndex({ "transactionHash": 1 });
db.logs.createIndex({ "blockNumber": 1 });
db.logs.createIndex({ "address": 1 });
db.logs.createIndex({ "topics.0": 1 });

// Crawler state collection
db.createCollection('crawler_state');
db.crawler_state.createIndex({ "key": 1 }, { unique: true });

// Initialize crawler state
db.crawler_state.insertOne({
    key: "last_processed_block",
    value: 0,
    updatedAt: new Date()
});

print('Database initialization completed successfully!');
