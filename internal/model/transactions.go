package model

import (
	"fmt"
	"strconv"
	"strings"

	"stuf/internal/component"
	"stuf/internal/money"
	"stuf/internal/repo"
	"stuf/internal/service"
)

func (a App) transactionListKey(s string, accountID int64) App {
	rows, err := a.transactionRows(accountID, nil)
	if err != nil {
		a.Error = err.Error()
		return a
	}
	if isNewKey(s) {
		a.Error = ""
		a.Form = map[string]string{"date": Today(), "type": service.TransactionTypeExpense, "currency": a.Config.Config.Currency}
		a.Field = 0
		return a.navPush(routeTransactionAdd, 0)
	}
	if isEditKey(s) && len(rows) > 0 {
		a = a.navSetMenu(clampListCursor(a.Menu, len(rows)))
		row := rows[a.Menu]
		txn, err := a.Svc.Transactions.GetByID(a.ctx, row.ID)
		if err != nil {
			a.Error = err.Error()
			return a
		}
		a.Form = a.transactionFormValues(txn)
		a.Field = 0
		return a.navPush(transactionEditPath(row.Ref), 0)
	}
	if isDeleteKey(s) && len(rows) > 0 {
		a = a.navSetMenu(clampListCursor(a.Menu, len(rows)))
		entry, err := a.Svc.Transactions.Delete(a.ctx, rows[a.Menu].ID)
		if err != nil {
			a.Error = err.Error()
			return a
		}
		a.History = append(a.History, entry)
		a.Error = ""
		return a.navSetMenu(clampListCursor(a.Menu, a.transactionListRowCount(accountID)))
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
		return a.navPush(transactionRefPath(rows[a.Menu].Ref), 0)
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
			nextRows, _ := a.transactionRows(accountID, nil)
			a = a.navSetMenu(clampListCursor(result.menu, len(nextRows)))
		}
	}
	return a
}

func (a App) accountTransactionListKey(s, name string) App {
	acct, err := a.Svc.Accounts.GetByName(a.ctx, name)
	if err != nil {
		a.Error = err.Error()
		return a
	}
	if isNewKey(s) {
		a.Error = ""
		a.Form = map[string]string{"date": Today(), "type": service.TransactionTypeExpense, "currency": acct.Code, "account": acct.Name}
		a.Field = 0
		return a.navPush(accountTransactionAddPath(name), 0)
	}
	return a.transactionListKey(s, acct.ID)
}

func (a App) transactionChildrenListKey(s, ref string) App {
	parent, err := a.transactionByRefString(ref)
	if err != nil {
		a.Error = err.Error()
		return a
	}
	rows, err := a.transactionRows(0, &parent.ID)
	if err != nil {
		a.Error = err.Error()
		return a
	}
	if isNewKey(s) {
		a.Error = ""
		a.Form = map[string]string{"date": parent.Date, "type": parent.Type, "currency": parent.Code, "account": parent.AccountName}
		a.Field = 0
		return a.navPush(transactionChildAddPath(ref), 0)
	}
	if isEditKey(s) && len(rows) > 0 {
		a = a.navSetMenu(clampListCursor(a.Menu, len(rows)))
		txn, err := a.Svc.Transactions.GetByID(a.ctx, rows[a.Menu].ID)
		if err != nil {
			a.Error = err.Error()
			return a
		}
		a.Form = a.transactionFormValues(txn)
		a.Field = 0
		return a.navPush(transactionEditPath(rows[a.Menu].Ref), 0)
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
		return a.navPush(transactionRefPath(rows[a.Menu].Ref), 0)
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
			nextRows, _ := a.transactionRows(0, &parent.ID)
			a = a.navSetMenu(clampListCursor(result.menu, len(nextRows)))
		}
	}
	return a
}

func (a App) transactionAddKey(s string, parentID *int64, accountID int64) App {
	next, submit := a.transactionFormKey(s, nil)
	if !submit {
		return next
	}
	return next.submitTransactionAdd(parentID, accountID)
}

func (a App) accountTransactionAddKey(s, name string) App {
	acct, err := a.Svc.Accounts.GetByName(a.ctx, name)
	if err != nil {
		a.Error = err.Error()
		return a
	}
	if a.Form["account"] == "" {
		a.Form["account"] = acct.Name
	}
	if a.Form["currency"] == "" {
		a.Form["currency"] = acct.Code
	}
	next, submit := a.transactionFormKey(s, nil)
	if !submit {
		return next
	}
	return next.submitTransactionAdd(nil, acct.ID)
}

func (a App) transactionChildAddKey(s, ref string) App {
	parent, err := a.transactionByRefString(ref)
	if err != nil {
		a.Error = err.Error()
		return a
	}
	if a.Form["account"] == "" {
		a.Form["account"] = parent.AccountName
	}
	locked := map[string]bool{"account": true}
	next, submit := a.transactionFormKey(s, locked)
	if !submit {
		return next
	}
	return next.submitTransactionAdd(&parent.ID, parent.AccountID)
}

func (a App) transactionEditKey(s, ref string) App {
	txn, err := a.transactionByRefString(ref)
	if err != nil {
		a.Error = err.Error()
		return a
	}
	locked := map[string]bool{}
	if txn.ParentID != nil {
		locked["account"] = true
	}
	next, submit := a.transactionFormKey(s, locked)
	if !submit {
		return next
	}
	updated, entry, err := next.Svc.Transactions.Update(next.ctx, txn.ID, next.Form["date"], next.Form["type"], strings.TrimSpace(next.Form["currency"]), next.Form["amount"], next.Form["notes"], splitTagNames(next.Form["tags"]))
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
	return next.navReplace(transactionRefPath(service.TransactionRef(updated.Ref)), next.Menu)
}

func (a App) transactionDetailKey(s, ref string) App {
	txn, err := a.transactionByRefString(ref)
	if err != nil {
		a.Error = err.Error()
		return a
	}
	action := a.actionIndex(s, 4)
	if action < 0 {
		return a
	}
	a = a.navSetMenu(action)
	switch action {
	case 0:
		return a.navPush(transactionChildrenListPath(ref), 0)
	case 1:
		a.Form = a.transactionFormValues(txn)
		a.Field = 0
		return a.navPush(transactionEditPath(ref), 0)
	case 2:
		a.Form = map[string]string{"date": txn.Date, "type": txn.Type, "currency": txn.Code, "account": txn.AccountName}
		a.Field = 0
		return a.navPush(transactionChildAddPath(ref), 0)
	case 3:
		entry, err := a.Svc.Transactions.Delete(a.ctx, txn.ID)
		if err != nil {
			a.Error = err.Error()
			return a
		}
		a.History = append(a.History, entry)
		a.Error = ""
		return a.goBack()
	}
	return a
}

func (a App) submitTransactionAdd(parentID *int64, fallbackAccountID int64) App {
	accountID := fallbackAccountID
	if parentID == nil {
		name := strings.TrimSpace(a.Form["account"])
		if name != "" {
			acct, err := a.Svc.Accounts.GetByName(a.ctx, name)
			if err != nil {
				a.Error = err.Error()
				return a
			}
			accountID = acct.ID
		}
	}
	txn, entry, err := a.Svc.Transactions.Add(a.ctx, parentID, accountID, a.Form["type"], strings.TrimSpace(a.Form["currency"]), a.Form["date"], a.Form["amount"], a.Form["notes"], splitTagNames(a.Form["tags"]))
	if err != nil {
		a.Error = err.Error()
		return a
	}
	a.History = append(a.History, entry)
	a.Form = map[string]string{}
	a.Field = 0
	a.Error = ""
	a.Nav.Pop()
	a = a.syncFromNav()
	return a.selectTransactionInCurrentList(service.TransactionRef(txn.Ref))
}

func (a App) transactionFormKey(s string, locked map[string]bool) (App, bool) {
	fields := []string{"date", "type", "currency", "amount", "account", "tags", "notes"}
	if a.Form["type"] == "" {
		a.Form["type"] = service.TransactionTypeExpense
	}
	if a.Form["date"] == "" {
		a.Form["date"] = Today()
	}
	if a.Form["currency"] == "" {
		a.Form["currency"] = a.Config.Config.Currency
	}
	if isSubmitKey(s) {
		a.clearCurrentTextCursor(fields)
		return a, true
	}
	if a.Field == 1 {
		return a.selectFieldKey(s, "type", []string{service.TransactionTypeExpense, service.TransactionTypeIncome}, fields)
	}
	if a.Field == 2 {
		return a.currencyFieldKey(s, fields)
	}
	if a.Field == 4 && locked != nil && locked["account"] {
		switch s {
		case "enter", "tab", "down":
			a.Field = 5
		case "shift+tab", "up":
			a.Field = 3
		}
		return a, false
	}
	if a.Field == 4 {
		return a.transactionAccountFieldKey(s, fields)
	}
	if a.Field == 5 {
		return a.tagFieldKey(s, fields)
	}
	return a.submitFormKey(s, fields)
}

func (a App) transactionFormView(locked map[string]string) string {
	fields := []string{"date", "type", "currency", "amount", "account", "tags", "notes"}
	options := map[string][]string{"currency": a.currencyOptions(), "type": []string{service.TransactionTypeExpense, service.TransactionTypeIncome}}
	return a.formViewWithOptions(fields, locked, options, nil)
}

func (a App) transactionAccountOptions() []string {
	accounts, err := a.Svc.Accounts.List(a.ctx, true)
	if err != nil {
		return nil
	}
	out := make([]string, len(accounts))
	for i, account := range accounts {
		out[i] = account.Name
	}
	return out
}

func (a App) currentTransactionAccountOptions() []tagOption {
	filter := a.accountFilter()
	var opts []tagOption
	for _, name := range a.transactionAccountOptions() {
		if filter != "" && !strings.Contains(name, filter) {
			continue
		}
		opts = append(opts, tagOption{Name: name})
	}
	return opts
}

func (a App) transactionAccountFieldKey(s string, fields []string) (App, bool) {
	options := a.currentTransactionAccountOptions()
	cursor := clampCursor(parseFormInt(a.Form[accountCursorKey]), len(options))
	a.setAccountSelectCursor(cursor)
	switch s {
	case "down":
		if len(options) > 0 {
			a.setAccountSelectCursor((cursor + 1) % len(options))
		}
	case "up":
		if len(options) > 0 {
			a.setAccountSelectCursor((cursor - 1 + len(options)) % len(options))
		}
	case "right":
		if len(options) > 0 {
			page := min(a.accountSelectPage()+1, tagPageCount(len(options))-1)
			a.setAccountSelectPage(page)
			a.setAccountSelectCursor(min(page*tagPageSize, len(options)-1))
		}
	case "left":
		if len(options) > 0 {
			page := max(a.accountSelectPage()-1, 0)
			a.setAccountSelectPage(page)
			a.setAccountSelectCursor(min(page*tagPageSize, len(options)-1))
		}
	case "backspace":
		if a.accountFilter() == "" {
			a.Form["account"] = ""
			return a, false
		}
		a.setAccountFilter(trimLastRune(a.accountFilter()))
		a.resetAccountSelectCursor()
	case "tab":
		a.clearAccountSelectState()
		a.Field = min(a.Field+1, len(fields))
	case "shift+tab":
		a.clearAccountSelectState()
		a.Field = max(a.Field-1, 0)
	case "enter":
		if len(options) == 0 {
			a.clearAccountSelectState()
			a.Field = min(a.Field+1, len(fields))
			return a, false
		}
		a.Form["account"] = options[cursor].Name
		a.clearAccountSelectState()
	default:
		input := newFilteredListInput(a.accountFilter(), sanitizeSlug)
		if updated, handled := input.handleKey(s); handled {
			a.setAccountFilter(updated.value())
			a.resetAccountSelectCursor()
		}
	}
	return a, false
}

func (a App) transactionAccountSelectLines() []string {
	filter := a.accountFilter()
	options := a.currentTransactionAccountOptions()
	cursor := clampCursor(parseFormInt(a.Form[accountCursorKey]), len(options))
	page := min(a.accountSelectPage(), tagPageCount(len(options))-1)
	start := page * tagPageSize
	end := min(start+tagPageSize, len(options))
	lines := []string{"", fmt.Sprintf("   > filter  : %s", placeholder(filter, "(type anything...)")), ""}
	if len(options) == 0 {
		lines = append(lines, "     (no matching accounts)", "", "     [00/00]")
		return lines
	}
	for i, option := range options[start:end] {
		prefix := "       "
		if start+i == cursor {
			prefix = "     > "
		}
		lines = append(lines, prefix+option.Name)
	}
	lines = append(lines, "", fmt.Sprintf("     [%02d/%02d]", cursor+1, len(options)))
	return lines
}

func (a App) transactionListScreen(accountID int64) screen {
	return screen{Path: a.Path, Body: a.transactionListBody(accountID, nil), Help: transactionListHelp()}
}

func (a App) accountTransactionListScreen(name string) screen {
	acct, err := a.Svc.Accounts.GetByName(a.ctx, name)
	if err != nil {
		return screen{Path: a.Path, Body: "error: " + err.Error() + "\n"}
	}
	return screen{Path: a.Path, Context: "account : " + name, Body: a.transactionListBody(acct.ID, nil), Help: transactionListHelp()}
}

func (a App) transactionAddScreen(parentID *int64, accountID int64) screen {
	return screen{Path: a.Path, Body: a.transactionFormView(nil), Help: transactionFormHelp()}
}

func (a App) accountTransactionAddScreen(name string) screen {
	if a.Form["account"] == "" {
		a.Form["account"] = name
	}
	return screen{Path: a.Path, Context: "account : " + name, Body: a.transactionFormView(nil), Help: transactionFormHelp()}
}

func (a App) transactionChildAddScreen(ref string) screen {
	parent, err := a.transactionByRefString(ref)
	if err != nil {
		return screen{Path: a.Path, Body: "error: " + err.Error() + "\n"}
	}
	if a.Form["account"] == "" {
		a.Form["account"] = parent.AccountName
	}
	locked := map[string]string{"account": parent.AccountName + " (locked to parent)"}
	context := fmt.Sprintf("parent    : %s\nremaining : %s", ref, a.transactionRemaining(parent).Format(parent.Code))
	return screen{Path: a.Path, Context: context, Body: a.transactionFormView(locked), Help: transactionFormHelp()}
}

func (a App) transactionEditScreen(ref string) screen {
	txn, err := a.transactionByRefString(ref)
	if err != nil {
		return screen{Path: a.Path, Body: "error: " + err.Error() + "\n"}
	}
	if a.Form["date"] == "" {
		a.Form = a.transactionFormValues(txn)
	}
	locked := map[string]string{}
	if txn.ParentID != nil {
		locked["account"] = txn.AccountName + " (locked to parent)"
	}
	return screen{Path: a.Path, Body: a.transactionFormView(locked), Help: transactionFormHelp()}
}

func (a App) transactionDetailScreen(ref string) screen {
	txn, err := a.transactionByRefString(ref)
	if err != nil {
		return screen{Path: a.Path, Body: "error: " + err.Error() + "\n"}
	}
	tags := a.transactionTagNames(txn.ID)
	children := a.transactionChildrenTotal(txn)
	remaining := a.transactionRemaining(txn)
	body := fmt.Sprintf("date      : %s\ntype      : %s\namount    : %s\ncurrency  : %s\naccount   : %s\nchildren  : %s\nremaining : %s\ntags      : %s\nnotes     : %s\n",
		txn.Date, txn.Type, txn.Amount.Format(txn.Code), txn.Code, txn.AccountName, children.Format(txn.Code), remaining.Format(txn.Code), formatTags(tags, nil), txn.Notes)
	return screen{Path: a.Path, Body: body, Actions: []string{"children", "edit transaction", "add child transaction", "delete transaction"}}
}

func (a App) transactionChildrenListScreen(ref string) screen {
	parent, err := a.transactionByRefString(ref)
	if err != nil {
		return screen{Path: a.Path, Body: "error: " + err.Error() + "\n"}
	}
	context := fmt.Sprintf("parent    : %s\namount    : %s\nexplained : %s\nremaining : %s", ref, parent.Amount.Format(parent.Code), a.transactionChildrenTotal(parent).Format(parent.Code), a.transactionRemaining(parent).Format(parent.Code))
	return screen{Path: a.Path, Context: context, Body: a.transactionListBody(0, &parent.ID), Help: transactionListHelp()}
}

func (a App) transactionListBody(accountID int64, parentID *int64) string {
	rows, err := a.transactionRows(accountID, parentID)
	if err != nil {
		return "error: " + err.Error() + "\n"
	}
	filter := a.listFilter()
	var lines []string
	lines = append(lines, "> filter : "+placeholder(filter, "(type anything...)"), "")
	headers := []string{"date", "type", "amount", "account", "tags", "notes"}
	tableRows := make([][]component.Cell, 0, len(rows))
	for _, row := range rows {
		date := strings.Repeat("  ", row.Depth) + row.Date
		tableRows = append(tableRows, []component.Cell{
			component.TextCell(date),
			component.TextCell(row.Type),
			component.TextCell(row.Amount),
			component.TextCell(row.Account),
			component.TextCell(formatTags(row.Tags, nil)),
			component.TextCell(row.Notes),
		})
	}
	layout := component.NewTableLayoutCells(headers, tableRows)
	lines = append(lines, layout.Header("  "))
	if len(rows) == 0 {
		lines = append(lines, "  (no transactions yet)")
		return strings.Join(lines, "\n") + "\n"
	}
	for i := range rows {
		prefix := "  "
		if i == a.Menu {
			prefix = "> "
		}
		lines = append(lines, layout.RowCells(prefix, tableRows[i]))
	}
	return strings.Join(lines, "\n") + "\n"
}

func (a App) transactionRows(accountID int64, parentID *int64) ([]transactionListRow, error) {
	var txns []repo.Transaction
	var err error
	switch {
	case parentID != nil:
		txns, err = a.Svc.Transactions.ListByParent(a.ctx, *parentID)
	case accountID != 0:
		txns, err = a.Svc.Transactions.ListByAccount(a.ctx, accountID)
	default:
		txns, err = a.Svc.Transactions.List(a.ctx)
	}
	if err != nil {
		return nil, err
	}
	depths := map[int64]int{}
	for _, txn := range txns {
		depths[txn.ID] = a.transactionDepth(txn)
	}
	filter := parseTransactionFilter(a.listFilter())
	var out []transactionListRow
	for _, txn := range txns {
		row, err := a.transactionRow(txn, depths[txn.ID])
		if err != nil {
			return nil, err
		}
		if !filter.Empty() && !filter.Match(row) {
			continue
		}
		out = append(out, row)
	}
	return out, nil
}

func (a App) transactionRow(txn repo.Transaction, depth int) (transactionListRow, error) {
	amount := txn.Amount.Format(txn.Code)
	tags := a.transactionTagNames(txn.ID)
	children, _ := a.Svc.Transactions.ListByParent(a.ctx, txn.ID)
	return transactionListRow{
		ID:          txn.ID,
		Ref:         service.TransactionRef(txn.Ref),
		RefNumber:   txn.Ref,
		ParentID:    txn.ParentID,
		Date:        txn.Date,
		Type:        txn.Type,
		Amount:      amount,
		Account:     txn.AccountName,
		Currency:    txn.Code,
		Notes:       txn.Notes,
		Tags:        tags,
		Depth:       depth,
		Selectable:  true,
		HasChildren: len(children) > 0,
	}, nil
}

func (a App) transactionDepth(txn repo.Transaction) int {
	depth := 0
	for txn.ParentID != nil {
		parent, err := a.Svc.Transactions.GetByID(a.ctx, *txn.ParentID)
		if err != nil {
			break
		}
		depth++
		txn = parent
	}
	return depth
}

func (a App) transactionListRowCount(accountID int64) int {
	rows, err := a.transactionRows(accountID, nil)
	if err != nil {
		return 0
	}
	return len(rows)
}

func (a App) transactionChildrenRowCount(parentID int64) int {
	rows, err := a.transactionRows(0, &parentID)
	if err != nil {
		return 0
	}
	return len(rows)
}

func (a App) selectTransactionInCurrentList(ref string) App {
	rows, err := a.transactionRows(0, nil)
	if name, ok := accountTransactionListName(a.Path); ok {
		if acct, acctErr := a.Svc.Accounts.GetByName(a.ctx, name); acctErr == nil {
			rows, err = a.transactionRows(acct.ID, nil)
		}
	}
	if childRef, ok := transactionChildrenListRef(a.Path); ok {
		if parent, parentErr := a.transactionByRefString(childRef); parentErr == nil {
			rows, err = a.transactionRows(0, &parent.ID)
		}
	}
	if err != nil {
		a.Error = err.Error()
		return a
	}
	idx := 0
	for i, row := range rows {
		if row.Ref == ref {
			idx = i
			break
		}
	}
	return a.navReplace(a.Path, idx)
}

func (a App) transactionByRefString(ref string) (repo.Transaction, error) {
	n, err := strconv.ParseInt(strings.TrimPrefix(ref, "tx-"), 10, 64)
	if err != nil {
		return repo.Transaction{}, err
	}
	return a.Svc.Transactions.GetByRef(a.ctx, n)
}

func (a App) transactionFormValues(txn repo.Transaction) map[string]string {
	return map[string]string{
		"date":     txn.Date,
		"type":     txn.Type,
		"amount":   rawAmount(txn.Amount.Amount, txn.Amount.Scale),
		"currency": txn.Code,
		"account":  txn.AccountName,
		"tags":     joinTagNames(a.transactionTagNames(txn.ID)),
		"notes":    txn.Notes,
	}
}

func (a App) transactionTagNames(id int64) []string {
	tags, err := a.Svc.Transactions.TagsByTransactionID(a.ctx, id)
	if err != nil {
		return nil
	}
	return tagNamesFromRepo(tags)
}

func tagNamesFromRepo(tags []repo.Tag) []string {
	out := make([]string, len(tags))
	for i, tag := range tags {
		out[i] = tag.Name
	}
	return out
}

func (a App) transactionChildrenTotal(txn repo.Transaction) money.Money {
	total := money.Money{Scale: txn.Amount.Scale}
	children, err := a.Svc.Transactions.ListByParent(a.ctx, txn.ID)
	if err != nil {
		return total
	}
	for _, child := range children {
		amount := child.Amount
		if child.Code != txn.Code {
			from, ferr := a.Svc.Currency.Get(a.ctx, child.Code)
			to, terr := a.Svc.Currency.Get(a.ctx, txn.Code)
			if ferr == nil && terr == nil {
				if converted, cerr := money.Convert(child.Amount, from.RateToUSD, to.RateToUSD, txn.Amount.Scale); cerr == nil {
					amount = converted
				}
			}
		}
		if next, err := total.Add(amount); err == nil {
			total = next
		}
	}
	return total
}

func (a App) transactionRemaining(txn repo.Transaction) money.Money {
	total := a.transactionChildrenTotal(txn)
	if remaining, err := txn.Amount.Sub(total); err == nil {
		return remaining
	}
	return money.Money{Scale: txn.Amount.Scale}
}

func transactionListHelp() []string {
	return []string{"type          : filter", "h/l           : type in filter", "up/down       : navigate", "left/right    : back/open", "ctrl+n        : new", "ctrl+e        : edit", "ctrl+d        : delete", "enter         : open", "esc           : back", "?             : help"}
}

func transactionFormHelp() []string {
	return []string{"type    : enter text", "up/down : choose type", "tab     : navigate", "enter   : confirm", "ctrl+s  : submit", "esc     : back", "?       : help"}
}
