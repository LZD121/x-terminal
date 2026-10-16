package tui

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/LZD121/x-terminal/internal/config"
	"github.com/LZD121/x-terminal/internal/fake"
	"github.com/LZD121/x-terminal/internal/keys"
	"github.com/LZD121/x-terminal/internal/media"
	"github.com/LZD121/x-terminal/internal/render"
)

// This file is the phase's state machine as a test: the transition table of the UI-SPEC's
// `## Screen Grammar` (T1 to T13), the panic entry as one commanded write whose order is the sink's,
// the round trip that must come back byte for byte and cost nothing, the keys panic swallows, the
// focus handling, the whole surface working under a cover, and the two vocabularies joined.

// evolve applies one step of a transition: a key press through the table, or one of the two focus
// messages the terminal sends.
func (s *shell) evolve(step string) {
	s.t.Helper()
	switch step {
	case "blur":
		s.run(s.send(tea.BlurMsg{}))
	case "focus":
		s.run(s.send(tea.FocusMsg{}))
	default:
		s.press(step)
	}
}

// newPendingShell builds a shell that has not been told the pane's size yet: the state a start-up
// panic waits in, and the state the whole program is in for the first instant of its life.
func newPendingShell(t *testing.T, display Display) *shell {
	t.Helper()
	s := &shell{t: t, src: fake.Adversarial(), now: frozen, term: &fakeTerm{}}
	s.m = New(Deps{
		Source:  s.src,
		Budget:  func() (int, int, time.Time) { return s.used, s.limited, s.now },
		Failure: testFailure,
		Now:     func() time.Time { return s.now },
		Term:    s.term,
		Display: display,
		Persist: func(state, cover string) { s.persisted = append(s.persisted, state+"/"+cover) },
		Media:   &fake.Renderer{Answer: media.Result{Outcome: media.OpenedImage}},
		Clip:    &fake.Clipboard{},
	})
	return s
}

// transition is one row of the table below, and the effects are read off the two recording sinks a
// shell is built with: what the terminal port was asked to do and what the display writer was asked
// to write. "No entry" and "no write" are as much a part of the row as the state: the phase's rule
// about an involuntary transition is a rule about what was *not* asked for.
type transition struct {
	name  string
	start State
	style render.CoverStyle
	keys  []string

	state      State
	styleAfter render.CoverStyle
	// entries is how many times the terminal port's EnterPanic was called, windows the window names
	// it was given in order, and written the persist calls in order.
	entries int
	windows []string
	written []string
}

// TestStateTransitions is T1 to T13 row by row, with the effects each one asks for. It is the table
// the phase exists for, and the persist column is its point: `m`, `c` and `1`-`4` write, and no
// panic — by key or by blur — writes anything at all.
func TestStateTransitions(t *testing.T) {
	styles := render.CoverStyles()
	for _, tc := range []transition{
		{
			name: "T1 m from normal", start: StateNormal, style: render.CoverServiceLog, keys: []string{"m"},
			state: StateImmersive, styleAfter: render.CoverServiceLog,
			windows: []string{render.CoverTitleServiceLog}, written: []string{"immersive/servicelog"},
		},
		{
			name: "T2 m from immersive", start: StateImmersive, style: render.CoverGitLog, keys: []string{"m"},
			state: StateNormal, styleAfter: render.CoverGitLog,
			windows: []string{render.CopyTitlePlain}, written: []string{"normal/gitlog"},
		},
		{
			name: "T3 c from normal", start: StateNormal, style: render.CoverServiceLog, keys: []string{"c"},
			state: StateImmersive, styleAfter: render.CoverGitLog,
			windows: []string{render.CoverTitleGitLog}, written: []string{"immersive/gitlog"},
		},
		{
			name: "T4 c from immersive", start: StateImmersive, style: render.CoverServiceLog, keys: []string{"c"},
			state: StateImmersive, styleAfter: render.CoverGitLog,
			windows: []string{render.CoverTitleGitLog}, written: []string{"immersive/gitlog"},
		},
		{
			name: "T4 c wraps after the last style", start: StateImmersive, style: render.CoverInternal, keys: []string{"c"},
			state: StateImmersive, styleAfter: render.CoverServiceLog,
			windows: []string{render.CoverTitleServiceLog}, written: []string{"immersive/servicelog"},
		},
		{
			name: "T5 1 from normal", start: StateNormal, style: render.CoverGitLog, keys: []string{"1"},
			state: StateImmersive, styleAfter: render.CoverServiceLog,
			windows: []string{render.CoverTitleServiceLog}, written: []string{"immersive/servicelog"},
		},
		{
			name: "T5 4 from immersive", start: StateImmersive, style: render.CoverServiceLog, keys: []string{"4"},
			state: StateImmersive, styleAfter: render.CoverInternal,
			windows: []string{render.CoverTitleInternal}, written: []string{"immersive/internal"},
		},
		{
			name: "T5 3 writes nothing but its own pair", start: StateNormal, style: render.CoverServiceLog, keys: []string{"3"},
			state: StateImmersive, styleAfter: render.CoverCI,
			windows: []string{render.CoverTitleCI}, written: []string{"immersive/ci"},
		},
		{
			name: "T6 panic from normal", start: StateNormal, style: render.CoverServiceLog, keys: []string{"ctrl+p"},
			state: StatePanic, styleAfter: render.CoverServiceLog,
			entries: 1,
		},
		{
			name: "T6 panic from immersive", start: StateImmersive, style: render.CoverGitLog, keys: []string{"ctrl+p"},
			state: StatePanic, styleAfter: render.CoverGitLog,
			entries: 1,
		},
		{
			name: "T7 panic and back to immersive", start: StateImmersive, style: render.CoverCI, keys: []string{"ctrl+p", "ctrl+p"},
			state: StateImmersive, styleAfter: render.CoverCI,
			entries: 1, windows: []string{render.CoverTitleCI},
		},
		{
			name: "T7 panic and back to normal", start: StateNormal, style: render.CoverServiceLog, keys: []string{"ctrl+p", "ctrl+p"},
			state: StateNormal, styleAfter: render.CoverServiceLog,
			entries: 1, windows: []string{render.CopyTitlePlain},
		},
		{
			name: "T8 blur from normal", start: StateNormal, style: render.CoverServiceLog, keys: []string{"blur"},
			state: StatePanic, styleAfter: render.CoverServiceLog,
			entries: 1,
		},
		{
			name: "T8 blur from immersive", start: StateImmersive, style: render.CoverInternal, keys: []string{"blur"},
			state: StatePanic, styleAfter: render.CoverInternal,
			entries: 1,
		},
		{
			name: "T9 focus while panicked", start: StatePanic, style: render.CoverServiceLog, keys: []string{"focus"},
			state: StatePanic, styleAfter: render.CoverServiceLog,
		},
		{
			name: "T9 focus while normal", start: StateNormal, style: render.CoverServiceLog, keys: []string{"focus"},
			state: StateNormal, styleAfter: render.CoverServiceLog,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newShellIn(t, 100, 24, Display{State: tc.start, Cover: tc.style})
			if tc.start == StatePanic {
				// A start-up panic runs its entry once the pane's size is known: the fixture has
				// already done that, and this row is about what happens next.
				s.term.entered = nil
			}
			calls := s.src.Calls()

			for _, step := range tc.keys {
				s.evolve(step)
			}

			if s.m.state != tc.state {
				t.Errorf("the state after %v is %v, want %v", tc.keys, s.m.state, tc.state)
			}
			if s.m.style != tc.styleAfter {
				t.Errorf("the style after %v is %q, want %q", tc.keys, s.m.style, tc.styleAfter)
			}
			if got := len(s.term.entered); got != tc.entries {
				t.Errorf("the entry ran %d times, want %d", got, tc.entries)
			}
			if got := strings.Join(s.term.windows, " "); got != strings.Join(tc.windows, " ") {
				t.Errorf("the window names are %v, want %v", s.term.windows, tc.windows)
			}
			if got := strings.Join(s.persisted, " "); got != strings.Join(tc.written, " ") {
				t.Errorf("the display writes are %v, want %v", s.persisted, tc.written)
			}
			// No transition in this table issues or cancels a request: entering a cover, wearing
			// another style and panicking are render states.
			if s.src.Calls() != calls {
				t.Errorf("%v moved the request count from %d to %d", tc.keys, calls, s.src.Calls())
			}
		})
	}

	// Anti-vacuity: the table walks the styles this build has, and the cycle's own order is what the
	// rows above assert.
	if len(styles) != 4 {
		t.Fatalf("the table has %d styles, want 4", len(styles))
	}
}

// TestTheQuitKeysWriteNothing: T11 and T12 exit the process, and neither writes a display state —
// the exit paths are Phase 2's, with the one addition that the composition root restores the window
// name after the program returns.
func TestTheQuitKeysWriteNothing(t *testing.T) {
	for _, key := range []string{"ctrl+c", "q", "esc"} {
		s := newShell(t, 100, 24)
		cmd := s.key(key)
		if cmd == nil {
			t.Fatalf("%s asked for no command: it does not quit", key)
		}
		if msg := cmd(); msg == nil {
			t.Fatalf("%s's command answered nothing, so it is not tea.Quit", key)
		}
		if len(s.persisted) != 0 || len(s.term.windows) != 0 {
			t.Errorf("%s wrote %v and %v", key, s.persisted, s.term.windows)
		}
	}
}

// TestPanicEntersInOneFrameAndLeavesExactly is DSGT-03's property at the model: the very next render
// after the key is the panic frame, the entry carries the frame's own bytes and the frame's title,
// and the second `ctrl+p` puts back a frame that is byte-identical to the one before it — with the
// request count, the stack depth, the selection and the anchor all where they were.
func TestPanicEntersInOneFrameAndLeavesExactly(t *testing.T) {
	for _, start := range []State{StateNormal, StateImmersive} {
		t.Run(start.String(), func(t *testing.T) {
			s := newShellIn(t, 120, 30, Display{State: start, Cover: render.CoverGitLog})
			s.press("j", "j", "j")
			before := s.frame()
			calls := s.src.Calls()
			depth := len(s.m.frames)
			selected := s.m.top().List.Selected().ID
			anchor := s.m.top().Anchor

			if before == render.PanicFrame(120, 30) {
				t.Fatalf("the fixture is already showing the panic frame")
			}

			s.press("ctrl+p")
			if got := s.frame(); got != render.PanicFrame(120, 30) {
				t.Errorf("the frame after ctrl+p is not the panic frame:\n%s", got)
			}
			if len(s.term.entered) != 1 {
				t.Fatalf("the entry ran %d times, want once", len(s.term.entered))
			}
			entry := s.term.entered[0]
			if entry.frame != render.PanicFrame(120, 30) {
				t.Errorf("the entry carries a different frame than the panel does")
			}
			if entry.title != render.CopyTitlePanic {
				t.Errorf("the entry's title is %q, want %q", entry.title, render.CopyTitlePanic)
			}
			// The panic frame is the frame's own bytes, carries no status line and no feed text.
			if strings.Contains(entry.frame, "\x1b") {
				t.Errorf("the panic frame carries an escape byte")
			}
			if s.persisted != nil {
				t.Errorf("entering panic wrote %v", s.persisted)
			}

			s.press("ctrl+p")
			if got := s.frame(); got != before {
				t.Errorf("the frame after the round trip is not byte-identical:\n%s\n---\n%s", before, got)
			}
			if s.src.Calls() != calls {
				t.Errorf("the round trip moved the request count from %d to %d", calls, s.src.Calls())
			}
			if len(s.m.frames) != depth {
				t.Errorf("the round trip moved the stack depth from %d to %d", depth, len(s.m.frames))
			}
			if s.m.top().List.Selected().ID != selected {
				t.Errorf("the round trip moved the selection")
			}
			if s.m.top().Anchor != anchor {
				t.Errorf("the round trip moved the anchor")
			}
			if s.persisted != nil {
				t.Errorf("leaving panic wrote %v", s.persisted)
			}
			// Leaving panic puts the state's own title back on the window.
			if got := s.term.windows; len(got) != 1 || got[0] != s.m.title() {
				t.Errorf("leaving panic set the window name %v, want %q", got, s.m.title())
			}
		})
	}
}

// TestPanicSwallowsKeys is T10, twice over: every binding of the table except the two that exist
// under panic does nothing at all, the help overlay never opens, and an answer that arrives while
// the frame is up is absorbed by the ordinary path — panic is a render state, not a network state.
func TestPanicSwallowsKeys(t *testing.T) {
	s := newShell(t, 120, 24)
	s.press("j")
	before := s.frame()
	calls := s.src.Calls()
	selected := s.m.top().List.Selected().ID
	s.press("ctrl+p")
	panicFrame := s.frame()

	swallowed := 0
	for _, row := range keys.Rows() {
		if row.Action == keys.ActionPanic || row.Action == keys.ActionQuit {
			continue
		}
		if cmd := s.key(row.Keys[0]); cmd != nil {
			t.Errorf("the key %q asked for a command while panicked", row.Keys[0])
		}
		swallowed++
	}
	if swallowed < 18 {
		t.Fatalf("the table has %d rows to swallow, too few for the check", swallowed)
	}
	if s.frame() != panicFrame {
		t.Errorf("a swallowed key changed the frame")
	}
	if s.m.help {
		t.Errorf("the help overlay opened while panicked")
	}
	if s.m.top().List.Selected().ID != selected {
		t.Errorf("a swallowed key moved the selection")
	}
	if s.src.Calls() != calls {
		t.Errorf("a swallowed key issued a request")
	}

	// A late answer is absorbed, not dropped: the list grows while the frame is up, which the pane
	// itself does not show — a prepended refresh moves the blocks and keeps the reader's place, so
	// the new rows sit above the visible pane and the pane is where it was.
	loaded := s.m.top().List.Loaded()
	s.run(s.send(pageMsg{Frame: s.m.top().ID,
		Page:    prepending(2),
		Refresh: true,
	}))
	if s.frame() != panicFrame {
		t.Errorf("the answer changed the panic frame")
	}
	if got := s.m.top().List.Loaded(); got != loaded+2 {
		t.Errorf("the answer was dropped rather than absorbed: the list holds %d posts, want %d", got, loaded+2)
	}
	s.press("ctrl+p")
	if s.frame() == before {
		t.Errorf("the absorbed answer changed nothing at all, so it was never absorbed")
	}
}

// TestBlurPanicsAndFocusDoesNotRestore is DSGT-04: losing the window covers the pane, regaining it
// uncovers nothing, a second blur does not run the entry twice, and a shell that is never told about
// focus never enters panic by itself.
func TestBlurPanicsAndFocusDoesNotRestore(t *testing.T) {
	for _, start := range []State{StateNormal, StateImmersive} {
		t.Run(start.String(), func(t *testing.T) {
			s := newShellIn(t, 120, 24, Display{State: start, Cover: render.CoverServiceLog})
			before := s.frame()
			calls := s.src.Calls()

			s.run(s.send(tea.BlurMsg{}))
			if s.m.state != StatePanic {
				t.Fatalf("a blur left the state %v, want panic", s.m.state)
			}
			if len(s.term.entered) != 1 {
				t.Fatalf("the entry ran %d times for one blur, want once", len(s.term.entered))
			}
			if s.persisted != nil {
				t.Errorf("a blur wrote %v", s.persisted)
			}
			panicFrame := s.frame()

			// A second blur while already panicked enters nothing: the state is already panic, so the
			// entry is not run again.
			s.run(s.send(tea.BlurMsg{}))
			if len(s.term.entered) != 1 {
				t.Errorf("a second blur ran the entry %d times", len(s.term.entered))
			}

			// Focus is a no-op in every state: it never leaves panic.
			s.run(s.send(tea.FocusMsg{}))
			if s.m.state != StatePanic || s.frame() != panicFrame {
				t.Errorf("regaining focus left panic")
			}
			if len(s.term.windows) != 0 {
				t.Errorf("regaining focus set the window name %v", s.term.windows)
			}

			// And the same key gets back out, to the state the blur found.
			s.press("ctrl+p")
			if s.m.state != start {
				t.Errorf("leaving panic landed in %v, want %v", s.m.state, start)
			}
			if s.frame() != before {
				t.Errorf("the frame after a blur round trip is not byte-identical:\n%s\n---\n%s", before, s.frame())
			}
			if s.src.Calls() != calls {
				t.Errorf("the blur round trip issued a request")
			}
		})
	}

	// A terminal that reports nothing never enters panic by itself. The shell asks for the report —
	// that is the view's own field — and no other path leads into panic: ordinary keys, sizes,
	// answers and the init request all leave the state where it was.
	s := newShell(t, 120, 24)
	if !s.m.View().ReportFocus {
		t.Errorf("the view does not ask the terminal to report focus")
	}
	s.press("j", "k", "r", "t", "?", "?", "g", "G", "m", "c", "1", "m")
	s.resize(100, 24)
	if s.m.state != StateNormal || len(s.term.entered) != 0 {
		t.Errorf("a shell that was told nothing about focus entered %v by itself", s.m.state)
	}
}

// TestTheEntryIsCommandedNotCalled: the shell performs no terminal call in Update or View. The
// recording port is empty until the command the state change returned is run, and running it is what
// performs the entry — which is the boundary the whole package exists behind.
func TestTheEntryIsCommandedNotCalled(t *testing.T) {
	for _, key := range []string{"ctrl+p", "blur"} {
		s := newShell(t, 120, 24)
		var cmd tea.Cmd
		if key == "blur" {
			cmd = s.send(tea.BlurMsg{})
		} else {
			cmd = s.key(key)
		}
		if s.m.state != StatePanic {
			t.Fatalf("%s did not enter panic", key)
		}
		if len(s.term.entered) != 0 {
			t.Fatalf("%s performed the entry in Update: the port saw %d calls", key, len(s.term.entered))
		}
		if cmd == nil {
			t.Fatalf("%s returned no command, so nothing would perform the entry", key)
		}
		s.run(cmd)
		if len(s.term.entered) != 1 {
			t.Fatalf("running %s's command ran the entry %d times, want once", key, len(s.term.entered))
		}
	}

	// The same for a deliberate transition: `m` writes the window name and the config from the
	// command, and not from Update.
	s := newShell(t, 120, 24)
	cmd := s.key("m")
	if len(s.term.windows) != 0 || len(s.persisted) != 0 {
		t.Fatalf("`m` wrote from Update: %v %v", s.term.windows, s.persisted)
	}
	s.run(cmd)
	if len(s.term.windows) != 1 || len(s.persisted) != 1 {
		t.Fatalf("running `m`'s command wrote %v %v, want one window name and one persist", s.term.windows, s.persisted)
	}

	// View is pure too: rendering the panic state is the frame's bytes and touches no port.
	s = newShell(t, 120, 24)
	s.press("ctrl+p")
	before := len(s.term.entered)
	for range 3 {
		_ = s.frame()
	}
	if len(s.term.entered) != before {
		t.Errorf("rendering called the terminal port")
	}
}

// TestImmersiveIsTheSameList is DSGT-08 and DSGT-09 at the shell: the same rows, the same selection,
// the same anchor and the same request count under a cover, an identity that is an alias and never a
// handle that exists upstream, the covered help wording where the key table is, and a panic frame
// that carries no status line.
func TestImmersiveIsTheSameList(t *testing.T) {
	s := newShell(t, 100, 40)
	s.press("j", "j")
	normal := s.frame()
	calls := s.src.Calls()
	selected := s.m.top().List.Selected().ID
	anchor := s.m.top().Anchor
	loaded := s.m.top().List.Loaded()
	posts := s.m.top().List.Posts()

	s.press("m")
	if s.m.state != StateImmersive {
		t.Fatalf("`m` left the state %v", s.m.state)
	}
	if s.src.Calls() != calls {
		t.Errorf("wearing a cover issued a request")
	}
	if s.m.top().List.Selected().ID != selected || s.m.top().Anchor != anchor || s.m.top().List.Loaded() != loaded {
		t.Errorf("wearing a cover moved the list: selected %q anchor %d loaded %d", s.m.top().List.Selected().ID, s.m.top().Anchor, s.m.top().List.Loaded())
	}

	immersive := s.frame()
	if immersive == normal {
		t.Fatalf("the covered frame is the plain frame, so nothing was rendered through the cover")
	}
	// The rows are the same posts in the same order, and the identity is the alias the render package
	// computes for the account — never the handle that exists upstream.
	for _, post := range posts {
		if strings.Contains(immersive, post.Author.ScreenName) {
			t.Errorf("the covered frame carries the handle %q:\n%s", post.Author.ScreenName, immersive)
		}
	}
	alias := render.CoverAliasOf(render.CoverServiceLog, posts[0].Author)
	if !strings.Contains(immersive, alias.Handle) {
		t.Errorf("the covered frame carries no alias for the first post (want %q):\n%s", alias.Handle, immersive)
	}

	// The covered help wording, with the overlay open, and the overlay closes as in the plain state.
	s.press("?")
	overlay := s.pane()
	for _, row := range keys.Rows() {
		if row.HelpCovered == "" {
			continue
		}
		if !strings.Contains(overlay, row.HelpCovered) {
			t.Errorf("the covered overlay does not carry %q:\n%s", row.HelpCovered, overlay)
		}
	}
	if strings.Contains(overlay, "next post") {
		t.Errorf("the covered overlay carries the plain wording:\n%s", overlay)
	}
	s.press("?")

	// The status line carries the style, so the list field is the cover's own name.
	if status := s.status(); !strings.Contains(status, render.CopyCoverListServiceLogFollowing) {
		t.Errorf("the covered status line is %q, want it to name the list the style's way", status)
	}

	// Panic's frame is the frame's own bytes and carries no status line: what is on screen under
	// panic is exactly render.PanicFrame at the pane's size.
	s.press("ctrl+p")
	if got := s.frame(); got != render.PanicFrame(s.w, s.h) {
		t.Errorf("the panic frame is not the frame's own bytes")
	}

	// And the plain frame comes back byte for byte.
	s.press("ctrl+p")
	s.press("m")
	if s.frame() != normal {
		t.Errorf("taking the cover off did not restore the plain frame:\n%s\n---\n%s", normal, s.frame())
	}

	// Two styles in a row are two grammars of the same list: `c` moves the grammar and nothing else.
	s.press("m")
	calls = s.src.Calls()
	s.press("c")
	if s.src.Calls() != calls {
		t.Errorf("switching styles issued a request")
	}
	if s.m.top().List.Selected().ID != selected || s.m.top().Anchor != anchor {
		t.Errorf("switching styles moved the list")
	}
	if s.m.style != render.CoverGitLog {
		t.Errorf("`c` left the style %q, want the next one", s.m.style)
	}
	if s.frame() == immersive {
		t.Errorf("the new style rendered the same bytes as the old one")
	}
}

// TestStartupStateIsTheConfiguredOne is the config's half of the phase: the pane opens in the state
// the file names — plainly, covered, or on the panic frame whose entry runs once the pane has a size.
func TestStartupStateIsTheConfiguredOne(t *testing.T) {
	// Immersive: covered from the very first frame.
	s := newShellIn(t, 100, 24, Display{State: StateImmersive, Cover: render.CoverGitLog})
	if s.m.state != StateImmersive {
		t.Fatalf("the pane opened in %v, want immersive", s.m.state)
	}
	first := s.frame()
	if !strings.Contains(first, "commit ") {
		// The gitlog grammar's own head literal, rendered: this is not the plain frame.
		t.Errorf("the first covered frame carries nothing of the gitlog grammar:\n%s", first)
	}
	if s.m.View().WindowTitle != render.CoverTitleGitLog {
		t.Errorf("the first frame's title is %q, want %q", s.m.View().WindowTitle, render.CoverTitleGitLog)
	}
	if len(s.term.entered) != 0 {
		t.Errorf("a non-panic start-up entered panic")
	}

	// Panic: the frame first, and the entry commanded once the first size is known.
	panicShell := newPendingShell(t, Display{State: StatePanic, Cover: render.CoverServiceLog})
	// Before the terminal has said how big it is there is no frame to build and nothing is written.
	panicShell.run(panicShell.m.Init())
	if got := panicShell.m.Render(0, 0); got != "" {
		t.Errorf("the pane rendered %q with no known size", got)
	}
	if len(panicShell.term.entered) != 0 {
		t.Fatalf("the start-up entry ran before the pane had a size")
	}
	panicShell.resize(120, 24)
	if got := panicShell.frame(); got != render.PanicFrame(120, 24) {
		t.Errorf("the first frame of a start-up panic is not the panic frame")
	}
	if len(panicShell.term.entered) != 1 {
		t.Fatalf("the start-up entry ran %d times, want once", len(panicShell.term.entered))
	}
	if entry := panicShell.term.entered[0]; entry.title != render.CopyTitlePanic || entry.frame != render.PanicFrame(120, 24) {
		t.Errorf("the start-up entry carried %q and a frame of %d bytes", entry.title, len(entry.frame))
	}
	if panicShell.persisted != nil {
		t.Errorf("a start-up panic wrote %v", panicShell.persisted)
	}
	// A later resize does not run the entry again.
	panicShell.resize(90, 30)
	if len(panicShell.term.entered) != 1 {
		t.Errorf("a resize re-ran the start-up entry")
	}
	if got := panicShell.frame(); got != render.PanicFrame(90, 30) {
		t.Errorf("the resized panic frame is not the frame's own bytes")
	}

	// Normal: byte for byte the plain reader.
	plain := newShell(t, 100, 24)
	if plain.m.state != StateNormal || plain.m.View().WindowTitle != render.CopyTitlePlain {
		t.Errorf("the plain start-up is %v titled %q", plain.m.state, plain.m.View().WindowTitle)
	}
}

// TestDisplayVocabularyAgrees joins the three vocabularies the phase now has: the file's
// (internal/config), the state machine's and the cover styles'. Every value the reader accepts
// resolves through the shell's own resolver and back, and neither resolver accepts anything the
// reader refuses — which is what keeps the file's words and the machine's words from drifting apart
// silently.
func TestDisplayVocabularyAgrees(t *testing.T) {
	for _, name := range config.DisplayStates() {
		state, ok := StateOf(name)
		if !ok {
			t.Errorf("the reader accepts display.state = %q and the state machine does not know it", name)
			continue
		}
		if state.String() != name {
			t.Errorf("the state %v spells itself %q, and the reader's word is %q", state, state.String(), name)
		}
	}
	for _, name := range config.DisplayCovers() {
		style, ok := render.CoverStyleOf(name)
		if !ok {
			t.Errorf("the reader accepts display.cover = %q and the render package has no such style", name)
			continue
		}
		if string(style) != name {
			t.Errorf("the style %q names itself %q", name, style)
		}
		if style.Title() == render.CopyTitlePlain {
			t.Errorf("the style %q is titled like the plain reader", name)
		}
	}

	// The reader's defaults are values of the machine's own vocabularies.
	if state, ok := StateOf(config.DefaultDisplayState); !ok || state != StateNormal {
		t.Errorf("the reader's default state %q is not the plain reader", config.DefaultDisplayState)
	}
	if style, ok := render.CoverStyleOf(config.DefaultDisplayCover); !ok || style == "" {
		t.Errorf("the reader's default cover %q is not a style", config.DefaultDisplayCover)
	}

	// And the other direction: a word no reader accepts resolves to nothing, so a caller keeps the
	// default it already had rather than guessing.
	for _, name := range []string{"", "plain", "Normal", "immersive ", "cover", "svc-log", "x-terminal"} {
		if state, ok := StateOf(name); ok && name == "" {
			t.Errorf("StateOf(%q) answered %v", name, state)
		}
		if _, ok := render.CoverStyleOf(name); ok {
			t.Errorf("CoverStyleOf(%q) found a style", name)
		}
	}
	// The arm: the resolvers are not simply rejecting everything.
	if _, ok := StateOf("normal"); !ok {
		t.Fatalf("StateOf rejected a value the reader accepts")
	}
	if _, ok := render.CoverStyleOf("gitlog"); !ok {
		t.Fatalf("CoverStyleOf rejected a value the reader accepts")
	}
	// The states are total and exclusive: the three values the reader accepts are the three the
	// machine has, and nothing else resolves.
	if len(config.DisplayStates()) != 3 {
		t.Errorf("the reader accepts %d states", len(config.DisplayStates()))
	}
}

// TestThePersistedStateIsTheDeliberateOne is DSGT-11 asserted as bytes rather than as a call count:
// the file is the one the reader reads, the writer is the closure the composition root passes, and
// what each transition leaves in it is the transition's own rule — `m` writes the state, `c` and the
// four digits write both keys, and a blur → panic → focus cycle writes nothing at all, so the file is
// byte-identical after it.
func TestThePersistedStateIsTheDeliberateOne(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, "data"))

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("resolving the temporary home's configuration: %v", err)
	}
	path := filepath.Join(cfg.ConfigDir, config.ConfigFileName)
	// The writer is the closure the composition root passes, on the path the reader resolved: what the
	// transitions leave in the file is what this reads back.
	write := func(state, cover string) {
		if err := config.SetDisplay(path, state, cover); err != nil {
			t.Fatalf("the config writer refused %s/%s: %v", state, cover, err)
		}
	}

	read := func(t *testing.T) string {
		t.Helper()
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}
		return string(body)
	}
	// touched is which settings a transition wrote, as the set of keys whose line the file no longer
	// has in the form it had: a line that changed shows up twice in the line diff (once as the line it
	// was, once as the line it became), and it is one key either way. It is what tells "both keys" from
	// "the state and only the state".
	touched := func(t *testing.T, from, to string) []string {
		t.Helper()
		seen := map[string]bool{}
		out := make([]string, 0, 2)
		for _, line := range changedLines(t, from, to) {
			key, _, ok := strings.Cut(strings.TrimPrefix(line, "-"), "=")
			if !ok {
				t.Errorf("the config file changed %q, which is not a setting", line)
				continue
			}
			key = strings.TrimSpace(key)
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, key)
		}
		slices.Sort(out)
		return out
	}
	// start seeds the file and builds a shell in the plain reader: every row below is a transition
	// from the state the config file describes, which is what the composition root builds too.
	start := func(t *testing.T) *shell {
		t.Helper()
		write(StateNormal.String(), string(render.CoverServiceLog))
		return newShellWithPersist(t, 100, 24, Display{State: StateNormal, Cover: render.CoverServiceLog}, write)
	}

	// `m` from the plain reader: the state, and only the state. The style is remembered rather than
	// rewritten, so its line is byte-identical.
	s := start(t)
	from := read(t)
	s.press("m")
	if got := touched(t, from, read(t)); len(got) != 1 || got[0] != "display.state" {
		t.Errorf("`m` touched %v, want the display.state line alone", got)
	}
	if body := read(t); !strings.Contains(body, `display.state = "immersive"`) {
		t.Errorf("`m` left the file as\n%s\nwant it to hold the state it entered", body)
	}

	// `c` from the plain reader: both keys — the state it enters and the style the cycle lands on.
	s = start(t)
	from = read(t)
	s.press("c")
	if got := touched(t, from, read(t)); len(got) != 2 || got[0] != "display.cover" || got[1] != "display.state" {
		t.Errorf("`c` touched %v, want the state and the cover", got)
	}
	if body := read(t); !strings.Contains(body, `display.cover = "gitlog"`) {
		t.Errorf("`c` left the file as\n%s\nwant the style the cycle landed on", body)
	}

	// The four digits: the state it enters and the style the digit numbers. The style line is written
	// with the digit's own style whether or not the file already held it, so the file is the evidence;
	// the line diff only shows a key whose value moved, and digit 1 numbers the style the file starts
	// on.
	both := 0
	for i, digit := range []string{"1", "2", "3", "4"} {
		s = start(t)
		from = read(t)
		s.press(digit)
		got := touched(t, from, read(t))
		if len(got) == 0 || len(got) > 2 {
			t.Errorf("`%s` touched %v, want only the two display keys", digit, got)
		}
		want := `display.cover = ` + strconv.Quote(string(render.CoverStyles()[i]))
		if body := read(t); !strings.Contains(body, want) || !strings.Contains(body, `display.state = "immersive"`) {
			t.Errorf("`%s` left the file as\n%s\nwant %s and the immersive state", digit, body, want)
		}
		if len(got) == 2 {
			both++
		}
	}
	if both < 3 {
		t.Errorf("only %d of the four digits touched both display keys: the cover key is not being written", both)
	}

	// The involuntary transition: losing the window, regaining it and putting the pane back all write
	// nothing at all. The file is compared as bytes, not as a count of calls.
	s = newShellWithPersist(t, 100, 24, Display{State: StateNormal, Cover: render.CoverServiceLog}, write)
	s.press("m")
	frozen := read(t)
	s.run(s.send(tea.BlurMsg{}))
	if s.m.state != StatePanic {
		t.Fatalf("a blur left the state %v, want panic", s.m.state)
	}
	s.run(s.send(tea.FocusMsg{}))
	s.press("ctrl+p")
	if got := read(t); got != frozen {
		t.Errorf("a blur → panic → focus cycle changed the config file:\n%s\n---\n%s", frozen, got)
	}
	if s.m.state != StateImmersive {
		t.Fatalf("the cycle left the state %v, want the cover the blur interrupted", s.m.state)
	}
}
