# 001 - Foundation Plan

## Overview

Build the foundation for the stuf TUI finance tool, plus the first real vertical slice: accounts and balances. This plan should prove that the app can boot, create and migrate a local SQLite database, load config, seed reference data, render a TUI shell, mutate data safely, record effective history, and undo current-session mutations.

This plan intentionally keeps the executable schema small. Future domains such as transactions, budgets, owed items, tags, and reports should be added by later plans and migrations. Deferred design notes are preserved at the bottom of this document as current intent, not as implementation scope for `001`.

**Stack**: Go, Bubble Tea, SQLite, Goose (migrations), SQLC (query generation)

## Scope

### In Scope

- Project scaffolding: `go.mod`, `go.sum`, `Makefile`, `sqlc.yaml`, package directories
- SQLite startup: create/open `db.sqlite`, verify stuf metadata, run migrations, validate required schema
- Embedded goose migrations
- Embedded currency/rate seed data
- Config loading/creation with app currency
- Money primitives: integer+scale storage, parsing, formatting, arithmetic, deterministic conversion
- Repository/service/model boundaries
- Shared mutation/history/undo boundary
- Minimal Bubble Tea shell with URL-style routing
- Dashboard with real account/balance-derived totals where available, and placeholders for deferred domains
- Accounts vertical slice
- Balances vertical slice
- Current-session visible history and `ctrl-z` undo for account/balance mutations
- Persisted effective history rows for account/balance mutations
- Tests across money, config, seeding, startup, repos, services, model behavior, and undo

### Out of Scope for 001

- Full transaction workflows
- Full tag workflows
- Full budget/category/allocation workflows
- Full owed/party/settlement workflows
- Real reports
- Effective transaction row implementation
- Query UI
- Export UI
- Config editing UI
- Custom currency creation
- Historical FX snapshots
- Runtime currency fetching
- Investment-specific features

## Principles

- **Store user input, compute everything else** — if it can be derived at runtime, don't persist it.
- **Balances anchor truth** — transactions explain movement but never update balances. In `001`, there are no transactions yet; accounts and balances establish the anchor model first.
- **History is effective mutation history, not an audit log** — persisted history is a single-branch recovery log for the current database state. Undo is handled in-memory for the current session. On undo, reverse the mutation and silently delete the corresponding persisted history row. No undo entry is appended.
- **No premature caching** — account totals are computed from latest balances at query time. Future budget/owed/report totals should follow the same principle.
- **Dates as TEXT** — `YYYY-MM-DD`, `YYYY-MM`, and ISO 8601 datetimes are stored as text and validated Go-side.
- **No lipgloss** — Bubble Tea is the framework; rendering is in-house plain string formatting.
- **TDD-first** — write tests before implementation where practical. Use unit tests for services, integration tests for repos with real SQLite, and model tests for Bubble Tea `Update`/`View` behavior.
- **Repository pattern** — repos wrap sqlc-generated queries, services depend on repo interfaces, models depend on service interfaces. Everything is mockable at layer boundaries.

## Architecture

### Dependency Flow

```text
main.go ──creates──> concrete repos (with real DB)
                  ──creates──> concrete services (with repos)
                  ──creates──> root model (with services)

model_test ──uses──> mock services
service_test ──uses──> mock repos
repo_test ──uses──> real SQLite (temp file)
```

Models must not hold database connections or repos directly. They receive service interfaces and own only UI state: current URL/path, config, route/session state, undo stack, visible session history, focused component state, and recoverable display errors.

### Mutation & Undo Boundary

All service-level mutations must go through a shared mutation/history boundary. That boundary records old/new JSON data, writes persisted effective history, and returns/registers a current-session undo operation.

Undo behavior:

- Reverse the DB mutation.
- Remove the visible session history row.
- Delete the corresponding persisted history row.
- Return to `/` and re-render.
- Do not append an undo history entry.

Tests must cover that each account/balance mutation records effective history and can be reversed through the undo path. Service interfaces should make mutation/undo participation hard to skip. Direct repo writes from models are not allowed.

### Directory Structure

Required for `001`:

```text
stuf/
├── cmd/
│   └── stuf/
│       └── main.go              # wiring only
├── internal/
│   ├── db/                      # sqlc generated (do not edit)
│   ├── migration/               # embedded goose migration files
│   ├── seed/                    # embedded currency/rate data
│   ├── config/                  # config loading/creation
│   │   └── config_test.go
│   ├── repo/                    # repository interfaces + impls + startup
│   │   ├── account.go
│   │   ├── account_test.go      # integration tests with real SQLite
│   │   ├── balance.go
│   │   ├── balance_test.go
│   │   ├── currency.go
│   │   ├── currency_test.go
│   │   ├── history.go
│   │   ├── history_test.go
│   │   └── sqlite.go            # DB connection, startup, seeding, constructors
│   ├── money/                   # money arithmetic, conversion, formatting
│   │   ├── money.go
│   │   └── money_test.go
│   ├── service/                 # business logic
│   │   ├── account.go
│   │   ├── account_test.go      # unit tests with mock repos
│   │   ├── balance.go
│   │   ├── balance_test.go
│   │   ├── currency.go
│   │   ├── currency_test.go
│   │   ├── history.go
│   │   └── history_test.go
│   ├── model/                   # bubbletea models
│   │   ├── app.go               # root model, routing
│   │   ├── app_test.go
│   │   ├── dashboard.go
│   │   └── accounts.go
│   └── component/               # reusable TUI components
│       ├── text_input.go
│       ├── text_input_test.go
│       ├── select_input.go
│       ├── select_input_test.go
│       ├── filter_list.go
│       ├── filter_list_test.go
│       ├── table.go
│       ├── table_test.go
│       └── form.go
├── plans/
│   └── 001-foundation-plan.md
├── README.md
├── go.mod
├── go.sum
├── Makefile
└── sqlc.yaml
```

Future domain packages/files should be added by future plans when their workflows are implemented. Do not create repo/service files for transactions, budgets, owed, tags, or reports in `001` unless a foundation dependency proves they are needed.

## Phase 1: Project Scaffolding

### Dependencies

- `github.com/charmbracelet/bubbletea` — TUI framework
- `modernc.org/sqlite` — pure-Go SQLite driver
- `github.com/pressly/goose/v3` — migrations
- `github.com/sqlc-dev/sqlc` — query generation (dev dependency)
- Standard library for JSON, formatting, file paths, time, and testing

### Makefile Targets

- `make generate` — run sqlc generate
- `make migrate` — run goose migrations against `db.sqlite`
- `make run` — build and run
- `make build` — build binary
- `make test` — run tests

## Phase 2: Minimal Database Schema

### Migration Strategy

- Use numbered goose migrations, starting with `00001_foundation.sql`.
- On startup, verify `db.sqlite` exists or create it.
- Verify the database is SQLite.
- Verify it is a stuf database via `app_meta`.
- Run pending migrations.
- Validate required foundation schema.
- Seed missing currency/rate data.
- Embed migrations and seed data into the binary via `embed`.

### SQLite Notes

- All integer types are `INTEGER`.
- Booleans are stored as `INTEGER` (`1=true`, `0=false`).
- Dates are stored as `TEXT` and validated Go-side.
- Money is stored as integer amount plus scale. Do not use floats.
- Foundation uses a minimal schema. Future domains get their own migrations.

### Tables in 001

#### `app_meta`

| Column | Type | Constraints | Notes |
|--------|------|-------------|-------|
| key | TEXT | PRIMARY KEY | |
| value | TEXT | NOT NULL | |

Verifies "this is a stuf database" and tracks app/schema metadata.

Required initial rows:

- `app = stuf`
- `schema = foundation`

#### `currencies`

| Column | Type | Constraints | Notes |
|--------|------|-------------|-------|
| id | INTEGER | PRIMARY KEY | |
| code | TEXT | UNIQUE NOT NULL | ISO 4217-like (`USD`, `HKD`) |
| name | TEXT | NOT NULL | Full name (`US Dollar`) |
| scale | INTEGER | NOT NULL DEFAULT 2 | Decimal places |
| created_at | TEXT | NOT NULL | ISO 8601 |
| updated_at | TEXT | NOT NULL | ISO 8601 |

Seeded from embedded data. Not user-creatable in v1.

#### `currency_rates`

| Column | Type | Constraints | Notes |
|--------|------|-------------|-------|
| id | INTEGER | PRIMARY KEY | |
| currency_id | INTEGER | UNIQUE NOT NULL REFERENCES currencies(id) | One rate per currency |
| rate_to_usd_amount | INTEGER | NOT NULL | e.g. HKD: `128` |
| rate_to_usd_scale | INTEGER | NOT NULL DEFAULT 3 | e.g. `128`, scale `3` = `0.128` |
| updated_at | TEXT | NOT NULL | ISO 8601 |

Rate is stored as integer+scale. `rate_to_usd_amount=128` and `rate_to_usd_scale=3` means `1 HKD ≈ 0.128 USD`. USD itself is `amount=1`, `scale=0`. Rounding must be deterministic and covered by money tests.

#### `accounts`

| Column | Type | Constraints | Notes |
|--------|------|-------------|-------|
| id | INTEGER | PRIMARY KEY | Internal, immutable |
| name | TEXT | UNIQUE NOT NULL | User-facing strict slug, editable |
| currency_id | INTEGER | NOT NULL REFERENCES currencies(id) | Locked if balances exist |
| on_budget | INTEGER | NOT NULL DEFAULT 1 | `1=true`, `0=false` |
| hidden | INTEGER | NOT NULL DEFAULT 0 | Hidden accounts excluded from default list |
| notes | TEXT | NOT NULL DEFAULT '' | Plain text |
| created_at | TEXT | NOT NULL | ISO 8601 |
| updated_at | TEXT | NOT NULL | ISO 8601 |

Account deletion is not implemented in v1. Accidental newly-created accounts can be undone during the current session if they are still undoable.

#### `balances`

| Column | Type | Constraints | Notes |
|--------|------|-------------|-------|
| id | INTEGER | PRIMARY KEY | |
| account_id | INTEGER | NOT NULL REFERENCES accounts(id) | |
| date | TEXT | NOT NULL | `YYYY-MM-DD` |
| amount | INTEGER | NOT NULL | e.g. `5000000` = HKD 50,000.00 |
| scale | INTEGER | NOT NULL DEFAULT 2 | |
| notes | TEXT | NOT NULL DEFAULT '' | Plain text |
| created_at | TEXT | NOT NULL | ISO 8601 |
| updated_at | TEXT | NOT NULL | ISO 8601 |

Constraints and indexes:

- `UNIQUE(account_id, date)`
- `INDEX(account_id, date)`

Account current balance is the latest balance entry for that account. If no balance exists, display `0` in the account currency with `as of: (no balance entered yet)`.

#### `history`

| Column | Type | Constraints | Notes |
|--------|------|-------------|-------|
| id | INTEGER | PRIMARY KEY | |
| timestamp | TEXT | NOT NULL | ISO 8601 |
| action | TEXT | NOT NULL CHECK(action IN ('create', 'add', 'edit', 'delete')) | User-facing verb |
| path | TEXT | NOT NULL | e.g. `/accounts/hsbc-one` |
| old_data | TEXT | | JSON, null for creates |
| new_data | TEXT | | JSON, null for deletes |

Persisted effective mutation history, not an audit log. It is a single-branch recovery log for the current database state. On current-session undo, reverse the DB mutation and silently delete the corresponding history row. No undo entry is appended.

Foundation action verbs:

- `create` for accounts
- `add` for balances
- `edit` for account/balance modifications
- `delete` for balance deletion

`INDEX(timestamp)`.

### Foundation Data Relationships

```text
currencies
  ├── currency_rates
  └── accounts
        └── balances

history records effective account/balance mutations
```

### Computed Values in 001

| Value | Computation |
|-------|-------------|
| Account current balance | Latest balance entry for that account, or zero if none |
| Account list total | Sum latest account balances, grouped by on-budget/off-budget |
| Dashboard total | Sum latest balances across visible accounts converted to app currency where possible |
| Dashboard budgeted | `0` until budgets exist |
| Dashboard growth | `0` placeholder until reports exist |
| Dashboard owed values | `0` until owed exists |

Missing display conversion data shows the original currency amount and a clear warning instead of silently converting. Converted totals that depend on missing rates omit the affected converted amount and show a warning.

## Phase 3: Money Package

### `money.Money` Type

```go
type Money struct {
    Amount int64
    Scale  int
}
```

Required behavior:

- `Add`
- `Sub`
- `Negate`
- `ConvertToScale(newScale)`
- `Equals`
- `IsZero`
- `IsPositive`
- `IsNegative`
- `Format(currencyCode string)`
- `Parse(input string)`

All arithmetic must validate scale compatibility or convert deterministically. Do not use floats.

### `money.CurrencyRate`

Used for cross-currency display conversion.

```go
func Convert(amount Money, rateToUSD Money, targetRateToUSD Money, targetScale int) (Money, error)
```

Exact signature can change during implementation, but tests must prove deterministic rounding and USD 1:1 behavior.

### Formula Parser

Formula parsing is deferred unless implementation naturally needs it for amount entry. Owed items are the first strong use case for formulas, so a later owed plan may own this.

## Phase 4: App Startup

### Startup Sequence

1. Check for `db.sqlite` in current working directory.
2. If not found, create it and run all migrations.
3. If found, verify it is a valid SQLite file.
4. Verify it is a stuf database via `app_meta`.
5. Run pending migrations.
6. Validate required foundation schema.
7. Seed missing currency/rate data from embedded data.
8. Load config from local `config.jsonc` in cwd, then fallback to global `~/.config/stuf/config.jsonc`.
9. If no config exists, create global config with detected currency or USD.
10. If USD fallback is used, show a startup warning that app currency defaulted to USD and can be changed in config.
11. Start Bubble Tea program.

### Config Structure

```jsonc
{
  // stuf config
  "currency": "HKD"
}
```

Minimal for v1. Date format is fixed ISO. Editing config from the UI is deferred.

## Phase 5: TUI Shell and Accounts Slice

The initial TUI shell proves the app boots, connects to DB, and shows dashboard/navigation structure. Accounts and balances prove the repo/service/model/form/history/undo stack end-to-end.

### Shell Requirements

- Bubble Tea `Model` holds current URL/path, service interfaces, config, route/session state, undo stack, visible session history, focused component state, and recoverable display errors.
- URL-based routing: `/`, `/accounts/`, `/accounts/create/`, `/accounts/list/`, etc.
- Global keybinds: `ctrl-c` quit, `ctrl-z` undo, `esc` back/exit, `?` help.
- Number hotkeys work only in menu screens, not in forms.
- All rendering uses in-house string formatting, no lipgloss.
- `/reports/`, `/budgets/`, `/transactions/`, and `/owed/` are not real workflows in `001`. They may be hidden or shown as placeholders, but must not pretend to be implemented.

### Dashboard in 001

The dashboard should show real values where accounts/balances make them possible, and explicit placeholders for deferred domains.

```text
# stuf

total       : HKD 0.00
budgeted    : HKD 0.00

period      : 2026-05

growth
on-budget  : HKD 0.00
total      : HKD 0.00

you owe ppl : HKD 0.00
ppl owe you : HKD 0.00

/

> 1) accounts
  2) settings
  3) backup

---
up/down : navigate
enter   : confirm
esc     : exit app
?       : help
```

If future menu placeholders are shown, their detail pages should clearly say `coming later`.

### Keybind Behavior

- `ctrl-c` quits immediately and gracefully, no confirmation. Quitting clears undo history.
- `esc` at `/` opens exit confirmation. Default selection is `no`.
- Exit confirmation shows `undo history will be cleared` only if current-session undo history exists.
- `esc` from exit confirmation cancels and returns to normal `/`.
- `esc` from a create/edit form discards the draft immediately.
- `esc` everywhere else goes back one level.
- `ctrl-z` undoes the latest visible history row, removes that row from visible history, returns to `/`, and re-renders.
- `?` shows context-sensitive help. Press `?` again or `esc` to exit help.
- `j/k`, `tab/shift-tab`, and arrows navigate where appropriate.
- Arrow keys should not conflict with text editing in forms.

### Accounts Requirements

Routes:

- `/accounts/`
- `/accounts/list/`
- `/accounts/hidden/`
- `/accounts/create/`
- `/accounts/{name}/`
- `/accounts/{name}/edit/`
- `/accounts/{name}/balances/`
- `/accounts/{name}/balances/add/`
- `/accounts/{name}/balances/{date}/`
- `/accounts/{name}/balances/{date}/edit/`

Account behavior:

- Creating an account does not ask for an opening balance.
- Account names are strict slugs.
- Account name is user-facing and editable.
- Internal account ID is immutable.
- Account currency defaults to app currency.
- Account currency can be edited only if the account has no balances.
- If balances exist, currency field is read-only/disabled.
- Account names must be unique.
- Keeping the same name while editing is allowed.
- Accounts can be hidden and shown.
- Hidden accounts are excluded from the default account list.
- Hidden accounts preserve balances, history, and future report relevance.
- Account deletion is deferred for v1.

### Balances Requirements

Balance behavior:

- Balance entries inherit account currency.
- Date defaults to today.
- Date is required.
- Balance amount is required.
- Positive, zero, and negative balances are allowed.
- Fiat balances accept up to the currency scale.
- Balances sort newest first.
- Only one balance is allowed per account per date.
- Duplicate account/date balances are rejected with a recoverable error.
- Users should edit an existing balance instead of replacing through add.
- Balance deletion happens immediately, without a confirmation screen in `001`.
- Balance deletion is undoable if it is in current-session undo history.

### Post-Mutation Navigation

After a successful mutation, redirect as follows:

| Action | Redirect |
|--------|----------|
| Create account | `/accounts/list/` |
| Edit account | `/accounts/{name}/` using updated name if changed |
| Hide/show account | `/accounts/{name}/` |
| Add balance | `/accounts/{name}/balances/` |
| Edit balance | `/accounts/{name}/balances/` |
| Delete balance | `/accounts/{name}/balances/` |

### Error Display Behavior

- Errors remain visible as long as the user is still on the current page.
- Errors disappear when the user navigates back.
- Errors disappear after a successful action on the same page.
- Errors should not crash the app. Recoverable errors show a clear message.
- Backend validation errors supplement frontend validation.

### Backup & Settings Screens

- `/settings/` shows active config path and app currency. Read-only. Editing happens via the config file directly.
- `/backup/` shows database path, last backup path if known, and a `create backup` action.
- Backup creates `db.YYYY-MM-DD-HHMM.sqlite` beside the active DB.
- Backup does not write undo history.

## Phase 6: Test Coverage

### Money Tests

- Parse valid money input.
- Reject invalid money input.
- Format by currency scale.
- Add/subtract compatible amounts.
- Convert scale deterministically.
- Convert currencies with seeded rates.
- USD rate is 1:1.
- Rounding is deterministic.

### Config Tests

- Parse valid config.
- Reject invalid config with clear error.
- Create default config when none exists.
- Location detection fallback to USD.

### Startup and Seeding Tests

- Fresh DB is created.
- Migrations run on fresh DB.
- Non-stuf SQLite DB is rejected with clear error.
- Required schema validation fails clearly if schema is incomplete.
- Fresh DB has expected currencies.
- Re-running seeding is idempotent.
- Currency rates are seeded correctly.

### Repo Tests

- Account create/list/get/update/hide/show with real SQLite temp DB.
- Account name uniqueness.
- Account currency lock checks when balances exist.
- Balance add/list/get/update/delete with real SQLite temp DB.
- Balance uniqueness by account/date.
- Latest balance query.
- History create/list/delete with real SQLite temp DB.

### Service Tests

- Account mutations validate input and record history.
- Balance mutations validate input and record history.
- Undo reverses account create where valid.
- Undo reverses account edit/hide/show.
- Undo reverses balance add/edit/delete.
- Direct mutation paths cannot skip history boundary.

### Model Tests

- Dashboard renders empty state.
- Account menu navigation.
- Account create form validation behavior.
- Account list/detail rendering.
- Balance add/edit/delete navigation.
- Recoverable errors remain on page and clear correctly.
- `esc`, `ctrl-c`, `ctrl-z`, and `?` behavior.

## Execution Order

1. **`go.mod` + directory structure** — initialize module, Makefile, sqlc config, package scaffolding.
2. **Minimal goose migration** — create `app_meta`, `currencies`, `currency_rates`, `accounts`, `balances`, and `history`.
3. **Currency seed data** — embed common currencies and USD-relative rates.
4. **`money` package** — arithmetic, conversion, formatting, parsing. TDD first.
5. **SQLC queries** — generate code for the minimal schema only.
6. **Repo package** — SQLite startup, migrations, seeding, account, balance, currency, and history repos. Integration tests with real SQLite temp files.
7. **Config package** — config discovery, validation, creation, and fallback behavior. Tests first.
8. **Shared mutation/history/undo boundary** — effective history write/delete plus current-session undo registration.
9. **Service package** — account, balance, currency, and history services. Unit tests with mock repos.
10. **Bubble Tea shell** — boot, dashboard render, routing, keybind framework. Model tests.
11. **Accounts/balances UI flows** — create/list/detail/edit/hide/show accounts; add/list/detail/edit/delete balances. Model tests.
12. **Verification** — `make generate`, `make test`, `make build`, and a manual smoke run.

## Expected Result After 001

After this plan is executed, a user should be able to:

- Run the TUI locally.
- Get a fresh `db.sqlite` and config automatically if missing.
- See an empty dashboard with real zero values.
- Create accounts.
- Add balances to accounts.
- See latest balances reflected in account lists/details and dashboard totals.
- Edit/hide/show accounts.
- Edit/delete balances.
- See visible session history for mutations.
- Press `ctrl-z` to undo the latest current-session mutation.
- Quit without losing persisted database state.
- Copy the SQLite database directly for backup or inspection.

## Deferred Domain Design Notes

The notes below preserve current v1 design intent from the README and earlier planning. They are **not executable scope for 001**. Future plans should use them as starting points, but they may be revised based on what is learned while building the foundation/accounts slice.

### Future Schema Candidates

Future migrations may add these tables:

- `tags`
- `budget_categories`
- `budgets`
- `allocations`
- `transactions`
- `transaction_tags`
- `parties`
- `owed_items`
- `settlements`

Tables with user-facing refs such as `transactions`, `owed_items`, and `settlements` should likely use `INTEGER PRIMARY KEY AUTOINCREMENT` so refs like `tx-000001`, `owed-000001`, and `set-000001` stay stable and are not reused after deletes.

### Tags

- Tags are transaction breadcrumbs for v1.
- Tags are transaction-only for v1.
- Tag names are strict slugs and globally unique.
- Tags have immutable internal IDs and editable names.
- Tags have plain-text multiline notes.
- Fresh app does not seed tags.
- Tags are not hidden for v1.
- Tag deletion, merge, usage counts, and detail backlinks are deferred.
- `transaction_tags` join table is enough for v1.
- Future taggable records can add their own join tables.

### Budgets and Categories

- Budgets are global envelope-style allocations, not monthly category budgets.
- Budgets carry over by default.
- Budgets behave like proxy accounts for on-budget money.
- Creating a budget is separate from allocating money to it.
- Budget names are strict slugs and globally unique.
- Budgets have exactly one currency.
- Budget currency is fixed once allocations or linked transactions exist.
- Accounts and budgets can be hidden; categories are not hidden for v1.
- Every budget belongs to one category.
- Budget categories use strict slugs and globally unique names.
- Seed built-in category `uncategorized` when budgets are implemented.
- `uncategorized` cannot be renamed or deleted in v1.
- Newly-created budgets default to `uncategorized`.
- Normal categories are shown even when empty; `uncategorized` is hidden when empty.
- Category deletion is deferred.

Potential budget tables:

- `budget_categories`: `id`, `name`, `notes`, timestamps.
- `budgets`: `id`, `name`, `currency_id`, `category_id`, default allocation fields, goal fields, `hidden`, `notes`, timestamps.
- `allocations`: `id`, `budget_id`, `date`, `delta_amount`, `delta_scale`, `notes`, timestamps.

Budget computations should remain query-time derived:

- Budget allocated = `SUM(allocations.delta_amount)` by budget.
- Budget spent = sum effective expense transaction rows linked to budget.
- Budget balance = allocated minus spent.
- Budget available = on-budget balances converted to app currency minus budget balances converted to app currency minus open `you_owe` remaining converted to app currency.
- Money people owe you does not increase available until it appears in on-budget balances.

Default allocation intent:

- If a budget has default allocation enabled, budget detail may show `apply default allocation`.
- Confirming creates one allocation dated today with `delta_amount=default_allocation_amount`.
- Bulk default allocation, automatic recurring allocation, and monthly allocation flows remain deferred.

Goal intent:

- Goal remaining = goal target amount minus budget balance.
- Goal monthly needed = remaining divided by months left, counting through target month inclusive.
- Multiple active goals, maintain-balance goals, and goal report drilldowns are deferred.

### Transactions

- Transactions explain account balance movement but do not update balances.
- Transaction currency is the event/explanation currency.
- Transaction currency defaults to selected account currency and remains editable.
- Each transaction has exactly one currency.
- Transaction budget link is nullable.
- V1 UI only allows budget links for expense transactions.
- One transaction links to at most one budget for v1.
- Child expense transactions can link to different budgets.
- Parent transactions may be unbudgeted while children split across budgets.
- Mixed-type children are blocked in v1 UI.
- Deleting a transaction with children is blocked in v1.
- Explicit transfer transactions are deferred.

Potential `transactions` fields:

- `id`, `ref`, `date`, `type`, `amount`, `scale`, `currency_id`, `account_id`, `parent_id`, `budget_id`, `notes`, timestamps.

### Effective Transaction Rows

Reports and budget-spent calculations should use effective rows, not raw parent + child rows, to prevent double counting.

- If a transaction has no children, it contributes itself as a single effective row.
- If a transaction has children, it contributes child effective rows plus one parent remaining row if remaining is non-zero.
- Apply recursively for deeper transaction trees.
- Parent remaining = parent amount minus sum child amounts converted to parent currency.
- Parent remaining rows are virtual/read-only and have no transaction ref.
- Parent remaining rows keep the parent date/account/type/budget/tags/notes.
- Budget spent uses the same effective-row logic.
- Parent remaining rows use the parent budget link if present.
- If converted children total exceeds parent amount, remaining becomes negative and advisory; it does not block input.
- Effective rows count in the coverage period containing their own transaction date.
- Parent remaining row counts on the parent transaction date.
- Child rows can appear in a different report period from their parent.

### Owed Items and Settlements

- Owed items behave like proxy accounts for money between people.
- Settlements explain owed item movement but do not update balances.
- Each owed item has exactly one currency.
- Different owed items can use different currencies.
- Owed item currency defaults to app currency.
- Each settlement has exactly one currency.
- Settlement currency defaults to owed item currency and remains editable.
- Settlement amount converts into owed item currency to reduce remaining.
- Missing settlement-to-owed-item conversion blocks confirmation because remaining math must be exact.
- Related transaction UX, transaction-settlement shortcuts, settlement tags, and owed report integration are deferred.

Potential owed tables:

- `parties`: `id`, `name`, `notes`, timestamps.
- `owed_items`: `id`, `ref`, `direction`, `party_id`, `date`, `amount`, `scale`, `formula`, `currency_id`, `notes`, timestamps.
- `settlements`: `id`, `ref`, `owed_item_id`, `date`, `amount`, `scale`, `currency_id`, `notes`, timestamps.

Owed remaining should be computed at query time:

- Remaining = owed amount minus sum settlements converted to owed item currency.
- Owed status is inferred: if remaining is zero, the item is settled.
- Settled items are hidden from open owed lists.

### Reports

Reports are deferred from `001`. The foundation dashboard may show placeholders, but real report screens should wait until transactions and effective rows exist.

Future report intent:

- Report growth = end balance minus start balance for period.
- Report income = sum effective income rows in period, or growth if no income transactions exist, marked `(assumed)`.
- Report expenses = income minus growth, marked `(derived)`.
- Explained expenses = sum effective expense transaction rows in the period.
- Unexplained expenses = derived expenses minus explained expenses.

Expense explanation display order:

1. Derived — income minus growth, or growth assumed as income when no income transactions exist.
2. Explained — sum effective expense transaction rows in the period.
3. Unexplained — remaining expense amount not explained by transactions.

Report period boundary intent:

| Rule | Definition |
|------|------------|
| Start balance | Latest balance on or before first day of period |
| End balance | Latest balance on or before last day of period |
| No start balance | Start = 0 |
| No end balance | End = start |
| Zero balances | Use 0 -> 0 |
| One usable balance | Assume flat: start = end = that balance |

### Currency Conversion Future Notes

- Account currency = balance currency.
- Budget currency = proxy account anchor currency.
- Owed item currency = proxy account anchor currency.
- Transaction currency = event/explanation currency.
- Settlement currency = payment event currency.
- Child transactions and settlements convert into parent/proxy anchor currency for remaining/balance math.
- Common currencies and rates are seeded from app data.
- No runtime network fetch is required for currency conversion in v1.
- Latest seeded/cached rates are enough for v1 current views.
- Missing conversion data should show original currency and omit affected converted totals with a clear warning.
- Historical conversion snapshots and manual conversion rate override are deferred.

### v1 Scope Exclusions

The following remain explicitly out of v1 unless later plans change scope:

- Account deletion, tag deletion, budget deletion, category deletion.
- Explicit transfer transactions.
- Rich tree visualizations in reports.
- Report-to-input shortcuts.
- Preserving dirty create drafts after `esc`.
- Opening original records from report detail.
- Recurring/monthly allocation flow.
- Yearly expense allocation flow.
- Bulk apply default allocations.
- Automatic recurring allocations.
- Multiple active goals per budget.
- Maintain-balance goals.
- Goal report drilldowns.
- Related transaction UX for owed.
- Transaction-settlement shortcuts.
- Settlement tags.
- Owed report integration.
- Tag merge.
- Tag usage counts.
- Tag detail backlinks.
- Custom currency creation.
- WAL mode.
- Historical currency rate snapshots.
- Manual currency rate overrides.
- Config editing UI.
- Export UI.
- Investment-specific features.
