package component

import "strings"

type FilterList struct {
	Items  []string
	Filter string
}

func (l FilterList) Visible() []string {
	var out []string
	for _, item := range l.Items {
		if strings.Contains(item, l.Filter) {
			out = append(out, item)
		}
	}
	return out
}
