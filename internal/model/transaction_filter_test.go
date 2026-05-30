package model

import "testing"

func TestTransactionFilterTextTagCurrencyAndTypeTerms(t *testing.T) {
	row := transactionListRow{
		Ref:      "tx-000001",
		Date:     "2026-05-28",
		Type:     "expense",
		Account:  "amex",
		Currency: "HKD",
		Notes:    "statement payment",
		Tags:     []string{"credit-card", "person/alice"},
	}
	tests := []struct {
		name   string
		filter string
		want   bool
	}{
		{name: "text notes", filter: "statement", want: true},
		{name: "text account", filter: "amex", want: true},
		{name: "text tag", filter: "alice", want: true},
		{name: "tag exact", filter: "tag:person/alice", want: true},
		{name: "currency exact", filter: "currency:HKD", want: true},
		{name: "type exact", filter: "type:expense", want: true},
		{name: "comma or", filter: "tag:missing,credit-card", want: true},
		{name: "repeated terms are and", filter: "type:expense tag:credit-card currency:HKD", want: true},
		{name: "and can fail", filter: "type:income tag:credit-card", want: false},
		{name: "negative tag excludes matching transaction", filter: "-tag:credit-card", want: false},
		{name: "negative tag allows non-matching transaction", filter: "-tag:missing", want: true},
		{name: "negative type excludes matching transaction", filter: "-type:expense", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseTransactionFilter(tt.filter).Match(row); got != tt.want {
				t.Fatalf("Match(%q) = %v, want %v", tt.filter, got, tt.want)
			}
		})
	}
}
