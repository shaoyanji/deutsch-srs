package backend

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/shaoyanji/deutsch-srs/internal/srs"
)

var queueNow = time.Date(2100, 3, 10, 9, 0, 0, 0, time.UTC)

func newQueueBackend(t *testing.T) *LocalBackend {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "queue.db")
	be, err := NewLocalBackend(dbPath, "")
	if err != nil {
		t.Fatalf("NewLocalBackend: %v", err)
	}
	be.now = func() time.Time { return queueNow }
	t.Cleanup(func() { _ = be.Close() })
	return be
}

// addReviewCard adds a card and answers it in the past, so it is due again now.
func addReviewCard(t *testing.T, be *LocalBackend, deck, front string) srs.Card {
	t.Helper()
	ctx := context.Background()
	card, err := be.AddCard(ctx, AddCardInput{Deck: deck, Front: front, Back: "back"})
	if err != nil {
		t.Fatalf("AddCard: %v", err)
	}
	be.now = func() time.Time { return queueNow.AddDate(0, 0, -10) }
	if err := be.AnswerCard(ctx, card.ID, srs.GradeGood); err != nil {
		t.Fatalf("AnswerCard: %v", err)
	}
	be.now = func() time.Time { return queueNow }
	return card
}

func TestReviewQueuePutsReviewsBeforeNewCards(t *testing.T) {
	ctx := context.Background()
	be := newQueueBackend(t)

	addReviewCard(t, be, "D", "wiederholung-1")
	addReviewCard(t, be, "D", "wiederholung-2")
	for i := 0; i < 3; i++ {
		if _, err := be.AddCard(ctx, AddCardInput{Deck: "D", Front: fmt.Sprintf("neu-%d", i), Back: "x"}); err != nil {
			t.Fatalf("AddCard: %v", err)
		}
	}
	if err := be.SetDeckNewPerDay(ctx, "D", 1); err != nil {
		t.Fatalf("SetDeckNewPerDay: %v", err)
	}

	q, err := be.ReviewQueue(ctx, "D", 50)
	if err != nil {
		t.Fatalf("ReviewQueue: %v", err)
	}
	if len(q.Cards) != 3 {
		t.Fatalf("queue=%d want 3 (2 reviews + 1 new)", len(q.Cards))
	}
	for i := 0; i < 2; i++ {
		if q.Cards[i].Reps == 0 {
			t.Fatalf("card %d should be a review: %+v", i, q.Cards[i])
		}
	}
	if q.Cards[2].Reps != 0 {
		t.Fatalf("last card should be new: %+v", q.Cards[2])
	}
	if q.NewPerDay != 1 || q.NewToday != 0 || q.NewRemaining != 0 {
		t.Fatalf("allowance reported wrong: %+v", q)
	}
}

func TestReviewQueueStopsIntroducingNewCardsAtTheCap(t *testing.T) {
	ctx := context.Background()
	be := newQueueBackend(t)

	for i := 0; i < 5; i++ {
		if _, err := be.AddCard(ctx, AddCardInput{Deck: "D", Front: fmt.Sprintf("neu-%d", i), Back: "x"}); err != nil {
			t.Fatalf("AddCard: %v", err)
		}
	}
	if err := be.SetDeckNewPerDay(ctx, "D", 2); err != nil {
		t.Fatalf("SetDeckNewPerDay: %v", err)
	}

	q, err := be.ReviewQueue(ctx, "D", 50)
	if err != nil {
		t.Fatalf("ReviewQueue: %v", err)
	}
	if len(q.Cards) != 2 {
		t.Fatalf("first batch=%d want 2", len(q.Cards))
	}

	// Answering them uses up the day's allowance.
	for _, c := range q.Cards {
		if err := be.AnswerCard(ctx, c.ID, srs.GradeGood); err != nil {
			t.Fatalf("AnswerCard: %v", err)
		}
	}
	q2, err := be.ReviewQueue(ctx, "D", 50)
	if err != nil {
		t.Fatalf("ReviewQueue: %v", err)
	}
	if len(q2.Cards) != 0 {
		t.Fatalf("second batch=%d want 0 (cap reached)", len(q2.Cards))
	}
	if q2.NewToday != 2 || q2.NewRemaining != 0 {
		t.Fatalf("usage reported wrong: %+v", q2)
	}
}

func TestReviewQueueKeepsReviewsComingWhenCapIsReached(t *testing.T) {
	ctx := context.Background()
	be := newQueueBackend(t)

	for i := 0; i < 4; i++ {
		addReviewCard(t, be, "D", fmt.Sprintf("wiederholung-%d", i))
	}
	if err := be.SetDeckNewPerDay(ctx, "D", 0); err != nil {
		t.Fatalf("SetDeckNewPerDay: %v", err)
	}
	for i := 0; i < 3; i++ {
		if _, err := be.AddCard(ctx, AddCardInput{Deck: "D", Front: fmt.Sprintf("neu-%d", i), Back: "x"}); err != nil {
			t.Fatalf("AddCard: %v", err)
		}
	}

	q, err := be.ReviewQueue(ctx, "D", 50)
	if err != nil {
		t.Fatalf("ReviewQueue: %v", err)
	}
	if len(q.Cards) != 4 {
		t.Fatalf("queue=%d want 4 reviews and no new cards", len(q.Cards))
	}
	for _, c := range q.Cards {
		if c.Reps == 0 {
			t.Fatalf("new card leaked past a zero cap: %+v", c)
		}
	}
}

func TestReviewQueueSplitsIntoBatches(t *testing.T) {
	ctx := context.Background()
	be := newQueueBackend(t)

	for i := 0; i < 7; i++ {
		addReviewCard(t, be, "D", fmt.Sprintf("wiederholung-%d", i))
	}

	q, err := be.ReviewQueue(ctx, "D", 3)
	if err != nil {
		t.Fatalf("ReviewQueue: %v", err)
	}
	if len(q.Cards) != 3 {
		t.Fatalf("batch=%d want 3", len(q.Cards))
	}
}

func TestIntroducedTodayCountsEachCardOnce(t *testing.T) {
	ctx := context.Background()
	be := newQueueBackend(t)

	card, err := be.AddCard(ctx, AddCardInput{Deck: "D", Front: "einmal", Back: "once"})
	if err != nil {
		t.Fatalf("AddCard: %v", err)
	}
	if err := be.AnswerCard(ctx, card.ID, srs.GradeGood); err != nil {
		t.Fatalf("AnswerCard: %v", err)
	}
	if err := be.AnswerCard(ctx, card.ID, srs.GradeGood); err != nil {
		t.Fatalf("AnswerCard: %v", err)
	}

	limits, err := be.DeckLimits(ctx)
	if err != nil {
		t.Fatalf("DeckLimits: %v", err)
	}
	if len(limits) != 1 {
		t.Fatalf("limits=%d want 1", len(limits))
	}
	if limits[0].NewToday != 1 {
		t.Fatalf("NewToday=%d want 1 (a card counts once)", limits[0].NewToday)
	}
	if !limits[0].IsDefault || limits[0].NewPerDay != DefaultNewPerDay {
		t.Fatalf("unexpected default limit: %+v", limits[0])
	}
}

func TestDeckLimitOverridesDefault(t *testing.T) {
	ctx := context.Background()
	be := newQueueBackend(t)
	be.SetDefaultNewPerDay(7)

	if _, err := be.AddCard(ctx, AddCardInput{Deck: "D", Front: "x", Back: "y"}); err != nil {
		t.Fatalf("AddCard: %v", err)
	}
	if err := be.SetDeckNewPerDay(ctx, "D", 3); err != nil {
		t.Fatalf("SetDeckNewPerDay: %v", err)
	}

	limits, err := be.DeckLimits(ctx)
	if err != nil {
		t.Fatalf("DeckLimits: %v", err)
	}
	if limits[0].IsDefault {
		t.Fatal("stored cap reported as default")
	}
	if limits[0].NewPerDay != 3 {
		t.Fatalf("NewPerDay=%d want 3", limits[0].NewPerDay)
	}

	q, err := be.ReviewQueue(ctx, "D", 50)
	if err != nil {
		t.Fatalf("ReviewQueue: %v", err)
	}
	if q.NewPerDay != 3 {
		t.Fatalf("queue used cap %d, want the deck's 3", q.NewPerDay)
	}
}
