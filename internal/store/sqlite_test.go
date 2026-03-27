package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func openTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() {
		_ = s.Close()
	})
	return s
}

func TestAddAndDueCards(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	s := openTestStore(t)

	_, err := s.AddCard(ctx, AddCardInput{Deck: "Starter", Front: "Guten Morgen", Back: "Good morning"}, now)
	if err != nil {
		t.Fatalf("AddCard: %v", err)
	}
	due, err := s.DueCards(ctx, "Starter", 10, now)
	if err != nil {
		t.Fatalf("DueCards: %v", err)
	}
	if len(due) != 1 {
		t.Fatalf("len(due)=%d", len(due))
	}
}

func TestListDecksAndStats(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	s := openTestStore(t)

	_, _ = s.AddCard(ctx, AddCardInput{Deck: "A", Front: "eins", Back: "one"}, now)
	_, _ = s.AddCard(ctx, AddCardInput{Deck: "A", Front: "zwei", Back: "two"}, now)
	_, _ = s.AddCard(ctx, AddCardInput{Deck: "B", Front: "drei", Back: "three"}, now)

	decks, err := s.ListDecks(ctx, now)
	if err != nil {
		t.Fatalf("ListDecks: %v", err)
	}
	if len(decks) != 2 {
		t.Fatalf("expected 2 decks, got %d", len(decks))
	}

	stats, err := s.Stats(ctx, now)
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.TotalCards != 3 {
		t.Fatalf("total=%d", stats.TotalCards)
	}
}

func TestSearchCards(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	s := openTestStore(t)

	_, _ = s.AddCard(ctx, AddCardInput{Deck: "Starter", Front: "Hund", Back: "dog", Tags: []string{"animal"}}, now)
	_, _ = s.AddCard(ctx, AddCardInput{Deck: "Starter", Front: "Katze", Back: "cat", Tags: []string{"animal"}}, now)

	got, err := s.SearchCards(ctx, "hund", 10)
	if err != nil {
		t.Fatalf("SearchCards: %v", err)
	}
	if len(got) != 1 || got[0].Back != "dog" {
		t.Fatalf("unexpected search result: %+v", got)
	}
}

func TestMigrateBackfillsLegacySchedulerColumns(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "legacy.db")

	legacyDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open legacy db: %v", err)
	}
	defer legacyDB.Close()

	_, err = legacyDB.Exec(`
		CREATE TABLE cards (
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
		);
		CREATE TABLE review_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			card_id TEXT NOT NULL,
			grade INTEGER NOT NULL,
			answered_at TEXT NOT NULL,
			interval_days INTEGER NOT NULL,
			ease REAL NOT NULL
		);`)
	if err != nil {
		t.Fatalf("create legacy schema: %v", err)
	}

	created := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC).Format(time.RFC3339)
	updated := time.Date(2025, 1, 5, 12, 0, 0, 0, time.UTC).Format(time.RFC3339)
	legacyDueAt := time.Date(2030, 6, 1, 9, 0, 0, 0, time.UTC).Format(time.RFC3339)
	_, err = legacyDB.Exec(`
		INSERT INTO cards (id, deck, front, back, note_type, tags, due_at, interval_days, ease, reps, lapses, suspended, created_at, updated_at)
		VALUES ('c1', 'Legacy', 'Hallo', 'hello', '', '', ?, 7, 2.2, 3, 0, 0, ?, ?)`,
		legacyDueAt, created, updated)
	if err != nil {
		t.Fatalf("insert legacy row: %v", err)
	}

	s, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open migrated store: %v", err)
	}
	defer s.Close()

	card, err := s.GetCard(ctx, "c1")
	if err != nil {
		t.Fatalf("GetCard: %v", err)
	}
	if card.StabilityDays <= 0 {
		t.Fatalf("expected stability backfill, got %f", card.StabilityDays)
	}
	if card.Difficulty < 0 || card.Difficulty > 1 {
		t.Fatalf("difficulty out of range: %f", card.Difficulty)
	}
	if card.LastReviewAt.IsZero() {
		t.Fatal("expected last_review_at backfill")
	}
	if card.DueAt.Format(time.RFC3339) != legacyDueAt {
		t.Fatalf("migration unexpectedly rewrote due_at: got=%s want=%s", card.DueAt.Format(time.RFC3339), legacyDueAt)
	}
}
