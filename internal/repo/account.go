package repo

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type AccountRepo struct{ store *Store }

type AccountCreate struct {
	Name       string
	CurrencyID int64
	OnBudget   bool
	Notes      string
}

func (r *AccountRepo) Create(ctx context.Context, in AccountCreate) (Account, error) {
	now := r.store.Clock().UTC().Format(time.RFC3339)
	res, err := r.store.DB.ExecContext(ctx, `INSERT INTO accounts(name, currency_id, on_budget, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`, in.Name, in.CurrencyID, boolInt(in.OnBudget), in.Notes, now, now)
	if err != nil {
		return Account{}, err
	}
	id, _ := res.LastInsertId()
	return r.GetByID(ctx, id)
}

func (r *AccountRepo) GetByID(ctx context.Context, id int64) (Account, error) {
	return scanAccount(r.store.DB.QueryRowContext(ctx, accountSelect+" WHERE a.id=?", id))
}

func (r *AccountRepo) GetByName(ctx context.Context, name string) (Account, error) {
	return scanAccount(r.store.DB.QueryRowContext(ctx, accountSelect+" WHERE a.name=?", name))
}

func (r *AccountRepo) List(ctx context.Context, includeHidden bool) ([]Account, error) {
	q := accountSelect
	if !includeHidden {
		q += " WHERE a.hidden=0"
	}
	q += " ORDER BY a.name"
	rows, err := r.store.DB.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Account
	for rows.Next() {
		a, err := scanAccount(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *AccountRepo) Update(ctx context.Context, a Account) (Account, error) {
	now := r.store.Clock().UTC().Format(time.RFC3339)
	_, err := r.store.DB.ExecContext(ctx, `UPDATE accounts SET name=?, currency_id=?, on_budget=?, hidden=?, notes=?, updated_at=? WHERE id=?`,
		a.Name, a.CurrencyID, boolInt(a.OnBudget), boolInt(a.Hidden), a.Notes, now, a.ID)
	if err != nil {
		return Account{}, err
	}
	return r.GetByID(ctx, a.ID)
}

func (r *AccountRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.store.DB.ExecContext(ctx, "DELETE FROM accounts WHERE id=?", id)
	return err
}

func (r *AccountRepo) HasBalances(ctx context.Context, id int64) (bool, error) {
	var n int
	if err := r.store.DB.QueryRowContext(ctx, "SELECT count(*) FROM balances WHERE account_id=?", id).Scan(&n); err != nil {
		return false, err
	}
	return n > 0, nil
}

const accountSelect = `SELECT a.id, a.name, a.currency_id, c.code, c.scale, a.on_budget, a.hidden, a.notes, a.created_at, a.updated_at
	FROM accounts a JOIN currencies c ON c.id = a.currency_id`

type accountScanner interface{ Scan(dest ...any) error }

func scanAccount(row accountScanner) (Account, error) {
	var a Account
	var onBudget, hidden int
	err := row.Scan(&a.ID, &a.Name, &a.CurrencyID, &a.Code, &a.Scale, &onBudget, &hidden, &a.Notes, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return Account{}, fmt.Errorf("account not found")
		}
		return Account{}, err
	}
	a.OnBudget = onBudget == 1
	a.Hidden = hidden == 1
	return a, nil
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
