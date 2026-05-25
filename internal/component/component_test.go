package component

import (
	"strings"
	"testing"
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
