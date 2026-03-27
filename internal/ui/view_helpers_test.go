package ui

import "testing"

func TestReviewQueueStatus(t *testing.T) {
	tests := []struct {
		name  string
		idx   int
		total int
		want  string
	}{
		{name: "empty", idx: 0, total: 0, want: "Queue: empty"},
		{name: "first card", idx: 0, total: 5, want: "Queue: card 1/5  remaining 4"},
		{name: "middle card", idx: 2, total: 5, want: "Queue: card 3/5  remaining 2"},
		{name: "complete", idx: 5, total: 5, want: "Queue: complete"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reviewQueueStatus(tt.idx, tt.total)
			if got != tt.want {
				t.Fatalf("got=%q want=%q", got, tt.want)
			}
		})
	}
}

func TestAddHelpers(t *testing.T) {
	if got := addRequiredFilled("deck", "front", ""); got != 2 {
		t.Fatalf("required filled=%d want=2", got)
	}
	if !addFormIsEmpty([]string{"", "  ", "\t"}) {
		t.Fatal("expected form to be empty")
	}
	if addFormIsEmpty([]string{"deck", "", ""}) {
		t.Fatal("expected form to be non-empty")
	}
}
