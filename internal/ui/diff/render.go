package diff

import (
	"fmt"
	"strings"

	"github.com/aihnatiuk/git-shame/internal/git"
	"github.com/aihnatiuk/git-shame/internal/ui/commitinfo"
	"github.com/aihnatiuk/git-shame/internal/ui/styles"
	"github.com/charmbracelet/x/ansi"

	"charm.land/lipgloss/v2"
)

// RenderTitleBar renders the top bar showing the file path and commit hash.
func RenderTitleBar(relFile, hash string, width int, s styles.DiffStyles) string {
	withWidth := s.TitleBar.Width(width)
	contentWidth := width - withWidth.GetHorizontalPadding()
	title := fmt.Sprintf("%s @ %s", relFile, hash)
	return withWidth.Render(ansi.Truncate(title, contentWidth, styles.Ellipsis))
}

// RenderHeader renders the commit metadata header. During loading it shows the
// spinner; once loaded it uses commitinfo.Render.
func RenderHeader(m *Model) string {
	switch m.state {
	case LoadStateLoading:
		return m.spinner.View() + " Loading diff"
	case LoadStateError:
		return m.styles.Error.Render("Error: " + m.loadErr.Error())
	case LoadStateLoaded:
		return commitinfo.Render(m.commit, 5, m.terminalWidth, m.styles)
	}
	return ""
}

// RenderBody renders the visible portion of the diff.
func RenderBody(m *Model) string {
	if m.state != LoadStateLoaded {
		return ""
	}
	if len(m.diff.Hunks) == 0 {
		return m.styles.Loading.Render("No changes to this file in this commit")
	}

	lineNumWidth := calcLineNumWidth(m.allDiffLines)
	gutterWidth := lineNumWidth*2 + 4 // oldNum + " " + newNum + " " + prefix + " "
	contentWidth := max(m.terminalWidth-gutterWidth, 0)

	start := m.vScrollOffset
	end := min(m.vScrollOffset+m.bodyHeight, len(m.allDiffLines))

	rows := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		dl := m.allDiffLines[i]
		highlighted := m.highlightedLines[i]
		row := renderDiffLine(dl, highlighted, lineNumWidth, contentWidth, m.styles)
		rows = append(rows, row)
	}

	return lipgloss.NewStyle().Height(m.bodyHeight).Render(
		lipgloss.JoinVertical(lipgloss.Position(0), rows...),
	)
}

// RenderStatusBar renders the bottom bar showing diff stats and scroll position.
func RenderStatusBar(m *Model) string {
	s := m.styles
	withWidth := s.StatusBar.Width(m.terminalWidth)

	if m.state == LoadStateLoading {
		return withWidth.Render(m.spinner.View() + " Loading diff")
	}
	if m.state == LoadStateError {
		return withWidth.Render("error")
	}
	if m.state != LoadStateLoaded {
		return withWidth.Render(" ")
	}

	total := len(m.allDiffLines)
	if total == 0 {
		return withWidth.Render(fmt.Sprintf("+%d -%d", m.diff.LinesAdded, m.diff.LinesDeleted))
	}

	pos := m.vScrollOffset + 1
	pct := pos * 100 / total
	status := fmt.Sprintf("+%d -%d  %d/%d %d%%",
		m.diff.LinesAdded, m.diff.LinesDeleted,
		pos, total, pct)

	return withWidth.AlignHorizontal(lipgloss.Position(1)).Render(status)
}

// renderDiffLine renders a single diff line with gutter (line numbers + prefix)
// and content. Line number colours match their respective prefix sign so the
// gutter reads as a cohesive indicator column.
func renderDiffLine(dl git.DiffLine, highlighted string, lineNumWidth, contentWidth int, s styles.DiffStyles) string {
	if dl.Type == git.DiffHunkHeader {
		return s.HunkHeader.Width(lineNumWidth*2 + 4 + contentWidth).Render(
			ansi.Truncate(dl.Content, lineNumWidth*2+4+contentWidth, styles.Ellipsis),
		)
	}

	if dl.Type == git.DiffNoNewline {
		return s.DiffContext.Render(
			ansi.Truncate(dl.Content, lineNumWidth*2+4+contentWidth, styles.Ellipsis),
		)
	}

	var oldNumStyle, newNumStyle lipgloss.Style
	var prefix string
	var renderedContent string

	switch dl.Type {
	case git.DiffAdded:
		oldNumStyle = s.OldLineNum
		newNumStyle = s.DiffAddedPrefix
		prefix = s.DiffAddedPrefix.Render("+")
		renderedContent = s.DiffContext.Width(contentWidth).MaxWidth(contentWidth).
			Render(ansi.Truncate(highlighted, contentWidth, styles.Ellipsis))

	case git.DiffRemoved:
		oldNumStyle = s.DiffRemovedPrefix
		newNumStyle = s.OldLineNum
		prefix = s.DiffRemovedPrefix.Render("-")
		renderedContent = s.DiffContext.Width(contentWidth).MaxWidth(contentWidth).
			Render(ansi.Truncate(highlighted, contentWidth, styles.Ellipsis))

	default: // DiffContext
		oldNumStyle = s.OldLineNum
		newNumStyle = s.NewLineNum
		prefix = " "
		renderedContent = s.DiffContext.Width(contentWidth).MaxWidth(contentWidth).
			Render(ansi.Truncate(highlighted, contentWidth, styles.Ellipsis))
	}

	oldNum := formatLineNum(dl.OldLine, lineNumWidth, oldNumStyle)
	newNum := formatLineNum(dl.NewLine, lineNumWidth, newNumStyle)

	return strings.Join([]string{oldNum, " ", newNum, " ", prefix, " ", renderedContent}, "")
}

// formatLineNum returns a right-justified line number string or spaces when n == 0.
func formatLineNum(n, width int, s lipgloss.Style) string {
	if n == 0 {
		return s.Render(fmt.Sprintf("%*s", width, ""))
	}
	return s.Render(fmt.Sprintf("%*d", width, n))
}

// calcLineNumWidth returns the number of digits needed for the largest line
// number in the diff (minimum 1).
func calcLineNumWidth(lines []git.DiffLine) int {
	maxN := 1
	for _, dl := range lines {
		if dl.OldLine > maxN {
			maxN = dl.OldLine
		}
		if dl.NewLine > maxN {
			maxN = dl.NewLine
		}
	}
	w := 0
	for n := maxN; n > 0; n /= 10 {
		w++
	}
	if w < 1 {
		return 1
	}
	return w
}
