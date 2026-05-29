package model

import "testing"

func TestAccountFilterTextTagAndCurrencyTerms(t *testing.T) {
	row := accountListRow{
		Name:         "cash",
		Notes:        "daily wallet",
		Currency:     "HKD",
		CurrencyName: "Hong Kong dollar",
		Tags:         []string{"family/shared", "wallet"},
	}
	tests := []struct {
		name   string
		filter string
		want   bool
	}{
		{name: "text notes", filter: "daily", want: true},
		{name: "text currency name", filter: "kong", want: true},
		{name: "text tag", filter: "shared", want: true},
		{name: "tag exact", filter: "tag:family/shared", want: true},
		{name: "comma or", filter: "tag:missing,wallet", want: true},
		{name: "repeated terms are and", filter: "tag:wallet currency:HKD", want: true},
		{name: "and can fail", filter: "tag:wallet currency:USD", want: false},
		{name: "negative tag excludes matching account", filter: "-tag:wallet", want: false},
		{name: "negative tag allows non-matching account", filter: "-tag:missing", want: true},
		{name: "negative tag comma excludes any matching tag", filter: "-tag:missing,wallet", want: false},
		{name: "positive and negative terms combine", filter: "currency:HKD -tag:missing", want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseAccountFilter(tt.filter).Match(row); got != tt.want {
				t.Fatalf("Match(%q) = %v, want %v", tt.filter, got, tt.want)
			}
		})
	}
}
