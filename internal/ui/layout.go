package ui

import "github.com/charmbracelet/lipgloss"

func (m *Model) paneWidths() (int, int) {
	total := m.width
	if total <= 0 {
		total = 100
	}
	left := total / 3
	if left < 24 {
		left = 24
	}
	right := total - left - 1
	if right < 40 {
		right = 40
	}
	return left, right
}

func joinHorizontal(left, right string) string {
	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}
