package ui

import "strings"

func (m Model) viewBrowse() string {
	var b strings.Builder
	b.WriteString(m.theme.Title.Render("Decks") + "\n\n")
	if len(m.decks) == 0 {
		b.WriteString(m.theme.Muted.Render("No decks yet. Add a card, or import a deck from Add mode."))
		if m.errText != "" {
			b.WriteString("\n\n" + m.theme.Error.Render("Browse error: "+m.errText))
		}
		return b.String()
	}
	nameWidth := m.deckNameWidth()
	b.WriteString(m.theme.Muted.Render(deckHeader(nameWidth)) + "\n")
	for i, d := range m.decks {
		line := deckRow(d.Name, d.DueCount, d.NewCount, d.ReviewCount, nameWidth)
		if i == m.deckCursor {
			line = m.theme.Selected.Render(line)
		}
		b.WriteString(line + "\n")
	}
	b.WriteString("\n" + m.theme.Muted.Render("enter starts review for selected deck"))
	if m.errText != "" {
		b.WriteString("\n\n" + m.theme.Error.Render("Browse error: "+m.errText))
	}
	return strings.TrimRight(b.String(), "\n")
}

// deckNameWidth keeps one deck row inside the left pane whatever the counts are.
func (m Model) deckNameWidth() int {
	leftW, _ := m.paneWidths()
	w := leftW - deckRowChrome
	if w < 8 {
		return 8
	}
	if w > 30 {
		return 30
	}
	return w
}
