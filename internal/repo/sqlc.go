package repo

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"stuf/internal/db"
	"stuf/internal/money"

	"modernc.org/sqlite"
)

func accountFromFields(id int64, name string, currencyID int64, parentID sql.NullInt64, code string, scale int64, onBudget, hidden int64, notes, createdAt, updatedAt string) Account {
	return Account{
		ID:         id,
		Name:       name,
		CurrencyID: currencyID,
		ParentID:   ptrFromNullInt64(parentID),
		Code:       code,
		Scale:      int(scale),
		OnBudget:   onBudget == 1,
		Hidden:     hidden == 1,
		Notes:      notes,
		CreatedAt:  createdAt,
		UpdatedAt:  updatedAt,
	}
}

func accountFromGetRow(row db.GetAccountByIDRow) Account {
	return accountFromFields(row.ID, row.Name, row.CurrencyID, row.ParentID, row.Code, row.Scale, row.OnBudget, row.Hidden, row.Notes, row.CreatedAt, row.UpdatedAt)
}

func accountFromNameRow(row db.GetAccountByNameRow) Account {
	return accountFromFields(row.ID, row.Name, row.CurrencyID, row.ParentID, row.Code, row.Scale, row.OnBudget, row.Hidden, row.Notes, row.CreatedAt, row.UpdatedAt)
}

func accountFromListRow(row db.ListAccountsRow) Account {
	return accountFromFields(row.ID, row.Name, row.CurrencyID, row.ParentID, row.Code, row.Scale, row.OnBudget, row.Hidden, row.Notes, row.CreatedAt, row.UpdatedAt)
}

func accountFromVisibleRow(row db.ListVisibleAccountsRow) Account {
	return accountFromFields(row.ID, row.Name, row.CurrencyID, row.ParentID, row.Code, row.Scale, row.OnBudget, row.Hidden, row.Notes, row.CreatedAt, row.UpdatedAt)
}

func accountFromRootRow(row db.ListRootAccountsRow) Account {
	return accountFromFields(row.ID, row.Name, row.CurrencyID, row.ParentID, row.Code, row.Scale, row.OnBudget, row.Hidden, row.Notes, row.CreatedAt, row.UpdatedAt)
}

func accountFromVisibleRootRow(row db.ListVisibleRootAccountsRow) Account {
	return accountFromFields(row.ID, row.Name, row.CurrencyID, row.ParentID, row.Code, row.Scale, row.OnBudget, row.Hidden, row.Notes, row.CreatedAt, row.UpdatedAt)
}

func accountFromChildRow(row db.ListChildAccountsRow) Account {
	return accountFromFields(row.ID, row.Name, row.CurrencyID, row.ParentID, row.Code, row.Scale, row.OnBudget, row.Hidden, row.Notes, row.CreatedAt, row.UpdatedAt)
}

func accountFromVisibleChildRow(row db.ListVisibleChildAccountsRow) Account {
	return accountFromFields(row.ID, row.Name, row.CurrencyID, row.ParentID, row.Code, row.Scale, row.OnBudget, row.Hidden, row.Notes, row.CreatedAt, row.UpdatedAt)
}

func accountFromDescendantRow(row db.ListDescendantAccountsRow) Account {
	return accountFromFields(row.ID, row.Name, row.CurrencyID, row.ParentID, row.Code, row.Scale, row.OnBudget, row.Hidden, row.Notes, row.CreatedAt, row.UpdatedAt)
}

func mapAccountErr(err error) error {
	if err == sql.ErrNoRows {
		return fmt.Errorf("account not found")
	}
	return err
}

func tagFromDB(t db.Tag) Tag {
	return Tag{
		ID:        t.ID,
		Name:      t.Name,
		Notes:     t.Notes,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}

func mapTagErr(err error) error {
	if err == sql.ErrNoRows {
		return fmt.Errorf("tag not found")
	}
	return err
}

func mapTagWriteErr(err error, name string) error {
	if isTagDuplicateNameErr(err) {
		return &TagDuplicateNameError{Name: name}
	}
	return err
}

func isTagDuplicateNameErr(err error) bool {
	var sqliteErr *sqlite.Error
	if errors.As(err, &sqliteErr) {
		return sqliteErr.Code() == 2067 && strings.Contains(sqliteErr.Error(), "tags.name")
	}
	return strings.Contains(err.Error(), "UNIQUE constraint failed: tags.name")
}

func budgetCategoryFromDB(c db.BudgetCategory) BudgetCategory {
	return BudgetCategory{
		ID:        c.ID,
		Name:      c.Name,
		Notes:     c.Notes,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func mapBudgetCategoryErr(err error) error {
	if err == sql.ErrNoRows {
		return fmt.Errorf("budget category not found")
	}
	return err
}

func mapBudgetCategoryWriteErr(err error, name string) error {
	if isBudgetCategoryDuplicateNameErr(err) {
		return &BudgetCategoryDuplicateNameError{Name: name}
	}
	return err
}

func isBudgetCategoryDuplicateNameErr(err error) bool {
	var sqliteErr *sqlite.Error
	if errors.As(err, &sqliteErr) {
		return sqliteErr.Code() == 2067 && strings.Contains(sqliteErr.Error(), "budget_categories.name")
	}
	return strings.Contains(err.Error(), "UNIQUE constraint failed: budget_categories.name")
}

func budgetFromFields(id int64, name string, currencyID, categoryID int64, categoryName, code string, scale, hidden int64, notes, createdAt, updatedAt string) Budget {
	return Budget{
		ID:           id,
		Name:         name,
		CurrencyID:   currencyID,
		CategoryID:   categoryID,
		CategoryName: categoryName,
		Code:         code,
		Scale:        int(scale),
		Hidden:       hidden == 1,
		Notes:        notes,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}
}

func budgetFromGetRow(row db.GetBudgetByIDRow) Budget {
	return budgetFromFields(row.ID, row.Name, row.CurrencyID, row.CategoryID, row.CategoryName, row.Code, row.Scale, row.Hidden, row.Notes, row.CreatedAt, row.UpdatedAt)
}

func budgetFromNameRow(row db.GetBudgetByNameRow) Budget {
	return budgetFromFields(row.ID, row.Name, row.CurrencyID, row.CategoryID, row.CategoryName, row.Code, row.Scale, row.Hidden, row.Notes, row.CreatedAt, row.UpdatedAt)
}

func budgetFromListRow(row db.ListBudgetsRow) Budget {
	return budgetFromFields(row.ID, row.Name, row.CurrencyID, row.CategoryID, row.CategoryName, row.Code, row.Scale, row.Hidden, row.Notes, row.CreatedAt, row.UpdatedAt)
}

func budgetFromVisibleRow(row db.ListVisibleBudgetsRow) Budget {
	return budgetFromFields(row.ID, row.Name, row.CurrencyID, row.CategoryID, row.CategoryName, row.Code, row.Scale, row.Hidden, row.Notes, row.CreatedAt, row.UpdatedAt)
}

func budgetFromCategoryRow(row db.ListBudgetsByCategoryIDRow) Budget {
	return budgetFromFields(row.ID, row.Name, row.CurrencyID, row.CategoryID, row.CategoryName, row.Code, row.Scale, row.Hidden, row.Notes, row.CreatedAt, row.UpdatedAt)
}

func mapBudgetErr(err error) error {
	if err == sql.ErrNoRows {
		return fmt.Errorf("budget not found")
	}
	return err
}

func mapBudgetWriteErr(err error, name string) error {
	if isBudgetDuplicateNameErr(err) {
		return &BudgetDuplicateNameError{Name: name}
	}
	return err
}

func isBudgetDuplicateNameErr(err error) bool {
	var sqliteErr *sqlite.Error
	if errors.As(err, &sqliteErr) {
		return sqliteErr.Code() == 2067 && strings.Contains(sqliteErr.Error(), "budgets.name")
	}
	return strings.Contains(err.Error(), "UNIQUE constraint failed: budgets.name")
}

func currencyFromFields(id int64, code, name string, scale int64, amount, rateScale sql.NullInt64, updated sql.NullString) Currency {
	c := Currency{
		ID:    id,
		Code:  code,
		Name:  name,
		Scale: int(scale),
	}
	if amount.Valid && rateScale.Valid {
		c.RateToUSD = money.Money{Amount: amount.Int64, Scale: int(rateScale.Int64)}
	}
	c.RateUpdatedAt = updated.String
	return c
}

func currencyFromCodeRow(row db.GetCurrencyByCodeRow) Currency {
	return currencyFromFields(row.ID, row.Code, row.Name, row.Scale, row.RateToUsdAmount, row.RateToUsdScale, row.UpdatedAt)
}

func currencyFromIDRow(row db.GetCurrencyByIDRow) Currency {
	return currencyFromFields(row.ID, row.Code, row.Name, row.Scale, row.RateToUsdAmount, row.RateToUsdScale, row.UpdatedAt)
}

func currencyFromListRow(row db.ListCurrenciesRow) Currency {
	return currencyFromFields(row.ID, row.Code, row.Name, row.Scale, row.RateToUsdAmount, row.RateToUsdScale, row.UpdatedAt)
}

func mapCurrencyErr(err error) error {
	if err == sql.ErrNoRows {
		return &CurrencyUnavailableError{}
	}
	return fmt.Errorf("currency not found: %w", err)
}

func mapCurrencyErrWithCode(err error, code string) error {
	if err == sql.ErrNoRows {
		return &CurrencyUnavailableError{Code: code}
	}
	return mapCurrencyErr(err)
}

func balanceFromDB(b db.Balance) Balance {
	return Balance{
		ID:        b.ID,
		AccountID: b.AccountID,
		Date:      b.Date,
		Amount:    money.Money{Amount: b.Amount, Scale: int(b.Scale)},
		Notes:     b.Notes,
		CreatedAt: b.CreatedAt,
		UpdatedAt: b.UpdatedAt,
	}
}

func budgetAllocationFromDB(a db.BudgetAllocation) BudgetAllocation {
	return BudgetAllocation{
		ID:        a.ID,
		BudgetID:  a.BudgetID,
		Date:      a.Date,
		Amount:    money.Money{Amount: a.Amount, Scale: int(a.Scale)},
		Notes:     a.Notes,
		CreatedAt: a.CreatedAt,
		UpdatedAt: a.UpdatedAt,
	}
}

func mapBudgetAllocationErr(err error) error {
	if err == sql.ErrNoRows {
		return fmt.Errorf("budget allocation not found")
	}
	return err
}

func mapBalanceErr(err error) error {
	if err == sql.ErrNoRows {
		return fmt.Errorf("balance not found")
	}
	return err
}

func historyFromDB(h db.History) History {
	out := History{
		ID:        h.ID,
		Timestamp: h.Timestamp,
		Action:    h.Action,
		Path:      h.Path,
	}
	if h.OldData.Valid {
		s := h.OldData.String
		out.OldData = &s
	}
	if h.NewData.Valid {
		s := h.NewData.String
		out.NewData = &s
	}
	return out
}

func nullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

func nullInt64(v *int64) sql.NullInt64 {
	if v == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *v, Valid: true}
}

func ptrFromNullInt64(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	out := v.Int64
	return &out
}
