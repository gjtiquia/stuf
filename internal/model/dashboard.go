package model

import (
	"fmt"
	"strings"

	"stuf/internal/component"
	"stuf/internal/money"
	"stuf/internal/service"
)

func (a App) dashboardContext() (string, error) {
	return a.dashboardContextWithOwed(true)
}

func (a App) dashboardContextWithoutOwed() (string, error) {
	return a.dashboardContextWithOwed(false)
}

func (a App) dashboardContextWithOwed(includeOwed bool) (string, error) {
	d, err := a.Svc.Dashboard.Summary(a.ctx)
	if err != nil {
		return "", err
	}
	cur := a.Config.Config.Currency
	return formatDashboardContext(d, cur, includeOwed), nil
}

func (a App) accountListDashboardContext() (string, error) {
	rows, err := a.accountListRowsWithFilter(a.listFilter())
	if err != nil {
		return "", err
	}
	d, err := a.Svc.Dashboard.SummaryForAccounts(a.ctx, dashboardAccountIDsForRows(rows))
	if err != nil {
		return "", err
	}
	return formatAccountListDashboardContext(d, a.Config.Config.Currency, rows), nil
}

func formatDashboardContext(d service.Dashboard, cur string, includeOwed bool) string {
	values := alignedMoneyValues(
		component.MoneyCell(d.Total, cur),
		component.MoneyCell(d.Budgeted, cur),
		component.MoneyCell(d.NetChanges[0].Change, cur),
		component.MoneyCell(d.NetChanges[1].Change, cur),
		component.MoneyCell(d.NetChanges[2].Change, cur),
		component.MoneyCell(d.HighToLows[0].Drop, cur),
		component.MoneyCell(d.HighToLows[1].Drop, cur),
		component.MoneyCell(d.HighToLows[2].Drop, cur),
		component.MoneyCell(d.Lows[0].Low, cur),
		component.MoneyCell(d.Lows[1].Low, cur),
		component.MoneyCell(d.Lows[2].Low, cur),
	)
	warnings := dashboardWarnings(d.Warnings)
	body := fmt.Sprintf(`as-of       : %s

total       : %s
budgeted    : %s

%s`, dashboardAsOf(d), values[0], values[1], dashboardSectionsWithValues(d, "on-budget ", values[2:11]))
	if includeOwed {
		owed := alignedMoneyValues(
			component.MoneyCell(money.Money{Scale: 2}, cur),
			component.MoneyCell(money.Money{Scale: 2}, cur),
		)
		body += fmt.Sprintf(`

you owe ppl : %s
ppl owe you : %s`, owed[0], owed[1])
	}
	body += "\n" + warnings
	return strings.TrimRight(body, "\n")
}

func formatAccountListDashboardContext(d service.Dashboard, cur string, rows []accountListRow) string {
	summaryValues := accountSummaryValues(rows, cur)
	values := alignedMoneyValues(
		component.MoneyCell(d.NetChanges[0].Change, cur),
		component.MoneyCell(d.NetChanges[1].Change, cur),
		component.MoneyCell(d.NetChanges[2].Change, cur),
		component.MoneyCell(d.HighToLows[0].Drop, cur),
		component.MoneyCell(d.HighToLows[1].Drop, cur),
		component.MoneyCell(d.HighToLows[2].Drop, cur),
		component.MoneyCell(d.Lows[0].Low, cur),
		component.MoneyCell(d.Lows[1].Low, cur),
		component.MoneyCell(d.Lows[2].Low, cur),
	)
	body := fmt.Sprintf(`as-of       : %s

on-budget   : %s
off-budget  : %s
total       : %s

%s`, dashboardAsOf(d), summaryValues[1], summaryValues[2], summaryValues[0], dashboardSectionsWithValues(d, "on-budget ", values))
	if warnings := dashboardWarnings(d.Warnings); warnings != "" {
		body += "\n" + warnings
	}
	return strings.TrimRight(body, "\n")
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
	values := alignedMoneyValues(
		component.MoneyCell(summary.Balance, acct.Code),
		component.MoneyCell(summary.Children, acct.Code),
		component.MoneyCell(summary.Remaining, acct.Code),
		component.MoneyCell(d.NetChanges[0].Change, acct.Code),
		component.MoneyCell(d.NetChanges[1].Change, acct.Code),
		component.MoneyCell(d.NetChanges[2].Change, acct.Code),
		component.MoneyCell(d.HighToLows[0].Drop, acct.Code),
		component.MoneyCell(d.HighToLows[1].Drop, acct.Code),
		component.MoneyCell(d.HighToLows[2].Drop, acct.Code),
		component.MoneyCell(d.Lows[0].Low, acct.Code),
		component.MoneyCell(d.Lows[1].Low, acct.Code),
		component.MoneyCell(d.Lows[2].Low, acct.Code),
	)
	lines := []string{
		fmt.Sprintf("account   : %s", acct.Name),
		fmt.Sprintf("balance   : %s", values[0]),
		fmt.Sprintf("children  : %s", values[1]),
		fmt.Sprintf("remaining : %s", values[2]),
		fmt.Sprintf("as of     : %s", dashboardAsOf(d)),
		fmt.Sprintf("on-budget : %t", acct.OnBudget),
		fmt.Sprintf("tags      : %s", formatTags(a.effectiveTagNames(acct.ID), nil)),
		fmt.Sprintf("direct    : %s", formatTags(a.directTagNames(acct.ID), nil)),
		fmt.Sprintf("inherited : %s", formatTags(a.inheritedTagNames(acct.ID), nil)),
	}
	if acct.Hidden {
		lines = append(lines, "hidden    : true")
	}
	lines = append(lines, fmt.Sprintf("notes     : %s", acct.Notes), "", dashboardSectionsWithValues(d, "", values[3:12]))
	if warnings := dashboardWarnings(d.Warnings); warnings != "" {
		lines = append(lines, warnings)
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n"), nil
}

func dashboardSections(d service.Dashboard, cur, headingPrefix string) string {
	values := alignedMoneyValues(
		component.MoneyCell(d.NetChanges[0].Change, cur),
		component.MoneyCell(d.NetChanges[1].Change, cur),
		component.MoneyCell(d.NetChanges[2].Change, cur),
		component.MoneyCell(d.HighToLows[0].Drop, cur),
		component.MoneyCell(d.HighToLows[1].Drop, cur),
		component.MoneyCell(d.HighToLows[2].Drop, cur),
		component.MoneyCell(d.Lows[0].Low, cur),
		component.MoneyCell(d.Lows[1].Low, cur),
		component.MoneyCell(d.Lows[2].Low, cur),
	)
	return dashboardSectionsWithValues(d, headingPrefix, values)
}

func dashboardSectionsWithValues(d service.Dashboard, headingPrefix string, values []string) string {
	return fmt.Sprintf(`%snet changes
%s     : %s
%s     : %s
%s     : %s

%shigh to lows
%s     : %s
%s     : %s
%s     : %s

%slows
%s     : %s
%s     : %s
%s     : %s`,
		headingPrefix,
		d.NetChanges[0].Period, values[0],
		d.NetChanges[1].Period, values[1],
		d.NetChanges[2].Period, values[2],
		headingPrefix,
		d.HighToLows[0].Period, values[3],
		d.HighToLows[1].Period, values[4],
		d.HighToLows[2].Period, values[5],
		headingPrefix,
		d.Lows[0].Period, values[6],
		d.Lows[1].Period, values[7],
		d.Lows[2].Period, values[8])
}

func dashboardAsOf(d service.Dashboard) string {
	asOf := d.AsOf
	if asOf == "" {
		asOf = "none"
	}
	if d.AsOfStale {
		return asOf + " [!]"
	}
	return asOf
}

func dashboardWarnings(warnings []string) string {
	if len(warnings) == 0 {
		return ""
	}
	return "warning: " + strings.Join(warnings, "; ") + "\n"
}

func (a App) dashboardScreen() screen {
	context, err := a.dashboardContext()
	if err != nil {
		return screen{Path: "/", Body: "error: " + err.Error() + "\n"}
	}
	return screen{
		Path:    "/",
		Context: context,
		Actions: []string{"accounts", "transactions (TODO)", "budgets", "owed (TODO)", "reports", "settings", "backup"},
	}
}
