package ui

import (
	"fmt"
	"strings"
)

func (m Model) viewSearchList() string {
	var b strings.Builder
	b.WriteString(m.theme.Title.Render("Search") + "\n\n")
	b.WriteString("Query: " + m.searchInput.View() + "\n")
	b.WriteString(m.theme.Muted.Render("enter search, j/k move results, esc back, tab next mode") + "\n\n")

	if m.errText != "" {
		b.WriteString(m.theme.Error.Render("Search error: "+m.errText) + "\n\n")
	}

	if !m.searchExecuted {
		b.WriteString(m.theme.Muted.Render("Type a query and press enter."))
		return strings.TrimRight(b.String(), "\n")
	}
	if len(m.searchResults) == 0 {
		b.WriteString(m.theme.Muted.Render("No results found for current query."))
		return strings.TrimRight(b.String(), "\n")
	}

	// The hint line wraps on a narrow pane, so leave it a row.
	rows := m.height - 10
	if rows < 3 {
		rows = 3
	}
	start, end := visibleWindow(m.searchCursor, len(m.searchResults), rows)
	for i := start; i < end; i++ {
		c := m.searchResults[i]
		line := fmt.Sprintf("%-16s %s", c.Deck, trimOneLine(c.Front, 30))
		if i == m.searchCursor {
			line = m.theme.Selected.Render(line)
		}
		b.WriteString(line + "\n")
	}
	if len(m.searchResults) > end-start {
		b.WriteString(m.theme.Muted.Render(fmt.Sprintf("%d-%d of %d", start+1, end, len(m.searchResults))) + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m Model) viewSearchDetail() string {
	var b strings.Builder
	b.WriteString(m.theme.Title.Render("Search Detail") + "\n\n")
	if m.errText != "" {
		b.WriteString(m.theme.Error.Render("Search error: "+m.errText) + "\n\n")
	}
	if !m.searchExecuted {
		b.WriteString(m.theme.Muted.Render("No query run yet. Use the Search pane to run a query."))
		return strings.TrimRight(b.String(), "\n")
	}
	if len(m.searchResults) == 0 {
		b.WriteString(m.theme.Muted.Render("No card selected because there are no results."))
		return strings.TrimRight(b.String(), "\n")
	}
	if m.searchCursor >= len(m.searchResults) {
		b.WriteString(m.theme.Error.Render("Selected result is out of range."))
		return strings.TrimRight(b.String(), "\n")
	}

	card := m.searchResults[m.searchCursor]
	b.WriteString(m.theme.StatusLine.Render(fmt.Sprintf("Result %d/%d  Deck: %s", m.searchCursor+1, len(m.searchResults), card.Deck)) + "\n\n")
	b.WriteString("Front\n")
	b.WriteString(m.theme.CardFront.Render(card.Front) + "\n\n")
	b.WriteString("Back\n")
	b.WriteString(m.theme.CardBack.Render(card.Back) + "\n\n")
	if len(card.Tags) > 0 {
		b.WriteString(m.theme.Muted.Render("Tags: "+strings.Join(card.Tags, ", ")) + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func trimOneLine(in string, max int) string {
	s := strings.ReplaceAll(in, "\n", " ")
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}
