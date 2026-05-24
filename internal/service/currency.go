package service

import (
	"context"

	"stuf/internal/repo"
)

type CurrencyService struct {
	Currencies *repo.CurrencyRepo
}

func (s CurrencyService) Exists(ctx context.Context, code string) bool {
	_, err := s.Currencies.GetByCode(ctx, code)
	return err == nil
}

func (s CurrencyService) Get(ctx context.Context, code string) (repo.Currency, error) {
	return s.Currencies.GetByCode(ctx, code)
}

func (s CurrencyService) List(ctx context.Context) ([]repo.Currency, error) {
	return s.Currencies.List(ctx)
}
