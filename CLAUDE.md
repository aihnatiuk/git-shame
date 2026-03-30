# CLAUDE.md

`shame` is a modern, interactive TUI app for navigating `git blame`, written in Go. It aims to provide a more modern, user-friendly and customizable experience than `tig blame`.

## Project Structure
```text
├── main.go               # Entry point, CLI flag/arg parsing
├── internal/
│   ├── git/              # Git CLI wrappers, blame/show parsers
│   ├── ui/               # Bubble Tea components
│   │   ├── app.go        # Root model & view switcher
│   │   ├── styles/       # Lipgloss theme definitions (BlameStyles, DiffStyles)
│   │   ├── blame/        # The Blame view
│   │   ├── diff/         # The Diff view
│   │   ├── commitinfo/   # Reusable commit metadata header renderer
│   │   └── colmenu/      # Column toggle popup (not implemented)
│   ├── highlight/        # Chroma syntax highlighting logic
│   └── config/           # YAML config & XDG path handling (not implemented)
```

## Core Tech Stack
```
module github.com/aihnatiuk/git-shame
go 1.26.1
require (
	charm.land/bubbles/v2 v2.0.0
	charm.land/bubbletea/v2 v2.0.1
	charm.land/lipgloss/v2 v2.0.0
	github.com/alecthomas/chroma/v2 v2.23.1
	github.com/charmbracelet/x/ansi v0.11.6
	github.com/mattn/go-runewidth v0.0.21
)
```

## Architecture
### Key data flow
1. `main.go` resolves the file to an absolute path, finds the repo root via `git rev-parse --show-toplevel`, computes the repo-relative path, and passes all four strings into `ui.NewApp`.
2. `blame.Model.Init()` dispatches `git.RunBlameCmd` (async goroutine) + `spinner.Tick`. The result arrives as `git.BlameResult`.
3. `App.Update` handles `tea.WindowSizeMsg` directly (calls `WithSize` on all child models), then delegates all other messages to the active child model.
4. Child→parent communication uses sentinel messages (`blame.OpenDiffMsg`), not callbacks.

### Column system (`internal/ui/blame/columns.go`)
`shame` supports the following list of columns: hash, author date, author name, commit summary, line number, source code content, filename. Each column has a `ColumnID` and a fixed width (except for the code column, which is flexible).
`defaultColumns()` defines the ordered column slice.
`RecalcWidths(cols, lines, termWidth)` is called on every `BlameResult` and `WindowSizeMsg`.

### Styles (`internal/ui/styles/styles.go`)
All lipgloss styles live in `BlameStyles` (returned by `styles.Default()`) and `DiffStyles` (returned by `styles.DefaultDiff()`). Add new style fields here; never construct ad-hoc lipgloss styles in render functions except for very specific cases.

## Development Standards
- Width measurement **must** use `runewidth.StringWidth()` or `lipgloss.Width()`, never `len()`.
- Use `log.Printf` / `log.Println` from the standard `log` package. Logs are written to `debug.log` (truncated on each run) if `DEBUG=1` env variable is set when running the app.
- Always use lipgloss v2 API for all styling, don't implement manual solutions for padding, alignment, truncation, etc., unless it's a very specific case that can't be solved with lipgloss.
- Truncation must be ansi-aware and use `ansi.Truncate`, `ansi.TruncateLeft` from `github.com/charmbracelet/x/ansi` to avoid cutting off wide characters or breaking ANSI escape codes.
- Aim for a clean separation of concerns and modular architecture, use design patterns where appropriate.
- Write clear, concise, and well-documented code. Don't overuse comments, use them to explain **non-obvious logic** and decisions. Documentation should not go into implementation details, but rather explain the purpose and behavior of functions, types, and packages. **Don't ever mention in the comments or documentation that something is not working but is going to be implemented later, don't mention implementation phases**.
- Performance is important and is a key aspect of the project.

## Commands
```bash
# Build
go build -o shame.exe .

# Run
./shame <file> [revision]
```

## Guides
Use the following guides to get additional context on specific topics of the project. Load them only when it is needed for the task at hand.
- [Agreed list of default keybindings](@.codemie/guides/keybindings.md)

## Implementation phases
**Phase 1** (done):
  - blame view, parent/child commit navigation using history stack, spinner during async blame loading, column system with dynamic width calculation, basic keybindings (`j`/`k`, `ctrl-d`/`ctrl-u`, `g`/`G`, `,`, `<`), horizontal scrolling of code column.

**Phase 2** (in progress):
  - ~~diff view (`internal/ui/diff/`)~~ (done; `d` key opens full-screen diff, `q` returns to blame)
  - ~~`git show` parser~~ (done; see `internal/git/show.go`)
  - ~~Chroma highlighting in blame~~ (done; default theme: `github-dark`, see `internal/highlight/`)
  - Whitespace indicators in code column, `·` for space, `→` for tab, `⏎` for EOL

**Phase 3**:
  - Detailed commit info view (file list + diff preview split, `Enter` key from blame)
  - Column toggle popup (`internal/ui/colmenu/`)
  
**Phase 4**:
  - YAML config (`internal/config/`)
  - XDG path `~/.config/shame/config.yaml`
  - Configurable syntax highlight theme (key: `highlight.theme`, default: `github-dark`)
  
**Phase 5**:
  - `/` and `?` vim-like search
  - filename/summary, commiter columns

Make sure to update this file as the development progresses, especially the project structure and implementation phases sections.
