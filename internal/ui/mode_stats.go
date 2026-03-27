package ui

import (
	"fmt"
	"strings"
)

func (m Model) viewStats() string {
	var b strings.Builder
	b.WriteString(m.theme.Title.Render("Stats") + "\n\n")
	if m.stats.BackendName == "" {
		b.WriteString(m.theme.Muted.Render("Stats not loaded yet. Press tab away and back to refresh."))
		if m.errText != "" {
			b.WriteString("\n\n" + m.theme.Error.Render("Stats error: "+m.errText))
		}
		return strings.TrimRight(b.String(), "\n")
	}
	b.WriteString(fmt.Sprintf("Backend: %s\n", m.stats.BackendName))
	b.WriteString(fmt.Sprintf("Healthy: %t\n", m.stats.BackendHealthy))
	if m.stats.HealthErrorText != "" {
		b.WriteString(m.theme.Error.Render("Health error: "+m.stats.HealthErrorText) + "\n")
	}
	b.WriteString(fmt.Sprintf("Total cards: %d\n", m.stats.TotalCards))
	b.WriteString(fmt.Sprintf("Due today: %d\n", m.stats.DueToday))
	b.WriteString(fmt.Sprintf("New cards: %d\n", m.stats.NewCards))
	b.WriteString(fmt.Sprintf("Reviews today: %d\n", m.stats.ReviewsToday))
	if m.errText != "" {
		b.WriteString("\n\n" + m.theme.Error.Render("Stats error: "+m.errText))
	}
	return strings.TrimRight(b.String(), "\n")
}
