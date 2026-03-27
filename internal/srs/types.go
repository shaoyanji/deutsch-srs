package srs

import "time"

type ReviewGrade int

const (
	GradeAgain ReviewGrade = 1
	GradeHard  ReviewGrade = 2
	GradeGood  ReviewGrade = 3
	GradeEasy  ReviewGrade = 4
)

type Card struct {
	ID           string
	Deck         string
	Front        string
	Back         string
	NoteType     string
	Tags         []string
	DueAt        time.Time
	IntervalDays int
	Ease         float64
	Reps         int
	Lapses       int
	Suspended    bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Deck struct {
	Name        string
	DueCount    int
	NewCount    int
	ReviewCount int
}
