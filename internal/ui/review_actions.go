package ui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/shaoyanji/deutsch-srs/internal/backend"
	"github.com/shaoyanji/deutsch-srs/internal/srs"
)

// undoDepth bounds the undo stack: a session is thousands of cards and only
// the last few actions are ever worth taking back.
const undoDepth = 50

// undoEntry is one review action as it was before it happened, which is both
// what undo has to write back and where the card has to reappear.
type undoEntry struct {
	card       srs.Card
	graded     bool
	requeued   bool
	countedNew bool
	index      int
	verb       string
}

type asideMsg struct {
	cardID  string
	suspend bool
	err     error
}

type undoMsg struct{ err error }

func (m *Model) pushUndo(e undoEntry) {
	m.undoStack = append(m.undoStack, e)
	if len(m.undoStack) > undoDepth {
		m.undoStack = m.undoStack[len(m.undoStack)-undoDepth:]
	}
}

// setAside takes the current card out of this session and asks the backend to
// hide it: suspended cards stay gone until unsuspended, buried ones come back
// tomorrow. Neither is a grade, so the schedule is left alone — which is the
// point, since a card you cannot face should not be scored as a lapse.
func (m Model) setAside(suspend bool) (tea.Model, tea.Cmd) {
	card := m.reviewCards[m.reviewIdx]
	index := m.reviewIdx
	m.reviewCards = append(m.reviewCards[:index], m.reviewCards[index+1:]...)
	m.revealed = false
	m.errText = ""
	verb := "buried"
	if suspend {
		verb = "suspended"
	}
	m.pushUndo(undoEntry{card: card, index: index, verb: verb})
	m.infoText = "Card " + verb + ": " + card.Front + "  (u to undo)"
	return m, m.cmdSetAside(card.ID, suspend)
}

// undoLast puts the last action back: the card returns to its slot in the
// queue, the backend restores the stored state, and a review it consumed stops
// counting toward today's new cards.
func (m Model) undoLast() (tea.Model, tea.Cmd) {
	if len(m.undoStack) == 0 {
		m.infoText = "Nothing to undo."
		return m, nil
	}
	e := m.undoStack[len(m.undoStack)-1]
	m.undoStack = m.undoStack[:len(m.undoStack)-1]

	// An "Again" grade queued a copy at the tail; that copy belongs to the
	// action being reverted, so it goes rather than leaving the card twice.
	if e.requeued {
		if n := len(m.reviewCards); n > 0 && m.reviewCards[n-1].ID == e.card.ID {
			m.reviewCards = m.reviewCards[:n-1]
		}
	}

	// A grade left the card in the slice (only the cursor moved); a bury or a
	// suspend removed it, so it has to be inserted again.
	idx := e.index
	if idx > len(m.reviewCards) {
		idx = len(m.reviewCards)
	}
	if idx < len(m.reviewCards) && m.reviewCards[idx].ID == e.card.ID {
		m.reviewCards[idx] = e.card
	} else {
		m.reviewCards = append(m.reviewCards, srs.Card{})
		copy(m.reviewCards[idx+1:], m.reviewCards[idx:])
		m.reviewCards[idx] = e.card
	}
	m.reviewIdx = idx
	m.revealed = false
	m.pendingAgainCard = nil
	m.errText = ""
	if e.countedNew {
		if m.reviewNewToday > 0 {
			m.reviewNewToday--
		}
		delete(m.newCounted, e.card.ID)
	}
	m.infoText = "Undid " + e.verb + ": " + e.card.Front
	return m, m.cmdUndoReview(backend.ReviewUndo{Card: e.card, Graded: e.graded})
}

func (m Model) cmdSetAside(cardID string, suspend bool) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		var err error
		if suspend {
			err = m.backend.SuspendCard(ctx, cardID)
		} else {
			err = m.backend.BuryCard(ctx, cardID)
		}
		return asideMsg{cardID: cardID, suspend: suspend, err: err}
	}
}

func (m Model) cmdUndoReview(undo backend.ReviewUndo) tea.Cmd {
	return func() tea.Msg {
		return undoMsg{err: m.backend.UndoReview(context.Background(), undo)}
	}
}
