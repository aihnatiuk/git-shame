package highlight

import (
	"bytes"
	"io"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/charmbracelet/x/ansi"
)

// IndicatorSGR is the ANSI SGR sequence used to color whitespace indicators (· and →).
const IndicatorSGR = "\x1b[38;2;80;80;80m"

// wsFormatter is a Chroma formatter that embeds whitespace indicators directly
// into the ANSI output. Tabs become → and spaces become · at the token level,
// so their color is always independent of the surrounding syntax highlighting.
// Non-whitespace text is emitted with the token's TrueColor foreground as usual.
// Background colors are skipped; use PaintBackground if needed.
//
// When fgOverrides is set, lines at the given 0-based indices have their token
// foreground replaced with the provided SGR string while preserving bold/italic.
type wsFormatter struct {
	tabWidth    int
	fgOverrides map[int]string // 0-based line index → replacement fg SGR
	currentLine int
}

func (f *wsFormatter) Format(w io.Writer, style *chroma.Style, it chroma.Iterator) error {
	buf := toBuf(w)
	for token := it(); token != chroma.EOF; token = it() {
		entry := style.Get(token.Type)
		f.writeTokenValue(buf, entry, token.Value)
	}
	return nil
}

// toBuf returns w as a *bytes.Buffer if possible, otherwise panics — the only
// caller always passes a *bytes.Buffer so this avoids a heap allocation.
func toBuf(w io.Writer) *bytes.Buffer {
	return w.(*bytes.Buffer)
}

// buildSyntaxFmt writes the ANSI escape sequence for a token's text attributes
// into buf. When fgOverride is non-empty it replaces the entry's fg colour while
// preserving bold/italic/underline.
func buildSyntaxFmt(buf *bytes.Buffer, entry chroma.StyleEntry, fgOverride string) {
	if entry.IsZero() && fgOverride == "" {
		return
	}
	if entry.Bold == chroma.Yes {
		buf.WriteString("\x1b[1m")
	}
	if entry.Underline == chroma.Yes {
		buf.WriteString("\x1b[4m")
	}
	if entry.Italic == chroma.Yes {
		buf.WriteString("\x1b[3m")
	}
	if fgOverride != "" {
		buf.WriteString(fgOverride)
	} else if entry.Colour.IsSet() {
		r, g, bl := entry.Colour.Red(), entry.Colour.Green(), entry.Colour.Blue()
		buf.WriteString("\x1b[38;2;")
		writeInt(buf, int(r))
		buf.WriteByte(';')
		writeInt(buf, int(g))
		buf.WriteByte(';')
		writeInt(buf, int(bl))
		buf.WriteByte('m')
	}
}

// writeInt writes a small non-negative integer to b without fmt allocation.
func writeInt(b *bytes.Buffer, n int) {
	if n == 0 {
		b.WriteByte('0')
		return
	}
	var buf [3]byte
	i := 3
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	b.Write(buf[i:])
}

// currentLineFg returns the fg override for the current line, or "" if none.
func (f *wsFormatter) currentLineFg() string {
	if f.fgOverrides == nil {
		return ""
	}
	return f.fgOverrides[f.currentLine]
}

// writeTokenValue splits text on newlines and delegates each segment to
// writeSegment, emitting bare newline bytes between segments. This ensures
// every line is self-contained so individual lines can be displayed standalone.
// The formatter's currentLine counter is incremented at each newline so that
// per-line fg overrides are applied to the correct output lines.
func (f *wsFormatter) writeTokenValue(buf *bytes.Buffer, entry chroma.StyleEntry, text string) {
	for {
		newLineIndex := strings.IndexByte(text, '\n')
		if newLineIndex < 0 {
			break
		}
		segment := text[:newLineIndex]
		if newLineIndex > 0 && segment[newLineIndex-1] == '\r' {
			segment = segment[:newLineIndex-1]
		}
		syntaxFmt := f.takeSyntaxFmt(buf, entry)
		f.writeSegment(buf, syntaxFmt, segment)
		buf.WriteByte('\n')
		f.currentLine++
		text = text[newLineIndex+1:]
	}
	if len(text) > 0 {
		syntaxFmt := f.takeSyntaxFmt(buf, entry)
		f.writeSegment(buf, syntaxFmt, text)
	}
}

// takeSyntaxFmt builds the ANSI SGR prefix for the current line into a
// temporary buffer and returns it as a string, avoiding per-token heap
// allocations in the common case where no SGR is needed.
func (f *wsFormatter) takeSyntaxFmt(scratch *bytes.Buffer, entry chroma.StyleEntry) string {
	fgOverride := f.currentLineFg()
	if entry.IsZero() && fgOverride == "" {
		return ""
	}
	start := scratch.Len()
	buildSyntaxFmt(scratch, entry, fgOverride)
	end := scratch.Len()
	s := string(scratch.Bytes()[start:end])
	scratch.Truncate(start)
	return s
}

// writeSegment emits one line of a token value. Non-whitespace runs are wrapped
// with syntaxFmt; each space becomes a dim · and each tab becomes a dim →
// followed by plain spaces to preserve column alignment.
func (f *wsFormatter) writeSegment(w *bytes.Buffer, syntaxFmt, text string) {
	segStart := 0
	for i, r := range text {
		if r != ' ' && r != '\t' {
			continue
		}
		if i > segStart {
			if syntaxFmt != "" {
				w.WriteString(syntaxFmt)
			}
			w.WriteString(text[segStart:i])
			if syntaxFmt != "" {
				w.WriteString(ansi.ResetStyle)
			}
		}
		w.WriteString(IndicatorSGR)
		if r == '\t' {
			w.WriteString("→")
			w.WriteString(ansi.ResetStyle)
			w.WriteString(strings.Repeat(" ", f.tabWidth-1))
		} else {
			w.WriteString("·")
			w.WriteString(ansi.ResetStyle)
		}
		segStart = i + 1 // space and tab are both single bytes
	}
	if segStart < len(text) {
		if syntaxFmt != "" {
			w.WriteString(syntaxFmt)
		}
		w.WriteString(text[segStart:])
		if syntaxFmt != "" {
			w.WriteString(ansi.ResetStyle)
		}
	}
}
