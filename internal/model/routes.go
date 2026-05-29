package model

import (
	"strings"
	"time"
)

const (
	routeRoot           = "/"
	routeAccountList    = "/accounts/list/"
	routeAccountCreate  = "/accounts/create/"
	routeTagList        = "/tags/list/"
	routeTagCreate      = "/tags/create/"
	routeBackup         = "/backup/"
	routeSettings       = "/settings/"
	routeReports        = "/reports/"
	routeReportsMonthly = "/reports/monthly/"
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
func reportMonthlyDetailPath(month string) string { return "/reports/monthly/" + month + "/" }

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
