package vivid

import (
	"strings"
	"testing"

	"github.com/LZD121/x-terminal/internal/domain"

	"github.com/LZD121/x-terminal/internal/render"
	"github.com/LZD121/x-terminal/internal/text"
)

// TestGeometry is the split's whole arithmetic as one table: the columns always sum to the width,
// the second column appears exactly at the floor, and nothing is ever given a negative cell.
func TestGeometry(t *testing.T) {
	for _, w := range []int{1, 20, 59, 73, 74, 75, 80, 120, 200, 997} {
		g := GeometryOf(w)
		if g.Nav < 0 || g.List < 0 || g.Reading < 0 {
			t.Fatalf("width %d: a column is negative: %+v", w, g)
		}
		if g.TwoColumn {
			if sum := g.Nav + g.List + g.Reading + 2; sum != w {
				t.Fatalf("width %d: columns + separators = %d, want %d (%+v)", w, sum, w, g)
			}
			if g.Reading < g.List {
				t.Fatalf("width %d: the reading pane (%d) is narrower than the list (%d)", w, g.Reading, g.List)
			}
		} else if g.List != w {
			t.Fatalf("width %d: one column must own the surface, list = %d", w, g.List)
		}
	}
	if GeometryOf(MinTwoColumn-1).TwoColumn || !GeometryOf(MinTwoColumn).TwoColumn {
		t.Fatalf("the floor is not where the second column appears: %d", MinTwoColumn)
	}
	// The floor is derived, not written down: the widest destination plus padding plus the plain
	// reader's own floor is exactly where it sits.
	widest := 0
	for _, d := range Destinations() {
		if n := text.Width(d.Name()); n > widest {
			widest = n
		}
	}
	if MinTwoColumn != render.MinPaneCols+widest+2+2 {
		t.Fatalf("the floor %d does not follow the longest destination %d", MinTwoColumn, widest)
	}
}

// TestEveryRowIsExactlyTheWidth is the frame contract, at every size the presentation will see and
// a few it will not: h rows of w cells, always, with the status bar last. Composed rows carry SGR
// escapes from the muted role, so the assertion is on display width, not bytes.
func TestEveryRowIsExactlyTheWidth(t *testing.T) {
	posts := []render.Block{
		{PostID: "1", Rows: []string{"> one", "  body"}},
		{PostID: "2", Rows: []string{"  two", "  body"}},
	}
	for _, w := range []int{1, 2, 40, 73, 74, 80, 121, 200} {
		for _, h := range []int{1, 2, 3, 24, 60} {
			v := View{
				Dest:     DestHome,
				ListName: "home",
				List:     posts,
				Marks:    Marks{New: map[domain.PostID]bool{"1": true}, Opened: map[domain.PostID]bool{"2": true}},
				Notice:   "nothing open",
				Reading:  []render.Block{{PostID: "1", Rows: []string{" a thread", " more"}}},
				Width:    w,
				Height:   h,
			}
			frame := Render(v)
			if h < 1 {
				continue
			}
			rows := strings.Split(frame, "\n")
			if len(rows) != h {
				t.Fatalf("%dx%d: %d rows, want %d", w, h, len(rows), h)
			}
			for i, row := range rows {
				if n := text.Width(row); n != w {
					t.Fatalf("%dx%d: row %d is %d cells, want %d: %q", w, h, i, n, w, row)
				}
			}
		}
	}
}

// TestTheNavigationColumnMarksTheDestination is the column's one behavioural claim: it lists every
// destination in order and marks the current one, and nothing else.
func TestTheNavigationColumnMarksTheDestination(t *testing.T) {
	labels := map[string]string{
		"home": "home", "search": "search", "bookmarks": "bookmarks", "likes": "likes", "profile": "profile",
	}
	for name, word := range labels {
		var dest Destination
		for _, d := range Destinations() {
			if d.Name() == name {
				dest = d
			}
		}
		rows := navRows(dest, 20, 40)
		if len(rows) != len(Destinations()) {
			t.Fatalf("%s: %d rows, want %d", name, len(rows), len(Destinations()))
		}
		for i, row := range rows {
			want := "  " + Destinations()[i].Name()
			if Destinations()[i].Name() == word {
				want = render.MarkSelection + " " + word
			}
			if row != want {
				t.Fatalf("%s: row %d = %q, want %q", name, i, row, want)
			}
		}
	}
}
