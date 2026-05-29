package model

import "strings"

// filteredListInput centralizes filter typing semantics shared by list screens.
type filteredListInput struct {
	filter   string
	sanitize func(string) string
}

type filterableListKeyResult struct {
	filter string
	menu   int
}

func newFilteredListInput(filter string, sanitize func(string) string) filteredListInput {
	if sanitize == nil {
		sanitize = func(s string) string { return s }
	}
	return filteredListInput{filter: filter, sanitize: sanitize}
}

func (f filteredListInput) value() string { return f.filter }

func (f filteredListInput) handleKey(key string) (filteredListInput, bool) {
	switch key {
	case "backspace":
		f.filter = trimLastRune(f.filter)
		return f, true
	default:
		if isTextInputKey(key) {
			f.filter = f.sanitize(f.filter + key)
			return f, true
		}
	}
	return f, false
}

func handleFilterableListKey(key, filter string, menu, rowCount int) (filterableListKeyResult, bool) {
	input := newFilteredListInput(filter, nil)
	switch key {
	case "backspace":
		updated, _ := input.handleKey(key)
		return filterableListKeyResult{
			filter: updated.value(),
			menu:   clampListCursor(menu, rowCount),
		}, true
	default:
		if updated, handled := input.handleKey(key); handled {
			return filterableListKeyResult{
				filter: updated.value(),
				menu:   0,
			}, true
		}
	}
	return filterableListKeyResult{filter: filter, menu: menu}, false
}

func filterStrings(options []string, filter string, sanitize func(string) string) []string {
	if sanitize == nil {
		sanitize = func(s string) string { return s }
	}
	filter = sanitize(filter)
	if filter == "" {
		return options
	}
	var out []string
	for _, option := range options {
		if strings.Contains(option, filter) {
			out = append(out, option)
		}
	}
	return out
}
