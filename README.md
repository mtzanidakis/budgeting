# Budgeting

A self-hosted Progressive Web App for tracking household income and expenses.
One Go binary, embedded vanilla-JS frontend, SQLite storage. Works offline,
installable on mobile, scriptable from the command line and from AI agents.

## Features

**Money tracking**
- Income and expense entries with required category, date, description, amount
- Edit and delete your own entries inline; other users' entries are visible but read-only
- Per-user, per-type categories (categories bound to either income or expense)
- Filter by user, type, date range, category description, and full-text search
- Greek-aware case-insensitive search (custom `lower_unicode()` SQL function)
- Pagination on the full actions list

**Visualization**
- Dashboard with 10 most recent actions across all users
- Monthly income vs. expense bar chart for any year
- Category breakdown (expenses + income) for any month, per user or aggregate

**Multi-user**
- Family / team use with per-user ownership of entries
- Shared categories across users
- Admin-provisioned accounts (no public signup)

**Authentication**
- Username + password login with bcrypt hashing
- Session cookies (HTTP-only, SameSite=Strict) for the web UI
- Per-user **API tokens** (`bdg_...`) for scripts and AI agents — generated from
  the web UI, optional expiry, soft-delete on revocation, throttled last-used
  tracking
- Token-based auth cannot manage other tokens (session-only gate)

**UX**
- Dark / light theme with persistent preference
- English + Greek UI, togglable
- Mobile-first responsive layout with burger menu below 768px
- Installable PWA with service worker and offline UI cache

**Operations**
- Structured JSON logs with per-request tracking
- Idempotent database migrations on startup
- Two Docker Compose configurations (production, development)
- Admin CLI for user and token management with direct database access
- CI publishes Docker images to GHCR on version tags
- CI publishes `budgeting-cli` binaries (linux/darwin × amd64/arm64) and the
  Claude Code skill to GitHub Releases via GoReleaser

## Tech stack

**Backend** — Go 1.26+, Chi router, SQLite3, bcrypt, slog, [caarlos0/env](https://github.com/caarlos0/env)

**Frontend** — Vanilla JavaScript (no build step), CSS variables, Chart.js, Service Worker

## Quick start

### Local

```bash
export SESSION_SECRET="$(openssl rand -base64 32)"
make build
make admin ARGS="user:add -username alice -name Alice"   # create first user
./bin/server
```

Open http://localhost:4666 and log in.

### Docker

```bash
echo "SESSION_SECRET=$(openssl rand -base64 32)" > .env
docker compose up -d
docker compose exec budgeting-app /app/admin user:add -username alice -name Alice
docker compose logs -f
```

The production `docker-compose.yml` keeps the port unexposed; use
`docker-compose.dev.yml` if you want `localhost:4666` reachable directly.

## Configuration

| Variable          | Required | Default                  | Description                             |
| ----------------- | -------- | ------------------------ | --------------------------------------- |
| `SESSION_SECRET`  | **yes**  | —                        | Secret for session cookie encryption    |
| `PORT`            | no       | `4666`                   | HTTP port                               |
| `DATABASE_PATH`   | no       | `./data/budgeting.db`    | SQLite file path                        |
| `LOG_LEVEL`       | no       | `info`                   | `debug` / `info` / `warn` / `error`     |
| `CURRENCY`        | no       | `€`                      | Currency symbol displayed in the UI     |

## Admin CLI

The `admin` binary operates directly on the database (not through the API).
Use it for bootstrap, user management, and token provisioning. Via make:

```bash
make admin ARGS="<command> [flags]"
```

Or directly: `./bin/admin <command> [flags]`.

### User management
```bash
admin user:add     -username <name> -name "<display name>"   # prompts for password
admin user:edit    -username <name> [-name "<new name>"]     # prompts for new password
admin user:delete  -username <name>                          # confirmation required
admin user:list
```

Leaving the password prompt empty on `user:add` generates a random one and
prints it once.

### Token management
```bash
admin token:list   -username <name>
admin token:add    -username <name> -name <label> [-expires YYYY-MM-DD]
admin token:delete -id <token-id>
```

`token:add` prints the raw token once — save it immediately.

### Actions query
```bash
admin actions:query -username <name> [-type income|expense] [-date-range YYYYMMDD-YYYYMMDD]
```

## API reference

All endpoints under `/api/*` return JSON. Dates are `YYYY-MM-DD`. Amounts are
decimal with a `.` separator. Sign is implied by `type`.

### Authentication

| Method | Auth      | Description                              |
| ------ | --------- | ---------------------------------------- |
| `POST /api/login`  | public  | `{username, password}` → sets session cookie |
| `POST /api/logout` | session | Invalidates the session                  |
| `GET /api/me`      | either  | Current user info                        |
| `GET /api/config`  | public  | Public app config (currency symbol)      |

Two auth methods are accepted on protected endpoints:

- **Session cookie** — set automatically after `POST /api/login`.
- **Bearer token** — `Authorization: Bearer bdg_...`, generated per-user from
  the web UI under **User menu → API Tokens**.

### Actions

| Method | Path | Notes |
| ------ | ---- | ----- |
| `GET /api/actions`         | List with filters (see below) |
| `POST /api/actions`        | Create — category required |
| `PUT /api/actions/{id}`    | Update — ownership required |
| `DELETE /api/actions/{id}` | Delete — ownership required |

`GET /api/actions` query params:
`username`, `type` (`income`/`expense`), `date_from`, `date_to` (inclusive),
`category_id`, `search` (substring match on description, Greek-aware),
`limit`, `offset`.

When `offset` is present the response is paginated: `{"actions": [...], "total": N}`.
Otherwise a plain array is returned.

Create / update body:
```json
{"type":"expense","date":"2026-04-19","description":"Groceries","amount":42.50,"category_id":3}
```

### Categories

| Method | Path | Notes |
| ------ | ---- | ----- |
| `GET /api/categories`         | List; optional `action_type` filter |
| `POST /api/categories`        | `{description, action_type}` |
| `PUT /api/categories/{id}`    | Update |
| `DELETE /api/categories/{id}` | Actions referencing the category keep their data but lose the link |

### Charts

| Method | Path | Query params |
| ------ | ---- | ------------ |
| `GET /api/charts/monthly`    | `year` (default current), `username` |
| `GET /api/charts/categories` | `year` (default current), `month` (default current), `username` |

### Users

| Method | Path | Notes |
| ------ | ---- | ----- |
| `GET /api/users`     | All users (for filter dropdown) |
| `PUT /api/profile`   | Update own name and/or password |

### API tokens

All three endpoints reject Bearer-token auth — session cookie only, so a
compromised token cannot create or revoke other tokens.

| Method | Path | Notes |
| ------ | ---- | ----- |
| `GET /api/tokens`         | Caller's active tokens (no raw values returned) |
| `POST /api/tokens`        | `{name, expires_at?}` — raw token in response, shown once |
| `DELETE /api/tokens/{id}` | Soft-delete (row retained for audit) |

Tokens start with `bdg_`, carry 32 random bytes, and are stored as SHA-256
hashes. Optional expiry (`expires_at`, null = never). `last_used_at` is
updated asynchronously at most once per minute per token.

## Programmatic / AI agent access

`budgeting-cli` is a standalone HTTP client for the API, shipped for scripts
and AI agents. No CGO, cross-platform. Pre-built binaries for linux and darwin
(amd64 + arm64) are attached to every GitHub release along with `skill/SKILL.md`.

### Install

Download the archive for your platform from the Releases page and put
`budgeting-cli` somewhere on `$PATH`. Or build from source:

```bash
CGO_ENABLED=0 go build -o bin/budgeting-cli ./cmd/budgeting-cli
```

### Configure

```bash
export BUDGETING_URL="http://localhost:4666"
export BUDGETING_TOKEN="bdg_..."   # from User menu → API Tokens
```

### Use

Compact JSON on stdout by default; `--pretty` for indented output.

```bash
budgeting-cli me
budgeting-cli categories list --type expense
budgeting-cli actions list --from 2026-04-01 --to 2026-04-30 --type expense
budgeting-cli actions create --type expense --date 2026-04-19 \
  --description "Groceries" --amount 42.50 --category 3
budgeting-cli charts monthly --year 2026 --pretty
```

Run `budgeting-cli help` for the complete reference.

### Claude Code

Each release archive includes `skill/SKILL.md`. Copy it to
`~/.claude/skills/budgeting/SKILL.md` and Claude Code will invoke
`budgeting-cli` automatically when you ask to log, query, or analyze
transactions — in English or Greek, including natural-language date ranges
and semantic category matching.

## Project structure

```
cmd/
  server/            HTTP server with embedded frontend
    frontend/        HTML, CSS, JS, PWA manifest, service worker
  admin/             Admin CLI (direct DB access, user + token management)
  budgeting-cli/     HTTP client for scripts and AI agents
internal/
  apiclient/         HTTP client library used by budgeting-cli
  auth/              Password hashing, session store, API token helpers
  config/            Env-based configuration
  database/          SQLite operations, migrations, query builders
  handlers/          HTTP handlers
  middleware/        Auth (session + Bearer), session-only guard, logging
  models/            Data structs
skill/SKILL.md       Claude Code skill bundled with CLI releases
.goreleaser.yaml     CLI binary release config
Dockerfile
docker-compose.yml
docker-compose.dev.yml
Makefile
```

## Development

```bash
make build      # server + admin + budgeting-cli
make run        # build + run server with development session secret
make test       # go test ./...
make clean      # remove bin/ and data/budgeting.db
```

Docker helpers: `make docker-build`, `docker-up`, `docker-down`, `docker-logs`.

Database migrations run idempotently on startup; no separate migration step.

## Security

- Passwords are bcrypt-hashed; API tokens are SHA-256 hashed (tokens carry
  full entropy, so the slow hash bcrypt provides is unnecessary).
- All API endpoints except `/api/login` and `/api/config` require auth.
- Session cookies are `HttpOnly` and `SameSite=Strict`.
- Action edits and deletes are gated by ownership at the database level.
- Category deletes do not cascade to actions (the `category_id` is cleared
  instead — data is never lost when reorganising).
- Token revocation is immediate and the audit record is preserved.

## License

MIT — see [LICENSE](LICENSE).
