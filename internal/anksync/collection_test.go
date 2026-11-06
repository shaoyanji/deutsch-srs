package anksync

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// makeCollection writes a collection with the tables and columns Anki's schema
// keeps in plain form: col (crt/conf), decks (id/name), notes (flds/tags) and
// cards (type/queue/due/ivl/factor).
func makeCollection(t *testing.T, now time.Time) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.anki2")
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatalf("create collection: %v", err)
	}
	defer db.Close()

	schema := []string{
		`create table col (id integer primary key, crt integer not null, conf text not null)`,
		`create table decks (id integer primary key, name text not null)`,
		`create table notes (id integer primary key, mid integer not null, mod integer not null,
			tags text not null, flds text not null)`,
		`create table cards (id integer primary key, nid integer not null, did integer not null,
			type integer not null, queue integer not null, due integer not null, ivl integer not null,
			factor integer not null, reps integer not null, lapses integer not null)`,
		`create table revlog (id integer primary key, cid integer not null)`,
	}
	for _, stmt := range schema {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("schema %q: %v", stmt, err)
		}
	}

	// Day 0 is ten days ago, so the review cards below sit inside the window.
	crt := now.AddDate(0, 0, -10).Unix()
	if _, err := db.Exec(`insert into col (id, crt, conf) values (1, ?, ?)`,
		crt, `{"rollover":4}`); err != nil {
		t.Fatalf("insert col: %v", err)
	}
	if _, err := db.Exec(`insert into decks (id, name) values (1, 'Deutsch::B2'), (2, 'Default')`); err != nil {
		t.Fatalf("insert decks: %v", err)
	}

	today := (now.Unix() - 4*3600 - crt) / 86400
	type note struct {
		id        int64
		deck      int64
		front     string
		back      string
		tags      string
		typ       int
		queue     int
		due       int64
		suspended bool
	}
	notes := []note{
		{id: 1, deck: 1, front: "die Übung", back: "the exercise", tags: " noun ", typ: 2, queue: 2, due: today - 2},
		{id: 2, deck: 1, front: "der Aufsatz", back: "<b>the essay</b>", tags: " noun ", typ: 1, queue: 1, due: now.Add(-90 * time.Second).Unix()},
		{id: 3, deck: 1, front: "verfassen", back: "to compose", tags: " verb ", typ: 0, queue: 0, due: 3},
		{id: 4, deck: 1, front: "die Prüfung", back: "the exam", tags: "", typ: 2, queue: -1, due: today - 5},
		{id: 5, deck: 2, front: "hallo", back: "hello", tags: "", typ: 2, queue: 2, due: today + 3},
	}
	for _, n := range notes {
		if _, err := db.Exec(`insert into notes (id, mid, mod, tags, flds) values (?, 1, ?, ?, ?)`,
			n.id, now.Unix(), n.tags, n.front+"\x1f"+n.back); err != nil {
			t.Fatalf("insert note %d: %v", n.id, err)
		}
		if _, err := db.Exec(`insert into cards (id, nid, did, type, queue, due, ivl, factor, reps, lapses)
			values (?, ?, ?, ?, ?, ?, 5, 2500, 3, 1)`, n.id, n.id, n.deck, n.typ, n.queue, n.due); err != nil {
			t.Fatalf("insert card %d: %v", n.id, err)
		}
	}
	return path
}

func TestCollectionReadsDecksAndDue(t *testing.T) {
	now := time.Now()
	col, err := OpenCollection(makeCollection(t, now))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer col.Close()

	decks, err := col.Decks(context.Background(), now)
	if err != nil {
		t.Fatalf("decks: %v", err)
	}
	if len(decks) != 2 {
		t.Fatalf("decks = %d, want 2", len(decks))
	}
	// Deutsch::B2 owns one review due, one learning due, one new, one suspended.
	if decks[0].Name != "Default" || decks[1].Name != "Deutsch::B2" {
		t.Fatalf("deck order = %q, %q", decks[0].Name, decks[1].Name)
	}
	if got := decks[1]; got.DueCount != 2 || got.NewCount != 1 {
		t.Errorf("Deutsch::B2 due=%d new=%d, want 2 and 1", got.DueCount, got.NewCount)
	}
	if got := decks[0]; got.DueCount != 0 || got.NewCount != 0 {
		t.Errorf("Default due=%d new=%d, want 0 and 0 (card due in three days)", got.DueCount, got.NewCount)
	}

	due, err := col.DueCards(context.Background(), "Deutsch", 10, now)
	if err != nil {
		t.Fatalf("due: %v", err)
	}
	if len(due) != 2 {
		t.Fatalf("due cards = %d, want 2 (suspended and future excluded)", len(due))
	}
	if due[0].Front != "der Aufsatz" {
		t.Errorf("first due card = %q, want the learning card that lapsed 90s ago", due[0].Front)
	}
	if due[0].Back != "the essay" {
		t.Errorf("back = %q, want HTML stripped", due[0].Back)
	}
	if len(due[0].Tags) != 1 || due[0].Tags[0] != "noun" {
		t.Errorf("tags = %v, want [noun]", due[0].Tags)
	}

	fresh, err := col.NewCards("Deutsch", 10, now)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if len(fresh) != 1 || fresh[0].Front != "verfassen" {
		t.Fatalf("new cards = %v, want the single unseen card", fresh)
	}
}

// TestReviewQueuePutsDueBeforeNew pins the batch order: a backlog of unseen
// cards must never push out cards that are already due.
func TestReviewQueuePutsDueBeforeNew(t *testing.T) {
	now := time.Now()
	col, err := OpenCollection(makeCollection(t, now))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer col.Close()

	be := &Backend{col: col, now: func() time.Time { return now }}
	ctx := context.Background()

	tight, err := be.ReviewQueue(ctx, "Deutsch", 2)
	if err != nil {
		t.Fatalf("queue: %v", err)
	}
	if len(tight.Cards) != 2 || tight.Cards[0].Front != "der Aufsatz" {
		t.Fatalf("queue = %d cards starting %q, want the two due cards", len(tight.Cards), tight.Cards[0].Front)
	}

	roomy, err := be.ReviewQueue(ctx, "Deutsch", 5)
	if err != nil {
		t.Fatalf("queue: %v", err)
	}
	if len(roomy.Cards) != 3 {
		t.Fatalf("queue = %d cards, want 2 due + 1 new", len(roomy.Cards))
	}
	if roomy.Cards[len(roomy.Cards)-1].Front != "verfassen" {
		t.Errorf("last card = %q, want the new card last", roomy.Cards[len(roomy.Cards)-1].Front)
	}
}

// TestSearchReachesACardThatIsNotDue pins the lookup the prerequisite panel
// makes: the rule a drill points at was learned weeks ago, so a search limited
// to today's queue would answer "no such card" for the one card asked for.
func TestSearchReachesACardThatIsNotDue(t *testing.T) {
	now := time.Now()
	col, err := OpenCollection(makeCollection(t, now))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer col.Close()

	be := &Backend{col: col, now: func() time.Time { return now }}
	found, err := be.SearchCards(context.Background(), "hallo", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(found) != 1 || found[0].Front != "hallo" {
		t.Fatalf("search = %+v, want the card due in three days", found)
	}
}
