package backend

import (
	"context"

	"github.com/shaoyanji/deutsch-srs/internal/srs"
)

type Backend interface {
	ListDecks(ctx context.Context) ([]srs.Deck, error)
	DueCards(ctx context.Context, deck string, limit int) ([]srs.Card, error)
	ReviewQueue(ctx context.Context, deck string, limit int) (Queue, error)
	AddCard(ctx context.Context, in AddCardInput) (srs.Card, error)
	AnswerCard(ctx context.Context, cardID string, grade srs.ReviewGrade) error
	// SuspendCard hides a card until it is explicitly unsuspended; BuryCard
	// takes it out of today's queue only. Neither is a grade.
	SuspendCard(ctx context.Context, cardID string) error
	UnsuspendCard(ctx context.Context, cardID string) error
	BuryCard(ctx context.Context, cardID string) error
	// UndoReview reverts the most recent review write on this backend.
	UndoReview(ctx context.Context, undo ReviewUndo) error
	SearchCards(ctx context.Context, query string, limit int) ([]srs.Card, error)
	Stats(ctx context.Context) (Stats, error)
	Health(ctx context.Context) error
}
