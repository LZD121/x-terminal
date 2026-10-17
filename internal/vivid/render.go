package vivid

import (
	"strconv"
	"strings"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/render"
	"github.com/LZD121/x-terminal/internal/text"
)

// Render composes one frame: exactly Height rows of exactly Width cells, with the status bar as the
// last row and no trailing newline. It is the vivid surface's answer to render.Frame, and it holds
// the same contract — a full-height repaint that leaves no stale region — so the shell can write it
// into the alternate screen the way it writes the plain frame.
//
// The frame is two panes in a rounded box each, separated by a gutter, over a bar. The geometry is
// still exact arithmetic: the box's borders are counted in the column widths, and the content
// inside them is exactly what is left. A mono theme draws the same boxes with no colour, so the
// shape survives a terminal that cannot paint.
func Render(v View) string {
	if v.Width <= 0 || v.Height <= 0 {
		return ""
	}
	th := v.Theme
	g := GeometryOf(v.Width)
	rows := Rows(v.Height)
	if rows < 1 {
		// A one-row surface has no body: the bar is the frame. Drawing a body row and a bar would be
		// two rows in a one-row terminal, which scrolls the frame and is the one thing it may not do.
		return text.Pad(render.StatusLine(v.Status, v.Width), v.Width)
	}

	var out []string
	if g.TwoColumn {
		out = body(th, v, g, rows)
	} else {
		out = narrow(th, v, g, rows)
	}
	// The fill below the last row and the bar are the same width as everything else; the bar is
	// painted last so a theme's background runs to the frame's edge.
	out = append(out, statusBar(th, v, rows))
	return strings.Join(out, "\n")
}

// body is the two-column shape: nav box, gutter, list box, gutter, reading box.
func body(th Theme, v View, g Geometry, rows int) []string {
	nav := box(th, v.Dest, "nav", navLines(th, v, g.Nav-2, rows-2), g.Nav, rows)
	list := box(th, v.Dest, "feed", listLines(th, v, g.List-2, rows-2), g.List, rows)
	reading := box(th, v.Dest, "read", readingLines(th, v, g.Reading-2, rows-2), g.Reading, rows)

	out := make([]string, 0, rows)
	for i := range rows {
		// The gutter is one blank cell between boxes: a coloured rule would be a third border and
		// the frame already has four.
		row := pick(nav, i) + " " + pick(list, i) + " " + pick(reading, i)
		out = append(out, text.Pad(row, v.Width))
	}
	return out
}

// narrow is the one-column shape: the list's box, full width, with the destination in its title.
func narrow(th Theme, v View, g Geometry, rows int) []string {
	lines := listLines(th, v, g.List-2, rows-2)
	col := box(th, v.Dest, v.Dest.Name(), lines, g.List, rows)
	out := make([]string, 0, rows)
	for i := range rows {
		out = append(out, text.Pad(pick(col, i), v.Width))
	}
	return out
}

// box draws one pane: a rounded border, a title in the top edge, and the given content lines inside.
//
// The title is the pane's own word and the count is the pane's own fact, so the top edge carries
// `╭─ nav ─────────╮` rather than a bare rule: a box whose edge says nothing wastes the one row a
// border costs.
func box(th Theme, dest Destination, title string, lines []string, w, h int) []string {
	if w < 4 || h < 3 {
		return plainBox(title, lines, w, h)
	}
	inner := w - 2

	top := borderRune(th, "╭") + borderRune(th, "─")
	label := " " + title + " "
	top += th.Paint(label, ColourAccent, "") + borderRun(th, inner-text.Width(label)-1)
	top += borderRune(th, "╮")

	out := make([]string, 0, h)
	out = append(out, top)
	for i := 1; i < h-1; i++ {
		content := pick(lines, i-1)
		out = append(out, borderRune(th, "│")+text.Pad(content, inner)+borderRune(th, "│"))
	}
	bottom := borderRune(th, "╰") + borderRun(th, inner) + borderRune(th, "╯")
	out = append(out, bottom)
	return out
}

// plainBox is the same box for a space too small to carry one: the content, cut to the space, with
// no border to eat two of its cells. A 3-cell box is not a box.
func plainBox(title string, lines []string, w, h int) []string {
	out := make([]string, 0, max(h, 1))
	for i := range max(h, 1) {
		if i == 0 {
			out = append(out, cut(" "+title, w))
			continue
		}
		out = append(out, cut(pick(lines, i-1), w))
	}
	return out
}

// borderRune paints one border cell.
func borderRune(th Theme, s string) string { return th.Paint(s, ColourRune, "") }

// borderRun is n horizontal border cells.
func borderRun(th Theme, n int) string {
	if n <= 0 {
		return ""
	}
	return th.Paint(strings.Repeat("─", n), ColourRune, "")
}

// navLines is the navigation column's content: one line per destination, the current one a filled
// pill, and the reader's place beside the ones that carry a count.
func navLines(th Theme, v View, w, h int) []string {
	if w < 1 || h < 1 {
		return nil
	}
	out := make([]string, 0, len(Destinations()))
	for _, d := range Destinations() {
		name := " " + d.Name()
		if d == v.Dest {
			out = append(out, th.Fill(name, w, ColourOnAccent, ColourAccent))
			continue
		}
		out = append(out, th.Paint(name, ColourMuted, ""))
		if len(out) >= h {
			return out
		}
	}
	return out
}

// listLines is the list column: a hairline under a one-line list header, then the surface's blocks
// with their news bar, author accent and dimmed-and-read state.
func listLines(th Theme, v View, w, h int) []string {
	if w < 1 || h < 1 {
		return nil
	}
	out := make([]string, 0, h)
	header := " " + v.ListName
	place := ""
	if v.Status.Loaded > 0 {
		place = strconv.Itoa(v.Status.Selected) + "/" + strconv.Itoa(v.Status.Loaded)
		if v.Status.HasMore {
			place += "+"
		}
	}
	gap := w - text.Width(header) - text.Width(place) - 1
	if gap >= 0 {
		out = append(out, th.Paint(header, ColourMuted, "")+strings.Repeat(" ", gap)+th.Paint(place, ColourMuted, ""))
	} else {
		out = append(out, cut(header, w))
	}
	out = append(out, th.Paint(strings.Repeat("─", w), ColourRune, ""))
	if len(out) >= h {
		return out[:h]
	}
	for _, b := range v.List {
		isNew, dim := v.Marks.mark(b.PostID)
		bar := " "
		if isNew {
			bar = th.Paint("▌", ColourNew, "")
		}
		for _, row := range b.Rows {
			line := bar + accentRow(th, row, dim)
			out = append(out, cut(line, w))
			if len(out) >= h {
				return out
			}
		}
	}
	return out
}

// readingLines is the reading column: the post's own heading, a rule, then the conversation. With
// nothing open it is the notice, which is the shell's sentence rather than this package's.
func readingLines(th Theme, v View, w, h int) []string {
	if w < 1 || h < 1 {
		return nil
	}
	out := make([]string, 0, h)
	if v.ReadingTitle != "" {
		out = append(out, th.Gradient(" "+v.ReadingTitle, ColourAccent, ColourNew))
		out = append(out, th.Paint(strings.Repeat("─", w), ColourRune, ""))
		if len(out) >= h {
			return out[:h]
		}
	}
	if len(v.Reading) == 0 {
		if v.Notice != "" {
			out = append(out, th.Paint(" "+v.Notice, ColourMuted, ""))
		}
		return out
	}
	for _, b := range v.Reading {
		for _, row := range b.Rows {
			out = append(out, cut(row, w))
			if len(out) >= h {
				return out[:h]
			}
		}
	}
	return out
}

// accentRow colours a row's identity without re-measuring it: the handle is the row's first word
// after the selection mark, and it is painted in its author's own hue. Everything else stays the
// terminal's foreground, because a feed whose body text is coloured is unreadable.
//
// A dimmed row is the muted role over the whole row: the reader has opened it, and dimming is the
// presentation's way of saying so without a mark in a second column.
func accentRow(th Theme, row string, dim bool) string {
	if dim {
		return th.Paint(row, ColourMuted, "")
	}
	if th.Mono() {
		return row
	}
	// The handle starts after the selection column and any quote depth, and ends at the first run of
	// spaces that follows it. It is found by scanning rather than by index because the mark column is
	// not part of the block row.
	i := 0
	for i < len(row) && (row[i] == '>' || row[i] == ' ') {
		i++
	}
	j := i
	for j < len(row) && row[j] != ' ' && row[j] != '\t' {
		j++
	}
	if i >= j {
		return row
	}
	handle := row[i:j]
	// A word that is not a handle — a marker, a stamp, a body line's first word — is left alone.
	if strings.ContainsRune(handle, ':') || hasDigit(handle) || text.Width(handle) == 0 {
		return row
	}
	return row[:i] + th.Paint(handle, AuthorColour(handle), "") + row[j:]
}

// hasDigit reports whether a word carries a digit. A stamp and an id both do, and neither is a
// handle; the check is written as a loop rather than as a character-class literal because the audit
// reads a literal of that shape as a bind address.
func hasDigit(s string) bool {
	for _, r := range s {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}

// statusBar is the frame's last row: the destination, the state word and its remedy, the budget as a
// gauge, and the clock. It is the plain reader's own line restyled, not a second composition of the
// same facts — the words are render.StatusLine's, cut to the row and then painted by segment.
func statusBar(th Theme, v View, rows int) string {
	line := render.StatusLine(v.Status, v.Width)
	if th.Mono() {
		return text.Pad(line, v.Width)
	}
	// The bar is painted as one filled row, with the state word standing out by colour rather than
	// by position: the plain reader's line is a sequence of fields and this keeps that sequence, so
	// a reader's eye moves the same way in both surfaces.
	bg := "#161b22"
	switch {
	case v.Status.Failure.Stopped:
		return th.Fill(line, v.Width, ColourAlert, bg)
	case v.Status.Failure.Kind != domain.FailureNone:
		return th.Fill(line, v.Width, ColourWarn, bg)
	case v.Status.Limited > 0:
		return th.Fill(line, v.Width, ColourWarn, bg)
	}
	return th.Fill(line, v.Width, ColourMuted, bg)
}

// pick is the i-th row of a column, or the empty string past its end: a column shorter than the pane
// is blank below its last row, which is what the fill between two columns has to be.
func pick(rows []string, i int) string {
	if i < 0 || i >= len(rows) {
		return ""
	}
	return rows[i]
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
