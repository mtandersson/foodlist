# Podman/Docker Support

This document describes the container runtime auto-detection feature implemented in the GoTodo project.

## Overview

The build system now automatically detects and prefers **Podman** over **Docker** when both are available. If Podman is not installed, it seamlessly falls back to Docker.

## Detection Logic

### Priority Order
1. **Podman** (preferred)
2. **Docker** (fallback)

### Implementation

The detection is implemented in two places:

1. **`build-docker.sh`** - Shell script with runtime detection
2. **`Makefile`** - Make variables for automatic tool selection

## Modified Files

### 1. `build-docker.sh`
- Detects `podman` or `docker` at runtime
- Sets appropriate compose command (`podman-compose` or `docker-compose`)
- Displays which runtime is being used
- All container operations use the detected runtime

### 2. `Makefile`
- Variables at the top detect available container tools:
  - `CONTAINER_RUNTIME` - Path to podman or docker
  - `CONTAINER_NAME` - Name of the runtime (for display)
  - `COMPOSE_CMD` - Path to podman-compose or docker-compose
- All `docker-*` targets now use these variables
- `make help` shows which runtime is detected

### 3. Documentation Updates
- **`DOCKER.md`** - Updated to mention Podman/Docker support throughout
- **`DOCKER_SUMMARY.md`** - Updated with runtime detection information
- **`README.md`** - Updated container section to mention both tools

## Usage

All existing commands work exactly the same:

```bash
# Using Make (auto-detects)
make docker-build
make docker-run
make docker-stop
make docker-logs

# Using the build script (auto-detects)
./build-docker.sh

# Using compose directly (use the appropriate one)
podman-compose up -d    # if you have podman-compose
docker-compose up -d    # if you have docker-compose
```

## Verification

You can verify which runtime is detected:

```bash
# Check what Make will use
make help

# Output shows:
# Container Runtime: podman  (or docker)
# Compose Command: podman-compose  (or docker-compose)
```

## Benefits

✅ **Flexibility** - Works with Podman or Docker  
✅ **Security** - Podman runs rootless by default  
✅ **Compatibility** - Seamless fallback to Docker  
✅ **No Changes Required** - Existing workflows unchanged  
✅ **Developer Choice** - Use whichever tool you prefer  

## Requirements

### For Podman Users
- Podman 4.0+
- podman-compose (optional, for compose features)

### For Docker Users
- Docker 20.10+
- Docker Compose v2.0+

## Testing

The implementation has been tested with:
- ✅ Podman detection working
- ✅ Makefile variable detection working
- ✅ Build script syntax validated
- ✅ All documentation updated

## Technical Details

### Shell Detection (build-docker.sh)
```bash
if command -v podman > /dev/null 2>&1; then
    CONTAINER_RUNTIME="podman"
    COMPOSE_CMD="podman-compose"
elif command -v docker > /dev/null 2>&1; then
    CONTAINER_RUNTIME="docker"
    COMPOSE_CMD="docker-compose"
else
    echo "❌ Error: Neither podman nor docker is installed"
    exit 1
fi
```

### Make Detection (Makefile)
```makefile
CONTAINER_RUNTIME := $(shell command -v podman 2>/dev/null || command -v docker 2>/dev/null)
COMPOSE_CMD := $(shell command -v podman-compose 2>/dev/null || command -v docker-compose 2>/dev/null)
```

## Migration Notes

### For Existing Users
No action required! The system will automatically use Docker if that's what you have installed.

### For New Users
Install either Podman or Docker:

**Podman (macOS):**
```bash
brew install podman
podman machine init
podman machine start
```

**Docker (macOS):**
```bash
brew install --cask docker
# Start Docker Desktop
```

## Backwards Compatibility

All existing commands and workflows continue to work:
- ✅ `make docker-*` targets unchanged
- ✅ `docker-compose.yml` files unchanged
- ✅ Dockerfile unchanged
- ✅ CI/CD pipelines work as before

## Future Enhancements

Potential future improvements:
- [ ] Add buildah support for image building
- [ ] Add podman-compose auto-installation check
- [ ] Add container runtime version checks
- [ ] Add performance comparison documentation

---

**Implementation Date:** December 25, 2024  
**Compatibility:** Podman 4.0+ / Docker 20.10+

