package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/shaoyanji/deutsch-srs/internal/srs"
)

// ErrCardNotFound is what the single-card mutators return when the id matches
// no row, so a wrong id is never mistaken for a successful no-op.
var ErrCardNotFound = errors.New("card not found")

// SetCardSuspended hides a card from every queue and from the due counts, or
// brings it back. Suspending is not a grade: no schedule field moves.
func (s *SQLiteStore) SetCardSuspended(ctx context.Context, id string, suspended bool, now time.Time) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE cards SET suspended = ?, updated_at = ? WHERE id = ?`,
		boolFlag(suspended), now.UTC().Format(time.RFC3339), id)
	if err != nil {
		return err
	}
	return affectedCard(res, id)
}

// SetCardDue moves one card's due date without touching its schedule. That is
// what burying means here: the card leaves today's queue, but what the learner's
// answers taught the scheduler (interval, ease, reps) stays exactly as it was.
func (s *SQLiteStore) SetCardDue(ctx context.Context, id string, due, now time.Time) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE cards SET due_at = ?, updated_at = ? WHERE id = ?`,
		due.UTC().Format(time.RFC3339), now.UTC().Format(time.RFC3339), id)
	if err != nil {
		return err
	}
	return affectedCard(res, id)
}

// RestoreCard puts back the review state a card had before an action. Undo is
// a single statement because every action in the review flow changes only these
// columns; a compensating operation per action kind would drift from them.
func (s *SQLiteStore) RestoreCard(ctx context.Context, card srs.Card) error {
	// The column default is the empty string, not a zero timestamp, and the
	// legacy backfill keys on that. Writing year 1 here would make an undone
	// new card look like a reviewed one.
	lastReview := ""
	if !card.LastReviewAt.IsZero() {
		lastReview = card.LastReviewAt.UTC().Format(time.RFC3339)
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE cards SET
			due_at = ?, interval_days = ?, ease = ?, stability_days = ?, difficulty = ?,
			last_review_at = ?, reps = ?, lapses = ?, suspended = ?, updated_at = ?
		WHERE id = ?`,
		card.DueAt.UTC().Format(time.RFC3339), card.IntervalDays, card.Ease,
		card.StabilityDays, card.Difficulty, lastReview,
		card.Reps, card.Lapses, boolFlag(card.Suspended),
		card.UpdatedAt.UTC().Format(time.RFC3339), card.ID)
	if err != nil {
		return err
	}
	return affectedCard(res, card.ID)
}

// DeleteLastReviewLog removes the newest log row for a card. Undo only ever
// reverts the action that just happened, so the newest row is the right one
// even when the card has years of history behind it.
func (s *SQLiteStore) DeleteLastReviewLog(ctx context.Context, cardID string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM review_log WHERE id = (SELECT MAX(id) FROM review_log WHERE card_id = ?)`,
		cardID)
	return err
}

func affectedCard(res sql.Result, id string) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("%w: %s", ErrCardNotFound, id)
	}
	return nil
}

func boolFlag(b bool) int {
	if b {
		return 1
	}
	return 0
}
