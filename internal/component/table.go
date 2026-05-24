package component

import "strings"

type Table struct {
	Rows [][]string
}

func (t Table) View() string {
	var lines []string
	for _, row := range t.Rows {
		lines = append(lines, strings.Join(row, "  "))
	}
	return strings.Join(lines, "\n")
}
