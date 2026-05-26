package repo

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"stuf/internal/db"
	"stuf/internal/money"

	"modernc.org/sqlite"
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
	res, err := r.store.Q.CreateBalance(ctx, db.CreateBalanceParams{
		AccountID: in.AccountID,
		Date:      in.Date,
		Amount:    in.Amount.Amount,
		Scale:     int64(in.Amount.Scale),
		Notes:     in.Notes,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		return Balance{}, mapBalanceWriteErr(err, in.Date)
	}
	id, _ := res.LastInsertId()
	return r.GetByID(ctx, id)
}

func (r *BalanceRepo) GetByID(ctx context.Context, id int64) (Balance, error) {
	b, err := r.store.Q.GetBalanceByID(ctx, id)
	if err != nil {
		return Balance{}, mapBalanceErr(err)
	}
	return balanceFromDB(b), nil
}

func (r *BalanceRepo) GetByAccountDate(ctx context.Context, accountID int64, date string) (Balance, error) {
	b, err := r.store.Q.GetBalanceByAccountDate(ctx, db.GetBalanceByAccountDateParams{
		AccountID: accountID,
		Date:      date,
	})
	if err != nil {
		return Balance{}, mapBalanceErr(err)
	}
	return balanceFromDB(b), nil
}

func (r *BalanceRepo) ListByAccount(ctx context.Context, accountID int64) ([]Balance, error) {
	rows, err := r.store.Q.ListBalancesByAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}
	out := make([]Balance, len(rows))
	for i, row := range rows {
		out[i] = balanceFromDB(row)
	}
	return out, nil
}

func (r *BalanceRepo) ListAllVisible(ctx context.Context) ([]struct {
	Account Account
	Balance Balance
}, error) {
	rows, err := r.store.Q.ListAllVisibleBalances(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]struct {
		Account Account
		Balance Balance
	}, len(rows))
	for i, row := range rows {
		out[i] = struct {
			Account Account
			Balance Balance
		}{
			Account: accountFromFields(row.AccountID, row.AccountName, row.CurrencyID, row.ParentID, row.Code, row.Scale, row.OnBudget, row.Hidden, row.AccountNotes, row.AccountCreatedAt, row.AccountUpdatedAt),
			Balance: Balance{
				ID:        row.BalanceID,
				AccountID: row.BalanceAccountID,
				Date:      row.Date,
				Amount:    money.Money{Amount: row.Amount, Scale: int(row.BalanceScale)},
				Notes:     row.BalanceNotes,
				CreatedAt: row.BalanceCreatedAt,
				UpdatedAt: row.BalanceUpdatedAt,
			},
		}
	}
	return out, nil
}

func (r *BalanceRepo) LatestByAccount(ctx context.Context, accountID int64) (Balance, bool, error) {
	b, err := r.store.Q.GetLatestBalanceByAccount(ctx, accountID)
	if err != nil {
		if err == sql.ErrNoRows {
			return Balance{}, false, nil
		}
		return Balance{}, false, err
	}
	return balanceFromDB(b), true, nil
}

func (r *BalanceRepo) Update(ctx context.Context, b Balance) (Balance, error) {
	now := r.store.Clock().UTC().Format(time.RFC3339)
	if err := r.store.Q.UpdateBalance(ctx, db.UpdateBalanceParams{
		Date:      b.Date,
		Amount:    b.Amount.Amount,
		Scale:     int64(b.Amount.Scale),
		Notes:     b.Notes,
		UpdatedAt: now,
		ID:        b.ID,
	}); err != nil {
		return Balance{}, mapBalanceWriteErr(err, b.Date)
	}
	return r.GetByID(ctx, b.ID)
}

func (r *BalanceRepo) Delete(ctx context.Context, id int64) error {
	return r.store.Q.DeleteBalance(ctx, id)
}

func mapBalanceWriteErr(err error, date string) error {
	if isBalanceDuplicateDateErr(err) {
		return &BalanceDuplicateDateError{Date: date}
	}
	return err
}

func isBalanceDuplicateDateErr(err error) bool {
	var sqliteErr *sqlite.Error
	if errors.As(err, &sqliteErr) {
		return sqliteErr.Code() == 2067 && strings.Contains(sqliteErr.Error(), "balances.account_id, balances.date")
	}
	return strings.Contains(err.Error(), "UNIQUE constraint failed: balances.account_id, balances.date")
}
