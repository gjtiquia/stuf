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
		component.MoneyCell(d.Trend.HighToHigh, cur),
		component.MoneyCell(d.Trend.LowToLow, cur),
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
%s`, values[0], values[1], d.AsOf, dashboardSectionsWithValues(d, "on-budget ", values[2:9]), values[9], values[10], warnings)
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
		component.MoneyCell(d.Trend.HighToHigh, acct.Code),
		component.MoneyCell(d.Trend.LowToLow, acct.Code),
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
	lines = append(lines, fmt.Sprintf("notes     : %s", acct.Notes), "", dashboardSectionsWithValues(d, "", values[3:10]))
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
		component.MoneyCell(d.Trend.HighToHigh, cur),
		component.MoneyCell(d.Trend.LowToLow, cur),
	)
	return dashboardSectionsWithValues(d, headingPrefix, values)
}

func dashboardSectionsWithValues(d service.Dashboard, headingPrefix string, values []string) string {
	currentLabel := monthLabel(d.Period)
	prevLabel := monthLabel(d.RecentMonths[0].Period)
	prevPrevLabel := monthLabel(d.RecentMonths[1].Period)
	return fmt.Sprintf(`%snet change to today
from %s start : %s
from %s high  : %s
from %s high  : %s

%srecent months
%s high to low : %s
%s high to low : %s

%s%s to %s trends
high to high    : %s
low to low      : %s`,
		headingPrefix,
		currentLabel, values[0],
		currentLabel, values[1],
		prevLabel, values[2],
		headingPrefix,
		prevLabel, values[3],
		prevPrevLabel, values[4],
		headingPrefix, prevPrevLabel, prevLabel, values[5], values[6])
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
