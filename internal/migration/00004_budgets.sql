-- +goose Up
CREATE TABLE IF NOT EXISTS budget_categories (
  id INTEGER PRIMARY KEY,
  name TEXT UNIQUE NOT NULL,
  notes TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

INSERT OR IGNORE INTO budget_categories (name, notes, created_at, updated_at)
VALUES ('uncategorized', 'default category', '1970-01-01T00:00:00Z', '1970-01-01T00:00:00Z');

CREATE TABLE IF NOT EXISTS budgets (
  id INTEGER PRIMARY KEY,
  name TEXT UNIQUE NOT NULL,
  currency_id INTEGER NOT NULL REFERENCES currencies(id),
  category_id INTEGER NOT NULL REFERENCES budget_categories(id),
  hidden INTEGER NOT NULL DEFAULT 0,
  notes TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_budgets_category_id ON budgets(category_id);
CREATE INDEX IF NOT EXISTS idx_budgets_hidden ON budgets(hidden);

CREATE TABLE IF NOT EXISTS budget_allocations (
  id INTEGER PRIMARY KEY,
  budget_id INTEGER NOT NULL REFERENCES budgets(id),
  date TEXT NOT NULL,
  amount INTEGER NOT NULL,
  scale INTEGER NOT NULL DEFAULT 2,
  notes TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_budget_allocations_budget_order ON budget_allocations(budget_id, date, created_at, id);

-- +goose Down
DROP INDEX IF EXISTS idx_budget_allocations_budget_order;
DROP TABLE IF EXISTS budget_allocations;
DROP INDEX IF EXISTS idx_budgets_hidden;
DROP INDEX IF EXISTS idx_budgets_category_id;
DROP TABLE IF EXISTS budgets;
DROP TABLE IF EXISTS budget_categories;
