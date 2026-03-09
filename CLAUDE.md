# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build
go build -o shame .

# Run
./shame <file> [revision]

# Run tests
go test ./...

# Run a single test package
go test ./internal/git/...

# Run with race detector
go test -race ./...
```

## Architecture

`shame` is a Bubble Tea TUI. The model hierarchy is:

```
tea.Program
  └── App (internal/ui/app.go)         — view switching, WindowSizeMsg dispatch
        └── blame.Model (internal/ui/blame/model.go)  — blame view (Phase 1, current)
```

### Key data flow

1. `main.go` resolves the file to an absolute path, finds the repo root via `git rev-parse --show-toplevel`, computes the repo-relative path (required by `git blame`), and passes all four strings into `ui.NewApp`.
2. `blame.Model.Init()` dispatches `git.RunBlameCmd` (async goroutine) + `spinner.Tick`. The result arrives as `git.BlameResult`.
3. `App.Update` handles `tea.WindowSizeMsg` directly (calls `blameModel.WithSize`), then delegates all other messages to the active child model.
4. Child→parent communication uses sentinel messages (`blame.OpenDiffMsg`), not callbacks.

### Bubble Tea contract

All models use **value receivers**. Never mutate slices in-place — always copy before appending (see `navigateToParent` in model.go for the canonical pattern). `RenderBody` in `render.go` takes a pointer only because it reads multiple fields; no mutation happens there.

### Column system (`internal/ui/blame/columns.go`)

`defaultColumns()` defines the ordered column slice. `RecalcWidths(cols, lines, termWidth)` is called on every `BlameResult` and `WindowSizeMsg`. `ColCode` is the flex column — it gets all remaining width after fixed columns are accounted for. Guard: if `len(lines) == 0`, `RecalcWidths` returns `cols` unchanged.

Width measurement **must** use `runewidth.StringWidth()` or `lipgloss.Width()`, never `len()`.

### Rendering (`internal/ui/blame/render.go`)

`RenderRow` applies cursor highlight per-cell (not as an outer wrapper) so that individual cell ANSI resets don't clear the highlight between columns. The cursor background color is extracted from `s.Cursor.GetBackground()` and passed into each `renderCell` call via `withBg`.

Horizontal scroll (`hScroll`) is applied **only to the Code column** inside `renderCell` — metadata columns (hash, date, author, line number) remain fixed. The left-truncation uses `ansi.TruncateLeft(content, hScroll, "")` from `github.com/charmbracelet/x/ansi`, which is ANSI-escape-code-aware and preserves embedded escape sequences. This makes it safe for Phase 2 syntax-highlighted content (pre-computed Chroma ANSI output stored on the line).

**Never call Chroma (syntax highlighting) inside `View()`** — it must be pre-computed in `Update()` once data loads.

### History stack

`,` (Parent): pushes `HistoryEntry{File, Revision, CursorLine}` onto `m.history`, then reloads at `commitHash + "^"`. Cap: 50 entries.

`<` (Back): pops the stack, sets `m.pendingCursor`, reloads. `BlameResult` handler applies `pendingCursor` after lines load.

### Styles (`internal/ui/styles/styles.go`)

All lipgloss styles live in `BlameStyles` (returned by `styles.Default()`). `BlameModel` holds a `styles.BlameStyles` value. Add new style fields here; never construct ad-hoc lipgloss styles in render functions except for cursor-bg overrides.

## Implementation phases

- **Phase 1** (done): blame view, navigation, history stack
- **Phase 2**: diff view (`internal/ui/diff/`), `git show` parser, Chroma highlighting in blame
- **Phase 3**: column toggle popup (`internal/ui/colmenu/`)
- **Phase 4**: YAML config (`internal/config/`), XDG path `~/.config/shame/config.yaml`
- **Phase 5**: `/` search, filename/summary columns, committer columns

## Module path

`github.com/aihnatiuk/git-shame` — use this in all imports, not the plan's `shame` shorthand.
