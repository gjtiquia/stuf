package model

import (
	"fmt"
	"strings"

	"stuf/internal/component"
	"stuf/internal/money"
	"stuf/internal/repo"
)

type budgetListRow struct {
	Budget   repo.Budget
	Balance  money.Money
	Display  component.Cell
	Category string
}

func (a App) budgetListKey(s string) App {
	if isNewKey(s) {
		a.Error = ""
		a.Form = map[string]string{"currency": a.Config.Config.Currency, "category": "uncategorized"}
		a.Field = 0
		return a.navPush(routeBudgetCreate, 0)
	}
	if isHiddenCycleKey(s) {
		a.Error = ""
		a.BudgetVisible = a.BudgetVisible.next()
		return a.navSetMenu(0)
	}
	if s == "ctrl+t" {
		a.Error = ""
		return a.navPush(routeBudgetCatList, 0)
	}
	rows, err := a.filteredBudgetRows()
	if err != nil {
		a.Error = err.Error()
		return a
	}
	if isEditKey(s) && len(rows) > 0 {
		a = a.navSetMenu(clampListCursor(a.Menu, len(rows)))
		b := rows[a.Menu].Budget
		a.Form = budgetFormValues(b)
		a.Field = 0
		return a.navPush(budgetEditPathFor(b.Name), 0)
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
		return a.navPush(budgetPath(rows[a.Menu].Budget.Name), 0)
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
			nextRows, _ := a.filteredBudgetRows()
			a = a.navSetMenu(clampListCursor(result.menu, len(nextRows)))
		}
	}
	return a
}

func (a App) budgetCreateKey(s string) App {
	if a.Form["currency"] == "" {
		a.Form["currency"] = a.Config.Config.Currency
	}
	if a.Form["category"] == "" {
		a.Form["category"] = "uncategorized"
	}
	next, submit := a.budgetFormKey(s)
	if !submit {
		return next
	}
	b, entry, err := next.Svc.Budgets.Create(next.ctx, strings.TrimSpace(next.Form["name"]), strings.TrimSpace(next.Form["currency"]), strings.TrimSpace(next.Form["category"]), next.Form["notes"])
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
	return next.selectBudgetInList(b.Name)
}

func (a App) budgetEditKey(s, name string) App {
	b, err := a.Svc.Budgets.GetByName(a.ctx, name)
	if err != nil {
		a.Error = err.Error()
		return a
	}
	next, submit := a.budgetFormKey(s)
	if !submit {
		return next
	}
	updated, entry, err := next.Svc.Budgets.Update(next.ctx, b.ID, strings.TrimSpace(next.Form["name"]), strings.TrimSpace(next.Form["currency"]), strings.TrimSpace(next.Form["category"]), b.Hidden, next.Form["notes"])
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
	if next.Path == routeBudgetList {
		return next.selectBudgetInList(updated.Name)
	}
	return next.navReplace(budgetPath(updated.Name), 0)
}

func (a App) budgetDetailKey(s, name string) App {
	b, err := a.Svc.Budgets.GetByName(a.ctx, name)
	if err != nil {
		a.Error = err.Error()
		return a
	}
	action := a.actionIndex(s, 3)
	if action < 0 {
		return a
	}
	a = a.navSetMenu(action)
	switch action {
	case 0:
		return a.navPush(budgetAllocationListPath(name), 0)
	case 1:
		a.Form = budgetFormValues(b)
		a.Field = 0
		return a.navPush(budgetEditPathFor(name), 0)
	case 2:
		updated, entry, err := a.Svc.Budgets.SetHidden(a.ctx, b.ID, !b.Hidden)
		if err != nil {
			a.Error = err.Error()
			return a
		}
		a.History = append(a.History, entry)
		a.Error = ""
		return a.navReplace(budgetPath(updated.Name), 0)
	}
	return a
}

func (a App) budgetFormKey(s string) (App, bool) {
	fields := []string{"name", "currency", "category", "notes"}
	if isSubmitKey(s) {
		a.clearCurrentTextCursor(fields)
		return a, true
	}
	if a.Field == 1 {
		return a.currencyFieldKey(s, fields)
	}
	return a.submitFormKey(s, fields)
}

func budgetFormValues(b repo.Budget) map[string]string {
	return map[string]string{"name": b.Name, "currency": b.Code, "category": b.CategoryName, "notes": b.Notes}
}

func (a App) selectBudgetInList(name string) App {
	rows, err := a.filteredBudgetRows()
	if err != nil {
		a.Error = err.Error()
		return a
	}
	idx := 0
	for i, row := range rows {
		if row.Budget.Name == name {
			idx = i
			break
		}
	}
	return a.navReplace(routeBudgetList, idx)
}

func (a App) budgetListRowCount() int {
	rows, err := a.filteredBudgetRows()
	if err != nil {
		return 0
	}
	return len(rows)
}

func (a App) filteredBudgetRows() ([]budgetListRow, error) {
	budgets, err := a.Svc.Budgets.List(a.ctx, a.BudgetVisible == accountVisibilityAll || a.BudgetVisible == accountVisibilityHiddenOnly)
	if err != nil {
		return nil, err
	}
	filter := strings.ToLower(a.listFilter())
	var out []budgetListRow
	for _, b := range budgets {
		if a.BudgetVisible == accountVisibilityHiddenOnly && !b.Hidden {
			continue
		}
		if a.BudgetVisible == accountVisibilityNonHidden && b.Hidden {
			continue
		}
		if filter != "" && !strings.Contains(strings.ToLower(b.Name), filter) && !strings.Contains(strings.ToLower(b.CategoryName), filter) && !strings.Contains(strings.ToLower(b.Notes), filter) {
			continue
		}
		bal, err := a.Svc.BudgetAllocations.Balance(a.ctx, b.ID)
		if err != nil {
			return nil, err
		}
		cell, err := a.budgetMoneyCell(b, bal)
		if err != nil {
			return nil, err
		}
		out = append(out, budgetListRow{Budget: b, Balance: bal, Display: cell, Category: b.CategoryName})
	}
	return out, nil
}

func (a App) budgetMoneyCell(b repo.Budget, amount money.Money) (component.Cell, error) {
	appCode := a.Config.Config.Currency
	if b.Code == appCode {
		return component.MoneyCell(amount, b.Code), nil
	}
	from, err := a.Svc.Currency.Get(a.ctx, b.Code)
	if err != nil {
		return component.MoneyCell(amount, b.Code), nil
	}
	to, err := a.Svc.Currency.Get(a.ctx, appCode)
	if err != nil {
		return component.MoneyCell(amount, b.Code), nil
	}
	converted, err := money.Convert(amount, from.RateToUSD, to.RateToUSD, to.Scale)
	if err != nil {
		return component.MoneyCell(amount, b.Code), nil
	}
	return component.MoneyCellWithTrailing(converted, appCode, "("+amount.Format(b.Code)+")"), nil
}

func (a App) budgetListDashboardContext() (string, error) {
	d, err := a.Svc.Dashboard.Summary(a.ctx)
	if err != nil {
		return "", err
	}
	cur := a.Config.Config.Currency
	values := alignedMoneyValues(component.MoneyCell(d.Total, cur), component.MoneyCell(d.Budgeted, cur), component.MoneyCell(d.Available, cur))
	body := fmt.Sprintf("on-budget : %s\nbudgeted  : %s\navailable : %s", values[0], values[1], values[2])
	if warnings := dashboardWarnings(d.Warnings); warnings != "" {
		body += "\n" + warnings
	}
	return strings.TrimRight(body, "\n"), nil
}

func (a App) budgetListScreen() screen {
	context, err := a.budgetListDashboardContext()
	if err != nil {
		return screen{Path: routeBudgetList, Body: "error: " + err.Error() + "\n"}
	}
	rows, err := a.filteredBudgetRows()
	if err != nil {
		return screen{Path: routeBudgetList, Context: context, Body: "error: " + err.Error() + "\n"}
	}
	lines := []string{fmt.Sprintf("showing : %s", a.BudgetVisible.label()), "", "> filter : " + placeholder(a.listFilter(), "(type anything...)"), ""}
	if len(rows) == 0 {
		lines = append(lines, "  name | balance | notes")
		if a.listFilter() == "" {
			lines = append(lines, "  (no budgets yet)")
		} else {
			lines = append(lines, "  (no results)")
		}
		return screen{Path: routeBudgetList, Context: context, Body: strings.Join(lines, "\n") + "\n", Help: budgetListHelp()}
	}
	tableRows := make([][]component.Cell, len(rows))
	for i, row := range rows {
		tableRows[i] = []component.Cell{component.TextCell(row.Budget.Name), row.Display, component.TextCell(row.Budget.Notes)}
	}
	layout := component.NewTableLayoutCells([]string{"name", "balance", "notes"}, tableRows)
	lastCategory := ""
	for i, row := range rows {
		if row.Category != lastCategory {
			if lastCategory != "" {
				lines = append(lines, "")
			}
			lines = append(lines, "  "+row.Category)
			lines = append(lines, layout.Header("  "))
			lastCategory = row.Category
		}
		prefix := "  "
		if i == a.Menu {
			prefix = "> "
		}
		lines = append(lines, layout.RowCells(prefix, tableRows[i]))
	}
	return screen{Path: routeBudgetList, Context: context, Body: strings.Join(lines, "\n") + "\n", Help: budgetListHelp()}
}

func (a App) budgetCreateScreen() screen {
	return screen{Path: routeBudgetCreate, Body: a.budgetFormView(), Help: a.formHelp([]string{"name", "currency", "category", "notes"})}
}

func (a App) budgetEditScreen(name string) screen {
	if a.Form["name"] == "" {
		if b, err := a.Svc.Budgets.GetByName(a.ctx, name); err == nil {
			a.Form = budgetFormValues(b)
		}
	}
	return screen{Path: budgetEditPathFor(name), Body: a.budgetFormView(), Help: a.formHelp([]string{"name", "currency", "category", "notes"})}
}

func (a App) budgetFormView() string {
	fields := []string{"name", "currency", "category", "notes"}
	options := map[string][]string{"currency": a.currencyOptions()}
	return a.formViewWithOptions(fields, nil, options, nil)
}

func (a App) budgetDetailScreen(name string) screen {
	b, err := a.Svc.Budgets.GetByName(a.ctx, name)
	if err != nil {
		return screen{Path: budgetPath(name), Body: "error: " + err.Error() + "\n"}
	}
	bal, err := a.Svc.BudgetAllocations.Balance(a.ctx, b.ID)
	if err != nil {
		return screen{Path: budgetPath(name), Body: "error: " + err.Error() + "\n"}
	}
	cell, _ := a.budgetMoneyCell(b, bal)
	value := component.NewMoneyColumn(cell).Render(cell)
	lines := []string{
		fmt.Sprintf("name      : %s", b.Name),
		fmt.Sprintf("category  : %s", b.CategoryName),
		fmt.Sprintf("allocated : %s", value),
		fmt.Sprintf("spent     : %s", zero(a.Config.Config.Currency)),
		fmt.Sprintf("balance   : %s", value),
	}
	if b.Hidden {
		lines = append(lines, "hidden    : true")
	}
	lines = append(lines, fmt.Sprintf("notes     : %s", b.Notes))
	action := "hide budget"
	if b.Hidden {
		action = "show budget"
	}
	return screen{Path: budgetPath(name), Body: strings.Join(lines, "\n") + "\n", Actions: []string{"allocations", "edit budget", action}}
}

func budgetListHelp() []string {
	return []string{"type          : filter", "h/l           : type in filter", "up/down       : navigate", "left/right    : back/open", "enter         : confirm", "ctrl+n        : new", "ctrl+e        : edit", "ctrl+h        : cycle hidden visibility", "ctrl+t        : categories", "esc           : back", "?             : help", "ctrl-z        : undo"}
}
