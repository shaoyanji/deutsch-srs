package backend

import (
	"context"
	"time"

	"github.com/shaoyanji/deutsch-srs/internal/srs"
)

// DefaultNewPerDay is how many unseen cards a deck introduces per day unless
// the deck says otherwise.
const DefaultNewPerDay = 20

// Queue is one review batch: due reviews first, then as many new cards as the
// deck's daily allowance still permits.
type Queue struct {
	Cards        []srs.Card
	NewPerDay    int
	NewToday     int
	NewRemaining int
}

// DeckLimit is the daily new-card setting of one deck, with today's usage.
type DeckLimit struct {
	Deck      string `json:"deck"`
	NewPerDay int    `json:"new_per_day"`
	IsDefault bool   `json:"is_default"`
	NewToday  int    `json:"new_today"`
}

// SetDefaultNewPerDay sets the allowance used by decks without their own setting.
func (b *LocalBackend) SetDefaultNewPerDay(n int) {
	if n >= 0 {
		b.defaultNewPerDay = n
	}
}

// ReviewQueue builds the next batch for a deck. Reviews come first, so a large
// backlog of unseen cards can never crowd out cards that are due again.
func (b *LocalBackend) ReviewQueue(ctx context.Context, deck string, limit int) (Queue, error) {
	if limit <= 0 {
		limit = 50
	}
	now := b.now().UTC()

	perDay, err := b.newPerDayFor(ctx, deck)
	if err != nil {
		return Queue{}, err
	}
	introduced, err := b.store.IntroducedToday(ctx, deck, startOfDayUTC(now))
	if err != nil {
		return Queue{}, err
	}
	allowance := perDay - introduced
	if allowance < 0 {
		allowance = 0
	}

	reviews, err := b.store.DueReviews(ctx, deck, limit, now)
	if err != nil {
		return Queue{}, err
	}

	q := Queue{NewPerDay: perDay, NewToday: introduced}
	if room := limit - len(reviews); room > 0 && allowance > 0 {
		if allowance < room {
			room = allowance
		}
		fresh, err := b.store.NewCards(ctx, deck, room, now)
		if err != nil {
			return Queue{}, err
		}
		reviews = append(reviews, fresh...)
		allowance -= len(fresh)
	}

	q.Cards = reviews
	q.NewRemaining = allowance
	return q, nil
}

// SetDeckNewPerDay stores a per-deck cap.
func (b *LocalBackend) SetDeckNewPerDay(ctx context.Context, deck string, n int) error {
	return b.store.SetDeckNewPerDay(ctx, deck, n, b.now().UTC())
}

// DeckLimits reports every deck with its effective cap and today's usage.
func (b *LocalBackend) DeckLimits(ctx context.Context) ([]DeckLimit, error) {
	stored, err := b.store.DeckNewPerDayAll(ctx)
	if err != nil {
		return nil, err
	}
	decks, err := b.store.ListDecks(ctx, b.now().UTC())
	if err != nil {
		return nil, err
	}

	out := make([]DeckLimit, 0, len(decks))
	for _, d := range decks {
		perDay, ok := stored[d.Name]
		if !ok {
			perDay = b.defaultNewPerDay
		}
		today, err := b.store.IntroducedToday(ctx, d.Name, startOfDayUTC(b.now().UTC()))
		if err != nil {
			return nil, err
		}
		out = append(out, DeckLimit{Deck: d.Name, NewPerDay: perDay, IsDefault: !ok, NewToday: today})
	}
	return out, nil
}

func (b *LocalBackend) newPerDayFor(ctx context.Context, deck string) (int, error) {
	stored, ok, err := b.store.DeckNewPerDay(ctx, deck)
	if err != nil {
		return 0, err
	}
	if ok {
		return stored, nil
	}
	return b.defaultNewPerDay, nil
}

func startOfDayUTC(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
