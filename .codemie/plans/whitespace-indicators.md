# Whitespace Indicators (Phase 2)

## Context

Phase 2 requires inline whitespace indicators in the code column (blame view) and diff view. Spaces show as `·` and tabs as `→`. This is on by default for now, with configuration planned for a later phase. The feature is exposed as a reusable utility function in `internal/text/text.go`.

## Approach

Apply `AddWhitespaceIndicators` **before** Chroma, replacing `ExpandTabs` at both call sites. Because `·` (U+00B7) and `→` (U+2192) have runewidth 1 — same as space — all width calculations remain correct. Chroma still highlights important tokens (keywords, strings, operators) correctly because Unicode non-word characters still act as token boundaries.

This is the only approach that preserves `→` for tabs, since tabs are lost after `ExpandTabs` runs.

## Files to Change

### 1. `internal/text/text.go` — add helper

Add `AddWhitespaceIndicators(s string, tabWidth int) string`:
- Iterate rune-by-rune
- `'\t'` → write `→` + `(tabWidth-1)` plain spaces (maintains visual width, plain spaces keep the tab extent visually uncluttered)
- `' '` → write `·`
- All other runes → pass through unchanged

Keep `ExpandTabs` as-is; it is still useful independently.

### 2. `internal/ui/blame/model.go:120-121` — replace call site

```go
// Before:
m.lines[i].Content = text.ExpandTabs(m.lines[i].Content, 4)
contents[i] = m.lines[i].Content

// After:
m.lines[i].Content = text.AddWhitespaceIndicators(m.lines[i].Content, 4)
contents[i] = m.lines[i].Content
```

`m.lines[i].Content` is used by `CalcMaxHScroll` (via `runewidth.StringWidth`) — width unchanged since `·` = runewidth 1 = space. Also used as fallback renderer when `highlightedLines` is shorter; will show `·` indicators consistently.

### 3. `internal/ui/diff/model.go:94` — replace call site

```go
// Before:
m.allDiffLines[i].Content = text.ExpandTabs(m.allDiffLines[i].Content, 4)

// After:
m.allDiffLines[i].Content = text.AddWhitespaceIndicators(m.allDiffLines[i].Content, 4)
```

`dl.Content` is read by both `buildHighlightedLines` (passes to Chroma) and `render.go:128` (`DiffRemoved` lines rendered directly). Both paths now show indicators. `DiffHunkHeader` / `DiffNoNewline` content (`@@ ... @@`, `\ No newline...`) contains no meaningful indentation so the substitution is harmless on those lines.

## Verification

```bash
go build -o shame.exe .
./shame <any-file>          # spaces show as ·, tabs show as →
# Press d                   # diff view: same indicators on added/context/removed lines
```
