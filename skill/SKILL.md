---
name: budgeting
description: Query and manage personal budget entries (income, expenses, categories) via the budgeting-cli tool. Use when the user asks to log transactions, review spending, query financial history, or generate summaries for their personal budgeting app.
---

# Budgeting

Client for a self-hosted family budgeting service. All operations go through
`budgeting-cli`, which talks to the HTTP API using a per-user token.

## Setup

Set two environment variables once per shell (or per command):

- `BUDGETING_URL` — base URL of the server, e.g. `http://localhost:4666`
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

`--from` and `--to` are **inclusive**. You are expected to resolve relative
date expressions (in any language) to concrete `YYYY-MM-DD` bounds yourself
before calling the CLI. Use the user's local calendar; if unsure about the
year, default to the current year, or ask.

Natural-language → CLI examples:

| User request | Resolved range | Command |
|---|---|---|
| "δώσε μου τα έξοδα από 13/4 μέχρι 16/4" | 2026-04-13 → 2026-04-16 | `budgeting-cli actions list --type expense --from 2026-04-13 --to 2026-04-16` |
| "δώσε μου τα έσοδα της πρώτης εβδομάδας του Μαρτίου" | 2026-03-01 → 2026-03-07 | `budgeting-cli actions list --type income --from 2026-03-01 --to 2026-03-07` |
| "show me last month's expenses" (today = 2026-04-19) | 2026-03-01 → 2026-03-31 | `budgeting-cli actions list --type expense --from 2026-03-01 --to 2026-03-31` |
| "έξοδα φαγητού τον Απρίλιο" | look up category id for "Φαγητό" first, then: | `budgeting-cli actions list --type expense --category <id> --from 2026-04-01 --to 2026-04-30` |
| "τι έκανα σήμερα" | today only | `budgeting-cli actions list --from 2026-04-19 --to 2026-04-19` |

"Πρώτη εβδομάδα" conventionally means days 1–7. "Εβδομάδα" without a
specifier means the calendar week containing the date (Mon–Sun). Ask if
ambiguous.

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
- **Resolve categories semantically, not literally**: whenever the user
  names a category — for filtering (`--category`), creating, or updating
  actions — run `categories list --type <type>` first and match by
  *meaning*, not exact string. The user's word is often a generic label
  for a more specific category description. Examples:
  - "φαγητό" / "food" → "Εστιατόρια, Καφέ / Delivery", or
    "Σούπερ μάρκετ", or both, depending on context (eating out vs. groceries)
  - "μεταφορές" / "transport" → "Βενζίνη", "ΜΜΜ", "Ταξί", "Παρκινγκ"
  - "σπίτι" → "Ενοίκιο", "ΔΕΗ", "ΕΥΔΑΠ", "Internet", κλπ
  Behavior:
  1. If **exactly one** category plausibly matches, use it and proceed.
  2. If **multiple** match, either combine them (make one call per category
     id and merge results) or ask the user which one(s) they meant — pick
     based on how ambiguous the request is.
  3. If **none** match, tell the user what categories exist and ask.
  4. Never invent a category id. Never create a new category silently just
     to fulfill a request — ask first.
- **Before deleting anything**, confirm with the user unless the request was
  unambiguous. Deletes are immediate (actions) or propagating (categories).
- **Authentication errors** (HTTP 401) mean the token is invalid, expired,
  or revoked. Tell the user to regenerate it from the web UI.
- **The token scope is full access** to the owning user's data. It cannot
  create or revoke other tokens — only the session-authenticated UI can.
