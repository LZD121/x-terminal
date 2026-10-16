package render

import (
	"strconv"
	"strings"

	"github.com/LZD121/x-terminal/internal/text"
)

// HelpKeyCols is the width of the help overlay's key column, in cells: the same 16 cells as the
// list's identity column, so the overlay is the same shape as the pane it replaces.
const HelpKeyCols = 16

// HelpRow is one row of the help overlay. This package does not import the keymap: the shell maps
// its one table into these rows, which is what keeps the overlay, the input handling and the keys
// subcommand from each having their own copy of the table.
type HelpRow struct {
	// Keys is the key list for the row, already joined the way the table writes it.
	Keys string
	// Text is what the keys do, rendered verbatim.
	Text string
	// Group is which part of the surface the keys belong to. It is deliberately not printed: the
	// overlay is one flat list in table order, and a group heading would be a heading.
	Group string
}

// Help lays the keymap out as pane rows: one row per input row, in the order given, with the keys
// column exactly HelpKeyCols cells wide and the help text immediately after it. There is no
// heading, no border, no rule and no group column. Column 1 is the selection column, so the table
// starts at column 2 like every other row of the frame; a key list wider than its column is cut
// with the truncation mark in its last cell, because a keystroke that is not on screen is not
// documented.
//
// pane is how many rows the pane has, and the overlay is cut to fit it. When the table is taller
// than the pane, the pane's last row says how many of the table's rows are not on it: the overlay
// has no scroll of its own (no key moves it, so a scroll would be unreachable), and a keymap that
// silently hid a quarter of its rows would be a lie about what the program does. Naming the cut
// costs exactly one row, and it is a row of the same pane: at column 2, in the muted role, with no
// key column — so it cannot be read as a binding.
//
// The status line stays where it is while the overlay is open: the shell composes the rows into
// the pane and calls Frame with the same status fields as before.
func Help(rows []HelpRow, w, pane int) []string {
	pal := For(RungFromEnv())
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		keys := text.Pad(cutField(row.Keys, HelpKeyCols, MarkTruncation), HelpKeyCols)
		out = append(out, cutField(" "+pal.paint(RoleMuted, keys)+row.Text, w, MarkTruncation))
	}
	if pane <= 0 {
		return nil
	}
	if len(out) <= pane {
		return out
	}
	kept := pane - 1
	missing := len(out) - kept
	cut := MarkTruncation + " " + strings.Replace(CopyHelpClipped, "{n}", strconv.Itoa(missing), 1)
	// The cut row is one of the pane's rows, in the overlay's own two-column shape: the key column is
	// left empty — a binding always names its keys there — and the sentence starts where the help text
	// starts, marked with the truncation mark the rest of the frame cuts fields with.
	return append(out[:kept:kept], cutField(" "+text.Pad("", HelpKeyCols)+pal.paint(RoleMuted, cut), w, MarkTruncation))
}
