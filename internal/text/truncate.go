package text

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// Truncate returns the longest prefix of s that fits in w cells, cut at a cluster boundary, with
// nothing appended: the `~` a caller prints to mark a cut field is the caller's literal, so a
// truncated field can be told from a naturally short one.
//
// The cut is the module's own Method.Truncate — a cut at a cluster boundary under the active
// method, and aware of the escape sequences Width ignores — rather than ansi.Truncate, which
// hardcodes grapheme semantics and would cut where Width does not agree. It is guarded by a
// re-measure and by a prefix check, because the pinned version (v0.11.9) gets a case wrong and
// because the module's cut carries escape sequences that follow the cut: its early return compares
// ansi.StringWidth, which is grapheme semantics whatever method it was asked for, so under tmux a
// cluster whose grapheme width is within w while its wcwidth width is not comes back whole and
// wider than w — the ZWJ family at w = 5 is two cells by grapheme and six by wcwidth, and all six
// are handed back — and with W = 3 on `abcdef\x1b[31m` it answers `abc\x1b[31m`, an escape sequence
// that lives past the cut and makes the result not a prefix of s at all. When the guard finds
// either, the prefix is walked back cluster by cluster here, with the iterator and the measure Wrap
// uses, so Width(Truncate(s, w)) <= w and strings.HasPrefix(s, Truncate(s, w)) hold in both
// environments.
func Truncate(s string, w int) string {
	if w < 1 {
		return ""
	}
	m := method()
	if cut := m.Truncate(s, w, ""); Width(cut) <= w && strings.HasPrefix(s, cut) {
		return cut
	}
	width, end := 0, 0
	for i := 0; i < len(s); {
		cluster, cw := ansi.FirstGraphemeCluster(s[i:], m)
		if width+cw > w {
			break
		}
		width += cw
		i += len(cluster)
		end = i
	}
	return s[:end]
}
