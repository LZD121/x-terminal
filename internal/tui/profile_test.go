package tui

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/render"
)

// TestColdProfileFramePaintsItsHeaderAtTheTop is R3 / DISC-03 / 02.3 / 02.6 and the shell's own
// opening rule (model.go's skip). The frame composes the account's header as its first block and the
// selection is the first post, so a pane that measured only from the selection dropped the header
// the frame had just built: the bio, the counts and the join date were composed and never painted,
// and no key reached them, because nothing sits above the anchor.
//
// A frame that has just opened presents its own first row at the pane's first row. The assertion is
// over the pane's own bytes — Render's output for this state — and it is checked at the widths the
// corpus stores plus the two heights the live reproduction used, because the defect was a property
// of the pane's geometry and not of one size.
//
// The second half is the other direction: the header is a block of the list, so scrolling moves it
// off the pane. A frame that painted the header permanently — pinned above a scrolling list — would
// pass the first half and fail this one, which is why both are here.
func TestColdProfileFramePaintsItsHeaderAtTheTop(t *testing.T) {
	for _, w := range []int{60, 80, 120, 200} {
		for _, h := range []int{24, 30, 40} {
			t.Run(fmt.Sprintf("%dx%d", w, h), func(t *testing.T) {
				s := goldenShell(t, w, h)
				openFirstProfile(t, s)

				header, ok := s.m.top().Profile.Header()
				if !ok {
					t.Fatal("the profile frame holds no header after opening it")
				}
				if header.Bio == "" || header.Followers == nil || header.Joined.IsZero() {
					t.Fatalf("the fixture's header carries no bio, no counts and no join date, so nothing below is about them: %+v", header)
				}

				// The header's own rows are the pane's first rows: the identity row, every bio line,
				// then the facts row, in the renderer's own order and wrapping.
				head := render.ProfileHeader(header, s.m.grammar(), w)
				if len(head.Rows) < 3 {
					t.Fatalf("the header is %d rows: the assertion below would see no bio line and no facts row", len(head.Rows))
				}
				rows := strings.Split(s.pane(), "\n")
				for i, want := range head.Rows {
					got := strings.TrimRight(ansi.Strip(rows[i]), " ")
					if got != strings.TrimRight(want, " ") {
						t.Errorf("pane row %d is %q, want the header's own row %q: the frame built a header and did not paint it", i+1, got, want)
					}
				}
				// The three things DISC-03 names, by the header's own values: the identity row names
				// the account, the bio's sentence is on the pane, and the count the header answered
				// with is too — so the rows above are this account's header and not any three rows.
				if !strings.Contains(rows[0], header.User.ScreenName) || !strings.Contains(rows[0], header.User.Name) {
					t.Errorf("pane row 1 is %q, want the identity row of @%s (%s)", strings.TrimRight(rows[0], " "), header.User.ScreenName, header.User.Name)
				}
				for _, want := range []string{header.Bio, strconv.Itoa(*header.Followers), strconv.Itoa(*header.Posts)} {
					if !strings.Contains(s.pane(), want) {
						t.Errorf("the pane does not carry %q:\n%s", want, s.pane())
					}
				}

				// Scrolling moves the header off rather than the header never having been there: the
				// pane is a window into one list, so a pane scrolled past every header row no longer
				// holds it, and no row of it is re-drawn.
				s.press("G")
				if skip := s.m.scroll(s.m.top(), w); skip >= len(head.Rows)+1 {
					after := s.pane()
					for _, gone := range []string{header.User.Name, header.Bio, strconv.Itoa(*header.Followers)} {
						if strings.Contains(after, gone) {
							t.Errorf("the pane still carries %q at scroll %d: the header is a block of the list and must go with it", gone, skip)
						}
					}
				} else {
					t.Logf("at %dx%d the list is not taller than the pane by more than its header: there is no scroll past it to check", w, h)
				}
			})
		}
	}
}

// openFirstProfile drives the presses that open the first account of the people results frame. The
// profile frame's own first request is the header; the posts page follows it.
func openFirstProfile(t *testing.T, s *shell) {
	t.Helper()
	s.press("u")
	typeQuery(s, "people")
	s.press("enter")
	before := s.src.Calls()
	s.press("enter")
	if s.src.Calls() == before {
		t.Fatal("opening the profile issued no request at all")
	}
}

func TestProfileOpensInTwoRequests(t *testing.T) {
	s := newShell(t, 80, 24)
	before := s.src.Calls()
	s.used = before

	s.press("a")
	if s.src.Calls() != before+2 {
		t.Fatalf("`a` issued %d requests past %d, want exactly 2", s.src.Calls()-before, before)
	}
	handle, ok := s.src.LastProfile()
	if !ok || handle == "" {
		t.Fatal("Profile was not called")
	}
	id, ok := s.src.LastUserPosts()
	if !ok || id == "" {
		t.Fatal("UserPosts was not called")
	}
	if s.m.top().Kind != frameProfile {
		t.Fatalf("top kind=%v, want profile", s.m.top().Kind)
	}
	header, have := s.m.top().Profile.Header()
	if !have {
		t.Fatal("header did not arrive")
	}
	if id != header.User.ID {
		t.Fatalf("UserPosts id=%q, want header id %q", id, header.User.ID)
	}
	s.used = before + 2
	status := s.status()
	acct, named := s.m.profileAccount(s.m.top())
	if !named {
		t.Fatal("the answered profile frame names no account: the header's own account is the field's value once it has arrived")
	}
	if !strings.Contains(status, render.ListLabelProfile(s.m.grammar(), acct)) {
		t.Errorf("status %q missing profile list label", status)
	}
	if !strings.Contains(status, budgetOf(before+2, 0)) {
		t.Errorf("status %q missing two-request budget", status)
	}

	calls := s.src.Calls()
	s.press("a")
	if s.src.Calls() != calls {
		t.Fatalf("second `a` issued %d requests", s.src.Calls()-calls)
	}
}

// TestProfileInFlightPostsSaysLoadingPosts is 02.2 and the UI-SPEC's state table, asserted over the
// rendered status line. The header arrives first and the posts page is asked for second, so there is
// a state in which the pane holds a header and one request is out — and that state says `loading
// posts`. The defect this arm exists for is the one the phase shipped with: the walk was not latched
// when the header answer asked for the posts, so the frame reported an empty read that succeeded
// (`no posts`) while the request was still in the air, which is the honesty rule this phase exists
// to protect.
//
// The other half is the state the wording is for: a successful read that brought nothing says `no
// posts`, so the two can never be confused — the notice follows the read, not the pane's emptiness.
func TestProfileInFlightPostsSaysLoadingPosts(t *testing.T) {
	s := goldenShell(t, 80, 24)
	s.press("u")
	typeQuery(s, "people")
	s.press("enter")

	// Open the profile and run the header's own request only: the posts command the header answer
	// returns is held, which is exactly the state this arm is about.
	header := s.key("enter")
	if header == nil {
		t.Fatal("opening the profile issued no header request")
	}
	msg := header()
	if msg == nil {
		t.Fatal("the header request produced no message")
	}
	next, posts := s.m.update(msg)
	s.m = next
	if posts == nil {
		t.Fatal("the header answer issued no posts request")
	}
	if s.m.outstanding() != 1 {
		t.Fatalf("outstanding requests = %d, want the posts request alone", s.m.outstanding())
	}
	if _, have := s.m.top().Profile.Header(); !have {
		t.Fatal("the header is not on the frame after its own answer")
	}
	if s.m.top().Profile.Walk().Loaded() != 0 {
		t.Fatal("the posts page arrived before the test held its command")
	}
	if !s.m.top().Profile.Walk().InFlight() {
		t.Error("the walk is not latched while its first post page is out: the frame cannot tell an empty read from one in flight")
	}
	if got := s.m.noticeFor(s.m.top()); got != render.NoticeLoadingPosts {
		t.Errorf("notice is %v, want %v", got, render.NoticeLoadingPosts)
	}
	status := s.status()
	if !strings.Contains(status, render.CopyNoticeLoadingPosts) {
		t.Errorf("status is %q, want it to carry %q", status, render.CopyNoticeLoadingPosts)
	}
	if strings.Contains(status, render.CopyNoticeNoPosts) {
		t.Errorf("status is %q: it claims an empty read succeeded while a request is out", status)
	}
	// The header is on the pane while the posts are in flight (02.2's first clause): the state's own
	// words, and the identity row they are about.
	head, _ := s.m.top().Profile.Header()
	if pane := s.pane(); !strings.Contains(pane, head.User.Name) || !strings.Contains(pane, head.Bio) {
		t.Errorf("the pane does not carry the header while the posts are in flight:\n%s", pane)
	}

	// The answer: a successful read that brought no post says `no posts` — the same pane, the other
	// state, and the two wordings never on screen together.
	s.src.WithProfilePages([]domain.PostPage{{Posts: nil, Next: ""}})
	answer := posts()
	if answer == nil {
		t.Fatal("the posts request produced no message")
	}
	next, _ = s.m.update(answer)
	s.m = next
	if s.m.outstanding() != 0 {
		t.Fatalf("outstanding requests = %d after the answer", s.m.outstanding())
	}
	if s.m.top().Profile.Walk().InFlight() {
		t.Error("the walk is still latched after its page was answered")
	}
	if got := s.m.noticeFor(s.m.top()); got != render.NoticeNoPosts {
		t.Errorf("notice after a successful empty read is %v, want %v", got, render.NoticeNoPosts)
	}
	if status := s.status(); !strings.Contains(status, render.CopyNoticeNoPosts) {
		t.Errorf("status is %q, want it to carry %q", status, render.CopyNoticeNoPosts)
	}
}

func TestProfileHeaderArrivesBeforeThePosts(t *testing.T) {
	s := newShell(t, 80, 24)
	s.src.FailPostsAfterHeader()
	before := s.src.Calls()
	s.press("a")
	if s.src.Calls() < before+1 {
		t.Fatalf("expected at least the header request, got %d", s.src.Calls()-before)
	}
	header, have := s.m.top().Profile.Header()
	if !have {
		t.Fatal("header should remain after posts failure")
	}
	if header.User.ScreenName == "" {
		t.Fatal("header identity empty")
	}
	if s.m.failure.Kind == domain.FailureNone {
		t.Fatal("posts failure should reach the status line")
	}
}

func TestOpenAuthorWithoutAnAuthor(t *testing.T) {
	s := newShell(t, 80, 24)
	found := false
	for range 40 {
		post, ok := s.m.top().post(s.m.top().selected())
		if ok && post.Author.ScreenName == "" {
			found = true
			break
		}
		s.press("j")
	}
	if !found {
		t.Skip("adversarial set has no authorless post in the first 40 rows")
	}
	before := s.src.Calls()
	s.press("a")
	if s.src.Calls() != before {
		t.Fatalf("`a` on no author spent %d requests", s.src.Calls()-before)
	}
	if s.m.notice != render.NoticeNoAuthor {
		t.Fatalf("notice=%v, want NoticeNoAuthor", s.m.notice)
	}
}

// TestGoRestoresTheProfileHeader is `05-UI-SPEC.md:319-323`'s "the header is always one key away"
// and `05-02-PLAN.md`'s acceptance criterion — "`g` restores a pane whose first row is the identity
// row" — asserted over the pane's own rows, under every style and at both reference widths.
//
// The defect it closes: after any scroll the frame is moved and its anchor is the reader's place,
// so the header sat above the pane with no key that brought it back. `g` now asks for the frame's
// beginning, which is the state the frame opened in — the opening rule (model.go's skip) presents
// the frame's first row again — and the header and its one-key promise come back with it.
//
// The second half is the promise itself, read the only way a header can be reached: the header is
// not a selectable row, so "reachable by `k`" means the key that walks up from the first post
// cannot take the pane off the top again. Pressing `k` at the frame's first row moves nothing, and
// the identity row has to still be the pane's first row afterwards.
func TestGoRestoresTheProfileHeader(t *testing.T) {
	styles := append([]render.CoverStyle{""}, render.CoverStyles()...)
	for _, style := range styles {
		for _, w := range []int{80, 120} {
			name := "plain"
			if style != "" {
				name = string(style)
			}
			t.Run(name+"/"+strconv.Itoa(w), func(t *testing.T) {
				s := goldenShell(t, w, 24)
				s.m.style, s.m.state = style, StateNormal
				if style != "" {
					s.m.state = StateImmersive
				}
				openFirstProfile(t, s)

				header, ok := s.m.top().Profile.Header()
				if !ok {
					t.Fatal("the profile frame holds no header after opening it")
				}
				identity := strings.TrimRight(render.ProfileHeader(header, s.m.grammar(), w).Rows[0], " ")
				if !strings.Contains(identity, s.m.grammar().Identity(header.User).Handle) {
					t.Fatalf("the header's first row %q does not name the account: the assertion below is about nothing", identity)
				}

				// Scroll past the header: a few steps first — the reader the report was made from
				// had scrolled four rows — and then to the end of the list, which no reader's
				// scroll can leave the header on screen through.
				for range 4 {
					s.press("j")
				}
				s.press("G")
				f := s.m.top()
				if !f.Moved {
					t.Fatalf("the frame is not moved after a scroll: the anchor the defect is about is not in play")
				}
				if skip := s.m.scroll(f, w); skip < 4 {
					t.Fatalf("the pane dropped %d rows above its first row: the header is still on screen, so `g` has nothing to restore", skip)
				}
				if strings.Contains(s.pane(), identity) {
					t.Fatalf("the scrolled pane still carries the identity row %q:\n%s", identity, s.pane())
				}

				s.press("g")
				if !frames(s).unmoved() {
					t.Fatal("`g` left the frame moved: the pane is still the anchor's and not the frame's own first row")
				}
				rows := strings.Split(s.pane(), "\n")
				if got := strings.TrimRight(ansi.Strip(rows[0]), " "); got != identity {
					t.Errorf("pane row 1 after `g` is %q, want the identity row %q:\n%s", got, identity, s.pane())
				}

				// The promise: one `k` from the first post cannot take the header away again.
				s.press("k")
				rows = strings.Split(s.pane(), "\n")
				if got := strings.TrimRight(ansi.Strip(rows[0]), " "); got != identity {
					t.Errorf("pane row 1 after `g` then `k` is %q, want the identity row %q: the header is not one key away\n%s", got, identity, s.pane())
				}
			})
		}
	}
}

// topFacts is the frame's own two facts the assertion above reads: whether it is at its opening
// state and where its anchor is.
type topFacts struct {
	moved  bool
	anchor int
}

func (f topFacts) unmoved() bool { return !f.moved }

// frames reads the top frame's facts for an assertion's message.
func frames(s *shell) topFacts {
	f := s.m.top()
	return topFacts{moved: f.Moved, anchor: f.Anchor}
}
