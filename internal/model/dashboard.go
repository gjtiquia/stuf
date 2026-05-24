package model

import (
	"fmt"
	"strings"
)

func (a App) dashboardContext() (string, error) {
	d, err := a.Svc.Dashboard.Summary(a.ctx)
	if err != nil {
		return "", err
	}
	cur := a.Config.Config.Currency
	warnings := ""
	if len(d.Warnings) > 0 {
		warnings = "\nwarning: " + strings.Join(d.Warnings, "; ") + "\n"
	}
	body := fmt.Sprintf(`total       : %s
budgeted    : %s

period      : %s

growth
on-budget  : %s
total      : %s

you owe ppl : %s
ppl owe you : %s
%s`, d.Total.Format(cur), zero(cur), d.Period, d.OnBudgetGrow.Format(cur), d.TotalGrow.Format(cur), zero(cur), zero(cur), warnings)
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
