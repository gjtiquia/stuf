package model

import (
	"context"
	"fmt"
	"strings"
	tea "github.com/charmbracelet/bubbletea"
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
}

type screen struct {
	Path    string
	Body    string
	Options string
	Actions []string
	Help    []string
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
		a = a.menuKey(s, []string{routeAccounts, "/transactions/", "/budgets/", "/owed/", "/reports/", routeSettings, routeBackup})
	case routeAccounts:
		a = a.menuKey(s, []string{routeAccountList, routeAccountHidden, routeAccountCreate})
	case routeAccountList:
		a = a.accountListKey(s, false)
	case routeAccountHidden:
		a = a.accountListKey(s, true)
	case routeAccountCreate:
		a = a.accountCreateKey(s)
	case routeBackup:
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

func exitConfirmActions() []string {
	return []string{"no", "yes"}
}

func (a App) exitConfirmKey(s string) (App, tea.Cmd) {
	actions := exitConfirmActions()
	switch s {
	case "down", "j", "tab":
		a = a.navSetMenu((a.Menu + 1) % len(actions))
	case "up", "k", "shift+tab":
		a = a.navSetMenu((a.Menu - 1 + len(actions)) % len(actions))
	case "esc":
		a.ExitAsk = false
	case "enter":
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
	case a.Path == routeAccounts:
		s := a.dashboardScreen()
		s.Path = routeAccounts
		s.Actions = []string{"list", "hidden", "create"}
		return s
	case a.Path == routeAccountList:
		return screen{Path: routeAccountList, Body: a.accountList(false), Help: listHelp()}
	case a.Path == routeAccountHidden:
		return screen{Path: routeAccountHidden, Body: a.accountList(true), Help: listHelp()}
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
		if name, ok := balancesName(a.Path); ok {
			return screen{Path: a.Path, Body: a.balanceList(name)}
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
	if s.Body != "" {
		b.WriteString(strings.TrimRight(s.Body, "\n") + "\n")
	}
	if s.Path != "" {
		if s.Body != "" {
			b.WriteString("\n")
		}
		b.WriteString(s.Path + "\n")
	}
	if s.Options != "" {
		if s.Path != "" || s.Body != "" {
			b.WriteString("\n")
		}
		b.WriteString(strings.TrimRight(s.Options, "\n") + "\n")
	}
	if len(s.Actions) > 0 {
		if s.Path != "" || s.Options != "" || s.Body != "" {
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
		return []string{"up/down/j/k   : navigate", "tab/shift-tab : navigate", "enter         : confirm", "esc           : back", "?             : help", "ctrl-z        : undo"}
	}
	return []string{"esc     : back", "?       : help", "ctrl-z  : undo"}
}

func (a App) formHelp(fields []string) []string {
	if a.Field >= len(fields) {
		return []string{"shift-tab : previous field", "enter     : confirm", "esc       : discard", "?         : help"}
	}
	if a.Field < len(fields) && fields[a.Field] == "notes" {
		return []string{"tab     : next field", "enter   : next field", "esc     : discard"}
	}
	return []string{"tab     : next field", "enter   : next field", "esc     : discard", "?       : help"}
}

func listHelp() []string {
	return []string{"up/down/j/k   : navigate", "tab/shift-tab : navigate", "enter         : confirm", "esc           : back", "?             : help", "ctrl-z        : undo"}
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
