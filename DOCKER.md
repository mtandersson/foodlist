# Docker Deployment Guide

This guide explains how to build and run the GoTodo application using Docker.

## Quick Start

### Using Docker Compose (Recommended)

1. **Build and start the application:**
   ```bash
   docker-compose up -d
   ```

2. **Access the application:**
   Open your browser to `http://localhost:8080`

3. **View logs:**
   ```bash
   docker-compose logs -f
   ```

4. **Stop the application:**
   ```bash
   docker-compose down
   ```

### Using Docker CLI

1. **Build the image:**
   ```bash
   docker build -t gotodo:latest .
   ```

2. **Run the container:**
   ```bash
   docker run -d \
     --name gotodo \
     -p 8080:8080 \
     -v $(pwd)/data:/app/data \
     gotodo:latest
   ```

3. **Access the application:**
   Open your browser to `http://localhost:8080`

## Architecture

The Docker image is built using a multi-stage build process:

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

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `DATA_DIR` | `/app/data` | Directory for `events.jsonl` file |
| `STATIC_DIR` | `/app/frontend/dist` | Frontend static files directory |

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

**Using Docker Compose:**
```bash
docker-compose build
docker-compose up -d
```

**Using Docker CLI:**
```bash
docker build -t gotodo:latest .
docker stop gotodo && docker rm gotodo
docker run -d --name gotodo -p 8080:8080 -v $(pwd)/data:/app/data gotodo:latest
```

### View real-time logs
```bash
docker-compose logs -f gotodo
```

## Deployment

### Deploy to Production

1. **Build the image:**
   ```bash
   docker build -t gotodo:v1.0.0 .
   ```

2. **Tag for your registry:**
   ```bash
   docker tag gotodo:v1.0.0 your-registry.com/gotodo:v1.0.0
   ```

3. **Push to registry:**
   ```bash
   docker push your-registry.com/gotodo:v1.0.0
   ```

4. **Deploy on server:**
   ```bash
   docker pull your-registry.com/gotodo:v1.0.0
   docker run -d \
     --name gotodo \
     --restart unless-stopped \
     -p 8080:8080 \
     -v /var/lib/gotodo/data:/app/data \
     your-registry.com/gotodo:v1.0.0
   ```

### Docker Compose Production

Create `docker-compose.prod.yml`:
```yaml
version: '3.8'

services:
  gotodo:
    image: your-registry.com/gotodo:v1.0.0
    ports:
      - "8080:8080"
    volumes:
      - /var/lib/gotodo/data:/app/data
    environment:
      - DATA_DIR=/app/data
    restart: always
```

Deploy:
```bash
docker-compose -f docker-compose.prod.yml up -d
```

## Troubleshooting

### Container won't start
```bash
docker logs gotodo
```

### Check if container is running
```bash
docker ps
```

### Inspect container
```bash
docker inspect gotodo
```

### Access container shell
```bash
docker exec -it gotodo sh
```

### Check event store file
```bash
docker exec gotodo cat /app/data/events.jsonl
```

### Health check status
```bash
docker inspect --format='{{json .State.Health}}' gotodo | jq
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

### Backup event store
```bash
docker cp gotodo:/app/data/events.jsonl ./backup-$(date +%Y%m%d).jsonl
```

### Restore event store
```bash
docker cp ./backup.jsonl gotodo:/app/data/events.jsonl
docker restart gotodo
```

## Network

### Custom network
```bash
docker network create gotodo-network
docker run -d \
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
name: Build and Push Docker Image

on:
  push:
    tags:
      - 'v*'

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Build Docker image
        run: docker build -t gotodo:${{ github.ref_name }} .
      
      - name: Push to registry
        run: |
          echo "${{ secrets.DOCKER_PASSWORD }}" | docker login -u "${{ secrets.DOCKER_USERNAME }}" --password-stdin
          docker push gotodo:${{ github.ref_name }}
```

## Support

For issues or questions:
1. Check the logs: `docker-compose logs -f`
2. Verify data directory permissions
3. Ensure port 8080 is available
4. Check Docker daemon is running

---

**Built with:** Go + Svelte + WebSockets + Event Sourcing

