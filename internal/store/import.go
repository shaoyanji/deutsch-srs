package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/shaoyanji/deutsch-srs/internal/srs"
)

const insertCardSQL = `INSERT INTO cards (
		id, deck, front, back, note_type, tags, due_at, interval_days, ease, stability_days, difficulty, last_review_at, reps, lapses, suspended, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

// BulkOptions controls a batched import.
type BulkOptions struct {
	SkipExisting bool
	DryRun       bool
}

// BulkResult reports what an import did.
type BulkResult struct {
	Added   int
	Skipped int
	Invalid int
}

func newCard(in AddCardInput, now time.Time) (srs.Card, error) {
	if strings.TrimSpace(in.Deck) == "" {
		return srs.Card{}, fmt.Errorf("deck is required")
	}
	if strings.TrimSpace(in.Front) == "" || strings.TrimSpace(in.Back) == "" {
		return srs.Card{}, fmt.Errorf("front and back are required")
	}
	ts := now.UTC()
	return srs.Card{
		ID:            uuid.NewString(),
		Deck:          in.Deck,
		Front:         in.Front,
		Back:          in.Back,
		NoteType:      in.NoteType,
		Tags:          in.Tags,
		DueAt:         ts,
		IntervalDays:  0,
		Ease:          2.5,
		StabilityDays: srs.DefaultStabilityDays,
		Difficulty:    srs.DefaultDifficulty,
		LastReviewAt:  ts,
		Reps:          0,
		Lapses:        0,
		Suspended:     false,
		CreatedAt:     ts,
		UpdatedAt:     ts,
	}, nil
}

func cardArgs(c srs.Card) []any {
	return []any{
		c.ID, c.Deck, c.Front, c.Back, c.NoteType, strings.Join(c.Tags, ","),
		c.DueAt.UTC().Format(time.RFC3339), c.IntervalDays, c.Ease, c.StabilityDays, c.Difficulty,
		c.LastReviewAt.UTC().Format(time.RFC3339), c.Reps, c.Lapses, boolToInt(c.Suspended),
		c.CreatedAt.UTC().Format(time.RFC3339), c.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

// deckKey is the identity of a card for import purposes: same deck, same front.
func deckKey(deck, front string) string {
	return strings.ToLower(strings.TrimSpace(deck)) + "\x00" + strings.ToLower(strings.TrimSpace(front))
}

// AddCardsBulk inserts every card in one transaction. With SkipExisting, a card
// whose (deck, front) pair is already stored is skipped, so re-running the same
// import is a no-op instead of a duplicate deck. Rows repeated inside one batch
// are collapsed the same way.
func (s *SQLiteStore) AddCardsBulk(ctx context.Context, cards []AddCardInput, now time.Time, opts BulkOptions) (BulkResult, error) {
	var res BulkResult
	if len(cards) == 0 {
		return res, nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return res, fmt.Errorf("begin import: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	seen := make(map[string]struct{}, len(cards))
	if opts.SkipExisting {
		// Compare with the same Unicode-aware folding used for incoming rows;
		// SQLite's LOWER() only folds ASCII, so "Öl" would never match "öl".
		rows, err := tx.QueryContext(ctx, `SELECT deck, front FROM cards`)
		if err != nil {
			return res, fmt.Errorf("read existing cards: %w", err)
		}
		for rows.Next() {
			var deck, front string
			if err := rows.Scan(&deck, &front); err != nil {
				rows.Close()
				return res, err
			}
			seen[deckKey(deck, front)] = struct{}{}
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return res, err
		}
		rows.Close()
	}

	stmt, err := tx.PrepareContext(ctx, insertCardSQL)
	if err != nil {
		return res, fmt.Errorf("prepare import: %w", err)
	}
	defer stmt.Close()

	ts := now.UTC()
	for _, in := range cards {
		card, err := newCard(in, ts)
		if err != nil {
			res.Invalid++
			continue
		}
		key := deckKey(card.Deck, card.Front)
		if _, dup := seen[key]; dup {
			res.Skipped++
			continue
		}
		seen[key] = struct{}{}
		if _, err := stmt.ExecContext(ctx, cardArgs(card)...); err != nil {
			return res, fmt.Errorf("insert %q: %w", card.Front, err)
		}
		res.Added++
	}

	if opts.DryRun {
		return res, nil
	}
	if err := tx.Commit(); err != nil {
		return res, fmt.Errorf("commit import: %w", err)
	}
	return res, nil
}
