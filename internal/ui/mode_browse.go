package ui

import (
	"fmt"
	"strings"
)

func (m Model) viewBrowse() string {
	var b strings.Builder
	b.WriteString(m.theme.Title.Render("Decks") + "\n\n")
	if len(m.decks) == 0 {
		b.WriteString(m.theme.Muted.Render("No decks available. Switch to Add mode and create your first card."))
		if m.errText != "" {
			b.WriteString("\n\n" + m.theme.Error.Render("Browse error: "+m.errText))
		}
		return b.String()
	}
	for i, d := range m.decks {
		line := fmt.Sprintf("%-20s due:%d new:%d rev:%d", d.Name, d.DueCount, d.NewCount, d.ReviewCount)
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
