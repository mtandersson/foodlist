# Container Deployment Guide

This guide explains how to build and run the GoTodo application using containers (Podman or Docker).

> **Note:** The build system automatically prefers Podman over Docker if both are installed. All commands work with either container runtime.

## Quick Start

### Using Make (Recommended - Auto-detects Podman/Docker)

1. **Build and start the application:**

   ```bash
   make docker-build
   make docker-run
   ```

2. **Access the application:**
   Open your browser to `http://localhost:8080`

3. **View logs:**

   ```bash
   make docker-logs
   ```

4. **Stop the application:**
   ```bash
   make docker-stop
   ```

### Using Compose Directly

The system will use `podman-compose` if available, otherwise `docker-compose`:

```bash
# Auto-detected compose command
podman-compose up -d   # or docker-compose up -d
podman-compose logs -f
podman-compose down
```

### Using Container CLI Directly

```bash
# The build script auto-detects podman or docker
./build-docker.sh

# Manual build (replace 'podman' with 'docker' if needed)
podman build -t gotodo:latest .

# Run the container
podman run -d \
  --name gotodo \
  -p 8080:8080 \
  -v $(pwd)/data:/app/data \
  gotodo:latest
```

## Container Runtime Detection

The build system automatically detects and prefers Podman over Docker:

1. Checks if `podman` is available → uses Podman
2. Falls back to `docker` if Podman is not found
3. Uses `podman-compose` or `docker-compose` accordingly

All Makefile targets and scripts handle this detection automatically.

## Architecture

The container image is built using a multi-stage build process:

1. **Stage 1 - Frontend Builder:**

   - Uses `node:20-alpine`
   - Installs npm dependencies
   - Builds the Svelte/TypeScript frontend
   - Output: Static files in `dist/`

2. **Stage 2 - Backend Builder:**

   - Uses `golang:1.21-alpine`
   - Downloads Go dependencies
   - Compiles the Go backend
   - Output: Single binary `gotodo`

3. **Stage 3 - Runtime:**
   - Uses `alpine:latest` (minimal footprint)
   - Copies the Go binary
   - Copies the frontend static files
   - Runs the application

## Features

- ✅ Single container serving both frontend and backend
- ✅ WebSocket support for real-time updates
- ✅ Event store persistence via Docker volumes
- ✅ Small image size (~20MB)
- ✅ Health checks included
- ✅ No external dependencies required

## Configuration

### Environment Variables

| Variable     | Default              | Description                       |
| ------------ | -------------------- | --------------------------------- |
| `PORT`       | `8080`               | HTTP server port                  |
| `DATA_DIR`   | `/app/data`          | Directory for `events.jsonl` file |
| `STATIC_DIR` | `/app/frontend/dist` | Frontend static files directory   |

### Volumes

- `/app/data` - Persist the event store (`events.jsonl`)

**Example with custom configuration:**

```bash
docker run -d \
  --name gotodo \
  -p 3000:3000 \
  -e PORT=3000 \
  -e DATA_DIR=/app/data \
  -v $(pwd)/data:/app/data \
  gotodo:latest
```

## Data Persistence

The event store file (`events.jsonl`) is stored in `/app/data` inside the container. To persist data across container restarts:

### Docker Compose (Already configured)

```yaml
volumes:
  - ./data:/app/data
```

### Docker CLI

```bash
docker run -v $(pwd)/data:/app/data ...
```

This mounts a local `./data` directory to the container's `/app/data`, ensuring your todos are saved even when the container is removed.

## Development

### Rebuild after code changes

**Using Make (Recommended):**

```bash
make docker-build
make docker-stop
make docker-run
```

**Using Compose:**

```bash
# Auto-detects podman-compose or docker-compose
podman-compose build  # or docker-compose build
podman-compose up -d
```

**Using Container CLI:**

```bash
# Auto-detects podman or docker
./build-docker.sh
podman stop gotodo && podman rm gotodo
podman run -d --name gotodo -p 8080:8080 -v $(pwd)/data:/app/data gotodo:latest
```

### View real-time logs

```bash
# Using Make
make docker-logs

# Or directly (auto-detected)
podman-compose logs -f gotodo
```

## Deployment

### Deploy to Production

1. **Build the image (auto-detects podman/docker):**

   ```bash
   ./build-docker.sh
   # or: make docker-build
   ```

2. **Tag for your registry:**

   ```bash
   # Replace CONTAINER with 'podman' or 'docker'
   CONTAINER=podman  # or docker
   $CONTAINER tag gotodo:latest your-registry.com/gotodo:v1.0.0
   ```

3. **Push to registry:**

   ```bash
   $CONTAINER push your-registry.com/gotodo:v1.0.0
   ```

4. **Deploy on server:**
   ```bash
   # On production server (use podman or docker)
   $CONTAINER pull your-registry.com/gotodo:v1.0.0
   $CONTAINER run -d \
     --name gotodo \
     --restart unless-stopped \
     -p 8080:8080 \
     -v /var/lib/gotodo/data:/app/data \
     your-registry.com/gotodo:v1.0.0
   ```

### Compose Production

The `docker-compose.prod.yml` file is already configured. Deploy with:

```bash
# Using Make (auto-detects)
make docker-prod

# Or directly
podman-compose -f docker-compose.prod.yml up -d
# or
docker-compose -f docker-compose.prod.yml up -d
```

## Troubleshooting

**Note:** Replace `podman` with `docker` in commands below if using Docker.

### Container won't start

```bash
podman logs gotodo
# or: make docker-logs
```

### Check if container is running

```bash
podman ps
```

### Inspect container

```bash
podman inspect gotodo
```

### Access container shell

```bash
podman exec -it gotodo sh
```

### Check event store file

```bash
podman exec gotodo cat /app/data/events.jsonl
```

### Health check status

```bash
podman inspect --format='{{json .State.Health}}' gotodo | jq
```

## Image Size

The final image is approximately **20-25MB** thanks to:

- Multi-stage build (build artifacts not included)
- Alpine Linux base (~5MB)
- Static Go binary (~10MB)
- Frontend static files (~5MB)

## Security

- Runs as non-root user (Alpine default)
- No unnecessary packages installed
- CA certificates included for HTTPS
- No exposed secrets or credentials

## Backup

**Note:** Replace `podman` with `docker` if using Docker.

### Backup event store

```bash
podman cp gotodo:/app/data/events.jsonl ./backup-$(date +%Y%m%d).jsonl
```

### Restore event store

```bash
podman cp ./backup.jsonl gotodo:/app/data/events.jsonl
podman restart gotodo
```

## Network

### Custom network

```bash
# Replace CONTAINER with 'podman' or 'docker'
CONTAINER=podman
$CONTAINER network create gotodo-network
$CONTAINER run -d \
  --name gotodo \
  --network gotodo-network \
  -p 8080:8080 \
  -v $(pwd)/data:/app/data \
  gotodo:latest
```

### Behind reverse proxy (Nginx/Traefik)

Example Nginx configuration:

```nginx
server {
    listen 80;
    server_name todo.example.com;

    location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Build and Push Container Image

on:
  push:
    tags:
      - "v*"

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Build container image
        run: |
          # Auto-detects podman or docker
          ./build-docker.sh

      - name: Push to registry
        run: |
          # Use docker in CI (pre-installed on GitHub runners)
          echo "${{ secrets.DOCKER_PASSWORD }}" | docker login -u "${{ secrets.DOCKER_USERNAME }}" --password-stdin
          docker tag gotodo:latest gotodo:${{ github.ref_name }}
          docker push gotodo:${{ github.ref_name }}
```

## Support

For issues or questions:

1. Check the logs: `make docker-logs`
2. Verify data directory permissions
3. Ensure port 8080 is available
4. Check container runtime is running (`podman info` or `docker info`)

---

**Built with:** Go + Svelte + WebSockets + Event Sourcing
