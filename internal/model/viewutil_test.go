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

func TestAccountFormValues(t *testing.T) {
	got := accountFormValues("cash", "HKD", true, "wallet")
	if got["name"] != "cash" || got["currency"] != "HKD" || got["on-budget"] != "true" || got["notes"] != "wallet" {
		t.Fatalf("unexpected form values: %#v", got)
	}
}
