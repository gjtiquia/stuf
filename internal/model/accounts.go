package model

import (
	"fmt"
	"strings"

	"stuf/internal/money"
)

type accountListRow struct {
	Name     string
	Balance  string
	Amount   money.Money
	Notes    string
	OnBudget bool
	AsOf     string
}

type accountListTableLayout struct {
	NameWidth    int
	BalanceWidth int
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
	if rows, err := next.accountListRows(false); err == nil {
		for i, row := range rows {
			if row.Name == acct.Name {
				listIdx = i
				break
			}
		}
	}
	next.Nav.Pop()
	next = next.syncFromNav()
	if next.Path == routeAccounts {
		next = next.navReplace(next.Path, 0)
	} else {
		next = next.navPush(routeAccounts, 0)
	}
	return next.navPush(routeAccountList, listIdx)
}

func (a App) accountListKey(s string, includeHidden bool) App {
	switch s {
	case "up", "k", "shift+tab":
		rows, err := a.accountListRows(includeHidden)
		if err != nil {
			a.Error = err.Error()
			return a
		}
		if len(rows) > 0 {
			a = a.navSetMenu((a.Menu - 1 + len(rows)) % len(rows))
		}
		return a
	case "down", "j", "tab":
		rows, err := a.accountListRows(includeHidden)
		if err != nil {
			a.Error = err.Error()
			return a
		}
		if len(rows) > 0 {
			a = a.navSetMenu((a.Menu + 1) % len(rows))
		}
		return a
	case "backspace":
		a.trimListFilter()
		a = a.navSetMenu(clampListCursor(a.Menu, a.accountListRowCount(includeHidden)))
		return a
	case "enter":
		rows, err := a.accountListRows(includeHidden)
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
		if isTextInputKey(s) {
			a.setListFilter(a.listFilter() + s)
			a = a.navSetMenu(0)
		}
		return a
	}
}

func (a App) accountListRowCount(includeHidden bool) int {
	rows, err := a.accountListRows(includeHidden)
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
		return a.navPush(accountBalancesPath(name), 0)
	case 1:
		return a.navPush(accountTransactionsPath(name), 0)
	case 2:
		a.Form = accountFormValues(acct.Name, acct.Code, acct.OnBudget, acct.Notes)
		a.Field = 0
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
	if next.Path != accountPath(updated.Name) {
		next = next.navReplace(accountPath(updated.Name), next.Menu)
	}
	return next
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

	totalStr := total.Format(appCurrency)
	if len(rows) == 0 {
		totalStr = zero(appCurrency)
	}
	var lines []string
	lines = append(lines, fmt.Sprintf("total       : %s", totalStr))
	if hasOnBudget {
		lines = append(lines, fmt.Sprintf("on-budget   : %s", onBudgetTotal.Format(appCurrency)))
	} else {
		lines = append(lines, fmt.Sprintf("on-budget   : %s", zero(appCurrency)))
	}
	if hasOffBudget {
		lines = append(lines, fmt.Sprintf("off-budget  : %s", offBudgetTotal.Format(appCurrency)))
	} else {
		lines = append(lines, fmt.Sprintf("off-budget  : %s", zero(appCurrency)))
	}
	return strings.Join(lines, "\n")
}

func (a App) accountList(includeHidden bool) string {
	allRows, err := a.accountListRowsWithFilter(includeHidden, "")
	if err != nil {
		return "error: " + err.Error() + "\n"
	}
	visible, err := a.accountListRows(includeHidden)
	if err != nil {
		return "error: " + err.Error() + "\n"
	}
	filter := a.listFilter()
	var lines []string
	if !includeHidden {
		lines = append(lines, accountSummary(allRows, a.Config.Config.Currency), "")
	}
	lines = append(lines, "> filter : "+placeholder(filter, "(type anything...)"), "")
	if len(visible) == 0 {
		lines = append(lines, "  (no results)")
		return strings.Join(lines, "\n") + "\n"
	}
	layout := accountListTableLayoutFor(visible, a.Config.Config.Currency)
	if includeHidden {
		lines = append(lines, layout.headerLine())
		for i, row := range visible {
			prefix := "  "
			if i == a.Menu {
				prefix = "> "
			}
			lines = append(lines, layout.rowLine(prefix, row.Name, row.Balance, row.Notes))
		}
		return strings.Join(lines, "\n") + "\n"
	}
	lines = appendAccountSection(lines, "on-budget accounts", visible, true, a.Menu, a.Config.Config.Currency, layout)
	lines = append(lines, "")
	lines = appendAccountSection(lines, "off-budget accounts", visible, false, a.Menu, a.Config.Config.Currency, layout)
	return strings.Join(lines, "\n") + "\n"
}

func accountListTableLayoutFor(rows []accountListRow, appCurrency string) accountListTableLayout {
	layout := accountListTableLayout{NameWidth: len("name"), BalanceWidth: len("balance")}
	if len(rows) == 0 {
		return layout
	}
	for _, onBudget := range []bool{true, false} {
		if total, ok := accountSectionTotal(rows, onBudget); ok {
			layout.NameWidth = max(layout.NameWidth, len("TOTAL"))
			layout.BalanceWidth = max(layout.BalanceWidth, len(total.Format(appCurrency)))
		}
	}
	for _, row := range rows {
		layout.NameWidth = max(layout.NameWidth, len(row.Name))
		layout.BalanceWidth = max(layout.BalanceWidth, len(row.Balance))
	}
	return layout
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

func (a App) accountListRows(includeHidden bool) ([]accountListRow, error) {
	return a.accountListRowsWithFilter(includeHidden, a.listFilter())
}

func (a App) accountListRowsWithFilter(includeHidden bool, filter string) ([]accountListRow, error) {
	accounts, err := a.Svc.Accounts.List(a.ctx, includeHidden)
	if err != nil {
		return nil, err
	}
	var visible []accountListRow
	for _, acct := range accounts {
		if includeHidden && !acct.Hidden {
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
		})
	}
	return visible, nil
}

func (a App) accountListBalance(code string, native money.Money) (money.Money, string, error) {
	appCur, err := a.Svc.Currency.Get(a.ctx, a.Config.Config.Currency)
	if err != nil {
		return money.Money{}, "", err
	}
	if code == appCur.Code {
		appAmount, err := native.ConvertToScale(appCur.Scale)
		if err != nil {
			return money.Money{}, "", err
		}
		return appAmount, appAmount.Format(appCur.Code), nil
	}
	cur, err := a.Svc.Currency.Get(a.ctx, code)
	if err != nil {
		return money.Money{}, "", err
	}
	appAmount, err := money.Convert(native, cur.RateToUSD, appCur.RateToUSD, appCur.Scale)
	if err != nil {
		return money.Money{}, "", err
	}
	return appAmount, fmt.Sprintf("%s (%s)", appAmount.Format(appCur.Code), native.Format(code)), nil
}

func appendAccountSection(lines []string, title string, rows []accountListRow, onBudget bool, selected int, appCurrency string, layout accountListTableLayout) []string {
	total, ok := accountSectionTotal(rows, onBudget)
	if !ok {
		return lines
	}
	lines = append(lines, "  "+title)
	lines = append(lines, layout.headerLine())
	lines = append(lines, layout.totalLine(total.Format(appCurrency)))
	lines = append(lines, "")
	for i, row := range rows {
		if row.OnBudget != onBudget {
			continue
		}
		prefix := "  "
		if i == selected {
			prefix = "> "
		}
		lines = append(lines, layout.rowLine(prefix, row.Name, row.Balance, row.Notes))
	}
	return lines
}

func (a App) accountDetailScreen(name string) screen {
	acct, err := a.Svc.Accounts.GetByName(a.ctx, name)
	if err != nil {
		return screen{Path: a.Path, Body: "error: " + err.Error() + "\n"}
	}
	bal, ok, _ := a.Svc.Accounts.CurrentBalance(a.ctx, acct.ID)
	amount, asOf := zero(acct.Code), "(no balance entered yet)"
	if ok {
		amount, asOf = bal.Amount.Format(acct.Code), bal.Date
	}
	hidden := ""
	actions := []string{"balances", "transactions (TODO)", "edit account", "hide account"}
	if acct.Hidden {
		hidden = "hidden    : true\n"
		actions = []string{"balances", "transactions (TODO)", "edit account", "show account"}
	}
	return screen{
		Path:    accountPath(name),
		Body:    fmt.Sprintf("name      : %s\nbalance   : %s\nas of     : %s\non-budget : %t\n%snotes     : %s\n", acct.Name, amount, asOf, acct.OnBudget, hidden, acct.Notes),
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
	})
}

func (l accountListTableLayout) headerLine() string {
	return fmt.Sprintf("  %-*s | %-*s | notes", l.NameWidth, "name", l.BalanceWidth, "balance")
}

func (l accountListTableLayout) totalLine(total string) string {
	return fmt.Sprintf("  %-*s | %-*s |", l.NameWidth, "TOTAL", l.BalanceWidth, total)
}

func (l accountListTableLayout) rowLine(prefix, name, balance, notes string) string {
	return fmt.Sprintf("%s%-*s | %-*s | %s", prefix, l.NameWidth, name, l.BalanceWidth, balance, notes)
}
