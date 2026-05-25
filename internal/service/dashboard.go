package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"stuf/internal/money"
	"stuf/internal/repo"
)

type Dashboard struct {
	Period                         string
	Total                          money.Money
	NetChangeFromMonthStart        money.Money
	NetChangeFromMonthHigh         money.Money
	NetChangeFromPreviousMonthHigh money.Money
	RecentMonths                   []DashboardMonthDrop
	Trend                          DashboardMonthTrend
	Warnings                       []string
}

type DashboardMonthDrop struct {
	Period string
	Drop   money.Money
}

type DashboardMonthTrend struct {
	FromPeriod string
	ToPeriod   string
	HighToHigh money.Money
	LowToLow   money.Money
}

type dashboardAccountHistory struct {
	Balances []dashboardBalance
}

type dashboardBalance struct {
	Date   time.Time
	Amount money.Money
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
	today := dateOnly(now())
	period := today.Format("2006-01")
	appCur, err := s.Currencies.GetByCode(ctx, s.AppCurrency)
	if err != nil {
		return Dashboard{}, err
	}
	accounts, err := s.Accounts.List(ctx, false)
	if err != nil {
		return Dashboard{}, err
	}
	zero := money.Money{Scale: appCur.Scale}
	out := Dashboard{
		Period:                         period,
		Total:                          zero,
		NetChangeFromMonthStart:        zero,
		NetChangeFromMonthHigh:         zero,
		NetChangeFromPreviousMonthHigh: zero,
		RecentMonths: []DashboardMonthDrop{
			{Period: today.AddDate(0, -1, 0).Format("2006-01"), Drop: zero},
			{Period: today.AddDate(0, -2, 0).Format("2006-01"), Drop: zero},
		},
		Trend: DashboardMonthTrend{
			FromPeriod: today.AddDate(0, -2, 0).Format("2006-01"),
			ToPeriod:   today.AddDate(0, -1, 0).Format("2006-01"),
			HighToHigh: zero,
			LowToLow:   zero,
		},
	}
	histories := []dashboardAccountHistory{}
	warned := map[string]bool{}
	for _, a := range accounts {
		if !a.OnBudget {
			continue
		}
		cur, err := s.Currencies.GetByID(ctx, a.CurrencyID)
		if err != nil {
			return Dashboard{}, err
		}
		bs, err := s.Balances.ListByAccount(ctx, a.ID)
		if err != nil {
			return Dashboard{}, err
		}
		history := dashboardAccountHistory{}
		for _, b := range bs {
			balanceDate, err := parseDashboardDate(b.Date, today.Location())
			if err != nil || balanceDate.After(today) {
				continue
			}
			converted, err := money.Convert(b.Amount, cur.RateToUSD, appCur.RateToUSD, appCur.Scale)
			if err != nil {
				if !warned[cur.Code] {
					out.Warnings = append(out.Warnings, fmt.Sprintf("missing conversion for %s", cur.Code))
					warned[cur.Code] = true
				}
				continue
			}
			history.Balances = append(history.Balances, dashboardBalance{Date: balanceDate, Amount: converted})
		}
		if len(history.Balances) > 0 {
			sort.Slice(history.Balances, func(i, j int) bool {
				return history.Balances[i].Date.Before(history.Balances[j].Date)
			})
			histories = append(histories, history)
		}
	}
	out.Total = totalLatestValue(histories, today, zero)
	monthStart := totalNearestValue(histories, monthBoundary(today), zero)
	monthHigh, _ := totalMonthHighLow(histories, today, today, zero)
	prevMonth := today.AddDate(0, -1, 0)
	prevPrevMonth := today.AddDate(0, -2, 0)
	prevHigh, prevLow := totalMonthHighLow(histories, prevMonth, today, zero)
	prevPrevHigh, prevPrevLow := totalMonthHighLow(histories, prevPrevMonth, today, zero)
	out.NetChangeFromMonthStart, _ = out.Total.Sub(monthStart)
	out.NetChangeFromMonthHigh, _ = out.Total.Sub(monthHigh)
	out.NetChangeFromPreviousMonthHigh, _ = out.Total.Sub(prevHigh)
	out.RecentMonths[0].Drop, _ = prevLow.Sub(prevHigh)
	out.RecentMonths[1].Drop, _ = prevPrevLow.Sub(prevPrevHigh)
	out.Trend.HighToHigh, _ = prevHigh.Sub(prevPrevHigh)
	out.Trend.LowToLow, _ = prevLow.Sub(prevPrevLow)
	return out, nil
}

func totalLatestValue(histories []dashboardAccountHistory, today time.Time, zero money.Money) money.Money {
	total := zero
	for _, history := range histories {
		value, ok := latestAccountValue(history, today)
		if ok {
			total, _ = total.Add(value)
		}
	}
	return total
}

func totalNearestValue(histories []dashboardAccountHistory, target time.Time, zero money.Money) money.Money {
	total := zero
	for _, history := range histories {
		value, ok := nearestAccountValue(history, target)
		if ok {
			total, _ = total.Add(value)
		}
	}
	return total
}

func totalMonthHighLow(histories []dashboardAccountHistory, month, today time.Time, zero money.Money) (money.Money, money.Money) {
	totalHigh := zero
	totalLow := zero
	ok := false
	for _, history := range histories {
		high, low, accountOK := accountMonthHighLow(history, month, today)
		if !accountOK {
			continue
		}
		totalHigh, _ = totalHigh.Add(high)
		totalLow, _ = totalLow.Add(low)
		ok = true
	}
	if !ok {
		return zero, zero
	}
	return totalHigh, totalLow
}

func accountMonthHighLow(history dashboardAccountHistory, month, today time.Time) (money.Money, money.Money, bool) {
	var high, low money.Money
	ok := false
	for _, balance := range history.Balances {
		if balance.Date.After(today) || !sameMonth(balance.Date, month) {
			continue
		}
		if !ok || balance.Amount.Amount > high.Amount {
			high = balance.Amount
		}
		if !ok || balance.Amount.Amount < low.Amount {
			low = balance.Amount
		}
		ok = true
	}
	if !ok {
		return money.Money{}, money.Money{}, false
	}
	return high, low, true
}

func latestAccountValue(history dashboardAccountHistory, today time.Time) (money.Money, bool) {
	for i := len(history.Balances) - 1; i >= 0; i-- {
		if !history.Balances[i].Date.After(today) {
			return history.Balances[i].Amount, true
		}
	}
	return money.Money{}, false
}

func nearestAccountValue(history dashboardAccountHistory, target time.Time) (money.Money, bool) {
	if len(history.Balances) == 0 {
		return money.Money{}, false
	}
	best := history.Balances[0]
	bestDist := absDuration(best.Date.Sub(target))
	for _, balance := range history.Balances[1:] {
		dist := absDuration(balance.Date.Sub(target))
		if dist < bestDist || (dist == bestDist && balance.Date.Before(best.Date)) {
			best = balance
			bestDist = dist
		}
	}
	return best.Amount, true
}

func sameMonth(date, month time.Time) bool {
	return date.Format("2006-01") == month.Format("2006-01")
}

func monthBoundary(t time.Time) time.Time {
	y, m, _ := t.Date()
	return time.Date(y, m, 1, 0, 0, 0, 0, t.Location())
}

func parseDashboardDate(date string, loc *time.Location) (time.Time, error) {
	return time.ParseInLocation("2006-01-02", date, loc)
}

func dateOnly(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}
