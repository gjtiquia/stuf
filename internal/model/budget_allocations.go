package model

import (
	"fmt"
	"strings"

	"stuf/internal/component"
	"stuf/internal/service"
)

func (a App) budgetAllocationListKey(s, name string) App {
	if isNewKey(s) {
		a.Error = ""
		a.Form = map[string]string{"action": service.AllocationActionSetTotal, "date": Today()}
		a.Field = 0
		return a.navPush(budgetAllocationAddPath(name), 0)
	}
	rows, err := a.budgetAllocationRows(name)
	if err != nil {
		a.Error = err.Error()
		return a
	}
	if (isEditKey(s) || isDeleteKey(s)) && len(rows) > 0 {
		a = a.navSetMenu(clampListCursor(a.Menu, len(rows)))
		row := rows[a.Menu]
		if isEditKey(s) {
			a.Form = map[string]string{"date": row.Allocation.Date, "amount": rawAmount(row.Allocation.Amount.Amount, row.Allocation.Amount.Scale), "notes": row.Allocation.Notes}
			a.Field = 0
			return a.navPush(budgetAllocationEditPath(name, row.Allocation.ID), 0)
		}
		entry, err := a.Svc.BudgetAllocations.Delete(a.ctx, row.Allocation.ID)
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

func (a App) budgetAllocationAddKey(s, name string) App {
	b, err := a.Svc.Budgets.GetByName(a.ctx, name)
	if err != nil {
		a.Error = err.Error()
		return a
	}
	if a.Form["action"] == "" {
		a.Form["action"] = service.AllocationActionSetTotal
	}
	fields := []string{"action", "amount", "date", "notes"}
	next, submit := a.allocationFormKey(s, fields)
	if !submit {
		return next
	}
	_, entry, err := next.Svc.BudgetAllocations.Add(next.ctx, b.ID, next.Form["action"], next.Form["amount"], next.Form["date"], next.Form["notes"])
	if err != nil {
		next.Error = err.Error()
		return next
	}
	next.History = append(next.History, entry)
	next.Form = map[string]string{}
	next.Field = 0
	next.Error = ""
	next.Nav.Pop()
	return next.navReplace(budgetAllocationListPath(name), 0)
}

func (a App) budgetAllocationEditKey(s, name string, id int64) App {
	b, err := a.Svc.Budgets.GetByName(a.ctx, name)
	if err != nil {
		a.Error = err.Error()
		return a
	}
	_ = b
	fields := []string{"amount", "date", "notes"}
	next, submit := a.submitFormKey(s, fields)
	if !submit {
		return next
	}
	updated, entry, err := next.Svc.BudgetAllocations.Update(next.ctx, id, next.Form["amount"], next.Form["date"], next.Form["notes"])
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
	if next.Path == budgetAllocationListPath(name) {
		rows, _ := next.budgetAllocationRows(name)
		for i, row := range rows {
			if row.Allocation.ID == updated.ID {
				return next.navReplace(budgetAllocationListPath(name), i)
			}
		}
	}
	return next
}

func (a App) allocationFormKey(s string, fields []string) (App, bool) {
	if isSubmitKey(s) {
		a.clearCurrentTextCursor(fields)
		return a, true
	}
	if a.Field == 0 {
		return a.selectFieldKey(s, "action", []string{service.AllocationActionSetTotal, service.AllocationActionAddMoney, service.AllocationActionRemoveMoney}, fields)
	}
	return a.submitFormKey(s, fields)
}

func (a App) budgetAllocationRows(name string) ([]service.BudgetAllocationRow, error) {
	b, err := a.Svc.Budgets.GetByName(a.ctx, name)
	if err != nil {
		return nil, err
	}
	return a.Svc.BudgetAllocations.ListWithBalances(a.ctx, b.ID)
}

func (a App) budgetAllocationListScreen(name string) screen {
	b, err := a.Svc.Budgets.GetByName(a.ctx, name)
	if err != nil {
		return screen{Path: budgetAllocationListPath(name), Body: "error: " + err.Error() + "\n"}
	}
	rows, err := a.Svc.BudgetAllocations.ListWithBalances(a.ctx, b.ID)
	if err != nil {
		return screen{Path: budgetAllocationListPath(name), Body: "error: " + err.Error() + "\n"}
	}
	if len(rows) == 0 {
		return screen{Path: budgetAllocationListPath(name), Body: "  date | change | balance | notes\n  (no allocations yet)\n", Help: budgetAllocationListHelp()}
	}
	tableRows := make([][]component.Cell, len(rows))
	for i, row := range rows {
		tableRows[i] = []component.Cell{
			component.TextCell(row.Allocation.Date),
			component.MoneyCell(row.Allocation.Amount, b.Code),
			component.MoneyCell(row.Balance, b.Code),
			component.TextCell(row.Allocation.Notes),
		}
	}
	layout := component.NewTableLayoutCells([]string{"date", "change", "balance", "notes"}, tableRows)
	lines := []string{layout.Header("  ")}
	for i, row := range tableRows {
		prefix := "  "
		if i == a.Menu {
			prefix = "> "
		}
		lines = append(lines, layout.RowCells(prefix, row))
	}
	return screen{Path: budgetAllocationListPath(name), Body: strings.Join(lines, "\n") + "\n", Help: budgetAllocationListHelp()}
}

func (a App) budgetAllocationAddScreen(name string) screen {
	b, err := a.Svc.Budgets.GetByName(a.ctx, name)
	if err != nil {
		return screen{Path: budgetAllocationAddPath(name), Body: "error: " + err.Error() + "\n"}
	}
	current, err := a.Svc.BudgetAllocations.Balance(a.ctx, b.ID)
	if err != nil {
		return screen{Path: budgetAllocationAddPath(name), Body: "error: " + err.Error() + "\n"}
	}
	context := fmt.Sprintf("current : %s", current.Format(b.Code))
	fields := []string{"action", "amount", "date", "notes"}
	options := map[string][]string{"action": {service.AllocationActionSetTotal, service.AllocationActionAddMoney, service.AllocationActionRemoveMoney}}
	prefixes := map[string]string{"amount": b.Code}
	return screen{Path: budgetAllocationAddPath(name), Context: context, Options: a.formViewWithOptions(fields, nil, options, prefixes), Help: a.formHelp(fields)}
}

func (a App) budgetAllocationEditScreen(name string, id int64) screen {
	b, err := a.Svc.Budgets.GetByName(a.ctx, name)
	if err != nil {
		return screen{Path: budgetAllocationEditPath(name, id), Body: "error: " + err.Error() + "\n"}
	}
	if a.Form["date"] == "" {
		rows, _ := a.Svc.BudgetAllocations.List(a.ctx, b.ID)
		for _, row := range rows {
			if row.ID == id {
				a.Form = map[string]string{"amount": rawAmount(row.Amount.Amount, row.Amount.Scale), "date": row.Date, "notes": row.Notes}
				break
			}
		}
	}
	fields := []string{"amount", "date", "notes"}
	prefixes := map[string]string{"amount": b.Code}
	return screen{Path: budgetAllocationEditPath(name, id), Options: a.formViewWithOptions(fields, nil, nil, prefixes), Help: a.formHelp(fields)}
}

func budgetAllocationListHelp() []string {
	return []string{"up/down : navigate", "ctrl+n  : allocate", "ctrl+e  : edit", "ctrl+d  : delete", "esc     : back", "?       : help", "ctrl-z  : undo"}
}
