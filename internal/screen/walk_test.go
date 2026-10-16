package screen

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/LZD121/x-terminal/internal/domain"
)

// walkNow is the instant every walk update in this file is called with. It is a value, not the wall
// clock, because the walk has no clock: a rate limit's reset is judged against the caller's time, and
// every test here says which time it means.
var walkNow = time.Date(2026, 10, 14, 9, 12, 0, 0, time.UTC)

// walkRows builds a page's worth of rows from ids, which is all a row needs to be for a walk: the
// walk keys de-duplication on the id the element supplies and reads nothing else.
func walkRows(ids ...string) []domain.Post {
	rows := make([]domain.Post, 0, len(ids))
	for _, id := range ids {
		rows = append(rows, domain.Post{ID: domain.PostID(id)})
	}
	return rows
}

// idsOf is the ids of a walk's rows, in list order: what most of these tests compare.
func idsOf(rows []domain.Post) []string {
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, string(row.ID))
	}
	return out
}

// newWalk is an empty walk of posts, the element this phase walks.
func newWalk() Walk[domain.Post, domain.PostPage] {
	return NewWalk[domain.Post, domain.PostPage](postKey)
}

// walkState is a walk's observable state as one comparable value: what "nothing changed" means in a
// test. A Walk cannot be compared with reflect.DeepEqual — it holds its own id function — so the tests
// compare this instead of the struct.
type walkState struct {
	rows                   []string
	next                   domain.Cursor
	inFlight, loadingOlder bool
	selected               string
	anchor, newRows        int
	stalled                bool
	failure                domain.Failure
	head, spent            []string
}

// stateOf reads that state off a walk, with the two sets that have no order of their own sorted so that
// two equal states compare equal. It is generic over the walk's own two type parameters, so the home
// frame and the thread frame are compared by the same helper.
func stateOf[T any, P any](w Walk[T, P]) walkState {
	spent := make([]string, 0, len(w.usedCursors))
	for c := range w.usedCursors {
		spent = append(spent, string(c))
	}
	slices.Sort(spent)
	heads := slices.Clone(w.headIDs)
	if heads == nil {
		heads = []string{}
	}
	rows := make([]string, 0, len(w.rows))
	for i := range w.rows {
		rows = append(rows, w.key(w.rows[i]))
	}
	return walkState{
		rows:         rows,
		next:         w.Next(),
		inFlight:     w.InFlight(),
		loadingOlder: w.LoadingOlder(),
		selected:     w.Selected().String(),
		anchor:       w.Anchor(),
		newRows:      w.NewRows(),
		stalled:      w.Stalled(),
		failure:      w.Failure(),
		head:         heads,
		spent:        spent,
	}
}

// feedReq is a request value of the kind a home frame builds: what the walk carries out with an
// effect, and never reads.
func feedReq(c domain.Cursor) FeedRequest {
	return FeedRequest{Kind: domain.Following, Cursor: c}
}

// TestWalkAbsorbsEachRowOnce is de-duplication: a row the walk already holds is not appended a second
// time, the order is the answers' own, and a refresh's count is a refresh's business — a page does not
// touch it.
func TestWalkAbsorbsEachRowOnce(t *testing.T) {
	w := newWalk()
	w = w.absorb(domain.PostPage{}, walkRows("1", "2", "3"), "c1")

	if got, want := idsOf(w.Rows()), []string{"1", "2", "3"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("the walk holds %v, want %v: the answers' order is the walk's order", got, want)
	}
	if w.Loaded() != 3 {
		t.Errorf("the walk holds %d rows, want 3", w.Loaded())
	}
	if w.NewRows() != 0 {
		t.Errorf("a page set the refresh count to %d, want it left at 0: that count belongs to a refresh", w.NewRows())
	}
	if w.Selected().ID != "1" {
		t.Errorf("the walk selected %q, want the first row: a walk with rows and nothing selected has no row for the selection column and nowhere for the movement keys to start", w.Selected())
	}
	if !w.Holds("2") || w.Holds("no-such-row") || w.Holds("") {
		t.Error("Holds disagrees with the rows the walk carries (the empty id is a row no walk holds)")
	}
	if w.Index("3") != 2 || w.Index("no-such-row") != -1 {
		t.Errorf("Index reports %d for the third row and %d for a row the walk does not hold", w.Index("3"), w.Index("no-such-row"))
	}
	if row, ok := w.Row(1); !ok || string(row.ID) != "2" {
		t.Errorf("Row(1) is %q (ok=%v), want the second row", row.ID, ok)
	}
	if _, ok := w.Row(3); ok {
		t.Error("Row reports a row at an index the walk does not hold")
	}

	// The overlap: two ids the walk holds and one it does not.
	w = w.absorb(domain.PostPage{}, walkRows("2", "3", "4"), "c2")
	if got, want := idsOf(w.Rows()), []string{"1", "2", "3", "4"}; !reflect.DeepEqual(got, want) {
		t.Errorf("after an overlapping page the walk holds %v, want %v: an id it already holds is not added twice", got, want)
	}
	if w.Loaded() != 4 {
		t.Errorf("the walk holds %d rows after an overlapping page, want 4", w.Loaded())
	}
	if w.Next() != "c2" {
		t.Errorf("the stored cursor is %q, want the answer's own", w.Next())
	}
}

// TestWalkStallsOnAPageThatAddsNothingUnseen is one half of the stall rule: an answer that says there
// is more while carrying nothing the walk does not already hold has not advanced.
func TestWalkStallsOnAPageThatAddsNothingUnseen(t *testing.T) {
	w := newWalk()
	w = w.absorb(domain.PostPage{}, walkRows("1", "2"), "c1")

	next, effect := w.Page(feedReq(w.Next()), walkNow)
	if effect.Kind != EffectPage || effect.Cursor != "c1" {
		t.Fatalf("the page request produced %v, want one page at the cursor the walk holds", effect)
	}
	next = next.absorb(domain.PostPage{}, walkRows("1", "2"), "c2")

	if !next.Stalled() {
		t.Fatal("a page that added no unseen row did not end the walk: the shell would ask for the same page again until the request budget ran out")
	}
	if next.HasMore() {
		t.Error("HasMore is true on a walk that has ended, so the position field would keep its + and imply there is more to read")
	}
	if next.Failure().Kind != domain.FailureNone {
		t.Error("a walk that ended as stalled is not a failure: nothing went wrong, the walk simply ended")
	}
	if next.LoadingOlder() || next.InFlight() {
		t.Error("the latches are still set after the answer arrived")
	}
}

// TestWalkStallsOnACursorThatComesBack is the other half: a page that hands back the cursor it was
// asked with is the upstream pointing at a position the walk has already read, whether or not it
// carried rows the walk did not hold.
func TestWalkStallsOnACursorThatComesBack(t *testing.T) {
	w := newWalk()
	w = w.absorb(domain.PostPage{}, walkRows("1"), "c1")

	next, effect := w.Page(feedReq(w.Next()), walkNow)
	if effect.Kind != EffectPage {
		t.Fatalf("the page request produced %v, want a page", effect)
	}
	next = next.absorb(domain.PostPage{}, walkRows("2"), "c1")

	if !next.Stalled() {
		t.Fatal("a page that repeated the cursor it was asked with did not end the walk")
	}
	if got, want := idsOf(next.Rows()), []string{"1", "2"}; !reflect.DeepEqual(got, want) {
		t.Errorf("the walk holds %v, want %v: a page that repeats a cursor still contributes the rows it carried", got, want)
	}

	// And the walk never retries the cursor it ended on.
	calls := 5
	for range calls {
		again, effect := next.Page(feedReq(next.Next()), walkNow)
		if effect.Kind != EffectNone {
			t.Fatalf("the walk asked for another page after it ended: %v", effect)
		}
		if again.InFlight() {
			t.Fatal("a refused page request latched the walk")
		}
	}
}

// TestWalkSingleFlight is the latch and its answers: one request at a time, the answer that belongs to
// a request releases it, and an answer from the other kind does not.
func TestWalkSingleFlight(t *testing.T) {
	w := newWalk()
	w = w.absorb(domain.PostPage{}, walkRows("1"), "c1")

	out, effect := w.Page(feedReq(w.Next()), walkNow)
	if effect.Kind != EffectPage || !out.InFlight() || !out.LoadingOlder() {
		t.Fatalf("the page request produced %v with inflight=%v loadingOlder=%v, want a latched page request", effect, out.InFlight(), out.LoadingOlder())
	}
	if got, ok := effect.Req.(FeedRequest); !ok || got.Cursor != "c1" {
		t.Errorf("the effect carries %#v, want the request the frame built", effect.Req)
	}

	again, second := out.Page(feedReq(out.Next()), walkNow)
	if second.Kind != EffectNone {
		t.Fatalf("a press while a page was out produced %v, want nothing: a held-down key must not become a burst", second)
	}
	if !reflect.DeepEqual(stateOf(out), stateOf(again)) {
		t.Error("a refused page request changed the walk")
	}

	released := out.absorb(domain.PostPage{}, walkRows("2"), "c2")
	if released.InFlight() || released.LoadingOlder() {
		t.Error("the page's own answer did not release the latch")
	}

	// A refresh's answer is a different event: an answer arriving while one is out belongs to
	// something else, and the latch it does not own stays set.
	refreshing, refreshEffect := released.Refresh(feedReq(""), walkNow)
	if refreshEffect.Kind != EffectRefresh || !refreshing.InFlight() {
		t.Fatalf("the refresh produced %v with inflight=%v, want a latched refresh", refreshEffect, refreshing.InFlight())
	}
	if len(refreshEffect.Head) != released.Loaded() {
		t.Errorf("the refresh carried %d head ids, want the %d rows the walk holds", len(refreshEffect.Head), released.Loaded())
	}
	stale := refreshing.absorb(domain.PostPage{}, walkRows("3"), "c3")
	if !stale.InFlight() {
		t.Error("a page's answer released a refresh's latch: the shell would then be free to issue a request while one is still out")
	}
	if stale.LoadingOlder() {
		t.Error("a page's answer marked the in-flight request as a page of the walk")
	}
	done := refreshing.refresh(nil, "", RefreshPrepend)
	if done.InFlight() {
		t.Error("the refresh's own answer did not release the latch")
	}
}

// TestWalkGateBlocksWhileTheStateForbidsARequest covers the gate the walk shares with the shell: a
// soft ban stops the process, a rate limit stops it until the instant the upstream named, and a
// retryable failure stops nothing.
func TestWalkGateBlocksWhileTheStateForbidsARequest(t *testing.T) {
	w := newWalk()
	w = w.absorb(domain.PostPage{}, walkRows("1"), "c1")

	ban := w.pageFailed(domain.Failure{Kind: domain.FailureSoftBan, Stopped: true})
	for _, at := range []time.Time{walkNow, walkNow.Add(time.Hour), walkNow.Add(365 * 24 * time.Hour)} {
		if _, effect := ban.Page(feedReq(ban.Next()), at); effect.Kind != EffectNone {
			t.Fatalf("a page was asked for at %s after the soft ban: %v", at, effect)
		}
		if _, effect := ban.Refresh(feedReq(""), at); effect.Kind != EffectNone {
			t.Fatalf("a refresh was asked for at %s after the soft ban: %v", at, effect)
		}
	}
	if ban.Next() == "" {
		t.Error("the soft ban cleared the walk's cursor: stopping the session is not the same as reaching the end of the list")
	}

	reset := walkNow.Add(15 * time.Minute)
	limited := w.pageFailed(domain.Failure{Kind: domain.FailureRateLimited, Reset: reset})
	if _, effect := limited.Page(feedReq(limited.Next()), walkNow); effect.Kind != EffectNone {
		t.Errorf("a page was asked for before the reset: %v", effect)
	}
	if _, effect := limited.Page(feedReq(limited.Next()), reset.Add(-time.Nanosecond)); effect.Kind != EffectNone {
		t.Errorf("a page was asked for an instant before the reset: %v", effect)
	}
	if _, effect := limited.Page(feedReq(limited.Next()), reset); effect.Kind != EffectPage {
		t.Errorf("at the reset the walk asked for nothing: %v", effect)
	}

	// No reset header is not a reset in the past: there is no instant to wait for.
	unknown := w.pageFailed(domain.Failure{Kind: domain.FailureRateLimited, Detail: "wait until the reset time from x-rate-limit-reset"})
	if _, effect := unknown.Page(feedReq(unknown.Next()), walkNow.Add(24*time.Hour)); effect.Kind != EffectNone {
		t.Errorf("a rate limit with no reset instant produced %v, want nothing", effect)
	}

	// And a state whose remedy is "backoff, then retry" does not block at all.
	shed := w.pageFailed(domain.Failure{Kind: domain.FailureLoadShed, Retry: true})
	if _, effect := shed.Page(feedReq(shed.Next()), walkNow); effect.Kind != EffectPage {
		t.Errorf("a load-shed failure produced %v, want the page request it retries with", effect)
	}
}

// TestWalkAsksForItsFirstPageOnce is the one case where a page request carries no cursor: a walk that
// has never absorbed a page asks for page one, and a walk that has absorbed one and reached the end of
// the list asks for nothing. It is how a frame the shell has just pushed gets its own first page
// through the same gate as every later one.
func TestWalkAsksForItsFirstPageOnce(t *testing.T) {
	w := newWalk()
	first, effect := w.Page(feedReq(""), walkNow)
	if effect.Kind != EffectPage || effect.Cursor != "" {
		t.Fatalf("a walk that has read nothing produced %v, want page one with the empty cursor", effect)
	}
	if _, second := first.Page(feedReq(""), walkNow); second.Kind != EffectNone {
		t.Errorf("a walk with a first page out asked for another: %v", second)
	}

	// Page one comes back with no cursor at all: the walk is at the end of the list and says so by
	// asking for nothing.
	end := first.absorb(domain.PostPage{}, walkRows("1"), "")
	if end.HasMore() || end.Stalled() {
		t.Errorf("an answer with no cursor left the walk with HasMore=%v Stalled=%v, want neither", end.HasMore(), end.Stalled())
	}
	if _, effect := end.Page(feedReq(""), walkNow); effect.Kind != EffectNone {
		t.Errorf("a walk at the end of its list asked for another page: %v", effect)
	}
}

// TestWalkRefreshPrependAgainstRefreshReplace runs the two refresh rules over the same rows: the home
// list prepends what it did not hold and moves nothing else, and the thread's replace re-reads the
// whole list and stands where the answer says it does.
func TestWalkRefreshPrependAgainstRefreshReplace(t *testing.T) {
	pre := newWalk()
	pre = pre.absorb(domain.PostPage{}, walkRows("1", "2"), "c1")
	pre = pre.Select(Target{ID: "2"})
	pre, effect := pre.Refresh(feedReq(""), walkNow)
	if effect.Kind != EffectRefresh {
		t.Fatalf("the refresh produced %v, want a refresh", effect)
	}
	pre = pre.refresh(walkRows("9", "1"), "", RefreshPrepend)

	if got, want := idsOf(pre.Rows()), []string{"9", "1", "2"}; !reflect.DeepEqual(got, want) {
		t.Errorf("the prepend left %v, want %v: the rows the walk did not hold go in front, in the answer's order", got, want)
	}
	if pre.NewRows() != 1 {
		t.Errorf("the prepend counted %d new rows, want the 1 it did not already hold", pre.NewRows())
	}
	if pre.Selected().ID != "2" {
		t.Errorf("the prepend moved the selection to %q: a refresh reads the top of the list, it does not move the reader", pre.Selected())
	}
	if pre.Next() != "c1" {
		t.Errorf("the prepend moved the walk's cursor to %q, want c1: the refresh does not move the walk", pre.Next())
	}
	if !pre.Holds("9") {
		t.Error("the prepended row is not held, so a later page would add it a second time")
	}

	repl := newWalk()
	repl = repl.absorb(domain.PostPage{}, walkRows("1", "2"), "c1")
	repl = repl.Select(Target{ID: "2"})
	repl, effect = repl.Refresh(feedReq(""), walkNow)
	if effect.Kind != EffectRefresh {
		t.Fatalf("the refresh produced %v, want a refresh", effect)
	}
	repl = repl.refresh(walkRows("9", "1"), "c2", RefreshReplace)

	if got, want := idsOf(repl.Rows()), []string{"9", "1"}; !reflect.DeepEqual(got, want) {
		t.Errorf("the replace left %v, want %v: this list is a ranking and re-reading it is the only honest way to make it current", got, want)
	}
	if repl.NewRows() != 1 {
		t.Errorf("the replace counted %d rows it did not hold before, want 1: the row it dropped is not one of them, and the row it kept is not new", repl.NewRows())
	}
	if repl.Next() != "c2" {
		t.Errorf("the walk continues from %q, want the answer's own cursor", repl.Next())
	}
	if repl.Holds("2") {
		t.Error("a row the answer dropped is still held: a later page could never bring it back")
	}
	if repl.Selected().ID != "2" {
		t.Errorf("the replace moved the selection on its own to %q; holding it is the caller's rule, through SelectNear", repl.Selected())
	}
}

// TestWalkSelectNearHoldsASelectionByTarget is the refresh's selection rule: the target while its row
// is still there, the same id at top level next, and otherwise the selection is left for the caller to
// fall back from.
func TestWalkSelectNearHoldsASelectionByTarget(t *testing.T) {
	quoted := domain.Post{ID: "1", Quote: &domain.Post{ID: "q"}}
	w := newWalk()
	w = w.absorb(domain.PostPage{}, []domain.Post{quoted}, "")
	w = w.Select(Target{ID: "q", Path: []string{"1"}})

	held := w.SelectNear(Target{ID: "q", Path: []string{"1"}})
	if !held.Selected().Equal(Target{ID: "q", Path: []string{"1"}}) {
		t.Errorf("the selection came back as %s, want the target it was: the row is still there under the same container", held.Selected())
	}

	// The row moved out of the container the cursor was on: the same id at top level is the fallback.
	loose := w.SelectNear(Target{ID: "1", Path: []string{"gone"}})
	if !loose.Selected().Equal(Target{ID: "1"}) {
		t.Errorf("the selection came back as %s, want the row %q at top level", loose.Selected(), "1")
	}

	// Neither is there: the walk leaves the value alone, and the caller falls back further — only the
	// caller knows what else its own rows hold.
	gone := w.SelectNear(Target{ID: "vanished", Path: []string{"gone"}})
	if !gone.Selected().Equal(w.Selected()) {
		t.Errorf("the selection became %s, want it left as %s", gone.Selected(), w.Selected())
	}
}

// TestWalkMoveClampsAtBothEnds is the movement rule: a step walks the caller's own row sequence and
// never leaves it, and a target the sequence does not hold is not a row to be on.
func TestWalkMoveClampsAtBothEnds(t *testing.T) {
	w := newWalk()
	w = w.absorb(domain.PostPage{}, walkRows("1", "2", "3"), "")
	seq := postTargets(w.Rows())
	if len(seq) != 3 {
		t.Fatalf("the sequence has %d rows, want 3", len(seq))
	}

	if got := w.Move(seq, -5); got.Selected().ID != "1" {
		t.Errorf("a step off the top landed on %s, want the first row", got.Selected())
	}
	if got := w.Move(seq, 99); got.Selected().ID != "3" {
		t.Errorf("a step off the bottom landed on %s, want the last row", got.Selected())
	}
	if got := w.Move(seq, 0); got.Selected().ID != "1" {
		t.Errorf("a zero step moved the selection to %s, want it left where it was", got.Selected())
	}
	atLast := w.Select(Target{ID: "3"})
	if got := atLast.Move(seq, -1); got.Selected().ID != "2" {
		t.Errorf("one step back from the last row landed on %s, want the second", got.Selected())
	}

	// A target the frame's own rows do not hold is not a row of this sequence.
	if _, ok := w.SelectAt(seq, Target{ID: "q", Path: []string{"1"}}); ok {
		t.Error("SelectAt accepted a target the sequence does not hold")
	}
	deep := newWalk()
	deep = deep.absorb(domain.PostPage{}, []domain.Post{{ID: "1", Quote: &domain.Post{ID: "q"}}}, "")
	quotedTarget := Target{ID: "q", Path: []string{"1"}}
	next, ok := deep.SelectAt(postTargets(deep.Rows()), quotedTarget)
	if !ok {
		t.Fatal("SelectAt refused a quoted row that is one of the frame's own rows: a quote has to be selectable for enter to open it")
	}
	if !next.Selected().Equal(quotedTarget) {
		t.Errorf("the selection is %s, want the quoted row %s", next.Selected(), quotedTarget)
	}

	// A selection that is not in the sequence has no origin, so nothing moves.
	orphan := deep.Select(Target{ID: "1"})
	if got := orphan.Move(postTargets(walkRows("7", "8")), 1); !got.Selected().Equal(orphan.Selected()) {
		t.Errorf("a move from a row that is not in the sequence landed on %s", got.Selected())
	}
}

// TestWalkAtEndIsTheLastLoadedRow is the pagination trigger as a question about the walk: the
// selection's block is the last loaded row and the walk has somewhere to go.
func TestWalkAtEndIsTheLastLoadedRow(t *testing.T) {
	w := newWalk()
	w = w.absorb(domain.PostPage{}, walkRows("1", "2"), "c1")

	if w.AtEnd(Target{ID: "1"}) {
		t.Error("AtEnd is true with the selection on a row above the last one")
	}
	if w.AtEnd(Target{ID: "3"}) {
		t.Error("AtEnd is true for a row the walk does not hold: an ancestor is not the end of a walk")
	}
	if !w.AtEnd(Target{ID: "2"}) {
		t.Error("AtEnd is false with the selection on the last loaded row and a cursor stored")
	}
	if !w.AtEnd(Target{ID: "q", Path: []string{"2"}}) {
		t.Error("AtEnd is false for a quoted row inside the last loaded row: the question is which block the selection is in")
	}

	stalled := w.absorb(domain.PostPage{}, walkRows("1"), "c1")
	if stalled.AtEnd(Target{ID: "2"}) {
		t.Error("AtEnd is true on a walk that has ended: there is nowhere to go")
	}
	ended := w.absorb(domain.PostPage{}, walkRows("3"), "")
	if ended.AtEnd(Target{ID: "3"}) {
		t.Error("AtEnd is true at the end of the list, where there is no cursor to page with")
	}
	empty := newWalk().absorb(domain.PostPage{}, nil, "c1")
	if empty.AtEnd(Target{ID: "1"}) {
		t.Error("AtEnd is true on a walk with no rows")
	}
}

// TestWalkPolicyLivesInOneFile is the one-implementation rule as a test. The identifiers that make up
// the walk's policy — the spent-cursor set, the dedupe set, the single-flight latch, the refresh diff —
// are named in walk.go and nowhere else in the package, because a frame that grew its own copy of one
// of them is the duplication this file exists to prevent. A later phase that forks the machinery fails
// here rather than in a review.
func TestWalkPolicyLivesInOneFile(t *testing.T) {
	// The patterns are the code that would be a second implementation: assignments to the latch, the
	// stall flag, the counters and the id map, not the words a comment may use.
	policy := []string{
		"usedCursors",
		"refreshPending",
		"loadingOlder",
		"newRows",
		"headIDs",
		"stalled = ",
		"inflight = ",
	}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("listing the package's sources: %v", err)
	}
	if len(files) < 2 {
		t.Fatalf("the package lists %d sources: this test is not looking at the package", len(files))
	}
	for _, file := range files {
		if file == "walk.go" || strings.HasSuffix(file, "_test.go") {
			continue
		}
		body, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("reading %s: %v", file, err)
		}
		for _, name := range policy {
			if strings.Contains(string(body), name) {
				t.Errorf("%s names the walk's %q: the dedupe set, the latch, the stall flag and the refresh diff live in walk.go alone", file, name)
			}
		}
	}
}

// TestPostTargetsWalksTheQuoteChain pins the row order a post list has: the post, then its quote, then
// the quote's quote, each with the containing posts in its path — so a quoted row is one step from the
// row that quotes it.
func TestPostTargetsWalksTheQuoteChain(t *testing.T) {
	deep := domain.Post{ID: "a", Quote: &domain.Post{ID: "b", Quote: &domain.Post{ID: "c"}}}
	got := postTargets([]domain.Post{deep})
	want := []Target{
		{ID: "a"},
		{ID: "b", Path: []string{"a"}},
		{ID: "c", Path: []string{"a", "b"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("the row order is %v, want %v: a post whose chain is two deep is three rows", got, want)
	}
	if len(got) != 1+2 {
		t.Errorf("a chain two deep produced %d rows, want 1 + quoteDepth", len(got))
	}

	flat := domain.Post{ID: "p", Quote: &domain.Post{ID: "q"}}
	targets := postTargets([]domain.Post{flat, {ID: "z"}})
	if len(targets) != 3 {
		t.Fatalf("a list of two posts, one of them quoting, produced %d rows, want 3", len(targets))
	}
	if targets[1].ID != "q" || !reflect.DeepEqual(targets[1].Path, []string{"p"}) {
		t.Errorf("the quoted row is %v, want q quoted by p", targets[1])
	}
	if targets[2].ID != "z" || len(targets[2].Path) != 0 {
		t.Errorf("the second block's row is %v, want z at top level", targets[2])
	}

	// A quote with no id is not a row: nothing can key it, and it ends the chain it is in.
	if n := len(postTargets([]domain.Post{{ID: "p", Quote: &domain.Post{Text: "no id"}}})); n != 1 {
		t.Errorf("a quote with no id produced %d rows, want the one row that has an id", n)
	}
}

// TestPostTargetsTheSamePostQuotedTwice is the reason a target carries a path: the same post quoted
// in two places is two rows, and the path is what tells them apart — which is what makes each of them
// separately selectable and separately openable.
func TestPostTargetsTheSamePostQuotedTwice(t *testing.T) {
	quote := &domain.Post{ID: "q"}
	posts := []domain.Post{
		{ID: "p1", Quote: quote},
		{ID: "p2", Quote: quote},
	}
	targets := postTargets(posts)
	if len(targets) != 4 {
		t.Fatalf("the two blocks produced %d rows, want 4", len(targets))
	}
	first, second := targets[1], targets[3]
	if first.ID != second.ID {
		t.Fatalf("the two quoted rows are %v and %v, want the same post quoted in two places", first, second)
	}
	if first.Equal(second) {
		t.Fatalf("the same post quoted in two places produced one target: %v", first)
	}
	if !reflect.DeepEqual(first.Path, []string{"p1"}) || !reflect.DeepEqual(second.Path, []string{"p2"}) {
		t.Errorf("the two paths are %v and %v, want each row named by the post that quotes it", first.Path, second.Path)
	}
}

// TestTargetTopNamesTheContainingBlock is the lookup a frame's geometry is made of: the top-level row
// is the row itself at top level and the outermost containing post for a quoted row, so the block a
// selected row is drawn in (and the position the status line counts) does not move when the cursor
// steps into a quote.
func TestTargetTopNamesTheContainingBlock(t *testing.T) {
	if got := (Target{ID: "p"}).Top(); got.ID != "p" || len(got.Path) != 0 {
		t.Errorf("a top-level target's Top is %v, want itself", got)
	}
	deep := Target{ID: "c", Path: []string{"a", "b"}}
	if got := deep.Top(); got.ID != "a" || len(got.Path) != 0 {
		t.Errorf("the quoted row's Top is %v, want the outermost containing post a", got)
	}
	if got := (Target{ID: "q", Path: []string{"p"}}).String(); !strings.Contains(got, "q") || !strings.Contains(got, "p") {
		t.Errorf("String() is %q, want both parts of the row so that a failure identifies it", got)
	}
	if got := (Target{ID: "p"}).String(); got != "p" {
		t.Errorf("a top-level target's String() is %q, want its id", got)
	}
}
