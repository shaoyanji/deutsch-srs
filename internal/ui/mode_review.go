package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/example/deutsch-srs/internal/srs"
)

func (m Model) viewReview() string {
	var b strings.Builder
	b.WriteString(m.theme.Title.Render("Review") + "\n\n")
	if len(m.reviewCards) == 0 {
		b.WriteString(m.theme.Muted.Render("No due cards in this deck. Press esc to return to Browse."))
		if m.errText != "" {
			b.WriteString("\n\n" + m.theme.Error.Render("Review error: "+m.errText))
		}
		return b.String()
	}
	if m.reviewIdx >= len(m.reviewCards) {
		b.WriteString(m.theme.StatusLine.Render(reviewQueueStatus(m.reviewIdx, len(m.reviewCards))) + "\n\n")
		b.WriteString("Queue complete.\n")
		b.WriteString(m.theme.Muted.Render("Press esc to return to Browse and choose another deck."))
		if m.errText != "" {
			b.WriteString("\n\n" + m.theme.Error.Render("Review error: "+m.errText))
		}
		return b.String()
	}

	card := m.reviewCards[m.reviewIdx]
	elapsedDays := time.Now().UTC().Sub(card.LastReviewAt).Hours() / 24.0
	if card.LastReviewAt.IsZero() {
		elapsedDays = 0
	}
	predicted := srs.PredictRecall(elapsedDays, card.StabilityDays)
	progress := fmt.Sprintf("%s  Deck: %s  Stability(days): %.2f  Difficulty(0 easy-1 hard): %.2f  Pred. recall: %.2f", reviewQueueStatus(m.reviewIdx, len(m.reviewCards)), card.Deck, card.StabilityDays, card.Difficulty, predicted)
	b.WriteString(m.theme.StatusLine.Render(progress) + "\n\n")
	b.WriteString("Front\n")
	b.WriteString(m.theme.CardFront.Render(card.Front) + "\n\n")
	if m.revealed {
		b.WriteString("Back\n")
		b.WriteString(m.theme.CardBack.Render(card.Back) + "\n\n")
		b.WriteString(m.theme.Muted.Render("Grade now: 1 Again, 2 Hard, 3 Good, 4 Easy"))
	} else {
		b.WriteString(m.theme.Muted.Render("Press space to reveal answer"))
	}
	if m.errText != "" {
		b.WriteString("\n\n" + m.theme.Error.Render("Review error: "+m.errText))
	}
	return strings.TrimRight(b.String(), "\n")
}
