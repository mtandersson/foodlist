# Structured Logging Implementation Summary

## Overview

Implemented structured logging for the GoTodo backend using Go's standard library `log/slog` package with support for both **logfmt** (default) and **JSON** formats.

## Changes Made

### 1. Backend Code Updates

#### `backend/main.go`
- Replaced `log` package with `log/slog`
- Added `setupLogger()` function that configures the logger based on `LOG_FORMAT` environment variable
- Updated all logging calls to use structured logging with key-value pairs
- Changed `log.Fatalf()` calls to `slog.Error()` followed by `os.Exit(1)` for better structured output

#### `backend/server.go`
- Replaced `log` package with `log/slog`
- Updated all logging calls throughout the server to use structured logging
- Added contextual fields to logs (e.g., `total_clients`, `event_count`, `command_type`, `event_type`)
- Fixed minor bug: used `event.EventType()` instead of non-existent `event.GetType()` method

### 2. Configuration

#### Environment Variable
- `LOG_FORMAT`: Controls log output format
  - `logfmt` (default): Human-readable key=value format
  - `json`: Machine-parseable JSON format
  - Any other value: Defaults to logfmt with a warning

#### Docker Configuration
- Updated `docker-compose.yml` with example `LOG_FORMAT=logfmt` setting
- Updated `docker-compose.prod.yml` with `LOG_FORMAT=json` (recommended for production)

### 3. Documentation

#### `README.md` (new file)
- Comprehensive project documentation
- Detailed logging configuration section with examples
- Environment variables documentation
- Architecture overview
- Development and testing instructions

#### `test-logging.sh` (new file)
- Automated test script to demonstrate both logging formats
- Tests default, explicit logfmt, JSON, and invalid format handling
- Provides clear visual output showing the differences between formats

## Log Format Examples

### LogFmt (Default)
```
time=2025-12-25T07:34:47.003+01:00 level=INFO msg="logger configured" format=logfmt
time=2025-12-25T07:34:47.006+01:00 level=INFO msg="initializing event store" file=/path/to/events.jsonl
time=2025-12-25T07:34:47.007+01:00 level=INFO msg="loaded events from store" event_count=40
time=2025-12-25T07:34:47.008+01:00 level=INFO msg="starting server" port=8080 websocket_endpoint=ws://localhost:8080/ws
time=2025-12-25T07:34:47.121+01:00 level=INFO msg="client connected" total_clients=1
time=2025-12-25T07:34:48.456+01:00 level=INFO msg="command received" type=CreateTodo message={"type":"CreateTodo",...}
```

### JSON
```json
{"time":"2025-12-25T07:34:47.003+01:00","level":"INFO","msg":"logger configured","format":"json"}
{"time":"2025-12-25T07:34:47.006+01:00","level":"INFO","msg":"initializing event store","file":"/path/to/events.jsonl"}
{"time":"2025-12-25T07:34:47.007+01:00","level":"INFO","msg":"loaded events from store","event_count":40}
{"time":"2025-12-25T07:34:47.008+01:00","level":"INFO","msg":"starting server","port":"8080","websocket_endpoint":"ws://localhost:8080/ws"}
{"time":"2025-12-25T07:34:47.121+01:00","level":"INFO","msg":"client connected","total_clients":1}
{"time":"2025-12-25T07:34:48.456+01:00","level":"INFO","msg":"command received","type":"CreateTodo","message":"{\"type\":\"CreateTodo\",...}"}
```

## Structured Log Fields

### Common Fields (all logs)
- `time`: ISO 8601 timestamp with timezone
- `level`: Log level (INFO, WARN, ERROR)
- `msg`: Human-readable message

### Context-Specific Fields

#### Startup
- `format`: Configured log format
- `file`: Event store file path
- `event_count`: Number of events loaded
- `port`: Server port
- `websocket_endpoint`: WebSocket URL
- `static_dir`: Static files directory

#### Runtime
- `total_clients`: Number of connected WebSocket clients
- `type`: Command type received
- `message`: Full command JSON
- `command_type`: Command type for errors
- `event_type`: Event type for persistence/broadcast
- `error`: Error message for failures
- `requested_format`: For invalid LOG_FORMAT values

## Benefits

1. **Machine Parseable**: JSON format integrates easily with log aggregation systems
2. **Structured Data**: Key-value pairs make filtering and searching easier
3. **Consistent Format**: All logs follow the same structure
4. **Better Context**: Additional fields provide more information about operations
5. **Production Ready**: JSON format ideal for log analysis tools (ELK, Grafana, etc.)
6. **Development Friendly**: LogFmt format is human-readable for local development

## Testing

Tested with:
1. Default configuration (logfmt)
2. Explicit logfmt configuration
3. JSON configuration
4. Invalid format (properly defaults to logfmt with warning)

All logging outputs are working correctly. The automated test script (`test-logging.sh`) demonstrates all formats.

## Notes

- The implementation uses Go's standard library `log/slog` (Go 1.21+)
- No external dependencies were added
- All existing functionality remains unchanged
- Log output goes to stdout (standard practice for containerized applications)
- The structured logging makes it clear when invalid commands are received (improved observability)

## Pre-existing Test Issues

Some unit tests in `server_test.go` send raw events instead of commands to the WebSocket. The structured logging now makes this more visible with "unknown command received" warnings. These tests should be updated to send proper commands (like `CreateTodoCommand`) instead of events (like `TodoCreated`). Tests that correctly send commands (e.g., `TestServer_SetListTitle`) pass successfully.

