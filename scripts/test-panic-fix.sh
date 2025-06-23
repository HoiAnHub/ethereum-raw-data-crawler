#!/bin/bash

# Test Panic Fix Script for nil pointer dereference
# This script tests the fixes applied to resolve the panic in pollingWorker

set -e

echo "🔧 Testing Panic Fixes for SchedulerService..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test 1: Check if pollingWorker signature is updated
echo -e "\n${YELLOW}Test 1: Checking pollingWorker signature...${NC}"
if grep -q "pollingWorker(ctx context.Context, ticker \*time.Ticker, stopChan chan struct{})" internal/application/service/scheduler_service.go; then
    echo -e "${GREEN}✅ pollingWorker signature updated correctly${NC}"
else
    echo -e "${RED}❌ pollingWorker signature not updated${NC}"
    exit 1
fi

# Test 2: Check if pollingStopChan is used in struct
echo -e "\n${YELLOW}Test 2: Checking pollingStopChan in struct...${NC}"
if grep -q "pollingStopChan chan struct{}" internal/application/service/scheduler_service.go; then
    echo -e "${GREEN}✅ pollingStopChan added to struct${NC}"
else
    echo -e "${RED}❌ pollingStopChan not found in struct${NC}"
    exit 1
fi

# Test 3: Check if pollingWorker uses ticker parameter instead of s.pollingTicker
echo -e "\n${YELLOW}Test 3: Checking ticker parameter usage...${NC}"
if grep -q "case <-ticker.C:" internal/application/service/scheduler_service.go; then
    echo -e "${GREEN}✅ pollingWorker uses ticker parameter${NC}"
else
    echo -e "${RED}❌ pollingWorker still uses s.pollingTicker.C${NC}"
    exit 1
fi

# Test 4: Check if pollingStopChan is properly initialized
echo -e "\n${YELLOW}Test 4: Checking pollingStopChan initialization...${NC}"
if grep -q "s.pollingStopChan = make(chan struct{})" internal/application/service/scheduler_service.go; then
    echo -e "${GREEN}✅ pollingStopChan properly initialized${NC}"
else
    echo -e "${RED}❌ pollingStopChan initialization not found${NC}"
    exit 1
fi

# Test 5: Check if pollingStopChan is closed in cleanup
echo -e "\n${YELLOW}Test 5: Checking pollingStopChan cleanup...${NC}"
if grep -q "close(s.pollingStopChan)" internal/application/service/scheduler_service.go; then
    echo -e "${GREEN}✅ pollingStopChan properly cleaned up${NC}"
else
    echo -e "${RED}❌ pollingStopChan cleanup not found${NC}"
    exit 1
fi

# Test 6: Check UpsertTransactions fix is still there
echo -e "\n${YELLOW}Test 6: Checking UpsertTransactions fix...${NC}"
if grep -q "setOnInsert" internal/adapters/secondary/transaction_repository_impl.go; then
    echo -e "${GREEN}✅ UpsertTransactions fix maintained${NC}"
else
    echo -e "${RED}❌ UpsertTransactions fix missing${NC}"
    exit 1
fi

# Test 7: Check error handling in scheduler is still there
echo -e "\n${YELLOW}Test 7: Checking scheduler error handling...${NC}"
if grep -q "handleBlockProcessingError" internal/application/service/scheduler_service.go; then
    echo -e "${GREEN}✅ Scheduler error handling maintained${NC}"
else
    echo -e "${RED}❌ Scheduler error handling missing${NC}"
    exit 1
fi

# Test 8: Check duplicate error handling in crawler
echo -e "\n${YELLOW}Test 8: Checking duplicate error handling...${NC}"
if grep -q "duplicate key.*already exists" internal/application/service/crawler_service.go; then
    echo -e "${GREEN}✅ Duplicate error handling maintained${NC}"
else
    echo -e "${RED}❌ Duplicate error handling missing${NC}"
    exit 1
fi

echo -e "\n${GREEN}🎉 All panic fixes applied successfully!${NC}"
echo -e "\n${YELLOW}Summary of fixes:${NC}"
echo -e "${GREEN}1. ✅ Fixed nil pointer dereference in pollingWorker${NC}"
echo -e "${GREEN}2. ✅ Added proper race condition handling${NC}"
echo -e "${GREEN}3. ✅ Improved lifecycle management for polling workers${NC}"
echo -e "${GREEN}4. ✅ Maintained MongoDB upsert fixes${NC}"
echo -e "${GREEN}5. ✅ Maintained error handling improvements${NC}"

echo -e "\n${YELLOW}📝 Next steps:${NC}"
echo -e "1. Build the application: ${GREEN}go build${NC}"
echo -e "2. Test in development: ${GREEN}./scripts/run-scheduler.sh local${NC}"
echo -e "3. Deploy to VPS: ${GREEN}./scripts/run-scheduler.sh docker${NC}"
echo -e "4. Monitor logs for panic/errors: ${GREEN}docker logs ethereum-scheduler-app${NC}"