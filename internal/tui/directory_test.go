package tui

import (
	"strings"
	"testing"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/fake"
)

// profileShell walks a shell to a profile frame through the people search — the path a reader
// takes, and the one that gives the frame a header with the numeric id the account reads are
// addressed by.
func profileShell(t *testing.T) (*shell, *fake.Directory) {
	t.Helper()
	s := peopleShell(t, 100, 30)
	s.press("u")
	typeQuery(s, "people")
	s.press("enter")
	s.press("enter")
	if got := s.m.top().Kind; got != frameProfile {
		t.Fatalf("the walk reached the frame kind %d, want the profile", got)
	}
	dir, _ := s.m.deps.Directory.(*fake.Directory)
	return s, dir
}

// TestEachListKeyAsksForItsOwnOperation is the four keys' whole contract: each pushes one frame and
// issues exactly one request, addressed by the profile's own id and naming its own read. A key that
// asked for the wrong operation would render a plausible list of the wrong people.
func TestEachListKeyAsksForItsOwnOperation(t *testing.T) {
	cases := []struct {
		key    string
		kind   domain.UserListKind
		expect func(fake.DirectoryCall) bool
	}{
		{"f", domain.FollowingList, func(c fake.DirectoryCall) bool {
			return !c.Media && c.Kind == domain.FollowingList
		}},
		{"e", domain.FollowersList, func(c fake.DirectoryCall) bool {
			return !c.Media && c.Kind == domain.FollowersList
		}},
		{"v", domain.VerifiedFollowersList, func(c fake.DirectoryCall) bool {
			return !c.Media && c.Kind == domain.VerifiedFollowersList
		}},
		{"M", domain.FollowingList, func(c fake.DirectoryCall) bool {
			return c.Media
		}},
	}
	for _, tc := range cases {
		t.Run(tc.key, func(t *testing.T) {
			s, dir := profileShell(t)
			owner, ok := s.m.frameOwner()
			if !ok || owner.ID == "" {
				t.Fatal("the profile has no account id to address the read with")
			}
			before := len(dir.Calls())
			s.press(tc.key)
			calls := dir.Calls()
			if len(calls) != before+1 {
				t.Fatalf("%d requests, want exactly 1", len(calls)-before)
			}
			got := calls[len(calls)-1]
			if !tc.expect(got) {
				t.Fatalf("the key asked for %+v", got)
			}
			if got.UserID != owner.ID {
				t.Fatalf("the request addressed %q, want the profile's id %q", got.UserID, owner.ID)
			}
		})
	}
}

// TestTheAccountFrameShowsTheRowsAndGoesBack: the pushed frame holds the answer's rows, and `q`
// returns to the profile with its own state intact — no refetch, no second request.
func TestTheAccountFrameShowsTheRowsAndGoesBack(t *testing.T) {
	s, dir := profileShell(t)
	before := len(dir.Calls())
	s.press("f")
	if got := s.m.top().Kind; got != frameAccounts {
		t.Fatalf("f reached the frame kind %d, want an account list", got)
	}
	rows := s.m.top().Accounts.Walk().Rows()
	if len(rows) == 0 {
		t.Fatal("the frame has no rows: the answer was not absorbed")
	}
	if !strings.Contains(s.status(), "following") {
		t.Fatalf("the status line does not name the list: %q", s.status())
	}
	s.press("q")
	if got := s.m.top().Kind; got != frameProfile {
		t.Fatalf("q reached the frame kind %d, want the profile back", got)
	}
	if got := len(dir.Calls()); got != before+1 {
		t.Fatalf("returning issued %d requests after the one page, want none", got-before-1)
	}
}

// TestAListOpenedFromAListKeepsTheOwner is the chained walk: `f` then `e` from the following list is
// still the same account's lists, because the owner is carried on the frame rather than re-derived
// from a profile that is no longer on top.
func TestAListOpenedFromAListKeepsTheOwner(t *testing.T) {
	s, dir := profileShell(t)
	owner, _ := s.m.frameOwner()
	s.press("f")
	s.press("e")
	if got := s.m.top().Kind; got != frameAccounts || s.m.top().AccountsKind != domain.FollowersList {
		t.Fatalf("the second key left %v/%v", s.m.top().Kind, s.m.top().AccountsKind)
	}
	if got := s.m.top().Owner.ID; got != owner.ID {
		t.Fatalf("the chained list is of %q, want %q", got, owner.ID)
	}
	calls := dir.Calls()
	if got := calls[len(calls)-1].UserID; got != owner.ID {
		t.Fatalf("the chained request addressed %q", got)
	}
}

// TestNoDirectoryIsANoticeAndNoRequest: a build without the port says so and spends nothing, the
// way a build without a translation endpoint answers `T`.
func TestNoDirectoryIsANoticeAndNoRequest(t *testing.T) {
	s, _ := profileShell(t)
	s.m.deps.Directory = nil
	s.press("f")
	if got := s.m.top().Kind; got != frameProfile {
		t.Fatalf("a shell with no lists pushed the frame kind %d", got)
	}
	if s.m.notice == 0 {
		t.Fatal("the key said nothing")
	}
}

// TestTheMediaFrameIsAPostList: the media tab absorbs into a posts walk, so `enter` opens a
// conversation from it exactly as from any other list of posts.
func TestTheMediaFrameIsAPostList(t *testing.T) {
	s, _ := profileShell(t)
	dir := s.m.deps.Directory.(*fake.Directory)
	dir.Posts = domain.PostPage{Posts: []domain.Post{{
		ID:     "m-1",
		Author: domain.User{ID: "a-1", ScreenName: "someone", Name: "Some One"},
		Text:   "an attachment",
	}}}
	s.press("M")
	if got := s.m.top().Kind; got != frameMedia {
		t.Fatalf("M reached the frame kind %d", got)
	}
	if got := len(s.m.top().Media.Walk().Rows()); got != 1 {
		t.Fatalf("%d media rows, want 1", got)
	}
	if !strings.Contains(s.status(), "media") {
		t.Fatalf("the status line does not name the tab: %q", s.status())
	}
}
