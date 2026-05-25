package model

func (a App) captureAccountListReturn(item string) App {
	a.ListReturn = listReturnState{
		Path:           routeAccountList,
		Item:           item,
		Filter:         a.listFilter(),
		AccountVisible: a.AccountVisible,
	}
	return a
}

func (a App) captureBalanceListReturn(name, item string) App {
	a.ListReturn = listReturnState{
		Path: accountBalanceListPath(name),
		Item: item,
	}
	return a
}

func (a App) returnToListOrigin(item string) (App, bool) {
	origin := a.ListReturn
	if origin.Path == "" {
		return a, false
	}
	a.ListReturn = listReturnState{}
	if origin.Path == routeAccountList {
		a.AccountVisible = origin.AccountVisible
		a.Form[formKeyFilter] = origin.Filter
		return a.selectAccountInCurrentList(item), true
	}
	if _, ok := balanceListName(origin.Path); ok {
		return a.selectBalanceInList(origin.Path, item), true
	}
	return a.navReplace(origin.Path, 0), true
}
