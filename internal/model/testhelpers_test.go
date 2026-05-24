package model

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"stuf/internal/repo"
)

func assertOrdered(t *testing.T, body string, earlier string, later string) {
	t.Helper()
	i := strings.Index(body, earlier)
	j := strings.Index(body, later)
	if i == -1 || j == -1 || i >= j {
		t.Fatalf("expected %q before %q in:\n%s", earlier, later, body)
	}
}

func assertRenderOrder(t *testing.T, view string, segments ...string) {
	t.Helper()
	pos := 0
	for _, segment := range segments {
		idx := strings.Index(view[pos:], segment)
		if idx == -1 {
			t.Fatalf("render order missing %q after position %d in:\n%s", segment, pos, view)
		}
		pos += idx + len(segment)
	}
}

func assertViewContains(t *testing.T, view string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(view, want) {
			t.Fatalf("view missing %q:\n%s", want, view)
		}
	}
}

func setCurrencyRate(t *testing.T, store *repo.Store, code string, amount int64, scale int) {
	t.Helper()
	if err := store.SetCurrencyRate(context.Background(), code, amount, scale); err != nil {
		t.Fatal(err)
	}
}

func press(app App, key tea.KeyType) App {
	m, _ := app.Update(tea.KeyMsg{Type: key})
	return m.(App)
}

func pressRunes(app App, s string) App {
	for _, r := range s {
		m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		app = m.(App)
	}
	return app
}

type standardAccounts struct {
	Cash   int64
	USD    int64
	Loan   int64
	Hidden int64
}

func seedStandardAccounts(t *testing.T, app App, store *repo.Store) standardAccounts {
	t.Helper()
	ctx := context.Background()
	setCurrencyRate(t, store, "HKD", 1, 0)
	setCurrencyRate(t, store, "USD", 10, 0)
	cash, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	usd, _, err := app.Svc.Accounts.Create(ctx, "usd-savings", "USD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	loan, _, err := app.Svc.Accounts.Create(ctx, "student-loan", "HKD", false, "negative until fully paid")
	if err != nil {
		t.Fatal(err)
	}
	hidden, _, err := app.Svc.Accounts.Create(ctx, "old-account", "HKD", true, "closed")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Accounts.SetHidden(ctx, hidden.ID, true); err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, cash.ID, "2026-05-21", "100.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, usd.ID, "2026-05-21", "50.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, loan.ID, "2026-05-21", "-25.00", ""); err != nil {
		t.Fatal(err)
	}
	return standardAccounts{Cash: cash.ID, USD: usd.ID, Loan: loan.ID, Hidden: hidden.ID}
}
