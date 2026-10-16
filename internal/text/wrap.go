package text

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// Wrap breaks s into the lines a pane w cells wide draws it as: a body wrapped to the pane is
// the same shape the pane measures, so no line ever reaches the screen wider than the frame
// says it is.
//
// The break is greedy on spaces: a line takes every whole token that fits behind the space that
// separated it from the previous one, and a token that fits on a line of its own but not on the
// current line starts the next one rather than being cut. A token wider than the measure — a URL
// — is broken at a cluster boundary instead of being allowed to overflow. No hyphen is ever
// inserted: a hyphen is a claim about the text, and the break of a long word is not one.
//
// The returned lines hold, in order, every non-space character of s, with the runs of whitespace
// that separated tokens collapsed to the single space a greedy wrap leaves behind. A line whose
// content is entirely zero-width — a lone combining mark, say — is a line like any other: it holds
// those characters, and the no-loss rule is about them rather than about cells. The one exception is
// a cluster wider than the whole measure, which no line can hold without the line lying about its
// width: the measure wins and that cluster is not written (w < 1 is treated as 1, and the program's
// own measure cap keeps w far above the two cells a cluster takes). An empty input returns no lines.
//
// The walk is hand-written on purpose; only the cluster iterator is shared with the module that
// measures. ansi.Wordwrap (and its Method form) leaves a token wider than the limit unbroken —
// the 220-cell URL comes back as one 220-cell line in an 8-cell pane — and ansi.Wrap, which does
// break it, returns a newline-joined string that can hold a line wider than the limit itself
// ("\n" plus the whole ZWJ family at w = 1) and decides on its own where a broken token's
// remainder joins the next word and what becomes of the spaces it passed. This function answers
// with lines, none wider than the measure and none ending in a space, which is cheapest to keep
// by walking the clusters here.
func Wrap(s string, w int) []string {
	if s == "" {
		return nil
	}
	if w < 1 {
		w = 1
	}
	m := method()
	var (
		lines []string
		cur   strings.Builder
		curW  int
	)
	// flush ends the current line and starts the next. A separator space that ended up last
	// belongs to no line — no line of Wrap ends with a space — so it is dropped with the width
	// it took. A line is ended by its having content, not by its measuring anything: a token of
	// zero-width characters is content, and a wrap that dropped it would lose characters the text
	// has.
	flush := func() {
		if cur.Len() == 0 {
			return
		}
		line := strings.TrimRight(cur.String(), " ")
		cur.Reset()
		curW = 0
		lines = append(lines, line)
	}
	for _, tok := range strings.Fields(s) {
		tokW := Width(tok)
		switch {
		case cur.Len() == 0:
			// An empty line takes the token whatever it costs: there is no earlier break. The
			// test is the builder's own content rather than its width, which is zero for a
			// zero-width token that has already been written.
		case tokW <= w && curW+1+tokW <= w:
			// The token and the space separating it from the previous one fit.
			cur.WriteByte(' ')
			curW++
		case tokW <= w:
			// It fits on a line of its own, so it moves there instead of being cut.
			flush()
		case curW+1 < w:
			// The token is wider than the measure and this is where it breaks: the space
			// stays behind on this line, and the token takes what is left.
			cur.WriteByte(' ')
			curW++
		default:
			// Wider than the measure, and not even a space fits behind it here.
			flush()
		}
		for i := 0; i < len(tok); {
			cluster, cw := ansi.FirstGraphemeCluster(tok[i:], m)
			i += len(cluster)
			if curW+cw > w {
				flush()
			}
			if curW+cw > w {
				// Wider than the measure even on a line of its own: not written, because a
				// line whose width is a lie is worse than a lost cell.
				continue
			}
			cur.WriteString(cluster)
			curW += cw
		}
	}
	flush()
	return lines
}
