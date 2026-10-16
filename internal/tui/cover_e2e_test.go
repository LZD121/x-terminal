package tui

// This file is the phase's model-level evidence: the panic round trip as captured facts, the entry's
// byte order and its dependent branches, the covered counterpart of every screen the corpus names,
// the title as a function of the state, and the promise that no alias table is written anywhere on
// disk.
//
// Two things about the shape are deliberate. The round trip is compared against bytes captured
// *before* the panic rather than against a second rendering computed the same way — a comparison of
// two fresh renders agrees with itself whether or not the state moved — and its facts are read off
// the model (the fake's own call count, the stack's depth, the latch the in-flight bookkeeping
// holds), so "nothing changed" is a measurement and not an intention. The disk leg points the config
// and data directories at a temporary home and runs the whole covered session through the same
// closure the composition root passes, because a second write path would be a second place an alias
// could land.

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/LZD121/x-terminal/internal/config"
	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/fake"
	"github.com/LZD121/x-terminal/internal/media"
	"github.com/LZD121/x-terminal/internal/render"
	"github.com/LZD121/x-terminal/internal/screen"
	"github.com/LZD121/x-terminal/internal/term"
	"github.com/LZD121/x-terminal/internal/xweb"
)

// roundTrip is the set of facts the phase's central claim is made of: the frame's own bytes, the
// source's call count, the frame stack's depth, the selected post, the anchor, the in-flight latch
// and the failure state on the status line. Every one of them is read from the model.
type roundTrip struct {
	state    State
	style    render.CoverStyle
	frame    string
	calls    int
	depth    int
	selected domain.PostID
	anchor   int
	pending  int
	failure  domain.Failure
}

// facts captures the round trip's facts from a shell as it stands.
func (s *shell) facts() roundTrip {
	top := s.m.top()
	return roundTrip{
		state:    s.m.state,
		style:    s.m.style,
		frame:    s.frame(),
		calls:    s.src.Calls(),
		depth:    len(s.m.frames),
		selected: domain.PostID(top.List.Selected().ID),
		anchor:   top.Anchor,
		pending:  s.m.outstanding(),
		failure:  s.m.failure,
	}
}

// TestPanicRoundTripIsExact is the phase's central claim: entering panic and leaving it are a render
// state and nothing else. From every non-panic start state the shell is walked to a non-trivial
// place — a selection that is not the first block, a scrolled anchor, more than one frame on the
// stack, a failure on the status line, a page in flight — and then every captured fact is required
// to be the one it was captured with, the frame byte for byte.
func TestPanicRoundTripIsExact(t *testing.T) {
	starts := []Display{
		{State: StateNormal, Cover: render.CoverServiceLog},
		{State: StateImmersive, Cover: render.CoverServiceLog},
		{State: StateImmersive, Cover: render.CoverGitLog},
		{State: StateImmersive, Cover: render.CoverCI},
		{State: StateImmersive, Cover: render.CoverInternal},
	}
	for _, start := range starts {
		t.Run(start.State.String()+"/"+string(start.Cover), func(t *testing.T) {
			s := newShellIn(t, 100, 30, start)

			// A non-trivial place: a selection that is not the first block, a half page (which moves
			// the anchor), a second frame on the stack from the feed toggle, a failure state on the
			// status line, and a page left in flight.
			s.press("j", "j", "j", "ctrl+d")
			s.press("t")
			s.src.FailNext(errRefused)
			s.press("r")
			s.key("G") // the command is deliberately not run: the page stays in flight

			before := s.facts()
			posts := s.m.top().List.Posts()
			if len(posts) == 0 {
				t.Fatalf("the shell holds no posts: the round trip below would be over an empty list")
			}
			if before.frame == render.PanicFrame(100, 30) {
				t.Fatalf("the shell is already showing the panic frame")
			}
			// The arms: the facts the round trip compares are the facts of a state that has something
			// to lose. A model with one frame, one block and nothing in flight could not see a stack,
			// an anchor or a latch bug at all.
			if before.depth < 2 {
				t.Fatalf("the walk left %d frames on the stack, want more than one", before.depth)
			}
			if before.selected == posts[0].ID {
				t.Fatalf("the walk left the selection on the first post %q: a moved selection is what the round trip is measured against", before.selected)
			}
			if before.pending == 0 {
				t.Fatalf("the walk left no request in flight: the latch's half of the round trip has nothing to hold")
			}
			if before.failure.Kind == domain.FailureNone {
				t.Fatalf("the walk left no failure on the status line: the state's half of the round trip has nothing to hold")
			}

			s.press("ctrl+p")
			if got := s.frame(); got != render.PanicFrame(100, 30) {
				t.Errorf("the frame after ctrl+p is not the panic frame:\n%s", got)
			}
			if len(s.term.entered) != 1 {
				t.Fatalf("the entry ran %d times, want once", len(s.term.entered))
			}
			if entry := s.term.entered[0]; entry.frame != render.PanicFrame(100, 30) || entry.title != render.CopyTitlePanic {
				t.Errorf("the entry carried %q and a frame of %d bytes, want the frame and %q", entry.title, len(entry.frame), render.CopyTitlePanic)
			}

			s.press("ctrl+p")
			assertSameFacts(t, before, s.facts())

			// A second leg: an answer that lands while the overlay is up is absorbed — the list behind
			// it gains the page — and nothing of it reaches the frame until the overlay is left,
			// which is what "panic swallows no event" means.
			s.press("ctrl+p")
			panicFrame := s.frame()
			loaded, calls := s.m.top().List.Loaded(), s.src.Calls()
			s.run(s.send(pageMsg{Frame: s.m.top().ID,
				Page:    prepending(2),
				Refresh: true,
			}))
			if got := s.frame(); got != panicFrame {
				t.Errorf("an answer changed the panic frame:\n%s", got)
			}
			if got := s.m.top().List.Loaded(); got != loaded+2 {
				t.Errorf("the answer was dropped rather than absorbed: the list holds %d posts, want %d", got, loaded+2)
			}
			if got := s.src.Calls(); got != calls {
				t.Errorf("absorbing an answer spent a request: the count moved from %d to %d", calls, got)
			}
			s.press("ctrl+p")
			if s.frame() == panicFrame {
				t.Errorf("the absorbed answer changed nothing after the overlay was left, so it was never absorbed")
			}
		})
	}
}

// assertSameFacts requires every captured fact to be the one it was captured with, naming the fact
// that moved rather than the whole struct: a round trip that reported "the model differs" would leave
// the reader to find out what an entry and an exit changed.
func assertSameFacts(t *testing.T, before, after roundTrip) {
	t.Helper()
	if after.frame != before.frame {
		t.Errorf("the frame after the round trip is not the frame before it byte for byte:\n%s\n---\n%s", before.frame, after.frame)
	}
	if after.calls != before.calls {
		t.Errorf("the round trip moved the source's call count from %d to %d", before.calls, after.calls)
	}
	if after.depth != before.depth {
		t.Errorf("the round trip moved the frame stack from %d frames to %d", before.depth, after.depth)
	}
	if after.selected != before.selected {
		t.Errorf("the round trip moved the selection from %q to %q", before.selected, after.selected)
	}
	if after.anchor != before.anchor {
		t.Errorf("the round trip moved the anchor from %d to %d", before.anchor, after.anchor)
	}
	if after.pending != before.pending {
		t.Errorf("the round trip moved the in-flight latch from %d to %d", before.pending, after.pending)
	}
	if after.failure != before.failure {
		t.Errorf("the round trip moved the failure state from %+v to %+v", before.failure, after.failure)
	}
	if after.state != before.state {
		t.Errorf("the round trip left the state %v, want the %v it was entered from", after.state, before.state)
	}
	if after.style != before.style {
		t.Errorf("the round trip moved the cover style from %q to %q", before.style, after.style)
	}
}

// recordingWriter is the sink's output: every write, in order and whole.
type recordingWriter struct{ writes []string }

func (w *recordingWriter) Write(p []byte) (int, error) {
	w.writes = append(w.writes, string(p))
	return len(p), nil
}

// newShellWithSink builds a shell whose terminal port is the sink under test: the same wiring as
// newShellIn, with the one seam the entry's order is read through replaced. sized says whether the
// shell has been told the pane's size yet, which is the difference between an ordinary start and a
// start-up panic's first instant.
func newShellWithSink(t *testing.T, w, h int, display Display, sink Term, sized bool) *shell {
	t.Helper()
	s := &shell{t: t, src: fake.Adversarial(), now: frozen, w: w, h: h}
	s.m = New(Deps{
		Source:  s.src,
		Budget:  func() (int, int, time.Time) { return s.used, s.limited, s.now },
		Failure: testFailure,
		Now:     func() time.Time { return s.now },
		Term:    sink,
		Display: display,
		Persist: func(state, cover string) { s.persisted = append(s.persisted, state+"/"+cover) },
		Media:   &fake.Renderer{Answer: media.Result{Outcome: media.OpenedImage}},
		Clip:    &fake.Clipboard{},
	})
	if sized {
		s.send(tea.WindowSizeMsg{Width: w, Height: h})
		s.run(s.m.Init())
	}
	return s
}

// TestPanicEntryOrderAndBranches is V6: the entry's own byte order, and the branches the pane
// variable's three cases take.
//
// The order is asserted on the bytes the sink was handed — every frame row, then ED 3, then the title
// — and the runner is a closure that reads the writer's own state at the moment it is called, which
// is what "the subprocess is dispatched after the write" means as a check rather than as a comment.
func TestPanicEntryOrderAndBranches(t *testing.T) {
	for _, branch := range []struct {
		name string
		pane string
	}{
		{"the pane variable is unset", ""},
		{"the pane variable is empty", ""},
		{"the pane variable names a pane", "%7"},
	} {
		t.Run(branch.name, func(t *testing.T) {
			out := &recordingWriter{}
			var ran [][]string
			sink := term.Sink{
				Out: out,
				Env: func(key string) string {
					if key == "TMUX_PANE" {
						return branch.pane
					}
					return ""
				},
				Run: func(name string, args ...string) ([]byte, error) {
					if len(out.writes) != 1 {
						t.Errorf("a tmux command ran after %d writes, want the one write of the frame's entry: the subprocess cannot precede the paint", len(out.writes))
					}
					ran = append(ran, append([]string{name}, args...))
					return nil, nil
				},
			}

			s := newShellWithSink(t, 80, 24, Display{State: StateNormal, Cover: render.CoverServiceLog}, sink, true)
			s.press("ctrl+p")

			// One write, and the three parts of it in the required order.
			if len(out.writes) != 1 {
				t.Fatalf("the entry wrote %d times, want once: the frame, ED 3 and the title are one write", len(out.writes))
			}
			entry := out.writes[0]
			rows := strings.Split(render.PanicFrame(80, 24), "\n")
			atFirst, atLast := strings.Index(entry, rows[0]), strings.Index(entry, rows[len(rows)-1])
			atErase := strings.Index(entry, "\x1b[3J")
			atTitle := strings.Index(entry, "\x1b]2;"+render.CopyTitlePanic)
			switch {
			case atFirst < 0 || atLast < 0:
				t.Fatalf("the write does not carry the frame's own rows:\n%q", entry)
			case atErase < 0:
				t.Fatalf("the write does not carry ED 3:\n%q", entry)
			case atTitle < 0:
				t.Fatalf("the write does not carry the title:\n%q", entry)
			case !(atFirst < atLast && atLast < atErase && atErase < atTitle):
				t.Errorf("the write places the parts at %d, %d, %d and %d: want every row, then ED 3, then the title", atFirst, atLast, atErase, atTitle)
			}
			if got := strings.Count(entry, "\x1b[3J"); got != 1 {
				t.Errorf("the write carries ED 3 %d times, want once", got)
			}

			// The multiplexer's own work: the clear and the rename, addressed to the pane the program
			// was started in, and nothing at all without one.
			if branch.pane == "" {
				if len(ran) != 0 {
					t.Errorf("the entry ran %v with no pane to address: the app never guesses a pane", ran)
				}
				return
			}
			want := [][]string{
				append([]string{"tmux"}, term.ClearHistoryArgs(branch.pane)...),
				append([]string{"tmux"}, term.RenameWindowArgs(branch.pane, render.CopyTitlePanic)...),
			}
			if len(ran) != len(want) {
				t.Fatalf("the entry ran %d commands, want the clear and the rename: %v", len(ran), ran)
			}
			for i := range want {
				if got := strings.Join(ran[i], " "); got != strings.Join(want[i], " ") {
					t.Errorf("command %d is %q, want %q", i+1, got, strings.Join(want[i], " "))
				}
			}
		})
	}

	// A start-up panic runs the same sequence after the first frame and not before it: the frame is
	// built from the pane's size, so there is nothing to write until the terminal has said how big
	// the pane is.
	t.Run("a start-up panic", func(t *testing.T) {
		out := &recordingWriter{}
		ran := 0
		sink := term.Sink{
			Out: out,
			Env: func(key string) string {
				if key == "TMUX_PANE" {
					return "%3"
				}
				return ""
			},
			Run: func(name string, args ...string) ([]byte, error) {
				ran++
				if len(out.writes) != 1 {
					t.Errorf("the start-up entry ran a tmux command after %d writes, want the one write", len(out.writes))
				}
				return nil, nil
			},
		}

		s := newShellWithSink(t, 0, 0, Display{State: StatePanic, Cover: render.CoverServiceLog}, sink, false)
		if len(out.writes) != 0 || ran != 0 {
			t.Fatalf("the start-up entry wrote or ran a command before the pane had a size: %q", out.writes)
		}

		s.resize(120, 30)
		if got := s.frame(); got != render.PanicFrame(120, 30) {
			t.Errorf("the first frame of a start-up panic is not the panic frame")
		}
		if len(out.writes) != 1 {
			t.Fatalf("the start-up entry wrote %d times, want once", len(out.writes))
		}
		entry := out.writes[0]
		rows := strings.Split(render.PanicFrame(120, 30), "\n")
		atFirst := strings.Index(entry, rows[0])
		atErase := strings.Index(entry, "\x1b[3J")
		atTitle := strings.Index(entry, "\x1b]2;"+render.CopyTitlePanic)
		if atFirst < 0 || atErase < 0 || atTitle < 0 || !(atFirst < atErase && atErase < atTitle) {
			t.Errorf("the start-up entry's parts are at %d, %d and %d: want every row, then ED 3, then the title", atFirst, atErase, atTitle)
		}
		if ran != 2 {
			t.Errorf("the start-up entry ran %d tmux commands, want the clear and the rename", ran)
		}

		// A later resize does not run the entry again: the state was ensured once.
		s.resize(100, 24)
		if len(out.writes) != 1 || ran != 2 {
			t.Errorf("a resize re-ran the start-up entry: %d writes and %d commands", len(out.writes), ran)
		}
	})
}

// screenState is one screen of Phase 2's surface, at the model's own level: what it is called, and
// how a fresh shell is walked to it. Every screen the corpus stores a frame for is here, because the
// claim under test is about the whole surface and not about one pane — "every screen that exists
// works under a cover with unchanged keys".
type screenState struct {
	name string
	// modal is the help overlay: while it is open the shell handles only help and back, so the legs
	// that press a key to change the display cannot run on it. That is Phase 2's rule and not this
	// phase's departure from it.
	modal bool
	reach func(t *testing.T, s *shell)
}

// screens is Phase 2's surface: the timeline, the empty read, the loading frame, the help overlay,
// the four named failure states and the stalled cursor.
func screens() []screenState {
	cold := func(f domain.Failure) func(t *testing.T, s *shell) {
		return func(t *testing.T, s *shell) {
			t.Helper()
			s.m = s.m.WithFailure(f)
		}
	}
	return []screenState{
		{
			name: "the timeline",
			reach: func(t *testing.T, s *shell) {
				t.Helper()
				s.press("j", "j")
			},
		},
		{
			name: "the empty read",
			reach: func(t *testing.T, s *shell) {
				t.Helper()
				s.run(s.send(pageMsg{Frame: s.m.top().ID, Page: domain.PostPage{FetchedAt: frozen}}))
			},
		},
		{
			name: "the loading frame",
			reach: func(t *testing.T, s *shell) {
				t.Helper()
				// The size is known and page one is out; no answer has arrived. A fresh model is the
				// only way to that state: the shell's first request is answered as soon as it is run.
				s.m = New(Deps{
					Source:  s.src,
					Budget:  func() (int, int, time.Time) { return s.used, s.limited, s.now },
					Failure: testFailure,
					Now:     func() time.Time { return s.now },
					Term:    s.term,
					Display: Display{State: s.m.state, Cover: s.m.style},
					Persist: func(state, cover string) { s.persisted = append(s.persisted, state+"/"+cover) },
					Media:   &fake.Renderer{Answer: media.Result{Outcome: media.OpenedImage}},
					Clip:    &fake.Clipboard{},
				})
				s.run(s.send(tea.WindowSizeMsg{Width: s.w, Height: s.h}))
			},
		},
		{
			name:  "the help overlay",
			modal: true,
			reach: func(t *testing.T, s *shell) {
				t.Helper()
				s.press("?")
			},
		},
		{
			name:  "session expired",
			reach: cold(domain.Failure{Kind: domain.FailureSessionExpired, Detail: xweb.OutcomeAuthRejected.Remedy(), Retry: true}),
		},
		{
			name: "rate limited",
			reach: cold(domain.Failure{
				Kind:   domain.FailureRateLimited,
				Reset:  frozen.Add(5*time.Hour + 20*time.Minute),
				Detail: "wait until the reset time from x-rate-limit-reset",
			}),
		},
		{
			name: "no route to upstream",
			reach: cold(domain.Failure{
				Kind:   domain.FailureNoRoute,
				Detail: "GET " + xweb.BaseURL + "/ failed: dial tcp: lookup x.com: no such host",
				Retry:  true,
			}),
		},
		{
			name:  "operation id stale",
			reach: cold(domain.Failure{Kind: domain.FailureOperationStale, Detail: xweb.OutcomeUnknownOperation.Remedy(), Retry: true}),
		},
		{
			name: "the stalled cursor",
			reach: func(t *testing.T, s *shell) {
				t.Helper()
				// The walk ends because the cursor came back: the list's own pagination trigger,
				// answered with the cursor it was asked with.
				list, eff := s.m.top().List.Update(screen.PageRequested{}, frozen)
				if eff.Kind != screen.EffectPage {
					t.Fatalf("the list asked for %v, want a page: the stalled state is not reachable this way", eff.Kind)
				}
				top := s.m.top()
				top.List = list
				s.m = s.m.setTop(top)
				s.src.StallWith(eff.Cursor)
				page, err := s.src.Home(context.Background(), top.List.Kind, eff.Cursor)
				if err != nil {
					t.Fatalf("the fake serves no page at %q: %v", eff.Cursor, err)
				}
				s.run(s.send(pageMsg{Frame: top.ID, Page: page, Cursor: eff.Cursor}))
				if !s.m.top().List.Stalled() {
					t.Fatalf("the list did not notice the cursor coming back: the state below is not the stalled one")
				}
			},
		},
	}
}

// TestTheWholeSurfaceUnderACover is DSGT-08: every screen this project has works under a cover with
// the same keys. For each screen, the covered run and the plain run are built from the same wiring
// and walked the same way; then the same key sequence is pressed against both, the two request counts
// must be equal, and taking the cover off must leave a frame byte-identical to the plain run's.
//
// The covered state is the same model as the plain one: the two shells must hold the same list, the
// same selection, the same anchor, the same failure and the same budget — the cover's difference is
// the grammar and the cover field, and nothing else.
func TestTheWholeSurfaceUnderACover(t *testing.T) {
	keys := []string{"j", "ctrl+d", "r", "t", "?", "?", "esc"}
	for _, screen := range screens() {
		t.Run(screen.name, func(t *testing.T) {
			plain := newShellIn(t, 100, 30, Display{State: StateNormal, Cover: render.CoverGitLog})
			covered := newShellIn(t, 100, 30, Display{State: StateImmersive, Cover: render.CoverGitLog})
			screen.reach(t, plain)
			screen.reach(t, covered)

			if plain.frame() == covered.frame() {
				t.Errorf("the covered frame is the plain frame byte for byte: the screen was not rendered through the cover")
			}
			assertSameState(t, plain, covered)

			plain.press(keys...)
			covered.press(keys...)
			if plain.src.Calls() != covered.src.Calls() {
				t.Errorf("the same keys cost %d requests under the cover and %d plainly: a cover changes no request", covered.src.Calls(), plain.src.Calls())
			}
			covered.press("m")
			if covered.m.state != StateNormal {
				t.Fatalf("`m` left the state %v: the frame below is not the plain reader's", covered.m.state)
			}
			// The two double-toggles come back to the frame they left, byte for byte, and neither
			// moves the display state.
			for _, pair := range [][]string{{"t", "t"}, {"ctrl+p", "ctrl+p"}} {
				before, state := covered.frame(), covered.m.state
				covered.press(pair...)
				if got := covered.frame(); got != before {
					t.Errorf("%v did not come back to the frame it left:\n%s\n---\n%s", pair, before, got)
				}
				if covered.m.state != state {
					t.Errorf("%v left the state %v, want the %v it started in", pair, covered.m.state, state)
				}
			}
			// The same model's own counterpart, where the state is not modal: wearing the cover and
			// taking it off again returns to the bytes it started from.
			if screen.modal {
				return
			}
			before := covered.frame()
			covered.press("m")
			if got := covered.frame(); got == before {
				t.Errorf("taking the cover off rendered the same bytes: the grammar did not change")
			}
			covered.press("m")
			if got := covered.frame(); got != before {
				t.Errorf("wearing the cover again did not return to the same bytes:\n%s\n---\n%s", before, got)
			}
		})
	}
}

// TestTheCoveredFailurePaneNamesNothingOfTheUpstream is DSGT-09 on the shipped path: the pane prints
// the classifier's own sentence whenever it has nothing else to show, and that sentence is the one
// string of this program a cover does not author.
//
// The state below is the verifier's own reproduction of the leak — immersive/gitlog at 80x24, the
// pane reading `upstream blocked` over `GET https://x.com/ failed: Get "https://x.com/": Bad Gateway`
// — driven through the shell's own Update and Render rather than through the render package, because
// the leak was at this call site and not in the copy table.
//
// The plain reader is the arm: it carries the sentence, host and all, so the covered absence is a
// change of the pane's words and not a pane that says nothing. Every cover is checked, because the
// rule is one call site and not one style's.
func TestTheCoveredFailurePaneNamesNothingOfTheUpstream(t *testing.T) {
	detail := `GET ` + xweb.BaseURL + `/ failed: Get "` + xweb.BaseURL + `/": Bad Gateway`
	failure := domain.Failure{Kind: domain.FailureUpstreamBlocked, Detail: detail, Retry: true}

	// A fresh model over the same wiring: this shell's list is loaded by the time a test can read it,
	// and the pane prints the classifier's sentence only when it has nothing else to show. A fresh
	// model with the failure set is the state the verifier reached with the shipped binary — the one
	// a user hits when a read fails and nothing is loaded yet.
	plain := newShellIn(t, 80, 24, Display{State: StateNormal})
	if got := paneOf(New(plain.m.deps).WithFailure(failure).Render(80, 24)); !strings.Contains(got, detail) || !strings.Contains(got, render.CopyUpstreamHost) {
		t.Fatalf("the plain pane does not carry the classifier's sentence %q with the host in it:\n%s", detail, got)
	}

	for _, style := range render.CoverStyles() {
		t.Run(string(style), func(t *testing.T) {
			s := newShellIn(t, 80, 24, Display{State: StateImmersive, Cover: style})
			frame := New(s.m.deps).WithFailure(failure).Render(80, 24)
			pane, status := paneOf(frame), statusOf(frame)
			if strings.Contains(pane, render.CopyUpstreamHost) || strings.Contains(strings.ToLower(pane), "twitter") {
				t.Errorf("the covered pane prints a name of the upstream:\n%s", pane)
			}
			if strings.Contains(pane, detail) {
				t.Errorf("the covered pane prints the classifier's own sentence:\n%s", pane)
			}
			for _, noun := range []string{render.CopyTitlePlain, "timeline", "post", "following", "for you"} {
				if strings.Contains(strings.ToLower(pane), noun) {
					t.Errorf("the covered pane prints %q, the product's own word:\n%s", noun, pane)
				}
			}
			// The state still says which failure it was and what to do about it: the head row keeps
			// the classifier's word with the host substituted and the line keeps its remedy, so a
			// covered failure pane can never render as nothing.
			if want := render.HostSubstituted(render.StateWord(failure.Kind)); !strings.Contains(pane, want) {
				t.Errorf("the covered pane does not carry its state word %q:\n%s", want, pane)
			}
			if want := render.CopyCoverFailureBody; !strings.Contains(pane, want) {
				t.Errorf("the covered pane does not carry the cover's own line %q:\n%s", want, pane)
			}
			if want := render.Remedy(failure.Kind); !strings.Contains(status, want) {
				t.Errorf("the covered status line does not carry the remedy %q: %q", want, status)
			}
		})
	}
}

// frameKindCheck is one surface a frame can show, with how a fresh shell is walked to it and what
// the covered frame of that surface must carry. It is the table the shell's own frameKind
// enumeration is joined to (model.go's frameKinds): the test below requires exactly one row per
// declared kind, so a surface added to the shell without a covered check for it fails there, and a
// row for a kind the shell does not declare fails too.
//
// The table is per *kind* and not per state: the states of a surface are the corpus's business (the
// stored frames in internal/render and internal/tui), and what has to hold for every kind is that it
// is reachable, that the cover is really worn on it, that it names nothing of the product, and that
// its own status field is the cover's words.
type frameKindCheck struct {
	kind frameKind
	name string
	// reach walks a shell that is otherwise wired like every other shell to this kind, including
	// any seam the surface needs (a saved frame needs a mirror). It is run against the plain shell
	// and the covered one alike.
	reach func(t *testing.T, s *shell)
	// words is the covered surface's own status field, read through the same lookup the shell reads
	// it with, so the check cannot drift from the words the program prints.
	words func(t *testing.T, s *shell) []string
}

// The style the per-kind checks are made under: one style is enough for the rule they state, which
// is about which surfaces exist and whether a covered frame of one is the cover's rendering of it
// (`TestTheWholeSurfaceUnderACover` walks every screen under gitlog for the same reason).
var (
	kindCheckStyle   = render.CoverGitLog
	kindCheckGrammar = kindCheckStyle.Grammar()
)

// frameKindChecks is every frame kind, once: see frameKindCheck.
func frameKindChecks() []frameKindCheck {
	style, grammar := kindCheckStyle, kindCheckGrammar
	return []frameKindCheck{
		{
			kind: frameHome,
			name: "the home timeline",
			reach: func(t *testing.T, s *shell) {
				t.Helper()
				s.press("j", "j")
			},
			words: func(t *testing.T, s *shell) []string { return []string{render.ListName(style, s.m.top().List.Kind)} },
		},
		{
			kind: frameThread,
			name: "one conversation",
			reach: func(t *testing.T, s *shell) {
				t.Helper()
				s.press("enter")
			},
			words: func(t *testing.T, s *shell) []string { return []string{render.ThreadListName(style)} },
		},
		{
			kind: frameSearch,
			name: "one query's results",
			reach: func(t *testing.T, s *shell) {
				t.Helper()
				s.press("/")
				typeQuery(s, "page cache")
				s.press("enter")
			},
			words: func(t *testing.T, s *shell) []string {
				f := s.m.top()
				return []string{render.ListLabelSearch(grammar, f.Search.Product(), f.Search.Query())}
			},
		},
		{
			kind: frameProfile,
			name: "one account's profile",
			reach: func(t *testing.T, s *shell) {
				t.Helper()
				s.press("u")
				typeQuery(s, "people")
				s.press("enter")
				s.press("enter")
			},
			words: func(t *testing.T, s *shell) []string {
				f := s.m.top()
				header, ok := f.Profile.Header()
				if !ok {
					t.Fatalf("the covered profile frame's header has not arrived: the surface's own words are the header's alias")
				}
				return []string{render.CoverAliasOf(style, header.User).Handle}
			},
		},
		{
			kind: frameSaved,
			name: "one saved list",
			reach: func(t *testing.T, s *shell) {
				t.Helper()
				s.m.deps.Saved = fake.NewMirror()
				s.m.deps.SavedCapability = func(domain.SavedKind) domain.Capability { return domain.CapNone }
				s.m.deps.SavedError = func(error) domain.Capability { return domain.CapNone }
				s.press("b")
			},
			words: func(t *testing.T, s *shell) []string {
				kind, _ := s.m.top().Saved.Kind().Saved()
				return []string{render.ListSaved(style, kind)}
			},
		},
	}
}

// kindShell is a fresh shell over the corpus's own source with the display handed in: the plain
// reader for the empty style, the style's cover for any other.
func kindShell(t *testing.T, style render.CoverStyle) *shell {
	t.Helper()
	s := peopleShell(t, 100, 30)
	s.m.state = StateNormal
	s.m.style = ""
	if style != "" {
		s.m.state = StateImmersive
		s.m.style = style
	}
	return s
}

// TestEveryFrameKindIsCheckedUnderACover joins the shell's own enumeration of its surfaces to a
// covered check for each of them: for every kind, a fresh plain shell and a fresh covered shell are
// walked to that surface by the same keys, and the covered frame must be the cover's rendering of it
// — the same kind on the stack, a different frame from the plain one byte for byte, none of the
// product's words in it, and the surface's own status field in the style's words.
//
// The join is the reason this exists: `frameKinds` (model.go) claims a screen added to the shell
// without a covered check for it fails here, and before this table it was true of nothing — the
// enumerator was dead code whose comment named a check that did not exist.
func TestEveryFrameKindIsCheckedUnderACover(t *testing.T) {
	style := kindCheckStyle
	declared := frameKinds()
	counts := map[frameKind]int{}
	for _, c := range frameKindChecks() {
		counts[c.kind]++
	}
	for _, kind := range declared {
		if counts[kind] != 1 {
			t.Errorf("the shell declares the frame kind %d and the covered check table has %d rows for it: a surface added to the shell needs a covered check of its own", kind, counts[kind])
		}
	}
	if len(frameKindChecks()) != len(declared) {
		t.Errorf("the covered check table has %d rows and the shell declares %d frame kinds: a row for a kind the shell does not declare is a check over nothing", len(frameKindChecks()), len(declared))
	}

	for _, check := range frameKindChecks() {
		t.Run(check.name, func(t *testing.T) {
			// The source is the corpus's own (posts and people): the search and profile rows need
			// people answers, and the per-kind check is about the surfaces the shipped program
			// reaches, not about which fixture a shell happens to be built on.
			plain := kindShell(t, "")
			covered := kindShell(t, style)
			check.reach(t, plain)
			check.reach(t, covered)

			if got := plain.m.top().Kind; got != check.kind {
				t.Fatalf("the plain walk reached the frame kind %d, want %d: the two shells below would not be the same surface", got, check.kind)
			}
			if got := covered.m.top().Kind; got != check.kind {
				t.Fatalf("the covered walk reached the frame kind %d, want %d", got, check.kind)
			}
			if plain.frame() == covered.frame() {
				t.Errorf("the covered frame is the plain frame byte for byte: the surface was not rendered through the cover")
			}
			body := strings.ToLower(ansi.Strip(covered.frame()))
			for _, leak := range savedLeakWords() {
				if strings.Contains(body, strings.ToLower(leak)) {
					t.Errorf("the covered %s frame holds %q, a word the cover did not author", check.name, leak)
				}
			}
			for _, word := range check.words(t, covered) {
				if word == "" {
					t.Fatalf("the check for %s names no words to look for: the assertion below would pass on an empty value", check.name)
				}
				if !strings.Contains(ansi.Strip(covered.frame()), word) {
					t.Errorf("the covered %s frame does not carry %q, the surface's own field in the cover's words", check.name, word)
				}
				if plainWord := strings.Contains(ansi.Strip(plain.frame()), word); plainWord {
					t.Errorf("the plain %s frame carries %q too: the covered field is not a change of words", check.name, word)
				}
			}
		})
	}
}

// TestTheCoveredProfileNamesOneAccount is D5: a profile frame's status field must name one account one
// way, whichever of the frame's two requests has answered. The covered field is an alias, and the
// alias is computed from the account's id when it has one and from its handle when it has not — so a
// field built from the handle the frame was opened on (`account: pump-sign-57` for `alice`) became a
// different account's spelling the moment the header arrived with the id (`account: batch-restore-09`).
// A cover that renamed the account mid-read is a tell and a lie.
//
// The check is the shipped path: the frame is opened by the shipped key and its header request is
// deliberately left unrun, so the field is read in the state it is in while the answer is out. It must
// name no account at all — a cover has no true name to print before the header, and a second alias is
// worse than none — and once the header has answered it must carry the pane's own account's alias.
// The plain arm is the same walk on the plain reader, where the field is the account's own handle and
// naming it before the answer is true; that is what makes the covered silence a rule and not a field
// that never renders.
func TestTheCoveredProfileNamesOneAccount(t *testing.T) {
	openProfile := func(t *testing.T, s *shell) tea.Cmd {
		t.Helper()
		s.press("u")
		typeQuery(s, "people")
		s.press("enter")
		cmd := s.key("enter")
		if cmd == nil {
			t.Fatal("opening a profile issued no command: the header request is what the state below is about")
		}
		return cmd
	}

	// The arm: the plain reader names the account it was opened on, before any answer has arrived.
	plain := kindShell(t, "")
	plainCmd := openProfile(t, plain)
	if _, ok := plain.m.top().Profile.Header(); ok {
		t.Fatal("the plain profile's header had already arrived: the pre-answer state is not the one under test")
	}
	if want := render.ListLabelProfile(render.Plain, domain.User{ScreenName: plain.m.top().Profile.Handle()}); !strings.Contains(plain.status(), want) {
		t.Fatalf("the plain frame's status line %q does not carry %q: the covered silence below would be a field that never renders", plain.status(), want)
	}
	plain.run(plainCmd)
	if _, ok := plain.m.top().Profile.Header(); !ok {
		t.Fatal("the plain profile's header never arrived: the arm above is not the state the covered frame is compared in")
	}

	for _, style := range render.CoverStyles() {
		t.Run(string(style), func(t *testing.T) {
			s := kindShell(t, style)
			cmd := openProfile(t, s)
			if header, ok := s.m.top().Profile.Header(); ok {
				t.Fatalf("the covered profile's header had already arrived (id %q): the pre-answer state is not the one under test", header.User.ID)
			}
			if before := s.status(); strings.Contains(before, "account:") {
				t.Errorf("the covered profile frame names an account before its header has answered: %q", before)
			}
			s.run(cmd)
			header, ok := s.m.top().Profile.Header()
			if !ok {
				t.Fatal("the covered profile's header never arrived: the field after the answer is not the one under test")
			}
			want := render.ListLabelProfile(style.Grammar(), header.User)
			if after := s.status(); !strings.Contains(after, want) {
				t.Errorf("the covered profile frame's status line %q does not carry %q, the alias of the account the pane's own header row prints", after, want)
			}
			// The alias the frame prints is the header's account's: one account, one name, once there
			// is an account to name.
			// The field's own projection: the table style names an account by its login, the three
			// others by the machine name, and which one it is is the grammar's own rule.
			if got := s.m.grammar().Identity(header.User).Handle; !strings.Contains(s.status(), got) {
				t.Errorf("the covered status line %q does not carry the header's own alias %q", s.status(), got)
			}
		})
	}
}

// assertSameState requires the two shells to be the same model behind the frame: the same list, the
// same selection at the same place in it, the same sticky failure, the same notice and the same budget.
// It is what makes "the covered frame differs from its plain twin in nothing but the grammar and the
// cover field" a check rather than a reading of the two frames.
//
// The anchor is deliberately not compared between the two shells. It is a pane row, and each shell
// measures its pane in its own grammar's rows: the same place two moves down the list is pane row 6
// for the plain reader and row 12 under gitlog, because a git record is taller than a post. That the
// covered shell's anchor follows the covered rows is the fix of this round — a covered shell whose
// anchor came from the plain row count would scroll the cursor out of its own pane (see
// TestTheSelectedHeadStaysInsideThePaneUnderEveryCover). What has to agree is the reader's place in
// the list's own order, which is the block the selection is in and not the row it sits at.
func assertSameState(t *testing.T, plain, covered *shell) {
	t.Helper()
	a, b := plain.m.top().List, covered.m.top().List
	switch {
	case a.Kind != b.Kind:
		t.Errorf("the two shells hold different feeds: %v and %v", a.Kind, b.Kind)
	case a.Selected().ID != b.Selected().ID:
		t.Errorf("the selections differ: %q and %q", a.Selected(), b.Selected())
	case selectedBlock(plain) != selectedBlock(covered):
		t.Errorf("the selected block is %d of the plain list and %d of the covered one: the same selection at two places in the order", selectedBlock(plain), selectedBlock(covered))
	case a.Loaded() != b.Loaded():
		t.Errorf("the loaded counts differ: %d and %d", a.Loaded(), b.Loaded())
	case a.HasMore() != b.HasMore():
		t.Errorf("the walk's ends differ: %v and %v", a.HasMore(), b.HasMore())
	case a.Stalled() != b.Stalled():
		t.Errorf("the stalled flags differ: %v and %v", a.Stalled(), b.Stalled())
	case len(a.Posts()) != len(b.Posts()):
		t.Errorf("the post counts differ: %d and %d", len(a.Posts()), len(b.Posts()))
	case plain.m.failure != covered.m.failure:
		t.Errorf("the failures differ: %+v and %+v", plain.m.failure, covered.m.failure)
	case plain.m.notice != covered.m.notice || plain.m.noticeCount != covered.m.noticeCount:
		t.Errorf("the notices differ: %v/%d and %v/%d", plain.m.notice, plain.m.noticeCount, covered.m.notice, covered.m.noticeCount)
	case len(plain.m.frames) != len(covered.m.frames):
		t.Errorf("the stack depths differ: %d and %d", len(plain.m.frames), len(covered.m.frames))
	case plain.m.help != covered.m.help:
		t.Errorf("the overlay's state differs: %v and %v", plain.m.help, covered.m.help)
	}
	// The content itself: every post of the plain run is in the covered run, with the same words and
	// the same instant. A cover that re-ordered, dropped or reworded a post would show up here.
	missing := 0
	for _, post := range a.Posts() {
		found := false
		for _, want := range b.Posts() {
			if post.ID == want.ID {
				found = post.Text == want.Text && post.Author == want.Author && post.CreatedAt.Equal(want.CreatedAt)
				break
			}
		}
		if !found {
			missing++
		}
	}
	if missing != 0 {
		t.Errorf("%d posts of the plain run are not in the covered run, or differ from them: the cover changes the presentation and not the content", missing)
	}
}

// selectedBlock is the selected post's place in its frame's own block order, or -1 when the frame has
// no block for it. It is the grammar-independent half of the reader's place, and the half
// assertSameState compares; the other half is the row the head sits at, which is measured in the
// frame's own grammar and never across two of them.
//
// -1 is the same answer on both sides: an empty list (the loading frame, the empty read) has no block
// order to have a place in, and the two shells are then required to agree that there is no place —
// which is a comparison and not a skip, since a place on one side and none on the other is exactly
// the divergence this function exists to catch.
func selectedBlock(s *shell) int {
	f := s.m.top()
	return blockIndex(f, f.selected().Top().ID)
}

// TestTheTitleIsAFunctionOfTheState is V10: the title names what is on screen, in every state, and
// the transition that enters a state asks the window for the same name the view carries. The panic
// row is the one exception and the one the UI-SPEC names: the entry carries the title in its own
// write, so nothing else sets it.
func TestTheTitleIsAFunctionOfTheState(t *testing.T) {
	rows := []struct {
		name   string
		keys   []string
		state  State
		title  string
		window []string
		entry  int
	}{
		{"the plain reader", nil, StateNormal, render.CopyTitlePlain, nil, 0},
		{"servicelog", []string{"1"}, StateImmersive, render.CoverTitleServiceLog, []string{render.CoverTitleServiceLog}, 0},
		{"gitlog", []string{"2"}, StateImmersive, render.CoverTitleGitLog, []string{render.CoverTitleGitLog}, 0},
		{"ci", []string{"3"}, StateImmersive, render.CoverTitleCI, []string{render.CoverTitleCI}, 0},
		{"internal", []string{"4"}, StateImmersive, render.CoverTitleInternal, []string{render.CoverTitleInternal}, 0},
		{"panic", []string{"ctrl+p"}, StatePanic, render.CopyTitlePanic, nil, 1},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			s := newShell(t, 100, 24)
			s.press(row.keys...)
			if s.m.state != row.state {
				t.Fatalf("the state after %v is %v, want %v", row.keys, s.m.state, row.state)
			}
			if got := s.m.View().WindowTitle; got != row.title {
				t.Errorf("the view's title is %q, want %q", got, row.title)
			}
			if got := strings.Join(s.term.windows, " "); got != strings.Join(row.window, " ") {
				t.Errorf("the transition asked the window for %v, want %v", s.term.windows, row.window)
			}
			if got := len(s.term.entered); got != row.entry {
				t.Fatalf("the entry ran %d times, want %d", got, row.entry)
			}
			if row.entry > 0 {
				if got := s.term.entered[0].title; got != row.title {
					t.Errorf("the entry's title is %q, want the frame's own name %q", got, row.title)
				}
			}

			// Leaving what was entered puts the state's own title back: the title is a function of the
			// state in every state, not only on the way in. The plain reader is the state the others
			// are entered from, so there is nothing for it to leave.
			switch row.state {
			case StateNormal:
				return
			case StatePanic:
				s.press("ctrl+p")
			default:
				s.press("m")
			}
			if got := s.m.View().WindowTitle; got != render.CopyTitlePlain {
				t.Errorf("the title after leaving %s is %q, want %q", row.name, got, render.CopyTitlePlain)
			}
		})
	}

	// The arm: no covered title contains a product name or a product noun. A title is where a leak
	// would be hardest to paint over — it is the window's frame and not a cell of the pane.
	styles := render.CoverStyles()
	if len(styles) != 4 {
		t.Fatalf("this build has %d styles, want the four the phase ships", len(styles))
	}
	for _, style := range styles {
		title := style.Title()
		if title == "" {
			t.Fatalf("the style %q has no title", style)
		}
		for _, word := range []string{render.CopyTitlePlain, "x-terminal", "post", "timeline", "feed", "following"} {
			if strings.Contains(strings.ToLower(title), strings.ToLower(word)) {
				t.Errorf("the covered title %q contains %q: the title names what is on screen and never the product", title, word)
			}
		}
	}
}

// TestNoAliasTableOnDisk is the phase's privacy promise asserted where it matters: on disk. The
// config and data directories are pointed at a temporary home, the whole covered session runs through
// the closure the composition root passes, and nothing but the config file's two documented lines may
// move. Then every file in both trees is searched for the fixture's aliases *and* for its handles: a
// config file that recorded an alias-to-handle mapping would fail here, and so would a diagnostics
// dump.
func TestNoAliasTableOnDisk(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, "data"))

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("resolving the temporary home's configuration: %v", err)
	}
	configPath := filepath.Join(cfg.ConfigDir, config.ConfigFileName)
	relConfig := filepath.ToSlash(filepath.Join("config", "x-terminal", config.ConfigFileName))
	if err := config.SetDisplay(configPath, StateNormal.String(), string(render.CoverServiceLog)); err != nil {
		t.Fatalf("writing the config the run starts from: %v", err)
	}
	before := snapshotTree(t, home)
	if len(before) == 0 {
		t.Fatal("the temporary home holds no file to compare against: the diff below would be vacuous")
	}

	// The run: the four styles, a style switch, a panic round trip, and a restart of the model from
	// the same config file — driven through the closure internal/cli's composition root passes, so a
	// second write path cannot hide behind a test's own writer.
	persist := func(state, cover string) {
		if err := config.SetDisplay(configPath, state, cover); err != nil {
			t.Errorf("the config writer refused the display %s/%s: %v", state, cover, err)
		}
	}
	s := newShellWithPersist(t, 100, 24, Display{State: StateImmersive, Cover: render.CoverServiceLog}, persist)
	s.press("c")      // a style switch in place: gitlog
	s.press("3")      // ci
	s.press("4")      // internal
	s.press("m")      // and back to the plain reader
	s.press("1", "c") // servicelog, then the switch the restart reads back
	s.press("ctrl+p") // the panic round trip, which writes nothing at all
	s.press("ctrl+p") //
	restarted := newShellWithPersist(t, 100, 24, displayFrom(t, configPath), persist)
	restarted.press("m") // the restart's own deliberate write
	after := snapshotTree(t, home)

	// The diff: the config file and nothing else, and inside it only the two documented lines.
	created, modified := diffTrees(t, before, after)
	if len(created) != 0 {
		t.Errorf("the covered run created %v: the phase's own state is the two config keys", created)
	}
	if len(modified) != 1 || modified[0] != relConfig {
		t.Errorf("the covered run modified %v, want %s alone", modified, relConfig)
	}
	changed := changedLines(t, before[relConfig].body, after[relConfig].body)
	if len(changed) != 2 {
		t.Errorf("the config file changed %d lines, want the two documented display lines:\n%s", len(changed), strings.Join(changed, "\n"))
	}
	for _, line := range changed {
		key, _, ok := strings.Cut(strings.TrimPrefix(line, "-"), "=")
		if !ok || (!strings.HasPrefix(strings.TrimSpace(key), "display.state") && !strings.HasPrefix(strings.TrimSpace(key), "display.cover")) {
			t.Errorf("the config file changed %q, want one of the two display lines", line)
		}
	}
	current, err := config.Load()
	if err != nil {
		t.Errorf("re-reading the config the run wrote: %v", err)
	} else if current.DisplayState != StateNormal.String() || current.DisplayCover != string(render.CoverGitLog) {
		t.Errorf("the config the run wrote holds %s/%s, want %s/%s", current.DisplayState, current.DisplayCover, StateNormal, render.CoverGitLog)
	}

	// The negative proof: no file anywhere in either tree carries an alias or a handle. The strings
	// searched for are the fixture's own, read off the model that was on screen.
	secrets := diskSecrets(s)
	if len(secrets) < 8 {
		t.Fatalf("the fixture's vocabulary is %d strings: too few for the search below to mean anything", len(secrets))
	}
	for path, file := range after {
		for _, secret := range secrets {
			if strings.Contains(file.body, secret) {
				t.Errorf("%s holds %q, an identity of the content it was built from: no file the phase writes may carry one", path, secret)
			}
		}
	}

	// The arms: the run really did write, and the search really can see a string it is handed in a
	// file of its own.
	if len(modified) == 0 && len(created) == 0 {
		t.Error("the covered run wrote nothing at all: the diff above proves nothing about what it writes")
	}
	planted := filepath.Join(home, "config", "x-terminal", "aliases-planted.txt")
	if err := os.WriteFile(planted, []byte(secrets[0]+"\n"), 0o600); err != nil {
		t.Fatalf("planting a file for the search's own arm: %v", err)
	}
	if !strings.Contains(snapshotTree(t, home)["config/x-terminal/aliases-planted.txt"].body, secrets[0]) {
		t.Error("the search cannot see a string it was handed: an absence it reports means nothing")
	}
}

// newShellWithPersist is newShellIn with the display writer replaced by the one under test: the shell
// is otherwise the shell the program runs.
func newShellWithPersist(t *testing.T, w, h int, display Display, persist func(state, cover string)) *shell {
	t.Helper()
	s := &shell{t: t, src: fake.Adversarial(), now: frozen, w: w, h: h, term: &fakeTerm{}}
	s.m = New(Deps{
		Source:  s.src,
		Budget:  func() (int, int, time.Time) { return s.used, s.limited, s.now },
		Failure: testFailure,
		Now:     func() time.Time { return s.now },
		Term:    s.term,
		Display: display,
		Persist: persist,
		Media:   &fake.Renderer{Answer: media.Result{Outcome: media.OpenedImage}},
		Clip:    &fake.Clipboard{},
	})
	s.send(tea.WindowSizeMsg{Width: w, Height: h})
	s.run(s.m.Init())
	return s
}

// displayFrom resolves the display state and cover the config file holds, the way the composition
// root resolves them: an unknown value keeps the default.
func displayFrom(t *testing.T, path string) Display {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	display := Display{State: StateNormal, Cover: render.CoverServiceLog}
	for _, line := range strings.Split(string(body), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"`)
		switch key {
		case "display.state":
			if state, ok := StateOf(value); ok {
				display.State = state
			}
		case "display.cover":
			if style, ok := render.CoverStyleOf(value); ok {
				display.Cover = style
			}
		}
	}
	return display
}

// fileSnapshot is one file's whole state: its size and its bytes. Both are kept, so a file that only
// changed size is a difference — and the size is not part of the bytes, so a diff of two bodies is a
// diff of lines rather than of a size line.
type fileSnapshot struct {
	size int
	body string
}

// snapshotTree is a directory tree's whole state: every file's path relative to the root, with its
// size and its bytes.
func snapshotTree(t *testing.T, root string) map[string]fileSnapshot {
	t.Helper()
	out := map[string]fileSnapshot{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		out[filepath.ToSlash(rel)] = fileSnapshot{size: int(info.Size()), body: string(body)}
		return nil
	})
	if err != nil {
		t.Fatalf("reading the tree under %s: %v", root, err)
	}
	return out
}

// diffTrees reports the paths a second snapshot created and the paths it modified, and fails on a path
// that is gone: a run that deleted a file is a difference too. A file whose bytes are the same but
// whose size is not is reported as modified, because the size is part of what is compared.
func diffTrees(t *testing.T, before, after map[string]fileSnapshot) (created, modified []string) {
	t.Helper()
	for path, now := range after {
		was, ok := before[path]
		switch {
		case !ok:
			created = append(created, path)
		case was != now:
			modified = append(modified, path)
		}
	}
	for path := range before {
		if _, ok := after[path]; !ok {
			t.Errorf("the covered run removed %s", path)
		}
	}
	slices.Sort(created)
	slices.Sort(modified)
	return created, modified
}

// changedLines is the lines one file's body has that the other does not, in both directions: the lines
// that moved. A blank line is a line like any other here, and a line the second body lost is prefixed
// with `-` so that a deletion is reported as a change rather than disappearing from the list.
func changedLines(t *testing.T, before, after string) []string {
	t.Helper()
	have := map[string]int{}
	for _, line := range strings.Split(before, "\n") {
		have[line]++
	}
	changed := make([]string, 0, 4)
	for _, line := range strings.Split(after, "\n") {
		if have[line] > 0 {
			have[line]--
			continue
		}
		changed = append(changed, line)
	}
	for line, count := range have {
		for range count {
			changed = append(changed, "-"+line)
		}
	}
	return changed
}

// diskSecrets is every string that must not reach a file: the fixture's handles and display names, and
// every projection of every account's alias in every style. The strings are read off the model that
// was on screen, so a run over different content is searched for that content's identities.
func diskSecrets(s *shell) []string {
	seen := map[string]bool{}
	out := make([]string, 0, 16)
	add := func(value string) {
		if value == "" || seen[value] {
			return
		}
		seen[value] = true
		out = append(out, value)
	}
	accounts := map[domain.User]bool{}
	for _, frame := range s.m.frames {
		for _, post := range frame.List.Posts() {
			accounts[post.Author] = true
			if post.RepostedBy != nil {
				accounts[*post.RepostedBy] = true
			}
		}
	}
	for account := range accounts {
		add(account.ScreenName)
		add(account.Name)
		add(account.ID)
		for _, style := range render.CoverStyles() {
			alias := render.CoverAliasOf(style, account)
			add(alias.Handle)
			add(alias.Name)
			add(alias.Login)
			add(alias.Email)
		}
	}
	return out
}

// savedFrameStates is every state a saved frame can be in that is not a row of the store, with how
// a fresh shell is walked to it. It is the saved surface's own table, beside screens(): the saved
// frame is where the product's repair vocabulary lives — the mirror's path and file, the program
// that pins an operation, the account a cache was filled from and the two lists' product names —
// and every one of these states is reachable from the shipped binary's own keys.
type savedFrameState struct {
	name string
	// covered is what a covered frame of this state must carry: the style's own words for it, so a
	// frame that rendered nothing at all cannot pass the absence below.
	covered []string
	// plain is a string the plain reader's own frame of the same state carries, and which no
	// covered frame may hold. Empty for a state that leaks nothing plainly (the first page in
	// flight), where the arm is not applicable rather than passing by accident.
	plain string
	reach func(t *testing.T, s *shell)
}

// savedFrameStates is the saved frame's whole non-row surface: the three capability states, the two
// empty reads, the failure with an empty store, and the first page in flight.
func savedFrameStates() []savedFrameState {
	status := func(owner string, synced bool) func(t *testing.T, s *shell) {
		return func(t *testing.T, s *shell) {
			t.Helper()
			mirror := s.m.deps.Saved.(*fake.Mirror)
			at := time.Time{}
			if synced {
				at = frozen
			}
			mirror.StatusOf(domain.Bookmark, domain.MirrorStatus{Owner: owner, LastSuccess: at})
			s.press("b")
		}
	}
	capability := func(c domain.Capability, owner string) func(t *testing.T, s *shell) {
		return func(t *testing.T, s *shell) {
			t.Helper()
			s.m.deps.SavedCapability = func(domain.SavedKind) domain.Capability { return c }
			mirror := s.m.deps.Saved.(*fake.Mirror)
			if c == domain.CapMirrorUnreadable {
				// A store the frame says it cannot read is a store whose reads fail: the frame's own
				// read is what answers with the failure, and the mapping that turns a store failure
				// into the capability is what puts the state on it. A mirror that handed back a page
				// while the frame wore this state would be a pair the shipped path cannot produce —
				// and a read that delivered its page releases the state, which is the frame's own
				// release rule (screen.Saved).
				mirror.FailPage(domain.Bookmark, errors.New("the mirror could not be read"))
				s.m.deps.SavedError = func(error) domain.Capability { return domain.CapMirrorUnreadable }
			}
			mirror.StatusOf(domain.Bookmark, domain.MirrorStatus{Owner: owner, LastSuccess: frozen})
			s.press("b")
		}
	}
	return []savedFrameState{
		{
			name:    "the list's operation is missing",
			covered: []string{render.CopyCoverCapUnavailable, render.CopyCoverCapBodyMissing},
			plain:   "doctor --pin",
			reach:   capability(domain.CapOperationMissing, "mira"),
		},
		{
			name:    "the store was filled by another account",
			covered: []string{render.CopyCoverCapWrongAccount, render.CopyCoverCapBodyWrongAccount},
			plain:   "this mirror was synced from @mira",
			reach:   capability(domain.CapWrongAccount, "mira"),
		},
		{
			name:    "the store cannot be read",
			covered: []string{render.CopyCoverCapUnreadable, render.CopyCoverCapBodyUnreadable},
			plain:   "move saved.db aside",
			reach:   capability(domain.CapMirrorUnreadable, "mira"),
		},
		{
			name:    "the empty read that was never filled",
			covered: []string{render.CopyCoverEmptyHeading, render.CopyCoverEmptyNeverMirrored},
			plain:   "nothing is mirrored for bookmarks",
			reach:   status("mira", false),
		},
		{
			name:    "the empty read that succeeded",
			covered: []string{render.CopyCoverEmptyHeading, render.CopyCoverEmptyBody},
			plain:   "no posts in bookmarks",
			reach:   status("mira", true),
		},
		{
			// The likes list's own empty answer: ambiguous between empty and no longer served, and
			// left open by a one-line hedge under every cover (`l` is not a cover-breaking key). The
			// state is reached through the shipped key, and the plain arm — the plain reader's own
			// hedged line — is the string the covered words must not be.
			name:    "the likes list with an empty store",
			covered: []string{render.CopyCoverEmptyHedged, render.CopyCoverEmptyHeading},
			plain:   render.CopyEmptyLikesHedged,
			reach: func(t *testing.T, s *shell) {
				t.Helper()
				s.press("l")
			},
		},
		{
			name: "the failure with an empty store",
			covered: []string{
				render.HostSubstituted(render.StateWord(domain.FailureUpstreamBlocked)),
				render.CopyCoverFailureBody,
				render.CopyCoverEmptyNeverMirrored,
			},
			plain: "nothing is mirrored for bookmarks",
			reach: func(t *testing.T, s *shell) {
				t.Helper()
				status("mira", false)(t, s)
				s.m = s.m.WithFailure(blockedFailure())
			},
		},
		{
			name:    "the first page in flight",
			covered: []string{render.CopyCoverLoading},
			reach: func(t *testing.T, s *shell) {
				t.Helper()
				// The command the frame pushed is held rather than run: the state is the instant
				// after `b` and before the store has answered.
				if cmd := s.key("b"); cmd == nil {
					t.Fatal("`b` pushed no saved frame")
				}
			},
		},
	}
}

// blockedFailure is the verifier's own reproduction of the leak, as a value: the reachability
// check's sentence with the host in it, on the one failure kind whose covered state word is a
// rename of the classifier's.
func blockedFailure() domain.Failure {
	return domain.Failure{
		Kind:   domain.FailureUpstreamBlocked,
		Detail: `GET ` + xweb.BaseURL + `/ failed: Get "` + xweb.BaseURL + `/": Bad Gateway`,
		Retry:  true,
	}
}

// savedLeakWords is every string a covered saved frame may not carry: the upstream's host and the
// product's own name (which the mirror's path spells), the store's file, the word the plain reader
// has for it, the program that pins an operation, the pin file, the two lists' product names, and
// the handle the store's owner field holds. None of them is a word the cover authored.
func savedLeakWords() []string {
	return []string{
		render.CopyUpstreamHost, render.CopyTitlePlain, "mirror", "saved.db", "doctor", "pins.json",
		render.CopyListBookmarks, render.CopyListLikes, "mira",
	}
}

// TestTheCoveredSavedPaneNamesNothingOfTheProduct is B1 on the shipped path: the saved frame's
// non-row branches — the capability state, the empty read and the failure that has nothing to show
// — are the three places the shell handed the renderer the plain grammar and the plain reader's
// facts (`Plain`/`""`) instead of the jumpers the pane is drawn under. The result was a covered
// pane printing `/home/.../x-terminal/saved.db`, `mirror unreadable`, `no posts in bookmarks` and
// the classifier's sentence with `x.com` in it.
//
// The scan is the phase's own rule, extended to this frame kind: a cover supplies its own words for
// every state, so no covered frame may hold a plain literal, the product's name, the upstream's
// host, a path, a purpose or a handle. The plain reader is the arm — a frame of the same state
// carries the string — so the covered absence is a change of the pane's words and not a pane that
// says nothing, and every state asserts the cover's own words are on screen besides.
func TestTheCoveredSavedPaneNamesNothingOfTheProduct(t *testing.T) {
	for _, w := range []int{80, 120} {
		for _, state := range savedFrameStates() {
			t.Run(strings.ReplaceAll(state.name, " ", "_")+"/"+strconv.Itoa(w), func(t *testing.T) {
				// The arm first: the plain reader's frame of the same state carries the string the
				// covered scan looks for. A state that leaks nothing plainly says so instead of
				// passing silently.
				if state.plain != "" {
					plain := newSavedShell(t, "", w)
					state.reach(t, plain)
					if !strings.Contains(plain.frame(), state.plain) {
						t.Fatalf("the plain frame of %q does not carry %q: the covered absence below would be vacuous:\n%s",
							state.name, state.plain, plain.frame())
					}
				}
				for _, style := range render.CoverStyles() {
					t.Run(string(style), func(t *testing.T) {
						s := newSavedShell(t, style, w)
						state.reach(t, s)
						frame := s.frame()
						oneLine := collapsed(frame, style.Grammar().BodyPrefix)
						body := strings.ToLower(ansi.Strip(frame))
						for _, leak := range savedLeakWords() {
							if strings.Contains(body, strings.ToLower(leak)) {
								t.Errorf("the covered %s frame holds %q, a word the cover did not author (state %q):\n%s",
									style, leak, state.name, frame)
							}
						}
						// The cover's own words, looked for with the pane's wrapping taken out: a
						// sentence longer than the measure is a row per wrapped line, and the
						// sentence is what the frame has to carry.
						for _, want := range state.covered {
							if !strings.Contains(oneLine, collapsed(want, "")) {
								t.Errorf("the covered %s frame does not carry %q, the cover's own words for %q:\n%s",
									style, want, state.name, frame)
							}
						}
					})
				}
			})
		}
	}
}

// newSavedShell is a shell whose saved frame can be reached: a mirror that answers with no rows, a
// capability seam the caller can point, and the operator's own store path — the string two of the
// three capability bodies print and which is the reason this state has to be covered at all.
func newSavedShell(t *testing.T, style render.CoverStyle, w int) *shell {
	t.Helper()
	display := Display{State: StateNormal}
	if style != "" {
		display = Display{State: StateImmersive, Cover: style}
	}
	s := newShellIn(t, w, 24, display)
	s.m.deps.Saved = fake.NewMirror()
	s.m.deps.SavedCapability = func(domain.SavedKind) domain.Capability { return domain.CapNone }
	s.m.deps.SavedError = func(error) domain.Capability { return domain.CapNone }
	s.m.deps.MirrorPath = "/home/user/.local/share/x-terminal/saved.db"
	return s
}

// collapsed is a frame's text as one line: every row's padding, the style's own body gutter, its
// colour and its line break taken out, so a sentence the pane wrapped over several rows can be
// looked for as the sentence it is. The gutter is a cell of the row and not a word of the sentence,
// and it is the one thing a wrapped body line carries that the sentence does not.
func collapsed(s, gutter string) string {
	if gutter != "" {
		s = strings.ReplaceAll(ansi.Strip(s), gutter, " ")
	}
	return strings.Join(strings.Fields(ansi.Strip(s)), " ")
}
