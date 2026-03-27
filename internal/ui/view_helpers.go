package ui

import (
	"strconv"
	"strings"
)

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
