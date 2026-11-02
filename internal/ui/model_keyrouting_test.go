package ui

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

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

func pressKey(m Model, r rune) Model {
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	return updated.(Model)
}

func TestModeKeysReachEveryMode(t *testing.T) {
	// Tab from browse lands in the add form, which swallows tab for field
	// navigation, so without the direct keys Search and Stats cannot be opened.
	base := Model{keys: defaultKeys(), mode: ModeBrowse, searchInput: textinput.New()}

	if got := pressKey(base, 's'); got.mode != ModeSearch {
		t.Fatalf("s opened %q, want search", got.mode)
	}
	if got := pressKey(base, 't'); got.mode != ModeStats {
		t.Fatalf("t opened %q, want stats", got.mode)
	}
	if got := pressKey(base, 'a'); got.mode != ModeAdd {
		t.Fatalf("a opened %q, want add", got.mode)
	}
}

func TestModeKeyDoesNotHijackSearchTyping(t *testing.T) {
	// Inside the search box an "s" has to reach the input, not jump to stats.
	inSearch := Model{keys: defaultKeys(), mode: ModeSearch, searchInput: textinput.New()}
	if got := pressKey(inSearch, 's'); got.mode != ModeSearch || got.searchListFocus {
		t.Fatalf("typing s in search left mode=%q listFocus=%v", got.mode, got.searchListFocus)
	}
}

func TestCycleModeStartsAtSearch(t *testing.T) {
	m := Model{keys: defaultKeys(), mode: ModeBrowse, searchInput: textinput.New()}
	m.cycleMode()
	if m.mode != ModeSearch {
		t.Fatalf("tab from browse reached %q, want search", m.mode)
	}
}
