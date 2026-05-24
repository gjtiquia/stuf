package model

import "strings"

// filteredListInput centralizes filter typing semantics shared by list screens.
type filteredListInput struct {
	filter   string
	sanitize func(string) string
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
			f.filter += f.sanitize(key)
			return f, true
		}
	}
	return f, false
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
