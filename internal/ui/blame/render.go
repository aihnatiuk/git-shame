package blame

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/aihnatiuk/git-shame/internal/git"
	"github.com/aihnatiuk/git-shame/internal/ui/styles"
	"github.com/charmbracelet/x/ansi"

	"charm.land/lipgloss/v2"
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
	pad := max(width-runewidth.StringWidth(status), 0)
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

	start := m.vScrollOffset
	end := min(m.vScrollOffset+m.bodyHeight, len(m.lines))

	rowsList := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		row := RenderRow(m.lines[i], m.columns, i == m.cursor, m.hScrollOffset, m.styles, m.bodyWidth)
		rowsList = append(rowsList, row)
	}

	return lipgloss.JoinVertical(lipgloss.Position(0), rowsList...)
}

// RenderRow renders a single blame line as a formatted row string.
//
// Cursor background is applied per-cell rather than as an outer wrapper so
// that individual cell resets (\x1b[0m) do not erase the highlight between
// columns. Horizontal scroll (hScroll) is applied only to the Code column so
// that metadata columns remain fixed while code scrolls.
func RenderRow(
	line git.BlameLine,
	cols []Column,
	cursor bool,
	hScroll int,
	s styles.BlameStyles,
	rowWidth int,
) string {
	var cursorBg color.Color
	sep := " "
	if cursor {
		cursorBg = s.Cursor.GetBackground()
		sep = lipgloss.NewStyle().Background(cursorBg).Render(sep)
	}

	var cells []string
	for _, col := range cols {
		if !col.Visible || col.Width == 0 {
			continue
		}
		cell := renderCell(line, col, hScroll, cursorBg, s)
		cells = append(cells, cell)
	}
	row := strings.Join(cells, sep)

	return row
}

// renderCell returns the styled string for a single column cell.
// cursorBg is non-nil when the row is the cursor row; each cell then
// incorporates the cursor background so the highlight spans all columns.
// hScroll is applied only to ColCode via ansi.TruncateLeft so that ANSI
// escape codes (e.g. from syntax highlighting in Phase 2) are preserved.
func renderCell(
	line git.BlameLine,
	col Column,
	hScroll int,
	cursorBg color.Color,
	s styles.BlameStyles,
) string {
	switch col.ID {
	case ColHash:
		hash := line.CommitHash
		if len(hash) > 8 {
			hash = hash[:8]
		}
		return withBg(s.Hash, cursorBg).
			MaxWidth(col.Width).
			Render(hash)
	case ColDate:
		return withBg(s.Date, cursorBg).
			MaxWidth(col.Width).
			Render(line.AuthorTime.Format("2006-01-02"))
	case ColAuthor:
		return withBg(s.Author, cursorBg).
			Width(col.Width).
			MaxWidth(col.Width).
			Render(line.Author)
	case ColSummary:
		return withBg(lipgloss.NewStyle(), cursorBg).
			MaxWidth(col.Width).
			Render(line.Summary)
	case ColLineNum:
		return withBg(s.LineNum, cursorBg).Render(fmt.Sprintf("%*d", col.Width, line.LineNum))
	case ColCode:
		content := ansi.TruncateLeft(line.Content, hScroll, "")
		padding := max(0, col.Width-lipgloss.Width(content))
		codeStyle := lipgloss.NewStyle().
			MaxWidth(col.Width).
			PaddingRight(padding)
		if cursorBg != nil {
			codeStyle = codeStyle.Inherit(s.Cursor)
		}
		return codeStyle.Render(content)
	case ColFilename:
		return withBg(lipgloss.NewStyle(), cursorBg).
			MaxWidth(col.Width).
			Render(line.Filename)
	}
	return ""
}

// withBg returns st with cursorBg applied when non-nil, otherwise st unchanged.
func withBg(style lipgloss.Style, cursorBg color.Color) lipgloss.Style {
	if cursorBg != nil {
		return style.Background(cursorBg)
	}
	return style
}
