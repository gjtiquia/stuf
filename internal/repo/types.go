package repo

import (
	"fmt"

	"stuf/internal/money"
)

type Currency struct {
	ID            int64
	Code          string
	Name          string
	Scale         int
	RateToUSD     money.Money
	RateUpdatedAt string
}

type Account struct {
	ID         int64
	Name       string
	CurrencyID int64
	ParentID   *int64
	Code       string
	Scale      int
	OnBudget   bool
	Hidden     bool
	Notes      string
	CreatedAt  string
	UpdatedAt  string
}

type Tag struct {
	ID        int64
	Name      string
	Notes     string
	CreatedAt string
	UpdatedAt string
}

type TagDuplicateNameError struct {
	Name string
}

func (e *TagDuplicateNameError) Error() string {
	if e.Name == "" {
		return "tag already exists; choose another name"
	}
	return fmt.Sprintf("tag already exists: %s; choose another name", e.Name)
}

type AccountDuplicateNameError struct {
	Name string
}

func (e *AccountDuplicateNameError) Error() string {
	if e.Name == "" {
		return "account already exists; choose another name"
	}
	return fmt.Sprintf("account already exists: %s; choose another name", e.Name)
}

type CurrencyUnavailableError struct {
	Code string
}

func (e *CurrencyUnavailableError) Error() string {
	if e.Code == "" {
		return "currency is unavailable"
	}
	return fmt.Sprintf("currency is unavailable: %s", e.Code)
}

type Balance struct {
	ID        int64
	AccountID int64
	Date      string
	Amount    money.Money
	Notes     string
	CreatedAt string
	UpdatedAt string
}

type BalanceDuplicateDateError struct {
	Date string
}

func (e *BalanceDuplicateDateError) Error() string {
	if e.Date == "" {
		return "balance already exists for that date; edit the existing balance instead"
	}
	return fmt.Sprintf("balance already exists for %s; edit the existing balance instead", e.Date)
}

type History struct {
	ID        int64
	Timestamp string
	Action    string
	Path      string
	OldData   *string
	NewData   *string
}
