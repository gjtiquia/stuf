package repo

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"stuf/internal/money"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(context.Background(), filepath.Join(t.TempDir(), "db.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	s.Clock = func() time.Time { return time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC) }
	return s
}

func TestFreshDBMigrationsAndSeeding(t *testing.T) {
	s := testStore(t)
	usd, err := s.Cur.GetByCode(context.Background(), "USD")
	if err != nil {
		t.Fatal(err)
	}
	if usd.RateToUSD.Amount != 1 || usd.RateToUSD.Scale != 0 {
		t.Fatalf("bad USD seed: %+v", usd)
	}
	if err := s.SeedCurrencies(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestRejectsNonStufSQLiteDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "db.sqlite")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("CREATE TABLE something_else (id INTEGER PRIMARY KEY)"); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	_, err = Open(context.Background(), path)
	if err == nil || !strings.Contains(err.Error(), "not a stuf database") {
		t.Fatalf("expected non-stuf rejection, got %v", err)
	}
}

func TestAccountBalanceHistoryRepos(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	hkd, _ := s.Cur.GetByCode(ctx, "HKD")
	a, err := s.Acct.Create(ctx, AccountCreate{Name: "hsbc-one", CurrencyID: hkd.ID, OnBudget: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Acct.Create(ctx, AccountCreate{Name: "hsbc-one", CurrencyID: hkd.ID, OnBudget: true}); err == nil {
		t.Fatal("expected unique account name error")
	}
	b, err := s.Bal.Create(ctx, BalanceCreate{AccountID: a.ID, Date: "2026-05-01", Amount: money.Money{Amount: 5000000, Scale: 2}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Bal.Create(ctx, BalanceCreate{AccountID: a.ID, Date: "2026-05-01", Amount: money.Money{Amount: 1, Scale: 2}}); err == nil {
		t.Fatal("expected unique balance date error")
	}
	latest, ok, err := s.Bal.LatestByAccount(ctx, a.ID)
	if err != nil || !ok || latest.ID != b.ID {
		t.Fatalf("latest = %+v %v %v", latest, ok, err)
	}
	h, err := s.Hist.Create(ctx, History{Action: "add", Path: "/accounts/hsbc-one/balances/2026-05-01"})
	if err != nil {
		t.Fatal(err)
	}
	rows, _ := s.Hist.List(ctx)
	if len(rows) != 1 || rows[0].ID != h.ID {
		t.Fatalf("history rows = %+v", rows)
	}
	if err := s.Hist.Delete(ctx, h.ID); err != nil {
		t.Fatal(err)
	}
}

func TestBackupPreservesDatabase(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	hkd, _ := s.Cur.GetByCode(ctx, "HKD")
	if _, err := s.Acct.Create(ctx, AccountCreate{Name: "cash", CurrencyID: hkd.ID, OnBudget: true}); err != nil {
		t.Fatal(err)
	}
	path, err := s.Backup(ctx, time.Date(2026, 5, 24, 15, 4, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	copyStore, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer copyStore.Close()
	if _, err := copyStore.Acct.GetByName(ctx, "cash"); err != nil {
		t.Fatal(err)
	}
}
