# Backend Configuration

The backend now uses environment variables for configuration, managed by:

- `godotenv` - Automatically loads `.env` files
- `github.com/caarlos0/env/v11` - Parses environment variables into a config struct

## Configuration Options

All configuration is done via environment variables. The backend will automatically load a `.env` file from the `backend/` directory if it exists.

### Available Environment Variables

| Variable     | Default            | Description                                                  |
| ------------ | ------------------ | ------------------------------------------------------------ |
| `BIND_ADDR`  | `localhost`        | IP address to bind the server to                             |
| `PORT`       | `8080`             | Port to listen on                                            |
| `STATIC_DIR` | `../frontend/dist` | Directory containing static frontend files                   |
| `DATA_DIR`   | `.`                | Directory where `events.jsonl` will be stored                |
| `LOG_FORMAT` | `logfmt`           | Log format: `logfmt` (human-readable) or `json` (structured) |

### Embeddings + auto-categorize

The server can suggest a category for new todos based on cosine similarity
between their embeddings and the embeddings of existing categorized items.
The feature is only active when `GEMINI_API_KEY` is set.

| Variable                                    | Default                  | Description                                                                                  |
| ------------------------------------------- | ------------------------ | -------------------------------------------------------------------------------------------- |
| `GEMINI_API_KEY`                            | _empty_                  | Google AI Studio API key. When empty, the embedding cache build is skipped and auto-categorize is disabled. |
| `EMBEDDING_MODEL`                           | `gemini-embedding-001`   | Gemini embedding model name (without `models/` prefix).                                      |
| `EMBEDDING_CACHE_FILE`                      | `<DATA_DIR>/embeddings.jsonl` | Path to the JSONL cache file. Validated to be inside `DATA_DIR`.                        |
| `EMBEDDING_BATCH_SIZE`                      | `100`                    | Texts per batchEmbedContents call (max 100).                                                 |
| `EMBEDDING_RPM`                             | `60`                     | Shared outbound request rate (startup batch + runtime hook share one bucket).                |
| `EMBEDDING_CATEGORIZE_ENABLED`              | `true`                   | Master switch. Set to `false` to disable auto-categorize even with an API key set.           |
| `EMBEDDING_CATEGORIZE_SIMILARITY_FLOOR`     | `0.55`                   | Per-item similarity floor; lower values contribute zero score.                               |
| `EMBEDDING_CATEGORIZE_RECENCY_WINDOW_DAYS`  | `30`                     | Recent-pass window in days.                                                                  |
| `EMBEDDING_CATEGORIZE_RECENT_WEIGHT`        | `0.70`                   | Blend factor: `combined = recentWeight*recent + (1-recentWeight)*all`.                       |
| `EMBEDDING_CATEGORIZE_POPULARITY_WEIGHT`    | `0.30`                   | Sublinear category-size bias `1 + weight*log(1+N)`. Set to `0` to disable.                   |
| `EMBEDDING_CATEGORIZE_MAX_SIM_GATE`         | `0.20`                   | Minimum floored similarity any single item must hit before its category is eligible.         |
| `EMBEDDING_CATEGORIZE_ACCEPTANCE_THRESHOLD` | `0.30`                   | Minimum final score required to emit a `TodoCategorized` suggestion.                         |

Operational counters are exposed at `GET /api/v1/auto-categorize/metrics`
(bearer-auth, returns 404 when the feature is disabled). See
[`backend/auto_categorize_metrics.go`](auto_categorize_metrics.go) for the
JSON schema.

### Suggestions (Förslag) tab

The server can build a list of items the user is likely to buy soon, based
on completed-todo history clustered by embedding similarity. This feature
**requires `GEMINI_API_KEY`** (and the embedding cache it enables) — without
it the engine is not initialized and the frontend hides the tab.

| Variable                          | Default | Description                                                                                          |
| --------------------------------- | ------- | ---------------------------------------------------------------------------------------------------- |
| `SUGGESTIONS_ENABLED`             | `true`  | Master switch. Set to `false` to disable the feature even when `GEMINI_API_KEY` is set.              |
| `SUGGESTIONS_MIN_PURCHASES`       | `3`     | Minimum number of historical purchases required before an item becomes eligible for suggestion.      |
| `SUGGESTIONS_MAX_INTERVAL_DAYS`   | `90`    | Items with an average purchase interval longer than this are considered too rare and skipped.        |
| `SUGGESTIONS_DUE_FRACTION`        | `0.667` | An item is suggested only when (now − lastPurchased) / avgInterval ≥ this fraction.                  |
| `SUGGESTIONS_DEDUP_SIMILARITY`    | `0.85`  | Cosine-similarity threshold for treating two strings as the same product (e.g. `mjölk` vs `2l mjölk`). |
| `SUGGESTIONS_RECENT_LIMIT`        | `6`     | Number of most-recent purchases used when computing the average purchase interval.                   |
| `SUGGESTIONS_RECOMPUTE_HOURS`     | `6`     | Background recompute cadence in hours. The engine also recomputes on relevant todo/category events.  |

The list is pushed to clients over WebSocket: a full `SuggestionsRollup`
on connect, then `SuggestionAdded` / `SuggestionRemoved` deltas. Both the
MCP tool `foodlist_suggestions` and the resource `foodlist://suggestions`
expose the same view to AI agents.

MCP (Model Context Protocol) streamable HTTP is always served at **`/mcp`** when the backend runs. It is not behind `SHARED_SECRET`; with `CIDR_WHITELIST` set, `/mcp` is still reachable for whitelisted clients (same idea as public PWA assets). Protect access at the network or reverse-proxy layer if the server is exposed.

### MCP: resources vs tools (protocol)

- **Resources** are read with JSON-RPC method **`resources/read`** and a **`uri`** (e.g. `foodlist://categories`). Discover URIs with **`resources/list`**. There is no standard **`resources/call`** in MCP; invoking by name is done with **`tools/call`** (parameter **`name`**).
- **Tools** are listed with **`tools/list`** and invoked with **`tools/call`** — this is what the MCP spec defines (plural `tools/...`, not `tool/list`).
- MCP protocol identifiers keep legacy `todo` naming for compatibility (for example `foodlist://todos` and `todo_id`), but they refer to grocery items.
- To fetch every defined category (including unused ones), use the **`foodlist_categories`** tool or **`resources/read`** with `uri: "foodlist://categories"`. **`foodlist_list`** only reflects categories that appear on grocery items in that markdown view.

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
