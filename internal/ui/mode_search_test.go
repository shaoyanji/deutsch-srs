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
