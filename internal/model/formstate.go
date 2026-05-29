package model

import (
	"strconv"
	"strings"
)

const (
	formKeyFilter       = "filter"
	currencyFilterKey   = "_currency_filter"
	currencyCursorKey   = "_currency_cursor"
	currencyPageKey     = "_currency_page"
	tagFilterKey        = "_tag_filter"
	tagCursorKey        = "_tag_cursor"
	tagPageKey          = "_tag_page"
	newTagsKey          = "_new_tags"
	textCursorKeyPrefix = "_cursor:"
)

func cursorKey(field string) string { return textCursorKeyPrefix + field }

func (a App) listFilter() string { return a.Form[formKeyFilter] }

func (a App) setListFilter(value string) { a.Form[formKeyFilter] = value }

func (a App) trimListFilter() {
	a.Form[formKeyFilter] = trimLastRune(a.Form[formKeyFilter])
}

func (a App) currencyFilter() string { return a.Form[currencyFilterKey] }

func (a App) setCurrencyFilter(value string) { a.Form[currencyFilterKey] = value }

func (a App) appendCurrencyFilter(value string) {
	a.Form[currencyFilterKey] += value
}

func (a App) trimCurrencyFilter() {
	a.Form[currencyFilterKey] = trimLastRune(a.Form[currencyFilterKey])
}

func (a App) clearCurrencySelectState() {
	delete(a.Form, currencyFilterKey)
	delete(a.Form, currencyCursorKey)
	delete(a.Form, currencyPageKey)
}

func (a App) tagFilter() string { return a.Form[tagFilterKey] }

func (a App) setTagFilter(value string) { a.Form[tagFilterKey] = sanitizeTagSlug(value) }

func (a App) clearTagSelectState() {
	delete(a.Form, tagFilterKey)
	delete(a.Form, tagCursorKey)
	delete(a.Form, tagPageKey)
}

func (a App) setTagSelectCursor(cursor int) {
	a.Form[tagCursorKey] = strconv.Itoa(cursor)
	a.Form[tagPageKey] = strconv.Itoa(cursor / currencyPageSize)
}

func (a App) resetTagSelectCursor() {
	a.setTagSelectCursor(0)
}

func (a App) tagSelectPage() int {
	return parseFormInt(a.Form[tagPageKey])
}

func (a App) setTagSelectPage(page int) {
	a.Form[tagPageKey] = strconv.Itoa(page)
}

func (a App) currencySelectCursor() int {
	return clampCursor(parseFormInt(a.Form[currencyCursorKey]), len(filterOptions(a.currencyOptions(), a.currencyFilter())))
}

func (a App) setCurrencySelectCursor(cursor int) {
	a.Form[currencyCursorKey] = strconv.Itoa(cursor)
	a.Form[currencyPageKey] = strconv.Itoa(currencyPageForCursor(cursor))
}

func (a App) resetCurrencySelectCursor() {
	a.setCurrencySelectCursor(0)
}

func (a App) currencySelectPage() int {
	return parseFormInt(a.Form[currencyPageKey])
}

func (a App) setCurrencySelectPage(page int) {
	a.Form[currencyPageKey] = strconv.Itoa(page)
}

func (a App) textCursor(field string) int {
	size := len([]rune(a.Form[field]))
	raw, ok := a.Form[cursorKey(field)]
	if !ok {
		return size
	}
	cursor := parseFormInt(raw)
	if cursor < 0 {
		return 0
	}
	if cursor > size {
		return size
	}
	return cursor
}

func (a App) setTextCursor(field string, cursor int) {
	size := len([]rune(a.Form[field]))
	cursor = max(0, min(cursor, size))
	a.Form[cursorKey(field)] = strconv.Itoa(cursor)
}

func (a App) resetTextCursor(field string) {
	a.setTextCursor(field, len([]rune(a.Form[field])))
}

func (a App) moveTextCursor(field string, delta int) {
	a.setTextCursor(field, a.textCursor(field)+delta)
}

func (a App) clearCurrentTextCursor(fields []string) {
	if a.Field < len(fields) {
		delete(a.Form, cursorKey(fields[a.Field]))
	}
}

func parseFormInt(value string) int {
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return n
}

func clampCursor(cursor, count int) int {
	if count <= 0 || cursor < 0 {
		return 0
	}
	if cursor >= count {
		return count - 1
	}
	return cursor
}

func clampListCursor(cursor, count int) int     { return clampCursor(cursor, count) }
func clampCurrencyCursor(cursor, count int) int { return clampCursor(cursor, count) }

func currencyPageForCursor(cursor int) int {
	if cursor < 0 {
		return 0
	}
	return cursor / currencyPageSize
}

func currencyPageCount(count int) int {
	if count <= 0 {
		return 1
	}
	return (count + currencyPageSize - 1) / currencyPageSize
}

func clampCurrencyPage(page, cursor, count int) int {
	if count <= 0 {
		return 0
	}
	page = max(page, 0)
	page = min(page, currencyPageCount(count)-1)
	cursorPage := currencyPageForCursor(cursor)
	if cursorPage != page {
		return cursorPage
	}
	return page
}

func filterOptions(options []string, filter string) []string {
	return filterStrings(options, filter, sanitizeCurrencyCode)
}

func sanitizeCurrencyCode(input string) string {
	var b strings.Builder
	for _, r := range input {
		switch {
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= 'a' && r <= 'z':
			b.WriteRune(r + ('A' - 'a'))
		}
	}
	return b.String()
}

func isFormTextField(field string, options map[string][]string) bool {
	return options == nil || len(options[field]) == 0
}

func renderCaret(value, fallback string, cursor int) string {
	if value == "" {
		return "|"
	}
	runes := []rune(value)
	cursor = max(0, min(cursor, len(runes)))
	return string(runes[:cursor]) + "|" + string(runes[cursor:])
}
