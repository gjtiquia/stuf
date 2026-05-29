package service

import (
	"context"
	"errors"
	"strings"

	"stuf/internal/repo"
)

const DefaultBudgetCategoryName = "uncategorized"

type BudgetCategoryService struct {
	Store      *repo.Store
	Categories *repo.BudgetCategoryRepo
	Budgets    *repo.BudgetRepo
	History    HistoryService
}

type budgetCategoryMutationData struct {
	Category repo.BudgetCategory
}

func ValidateBudgetSlug(name, kind string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New(kind + " name is required")
	}
	if err := ValidateSlug(strings.TrimSpace(name)); err != nil {
		return errors.New(kind + " name must be a strict slug: lowercase letters, digits, and hyphens only")
	}
	return nil
}

func (s BudgetCategoryService) Create(ctx context.Context, name, notes string) (repo.BudgetCategory, SessionEntry, error) {
	name = strings.TrimSpace(name)
	if err := ValidateBudgetSlug(name, "budget category"); err != nil {
		return repo.BudgetCategory{}, SessionEntry{}, err
	}
	if name == DefaultBudgetCategoryName {
		return repo.BudgetCategory{}, SessionEntry{}, errors.New("uncategorized is built in")
	}
	var out repo.BudgetCategory
	var entry SessionEntry
	err := s.Store.WithWriteTx(ctx, func(tx *repo.Store) error {
		cat, err := tx.BudCat.Create(ctx, name, notes)
		if err != nil {
			return err
		}
		history := HistoryService{Repo: tx.Hist, Now: s.History.Now}
		e, err := history.Record(ctx, "create", "/budgets/categories/"+cat.Name, nil, budgetCategoryMutationData{Category: cat}, func(ctx context.Context) error {
			return s.deleteIfUnused(ctx, cat.ID)
		})
		if err != nil {
			return err
		}
		out, entry = cat, e
		return nil
	})
	return out, entry, err
}

func (s BudgetCategoryService) Update(ctx context.Context, id int64, name, notes string) (repo.BudgetCategory, SessionEntry, error) {
	name = strings.TrimSpace(name)
	if err := ValidateBudgetSlug(name, "budget category"); err != nil {
		return repo.BudgetCategory{}, SessionEntry{}, err
	}
	old, err := s.Categories.GetByID(ctx, id)
	if err != nil {
		return repo.BudgetCategory{}, SessionEntry{}, err
	}
	if old.Name == DefaultBudgetCategoryName {
		return repo.BudgetCategory{}, SessionEntry{}, errors.New("uncategorized cannot be edited")
	}
	if name == DefaultBudgetCategoryName {
		return repo.BudgetCategory{}, SessionEntry{}, errors.New("uncategorized is built in")
	}
	next := old
	next.Name, next.Notes = name, notes
	var out repo.BudgetCategory
	var entry SessionEntry
	err = s.Store.WithWriteTx(ctx, func(tx *repo.Store) error {
		updated, err := tx.BudCat.Update(ctx, next)
		if err != nil {
			return err
		}
		history := HistoryService{Repo: tx.Hist, Now: s.History.Now}
		e, err := history.Record(ctx, "edit", "/budgets/categories/"+updated.Name, budgetCategoryMutationData{Category: old}, budgetCategoryMutationData{Category: updated}, func(ctx context.Context) error {
			_, err := s.Categories.Update(ctx, old)
			return err
		})
		if err != nil {
			return err
		}
		out, entry = updated, e
		return nil
	})
	return out, entry, err
}

func (s BudgetCategoryService) List(ctx context.Context) ([]repo.BudgetCategory, error) {
	return s.Categories.List(ctx)
}

func (s BudgetCategoryService) GetByName(ctx context.Context, name string) (repo.BudgetCategory, error) {
	return s.Categories.GetByName(ctx, name)
}

func (s BudgetCategoryService) GetByID(ctx context.Context, id int64) (repo.BudgetCategory, error) {
	return s.Categories.GetByID(ctx, id)
}

func (s BudgetCategoryService) deleteIfUnused(ctx context.Context, id int64) error {
	cat, err := s.Categories.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if cat.Name == DefaultBudgetCategoryName {
		return nil
	}
	has, err := s.Categories.HasBudgets(ctx, id)
	if err != nil {
		return err
	}
	if has {
		return nil
	}
	return s.Categories.Delete(ctx, id)
}
