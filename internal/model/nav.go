package model

func (a App) syncFromNav() App {
	prevPath := a.Path
	cur := a.Nav.Current()
	a.Path = cur.Path
	if a.Path == routeAccountList && prevPath != routeAccountList {
		a.AccountVisible = accountVisibilityNonHidden
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
