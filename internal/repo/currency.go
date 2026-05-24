package repo

import (
	"context"
)

type CurrencyRepo struct{ store *Store }

func (r *CurrencyRepo) GetByCode(ctx context.Context, code string) (Currency, error) {
	row, err := r.store.Q.GetCurrencyByCode(ctx, code)
	if err != nil {
		return Currency{}, mapCurrencyErr(err)
	}
	return currencyFromCodeRow(row), nil
}

func (r *CurrencyRepo) GetByID(ctx context.Context, id int64) (Currency, error) {
	row, err := r.store.Q.GetCurrencyByID(ctx, id)
	if err != nil {
		return Currency{}, mapCurrencyErr(err)
	}
	return currencyFromIDRow(row), nil
}

func (r *CurrencyRepo) List(ctx context.Context) ([]Currency, error) {
	rows, err := r.store.Q.ListCurrencies(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Currency, len(rows))
	for i, row := range rows {
		out[i] = currencyFromListRow(row)
	}
	return out, nil
}
