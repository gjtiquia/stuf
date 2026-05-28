package model

import (
	"fmt"
	"strings"
	"time"

	"stuf/internal/component"
	"stuf/internal/money"
	"stuf/internal/service"
)

func (a App) dashboardContext() (string, error) {
	d, err := a.Svc.Dashboard.Summary(a.ctx)
	if err != nil {
		return "", err
	}
	cur := a.Config.Config.Currency
	values := alignedMoneyValues(
		component.MoneyCell(d.Total, cur),
		component.MoneyCell(money.Money{Scale: 2}, cur),
		component.MoneyCell(d.NetChangeFromMonthStart, cur),
		component.MoneyCell(d.NetChangeFromMonthHigh, cur),
		component.MoneyCell(d.NetChangeFromPreviousMonthHigh, cur),
		component.MoneyCell(d.RecentMonths[0].Drop, cur),
		component.MoneyCell(d.RecentMonths[1].Drop, cur),
		component.MoneyCell(d.RecentMonths[2].Drop, cur),
		component.MoneyCell(d.HighTrends[0].Change, cur),
		component.MoneyCell(d.HighTrends[1].Change, cur),
		component.MoneyCell(d.HighTrends[2].Change, cur),
		component.MoneyCell(d.LowTrends[0].Change, cur),
		component.MoneyCell(d.LowTrends[1].Change, cur),
		component.MoneyCell(d.LowTrends[2].Change, cur),
		component.MoneyCell(money.Money{Scale: 2}, cur),
		component.MoneyCell(money.Money{Scale: 2}, cur),
	)
	warnings := dashboardWarnings(d.Warnings)
	body := fmt.Sprintf(`total       : %s
budgeted    : %s
as of       : %s

%s
you owe ppl : %s
ppl owe you : %s
%s`, values[0], values[1], d.AsOf, dashboardSectionsWithValues(d, "on-budget ", values[2:14]), values[14], values[15], warnings)
	return strings.TrimRight(body, "\n"), nil
}

func (a App) accountDashboardContext(name string) (string, error) {
	acct, err := a.Svc.Accounts.GetByName(a.ctx, name)
	if err != nil {
		return "", err
	}
	d, err := a.Svc.Dashboard.AccountSummary(a.ctx, acct.ID)
	if err != nil {
		return "", err
	}
	summary, err := a.Svc.Accounts.TreeSummary(a.ctx, acct.ID, acct.Code)
	if err != nil {
		return "", err
	}
	asOf := "(no balance entered yet)"
	if summary.AsOf != "" {
		asOf = summary.AsOf
	}
	values := alignedMoneyValues(
		component.MoneyCell(summary.Balance, acct.Code),
		component.MoneyCell(summary.Children, acct.Code),
		component.MoneyCell(summary.Remaining, acct.Code),
		component.MoneyCell(d.NetChangeFromMonthStart, acct.Code),
		component.MoneyCell(d.NetChangeFromMonthHigh, acct.Code),
		component.MoneyCell(d.NetChangeFromPreviousMonthHigh, acct.Code),
		component.MoneyCell(d.RecentMonths[0].Drop, acct.Code),
		component.MoneyCell(d.RecentMonths[1].Drop, acct.Code),
		component.MoneyCell(d.RecentMonths[2].Drop, acct.Code),
		component.MoneyCell(d.HighTrends[0].Change, acct.Code),
		component.MoneyCell(d.HighTrends[1].Change, acct.Code),
		component.MoneyCell(d.HighTrends[2].Change, acct.Code),
		component.MoneyCell(d.LowTrends[0].Change, acct.Code),
		component.MoneyCell(d.LowTrends[1].Change, acct.Code),
		component.MoneyCell(d.LowTrends[2].Change, acct.Code),
	)
	lines := []string{
		fmt.Sprintf("account   : %s", acct.Name),
		fmt.Sprintf("balance   : %s", values[0]),
		fmt.Sprintf("children  : %s", values[1]),
		fmt.Sprintf("remaining : %s", values[2]),
		fmt.Sprintf("as of     : %s", asOf),
		fmt.Sprintf("on-budget : %t", acct.OnBudget),
	}
	if acct.Hidden {
		lines = append(lines, "hidden    : true")
	}
	lines = append(lines, fmt.Sprintf("notes     : %s", acct.Notes), "", dashboardSectionsWithValues(d, "", values[3:15]))
	if warnings := dashboardWarnings(d.Warnings); warnings != "" {
		lines = append(lines, warnings)
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n"), nil
}

func dashboardSections(d service.Dashboard, cur, headingPrefix string) string {
	values := alignedMoneyValues(
		component.MoneyCell(d.NetChangeFromMonthStart, cur),
		component.MoneyCell(d.NetChangeFromMonthHigh, cur),
		component.MoneyCell(d.NetChangeFromPreviousMonthHigh, cur),
		component.MoneyCell(d.RecentMonths[0].Drop, cur),
		component.MoneyCell(d.RecentMonths[1].Drop, cur),
		component.MoneyCell(d.RecentMonths[2].Drop, cur),
		component.MoneyCell(d.HighTrends[0].Change, cur),
		component.MoneyCell(d.HighTrends[1].Change, cur),
		component.MoneyCell(d.HighTrends[2].Change, cur),
		component.MoneyCell(d.LowTrends[0].Change, cur),
		component.MoneyCell(d.LowTrends[1].Change, cur),
		component.MoneyCell(d.LowTrends[2].Change, cur),
	)
	return dashboardSectionsWithValues(d, headingPrefix, values)
}

func dashboardSectionsWithValues(d service.Dashboard, headingPrefix string, values []string) string {
	currentLabel := monthLabel(d.Period)
	recentLabels := make([]string, len(d.RecentMonths))
	for i, month := range d.RecentMonths {
		recentLabels[i] = monthLabel(month.Period)
	}
	highTrendLabels := make([][2]string, len(d.HighTrends))
	for i, trend := range d.HighTrends {
		highTrendLabels[i] = [2]string{monthLabel(trend.FromPeriod), monthLabel(trend.ToPeriod)}
	}
	lowTrendLabels := make([][2]string, len(d.LowTrends))
	for i, trend := range d.LowTrends {
		lowTrendLabels[i] = [2]string{monthLabel(trend.FromPeriod), monthLabel(trend.ToPeriod)}
	}
	return fmt.Sprintf(`%snet change to today
from %s start : %s
from %s high  : %s
from %s high  : %s

%srecent months
%s high to low : %s
%s high to low : %s
%s high to low : %s

%shigh to high trends
%s to %s      : %s
%s to %s      : %s
%s to %s      : %s

%slow to low trends
%s to %s      : %s
%s to %s      : %s
%s to %s      : %s`,
		headingPrefix,
		currentLabel, values[0],
		currentLabel, values[1],
		recentLabels[1], values[2],
		headingPrefix,
		recentLabels[0], values[3],
		recentLabels[1], values[4],
		recentLabels[2], values[5],
		headingPrefix,
		highTrendLabels[0][0], highTrendLabels[0][1], values[6],
		highTrendLabels[1][0], highTrendLabels[1][1], values[7],
		highTrendLabels[2][0], highTrendLabels[2][1], values[8],
		headingPrefix,
		lowTrendLabels[0][0], lowTrendLabels[0][1], values[9],
		lowTrendLabels[1][0], lowTrendLabels[1][1], values[10],
		lowTrendLabels[2][0], lowTrendLabels[2][1], values[11])
}

func dashboardWarnings(warnings []string) string {
	if len(warnings) == 0 {
		return ""
	}
	return "warning: " + strings.Join(warnings, "; ") + "\n"
}

func monthLabel(period string) string {
	t, err := time.Parse("2006-01", period)
	if err != nil {
		return period
	}
	return strings.ToLower(t.Format("Jan"))
}

func (a App) dashboardScreen() screen {
	context, err := a.dashboardContext()
	if err != nil {
		return screen{Path: "/", Body: "error: " + err.Error() + "\n"}
	}
	return screen{
		Path:    "/",
		Context: context,
		Actions: []string{"accounts", "transactions (TODO)", "budgets (TODO)", "owed (TODO)", "reports (TODO)", "settings", "backup"},
	}
}
