# Stage 1: Build Frontend (Svelte/TypeScript)
FROM node:24.15.0-alpine@sha256:d1b3b4da11eefd5941e7f0b9cf17783fc99d9c6fc34884a665f40a06dbdfc94f AS frontend-builder

# Accept VERSION build arg
ARG VERSION

WORKDIR /app/frontend

# Copy frontend package files and npm config (legacy-peer-deps for vite-plugin-pwa vs vite 8)
COPY frontend/package.json frontend/package-lock.json frontend/.npmrc ./

# Install dependencies
RUN npm ci

# Copy frontend source
COPY frontend/ ./

# Build frontend with version injection
RUN if [ -n "$VERSION" ]; then \
      RELEASE_VERSION=$VERSION npm run build; \
    else \
      npm run build; \
    fi

# Stage 2: Build Backend (Go)
FROM golang:1.26.3-alpine@sha256:91eda9776261207ea25fd06b5b7fed8d397dd2c0a283e77f2ab6e91bfa71079d AS backend-builder

# Accept VERSION build arg
ARG VERSION

WORKDIR /app/backend

# Copy go mod files
COPY backend/go.mod backend/go.sum ./

# Download dependencies
RUN go mod download

# Copy backend source
COPY backend/ ./

# Build the Go application with version injection
RUN if [ -n "$VERSION" ]; then \
      CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags "-X main.version=$VERSION" -o foodlist .; \
    else \
      CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o foodlist .; \
    fi

# Create data directory structure for distroless (no shell available)
RUN mkdir -p /app/data

# Stage 3: Final Runtime Image
FROM gcr.io/distroless/static-debian13:nonroot@sha256:e3f945647ffb95b5839c07038d64f9811adf17308b9121d8a2b87b6a22a80a39

WORKDIR /app

# Copy the built Go binary from backend-builder
COPY --from=backend-builder /app/backend/foodlist .

# Copy the built frontend from frontend-builder
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist

# Copy data directory structure from backend-builder (distroless has no shell for RUN commands)
COPY --chown=nonroot:nonroot --from=backend-builder /app/data /app/data

# Expose port
EXPOSE 8080

# Set environment variables
ENV DATA_DIR=/app/data
ENV STATIC_DIR=/app/frontend/dist
ENV PORT=8080

# Run the application
CMD ["./foodlist"]

