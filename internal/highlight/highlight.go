// Package highlight provides Chroma-based syntax highlighting for source lines.
// It produces ANSI TrueColor (16M) escape sequences suitable for terminal display.
package highlight

import (
	"bytes"
	"image/color"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	chromastyles "github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/x/ansi"
)

// DefaultTheme is the Chroma style used for syntax highlighting.
// Will become user-configurable in Phase 4 via ~/.config/shame/config.yaml.
const DefaultTheme = "github-dark"

// HighlightLines applies Chroma syntax highlighting to a slice of raw source lines.
// Language is detected from filename; plain text is used as a fallback.
// Returns ANSI-escaped strings with the same length as the input slice.
// On any error the input lines are returned unchanged.
func HighlightLines(filename string, lines []string) []string {
	result, err := highlightLines(filename, lines, nil)
	if err != nil {
		return lines
	}
	return result
}

// HighlightLinesWithFgOverride applies Chroma syntax highlighting with optional
// per-line foreground overrides. overrides maps 0-based line indices to ANSI
// foreground SGR strings; for those lines the token foreground is replaced with
// the override while bold/italic/underline from the theme are preserved.
// On any error the input lines are returned unchanged.
func HighlightLinesWithFgOverride(filename string, lines []string, overrides map[int]string) []string {
	result, err := highlightLines(filename, lines, overrides)
	if err != nil {
		return lines
	}
	return result
}

// PaintBackground re-injects bg SGR after every \x1b[m / \x1b[0m in s.
// This is necessary because Chroma emits reset sequences between tokens that
// would otherwise clear any background color applied before the content.
func PaintBackground(s string, bg color.Color) string {
	bgSGR := ansi.Style{}.BackgroundColor(bg).String()
	s = strings.ReplaceAll(s, "\x1b[m", "\x1b[m"+bgSGR)
	s = strings.ReplaceAll(s, "\x1b[0m", "\x1b[0m"+bgSGR)
	return s
}

func highlightLines(filename string, lines []string, fgOverrides map[int]string) ([]string, error) {
	lexer := lexers.Match(filename)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	style := chromastyles.Get(DefaultTheme)
	if style == nil {
		style = chromastyles.Fallback
	}

	// Tokenise the whole source at once so multi-line constructs (e.g. block
	// comments, heredocs) are highlighted correctly.
	iterator, err := lexer.Tokenise(nil, strings.Join(lines, "\n"))
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := (&wsFormatter{tabWidth: 4, fgOverrides: fgOverrides}).Format(&buf, style, iterator); err != nil {
		return nil, err
	}

	// Chroma tokens never span newlines, so splitting the ANSI output by "\n"
	// gives one independently-styled string per source line.
	result := strings.Split(buf.String(), "\n")

	// Trim the trailing empty entry produced by the final newline.
	if len(result) > 0 && result[len(result)-1] == "" {
		result = result[:len(result)-1]
	}

	// Guard: ensure the slice is at least as long as the input.
	for len(result) < len(lines) {
		result = append(result, "")
	}

	return result[:len(lines)], nil
}
