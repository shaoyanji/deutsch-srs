package srs

import (
	"fmt"
	"math"
	"time"
)

const (
	minEase = 1.3
	maxEase = 2.8
)

func NextCard(card Card, grade ReviewGrade, now time.Time) (Card, error) {
	out := card
	out.UpdatedAt = now.UTC()
	base := card.IntervalDays
	if base < 1 {
		base = 1
	}

	switch grade {
	case GradeAgain:
		out.Lapses++
		out.Reps = 0
		out.IntervalDays = 0
		out.Ease = clampEase(card.Ease - 0.2)
		out.DueAt = now.UTC().Add(10 * time.Minute)
	case GradeHard:
		out.Reps = card.Reps + 1
		out.Ease = clampEase(card.Ease - 0.15)
		out.IntervalDays = maxInt(1, int(math.Round(float64(base)*1.2)))
		out.DueAt = now.UTC().AddDate(0, 0, out.IntervalDays)
	case GradeGood:
		out.Reps = card.Reps + 1
		out.Ease = clampEase(card.Ease)
		out.IntervalDays = maxInt(1, int(math.Round(float64(base)*out.Ease)))
		out.DueAt = now.UTC().AddDate(0, 0, out.IntervalDays)
	case GradeEasy:
		out.Reps = card.Reps + 1
		out.Ease = clampEase(card.Ease + 0.1)
		out.IntervalDays = maxInt(2, int(math.Round(float64(base)*(out.Ease+0.25))))
		out.DueAt = now.UTC().AddDate(0, 0, out.IntervalDays)
	default:
		return Card{}, fmt.Errorf("invalid review grade: %d", grade)
	}

	return out, nil
}

func clampEase(v float64) float64 {
	if v == 0 {
		v = 2.5
	}
	if v < minEase {
		return minEase
	}
	if v > maxEase {
		return maxEase
	}
	return v
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
