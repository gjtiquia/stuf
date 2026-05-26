-- +goose Up
ALTER TABLE accounts ADD COLUMN parent_id INTEGER REFERENCES accounts(id);

CREATE INDEX IF NOT EXISTS idx_accounts_parent_id ON accounts(parent_id);

-- +goose Down
DROP INDEX IF EXISTS idx_accounts_parent_id;

