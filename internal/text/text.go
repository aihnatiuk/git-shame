// Package text provides string utilities for terminal display.
package text

import "strings"

// ExpandTabs replaces tab characters with spaces of the given width.
// This makes width calculations correct for ansi.Truncate and ansi.TruncateLeft,
// which treat tabs as single characters otherwise.
func ExpandTabs(s string, tabWidth int) string {
	return strings.ReplaceAll(s, "\t", strings.Repeat(" ", tabWidth))
}
