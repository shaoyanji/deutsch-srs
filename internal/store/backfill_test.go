package store

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestReopenDoesNotRewriteCards(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	dbPath := filepath.Join(t.TempDir(), "reopen.db")

	s, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	bulk := make([]AddCardInput, 0, 50)
	for i := 0; i < 50; i++ {
		bulk = append(bulk, AddCardInput{Deck: "Wortschatz", Front: fmt.Sprintf("%c-wort-%d", 'a'+i%26, i), Back: "word"})
	}
	if _, err := s.AddCardsBulk(ctx, bulk, now, BulkOptions{}); err != nil {
		t.Fatalf("bulk add: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// Count every UPDATE the next open performs on cards.
	raw, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open raw: %v", err)
	}
	if _, err := raw.Exec(`
		CREATE TABLE update_probe (n INTEGER NOT NULL);
		INSERT INTO update_probe (n) VALUES (0);
		CREATE TRIGGER probe_card_updates AFTER UPDATE ON cards BEGIN
			UPDATE update_probe SET n = n + 1;
		END;`); err != nil {
		raw.Close()
		t.Fatalf("install probe: %v", err)
	}
	raw.Close()

	reopened, err := Open(dbPath)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer reopened.Close()

	var updates int
	if err := reopened.db.QueryRowContext(ctx, `SELECT n FROM update_probe`).Scan(&updates); err != nil {
		t.Fatalf("read probe: %v", err)
	}
	// Opening a database whose cards are already correct must write nothing: the
	// backfill used to UPDATE every row in autocommit, one transaction per card.
	if updates != 0 {
		t.Fatalf("reopen rewrote %d cards, want 0", updates)
	}
}

func TestBackfillRecordsItsMarker(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)

	var marker int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_meta WHERE key = ?`, schedulerBackfillMarker).Scan(&marker); err != nil {
		t.Fatalf("read marker: %v", err)
	}
	if marker != 1 {
		t.Fatalf("scheduler backfill marker missing: %d", marker)
	}
}

func TestBackfillOnlyFixesStaleRows(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "stale.db")

	legacy, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open legacy: %v", err)
	}
	_, err = legacy.Exec(`CREATE TABLE cards (
		id TEXT PRIMARY KEY,
		deck TEXT NOT NULL,
		front TEXT NOT NULL,
		back TEXT NOT NULL,
		note_type TEXT NOT NULL DEFAULT '',
		tags TEXT NOT NULL DEFAULT '',
		due_at TEXT NOT NULL,
		interval_days INTEGER NOT NULL DEFAULT 0,
		ease REAL NOT NULL DEFAULT 2.5,
		reps INTEGER NOT NULL DEFAULT 0,
		lapses INTEGER NOT NULL DEFAULT 0,
		suspended INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);`)
	if err != nil {
		legacy.Close()
		t.Fatalf("create legacy schema: %v", err)
	}
	// A legacy row has no scheduler columns at all, so its last_review_at is
	// empty after the ALTER TABLE and must be filled from updated_at.
	_, err = legacy.Exec(`INSERT INTO cards (id, deck, front, back, note_type, tags, due_at,
		interval_days, ease, reps, lapses, suspended, created_at, updated_at)
		VALUES ('legacy-1', 'Alt', 'Wasser', 'water', '', '', ?, 5, 2.5, 2, 0, 0, ?, ?)`,
		"2026-01-02T10:00:00Z", "2025-06-01T08:00:00Z", "2025-06-02T08:00:00Z")
	if err != nil {
		legacy.Close()
		t.Fatalf("insert legacy card: %v", err)
	}
	legacy.Close()

	s, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer s.Close()

	card, err := s.GetCard(ctx, "legacy-1")
	if err != nil {
		t.Fatalf("GetCard: %v", err)
	}
	if card.LastReviewAt.Format(time.RFC3339) != "2025-06-02T08:00:00Z" {
		t.Fatalf("last_review_at = %s, want the updated_at fallback", card.LastReviewAt.Format(time.RFC3339))
	}
	// ADD COLUMN gives existing rows its default, so stability is already valid
	// and keeps 1.0 rather than being derived from the 5-day interval. That is the
	// behaviour this migration has always had; the test pins it so a rewrite of the
	// update logic cannot quietly start re-deriving live scheduler state.
	if card.StabilityDays != 1 {
		t.Fatalf("stability_days = %v, want the column default (1)", card.StabilityDays)
	}
}
