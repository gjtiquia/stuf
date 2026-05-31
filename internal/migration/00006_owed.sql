-- +goose Up
CREATE TABLE IF NOT EXISTS owed_ledgers (
  id INTEGER PRIMARY KEY,
  name TEXT UNIQUE NOT NULL,
  currency_id INTEGER NOT NULL REFERENCES currencies(id),
  notes TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS owed_transactions (
  id INTEGER PRIMARY KEY,
  ledger_id INTEGER NOT NULL REFERENCES owed_ledgers(id),
  date TEXT NOT NULL,
  currency_id INTEGER NOT NULL REFERENCES currencies(id),
  amount INTEGER NOT NULL,
  scale INTEGER NOT NULL DEFAULT 2,
  formula TEXT NOT NULL DEFAULT '',
  notes TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_owed_transactions_ledger_order ON owed_transactions(ledger_id, date, created_at, id);
CREATE INDEX IF NOT EXISTS idx_owed_transactions_currency_id ON owed_transactions(currency_id);

-- +goose Down
DROP INDEX IF EXISTS idx_owed_transactions_currency_id;
DROP INDEX IF EXISTS idx_owed_transactions_ledger_order;
DROP TABLE IF EXISTS owed_transactions;
DROP TABLE IF EXISTS owed_ledgers;
