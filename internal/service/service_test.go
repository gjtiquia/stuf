package service

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"stuf/internal/money"
	"stuf/internal/repo"
)

func serviceStack(t *testing.T) (*repo.Store, AccountService, BalanceService, DashboardService, HistoryService) {
	t.Helper()
	ctx := context.Background()
	s, err := repo.Open(ctx, filepath.Join(t.TempDir(), "db.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	s.Clock = func() time.Time { return time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC) }
	h := HistoryService{Repo: s.Hist, Now: s.Clock}
	a := AccountService{Store: s, Accounts: s.Acct, Balances: s.Bal, Currency: s.Cur, History: h, AppCurrency: "HKD"}
	b := BalanceService{Store: s, Accounts: s.Acct, Balances: s.Bal, History: h}
	d := DashboardService{Accounts: s.Acct, Balances: s.Bal, Currencies: s.Cur, AppCurrency: "HKD", Now: s.Clock}
	return s, a, b, d, h
}

func setServiceRate(t *testing.T, store *repo.Store, code string, amount int64, scale int) {
	t.Helper()
	if err := store.SetCurrencyRate(context.Background(), code, amount, scale); err != nil {
		t.Fatal(err)
	}
}

func TestAccountMutationRecordsHistoryAndUndo(t *testing.T) {
	ctx := context.Background()
	s, accounts, _, _, history := serviceStack(t)
	a, entry, err := accounts.Create(ctx, "cash", "", true, "")
	if err != nil {
		t.Fatal(err)
	}
	rows, _ := s.Hist.List(ctx)
	if len(rows) != 1 || entry.Action != "create" {
		t.Fatalf("history not recorded: %+v %+v", rows, entry)
	}
	if err := history.Undo(ctx, entry); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Acct.GetByID(ctx, a.ID); err == nil {
		t.Fatal("account still exists after undo")
	}
	rows, _ = s.Hist.List(ctx)
	if len(rows) != 0 {
		t.Fatalf("persisted history not deleted: %+v", rows)
	}
}

func TestAccountDuplicateNameReturnsDomainError(t *testing.T) {
	ctx := context.Background()
	_, accounts, _, _, _ := serviceStack(t)
	cash, _, err := accounts.Create(ctx, "cash", "", true, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := accounts.Create(ctx, "cash", "", true, "duplicate"); err == nil {
		t.Fatal("expected duplicate account name error")
	} else {
		var dup *repo.AccountDuplicateNameError
		if !errors.As(err, &dup) {
			t.Fatalf("expected duplicate account name domain error, got %T %[1]v", err)
		}
		if dup.Name != "cash" {
			t.Fatalf("duplicate account name = %q", dup.Name)
		}
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			t.Fatalf("duplicate account error should hide raw sqlite error: %v", err)
		}
	}
	savings, _, err := accounts.Create(ctx, "savings", "", true, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := accounts.Update(ctx, savings.ID, "cash", "HKD", true, false, "collides"); err == nil {
		t.Fatal("expected duplicate account name error on update")
	} else {
		var dup *repo.AccountDuplicateNameError
		if !errors.As(err, &dup) {
			t.Fatalf("expected duplicate account name domain error on update, got %T %[1]v", err)
		}
		if dup.Name != "cash" {
			t.Fatalf("update duplicate account name = %q", dup.Name)
		}
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			t.Fatalf("update duplicate account error should hide raw sqlite error: %v", err)
		}
	}
	if got, err := accounts.GetByName(ctx, "cash"); err != nil || got.ID != cash.ID {
		t.Fatalf("original account should remain, got %+v err=%v", got, err)
	}
}

func TestAccountInvalidCurrencyReturnsFriendlyError(t *testing.T) {
	ctx := context.Background()
	_, accounts, _, _, _ := serviceStack(t)
	if _, _, err := accounts.Create(ctx, "cash", "ZZZ", true, ""); err == nil {
		t.Fatal("expected invalid currency error")
	} else if got := err.Error(); got != "currency is unavailable: ZZZ" {
		t.Fatalf("invalid currency error = %q", got)
	}
	acct, _, err := accounts.Create(ctx, "cash", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := accounts.Update(ctx, acct.ID, "cash", "ZZZ", true, false, ""); err == nil {
		t.Fatal("expected invalid currency update error")
	} else if got := err.Error(); got != "currency is unavailable: ZZZ" {
		t.Fatalf("invalid currency update error = %q", got)
	}
}

func TestChildAccountInheritsOnBudgetAndCannotDiverge(t *testing.T) {
	ctx := context.Background()
	_, accounts, _, _, _ := serviceStack(t)
	parent, _, err := accounts.Create(ctx, "investment", "HKD", false, "")
	if err != nil {
		t.Fatal(err)
	}
	child, _, err := accounts.CreateChild(ctx, parent.ID, "investment-hkd", "HKD", "")
	if err != nil {
		t.Fatal(err)
	}
	if child.OnBudget {
		t.Fatalf("child should inherit off-budget parent: %+v", child)
	}
	if _, _, err := accounts.Update(ctx, child.ID, child.Name, child.Code, true, child.Hidden, child.Notes); err == nil {
		t.Fatal("expected child on-budget divergence to be rejected")
	}
	updated, _, err := accounts.Update(ctx, parent.ID, parent.Name, parent.Code, true, parent.Hidden, parent.Notes)
	if err != nil {
		t.Fatal(err)
	}
	if !updated.OnBudget {
		t.Fatal("parent should update to on-budget")
	}
	child, err = accounts.GetByName(ctx, "investment-hkd")
	if err != nil {
		t.Fatal(err)
	}
	if !child.OnBudget {
		t.Fatal("child should cascade to parent on-budget value")
	}
}

func TestAccountTreeBalancesAvoidDoubleCounting(t *testing.T) {
	ctx := context.Background()
	_, accounts, balances, dashboard, _ := serviceStack(t)
	setServiceRate(t, accounts.Store, "HKD", 1, 0)
	setServiceRate(t, accounts.Store, "USD", 10, 0)
	parent, _, err := accounts.Create(ctx, "investment", "HKD", false, "")
	if err != nil {
		t.Fatal(err)
	}
	usd, _, err := accounts.CreateChild(ctx, parent.ID, "investment-usd", "USD", "")
	if err != nil {
		t.Fatal(err)
	}
	hkd, _, err := accounts.CreateChild(ctx, parent.ID, "investment-hkd", "HKD", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := balances.Add(ctx, parent.ID, "2026-05-24", "500000.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := balances.Add(ctx, usd.ID, "2026-05-24", "32000.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := balances.Add(ctx, hkd.ID, "2026-05-24", "100000.00", ""); err != nil {
		t.Fatal(err)
	}
	summary, err := accounts.TreeSummary(ctx, parent.ID, "HKD")
	if err != nil {
		t.Fatal(err)
	}
	assertMoneyAmount(t, "parent display", summary.Balance, 50000000)
	assertMoneyAmount(t, "children", summary.Children, 42000000)
	assertMoneyAmount(t, "remaining", summary.Remaining, 8000000)

	d, err := dashboard.Summary(ctx)
	if err != nil {
		t.Fatal(err)
	}
	assertMoneyAmount(t, "dashboard excludes off-budget parent", d.Total, 0)

	if _, _, err := accounts.Update(ctx, parent.ID, parent.Name, parent.Code, true, parent.Hidden, parent.Notes); err != nil {
		t.Fatal(err)
	}
	d, err = dashboard.Summary(ctx)
	if err != nil {
		t.Fatal(err)
	}
	assertMoneyAmount(t, "dashboard counts parent once", d.Total, 50000000)
}

func TestAccountTreeWithoutOwnBalanceDerivesFromChildren(t *testing.T) {
	ctx := context.Background()
	_, accounts, balances, dashboard, _ := serviceStack(t)
	setServiceRate(t, accounts.Store, "HKD", 1, 0)
	setServiceRate(t, accounts.Store, "USD", 10, 0)
	parent, _, err := accounts.Create(ctx, "hsbc-one", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	hkd, _, err := accounts.CreateChild(ctx, parent.ID, "hsbc-hkd", "HKD", "")
	if err != nil {
		t.Fatal(err)
	}
	usd, _, err := accounts.CreateChild(ctx, parent.ID, "hsbc-usd", "USD", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := balances.Add(ctx, hkd.ID, "2026-05-21", "35000.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := balances.Add(ctx, usd.ID, "2026-05-24", "1000.00", ""); err != nil {
		t.Fatal(err)
	}
	summary, err := accounts.TreeSummary(ctx, parent.ID, "HKD")
	if err != nil {
		t.Fatal(err)
	}
	assertMoneyAmount(t, "derived display", summary.Balance, 4500000)
	assertMoneyAmount(t, "children", summary.Children, 4500000)
	assertMoneyAmount(t, "remaining", summary.Remaining, 0)
	if summary.AsOf != "2026-05-24" {
		t.Fatalf("as of = %s", summary.AsOf)
	}
	d, err := dashboard.Summary(ctx)
	if err != nil {
		t.Fatal(err)
	}
	assertMoneyAmount(t, "dashboard derived parent", d.Total, 4500000)
}

func TestDeleteEmptyAccountUndo(t *testing.T) {
	ctx := context.Background()
	s, accounts, balances, _, history := serviceStack(t)
	parent, _, err := accounts.Create(ctx, "cash", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	child, _, err := accounts.CreateChild(ctx, parent.ID, "cash-child", "HKD", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := accounts.DeleteEmpty(ctx, parent.ID); err == nil {
		t.Fatal("expected parent with child delete to fail")
	}
	if _, _, err := balances.Add(ctx, child.ID, "2026-05-24", "1.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := accounts.DeleteEmpty(ctx, child.ID); err == nil {
		t.Fatal("expected child with balance delete to fail")
	}
	bal, err := balances.GetByAccountDate(ctx, child.ID, "2026-05-24")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := balances.Delete(ctx, bal.ID); err != nil {
		t.Fatal(err)
	}
	deleted, entry, err := accounts.DeleteEmpty(ctx, child.ID)
	if err != nil {
		t.Fatal(err)
	}
	if deleted.ID != child.ID {
		t.Fatalf("deleted = %+v", deleted)
	}
	if _, err := s.Acct.GetByID(ctx, child.ID); err == nil {
		t.Fatal("child still exists after delete")
	}
	if err := history.Undo(ctx, entry); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Acct.GetByID(ctx, child.ID); err != nil {
		t.Fatalf("undo should restore child: %v", err)
	}
}

func TestBalanceMutationsAndUndo(t *testing.T) {
	ctx := context.Background()
	s, accounts, balances, _, history := serviceStack(t)
	a, _, err := accounts.Create(ctx, "cash", "", true, "")
	if err != nil {
		t.Fatal(err)
	}
	b, entry, err := balances.Add(ctx, a.ID, "2026-05-24", "10.50", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := history.Undo(ctx, entry); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Bal.GetByID(ctx, b.ID); err == nil {
		t.Fatal("balance still exists after undo")
	}
}

func TestBalanceDuplicateDateReturnsDomainError(t *testing.T) {
	ctx := context.Background()
	_, accounts, balances, _, _ := serviceStack(t)
	a, _, err := accounts.Create(ctx, "cash", "", true, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := balances.Add(ctx, a.ID, "2026-05-24", "10.50", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := balances.Add(ctx, a.ID, "2026-05-24", "11.00", "duplicate"); err == nil {
		t.Fatal("expected duplicate balance date error")
	} else {
		var dup *repo.BalanceDuplicateDateError
		if !errors.As(err, &dup) {
			t.Fatalf("expected duplicate date domain error, got %T %[1]v", err)
		}
		if dup.Date != "2026-05-24" {
			t.Fatalf("duplicate error date = %q", dup.Date)
		}
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			t.Fatalf("duplicate error should hide raw sqlite error: %v", err)
		}
	}
	if _, _, err := balances.Add(ctx, a.ID, "2026-05-25", "12.00", "second"); err != nil {
		t.Fatal(err)
	}
	first, err := balances.GetByAccountDate(ctx, a.ID, "2026-05-24")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := balances.Update(ctx, first.ID, "2026-05-25", "12.50", "collides"); err == nil {
		t.Fatal("expected duplicate balance date error on update")
	} else {
		var dup *repo.BalanceDuplicateDateError
		if !errors.As(err, &dup) {
			t.Fatalf("expected duplicate date domain error on update, got %T %[1]v", err)
		}
		if dup.Date != "2026-05-25" {
			t.Fatalf("update duplicate error date = %q", dup.Date)
		}
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			t.Fatalf("update duplicate error should hide raw sqlite error: %v", err)
		}
	}
}

func TestDashboardSingleAccountTotalUsesLatestAndStartUsesAsOfBoundary(t *testing.T) {
	ctx := context.Background()
	_, accounts, balances, dashboard, _ := serviceStack(t)
	dashboard.Now = func() time.Time { return time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC) }
	a, _, err := accounts.Create(ctx, "checking", "", true, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := balances.Add(ctx, a.ID, "2026-03-30", "6000.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := balances.Add(ctx, a.ID, "2026-04-08", "3500.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := balances.Add(ctx, a.ID, "2026-05-25", "1300.00", ""); err != nil {
		t.Fatal(err)
	}

	summary, err := dashboard.Summary(ctx)
	if err != nil {
		t.Fatal(err)
	}
	assertMoneyAmount(t, "total", summary.Total, 130000)
	assertMoneyAmount(t, "from month start", summary.NetChangeFromMonthStart, -220000)
	assertMoneyAmount(t, "from previous month high", summary.NetChangeFromPreviousMonthHigh, -470000)
}

func TestDashboardAsOfBoundaryDoesNotUseNearerFutureSnapshot(t *testing.T) {
	ctx := context.Background()
	_, accounts, balances, dashboard, _ := serviceStack(t)
	dashboard.Now = func() time.Time { return time.Date(2026, 5, 28, 12, 0, 0, 0, time.UTC) }
	acct, _, err := accounts.Create(ctx, "usd-savings", "USD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range []struct {
		date   string
		amount string
	}{
		{"2026-04-01", "268.40"},
		{"2026-05-13", "48.64"},
		{"2026-05-19", "246.64"},
		{"2026-05-28", "246.64"},
	} {
		if _, _, err := balances.Add(ctx, acct.ID, row.date, row.amount, ""); err != nil {
			t.Fatal(err)
		}
	}

	summary, err := dashboard.AccountSummary(ctx, acct.ID)
	if err != nil {
		t.Fatal(err)
	}
	assertMoneyAmount(t, "total", summary.Total, 24664)
	assertMoneyAmount(t, "from month start", summary.NetChangeFromMonthStart, -2176)
	assertMoneyAmount(t, "from month high", summary.NetChangeFromMonthHigh, -2176)
	assertMoneyAmount(t, "from previous month high", summary.NetChangeFromPreviousMonthHigh, -2176)
	assertMoneyAmount(t, "apr drop", summary.RecentMonths[0].Drop, 0)
	assertMoneyAmount(t, "mar drop", summary.RecentMonths[1].Drop, 0)
	assertMoneyAmount(t, "high trend", summary.Trend.HighToHigh, 0)
	assertMoneyAmount(t, "low trend", summary.Trend.LowToLow, 0)
}

func TestDashboardSummaryExposesAsOfDate(t *testing.T) {
	ctx := context.Background()
	_, _, _, dashboard, _ := serviceStack(t)
	dashboard.Now = func() time.Time { return time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC) }
	summary, err := dashboard.Summary(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if summary.AsOf != "2026-05-25" {
		t.Fatalf("as of = %s", summary.AsOf)
	}
}

func TestDashboardAccountSummaryUsesNativeAccountCurrency(t *testing.T) {
	ctx := context.Background()
	_, accounts, balances, dashboard, _ := serviceStack(t)
	dashboard.Now = func() time.Time { return time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC) }
	acct, _, err := accounts.Create(ctx, "checking", "HKD", false, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := accounts.SetHidden(ctx, acct.ID, true); err != nil {
		t.Fatal(err)
	}
	if _, _, err := balances.Add(ctx, acct.ID, "2026-04-08", "3500.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := balances.Add(ctx, acct.ID, "2026-05-25", "1300.00", ""); err != nil {
		t.Fatal(err)
	}
	summary, err := dashboard.AccountSummary(ctx, acct.ID)
	if err != nil {
		t.Fatal(err)
	}
	if summary.AsOf != "2026-05-25" {
		t.Fatalf("as of = %s", summary.AsOf)
	}
	assertMoneyAmount(t, "account total", summary.Total, 130000)
	assertMoneyAmount(t, "account month start", summary.NetChangeFromMonthStart, -220000)
	assertMoneyAmount(t, "account month high", summary.NetChangeFromMonthHigh, -220000)
}

func TestDashboardSingleSnapshotCarriesFlatBeforeFirstBalance(t *testing.T) {
	ctx := context.Background()
	_, accounts, balances, dashboard, _ := serviceStack(t)
	dashboard.Now = func() time.Time { return time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC) }
	acct, _, err := accounts.Create(ctx, "checking", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := balances.Add(ctx, acct.ID, "2026-05-25", "500.00", ""); err != nil {
		t.Fatal(err)
	}

	summary, err := dashboard.Summary(ctx)
	if err != nil {
		t.Fatal(err)
	}
	assertMoneyAmount(t, "total", summary.Total, 50000)
	assertMoneyAmount(t, "from month start", summary.NetChangeFromMonthStart, 0)
	assertMoneyAmount(t, "from previous month high", summary.NetChangeFromPreviousMonthHigh, 0)
	assertMoneyAmount(t, "apr drop", summary.RecentMonths[0].Drop, 0)
	assertMoneyAmount(t, "mar drop", summary.RecentMonths[1].Drop, 0)
	assertMoneyAmount(t, "high trend", summary.Trend.HighToHigh, 0)
	assertMoneyAmount(t, "low trend", summary.Trend.LowToLow, 0)
}

func TestDashboardAccountSummaryParentUsesChildrenPlusRemaining(t *testing.T) {
	ctx := context.Background()
	_, accounts, balances, dashboard, _ := serviceStack(t)
	dashboard.Now = func() time.Time { return time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC) }
	parent, _, err := accounts.Create(ctx, "checking", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	child, _, err := accounts.CreateChild(ctx, parent.ID, "checking-card", "HKD", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range []struct {
		accountID int64
		date      string
		amount    string
	}{
		{parent.ID, "2026-04-10", "1000.00"},
		{parent.ID, "2026-05-25", "900.00"},
		{child.ID, "2026-04-05", "400.00"},
		{child.ID, "2026-04-20", "300.00"},
		{child.ID, "2026-05-10", "350.00"},
		{child.ID, "2026-05-25", "600.00"},
	} {
		if _, _, err := balances.Add(ctx, row.accountID, row.date, row.amount, ""); err != nil {
			t.Fatal(err)
		}
	}

	summary, err := dashboard.AccountSummary(ctx, parent.ID)
	if err != nil {
		t.Fatal(err)
	}
	assertMoneyAmount(t, "total", summary.Total, 90000)
	assertMoneyAmount(t, "from month start", summary.NetChangeFromMonthStart, 0)
	assertMoneyAmount(t, "from previous month high", summary.NetChangeFromPreviousMonthHigh, -10000)
	assertMoneyAmount(t, "apr drop", summary.RecentMonths[0].Drop, -10000)
}

func TestDashboardAccountSummaryParentNewChildrenDoNotCreateFakeMonthStartDrop(t *testing.T) {
	ctx := context.Background()
	_, accounts, balances, dashboard, _ := serviceStack(t)
	dashboard.Now = func() time.Time { return time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC) }
	parent, _, err := accounts.Create(ctx, "checking", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	first, _, err := accounts.CreateChild(ctx, parent.ID, "checking-card", "HKD", "")
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := accounts.CreateChild(ctx, parent.ID, "checking-savings", "HKD", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range []struct {
		accountID int64
		date      string
		amount    string
	}{
		{parent.ID, "2026-05-09", "7600.00"},
		{parent.ID, "2026-05-25", "7300.00"},
		{first.ID, "2026-05-25", "5000.00"},
		{second.ID, "2026-05-25", "2000.00"},
	} {
		if _, _, err := balances.Add(ctx, row.accountID, row.date, row.amount, ""); err != nil {
			t.Fatal(err)
		}
	}

	summary, err := dashboard.AccountSummary(ctx, parent.ID)
	if err != nil {
		t.Fatal(err)
	}
	assertMoneyAmount(t, "total", summary.Total, 730000)
	assertMoneyAmount(t, "from month start", summary.NetChangeFromMonthStart, -30000)
	assertMoneyAmount(t, "from month high", summary.NetChangeFromMonthHigh, -30000)
}

func TestDashboardParentRemainingUsesChildAsOfAtParentSnapshot(t *testing.T) {
	ctx := context.Background()
	_, accounts, balances, dashboard, _ := serviceStack(t)
	dashboard.Now = func() time.Time { return time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC) }
	parent, _, err := accounts.Create(ctx, "checking", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	child, _, err := accounts.CreateChild(ctx, parent.ID, "checking-card", "HKD", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range []struct {
		accountID int64
		date      string
		amount    string
	}{
		{parent.ID, "2026-05-09", "1000.00"},
		{parent.ID, "2026-05-25", "1000.00"},
		{child.ID, "2026-04-01", "100.00"},
		{child.ID, "2026-05-25", "900.00"},
	} {
		if _, _, err := balances.Add(ctx, row.accountID, row.date, row.amount, ""); err != nil {
			t.Fatal(err)
		}
	}

	summary, err := dashboard.AccountSummary(ctx, parent.ID)
	if err != nil {
		t.Fatal(err)
	}
	assertMoneyAmount(t, "total", summary.Total, 100000)
	assertMoneyAmount(t, "from month start", summary.NetChangeFromMonthStart, 0)
	assertMoneyAmount(t, "from month high", summary.NetChangeFromMonthHigh, -80000)
}

func TestDashboardAccountSummarySparseParentKeepsChildHistory(t *testing.T) {
	ctx := context.Background()
	_, accounts, balances, dashboard, _ := serviceStack(t)
	dashboard.Now = func() time.Time { return time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC) }
	parent, _, err := accounts.Create(ctx, "checking", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	child, _, err := accounts.CreateChild(ctx, parent.ID, "checking-card", "HKD", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range []struct {
		accountID int64
		date      string
		amount    string
	}{
		{parent.ID, "2026-05-25", "1000.00"},
		{child.ID, "2026-03-05", "300.00"},
		{child.ID, "2026-03-20", "250.00"},
		{child.ID, "2026-04-05", "400.00"},
		{child.ID, "2026-04-20", "350.00"},
		{child.ID, "2026-05-25", "600.00"},
	} {
		if _, _, err := balances.Add(ctx, row.accountID, row.date, row.amount, ""); err != nil {
			t.Fatal(err)
		}
	}

	summary, err := dashboard.AccountSummary(ctx, parent.ID)
	if err != nil {
		t.Fatal(err)
	}
	assertMoneyAmount(t, "total", summary.Total, 100000)
	assertMoneyAmount(t, "apr drop", summary.RecentMonths[0].Drop, -15000)
	assertMoneyAmount(t, "mar drop", summary.RecentMonths[1].Drop, -5000)
	assertMoneyAmount(t, "high trend", summary.Trend.HighToHigh, 10000)
	assertMoneyAmount(t, "low trend", summary.Trend.LowToLow, 0)
}

func TestDashboardAccountSummaryNewChildCarriesFlatWhileExistingChildKeepsHistory(t *testing.T) {
	ctx := context.Background()
	_, accounts, balances, dashboard, _ := serviceStack(t)
	dashboard.Now = func() time.Time { return time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC) }
	parent, _, err := accounts.Create(ctx, "checking", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	existing, _, err := accounts.CreateChild(ctx, parent.ID, "checking-card", "HKD", "")
	if err != nil {
		t.Fatal(err)
	}
	newChild, _, err := accounts.CreateChild(ctx, parent.ID, "checking-savings", "HKD", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range []struct {
		accountID int64
		date      string
		amount    string
	}{
		{parent.ID, "2026-05-09", "1000.00"},
		{parent.ID, "2026-05-25", "900.00"},
		{existing.ID, "2026-04-05", "400.00"},
		{existing.ID, "2026-04-20", "350.00"},
		{existing.ID, "2026-05-25", "500.00"},
		{newChild.ID, "2026-05-25", "200.00"},
	} {
		if _, _, err := balances.Add(ctx, row.accountID, row.date, row.amount, ""); err != nil {
			t.Fatal(err)
		}
	}

	summary, err := dashboard.AccountSummary(ctx, parent.ID)
	if err != nil {
		t.Fatal(err)
	}
	assertMoneyAmount(t, "total", summary.Total, 90000)
	assertMoneyAmount(t, "from month start", summary.NetChangeFromMonthStart, -10000)
	assertMoneyAmount(t, "apr drop", summary.RecentMonths[0].Drop, -5000)
}

func TestDashboardAccountSummaryParentWithoutOwnBalanceUsesChildrenOnly(t *testing.T) {
	ctx := context.Background()
	_, accounts, balances, dashboard, _ := serviceStack(t)
	dashboard.Now = func() time.Time { return time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC) }
	parent, _, err := accounts.Create(ctx, "checking", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	first, _, err := accounts.CreateChild(ctx, parent.ID, "checking-card", "HKD", "")
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := accounts.CreateChild(ctx, parent.ID, "checking-savings", "HKD", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range []struct {
		accountID int64
		date      string
		amount    string
	}{
		{first.ID, "2026-04-05", "100.00"},
		{first.ID, "2026-04-20", "80.00"},
		{first.ID, "2026-05-25", "120.00"},
		{second.ID, "2026-04-10", "50.00"},
		{second.ID, "2026-04-25", "30.00"},
		{second.ID, "2026-05-25", "40.00"},
	} {
		if _, _, err := balances.Add(ctx, row.accountID, row.date, row.amount, ""); err != nil {
			t.Fatal(err)
		}
	}

	summary, err := dashboard.AccountSummary(ctx, parent.ID)
	if err != nil {
		t.Fatal(err)
	}
	assertMoneyAmount(t, "total", summary.Total, 16000)
	assertMoneyAmount(t, "from previous month high", summary.NetChangeFromPreviousMonthHigh, 1000)
	assertMoneyAmount(t, "apr drop", summary.RecentMonths[0].Drop, -4000)
}

func TestDashboardAccountSummaryIncludesChangingRemainingHistory(t *testing.T) {
	ctx := context.Background()
	_, accounts, balances, dashboard, _ := serviceStack(t)
	dashboard.Now = func() time.Time { return time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC) }
	parent, _, err := accounts.Create(ctx, "checking", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	child, _, err := accounts.CreateChild(ctx, parent.ID, "checking-card", "HKD", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range []struct {
		accountID int64
		date      string
		amount    string
	}{
		{parent.ID, "2026-03-05", "800.00"},
		{parent.ID, "2026-03-20", "700.00"},
		{parent.ID, "2026-04-05", "1000.00"},
		{parent.ID, "2026-04-20", "900.00"},
		{parent.ID, "2026-05-05", "1200.00"},
		{parent.ID, "2026-05-25", "1000.00"},
		{child.ID, "2026-03-05", "300.00"},
		{child.ID, "2026-03-20", "300.00"},
		{child.ID, "2026-04-05", "400.00"},
		{child.ID, "2026-04-20", "500.00"},
		{child.ID, "2026-05-05", "600.00"},
		{child.ID, "2026-05-25", "700.00"},
	} {
		if _, _, err := balances.Add(ctx, row.accountID, row.date, row.amount, ""); err != nil {
			t.Fatal(err)
		}
	}

	summary, err := dashboard.AccountSummary(ctx, parent.ID)
	if err != nil {
		t.Fatal(err)
	}
	assertMoneyAmount(t, "total", summary.Total, 100000)
	assertMoneyAmount(t, "from month high", summary.NetChangeFromMonthHigh, -30000)
	assertMoneyAmount(t, "apr drop", summary.RecentMonths[0].Drop, -40000)
	assertMoneyAmount(t, "high trend", summary.Trend.HighToHigh, 30000)
	assertMoneyAmount(t, "low trend", summary.Trend.LowToLow, 0)
}

func TestDashboardAccountSummaryNestedChildrenFutureAndHiddenRollup(t *testing.T) {
	ctx := context.Background()
	_, accounts, balances, dashboard, _ := serviceStack(t)
	dashboard.Now = func() time.Time { return time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC) }
	parent, _, err := accounts.Create(ctx, "checking", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	group, _, err := accounts.CreateChild(ctx, parent.ID, "checking-group", "HKD", "")
	if err != nil {
		t.Fatal(err)
	}
	grandchild, _, err := accounts.CreateChild(ctx, group.ID, "checking-grandchild", "HKD", "")
	if err != nil {
		t.Fatal(err)
	}
	leaf, _, err := accounts.CreateChild(ctx, parent.ID, "checking-leaf", "HKD", "")
	if err != nil {
		t.Fatal(err)
	}
	hidden, _, err := accounts.CreateChild(ctx, parent.ID, "checking-hidden", "HKD", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := accounts.SetHidden(ctx, hidden.ID, true); err != nil {
		t.Fatal(err)
	}
	for _, row := range []struct {
		accountID int64
		date      string
		amount    string
	}{
		{parent.ID, "2026-05-25", "1000.00"},
		{parent.ID, "2026-06-01", "5000.00"},
		{grandchild.ID, "2026-05-25", "400.00"},
		{grandchild.ID, "2026-06-01", "999.00"},
		{leaf.ID, "2026-05-24", "300.00"},
		{hidden.ID, "2026-05-25", "900.00"},
	} {
		if _, _, err := balances.Add(ctx, row.accountID, row.date, row.amount, ""); err != nil {
			t.Fatal(err)
		}
	}

	summary, err := dashboard.AccountSummary(ctx, parent.ID)
	if err != nil {
		t.Fatal(err)
	}
	assertMoneyAmount(t, "total", summary.Total, 100000)
	assertMoneyAmount(t, "from month high", summary.NetChangeFromMonthHigh, 0)
}

func TestDashboardFutureOnlyHistoryDoesNotCarryIntoToday(t *testing.T) {
	ctx := context.Background()
	_, accounts, balances, dashboard, _ := serviceStack(t)
	dashboard.Now = func() time.Time { return time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC) }
	acct, _, err := accounts.Create(ctx, "checking", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := balances.Add(ctx, acct.ID, "2026-06-01", "500.00", ""); err != nil {
		t.Fatal(err)
	}

	summary, err := dashboard.Summary(ctx)
	if err != nil {
		t.Fatal(err)
	}
	assertMoneyAmount(t, "total", summary.Total, 0)
	assertMoneyAmount(t, "from month start", summary.NetChangeFromMonthStart, 0)
	assertMoneyAmount(t, "from month high", summary.NetChangeFromMonthHigh, 0)
}

func TestDashboardSingleAccountSameDateSnapshotIsNotFutureInLocalTimezone(t *testing.T) {
	ctx := context.Background()
	_, accounts, balances, dashboard, _ := serviceStack(t)
	hongKong := time.FixedZone("HKT", 8*60*60)
	dashboard.Now = func() time.Time { return time.Date(2026, 5, 25, 12, 0, 0, 0, hongKong) }
	a, _, err := accounts.Create(ctx, "checking", "", true, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := balances.Add(ctx, a.ID, "2026-03-30", "6000.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := balances.Add(ctx, a.ID, "2026-04-08", "3500.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := balances.Add(ctx, a.ID, "2026-05-25", "1300.00", ""); err != nil {
		t.Fatal(err)
	}

	summary, err := dashboard.Summary(ctx)
	if err != nil {
		t.Fatal(err)
	}
	assertMoneyAmount(t, "total", summary.Total, 130000)
}

func TestDashboardNetChangeUsesOnBudgetSnapshotTotals(t *testing.T) {
	ctx := context.Background()
	_, accounts, balances, dashboard, _ := serviceStack(t)
	a, _, err := accounts.Create(ctx, "cash", "", true, "")
	if err != nil {
		t.Fatal(err)
	}
	wallet, _, err := accounts.Create(ctx, "wallet", "", true, "")
	if err != nil {
		t.Fatal(err)
	}
	offBudget, _, err := accounts.Create(ctx, "investments", "", false, "")
	if err != nil {
		t.Fatal(err)
	}
	hidden, _, err := accounts.Create(ctx, "hidden", "", true, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := balances.Add(ctx, a.ID, "2026-03-05", "100.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := balances.Add(ctx, a.ID, "2026-03-20", "80.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := balances.Add(ctx, a.ID, "2026-04-05", "120.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := balances.Add(ctx, wallet.ID, "2026-04-05", "5.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := balances.Add(ctx, a.ID, "2026-04-20", "90.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := balances.Add(ctx, a.ID, "2026-05-02", "110.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := balances.Add(ctx, a.ID, "2026-05-10", "150.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := balances.Add(ctx, wallet.ID, "2026-05-10", "10.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := balances.Add(ctx, a.ID, "2026-05-24", "130.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := balances.Add(ctx, wallet.ID, "2026-05-24", "20.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := balances.Add(ctx, a.ID, "2026-06-01", "999.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := balances.Add(ctx, offBudget.ID, "2026-05-24", "999.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := balances.Add(ctx, hidden.ID, "2026-05-24", "999.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := accounts.SetHidden(ctx, hidden.ID, true); err != nil {
		t.Fatal(err)
	}
	summary, err := dashboard.Summary(ctx)
	if err != nil {
		t.Fatal(err)
	}
	assertMoneyAmount(t, "total", summary.Total, 15000)
	assertMoneyAmount(t, "from month start", summary.NetChangeFromMonthStart, 5500)
	assertMoneyAmount(t, "from month high", summary.NetChangeFromMonthHigh, -2000)
	assertMoneyAmount(t, "from previous month high", summary.NetChangeFromPreviousMonthHigh, 2500)
	if len(summary.RecentMonths) != 2 {
		t.Fatalf("recent months = %+v", summary.RecentMonths)
	}
	if summary.RecentMonths[0].Period != "2026-04" || summary.RecentMonths[1].Period != "2026-03" {
		t.Fatalf("recent month periods = %+v", summary.RecentMonths)
	}
	assertMoneyAmount(t, "apr drop", summary.RecentMonths[0].Drop, -4000)
	assertMoneyAmount(t, "mar drop", summary.RecentMonths[1].Drop, -2000)
	if summary.Trend.FromPeriod != "2026-03" || summary.Trend.ToPeriod != "2026-04" {
		t.Fatalf("trend periods = %+v", summary.Trend)
	}
	assertMoneyAmount(t, "high trend", summary.Trend.HighToHigh, 2000)
	assertMoneyAmount(t, "low trend", summary.Trend.LowToLow, 0)
}

func TestDashboardTotalSumsLatestPerAccountWithDifferentSnapshotDates(t *testing.T) {
	ctx := context.Background()
	_, accounts, balances, dashboard, _ := serviceStack(t)
	checking, _, err := accounts.Create(ctx, "checking", "", true, "")
	if err != nil {
		t.Fatal(err)
	}
	savings, _, err := accounts.Create(ctx, "savings", "", true, "")
	if err != nil {
		t.Fatal(err)
	}
	wallet, _, err := accounts.Create(ctx, "wallet", "", true, "")
	if err != nil {
		t.Fatal(err)
	}
	offBudget, _, err := accounts.Create(ctx, "brokerage", "", false, "")
	if err != nil {
		t.Fatal(err)
	}
	hidden, _, err := accounts.Create(ctx, "hidden", "", true, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := balances.Add(ctx, checking.ID, "2026-05-01", "1000.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := balances.Add(ctx, checking.ID, "2026-05-24", "1200.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := balances.Add(ctx, savings.ID, "2026-05-20", "300.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := balances.Add(ctx, wallet.ID, "2026-05-10", "40.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := balances.Add(ctx, wallet.ID, "2026-06-01", "999.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := balances.Add(ctx, offBudget.ID, "2026-05-24", "5000.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := balances.Add(ctx, hidden.ID, "2026-05-24", "7000.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := accounts.SetHidden(ctx, hidden.ID, true); err != nil {
		t.Fatal(err)
	}

	summary, err := dashboard.Summary(ctx)
	if err != nil {
		t.Fatal(err)
	}
	assertMoneyAmount(t, "total", summary.Total, 154000)
}

func TestDashboardTotalSumsLatestPerAccountInLocalTimezone(t *testing.T) {
	ctx := context.Background()
	_, accounts, balances, dashboard, _ := serviceStack(t)
	hongKong := time.FixedZone("HKT", 8*60*60)
	dashboard.Now = func() time.Time { return time.Date(2026, 5, 25, 12, 0, 0, 0, hongKong) }
	checking, _, err := accounts.Create(ctx, "checking", "", true, "")
	if err != nil {
		t.Fatal(err)
	}
	savings, _, err := accounts.Create(ctx, "savings", "", true, "")
	if err != nil {
		t.Fatal(err)
	}
	wallet, _, err := accounts.Create(ctx, "wallet", "", true, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := balances.Add(ctx, checking.ID, "2026-04-08", "3500.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := balances.Add(ctx, checking.ID, "2026-05-25", "1300.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := balances.Add(ctx, savings.ID, "2026-05-24", "700.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := balances.Add(ctx, wallet.ID, "2026-05-25", "50.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := balances.Add(ctx, wallet.ID, "2026-06-01", "999.00", ""); err != nil {
		t.Fatal(err)
	}

	summary, err := dashboard.Summary(ctx)
	if err != nil {
		t.Fatal(err)
	}
	assertMoneyAmount(t, "total", summary.Total, 205000)
}

func TestDashboardHighLowMetricsUsePerAccountValuesThenSum(t *testing.T) {
	ctx := context.Background()
	_, accounts, balances, dashboard, _ := serviceStack(t)
	dashboard.Now = func() time.Time { return time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC) }
	checking, _, err := accounts.Create(ctx, "checking", "", true, "")
	if err != nil {
		t.Fatal(err)
	}
	savings, _, err := accounts.Create(ctx, "savings", "", true, "")
	if err != nil {
		t.Fatal(err)
	}

	for _, row := range []struct {
		accountID int64
		date      string
		amount    string
	}{
		{checking.ID, "2026-03-05", "100.00"},
		{checking.ID, "2026-03-20", "50.00"},
		{savings.ID, "2026-03-10", "200.00"},
		{savings.ID, "2026-03-25", "120.00"},
		{checking.ID, "2026-04-05", "300.00"},
		{checking.ID, "2026-04-20", "100.00"},
		{savings.ID, "2026-04-10", "150.00"},
		{savings.ID, "2026-04-25", "80.00"},
		{checking.ID, "2026-05-05", "500.00"},
		{checking.ID, "2026-05-25", "200.00"},
		{savings.ID, "2026-05-10", "50.00"},
		{savings.ID, "2026-05-20", "400.00"},
		{savings.ID, "2026-05-25", "300.00"},
	} {
		if _, _, err := balances.Add(ctx, row.accountID, row.date, row.amount, ""); err != nil {
			t.Fatal(err)
		}
	}

	summary, err := dashboard.Summary(ctx)
	if err != nil {
		t.Fatal(err)
	}
	assertMoneyAmount(t, "total", summary.Total, 50000)
	assertMoneyAmount(t, "from month high", summary.NetChangeFromMonthHigh, -40000)
	assertMoneyAmount(t, "from previous month high", summary.NetChangeFromPreviousMonthHigh, 5000)
	assertMoneyAmount(t, "apr drop", summary.RecentMonths[0].Drop, -32000)
	assertMoneyAmount(t, "mar drop", summary.RecentMonths[1].Drop, -13000)
	assertMoneyAmount(t, "high trend", summary.Trend.HighToHigh, 15000)
	assertMoneyAmount(t, "low trend", summary.Trend.LowToLow, -4000)
}

func TestDashboardNoBalanceHistoryUsesZeroValues(t *testing.T) {
	ctx := context.Background()
	_, _, _, dashboard, _ := serviceStack(t)
	summary, err := dashboard.Summary(ctx)
	if err != nil {
		t.Fatal(err)
	}
	assertMoneyAmount(t, "total", summary.Total, 0)
	assertMoneyAmount(t, "from month start", summary.NetChangeFromMonthStart, 0)
	assertMoneyAmount(t, "from month high", summary.NetChangeFromMonthHigh, 0)
	assertMoneyAmount(t, "from previous month high", summary.NetChangeFromPreviousMonthHigh, 0)
	assertMoneyAmount(t, "recent month 1", summary.RecentMonths[0].Drop, 0)
	assertMoneyAmount(t, "recent month 2", summary.RecentMonths[1].Drop, 0)
	assertMoneyAmount(t, "high trend", summary.Trend.HighToHigh, 0)
	assertMoneyAmount(t, "low trend", summary.Trend.LowToLow, 0)
}

func assertMoneyAmount(t *testing.T, name string, got money.Money, want int64) {
	t.Helper()
	if got.Amount != want {
		t.Fatalf("%s = %d, want %d", name, got.Amount, want)
	}
}
