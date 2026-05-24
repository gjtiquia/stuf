-- +goose Up
CREATE TABLE IF NOT EXISTS app_meta (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);

INSERT OR IGNORE INTO app_meta(key, value) VALUES ('app', 'stuf');
INSERT OR IGNORE INTO app_meta(key, value) VALUES ('schema', 'foundation');

CREATE TABLE IF NOT EXISTS currencies (
  id INTEGER PRIMARY KEY,
  code TEXT UNIQUE NOT NULL,
  name TEXT NOT NULL,
  scale INTEGER NOT NULL DEFAULT 2,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS currency_rates (
  id INTEGER PRIMARY KEY,
  currency_id INTEGER UNIQUE NOT NULL REFERENCES currencies(id),
  rate_to_usd_amount INTEGER NOT NULL,
  rate_to_usd_scale INTEGER NOT NULL DEFAULT 3,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS accounts (
  id INTEGER PRIMARY KEY,
  name TEXT UNIQUE NOT NULL,
  currency_id INTEGER NOT NULL REFERENCES currencies(id),
  on_budget INTEGER NOT NULL DEFAULT 1,
  hidden INTEGER NOT NULL DEFAULT 0,
  notes TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS balances (
  id INTEGER PRIMARY KEY,
  account_id INTEGER NOT NULL REFERENCES accounts(id),
  date TEXT NOT NULL,
  amount INTEGER NOT NULL,
  scale INTEGER NOT NULL DEFAULT 2,
  notes TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE(account_id, date)
);

CREATE INDEX IF NOT EXISTS idx_balances_account_date ON balances(account_id, date);

CREATE TABLE IF NOT EXISTS history (
  id INTEGER PRIMARY KEY,
  timestamp TEXT NOT NULL,
  action TEXT NOT NULL CHECK(action IN ('create', 'add', 'edit', 'delete')),
  path TEXT NOT NULL,
  old_data TEXT,
  new_data TEXT
);

CREATE INDEX IF NOT EXISTS idx_history_timestamp ON history(timestamp);

-- +goose Down
DROP TABLE IF EXISTS history;
DROP TABLE IF EXISTS balances;
DROP TABLE IF EXISTS accounts;
DROP TABLE IF EXISTS currency_rates;
DROP TABLE IF EXISTS currencies;
DROP TABLE IF EXISTS app_meta;
