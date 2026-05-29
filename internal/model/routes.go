package model

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	routeRoot            = "/"
	routeAccountList     = "/accounts/list/"
	routeAccountCreate   = "/accounts/create/"
	routeTagList         = "/tags/list/"
	routeTagCreate       = "/tags/create/"
	routeBudgetList      = "/budgets/list/"
	routeBudgetCreate    = "/budgets/create/"
	routeBudgetCatList   = "/budgets/categories/list/"
	routeBudgetCatCreate = "/budgets/categories/create/"
	routeBackup          = "/backup/"
	routeSettings        = "/settings/"
	routeReports         = "/reports/"
	routeReportsMonthly  = "/reports/monthly/"
)

func accountPath(name string) string            { return "/accounts/" + name + "/" }
func accountEditPathFor(name string) string     { return "/accounts/" + name + "/edit/" }
func accountBalanceListPath(name string) string { return "/accounts/" + name + "/balances/list/" }
func accountBalancePath(name, date string) string {
	return "/accounts/" + name + "/balances/" + date + "/"
}
func accountBalanceAddPath(name string) string { return "/accounts/" + name + "/balances/add/" }
func accountBalanceEditPath(name, date string) string {
	return "/accounts/" + name + "/balances/" + date + "/edit/"
}
func accountTransactionsPath(name string) string  { return "/accounts/" + name + "/transactions/" }
func accountChildrenListPath(name string) string  { return "/accounts/" + name + "/children/list/" }
func accountChildCreatePath(name string) string   { return "/accounts/" + name + "/children/create/" }
func tagEditPathFor(name string) string           { return "/tags/" + name + "/edit/" }
func budgetPath(name string) string               { return "/budgets/" + name + "/" }
func budgetEditPathFor(name string) string        { return "/budgets/" + name + "/edit/" }
func budgetAllocationListPath(name string) string { return "/budgets/" + name + "/allocations/list/" }
func budgetAllocationAddPath(name string) string  { return "/budgets/" + name + "/allocations/add/" }
func budgetAllocationEditPath(name string, id int64) string {
	return fmt.Sprintf("/budgets/%s/allocations/%d/edit/", name, id)
}
func budgetCategoryPath(name string) string        { return "/budgets/categories/" + name + "/" }
func budgetCategoryEditPathFor(name string) string { return "/budgets/categories/" + name + "/edit/" }
func budgetCategoryBudgetCreatePath(name string) string {
	return "/budgets/categories/" + name + "/create-budget/"
}
func reportMonthlyDetailPath(month string) string { return "/reports/monthly/" + month + "/" }
func reportMonthlyAccountPath(month, name string) string {
	return "/reports/monthly/" + month + "/accounts/" + name + "/"
}

func Today() string { return time.Now().Format("2006-01-02") }

func reportMonthlyDetailMonth(path string) (string, bool) {
	if !strings.HasPrefix(path, "/reports/monthly/") || !strings.HasSuffix(path, "/") {
		return "", false
	}
	month := strings.TrimSuffix(strings.TrimPrefix(path, "/reports/monthly/"), "/")
	if strings.Contains(month, "/") || len(month) != len("2006-01") {
		return "", false
	}
	if _, err := time.Parse("2006-01", month); err != nil {
		return "", false
	}
	return month, true
}

func reportMonthlyAccount(path string) (string, string, bool) {
	if !strings.HasPrefix(path, "/reports/monthly/") || !strings.HasSuffix(path, "/") {
		return "", "", false
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(path, "/reports/monthly/"), "/"), "/")
	if len(parts) != 3 || parts[1] != "accounts" || parts[2] == "" {
		return "", "", false
	}
	month := parts[0]
	if len(month) != len("2006-01") {
		return "", "", false
	}
	if _, err := time.Parse("2006-01", month); err != nil {
		return "", "", false
	}
	if strings.Contains(parts[2], "/") {
		return "", "", false
	}
	return month, parts[2], true
}

func accountDetailName(path string) (string, bool) {
	if !strings.HasPrefix(path, "/accounts/") || !strings.HasSuffix(path, "/") {
		return "", false
	}
	rest := strings.TrimPrefix(path, "/accounts/")
	parts := strings.Split(strings.TrimSuffix(rest, "/"), "/")
	if len(parts) == 1 && parts[0] != "" && parts[0] != "list" && parts[0] != "create" {
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

func accountChildrenListName(path string) (string, bool) {
	if !strings.HasPrefix(path, "/accounts/") || !strings.HasSuffix(path, "/children/list/") {
		return "", false
	}
	name := strings.TrimSuffix(strings.TrimPrefix(path, "/accounts/"), "/children/list/")
	if strings.Contains(name, "/") {
		return "", false
	}
	return name, name != ""
}

func accountChildCreateName(path string) (string, bool) {
	if !strings.HasPrefix(path, "/accounts/") || !strings.HasSuffix(path, "/children/create/") {
		return "", false
	}
	name := strings.TrimSuffix(strings.TrimPrefix(path, "/accounts/"), "/children/create/")
	if strings.Contains(name, "/") {
		return "", false
	}
	return name, name != ""
}

func accountChildCreatePathMatch(path string) bool {
	_, ok := accountChildCreateName(path)
	return ok
}

func tagEditName(path string) (string, bool) {
	if !strings.HasPrefix(path, "/tags/") || !strings.HasSuffix(path, "/edit/") {
		return "", false
	}
	name := strings.TrimSuffix(strings.TrimPrefix(path, "/tags/"), "/edit/")
	return name, name != ""
}

func budgetDetailName(path string) (string, bool) {
	if !strings.HasPrefix(path, "/budgets/") || !strings.HasSuffix(path, "/") {
		return "", false
	}
	rest := strings.Trim(strings.TrimPrefix(path, "/budgets/"), "/")
	if rest == "" || rest == "list" || rest == "create" || strings.HasPrefix(rest, "categories") || strings.HasPrefix(rest, "hidden") || strings.Contains(rest, "/") {
		return "", false
	}
	return rest, true
}

func budgetEditName(path string) (string, bool) {
	if !strings.HasPrefix(path, "/budgets/") || !strings.HasSuffix(path, "/edit/") {
		return "", false
	}
	name := strings.TrimSuffix(strings.TrimPrefix(path, "/budgets/"), "/edit/")
	if name == "" || strings.Contains(name, "/") {
		return "", false
	}
	return name, true
}

func budgetEditPath(path string) bool {
	_, ok := budgetEditName(path)
	return ok
}

func budgetAllocationListName(path string) (string, bool) {
	if !strings.HasPrefix(path, "/budgets/") || !strings.HasSuffix(path, "/allocations/list/") {
		return "", false
	}
	name := strings.TrimSuffix(strings.TrimPrefix(path, "/budgets/"), "/allocations/list/")
	if name == "" || strings.Contains(name, "/") {
		return "", false
	}
	return name, true
}

func budgetAllocationAddName(path string) (string, bool) {
	if !strings.HasPrefix(path, "/budgets/") || !strings.HasSuffix(path, "/allocations/add/") {
		return "", false
	}
	name := strings.TrimSuffix(strings.TrimPrefix(path, "/budgets/"), "/allocations/add/")
	if name == "" || strings.Contains(name, "/") {
		return "", false
	}
	return name, true
}

func budgetAllocationAddPathMatch(path string) bool {
	_, ok := budgetAllocationAddName(path)
	return ok
}

func budgetAllocationEditName(path string) (string, int64, bool) {
	if !strings.HasPrefix(path, "/budgets/") || !strings.HasSuffix(path, "/edit/") {
		return "", 0, false
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(path, "/budgets/"), "/"), "/")
	if len(parts) != 4 || parts[1] != "allocations" || parts[3] != "edit" {
		return "", 0, false
	}
	id, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil || parts[0] == "" {
		return "", 0, false
	}
	return parts[0], id, true
}

func budgetCategoryDetailName(path string) (string, bool) {
	if !strings.HasPrefix(path, "/budgets/categories/") || !strings.HasSuffix(path, "/") {
		return "", false
	}
	name := strings.Trim(strings.TrimPrefix(path, "/budgets/categories/"), "/")
	if name == "" || name == "list" || name == "create" || strings.Contains(name, "/") {
		return "", false
	}
	return name, true
}

func budgetCategoryEditName(path string) (string, bool) {
	if !strings.HasPrefix(path, "/budgets/categories/") || !strings.HasSuffix(path, "/edit/") {
		return "", false
	}
	name := strings.TrimSuffix(strings.TrimPrefix(path, "/budgets/categories/"), "/edit/")
	if name == "" || strings.Contains(name, "/") {
		return "", false
	}
	return name, true
}

func budgetCategoryBudgetCreateName(path string) (string, bool) {
	if !strings.HasPrefix(path, "/budgets/categories/") || !strings.HasSuffix(path, "/create-budget/") {
		return "", false
	}
	name := strings.TrimSuffix(strings.TrimPrefix(path, "/budgets/categories/"), "/create-budget/")
	if name == "" || strings.Contains(name, "/") {
		return "", false
	}
	return name, true
}

func balanceListName(path string) (string, bool) {
	if !strings.HasPrefix(path, "/accounts/") || !strings.HasSuffix(path, "/balances/list/") {
		return "", false
	}
	name := strings.TrimSuffix(strings.TrimPrefix(path, "/accounts/"), "/balances/list/")
	if strings.Contains(name, "/") {
		return "", false
	}
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
	if len(parts) == 3 && parts[1] == "balances" && parts[2] != "add" && parts[2] != "list" {
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
