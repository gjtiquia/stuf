package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"stuf/internal/config"
	"stuf/internal/model"
	"stuf/internal/repo"
	"stuf/internal/service"
)

func main() {
	migrateOnly := flag.Bool("migrate-only", false, "create/open db, run migrations, seed data, then exit")
	flag.Parse()

	ctx := context.Background()
	cwd, err := os.Getwd()
	must(err)

	store, err := repo.Open(ctx, filepath.Join(cwd, "db.sqlite"))
	must(err)
	defer store.Close()

	if *migrateOnly {
		return
	}

	curSvc := service.CurrencyService{Currencies: store.Cur}
	loaded, err := config.Load(ctx, config.Options{
		CurrencyExists: curSvc.Exists,
		DetectCurrency: func() (string, bool) {
			return "", false
		},
	})
	must(err)

	if loaded.Warning != "" {
		fmt.Fprintln(os.Stderr, "warning:", loaded.Warning)
	}

	history := service.HistoryService{Repo: store.Hist, Now: store.Clock}
	accounts := service.AccountService{Store: store, Accounts: store.Acct, Balances: store.Bal, Currency: store.Cur, History: history, AppCurrency: loaded.Config.Currency}
	balances := service.BalanceService{Store: store, Accounts: store.Acct, Balances: store.Bal, History: history}
	dashboard := service.DashboardService{Accounts: store.Acct, Balances: store.Bal, Currencies: store.Cur, AppCurrency: loaded.Config.Currency}

	app := model.New(ctx, model.Services{
		Accounts:  accounts,
		Balances:  balances,
		Dashboard: dashboard,
		History:   history,
		Backup: func(ctx context.Context) (string, error) {
			return store.Backup(ctx, time.Now())
		},
	}, loaded)

	_, err = tea.NewProgram(app).Run()
	must(err)
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "stuf:", err)
		os.Exit(1)
	}
}
