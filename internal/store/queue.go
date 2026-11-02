package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/shaoyanji/deutsch-srs/internal/srs"
)

const cardColumns = `id, deck, front, back, note_type, tags, due_at, interval_days, ease, stability_days, difficulty, last_review_at, reps, lapses, suspended, created_at, updated_at`

// DueReviews returns cards the scheduler already knows (reps > 0), oldest due first.
func (s *SQLiteStore) DueReviews(ctx context.Context, deck string, limit int, now time.Time) ([]srs.Card, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+cardColumns+`
		FROM cards
		WHERE deck = ? AND suspended = 0 AND reps > 0 AND due_at <= ?
		ORDER BY due_at ASC, created_at ASC
		LIMIT ?`, deck, now.UTC().Format(time.RFC3339), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCards(rows)
}

// NewCards returns cards that have never been reviewed, oldest first.
func (s *SQLiteStore) NewCards(ctx context.Context, deck string, limit int, now time.Time) ([]srs.Card, error) {
	if limit <= 0 {
		return nil, nil
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+cardColumns+`
		FROM cards
		WHERE deck = ? AND suspended = 0 AND reps = 0 AND due_at <= ?
		ORDER BY created_at ASC, id ASC
		LIMIT ?`, deck, now.UTC().Format(time.RFC3339), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCards(rows)
}

// IntroducedToday counts the cards in a deck whose first-ever review happened
// since the given time. A card counts once, however often it was answered.
func (s *SQLiteStore) IntroducedToday(ctx context.Context, deck string, since time.Time) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM (
			SELECT card_id, MIN(answered_at) AS first_at
			FROM review_log
			GROUP BY card_id
		) first_review
		JOIN cards c ON c.id = first_review.card_id
		WHERE c.deck = ? AND first_review.first_at >= ?`,
		deck, since.UTC().Format(time.RFC3339)).Scan(&n)
	return n, err
}

// DeckNewPerDay reports the stored daily new-card cap for a deck. ok is false
// when the deck has no setting and the app default applies.
func (s *SQLiteStore) DeckNewPerDay(ctx context.Context, deck string) (int, bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT new_per_day FROM deck_settings WHERE deck = ?`, deck).Scan(&n)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return n, true, nil
}

// SetDeckNewPerDay stores the cap. Zero is a valid value: no new cards at all.
func (s *SQLiteStore) SetDeckNewPerDay(ctx context.Context, deck string, n int, now time.Time) error {
	if n < 0 {
		return fmt.Errorf("new cards per day must be zero or more")
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO deck_settings (deck, new_per_day, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(deck) DO UPDATE SET new_per_day = excluded.new_per_day, updated_at = excluded.updated_at`,
		deck, n, now.UTC().Format(time.RFC3339))
	return err
}

// DeckNewPerDayAll returns every stored cap, keyed by deck name.
func (s *SQLiteStore) DeckNewPerDayAll(ctx context.Context) (map[string]int, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT deck, new_per_day FROM deck_settings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]int{}
	for rows.Next() {
		var deck string
		var n int
		if err := rows.Scan(&deck, &n); err != nil {
			return nil, err
		}
		out[deck] = n
	}
	return out, rows.Err()
}
