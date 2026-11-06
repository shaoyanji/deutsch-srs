package ui

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/shaoyanji/deutsch-srs/internal/backend"
	"github.com/shaoyanji/deutsch-srs/internal/srs"
)

type fakeBackend struct {
	answers   []string
	buried    []string
	suspended []string
	undos     []backend.ReviewUndo
	err       error
}

func (f *fakeBackend) ListDecks(context.Context) ([]srs.Deck, error) { return nil, f.err }

func (f *fakeBackend) DueCards(context.Context, string, int) ([]srs.Card, error) {
	return nil, f.err
}

func (f *fakeBackend) ReviewQueue(context.Context, string, int) (backend.Queue, error) {
	return backend.Queue{}, f.err
}

func (f *fakeBackend) AddCard(context.Context, backend.AddCardInput) (srs.Card, error) {
	return srs.Card{}, f.err
}

func (f *fakeBackend) AnswerCard(_ context.Context, id string, _ srs.ReviewGrade) error {
	f.answers = append(f.answers, id)
	return f.err
}

func (f *fakeBackend) SuspendCard(_ context.Context, id string) error {
	f.suspended = append(f.suspended, id)
	return f.err
}

func (f *fakeBackend) UnsuspendCard(_ context.Context, id string) error { return f.err }

func (f *fakeBackend) BuryCard(_ context.Context, id string) error {
	f.buried = append(f.buried, id)
	return f.err
}

func (f *fakeBackend) UndoReview(_ context.Context, undo backend.ReviewUndo) error {
	f.undos = append(f.undos, undo)
	return f.err
}

func (f *fakeBackend) SearchCards(context.Context, string, int) ([]srs.Card, error) {
	return nil, f.err
}

func (f *fakeBackend) Stats(context.Context) (backend.Stats, error) { return backend.Stats{}, f.err }

func (f *fakeBackend) Health(context.Context) error { return f.err }

func reviewModel(be backend.Backend, cards ...srs.Card) Model {
	return Model{
		backend:     be,
		keys:        defaultKeys(),
		mode:        ModeReview,
		reviewDeck:  "Wortschatz",
		reviewCards: cards,
		newCounted:  map[string]bool{},
	}
}

func pressRune(m Model, r rune) (Model, tea.Cmd) {
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	return updated.(Model), cmd
}

func testCard(id, front string) srs.Card {
	return srs.Card{ID: id, Deck: "Wortschatz", Front: front, Back: front + "?", DueAt: time.Now()}
}

func TestBuryTakesTheCardOutOfTheSessionWithoutGradingIt(t *testing.T) {
	be := &fakeBackend{}
	m := reviewModel(be, testCard("a", "Haus"), testCard("b", "Baum"))

	m, cmd := pressRune(m, 'b')
	if len(m.reviewCards) != 1 || m.reviewCards[0].ID != "b" {
		t.Fatalf("bury left %d cards: %+v", len(m.reviewCards), m.reviewCards)
	}
	if len(m.undoStack) != 1 {
		t.Fatalf("undo stack = %d, want 1", len(m.undoStack))
	}
	if len(be.answers) != 0 {
		t.Fatalf("bury graded the card: %v", be.answers)
	}
	if cmd == nil {
		t.Fatal("bury did not ask the backend")
	}
	msg, ok := cmd().(asideMsg)
	if !ok {
		t.Fatalf("unexpected message %T", cmd())
	}
	if msg.err != nil || msg.suspend {
		t.Fatalf("aside message = %+v", msg)
	}
	if len(be.buried) != 1 || be.buried[0] != "a" {
		t.Fatalf("buried = %v", be.buried)
	}
}

func TestSuspendIsItsOwnAction(t *testing.T) {
	be := &fakeBackend{}
	m := reviewModel(be, testCard("a", "Haus"))

	m, cmd := pressRune(m, 'x')
	if len(m.reviewCards) != 0 {
		t.Fatalf("suspended card still in the session: %+v", m.reviewCards)
	}
	if cmd == nil {
		t.Fatal("suspend did not ask the backend")
	}
	msg := cmd().(asideMsg)
	if !msg.suspend {
		t.Fatalf("expected a suspend message, got %+v", msg)
	}
	if len(be.suspended) != 1 || be.suspended[0] != "a" {
		t.Fatalf("suspended = %v", be.suspended)
	}
	if len(be.answers) != 0 {
		t.Fatalf("suspend graded the card: %v", be.answers)
	}
}

func TestUndoRestoresABuriedCardToItsSlot(t *testing.T) {
	be := &fakeBackend{}
	m := reviewModel(be, testCard("a", "Haus"), testCard("b", "Baum"), testCard("c", "Fluss"))

	m, _ = pressRune(m, 'b') // bury the first card
	m, _ = pressRune(m, 'b') // bury what is now the first card
	if len(m.reviewCards) != 1 || m.reviewCards[0].ID != "c" {
		t.Fatalf("queue after two buries: %+v", m.reviewCards)
	}

	m, cmd := pressRune(m, 'u')
	cmd() // undo reaches the backend through the command, not the keystroke
	if len(m.reviewCards) != 2 || m.reviewCards[0].ID != "b" || m.reviewCards[1].ID != "c" {
		t.Fatalf("undo put the card back at the wrong slot: %+v", m.reviewCards)
	}
	if m.reviewIdx != 0 {
		t.Fatalf("reviewIdx = %d, want the restored card", m.reviewIdx)
	}
	if len(be.undos) != 1 || be.undos[0].Card.ID != "b" || be.undos[0].Graded {
		t.Fatalf("undo review = %+v", be.undos)
	}
	if cmd == nil {
		t.Fatal("undo did not ask the backend")
	}
}

func TestUndoAfterAGradePutsTheCardBackAndUncountsIt(t *testing.T) {
	be := &fakeBackend{}
	card := testCard("a", "Haus")
	card.IntervalDays = 1
	m := reviewModel(be, card)
	m.revealed = true

	m, answerCmd := pressRune(m, '3')
	answerCmd()
	if len(be.answers) != 1 {
		t.Fatalf("grades sent = %v", be.answers)
	}
	// The grade only reaches the undo stack once the backend stored it.
	updated, _ := m.Update(answerMsg{cardID: "a"})
	m = updated.(Model)
	if m.reviewIdx != 1 || m.reviewNewToday != 1 {
		t.Fatalf("after grade idx=%d newToday=%d", m.reviewIdx, m.reviewNewToday)
	}
	if len(m.undoStack) != 1 {
		t.Fatalf("undo stack = %d, want 1", len(m.undoStack))
	}

	m, undoCmd := pressRune(m, 'u')
	undoCmd()
	if m.reviewIdx != 0 {
		t.Fatalf("reviewIdx = %d, want the graded card back", m.reviewIdx)
	}
	if m.reviewNewToday != 0 {
		t.Fatalf("new today = %d, want the undo to uncount it", m.reviewNewToday)
	}
	if len(m.newCounted) != 0 {
		t.Fatalf("newCounted still holds %v", m.newCounted)
	}
	if len(be.undos) != 1 || !be.undos[0].Graded {
		t.Fatalf("undo review = %+v", be.undos)
	}
	if be.undos[0].Card.Reps != card.Reps || be.undos[0].Card.IntervalDays != card.IntervalDays {
		t.Fatalf("undo did not carry the pre-grade card: %+v", be.undos[0].Card)
	}
}

func TestUndoWorksAfterTheQueueRunsOut(t *testing.T) {
	be := &fakeBackend{}
	m := reviewModel(be, testCard("a", "Haus"))
	m.revealed = true

	m, _ = pressRune(m, '3')
	updated, _ := m.Update(answerMsg{cardID: "a"})
	m = updated.(Model)
	if m.reviewIdx != 1 || m.reviewIdx != len(m.reviewCards) {
		t.Fatalf("batch not finished: idx=%d len=%d", m.reviewIdx, len(m.reviewCards))
	}

	m, _ = pressRune(m, 'u')
	if m.reviewIdx != 0 || len(m.reviewCards) != 1 {
		t.Fatalf("undo on the finished batch: idx=%d len=%d", m.reviewIdx, len(m.reviewCards))
	}
}

func TestUndoOfAgainDropsTheRequeuedCopy(t *testing.T) {
	be := &fakeBackend{}
	m := reviewModel(be, testCard("a", "Haus"))
	m.revealed = true

	m, _ = pressRune(m, '1')
	updated, _ := m.Update(answerMsg{cardID: "a"})
	m = updated.(Model)
	if len(m.reviewCards) != 2 {
		t.Fatalf("Again should queue the card at the tail: %+v", m.reviewCards)
	}

	m, _ = pressRune(m, 'u')
	if len(m.reviewCards) != 1 || m.reviewCards[0].ID != "a" {
		t.Fatalf("undo left a duplicate: %+v", m.reviewCards)
	}
	if m.reviewIdx != 0 {
		t.Fatalf("reviewIdx = %d", m.reviewIdx)
	}
}

func TestUndoWithNothingToUndoIsQuiet(t *testing.T) {
	be := &fakeBackend{}
	m := reviewModel(be, testCard("a", "Haus"))

	m, cmd := pressRune(m, 'u')
	if cmd != nil {
		t.Fatal("empty undo should not call the backend")
	}
	if len(m.reviewCards) != 1 || m.reviewIdx != 0 {
		t.Fatalf("empty undo changed the queue: %+v", m.reviewCards)
	}
	if m.infoText == "" {
		t.Fatal("empty undo said nothing")
	}
}

func TestNewQueueClearsTheUndoHistory(t *testing.T) {
	be := &fakeBackend{}
	m := reviewModel(be, testCard("a", "Haus"))
	m, _ = pressRune(m, 'b')

	updated, _ := m.Update(queueMsg{queue: backend.Queue{Cards: []srs.Card{testCard("z", "Zeit")}}})
	m = updated.(Model)
	if len(m.undoStack) != 0 {
		t.Fatalf("undo stack survived a new queue: %d", len(m.undoStack))
	}
}
