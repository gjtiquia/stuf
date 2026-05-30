package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"stuf/internal/money"
	"stuf/internal/repo"
)

const (
	TransactionTypeIncome  = "income"
	TransactionTypeExpense = "expense"
)

type TransactionService struct {
	Store        *repo.Store
	Transactions *repo.TransactionRepo
	Accounts     *repo.AccountRepo
	Currency     *repo.CurrencyRepo
	Tags         *repo.TagRepo
	History      HistoryService
}

type transactionMutationData struct {
	Transaction repo.Transaction
	Tags        []repo.Tag
}

func (s TransactionService) Add(ctx context.Context, parentID *int64, accountID int64, typ, currencyCode, date, amountText, notes string, tagNames []string) (repo.Transaction, SessionEntry, error) {
	in, err := s.prepare(ctx, parentID, accountID, typ, currencyCode, date, amountText, notes)
	if err != nil {
		return repo.Transaction{}, SessionEntry{}, err
	}
	tagNames, err = normalizeTagNames(tagNames)
	if err != nil {
		return repo.Transaction{}, SessionEntry{}, err
	}
	var out repo.Transaction
	var entry SessionEntry
	err = s.Store.WithWriteTx(ctx, func(tx *repo.Store) error {
		txn, err := tx.Txn.Create(ctx, in)
		if err != nil {
			return err
		}
		tags, createdTags, err := s.resolveTagsWith(ctx, tx.Tag, tagNames)
		if err != nil {
			return err
		}
		if err := tx.Tag.SetTransactionTags(ctx, txn.ID, tagIDs(tags)); err != nil {
			return err
		}
		history := HistoryService{Repo: tx.Hist, Now: s.History.Now}
		e, err := history.Record(ctx, "add", transactionPath(txn), nil, transactionMutationData{Transaction: txn, Tags: tags}, func(ctx context.Context) error {
			if err := s.Tags.SetTransactionTags(ctx, txn.ID, nil); err != nil {
				return err
			}
			if err := s.Transactions.Delete(ctx, txn.ID); err != nil {
				return err
			}
			for _, tag := range createdTags {
				_ = s.Tags.DeleteIfUnused(ctx, tag.ID)
			}
			return nil
		})
		if err != nil {
			return err
		}
		out, entry = txn, e
		return nil
	})
	return out, entry, err
}

func (s TransactionService) Update(ctx context.Context, id int64, date, typ, currencyCode, amountText, notes string, tagNames []string) (repo.Transaction, SessionEntry, error) {
	old, err := s.Transactions.GetByID(ctx, id)
	if err != nil {
		return repo.Transaction{}, SessionEntry{}, err
	}
	oldTags, err := s.Tags.ListByTransactionID(ctx, id)
	if err != nil {
		return repo.Transaction{}, SessionEntry{}, err
	}
	parentID := old.ParentID
	if currencyCode == "" {
		currencyCode = old.Code
	}
	in, err := s.prepare(ctx, parentID, old.AccountID, typ, currencyCode, date, amountText, notes)
	if err != nil {
		return repo.Transaction{}, SessionEntry{}, err
	}
	tagNames, err = normalizeTagNames(tagNames)
	if err != nil {
		return repo.Transaction{}, SessionEntry{}, err
	}
	if tagNames == nil {
		tagNames = tagNamesFromTags(oldTags)
	}
	next := old
	next.ParentID = in.ParentID
	next.AccountID = in.AccountID
	next.Type = in.Type
	next.CurrencyID = in.CurrencyID
	next.Code = currencyCode
	next.Date = in.Date
	next.Amount = in.Amount
	next.Notes = in.Notes
	var out repo.Transaction
	var entry SessionEntry
	err = s.Store.WithWriteTx(ctx, func(tx *repo.Store) error {
		updated, err := tx.Txn.Update(ctx, next)
		if err != nil {
			return err
		}
		tags, createdTags, err := s.resolveTagsWith(ctx, tx.Tag, tagNames)
		if err != nil {
			return err
		}
		if err := tx.Tag.SetTransactionTags(ctx, updated.ID, tagIDs(tags)); err != nil {
			return err
		}
		history := HistoryService{Repo: tx.Hist, Now: s.History.Now}
		e, err := history.Record(ctx, "edit", transactionPath(updated), transactionMutationData{Transaction: old, Tags: oldTags}, transactionMutationData{Transaction: updated, Tags: tags}, func(ctx context.Context) error {
			if _, err := s.Transactions.Update(ctx, old); err != nil {
				return err
			}
			if err := s.Tags.SetTransactionTags(ctx, old.ID, tagIDs(oldTags)); err != nil {
				return err
			}
			for _, tag := range createdTags {
				_ = s.Tags.DeleteIfUnused(ctx, tag.ID)
			}
			return nil
		})
		if err != nil {
			return err
		}
		out, entry = updated, e
		return nil
	})
	return out, entry, err
}

func (s TransactionService) Delete(ctx context.Context, id int64) (SessionEntry, error) {
	old, err := s.Transactions.GetByID(ctx, id)
	if err != nil {
		return SessionEntry{}, err
	}
	hasChildren, err := s.Transactions.HasChildren(ctx, id)
	if err != nil {
		return SessionEntry{}, err
	}
	if hasChildren {
		return SessionEntry{}, errors.New("delete child transactions before deleting parent transaction")
	}
	oldTags, err := s.Tags.ListByTransactionID(ctx, id)
	if err != nil {
		return SessionEntry{}, err
	}
	var entry SessionEntry
	err = s.Store.WithWriteTx(ctx, func(tx *repo.Store) error {
		if err := tx.Tag.SetTransactionTags(ctx, old.ID, nil); err != nil {
			return err
		}
		if err := tx.Txn.Delete(ctx, id); err != nil {
			return err
		}
		history := HistoryService{Repo: tx.Hist, Now: s.History.Now}
		e, err := history.Record(ctx, "delete", transactionPath(old), transactionMutationData{Transaction: old, Tags: oldTags}, nil, func(ctx context.Context) error {
			restored, err := s.Transactions.Create(ctx, repo.TransactionCreate{Ref: old.Ref, ParentID: old.ParentID, AccountID: old.AccountID, Type: old.Type, CurrencyID: old.CurrencyID, Date: old.Date, Amount: old.Amount, Notes: old.Notes})
			if err != nil {
				return err
			}
			return s.Tags.SetTransactionTags(ctx, restored.ID, tagIDs(oldTags))
		})
		if err != nil {
			return err
		}
		entry = e
		return nil
	})
	return entry, err
}

func (s TransactionService) List(ctx context.Context) ([]repo.Transaction, error) {
	return s.Transactions.List(ctx)
}

func (s TransactionService) ListByAccount(ctx context.Context, accountID int64) ([]repo.Transaction, error) {
	return s.Transactions.ListByAccount(ctx, accountID)
}

func (s TransactionService) ListByParent(ctx context.Context, parentID int64) ([]repo.Transaction, error) {
	return s.Transactions.ListByParent(ctx, parentID)
}

func (s TransactionService) GetByID(ctx context.Context, id int64) (repo.Transaction, error) {
	return s.Transactions.GetByID(ctx, id)
}

func (s TransactionService) GetByRef(ctx context.Context, ref int64) (repo.Transaction, error) {
	return s.Transactions.GetByRef(ctx, ref)
}

func (s TransactionService) TagsByTransactionID(ctx context.Context, id int64) ([]repo.Tag, error) {
	return s.Tags.ListByTransactionID(ctx, id)
}

func (s TransactionService) prepare(ctx context.Context, parentID *int64, accountID int64, typ, currencyCode, date, amountText, notes string) (repo.TransactionCreate, error) {
	typ = strings.TrimSpace(strings.ToLower(typ))
	if typ != TransactionTypeIncome && typ != TransactionTypeExpense {
		return repo.TransactionCreate{}, errors.New("transaction type must be income or expense")
	}
	if !datePattern.MatchString(date) {
		return repo.TransactionCreate{}, errors.New("date must be YYYY-MM-DD")
	}
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return repo.TransactionCreate{}, errors.New("date must be a valid YYYY-MM-DD date")
	}
	if parentID != nil {
		parent, err := s.Transactions.GetByID(ctx, *parentID)
		if err != nil {
			return repo.TransactionCreate{}, err
		}
		if typ == "" {
			typ = parent.Type
		}
		if typ != parent.Type {
			return repo.TransactionCreate{}, errors.New("child transaction type must match parent transaction type")
		}
		accountID = parent.AccountID
	}
	acct, err := s.Accounts.GetByID(ctx, accountID)
	if err != nil {
		return repo.TransactionCreate{}, err
	}
	if currencyCode == "" {
		currencyCode = acct.Code
	}
	cur, err := s.Currency.GetByCode(ctx, currencyCode)
	if err != nil {
		return repo.TransactionCreate{}, err
	}
	amount, err := parsePositiveTransactionAmount(date, amountText, cur.Scale)
	if err != nil {
		return repo.TransactionCreate{}, err
	}
	return repo.TransactionCreate{ParentID: parentID, AccountID: accountID, Type: typ, CurrencyID: cur.ID, Date: date, Amount: amount, Notes: notes}, nil
}

func (s TransactionService) resolveTagsWith(ctx context.Context, tagsRepo *repo.TagRepo, names []string) ([]repo.Tag, []repo.Tag, error) {
	var tags []repo.Tag
	var created []repo.Tag
	for _, name := range names {
		tag, err := tagsRepo.GetByName(ctx, name)
		if err != nil {
			tag, err = tagsRepo.Create(ctx, name, "")
			if err != nil {
				return nil, nil, err
			}
			created = append(created, tag)
		}
		tags = append(tags, tag)
	}
	return tags, created, nil
}

func parsePositiveTransactionAmount(date, input string, scale int) (money.Money, error) {
	amount, err := parseBalanceAmount(date, input, scale)
	if err != nil {
		return money.Money{}, err
	}
	if amount.Amount <= 0 {
		return money.Money{}, errors.New("transaction amount must be positive")
	}
	return amount, nil
}

func TransactionRef(ref int64) string {
	return fmt.Sprintf("tx-%06d", ref)
}

func transactionPath(t repo.Transaction) string {
	return "/transactions/" + TransactionRef(t.Ref)
}
