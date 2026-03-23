// Package highlight provides Chroma-based syntax highlighting for source lines.
// It produces ANSI TrueColor (16M) escape sequences suitable for terminal display.
package highlight

import (
	"bytes"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	chromastyles "github.com/alecthomas/chroma/v2/styles"
)

// DefaultTheme is the Chroma style used for syntax highlighting.
// Will become user-configurable in Phase 4 via ~/.config/shame/config.yaml.
const DefaultTheme = "github-dark"

// HighlightLines applies Chroma syntax highlighting to a slice of raw source lines.
// Language is detected from filename; plain text is used as a fallback.
// Returns ANSI-escaped strings with the same length as the input slice.
// On any error the input lines are returned unchanged.
func HighlightLines(filename string, lines []string) []string {
	result, err := highlightLines(filename, lines)
	if err != nil {
		return lines
	}
	return result
}

func highlightLines(filename string, lines []string) ([]string, error) {
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
	if err := formatters.TTY16m.Format(&buf, style, iterator); err != nil {
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
