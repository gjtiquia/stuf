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
	AsOfStale                      bool
	Period                         string
	Total                          money.Money
	Budgeted                       money.Money
	Available                      money.Money
	PplOweYou                      money.Money
	NetChangeFromMonthStart        money.Money
	NetChangeFromMonthHigh         money.Money
	NetChangeFromPreviousMonthHigh money.Money
	RecentMonths                   []DashboardMonthDrop
	HighTrends                     []DashboardMonthTrendPoint
	LowTrends                      []DashboardMonthTrendPoint
	NetChanges                     []DashboardMonthNetChange
	HighToLows                     []DashboardMonthDrawdown
	Lows                           []DashboardMonthLow
	Warnings                       []string
}

type DashboardMonthDrop struct {
	Period string
	Drop   money.Money
}

type DashboardMonthTrendPoint struct {
	FromPeriod string
	ToPeriod   string
	Change     money.Money
}

type DashboardMonthNetChange struct {
	Period string
	Change money.Money
}

type DashboardMonthDrawdown struct {
	Period string
	Drop   money.Money
}

type DashboardMonthLow struct {
	Period string
	Low    money.Money
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
	Budgets     *repo.BudgetRepo
	Allocations *repo.BudgetAllocationRepo
	OwedLedgers *repo.OwedLedgerRepo
	OwedTxns    *repo.OwedTransactionRepo
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
	if s.Budgets != nil && s.Allocations != nil {
		budgeted, budgetWarnings, err := s.budgetedTotal(ctx, appCur, today)
		if err != nil {
			return Dashboard{}, err
		}
		out.Budgeted = budgeted
		out.Available, _ = out.Total.Sub(budgeted)
		for _, warning := range budgetWarnings {
			if !warned[warning] {
				warnings = append(warnings, warning)
				warned[warning] = true
			}
		}
	} else {
		out.Budgeted = money.Money{Scale: appCur.Scale}
		out.Available = out.Total
	}
	out.PplOweYou = money.Money{Scale: appCur.Scale}
	if s.OwedLedgers != nil && s.OwedTxns != nil {
		owedSvc := OwedTransactionService{Ledgers: s.OwedLedgers, Transactions: s.OwedTxns, Currency: s.Currencies}
		owedTotal, owedWarnings, err := owedSvc.NetTotal(ctx, s.AppCurrency)
		if err != nil {
			return Dashboard{}, err
		}
		out.PplOweYou = owedTotal
		for _, warning := range owedWarnings {
			if !warned[warning] {
				warnings = append(warnings, warning)
				warned[warning] = true
			}
		}
	}
	out.Warnings = warnings
	return out, nil
}

func (s DashboardService) SummaryForAccounts(ctx context.Context, accountIDs []int64) (Dashboard, error) {
	today := s.today()
	appCur, err := s.Currencies.GetByCode(ctx, s.AppCurrency)
	if err != nil {
		return Dashboard{}, err
	}
	histories := []dashboardAccountHistory{}
	warnings := []string{}
	warned := map[string]bool{}
	seen := map[int64]bool{}
	for _, accountID := range accountIDs {
		if seen[accountID] {
			continue
		}
		seen[accountID] = true
		a, err := s.Accounts.GetByID(ctx, accountID)
		if err != nil {
			return Dashboard{}, err
		}
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
	out.Budgeted = money.Money{Scale: appCur.Scale}
	out.Available = out.Total
	out.PplOweYou = money.Money{Scale: appCur.Scale}
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
	out.Budgeted = money.Money{Scale: cur.Scale}
	out.Available = out.Total
	out.PplOweYou = money.Money{Scale: cur.Scale}
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

func (s DashboardService) budgetedTotal(ctx context.Context, appCur repo.Currency, today time.Time) (money.Money, []string, error) {
	budgets, err := s.Budgets.List(ctx, false)
	if err != nil {
		return money.Money{}, nil, err
	}
	total := money.Money{Scale: appCur.Scale}
	warnings := []string{}
	warned := map[string]bool{}
	for _, budget := range budgets {
		cur, err := s.Currencies.GetByID(ctx, budget.CurrencyID)
		if err != nil {
			return money.Money{}, nil, err
		}
		balance, err := s.budgetBalanceOn(ctx, budget.ID, today, money.Money{Scale: budget.Scale})
		if err != nil {
			return money.Money{}, nil, err
		}
		converted, err := money.Convert(balance, cur.RateToUSD, appCur.RateToUSD, appCur.Scale)
		if err != nil {
			warning := fmt.Sprintf("missing conversion for %s", cur.Code)
			if !warned[warning] {
				warnings = append(warnings, warning)
				warned[warning] = true
			}
			continue
		}
		total, err = total.Add(converted)
		if err != nil {
			return money.Money{}, nil, err
		}
	}
	return total, warnings, nil
}

func (s DashboardService) budgetBalanceOn(ctx context.Context, budgetID int64, today time.Time, zero money.Money) (money.Money, error) {
	allocs, err := s.Allocations.ListByBudget(ctx, budgetID)
	if err != nil {
		return money.Money{}, err
	}
	total := zero
	for _, alloc := range allocs {
		allocationDate, err := parseDashboardDate(alloc.Date, today.Location())
		if err != nil || allocationDate.After(today) {
			continue
		}
		total, err = total.Add(alloc.Amount)
		if err != nil {
			return money.Money{}, err
		}
	}
	return total, nil
}

func remainingAccountHistory(parent dashboardAccountHistory, children []dashboardAccountHistory, zero money.Money) dashboardAccountHistory {
	remaining := dashboardAccountHistory{}
	for _, balance := range parent.Balances {
		childTotal := totalAsOfValue(children, balance.Date, zero)
		amount, _ := balance.Amount.Sub(childTotal)
		remaining.Balances = append(remaining.Balances, dashboardBalance{Date: balance.Date, Amount: amount})
	}
	return remaining
}

func dashboardFromHistories(today time.Time, zero money.Money, histories []dashboardAccountHistory) Dashboard {
	period := today.Format("2006-01")
	asOfDate, hasAsOf := latestHistoryDate(histories)
	calcDate := today
	asOf := "none"
	asOfStale := true
	if hasAsOf {
		calcDate = asOfDate
		asOf = asOfDate.Format("2006-01-02")
		asOfStale = asOfDate.Before(today)
	}
	out := Dashboard{
		AsOf:                           asOf,
		AsOfStale:                      asOfStale,
		Period:                         period,
		Total:                          zero,
		NetChangeFromMonthStart:        zero,
		NetChangeFromMonthHigh:         zero,
		NetChangeFromPreviousMonthHigh: zero,
	}
	out.Total = totalAsOfValue(histories, calcDate, zero)
	monthStart := totalAsOfValue(histories, monthBoundary(today), zero)
	months := recentMonthBoundaries(today, 4)
	highs := make([]money.Money, len(months))
	lows := make([]money.Money, len(months))
	for i, month := range months {
		highs[i], lows[i] = totalMonthHighLow(histories, month, calcDate, zero)
	}
	out.NetChangeFromMonthStart, _ = out.Total.Sub(monthStart)
	out.NetChangeFromMonthHigh, _ = out.Total.Sub(highs[0])
	out.NetChangeFromPreviousMonthHigh, _ = out.Total.Sub(highs[1])
	for i := 0; i < 3; i++ {
		drop, _ := lows[i].Sub(highs[i])
		startValue := totalAsOfValue(histories, monthBoundary(months[i]), zero)
		endValue := totalAsOfValue(histories, dashboardMonthEnd(months[i], today, calcDate), zero)
		change, _ := endValue.Sub(startValue)
		period := months[i].Format("2006-01")
		out.NetChanges = append(out.NetChanges, DashboardMonthNetChange{
			Period: period,
			Change: change,
		})
		out.HighToLows = append(out.HighToLows, DashboardMonthDrawdown{
			Period: period,
			Drop:   drop,
		})
		out.Lows = append(out.Lows, DashboardMonthLow{
			Period: period,
			Low:    lows[i],
		})
		out.RecentMonths = append(out.RecentMonths, DashboardMonthDrop{
			Period: period,
			Drop:   drop,
		})
		highChange, _ := highs[i].Sub(highs[i+1])
		lowChange, _ := lows[i].Sub(lows[i+1])
		out.HighTrends = append(out.HighTrends, DashboardMonthTrendPoint{
			FromPeriod: months[i+1].Format("2006-01"),
			ToPeriod:   months[i].Format("2006-01"),
			Change:     highChange,
		})
		out.LowTrends = append(out.LowTrends, DashboardMonthTrendPoint{
			FromPeriod: months[i+1].Format("2006-01"),
			ToPeriod:   months[i].Format("2006-01"),
			Change:     lowChange,
		})
	}
	return out
}

func totalAsOfValue(histories []dashboardAccountHistory, target time.Time, zero money.Money) money.Money {
	total := zero
	for _, history := range histories {
		value, ok := accountAsOfValue(history, target)
		if ok {
			total, _ = total.Add(value)
		}
	}
	return total
}

func latestHistoryDate(histories []dashboardAccountHistory) (time.Time, bool) {
	var latest time.Time
	ok := false
	for _, history := range histories {
		for _, balance := range history.Balances {
			if !ok || balance.Date.After(latest) {
				latest = balance.Date
				ok = true
			}
		}
	}
	return latest, ok
}

func dashboardMonthEnd(month, today, asOf time.Time) time.Time {
	start := monthBoundary(month)
	if sameMonth(month, today) {
		if asOf.Before(start) {
			return start
		}
		return asOf
	}
	return start.AddDate(0, 1, -1)
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
	monthStart := monthBoundary(month)
	carried, ok := accountAsOfValue(history, monthStart)
	if ok {
		high = carried
		low = carried
	}
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

func accountAsOfValue(history dashboardAccountHistory, target time.Time) (money.Money, bool) {
	if len(history.Balances) == 0 {
		return money.Money{}, false
	}
	if !target.After(history.Balances[0].Date) {
		return history.Balances[0].Amount, true
	}
	for i := len(history.Balances) - 1; i >= 0; i-- {
		if !history.Balances[i].Date.After(target) {
			return history.Balances[i].Amount, true
		}
	}
	return money.Money{}, false
}

func sameMonth(date, month time.Time) bool {
	return date.Format("2006-01") == month.Format("2006-01")
}

func monthBoundary(t time.Time) time.Time {
	y, m, _ := t.Date()
	return time.Date(y, m, 1, 0, 0, 0, 0, t.Location())
}

func recentMonthBoundaries(today time.Time, count int) []time.Time {
	if count <= 0 {
		return nil
	}
	baseMonth := monthBoundary(today)
	months := make([]time.Time, 0, count)
	for i := 0; i < count; i++ {
		months = append(months, baseMonth.AddDate(0, -i, 0))
	}
	return months
}

func parseDashboardDate(date string, loc *time.Location) (time.Time, error) {
	return time.ParseInLocation("2006-01-02", date, loc)
}

func dateOnly(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}
