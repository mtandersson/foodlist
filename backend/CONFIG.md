# Backend Configuration

The backend now uses environment variables for configuration, managed by:
- `godotenv` - Automatically loads `.env` files
- `github.com/caarlos0/env/v11` - Parses environment variables into a config struct

## Configuration Options

All configuration is done via environment variables. The backend will automatically load a `.env` file from the `backend/` directory if it exists.

### Available Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `BIND_ADDR` | `localhost` | IP address to bind the server to |
| `PORT` | `8080` | Port to listen on |
| `STATIC_DIR` | `../frontend/dist` | Directory containing static frontend files |
| `DATA_DIR` | `.` | Directory where `events.jsonl` will be stored |
| `LOG_FORMAT` | `logfmt` | Log format: `logfmt` (human-readable) or `json` (structured) |

MCP (Model Context Protocol) streamable HTTP is always served at **`/mcp`** when the backend runs. It is not behind `SHARED_SECRET`; with `CIDR_WHITELIST` set, `/mcp` is still reachable for whitelisted clients (same idea as public PWA assets). Protect access at the network or reverse-proxy layer if the server is exposed.

### MCP: resources vs tools (protocol)

- **Resources** are read with JSON-RPC method **`resources/read`** and a **`uri`** (e.g. `foodlist://categories`). Discover URIs with **`resources/list`**. There is no standard **`resources/call`** in MCP; invoking by name is done with **`tools/call`** (parameter **`name`**).
- **Tools** are listed with **`tools/list`** and invoked with **`tools/call`** — this is what the MCP spec defines (plural `tools/...`, not `tool/list`).
- To fetch every defined category (including unused ones), use the **`foodlist_categories`** tool or **`resources/read`** with `uri: "foodlist://categories"`. **`foodlist_list`** only reflects categories that appear on todos in that markdown view.

## Usage

### With .env file (recommended for local development)

1. Copy the example file:
   ```bash
   cp env.example .env
   ```

2. Edit `.env` with your desired configuration

3. Run the backend:
   ```bash
   go run .
   ```

### With environment variables directly

```bash
PORT=3000 BIND_ADDR=0.0.0.0 LOG_FORMAT=json go run .
```

### In Docker

Environment variables can be set in `docker-compose.yml` or passed via `-e` flag:

```bash
docker run -e PORT=3000 -e BIND_ADDR=0.0.0.0 foodlist
```

## Example .env file

See `env.example` for a complete example configuration file.

