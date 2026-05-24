package repo

import "stuf/internal/money"

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
	Code       string
	Scale      int
	OnBudget   bool
	Hidden     bool
	Notes      string
	CreatedAt  string
	UpdatedAt  string
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

type History struct {
	ID        int64
	Timestamp string
	Action    string
	Path      string
	OldData   *string
	NewData   *string
}
