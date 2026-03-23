package blame

import (
	"fmt"

	"charm.land/lipgloss/v2"
	"github.com/aihnatiuk/git-shame/internal/git"
	"github.com/mattn/go-runewidth"
)

// ColumnID identifies a column in the blame view.
type ColumnID int

const (
	ColHash     ColumnID = iota
	ColDate              // author date
	ColAuthor            // author name
	ColSummary           // commit summary / message first line
	ColLineNum           // file line number
	ColCode              // source code content (flex column)
	ColFilename          // filename (relevant when code moved between files)
)

// Column holds the configuration and computed display width for one table column.
type Column struct {
	ID        ColumnID
	Label     string
	Visible   bool
	Width     int // computed display width in terminal columns; 0 = flex (ColCode)
	MinWidth  int
	MaxWidth  int    // 0 = uncapped
	ToggleKey string // optional key shortcut for Phase 3 column menu
}

// defaultColumns returns the canonical ordered column slice with default settings.
// Widths are computed later by RecalcWidths once line data is available.
func defaultColumns() []Column {
	return []Column{
		{ID: ColHash, Label: "Hash", Visible: true, MinWidth: 8, MaxWidth: 8},
		{ID: ColDate, Label: "Date", Visible: true, MinWidth: 10, MaxWidth: 10},
		{ID: ColAuthor, Label: "Author", Visible: true, MinWidth: 8, MaxWidth: 20},
		{ID: ColSummary, Label: "Summary", Visible: false, MinWidth: 10, MaxWidth: 40},
		{ID: ColLineNum, Label: "#", Visible: true, MinWidth: 3, MaxWidth: 6},
		{ID: ColCode, Label: "Code", Visible: true, MinWidth: 10, MaxWidth: 0}, // flex
		{ID: ColFilename, Label: "File", Visible: false, MinWidth: 8, MaxWidth: 30},
	}
}

// RecalcWidths recomputes column widths based on the terminal width and actual
// line data. Must be called after BlameResult arrives and on WindowSizeMsg.
// If lines is nil the function returns cols unchanged (called again when data loads).
func RecalcWidths(cols []Column, lines []git.BlameLine, termWidth int) []Column {
	if len(lines) == 0 {
		return cols
	}

	out := make([]Column, len(cols))
	copy(out, cols)

	// Measure the widest author name in actual data.
	maxAuthor := 0
	for _, line := range lines {
		if authorWidth := runewidth.StringWidth(line.Author); authorWidth > maxAuthor {
			maxAuthor = authorWidth
		}
	}

	// Calculate the number of digits needed for the largest line number.
	lineNumWidth := len(fmt.Sprintf("%d", len(lines)))

	used := 0 // total columns consumed by non-flex visible columns + separators
	flexIdx := -1

	for i := range out {
		if !out[i].Visible {
			out[i].Width = 0
			continue
		}
		var w int
		switch out[i].ID {
		case ColHash:
			w = 8
		case ColDate:
			w = 10 // "2006-01-02"
		case ColAuthor:
			w = clamp(maxAuthor, out[i].MinWidth, out[i].MaxWidth)
		case ColSummary:
			w = out[i].MaxWidth
			if w == 0 {
				w = out[i].MinWidth
			}
		case ColLineNum:
			w = clamp(lineNumWidth, out[i].MinWidth, out[i].MaxWidth)
		case ColFilename:
			w = out[i].MaxWidth
			if w == 0 {
				w = out[i].MinWidth
			}
		case ColCode:
			flexIdx = i
			continue
		}
		out[i].Width = w
		used += w + 1 // +1 for separator
	}

	// Give remaining width to the flex (Code) column.
	if flexIdx >= 0 {
		remaining := max(termWidth-used, out[flexIdx].MinWidth)
		out[flexIdx].Width = remaining
	}

	return out
}

// CalcMaxHScroll returns the maximum allowed horizontal scroll offset for the
// Code column: the number of columns by which the longest line exceeds the
// visible code column width. A terminal resize or new blame load must call
// this again because both the line content and the code column width can change.
func CalcMaxHScroll(cols []Column, lines []git.BlameLine) int {
	codeWidth := 0
	for _, col := range cols {
		if col.ID == ColCode && col.Visible {
			codeWidth = col.Width
			break
		}
	}

	if codeWidth == 0 || len(lines) == 0 {
		return 0
	}

	maxContent := 0
	for _, line := range lines {
		if w := lipgloss.Width(line.Content); w > maxContent {
			maxContent = w
		}
	}

	if maxContent <= codeWidth {
		return 0
	}

	return maxContent - codeWidth
}

func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if max > 0 && v > max {
		return max
	}
	return v
}
