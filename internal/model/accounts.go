package model

import (
	"fmt"
	"strings"

	"stuf/internal/component"
	"stuf/internal/money"
	"stuf/internal/repo"
)

type accountListRow struct {
	ID           int64
	Name         string
	Balance      component.Cell
	Amount       money.Money
	Notes        string
	Tags         []string
	Currency     string
	CurrencyName string
	OnBudget     bool
	AsOf         string
	Hidden       bool
	Depth        int
	Virtual      bool
	Match        bool
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
	acct, entry, err := next.Svc.Accounts.CreateWithTags(next.ctx, name, currency, onBudget, next.Form["notes"], splitTagNames(next.Form["tags"]))
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

func (a App) accountChildCreateKey(s, parentName string) App {
	parent, err := a.Svc.Accounts.GetByName(a.ctx, parentName)
	if err != nil {
		a.Error = err.Error()
		return a
	}
	if a.Form["currency"] == "" {
		a.Form["currency"] = a.Config.Config.Currency
	}
	next, submit := a.childAccountFormKey(s, nil)
	if !submit {
		return next
	}
	name := strings.TrimSpace(next.Form["name"])
	currency := strings.TrimSpace(next.Form["currency"])
	acct, entry, err := next.Svc.Accounts.CreateChildWithTags(next.ctx, parent.ID, name, currency, next.Form["notes"], splitTagNames(next.Form["tags"]))
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
	if rows, err := next.childAccountListRows(parent.ID); err == nil {
		for i, row := range rows {
			if row.Name == acct.Name {
				listIdx = i
				break
			}
		}
	}
	next.Nav.Pop()
	next = next.syncFromNav()
	return next.navReplace(accountChildrenListPath(parentName), listIdx)
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
		idx := a.accountSelectableIndex(rows, a.Menu)
		acct, err := a.Svc.Accounts.GetByName(a.ctx, rows[idx].Name)
		if err != nil {
			a.Error = err.Error()
			return a
		}
		a.Error = ""
		a = a.captureAccountListReturn(acct.Name)
		a.Form = accountFormValues(acct.Name, acct.Code, acct.OnBudget, acct.Notes, a.directTagNames(acct.ID))
		a.Field = 0
		return a.navPush(accountEditPathFor(acct.Name), 0)
	}
	if isHiddenCycleKey(s) {
		a.Error = ""
		a.AccountVisible = a.AccountVisible.next()
		return a.navSetMenu(clampListCursor(0, a.accountListRowCount()))
	}
	if s == "ctrl+t" {
		a.Error = ""
		return a.navPush(routeTagList, 0)
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
		idx := a.accountSelectableIndex(rows, a.Menu)
		return a.navPush(accountPath(rows[idx].Name), 0)
	case "up", "shift+tab":
		rows, err := a.accountListRows()
		if err != nil {
			a.Error = err.Error()
			return a
		}
		if len(rows) > 0 {
			a = a.navSetMenu(nextAccountSelectableIndex(rows, a.Menu, -1))
		}
		return a
	case "down", "tab":
		rows, err := a.accountListRows()
		if err != nil {
			a.Error = err.Error()
			return a
		}
		if len(rows) > 0 {
			a = a.navSetMenu(nextAccountSelectableIndex(rows, a.Menu, 1))
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
		idx := a.accountSelectableIndex(rows, a.Menu)
		a = a.navSetMenu(idx)
		return a.navPush(accountPath(rows[idx].Name), 0)
	default:
		if result, handled := handleFilterableListKey(s, a.listFilter(), a.Menu, a.accountListRowCount()); handled {
			a.setListFilter(result.filter)
			a = a.navSetMenu(result.menu)
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

func (a App) accountSelectableIndex(rows []accountListRow, preferred int) int {
	if len(rows) == 0 {
		return 0
	}
	preferred = clampListCursor(preferred, len(rows))
	if !rows[preferred].Virtual {
		return preferred
	}
	for i := preferred + 1; i < len(rows); i++ {
		if !rows[i].Virtual {
			return i
		}
	}
	for i := preferred - 1; i >= 0; i-- {
		if !rows[i].Virtual {
			return i
		}
	}
	return preferred
}

func nextAccountSelectableIndex(rows []accountListRow, current, delta int) int {
	if len(rows) == 0 {
		return 0
	}
	idx := clampListCursor(current, len(rows))
	for i := 0; i < len(rows); i++ {
		idx = (idx + delta + len(rows)) % len(rows)
		if !rows[idx].Virtual {
			return idx
		}
	}
	return clampListCursor(current, len(rows))
}

func (a App) accountDetailKey(s, name string) App {
	acct, err := a.Svc.Accounts.GetByName(a.ctx, name)
	if err != nil {
		a.Error = err.Error()
		return a
	}
	empty, err := a.Svc.Accounts.IsEmpty(a.ctx, acct.ID)
	if err != nil {
		a.Error = err.Error()
		return a
	}
	count := 5
	if empty {
		count = 6
	}
	action := a.actionIndex(s, count)
	if action < 0 {
		return a
	}
	a = a.navSetMenu(action)
	switch action {
	case 0:
		return a.navPush(accountBalanceListPath(name), 0)
	case 1:
		return a.navPush(accountChildrenListPath(name), 0)
	case 2:
		return a.navPush(accountTransactionsPath(name), 0)
	case 3:
		a.Form = accountFormValues(acct.Name, acct.Code, acct.OnBudget, acct.Notes, a.directTagNames(acct.ID))
		a.Field = 0
		a.ListReturn = listReturnState{}
		return a.navPush(accountEditPathFor(name), 0)
	case 4:
		updated, entry, err := a.Svc.Accounts.SetHidden(a.ctx, acct.ID, !acct.Hidden)
		if err != nil {
			a.Error = err.Error()
			return a
		}
		a.History = append(a.History, entry)
		return a.navReplace(accountPath(updated.Name), action)
	case 5:
		deleted, entry, err := a.Svc.Accounts.DeleteEmpty(a.ctx, acct.ID)
		if err != nil {
			a.Error = err.Error()
			return a
		}
		a.History = append(a.History, entry)
		if deleted.ParentID != nil {
			parent, err := a.Svc.Accounts.GetByID(a.ctx, *deleted.ParentID)
			if err != nil {
				return a.navReset()
			}
			return a.navReplace(accountChildrenListPath(parent.Name), 0)
		}
		return a.navReplace(routeAccountList, 0)
	}
	return a
}

func (a App) accountChildrenListKey(s, parentName string) App {
	parent, err := a.Svc.Accounts.GetByName(a.ctx, parentName)
	if err != nil {
		a.Error = err.Error()
		return a
	}
	rows, err := a.childAccountListRows(parent.ID)
	if err != nil {
		a.Error = err.Error()
		return a
	}
	if isNewKey(s) {
		a.Error = ""
		a.Field = 0
		a.Form = map[string]string{}
		return a.navPush(accountChildCreatePath(parentName), 0)
	}
	if isEditKey(s) {
		if len(rows) == 0 {
			return a
		}
		a = a.navSetMenu(clampListCursor(a.Menu, len(rows)))
		idx := a.accountSelectableIndex(rows, a.Menu)
		acct, err := a.Svc.Accounts.GetByName(a.ctx, rows[idx].Name)
		if err != nil {
			a.Error = err.Error()
			return a
		}
		a = a.captureChildAccountListReturn(parentName, acct.Name)
		a.Form = accountFormValues(acct.Name, acct.Code, acct.OnBudget, acct.Notes, a.directTagNames(acct.ID))
		a.Field = 0
		return a.navPush(accountEditPathFor(acct.Name), 0)
	}
	if isDeleteKey(s) {
		return a
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
		idx := a.accountSelectableIndex(rows, a.Menu)
		return a.navPush(accountPath(rows[idx].Name), 0)
	case "up", "shift+tab":
		if len(rows) > 0 {
			a = a.navSetMenu(nextAccountSelectableIndex(rows, a.Menu, -1))
		}
	case "down", "tab":
		if len(rows) > 0 {
			a = a.navSetMenu(nextAccountSelectableIndex(rows, a.Menu, 1))
		}
	default:
		if result, handled := handleFilterableListKey(s, a.listFilter(), a.Menu, a.childAccountListRowCount(parent.ID)); handled {
			a.setListFilter(result.filter)
			a = a.navSetMenu(result.menu)
		}
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
	var next App
	var submit bool
	if acct.ParentID != nil {
		next, submit = a.childAccountFormKey(s, locked)
	} else {
		next, submit = a.accountFormKey(s, locked)
	}
	if !submit {
		return next
	}
	onBudget := acct.OnBudget
	if acct.ParentID == nil {
		onBudget = parseBoolDefault(next.Form["on-budget"], acct.OnBudget)
	}
	updated, entry, err := next.Svc.Accounts.UpdateWithTags(next.ctx, acct.ID, strings.TrimSpace(next.Form["name"]), strings.TrimSpace(next.Form["currency"]), onBudget, acct.Hidden, next.Form["notes"], splitTagNames(next.Form["tags"]))
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

func (a App) selectChildAccountInList(path, name string) App {
	parentName, ok := accountChildrenListName(path)
	if !ok {
		return a.navReplace(path, 0)
	}
	parent, err := a.Svc.Accounts.GetByName(a.ctx, parentName)
	if err != nil {
		a.Error = err.Error()
		return a
	}
	rows, err := a.childAccountListRows(parent.ID)
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
	return a.navReplace(path, idx)
}

func accountSummaryValues(rows []accountListRow, appCurrency string) []string {
	total := money.Money{Scale: 2}
	if len(rows) > 0 {
		total = money.Money{Scale: rows[0].Amount.Scale}
	}
	onBudgetTotal, hasOnBudget := accountSectionTotal(rows, true)
	offBudgetTotal, hasOffBudget := accountSectionTotal(rows, false)
	if hasOnBudget {
		total, _ = total.Add(onBudgetTotal)
	}
	if hasOffBudget {
		total, _ = total.Add(offBudgetTotal)
	}

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
	return alignedMoneyValues(totalCell, onBudgetCell, offBudgetCell)
}

func (a App) accountList() string {
	return a.accountListBody()
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
				component.TextCell(accountRowDisplayName(row)),
				row.Balance,
				component.TextCell(row.Notes),
				component.TextCell(strings.Join(row.Tags, ", ")),
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
			component.TextCell(accountRowDisplayName(row)),
			row.Balance,
			component.TextCell(row.Notes),
			component.TextCell(strings.Join(row.Tags, ", ")),
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
	headers := []string{"name", "balance", "notes", "tags"}
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
	skipDescendantsOfDepth := -1
	for _, row := range rows {
		if skipDescendantsOfDepth >= 0 && row.Depth > skipDescendantsOfDepth {
			continue
		}
		if skipDescendantsOfDepth >= 0 && row.Depth <= skipDescendantsOfDepth {
			skipDescendantsOfDepth = -1
		}
		if row.OnBudget != onBudget || !row.Match {
			continue
		}
		if row.Virtual {
			continue
		}
		found = true
		skipDescendantsOfDepth = row.Depth
		next, err := total.Add(row.Amount)
		if err == nil {
			total = next
		}
	}
	return total, found
}

func dashboardAccountIDsForRows(rows []accountListRow) []int64 {
	var ids []int64
	skipDescendantsOfDepth := -1
	for _, row := range rows {
		if skipDescendantsOfDepth >= 0 && row.Depth > skipDescendantsOfDepth {
			continue
		}
		if skipDescendantsOfDepth >= 0 && row.Depth <= skipDescendantsOfDepth {
			skipDescendantsOfDepth = -1
		}
		if !row.Match || row.Virtual {
			continue
		}
		ids = append(ids, row.ID)
		skipDescendantsOfDepth = row.Depth
	}
	return ids
}

func (a App) accountListRows() ([]accountListRow, error) {
	return a.accountListRowsWithFilter(a.listFilter())
}

func (a App) accountListRowsWithFilter(filter string) ([]accountListRow, error) {
	accounts, err := a.Svc.Accounts.ListRoots(a.ctx, a.AccountVisible != accountVisibilityNonHidden)
	if err != nil {
		return nil, err
	}
	parsed := parseAccountFilter(filter)
	var visible []accountListRow
	for _, acct := range accounts {
		rows, err := a.accountTreeRows(acct, 0, parsed)
		if err != nil {
			return nil, err
		}
		visible = append(visible, rows...)
	}
	return visible, nil
}

func (a App) accountTreeRows(acct repo.Account, depth int, filter accountFilter) ([]accountListRow, error) {
	if a.AccountVisible == accountVisibilityNonHidden && acct.Hidden {
		return nil, nil
	}
	if a.AccountVisible == accountVisibilityHiddenOnly && !acct.Hidden {
		return nil, nil
	}
	children, err := a.Svc.Accounts.ListChildren(a.ctx, acct.ID, a.AccountVisible != accountVisibilityNonHidden)
	if err != nil {
		return nil, err
	}
	var out []accountListRow
	var childRows []accountListRow
	for _, child := range children {
		rows, err := a.accountTreeRows(child, depth+1, filter)
		if err != nil {
			return nil, err
		}
		childRows = append(childRows, rows...)
	}
	row, err := a.accountRow(acct, depth)
	if err != nil {
		return nil, err
	}
	matches := filter.Empty() || filter.Match(row)
	row.Match = matches
	if matches || len(childRows) > 0 {
		out = append(out, row)
		out = append(out, childRows...)
		if row.Depth == depth && row.Name != "remaining" {
			remaining, err := a.remainingRow(acct, depth+1)
			if err != nil {
				return nil, err
			}
			if remaining != nil {
				remaining.Match = matches
				out = append(out, *remaining)
			}
		}
	}
	return out, nil
}

func (a App) accountRow(acct repo.Account, depth int) (accountListRow, error) {
	summary, err := a.Svc.Accounts.TreeSummary(a.ctx, acct.ID, a.Config.Config.Currency)
	if err != nil {
		return accountListRow{}, err
	}
	appAmount := summary.Balance
	balance := component.MoneyCell(appAmount, a.Config.Config.Currency)
	if acct.Code != a.Config.Config.Currency && summary.HasOwnBalance {
		if bal, ok, err := a.Svc.Accounts.CurrentBalance(a.ctx, acct.ID); err != nil {
			return accountListRow{}, err
		} else if ok {
			balance = component.MoneyCellWithTrailing(appAmount, a.Config.Config.Currency, fmt.Sprintf("(%s)", bal.Amount.Format(acct.Code)))
		}
	}
	asOf := "(no balance entered yet)"
	if summary.AsOf != "" {
		asOf = summary.AsOf
	}
	currencyName := ""
	if cur, err := a.Svc.Currency.Get(a.ctx, acct.Code); err == nil {
		currencyName = cur.Name
	}
	return accountListRow{
		ID:           acct.ID,
		Name:         acct.Name,
		Balance:      balance,
		Amount:       appAmount,
		Notes:        acct.Notes,
		Tags:         a.effectiveTagNames(acct.ID),
		Currency:     acct.Code,
		CurrencyName: currencyName,
		OnBudget:     acct.OnBudget,
		AsOf:         asOf,
		Hidden:       acct.Hidden,
		Depth:        depth,
	}, nil
}

func (a App) remainingRow(acct repo.Account, depth int) (*accountListRow, error) {
	summary, err := a.Svc.Accounts.TreeSummary(a.ctx, acct.ID, a.Config.Config.Currency)
	if err != nil {
		return nil, err
	}
	children, err := a.Svc.Accounts.ListChildren(a.ctx, acct.ID, false)
	if err != nil {
		return nil, err
	}
	if len(children) == 0 {
		return nil, nil
	}
	if !summary.HasOwnBalance || summary.Remaining.Amount == 0 {
		return nil, nil
	}
	return &accountListRow{
		Name:     "remaining",
		Balance:  component.MoneyCell(summary.Remaining, a.Config.Config.Currency),
		Amount:   summary.Remaining,
		OnBudget: acct.OnBudget,
		AsOf:     summary.AsOf,
		Depth:    depth,
		Virtual:  true,
	}, nil
}

func (a App) childAccountListRows(parentID int64) ([]accountListRow, error) {
	return a.childAccountListRowsWithFilter(parentID, a.listFilter())
}

func (a App) childAccountListRowsWithFilter(parentID int64, filter string) ([]accountListRow, error) {
	children, err := a.Svc.Accounts.ListChildren(a.ctx, parentID, false)
	if err != nil {
		return nil, err
	}
	parsed := parseAccountFilter(filter)
	out := make([]accountListRow, 0, len(children))
	for _, child := range children {
		row, err := a.accountRow(child, 0)
		if err != nil {
			return nil, err
		}
		if !parsed.Empty() && !parsed.Match(row) {
			continue
		}
		row.Match = true
		out = append(out, row)
	}
	return out, nil
}

func (a App) childAccountListRowCount(parentID int64) int {
	rows, err := a.childAccountListRows(parentID)
	if err != nil {
		return 0
	}
	return len(rows)
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
			component.TextCell(accountRowDisplayName(row)),
			row.Balance,
			component.TextCell(row.Notes),
			component.TextCell(strings.Join(row.Tags, ", ")),
		}))
	}
	return lines
}

func accountRowDisplayName(row accountListRow) string {
	if row.Depth <= 0 {
		return row.Name
	}
	return strings.Repeat("  ", row.Depth) + row.Name
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
	actions := []string{"balances", "child accounts", "transactions", "edit account", "hide account"}
	if acct.Hidden {
		actions = []string{"balances", "child accounts", "transactions", "edit account", "show account"}
	}
	if empty, err := a.Svc.Accounts.IsEmpty(a.ctx, acct.ID); err == nil && empty {
		actions = append(actions, "delete account")
	}
	return screen{
		Path:    accountPath(name),
		Context: context,
		Actions: actions,
	}
}

func (a App) accountChildrenListScreen(name string) screen {
	acct, err := a.Svc.Accounts.GetByName(a.ctx, name)
	if err != nil {
		return screen{Path: a.Path, Body: "error: " + err.Error() + "\n"}
	}
	context, err := a.accountDashboardContext(name)
	if err != nil {
		return screen{Path: a.Path, Body: "error: " + err.Error() + "\n"}
	}
	context = "parent    : " + acct.Name + "\n" + context
	rows, err := a.childAccountListRows(acct.ID)
	if err != nil {
		return screen{Path: a.Path, Body: "error: " + err.Error() + "\n"}
	}
	var lines []string
	lines = append(lines, "> filter : "+placeholder(a.listFilter(), "(type anything...)"), "")
	if len(rows) == 0 {
		lines = append(lines, "  name | balance | notes")
		if a.listFilter() == "" {
			lines = append(lines, "  (no child accounts yet)")
		} else {
			lines = append(lines, "  (no results)")
		}
	} else {
		layout := accountListTableLayoutFor(rows, a.Config.Config.Currency, false)
		lines = append(lines, layout.Header("  "))
		for i, row := range rows {
			prefix := "  "
			if i == a.Menu {
				prefix = "> "
			}
			lines = append(lines, layout.RowCells(prefix, []component.Cell{
				component.TextCell(row.Name),
				row.Balance,
				component.TextCell(row.Notes),
				component.TextCell(strings.Join(row.Tags, ", ")),
			}))
		}
	}
	return screen{
		Path:    accountChildrenListPath(name),
		Context: context,
		Body:    strings.Join(lines, "\n") + "\n",
		Help:    childAccountListHelp(),
	}
}

func (a App) accountChildCreateScreen(name string) screen {
	acct, err := a.Svc.Accounts.GetByName(a.ctx, name)
	if err != nil {
		return screen{Path: a.Path, Body: "error: " + err.Error() + "\n"}
	}
	context := fmt.Sprintf("parent         : %s\non-budget      : %t\ninherited tags : %s", acct.Name, acct.OnBudget, formatTags(a.effectiveTagNames(acct.ID), nil))
	return screen{Path: accountChildCreatePath(name), Context: context, Body: a.childAccountFormView(nil), Help: a.childAccountFormHelp()}
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
	if acct.ParentID != nil {
		body := a.childAccountFormView(locked) + "\ninherited : " + formatTags(a.inheritedTagNames(acct.ID), nil) + "\n"
		return screen{Path: a.Path, Body: body, Help: a.childAccountFormHelp()}
	}
	return screen{Path: a.Path, Body: a.accountFormView(locked), Help: a.accountFormHelp()}
}

func (a App) accountFormView(locked map[string]string) string {
	return a.formViewWithOptions([]string{"name", "currency", "on-budget", "notes", "tags"}, locked, map[string][]string{
		"currency":  a.currencyOptions(),
		"on-budget": {"true", "false"},
	}, nil)
}

func (a App) childAccountFormView(locked map[string]string) string {
	return a.formViewWithOptions([]string{"name", "currency", "notes", "tags"}, locked, map[string][]string{
		"currency": a.currencyOptions(),
	}, nil)
}

func (a App) childAccountFormHelp() []string {
	return []string{"type    : enter text", "tab     : navigate", "enter   : confirm", "esc     : back", "?       : help"}
}
