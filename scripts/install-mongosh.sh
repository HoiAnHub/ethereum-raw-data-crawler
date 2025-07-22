#!/bin/bash

# Install MongoDB Shell (mongosh) Script
# This script installs mongosh on various Linux distributions

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}Installing MongoDB Shell (mongosh)...${NC}"

# Check if mongosh is already installed
if command -v mongosh >/dev/null 2>&1; then
    echo -e "${GREEN}✅ mongosh is already installed${NC}"
    mongosh --version
    exit 0
fi

# Detect OS
if [ -f /etc/os-release ]; then
    . /etc/os-release
    OS=$NAME
    VER=$VERSION_ID
else
    echo -e "${RED}❌ Could not detect OS${NC}"
    exit 1
fi

echo -e "${BLUE}Detected OS: ${YELLOW}$OS $VER${NC}"

# Install based on OS
case $OS in
    "Ubuntu"|"Debian GNU/Linux")
        echo -e "${BLUE}Installing on Ubuntu/Debian...${NC}"

        # Add MongoDB GPG key
        wget -qO - https://www.mongodb.org/static/pgp/server-7.0.asc | sudo apt-key add -

        # Add MongoDB repository
        echo "deb [ arch=amd64,arm64 ] https://repo.mongodb.org/apt/ubuntu jammy/mongodb-org/7.0 multiverse" | sudo tee /etc/apt/sources.list.d/mongodb-org-7.0.list

        # Update package list
        sudo apt-get update

        # Install mongosh
        sudo apt-get install -y mongodb-mongosh

        ;;

    "CentOS Linux"|"Red Hat Enterprise Linux"|"Amazon Linux")
        echo -e "${BLUE}Installing on CentOS/RHEL/Amazon Linux...${NC}"

        # Create MongoDB repository file
        cat << EOF | sudo tee /etc/yum.repos.d/mongodb-org-7.0.repo
[mongodb-org-7.0]
name=MongoDB Repository
baseurl=https://repo.mongodb.org/yum/redhat/\$releasever/mongodb-org/7.0/x86_64/
gpgcheck=1
enabled=1
gpgkey=https://www.mongodb.org/static/pgp/server-7.0.asc
EOF

        # Install mongosh
        sudo yum install -y mongodb-mongosh

        ;;

    *)
        echo -e "${YELLOW}⚠ Unsupported OS: $OS${NC}"
        echo -e "${YELLOW}⚠ Please install mongosh manually:${NC}"
        echo -e "${YELLOW}   https://docs.mongodb.com/mongodb-shell/install/${NC}"
        exit 1
        ;;
esac

# Verify installation
if command -v mongosh >/dev/null 2>&1; then
    echo -e "${GREEN}✅ mongosh installed successfully${NC}"
    mongosh --version
else
    echo -e "${RED}❌ mongosh installation failed${NC}"
    exit 1
fi

echo -e "${GREEN}✅ MongoDB Shell installation completed${NC}"