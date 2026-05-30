-- +goose Up
CREATE TABLE IF NOT EXISTS transactions (
  id INTEGER PRIMARY KEY,
  ref INTEGER UNIQUE NOT NULL,
  parent_id INTEGER REFERENCES transactions(id),
  account_id INTEGER NOT NULL REFERENCES accounts(id),
  type TEXT NOT NULL CHECK(type IN ('income', 'expense')),
  currency_id INTEGER NOT NULL REFERENCES currencies(id),
  date TEXT NOT NULL,
  amount INTEGER NOT NULL CHECK(amount >= 0),
  scale INTEGER NOT NULL DEFAULT 2,
  notes TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_transactions_parent_id ON transactions(parent_id);
CREATE INDEX IF NOT EXISTS idx_transactions_account_date ON transactions(account_id, date, created_at, id);
CREATE INDEX IF NOT EXISTS idx_transactions_type ON transactions(type);
CREATE INDEX IF NOT EXISTS idx_transactions_currency_id ON transactions(currency_id);

CREATE TABLE IF NOT EXISTS transaction_tags (
  transaction_id INTEGER NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
  tag_id INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
  created_at TEXT NOT NULL,
  PRIMARY KEY(transaction_id, tag_id)
);

CREATE INDEX IF NOT EXISTS idx_transaction_tags_tag_id ON transaction_tags(tag_id);

-- +goose Down
DROP INDEX IF EXISTS idx_transaction_tags_tag_id;
DROP TABLE IF EXISTS transaction_tags;
DROP INDEX IF EXISTS idx_transactions_currency_id;
DROP INDEX IF EXISTS idx_transactions_type;
DROP INDEX IF EXISTS idx_transactions_account_date;
DROP INDEX IF EXISTS idx_transactions_parent_id;
DROP TABLE IF EXISTS transactions;
