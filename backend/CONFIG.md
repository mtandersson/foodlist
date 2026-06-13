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

### Recipes (Recept) tab

The recipes feature lets users upload photos of paper recipes, run them
through a vision LLM, review/edit the parsed result, and save them. Saved
recipes can later be browsed, sent into the shopping list ingredient by
ingredient, or stepped through in a shared "cook mode".

**Security boundary: `SHARED_SECRET` + `CIDR_WHITELIST`.** Recipe HTTP
routes (`GET`, `POST`, `PATCH`, `DELETE` under `/api/v1/recipes/...`)
carry NO application-level auth — no bearer token, no cookie session,
no per-route check. They sit on the same mux as the homepage,
WebSocket, and static assets, and are protected by exactly the same
mechanism the rest of the app already relies on:

- `SHARED_SECRET` mounts every route under a non-guessable URL prefix
  (`/<secret>/...`). Without the secret you cannot even guess the
  recipes path.
- `CIDR_WHITELIST` runs `IPWhitelistMiddleware` over the entire mux,
  rejecting any caller whose source IP is outside the allowed
  networks before the request reaches a handler.

Together those two are the security perimeter — a leaked URL alone
cannot reach the recipes API from off-network, and an on-network
attacker cannot reach it without the secret prefix. No further
per-route auth is added, by design.

`FOODLIST_API_TOKEN` is **not** consulted for recipe routes; it only
applies to the legacy `/api/v1/state` and `/api/v1/command` JSON
endpoints kept for AppleScript/HTTP integrations.

| Variable                   | Default                | Description                                                                                |
| -------------------------- | ---------------------- | ------------------------------------------------------------------------------------------ |
| `RECIPE_DIR`               | `<DATA_DIR>/recipes`   | Storage directory. Validated to live inside `DATA_DIR`.                                    |
| `RECIPE_LLM_BASE_URL`      | _empty_                | OpenAI-compatible base URL (e.g. `https://api.opencode.ai/v1`). Only http(s).              |
| `RECIPE_LLM_API_KEY`       | _empty_                | Bearer token sent to the LLM. **Never logged**, even on error paths.                       |
| `RECIPE_LLM_MODEL`         | _empty_                | Vision-capable model name. The frontend `recipesParse` flag goes true only with all three. |
| `RECIPE_PARSE_RPM`         | `10`                   | Per-process rate limit on `POST /recipes/parse`. 429 on overflow.                          |
| `RECIPE_MAX_IMAGE_PIXELS`  | `24000000`             | Decompression-bomb cap; uploads decoded to more pixels are rejected.                       |

**Security properties**:

- **Path traversal**: server-generated UUIDs for every recipe ID; route
  IDs are UUID-validated before any filesystem call; `filepath.Abs` +
  prefix check against `RECIPE_DIR`; sidecar extension comes from a
  sniffed-MIME allowlist (`image/jpeg→.jpg`, `image/png→.png`,
  `image/webp→.webp`), never from the client filename.
- **SSRF**: the LLM base URL is read from env at startup and is never
  user-supplied. Operators must avoid pointing it at private/loopback
  addresses that host other internal services.
- **Image safety**: 10 MB transport cap (`MaxBytesReader`), MIME sniffed
  via `http.DetectContentType` against the allowlist plus an explicit
  ISO-BMFF `ftyp` brand check for HEIC, pixel cap enforced via
  `image.DecodeConfig`. iPhone HEIC/HEIF uploads are transcoded to
  JPEG server-side via `goheif` before the bounds check, the LLM call,
  or the sidecar write — only browser-renderable mimes (JPEG/PNG/WebP)
  are ever stored. Sidecar served with the stored MIME plus
  `X-Content-Type-Options: nosniff` and `Content-Disposition: inline`.
- **Logging**: API key, image bytes, request body, and LLM response body
  are never logged. Only model name, image size, and HTTP status are
  emitted (success or failure).
- **Stored XSS**: recipe ingredient/instruction/section text is
  rendered via Svelte text bindings only. The recipe `description`
  field is the only `{@html}` site in the recipe views; it routes
  through `frontend/src/lib/markdown.ts`, which parses with `marked`
  (gfm + breaks, no mangle/headerIds extensions) and sanitizes with
  `DOMPurify` using a narrow tag allowlist (no `<script>`, `<img>`,
  `<iframe>`, `<style>`, `<object>`, `<svg>`, no `on*` attributes).
  Anchor `href` values are validated through `URL` parsing and must
  begin with `http://`, `https://`, or `mailto:`; user-info, control
  characters, and protocol-relative URLs are stripped. Links are
  forced to `target="_blank" rel="noopener noreferrer"`. Tested in
  `frontend/src/lib/markdown.test.ts`.
- **Content-Security-Policy**: `SecurityHeadersMiddleware` in
  `backend/middleware.go` sets a baseline policy on every response:
  `default-src 'self'`, `script-src 'self'` (no inline, no eval),
  `object-src 'none'`, `frame-ancestors 'none'`, `base-uri 'self'`,
  plus `X-Content-Type-Options: nosniff` and
  `Referrer-Policy: strict-origin-when-cross-origin`. Even if the
  marked + DOMPurify pipeline ever regressed and emitted an inline
  `<script>` tag, the browser would refuse to execute it.
- **MCP untrusted-content banner**: `foodlist_recipe_get` prefixes
  every response with an explicit warning that the title,
  description, and steps are user-supplied and must not be treated
  as agent instructions. This is the prompt-injection mitigation
  documented in the recipe-sections security review.

**Cook mode** (the shared "check off each step" workflow) is purely
WebSocket and ephemeral: `Cook*` commands are dispatched by a dedicated
handler that does NOT touch the event store. Cook session state survives
multi-client edits but is dropped on server restart and on recipe DELETE,
and is pruned when a recipe's total step count (summed across all
sections, via `recipeTotalSteps`) shrinks below a previously-checked
index after a PATCH. New clients receive a `CookStateRollup` snapshot
on connect so they sync up with whatever any other tab has already
checked off. Cook step indices are flat: section 0 contributes 0..N-1,
section 1 picks up at N, etc. The same flat indexing is exposed to
the MCP `foodlist_recipe_add_ingredients` tool as a 1-based global
ingredient index across sections.

MCP (Model Context Protocol) streamable HTTP is served at **`/mcp`** and, when `SHARED_SECRET` is configured, also at **`/<secret>/mcp`**. With `CIDR_WHITELIST` active, `/mcp` is reachable only by whitelisted IPs (same posture as `/ws`), while `/<secret>/mcp` remains accessible to any client that knows the shared secret. Protect access at the network or reverse-proxy layer if the server is exposed.

### MCP: resources vs tools (protocol)

- **Resources** are read with JSON-RPC method **`resources/read`** and a **`uri`** (e.g. `foodlist://categories`). Discover URIs with **`resources/list`**. There is no standard **`resources/call`** in MCP; invoking by name is done with **`tools/call`** (parameter **`name`**).
- **Tools** are listed with **`tools/list`** and invoked with **`tools/call`** — this is what the MCP spec defines (plural `tools/...`, not `tool/list`).
- MCP protocol identifiers keep legacy `todo` naming for compatibility (for example `foodlist://todos` and `todo_id`), but they refer to grocery items.
- To fetch every defined category (including unused ones), use the **`foodlist_categories`** tool or **`resources/read`** with `uri: "foodlist://categories"`. **`foodlist_list`** only reflects categories that appear on grocery items in that markdown view.

### MCP: recipes

The recipes MCP surface is read-mostly and mirrors the HTTP API. It is
mounted automatically; when the recipes feature is disabled the tools
return graceful "_disabled_" responses and the resource returns an empty
JSON array (so an MCP client can still introspect the server).

- **Tool `foodlist_recipes_list`** — markdown list of saved recipes
  (title + id), newest first.
- **Tool `foodlist_recipe_get`** — markdown view of a single recipe
  (title, optional description, one heading per section, ingredients
  with optional amount/unit, globally numbered instructions). Output
  always begins with an untrusted-content banner so the calling agent
  does not treat embedded text as instructions. Argument: `recipe_id`
  (UUID).
- **Tool `foodlist_recipe_add_ingredients`** — adds a recipe's ingredients
  to the shopping list as todo items via the same `CreateTodo` event the
  UI emits. Arguments: `recipe_id`, optional `indexes` (**1-based and
  global across sections**; empty means "all"), optional `category_id`.
  When an ingredient row carries both `amount` and `unit`, the
  structured-input precedence kicks in so
  the server skips `ParseIngredientInput` and trusts those values.
- **Resource `foodlist://recipes`** — JSON array of recipe metadata
  (id, title, image filename, timestamps).

Image uploads, the LLM parse path, and `PATCH`/`DELETE` are intentionally
HTTP-only because they are either binary, costly, or destructive in ways
that don't fit the JSON-RPC tool surface.

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
