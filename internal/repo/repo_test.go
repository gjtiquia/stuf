package repo

import (
	"context"
	"database/sql"
	"errors"
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

func ptrInt64(v int64) *int64 { return &v }

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
	} else {
		var dup *AccountDuplicateNameError
		if !errors.As(err, &dup) {
			t.Fatalf("expected duplicate account name domain error, got %T %[1]v", err)
		}
		if dup.Name != "hsbc-one" {
			t.Fatalf("duplicate account name = %q", dup.Name)
		}
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			t.Fatalf("duplicate account error should hide raw sqlite error: %v", err)
		}
	}
	b, err := s.Bal.Create(ctx, BalanceCreate{AccountID: a.ID, Date: "2026-05-01", Amount: money.Money{Amount: 5000000, Scale: 2}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Bal.Create(ctx, BalanceCreate{AccountID: a.ID, Date: "2026-05-01", Amount: money.Money{Amount: 1, Scale: 2}}); err == nil {
		t.Fatal("expected unique balance date error")
	} else {
		var dup *BalanceDuplicateDateError
		if !errors.As(err, &dup) {
			t.Fatalf("expected duplicate date domain error, got %T %[1]v", err)
		}
		if dup.Date != "2026-05-01" {
			t.Fatalf("duplicate error date = %q", dup.Date)
		}
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			t.Fatalf("duplicate error should hide raw sqlite error: %v", err)
		}
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

func TestAccountListVisibleVsHidden(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	hkd, _ := s.Cur.GetByCode(ctx, "HKD")
	visible, err := s.Acct.Create(ctx, AccountCreate{Name: "cash", CurrencyID: hkd.ID, OnBudget: true})
	if err != nil {
		t.Fatal(err)
	}
	hidden, err := s.Acct.Create(ctx, AccountCreate{Name: "old", CurrencyID: hkd.ID, OnBudget: true})
	if err != nil {
		t.Fatal(err)
	}
	hidden.Hidden = true
	if _, err := s.Acct.Update(ctx, hidden); err != nil {
		t.Fatal(err)
	}
	visibleOnly, err := s.Acct.List(ctx, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(visibleOnly) != 1 || visibleOnly[0].ID != visible.ID {
		t.Fatalf("visible list = %+v", visibleOnly)
	}
	all, err := s.Acct.List(ctx, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("all list = %+v", all)
	}
}

func TestAccountParentChildRepos(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	hkd, _ := s.Cur.GetByCode(ctx, "HKD")
	parent, err := s.Acct.Create(ctx, AccountCreate{Name: "investment", CurrencyID: hkd.ID, OnBudget: false})
	if err != nil {
		t.Fatal(err)
	}
	child, err := s.Acct.Create(ctx, AccountCreate{Name: "investment-hkd", CurrencyID: hkd.ID, OnBudget: false, ParentID: ptrInt64(parent.ID)})
	if err != nil {
		t.Fatal(err)
	}
	if child.ParentID == nil || *child.ParentID != parent.ID {
		t.Fatalf("child parent id = %v, want %d", child.ParentID, parent.ID)
	}
	roots, err := s.Acct.ListRoots(ctx, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(roots) != 1 || roots[0].ID != parent.ID {
		t.Fatalf("roots = %+v", roots)
	}
	children, err := s.Acct.ListChildren(ctx, parent.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(children) != 1 || children[0].ID != child.ID {
		t.Fatalf("children = %+v", children)
	}
	empty, err := s.Acct.IsEmpty(ctx, parent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if empty {
		t.Fatal("parent with child should not be empty")
	}
	empty, err = s.Acct.IsEmpty(ctx, child.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !empty {
		t.Fatal("child with no balances or children should be empty")
	}
	if _, err := s.Bal.Create(ctx, BalanceCreate{AccountID: child.ID, Date: "2026-05-01", Amount: money.Money{Amount: 10000, Scale: 2}}); err != nil {
		t.Fatal(err)
	}
	empty, err = s.Acct.IsEmpty(ctx, child.ID)
	if err != nil {
		t.Fatal(err)
	}
	if empty {
		t.Fatal("child with balance should not be empty")
	}
}

func TestAccountUpdateDeleteNotFoundAndHasBalances(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	hkd, _ := s.Cur.GetByCode(ctx, "HKD")
	a, err := s.Acct.Create(ctx, AccountCreate{Name: "cash", CurrencyID: hkd.ID, OnBudget: true, Notes: "notes"})
	if err != nil {
		t.Fatal(err)
	}
	has, err := s.Acct.HasBalances(ctx, a.ID)
	if err != nil || has {
		t.Fatalf("HasBalances empty = %v %v", has, err)
	}
	a.Name = "savings"
	a.OnBudget = false
	a.Notes = "updated"
	updated, err := s.Acct.Update(ctx, a)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "savings" || updated.OnBudget || updated.Notes != "updated" {
		t.Fatalf("update = %+v", updated)
	}
	if _, err := s.Bal.Create(ctx, BalanceCreate{AccountID: a.ID, Date: "2026-05-01", Amount: money.Money{Amount: 100, Scale: 2}}); err != nil {
		t.Fatal(err)
	}
	has, err = s.Acct.HasBalances(ctx, a.ID)
	if err != nil || !has {
		t.Fatalf("HasBalances with data = %v %v", has, err)
	}
	if err := s.Acct.Delete(ctx, a.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Acct.GetByID(ctx, a.ID); err == nil || !strings.Contains(err.Error(), "account not found") {
		t.Fatalf("expected not found after delete, got %v", err)
	}
	if _, err := s.Acct.GetByName(ctx, "savings"); err == nil || !strings.Contains(err.Error(), "account not found") {
		t.Fatalf("expected not found by name, got %v", err)
	}
}

func TestBalanceGetByDateListUpdateDeleteNotFoundAndLatestEmpty(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	hkd, _ := s.Cur.GetByCode(ctx, "HKD")
	a, _ := s.Acct.Create(ctx, AccountCreate{Name: "cash", CurrencyID: hkd.ID, OnBudget: true})
	latest, ok, err := s.Bal.LatestByAccount(ctx, a.ID)
	if err != nil || ok || latest.ID != 0 {
		t.Fatalf("latest empty = %+v %v %v", latest, ok, err)
	}
	b1, err := s.Bal.Create(ctx, BalanceCreate{AccountID: a.ID, Date: "2026-05-01", Amount: money.Money{Amount: 10000, Scale: 2}, Notes: "first"})
	if err != nil {
		t.Fatal(err)
	}
	b2, err := s.Bal.Create(ctx, BalanceCreate{AccountID: a.ID, Date: "2026-06-01", Amount: money.Money{Amount: 15000, Scale: 2}})
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.Bal.GetByAccountDate(ctx, a.ID, "2026-05-01")
	if err != nil || got.ID != b1.ID || got.Notes != "first" {
		t.Fatalf("get by date = %+v %v", got, err)
	}
	list, err := s.Bal.ListByAccount(ctx, a.ID)
	if err != nil || len(list) != 2 || list[0].ID != b2.ID || list[1].ID != b1.ID {
		t.Fatalf("list order = %+v %v", list, err)
	}
	b1.Amount = money.Money{Amount: 12000, Scale: 2}
	b1.Notes = "edited"
	updated, err := s.Bal.Update(ctx, b1)
	if err != nil || updated.Amount.Amount != 12000 || updated.Notes != "edited" {
		t.Fatalf("update = %+v %v", updated, err)
	}
	if err := s.Bal.Delete(ctx, b1.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Bal.GetByID(ctx, b1.ID); err == nil || !strings.Contains(err.Error(), "balance not found") {
		t.Fatalf("expected not found after delete, got %v", err)
	}
}

func TestCurrencyGetByIDListNotFoundAndMissingRate(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	usd, err := s.Cur.GetByCode(ctx, "USD")
	if err != nil {
		t.Fatal(err)
	}
	byID, err := s.Cur.GetByID(ctx, usd.ID)
	if err != nil || byID.Code != "USD" || byID.RateToUSD.Amount != 1 {
		t.Fatalf("get by id = %+v %v", byID, err)
	}
	list, err := s.Cur.List(ctx)
	if err != nil || len(list) == 0 {
		t.Fatalf("list = %+v %v", list, err)
	}
	for i := 1; i < len(list); i++ {
		if list[i-1].Code >= list[i].Code {
			t.Fatalf("list not ordered by code: %+v", list)
		}
	}
	if _, err := s.Cur.GetByCode(ctx, "ZZZ"); err == nil || err.Error() != "currency is unavailable: ZZZ" {
		t.Fatalf("expected not found, got %v", err)
	} else if strings.Contains(err.Error(), "sql: no rows") {
		t.Fatalf("currency error should hide raw sql error: %v", err)
	}
}

func TestHistoryOrderingAndNullableData(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	old := "old"
	newData := "new"
	h1, err := s.Hist.Create(ctx, History{Timestamp: "2026-05-24T10:00:00Z", Action: "create", Path: "/a", NewData: &newData})
	if err != nil {
		t.Fatal(err)
	}
	h2, err := s.Hist.Create(ctx, History{Timestamp: "2026-05-24T11:00:00Z", Action: "edit", Path: "/b", OldData: &old, NewData: &newData})
	if err != nil {
		t.Fatal(err)
	}
	h3, err := s.Hist.Create(ctx, History{Timestamp: "2026-05-24T11:00:00Z", Action: "delete", Path: "/c"})
	if err != nil {
		t.Fatal(err)
	}
	rows, err := s.Hist.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 || rows[0].ID != h1.ID || rows[1].ID != h2.ID || rows[2].ID != h3.ID {
		t.Fatalf("history order = %+v", rows)
	}
	if rows[0].OldData != nil || rows[0].NewData == nil || *rows[0].NewData != "new" {
		t.Fatalf("h1 data = %+v", rows[0])
	}
	if rows[1].OldData == nil || *rows[1].OldData != "old" {
		t.Fatalf("h2 old data = %+v", rows[1])
	}
	if rows[2].OldData != nil || rows[2].NewData != nil {
		t.Fatalf("h3 nullable = %+v", rows[2])
	}
}

func TestSeedCurrenciesIdempotent(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	before, err := s.Cur.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SeedCurrencies(ctx); err != nil {
		t.Fatal(err)
	}
	after, err := s.Cur.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(before) {
		t.Fatalf("seed changed count: before=%d after=%d", len(before), len(after))
	}
	usd, err := s.Cur.GetByCode(ctx, "USD")
	if err != nil || usd.RateToUSD.Amount != 1 {
		t.Fatalf("USD after reseed = %+v %v", usd, err)
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
