package model

func (a App) syncFromNav() App {
	cur := a.Nav.Current()
	a.Path = cur.Path
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
	case routeAccounts:
		return 3
	case routeAccountList:
		return a.accountListRowCount(false)
	case routeAccountHidden:
		return a.accountListRowCount(true)
	case routeBackup:
		return 1
	default:
		if _, ok := accountDetailName(path); ok {
			return 4
		}
		if _, _, ok := balanceDetailName(path); ok {
			return 2
		}
		if name, ok := balancesName(path); ok {
			acct, err := a.Svc.Accounts.GetByName(a.ctx, name)
			if err != nil {
				return 1
			}
			rows, err := a.Svc.Balances.List(a.ctx, acct.ID)
			if err != nil {
				return 1
			}
			if len(rows) == 0 {
				return 1
			}
			return len(rows) + 1
		}
	}
	return 1
}
