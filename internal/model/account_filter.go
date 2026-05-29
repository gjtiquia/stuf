package model

import (
	"strings"
)

type accountFilter struct {
	Terms []accountFilterTerm
}

type accountFilterTerm struct {
	Kind   string
	Values []string
}

func parseAccountFilter(input string) accountFilter {
	var out accountFilter
	for _, raw := range strings.Fields(input) {
		if strings.Contains(raw, ":") {
			parts := strings.SplitN(raw, ":", 2)
			kind := strings.ToLower(parts[0])
			var values []string
			for _, value := range strings.Split(parts[1], ",") {
				value = strings.TrimSpace(value)
				if value != "" {
					values = append(values, value)
				}
			}
			if len(values) > 0 && (kind == "tag" || kind == "currency") {
				out.Terms = append(out.Terms, accountFilterTerm{Kind: kind, Values: values})
				continue
			}
		}
		out.Terms = append(out.Terms, accountFilterTerm{Kind: "text", Values: []string{raw}})
	}
	return out
}

func (f accountFilter) Empty() bool { return len(f.Terms) == 0 }

func (f accountFilter) Match(row accountListRow) bool {
	for _, term := range f.Terms {
		if !term.Match(row) {
			return false
		}
	}
	return true
}

func (t accountFilterTerm) Match(row accountListRow) bool {
	for _, value := range t.Values {
		switch t.Kind {
		case "tag":
			for _, tag := range row.Tags {
				if tag == value {
					return true
				}
			}
		case "currency":
			if strings.EqualFold(row.Currency, value) {
				return true
			}
		default:
			needle := strings.ToLower(value)
			if strings.Contains(strings.ToLower(row.Name), needle) ||
				strings.Contains(strings.ToLower(row.Notes), needle) ||
				strings.Contains(strings.ToLower(row.Currency), needle) ||
				strings.Contains(strings.ToLower(row.CurrencyName), needle) {
				return true
			}
			for _, tag := range row.Tags {
				if strings.Contains(strings.ToLower(tag), needle) {
					return true
				}
			}
		}
	}
	return false
}
