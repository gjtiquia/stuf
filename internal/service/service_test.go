package service

import (
	"context"
	"errors"
	"fmt"
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
	a := AccountService{Store: s, Accounts: s.Acct, Balances: s.Bal, Currency: s.Cur, Tags: s.Tag, History: h, AppCurrency: "HKD"}
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

func TestAccountTagsCreateEditInheritanceAndUndo(t *testing.T) {
	ctx := context.Background()
	s, accounts, _, _, history := serviceStack(t)
	parent, _, err := accounts.CreateWithTags(ctx, "household", "HKD", true, "", []string{"family/shared", "wallet", "wallet"})
	if err != nil {
		t.Fatal(err)
	}
	if len(accountTagNames(t, accounts, parent.ID, false)) != 2 {
		t.Fatalf("duplicate tags should be suppressed, got %v", accountTagNames(t, accounts, parent.ID, false))
	}
	child, _, err := accounts.CreateChildWithTags(ctx, parent.ID, "household-cash", "HKD", "", []string{"cash"})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(accountTagNames(t, accounts, child.ID, true), ","); got != "cash,family/shared,wallet" {
		t.Fatalf("child effective tags = %s", got)
	}
	updated, editEntry, err := accounts.UpdateWithTags(ctx, child.ID, child.Name, child.Code, child.OnBudget, child.Hidden, "tiny wallet", []string{"wallet"})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Notes != "tiny wallet" {
		t.Fatalf("notes were not updated: %+v", updated)
	}
	if got := strings.Join(accountTagNames(t, accounts, child.ID, false), ","); got != "wallet" {
		t.Fatalf("child direct tags after edit = %s", got)
	}
	if err := history.Undo(ctx, editEntry); err != nil {
		t.Fatal(err)
	}
	child, err = accounts.GetByName(ctx, "household-cash")
	if err != nil {
		t.Fatal(err)
	}
	if child.Notes != "" || strings.Join(accountTagNames(t, accounts, child.ID, false), ",") != "cash" {
		t.Fatalf("undo should restore notes and direct tags: child=%+v tags=%v", child, accountTagNames(t, accounts, child.ID, false))
	}
	temp, entry, err := accounts.CreateWithTags(ctx, "temporary", "HKD", true, "", []string{"one-off"})
	if err != nil {
		t.Fatal(err)
	}
	other, _, err := accounts.CreateWithTags(ctx, "other", "HKD", true, "", []string{"one-off"})
	if err != nil {
		t.Fatal(err)
	}
	if err := history.Undo(ctx, entry); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Acct.GetByID(ctx, temp.ID); err == nil {
		t.Fatal("tagged account should be undone")
	}
	if _, err := s.Tag.GetByName(ctx, "one-off"); err != nil {
		t.Fatal("inline-created tag should remain if another account now uses it")
	}
	if got := strings.Join(accountTagNames(t, accounts, other.ID, false), ","); got != "one-off" {
		t.Fatalf("other account should keep shared tag, got %s", got)
	}
	unused, unusedEntry, err := accounts.CreateWithTags(ctx, "unused-tag-account", "HKD", true, "", []string{"unused-inline"})
	if err != nil {
		t.Fatal(err)
	}
	if err := history.Undo(ctx, unusedEntry); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Acct.GetByID(ctx, unused.ID); err == nil {
		t.Fatal("unused tag account should be undone")
	}
	if _, err := s.Tag.GetByName(ctx, "unused-inline"); err == nil {
		t.Fatal("unused inline-created tag should be undone when account create is undone")
	}
}

func TestAccountCreateWithTagsRollsBackWhenHistoryFails(t *testing.T) {
	ctx := context.Background()
	s, accounts, _, _, _ := serviceStack(t)
	if _, err := s.DB.ExecContext(ctx, "DROP TABLE history"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := accounts.CreateWithTags(ctx, "cash", "HKD", true, "", []string{"owner/me"}); err == nil {
		t.Fatal("expected history failure")
	}
	if _, err := s.Acct.GetByName(ctx, "cash"); err == nil {
		t.Fatal("account should be rolled back when history recording fails")
	}
	if _, err := s.Tag.GetByName(ctx, "owner/me"); err == nil {
		t.Fatal("inline-created tag should be rolled back when account create fails")
	}
}

func TestAccountUpdateWithTagsRollsBackWhenHistoryFails(t *testing.T) {
	ctx := context.Background()
	s, accounts, _, _, _ := serviceStack(t)
	acct, _, err := accounts.CreateWithTags(ctx, "cash", "HKD", true, "old notes", []string{"old"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.DB.ExecContext(ctx, "DROP TABLE history"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := accounts.UpdateWithTags(ctx, acct.ID, "cash", "HKD", true, false, "new notes", []string{"new"}); err == nil {
		t.Fatal("expected history failure")
	}
	unchanged, err := s.Acct.GetByID(ctx, acct.ID)
	if err != nil {
		t.Fatal(err)
	}
	if unchanged.Notes != "old notes" {
		t.Fatalf("account notes should be rolled back, got %q", unchanged.Notes)
	}
	if got := strings.Join(accountTagNames(t, accounts, acct.ID, false), ","); got != "old" {
		t.Fatalf("direct tags should be rolled back, got %q", got)
	}
	if _, err := s.Tag.GetByName(ctx, "new"); err == nil {
		t.Fatal("inline-created tag should be rolled back when account update fails")
	}
}

func TestTagValidationRenameAndUndo(t *testing.T) {
	ctx := context.Background()
	s, accounts, _, _, history := serviceStack(t)
	tags := TagService{Store: s, Tags: s.Tag, History: history}
	if _, _, err := tags.Create(ctx, "bad//tag", ""); err == nil {
		t.Fatal("expected invalid slash form")
	}
	if _, _, err := tags.Create(ctx, "bad/", ""); err == nil {
		t.Fatal("expected trailing slash to be invalid")
	}
	tag, _, err := tags.Create(ctx, "family/shared", "old")
	if err != nil {
		t.Fatal(err)
	}
	account, _, err := accounts.CreateWithTags(ctx, "cash", "HKD", true, "", []string{"family/shared"})
	if err != nil {
		t.Fatal(err)
	}
	renamed, entry, err := tags.Update(ctx, tag.ID, "family/core", "new")
	if err != nil {
		t.Fatal(err)
	}
	if renamed.ID != tag.ID {
		t.Fatal("tag rename should preserve tag id")
	}
	if got := strings.Join(accountTagNames(t, accounts, account.ID, true), ","); got != "family/core" {
		t.Fatalf("account should display renamed tag through id, got %s", got)
	}
	if err := history.Undo(ctx, entry); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(accountTagNames(t, accounts, account.ID, true), ","); got != "family/shared" {
		t.Fatalf("undo should restore old tag name, got %s", got)
	}
}

func TestTagCreateRollsBackWhenHistoryFails(t *testing.T) {
	ctx := context.Background()
	s, _, _, _, history := serviceStack(t)
	tags := TagService{Store: s, Tags: s.Tag, History: history}
	if _, err := s.DB.ExecContext(ctx, "DROP TABLE history"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := tags.Create(ctx, "owner/me", "owner tag"); err == nil {
		t.Fatal("expected history failure")
	}
	if _, err := s.Tag.GetByName(ctx, "owner/me"); err == nil {
		t.Fatal("tag should be rolled back when history recording fails")
	}
}

func TestTagUpdateRollsBackWhenHistoryFails(t *testing.T) {
	ctx := context.Background()
	s, _, _, _, history := serviceStack(t)
	tags := TagService{Store: s, Tags: s.Tag, History: history}
	tag, _, err := tags.Create(ctx, "owner/me", "old notes")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.DB.ExecContext(ctx, "DROP TABLE history"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := tags.Update(ctx, tag.ID, "owner/family", "new notes"); err == nil {
		t.Fatal("expected history failure")
	}
	unchanged, err := s.Tag.GetByID(ctx, tag.ID)
	if err != nil {
		t.Fatal(err)
	}
	if unchanged.Name != "owner/me" || unchanged.Notes != "old notes" {
		t.Fatalf("tag should be rolled back, got %+v", unchanged)
	}
	if _, err := s.Tag.GetByName(ctx, "owner/family"); err == nil {
		t.Fatal("renamed tag should not exist after rollback")
	}
}

func accountTagNames(t *testing.T, accounts AccountService, accountID int64, effective bool) []string {
	t.Helper()
	var (
		tags []repo.Tag
		err  error
	)
	if effective {
		tags, err = accounts.ListEffectiveTags(context.Background(), accountID)
	} else {
		tags, err = accounts.ListDirectTags(context.Background(), accountID)
	}
	if err != nil {
		t.Fatal(err)
	}
	out := make([]string, len(tags))
	for i, tag := range tags {
		out[i] = tag.Name
	}
	return out
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

func TestBalanceAddRollsBackWhenHistoryFails(t *testing.T) {
	ctx := context.Background()
	s, accounts, balances, _, _ := serviceStack(t)
	a, _, err := accounts.Create(ctx, "cash", "", true, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.DB.ExecContext(ctx, "DROP TABLE history"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := balances.Add(ctx, a.ID, "2026-05-24", "10.50", "opening"); err == nil {
		t.Fatal("expected history failure")
	}
	if _, err := s.Bal.GetByAccountDate(ctx, a.ID, "2026-05-24"); err == nil {
		t.Fatal("balance should be rolled back when history recording fails")
	}
}

func TestBalanceUpdateRollsBackWhenHistoryFails(t *testing.T) {
	ctx := context.Background()
	s, accounts, balances, _, _ := serviceStack(t)
	a, _, err := accounts.Create(ctx, "cash", "", true, "")
	if err != nil {
		t.Fatal(err)
	}
	bal, _, err := balances.Add(ctx, a.ID, "2026-05-24", "10.50", "old notes")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.DB.ExecContext(ctx, "DROP TABLE history"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := balances.Update(ctx, bal.ID, "2026-05-25", "99.00", "new notes"); err == nil {
		t.Fatal("expected history failure")
	}
	unchanged, err := s.Bal.GetByID(ctx, bal.ID)
	if err != nil {
		t.Fatal(err)
	}
	if unchanged.Date != "2026-05-24" || unchanged.Amount.Amount != 1050 || unchanged.Notes != "old notes" {
		t.Fatalf("balance should be rolled back, got %+v", unchanged)
	}
	if _, err := s.Bal.GetByAccountDate(ctx, a.ID, "2026-05-25"); err == nil {
		t.Fatal("updated balance date should not exist after rollback")
	}
}

func TestBalanceDeleteRollsBackWhenHistoryFails(t *testing.T) {
	ctx := context.Background()
	s, accounts, balances, _, _ := serviceStack(t)
	a, _, err := accounts.Create(ctx, "cash", "", true, "")
	if err != nil {
		t.Fatal(err)
	}
	bal, _, err := balances.Add(ctx, a.ID, "2026-05-24", "10.50", "opening")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.DB.ExecContext(ctx, "DROP TABLE history"); err != nil {
		t.Fatal(err)
	}
	if _, err := balances.Delete(ctx, bal.ID); err == nil {
		t.Fatal("expected history failure")
	}
	if _, err := s.Bal.GetByID(ctx, bal.ID); err != nil {
		t.Fatalf("balance should be restored after failed delete history: %v", err)
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
	assertMoneyAmount(t, "may drop", summary.RecentMonths[0].Drop, -21976)
	assertMoneyAmount(t, "apr drop", summary.RecentMonths[1].Drop, 0)
	assertMoneyAmount(t, "mar drop", summary.RecentMonths[2].Drop, 0)
	assertMoneyAmount(t, "high trend", summary.HighTrends[1].Change, 0)
	assertMoneyAmount(t, "low trend", summary.LowTrends[1].Change, 0)
}

func TestDashboardSummaryExposesAsOfDate(t *testing.T) {
	ctx := context.Background()
	_, accounts, balances, dashboard, _ := serviceStack(t)
	dashboard.Now = func() time.Time { return time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC) }
	acct, _, err := accounts.Create(ctx, "checking", "", true, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := balances.Add(ctx, acct.ID, "2026-05-24", "100.00", ""); err != nil {
		t.Fatal(err)
	}
	summary, err := dashboard.Summary(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if summary.AsOf != "2026-05-24" {
		t.Fatalf("as of = %s", summary.AsOf)
	}
	if !summary.AsOfStale {
		t.Fatal("as of should be stale when latest snapshot is before today")
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
	assertMoneyAmount(t, "may drop", summary.RecentMonths[0].Drop, 0)
	assertMoneyAmount(t, "apr drop", summary.RecentMonths[1].Drop, 0)
	assertMoneyAmount(t, "mar drop", summary.RecentMonths[2].Drop, 0)
	assertMoneyAmount(t, "high trend", summary.HighTrends[1].Change, 0)
	assertMoneyAmount(t, "low trend", summary.LowTrends[1].Change, 0)
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
	assertMoneyAmount(t, "apr drop", summary.RecentMonths[1].Drop, -10000)
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
	assertMoneyAmount(t, "apr drop", summary.RecentMonths[1].Drop, -15000)
	assertMoneyAmount(t, "mar drop", summary.RecentMonths[2].Drop, -5000)
	assertMoneyAmount(t, "high trend", summary.HighTrends[1].Change, 10000)
	assertMoneyAmount(t, "low trend", summary.LowTrends[1].Change, 0)
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
	assertMoneyAmount(t, "apr drop", summary.RecentMonths[1].Drop, -5000)
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
	assertMoneyAmount(t, "apr drop", summary.RecentMonths[1].Drop, -4000)
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
	assertMoneyAmount(t, "apr drop", summary.RecentMonths[1].Drop, -40000)
	assertMoneyAmount(t, "high trend", summary.HighTrends[1].Change, 30000)
	assertMoneyAmount(t, "low trend", summary.LowTrends[1].Change, 0)
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
	if len(summary.RecentMonths) != 3 {
		t.Fatalf("recent months = %+v", summary.RecentMonths)
	}
	if summary.RecentMonths[0].Period != "2026-05" || summary.RecentMonths[1].Period != "2026-04" || summary.RecentMonths[2].Period != "2026-03" {
		t.Fatalf("recent month periods = %+v", summary.RecentMonths)
	}
	assertMoneyAmount(t, "may drop", summary.RecentMonths[0].Drop, -7500)
	assertMoneyAmount(t, "apr drop", summary.RecentMonths[1].Drop, -4000)
	assertMoneyAmount(t, "mar drop", summary.RecentMonths[2].Drop, -2000)
	if summary.HighTrends[1].FromPeriod != "2026-03" || summary.HighTrends[1].ToPeriod != "2026-04" {
		t.Fatalf("trend periods = %+v", summary.HighTrends[1])
	}
	assertMoneyAmount(t, "high trend", summary.HighTrends[1].Change, 2000)
	assertMoneyAmount(t, "low trend", summary.LowTrends[1].Change, 0)
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
	assertMoneyAmount(t, "apr drop", summary.RecentMonths[1].Drop, -32000)
	assertMoneyAmount(t, "mar drop", summary.RecentMonths[2].Drop, -13000)
	assertMoneyAmount(t, "high trend", summary.HighTrends[1].Change, 15000)
	assertMoneyAmount(t, "low trend", summary.LowTrends[1].Change, -4000)
}

func TestDashboardExpandedThreeMonthContextUsesDistinctMonthRows(t *testing.T) {
	ctx := context.Background()
	_, accounts, balances, dashboard, _ := serviceStack(t)
	dashboard.Now = func() time.Time { return time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC) }
	acct, _, err := accounts.Create(ctx, "checking", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range []struct {
		date   string
		amount string
	}{
		{"2026-02-01", "100.00"},
		{"2026-02-10", "140.00"},
		{"2026-02-20", "90.00"},
		{"2026-03-05", "200.00"},
		{"2026-03-20", "150.00"},
		{"2026-04-10", "250.00"},
		{"2026-04-25", "180.00"},
		{"2026-05-10", "300.00"},
		{"2026-05-25", "240.00"},
	} {
		if _, _, err := balances.Add(ctx, acct.ID, row.date, row.amount, ""); err != nil {
			t.Fatal(err)
		}
	}

	summary, err := dashboard.Summary(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(summary.RecentMonths) != 3 {
		t.Fatalf("recent months = %+v", summary.RecentMonths)
	}
	assertPeriod(t, "recent month 0", summary.RecentMonths[0].Period, "2026-05")
	assertMoneyAmount(t, "may drop", summary.RecentMonths[0].Drop, -12000)
	assertPeriod(t, "recent month 1", summary.RecentMonths[1].Period, "2026-04")
	assertMoneyAmount(t, "apr drop", summary.RecentMonths[1].Drop, -10000)
	assertPeriod(t, "recent month 2", summary.RecentMonths[2].Period, "2026-03")
	assertMoneyAmount(t, "mar drop", summary.RecentMonths[2].Drop, -11000)
	assertTrend(t, "high trend 0", summary.HighTrends[0], "2026-04", "2026-05", 5000)
	assertTrend(t, "high trend 1", summary.HighTrends[1], "2026-03", "2026-04", 5000)
	assertTrend(t, "high trend 2", summary.HighTrends[2], "2026-02", "2026-03", 6000)
	assertTrend(t, "low trend 0", summary.LowTrends[0], "2026-04", "2026-05", 3000)
	assertTrend(t, "low trend 1", summary.LowTrends[1], "2026-03", "2026-04", 6000)
	assertTrend(t, "low trend 2", summary.LowTrends[2], "2026-02", "2026-03", 0)
}

func TestDashboardCompactRowsUseAsOfAndMonthBoundaries(t *testing.T) {
	ctx := context.Background()
	_, accounts, balances, dashboard, _ := serviceStack(t)
	dashboard.Now = func() time.Time { return time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC) }
	acct, _, err := accounts.Create(ctx, "checking", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range []struct {
		date   string
		amount string
	}{
		{"2026-03-01", "90.00"},
		{"2026-03-10", "120.00"},
		{"2026-03-31", "100.00"},
		{"2026-04-01", "150.00"},
		{"2026-04-15", "180.00"},
		{"2026-04-30", "130.00"},
		{"2026-05-01", "200.00"},
		{"2026-05-12", "260.00"},
		{"2026-05-20", "210.00"},
		{"2026-05-26", "999.00"},
	} {
		if _, _, err := balances.Add(ctx, acct.ID, row.date, row.amount, ""); err != nil {
			t.Fatal(err)
		}
	}

	summary, err := dashboard.Summary(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if summary.AsOf != "2026-05-20" {
		t.Fatalf("as of = %s", summary.AsOf)
	}
	if !summary.AsOfStale {
		t.Fatal("as of should be stale")
	}
	assertMoneyAmount(t, "total", summary.Total, 21000)
	assertMonthValue(t, "may net change", summary.NetChanges[0].Period, summary.NetChanges[0].Change, "2026-05", 1000)
	assertMonthValue(t, "apr net change", summary.NetChanges[1].Period, summary.NetChanges[1].Change, "2026-04", -2000)
	assertMonthValue(t, "mar net change", summary.NetChanges[2].Period, summary.NetChanges[2].Change, "2026-03", 1000)
	assertMonthValue(t, "may high to low", summary.HighToLows[0].Period, summary.HighToLows[0].Drop, "2026-05", -6000)
	assertMonthValue(t, "apr high to low", summary.HighToLows[1].Period, summary.HighToLows[1].Drop, "2026-04", -5000)
	assertMonthValue(t, "mar high to low", summary.HighToLows[2].Period, summary.HighToLows[2].Drop, "2026-03", -3000)
	assertMonthValue(t, "may low", summary.Lows[0].Period, summary.Lows[0].Low, "2026-05", 20000)
	assertMonthValue(t, "apr low", summary.Lows[1].Period, summary.Lows[1].Low, "2026-04", 13000)
	assertMonthValue(t, "mar low", summary.Lows[2].Period, summary.Lows[2].Low, "2026-03", 9000)
}

func TestDashboardExpandedParentContextUsesChildrenPlusRemaining(t *testing.T) {
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
		{parent.ID, "2026-04-01", "1000.00"},
		{parent.ID, "2026-05-25", "900.00"},
		{child.ID, "2026-04-01", "400.00"},
		{child.ID, "2026-05-10", "700.00"},
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
	assertMoneyAmount(t, "from month high", summary.NetChangeFromMonthHigh, -40000)
	assertPeriod(t, "recent month 0", summary.RecentMonths[0].Period, "2026-05")
	assertMoneyAmount(t, "may drop", summary.RecentMonths[0].Drop, -60000)
	assertTrend(t, "high trend 0", summary.HighTrends[0], "2026-04", "2026-05", 30000)
	assertTrend(t, "low trend 0", summary.LowTrends[0], "2026-04", "2026-05", -30000)
}

func TestDashboardNoBalanceHistoryUsesZeroValues(t *testing.T) {
	ctx := context.Background()
	_, _, _, dashboard, _ := serviceStack(t)
	summary, err := dashboard.Summary(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if summary.AsOf != "none" {
		t.Fatalf("as of = %s", summary.AsOf)
	}
	if !summary.AsOfStale {
		t.Fatal("as of should be stale when no snapshots exist")
	}
	assertMoneyAmount(t, "total", summary.Total, 0)
	assertMoneyAmount(t, "from month start", summary.NetChangeFromMonthStart, 0)
	assertMoneyAmount(t, "from month high", summary.NetChangeFromMonthHigh, 0)
	assertMoneyAmount(t, "from previous month high", summary.NetChangeFromPreviousMonthHigh, 0)
	assertMoneyAmount(t, "recent month 1", summary.RecentMonths[0].Drop, 0)
	assertMoneyAmount(t, "recent month 2", summary.RecentMonths[1].Drop, 0)
	assertMoneyAmount(t, "recent month 3", summary.RecentMonths[2].Drop, 0)
	assertMoneyAmount(t, "high trend", summary.HighTrends[1].Change, 0)
	assertMoneyAmount(t, "low trend", summary.LowTrends[1].Change, 0)
	assertMonthValue(t, "net change 1", summary.NetChanges[0].Period, summary.NetChanges[0].Change, "2026-05", 0)
	assertMonthValue(t, "high to low 1", summary.HighToLows[0].Period, summary.HighToLows[0].Drop, "2026-05", 0)
	assertMonthValue(t, "low 1", summary.Lows[0].Period, summary.Lows[0].Low, "2026-05", 0)
}

func TestReportMonthlyRowsUseOnBudgetStartEndChangeAndHighLow(t *testing.T) {
	ctx := context.Background()
	store, accounts, balances, _, _ := serviceStack(t)
	reports := ReportService{Accounts: store.Acct, Balances: store.Bal, Currencies: store.Cur, AppCurrency: "HKD", Now: store.Clock}
	cash, _, err := accounts.Create(ctx, "cash", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	card, _, err := accounts.Create(ctx, "credit-card", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	offBudget, _, err := accounts.Create(ctx, "investment", "HKD", false, "")
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
		{offBudget.ID, "2026-05-01", "9999.00"},
		{offBudget.ID, "2026-05-24", "19999.00"},
	} {
		if _, _, err := balances.Add(ctx, add.accountID, add.date, add.amount, ""); err != nil {
			t.Fatal(err)
		}
	}
	rows, warnings, err := reports.MonthlyRows(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %+v", warnings)
	}
	if len(rows) != 1 || rows[0].Period != "2026-05" {
		t.Fatalf("monthly rows = %+v", rows)
	}
	if rows[0].Coverage.Start != "2026-05-01" || rows[0].Coverage.End != "2026-05-24" {
		t.Fatalf("coverage = %+v", rows[0].Coverage)
	}
	metrics := rows[0].Metrics
	assertMoneyAmount(t, "start", metrics.Start, 100000)
	assertMoneyAmount(t, "end", metrics.End, 50000)
	assertMoneyAmount(t, "change", metrics.Change, -50000)
	assertMoneyAmount(t, "high", metrics.High, 150000)
	assertMoneyAmount(t, "low", metrics.Low, 50000)
	assertMoneyAmount(t, "high-to-low", metrics.HighToLow, -100000)
}

func TestReportMonthlyDetailIncludesAccountTreeRemainingRows(t *testing.T) {
	ctx := context.Background()
	store, accounts, balances, _, _ := serviceStack(t)
	reports := ReportService{Accounts: store.Acct, Balances: store.Bal, Currencies: store.Cur, AppCurrency: "HKD", Now: store.Clock}
	parent, _, err := accounts.Create(ctx, "bank", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	child, _, err := accounts.CreateChild(ctx, parent.ID, "wallet", "HKD", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, add := range []struct {
		accountID int64
		date      string
		amount    string
	}{
		{parent.ID, "2026-05-01", "1000.00"},
		{parent.ID, "2026-05-24", "1200.00"},
		{child.ID, "2026-05-01", "200.00"},
		{child.ID, "2026-05-24", "300.00"},
	} {
		if _, _, err := balances.Add(ctx, add.accountID, add.date, add.amount, ""); err != nil {
			t.Fatal(err)
		}
	}
	detail, err := reports.MonthlyDetail(ctx, "2026-05")
	if err != nil {
		t.Fatal(err)
	}
	if detail.Period != "2026-05" || detail.Coverage.Start != "2026-05-01" || detail.Coverage.End != "2026-05-24" {
		t.Fatalf("detail period/coverage = %+v", detail)
	}
	if len(detail.Rows) != 3 {
		t.Fatalf("account rows = %+v", detail.Rows)
	}
	parentRow, childRow, remainingRow := detail.Rows[0], detail.Rows[1], detail.Rows[2]
	if parentRow.Name != "bank" || parentRow.Depth != 0 || parentRow.Virtual {
		t.Fatalf("parent row = %+v", parentRow)
	}
	assertMoneyAmount(t, "parent start", parentRow.Metrics.Start, 100000)
	assertMoneyAmount(t, "parent end", parentRow.Metrics.End, 120000)
	if childRow.Name != "wallet" || childRow.Depth != 1 || childRow.Virtual {
		t.Fatalf("child row = %+v", childRow)
	}
	assertMoneyAmount(t, "child start", childRow.Metrics.Start, 20000)
	assertMoneyAmount(t, "child end", childRow.Metrics.End, 30000)
	if remainingRow.Name != "remaining" || remainingRow.Depth != 1 || !remainingRow.Virtual {
		t.Fatalf("remaining row = %+v", remainingRow)
	}
	assertMoneyAmount(t, "remaining start", remainingRow.Metrics.Start, 80000)
	assertMoneyAmount(t, "remaining end", remainingRow.Metrics.End, 90000)
}

func TestReportMonthlyAccountDetailShowsBoundaryAndSnapshotRows(t *testing.T) {
	ctx := context.Background()
	store, accounts, balances, _, _ := serviceStack(t)
	reports := ReportService{Accounts: store.Acct, Balances: store.Bal, Currencies: store.Cur, AppCurrency: "HKD", Now: store.Clock}
	cash, _, err := accounts.Create(ctx, "cash", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, add := range []struct {
		date   string
		amount string
	}{
		{"2026-03-01", "1000.00"},
		{"2026-03-12", "400.00"},
		{"2026-03-29", "1600.00"},
		{"2026-04-02", "2000.00"},
	} {
		if _, _, err := balances.Add(ctx, cash.ID, add.date, add.amount, ""); err != nil {
			t.Fatal(err)
		}
	}
	detail, err := reports.MonthlyAccountDetail(ctx, "2026-03", "cash")
	if err != nil {
		t.Fatal(err)
	}
	if detail.AccountName != "cash" || detail.Period != "2026-03" || detail.Coverage.Start != "2026-03-01" || detail.Coverage.End != "2026-04-01" {
		t.Fatalf("account detail = %+v", detail)
	}
	assertMoneyAmount(t, "detail start", detail.Metrics.Start, 100000)
	assertMoneyAmount(t, "detail end", detail.Metrics.End, 160000)
	if len(detail.Snapshots) != 4 {
		t.Fatalf("snapshots = %+v", detail.Snapshots)
	}
	wants := []struct {
		date   string
		amount int64
		note   string
	}{
		{"2026-03-01", 100000, "start boundary"},
		{"2026-03-12", 40000, "snapshot"},
		{"2026-03-29", 160000, "snapshot"},
		{"2026-04-01", 160000, "end boundary"},
	}
	for i, want := range wants {
		got := detail.Snapshots[i]
		if got.Date != want.date || got.Notes != want.note {
			t.Fatalf("snapshot %d = %+v, want date %s note %s", i, got, want.date, want.note)
		}
		assertMoneyAmount(t, fmt.Sprintf("snapshot %d", i), got.Balance, want.amount)
	}
}

func TestReportMonthlyCoverageUsesSharedPeriodBoundaries(t *testing.T) {
	ctx := context.Background()
	store, accounts, balances, _, _ := serviceStack(t)
	reports := ReportService{Accounts: store.Acct, Balances: store.Bal, Currencies: store.Cur, AppCurrency: "HKD", Now: store.Clock}
	cash, _, err := accounts.Create(ctx, "cash", "HKD", true, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := balances.Add(ctx, cash.ID, "2026-04-02", "1000.00", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := balances.Add(ctx, cash.ID, "2026-05-24", "1200.00", ""); err != nil {
		t.Fatal(err)
	}
	rows, _, err := reports.MonthlyRows(ctx, 2)
	if err != nil {
		t.Fatal(err)
	}
	if rows[0].Period != "2026-05" || rows[0].Coverage.Start != "2026-05-01" || rows[0].Coverage.End != "2026-05-24" {
		t.Fatalf("may coverage = %+v", rows[0])
	}
	if rows[1].Period != "2026-04" || rows[1].Coverage.Start != "2026-04-01" || rows[1].Coverage.End != "2026-05-01" {
		t.Fatalf("april coverage = %+v", rows[1])
	}
	if rows[1].Coverage.End != rows[0].Coverage.Start {
		t.Fatalf("coverage boundary should be shared, got april end %s and may start %s", rows[1].Coverage.End, rows[0].Coverage.Start)
	}
}

func TestReportMonthlyRowsDoNotDuplicateMonthsAfterLongMonths(t *testing.T) {
	ctx := context.Background()
	store, _, _, _, _ := serviceStack(t)
	store.Clock = func() time.Time { return time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC) }
	reports := ReportService{Accounts: store.Acct, Balances: store.Bal, Currencies: store.Cur, AppCurrency: "HKD", Now: store.Clock}
	rows, _, err := reports.MonthlyRows(ctx, 5)
	if err != nil {
		t.Fatal(err)
	}
	var periods []string
	for _, row := range rows {
		periods = append(periods, row.Period)
	}
	want := []string{"2026-05", "2026-04", "2026-03", "2026-02", "2026-01"}
	if strings.Join(periods, ",") != strings.Join(want, ",") {
		t.Fatalf("periods = %+v, want %+v", periods, want)
	}
}

func assertPeriod(t *testing.T, name, got, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("%s = %s, want %s", name, got, want)
	}
}

func assertTrend(t *testing.T, name string, got DashboardMonthTrendPoint, from, to string, want int64) {
	t.Helper()
	if got.FromPeriod != from || got.ToPeriod != to {
		t.Fatalf("%s periods = %+v, want %s -> %s", name, got, from, to)
	}
	assertMoneyAmount(t, name, got.Change, want)
}

func assertMonthValue(t *testing.T, name, gotPeriod string, got money.Money, wantPeriod string, want int64) {
	t.Helper()
	assertPeriod(t, name+" period", gotPeriod, wantPeriod)
	assertMoneyAmount(t, name, got, want)
}

func assertMoneyAmount(t *testing.T, name string, got money.Money, want int64) {
	t.Helper()
	if got.Amount != want {
		t.Fatalf("%s = %d, want %d", name, got.Amount, want)
	}
}
