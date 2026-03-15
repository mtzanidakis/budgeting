# CLAUDE.md

## Project Overview

Family budgeting PWA — Go backend with embedded vanilla JS frontend, SQLite database.
Multi-user income/expense tracking with categories, charts, and i18n (EN/EL).

## Build & Run

```bash
# CGO is required (SQLite3 driver)
make build        # Build server + CLI to bin/
make run          # Build and run server
make test         # go test -v ./...
make clean        # Remove bin/

# Manual build (version injected via ldflags)
CGO_ENABLED=1 go build -o bin/server ./cmd/server
CGO_ENABLED=1 go build -o bin/cli ./cmd/cli
```

## Project Structure

```
cmd/server/          Main HTTP server (Chi router), embeds frontend/
cmd/server/frontend/ Static assets: HTML template, CSS, vanilla JS, PWA manifest
cmd/cli/             Admin CLI for user management
internal/auth/       Password hashing (bcrypt), session store (in-memory)
internal/config/     Env var config via caarlos0/env
internal/database/   SQLite3 operations, schema migrations, query builders
internal/handlers/   HTTP handlers (auth, actions, categories, users, config, static)
internal/middleware/  Auth + logging middleware
internal/models/     Data structs (User, Action, Category, ActionType)
internal/version/    Build-time version injection
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

## Environment Variables

- `SESSION_SECRET` — **required**
- `PORT` — default `8080`
- `DATABASE_PATH` — default `./data/budgeting.db`
- `LOG_LEVEL` — default `info`
- `CURRENCY` — default `€`

## Database

SQLite3 with foreign keys enabled. Custom `LOWER_UNICODE()` function registered for Greek text search.
Tables: `users`, `categories`, `actions`. Category is required on actions.

## Docker

```bash
make docker-build   # Build image
make docker-up      # Start containers
make docker-down    # Stop containers
```

CI releases Docker images to ghcr.io on version tags (`v*.*.*`).
