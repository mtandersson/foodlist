#!/bin/bash
# Build script for air that injects version from VERSION file

set -e

# Read version from VERSION file (relative to project root)
VERSION_FILE="../VERSION"
if [ -f "$VERSION_FILE" ]; then
  VERSION=$(cat "$VERSION_FILE" | tr -d '\n' | tr -d ' ')
  if [ -n "$VERSION" ]; then
    # Append -dev suffix for local dev builds
    if [ -z "$CI" ] && [ -z "$GITHUB_ACTIONS" ]; then
      VERSION="${VERSION}-dev"
    fi
    go build -ldflags "-X main.version=$VERSION" -o ./tmp/main .
    exit 0
  fi
fi

# Fallback: build without version (will use default "dev")
go build -o ./tmp/main .

