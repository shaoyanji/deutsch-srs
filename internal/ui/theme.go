package ui

import "github.com/charmbracelet/lipgloss"

type Theme struct {
	App        lipgloss.Style
	LeftPane   lipgloss.Style
	RightPane  lipgloss.Style
	Title      lipgloss.Style
	Muted      lipgloss.Style
	Selected   lipgloss.Style
	Error      lipgloss.Style
	HelpBox    lipgloss.Style
	CardFront  lipgloss.Style
	CardBack   lipgloss.Style
	StatusLine lipgloss.Style
}

func DefaultTheme() Theme {
	border := lipgloss.RoundedBorder()
	return Theme{
		App:       lipgloss.NewStyle().Padding(0, 1),
		LeftPane:  lipgloss.NewStyle().Border(border).BorderForeground(lipgloss.Color("239")).Padding(0, 1),
		RightPane: lipgloss.NewStyle().Border(border).BorderForeground(lipgloss.Color("239")).Padding(0, 1),
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("252")),
		Muted:      lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
		Selected:   lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Background(lipgloss.Color("238")),
		Error:      lipgloss.NewStyle().Foreground(lipgloss.Color("167")).Bold(true),
		HelpBox:    lipgloss.NewStyle().Border(border).BorderForeground(lipgloss.Color("241")).Padding(1, 2),
		CardFront:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252")),
		CardBack:   lipgloss.NewStyle().Foreground(lipgloss.Color("250")),
		StatusLine: lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
	}
}
