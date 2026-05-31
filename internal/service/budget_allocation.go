package service

import (
	"context"
	"errors"
	"math"

	"stuf/internal/money"
	"stuf/internal/repo"
)

const (
	AllocationActionSetTotal    = "set total"
	AllocationActionAddMoney    = "add money"
	AllocationActionRemoveMoney = "remove money"
	AllocationActionTransferTo  = "transfer to"
)

type BudgetAllocationService struct {
	Store       *repo.Store
	Budgets     *repo.BudgetRepo
	Allocations *repo.BudgetAllocationRepo
	History     HistoryService
}

type BudgetAllocationRow struct {
	Allocation repo.BudgetAllocation
	Balance    money.Money
}

type budgetAllocationMutationData struct {
	Allocation repo.BudgetAllocation
}

type budgetAllocationTransferMutationData struct {
	Source repo.BudgetAllocation
	Target repo.BudgetAllocation
}

func (s BudgetAllocationService) Add(ctx context.Context, budgetID int64, action, amountText, date, notes string) (repo.BudgetAllocation, SessionEntry, error) {
	b, err := s.Budgets.GetByID(ctx, budgetID)
	if err != nil {
		return repo.BudgetAllocation{}, SessionEntry{}, err
	}
	amount, err := parseAllocationAmount(date, amountText, b.Scale)
	if err != nil {
		return repo.BudgetAllocation{}, SessionEntry{}, err
	}
	delta, err := s.deltaForAction(ctx, b, action, amount, date)
	if err != nil {
		return repo.BudgetAllocation{}, SessionEntry{}, err
	}
	var out repo.BudgetAllocation
	var entry SessionEntry
	err = s.Store.WithWriteTx(ctx, func(tx *repo.Store) error {
		alloc, err := tx.Alloc.Create(ctx, repo.BudgetAllocationCreate{BudgetID: budgetID, Date: date, Amount: delta, Notes: notes})
		if err != nil {
			return err
		}
		history := HistoryService{Repo: tx.Hist, Now: s.History.Now}
		e, err := history.Record(ctx, "add", "/budgets/"+b.Name+"/allocations/"+alloc.Date, nil, budgetAllocationMutationData{Allocation: alloc}, func(ctx context.Context) error {
			return s.Allocations.Delete(ctx, alloc.ID)
		})
		if err != nil {
			return err
		}
		out, entry = alloc, e
		return nil
	})
	return out, entry, err
}

func (s BudgetAllocationService) TransferTo(ctx context.Context, sourceBudgetID int64, targetBudgetName, amountText, date, notes string) (repo.BudgetAllocation, repo.BudgetAllocation, SessionEntry, error) {
	sourceBudget, err := s.Budgets.GetByID(ctx, sourceBudgetID)
	if err != nil {
		return repo.BudgetAllocation{}, repo.BudgetAllocation{}, SessionEntry{}, err
	}
	targetBudget, err := s.Budgets.GetByName(ctx, targetBudgetName)
	if err != nil {
		return repo.BudgetAllocation{}, repo.BudgetAllocation{}, SessionEntry{}, err
	}
	if targetBudget.ID == sourceBudget.ID {
		return repo.BudgetAllocation{}, repo.BudgetAllocation{}, SessionEntry{}, errors.New("target budget must be different from source budget")
	}
	sourceAmount, err := parseAllocationAmount(date, amountText, sourceBudget.Scale)
	if err != nil {
		return repo.BudgetAllocation{}, repo.BudgetAllocation{}, SessionEntry{}, err
	}
	if !sourceAmount.IsPositive() {
		return repo.BudgetAllocation{}, repo.BudgetAllocation{}, SessionEntry{}, errors.New("amount must be positive")
	}
	targetAmount, err := sourceAmount.ConvertToScale(targetBudget.Scale)
	if err != nil {
		return repo.BudgetAllocation{}, repo.BudgetAllocation{}, SessionEntry{}, err
	}
	if !targetAmount.IsPositive() {
		return repo.BudgetAllocation{}, repo.BudgetAllocation{}, SessionEntry{}, errors.New("amount must be positive")
	}
	targetNotes := "transfer from " + sourceBudget.Name
	var sourceOut repo.BudgetAllocation
	var targetOut repo.BudgetAllocation
	var entry SessionEntry
	err = s.Store.WithWriteTx(ctx, func(tx *repo.Store) error {
		sourceAlloc, err := tx.Alloc.Create(ctx, repo.BudgetAllocationCreate{BudgetID: sourceBudget.ID, Date: date, Amount: sourceAmount.Negate(), Notes: notes})
		if err != nil {
			return err
		}
		targetAlloc, err := tx.Alloc.Create(ctx, repo.BudgetAllocationCreate{BudgetID: targetBudget.ID, Date: date, Amount: targetAmount, Notes: targetNotes})
		if err != nil {
			return err
		}
		history := HistoryService{Repo: tx.Hist, Now: s.History.Now}
		e, err := history.Record(ctx, "add", "/budgets/"+sourceBudget.Name+"/allocations/"+sourceAlloc.Date, nil, budgetAllocationTransferMutationData{Source: sourceAlloc, Target: targetAlloc}, func(ctx context.Context) error {
			return s.Store.WithWriteTx(ctx, func(tx *repo.Store) error {
				if err := tx.Alloc.Delete(ctx, sourceAlloc.ID); err != nil {
					return err
				}
				return tx.Alloc.Delete(ctx, targetAlloc.ID)
			})
		})
		if err != nil {
			return err
		}
		sourceOut, targetOut, entry = sourceAlloc, targetAlloc, e
		return nil
	})
	return sourceOut, targetOut, entry, err
}

func (s BudgetAllocationService) Update(ctx context.Context, id int64, amountText, date, notes string) (repo.BudgetAllocation, SessionEntry, error) {
	old, err := s.Allocations.GetByID(ctx, id)
	if err != nil {
		return repo.BudgetAllocation{}, SessionEntry{}, err
	}
	b, err := s.Budgets.GetByID(ctx, old.BudgetID)
	if err != nil {
		return repo.BudgetAllocation{}, SessionEntry{}, err
	}
	amount, err := parseAllocationAmount(date, amountText, b.Scale)
	if err != nil {
		return repo.BudgetAllocation{}, SessionEntry{}, err
	}
	next := old
	next.Date, next.Amount, next.Notes = date, amount, notes
	var out repo.BudgetAllocation
	var entry SessionEntry
	err = s.Store.WithWriteTx(ctx, func(tx *repo.Store) error {
		updated, err := tx.Alloc.Update(ctx, next)
		if err != nil {
			return err
		}
		history := HistoryService{Repo: tx.Hist, Now: s.History.Now}
		e, err := history.Record(ctx, "edit", "/budgets/"+b.Name+"/allocations/"+updated.Date, budgetAllocationMutationData{Allocation: old}, budgetAllocationMutationData{Allocation: updated}, func(ctx context.Context) error {
			_, err := s.Allocations.Update(ctx, old)
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

func (s BudgetAllocationService) Delete(ctx context.Context, id int64) (SessionEntry, error) {
	old, err := s.Allocations.GetByID(ctx, id)
	if err != nil {
		return SessionEntry{}, err
	}
	b, err := s.Budgets.GetByID(ctx, old.BudgetID)
	if err != nil {
		return SessionEntry{}, err
	}
	var entry SessionEntry
	err = s.Store.WithWriteTx(ctx, func(tx *repo.Store) error {
		if err := tx.Alloc.Delete(ctx, id); err != nil {
			return err
		}
		history := HistoryService{Repo: tx.Hist, Now: s.History.Now}
		e, err := history.Record(ctx, "delete", "/budgets/"+b.Name+"/allocations/"+old.Date, budgetAllocationMutationData{Allocation: old}, nil, func(ctx context.Context) error {
			_, err := s.Allocations.Create(ctx, repo.BudgetAllocationCreate{BudgetID: old.BudgetID, Date: old.Date, Amount: old.Amount, Notes: old.Notes})
			return err
		})
		if err != nil {
			return err
		}
		entry = e
		return nil
	})
	return entry, err
}

func (s BudgetAllocationService) List(ctx context.Context, budgetID int64) ([]repo.BudgetAllocation, error) {
	return s.Allocations.ListByBudget(ctx, budgetID)
}

func (s BudgetAllocationService) ListWithBalances(ctx context.Context, budgetID int64) ([]BudgetAllocationRow, error) {
	allocs, err := s.Allocations.ListByBudget(ctx, budgetID)
	if err != nil {
		return nil, err
	}
	out := make([]BudgetAllocationRow, 0, len(allocs))
	balance := money.Money{}
	for i, alloc := range allocs {
		if i == 0 {
			balance.Scale = alloc.Amount.Scale
		}
		balance, err = balance.Add(alloc.Amount)
		if err != nil {
			return nil, err
		}
		out = append(out, BudgetAllocationRow{Allocation: alloc, Balance: balance})
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}

func (s BudgetAllocationService) Balance(ctx context.Context, budgetID int64) (money.Money, error) {
	b, err := s.Budgets.GetByID(ctx, budgetID)
	if err != nil {
		return money.Money{}, err
	}
	return s.balanceThrough(ctx, budgetID, "9999-12-31", math.MaxInt64, money.Money{Scale: b.Scale})
}

func (s BudgetAllocationService) BalanceOn(ctx context.Context, budgetID int64, date string) (money.Money, error) {
	b, err := s.Budgets.GetByID(ctx, budgetID)
	if err != nil {
		return money.Money{}, err
	}
	return s.balanceThrough(ctx, budgetID, date, math.MaxInt64, money.Money{Scale: b.Scale})
}

func (s BudgetAllocationService) deltaForAction(ctx context.Context, b repo.Budget, action string, amount money.Money, date string) (money.Money, error) {
	switch action {
	case AllocationActionSetTotal:
		current, err := s.BalanceOn(ctx, b.ID, date)
		if err != nil {
			return money.Money{}, err
		}
		return amount.Sub(current)
	case AllocationActionAddMoney:
		if !amount.IsPositive() {
			return money.Money{}, errors.New("amount must be positive")
		}
		return amount, nil
	case AllocationActionRemoveMoney:
		if !amount.IsPositive() {
			return money.Money{}, errors.New("amount must be positive")
		}
		return amount.Negate(), nil
	default:
		return money.Money{}, errors.New("allocation action must be set total, add money, or remove money")
	}
}

func (s BudgetAllocationService) balanceThrough(ctx context.Context, budgetID int64, date string, maxID int64, zero money.Money) (money.Money, error) {
	allocs, err := s.Allocations.ListByBudget(ctx, budgetID)
	if err != nil {
		return money.Money{}, err
	}
	total := zero
	for _, alloc := range allocs {
		if alloc.Date > date || (alloc.Date == date && alloc.ID > maxID) {
			continue
		}
		total, err = total.Add(alloc.Amount)
		if err != nil {
			return money.Money{}, err
		}
	}
	return total, nil
}

func parseAllocationAmount(date, input string, scale int) (money.Money, error) {
	return parseBalanceAmount(date, input, scale)
}
