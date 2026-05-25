package component

import (
	"strings"

	"stuf/internal/money"
)

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
	columns []tableColumn
}

type Cell struct {
	text     string
	money    *moneyCell
	trailing string
}

type moneyCell struct {
	parts money.FormatParts
}

type tableColumn struct {
	width         int
	money         bool
	currencyWidth int
	numberWidth   int
	hasNegative   bool
}

type MoneyColumn struct {
	column tableColumn
}

func TextCell(text string) Cell {
	return Cell{text: text}
}

func MoneyCell(amount money.Money, currencyCode string) Cell {
	return Cell{money: &moneyCell{parts: amount.FormatParts(currencyCode)}}
}

func MoneyCellWithTrailing(amount money.Money, currencyCode, trailing string) Cell {
	cell := MoneyCell(amount, currencyCode)
	cell.trailing = trailing
	return cell
}

func NewMoneyColumn(cells ...Cell) MoneyColumn {
	column := tableColumn{money: true}
	for _, cell := range cells {
		if cell.money == nil {
			continue
		}
		column.currencyWidth = max(column.currencyWidth, len(cell.money.parts.Currency))
		column.numberWidth = max(column.numberWidth, len(cell.money.parts.Number))
		column.hasNegative = column.hasNegative || cell.money.parts.Negative
	}
	column.width = moneyWidth(column)
	for _, cell := range cells {
		if cell.money == nil {
			continue
		}
		column.width = max(column.width, len(renderMoney(cell, column)))
	}
	return MoneyColumn{column: column}
}

func (c MoneyColumn) Render(cell Cell) string {
	if cell.money == nil {
		return cell.text
	}
	return renderMoney(cell, c.column)
}

func NewTableLayout(headers []string, rows [][]string) TableLayout {
	cellRows := make([][]Cell, 0, len(rows))
	for _, row := range rows {
		cellRow := make([]Cell, 0, len(row))
		for _, cell := range row {
			cellRow = append(cellRow, TextCell(cell))
		}
		cellRows = append(cellRows, cellRow)
	}
	return NewTableLayoutCells(headers, cellRows)
}

func NewTableLayoutCells(headers []string, rows [][]Cell) TableLayout {
	widths := make([]int, len(headers))
	columns := make([]tableColumn, len(headers))
	for i, header := range headers {
		widths[i] = len(header)
		columns[i].width = len(header)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i >= len(widths) {
				break
			}
			if cell.money != nil {
				columns[i].money = true
				columns[i].currencyWidth = max(columns[i].currencyWidth, len(cell.money.parts.Currency))
				columns[i].numberWidth = max(columns[i].numberWidth, len(cell.money.parts.Number))
				columns[i].hasNegative = columns[i].hasNegative || cell.money.parts.Negative
				continue
			}
			columns[i].width = max(columns[i].width, len(cell.text))
		}
	}
	for i := range columns {
		if columns[i].money {
			columns[i].width = moneyWidth(columns[i])
			for _, row := range rows {
				if i >= len(row) || row[i].money == nil {
					continue
				}
				columns[i].width = max(columns[i].width, len(renderMoney(row[i], columns[i])))
			}
			columns[i].width = max(columns[i].width, len(headers[i]))
		}
		widths[i] = columns[i].width
	}
	return TableLayout{headers: append([]string(nil), headers...), widths: widths, columns: columns}
}

func (l TableLayout) Header(prefix string) string {
	return l.Row(prefix, l.headers)
}

func (l TableLayout) Row(prefix string, cells []string) string {
	cellRow := make([]Cell, 0, len(cells))
	for _, cell := range cells {
		cellRow = append(cellRow, TextCell(cell))
	}
	return l.RowCells(prefix, cellRow)
}

func (l TableLayout) RowCells(prefix string, cells []Cell) string {
	parts := make([]string, len(l.headers))
	for i := range l.headers {
		cell := TextCell("")
		if i < len(cells) {
			cell = cells[i]
		}
		rendered := l.renderCell(i, cell)
		if i < len(l.headers)-1 {
			parts[i] = padRight(rendered, l.widths[i])
		} else {
			parts[i] = rendered
		}
	}
	return prefix + strings.Join(parts, " | ")
}

func (l TableLayout) renderCell(index int, cell Cell) string {
	if index < len(l.columns) && cell.money != nil {
		return renderMoney(cell, l.columns[index])
	}
	return cell.text
}

func moneyWidth(column tableColumn) int {
	width := column.currencyWidth + 1 + column.numberWidth
	if column.hasNegative {
		width += 2
	}
	return width
}

func renderMoney(cell Cell, column tableColumn) string {
	parts := cell.money.parts
	var b strings.Builder
	b.WriteString(padRight(parts.Currency, column.currencyWidth))
	b.WriteByte(' ')
	if column.hasNegative {
		if parts.Negative {
			b.WriteByte('(')
		} else {
			b.WriteByte(' ')
		}
	}
	b.WriteString(padLeft(parts.Number, column.numberWidth))
	if column.hasNegative {
		if parts.Negative {
			b.WriteByte(')')
		} else {
			b.WriteByte(' ')
		}
	}
	if cell.trailing != "" {
		b.WriteByte(' ')
		b.WriteString(cell.trailing)
	}
	return b.String()
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

func padLeft(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return strings.Repeat(" ", width-len(s)) + s
}
