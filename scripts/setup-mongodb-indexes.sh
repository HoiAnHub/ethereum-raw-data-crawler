#!/bin/bash

# Setup MongoDB Indexes Script
# This script creates necessary indexes for the external MongoDB

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}Setting up MongoDB indexes...${NC}"

# Check if .env file exists
if [ ! -f .env ]; then
    echo -e "${YELLOW}⚠ .env file not found. Copy from env.example first${NC}"
    exit 1
fi

# Load environment variables
source .env

# Check if MONGO_URI is set
if [ -z "$MONGO_URI" ]; then
    echo -e "${RED}❌ MONGO_URI is not set in .env file${NC}"
    exit 1
fi

DATABASE="${MONGO_DATABASE:-ethereum_raw_data}"

echo -e "${BLUE}Database: ${YELLOW}$DATABASE${NC}"

# Check if mongosh is available
if ! command -v mongosh >/dev/null 2>&1; then
    echo -e "${RED}❌ mongosh is required but not installed${NC}"
    echo -e "${YELLOW}Please install MongoDB Shell: https://docs.mongodb.com/mongodb-shell/install/${NC}"
    exit 1
fi

# Create indexes
echo -e "${BLUE}Creating indexes...${NC}"

# Blocks collection indexes
echo -e "${BLUE}Creating blocks collection indexes...${NC}"
mongosh "$MONGO_URI" --eval "
db = db.getSiblingDB('$DATABASE');

// Blocks collection indexes
db.blocks.createIndex({number: 1}, {unique: true});
db.blocks.createIndex({hash: 1}, {unique: true});
db.blocks.createIndex({network: 1, number: 1});
db.blocks.createIndex({timestamp: 1});
db.blocks.createIndex({status: 1});

print('✅ Blocks indexes created');
" --quiet

# Transactions collection indexes
echo -e "${BLUE}Creating transactions collection indexes...${NC}"
mongosh "$MONGO_URI" --eval "
db = db.getSiblingDB('$DATABASE');

// Transactions collection indexes
db.transactions.createIndex({hash: 1}, {unique: true});
db.transactions.createIndex({block_hash: 1});
db.transactions.createIndex({block_number: 1});
db.transactions.createIndex({from: 1});
db.transactions.createIndex({to: 1});
db.transactions.createIndex({network: 1, block_number: 1});
db.transactions.createIndex({tx_status: 1});

print('✅ Transactions indexes created');
" --quiet

# Crawler metrics collection indexes
echo -e "${BLUE}Creating crawler_metrics collection indexes...${NC}"
mongosh "$MONGO_URI" --eval "
db = db.getSiblingDB('$DATABASE');

// Crawler metrics collection indexes
db.crawler_metrics.createIndex({timestamp: 1});
db.crawler_metrics.createIndex({network: 1, timestamp: 1});

print('✅ Crawler metrics indexes created');
" --quiet

# System health collection indexes
echo -e "${BLUE}Creating system_health collection indexes...${NC}"
mongosh "$MONGO_URI" --eval "
db = db.getSiblingDB('$DATABASE');

// System health collection indexes
db.system_health.createIndex({timestamp: 1});
db.system_health.createIndex({network: 1, timestamp: 1});
db.system_health.createIndex({status: 1});

print('✅ System health indexes created');
" --quiet

echo -e "${GREEN}✅ All MongoDB indexes created successfully${NC}"

# Show index information
echo -e "${BLUE}Index information:${NC}"
mongosh "$MONGO_URI" --eval "
db = db.getSiblingDB('$DATABASE');

print('\\n📊 Blocks indexes:');
db.blocks.getIndexes().forEach(function(index) {
    print('  - ' + index.name + ': ' + JSON.stringify(index.key));
});

print('\\n📊 Transactions indexes:');
db.transactions.getIndexes().forEach(function(index) {
    print('  - ' + index.name + ': ' + JSON.stringify(index.key));
});

print('\\n📊 Crawler metrics indexes:');
db.crawler_metrics.getIndexes().forEach(function(index) {
    print('  - ' + index.name + ': ' + JSON.stringify(index.key));
});

print('\\n📊 System health indexes:');
db.system_health.getIndexes().forEach(function(index) {
    print('  - ' + index.name + ': ' + JSON.stringify(index.key));
});
" --quiet