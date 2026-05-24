package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"

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

func ValidateSlug(name string) error {
	if !slugPattern.MatchString(name) {
		return errors.New("account name must be a strict slug: lowercase letters, digits, and hyphens only")
	}
	return nil
}

func (s AccountService) Create(ctx context.Context, name, currencyCode string, onBudget bool, notes string) (repo.Account, SessionEntry, error) {
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
		a, err := s.Accounts.Create(ctx, repo.AccountCreate{Name: name, CurrencyID: cur.ID, OnBudget: onBudget, Notes: notes})
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

func (s AccountService) GetByName(ctx context.Context, name string) (repo.Account, error) {
	return s.Accounts.GetByName(ctx, name)
}

func (s AccountService) CurrentBalance(ctx context.Context, accountID int64) (repo.Balance, bool, error) {
	return s.Balances.LatestByAccount(ctx, accountID)
}

func AccountPath(a repo.Account) string { return fmt.Sprintf("/accounts/%s", a.Name) }
