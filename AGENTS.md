# FoodList — Agent Quick Reference

Real-time event-sourced grocery list. Go backend + Svelte 5 frontend. WebSocket sync. JSONL event store.

## Development Commands

| Task | Command |
|------|---------|
| Full build (frontend + backend) | `make build` |
| Live-reload dev (localhost, **no security**) | `make dev` |
| Live-reload dev (network access, **no security**) | `make dev-network` |
| Live-reload dev (JSON logging) | `make dev-json` |
| Secure dev (production build, **requires** `backend/.env`) | `make dev-secure` |
| Secure dev + network access | `make dev-secure-net` |
| Run all tests | `make test` |
| Fast unit tests only (parser, no server/websocket) | `make test-unit` |
| Backend lint | `make lint` |
| Frontend typecheck | `cd frontend && npm run check` |

- `make dev` / `make dev-network` **explicitly disable** `SHARED_SECRET` and `CIDR_WHITELIST` by setting them to empty strings. Do not assume security is active in dev.
- `make dev-secure` and `make dev-secure-net` will **exit immediately** if `backend/.env` is missing.
- Live reload requires `air`: `go install github.com/air-verse/air@latest`

## Running Single Tests / Focused Verification

**Backend:**
```bash
cd backend && go test -v -run TestParseIngredientInput -timeout=5s .
```

**Frontend:**
```bash
cd frontend && npm run test:run -- src/lib/quantityParser.test.ts
```

**Required order before finishing work:**
1. `make lint` (backend)
2. `cd frontend && npm run check` (frontend typecheck: runs `svelte-check --tsconfig ./tsconfig.app.json && tsc -p tsconfig.node.json`)
3. `make test`

## Monorepo Boundaries

| Directory | Technology | Entrypoint | Notes |
|-----------|------------|------------|-------|
| `backend/` | Go 1.26.2 | `main.go` | Module name: `foodlist`. Air config: `.air.toml`. Build script: `build-dev.sh` |
| `frontend/` | Svelte 5 + Vite + TypeScript | `src/main.ts` | Strict TS. Uses Svelte 5 runes (`$state`, `$derived`, `$effect`). No separate lint script; rely on `npm run check` |
| `schema/` | JSON Schema | `events.schema.json` | Source of truth for event types. Codegen via `schema/generate.sh` produces `backend/events_gen.go` and frontend types |

## Architecture & Toolchain Quirks

- **Event sourcing:** All state is an in-memory projection from append-only JSONL (`backend/events.jsonl` or `DATA_DIR/events.jsonl`). There is no database.
- **WebSocket endpoint:** `/ws`. Auto-reconnecting client in `frontend/src/lib/websocket.ts`.
- **MCP server:** Always exposed at `/mcp`. **Not** behind `SHARED_SECRET` or `CIDR_WHITELIST`. Protect at the network/reverse-proxy layer if exposed.
- **Secure path routing:** When `SHARED_SECRET` is set, the app is served from `/<secret>/` (e.g., `/dev/`). Root `/` returns 404. `make dev-secure` uses `/dev/`.
- **Container runtime:** Makefile auto-detects `podman` > `docker`. Compose command auto-detected as `podman-compose` > `docker-compose` > `<runtime> compose`.
- **Version injection:** `make build` reads `VERSION` file and injects via `-ldflags "-X main.version=..."`. CI does the same for release binaries.

## Environment Variables

Backend reads from `backend/.env` (auto-loaded by `godotenv`) and env vars:

| Var | Default | Notes |
|-----|---------|-------|
| `PORT` | `8080` | |
| `BIND_ADDR` | `localhost` | Use `0.0.0.0` for network access |
| `DATA_DIR` | `.` | Where `events.jsonl` lives |
| `STATIC_DIR` | `../frontend/dist` | |
| `LOG_FORMAT` | `logfmt` | Also accepts `json` |
| `SHARED_SECRET` | — | Secret path component (optional) |
| `CIDR_WHITELIST` | — | Comma-separated CIDR blocks (optional) |

## Style & Lint

**Backend (`backend/.golangci.yml`):**
- Formatters: `gofmt`, `goimports`, `gofumpt` (module-path: `foodlist`)
- Linters enabled: `govet`, `staticcheck`, `gocritic`, `revive`
- Linters disabled: `errcheck`, `godox`
- Run: `make lint`

**Frontend:**
- Svelte 5 runes required. Avoid `any`; use `unknown` + type guards.
- Component files: PascalCase (`TodoItem.svelte`)
- Store logic: `src/lib/store.ts`, WebSocket client: `src/lib/websocket.ts`

## Testing Quirks

- Backend tests in CI run with `-race` and `-timeout=30s`. Default `make test` does **not** use `-race`.
- Frontend tests use Vitest + jsdom + `@testing-library/svelte`.
- `make test-unit` is the fast path: only parser/quantity extraction tests, no server startup.
- Backend test files are co-located: `*_test.go` in same package.

## CI / Release

- **Branches:** `main` (releases), `develop` (CI on push)
- **Commits:** Conventional Commits mandatory. Types: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`, `revert`.
- **Version bump rules (`.releaserc.cjs`):** `feat` → minor, `fix`/`perf`/`refactor`/`revert`/`chore` → patch, `style(ui)` → patch, breaking → major. `docs`/`style`/`test`/`build`/`ci` normally do **not** release (except `style(ui)`).
- **Release flow:** Push to `main` → run tests → semantic-release → update `VERSION` and `CHANGELOG.md` → build cross-platform binaries → push Docker image to `ghcr.io/OWNER/foodlist`.
- **Commitlint + Husky:** Configured in root (`commitlint.config.js`, `.huskyrc.json`). Validates commit messages locally if husky is installed.

## Environment: NixOS

This project runs on NixOS. Key implications:
- System packages managed via `nix`. Use `nix-shell -p <pkg>` for ad-hoc tools.
- For screenshot tooling and NixOS browser quirks, see `~/.cursor/rules/nixos-screenshots.mdc`.
- Backend must be backgrounded with `nohup sh -c '...' &` since it blocks on stdin.

## Taking Screenshots

See `.cursor/rules/screenshots.mdc` for the full workflow. Key points:
- FoodList requires a running backend (WebSocket SPA). Seed `events.jsonl`, start server on a temp port, then screenshot via HTTP.
- Capture both desktop (1280×800) and mobile iOS (375×667) in **light mode**.
- Wait at least 5 seconds for WebSocket to connect before capturing.
- Always back up and restore `events.jsonl` after screenshots.

## Creating PRs

See `.cursor/rules/pr-creation.mdc` for the full workflow. Key points:
- Always run lint, typecheck, and tests before creating a PR.
- Include before/after screenshots in the PR body for visual changes.
- Commit messages must follow Conventional Commits.
- Target branch is `main`.

## Existing Instruction Sources

- `CONTRIBUTING.md` — Full conventional commits guide and examples.
- `backend/CONFIG.md` — Backend env var reference and MCP protocol notes.
- `.cursor/rules/base.mdc` — Cursor IDE rules (workflow, lint/test requirements, commit format).
- `.cursor/rules/go_style.mdc` — Go-specific conventions.
- `.cursor/rules/typescript_style.mdc` — TypeScript/Svelte-specific conventions.
- `.cursor/rules/screenshots.mdc` — How to take headless screenshots for FoodList.
- `.cursor/rules/pr-creation.mdc` — FoodList PR creation workflow.
- `~/.cursor/rules/nixos-screenshots.mdc` — NixOS-specific browser/tooling quirks (user-level).
- `~/.cursor/rules/pr-creation.mdc` — General PR workflow with gh CLI (user-level).
