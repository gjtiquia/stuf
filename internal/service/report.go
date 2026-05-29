package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"stuf/internal/money"
	"stuf/internal/repo"
)

type ReportService struct {
	Accounts    *repo.AccountRepo
	Balances    *repo.BalanceRepo
	Currencies  *repo.CurrencyRepo
	AppCurrency string
	Now         func() time.Time
}

type ReportMonthlyRow struct {
	Period  string
	Metrics ReportPeriodMetrics
}

type ReportMonthlyDetail struct {
	Period      string
	Coverage    ReportCoverage
	Metrics     ReportPeriodMetrics
	Rows        []ReportAccountRow
	Warnings    []string
	AppCurrency string
}

type ReportCoverage struct {
	Start string
	End   string
}

type ReportPeriodMetrics struct {
	Start     money.Money
	End       money.Money
	Change    money.Money
	High      money.Money
	Low       money.Money
	HighToLow money.Money
}

type ReportAccountRow struct {
	ID      int64
	Name    string
	Depth   int
	Virtual bool
	Metrics ReportPeriodMetrics
}

func (s ReportService) MonthlyRows(ctx context.Context, count int) ([]ReportMonthlyRow, []string, error) {
	if count <= 0 {
		return nil, nil, nil
	}
	today := s.today()
	appCur, err := s.Currencies.GetByCode(ctx, s.AppCurrency)
	if err != nil {
		return nil, nil, err
	}
	histories, warnings, err := s.onBudgetHistories(ctx, appCur, today)
	if err != nil {
		return nil, nil, err
	}
	calcDate := reportCalcDate(histories, today)
	zero := money.Money{Scale: appCur.Scale}
	rows := make([]ReportMonthlyRow, 0, count)
	for i := 0; i < count; i++ {
		month := today.AddDate(0, -i, 0)
		rows = append(rows, ReportMonthlyRow{
			Period:  month.Format("2006-01"),
			Metrics: reportPeriodMetrics(histories, month, calcDate, zero),
		})
	}
	return rows, warnings, nil
}

func (s ReportService) MonthlyDetail(ctx context.Context, period string) (ReportMonthlyDetail, error) {
	month, err := time.ParseInLocation("2006-01", period, s.today().Location())
	if err != nil {
		return ReportMonthlyDetail{}, fmt.Errorf("month must be YYYY-MM")
	}
	today := s.today()
	appCur, err := s.Currencies.GetByCode(ctx, s.AppCurrency)
	if err != nil {
		return ReportMonthlyDetail{}, err
	}
	histories, warnings, err := s.onBudgetHistories(ctx, appCur, today)
	if err != nil {
		return ReportMonthlyDetail{}, err
	}
	calcDate := reportCalcDate(histories, today)
	zero := money.Money{Scale: appCur.Scale}
	rows, rowWarnings, err := s.monthlyAccountRows(ctx, month, calcDate, appCur, today)
	if err != nil {
		return ReportMonthlyDetail{}, err
	}
	warnings = appendUniqueStrings(warnings, rowWarnings...)
	return ReportMonthlyDetail{
		Period:      month.Format("2006-01"),
		Coverage:    reportCoverage(month),
		Metrics:     reportPeriodMetrics(histories, month, calcDate, zero),
		Rows:        rows,
		Warnings:    warnings,
		AppCurrency: appCur.Code,
	}, nil
}

func (s ReportService) today() time.Time {
	now := time.Now
	if s.Now != nil {
		now = s.Now
	}
	return dateOnly(now())
}

func (s ReportService) dashboard() DashboardService {
	return DashboardService{
		Accounts:    s.Accounts,
		Balances:    s.Balances,
		Currencies:  s.Currencies,
		AppCurrency: s.AppCurrency,
		Now:         s.Now,
	}
}

func (s ReportService) onBudgetHistories(ctx context.Context, appCur repo.Currency, today time.Time) ([]dashboardAccountHistory, []string, error) {
	ds := s.dashboard()
	accounts, err := s.Accounts.ListRoots(ctx, false)
	if err != nil {
		return nil, nil, err
	}
	var histories []dashboardAccountHistory
	var warnings []string
	for _, acct := range accounts {
		if !acct.OnBudget {
			continue
		}
		cur, err := s.Currencies.GetByID(ctx, acct.CurrencyID)
		if err != nil {
			return nil, nil, err
		}
		accountHistories, accountWarnings, err := ds.effectiveAccountHistories(ctx, acct, cur, appCur, today)
		if err != nil {
			return nil, nil, err
		}
		histories = append(histories, accountHistories...)
		warnings = appendUniqueStrings(warnings, accountWarnings...)
	}
	return histories, warnings, nil
}

func (s ReportService) monthlyAccountRows(ctx context.Context, month, calcDate time.Time, appCur repo.Currency, today time.Time) ([]ReportAccountRow, []string, error) {
	accounts, err := s.Accounts.ListRoots(ctx, false)
	if err != nil {
		return nil, nil, err
	}
	var rows []ReportAccountRow
	var warnings []string
	for _, acct := range accounts {
		if !acct.OnBudget {
			continue
		}
		accountRows, accountWarnings, err := s.monthlyAccountTreeRows(ctx, acct, month, calcDate, appCur, today, 0)
		if err != nil {
			return nil, nil, err
		}
		rows = append(rows, accountRows...)
		warnings = appendUniqueStrings(warnings, accountWarnings...)
	}
	return rows, warnings, nil
}

func (s ReportService) monthlyAccountTreeRows(ctx context.Context, acct repo.Account, month, calcDate time.Time, appCur repo.Currency, today time.Time, depth int) ([]ReportAccountRow, []string, error) {
	ds := s.dashboard()
	cur, err := s.Currencies.GetByID(ctx, acct.CurrencyID)
	if err != nil {
		return nil, nil, err
	}
	histories, warnings, err := ds.effectiveAccountHistories(ctx, acct, cur, appCur, today)
	if err != nil {
		return nil, nil, err
	}
	zero := money.Money{Scale: appCur.Scale}
	rows := []ReportAccountRow{{
		ID:      acct.ID,
		Name:    acct.Name,
		Depth:   depth,
		Metrics: reportPeriodMetrics(histories, month, calcDate, zero),
	}}
	children, err := s.Accounts.ListChildren(ctx, acct.ID, false)
	if err != nil {
		return nil, nil, err
	}
	var childHistories []dashboardAccountHistory
	for _, child := range children {
		childRows, childWarnings, err := s.monthlyAccountTreeRows(ctx, child, month, calcDate, appCur, today, depth+1)
		if err != nil {
			return nil, nil, err
		}
		rows = append(rows, childRows...)
		warnings = appendUniqueStrings(warnings, childWarnings...)
		childCur, err := s.Currencies.GetByID(ctx, child.CurrencyID)
		if err != nil {
			return nil, nil, err
		}
		childEffective, childWarnings, err := ds.effectiveAccountHistories(ctx, child, childCur, appCur, today)
		if err != nil {
			return nil, nil, err
		}
		childHistories = append(childHistories, childEffective...)
		warnings = appendUniqueStrings(warnings, childWarnings...)
	}
	if len(children) > 0 {
		own, ownWarnings, err := ds.accountHistory(ctx, acct.ID, cur, appCur, today)
		if err != nil {
			return nil, nil, err
		}
		warnings = appendUniqueStrings(warnings, ownWarnings...)
		if len(own.Balances) > 0 {
			remaining := remainingAccountHistory(own, childHistories, zero)
			metrics := reportPeriodMetrics([]dashboardAccountHistory{remaining}, month, calcDate, zero)
			if !reportMetricsZero(metrics) {
				rows = append(rows, ReportAccountRow{
					Name:    "remaining",
					Depth:   depth + 1,
					Virtual: true,
					Metrics: metrics,
				})
			}
		}
	}
	return rows, warnings, nil
}

func reportPeriodMetrics(histories []dashboardAccountHistory, month, calcDate time.Time, zero money.Money) ReportPeriodMetrics {
	start := totalAsOfValue(histories, monthBoundary(month), zero)
	end := totalAsOfValue(histories, dashboardMonthEnd(month, calcDate, calcDate), zero)
	change, _ := end.Sub(start)
	high, low := totalMonthHighLow(histories, month, calcDate, zero)
	highToLow, _ := low.Sub(high)
	return ReportPeriodMetrics{
		Start:     start,
		End:       end,
		Change:    change,
		High:      high,
		Low:       low,
		HighToLow: highToLow,
	}
}

func reportCoverage(month time.Time) ReportCoverage {
	start := monthBoundary(month)
	return ReportCoverage{
		Start: start.Format("2006-01-02"),
		End:   start.AddDate(0, 1, -1).Format("2006-01-02"),
	}
}

func reportCalcDate(histories []dashboardAccountHistory, today time.Time) time.Time {
	latest, ok := latestHistoryDate(histories)
	if !ok || latest.After(today) {
		return today
	}
	return latest
}

func reportMetricsZero(metrics ReportPeriodMetrics) bool {
	return metrics.Start.IsZero() &&
		metrics.End.IsZero() &&
		metrics.Change.IsZero() &&
		metrics.High.IsZero() &&
		metrics.Low.IsZero() &&
		metrics.HighToLow.IsZero()
}

func appendUniqueStrings(items []string, more ...string) []string {
	seen := map[string]bool{}
	for _, item := range items {
		seen[item] = true
	}
	for _, item := range more {
		if seen[item] {
			continue
		}
		items = append(items, item)
		seen[item] = true
	}
	sort.Strings(items)
	return items
}
