---
name: budgeting
description: Query and manage personal budget entries (income, expenses, categories) via the budgeting-cli tool. Use when the user asks to log transactions, review spending, query financial history, or generate summaries for their personal budgeting app.
---

# Budgeting

Client for a self-hosted family budgeting service. All operations go through
`budgeting-cli`, which talks to the HTTP API using a per-user token.

## Setup

Set two environment variables once per shell (or per command):

- `BUDGETING_URL` — base URL of the server, e.g. `http://localhost:8080`
- `BUDGETING_TOKEN` — API token generated from the web UI under
  **User menu → API Tokens → Generate new token**. Tokens start with `bdg_`.

You can also pass `--url` and `--token` flags per-invocation to override the
env vars. Add `--pretty` to any command to indent the JSON output.

## Data model

- **Action**: a single income or expense entry. Fields: `id`, `user_id`,
  `type` ("income"|"expense"), `date` (YYYY-MM-DD), `description`, `amount`
  (decimal, dot separator), `category_id`.
- **Category**: a label bound to a specific `action_type`. Fields: `id`,
  `description`, `action_type` ("income"|"expense").
- Every action requires a `category_id` matching its type. If the user names
  a category, look up its ID with `categories list` before creating an action.

## Commands

All commands print JSON on stdout. Errors go to stderr with non-zero exit.

### me — current user
```
budgeting-cli me
```

### actions list — filter and list
```
budgeting-cli actions list [--from YYYY-MM-DD] [--to YYYY-MM-DD]
                           [--type income|expense] [--category ID]
                           [--user NAME] [--search TEXT]
                           [--limit N] [--offset N]
```
Without `--offset`, returns a plain JSON array. With `--offset`, returns
`{"actions": [...], "total": N}` for pagination.

### actions create — log a new entry
```
budgeting-cli actions create --type expense --date 2026-04-19 \
  --description "Groceries" --amount 42.50 --category 3
```
All flags are required.

### actions update / delete
```
budgeting-cli actions update <ID> --type --date --description --amount --category
budgeting-cli actions delete <ID>
```
Update requires the full set of fields (same as create). Delete requires
ownership — the token's user must own the action.

### categories list / create / update / delete
```
budgeting-cli categories list [--type income|expense]
budgeting-cli categories create --description "Food" --type expense
budgeting-cli categories update <ID> --description --type
budgeting-cli categories delete <ID>
```
Deleting a category sets `category_id` of existing actions to null — the
server does not block it. Warn the user before deleting categories in use.

### charts monthly — yearly income/expense breakdown
```
budgeting-cli charts monthly [--year 2026]
```
Returns 12 months with totals.

### charts categories — category breakdown for a month
```
budgeting-cli charts categories [--year 2026] [--month 4]
```
Returns expense and income summaries grouped by category.

## Workflow notes

- **Dates** always use ISO format `YYYY-MM-DD` on input. The API may echo
  timestamps with timezone; trim before re-sending if needed.
- **Amounts** are decimals with `.` separator. Never negative — sign is
  implied by `type`.
- **Before creating an action, resolve the category**: run `categories list
  --type <type>` and match by description. Ask the user if ambiguous.
- **Before deleting anything**, confirm with the user unless the request was
  unambiguous. Deletes are immediate (actions) or propagating (categories).
- **Authentication errors** (HTTP 401) mean the token is invalid, expired,
  or revoked. Tell the user to regenerate it from the web UI.
- **The token scope is full access** to the owning user's data. It cannot
  create or revoke other tokens — only the session-authenticated UI can.
