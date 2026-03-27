package srs

import (
	"math"
	"testing"
	"time"
)

func TestPredictRecall(t *testing.T) {
	got := PredictRecall(1.0, 2.0)
	want := math.Exp(-0.5)
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("recall=%f want=%f", got, want)
	}
}

func TestNextIntervalDaysFromTargetRetention(t *testing.T) {
	got := NextIntervalDays(10, 0.90)
	want := int(math.Round(-10 * math.Log(0.90)))
	if got != want {
		t.Fatalf("interval=%d want=%d", got, want)
	}
}

func TestAgainLowersStabilityAndIncrementsLapse(t *testing.T) {
	now := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	card := Card{
		ID:            "c1",
		StabilityDays: 4.0,
		Difficulty:    0.35,
		LastReviewAt:  now.AddDate(0, 0, -3),
		Reps:          3,
		Lapses:        0,
	}
	got, err := UpdateAfterReview(card, GradeAgain, now)
	if err != nil {
		t.Fatalf("UpdateAfterReview: %v", err)
	}
	if got.StabilityDays >= card.StabilityDays {
		t.Fatalf("stability did not drop: before=%f after=%f", card.StabilityDays, got.StabilityDays)
	}
	if got.Lapses != 1 {
		t.Fatalf("lapses=%d want=1", got.Lapses)
	}
	if got.Reps != 0 {
		t.Fatalf("reps=%d want=0", got.Reps)
	}
}

func TestHardGoodEasyIntervalGrowthOrder(t *testing.T) {
	now := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	base := Card{
		ID:            "c1",
		StabilityDays: 2.0,
		Difficulty:    0.35,
		LastReviewAt:  now.AddDate(0, 0, -2),
		Reps:          3,
	}
	hard, err := UpdateAfterReview(base, GradeHard, now)
	if err != nil {
		t.Fatalf("hard err: %v", err)
	}
	good, err := UpdateAfterReview(base, GradeGood, now)
	if err != nil {
		t.Fatalf("good err: %v", err)
	}
	easy, err := UpdateAfterReview(base, GradeEasy, now)
	if err != nil {
		t.Fatalf("easy err: %v", err)
	}
	if !(hard.IntervalDays <= good.IntervalDays && good.IntervalDays <= easy.IntervalDays) {
		t.Fatalf("interval order wrong: hard=%d good=%d easy=%d", hard.IntervalDays, good.IntervalDays, easy.IntervalDays)
	}
}

func TestDifficultyClamp(t *testing.T) {
	now := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	card := Card{
		ID:            "c1",
		StabilityDays: 1.0,
		Difficulty:    0.99,
		LastReviewAt:  now.AddDate(0, 0, -1),
	}
	got, err := UpdateAfterReview(card, GradeHard, now)
	if err != nil {
		t.Fatalf("UpdateAfterReview: %v", err)
	}
	if got.Difficulty < 0 || got.Difficulty > 1 {
		t.Fatalf("difficulty out of range: %f", got.Difficulty)
	}
}

func TestNextCardInvalidGrade(t *testing.T) {
	_, err := NextCard(Card{StabilityDays: 1, Difficulty: 0.35}, ReviewGrade(42), time.Now())
	if err == nil {
		t.Fatal("expected invalid-grade error")
	}
}

func TestRepeatedGoodReviewsGrowStabilityAndIntervals(t *testing.T) {
	now := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	card := Card{
		ID:            "seq-1",
		StabilityDays: 1.0,
		Difficulty:    0.35,
		LastReviewAt:  now.AddDate(0, 0, -1),
		Reps:          0,
	}

	prevInterval := 0
	prevStability := card.StabilityDays
	for i := 0; i < 4; i++ {
		next, err := UpdateAfterReview(card, GradeGood, now)
		if err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
		if next.IntervalDays < prevInterval {
			t.Fatalf("step %d interval regressed: prev=%d next=%d", i, prevInterval, next.IntervalDays)
		}
		if next.StabilityDays <= prevStability {
			t.Fatalf("step %d stability did not grow: prev=%f next=%f", i, prevStability, next.StabilityDays)
		}
		prevInterval = next.IntervalDays
		prevStability = next.StabilityDays
		now = now.AddDate(0, 0, next.IntervalDays)
		card = next
	}
}

func TestAgainAfterGrowthResetsMemoryState(t *testing.T) {
	now := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	card := Card{
		ID:            "seq-2",
		StabilityDays: 1.0,
		Difficulty:    0.35,
		LastReviewAt:  now.AddDate(0, 0, -1),
		Reps:          0,
	}
	g1, err := UpdateAfterReview(card, GradeGood, now)
	if err != nil {
		t.Fatalf("good1: %v", err)
	}
	now = now.AddDate(0, 0, g1.IntervalDays)
	g2, err := UpdateAfterReview(g1, GradeGood, now)
	if err != nil {
		t.Fatalf("good2: %v", err)
	}

	now = now.AddDate(0, 0, g2.IntervalDays)
	again, err := UpdateAfterReview(g2, GradeAgain, now)
	if err != nil {
		t.Fatalf("again: %v", err)
	}
	if again.StabilityDays >= g2.StabilityDays {
		t.Fatalf("expected reset stability lower than grown stability: grown=%f again=%f", g2.StabilityDays, again.StabilityDays)
	}
	if again.Reps != 0 {
		t.Fatalf("again should reset reps, got %d", again.Reps)
	}
	if again.Lapses != g2.Lapses+1 {
		t.Fatalf("again should increment lapses: before=%d after=%d", g2.Lapses, again.Lapses)
	}
}
