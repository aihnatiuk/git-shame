# Plan: Git Diff View + git show Parser (Phase 2)

## Context
Implementing the full-screen file diff view (`d` key in blame) and the `git show` parser. The view shows a commit info header (reusable component) above a unified diff for the blamed file only, with Chroma syntax highlighting and old/new line numbers. `Enter` key is reserved for a future "detailed commit info" view (file list + diff preview split).

---

## Implementation Order

### 1. Export `PaintBackground` from highlight package
**File:** `internal/highlight/highlight.go`

Add exported function (move logic from `blame/render.go`'s unexported `paintBackground`):
```go
// PaintBackground re-injects bg SGR after every \x1b[m / \x1b[0m in s.
func PaintBackground(s string, bg color.Color) string {
    bgSGR := ansi.Style{}.BackgroundColor(bg).String()
    s = strings.ReplaceAll(s, "\x1b[m", "\x1b[m"+bgSGR)
    s = strings.ReplaceAll(s, "\x1b[0m", "\x1b[0m"+bgSGR)
    return s
}
```
New imports: `"image/color"`, `"strings"`, `"github.com/charmbracelet/x/ansi"`.

**File:** `internal/ui/blame/render.go`
Remove unexported `paintBackground`. Replace its two call sites with `highlight.PaintBackground(...)`. Add import for `highlight` package.

---

### 2. Create `internal/git/show.go`
New types:

```go
type DiffLineType int
const (
    DiffContext DiffLineType = iota
    DiffAdded
    DiffRemoved
    DiffHunkHeader
    DiffNoNewline
)

type DiffLine struct {
    Type    DiffLineType
    Content string // prefix char stripped
    OldLine int    // 0 = N/A
    NewLine int    // 0 = N/A
}

type Hunk struct {
    Header   string // raw "@@ -a,b +c,d @@ ..." text
    OldStart, OldCount int
    NewStart, NewCount int
    Lines    []DiffLine
}

type FileDiff struct {
    OldFile, NewFile         string
    Hunks                    []Hunk
    LinesAdded, LinesDeleted int
}

type CommitInfo struct {
    Hash, Author, AuthorEmail   string
    Committer, CommitterEmail   string
    AuthorTime, CommitTime      time.Time
    Subject, Body               string
}

// ShowResult is the tea.Msg returned by RunShowCmd.
type ShowResult struct {
    Commit CommitInfo
    Diff   FileDiff
    Err    error
}
```

Functions:
```go
func RunShowCmd(repoRoot, relFile, hash string) tea.Cmd
func runShow(repoRoot, relFile, hash string) (CommitInfo, FileDiff, error)
func parseShow(data []byte) (CommitInfo, FileDiff, error)
func parseDiff(lines []string) FileDiff
func parseHunkHeader(line string) Hunk
func parseHunkRange(s string) [2]int
```

**git command:**
```
git show --format=tformat:"%H%n%aN%n%aE%n%at%n%cN%n%cE%n%ct%n%s%n%b" --patch <hash> -- <relFile>
```

**Parsing strategy:**
- Lines 0–7: fixed fields (hash, authorName, authorEmail, authorUnixTime, committerName, committerEmail, committerUnixTime, subject)
- Lines 8+: body — collect until first line starting with `diff --git `, trim trailing blank lines, join with `\n`
- Diff: everything from the `diff --git ` boundary onward → `parseDiff`

**`parseDiff` line classification:**
- `diff --git ` → skip (OldFile/NewFile set by `---`/`+++`)
- `--- ` → `fd.OldFile = strings.TrimPrefix(raw[4:], "a/")`
- `+++ ` → `fd.NewFile = strings.TrimPrefix(raw[4:], "b/")`
- `@@ ` → parse hunk header, store raw as `h.Header`, reset `oldLine`/`newLine` counters
- `+` prefix → `DiffAdded`, increment `newLine` and `LinesAdded`
- `-` prefix → `DiffRemoved`, increment `oldLine` and `LinesDeleted`
- ` ` prefix → `DiffContext`, increment both
- `\` prefix → `DiffNoNewline`

**Edge cases:**
- No `diff --git` line (commit didn't change this file): `FileDiff` stays zero-valued; view shows "No changes to this file"
- Binary files: no hunks parsed; same handling
- Root commits: `git show` handles them natively (shows all content as added)

---

### 3. Add `DiffStyles` to `internal/ui/styles/styles.go`

New struct alongside `BlameStyles`:
```go
type DiffStyles struct {
    TitleBar, StatusBar, Error, Loading, Row lipgloss.Style
    Hash, Author, Date                       lipgloss.Style
    DiffAdded, DiffRemoved, DiffContext      lipgloss.Style // bg colors
    HunkHeader                               lipgloss.Style // cyan
    OldLineNum, NewLineNum                   lipgloss.Style
    DiffAddedPrefix, DiffRemovedPrefix       lipgloss.Style
}

func DefaultDiff() DiffStyles
```

Color palette (256-color): TitleBar/StatusBar same as BlameStyles; `DiffAdded` bg=22 (dark green); `DiffRemoved` bg=52 (dark red); `HunkHeader` fg=37 (cyan) bold; `DiffAddedPrefix` fg=76; `DiffRemovedPrefix` fg=196; `OldLineNum`/`NewLineNum` fg=240; `Hash` fg=214; `Author` fg=183; `Date` fg=108.

---

### 4. Create `internal/ui/commitinfo/commitinfo.go`

Pure rendering, no Bubble Tea model. Single exported function:
```go
func Render(info git.CommitInfo, maxBodyLines int, width int, s styles.DiffStyles) string
```

Layout (each line of returned string):
1. `"commit " + info.Hash` styled with `s.Hash`
2. `"Author: "` + author+email styled with `s.Author`, right-aligned date styled with `s.Date` (omit date if terminal too narrow; use `runewidth` for padding calculation)
3. `"Commit: "` + committer+email + date — only when `info.Committer != info.Author || info.CommitterEmail != info.AuthorEmail`
4. Blank line
5. `"    " + subject` (4-space indent), truncated with `ansi.Truncate`
6. Up to `maxBodyLines` body lines, each `"    " + line`; append `"    ..."` if truncated
7. Trailing blank line (separator before diff body)

Date format: `"2006-01-02 15:04"`

---

### 5. Create `internal/ui/diff/keys.go`

```go
type KeyMap struct {
    Up, Down, HalfPageUp, HalfPageDown key.Binding
    GoToTop, GoToBottom, Quit          key.Binding
}
func DefaultKeyMap() KeyMap // same bindings as blame (j/k/ctrl-d/ctrl-u/g/G/q)
```
`ctrl+c` handled at App level; not listed here.

---

### 6. Create `internal/ui/diff/model.go`

```go
type CloseDiffMsg struct{} // sentinel → App to switch back to blame

type LoadState int // same iota as blame: Idle/Loading/Loaded/Error

type Model struct {
    commit           git.CommitInfo
    diff             git.FileDiff
    allDiffLines     []git.DiffLine  // flattened: hunk headers + content lines
    highlightedLines []string        // parallel to allDiffLines, Chroma ANSI
    loadErr          error
    state            LoadState
    repoRoot, relFile, hash string
    terminalWidth, terminalHeight int
    headerHeight      int  // computed after ShowResult; defaults to 1 (spinner line)
    bodyHeight        int
    vScrollOffset     int
    spinner           spinner.Model
    keys              KeyMap
    styles            styles.DiffStyles
}

func New(repoRoot, relFile, hash string) Model
func (m Model) Init() tea.Cmd           // RunShowCmd + spinner.Tick
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd)
func (m Model) View() tea.View          // AltScreen = true
func (m Model) WithSize(w, h int) Model
```

**Update message handling:**
- `git.ShowResult`: set state, store commit+diff, call `flattenDiffLines`, `buildHighlightedLines`, `computeHeaderHeight`, `computeBodyHeight`, `adjustScrollOffset`
- `spinner.TickMsg`: forward only when `LoadStateLoading`
- `tea.KeyMsg`: gate on `LoadStateLoaded`; on `Quit` return `CloseDiffMsg` sentinel

**Helper functions (unexported, in model.go):**
```go
func flattenDiffLines(fd git.FileDiff) []git.DiffLine
// Injects synthetic DiffHunkHeader entries (Content = h.Header) before each hunk's lines.

func buildHighlightedLines(fd git.FileDiff, relFile string) []string
// Calls highlight.HighlightLines on all content lines in flattened order.

func computeHeaderHeight(info git.CommitInfo, width int, s styles.DiffStyles) int
// Returns lipgloss.Height(commitinfo.Render(info, 5, width, s))

func computeBodyHeight(termHeight, headerHeight int) int
// max(termHeight - 2 - headerHeight, 1)
```

**`WithSize`:** Recomputes `headerHeight` (if loaded) and `bodyHeight`, then calls `adjustScrollOffset`.

**`adjustScrollOffset`:** Clamps `vScrollOffset` to `[0, max(0, len(allDiffLines)-bodyHeight)]`.

**View layout:** `lipgloss.JoinVertical(title, header, body, status)` wrapped in full-terminal size style.

---

### 7. Create `internal/ui/diff/render.go`

```go
func RenderTitleBar(relFile, hash string, width int, s styles.DiffStyles) string
func RenderHeader(m *Model) string   // spinner during loading, commitinfo.Render when loaded
func RenderBody(m *Model) string
func RenderStatusBar(m *Model) string // shows "+N -N  pos/total pct%"
```

**`RenderBody` line rendering:**
- `lineNumWidth = calcLineNumWidth(allDiffLines)` (digit count of max line number, min 1)
- `gutterWidth = lineNumWidth*2 + 3` (oldNum + space + newNum + space + prefix + space)
- For `DiffHunkHeader`: render full width with `s.HunkHeader`, skip gutter
- For `DiffNoNewline`: render with `s.DiffContext`
- For others: `oldNum + " " + newNum + " " + prefix + " " + content`
  - `content = ansi.Truncate(highlighted, contentWidth, "…")`
  - If `bg != nil`: `content = highlight.PaintBackground(content, bg)`
  - Apply `contentStyle.Width(contentWidth).MaxWidth(contentWidth).Render(content)`
- Guard: `contentWidth = max(terminalWidth - gutterWidth, 0)`

---

### 8. Modify `internal/ui/blame/model.go`

Extend `OpenDiffMsg`:
```go
type OpenDiffMsg struct {
    CommitHash string
    RepoRoot   string
    RelFile    string
}
```

Update `handleKey` OpenDiff case to populate `RepoRoot: m.repoRoot, RelFile: m.relFile`.

Add accessors:
```go
func (m Model) TerminalWidth() int  { return m.terminalWidth }
func (m Model) TerminalHeight() int { return m.terminalHeight }
```

---

### 9. Modify `internal/ui/blame/keys.go`

- Change `OpenDiff` key from `"enter"` to `"d"`, help: `"d", "open diff"`
- Add `OpenCommitInfo key.Binding` field to `KeyMap`
- In `DefaultKeyMap()`: `key.WithKeys("enter")`, help: `"enter", "open commit info"`
- No handler in `handleKey` yet

---

### 10. Modify `internal/ui/app.go`

```go
import "github.com/aihnatiuk/git-shame/internal/ui/diff"

const (
    ViewBlame ViewID = iota
    ViewDiff
)

type App struct {
    activeView ViewID
    blameModel blame.Model
    diffModel  diff.Model
}
```

Update `Update`:
- `tea.WindowSizeMsg`: call `WithSize` on both models
- `blame.OpenDiffMsg`: create `diff.New(msg.RepoRoot, msg.RelFile, msg.CommitHash)`, call `WithSize` using `a.blameModel.TerminalWidth()/TerminalHeight()`, set `activeView = ViewDiff`, return `diffModel.Init()`
- `diff.CloseDiffMsg`: set `activeView = ViewBlame`
- Delegation switch: add `case ViewDiff` that delegates to `diffModel.Update`

Update `View`: add `case ViewDiff` returning `diffModel.View()`.

---

### 11. Update `.codemie/guides/keybindings.md`

Replace `Enter | Open diff view for current line's commit` with:
- `d` → Open full-screen diff for current line's commit
- `Enter` → Open commit info (changed files + diff preview)

---

### 12. Update `CLAUDE.md`

- **Project structure**: add `commitinfo/` and update `diff/` entry as implemented
- **Phase 2**: mark diff view, git show parser as done (~~strikethrough~~)
- **New phase entry**: "Detailed commit info view (file list + diff preview split, `Enter` key)" — add as a future phase item

---

## Verification

1. `go build -o shame.exe .` — must compile with no errors
2. Run `DEBUG=1 ./shame <any-file>` — `d` opens diff, `q` returns to blame
3. Commit info header shows hash/author/date/message; added lines green, removed lines red
4. Syntax highlighting visible through diff backgrounds
5. Line numbers: added lines have blank old column, removed lines have blank new column
6. Resize: layout reflows without crash
7. Root commit: all lines shown as added
8. File with no diff in commit: shows "No changes to this file"
9. Long commit message: body truncated at 5 lines with `...`
10. `ctrl+c` quits from both views
