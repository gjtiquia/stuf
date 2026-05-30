package repo

import (
	"context"
	"time"

	"stuf/internal/db"
	"stuf/internal/money"
)

type TransactionRepo struct{ store *Store }

type TransactionCreate struct {
	Ref        int64
	ParentID   *int64
	AccountID  int64
	Type       string
	CurrencyID int64
	Date       string
	Amount     money.Money
	Notes      string
}

func (r *TransactionRepo) Create(ctx context.Context, in TransactionCreate) (Transaction, error) {
	now := r.store.Clock().UTC().Format(time.RFC3339)
	ref := in.Ref
	if ref == 0 {
		var err error
		ref, err = r.store.Q.NextTransactionRef(ctx)
		if err != nil {
			return Transaction{}, err
		}
	}
	res, err := r.store.Q.CreateTransaction(ctx, db.CreateTransactionParams{
		Ref:        ref,
		ParentID:   nullInt64(in.ParentID),
		AccountID:  in.AccountID,
		Type:       in.Type,
		CurrencyID: in.CurrencyID,
		Date:       in.Date,
		Amount:     in.Amount.Amount,
		Scale:      int64(in.Amount.Scale),
		Notes:      in.Notes,
		CreatedAt:  now,
		UpdatedAt:  now,
	})
	if err != nil {
		return Transaction{}, err
	}
	id, _ := res.LastInsertId()
	return r.GetByID(ctx, id)
}

func (r *TransactionRepo) GetByID(ctx context.Context, id int64) (Transaction, error) {
	row, err := r.store.Q.GetTransactionByID(ctx, id)
	if err != nil {
		return Transaction{}, mapTransactionErr(err)
	}
	return transactionFromGetRow(row), nil
}

func (r *TransactionRepo) GetByRef(ctx context.Context, ref int64) (Transaction, error) {
	row, err := r.store.Q.GetTransactionByRef(ctx, ref)
	if err != nil {
		return Transaction{}, mapTransactionErr(err)
	}
	return transactionFromRefRow(row), nil
}

func (r *TransactionRepo) List(ctx context.Context) ([]Transaction, error) {
	rows, err := r.store.Q.ListTransactions(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Transaction, len(rows))
	for i, row := range rows {
		out[i] = transactionFromListRow(row)
	}
	return out, nil
}

func (r *TransactionRepo) ListByAccount(ctx context.Context, accountID int64) ([]Transaction, error) {
	rows, err := r.store.Q.ListTransactionsByAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}
	out := make([]Transaction, len(rows))
	for i, row := range rows {
		out[i] = transactionFromAccountRow(row)
	}
	return out, nil
}

func (r *TransactionRepo) ListByParent(ctx context.Context, parentID int64) ([]Transaction, error) {
	rows, err := r.store.Q.ListTransactionsByParent(ctx, accountIDParam(parentID))
	if err != nil {
		return nil, err
	}
	out := make([]Transaction, len(rows))
	for i, row := range rows {
		out[i] = transactionFromParentRow(row)
	}
	return out, nil
}

func (r *TransactionRepo) Update(ctx context.Context, t Transaction) (Transaction, error) {
	now := r.store.Clock().UTC().Format(time.RFC3339)
	if err := r.store.Q.UpdateTransaction(ctx, db.UpdateTransactionParams{
		ParentID:   nullInt64(t.ParentID),
		AccountID:  t.AccountID,
		Type:       t.Type,
		CurrencyID: t.CurrencyID,
		Date:       t.Date,
		Amount:     t.Amount.Amount,
		Scale:      int64(t.Amount.Scale),
		Notes:      t.Notes,
		UpdatedAt:  now,
		ID:         t.ID,
	}); err != nil {
		return Transaction{}, err
	}
	return r.GetByID(ctx, t.ID)
}

func (r *TransactionRepo) Delete(ctx context.Context, id int64) error {
	return r.store.Q.DeleteTransaction(ctx, id)
}

func (r *TransactionRepo) HasChildren(ctx context.Context, id int64) (bool, error) {
	n, err := r.store.Q.CountTransactionsByParentID(ctx, accountIDParam(id))
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
