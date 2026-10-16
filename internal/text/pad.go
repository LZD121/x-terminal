package text

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// Pad returns s followed by as many spaces as make it exactly w cells wide, for a field that has
// to hold a column of the frame whatever is put in it: a head row's identity column is the same
// width for every post, or the columns behind it drift.
//
// Width(Pad(s, w)) == w for every w >= 1. A string wider than its column is cut with Truncate
// first, and the spaces are counted from what the cut actually left — Truncate lands on a
// cluster boundary, so a wide cluster straddling the limit leaves the field a cell short (or
// empty), and the fill closes that gap rather than overflowing the column. w < 1 returns "":
// nothing can be zero cells wide.
//
// The fill is measured rather than subtracted, because two shapes make "Width(s) + n" and
// "Width(s + n spaces)" two different numbers. A trailing UAX#29 Prepend character absorbs the
// space that follows it into its cluster, so the field is one cell short — the cluster takes that
// cell and no more, and one more space closes the field. A trailing escape sequence that never
// terminates swallows every byte appended behind it, so no fill can close the field at all: those
// bytes draw nothing, and they are dropped, cluster by cluster, instead of being allowed to
// swallow the column.
func Pad(s string, w int) string {
	if w < 1 {
		return ""
	}
	if Width(s) > w {
		s = Truncate(s, w)
	}
	for {
		width := Width(s)
		if width >= w {
			return s
		}
		filled := s + strings.Repeat(" ", w-width)
		switch got := Width(filled); {
		case got == w:
			return filled
		case got > width:
			// The fill counted, and one of its cells was absorbed by a trailing Prepend cluster:
			// the shortfall is what the cluster took, and the spaces that follow a counted space
			// are counted themselves, so one more run of them closes the field.
			return filled + strings.Repeat(" ", w-got)
		default:
			// Not one cell of the fill is counted. The bytes at the end of s open an escape
			// sequence that nothing terminates, and everything appended behind those bytes is
			// swallowed: the whole cluster they are is dropped and the field measured again. The
			// walk ends at the empty string at the latest, where the fill alone is the column.
			s = s[:len(s)-len(lastCluster(s))]
		}
	}
}

// lastCluster returns the last grapheme cluster of s, by the same iterator Wrap and Truncate walk
// with. It is the unit an unterminated escape sequence is dropped as: the module's iterator treats
// the introducer and everything it swallows as one cluster, so no half sequence is ever left behind.
func lastCluster(s string) string {
	m := method()
	last := s
	for i := 0; i < len(s); {
		cluster, _ := ansi.FirstGraphemeCluster(s[i:], m)
		if cluster == "" {
			break
		}
		last = cluster
		i += len(cluster)
	}
	return last
}
