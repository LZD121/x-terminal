package tui

import (
	"context"
	"strings"
	"testing"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/media"
	"github.com/LZD121/x-terminal/internal/render"
)

// linksShell is a shell whose selection is on the corpus's post with body links.
func linksShell(t *testing.T) *shell {
	t.Helper()
	s := newShell(t, 100, 30)
	for range 40 {
		post, ok := s.m.top().post(s.m.top().selected())
		if ok && len(post.Links) > 0 {
			return s
		}
		s.press("j")
	}
	t.Skip("the corpus holds no post with body links")
	return nil
}

// TestLIsTheMenuAndADigitFollowsAHashtag: `L` opens a menu that spends nothing, and a digit on a
// hashtag pushes a search frame and issues exactly one request.
func TestLIsTheMenuAndADigitFollowsAHashtag(t *testing.T) {
	s := linksShell(t)
	before := s.src.Calls()
	s.run(s.key("L"))
	if s.m.links == nil {
		t.Fatal("L did not open the menu")
	}
	if got := s.src.Calls(); got != before {
		t.Fatalf("opening the menu issued %d requests", got-before)
	}
	pane := s.pane()
	for _, want := range []string{"links", "1  example.invalid", "#example", "@someone"} {
		if !strings.Contains(pane, want) {
			t.Fatalf("the menu does not carry %q:\n%s", want, pane)
		}
	}
	// Digit 2 is the hashtag.
	s.press("2")
	if s.m.links != nil {
		t.Fatal("the menu stayed open after a choice")
	}
	if got := s.m.top().Kind; got != frameSearch {
		t.Fatalf("the hashtag reached frame kind %d, want the search frame", got)
	}
	if q := s.m.top().Search.Query(); q != "#example" {
		t.Fatalf("the search was for %q", q)
	}
}

// TestADigitFollowsAMention: the third row is a profile.
func TestADigitFollowsAMention(t *testing.T) {
	s := linksShell(t)
	s.press("L")
	s.press("3")
	if got := s.m.top().Kind; got != frameProfile {
		t.Fatalf("the mention reached frame kind %d, want a profile", got)
	}
	if h := s.m.top().Profile.Handle(); h != "someone" {
		t.Fatalf("the profile is %q, want someone", h)
	}
}

// TestADigitFollowsAURLThroughTheOpener: the first row leaves the program, through the same seam the
// permalink uses.
func TestADigitFollowsAURLThroughTheOpener(t *testing.T) {
	s := linksShell(t)
	opened := &fakeRenderer{}
	s.m.deps.Media = opened
	s.press("L")
	s.press("1")
	if len(opened.URLs) != 1 {
		t.Fatalf("%d URLs opened, want 1", len(opened.URLs))
	}
	if !strings.Contains(opened.URLs[0], "example.invalid") {
		t.Fatalf("opened %q", opened.URLs[0])
	}
	if s.m.links != nil {
		t.Fatal("the menu stayed open")
	}
}

// TestEscapeClosesTheMenuWithoutFollowing: escape restores the pane byte for byte.
func TestEscapeClosesTheMenuWithoutFollowing(t *testing.T) {
	s := linksShell(t)
	before := s.frame()
	s.press("L")
	if s.frame() == before {
		t.Fatal("the menu did not change the pane: the test would pass on a no-op")
	}
	s.press("esc")
	if s.frame() != before {
		t.Fatal("closing the menu did not restore the frame")
	}
}

// TestLOnAPostWithNoLinksSaysSo: a successful outcome in the notice field, and no menu.
func TestLOnAPostWithNoLinksSaysSo(t *testing.T) {
	s := newShell(t, 100, 30)
	s.press("L")
	if s.m.links != nil {
		t.Fatal("a menu opened over a post with no links")
	}
	if !strings.Contains(s.status(), "no link") {
		t.Fatalf("the status line says %q", s.status())
	}
}

// TestTheMenuIsUnavailableUnderACover: a list of addresses under a cover is the leak the disguise
// exists to prevent, so the key is refused before the menu opens.
func TestTheMenuIsUnavailableUnderACover(t *testing.T) {
	s := newShellIn(t, 100, 30, Display{State: StateImmersive, Cover: render.CoverServiceLog})
	s.press("L")
	if s.m.links != nil {
		t.Fatal("the menu opened under a cover")
	}
}

// fakeRenderer records the URLs it was asked to open.
type fakeRenderer struct {
	URLs []string
}

func (f *fakeRenderer) Open(_ context.Context, ref domain.MediaRef) (media.Result, error) {
	return media.Result{Outcome: media.OpenedImage}, nil
}

func (f *fakeRenderer) OpenURL(_ context.Context, url string) (media.Result, error) {
	f.URLs = append(f.URLs, url)
	return media.Result{Outcome: media.OpenedURL}, nil
}
