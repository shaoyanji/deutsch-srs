package ui

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/shaoyanji/deutsch-srs/internal/srs"
)

// The author (anki-author's note_tags) gives every card an `id-<key>` tag and
// every drill a `req-<key>` tag naming the rule it tests. Following the edge is
// therefore one tag lookup: the reader takes the key off the drill and searches
// for the card that carries it, and never has to re-derive how a key is built.
const (
	prereqTagPrefix = "req-"
	cardTagPrefix   = "id-"
)

type prereqMsg struct {
	key   string
	card  srs.Card
	found bool
	err   error
}

// prereqKey is the rule the card points at, or "" when it points at nothing.
func prereqKey(tags []string) string {
	for _, tag := range tags {
		if key, ok := strings.CutPrefix(tag, prereqTagPrefix); ok && key != "" {
			return key
		}
	}
	return ""
}

func hasTag(tags []string, want string) bool {
	for _, tag := range tags {
		if tag == want {
			return true
		}
	}
	return false
}

// currentPrereqKey is the key for the card on screen, if any.
func (m Model) currentPrereqKey() string {
	if m.reviewIdx < 0 || m.reviewIdx >= len(m.reviewCards) {
		return ""
	}
	return prereqKey(m.reviewCards[m.reviewIdx].Tags)
}

// resetPrereq drops the loaded rule, so a panel opened on one drill cannot be
// read next to another.
func (m *Model) resetPrereq() {
	m.prereq = nil
	m.prereqKey = ""
	m.prereqErr = ""
	m.prereqOpen = false
}

// withPrereq batches the lookup for the card now on screen onto the commands a
// movement already returns, so the rule is ready by the time the learner asks.
func (m Model) withPrereq(cmds ...tea.Cmd) tea.Cmd {
	if key := m.currentPrereqKey(); key != "" {
		cmds = append(cmds, m.cmdLoadPrereq(key))
	}
	return tea.Batch(cmds...)
}

func (m Model) cmdLoadPrereq(key string) tea.Cmd {
	return func() tea.Msg {
		cards, err := m.backend.SearchCards(context.Background(), cardTagPrefix+key, 20)
		if err != nil {
			return prereqMsg{key: key, err: err}
		}
		for _, card := range cards {
			if hasTag(card.Tags, cardTagPrefix+key) {
				return prereqMsg{key: key, card: card, found: true}
			}
		}
		return prereqMsg{key: key}
	}
}

// prereqLabel is what the footer prints: the rule's own front once it has been
// resolved, and the key before that, so the line is never blank.
func (m Model) prereqLabel(key string) string {
	if m.prereqKey == key && m.prereq != nil {
		return m.prereq.Front
	}
	return key
}

// viewPrereq renders the rule a drill tests. It is read-only: opening it is a
// detour from the drill, not a second queue, so the drill stays where it was.
func (m Model) viewPrereq(key string) string {
	var b strings.Builder
	b.WriteString(m.theme.Title.Render("Rule this card tests") + "\n")
	switch {
	case m.prereqKey == key && m.prereqErr != "":
		b.WriteString(m.theme.Error.Render(m.prereqErr))
	case m.prereqKey == key && m.prereq != nil:
		b.WriteString(m.theme.CardFront.Render(m.prereq.Front) + "\n\n")
		b.WriteString(m.theme.CardBack.Render(m.prereq.Back) + "\n\n")
		b.WriteString(m.theme.Muted.Render("r or esc closes it; the drill stays where it was"))
	default:
		b.WriteString(m.theme.Muted.Render("Looking for " + key + " …"))
	}
	return b.String()
}
