.PHONY: help build run stop clean logs test docker-build docker-run docker-stop docker-clean dev-json test-logging

# Default target
help:
	@echo "GoTodo - Available Commands:"
	@echo ""
	@echo "Development:"
	@echo "  make build          - Build frontend and backend"
	@echo "  make run            - Run development servers"
	@echo "  make dev            - Run with live reload (logfmt logging)"
	@echo "  make dev-json       - Run with live reload (JSON logging)"
	@echo "  make test           - Run all tests"
	@echo "  make test-logging   - Test structured logging formats"
	@echo "  make clean          - Clean build artifacts"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-build   - Build Docker image"
	@echo "  make docker-run     - Run with docker-compose"
	@echo "  make docker-stop    - Stop Docker containers"
	@echo "  make docker-clean   - Remove Docker containers and images"
	@echo "  make docker-logs    - View Docker logs"
	@echo ""

# Development targets
build:
	@echo "Building frontend..."
	cd frontend && npm install && npm run build
	@echo "Building backend..."
	cd backend && go build -o gotodo

run:
	@echo "Starting development servers..."
	@echo "Backend: http://localhost:8080"
	@echo "Frontend: http://localhost:5173"
	cd backend && ./gotodo &
	cd frontend && npm run dev

dev:
	@command -v air >/dev/null || { echo "❌ 'air' not installed. Install with: go install github.com/air-verse/air@latest"; exit 1; }
	@echo "Starting frontend (Vite dev server with auto-reload)..."
	cd frontend && npm install
	cd frontend && npm run dev -- --host --port 5173 &
	@echo "Starting backend with live-reload (air)..."
	@(sleep 3 && open http://localhost:5173) &
	cd backend && air

dev-json:
	@command -v air >/dev/null || { echo "❌ 'air' not installed. Install with: go install github.com/air-verse/air@latest"; exit 1; }
	@echo "Starting frontend (Vite dev server with auto-reload)..."
	cd frontend && npm install
	cd frontend && npm run dev -- --host --port 5173 &
	@echo "Starting backend with live-reload (air) - JSON logging..."
	@(sleep 3 && open http://localhost:5173) &
	cd backend && LOG_FORMAT=json air

test-logging:
	@echo "Testing structured logging formats..."
	./test-logging.sh

lint:
	@echo "Running Go linters (golangci-lint)..."
	@cd backend && golangci-lint run ./...

test:
	@echo "Running backend tests..."
	cd backend && go test -v -cover ./...
	@echo "Running frontend tests..."
	cd frontend && npm test

clean:
	@echo "Cleaning build artifacts..."
	rm -rf frontend/dist
	rm -rf frontend/node_modules
	rm -f backend/gotodo
	rm -f backend/events.jsonl
	rm -f events.jsonl

# Docker targets
docker-build:
	@echo "Building Docker image..."
	./build-docker.sh

docker-run:
	@echo "Starting with Docker Compose..."
	docker-compose up -d
	@echo "Application running at http://localhost:8080"

docker-stop:
	@echo "Stopping Docker containers..."
	docker-compose down

docker-clean:
	@echo "Removing Docker containers and images..."
	docker-compose down -v
	docker rmi gotodo:latest 2>/dev/null || true

docker-logs:
	docker-compose logs -f

# Production Docker
docker-prod:
	@echo "Building and running production Docker setup..."
	docker build -t gotodo:latest .
	docker-compose -f docker-compose.prod.yml up -d
	@echo "Production deployment running at http://localhost:80"

# Quick start
quick-start: docker-build docker-run
	@echo ""
	@echo "✅ GoTodo is now running!"
	@echo "🌐 Open http://localhost:8080 in your browser"
	@echo ""
	@echo "View logs: make docker-logs"
	@echo "Stop app:  make docker-stop"

