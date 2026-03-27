package ui

import (
	"fmt"
	"strings"
)

func (m Model) viewAdd() string {
	var b strings.Builder
	b.WriteString(m.theme.Title.Render("Add Card") + "\n\n")
	labels := []string{"Deck *", "Front *", "Back *", "Tags (comma)"}
	values := make([]string, 0, len(m.addInputs))
	for i, in := range m.addInputs {
		values = append(values, in.Value())
		cursor := "  "
		if i == m.focusedInput {
			cursor = "> "
		}
		line := cursor + labels[i] + ": " + in.View()
		if i == m.focusedInput {
			line = m.theme.Selected.Render(line)
		}
		b.WriteString(line + "\n")
	}
	required := addRequiredFilled(values[0], values[1], values[2])
	b.WriteString("\n" + m.theme.StatusLine.Render(fmt.Sprintf("Required fields: %d/3", required)) + "\n")
	if addFormIsEmpty(values) {
		b.WriteString(m.theme.Muted.Render("Form is empty. Fill Deck, Front, and Back to add a card.") + "\n")
	}
	if m.infoText != "" {
		b.WriteString(m.theme.Muted.Render(m.infoText) + "\n")
	}
	if m.errText != "" {
		b.WriteString(m.theme.Error.Render("Add error: "+m.errText) + "\n")
	}
	b.WriteString("\n")
	b.WriteString(m.theme.Muted.Render("tab/j/k move fields, enter moves forward, enter on Tags submits, esc cancel"))
	return strings.TrimRight(b.String(), "\n")
}
