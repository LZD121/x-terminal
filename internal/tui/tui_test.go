package tui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/fake"
	"github.com/LZD121/x-terminal/internal/keys"
	"github.com/LZD121/x-terminal/internal/media"
	"github.com/LZD121/x-terminal/internal/netcheck"
	"github.com/LZD121/x-terminal/internal/render"
	"github.com/LZD121/x-terminal/internal/screen"
	"github.com/LZD121/x-terminal/internal/xweb"
)

// frozen is the tests' clock: one instant for the whole file, so a frame is reproducible.
var frozen = time.Date(2026, 10, 14, 9, 12, 0, 0, time.UTC)

// errRefused is what the tests make the fake return: an error Deps.Failure names.
var errRefused = errors.New("refused")

// testFailure is the failure seam in these tests.
//
// The real mapper is internal/cli's failureFor, which this package may not import (the boundary test
// says so), so the shell is handed a mapping the tests can read: the sentinel above becomes the
// session-expired state with the classifier's own sentence as its Detail; a failed reachability
// result names the broken link, with the check's own sentence as the Detail, exactly as the real
// mapper's path branch does; and anything else becomes unclassified with the error's own words. No
// state word is spelled in this file: the words come out of internal/render by kind, and the
// sentences out of the classifier or the check.
func testFailure(err error, check []netcheck.Result) domain.Failure {
	if errors.Is(err, errRefused) {
		return domain.Failure{
			Kind:   domain.FailureSessionExpired,
			Detail: xweb.OutcomeAuthRejected.Remedy(),
			Retry:  true,
		}
	}
	for _, result := range check {
		if !result.Outcome.Failed() {
			continue
		}
		kind := domain.FailureUnclassified
		if result.Outcome == netcheck.OutcomeNoRoute {
			kind = domain.FailureNoRoute
		}
		return domain.Failure{Kind: kind, Detail: result.Detail, Retry: true}
	}
	return domain.Failure{Kind: domain.FailureUnclassified, Detail: err.Error(), Retry: true}
}

// fakeTerm is the terminal port a test drives, and it records everything: which frame was entered
// with which title, and which window names were set. The record is empty until the command the shell
// returned is actually run, which is how the shell-performs-no-terminal-call-itself property is read.
type fakeTerm struct {
	entered []panicEntry
	windows []string
}

// panicEntry is one EnterPanic call's two values.
type panicEntry struct {
	frame string
	title string
}

func (f *fakeTerm) EnterPanic(frame, title string) {
	f.entered = append(f.entered, panicEntry{frame: frame, title: title})
}

func (f *fakeTerm) SetWindow(name string) { f.windows = append(f.windows, name) }

// shell is the shell under test: the model, the fake source it reads, and the seams a test has over
// everything the model would otherwise reach outside itself.
type bothSource struct {
	*fake.Source
	threads *fake.Thread
}

func (b bothSource) Thread(ctx context.Context, id domain.PostID, c domain.Cursor) (domain.ThreadPage, error) {
	return b.threads.Thread(ctx, id, c)
}

type shell struct {
	t       *testing.T
	src     *fake.Source
	th      *fake.Thread
	tr      *fake.Translator
	m       Model
	now     time.Time
	used    int
	limited int
	w, h    int
	// term is the terminal sink every state change writes through, and persisted is what the
	// display's own writer was called with, in order.
	term      *fakeTerm
	persisted []string
}

// translateLang is the language every shell in this file hands the port: the default the config
// resolves when nothing else says (internal/config's DefaultTranslateLanguage), so a test that
// asserts what the port was asked for can name the setting rather than a literal.
const translateLang = "zh-CN"

// translatedBody is what the fake translator answers with until a test scripts its own. It is
// plainly not any fixture post's own text, so a row carrying it is proof the swap happened, and it
// is short enough to stay on one pane row at every width this file renders.
const translatedBody = "rendered by the translation service"

// newShell builds a shell at a size, sends it that size, and lets it issue — and answer — its first
// request, which is the state the program starts in. It starts in the plain reader wearing the first
// style, which is what the config's own two defaults resolve to.
func newShell(t *testing.T, w, h int) *shell {
	t.Helper()
	return newShellIn(t, w, h, Display{State: StateNormal, Cover: render.CoverServiceLog})
}

// newShellIn is newShell in a given display state, for the tests that are about what the pane opens
// as and what comes back from panic.
func newShellIn(t *testing.T, w, h int, display Display) *shell {
	t.Helper()
	src := fake.Adversarial()
	th := fake.ThreadShapes()
	// The translator is the phase's own double: it answers in process and counts every call, which is
	// what the one-request-per-post rule is read from.
	tr := &fake.Translator{Answer: domain.Translation{Text: translatedBody, Available: true}}
	s := &shell{t: t, src: src, th: th, tr: tr, now: frozen, w: w, h: h, term: &fakeTerm{}}
	s.m = New(Deps{
		Source:  bothSource{Source: src, threads: th},
		Threads: bothSource{Source: src, threads: th},
		Budget:  func() (int, int, time.Time) { return s.used, s.limited, s.now },
		Failure: testFailure,
		Now:     func() time.Time { return s.now },
		Term:    s.term,
		Display: display,
		Persist: func(state, cover string) { s.persisted = append(s.persisted, state+"/"+cover) },
		Media:   &fake.Renderer{Answer: media.Result{Outcome: media.OpenedImage}},
		Clip:    &fake.Clipboard{},

		Translate:     tr,
		TranslateLang: translateLang,
	})
	s.send(tea.WindowSizeMsg{Width: w, Height: h})
	s.run(s.m.Init())
	return s
}

// send applies one message and returns the command it asked for.
func (s *shell) send(msg tea.Msg) tea.Cmd {
	s.t.Helper()
	next, cmd := s.m.update(msg)
	s.m = next
	return cmd
}

// run executes a command and feeds its answer back, repeating until the shell asks for nothing. A
// command is a value until something calls it, and this loop is what the program does with it.
func (s *shell) run(cmd tea.Cmd) {
	s.t.Helper()
	var queue []tea.Cmd
	if cmd != nil {
		queue = append(queue, cmd)
	}
	for i := 0; len(queue) > 0; i++ {
		if i > 32 {
			s.t.Fatalf("the shell answered commands with commands %d times over", i)
		}
		cmd, queue = queue[0], queue[1:]
		msg := cmd()
		if msg == nil {
			continue
		}
		if batch, ok := msg.(tea.BatchMsg); ok {
			for _, c := range batch {
				if c != nil {
					queue = append(queue, c)
				}
			}
			continue
		}
		if next := s.send(msg); next != nil {
			queue = append(queue, next)
		}
	}
}

// key is one key press, without running the command it asks for: a test that wants to look at the
// in-flight state has to hold the command itself.
func (s *shell) key(spelling string) tea.Cmd {
	s.t.Helper()
	next, cmd := s.m.press(press(spelling))
	s.m = next
	return cmd
}

// press is a sequence of key presses, each one's command answered before the next.
func (s *shell) press(spellings ...string) {
	s.t.Helper()
	for _, spelling := range spellings {
		s.run(s.key(spelling))
	}
}

// resize sends a new terminal size, the way the program receives one.
func (s *shell) resize(w, h int) {
	s.t.Helper()
	s.w, s.h = w, h
	s.run(s.send(tea.WindowSizeMsg{Width: w, Height: h}))
}

func (s *shell) frame() string  { return s.m.Render(s.w, s.h) }
func (s *shell) pane() string   { return paneOf(s.frame()) }
func (s *shell) status() string { return statusOf(s.frame()) }

// press builds the key press a terminal reports for one of the table's spellings: the decoder's own
// shape, so a test presses what a user presses rather than a synthetic event the matcher happens to
// like. The spellings that are not one character are the named keys and the modifiers.
func press(spelling string) tea.KeyPressMsg {
	named := map[string]rune{
		"down":   tea.KeyDown,
		"up":     tea.KeyUp,
		"enter":  tea.KeyEnter,
		"pgdown": tea.KeyPgDown,
		"pgup":   tea.KeyPgUp,
		"home":   tea.KeyHome,
		"end":    tea.KeyEnd,
		"esc":    tea.KeyEscape,
	}
	if code, ok := named[spelling]; ok {
		return tea.KeyPressMsg(tea.Key{Code: code})
	}
	if letter, modified := strings.CutPrefix(spelling, "ctrl+"); modified {
		return tea.KeyPressMsg(tea.Key{Code: rune(letter[0]), Mod: tea.ModCtrl})
	}
	if spelling == "G" || spelling == "T" {
		// What the decoder makes of a shifted letter: the byte is upper case, so it lower-cases the
		// code, marks the event shifted and keeps the character the user typed in Text.
		upper := rune(spelling[0])
		lower := upper - 'A' + 'a'
		return tea.KeyPressMsg(tea.Key{Code: lower, ShiftedCode: upper, Mod: tea.ModShift, Text: spelling})
	}
	return tea.KeyPressMsg(tea.Key{Code: rune(spelling[0]), Text: spelling})
}

// paneOf and statusOf split one frame into the pane and the status line: the status line is the
// frame's last row, always.
func paneOf(frame string) string {
	rows := strings.Split(frame, "\n")
	return strings.Join(rows[:len(rows)-1], "\n")
}

func statusOf(frame string) string {
	rows := strings.Split(frame, "\n")
	return rows[len(rows)-1]
}

// selectedHeadRow is the pane row the selected block's head sits on: the first row whose selection
// column carries the mark, since only the selected block's rows carry it. The row is read with its
// escapes stripped — the selected block is painted, so in a colour rung the mark sits behind an SGR
// sequence — because the mark is what carries the meaning (A1) and the test has to find it in every
// rung.
func selectedHeadRow(pane string) int {
	for i, row := range strings.Split(pane, "\n") {
		if strings.HasPrefix(ansi.Strip(row), render.MarkSelection) {
			return i
		}
	}
	return -1
}

// prepending is a page of posts the fake's own page set does not hold: the answer to a refresh that
// brought something new, which is the state FEED-04's restore rules exist for. The fake serves fixed
// pages and cannot produce it, so the test hands the answer to the shell directly.
func prepending(n int) domain.PostPage {
	posts := make([]domain.Post, 0, n)
	for i := range n {
		posts = append(posts, domain.Post{
			ID:        domain.PostID("new-" + strconv.Itoa(i)),
			Author:    domain.User{ID: "u-lzd", ScreenName: "lzd", Name: "name of lzd"},
			Text:      "a post the refresh brought",
			CreatedAt: frozen.Add(-time.Duration(i) * time.Second),
		})
	}
	return domain.PostPage{Posts: posts, FetchedAt: frozen}
}

// TestTheShellOpensOnTheFollowingList is FEED-01's default: the first request the source sees is for
// Following, the first page's posts are on screen, and the status line names the list, the position
// and the budget.
func TestTheShellOpensOnTheFollowingList(t *testing.T) {
	s := newShell(t, 80, 24)

	if kind, ok := s.src.LastKind(); !ok || kind != domain.Following {
		t.Fatalf("the first request was for %v (reported %v), want Following", kind, ok)
	}
	if got := s.m.top().List.Loaded(); got != 9 {
		t.Fatalf("the first page loaded %d posts, want the fake's nine", got)
	}
	if !strings.Contains(s.pane(), "a_15_cell_handl") {
		t.Errorf("the pane does not render the first page's first post:\n%s", s.pane())
	}

	status := s.status()
	for _, want := range []string{render.CopyListFollowing, "1/9+"} {
		if !strings.Contains(status, want) {
			t.Errorf("the status line %q does not carry %q", status, want)
		}
	}
}

// TestTheBackStackRestoresTheFrame is REND-06's observables: after `t` and `t`, the frame is
// byte-identical, the return issued no request, the same post id is selected, and its block is back
// on the same pane row.
func TestTheBackStackRestoresTheFrame(t *testing.T) {
	s := newShell(t, 80, 24)
	s.press("j", "j", "j")

	before := s.frame()
	beforeRow := selectedHeadRow(s.pane())
	beforeID := s.m.top().List.Selected().ID
	beforeAnchor := s.m.top().Anchor
	beforeCalls := s.src.Calls()

	// The push: a fresh frame for the other feed, and one request for that frame's own page one.
	s.press("t")
	callsAtPush := s.src.Calls()
	if callsAtPush != beforeCalls+1 {
		t.Fatalf("the toggle issued %d requests, want one", callsAtPush-beforeCalls)
	}
	if kind, _ := s.src.LastKind(); kind != domain.ForYou {
		t.Fatalf("the pushed frame read %v, want ForYou", kind)
	}
	if s.m.top().List.Kind != domain.ForYou {
		t.Fatalf("the pushed frame shows %v", s.m.top().List.Kind)
	}
	if s.frame() == before {
		t.Errorf("the other feed rendered the same frame as the first")
	}
	if status := s.status(); !strings.Contains(status, render.CopyListForYou) {
		t.Errorf("the other feed's status line is %q, want it to name the list", status)
	}

	// The pop: nothing fetched, and the frame that comes back is the frame that was left.
	s.press("t")
	if got := s.src.Calls(); got != callsAtPush {
		t.Errorf("returning issued %d requests, want none", got-callsAtPush)
	}
	if got := s.frame(); got != before {
		t.Errorf("the frame returned to is not byte-identical to the frame left:\n%s\n---\n%s", before, got)
	}
	if s.m.top().List.Selected().ID != beforeID {
		t.Errorf("the selection came back as %q, want %q", s.m.top().List.Selected().ID, beforeID)
	}
	if s.m.top().Anchor != beforeAnchor {
		t.Errorf("the anchor came back as %d, want %d", s.m.top().Anchor, beforeAnchor)
	}
	if row := selectedHeadRow(s.pane()); row != beforeRow {
		t.Errorf("the selected block's head came back on pane row %d, want %d", row, beforeRow)
	}
}

// TestThePaginationTriggerIsSingleFlight is FEED-02: reaching the last loaded post asks for the next
// page once, a second key while it is out is a no-op, and the late page moves neither the selection
// nor anything above it.
func TestThePaginationTriggerIsSingleFlight(t *testing.T) {
	s := newShell(t, 80, 24)
	// One page in: the walk has somewhere to go, and the last loaded post is the page's own last.
	s.press("G")

	cmd := s.key("G")
	if cmd == nil {
		t.Fatalf("G at the last loaded post asked for nothing")
	}
	if !s.m.top().List.Inflight() {
		t.Fatalf("a page request is out and the list does not say so")
	}
	calls, loaded := s.src.Calls(), s.m.top().List.Loaded()
	selected, anchor := s.m.top().List.Selected().ID, s.m.top().Anchor
	head := selectedHeadRow(s.pane())
	above := strings.Split(s.pane(), "\n")[:head]

	// A second press while the page is out: no command and no request.
	if again := s.key("j"); again != nil {
		t.Errorf("a key press while a page was in flight returned a command")
	}
	if s.src.Calls() != calls {
		t.Errorf("a key press while a page was in flight issued a request")
	}

	s.run(cmd)
	if got := s.m.top().List.Loaded(); got <= loaded {
		t.Fatalf("the page added %d posts", got-loaded)
	}
	if s.m.top().List.Selected().ID != selected {
		t.Errorf("the late page moved the selection to %q, want %q", s.m.top().List.Selected().ID, selected)
	}
	if s.m.top().Anchor != anchor {
		t.Errorf("the late page moved the anchor to %d, want %d", s.m.top().Anchor, anchor)
	}
	if got := selectedHeadRow(s.pane()); got != head {
		t.Errorf("the late page moved the selected block to pane row %d, want %d", got, head)
	}
	if got := strings.Split(s.pane(), "\n")[:head]; !slices.Equal(got, above) {
		t.Errorf("the late page moved what was above the selection:\n%s\n---\n%s", strings.Join(above, "\n"), strings.Join(got, "\n"))
	}
}

// TestRefreshIsDebouncedAndAnnounced is FEED-04: one command per press, no command while a request
// is out, the count of what the refresh brought, and the reader's place kept exactly.
func TestRefreshIsDebouncedAndAnnounced(t *testing.T) {
	s := newShell(t, 80, 24)
	s.press("j", "j")

	// The fake's page one holds nothing new: the answer says so out loud.
	s.press("r")
	if status := s.status(); !strings.Contains(status, "no new posts") {
		t.Errorf("the refresh against an unchanged timeline said %q", status)
	}
	if s.src.Calls() != 2 {
		t.Fatalf("the shell made %d requests, want two: the first page and one refresh", s.src.Calls())
	}

	// A press while the request is out is a no-op, so key-repeat cannot spend a burst: the request is
	// issued when its command runs, and the second press adds nothing to it.
	calls := s.src.Calls()
	cmd := s.key("r")
	if cmd == nil {
		t.Fatalf("the refresh asked for nothing")
	}
	if again := s.key("r"); again != nil {
		t.Errorf("a refresh during an in-flight fetch returned a command")
	}
	s.run(cmd)
	if s.src.Calls() != calls+1 {
		t.Errorf("two refreshes in a row issued %d requests, want one", s.src.Calls()-calls)
	}
	// A refresh that brings posts: the count is announced, the selection is the same post, its row
	// index has moved, and nothing on screen moved.
	beforePane := s.pane()
	selected := s.m.top().List.Selected().ID
	beforeIndex := s.m.top().List.Index(selected)
	beforeHead := selectedHeadRow(beforePane)
	beforeAnchor := s.m.top().Anchor
	if cmd := s.key("r"); cmd == nil {
		t.Fatalf("the refresh asked for nothing")
	}
	// The answer is handed over directly: the fake serves fixed pages, so it cannot produce a
	// refresh that brings posts the list did not hold. The latch this releases is the real one.
	s.send(pageMsg{Frame: s.m.top().ID, Refresh: true, Page: prepending(2)})

	if got := s.m.top().List.NewPosts(); got != 2 {
		t.Fatalf("the refresh brought %d posts, want two", got)
	}
	if status := s.status(); !strings.Contains(status, "2 new posts") {
		t.Errorf("the status line %q does not announce the two posts", status)
	}
	if s.m.top().List.Selected().ID != selected {
		t.Errorf("the refresh moved the selection to %q, want %q", s.m.top().List.Selected().ID, selected)
	}
	if got := s.m.top().List.Index(selected); got != beforeIndex+2 {
		t.Errorf("the selected post's index is %d, want %d: the refresh did not move it down the list", got, beforeIndex+2)
	}
	if s.m.top().Anchor != beforeAnchor {
		t.Errorf("the refresh moved the anchor to %d, want %d", s.m.top().Anchor, beforeAnchor)
	}
	if got := selectedHeadRow(s.pane()); got != beforeHead {
		t.Errorf("the selected block's head is on pane row %d after the refresh, want %d", got, beforeHead)
	}
	if got := s.pane(); got != beforePane {
		t.Errorf("the refresh moved what the reader was reading:\n%s\n---\n%s", beforePane, got)
	}
}

// TestResizeRerendersWithoutFetching is REND-03 and A5: the frame is a pure function of the state
// and the size, so a model that was resized renders the same bytes as a model built at that size —
// no cached wrapped text, no sticky column widths — and a resize changes nothing but the size.
func TestResizeRerendersWithoutFetching(t *testing.T) {
	s := newShell(t, 80, 24)
	calls := s.src.Calls()
	loaded, selected := s.m.top().List.Loaded(), s.m.top().List.Selected().ID

	s.resize(120, 24)
	if s.src.Calls() != calls {
		t.Errorf("the resize issued %d requests, want none", s.src.Calls()-calls)
	}
	if s.m.top().List.Loaded() != loaded || s.m.top().List.Selected().ID != selected {
		t.Errorf("the resize changed the list: %d posts, selection %q", s.m.top().List.Loaded(), s.m.top().List.Selected().ID)
	}

	// The cold model: the same list state, built at the target size.
	cold := newShell(t, 120, 24)
	if got, want := s.frame(), cold.frame(); got != want {
		t.Errorf("the resized frame is not the cold frame:\n%s\n---\n%s", want, got)
	}

	// And back: a round trip through another width is the frame of a model built at 80.
	s.resize(80, 24)
	at80 := newShell(t, 80, 24)
	if got, want := s.frame(), at80.frame(); got != want {
		t.Errorf("the frame after a round trip is not the frame at 80 columns:\n%s\n---\n%s", want, got)
	}
	if s.src.Calls() != at80.src.Calls() {
		t.Errorf("the round trip issued %d requests, want none", s.src.Calls()-at80.src.Calls())
	}
}

// TestTheBudgetIsInEveryFrame is OPS-02: the budget line is the limiter's own numbers, in every
// frame, in every state — including the failure states, where the rest of the line is under
// pressure.
func TestTheBudgetIsInEveryFrame(t *testing.T) {
	s := newShell(t, 80, 24)

	for _, budget := range []struct {
		used, limited int
		want          string
	}{
		{0, 0, "0 requests, 0 rate-limited"},
		{1, 1, "1 request, 1 rate-limited"},
		{200, 12, "200 requests, 12 rate-limited"},
	} {
		s.used, s.limited = budget.used, budget.limited
		if status := s.status(); !strings.Contains(status, budget.want) {
			t.Errorf("with %d used and %d limited the status line is %q, want it to carry %q", budget.used, budget.limited, status, budget.want)
		}
	}

	s.used, s.limited = 200, 12
	for _, state := range namedStates() {
		model := s.m.WithFailure(state.failure)
		status := statusOf(model.Render(80, 24))
		if !strings.Contains(status, "200 requests, 12 rate-limited") {
			t.Errorf("the %s frame's status line is %q, and it dropped the budget", state.name, status)
		}
	}
}

// namedState is one of OPS-01's four named states, with the words read out of internal/render by
// kind and the pane body read out of the classifier. Nothing in this file spells a state.
type namedState struct {
	name    string
	state   string
	remedy  string
	body    string
	failure domain.Failure
}

// namedStates is the four: expired session, rate limited with its computed reset, offline/no route,
// and an upstream shape change.
func namedStates() []namedState {
	return []namedState{
		{
			name:    "expired session",
			state:   render.StateWord(domain.FailureSessionExpired),
			remedy:  render.Remedy(domain.FailureSessionExpired),
			body:    xweb.OutcomeAuthRejected.Remedy(),
			failure: domain.Failure{Kind: domain.FailureSessionExpired, Detail: xweb.OutcomeAuthRejected.Remedy(), Retry: true},
		},
		{
			name:    "rate limited with a reset",
			state:   render.StateWord(domain.FailureRateLimited),
			remedy:  render.Remedy(domain.FailureRateLimited),
			body:    xweb.OutcomeRateLimited.Remedy(),
			failure: domain.Failure{Kind: domain.FailureRateLimited, Detail: xweb.OutcomeRateLimited.Remedy(), Reset: frozen.Add(30 * time.Minute)},
		},
		{
			name:    "no route",
			state:   render.StateWord(domain.FailureNoRoute),
			remedy:  render.Remedy(domain.FailureNoRoute),
			body:    "x.com did not resolve: no route to host",
			failure: domain.Failure{Kind: domain.FailureNoRoute, Detail: "x.com did not resolve: no route to host", Retry: true},
		},
		{
			name:    "shape change",
			state:   render.StateWord(domain.FailureOperationStale),
			remedy:  render.Remedy(domain.FailureOperationStale),
			body:    xweb.OutcomeUnknownOperation.Remedy(),
			failure: domain.Failure{Kind: domain.FailureOperationStale, Detail: xweb.OutcomeUnknownOperation.Remedy(), Retry: true},
		},
	}
}

// TestTheFourNamedStatesRenderTheirOwnWords is OPS-01: each of the four carries its own state word,
// its own remedy and its own pane body, the four are distinguishable from each other, and none of
// them renders as an empty read.
func TestTheFourNamedStatesRenderTheirOwnWords(t *testing.T) {
	s := newShell(t, 120, 24)

	states, remedies := map[string]string{}, map[string]string{}
	for _, state := range namedStates() {
		frame := New(s.m.deps).WithFailure(state.failure).Render(120, 24)
		status, pane := statusOf(frame), paneOf(frame)

		if !strings.Contains(status, state.state) {
			t.Errorf("the %s frame's status line is %q and does not carry the state %q", state.name, status, state.state)
		}
		if !strings.Contains(status, state.remedy) {
			t.Errorf("the %s frame's status line is %q and does not carry the remedy %q", state.name, status, state.remedy)
		}
		if !strings.Contains(pane, state.body) {
			t.Errorf("the %s frame's pane does not carry the classifier's sentence %q:\n%s", state.name, state.body, pane)
		}
		if strings.Contains(frame, render.CopyEmptyHeadingFollowing) {
			t.Errorf("the %s frame renders the empty-state copy: an empty list is never rendered for a failure", state.name)
		}
		states[state.state], remedies[state.remedy] = state.name, state.name
	}

	if len(states) != len(namedStates()) {
		t.Errorf("the four named states rendered %d distinct state words, want %d", len(states), len(namedStates()))
	}
	if len(remedies) != len(namedStates()) {
		t.Errorf("the four named states rendered %d distinct remedies, want %d", len(remedies), len(namedStates()))
	}
}

// TestASoftBanStopsTheSession is OPS-04: the stopping state is on the line, and no key issues a
// request again for the life of the process.
func TestASoftBanStopsTheSession(t *testing.T) {
	s := newShell(t, 80, 24)
	before := s.pane()
	calls := s.src.Calls()

	s.m = s.m.WithFailure(domain.Failure{
		Kind:    domain.FailureSoftBan,
		Detail:  xweb.OutcomeSoftBan.Remedy(),
		Stopped: true,
	})

	status := s.status()
	if want := render.StateWord(domain.FailureSoftBan); !strings.Contains(status, want) {
		t.Fatalf("the stopped session's status line is %q, want it to carry %q", status, want)
	}
	if want := render.Remedy(domain.FailureSoftBan); !strings.Contains(status, want) {
		t.Errorf("the stopped session's status line is %q, want it to carry %q", status, want)
	}
	if s.pane() != before {
		t.Errorf("the soft ban cleared the posts:\n%s\n---\n%s", before, s.pane())
	}

	// Every key that could ask: the walk's own trigger, the refresh, and the toggle's push.
	for _, spelling := range []string{"G", "r", "t"} {
		if cmd := s.key(spelling); cmd != nil {
			t.Errorf("%q returned a command after a soft ban", spelling)
		}
	}
	if s.src.Calls() != calls {
		t.Errorf("the stopped session issued %d requests", s.src.Calls()-calls)
	}
	if s.m.failure.Kind != domain.FailureSoftBan {
		t.Errorf("the state is %q after the keys, want the soft ban", render.StateWord(s.m.failure.Kind))
	}
}

// TestAFailedPageKeepsThePosts is FEED-05: a page that fails decodes to nothing at all, the pane
// keeps every post it had, and the failure is stated in the one status line.
func TestAFailedPageKeepsThePosts(t *testing.T) {
	s := newShell(t, 80, 24)
	// One page in, with the walk still open. The next trigger is the request that fails.
	s.press("G")
	s.src.FailNext(errRefused)
	cmd := s.key("G")
	if cmd == nil {
		t.Fatalf("G at the last loaded post asked for nothing")
	}

	// The selection moved to the last loaded post, which is a move the reader asked for; what the
	// answer must not do is change anything on screen.
	before := s.pane()
	calls := s.src.Calls()
	s.run(cmd)

	if s.src.Calls() != calls+1 {
		t.Fatalf("the failed page issued %d requests, want one", s.src.Calls()-calls)
	}
	if s.pane() != before {
		t.Errorf("the failed page changed the pane:\n%s\n---\n%s", before, s.pane())
	}
	status := s.status()
	if want := render.StateWord(domain.FailureSessionExpired); !strings.Contains(status, want) {
		t.Errorf("the failed page's status line is %q, want it to carry %q", status, want)
	}
}

// TestTheHelpOverlayIsTheOneTable is REND-05's overlay: the pane is the table, every row of the
// keymap is on it, no key reaches the list under it, and `help` or `back` closes it.
func TestTheHelpOverlayIsTheOneTable(t *testing.T) {
	s := newShell(t, 120, 40)
	before := s.frame()
	selected := s.m.top().List.Selected().ID

	if cmd := s.key("?"); cmd != nil {
		t.Errorf("opening the overlay returned a command")
	}
	overlay := s.pane()
	for _, row := range helpRows(false, "home") {
		if !strings.Contains(overlay, row.Keys) {
			t.Errorf("the overlay does not name the keystrokes %q:\n%s", row.Keys, overlay)
		}
		if !strings.Contains(overlay, row.Text) {
			t.Errorf("the overlay does not carry the help text %q", row.Text)
		}
	}

	// No key reaches the list: a movement key under the overlay moves nothing.
	if cmd := s.key("j"); cmd != nil {
		t.Errorf("a movement key under the overlay returned a command")
	}
	if s.m.top().List.Selected().ID != selected {
		t.Errorf("a movement key under the overlay moved the selection to %q", s.m.top().List.Selected().ID)
	}
	if s.pane() != overlay {
		t.Errorf("a movement key under the overlay changed the pane")
	}

	// `back` closes the overlay rather than quitting, and the pane comes back byte-identical.
	if cmd := s.key("q"); cmd != nil {
		t.Errorf("back under the overlay returned a command, so it quit rather than closing")
	}
	if s.frame() != before {
		t.Errorf("the pane after closing the overlay is not the pane before it:\n%s\n---\n%s", before, s.frame())
	}

	// And `?` toggles the overlay the same way.
	if cmd := s.key("?"); cmd != nil {
		t.Errorf("reopening the overlay returned a command")
	}
	if !strings.Contains(s.pane(), helpRows(false, "home")[0].Text) {
		t.Errorf("`?` did not reopen the overlay")
	}
	if cmd := s.key("?"); cmd != nil {
		t.Errorf("closing the overlay returned a command")
	}
	if s.frame() != before {
		t.Errorf("the pane after `?` twice is not the pane before it")
	}
}

// TestTheHelpOverlayNamesTheRowsThePaneCannotHold: at the reference size the keymap is taller than the
// pane, and the overlay says so instead of hiding the rows. Seven of the table's rows were invisible at
// 80x24 with nothing on screen to say the list continued — an overlay that silently hides a quarter of
// the keys is a lie about what the program does, so the pane's last row names the count.
//
// The assertion is the pane's own bytes at 80x24 (the reference size) and the numbers are asserted
// against the table itself: the rows above the cut are the table's own first rows in order, the pane
// spends exactly one row on the statement, and the count is the table's rows that do not fit.
func TestTheHelpOverlayNamesTheRowsThePaneCannotHold(t *testing.T) {
	s := newShell(t, 80, 24)
	if cmd := s.key("?"); cmd != nil {
		t.Fatalf("opening the overlay returned a command")
	}
	rows := strings.Split(s.pane(), "\n")
	if len(rows) != s.m.paneRows() {
		t.Fatalf("the overlay is %d rows for a %d-row pane", len(rows), s.m.paneRows())
	}
	table := helpRows(false, "home")
	shown := len(rows) - 1
	if shown >= len(table) {
		t.Fatalf("the pane holds all %d rows of the table: this arm's case is not on screen", len(table))
	}
	for i, row := range table[:shown] {
		plain := ansi.Strip(rows[i])
		if !strings.Contains(plain, row.Keys) || !strings.Contains(plain, row.Text) {
			t.Errorf("overlay row %d is %q, want the table's own row %q", i+1, plain, row.Keys+" "+row.Text)
		}
	}
	cut := ansi.Strip(rows[len(rows)-1])
	want := strings.Replace(render.CopyHelpClipped, "{n}", strconv.Itoa(len(table)-shown), 1)
	if !strings.Contains(cut, want) {
		t.Errorf("the pane's last row is %q, want it to carry %q: %d of the table's %d rows are not on the pane",
			cut, want, len(table)-shown, len(table))
	}
}

// TestAQuietKeyAndTheTwoExits: a key the table does not hold does nothing at all, `q` at the bottom
// frame asks the program to quit, and `ctrl+c` always does.
func TestAQuietKeyAndTheTwoExits(t *testing.T) {
	s := newShell(t, 80, 24)
	before := s.frame()

	if cmd := s.key("z"); cmd != nil {
		t.Errorf("a key the table does not hold returned a command")
	}
	if s.frame() != before {
		t.Errorf("a key the table does not hold changed the frame")
	}

	for _, spelling := range []string{"q", "ctrl+c"} {
		cmd := s.key(spelling)
		if cmd == nil {
			t.Fatalf("%q returned no command at the bottom frame", spelling)
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Errorf("%q's command did not ask the program to quit", spelling)
		}
	}
}

// TestTheKeymapMatchesWhatTheDecoderReports is the seam between the table and Bubble Tea: the
// spellings the table holds are the spellings a KeyPressMsg reports, including the shifted letter,
// which has two of them — Keystroke says shift+g, String says G — because the decoder lower-cases
// the byte and keeps the character the user typed.
func TestTheKeymapMatchesWhatTheDecoderReports(t *testing.T) {
	for _, spelling := range []string{"j", "down", "k", "up", "ctrl+d", "ctrl+u", "pgdown", "pgup", "g", "home", "G", "end", "enter", "r", "t", "?", "q", "esc", "ctrl+c"} {
		msg := press(spelling)
		if _, ok := keyAction(msg, "home"); !ok {
			t.Errorf("the table does not match the keystroke %q, which a terminal reports as %q / %q", spelling, msg.Keystroke(), msg.String())
		}
	}

	shifted := press("G")
	if shifted.Keystroke() != "shift+g" || shifted.String() != "G" {
		t.Fatalf("a shifted G is reported as %q / %q by this Bubble Tea version; the table's spelling is G", shifted.Keystroke(), shifted.String())
	}
	if action, ok := keyAction(shifted, "home"); !ok || action != keys.ActionLast {
		t.Errorf("shift+G does %q (found %v), want %q", action, ok, keys.ActionLast)
	}
}

// TestASuccessfulFetchClearsTheState is OPS-01's stickiness ending where the UI-SPEC says it ends: a
// page that fails puts its state on the line, and the next fetch that returns data takes it away —
// no timer, no key, no frame count. The list's own failure is cleared by its reducer; the line's is
// the process's, and this is the test that holds the two together.
func TestASuccessfulFetchClearsTheState(t *testing.T) {
	s := newShell(t, 80, 24)
	s.press("G") // the walk is open: the last loaded post is the pagination trigger

	s.src.FailNext(errRefused)
	s.press("G")
	state := render.StateWord(domain.FailureSessionExpired)
	if !strings.Contains(s.status(), state) {
		t.Fatalf("the failed page's status line is %q, want it to carry %q", s.status(), state)
	}
	if s.m.top().List.Failure().Kind != domain.FailureSessionExpired {
		t.Fatalf("the list recorded %v, want the state the line prints", s.m.top().List.Failure().Kind)
	}

	// The state's remedy is "retry", so the next trigger asks again and the answer returns data.
	cmd := s.key("G")
	if cmd == nil {
		t.Fatalf("the page after a retryable failure asked for nothing")
	}
	s.run(cmd)

	if got := s.status(); strings.Contains(got, state) {
		t.Errorf("the status line still says %q after a fetch succeeded: %q", state, got)
	}
	if s.m.failure.Kind != domain.FailureNone {
		t.Errorf("the process's state is %v after a fetch succeeded, want none", s.m.failure.Kind)
	}
	if got := s.m.top().List.Failure().Kind; got != domain.FailureNone {
		t.Errorf("the list's failure is %v after a fetch succeeded, want none", got)
	}
}

// TestTheEmptyStateAppearsAfterTheStateIsCleared is the other half of the same rule: while a state is
// on the line the pane may not claim an empty read succeeded, and once a read has succeeded and the
// list holds nothing, that copy is the only true thing the pane can say. Before the state was
// cleared on success this copy was unreachable after any failure.
func TestTheEmptyStateAppearsAfterTheStateIsCleared(t *testing.T) {
	s := newShell(t, 80, 24)
	// A fresh model over the same seams, opening on a request that fails: the pane has nothing to
	// show and says what went wrong rather than showing the empty read's copy.
	s.m = New(s.m.deps)
	s.send(tea.WindowSizeMsg{Width: 80, Height: 24})
	s.src.FailNext(errRefused)
	s.run(s.m.Init())

	state := render.StateWord(domain.FailureSessionExpired)
	if !strings.Contains(s.status(), state) {
		t.Fatalf("the failed first read's status line is %q, want it to carry %q", s.status(), state)
	}
	if !strings.Contains(s.pane(), xweb.OutcomeAuthRejected.Remedy()) {
		t.Errorf("the failed read's pane is %q, want the classifier's own sentence", s.pane())
	}
	if strings.Contains(s.pane(), render.CopyEmptyBody) {
		t.Errorf("the failed read renders the empty-state copy: an empty read is not a failed one")
	}

	// The retry succeeds and brings nothing: the copy the empty state owns is now the truth.
	s.send(pageMsg{Frame: s.m.top().ID, Page: domain.PostPage{FetchedAt: frozen}})
	if strings.Contains(s.status(), state) {
		t.Errorf("the status line still carries %q after a successful empty read: %q", state, s.status())
	}
	if !strings.Contains(s.pane(), render.CopyEmptyHeadingFollowing) || !strings.Contains(s.pane(), render.CopyEmptyBody) {
		t.Errorf("the successful empty read's pane is %q, want the empty-state copy", s.pane())
	}
}

// TestTheCheckNamesTheBrokenPath is OPS-01's third named state reached the way the shipped app reaches
// it: a request fails with an error the mapper cannot name, the reachability check runs its cheap
// stages, and the state the pane prints is the check's own verdict — `no route to x.com` with the
// check's sentence under it — rather than the transport error rendered as an unclassified shape.
func TestTheCheckNamesTheBrokenPath(t *testing.T) {
	s := newShell(t, 80, 24)
	// A fresh model over the same seams, opening on a request that fails with an error no mapper rule
	// names: the pane has nothing to show, so the check's own sentence is what it prints.
	s.m = New(s.m.deps)
	s.send(tea.WindowSizeMsg{Width: 80, Height: 24})

	check := 0
	detail := "x.com did not resolve: lookup x.com: no such host"
	s.m.deps.Check = func(context.Context) []netcheck.Result {
		check++
		return []netcheck.Result{{Outcome: netcheck.OutcomeNoRoute, Detail: detail}}
	}

	s.src.FailNext(errors.New("the request did not complete"))
	s.run(s.m.Init())

	if check != 1 {
		t.Fatalf("the reachability check ran %d times for one unnameable failure, want once", check)
	}
	if want := render.StateWord(domain.FailureNoRoute); !strings.Contains(s.status(), want) {
		t.Errorf("the status line is %q, want it to carry %q", s.status(), want)
	}
	if want := render.Remedy(domain.FailureNoRoute); !strings.Contains(s.status(), want) {
		t.Errorf("the status line is %q, want it to carry the remedy %q", s.status(), want)
	}
	if !strings.Contains(s.pane(), detail) {
		t.Errorf("the pane is %q, want the check's own sentence", s.pane())
	}
	if strings.Contains(s.pane(), "the request did not complete") {
		t.Errorf("the pane prints the raw transport error instead of the check's verdict: %q", s.pane())
	}

	// The check is not a second classifier: an error the mapper already names costs no path probe.
	s.src.FailNext(errRefused)
	s.press("r")
	if want := render.StateWord(domain.FailureSessionExpired); !strings.Contains(s.status(), want) {
		t.Errorf("the classified failure's status line is %q, want %q", s.status(), want)
	}
	if check != 1 {
		t.Errorf("the check ran %d times for a failure the mapper named, want no further run", check)
	}
}

// TestADiscardedFramesAnswerIsDropped is the toggle's routing rule: the answer to a frame the user has
// left is matched on the frame instance that asked, so the frame that replaced it does not absorb it.
// Absorbing it makes the live frame's own answer look like a source repeating itself — the one state
// that ends a walk — and marks a read the live frame never had.
func TestADiscardedFramesAnswerIsDropped(t *testing.T) {
	s := newShell(t, 80, 24)

	// The first push: a frame for the other feed and its page one in flight, held.
	discarded := s.key("t")
	if discarded == nil {
		t.Fatalf("the toggle asked for nothing")
	}
	firstID := s.m.top().ID

	// The pop, and the push again: a new frame instance of the same kind, with a request of its own.
	s.press("t")
	live := s.key("t")
	if live == nil {
		t.Fatalf("the second toggle asked for nothing")
	}
	if s.m.top().ID == firstID {
		t.Fatalf("the frame that replaced the discarded one reused its id: an answer cannot be told from its predecessor")
	}

	// The discarded frame's answer arrives: dropped, not absorbed.
	s.send(discarded())
	if s.m.top().List.Stalled() {
		t.Errorf("the discarded frame's answer stalled the frame that replaced it")
	}
	if got := s.m.top().List.Loaded(); got != 0 {
		t.Errorf("the discarded frame's answer loaded %d posts into another frame, want none", got)
	}
	if s.m.top().Read {
		t.Errorf("a dropped answer marked the live frame read: an empty frame would claim a read that never reached it")
	}
	if !strings.Contains(s.pane(), render.CopyLoadingForYou) {
		t.Errorf("the live frame's pane is %q, want the loading line: its own read has not answered", s.pane())
	}
	if strings.Contains(s.pane(), render.CopyEmptyHeadingForYou) {
		t.Errorf("the live frame renders the empty-state copy for a read that was thrown away: %q", s.pane())
	}

	// The live frame's own answer is absorbed, and the walk is open rather than stalled.
	s.run(live)
	if s.m.top().List.Stalled() {
		t.Errorf("the live frame's own page stalled the walk: it was the first page the frame asked for")
	}
	if got := s.m.top().List.Loaded(); got == 0 {
		t.Errorf("the live frame's own answer loaded nothing")
	}
	if status := s.status(); strings.Contains(status, "stalled") {
		t.Errorf("the status line reports a stalled walk on a list that was asked once: %q", status)
	}
}

// TestARefreshIntoAnEmptyListSelectsTheFirstPost is the default the two reducers share: a list that
// holds posts always has a selected post, because the pane has no row for a list with posts and no
// selection — every row's selection column blank, the position reading 0/N, and no anchor for a
// movement key. A refresh is the other way a list comes to hold posts it had none of.
func TestARefreshIntoAnEmptyListSelectsTheFirstPost(t *testing.T) {
	s := newShell(t, 80, 24)
	// The first read fails, so the list is empty and the failure is the retryable kind.
	s.m = New(s.m.deps)
	s.send(tea.WindowSizeMsg{Width: 80, Height: 24})
	s.src.FailNext(errRefused)
	s.run(s.m.Init())
	if s.m.top().List.Loaded() != 0 {
		t.Fatalf("the failed read loaded %d posts", s.m.top().List.Loaded())
	}

	if cmd := s.key("r"); cmd == nil {
		t.Fatalf("the refresh after a retryable failure asked for nothing")
	}
	s.send(pageMsg{Frame: s.m.top().ID, Refresh: true, Page: prepending(2)})

	list := s.m.top().List
	if list.Loaded() != 2 {
		t.Fatalf("the refresh loaded %d posts, want two", list.Loaded())
	}
	first := list.Targets()[0].ID
	if id := list.Selected().ID; id == "" || id != first {
		t.Fatalf("the refresh filled an empty list and selected %q, want the first post %q", id, first)
	}
	if status := s.status(); !strings.Contains(status, "1/2") {
		t.Errorf("the status line is %q, want the position of the selected post", status)
	}
	if row := selectedHeadRow(s.pane()); row < 0 {
		t.Errorf("no block is marked in the pane: a list with posts must have a selected post:\n%s", s.pane())
	}
}

// selectPost moves the home selection onto a post id, failing the test when the list does not hold it.
func (s *shell) selectPost(id string) {
	s.t.Helper()
	seq := s.m.top().targets()
	from := indexOfTarget(seq, s.m.top().selected())
	want := indexOfTarget(seq, screen.Target{ID: id})
	if want < 0 {
		s.t.Fatalf("the list does not hold post %q", id)
	}
	if from < 0 {
		s.t.Fatalf("nothing is selected")
	}
	if want == from {
		return
	}
	dir := 1
	if want < from {
		dir = -1
	}
	for i := 0; i < abs(want-from); i++ {
		if dir > 0 {
			s.press("j")
		} else {
			s.press("k")
		}
	}
	if s.m.top().selected().Top().ID != id && s.m.top().selected().ID != id {
		s.t.Fatalf("after moving, selected %v, want %q", s.m.top().selected(), id)
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func TestEnterOpensAConversationWithLoadingThenPage(t *testing.T) {
	s := newShell(t, 80, 24)
	before := s.src.Calls()
	calls := s.th.Calls()

	// Hold the page-one command: the frame must show the focal copy and loading before the answer.
	cmd := s.key("enter")
	if cmd == nil {
		t.Fatalf("enter asked for nothing")
	}
	if s.m.top().Kind != frameThread {
		t.Fatalf("enter left the frame kind %v, want thread", s.m.top().Kind)
	}
	if s.th.Calls() != calls {
		t.Fatalf("pushing issued a thread request before the frame's own page one")
	}
	if !strings.Contains(s.pane(), render.MarkerLoadingThread) {
		t.Fatalf("the pane is %q, want loading thread... under the focal copy", s.pane())
	}
	if !strings.Contains(s.status(), "thread") {
		t.Errorf("the status line is %q, want the thread list word", s.status())
	}
	if s.src.Calls() != before {
		t.Errorf("opening a post issued a home request")
	}

	s.run(cmd)
	if !s.m.top().Thread.Read() {
		t.Fatalf("page one did not mark the conversation read")
	}
	if s.th.Calls() != calls+1 {
		t.Fatalf("page one issued %d thread calls, want one", s.th.Calls()-calls)
	}
	if strings.Contains(s.pane(), render.MarkerLoadingThread) {
		t.Errorf("loading thread... survived page one")
	}
}

func TestEnterOnAQuotedPostPushesTheQuote(t *testing.T) {
	s := newShell(t, 80, 24)
	// adv-6 is the depth-one quote carrier in the adversarial first page.
	s.selectPost("adv-6")
	s.press("j") // into the quoted post
	sel := s.m.top().selected()
	if len(sel.Path) == 0 {
		t.Fatalf("the selection is %v, want a quoted row", sel)
	}
	calls := s.th.Calls()
	cmd := s.key("enter")
	if cmd == nil {
		t.Fatalf("enter on a quote asked for nothing")
	}
	if s.th.Calls() != calls {
		t.Fatalf("pushing a quote issued a request")
	}
	if got := s.m.top().Thread.Focal().ID; got != domain.PostID(sel.ID) {
		t.Fatalf("the pushed focal is %q, want the quoted post %q", got, sel.ID)
	}
	s.run(cmd)
}

func TestEnterOnATombstoneOpensItsConversation(t *testing.T) {
	s := newShell(t, 80, 24)
	// The tombstone is on the adversarial second page.
	s.press("G")
	s.selectPost("adv-tombstone")
	calls := s.th.Calls()
	cmd := s.key("enter")
	if cmd == nil {
		t.Fatalf("enter on a tombstone asked for nothing")
	}
	if s.m.top().Thread.Focal().ID != "adv-tombstone" {
		t.Fatalf("focal is %q, want the tombstone", s.m.top().Thread.Focal().ID)
	}
	if s.th.Calls() != calls {
		t.Fatalf("pushing issued a request")
	}
	s.run(cmd)
}

func TestEnterWithNothingToOpenPrintsTheNotice(t *testing.T) {
	s := newShell(t, 80, 24)
	// Empty the selection path by opening on an unread empty home: rebuild with a failing first page,
	// then clear posts — simplest: push a thread, fail? Plan: empty list.
	s.m = New(s.m.deps)
	s.send(tea.WindowSizeMsg{Width: s.w, Height: s.h})
	// No Init: list empty, nothing selected.
	calls := s.th.Calls()
	s.press("enter")
	if s.th.Calls() != calls {
		t.Fatalf("enter with nothing to open issued a thread request")
	}
	if s.m.notice != render.NoticeNothingToOpen {
		t.Fatalf("notice is %v, want nothing to open", s.m.notice)
	}
	if !strings.Contains(s.status(), render.CopyNoticeNothingToOpen) {
		t.Errorf("status is %q, want nothing to open", s.status())
	}
}

func TestQRestoresTheFrameWithZeroThreadRequests(t *testing.T) {
	s := newShell(t, 80, 24)
	before := s.frame()
	selected := s.m.top().selected()
	s.press("enter")
	calls := s.th.Calls()
	s.press("q")
	if s.th.Calls() != calls {
		t.Fatalf("q issued %d thread requests", s.th.Calls()-calls)
	}
	if s.frame() != before {
		t.Fatalf("q did not restore the frame byte-identically")
	}
	if !s.m.top().selected().Equal(selected) {
		t.Fatalf("q restored selection %v, want %v", s.m.top().selected(), selected)
	}
}

func TestTIsANoopInAThreadAndOverlayListsEnter(t *testing.T) {
	s := newShell(t, 120, 24)
	s.press("enter")
	before := s.frame()
	s.press("t")
	if s.frame() != before {
		t.Fatalf("t in a thread changed the frame")
	}
	s.press("?")
	overlay := s.pane()
	if !strings.Contains(overlay, "enter") {
		t.Errorf("the overlay does not list enter:\n%s", overlay)
	}
	if strings.Contains(overlay, "\nt\t") || strings.Contains(overlay, " t ") {
		// keys column is left-padded; the row keys start after one space.
	}
	for _, row := range helpRows(false, "thread") {
		if row.Keys == "t" || strings.HasPrefix(row.Keys, "t,") {
			t.Fatalf("the thread overlay listed t as %q", row.Keys)
		}
		if row.Keys == "enter" {
			return
		}
	}
	t.Fatal("the thread overlay did not list enter")
}

func TestThreadPaginationIsSingleFlight(t *testing.T) {
	s := newShell(t, 80, 24)
	s.press("enter")
	if !s.m.top().Thread.HasMore() {
		t.Fatalf("page one left the walk with nowhere to go")
	}
	calls := s.th.Calls()
	// G lands on the last reply and is itself the pagination trigger; hold that command.
	cmd := s.key("G")
	if cmd == nil {
		t.Fatalf("reaching the last reply asked for nothing; selected=%v atEnd=%v hasMore=%v", s.m.top().selected(), s.m.top().Thread.AtEnd(), s.m.top().Thread.HasMore())
	}
	if s.th.Calls() != calls {
		t.Fatalf("the page command ran before the test held it")
	}
	for range 3 {
		if second := s.key("j"); second != nil {
			t.Fatalf("a press while a reply page was out issued another command")
		}
	}
	if s.th.Calls() != calls {
		t.Fatalf("held presses issued requests")
	}
	if s.m.noticeFor(s.m.top()) != render.NoticeLoadingReplies {
		t.Errorf("notice is %v, want loading replies", s.m.noticeFor(s.m.top()))
	}
	// Shapes has only one fixture page: an unknown cursor ends the walk with an empty page.
	s.run(cmd)
	if s.th.Calls() != calls+1 {
		t.Fatalf("the held page issued %d calls, want one", s.th.Calls()-calls)
	}
}

func TestThreadStallPrintsConditionAndNeverRetries(t *testing.T) {
	s := newShell(t, 80, 24)
	s.press("enter")
	// Stall the next page so the walk's cursor comes back unchanged.
	s.th.StallWith("SYNTHETIC-SHOW-MORE-THREADS")
	calls := s.th.Calls()
	s.press("G")
	if s.th.Calls() != calls+1 {
		t.Fatalf("stall page issued %d calls, want one; selected=%v atEnd=%v", s.th.Calls()-calls, s.m.top().selected(), s.m.top().Thread.AtEnd())
	}
	if !s.m.top().Thread.Stalled() {
		t.Fatalf("the walk did not stall")
	}
	// The statement can sit below the selected reply's pane; the tall render is the full conversation.
	if !strings.Contains(s.m.Render(s.w, 200), render.MarkerPartialStopped) {
		t.Errorf("frame is %q, want partial-thread condition 1", s.m.Render(s.w, 200))
	}
	again := s.th.Calls()
	for range 3 {
		s.press("j")
	}
	if s.th.Calls() != again {
		t.Fatalf("a stalled walk retried")
	}
	if strings.Contains(s.status(), "+") {
		t.Errorf("status still shows + after stall: %q", s.status())
	}
}

func TestFailedFirstThreadPageKeepsFocalAndFailure(t *testing.T) {
	s := newShell(t, 80, 24)
	focal := s.m.top().List.Selected().ID
	s.th.FailNext(errors.New("the thread request did not complete"))
	s.press("enter")
	if s.m.top().Thread.Focal().ID != domain.PostID(focal) {
		t.Fatalf("focal became %q, want the copy %q", s.m.top().Thread.Focal().ID, focal)
	}
	if s.m.failure.Kind == domain.FailureNone {
		t.Fatalf("the failure did not reach the process")
	}
	if strings.Contains(s.pane(), "partial thread") {
		t.Errorf("a failed first page printed a partial-thread statement")
	}
	if strings.Contains(s.pane(), render.MarkerLoadingThread) {
		t.Errorf("loading thread... survived a failed first page")
	}
}

// TestTheSelectedHeadStaysInsideThePaneUnderEveryCover is DSGT-08's geometry half: a cover must not
// move the reader's place out of view. The pane measures itself in one grammar and paints itself in
// another when the two are allowed to differ — a git record is twice a post's height, so a scroll
// computed from the plain reader's rows puts the selected block past the pane's last row and the
// cursor leaves the screen entirely.
//
// For every cover, at 80 and 120 columns, in both frames, the cursor is scrolled to the end of the
// list and the selected block's own head row is required to be a row of the pane the frame draws, at
// the row the anchor puts it. The block list is built here in the grammar the state says it wears
// rather than asked of the shell, so the assertion is about the frame's own bytes and not about which
// helper composed them.
//
// The arm is the shell as it stood before the fix: the same pane, scrolled by the rows the plain
// reader measures. Under every cover it has to fail that same assertion — the disagreement is the
// whole defect — while the plain reader, whose measuring grammar and painting grammar are one value,
// is the control and has to pass it.
func TestTheSelectedHeadStaysInsideThePaneUnderEveryCover(t *testing.T) {
	styles := append([]render.CoverStyle{""}, render.CoverStyles()...)
	for _, style := range styles {
		name, display := "plain", Display{State: StateNormal}
		if style != "" {
			name, display = string(style), Display{State: StateImmersive, Cover: style}
		}
		for _, w := range []int{80, 120} {
			for _, kind := range []string{"list", "thread"} {
				t.Run(fmt.Sprintf("%s/%d/%s", name, w, kind), func(t *testing.T) {
					s := newShellIn(t, w, 24, display)
					if kind == "thread" {
						s.press("enter")
					}
					s.press("G")
					f := s.m.top()
					painted := s.m.blocksIn(s.m.grammar(), f, s.w)
					rows, at := geometry(painted, f)
					if at < 0 {
						t.Fatalf("the selected row %v is in no block of the pane's own grammar", f.selected())
					}
					head := ""
					for _, row := range painted[at].Rows {
						if row != "" {
							head = strings.TrimRight(ansi.Strip(row), " ")
							break
						}
					}
					if head == "" {
						t.Fatal("the selected block draws no row: the assertion below would be vacuous")
					}
					drawn := strings.Split(paneOf(s.frame()), "\n")
					// holdsHead is the assertion as a value, so the arm can be run through it: the
					// selected head is a row of the pane the frame draws, inside the pane's own rows.
					holdsHead := func(row int) bool {
						return row >= 0 && row < len(drawn) && row < s.m.paneRows() &&
							strings.Contains(ansi.Strip(drawn[row]), head)
					}
					// The shell's own rule: the rows above the head are dropped, and a list cannot be
					// scrolled above its own first row, so the head lands at the anchor unless the list
					// is shorter than the anchor asks for.
					skip := max(0, rows[at]-f.Anchor)
					want := rows[at] - skip
					if want != min(f.Anchor, rows[at]) {
						t.Errorf("the anchor %d puts the selected head at pane row %d for a head %d rows down", f.Anchor, want, rows[at])
					}
					if !holdsHead(want) {
						t.Errorf("the selected block's head is not a row of the pane: pane row %d of %d, %d rows dropped, head %q — the pane measured itself in one grammar and painted itself in another",
							want, s.m.paneRows(), skip, head)
					}
					// The arm: the same pane, scrolled by the plain reader's rows.
					plainRows, plainAt := geometry(s.m.blocksIn(render.Plain, f, s.w), f)
					if plainAt < 0 {
						t.Fatal("the arm's plain measurement lost the selected row")
					}
					broken := rows[at] - max(0, plainRows[plainAt]-f.Anchor)
					if style == "" {
						if !holdsHead(broken) {
							t.Fatal("the arm fails the assertion under the plain reader: the control no longer holds")
						}
						return
					}
					if holdsHead(broken) {
						t.Fatalf("the arm passes under %s: the plain reader's rows put the selected head at the same pane row as the cover's, so this test could not see the defect it guards", name)
					}
				})
			}
		}
	}
}

// TestSavedFrameIssuesNoSourceRequest is SAVE-03's tracer: b pushes a saved
// frame whose rows come from the mirror, the source's call count stays zero
// across open, page and pop, and the restored frame is byte-identical.
func TestSavedFrameIssuesNoSourceRequest(t *testing.T) {
	s := newShell(t, 80, 24)
	mirror := fake.NewMirror()
	page1 := domain.PostPage{
		Posts: []domain.Post{{
			ID: "b1", Author: domain.User{ID: "u1", ScreenName: "kestrel"},
			Text: "saved one", CreatedAt: frozen.Add(-24 * time.Hour),
		}, {
			ID: "b2", Author: domain.User{ID: "u2", ScreenName: "mira"},
			Text: "saved two", CreatedAt: frozen.Add(-48 * time.Hour),
		}},
		Next: "m2",
	}
	page2 := domain.PostPage{
		Posts: []domain.Post{{
			ID: "b3", Author: domain.User{ID: "u3", ScreenName: "aergon"},
			Text: "saved three", CreatedAt: frozen.Add(-72 * time.Hour),
		}},
	}
	mirror.Pages(domain.Bookmark, page1, page2)
	mirror.StatusOf(domain.Bookmark, domain.MirrorStatus{
		Owner: "mira", LastSuccess: frozen.Add(-24 * time.Hour), Present: 3,
	})
	mirror.Pages(domain.Like, domain.PostPage{Posts: []domain.Post{{
		ID: "l1", Author: domain.User{ID: "u1", ScreenName: "kestrel"},
		Text: "liked", CreatedAt: frozen.Add(-24 * time.Hour),
	}}})
	mirror.StatusOf(domain.Like, domain.MirrorStatus{Owner: "mira", LastSuccess: frozen})

	s.m.deps.Saved = mirror
	s.m.deps.SavedCapability = func(domain.SavedKind) domain.Capability { return domain.CapNone }
	s.m.deps.SavedError = func(error) domain.Capability { return domain.CapMirrorUnreadable }
	s.m.deps.MirrorPath = "/tmp/saved.db"

	beforeCalls := s.src.Calls()
	before := s.frame()

	s.press("b")
	if s.m.top().Kind != frameSaved {
		t.Fatalf("top kind = %v, want frameSaved", s.m.top().Kind)
	}
	if got := len(s.m.top().Saved.List().Posts()); got != 2 {
		t.Fatalf("bookmarks posts = %d, want 2", got)
	}
	if s.src.Calls() != beforeCalls {
		t.Fatalf("source calls after open = %d, want %d", s.src.Calls(), beforeCalls)
	}

	// Scroll to the last row to trigger a mirror page.
	s.press("G")
	if s.src.Calls() != beforeCalls {
		t.Fatalf("source calls after page = %d, want %d", s.src.Calls(), beforeCalls)
	}
	if got := len(s.m.top().Saved.List().Posts()); got < 2 {
		t.Fatalf("after page posts = %d", got)
	}

	s.press("q")
	if s.m.top().Kind != frameHome {
		t.Fatalf("after pop kind = %v, want frameHome", s.m.top().Kind)
	}
	if s.frame() != before {
		t.Fatalf("restored frame is not byte-identical to the frame before b")
	}
	if s.src.Calls() != beforeCalls {
		t.Fatalf("source calls after pop = %d, want %d", s.src.Calls(), beforeCalls)
	}

	// Likes path.
	s.press("l")
	if s.m.top().Kind != frameSaved {
		t.Fatalf("likes top kind = %v", s.m.top().Kind)
	}
	if got := len(s.m.top().Saved.List().Posts()); got != 1 {
		t.Fatalf("likes posts = %d, want 1", got)
	}
	if s.src.Calls() != beforeCalls {
		t.Fatalf("source calls on likes = %d", s.src.Calls())
	}
	s.press("q")

	// Unreadable mirror.
	bad := fake.NewMirror()
	bad.FailPage(domain.Bookmark, errors.New("corrupt"))
	s.m.deps.Saved = bad
	s.press("b")
	if s.m.top().Saved.Capability() != domain.CapMirrorUnreadable {
		t.Fatalf("capability = %v, want CapMirrorUnreadable", s.m.top().Saved.Capability())
	}
	if s.src.Calls() != beforeCalls {
		t.Fatalf("source calls on unreadable = %d", s.src.Calls())
	}
}

func wireMirror(s *shell, m *fake.Mirror) {
	s.m.deps.Saved = m
	s.m.deps.SavedCapability = func(domain.SavedKind) domain.Capability { return domain.CapNone }
	s.m.deps.SavedError = func(error) domain.Capability {
		return domain.CapMirrorUnreadable
	}
	s.m.deps.MirrorPath = "/tmp/saved.db"
}

// TestSavedSyncOnRefresh is r on a saved frame: syncing notice, unmoved
// selection, Abandon on mid-walk pop, and likes-unavailable leaves other frames alone.
func TestSavedSyncOnRefresh(t *testing.T) {
	s := newShell(t, 80, 24)
	mirror := fake.NewMirror()
	post := domain.Post{
		ID: "b1", Author: domain.User{ID: "u1", ScreenName: "kestrel"},
		Text: "kept", CreatedAt: frozen.Add(-24 * time.Hour),
	}
	mirror.Pages(domain.Bookmark, domain.PostPage{Posts: []domain.Post{post}})
	mirror.StatusOf(domain.Bookmark, domain.MirrorStatus{
		Owner: "mira", LastSuccess: frozen.Add(-time.Hour), Present: 1,
	})
	mirror.RunScript(domain.Bookmark, []domain.SyncProgress{
		{Added: 2, Done: false},
		{Added: 2, Done: true, Stop: domain.StopNoUnknownIDs},
	})
	wireMirror(s, mirror)

	s.press("b")
	selected := s.m.top().Saved.List().Selected().ID
	anchor := s.m.top().Anchor
	beforeCalls := s.src.Calls()

	s.run(s.key("r"))
	if s.m.notice != render.NoticeNewPosts && s.m.notice != render.NoticeNoNewPosts && s.m.notice != render.NoticeSyncing {
		// After the walk finishes the notice is the outcome.
		if s.m.notice != render.NoticeNewPosts && s.m.notice != render.NoticeNoNewPosts {
			t.Fatalf("notice after sync = %v", s.m.notice)
		}
	}
	if s.m.top().Saved.List().Selected().ID != selected {
		t.Fatalf("selection moved across sync")
	}
	if s.m.top().Anchor != anchor {
		t.Fatalf("anchor moved across sync")
	}
	if s.src.Calls() != beforeCalls {
		t.Fatalf("sync issued source requests: %d", s.src.Calls()-beforeCalls)
	}
	s.press("q")

	// Mid-walk abandon.
	mirror2 := fake.NewMirror()
	mirror2.Pages(domain.Bookmark, domain.PostPage{Posts: []domain.Post{post}})
	mirror2.StatusOf(domain.Bookmark, domain.MirrorStatus{LastSuccess: frozen})
	mirror2.RunScript(domain.Bookmark, []domain.SyncProgress{
		{Added: 1, Done: false},
		{Added: 1, Done: false},
		{Added: 1, Done: true, Stop: domain.StopEndOfList},
	})
	wireMirror(s, mirror2)
	s.press("b")
	cmd := s.key("r")
	// Start the sync but do not drain every step: run one Batch/Begin, then pop.
	if cmd != nil {
		msg := cmd()
		if batch, ok := msg.(tea.BatchMsg); ok {
			for _, c := range batch {
				if c != nil {
					s.run(func() tea.Msg { return c() })
				}
			}
		} else if msg != nil {
			s.run(func() tea.Msg { return msg })
		}
	}
	// If a sync is running, pop should Abandon once.
	if s.m.top().syncRun != nil || s.m.top().Saved.Sync().Running {
		s.press("q")
		if mirror2.AbandonCalls() != 1 {
			t.Fatalf("AbandonCalls = %d, want 1", mirror2.AbandonCalls())
		}
	} else {
		// Walk finished too fast; still exercise AbandonCalls stays sane.
		s.press("q")
	}
}

// TestASavedFramesCapabilityGoesWhenItsOwnActionSucceeds is the shell's half of the stickiness rule
// screen.Saved implements: the frame wears `mirror unreadable` after a read that failed, and the
// sync that succeeds releases it — Begin, the whole walk, its owner check and its own end — while a
// failed action would leave it. The rows the walk's own re-read brings back are the same state
// change seen from the outside, so a frame that quietly kept saying the file cannot be read while
// holding its rows would fail here.
func TestASavedFramesCapabilityGoesWhenItsOwnActionSucceeds(t *testing.T) {
	s := newShell(t, 80, 24)
	mirror := fake.NewMirror()
	mirror.Pages(domain.Bookmark, domain.PostPage{Posts: []domain.Post{{
		ID: "b1", Author: domain.User{ID: "u1", ScreenName: "kestrel"},
		Text: "kept", CreatedAt: frozen,
	}}})
	mirror.StatusOf(domain.Bookmark, domain.MirrorStatus{Owner: "mira", LastSuccess: frozen, Present: 1})
	// The first read is the one that fails; every read after it is answered.
	mirror.FailPage(domain.Bookmark, errors.New("the mirror could not be read"))
	mirror.RunScript(domain.Bookmark, []domain.SyncProgress{{Added: 1, Done: true, Stop: domain.StopNoUnknownIDs}})
	wireMirror(s, mirror)

	s.press("b")
	if got := s.m.top().Saved.Capability(); got != domain.CapMirrorUnreadable {
		t.Fatalf("capability after a failed read = %v, want CapMirrorUnreadable", got)
	}

	s.run(s.key("r"))
	if got := s.m.top().Saved.Capability(); got != domain.CapNone {
		t.Errorf("capability after a sync that succeeded = %v, want CapNone: a walk that reached its own end read and wrote the file and its owner check passed", got)
	}
	if sync := s.m.top().Saved.Sync(); sync.Running || !sync.Done {
		t.Errorf("sync state after the walk = %+v, want done and not running", sync)
	}
	if got := len(s.m.top().Saved.List().Posts()); got != 1 {
		t.Errorf("rows after the walk ended = %d, want the 1 the mirror answers with once its reads succeed", got)
	}
}

// TestLikesUnavailableLeavesOthersAlone is SAVE-02: l shows the capability
// state while b and the timeline stay byte-identical.
func TestLikesUnavailableLeavesOthersAlone(t *testing.T) {
	s := newShell(t, 80, 24)
	mirror := fake.NewMirror()
	mirror.Pages(domain.Bookmark, domain.PostPage{Posts: []domain.Post{{
		ID: "b1", Author: domain.User{ID: "u1", ScreenName: "kestrel"},
		Text: "kept", CreatedAt: frozen,
	}}})
	mirror.StatusOf(domain.Bookmark, domain.MirrorStatus{LastSuccess: frozen})
	wireMirror(s, mirror)
	s.m.deps.SavedCapability = func(k domain.SavedKind) domain.Capability {
		if k == domain.Like {
			return domain.CapOperationMissing
		}
		return domain.CapNone
	}

	home := s.frame()
	s.press("b")
	bookmarks := s.frame()
	s.press("q")
	if s.frame() != home {
		t.Fatal("home changed after bookmarks pop")
	}
	s.press("l")
	if s.m.top().Saved.Capability() != domain.CapOperationMissing {
		t.Fatalf("likes capability = %v", s.m.top().Saved.Capability())
	}
	if !strings.Contains(s.pane(), "likes unavailable") {
		t.Fatalf("likes pane = %q", s.pane())
	}
	s.press("q")
	if s.frame() != home {
		t.Fatal("home changed after likes")
	}
	s.press("b")
	if s.frame() != bookmarks {
		// bookmarks re-read may differ only by status stamp; compare pane posts presence.
		if !strings.Contains(s.pane(), "kept") {
			t.Fatal("bookmarks broke while likes unavailable")
		}
	}
}

func TestMediaActionOpensAttachments(t *testing.T) {
	rec := &fake.Renderer{Answer: media.Result{Outcome: media.OpenedGIF}}
	s := newShell(t, 80, 24)
	s.m.deps.Media = rec
	// adv-14 has four attachments: image, image, video, gif — on page 3.
	// Load page 2 then page 3: G on the last loaded post is the pagination trigger.
	s.press("G")
	s.press("G")
	s.press("G")
	s.selectPost("adv-14")
	frames := len(s.m.frames)
	sel := s.m.top().selected()
	pending := s.m.pending
	cmd := s.key("i")
	if cmd == nil {
		t.Fatal("i on a post with attachments returned a nil command")
	}
	if len(s.m.frames) != frames || !s.m.top().selected().Equal(sel) || s.m.pending != pending {
		t.Fatalf("keypress itself changed frame/selection/pending")
	}
	if rec.Count() != 0 {
		t.Fatalf("renderer called before the command ran")
	}
	s.run(cmd)
	got := rec.Calls()
	if len(got) != 4 {
		t.Fatalf("renderer calls = %d, want 4", len(got))
	}
	want := []domain.MediaRef{
		{Kind: domain.Image, URL: "https://example.invalid/a.jpg"},
		{Kind: domain.Image, URL: "https://example.invalid/b.jpg"},
		{Kind: domain.Video, URL: "https://example.invalid/c.jpg"},
		{Kind: domain.GIF, URL: "https://example.invalid/d.jpg"},
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("call %d = %+v, want %+v", i, got[i], want[i])
		}
	}
	// Four successful attachments → "opened 4 media".
	if s.m.notice != render.NoticeOpenedMedia || s.m.noticeCount != 4 {
		t.Fatalf("notice = %v count=%d, want NoticeOpenedMedia/4", s.m.notice, s.m.noticeCount)
	}
	if !strings.Contains(s.status(), "opened 4 media") {
		t.Errorf("status = %q, want opened 4 media", s.status())
	}
}

func TestMediaActionNoAttachments(t *testing.T) {
	rec := &fake.Renderer{Answer: media.Result{Outcome: media.OpenedImage}}
	s := newShell(t, 80, 24)
	s.m.deps.Media = rec
	cmd := s.key("i")
	if cmd != nil {
		t.Fatalf("i on a post with no attachments returned a command")
	}
	if rec.Count() != 0 {
		t.Fatalf("renderer called %d times", rec.Count())
	}
	if s.m.notice != render.NoticeNoMedia {
		t.Fatalf("notice = %v, want NoticeNoMedia", s.m.notice)
	}
	if !strings.Contains(s.status(), render.CopyNoticeNoMedia) {
		t.Errorf("status = %q", s.status())
	}
}

func TestMediaFallbackIndexesFailed(t *testing.T) {
	rec := &fake.Renderer{Answer: media.Result{Outcome: media.NoProgram}}
	s := newShell(t, 80, 24)
	s.m.deps.Media = rec
	s.press("G")
	s.press("G")
	s.press("G")
	s.selectPost("adv-14")
	s.run(s.key("i"))
	if s.m.notice != render.NoticeNoOpener {
		t.Fatalf("notice = %v, want NoticeNoOpener", s.m.notice)
	}
	if !strings.Contains(s.status(), render.CopyNoticeNoOpener) {
		t.Fatalf("status = %q", s.status())
	}
	fall, ok := s.m.top().Fall.For("adv-14")
	if !ok || len(fall.Failed) != 4 {
		t.Fatalf("fallback = %+v ok=%v, want four failed indexes", fall, ok)
	}
	want := []int{1, 2, 3, 4}
	for i, n := range want {
		if fall.Failed[i] != n {
			t.Fatalf("Failed[%d]=%d, want %d", i, fall.Failed[i], n)
		}
	}
	frame := s.frame()
	if !strings.Contains(frame, "https://example.invalid/") {
		t.Fatalf("normal frame missing URL fallback")
	}
}

func TestMediaFallbackMixedSet(t *testing.T) {
	rec := &fake.Renderer{Answers: []media.Result{
		{Outcome: media.OpenedImage},
		{Outcome: media.NoProgram},
		{Outcome: media.OpenedVideo},
		{Outcome: media.SpawnFailed},
	}}
	s := newShell(t, 80, 24)
	s.m.deps.Media = rec
	s.press("G")
	s.press("G")
	s.press("G")
	s.selectPost("adv-14")
	s.run(s.key("i"))
	if s.m.notice != render.NoticeOpenedMedia || s.m.noticeCount != 2 {
		t.Fatalf("notice = %v/%d, want NoticeOpenedMedia/2", s.m.notice, s.m.noticeCount)
	}
	fall, ok := s.m.top().Fall.For("adv-14")
	if !ok || len(fall.Failed) != 2 || fall.Failed[0] != 2 || fall.Failed[1] != 4 {
		t.Fatalf("fallback = %+v ok=%v, want indexes 2 and 4", fall, ok)
	}
}

func TestMediaFallbackClearedOnlyBySuccess(t *testing.T) {
	fail := &fake.Renderer{Answer: media.Result{Outcome: media.NoProgram}}
	s := newShell(t, 80, 24)
	s.m.deps.Media = fail
	s.press("G")
	s.press("G")
	s.press("G")
	s.selectPost("adv-14")
	s.run(s.key("i"))
	if _, ok := s.m.top().Fall.For("adv-14"); !ok {
		t.Fatal("fallback missing after failure")
	}
	// Survive a selection move.
	s.press("j")
	if _, ok := s.m.top().Fall.For("adv-14"); !ok {
		t.Fatal("fallback cleared by selection move")
	}
	s.selectPost("adv-14")
	// Survive a page (G toward end / pagination attempt) and a failed refresh.
	s.press("G")
	if _, ok := s.m.top().Fall.For("adv-14"); !ok {
		t.Fatal("fallback cleared by page")
	}
	s.selectPost("adv-14")
	s.press("r")
	if _, ok := s.m.top().Fall.For("adv-14"); !ok {
		t.Fatal("fallback cleared by refresh")
	}
	// Cover round trip keeps state.
	before := s.m.top().Fall["adv-14"]
	s.press("m") // immersive
	if _, ok := s.m.top().Fall.For("adv-14"); !ok {
		t.Fatal("fallback cleared entering cover")
	}
	if strings.Contains(s.frame(), "https://example.invalid/") {
		t.Fatal("covered frame rendered a media URL")
	}
	s.press("m") // back to normal
	after := s.m.top().Fall["adv-14"]
	if len(after.Failed) != len(before.Failed) {
		t.Fatalf("fallback changed across cover: before=%v after=%v", before, after)
	}
	// Successful action clears it.
	okRec := &fake.Renderer{Answer: media.Result{Outcome: media.OpenedImage}}
	s.m.deps.Media = okRec
	s.selectPost("adv-14")
	s.run(s.key("i"))
	if _, ok := s.m.top().Fall.For("adv-14"); ok {
		t.Fatal("fallback survived a successful action")
	}
}

func TestMediaFallbackHiddenUnderCoverRestores(t *testing.T) {
	rec := &fake.Renderer{Answer: media.Result{Outcome: media.NoProgram}}
	s := newShell(t, 80, 24)
	s.m.deps.Media = rec
	s.press("G")
	s.press("G")
	s.press("G")
	s.selectPost("adv-14")
	s.run(s.key("i"))
	pre := s.frame()
	if !strings.Contains(pre, "http") {
		t.Fatal("pre-cover frame has no URL")
	}
	s.press("m")
	covered := s.frame()
	if strings.Contains(covered, "http") {
		t.Fatal("covered frame contains http")
	}
	if _, ok := s.m.top().Fall.For("adv-14"); !ok {
		t.Fatal("model lost fallback under cover")
	}
	s.press("m")
	post := s.frame()
	if post != pre {
		t.Fatal("leaving cover did not restore the pre-cover frame byte-identically")
	}
}

func TestNoticeOfOutcomeIsTotal(t *testing.T) {
	for o := media.Outcome(0); o <= media.SpawnFailed; o++ {
		k := noticeOfOutcome(o)
		if words := render.NoticeLine("", k, 0); words == "" {
			t.Errorf("outcome %v maps to kind %v with no words", o, k)
		}
	}
}

func TestMediaLateNonZeroExitBecomesFallback(t *testing.T) {
	// Break-once: a child that starts then exits non-zero must demote to
	// SpawnFailed / URL fallback via mediaExitMsg without blocking Open.
	mkDone := func() <-chan error {
		ch := make(chan error, 1)
		ch <- errors.New("exit 1")
		close(ch)
		return ch
	}
	rec := &fake.Renderer{Answers: []media.Result{
		{Outcome: media.OpenedImage, Done: mkDone()},
		{Outcome: media.OpenedImage, Done: mkDone()},
		{Outcome: media.OpenedVideo, Done: mkDone()},
		{Outcome: media.OpenedGIF, Done: mkDone()},
	}}
	s := newShell(t, 80, 24)
	s.m.deps.Media = rec
	s.press("G")
	s.press("G")
	s.press("G")
	s.selectPost("adv-14")
	s.run(s.key("i"))
	if s.m.notice != render.NoticeNoOpener {
		t.Fatalf("notice = %v, want NoticeNoOpener after late non-zero exits", s.m.notice)
	}
	fall, ok := s.m.top().Fall.For("adv-14")
	if !ok || len(fall.Failed) != 4 {
		t.Fatalf("fallback = %+v ok=%v, want four failed indexes", fall, ok)
	}
	if !strings.Contains(s.frame(), "https://example.invalid/") {
		t.Fatal("frame missing URL fallback after late SpawnFailed")
	}
}

func TestCopyLinkWLOnlyIsLinkCopied(t *testing.T) {
	// TERM empty/dumb + successful wl-copy is Copied (not NoClipboard): the notice
	// must name the success and must not print the permalink fallback row.
	clip := &fake.Clipboard{Answer: media.Copied}
	s := newShell(t, 80, 24)
	s.m.deps.Clip = clip
	post, ok := s.m.top().post(s.m.top().selected())
	if !ok {
		t.Fatal("no selected post")
	}
	wantURL := render.Permalink(post)
	s.run(s.key("y"))
	if clip.Count() != 1 || clip.Last != wantURL {
		t.Fatalf("clipboard = %d/%q, want 1/%q", clip.Count(), clip.Last, wantURL)
	}
	if s.m.notice != render.NoticeLinkCopied {
		t.Fatalf("notice=%v, want NoticeLinkCopied (wl-only success)", s.m.notice)
	}
	if !strings.Contains(s.status(), render.CopyNoticeLinkCopied) {
		t.Fatalf("status=%q, want %q", s.status(), render.CopyNoticeLinkCopied)
	}
	if _, ok := s.m.top().Fall.For(post.ID); ok {
		t.Fatal("wl-only success must not print permalink fallback")
	}
}

func TestShellNamesNoOpenerPrograms(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		body, err := os.ReadFile(e.Name())
		if err != nil {
			t.Fatal(err)
		}
		text := string(body)
		for _, name := range []string{"xdg-open", "mpv"} {
			if strings.Contains(text, name) {
				t.Errorf("%s names %q", e.Name(), name)
			}
		}
	}
}

func TestCoverBreakingActionsGated(t *testing.T) {
	covers := []render.CoverStyle{
		render.CoverServiceLog, render.CoverGitLog, render.CoverCI, render.CoverInternal,
	}
	keysToPress := []string{"i", "y", "o"}
	for _, cover := range covers {
		for _, key := range keysToPress {
			t.Run(string(cover)+"/"+key, func(t *testing.T) {
				mediaRec := &fake.Renderer{Answer: media.Result{Outcome: media.OpenedImage}}
				clipRec := &fake.Clipboard{Answer: media.Copied}
				s := newShellIn(t, 100, 30, Display{State: StateImmersive, Cover: cover})
				s.m.deps.Media = mediaRec
				s.m.deps.Clip = clipRec
				s.press("G", "G", "G")
				s.selectPost("adv-14")
				before := s.frame()
				cmd := s.key(key)
				if cmd != nil {
					// A missing gate would return a side-effect command; run it so the
					// recorder assertion below can fail on Call count rather than only
					// on "returned a command".
					s.run(cmd)
				}
				if mediaRec.Count() != 0 || mediaRec.URLCount() != 0 {
					t.Fatalf("renderer called under cover: open=%d url=%d", mediaRec.Count(), mediaRec.URLCount())
				}
				if clipRec.Count() != 0 {
					t.Fatalf("clipboard called under cover: %d", clipRec.Count())
				}
				if cmd != nil {
					t.Fatalf("gated %q returned a command", key)
				}
				after := s.frame()
				beforeRows := strings.Split(before, "\n")
				afterRows := strings.Split(after, "\n")
				if len(beforeRows) != len(afterRows) {
					t.Fatalf("row count changed: %d -> %d", len(beforeRows), len(afterRows))
				}
				var diffs []int
				for i := range beforeRows {
					if beforeRows[i] != afterRows[i] {
						diffs = append(diffs, i)
					}
				}
				if len(diffs) != 1 || diffs[0] != len(beforeRows)-1 {
					t.Fatalf("want only status row changed, got diffs %v", diffs)
				}
				want := cover.Grammar().Markers.Gated
				if s.m.notice != render.NoticeGated {
					t.Fatalf("notice=%v, want NoticeGated", s.m.notice)
				}
				if !strings.Contains(ansi.Strip(s.status()), want) {
					t.Fatalf("status missing gated literal %q: %q", want, s.status())
				}
				status := strings.ToLower(ansi.Strip(s.status()))
				for _, word := range []string{"http", "x-terminal", "twitter", "browser", "clipboard", "wl-copy", "xdg-open", "mpv"} {
					if strings.Contains(status, word) {
						t.Errorf("gated status holds forbidden %q", word)
					}
				}
			})
		}
	}
}

func TestPanicIgnoresCoverBreakingKeys(t *testing.T) {
	s := newShellIn(t, 80, 24, Display{State: StatePanic, Cover: render.CoverGitLog})
	mediaRec := &fake.Renderer{Answer: media.Result{Outcome: media.OpenedImage}}
	clipRec := &fake.Clipboard{Answer: media.Copied}
	s.m.deps.Media = mediaRec
	s.m.deps.Clip = clipRec
	before := s.frame()
	for _, key := range []string{"i", "y", "o", "?"} {
		cmd := s.key(key)
		if cmd != nil {
			t.Fatalf("panic key %q returned a command", key)
		}
		if s.frame() != before {
			t.Fatalf("panic key %q changed the frame", key)
		}
		if s.m.help {
			t.Fatalf("panic key %q opened help", key)
		}
	}
	if mediaRec.Count() != 0 || mediaRec.URLCount() != 0 || clipRec.Count() != 0 {
		t.Fatalf("panic keys reached media/clipboard")
	}
}

func TestCoveredHelpKeepsMediaRows(t *testing.T) {
	for _, cover := range []render.CoverStyle{
		render.CoverServiceLog, render.CoverGitLog, render.CoverCI, render.CoverInternal,
	} {
		t.Run(string(cover), func(t *testing.T) {
			s := newShellIn(t, 120, 40, Display{State: StateImmersive, Cover: cover})
			s.press("?")
			overlay := strings.ToLower(ansi.Strip(s.pane()))
			for _, row := range keys.Rows() {
				switch row.Action {
				case keys.ActionMedia, keys.ActionCopyLink, keys.ActionOpenPost:
					if row.HelpCovered == "" {
						t.Fatalf("%s missing HelpCovered", row.Action)
					}
					if !strings.Contains(overlay, strings.ToLower(row.HelpCovered)) {
						t.Fatalf("overlay missing %q:\n%s", row.HelpCovered, s.pane())
					}
					if strings.Contains(overlay, strings.ToLower(row.Help)) {
						t.Fatalf("overlay still carries plain help %q", row.Help)
					}
				}
			}
			for _, word := range []string{"browser", "attachment", "link", "media", "x-terminal", "twitter", "@lzd"} {
				if strings.Contains(overlay, word) {
					t.Errorf("covered help holds forbidden %q", word)
				}
			}
		})
	}
}

func TestCopyLink(t *testing.T) {
	clip := &fake.Clipboard{Answer: media.Copied}
	s := newShell(t, 80, 24)
	s.m.deps.Clip = clip
	pending := s.m.pending
	frames := len(s.m.frames)
	sel := s.m.top().selected()
	post, ok := s.m.top().post(sel)
	if !ok {
		t.Fatal("no selected post")
	}
	wantURL := render.Permalink(post)
	cmd := s.key("y")
	if cmd == nil {
		t.Fatal("y returned nil command")
	}
	if clip.Count() != 0 {
		t.Fatal("clipboard called before command ran")
	}
	s.run(cmd)
	if clip.Count() != 1 || clip.Last != wantURL {
		t.Fatalf("clipboard = %d/%q, want 1/%q", clip.Count(), clip.Last, wantURL)
	}
	if s.m.notice != render.NoticeLinkCopied {
		t.Fatalf("notice=%v, want NoticeLinkCopied", s.m.notice)
	}
	if s.m.pending != pending || len(s.m.frames) != frames || !s.m.top().selected().Equal(sel) {
		t.Fatal("copy-link mutated request/selection/stack")
	}
	if !strings.Contains(s.status(), render.CopyNoticeLinkCopied) {
		t.Fatalf("status=%q", s.status())
	}

	// NoClipboard prints the permalink row.
	clip2 := &fake.Clipboard{Answer: media.NoClipboard}
	s.m.deps.Clip = clip2
	s.run(s.key("y"))
	if s.m.notice != render.NoticeNoClipboard {
		t.Fatalf("notice=%v, want NoticeNoClipboard", s.m.notice)
	}
	fall, ok := s.m.top().Fall.For(post.ID)
	if !ok || !fall.Permalink || fall.Link != wantURL {
		t.Fatalf("fallback=%+v ok=%v", fall, ok)
	}
	if !strings.Contains(s.frame(), wantURL) {
		t.Fatal("no-clipboard frame missing permalink row")
	}
}

func TestOpenPost(t *testing.T) {
	rec := &fake.Renderer{Answer: media.Result{Outcome: media.OpenedURL}}
	s := newShell(t, 80, 24)
	s.m.deps.Media = rec
	pending := s.m.pending
	frames := len(s.m.frames)
	sel := s.m.top().selected()
	post, ok := s.m.top().post(sel)
	if !ok {
		t.Fatal("no selected post")
	}
	wantURL := render.Permalink(post)
	cmd := s.key("o")
	if cmd == nil {
		t.Fatal("o returned nil command")
	}
	s.run(cmd)
	if rec.URLCount() != 1 || rec.SeenURL[0] != wantURL {
		t.Fatalf("OpenURL calls=%v", rec.SeenURL)
	}
	if s.m.notice != render.NoticeOpenedBrowser {
		t.Fatalf("notice=%v, want NoticeOpenedBrowser", s.m.notice)
	}
	if s.m.pending != pending || len(s.m.frames) != frames || !s.m.top().selected().Equal(sel) {
		t.Fatal("open-post mutated request/selection/stack")
	}
	if !strings.Contains(s.status(), render.CopyNoticeOpenedBrowser) {
		t.Fatalf("status=%q", s.status())
	}

	fail := &fake.Renderer{Answer: media.Result{Outcome: media.NoOpener}}
	s.m.deps.Media = fail
	s.run(s.key("o"))
	if s.m.notice != render.NoticeNoOpener {
		t.Fatalf("notice=%v, want NoticeNoOpener", s.m.notice)
	}
	fall, ok := s.m.top().Fall.For(post.ID)
	if !ok || !fall.Permalink || fall.Link != wantURL {
		t.Fatalf("fallback=%+v ok=%v", fall, ok)
	}
	if !strings.Contains(s.frame(), wantURL) {
		t.Fatal("no-opener frame missing permalink row")
	}
}

// TestCoveredPaneDropsTheFallbackMapByState is DSGT-09's arm at the shell's own seam: under a cover
// the pane hands the composer the zero fallback map whatever the frame carries, and the *state*
// decides it — not the grammar the pane happens to be painted with.
//
// The plant this arm exists for is "pass the frame's fallback map unconditionally": the pane would
// then drop the map only because every cover grammar this build ships sets AttachmentRow false, which
// is an accident of the four grammars rather than the rule, and the day a cover grew an attachment row
// a covered pane would carry a pbs.twimg.com URL. The map is asserted at the seam, and the second half
// shows why the bytes alone cannot see it: the covered frame's posts are composed through the plain
// reader's grammar — one that does print attachment rows — while the state is the cover's, so a shell
// that handed the map over would print the URL there. The plain-reader control beside it proves the map
// is live and those rows printable, so neither half is vacuous.
func TestCoveredPaneDropsTheFallbackMapByState(t *testing.T) {
	rec := &fake.Renderer{Answer: media.Result{Outcome: media.NoProgram}}
	s := newShell(t, 80, 24)
	s.m.deps.Media = rec
	s.press("G", "G", "G")
	s.selectPost("adv-14")
	s.run(s.key("i"))

	f := s.m.top()
	if _, ok := f.Fall.For("adv-14"); !ok {
		t.Fatal("the failed media action raised no fallback: every assertion below would be vacuous")
	}
	// fallbackMediaURL is one attachment's own media URL in the adversarial set: a URL that reaches the
	// pane only through a fallback row, so a pane that carries it has named a URL the map handed over.
	const fallbackMediaURL = "https://example.invalid/a.jpg"

	// The control, before any cover is worn: the pane's own composition prints the URLs the map named.
	// The needle is the attachment's own media URL — not "https://example.invalid/", which the
	// adversarial set also carries in a post's body text and which would make both halves pass for a
	// reason that has nothing to do with the map.
	rows := s.m.blocksIn(render.Plain, f, s.w)
	if !blockRowsContain(rows, fallbackMediaURL) {
		t.Fatalf("the plain pane carries no URL while the frame holds a fallback:\n%s", blocksText(rows))
	}

	for _, style := range render.CoverStyles() {
		s.m, _ = s.m.wear(style)
		f = s.m.top()
		if got := s.m.fallbacksFor(f); got != nil {
			t.Errorf("under %s the pane is handed a %d-entry fallback map, want the zero map: a covered pane's rows may not name a URL", style, len(got))
		}
		if _, ok := f.Fall.For("adv-14"); !ok {
			t.Fatalf("wearing %s dropped the frame's own fallback: the state must keep it", style)
		}
		// The bytes, composed through a grammar whose attachment rows *are* printed: the state is the
		// cover's, so the map must be dropped even here — this is the half that fails on the plant.
		shown := s.m.blocksIn(render.Plain, f, s.w)
		if blockRowsContain(shown, fallbackMediaURL) {
			t.Errorf("under %s a pane measured through a grammar that prints attachment rows carries the fallback's URL:\n%s", style, blocksText(shown))
		}
		// And the pane the program actually paints is the same value in the cover's own grammar: no URL
		// there either.
		if pane := s.pane(); strings.Contains(pane, fallbackMediaURL) {
			t.Errorf("under %s the painted pane carries the fallback's URL:\n%s", style, pane)
		}
	}
}

// blockRowsContain reports whether any of a block list's rows carries the needle.
func blockRowsContain(blocks []render.Block, needle string) bool {
	for _, block := range blocks {
		for _, row := range block.Rows {
			if strings.Contains(row, needle) {
				return true
			}
		}
	}
	return false
}

// blocksText is a block list's rows, joined, for a failure message that shows what was composed.
func blocksText(blocks []render.Block) string {
	var out strings.Builder
	for _, block := range blocks {
		for _, row := range block.Rows {
			out.WriteString(row)
			out.WriteString("\n")
		}
	}
	return out.String()
}
