package vivid

import (
	"github.com/LZD121/x-terminal/internal/render"
	"github.com/LZD121/x-terminal/internal/text"
)

// Geometry is where the columns are, in cells, for one surface width. It is a value rather than five
// ints passed around so that a test can hold the whole answer at once and every column reads the same
// one.
//
// The two separators are what makes the arithmetic exact: nav + sep + list + sep + reading == width,
// and the reading column has no right margin — its last cell is the surface's last cell, because a
// gap there would read as a third pane that does not exist.
type Geometry struct {
	Nav, List, Reading int
	// TwoColumn is whether the reading column is drawn at all. Below MinTwoColumn there is one
	// column, and the reading pane is where a post is opened instead of beside the list — the
	// presentation degrades rather than truncating either column.
	TwoColumn bool
}

// Separator is the column between two panes: one cell of a vertical rule. It is a cell of its own so
// that the panes' content widths are the widths the block grammar was measured with.
const Separator = "│"

// minNavWidth is the narrowest a nav column may be and still name every destination: the longest
// name plus one cell of padding on each side, plus the box's two border cells. A column narrower
// than this silently drops a destination — which is how "bookmarks" vanished from a live frame.
func minNavWidth() int {
	return navWidth() + 2
}

// MinTwoColumn is the width at which the second column appears. It is the plain reader's own floor
// (render.MinPaneCols) plus the nav column and two separators: below the plain floor a post cannot be
// laid out, so a second column beside it cannot either. It is a var because the nav column's width is
// computed from the destination names — a new destination must move the floor rather than be cut.
var MinTwoColumn = render.MinPaneCols + minNavWidth() + 1

// NavWidth is the navigation column's width: the longest destination name plus one cell of padding
// on each side. It is computed from the names rather than written down, so a new destination widens
// the column instead of being cut.
func navWidth() int {
	widest := 0
	for _, d := range Destinations() {
		if n := text.Width(d.Name()); n > widest {
			widest = n
		}
	}
	return widest + 2
}

// GeometryOf splits a surface width into its columns.
//
// Below MinTwoColumn the whole surface is the list, and the nav column is not drawn either: a
// twelve-cell column on a fifty-cell terminal would leave the list too narrow to read a post in,
// which is the same failure the plain reader refuses. The status bar still names the destination, so
// the reader is never lost.
func GeometryOf(width int) Geometry {
	if width < MinTwoColumn {
		return Geometry{List: width}
	}
	nav := minNavWidth()
	// Two one-cell gutters between the three boxes.
	rest := width - nav - 2
	// The list gets the smaller half of what is left and the reading pane the larger: the list is
	// a menu of one-line heads and the pane is prose. An odd cell goes to the list, because a stray
	// cell at the pane's right edge is less visible than one at the list's.
	list := rest / 2
	return Geometry{Nav: nav, List: list, Reading: rest - list, TwoColumn: true}
}

// Rows is how many rows a pane occupies: the surface minus the status bar.
func Rows(height int) int {
	if height < 2 {
		return 0
	}
	return height - 1
}
