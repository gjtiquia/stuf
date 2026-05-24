package service

import (
	"context"
	"fmt"
	"time"

	"stuf/internal/money"
	"stuf/internal/repo"
)

type Dashboard struct {
	Period       string
	Total        money.Money
	OnBudgetGrow money.Money
	TotalGrow    money.Money
	Warnings     []string
}

type DashboardService struct {
	Accounts    *repo.AccountRepo
	Balances    *repo.BalanceRepo
	Currencies  *repo.CurrencyRepo
	AppCurrency string
	Now         func() time.Time
}

func (s DashboardService) Summary(ctx context.Context) (Dashboard, error) {
	now := time.Now
	if s.Now != nil {
		now = s.Now
	}
	period := now().Format("2006-01")
	appCur, err := s.Currencies.GetByCode(ctx, s.AppCurrency)
	if err != nil {
		return Dashboard{}, err
	}
	accounts, err := s.Accounts.List(ctx, false)
	if err != nil {
		return Dashboard{}, err
	}
	out := Dashboard{Period: period, Total: money.Money{Scale: appCur.Scale}, OnBudgetGrow: money.Money{Scale: appCur.Scale}, TotalGrow: money.Money{Scale: appCur.Scale}}
	start, _ := time.Parse("2006-01-02", period+"-01")
	end := start.AddDate(0, 1, 0)
	for _, a := range accounts {
		cur, err := s.Currencies.GetByID(ctx, a.CurrencyID)
		if err != nil {
			return Dashboard{}, err
		}
		if b, ok, err := s.Balances.LatestByAccount(ctx, a.ID); err != nil {
			return Dashboard{}, err
		} else if ok {
			converted, err := money.Convert(b.Amount, cur.RateToUSD, appCur.RateToUSD, appCur.Scale)
			if err != nil {
				out.Warnings = append(out.Warnings, fmt.Sprintf("missing conversion for %s", cur.Code))
			} else {
				out.Total, _ = out.Total.Add(converted)
			}
		}
		bs, err := s.Balances.ListByAccount(ctx, a.ID)
		if err != nil {
			return Dashboard{}, err
		}
		growthNative := boundaryValue(bs, end)
		startNative := boundaryValue(bs, start)
		growthNative.Amount -= startNative.Amount
		converted, err := money.Convert(growthNative, cur.RateToUSD, appCur.RateToUSD, appCur.Scale)
		if err == nil {
			out.TotalGrow, _ = out.TotalGrow.Add(converted)
			if a.OnBudget {
				out.OnBudgetGrow, _ = out.OnBudgetGrow.Add(converted)
			}
		}
	}
	return out, nil
}

func boundaryValue(balances []repo.Balance, boundary time.Time) money.Money {
	if len(balances) == 0 {
		return money.Money{Scale: 2}
	}
	best := balances[0]
	bestDist := absDays(best.Date, boundary)
	for _, b := range balances[1:] {
		dist := absDays(b.Date, boundary)
		if dist < bestDist || (dist == bestDist && b.Date < best.Date) {
			best, bestDist = b, dist
		}
	}
	return best.Amount
}

func absDays(date string, boundary time.Time) int {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return 1 << 30
	}
	d := int(t.Sub(boundary).Hours() / 24)
	if d < 0 {
		return -d
	}
	return d
}
