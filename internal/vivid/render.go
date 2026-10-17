package vivid

import (
	"strings"

	"github.com/LZD121/x-terminal/internal/render"
	"github.com/LZD121/x-terminal/internal/text"
)

// The presentation's own marks. They are the presentation's words and not a cover's, because nothing
// here is disguised: a new post is new, and a read one is quieter.
const (
	// MarkNew is the mark column's news mark: a plus in the one cell every list row starts with.
	MarkNew = "+"
)

// Render composes one frame: exactly Height rows of exactly Width cells, with the status bar as the
// last row and no trailing newline. It is the vivid surface's answer to render.Frame, and it holds
// the same contract — a full-height repaint that leaves no stale region — so the shell can write it
// into the alternate screen the way it writes the plain frame.
//
// Every row is assembled from already-painted pieces. The pieces are not truncated here: a block row
// wider than its column is a bug in the block builder, exactly as it is in render.Frame, and the one
// row truncation is legal on is the status bar, which cuts itself.
func Render(v View) string {
	if v.Width <= 0 || v.Height <= 0 {
		return ""
	}
	g := geometry(v.Width)
	rows := Rows(v.Height)
	out := make([]string, 0, v.Height)

	nav := navRows(v.Dest, g.Nav, rows)
	list := listRows(v, g.List, rows)
	var reading []string
	if g.TwoColumn {
		reading = readingRows(v, g.Reading, rows)
	}

	for i := range rows {
		row := ""
		switch {
		case g.TwoColumn:
			row = text.Pad(pick(nav, i), g.Nav) + Separator +
				text.Pad(pick(list, i), g.List) + Separator +
				text.Pad(pick(reading, i), g.Reading)
		default:
			// One column: the list owns the surface, and the destination still shows in the bar.
			row = text.Pad(pick(list, i), g.List)
		}
		out = append(out, text.Pad(row, v.Width))
	}
	out = append(out, text.Pad(render.StatusLine(v.Status, v.Width), v.Width))
	return strings.Join(out, "\n")
}

// cut is one string fitted to a column, with the module's own truncation mark when it had to lose
// cells. The mark goes in the last cell, which is what tells a cut row from one that merely ended
// there.
func cut(s string, w int) string {
	if text.Width(s) <= w {
		return s
	}
	if w <= 0 {
		return ""
	}
	if w == 1 {
		return render.MarkTruncation
	}
	return text.Truncate(s, w-1) + render.MarkTruncation
}

// pick is the i-th row of a column, or the empty string past its end: a column shorter than the pane
// is blank below its last row, which is what the fill between two columns has to be.
func pick(rows []string, i int) string {
	if i < 0 || i >= len(rows) {
		return ""
	}
	return rows[i]
}

// navRows is the navigation column: one row per destination, the current one marked, then its own
// heading above them. The heading is the presentation's one word for what the column is, and it is
// the only row that is not a destination.
func navRows(dest Destination, w, rows int) []string {
	if w < 1 || rows < 1 {
		return nil
	}
	out := make([]string, 0, len(Destinations())+2)
	for _, d := range Destinations() {
		mark := " "
		if d == dest {
			mark = render.MarkSelection
		}
		line := mark + " " + d.Name()
		if text.Width(line) > w {
			line = cut(line, w)
		}
		out = append(out, line)
		if len(out) >= rows {
			return out
		}
	}
	return out
}

// listRows is the list column: the surface's blocks, each row prefixed with the mark column. The
// prefix is the presentation's own cell, and it is why the list's blocks are built with one cell less
// of measure than the column — the shell does that, not this function.
func listRows(v View, w, rows int) []string {
	if w < 1 || rows < 1 {
		return nil
	}
	out := make([]string, 0, rows)
	for _, b := range v.List {
		isNew, dim := v.Marks.mark(b.PostID)
		for _, row := range b.Rows {
			line := markNew(row)
			if !isNew {
				line = blankMark(row)
			}
			if dim {
				line = dimRow(line)
			}
			if text.Width(line) > w {
				line = cut(line, w)
			}
			out = append(out, line)
			if len(out) >= rows {
				return out
			}
		}
	}
	return out
}

// readingRows is the reading column: a heading, then the conversation's own blocks. With nothing to
// show it is its notice, which is the shell's sentence rather than this package's.
func readingRows(v View, w, rows int) []string {
	if w < 1 || rows < 1 {
		return nil
	}
	out := make([]string, 0, rows)
	if v.ReadingTitle != "" {
		out = append(out, cut(v.ReadingTitle, w))
		out = append(out, cut(strings.Repeat("─", w), w))
	}
	if len(v.Reading) == 0 {
		if v.Notice != "" && len(out) < rows {
			out = append(out, cut(v.Notice, w))
		}
		return out
	}
	for _, b := range v.Reading {
		for _, row := range b.Rows {
			if text.Width(row) > w {
				row = cut(row, w)
			}
			out = append(out, row)
			if len(out) >= rows {
				return out
			}
		}
	}
	return out
}

// markNew prefixes the row's own mark column with the news mark. The column is the presentation's
// own cell, one cell wide, and every list row carries it: the shell builds the blocks with one cell
// less of measure than the column, so the prefix fits rather than pushing the row past its width, and
// the selection mark keeps column 2 where the block builder put it. A new post that is also selected
// therefore reads `+>`, which is two facts side by side rather than one replacing the other.
func markNew(row string) string {
	return MarkNew + row
}

// blankMark is the mark column of an ordinary row: one cell, so every row of the column starts at the
// same place whether or not it is news.
func blankMark(row string) string {
	return " " + row
}

// dimRow paints a read row in the muted role. The role is looked up once per row rather than stored,
// because the palette is the terminal's state and not the view's.
func dimRow(row string) string {
	pal := render.For(render.RungFromEnv())
	painted := pal.SGR(render.RoleMuted)
	if painted == "" {
		return row
	}
	return painted + row + "\x1b[0m"
}
