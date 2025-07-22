#!/bin/bash

# Test MongoDB Connection Script
# This script tests the connection to external MongoDB

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}Testing MongoDB Connection...${NC}"

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

echo -e "${BLUE}MongoDB URI: ${YELLOW}$MONGO_URI${NC}"
echo -e "${BLUE}Database: ${YELLOW}${MONGO_DATABASE:-ethereum_raw_data}${NC}"

# Test connection using mongosh (if available)
if command -v mongosh >/dev/null 2>&1; then
    echo -e "${BLUE}Testing connection with mongosh...${NC}"

    # Extract connection string without database name for testing
    CONNECTION_STRING=$(echo "$MONGO_URI" | sed 's|/[^/?]*\(?.*\)\?$||')

    if mongosh "$CONNECTION_STRING" --eval "db.runCommand('ping')" --quiet >/dev/null 2>&1; then
        echo -e "${GREEN}✅ MongoDB connection successful${NC}"

        # Test database access
        if mongosh "$MONGO_URI" --eval "db.runCommand('ping')" --quiet >/dev/null 2>&1; then
            echo -e "${GREEN}✅ Database access successful${NC}"
        else
            echo -e "${RED}❌ Database access failed${NC}"
            exit 1
        fi
    else
        echo -e "${RED}❌ MongoDB connection failed${NC}"
        exit 1
    fi
else
    echo -e "${YELLOW}⚠ mongosh not found, testing with curl...${NC}"

        # Extract host and port from URI for basic connectivity test
    if [[ "$MONGO_URI" =~ mongodb://([^@]+)@([^:/]+):?([0-9]*)/ ]]; then
        # Format: mongodb://username:password@host:port/database
        HOST="${BASH_REMATCH[2]}"
        PORT="${BASH_REMATCH[3]:-27017}"
    elif [[ "$MONGO_URI" =~ mongodb://([^:/]+):?([0-9]*)/ ]]; then
        # Format: mongodb://host:port/database (no auth)
        HOST="${BASH_REMATCH[1]}"
        PORT="${BASH_REMATCH[2]:-27017}"
    elif [[ "$MONGO_URI" =~ mongodb\+srv://([^@]+)@([^/]+)/ ]]; then
        # Format: mongodb+srv://username:password@cluster.mongodb.net/database
        HOST="${BASH_REMATCH[2]}"
        PORT="27017"
    else
        echo -e "${YELLOW}⚠ Could not parse MongoDB URI for connectivity test${NC}"
        echo -e "${YELLOW}⚠ Please install mongosh for full connection testing${NC}"
        exit 1
    fi

    echo -e "${BLUE}Testing connectivity to ${HOST}:${PORT}...${NC}"

    if timeout 5 bash -c "</dev/tcp/$HOST/$PORT" 2>/dev/null; then
        echo -e "${GREEN}✅ Network connectivity successful${NC}"
    else
        echo -e "${RED}❌ Network connectivity failed${NC}"
        echo -e "${YELLOW}⚠ This might be due to:${NC}"
        echo -e "${YELLOW}   - Firewall blocking port ${PORT}${NC}"
        echo -e "${YELLOW}   - MongoDB server not running${NC}"
        echo -e "${YELLOW}   - Network connectivity issues${NC}"
        exit 1
    fi
fi

echo -e "${GREEN}✅ MongoDB connection test completed${NC}"