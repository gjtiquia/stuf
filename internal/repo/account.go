package repo

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"stuf/internal/db"

	"modernc.org/sqlite"
)

type AccountRepo struct{ store *Store }

type AccountCreate struct {
	Name       string
	CurrencyID int64
	ParentID   *int64
	OnBudget   bool
	Hidden     bool
	Notes      string
}

func (r *AccountRepo) Create(ctx context.Context, in AccountCreate) (Account, error) {
	now := r.store.Clock().UTC().Format(time.RFC3339)
	res, err := r.store.Q.CreateAccount(ctx, db.CreateAccountParams{
		Name:       in.Name,
		CurrencyID: in.CurrencyID,
		OnBudget:   boolInt(in.OnBudget),
		Hidden:     boolInt(in.Hidden),
		Notes:      in.Notes,
		ParentID:   nullInt64(in.ParentID),
		CreatedAt:  now,
		UpdatedAt:  now,
	})
	if err != nil {
		return Account{}, mapAccountWriteErr(err, in.Name)
	}
	id, _ := res.LastInsertId()
	return r.GetByID(ctx, id)
}

func (r *AccountRepo) GetByID(ctx context.Context, id int64) (Account, error) {
	row, err := r.store.Q.GetAccountByID(ctx, id)
	if err != nil {
		return Account{}, mapAccountErr(err)
	}
	return accountFromGetRow(row), nil
}

func (r *AccountRepo) GetByName(ctx context.Context, name string) (Account, error) {
	row, err := r.store.Q.GetAccountByName(ctx, name)
	if err != nil {
		return Account{}, mapAccountErr(err)
	}
	return accountFromNameRow(row), nil
}

func (r *AccountRepo) List(ctx context.Context, includeHidden bool) ([]Account, error) {
	if includeHidden {
		rows, err := r.store.Q.ListAccounts(ctx)
		if err != nil {
			return nil, err
		}
		out := make([]Account, len(rows))
		for i, row := range rows {
			out[i] = accountFromListRow(row)
		}
		return out, nil
	}
	rows, err := r.store.Q.ListVisibleAccounts(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Account, len(rows))
	for i, row := range rows {
		out[i] = accountFromVisibleRow(row)
	}
	return out, nil
}

func (r *AccountRepo) Update(ctx context.Context, a Account) (Account, error) {
	now := r.store.Clock().UTC().Format(time.RFC3339)
	if err := r.store.Q.UpdateAccount(ctx, db.UpdateAccountParams{
		Name:       a.Name,
		CurrencyID: a.CurrencyID,
		OnBudget:   boolInt(a.OnBudget),
		Hidden:     boolInt(a.Hidden),
		Notes:      a.Notes,
		ParentID:   nullInt64(a.ParentID),
		UpdatedAt:  now,
		ID:         a.ID,
	}); err != nil {
		return Account{}, mapAccountWriteErr(err, a.Name)
	}
	return r.GetByID(ctx, a.ID)
}

func (r *AccountRepo) Delete(ctx context.Context, id int64) error {
	return r.store.Q.DeleteAccount(ctx, id)
}

func (r *AccountRepo) HasBalances(ctx context.Context, id int64) (bool, error) {
	n, err := r.store.Q.CountBalancesByAccountID(ctx, id)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *AccountRepo) HasChildren(ctx context.Context, id int64) (bool, error) {
	n, err := r.store.Q.CountChildrenByAccountID(ctx, accountIDParam(id))
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *AccountRepo) IsEmpty(ctx context.Context, id int64) (bool, error) {
	hasBalances, err := r.HasBalances(ctx, id)
	if err != nil {
		return false, err
	}
	if hasBalances {
		return false, nil
	}
	hasChildren, err := r.HasChildren(ctx, id)
	if err != nil {
		return false, err
	}
	return !hasChildren, nil
}

func (r *AccountRepo) ListRoots(ctx context.Context, includeHidden bool) ([]Account, error) {
	if includeHidden {
		rows, err := r.store.Q.ListRootAccounts(ctx)
		if err != nil {
			return nil, err
		}
		out := make([]Account, len(rows))
		for i, row := range rows {
			out[i] = accountFromRootRow(row)
		}
		return out, nil
	}
	rows, err := r.store.Q.ListVisibleRootAccounts(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Account, len(rows))
	for i, row := range rows {
		out[i] = accountFromVisibleRootRow(row)
	}
	return out, nil
}

func (r *AccountRepo) ListChildren(ctx context.Context, id int64, includeHidden bool) ([]Account, error) {
	if includeHidden {
		rows, err := r.store.Q.ListChildAccounts(ctx, accountIDParam(id))
		if err != nil {
			return nil, err
		}
		out := make([]Account, len(rows))
		for i, row := range rows {
			out[i] = accountFromChildRow(row)
		}
		return out, nil
	}
	rows, err := r.store.Q.ListVisibleChildAccounts(ctx, accountIDParam(id))
	if err != nil {
		return nil, err
	}
	out := make([]Account, len(rows))
	for i, row := range rows {
		out[i] = accountFromVisibleChildRow(row)
	}
	return out, nil
}

func (r *AccountRepo) ListDescendants(ctx context.Context, id int64) ([]Account, error) {
	rows, err := r.store.Q.ListDescendantAccounts(ctx, accountIDParam(id))
	if err != nil {
		return nil, err
	}
	out := make([]Account, len(rows))
	for i, row := range rows {
		out[i] = accountFromDescendantRow(row)
	}
	return out, nil
}

func boolInt(v bool) int64 {
	if v {
		return 1
	}
	return 0
}

func accountIDParam(v int64) sql.NullInt64 { return sql.NullInt64{Int64: v, Valid: true} }

func mapAccountWriteErr(err error, name string) error {
	if isAccountDuplicateNameErr(err) {
		return &AccountDuplicateNameError{Name: name}
	}
	return err
}

func isAccountDuplicateNameErr(err error) bool {
	var sqliteErr *sqlite.Error
	if errors.As(err, &sqliteErr) {
		return sqliteErr.Code() == 2067 && strings.Contains(sqliteErr.Error(), "accounts.name")
	}
	return strings.Contains(err.Error(), "UNIQUE constraint failed: accounts.name")
}
