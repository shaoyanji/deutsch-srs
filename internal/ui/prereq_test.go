package ui

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/shaoyanji/deutsch-srs/internal/backend"
	"github.com/shaoyanji/deutsch-srs/internal/srs"
)

// prereqBackend answers the tag lookup the panel makes and records the key the
// drill asked for, so the test sees what was searched as well as what came back.
type prereqBackend struct {
	fakeBackend
	cards []srs.Card
}

func (b *prereqBackend) SearchCards(_ context.Context, _ string, _ int) ([]srs.Card, error) {
	return b.cards, nil
}

func ruleCard(id, front string, tags ...string) srs.Card {
	return srs.Card{ID: id, Deck: "Konnektoren", Front: front, Back: "zwar … aber", Tags: tags}
}

func feed(m Model, msg tea.Msg) Model {
	updated, _ := m.Update(msg)
	return updated.(Model)
}

func TestPrereqKeyReadsTheTagTheAuthorWrote(t *testing.T) {
	if got := prereqKey([]string{"noun", "req-zwar-aber-1a2b3c"}); got != "zwar-aber-1a2b3c" {
		t.Fatalf("prereqKey = %q", got)
	}
	if got := prereqKey([]string{"id-nur-selbst-1a2b3c", "noun"}); got != "" {
		t.Fatalf("a card with no req- tag reported %q", got)
	}
}

func TestLoadingAQueueAsksForTheRuleOnScreen(t *testing.T) {
	be := &prereqBackend{}
	drill := testCard("a", "Er ist zwar müde, aber …")
	drill.Tags = []string{"id-er-ist-zwar-muede-9f9f9f", "req-zwar-aber-1a2b3c"}
	m := reviewModel(be, drill)

	updated, cmd := m.Update(queueMsg{queue: backend.Queue{Cards: []srs.Card{drill}}})
	if cmd == nil {
		t.Fatal("entering review did not ask for the rule")
	}
	if got := updated.(Model).currentPrereqKey(); got != "zwar-aber-1a2b3c" {
		t.Fatalf("the card on screen points at %q", got)
	}
}

func TestOpeningTheRuleResolvesTheTagAndShowsItsFront(t *testing.T) {
	be := &prereqBackend{cards: []srs.Card{
		// A near miss the exact tag check has to reject: same prefix, other card.
		ruleCard("1", "zwar … aber", "id-zwar-aber-1a2b3d"),
		ruleCard("2", "zwar … aber (kontrastiv)", "id-zwar-aber-1a2b3c"),
	}}
	drill := testCard("a", "Er ist zwar müde, aber …")
	drill.Tags = []string{"id-er-ist-zwar-muede-9f9f9f", "req-zwar-aber-1a2b3c"}
	m := reviewModel(be, drill)

	m, _ = pressRune(m, 'r')
	if !m.prereqOpen {
		t.Fatal("r did not open the panel")
	}

	m = feed(m, m.cmdLoadPrereq("zwar-aber-1a2b3c")())
	if m.prereqErr != "" {
		t.Fatalf("resolve reported %q", m.prereqErr)
	}
	if m.prereq == nil || m.prereq.ID != "2" {
		t.Fatalf("panel resolved %+v, want the card whose id- tag matches exactly", m.prereq)
	}
	view := m.viewReview()
	if !strings.Contains(view, "Rule: zwar … aber (kontrastiv)") {
		t.Fatalf("view does not name the rule:\n%s", view)
	}
	if !strings.Contains(view, "zwar … aber") {
		t.Fatalf("panel does not show the rule body:\n%s", view)
	}
}

func TestEscClosesThePanelInsteadOfLeavingReview(t *testing.T) {
	drill := testCard("a", "Er ist zwar müde")
	drill.Tags = []string{"req-zwar-aber-1a2b3c"}
	m := reviewModel(&fakeBackend{}, drill)

	m, _ = pressRune(m, 'r')
	if !m.prereqOpen {
		t.Fatal("r did not open the panel")
	}
	m = feed(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.prereqOpen {
		t.Fatal("esc left the panel open")
	}
	if m.mode != ModeReview {
		t.Fatalf("esc left review mode: %q", m.mode)
	}

	m = feed(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.mode != ModeBrowse {
		t.Fatalf("second esc stayed in %q, want browse", m.mode)
	}
}

func TestRulePanelSaysSoWhenTheCardTestsNothing(t *testing.T) {
	m := reviewModel(&fakeBackend{}, testCard("a", "Haus"))

	m, _ = pressRune(m, 'r')
	if m.prereqOpen {
		t.Fatal("r opened a panel for a card with no prerequisite")
	}
	if !strings.Contains(m.infoText, "no prerequisite") {
		t.Fatalf("info = %q", m.infoText)
	}
}

func TestAStaleLookupIsDropped(t *testing.T) {
	drill := testCard("b", "Er ist zwar müde")
	drill.Tags = []string{"req-zwar-aber-1a2b3c"}
	m := reviewModel(&fakeBackend{}, drill)

	// The answer belongs to a card the learner has already graded past.
	m = feed(m, prereqMsg{key: "some-other-key", card: ruleCard("9", "alt"), found: true})
	if m.prereq != nil || m.prereqKey != "" {
		t.Fatalf("stale lookup was kept: %+v", m.prereq)
	}
}

func TestAMissingRuleIsReportedNotSilent(t *testing.T) {
	drill := testCard("b", "Er ist zwar müde")
	drill.Tags = []string{"req-zwar-aber-1a2b3c"}
	m := reviewModel(&fakeBackend{}, drill)

	m = feed(m, prereqMsg{key: "zwar-aber-1a2b3c"})
	if m.prereqErr == "" {
		t.Fatal("an unresolved rule said nothing")
	}
	if !strings.Contains(m.viewPrereq("zwar-aber-1a2b3c"), "id-zwar-aber-1a2b3c") {
		t.Fatalf("the panel does not name the tag it looked for:\n%s", m.viewPrereq("zwar-aber-1a2b3c"))
	}
}
