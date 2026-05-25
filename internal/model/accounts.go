package model

import (
	"fmt"
	"strings"

	"stuf/internal/component"
	"stuf/internal/money"
)

type accountListRow struct {
	Name     string
	Balance  component.Cell
	Amount   money.Money
	Notes    string
	OnBudget bool
	AsOf     string
	Hidden   bool
}

type accountVisibilityMode int

const (
	accountVisibilityNonHidden accountVisibilityMode = iota
	accountVisibilityHiddenOnly
	accountVisibilityAll
)

func (m accountVisibilityMode) label() string {
	switch m {
	case accountVisibilityHiddenOnly:
		return "hidden-only"
	case accountVisibilityAll:
		return "all"
	default:
		return "non-hidden"
	}
}

func (m accountVisibilityMode) next() accountVisibilityMode {
	switch m {
	case accountVisibilityNonHidden:
		return accountVisibilityHiddenOnly
	case accountVisibilityHiddenOnly:
		return accountVisibilityAll
	default:
		return accountVisibilityNonHidden
	}
}

func (a App) accountCreateKey(s string) App {
	if a.Form["currency"] == "" {
		a.Form["currency"] = a.Config.Config.Currency
	}
	if a.Form["on-budget"] == "" {
		a.Form["on-budget"] = "true"
	}
	next, submit := a.accountFormKey(s, nil)
	if !submit {
		return next
	}
	name := strings.TrimSpace(next.Form["name"])
	currency := strings.TrimSpace(next.Form["currency"])
	onBudget := parseBoolDefault(next.Form["on-budget"], true)
	acct, entry, err := next.Svc.Accounts.Create(next.ctx, name, currency, onBudget, next.Form["notes"])
	if err != nil {
		next.Error = err.Error()
		return next
	}
	next.History = append(next.History, entry)
	next.SelectedAccount = acct.Name
	next.Form = map[string]string{}
	next.Field = 0
	next.Error = ""
	listIdx := 0
	next.AccountVisible = accountVisibilityNonHidden
	if rows, err := next.accountListRows(); err == nil {
		for i, row := range rows {
			if row.Name == acct.Name {
				listIdx = i
				break
			}
		}
	}
	next.Nav.Pop()
	next = next.syncFromNav()
	if next.Path == routeAccountList {
		return next.navReplace(routeAccountList, listIdx)
	}
	return next.navPush(routeAccountList, listIdx)
}

func (a App) accountListKey(s string) App {
	if isNewKey(s) {
		a.Error = ""
		a.Field = 0
		return a.navPush(routeAccountCreate, 0)
	}
	if isEditKey(s) {
		rows, err := a.accountListRows()
		if err != nil {
			a.Error = err.Error()
			return a
		}
		if len(rows) == 0 {
			return a
		}
		a = a.navSetMenu(clampListCursor(a.Menu, len(rows)))
		acct, err := a.Svc.Accounts.GetByName(a.ctx, rows[a.Menu].Name)
		if err != nil {
			a.Error = err.Error()
			return a
		}
		a.Error = ""
		a = a.captureAccountListReturn(acct.Name)
		a.Form = accountFormValues(acct.Name, acct.Code, acct.OnBudget, acct.Notes)
		a.Field = 0
		return a.navPush(accountEditPathFor(acct.Name), 0)
	}
	if isHiddenCycleKey(s) {
		a.Error = ""
		a.AccountVisible = a.AccountVisible.next()
		return a.navSetMenu(clampListCursor(0, a.accountListRowCount()))
	}
	switch s {
	case "left":
		a.Error = ""
		return a.goBack()
	case "right":
		rows, err := a.accountListRows()
		if err != nil {
			a.Error = err.Error()
			return a
		}
		if len(rows) == 0 {
			return a
		}
		a = a.navSetMenu(clampListCursor(a.Menu, len(rows)))
		return a.navPush(accountPath(rows[a.Menu].Name), 0)
	case "up", "shift+tab":
		rows, err := a.accountListRows()
		if err != nil {
			a.Error = err.Error()
			return a
		}
		if len(rows) > 0 {
			a = a.navSetMenu((a.Menu - 1 + len(rows)) % len(rows))
		}
		return a
	case "down", "tab":
		rows, err := a.accountListRows()
		if err != nil {
			a.Error = err.Error()
			return a
		}
		if len(rows) > 0 {
			a = a.navSetMenu((a.Menu + 1) % len(rows))
		}
		return a
	case "backspace":
		input := newFilteredListInput(a.listFilter(), nil)
		updated, _ := input.handleKey("backspace")
		a.setListFilter(updated.value())
		a = a.navSetMenu(clampListCursor(a.Menu, a.accountListRowCount()))
		return a
	case "enter":
		rows, err := a.accountListRows()
		if err != nil {
			a.Error = err.Error()
			return a
		}
		if len(rows) == 0 {
			return a
		}
		a = a.navSetMenu(clampListCursor(a.Menu, len(rows)))
		return a.navPush(accountPath(rows[a.Menu].Name), 0)
	default:
		input := newFilteredListInput(a.listFilter(), nil)
		if updated, handled := input.handleKey(s); handled {
			a.setListFilter(updated.value())
			a = a.navSetMenu(0)
		}
		return a
	}
}

func (a App) accountListRowCount() int {
	rows, err := a.accountListRows()
	if err != nil {
		return 0
	}
	return len(rows)
}

func (a App) accountDetailKey(s, name string) App {
	acct, err := a.Svc.Accounts.GetByName(a.ctx, name)
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
		return a.navPush(accountBalanceListPath(name), 0)
	case 1:
		return a.navPush(accountTransactionsPath(name), 0)
	case 2:
		a.Form = accountFormValues(acct.Name, acct.Code, acct.OnBudget, acct.Notes)
		a.Field = 0
		a.ListReturn = listReturnState{}
		return a.navPush(accountEditPathFor(name), 0)
	case 3:
		updated, entry, err := a.Svc.Accounts.SetHidden(a.ctx, acct.ID, !acct.Hidden)
		if err != nil {
			a.Error = err.Error()
			return a
		}
		a.History = append(a.History, entry)
		return a.navReplace(accountPath(updated.Name), action)
	}
	return a
}

func (a App) accountEditKey(s, name string) App {
	acct, err := a.Svc.Accounts.GetByName(a.ctx, name)
	if err != nil {
		a.Error = err.Error()
		return a
	}
	locked := map[string]bool{}
	if has, err := a.Svc.Accounts.HasBalances(a.ctx, acct.ID); err == nil && has {
		locked["currency"] = true
	}
	next, submit := a.accountFormKey(s, locked)
	if !submit {
		return next
	}
	updated, entry, err := next.Svc.Accounts.Update(next.ctx, acct.ID, strings.TrimSpace(next.Form["name"]), strings.TrimSpace(next.Form["currency"]), parseBoolDefault(next.Form["on-budget"], acct.OnBudget), acct.Hidden, next.Form["notes"])
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
	if returned, ok := next.returnToListOrigin(updated.Name); ok {
		return returned
	}
	if next.Path != accountPath(updated.Name) {
		next = next.navReplace(accountPath(updated.Name), next.Menu)
	}
	return next
}

func (a App) selectAccountInCurrentList(name string) App {
	rows, err := a.accountListRows()
	if err != nil {
		a.Error = err.Error()
		return a
	}
	idx := clampListCursor(a.Menu, len(rows))
	for i, row := range rows {
		if row.Name == name {
			idx = i
			break
		}
	}
	return a.navReplace(routeAccountList, idx)
}

func accountSummary(rows []accountListRow, appCurrency string) string {
	total := money.Money{Scale: 2}
	if len(rows) > 0 {
		total = money.Money{Scale: rows[0].Amount.Scale}
	}
	for _, row := range rows {
		next, err := total.Add(row.Amount)
		if err == nil {
			total = next
		}
	}
	onBudgetTotal, hasOnBudget := accountSectionTotal(rows, true)
	offBudgetTotal, hasOffBudget := accountSectionTotal(rows, false)

	totalCell := component.MoneyCell(total, appCurrency)
	if len(rows) == 0 {
		totalCell = component.MoneyCell(money.Money{Scale: 2}, appCurrency)
	}
	onBudgetCell := component.MoneyCell(money.Money{Scale: 2}, appCurrency)
	if hasOnBudget {
		onBudgetCell = component.MoneyCell(onBudgetTotal, appCurrency)
	}
	offBudgetCell := component.MoneyCell(money.Money{Scale: 2}, appCurrency)
	if hasOffBudget {
		offBudgetCell = component.MoneyCell(offBudgetTotal, appCurrency)
	}
	values := alignedMoneyValues(totalCell, onBudgetCell, offBudgetCell)
	var lines []string
	lines = append(lines, fmt.Sprintf("total       : %s", values[0]))
	lines = append(lines, fmt.Sprintf("on-budget   : %s", values[1]))
	lines = append(lines, fmt.Sprintf("off-budget  : %s", values[2]))
	return strings.Join(lines, "\n")
}

func (a App) accountListContext() string {
	if a.AccountVisible != accountVisibilityNonHidden {
		return ""
	}
	allRows, err := a.accountListRowsWithFilter("")
	if err != nil {
		return "error: " + err.Error()
	}
	return accountSummary(allRows, a.Config.Config.Currency)
}

func (a App) accountList() string {
	context := a.accountListContext()
	body := a.accountListBody()
	if context == "" {
		return body
	}
	return context + "\n\n" + body
}

func (a App) accountListBody() string {
	visible, err := a.accountListRows()
	if err != nil {
		return "error: " + err.Error() + "\n"
	}
	filter := a.listFilter()
	var lines []string
	lines = append(lines, "showing : "+a.AccountVisible.label(), "")
	lines = append(lines, "> filter : "+placeholder(filter, "(type anything...)"), "")
	if len(visible) == 0 {
		lines = append(lines, "  (no results)")
		return strings.Join(lines, "\n") + "\n"
	}
	layout := accountListTableLayoutFor(visible, a.Config.Config.Currency, a.AccountVisible == accountVisibilityAll)
	if a.AccountVisible != accountVisibilityNonHidden {
		lines = append(lines, layout.Header("  "))
		for i, row := range visible {
			prefix := "  "
			if i == a.Menu {
				prefix = "> "
			}
			cells := []component.Cell{
				component.TextCell(row.Name),
				row.Balance,
				component.TextCell(row.Notes),
			}
			if a.AccountVisible == accountVisibilityAll {
				hidden := ""
				if row.Hidden {
					hidden = "true"
				}
				cells = append(cells, component.TextCell(hidden))
			}
			lines = append(lines, layout.RowCells(prefix, cells))
		}
		return strings.Join(lines, "\n") + "\n"
	}
	lines = appendAccountSection(lines, "on-budget accounts", visible, true, a.Menu, a.Config.Config.Currency, layout)
	lines = append(lines, "")
	lines = appendAccountSection(lines, "off-budget accounts", visible, false, a.Menu, a.Config.Config.Currency, layout)
	return strings.Join(lines, "\n") + "\n"
}

func accountListTableLayoutFor(rows []accountListRow, appCurrency string, includeHiddenColumn bool) component.TableLayout {
	tableRows := make([][]component.Cell, 0, len(rows)+2)
	for _, onBudget := range []bool{true, false} {
		if total, ok := accountSectionTotal(rows, onBudget); ok {
			cells := []component.Cell{
				component.TextCell("TOTAL"),
				component.MoneyCell(total, appCurrency),
				component.TextCell(""),
			}
			if includeHiddenColumn {
				cells = append(cells, component.TextCell(""))
			}
			tableRows = append(tableRows, cells)
		}
	}
	for _, row := range rows {
		cells := []component.Cell{
			component.TextCell(row.Name),
			row.Balance,
			component.TextCell(row.Notes),
		}
		if includeHiddenColumn {
			hidden := ""
			if row.Hidden {
				hidden = "true"
			}
			cells = append(cells, component.TextCell(hidden))
		}
		tableRows = append(tableRows, cells)
	}
	headers := []string{"name", "balance", "notes"}
	if includeHiddenColumn {
		headers = append(headers, "hidden")
	}
	return component.NewTableLayoutCells(headers, tableRows)
}

func accountSectionTotal(rows []accountListRow, onBudget bool) (money.Money, bool) {
	if len(rows) == 0 {
		return money.Money{}, false
	}
	total := money.Money{Scale: rows[0].Amount.Scale}
	found := false
	for _, row := range rows {
		if row.OnBudget != onBudget {
			continue
		}
		found = true
		next, err := total.Add(row.Amount)
		if err == nil {
			total = next
		}
	}
	return total, found
}

func (a App) accountListRows() ([]accountListRow, error) {
	return a.accountListRowsWithFilter(a.listFilter())
}

func (a App) accountListRowsWithFilter(filter string) ([]accountListRow, error) {
	accounts, err := a.Svc.Accounts.List(a.ctx, a.AccountVisible != accountVisibilityNonHidden)
	if err != nil {
		return nil, err
	}
	var visible []accountListRow
	for _, acct := range accounts {
		if a.AccountVisible == accountVisibilityNonHidden && acct.Hidden {
			continue
		}
		if a.AccountVisible == accountVisibilityHiddenOnly && !acct.Hidden {
			continue
		}
		if filter != "" && !strings.Contains(acct.Name, filter) && !strings.Contains(acct.Notes, filter) {
			continue
		}
		native := money.Money{Scale: acct.Scale}
		asOf := "(no balance entered yet)"
		if bal, ok, err := a.Svc.Accounts.CurrentBalance(a.ctx, acct.ID); err != nil {
			return nil, err
		} else if ok {
			native = bal.Amount
			asOf = bal.Date
		}
		appAmount, balance, err := a.accountListBalance(acct.Code, native)
		if err != nil {
			return nil, err
		}
		visible = append(visible, accountListRow{
			Name:     acct.Name,
			Balance:  balance,
			Amount:   appAmount,
			Notes:    acct.Notes,
			OnBudget: acct.OnBudget,
			AsOf:     asOf,
			Hidden:   acct.Hidden,
		})
	}
	return visible, nil
}

func (a App) accountListBalance(code string, native money.Money) (money.Money, component.Cell, error) {
	appCur, err := a.Svc.Currency.Get(a.ctx, a.Config.Config.Currency)
	if err != nil {
		return money.Money{}, component.Cell{}, err
	}
	if code == appCur.Code {
		appAmount, err := native.ConvertToScale(appCur.Scale)
		if err != nil {
			return money.Money{}, component.Cell{}, err
		}
		return appAmount, component.MoneyCell(appAmount, appCur.Code), nil
	}
	cur, err := a.Svc.Currency.Get(a.ctx, code)
	if err != nil {
		return money.Money{}, component.Cell{}, err
	}
	appAmount, err := money.Convert(native, cur.RateToUSD, appCur.RateToUSD, appCur.Scale)
	if err != nil {
		return money.Money{}, component.Cell{}, err
	}
	return appAmount, component.MoneyCellWithTrailing(appAmount, appCur.Code, fmt.Sprintf("(%s)", native.Format(code))), nil
}

func appendAccountSection(lines []string, title string, rows []accountListRow, onBudget bool, selected int, appCurrency string, layout component.TableLayout) []string {
	total, ok := accountSectionTotal(rows, onBudget)
	if !ok {
		return lines
	}
	lines = append(lines, "  "+title)
	lines = append(lines, layout.Header("  "))
	lines = append(lines, layout.RowCells("  ", []component.Cell{
		component.TextCell("TOTAL"),
		component.MoneyCell(total, appCurrency),
		component.TextCell(""),
	}))
	lines = append(lines, "")
	for i, row := range rows {
		if row.OnBudget != onBudget {
			continue
		}
		prefix := "  "
		if i == selected {
			prefix = "> "
		}
		lines = append(lines, layout.RowCells(prefix, []component.Cell{
			component.TextCell(row.Name),
			row.Balance,
			component.TextCell(row.Notes),
		}))
	}
	return lines
}

func (a App) accountDetailScreen(name string) screen {
	acct, err := a.Svc.Accounts.GetByName(a.ctx, name)
	if err != nil {
		return screen{Path: a.Path, Body: "error: " + err.Error() + "\n"}
	}
	context, err := a.accountDashboardContext(name)
	if err != nil {
		return screen{Path: a.Path, Body: "error: " + err.Error() + "\n"}
	}
	actions := []string{"balances", "transactions (TODO)", "edit account", "hide account"}
	if acct.Hidden {
		actions = []string{"balances", "transactions (TODO)", "edit account", "show account"}
	}
	return screen{
		Path:    accountPath(name),
		Context: context,
		Actions: actions,
	}
}

func (a App) accountEditScreen() screen {
	name, _ := accountEditName(a.Path)
	acct, err := a.Svc.Accounts.GetByName(a.ctx, name)
	if err != nil {
		return screen{Path: a.Path, Body: "error: " + err.Error() + "\n"}
	}
	locked := map[string]string{}
	if has, err := a.Svc.Accounts.HasBalances(a.ctx, acct.ID); err == nil && has {
		locked["currency"] = acct.Code + " (locked because balances exist)"
	}
	return screen{Path: a.Path, Body: a.accountFormView(locked), Help: a.accountFormHelp()}
}

func (a App) accountFormView(locked map[string]string) string {
	return a.formViewWithOptions([]string{"name", "currency", "on-budget", "notes"}, locked, map[string][]string{
		"currency":  a.currencyOptions(),
		"on-budget": {"true", "false"},
	}, nil)
}
