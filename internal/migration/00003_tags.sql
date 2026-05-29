-- +goose Up
CREATE TABLE IF NOT EXISTS tags (
  id INTEGER PRIMARY KEY,
  name TEXT UNIQUE NOT NULL,
  notes TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS account_tags (
  account_id INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
  tag_id INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
  created_at TEXT NOT NULL,
  PRIMARY KEY(account_id, tag_id)
);

CREATE INDEX IF NOT EXISTS idx_account_tags_tag_id ON account_tags(tag_id);

-- +goose Down
DROP INDEX IF EXISTS idx_account_tags_tag_id;
DROP TABLE IF EXISTS account_tags;
DROP TABLE IF EXISTS tags;
