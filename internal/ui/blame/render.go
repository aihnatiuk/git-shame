package blame

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/aihnatiuk/git-shame/internal/git"
	"github.com/aihnatiuk/git-shame/internal/highlight"
	"github.com/aihnatiuk/git-shame/internal/ui/styles"
	"github.com/charmbracelet/x/ansi"

	"charm.land/lipgloss/v2"
)

// RenderTitleBar renders the top bar showing the current file and revision.
func RenderTitleBar(file, revision string, width int, s styles.BlameStyles) string {
	withWidth := s.TitleBar.Width(width)
	contentWidth := width - withWidth.GetHorizontalPadding()
	if revision == "" {
		revision = "HEAD"
	}
	title := fmt.Sprintf("%s @ %s", file, revision)

	return withWidth.Render(ansi.Truncate(title, contentWidth, "…"))
}

// RenderStatusBar renders the bottom bar with cursor position info.
// When statusMsg is non-empty it is shown instead of the position counter.
func RenderStatusBar(m *Model) string {
	s := m.styles
	withWidth := s.StatusBar.Width(m.bodyWidth)
	contentWidth := m.bodyWidth - withWidth.GetHorizontalPadding()

	if m.state == LoadStateLoading {
		return withWidth.Render(m.spinner.View() + " Loading blame")
	}
	if m.statusMessage != "" {
		return withWidth.Render(ansi.Truncate(m.statusMessage, contentWidth, "…"))
	}
	total := len(m.lines)
	if total == 0 {
		return s.StatusBar.Render(" ")
	}

	pct := (m.cursor + 1) * 100 / total
	status := fmt.Sprintf("%d/%d  %d%%", m.cursor+1, total, pct)

	return withWidth.
		AlignHorizontal(lipgloss.Position(1)).
		Render(status)
}

// RenderBody renders the visible rows of the blame table.
func RenderBody(m *Model) string {
	if m.state == LoadStateError {
		return m.styles.Error.Render("Error: " + m.loadErr.Error())
	}
	if len(m.lines) == 0 {
		return m.styles.Loading.Render("(no data)")
	}

	start := m.vScrollOffset
	end := min(m.vScrollOffset+m.bodyHeight, len(m.lines))

	rowsList := make([]string, 0, end-start)
	rowStyle := m.styles.Row.MaxWidth(m.bodyWidth)
	for i := start; i < end; i++ {
		isActiveRow := i == m.cursor
		var highlighted string
		if i < len(m.highlightedLines) {
			highlighted = m.highlightedLines[i]
		} else {
			highlighted = m.lines[i].Content
		}
		row := RenderRow(m.lines[i], highlighted, m.columns, isActiveRow, m.hScrollOffset, m.styles)
		rowsList = append(rowsList, rowStyle.Render(row))
	}

	return lipgloss.NewStyle().Height(m.bodyHeight).Render(
		lipgloss.JoinVertical(lipgloss.Position(0), rowsList...),
	)
}

// RenderRow renders a single blame line as a formatted row string.
//
// Cursor background is applied per-cell rather than as an outer wrapper so
// that individual cell resets (\x1b[0m) do not erase the highlight between
// columns. Horizontal scroll (hScroll) is applied only to the Code column so
// that metadata columns remain fixed while code scrolls.
func RenderRow(
	line git.BlameLine,
	highlightedContent string,
	cols []Column,
	isActive bool,
	hScroll int,
	s styles.BlameStyles,
) string {
	var activeBg color.Color
	sep := " "
	if isActive {
		activeBg = s.Cursor.GetBackground()
		sep = lipgloss.NewStyle().Background(activeBg).Render(sep)
	}

	var cells []string
	for _, col := range cols {
		if !col.Visible || col.Width == 0 {
			continue
		}
		cell := renderCell(line, highlightedContent, col, hScroll, activeBg, s)
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
	highlightedContent string,
	col Column,
	hScroll int,
	activeBg color.Color,
	s styles.BlameStyles,
) string {
	switch col.ID {
	case ColHash:
		hash := line.CommitHash
		if len(hash) > 8 {
			hash = hash[:8]
		}
		return withBg(s.Hash, activeBg).
			MaxWidth(col.Width).
			Render(hash)
	case ColDate:
		return withBg(s.Date, activeBg).
			MaxWidth(col.Width).
			Render(line.AuthorTime.Format("2006-01-02"))
	case ColAuthor:
		return withBg(s.Author, activeBg).
			Width(col.Width).
			MaxWidth(col.Width).
			Render(line.Author)
	case ColSummary:
		return withBg(lipgloss.NewStyle(), activeBg).
			MaxWidth(col.Width).
			Render(line.Summary)
	case ColLineNum:
		return withBg(s.LineNum, activeBg).Render(fmt.Sprintf("%*d", col.Width, line.LineNum))
	case ColCode:
		content := ansi.TruncateLeft(highlightedContent, hScroll, "")
		padding := max(0, col.Width-lipgloss.Width(content))
		codeStyle := lipgloss.NewStyle().
			MaxWidth(col.Width).
			PaddingRight(padding)
		if activeBg != nil {
			content = highlight.PaintBackground(content, activeBg)
			codeStyle = codeStyle.Background(activeBg)
		}
		return codeStyle.Render(content)
	case ColFilename:
		return withBg(lipgloss.NewStyle(), activeBg).
			MaxWidth(col.Width).
			Render(line.Filename)
	}
	return ""
}

// withBg returns style with cursorBg applied when non-nil, otherwise style unchanged.
func withBg(style lipgloss.Style, cursorBg color.Color) lipgloss.Style {
	if cursorBg != nil {
		return style.Background(cursorBg)
	}
	return style
}
