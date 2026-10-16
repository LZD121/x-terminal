package screen

import (
	"slices"
	"strings"
	"time"

	"github.com/LZD121/x-terminal/internal/domain"
)

// Target is the row the cursor is on: the row's own identity as the plain string the element
// supplies, plus the chain of containing posts when that row is a post quoted inside another.
//
// Path is outermost first and empty at top level, so a post quoted inside a post is one row of that
// post's block, and the same post quoted in two places is two distinct targets: the path is what
// tells them apart, and it is what makes a quoted post selectable — and therefore openable — with
// the same key every other row uses rather than with a key of its own.
//
// ID is a string and not a new id type: the walk keys a row by the string the element supplies (a
// post id, an account id) and nothing above the port parses one.
type Target struct {
	// ID is the row's own id, as the element spells it.
	ID string
	// Path is the chain of posts the row is quoted inside: the outermost containing post first, the
	// one that quotes the row last, and empty when the row is a row of the list itself.
	Path []string
}

// Top is the top-level row the target is inside: the row itself when it is a row of the list, and
// the outermost containing post when it is a post quoted inside one. It is what a frame looks the
// selection's block up by — the block a selected row is drawn in is the block its outermost
// container names, not the row's own id — and what the status line's position counts.
func (t Target) Top() Target {
	if len(t.Path) == 0 {
		return t
	}
	return Target{ID: t.Path[0]}
}

// Equal reports whether two targets name the same row. A Target carries a slice, so it is compared
// through this and never with ==.
func (t Target) Equal(other Target) bool {
	return t.ID == other.ID && slices.Equal(t.Path, other.Path)
}

// String names both parts of a target, so that a test failure identifies the row rather than printing
// a bare id that could be either of two rows.
func (t Target) String() string {
	if len(t.Path) == 0 {
		return t.ID
	}
	return t.ID + " (quoted by " + strings.Join(t.Path, ", ") + ")"
}

// Walk is one list's state, shared by every frame in the product: the rows in list order, the pages
// absorbed, the opaque cursor the walk continues from, the id-keyed dedupe set, the cursors already
// spent, the single-flight latch, the sticky failure, the selected row and the anchor.
//
// The two type parameters are what makes it shared. T is one row (a post in this phase, an account,
// a search result or a saved item in the phases after it) and P is the page value the frame's own
// read returns. The id function is the one thing the walk cannot derive for itself: it keys
// de-duplication, the spent cursors and the refresh diff on the string the element supplies.
//
// One implementation, extended in place. A fork of the dedupe set, the stall rule, the single-flight
// latch or the refresh diff is a defect rather than an alternative: the home list and the thread
// frame are the same machinery with different rows, and the phases after this one add their own rows
// to this file instead of writing a second list. TestTheWalkPolicyLivesInOneFile is what holds that
// rule as a test rather than as an intention.
//
// Nothing here performs anything: a walk takes values and returns values — no source, no file, no
// terminal, and no clock other than the one the caller passes.
type Walk[T any, P any] struct {
	// idOf names one row. It is set at construction, and a frame whose zero value is a usable state
	// fills it in at the seam through keys.
	idOf func(T) string

	// rows is the walk in list order: the answers' own order, never re-sorted. pages is every page
	// the walk has absorbed, kept so that the walk can tell "there is nothing more" from "nothing has
	// been read". next is the cursor the walk continues from, empty at the end of the walk.
	rows  []T
	pages []P
	next  domain.Cursor

	// seen is every row id the walk holds, and it is the only thing de-duplication is keyed on.
	// usedCursors is every cursor the walk has fetched a page with, which is how a cursor that comes
	// back a second time is caught.
	seen        map[string]bool
	usedCursors map[domain.Cursor]bool

	// inflight is the single-flight latch: while it is set, no update may ask for anything, because
	// one request is already out and a second one would be a burst the user did not ask for.
	// loadingOlder and refreshPending say which request the latch belongs to, so that the answer
	// which belongs to it is the one that releases it. requested is the cursor the in-flight page
	// request asked for: an answer that hands back the cursor it was asked with has not moved.
	inflight       bool
	loadingOlder   bool
	refreshPending bool
	requested      domain.Cursor

	// failure is the last request's failure, sticky until a fetch succeeds. newRows is how many rows
	// the last refresh brought. stalled ends the walk: the cursor repeated or the page added nothing,
	// and the walk will not ask again. headIDs is what the last refresh request was asked to diff
	// against — the ids the effect carried, kept so the walk can say what it asked for.
	failure domain.Failure
	newRows int
	stalled bool
	headIDs []string

	// selected is the row the cursor is on, and anchor is where that row's block head sits inside the
	// pane: its distance from the pane's first row. The arithmetic that computes the anchor stays in
	// internal/tui, where the pane geometry is; the walk stores the value the shell works out.
	selected Target
	anchor   int
}

// NewWalk starts an empty walk whose rows are named by idOf. A walk with no id function cannot key a
// row, so the frames pass theirs here — and at the seam (keys) for a frame whose zero value is a
// usable state.
func NewWalk[T any, P any](idOf func(T) string) Walk[T, P] {
	return Walk[T, P]{idOf: idOf}
}

// keys fills in the walk's id function when it has none. A frame's zero value is a usable state, so
// the id function is supplied where the frame and the walk meet rather than required at construction.
func (w Walk[T, P]) keys(idOf func(T) string) Walk[T, P] {
	if w.idOf == nil {
		w.idOf = idOf
	}
	return w
}

// key is one row's id, as the element spells it.
func (w Walk[T, P]) key(row T) string { return w.idOf(row) }

// Rows returns the walk in list order. The slice is the walk's own: a caller reads it and does not
// modify it. (No copy: a frame asks for this on every render, and copying the list per frame would be
// an allocation in the one path that must not have one.)
func (w Walk[T, P]) Rows() []T { return w.rows }

// Row returns the row at an index, and false when the walk does not hold that index.
func (w Walk[T, P]) Row(i int) (T, bool) {
	var zero T
	if i < 0 || i >= len(w.rows) {
		return zero, false
	}
	return w.rows[i], true
}

// Loaded is how many rows the walk holds.
func (w Walk[T, P]) Loaded() int { return len(w.rows) }

// Next is the cursor the walk continues from, empty at the end of the walk. It is carried in the
// status line's position field as the `+` that says there is more.
func (w Walk[T, P]) Next() domain.Cursor { return w.next }

// HasMore reports whether the walk may continue: a cursor is stored and the walk has not stalled.
func (w Walk[T, P]) HasMore() bool { return w.next != "" && !w.stalled }

// Stalled reports whether the walk has ended without reaching the end of the list: a cursor came back
// a second time, or a page contributed nothing. It is a flag and not a countdown: the walk never
// retries the same cursor, and nothing here waits for a timer to try.
func (w Walk[T, P]) Stalled() bool { return w.stalled }

// InFlight reports whether a request is out. No update issues another while it is true.
func (w Walk[T, P]) InFlight() bool { return w.inflight }

// LoadingOlder reports whether the request in flight is a page of the walk rather than a refresh.
func (w Walk[T, P]) LoadingOlder() bool { return w.loadingOlder }

// Failure is the last failure, FailureNone when there is none. It is sticky: only a successful fetch
// clears it, and nothing here runs on a timer.
func (w Walk[T, P]) Failure() domain.Failure { return w.failure }

// NewRows is how many rows the last refresh brought that the walk did not already hold. Zero is the
// documented "nothing new" notice, which is a different thing from no refresh having happened at all.
func (w Walk[T, P]) NewRows() int { return w.newRows }

// Selected is the row the cursor is on, empty when nothing is selected.
func (w Walk[T, P]) Selected() Target { return w.selected }

// Anchor is the selected block's head row's distance from the pane's first row.
func (w Walk[T, P]) Anchor() int { return w.anchor }

// WithAnchor returns the walk with the anchor the shell worked out.
func (w Walk[T, P]) WithAnchor(anchor int) Walk[T, P] {
	w.anchor = anchor
	return w
}

// Index is where a row sits in the walk, or -1 when the walk does not hold it. It is the walk's own
// row order that is searched, so a quoted post's own row is found through Target.Top, not here.
func (w Walk[T, P]) Index(id string) int {
	for i := range w.rows {
		if w.key(w.rows[i]) == id {
			return i
		}
	}
	return -1
}

// Holds reports whether the walk holds a row with this id. The empty id is held by no walk: nothing
// selected is not a row.
func (w Walk[T, P]) Holds(id string) bool { return id != "" && w.Index(id) >= 0 }

// started reports whether the walk has absorbed a page at all. It is the difference between "there is
// nothing more" and "nothing has been read", which is the difference the page gate turns on.
func (w Walk[T, P]) started() bool { return len(w.pages) > 0 }

// indexOf is where a target sits in a caller's row sequence, or -1 when the sequence does not hold it.
func indexOf(seq []Target, t Target) int {
	for i := range seq {
		if seq[i].Equal(t) {
			return i
		}
	}
	return -1
}

// Select moves the cursor onto a target, and a target whose top-level row the walk does not hold
// selects nothing rather than inventing a place: a refresh can take a row away, and a selection that
// silently re-pointed at whatever now sits where it was would be the pane pointing at a row the
// reader did not choose.
func (w Walk[T, P]) Select(t Target) Walk[T, P] {
	if !w.Holds(t.Top().ID) {
		return w
	}
	w.selected = t
	return w
}

// SelectAt moves the cursor onto a target that appears in the caller's own row sequence — the sequence a
// frame builds from its own rows — and reports whether it was there. A frame's own Select goes through
// this rather than through Select, because a frame's rows are more than the walk's own: a post quoted
// inside a selected block is a row of that block, and a conversation's rows include the posts above the
// reply stream. A target that is not one of the sequence's rows is not a row this frame can be on.
func (w Walk[T, P]) SelectAt(seq []Target, t Target) (Walk[T, P], bool) {
	if indexOf(seq, t) < 0 {
		return w, false
	}
	w.selected = t
	return w, true
}

// SelectNear is the refresh's rule for holding a selection: the target itself while the walk still
// holds the row it names, then the same id's shallowest occurrence, and nothing when neither is
// there — the caller decides what to fall back to, because only the caller knows what else its own
// rows contain.
func (w Walk[T, P]) SelectNear(t Target) Walk[T, P] {
	if w.Holds(t.Top().ID) {
		return w.Select(t)
	}
	// The same id anywhere at top level: a refresh can move a row out of the container the cursor was
	// on, and a row that is still there under another name is still the row the reader chose.
	if w.Holds(t.ID) {
		return w.Select(Target{ID: t.ID})
	}
	return w
}

// Move walks the caller's own row sequence by a step and clamps at both ends, so the selection never
// leaves the sequence: a step off the top lands on the first row and a step off the bottom lands on
// the last. A selection that is not in the sequence has no origin to move from, and nothing moves.
func (w Walk[T, P]) Move(seq []Target, step int) Walk[T, P] {
	cur := indexOf(seq, w.selected)
	if cur < 0 || step == 0 {
		return w
	}
	next, last := cur+step, len(seq)-1
	if next < 0 {
		next = 0
	}
	if next > last {
		next = last
	}
	w.selected = seq[next]
	return w
}

// AtEnd reports whether the selection's block is the last loaded row and the walk has somewhere to
// go: the state in which a shell asks for the next page. It is a question about the walk, never about
// the pane, which is why no pixel geometry is involved — and a row the walk does not hold (an
// ancestor, in a thread) is never at the end of it.
func (w Walk[T, P]) AtEnd(sel Target) bool {
	if len(w.rows) == 0 {
		return false
	}
	return sel.Top().ID == w.key(w.rows[len(w.rows)-1]) && w.HasMore()
}

// Page asks for one more page of the walk, and returns the walk with the single-flight latch set and
// the effect that asks for it. It is the only way a page request is born.
//
// It refuses when a request is out, when the walk has stalled, when the last failure forbids a
// request at now, and when there is no cursor to carry — an empty cursor is the end of the walk once
// a page has been absorbed, and page one before that, which is what lets a frame the shell has just
// pushed ask for its own first page through the same gate as any other.
func (w Walk[T, P]) Page(req Request, now time.Time) (Walk[T, P], Effect) {
	switch {
	case w.inflight:
		// A second key press while a page is out is not a queued request: a slow upstream would turn
		// a held-down key into a burst.
		return w, Effect{}
	case w.stalled:
		// The walk is over. Nothing asks the upstream for the same cursor again.
		return w, Effect{}
	case w.failure.Stopped:
		return w, Effect{}
	case w.next == "" && w.started():
		return w, Effect{}
	case w.blocked(now):
		return w, Effect{}
	}
	w.inflight, w.loadingOlder = true, true
	w.requested = w.next
	return w, Effect{Kind: EffectPage, Req: req, Cursor: w.next}
}

// Refresh asks for page one again, and returns the walk with the latch set and the effect that asks
// for it.
//
// It is gated like a page request, and on one thing less: a refresh does not stop the walk, so a
// stalled walk does not stop a refresh, and the ids the walk holds travel with the effect.
func (w Walk[T, P]) Refresh(req Request, now time.Time) (Walk[T, P], Effect) {
	switch {
	case w.inflight, w.failure.Stopped, w.blocked(now):
		return w, Effect{}
	}
	w.inflight, w.refreshPending = true, true
	w.headIDs = w.headIDsOf()
	return w, Effect{Kind: EffectRefresh, Req: req, Head: slices.Clone(w.headIDs)}
}

// absorb folds one page of the walk into it: the rows in the answer's order, where the walk then
// continues, and the page value itself, which the walk keeps so that no frame has to hold a second
// copy of its own pages.
//
// The rows are de-duplicated against every id the walk already holds, and what survives is appended
// in the order the answer carried it: the list order is the answer's order. The failure the answer
// resolved is cleared, and if the row contributed nothing unseen — or hands back a cursor the walk
// has already spent — the walk ends as stalled instead of asking for the same page again.
func (w Walk[T, P]) absorb(page P, rows []T, next domain.Cursor) Walk[T, P] {
	// The latches are released by the answer that belongs to them. A refresh's answer is a different
	// event, so a page arriving while one is pending is a page from before it: one request is still
	// out, and the shell must not be told otherwise.
	w.loadingOlder = false
	w.inflight = w.refreshPending
	used := w.requested
	w.requested = ""

	if used != "" {
		if w.usedCursors == nil {
			w.usedCursors = map[domain.Cursor]bool{}
		}
		w.usedCursors[used] = true
	}

	added := w.appendRows(rows)
	w.pages = append(w.pages, page)
	w.next, w.failure = next, domain.Failure{}
	w.defaultSelection()

	// A page that added nothing unseen is one that says "there is more" without having anything to
	// add; a page that hands back the cursor it was asked with, or one the walk has used before, is
	// the upstream pointing at a position the walk has already read. Either way the walk ends here
	// and the shell reports it, rather than looping until the request budget runs out.
	if next != "" && (added == 0 || next == used || w.usedCursors[next]) {
		w.stalled = true
	}
	return w
}

// RefreshMode is how a refresh's answer folds into a walk: the two lists in this program disagree
// about it, and they are right to.
type RefreshMode uint8

const (
	// RefreshPrepend is the home rule: the rows the walk does not already hold are prepended in the
	// answer's order, and the selection, the pages and the walk's cursor are untouched. A timeline is
	// a stream of new posts at the top, so what arrived is what is new.
	RefreshPrepend RefreshMode = iota
	// RefreshReplace is the thread's rule: this list is a ranking and re-reading it is the only
	// honest way to make it current, so the answer's rows become the walk's rows and the walk's
	// cursor becomes the answer's.
	RefreshReplace
)

// refresh folds the answer to a refresh into the walk.
//
// The count of rows the walk did not hold before is what the notice line reports — including the zero
// that means "nothing new", which is a successful answer and not a failure. The failure the answer
// resolved is cleared, and a walk with rows and no selection gets the first row, because a pane with
// rows and no selection has no row for the selection column and nowhere for the movement keys to
// start.
func (w Walk[T, P]) refresh(rows []T, next domain.Cursor, mode RefreshMode) Walk[T, P] {
	w.refreshPending, w.loadingOlder = false, false
	w.inflight = false

	switch mode {
	case RefreshReplace:
		// The rows are replaced, so the dedupe set is the answer's own: a row the answer dropped must
		// not stay "held" and stop a later page from bringing it back.
		seen := make(map[string]bool, len(rows))
		fresh := make([]T, 0, len(rows))
		added := 0
		for _, row := range rows {
			id := w.key(row)
			if seen[id] {
				continue
			}
			seen[id] = true
			if !w.seen[id] {
				added++
			}
			fresh = append(fresh, row)
		}
		w.rows, w.seen, w.newRows = fresh, seen, added
		w.next = next
		// The walk stands where the answer says it does, so the evidence that ended it is gone with
		// the cursor it was about: a stall is a fact about a cursor, and that cursor has been replaced
		// by a fresh read. A page that repeats a cursor stalls again on its own.
		w.stalled = false
	default:
		fresh := make([]T, 0, len(rows))
		for _, row := range rows {
			if w.marked(w.key(row)) {
				fresh = append(fresh, row)
			}
		}
		w.rows = append(fresh, w.rows...)
		w.newRows = len(fresh)
	}
	w.failure = domain.Failure{}
	w.defaultSelection()
	return w
}

// pageFailed records a failure and changes nothing else.
//
// Every row, every page and the selection stay exactly as they were: a page that fails decodes to
// nothing at all, so there is no half-page to append and nothing to take back. The failure is sticky
// until a fetch succeeds.
func (w Walk[T, P]) pageFailed(f domain.Failure) Walk[T, P] {
	w.inflight, w.loadingOlder, w.refreshPending = false, false, false
	w.requested = ""
	w.failure = f
	return w
}

// appendRows appends the rows the walk does not already hold, in order, and reports how many those
// were.
func (w *Walk[T, P]) appendRows(rows []T) int {
	added := 0
	for _, row := range rows {
		if w.marked(w.key(row)) {
			w.rows = append(w.rows, row)
			added++
		}
	}
	return added
}

// marked records a row id and reports whether the walk did not already hold it. It is the whole of
// de-duplication: one map, keyed on the id the element supplies, and no comparison of bodies, cursors
// or positions.
func (w *Walk[T, P]) marked(id string) bool {
	if w.seen == nil {
		w.seen = map[string]bool{}
	}
	if w.seen[id] {
		return false
	}
	w.seen[id] = true
	return true
}

// defaultSelection selects the first row when a walk has rows and nothing selected: a pane with rows
// and no selection has no row for the selection column and nowhere for the movement keys to start.
func (w *Walk[T, P]) defaultSelection() {
	if w.selected.ID == "" && len(w.rows) > 0 {
		w.selected = Target{ID: w.key(w.rows[0])}
	}
}

// headIDsOf is the ids of every row the walk holds, in list order: what a refresh asks the upstream
// to diff against. It is every loaded id rather than a leading window, because which of them the
// answer will mention is the upstream's business and not this program's guess.
func (w Walk[T, P]) headIDsOf() []string {
	if len(w.rows) == 0 {
		return nil
	}
	ids := make([]string, 0, len(w.rows))
	for i := range w.rows {
		ids = append(ids, w.key(w.rows[i]))
	}
	return ids
}

// blocked reports whether the last failure forbids issuing a request at now.
//
// Three rules, and they are the reason this takes a time at all:
//
//   - the soft ban stops the session: it blocks for the rest of the process, with no exception and no
//     timer (OPS-04);
//   - a failure whose remedy is "do not retry" blocks until something else resolves it — a successful
//     fetch, a re-copied cookie, a restart — and `Retry` is the caller's word for that (internal/cli
//     sets it from the state table);
//   - the rate-limited row is the one state with an end the upstream itself names: the reset instant.
//     Without a reset header there is no instant to wait for, and "no retry; reset unknown" means
//     exactly that.
func (w Walk[T, P]) blocked(now time.Time) bool {
	switch {
	case w.failure.Kind == domain.FailureNone:
		return false
	case w.failure.Stopped:
		return true
	case w.failure.Retry:
		return false
	case w.failure.Kind == domain.FailureRateLimited:
		return w.failure.Reset.IsZero() || now.Before(w.failure.Reset)
	default:
		return true
	}
}

// Request is what an effect names as the read it wants, as a value. It is a closed union: the marker
// method is unexported, so a request can only be one of the cases declared in this file — and a frame
// kind added later adds a case here rather than a field on Effect that the shell has to interpret.
//
// At re-issues the request at another position: the same read, with the walk's cursor substituted.
// It is on the union rather than on the cases alone because a frame that holds its request as a
// Request — the results frame is one — re-issues it for the next page without rebuilding the case,
// and without its own file ever naming a cursor.
type Request interface {
	request()
	// At returns this request at c: the same read, with the cursor substituted.
	At(c domain.Cursor) Request
}

// FeedRequest is one page of a home timeline: which feed, and where in it.
type FeedRequest struct {
	// Kind is which home timeline the page belongs to.
	Kind domain.HomeKind
	// Cursor is the position to fetch: the next page's cursor for a page request, and empty — page
	// one — for a refresh.
	Cursor domain.Cursor
}

// At returns this request at c: the same read, with the walk's cursor substituted. It is how a
// frame re-issues its own request for the next page without rebuilding the case.
func (r FeedRequest) At(c domain.Cursor) Request {
	r.Cursor = c
	return r
}

// ThreadRequest is one page of a conversation: the post the conversation is of, and where the reply
// walk has reached.
type ThreadRequest struct {
	// Focal is the post the conversation is of. It is the same post page one and every later page are
	// asked for, so that an answer cannot be absorbed by the wrong conversation.
	Focal domain.PostID
	// Cursor is where the reply walk continues from, empty for page one.
	Cursor domain.Cursor
}

// At returns this request at c.
func (r ThreadRequest) At(c domain.Cursor) Request {
	r.Cursor = c
	return r
}

// SearchRequest is one page of a keyword search: the query, the product, and where the walk has
// reached. It is the one shared edit this phase makes to the sealed union — append-only, no new
// walk and no new latch.
type SearchRequest struct {
	Query   string
	Product domain.SearchProduct
	Cursor  domain.Cursor
}

// At returns this request at c.
func (r SearchRequest) At(c domain.Cursor) Request {
	r.Cursor = c
	return r
}

func (FeedRequest) request()   {}
func (ThreadRequest) request() {}
func (SearchRequest) request() {}

// PeopleRequest is one page of a people search: the query and where the walk has
// reached. It is its own case because a page of accounts is not a post SearchRequest
// — the walk is shared, the request that names the read is not.
type PeopleRequest struct {
	Query  string
	Cursor domain.Cursor
}

// At returns this request at c.
func (r PeopleRequest) At(c domain.Cursor) Request {
	r.Cursor = c
	return r
}

func (PeopleRequest) request() {}

// ProfileRequest is the profile frame's first read: one account's header by handle.
// It carries no cursor — a header is one object — and At is a no-op on the cursor so
// the sealed Request contract stays uniform across cases.
type ProfileRequest struct {
	Handle string
}

// At returns this request unchanged: a header has no position.
func (r ProfileRequest) At(c domain.Cursor) Request { return r }

// UserPostsRequest is one page of one account's recent posts, addressed by the
// numeric id the header returned. The id is the only thing that addresses the
// read: nothing above the port parses one and nothing guesses one.
type UserPostsRequest struct {
	ID     string
	Cursor domain.Cursor
}

// At returns this request at c.
func (r UserPostsRequest) At(c domain.Cursor) Request {
	r.Cursor = c
	return r
}

func (ProfileRequest) request()   {}
func (UserPostsRequest) request() {}

// ResultsFetched is one page of a results frame arriving, and ResultsRefreshed is the answer to
// its refresh. Both carry the walk's vocabulary already read out of the page — the rows and where
// the walk continues — because the frame is generic over the page value and the shell is the one
// place that knows its concrete type. They are declared beside the walk for the same reason: the
// vocabulary is the walk's, and a results frame's own file never names a cursor.
type ResultsFetched[T any, P any] struct {
	Page P
	Rows []T
	Next domain.Cursor
}

// ResultsRefreshed is the answer to a results frame's refresh: page one again, in the same shape.
type ResultsRefreshed[T any, P any] struct {
	Page P
	Rows []T
	Next domain.Cursor
}

func (ResultsFetched[T, P]) event()   {}
func (ResultsRefreshed[T, P]) event() {}

// PostTargets is postTargets at the shell's seam: the results frame's rows are posts, and they take
// the same row order as the home list's — one implementation, exported so the shell's dispatch does
// not re-derive it.
func PostTargets(posts []domain.Post) []Target { return postTargets(posts) }

// UserTargets is the people list's row order: one target per account, keyed by the account id.
func UserTargets(users []domain.User) []Target {
	out := make([]Target, 0, len(users))
	for _, u := range users {
		out = append(out, Target{ID: u.ID})
	}
	return out
}

// PostFor is postFor at the shell's seam: the copy of a post a results frame holds for a target,
// quote chain included.
func PostFor(posts []domain.Post, t Target) (domain.Post, bool) { return postFor(posts, t) }

// postTargets is the row order a post list has, shared by the home list and the thread frame: for
// each post in order, the post itself and then its quote chain — the post, its quote, the quote's
// quote — each with the containing posts in its Path.
//
// A quoted post is therefore one step from the post that quotes it in every list, and a post quoted
// twice is two rows with two different paths. A quote with no id is not a row (nothing can key it) and
// ends the chain it is in.
func postTargets(posts []domain.Post) []Target {
	out := make([]Target, 0, len(posts))
	for _, post := range posts {
		out = append(out, Target{ID: string(post.ID)})
		path := []string{string(post.ID)}
		for q := post.Quote; q != nil && q.ID != ""; q = q.Quote {
			out = append(out, Target{ID: string(q.ID), Path: slices.Clone(path)})
			path = append(path, string(q.ID))
		}
	}
	return out
}

// postFor resolves a target against a list of posts: the row the target's top-level id names, then
// that row's quote chain down to the target's own id. It is the copy of a post a frame holds — what a
// shell opens a frame from, so that opening a post or a quoted post costs no request.
func postFor(posts []domain.Post, t Target) (domain.Post, bool) {
	for _, post := range posts {
		if string(post.ID) != t.Top().ID {
			continue
		}
		node, path := post, []string{string(post.ID)}
		for _, step := range t.Path {
			if path[len(path)-1] != step || node.Quote == nil {
				return domain.Post{}, false
			}
			node = *node.Quote
			path = append(path, string(node.ID))
		}
		if string(node.ID) == t.ID {
			return node, true
		}
		return domain.Post{}, false
	}
	return domain.Post{}, false
}
