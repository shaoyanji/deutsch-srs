package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/shaoyanji/deutsch-srs/internal/srs"
)

// searchIndexMarker records that the index has been populated once for this
// database. Without it every open would rebuild the whole index (seconds on a
// 77k-card deck) to recover from a state only an upgrade can produce.
const searchIndexMarker = "cards_fts_built_at"

// searchCardsSQL reads the FTS index instead of scanning cards. The bm25 weights
// follow the indexed column order: front, back, tags, deck.
const searchCardsSQL = `
	SELECT c.id, c.deck, c.front, c.back, c.note_type, c.tags, c.due_at, c.interval_days, c.ease,
		c.stability_days, c.difficulty, c.last_review_at, c.reps, c.lapses, c.suspended, c.created_at, c.updated_at
	FROM cards_fts
	JOIN cards c ON c.rowid = cards_fts.rowid
	WHERE cards_fts MATCH ?
	ORDER BY bm25(cards_fts, 4.0, 2.0, 1.0, 0.5), c.updated_at DESC
	LIMIT ?`

// SearchCards answers a query from the FTS5 index, so the cost is the same on a
// 200-card deck and a 77k-card one, where the LIKE '%q%' scan it replaced read
// every row.
//
// It runs two passes: whole-word hits for the query, then prefix hits. bm25
// cannot separate "Haus" from "Hausschlüssel" for the query "haus" — both are one
// token and score the same — so the exact pass is what puts the word the learner
// typed at the top instead of a compound.
func (s *SQLiteStore) SearchCards(ctx context.Context, query string, limit int) ([]srs.Card, error) {
	if limit <= 0 {
		limit = 20
	}
	tokens := ftsTokens(query)
	if len(tokens) == 0 {
		return nil, nil
	}

	out := make([]srs.Card, 0, limit)
	seen := make(map[string]struct{}, limit)
	for _, match := range []string{ftsPhrase(tokens), ftsPrefixes(tokens)} {
		if len(out) >= limit {
			break
		}
		cards, err := s.searchMatch(ctx, match, limit)
		if err != nil {
			return nil, err
		}
		for _, card := range cards {
			if _, dup := seen[card.ID]; dup {
				continue
			}
			seen[card.ID] = struct{}{}
			out = append(out, card)
			if len(out) == limit {
				break
			}
		}
	}
	return out, nil
}

func (s *SQLiteStore) searchMatch(ctx context.Context, match string, limit int) ([]srs.Card, error) {
	rows, err := s.db.QueryContext(ctx, searchCardsSQL, match, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCards(rows)
}

// ftsTokens splits user input into searchable words. Only letters and digits are
// kept, so a quote, a star or an FTS operator typed by the user is dropped
// instead of changing the shape of the query.
func ftsTokens(raw string) []string {
	return strings.FieldsFunc(raw, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

// ftsPhrase matches the tokens as one phrase, i.e. adjacent words in order.
func ftsPhrase(tokens []string) string {
	return `"` + strings.Join(tokens, " ") + `"`
}

// ftsPrefixes matches every token as a prefix and requires all of them on the
// same card, so "hund" finds Hund and Hundehütte and "ich lern" still needs both.
func ftsPrefixes(tokens []string) string {
	quoted := make([]string, len(tokens))
	for i, tok := range tokens {
		quoted[i] = `"` + tok + `"*`
	}
	return strings.Join(quoted, " ")
}

// ensureSearchIndex fills the index for a database created before it existed.
// Triggers keep it current from then on, so this runs once per database.
func (s *SQLiteStore) ensureSearchIndex(ctx context.Context) error {
	var builtAt string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM schema_meta WHERE key = ?`, searchIndexMarker).Scan(&builtAt)
	switch {
	case err == nil:
		return nil
	case !errors.Is(err, sql.ErrNoRows):
		return fmt.Errorf("read search index marker: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO cards_fts(cards_fts) VALUES('rebuild')`); err != nil {
		return fmt.Errorf("build search index: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `INSERT OR REPLACE INTO schema_meta (key, value) VALUES (?, ?)`,
		searchIndexMarker, time.Now().UTC().Format(time.RFC3339)); err != nil {
		return fmt.Errorf("record search index build: %w", err)
	}
	return nil
}
