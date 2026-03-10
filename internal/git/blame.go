package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

// RepoRoot returns the root directory of the git repository containing path.
func RepoRoot(path string) (string, error) {
	dir := path
	if !isDir(path) {
		dir = filepath.Dir(path)
	}
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// RelPath returns the file path relative to the repo root.
// git blame requires a path relative to the repo root when using a revision.
func RelPath(repoRoot, absFile string) (string, error) {
	rel, err := filepath.Rel(repoRoot, absFile)
	if err != nil {
		return "", err
	}
	// Git expects forward slashes even on Windows.
	return filepath.ToSlash(rel), nil
}

// RunBlameCmd returns a tea.Cmd that runs git blame asynchronously.
// When done it sends a BlameResult message to the Bubble Tea runtime.
func RunBlameCmd(repoRoot, relFile, revision string) tea.Cmd {
	return func() tea.Msg {
		lines, err := runBlame(repoRoot, relFile, revision)
		return BlameResult{Lines: lines, Err: err}
	}
}

// runBlame shells out to `git blame --porcelain` and parses the output.
func runBlame(repoRoot, relFile, revision string) ([]BlameLine, error) {
	args := []string{"blame", "--porcelain"}
	if revision != "" {
		args = append(args, revision)
	}
	args = append(args, "--", relFile)

	cmd := exec.Command("git", args...)
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if ok := isExitError(err, &exitErr); ok {
			return nil, fmt.Errorf("git blame: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("git blame: %w", err)
	}
	return parsePorcelain(out)
}

// parsePorcelain parses git blame --porcelain output into a slice of BlameLine.
// The format is described in git-blame(1). Key invariants:
//   - A header line starts with a 40-hex SHA followed by space-separated integers.
//   - Metadata key-value lines follow the header (one field per line).
//   - The actual line content is always prefixed by a literal TAB character.
//   - Metadata for a commit is only emitted once (on its first appearance).
func parsePorcelain(data []byte) ([]BlameLine, error) {
	data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	lines := bytes.Split(data, []byte("\n"))
	metaCache := make(map[string]*CommitMeta)
	var result []BlameLine
	var current *CommitMeta

	i := 0
	for i < len(lines) {
		line := lines[i]

		// Skip empty lines.
		if len(line) == 0 {
			i++
			continue
		}

		// Content line: always starts with a literal TAB.
		if line[0] == '\t' {
			content := string(line[1:])
			if current != nil {
				result = append(result, BlameLine{
					CommitHash:  current.Hash,
					Author:      current.Author,
					AuthorEmail: current.AuthorEmail,
					AuthorTime:  current.AuthorTime,
					Summary:     current.Summary,
					LineNum:     len(result) + 1,
					Content:     content,
					Filename:    current.Filename,
					Previous:    current.Previous,
				})
			}
			i++
			continue
		}

		// Header line: "<40-char-sha> <orig-line> <final-line> [<num-lines>]"
		// Detect by checking if the line starts with a 40-char hex string.
		if isHexLine(line) {
			parts := bytes.Fields(line)
			if len(parts) >= 3 {
				sha := string(parts[0])
				finalLine, _ := strconv.Atoi(string(parts[2]))

				if meta, ok := metaCache[sha]; ok {
					current = meta
					// Update lineNum for this entry (will be set correctly below).
					_ = finalLine
				} else {
					meta = &CommitMeta{Hash: sha}
					metaCache[sha] = meta
					current = meta
				}
				_ = finalLine
			}
			i++
			continue
		}

		// Metadata key-value line.
		if current != nil {
			kv := line
			switch {
			case bytes.HasPrefix(kv, []byte("author ")):
				current.Author = string(kv[7:])
			case bytes.HasPrefix(kv, []byte("author-mail ")):
				current.AuthorEmail = strings.Trim(string(kv[12:]), "<>")
			case bytes.HasPrefix(kv, []byte("author-time ")):
				ts, _ := strconv.ParseInt(string(kv[12:]), 10, 64)
				current.AuthorTime = time.Unix(ts, 0)
			case bytes.HasPrefix(kv, []byte("summary ")):
				current.Summary = string(kv[8:])
			case bytes.HasPrefix(kv, []byte("filename ")):
				current.Filename = string(kv[9:])
			case bytes.HasPrefix(kv, []byte("previous ")):
				parts := bytes.Fields(kv[9:])
				if len(parts) >= 2 {
					current.Previous = PreviousCommit{
						Hash:     string(parts[0]),
						Filename: string(parts[1]),
					}
				}
			}
		}
		i++
	}

	// Fix up LineNum: use the actual 1-based index in the result slice.
	for idx := range result {
		result[idx].LineNum = idx + 1
	}

	return result, nil
}

// isHexLine returns true if line starts with exactly 40 hex characters followed
// by a space. This identifies header lines in the porcelain format.
func isHexLine(line []byte) bool {
	if len(line) < 41 {
		return false
	}
	for _, b := range line[:40] {
		if !((b >= '0' && b <= '9') || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')) {
			return false
		}
	}
	return line[40] == ' '
}

func isDir(path string) bool {
	// Cheap check — treat as file if it has an extension, dir otherwise.
	return filepath.Ext(path) == ""
}

func isExitError(err error, target **exec.ExitError) bool {
	var ee *exec.ExitError
	if e, ok := err.(*exec.ExitError); ok {
		ee = e
		*target = ee
		return true
	}
	return false
}
