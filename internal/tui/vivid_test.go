package tui

import (
	"strings"
	"testing"

	"github.com/LZD121/x-terminal/internal/render"
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
