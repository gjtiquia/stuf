package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"stuf/internal/money"
	"stuf/internal/repo"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$`)

type AccountService struct {
	Store       *repo.Store
	Accounts    *repo.AccountRepo
	Balances    *repo.BalanceRepo
	Currency    *repo.CurrencyRepo
	History     HistoryService
	AppCurrency string
}

type AccountTreeSummary struct {
	Account       repo.Account
	Balance       money.Money
	Children      money.Money
	Remaining     money.Money
	AsOf          string
	HasOwnBalance bool
}

func ValidateSlug(name string) error {
	if !slugPattern.MatchString(name) {
		return errors.New("account name must be a strict slug: lowercase letters, digits, and hyphens only")
	}
	return nil
}

func (s AccountService) Create(ctx context.Context, name, currencyCode string, onBudget bool, notes string) (repo.Account, SessionEntry, error) {
	return s.create(ctx, name, currencyCode, nil, onBudget, false, notes)
}

func (s AccountService) CreateChild(ctx context.Context, parentID int64, name, currencyCode string, notes string) (repo.Account, SessionEntry, error) {
	parent, err := s.Accounts.GetByID(ctx, parentID)
	if err != nil {
		return repo.Account{}, SessionEntry{}, err
	}
	return s.create(ctx, name, currencyCode, &parent.ID, parent.OnBudget, false, notes)
}

func (s AccountService) create(ctx context.Context, name, currencyCode string, parentID *int64, onBudget, hidden bool, notes string) (repo.Account, SessionEntry, error) {
	if err := ValidateSlug(name); err != nil {
		return repo.Account{}, SessionEntry{}, err
	}
	if currencyCode == "" {
		currencyCode = s.AppCurrency
	}
	cur, err := s.Currency.GetByCode(ctx, currencyCode)
	if err != nil {
		return repo.Account{}, SessionEntry{}, err
	}
	var out repo.Account
	var entry SessionEntry
	err = s.Store.WithWriteLock(func() error {
		a, err := s.Accounts.Create(ctx, repo.AccountCreate{Name: name, CurrencyID: cur.ID, ParentID: parentID, OnBudget: onBudget, Hidden: hidden, Notes: notes})
		if err != nil {
			return err
		}
		e, err := s.History.Record(ctx, "create", "/accounts/"+a.Name, nil, a, func(ctx context.Context) error {
			return s.Accounts.Delete(ctx, a.ID)
		})
		if err != nil {
			return err
		}
		out, entry = a, e
		return nil
	})
	return out, entry, err
}

func (s AccountService) Update(ctx context.Context, id int64, name, currencyCode string, onBudget, hidden bool, notes string) (repo.Account, SessionEntry, error) {
	if err := ValidateSlug(name); err != nil {
		return repo.Account{}, SessionEntry{}, err
	}
	old, err := s.Accounts.GetByID(ctx, id)
	if err != nil {
		return repo.Account{}, SessionEntry{}, err
	}
	if old.ParentID != nil {
		parent, err := s.Accounts.GetByID(ctx, *old.ParentID)
		if err != nil {
			return repo.Account{}, SessionEntry{}, err
		}
		if onBudget != parent.OnBudget {
			return repo.Account{}, SessionEntry{}, errors.New("child account on-budget status must match parent")
		}
	}
	currencyID := old.CurrencyID
	if currencyCode != "" && currencyCode != old.Code {
		has, err := s.Accounts.HasBalances(ctx, id)
		if err != nil {
			return repo.Account{}, SessionEntry{}, err
		}
		if has {
			return repo.Account{}, SessionEntry{}, errors.New("account currency cannot be changed after balances exist")
		}
		cur, err := s.Currency.GetByCode(ctx, currencyCode)
		if err != nil {
			return repo.Account{}, SessionEntry{}, err
		}
		currencyID = cur.ID
	}
	next := old
	next.Name, next.CurrencyID, next.OnBudget, next.Hidden, next.Notes = name, currencyID, onBudget, hidden, notes
	var out repo.Account
	var entry SessionEntry
	err = s.Store.WithWriteLock(func() error {
		updated, err := s.Accounts.Update(ctx, next)
		if err != nil {
			return err
		}
		if old.ParentID == nil && old.OnBudget != updated.OnBudget {
			if err := s.cascadeOnBudget(ctx, updated.ID, updated.OnBudget); err != nil {
				return err
			}
		}
		e, err := s.History.Record(ctx, "edit", "/accounts/"+updated.Name, old, updated, func(ctx context.Context) error {
			_, err := s.Accounts.Update(ctx, old)
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

func (s AccountService) cascadeOnBudget(ctx context.Context, accountID int64, onBudget bool) error {
	descendants, err := s.Accounts.ListDescendants(ctx, accountID)
	if err != nil {
		return err
	}
	for _, child := range descendants {
		child.OnBudget = onBudget
		if _, err := s.Accounts.Update(ctx, child); err != nil {
			return err
		}
	}
	return nil
}

func (s AccountService) DeleteEmpty(ctx context.Context, id int64) (repo.Account, SessionEntry, error) {
	old, err := s.Accounts.GetByID(ctx, id)
	if err != nil {
		return repo.Account{}, SessionEntry{}, err
	}
	empty, err := s.Accounts.IsEmpty(ctx, id)
	if err != nil {
		return repo.Account{}, SessionEntry{}, err
	}
	if !empty {
		return repo.Account{}, SessionEntry{}, errors.New("account is not empty; hide it instead")
	}
	var entry SessionEntry
	err = s.Store.WithWriteLock(func() error {
		if err := s.Accounts.Delete(ctx, old.ID); err != nil {
			return err
		}
		e, err := s.History.Record(ctx, "delete", "/accounts/"+old.Name, old, nil, func(ctx context.Context) error {
			_, err := s.Accounts.Create(ctx, repo.AccountCreate{
				Name:       old.Name,
				CurrencyID: old.CurrencyID,
				ParentID:   old.ParentID,
				OnBudget:   old.OnBudget,
				Hidden:     old.Hidden,
				Notes:      old.Notes,
			})
			return err
		})
		if err != nil {
			return err
		}
		entry = e
		return nil
	})
	return old, entry, err
}

func (s AccountService) SetHidden(ctx context.Context, id int64, hidden bool) (repo.Account, SessionEntry, error) {
	old, err := s.Accounts.GetByID(ctx, id)
	if err != nil {
		return repo.Account{}, SessionEntry{}, err
	}
	return s.Update(ctx, id, old.Name, old.Code, old.OnBudget, hidden, old.Notes)
}

func (s AccountService) List(ctx context.Context, includeHidden bool) ([]repo.Account, error) {
	return s.Accounts.List(ctx, includeHidden)
}

func (s AccountService) ListRoots(ctx context.Context, includeHidden bool) ([]repo.Account, error) {
	return s.Accounts.ListRoots(ctx, includeHidden)
}

func (s AccountService) ListChildren(ctx context.Context, accountID int64, includeHidden bool) ([]repo.Account, error) {
	return s.Accounts.ListChildren(ctx, accountID, includeHidden)
}

func (s AccountService) GetByName(ctx context.Context, name string) (repo.Account, error) {
	return s.Accounts.GetByName(ctx, name)
}

func (s AccountService) GetByID(ctx context.Context, id int64) (repo.Account, error) {
	return s.Accounts.GetByID(ctx, id)
}

func (s AccountService) CurrentBalance(ctx context.Context, accountID int64) (repo.Balance, bool, error) {
	return s.Balances.LatestByAccount(ctx, accountID)
}

func (s AccountService) HasBalances(ctx context.Context, accountID int64) (bool, error) {
	return s.Accounts.HasBalances(ctx, accountID)
}

func (s AccountService) IsEmpty(ctx context.Context, accountID int64) (bool, error) {
	return s.Accounts.IsEmpty(ctx, accountID)
}

func (s AccountService) TreeSummary(ctx context.Context, accountID int64, targetCurrencyCode string) (AccountTreeSummary, error) {
	a, err := s.Accounts.GetByID(ctx, accountID)
	if err != nil {
		return AccountTreeSummary{}, err
	}
	target, err := s.Currency.GetByCode(ctx, targetCurrencyCode)
	if err != nil {
		return AccountTreeSummary{}, err
	}
	return s.treeSummary(ctx, a, target)
}

func (s AccountService) treeSummary(ctx context.Context, a repo.Account, target repo.Currency) (AccountTreeSummary, error) {
	zero := money.Money{Scale: target.Scale}
	out := AccountTreeSummary{Account: a, Balance: zero, Children: zero, Remaining: zero}
	children, err := s.Accounts.ListChildren(ctx, a.ID, false)
	if err != nil {
		return AccountTreeSummary{}, err
	}
	for _, child := range children {
		childSummary, err := s.treeSummary(ctx, child, target)
		if err != nil {
			return AccountTreeSummary{}, err
		}
		out.Children, _ = out.Children.Add(childSummary.Balance)
		out.AsOf = maxDateString(out.AsOf, childSummary.AsOf)
	}
	if bal, ok, err := s.Balances.LatestByAccount(ctx, a.ID); err != nil {
		return AccountTreeSummary{}, err
	} else if ok {
		cur, err := s.Currency.GetByID(ctx, a.CurrencyID)
		if err != nil {
			return AccountTreeSummary{}, err
		}
		converted, err := money.Convert(bal.Amount, cur.RateToUSD, target.RateToUSD, target.Scale)
		if err != nil {
			return AccountTreeSummary{}, err
		}
		out.Balance = converted
		out.Remaining, _ = out.Balance.Sub(out.Children)
		out.AsOf = maxDateString(out.AsOf, bal.Date)
		out.HasOwnBalance = true
		return out, nil
	}
	out.Balance = out.Children
	out.Remaining = zero
	return out, nil
}

func AccountPath(a repo.Account) string { return fmt.Sprintf("/accounts/%s", a.Name) }

func maxDateString(a, b string) string {
	if b > a {
		return b
	}
	return a
}
