package model

import (
	"fmt"
	"sort"
	"strings"

	"stuf/internal/repo"
)

const tagPageSize = 8

type tagOption struct {
	Name   string
	Create bool
}

func joinTagNames(names []string) string {
	return strings.Join(uniqueSortedTags(names), ",")
}

func splitTagNames(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := []string{}
	seen := map[string]bool{}
	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	return out
}

func uniqueSortedTags(names []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func formatTags(names []string, newNames []string) string {
	names = splitTagNames(joinTagNames(names))
	if len(names) == 0 {
		return "[]"
	}
	newSet := map[string]bool{}
	for _, name := range newNames {
		newSet[name] = true
	}
	parts := make([]string, len(names))
	for i, name := range names {
		if newSet[name] {
			name += "*"
		}
		parts[i] = name
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func tagNames(tags []repo.Tag) []string {
	out := make([]string, len(tags))
	for i, tag := range tags {
		out[i] = tag.Name
	}
	return out
}

func (a App) directTagNames(accountID int64) []string {
	tags, err := a.Svc.Accounts.ListDirectTags(a.ctx, accountID)
	if err != nil {
		return nil
	}
	return tagNames(tags)
}

func (a App) effectiveTagNames(accountID int64) []string {
	tags, err := a.Svc.Accounts.ListEffectiveTags(a.ctx, accountID)
	if err != nil {
		return nil
	}
	return tagNames(tags)
}

func (a App) inheritedTagNames(accountID int64) []string {
	direct := map[string]bool{}
	for _, name := range a.directTagNames(accountID) {
		direct[name] = true
	}
	var out []string
	for _, name := range a.effectiveTagNames(accountID) {
		if !direct[name] {
			out = append(out, name)
		}
	}
	return out
}

func (a App) tagOptions() []string {
	tags, err := a.Svc.Tags.List(a.ctx)
	if err != nil {
		return nil
	}
	out := make([]string, len(tags))
	for i, tag := range tags {
		out[i] = tag.Name
	}
	return out
}

func (a App) currentTagOptions() []tagOption {
	selected := map[string]bool{}
	for _, name := range splitTagNames(a.Form["tags"]) {
		selected[name] = true
	}
	filter := a.tagFilter()
	var opts []tagOption
	exact := false
	for _, name := range a.tagOptions() {
		if selected[name] {
			continue
		}
		if filter != "" && !strings.Contains(name, filter) {
			continue
		}
		if name == filter {
			exact = true
		}
		opts = append(opts, tagOption{Name: name})
	}
	if filter != "" && !exact {
		opts = append(opts, tagOption{Name: filter, Create: true})
	}
	return opts
}

func (a App) tagFieldKey(s string, fields []string) (App, bool) {
	options := a.currentTagOptions()
	cursor := clampCursor(parseFormInt(a.Form[tagCursorKey]), len(options))
	a.setTagSelectCursor(cursor)
	switch s {
	case "down":
		if len(options) > 0 {
			a.setTagSelectCursor((cursor + 1) % len(options))
		}
	case "up":
		if len(options) > 0 {
			a.setTagSelectCursor((cursor - 1 + len(options)) % len(options))
		}
	case "right":
		if len(options) > 0 {
			page := min(a.tagSelectPage()+1, tagPageCount(len(options))-1)
			a.setTagSelectPage(page)
			a.setTagSelectCursor(min(page*tagPageSize, len(options)-1))
		}
	case "left":
		if len(options) > 0 {
			page := max(a.tagSelectPage()-1, 0)
			a.setTagSelectPage(page)
			a.setTagSelectCursor(min(page*tagPageSize, len(options)-1))
		}
	case "backspace":
		if a.tagFilter() == "" {
			selected := splitTagNames(a.Form["tags"])
			if len(selected) > 0 {
				removed := selected[len(selected)-1]
				a.Form["tags"] = joinTagNames(selected[:len(selected)-1])
				a.Form[newTagsKey] = joinTagNames(removeTagName(splitTagNames(a.Form[newTagsKey]), removed))
			}
			return a, false
		}
		a.setTagFilter(trimLastRune(a.tagFilter()))
		a.resetTagSelectCursor()
	case "tab":
		a.clearTagSelectState()
		a.Field = min(a.Field+1, len(fields))
	case "shift+tab":
		a.clearTagSelectState()
		a.Field = max(a.Field-1, 0)
	case "enter":
		if len(options) == 0 {
			if strings.TrimSpace(a.Form["name"]) != "" {
				a.clearTagSelectState()
				return a, true
			}
			a.clearTagSelectState()
			a.Field = min(a.Field+1, len(fields))
			return a, false
		}
		opt := options[cursor]
		selected := append(splitTagNames(a.Form["tags"]), opt.Name)
		a.Form["tags"] = joinTagNames(selected)
		if opt.Create {
			a.Form[newTagsKey] = joinTagNames(append(splitTagNames(a.Form[newTagsKey]), opt.Name))
		}
		a.clearTagSelectState()
	default:
		input := newFilteredListInput(a.tagFilter(), sanitizeTagSlug)
		if updated, handled := input.handleKey(s); handled {
			a.setTagFilter(updated.value())
			a.resetTagSelectCursor()
		}
	}
	return a, false
}

func removeTagName(names []string, removed string) []string {
	var out []string
	for _, name := range names {
		if name != removed {
			out = append(out, name)
		}
	}
	return out
}

func tagPageCount(count int) int {
	if count == 0 {
		return 1
	}
	return (count + tagPageSize - 1) / tagPageSize
}

func (a App) tagSelectLines() []string {
	filter := a.tagFilter()
	options := a.currentTagOptions()
	cursor := clampCursor(parseFormInt(a.Form[tagCursorKey]), len(options))
	page := min(a.tagSelectPage(), tagPageCount(len(options))-1)
	start := page * tagPageSize
	end := min(start+tagPageSize, len(options))
	lines := []string{"", fmt.Sprintf("   > filter  : %s", placeholder(filter, "(type anything...)")), ""}
	if len(options) == 0 {
		lines = append(lines, "     (no matching tags)", "", "     [00/00]")
		return lines
	}
	for i, option := range options[start:end] {
		prefix := "       "
		if start+i == cursor {
			prefix = "     > "
		}
		label := option.Name
		if option.Create {
			label = fmt.Sprintf("(create new %q)", option.Name)
		}
		lines = append(lines, prefix+label)
	}
	lines = append(lines, "", fmt.Sprintf("     [%02d/%02d]", cursor+1, len(options)))
	return lines
}
