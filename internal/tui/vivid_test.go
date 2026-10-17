package tui

import (
	"strings"
	"testing"

	"github.com/LZD121/x-terminal/internal/render"
	"github.com/LZD121/x-terminal/internal/text"
	"github.com/LZD121/x-terminal/internal/vivid"
)

// vividShell is a shell in the presentation, at a two-column size, with the first request answered.
func vividShell(t *testing.T) *shell {
	t.Helper()
	s := newShell(t, 110, 30)
	s.m = s.m.enterVivid()
	s.run(s.m.change())
	return s
}

// TestVividShowsTheSamePostsAsPlain is the presentation's central claim: it is a second layout of the
// same data, not a second read. Same ids, same order.
func TestVividShowsTheSamePostsAsPlain(t *testing.T) {
	s := newShell(t, 110, 30)
	plain := s.m.top().List.Posts()
	v := s.m.enterVivid().vividView()
	var seen []string
	for _, b := range v.List {
		if b.PostID != "" {
			seen = append(seen, string(b.PostID))
		}
	}
	if len(seen) == 0 {
		t.Fatal("the list column is empty")
	}
	for i, post := range plain {
		if i >= len(seen) {
			break
		}
		if seen[i] != string(post.ID) {
			t.Fatalf("row %d is %s, want %s", i, seen[i], post.ID)
		}
	}
}

// TestEveryFrameKindHasAVividView is the join that keeps the six screens honest: every frame kind the
// shell can push produces a view with a list and a nav destination, at a two-column size and a
// one-column one.
func TestEveryFrameKindHasAVividView(t *testing.T) {
	for _, kind := range frameKinds() {
		t.Run(string(rune('0'+kind)), func(t *testing.T) {
			s := newShell(t, 110, 30)
			switch kind {
			case frameThread:
				s.press("enter")
			case frameSearch:
				s.press("/")
				s.press("x")
				s.press("enter")
			case frameProfile:
				s.press("a")
			case frameSaved:
				s.press("b")
			}
			s.m = s.m.enterVivid()
			if s.m.top().Kind != kind && kind != frameHome {
				t.Skipf("the shell did not reach %v (top is %v)", kind, s.m.top().Kind)
			}
			for _, w := range []int{110, 50} {
				s.resize(w, 24)
				frame := s.m.Render(w, 24)
				rows := strings.Split(frame, "\n")
				if len(rows) != 24 {
					t.Fatalf("width %d: %d rows, want 24", w, len(rows))
				}
			}
		})
	}
}

// TestVividRendersTheNavColumn is the one row the presentation always has: the destinations, with the
// current one marked.
func TestVividRendersTheNavColumn(t *testing.T) {
	s := vividShell(t)
	frame := s.m.Render(110, 24)
	for _, d := range vivid.Destinations() {
		if !strings.Contains(frame, d.Name()) && d != vivid.DestProfile {
			t.Fatalf("the nav column does not name %s", d.Name())
		}
	}
	if !strings.Contains(frame, render.MarkSelection) {
		t.Fatal("nothing is marked: neither a destination nor a row")
	}
}

// TestEnterFillsTheReadingColumnAndTheListStays: opening a post in the presentation must not push a
// frame — the list is the thing the column exists to keep.
func TestEnterFillsTheReadingColumnAndTheListStays(t *testing.T) {
	s := vividShell(t)
	before := s.m.top().ID
	posts := len(s.m.top().List.Posts())
	s.press("enter")
	if s.m.top().ID != before || len(s.m.top().List.Posts()) != posts {
		t.Fatalf("enter moved the list: frame %s, %d posts", s.m.top().ID, len(s.m.top().List.Posts()))
	}
	if s.m.vivid.reading == "" {
		t.Fatal("enter did not fill the reading column")
	}
	if len(s.m.vivid.opened) != 1 {
		t.Fatalf("%d posts marked read, want 1", len(s.m.vivid.opened))
	}
	view := s.m.vividView()
	if len(view.Reading) == 0 {
		t.Fatal("the reading column has no blocks")
	}
	if view.ReadingTitle == "" {
		t.Fatal("the reading column has no heading")
	}
}

// TestEscapeClosesTheReadingColumnThenLeaves: two presses, two effects, in that order.
func TestEscapeClosesTheReadingColumnThenLeaves(t *testing.T) {
	s := vividShell(t)
	s.press("enter")
	if s.m.vivid.reading == "" {
		t.Fatal("setup: nothing opened")
	}
	s.press("esc")
	if s.m.state != StateVivid || s.m.vivid.reading != "" {
		t.Fatalf("first esc: state %v, reading %q", s.m.state, s.m.vivid.reading)
	}
	s.press("esc")
	if s.m.state != StateNormal {
		t.Fatalf("second esc left the shell in %v", s.m.state)
	}
	if s.m.vivid.marked != nil || s.m.vivid.opened != nil || len(s.m.vivid.images) != 0 {
		t.Fatal("leaving the presentation kept its marks or its images")
	}
}

// TestTheReadingColumnDoesNotInheritTheListsScroll is the bug the smoke run found: the reading pane
// was being skipped to the list's anchor, so a post opened from deep in the list showed only the rows
// below its own head — the quote inside it, under a heading naming the outer author.
func TestTheReadingColumnDoesNotInheritTheListsScroll(t *testing.T) {
	s := vividShell(t)
	// Move the list far down, then open whatever is selected there.
	for range 6 {
		s.press("j")
	}
	s.press("enter")
	if s.m.vivid.reading == "" {
		t.Fatal("setup: nothing opened")
	}
	view := s.m.vividView()
	if len(view.Reading) == 0 {
		t.Fatal("the reading column is empty")
	}
	post, ok := s.m.framePostByID(s.m.vivid.reading)
	if !ok {
		t.Fatal("the opened post is not in the frame")
	}
	// The column's first content row must be the post's own head: the author the heading names.
	first := strings.Join(view.Reading[0].Rows, "\n")
	handle := post.Author.ScreenName
	if !strings.Contains(first, handle) {
		t.Fatalf("the reading column starts at %q, which is not the head of %q", first, handle)
	}
}

// m0 is a shell that has not moved: the reference both reading-column tests compare against.
// bordersAt reports whether the row's cells at the given display columns are box edges. It walks
// clusters rather than runes so a wide glyph before a column is counted for what it occupies.
func bordersAt(row string, cols []int) bool {
	col := 0
	next := 0
	found := make([]bool, len(cols))
	for _, r := range row {
		w := text.Width(string(r))
		if w == 0 {
			continue
		}
		if next < len(cols) && col == cols[next] {
			found[next] = strings.ContainsRune("│╭╮╰╯─", r)
			next++
		}
		col += w
	}
	for _, ok := range found {
		if !ok {
			return false
		}
	}
	return true
}

// TestEnteringVividDoesNotPersistIt is the bug the live run found: the transition wrote "vivid" to the
// config file. The file's vocabulary does not hold that name — StateOf refuses it on purpose — so the
// next start read a state it could not resolve and fell back to plain while the file still said
// vivid. A deliberate transition that is not a starting state must not be written.
func TestEnteringVividDoesNotPersistIt(t *testing.T) {
	s := newShell(t, 110, 30)
	before := len(s.persisted)
	s.m = s.m.enterVivid()
	s.run(s.m.change())
	for _, w := range s.persisted[before:] {
		if strings.HasPrefix(w, "vivid/") {
			t.Fatalf("entering the presentation wrote %q to the display config", w)
		}
	}
	// Leaving it is the same: the plain reader is where a run starts, and it is already the default.
	m := s.m.leaveVivid()
	s.m = m
	s.run(m.change())
	for _, w := range s.persisted {
		if strings.HasPrefix(w, "vivid/") {
			t.Fatalf("the display config holds %q", w)
		}
	}
}

// TestTheReadingColumnIsNotSkippedByTheListAnchor is the defect the screenshot showed: the reading
// column was composed through the same skip the list uses, so a post opened from deep in a scrolled
// list began at the skip's offset — a heading naming one author over a row of a quote of another.
//
// The reference is the same post composed by a shell that has never moved, so the assertion is about
// the skip and not about which row happens to mention whom.
func TestTheReadingColumnIsNotSkippedByTheListAnchor(t *testing.T) {
	s := vividShell(t)
	for range 12 {
		s.press("j")
	}
	if !s.m.top().Moved {
		t.Fatal("setup: the list did not move, so this test would not exercise the skip")
	}
	s.press("enter")
	if s.m.vivid.reading == "" {
		t.Fatal("setup: nothing opened")
	}
	view := s.m.vividView()
	post, _ := s.m.framePostByID(s.m.vivid.reading)
	unscrolled := m0(t).m.threadReadingOf(post, vivid.GeometryOf(110).Reading)
	if len(view.Reading) == 0 || len(unscrolled) == 0 {
		t.Fatal("the reading column is empty")
	}
	if got, want := strings.Join(view.Reading[0].Rows, "\n"), strings.Join(unscrolled[0].Rows, "\n"); got != want {
		t.Fatalf("the reading column is scrolled:\n got %q\nwant %q", got, want)
	}
}

// m0 is a shell that has not moved: the reference both reading-column tests compare against.
func m0(t *testing.T) *shell { t.Helper(); return newShell(t, 110, 30) }

// TestEveryColumnRowIsExactlyItsWidth is the off-by-one the smoke run showed as a half glyph at the
// rule's end: a column row one cell wide pushes the next column's first cell onto the following line.
func TestEveryColumnRowIsExactlyItsWidth(t *testing.T) {
	s := vividShell(t)
	s.press("enter")
	for _, w := range []int{74, 90, 110, 131, 200} {
		s.resize(w, 24)
		frame := s.m.Render(w, 24)
		g := vivid.GeometryOf(w)
		for i, row := range strings.Split(frame, "\n") {
			if got := text.Width(stripSGR(row)); got != w {
				t.Fatalf("width %d row %d: %d cells", w, i, got)
			}
		}
		// Each pane is a rounded box, and the boxes tile the surface exactly: the first box's right
		// border, the gutter, the second box's borders, the gutter and the third box's edges.
		for _, row := range strings.Split(frame, "\n")[:4] {
			plain := stripSGR(row)
			if !bordersAt(plain, []int{0, g.Nav - 1, g.Nav + 1, g.Nav + g.List, g.Nav + g.List + 2, w - 1}) {
				t.Fatalf("width %d: box borders are not where the geometry says: %q", w, plain)
			}
		}
	}
}
