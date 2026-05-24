package repo

import (
	"context"
	"database/sql"
	"time"

	"stuf/internal/db"
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
		return Balance{}, err
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
			Account: accountFromFields(row.AccountID, row.AccountName, row.CurrencyID, row.Code, row.Scale, row.OnBudget, row.Hidden, row.AccountNotes, row.AccountCreatedAt, row.AccountUpdatedAt),
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
		return Balance{}, err
	}
	return r.GetByID(ctx, b.ID)
}

func (r *BalanceRepo) Delete(ctx context.Context, id int64) error {
	return r.store.Q.DeleteBalance(ctx, id)
}
