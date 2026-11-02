package backend

import "github.com/shaoyanji/deutsch-srs/internal/srs"

type AddCardInput struct {
	Deck     string
	Front    string
	Back     string
	NoteType string
	Tags     []string
}

type Stats struct {
	TotalCards      int
	DueToday        int
	NewCards        int
	ReviewsToday    int
	BackendName     string
	BackendHealthy  bool
	HealthErrorText string
}

// ReviewUndo is what an action replaced: the card exactly as it was before the
// action, plus whether that action wrote a review_log row that has to come back
// out. A handler that keeps this can revert a grade, a bury and a suspend with
// one call instead of a compensating operation per action kind.
type ReviewUndo struct {
	Card   srs.Card
	Graded bool
}
