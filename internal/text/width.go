// Package text is the program's one width model: it measures a string (Width), breaks it into
// the lines a pane draws (Wrap), fills a field to a column (Pad) and cuts one short (Truncate).
// Every rendered line passes through these four functions, so a frame's geometry is decided in
// exactly one place. A second way to measure, or a wrap that disagrees with the measurement, is
// the defect that makes a terminal UI look broken in a way no screenshot catches: the terminal,
// not the frame, decides where a line it considers too long goes next.
//
// The measure follows the environment the frame is drawn into — see method — and the cluster
// iterator Wrap and Truncate walk comes from the same module as that measure, so a wrap point
// and a width can never come from two different ideas of what a cluster is.
//
// The package imports the standard library and github.com/charmbracelet/x/ansi, nothing else:
// nothing here formats a number, and nothing here counts bytes or code points — a code point is
// not a cell. It measures the text, not the escape sequences around it: Width counts an ANSI
// sequence as zero cells, as the module does, while Wrap and Truncate walk clusters and expect the
// unwrapped content the renderer colours afterwards.
package text

import (
	"os"

	"github.com/charmbracelet/x/ansi"
)

// method reports the width semantics of the terminal the frame is drawn into.
//
// $TMUX set means the drawing surface is a multiplexer's grid, which models a screen as cells
// holding one code point each: there a ZWJ family measures as the six cells its code points
// occupy (ansi.WcWidth). $TMUX unset means the program writes to a terminal directly — Ghostty
// here — which implements grapheme clusters: there the same family is one two-cell cluster
// (ansi.GraphemeWidth).
//
// This is the only branch on $TMUX for width in the program. A second one is a second measure,
// and two measures are how a frame comes to disagree with the terminal drawing it.
func method() ansi.Method {
	if os.Getenv("TMUX") != "" {
		return ansi.WcWidth
	}
	return ansi.GraphemeWidth
}

// Width returns the number of cells s occupies on screen: the program's only string
// measurement. ANSI escape sequences measure zero, and wide clusters measure what the running
// terminal will draw them as — two cells for a CJK character in either environment, two for a
// ZWJ family in a bare terminal and six under tmux (Method.StringWidth, the field dispatch both
// ansi.StringWidth and ansi.StringWidthWc call into).
func Width(s string) int {
	return method().StringWidth(s)
}
