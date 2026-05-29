package repo

import (
	"context"
	"time"

	"stuf/internal/db"
)

type BudgetCategoryRepo struct{ store *Store }

func (r *BudgetCategoryRepo) Create(ctx context.Context, name, notes string) (BudgetCategory, error) {
	now := r.store.Clock().UTC().Format(time.RFC3339)
	res, err := r.store.Q.CreateBudgetCategory(ctx, db.CreateBudgetCategoryParams{
		Name:      name,
		Notes:     notes,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		return BudgetCategory{}, mapBudgetCategoryWriteErr(err, name)
	}
	id, _ := res.LastInsertId()
	return r.GetByID(ctx, id)
}

func (r *BudgetCategoryRepo) GetByID(ctx context.Context, id int64) (BudgetCategory, error) {
	row, err := r.store.Q.GetBudgetCategoryByID(ctx, id)
	if err != nil {
		return BudgetCategory{}, mapBudgetCategoryErr(err)
	}
	return budgetCategoryFromDB(row), nil
}

func (r *BudgetCategoryRepo) GetByName(ctx context.Context, name string) (BudgetCategory, error) {
	row, err := r.store.Q.GetBudgetCategoryByName(ctx, name)
	if err != nil {
		return BudgetCategory{}, mapBudgetCategoryErr(err)
	}
	return budgetCategoryFromDB(row), nil
}

func (r *BudgetCategoryRepo) List(ctx context.Context) ([]BudgetCategory, error) {
	rows, err := r.store.Q.ListBudgetCategories(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]BudgetCategory, len(rows))
	for i, row := range rows {
		out[i] = budgetCategoryFromDB(row)
	}
	return out, nil
}

func (r *BudgetCategoryRepo) Update(ctx context.Context, c BudgetCategory) (BudgetCategory, error) {
	now := r.store.Clock().UTC().Format(time.RFC3339)
	if err := r.store.Q.UpdateBudgetCategory(ctx, db.UpdateBudgetCategoryParams{
		Name:      c.Name,
		Notes:     c.Notes,
		UpdatedAt: now,
		ID:        c.ID,
	}); err != nil {
		return BudgetCategory{}, mapBudgetCategoryWriteErr(err, c.Name)
	}
	return r.GetByID(ctx, c.ID)
}

func (r *BudgetCategoryRepo) Delete(ctx context.Context, id int64) error {
	return r.store.Q.DeleteBudgetCategory(ctx, id)
}

func (r *BudgetCategoryRepo) HasBudgets(ctx context.Context, id int64) (bool, error) {
	n, err := r.store.Q.CountBudgetsByCategoryID(ctx, id)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
