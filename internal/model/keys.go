package model

// Horizontal navigation keys are contextual:
// - menus: left/h back, right/l open (yazi-style)
// - paginated selects: left/right page; h/l stay available for filter typing
// - list-backed detail: left/h previous item, right/l next item
// - text fields: left/right move caret; h/l type normally
// - list screens: ctrl+n opens the matching create/add flow when one exists

func isMenuBackKey(s string) bool {
	return s == "left" || s == "h"
}

func isMenuForwardKey(s string) bool {
	return s == "right" || s == "l"
}

func isItemPrevKey(s string) bool {
	return isMenuBackKey(s)
}

func isItemNextKey(s string) bool {
	return isMenuForwardKey(s)
}

func isNewKey(s string) bool {
	return s == "ctrl+n"
}

func isVerticalNextKey(s string) bool {
	switch s {
	case "down", "j", "tab":
		return true
	default:
		return false
	}
}

func isVerticalPrevKey(s string) bool {
	switch s {
	case "up", "k", "shift+tab":
		return true
	default:
		return false
	}
}
