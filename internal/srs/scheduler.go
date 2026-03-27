package srs

import (
	"fmt"
	"math"
	"time"
)

const (
	TargetRetention      = 0.90
	DefaultStabilityDays = 1.0
	DefaultDifficulty    = 0.35
	minStabilityDays     = 0.5
	minElapsedDays       = 1.0 / (24.0 * 60.0) // one minute
	minDifficulty        = 0.0
	maxDifficulty        = 1.0
)

func PredictRecall(elapsedDays, stabilityDays float64) float64 {
	if stabilityDays <= 0 {
		stabilityDays = DefaultStabilityDays
	}
	if elapsedDays < minElapsedDays {
		elapsedDays = minElapsedDays
	}
	return math.Exp(-elapsedDays / stabilityDays)
}

func NextIntervalDays(stabilityDays, targetRetention float64) int {
	if stabilityDays <= 0 {
		stabilityDays = DefaultStabilityDays
	}
	if targetRetention <= 0 || targetRetention >= 1 {
		targetRetention = TargetRetention
	}
	v := int(math.Round(-stabilityDays * math.Log(targetRetention)))
	if v < 1 {
		return 1
	}
	return v
}

func UpdateAfterReview(card Card, grade ReviewGrade, now time.Time) (Card, error) {
	out := card
	now = now.UTC()

	stability := card.StabilityDays
	if stability <= 0 {
		stability = DefaultStabilityDays
	}
	difficulty := clampDifficulty(card.Difficulty)

	last := card.LastReviewAt
	if last.IsZero() {
		if !card.UpdatedAt.IsZero() {
			last = card.UpdatedAt
		} else if !card.CreatedAt.IsZero() {
			last = card.CreatedAt
		} else {
			last = now
		}
	}

	elapsedDays := now.Sub(last.UTC()).Hours() / 24.0
	r := PredictRecall(elapsedDays, stability)

	switch grade {
	case GradeAgain:
		stability = math.Max(minStabilityDays, stability*0.25)
		difficulty = clampDifficulty(difficulty + 0.08)
		out.Reps = 0
		out.Lapses++
	case GradeHard:
		stability = stability * (1.15 + 0.60*(1-r)*(1-difficulty))
		difficulty = clampDifficulty(difficulty + 0.03)
		out.Reps = card.Reps + 1
	case GradeGood:
		stability = stability * (1.40 + 1.20*(1-r)*(1-difficulty))
		difficulty = clampDifficulty(difficulty - 0.01)
		out.Reps = card.Reps + 1
	case GradeEasy:
		stability = stability * (1.70 + 1.50*(1-r)*(1-difficulty))
		difficulty = clampDifficulty(difficulty - 0.03)
		out.Reps = card.Reps + 1
	default:
		return Card{}, fmt.Errorf("invalid review grade: %d", grade)
	}

	intervalDays := NextIntervalDays(stability, TargetRetention)
	out.StabilityDays = stability
	out.Difficulty = difficulty
	out.LastReviewAt = now
	out.IntervalDays = intervalDays
	out.Ease = compatibilityEaseFromDifficulty(difficulty)
	out.DueAt = now.AddDate(0, 0, intervalDays)
	out.UpdatedAt = now
	return out, nil
}

func NextCard(card Card, grade ReviewGrade, now time.Time) (Card, error) {
	return UpdateAfterReview(card, grade, now)
}

func clampDifficulty(v float64) float64 {
	if v < minDifficulty {
		return minDifficulty
	}
	if v > maxDifficulty {
		return maxDifficulty
	}
	return v
}

func compatibilityEaseFromDifficulty(difficulty float64) float64 {
	// Keep legacy field populated for compatibility/display while stability/difficulty drive scheduling.
	ease := 2.8 - (clampDifficulty(difficulty) * 1.5)
	if ease < 1.3 {
		return 1.3
	}
	if ease > 2.8 {
		return 2.8
	}
	return ease
}
