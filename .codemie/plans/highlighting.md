# Chroma Syntax Highlighting — Implementation Plan

## Goal
Add syntax highlighting to the `ColCode` column using [Chroma v2](https://github.com/alecthomas/chroma). Default theme: `github-dark`. Color format: TrueColor (16M). Theme will become configurable in Phase 4 (YAML config).

---

## Decisions
- Highlighted strings are stored as `[]string` on `blame.Model` (separate from `git.BlameLine.Content`, which stays raw).
- Language is detected by filename (`lexers.Match`), falling back to plain text.
- Highlighting runs once when `BlameResult` arrives — not on every render.
- On error, fall back silently to raw content (no crash, no visual noise).

---

## Files to Change

### 1. Add dependency
```bash
go get github.com/alecthomas/chroma/v2@v2.23.1
```

### 2. Create `internal/highlight/highlight.go` (new file)
- Package `highlight`
- `HighlightLines(filename string, lines []string) []string`
  - Detect language with `lexers.Match(filename)`, fallback to `lexers.Fallback`
  - Coalesce the lexer (`chroma.Coalesce`)
  - Use `styles.Get("github-dark")`, fallback to `styles.Fallback`
  - Use `formatters.TTY16M` (TrueColor ANSI formatter)
  - Join `lines` with `"\n"`, tokenize the whole source at once (needed for correct multi-line token context)
  - Format to `bytes.Buffer`, split output by `"\n"`
  - Handle trailing empty entry from final newline
  - Return the same number of elements as input `lines`; on any error return plain `lines`

### 3. `internal/ui/blame/model.go`
- Add field: `highlightedLines []string`
- In `Update` when handling `git.BlameResult` (success path):
  ```go
  contents := make([]string, len(msg.Lines))
  for i, l := range msg.Lines { contents[i] = l.Content }
  m.highlightedLines = highlight.HighlightLines(m.relFile, contents)
  ```
- Reset `m.highlightedLines = nil` in the loading/error paths so stale data isn't shown.

### 4. `internal/ui/blame/render.go`
- `RenderBody`: pass `m.highlightedLines[i]` to `RenderRow` (guard: if `highlightedLines` is nil or out-of-bounds, fall back to `line.Content`)
- `RenderRow`: add parameter `highlightedContent string`, pass it down to `renderCell`
- `renderCell`: add parameter `highlightedContent string`; use it instead of `line.Content` in the `ColCode` case

### 5. `internal/ui/blame/columns.go`
- Remove the `// Note: replace with ansi.StringWidth in Phase 2…` comment from `CalcMaxHScroll`, since `line.Content` remains raw text and `lipgloss.Width` is correct.

### 6. `CLAUDE.md`
- Mark "Chroma highlighting in blame" as done in Phase 2
- Add note: "github-dark is the default theme; will become user-configurable in Phase 4 via `~/.config/shame/config.yaml`"

---

## Key Chroma API (v2)
```go
import (
    "github.com/alecthomas/chroma/v2"
    "github.com/alecthomas/chroma/v2/formatters"
    "github.com/alecthomas/chroma/v2/lexers"
    "github.com/alecthomas/chroma/v2/styles"
)

lexer := lexers.Match(filename)   // nil if unknown
if lexer == nil { lexer = lexers.Fallback }
lexer = chroma.Coalesce(lexer)

style := styles.Get("github-dark")
if style == nil { style = styles.Fallback }

iter, _ := lexer.Tokenise(nil, source)
formatters.TTY16M.Format(&buf, style, iter)
```

---

## What We Are NOT Doing
- No per-render highlighting (too slow).
- No cache invalidation logic (highlighting is tied to blame load, which always replaces `lines`).
- No `ansi.StringWidth` change in `CalcMaxHScroll` (raw `Content` is still used there).
