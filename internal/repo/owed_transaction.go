package repo

import (
	"context"
	"time"

	"stuf/internal/db"
	"stuf/internal/money"
)

type OwedTransactionRepo struct{ store *Store }

type OwedTransactionCreate struct {
	LedgerID   int64
	Date       string
	CurrencyID int64
	Amount     money.Money
	Formula    string
	Notes      string
}

func (r *OwedTransactionRepo) Create(ctx context.Context, in OwedTransactionCreate) (OwedTransaction, error) {
	now := r.store.Clock().UTC().Format(time.RFC3339)
	res, err := r.store.Q.CreateOwedTransaction(ctx, db.CreateOwedTransactionParams{
		LedgerID:   in.LedgerID,
		Date:       in.Date,
		CurrencyID: in.CurrencyID,
		Amount:     in.Amount.Amount,
		Scale:      int64(in.Amount.Scale),
		Formula:    in.Formula,
		Notes:      in.Notes,
		CreatedAt:  now,
		UpdatedAt:  now,
	})
	if err != nil {
		return OwedTransaction{}, err
	}
	id, _ := res.LastInsertId()
	return r.GetByID(ctx, id)
}

func (r *OwedTransactionRepo) GetByID(ctx context.Context, id int64) (OwedTransaction, error) {
	row, err := r.store.Q.GetOwedTransactionByID(ctx, id)
	if err != nil {
		return OwedTransaction{}, mapOwedTransactionErr(err)
	}
	return owedTransactionFromGetRow(row), nil
}

func (r *OwedTransactionRepo) ListByLedger(ctx context.Context, ledgerID int64) ([]OwedTransaction, error) {
	rows, err := r.store.Q.ListOwedTransactionsByLedger(ctx, ledgerID)
	if err != nil {
		return nil, err
	}
	out := make([]OwedTransaction, len(rows))
	for i, row := range rows {
		out[i] = owedTransactionFromListRow(row)
	}
	return out, nil
}

func (r *OwedTransactionRepo) Update(ctx context.Context, t OwedTransaction) (OwedTransaction, error) {
	now := r.store.Clock().UTC().Format(time.RFC3339)
	if err := r.store.Q.UpdateOwedTransaction(ctx, db.UpdateOwedTransactionParams{
		Date:       t.Date,
		CurrencyID: t.CurrencyID,
		Amount:     t.Amount.Amount,
		Scale:      int64(t.Amount.Scale),
		Formula:    t.Formula,
		Notes:      t.Notes,
		UpdatedAt:  now,
		ID:         t.ID,
	}); err != nil {
		return OwedTransaction{}, err
	}
	return r.GetByID(ctx, t.ID)
}

func (r *OwedTransactionRepo) Delete(ctx context.Context, id int64) error {
	return r.store.Q.DeleteOwedTransaction(ctx, id)
}
