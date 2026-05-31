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
	accounts := service.AccountService{Store: store, Accounts: store.Acct, Balances: store.Bal, Currency: store.Cur, Tags: store.Tag, History: history, AppCurrency: loaded.Config.Currency}
	tags := service.TagService{Store: store, Tags: store.Tag, History: history}
	balances := service.BalanceService{Store: store, Accounts: store.Acct, Balances: store.Bal, History: history}
	budgetCategories := service.BudgetCategoryService{Store: store, Categories: store.BudCat, Budgets: store.Bud, History: history}
	budgets := service.BudgetService{Store: store, Budgets: store.Bud, Categories: store.BudCat, Currency: store.Cur, Allocations: store.Alloc, History: history, AppCurrency: loaded.Config.Currency}
	allocations := service.BudgetAllocationService{Store: store, Budgets: store.Bud, Allocations: store.Alloc, History: history}
	transactions := service.TransactionService{Store: store, Transactions: store.Txn, Accounts: store.Acct, Currency: store.Cur, Tags: store.Tag, History: history}
	owedLedgers := service.OwedLedgerService{Store: store, Ledgers: store.Owed, Transactions: store.OwedTx, Currency: store.Cur, History: history, AppCurrency: loaded.Config.Currency}
	owedTransactions := service.OwedTransactionService{Store: store, Ledgers: store.Owed, Transactions: store.OwedTx, Currency: store.Cur, History: history}
	dashboard := service.DashboardService{Accounts: store.Acct, Balances: store.Bal, Budgets: store.Bud, Allocations: store.Alloc, OwedLedgers: store.Owed, OwedTxns: store.OwedTx, Currencies: store.Cur, AppCurrency: loaded.Config.Currency}
	reports := service.ReportService{Accounts: store.Acct, Balances: store.Bal, Currencies: store.Cur, AppCurrency: loaded.Config.Currency}

	app := model.New(ctx, model.Services{
		Accounts:          accounts,
		Balances:          balances,
		Currency:          curSvc,
		Tags:              tags,
		BudgetCategories:  budgetCategories,
		Budgets:           budgets,
		BudgetAllocations: allocations,
		Transactions:      transactions,
		OwedLedgers:       owedLedgers,
		OwedTransactions:  owedTransactions,
		Dashboard:         dashboard,
		Reports:           reports,
		History:           history,
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
