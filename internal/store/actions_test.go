package store

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestSuspendHidesCardFromEveryQueue(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	s := openTestStore(t)

	card, err := s.AddCard(ctx, AddCardInput{Deck: "Starter", Front: "Haus", Back: "house"}, now)
	if err != nil {
		t.Fatalf("AddCard: %v", err)
	}
	if err := s.SetCardSuspended(ctx, card.ID, true, now); err != nil {
		t.Fatalf("SetCardSuspended: %v", err)
	}

	due, err := s.DueCards(ctx, "Starter", 10, now)
	if err != nil {
		t.Fatalf("DueCards: %v", err)
	}
	if len(due) != 0 {
		t.Fatalf("suspended card still due: %+v", due)
	}
	newCards, err := s.NewCards(ctx, "Starter", 10, now)
	if err != nil {
		t.Fatalf("NewCards: %v", err)
	}
	if len(newCards) != 0 {
		t.Fatalf("suspended card still offered as new: %+v", newCards)
	}
	stats, err := s.Stats(ctx, now)
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.DueToday != 0 || stats.NewCards != 0 {
		t.Fatalf("suspended card counted: %+v", stats)
	}

	if err := s.SetCardSuspended(ctx, card.ID, false, now); err != nil {
		t.Fatalf("unsuspend: %v", err)
	}
	due, err = s.DueCards(ctx, "Starter", 10, now)
	if err != nil {
		t.Fatalf("DueCards after unsuspend: %v", err)
	}
	if len(due) != 1 {
		t.Fatalf("unsuspended card missing from due list: %d", len(due))
	}
}

func TestSuspendUnknownCardIsAnError(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	s := openTestStore(t)

	if err := s.SetCardSuspended(ctx, "nope", true, now); !errors.Is(err, ErrCardNotFound) {
		t.Fatalf("expected ErrCardNotFound, got %v", err)
	}
}

func TestBuryPushesDueWithoutTouchingTheSchedule(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	s := openTestStore(t)

	card, err := s.AddCard(ctx, AddCardInput{Deck: "Starter", Front: "Haus", Back: "house"}, now)
	if err != nil {
		t.Fatalf("AddCard: %v", err)
	}
	card.IntervalDays = 4
	card.Ease = 2.6
	card.Reps = 3
	card.LastReviewAt = now.Add(-48 * time.Hour)
	if err := s.UpdateCardSchedule(ctx, card); err != nil {
		t.Fatalf("UpdateCardSchedule: %v", err)
	}

	if err := s.SetCardDue(ctx, card.ID, now.Add(24*time.Hour), now); err != nil {
		t.Fatalf("SetCardDue: %v", err)
	}
	got, err := s.GetCard(ctx, card.ID)
	if err != nil {
		t.Fatalf("GetCard: %v", err)
	}
	if !got.DueAt.After(now) {
		t.Fatalf("buried card still due now: %v", got.DueAt)
	}
	if got.IntervalDays != 4 || got.Ease != 2.6 || got.Reps != 3 {
		t.Fatalf("bury changed the schedule: %+v", got)
	}
	due, err := s.DueCards(ctx, "Starter", 10, now)
	if err != nil {
		t.Fatalf("DueCards: %v", err)
	}
	if len(due) != 0 {
		t.Fatalf("buried card still in today's queue: %+v", due)
	}
}

func TestRestoreCardRevertsAGradeAndItsLogRow(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	s := openTestStore(t)

	before, err := s.AddCard(ctx, AddCardInput{Deck: "Starter", Front: "Haus", Back: "house"}, now)
	if err != nil {
		t.Fatalf("AddCard: %v", err)
	}

	graded := before
	graded.Reps = 1
	graded.IntervalDays = 3
	graded.Ease = 2.5
	graded.LastReviewAt = now
	graded.DueAt = now.Add(72 * time.Hour)
	graded.UpdatedAt = now
	if err := s.UpdateCardSchedule(ctx, graded); err != nil {
		t.Fatalf("UpdateCardSchedule: %v", err)
	}
	if err := s.InsertReviewLog(ctx, before.ID, 3, now, 3, 2.5); err != nil {
		t.Fatalf("InsertReviewLog: %v", err)
	}

	if err := s.RestoreCard(ctx, before); err != nil {
		t.Fatalf("RestoreCard: %v", err)
	}
	if err := s.DeleteLastReviewLog(ctx, before.ID); err != nil {
		t.Fatalf("DeleteLastReviewLog: %v", err)
	}

	got, err := s.GetCard(ctx, before.ID)
	if err != nil {
		t.Fatalf("GetCard: %v", err)
	}
	if got.Reps != before.Reps || got.IntervalDays != before.IntervalDays || !got.LastReviewAt.Equal(before.LastReviewAt) {
		t.Fatalf("restore left the card reviewed: %+v", got)
	}
	if !got.DueAt.Equal(before.DueAt) {
		t.Fatalf("due at %v, want %v", got.DueAt, before.DueAt)
	}

	stats, err := s.Stats(ctx, now)
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.ReviewsToday != 0 {
		t.Fatalf("reviews today = %d, want the undone review gone", stats.ReviewsToday)
	}
}

func TestDeleteLastReviewLogKeepsEarlierHistory(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	s := openTestStore(t)

	card, err := s.AddCard(ctx, AddCardInput{Deck: "Starter", Front: "Haus", Back: "house"}, now)
	if err != nil {
		t.Fatalf("AddCard: %v", err)
	}
	for i := 0; i < 3; i++ {
		if err := s.InsertReviewLog(ctx, card.ID, 3, now.Add(time.Duration(i)*time.Hour), 1, 2.5); err != nil {
			t.Fatalf("InsertReviewLog %d: %v", i, err)
		}
	}
	if err := s.DeleteLastReviewLog(ctx, card.ID); err != nil {
		t.Fatalf("DeleteLastReviewLog: %v", err)
	}
	var n int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM review_log WHERE card_id = ?`, card.ID).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 2 {
		t.Fatalf("review_log rows = %d, want 2", n)
	}
}

func TestDeleteLastReviewLogOnCardWithNoHistoryIsQuiet(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	s := openTestStore(t)

	card, err := s.AddCard(ctx, AddCardInput{Deck: "Starter", Front: "Haus", Back: "house"}, now)
	if err != nil {
		t.Fatalf("AddCard: %v", err)
	}
	if err := s.DeleteLastReviewLog(ctx, card.ID); err != nil {
		t.Fatalf("DeleteLastReviewLog on a never-logged card: %v", err)
	}
}
