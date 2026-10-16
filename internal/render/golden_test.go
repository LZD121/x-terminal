package render

// This file is the view specification's model half: one corpus of states, rendered through
// internal/render at a pinned environment and an injected clock, and stored as golden frames under
// testdata/TestGoldenFrames/. corpus_test.go is the audit that reads those stored bytes back.
//
// Two things about the corpus's shape are deliberate.
//
// The environment is pinned in the test, never inherited: TMUX, TERM, COLORTERM and NO_COLOR are
// set for every render, because this host's non-interactive shells export TERM=dumb and NO_COLOR=1,
// $TMUX switches the width model and $TZ moves every stamp — an inherited environment would make
// the stored bytes a property of the shell that generated them.
//
// The states come from the real code path. The timeline frames are the fake's adversarial page set
// absorbed through screen.List's own reducers, the failure frames are domain.Failure values of the
// four named kinds, the help frames are keys.Rows() mapped into HelpRow, and every frame is
// composed by the three functions the shell's own Render composes (Blocks/StateBlock/Help, the
// status fields, Frame). A stored frame is therefore the screen's own bytes for that state rather
// than a picture of something close to it — which is what makes a disagreement with the terminal
// captures in capture_tmux_test.go and capture_pty_test.go a bug report.
import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/exp/golden"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/fake"
	"github.com/LZD121/x-terminal/internal/keys"
	"github.com/LZD121/x-terminal/internal/screen"
	"github.com/LZD121/x-terminal/internal/xweb"
)

// corpusNow is the corpus's clock: the instant every stored frame's stamps, position and rate-limit
// reset are rendered against, so a stored frame is a fact about the state and not about the day it
// was generated.
var corpusNow = time.Date(2026, 10, 14, 9, 12, 0, 0, time.UTC)

// corpusH is the height of every stored frame: the pane is corpusH-1 rows and the status line is
// the last one.
const corpusH = 24

// twinSuffix is what a frame's multiplexer twin is stored as. The name is part of the file's path,
// so it is written once, here.
const twinSuffix = "_tmux"

// environment is the four variables the renderer reads: two of them (TMUX) choose the width model
// and the other three choose the colour rung. Every one is set for every frame, including the ones
// whose value is empty, so no frame can inherit one from the shell running the test.
type environment struct {
	tmux      string
	term      string
	colorterm string
	noColor   string
}

// The environments the corpus is rendered in.
//
// The state frames hold no colour at all (TERM=dumb, NO_COLOR set): they are the layout and the
// words, and the rung matrix's four frames are where colour is specified. The tmux twins keep the
// same rung, because what the twin is for is the width model, not the palette.
var (
	envPlain = environment{term: "dumb", noColor: "1"}
	// envTmux carries a plausible TMUX (socket,pid,index); only its emptiness is read.
	envTmux      = environment{tmux: "/tmp/xterm-golden,1234,0", term: "dumb", noColor: "1"}
	envTrueColor = environment{term: "xterm-256color", colorterm: "truecolor"}
	env256       = environment{term: "xterm-256color"}
	env16        = environment{term: "xterm"}
	envMono      = environment{term: "xterm-256color", noColor: "1"}
)

// pin sets the environment for one render. TZ goes with it: the clock is injected as UTC, and a
// stamp that followed the host's zone would be a stored frame that moved with the host.
func (e environment) pin(t *testing.T) {
	t.Helper()
	t.Setenv("TZ", "UTC")
	t.Setenv("TMUX", e.tmux)
	t.Setenv("TERM", e.term)
	t.Setenv("COLORTERM", e.colorterm)
	t.Setenv("NO_COLOR", e.noColor)
}

// paneKind is what a frame's pane draws: the list's own blocks, or the one message block a state
// with nothing to show draws, or the help overlay. The list, the message and the overlay are the
// three cases the shell's own pane() switches on.
type paneKind uint8

const (
	// paneList is the list's blocks, scrolled to the state's window.
	paneList paneKind = iota
	// paneLoading is the loading line: the first page is out and no answer has arrived.
	paneLoading
	// paneEmpty is the empty read that succeeded: the heading and the body copy.
	paneEmpty
	// paneFailure is a failure with nothing to show: the state word and the failure's sentence.
	paneFailure
	paneHelp
	// panePanic is the panic frame: one constant frame that asks the state for nothing but the
	// pane's size, so its pane is the frame's own bytes rather than a list's.
	panePanic
	paneSaved
)

// state is one frame's state: what the pane shows and what the status line says. It is the shell's
// own frame value read from the other side — the list, the pane's window into it, the sticky
// failure and the budget — with the scroll stated instead of the anchor the shell stores, because
// it is the pane's top that a corpus frame is chosen by.
type state struct {
	// list is the list the frame is of: its posts, its walk and its selection. It is the zero Kind
	// for the frames whose pane is a message.
	list screen.List
	// scroll is how many rows of the list sit above the pane's first row: the pane's window into a
	// list taller than it, which the shell derives from the anchor it stores (rowsAbove - anchor)
	// and slices the blocks to (internal/tui's scroll and scrolled). The corpus states the scroll
	// rather than the anchor because it is the pane's window that a stored frame is chosen by.
	scroll int
	// at is the post whose block the pane's first row is: the reader's place, scrolled to the top of
	// the pane, which is the window the depth-1 and depth-2 quotes are in.
	at domain.PostID
	// bottom is the anchor at its ceiling, which is the state `G` leaves the pane in: the selected
	// head's row is the pane's last visible row and everything below it is off the frame. It is the
	// shell's own rule — reanchor clamps the anchor into the pane — and the reason the block's body
	// is not in that frame.
	bottom bool
	// pane is what the pane draws.
	pane paneKind
	// cover is the display style the frame is rendered through: the zero style is the plain
	// reader, so a covered frame is the same state with one value changed and nothing else.
	cover CoverStyle
	// failure is the sticky failure the status line prints, and the pane's own words when the pane
	// has nothing else to show.
	failure domain.Failure
	// used and limited are the budget the status line carries.
	used, limited int
	// conversation is the thread frame's conversation when pane is paneThread.
	conversation Conversation
	// threadSel is the row the cursor is on in a conversation.
	threadSel Selection
	// threadHasMore is whether the reply walk holds a cursor: the position's `+`.
	threadHasMore bool
	// threadNotice is an explicit notice for a conversation frame; NoticeNone falls through to the
	// conversation's own stalled notice when the walk stopped.
	threadNotice NoticeKind
	// threadAt is the post whose block the pane's first row is in a conversation, analogous to at.
	threadAt domain.PostID
	// saved is a saved-frame state when pane is paneSaved.
	saved       screen.Saved
	savedKind   domain.SavedKind
	syncedAt    time.Time
	syncedOn    bool
	capability  domain.Capability
	capFacts    CapabilityFacts
	notice      NoticeKind
	noticeCount int
	// fall is the per-post URL fallback map the shell would pass into Blocks.
	fall Fallbacks
}

// frame is one stored frame: its name — which is its file's name under testdata/TestGoldenFrames/ —
// its width, the environment it is rendered in and its state.
type frame struct {
	name  string
	w     int
	h     int // 0 means corpusH
	env   environment
	state state
}

// grammar is the grammar this state's frame is rendered through: the cover's own, or the plain
// reader's. It is the one place the state decides which grammar the composer is handed.
func (s state) grammar() Grammar { return s.cover.Grammar() }

// blocks is the frame's pane: the rows the state's pane draws at width w. pane is the pane's own
// height (the frame's height less its status line), which is what the help overlay is cut to — the
// same number the shell hands it, so a stored overlay frame is the shipped pane's own bytes.
func (s state) blocks(w, pane int) []Block {
	g := s.grammar()
	switch s.pane {
	case panePanic:
		// The panic frame is composed from the pane's size and nothing else.
		return nil
	case paneHelp:
		return []Block{{Rows: Help(keymapRows(s.cover != ""), w, pane)}}
	case paneLoading:
		return []Block{StateBlock(g, w, LoadingLine(s.cover, s.list.Kind), "")}
	case paneEmpty:
		return []Block{StateBlock(g, w, EmptyHeading(s.cover, s.list.Kind), EmptyBody(s.cover))}
	case paneFailure:
		// The shell's own rule for the pane's failure words: the classifier's sentence under the
		// plain reader, the cover's own line under a cover. The corpus calls the same function the
		// shell calls, so a stored covered failure frame is the program's bytes and not a picture
		// of them.
		word, body := FailureWords(s.cover, s.failure.Kind, s.failure.Detail)
		return []Block{StateBlock(g, w, word, body)}
	case paneThread:
		blocks := ThreadBlocks(s.conversation, g, w, s.threadSel, corpusNow, nil)
		switch {
		case s.threadAt != "":
			s.scroll = blockOffset(blocks, s.threadAt)
		case s.at != "":
			s.scroll = blockOffset(blocks, s.at)
		}
		return window(blocks, s.scroll)
	case paneSaved:
		// The shell's own branch order and the shell's own call: the capability state first, then a
		// failure with an empty store (the third line's case), then the empty read. Every one of
		// them is read through the frame's grammar, which is what a covered saved frame needs.
		if s.capability != domain.CapNone {
			return CapabilityBlock(g, s.capability, s.capFacts, w)
		}
		if len(s.saved.List().Posts()) == 0 {
			if s.failure.Kind != domain.FailureNone {
				return StateBlockWithMirror(g, s.failure, s.savedKind, false, w)
			}
			return SavedEmptyState(g, s.savedKind, s.syncedOn && !s.syncedAt.IsZero(), w)
		}
		blocks := Blocks(s.saved.List().Posts(), g, w, selectedRow(s.saved.List()), corpusNow, s.fall)
		return window(blocks, s.scroll)
	}
	blocks := Blocks(s.list.Posts(), g, w, selectedRow(s.list), corpusNow, s.fall)
	switch {
	case s.bottom:
		s.scroll = max(0, headOffset(blocks, domain.PostID(s.list.Selected().ID))-(corpusH-2))
	case s.at != "":
		s.scroll = blockOffset(blocks, s.at)
	}
	return window(blocks, s.scroll)
}

// status is the frame's status line, composed from the list's own accessors and the state's own
// budget the way the shell composes it from the same values: no field of the line is a second copy
// of a fact, which is what stops the corpus's status lines from drifting from the program's.
func (s state) status() StatusFields {
	if s.pane == paneThread {
		return threadStatus(s)
	}
	if s.pane == paneSaved {
		fields := StatusFields{
			Failure:        s.failure,
			Notice:         s.notice,
			NoticeCount:    s.noticeCount,
			List:           ListSaved(s.cover, s.savedKind),
			Synced:         SyncedField{On: s.syncedOn, At: s.syncedAt},
			Capability:     s.capability,
			CapabilityList: s.savedKind.List(),
			Used:           s.used,
			Limited:        s.limited,
			Now:            corpusNow,
			Cover:          s.cover,
			Gated:          s.grammar().Markers.Gated,
		}
		if s.capability != domain.CapNone {
			fields.Failure = domain.Failure{}
		}
		l := s.saved.List()
		if l.Loaded() > 0 {
			fields.Selected = l.Index(l.Selected().Top().ID) + 1
			fields.Loaded = l.Loaded()
			fields.HasMore = l.HasMore()
		}
		return fields
	}
	notice := s.notice
	if notice == NoticeNone {
		notice = s.noticeFor()
	}
	return StatusFields{
		Failure:     s.failure,
		Notice:      notice,
		NoticeCount: s.noticeCount,
		List:        ListName(s.cover, s.list.Kind),
		Selected:    s.list.Index(s.list.Selected().ID) + 1,
		Loaded:      s.list.Loaded(),
		HasMore:     s.list.HasMore(),
		Used:        s.used,
		Limited:     s.limited,
		Now:         corpusNow,
		Cover:       s.cover,
		Gated:       s.grammar().Markers.Gated,
	}
}

// noticeFor is the shell's own rule as far as this corpus needs it: what the list itself says — a
// walk that ended because the cursor stopped moving. The other notices are the refresh's answer,
// and no frame in this corpus carries one.
func (s state) noticeFor() NoticeKind {
	if s.list.Stalled() {
		return NoticeStalled
	}
	return NoticeNone
}

// render is the frame's bytes: the frame composer on the pane and the status line, which is exactly
// what the shell's Render is — and, for the panic states, the panic frame's own bytes, because that
// frame is composed from the pane's size alone and carries no status line. It reads the
// environment, which the caller has pinned.
func (f frame) render() string {
	h := f.h
	if h == 0 {
		h = corpusH
	}
	if f.state.pane == panePanic {
		return PanicFrame(f.w, h)
	}
	return Frame(f.state.blocks(f.w, h-1), f.state.status(), f.w, h)
}

// bytes renders the frame in one environment, pinned for the render and left pinned afterwards:
// every render in this file decides what it draws from the environment, so the environment is part
// of the call.
func (f frame) bytes(t *testing.T, env environment) string {
	t.Helper()
	env.pin(t)
	return f.render()
}

// environments reports which environments this frame is stored in: its own, and the multiplexer's
// when the two produce different bytes.
//
// The decision is the bytes themselves rather than a list of names: the width model switches with
// $TMUX, and the two measures disagree on a cluster exactly where the frame containing it renders
// differently — the ZWJ family is one two-cell cluster in a bare terminal and six cells under a
// multiplexer. A frame whose text holds no such cluster renders the same either way, and a stored
// twin of it would be a second copy of one fact. corpus_test.go asserts the same equivalence from
// the stored files.
func (f frame) environments(t *testing.T) []environment {
	t.Helper()
	plain := f.bytes(t, f.env)
	twin := f.env
	twin.tmux = envTmux.tmux
	if f.bytes(t, twin) == plain {
		return []environment{f.env}
	}
	return []environment{f.env, twin}
}

// TestGoldenFrames stores the corpus and compares it. With -update it writes
// testdata/TestGoldenFrames/<name>.golden; without it, every frame is compared byte for byte
// against the stored file, which is the whole of what the corpus is for — a frame that moved by one
// cell, one space of padding or one escape sequence is a change to the view specification.
func TestGoldenFrames(t *testing.T) {
	for _, f := range corpus(t) {
		for _, env := range f.environments(t) {
			name := f.name
			if env.tmux != "" {
				name += twinSuffix
			}
			t.Run(name, func(t *testing.T) {
				golden.RequireEqual(t, f.bytes(t, env))
			})
		}
	}
}

// corpus is every stored frame: the UI-SPEC's own list, in its order, plus the one frame its
// checklist requires and its list does not name, plus the DSGT-08 covered cross-product
// (every later screen × every CoverStyle × 80 and 120).
//
// The list is the fake's whole adversarial page set: three pages, so the list holds every case the
// checklist names — the 15-cell and the 3-cell handle, the CJK run, the emoji, the ZWJ family, the
// over-measure URL, the exact-fill and one-short lines, both quote depths, the continuation pair,
// the repost, the limited post, the tombstone and the attachment lines. The width-bearing frames
// show the list from its top with the CJK post selected (a block that is not the pane's first row);
// timeline_last_80 shows it scrolled to its end with the last post selected, which is where the
// continuation pair, the two markers, the tombstone and the attachment lines are on screen.
//
// Covered frames are generated as a cross-product so a screen without a covered frame is a
// missing file and TestGoldenFrames / the corpus file-set arm fail on it — that failure mode is
// the point of the growth.
func corpus(t *testing.T) []frame {
	t.Helper()

	// The list, absorbed through the reducers the shell drives it through. Recording it here rather
	// than rebuilding it per frame is what makes two frames of the same state share the state.
	list := absorbed(t)

	// The selections: the CJK post, which is the list's second block, the list's last post, which is
	// the block the `G` frame sits on, and the post that carries the depth-1 quote.
	top := selected(t, list, list.Posts()[1].ID)
	end := selected(t, list, list.Posts()[len(list.Posts())-1].ID)
	quote := quotedOnce(t, list)
	stalledList := selected(t, stalledWalk(t), quote)

	// The four named failure states of OPS-01. Each cold frame's body line is the classifier's own
	// sentence — domain.Failure.Detail, which is what the pane prints — and each frame carries the
	// widest budget the status line has to fit, three digits, so that the frame shows the drop rule
	// working rather than a comfortable line.
	expired := domain.Failure{Kind: domain.FailureSessionExpired, Detail: xweb.OutcomeAuthRejected.Remedy(), Retry: true}
	rateLimited := domain.Failure{Kind: domain.FailureRateLimited, Reset: corpusNow.Add(5*time.Hour + 20*time.Minute), Detail: "wait until the reset time from x-rate-limit-reset"}
	offline := domain.Failure{Kind: domain.FailureNoRoute, Detail: routeSentence(), Retry: true}
	stale := domain.Failure{Kind: domain.FailureOperationStale, Detail: xweb.OutcomeUnknownOperation.Remedy(), Retry: true}

	// The timeline state the frames that show the list share: the list at its top, the walk's
	// cursor stored, and the rate-limited state — the one failure the UI-SPEC's own 80-column frame
	// carries, and the one FEED-05 says leaves every loaded post exactly where it was.
	timeline := state{list: top, failure: rateLimited, used: 200, limited: 12}
	bottom := state{list: end, failure: rateLimited, used: 200, limited: 12, bottom: true}
	// The stalled frame: the walk ended and the pane is where the reader was — the post they were
	// reading at the pane's first row, which is where the depth-1 quote is and the depth-2 quote
	// below it. Its budget is the small one, because the notice is the point of that frame and a
	// three-digit budget would push the notice off an 80-column line.
	stalled := state{list: stalledList, at: quote, used: 6, limited: 1}
	empty := state{pane: paneEmpty, list: screen.List{Kind: domain.Following}}
	loading := state{pane: paneLoading, list: screen.List{Kind: domain.Following}}
	// The failure whose sentence is longer than the pane: what the live binary printed when a
	// request could not reach the upstream and the state carried the transport's own error — a
	// GraphQL URL past 200 cells, then the dial failure behind it. The body has to wrap like a post
	// body; cut at the pane's edge with no mark it would be the pane saying less than it knows. The
	// sentence holds no digit, so the corpus's "a count is in no frame" absence stays about the
	// renderer and not about a URL's id.
	longFailure := domain.Failure{
		Kind: domain.FailureUnclassified,
		Detail: `the request did not complete: Get "` + xweb.BaseURL + "/i/api/graphql/" +
			strings.Repeat("AbCdEfGhIj", 12) + `/HomeLatestTimeline": dial tcp: lookup x.com: no such host`,
		Retry: true,
	}
	help := state{pane: paneHelp, list: top, failure: rateLimited, used: 200, limited: 12}
	cold := func(f domain.Failure) state {
		return state{pane: paneFailure, list: screen.List{Kind: domain.Following}, failure: f, used: 200, limited: 12}
	}

	plain := []frame{
		// The timeline at four widths: the measure is min(W-3, 96), so the body does not stretch to
		// 120 cells and the 60-column floor wraps at 57. 200 is the frame that proves the cap is
		// the cap and not the pane.
		{name: "timeline_60", w: 60, env: envPlain, state: timeline},
		{name: "timeline_80", w: 80, env: envPlain, state: timeline},
		{name: "timeline_120", w: 120, env: envPlain, state: timeline},
		{name: "timeline_200", w: 200, env: envPlain, state: timeline},
		// The three states with nothing to show, in the block shape every post uses.
		{name: "empty_80", w: 80, env: envPlain, state: empty},
		{name: "empty_120", w: 120, env: envPlain, state: empty},
		{name: "loading_80", w: 80, env: envPlain, state: loading},
		// The overlay: the one keymap table, at both widths, over the timeline's own status line.
		// With 24 keymap rows the last one (`ctrl+p`) is clipped by the 23-row pane; that is the
		// stored frame's honest height, not a second corpus height.
		{name: "help_80", w: 80, env: envPlain, state: help},
		{name: "help_120", w: 120, env: envPlain, state: help},
		// The four named failure states, cold: the pane has nothing to show and prints the state's
		// own words; the status line carries the state, the remedy and the budget.
		{name: "expired_80", w: 80, env: envPlain, state: cold(expired)},
		{name: "expired_120", w: 120, env: envPlain, state: cold(expired)},
		{name: "offline_80", w: 80, env: envPlain, state: cold(offline)},
		{name: "offline_120", w: 120, env: envPlain, state: cold(offline)},
		{name: "schemachange_80", w: 80, env: envPlain, state: cold(stale)},
		{name: "schemachange_120", w: 120, env: envPlain, state: cold(stale)},
		// The failure whose sentence is longer than the measure: the state word on the head row and
		// the body wrapped over as many rows as the sentence needs, rather than cut at the pane's
		// edge with no mark.
		{name: "longdetail_80", w: 80, env: envPlain, state: cold(longFailure)},
		// The walk that ended: the posts it received are on screen and the notice says why the walk
		// stopped, with the position's `+` gone because there is nowhere to go.
		{name: "stalled_80", w: 80, env: envPlain, state: stalled},
		// The pane below its floor. The name is the UI-SPEC's; the width is the floor's neighbour
		// (MinPaneCols-1), because that is the pane the narrow line is rendered in — at 79 columns
		// the pane lays posts out and the narrow line never appears. See the corpus's note in
		// 02-05-SUMMARY.md.
		{name: "narrow_79", w: MinPaneCols - 1, env: envPlain, state: timeline},
		// The colour ladder over one state: the same timeline, at each rung, so a rung that moved a
		// character or dropped a marker is visible as a diff of one frame against another.
		{name: "color_truecolor", w: 80, env: envTrueColor, state: timeline},
		{name: "color_256", w: 80, env: env256, state: timeline},
		{name: "color_16", w: 80, env: env16, state: timeline},
		{name: "color_mono", w: 80, env: envMono, state: timeline},
		// The selection on the pane's last visible row: the other end of the checklist's pair of
		// selections, stored as `G` leaves it — the anchor at its ceiling, the selected head on the
		// pane's last row, and the walk's end because the whole list is loaded.
		{name: "timeline_last_80", w: 80, env: envPlain, state: bottom},
	}

	// The phase's frames: the same loaded model the timeline frames show — the same list, the same
	// selection, the same budget — with the one failure whose covered line is longer than the pane
	// at 80 columns, so the drop rule's declared consequence is visible in the stored bytes: the
	// `list` field disappears under the cover at 80 (the UI-SPEC names `session expired` as one of
	// the three states where it does), and it is back at 120. The plain twin of that state keeps
	// its plain name at 80 in the same 79 cells, which is the arm the audit reads the drop against.
	// Four styles, each over the same state: the covered frames differ from each other in nothing
	// but the grammar and the cover field, which is the whole of what wearing a cover changes.
	covered := state{list: coveredSafeList(t), failure: expired, used: 200, limited: 12}
	// The covered failure panes are the phase's hardest promise, stored: the same cold no-route
	// state the plain `offline_{80,120}` frames store, whose detail is the reachability check's own
	// sentence with the host in it twice. A covered frame must hold none of it — the pane it shows
	// says what a tool would say and the head row still says which failure it was — while its plain
	// twin holds the check's sentence byte for byte, which is the arm the audit reads the absence
	// against rather than a frame that is quiet because nothing was ever there.
	coveredFailure := withCover(cold(offline), CoverGitLog)
	phase := []frame{
		{name: "cover_servicelog_80", w: 80, env: envPlain, state: withCover(covered, CoverServiceLog)},
		{name: "cover_servicelog_120", w: 120, env: envPlain, state: withCover(covered, CoverServiceLog)},
		{name: "cover_gitlog_80", w: 80, env: envPlain, state: withCover(covered, CoverGitLog)},
		{name: "cover_gitlog_120", w: 120, env: envPlain, state: withCover(covered, CoverGitLog)},
		{name: "cover_ci_80", w: 80, env: envPlain, state: withCover(covered, CoverCI)},
		{name: "cover_ci_120", w: 120, env: envPlain, state: withCover(covered, CoverCI)},
		{name: "cover_internal_80", w: 80, env: envPlain, state: withCover(covered, CoverInternal)},
		{name: "cover_internal_120", w: 120, env: envPlain, state: withCover(covered, CoverInternal)},
		// The covered failure panes, at both widths, under the style the whole-surface tests drive.
		{name: "cover_gitlog_offline_80", w: 80, env: envPlain, state: coveredFailure},
		{name: "cover_gitlog_offline_120", w: 120, env: envPlain, state: coveredFailure},
		// The panic frames are the other state's surface: one constant frame at each width, composed
		// from the pane's size and nothing else, so they carry no status line and no list at all.
		{name: "panic_80", w: 80, env: envPlain, state: state{pane: panePanic}},
		{name: "panic_120", w: 120, env: envPlain, state: state{pane: panePanic}},
	}

	return append(append(append(append(append(plain, phase...), threadFrames(t)...), savedFrames(t)...), mediaFrames(t)...), coveredCrossProduct(t)...)
}

// withCover is the state with a cover worn: the same list, the same failure, the same budget and
// the same pane, with the one field a covered frame adds.
func withCover(s state, style CoverStyle) state {
	s.cover = style
	return s
}

// absorbed is the corpus's list: the fake's whole page set walked into it the way the shell walks
// it — page one with the empty cursor, and every later page through the list's own pagination
// trigger — with the recorded fixture's engagement numbers on the posts.
func absorbed(t *testing.T) screen.List {
	t.Helper()
	return walk(t, fake.Adversarial(), 3)
}

// walk drives a list over a source's pages.
func walk(t *testing.T, src *fake.Source, pages int) screen.List {
	t.Helper()
	list := absorb(t, src, screen.List{Kind: domain.Following}, "")
	for i := 1; i < pages; i++ {
		next, eff := list.Update(screen.PageRequested{}, corpusNow)
		if eff.Kind != screen.EffectPage {
			t.Fatalf("the walk asked for %v after page %d, want a page", eff.Kind, i)
		}
		list = absorb(t, src, next, eff.Cursor)
	}
	return list
}

// stalledWalk is the walk that ends because the cursor repeated: page one, then a request answered
// with the cursor it was asked with, which is the upstream pointing at a position the walk has
// already read. The list keeps every post it did receive, which is what the frame shows.
func stalledWalk(t *testing.T) screen.List {
	t.Helper()
	src := fake.Adversarial()
	list := absorb(t, src, screen.List{Kind: domain.Following}, "")
	next, eff := list.Update(screen.PageRequested{}, corpusNow)
	if eff.Kind != screen.EffectPage {
		t.Fatalf("the walk asked for %v with a cursor stored, want a page", eff.Kind)
	}
	src.StallWith(eff.Cursor)
	list = absorb(t, src, next, eff.Cursor)
	if !list.Stalled() {
		t.Fatalf("the list did not notice the cursor %q coming back: it is not stalled", eff.Cursor)
	}
	return list
}

// absorb fetches one page with the cursor and hands it to the list, the way the shell answers the
// request it issued.
func absorb(t *testing.T, src *fake.Source, list screen.List, c domain.Cursor) screen.List {
	t.Helper()
	page, err := src.Home(context.Background(), domain.Following, c)
	if err != nil {
		t.Fatalf("the fake serves no page at %q: %v", c, err)
	}
	next, _ := list.Update(screen.PageFetched{Page: withCounts(t, page)}, corpusNow)
	return next
}

// selected moves a list's selection through the list's own reducer, and fails rather than rendering
// a frame whose selection is not the one the corpus asked for.
func selected(t *testing.T, list screen.List, id domain.PostID) screen.List {
	t.Helper()
	next, _ := list.Update(screen.Select{Target: screen.Target{ID: string(id)}}, corpusNow)
	if next.Selected().ID != string(id) {
		t.Fatalf("the list does not hold the post %q: it selected %q instead", id, next.Selected())
	}
	return next
}

// withCounts puts engagement numbers on a page's posts: the recorded fixture's own, cloned rather
// than mutated in place.
//
// The fake's adversarial posts carry none — they are hand-authored, and a post whose counts are all
// zero renders the same whether or not the grammar prints them. With the recorded fixture's real
// numbers on them, the audit's "no stored frame contains an engagement count" is a check rather
// than a tautology: a renderer that printed a post's counts would print these.
func withCounts(t *testing.T, page domain.PostPage) domain.PostPage {
	t.Helper()
	counts := engagement(t)
	posts := slices.Clone(page.Posts)
	for i := range posts {
		posts[i].Counts = counts
	}
	page.Posts = posts
	return page
}

// engagement is the recorded fixture's own engagement numbers — the fixture the project reads its
// timelines from, rather than numbers invented here for a test to search for. It fails if they are
// zero, because the absence the audit checks would then be vacuous.
func engagement(t *testing.T) domain.Counts {
	t.Helper()
	page, err := fake.Recorded().Home(context.Background(), domain.Following, "")
	if err != nil {
		t.Fatalf("the recorded fixture does not serve its first page: %v", err)
	}
	if len(page.Posts) == 0 {
		t.Fatal("the recorded fixture's first page holds no posts")
	}
	counts := page.Posts[0].Counts
	if counts == (domain.Counts{}) {
		t.Fatal("the recorded fixture's first post carries no engagement numbers: the corpus's absence check would be vacuous")
	}
	return counts
}

// routeSentence is netcheck's own sentence for a lost route: the check builds it from the error the
// navigation request failed with, as `GET <base>/ failed: <err>`. The corpus cannot run the check —
// it reaches the network, and no test here does — so the error half is a representative one.
func routeSentence() string {
	return "GET " + xweb.BaseURL + "/ failed: dial tcp: lookup x.com: no such host"
}

// keymapRows is the one keymap table mapped into the overlay's rows, exactly as the shell maps it:
// the keystrokes through the table's own join rule, the help text rendered verbatim and the group
// carried along (the overlay does not print it). It is the whole table — help_test.go's helpRows is
// the hand-written sample its layout assertions are made against — so the corpus's overlay frames
// are the overlay the program actually draws.
// keymapRows is the one keymap table mapped into the overlay's rows, exactly as the shell maps it:
// the keystrokes through the table's own join rule, the help text rendered verbatim — in the
// covered wording when a cover is worn — and the group carried along (the overlay does not print
// it). It is the whole table — help_test.go's helpRows is the hand-written sample its layout
// assertions are made against — so the corpus's overlay frames are the overlay the program draws.
func keymapRows(covered bool) []HelpRow {
	rows := make([]HelpRow, 0, len(keys.Rows()))
	for _, row := range keys.Rows() {
		rows = append(rows, HelpRow{Keys: row.KeysJoined(), Text: row.HelpText(covered), Group: row.Group})
	}
	return rows
}

// selectedRow is the corpus's selection as the renderer's own value: the list's Target converted at the
// same seam the shell converts it, so a stored frame with a quoted row selected is the program's bytes
// rather than a picture of them.
func selectedRow(list screen.List) Selection {
	sel := Selection{ID: domain.PostID(list.Selected().ID)}
	for _, id := range list.Selected().Path {
		sel.Path = append(sel.Path, domain.PostID(id))
	}
	return sel
}

// headOffset is the row the selected block's head sits on in the list: the block's first row plus
// the blank separator rows the block builder puts in front of a block that follows another. It is
// the same row internal/tui's heads finds by taking the block's first non-empty row, and it is what
// the anchor — the selected head's distance from the pane's first row — counts from.
func headOffset(blocks []Block, id domain.PostID) int {
	offset := blockOffset(blocks, id)
	for _, block := range blocks {
		if block.PostID != id {
			continue
		}
		for _, row := range block.Rows {
			if row != "" {
				break
			}
			offset++
		}
		break
	}
	return offset
}

// blockOffset is how many rows of the list sit above a given post's block: the scroll that puts
// that block's first row on the pane's first row. It is the shell's own arithmetic read the other
// way: the anchor — the selected head's distance from the pane's first row — is one for a block that
// follows another, so the reader's place at the top of the pane is a position the pane can hold.
func blockOffset(blocks []Block, id domain.PostID) int {
	offset := 0
	for _, block := range blocks {
		if block.PostID == id {
			return offset
		}
		offset += len(block.Rows)
	}
	return 0
}

// quotedOnce is the post that quotes a post quoting nothing further: the depth-1 quote case, and the
// place the stalled frame's pane is scrolled to. The fixture holds exactly one, and a second one
// would mean the corpus's frame could not be the case it is named for.
func quotedOnce(t *testing.T, list screen.List) domain.PostID {
	t.Helper()
	found := domain.PostID("")
	for _, post := range list.Posts() {
		if post.Quote == nil || post.Quote.Quote != nil {
			continue
		}
		if found != "" {
			t.Fatalf("the list holds two depth-1 quotes (%s and %s): the corpus's frame cannot be the case it is named for", found, post.ID)
		}
		found = post.ID
	}
	if found == "" {
		t.Fatal("the list holds no post whose quote quotes nothing further: the corpus has no depth-1 quote to show")
	}
	return found
}

// window drops the rows of the list that sit above the pane's first row: the pane is a window into
// a list taller than it, and the frame draws the rows from the window's own top. The shell does
// exactly this to the blocks it built (internal/tui's scrolled), and the rows of a block that is
// partly scrolled out are its own — a slice of them, never a re-wrap.
func window(blocks []Block, skip int) []Block {
	if skip <= 0 {
		return blocks
	}
	out := make([]Block, 0, len(blocks))
	for _, block := range blocks {
		switch {
		case skip >= len(block.Rows):
			skip -= len(block.Rows)
		case skip > 0:
			out = append(out, Block{PostID: block.PostID, Rows: block.Rows[skip:]})
			skip = 0
		default:
			out = append(out, block)
		}
	}
	return out
}

// savedFrames is Phase 6's golden corpus: the frames 06-UI-SPEC.md's table lists.
func savedFrames(t *testing.T) []frame {
	t.Helper()
	posts := savedCorpusPosts()
	var saved screen.Saved
	saved = screen.NewSaved(domain.ListBookmarks, domain.CapNone)
	page := domain.PostPage{Posts: posts, Next: "more"}
	saved = saved.Update(screen.MirrorPage{Page: page})
	saved = saved.Update(screen.StatusRead{
		Owner: "mira", LastSuccess: time.Date(2026, 1, 2, 15, 4, 0, 0, time.UTC),
	})
	// Select the second post (mira) so " > " appears on a mirrored row.
	if len(posts) > 1 {
		next, _ := saved.List().Update(screen.Select{Target: screen.Target{ID: string(posts[1].ID)}}, corpusNow)
		saved = saved.WithList(next)
	}
	offline := domain.Failure{Kind: domain.FailureNoRoute, Detail: "check the proxy", Retry: true}
	base := state{
		pane: paneSaved, saved: saved, savedKind: domain.Bookmark,
		syncedOn: true, syncedAt: time.Date(2026, 1, 2, 15, 4, 0, 0, time.UTC),
		failure: offline, used: 12, limited: 0,
	}
	never := state{
		pane: paneSaved, savedKind: domain.Bookmark, syncedOn: true,
		used: 12, limited: 0,
	}
	emptySynced := state{
		pane: paneSaved, savedKind: domain.Bookmark,
		syncedOn: true, syncedAt: time.Date(2026, 1, 2, 15, 4, 0, 0, time.UTC),
		used: 12, limited: 0,
	}
	// Force Read-equivalent empty by using SavedEmptyState path (no posts).
	hedg80 := state{
		pane: paneSaved, savedKind: domain.Like, syncedOn: true, syncedAt: corpusNow,
		used: 12, limited: 0,
	}
	unavail := state{
		pane: paneSaved, savedKind: domain.Like, syncedOn: true,
		capability: domain.CapOperationMissing,
		capFacts:   CapabilityFacts{List: "likes", Path: "/tmp/saved.db"},
		used:       12, limited: 0,
	}
	wrong := state{
		pane: paneSaved, saved: saved, savedKind: domain.Bookmark,
		syncedOn: true, syncedAt: time.Date(2026, 1, 2, 15, 4, 0, 0, time.UTC),
		capability: domain.CapWrongAccount,
		capFacts:   CapabilityFacts{List: "bookmarks", Owner: "mira", Path: "/tmp/saved.db"},
		used:       12, limited: 0,
	}
	unreadable := state{
		pane: paneSaved, savedKind: domain.Bookmark, syncedOn: true,
		capability: domain.CapMirrorUnreadable,
		capFacts:   CapabilityFacts{List: "bookmarks", Path: "/tmp/saved.db"},
		used:       12, limited: 0,
	}
	syncing := base
	syncing.notice = NoticeSyncing
	syncing.noticeCount = 40
	syncing.used = 200
	syncing.limited = 12
	syncing.syncedAt = corpusNow // today form
	absent := base
	absent.notice = NoticeNoLongerSaved
	absent.noticeCount = 3
	absent.failure = domain.Failure{}
	color := base
	color.failure = domain.Failure{}
	return []frame{
		{name: "bookmarks_80", w: 80, env: envPlain, state: base},
		{name: "bookmarks_120", w: 120, env: envPlain, state: base},
		{name: "bookmarks_never_synced_80", w: 80, env: envPlain, state: never},
		{name: "bookmarks_synced_empty_80", w: 80, env: envPlain, state: emptySynced},
		{name: "likes_hedged_80", w: 80, env: envPlain, state: hedg80},
		{name: "likes_hedged_120", w: 120, env: envPlain, state: hedg80},
		{name: "likes_unavailable_80", w: 80, env: envPlain, state: unavail},
		{name: "likes_unavailable_120", w: 120, env: envPlain, state: unavail},
		{name: "wrong_account_80", w: 80, env: envPlain, state: wrong},
		{name: "mirror_unreadable_80", w: 80, env: envPlain, state: unreadable},
		{name: "bookmarks_syncing_80", w: 80, env: envPlain, state: syncing},
		{name: "bookmarks_absent_rows_80", w: 80, env: envPlain, state: absent},
		{name: "color_truecolor_saved", w: 80, env: envTrueColor, state: color},
		{name: "color_256_saved", w: 80, env: env256, state: color},
		{name: "color_16_saved", w: 80, env: env16, state: color},
		{name: "color_mono_saved", w: 80, env: envMono, state: color},
	}
}

func savedCorpusPosts() []domain.Post {
	kestrel := domain.User{ID: "u-kestrel", ScreenName: "kestrel"}
	mira := domain.User{ID: "u-mira", ScreenName: "mira"}
	aergon := domain.User{ID: "u-aergon", ScreenName: "aergon"}
	long := domain.User{ID: "u-long", ScreenName: "longhandle_name"}
	return []domain.Post{
		{ID: "s1", Author: kestrel, Text: "saved for later: the mirror holds the two saved lists and nothing else.", CreatedAt: time.Date(2026, 10, 13, 21, 4, 0, 0, time.UTC)},
		{ID: "s2", Author: mira, Text: "a bookmark is a post, so the row is the row - only the status line knows when the mirror was last filled.\nCJK: 日本語と絵文字👨‍👩‍👧‍👦",
			CreatedAt: time.Date(2026, 10, 12, 18, 20, 0, 0, time.UTC),
			Media:     []domain.MediaRef{{Kind: domain.Image}},
		},
		{ID: "s3", Author: aergon, Text: "the likes list needs an operation id the bundle scrape never saw.", CreatedAt: time.Date(2026, 10, 12, 7, 45, 0, 0, time.UTC)},
		{ID: "s4", Author: kestrel, Visibility: domain.Unavailable, CreatedAt: time.Date(2026, 10, 11, 9, 0, 0, 0, time.UTC)},
		{ID: "s5", Author: long, Text: "fifteen-cell handle row", CreatedAt: time.Date(2026, 10, 10, 9, 0, 0, 0, time.UTC)},
	}
}

// mediaFrames is Phase 7's media/link golden corpus: attachment fallbacks,
// permalink/copy notices, and the four covered gated refusals.
func mediaFrames(t *testing.T) []frame {
	t.Helper()
	author := domain.User{ID: "u-aergon", ScreenName: "aergon", Name: "name of aergon"}
	videoURL := "https://video.twimg.com/ext_tw_video/1812041/pu/vid/1280x720/9nQ4kLm3xT8aBcD.mp4"
	mixed := domain.Post{
		ID: "media-mixed", Author: author, CreatedAt: corpusNow.Add(-time.Hour),
		Text: "two clips from the archive; one of them needs a player.",
		Media: []domain.MediaRef{
			{Kind: domain.Image, URL: "https://pbs.twimg.com/media/opened.jpg"},
			{Kind: domain.Video, URL: "https://pbs.twimg.com/media/poster.jpg", Playable: videoURL},
		},
	}
	opened := domain.Post{
		ID: "media-opened", Author: author, CreatedAt: corpusNow.Add(-time.Hour),
		Text: "both attachments opened as asked.",
		Media: []domain.MediaRef{
			{Kind: domain.Image, URL: "https://pbs.twimg.com/media/a.jpg"},
			{Kind: domain.Video, URL: "https://pbs.twimg.com/media/b.jpg", Playable: "https://video.twimg.com/opened.mp4"},
		},
	}
	all := domain.Post{
		ID: "media-all", Author: author, CreatedAt: corpusNow.Add(-time.Hour),
		Text: "four attachments and no opener at all.",
		Media: []domain.MediaRef{
			{Kind: domain.Image, URL: "https://pbs.twimg.com/media/" + strings.Repeat("A", 80) + "1.jpg"},
			{Kind: domain.Image, URL: "https://pbs.twimg.com/media/" + strings.Repeat("B", 80) + "2.jpg"},
			{Kind: domain.Video, URL: "https://pbs.twimg.com/media/poster3.jpg", Playable: "https://video.twimg.com/" + strings.Repeat("C", 80) + "3.mp4"},
			{Kind: domain.GIF, URL: "https://pbs.twimg.com/media/" + strings.Repeat("D", 80) + "4.gif"},
		},
	}
	none := domain.Post{
		ID: "media-none", Author: author, CreatedAt: corpusNow.Add(-time.Hour),
		Text: "a post with no attachments at all.",
	}
	measure := Plain.measure(80, 0, 0)
	urlCol := measure - EntCol
	exactURL := strings.Repeat("u", urlCol*2)
	exact := domain.Post{
		ID: "media-exact", Author: author, CreatedAt: corpusNow.Add(-time.Hour),
		Text:  "a URL whose length is an exact multiple of the wrap column.",
		Media: []domain.MediaRef{{Kind: domain.Image, URL: exactURL}},
	}
	linkPost := domain.Post{
		ID: "media-link", Author: author, CreatedAt: corpusNow.Add(-time.Hour),
		Text: "a record whose permalink is copied or opened.",
	}
	link := Permalink(linkPost)
	mk := func(posts []domain.Post, id domain.PostID, fall Fallbacks, notice NoticeKind, count int) state {
		list := absorbPosts(t, posts, id)
		return state{list: list, fall: fall, notice: notice, noticeCount: count, used: 12, limited: 0}
	}
	mixedState := mk([]domain.Post{mixed}, mixed.ID, Fallbacks{mixed.ID: {Failed: []int{2}}}, NoticeNoOpener, 0)
	openedState := mk([]domain.Post{opened}, opened.ID, nil, NoticeOpenedVideo, 0)
	allState := mk([]domain.Post{all}, all.ID, Fallbacks{all.ID: {Failed: []int{1, 2, 3, 4}}}, NoticeNoOpener, 0)
	noneState := mk([]domain.Post{none}, none.ID, nil, NoticeNoMedia, 0)
	exactState := mk([]domain.Post{exact}, exact.ID, Fallbacks{exact.ID: {Failed: []int{1}}}, NoticeNoOpener, 0)
	permalinkState := mk([]domain.Post{linkPost}, linkPost.ID, Fallbacks{linkPost.ID: {Permalink: true, Link: link}}, NoticeNoOpener, 0)
	copiedState := mk([]domain.Post{linkPost}, linkPost.ID, nil, NoticeLinkCopied, 0)
	copiedWLState := mk([]domain.Post{linkPost}, linkPost.ID, nil, NoticeLinkCopiedWL, 0)
	copiedWLState.failure = domain.Failure{Kind: domain.FailureSessionExpired, Detail: "session expired"}
	// Covered gated frames use the URL-/noun-safe feed so the DSGT-08 no-http and
	// product-noun arms stay meaningful over the whole covered corpus.
	gatedBase := state{list: coveredSafeList(t), notice: NoticeGated, used: 12, limited: 0}
	return []frame{
		{name: "media_opened_80", w: 80, env: envPlain, state: openedState},
		{name: "media_fallback_80", w: 80, env: envPlain, state: mixedState},
		{name: "media_fallback_120", w: 120, env: envPlain, state: mixedState},
		{name: "media_no_opener_all_80", w: 80, env: envPlain, state: allState},
		{name: "media_none_80", w: 80, env: envPlain, state: noneState},
		{name: "media_wrap_exact_80", w: 80, env: envPlain, state: exactState},
		{name: "permalink_fallback_80", w: 80, env: envPlain, state: permalinkState},
		{name: "media_copied_80", w: 80, env: envPlain, state: copiedState},
		{name: "media_copied_wl_80", w: 80, env: envPlain, state: copiedWLState},
		{name: "media_gated_servicelog_80", w: 80, env: envPlain, state: withCover(gatedBase, CoverServiceLog)},
		{name: "media_gated_gitlog_80", w: 80, env: envPlain, state: withCover(gatedBase, CoverGitLog)},
		{name: "media_gated_ci_80", w: 80, env: envPlain, state: withCover(gatedBase, CoverCI)},
		{name: "media_gated_internal_80", w: 80, env: envPlain, state: withCover(gatedBase, CoverInternal)},
		{name: "color_truecolor_fallback", w: 80, env: envTrueColor, state: mixedState},
		{name: "color_256_fallback", w: 80, env: env256, state: mixedState},
		{name: "color_16_fallback", w: 80, env: env16, state: mixedState},
		{name: "color_mono_fallback", w: 80, env: envMono, state: mixedState},
	}
}

// absorbPosts builds a one-page Following list from posts and selects id.
func absorbPosts(t *testing.T, posts []domain.Post, id domain.PostID) screen.List {
	t.Helper()
	list := screen.List{Kind: domain.Following}
	next, _ := list.Update(screen.PageFetched{Page: domain.PostPage{Posts: posts}}, corpusNow)
	return selected(t, next, id)
}
