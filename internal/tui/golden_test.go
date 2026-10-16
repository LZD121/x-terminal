package tui

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/exp/golden"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/render"
)

// goldenNow is the corpus clock: every stored frame's stamps and budget reset are rendered against it.
var goldenNow = time.Date(2026, 10, 14, 9, 12, 0, 0, time.UTC)

const goldenH = 24

// The four profile cases 05-03's audit names. The words live here rather than in corpus_test.go
// because the frames are generated from them: the audit reads the stored bytes back and compares them
// with the same values, so a frame and its assertion cannot drift apart.
const (
	// threeLineBio wraps to three lines at the 80-column pane's own measure (77 cells) and no fewer:
	// the case is that a header's bio keeps every line, so the audit reassembles the header's bio rows
	// and compares them with this sentence.
	threeLineBio = "reads timelines in a terminal, mostly the slow ones, and writes down what the cursor did when it came back around, which is where the interesting part usually is"
	// repostHandle is the account the repost case was reposted by: the plain grammar's repost marker
	// names the account that surfaced the post, so the marker's text is "reposted by " + this.
	repostHandle = "lzd"
	// quotedBody is the quoted post's own text in the quote case.
	quotedBody = "the quoted body, one level down"
)

// caseProfile is the header the four case frames carry: the adversarial profile's own shape, so a
// case frame differs from the others only in the post set the case is about (or, for the bio case, in
// the bio).
func caseProfile() domain.Profile {
	followers, following, posts := 7, 3, 11
	return domain.Profile{
		User:      domain.User{ID: "42", ScreenName: "alice", Name: "Maria"},
		Bio:       "reads timelines in a terminal",
		Followers: &followers,
		Following: &following,
		Posts:     &posts,
		Joined:    time.Date(2011, time.March, 1, 0, 0, 0, 0, time.UTC),
	}
}

// casePost is one post of a case frame: a one-line body, so the case it carries is on the pane's first
// rows and the frame is about the case rather than about the wrap.
func casePost(id, handle, body string) domain.Post {
	return domain.Post{
		ID:             domain.PostID(id),
		Author:         domain.User{ID: "u-" + handle, ScreenName: handle, Name: "name of " + handle},
		Text:           body,
		CreatedAt:      goldenNow.Add(-10 * time.Minute),
		ConversationID: id,
	}
}

type goldenEnv struct {
	tmux, term, colorterm, noColor string
}

func (e goldenEnv) pin(t *testing.T) {
	t.Helper()
	t.Setenv("TZ", "UTC")
	t.Setenv("TMUX", e.tmux)
	t.Setenv("TERM", e.term)
	t.Setenv("COLORTERM", e.colorterm)
	t.Setenv("NO_COLOR", e.noColor)
}

var (
	gEnvPlain     = goldenEnv{term: "dumb", noColor: "1"}
	gEnvTrueColor = goldenEnv{term: "xterm-256color", colorterm: "truecolor"}
	gEnv256       = goldenEnv{term: "xterm-256color"}
	gEnv16        = goldenEnv{term: "xterm"}
	gEnvMono      = goldenEnv{term: "xterm-256color", noColor: "1"}
)

// goldenShell builds a Model driven by AdversarialPeople (posts + people) with a pinned clock.
func goldenShell(t *testing.T, w, h int) *shell {
	t.Helper()
	s := peopleShell(t, w, h)
	s.now = goldenNow
	s.used = s.src.Calls()
	return s
}

func typeQuery(s *shell, q string) {
	for _, r := range q {
		s.press(string(r))
	}
}

// goldenFrames is the phase's stored view specification: each name is a UI-SPEC frame.
func goldenFrames(t *testing.T) []struct {
	name  string
	w     int
	env   goldenEnv
	drive func(*testing.T, *shell)
} {
	searchCJK := "日本語 🔍 timeline"
	longQuery := strings.Repeat("q", 40)

	openSearch := func(q string) func(*testing.T, *shell) {
		return func(t *testing.T, s *shell) {
			s.press("/")
			typeQuery(s, q)
			s.press("enter")
			s.used = s.src.Calls()
		}
	}
	openPeople := func(q string) func(*testing.T, *shell) {
		return func(t *testing.T, s *shell) {
			s.press("u")
			typeQuery(s, q)
			s.press("enter")
			s.used = s.src.Calls()
		}
	}
	openProfile := func(handle string) func(*testing.T, *shell) {
		return func(t *testing.T, s *shell) {
			s.press("u")
			typeQuery(s, "people")
			s.press("enter")
			// Prefer the first account; open it.
			s.press("enter")
			s.used = s.src.Calls()
			_ = handle
		}
	}

	frames := []struct {
		name  string
		w     int
		env   goldenEnv
		drive func(*testing.T, *shell)
	}{
		{"search_60", 60, gEnvPlain, openSearch(searchCJK)},
		{"search_80", 80, gEnvPlain, openSearch(searchCJK)},
		{"search_120", 120, gEnvPlain, openSearch(searchCJK)},
		{"search_200", 200, gEnvPlain, openSearch(searchCJK)},
		{"search_top_80", 80, gEnvPlain, func(t *testing.T, s *shell) {
			openSearch("page cache")(t, s)
			s.press("s")
			s.used = s.src.Calls()
		}},
		{"search_people_80", 80, gEnvPlain, openPeople("people")},
		{"search_people_120", 120, gEnvPlain, openPeople("people")},
		{"search_empty_80", 80, gEnvPlain, func(t *testing.T, s *shell) {
			s.src.EmptySearch()
			openSearch("nothing")(t, s)
		}},
		{"profile_60", 60, gEnvPlain, openProfile("alice")},
		{"profile_80", 80, gEnvPlain, openProfile("alice")},
		{"profile_120", 120, gEnvPlain, openProfile("alice")},
		{"profile_200", 200, gEnvPlain, openProfile("alice")},
		{"profile_empty_80", 80, gEnvPlain, func(t *testing.T, s *shell) {
			s.src.WithProfile(domain.Profile{
				User: domain.User{ID: "empty", ScreenName: "empty", Name: "Empty"},
				Bio:  "reads timelines in a terminal",
			})
			s.src.WithProfilePages([]domain.PostPage{{Posts: nil, Next: ""}})
			openProfile("empty")(t, s)
		}},
		{"profile_bare_80", 80, gEnvPlain, func(t *testing.T, s *shell) {
			s.press("u")
			typeQuery(s, "people")
			s.press("enter")
			if rows := s.m.top().People.Walk().Rows(); len(rows) > 0 {
				s.src.WithProfile(domain.Profile{
					User: domain.User{ID: rows[0].ID, ScreenName: rows[0].ScreenName, Name: "Bare"},
				})
				s.src.WithProfilePages([]domain.PostPage{{Posts: nil, Next: ""}})
			}
			s.press("enter")
			s.used = s.src.Calls()
		}},
		{"profile_loading_80", 80, gEnvPlain, func(t *testing.T, s *shell) {
			// Header arrived, posts in flight: open profile then leave the posts command unrun.
			s.press("u")
			typeQuery(s, "people")
			s.press("enter")
			cmd := s.key("enter")
			if cmd == nil {
				t.Fatal("opening the profile issued no command")
			}
			// Run only the header answer; leave posts outstanding if possible.
			msg := cmd()
			if msg != nil {
				next, nextCmd := s.m.update(msg)
				s.m = next
				_ = nextCmd // deliberately not run: loading posts
			}
			s.used = s.src.Calls()
		}},
		// The four profile cases 05-03's must-have names and the UI-SPEC's fixed frame list does not:
		// a bio that wraps to three lines in the header, a first post that is a repost below the
		// header, a post that quotes another, and a tombstone. They are one case per frame in the
		// corpus's own naming scheme (`<surface>_<case>_<width>`, the scheme `profile_scrolled_80` and
		// `search_top_80` already use) so that the audit can name, in its failure message, the frame
		// the case was lost from, and so that each case can be planted separately.
		{"profile_bio_80", 80, gEnvPlain, func(t *testing.T, s *shell) {
			header := caseProfile()
			header.Bio = threeLineBio
			s.src.WithProfile(header)
			s.src.WithProfilePages([]domain.PostPage{{Posts: nil, Next: ""}})
			openProfile("alice")(t, s)
		}},
		{"profile_repost_80", 80, gEnvPlain, func(t *testing.T, s *shell) {
			repost := casePost("case-repost", "someone else", "a post that was reposted")
			repost.RepostedBy = &domain.User{ID: "u-lzd", ScreenName: repostHandle, Name: "name of " + repostHandle}
			s.src.WithProfile(caseProfile())
			s.src.WithProfilePages([]domain.PostPage{{Posts: []domain.Post{repost}, Next: ""}})
			openProfile("alice")(t, s)
		}},
		{"profile_quote_80", 80, gEnvPlain, func(t *testing.T, s *shell) {
			quoted := casePost("case-quote-child", "lzd", quotedBody)
			quoted.CreatedAt = goldenNow.Add(-time.Hour)
			parent := casePost("case-quote-parent", "lzd", "a body with a quote under it")
			parent.Quote = &quoted
			s.src.WithProfile(caseProfile())
			s.src.WithProfilePages([]domain.PostPage{{Posts: []domain.Post{parent}, Next: ""}})
			openProfile("alice")(t, s)
		}},
		{"profile_tombstone_80", 80, gEnvPlain, func(t *testing.T, s *shell) {
			tombstone := domain.Post{ID: "case-tombstone", Visibility: domain.Unavailable}
			s.src.WithProfile(caseProfile())
			s.src.WithProfilePages([]domain.PostPage{{Posts: []domain.Post{tombstone}, Next: ""}})
			openProfile("alice")(t, s)
		}},
		{"prompt_80", 80, gEnvPlain, func(t *testing.T, s *shell) {
			s.press("/")
			typeQuery(s, "page cache")
		}},
		{"prompt_120", 120, gEnvPlain, func(t *testing.T, s *shell) {
			s.press("/")
			typeQuery(s, longQuery)
		}},
		{"search_stalled_80", 80, gEnvPlain, func(t *testing.T, s *shell) {
			s.src.StallWith("adversarial-search-stuck")
			openSearch("page cache")(t, s)
			s.press("G")
			s.press("pgdown")
			s.used = s.src.Calls()
		}},
		{"search_failure_80", 80, gEnvPlain, func(t *testing.T, s *shell) {
			openSearch("page cache")(t, s)
			s.src.FailNext(errRefused)
			s.press("G")
			s.press("pgdown")
			s.used = s.src.Calls()
		}},
		{"profile_scrolled_80", 80, gEnvPlain, func(t *testing.T, s *shell) {
			openProfile("alice")(t, s)
			for range 12 {
				s.press("j")
			}
			s.used = s.src.Calls()
		}},
	}

	// Colour rung matrix.
	for _, rung := range []struct {
		suffix string
		env    goldenEnv
	}{
		{"truecolor", gEnvTrueColor},
		{"256", gEnv256},
		{"16", gEnv16},
		{"mono", gEnvMono},
	} {
		rung := rung
		frames = append(frames,
			struct {
				name  string
				w     int
				env   goldenEnv
				drive func(*testing.T, *shell)
			}{"search_color_" + rung.suffix, 80, rung.env, openSearch(searchCJK)},
			struct {
				name  string
				w     int
				env   goldenEnv
				drive func(*testing.T, *shell)
			}{"profile_color_" + rung.suffix, 80, rung.env, openProfile("alice")},
		)
	}

	// Covered cross-product: search results, people and profile under every
	// CoverStyle at both audit widths. A missing covered frame fails TestGoldenFrames.
	for _, style := range render.CoverStyles() {
		style := style
		name := string(style)
		for _, w := range []int{80, 120} {
			w := w
			frames = append(frames,
				struct {
					name  string
					w     int
					env   goldenEnv
					drive func(*testing.T, *shell)
				}{"search_" + name + "_" + strconv.Itoa(w), w, gEnvPlain, func(t *testing.T, s *shell) {
					s.m.style = style
					s.m.state = StateImmersive
					openSearch(searchCJK)(t, s)
				}},
				struct {
					name  string
					w     int
					env   goldenEnv
					drive func(*testing.T, *shell)
				}{"search_people_" + name + "_" + strconv.Itoa(w), w, gEnvPlain, func(t *testing.T, s *shell) {
					s.m.style = style
					s.m.state = StateImmersive
					openPeople("people")(t, s)
				}},
				struct {
					name  string
					w     int
					env   goldenEnv
					drive func(*testing.T, *shell)
				}{"profile_" + name + "_" + strconv.Itoa(w), w, gEnvPlain, func(t *testing.T, s *shell) {
					s.m.style = style
					s.m.state = StateImmersive
					openProfile("alice")(t, s)
				}},
			)
		}
	}
	return frames
}

func TestGoldenFrames(t *testing.T) {
	for _, f := range goldenFrames(t) {
		f := f
		t.Run(f.name, func(t *testing.T) {
			f.env.pin(t)
			s := goldenShell(t, f.w, goldenH)
			f.drive(t, s)
			got := s.m.Render(f.w, goldenH)
			golden.RequireEqual(t, got)
		})
	}
}

// TestShowNewFrames prints the four review frames when run with -v.
func TestShowNewFrames(t *testing.T) {
	gEnvPlain.pin(t)
	for _, name := range []string{"search_80", "search_people_80", "profile_80", "prompt_80"} {
		var drive func(*testing.T, *shell)
		var w int
		for _, f := range goldenFrames(t) {
			if f.name == name {
				drive, w = f.drive, f.w
				break
			}
		}
		if drive == nil {
			t.Fatalf("no golden named %s", name)
		}
		s := goldenShell(t, w, goldenH)
		before := s.src.Calls()
		drive(t, s)
		frame := s.m.Render(w, goldenH)
		t.Logf("\n======== %s ========\n%s\n======== end %s ========\n", name, frame, name)
		_ = before
	}
}
