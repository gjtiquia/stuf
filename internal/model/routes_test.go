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

	reportMonth := "2026-05"
	reportPath := reportMonthlyDetailPath(reportMonth)
	if gotMonth, ok := reportMonthlyDetailMonth(reportPath); !ok || gotMonth != reportMonth {
		t.Fatalf("reportMonthlyDetailMonth(%q) = %q, %v", reportPath, gotMonth, ok)
	}
	for _, invalid := range []string{routeReportsMonthly, "/reports/monthly/2026-5/", "/reports/monthly/2026-13/", "/reports/monthly/2026-05/extra/"} {
		if gotMonth, ok := reportMonthlyDetailMonth(invalid); ok {
			t.Fatalf("reportMonthlyDetailMonth(%q) = %q, true; want false", invalid, gotMonth)
		}
	}
}
