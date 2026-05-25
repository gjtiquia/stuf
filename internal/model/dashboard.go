package model

import (
	"fmt"
	"strings"
	"time"

	"stuf/internal/component"
	"stuf/internal/money"
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
	currentLabel := monthLabel(d.Period)
	prevLabel := monthLabel(d.RecentMonths[0].Period)
	prevPrevLabel := monthLabel(d.RecentMonths[1].Period)
	warnings := ""
	if len(d.Warnings) > 0 {
		warnings = "\nwarning: " + strings.Join(d.Warnings, "; ") + "\n"
	}
	body := fmt.Sprintf(`total       : %s
budgeted    : %s

period      : %s

on-budget net change to today
from %s start : %s
from %s high  : %s
from %s high  : %s

on-budget recent months
%s high to low : %s
%s high to low : %s

on-budget %s to %s trends
high to high    : %s
low to low      : %s

you owe ppl : %s
ppl owe you : %s
%s`, values[0], values[1], d.Period,
		currentLabel, values[2],
		currentLabel, values[3],
		prevLabel, values[4],
		prevLabel, values[5],
		prevPrevLabel, values[6],
		prevPrevLabel, prevLabel, values[7], values[8],
		values[9], values[10], warnings)
	return strings.TrimRight(body, "\n"), nil
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
