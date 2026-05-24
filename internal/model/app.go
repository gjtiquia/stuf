package model

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"stuf/internal/config"
	"stuf/internal/money"
	"stuf/internal/repo"
	"stuf/internal/service"
)

type Services struct {
	Accounts  service.AccountService
	Balances  service.BalanceService
	Dashboard service.DashboardService
	History   service.HistoryService
	Backup    func(context.Context) (string, error)
}

type App struct {
	ctx             context.Context
	Svc             Services
	Config          config.Loaded
	Path            string
	Menu            int
	History         []service.SessionEntry
	Error           string
	Help            bool
	ExitAsk         bool
	LastBackup      string
	Form            map[string]string
	Field           int
	SelectedAccount string
}

type screen struct {
	Path    string
	Body    string
	Actions []string
	Help    []string
}

const currencyPageSize = 8

func New(ctx context.Context, svc Services, cfg config.Loaded) App {
	return App{ctx: ctx, Svc: svc, Config: cfg, Path: "/", Form: map[string]string{}}
}

func (a App) Init() tea.Cmd { return nil }

func (a App) notesFocused() bool {
	switch {
	case a.Path == "/accounts/create/":
		return a.Field == 3
	case accountEditPath(a.Path):
		return a.Field == 3
	case balanceAddPath(a.Path), balanceEditPath(a.Path):
		return a.Field == 2
	default:
		return false
	}
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return a, nil
	}
	s := key.String()
	if s == "ctrl+c" {
		a.History = nil
		return a, tea.Quit
	}
	if s == "?" && !a.notesFocused() {
		a.Help = !a.Help
		return a, nil
	}
	if a.Help && s == "esc" {
		a.Help = false
		return a, nil
	}
	if s == "ctrl+z" {
		return a.undo(), nil
	}
	if a.ExitAsk {
		if s == "enter" || s == "n" || s == "esc" {
			a.ExitAsk = false
			return a, nil
		}
		if s == "y" {
			a.History = nil
			return a, tea.Quit
		}
	}
	if s == "esc" {
		if a.Path == "/" {
			a.ExitAsk = true
			return a, nil
		}
		a.Error = ""
		a.Path = parentPath(a.Path)
		return a, nil
	}
	switch a.Path {
	case "/":
		a = a.menuKey(s, []string{"/accounts/", "/transactions/", "/budgets/", "/owed/", "/reports/", "/settings/", "/backup/"})
	case "/accounts/":
		a = a.menuKey(s, []string{"/accounts/", "/accounts/list/", "/accounts/hidden/", "/accounts/create/"})
	case "/accounts/list/":
		a = a.accountListKey(s, false)
	case "/accounts/hidden/":
		a = a.accountListKey(s, true)
	case "/accounts/create/":
		a = a.accountCreateKey(s)
	case "/backup/":
		if s == "enter" || s == "1" {
			p, err := a.Svc.Backup(a.ctx)
			if err != nil {
				a.Error = err.Error()
			} else {
				a.Error = ""
				a.LastBackup = p
			}
		}
	default:
		if name, ok := accountDetailName(a.Path); ok {
			a = a.accountDetailKey(s, name)
		} else if name, ok := balanceAddName(a.Path); ok {
			a = a.balanceAddKey(s, name)
		} else if name, date, ok := balanceEditName(a.Path); ok {
			a = a.balanceEditKey(s, name, date)
		} else if name, date, ok := balanceDetailName(a.Path); ok {
			a = a.balanceDetailKey(s, name, date)
		} else if name, ok := balancesName(a.Path); ok {
			a = a.balanceListKey(s, name)
		} else if name, ok := accountEditName(a.Path); ok {
			a = a.accountEditKey(s, name)
		}
	}
	return a, nil
}

func (a App) menuKey(s string, routes []string) App {
	switch s {
	case "down", "j":
		a.Menu = (a.Menu + 1) % len(routes)
	case "up", "k":
		a.Menu = (a.Menu - 1 + len(routes)) % len(routes)
	case "enter":
		a.Path = routes[a.Menu]
		a.Menu = 0
	default:
		if len(s) == 1 && s[0] >= '1' && int(s[0]-'1') < len(routes) {
			a.Path = routes[s[0]-'1']
			a.Menu = 0
		}
	}
	return a
}

func (a App) accountCreateKey(s string) App {
	if a.Form["currency"] == "" {
		a.Form["currency"] = a.Config.Config.Currency
	}
	if a.Form["on-budget"] == "" {
		a.Form["on-budget"] = "true"
	}
	next, submit := a.accountFormKey(s, nil)
	if !submit {
		return next
	}
	name := strings.TrimSpace(next.Form["name"])
	currency := strings.TrimSpace(next.Form["currency"])
	onBudget := parseBoolDefault(next.Form["on-budget"], true)
	acct, entry, err := next.Svc.Accounts.Create(next.ctx, name, currency, onBudget, next.Form["notes"])
	if err != nil {
		next.Error = err.Error()
		return next
	}
	next.History = append(next.History, entry)
	next.SelectedAccount = acct.Name
	next.Form = map[string]string{}
	next.Field = 0
	next.Error = ""
	next.Path = "/accounts/list/"
	return next
}

func (a App) accountListKey(s string, includeHidden bool) App {
	switch s {
	case "up":
		rows, err := a.accountListRows(includeHidden)
		if err != nil {
			a.Error = err.Error()
			return a
		}
		if len(rows) > 0 {
			a.Menu = (a.Menu - 1 + len(rows)) % len(rows)
		}
		return a
	case "down":
		rows, err := a.accountListRows(includeHidden)
		if err != nil {
			a.Error = err.Error()
			return a
		}
		if len(rows) > 0 {
			a.Menu = (a.Menu + 1) % len(rows)
		}
		return a
	case "backspace":
		a.Form["filter"] = trimLastRune(a.Form["filter"])
		a.Menu = clampListCursor(a.Menu, a.accountListRowCount(includeHidden))
		return a
	case "enter":
		rows, err := a.accountListRows(includeHidden)
		if err != nil {
			a.Error = err.Error()
			return a
		}
		if len(rows) == 0 {
			return a
		}
		a.Menu = clampListCursor(a.Menu, len(rows))
		a.Path = "/accounts/" + rows[a.Menu].Name + "/"
		a.Menu = 0
		return a
	default:
		if isTextInputKey(s) {
			a.Form["filter"] += s
			a.Menu = 0
		}
		return a
	}
}

func (a App) accountListRowCount(includeHidden bool) int {
	rows, err := a.accountListRows(includeHidden)
	if err != nil {
		return 0
	}
	return len(rows)
}

func (a App) accountDetailKey(s, name string) App {
	acct, err := a.Svc.Accounts.GetByName(a.ctx, name)
	if err != nil {
		a.Error = err.Error()
		return a
	}
	action := a.actionIndex(s, 4)
	if acct.Hidden {
		switch action {
		case 0:
			updated, entry, err := a.Svc.Accounts.SetHidden(a.ctx, acct.ID, false)
			if err != nil {
				a.Error = err.Error()
				return a
			}
			a.History = append(a.History, entry)
			a.Path = "/accounts/" + updated.Name + "/"
		case 1:
			a.Path = "/accounts/" + name + "/balances/"
			a.Menu = 0
		case 2:
			a.Path = "/accounts/" + name + "/transactions/"
			a.Menu = 0
		case 3:
			a.Path = "/accounts/" + name + "/edit/"
			a.Form = map[string]string{"name": acct.Name, "currency": acct.Code, "on-budget": fmt.Sprintf("%t", acct.OnBudget), "notes": acct.Notes}
			a.Field = 0
			a.Menu = 0
		}
		return a
	}
	switch action {
	case 0:
		a.Path = "/accounts/" + name + "/balances/"
		a.Menu = 0
	case 1:
		a.Path = "/accounts/" + name + "/transactions/"
		a.Menu = 0
	case 2:
		a.Path = "/accounts/" + name + "/edit/"
		a.Form = map[string]string{"name": acct.Name, "currency": acct.Code, "on-budget": fmt.Sprintf("%t", acct.OnBudget), "notes": acct.Notes}
		a.Field = 0
		a.Menu = 0
	case 3:
		updated, entry, err := a.Svc.Accounts.SetHidden(a.ctx, acct.ID, !acct.Hidden)
		if err != nil {
			a.Error = err.Error()
			return a
		}
		a.History = append(a.History, entry)
		a.Path = "/accounts/" + updated.Name + "/"
	}
	return a
}

func (a App) accountEditKey(s, name string) App {
	acct, err := a.Svc.Accounts.GetByName(a.ctx, name)
	if err != nil {
		a.Error = err.Error()
		return a
	}
	locked := map[string]bool{}
	if has, err := a.Svc.Accounts.Accounts.HasBalances(a.ctx, acct.ID); err == nil && has {
		locked["currency"] = true
	}
	next, submit := a.accountFormKey(s, locked)
	if !submit {
		return next
	}
	updated, entry, err := next.Svc.Accounts.Update(next.ctx, acct.ID, strings.TrimSpace(next.Form["name"]), strings.TrimSpace(next.Form["currency"]), parseBoolDefault(next.Form["on-budget"], acct.OnBudget), acct.Hidden, next.Form["notes"])
	if err != nil {
		next.Error = err.Error()
		return next
	}
	next.History = append(next.History, entry)
	next.Form = map[string]string{}
	next.Field = 0
	next.Error = ""
	next.Path = "/accounts/" + updated.Name + "/"
	return next
}

func (a App) balanceAddKey(s, name string) App {
	acct, err := a.Svc.Accounts.GetByName(a.ctx, name)
	if err != nil {
		a.Error = err.Error()
		return a
	}
	if s == "enter" {
		_, entry, err := a.Svc.Balances.Add(a.ctx, acct.ID, a.Form["date"], a.Form["balance"], a.Form["notes"])
		if err != nil {
			a.Error = err.Error()
			return a
		}
		a.History = append(a.History, entry)
		a.Form = map[string]string{}
		a.Field = 0
		a.Error = ""
		a.Path = "/accounts/" + name + "/balances/"
		return a
	}
	return a.formKey(s, []string{"date", "balance", "notes"})
}

func (a App) balanceListKey(s, name string) App {
	acct, err := a.Svc.Accounts.GetByName(a.ctx, name)
	if err != nil {
		a.Error = err.Error()
		return a
	}
	rows, err := a.Svc.Balances.List(a.ctx, acct.ID)
	if err != nil {
		a.Error = err.Error()
		return a
	}
	var routes []string
	for _, row := range rows {
		routes = append(routes, "/accounts/"+name+"/balances/"+row.Date+"/")
	}
	routes = append(routes, "/accounts/"+name+"/balances/add/")
	a = a.menuKey(s, routes)
	if a.Path == "/accounts/"+name+"/balances/add/" {
		a.Form = map[string]string{"date": Today()}
		a.Field = 0
	}
	return a
}

func (a App) balanceDetailKey(s, name, date string) App {
	acct, err := a.Svc.Accounts.GetByName(a.ctx, name)
	if err != nil {
		a.Error = err.Error()
		return a
	}
	bal, err := a.Svc.Balances.Balances.GetByAccountDate(a.ctx, acct.ID, date)
	if err != nil {
		a.Error = err.Error()
		return a
	}
	action := a.actionIndex(s, 2)
	switch action {
	case 0:
		a.Path = "/accounts/" + name + "/balances/" + date + "/edit/"
		a.Form = map[string]string{"date": bal.Date, "balance": rawAmount(bal.Amount.Amount, bal.Amount.Scale), "notes": bal.Notes}
		a.Field = 0
		a.Menu = 0
	case 1:
		entry, err := a.Svc.Balances.Delete(a.ctx, bal.ID)
		if err != nil {
			a.Error = err.Error()
			return a
		}
		a.History = append(a.History, entry)
		a.Path = "/accounts/" + name + "/balances/"
		a.Error = ""
	}
	return a
}

func (a *App) actionIndex(s string, count int) int {
	switch s {
	case "down", "j":
		a.Menu = (a.Menu + 1) % count
		return -1
	case "up", "k":
		a.Menu = (a.Menu - 1 + count) % count
		return -1
	case "enter":
		return a.Menu
	default:
		if len(s) == 1 && s[0] >= '1' && int(s[0]-'1') < count {
			return int(s[0] - '1')
		}
	}
	return -1
}

func (a App) accountFormKey(s string, locked map[string]bool) (App, bool) {
	fields := []string{"name", "currency", "on-budget", "notes"}
	if a.Field == 1 && locked != nil && locked["currency"] {
		switch s {
		case "enter", "tab", "down":
			a.Field = 2
		case "shift+tab", "up":
			a.Field = 0
		}
		return a, false
	}
	if a.Field == 1 {
		return a.currencyFieldKey(s, fields)
	}
	if a.Field == 2 {
		return a.selectFieldKey(s, "on-budget", []string{"true", "false"}, fields)
	}
	if s == "enter" {
		if a.Field >= len(fields) {
			return a, true
		}
		a.Field++
		return a, false
	}
	return a.formKey(s, fields), false
}

func (a App) currencyFieldKey(s string, fields []string) (App, bool) {
	const filterField = "_currency_filter"
	const cursorField = "_currency_cursor"
	const pageField = "_currency_page"
	options := a.currencyOptions()
	if a.Form["currency"] == "" {
		a.Form["currency"] = a.Config.Config.Currency
	}
	filtered := filterOptions(options, a.Form[filterField])
	cursor := clampCurrencyCursor(parseFormInt(a.Form[cursorField]), len(filtered))
	a.Form[cursorField] = strconv.Itoa(cursor)
	a.Form[pageField] = strconv.Itoa(currencyPageForCursor(cursor))
	clearTransient := func() {
		delete(a.Form, filterField)
		delete(a.Form, cursorField)
		delete(a.Form, pageField)
	}
	resetCursor := func() {
		a.Form[cursorField] = "0"
		a.Form[pageField] = "0"
	}
	switch s {
	case "down":
		if len(filtered) == 0 {
			return a, false
		}
		cursor = (cursor + 1) % len(filtered)
		a.Form[cursorField] = strconv.Itoa(cursor)
		a.Form[pageField] = strconv.Itoa(currencyPageForCursor(cursor))
	case "up":
		if len(filtered) == 0 {
			return a, false
		}
		cursor = (cursor - 1 + len(filtered)) % len(filtered)
		a.Form[cursorField] = strconv.Itoa(cursor)
		a.Form[pageField] = strconv.Itoa(currencyPageForCursor(cursor))
	case "right":
		if len(filtered) == 0 {
			return a, false
		}
		page := min(parseFormInt(a.Form[pageField])+1, currencyPageCount(len(filtered))-1)
		cursor = min(page*currencyPageSize, len(filtered)-1)
		a.Form[cursorField] = strconv.Itoa(cursor)
		a.Form[pageField] = strconv.Itoa(page)
	case "left":
		if len(filtered) == 0 {
			return a, false
		}
		page := max(parseFormInt(a.Form[pageField])-1, 0)
		cursor = min(page*currencyPageSize, len(filtered)-1)
		a.Form[cursorField] = strconv.Itoa(cursor)
		a.Form[pageField] = strconv.Itoa(page)
	case "backspace":
		a.Form[filterField] = trimLastRune(a.Form[filterField])
		resetCursor()
	case "tab":
		clearTransient()
		a.Field = min(a.Field+1, len(fields))
	case "enter":
		if len(filtered) == 0 {
			return a, false
		}
		a.Form["currency"] = filtered[cursor]
		clearTransient()
		a.Field = min(a.Field+1, len(fields))
	case "shift+tab":
		clearTransient()
		a.Field = max(a.Field-1, 0)
	default:
		if strings.HasPrefix(s, "set currency=") {
			a.Form["currency"] = sanitizeCurrencyCode(strings.TrimPrefix(s, "set currency="))
			clearTransient()
			return a, false
		}
		if isTextInputKey(s) {
			a.Form[filterField] += sanitizeCurrencyCode(s)
			resetCursor()
		}
	}
	return a, false
}

func (a App) selectFieldKey(s, field string, options []string, fields []string) (App, bool) {
	if len(options) == 0 {
		return a, false
	}
	if a.Form[field] == "" {
		a.Form[field] = options[0]
	}
	idx := indexOf(options, a.Form[field])
	if idx < 0 {
		idx = 0
		a.Form[field] = options[idx]
	}
	switch s {
	case "down", "j":
		idx = (idx + 1) % len(options)
		a.Form[field] = options[idx]
	case "up", "k":
		idx = (idx - 1 + len(options)) % len(options)
		a.Form[field] = options[idx]
	case "tab":
		a.Field = min(a.Field+1, len(fields))
	case "shift+tab":
		a.Field = max(a.Field-1, 0)
	case "enter":
		a.Field = min(a.Field+1, len(fields))
	}
	return a, false
}

func (a App) currencyOptions() []string {
	currencies, err := a.Svc.Accounts.Currency.List(a.ctx)
	if err != nil {
		return []string{a.Config.Config.Currency}
	}
	var out []string
	for _, cur := range currencies {
		if cur.Code != a.Config.Config.Currency {
			out = append(out, cur.Code)
		}
	}
	sort.Strings(out)
	out = append([]string{a.Config.Config.Currency}, out...)
	if len(out) == 0 {
		return []string{a.Config.Config.Currency}
	}
	return out
}

func (a App) balanceEditKey(s, name, date string) App {
	acct, err := a.Svc.Accounts.GetByName(a.ctx, name)
	if err != nil {
		a.Error = err.Error()
		return a
	}
	bal, err := a.Svc.Balances.Balances.GetByAccountDate(a.ctx, acct.ID, date)
	if err != nil {
		a.Error = err.Error()
		return a
	}
	if s == "enter" {
		_, entry, err := a.Svc.Balances.Update(a.ctx, bal.ID, a.Form["date"], a.Form["balance"], a.Form["notes"])
		if err != nil {
			a.Error = err.Error()
			return a
		}
		a.History = append(a.History, entry)
		a.Form = map[string]string{}
		a.Field = 0
		a.Error = ""
		a.Path = "/accounts/" + name + "/balances/"
		return a
	}
	return a.formKey(s, []string{"date", "balance", "notes"})
}

func (a App) formKey(s string, fields []string) App {
	if strings.HasPrefix(s, "set ") {
		parts := strings.SplitN(strings.TrimPrefix(s, "set "), "=", 2)
		if len(parts) == 2 {
			a.Form[parts[0]] = normalizeFieldInput(parts[0], "", parts[1])
			a.resetTextCursor(parts[0])
		}
		return a
	}
	if len(fields) == 0 {
		return a
	}
	fieldCount := len(fields) + 1
	switch s {
	case "tab", "down":
		a.clearCurrentTextCursor(fields)
		a.Field = (a.Field + 1) % fieldCount
	case "shift+tab", "up":
		a.clearCurrentTextCursor(fields)
		a.Field = (a.Field - 1 + fieldCount) % fieldCount
	case "left":
		if a.Field < len(fields) {
			a.moveTextCursor(fields[a.Field], -1)
		}
	case "right":
		if a.Field < len(fields) {
			a.moveTextCursor(fields[a.Field], 1)
		}
	case "backspace":
		if a.Field >= len(fields) {
			return a
		}
		field := fields[a.Field]
		if cursor := a.textCursor(field); cursor > 0 {
			runes := []rune(a.Form[field])
			a.Form[field] = string(append(runes[:cursor-1], runes[cursor:]...))
			a.setTextCursor(field, cursor-1)
		}
	default:
		if a.Field < len(fields) && (isTextInputKey(s) || (s == "?" && fields[a.Field] == "notes")) {
			field := fields[a.Field]
			current := a.Form[field]
			cursor := a.textCursor(field)
			runes := []rune(current)
			next := string(runes[:cursor]) + s + string(runes[cursor:])
			a.Form[field] = normalizeFieldValue(field, next)
			if field == "name" {
				a.resetTextCursor(field)
			} else {
				a.setTextCursor(field, cursor+len([]rune(s)))
			}
		}
	}
	return a
}

func (a App) undo() App {
	if len(a.History) == 0 {
		return a
	}
	entry := a.History[len(a.History)-1]
	if err := a.Svc.History.Undo(a.ctx, entry); err != nil {
		a.Error = err.Error()
		return a
	}
	a.History = a.History[:len(a.History)-1]
	a.Path = "/"
	a.Error = ""
	return a
}

func (a App) View() string {
	s := a.screen()
	return a.render(s)
}

func (a App) screen() screen {
	switch {
	case a.Path == "/":
		return a.dashboardScreen()
	case a.Path == "/accounts/":
		s := a.dashboardScreen()
		s.Path = "/accounts/"
		s.Actions = []string{"overview", "list", "hidden", "create"}
		return s
	case a.Path == "/accounts/list/":
		return screen{Path: "/accounts/list/", Body: a.accountList(false), Help: listHelp()}
	case a.Path == "/accounts/hidden/":
		return screen{Path: "/accounts/hidden/", Body: a.accountList(true), Help: listHelp()}
	case a.Path == "/accounts/create/":
		return screen{Path: "/accounts/create/", Body: a.accountFormView(nil), Help: a.accountFormHelp()}
	case a.Path == "/settings/":
		return screen{Path: "/settings/", Body: fmt.Sprintf("config   : %s\ncurrency : %s\n", a.Config.Path, a.Config.Config.Currency)}
	case a.Path == "/backup/":
		return screen{
			Path:    "/backup/",
			Body:    fmt.Sprintf("last backup : %s\n\nrestore     : close stuf, replace db.sqlite with backup renamed to db.sqlite, reopen stuf\n", a.LastBackup),
			Actions: []string{"create backup"},
		}
	case strings.Contains(a.Path, "transactions") || strings.Contains(a.Path, "budgets") || strings.Contains(a.Path, "owed") || strings.Contains(a.Path, "reports"):
		return screen{Path: a.Path, Body: "(TODO)\n"}
	default:
		if name, ok := accountDetailName(a.Path); ok {
			return a.accountDetailScreen(name)
		}
		if name, ok := balancesName(a.Path); ok {
			return screen{Path: a.Path, Body: a.balanceList(name)}
		}
		if _, ok := balanceAddName(a.Path); ok {
			return screen{Path: a.Path, Body: a.formView([]string{"date", "balance", "notes"}, nil), Help: a.formHelp([]string{"date", "balance", "notes"})}
		}
		if name, date, ok := balanceDetailName(a.Path); ok {
			return a.balanceDetailScreen(name, date)
		}
		if _, _, ok := balanceEditName(a.Path); ok {
			return screen{Path: a.Path, Body: a.formView([]string{"date", "balance", "notes"}, nil), Help: a.formHelp([]string{"date", "balance", "notes"})}
		}
		if _, ok := accountEditName(a.Path); ok {
			return a.accountEditScreen()
		}
		return screen{Path: a.Path}
	}
}

func (a App) render(s screen) string {
	var b strings.Builder

	// empty space after goose migrations running
	b.WriteString("\n")

	if len(a.History) > 0 {
		b.WriteString("history (ctrl-z to undo)\n")
		for _, h := range a.History {
			b.WriteString(h.Line() + "\n")
		}
		b.WriteString("\n")
	}
	if a.Error != "" {
		b.WriteString("error: " + a.Error + "\n\n")
	}
	if a.Help {
		return b.String() + "help\n" + strings.Join(a.helpLines(s), "\n") + "\n"
	}
	if a.ExitAsk {
		b.WriteString("exit app? no\n")
		if len(a.History) > 0 {
			b.WriteString("undo history will be cleared\n")
		}
		return b.String()
	}
	b.WriteString("# stuf\n\n")
	if s.Body != "" {
		b.WriteString(strings.TrimRight(s.Body, "\n") + "\n")
	}
	if s.Path != "" {
		if s.Body != "" {
			b.WriteString("\n")
		}
		b.WriteString(s.Path + "\n")
	}
	if len(s.Actions) > 0 {
		if s.Path != "" {
			b.WriteString("\n")
		} else if s.Body != "" {
			b.WriteString("\n")
		}
		b.WriteString(menuItems(s.Actions, a.Menu))
	}
	b.WriteString("\n---\n")
	b.WriteString(strings.Join(a.helpLines(s), "\n"))
	b.WriteString("\n")
	return b.String()
}

func (a App) helpLines(s screen) []string {
	if len(s.Help) > 0 {
		return s.Help
	}
	if len(s.Actions) > 0 {
		return []string{"up/down : navigate", "enter   : confirm", "esc     : back", "?       : help", "ctrl-z  : undo"}
	}
	return []string{"esc     : back", "?       : help", "ctrl-z  : undo"}
}

func (a App) formHelp(fields []string) []string {
	if a.Field < len(fields) && fields[a.Field] == "notes" {
		return []string{"tab     : next field", "enter   : confirm", "esc     : discard"}
	}
	return []string{"tab     : next field", "enter   : confirm", "esc     : discard", "?       : help"}
}

func listHelp() []string {
	return []string{"up/down : navigate", "enter   : confirm", "esc     : back", "?       : help", "ctrl-z  : undo"}
}

func (a App) accountFormHelp() []string {
	if a.Field == 1 {
		return []string{"type       : filter", "up/down    : move cursor", "left/right : next/prev page", "enter      : confirm", "tab        : navigate", "esc        : back", "?          : help"}
	}
	if a.Field == 2 {
		return []string{"up/down : move cursor", "enter   : confirm", "tab     : navigate", "esc     : back", "?       : help"}
	}
	if a.Field >= 4 {
		return []string{"shift-tab : navigate", "enter     : confirm", "esc       : back", "?         : help"}
	}
	return a.formHelp([]string{"name", "currency", "on-budget", "notes"})
}

func (a App) dashboardScreen() screen {
	d, err := a.Svc.Dashboard.Summary(a.ctx)
	if err != nil {
		return screen{Path: "/", Body: "error: " + err.Error() + "\n"}
	}
	cur := a.Config.Config.Currency
	warnings := ""
	if len(d.Warnings) > 0 {
		warnings = "\nwarning: " + strings.Join(d.Warnings, "; ") + "\n"
	}
	body := fmt.Sprintf(`total       : %s
budgeted    : %s

period      : %s

growth
on-budget  : %s
total      : %s

you owe ppl : %s
ppl owe you : %s
%s`, d.Total.Format(cur), zero(cur), d.Period, d.OnBudgetGrow.Format(cur), d.TotalGrow.Format(cur), zero(cur), zero(cur), warnings)
	return screen{
		Path:    "/",
		Body:    body,
		Actions: []string{"accounts", "transactions (TODO)", "budgets (TODO)", "owed (TODO)", "reports (TODO)", "settings", "backup"},
	}
}

func (a App) accountList(includeHidden bool) string {
	visible, err := a.accountListRows(includeHidden)
	if err != nil {
		return "error: " + err.Error() + "\n"
	}
	filter := a.Form["filter"]
	var lines []string
	lines = append(lines, "> filter : "+placeholder(filter, "(type anything...)"), "")
	if len(visible) == 0 {
		lines = append(lines, "  (no results)")
		return strings.Join(lines, "\n") + "\n"
	}
	layout := accountListTableLayoutFor(visible, a.Config.Config.Currency)
	if includeHidden {
		lines = append(lines, layout.headerLine())
		for i, row := range visible {
			prefix := "  "
			if i == a.Menu {
				prefix = "> "
			}
			lines = append(lines, layout.rowLine(prefix, row.Name, row.Balance, row.Notes))
		}
		return strings.Join(lines, "\n") + "\n"
	}
	lines = appendAccountSection(lines, "on-budget accounts", visible, true, a.Menu, a.Config.Config.Currency, layout)
	lines = append(lines, "")
	lines = appendAccountSection(lines, "off-budget accounts", visible, false, a.Menu, a.Config.Config.Currency, layout)
	return strings.Join(lines, "\n") + "\n"
}

type accountListRow struct {
	Name     string
	Balance  string
	Amount   money.Money
	Notes    string
	OnBudget bool
	AsOf     string
}

type accountListTableLayout struct {
	NameWidth    int
	BalanceWidth int
}

func accountListTableLayoutFor(rows []accountListRow, appCurrency string) accountListTableLayout {
	layout := accountListTableLayout{NameWidth: len("name"), BalanceWidth: len("balance")}
	if len(rows) == 0 {
		return layout
	}
	for _, onBudget := range []bool{true, false} {
		if total, ok := accountSectionTotal(rows, onBudget); ok {
			layout.NameWidth = max(layout.NameWidth, len("TOTAL"))
			layout.BalanceWidth = max(layout.BalanceWidth, len(total.Format(appCurrency)))
		}
	}
	for _, row := range rows {
		layout.NameWidth = max(layout.NameWidth, len(row.Name))
		layout.BalanceWidth = max(layout.BalanceWidth, len(row.Balance))
	}
	return layout
}

func accountSectionTotal(rows []accountListRow, onBudget bool) (money.Money, bool) {
	if len(rows) == 0 {
		return money.Money{}, false
	}
	total := money.Money{Scale: rows[0].Amount.Scale}
	found := false
	for _, row := range rows {
		if row.OnBudget != onBudget {
			continue
		}
		found = true
		next, err := total.Add(row.Amount)
		if err == nil {
			total = next
		}
	}
	return total, found
}

func (l accountListTableLayout) headerLine() string {
	return fmt.Sprintf("  %-*s | %-*s | notes", l.NameWidth, "name", l.BalanceWidth, "balance")
}

func (l accountListTableLayout) totalLine(total string) string {
	return fmt.Sprintf("  %-*s | %-*s |", l.NameWidth, "TOTAL", l.BalanceWidth, total)
}

func (l accountListTableLayout) rowLine(prefix, name, balance, notes string) string {
	return fmt.Sprintf("%s%-*s | %-*s | %s", prefix, l.NameWidth, name, l.BalanceWidth, balance, notes)
}

func (a App) accountListRows(includeHidden bool) ([]accountListRow, error) {
	accounts, err := a.Svc.Accounts.List(a.ctx, includeHidden)
	if err != nil {
		return nil, err
	}
	appCur, err := a.Svc.Accounts.Currency.GetByCode(a.ctx, a.Config.Config.Currency)
	if err != nil {
		return nil, err
	}
	filter := a.Form["filter"]
	var visible []accountListRow
	for _, acct := range accounts {
		if includeHidden && !acct.Hidden {
			continue
		}
		if filter != "" && !strings.Contains(acct.Name, filter) && !strings.Contains(acct.Notes, filter) {
			continue
		}
		native := money.Money{Scale: acct.Scale}
		asOf := "(no balance entered yet)"
		if bal, ok, err := a.Svc.Accounts.CurrentBalance(a.ctx, acct.ID); err != nil {
			return nil, err
		} else if ok {
			native = bal.Amount
			asOf = bal.Date
		}
		appAmount, balance, err := a.accountListBalance(acct.Code, native, appCur)
		if err != nil {
			return nil, err
		}
		visible = append(visible, accountListRow{
			Name:     acct.Name,
			Balance:  balance,
			Amount:   appAmount,
			Notes:    acct.Notes,
			OnBudget: acct.OnBudget,
			AsOf:     asOf,
		})
	}
	return visible, nil
}

func (a App) accountListBalance(code string, native money.Money, appCur repo.Currency) (money.Money, string, error) {
	if code == appCur.Code {
		appAmount, err := native.ConvertToScale(appCur.Scale)
		if err != nil {
			return money.Money{}, "", err
		}
		return appAmount, appAmount.Format(appCur.Code), nil
	}
	cur, err := a.Svc.Accounts.Currency.GetByCode(a.ctx, code)
	if err != nil {
		return money.Money{}, "", err
	}
	appAmount, err := money.Convert(native, cur.RateToUSD, appCur.RateToUSD, appCur.Scale)
	if err != nil {
		return money.Money{}, "", err
	}
	return appAmount, fmt.Sprintf("%s (%s)", appAmount.Format(appCur.Code), native.Format(code)), nil
}

func appendAccountSection(lines []string, title string, rows []accountListRow, onBudget bool, selected int, appCurrency string, layout accountListTableLayout) []string {
	total, ok := accountSectionTotal(rows, onBudget)
	if !ok {
		return lines
	}
	lines = append(lines, "  "+title)
	lines = append(lines, layout.headerLine())
	lines = append(lines, layout.totalLine(total.Format(appCurrency)))
	for i, row := range rows {
		if row.OnBudget != onBudget {
			continue
		}
		prefix := "  "
		if i == selected {
			prefix = "> "
		}
		lines = append(lines, layout.rowLine(prefix, row.Name, row.Balance, row.Notes))
	}
	return lines
}

func (a App) accountDetailScreen(name string) screen {
	acct, err := a.Svc.Accounts.GetByName(a.ctx, name)
	if err != nil {
		return screen{Path: a.Path, Body: "error: " + err.Error() + "\n"}
	}
	bal, ok, _ := a.Svc.Accounts.CurrentBalance(a.ctx, acct.ID)
	amount, asOf := zero(acct.Code), "(no balance entered yet)"
	if ok {
		amount, asOf = bal.Amount.Format(acct.Code), bal.Date
	}
	hidden := ""
	actions := []string{"balances", "transactions (TODO)", "edit account", "hide account"}
	if acct.Hidden {
		hidden = "hidden    : true\n"
		actions = []string{"show account", "balances", "transactions (TODO)", "edit account"}
	}
	return screen{
		Path:    "/accounts/" + name + "/",
		Body:    fmt.Sprintf("name      : %s\nbalance   : %s\nas of     : %s\non-budget : %t\n%snotes     : %s\n", acct.Name, amount, asOf, acct.OnBudget, hidden, acct.Notes),
		Actions: actions,
	}
}

func (a App) accountEditScreen() screen {
	name, _ := accountEditName(a.Path)
	acct, err := a.Svc.Accounts.GetByName(a.ctx, name)
	if err != nil {
		return screen{Path: a.Path, Body: "error: " + err.Error() + "\n"}
	}
	locked := map[string]string{}
	if has, err := a.Svc.Accounts.Accounts.HasBalances(a.ctx, acct.ID); err == nil && has {
		locked["currency"] = acct.Code + " (locked because balances exist)"
	}
	return screen{Path: a.Path, Body: a.accountFormView(locked), Help: a.accountFormHelp()}
}

func (a App) accountFormView(locked map[string]string) string {
	return a.formViewWithOptions([]string{"name", "currency", "on-budget", "notes"}, locked, map[string][]string{
		"currency":  a.currencyOptions(),
		"on-budget": {"true", "false"},
	})
}

func (a App) balanceList(name string) string {
	acct, err := a.Svc.Accounts.GetByName(a.ctx, name)
	if err != nil {
		return "error: " + err.Error() + "\n"
	}
	rows, err := a.Svc.Balances.List(a.ctx, acct.ID)
	if err != nil {
		return "error: " + err.Error() + "\n"
	}
	bal, ok, _ := a.Svc.Accounts.CurrentBalance(a.ctx, acct.ID)
	amount := zero(acct.Code)
	asOf := "(no balance entered yet)"
	if ok {
		amount = bal.Amount.Format(acct.Code)
		asOf = bal.Date
	}
	lines := []string{
		fmt.Sprintf("name        : %s", acct.Name),
		fmt.Sprintf("balance     : %s", amount),
		fmt.Sprintf("as of       : %s", asOf),
		"",
		"  date       | balance      | notes",
	}
	if len(rows) == 0 {
		lines = append(lines, "  (no balances yet)", "")
		lines = append(lines, menuItems([]string{"add balance"}, a.Menu))
		return strings.Join(lines, "\n") + "\n"
	}
	for i, row := range rows {
		prefix := "  "
		if a.Menu == i {
			prefix = "> "
		}
		lines = append(lines, fmt.Sprintf("%s%s | %-12s | %s", prefix, row.Date, row.Amount.Format(acct.Code), row.Notes))
	}
	selectedAction := -1
	if a.Menu == len(rows) {
		selectedAction = 0
	}
	lines = append(lines, "", menuItems([]string{"add balance"}, selectedAction))
	return strings.Join(lines, "\n") + "\n"
}

func (a App) balanceDetailScreen(name, date string) screen {
	acct, err := a.Svc.Accounts.GetByName(a.ctx, name)
	if err != nil {
		return screen{Path: a.Path, Body: "error: " + err.Error() + "\n"}
	}
	bal, err := a.Svc.Balances.Balances.GetByAccountDate(a.ctx, acct.ID, date)
	if err != nil {
		return screen{Path: a.Path, Body: "error: " + err.Error() + "\n"}
	}
	return screen{
		Path:    "/accounts/" + name + "/balances/" + date + "/",
		Body:    fmt.Sprintf("account : %s\ndate    : %s\nbalance : %s\nnotes   : %s\n", name, date, bal.Amount.Format(acct.Code), bal.Notes),
		Actions: []string{"edit balance", "delete balance"},
	}
}

func (a App) formView(fields []string, locked map[string]string) string {
	return a.formViewWithOptions(fields, locked, nil)
}

func (a App) formViewWithOptions(fields []string, locked map[string]string, options map[string][]string) string {
	var lines []string
	for i, field := range fields {
		if i > 0 {
			lines = append(lines, "")
		}
		prefix := "  "
		if i == a.Field {
			prefix = "> "
		}
		value := a.Form[field]
		if value == "" && field == "currency" {
			value = a.Config.Config.Currency
		}
		if value == "" && field == "on-budget" {
			value = "true"
		}
		if locked != nil && locked[field] != "" {
			value = locked[field]
		}
		renderedValue := placeholder(value, placeholderFor(field))
		if i == a.Field && isFormTextField(field, options) && (locked == nil || locked[field] == "") {
			renderedValue = renderCaret(value, placeholderFor(field), a.textCursor(field))
		}
		lines = append(lines, fmt.Sprintf("%s%d) %-9s: %s", prefix, i+1, field, renderedValue))
		if i == a.Field && options != nil && len(options[field]) > 0 && (locked == nil || locked[field] == "") {
			selected := value
			fieldOptions := options[field]
			if field == "currency" {
				lines = append(lines, a.currencySelectLines(fieldOptions)...)
				continue
			} else {
				lines = append(lines, "")
			}
			for _, option := range fieldOptions {
				optionPrefix := "       "
				if option == selected {
					optionPrefix = "     > "
				}
				lines = append(lines, optionPrefix+option)
			}
		}
	}
	confirmPrefix := "  "
	if a.Field == len(fields) {
		confirmPrefix = "> "
	}
	lines = append(lines, "", confirmPrefix+"[confirm]")
	return strings.Join(lines, "\n") + "\n"
}

func (a App) currencySelectLines(options []string) []string {
	filter := a.Form["_currency_filter"]
	filtered := filterOptions(options, filter)
	cursor := clampCurrencyCursor(parseFormInt(a.Form["_currency_cursor"]), len(filtered))
	page := clampCurrencyPage(parseFormInt(a.Form["_currency_page"]), cursor, len(filtered))
	start := page * currencyPageSize
	end := min(start+currencyPageSize, len(filtered))
	lines := []string{"", fmt.Sprintf("   > filter  : %s", placeholder(filter, "(type anything...)")), ""}
	if len(filtered) == 0 {
		lines = append(lines, "     (no matching currencies)", "", "     [00/00]")
		return lines
	}
	for i, option := range filtered[start:end] {
		optionPrefix := "       "
		if start+i == cursor {
			optionPrefix = "     > "
		}
		lines = append(lines, optionPrefix+option)
	}
	lines = append(lines, "", fmt.Sprintf("     [%02d/%02d]", cursor+1, len(filtered)))
	return lines
}

func menuItems(items []string, selected int) string {
	var b strings.Builder
	for i, item := range items {
		prefix := "  "
		if i == selected {
			prefix = "> "
		}
		b.WriteString(fmt.Sprintf("%s%d) %s\n", prefix, i+1, item))
	}
	return b.String()
}

func zero(code string) string { return code + " 0.00" }

func placeholder(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func placeholderFor(field string) string {
	switch field {
	case "name", "notes":
		return "(type anything...)"
	case "balance":
		return "(type amount...)"
	default:
		return ""
	}
}

func normalizeFieldInput(field, current, input string) string {
	return normalizeFieldValue(field, current+input)
}

func normalizeFieldValue(field, value string) string {
	if field != "name" {
		return value
	}
	return sanitizeSlug(value)
}

func isTextInputKey(input string) bool {
	switch input {
	case "", "left", "right", "up", "down", "enter", "esc", "tab", "shift+tab", "backspace", "ctrl+c", "ctrl+z", "?":
		return false
	default:
		return true
	}
}

func filterOptions(options []string, filter string) []string {
	filter = sanitizeCurrencyCode(filter)
	if filter == "" {
		return options
	}
	var out []string
	for _, option := range options {
		if strings.Contains(option, filter) {
			out = append(out, option)
		}
	}
	return out
}

func sanitizeCurrencyCode(input string) string {
	var b strings.Builder
	for _, r := range input {
		switch {
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= 'a' && r <= 'z':
			b.WriteRune(r + ('A' - 'a'))
		}
	}
	return b.String()
}

func parseFormInt(value string) int {
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return n
}

func clampCurrencyCursor(cursor, count int) int {
	if count <= 0 || cursor < 0 {
		return 0
	}
	if cursor >= count {
		return count - 1
	}
	return cursor
}

func clampListCursor(cursor, count int) int {
	if count <= 0 || cursor < 0 {
		return 0
	}
	if cursor >= count {
		return count - 1
	}
	return cursor
}

func currencyPageForCursor(cursor int) int {
	if cursor < 0 {
		return 0
	}
	return cursor / currencyPageSize
}

func currencyPageCount(count int) int {
	if count <= 0 {
		return 1
	}
	return (count + currencyPageSize - 1) / currencyPageSize
}

func clampCurrencyPage(page, cursor, count int) int {
	if count <= 0 {
		return 0
	}
	page = max(page, 0)
	page = min(page, currencyPageCount(count)-1)
	cursorPage := currencyPageForCursor(cursor)
	if cursorPage != page {
		return cursorPage
	}
	return page
}

func trimLastRune(s string) string {
	if s == "" {
		return ""
	}
	runes := []rune(s)
	return string(runes[:len(runes)-1])
}

func cursorKey(field string) string { return "_cursor:" + field }

func (a App) textCursor(field string) int {
	size := len([]rune(a.Form[field]))
	raw, ok := a.Form[cursorKey(field)]
	if !ok {
		return size
	}
	cursor := parseFormInt(raw)
	if cursor < 0 {
		return 0
	}
	if cursor > size {
		return size
	}
	return cursor
}

func (a App) setTextCursor(field string, cursor int) {
	size := len([]rune(a.Form[field]))
	cursor = max(0, min(cursor, size))
	a.Form[cursorKey(field)] = strconv.Itoa(cursor)
}

func (a App) resetTextCursor(field string) {
	a.setTextCursor(field, len([]rune(a.Form[field])))
}

func (a App) moveTextCursor(field string, delta int) {
	a.setTextCursor(field, a.textCursor(field)+delta)
}

func (a App) clearCurrentTextCursor(fields []string) {
	if a.Field < len(fields) {
		delete(a.Form, cursorKey(fields[a.Field]))
	}
}

func isFormTextField(field string, options map[string][]string) bool {
	return options == nil || len(options[field]) == 0
}

func renderCaret(value, fallback string, cursor int) string {
	if value == "" {
		return "|"
	}
	runes := []rune(value)
	cursor = max(0, min(cursor, len(runes)))
	return string(runes[:cursor]) + "|" + string(runes[cursor:])
}

func sanitizeSlug(input string) string {
	var b strings.Builder
	lastHyphen := false
	for _, r := range input {
		switch {
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + ('a' - 'A'))
			lastHyphen = false
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			lastHyphen = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			lastHyphen = false
		case r == '-' || r == ' ' || r == '\t' || r == '\n' || r == '\r':
			if b.Len() > 0 && !lastHyphen {
				b.WriteByte('-')
				lastHyphen = true
			}
		}
	}
	return b.String()
}

func parseBoolDefault(value string, fallback bool) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "yes", "1", "on":
		return true
	case "false", "no", "0", "off":
		return false
	default:
		return fallback
	}
}

func indexOf(values []string, needle string) int {
	for i, value := range values {
		if value == needle {
			return i
		}
	}
	return -1
}

func rawAmount(amount int64, scale int) string {
	sign := ""
	if amount < 0 {
		sign = "-"
		amount = -amount
	}
	if scale == 0 {
		return fmt.Sprintf("%s%d", sign, amount)
	}
	div := int64(1)
	for range scale {
		div *= 10
	}
	return fmt.Sprintf("%s%d.%0*d", sign, amount/div, scale, amount%div)
}

func parentPath(path string) string {
	path = strings.TrimSuffix(path, "/")
	if path == "" {
		return "/"
	}
	i := strings.LastIndex(path, "/")
	if i <= 0 {
		return "/"
	}
	return path[:i+1]
}

func Today() string { return time.Now().Format("2006-01-02") }

func accountDetailName(path string) (string, bool) {
	if !strings.HasPrefix(path, "/accounts/") || !strings.HasSuffix(path, "/") {
		return "", false
	}
	rest := strings.TrimPrefix(path, "/accounts/")
	parts := strings.Split(strings.TrimSuffix(rest, "/"), "/")
	if len(parts) == 1 && parts[0] != "" && parts[0] != "list" && parts[0] != "hidden" && parts[0] != "create" {
		return parts[0], true
	}
	return "", false
}

func accountEditName(path string) (string, bool) {
	if !strings.HasPrefix(path, "/accounts/") || !strings.HasSuffix(path, "/edit/") {
		return "", false
	}
	name := strings.TrimSuffix(strings.TrimPrefix(path, "/accounts/"), "/edit/")
	if strings.Contains(name, "/") {
		return "", false
	}
	return name, name != ""
}

func accountEditPath(path string) bool {
	_, ok := accountEditName(path)
	return ok
}

func balancesName(path string) (string, bool) {
	if !strings.HasPrefix(path, "/accounts/") || !strings.HasSuffix(path, "/balances/") {
		return "", false
	}
	name := strings.TrimSuffix(strings.TrimPrefix(path, "/accounts/"), "/balances/")
	return name, name != ""
}

func balanceAddName(path string) (string, bool) {
	if !strings.HasPrefix(path, "/accounts/") || !strings.HasSuffix(path, "/balances/add/") {
		return "", false
	}
	name := strings.TrimSuffix(strings.TrimPrefix(path, "/accounts/"), "/balances/add/")
	return name, name != ""
}

func balanceAddPath(path string) bool {
	_, ok := balanceAddName(path)
	return ok
}

func balanceDetailName(path string) (string, string, bool) {
	if !strings.HasPrefix(path, "/accounts/") || !strings.HasSuffix(path, "/") {
		return "", "", false
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(path, "/accounts/"), "/"), "/")
	if len(parts) == 3 && parts[1] == "balances" && parts[2] != "add" {
		return parts[0], parts[2], true
	}
	return "", "", false
}

func balanceEditName(path string) (string, string, bool) {
	if !strings.HasPrefix(path, "/accounts/") || !strings.HasSuffix(path, "/edit/") {
		return "", "", false
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(path, "/accounts/"), "/"), "/")
	if len(parts) == 4 && parts[1] == "balances" && parts[3] == "edit" {
		return parts[0], parts[2], true
	}
	return "", "", false
}

func balanceEditPath(path string) bool {
	_, _, ok := balanceEditName(path)
	return ok
}
