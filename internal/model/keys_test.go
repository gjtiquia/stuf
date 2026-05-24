package model

import "testing"

func TestHorizontalNavKeyPredicates(t *testing.T) {
	for _, key := range []string{"left", "h"} {
		if !isMenuBackKey(key) || !isItemPrevKey(key) {
			t.Fatalf("%q should be menu back and item prev", key)
		}
		if isMenuForwardKey(key) || isItemNextKey(key) {
			t.Fatalf("%q should not be forward/next", key)
		}
	}
	for _, key := range []string{"right", "l"} {
		if !isMenuForwardKey(key) || !isItemNextKey(key) {
			t.Fatalf("%q should be menu forward and item next", key)
		}
		if isMenuBackKey(key) || isItemPrevKey(key) {
			t.Fatalf("%q should not be back/prev", key)
		}
	}
	for _, key := range []string{"enter", "esc", "j", "k", "up", "down"} {
		if isMenuBackKey(key) || isMenuForwardKey(key) || isItemPrevKey(key) || isItemNextKey(key) {
			t.Fatalf("%q should not match horizontal nav keys", key)
		}
	}
	for _, key := range []string{"down", "j", "tab"} {
		if !isVerticalNextKey(key) {
			t.Fatalf("%q should be vertical next", key)
		}
	}
	for _, key := range []string{"up", "k", "shift+tab"} {
		if !isVerticalPrevKey(key) {
			t.Fatalf("%q should be vertical prev", key)
		}
	}
}
