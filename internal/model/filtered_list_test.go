package model

import "testing"

func TestFilteredListInputTypingAndBackspace(t *testing.T) {
	input := newFilteredListInput("", nil)
	updated, handled := input.handleKey("j")
	if !handled || updated.value() != "j" {
		t.Fatalf("expected j appended, got %q handled=%t", updated.value(), handled)
	}
	updated, handled = updated.handleKey("k")
	if !handled || updated.value() != "jk" {
		t.Fatalf("expected jk, got %q", updated.value())
	}
	updated, handled = updated.handleKey("backspace")
	if !handled || updated.value() != "j" {
		t.Fatalf("expected j after backspace, got %q", updated.value())
	}
}

func TestFilteredListInputSanitizesCurrencyCodes(t *testing.T) {
	input := newFilteredListInput("", sanitizeCurrencyCode)
	updated, handled := input.handleKey("j py!!")
	if !handled || updated.value() != "JPY" {
		t.Fatalf("expected JPY, got %q handled=%t", updated.value(), handled)
	}
}

func TestFilterStringsUsesSanitizer(t *testing.T) {
	options := []string{"HKD", "JPY", "USD"}
	got := filterStrings(options, "jpy", sanitizeCurrencyCode)
	if len(got) != 1 || got[0] != "JPY" {
		t.Fatalf("filterStrings = %#v", got)
	}
}

func TestFilteredListInputIgnoresNavigationKeys(t *testing.T) {
	input := newFilteredListInput("cash", nil)
	for _, key := range []string{"up", "down", "tab", "enter", "esc"} {
		updated, handled := input.handleKey(key)
		if handled || updated.value() != "cash" {
			t.Fatalf("key %q should not change filter, got %q handled=%t", key, updated.value(), handled)
		}
	}
}
