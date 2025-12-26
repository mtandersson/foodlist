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

````text
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
````

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

# All tests
make test
```

## CI/CD and Releases

This project uses **Conventional Commits** for automated versioning and releases.

### Commit Message Format

All commit messages must follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

**Common types:**

- `feat`: New feature (minor version bump)
- `fix`: Bug fix (patch version bump)
- `docs`: Documentation changes
- `refactor`: Code refactoring
- `perf`: Performance improvements
- `test`: Test updates
- `ci`: CI/CD changes
- `chore`: Other changes

**Examples:**

```bash
feat(backend): add user authentication
fix(websocket): resolve connection leak
docs(readme): update installation instructions
feat(api)!: change event format (breaking change)
```

See [`CONTRIBUTING.md`](CONTRIBUTING.md) for detailed guidelines.

### Using Cursor IDE

**Cursor AI is configured to suggest semantic commits!**

When committing in Cursor:

1. Stage your changes
2. Click in the commit message field
3. Press **Cmd+K** (Mac) or **Ctrl+K** (Windows/Linux)
4. Cursor will suggest a properly formatted conventional commit

See [`docs/guides/CURSOR_COMMIT_SETUP.md`](docs/guides/CURSOR_COMMIT_SETUP.md) for configuration details.

### Automated Releases

When you push to `main` branch:

1. **CI Pipeline** runs automatically:

   - Backend tests (Go)
   - Frontend tests (Vitest)
   - Linting (golangci-lint)
   - Docker build

2. **Release Pipeline** (after tests pass):
   - Analyzes commit messages since last release
   - Determines next version based on commit types
   - Generates changelog automatically
   - Creates GitHub release with release notes
   - Builds and pushes Docker images to GitHub Container Registry
   - Creates binary artifacts for multiple platforms

### Version Bumping

- **MAJOR** (1.0.0 → 2.0.0): Breaking changes (`feat!:` or `BREAKING CHANGE:`)
- **MINOR** (1.0.0 → 1.1.0): New features (`feat:`)
- **PATCH** (1.0.0 → 1.0.1): Bug fixes and other changes (`fix:`, `docs:`, etc.)

### Docker Images

Docker images are automatically published to GitHub Container Registry:

```bash
# Pull latest version
docker pull ghcr.io/OWNER/foodlist:latest

# Pull specific version
docker pull ghcr.io/OWNER/foodlist:1.2.3
```

Replace `OWNER` with your GitHub username or organization.

### Dependency Updates (Renovate)

Renovate is configured to automatically update dependencies:

- Go modules (backend)
- npm packages (frontend)
- GitHub Actions
- Docker images

**Setup:** See [`docs/setup/RENOVATE_SETUP.md`](docs/setup/RENOVATE_SETUP.md) for configuration options.

**Recommended:** Install the [Renovate GitHub App](https://github.com/apps/renovate) for zero-configuration automated updates.

### Local Commit Validation (Optional)

Install commitlint and husky for local validation:

```bash
npm install --save-dev @commitlint/{config-conventional,cli} husky
npx husky install
npx husky add .husky/commit-msg 'npx --no -- commitlint --edit ${1}'
```

This will validate your commit messages before they're committed.

## Contributing

Please read [`CONTRIBUTING.md`](CONTRIBUTING.md) for details on our conventional commit guidelines and development workflow.

## Documentation

### Core Documentation

- [`CONTRIBUTING.md`](CONTRIBUTING.md) - Conventional commits guide and development workflow
- [`CHANGELOG.md`](CHANGELOG.md) - Version history and release notes

### Setup & Configuration

- [`docs/setup/CI_CD_GUIDE.md`](docs/setup/CI_CD_GUIDE.md) - CI/CD pipeline documentation
- [`docs/setup/DOCKER.md`](docs/setup/DOCKER.md) - Container deployment guide (Podman/Docker)
- [`docs/setup/RENOVATE_SETUP.md`](docs/setup/RENOVATE_SETUP.md) - Dependency update automation

### Development Guides

- [`docs/guides/AI_DEVELOPMENT_GUIDE.md`](docs/guides/AI_DEVELOPMENT_GUIDE.md) - Detailed development instructions
- [`docs/guides/AI_QUICK_GUIDE.md`](docs/guides/AI_QUICK_GUIDE.md) - TDD workflow and architecture guide
- [`docs/guides/CURSOR_QUICK_START.md`](docs/guides/CURSOR_QUICK_START.md) - Getting started with Cursor IDE
- [`docs/guides/COMMIT_QUICK_REFERENCE.md`](docs/guides/COMMIT_QUICK_REFERENCE.md) - Quick commit message reference
- [`docs/guides/TEST_COVERAGE_REPORT.md`](docs/guides/TEST_COVERAGE_REPORT.md) - Test coverage details

### Features

- [`docs/features/LOCAL_STORAGE_PERSISTENCE.md`](docs/features/LOCAL_STORAGE_PERSISTENCE.md) - Offline persistence implementation
- [`docs/features/MOBILE_CATEGORY_SELECTION.md`](docs/features/MOBILE_CATEGORY_SELECTION.md) - Mobile UI enhancements

### Architecture

- [`docs/architecture/STRUCTURED_LOGGING.md`](docs/architecture/STRUCTURED_LOGGING.md) - Logging implementation details
- [`docs/architecture/WEBSOCKET_NETWORK_FIX.md`](docs/architecture/WEBSOCKET_NETWORK_FIX.md) - WebSocket architecture

## License

MIT
