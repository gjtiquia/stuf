package model

import (
	"strings"

	"stuf/internal/component"
)

func (a App) tagListKey(s string) App {
	if isNewKey(s) {
		a.Error = ""
		a.Form = map[string]string{}
		a.Field = 0
		return a.navPush(routeTagCreate, 0)
	}
	rows, err := a.filteredTags()
	if err != nil {
		a.Error = err.Error()
		return a
	}
	if (isEditKey(s) || s == "enter" || s == "right") && len(rows) > 0 {
		a = a.navSetMenu(clampListCursor(a.Menu, len(rows)))
		tag := rows[a.Menu]
		a.Form = map[string]string{"tag-name": tag.Name, "notes": tag.Notes}
		a.Field = 0
		return a.navPush(tagEditPathFor(tag.Name), 0)
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
	default:
		if result, handled := handleFilterableListKey(s, a.listFilter(), a.Menu, len(rows)); handled {
			a.setListFilter(result.filter)
			nextRows, _ := a.filteredTags()
			a = a.navSetMenu(clampListCursor(result.menu, len(nextRows)))
		}
	}
	return a
}

func (a App) tagCreateKey(s string) App {
	next, submit := a.submitFormKey(s, []string{"tag-name", "notes"})
	if !submit {
		return next
	}
	tag, entry, err := next.Svc.Tags.Create(next.ctx, strings.TrimSpace(next.Form["tag-name"]), next.Form["notes"])
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
	return next.selectTagInList(tag.Name)
}

func (a App) tagEditKey(s, name string) App {
	tag, err := a.Svc.Tags.GetByName(a.ctx, name)
	if err != nil {
		a.Error = err.Error()
		return a
	}
	next, submit := a.submitFormKey(s, []string{"tag-name", "notes"})
	if !submit {
		return next
	}
	updated, entry, err := next.Svc.Tags.Update(next.ctx, tag.ID, strings.TrimSpace(next.Form["tag-name"]), next.Form["notes"])
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
	return next.selectTagInList(updated.Name)
}

func (a App) selectTagInList(name string) App {
	rows, err := a.filteredTags()
	if err != nil {
		a.Error = err.Error()
		return a
	}
	idx := 0
	for i, tag := range rows {
		if tag.Name == name {
			idx = i
			break
		}
	}
	return a.navReplace(routeTagList, idx)
}

func (a App) filteredTags() ([]tagListRow, error) {
	tags, err := a.Svc.Tags.List(a.ctx)
	if err != nil {
		return nil, err
	}
	filter := strings.ToLower(a.listFilter())
	var out []tagListRow
	for _, tag := range tags {
		if filter != "" && !strings.Contains(strings.ToLower(tag.Name), filter) && !strings.Contains(strings.ToLower(tag.Notes), filter) {
			continue
		}
		out = append(out, tagListRow{Name: tag.Name, Notes: tag.Notes})
	}
	return out, nil
}

type tagListRow struct {
	Name  string
	Notes string
}

func (a App) tagListScreen() screen {
	rows, err := a.filteredTags()
	if err != nil {
		return screen{Path: routeTagList, Body: "error: " + err.Error() + "\n"}
	}
	lines := []string{"> filter : " + placeholder(a.listFilter(), "(type anything...)"), ""}
	if len(rows) == 0 {
		lines = append(lines, "  name | notes")
		if a.listFilter() == "" {
			lines = append(lines, "  (no tags yet)")
		} else {
			lines = append(lines, "  (no results)")
		}
	} else {
		tableRows := make([][]component.Cell, len(rows))
		for i, row := range rows {
			tableRows[i] = []component.Cell{component.TextCell(row.Name), component.TextCell(row.Notes)}
		}
		layout := component.NewTableLayoutCells([]string{"name", "notes"}, tableRows)
		lines = append(lines, layout.Header("  "))
		for i, row := range rows {
			prefix := "  "
			if i == a.Menu {
				prefix = "> "
			}
			lines = append(lines, layout.RowCells(prefix, []component.Cell{component.TextCell(row.Name), component.TextCell(row.Notes)}))
		}
	}
	return screen{Path: routeTagList, Body: strings.Join(lines, "\n") + "\n", Help: tagListHelp()}
}

func (a App) tagCreateScreen() screen {
	return screen{Path: routeTagCreate, Body: a.tagFormView(), Help: tagFormHelp()}
}

func (a App) tagEditScreen(name string) screen {
	if a.Form["tag-name"] == "" && a.Form["notes"] == "" {
		if tag, err := a.Svc.Tags.GetByName(a.ctx, name); err == nil {
			a.Form = map[string]string{"tag-name": tag.Name, "notes": tag.Notes}
		}
	}
	return screen{Path: tagEditPathFor(name), Body: a.tagFormView(), Help: tagFormHelp()}
}

func (a App) tagFormView() string {
	return a.formView([]string{"tag-name", "notes"}, nil)
}

func tagListHelp() []string {
	return []string{"type          : filter", "h/l           : type in filter", "up/down       : navigate", "left/right    : back/open", "enter         : confirm", "ctrl+n        : new", "ctrl+e        : edit", "esc           : back", "?             : help", "ctrl-z        : undo"}
}

func tagFormHelp() []string {
	return []string{"type    : enter text", "tab     : navigate", "enter   : confirm", "ctrl+s  : submit", "esc     : back", "?       : help"}
}
