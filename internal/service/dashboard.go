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
	AsOf                           string
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
	today := s.today()
	appCur, err := s.Currencies.GetByCode(ctx, s.AppCurrency)
	if err != nil {
		return Dashboard{}, err
	}
	accounts, err := s.Accounts.ListRoots(ctx, false)
	if err != nil {
		return Dashboard{}, err
	}
	histories := []dashboardAccountHistory{}
	warnings := []string{}
	warned := map[string]bool{}
	for _, a := range accounts {
		if !a.OnBudget {
			continue
		}
		cur, err := s.Currencies.GetByID(ctx, a.CurrencyID)
		if err != nil {
			return Dashboard{}, err
		}
		accountHistories, accountWarnings, err := s.effectiveAccountHistories(ctx, a, cur, appCur, today)
		if err != nil {
			return Dashboard{}, err
		}
		for _, warning := range accountWarnings {
			if !warned[warning] {
				warnings = append(warnings, warning)
				warned[warning] = true
			}
		}
		histories = append(histories, accountHistories...)
	}
	out := dashboardFromHistories(today, money.Money{Scale: appCur.Scale}, histories)
	out.Warnings = warnings
	return out, nil
}

func (s DashboardService) AccountSummary(ctx context.Context, accountID int64) (Dashboard, error) {
	today := s.today()
	acct, err := s.Accounts.GetByID(ctx, accountID)
	if err != nil {
		return Dashboard{}, err
	}
	cur, err := s.Currencies.GetByID(ctx, acct.CurrencyID)
	if err != nil {
		return Dashboard{}, err
	}
	histories, warnings, err := s.effectiveAccountHistories(ctx, acct, cur, cur, today)
	if err != nil {
		return Dashboard{}, err
	}
	out := dashboardFromHistories(today, money.Money{Scale: cur.Scale}, histories)
	out.Warnings = warnings
	return out, nil
}

func (s DashboardService) today() time.Time {
	now := time.Now
	if s.Now != nil {
		now = s.Now
	}
	return dateOnly(now())
}

func (s DashboardService) accountHistory(ctx context.Context, accountID int64, cur repo.Currency, target repo.Currency, today time.Time) (dashboardAccountHistory, []string, error) {
	bs, err := s.Balances.ListByAccount(ctx, accountID)
	if err != nil {
		return dashboardAccountHistory{}, nil, err
	}
	history := dashboardAccountHistory{}
	var warnings []string
	warned := false
	for _, b := range bs {
		balanceDate, err := parseDashboardDate(b.Date, today.Location())
		if err != nil || balanceDate.After(today) {
			continue
		}
		converted, err := money.Convert(b.Amount, cur.RateToUSD, target.RateToUSD, target.Scale)
		if err != nil {
			if !warned {
				warnings = append(warnings, fmt.Sprintf("missing conversion for %s", cur.Code))
				warned = true
			}
			continue
		}
		history.Balances = append(history.Balances, dashboardBalance{Date: balanceDate, Amount: converted})
	}
	if len(history.Balances) > 0 {
		sort.Slice(history.Balances, func(i, j int) bool {
			return history.Balances[i].Date.Before(history.Balances[j].Date)
		})
	}
	return history, warnings, nil
}

func (s DashboardService) effectiveAccountHistories(ctx context.Context, a repo.Account, cur repo.Currency, target repo.Currency, today time.Time) ([]dashboardAccountHistory, []string, error) {
	history, warnings, err := s.accountHistory(ctx, a.ID, cur, target, today)
	if err != nil {
		return nil, nil, err
	}
	children, err := s.Accounts.ListChildren(ctx, a.ID, false)
	if err != nil {
		return nil, nil, err
	}
	if len(children) == 0 {
		if len(history.Balances) > 0 {
			return []dashboardAccountHistory{history}, warnings, nil
		}
		return nil, warnings, nil
	}
	var histories []dashboardAccountHistory
	warned := map[string]bool{}
	for _, warning := range warnings {
		warned[warning] = true
	}
	for _, child := range children {
		childCur, err := s.Currencies.GetByID(ctx, child.CurrencyID)
		if err != nil {
			return nil, nil, err
		}
		childHistories, childWarnings, err := s.effectiveAccountHistories(ctx, child, childCur, target, today)
		if err != nil {
			return nil, nil, err
		}
		histories = append(histories, childHistories...)
		for _, warning := range childWarnings {
			if !warned[warning] {
				warnings = append(warnings, warning)
				warned[warning] = true
			}
		}
	}
	if len(history.Balances) > 0 {
		histories = append(histories, remainingAccountHistory(history, histories, money.Money{Scale: target.Scale}))
	}
	return histories, warnings, nil
}

func remainingAccountHistory(parent dashboardAccountHistory, children []dashboardAccountHistory, zero money.Money) dashboardAccountHistory {
	remaining := dashboardAccountHistory{}
	for _, balance := range parent.Balances {
		childTotal := totalNearestValue(children, balance.Date, zero)
		amount, _ := balance.Amount.Sub(childTotal)
		remaining.Balances = append(remaining.Balances, dashboardBalance{Date: balance.Date, Amount: amount})
	}
	return remaining
}

func dashboardFromHistories(today time.Time, zero money.Money, histories []dashboardAccountHistory) Dashboard {
	period := today.Format("2006-01")
	out := Dashboard{
		AsOf:                           today.Format("2006-01-02"),
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
	return out
}

func totalLatestValue(histories []dashboardAccountHistory, today time.Time, zero money.Money) money.Money {
	total := zero
	for _, history := range histories {
		value, ok := accountValueAt(history, today)
		if ok {
			total, _ = total.Add(value)
		}
	}
	return total
}

func totalNearestValue(histories []dashboardAccountHistory, target time.Time, zero money.Money) money.Money {
	total := zero
	for _, history := range histories {
		value, ok := accountValueAt(history, target)
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
		carried, carriedOK := accountValueAt(history, monthBoundary(month))
		if !carriedOK {
			return money.Money{}, money.Money{}, false
		}
		return carried, carried, true
	}
	return high, low, true
}

func accountValueAt(history dashboardAccountHistory, target time.Time) (money.Money, bool) {
	if len(history.Balances) == 0 {
		return money.Money{}, false
	}
	if !target.After(history.Balances[0].Date) {
		return history.Balances[0].Amount, true
	}
	last := history.Balances[len(history.Balances)-1]
	if !target.Before(last.Date) {
		return last.Amount, true
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
