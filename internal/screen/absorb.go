package screen

import (
	"time"

	"github.com/LZD121/x-terminal/internal/domain"
)

// Event is one thing that happened to a list: a key that may ask for a page, an answer that
// arrived, a failure, or a selection move.
//
// It is a closed set, and it is the same set for every frame in the product: the thread frame's own
// events (thread.go) are cases of it too, and a frame ignores the events that are not its own in the
// same way it ignores one this build does not know. The marker method is unexported, so no package
// outside this one can add an event — the state machine's transitions are exactly the ones written
// here and in thread.go, and every one of them is tested.
type Event interface{ event() }

// PageRequested asks for one more page of the walk, which is what reaching the last loaded post
// with a stored cursor amounts to. A repeated key while a page is in flight is a no-op rather than
// a queued request.
type PageRequested struct{}

// PageFetched is one page of the walk arriving.
type PageFetched struct{ Page domain.PostPage }

// PageFailed is a request that did not deliver data — the answer, or the failure of the request
// itself. It is the failure the status line prints, already named by the classifier in
// internal/cli.
type PageFailed struct{ Failure domain.Failure }

// RefreshRequested asks for page one again, to find out what is new at the top. It moves nothing:
// the posts it brings are prepended and the selection is left where it was.
type RefreshRequested struct{}

// Refreshed is the answer to a refresh.
type Refreshed struct{ Page domain.PostPage }

// Select moves the selection onto a row the list holds. The row is a Target and not an id, because a
// list's rows are its posts and their quoted posts: a quoted post is selectable, and a target the
// list does not hold moves nothing.
type Select struct{ Target Target }

func (PageRequested) event()    {}
func (PageFetched) event()      {}
func (PageFailed) event()       {}
func (RefreshRequested) event() {}
func (Refreshed) event()        {}
func (Select) event()           {}

// Update applies one event and returns the next state with the single effect it asks for.
//
// now is the caller's clock, and only the two events that may issue a request look at it: a rate
// limit is judged against the reset instant the upstream sent, never against a timer of this
// package's own. Everything else — absorbing an answer, moving the selection, a failed page — is
// decided without a clock, because none of it depends on what time it is.
//
// An event this build does not know (including a nil Event, and including the thread frame's own
// events) is a no-op: a state machine that guessed at an unknown transition is a state machine whose
// tests prove nothing.
//
// Each case is a fold into the shared walk, and none of them keeps a rule of its own: this file is
// the home frame's vocabulary — which timeline, which page value, which refresh rule — and walk.go
// is where the de-duplication, the stall rule, the single-flight latch, the request gate and the
// refresh diff live.
func (l List) Update(ev Event, now time.Time) (List, Effect) {
	l = l.keys()
	switch e := ev.(type) {
	case PageRequested:
		return l.pageRequested(now)
	case PageFetched:
		return l.pageFetched(e.Page)
	case PageFailed:
		return l.pageFailed(e.Failure)
	case RefreshRequested:
		return l.refreshRequested(now)
	case Refreshed:
		return l.refreshed(e.Page)
	case Select:
		return l.selectTarget(e.Target)
	}
	return l, Effect{}
}

// keys is the frame with its walk's row key in place, which every fold below needs.
func (l List) keys() List {
	l.walk = l.w()
	return l
}

// pageRequested is the pagination trigger: one page, at the cursor the walk has reached. The walk
// refuses the request when a page is already out, when the walk has ended, when there is no cursor to
// carry, or when the last failure forbids a request at this time; the only effect it can produce is a
// page of this list's own timeline.
func (l List) pageRequested(now time.Time) (List, Effect) {
	next, effect := l.w().Page(FeedRequest{Kind: l.Kind, Cursor: l.w().Next()}, now)
	l.walk = next
	return l, effect
}

// pageFetched absorbs one page of the walk: the answer's posts in the answer's order, the page's own
// cursor, and the stall rule and de-duplication that come with them (walk.go).
func (l List) pageFetched(page domain.PostPage) (List, Effect) {
	l.walk = l.w().absorb(page, page.Posts, page.Next)
	return l, Effect{}
}

// pageFailed records a failure and changes nothing else: every post, every page and the selection stay
// exactly as they were, and the failure is what the status line prints until a fetch succeeds.
func (l List) pageFailed(failure domain.Failure) (List, Effect) {
	l.walk = l.w().pageFailed(failure)
	return l, Effect{}
}

// refreshRequested asks for page one again. The walk is gated as it is for a page; the difference is
// what the answer does — a refresh prepends and moves nothing, so a walk that has ended does not stop
// it.
func (l List) refreshRequested(now time.Time) (List, Effect) {
	next, effect := l.w().Refresh(FeedRequest{Kind: l.Kind}, now)
	l.walk = next
	return l, effect
}

// refreshed absorbs the answer to a refresh under the home rule: the posts the list does not already
// hold are prepended in the answer's order, how many there were is what the notice line reports, and
// the selection, the pages and the walk's cursor stay where they were. A refresh reads the top of the
// timeline, it does not move the reader.
func (l List) refreshed(page domain.PostPage) (List, Effect) {
	l.walk = l.w().refresh(page.Posts, page.Next, RefreshPrepend)
	return l, Effect{}
}

// selectTarget moves the selection onto a row of this frame's own row order. A target the frame does
// not hold moves nothing: the check matters because a refresh prepends posts, and every row moves
// down — a selection that silently re-pointed at whatever now sits at the old index would be the pane
// pointing at a post the reader did not choose.
func (l List) selectTarget(t Target) (List, Effect) {
	next, ok := l.w().SelectAt(l.Targets(), t)
	if !ok {
		return l, Effect{}
	}
	l.walk = next
	return l, Effect{}
}
