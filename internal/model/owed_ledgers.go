package model

import (
	"fmt"
	"strings"

	"stuf/internal/component"
	"stuf/internal/money"
	"stuf/internal/repo"
)

type owedLedgerListRow struct {
	Ledger  repo.OwedLedger
	Balance money.Money
	Display component.Cell
}

func (a App) owedLedgerListKey(s string) App {
	if isNewKey(s) {
		a.Error = ""
		a.Form = map[string]string{"currency": a.Config.Config.Currency}
		a.Field = 0
		return a.navPush(routeOwedCreate, 0)
	}
	rows, err := a.filteredOwedLedgerRows()
	if err != nil {
		a.Error = err.Error()
		return a
	}
	if isEditKey(s) && len(rows) > 0 {
		a = a.navSetMenu(clampListCursor(a.Menu, len(rows)))
		ledger := rows[a.Menu].Ledger
		a.Form = owedLedgerFormValues(ledger)
		a.Field = 0
		return a.navPush(owedLedgerEditPathFor(ledger.Name), 0)
	}
	switch s {
	case "left":
		a.Error = ""
		return a.goBack()
	case "right", "enter":
		if len(rows) == 0 {
			return a
		}
		a = a.navSetMenu(clampListCursor(a.Menu, len(rows)))
		return a.navPush(owedLedgerPath(rows[a.Menu].Ledger.Name), 0)
	case "up", "shift+tab":
		if len(rows) > 0 {
			a = a.navSetMenu(clampListCursor(a.Menu-1, len(rows)))
		}
	case "down", "tab":
		if len(rows) > 0 {
			a = a.navSetMenu(clampListCursor(a.Menu+1, len(rows)))
		}
	default:
		if result, handled := handleFilterableListKey(s, a.listFilter(), a.Menu, len(rows)); handled {
			a.setListFilter(result.filter)
			nextRows, _ := a.filteredOwedLedgerRows()
			a = a.navSetMenu(clampListCursor(result.menu, len(nextRows)))
		}
	}
	return a
}

func (a App) owedLedgerCreateKey(s string) App {
	if a.Form["currency"] == "" {
		a.Form["currency"] = a.Config.Config.Currency
	}
	next, submit := a.owedLedgerFormKey(s)
	if !submit {
		return next
	}
	ledger, entry, err := next.Svc.OwedLedgers.Create(next.ctx, strings.TrimSpace(next.Form["name"]), strings.TrimSpace(next.Form["currency"]), next.Form["notes"])
	if err != nil {
		next.Error = err.Error()
		return next
	}
	next.History = append(next.History, entry)
	next.Form = map[string]string{}
	next.Field = 0
	next.Error = ""
	next.Nav.Pop()
	return next.navReplace(owedLedgerPath(ledger.Name), 0)
}

func (a App) owedLedgerEditKey(s, name string) App {
	ledger, err := a.Svc.OwedLedgers.GetByName(a.ctx, name)
	if err != nil {
		a.Error = err.Error()
		return a
	}
	next, submit := a.owedLedgerFormKey(s)
	if !submit {
		return next
	}
	updated, entry, err := next.Svc.OwedLedgers.Update(next.ctx, ledger.ID, strings.TrimSpace(next.Form["name"]), strings.TrimSpace(next.Form["currency"]), next.Form["notes"])
	if err != nil {
		next.Error = err.Error()
		return next
	}
	next.History = append(next.History, entry)
	next.Form = map[string]string{}
	next.Field = 0
	next.Error = ""
	next.Nav.Pop()
	next = next.syncFromNav()
	return next.navReplace(owedLedgerPath(updated.Name), 0)
}

func (a App) owedLedgerDetailKey(s, name string) App {
	action := a.actionIndex(s, 2)
	if action < 0 {
		return a
	}
	a = a.navSetMenu(action)
	switch action {
	case 0:
		return a.navPush(owedTransactionListPath(name), 0)
	case 1:
		ledger, err := a.Svc.OwedLedgers.GetByName(a.ctx, name)
		if err != nil {
			a.Error = err.Error()
			return a
		}
		a.Form = owedLedgerFormValues(ledger)
		a.Field = 0
		return a.navPush(owedLedgerEditPathFor(name), 0)
	}
	return a
}

func (a App) owedLedgerFormKey(s string) (App, bool) {
	fields := []string{"name", "currency", "notes"}
	if isSubmitKey(s) {
		a.clearCurrentTextCursor(fields)
		return a, true
	}
	if a.Field == 1 {
		return a.currencyFieldKey(s, fields)
	}
	return a.submitFormKey(s, fields)
}

func (a App) owedLedgerListScreen() screen {
	context, err := a.owedDashboardContext()
	if err != nil {
		return screen{Path: routeOwedList, Body: "error: " + err.Error() + "\n"}
	}
	rows, err := a.filteredOwedLedgerRows()
	if err != nil {
		return screen{Path: routeOwedList, Context: context, Body: "error: " + err.Error() + "\n"}
	}
	lines := []string{"> filter : " + placeholder(a.listFilter(), "(type anything...)"), ""}
	if len(rows) == 0 {
		lines = append(lines, "  ledger | balance | notes")
		if a.listFilter() == "" {
			lines = append(lines, "  (no owed ledgers yet)")
		} else {
			lines = append(lines, "  (no results)")
		}
		return screen{Path: routeOwedList, Context: context, Body: strings.Join(lines, "\n") + "\n", Help: owedLedgerListHelp()}
	}
	tableRows := make([][]component.Cell, len(rows))
	for i, row := range rows {
		tableRows[i] = []component.Cell{component.TextCell(row.Ledger.Name), row.Display, component.TextCell(row.Ledger.Notes)}
	}
	layout := component.NewTableLayoutCells([]string{"ledger", "balance", "notes"}, tableRows)
	lines = append(lines, layout.Header("  "))
	for i, row := range tableRows {
		prefix := "  "
		if i == a.Menu {
			prefix = "> "
		}
		lines = append(lines, layout.RowCells(prefix, row))
	}
	return screen{Path: routeOwedList, Context: context, Body: strings.Join(lines, "\n") + "\n", Help: owedLedgerListHelp()}
}

func (a App) owedLedgerCreateScreen() screen {
	return screen{Path: routeOwedCreate, Body: a.owedLedgerFormView(), Help: a.formHelp([]string{"name", "currency", "notes"})}
}

func (a App) owedLedgerEditScreen(name string) screen {
	if a.Form["name"] == "" {
		if ledger, err := a.Svc.OwedLedgers.GetByName(a.ctx, name); err == nil {
			a.Form = owedLedgerFormValues(ledger)
		}
	}
	return screen{Path: owedLedgerEditPathFor(name), Body: a.owedLedgerFormView(), Help: a.formHelp([]string{"name", "currency", "notes"})}
}

func (a App) owedLedgerDetailScreen(name string) screen {
	ledger, err := a.Svc.OwedLedgers.GetByName(a.ctx, name)
	if err != nil {
		return screen{Path: owedLedgerPath(name), Body: "error: " + err.Error() + "\n"}
	}
	balance, err := a.Svc.OwedTransactions.Balance(a.ctx, ledger.ID)
	if err != nil {
		return screen{Path: owedLedgerPath(name), Body: "error: " + err.Error() + "\n"}
	}
	lines := []string{
		fmt.Sprintf("name     : %s", ledger.Name),
		fmt.Sprintf("currency : %s", ledger.Code),
		fmt.Sprintf("balance  : %s", balance.Format(ledger.Code)),
		fmt.Sprintf("notes    : %s", ledger.Notes),
	}
	return screen{Path: owedLedgerPath(name), Body: strings.Join(lines, "\n") + "\n", Actions: []string{"transactions", "edit ledger"}}
}

func (a App) owedLedgerFormView() string {
	fields := []string{"name", "currency", "notes"}
	options := map[string][]string{"currency": a.currencyOptions()}
	return a.formViewWithOptions(fields, nil, options, nil)
}

func (a App) filteredOwedLedgerRows() ([]owedLedgerListRow, error) {
	ledgers, err := a.Svc.OwedLedgers.List(a.ctx)
	if err != nil {
		return nil, err
	}
	filter := strings.ToLower(a.listFilter())
	var out []owedLedgerListRow
	for _, ledger := range ledgers {
		if filter != "" && !strings.Contains(strings.ToLower(ledger.Name), filter) && !strings.Contains(strings.ToLower(ledger.Notes), filter) {
			continue
		}
		balance, err := a.Svc.OwedTransactions.Balance(a.ctx, ledger.ID)
		if err != nil {
			return nil, err
		}
		cell, err := a.owedLedgerMoneyCell(ledger, balance)
		if err != nil {
			return nil, err
		}
		out = append(out, owedLedgerListRow{Ledger: ledger, Balance: balance, Display: cell})
	}
	return out, nil
}

func (a App) owedLedgerMoneyCell(ledger repo.OwedLedger, amount money.Money) (component.Cell, error) {
	appCode := a.Config.Config.Currency
	if ledger.Code == appCode {
		return component.MoneyCell(amount, ledger.Code), nil
	}
	from, err := a.Svc.Currency.Get(a.ctx, ledger.Code)
	if err != nil {
		return component.MoneyCell(amount, ledger.Code), nil
	}
	to, err := a.Svc.Currency.Get(a.ctx, appCode)
	if err != nil {
		return component.MoneyCell(amount, ledger.Code), nil
	}
	converted, err := money.Convert(amount, from.RateToUSD, to.RateToUSD, to.Scale)
	if err != nil {
		return component.MoneyCell(amount, ledger.Code), nil
	}
	return component.MoneyCellWithTrailing(converted, appCode, "("+amount.Format(ledger.Code)+")"), nil
}

func (a App) owedDashboardContext() (string, error) {
	d, err := a.Svc.Dashboard.Summary(a.ctx)
	if err != nil {
		return "", err
	}
	values := alignedMoneyValues(component.MoneyCell(d.PplOweYou, a.Config.Config.Currency))
	return "ppl owe you : " + values[0], nil
}

func owedLedgerFormValues(ledger repo.OwedLedger) map[string]string {
	return map[string]string{"name": ledger.Name, "currency": ledger.Code, "notes": ledger.Notes}
}

func owedLedgerListHelp() []string {
	return []string{"type    : filter", "up/down : navigate", "enter   : open ledger", "ctrl+n  : new ledger", "ctrl+e  : edit ledger", "esc     : back", "?       : help", "ctrl-z  : undo"}
}
