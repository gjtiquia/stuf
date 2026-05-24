package component

import "testing"

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
