package store

import (
	"context"
	"testing"
	"time"
)

func TestAddCardsBulkSkipsExisting(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	s := openTestStore(t)

	cards := []AddCardInput{
		{Deck: "Wortschatz", Front: "Hund", Back: "dog"},
		{Deck: "wortschatz", Front: "hund", Back: "dog (dup case)"},
		{Deck: "Wortschatz", Front: "Katze", Back: "cat"},
	}

	res, err := s.AddCardsBulk(ctx, cards, now, BulkOptions{SkipExisting: true})
	if err != nil {
		t.Fatalf("first import: %v", err)
	}
	if res.Added != 2 || res.Skipped != 1 {
		t.Fatalf("first import added=%d skipped=%d, want 2/1", res.Added, res.Skipped)
	}

	res, err = s.AddCardsBulk(ctx, cards, now, BulkOptions{SkipExisting: true})
	if err != nil {
		t.Fatalf("second import: %v", err)
	}
	if res.Added != 0 || res.Skipped != 3 {
		t.Fatalf("second import added=%d skipped=%d, want 0/3", res.Added, res.Skipped)
	}

	n, err := s.CardCount(ctx)
	if err != nil {
		t.Fatalf("CardCount: %v", err)
	}
	if n != 2 {
		t.Fatalf("cards=%d want=2 (re-import must not duplicate)", n)
	}
}

// SQLite's LOWER() folds ASCII only, so a non-ASCII headword used to slip past
// the duplicate check and be imported twice.
func TestAddCardsBulkSkipsExistingNonASCII(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	s := openTestStore(t)

	cards := []AddCardInput{{Deck: "Wortschatz", Front: "Öl", Back: "oil"}}
	if _, err := s.AddCardsBulk(ctx, cards, now, BulkOptions{SkipExisting: true}); err != nil {
		t.Fatalf("first import: %v", err)
	}
	res, err := s.AddCardsBulk(ctx, cards, now, BulkOptions{SkipExisting: true})
	if err != nil {
		t.Fatalf("second import: %v", err)
	}
	if res.Added != 0 || res.Skipped != 1 {
		t.Fatalf("re-import added=%d skipped=%d, want 0/1", res.Added, res.Skipped)
	}
}

func TestAddCardsBulkDryRunWritesNothing(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	s := openTestStore(t)

	res, err := s.AddCardsBulk(ctx, []AddCardInput{{Deck: "D", Front: "a", Back: "b"}}, now, BulkOptions{SkipExisting: true, DryRun: true})
	if err != nil {
		t.Fatalf("dry run: %v", err)
	}
	if res.Added != 1 {
		t.Fatalf("dry run added=%d want=1", res.Added)
	}
	n, err := s.CardCount(ctx)
	if err != nil {
		t.Fatalf("CardCount: %v", err)
	}
	if n != 0 {
		t.Fatalf("dry run wrote %d cards", n)
	}
}

func TestAddCardsBulkCountsInvalidRows(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	s := openTestStore(t)

	res, err := s.AddCardsBulk(ctx, []AddCardInput{
		{Deck: "D", Front: "ok", Back: "fine"},
		{Deck: "D", Front: "", Back: "no front"},
		{Deck: "", Front: "no deck", Back: "x"},
	}, now, BulkOptions{SkipExisting: true})
	if err != nil {
		t.Fatalf("bulk: %v", err)
	}
	if res.Added != 1 || res.Invalid != 2 {
		t.Fatalf("added=%d invalid=%d, want 1/2", res.Added, res.Invalid)
	}
}
