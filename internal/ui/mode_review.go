package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/shaoyanji/deutsch-srs/internal/srs"
)

func (m Model) viewReview() string {
	var b strings.Builder
	b.WriteString(m.theme.Title.Render("Review") + "\n\n")

	if len(m.reviewCards) == 0 {
		b.WriteString(m.theme.Muted.Render("Nothing due in this deck right now.") + "\n")
		if m.reviewNewPerDay > 0 {
			b.WriteString(m.theme.StatusLine.Render(fmt.Sprintf("New cards today: %d/%d", m.reviewNewToday, m.reviewNewPerDay)) + "\n")
			if m.reviewNewToday >= m.reviewNewPerDay {
				b.WriteString(m.theme.Muted.Render("Daily new-card limit reached. Come back tomorrow, or raise it:") + "\n")
				b.WriteString(m.theme.Muted.Render("  deutsch-srs-cli limit --deck \""+m.reviewDeck+"\" --new-per-day N") + "\n")
			}
		}
		b.WriteString("\n" + m.theme.Muted.Render("Press esc to choose another deck.") + "\n")
		if m.errText != "" {
			b.WriteString("\n" + m.theme.Error.Render("Review error: "+m.errText))
		}
		return strings.TrimRight(b.String(), "\n")
	}

	if m.reviewIdx >= len(m.reviewCards) {
		b.WriteString(m.theme.StatusLine.Render(reviewQueueStatus(m.reviewIdx, len(m.reviewCards))) + "\n\n")
		b.WriteString(fmt.Sprintf("Batch complete: %d cards.\n", len(m.reviewCards)))
		b.WriteString(m.theme.Muted.Render("Press enter for the next batch, esc to return to the deck list."))
		if m.reviewNewPerDay > 0 {
			b.WriteString("\n\n" + m.theme.StatusLine.Render(fmt.Sprintf("New cards today: %d/%d", m.reviewNewToday, m.reviewNewPerDay)))
		}
		return strings.TrimRight(b.String(), "\n")
	}

	card := m.reviewCards[m.reviewIdx]
	elapsed := time.Now().UTC().Sub(card.LastReviewAt).Hours() / 24.0
	if card.LastReviewAt.IsZero() {
		elapsed = 0
	}
	predicted := srs.PredictRecall(elapsed, card.StabilityDays)

	status := fmt.Sprintf("%s  Deck: %s", reviewQueueStatus(m.reviewIdx, len(m.reviewCards)), card.Deck)
	if m.reviewNewPerDay > 0 {
		status += fmt.Sprintf("  new today: %d/%d", m.reviewNewToday, m.reviewNewPerDay)
	}
	b.WriteString(m.theme.StatusLine.Render(status) + "\n")
	b.WriteString(m.theme.Muted.Render(fmt.Sprintf("Stability %.2fd | Difficulty %.2f | Pred. recall %.2f", card.StabilityDays, card.Difficulty, predicted)) + "\n\n")
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
