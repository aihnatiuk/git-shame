package styles

import "charm.land/lipgloss/v2"

// DiffStyles holds all lipgloss styles used by the diff view.
type DiffStyles struct {
	TitleBar  lipgloss.Style
	StatusBar lipgloss.Style
	Error     lipgloss.Style
	Loading   lipgloss.Style
	Row       lipgloss.Style

	// Commit info header
	Hash   lipgloss.Style
	Author lipgloss.Style
	Date   lipgloss.Style

	DiffContext lipgloss.Style

	// Diff gutter
	HunkHeader        lipgloss.Style
	OldLineNum        lipgloss.Style
	NewLineNum        lipgloss.Style
	DiffAddedPrefix   lipgloss.Style
	DiffRemovedPrefix lipgloss.Style
}

// DefaultDiff returns the default DiffStyles.
func DefaultDiff() DiffStyles {
	return DiffStyles{
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

		Error: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true),

		Loading: lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")),

		Row: lipgloss.NewStyle().
			Height(1).
			MaxHeight(1),

		Hash: lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")),

		Author: lipgloss.NewStyle().
			Foreground(lipgloss.Color("183")),

		Date: lipgloss.NewStyle().
			Foreground(lipgloss.Color("108")),

		DiffContext: lipgloss.NewStyle(),

		HunkHeader: lipgloss.NewStyle().
			Foreground(lipgloss.Color("37")).
			Bold(true),

		OldLineNum: lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")),

		NewLineNum: lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")),

		DiffAddedPrefix: lipgloss.NewStyle().
			Foreground(lipgloss.Color("76")),

		DiffRemovedPrefix: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")),
	}
}

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
			Background(lipgloss.Color("24")),

		Hash: lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")),

		Date: lipgloss.NewStyle().
			Foreground(lipgloss.Color("108")),

		Author: lipgloss.NewStyle().
			Foreground(lipgloss.Color("183")),

		LineNum: lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")),

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
