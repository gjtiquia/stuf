package model

import (
	"fmt"
	"strings"
)

func menuItems(items []string, selected int) string {
	var b strings.Builder
	for i, item := range items {
		prefix := "  "
		if i == selected {
			prefix = "> "
		}
		b.WriteString(fmt.Sprintf("%s%d) %s\n", prefix, i+1, item))
	}
	return b.String()
}

func zero(code string) string { return code + " 0.00" }

func placeholder(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func placeholderFor(field string) string {
	switch field {
	case "name", "notes":
		return "(type anything...)"
	case "balance":
		return "(type amount...)"
	default:
		return ""
	}
}

func normalizeFieldInput(field, current, input string) string {
	return normalizeFieldValue(field, current+input)
}

func normalizeFieldValue(field, value string) string {
	if field != "name" {
		return value
	}
	return sanitizeSlug(value)
}

func isTextInputKey(input string) bool {
	switch input {
	case "", "left", "right", "up", "down", "enter", "esc", "tab", "shift+tab", "backspace", "ctrl+c", "ctrl+z", "?":
		return false
	default:
		return true
	}
}

func trimLastRune(s string) string {
	if s == "" {
		return ""
	}
	runes := []rune(s)
	return string(runes[:len(runes)-1])
}

func sanitizeSlug(input string) string {
	var b strings.Builder
	lastHyphen := false
	for _, r := range input {
		switch {
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + ('a' - 'A'))
			lastHyphen = false
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			lastHyphen = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			lastHyphen = false
		case r == '-' || r == ' ' || r == '\t' || r == '\n' || r == '\r':
			if b.Len() > 0 && !lastHyphen {
				b.WriteByte('-')
				lastHyphen = true
			}
		}
	}
	return b.String()
}

func parseBoolDefault(value string, fallback bool) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "yes", "1", "on":
		return true
	case "false", "no", "0", "off":
		return false
	default:
		return fallback
	}
}

func indexOf(values []string, needle string) int {
	for i, value := range values {
		if value == needle {
			return i
		}
	}
	return -1
}

func accountFormValues(name, code string, onBudget bool, notes string) map[string]string {
	return map[string]string{
		"name":      name,
		"currency":  code,
		"on-budget": fmt.Sprintf("%t", onBudget),
		"notes":     notes,
	}
}

func rawAmount(amount int64, scale int) string {
	sign := ""
	if amount < 0 {
		sign = "-"
		amount = -amount
	}
	if scale == 0 {
		return fmt.Sprintf("%s%d", sign, amount)
	}
	div := int64(1)
	for range scale {
		div *= 10
	}
	return fmt.Sprintf("%s%d.%0*d", sign, amount/div, scale, amount%div)
}
