# Stage 1: Build Frontend (Svelte/TypeScript)
FROM node:24.13.0-alpine@sha256:931d7d57f8c1fd0e2179dbff7cc7da4c9dd100998bc2b32afc85142d8efbc213 AS frontend-builder

# Accept VERSION build arg
ARG VERSION

WORKDIR /app/frontend

# Copy frontend package files
COPY frontend/package*.json ./

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
FROM golang:1.25.6-alpine@sha256:98e6cffc31ccc44c7c15d83df1d69891efee8115a5bb7ede2bf30a38af3e3c92 AS backend-builder

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
FROM gcr.io/distroless/static-debian13:nonroot

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

