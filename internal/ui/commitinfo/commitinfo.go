// Package commitinfo provides a reusable commit metadata header renderer.
package commitinfo

import (
	"fmt"
	"strings"

	"github.com/aihnatiuk/git-shame/internal/git"
	"github.com/aihnatiuk/git-shame/internal/ui/styles"
	"github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"

	"charm.land/lipgloss/v2"
)

const dateFormat = "2006-01-02 15:04"

// Render returns a multi-line commit info header string suitable for display
// above a diff body. maxBodyLines limits how many body lines are shown before
// a "..." truncation indicator. width is the terminal width used for layout.
func Render(info git.CommitInfo, maxBodyLines int, width int, s styles.DiffStyles) string {
	var sb strings.Builder

	// Line 1: commit hash
	sb.WriteString(s.Hash.Render("commit " + info.Hash))
	sb.WriteByte('\n')

	// Line 2: Author + right-aligned date
	authorDate := info.AuthorTime.Format(dateFormat)
	authorPart := "Author: " + info.Author + " <" + info.AuthorEmail + ">"
	sb.WriteString(renderWithRightAligned(authorPart, authorDate, width, s.Author, s.Date))
	sb.WriteByte('\n')

	// Line 3 (conditional): Committer line when different from author
	if info.Committer != info.Author || info.CommitterEmail != info.AuthorEmail {
		commitDate := info.CommitTime.Format(dateFormat)
		committerPart := "Commit: " + info.Committer + " <" + info.CommitterEmail + ">"
		sb.WriteString(renderWithRightAligned(committerPart, commitDate, width, s.Author, s.Date))
		sb.WriteByte('\n')
	}

	// Blank separator
	sb.WriteByte('\n')

	// Subject line (4-space indent)
	subject := "    " + ansi.Truncate(info.Subject, max(0, width-4), styles.Ellipsis)
	sb.WriteString(subject)
	sb.WriteByte('\n')

	// Body lines
	if info.Body != "" {
		bodyLines := strings.Split(info.Body, "\n")
		truncated := false
		if len(bodyLines) > maxBodyLines {
			bodyLines = bodyLines[:maxBodyLines]
			truncated = true
		}
		for _, bl := range bodyLines {
			sb.WriteString("    ")
			sb.WriteString(ansi.Truncate(bl, max(0, width-4), styles.Ellipsis))
			sb.WriteByte('\n')
		}
		if truncated {
			sb.WriteString("    ...")
			sb.WriteByte('\n')
		}
	}

	// Trailing blank line as separator before diff body
	sb.WriteByte('\n')

	return sb.String()
}

// renderWithRightAligned renders left text with right text aligned to the far
// right of width. Falls back to just the left text if terminal is too narrow.
func renderWithRightAligned(left, right string, width int, leftStyle, rightStyle lipgloss.Style) string {
	leftRendered := leftStyle.Render(left)
	rightRendered := rightStyle.Render(right)

	leftW := runewidth.StringWidth(ansi.Strip(leftRendered))
	rightW := runewidth.StringWidth(ansi.Strip(rightRendered))
	gap := width - leftW - rightW

	if gap < 1 {
		// Not enough space; omit date.
		return leftRendered
	}

	return leftRendered + fmt.Sprintf("%*s", gap, "") + rightRendered
}
