package model

import (
	"fmt"
	"strings"

	"stuf/internal/component"
)

func (a App) balanceAddKey(s, name string) App {
	acct, err := a.Svc.Accounts.GetByName(a.ctx, name)
	if err != nil {
		a.Error = err.Error()
		return a
	}
	fields := []string{"date", "balance", "notes"}
	next, submit := a.submitFormKey(s, fields)
	if !submit {
		return next
	}
	_, entry, err := next.Svc.Balances.Add(next.ctx, acct.ID, next.Form["date"], next.Form["balance"], next.Form["notes"])
	if err != nil {
		next.Error = err.Error()
		return next
	}
	next.History = append(next.History, entry)
	next.Form = map[string]string{}
	next.Field = 0
	next.Error = ""
	next.Nav.Pop()
	return next.navReplace(accountBalanceListPath(name), 0)
}

func (a App) balanceMenuKey(s, name string) App {
	routes := []string{accountBalanceListPath(name), accountBalanceAddPath(name)}
	a = a.menuKey(s, routes)
	if a.Path == accountBalanceAddPath(name) {
		a.Form = map[string]string{"date": Today()}
		a.Field = 0
	}
	return a
}

func (a App) balanceListTableKey(s, name string) App {
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
	switch s {
	case "left":
		a.Error = ""
		return a.goBack()
	case "right":
		if len(rows) == 0 {
			return a
		}
		a = a.navSetMenu(clampListCursor(a.Menu, len(rows)))
		return a.navPush(accountBalancePath(name, rows[a.Menu].Date), 0)
	case "up", "k", "shift+tab":
		if len(rows) > 0 {
			a = a.navSetMenu((a.Menu - 1 + len(rows)) % len(rows))
		}
		return a
	case "down", "j", "tab":
		if len(rows) > 0 {
			a = a.navSetMenu((a.Menu + 1) % len(rows))
		}
		return a
	case "enter":
		if len(rows) == 0 {
			return a
		}
		a = a.navSetMenu(clampListCursor(a.Menu, len(rows)))
		return a.navPush(accountBalancePath(name, rows[a.Menu].Date), 0)
	default:
		return a
	}
}

func (a App) balanceDetailKey(s, name, date string) App {
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
	currentIdx := -1
	for i, row := range rows {
		if row.Date == date {
			currentIdx = i
			break
		}
	}
	if isItemPrevKey(s) {
		if currentIdx >= 0 && currentIdx < len(rows)-1 {
			return a.navReplace(accountBalancePath(name, rows[currentIdx+1].Date), a.Menu)
		}
		return a
	}
	if isItemNextKey(s) {
		if currentIdx > 0 {
			return a.navReplace(accountBalancePath(name, rows[currentIdx-1].Date), a.Menu)
		}
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
	fields := []string{"date", "balance", "notes"}
	next, submit := a.submitFormKey(s, fields)
	if !submit {
		return next
	}
	_, entry, err := next.Svc.Balances.Update(next.ctx, bal.ID, next.Form["date"], next.Form["balance"], next.Form["notes"])
	if err != nil {
		next.Error = err.Error()
		return next
	}
	next.History = append(next.History, entry)
	next.Form = map[string]string{}
	next.Field = 0
	next.Error = ""
	next.Nav.Pop()
	next.Nav.Pop()
	return next.syncFromNav()
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
		Context: strings.TrimRight(a.balanceSummary(name), "\n"),
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
		Context: strings.TrimRight(a.balanceSummary(name), "\n"),
		Options: a.balanceFormView(acct.Code),
		Help:    a.formHelp(fields),
	}
}

func (a App) balanceListBody(name string) string {
	acct, err := a.Svc.Accounts.GetByName(a.ctx, name)
	if err != nil {
		return "error: " + err.Error() + "\n"
	}
	rows, err := a.Svc.Balances.List(a.ctx, acct.ID)
	if err != nil {
		return "error: " + err.Error() + "\n"
	}
	lines := []string{"  date       | balance      | notes"}
	if len(rows) == 0 {
		lines = append(lines, "  (no balances yet)")
		return strings.Join(lines, "\n") + "\n"
	}
	tableRows := make([][]string, 0, len(rows))
	for _, row := range rows {
		tableRows = append(tableRows, []string{row.Date, row.Amount.Format(acct.Code), row.Notes})
	}
	layout := component.NewTableLayout([]string{"date", "balance", "notes"}, tableRows)
	lines[0] = layout.Header("  ")
	for i, row := range rows {
		prefix := "  "
		if a.Menu == i {
			prefix = "> "
		}
		lines = append(lines, layout.Row(prefix, []string{row.Date, row.Amount.Format(acct.Code), row.Notes}))
	}
	return strings.Join(lines, "\n") + "\n"
}

func (a App) balanceListTable(name string) string {
	context := strings.TrimRight(a.balanceSummary(name), "\n")
	body := a.balanceListBody(name)
	if context == "" {
		return body
	}
	return context + "\n\n" + body
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
	rows, err := a.Svc.Balances.List(a.ctx, acct.ID)
	if err != nil {
		return screen{Path: a.Path, Body: "error: " + err.Error() + "\n"}
	}
	currentIdx := -1
	for i, row := range rows {
		if row.Date == date {
			currentIdx = i
			break
		}
	}
	return screen{
		Path:    accountBalancePath(name, date),
		Context: fmt.Sprintf("account : %s\ndate    : %s\nbalance : %s\nnotes   : %s", name, date, bal.Amount.Format(acct.Code), bal.Notes),
		Actions: []string{"edit balance", "delete balance"},
		Help:    balanceDetailHelp(currentIdx, len(rows)),
	}
}

func balanceDetailHelp(currentIdx, count int) []string {
	lines := []string{"up/down/j/k : navigate", "enter       : confirm", "esc         : back", "?           : help", "ctrl-z      : undo"}
	if currentIdx >= 0 && currentIdx < count-1 {
		lines = append(lines, "left/h      : older")
	}
	if currentIdx > 0 {
		lines = append(lines, "right/l     : newer")
	}
	return lines
}
