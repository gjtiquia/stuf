package model

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"stuf/internal/config"
	"stuf/internal/repo"
	"stuf/internal/service"
)

func testApp(t *testing.T) (App, *repo.Store) {
	t.Helper()
	ctx := context.Background()
	s, err := repo.Open(ctx, filepath.Join(t.TempDir(), "db.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	s.Clock = func() time.Time { return time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC) }
	h := service.HistoryService{Repo: s.Hist, Now: s.Clock}
	cfg := config.Loaded{Config: config.Config{Currency: "HKD"}, Path: "/tmp/config.jsonc"}
	return New(ctx, Services{
		Accounts:  service.AccountService{Store: s, Accounts: s.Acct, Balances: s.Bal, Currency: s.Cur, History: h, AppCurrency: "HKD"},
		Balances:  service.BalanceService{Store: s, Accounts: s.Acct, Balances: s.Bal, History: h},
		Dashboard: service.DashboardService{Accounts: s.Acct, Balances: s.Bal, Currencies: s.Cur, AppCurrency: "HKD", Now: s.Clock},
		History:   h,
		Backup: func(context.Context) (string, error) {
			return "/tmp/db.2026-05-24-1200.sqlite", nil
		},
	}, cfg), s
}

func appWithNav(app App, frames ...navFrame) App {
	app.Nav = NavigationStack{frames: frames}
	return app.syncFromNav()
}

func TestDashboardRendersEmptyStateAndTODOs(t *testing.T) {
	app, _ := testApp(t)
	view := app.View()
	for _, want := range []string{"# stuf", "total       : HKD 0.00", "period      : 2026-05", "transactions (TODO)"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view missing %q:\n%s", want, view)
		}
	}
}

func TestURLRendersImmediatelyAboveActions(t *testing.T) {
	app, _ := testApp(t)
	view := app.View()
	assertOrdered(t, view, "ppl owe you : HKD 0.00", "\n/\n\n> 1) accounts")
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	app = m.(App)
	view = app.View()
	assertOrdered(t, view, "ppl owe you : HKD 0.00", "\n/accounts/\n\n> 1) overview")
}

func TestMenuNavigationMovesVisibleSelection(t *testing.T) {
	app, _ := testApp(t)
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyDown})
	app = m.(App)
	view := app.View()
	if !strings.Contains(view, "> 2) transactions (TODO)") {
		t.Fatalf("down did not move dashboard marker:\n%s", view)
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	app = m.(App)
	view = app.View()
	if !strings.Contains(view, "> 1) accounts") {
		t.Fatalf("k did not move dashboard marker back:\n%s", view)
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	app = m.(App)
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	app = m.(App)
	view = app.View()
	if !strings.Contains(view, "> 2) list") {
		t.Fatalf("j did not move account menu marker:\n%s", view)
	}
}

func TestAccountsDashboardMatchesReadmeActions(t *testing.T) {
	app, _ := testApp(t)
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	app = m.(App)
	view := app.View()
	for _, want := range []string{
		"total       : HKD 0.00",
		"/accounts/",
		"> 1) overview",
		"  2) list",
		"  3) hidden",
		"  4) create",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("accounts dashboard missing %q:\n%s", want, view)
		}
	}
}

func TestAccountListReadmeShape(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	if _, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, "main cash"); err != nil {
		t.Fatal(err)
	}
	off, _, err := app.Svc.Accounts.Create(ctx, "investment", "HKD", false, "brokerage")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Accounts.SetHidden(ctx, off.ID, true); err != nil {
		t.Fatal(err)
	}
	app.Path = "/accounts/list/"
	view := app.View()
	for _, want := range []string{
		"> filter : (type anything...)",
		"on-budget accounts",
		"| balance",
		"| notes",
		"TOTAL |",
		"> cash",
		"main cash",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("account list missing %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "> 1) cash") {
		t.Fatalf("account list rows should not render menu numbers:\n%s", view)
	}
	assertOrdered(t, view, "TOTAL |", "\n\n> cash")
	app.Path = "/accounts/hidden/"
	app.Menu = 0
	view = app.View()
	for _, want := range []string{"> filter : (type anything...)", "| balance", "> investment", "brokerage"} {
		if !strings.Contains(view, want) {
			t.Fatalf("hidden accounts missing %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "> 1) investment") {
		t.Fatalf("hidden account rows should not render menu numbers:\n%s", view)
	}
}

func TestAccountListNoResultsShape(t *testing.T) {
	app, _ := testApp(t)
	app.Form["filter"] = "amex"
	app.Path = "/accounts/list/"
	view := app.View()
	if !strings.Contains(view, "> filter : amex") || strings.Contains(view, "> filter : amex|") || !strings.Contains(view, "(no results)") {
		t.Fatalf("no-results shape missing:\n%s", view)
	}
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = m.(App)
	if app.Path != "/accounts/list/" {
		t.Fatalf("enter on no results should stay on list, got %s", app.Path)
	}
}

func TestAccountListFilterTypingAndFilteredNavigation(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	for _, name := range []string{"cash", "savings", "travel"} {
		if _, _, err := app.Svc.Accounts.Create(ctx, name, "HKD", true, ""); err != nil {
			t.Fatal(err)
		}
	}
	app.Path = "/accounts/list/"
	for _, r := range "sav" {
		m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		app = m.(App)
	}
	view := app.View()
	if !strings.Contains(view, "> filter : sav") || !strings.Contains(view, "> savings") || strings.Contains(view, "cash") || strings.Contains(view, "travel") {
		t.Fatalf("typed filter did not narrow account list:\n%s", view)
	}
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	app = m.(App)
	if view = app.View(); !strings.Contains(view, "> filter : sa") || !strings.Contains(view, "> savings") {
		t.Fatalf("backspace did not update account filter:\n%s", view)
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = m.(App)
	if app.Path != "/accounts/savings/" {
		t.Fatalf("enter should open selected filtered account, got %s", app.Path)
	}
}

func TestAccountListTableColumnsAlignWithConvertedBalances(t *testing.T) {
	app, store := testApp(t)
	ctx := context.Background()
	setCurrencyRate(t, store, "HKD", 1, 0)
	setCurrencyRate(t, store, "USD", 10, 0)
	cfg := app.Config
	cfg.Config.Currency = "USD"
	app.Config = cfg
	acct, _, err := app.Svc.Accounts.Create(ctx, "hsbc-one", "HKD", true, "dunno if need to split...?")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, acct.ID, "2026-05-21", "0.00", ""); err != nil {
		t.Fatal(err)
	}
	app.Path = "/accounts/list/"
	view := app.View()
	var tableLines []string
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(line, " | ") && (strings.Contains(line, "name") || strings.Contains(line, "TOTAL") || strings.Contains(line, "hsbc-one")) {
			tableLines = append(tableLines, line)
		}
	}
	if len(tableLines) < 3 {
		t.Fatalf("expected header, total, and account rows, got %d lines:\n%s", len(tableLines), view)
	}
	wantIdx := notesColumnIndex(tableLines[0])
	for i, line := range tableLines[1:] {
		if got := notesColumnIndex(line); got != wantIdx {
			t.Fatalf("notes column misaligned on line %d: want index %d, got %d\n%s", i+1, wantIdx, got, view)
		}
	}
}

func notesColumnIndex(line string) int {
	idx := -1
	start := 0
	for range 2 {
		pos := strings.Index(line[start:], " |")
		if pos < 0 {
			return -1
		}
		idx = start + pos
		start = idx + 2
	}
	return idx
}

func TestAccountListTotalsAndForeignCurrencyDisplay(t *testing.T) {
	app, store := testApp(t)
	ctx := context.Background()
	setCurrencyRate(t, store, "HKD", 1, 0)
	setCurrencyRate(t, store, "USD", 10, 0)
	cash, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, "wallet")
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
	if _, _, err := app.Svc.Balances.Add(ctx, cash.ID, "2026-05-21", "100.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, usd.ID, "2026-05-21", "50.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, loan.ID, "2026-05-21", "-25.00", ""); err != nil {
		t.Fatal(err)
	}
	app.Path = "/accounts/list/"
	view := app.View()
	for _, want := range []string{
		"| HKD 600.00",
		"> cash",
		"usd-savings",
		"HKD 500.00 (USD 50.00)",
		"| HKD -25.00",
		"student-loan",
		"negative until fully paid",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("account list totals/conversion missing %q:\n%s", want, view)
		}
	}
}

func TestAccountDetailVisibleAndHiddenReadmeShape(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	acct, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, "wallet")
	if err != nil {
		t.Fatal(err)
	}
	app.Path = "/accounts/cash/"
	view := app.View()
	for _, want := range []string{
		"name      : cash",
		"balance   : HKD 0.00",
		"as of     : (no balance entered yet)",
		"on-budget : true",
		"notes     : wallet",
		"> 1) balances",
		"  2) transactions (TODO)",
		"  3) edit account",
		"  4) hide account",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("visible account detail missing %q:\n%s", want, view)
		}
	}
	if _, _, err := app.Svc.Accounts.SetHidden(ctx, acct.ID, true); err != nil {
		t.Fatal(err)
	}
	app.Path = "/accounts/cash/"
	app.Menu = 0
	view = app.View()
	for _, want := range []string{"hidden    : true", "> 1) balances", "  2) transactions (TODO)", "  3) edit account", "  4) show account"} {
		if !strings.Contains(view, want) {
			t.Fatalf("hidden account detail missing %q:\n%s", want, view)
		}
	}
}

func TestFormsReadmeShapeAndLockedCurrency(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	app.Path = "/accounts/create/"
	view := app.View()
	for _, want := range []string{"> 1) name     : |", "2) currency : HKD", "3) on-budget: true", "4) notes", "[confirm]"} {
		if !strings.Contains(view, want) {
			t.Fatalf("account create form missing %q:\n%s", want, view)
		}
	}
	acct, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, acct.ID, "2026-05-01", "1.00", ""); err != nil {
		t.Fatal(err)
	}
	app.Path = "/accounts/cash/edit/"
	app.Form = map[string]string{"name": "cash", "currency": "HKD", "on-budget": "true"}
	view = app.View()
	if !strings.Contains(view, "currency : HKD (locked because balances exist)") {
		t.Fatalf("locked currency missing:\n%s", view)
	}
	app.Path = "/accounts/cash/balances/add/"
	app.Form = map[string]string{"date": "2026-05-24"}
	view = app.View()
	for _, want := range []string{"> 1) date", "2026-05-24|", "2) balance", "(type amount...)", "3) notes", "[confirm]"} {
		if !strings.Contains(view, want) {
			t.Fatalf("balance add form missing %q:\n%s", want, view)
		}
	}
}

func TestAccountCreateSelectFocusAndConfirm(t *testing.T) {
	app, store := testApp(t)
	app.Path = "/accounts/create/"
	view := app.View()
	for _, want := range []string{"> 1) name     : |", "2) currency : HKD", "3) on-budget: true", "  [confirm]"} {
		if !strings.Contains(view, want) {
			t.Fatalf("initial account form missing %q:\n%s", want, view)
		}
	}
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyTab})
	app = m.(App)
	view = app.View()
	for _, want := range []string{"> 2) currency", "   > filter  : (type anything...)", "     > HKD", "       AUD", "       BRL", "       CAD", "     [01/30]", "type       : filter", "left/right : next/prev page"} {
		if !strings.Contains(view, want) {
			t.Fatalf("currency select missing %q:\n%s", want, view)
		}
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyTab})
	app = m.(App)
	view = app.View()
	for _, want := range []string{"> 3) on-budget", "     > true", "false"} {
		if !strings.Contains(view, want) {
			t.Fatalf("on-budget select missing %q:\n%s", want, view)
		}
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyDown})
	app = m.(App)
	if view = app.View(); !strings.Contains(view, "on-budget: false") || !strings.Contains(view, "     > false") {
		t.Fatalf("on-budget select did not choose false:\n%s", view)
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = m.(App)
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyTab})
	app = m.(App)
	if view = app.View(); !strings.Contains(view, "> [confirm]") || !strings.Contains(view, "shift-tab : navigate") {
		t.Fatalf("confirm focus/footer missing:\n%s", view)
	}
	app.Form["name"] = "off-budget-cash"
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = m.(App)
	if app.Path != "/accounts/list/" {
		t.Fatalf("confirm did not submit account form: %s\n%s", app.Path, app.View())
	}
	acct, err := store.Acct.GetByName(context.Background(), "off-budget-cash")
	if err != nil {
		t.Fatal(err)
	}
	if acct.OnBudget {
		t.Fatal("expected account to be created off-budget")
	}
}

func TestAccountFormSpacingMatchesReadmeComponents(t *testing.T) {
	app, _ := testApp(t)
	app.Path = "/accounts/create/"
	view := app.View()
	assertOrdered(t, view, "> 1) name", "\n\n  2) currency")
	assertOrdered(t, view, "  2) currency", "\n\n  3) on-budget")
	assertOrdered(t, view, "  4) notes", "\n\n  [confirm]")
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyTab})
	app = m.(App)
	view = app.View()
	assertOrdered(t, view, "> 2) currency", "\n\n   > filter  : (type anything...)")
	assertOrdered(t, view, "   > filter  : (type anything...)", "\n\n     > HKD")
	assertOrdered(t, view, "     [01/30]", "\n\n  3) on-budget")
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyTab})
	app = m.(App)
	view = app.View()
	assertOrdered(t, view, "> 3) on-budget", "\n\n     > true")
	assertOrdered(t, view, "     > true", "\n       false\n\n  4) notes")
}

func TestCurrencySelectFiltersTypedInput(t *testing.T) {
	app, _ := testApp(t)
	app.Path = "/accounts/create/"
	app.Field = 1
	for _, r := range "j py!!🙂" {
		m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		app = m.(App)
	}
	view := app.View()
	for _, want := range []string{"currency : HKD", "filter  : JPY", "     > JPY", "     [01/01]"} {
		if !strings.Contains(view, want) {
			t.Fatalf("currency filter missing %q:\n%s", want, view)
		}
	}
	for _, unwanted := range []string{"       HKD", "       USD"} {
		if strings.Contains(view, unwanted) {
			t.Fatalf("currency filter should hide %q:\n%s", unwanted, view)
		}
	}
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	app = m.(App)
	if view = app.View(); !strings.Contains(view, "filter  : JP") || strings.Contains(view, "filter  : JP|") || !strings.Contains(view, "JPY") {
		t.Fatalf("currency backspace did not update filter:\n%s", view)
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("zz")})
	app = m.(App)
	if view = app.View(); !strings.Contains(view, "filter  : JPZZ") || strings.Contains(view, "filter  : JPZZ|") || !strings.Contains(view, "(no matching currencies)") || !strings.Contains(view, "[00/00]") {
		t.Fatalf("currency no-match state missing:\n%s", view)
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = m.(App)
	if app.Field != 1 {
		t.Fatalf("enter should stay on currency when filter has no matches, field=%d", app.Field)
	}
	if app.Form["currency"] != "HKD" {
		t.Fatalf("no-match filter should not overwrite selected currency, got %q", app.Form["currency"])
	}
	for i := 0; i < 2; i++ {
		m, _ = app.Update(tea.KeyMsg{Type: tea.KeyBackspace})
		app = m.(App)
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = m.(App)
	if app.Field != 2 {
		t.Fatalf("enter should confirm highlighted currency and move to on-budget, field=%d", app.Field)
	}
	if app.Form["currency"] != "JPY" {
		t.Fatalf("enter should confirm highlighted currency, got %q", app.Form["currency"])
	}
	if _, ok := app.Form["_currency_filter"]; ok {
		t.Fatal("currency filter should clear after confirm")
	}
}

func TestCurrencySelectNavigationPaginationAndSetSanitization(t *testing.T) {
	app, store := testApp(t)
	ctx := context.Background()
	now := store.Clock().UTC().Format(time.RFC3339)
	for _, code := range []string{"AUD", "BRL", "CAD", "CHF", "CNY", "INR", "KRW", "MXN", "NZD", "SGD", "THB"} {
		if _, err := store.DB.ExecContext(ctx, `INSERT INTO currencies(code, name, scale, created_at, updated_at) VALUES (?, ?, 2, ?, ?)
			ON CONFLICT(code) DO UPDATE SET name=excluded.name`, code, code, now, now); err != nil {
			t.Fatal(err)
		}
	}
	app.Path = "/accounts/create/"
	app.Field = 1
	if view := app.View(); !strings.Contains(view, "     > HKD") || !strings.Contains(view, "       AUD") {
		t.Fatalf("app currency should be first with remaining currencies alphabetical:\n%s", view)
	}
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyDown})
	app = m.(App)
	if view := app.View(); !strings.Contains(view, "     > AUD") {
		t.Fatalf("down should move option cursor:\n%s", view)
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	app = m.(App)
	if view := app.View(); !strings.Contains(view, "filter  : K") || strings.Contains(view, "filter  : K|") || !strings.Contains(view, "KRW") {
		t.Fatalf("k should type into currency filter:\n%s", view)
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	app = m.(App)
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRight})
	app = m.(App)
	if view := app.View(); !strings.Contains(view, "     > EUR") || !strings.Contains(view, "[09/30]") {
		t.Fatalf("right should move to next currency page:\n%s", view)
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyLeft})
	app = m.(App)
	if view := app.View(); !strings.Contains(view, "     > HKD") || !strings.Contains(view, "[01/30]") {
		t.Fatalf("left should move back to first currency page:\n%s", view)
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("set currency=u sd!!")})
	app = m.(App)
	if app.Form["currency"] != "USD" {
		t.Fatalf("set currency should sanitize to uppercase no-space code, got %q", app.Form["currency"])
	}
}

func TestAccountEditSelectToggleAndLockedCurrencyOptions(t *testing.T) {
	app, store := testApp(t)
	ctx := context.Background()
	acct, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, "wallet")
	if err != nil {
		t.Fatal(err)
	}
	app.Path = "/accounts/cash/edit/"
	app.Form = map[string]string{"name": "cash", "currency": "HKD", "on-budget": "true", "notes": "wallet"}
	app.Field = 2
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyDown})
	app = m.(App)
	if view := app.View(); !strings.Contains(view, "on-budget: false") || !strings.Contains(view, "     > false") {
		t.Fatalf("edit on-budget select did not toggle:\n%s", view)
	}
	app.Field = 4
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = m.(App)
	updated, err := store.Acct.GetByName(ctx, "cash")
	if err != nil {
		t.Fatal(err)
	}
	if updated.OnBudget {
		t.Fatal("expected edit to persist off-budget")
	}
	if _, _, err := app.Svc.Balances.Add(ctx, acct.ID, "2026-05-01", "1.00", ""); err != nil {
		t.Fatal(err)
	}
	app.Path = "/accounts/cash/edit/"
	app.Form = map[string]string{"name": "cash", "currency": "HKD", "on-budget": "false"}
	app.Field = 1
	view := app.View()
	if !strings.Contains(view, "currency : HKD (locked because balances exist)") {
		t.Fatalf("locked currency missing:\n%s", view)
	}
	if strings.Contains(view, "     > HKD") {
		t.Fatalf("locked currency should not render selectable options:\n%s", view)
	}
}

func TestBalancesScreensReadmeShape(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	acct, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	app.Path = "/accounts/cash/balances/"
	view := app.View()
	for _, want := range []string{"name        : cash", "balance     : HKD 0.00", "date       | balance", "(no balances yet)", "> 1) add balance"} {
		if !strings.Contains(view, want) {
			t.Fatalf("empty balances missing %q:\n%s", want, view)
		}
	}
	if _, _, err := app.Svc.Balances.Add(ctx, acct.ID, "2026-05-21", "50000.00", "initial balance"); err != nil {
		t.Fatal(err)
	}
	app.Path = "/accounts/cash/balances/"
	view = app.View()
	for _, want := range []string{"> 2026-05-21 | HKD 50000.00", "initial balance", "  1) add balance"} {
		if !strings.Contains(view, want) {
			t.Fatalf("balances list missing %q:\n%s", want, view)
		}
	}
	app.Path = "/accounts/cash/balances/2026-05-21/"
	view = app.View()
	for _, want := range []string{"account : cash", "date    : 2026-05-21", "balance : HKD 50000.00", "> 1) edit balance", "2) delete balance"} {
		if !strings.Contains(view, want) {
			t.Fatalf("balance detail missing %q:\n%s", want, view)
		}
	}
	app.Path = "/accounts/cash/balances/2026-05-21/edit/"
	app.Form = map[string]string{"date": "2026-05-21", "balance": "50000.00", "notes": "initial balance"}
	view = app.View()
	for _, want := range []string{"> 1) date", "2026-05-21|", "2) balance", "50000.00", "3) notes", "[confirm]"} {
		if !strings.Contains(view, want) {
			t.Fatalf("balance edit missing %q:\n%s", want, view)
		}
	}
}

func assertOrdered(t *testing.T, body string, earlier string, later string) {
	t.Helper()
	i := strings.Index(body, earlier)
	j := strings.Index(body, later)
	if i == -1 || j == -1 || i >= j {
		t.Fatalf("expected %q before %q in:\n%s", earlier, later, body)
	}
}

func setCurrencyRate(t *testing.T, store *repo.Store, code string, amount int64, scale int) {
	t.Helper()
	ctx := context.Background()
	now := store.Clock().UTC().Format(time.RFC3339)
	var id int64
	if err := store.DB.QueryRowContext(ctx, "SELECT id FROM currencies WHERE code=?", code).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if _, err := store.DB.ExecContext(ctx, `INSERT INTO currency_rates(currency_id, rate_to_usd_amount, rate_to_usd_scale, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(currency_id) DO UPDATE SET rate_to_usd_amount=excluded.rate_to_usd_amount, rate_to_usd_scale=excluded.rate_to_usd_scale, updated_at=excluded.updated_at`,
		id, amount, scale, now); err != nil {
		t.Fatal(err)
	}
}

func TestListAndDetailNavigationMarkersStayInSync(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	if _, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Accounts.Create(ctx, "savings", "HKD", true, ""); err != nil {
		t.Fatal(err)
	}
	app.Path = "/accounts/list/"
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyDown})
	app = m.(App)
	if view := app.View(); !strings.Contains(view, "> savings") || strings.Contains(view, "> cash") {
		t.Fatalf("account list marker out of sync:\n%s", view)
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = m.(App)
	if app.Path != "/accounts/savings/" {
		t.Fatalf("enter should open selected account, got %s", app.Path)
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyDown})
	app = m.(App)
	if view := app.View(); !strings.Contains(view, "> 2) transactions (TODO)") || strings.Contains(view, "> 1) balances") {
		t.Fatalf("account detail marker out of sync:\n%s", view)
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = m.(App)
	if app.Path != "/accounts/savings/transactions/" {
		t.Fatalf("enter should run selected detail action, got %s", app.Path)
	}
}

func TestFormFocusBackspaceAndEscapeAreVisible(t *testing.T) {
	app, _ := testApp(t)
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: "/accounts/", Menu: 3}, navFrame{Path: "/accounts/create/", Menu: 0})
	if view := app.View(); !strings.Contains(view, "> 1) name     : |") || strings.Contains(view, "(type anything...)|") || strings.Contains(view, "currency : HKD|") {
		t.Fatalf("initial focused caret or unfocused field rendering wrong:\n%s", view)
	}
	for _, r := range "cash" {
		m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		app = m.(App)
	}
	if view := app.View(); !strings.Contains(view, "> 1) name") || !strings.Contains(view, "name     : cash|") {
		t.Fatalf("typed text or focus marker missing:\n%s", view)
	}
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	app = m.(App)
	if view := app.View(); !strings.Contains(view, "> 1) name") || !strings.Contains(view, "name     : cas|") {
		t.Fatalf("backspace did not update visible field:\n%s", view)
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyTab})
	app = m.(App)
	if view := app.View(); !strings.Contains(view, "> 2) currency") || !strings.Contains(view, "currency : HKD") {
		t.Fatalf("tab did not move visible form focus:\n%s", view)
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	app = m.(App)
	if app.Path != "/accounts/" || app.Error != "" {
		t.Fatalf("esc should discard form and return to account menu: path=%s error=%q", app.Path, app.Error)
	}
}

func TestTextCaretAndCursorMovement(t *testing.T) {
	app, _ := testApp(t)
	app.Path = "/accounts/create/"
	app.Field = 3
	for _, r := range "hi  there" {
		m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		app = m.(App)
	}
	if view := app.View(); !strings.Contains(view, "notes    : hi  there|") {
		t.Fatalf("notes caret should show trailing position after spaces/text:\n%s", view)
	}
	for i := 0; i < 7; i++ {
		m, _ := app.Update(tea.KeyMsg{Type: tea.KeyLeft})
		app = m.(App)
	}
	if view := app.View(); !strings.Contains(view, "notes    : hi|  there") {
		t.Fatalf("left should move text cursor inside notes:\n%s", view)
	}
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("!")})
	app = m.(App)
	if got := app.Form["notes"]; got != "hi!  there" {
		t.Fatalf("typing should insert at notes cursor, got %q", got)
	}
	if view := app.View(); !strings.Contains(view, "notes    : hi!|  there") {
		t.Fatalf("inserted text caret missing:\n%s", view)
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	app = m.(App)
	if got := app.Form["notes"]; got != "hi  there" {
		t.Fatalf("backspace should delete before notes cursor, got %q", got)
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRight})
	app = m.(App)
	if view := app.View(); !strings.Contains(view, "notes    : hi | there") {
		t.Fatalf("right should move notes cursor over a visible space:\n%s", view)
	}
}

func TestQuestionMarkTypesInNotesInsteadOfOpeningHelp(t *testing.T) {
	app, _ := testApp(t)
	app.Path = "/accounts/create/"
	app.Field = 3
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	app = m.(App)
	if app.Help {
		t.Fatal("? should not open help while account notes are focused")
	}
	if got := app.Form["notes"]; got != "?" {
		t.Fatalf("? should type into account notes, got %q", got)
	}
	view := app.View()
	if !strings.Contains(view, "notes    : ?|") || strings.Contains(view, "?       : help") {
		t.Fatalf("account notes view/footer wrong:\n%s", view)
	}

	app, _ = testApp(t)
	if _, _, err := app.Svc.Accounts.Create(context.Background(), "cash", "HKD", true, ""); err != nil {
		t.Fatal(err)
	}
	app.Path = "/accounts/cash/balances/add/"
	app.Form = map[string]string{"date": "2026-05-24"}
	app.Field = 2
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	app = m.(App)
	if app.Help {
		t.Fatal("? should not open help while balance notes are focused")
	}
	if got := app.Form["notes"]; got != "?" {
		t.Fatalf("? should type into balance notes, got %q", got)
	}
	view = app.View()
	if !strings.Contains(view, "notes    : ?|") || strings.Contains(view, "?       : help") {
		t.Fatalf("balance notes view/footer wrong:\n%s", view)
	}
}

func TestSlugCaretResetsAfterSanitization(t *testing.T) {
	app, _ := testApp(t)
	app.Path = "/accounts/create/"
	for _, r := range "Cash Account" {
		m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		app = m.(App)
	}
	for i := 0; i < 4; i++ {
		m, _ := app.Update(tea.KeyMsg{Type: tea.KeyLeft})
		app = m.(App)
	}
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" VIP ")})
	app = m.(App)
	if got := app.Form["name"]; got != "cash-acc-vip-ount" {
		t.Fatalf("name should sanitize inserted text, got %q", got)
	}
	if view := app.View(); !strings.Contains(view, "name     : cash-acc-vip-ount|") {
		t.Fatalf("sanitized slug cursor should reset to end:\n%s", view)
	}
}

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

func TestAccountNameInputSanitizesOnlyName(t *testing.T) {
	app, _ := testApp(t)
	app.Path = "/accounts/create/"
	for _, r := range "HSBC  One!!/🙂" {
		m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		app = m.(App)
	}
	if view := app.View(); !strings.Contains(view, "name     : hsbc-one") {
		t.Fatalf("typed account name was not sanitized:\n%s", view)
	}
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("set name=Bad Name!!")})
	app = m.(App)
	if view := app.View(); !strings.Contains(view, "name     : bad-name") {
		t.Fatalf("set name was not sanitized:\n%s", view)
	}
	app.Field = 3
	for _, r := range "Hello World! 管家" {
		m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		app = m.(App)
	}
	if got := app.Form["notes"]; got != "Hello World! 管家" {
		t.Fatalf("notes should preserve raw input, got %q", got)
	}
}

func TestSanitizedNameSubmitsAndEditRedirectsToNewSlug(t *testing.T) {
	app, store := testApp(t)
	app.Path = "/accounts/create/"
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("set name=My Cash!!")})
	app = m.(App)
	app.Field = 4
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = m.(App)
	if app.Path != "/accounts/list/" {
		t.Fatalf("sanitized create did not submit: %s\n%s", app.Path, app.View())
	}
	acct, err := store.Acct.GetByName(context.Background(), "my-cash")
	if err != nil {
		t.Fatal(err)
	}
	app.Path = "/accounts/my-cash/edit/"
	app.Form = map[string]string{"name": acct.Name, "currency": "HKD", "on-budget": "true"}
	app.Field = 0
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("set name=New CASH Account!!")})
	app = m.(App)
	app.Field = 4
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = m.(App)
	if app.Path != "/accounts/new-cash-account/" {
		t.Fatalf("edit did not redirect to sanitized slug: %s\n%s", app.Path, app.View())
	}
}

func TestBalanceListDetailEditDeleteNavigation(t *testing.T) {
	app, store := testApp(t)
	ctx := context.Background()
	acct, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, acct.ID, "2026-05-01", "100.00", "start"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, acct.ID, "2026-06-01", "150.00", "end"); err != nil {
		t.Fatal(err)
	}
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: "/accounts/cash/", Menu: 0}, navFrame{Path: "/accounts/cash/balances/", Menu: 0})
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyDown})
	app = m.(App)
	view := app.View()
	if !strings.Contains(view, "> 2026-05-01") || strings.Contains(view, "> 1) add balance") {
		t.Fatalf("balance list marker out of sync:\n%s", view)
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = m.(App)
	if app.Path != "/accounts/cash/balances/2026-05-01/" {
		t.Fatalf("enter should open selected balance, got %s", app.Path)
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyDown})
	app = m.(App)
	if view := app.View(); !strings.Contains(view, "> 2) delete balance") || strings.Contains(view, "> 1) edit balance") {
		t.Fatalf("balance detail marker out of sync:\n%s", view)
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = m.(App)
	if app.Path != "/accounts/cash/balances/" {
		t.Fatalf("delete should return to balances list, got %s", app.Path)
	}
	if _, err := store.Bal.GetByAccountDate(ctx, acct.ID, "2026-05-01"); err == nil {
		t.Fatal("selected balance should have been deleted")
	}
}

func TestAccountCreateValidationHistoryAndUndo(t *testing.T) {
	app, store := testApp(t)
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: "/accounts/", Menu: 3}, navFrame{Path: "/accounts/create/", Menu: 0})
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("set name=!!!")})
	app = m.(App)
	app.Field = 4
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = m.(App)
	if !strings.Contains(app.View(), "strict slug") {
		t.Fatalf("expected validation error:\n%s", app.View())
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("set name=cash")})
	app = m.(App)
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("set currency=HKD")})
	app = m.(App)
	app.Field = 4
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = m.(App)
	if app.Path != "/accounts/list/" || len(app.History) != 1 {
		t.Fatalf("bad post-create state: %+v", app)
	}
	if _, err := store.Acct.GetByName(context.Background(), "cash"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(app.View(), "history (ctrl-z to undo)") {
		t.Fatalf("missing visible history:\n%s", app.View())
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyCtrlZ})
	app = m.(App)
	if len(app.History) != 0 || app.Path != "/" {
		t.Fatalf("bad undo state: %+v", app)
	}
	if _, err := store.Acct.GetByName(context.Background(), "cash"); err == nil {
		t.Fatal("account still exists after model undo")
	}
}

func TestEscHelpAndBackup(t *testing.T) {
	app, _ := testApp(t)
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	app = m.(App)
	if !strings.Contains(app.View(), "ctrl-z") {
		t.Fatal(app.View())
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	app = m.(App)
	if app.Help {
		t.Fatal("help should close on esc")
	}
	app.Path = "/backup/"
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = m.(App)
	if !strings.Contains(app.View(), "db.2026-05-24-1200.sqlite") {
		t.Fatal(app.View())
	}
	app.Path = "/"
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	app = m.(App)
	if !strings.Contains(app.View(), "exit app? no") {
		t.Fatal(app.View())
	}
}

func TestManualAccountAndBalanceFlow(t *testing.T) {
	app, _ := testApp(t)
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: "/accounts/", Menu: 3}, navFrame{Path: "/accounts/create/", Menu: 0})
	for _, r := range "cash" {
		m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		app = m.(App)
	}
	app.Field = 4
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = m.(App)
	if app.Path != "/accounts/list/" {
		t.Fatalf("create path = %s view:\n%s", app.Path, app.View())
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = m.(App)
	if app.Path != "/accounts/cash/" {
		t.Fatalf("detail path = %s", app.Path)
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	app = m.(App)
	if app.Path != "/accounts/cash/transactions/" {
		t.Fatalf("transactions TODO path = %s", app.Path)
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	app = m.(App)
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	app = m.(App)
	if app.Path != "/accounts/cash/balances/" {
		t.Fatalf("balances path = %s", app.Path)
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = m.(App)
	if app.Path != "/accounts/cash/balances/add/" {
		t.Fatalf("empty balances enter should add balance, got %s", app.Path)
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyTab})
	app = m.(App)
	for _, r := range "123.45" {
		m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		app = m.(App)
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = m.(App)
	if app.Path != "/accounts/cash/balances/" || !strings.Contains(app.View(), "HKD 123.45") {
		t.Fatalf("balance flow failed path=%s view:\n%s", app.Path, app.View())
	}
}

func TestMenuCursorRestoresOnBackFromAccountList(t *testing.T) {
	app, _ := testApp(t)
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	app = m.(App)
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	app = m.(App)
	if app.Path != "/accounts/list/" {
		t.Fatalf("expected account list, got %s", app.Path)
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	app = m.(App)
	if app.Path != "/accounts/" {
		t.Fatalf("expected accounts menu, got %s", app.Path)
	}
	if view := app.View(); !strings.Contains(view, "> 2) list") || strings.Contains(view, "> 1) overview") {
		t.Fatalf("expected list cursor restored on accounts menu:\n%s", view)
	}
}

func TestMenuCursorRestoresOnBackFromAccountsToHome(t *testing.T) {
	app, _ := testApp(t)
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	app = m.(App)
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	app = m.(App)
	if app.Path != "/" {
		t.Fatalf("expected home, got %s", app.Path)
	}
	if view := app.View(); !strings.Contains(view, "> 1) accounts") {
		t.Fatalf("expected accounts cursor restored on home:\n%s", view)
	}
}

func TestMenuCursorRestoresOnBackFromAccountDetailChild(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	if _, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, ""); err != nil {
		t.Fatal(err)
	}
	app.Path = "/accounts/cash/"
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: "/accounts/", Menu: 0}, navFrame{Path: "/accounts/cash/", Menu: 0})
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyDown})
	app = m.(App)
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = m.(App)
	if app.Path != "/accounts/cash/transactions/" {
		t.Fatalf("expected transactions route, got %s", app.Path)
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	app = m.(App)
	if app.Path != "/accounts/cash/" {
		t.Fatalf("expected account detail, got %s", app.Path)
	}
	if view := app.View(); !strings.Contains(view, "> 2) transactions (TODO)") || strings.Contains(view, "> 1) balances") {
		t.Fatalf("expected transactions cursor restored on account detail:\n%s", view)
	}
}

func TestMenuCursorRestoresOnBackFromBalanceDetail(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	acct, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, acct.ID, "2026-05-01", "100.00", "start"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, acct.ID, "2026-06-01", "150.00", "end"); err != nil {
		t.Fatal(err)
	}
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: "/accounts/cash/", Menu: 0}, navFrame{Path: "/accounts/cash/balances/", Menu: 0})
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyDown})
	app = m.(App)
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = m.(App)
	if app.Path != "/accounts/cash/balances/2026-05-01/" {
		t.Fatalf("expected balance detail, got %s", app.Path)
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	app = m.(App)
	if app.Path != "/accounts/cash/balances/" {
		t.Fatalf("expected balances list, got %s", app.Path)
	}
	if view := app.View(); !strings.Contains(view, "> 2026-05-01") || strings.Contains(view, "> 2026-06-01") {
		t.Fatalf("expected selected balance cursor restored:\n%s", view)
	}
}

func TestMenuCursorDoesNotPersistAfterLeavingStack(t *testing.T) {
	app, _ := testApp(t)
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	app = m.(App)
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	app = m.(App)
	if app.Path != "/accounts/hidden/" {
		t.Fatalf("expected hidden accounts, got %s", app.Path)
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	app = m.(App)
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	app = m.(App)
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("7")})
	app = m.(App)
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	app = m.(App)
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	app = m.(App)
	if view := app.View(); !strings.Contains(view, "> 1) overview") || strings.Contains(view, "> 3) hidden") {
		t.Fatalf("expected default accounts cursor after re-entering popped screen:\n%s", view)
	}
}

func TestUndoResetsNavigationStack(t *testing.T) {
	app, store := testApp(t)
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	app = m.(App)
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	app = m.(App)
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: "/accounts/", Menu: 1}, navFrame{Path: "/accounts/create/", Menu: 0})
	app.Form = map[string]string{"name": "cash", "currency": "HKD", "on-budget": "true"}
	app.Field = 4
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = m.(App)
	if app.Path != "/accounts/list/" {
		t.Fatalf("expected account list after create, got %s", app.Path)
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyCtrlZ})
	app = m.(App)
	if app.Path != "/" {
		t.Fatalf("undo should return home, got %s", app.Path)
	}
	if _, err := store.Acct.GetByName(context.Background(), "cash"); err == nil {
		t.Fatal("account should be undone")
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	app = m.(App)
	if view := app.View(); !strings.Contains(view, "> 1) overview") || strings.Contains(view, "> 2) list") {
		t.Fatalf("accounts menu should start fresh after undo clears navigation stack:\n%s", view)
	}
}

func TestAccountDetailResetsCursorAfterLeaveAndReturn(t *testing.T) {
	app, store := testApp(t)
	ctx := context.Background()
	if _, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, ""); err != nil {
		t.Fatal(err)
	}
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	app = m.(App)
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	app = m.(App)
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = m.(App)
	for range 3 {
		m, _ = app.Update(tea.KeyMsg{Type: tea.KeyDown})
		app = m.(App)
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = m.(App)
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = m.(App)
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	app = m.(App)
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	app = m.(App)
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	app = m.(App)
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = m.(App)
	if app.Path != "/accounts/cash/" {
		t.Fatalf("expected account detail, got %s", app.Path)
	}
	if view := app.View(); !strings.Contains(view, "> 1) balances") || strings.Contains(view, "> 4) hide account") {
		t.Fatalf("expected balances cursor after re-entering account detail:\n%s", view)
	}
	if _, err := store.Acct.GetByName(ctx, "cash"); err != nil {
		t.Fatal(err)
	}
}

func TestAccountCreateRedirectRestoresListCursorOnBack(t *testing.T) {
	app, store := testApp(t)
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: "/accounts/", Menu: 3}, navFrame{Path: "/accounts/create/", Menu: 0})
	app.Form = map[string]string{"name": "cash", "currency": "HKD", "on-budget": "true"}
	app.Field = 4
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = m.(App)
	if app.Path != "/accounts/list/" {
		t.Fatalf("expected account list after create, got %s", app.Path)
	}
	if view := app.View(); !strings.Contains(view, "> cash") {
		t.Fatalf("expected created account selected in list:\n%s", view)
	}
	if _, err := store.Acct.GetByName(context.Background(), "cash"); err != nil {
		t.Fatal(err)
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	app = m.(App)
	if app.Path != "/accounts/" {
		t.Fatalf("expected accounts menu, got %s", app.Path)
	}
	if view := app.View(); !strings.Contains(view, "> 2) list") {
		t.Fatalf("expected list selected after backing out of post-create list:\n%s", view)
	}
}

func TestTabNavigatesMenus(t *testing.T) {
	app, _ := testApp(t)
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyTab})
	app = m.(App)
	if view := app.View(); !strings.Contains(view, "> 2) transactions (TODO)") || strings.Contains(view, "> 1) accounts") {
		t.Fatalf("tab should move home menu cursor down:\n%s", view)
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	app = m.(App)
	if view := app.View(); !strings.Contains(view, "> 1) accounts") || strings.Contains(view, "> 2) transactions (TODO)") {
		t.Fatalf("shift-tab should move home menu cursor up:\n%s", view)
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	app = m.(App)
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyTab})
	app = m.(App)
	if view := app.View(); !strings.Contains(view, "> 2) list") || strings.Contains(view, "> 1) overview") {
		t.Fatalf("tab should move accounts menu cursor down:\n%s", view)
	}
}

func TestTabDoesNotNavigateCurrencySelectOptions(t *testing.T) {
	app, _ := testApp(t)
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: "/accounts/", Menu: 3}, navFrame{Path: "/accounts/create/", Menu: 0})
	app.Field = 1
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyTab})
	app = m.(App)
	if app.Field != 2 {
		t.Fatalf("tab on currency select should move to next form field, got field %d", app.Field)
	}
	if view := app.View(); !strings.Contains(view, "> 3) on-budget") {
		t.Fatalf("expected on-budget field focused after tab:\n%s", view)
	}
}
