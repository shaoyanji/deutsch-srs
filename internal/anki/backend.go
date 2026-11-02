package anki

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/shaoyanji/deutsch-srs/internal/backend"
	"github.com/shaoyanji/deutsch-srs/internal/srs"
)

type Backend struct {
	client *Client
	now    func() time.Time
}

func NewBackend(baseURL string) *Backend {
	return &Backend{
		client: NewClient(baseURL),
		now:    time.Now,
	}
}

func (b *Backend) ListDecks(ctx context.Context) ([]srs.Deck, error) {
	names, err := b.client.DeckNames(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]srs.Deck, 0, len(names))
	for _, n := range names {
		out = append(out, srs.Deck{Name: n})
	}
	return out, nil
}

func (b *Backend) DueCards(ctx context.Context, deck string, limit int) ([]srs.Card, error) {
	query := fmt.Sprintf("deck:%q is:due", deck)
	ids, err := b.client.FindCards(ctx, query)
	if err != nil {
		return nil, err
	}
	if limit > 0 && len(ids) > limit {
		ids = ids[:limit]
	}
	if len(ids) == 0 {
		return nil, nil
	}
	infos, err := b.client.CardsInfo(ctx, ids)
	if err != nil {
		return nil, err
	}
	now := b.now().UTC()
	out := make([]srs.Card, 0, len(infos))
	for _, info := range infos {
		out = append(out, srs.Card{
			ID:        strconv.FormatInt(info.CardID, 10),
			Deck:      info.Deck,
			Front:     info.Front,
			Back:      info.Back,
			Tags:      info.Tags,
			DueAt:     now,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}
	return out, nil
}

func (b *Backend) AddCard(ctx context.Context, in backend.AddCardInput) (srs.Card, error) {
	id, err := b.client.AddNote(ctx, in.Deck, in.Front, in.Back, in.Tags)
	if err != nil {
		return srs.Card{}, err
	}
	now := b.now().UTC()
	return srs.Card{
		ID:        strconv.FormatInt(id, 10),
		Deck:      in.Deck,
		Front:     in.Front,
		Back:      in.Back,
		NoteType:  in.NoteType,
		Tags:      in.Tags,
		DueAt:     now,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (b *Backend) AnswerCard(context.Context, string, srs.ReviewGrade) error {
	return fmt.Errorf("answering cards is intentionally unsupported in AnkiConnect backend")
}

func (b *Backend) SearchCards(ctx context.Context, query string, limit int) ([]srs.Card, error) {
	ids, err := b.client.FindCards(ctx, query)
	if err != nil {
		return nil, err
	}
	if limit > 0 && len(ids) > limit {
		ids = ids[:limit]
	}
	if len(ids) == 0 {
		return nil, nil
	}
	infos, err := b.client.CardsInfo(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]srs.Card, 0, len(infos))
	for _, info := range infos {
		out = append(out, srs.Card{
			ID:    strconv.FormatInt(info.CardID, 10),
			Deck:  info.Deck,
			Front: info.Front,
			Back:  info.Back,
			Tags:  info.Tags,
		})
	}
	return out, nil
}

func (b *Backend) Stats(ctx context.Context) (backend.Stats, error) {
	ids, err := b.client.FindCards(ctx, "")
	if err != nil {
		return backend.Stats{}, err
	}
	due, err := b.client.FindCards(ctx, "is:due")
	if err != nil {
		return backend.Stats{}, err
	}
	reviewed, err := b.client.NumReviewsToday(ctx)
	if err != nil {
		return backend.Stats{}, err
	}
	healthErr := b.Health(ctx)
	out := backend.Stats{
		TotalCards:   len(ids),
		DueToday:     len(due),
		ReviewsToday: reviewed,
		BackendName:  "anki",
	}
	if healthErr != nil {
		out.HealthErrorText = healthErr.Error()
	} else {
		out.BackendHealthy = true
	}
	return out, nil
}

func (b *Backend) Health(ctx context.Context) error {
	return b.client.Health(ctx)
}

// ReviewQueue delegates to Anki's own scheduler: Anki decides what is due and
// in what order, so no daily new-card cap is applied here.
func (b *Backend) ReviewQueue(ctx context.Context, deck string, limit int) (backend.Queue, error) {
	cards, err := b.DueCards(ctx, deck, limit)
	if err != nil {
		return backend.Queue{}, err
	}
	return backend.Queue{Cards: cards}, nil
}

// Unsupported in AnkiConnect: bury and suspend are card actions there, so they
// map straight onto the collection and no local state is kept.

func (b *Backend) SuspendCard(ctx context.Context, cardID string) error {
	return b.cardAction(ctx, "suspend", cardID)
}

func (b *Backend) UnsuspendCard(ctx context.Context, cardID string) error {
	return b.cardAction(ctx, "unsuspend", cardID)
}

func (b *Backend) BuryCard(ctx context.Context, cardID string) error {
	return b.cardAction(ctx, "bury", cardID)
}

// UndoReview leans on Anki's own undo stack instead of replaying state: the
// grade, bury and suspend above were all Anki actions, so the last one is the
// one the caller means.
func (b *Backend) UndoReview(context.Context, backend.ReviewUndo) error {
	return b.client.Call(context.Background(), "undo", nil, nil)
}

func (b *Backend) cardAction(ctx context.Context, action, cardID string) error {
	id, err := strconv.ParseInt(cardID, 10, 64)
	if err != nil {
		return fmt.Errorf("anki %s: card id %q: %w", action, cardID, err)
	}
	return b.client.Call(ctx, action, map[string]any{"cards": []int64{id}}, nil)
}
