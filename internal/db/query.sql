-- Startup / meta

-- name: GetAppMetaApp :one
SELECT value FROM app_meta WHERE key = 'app';

-- Currency seeding

-- name: UpsertCurrency :exec
INSERT INTO currencies (code, name, scale, created_at, updated_at)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT (code) DO UPDATE SET
  name = excluded.name,
  scale = excluded.scale,
  updated_at = excluded.updated_at;

-- name: GetCurrencyIDByCode :one
SELECT id FROM currencies WHERE code = ?;

-- name: UpsertCurrencyRate :exec
INSERT INTO currency_rates (currency_id, rate_to_usd_amount, rate_to_usd_scale, updated_at)
VALUES (?, ?, ?, ?)
ON CONFLICT (currency_id) DO UPDATE SET
  rate_to_usd_amount = excluded.rate_to_usd_amount,
  rate_to_usd_scale = excluded.rate_to_usd_scale,
  updated_at = excluded.updated_at;

-- name: UpsertCurrencyNameOnly :exec
INSERT INTO currencies (code, name, scale, created_at, updated_at)
VALUES (?, ?, 2, ?, ?)
ON CONFLICT (code) DO UPDATE SET name = excluded.name;

-- Currencies

-- name: GetCurrencyByCode :one
SELECT
  c.id,
  c.code,
  c.name,
  c.scale,
  cr.rate_to_usd_amount,
  cr.rate_to_usd_scale,
  cr.updated_at
FROM currencies c
LEFT JOIN currency_rates cr ON cr.currency_id = c.id
WHERE c.code = ?;

-- name: GetCurrencyByID :one
SELECT
  c.id,
  c.code,
  c.name,
  c.scale,
  cr.rate_to_usd_amount,
  cr.rate_to_usd_scale,
  cr.updated_at
FROM currencies c
LEFT JOIN currency_rates cr ON cr.currency_id = c.id
WHERE c.id = ?;

-- name: ListCurrencies :many
SELECT
  c.id,
  c.code,
  c.name,
  c.scale,
  cr.rate_to_usd_amount,
  cr.rate_to_usd_scale,
  cr.updated_at
FROM currencies c
LEFT JOIN currency_rates cr ON cr.currency_id = c.id
ORDER BY c.code;

-- Accounts

-- name: CreateAccount :execresult
INSERT INTO accounts (name, currency_id, on_budget, hidden, notes, parent_id, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetAccountByID :one
SELECT
  a.id,
  a.name,
  a.currency_id,
  a.parent_id,
  c.code,
  c.scale,
  a.on_budget,
  a.hidden,
  a.notes,
  a.created_at,
  a.updated_at
FROM accounts a
JOIN currencies c ON c.id = a.currency_id
WHERE a.id = ?;

-- name: GetAccountByName :one
SELECT
  a.id,
  a.name,
  a.currency_id,
  a.parent_id,
  c.code,
  c.scale,
  a.on_budget,
  a.hidden,
  a.notes,
  a.created_at,
  a.updated_at
FROM accounts a
JOIN currencies c ON c.id = a.currency_id
WHERE a.name = ?;

-- name: ListAccounts :many
SELECT
  a.id,
  a.name,
  a.currency_id,
  a.parent_id,
  c.code,
  c.scale,
  a.on_budget,
  a.hidden,
  a.notes,
  a.created_at,
  a.updated_at
FROM accounts a
JOIN currencies c ON c.id = a.currency_id
ORDER BY a.name;

-- name: ListVisibleAccounts :many
SELECT
  a.id,
  a.name,
  a.currency_id,
  a.parent_id,
  c.code,
  c.scale,
  a.on_budget,
  a.hidden,
  a.notes,
  a.created_at,
  a.updated_at
FROM accounts a
JOIN currencies c ON c.id = a.currency_id
WHERE a.hidden = 0
ORDER BY a.name;

-- name: UpdateAccount :exec
UPDATE accounts
SET name = ?, currency_id = ?, on_budget = ?, hidden = ?, notes = ?, parent_id = ?, updated_at = ?
WHERE id = ?;

-- name: DeleteAccount :exec
DELETE FROM accounts WHERE id = ?;

-- name: CountBalancesByAccountID :one
SELECT count(*) FROM balances WHERE account_id = ?;

-- name: CountChildrenByAccountID :one
SELECT count(*) FROM accounts WHERE parent_id = ?;

-- Tags

-- name: CreateTag :execresult
INSERT INTO tags (name, notes, created_at, updated_at)
VALUES (?, ?, ?, ?);

-- name: GetTagByID :one
SELECT id, name, notes, created_at, updated_at
FROM tags
WHERE id = ?;

-- name: GetTagByName :one
SELECT id, name, notes, created_at, updated_at
FROM tags
WHERE name = ?;

-- name: ListTags :many
SELECT id, name, notes, created_at, updated_at
FROM tags
ORDER BY name;

-- name: UpdateTag :exec
UPDATE tags
SET name = ?, notes = ?, updated_at = ?
WHERE id = ?;

-- name: DeleteTag :exec
DELETE FROM tags WHERE id = ?;

-- name: DeleteAccountTagsByAccountID :exec
DELETE FROM account_tags WHERE account_id = ?;

-- name: AddAccountTag :exec
INSERT OR IGNORE INTO account_tags (account_id, tag_id, created_at)
VALUES (?, ?, ?);

-- name: ListTagsByAccountID :many
SELECT t.id, t.name, t.notes, t.created_at, t.updated_at
FROM tags t
JOIN account_tags at ON at.tag_id = t.id
WHERE at.account_id = ?
ORDER BY t.name;

-- name: ListEffectiveTagsByAccountID :many
WITH RECURSIVE ancestors(id, parent_id) AS (
  SELECT accounts.id, accounts.parent_id FROM accounts WHERE accounts.id = ?
  UNION ALL
  SELECT a.id, a.parent_id FROM accounts a JOIN ancestors x ON x.parent_id = a.id
)
SELECT DISTINCT t.id, t.name, t.notes, t.created_at, t.updated_at
FROM tags t
JOIN account_tags at ON at.tag_id = t.id
JOIN ancestors x ON x.id = at.account_id
ORDER BY t.name;

-- name: CountAccountTagsByTagID :one
SELECT count(*)
FROM account_tags
WHERE tag_id = ?;

-- name: ListRootAccounts :many
SELECT
  a.id,
  a.name,
  a.currency_id,
  a.parent_id,
  c.code,
  c.scale,
  a.on_budget,
  a.hidden,
  a.notes,
  a.created_at,
  a.updated_at
FROM accounts a
JOIN currencies c ON c.id = a.currency_id
WHERE a.parent_id IS NULL
ORDER BY a.name;

-- name: ListVisibleRootAccounts :many
SELECT
  a.id,
  a.name,
  a.currency_id,
  a.parent_id,
  c.code,
  c.scale,
  a.on_budget,
  a.hidden,
  a.notes,
  a.created_at,
  a.updated_at
FROM accounts a
JOIN currencies c ON c.id = a.currency_id
WHERE a.parent_id IS NULL AND a.hidden = 0
ORDER BY a.name;

-- name: ListChildAccounts :many
SELECT
  a.id,
  a.name,
  a.currency_id,
  a.parent_id,
  c.code,
  c.scale,
  a.on_budget,
  a.hidden,
  a.notes,
  a.created_at,
  a.updated_at
FROM accounts a
JOIN currencies c ON c.id = a.currency_id
WHERE a.parent_id = ?
ORDER BY a.name;

-- name: ListVisibleChildAccounts :many
SELECT
  a.id,
  a.name,
  a.currency_id,
  a.parent_id,
  c.code,
  c.scale,
  a.on_budget,
  a.hidden,
  a.notes,
  a.created_at,
  a.updated_at
FROM accounts a
JOIN currencies c ON c.id = a.currency_id
WHERE a.parent_id = ? AND a.hidden = 0
ORDER BY a.name;

-- name: ListDescendantAccounts :many
WITH RECURSIVE descendants(id) AS (
  SELECT accounts.id FROM accounts WHERE accounts.parent_id = ?
  UNION ALL
  SELECT a.id FROM accounts a JOIN descendants d ON a.parent_id = d.id
)
SELECT
  a.id,
  a.name,
  a.currency_id,
  a.parent_id,
  c.code,
  c.scale,
  a.on_budget,
  a.hidden,
  a.notes,
  a.created_at,
  a.updated_at
FROM accounts a
JOIN descendants d ON d.id = a.id
JOIN currencies c ON c.id = a.currency_id
ORDER BY a.name;

-- Balances

-- name: CreateBalance :execresult
INSERT INTO balances (account_id, date, amount, scale, notes, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: GetBalanceByID :one
SELECT id, account_id, date, amount, scale, notes, created_at, updated_at
FROM balances
WHERE id = ?;

-- name: GetBalanceByAccountDate :one
SELECT id, account_id, date, amount, scale, notes, created_at, updated_at
FROM balances
WHERE account_id = ? AND date = ?;

-- name: ListBalancesByAccount :many
SELECT id, account_id, date, amount, scale, notes, created_at, updated_at
FROM balances
WHERE account_id = ?
ORDER BY date DESC;

-- name: GetLatestBalanceByAccount :one
SELECT id, account_id, date, amount, scale, notes, created_at, updated_at
FROM balances
WHERE account_id = ?
ORDER BY date DESC
LIMIT 1;

-- name: ListAllVisibleBalances :many
SELECT
  a.id AS account_id,
  a.name AS account_name,
  a.currency_id,
  a.parent_id,
  c.code,
  c.scale,
  a.on_budget,
  a.hidden,
  a.notes AS account_notes,
  a.created_at AS account_created_at,
  a.updated_at AS account_updated_at,
  b.id AS balance_id,
  b.account_id AS balance_account_id,
  b.date,
  b.amount,
  b.scale AS balance_scale,
  b.notes AS balance_notes,
  b.created_at AS balance_created_at,
  b.updated_at AS balance_updated_at
FROM accounts a
JOIN currencies c ON c.id = a.currency_id
JOIN balances b ON b.account_id = a.id
WHERE a.hidden = 0
ORDER BY a.id, b.date;

-- name: UpdateBalance :exec
UPDATE balances
SET date = ?, amount = ?, scale = ?, notes = ?, updated_at = ?
WHERE id = ?;

-- name: DeleteBalance :exec
DELETE FROM balances WHERE id = ?;

-- History

-- name: CreateHistory :execresult
INSERT INTO history (timestamp, action, path, old_data, new_data)
VALUES (?, ?, ?, ?, ?);

-- name: ListHistory :many
SELECT id, timestamp, action, path, old_data, new_data
FROM history
ORDER BY timestamp, id;

-- name: DeleteHistory :exec
DELETE FROM history WHERE id = ?;
