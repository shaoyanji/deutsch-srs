package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"
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
