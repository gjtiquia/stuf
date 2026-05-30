package model

import "strings"

type transactionFilter struct {
	Terms []transactionFilterTerm
}

type transactionFilterTerm struct {
	Kind    string
	Values  []string
	Negated bool
}

type transactionListRow struct {
	ID          int64
	Ref         string
	RefNumber   int64
	ParentID    *int64
	Date        string
	Type        string
	Amount      string
	Account     string
	Currency    string
	Notes       string
	Tags        []string
	Depth       int
	Selectable  bool
	HasChildren bool
}

func parseTransactionFilter(input string) transactionFilter {
	var out transactionFilter
	for _, raw := range strings.Fields(input) {
		term := raw
		negated := false
		if strings.HasPrefix(term, "-") {
			negated = true
			term = strings.TrimPrefix(term, "-")
		}
		if strings.Contains(term, ":") {
			parts := strings.SplitN(term, ":", 2)
			kind := strings.ToLower(parts[0])
			var values []string
			for _, value := range strings.Split(parts[1], ",") {
				value = strings.TrimSpace(value)
				if value != "" {
					values = append(values, value)
				}
			}
			if len(values) > 0 && (kind == "tag" || kind == "currency" || kind == "type") {
				out.Terms = append(out.Terms, transactionFilterTerm{Kind: kind, Values: values, Negated: negated})
				continue
			}
		}
		out.Terms = append(out.Terms, transactionFilterTerm{Kind: "text", Values: []string{raw}})
	}
	return out
}

func (f transactionFilter) Empty() bool { return len(f.Terms) == 0 }

func (f transactionFilter) Match(row transactionListRow) bool {
	for _, term := range f.Terms {
		if !term.Match(row) {
			return false
		}
	}
	return true
}

func (t transactionFilterTerm) Match(row transactionListRow) bool {
	matched := t.matchPositive(row)
	if t.Negated {
		return !matched
	}
	return matched
}

func (t transactionFilterTerm) matchPositive(row transactionListRow) bool {
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
		case "type":
			if strings.EqualFold(row.Type, value) {
				return true
			}
		default:
			needle := strings.ToLower(value)
			if strings.Contains(strings.ToLower(row.Ref), needle) ||
				strings.Contains(strings.ToLower(row.Date), needle) ||
				strings.Contains(strings.ToLower(row.Type), needle) ||
				strings.Contains(strings.ToLower(row.Account), needle) ||
				strings.Contains(strings.ToLower(row.Currency), needle) ||
				strings.Contains(strings.ToLower(row.Notes), needle) {
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
