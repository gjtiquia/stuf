package repo

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"stuf/internal/money"
)

type BalanceRepo struct{ store *Store }

type BalanceCreate struct {
	AccountID int64
	Date      string
	Amount    money.Money
	Notes     string
}

func (r *BalanceRepo) Create(ctx context.Context, in BalanceCreate) (Balance, error) {
	now := r.store.Clock().UTC().Format(time.RFC3339)
	res, err := r.store.DB.ExecContext(ctx, `INSERT INTO balances(account_id, date, amount, scale, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, in.AccountID, in.Date, in.Amount.Amount, in.Amount.Scale, in.Notes, now, now)
	if err != nil {
		return Balance{}, err
	}
	id, _ := res.LastInsertId()
	return r.GetByID(ctx, id)
}

func (r *BalanceRepo) GetByID(ctx context.Context, id int64) (Balance, error) {
	return scanBalance(r.store.DB.QueryRowContext(ctx, balanceSelect+" WHERE id=?", id))
}

func (r *BalanceRepo) GetByAccountDate(ctx context.Context, accountID int64, date string) (Balance, error) {
	return scanBalance(r.store.DB.QueryRowContext(ctx, balanceSelect+" WHERE account_id=? AND date=?", accountID, date))
}

func (r *BalanceRepo) ListByAccount(ctx context.Context, accountID int64) ([]Balance, error) {
	rows, err := r.store.DB.QueryContext(ctx, balanceSelect+" WHERE account_id=? ORDER BY date DESC", accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBalances(rows)
}

func (r *BalanceRepo) ListAllVisible(ctx context.Context) ([]struct {
	Account Account
	Balance Balance
}, error) {
	rows, err := r.store.DB.QueryContext(ctx, `SELECT `+accountColumns("a")+`, b.id, b.account_id, b.date, b.amount, b.scale, b.notes, b.created_at, b.updated_at
		FROM accounts a
		JOIN currencies c ON c.id = a.currency_id
		JOIN balances b ON b.account_id = a.id
		WHERE a.hidden = 0
		ORDER BY a.id, b.date`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []struct {
		Account Account
		Balance Balance
	}
	for rows.Next() {
		var a Account
		var b Balance
		var onBudget, hidden int
		if err := rows.Scan(&a.ID, &a.Name, &a.CurrencyID, &a.Code, &a.Scale, &onBudget, &hidden, &a.Notes, &a.CreatedAt, &a.UpdatedAt,
			&b.ID, &b.AccountID, &b.Date, &b.Amount.Amount, &b.Amount.Scale, &b.Notes, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		a.OnBudget = onBudget == 1
		a.Hidden = hidden == 1
		out = append(out, struct {
			Account Account
			Balance Balance
		}{a, b})
	}
	return out, rows.Err()
}

func (r *BalanceRepo) LatestByAccount(ctx context.Context, accountID int64) (Balance, bool, error) {
	b, err := scanBalance(r.store.DB.QueryRowContext(ctx, balanceSelect+" WHERE account_id=? ORDER BY date DESC LIMIT 1", accountID))
	if err != nil {
		if err.Error() == "balance not found" {
			return Balance{}, false, nil
		}
		return Balance{}, false, err
	}
	return b, true, nil
}

func (r *BalanceRepo) Update(ctx context.Context, b Balance) (Balance, error) {
	now := r.store.Clock().UTC().Format(time.RFC3339)
	_, err := r.store.DB.ExecContext(ctx, `UPDATE balances SET date=?, amount=?, scale=?, notes=?, updated_at=? WHERE id=?`,
		b.Date, b.Amount.Amount, b.Amount.Scale, b.Notes, now, b.ID)
	if err != nil {
		return Balance{}, err
	}
	return r.GetByID(ctx, b.ID)
}

func (r *BalanceRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.store.DB.ExecContext(ctx, "DELETE FROM balances WHERE id=?", id)
	return err
}

const balanceSelect = `SELECT id, account_id, date, amount, scale, notes, created_at, updated_at FROM balances`

func accountColumns(alias string) string {
	return alias + `.id, ` + alias + `.name, ` + alias + `.currency_id, c.code, c.scale, ` + alias + `.on_budget, ` + alias + `.hidden, ` + alias + `.notes, ` + alias + `.created_at, ` + alias + `.updated_at`
}

type balanceScanner interface{ Scan(dest ...any) error }

func scanBalance(row balanceScanner) (Balance, error) {
	var b Balance
	if err := row.Scan(&b.ID, &b.AccountID, &b.Date, &b.Amount.Amount, &b.Amount.Scale, &b.Notes, &b.CreatedAt, &b.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return Balance{}, fmt.Errorf("balance not found")
		}
		return Balance{}, err
	}
	return b, nil
}

func scanBalances(rows *sql.Rows) ([]Balance, error) {
	var out []Balance
	for rows.Next() {
		b, err := scanBalance(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}
