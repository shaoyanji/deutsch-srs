package ui

import (
	"strings"
	"testing"
)

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

func TestCompactCount(t *testing.T) {
	tests := map[int]string{
		0: "0", 7: "7", 999: "999", 1000: "1.0k", 1500: "1.5k",
		77132: "77k", 999999: "999k", 2000000: "2M",
	}
	for in, want := range tests {
		if got := compactCount(in); got != want {
			t.Fatalf("compactCount(%d)=%q want=%q", in, got, want)
		}
	}
}

// A 77k-card deck used to wrap its row onto a second line in the left pane.
func TestDeckRowStaysInsidePane(t *testing.T) {
	const paneContent = 38 // 120-column terminal, left pane is a third of it
	row := deckRow("Wortschatz B2", 77132, 77132, 0, 15)
	if got := len([]rune(row)); got > paneContent {
		t.Fatalf("row is %d cells wide, pane holds %d: %q", got, paneContent, row)
	}
	if !strings.Contains(row, "77k") {
		t.Fatalf("expected compact due count in %q", row)
	}
}

func TestDeckRowTruncatesLongName(t *testing.T) {
	row := deckRow("Ein sehr langer Stapelname fuer B2 Schreiben", 3, 1, 0, 12)
	name := strings.SplitN(row, " ", 2)[0]
	if runes := []rune(name); len(runes) > 12 {
		t.Fatalf("name column overflows: %q", name)
	}
}
