package service

import (
	"context"
	"errors"
	"regexp"
	"time"

	"stuf/internal/money"
	"stuf/internal/repo"
)

var datePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

type BalanceService struct {
	Store    *repo.Store
	Accounts *repo.AccountRepo
	Balances *repo.BalanceRepo
	History  HistoryService
}

func (s BalanceService) Add(ctx context.Context, accountID int64, date, amountText, notes string) (repo.Balance, SessionEntry, error) {
	a, err := s.Accounts.GetByID(ctx, accountID)
	if err != nil {
		return repo.Balance{}, SessionEntry{}, err
	}
	amount, err := parseBalanceAmount(date, amountText, a.Scale)
	if err != nil {
		return repo.Balance{}, SessionEntry{}, err
	}
	var out repo.Balance
	var entry SessionEntry
	err = s.Store.WithWriteLock(func() error {
		b, err := s.Balances.Create(ctx, repo.BalanceCreate{AccountID: accountID, Date: date, Amount: amount, Notes: notes})
		if err != nil {
			return err
		}
		e, err := s.History.Record(ctx, "add", "/accounts/"+a.Name+"/balances/"+b.Date, nil, b, func(ctx context.Context) error {
			return s.Balances.Delete(ctx, b.ID)
		})
		if err != nil {
			return err
		}
		out, entry = b, e
		return nil
	})
	return out, entry, err
}

func (s BalanceService) Update(ctx context.Context, id int64, date, amountText, notes string) (repo.Balance, SessionEntry, error) {
	old, err := s.Balances.GetByID(ctx, id)
	if err != nil {
		return repo.Balance{}, SessionEntry{}, err
	}
	a, err := s.Accounts.GetByID(ctx, old.AccountID)
	if err != nil {
		return repo.Balance{}, SessionEntry{}, err
	}
	amount, err := parseBalanceAmount(date, amountText, a.Scale)
	if err != nil {
		return repo.Balance{}, SessionEntry{}, err
	}
	next := old
	next.Date, next.Amount, next.Notes = date, amount, notes
	var out repo.Balance
	var entry SessionEntry
	err = s.Store.WithWriteLock(func() error {
		updated, err := s.Balances.Update(ctx, next)
		if err != nil {
			return err
		}
		e, err := s.History.Record(ctx, "edit", "/accounts/"+a.Name+"/balances/"+updated.Date, old, updated, func(ctx context.Context) error {
			_, err := s.Balances.Update(ctx, old)
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

func (s BalanceService) Delete(ctx context.Context, id int64) (SessionEntry, error) {
	old, err := s.Balances.GetByID(ctx, id)
	if err != nil {
		return SessionEntry{}, err
	}
	a, err := s.Accounts.GetByID(ctx, old.AccountID)
	if err != nil {
		return SessionEntry{}, err
	}
	var entry SessionEntry
	err = s.Store.WithWriteLock(func() error {
		if err := s.Balances.Delete(ctx, id); err != nil {
			return err
		}
		e, err := s.History.Record(ctx, "delete", "/accounts/"+a.Name+"/balances/"+old.Date, old, nil, func(ctx context.Context) error {
			_, err := s.Balances.Create(ctx, repo.BalanceCreate{AccountID: old.AccountID, Date: old.Date, Amount: old.Amount, Notes: old.Notes})
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

func (s BalanceService) List(ctx context.Context, accountID int64) ([]repo.Balance, error) {
	return s.Balances.ListByAccount(ctx, accountID)
}

func parseBalanceAmount(date, input string, scale int) (money.Money, error) {
	if !datePattern.MatchString(date) {
		return money.Money{}, errors.New("date must be YYYY-MM-DD")
	}
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return money.Money{}, errors.New("date must be a valid YYYY-MM-DD date")
	}
	if input == "" {
		return money.Money{}, errors.New("amount is required")
	}
	return money.NormalizeInput(input, scale)
}
