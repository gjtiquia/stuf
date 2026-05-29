package model

import (
	"fmt"
	"strings"

	"stuf/internal/component"
	"stuf/internal/service"
)

type budgetCategoryListRow struct {
	Name    string
	Budgets int
	Notes   string
}

func (a App) budgetCategoryListKey(s string) App {
	if isNewKey(s) {
		a.Error = ""
		a.Form = map[string]string{}
		a.Field = 0
		return a.navPush(routeBudgetCatCreate, 0)
	}
	rows, err := a.filteredBudgetCategories()
	if err != nil {
		a.Error = err.Error()
		return a
	}
	if isEditKey(s) && len(rows) > 0 {
		a = a.navSetMenu(clampListCursor(a.Menu, len(rows)))
		row := rows[a.Menu]
		a.Form = map[string]string{"name": row.Name, "notes": row.Notes}
		a.Field = 0
		return a.navPush(budgetCategoryEditPathFor(row.Name), 0)
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
		return a.navPush(budgetCategoryPath(rows[a.Menu].Name), 0)
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
			nextRows, _ := a.filteredBudgetCategories()
			a = a.navSetMenu(clampListCursor(result.menu, len(nextRows)))
		}
	}
	return a
}

func (a App) budgetCategoryCreateKey(s string) App {
	next, submit := a.submitFormKey(s, []string{"name", "notes"})
	if !submit {
		return next
	}
	cat, entry, err := next.Svc.BudgetCategories.Create(next.ctx, strings.TrimSpace(next.Form["name"]), next.Form["notes"])
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
	return next.selectBudgetCategoryInList(cat.Name)
}

func (a App) budgetCategoryEditKey(s, name string) App {
	cat, err := a.Svc.BudgetCategories.GetByName(a.ctx, name)
	if err != nil {
		a.Error = err.Error()
		return a
	}
	next, submit := a.submitFormKey(s, []string{"name", "notes"})
	if !submit {
		return next
	}
	updated, entry, err := next.Svc.BudgetCategories.Update(next.ctx, cat.ID, strings.TrimSpace(next.Form["name"]), next.Form["notes"])
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
	return next.navReplace(budgetCategoryPath(updated.Name), 0)
}

func (a App) budgetCategoryDetailKey(s, name string) App {
	action := a.actionIndex(s, 3)
	if action < 0 {
		return a
	}
	switch action {
	case 0:
		a.Form["category"] = name
		return a.navPush(routeBudgetList, 0)
	case 1:
		a.Form = map[string]string{"currency": a.Config.Config.Currency, "category": name}
		a.Field = 0
		return a.navPush(budgetCategoryBudgetCreatePath(name), 0)
	case 2:
		if name == service.DefaultBudgetCategoryName {
			a.Error = "uncategorized cannot be edited"
			return a
		}
		if cat, err := a.Svc.BudgetCategories.GetByName(a.ctx, name); err == nil {
			a.Form = map[string]string{"name": cat.Name, "notes": cat.Notes}
		}
		a.Field = 0
		return a.navPush(budgetCategoryEditPathFor(name), 0)
	}
	return a
}

func (a App) budgetCategoryBudgetCreateKey(s, name string) App {
	if a.Form["category"] == "" {
		a.Form["category"] = name
	}
	return a.budgetCreateKey(s)
}

func (a App) filteredBudgetCategories() ([]budgetCategoryListRow, error) {
	cats, err := a.Svc.BudgetCategories.List(a.ctx)
	if err != nil {
		return nil, err
	}
	filter := strings.ToLower(a.listFilter())
	var out []budgetCategoryListRow
	for _, cat := range cats {
		budgets, err := a.Svc.Budgets.ListByCategory(a.ctx, cat.ID)
		if err != nil {
			return nil, err
		}
		if cat.Name == service.DefaultBudgetCategoryName && len(budgets) == 0 {
			continue
		}
		if filter != "" && !strings.Contains(strings.ToLower(cat.Name), filter) && !strings.Contains(strings.ToLower(cat.Notes), filter) {
			continue
		}
		out = append(out, budgetCategoryListRow{Name: cat.Name, Budgets: len(budgets), Notes: cat.Notes})
	}
	return out, nil
}

func (a App) selectBudgetCategoryInList(name string) App {
	rows, err := a.filteredBudgetCategories()
	if err != nil {
		a.Error = err.Error()
		return a
	}
	idx := 0
	for i, row := range rows {
		if row.Name == name {
			idx = i
			break
		}
	}
	return a.navReplace(routeBudgetCatList, idx)
}

func (a App) budgetCategoryListScreen() screen {
	rows, err := a.filteredBudgetCategories()
	if err != nil {
		return screen{Path: routeBudgetCatList, Body: "error: " + err.Error() + "\n"}
	}
	lines := []string{"> filter : " + placeholder(a.listFilter(), "(type anything...)"), ""}
	if len(rows) == 0 {
		lines = append(lines, "  name | budgets | notes", "  (no categories yet)")
	} else {
		tableRows := make([][]component.Cell, len(rows))
		for i, row := range rows {
			tableRows[i] = []component.Cell{component.TextCell(row.Name), component.TextCell(fmt.Sprintf("%d", row.Budgets)), component.TextCell(row.Notes)}
		}
		layout := component.NewTableLayoutCells([]string{"name", "budgets", "notes"}, tableRows)
		lines = append(lines, layout.Header("  "))
		for i, row := range rows {
			prefix := "  "
			if i == a.Menu {
				prefix = "> "
			}
			lines = append(lines, layout.RowCells(prefix, []component.Cell{component.TextCell(row.Name), component.TextCell(fmt.Sprintf("%d", row.Budgets)), component.TextCell(row.Notes)}))
		}
	}
	return screen{Path: routeBudgetCatList, Body: strings.Join(lines, "\n") + "\n", Help: budgetCategoryListHelp()}
}

func (a App) budgetCategoryCreateScreen() screen {
	return screen{Path: routeBudgetCatCreate, Body: a.formView([]string{"name", "notes"}, nil), Help: a.formHelp([]string{"name", "notes"})}
}

func (a App) budgetCategoryEditScreen(name string) screen {
	if a.Form["name"] == "" {
		if cat, err := a.Svc.BudgetCategories.GetByName(a.ctx, name); err == nil {
			a.Form = map[string]string{"name": cat.Name, "notes": cat.Notes}
		}
	}
	return screen{Path: budgetCategoryEditPathFor(name), Body: a.formView([]string{"name", "notes"}, nil), Help: a.formHelp([]string{"name", "notes"})}
}

func (a App) budgetCategoryDetailScreen(name string) screen {
	cat, err := a.Svc.BudgetCategories.GetByName(a.ctx, name)
	if err != nil {
		return screen{Path: budgetCategoryPath(name), Body: "error: " + err.Error() + "\n"}
	}
	budgets, err := a.Svc.Budgets.ListByCategory(a.ctx, cat.ID)
	if err != nil {
		return screen{Path: budgetCategoryPath(name), Body: "error: " + err.Error() + "\n"}
	}
	return screen{
		Path:    budgetCategoryPath(name),
		Body:    fmt.Sprintf("name    : %s\nbudgets : %d\nnotes   : %s\n", cat.Name, len(budgets), cat.Notes),
		Actions: []string{"budgets", "create budget in category", "edit category"},
	}
}

func (a App) budgetCategoryBudgetCreateScreen(name string) screen {
	if a.Form["category"] == "" {
		a.Form["category"] = name
	}
	return screen{Path: budgetCategoryBudgetCreatePath(name), Body: a.budgetFormView(), Help: a.formHelp([]string{"name", "currency", "category", "notes"})}
}

func budgetCategoryListHelp() []string {
	return []string{"type          : filter", "h/l           : type in filter", "up/down       : navigate", "left/right    : back/open", "enter         : confirm", "ctrl+n        : new", "ctrl+e        : edit", "esc           : back", "?             : help", "ctrl-z        : undo"}
}
