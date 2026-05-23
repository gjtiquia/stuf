# 001 - Foundation Plan

## Overview

Build the foundation for the stuf TUI finance tool. This plan covers project scaffolding, database schema, and the minimal Go scaffolding needed to start building features on top.

**Stack**: Go, Bubble Tea, SQLite, Goose (migrations), SQLC (query generation)

## Principles

- **Store user input, compute everything else** — if it can be derived at runtime, don't persist it
- **Balances anchor truth** — transactions explain movement but never update balances
- **History is write-only audit log** — undo is handled in-memory; on undo, the corresponding history row is silently deleted to keep history accurate. This is an intentional decision: history must reflect what is actually in effect, not what was ever done. No undo entries are appended. Future undo-via-history is still possible because the JSON blob contains enough info to reconstruct reversals
- **No premature caching** — budget balances, owed remaining, available, expense explanation are all computed at query time
- **Dates as TEXT** — YYYY-MM-DD and YYYY-MM stored as text, validated Go-side
- **No lipgloss** — all rendering is in-house using plain string formatting. Bubble Tea is the framework; we build the view layer ourselves
- **TDD-first** — write tests before implementation. unit tests for services, integration tests for repos with real SQLite, model tests for bubbletea Update/View
- **Repository pattern** — repos wrap sqlc-generated queries, services depend on repo interfaces, models depend on service interfaces. Everything mockable at every layer boundary

## Architecture

### Dependency Flow

```
main.go ──creates──> concrete repos (with real DB)
                  ──creates──> concrete services (with repos)
                  ──creates──> root model (with services)

model_test ──uses──> mock services
service_test ──uses──> mock repos
repo_test ──uses──> real SQLite (temp file)
```

### Directory Structure

```
stuf/
├── cmd/
│   └── stuf/
│       └── main.go              # wiring only
├── internal/
│   ├── db/                      # sqlc generated (do not edit)
│   ├── migration/               # goose migration files
│   ├── seed/                    # embedded currency/rate data
│   ├── config/                  # config loading/creation
│   │   └── config_test.go
│   ├── repo/                    # repository interfaces + impls + startup
│   │   ├── account.go
│   │   ├── account_test.go      # integration tests with real SQLite
│   │   ├── balance.go
│   │   ├── balance_test.go
│   │   ├── transaction.go
│   │   ├── transaction_test.go
│   │   ├── tag.go
│   │   ├── tag_test.go
│   │   ├── budget.go
│   │   ├── budget_test.go
│   │   ├── allocation.go
│   │   ├── allocation_test.go
│   │   ├── owed.go
│   │   ├── owed_test.go
│   │   ├── settlement.go
│   │   ├── settlement_test.go
│   │   ├── party.go
│   │   ├── party_test.go
│   │   ├── history.go
│   │   ├── history_test.go
│   │   └── sqlite.go            # DB connection, startup, seeding, constructors
│   ├── service/                 # business logic
│   │   ├── account.go
│   │   ├── account_test.go      # unit tests with mock repos
│   │   ├── money.go             # Money type + arithmetic + conversion
│   │   ├── money_test.go
│   │   ├── formula.go           # formula parsing/evaluation
│   │   ├── formula_test.go
│   │   ├── currency.go
│   │   ├── currency_test.go
│   │   ├── history.go
│   │   ├── history_test.go
│   │   └── ...                  # one per domain
│   ├── model/                   # bubbletea models
│   │   ├── app.go               # root model, routing
│   │   ├── app_test.go
│   │   ├── dashboard.go
│   │   ├── accounts.go
│   │   └── ...
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

## Phase 1: Project Scaffolding

### Dependencies

- `github.com/charmbracelet/bubbletea` — TUI framework
- `modernc.org/sqlite` — pure-Go SQLite driver
- `github.com/pressly/goose/v3` — migrations
- `github.com/sqlc-dev/sqlc` — query generation (dev dependency)
- Standard library for JSON, formatting, etc.

### Makefile Targets

- `make generate` — run sqlc generate
- `make migrate` — run goose migrations
- `make run` — build and run
- `make build` — build binary
- `make test` — run tests

## Phase 2: Database Schema

### Migration Strategy

- Numbered goose migrations (`00001_init_schema.sql`, etc.)
- On startup: verify `db.sqlite` exists (create if not), verify it's a stuf database via `app_meta` table, run pending migrations, validate schema, seed missing currency/rate data
- Embedded migration files and seed data in binary via `embed`

### SQLite Notes

- All integer types are `INTEGER` (SQLite does not differentiate BIGINT/INT)
- Booleans stored as `INTEGER` (1=true, 0=false)
- Dates stored as `TEXT` (YYYY-MM-DD, YYYY-MM, ISO 8601 datetimes), validated Go-side
- Refs derived from auto-increment ID: `tx-000001`, `owed-000001`, `set-000001`

### Tables

#### `app_meta`

| Column | Type | Constraints | Notes |
|--------|------|-------------|-------|
| key | TEXT | PRIMARY KEY | |
| value | TEXT | | |

Verifies "this is a stuf database" and tracks app version.

#### `currencies`

| Column | Type | Constraints | Notes |
|--------|------|-------------|-------|
| id | INTEGER | PRIMARY KEY | |
| code | TEXT | UNIQUE NOT NULL | ISO 4217-like (USD, HKD) |
| name | TEXT | NOT NULL | Full name (US Dollar) |
| scale | INTEGER | NOT NULL DEFAULT 2 | Decimal places |
| created_at | TEXT | NOT NULL | ISO 8601 |
| updated_at | TEXT | NOT NULL | |

Seeded from embedded data. Not user-creatable in v1.

#### `currency_rates`

| Column | Type | Constraints | Notes |
|--------|------|-------------|-------|
| id | INTEGER | PRIMARY KEY | |
| currency_id | INTEGER | UNIQUE REFERENCES currencies(id) | One rate per currency |
| rate_to_usd_amount | INTEGER | NOT NULL | e.g. HKD: 781 → 1 HKD ≈ 0.128 USD |
| rate_to_usd_scale | INTEGER | NOT NULL DEFAULT 2 | Scale for the rate |
| updated_at | TEXT | NOT NULL | |

Rate stored as integer+scale (same money pattern). To convert HKD to USD: `amount * rate_to_usd_amount / (10^rate_to_usd_scale * 10^from_scale)`. USD itself has rate 1/1 (amount=1, scale=0).

#### `accounts`

| Column | Type | Constraints | Notes |
|--------|------|-------------|-------|
| id | INTEGER | PRIMARY KEY | Internal, immutable |
| name | TEXT | UNIQUE NOT NULL | User-facing slug, editable |
| currency_id | INTEGER | REFERENCES currencies(id) NOT NULL | Locked if balances exist |
| on_budget | INTEGER | NOT NULL DEFAULT 1 | 1=true, 0=false |
| hidden | INTEGER | NOT NULL DEFAULT 0 | |
| notes | TEXT | NOT NULL DEFAULT '' | |
| created_at | TEXT | NOT NULL | |
| updated_at | TEXT | NOT NULL | |

#### `balances`

| Column | Type | Constraints | Notes |
|--------|------|-------------|-------|
| id | INTEGER | PRIMARY KEY | |
| account_id | INTEGER | REFERENCES accounts(id) NOT NULL | |
| date | TEXT | NOT NULL | YYYY-MM-DD |
| amount | INTEGER | NOT NULL | e.g. 5000000 = HKD 50,000.00 |
| scale | INTEGER | NOT NULL DEFAULT 2 | |
| notes | TEXT | NOT NULL DEFAULT '' | |
| created_at | TEXT | NOT NULL | |
| updated_at | TEXT | NOT NULL | |

**UNIQUE(account_id, date)**. **INDEX(account_id, date)**.

#### `tags`

| Column | Type | Constraints | Notes |
|--------|------|-------------|-------|
| id | INTEGER | PRIMARY KEY | Internal, immutable |
| name | TEXT | UNIQUE NOT NULL | Strict slug |
| notes | TEXT | NOT NULL DEFAULT '' | |
| created_at | TEXT | NOT NULL | |
| updated_at | TEXT | NOT NULL | |

#### `budget_categories`

| Column | Type | Constraints | Notes |
|--------|------|-------------|-------|
| id | INTEGER | PRIMARY KEY | |
| name | TEXT | UNIQUE NOT NULL | Strict slug |
| notes | TEXT | NOT NULL DEFAULT '' | |
| created_at | TEXT | NOT NULL | |
| updated_at | TEXT | NOT NULL | |

Seed `uncategorized` (id=1, cannot be renamed or deleted).

#### `budgets`

| Column | Type | Constraints | Notes |
|--------|------|-------------|-------|
| id | INTEGER | PRIMARY KEY | |
| name | TEXT | UNIQUE NOT NULL | Strict slug |
| currency_id | INTEGER | REFERENCES currencies(id) NOT NULL | Locked if allocations/linked txns exist |
| category_id | INTEGER | REFERENCES budget_categories(id) NOT NULL DEFAULT 1 | |
| has_default_allocation | INTEGER | NOT NULL DEFAULT 0 | |
| default_allocation_amount | INTEGER | | Nullable, set when has_default_allocation=1 |
| default_allocation_scale | INTEGER | | Nullable |
| has_goal | INTEGER | NOT NULL DEFAULT 0 | |
| goal_target_amount | INTEGER | | Nullable, set when has_goal=1 |
| goal_target_scale | INTEGER | | Nullable |
| goal_target_month | TEXT | | YYYY-MM, nullable |
| hidden | INTEGER | NOT NULL DEFAULT 0 | |
| notes | TEXT | NOT NULL DEFAULT '' | |
| created_at | TEXT | NOT NULL | |
| updated_at | TEXT | NOT NULL | |

Currency for allocations/goals inherits from budget. No separate currency columns on those tables.

#### `allocations`

| Column | Type | Constraints | Notes |
|--------|------|-------------|-------|
| id | INTEGER | PRIMARY KEY | |
| budget_id | INTEGER | REFERENCES budgets(id) NOT NULL | |
| date | TEXT | NOT NULL | YYYY-MM-DD |
| delta_amount | INTEGER | NOT NULL | Can be negative |
| delta_scale | INTEGER | NOT NULL DEFAULT 2 | |
| notes | TEXT | NOT NULL DEFAULT '' | |
| created_at | TEXT | NOT NULL | |
| updated_at | TEXT | NOT NULL | |

Budget balance = SUM(delta_amount) filtered by budget_id. **INDEX(budget_id, date)**.

#### `transactions`

| Column | Type | Constraints | Notes |
|--------|------|-------------|-------|
| id | INTEGER | PRIMARY KEY | Internal |
| ref | TEXT | UNIQUE NOT NULL | Derived: tx-{zero-padded id} |
| date | TEXT | NOT NULL | YYYY-MM-DD |
| type | TEXT | NOT NULL CHECK(type IN ('income', 'expense')) | |
| amount | INTEGER | NOT NULL | Always positive |
| scale | INTEGER | NOT NULL DEFAULT 2 | |
| currency_id | INTEGER | REFERENCES currencies(id) NOT NULL | |
| account_id | INTEGER | REFERENCES accounts(id) NOT NULL | |
| parent_id | INTEGER | REFERENCES transactions(id) | Nullable, self-ref |
| budget_id | INTEGER | REFERENCES budgets(id) | Nullable, expense-only in v1 |
| notes | TEXT | NOT NULL DEFAULT '' | |
| created_at | TEXT | NOT NULL | |
| updated_at | TEXT | NOT NULL | |

**INDEX(account_id, date)**. **INDEX(parent_id)**. **INDEX(budget_id)**.

#### `transaction_tags`

| Column | Type | Constraints | Notes |
|--------|------|-------------|-------|
| transaction_id | INTEGER | REFERENCES transactions(id) NOT NULL | |
| tag_id | INTEGER | REFERENCES tags(id) NOT NULL | |

**PRIMARY KEY(transaction_id, tag_id)**.

#### `parties`

| Column | Type | Constraints | Notes |
|--------|------|-------------|-------|
| id | INTEGER | PRIMARY KEY | Internal, immutable |
| name | TEXT | UNIQUE NOT NULL | Strict slug, editable |
| notes | TEXT | NOT NULL DEFAULT '' | |
| created_at | TEXT | NOT NULL | |
| updated_at | TEXT | NOT NULL | |

#### `owed_items`

| Column | Type | Constraints | Notes |
|--------|------|-------------|-------|
| id | INTEGER | PRIMARY KEY | Internal |
| ref | TEXT | UNIQUE NOT NULL | Derived: owed-{zero-padded id} |
| direction | TEXT | NOT NULL CHECK(direction IN ('you_owe', 'owes_you')) | |
| party_id | INTEGER | REFERENCES parties(id) NOT NULL | |
| date | TEXT | NOT NULL | YYYY-MM-DD |
| amount | INTEGER | NOT NULL | Computed from formula or direct input |
| scale | INTEGER | NOT NULL DEFAULT 2 | |
| formula | TEXT | | Raw formula, e.g. =1000/2. Nullable |
| currency_id | INTEGER | REFERENCES currencies(id) NOT NULL | |
| notes | TEXT | NOT NULL DEFAULT '' | |
| created_at | TEXT | NOT NULL | |
| updated_at | TEXT | NOT NULL | |

Remaining = amount - SUM(settlements converted to owed item currency). Computed at query time.

#### `settlements`

| Column | Type | Constraints | Notes |
|--------|------|-------------|-------|
| id | INTEGER | PRIMARY KEY | Internal |
| ref | TEXT | UNIQUE NOT NULL | Derived: set-{zero-padded id} |
| owed_item_id | INTEGER | REFERENCES owed_items(id) NOT NULL | |
| date | TEXT | NOT NULL | YYYY-MM-DD |
| amount | INTEGER | NOT NULL | In settlement currency |
| scale | INTEGER | NOT NULL DEFAULT 2 | |
| currency_id | INTEGER | REFERENCES currencies(id) NOT NULL | Defaults to owed item currency |
| notes | TEXT | NOT NULL DEFAULT '' | |
| created_at | TEXT | NOT NULL | |
| updated_at | TEXT | NOT NULL | |

**INDEX(owed_item_id, date)**.

#### `history`

| Column | Type | Constraints | Notes |
|--------|------|-------------|-------|
| id | INTEGER | PRIMARY KEY | |
| timestamp | TEXT | NOT NULL | ISO 8601 |
| action | TEXT | NOT NULL | 'create', 'update', 'delete' |
| path | TEXT | NOT NULL | e.g. /accounts/hsbc-one |
| old_data | TEXT | | JSON, null for creates |
| new_data | TEXT | | JSON, null for deletes |

Write-only audit log. On undo during a session: reverse the DB mutation, then **silently delete** the corresponding history row. This is intentional — history must reflect what is actually in effect, not what was ever done. **INDEX(timestamp)**.

### Data Relationships

```
currencies ←────────────────────────────────────────┐
    │                                                 │
    ├── currency_rates                                │
    │                                                 │
    ├── accounts                                      │
    │     ├── balances                                │
    │     └── transactions (by account_id)             │
    │           └── transaction_tags ←── tags          │
    │                                                 │
    ├── budget_categories                             │
    │     └── budgets                                 │
    │           ├── allocations                        │
    │           └── transactions (by budget_id)        │
    │                                                 │
    └── owed_items ←── parties                        │
          └── settlements
```

### Computed Values (Not Stored)

| Value | Computation |
|-------|-------------|
| Account current balance | Latest balance entry for that account |
| Budget balance | SUM(allocations delta_amount) by budget_id |
| Budget spent | SUM of effective expense transaction amounts linked to budget |
| Budget available | on-budget balances - budgeted - open you-owe remaining |
| Owed remaining | owed amount - SUM(settlements converted to owed item currency) |
| Report growth | end balance - start balance for period |
| Report income | SUM of effective income rows in period, or growth if none |
| Report expenses | income - growth (derived), explained by transactions |
| Parent remaining | parent amount - SUM(child amounts converted to parent currency) |

## Phase 3: Money Type

### `money.Money` Type

```go
type Money struct {
    Amount int64
    Scale  int
}
```

Methods: `Add`, `Sub`, `Negate`, `ConvertToScale(newScale)`, `Equals`, `IsZero`, `IsPositive`, `IsNegative`, `Format(currencyCode)`, `Parse(input)`. All arithmetic validates scale compatibility or converts.

### `money.CurrencyRate`

Used for cross-currency conversion: `Convert(rate Money, targetScale int) Money`.

### Formula Parser

```go
func ParseFormula(input string) (int64, int, error)
```

Supports: `=1000/2`, `=500+200`, `=100*3-50`, parentheses, decimals. Returns computed amount + scale. Invalid formulas return errors.

## Phase 4: App Startup

### Startup Sequence

1. Check for `db.sqlite` in current working directory
2. If not found, create it and run all migrations
3. If found, verify it's a valid SQLite file
4. Verify it's a stuf database via `app_meta` table
5. Run pending migrations
6. Validate required schema
7. Seed missing currency/rate data from embedded data
8. Load config (local `config.jsonc` in cwd, fallback to global `~/.config/stuf/config.jsonc`)
9. If no config, create global config with detected or default USD currency
10. Start bubbletea program

### Config Structure

```jsonc
{
  // stuf config
  "currency": "HKD"
}
```

Minimal for v1. Just app currency. Date format is fixed ISO.

## Phase 5: TUI Shell (Minimal)

The initial TUI shell just proves the app boots, connects to DB, and shows the dashboard structure. Real feature screens come in subsequent plans.

- Bubbletea `Model` struct holds: current URL/path, DB connection, config, undo stack, visible session history
- URL-based routing: `/`, `/accounts/`, `/accounts/create/`, etc.
- Global keybinds: `ctrl-c` quit, `ctrl-z` undo, `esc` back/exit, `?` help
- Number hotkeys work only in menu screens, not in forms
- All rendering via in-house string formatting, no lipgloss

## Phase 6: Config & Seeding — Test Coverage

### Config Tests

- Parse valid config
- Reject invalid config with clear error
- Create default config when none exists
- Location detection fallback to USD

### Seeding Tests

- Fresh DB has all expected currencies
- Re-running seeding is idempotent (no duplicates)
- Currency rates are seeded correctly
- USD rate is 1:1

## Execution Order

1. **`go.mod` + directory structure** — initialize module, create scaffolding
2. **Currency seed data** — embed JSON of common currencies and USD rates
3. **`money` package** — Money type, arithmetic, conversion, formatting, formula parser. **TDD: write tests first**
4. **`formula` package** — formula parsing/evaluation. **TDD: write tests first**
5. **Goose migration 00001** — all tables, indexes, constraints
6. **`repo` package** — repository interfaces and SQLite implementations. **TDD: integration tests with real SQLite temp files**
7. **App startup logic** — DB init, config, seeding. **TDD**
8. **sqlc config + queries** — generate query code
9. **`service` package** — business logic per domain. **TDD: unit tests with mock repos**
10. **Bubbletea shell** — boot, connect, dashboard render, nav framework. **TDD: model tests**
11. **First feature: accounts** — prove the full stack works end-to-end. **TDD**