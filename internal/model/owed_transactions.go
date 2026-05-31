package model

import (
	"fmt"
	"strings"

	"stuf/internal/component"
	"stuf/internal/money"
	"stuf/internal/repo"
	"stuf/internal/service"
)

func (a App) owedTransactionListKey(s, name string) App {
	if isNewKey(s) {
		ledger, err := a.Svc.OwedLedgers.GetByName(a.ctx, name)
		if err != nil {
			a.Error = err.Error()
			return a
		}
		a.Error = ""
		a.Form = map[string]string{"date": Today(), "currency": ledger.Code}
		a.Field = 0
		return a.navPush(owedTransactionAddPath(name), 0)
	}
	rows, err := a.owedTransactionRows(name)
	if err != nil {
		a.Error = err.Error()
		return a
	}
	if (isEditKey(s) || isDeleteKey(s)) && len(rows) > 0 {
		a = a.navSetMenu(clampListCursor(a.Menu, len(rows)))
		row := rows[a.Menu]
		if isEditKey(s) {
			a.Form = owedTransactionFormValues(row.Transaction)
			a.Field = 0
			return a.navPush(owedTransactionEditPath(name, row.Transaction.ID), 0)
		}
		entry, err := a.Svc.OwedTransactions.Delete(a.ctx, row.Transaction.ID)
		if err != nil {
			a.Error = err.Error()
			return a
		}
		a.History = append(a.History, entry)
		a.Error = ""
		return a.navSetMenu(clampListCursor(a.Menu, len(rows)-1))
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
		return a.navPush(owedTransactionDetailPath(rows[a.Menu].Transaction), 0)
	case "up", "shift+tab":
		if len(rows) > 0 {
			a = a.navSetMenu(clampListCursor(a.Menu-1, len(rows)))
		}
	case "down", "tab":
		if len(rows) > 0 {
			a = a.navSetMenu(clampListCursor(a.Menu+1, len(rows)))
		}
	}
	return a
}

func (a App) owedTransactionAddKey(s, name string) App {
	ledger, err := a.Svc.OwedLedgers.GetByName(a.ctx, name)
	if err != nil {
		a.Error = err.Error()
		return a
	}
	if a.Form["currency"] == "" {
		a.Form["currency"] = ledger.Code
	}
	if a.Form["date"] == "" {
		a.Form["date"] = Today()
	}
	next, submit := a.owedTransactionFormKey(s)
	if !submit {
		return next
	}
	txn, entry, err := next.Svc.OwedTransactions.Add(next.ctx, ledger.ID, next.Form["date"], next.Form["currency"], next.Form["amount"], next.Form["notes"])
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
	return next.selectOwedTransactionInList(name, txn.ID)
}

func (a App) owedTransactionEditKey(s, name string, id int64) App {
	ledger, err := a.Svc.OwedLedgers.GetByName(a.ctx, name)
	if err != nil {
		a.Error = err.Error()
		return a
	}
	_ = ledger
	next, submit := a.owedTransactionFormKey(s)
	if !submit {
		return next
	}
	updated, entry, err := next.Svc.OwedTransactions.Update(next.ctx, id, next.Form["date"], next.Form["currency"], next.Form["amount"], next.Form["notes"])
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
	return next.selectOwedTransactionInList(name, updated.ID)
}

func (a App) owedTransactionDetailKey(s, name string, id int64) App {
	action := a.actionIndex(s, 2)
	if action < 0 {
		return a
	}
	a = a.navSetMenu(action)
	switch action {
	case 0:
		txn, err := a.Svc.OwedTransactions.GetByID(a.ctx, id)
		if err != nil {
			a.Error = err.Error()
			return a
		}
		a.Form = owedTransactionFormValues(txn)
		a.Field = 0
		return a.navPush(owedTransactionEditPath(name, id), 0)
	case 1:
		entry, err := a.Svc.OwedTransactions.Delete(a.ctx, id)
		if err != nil {
			a.Error = err.Error()
			return a
		}
		a.History = append(a.History, entry)
		a.Error = ""
		return a.navReplace(owedTransactionListPath(name), 0)
	}
	return a
}

func (a App) owedTransactionFormKey(s string) (App, bool) {
	fields := []string{"date", "currency", "amount", "notes"}
	if isSubmitKey(s) {
		a.clearCurrentTextCursor(fields)
		return a, true
	}
	if a.Field == 1 {
		return a.currencyFieldKey(s, fields)
	}
	if s == "enter" {
		if a.Field >= len(fields) {
			return a, true
		}
		a.clearCurrentTextCursor(fields)
		a.Field++
		return a, false
	}
	return a.owedTransactionFormInputKey(s, fields), false
}

func (a App) owedTransactionFormInputKey(s string, fields []string) App {
	if strings.HasPrefix(s, "set ") {
		parts := strings.SplitN(strings.TrimPrefix(s, "set "), "=", 2)
		if len(parts) == 2 {
			a.Form[parts[0]] = normalizeOwedTransactionFieldValue(parts[0], parts[1])
			a.resetTextCursor(parts[0])
		}
		return a
	}
	fieldCount := len(fields) + 1
	switch s {
	case "tab", "down":
		a.clearCurrentTextCursor(fields)
		a.Field = (a.Field + 1) % fieldCount
	case "shift+tab", "up":
		a.clearCurrentTextCursor(fields)
		a.Field = (a.Field - 1 + fieldCount) % fieldCount
	case "left":
		if a.Field < len(fields) {
			a.moveTextCursor(fields[a.Field], -1)
		}
	case "right":
		if a.Field < len(fields) {
			a.moveTextCursor(fields[a.Field], 1)
		}
	case "backspace":
		if a.Field >= len(fields) {
			return a
		}
		field := fields[a.Field]
		if cursor := a.textCursor(field); cursor > 0 {
			runes := []rune(a.Form[field])
			a.Form[field] = string(append(runes[:cursor-1], runes[cursor:]...))
			a.setTextCursor(field, cursor-1)
		}
	default:
		if a.Field < len(fields) && (isTextInputKey(s) || (s == "?" && fields[a.Field] == "notes")) {
			field := fields[a.Field]
			current := a.Form[field]
			cursor := a.textCursor(field)
			runes := []rune(current)
			next := string(runes[:cursor]) + s + string(runes[cursor:])
			a.Form[field] = normalizeOwedTransactionFieldValue(field, next)
			if field == "date" || field == "amount" {
				a.resetTextCursor(field)
			} else {
				a.setTextCursor(field, cursor+len([]rune(s)))
			}
		}
	}
	return a
}

func (a App) owedTransactionRows(name string) ([]service.OwedTransactionRow, error) {
	ledger, err := a.Svc.OwedLedgers.GetByName(a.ctx, name)
	if err != nil {
		return nil, err
	}
	return a.Svc.OwedTransactions.ListWithBalances(a.ctx, ledger.ID)
}

func (a App) owedTransactionListScreen(name string) screen {
	ledger, err := a.Svc.OwedLedgers.GetByName(a.ctx, name)
	if err != nil {
		return screen{Path: owedTransactionListPath(name), Body: "error: " + err.Error() + "\n"}
	}
	rows, err := a.Svc.OwedTransactions.ListWithBalances(a.ctx, ledger.ID)
	if err != nil {
		return screen{Path: owedTransactionListPath(name), Body: "error: " + err.Error() + "\n"}
	}
	context := fmt.Sprintf("ledger   : %s\ncurrency : %s\nbalance  : %s", ledger.Name, ledger.Code, lastOwedBalance(rows, ledger).Format(ledger.Code))
	if len(rows) == 0 {
		return screen{Path: owedTransactionListPath(name), Context: context, Body: "  date | currency | amount | balance | notes\n  (no owed transactions yet)\n", Help: owedTransactionListHelp()}
	}
	tableRows := make([][]component.Cell, len(rows))
	for i, row := range rows {
		tableRows[i] = []component.Cell{
			component.TextCell(row.Transaction.Date),
			component.TextCell(row.Transaction.Code),
			component.TextCell(owedTransactionAmountDisplay(row.Transaction)),
			component.MoneyCell(row.Balance, ledger.Code),
			component.TextCell(row.Transaction.Notes),
		}
	}
	layout := component.NewTableLayoutCells([]string{"date", "currency", "amount", "balance", "notes"}, tableRows)
	lines := []string{layout.Header("  ")}
	for i, row := range tableRows {
		prefix := "  "
		if i == a.Menu {
			prefix = "> "
		}
		lines = append(lines, layout.RowCells(prefix, row))
	}
	return screen{Path: owedTransactionListPath(name), Context: context, Body: strings.Join(lines, "\n") + "\n", Help: owedTransactionListHelp()}
}

func (a App) owedTransactionAddScreen(name string) screen {
	ledger, err := a.Svc.OwedLedgers.GetByName(a.ctx, name)
	if err != nil {
		return screen{Path: owedTransactionAddPath(name), Body: "error: " + err.Error() + "\n"}
	}
	current, err := a.Svc.OwedTransactions.Balance(a.ctx, ledger.ID)
	if err != nil {
		return screen{Path: owedTransactionAddPath(name), Body: "error: " + err.Error() + "\n"}
	}
	context := fmt.Sprintf("ledger  : %s\ncurrent : %s", ledger.Name, current.Format(ledger.Code))
	fields := []string{"date", "currency", "amount", "notes"}
	options := map[string][]string{"currency": a.currencyOptions()}
	prefixes := map[string]string{"amount": a.Form["currency"]}
	if prefixes["amount"] == "" {
		prefixes["amount"] = ledger.Code
	}
	return screen{Path: owedTransactionAddPath(name), Context: context, Options: a.owedTransactionFormView(fields, options, prefixes), Help: a.formHelp(fields)}
}

func (a App) owedTransactionEditScreen(name string, id int64) screen {
	ledger, err := a.Svc.OwedLedgers.GetByName(a.ctx, name)
	if err != nil {
		return screen{Path: owedTransactionEditPath(name, id), Body: "error: " + err.Error() + "\n"}
	}
	if a.Form["date"] == "" {
		if txn, err := a.Svc.OwedTransactions.GetByID(a.ctx, id); err == nil {
			a.Form = owedTransactionFormValues(txn)
		}
	}
	context := fmt.Sprintf("ledger  : %s", ledger.Name)
	fields := []string{"date", "currency", "amount", "notes"}
	options := map[string][]string{"currency": a.currencyOptions()}
	prefixes := map[string]string{"amount": a.Form["currency"]}
	if prefixes["amount"] == "" {
		prefixes["amount"] = ledger.Code
	}
	return screen{Path: owedTransactionEditPath(name, id), Context: context, Options: a.owedTransactionFormView(fields, options, prefixes), Help: a.formHelp(fields)}
}

func (a App) owedTransactionDetailScreen(name string, id int64) screen {
	txn, err := a.Svc.OwedTransactions.GetByID(a.ctx, id)
	if err != nil {
		return screen{Path: owedTransactionDetailPathFor(name, id), Body: "error: " + err.Error() + "\n"}
	}
	balance, err := a.Svc.OwedTransactions.Balance(a.ctx, txn.LedgerID)
	if err != nil {
		return screen{Path: owedTransactionDetailPathFor(name, id), Body: "error: " + err.Error() + "\n"}
	}
	lines := []string{
		fmt.Sprintf("date     : %s", txn.Date),
		fmt.Sprintf("ledger   : %s", txn.LedgerName),
		fmt.Sprintf("currency : %s", txn.Code),
		fmt.Sprintf("amount   : %s", owedTransactionAmountDisplay(txn)),
	}
	if txn.Formula != "" {
		lines = append(lines, fmt.Sprintf("formula  : %s", txn.Formula))
	}
	lines = append(lines,
		fmt.Sprintf("balance  : %s", balance.Format(txn.LedgerCode)),
		fmt.Sprintf("notes    : %s", txn.Notes),
	)
	return screen{Path: owedTransactionDetailPath(txn), Body: strings.Join(lines, "\n") + "\n", Actions: []string{"edit transaction", "delete transaction"}}
}

func (a App) owedTransactionFormView(fields []string, options map[string][]string, prefixes map[string]string) string {
	var lines []string
	for i, field := range fields {
		if i > 0 {
			lines = append(lines, "")
		}
		prefix := "  "
		if i == a.Field {
			prefix = "> "
		}
		value := a.Form[field]
		if value == "" && field == "currency" {
			value = a.Config.Config.Currency
		}
		renderedValue := placeholder(value, placeholderFor(field))
		if field == "amount" {
			currency := prefixes[field]
			if currency == "" {
				currency = a.Form["currency"]
			}
			if i == a.Field {
				renderedValue = renderOwedAmountCaret(value, currency)
			} else {
				renderedValue = a.formatOwedAmountDisplay(value, currency)
			}
		} else if i == a.Field && isFormTextField(field, options) {
			renderedValue = renderCaret(value, placeholderFor(field), a.textCursor(field))
		}
		lines = append(lines, fmt.Sprintf("%s%d) %-9s: %s", prefix, i+1, field, renderedValue))
		if i == a.Field && options != nil && len(options[field]) > 0 {
			if field == "currency" {
				lines = append(lines, a.currencySelectLines(options[field])...)
				continue
			}
		}
	}
	confirmPrefix := "  "
	if a.Field == len(fields) {
		confirmPrefix = "> "
	}
	lines = append(lines, "", confirmPrefix+"[confirm]")
	return strings.Join(lines, "\n") + "\n"
}

func (a App) formatOwedAmountDisplay(value, currency string) string {
	if value == "" {
		return currency + " (type amount or =formula...)"
	}
	if strings.HasPrefix(value, "=") {
		amount, _, err := a.Svc.OwedTransactions.PreviewAmount(a.ctx, currency, value)
		if err != nil {
			return value
		}
		return amount.Format(currency) + " (" + value + ")"
	}
	return formatBalanceDisplay(value, currency)
}

func (a App) selectOwedTransactionInList(name string, id int64) App {
	rows, err := a.owedTransactionRows(name)
	if err != nil {
		a.Error = err.Error()
		return a
	}
	idx := 0
	for i, row := range rows {
		if row.Transaction.ID == id {
			idx = i
			break
		}
	}
	return a.navReplace(owedTransactionListPath(name), idx)
}

func owedTransactionFormValues(txn repo.OwedTransaction) map[string]string {
	amount := rawAmount(txn.Amount.Amount, txn.Amount.Scale)
	if txn.Formula != "" {
		amount = txn.Formula
	}
	return map[string]string{"date": txn.Date, "currency": txn.Code, "amount": amount, "notes": txn.Notes}
}

func owedTransactionAmountDisplay(txn repo.OwedTransaction) string {
	out := txn.Amount.Format(txn.Code)
	if txn.Formula != "" {
		out += " (" + txn.Formula + ")"
	}
	return out
}

func normalizeOwedTransactionFieldValue(field, value string) string {
	if field == "amount" {
		return sanitizeOwedAmount(value)
	}
	return normalizeFieldValue(field, value)
}

func sanitizeOwedAmount(input string) string {
	if !strings.HasPrefix(input, "=") {
		return sanitizeBalanceAmount(input)
	}
	var b strings.Builder
	for i, r := range input {
		switch {
		case i == 0 && r == '=':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '.' || r == '+' || r == '-' || r == '*' || r == '/' || r == '(' || r == ')' || r == ' ':
			b.WriteRune(r)
		}
	}
	return b.String()
}

func renderOwedAmountCaret(value, currency string) string {
	if strings.HasPrefix(value, "=") {
		return value + "|"
	}
	return renderBalanceCaret(value, currency)
}

func owedTransactionDetailPath(txn repo.OwedTransaction) string {
	return owedTransactionDetailPathFor(txn.LedgerName, txn.ID)
}

func owedTransactionDetailPathFor(name string, id int64) string {
	return fmt.Sprintf("/owed/ledgers/%s/transactions/%s/", name, service.OwedTransactionRef(id))
}

func lastOwedBalance(rows []service.OwedTransactionRow, ledger repo.OwedLedger) money.Money {
	if len(rows) == 0 {
		return money.Money{Scale: ledger.Scale}
	}
	return rows[len(rows)-1].Balance
}

func owedTransactionListHelp() []string {
	return []string{"up/down : navigate", "ctrl+n  : add transaction", "ctrl+e  : edit", "ctrl+d  : delete", "esc     : back", "?       : help", "ctrl-z  : undo"}
}
