package repo

import (
	"context"
	"database/sql"
	"fmt"

	"stuf/internal/money"
)

type CurrencyRepo struct{ store *Store }

func (r *CurrencyRepo) GetByCode(ctx context.Context, code string) (Currency, error) {
	row := r.store.DB.QueryRowContext(ctx, `SELECT c.id, c.code, c.name, c.scale, cr.rate_to_usd_amount, cr.rate_to_usd_scale, cr.updated_at
		FROM currencies c LEFT JOIN currency_rates cr ON cr.currency_id = c.id WHERE c.code=?`, code)
	return scanCurrency(row)
}

func (r *CurrencyRepo) GetByID(ctx context.Context, id int64) (Currency, error) {
	row := r.store.DB.QueryRowContext(ctx, `SELECT c.id, c.code, c.name, c.scale, cr.rate_to_usd_amount, cr.rate_to_usd_scale, cr.updated_at
		FROM currencies c LEFT JOIN currency_rates cr ON cr.currency_id = c.id WHERE c.id=?`, id)
	return scanCurrency(row)
}

func (r *CurrencyRepo) List(ctx context.Context) ([]Currency, error) {
	rows, err := r.store.DB.QueryContext(ctx, `SELECT c.id, c.code, c.name, c.scale, cr.rate_to_usd_amount, cr.rate_to_usd_scale, cr.updated_at
		FROM currencies c LEFT JOIN currency_rates cr ON cr.currency_id = c.id ORDER BY c.code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Currency
	for rows.Next() {
		c, err := scanCurrency(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

type currencyScanner interface {
	Scan(dest ...any) error
}

func scanCurrency(row currencyScanner) (Currency, error) {
	var c Currency
	var amount sql.NullInt64
	var scale sql.NullInt64
	var updated sql.NullString
	if err := row.Scan(&c.ID, &c.Code, &c.Name, &c.Scale, &amount, &scale, &updated); err != nil {
		return Currency{}, fmt.Errorf("currency not found: %w", err)
	}
	if amount.Valid && scale.Valid {
		c.RateToUSD = money.Money{Amount: amount.Int64, Scale: int(scale.Int64)}
	}
	c.RateUpdatedAt = updated.String
	return c, nil
}
