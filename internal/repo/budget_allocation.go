package repo

import (
	"context"
	"time"

	"stuf/internal/db"
	"stuf/internal/money"
)

type BudgetAllocationRepo struct{ store *Store }

type BudgetAllocationCreate struct {
	BudgetID int64
	Date     string
	Amount   money.Money
	Notes    string
}

func (r *BudgetAllocationRepo) Create(ctx context.Context, in BudgetAllocationCreate) (BudgetAllocation, error) {
	now := r.store.Clock().UTC().Format(time.RFC3339)
	res, err := r.store.Q.CreateBudgetAllocation(ctx, db.CreateBudgetAllocationParams{
		BudgetID:  in.BudgetID,
		Date:      in.Date,
		Amount:    in.Amount.Amount,
		Scale:     int64(in.Amount.Scale),
		Notes:     in.Notes,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		return BudgetAllocation{}, err
	}
	id, _ := res.LastInsertId()
	return r.GetByID(ctx, id)
}

func (r *BudgetAllocationRepo) GetByID(ctx context.Context, id int64) (BudgetAllocation, error) {
	row, err := r.store.Q.GetBudgetAllocationByID(ctx, id)
	if err != nil {
		return BudgetAllocation{}, mapBudgetAllocationErr(err)
	}
	return budgetAllocationFromDB(row), nil
}

func (r *BudgetAllocationRepo) ListByBudget(ctx context.Context, budgetID int64) ([]BudgetAllocation, error) {
	rows, err := r.store.Q.ListBudgetAllocationsByBudget(ctx, budgetID)
	if err != nil {
		return nil, err
	}
	out := make([]BudgetAllocation, len(rows))
	for i, row := range rows {
		out[i] = budgetAllocationFromDB(row)
	}
	return out, nil
}

func (r *BudgetAllocationRepo) Update(ctx context.Context, a BudgetAllocation) (BudgetAllocation, error) {
	now := r.store.Clock().UTC().Format(time.RFC3339)
	if err := r.store.Q.UpdateBudgetAllocation(ctx, db.UpdateBudgetAllocationParams{
		Date:      a.Date,
		Amount:    a.Amount.Amount,
		Scale:     int64(a.Amount.Scale),
		Notes:     a.Notes,
		UpdatedAt: now,
		ID:        a.ID,
	}); err != nil {
		return BudgetAllocation{}, err
	}
	return r.GetByID(ctx, a.ID)
}

func (r *BudgetAllocationRepo) Delete(ctx context.Context, id int64) error {
	return r.store.Q.DeleteBudgetAllocation(ctx, id)
}
