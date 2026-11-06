package anksync

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/shaoyanji/deutsch-srs/internal/srs"
)

// Collection is a read-only view of an Anki .anki2 file, opened from the copy
// the sync server served. It reads the tables the Anki schema keeps plain
// (notes.flds, cards.*, decks.name) and does the scheduling arithmetic itself,
// so no Anki binary is involved.
type Collection struct {
	db       *sql.DB
	crt      int64
	rollover int
	decks    map[int64]string
}

// OpenCollection opens a collection file without ever writing to it.
func OpenCollection(path string) (*Collection, error) {
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("open collection: %w", err)
	}
	db.SetMaxOpenConns(1)
	c := &Collection{db: db, decks: map[int64]string{}}
	if err := c.load(); err != nil {
		db.Close()
		return nil, err
	}
	return c, nil
}

func (c *Collection) Close() error { return c.db.Close() }

func (c *Collection) load() error {
	var conf string
	if err := c.db.QueryRow(`select crt, conf from col limit 1`).Scan(&c.crt, &conf); err != nil {
		return fmt.Errorf("read col table: %w", err)
	}
	var cfg struct {
		Rollover int `json:"rollover"`
	}
	if err := json.Unmarshal([]byte(conf), &cfg); err == nil && cfg.Rollover > 0 {
		c.rollover = cfg.Rollover
	} else {
		c.rollover = 4
	}

	rows, err := c.db.Query(`select id, name from decks`)
	if err != nil {
		return fmt.Errorf("read decks: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return err
		}
		c.decks[id] = name
	}
	return rows.Err()
}

// today is the Anki day number, i.e. how many rollover boundaries have passed
// since the collection was created. Review cards store their due date in this
// unit; learning cards store epoch seconds instead.
func (c *Collection) today(now time.Time) int64 {
	return (now.Unix() - int64(c.rollover)*3600 - c.crt) / 86400
}

// cardRow is one cards+notes join, before it is mapped to srs.Card.
type cardRow struct {
	id, did, mod int64
	flds, tags   string
	typ, queue   int
	due, ivl     int64
	factor       int
	reps, lapses int
}

func (c *Collection) selectCards() ([]cardRow, error) {
	rows, err := c.db.Query(`
		select c.id, c.did, coalesce(n.mod, 0), coalesce(n.flds, ''), coalesce(n.tags, ''),
		       c.type, c.queue, c.due, c.ivl, c.factor, c.reps, c.lapses
		from cards c left join notes n on n.id = c.nid`)
	if err != nil {
		return nil, fmt.Errorf("read cards: %w", err)
	}
	defer rows.Close()

	var out []cardRow
	for rows.Next() {
		var r cardRow
		if err := rows.Scan(&r.id, &r.did, &r.mod, &r.flds, &r.tags,
			&r.typ, &r.queue, &r.due, &r.ivl, &r.factor, &r.reps, &r.lapses); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// Decks lists every deck in the collection with its own (not inherited) counts.
func (c *Collection) Decks(ctx context.Context, now time.Time) ([]srs.Deck, error) {
	rows, err := c.selectCards()
	if err != nil {
		return nil, err
	}
	byID := make(map[int64]*srs.Deck, len(c.decks))
	for id, name := range c.decks {
		byID[id] = &srs.Deck{Name: name}
	}
	today := c.today(now)
	for _, r := range rows {
		d := byID[r.did]
		if d == nil || r.queue == -1 {
			continue
		}
		switch {
		case r.typ == 0:
			d.NewCount++
		case r.typ == 2 && r.due <= today:
			d.DueCount++
		case (r.typ == 1 || r.typ == 3) && r.due <= now.Unix():
			d.DueCount++
		}
	}
	out := make([]srs.Deck, 0, len(byID))
	for _, d := range byID {
		out = append(out, *d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// AllCards returns every card in the collection, due or not. A lookup needs the
// whole set: the rule a drill points at is usually learned long ago, so a search
// restricted to today's queue would miss exactly the card being asked for.
func (c *Collection) AllCards(now time.Time) ([]srs.Card, error) {
	rows, err := c.selectCards()
	if err != nil {
		return nil, err
	}
	return c.mapCards(rows, now), nil
}

// DueCards returns the cards the server's scheduler says are due now, in due
// order. New cards are not included: they are not "due", they are an allowance.
func (c *Collection) DueCards(_ context.Context, deck string, limit int, now time.Time) ([]srs.Card, error) {
	rows, err := c.selectDue(deck, now)
	if err != nil {
		return nil, err
	}
	if limit > 0 && len(rows) > limit {
		rows = rows[:limit]
	}
	return c.mapCards(rows, now), nil
}

// NewCards returns unseen cards in the order Anki would introduce them.
func (c *Collection) NewCards(deck string, limit int, now time.Time) ([]srs.Card, error) {
	all, err := c.selectCards()
	if err != nil {
		return nil, err
	}
	var picked []cardRow
	for _, r := range all {
		if r.typ != 0 || r.queue == -1 || !c.inDeck(r.did, deck) {
			continue
		}
		picked = append(picked, r)
	}
	sort.Slice(picked, func(i, j int) bool { return picked[i].due < picked[j].due })
	if limit > 0 && len(picked) > limit {
		picked = picked[:limit]
	}
	return c.mapCards(picked, now), nil
}

func (c *Collection) selectDue(deck string, now time.Time) ([]cardRow, error) {
	all, err := c.selectCards()
	if err != nil {
		return nil, err
	}
	today := c.today(now)
	var due []cardRow
	for _, r := range all {
		if r.queue == -1 || !c.inDeck(r.did, deck) {
			continue
		}
		switch {
		case r.typ == 2 && r.due <= today:
			due = append(due, r)
		case (r.typ == 1 || r.typ == 3) && r.due <= now.Unix():
			due = append(due, r)
		}
	}
	// Anki serves learning cards that are due before reviews, then orders each
	// group by its own due time. Reviews first would hide a card you are
	// actively learning behind a two-week backlog.
	sort.SliceStable(due, func(i, j int) bool {
		li, lj := isLearning(due[i]), isLearning(due[j])
		if li != lj {
			return li
		}
		return c.dueAt(due[i], now).Before(c.dueAt(due[j], now))
	})
	return due, nil
}

func isLearning(r cardRow) bool { return r.typ == 1 || r.typ == 3 }

// inDeck matches a deck and its subdecks, the way Anki's `deck:X` filter does.
func (c *Collection) inDeck(id int64, deck string) bool {
	if deck == "" {
		return true
	}
	name := c.decks[id]
	return name == deck || strings.HasPrefix(name, deck+"::")
}

// dueAt converts the stored due value to a time. Review cards count days since
// collection creation; learning and relearning cards store epoch seconds.
func (c *Collection) dueAt(r cardRow, now time.Time) time.Time {
	switch {
	case r.typ == 0:
		return now
	case r.typ == 2:
		return time.Unix(c.crt+int64(c.rollover)*3600+r.due*86400, 0)
	default:
		return time.Unix(r.due, 0)
	}
}

func (c *Collection) mapCards(rows []cardRow, now time.Time) []srs.Card {
	out := make([]srs.Card, 0, len(rows))
	for _, r := range rows {
		fields := strings.Split(r.flds, "\x1f")
		front, back := "", ""
		if len(fields) > 0 {
			front = plainText(fields[0])
		}
		if len(fields) > 1 {
			back = plainText(strings.Join(fields[1:], "\n"))
		}
		out = append(out, srs.Card{
			ID:           fmt.Sprint(r.id),
			Deck:         c.decks[r.did],
			Front:        front,
			Back:         back,
			NoteType:     "Anki",
			Tags:         strings.Fields(r.tags),
			DueAt:        c.dueAt(r, now),
			IntervalDays: int(r.ivl),
			Ease:         float64(r.factor) / 1000,
			Reps:         r.reps,
			Lapses:       r.lapses,
			Suspended:    r.queue == -1,
			UpdatedAt:    time.Unix(r.mod, 0),
		})
	}
	return out
}

func (c *Collection) ReviewsToday(now time.Time) (int, error) {
	var n int
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	err := c.db.QueryRow(`select count(*) from revlog where id >= ?`, start.UnixMilli()).Scan(&n)
	if err != nil {
		return 0, err
	}
	return n, nil
}

var (
	tagPattern   = regexp.MustCompile(`<[^>]*>`)
	soundPattern = regexp.MustCompile(`\[sound:[^\]]*\]`)
	brPattern    = regexp.MustCompile(`(?i)<br\s*/?>`)
	newlineTags  = regexp.MustCompile(`(?i)</(div|p|li|tr)>`)
)

// plainText strips the HTML Anki stores in fields; the TUI renders text.
func plainText(field string) string {
	s := soundPattern.ReplaceAllString(field, "")
	s = brPattern.ReplaceAllString(s, "\n")
	s = newlineTags.ReplaceAllString(s, "\n")
	s = tagPattern.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	s = strings.ReplaceAll(s, "\u00a0", " ")
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = strings.TrimSpace(lines[i])
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}
