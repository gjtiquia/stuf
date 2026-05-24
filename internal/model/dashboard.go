package model

import (
	"fmt"
	"strings"
)

func (a App) dashboardScreen() screen {
	d, err := a.Svc.Dashboard.Summary(a.ctx)
	if err != nil {
		return screen{Path: "/", Body: "error: " + err.Error() + "\n"}
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
	return screen{
		Path:    "/",
		Body:    body,
		Actions: []string{"accounts", "transactions (TODO)", "budgets (TODO)", "owed (TODO)", "reports (TODO)", "settings", "backup"},
	}
}
