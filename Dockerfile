# Stage 1: Build Frontend (Svelte/TypeScript)
FROM node:20-alpine AS frontend-builder

WORKDIR /app/frontend

# Copy frontend package files
COPY frontend/package*.json ./

# Install dependencies
RUN npm ci

# Copy frontend source
COPY frontend/ ./

# Build frontend
RUN npm run build

# Stage 2: Build Backend (Go)
FROM golang:1.25.5-alpine AS backend-builder

WORKDIR /app/backend

# Copy go mod files
COPY backend/go.mod backend/go.sum ./

# Download dependencies
RUN go mod download

# Copy backend source
COPY backend/ ./

# Build the Go application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o foodlist .

# Stage 3: Final Runtime Image
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the built Go binary from backend-builder
COPY --from=backend-builder /app/backend/foodlist .

# Copy the built frontend from frontend-builder
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist

# Create directory for event store
RUN mkdir -p /app/data

# Expose port
EXPOSE 8080

# Set environment variables
ENV DATA_DIR=/app/data
ENV STATIC_DIR=/app/frontend/dist
ENV PORT=8080

# Run the application
CMD ["./foodlist"]

