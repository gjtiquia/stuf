package repo

import (
	"database/sql"
	"fmt"

	"stuf/internal/db"
	"stuf/internal/money"
)

func accountFromFields(id int64, name string, currencyID int64, code string, scale int64, onBudget, hidden int64, notes, createdAt, updatedAt string) Account {
	return Account{
		ID:         id,
		Name:       name,
		CurrencyID: currencyID,
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
	return accountFromFields(row.ID, row.Name, row.CurrencyID, row.Code, row.Scale, row.OnBudget, row.Hidden, row.Notes, row.CreatedAt, row.UpdatedAt)
}

func accountFromNameRow(row db.GetAccountByNameRow) Account {
	return accountFromFields(row.ID, row.Name, row.CurrencyID, row.Code, row.Scale, row.OnBudget, row.Hidden, row.Notes, row.CreatedAt, row.UpdatedAt)
}

func accountFromListRow(row db.ListAccountsRow) Account {
	return accountFromFields(row.ID, row.Name, row.CurrencyID, row.Code, row.Scale, row.OnBudget, row.Hidden, row.Notes, row.CreatedAt, row.UpdatedAt)
}

func accountFromVisibleRow(row db.ListVisibleAccountsRow) Account {
	return accountFromFields(row.ID, row.Name, row.CurrencyID, row.Code, row.Scale, row.OnBudget, row.Hidden, row.Notes, row.CreatedAt, row.UpdatedAt)
}

func mapAccountErr(err error) error {
	if err == sql.ErrNoRows {
		return fmt.Errorf("account not found")
	}
	return err
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
	return fmt.Errorf("currency not found: %w", err)
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
