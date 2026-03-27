package backend

import (
	"context"

	"github.com/shaoyanji/deutsch-srs/internal/srs"
)

type Backend interface {
	ListDecks(ctx context.Context) ([]srs.Deck, error)
	DueCards(ctx context.Context, deck string, limit int) ([]srs.Card, error)
	AddCard(ctx context.Context, in AddCardInput) (srs.Card, error)
	AnswerCard(ctx context.Context, cardID string, grade srs.ReviewGrade) error
	SearchCards(ctx context.Context, query string, limit int) ([]srs.Card, error)
	Stats(ctx context.Context) (Stats, error)
	Health(ctx context.Context) error
}
