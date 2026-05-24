package model

import "testing"

func TestNavigationStackPushPopRestore(t *testing.T) {
	nav := NewNavigationStack()
	nav.setCurrentMenu(1)
	nav.Push("/accounts/", 0)
	nav.setCurrentMenu(2)
	nav.Push("/accounts/list/", 0)
	if got := nav.Current(); got.Path != "/accounts/list/" || got.Menu != 0 {
		t.Fatalf("unexpected current frame: %+v", got)
	}
	if !nav.Pop() {
		t.Fatal("expected pop to succeed")
	}
	got := nav.Current()
	if got.Path != "/accounts/" || got.Menu != 2 {
		t.Fatalf("expected popped frame with saved menu, got %+v", got)
	}
}

func TestNavigationStackReplace(t *testing.T) {
	nav := NewNavigationStack()
	nav.setCurrentMenu(3)
	nav.Replace("/accounts/cash/", 3)
	got := nav.Current()
	if got.Path != "/accounts/cash/" || got.Menu != 3 {
		t.Fatalf("replace should keep menu on same frame, got %+v", got)
	}
	if nav.Len() != 1 {
		t.Fatalf("replace should not grow stack, len=%d", nav.Len())
	}
}

func TestNavigationStackReset(t *testing.T) {
	nav := NewNavigationStack()
	nav.Push("/accounts/", 2)
	nav.Push("/accounts/list/", 4)
	nav.Reset()
	got := nav.Current()
	if got.Path != "/" || got.Menu != 0 || nav.Len() != 1 {
		t.Fatalf("reset should clear stack to root, got %+v len=%d", got, nav.Len())
	}
}

func TestNavigationStackSetBelowTop(t *testing.T) {
	nav := NewNavigationStack()
	nav.Push("/accounts/", 3)
	nav.Push("/accounts/create/", 0)
	if !nav.SetBelowTop("/accounts/", 1) {
		t.Fatal("expected set below top to succeed")
	}
	if got := nav.frames[len(nav.frames)-2]; got.Path != "/accounts/" || got.Menu != 1 {
		t.Fatalf("expected parent frame updated, got %+v", got)
	}
}
