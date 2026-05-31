package repo

import (
	"context"
	"time"

	"stuf/internal/db"
)

type OwedLedgerRepo struct{ store *Store }

type OwedLedgerCreate struct {
	Name       string
	CurrencyID int64
	Notes      string
}

func (r *OwedLedgerRepo) Create(ctx context.Context, in OwedLedgerCreate) (OwedLedger, error) {
	now := r.store.Clock().UTC().Format(time.RFC3339)
	res, err := r.store.Q.CreateOwedLedger(ctx, db.CreateOwedLedgerParams{
		Name:       in.Name,
		CurrencyID: in.CurrencyID,
		Notes:      in.Notes,
		CreatedAt:  now,
		UpdatedAt:  now,
	})
	if err != nil {
		return OwedLedger{}, mapOwedLedgerWriteErr(err, in.Name)
	}
	id, _ := res.LastInsertId()
	return r.GetByID(ctx, id)
}

func (r *OwedLedgerRepo) GetByID(ctx context.Context, id int64) (OwedLedger, error) {
	row, err := r.store.Q.GetOwedLedgerByID(ctx, id)
	if err != nil {
		return OwedLedger{}, mapOwedLedgerErr(err)
	}
	return owedLedgerFromGetRow(row), nil
}

func (r *OwedLedgerRepo) GetByName(ctx context.Context, name string) (OwedLedger, error) {
	row, err := r.store.Q.GetOwedLedgerByName(ctx, name)
	if err != nil {
		return OwedLedger{}, mapOwedLedgerErr(err)
	}
	return owedLedgerFromNameRow(row), nil
}

func (r *OwedLedgerRepo) List(ctx context.Context) ([]OwedLedger, error) {
	rows, err := r.store.Q.ListOwedLedgers(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]OwedLedger, len(rows))
	for i, row := range rows {
		out[i] = owedLedgerFromListRow(row)
	}
	return out, nil
}

func (r *OwedLedgerRepo) Update(ctx context.Context, l OwedLedger) (OwedLedger, error) {
	now := r.store.Clock().UTC().Format(time.RFC3339)
	if err := r.store.Q.UpdateOwedLedger(ctx, db.UpdateOwedLedgerParams{
		Name:       l.Name,
		CurrencyID: l.CurrencyID,
		Notes:      l.Notes,
		UpdatedAt:  now,
		ID:         l.ID,
	}); err != nil {
		return OwedLedger{}, mapOwedLedgerWriteErr(err, l.Name)
	}
	return r.GetByID(ctx, l.ID)
}

func (r *OwedLedgerRepo) Delete(ctx context.Context, id int64) error {
	return r.store.Q.DeleteOwedLedger(ctx, id)
}

func (r *OwedLedgerRepo) HasTransactions(ctx context.Context, id int64) (bool, error) {
	n, err := r.store.Q.CountOwedTransactionsByLedgerID(ctx, id)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
