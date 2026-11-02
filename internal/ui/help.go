package ui

import (
	"strings"
)

func (m Model) renderHelpOverlay(base string) string {
	lines := []string{
		"deutsch-srs help",
		"",
		"Browse: j/k or arrows move, enter starts review",
		"Review: space reveal, 1/2/3/4 grade, b bury, x suspend, u undo, esc back",
		"Add: tab switch fields, enter submit",
		"Global: s search, a add, t stats, ? toggle help, q quit",
	}
	box := m.theme.HelpBox.Render(strings.Join(lines, "\n"))
	return base + "\n\n" + box
}
