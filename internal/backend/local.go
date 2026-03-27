package backend

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/shaoyanji/deutsch-srs/internal/importexport"
	"github.com/shaoyanji/deutsch-srs/internal/srs"
	"github.com/shaoyanji/deutsch-srs/internal/store"
)

type LocalBackend struct {
	store *store.SQLiteStore
	now   func() time.Time
}

func NewLocalBackend(dbPath, starterDeckPath string) (*LocalBackend, error) {
	st, err := store.Open(dbPath)
	if err != nil {
		return nil, err
	}
	b := &LocalBackend{
		store: st,
		now:   time.Now,
	}
	if err := b.seedIfEmpty(context.Background(), starterDeckPath); err != nil {
		_ = st.Close()
		return nil, fmt.Errorf("local startup seed failed: %w", err)
	}
	return b, nil
}

func (b *LocalBackend) Close() error {
	return b.store.Close()
}

func (b *LocalBackend) seedIfEmpty(ctx context.Context, starterDeckPath string) error {
	starterDeckPath = strings.TrimSpace(starterDeckPath)
	if starterDeckPath == "" {
		return nil
	}
	n, err := b.store.CardCount(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	deckFile, err := importexport.LoadYAML(starterDeckPath)
	if err != nil {
		return fmt.Errorf("load starter deck %q: %w", starterDeckPath, err)
	}
	for _, c := range deckFile.Cards {
		_, err := b.store.AddCard(ctx, store.AddCardInput{
			Deck:     deckFile.Deck,
			Front:    c.Front,
			Back:     c.Back,
			NoteType: c.NoteType,
			Tags:     c.Tags,
		}, b.now().UTC())
		if err != nil {
			return fmt.Errorf("seed card: %w", err)
		}
	}
	return nil
}

func (b *LocalBackend) ListDecks(ctx context.Context) ([]srs.Deck, error) {
	return b.store.ListDecks(ctx, b.now().UTC())
}

func (b *LocalBackend) DueCards(ctx context.Context, deck string, limit int) ([]srs.Card, error) {
	return b.store.DueCards(ctx, deck, limit, b.now().UTC())
}

func (b *LocalBackend) AddCard(ctx context.Context, in AddCardInput) (srs.Card, error) {
	return b.store.AddCard(ctx, store.AddCardInput{
		Deck:     in.Deck,
		Front:    in.Front,
		Back:     in.Back,
		NoteType: in.NoteType,
		Tags:     in.Tags,
	}, b.now().UTC())
}

func (b *LocalBackend) AnswerCard(ctx context.Context, cardID string, grade srs.ReviewGrade) error {
	card, err := b.store.GetCard(ctx, cardID)
	if err != nil {
		return err
	}
	next, err := srs.NextCard(card, grade, b.now().UTC())
	if err != nil {
		return err
	}
	if err := b.store.UpdateCardSchedule(ctx, next); err != nil {
		return err
	}
	return b.store.InsertReviewLog(ctx, cardID, grade, b.now().UTC(), next.IntervalDays, next.Ease)
}

func (b *LocalBackend) SearchCards(ctx context.Context, query string, limit int) ([]srs.Card, error) {
	return b.store.SearchCards(ctx, query, limit)
}

func (b *LocalBackend) Stats(ctx context.Context) (Stats, error) {
	st, err := b.store.Stats(ctx, b.now().UTC())
	if err != nil {
		return Stats{}, err
	}
	healthErr := b.Health(ctx)
	out := Stats{
		TotalCards:   st.TotalCards,
		DueToday:     st.DueToday,
		NewCards:     st.NewCards,
		ReviewsToday: st.ReviewsToday,
		BackendName:  "local",
	}
	if healthErr != nil {
		out.HealthErrorText = healthErr.Error()
	} else {
		out.BackendHealthy = true
	}
	return out, nil
}

func (b *LocalBackend) Health(ctx context.Context) error {
	return b.store.Health(ctx)
}
