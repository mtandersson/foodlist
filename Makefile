.PHONY: help build run stop clean logs test docker-build docker-run docker-stop docker-clean dev-json dev-network test-logging

# Detect container runtime (prefer podman over docker)
CONTAINER_RUNTIME := $(shell command -v podman 2>/dev/null || command -v docker 2>/dev/null)
ifeq ($(CONTAINER_RUNTIME),)
    CONTAINER_RUNTIME := docker
endif
CONTAINER_NAME := $(notdir $(CONTAINER_RUNTIME))

# Detect compose command (prefer podman-compose, then docker-compose, then native compose)
COMPOSE_CMD := $(shell command -v podman-compose 2>/dev/null || command -v docker-compose 2>/dev/null || echo "$(CONTAINER_RUNTIME) compose")
COMPOSE_NAME := $(notdir $(COMPOSE_CMD))

# Default target
help:
	@echo "FoodList - Available Commands:"
	@echo ""
	@echo "Container Runtime: $(CONTAINER_NAME)"
	@echo "Compose Command: $(COMPOSE_NAME)"
	@echo ""
	@echo "Development:"
	@echo "  make build          - Build frontend and backend"
	@echo "  make run            - Run development servers"
	@echo "  make dev            - Run with live reload (logfmt logging)"
	@echo "  make dev-json       - Run with live reload (JSON logging)"
	@echo "  make dev-network    - Run with live reload, accessible from network (for phone testing)"
	@echo "  make test           - Run backend and frontend unit tests"
	@echo "  make test-e2e       - Run E2E tests with isolated test server"
	@echo "  make test-all       - Run all tests (unit + e2e)"
	@echo "  make test-logging   - Test structured logging formats"
	@echo "  make lint           - Run Go linters"
	@echo "  make clean          - Clean build artifacts"
	@echo ""
	@echo "Container:"
	@echo "  make docker-build   - Build container image"
	@echo "  make docker-run     - Run with compose"
	@echo "  make docker-stop    - Stop containers"
	@echo "  make docker-clean   - Remove containers and images"
	@echo "  make docker-logs    - View container logs"
	@echo ""

# Development targets
build:
	@echo "Building frontend..."
	cd frontend && npm install && npm run build
	@echo "Building backend..."
	cd backend && go build -o foodlist

run:
	@echo "Starting development servers..."
	@echo "Backend: http://localhost:8080"
	@echo "Frontend: http://localhost:5173"
	cd backend && ./foodlist &
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

dev-network:
	@command -v air >/dev/null || { echo "❌ 'air' not installed. Install with: go install github.com/air-verse/air@latest"; exit 1; }
	@echo "Starting frontend (Vite dev server with network access)..."
	cd frontend && npm install
	cd frontend && npm run dev -- --host 0.0.0.0 --port 5173 &
	@echo ""
	@echo "Starting backend with live-reload (air) - network accessible..."
	@echo ""
	@echo "🌐 Network Access Enabled:"
	@echo "   Frontend: http://$(shell ipconfig getifaddr en0 || hostname -I | awk '{print $$1}'):5173"
	@echo "   Backend:  http://$(shell ipconfig getifaddr en0 || hostname -I | awk '{print $$1}'):8080"
	@echo ""
	@echo "📱 Use the frontend URL above to access from your phone"
	@echo ""
	@(sleep 3 && open http://localhost:5173) &
	cd backend && BIND_ADDR=0.0.0.0 air

test-logging:
	@echo "Testing structured logging formats..."
	./test-logging.sh

lint:
	@echo "Running Go linters (golangci-lint)..."
	@cd backend && golangci-lint run ./...

test:
	@echo "Running backend tests..."
	cd backend && go test -v -race -cover ./...
	@echo "Running frontend tests..."
	cd frontend && npm test

test-e2e:
	@echo "Running E2E tests with isolated test server..."
	cd e2e && npm run test

test-all: test test-e2e
	@echo "All tests passed!"

clean:
	@echo "Cleaning build artifacts..."
	rm -rf frontend/dist
	rm -rf frontend/node_modules
	rm -f backend/foodlist
	rm -f backend/events.jsonl
	rm -f events.jsonl

# Container targets
docker-build:
	@echo "Building container image using $(CONTAINER_NAME)..."
	./build-docker.sh

docker-run:
	@echo "Starting with $(notdir $(COMPOSE_CMD))..."
	$(COMPOSE_CMD) up -d
	@echo "Application running at http://localhost:8080"

docker-stop:
	@echo "Stopping containers..."
	$(COMPOSE_CMD) down

docker-clean:
	@echo "Removing containers and images..."
	$(COMPOSE_CMD) down -v
	$(CONTAINER_RUNTIME) rmi foodlist:latest 2>/dev/null || true

docker-logs:
	$(COMPOSE_CMD) logs -f

# Production Container
docker-prod:
	@echo "Building and running production container setup..."
	$(CONTAINER_RUNTIME) build -t foodlist:latest .
	$(COMPOSE_CMD) -f docker-compose.prod.yml up -d
	@echo "Production deployment running at http://localhost:80"

# Quick start
quick-start: docker-build docker-run
	@echo ""
	@echo "✅ FoodList is now running!"
	@echo "🌐 Open http://localhost:8080 in your browser"
	@echo ""
	@echo "View logs: make docker-logs"
	@echo "Stop app:  make docker-stop"

