#!/bin/bash

# GoTodo Container Build Script
# Supports both Podman and Docker

set -e

# Detect container runtime (prefer podman over docker)
if command -v podman > /dev/null 2>&1; then
    CONTAINER_RUNTIME="podman"
    echo "🦭 Using Podman for container builds..."
elif command -v docker > /dev/null 2>&1; then
    CONTAINER_RUNTIME="docker"
    echo "🐳 Using Docker for container builds..."
else
    echo "❌ Error: Neither podman nor docker is installed"
    exit 1
fi

# Detect compose command (prefer podman-compose, fallback to docker-compose)
if command -v podman-compose > /dev/null 2>&1; then
    COMPOSE_CMD="podman-compose"
elif command -v docker-compose > /dev/null 2>&1; then
    COMPOSE_CMD="docker-compose"
else
    COMPOSE_CMD="$CONTAINER_RUNTIME compose"
    echo "ℹ️  Note: Using '$COMPOSE_CMD' (standalone compose tools not found)"
fi

echo ""

# Check if container runtime is running/available
if ! $CONTAINER_RUNTIME info > /dev/null 2>&1; then
    echo "❌ Error: $CONTAINER_RUNTIME is not running or not accessible"
    exit 1
fi

# Build the image
echo "📦 Building multi-stage container image..."
$CONTAINER_RUNTIME build -t gotodo:latest .

echo ""
echo "✅ Build complete!"
echo ""
echo "📊 Image details:"
$CONTAINER_RUNTIME images gotodo:latest

echo ""
echo "🚀 To run the application:"
echo "   $COMPOSE_CMD up -d"
echo ""
echo "Or using $CONTAINER_RUNTIME directly:"
echo "   $CONTAINER_RUNTIME run -d --name gotodo -p 8080:8080 -v \$(pwd)/data:/app/data gotodo:latest"
echo ""
echo "🌐 Access the app at: http://localhost:8080"

