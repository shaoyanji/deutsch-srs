package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

const searchTestNow = "2026-01-02T10:00:00Z"

func seedSearchDeck(t *testing.T, s *SQLiteStore) {
	t.Helper()
	ctx := context.Background()
	now := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	cards := []AddCardInput{
		{Deck: "Vokabeln", Front: "Hund", Back: "dog", Tags: []string{"tier"}},
		{Deck: "Vokabeln", Front: "Hundehütte", Back: "dog kennel", Tags: []string{"tier"}},
		{Deck: "Vokabeln", Front: "Dachshund", Back: "dachshund", Tags: []string{"tier"}},
		{Deck: "Vokabeln", Front: "über", Back: "over", Tags: []string{"praeposition"}},
		{Deck: "Vokabeln", Front: "schön", Back: "beautiful", Tags: []string{"adjektiv"}},
		{Deck: "Sätze", Front: "Ich lerne Deutsch.", Back: "I am learning German.", Tags: []string{"satz"}},
		{Deck: "Sätze", Front: "Ich lerne kein Deutsch.", Back: "I am not learning German.", Tags: []string{"satz"}},
	}
	for _, in := range cards {
		if _, err := s.AddCard(ctx, in, now); err != nil {
			t.Fatalf("AddCard(%q): %v", in.Front, err)
		}
	}
}

func searchFronts(t *testing.T, s *SQLiteStore, query string, limit int) []string {
	t.Helper()
	got, err := s.SearchCards(context.Background(), query, limit)
	if err != nil {
		t.Fatalf("SearchCards(%q): %v", query, err)
	}
	out := make([]string, 0, len(got))
	for _, c := range got {
		out = append(out, c.Front)
	}
	return out
}

func TestSearchCardsPrefixMatchesWordStarts(t *testing.T) {
	s := openTestStore(t)
	seedSearchDeck(t, s)

	got := searchFronts(t, s, "hund", 10)
	// A prefix query finds Hund and Hundehütte; Dachshund contains the string but
	// does not start with it, which is the trade the index buys.
	if len(got) != 2 || got[0] != "Hund" || got[1] != "Hundehütte" {
		t.Fatalf("prefix search results = %v, want [Hund Hundehütte]", got)
	}
}

func TestSearchCardsMatchesBackAndTags(t *testing.T) {
	s := openTestStore(t)
	seedSearchDeck(t, s)

	if got := searchFronts(t, s, "kennel", 10); len(got) != 1 || got[0] != "Hundehütte" {
		t.Fatalf("back search = %v", got)
	}
	if got := searchFronts(t, s, "adjektiv", 10); len(got) != 1 || got[0] != "schön" {
		t.Fatalf("tag search = %v", got)
	}
	if got := searchFronts(t, s, "Sätze", 10); len(got) != 2 {
		t.Fatalf("deck-name search = %v", got)
	}
}

func TestSearchCardsMultiTokenRequiresEveryWord(t *testing.T) {
	s := openTestStore(t)
	seedSearchDeck(t, s)

	got := searchFronts(t, s, "ich lern", 10)
	if len(got) != 2 {
		t.Fatalf("multi-token search = %v", got)
	}
	if got := searchFronts(t, s, "ich hund", 10); len(got) != 0 {
		t.Fatalf("tokens from different cards matched: %v", got)
	}
}

func TestSearchCardsRanksWholeWordAboveCompound(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	s := openTestStore(t)
	for _, front := range []string{"Hausschlüssel", "Häuser", "Haus"} {
		if _, err := s.AddCard(ctx, AddCardInput{Deck: "Vokabeln", Front: front, Back: "house"}, now); err != nil {
			t.Fatalf("AddCard(%q): %v", front, err)
		}
	}

	got := searchFronts(t, s, "haus", 10)
	if len(got) != 3 {
		t.Fatalf("results = %v, want all three", got)
	}
	if got[0] != "Haus" {
		t.Fatalf("exact word did not rank first: %v", got)
	}
}

func TestSearchCardsFoldsUmlauts(t *testing.T) {
	s := openTestStore(t)
	seedSearchDeck(t, s)

	// unicode61 folds diacritics, so a learner without a German keyboard can
	// still find the card. The flip side is that "schon" also finds "schön".
	if got := searchFronts(t, s, "uber", 10); len(got) != 1 || got[0] != "über" {
		t.Fatalf("ascii query for umlaut card = %v", got)
	}
	if got := searchFronts(t, s, "schon", 10); len(got) != 1 || got[0] != "schön" {
		t.Fatalf("folded match for schön = %v", got)
	}
}

func TestSearchCardsTreatsOperatorsAsText(t *testing.T) {
	s := openTestStore(t)
	seedSearchDeck(t, s)

	// Input that would be FTS5 syntax if passed through must not error or widen
	// the query: quotes, stars, parentheses and keywords all become tokens.
	for _, q := range []string{`hund"`, `hund*`, `NEAR(`, `OR`, `"`, `-hund`, `hund OR katze`} {
		if _, err := s.SearchCards(context.Background(), q, 10); err != nil {
			t.Fatalf("SearchCards(%q) errored: %v", q, err)
		}
	}
	// OR is a token here, not an operator, so this narrows to nothing instead of
	// widening to every card.
	if got := searchFronts(t, s, "hund OR katze", 10); len(got) != 0 {
		t.Fatalf("operator-shaped query matched %v, want none", got)
	}
}

func TestSearchCardsEmptyQueryReturnsNothing(t *testing.T) {
	s := openTestStore(t)
	seedSearchDeck(t, s)

	for _, q := range []string{"", "   ", "!!!", "\n\t"} {
		got, err := s.SearchCards(context.Background(), q, 10)
		if err != nil {
			t.Fatalf("SearchCards(%q): %v", q, err)
		}
		if len(got) != 0 {
			t.Fatalf("SearchCards(%q) returned %d cards, want none", q, len(got))
		}
	}
}

func TestSearchCardsRespectsLimit(t *testing.T) {
	s := openTestStore(t)
	seedSearchDeck(t, s)

	if got := searchFronts(t, s, "hund", 1); len(got) != 1 {
		t.Fatalf("limit 1 returned %v", got)
	}
}

func TestSearchIndexFollowsBulkImport(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	s := openTestStore(t)

	bulk := []AddCardInput{
		{Deck: "Wortschatz", Front: "Zeit", Back: "time"},
		{Deck: "Wortschatz", Front: "Zeitung", Back: "newspaper"},
	}
	if _, err := s.AddCardsBulk(ctx, bulk, now, BulkOptions{SkipExisting: true}); err != nil {
		t.Fatalf("AddCardsBulk: %v", err)
	}
	if got := searchFronts(t, s, "zeit", 10); len(got) != 2 {
		t.Fatalf("bulk-imported cards missing from index: %v", got)
	}
}

func TestSearchIndexSurvivesReopen(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "reopen.db")
	s, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	seedSearchDeck(t, s)

	var marker int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_meta WHERE key = ?`, searchIndexMarker).Scan(&marker); err != nil {
		t.Fatalf("read marker: %v", err)
	}
	if marker != 1 {
		t.Fatalf("index build marker missing after first open")
	}
	if err := s.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	reopened, err := Open(dbPath)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer reopened.Close()
	if got := searchFronts(t, reopened, "hund", 10); len(got) != 2 {
		t.Fatalf("search after reopen = %v", got)
	}
}

func TestSearchIndexBackfillsCardsInsertedBeforeItExisted(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "legacy.db")

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
		stability_days REAL NOT NULL DEFAULT 1.0,
		difficulty REAL NOT NULL DEFAULT 0.35,
		last_review_at TEXT NOT NULL DEFAULT '',
		reps INTEGER NOT NULL DEFAULT 0,
		lapses INTEGER NOT NULL DEFAULT 0,
		suspended INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);`)
	if err != nil {
		t.Fatalf("create legacy schema: %v", err)
	}
	_, err = legacy.Exec(`INSERT INTO cards (id, deck, front, back, note_type, tags, due_at, interval_days, ease,
		stability_days, difficulty, last_review_at, reps, lapses, suspended, created_at, updated_at)
		VALUES ('legacy-1', 'Alt', 'Wasser', 'water', '', 'nomen', ?, 0, 2.5, 1.0, 0.35, '', 0, 0, 0, ?, ?)`,
		searchTestNow, searchTestNow, searchTestNow)
	if err != nil {
		t.Fatalf("insert legacy card: %v", err)
	}
	if err := legacy.Close(); err != nil {
		t.Fatalf("close legacy: %v", err)
	}

	s, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open store over legacy db: %v", err)
	}
	defer s.Close()
	if got := searchFronts(t, s, "wasser", 10); len(got) != 1 || got[0] != "Wasser" {
		t.Fatalf("card that predates the index was not backfilled: %v", got)
	}
}
