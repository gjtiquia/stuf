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
		"name        | balance",
		"TOTAL       |",
		"> 1) cash",
		"main cash",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("account list missing %q:\n%s", want, view)
		}
	}
	app.Path = "/accounts/hidden/"
	app.Menu = 0
	view = app.View()
	for _, want := range []string{"> filter : (type anything...)", "name        | balance", "> 1) investment", "brokerage"} {
		if !strings.Contains(view, want) {
			t.Fatalf("hidden accounts missing %q:\n%s", want, view)
		}
	}
}

func TestAccountListNoResultsShape(t *testing.T) {
	app, _ := testApp(t)
	app.Form["filter"] = "amex"
	app.Path = "/accounts/list/"
	view := app.View()
	if !strings.Contains(view, "> filter : amex") || !strings.Contains(view, "(no results)") {
		t.Fatalf("no-results shape missing:\n%s", view)
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
	for _, want := range []string{"hidden    : true", "> 1) show account", "  2) balances", "  3) transactions (TODO)", "  4) edit account"} {
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
	for _, want := range []string{"> 1) name", "2) currency : HKD", "3) on-budget: true", "4) notes", "[confirm]"} {
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
	for _, want := range []string{"> 1) date", "2) balance", "(type amount...)", "3) notes", "[confirm]"} {
		if !strings.Contains(view, want) {
			t.Fatalf("balance add form missing %q:\n%s", want, view)
		}
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
	for _, want := range []string{"> 1) date", "2) balance", "50000.00", "3) notes", "[confirm]"} {
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
	if view := app.View(); !strings.Contains(view, "> 2) savings") || strings.Contains(view, "> 1) cash") {
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
	app.Path = "/accounts/create/"
	for _, r := range "cash" {
		m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		app = m.(App)
	}
	if view := app.View(); !strings.Contains(view, "> 1) name") || !strings.Contains(view, "name     : cash") {
		t.Fatalf("typed text or focus marker missing:\n%s", view)
	}
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	app = m.(App)
	if view := app.View(); !strings.Contains(view, "> 1) name") || !strings.Contains(view, "name     : cas") {
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
	app.Path = "/accounts/cash/balances/"
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
	app.Path = "/accounts/create/"
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("set name=Bad Name")})
	app = m.(App)
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = m.(App)
	if !strings.Contains(app.View(), "strict slug") {
		t.Fatalf("expected validation error:\n%s", app.View())
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("set name=cash")})
	app = m.(App)
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("set currency=HKD")})
	app = m.(App)
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
	app.Path = "/accounts/create/"
	for _, r := range "cash" {
		m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		app = m.(App)
	}
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
	app.Path = "/accounts/cash/balances/"
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
