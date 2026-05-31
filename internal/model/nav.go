package model

func (a App) syncFromNav() App {
	prevPath := a.Path
	cur := a.Nav.Current()
	a.Path = cur.Path
	if a.Path == routeAccountList && prevPath != routeAccountList {
		a.AccountVisible = accountVisibilityNonHidden
	}
	if a.Path == routeBudgetList && prevPath != routeBudgetList {
		a.BudgetVisible = accountVisibilityNonHidden
	}
	a.Menu = clampListCursor(cur.Menu, a.menuCountFor(cur.Path))
	a.Nav.setCurrentMenu(a.Menu)
	return a
}

func (a App) navSetMenu(menu int) App {
	a.Menu = clampListCursor(menu, a.menuCountFor(a.Path))
	a.Nav.setCurrentMenu(a.Menu)
	return a
}

func (a App) navPush(path string, menu int) App {
	a.Nav.setCurrentMenu(a.Menu)
	a.Nav.Push(path, menu)
	return a.syncFromNav()
}

func (a App) navReplace(path string, menu int) App {
	a.Nav.Replace(path, menu)
	return a.syncFromNav()
}

func (a App) navReset() App {
	a.Nav.Reset()
	return a.syncFromNav()
}

func (a App) goBack() App {
	a.Nav.setCurrentMenu(a.Menu)
	if !a.Nav.Pop() {
		return a
	}
	return a.syncFromNav()
}

func (a App) menuCountFor(path string) int {
	switch path {
	case routeRoot:
		return 7
	case routeAccountList:
		return a.accountListRowCount()
	case routeTagList:
		rows, err := a.filteredTags()
		if err != nil {
			return 0
		}
		return len(rows)
	case routeBudgetList:
		return a.budgetListRowCount()
	case routeOwedList:
		rows, err := a.filteredOwedLedgerRows()
		if err != nil {
			return 0
		}
		return len(rows)
	case routeTransactionList:
		return a.transactionListRowCount(0)
	case routeBudgetCatList:
		rows, err := a.filteredBudgetCategories()
		if err != nil {
			return 0
		}
		return len(rows)
	case routeReports:
		return len(reportActions())
	case routeReportsMonthly:
		rows, err := a.filteredReportMonthlyRows()
		if err != nil {
			return 0
		}
		return len(rows)
	case routeBackup:
		return 1
	default:
		if month, name, ok := reportMonthlyAccount(path); ok {
			detail, err := a.Svc.Reports.MonthlyAccountDetail(a.ctx, month, name)
			if err != nil {
				return 0
			}
			return len(detail.Snapshots)
		}
		if month, ok := reportMonthlyDetailMonth(path); ok {
			rows, err := a.filteredReportAccountRows(month)
			if err != nil {
				return 0
			}
			return len(rows)
		}
		if _, ok := accountDetailName(path); ok {
			return 4
		}
		if name, ok := accountTransactionListName(path); ok {
			acct, err := a.Svc.Accounts.GetByName(a.ctx, name)
			if err != nil {
				return 0
			}
			return a.transactionListRowCount(acct.ID)
		}
		if ref, ok := transactionChildrenListRef(path); ok {
			tx, err := a.transactionByRefString(ref)
			if err != nil {
				return 0
			}
			return a.transactionChildrenRowCount(tx.ID)
		}
		if _, ok := transactionRef(path); ok {
			return 4
		}
		if _, ok := budgetDetailName(path); ok {
			return 3
		}
		if _, ok := owedLedgerDetailName(path); ok {
			return 2
		}
		if name, ok := owedTransactionListName(path); ok {
			rows, err := a.owedTransactionRows(name)
			if err != nil {
				return 0
			}
			return len(rows)
		}
		if _, _, ok := owedTransactionRefName(path); ok {
			return 2
		}
		if name, ok := budgetAllocationListName(path); ok {
			rows, err := a.budgetAllocationRows(name)
			if err != nil {
				return 0
			}
			return len(rows)
		}
		if _, ok := budgetCategoryDetailName(path); ok {
			return 3
		}
		if name, ok := balanceListName(path); ok {
			acct, err := a.Svc.Accounts.GetByName(a.ctx, name)
			if err != nil {
				return 0
			}
			rows, err := a.Svc.Balances.List(a.ctx, acct.ID)
			if err != nil {
				return 0
			}
			return len(rows)
		}
		if name, ok := accountChildrenListName(path); ok {
			parent, err := a.Svc.Accounts.GetByName(a.ctx, name)
			if err != nil {
				return 0
			}
			rows, err := a.childAccountListRows(parent.ID)
			if err != nil {
				return 0
			}
			return len(rows)
		}
		if _, _, ok := balanceDetailName(path); ok {
			return 2
		}
	}
	return 1
}
