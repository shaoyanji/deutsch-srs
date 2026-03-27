package ui

import "testing"

func TestIsTextEntryFocused(t *testing.T) {
	m := Model{}

	m.mode = ModeBrowse
	if m.isTextEntryFocused() {
		t.Fatal("browse should not be text-entry focused")
	}

	m.mode = ModeAdd
	if !m.isTextEntryFocused() {
		t.Fatal("add mode should be text-entry focused")
	}

	m.mode = ModeSearch
	m.searchListFocus = false
	if !m.isTextEntryFocused() {
		t.Fatal("search query input should be text-entry focused")
	}

	m.searchListFocus = true
	if m.isTextEntryFocused() {
		t.Fatal("search results focus should not be text-entry focused")
	}
}
