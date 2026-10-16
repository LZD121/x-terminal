package screen

import (
	"context"
	"testing"
	"time"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/fake"
)

// testNow is the instant every update in this file is called with. It is a value, not the wall
// clock, because the state machine has no clock: a rate limit's reset is judged against the caller's
// time, and every test here says which time it means.
var testNow = time.Date(2026, 10, 14, 9, 12, 0, 0, time.UTC)

// TestPaginationAbsorbsEachPostOnce is the FEED-02 proof: two pages of the recorded response land
// in the list once each, in the answers' order, and a third answer that repeats three ids the list
// already holds adds only the two it does not.
//
// The overlap is what the upstream does when its window slides: a page that begins where the last
// one ended would never exercise de-duplication, which is why the third answer is assembled by hand
// out of the fake's own posts.
func TestPaginationAbsorbsEachPostOnce(t *testing.T) {
	source := fake.Recorded()
	var list List

	first := fetch(t, source, "")
	second := fetch(t, source, first.Next)
	list = absorbs(t, list, first)
	list = absorbs(t, list, second)

	if got, want := list.Loaded(), len(first.Posts)+len(second.Posts); got != want {
		t.Fatalf("the list holds %d posts after two pages of %d and %d, want %d", got, len(first.Posts), len(second.Posts), want)
	}
	requireUnique(t, list.Posts())
	requireOrder(t, list.Posts(), append(append([]domain.Post{}, first.Posts...), second.Posts...))

	if got := list.Selected().ID; got != string(first.Posts[0].ID) {
		t.Errorf("the selection is %q, want the first page's first post: a list with posts and nothing selected has no row for the selection column and nowhere for the movement keys to start", got)
	}
	if list.Stalled() || !list.HasMore() || list.Next() != second.Next {
		t.Errorf("after two pages the walk is at %q (stalled=%v), want the second page's next cursor", list.Next(), list.Stalled())
	}

	// The overlap: the three ids the list already holds, then two it does not.
	third := fetch(t, source, second.Next)
	tail := append([]domain.Post{}, second.Posts[len(second.Posts)-3:]...)
	overlap := domain.PostPage{Posts: append(tail, third.Posts[0], third.Posts[1]), Next: "a-cursor-that-moved"}
	before := list.Loaded()
	list = absorbs(t, list, overlap)

	if got := list.Loaded(); got != before+2 {
		t.Errorf("the overlapping page added %d posts, want the 2 it carried that the list did not hold", got-before)
	}
	requireUnique(t, list.Posts())
	want := append(append([]domain.Post{}, list.Posts()[:before]...), third.Posts[0], third.Posts[1])
	requireOrder(t, list.Posts(), want)
}

// TestARepeatedCursorStallsTheWalk is the FEED-03 proof: an answer that hands the walk the cursor it
// was asked with has not moved, so the walk ends as stalled — the shell reports it and nothing ever
// asks for that cursor again.
func TestARepeatedCursorStallsTheWalk(t *testing.T) {
	source := fake.Recorded()
	var list List
	list = absorbs(t, list, fetch(t, source, ""))

	repeated := list.Next()
	if repeated == "" {
		t.Fatal("the first page carries no next cursor, so this test has nothing to repeat")
	}
	source.StallWith(repeated)

	var effect Effect
	list, effect = list.Update(PageRequested{}, testNow)
	if effect.Kind != EffectPage || effect.Cursor != repeated {
		t.Fatalf("the page request asked for %v, want one page at %q", effect, repeated)
	}
	page, ok := serve(t, source, effect)
	if !ok {
		t.Fatal("the effect was not performed")
	}
	list = absorbs(t, list, page)

	if !list.Stalled() {
		t.Fatal("a page that hands back the cursor it was asked with did not stall the walk: the shell would ask for the same page again, and again, until the request budget ran out")
	}
	if list.HasMore() {
		t.Error("HasMore is true on a stalled walk, so the position field would keep its + and imply there is more to read")
	}
	if list.Next() != repeated {
		t.Errorf("the stored cursor is %q, want the one the stall was detected on: the list does not pretend the walk moved", list.Next())
	}
	if list.LoadingOlder() || list.Inflight() {
		t.Error("the latches are still set after the answer arrived")
	}
	if list.Failure().Kind != domain.FailureNone {
		t.Error("a stalled walk is not a failure: nothing went wrong, the walk simply ended")
	}

	calls := source.Calls()
	for range 5 {
		var again Effect
		list, again = list.Update(PageRequested{}, testNow)
		if page, ok := serve(t, source, again); ok {
			t.Fatalf("a request was issued for %d posts after the stall", len(page.Posts))
		}
	}
	if got := source.Calls(); got != calls {
		t.Errorf("the source was called %d more times after the stall, want none", got-calls)
	}
}

// TestAPageThatAddsNothingUnseenStallsTheWalk is the other half of FEED-03: a page that says there is
// more while carrying nothing the list does not already hold has not advanced either.
func TestAPageThatAddsNothingUnseenStallsTheWalk(t *testing.T) {
	source := fake.Recorded()
	var list List
	first := fetch(t, source, "")
	list = absorbs(t, list, first)

	if first.Next == "" {
		t.Fatal("the page carries no next cursor, so it cannot be the page that claims there is more")
	}
	list = absorbs(t, list, first)

	if !list.Stalled() {
		t.Fatal("a page that added no unseen post did not stall the walk")
	}
	if list.Loaded() != len(first.Posts) {
		t.Errorf("the list holds %d posts after absorbing the same page twice, want the %d it holds once", list.Loaded(), len(first.Posts))
	}
}

// TestAFailedPageKeepsEveryPostAndTheSelection is the FEED-05 proof: a page that fails decodes to
// nothing at all, so there is no half-page to append and nothing to take back. Every post, every
// page and the selection stay, and the failure is recorded once, for the status line.
func TestAFailedPageKeepsEveryPostAndTheSelection(t *testing.T) {
	source := fake.Recorded()
	var list List
	first := fetch(t, source, "")
	second := fetch(t, source, first.Next)
	list = absorbs(t, list, first)
	list = absorbs(t, list, second)

	selected := second.Posts[2].ID
	list, _ = list.Update(Select{Target: Target{ID: string(selected)}}, testNow)

	var effect Effect
	list, effect = list.Update(PageRequested{}, testNow)
	if effect.Kind != EffectPage {
		t.Fatalf("the page request produced %v, want a page request", effect)
	}

	failure := domain.Failure{
		Kind:   domain.FailureLoadShed,
		Detail: "backoff, then retry",
		Retry:  true,
	}
	posts, cursor, loaded := list.Posts(), list.Next(), list.Loaded()

	list, effect = list.Update(PageFailed{Failure: failure}, testNow)
	if effect.Kind != EffectNone {
		t.Fatalf("the failure produced the effect %v, want none: a failure is reported, never acted on", effect)
	}
	if got := list.Failure(); got != failure {
		t.Errorf("the list records the failure %+v, want the one it was handed", got)
	}
	if !equalPosts(list.Posts(), posts) {
		t.Error("a failed page changed the list: the posts it kept are the posts the user can still read")
	}
	if list.Loaded() != loaded || list.Next() != cursor {
		t.Errorf("after a failure the list holds %d posts at cursor %q, want %d at %q", list.Loaded(), list.Next(), loaded, cursor)
	}
	if got := list.Selected().ID; got != string(selected) {
		t.Errorf("the selection moved to %q on a failure, want %q left where the reader put it", got, selected)
	}
	if list.Inflight() || list.LoadingOlder() {
		t.Error("the latches are still set after a failure, so no later request could be issued")
	}
}

// TestRefreshPrependsUnseenPostsAndReportsTheCount is the refresh half of FEED-04: the answer's new
// posts go in front in the answer's order, the count is what the notice line prints, and neither the
// selection nor the walk's cursor moves.
func TestRefreshPrependsUnseenPostsAndReportsTheCount(t *testing.T) {
	source := fake.Recorded()
	var list List
	first := fetch(t, source, "")
	list = absorbs(t, list, first)

	selected := first.Posts[3].ID
	list, _ = list.Update(Select{Target: Target{ID: string(selected)}}, testNow)
	// A retryable failure first, so that the refresh has something to resolve: a successful fetch is
	// what ends a sticky state, and there is no timer that does it.
	list, _ = list.Update(PageFailed{Failure: domain.Failure{Kind: domain.FailureLoadShed, Detail: "backoff, then retry", Retry: true}}, testNow)

	loaded, cursor := list.Loaded(), list.Next()

	var effect Effect
	list, effect = list.Update(RefreshRequested{}, testNow)
	if effect.Kind != EffectRefresh {
		t.Fatalf("the refresh produced %v, want a refresh effect", effect)
	}
	if effect.Cursor != "" {
		t.Errorf("the refresh carried the cursor %q, want the empty one that asks for page one", effect.Cursor)
	}
	if len(effect.Head) != loaded {
		t.Errorf("the refresh carried %d head ids, want the %d posts the list holds", len(effect.Head), loaded)
	}
	if !list.Inflight() {
		t.Error("the refresh did not latch: a held-down key would issue one request per repeat")
	}

	fresh := []domain.Post{{ID: "refresh-1", Text: "the newest post"}, {ID: "refresh-2", Text: "the one after it"}}
	page, ok := serve(t, source, effect)
	if !ok {
		t.Fatal("the refresh effect was not performed")
	}
	if len(page.Posts) == 0 {
		t.Fatal("the served refresh page is empty")
	}
	list, effect = list.Update(Refreshed{Page: domain.PostPage{Posts: fresh, Next: page.Next, FetchedAt: page.FetchedAt}}, testNow)
	if effect.Kind != EffectNone {
		t.Fatalf("the refresh answer produced the effect %v, want none", effect)
	}

	if list.NewPosts() != len(fresh) {
		t.Errorf("the notice would read %d new posts, want %d", list.NewPosts(), len(fresh))
	}
	if got := list.Posts()[0].ID; got != fresh[0].ID {
		t.Errorf("the list now starts at %q, want the refresh's first post: the answer's order is the list's order", got)
	}
	if got := list.Posts()[1].ID; got != fresh[1].ID {
		t.Errorf("the list's second post is %q, want the refresh's second", got)
	}
	if got := list.Selected().ID; got != string(selected) {
		t.Errorf("the refresh moved the selection to %q, want %q: a refresh reads the top of the timeline, it does not move the reader", got, selected)
	}
	if got, want := list.Index(string(selected)), indexIn(first.Posts, selected)+len(fresh); got != want {
		t.Errorf("the selected post sits at index %d, want %d: the posts above it moved down, the post did not move", got, want)
	}
	if list.Next() != cursor || !list.HasMore() {
		t.Errorf("the refresh moved the walk's cursor to %q, want %q left where it was", list.Next(), cursor)
	}
	if list.Loaded() != loaded+len(fresh) {
		t.Errorf("the list holds %d posts, want %d", list.Loaded(), loaded+len(fresh))
	}
	if list.Failure().Kind != domain.FailureNone {
		t.Errorf("the successful refresh left the failure %+v in place, want it cleared", list.Failure())
	}
	if list.Inflight() {
		t.Error("the refresh latch is still set after the answer arrived")
	}
}

// TestRefreshWithNothingNewReportsZero is the documented "no new posts" notice: a successful answer
// that carries nothing the list does not hold reports zero, and a zero is not a failure.
func TestRefreshWithNothingNewReportsZero(t *testing.T) {
	source := fake.Recorded()
	var list List
	first := fetch(t, source, "")
	list = absorbs(t, list, first)

	selected := first.Posts[4].ID
	list, _ = list.Update(Select{Target: Target{ID: string(selected)}}, testNow)

	var effect Effect
	list, effect = list.Update(RefreshRequested{}, testNow)
	if effect.Kind != EffectRefresh {
		t.Fatalf("the refresh produced %v, want a refresh effect", effect)
	}
	// The same page the list already holds: every one of its ids is seen, so none of them is new.
	list, _ = list.Update(Refreshed{Page: first}, testNow)

	if got := list.NewPosts(); got != 0 {
		t.Errorf("the notice would read %d new posts, want 0", got)
	}
	if list.Loaded() != len(first.Posts) {
		t.Errorf("the list holds %d posts, want the %d it held: a refresh that brings nothing new adds nothing", list.Loaded(), len(first.Posts))
	}
	if got := list.Selected().ID; got != string(selected) {
		t.Errorf("the selection moved to %q, want %q", got, selected)
	}
	requireUnique(t, list.Posts())
}

// TestAPageRequestWhileOneIsInFlightIsANoOp is the single-flight latch: a key repeat while a page is
// out asks for nothing, because a slow upstream must not be turned into a burst of requests.
func TestAPageRequestWhileOneIsInFlightIsANoOp(t *testing.T) {
	source := fake.Recorded()
	var list List
	list = absorbs(t, list, fetch(t, source, ""))

	var effect Effect
	list, effect = list.Update(PageRequested{}, testNow)
	if effect.Kind != EffectPage {
		t.Fatalf("the first page request produced %v, want a page request", effect)
	}
	if !list.Inflight() || !list.LoadingOlder() {
		t.Error("the page request did not latch, so nothing stops a second one")
	}

	calls := source.Calls()
	for range 3 {
		var again Effect
		list, again = list.Update(PageRequested{}, testNow)
		if page, ok := serve(t, source, again); ok {
			t.Fatalf("a press while a page was in flight issued a request for %d posts", len(page.Posts))
		}
		if !list.LoadingOlder() {
			t.Error("a no-op press cleared the page latch, so the answer would have nothing to release")
		}
	}
	if got := source.Calls(); got != calls {
		t.Errorf("the source was called %d more times while a page was in flight, want none", got-calls)
	}

	// The answer releases the latch, and the next press may ask again.
	page, _ := serve(t, source, effect)
	list = absorbs(t, list, page)
	if list.Inflight() {
		t.Error("the latch is still set after the page arrived")
	}
	if _, again := list.Update(PageRequested{}, testNow); again.Kind != EffectPage && again.Kind != EffectNone {
		t.Errorf("the next press produced %v", again)
	}
}

// TestASoftBanStopsTheSessionForTheProcess is OPS-04: after the account is held back, no page
// request and no refresh is issued again — not on a repeat, and not later, because the rule is a
// flag and not a timer.
func TestASoftBanStopsTheSessionForTheProcess(t *testing.T) {
	source := fake.Recorded()
	var list List
	first := fetch(t, source, "")
	list = absorbs(t, list, first)

	list, _ = list.Update(PageRequested{}, testNow)
	ban := domain.Failure{Kind: domain.FailureSoftBan, Detail: "stop for the session", Stopped: true}
	list, _ = list.Update(PageFailed{Failure: ban}, testNow)

	if got := list.Failure(); !got.Stopped {
		t.Fatalf("the list records %+v, want the stopped failure the status line prints as `soft ban  stop for the session`", got)
	}
	if list.Loaded() != len(first.Posts) {
		t.Errorf("the list holds %d posts after the ban, want the %d it had: the pane stops appending, it does not empty", list.Loaded(), len(first.Posts))
	}

	calls := source.Calls()
	for i, at := range []time.Time{testNow, testNow.Add(time.Hour), testNow.Add(365 * 24 * time.Hour)} {
		for _, ev := range []Event{PageRequested{}, RefreshRequested{}, PageRequested{}} {
			var effect Effect
			list, effect = list.Update(ev, at)
			if page, ok := serve(t, source, effect); ok {
				t.Fatalf("round %d at %s: %T issued a request for %d posts after the soft ban", i, at, ev, len(page.Posts))
			}
		}
	}
	if got := source.Calls(); got != calls {
		t.Errorf("the source was called %d more times after the soft ban, want none for the life of the process", got-calls)
	}
	if list.Next() == "" {
		t.Error("the soft ban cleared the walk's cursor: stopping the session is not the same as reaching the end of the timeline, and the list keeps every post it loaded")
	}
	if list.LoadingOlder() || list.Inflight() {
		t.Error("the latches are still set after the ban, so a later answer could never be absorbed")
	}
}

// TestARateLimitedFailureBlocksUntilItsReset is the one state with an end the upstream names: the
// reset instant. Before it the list asks for nothing; at it, one effect; without a reset header,
// nothing at all, because there is no instant to wait for.
func TestARateLimitedFailureBlocksUntilItsReset(t *testing.T) {
	source := fake.Recorded()
	var list List
	first := fetch(t, source, "")
	list = absorbs(t, list, first)
	cursor := list.Next()

	reset := testNow.Add(15 * time.Minute)
	list, _ = list.Update(PageFailed{Failure: domain.Failure{
		Kind:   domain.FailureRateLimited,
		Detail: "wait until the reset time from x-rate-limit-reset",
		Reset:  reset,
	}}, testNow)

	calls := source.Calls()
	for _, ev := range []Event{PageRequested{}, RefreshRequested{}, PageRequested{}} {
		var effect Effect
		list, effect = list.Update(ev, testNow)
		if page, ok := serve(t, source, effect); ok {
			t.Fatalf("%T issued a request for %d posts before the reset", ev, len(page.Posts))
		}
	}
	if got := source.Calls(); got != calls {
		t.Errorf("the source was called %d more times before the reset, want none", got-calls)
	}

	// The reset passes: exactly one effect, at the cursor the walk was at, and the latch is set
	// again so a held key cannot issue a second.
	var effect Effect
	list, effect = list.Update(PageRequested{}, reset)
	if effect.Kind != EffectPage || effect.Cursor != cursor {
		t.Fatalf("at the reset the list asked for %v, want one page at %q", effect, cursor)
	}
	if !list.Inflight() {
		t.Error("the request at the reset did not latch")
	}
	list, effect = list.Update(PageRequested{}, reset)
	if effect.Kind != EffectNone {
		t.Errorf("a second press at the reset produced %v, want none", effect)
	}

	// No reset header is not a reset in the past: "no retry; reset unknown" means the list waits for
	// something else to resolve the state.
	var unknown List
	unknown = absorbs(t, unknown, first)
	unknown, _ = unknown.Update(PageFailed{Failure: domain.Failure{Kind: domain.FailureRateLimited, Detail: "wait until the reset time from x-rate-limit-reset"}}, testNow)
	if _, effect := unknown.Update(PageRequested{}, testNow.Add(24*time.Hour)); effect.Kind != EffectNone {
		t.Errorf("a rate limit with no reset instant produced %v, want none: there is no instant to wait for", effect)
	}

	// And a state whose remedy is "backoff, then retry" does not block at all.
	var shed List
	shed = absorbs(t, shed, first)
	shed, _ = shed.Update(PageFailed{Failure: domain.Failure{Kind: domain.FailureLoadShed, Detail: "backoff, then retry", Retry: true}}, testNow)
	if _, effect := shed.Update(PageRequested{}, testNow); effect.Kind != EffectPage {
		t.Errorf("a load-shed failure produced %v, want the page request it retries with", effect)
	}
}

// TestSelectionMovesThroughTheListsOwnRows covers the pure lookups the shell moves the selection with:
// the row order is the frame's own — every post and then the posts it quotes — movement walks that
// sequence and clamps at both ends, and a Select for a row the list does not hold changes nothing.
func TestSelectionMovesThroughTheListsOwnRows(t *testing.T) {
	source := fake.Recorded()
	var list List
	first := fetch(t, source, "")
	list = absorbs(t, list, first)

	targets := list.Targets()
	if len(targets) < len(first.Posts) {
		t.Fatalf("the frame has %d rows for %d posts, want at least one row for every post", len(targets), len(first.Posts))
	}
	if got := list.Selected().ID; got != string(first.Posts[0].ID) {
		t.Errorf("the selection is %q, want the first post of the first page", got)
	}
	if got := list.Index(string(first.Posts[3].ID)); got != 3 {
		t.Errorf("the fourth post sits at index %d, want 3", got)
	}
	if list.Index("no-such-post") != -1 {
		t.Error("Index does not report -1 for a post the list does not hold")
	}

	// Movement walks the frame's own row order, and clamps at both ends.
	if got := list.Move(-1); got.Selected().ID != string(first.Posts[0].ID) {
		t.Errorf("a step back from the first row landed on %q, want the first row", got.Selected())
	}
	if got := list.Move(len(targets)); !got.Selected().Equal(targets[len(targets)-1]) {
		t.Errorf("a step past the last row landed on %s, want the last row %s", got.Selected(), targets[len(targets)-1])
	}
	if list.Move(1).Selected().Equal(list.Selected()) {
		t.Error("one step forward did not move the selection to the next row")
	}

	// A row resolves to the copy the frame carries, which is what the shell opens a frame from.
	if post, ok := list.Post(Target{ID: string(first.Posts[0].ID)}); !ok || post.ID != first.Posts[0].ID {
		t.Errorf("Post resolved the first post to %q (ok=%v), want the copy the list holds", post.ID, ok)
	}
	if _, ok := list.Post(Target{ID: "no-such-post"}); ok {
		t.Error("Post resolved a row the list does not hold")
	}

	if list.AtEnd() {
		t.Error("AtEnd is true while the selection is the first post")
	}
	last := string(first.Posts[len(first.Posts)-1].ID)
	list, _ = list.Update(Select{Target: Target{ID: last}}, testNow)
	if !list.AtEnd() {
		t.Errorf("the selection is the last loaded post with the cursor %q stored, want AtEnd", list.Next())
	}

	list, _ = list.Update(Select{Target: Target{ID: "no-such-post"}}, testNow)
	if got := list.Selected().ID; got != last {
		t.Error("a Select for a row the list does not hold moved the selection: a prepend would then re-point the pane at a row the reader did not choose")
	}
}

// TestAQuotedPostIsARowOfTheList is THRD-02's mechanism at the reducer: a quoted post is a row the
// cursor can land on, and the frame holds its own copy — so the shell opens the quote's conversation
// from what is already on screen, with no key of its own and no request.
//
// The page is built by hand because the recorded fixture's quotes are the renderer's business; here the
// question is only what the frame does with one.
func TestAQuotedPostIsARowOfTheList(t *testing.T) {
	list := absorbs(t, List{Kind: domain.Following}, domain.PostPage{
		Posts: []domain.Post{{ID: "p", Text: "the post that quotes", Quote: &domain.Post{ID: "q", Text: "the quoted post"}}},
		Next:  "a-cursor",
	})
	target := Target{ID: "q", Path: []string{"p"}}

	if got := list.Targets(); len(got) != 2 || !got[1].Equal(target) {
		t.Fatalf("the frame's rows are %v, want the post and then the post it quotes", got)
	}
	next, _ := list.Update(Select{Target: target}, testNow)
	if !next.Selected().Equal(target) {
		t.Errorf("the selection is %s, want the quoted row %s: a quote is a row like any other", next.Selected(), target)
	}
	if quoted, ok := next.Post(target); !ok || quoted.ID != "q" {
		t.Errorf("Post resolved the quoted row to %q (ok=%v), want the quoted post's own copy", quoted.ID, ok)
	}
	if next.Index("q") != -1 {
		t.Error("Index found the quoted post among the list's own posts: a quote is a row of the block that quotes it, and Target.Top is what names that block")
	}
}

// TestWalkingTheAdversarialSetReachesItsEnd walks the whole hand-authored set the golden frames are
// rendered from, page by page, the way the shell does: one request per press, no duplicate, no
// stall, and an end-of-list the cursor reaches rather than a walk that ends because something went
// wrong.
func TestWalkingTheAdversarialSetReachesItsEnd(t *testing.T) {
	source := fake.Adversarial()
	var list List
	list = absorbs(t, list, fetch(t, source, ""))

	for page := 1; ; page++ {
		if page > 8 {
			t.Fatal("the walk does not end after eight pages: the cursors the adversarial set hands out do not lead to end-of-list")
		}
		if !list.HasMore() {
			break
		}
		var effect Effect
		list, effect = list.Update(PageRequested{}, testNow)
		if effect.Kind != EffectPage {
			t.Fatalf("page %d: the press produced %v, want a page request", page, effect)
		}
		next, ok := serve(t, source, effect)
		if !ok {
			t.Fatalf("page %d: the effect was not performed", page)
		}
		list = absorbs(t, list, next)
		if list.Stalled() {
			t.Fatalf("page %d stalled the walk, and nothing in the adversarial set repeats a cursor", page)
		}
	}

	requireUnique(t, list.Posts())
	if got := list.Loaded(); got != 16 {
		t.Errorf("the adversarial set walks to %d posts, want the 16 it holds", got)
	}
	if list.Next() != "" {
		t.Errorf("the walk ended at the cursor %q, want end-of-list", list.Next())
	}
	if list.Stalled() {
		t.Error("the walk ended as stalled, want as end-of-list: those are different sentences on the status line")
	}
}

// fetch asks the fake for one page, which is the only thing that produces a PostPage in these tests.
func fetch(t *testing.T, source *fake.Source, c domain.Cursor) domain.PostPage {
	t.Helper()
	page, err := source.Home(context.Background(), domain.Following, c)
	if err != nil {
		t.Fatalf("fetching the page at %q: %v", c, err)
	}
	return page
}

// serve performs one effect the way the shell does — it is the only path from an update to a
// request — and reports whether anything was asked for. Tests that claim "no request" call it on the
// effect they got, so that the claim is about the source's call count and not about a struct field.
func serve(t *testing.T, source *fake.Source, effect Effect) (domain.PostPage, bool) {
	t.Helper()
	switch effect.Kind {
	case EffectPage:
		return fetch(t, source, effect.Cursor), true
	case EffectRefresh:
		return fetch(t, source, ""), true
	default:
		return domain.PostPage{}, false
	}
}

// absorbs feeds one page in the way the shell does, and fails if absorbing an answer asked for
// anything: the state machine issues requests on its own events, never while reading one.
func absorbs(t *testing.T, list List, page domain.PostPage) List {
	t.Helper()
	next, effect := list.Update(PageFetched{Page: page}, testNow)
	if effect.Kind != EffectNone {
		t.Fatalf("absorbing a page produced the effect %v, want none", effect)
	}
	return next
}

// requireUnique fails when two posts in the list share an id, which is the whole of FEED-02.
func requireUnique(t *testing.T, posts []domain.Post) {
	t.Helper()
	seen := map[domain.PostID]int{}
	for i, post := range posts {
		if first, ok := seen[post.ID]; ok {
			t.Fatalf("the post %q appears at index %d and again at %d", post.ID, first, i)
		}
		seen[post.ID] = i
	}
}

// requireOrder fails when the list is not the given posts in the given order, which is what "the
// list order is the answer's order" means.
func requireOrder(t *testing.T, got, want []domain.Post) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("the list holds %d posts, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].ID != want[i].ID {
			t.Fatalf("the list's post %d is %q, want %q: the order the answers carried is the list's order", i, got[i].ID, want[i].ID)
		}
	}
}

// equalPosts reports whether two post slices are the same posts in the same order.
func equalPosts(a, b []domain.Post) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].ID != b[i].ID {
			return false
		}
	}
	return true
}

// indexIn is the test's own index lookup, used to compute where a selected post should have moved
// to without asking the list the question the test is about.
func indexIn(posts []domain.Post, id domain.PostID) int {
	for i := range posts {
		if posts[i].ID == id {
			return i
		}
	}
	return -1
}
