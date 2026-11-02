package ui

import "testing"

func TestTrimOneLine(t *testing.T) {
	if got := trimOneLine("Hallo\nWelt", 20); got != "Hallo Welt" {
		t.Fatalf("got=%q", got)
	}
	if got := trimOneLine("abcdefghijklmnopqrstuvwxyz", 10); got != "abcdefg..." {
		t.Fatalf("got=%q", got)
	}
}

func TestVisibleWindowKeepsCursorOnScreen(t *testing.T) {
	start, end := visibleWindow(0, 100, 10)
	if start != 0 || end != 10 {
		t.Fatalf("cursor at top: window %d-%d", start, end)
	}

	start, end = visibleWindow(50, 100, 10)
	if 50 < start || 50 >= end {
		t.Fatalf("cursor 50 outside %d-%d", start, end)
	}
	if end-start != 10 {
		t.Fatalf("window is %d rows, want 10", end-start)
	}

	start, end = visibleWindow(99, 100, 10)
	if end != 100 || end-start != 10 {
		t.Fatalf("cursor at bottom: window %d-%d", start, end)
	}

	start, end = visibleWindow(2, 5, 10)
	if start != 0 || end != 5 {
		t.Fatalf("short list: window %d-%d", start, end)
	}
}
