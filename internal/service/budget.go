package service

import (
	"context"
	"errors"
	"strings"

	"stuf/internal/repo"
)

type BudgetService struct {
	Store       *repo.Store
	Budgets     *repo.BudgetRepo
	Categories  *repo.BudgetCategoryRepo
	Currency    *repo.CurrencyRepo
	Allocations *repo.BudgetAllocationRepo
	History     HistoryService
	AppCurrency string
}

type budgetMutationData struct {
	Budget repo.Budget
}

func (s BudgetService) Create(ctx context.Context, name, currencyCode, categoryName, notes string) (repo.Budget, SessionEntry, error) {
	name = strings.TrimSpace(name)
	if err := ValidateBudgetSlug(name, "budget"); err != nil {
		return repo.Budget{}, SessionEntry{}, err
	}
	if currencyCode == "" {
		currencyCode = s.AppCurrency
	}
	categoryName = normalizeBudgetCategoryName(categoryName)
	if err := ValidateBudgetSlug(categoryName, "budget category"); err != nil {
		return repo.Budget{}, SessionEntry{}, err
	}
	cur, err := s.Currency.GetByCode(ctx, currencyCode)
	if err != nil {
		return repo.Budget{}, SessionEntry{}, err
	}
	var out repo.Budget
	var entry SessionEntry
	err = s.Store.WithWriteTx(ctx, func(tx *repo.Store) error {
		cat, createdCat, err := resolveBudgetCategory(ctx, tx.BudCat, categoryName)
		if err != nil {
			return err
		}
		b, err := tx.Bud.Create(ctx, repo.BudgetCreate{Name: name, CurrencyID: cur.ID, CategoryID: cat.ID, Notes: notes})
		if err != nil {
			return err
		}
		history := HistoryService{Repo: tx.Hist, Now: s.History.Now}
		e, err := history.Record(ctx, "create", "/budgets/"+b.Name, nil, budgetMutationData{Budget: b}, func(ctx context.Context) error {
			has, err := s.Budgets.HasAllocations(ctx, b.ID)
			if err != nil {
				return err
			}
			if has {
				return nil
			}
			if err := s.Budgets.Delete(ctx, b.ID); err != nil {
				return err
			}
			if createdCat {
				_ = (BudgetCategoryService{Categories: s.Categories, Budgets: s.Budgets}).deleteIfUnused(ctx, cat.ID)
			}
			return nil
		})
		if err != nil {
			return err
		}
		out, entry = b, e
		return nil
	})
	return out, entry, err
}

func (s BudgetService) Update(ctx context.Context, id int64, name, currencyCode, categoryName string, hidden bool, notes string) (repo.Budget, SessionEntry, error) {
	name = strings.TrimSpace(name)
	if err := ValidateBudgetSlug(name, "budget"); err != nil {
		return repo.Budget{}, SessionEntry{}, err
	}
	categoryName = normalizeBudgetCategoryName(categoryName)
	if err := ValidateBudgetSlug(categoryName, "budget category"); err != nil {
		return repo.Budget{}, SessionEntry{}, err
	}
	old, err := s.Budgets.GetByID(ctx, id)
	if err != nil {
		return repo.Budget{}, SessionEntry{}, err
	}
	currencyID := old.CurrencyID
	if currencyCode != "" && currencyCode != old.Code {
		has, err := s.Budgets.HasAllocations(ctx, id)
		if err != nil {
			return repo.Budget{}, SessionEntry{}, err
		}
		if has {
			return repo.Budget{}, SessionEntry{}, errors.New("budget currency cannot be changed after allocations exist")
		}
		cur, err := s.Currency.GetByCode(ctx, currencyCode)
		if err != nil {
			return repo.Budget{}, SessionEntry{}, err
		}
		currencyID = cur.ID
	}
	var out repo.Budget
	var entry SessionEntry
	err = s.Store.WithWriteTx(ctx, func(tx *repo.Store) error {
		cat, createdCat, err := resolveBudgetCategory(ctx, tx.BudCat, categoryName)
		if err != nil {
			return err
		}
		next := old
		next.Name, next.CurrencyID, next.CategoryID, next.Hidden, next.Notes = name, currencyID, cat.ID, hidden, notes
		updated, err := tx.Bud.Update(ctx, next)
		if err != nil {
			return err
		}
		history := HistoryService{Repo: tx.Hist, Now: s.History.Now}
		e, err := history.Record(ctx, "edit", "/budgets/"+updated.Name, budgetMutationData{Budget: old}, budgetMutationData{Budget: updated}, func(ctx context.Context) error {
			_, err := s.Budgets.Update(ctx, old)
			if createdCat {
				_ = (BudgetCategoryService{Categories: s.Categories, Budgets: s.Budgets}).deleteIfUnused(ctx, cat.ID)
			}
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

func (s BudgetService) SetHidden(ctx context.Context, id int64, hidden bool) (repo.Budget, SessionEntry, error) {
	old, err := s.Budgets.GetByID(ctx, id)
	if err != nil {
		return repo.Budget{}, SessionEntry{}, err
	}
	return s.Update(ctx, id, old.Name, old.Code, old.CategoryName, hidden, old.Notes)
}

func (s BudgetService) List(ctx context.Context, includeHidden bool) ([]repo.Budget, error) {
	return s.Budgets.List(ctx, includeHidden)
}

func (s BudgetService) ListByCategory(ctx context.Context, categoryID int64) ([]repo.Budget, error) {
	return s.Budgets.ListByCategory(ctx, categoryID)
}

func (s BudgetService) GetByName(ctx context.Context, name string) (repo.Budget, error) {
	return s.Budgets.GetByName(ctx, name)
}

func (s BudgetService) GetByID(ctx context.Context, id int64) (repo.Budget, error) {
	return s.Budgets.GetByID(ctx, id)
}

func normalizeBudgetCategoryName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return DefaultBudgetCategoryName
	}
	return name
}

func resolveBudgetCategory(ctx context.Context, cats *repo.BudgetCategoryRepo, name string) (repo.BudgetCategory, bool, error) {
	cat, err := cats.GetByName(ctx, name)
	if err == nil {
		return cat, false, nil
	}
	cat, err = cats.Create(ctx, name, "")
	return cat, true, err
}
