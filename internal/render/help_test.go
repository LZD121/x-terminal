package render

import (
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/LZD121/x-terminal/internal/text"
)

// helpRows is the keymap the overlay is laid out from, in the shell's own shape: the keys joined
// the way the table writes them and the help text verbatim.
func helpRows() []HelpRow {
	return []HelpRow{
		{Keys: "j, down", Text: "next post", Group: "navigate"},
		{Keys: "k, up", Text: "previous post", Group: "navigate"},
		{Keys: "ctrl+d", Text: "half a pane down", Group: "navigate"},
		{Keys: "g, home", Text: "first loaded post", Group: "navigate"},
		{Keys: "r", Text: "fetch newer posts, keep the place", Group: "feed"},
		{Keys: "t", Text: "switch between following and for you", Group: "feed"},
		{Keys: "?", Text: "show or hide this list", Group: "view"},
		{Keys: "q, esc", Text: "back, or quit at the start", Group: "app"},
		{Keys: "ctrl+c", Text: "quit now", Group: "app"},
	}
}

// helpPane is a pane taller than this file's sample table: the arms below are about the keys column,
// the width and the colour, and TestHelpNamesTheRowsThePaneCannotHold is the one about the cut.
const helpPane = 40

// TestHelpIsTheTable: one row per input row, in the order given, with the keys column exactly
// HelpKeyCols cells wide and the help text right behind it — no heading, no border, no rule, and no
// group column. The exact row is asserted, so a heading or a rule cannot be slipped in without
// failing here.
func TestHelpIsTheTable(t *testing.T) {
	plainEnv(t)
	rows := helpRows()
	out := Help(rows, 80, helpPane)
	if len(out) != len(rows) {
		t.Fatalf("the overlay has %d rows for %d bindings", len(out), len(rows))
	}
	for i, row := range rows {
		want := " " + text.Pad(row.Keys, HelpKeyCols) + row.Text
		if out[i] != want {
			t.Fatalf("row %d is %q, want %q", i+1, out[i], want)
		}
		if row.Group != "" && strings.Contains(out[i], row.Group) {
			t.Fatalf("row %d prints its group name: %q", i+1, out[i])
		}
		if got := text.Width(out[i][1 : 1+HelpKeyCols]); got != HelpKeyCols {
			t.Fatalf("row %d's keys column is %d cells, want %d", i+1, got, HelpKeyCols)
		}
	}
	if len(out) == 0 {
		t.Fatal("the overlay is empty, so every assertion above is vacuous")
	}
}

// TestHelpFitsThePane: at the widths the overlay is used at, a row that fits is the whole row and
// a row that does not is cut with the truncation mark rather than wrapping into the binding below
// it.
func TestHelpFitsThePane(t *testing.T) {
	plainEnv(t)
	rows := helpRows()
	for _, w := range []int{60, 80, 120, 200} {
		out := Help(rows, w, helpPane)
		if len(out) != len(rows) {
			t.Fatalf("at %d columns the overlay has %d rows, want %d", w, len(out), len(rows))
		}
		for i, row := range rows {
			want := " " + text.Pad(row.Keys, HelpKeyCols) + row.Text
			got := out[i]
			if text.Width(want) <= w {
				if got != want {
					t.Fatalf("at %d columns overlay row %d is %q, want %q", w, i+1, got, want)
				}
				continue
			}
			if !strings.HasSuffix(got, MarkTruncation) {
				t.Fatalf("at %d columns overlay row %d is cut without the mark: %q", w, i+1, got)
			}
			if !strings.HasPrefix(want, strings.TrimSuffix(got, MarkTruncation)) {
				t.Fatalf("at %d columns overlay row %d is not the row cut at the pane: %q", w, i+1, got)
			}
		}
	}
	if len(Help(rows, 200, helpPane)) == 0 {
		t.Fatal("the overlay is empty, so every assertion above is vacuous")
	}
}

// TestHelpCutsAKeyListItCannotHold: a key list wider than its column is cut with the truncation
// mark in the last cell rather than silently dropped, because a keystroke that is not on screen is
// not documented.
func TestHelpCutsAKeyListItCannotHold(t *testing.T) {
	plainEnv(t)
	row := HelpRow{Keys: "ctrl+alt+shift+super+x", Text: "a key list no column holds"}
	out := Help([]HelpRow{row}, 80, helpPane)
	if len(out) != 1 {
		t.Fatalf("the overlay has %d rows, want 1", len(out))
	}
	keys := out[0][1 : 1+HelpKeyCols]
	if !strings.HasSuffix(keys, MarkTruncation) {
		t.Fatalf("the cut key list is %q, want the truncation mark in its last cell", keys)
	}
	if got := text.Width(keys); got != HelpKeyCols {
		t.Fatalf("the keys column is %d cells, want %d", got, HelpKeyCols)
	}
	plain := Help([]HelpRow{{Keys: "q, esc", Text: "back, or quit at the start"}}, 80, helpPane)[0]
	if strings.Contains(plain, MarkTruncation) {
		t.Fatalf("a key list that fits was marked as cut: %q", plain)
	}
}

// TestHelpColourIsOnlyASecondStatement: the key column is the muted run, the help text is the
// dominant one, and stripping the colour leaves the mono overlay's characters.
func TestHelpColourIsOnlyASecondStatement(t *testing.T) {
	rows := helpRows()
	plainEnv(t)
	mono := Help(rows, 80, helpPane)
	colourEnv(t)
	colour := Help(rows, 80, helpPane)

	coloured := 0
	for i, row := range colour {
		if row != ansi.Strip(row) {
			coloured++
		}
		if got := ansi.Strip(row); got != mono[i] {
			t.Fatalf("row %d strips to %q, want the mono row %q", i+1, got, mono[i])
		}
	}
	if coloured == 0 {
		t.Fatal("the coloured overlay carries no escape: the key column is not painted")
	}
	if !strings.Contains(colour[0], sgrMutedTrueColor) {
		t.Fatalf("the key column is not the muted run: %q", colour[0])
	}
}

// TestHelpNamesTheRowsThePaneCannotHold: the keymap is taller than the pane a 24-row terminal has, so
// the overlay shows the rows it can and says how many it cannot. A quarter of the keymap must not
// disappear without a mark — the pane's own words are the only place the reader can learn that the
// list continues, because the overlay has no scroll and no key moves it.
//
// The numbers are the assertion as much as the words are: the pane holds 23 rows at H=24, the cut
// costs exactly one of them, and the count in the sentence is the table's remaining rows. The cut row
// is also asserted not to be a binding — its key column is empty — so it cannot be read as a key.
func TestHelpNamesTheRowsThePaneCannotHold(t *testing.T) {
	plainEnv(t)
	rows := keymapRows(false)
	if len(rows) < 4 {
		t.Fatalf("the table has %d rows: the cut could not be told from the whole overlay", len(rows))
	}

	// A pane that holds the table: every row, and no sentence about a cut.
	whole := Help(rows, 80, len(rows))
	if len(whole) != len(rows) {
		t.Fatalf("with room for the whole table the overlay has %d rows, want %d", len(whole), len(rows))
	}
	for i, row := range whole {
		if strings.Contains(row, "more keys than the pane can show") {
			t.Fatalf("row %d of an uncut overlay names a cut: %q", i+1, row)
		}
	}

	// The pane a 24-row terminal has: 22 bindings and one row naming the other 8.
	const pane = 23
	out := Help(rows, 80, pane)
	if len(out) != pane {
		t.Fatalf("the overlay is %d rows for a %d-row pane", len(out), pane)
	}
	shown := pane - 1
	for i := range shown {
		if out[i] != whole[i] {
			t.Fatalf("overlay row %d is %q, want the table's own row %q", i+1, out[i], whole[i])
		}
	}
	missing := len(rows) - shown
	want := strings.Replace(CopyHelpClipped, "{n}", strconv.Itoa(missing), 1)
	cut := out[pane-1]
	if !strings.Contains(cut, want) {
		t.Fatalf("the last row is %q, want it to carry %q for %d rows that do not fit", cut, want, missing)
	}
	if keys := strings.TrimSpace(cut[1 : 1+HelpKeyCols]); keys != "" {
		t.Fatalf("the cut row fills the key column with %q: it would read as a binding", keys)
	}
	if textCol := cut[1+HelpKeyCols:]; !strings.HasPrefix(textCol, MarkTruncation+" ") {
		t.Fatalf("the cut row's text column is %q, want it to open with the truncation mark", textCol)
	}
	if width := text.Width(cut); width > 80 {
		t.Fatalf("the cut row is %d cells wide in an 80-cell pane", width)
	}
	for i, row := range out {
		if width := text.Width(row); width > 80 {
			t.Fatalf("overlay row %d is %d cells wide in an 80-cell pane: %q", i+1, width, row)
		}
	}
	// One row is the whole cost of saying it: a pane one row taller shows one more binding.
	if taller := Help(rows, 80, pane+1); len(taller) != pane+1 {
		t.Fatalf("a %d-row pane shows %d rows", pane+1, len(taller))
	}
}
