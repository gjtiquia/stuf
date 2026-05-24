package model

import "testing"

func TestSanitizeSlug(t *testing.T) {
	tests := map[string]string{
		"HSBC One":           "hsbc-one",
		"foo  bar":           "foo-bar",
		"foo---bar":          "foo-bar",
		"__Foo!! Bar//Baz🙂":  "foo-barbaz",
		"  ---Leading Space": "leading-space",
		"already-good-123":   "already-good-123",
		"trailing space ":    "trailing-space-",
		"under_score":        "underscore",
	}
	for input, want := range tests {
		if got := sanitizeSlug(input); got != want {
			t.Fatalf("sanitizeSlug(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestSanitizeDateInput(t *testing.T) {
	tests := map[string]string{
		"":               "",
		"2026":           "2026",
		"20260":          "2026-0",
		"202605":         "2026-05",
		"2026052":        "2026-05-2",
		"20260524":       "2026-05-24",
		"2026/05/24":     "2026-05-24",
		"2026 05 xx 24":  "2026-05-24",
		"202605241999":   "2026-05-24",
		"abc":            "",
	}
	for input, want := range tests {
		if got := sanitizeDateInput(input); got != want {
			t.Fatalf("sanitizeDateInput(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestAccountFormValues(t *testing.T) {
	got := accountFormValues("cash", "HKD", true, "wallet")
	if got["name"] != "cash" || got["currency"] != "HKD" || got["on-budget"] != "true" || got["notes"] != "wallet" {
		t.Fatalf("unexpected form values: %#v", got)
	}
}
