package model

import (
	"context"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"strings"
	"stuf/internal/config"
	"stuf/internal/service"
)

type Services struct {
	Accounts  service.AccountService
	Balances  service.BalanceService
	Currency  service.CurrencyService
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
	Nav             NavigationStack
	History         []service.SessionEntry
	Error           string
	Help            bool
	ExitAsk         bool
	LastBackup      string
	Form            map[string]string
	Field           int
	SelectedAccount string
	AccountVisible  accountVisibilityMode
	ListReturn      listReturnState
}

type screen struct {
	Path    string
	Context string
	Body    string
	Options string
	Actions []string
	Help    []string
}

type listReturnState struct {
	Path           string
	Item           string
	Filter         string
	AccountVisible accountVisibilityMode
}

const currencyPageSize = 8

func New(ctx context.Context, svc Services, cfg config.Loaded) App {
	return App{ctx: ctx, Svc: svc, Config: cfg, Path: "/", Form: map[string]string{}, Nav: NewNavigationStack()}
}

func (a App) Init() tea.Cmd { return nil }

func (a App) notesFocused() bool {
	switch {
	case a.Path == routeAccountCreate:
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
		return a.exitConfirmKey(s)
	}
	if s == "esc" {
		if a.Path == routeRoot {
			a.ExitAsk = true
			a = a.navSetMenu(0)
			return a, nil
		}
		a.Error = ""
		a = a.goBack()
		return a, nil
	}
	switch a.Path {
	case routeRoot:
		a = a.menuKey(s, []string{routeAccountList, "/transactions/", "/budgets/", "/owed/", "/reports/", routeSettings, routeBackup})
	case routeAccountList:
		a = a.accountListKey(s)
	case routeAccountCreate:
		a = a.accountCreateKey(s)
	case routeBackup:
		a = a.backupKey(s)
	case routeSettings:
		if isMenuBackKey(s) {
			a.Error = ""
			a = a.goBack()
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
		} else if name, ok := balanceListName(a.Path); ok {
			a = a.balanceListTableKey(s, name)
		} else if name, ok := accountEditName(a.Path); ok {
			a = a.accountEditKey(s, name)
		}
	}
	return a, nil
}

func exitConfirmActions() []string {
	return []string{"no", "yes"}
}

func (a App) backupKey(s string) App {
	if isMenuBackKey(s) {
		a.Error = ""
		return a.goBack()
	}
	if isMenuForwardKey(s) || s == "enter" || s == "1" {
		p, err := a.Svc.Backup(a.ctx)
		if err != nil {
			a.Error = "could not create backup: " + err.Error()
		} else {
			a.Error = ""
			a.LastBackup = p
		}
	}
	return a
}

func (a App) exitConfirmKey(s string) (App, tea.Cmd) {
	actions := exitConfirmActions()
	switch s {
	case "down", "j", "tab":
		a = a.navSetMenu((a.Menu + 1) % len(actions))
	case "up", "k", "shift+tab":
		a = a.navSetMenu((a.Menu - 1 + len(actions)) % len(actions))
	case "esc", "left", "h":
		a.ExitAsk = false
	case "enter", "right", "l":
		return a.exitConfirmSelect(a.Menu)
	default:
		if len(s) == 1 && s[0] >= '1' && int(s[0]-'1') < len(actions) {
			a = a.navSetMenu(int(s[0] - '1'))
			return a.exitConfirmSelect(a.Menu)
		}
	}
	return a, nil
}

func (a App) exitConfirmSelect(idx int) (App, tea.Cmd) {
	if idx == 1 {
		a.History = nil
		return a, tea.Quit
	}
	a.ExitAsk = false
	return a, nil
}

func (a App) menuKey(s string, routes []string) App {
	if isMenuBackKey(s) {
		if a.Path == routeRoot {
			a.ExitAsk = true
			return a.navSetMenu(0)
		}
		a.Error = ""
		return a.goBack()
	}
	if isMenuForwardKey(s) {
		s = "enter"
	}
	switch s {
	case "down", "j", "tab":
		a = a.navSetMenu((a.Menu + 1) % len(routes))
	case "up", "k", "shift+tab":
		a = a.navSetMenu((a.Menu - 1 + len(routes)) % len(routes))
	case "enter":
		a = a.navSetMenu(a.Menu)
		next := routes[a.Menu]
		if next != a.Path {
			a = a.navPush(next, 0)
		}
	default:
		if len(s) == 1 && s[0] >= '1' && int(s[0]-'1') < len(routes) {
			a = a.navSetMenu(int(s[0] - '1'))
			next := routes[a.Menu]
			if next != a.Path {
				a = a.navPush(next, 0)
			}
		}
	}
	return a
}

func (a *App) actionIndex(s string, count int) int {
	if isMenuBackKey(s) {
		a.Error = ""
		*a = a.goBack()
		return -1
	}
	if isMenuForwardKey(s) {
		s = "enter"
	}
	switch s {
	case "down", "j", "tab":
		*a = a.navSetMenu((a.Menu + 1) % count)
		return -1
	case "up", "k", "shift+tab":
		*a = a.navSetMenu((a.Menu - 1 + count) % count)
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
	return a.navReset()
}

func (a App) View() string {
	s := a.screen()
	return a.render(s)
}

func (a App) screen() screen {
	switch {
	case a.Path == routeRoot:
		s := a.dashboardScreen()
		if a.ExitAsk {
			s.Body += "\nquit stuf?"
			if len(a.History) > 0 {
				s.Body += "\nundo history will be cleared"
			}
			s.Body += "\n"
			s.Path = ""
			s.Actions = exitConfirmActions()
		}
		return s
	case a.Path == routeAccountList:
		context, err := a.dashboardContext()
		if err != nil {
			return screen{Path: routeAccountList, Body: "error: " + err.Error() + "\n"}
		}
		body := strings.TrimRight(a.accountList(), "\n")
		return screen{Path: routeAccountList, Context: context, Body: body, Help: accountListHelp()}
	case a.Path == routeAccountCreate:
		return screen{Path: routeAccountCreate, Body: a.accountFormView(nil), Help: a.accountFormHelp()}
	case a.Path == routeSettings:
		return screen{Path: routeSettings, Body: fmt.Sprintf("config   : %s\ncurrency : %s\n", a.Config.Path, a.Config.Config.Currency)}
	case a.Path == routeBackup:
		return screen{
			Path:    routeBackup,
			Body:    fmt.Sprintf("last backup : %s\n\nrestore     : close stuf, replace db.sqlite with backup renamed to db.sqlite, reopen stuf\n", a.LastBackup),
			Actions: []string{"create backup"},
		}
	case strings.Contains(a.Path, "transactions") || strings.Contains(a.Path, "budgets") || strings.Contains(a.Path, "owed") || strings.Contains(a.Path, "reports"):
		return screen{Path: a.Path, Body: "(TODO)\n"}
	default:
		if name, ok := accountDetailName(a.Path); ok {
			return a.accountDetailScreen(name)
		}
		if name, ok := balanceListName(a.Path); ok {
			context, err := a.accountDashboardContext(name)
			if err != nil {
				return screen{Path: a.Path, Body: "error: " + err.Error() + "\n"}
			}
			return screen{
				Path:    a.Path,
				Context: context,
				Body:    a.balanceListBody(name),
				Help:    tableListHelp(),
			}
		}
		if name, ok := balanceAddName(a.Path); ok {
			return a.balanceAddScreen(name)
		}
		if name, date, ok := balanceDetailName(a.Path); ok {
			return a.balanceDetailScreen(name, date)
		}
		if name, date, ok := balanceEditName(a.Path); ok {
			return a.balanceEditScreen(name, date)
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
	b.WriteString("# stuf\n\n")
	if s.Context != "" {
		b.WriteString(strings.TrimRight(s.Context, "\n") + "\n")
	}
	if s.Path != "" {
		if s.Context != "" {
			b.WriteString("\n")
		}
		b.WriteString(s.Path + "\n")
	}
	if s.Body != "" {
		if s.Context != "" || s.Path != "" {
			b.WriteString("\n")
		}
		b.WriteString(strings.TrimRight(s.Body, "\n") + "\n")
	}
	if s.Options != "" {
		if s.Context != "" || s.Path != "" || s.Body != "" {
			b.WriteString("\n")
		}
		b.WriteString(strings.TrimRight(s.Options, "\n") + "\n")
	}
	if len(s.Actions) > 0 {
		if s.Context != "" || s.Path != "" || s.Body != "" || s.Options != "" {
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
		return []string{"up/down/j/k   : navigate", "tab/shift-tab : navigate", "left/h        : back", "right/l       : open", "enter         : confirm", "esc           : back", "?             : help", "ctrl-z        : undo"}
	}
	return []string{"left/h  : back", "esc     : back", "?       : help", "ctrl-z  : undo"}
}

func (a App) formHelp(fields []string) []string {
	if a.Field >= len(fields) {
		return []string{"shift-tab : previous field", "enter     : confirm", "ctrl+s    : submit", "esc       : discard", "?         : help"}
	}
	if a.Field < len(fields) && fields[a.Field] == "notes" {
		return []string{"tab     : next field", "enter   : next field", "ctrl+s  : submit", "esc     : discard"}
	}
	return []string{"tab     : next field", "enter   : next field", "ctrl+s  : submit", "esc     : discard", "?       : help"}
}

func accountListHelp() []string {
	return []string{"type          : filter", "h/l           : type in filter", "up/down       : navigate", "tab/shift-tab : navigate", "left/right    : back/open", "backspace     : edit filter", "enter         : confirm", "ctrl+n        : new", "ctrl+e        : edit", "ctrl+h        : hidden", "esc           : back", "?             : help", "ctrl-z        : undo"}
}

func tableListHelp() []string {
	return []string{"up/down/j/k   : navigate", "tab/shift-tab : navigate", "left/right    : back/open", "enter         : confirm", "ctrl+n        : new", "ctrl+e        : edit", "ctrl+d        : delete", "esc           : back", "?             : help", "ctrl-z        : undo"}
}

func (a App) accountFormHelp() []string {
	if a.Field == 1 {
		return []string{"type       : filter", "h/l        : type in filter", "up/down    : move cursor", "left/right : next/prev page", "enter      : confirm", "ctrl+s     : submit", "tab        : navigate", "esc        : back", "?          : help"}
	}
	if a.Field == 2 {
		return []string{"up/down : move cursor", "enter   : confirm", "ctrl+s  : submit", "tab     : navigate", "esc     : back", "?       : help"}
	}
	if a.Field >= 4 {
		return []string{"shift-tab : navigate", "enter     : confirm", "ctrl+s    : submit", "esc       : back", "?         : help"}
	}
	return a.formHelp([]string{"name", "currency", "on-budget", "notes"})
}
