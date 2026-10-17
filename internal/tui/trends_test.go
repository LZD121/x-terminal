package tui

import (
	"strings"
	"testing"

	"github.com/LZD121/x-terminal/internal/render"
)

// TestExploreListsTrendsAndEnterSearches: `x` pushes the frame and issues one request, and `enter`
// on a row searches that row's own query — not its name, which is only what is printed.
func TestExploreListsTrendsAndEnterSearches(t *testing.T) {
	s := newShell(t, 100, 30)
	before := s.src.Calls()
	s.press("x")
	if got := s.m.top().Kind; got != frameTrends {
		t.Fatalf("x reached the frame kind %d", got)
	}
	if len(s.m.top().Trends) != 2 {
		t.Fatalf("%d trends", len(s.m.top().Trends))
	}
	if got := s.src.Calls(); got != before {
		t.Fatalf("opening Explore spent %d graph requests", got-before)
	}
	if !strings.Contains(s.status(), "trends") {
		t.Fatalf("the status line says %q", s.status())
	}
	// Second row's query, not its name.
	s.press("j")
	s.press("enter")
	if got := s.m.top().Kind; got != frameSearch {
		t.Fatalf("enter reached frame kind %d, want a search", got)
	}
	if q := s.m.top().Search.Query(); q != "#b" {
		t.Fatalf("searched %q, want the second trend's query", q)
	}
}

// TestExploreRefusesWithoutThePort: a build with no directory says so and pushes nothing.
func TestExploreRefusesWithoutThePort(t *testing.T) {
	s := newShell(t, 100, 30)
	s.m.deps.Directory = nil
	s.press("x")
	if got := s.m.top().Kind; got != frameHome {
		t.Fatalf("a shell with no directory reached the frame kind %d", got)
	}
	if s.m.notice == 0 {
		t.Fatal("the key said nothing")
	}
}

// TestExploreIsDrawnUnderACoverWithTheStylesWords: the surface works under a cover and its status
// field is the cover's word.
func TestExploreIsDrawnUnderACover(t *testing.T) {
	s := newShellIn(t, 100, 30, Display{State: StateImmersive, Cover: render.CoverServiceLog})
	s.press("x")
	if got := s.m.top().Kind; got != frameTrends {
		t.Fatalf("under a cover x reached %d", got)
	}
	if !strings.Contains(s.status(), render.CopyCoverListTrends) {
		t.Fatalf("the covered status line is %q", s.status())
	}
}
