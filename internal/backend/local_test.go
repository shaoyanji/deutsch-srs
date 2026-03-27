package backend

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/example/deutsch-srs/internal/srs"
)

func newLocalBackendForTest(t *testing.T) *LocalBackend {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "local.db")
	starter := filepath.Join("..", "..", "decks", "starter.yaml")
	be, err := NewLocalBackend(dbPath, starter)
	if err != nil {
		t.Fatalf("NewLocalBackend: %v", err)
	}
	be.now = func() time.Time {
		return time.Date(2100, 1, 2, 10, 0, 0, 0, time.UTC)
	}
	t.Cleanup(func() {
		_ = be.Close()
	})
	return be
}

func TestLocalReviewLoopEndToEnd(t *testing.T) {
	ctx := context.Background()
	be := newLocalBackendForTest(t)

	decks, err := be.ListDecks(ctx)
	if err != nil {
		t.Fatalf("ListDecks: %v", err)
	}
	if len(decks) == 0 {
		t.Fatal("expected seeded deck")
	}
	deck := decks[0].Name

	dueBefore, err := be.DueCards(ctx, deck, 50)
	if err != nil {
		t.Fatalf("DueCards(before): %v", err)
	}
	if len(dueBefore) == 0 {
		t.Fatal("expected due cards from starter seed")
	}

	if err := be.AnswerCard(ctx, dueBefore[0].ID, srs.GradeGood); err != nil {
		t.Fatalf("AnswerCard: %v", err)
	}

	dueAfter, err := be.DueCards(ctx, deck, 50)
	if err != nil {
		t.Fatalf("DueCards(after): %v", err)
	}
	if len(dueAfter) >= len(dueBefore) {
		t.Fatalf("expected due queue to shrink, before=%d after=%d", len(dueBefore), len(dueAfter))
	}
}

func TestLocalStartupSeedsWhenEmptyAndStarterProvided(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "startup-seed.db")
	starter := filepath.Join("..", "..", "decks", "starter.yaml")

	be, err := NewLocalBackend(dbPath, starter)
	if err != nil {
		t.Fatalf("NewLocalBackend: %v", err)
	}
	defer func() { _ = be.Close() }()

	decks, err := be.ListDecks(ctx)
	if err != nil {
		t.Fatalf("ListDecks: %v", err)
	}
	if len(decks) == 0 {
		t.Fatal("expected startup seeding to create at least one deck")
	}
	if decks[0].DueCount+decks[0].NewCount+decks[0].ReviewCount == 0 {
		t.Fatal("expected seeded deck to have cards")
	}
}

func TestStarterSeedingIsIdempotent(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "seed.db")
	starter := filepath.Join("..", "..", "decks", "starter.yaml")

	be1, err := NewLocalBackend(dbPath, starter)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	be1.now = func() time.Time {
		return time.Date(2100, 1, 2, 10, 0, 0, 0, time.UTC)
	}
	stats1, err := be1.Stats(ctx)
	if err != nil {
		t.Fatalf("stats1: %v", err)
	}
	_ = be1.Close()

	be2, err := NewLocalBackend(dbPath, starter)
	if err != nil {
		t.Fatalf("second open: %v", err)
	}
	be2.now = func() time.Time {
		return time.Date(2100, 1, 2, 10, 0, 0, 0, time.UTC)
	}
	defer func() { _ = be2.Close() }()

	stats2, err := be2.Stats(ctx)
	if err != nil {
		t.Fatalf("stats2: %v", err)
	}
	if stats1.TotalCards != stats2.TotalCards {
		t.Fatalf("starter seed not idempotent: first=%d second=%d", stats1.TotalCards, stats2.TotalCards)
	}
}

func TestLocalStartupInvalidStarterPathReturnsClearError(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "badstarter.db")
	_, err := NewLocalBackend(dbPath, "/does/not/exist/starter.yaml")
	if err == nil {
		t.Fatal("expected error for invalid starter path")
	}
	if got := err.Error(); got == "" || !containsAll(got, "local startup seed failed", "starter.yaml") {
		t.Fatalf("unexpected error text: %q", got)
	}
}

func TestAddCardPersistsAcrossReopen(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "persist.db")
	starter := filepath.Join("..", "..", "decks", "starter.yaml")

	be1, err := NewLocalBackend(dbPath, starter)
	if err != nil {
		t.Fatalf("open1: %v", err)
	}
	be1.now = func() time.Time {
		return time.Date(2100, 1, 2, 10, 0, 0, 0, time.UTC)
	}
	card, err := be1.AddCard(ctx, AddCardInput{
		Deck:  "Starter German",
		Front: "Wasser",
		Back:  "water",
		Tags:  []string{"noun"},
	})
	if err != nil {
		t.Fatalf("AddCard: %v", err)
	}
	if card.ID == "" {
		t.Fatal("expected card ID")
	}
	_ = be1.Close()

	be2, err := NewLocalBackend(dbPath, starter)
	if err != nil {
		t.Fatalf("open2: %v", err)
	}
	be2.now = func() time.Time {
		return time.Date(2100, 1, 2, 10, 0, 0, 0, time.UTC)
	}
	defer func() { _ = be2.Close() }()

	results, err := be2.SearchCards(ctx, "Wasser", 10)
	if err != nil {
		t.Fatalf("SearchCards: %v", err)
	}
	if len(results) != 1 || results[0].Back != "water" {
		t.Fatalf("unexpected persisted result: %+v", results)
	}
}

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !strings.Contains(s, p) {
			return false
		}
	}
	return true
}
