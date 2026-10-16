package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/render"
	"github.com/LZD121/x-terminal/internal/xweb"
)

// The post every arm below selects: the first block of the adversarial set's first page, so the
// pane's opening rule puts it on screen whatever the pane's size, and its body is one row at every
// width this file renders.
const (
	translateFirstPostID   = "adv-0"
	translateFirstPostBody = "a fifteen-cell handle, the widest the column holds"
)

// translatedText is the body the pane is asked to carry after one `T`.
// TestTranslateShowsTheTranslatedBodyInPlace is the whole feature in one arm: one key on the post the
// cursor is on replaces its body with the service's answer, in the same block of the same list, in
// the same frame — no pane is pushed, no list is rewritten and no screen is left.
func TestTranslateShowsTheTranslatedBodyInPlace(t *testing.T) {
	s := newShell(t, 120, 40)
	s.selectPost(translateFirstPostID)

	before := s.pane()
	if !strings.Contains(before, translateFirstPostBody) {
		t.Fatalf("the pane does not show the post's own body before the key:\n%s", before)
	}
	if strings.Contains(before, translatedBody) {
		t.Fatalf("the pane already carries a translation before the key")
	}
	headRow := selectedHeadRow(before)
	blocks := len(s.m.blocksAt(s.m.top(), s.w))
	frames := len(s.m.frames)

	s.press("T")

	after := s.pane()
	if !strings.Contains(after, translatedBody) {
		t.Fatalf("the pane does not carry the translated body after `T`:\n%s", after)
	}
	if strings.Contains(after, translateFirstPostBody) {
		t.Errorf("the original body is still on the pane: the translation did not replace it")
	}
	if got := len(s.m.blocksAt(s.m.top(), s.w)); got != blocks {
		t.Errorf("the frame holds %d blocks after the translation, want the same %d", got, blocks)
	}
	if got := selectedHeadRow(after); got != headRow {
		t.Errorf("the selection mark moved from pane row %d to %d: the block was not replaced in place", headRow, got)
	}
	if got := len(s.m.frames); got != frames {
		t.Errorf("the frame stack grew to %d: a translation must not push a pane", got)
	}

	// One request, for the post the cursor was on, in the language the composition root resolved.
	calls := s.tr.Calls()
	if len(calls) != 1 {
		t.Fatalf("the port was called %d times, want exactly one", len(calls))
	}
	if calls[0].ID != translateFirstPostID || calls[0].Lang != translateLang {
		t.Errorf("the port was asked for %q in %q, want %q in %q", calls[0].ID, calls[0].Lang, translateFirstPostID, translateLang)
	}
	if s.m.notice != render.NoticeTranslated || !strings.Contains(s.status(), render.CopyNoticeTranslated) {
		t.Errorf("notice is %v and the status line is %q, want the translated notice", s.m.notice, s.status())
	}
	if s.m.pending != 0 {
		t.Errorf("pending is %d after the answer, want 0: the answer must release the request", s.m.pending)
	}
}

// TestMovingOffTheTranslatedPostRevertsIt is the selection-scoped half: the translated view is a
// function of the selection and the memory of answers, so stepping off the post reverts it — and the
// post is still on screen, which is what makes the arm more than a frame comparison.
func TestMovingOffTheTranslatedPostRevertsIt(t *testing.T) {
	s := newShell(t, 120, 40)
	s.selectPost(translateFirstPostID)
	s.press("T")
	if !strings.Contains(s.pane(), translatedBody) {
		t.Fatalf("`T` did not show the translation")
	}

	s.press("j")

	pane := s.pane()
	if strings.Contains(pane, translatedBody) {
		t.Errorf("the translated body is still on the pane after moving off the post:\n%s", pane)
	}
	if !strings.Contains(pane, translateFirstPostBody) {
		t.Errorf("the post's own body is not back after moving off it:\n%s", pane)
	}
}

// TestReachingThePostAgainSpendsNothing is the one-press-per-post rule, read off the port's own call
// count: coming back to a post whose answer is remembered re-renders it and issues nothing.
func TestReachingThePostAgainSpendsNothing(t *testing.T) {
	s := newShell(t, 120, 40)
	s.selectPost(translateFirstPostID)
	s.press("T")
	calls := s.tr.Count()
	if calls != 1 {
		t.Fatalf("the port was called %d times for the first `T`, want one", calls)
	}

	s.press("j", "k")

	if !strings.Contains(s.pane(), translatedBody) {
		t.Fatalf("the translation is not back after returning to the post:\n%s", s.pane())
	}
	if got := s.tr.Count(); got != calls {
		t.Errorf("the port was called %d times, want the same %d: a remembered post costs no second request", got, calls)
	}
}

// TestTTranslateTwiceCostsOneRequest: two presses on the same post are one request, and the second
// press changes nothing on screen — it re-announces what is already known.
func TestTTranslateTwiceCostsOneRequest(t *testing.T) {
	s := newShell(t, 120, 40)
	s.selectPost(translateFirstPostID)
	s.press("T")
	after := s.frame()

	s.press("T")

	if got := s.frame(); got != after {
		t.Errorf("the second `T` changed the frame:\n%s\n---\n%s", after, got)
	}
	if got := s.tr.Count(); got != 1 {
		t.Errorf("the port was called %d times for two presses on one post, want one", got)
	}
	if s.m.notice != render.NoticeTranslated {
		t.Errorf("notice is %v after the second press, want the translated notice", s.m.notice)
	}
}

// TestNoTranslationRendersTheNoticeAndNoBody is the honesty rule the endpoint's own measurement
// forced: a `200` carrying no translation is the service answering, so it renders as the notice
// field's own sentence and the pane keeps the post's body — never an empty body presented as a
// translation.
func TestNoTranslationRendersTheNoticeAndNoBody(t *testing.T) {
	s := newShell(t, 120, 40)
	s.tr.Answer = domain.Translation{}
	s.selectPost(translateFirstPostID)

	s.press("T")

	if s.m.notice != render.NoticeNoTranslation || !strings.Contains(s.status(), render.CopyNoticeNoTranslation) {
		t.Errorf("notice is %v and the status line is %q, want the no-translation notice", s.m.notice, s.status())
	}
	pane := s.pane()
	if !strings.Contains(pane, translateFirstPostBody) {
		t.Errorf("the pane does not carry the post's own body: an empty answer must not blank it:\n%s", pane)
	}
	if strings.Contains(pane, translatedBody) {
		t.Errorf("the pane carries a translation the service never gave")
	}
	if strings.TrimSpace(render.NoticeLine("", render.NoticeNoTranslation, 0)) == "" {
		t.Fatalf("the no-translation notice has no words")
	}

	// The answer is remembered as what it was, so a second press spends nothing either.
	s.press("T")
	if got := s.tr.Count(); got != 1 {
		t.Errorf("the port was called %d times, want one: a post with no translation is answered too", got)
	}
	if s.m.notice != render.NoticeNoTranslation {
		t.Errorf("notice is %v after the second press, want the no-translation notice", s.m.notice)
	}
}

// TestARefusalNamesTheStateAndSetsNoNotice: a refusal is not a notice. It reaches the process's
// state field through the mapper, in the classifier's own sentence, and the notice field is left
// empty rather than carrying a weaker second account of the same thing.
func TestARefusalNamesTheStateAndSetsNoNotice(t *testing.T) {
	s := newShell(t, 120, 40)
	// A refusal as the port really returns one: the transport's own classified answer.
	refusal := &xweb.APIError{Operation: "translateTweet", Outcome: xweb.OutcomeUnknownOperation, Status: 404}
	s.tr.FailNext(refusal)
	s.selectPost(translateFirstPostID)

	s.press("T")

	want := testFailure(refusal, nil)
	if s.m.failure != want {
		t.Errorf("failure is %+v, want the mapper's own answer %+v", s.m.failure, want)
	}
	if s.m.failure.Kind == domain.FailureNone {
		t.Fatalf("a refusal left no state on the line")
	}
	if !strings.Contains(want.Detail, xweb.OutcomeUnknownOperation.Remedy()) {
		t.Fatalf("the mapper's detail %q does not carry the classifier's own sentence", want.Detail)
	}
	if s.m.notice != render.NoticeNone {
		t.Errorf("notice is %v after a refusal, want none", s.m.notice)
	}
	if !strings.Contains(s.status(), render.StateWord(s.m.failure.Kind)) {
		t.Errorf("the status line is %q and does not carry the state word %q", s.status(), render.StateWord(s.m.failure.Kind))
	}
	// The refusal's own sentence is the mapper's Detail, and the line is where a failure with posts
	// loaded is stated: view.pane leaves the posts exactly where they were, so the state is on the
	// line and the list is not rewritten to hold it.
	if !strings.Contains(s.status(), render.Remedy(s.m.failure.Kind)) {
		t.Errorf("the status line is %q and does not carry the remedy %q", s.status(), render.Remedy(s.m.failure.Kind))
	}
	for _, word := range []string{render.CopyNoticeTranslated, render.CopyNoticeNoTranslation} {
		if strings.Contains(s.status(), word) {
			t.Errorf("the status line carries the notice %q beside a refusal", word)
		}
	}
	if s.m.pending != 0 {
		t.Errorf("pending is %d after a refusal, want 0", s.m.pending)
	}
	if !strings.Contains(s.pane(), translateFirstPostBody) {
		t.Errorf("the refused post's own body left the pane:\n%s", s.pane())
	}
}

// TestNoPortOrLanguageSpendsNothing: a build with no translation endpoint, and a build whose
// language never resolved, are wiring facts rather than failures. Each says so in the notice field
// and issues nothing — a refused press must not read as a request that failed.
func TestNoPortOrLanguageSpendsNothing(t *testing.T) {
	for _, arm := range []struct {
		name   string
		remove func(*shell)
	}{
		{name: "no port", remove: func(s *shell) { s.m.deps.Translate = nil }},
		{name: "no language", remove: func(s *shell) { s.m.deps.TranslateLang = "" }},
	} {
		t.Run(arm.name, func(t *testing.T) {
			s := newShell(t, 120, 40)
			s.selectPost(translateFirstPostID)
			arm.remove(s)

			cmd := s.key("T")

			if cmd != nil {
				t.Fatalf("`T` with %s returned a command: nothing may be spent", arm.name)
			}
			if got := s.tr.Count(); got != 0 {
				t.Errorf("the port was called %d times with %s, want none", got, arm.name)
			}
			if s.m.notice != render.NoticeNoTranslation {
				t.Errorf("notice is %v with %s, want the no-translation notice", s.m.notice, arm.name)
			}
			if !strings.Contains(s.status(), render.CopyNoticeNoTranslation) {
				t.Errorf("status is %q with %s, want the no-translation sentence", s.status(), arm.name)
			}
		})
	}
}

// TestTranslateWithNoPostUnderTheCursor: a frame whose selection names no post — a people row, an
// empty list — says there is nothing to open and asks the port nothing.
func TestTranslateWithNoPostUnderTheCursor(t *testing.T) {
	s := newShell(t, 120, 40)
	// The same shape TestEnterWithNothingToOpenPrintsTheNotice uses: a shell at its start with no
	// first page, so the list holds no post and nothing is selected.
	s.m = New(s.m.deps)
	s.send(tea.WindowSizeMsg{Width: s.w, Height: s.h})

	if cmd := s.key("T"); cmd != nil {
		t.Fatalf("`T` with nothing to translate returned a command")
	}
	if got := s.tr.Count(); got != 0 {
		t.Errorf("the port was called %d times with nothing to translate", got)
	}
	if s.m.notice != render.NoticeNothingToOpen {
		t.Errorf("notice is %v, want the nothing-to-open notice", s.m.notice)
	}
	if !strings.Contains(s.status(), render.CopyNoticeNothingToOpen) {
		t.Errorf("status is %q, want the nothing-to-open sentence", s.status())
	}
}

// TestTheCoveredTranslationIsTheStylesOwn: `T` is not a cover-breaking key, so under each of the four
// covers the translated body lands in the cover's own grammar and the notice is the style's own
// wording — the plain sentence never reaches a covered frame.
func TestTheCoveredTranslationIsTheStylesOwn(t *testing.T) {
	for _, cover := range render.CoverStyles() {
		t.Run(string(cover), func(t *testing.T) {
			s := newShellIn(t, 120, 40, Display{State: StateImmersive, Cover: cover})
			s.selectPost(translateFirstPostID)

			s.press("T")

			frame := ansi.Strip(s.frame())
			if !strings.Contains(frame, translatedBody) {
				t.Fatalf("the covered frame does not carry the translated body:\n%s", frame)
			}
			if strings.Contains(frame, translateFirstPostBody) {
				t.Errorf("the covered frame still carries the post's own body")
			}
			want := render.NoticeLine(cover, render.NoticeTranslated, 0)
			if want == "" || want == render.CopyNoticeTranslated {
				t.Fatalf("the %s cover has no wording of its own for a translation: %q", cover, want)
			}
			if !strings.Contains(ansi.Strip(s.status()), want) {
				t.Errorf("the covered status line is %q, want the style's own %q", ansi.Strip(s.status()), want)
			}
			for _, plain := range []string{render.CopyNoticeTranslated, render.CopyNoticeNoTranslation} {
				if strings.Contains(frame, plain) {
					t.Errorf("the covered frame carries the plain wording %q", plain)
				}
			}
			if got := s.tr.Count(); got != 1 {
				t.Errorf("the port was called %d times under the %s cover, want one", got, cover)
			}
		})
	}
}

// TestTheTranslationMapIsNotMutatedInPlace: Model is a value, so a model held before an answer keeps
// rendering the original. A map written through in place would move under every value that shares it
// — a command that captured a model, a test holding one — and the pane would then disagree with the
// state it was rendered from.
func TestTheTranslationMapIsNotMutatedInPlace(t *testing.T) {
	s := newShell(t, 120, 40)
	s.selectPost(translateFirstPostID)
	held := s.m

	s.press("T")

	if !strings.Contains(s.pane(), translatedBody) {
		t.Fatalf("`T` did not show the translation on the live model")
	}
	before := held.Render(held.w, held.h)
	if strings.Contains(before, translatedBody) {
		t.Errorf("the model held before the answer renders the translation: the map was mutated in place")
	}
	if !strings.Contains(before, translateFirstPostBody) {
		t.Errorf("the model held before the answer no longer renders the original body:\n%s", before)
	}
}
