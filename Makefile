.PHONY: help build run stop clean logs test docker-build docker-run docker-stop docker-clean dev-json test-logging

# Detect container runtime (prefer podman over docker)
CONTAINER_RUNTIME := $(shell command -v podman 2>/dev/null || command -v docker 2>/dev/null)
ifeq ($(CONTAINER_RUNTIME),)
    CONTAINER_RUNTIME := docker
endif
CONTAINER_NAME := $(notdir $(CONTAINER_RUNTIME))

# Detect compose command
COMPOSE_CMD := $(shell command -v podman-compose 2>/dev/null || command -v docker-compose 2>/dev/null)
ifeq ($(COMPOSE_CMD),)
    COMPOSE_CMD := docker-compose
endif

# Default target
help:
	@echo "GoTodo - Available Commands:"
	@echo ""
	@echo "Container Runtime: $(CONTAINER_NAME)"
	@echo "Compose Command: $(notdir $(COMPOSE_CMD))"
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
	$(CONTAINER_RUNTIME) rmi gotodo:latest 2>/dev/null || true

docker-logs:
	$(COMPOSE_CMD) logs -f

# Production Container
docker-prod:
	@echo "Building and running production container setup..."
	$(CONTAINER_RUNTIME) build -t gotodo:latest .
	$(COMPOSE_CMD) -f docker-compose.prod.yml up -d
	@echo "Production deployment running at http://localhost:80"

# Quick start
quick-start: docker-build docker-run
	@echo ""
	@echo "✅ GoTodo is now running!"
	@echo "🌐 Open http://localhost:8080 in your browser"
	@echo ""
	@echo "View logs: make docker-logs"
	@echo "Stop app:  make docker-stop"

