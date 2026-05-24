package model

import (
	"fmt"
	"strings"
)

func (a App) balanceAddKey(s, name string) App {
	acct, err := a.Svc.Accounts.GetByName(a.ctx, name)
	if err != nil {
		a.Error = err.Error()
		return a
	}
	if s == "enter" {
		_, entry, err := a.Svc.Balances.Add(a.ctx, acct.ID, a.Form["date"], a.Form["balance"], a.Form["notes"])
		if err != nil {
			a.Error = err.Error()
			return a
		}
		a.History = append(a.History, entry)
		a.Form = map[string]string{}
		a.Field = 0
		a.Error = ""
		a.Nav.Pop()
		return a.syncFromNav()
	}
	return a.formKey(s, []string{"date", "balance", "notes"})
}

func (a App) balanceListKey(s, name string) App {
	acct, err := a.Svc.Accounts.GetByName(a.ctx, name)
	if err != nil {
		a.Error = err.Error()
		return a
	}
	rows, err := a.Svc.Balances.List(a.ctx, acct.ID)
	if err != nil {
		a.Error = err.Error()
		return a
	}
	var routes []string
	for _, row := range rows {
		routes = append(routes, accountBalancePath(name, row.Date))
	}
	routes = append(routes, accountBalanceAddPath(name))
	a = a.menuKey(s, routes)
	if a.Path == accountBalanceAddPath(name) {
		a.Form = map[string]string{"date": Today()}
		a.Field = 0
	}
	return a
}

func (a App) balanceDetailKey(s, name, date string) App {
	acct, err := a.Svc.Accounts.GetByName(a.ctx, name)
	if err != nil {
		a.Error = err.Error()
		return a
	}
	bal, err := a.Svc.Balances.GetByAccountDate(a.ctx, acct.ID, date)
	if err != nil {
		a.Error = err.Error()
		return a
	}
	action := a.actionIndex(s, 2)
	if action < 0 {
		return a
	}
	a = a.navSetMenu(action)
	switch action {
	case 0:
		a.Form = map[string]string{"date": bal.Date, "balance": rawAmount(bal.Amount.Amount, bal.Amount.Scale), "notes": bal.Notes}
		a.Field = 0
		return a.navPush(accountBalanceEditPath(name, date), 0)
	case 1:
		entry, err := a.Svc.Balances.Delete(a.ctx, bal.ID)
		if err != nil {
			a.Error = err.Error()
			return a
		}
		a.History = append(a.History, entry)
		a.Error = ""
		a.Nav.Pop()
		return a.syncFromNav()
	}
	return a
}

func (a App) balanceEditKey(s, name, date string) App {
	acct, err := a.Svc.Accounts.GetByName(a.ctx, name)
	if err != nil {
		a.Error = err.Error()
		return a
	}
	bal, err := a.Svc.Balances.GetByAccountDate(a.ctx, acct.ID, date)
	if err != nil {
		a.Error = err.Error()
		return a
	}
	if s == "enter" {
		_, entry, err := a.Svc.Balances.Update(a.ctx, bal.ID, a.Form["date"], a.Form["balance"], a.Form["notes"])
		if err != nil {
			a.Error = err.Error()
			return a
		}
		a.History = append(a.History, entry)
		a.Form = map[string]string{}
		a.Field = 0
		a.Error = ""
		a.Nav.Pop()
		a.Nav.Pop()
		return a.syncFromNav()
	}
	return a.formKey(s, []string{"date", "balance", "notes"})
}

func (a App) balanceSummary(name string) string {
	acct, err := a.Svc.Accounts.GetByName(a.ctx, name)
	if err != nil {
		return "error: " + err.Error() + "\n"
	}
	bal, ok, _ := a.Svc.Accounts.CurrentBalance(a.ctx, acct.ID)
	amount := zero(acct.Code)
	asOf := "(no balance entered yet)"
	if ok {
		amount = bal.Amount.Format(acct.Code)
		asOf = bal.Date
	}
	return fmt.Sprintf("name        : %s\nbalance     : %s\nas of       : %s\n", acct.Name, amount, asOf)
}

func (a App) balanceFormView(currency string) string {
	fields := []string{"date", "balance", "notes"}
	prefixes := map[string]string{"balance": currency}
	return a.formViewWithOptions(fields, nil, nil, prefixes)
}

func (a App) balanceAddScreen(name string) screen {
	acct, err := a.Svc.Accounts.GetByName(a.ctx, name)
	if err != nil {
		return screen{Path: accountBalanceAddPath(name), Body: "error: " + err.Error() + "\n"}
	}
	fields := []string{"date", "balance", "notes"}
	return screen{
		Path:    accountBalanceAddPath(name),
		Body:    a.balanceSummary(name),
		Options: a.balanceFormView(acct.Code),
		Help:    a.formHelp(fields),
	}
}

func (a App) balanceEditScreen(name, date string) screen {
	acct, err := a.Svc.Accounts.GetByName(a.ctx, name)
	if err != nil {
		return screen{Path: accountBalanceEditPath(name, date), Body: "error: " + err.Error() + "\n"}
	}
	fields := []string{"date", "balance", "notes"}
	return screen{
		Path:    accountBalanceEditPath(name, date),
		Body:    a.balanceSummary(name),
		Options: a.balanceFormView(acct.Code),
		Help:    a.formHelp(fields),
	}
}

func (a App) balanceList(name string) string {
	acct, err := a.Svc.Accounts.GetByName(a.ctx, name)
	if err != nil {
		return "error: " + err.Error() + "\n"
	}
	rows, err := a.Svc.Balances.List(a.ctx, acct.ID)
	if err != nil {
		return "error: " + err.Error() + "\n"
	}
	lines := []string{strings.TrimRight(a.balanceSummary(name), "\n"), "", "  date       | balance      | notes"}
	if len(rows) == 0 {
		lines = append(lines, "  (no balances yet)", "")
		lines = append(lines, menuItems([]string{"add balance"}, a.Menu))
		return strings.Join(lines, "\n") + "\n"
	}
	for i, row := range rows {
		prefix := "  "
		if a.Menu == i {
			prefix = "> "
		}
		lines = append(lines, fmt.Sprintf("%s%s | %-12s | %s", prefix, row.Date, row.Amount.Format(acct.Code), row.Notes))
	}
	selectedAction := -1
	if a.Menu == len(rows) {
		selectedAction = 0
	}
	lines = append(lines, "", menuItems([]string{"add balance"}, selectedAction))
	return strings.Join(lines, "\n") + "\n"
}

func (a App) balanceDetailScreen(name, date string) screen {
	acct, err := a.Svc.Accounts.GetByName(a.ctx, name)
	if err != nil {
		return screen{Path: a.Path, Body: "error: " + err.Error() + "\n"}
	}
	bal, err := a.Svc.Balances.GetByAccountDate(a.ctx, acct.ID, date)
	if err != nil {
		return screen{Path: a.Path, Body: "error: " + err.Error() + "\n"}
	}
	return screen{
		Path:    accountBalancePath(name, date),
		Body:    fmt.Sprintf("account : %s\ndate    : %s\nbalance : %s\nnotes   : %s\n", name, date, bal.Amount.Format(acct.Code), bal.Notes),
		Actions: []string{"edit balance", "delete balance"},
	}
}
