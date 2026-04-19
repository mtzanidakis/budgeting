# CLAUDE.md

## Project Overview

Family budgeting PWA — Go backend with embedded vanilla JS frontend, SQLite database.
Multi-user income/expense tracking with categories, charts, and i18n (EN/EL).

## Build & Run

```bash
# CGO is required for server + admin (SQLite3 driver)
make build        # Build server + admin CLI + budgeting-cli to bin/
make run          # Build and run server
make test         # go test -v ./...
make clean        # Remove bin/

# Manual build (version injected via ldflags)
CGO_ENABLED=1 go build -o bin/server ./cmd/server
CGO_ENABLED=1 go build -o bin/admin ./cmd/admin
CGO_ENABLED=0 go build -o bin/budgeting-cli ./cmd/budgeting-cli
```

## Project Structure

```
cmd/server/          Main HTTP server (Chi router), embeds frontend/
cmd/server/frontend/ Static assets: HTML template, CSS, vanilla JS, PWA manifest
cmd/admin/           Admin CLI (user:* and token:* commands, direct DB access)
cmd/budgeting-cli/   HTTP client (no CGO) for scripts/AI agents, uses API tokens
internal/apiclient/  HTTP client library used by budgeting-cli
internal/auth/       Password hashing (bcrypt), session store, API token helpers (SHA-256)
internal/config/     Env var config via caarlos0/env
internal/database/   SQLite3 operations, schema migrations, query builders
internal/handlers/   HTTP handlers (auth, actions, categories, users, tokens, config, static)
internal/middleware/  Auth (session + Bearer) + logging middleware, RequireSessionAuth guard
internal/models/     Data structs (User, Action, Category, APIToken, ActionType)
internal/version/    Build-time version injection
skill/SKILL.md       Claude Code skill bundled with budgeting-cli release archives
```

## Key Conventions

- **Error handling**: `fmt.Errorf("failed to X: %w", err)` — wrap and propagate, never panic
- **Handlers**: Method receivers on `*XyzHandler` structs, constructors `NewXyzHandler(db)`
- **JSON responses**: All via `respondJSON(w, statusCode, data)` helper
- **Database**: Raw SQL with `?` placeholders, no ORM. Dynamic filters via `ActionFilters` struct
- **Migrations**: Idempotent `CREATE TABLE/INDEX IF NOT EXISTS` in `db.Migrate()`, runs on startup
- **Frontend state**: Single global `state` object, `render()` rebuilds the DOM
- **i18n**: Translation keys in `i18n.js`, accessed via `t('key.name')`; two locales: `en`, `el`
- **Dates**: Display as DD/MM/YYYY, API format YYYY-MM-DD
- **Sessions**: In-memory (lost on server restart), HTTP-only cookie, SameSite=Strict
- **API tokens**: `bdg_` prefix, SHA-256 hash stored, soft-deleted on revoke, throttled `last_used_at`
- **Auth middleware**: accepts session cookie OR `Authorization: Bearer bdg_...`. `RequireSessionAuth()` guards token-management endpoints so tokens cannot manage other tokens.

## Environment Variables

- `SESSION_SECRET` — **required**
- `PORT` — default `4666`
- `DATABASE_PATH` — default `./data/budgeting.db`
- `LOG_LEVEL` — default `info`
- `CURRENCY` — default `€`

## Database

SQLite3 with foreign keys enabled. Custom `LOWER_UNICODE()` function registered for Greek text search.
Tables: `users`, `categories`, `actions`, `api_tokens`. Category is required on actions.
`api_tokens` uses soft-delete (`deleted_at`) and supports optional `expires_at`.

## Docker

```bash
make docker-build   # Build image
make docker-up      # Start containers
make docker-down    # Stop containers
```

CI releases Docker images to ghcr.io on version tags (`v*.*.*`). A second
workflow (`.github/workflows/release-cli.yml`) runs goreleaser on the same tag
to attach `budgeting-cli` binaries (linux/darwin × amd64/arm64) and
`skill/SKILL.md` to the GitHub Release.
