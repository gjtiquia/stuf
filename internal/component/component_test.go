package component

import (
	"strings"
	"testing"

	"stuf/internal/money"
)

func TestSmallComponents(t *testing.T) {
	if got := (TextInput{Label: "name", Value: "cash"}).View(); got != "name: cash" {
		t.Fatal(got)
	}
	if got := (SelectInput{Options: []string{"USD", "HKD"}, Index: 1}).Selected(); got != "HKD" {
		t.Fatal(got)
	}
	if got := len((FilterList{Items: []string{"cash", "bank"}, Filter: "a"}).Visible()); got != 2 {
		t.Fatal(got)
	}
	if got := (Form{Fields: map[string]string{"name": "cash"}}).Value("name"); got != "cash" {
		t.Fatal(got)
	}
}

func TestTableLayoutAlignsDynamicColumns(t *testing.T) {
	layout := NewTableLayout(
		[]string{"name", "balance", "notes"},
		[][]string{
			{"cash", "HKD 10.00", ""},
			{"student-loan", "HKD (200,000.00)", "negative until paid"},
		},
	)

	if got, want := layout.Header("  "), "  name         | balance          | notes"; got != want {
		t.Fatalf("header = %q, want %q", got, want)
	}
	if got, want := layout.Row("> ", []string{"cash", "HKD 10.00", ""}), "> cash         | HKD 10.00        | "; got != want {
		t.Fatalf("short row = %q, want %q", got, want)
	}
	if got, want := layout.Row("  ", []string{"student-loan", "HKD (200,000.00)", "negative until paid"}), "  student-loan | HKD (200,000.00) | negative until paid"; got != want {
		t.Fatalf("long row = %q, want %q", got, want)
	}
}

func TestTableLayoutTreatsPrefixOutsideColumnWidths(t *testing.T) {
	layout := NewTableLayout([]string{"date", "balance"}, [][]string{{"2026-05-25", "HKD 13,010.40"}})

	unselected := layout.Row("  ", []string{"2026-05-25", "HKD 13,010.40"})
	selected := layout.Row("> ", []string{"2026-05-25", "HKD 13,010.40"})
	if strings.Index(unselected, "|") != strings.Index(selected, "|") {
		t.Fatalf("prefix should not shift columns:\nunselected: %q\nselected:   %q", unselected, selected)
	}
}

func TestTableLayoutAlignsMoneyCells(t *testing.T) {
	rows := [][]Cell{
		{TextCell("small"), MoneyCell(money.Money{Amount: 1000, Scale: 2}, "HKD"), TextCell("")},
		{TextCell("medium"), MoneyCell(money.Money{Amount: 100000, Scale: 2}, "HKD"), TextCell("")},
		{TextCell("large"), MoneyCell(money.Money{Amount: 123456789, Scale: 2}, "HKD"), TextCell("")},
		{TextCell("negative"), MoneyCell(money.Money{Amount: -123456, Scale: 2}, "HKD"), TextCell("")},
	}
	layout := NewTableLayoutCells([]string{"name", "balance", "notes"}, rows)

	rendered := []string{
		layout.RowCells("  ", rows[0]),
		layout.RowCells("> ", rows[1]),
		layout.RowCells("  ", rows[2]),
		layout.RowCells("  ", rows[3]),
	}
	decimalAt := -1
	commaAt := -1
	for _, line := range rendered {
		if strings.Index(line, "HKD") < 0 {
			t.Fatalf("currency missing from %q", line)
		}
		gotDecimal := strings.Index(line, ".")
		if decimalAt < 0 {
			decimalAt = gotDecimal
		} else if gotDecimal != decimalAt {
			t.Fatalf("decimal shifted:\n%s", strings.Join(rendered, "\n"))
		}
		if strings.Contains(line, ",") {
			gotComma := strings.LastIndex(line, ",")
			if commaAt < 0 {
				commaAt = gotComma
			} else if gotComma != commaAt {
				t.Fatalf("comma shifted:\n%s", strings.Join(rendered, "\n"))
			}
		}
	}
	if strings.Index(rendered[0], "|") != strings.Index(rendered[1], "|") {
		t.Fatalf("prefix should not shift columns:\n%s", strings.Join(rendered, "\n"))
	}
	if !strings.Contains(rendered[3], "HKD (") || !strings.Contains(rendered[3], ")") {
		t.Fatalf("negative accounting parens missing:\n%s", rendered[3])
	}
}

func TestMoneyCellWithTrailingKeepsPrimaryAmountAligned(t *testing.T) {
	rows := [][]Cell{
		{TextCell("cash"), MoneyCell(money.Money{Amount: 100000, Scale: 2}, "HKD")},
		{TextCell("yen"), MoneyCellWithTrailing(money.Money{Amount: 100000, Scale: 2}, "HKD", "(JPY 20,000)")},
		{TextCell("debt"), MoneyCell(money.Money{Amount: -2500, Scale: 2}, "HKD")},
	}
	layout := NewTableLayoutCells([]string{"name", "balance"}, rows)
	lines := []string{
		layout.RowCells("  ", rows[0]),
		layout.RowCells("  ", rows[1]),
		layout.RowCells("  ", rows[2]),
	}
	if strings.Index(lines[0], ".") != strings.Index(lines[1], ".") || strings.Index(lines[1], ".") != strings.Index(lines[2], ".") {
		t.Fatalf("primary decimals should align:\n%s", strings.Join(lines, "\n"))
	}
	if !strings.Contains(lines[1], "(JPY 20,000)") {
		t.Fatalf("trailing native amount missing:\n%s", lines[1])
	}
}

func TestMoneyColumnAlignsDashboardShape(t *testing.T) {
	cells := []Cell{
		MoneyCell(money.Money{Amount: 1301040, Scale: 2}, "HKD"),
		MoneyCell(money.Money{Amount: 0, Scale: 2}, "HKD"),
		MoneyCell(money.Money{Amount: -2394348, Scale: 2}, "HKD"),
		MoneyCell(money.Money{Amount: -2394348, Scale: 2}, "HKD"),
	}
	column := NewMoneyColumn(cells...)
	lines := []string{
		column.Render(cells[0]),
		column.Render(cells[1]),
		column.Render(cells[2]),
		column.Render(cells[3]),
	}
	decimalAt := strings.Index(lines[0], ".")
	for _, line := range lines[1:] {
		if got := strings.Index(line, "."); got != decimalAt {
			t.Fatalf("dashboard-shaped money decimals shifted:\n%s", strings.Join(lines, "\n"))
		}
	}
	if !strings.Contains(lines[2], "HKD (23,943.48)") {
		t.Fatalf("negative accounting output changed unexpectedly: %q", lines[2])
	}
}
