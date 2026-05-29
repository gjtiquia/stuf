package repo

import (
	"context"
	"time"

	"stuf/internal/db"
)

type BudgetRepo struct{ store *Store }

type BudgetCreate struct {
	Name       string
	CurrencyID int64
	CategoryID int64
	Hidden     bool
	Notes      string
}

func (r *BudgetRepo) Create(ctx context.Context, in BudgetCreate) (Budget, error) {
	now := r.store.Clock().UTC().Format(time.RFC3339)
	res, err := r.store.Q.CreateBudget(ctx, db.CreateBudgetParams{
		Name:       in.Name,
		CurrencyID: in.CurrencyID,
		CategoryID: in.CategoryID,
		Hidden:     boolInt(in.Hidden),
		Notes:      in.Notes,
		CreatedAt:  now,
		UpdatedAt:  now,
	})
	if err != nil {
		return Budget{}, mapBudgetWriteErr(err, in.Name)
	}
	id, _ := res.LastInsertId()
	return r.GetByID(ctx, id)
}

func (r *BudgetRepo) GetByID(ctx context.Context, id int64) (Budget, error) {
	row, err := r.store.Q.GetBudgetByID(ctx, id)
	if err != nil {
		return Budget{}, mapBudgetErr(err)
	}
	return budgetFromGetRow(row), nil
}

func (r *BudgetRepo) GetByName(ctx context.Context, name string) (Budget, error) {
	row, err := r.store.Q.GetBudgetByName(ctx, name)
	if err != nil {
		return Budget{}, mapBudgetErr(err)
	}
	return budgetFromNameRow(row), nil
}

func (r *BudgetRepo) List(ctx context.Context, includeHidden bool) ([]Budget, error) {
	if includeHidden {
		rows, err := r.store.Q.ListBudgets(ctx)
		if err != nil {
			return nil, err
		}
		out := make([]Budget, len(rows))
		for i, row := range rows {
			out[i] = budgetFromListRow(row)
		}
		return out, nil
	}
	rows, err := r.store.Q.ListVisibleBudgets(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Budget, len(rows))
	for i, row := range rows {
		out[i] = budgetFromVisibleRow(row)
	}
	return out, nil
}

func (r *BudgetRepo) ListByCategory(ctx context.Context, categoryID int64) ([]Budget, error) {
	rows, err := r.store.Q.ListBudgetsByCategoryID(ctx, categoryID)
	if err != nil {
		return nil, err
	}
	out := make([]Budget, len(rows))
	for i, row := range rows {
		out[i] = budgetFromCategoryRow(row)
	}
	return out, nil
}

func (r *BudgetRepo) Update(ctx context.Context, b Budget) (Budget, error) {
	now := r.store.Clock().UTC().Format(time.RFC3339)
	if err := r.store.Q.UpdateBudget(ctx, db.UpdateBudgetParams{
		Name:       b.Name,
		CurrencyID: b.CurrencyID,
		CategoryID: b.CategoryID,
		Hidden:     boolInt(b.Hidden),
		Notes:      b.Notes,
		UpdatedAt:  now,
		ID:         b.ID,
	}); err != nil {
		return Budget{}, mapBudgetWriteErr(err, b.Name)
	}
	return r.GetByID(ctx, b.ID)
}

func (r *BudgetRepo) Delete(ctx context.Context, id int64) error {
	return r.store.Q.DeleteBudget(ctx, id)
}

func (r *BudgetRepo) HasAllocations(ctx context.Context, id int64) (bool, error) {
	n, err := r.store.Q.CountAllocationsByBudgetID(ctx, id)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
