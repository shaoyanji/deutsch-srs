package ui

import (
	"fmt"
	"strconv"
	"strings"
)

// deckRowChrome is the width a deck row spends on padding and the counts column.
// One separator plus three %4s columns separated by single spaces.
const deckRowChrome = 1 + 4 + 1 + 4 + 1 + 4 + 2

// visibleWindow returns the slice of a list to render so the cursor stays on
// screen. A search can return a hundred rows; drawing all of them pushes the
// query line and the best hits off the top of the terminal.
func visibleWindow(cursor, total, rows int) (start, end int) {
	if rows < 1 {
		rows = 1
	}
	if total <= rows {
		return 0, total
	}
	start = cursor - rows + 1
	if start < 0 {
		start = 0
	}
	end = start + rows
	if end > total {
		end = total
		start = end - rows
	}
	return start, end
}

func reviewQueueStatus(idx, total int) string {
	if total <= 0 {
		return "Queue: empty"
	}
	if idx >= total {
		return "Queue: complete"
	}
	current := idx + 1
	remaining := total - current
	return "Queue: card " + itoa(current) + "/" + itoa(total) + "  remaining " + itoa(remaining)
}

// deckRow lays out one deck as name plus fixed-width counts, so a five-digit
// count cannot push the row past the pane and wrap it onto a second line.
func deckRow(name string, due, newCount, rev, nameWidth int) string {
	counts := fmt.Sprintf("%4s %4s %4s", compactCount(due), compactCount(newCount), compactCount(rev))
	return padRight(truncateRunes(name, nameWidth), nameWidth) + " " + counts
}

// deckHeader labels the count columns so the rows can stay terse.
func deckHeader(nameWidth int) string {
	return padRight("deck", nameWidth) + fmt.Sprintf(" %4s %4s %4s", "due", "new", "rev")
}

// compactCount shortens large numbers so the review backlog stays one row wide.
func compactCount(n int) string {
	switch {
	case n <= 0:
		return "0"
	case n < 1000:
		return strconv.Itoa(n)
	case n < 10000:
		return strconv.FormatFloat(float64(n)/1000, 'f', 1, 64) + "k"
	case n < 1000000:
		return strconv.Itoa(n/1000) + "k"
	default:
		return strconv.Itoa(n/1000000) + "M"
	}
}

func truncateRunes(s string, width int) string {
	if width <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= width {
		return s
	}
	if width <= 3 {
		return string(r[:width])
	}
	return string(r[:width-1]) + "…"
}

func padRight(s string, width int) string {
	if n := width - len([]rune(s)); n > 0 {
		return s + strings.Repeat(" ", n)
	}
	return s
}

func addRequiredFilled(deck, front, back string) int {
	n := 0
	if strings.TrimSpace(deck) != "" {
		n++
	}
	if strings.TrimSpace(front) != "" {
		n++
	}
	if strings.TrimSpace(back) != "" {
		n++
	}
	return n
}

func addFormIsEmpty(values []string) bool {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return false
		}
	}
	return true
}

func itoa(v int) string {
	return strconv.Itoa(v)
}
