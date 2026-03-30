package git

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

// DiffLineType classifies a line within a unified diff hunk.
type DiffLineType int

const (
	DiffContext   DiffLineType = iota
	DiffAdded                  // line exists only in new version
	DiffRemoved                // line exists only in old version
	DiffHunkHeader             // @@ ... @@ synthetic separator
	DiffNoNewline              // "\ No newline at end of file"
)

// DiffLine is a single line in a parsed unified diff.
type DiffLine struct {
	Type    DiffLineType
	Content string // prefix char stripped
	OldLine int    // 1-based line number in old file; 0 = not applicable
	NewLine int    // 1-based line number in new file; 0 = not applicable
}

// Hunk is one @@ ... @@ section of a unified diff.
type Hunk struct {
	Header   string // raw "@@ -a,b +c,d @@ ..." text
	OldStart int
	OldCount int
	NewStart int
	NewCount int
	Lines    []DiffLine
}

// FileDiff holds the parsed diff for a single file.
type FileDiff struct {
	OldFile      string
	NewFile      string
	Hunks        []Hunk
	LinesAdded   int
	LinesDeleted int
}

// CommitInfo holds metadata about a single commit.
type CommitInfo struct {
	Hash           string
	Author         string
	AuthorEmail    string
	Committer      string
	CommitterEmail string
	AuthorTime     time.Time
	CommitTime     time.Time
	Subject        string
	Body           string
}

// ShowResult is the tea.Msg returned by RunShowCmd.
type ShowResult struct {
	Commit CommitInfo
	Diff   FileDiff
	Err    error
}

// RunShowCmd returns a tea.Cmd that runs git show asynchronously.
// When done it sends a ShowResult message to the Bubble Tea runtime.
func RunShowCmd(repoRoot, relFile, hash string) tea.Cmd {
	return func() tea.Msg {
		commit, diff, err := runShow(repoRoot, relFile, hash)
		return ShowResult{Commit: commit, Diff: diff, Err: err}
	}
}

func runShow(repoRoot, relFile, hash string) (CommitInfo, FileDiff, error) {
	format := "%H%n%aN%n%aE%n%at%n%cN%n%cE%n%ct%n%s%n%b"
	args := []string{
		"show",
		"--format=tformat:" + format,
		"--patch",
		hash,
		"--",
		relFile,
	}
	cmd := exec.Command("git", args...)
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if ok := isExitError(err, &exitErr); ok {
			return CommitInfo{}, FileDiff{}, fmt.Errorf("git show: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return CommitInfo{}, FileDiff{}, fmt.Errorf("git show: %w", err)
	}
	return parseShow(out)
}

// parseShow splits the tformat output into commit metadata and the diff section.
func parseShow(data []byte) (CommitInfo, FileDiff, error) {
	raw := strings.ReplaceAll(string(data), "\r\n", "\n")
	lines := strings.Split(raw, "\n")

	if len(lines) < 8 {
		return CommitInfo{}, FileDiff{}, fmt.Errorf("git show: unexpected output format")
	}

	authorUnix, _ := strconv.ParseInt(lines[3], 10, 64)
	commitUnix, _ := strconv.ParseInt(lines[6], 10, 64)

	commit := CommitInfo{
		Hash:           lines[0],
		Author:         lines[1],
		AuthorEmail:    lines[2],
		AuthorTime:     time.Unix(authorUnix, 0),
		Committer:      lines[4],
		CommitterEmail: lines[5],
		CommitTime:     time.Unix(commitUnix, 0),
		Subject:        lines[7],
	}

	// Lines[8:] start with the body, which ends at the first "diff --git " line.
	bodyLines := []string{}
	diffStart := -1
	for i := 8; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "diff --git ") {
			diffStart = i
			break
		}
		bodyLines = append(bodyLines, lines[i])
	}

	// Trim leading/trailing blank lines from body.
	for len(bodyLines) > 0 && bodyLines[0] == "" {
		bodyLines = bodyLines[1:]
	}
	for len(bodyLines) > 0 && bodyLines[len(bodyLines)-1] == "" {
		bodyLines = bodyLines[:len(bodyLines)-1]
	}
	commit.Body = strings.Join(bodyLines, "\n")

	var fd FileDiff
	if diffStart >= 0 {
		fd = parseDiff(lines[diffStart:])
	}

	return commit, fd, nil
}

var hunkHeaderRe = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

// parseDiff parses a unified diff (lines starting from the first "diff --git" line).
func parseDiff(lines []string) FileDiff {
	var fd FileDiff
	var currentHunk *Hunk
	oldLine := 0
	newLine := 0

	for _, raw := range lines {
		switch {
		case strings.HasPrefix(raw, "diff --git "):
			// Start of a new file diff block; ignore (single file context).
			currentHunk = nil

		case strings.HasPrefix(raw, "--- "):
			fd.OldFile = strings.TrimPrefix(raw[4:], "a/")

		case strings.HasPrefix(raw, "+++ "):
			fd.NewFile = strings.TrimPrefix(raw[4:], "b/")

		case strings.HasPrefix(raw, "@@ "):
			h := parseHunkHeader(raw)
			oldLine = h.OldStart
			newLine = h.NewStart
			fd.Hunks = append(fd.Hunks, h)
			currentHunk = &fd.Hunks[len(fd.Hunks)-1]

		case len(raw) > 0 && raw[0] == '+':
			if currentHunk != nil {
				currentHunk.Lines = append(currentHunk.Lines, DiffLine{
					Type:    DiffAdded,
					Content: raw[1:],
					NewLine: newLine,
				})
				newLine++
				fd.LinesAdded++
			}

		case len(raw) > 0 && raw[0] == '-':
			if currentHunk != nil {
				currentHunk.Lines = append(currentHunk.Lines, DiffLine{
					Type:    DiffRemoved,
					Content: raw[1:],
					OldLine: oldLine,
				})
				oldLine++
				fd.LinesDeleted++
			}

		case len(raw) > 0 && raw[0] == ' ':
			if currentHunk != nil {
				currentHunk.Lines = append(currentHunk.Lines, DiffLine{
					Type:    DiffContext,
					Content: raw[1:],
					OldLine: oldLine,
					NewLine: newLine,
				})
				oldLine++
				newLine++
			}

		case strings.HasPrefix(raw, `\`):
			if currentHunk != nil {
				currentHunk.Lines = append(currentHunk.Lines, DiffLine{
					Type:    DiffNoNewline,
					Content: raw,
				})
			}
		}
	}

	return fd
}

// parseHunkHeader parses a "@@ -a,b +c,d @@ ..." line into a Hunk.
func parseHunkHeader(line string) Hunk {
	h := Hunk{Header: line}
	m := hunkHeaderRe.FindStringSubmatch(line)
	if m == nil {
		return h
	}
	h.OldStart, _ = strconv.Atoi(m[1])
	if m[2] != "" {
		h.OldCount, _ = strconv.Atoi(m[2])
	} else {
		h.OldCount = 1
	}
	h.NewStart, _ = strconv.Atoi(m[3])
	if m[4] != "" {
		h.NewCount, _ = strconv.Atoi(m[4])
	} else {
		h.NewCount = 1
	}
	return h
}
