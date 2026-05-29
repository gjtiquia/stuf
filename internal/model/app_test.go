package model

import (
	"context"
	"errors"
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
		Accounts:          service.AccountService{Store: s, Accounts: s.Acct, Balances: s.Bal, Currency: s.Cur, Tags: s.Tag, History: h, AppCurrency: "HKD"},
		Balances:          service.BalanceService{Store: s, Accounts: s.Acct, Balances: s.Bal, History: h},
		Currency:          service.CurrencyService{Currencies: s.Cur},
		Tags:              service.TagService{Store: s, Tags: s.Tag, History: h},
		BudgetCategories:  service.BudgetCategoryService{Store: s, Categories: s.BudCat, Budgets: s.Bud, History: h},
		Budgets:           service.BudgetService{Store: s, Budgets: s.Bud, Categories: s.BudCat, Currency: s.Cur, Allocations: s.Alloc, History: h, AppCurrency: "HKD"},
		BudgetAllocations: service.BudgetAllocationService{Store: s, Budgets: s.Bud, Allocations: s.Alloc, History: h},
		Dashboard:         service.DashboardService{Accounts: s.Acct, Balances: s.Bal, Budgets: s.Bud, Allocations: s.Alloc, Currencies: s.Cur, AppCurrency: "HKD", Now: s.Clock},
		Reports:           service.ReportService{Accounts: s.Acct, Balances: s.Bal, Currencies: s.Cur, AppCurrency: "HKD", Now: s.Clock},
		History:           h,
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
	for _, want := range []string{"# stuf", "as-of       : none [!]", "total       : HKD 0.00", "on-budget net changes", "on-budget high to lows", "on-budget lows", "transactions (TODO)"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view missing %q:\n%s", want, view)
		}
	}
	for _, old := range []string{"net change to today", "recent months", "high to high trends", "low to low trends"} {
		if strings.Contains(view, old) {
			t.Fatalf("view should not contain old dashboard label %q:\n%s", old, view)
		}
	}
}

func TestDashboardRendersNetChangeFromBalanceSnapshots(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	acct, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, acct.ID, "2026-05-02", "100.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, acct.ID, "2026-05-10", "150.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, acct.ID, "2026-05-24", "130.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, acct.ID, "2026-06-01", "999.00", ""); err != nil {
		t.Fatal(err)
	}
	view := app.View()
	for _, want := range []string{
		"as-of       : 2026-05-24",
		"total       : HKD  130.00",
		"on-budget net changes",
		"2026-05     : HKD   30.00",
		"on-budget high to lows",
		"2026-05     : HKD ( 50.00)",
		"on-budget lows",
		"2026-05     : HKD  100.00",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("dashboard net change missing %q:\n%s", want, view)
		}
	}
}

func TestDashboardMoneyDecimalsAlignWithNegativeNetChange(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	acct, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, acct.ID, "2026-05-01", "36953.88", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, acct.ID, "2026-05-24", "13010.40", ""); err != nil {
		t.Fatal(err)
	}
	view := app.View()
	lines := linesContainingAny(view, []string{
		"as-of       :",
		"total       : HKD",
		"budgeted    : HKD",
		"2026-05     : HKD",
	})
	lines = moneyParts(lines)
	assertSamePrimaryPunctuationIndex(t, lines, ".")
	if !strings.Contains(view, "2026-05     : HKD (23,943.48)") {
		t.Fatalf("negative net change should use aligned accounting format:\n%s", view)
	}
}

func TestURLRendersImmediatelyAboveActions(t *testing.T) {
	app, _ := testApp(t)
	view := app.View()
	assertOrdered(t, view, "ppl owe you : HKD 0.00", "\n/\n\n> 1) accounts")
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	app = m.(App)
	view = app.View()
	assertOrdered(t, view, "on-budget lows", "\n/accounts/list/\n\n")
}

func TestAccountListRenderOrder(t *testing.T) {
	app, _ := testApp(t)
	app.Path = routeAccountList
	view := app.View()
	assertRenderOrder(t, view,
		"# stuf",
		"as-of       : none [!]",
		"on-budget   : HKD 0.00",
		"off-budget  : HKD 0.00",
		"total       : HKD 0.00",
		"on-budget net changes",
		"2026-05     : HKD 0.00",
		"on-budget high to lows",
		"2026-05     : HKD 0.00",
		"on-budget lows",
		"2026-05     : HKD 0.00",
		"/accounts/list/",
		"showing : non-hidden",
		"> filter : (type anything...)",
		"---",
	)
	assertNotContains(t, view, "you owe ppl")
	assertNotContains(t, view, "ppl owe you")
}

func TestBalanceListRenderOrder(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	acct, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, acct.ID, "2026-05-21", "50000.00", "initial balance"); err != nil {
		t.Fatal(err)
	}
	app.Path = "/accounts/cash/balances/list/"
	view := app.View()
	assertRenderOrder(t, view,
		"# stuf",
		"account   : cash",
		"balance   : HKD 50,000.00",
		"as of     : 2026-05-21",
		"net changes",
		"/accounts/cash/balances/list/",
		"  date       | balance       | notes",
		"> 2026-05-21 | HKD 50,000.00",
		"---",
	)
}

func TestBudgetHappyPathThroughUIAndServices(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	cash, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, cash.ID, "2026-05-01", "5000.00", "start"); err != nil {
		t.Fatal(err)
	}

	app = pressRunes(app, "3")
	if app.Path != routeBudgetList {
		t.Fatalf("budgets should open list route, got %s", app.Path)
	}
	assertViewContains(t, app.View(), "/budgets/list/", "on-budget : HKD 5,000.00", "budgeted  : HKD     0.00", "available : HKD 5,000.00")

	app = press(app, tea.KeyCtrlN)
	if app.Path != routeBudgetCreate {
		t.Fatalf("budget create route = %s", app.Path)
	}
	app.Form["name"] = "groceries"
	app.Form["category"] = "daily"
	app = press(app, tea.KeyCtrlS)
	if app.Path != routeBudgetList || app.Error != "" {
		t.Fatalf("budget create failed path=%s error=%q\n%s", app.Path, app.Error, app.View())
	}
	assertViewContains(t, app.View(), "daily", "> groceries")

	app = press(app, tea.KeyEnter)
	if app.Path != budgetPath("groceries") {
		t.Fatalf("budget detail route = %s", app.Path)
	}
	app = press(app, tea.KeyEnter)
	if app.Path != budgetAllocationListPath("groceries") {
		t.Fatalf("allocation list route = %s", app.Path)
	}
	app = press(app, tea.KeyCtrlN)
	app.Form["amount"] = "3000.00"
	app.Form["date"] = "2026-05-02"
	app = press(app, tea.KeyCtrlS)
	if app.Path != budgetAllocationListPath("groceries") || app.Error != "" {
		t.Fatalf("allocation add failed path=%s error=%q\n%s", app.Path, app.Error, app.View())
	}

	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: routeBudgetList, Menu: 0})
	assertViewContains(t, app.View(), "budgeted  : HKD 3,000.00", "available : HKD 2,000.00")
	if _, _, err := app.Svc.Balances.Add(ctx, cash.ID, "2026-05-24", "2000.00", "drop"); err != nil {
		t.Fatal(err)
	}
	assertViewContains(t, app.View(), "available : HKD (1,000.00)")
	budget, err := app.Svc.Budgets.GetByName(ctx, "groceries")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.BudgetAllocations.Add(ctx, budget.ID, service.AllocationActionRemoveMoney, "1000.00", "2026-05-24", "rebalance"); err != nil {
		t.Fatal(err)
	}
	assertViewContains(t, app.View(), "budgeted  : HKD 2,000.00", "available : HKD     0.00")
}

func TestBudgetCategoryFieldSelectsExistingCategory(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	if _, _, err := app.Svc.BudgetCategories.Create(ctx, "daily", "day to day"); err != nil {
		t.Fatal(err)
	}
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: routeBudgetList, Menu: 0}, navFrame{Path: routeBudgetCreate, Menu: 0})
	app.Form = map[string]string{"name": "groceries", "currency": "HKD", "category": "uncategorized"}
	app.Field = 2
	app = pressRunes(app, "dai")
	view := app.View()
	assertViewContains(t, view, "> 3) category", "> filter  : dai", "> daily")
	app = press(app, tea.KeyEnter)
	if app.Form["category"] != "daily" {
		t.Fatalf("category selection = %q", app.Form["category"])
	}
	app = press(app, tea.KeyCtrlS)
	if app.Path != routeBudgetList || app.Error != "" {
		t.Fatalf("budget create with selected category failed path=%s error=%q\n%s", app.Path, app.Error, app.View())
	}
	b, err := app.Svc.Budgets.GetByName(ctx, "groceries")
	if err != nil {
		t.Fatal(err)
	}
	if b.CategoryName != "daily" {
		t.Fatalf("budget category = %q", b.CategoryName)
	}
}

func TestBudgetCategoryFieldCreatesCategoryInline(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: routeBudgetList, Menu: 0}, navFrame{Path: routeBudgetCreate, Menu: 0})
	app.Form = map[string]string{"name": "japan-trip", "currency": "HKD", "category": "uncategorized"}
	app.Field = 2
	app = pressRunes(app, "travel")
	view := app.View()
	assertViewContains(t, view, "> filter  : travel", `(create new "travel")`)
	app = press(app, tea.KeyEnter)
	if app.Form["category"] != "travel" {
		t.Fatalf("category create selection = %q", app.Form["category"])
	}
	app = press(app, tea.KeyCtrlS)
	if app.Path != routeBudgetList || app.Error != "" {
		t.Fatalf("budget create with inline category failed path=%s error=%q\n%s", app.Path, app.Error, app.View())
	}
	if _, err := app.Svc.BudgetCategories.GetByName(ctx, "travel"); err != nil {
		t.Fatal(err)
	}
	b, err := app.Svc.Budgets.GetByName(ctx, "japan-trip")
	if err != nil {
		t.Fatal(err)
	}
	if b.CategoryName != "travel" {
		t.Fatalf("budget category = %q", b.CategoryName)
	}
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
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyCtrlH})
	app = m.(App)
	view = app.View()
	if !strings.Contains(view, "showing : hidden-only") {
		t.Fatalf("ctrl+h did not cycle account list visibility:\n%s", view)
	}
}

func TestReportsMonthlyNavigationAndRendering(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	cash, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	card, _, err := app.Svc.Accounts.Create(ctx, "credit-card", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	investment, _, err := app.Svc.Accounts.Create(ctx, "investment", "HKD", false, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, add := range []struct {
		accountID int64
		date      string
		amount    string
	}{
		{cash.ID, "2026-05-01", "1000.00"},
		{cash.ID, "2026-05-10", "1500.00"},
		{cash.ID, "2026-05-24", "800.00"},
		{card.ID, "2026-05-01", "0.00"},
		{card.ID, "2026-05-20", "-300.00"},
		{investment.ID, "2026-05-01", "9999.00"},
		{investment.ID, "2026-05-24", "19999.00"},
	} {
		if _, _, err := app.Svc.Balances.Add(ctx, add.accountID, add.date, add.amount, ""); err != nil {
			t.Fatal(err)
		}
	}

	app = pressRunes(app, "5")
	if app.Path != routeReports {
		t.Fatalf("reports path = %s", app.Path)
	}
	assertViewContains(t, app.View(), "/reports/", "on-budget net change", "current month", "monthly", "rolling 3 months (TODO)", "HKD (500.00)")

	app = press(app, tea.KeyEnter)
	if app.Path != routeReportsMonthly {
		t.Fatalf("monthly reports path = %s", app.Path)
	}
	view := app.View()
	assertViewContains(t, view, "/reports/monthly/", "month", "start", "end", "change", "high-to-low", "2026-05", "HKD (500.00)")
	assertNotContains(t, view, "investment")

	app = press(app, tea.KeyEnter)
	if app.Path != reportMonthlyDetailPath("2026-05") {
		t.Fatalf("monthly detail path = %s", app.Path)
	}
	view = app.View()
	assertViewContains(t, view, "period      : 2026-05", "coverage    : 2026-05-01 -> 2026-05-24", "on-budget accounts", "cash", "credit-card", "HKD (300.00)")
	assertNotContains(t, view, "investment")

	app = press(app, tea.KeyEnter)
	if app.Path != reportMonthlyAccountPath("2026-05", "cash") {
		t.Fatalf("monthly account detail path = %s", app.Path)
	}
	view = app.View()
	assertViewContains(t, view, "account     : cash", "period      : 2026-05", "snapshots", "date", "balance", "kind", "notes", "2026-05-01", "2026-05-10", "2026-05-24", "snapshot")
	app = press(app, tea.KeyEsc)

	app = pressRunes(app, "credit")
	view = app.View()
	assertViewContains(t, view, "credit-card")
	assertNotContains(t, view, "cash")

	app = press(app, tea.KeyRight)
	if app.Path != reportMonthlyDetailPath("2026-06") {
		t.Fatalf("next month path = %s", app.Path)
	}
	app = press(app, tea.KeyLeft)
	if app.Path != reportMonthlyDetailPath("2026-05") {
		t.Fatalf("previous month path = %s", app.Path)
	}
}

func TestAccountsActionOpensList(t *testing.T) {
	app, _ := testApp(t)
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	app = m.(App)
	view := app.View()
	for _, want := range []string{
		"total       : HKD 0.00",
		"/accounts/list/",
		"showing : non-hidden",
		"> filter : (type anything...)",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("accounts list missing %q:\n%s", want, view)
		}
	}
}

func TestAccountsListSummaryNavigation(t *testing.T) {
	app, _ := testApp(t)
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	app = m.(App)
	if app.Path != "/accounts/list/" {
		t.Fatalf("expected /accounts/list/, got %s", app.Path)
	}
	view := app.View()
	for _, want := range []string{
		"/accounts/list/",
		"total       : HKD 0.00",
		"on-budget   : HKD 0.00",
		"off-budget  : HKD 0.00",
		"showing : non-hidden",
		"> filter : (type anything...)",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("list summary empty state missing %q:\n%s", want, view)
		}
	}
	assertOrdered(t, view, "off-budget  : HKD 0.00", "on-budget net changes")
	assertOrdered(t, view, "total       : HKD 0.00", "on-budget net changes")
	assertOrdered(t, view, "\n/accounts/list/\n\nshowing : non-hidden", "\n> filter : (type anything...)")
}

func TestAccountsListSummaryTotals(t *testing.T) {
	app, store := testApp(t)
	seedStandardAccounts(t, app, store)
	app.Path = routeAccountList
	view := app.View()
	for _, want := range []string{
		"total       : HKD  575.00",
		"on-budget   : HKD  600.00",
		"off-budget  : HKD ( 25.00)",
		"showing : non-hidden",
		"> filter : (type anything...)",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("list summary totals missing %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "old-account") {
		t.Fatalf("hidden account should be excluded from list summary:\n%s", view)
	}
}

func TestAccountListFilteredSummaryCountsOnlyMatchedMoney(t *testing.T) {
	app, store := testApp(t)
	ctx := context.Background()
	setCurrencyRate(t, store, "HKD", 1, 0)
	parent, _, err := app.Svc.Accounts.CreateWithTags(ctx, "household", "HKD", true, "", []string{"family"})
	if err != nil {
		t.Fatal(err)
	}
	child, _, err := app.Svc.Accounts.CreateChildWithTags(ctx, parent.ID, "household-cash", "HKD", "", []string{"wallet"})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, parent.ID, "2026-05-24", "500.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, child.ID, "2026-05-24", "100.00", ""); err != nil {
		t.Fatal(err)
	}
	app.Path = routeAccountList
	app.Form[formKeyFilter] = "tag:wallet"
	view := app.View()
	assertViewContains(t, view, "> household", "household-cash", "wallet")
	assertRenderOrder(t, view, "on-budget   : HKD 100.00", "off-budget  : HKD   0.00", "total       : HKD 100.00", "/accounts/list/", "showing : non-hidden")
	if strings.Contains(view, "filtered total") {
		t.Fatalf("filtered account list should not render duplicate summary labels:\n%s", view)
	}
	if !strings.Contains(view, "total       : HKD 100.00") && !strings.Contains(view, "total       : HKD  100.00") {
		t.Fatalf("child-only tag filter should count only child money:\n%s", view)
	}
}

func TestAccountListNegativeTagFilterExcludesTaggedAccounts(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	if _, _, err := app.Svc.Accounts.CreateWithTags(ctx, "cash", "HKD", true, "", []string{"wallet"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Accounts.CreateWithTags(ctx, "savings", "HKD", true, "", []string{"bank"}); err != nil {
		t.Fatal(err)
	}
	app.Path = routeAccountList
	app.Form[formKeyFilter] = "-tag:wallet"

	view := app.View()
	assertViewContains(t, view, "savings", "bank")
	if strings.Contains(view, "> cash") || strings.Contains(view, "  cash") {
		t.Fatalf("negative tag filter should exclude tagged account:\n%s", view)
	}
}

func TestChildAccountListFiltersByEffectiveTags(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	parent, _, err := app.Svc.Accounts.CreateWithTags(ctx, "household", "HKD", true, "", []string{"family"})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Accounts.CreateChildWithTags(ctx, parent.ID, "household-cash", "HKD", "", []string{"wallet"}); err != nil {
		t.Fatal(err)
	}
	app.Path = accountChildrenListPath(parent.Name)
	app.Form[formKeyFilter] = "tag:family"

	view := app.View()
	assertViewContains(t, view, "household-cash", "family")
}

func TestAccountTreeHappyPathWithParentBalance(t *testing.T) {
	app, store := testApp(t)
	ctx := context.Background()
	setCurrencyRate(t, store, "HKD", 1, 0)
	setCurrencyRate(t, store, "USD", 10, 0)
	parent, _, err := app.Svc.Accounts.Create(ctx, "investment", "HKD", false, "broker total")
	if err != nil {
		t.Fatal(err)
	}
	usd, _, err := app.Svc.Accounts.CreateChild(ctx, parent.ID, "investment-usd", "USD", "")
	if err != nil {
		t.Fatal(err)
	}
	hkd, _, err := app.Svc.Accounts.CreateChild(ctx, parent.ID, "investment-hkd", "HKD", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, parent.ID, "2026-05-24", "500000.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, usd.ID, "2026-05-24", "32000.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, hkd.ID, "2026-05-24", "100000.00", ""); err != nil {
		t.Fatal(err)
	}
	app.Path = routeAccountList
	view := app.View()
	assertViewContains(t, view,
		"off-budget  : HKD 500,000.00",
		"investment",
		"investment-usd",
		"investment-hkd",
		"remaining",
		"HKD  80,000.00",
	)
	if strings.Contains(view, "HKD 920,000.00") {
		t.Fatalf("account list should not double count parent and children:\n%s", view)
	}
	app.Path = "/accounts/investment/"
	view = app.View()
	assertViewContains(t, view,
		"balance   : HKD 500,000.00",
		"children  : HKD 420,000.00",
		"remaining : HKD  80,000.00",
	)
}

func TestAccountTreeHappyPathWithoutParentBalance(t *testing.T) {
	app, store := testApp(t)
	ctx := context.Background()
	setCurrencyRate(t, store, "HKD", 1, 0)
	setCurrencyRate(t, store, "USD", 10, 0)
	parent, _, err := app.Svc.Accounts.Create(ctx, "hsbc-one", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	hkd, _, err := app.Svc.Accounts.CreateChild(ctx, parent.ID, "hsbc-hkd", "HKD", "")
	if err != nil {
		t.Fatal(err)
	}
	usd, _, err := app.Svc.Accounts.CreateChild(ctx, parent.ID, "hsbc-usd", "USD", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, hkd.ID, "2026-05-21", "35000.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, usd.ID, "2026-05-24", "1000.00", ""); err != nil {
		t.Fatal(err)
	}
	app.Path = routeAccountList
	view := app.View()
	assertViewContains(t, view,
		"on-budget   : HKD 45,000.00",
		"hsbc-one",
		"hsbc-hkd",
		"hsbc-usd",
	)
	if strings.Contains(view, "remaining") {
		t.Fatalf("no-own-balance parent should not show remaining row in account list:\n%s", view)
	}
	app.Path = "/accounts/hsbc-one/"
	view = app.View()
	assertViewContains(t, view,
		"balance   : HKD 45,000.00",
		"children  : HKD 45,000.00",
		"remaining : HKD",
		"0.00",
		"as of     : 2026-05-24",
	)
}

func TestChildAccountCreateEditAndDeleteFlow(t *testing.T) {
	app, store := testApp(t)
	ctx := context.Background()
	parent, _, err := app.Svc.Accounts.Create(ctx, "investment", "HKD", false, "")
	if err != nil {
		t.Fatal(err)
	}
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: "/accounts/investment/", Menu: 1})
	app = press(app, tea.KeyEnter)
	if app.Path != "/accounts/investment/children/list/" {
		t.Fatalf("child accounts action should open child list, got %s", app.Path)
	}
	app = press(app, tea.KeyCtrlN)
	if app.Path != "/accounts/investment/children/create/" {
		t.Fatalf("ctrl+n on child list should open child create, got %s", app.Path)
	}
	app.Form["name"] = "investment-usd"
	app.Form["currency"] = "USD"
	app = press(app, tea.KeyCtrlS)
	if app.Path != "/accounts/investment/children/list/" {
		t.Fatalf("child create should return to child list, got %s", app.Path)
	}
	child, err := store.Acct.GetByName(ctx, "investment-usd")
	if err != nil {
		t.Fatal(err)
	}
	if child.ParentID == nil || *child.ParentID != parent.ID || child.OnBudget {
		t.Fatalf("child account inheritance wrong: %+v", child)
	}
	view := app.View()
	assertViewContains(t, view, "parent", "investment", "> investment-usd")
	if strings.Contains(view, "\n  remaining") || strings.Contains(view, "\n> remaining") {
		t.Fatalf("child list table should not show remaining row:\n%s", view)
	}
	app.Path = "/accounts/investment-usd/edit/"
	view = app.View()
	if strings.Contains(view, "on-budget") {
		t.Fatalf("child edit should omit on-budget field:\n%s", view)
	}
	app.Path = "/accounts/investment-usd/"
	view = app.View()
	assertViewContains(t, view, "hide account", "delete account")
	app.Menu = 5
	app = press(app, tea.KeyEnter)
	if _, err := store.Acct.GetByName(ctx, "investment-usd"); err == nil {
		t.Fatal("empty child account should be deleted")
	}
	app = press(app, tea.KeyCtrlZ)
	if _, err := store.Acct.GetByName(ctx, "investment-usd"); err != nil {
		t.Fatalf("undo should restore deleted child: %v", err)
	}
}

func TestEmptyChildAccountListIsFilterableListNotActionMenu(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	if _, _, err := app.Svc.Accounts.Create(ctx, "investment", "HKD", false, ""); err != nil {
		t.Fatal(err)
	}
	app = appWithNav(app,
		navFrame{Path: "/", Menu: 0},
		navFrame{Path: "/accounts/investment/", Menu: 1},
		navFrame{Path: "/accounts/investment/children/list/", Menu: 0},
	)
	view := app.View()
	assertViewContains(t, view, "parent", "investment", "> filter : (type anything...)", "  name | balance | notes", "  (no child accounts yet)", "ctrl+n        : new")
	if strings.Contains(view, "1) add child account") || strings.Contains(view, "ctrl+d") {
		t.Fatalf("child list should not render an add/delete action menu:\n%s", view)
	}

	app = press(app, tea.KeyEnter)
	if app.Path != "/accounts/investment/children/list/" || app.Form["filter"] != "" {
		t.Fatalf("enter on empty child list should stay put without filtering, path=%s filter=%q", app.Path, app.Form["filter"])
	}
	app = press(app, tea.KeyRight)
	if app.Path != "/accounts/investment/children/list/" || app.Form["filter"] != "" {
		t.Fatalf("right on empty child list should stay put without filtering, path=%s filter=%q", app.Path, app.Form["filter"])
	}
	app = press(app, tea.KeyCtrlE)
	if app.Path != "/accounts/investment/children/list/" || app.Form["filter"] != "" {
		t.Fatalf("ctrl+e on empty child list should stay put without filtering, path=%s filter=%q", app.Path, app.Form["filter"])
	}
	app = press(app, tea.KeyCtrlN)
	if app.Path != "/accounts/investment/children/create/" {
		t.Fatalf("ctrl+n on empty child list should open child create, got %s", app.Path)
	}
}

func TestChildAccountListNavigationFilteringAndReturn(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	parent, _, err := app.Svc.Accounts.Create(ctx, "investment", "HKD", false, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Accounts.CreateChild(ctx, parent.ID, "investment-hkd", "HKD", "local cash"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Accounts.CreateChild(ctx, parent.ID, "investment-usd", "USD", "dollars"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Accounts.CreateChild(ctx, parent.ID, "investment-jk", "HKD", "vim letters"); err != nil {
		t.Fatal(err)
	}
	app = appWithNav(app,
		navFrame{Path: "/", Menu: 0},
		navFrame{Path: "/accounts/investment/", Menu: 1},
		navFrame{Path: "/accounts/investment/children/list/", Menu: 0},
	)
	view := app.View()
	assertViewContains(t, view, "> investment-hkd", "  investment-usd")
	if strings.Contains(view, "\n  remaining") || strings.Contains(view, "\n> remaining") || strings.Contains(view, "1) add child account") {
		t.Fatalf("child list should show only child rows and no action menu:\n%s", view)
	}

	for _, key := range []tea.KeyType{tea.KeyDown, tea.KeyUp, tea.KeyTab, tea.KeyShiftTab} {
		app = appWithNav(app,
			navFrame{Path: "/", Menu: 0},
			navFrame{Path: "/accounts/investment/", Menu: 1},
			navFrame{Path: "/accounts/investment/children/list/", Menu: 0},
		)
		app = press(app, key)
		view = app.View()
		if !strings.Contains(view, "> investment-jk") && !strings.Contains(view, "> investment-usd") {
			t.Fatalf("key %v should move away from the first child:\n%s", key, view)
		}
	}
	app = appWithNav(app,
		navFrame{Path: "/", Menu: 0},
		navFrame{Path: "/accounts/investment/", Menu: 1},
		navFrame{Path: "/accounts/investment/children/list/", Menu: 0},
	)
	app.Form = map[string]string{}
	app = pressRunes(app, "j")
	view = app.View()
	assertViewContains(t, view, "> filter : j", "> investment-jk")
	if strings.Contains(view, "> investment-usd") {
		t.Fatalf("j should type into filter, not navigate to usd:\n%s", view)
	}
	app = pressRunes(app, "k")
	view = app.View()
	assertViewContains(t, view, "> filter : jk", "> investment-jk")
	if strings.Contains(view, "> investment-hkd") || strings.Contains(view, "> investment-usd") {
		t.Fatalf("k should continue filtering, not navigate:\n%s", view)
	}

	app = appWithNav(app,
		navFrame{Path: "/", Menu: 0},
		navFrame{Path: "/accounts/investment/", Menu: 1},
		navFrame{Path: "/accounts/investment/children/list/", Menu: 0},
	)
	app.Form = map[string]string{}
	app = pressRunes(app, "usd")
	view = app.View()
	assertViewContains(t, view, "> filter : usd", "> investment-usd")
	if strings.Contains(view, "investment-hkd") || strings.Contains(view, "investment-jk") {
		t.Fatalf("name filter should hide non-matching child:\n%s", view)
	}
	app = press(app, tea.KeyBackspace)
	assertViewContains(t, app.View(), "> filter : us", "> investment-usd")
	app = pressRunes(app, "zz")
	view = app.View()
	assertViewContains(t, view, "> filter : uszz", "  (no results)")
	if strings.Contains(view, "> investment") {
		t.Fatalf("no-match filter should not leave a stale selected child:\n%s", view)
	}

	app = appWithNav(app,
		navFrame{Path: "/", Menu: 0},
		navFrame{Path: "/accounts/investment/", Menu: 1},
		navFrame{Path: "/accounts/investment/children/list/", Menu: 0},
	)
	app.Form = map[string]string{}
	app = pressRunes(app, "dollars")
	view = app.View()
	assertViewContains(t, view, "> filter : dollars", "> investment-usd")
	if strings.Contains(view, "investment-hkd") || strings.Contains(view, "investment-jk") {
		t.Fatalf("notes filter should hide non-matching child:\n%s", view)
	}
	app = pressRunes(app, "hl")
	if app.Path != "/accounts/investment/children/list/" || app.Form["filter"] != "dollarshl" {
		t.Fatalf("h/l should type into child list filter, path=%s filter=%q", app.Path, app.Form["filter"])
	}

	app.Form = map[string]string{}
	app = appWithNav(app,
		navFrame{Path: "/", Menu: 0},
		navFrame{Path: "/accounts/investment/", Menu: 1},
		navFrame{Path: "/accounts/investment/children/list/", Menu: 2},
	)
	app = press(app, tea.KeyEnter)
	if app.Path != "/accounts/investment-usd/" {
		t.Fatalf("enter should open selected child detail, got %s", app.Path)
	}

	app.Form = map[string]string{}
	app = appWithNav(app,
		navFrame{Path: "/", Menu: 0},
		navFrame{Path: "/accounts/investment/", Menu: 1},
		navFrame{Path: "/accounts/investment/children/list/", Menu: 2},
	)
	app.Form["filter"] = "usd"
	app = press(app, tea.KeyCtrlE)
	if app.Path != "/accounts/investment-usd/edit/" {
		t.Fatalf("ctrl+e should edit selected child, got %s", app.Path)
	}
	app.Form["name"] = "investment-usd-main"
	app = press(app, tea.KeyCtrlS)
	if app.Path != "/accounts/investment/children/list/" {
		t.Fatalf("child edit should return to child list, got %s", app.Path)
	}
	assertViewContains(t, app.View(), "> filter : usd", "> investment-usd-main")
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
		"total       : HKD 0.00",
		"showing : non-hidden",
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
	app.AccountVisible = accountVisibilityHiddenOnly
	app.Menu = 0
	view = app.View()
	for _, want := range []string{"showing : hidden-only", "> filter : (type anything...)", "| balance", "> investment", "brokerage"} {
		if !strings.Contains(view, want) {
			t.Fatalf("hidden accounts missing %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "> 1) investment") {
		t.Fatalf("hidden account rows should not render menu numbers:\n%s", view)
	}
}

func TestAccountListVisibilityCyclesAndResets(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	if _, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, "main cash"); err != nil {
		t.Fatal(err)
	}
	hidden, _, err := app.Svc.Accounts.Create(ctx, "old-account", "HKD", true, "closed")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Accounts.SetHidden(ctx, hidden.ID, true); err != nil {
		t.Fatal(err)
	}
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: "/accounts/list/", Menu: 7})

	view := app.View()
	assertViewContains(t, view, "showing : non-hidden", "> cash")
	if strings.Contains(view, "old-account") {
		t.Fatalf("default account list should exclude hidden accounts:\n%s", view)
	}

	app = press(app, tea.KeyCtrlH)
	view = app.View()
	assertViewContains(t, view, "showing : hidden-only", "> old-account")
	if app.Menu != 0 || strings.Contains(view, "cash |") {
		t.Fatalf("hidden-only should clamp cursor and exclude visible accounts: menu=%d\n%s", app.Menu, view)
	}

	app = press(app, tea.KeyCtrlH)
	view = app.View()
	assertViewContains(t, view, "showing : all", "hidden", "old-account", "true", "cash")

	app = press(app, tea.KeyCtrlH)
	assertViewContains(t, app.View(), "showing : non-hidden")

	app = pressRunes(app, "hl")
	if app.AccountVisible != accountVisibilityNonHidden || !strings.Contains(app.View(), "> filter : hl") {
		t.Fatalf("plain h/l should type into the filter without cycling visibility:\n%s", app.View())
	}

	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: "/accounts/list/", Menu: 0})
	app = press(app, tea.KeyCtrlH)
	app = press(app, tea.KeyEsc)
	app = pressRunes(app, "1")
	assertViewContains(t, app.View(), "showing : non-hidden")
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

func TestAccountListFilterAcceptsJKKeys(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	if _, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Accounts.Create(ctx, "bank-jk", "HKD", true, ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Accounts.Create(ctx, "savings", "HKD", true, ""); err != nil {
		t.Fatal(err)
	}
	app.Path = "/accounts/list/"
	app.Menu = 2

	for _, r := range "jk" {
		m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		app = m.(App)
	}
	view := app.View()
	if !strings.Contains(view, "> filter : jk") || strings.Contains(view, "> filter : jk|") {
		t.Fatalf("typed j/k should append to filter:\n%s", view)
	}
	if strings.Contains(view, "> cash") || strings.Contains(view, "> savings") {
		t.Fatalf("j/k should filter, not navigate:\n%s", view)
	}
	if !strings.Contains(view, "> bank-jk") {
		t.Fatalf("filter jk should match bank-jk:\n%s", view)
	}
}

func TestAccountListArrowKeysStillNavigate(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	if _, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Accounts.Create(ctx, "savings", "HKD", true, ""); err != nil {
		t.Fatal(err)
	}
	app.Path = "/accounts/list/"
	app.Menu = 0

	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyDown})
	app = m.(App)
	if view := app.View(); !strings.Contains(view, "> savings") || strings.Contains(view, "> cash") {
		t.Fatalf("down should move account list selection:\n%s", view)
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyUp})
	app = m.(App)
	if view := app.View(); !strings.Contains(view, "> cash") || strings.Contains(view, "> savings") {
		t.Fatalf("up should move account list selection back:\n%s", view)
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

func TestAccountListSectionsShareColumnLayout(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	cash, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	loan, _, err := app.Svc.Accounts.Create(ctx, "very-long-student-loan", "HKD", false, "negative until fully paid")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, cash.ID, "2026-05-25", "10.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, loan.ID, "2026-05-25", "-1234567.89", ""); err != nil {
		t.Fatal(err)
	}
	app.Path = routeAccountList
	view := app.View()

	var tableLines []string
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(line, " | ") && (strings.Contains(line, "name") || strings.Contains(line, "TOTAL") || strings.Contains(line, "cash") || strings.Contains(line, "student-loan")) {
			tableLines = append(tableLines, line)
		}
	}
	if len(tableLines) < 6 {
		t.Fatalf("expected both account sections to render table rows, got %d lines:\n%s", len(tableLines), view)
	}
	wantBalanceIdx := balanceColumnIndex(tableLines[0])
	wantNotesIdx := notesColumnIndex(tableLines[0])
	for i, line := range tableLines[1:] {
		if got := balanceColumnIndex(line); got != wantBalanceIdx {
			t.Fatalf("balance column misaligned on line %d: want index %d, got %d\n%s", i+1, wantBalanceIdx, got, view)
		}
		if got := notesColumnIndex(line); got != wantNotesIdx {
			t.Fatalf("notes column misaligned on line %d: want index %d, got %d\n%s", i+1, wantNotesIdx, got, view)
		}
	}
}

func TestBalanceListColumnsAlignWithLongBalances(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	acct, _, err := app.Svc.Accounts.Create(ctx, "hsbc-109", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, acct.ID, "2026-05-25", "13010.40", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, acct.ID, "2026-05-01", "17800.20", "(also fake)"); err != nil {
		t.Fatal(err)
	}
	app.Path = "/accounts/hsbc-109/balances/list/"
	view := app.View()

	var tableLines []string
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(line, " | ") && (strings.Contains(line, "date") || strings.Contains(line, "2026-05-25") || strings.Contains(line, "2026-05-01")) {
			tableLines = append(tableLines, line)
		}
	}
	if len(tableLines) != 3 {
		t.Fatalf("expected balance header and two rows, got %d lines:\n%s", len(tableLines), view)
	}
	wantNotesIdx := notesColumnIndex(tableLines[0])
	for i, line := range tableLines[1:] {
		if got := notesColumnIndex(line); got != wantNotesIdx {
			t.Fatalf("notes column misaligned on line %d: want index %d, got %d\n%s", i+1, wantNotesIdx, got, view)
		}
	}
}

func balanceColumnIndex(line string) int {
	return strings.Index(line, " |")
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
	seedStandardAccounts(t, app, store)
	app.Path = routeAccountList
	view := app.View()
	for _, want := range []string{
		"| HKD  600.00",
		"> cash",
		"usd-savings",
		"HKD  500.00  (USD 50.00)",
		"| HKD ( 25.00)",
		"student-loan",
		"negative until fully paid",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("account list totals/conversion missing %q:\n%s", want, view)
		}
	}
}

func TestAccountListMoneyColumnsAlignPunctuation(t *testing.T) {
	app, store := testApp(t)
	seedStandardAccounts(t, app, store)
	app.Path = routeAccountList
	view := app.View()
	lines := linesContaining(view, []string{"TOTAL", "cash", "usd-savings", "student-loan"})
	assertSamePrimaryPunctuationIndex(t, lines, ".")
	assertSamePrimaryPunctuationIndexIfPresent(t, lines, ",")
}

func TestBalanceListMoneyColumnAlignsPositiveNegativeAndLargeValues(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	acct, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range []struct {
		date   string
		amount string
	}{
		{"2026-05-21", "10.00"},
		{"2026-05-22", "1000.00"},
		{"2026-05-23", "1234567.89"},
		{"2026-05-24", "-1234.56"},
	} {
		if _, _, err := app.Svc.Balances.Add(ctx, acct.ID, entry.date, entry.amount, ""); err != nil {
			t.Fatal(err)
		}
	}
	app.Path = "/accounts/cash/balances/list/"
	view := app.View()
	lines := linesContaining(view, []string{"2026-05-21", "2026-05-22", "2026-05-23", "2026-05-24"})
	assertSamePrimaryPunctuationIndex(t, lines, ".")
	assertSamePrimaryPunctuationIndex(t, lines, ",")
	if !strings.Contains(view, "HKD (    1,234.56)") {
		t.Fatalf("negative balance should reserve accounting parens without shifting decimals:\n%s", view)
	}
}

func TestNegativeBalanceDisplayUsesBracketsEditUsesMinus(t *testing.T) {
	app, store := testApp(t)
	seedStandardAccounts(t, app, store)
	app.Path = routeAccountList
	view := app.View()
	for _, want := range []string{"student-loan", "HKD ( 25.00)"} {
		if !strings.Contains(view, want) {
			t.Fatalf("rendered negative balance missing %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "HKD -25.00") {
		t.Fatalf("rendered view should use brackets, not minus:\n%s", view)
	}

	app.Path = "/accounts/student-loan/balances/2026-05-21/edit/"
	app.Form = map[string]string{"date": "2026-05-21", "balance": "-25.00", "notes": ""}
	app.Field = 1
	view = app.View()
	if !strings.Contains(view, "balance  : HKD -25.00|") {
		t.Fatalf("edit form should keep minus sign for raw input:\n%s", view)
	}
	if strings.Contains(view, "balance  : HKD (25.00)") {
		t.Fatalf("edit form field should not use bracket notation:\n%s", view)
	}
	if !strings.Contains(view, "balance     : HKD (25.00)") {
		t.Fatalf("context summary should use bracket notation:\n%s", view)
	}
	_ = store
}

func linesContaining(text string, needles []string) []string {
	var out []string
	for _, line := range strings.Split(text, "\n") {
		if !strings.Contains(line, "|") {
			continue
		}
		for _, needle := range needles {
			if strings.Contains(line, needle) {
				out = append(out, line)
				break
			}
		}
	}
	return out
}

func linesContainingAny(text string, needles []string) []string {
	var out []string
	for _, line := range strings.Split(text, "\n") {
		for _, needle := range needles {
			if strings.Contains(line, needle) {
				out = append(out, line)
				break
			}
		}
	}
	return out
}

func moneyParts(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if idx := strings.Index(line, "HKD"); idx >= 0 {
			out = append(out, line[idx:])
		}
	}
	return out
}

func assertSamePrimaryPunctuationIndex(t *testing.T, lines []string, punctuation string) {
	t.Helper()
	assertSamePrimaryPunctuationIndexFor(t, lines, punctuation, true)
}

func assertSamePrimaryPunctuationIndexIfPresent(t *testing.T, lines []string, punctuation string) {
	t.Helper()
	assertSamePrimaryPunctuationIndexFor(t, lines, punctuation, false)
}

func assertSamePrimaryPunctuationIndexFor(t *testing.T, lines []string, punctuation string, require bool) {
	t.Helper()
	want := -1
	for _, line := range lines {
		idx := strings.Index(line, punctuation)
		if punctuation == "," {
			idx = strings.LastIndex(line, punctuation)
		}
		if idx < 0 {
			continue
		}
		if want < 0 {
			want = idx
			continue
		}
		if idx != want {
			t.Fatalf("%q shifted in money column:\n%s", punctuation, strings.Join(lines, "\n"))
		}
	}
	if require && want < 0 {
		t.Fatalf("no %q found in lines:\n%s", punctuation, strings.Join(lines, "\n"))
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
		"account   : cash",
		"balance   : HKD 0.00",
		"children  : HKD 0.00",
		"remaining : HKD 0.00",
		"as of     : none [!]",
		"on-budget : true",
		"notes     : wallet",
		"net changes",
		"high to lows",
		"lows",
		"> 1) balances",
		"  2) child accounts",
		"  3) transactions (TODO)",
		"  4) edit account",
		"  5) hide account",
		"  6) delete account",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("visible account detail missing %q:\n%s", want, view)
		}
	}
	assertNotContains(t, view, "you owe ppl")
	assertNotContains(t, view, "ppl owe you")
	if _, _, err := app.Svc.Accounts.SetHidden(ctx, acct.ID, true); err != nil {
		t.Fatal(err)
	}
	app.Path = "/accounts/cash/"
	app.Menu = 0
	view = app.View()
	for _, want := range []string{"hidden    : true", "> 1) balances", "  2) child accounts", "  3) transactions (TODO)", "  4) edit account", "  5) show account", "  6) delete account"} {
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
	for _, want := range []string{"> 1) date", "2026-05-24|", "2) balance", "HKD (type amount...)", "3) notes", "[confirm]"} {
		if !strings.Contains(view, want) {
			t.Fatalf("balance add form missing %q:\n%s", want, view)
		}
	}
	assertRenderOrder(t, view,
		"name        : cash",
		"/accounts/cash/balances/add/",
		"> 1) date",
	)
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
	for _, want := range []string{"> 2) currency", "   > filter  : (type anything...)", "     > HKD", "       AUD", "       BRL", "       CAD", "     [01/30]", "type       : filter", "left/right : next/prev page", "ctrl+s     : submit"} {
		if !strings.Contains(view, want) {
			t.Fatalf("currency select missing %q:\n%s", want, view)
		}
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyTab})
	app = m.(App)
	view = app.View()
	for _, want := range []string{"> 3) on-budget", "     > true", "false", "ctrl+s  : submit"} {
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
	if view = app.View(); !strings.Contains(view, "> 5) tags") || !strings.Contains(view, "filter  : (type anything...)") {
		t.Fatalf("tags focus missing:\n%s", view)
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = m.(App)
	if view = app.View(); !strings.Contains(view, "> [confirm]") || !strings.Contains(view, "shift-tab : navigate") || !strings.Contains(view, "ctrl+s    : submit") {
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

func TestCtrlSFromAccountNameFieldSubmits(t *testing.T) {
	app, store := testApp(t)
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: "/accounts/list/", Menu: 0}, navFrame{Path: "/accounts/create/", Menu: 0})
	app = pressRunes(app, "cash")
	app = press(app, tea.KeyCtrlS)
	if app.Path != "/accounts/list/" {
		t.Fatalf("ctrl+s from name field should submit account form, got %s\n%s", app.Path, app.View())
	}
	acct, err := store.Acct.GetByName(context.Background(), "cash")
	if err != nil {
		t.Fatal(err)
	}
	if acct.Code != "HKD" || !acct.OnBudget {
		t.Fatalf("created account defaults = code %s onBudget %t", acct.Code, acct.OnBudget)
	}
}

func TestCtrlSFromCurrencyFilterSubmitsCommittedCurrency(t *testing.T) {
	app, store := testApp(t)
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: "/accounts/list/", Menu: 0}, navFrame{Path: "/accounts/create/", Menu: 0})
	app.Form["name"] = "cash"
	app.Field = 1
	app = pressRunes(app, "j")
	if view := app.View(); !strings.Contains(view, "filter  : J") || !strings.Contains(view, "> JPY") {
		t.Fatalf("expected JPY highlighted by currency filter:\n%s", view)
	}
	app = press(app, tea.KeyCtrlS)
	if app.Path != "/accounts/list/" {
		t.Fatalf("ctrl+s from currency filter should submit account form, got %s\n%s", app.Path, app.View())
	}
	acct, err := store.Acct.GetByName(context.Background(), "cash")
	if err != nil {
		t.Fatal(err)
	}
	if acct.Code != "HKD" {
		t.Fatalf("ctrl+s should submit committed currency HKD, got %s", acct.Code)
	}
}

func TestCtrlSFromAccountSelectSubmitsSelectedValue(t *testing.T) {
	app, store := testApp(t)
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: "/accounts/list/", Menu: 0}, navFrame{Path: "/accounts/create/", Menu: 0})
	app.Form["name"] = "off-budget-cash"
	app.Field = 2
	app = press(app, tea.KeyDown)
	app = press(app, tea.KeyCtrlS)
	if app.Path != "/accounts/list/" {
		t.Fatalf("ctrl+s from on-budget select should submit account form, got %s\n%s", app.Path, app.View())
	}
	acct, err := store.Acct.GetByName(context.Background(), "off-budget-cash")
	if err != nil {
		t.Fatal(err)
	}
	if acct.OnBudget {
		t.Fatal("ctrl+s should submit selected on-budget=false value")
	}
}

func TestCtrlSValidationErrorPreservesAccountForm(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	if _, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, ""); err != nil {
		t.Fatal(err)
	}
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: "/accounts/list/", Menu: 0}, navFrame{Path: "/accounts/create/", Menu: 0})
	app.Form = map[string]string{"name": "cash", "currency": "HKD", "on-budget": "true", "notes": "keep me"}
	app = press(app, tea.KeyCtrlS)
	if app.Path != "/accounts/create/" || app.Error == "" {
		t.Fatalf("ctrl+s validation error should stay on form with error, path=%s error=%q", app.Path, app.Error)
	}
	if app.Form["name"] != "cash" || app.Form["notes"] != "keep me" {
		t.Fatalf("ctrl+s validation error should preserve form, got %#v", app.Form)
	}
}

func TestAccountCreateDuplicateNameErrorIsFriendlyAndRecoverable(t *testing.T) {
	app, store := testApp(t)
	ctx := context.Background()
	if _, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, ""); err != nil {
		t.Fatal(err)
	}
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: "/accounts/list/", Menu: 0}, navFrame{Path: "/accounts/create/", Menu: 0})
	app.Form = map[string]string{"name": "cash", "currency": "HKD", "on-budget": "true", "notes": "keep me"}
	app.Field = 5

	app = press(app, tea.KeyEnter)
	if app.Path != "/accounts/create/" {
		t.Fatalf("duplicate account create should stay on form, got %s", app.Path)
	}
	if app.Field != 5 {
		t.Fatalf("duplicate account create should preserve field, got %d", app.Field)
	}
	if app.Form["name"] != "cash" || app.Form["notes"] != "keep me" {
		t.Fatalf("duplicate account create should preserve form, got %#v", app.Form)
	}
	view := app.View()
	if !strings.Contains(view, "account already exists: cash; choose another name") {
		t.Fatalf("duplicate account create should show friendly error:\n%s", view)
	}
	for _, raw := range []string{"UNIQUE constraint failed", "constraint failed", "sql: no rows"} {
		if strings.Contains(view, raw) {
			t.Fatalf("duplicate account create should hide raw error %q:\n%s", raw, view)
		}
	}
	if len(app.History) != 0 {
		t.Fatalf("duplicate account create should not append history, got %d entries", len(app.History))
	}

	app = press(app, tea.KeyShiftTab)
	if app.Field != 4 {
		t.Fatalf("shift+tab after duplicate account error should move cursor to tags, got %d", app.Field)
	}
	app.Form["name"] = "wallet"
	app.Field = 5
	app = press(app, tea.KeyEnter)
	if app.Path != "/accounts/list/" {
		t.Fatalf("corrected account create should return to list, got %s", app.Path)
	}
	if _, err := store.Acct.GetByName(ctx, "wallet"); err != nil {
		t.Fatalf("corrected account was not saved: %v", err)
	}
}

func TestAccountEditDuplicateNameErrorIsFriendlyAndRecoverable(t *testing.T) {
	app, store := testApp(t)
	ctx := context.Background()
	cash, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Accounts.Create(ctx, "savings", "HKD", true, "rainy"); err != nil {
		t.Fatal(err)
	}
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: "/accounts/list/", Menu: 0})
	app = press(app, tea.KeyCtrlE)
	if app.Path != "/accounts/cash/edit/" {
		t.Fatalf("ctrl+e should open account edit form, got %s", app.Path)
	}
	app.Form = map[string]string{"name": "savings", "currency": "HKD", "on-budget": "true", "notes": "rename collision"}
	app.Field = 5

	app = press(app, tea.KeyEnter)
	if app.Path != "/accounts/cash/edit/" {
		t.Fatalf("duplicate account edit should stay on form, got %s", app.Path)
	}
	if app.Field != 5 {
		t.Fatalf("duplicate account edit should preserve field, got %d", app.Field)
	}
	if app.Form["name"] != "savings" || app.Form["notes"] != "rename collision" {
		t.Fatalf("duplicate account edit should preserve form, got %#v", app.Form)
	}
	view := app.View()
	if !strings.Contains(view, "account already exists: savings; choose another name") {
		t.Fatalf("duplicate account edit should show friendly error:\n%s", view)
	}
	for _, raw := range []string{"UNIQUE constraint failed", "constraint failed", "sql: no rows"} {
		if strings.Contains(view, raw) {
			t.Fatalf("duplicate account edit should hide raw error %q:\n%s", raw, view)
		}
	}
	if len(app.History) != 0 {
		t.Fatalf("duplicate account edit should not append history, got %d entries", len(app.History))
	}

	app.Form["name"] = "wallet"
	app.Field = 5
	app = press(app, tea.KeyEnter)
	if app.Path != "/accounts/list/" {
		t.Fatalf("corrected account edit should return to list, got %s", app.Path)
	}
	if _, err := store.Acct.GetByID(ctx, cash.ID); err != nil {
		t.Fatalf("edited account missing: %v", err)
	}
	if _, err := store.Acct.GetByName(ctx, "wallet"); err != nil {
		t.Fatalf("corrected account rename was not saved: %v", err)
	}
}

func TestAccountFormInvalidCurrencyErrorsAreFriendlyAndRecoverable(t *testing.T) {
	app, _ := testApp(t)
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: "/accounts/list/", Menu: 0}, navFrame{Path: "/accounts/create/", Menu: 0})
	app.Form = map[string]string{"name": "cash", "currency": "ZZZ", "on-budget": "true", "notes": "keep"}
	app.Field = 5

	app = press(app, tea.KeyCtrlS)
	if app.Path != "/accounts/create/" || app.Field != 5 {
		t.Fatalf("invalid currency create should stay on form and field, path=%s field=%d", app.Path, app.Field)
	}
	view := app.View()
	if !strings.Contains(view, "currency is unavailable: ZZZ") {
		t.Fatalf("invalid currency create should show friendly error:\n%s", view)
	}
	for _, raw := range []string{"sql: no rows", "currency not found"} {
		if strings.Contains(view, raw) {
			t.Fatalf("invalid currency create should hide raw error %q:\n%s", raw, view)
		}
	}
	if app.Form["currency"] != "ZZZ" || app.Form["notes"] != "keep" {
		t.Fatalf("invalid currency create should preserve form, got %#v", app.Form)
	}

	ctx := context.Background()
	acct, _, err := app.Svc.Accounts.Create(ctx, "wallet", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: "/accounts/wallet/", Menu: 2}, navFrame{Path: "/accounts/wallet/edit/", Menu: 0})
	app.Form = map[string]string{"name": "wallet", "currency": "ZZZ", "on-budget": "true", "notes": "edit keep"}
	app.Field = 5

	app = press(app, tea.KeyCtrlS)
	if app.Path != "/accounts/wallet/edit/" || app.Field != 5 {
		t.Fatalf("invalid currency edit should stay on form and field, path=%s field=%d", app.Path, app.Field)
	}
	view = app.View()
	if !strings.Contains(view, "currency is unavailable: ZZZ") {
		t.Fatalf("invalid currency edit should show friendly error:\n%s", view)
	}
	for _, raw := range []string{"sql: no rows", "currency not found"} {
		if strings.Contains(view, raw) {
			t.Fatalf("invalid currency edit should hide raw error %q:\n%s", raw, view)
		}
	}
	if app.Form["currency"] != "ZZZ" || app.Form["notes"] != "edit keep" {
		t.Fatalf("invalid currency edit should preserve form, got %#v", app.Form)
	}
	if got, err := app.Svc.Accounts.GetByName(ctx, "wallet"); err != nil || got.ID != acct.ID || got.Code != "HKD" {
		t.Fatalf("invalid currency edit should not mutate account, got %+v err=%v", got, err)
	}
}

func TestBackupFailureHasContextAndStaysOnBackup(t *testing.T) {
	app, _ := testApp(t)
	app.Svc.Backup = func(context.Context) (string, error) {
		return "", errors.New("disk full")
	}
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: "/backup/", Menu: 0})

	app = press(app, tea.KeyEnter)
	if app.Path != "/backup/" {
		t.Fatalf("backup failure should stay on backup screen, got %s", app.Path)
	}
	view := app.View()
	if !strings.Contains(view, "could not create backup: disk full") {
		t.Fatalf("backup failure should show contextual error:\n%s", view)
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
	for _, code := range []string{"AUD", "BRL", "CAD", "CHF", "CNY", "INR", "KRW", "MXN", "NZD", "SGD", "THB"} {
		if err := store.UpsertCurrencyNameOnly(ctx, code, code); err != nil {
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
	app.Field = 5
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

func TestBalanceAddScreenRenderOrder(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	if _, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, ""); err != nil {
		t.Fatal(err)
	}
	app.Path = "/accounts/cash/balances/add/"
	app.Form = map[string]string{"date": "2026-05-24"}
	view := app.View()
	assertRenderOrder(t, view,
		"# stuf",
		"name        : cash",
		"balance     : HKD 0.00",
		"as of       : (no balance entered yet)",
		"/accounts/cash/balances/add/",
		"> 1) date",
		"[confirm]",
		"---",
	)
}

func TestBalanceEditScreenRenderOrder(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	acct, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, acct.ID, "2026-05-21", "50000.00", "initial balance"); err != nil {
		t.Fatal(err)
	}
	app.Path = "/accounts/cash/balances/2026-05-21/edit/"
	app.Form = map[string]string{"date": "2026-05-21", "balance": "50000.00", "notes": "initial balance"}
	view := app.View()
	assertRenderOrder(t, view,
		"# stuf",
		"name        : cash",
		"balance     : HKD 50,000.00",
		"as of       : 2026-05-21",
		"/accounts/cash/balances/2026-05-21/edit/",
		"> 1) date",
		"[confirm]",
		"---",
	)
}

func TestBalancesScreensReadmeShape(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	acct, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	app.Path = "/accounts/cash/balances/list/"
	view := app.View()
	for _, want := range []string{"account   : cash", "balance   : HKD 0.00", "as of     : none [!]", "net changes", "/accounts/cash/balances/list/", "(no balances yet)"} {
		if !strings.Contains(view, want) {
			t.Fatalf("empty balances list missing %q:\n%s", want, view)
		}
	}
	assertNotContains(t, view, "you owe ppl")
	assertNotContains(t, view, "ppl owe you")
	if _, _, err := app.Svc.Balances.Add(ctx, acct.ID, "2026-05-21", "50000.00", "initial balance"); err != nil {
		t.Fatal(err)
	}
	app.Path = "/accounts/cash/balances/list/"
	view = app.View()
	for _, want := range []string{"> 2026-05-21 | HKD 50,000.00", "initial balance", "/accounts/cash/balances/list/"} {
		if !strings.Contains(view, want) {
			t.Fatalf("balances list missing %q:\n%s", want, view)
		}
	}
	app.Path = "/accounts/cash/balances/2026-05-21/"
	view = app.View()
	for _, want := range []string{"account   : cash", "net changes", "date    : 2026-05-21", "balance : HKD 50,000.00", "> 1) edit balance", "2) delete balance"} {
		if !strings.Contains(view, want) {
			t.Fatalf("balance detail missing %q:\n%s", want, view)
		}
	}
	assertNotContains(t, view, "you owe ppl")
	assertNotContains(t, view, "ppl owe you")
	app.Path = "/accounts/cash/balances/2026-05-21/edit/"
	app.Form = map[string]string{"date": "2026-05-21", "balance": "50000.00", "notes": "initial balance"}
	view = app.View()
	for _, want := range []string{"> 1) date", "2026-05-21|", "2) balance", "HKD 50,000.00", "3) notes", "[confirm]"} {
		if !strings.Contains(view, want) {
			t.Fatalf("balance edit missing %q:\n%s", want, view)
		}
	}
	assertRenderOrder(t, view,
		"name        : cash",
		"/accounts/cash/balances/2026-05-21/edit/",
		"> 1) date",
	)
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
	if view := app.View(); !strings.Contains(view, "> 2) child accounts") || strings.Contains(view, "> 1) balances") {
		t.Fatalf("account detail marker out of sync:\n%s", view)
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = m.(App)
	if app.Path != "/accounts/savings/children/list/" {
		t.Fatalf("enter should run selected detail action, got %s", app.Path)
	}
}

func TestFormFocusBackspaceAndEscapeAreVisible(t *testing.T) {
	app, _ := testApp(t)
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: "/accounts/list/", Menu: 0}, navFrame{Path: "/accounts/create/", Menu: 0})
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
	if app.Path != "/accounts/list/" || app.Error != "" {
		t.Fatalf("esc should discard form and return to account list: path=%s error=%q", app.Path, app.Error)
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

func TestBalanceDateInputSanitizesOnTypingAndSet(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	if _, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, ""); err != nil {
		t.Fatal(err)
	}
	app.Path = "/accounts/cash/balances/add/"
	app.Form = map[string]string{"date": ""}
	app.Field = 0
	for _, r := range "2026/05/24" {
		m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		app = m.(App)
	}
	if got := app.Form["date"]; got != "2026-05-24" {
		t.Fatalf("typed balance date was not sanitized, got %q", got)
	}
	if view := app.View(); !strings.Contains(view, "date     : 2026-05-24|") {
		t.Fatalf("sanitized date caret should reset to end:\n%s", view)
	}
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("set date=2026 05 xx 24")})
	app = m.(App)
	if got := app.Form["date"]; got != "2026-05-24" {
		t.Fatalf("set date was not sanitized, got %q", got)
	}
	app.Field = 2
	for _, r := range "Hello World" {
		m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		app = m.(App)
	}
	if got := app.Form["notes"]; got != "Hello World" {
		t.Fatalf("balance notes should preserve raw input, got %q", got)
	}
}

func TestBalanceAddEnterAdvancesFields(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	if _, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, ""); err != nil {
		t.Fatal(err)
	}
	app = appWithNav(app,
		navFrame{Path: "/", Menu: 0},
		navFrame{Path: "/accounts/cash/", Menu: 0},
		navFrame{Path: "/accounts/cash/balances/list/", Menu: 0},
		navFrame{Path: "/accounts/cash/balances/add/", Menu: 0},
	)
	app.Form = map[string]string{"date": "2026-05-24", "balance": "100.00", "notes": "test note"}
	app.Field = 0

	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = m.(App)
	if app.Path != "/accounts/cash/balances/add/" {
		t.Fatalf("enter on date should not submit, got path %s", app.Path)
	}
	if app.Field != 1 {
		t.Fatalf("enter on date should move to balance field, field=%d", app.Field)
	}
	if app.Form["date"] != "2026-05-24" {
		t.Fatalf("enter on date should not clear form, date=%q", app.Form["date"])
	}

	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = m.(App)
	if app.Path != "/accounts/cash/balances/add/" {
		t.Fatalf("enter on balance should not submit, got path %s", app.Path)
	}
	if app.Field != 2 {
		t.Fatalf("enter on balance should move to notes field, field=%d", app.Field)
	}
	if app.Form["balance"] != "100.00" {
		t.Fatalf("enter on balance should not clear form, balance=%q", app.Form["balance"])
	}

	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = m.(App)
	if app.Path != "/accounts/cash/balances/add/" {
		t.Fatalf("enter on notes should not submit, got path %s", app.Path)
	}
	if app.Field != 3 {
		t.Fatalf("enter on notes should move to confirm, field=%d", app.Field)
	}
	if app.Form["notes"] != "test note" {
		t.Fatalf("enter on notes should not clear form, notes=%q", app.Form["notes"])
	}
	if view := app.View(); !strings.Contains(view, "> [confirm]") {
		t.Fatalf("confirm row should be focused:\n%s", view)
	}
}

func TestBalanceAddEnterOnConfirmSubmits(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	if _, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, ""); err != nil {
		t.Fatal(err)
	}
	app = appWithNav(app,
		navFrame{Path: "/", Menu: 0},
		navFrame{Path: "/accounts/cash/", Menu: 0},
		navFrame{Path: "/accounts/cash/balances/list/", Menu: 0},
		navFrame{Path: "/accounts/cash/balances/add/", Menu: 0},
	)
	app.Form = map[string]string{"date": "2026-05-24", "balance": "100.00", "notes": "test note"}
	app.Field = 3

	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = m.(App)
	if app.Path != "/accounts/cash/balances/list/" {
		t.Fatalf("enter on confirm should submit and return to balances list, got %s", app.Path)
	}
	if app.Form["balance"] != "" {
		t.Fatalf("form should clear after submit, balance=%q", app.Form["balance"])
	}
	acct, err := app.Svc.Accounts.GetByName(ctx, "cash")
	if err != nil {
		t.Fatal(err)
	}
	saved, err := app.Svc.Balances.GetByAccountDate(ctx, acct.ID, "2026-05-24")
	if err != nil {
		t.Fatal(err)
	}
	if got := saved.Amount.Format("HKD"); got != "HKD 100.00" {
		t.Fatalf("saved balance = %q", got)
	}
	if saved.Notes != "test note" {
		t.Fatalf("saved notes = %q", saved.Notes)
	}
}

func TestBalanceAddCtrlSSubmitsFromBalanceField(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	if _, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, ""); err != nil {
		t.Fatal(err)
	}
	app = appWithNav(app,
		navFrame{Path: "/", Menu: 0},
		navFrame{Path: "/accounts/cash/", Menu: 0},
		navFrame{Path: "/accounts/cash/balances/list/", Menu: 0},
		navFrame{Path: "/accounts/cash/balances/add/", Menu: 0},
	)
	app.Form = map[string]string{"date": "2026-05-24", "balance": "100.00", "notes": "test note"}
	app.Field = 1

	app = press(app, tea.KeyCtrlS)
	if app.Path != "/accounts/cash/balances/list/" {
		t.Fatalf("ctrl+s should submit add balance and return to list, got %s", app.Path)
	}
	acct, err := app.Svc.Accounts.GetByName(ctx, "cash")
	if err != nil {
		t.Fatal(err)
	}
	saved, err := app.Svc.Balances.GetByAccountDate(ctx, acct.ID, "2026-05-24")
	if err != nil {
		t.Fatal(err)
	}
	if got := saved.Amount.Format("HKD"); got != "HKD 100.00" {
		t.Fatalf("saved balance = %q", got)
	}
}

func TestBalanceAddDuplicateDateErrorPreservesFormAndRecovers(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	acct, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, acct.ID, "2026-05-24", "100.00", "original"); err != nil {
		t.Fatal(err)
	}
	app = appWithNav(app,
		navFrame{Path: "/", Menu: 0},
		navFrame{Path: "/accounts/cash/", Menu: 0},
		navFrame{Path: "/accounts/cash/balances/list/", Menu: 0},
		navFrame{Path: "/accounts/cash/balances/add/", Menu: 0},
	)
	app.Form = map[string]string{"date": "2026-05-24", "balance": "125.00", "notes": "corrected"}
	app.Field = 3

	app = press(app, tea.KeyEnter)
	if app.Path != "/accounts/cash/balances/add/" {
		t.Fatalf("duplicate balance add should stay on form, got %s", app.Path)
	}
	if app.Field != 3 {
		t.Fatalf("duplicate balance add should preserve field, got %d", app.Field)
	}
	if app.Form["date"] != "2026-05-24" || app.Form["balance"] != "125.00" || app.Form["notes"] != "corrected" {
		t.Fatalf("duplicate balance add should preserve form, got %#v", app.Form)
	}
	view := app.View()
	if !strings.Contains(view, "balance already exists for 2026-05-24; edit the existing balance instead") {
		t.Fatalf("duplicate balance add should show friendly error:\n%s", view)
	}
	if strings.Contains(view, "UNIQUE constraint failed") {
		t.Fatalf("duplicate balance add should hide raw sqlite error:\n%s", view)
	}
	if len(app.History) != 0 {
		t.Fatalf("duplicate balance add should not append history, got %d entries", len(app.History))
	}

	app = press(app, tea.KeyShiftTab)
	if app.Field != 2 {
		t.Fatalf("shift+tab after duplicate error should move cursor to notes, got %d", app.Field)
	}
	app.Field = 0
	app.Form["date"] = "2026-05-25"
	app.Field = 3
	app = press(app, tea.KeyEnter)
	if app.Path != "/accounts/cash/balances/list/" {
		t.Fatalf("corrected balance add should return to list, got %s", app.Path)
	}
	if _, err := app.Svc.Balances.GetByAccountDate(ctx, acct.ID, "2026-05-25"); err != nil {
		t.Fatalf("corrected balance was not saved: %v", err)
	}
}

func TestBalanceAddDuplicateDateErrorCanReturnToListAndEditExisting(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	acct, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, acct.ID, "2026-05-24", "100.00", "original"); err != nil {
		t.Fatal(err)
	}
	app = appWithNav(app,
		navFrame{Path: "/", Menu: 0},
		navFrame{Path: "/accounts/cash/", Menu: 0},
		navFrame{Path: "/accounts/cash/balances/list/", Menu: 0},
		navFrame{Path: "/accounts/cash/balances/add/", Menu: 0},
	)
	app.Form = map[string]string{"date": "2026-05-24", "balance": "125.00", "notes": "duplicate"}
	app.Field = 3

	app = press(app, tea.KeyEnter)
	app = press(app, tea.KeyEsc)
	if app.Path != "/accounts/cash/balances/list/" {
		t.Fatalf("esc after duplicate error should return to balance list, got %s", app.Path)
	}
	view := app.View()
	if !strings.Contains(view, "> 2026-05-24") || !strings.Contains(view, "original") {
		t.Fatalf("existing balance should remain selectable after duplicate error:\n%s", view)
	}

	app = press(app, tea.KeyCtrlE)
	if app.Path != "/accounts/cash/balances/2026-05-24/edit/" {
		t.Fatalf("ctrl+e after duplicate error should edit selected balance, got %s", app.Path)
	}
	if app.Form["date"] != "2026-05-24" || app.Form["balance"] != "100.00" || app.Form["notes"] != "original" {
		t.Fatalf("edit existing should populate selected balance, got %#v", app.Form)
	}
}

func TestBalanceEditEnterOnConfirmSubmits(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	acct, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, acct.ID, "2026-05-21", "50000.00", "initial balance"); err != nil {
		t.Fatal(err)
	}
	app = appWithNav(app,
		navFrame{Path: "/", Menu: 0},
		navFrame{Path: "/accounts/cash/", Menu: 0},
		navFrame{Path: "/accounts/cash/balances/list/", Menu: 0},
		navFrame{Path: "/accounts/cash/balances/2026-05-21/", Menu: 0},
		navFrame{Path: "/accounts/cash/balances/2026-05-21/edit/", Menu: 0},
	)
	app.Form = map[string]string{"date": "2026-05-21", "balance": "60000.00", "notes": "updated"}
	app.Field = 1

	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = m.(App)
	if app.Path != "/accounts/cash/balances/2026-05-21/edit/" {
		t.Fatalf("enter on balance should not submit edit, got path %s", app.Path)
	}
	if app.Field != 2 {
		t.Fatalf("enter on balance should move to notes, field=%d", app.Field)
	}

	app.Field = 3
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = m.(App)
	if app.Path != "/accounts/cash/balances/list/" {
		t.Fatalf("enter on confirm should submit edit and return to balances list, got %s", app.Path)
	}
	saved, err := app.Svc.Balances.GetByAccountDate(ctx, acct.ID, "2026-05-21")
	if err != nil {
		t.Fatal(err)
	}
	if got := saved.Amount.Format("HKD"); got != "HKD 60,000.00" {
		t.Fatalf("updated balance = %q", got)
	}
	if saved.Notes != "updated" {
		t.Fatalf("updated notes = %q", saved.Notes)
	}
}

func TestBalanceEditCtrlSSubmitsFromBalanceField(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	acct, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, acct.ID, "2026-05-21", "50000.00", "initial balance"); err != nil {
		t.Fatal(err)
	}
	app = appWithNav(app,
		navFrame{Path: "/", Menu: 0},
		navFrame{Path: "/accounts/cash/", Menu: 0},
		navFrame{Path: "/accounts/cash/balances/list/", Menu: 0},
		navFrame{Path: "/accounts/cash/balances/2026-05-21/", Menu: 0},
		navFrame{Path: "/accounts/cash/balances/2026-05-21/edit/", Menu: 0},
	)
	app.Form = map[string]string{"date": "2026-05-21", "balance": "60000.00", "notes": "updated"}
	app.Field = 1

	app = press(app, tea.KeyCtrlS)
	if app.Path != "/accounts/cash/balances/list/" {
		t.Fatalf("ctrl+s should submit edit balance and return to list, got %s", app.Path)
	}
	saved, err := app.Svc.Balances.GetByAccountDate(ctx, acct.ID, "2026-05-21")
	if err != nil {
		t.Fatal(err)
	}
	if got := saved.Amount.Format("HKD"); got != "HKD 60,000.00" {
		t.Fatalf("updated balance = %q", got)
	}
}

func TestBalanceEditDuplicateDateErrorPreservesFormAndRecovers(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	acct, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, acct.ID, "2026-05-24", "100.00", "first"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, acct.ID, "2026-05-25", "200.00", "second"); err != nil {
		t.Fatal(err)
	}
	app = appWithNav(app,
		navFrame{Path: "/", Menu: 0},
		navFrame{Path: "/accounts/cash/", Menu: 0},
		navFrame{Path: "/accounts/cash/balances/list/", Menu: 1},
	)
	app = press(app, tea.KeyCtrlE)
	if app.Path != "/accounts/cash/balances/2026-05-24/edit/" {
		t.Fatalf("ctrl+e should open older balance edit form, got %s", app.Path)
	}
	app.Form = map[string]string{"date": "2026-05-25", "balance": "125.00", "notes": "collides"}
	app.Field = 3

	app = press(app, tea.KeyEnter)
	if app.Path != "/accounts/cash/balances/2026-05-24/edit/" {
		t.Fatalf("duplicate balance edit should stay on form, got %s", app.Path)
	}
	if app.Field != 3 {
		t.Fatalf("duplicate balance edit should preserve field, got %d", app.Field)
	}
	if app.Form["date"] != "2026-05-25" || app.Form["balance"] != "125.00" || app.Form["notes"] != "collides" {
		t.Fatalf("duplicate balance edit should preserve form, got %#v", app.Form)
	}
	view := app.View()
	if !strings.Contains(view, "balance already exists for 2026-05-25; edit the existing balance instead") {
		t.Fatalf("duplicate balance edit should show friendly error:\n%s", view)
	}
	if strings.Contains(view, "UNIQUE constraint failed") {
		t.Fatalf("duplicate balance edit should hide raw sqlite error:\n%s", view)
	}
	if len(app.History) != 0 {
		t.Fatalf("duplicate balance edit should not append history, got %d entries", len(app.History))
	}

	app = press(app, tea.KeyShiftTab)
	if app.Field != 2 {
		t.Fatalf("shift+tab after duplicate edit error should move cursor to notes, got %d", app.Field)
	}
	app.Form["date"] = "2026-05-26"
	app.Field = 3
	app = press(app, tea.KeyEnter)
	if app.Path != "/accounts/cash/balances/list/" {
		t.Fatalf("corrected balance edit should return to list, got %s", app.Path)
	}
	if _, err := app.Svc.Balances.GetByAccountDate(ctx, acct.ID, "2026-05-26"); err != nil {
		t.Fatalf("corrected balance edit was not saved: %v", err)
	}
}

func TestBalanceFormHelpEnterSemantics(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	if _, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, ""); err != nil {
		t.Fatal(err)
	}
	app.Path = "/accounts/cash/balances/add/"
	app.Form = map[string]string{"date": "2026-05-24"}
	app.Field = 0
	view := app.View()
	if !strings.Contains(view, "enter   : next field") || !strings.Contains(view, "ctrl+s  : submit") {
		t.Fatalf("date field help should say enter advances:\n%s", view)
	}
	app.Field = 2
	view = app.View()
	if !strings.Contains(view, "enter   : next field") || !strings.Contains(view, "ctrl+s  : submit") || strings.Contains(view, "?       : help") {
		t.Fatalf("notes field help should say enter advances and hide ?:\n%s", view)
	}
	app.Field = 3
	view = app.View()
	if !strings.Contains(view, "enter     : confirm") || !strings.Contains(view, "ctrl+s    : submit") {
		t.Fatalf("confirm row help should say enter submits:\n%s", view)
	}
}

func TestBalanceAmountInputFormatting(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	if _, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, ""); err != nil {
		t.Fatal(err)
	}
	app.Path = "/accounts/cash/balances/add/"
	app.Form = map[string]string{"date": "2026-05-24"}
	app.Field = 1
	for _, r := range "1,234abc.56xx" {
		m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		app = m.(App)
	}
	if got := app.Form["balance"]; got != "1234.56" {
		t.Fatalf("stored balance should be sanitized numeric, got %q", got)
	}
	if view := app.View(); !strings.Contains(view, "balance  : HKD 1,234.56|") {
		t.Fatalf("balance field should render currency and commas:\n%s", view)
	}
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("set balance=9,999.00!!")})
	app = m.(App)
	if got := app.Form["balance"]; got != "9999.00" {
		t.Fatalf("set balance should sanitize, got %q", got)
	}
	for i := 0; i < 3; i++ {
		m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
		app = m.(App)
	}
	if app.Form["balance"] != "" {
		t.Fatalf("form should clear after submit, balance=%q", app.Form["balance"])
	}
	acct, err := app.Svc.Accounts.GetByName(ctx, "cash")
	if err != nil {
		t.Fatal(err)
	}
	saved, err := app.Svc.Balances.GetByAccountDate(ctx, acct.ID, "2026-05-24")
	if err != nil {
		t.Fatal(err)
	}
	if got := saved.Amount.Format("HKD"); got != "HKD 9,999.00" {
		t.Fatalf("saved balance = %q", got)
	}
}

func TestSanitizedNameSubmitsAndEditRedirectsToNewSlug(t *testing.T) {
	app, store := testApp(t)
	app.Path = "/accounts/create/"
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("set name=My Cash!!")})
	app = m.(App)
	app.Field = 5
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = m.(App)
	if app.Path != "/accounts/list/" {
		t.Fatalf("sanitized create did not submit: %s\n%s", app.Path, app.View())
	}
	acct, err := store.Acct.GetByName(context.Background(), "my-cash")
	if err != nil {
		t.Fatal(err)
	}
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: "/accounts/my-cash/", Menu: 2}, navFrame{Path: "/accounts/my-cash/edit/", Menu: 0})
	app.Form = map[string]string{"name": acct.Name, "currency": "HKD", "on-budget": "true"}
	app.Field = 0
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("set name=New CASH Account!!")})
	app = m.(App)
	app.Field = 5
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
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: "/accounts/cash/", Menu: 0}, navFrame{Path: "/accounts/cash/balances/list/", Menu: 0})
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyDown})
	app = m.(App)
	view := app.View()
	if !strings.Contains(view, "> 2026-05-01") || strings.Contains(view, "> 2026-06-01") {
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
	if app.Path != "/accounts/cash/balances/list/" {
		t.Fatalf("delete should return to balances list, got %s", app.Path)
	}
	if _, err := store.Bal.GetByAccountDate(ctx, acct.ID, "2026-05-01"); err == nil {
		t.Fatal("selected balance should have been deleted")
	}
}

func TestAccountCreateValidationHistoryAndUndo(t *testing.T) {
	app, store := testApp(t)
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: "/accounts/list/", Menu: 0}, navFrame{Path: "/accounts/create/", Menu: 0})
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("set name=!!!")})
	app = m.(App)
	app.Field = 5
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = m.(App)
	if !strings.Contains(app.View(), "strict slug") {
		t.Fatalf("expected validation error:\n%s", app.View())
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("set name=cash")})
	app = m.(App)
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("set currency=HKD")})
	app = m.(App)
	app.Field = 5
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
	view := app.View()
	for _, want := range []string{
		"# stuf",
		"total       : HKD 0.00",
		"quit stuf?",
		"> 1) no",
		"  2) yes",
		"up/down/j/k   : navigate",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("exit confirmation missing %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "\n/\n") {
		t.Fatalf("exit confirmation should hide url path:\n%s", view)
	}
	if strings.Contains(view, "> 1) accounts") {
		t.Fatalf("exit confirmation should replace menu items:\n%s", view)
	}
}

func TestExitConfirmationRequiresYesToQuit(t *testing.T) {
	app, _ := testApp(t)
	app.Path = "/"
	m, cmd := app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	app = m.(App)
	if cmd != nil {
		t.Fatal("esc should open confirmation, not quit")
	}
	if !app.ExitAsk || app.Menu != 0 {
		t.Fatalf("expected exit confirmation with no selected, got ExitAsk=%t Menu=%d", app.ExitAsk, app.Menu)
	}

	m, cmd = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = m.(App)
	if cmd != nil {
		t.Fatal("enter on no should cancel confirmation, not quit")
	}
	if app.ExitAsk {
		t.Fatal("enter on no should close confirmation")
	}

	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	app = m.(App)
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyDown})
	app = m.(App)
	if !strings.Contains(app.View(), "> 2) yes") {
		t.Fatalf("down should select yes:\n%s", app.View())
	}
	m, cmd = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter on yes should quit")
	}

	app.Path = "/"
	app.ExitAsk = false
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	app = m.(App)
	m, cmd = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	if cmd == nil {
		t.Fatal("hotkey 2 should quit")
	}

	app.Path = "/"
	app.ExitAsk = false
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	app = m.(App)
	m, cmd = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	app = m.(App)
	if cmd != nil {
		t.Fatal("y should not quit directly")
	}
}

func TestExitConfirmationHistoryWarningAndCancel(t *testing.T) {
	app, store := testApp(t)
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: "/accounts/list/", Menu: 0}, navFrame{Path: "/accounts/create/", Menu: 0})
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("set name=cash")})
	app = m.(App)
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("set currency=HKD")})
	app = m.(App)
	app.Field = 5
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = m.(App)
	if len(app.History) != 1 {
		t.Fatalf("expected undo history after account create, got %d entries", len(app.History))
	}
	app.Path = "/"
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	app = m.(App)
	view := app.View()
	if !strings.Contains(view, "history (ctrl-z to undo)") || !strings.Contains(view, "undo history will be cleared") {
		t.Fatalf("expected undo history warning in exit confirmation:\n%s", view)
	}

	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	app = m.(App)
	if len(app.History) != 1 {
		t.Fatalf("esc cancel should preserve undo history, got %d entries", len(app.History))
	}
	if _, err := store.Acct.GetByName(context.Background(), "cash"); err != nil {
		t.Fatal(err)
	}

	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	app = m.(App)
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = m.(App)
	if len(app.History) != 1 {
		t.Fatalf("enter on no should preserve undo history, got %d entries", len(app.History))
	}
}

func TestManualAccountAndBalanceFlow(t *testing.T) {
	app, _ := testApp(t)
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: "/accounts/list/", Menu: 0}, navFrame{Path: "/accounts/create/", Menu: 0})
	for _, r := range "cash" {
		m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		app = m.(App)
	}
	app.Field = 5
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
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	app = m.(App)
	if app.Path != "/accounts/cash/transactions/" {
		t.Fatalf("transactions TODO path = %s", app.Path)
	}
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: "/accounts/list/", Menu: 0}, navFrame{Path: "/accounts/cash/", Menu: 0})
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	app = m.(App)
	if app.Path != "/accounts/cash/balances/list/" {
		t.Fatalf("balances path = %s", app.Path)
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyCtrlN})
	app = m.(App)
	if app.Path != "/accounts/cash/balances/add/" {
		t.Fatalf("ctrl+n on balance list should open add form, got %s", app.Path)
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyTab})
	app = m.(App)
	for _, r := range "123.45" {
		m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		app = m.(App)
	}
	for i := 0; i < 3; i++ {
		m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
		app = m.(App)
	}
	if app.Path != "/accounts/cash/balances/list/" || !strings.Contains(app.View(), "HKD 123.45") {
		t.Fatalf("balance flow failed path=%s view:\n%s", app.Path, app.View())
	}
}

func TestMenuCursorRestoresOnBackFromAccountList(t *testing.T) {
	app, _ := testApp(t)
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	app = m.(App)
	if app.Path != "/accounts/list/" {
		t.Fatalf("expected account list, got %s", app.Path)
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	app = m.(App)
	if app.Path != "/" {
		t.Fatalf("expected home, got %s", app.Path)
	}
	if view := app.View(); !strings.Contains(view, "> 1) accounts") {
		t.Fatalf("expected accounts cursor restored on home:\n%s", view)
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
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: "/accounts/list/", Menu: 0}, navFrame{Path: "/accounts/cash/", Menu: 0})
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyDown})
	app = m.(App)
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyDown})
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
	if view := app.View(); !strings.Contains(view, "> 3) transactions (TODO)") || strings.Contains(view, "> 1) balances") {
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
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: "/accounts/cash/", Menu: 0}, navFrame{Path: "/accounts/cash/balances/list/", Menu: 0})
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyDown})
	app = m.(App)
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = m.(App)
	if app.Path != "/accounts/cash/balances/2026-05-01/" {
		t.Fatalf("expected balance detail, got %s", app.Path)
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	app = m.(App)
	if app.Path != "/accounts/cash/balances/list/" {
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
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyCtrlH})
	app = m.(App)
	if app.Path != "/accounts/list/" || app.AccountVisible != accountVisibilityHiddenOnly {
		t.Fatalf("expected hidden-only account list, got path=%s mode=%s", app.Path, app.AccountVisible.label())
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	app = m.(App)
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("7")})
	app = m.(App)
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	app = m.(App)
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	app = m.(App)
	if view := app.View(); !strings.Contains(view, "showing : non-hidden") {
		t.Fatalf("expected default account visibility after re-entering list:\n%s", view)
	}
}

func TestUndoResetsNavigationStack(t *testing.T) {
	app, store := testApp(t)
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	app = m.(App)
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	app = m.(App)
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: "/accounts/list/", Menu: 0}, navFrame{Path: "/accounts/create/", Menu: 0})
	app.Form = map[string]string{"name": "cash", "currency": "HKD", "on-budget": "true"}
	app.Field = 5
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
	if view := app.View(); !strings.Contains(view, "showing : non-hidden") {
		t.Fatalf("account list should start fresh after undo clears navigation stack:\n%s", view)
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
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = m.(App)
	for range 1 {
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
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: "/accounts/list/", Menu: 0})
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = m.(App)
	if app.Path != "/accounts/cash/" {
		t.Fatalf("expected account detail, got %s", app.Path)
	}
	if view := app.View(); !strings.Contains(view, "> 1) balances") || strings.Contains(view, "> 2) child accounts") {
		t.Fatalf("expected balances cursor after re-entering account detail:\n%s", view)
	}
	if _, err := store.Acct.GetByName(ctx, "cash"); err != nil {
		t.Fatal(err)
	}
}

func TestAccountCreateRedirectRestoresListCursorOnBack(t *testing.T) {
	app, store := testApp(t)
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: "/accounts/list/", Menu: 0}, navFrame{Path: "/accounts/create/", Menu: 0})
	app.Form = map[string]string{"name": "cash", "currency": "HKD", "on-budget": "true"}
	app.Field = 5
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
	if app.Path != "/" {
		t.Fatalf("expected home, got %s", app.Path)
	}
	if view := app.View(); !strings.Contains(view, "> 1) accounts") {
		t.Fatalf("expected accounts selected after backing out of post-create list:\n%s", view)
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
	if app.Path != "/accounts/list/" {
		t.Fatalf("entering accounts should open list, got %s", app.Path)
	}
}

func TestTabDoesNotNavigateCurrencySelectOptions(t *testing.T) {
	app, _ := testApp(t)
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: "/accounts/list/", Menu: 0}, navFrame{Path: "/accounts/create/", Menu: 0})
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

func TestMenuHorizontalBackAndOpen(t *testing.T) {
	app, _ := testApp(t)
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	app = m.(App)
	if app.Path != "/accounts/list/" {
		t.Fatalf("right/l should open selected menu item, got %s", app.Path)
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyLeft})
	app = m.(App)
	if app.Path != "/" {
		t.Fatalf("left arrow should go back from list, got %s", app.Path)
	}
}

func TestExitConfirmHorizontalKeys(t *testing.T) {
	app, _ := testApp(t)
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	app = m.(App)
	if !app.ExitAsk {
		t.Fatal("left/h on home should open exit confirmation like esc")
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	app = m.(App)
	if app.ExitAsk {
		t.Fatal("left/h on exit confirmation should cancel like esc")
	}

	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	app = m.(App)
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyDown})
	app = m.(App)
	m, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	if cmd == nil {
		t.Fatal("right/l on yes should quit like enter")
	}
	_ = m
}

func TestAccountListHLFilterText(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	if _, _, err := app.Svc.Accounts.Create(ctx, "alpha", "HKD", true, ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Accounts.Create(ctx, "hotel-hl", "HKD", true, ""); err != nil {
		t.Fatal(err)
	}
	app.Path = "/accounts/list/"
	for _, r := range "hl" {
		m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		app = m.(App)
	}
	view := app.View()
	if !strings.Contains(view, "> filter : hl") {
		t.Fatalf("h/l should append to account filter:\n%s", view)
	}
	if !strings.Contains(view, "> hotel-hl") || strings.Contains(view, "> alpha") {
		t.Fatalf("h/l should filter, not navigate:\n%s", view)
	}
}

func TestAccountListLeftRightBackOpen(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	if _, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, ""); err != nil {
		t.Fatal(err)
	}
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: "/accounts/list/", Menu: 0})
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRight})
	app = m.(App)
	if app.Path != "/accounts/cash/" {
		t.Fatalf("right should open selected account, got %s", app.Path)
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyLeft})
	app = m.(App)
	if app.Path != "/accounts/list/" {
		t.Fatalf("left should go back to account list, got %s", app.Path)
	}
}

func TestCurrencySelectHLFilterText(t *testing.T) {
	app, _ := testApp(t)
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: "/accounts/list/", Menu: 0}, navFrame{Path: "/accounts/create/", Menu: 0})
	app.Field = 1
	for _, r := range "hl" {
		m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		app = m.(App)
	}
	view := app.View()
	if !strings.Contains(view, "> filter  : HL") {
		t.Fatalf("h/l should append to currency filter:\n%s", view)
	}
}

func TestTextFieldHLTyping(t *testing.T) {
	app, _ := testApp(t)
	app.Path = "/accounts/create/"
	app.Field = 3
	for _, r := range "hl" {
		m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		app = m.(App)
	}
	if got := app.Form["notes"]; got != "hl" {
		t.Fatalf("h/l should type into notes field, got %q", got)
	}
}

func TestBalanceDetailPrevNextNavigation(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	acct, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, acct.ID, "2026-06-01", "150.00", "newer"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, acct.ID, "2026-05-01", "100.00", "older"); err != nil {
		t.Fatal(err)
	}
	app = appWithNav(app,
		navFrame{Path: "/", Menu: 0},
		navFrame{Path: "/accounts/cash/", Menu: 0},
		navFrame{Path: "/accounts/cash/balances/list/", Menu: 0},
		navFrame{Path: "/accounts/cash/balances/2026-06-01/", Menu: 0},
	)
	view := app.View()
	if !strings.Contains(view, "left/h      : older") || strings.Contains(view, "right/l     : newer") {
		t.Fatalf("newest balance detail help should show older only:\n%s", view)
	}
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	app = m.(App)
	if app.Path != "/accounts/cash/balances/2026-05-01/" {
		t.Fatalf("left/h should move to older balance, got %s", app.Path)
	}
	view = app.View()
	if strings.Contains(view, "left/h      : older") {
		t.Fatalf("oldest balance should hide older shortcut:\n%s", view)
	}
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	app = m.(App)
	if app.Path != "/accounts/cash/balances/2026-06-01/" {
		t.Fatalf("right/l should move to newer balance, got %s", app.Path)
	}
}

func TestMenuHorizontalArrowKeys(t *testing.T) {
	app, _ := testApp(t)
	app = pressRunes(app, "1")
	if app.Path != "/accounts/list/" {
		t.Fatalf("expected account list, got %s", app.Path)
	}
	app = press(app, tea.KeyLeft)
	if app.Path != "/" {
		t.Fatalf("left arrow should go back from account list, got %s", app.Path)
	}
	app = press(app, tea.KeyRight)
	if app.Path != "/accounts/list/" {
		t.Fatalf("right arrow should open selected menu item, got %s", app.Path)
	}
}

func TestExitConfirmRightLOnNoCancels(t *testing.T) {
	app, _ := testApp(t)
	app = press(app, tea.KeyEsc)
	if !app.ExitAsk || app.Menu != 0 {
		t.Fatalf("expected exit confirmation on no, got ExitAsk=%t Menu=%d", app.ExitAsk, app.Menu)
	}
	app = press(app, tea.KeyRight)
	if app.ExitAsk {
		t.Fatal("right/l on no should cancel exit confirmation, not quit")
	}
	if app.Path != "/" {
		t.Fatalf("expected to remain on home, got %s", app.Path)
	}
}

func TestAccountDetailHorizontalBackAndOpen(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	if _, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, ""); err != nil {
		t.Fatal(err)
	}
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: "/accounts/list/", Menu: 0}, navFrame{Path: "/accounts/cash/", Menu: 0})
	app = press(app, tea.KeyRight)
	if app.Path != "/accounts/cash/balances/list/" {
		t.Fatalf("right/l should open highlighted action, got %s", app.Path)
	}
	app = press(app, tea.KeyLeft)
	if app.Path != "/accounts/cash/" {
		t.Fatalf("left/h should go back from account detail menu, got %s", app.Path)
	}
}

func TestBalanceDetailPrevNextTakesPriorityOverMenuBack(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	acct, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, acct.ID, "2026-06-01", "150.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, acct.ID, "2026-05-01", "100.00", ""); err != nil {
		t.Fatal(err)
	}
	app = appWithNav(app,
		navFrame{Path: "/", Menu: 0},
		navFrame{Path: "/accounts/cash/", Menu: 0},
		navFrame{Path: "/accounts/cash/balances/list/", Menu: 0},
		navFrame{Path: "/accounts/cash/balances/2026-06-01/", Menu: 0},
	)
	stackBefore := app.Nav.Len()
	app = press(app, tea.KeyLeft)
	if app.Path != "/accounts/cash/balances/2026-05-01/" {
		t.Fatalf("left/h on balance detail should move to older balance, not go back, got %s", app.Path)
	}
	if app.Nav.Len() != stackBefore {
		t.Fatalf("lateral balance navigation should replace route, stack went from %d to %d", stackBefore, app.Nav.Len())
	}
}

func TestBalanceDetailBoundaryNoOps(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	acct, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, acct.ID, "2026-06-01", "150.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, acct.ID, "2026-05-01", "100.00", ""); err != nil {
		t.Fatal(err)
	}

	app = appWithNav(app,
		navFrame{Path: "/", Menu: 0},
		navFrame{Path: "/accounts/cash/balances/list/", Menu: 0},
		navFrame{Path: "/accounts/cash/balances/2026-06-01/", Menu: 1},
	)
	app = pressRunes(app, "l")
	if app.Path != "/accounts/cash/balances/2026-06-01/" || app.Menu != 1 {
		t.Fatalf("right/l at newest boundary should no-op, got path=%s menu=%d", app.Path, app.Menu)
	}

	app = appWithNav(app,
		navFrame{Path: "/", Menu: 0},
		navFrame{Path: "/accounts/cash/balances/list/", Menu: 0},
		navFrame{Path: "/accounts/cash/balances/2026-05-01/", Menu: 1},
	)
	app = pressRunes(app, "h")
	if app.Path != "/accounts/cash/balances/2026-05-01/" || app.Menu != 1 {
		t.Fatalf("left/h at oldest boundary should no-op, got path=%s menu=%d", app.Path, app.Menu)
	}
}

func TestBalanceDetailSingleBalanceNoLateralHelp(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	acct, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, acct.ID, "2026-06-01", "150.00", ""); err != nil {
		t.Fatal(err)
	}
	app = appWithNav(app,
		navFrame{Path: "/", Menu: 0},
		navFrame{Path: "/accounts/cash/balances/list/", Menu: 0},
		navFrame{Path: "/accounts/cash/balances/2026-06-01/", Menu: 0},
	)
	view := app.View()
	for _, line := range []string{"left/h      : older", "right/l     : newer", "left/h      :", "right/l     :"} {
		if strings.Contains(view, line) {
			t.Fatalf("single balance detail should hide lateral shortcuts, found %q in:\n%s", line, view)
		}
	}
	app = pressRunes(app, "hl")
	if app.Path != "/accounts/cash/balances/2026-06-01/" {
		t.Fatalf("h/l with only one balance should not lateral-nav, got %s", app.Path)
	}
}

func TestBalanceListLeftRightBackOpen(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	acct, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, acct.ID, "2026-06-01", "150.00", ""); err != nil {
		t.Fatal(err)
	}
	app = appWithNav(app,
		navFrame{Path: "/", Menu: 0},
		navFrame{Path: "/accounts/cash/", Menu: 0},
		navFrame{Path: "/accounts/cash/balances/list/", Menu: 0},
	)
	app = press(app, tea.KeyLeft)
	if app.Path != "/accounts/cash/" {
		t.Fatalf("left on balance list should go back, got %s", app.Path)
	}
	app = appWithNav(app,
		navFrame{Path: "/", Menu: 0},
		navFrame{Path: "/accounts/cash/", Menu: 0},
		navFrame{Path: "/accounts/cash/balances/list/", Menu: 0},
	)
	app = press(app, tea.KeyRight)
	if app.Path != "/accounts/cash/balances/2026-06-01/" {
		t.Fatalf("right on balance list should open selected balance, got %s", app.Path)
	}
}

func TestBalanceListCtrlEditDeleteShortcuts(t *testing.T) {
	app, store := testApp(t)
	ctx := context.Background()
	acct, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, acct.ID, "2026-06-01", "150.00", "end"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, acct.ID, "2026-05-01", "100.00", "start"); err != nil {
		t.Fatal(err)
	}

	app = appWithNav(app,
		navFrame{Path: "/", Menu: 0},
		navFrame{Path: "/accounts/cash/", Menu: 0},
		navFrame{Path: "/accounts/cash/balances/list/", Menu: 1},
	)
	app = press(app, tea.KeyCtrlE)
	if app.Path != "/accounts/cash/balances/2026-05-01/edit/" {
		t.Fatalf("ctrl+e on balance list should open selected edit form, got %s", app.Path)
	}
	if app.Form["date"] != "2026-05-01" || app.Form["balance"] != "100.00" || app.Form["notes"] != "start" {
		t.Fatalf("ctrl+e should populate selected balance form, got %#v", app.Form)
	}

	app = appWithNav(app,
		navFrame{Path: "/", Menu: 0},
		navFrame{Path: "/accounts/cash/", Menu: 0},
		navFrame{Path: "/accounts/cash/balances/list/", Menu: 1},
	)
	app = press(app, tea.KeyCtrlD)
	if app.Path != "/accounts/cash/balances/list/" {
		t.Fatalf("ctrl+d on balance list should stay on list, got %s", app.Path)
	}
	if _, err := store.Bal.GetByAccountDate(ctx, acct.ID, "2026-05-01"); err == nil {
		t.Fatal("ctrl+d should delete the selected balance")
	}
	if len(app.History) != 1 {
		t.Fatalf("ctrl+d should append undo history, got %d entries", len(app.History))
	}
	view := app.View()
	if !strings.Contains(view, "> 2026-06-01") || strings.Contains(view, "| HKD 100.00 | start") {
		t.Fatalf("ctrl+d should clamp selection to remaining balance:\n%s", view)
	}
}

func TestBalanceListEditSubmitReturnsToList(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	acct, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, acct.ID, "2026-06-01", "150.00", "end"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, acct.ID, "2026-05-01", "100.00", "start"); err != nil {
		t.Fatal(err)
	}

	app = appWithNav(app,
		navFrame{Path: "/", Menu: 0},
		navFrame{Path: "/accounts/cash/", Menu: 0},
		navFrame{Path: "/accounts/cash/balances/list/", Menu: 1},
	)
	app = press(app, tea.KeyCtrlE)
	app.Form["date"] = "2026-05-15"
	app.Form["notes"] = "updated"
	app = press(app, tea.KeyCtrlS)
	if app.Path != "/accounts/cash/balances/list/" {
		t.Fatalf("ctrl+s from list-launched balance edit should return to list, got %s", app.Path)
	}
	view := app.View()
	if !strings.Contains(view, "> 2026-05-15") || !strings.Contains(view, "updated") {
		t.Fatalf("updated balance should be selected in list:\n%s", view)
	}
	if strings.Contains(view, "2026-05-01") {
		t.Fatalf("old balance date should not remain in list:\n%s", view)
	}
}

func TestBalanceListEditConfirmReturnsToList(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	acct, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, acct.ID, "2026-05-01", "100.00", "start"); err != nil {
		t.Fatal(err)
	}

	app = appWithNav(app,
		navFrame{Path: "/", Menu: 0},
		navFrame{Path: "/accounts/cash/", Menu: 0},
		navFrame{Path: "/accounts/cash/balances/list/", Menu: 0},
	)
	app = press(app, tea.KeyCtrlE)
	app.Form["notes"] = "confirmed"
	app.Field = 3
	app = press(app, tea.KeyEnter)
	if app.Path != "/accounts/cash/balances/list/" {
		t.Fatalf("confirm from list-launched balance edit should return to list, got %s", app.Path)
	}
	if view := app.View(); !strings.Contains(view, "> 2026-05-01") || !strings.Contains(view, "confirmed") {
		t.Fatalf("confirmed balance edit should be visible on list:\n%s", view)
	}
}

func TestAccountListFilterHLDoesNotTriggerBackOrOpen(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	if _, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, ""); err != nil {
		t.Fatal(err)
	}
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: "/accounts/list/", Menu: 0})
	app = pressRunes(app, "h")
	if app.Path != "/accounts/list/" {
		t.Fatalf("h on account list should type into filter, not go back/open, got %s", app.Path)
	}
	if !strings.Contains(app.View(), "> filter : h") {
		t.Fatalf("h should appear in filter:\n%s", app.View())
	}
}

func TestCtrlNFromAccountListsOpensCreate(t *testing.T) {
	app, _ := testApp(t)
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: "/accounts/list/", Menu: 0})
	app = press(app, tea.KeyCtrlN)
	if app.Path != "/accounts/create/" {
		t.Fatalf("ctrl+n on account list should open account create, got %s", app.Path)
	}
	if app.Field != 0 {
		t.Fatalf("account create should start on first field, got field=%d", app.Field)
	}

	app, _ = testApp(t)
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: "/accounts/list/", Menu: 0})
	app = press(app, tea.KeyCtrlH)
	app = press(app, tea.KeyCtrlN)
	if app.Path != "/accounts/create/" {
		t.Fatalf("ctrl+n on hidden-only account list should open account create, got %s", app.Path)
	}
}

func TestCtrlTFromAccountListOpensTagList(t *testing.T) {
	app, _ := testApp(t)
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: "/accounts/list/", Menu: 0})
	app = press(app, tea.KeyCtrlT)
	if app.Path != routeTagList {
		t.Fatalf("ctrl+t on account list should open tag list, got %s", app.Path)
	}
	assertViewContains(t, app.View(), "/tags/list/", "name | notes", "(no tags yet)", "ctrl+n")
}

func TestTagRoutesCreateEditAndListSelection(t *testing.T) {
	app, store := testApp(t)
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: routeTagList, Menu: 0})
	app = press(app, tea.KeyCtrlN)
	if app.Path != routeTagCreate {
		t.Fatalf("ctrl+n on tag list should open tag create, got %s", app.Path)
	}
	app.Form = map[string]string{"tag-name": "family/shared", "notes": "household tag"}
	app.Field = 2
	app = press(app, tea.KeyEnter)
	if app.Path != routeTagList {
		t.Fatalf("tag create should return to tag list, got %s\n%s", app.Path, app.View())
	}
	assertViewContains(t, app.View(), "family/shared", "household tag")
	tag, err := store.Tag.GetByName(context.Background(), "family/shared")
	if err != nil {
		t.Fatal(err)
	}
	app = press(app, tea.KeyEnter)
	if app.Path != tagEditPathFor("family/shared") {
		t.Fatalf("enter on tag list should open tag edit, got %s", app.Path)
	}
	app.Form = map[string]string{"tag-name": "family/core", "notes": "renamed"}
	app.Field = 2
	app = press(app, tea.KeyEnter)
	if app.Path != routeTagList {
		t.Fatalf("tag edit should return to tag list, got %s", app.Path)
	}
	if _, err := store.Tag.GetByName(context.Background(), "family/shared"); err == nil {
		t.Fatal("old tag name should not remain after rename")
	}
	renamed, err := store.Tag.GetByName(context.Background(), "family/core")
	if err != nil {
		t.Fatal(err)
	}
	if renamed.ID != tag.ID {
		t.Fatal("tag rename should preserve id")
	}
	assertViewContains(t, app.View(), "family/core", "renamed")
}

func TestTagListNavigationCanSelectRowsAfterFirst(t *testing.T) {
	app, store := testApp(t)
	if _, err := store.Tag.Create(context.Background(), "family", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Tag.Create(context.Background(), "wallet", ""); err != nil {
		t.Fatal(err)
	}
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: routeTagList, Menu: 0})

	app = press(app, tea.KeyDown)
	if view := app.View(); !strings.Contains(view, "> wallet") {
		t.Fatalf("down should move tag list selection to second row:\n%s", view)
	}
	app = press(app, tea.KeyEnter)
	if app.Path != tagEditPathFor("wallet") {
		t.Fatalf("enter should open selected tag, got %s", app.Path)
	}
}

func TestAccountCreateTagFieldInlineCreateAndFilter(t *testing.T) {
	app, store := testApp(t)
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: "/accounts/list/", Menu: 0}, navFrame{Path: "/accounts/create/", Menu: 0})
	app.Form = map[string]string{"name": "cash", "currency": "HKD", "on-budget": "true"}
	app.Field = 4
	app = pressRunes(app, "family/shared")
	app = press(app, tea.KeyEnter)
	if app.Form["tags"] != "family/shared" || app.Form[newTagsKey] != "family/shared" {
		t.Fatalf("inline tag should be selected as draft-created, form=%#v", app.Form)
	}
	app = press(app, tea.KeyEnter)
	if app.Path != routeAccountList {
		t.Fatalf("second enter on empty tag filter should submit account, got %s\n%s", app.Path, app.View())
	}
	acct, err := store.Acct.GetByName(context.Background(), "cash")
	if err != nil {
		t.Fatal(err)
	}
	tags, err := store.Tag.ListEffectiveByAccountID(context.Background(), acct.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got := tagNames(tags); strings.Join(got, ",") != "family/shared" {
		t.Fatalf("account effective tags = %v", got)
	}
	assertViewContains(t, app.View(), "tags", "family/shared")
	app = pressRunes(app, "tag:family/shared")
	if view := app.View(); strings.Contains(view, "filtered total") || !strings.Contains(view, "> cash") {
		t.Fatalf("tag filter should match account without relabeling totals:\n%s", view)
	}
}

func TestSelectedExistingTagDoesNotShowCreateOption(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	if _, _, err := app.Svc.Tags.Create(ctx, "wallet", ""); err != nil {
		t.Fatal(err)
	}
	app.Form = map[string]string{"tags": "wallet"}
	app.setTagFilter("wallet")
	options := app.currentTagOptions()
	if len(options) != 0 {
		t.Fatalf("selected existing tag should not appear as selectable or creatable, got %#v", options)
	}
}

func TestPlainNStillTypesIntoAccountListFilter(t *testing.T) {
	app, _ := testApp(t)
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: "/accounts/list/", Menu: 0})
	app = pressRunes(app, "n")
	if app.Path != "/accounts/list/" {
		t.Fatalf("plain n should stay on account list, got %s", app.Path)
	}
	if !strings.Contains(app.View(), "> filter : n") {
		t.Fatalf("plain n should appear in filter:\n%s", app.View())
	}
}

func TestCtrlEFromAccountListOpensEdit(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	if _, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, "main cash"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Accounts.Create(ctx, "savings", "USD", false, "rainy day"); err != nil {
		t.Fatal(err)
	}
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: "/accounts/list/", Menu: 1})
	app = press(app, tea.KeyCtrlE)
	if app.Path != "/accounts/savings/edit/" {
		t.Fatalf("ctrl+e on account list should open selected account edit, got %s", app.Path)
	}
	for key, want := range map[string]string{"name": "savings", "currency": "USD", "on-budget": "false", "notes": "rainy day"} {
		if app.Form[key] != want {
			t.Fatalf("edit form %s = %q, want %q; form=%#v", key, app.Form[key], want, app.Form)
		}
	}
}

func TestAccountListEditSubmitReturnsToList(t *testing.T) {
	app, store := testApp(t)
	ctx := context.Background()
	if _, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, "main cash"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Accounts.Create(ctx, "savings", "USD", false, "rainy day"); err != nil {
		t.Fatal(err)
	}
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: "/accounts/list/", Menu: 1})
	app = press(app, tea.KeyCtrlE)
	app.Form["notes"] = "updated note"
	app = press(app, tea.KeyCtrlS)
	if app.Path != "/accounts/list/" {
		t.Fatalf("ctrl+s from list-launched account edit should return to list, got %s", app.Path)
	}
	if view := app.View(); !strings.Contains(view, "> savings") || !strings.Contains(view, "updated note") {
		t.Fatalf("updated account should be selected in list:\n%s", view)
	}
	acct, err := store.Acct.GetByName(ctx, "savings")
	if err != nil {
		t.Fatal(err)
	}
	if acct.Notes != "updated note" {
		t.Fatalf("account notes = %q", acct.Notes)
	}
}

func TestAccountListEditConfirmReturnsToList(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	if _, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, "main cash"); err != nil {
		t.Fatal(err)
	}
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: "/accounts/list/", Menu: 0})
	app = press(app, tea.KeyCtrlE)
	app.Form["notes"] = "confirmed note"
	app.Field = 5
	app = press(app, tea.KeyEnter)
	if app.Path != "/accounts/list/" {
		t.Fatalf("confirm from list-launched account edit should return to list, got %s", app.Path)
	}
	if view := app.View(); !strings.Contains(view, "> cash") || !strings.Contains(view, "confirmed note") {
		t.Fatalf("confirmed edit should be visible on list:\n%s", view)
	}
}

func TestAccountListEditPreservesFilterAndVisibility(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	if _, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, "visible"); err != nil {
		t.Fatal(err)
	}
	hidden, _, err := app.Svc.Accounts.Create(ctx, "old-account", "HKD", true, "closed")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Accounts.SetHidden(ctx, hidden.ID, true); err != nil {
		t.Fatal(err)
	}
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: "/accounts/list/", Menu: 0})
	app.AccountVisible = accountVisibilityHiddenOnly
	app.Form[formKeyFilter] = "old"
	app = press(app, tea.KeyCtrlE)
	app.Form["name"] = "archived-account"
	app = press(app, tea.KeyCtrlS)
	if app.Path != "/accounts/list/" || app.AccountVisible != accountVisibilityHiddenOnly || app.Form[formKeyFilter] != "old" {
		t.Fatalf("list edit should preserve list state, path=%s mode=%s filter=%q", app.Path, app.AccountVisible.label(), app.Form[formKeyFilter])
	}
	if app.Menu != 0 {
		t.Fatalf("list cursor should clamp after edited account leaves filter, got %d", app.Menu)
	}
	if view := app.View(); !strings.Contains(view, "showing : hidden-only") || !strings.Contains(view, "> filter : old") || !strings.Contains(view, "(no results)") {
		t.Fatalf("filtered hidden list state not preserved:\n%s", view)
	}
}

func TestCtrlNFromBalanceListOpensAdd(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	if _, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, ""); err != nil {
		t.Fatal(err)
	}
	app = appWithNav(app,
		navFrame{Path: "/", Menu: 0},
		navFrame{Path: "/accounts/cash/", Menu: 0},
		navFrame{Path: "/accounts/cash/balances/list/", Menu: 0},
	)
	app = press(app, tea.KeyCtrlN)
	if app.Path != "/accounts/cash/balances/add/" {
		t.Fatalf("ctrl+n on balance list should open add balance, got %s", app.Path)
	}
	if app.Form["date"] != Today() || app.Field != 0 {
		t.Fatalf("balance add defaults not initialized: form=%#v field=%d", app.Form, app.Field)
	}
}

func TestCurrencySelectHLDoesNotPaginate(t *testing.T) {
	app, store := testApp(t)
	ctx := context.Background()
	for _, code := range []string{"AUD", "BRL", "CAD", "CHF", "CNY", "INR", "KRW", "MXN", "NZD", "SGD", "THB"} {
		if err := store.UpsertCurrencyNameOnly(ctx, code, code); err != nil {
			t.Fatal(err)
		}
	}
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: "/accounts/create/", Menu: 0})
	app.Field = 1
	app = press(app, tea.KeyRight)
	view := app.View()
	if !strings.Contains(view, "     > EUR") || !strings.Contains(view, "[09/30]") {
		t.Fatalf("right should paginate unfiltered currency options:\n%s", view)
	}
	app = pressRunes(app, "h")
	view = app.View()
	if !strings.Contains(view, "> filter  : H") {
		t.Fatalf("h should type into currency filter instead of paginating:\n%s", view)
	}
	if strings.Contains(view, "     > EUR") || !strings.Contains(view, "     > HKD") {
		t.Fatalf("h filter should reset to filtered page 1, not keep unfiltered page 2 selection:\n%s", view)
	}
	app = pressRunes(app, "l")
	view = app.View()
	if !strings.Contains(view, "> filter  : HL") {
		t.Fatalf("l should append to currency filter, not paginate or open:\n%s", view)
	}
}

func TestTextFieldHLAndCaretIndependence(t *testing.T) {
	app, _ := testApp(t)
	app.Path = "/accounts/create/"
	app.Field = 3
	app = pressRunes(app, "oh")
	if app.Form["notes"] != "oh" {
		t.Fatalf("expected notes oh, got %q", app.Form["notes"])
	}
	app = press(app, tea.KeyLeft)
	if !strings.Contains(app.View(), "notes    : o|h") {
		t.Fatalf("left should move caret independently of h/l typing:\n%s", app.View())
	}
}

func TestBackupHorizontalKeys(t *testing.T) {
	app, _ := testApp(t)
	app = pressRunes(app, "7")
	if app.Path != routeBackup {
		t.Fatalf("expected backup screen, got %s", app.Path)
	}
	app = press(app, tea.KeyRight)
	if app.LastBackup == "" {
		t.Fatal("right/l should trigger backup like enter")
	}
	app = press(app, tea.KeyLeft)
	if app.Path != "/" {
		t.Fatalf("left/h should go back from backup, got %s", app.Path)
	}
}

func TestSettingsHorizontalBack(t *testing.T) {
	app, _ := testApp(t)
	app = pressRunes(app, "6")
	if app.Path != routeSettings {
		t.Fatalf("expected settings screen, got %s", app.Path)
	}
	app = pressRunes(app, "h")
	if app.Path != "/" {
		t.Fatalf("left/h should go back from settings, got %s", app.Path)
	}
}

func TestNavigationHelpFooters(t *testing.T) {
	app, _ := testApp(t)
	assertViewContains(t, app.View(), "left/h        : back", "right/l       : open")

	app = pressRunes(app, "1")
	assertViewContains(t, app.View(), "h/l           : type in filter", "left/right    : back/open", "ctrl+n        : new")

	ctx := context.Background()
	acct, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, acct.ID, "2026-06-01", "150.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, acct.ID, "2026-05-01", "100.00", ""); err != nil {
		t.Fatal(err)
	}
	app = appWithNav(app,
		navFrame{Path: "/", Menu: 0},
		navFrame{Path: "/accounts/cash/", Menu: 0},
		navFrame{Path: "/accounts/cash/balances/list/", Menu: 0},
	)
	assertViewContains(t, app.View(), "left/right    : back/open", "ctrl+n        : new")

	app = appWithNav(app,
		navFrame{Path: "/", Menu: 0},
		navFrame{Path: "/accounts/cash/balances/list/", Menu: 0},
		navFrame{Path: "/accounts/cash/balances/2026-06-01/", Menu: 0},
	)
	assertViewContains(t, app.View(), "left/h      : older")
}

func TestBalanceDetailLateralNavPreservesMenuSelection(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	acct, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, acct.ID, "2026-06-01", "150.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, acct.ID, "2026-05-01", "100.00", ""); err != nil {
		t.Fatal(err)
	}
	app = appWithNav(app,
		navFrame{Path: "/", Menu: 0},
		navFrame{Path: "/accounts/cash/balances/list/", Menu: 0},
		navFrame{Path: "/accounts/cash/balances/2026-06-01/", Menu: 1},
	)
	app = pressRunes(app, "h")
	if app.Menu != 1 {
		t.Fatalf("lateral navigation should preserve menu cursor, got %d", app.Menu)
	}
	if !strings.Contains(app.View(), "> 2) delete balance") {
		t.Fatalf("menu selection should stay on delete after lateral nav:\n%s", app.View())
	}
}

func TestAccountCreateCurrencyHelpShowsHLFilterHint(t *testing.T) {
	app, _ := testApp(t)
	app = appWithNav(app, navFrame{Path: "/", Menu: 0}, navFrame{Path: "/accounts/create/", Menu: 0})
	app.Field = 1
	assertViewContains(t, app.View(), "h/l        : type in filter", "left/right : next/prev page")
}

func TestBalanceDetailAtBoundaryEscStillGoesBack(t *testing.T) {
	app, _ := testApp(t)
	ctx := context.Background()
	acct, _, err := app.Svc.Accounts.Create(ctx, "cash", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Svc.Balances.Add(ctx, acct.ID, "2026-05-01", "100.00", ""); err != nil {
		t.Fatal(err)
	}
	app = appWithNav(app,
		navFrame{Path: "/", Menu: 0},
		navFrame{Path: "/accounts/cash/balances/list/", Menu: 0},
		navFrame{Path: "/accounts/cash/balances/2026-05-01/", Menu: 0},
	)
	app = pressRunes(app, "h")
	if app.Path != "/accounts/cash/balances/2026-05-01/" {
		t.Fatalf("left/h at oldest boundary should stay on detail, got %s", app.Path)
	}
	app = press(app, tea.KeyEsc)
	if app.Path != "/accounts/cash/balances/list/" {
		t.Fatalf("esc should still go back from balance detail at boundary, got %s", app.Path)
	}
}
