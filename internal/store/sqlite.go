package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/shaoyanji/deutsch-srs/internal/srs"
)

type AddCardInput struct {
	Deck     string
	Front    string
	Back     string
	NoteType string
	Tags     []string
}

type Stats struct {
	TotalCards   int
	DueToday     int
	NewCards     int
	ReviewsToday int
}

type SQLiteStore struct {
	db *sql.DB
}

func Open(path string) (*SQLiteStore, error) {
	// A first run against the CLI's default path (~/.local/share/...) has no
	// directory yet, and sqlite reports that as a bare "unable to open database
	// file". Create it here so every caller gets the same behaviour.
	if path != "" && path != ":memory:" {
		if dir := filepath.Dir(path); dir != "." && dir != "" {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("create database directory: %w", err)
			}
		}
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	store := &SQLiteStore{db: db}
	if err := store.Migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteStore) Migrate(ctx context.Context) error {
	for _, stmt := range migrations {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	if err := s.ensureCardSchedulerColumns(ctx); err != nil {
		return fmt.Errorf("migrate scheduler columns: %w", err)
	}
	if err := s.backfillCardSchedulerState(ctx, time.Now().UTC()); err != nil {
		return fmt.Errorf("backfill scheduler state: %w", err)
	}
	if err := s.ensureSearchIndex(ctx); err != nil {
		return fmt.Errorf("migrate search index: %w", err)
	}
	return nil
}

func (s *SQLiteStore) Health(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *SQLiteStore) CardCount(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM cards`).Scan(&n)
	return n, err
}

func (s *SQLiteStore) AddCard(ctx context.Context, in AddCardInput, now time.Time) (srs.Card, error) {
	card, err := newCard(in, now)
	if err != nil {
		return srs.Card{}, err
	}
	if _, err := s.db.ExecContext(ctx, insertCardSQL, cardArgs(card)...); err != nil {
		return srs.Card{}, err
	}
	return card, nil
}

func (s *SQLiteStore) ListDecks(ctx context.Context, now time.Time) ([]srs.Deck, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT deck,
			SUM(CASE WHEN suspended = 0 AND due_at <= ? THEN 1 ELSE 0 END) AS due_count,
			SUM(CASE WHEN suspended = 0 AND reps = 0 THEN 1 ELSE 0 END) AS new_count,
			SUM(CASE WHEN suspended = 0 AND reps > 0 THEN 1 ELSE 0 END) AS review_count
		FROM cards
		GROUP BY deck
		ORDER BY deck ASC`, now.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var decks []srs.Deck
	for rows.Next() {
		var d srs.Deck
		if err := rows.Scan(&d.Name, &d.DueCount, &d.NewCount, &d.ReviewCount); err != nil {
			return nil, err
		}
		decks = append(decks, d)
	}
	return decks, rows.Err()
}

func (s *SQLiteStore) DueCards(ctx context.Context, deck string, limit int, now time.Time) ([]srs.Card, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, deck, front, back, note_type, tags, due_at, interval_days, ease, stability_days, difficulty, last_review_at, reps, lapses, suspended, created_at, updated_at
		FROM cards
		WHERE deck = ? AND suspended = 0 AND due_at <= ?
		ORDER BY due_at ASC, created_at ASC
		LIMIT ?`, deck, now.UTC().Format(time.RFC3339), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCards(rows)
}

func (s *SQLiteStore) AllCards(ctx context.Context) ([]srs.Card, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, deck, front, back, note_type, tags, due_at, interval_days, ease, stability_days, difficulty, last_review_at, reps, lapses, suspended, created_at, updated_at
		FROM cards
		ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCards(rows)
}

func (s *SQLiteStore) GetCard(ctx context.Context, id string) (srs.Card, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, deck, front, back, note_type, tags, due_at, interval_days, ease, stability_days, difficulty, last_review_at, reps, lapses, suspended, created_at, updated_at
		FROM cards WHERE id = ?`, id)
	card, err := scanCard(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return srs.Card{}, fmt.Errorf("card not found: %s", id)
		}
		return srs.Card{}, err
	}
	return card, nil
}

func (s *SQLiteStore) UpdateCardSchedule(ctx context.Context, card srs.Card) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE cards SET
			due_at = ?, interval_days = ?, ease = ?, stability_days = ?, difficulty = ?, last_review_at = ?, reps = ?, lapses = ?, updated_at = ?
		WHERE id = ?`,
		card.DueAt.UTC().Format(time.RFC3339), card.IntervalDays, card.Ease, card.StabilityDays, card.Difficulty, card.LastReviewAt.UTC().Format(time.RFC3339),
		card.Reps, card.Lapses, card.UpdatedAt.UTC().Format(time.RFC3339), card.ID,
	)
	return err
}

func (s *SQLiteStore) InsertReviewLog(ctx context.Context, cardID string, grade srs.ReviewGrade, answeredAt time.Time, intervalDays int, ease float64) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO review_log (card_id, grade, answered_at, interval_days, ease)
		VALUES (?, ?, ?, ?, ?)`,
		cardID, int(grade), answeredAt.UTC().Format(time.RFC3339), intervalDays, ease,
	)
	return err
}

func (s *SQLiteStore) Stats(ctx context.Context, now time.Time) (Stats, error) {
	var out Stats
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM cards`).Scan(&out.TotalCards); err != nil {
		return Stats{}, err
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM cards WHERE suspended = 0 AND due_at <= ?`, now.UTC().Format(time.RFC3339)).Scan(&out.DueToday); err != nil {
		return Stats{}, err
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM cards WHERE suspended = 0 AND reps = 0`).Scan(&out.NewCards); err != nil {
		return Stats{}, err
	}
	dayStart := time.Date(now.UTC().Year(), now.UTC().Month(), now.UTC().Day(), 0, 0, 0, 0, time.UTC)
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM review_log WHERE answered_at >= ?`, dayStart.Format(time.RFC3339)).Scan(&out.ReviewsToday); err != nil {
		return Stats{}, err
	}
	return out, nil
}

func scanCards(rows *sql.Rows) ([]srs.Card, error) {
	var out []srs.Card
	for rows.Next() {
		card, err := scanCard(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, card)
	}
	return out, rows.Err()
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanCard(scanner rowScanner) (srs.Card, error) {
	var (
		card            srs.Card
		dueAt, created  string
		updatedAt, tags string
		lastReviewAt    string
		suspended       int
	)
	if err := scanner.Scan(
		&card.ID, &card.Deck, &card.Front, &card.Back, &card.NoteType, &tags, &dueAt, &card.IntervalDays, &card.Ease,
		&card.StabilityDays, &card.Difficulty, &lastReviewAt, &card.Reps, &card.Lapses, &suspended, &created, &updatedAt,
	); err != nil {
		return srs.Card{}, err
	}
	d, err := time.Parse(time.RFC3339, dueAt)
	if err != nil {
		return srs.Card{}, err
	}
	c, err := time.Parse(time.RFC3339, created)
	if err != nil {
		return srs.Card{}, err
	}
	u, err := time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		return srs.Card{}, err
	}
	card.DueAt = d
	card.CreatedAt = c
	card.UpdatedAt = u
	if lastReviewAt != "" {
		if lr, err := time.Parse(time.RFC3339, lastReviewAt); err == nil {
			card.LastReviewAt = lr
		}
	}
	card.Suspended = suspended == 1
	if tags != "" {
		card.Tags = strings.Split(tags, ",")
	}
	return card, nil
}

func (s *SQLiteStore) ensureCardSchedulerColumns(ctx context.Context) error {
	type col struct {
		name string
		sql  string
	}
	required := []col{
		{name: "stability_days", sql: `ALTER TABLE cards ADD COLUMN stability_days REAL NOT NULL DEFAULT 1.0`},
		{name: "difficulty", sql: `ALTER TABLE cards ADD COLUMN difficulty REAL NOT NULL DEFAULT 0.35`},
		{name: "last_review_at", sql: `ALTER TABLE cards ADD COLUMN last_review_at TEXT NOT NULL DEFAULT ''`},
	}
	existing := map[string]bool{}
	rows, err := s.db.QueryContext(ctx, `PRAGMA table_info(cards)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notNull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notNull, &dflt, &pk); err != nil {
			return err
		}
		existing[name] = true
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, c := range required {
		if existing[c.name] {
			continue
		}
		if _, err := s.db.ExecContext(ctx, c.sql); err != nil {
			return err
		}
	}
	return nil
}

// schedulerBackfillMarker records that the legacy scheduler columns have been
// backfilled for this database. The migration is a one-off, so without a marker
// every open pays for it again.
const schedulerBackfillMarker = "scheduler_backfill_at"

func (s *SQLiteStore) backfillCardSchedulerState(ctx context.Context, now time.Time) error {
	var done string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM schema_meta WHERE key = ?`, schedulerBackfillMarker).Scan(&done)
	switch {
	case err == nil:
		return nil
	case !errors.Is(err, sql.ErrNoRows):
		return fmt.Errorf("read scheduler backfill marker: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin scheduler backfill: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Only rows the migration actually changes are read and rewritten. Reading
	// every card and then running one autocommit UPDATE per row cost 2.3 s of
	// startup on a 77k-card deck to write back the values it had just read.
	rows, err := tx.QueryContext(ctx, `
		SELECT id, interval_days, ease, stability_days, difficulty, last_review_at, updated_at, created_at
		FROM cards
		WHERE stability_days <= 0
			OR difficulty < 0
			OR difficulty > 1
			OR LENGTH(TRIM(last_review_at)) = 0`)
	if err != nil {
		return err
	}

	type rowData struct {
		id           string
		intervalDays int
		ease         float64
		stability    float64
		difficulty   float64
		lastReview   string
		updatedAt    string
		createdAt    string
	}
	var stale []rowData
	for rows.Next() {
		var r rowData
		if err := rows.Scan(&r.id, &r.intervalDays, &r.ease, &r.stability, &r.difficulty, &r.lastReview, &r.updatedAt, &r.createdAt); err != nil {
			rows.Close()
			return err
		}
		stale = append(stale, r)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	if len(stale) > 0 {
		stmt, err := tx.PrepareContext(ctx, `
			UPDATE cards
			SET stability_days = ?, difficulty = ?, last_review_at = ?
			WHERE id = ?`)
		if err != nil {
			return fmt.Errorf("prepare scheduler backfill: %w", err)
		}
		defer stmt.Close()

		fallback := now.UTC().Format(time.RFC3339)
		for _, r := range stale {
			stability := r.stability
			if stability <= 0 {
				if r.intervalDays > 0 {
					stability = float64(r.intervalDays)
				} else {
					stability = srs.DefaultStabilityDays
				}
			}

			difficulty := r.difficulty
			if difficulty < 0 || difficulty > 1 {
				difficulty = deriveDifficultyFromEase(r.ease)
			}

			lastReviewAt := strings.TrimSpace(r.lastReview)
			if lastReviewAt == "" {
				if updated := strings.TrimSpace(r.updatedAt); updated != "" {
					lastReviewAt = updated
				} else if created := strings.TrimSpace(r.createdAt); created != "" {
					lastReviewAt = created
				} else {
					lastReviewAt = fallback
				}
			}

			if _, err := stmt.ExecContext(ctx, stability, difficulty, lastReviewAt, r.id); err != nil {
				return fmt.Errorf("backfill card %s: %w", r.id, err)
			}
		}
	}

	if _, err := tx.ExecContext(ctx, `INSERT OR REPLACE INTO schema_meta (key, value) VALUES (?, ?)`,
		schedulerBackfillMarker, time.Now().UTC().Format(time.RFC3339)); err != nil {
		return fmt.Errorf("record scheduler backfill: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit scheduler backfill: %w", err)
	}
	return nil
}

func deriveDifficultyFromEase(ease float64) float64 {
	if ease <= 0 {
		return srs.DefaultDifficulty
	}
	d := 0.35 + ((2.5 - ease) * 0.35)
	if d < 0 {
		return 0
	}
	if d > 1 {
		return 1
	}
	return math.Round(d*1000) / 1000
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
