package srs

import (
	"testing"
	"time"
)

func TestNextCard(t *testing.T) {
	now := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	base := Card{ID: "c1", Ease: 2.5, IntervalDays: 4, Reps: 3}

	tests := []struct {
		name      string
		grade     ReviewGrade
		wantReps  int
		wantLapse int
		wantInt   int
		wantDue   time.Time
	}{
		{
			name:      "again resets",
			grade:     GradeAgain,
			wantReps:  0,
			wantLapse: 1,
			wantInt:   0,
			wantDue:   now.Add(10 * time.Minute),
		},
		{
			name:      "hard modest growth",
			grade:     GradeHard,
			wantReps:  4,
			wantLapse: 0,
			wantInt:   5,
			wantDue:   now.AddDate(0, 0, 5),
		},
		{
			name:      "good normal growth",
			grade:     GradeGood,
			wantReps:  4,
			wantLapse: 0,
			wantInt:   10,
			wantDue:   now.AddDate(0, 0, 10),
		},
		{
			name:      "easy larger growth",
			grade:     GradeEasy,
			wantReps:  4,
			wantLapse: 0,
			wantInt:   11,
			wantDue:   now.AddDate(0, 0, 11),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NextCard(base, tt.grade, now)
			if err != nil {
				t.Fatalf("NextCard error: %v", err)
			}
			if got.Reps != tt.wantReps {
				t.Fatalf("reps=%d want=%d", got.Reps, tt.wantReps)
			}
			if got.Lapses != tt.wantLapse {
				t.Fatalf("lapses=%d want=%d", got.Lapses, tt.wantLapse)
			}
			if got.IntervalDays != tt.wantInt {
				t.Fatalf("interval=%d want=%d", got.IntervalDays, tt.wantInt)
			}
			if !got.DueAt.Equal(tt.wantDue) {
				t.Fatalf("due=%s want=%s", got.DueAt, tt.wantDue)
			}
		})
	}
}

func TestNextCardInvalidGrade(t *testing.T) {
	_, err := NextCard(Card{Ease: 2.5}, ReviewGrade(42), time.Now())
	if err == nil {
		t.Fatal("expected error for invalid grade")
	}
}
