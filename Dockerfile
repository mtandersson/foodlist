# syntax=docker/dockerfile:1.27@sha256:bde3983e9c939224420ddaf6b784cc30e09b035a4dea01f581230c50809f372e
# The syntax directive enables BuildKit `--mount=type=cache` so the Go module
# cache and Go build cache (which contains compiled cgo C objects from
# goheif's vendored libde265/dav1d) survive across image builds.

# Stage 1: Build Frontend (Svelte/TypeScript)
FROM node:24.20.0-alpine@sha256:e67514e5d0f6c46656005e1b693b2ec9d52e80b641307de684d4a015ba7a4eaf AS frontend-builder

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
FROM golang:1.27.1-alpine@sha256:cf6fca6641884b8433441b2b0652976f975e1d0fdd26d177eaaf8596087f3125 AS backend-builder

# Accept VERSION build arg
ARG VERSION

WORKDIR /app/backend

# goheif (HEIC/HEIF transcoding) requires cgo for its bundled libde265/dav1d
# decoders, so we install a C toolchain and statically link against musl so the
# resulting binary still runs on distroless/static.
RUN apk add --no-cache gcc g++ musl-dev

# Copy go mod files
COPY backend/go.mod backend/go.sum ./

# Download dependencies. The module cache is mounted so unchanged go.sum
# entries are not re-downloaded between builds.
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    go mod download

# Copy backend source
COPY backend/ ./

# Build the Go application with version injection. CGO is enabled but the
# binary is statically linked (osusergo/netgo + -extldflags -static) so it has
# no runtime libc dependency and remains compatible with distroless/static.
#
# The Go build cache (/root/.cache/go-build) is mounted so that goheif's
# vendored libde265/dav1d C sources are not recompiled on every build when
# only Go code in this repo changes. Dropping `-a` lets the build cache
# actually be used; without it Go ignores cached artifacts and rebuilds
# everything from scratch (which is what made each Docker build slow).
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    if [ -n "$VERSION" ]; then \
      CGO_ENABLED=1 GOOS=linux go build -tags "osusergo netgo" -ldflags "-w -s -X main.version=$VERSION -extldflags '-static'" -o foodlist .; \
    else \
      CGO_ENABLED=1 GOOS=linux go build -tags "osusergo netgo" -ldflags "-w -s -extldflags '-static'" -o foodlist .; \
    fi

# Create data directory structure for distroless (no shell available)
RUN mkdir -p /app/data

# Stage 3: Final Runtime Image
FROM gcr.io/distroless/static-debian13:nonroot@sha256:1c2c046bc09ed40fad370b599a0b1ae7987f55b01e247cf27a7c27cd97e5bbc7

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

