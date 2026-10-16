package text

import (
	"github.com/charmbracelet/x/ansi"
)

// DropLastCluster returns s without its last grapheme cluster: what backspace means in a text
// field. The unit is the module's own iterator — the same one Wrap and Truncate walk with — so a
// backspace can never leave half a cluster on screen: an `e` with its combining mark, a ZWJ
// family and an emoji each go in one step. An empty string stays empty.
func DropLastCluster(s string) string {
	if s == "" {
		return s
	}
	m := method()
	last := 0
	for i := 0; i < len(s); {
		cluster, _ := ansi.FirstGraphemeCluster(s[i:], m)
		if cluster == "" {
			break
		}
		last = i
		i += len(cluster)
	}
	return s[:last]
}

// Tail returns the longest suffix of s that fits in w cells, cut at a cluster boundary: what a
// one-row field shows of a text longer than the row, so that the caret stays on screen. It is
// Truncate's mirror, walked with the same iterator and the same measure, and it appends nothing:
// the row's own prefix says what the field is, so a cut tail needs no mark of its own.
//
// Width(Tail(s, w)) <= w for every w >= 0, and the result is a suffix of s.
func Tail(s string, w int) string {
	if w < 1 {
		return ""
	}
	if Width(s) <= w {
		return s
	}
	m := method()
	// One pass to index the clusters, then a walk back from the end until the width is spent.
	// The index is a local of this call: nothing here is cached, because a width is a fact about
	// the environment the frame is drawn into.
	type span struct {
		start, end, cells int
	}
	var clusters []span
	for i := 0; i < len(s); {
		cluster, cw := ansi.FirstGraphemeCluster(s[i:], m)
		if cluster == "" {
			break
		}
		clusters = append(clusters, span{i, i + len(cluster), cw})
		i += len(cluster)
	}
	width, start := 0, len(s)
	for i := len(clusters) - 1; i >= 0; i-- {
		if width+clusters[i].cells > w {
			break
		}
		width += clusters[i].cells
		start = clusters[i].start
	}
	return s[start:]
}
