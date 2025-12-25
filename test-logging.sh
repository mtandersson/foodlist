#!/bin/bash
# Test script to demonstrate structured logging formats

set -e

cd "$(dirname "$0")/backend"

echo "======================================"
echo "Testing GoTodo Structured Logging"
echo "======================================"
echo ""

# Build the application
echo "Building application..."
go build -o gotodo_test . || exit 1
echo "✓ Build successful"
echo ""

# Test logfmt format (default)
echo "======================================"
echo "Test 1: Default logfmt format"
echo "======================================"
export DATA_DIR="."
export PORT="8081"
timeout 3s ./gotodo_test 2>&1 || true
echo ""

# Test logfmt format (explicit)
echo "======================================"
echo "Test 2: Explicit logfmt format"
echo "======================================"
export LOG_FORMAT="logfmt"
export PORT="8082"
timeout 3s ./gotodo_test 2>&1 || true
echo ""

# Test JSON format
echo "======================================"
echo "Test 3: JSON format"
echo "======================================"
export LOG_FORMAT="json"
export PORT="8083"
timeout 3s ./gotodo_test 2>&1 || true
echo ""

# Test invalid format (should default to logfmt with warning)
echo "======================================"
echo "Test 4: Invalid format (should warn)"
echo "======================================"
export LOG_FORMAT="invalid"
export PORT="8084"
timeout 3s ./gotodo_test 2>&1 || true
echo ""

# Cleanup
rm -f gotodo_test
echo "======================================"
echo "All tests complete!"
echo "======================================"

