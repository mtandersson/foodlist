#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Starting E2E Test Setup${NC}"

# Create temporary directory for test data
TEST_DATA_DIR=$(mktemp -d)
TEST_DB_FILE="$TEST_DATA_DIR/events-test.jsonl"
BACKEND_PID_FILE="$TEST_DATA_DIR/backend.pid"
BACKEND_LOG_FILE="$TEST_DATA_DIR/backend.log"

echo -e "${YELLOW}Test data directory: $TEST_DATA_DIR${NC}"
echo -e "${YELLOW}Test database: $TEST_DB_FILE${NC}"

# Cleanup function
cleanup() {
  echo -e "\n${YELLOW}Cleaning up...${NC}"
  
  # Kill backend server if running
  if [ -f "$BACKEND_PID_FILE" ]; then
    BACKEND_PID=$(cat "$BACKEND_PID_FILE")
    if kill -0 "$BACKEND_PID" 2>/dev/null; then
      echo "Stopping backend server (PID: $BACKEND_PID)"
      kill "$BACKEND_PID" || true
      sleep 1
      # Force kill if still running
      kill -9 "$BACKEND_PID" 2>/dev/null || true
    fi
    rm -f "$BACKEND_PID_FILE"
  fi
  
  # Remove temporary directory
  echo "Removing test data directory: $TEST_DATA_DIR"
  rm -rf "$TEST_DATA_DIR"
  
  echo -e "${GREEN}Cleanup complete${NC}"
}

# Set trap to cleanup on exit
trap cleanup EXIT INT TERM

# Check if backend binary exists
if [ ! -f "../backend/foodlist" ]; then
  echo -e "${YELLOW}Backend binary not found. Building...${NC}"
  cd ../backend && go build -o foodlist . && cd ../e2e
  echo -e "${GREEN}Backend built successfully${NC}"
fi

# Check if frontend is built
if [ ! -d "../frontend/dist" ]; then
  echo -e "${YELLOW}Frontend not built. Building...${NC}"
  cd ../frontend && npm ci && npm run build && cd ../e2e
  echo -e "${GREEN}Frontend built successfully${NC}"
fi

# Start backend server with test database
echo -e "${GREEN}Starting backend server for E2E tests${NC}"
cd ../backend
DATA_DIR="$TEST_DATA_DIR" \
  PORT=5174 \
  BIND_ADDR=localhost \
  STATIC_DIR=../frontend/dist \
  LOG_FORMAT=logfmt \
  ./foodlist > "$BACKEND_LOG_FILE" 2>&1 &

BACKEND_PID=$!
echo $BACKEND_PID > "$BACKEND_PID_FILE"
cd ../e2e

echo "Backend server started (PID: $BACKEND_PID)"
echo "Log file: $BACKEND_LOG_FILE"

# Wait for backend to be ready
echo -e "${YELLOW}Waiting for backend server to start...${NC}"
MAX_RETRIES=30
RETRY_COUNT=0

while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
  if curl -f http://localhost:5174 > /dev/null 2>&1; then
    echo -e "${GREEN}Backend server is ready!${NC}"
    break
  fi
  
  if ! kill -0 "$BACKEND_PID" 2>/dev/null; then
    echo -e "${RED}Backend server died unexpectedly!${NC}"
    echo -e "${RED}Backend logs:${NC}"
    cat "$BACKEND_LOG_FILE"
    exit 1
  fi
  
  RETRY_COUNT=$((RETRY_COUNT + 1))
  if [ $RETRY_COUNT -eq $MAX_RETRIES ]; then
    echo -e "${RED}Backend server failed to start within 30 seconds${NC}"
    echo -e "${RED}Backend logs:${NC}"
    cat "$BACKEND_LOG_FILE"
    exit 1
  fi
  
  sleep 1
done

# Run Cypress tests
echo -e "${GREEN}Running Cypress E2E tests${NC}"
export CYPRESS_BASE_URL=http://localhost:5174

if npm run test:ci; then
  echo -e "${GREEN}E2E tests passed!${NC}"
  exit 0
else
  echo -e "${RED}E2E tests failed!${NC}"
  echo -e "${RED}Backend logs:${NC}"
  cat "$BACKEND_LOG_FILE"
  exit 1
fi

