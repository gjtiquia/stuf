package model

import (
	"fmt"
	"sort"
	"strings"
)

func (a App) accountFormKey(s string, locked map[string]bool) (App, bool) {
	fields := []string{"name", "currency", "on-budget", "notes", "tags"}
	if isSubmitKey(s) {
		a.clearCurrentTextCursor(fields)
		return a, true
	}
	if a.Field == 1 && locked != nil && locked["currency"] {
		switch s {
		case "enter", "tab", "down":
			a.Field = 2
		case "shift+tab", "up":
			a.Field = 0
		}
		return a, false
	}
	if a.Field == 1 {
		return a.currencyFieldKey(s, fields)
	}
	if a.Field == 2 {
		return a.selectFieldKey(s, "on-budget", []string{"true", "false"}, fields)
	}
	if a.Field == 4 {
		return a.tagFieldKey(s, fields)
	}
	return a.submitFormKey(s, fields)
}

func (a App) childAccountFormKey(s string, locked map[string]bool) (App, bool) {
	fields := []string{"name", "currency", "notes", "tags"}
	if isSubmitKey(s) {
		a.clearCurrentTextCursor(fields)
		return a, true
	}
	if a.Field == 1 && locked != nil && locked["currency"] {
		switch s {
		case "enter", "tab", "down":
			a.Field = 2
		case "shift+tab", "up":
			a.Field = 0
		}
		return a, false
	}
	if a.Field == 1 {
		return a.currencyFieldKey(s, fields)
	}
	if a.Field == 3 {
		return a.tagFieldKey(s, fields)
	}
	return a.submitFormKey(s, fields)
}

func (a App) currencyFieldKey(s string, fields []string) (App, bool) {
	options := a.currencyOptions()
	if a.Form["currency"] == "" {
		a.Form["currency"] = a.Config.Config.Currency
	}
	filtered := filterOptions(options, a.currencyFilter())
	cursor := clampCurrencyCursor(parseFormInt(a.Form[currencyCursorKey]), len(filtered))
	a.setCurrencySelectCursor(cursor)
	switch s {
	case "down":
		if len(filtered) == 0 {
			return a, false
		}
		cursor = (cursor + 1) % len(filtered)
		a.setCurrencySelectCursor(cursor)
	case "up":
		if len(filtered) == 0 {
			return a, false
		}
		cursor = (cursor - 1 + len(filtered)) % len(filtered)
		a.setCurrencySelectCursor(cursor)
	case "right":
		if len(filtered) == 0 {
			return a, false
		}
		page := min(a.currencySelectPage()+1, currencyPageCount(len(filtered))-1)
		cursor = min(page*currencyPageSize, len(filtered)-1)
		a.setCurrencySelectCursor(cursor)
		a.setCurrencySelectPage(page)
	case "left":
		if len(filtered) == 0 {
			return a, false
		}
		page := max(a.currencySelectPage()-1, 0)
		cursor = min(page*currencyPageSize, len(filtered)-1)
		a.setCurrencySelectCursor(cursor)
		a.setCurrencySelectPage(page)
	case "backspace":
		input := newFilteredListInput(a.currencyFilter(), sanitizeCurrencyCode)
		updated, _ := input.handleKey("backspace")
		a.setCurrencyFilter(updated.value())
		a.resetCurrencySelectCursor()
	case "tab":
		a.clearCurrencySelectState()
		a.Field = min(a.Field+1, len(fields))
	case "enter":
		if len(filtered) == 0 {
			return a, false
		}
		a.Form["currency"] = filtered[cursor]
		a.clearCurrencySelectState()
		a.Field = min(a.Field+1, len(fields))
	case "shift+tab":
		a.clearCurrencySelectState()
		a.Field = max(a.Field-1, 0)
	default:
		if strings.HasPrefix(s, "set currency=") {
			a.Form["currency"] = sanitizeCurrencyCode(strings.TrimPrefix(s, "set currency="))
			a.clearCurrencySelectState()
			return a, false
		}
		input := newFilteredListInput(a.currencyFilter(), sanitizeCurrencyCode)
		if updated, handled := input.handleKey(s); handled {
			a.setCurrencyFilter(updated.value())
			a.resetCurrencySelectCursor()
		}
	}
	return a, false
}

func (a App) selectFieldKey(s, field string, options []string, fields []string) (App, bool) {
	if len(options) == 0 {
		return a, false
	}
	if a.Form[field] == "" {
		a.Form[field] = options[0]
	}
	idx := indexOf(options, a.Form[field])
	if idx < 0 {
		idx = 0
		a.Form[field] = options[idx]
	}
	switch s {
	case "down", "j":
		idx = (idx + 1) % len(options)
		a.Form[field] = options[idx]
	case "up", "k":
		idx = (idx - 1 + len(options)) % len(options)
		a.Form[field] = options[idx]
	case "tab":
		a.Field = min(a.Field+1, len(fields))
	case "shift+tab":
		a.Field = max(a.Field-1, 0)
	case "enter":
		a.Field = min(a.Field+1, len(fields))
	}
	return a, false
}

func (a App) currencyOptions() []string {
	currencies, err := a.Svc.Currency.List(a.ctx)
	if err != nil {
		return []string{a.Config.Config.Currency}
	}
	var out []string
	for _, cur := range currencies {
		if cur.Code != a.Config.Config.Currency {
			out = append(out, cur.Code)
		}
	}
	sort.Strings(out)
	out = append([]string{a.Config.Config.Currency}, out...)
	if len(out) == 0 {
		return []string{a.Config.Config.Currency}
	}
	return out
}

func (a App) submitFormKey(s string, fields []string) (App, bool) {
	if isSubmitKey(s) {
		a.clearCurrentTextCursor(fields)
		return a, true
	}
	if s == "enter" {
		if a.Field >= len(fields) {
			return a, true
		}
		a.clearCurrentTextCursor(fields)
		a.Field++
		return a, false
	}
	return a.formKey(s, fields), false
}

func (a App) formKey(s string, fields []string) App {
	if strings.HasPrefix(s, "set ") {
		parts := strings.SplitN(strings.TrimPrefix(s, "set "), "=", 2)
		if len(parts) == 2 {
			a.Form[parts[0]] = normalizeFieldInput(parts[0], "", parts[1])
			a.resetTextCursor(parts[0])
		}
		return a
	}
	if len(fields) == 0 {
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
			a.Form[field] = normalizeFieldValue(field, next)
			if field == "name" || field == "date" || field == "balance" || field == "amount" {
				a.resetTextCursor(field)
			} else {
				a.setTextCursor(field, cursor+len([]rune(s)))
			}
		}
	}
	return a
}

func (a App) formView(fields []string, locked map[string]string) string {
	return a.formViewWithOptions(fields, locked, nil, nil)
}

func (a App) formViewWithOptions(fields []string, locked map[string]string, options map[string][]string, prefixes map[string]string) string {
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
		if value == "" && field == "on-budget" {
			value = "true"
		}
		if locked != nil && locked[field] != "" {
			value = locked[field]
		}
		renderedValue := placeholder(value, placeholderFor(field))
		if field == "tags" {
			renderedValue = formatTags(splitTagNames(value), splitTagNames(a.Form[newTagsKey]))
		}
		if (field == "balance" || field == "amount") && prefixes != nil && prefixes[field] != "" {
			currency := prefixes[field]
			if i == a.Field && isFormTextField(field, options) && (locked == nil || locked[field] == "") {
				renderedValue = renderBalanceCaret(value, currency)
			} else {
				renderedValue = formatBalanceDisplay(value, currency)
			}
		} else if field != "tags" && i == a.Field && isFormTextField(field, options) && (locked == nil || locked[field] == "") {
			renderedValue = renderCaret(value, placeholderFor(field), a.textCursor(field))
		}
		lines = append(lines, fmt.Sprintf("%s%d) %-9s: %s", prefix, i+1, field, renderedValue))
		if i == a.Field && field == "tags" && (locked == nil || locked[field] == "") {
			lines = append(lines, a.tagSelectLines()...)
			continue
		}
		if i == a.Field && field == "category" && (locked == nil || locked[field] == "") {
			lines = append(lines, a.budgetCategorySelectLines()...)
			continue
		}
		if i == a.Field && field == "account" && (locked == nil || locked[field] == "") {
			lines = append(lines, a.transactionAccountSelectLines()...)
			continue
		}
		if i == a.Field && field == "to" && (locked == nil || locked[field] == "") {
			lines = append(lines, a.budgetTransferToSelectLines()...)
			continue
		}
		if i == a.Field && options != nil && len(options[field]) > 0 && (locked == nil || locked[field] == "") {
			selected := value
			fieldOptions := options[field]
			if field == "currency" {
				lines = append(lines, a.currencySelectLines(fieldOptions)...)
				continue
			} else {
				lines = append(lines, "")
			}
			for _, option := range fieldOptions {
				optionPrefix := "       "
				if option == selected {
					optionPrefix = "     > "
				}
				lines = append(lines, optionPrefix+option)
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

func (a App) currencySelectLines(options []string) []string {
	filter := a.currencyFilter()
	filtered := filterOptions(options, filter)
	cursor := clampCurrencyCursor(parseFormInt(a.Form[currencyCursorKey]), len(filtered))
	page := clampCurrencyPage(parseFormInt(a.Form[currencyPageKey]), cursor, len(filtered))
	start := page * currencyPageSize
	end := min(start+currencyPageSize, len(filtered))
	lines := []string{"", fmt.Sprintf("   > filter  : %s", placeholder(filter, "(type anything...)")), ""}
	if len(filtered) == 0 {
		lines = append(lines, "     (no matching currencies)", "", "     [00/00]")
		return lines
	}
	for i, option := range filtered[start:end] {
		optionPrefix := "       "
		if start+i == cursor {
			optionPrefix = "     > "
		}
		lines = append(lines, optionPrefix+option)
	}
	lines = append(lines, "", fmt.Sprintf("     [%02d/%02d]", cursor+1, len(filtered)))
	return lines
}
