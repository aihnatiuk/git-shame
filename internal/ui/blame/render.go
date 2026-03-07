package blame

import (
	"fmt"
	"strings"

	"github.com/aihnatiuk/git-shame/internal/git"
	"github.com/aihnatiuk/git-shame/internal/ui/styles"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// RenderTitleBar renders the top bar showing the current file and revision.
func RenderTitleBar(file, revision string, width int, s styles.BlameStyles) string {
	rev := revision
	if rev == "" {
		rev = "HEAD"
	}
	title := fmt.Sprintf("  %s @ %s  ", file, rev)
	// Pad to full terminal width so the background fills the bar.
	pad := width - runewidth.StringWidth(title)
	if pad > 0 {
		title += strings.Repeat(" ", pad)
	}
	return s.TitleBar.Render(title)
}

// RenderStatusBar renders the bottom bar with cursor position info.
func RenderStatusBar(cursor, total, width int, s styles.BlameStyles) string {
	if total == 0 {
		return s.StatusBar.Render(strings.Repeat(" ", width))
	}
	pct := (cursor + 1) * 100 / total
	status := fmt.Sprintf("  %d/%d  %d%%  ", cursor+1, total, pct)
	pad := width - runewidth.StringWidth(status)
	if pad < 0 {
		pad = 0
	}
	line := strings.Repeat(" ", pad) + status
	return s.StatusBar.Render(line)
}

// RenderBody renders the visible rows of the blame table.
func RenderBody(m *Model) string {
	if m.state == LoadStateLoading {
		return m.spinner.View() + "  Loading blame…"
	}
	if m.state == LoadStateError {
		return m.styles.Error.Render("Error: " + m.loadErr.Error())
	}
	if len(m.lines) == 0 {
		return m.styles.Loading.Render("(no data)")
	}

	start := m.offset
	end := min(m.offset+m.bodyH, len(m.lines))

	var sb strings.Builder
	for i := start; i < end; i++ {
		row := RenderRow(m.lines[i], m.columns, i == m.cursor, m.hScroll, m.styles)
		sb.WriteString(row)
		if i < end-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

// RenderRow renders a single blame line as a formatted row string.
//
// Cursor background is applied per-cell rather than as an outer wrapper so
// that individual cell resets (\x1b[0m) do not erase the highlight between
// columns.
func RenderRow(
	line git.BlameLine,
	cols []Column,
	cursor bool,
	hScroll int,
	s styles.BlameStyles,
) string {
	var cursorBg lipgloss.TerminalColor
	sep := s.Separator
	if cursor {
		cursorBg = s.Cursor.GetBackground()
		sep = lipgloss.NewStyle().Background(cursorBg).Render(sep)
	}

	var cells []string
	for _, col := range cols {
		if !col.Visible || col.Width == 0 {
			continue
		}
		cell := renderCell(line, col, cursorBg, s)
		// Truncate/pad to the exact column width (ANSI-aware via lipgloss).
		cell = truncatePad(cell, col.Width, cursorBg)
		cells = append(cells, cell)
	}
	row := strings.Join(cells, sep)

	// Apply horizontal scroll: strip leading hScroll visible columns from row.
	if hScroll > 0 {
		row = scrollRight(row, hScroll)
	}

	return row
}

// renderCell returns the styled string for a single column cell.
// cursorBg is non-nil when the row is the cursor row; each cell then
// incorporates the cursor background so the highlight spans all columns.
func renderCell(
	line git.BlameLine,
	col Column,
	cursorBg lipgloss.TerminalColor,
	s styles.BlameStyles,
) string {
	switch col.ID {
	case ColHash:
		hash := line.CommitHash
		if len(hash) > 8 {
			hash = hash[:8]
		}
		return withBg(s.Hash, cursorBg).Render(hash)
	case ColDate:
		return withBg(s.Date, cursorBg).Render(line.AuthorTime.Format("2006-01-02"))
	case ColAuthor:
		return withBg(s.Author, cursorBg).Render(line.Author)
	case ColSummary:
		return withBg(lipgloss.NewStyle(), cursorBg).Render(line.Summary)
	case ColLineNum:
		return withBg(s.LineNum, cursorBg).Render(fmt.Sprintf("%*d", col.Width, line.LineNum))
	case ColCode:
		defaultTabWidth := 4
		content := expandTabs(line.Content, defaultTabWidth)
		if cursorBg != nil {
			return s.Cursor.Render(content)
		}
		return content
	case ColFilename:
		return withBg(lipgloss.NewStyle(), cursorBg).Render(line.Filename)
	}
	return ""
}

// withBg returns st with cursorBg applied when non-nil, otherwise st unchanged.
func withBg(st lipgloss.Style, cursorBg lipgloss.TerminalColor) lipgloss.Style {
	if cursorBg != nil {
		return st.Background(cursorBg)
	}
	return st
}

// expandTabs replaces tabs with spaces for consistent width measurement and rendering.
func expandTabs(s string, tabWidth int) string {
	return strings.ReplaceAll(s, "\t", strings.Repeat(" ", tabWidth))
}

// truncatePad truncates or right-pads text to exactly width visible columns.
// It is ANSI-escape-code aware via lipgloss.Width for width measurement.
// When cursorBg is non-nil any added padding spaces carry the cursor background
// colour so the row highlight is unbroken.
func truncatePad(text string, width int, cursorBg lipgloss.TerminalColor) string {
	textWidth := lipgloss.Width(text)
	if textWidth == width {
		return text
	}
	if textWidth > width {
		return runewidth.Truncate(text, width, "")
	}
	// Pad with spaces, optionally styled with the cursor background.
	pad := strings.Repeat(" ", width-textWidth)
	if cursorBg != nil {
		pad = lipgloss.NewStyle().Background(cursorBg).Render(pad)
	}
	return text + pad
}

// scrollRight strips hScroll visible columns from the left of the string.
func scrollRight(s string, hScroll int) string {
	runes := []rune(s)
	skipped := 0
	start := 0
	for start < len(runes) {
		w := runewidth.RuneWidth(runes[start])
		if skipped+w > hScroll {
			break
		}
		skipped += w
		start++
	}
	return string(runes[start:])
}
