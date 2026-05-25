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

type TableLayout struct {
	headers []string
	widths  []int
}

func NewTableLayout(headers []string, rows [][]string) TableLayout {
	widths := make([]int, len(headers))
	for i, header := range headers {
		widths[i] = len(header)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i >= len(widths) {
				break
			}
			widths[i] = max(widths[i], len(cell))
		}
	}
	return TableLayout{headers: append([]string(nil), headers...), widths: widths}
}

func (l TableLayout) Header(prefix string) string {
	return l.Row(prefix, l.headers)
}

func (l TableLayout) Row(prefix string, cells []string) string {
	parts := make([]string, len(l.headers))
	for i := range l.headers {
		cell := ""
		if i < len(cells) {
			cell = cells[i]
		}
		if i < len(l.headers)-1 {
			parts[i] = padRight(cell, l.widths[i])
		} else {
			parts[i] = cell
		}
	}
	return prefix + strings.Join(parts, " | ")
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
