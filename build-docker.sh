#!/bin/bash

# GoTodo Docker Build Script

set -e

echo "🐳 Building GoTodo Docker Image..."
echo ""

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "❌ Error: Docker is not running"
    exit 1
fi

# Build the image
echo "📦 Building multi-stage Docker image..."
docker build -t gotodo:latest .

echo ""
echo "✅ Build complete!"
echo ""
echo "📊 Image details:"
docker images gotodo:latest

echo ""
echo "🚀 To run the application:"
echo "   docker-compose up -d"
echo ""
echo "Or using Docker directly:"
echo "   docker run -d --name gotodo -p 8080:8080 -v \$(pwd)/data:/app/data gotodo:latest"
echo ""
echo "🌐 Access the app at: http://localhost:8080"

