package styles

import "charm.land/lipgloss/v2"

// BlameStyles holds all lipgloss styles used by the blame view.
type BlameStyles struct {
	TitleBar  lipgloss.Style
	StatusBar lipgloss.Style
	Cursor    lipgloss.Style
	Hash      lipgloss.Style
	Date      lipgloss.Style
	Author    lipgloss.Style
	LineNum   lipgloss.Style
	Loading   lipgloss.Style
	Error     lipgloss.Style
	Row       lipgloss.Style
}

// Default returns the default BlameStyles.
func Default() BlameStyles {
	return BlameStyles{
		TitleBar: lipgloss.NewStyle().
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("230")).
			MaxHeight(1).
			Padding(0, 1, 0, 1).
			Bold(true),

		StatusBar: lipgloss.NewStyle().
			Background(lipgloss.Color("237")).
			Foreground(lipgloss.Color("250")).
			MaxHeight(1).
			Padding(0, 1, 0, 1),

		Cursor: lipgloss.NewStyle().
			Background(lipgloss.Color("24")).
			Foreground(lipgloss.Color("255")),

		Hash: lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")),

		Date: lipgloss.NewStyle().
			Foreground(lipgloss.Color("108")),

		Author: lipgloss.NewStyle().
			Foreground(lipgloss.Color("183")),

		LineNum: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),

		Loading: lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")),

		Error: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true),

		Row: lipgloss.NewStyle().
			Height(1).
			MaxHeight(1),
	}
}
