package service

import (
	"context"
	"path/filepath"
	"testing"
	"time"

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

func TestDashboardGrowthNearestBoundaryAndHiddenOmitted(t *testing.T) {
	ctx := context.Background()
	_, accounts, balances, dashboard, _ := serviceStack(t)
	a, _, err := accounts.Create(ctx, "cash", "", true, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := balances.Add(ctx, a.ID, "2026-05-02", "100.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := balances.Add(ctx, a.ID, "2026-06-01", "150.00", ""); err != nil {
		t.Fatal(err)
	}
	summary, err := dashboard.Summary(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if summary.TotalGrow.Amount != 5000 {
		t.Fatalf("growth = %+v", summary.TotalGrow)
	}
	if _, _, err := accounts.SetHidden(ctx, a.ID, true); err != nil {
		t.Fatal(err)
	}
	summary, err = dashboard.Summary(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Total.Amount != 0 || summary.TotalGrow.Amount != 0 {
		t.Fatalf("hidden account included: %+v", summary)
	}
}
