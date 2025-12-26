# FoodList

A real-time, event-sourced todo list application built with Go, Svelte, and WebSockets.

## Features

- Real-time synchronization across multiple clients
- Event sourcing architecture with JSONL storage
- Optimistic UI updates
- Persistent storage
- Live reload for development
- Structured logging with multiple formats

## Quick Start

### Using Make

```bash
# Build the project
make build

# Run development servers with live reload
make dev

# Run development servers accessible from network (for phone testing)
make dev-network

# Run tests
make test
```

### Using Containers (Podman/Docker)

The build system automatically prefers Podman if installed, with fallback to Docker.

```bash
# Quick start (auto-detects Podman or Docker)
make quick-start

# Or manually
make docker-build
make docker-run
```

## Configuration

### Environment Variables

- `PORT` - HTTP server port (default: `8080`)
- `BIND_ADDR` - Address to bind server to (default: `localhost`, use `0.0.0.0` for network access)
- `DATA_DIR` - Directory for event storage (default: `.`)
- `STATIC_DIR` - Directory for static files (default: `../frontend/dist`)
- `LOG_FORMAT` - Log output format: `logfmt` or `json` (default: `logfmt`)

### Structured Logging

The backend uses Go's `log/slog` package for structured logging. You can configure the log format using the `LOG_FORMAT` environment variable:

**LogFmt format (default):**

```bash
# No environment variable needed, logfmt is the default
./backend/foodlist

# Or explicitly set it
LOG_FORMAT=logfmt ./backend/foodlist
```

Example output:

```text
time=2025-12-25T07:34:47.003+01:00 level=INFO msg="logger configured" format=logfmt
time=2025-12-25T07:34:47.006+01:00 level=INFO msg="initializing event store" file=/path/to/events.jsonl
time=2025-12-25T07:34:47.007+01:00 level=INFO msg="loaded events from store" event_count=40
time=2025-12-25T07:34:47.008+01:00 level=INFO msg="starting server" port=8080 websocket_endpoint=ws://localhost:8080/ws
time=2025-12-25T07:34:47.121+01:00 level=INFO msg="client connected" total_clients=1
time=2025-12-25T07:34:48.456+01:00 level=INFO msg="command received" type=CreateTodo message={"type":"CreateTodo",...}
```

**JSON format:**

```bash
LOG_FORMAT=json ./backend/foodlist
```

Example output:

```json
{"time":"2025-12-25T07:34:47.003+01:00","level":"INFO","msg":"logger configured","format":"json"}
{"time":"2025-12-25T07:34:47.006+01:00","level":"INFO","msg":"initializing event store","file":"/path/to/events.jsonl"}
{"time":"2025-12-25T07:34:47.007+01:00","level":"INFO","msg":"loaded events from store","event_count":40}
{"time":"2025-12-25T07:34:47.008+01:00","level":"INFO","msg":"starting server","port":"8080","websocket_endpoint":"ws://localhost:8080/ws"}
{"time":"2025-12-25T07:34:47.121+01:00","level":"INFO","msg":"client connected","total_clients":1}
{"time":"2025-12-25T07:34:48.456+01:00","level":"INFO","msg":"command received","type":"CreateTodo","message":"{\"type\":\"CreateTodo\",...}"}
```

JSON format is particularly useful for:

- Log aggregation systems (e.g., ELK stack, Grafana Loki)
- Structured log parsing and analysis
- Cloud logging services (e.g., CloudWatch, Stackdriver)

### Using with Containers

You can set the log format in your compose configuration (`docker-compose.yml`):

```yaml
services:
  foodlist:
    environment:
      - LOG_FORMAT=json # or logfmt
```

## Architecture

### Backend (Go)

- Event Store: JSONL-based append-only log
- State Projection: In-memory state built from events
- WebSocket Server: Real-time communication with clients
- Structured Logging: `log/slog` with logfmt (default) or JSON formats

### Frontend (Svelte + TypeScript)

- WebSocket Client: Auto-reconnecting connection
- State Store: Optimistic updates with reconciliation
- UI Components: Reactive and animated

### Event Sourcing

All state changes are represented as immutable events:

- `TodoCreated`
- `TodoCompleted`
- `TodoUncompleted`
- `TodoStarred`
- `TodoUnstarred`
- `TodoReordered`
- `TodoRenamed`
- `ListTitleChanged`

Events are defined in `schema/events.schema.json` and code is generated for both backend and frontend.

## Development

### Prerequisites

- Go 1.21+
- Node.js 18+
- Air (for live reload): `go install github.com/air-verse/air@latest`

### Development Modes

**Local Development (default):**

```bash
make dev
```

This starts the development servers on `localhost` only, accessible at:
- Frontend: http://localhost:5173
- Backend: http://localhost:8080

**Network Development (for phone/tablet testing):**

```bash
make dev-network
```

This starts the development servers accessible from your local network. The command will display your network IP address. Example output:

```text
🌐 Network Access Enabled:
   Frontend: http://192.168.1.100:5173
   Backend:  http://192.168.1.100:8080

📱 Use the frontend URL above to access from your phone
```text

To test on your phone:
1. Ensure your phone is on the same Wi-Fi network as your computer
2. Run `make dev-network`
3. Open the displayed frontend URL on your phone's browser
4. The app will connect to the backend via WebSocket proxy through the frontend

**JSON Logging Mode:**

```bash
make dev-json
```

Same as `make dev` but with JSON-formatted logs (useful for log analysis tools).

### Project Structure

```text
foodlist/
├── backend/          # Go backend
│   ├── main.go      # Entry point, logger setup
│   ├── server.go    # WebSocket server
│   ├── store.go     # Event store
│   ├── state.go     # State projection
│   └── events_gen.go # Generated event types
├── frontend/        # Svelte frontend
│   └── src/
│       ├── lib/
│       │   ├── store.ts      # State management
│       │   ├── websocket.ts  # WebSocket client
│       │   └── *.svelte      # UI components
│       └── App.svelte
├── e2e/             # Cypress E2E tests
├── schema/          # Event schema definitions
│   ├── events.schema.json
│   └── generate.sh
└── Makefile         # Build and run commands
```

### Testing

```bash
# Backend tests
cd backend && go test -v -cover ./...

# Frontend tests
cd frontend && npm test

# E2E tests
cd e2e && npx cypress run

# All tests
make test
```

## Documentation

- `AI_QUICK_GUIDE.md` - TDD workflow and architecture guide
- `AI_DEVELOPMENT_GUIDE.md` - Detailed development instructions
- `DOCKER.md` - Container deployment guide (Podman/Docker)
- `TEST_COVERAGE_REPORT.md` - Test coverage details

## License

MIT
