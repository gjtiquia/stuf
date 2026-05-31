package model

import "testing"

func TestRouteBuildersAndParsers(t *testing.T) {
	name := "cash"
	date := "2026-05-21"

	path := accountPath(name)
	if got, ok := accountDetailName(path); !ok || got != name {
		t.Fatalf("accountDetailName(%q) = %q, %v", path, got, ok)
	}

	editPath := accountEditPathFor(name)
	if got, ok := accountEditName(editPath); !ok || got != name {
		t.Fatalf("accountEditName(%q) = %q, %v", editPath, got, ok)
	}
	if !accountEditPath(editPath) {
		t.Fatalf("accountEditPath(%q) = false", editPath)
	}

	listPath := accountBalanceListPath(name)
	if got, ok := balanceListName(listPath); !ok || got != name {
		t.Fatalf("balanceListName(%q) = %q, %v", listPath, got, ok)
	}

	addPath := accountBalanceAddPath(name)
	if got, ok := balanceAddName(addPath); !ok || got != name {
		t.Fatalf("balanceAddName(%q) = %q, %v", addPath, got, ok)
	}
	if !balanceAddPath(addPath) {
		t.Fatalf("balanceAddPath(%q) = false", addPath)
	}

	detailPath := accountBalancePath(name, date)
	if gotName, gotDate, ok := balanceDetailName(detailPath); !ok || gotName != name || gotDate != date {
		t.Fatalf("balanceDetailName(%q) = %q %q, %v", detailPath, gotName, gotDate, ok)
	}

	editBalancePath := accountBalanceEditPath(name, date)
	if gotName, gotDate, ok := balanceEditName(editBalancePath); !ok || gotName != name || gotDate != date {
		t.Fatalf("balanceEditName(%q) = %q %q, %v", editBalancePath, gotName, gotDate, ok)
	}
	if !balanceEditPath(editBalancePath) {
		t.Fatalf("balanceEditPath(%q) = false", editBalancePath)
	}

	budget := "groceries"
	if got, ok := budgetDetailName(budgetPath(budget)); !ok || got != budget {
		t.Fatalf("budgetDetailName = %q, %v", got, ok)
	}
	if got, ok := budgetEditName(budgetEditPathFor(budget)); !ok || got != budget {
		t.Fatalf("budgetEditName = %q, %v", got, ok)
	}
	if got, ok := budgetAllocationListName(budgetAllocationListPath(budget)); !ok || got != budget {
		t.Fatalf("budgetAllocationListName = %q, %v", got, ok)
	}
	if got, ok := budgetAllocationAddName(budgetAllocationAddPath(budget)); !ok || got != budget {
		t.Fatalf("budgetAllocationAddName = %q, %v", got, ok)
	}
	if gotName, gotID, ok := budgetAllocationEditName(budgetAllocationEditPath(budget, 42)); !ok || gotName != budget || gotID != 42 {
		t.Fatalf("budgetAllocationEditName = %q %d, %v", gotName, gotID, ok)
	}
	ledger := "alex"
	if got, ok := owedLedgerDetailName(owedLedgerPath(ledger)); !ok || got != ledger {
		t.Fatalf("owedLedgerDetailName = %q, %v", got, ok)
	}
	if got, ok := owedLedgerEditName(owedLedgerEditPathFor(ledger)); !ok || got != ledger {
		t.Fatalf("owedLedgerEditName = %q, %v", got, ok)
	}
	if got, ok := owedTransactionListName(owedTransactionListPath(ledger)); !ok || got != ledger {
		t.Fatalf("owedTransactionListName = %q, %v", got, ok)
	}
	if got, ok := owedTransactionAddName(owedTransactionAddPath(ledger)); !ok || got != ledger {
		t.Fatalf("owedTransactionAddName = %q, %v", got, ok)
	}
	if gotName, gotID, ok := owedTransactionEditName(owedTransactionEditPath(ledger, 42)); !ok || gotName != ledger || gotID != 42 {
		t.Fatalf("owedTransactionEditName = %q %d, %v", gotName, gotID, ok)
	}
	if gotName, gotID, ok := owedTransactionRefName(owedTransactionDetailPathFor(ledger, 42)); !ok || gotName != ledger || gotID != 42 {
		t.Fatalf("owedTransactionRefName = %q %d, %v", gotName, gotID, ok)
	}
	category := "daily"
	if got, ok := budgetCategoryDetailName(budgetCategoryPath(category)); !ok || got != category {
		t.Fatalf("budgetCategoryDetailName = %q, %v", got, ok)
	}
	if got, ok := budgetCategoryEditName(budgetCategoryEditPathFor(category)); !ok || got != category {
		t.Fatalf("budgetCategoryEditName = %q, %v", got, ok)
	}
	if got, ok := budgetCategoryBudgetCreateName(budgetCategoryBudgetCreatePath(category)); !ok || got != category {
		t.Fatalf("budgetCategoryBudgetCreateName = %q, %v", got, ok)
	}

	reportMonth := "2026-05"
	reportPath := reportMonthlyDetailPath(reportMonth)
	if gotMonth, ok := reportMonthlyDetailMonth(reportPath); !ok || gotMonth != reportMonth {
		t.Fatalf("reportMonthlyDetailMonth(%q) = %q, %v", reportPath, gotMonth, ok)
	}
	accountReportPath := reportMonthlyAccountPath(reportMonth, name)
	if gotMonth, gotName, ok := reportMonthlyAccount(accountReportPath); !ok || gotMonth != reportMonth || gotName != name {
		t.Fatalf("reportMonthlyAccount(%q) = %q %q, %v", accountReportPath, gotMonth, gotName, ok)
	}
	for _, invalid := range []string{routeReportsMonthly, "/reports/monthly/2026-5/", "/reports/monthly/2026-13/", "/reports/monthly/2026-05/extra/"} {
		if gotMonth, ok := reportMonthlyDetailMonth(invalid); ok {
			t.Fatalf("reportMonthlyDetailMonth(%q) = %q, true; want false", invalid, gotMonth)
		}
	}
	for _, invalid := range []string{reportPath, "/reports/monthly/2026-05/accounts/", "/reports/monthly/2026-05/accounts/cash/extra/", "/reports/monthly/2026-13/accounts/cash/"} {
		if gotMonth, gotName, ok := reportMonthlyAccount(invalid); ok {
			t.Fatalf("reportMonthlyAccount(%q) = %q %q, true; want false", invalid, gotMonth, gotName)
		}
	}
}
