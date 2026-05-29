package model

import (
	"fmt"
	"strings"
	"time"

	"stuf/internal/component"
	"stuf/internal/money"
	"stuf/internal/service"
)

const reportMonthCount = 12

func reportActions() []string {
	return []string{"monthly", "rolling 3 months (TODO)", "rolling 6 months (TODO)", "rolling 12 months (TODO)", "year-to-date (TODO)", "annual (TODO)"}
}

func reportRoutes() []string {
	return []string{routeReportsMonthly, "/reports/rolling-3/", "/reports/rolling-6/", "/reports/rolling-12/", "/reports/year-to-date/", "/reports/annual/"}
}

func (a App) reportsMenuKey(s string) App {
	return a.menuKey(s, reportRoutes())
}

func (a App) reportsMenuScreen() screen {
	context, err := a.reportsMenuContext()
	if err != nil {
		return screen{Path: routeReports, Body: "error: " + err.Error() + "\n"}
	}
	return screen{Path: routeReports, Context: context, Actions: reportActions()}
}

func (a App) reportsMenuContext() (string, error) {
	rows, warnings, err := a.Svc.Reports.MonthlyRows(a.ctx, reportMonthCount)
	if err != nil {
		return "", err
	}
	zero := money.Money{Scale: 2}
	if len(rows) > 0 {
		zero = money.Money{Scale: rows[0].Metrics.Change.Scale}
	}
	current := sumMonthlyChanges(rows, 1, zero)
	rolling3 := sumMonthlyChanges(rows, 3, zero)
	rolling6 := sumMonthlyChanges(rows, 6, zero)
	rolling12 := sumMonthlyChanges(rows, 12, zero)
	ytd := sumYearToDate(rows, zero)
	values := alignedMoneyValues(
		component.MoneyCell(current, a.Config.Config.Currency),
		component.MoneyCell(rolling3, a.Config.Config.Currency),
		component.MoneyCell(rolling6, a.Config.Config.Currency),
		component.MoneyCell(rolling12, a.Config.Config.Currency),
		component.MoneyCell(ytd, a.Config.Config.Currency),
	)
	context := fmt.Sprintf(`on-budget net change

current month     : %s
rolling 3 months  : %s
rolling 6 months  : %s
rolling 12 months : %s
year-to-date      : %s`, values[0], values[1], values[2], values[3], values[4])
	if warningText := dashboardWarnings(warnings); warningText != "" {
		context += "\n" + warningText
	}
	return strings.TrimRight(context, "\n"), nil
}

func sumMonthlyChanges(rows []service.ReportMonthlyRow, count int, zero money.Money) money.Money {
	total := zero
	if count > len(rows) {
		count = len(rows)
	}
	for _, row := range rows[:count] {
		total, _ = total.Add(row.Metrics.Change)
	}
	return total
}

func sumYearToDate(rows []service.ReportMonthlyRow, zero money.Money) money.Money {
	total := zero
	year := ""
	if len(rows) > 0 {
		year = strings.SplitN(rows[0].Period, "-", 2)[0]
	}
	for _, row := range rows {
		if strings.HasPrefix(row.Period, year+"-") {
			total, _ = total.Add(row.Metrics.Change)
		}
	}
	return total
}

func (a App) reportMonthlyListKey(s string) App {
	rows, err := a.filteredReportMonthlyRows()
	if err != nil {
		a.Error = err.Error()
		return a
	}
	if (s == "enter" || s == "right") && len(rows) > 0 {
		a = a.navSetMenu(clampListCursor(a.Menu, len(rows)))
		return a.navPush(reportMonthlyDetailPath(rows[a.Menu].Period), 0)
	}
	switch s {
	case "left":
		a.Error = ""
		return a.goBack()
	case "up", "shift+tab":
		if len(rows) > 0 {
			a = a.navSetMenu(clampListCursor(a.Menu-1, len(rows)))
		}
	case "down", "tab":
		if len(rows) > 0 {
			a = a.navSetMenu(clampListCursor(a.Menu+1, len(rows)))
		}
	default:
		if result, handled := handleFilterableListKey(s, a.listFilter(), a.Menu, len(rows)); handled {
			a.setListFilter(result.filter)
			nextRows, _ := a.filteredReportMonthlyRows()
			a = a.navSetMenu(clampListCursor(result.menu, len(nextRows)))
		}
	}
	return a
}

func (a App) reportMonthlyListScreen() screen {
	context, err := a.reportMonthlyListContext()
	if err != nil {
		return screen{Path: routeReportsMonthly, Body: "error: " + err.Error() + "\n"}
	}
	body, err := a.reportMonthlyListBody()
	if err != nil {
		return screen{Path: routeReportsMonthly, Context: context, Body: "error: " + err.Error() + "\n"}
	}
	return screen{Path: routeReportsMonthly, Context: context, Body: body, Help: reportMonthlyListHelp()}
}

func (a App) reportMonthlyListContext() (string, error) {
	rows, warnings, err := a.Svc.Reports.MonthlyRows(a.ctx, 1)
	if err != nil {
		return "", err
	}
	if len(rows) == 0 {
		return "", nil
	}
	context := "current month\n\n" + reportSummaryContext(rows[0].Period, rows[0].Coverage.Start+" -> "+rows[0].Coverage.End, rows[0].Metrics, a.Config.Config.Currency)
	if warningText := dashboardWarnings(warnings); warningText != "" {
		context += "\n" + warningText
	}
	return strings.TrimRight(context, "\n"), nil
}

func (a App) reportMonthlyListBody() (string, error) {
	rows, err := a.filteredReportMonthlyRows()
	if err != nil {
		return "", err
	}
	lines := []string{"> filter   : " + placeholder(a.listFilter(), "(type anything...)"), ""}
	if len(rows) == 0 {
		lines = append(lines, "  month | start | end | change | high | low | high-to-low", "  (no results)")
		return strings.Join(lines, "\n") + "\n", nil
	}
	layout := reportMonthlyTableLayout(rows, a.Config.Config.Currency)
	lines = append(lines, layout.Header("  "))
	for i, row := range rows {
		prefix := "  "
		if i == a.Menu {
			prefix = "> "
		}
		lines = append(lines, layout.RowCells(prefix, reportMonthlyRowCells(row, a.Config.Config.Currency)))
	}
	return strings.Join(lines, "\n") + "\n", nil
}

func (a App) filteredReportMonthlyRows() ([]service.ReportMonthlyRow, error) {
	rows, _, err := a.Svc.Reports.MonthlyRows(a.ctx, reportMonthCount)
	if err != nil {
		return nil, err
	}
	filter := strings.ToLower(a.listFilter())
	if filter == "" {
		return rows, nil
	}
	var out []service.ReportMonthlyRow
	for _, row := range rows {
		if strings.Contains(strings.ToLower(row.Period), filter) {
			out = append(out, row)
		}
	}
	return out, nil
}

func reportMonthlyTableLayout(rows []service.ReportMonthlyRow, cur string) component.TableLayout {
	tableRows := make([][]component.Cell, len(rows))
	for i, row := range rows {
		tableRows[i] = reportMonthlyRowCells(row, cur)
	}
	return component.NewTableLayoutCells([]string{"month", "start", "end", "change", "high", "low", "high-to-low"}, tableRows)
}

func reportMonthlyRowCells(row service.ReportMonthlyRow, cur string) []component.Cell {
	return []component.Cell{
		component.TextCell(row.Period),
		component.MoneyCell(row.Metrics.Start, cur),
		component.MoneyCell(row.Metrics.End, cur),
		component.MoneyCell(row.Metrics.Change, cur),
		component.MoneyCell(row.Metrics.High, cur),
		component.MoneyCell(row.Metrics.Low, cur),
		component.MoneyCell(row.Metrics.HighToLow, cur),
	}
}

func (a App) reportMonthlyDetailKey(s, month string) App {
	rows, err := a.filteredReportAccountRows(month)
	if err != nil {
		a.Error = err.Error()
		return a
	}
	switch {
	case s == "enter" && len(rows) > 0:
		a = a.navSetMenu(clampListCursor(a.Menu, len(rows)))
		row := rows[a.Menu]
		if row.Virtual {
			return a
		}
		return a.navPush(reportMonthlyAccountPath(month, row.Name), 0)
	case isItemPrevKey(s):
		return a.navReplace(reportMonthlyDetailPath(shiftReportMonth(month, -1)), a.Menu)
	case isItemNextKey(s):
		return a.navReplace(reportMonthlyDetailPath(shiftReportMonth(month, 1)), a.Menu)
	}
	switch s {
	case "up", "shift+tab":
		if len(rows) > 0 {
			a = a.navSetMenu(clampListCursor(a.Menu-1, len(rows)))
		}
	case "down", "tab":
		if len(rows) > 0 {
			a = a.navSetMenu(clampListCursor(a.Menu+1, len(rows)))
		}
	default:
		if result, handled := handleFilterableListKey(s, a.listFilter(), a.Menu, len(rows)); handled {
			a.setListFilter(result.filter)
			nextRows, _ := a.filteredReportAccountRows(month)
			a = a.navSetMenu(clampListCursor(result.menu, len(nextRows)))
		}
	}
	return a
}

func (a App) reportMonthlyAccountKey(s, month, name string) App {
	detail, err := a.Svc.Reports.MonthlyAccountDetail(a.ctx, month, name)
	if err != nil {
		a.Error = err.Error()
		return a
	}
	switch {
	case isItemPrevKey(s):
		return a.navReplace(reportMonthlyAccountPath(shiftReportMonth(month, -1), name), a.Menu)
	case isItemNextKey(s):
		return a.navReplace(reportMonthlyAccountPath(shiftReportMonth(month, 1), name), a.Menu)
	case isVerticalPrevKey(s):
		if len(detail.Snapshots) > 0 {
			a = a.navSetMenu(clampListCursor(a.Menu-1, len(detail.Snapshots)))
		}
	case isVerticalNextKey(s):
		if len(detail.Snapshots) > 0 {
			a = a.navSetMenu(clampListCursor(a.Menu+1, len(detail.Snapshots)))
		}
	}
	return a
}

func (a App) reportMonthlyDetailScreen(month string) screen {
	detail, err := a.Svc.Reports.MonthlyDetail(a.ctx, month)
	if err != nil {
		return screen{Path: reportMonthlyDetailPath(month), Body: "error: " + err.Error() + "\n"}
	}
	body, err := a.reportMonthlyDetailBody(detail)
	if err != nil {
		return screen{Path: reportMonthlyDetailPath(month), Context: reportMonthlyDetailContext(detail), Body: "error: " + err.Error() + "\n"}
	}
	return screen{
		Path:    reportMonthlyDetailPath(month),
		Context: reportMonthlyDetailContext(detail),
		Body:    body,
		Help:    reportMonthlyDetailHelp(),
	}
}

func reportMonthlyDetailContext(detail service.ReportMonthlyDetail) string {
	context := reportSummaryContext(detail.Period, detail.Coverage.Start+" -> "+detail.Coverage.End, detail.Metrics, detail.AppCurrency)
	if warningText := dashboardWarnings(detail.Warnings); warningText != "" {
		context += "\n" + warningText
	}
	return strings.TrimRight(context, "\n")
}

func (a App) reportMonthlyAccountScreen(month, name string) screen {
	detail, err := a.Svc.Reports.MonthlyAccountDetail(a.ctx, month, name)
	if err != nil {
		return screen{Path: reportMonthlyAccountPath(month, name), Body: "error: " + err.Error() + "\n"}
	}
	return screen{
		Path:    reportMonthlyAccountPath(month, name),
		Context: reportMonthlyAccountContext(detail),
		Body:    reportMonthlyAccountBody(detail, a.Menu),
		Help:    reportMonthlyAccountHelp(),
	}
}

func reportMonthlyAccountContext(detail service.ReportMonthlyAccountDetail) string {
	values := alignedMoneyValues(
		component.MoneyCell(detail.Metrics.Start, detail.AppCurrency),
		component.MoneyCell(detail.Metrics.End, detail.AppCurrency),
		component.MoneyCell(detail.Metrics.Change, detail.AppCurrency),
		component.MoneyCell(detail.Metrics.High, detail.AppCurrency),
		component.MoneyCell(detail.Metrics.Low, detail.AppCurrency),
		component.MoneyCell(detail.Metrics.HighToLow, detail.AppCurrency),
	)
	context := fmt.Sprintf(`account     : %s
period      : %s
coverage    : %s -> %s

start       : %s
end         : %s
change      : %s

high        : %s
low         : %s
high-to-low : %s`, detail.AccountName, detail.Period, detail.Coverage.Start, detail.Coverage.End, values[0], values[1], values[2], values[3], values[4], values[5])
	if warningText := dashboardWarnings(detail.Warnings); warningText != "" {
		context += "\n" + warningText
	}
	return strings.TrimRight(context, "\n")
}

func reportMonthlyAccountBody(detail service.ReportMonthlyAccountDetail, selected int) string {
	lines := []string{"snapshots"}
	if len(detail.Snapshots) == 0 {
		lines = append(lines, "  date | balance | kind | notes", "  (no snapshots)")
		return strings.Join(lines, "\n") + "\n"
	}
	tableRows := make([][]component.Cell, len(detail.Snapshots))
	for i, row := range detail.Snapshots {
		tableRows[i] = reportSnapshotRowCells(row, detail.AppCurrency)
	}
	layout := component.NewTableLayoutCells([]string{"date", "balance", "kind", "notes"}, tableRows)
	lines = append(lines, layout.Header("  "))
	for i, row := range detail.Snapshots {
		prefix := "  "
		if i == selected {
			prefix = "> "
		}
		lines = append(lines, layout.RowCells(prefix, reportSnapshotRowCells(row, detail.AppCurrency)))
	}
	return strings.Join(lines, "\n") + "\n"
}

func reportSnapshotRowCells(row service.ReportSnapshotRow, cur string) []component.Cell {
	return []component.Cell{
		component.TextCell(row.Date),
		component.MoneyCell(row.Balance, cur),
		component.TextCell(row.Kind),
		component.TextCell(row.Notes),
	}
}

func reportSummaryContext(period, coverage string, metrics service.ReportPeriodMetrics, cur string) string {
	values := alignedMoneyValues(
		component.MoneyCell(metrics.Start, cur),
		component.MoneyCell(metrics.End, cur),
		component.MoneyCell(metrics.Change, cur),
		component.MoneyCell(metrics.High, cur),
		component.MoneyCell(metrics.Low, cur),
		component.MoneyCell(metrics.HighToLow, cur),
	)
	return fmt.Sprintf(`period      : %s
coverage    : %s

on-budget
start       : %s
end         : %s
change      : %s

high        : %s
low         : %s
high-to-low : %s`, period, coverage, values[0], values[1], values[2], values[3], values[4], values[5])
}

func (a App) reportMonthlyDetailBody(detail service.ReportMonthlyDetail) (string, error) {
	rows := filterReportAccountRows(detail.Rows, a.listFilter())
	lines := []string{"> filter   : " + placeholder(a.listFilter(), "(type anything...)"), "", "  on-budget accounts"}
	if len(rows) == 0 {
		lines = append(lines, "  account | start | end | change | high | low | high-to-low", "  (no results)")
		return strings.Join(lines, "\n") + "\n", nil
	}
	layout := reportAccountTableLayout(rows, detail.AppCurrency)
	lines = append(lines, layout.Header("  "))
	for i, row := range rows {
		prefix := "  "
		if i == a.Menu {
			prefix = "> "
		}
		lines = append(lines, layout.RowCells(prefix, reportAccountRowCells(row, detail.AppCurrency)))
	}
	return strings.Join(lines, "\n") + "\n", nil
}

func (a App) filteredReportAccountRows(month string) ([]service.ReportAccountRow, error) {
	detail, err := a.Svc.Reports.MonthlyDetail(a.ctx, month)
	if err != nil {
		return nil, err
	}
	return filterReportAccountRows(detail.Rows, a.listFilter()), nil
}

func filterReportAccountRows(rows []service.ReportAccountRow, filter string) []service.ReportAccountRow {
	filter = strings.ToLower(filter)
	if filter == "" {
		return rows
	}
	var out []service.ReportAccountRow
	for _, row := range rows {
		if strings.Contains(strings.ToLower(row.Name), filter) {
			out = append(out, row)
		}
	}
	return out
}

func reportAccountTableLayout(rows []service.ReportAccountRow, cur string) component.TableLayout {
	tableRows := make([][]component.Cell, len(rows))
	for i, row := range rows {
		tableRows[i] = reportAccountRowCells(row, cur)
	}
	return component.NewTableLayoutCells([]string{"account", "start", "end", "change", "high", "low", "high-to-low"}, tableRows)
}

func reportAccountRowCells(row service.ReportAccountRow, cur string) []component.Cell {
	return []component.Cell{
		component.TextCell(strings.Repeat("  ", row.Depth) + row.Name),
		component.MoneyCell(row.Metrics.Start, cur),
		component.MoneyCell(row.Metrics.End, cur),
		component.MoneyCell(row.Metrics.Change, cur),
		component.MoneyCell(row.Metrics.High, cur),
		component.MoneyCell(row.Metrics.Low, cur),
		component.MoneyCell(row.Metrics.HighToLow, cur),
	}
}

func shiftReportMonth(month string, delta int) string {
	t, err := time.Parse("2006-01", month)
	if err != nil {
		return month
	}
	return t.AddDate(0, delta, 0).Format("2006-01")
}

func reportMonthlyListHelp() []string {
	return []string{"type          : filter", "h/l           : type in filter", "up/down       : navigate", "left/right    : back/open", "enter         : confirm", "esc           : back", "?             : help", "ctrl-z        : undo"}
}

func reportMonthlyDetailHelp() []string {
	return []string{"type          : filter", "up/down       : navigate", "left/h        : previous month", "right/l       : next month", "backspace     : edit filter", "enter         : confirm", "esc           : back", "?             : help", "ctrl-z        : undo"}
}

func reportMonthlyAccountHelp() []string {
	return []string{"up/down       : navigate", "left/h        : previous month", "right/l       : next month", "esc           : back", "?             : help", "ctrl-z        : undo"}
}
