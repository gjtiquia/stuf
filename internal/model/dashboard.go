package model

import (
	"fmt"
	"strings"

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
		component.MoneyCell(d.OnBudgetGrow, cur),
		component.MoneyCell(d.TotalGrow, cur),
		component.MoneyCell(money.Money{Scale: 2}, cur),
		component.MoneyCell(money.Money{Scale: 2}, cur),
	)
	warnings := ""
	if len(d.Warnings) > 0 {
		warnings = "\nwarning: " + strings.Join(d.Warnings, "; ") + "\n"
	}
	body := fmt.Sprintf(`total       : %s
budgeted    : %s

period      : %s

growth
on-budget   : %s
total       : %s

you owe ppl : %s
ppl owe you : %s
%s`, values[0], values[1], d.Period, values[2], values[3], values[4], values[5], warnings)
	return strings.TrimRight(body, "\n"), nil
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
