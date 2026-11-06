package ui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	HardQuit   key.Binding
	Up         key.Binding
	Down       key.Binding
	Select     key.Binding
	Reveal     key.Binding
	Again      key.Binding
	Hard       key.Binding
	Good       key.Binding
	Easy       key.Binding
	Bury       key.Binding
	Suspend    key.Binding
	Undo       key.Binding
	Prereq     key.Binding
	Tab        key.Binding
	Back       key.Binding
	Help       key.Binding
	Quit       key.Binding
	NextInput  key.Binding
	PrevInput  key.Binding
	SubmitCard key.Binding
	ModeSearch key.Binding
	ModeAdd    key.Binding
	ModeStats  key.Binding
}

func defaultKeys() keyMap {
	return keyMap{
		HardQuit:   key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "hard quit")),
		Up:         key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k/up", "up")),
		Down:       key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j/down", "down")),
		Select:     key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
		Reveal:     key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "reveal")),
		Again:      key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "again")),
		Hard:       key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "hard")),
		Good:       key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "good")),
		Easy:       key.NewBinding(key.WithKeys("4"), key.WithHelp("4", "easy")),
		Bury:       key.NewBinding(key.WithKeys("b"), key.WithHelp("b", "bury")),
		Suspend:    key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "suspend")),
		Undo:       key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "undo")),
		Prereq:     key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "rule")),
		Tab:        key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "switch")),
		Back:       key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		Help:       key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Quit:       key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
		NextInput:  key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next field")),
		PrevInput:  key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "prev field")),
		SubmitCard: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "submit")),
		ModeSearch: key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "search")),
		ModeAdd:    key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add")),
		ModeStats:  key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "stats")),
	}
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Select, k.Reveal, k.Again, k.Hard, k.Good, k.Easy, k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Select, k.Reveal, k.Tab},
		{k.Again, k.Hard, k.Good, k.Easy, k.Back, k.Help, k.Quit},
		{k.Bury, k.Suspend, k.Undo, k.Prereq},
	}
}
