package vivid

import (
	"strings"
	"testing"

	"github.com/LZD121/x-terminal/internal/render"
	"github.com/LZD121/x-terminal/internal/text"
)

// TestAMonoThemePaintsNothing is the degradation rule the whole style layer rests on: a terminal with
// no colour, or a reader who asked for none, gets the same frame with no escape bytes at all.
func TestAMonoThemePaintsNothing(t *testing.T) {
	mono := NewTheme(render.RungMono)
	if got := mono.Paint("text", ColourAccent, ColourSelection); got != "text" {
		t.Fatalf("a mono theme painted: %q", got)
	}
	if got := mono.Fill("text", 8, ColourAccent, ColourSelection); got != "text    " {
		t.Fatalf("a mono theme's fill is not plain padding: %q", got)
	}
	if got := mono.Gradient("heading", ColourAccent, ColourNew); got != "heading" {
		t.Fatalf("a mono theme gradient: %q", got)
	}
	if !mono.Mono() {
		t.Fatal("a mono theme does not report itself")
	}
}

// TestEveryRungPreservesWidth is the invariant that lets the frame's geometry stay exact: whatever a
// rung paints, the display width of the result is the display width of the input.
func TestEveryRungPreservesWidth(t *testing.T) {
	for _, rung := range []render.Rung{render.RungTrueColor, render.Rung256, render.Rung16, render.RungMono} {
		th := NewTheme(rung)
		for _, s := range []string{"plain", "a wide 日本語 row", "emoji 🚀 row", ""} {
			if got, want := text.Width(th.Paint(s, ColourAccent, ColourSelection)), text.Width(s); got != want {
				t.Fatalf("rung %v: Paint(%q) is %d cells, want %d", rung, s, got, want)
			}
			if got, want := text.Width(th.Gradient(s, ColourAccent, ColourNew)), text.Width(s); got != want {
				t.Fatalf("rung %v: Gradient(%q) is %d cells, want %d", rung, s, got, want)
			}
			if got, want := text.Width(th.Fill(s, 20, ColourAccent, ColourSelection)), 20; got != want {
				t.Fatalf("rung %v: Fill(%q) is %d cells, want 20", rung, s, got)
			}
		}
	}
}

// TestTheAuthorColourIsStableAndBounded: the same handle is the same colour in every run, and an
// empty handle is not a colour at all.
func TestTheAuthorColourIsStableAndBounded(t *testing.T) {
	first := AuthorColour("someone")
	for range 5 {
		if got := AuthorColour("someone"); got != first {
			t.Fatalf("the colour changed: %q then %q", first, got)
		}
	}
	// Case is not identity: the same person does not change colour because a payload capitalised
	// their handle.
	if AuthorColour("SomeOne") != first {
		t.Fatal("a capitalised handle changed colour")
	}
	if AuthorColour("") != ColourMuted {
		t.Fatal("an empty handle got an accent")
	}
	// Every value returned is one of the palette's, and the palette is more than one colour.
	seen := map[string]bool{}
	for _, h := range []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m"} {
		c := AuthorColour(h)
		seen[c] = true
		if !strings.HasPrefix(c, "#") || len(c) != 7 {
			t.Fatalf("%q got %q, which is not a hex colour", h, c)
		}
	}
	if len(seen) < 2 {
		t.Fatalf("thirteen handles produced %d colours: the hash is not spreading", len(seen))
	}
}

// TestTheGaugeIsTheFloor is the budget bar's arithmetic: full at the floor, half at half, and never
// wider than its cells.
func TestTheGaugeIsTheFloor(t *testing.T) {
	cases := []struct {
		used, floor, cells int
		want               string
	}{
		{0, 10, 10, "▱▱▱▱▱▱▱▱▱▱"},
		{5, 10, 10, "▰▰▰▰▰▱▱▱▱▱"},
		{10, 10, 10, "▰▰▰▰▰▰▰▰▰▰"},
		{25, 10, 10, "▰▰▰▰▰▰▰▰▰▰"},
		{-3, 10, 4, "▱▱▱▱"},
		// A floor below one is clamped up, so one request on a floor of one is a full bar.
		{1, 0, 4, "▰▰▰▰"},
	}
	for _, c := range cases {
		if got := Gauge(c.used, c.floor, c.cells); got != c.want {
			t.Fatalf("Gauge(%d,%d,%d) = %q, want %q", c.used, c.floor, c.cells, got, c.want)
		}
	}
	if Gauge(3, 10, 0) != "" {
		t.Fatal("a zero-cell gauge drew something")
	}
}
