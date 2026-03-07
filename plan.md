# Plan: shame — Git Blame TUI

## Context

Building `shame` from scratch: a modern, interactive terminal UI for `git blame` exploration, written in Go. The tool aims to replace `tig blame` with a more user-friendly experience. This plan covers the full intended feature set, with Phase 1 as the initial implementation target.

---

## Confirmed Decisions

| Topic | Decision |
|---|---|
| Language | Go 1.26.1 |
| Module name | `shame` |
| TUI framework | Bubble Tea + Bubbles + lipgloss |
| Syntax highlighting | Chroma v2 (both blame and diff views) |
| Git integration | Git CLI subprocess (`git blame --porcelain`, `git show`) |
| Long lines | Truncate at terminal width + horizontal scroll |
| Config file | XDG only — `~/.config/shame/config.yaml` |
| Search scope | Code content only |
| Enter key | Opens diff view (full-screen replacement) |
| Parent commit nav | `,` (comma) re-blames at parent; `<` goes back |
| History breadcrumb | Title bar shows `file @ revision` |
| Column toggles | Dedicated popup menu (`c` key) + optional keybindings in config + default visibility in config |

---

## Keybindings (defaults)

| Key | Action |
|---|---|
| `j` / `k` | Move cursor down / up |
| `ctrl-d` / `ctrl-u` | Half page down / up |
| `g` / `G` | First / last line |
| `,` | Navigate to parent commit (re-blame) |
| `<` | Go back in blame history |
| `Enter` | Open diff view for current line's commit |
| `q` | Quit (from blame) / return to blame (from diff) |
| `/` | Start code search |
| `n` / `N` | Next / previous search match |
| `c` | Open column toggle menu |
| `left`/`h`, `right`/`l` | Horizontal scroll |

---

## Project Structure

```
C:\Projects\git-shame\
├── main.go                          # CLI entry point, arg parsing
├── go.mod
├── go.sum
└── internal/
    ├── git/
    │   ├── types.go                 # BlameLine, CommitMeta, DiffResult
    │   ├── blame.go                 # RunBlameCmd, git blame --porcelain parser
    │   ├── blame_test.go            # Parser unit tests with fixture
    │   └── diff.go                  # RunDiffCmd, git show parser (Phase 2)
    ├── ui/
    │   ├── app.go                   # Root tea.Model; manages active view, WindowSizeMsg
    │   ├── blame/
    │   │   ├── model.go             # BlameModel: full tea.Model with history stack
    │   │   ├── columns.go           # Column type, defaultColumns(), RecalcWidths()
    │   │   ├── keys.go              # KeyMap using bubbles/key
    │   │   └── render.go            # RenderRow, RenderBody, RenderTitleBar, RenderStatusBar
    │   ├── diff/
    │   │   ├── model.go             # DiffModel: split pane (files left, diff right) (Phase 2)
    │   │   └── keys.go              # KeyMap for diff view (Phase 2)
    │   ├── colmenu/
    │   │   └── model.go             # Column toggle popup with checkboxes (Phase 3)
    │   └── styles/
    │       └── styles.go            # All lipgloss styles, theme application
    ├── highlight/
    │   └── highlight.go             # Chroma wrapper, pre-highlight lines after load (Phase 2)
    └── config/
        ├── config.go                # YAML config struct, Load(), XDG path resolution (Phase 4)
        └── defaults.go              # Default Config values (Phase 4)
```

---

## Key Types

### `internal/git/types.go`

```go
type BlameLine struct {
    CommitHash  string    // full 40-char SHA
    Author      string
    AuthorEmail string
    AuthorTime  time.Time
    Summary     string    // first line of commit message
    LineNum     int       // final line number in the file (1-indexed)
    Content     string    // raw line content (no newline)
    Filename    string    // "filename" from porcelain (handles renames)
    Previous    string    // parent SHA (from "previous" field), may be empty
}
```

### `internal/ui/blame/model.go`

```go
type HistoryEntry struct {
    File       string
    Revision   string
    CursorLine int
}

type Model struct {
    lines         []git.BlameLine
    loadErr       error
    state         LoadState
    cursor        int
    offset        int
    hScroll       int
    history       []HistoryEntry
    pendingCursor int
    file          string
    revision      string   // empty = HEAD
    width, height int
    bodyH         int      // height - 2
    columns       []Column
    spinner       spinner.Model
    keys          KeyMap
}
```

---

## Architecture

### Bubble Tea Model Hierarchy

```
tea.Program
  └── App (ui/app.go)
        ├── activeView  ViewID  (ViewBlame | ViewDiff)
        ├── blameModel  blame.Model
        └── diffModel   diff.Model
```

- Child → App communication via sentinel `tea.Msg`: `OpenDiffMsg`, `CloseDiffMsg`
- `WindowSizeMsg` handled at App level, forwarded to child models
- All models use **value receivers** (Bubble Tea immutability contract)

### Async Loading

- `git.RunBlameCmd(file, revision) tea.Cmd` shells out in a goroutine
- Returns `git.BlameResult{Lines, Err}` message when done
- `BlameModel.Init()` returns `tea.Batch(RunBlameCmd(...), spinner.Tick)`

### History Stack

- `,` key: push `HistoryEntry`, reload at `commitHash + "^"`
- `<` key: pop stack, set `pendingCursor`, reload at previous state
- `BlameResult` handler applies `pendingCursor` after lines load
- Cap at 50 entries

### Column Width Calculation (`RecalcWidths`)

1. Measure max author width from data
2. Fixed: Hash=8, Date=10, LineNum=digits-of-line-count
3. Dynamic: Author = measured, capped at 20
4. Flex: Code = `termWidth - sum(others) - separators`
5. Guard: skip if `lines == nil`

---

## Dependencies

```
module shame

go 1.26

require (
    github.com/charmbracelet/bubbletea  latest
    github.com/charmbracelet/bubbles    latest
    github.com/charmbracelet/lipgloss   latest
    github.com/alecthomas/chroma/v2     latest
    github.com/mattn/go-runewidth       latest
    gopkg.in/yaml.v3                    latest
)
```

---

## Implementation Phases

### Phase 1 (Initial)
1. `go.mod` init + install dependencies
2. `internal/git/types.go` — data types
3. `internal/git/blame.go` — `--porcelain` parser + `RunBlameCmd`
4. `internal/ui/styles/styles.go` — lipgloss style constants
5. `internal/ui/blame/columns.go` — Column type, `defaultColumns()`, `RecalcWidths()`
6. `internal/ui/blame/keys.go` — `KeyMap` with `bubbles/key`
7. `internal/ui/blame/render.go` — `RenderRow`, `RenderBody`, `RenderTitleBar`, `RenderStatusBar`
8. `internal/ui/blame/model.go` — full `BlameModel` (Init/Update/View, history stack, spinner)
9. `internal/ui/app.go` — root `App` model
10. `main.go` — CLI arg parsing, program setup

### Phase 2
- `internal/git/diff.go` — `git show` parser
- `internal/highlight/highlight.go` — Chroma pre-highlighting
- `internal/ui/diff/model.go` — split-pane diff view
- `Enter` → diff, `q` → blame
- Chroma highlighting in blame code column

### Phase 3
- `internal/ui/colmenu/model.go` — column toggle popup (`c` key)
- Horizontal scroll wired (`h`/`l`/arrows)
- `g`/`G` first/last line

### Phase 4
- `internal/config/` — YAML config (XDG), keybindings, theme, column defaults

### Phase 5
- `/` search, `n`/`N` cycle matches
- Filename + Summary columns
- Committer columns
- `?` reverse search

---

## Architectural Pitfalls

1. **git working directory**: detect repo root with `git rev-parse --show-toplevel`; set `cmd.Dir = repoRoot`; pass file as path relative to repo root
2. **Immutability**: value receivers only; never mutate slices in-place
3. **Rendering perf**: Chroma pre-computed in `Update()`, never in `View()`
4. **Width measurement**: `runewidth.StringWidth()` / `lipgloss.Width()` — never `len()`
5. **WindowSizeMsg race**: `RecalcWidths` guards on `lines == nil`
6. **History cap**: max 50 entries
7. **Porcelain parsing**: detect content lines by `\t` prefix only

---

## Verification (Phase 1)

1. `go build -o shame .` compiles without errors
2. `./shame <file>` in a git repo renders blame view with columns
3. `j`/`k` moves cursor; viewport scrolls at edges
4. `ctrl-d`/`ctrl-u` half-page scroll
5. `,` re-blames at parent; title bar updates
6. `<` restores previous blame state with cursor position
7. `./shame <file> <revision>` loads blame at specified revision
8. Terminal resize reflows columns correctly
