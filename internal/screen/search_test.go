package screen

import (
	"testing"
	"time"

	"github.com/LZD121/x-terminal/internal/domain"
)

// searchNow is the frame tests' clock.
var searchNow = time.Date(2026, 10, 14, 9, 12, 0, 0, time.UTC)

// searchPosts builds n posts with ids of the given prefix, oldest first in the order the answer
// carried them.
func searchPosts(prefix string, n int) []domain.Post {
	posts := make([]domain.Post, 0, n)
	for i := range n {
		posts = append(posts, domain.Post{
			ID:     domain.PostID(prefix + "-" + string(rune('a'+i))),
			Author: domain.User{ID: "u-lzd", ScreenName: "lzd", Name: "name of lzd"},
			Text:   "a result",
		})
	}
	return posts
}

// searchPage is one answer: the posts, and where the walk continues.
func searchPage(prefix string, n int, next domain.Cursor) domain.PostPage {
	posts := searchPosts(prefix, n)
	return domain.PostPage{Posts: posts, Next: next, Requested: len(posts)}
}

// fetched is the ResultsFetched for a page, with the walk's vocabulary read out of it the way the
// shell reads it out.
func fetched(page domain.PostPage) ResultsFetched[domain.Post, domain.PostPage] {
	return ResultsFetched[domain.Post, domain.PostPage]{Page: page, Rows: page.Posts, Next: page.Next}
}

// TestTheResultsFrameOpensWithOnePageRequest: the constructor's one effect is page one of the
// frame's own request at the empty cursor, and the frame holds nothing else.
func TestTheResultsFrameOpensWithOnePageRequest(t *testing.T) {
	r, effect := NewPostResults("page cache", domain.Latest)
	if effect.Kind != EffectPage {
		t.Fatalf("the opening effect is %v, want a page request", effect.Kind)
	}
	req, ok := effect.Req.(SearchRequest)
	if !ok {
		t.Fatalf("the opening effect's request is %T, want SearchRequest", effect.Req)
	}
	if req.Query != "page cache" || req.Product != domain.Latest || req.Cursor != "" {
		t.Errorf("the opening request is %+v, want the query under latest at the empty cursor", req)
	}
	if r.Query() != "page cache" || r.Product() != domain.Latest {
		t.Errorf("the frame names %q under %v, want the query under latest", r.Query(), r.Product())
	}
	if r.Walk().Loaded() != 0 || r.Walk().InFlight() {
		t.Errorf("a fresh frame holds %d rows with a latch set: the constructor issues no request of its own", r.Walk().Loaded())
	}
}

// TestTheResultsFramePagesAtTheWalksCursor: a page request carries the cursor the walk has
// reached, the answer appends in the answer's order, and no post appears twice.
func TestTheResultsFramePagesAtTheWalksCursor(t *testing.T) {
	r, _ := NewPostResults("page cache", domain.Latest)
	r, _ = r.Update(fetched(searchPage("one", 3, "cursor-1")), searchNow)

	r, effect := r.Update(ResultsRequested{}, searchNow)
	if effect.Kind != EffectPage {
		t.Fatalf("the page trigger's effect is %v, want a page request", effect.Kind)
	}
	req := effect.Req.(SearchRequest)
	if req.Cursor != "cursor-1" {
		t.Errorf("the page request carries %q, want the walk's cursor", req.Cursor)
	}
	if req.Query != "page cache" || req.Product != domain.Latest {
		t.Errorf("the page request is %+v, want the frame's own read re-issued", req)
	}

	// A second trigger while the latch is set is a no-op.
	if _, again := r.Update(ResultsRequested{}, searchNow); again.Kind != EffectNone {
		t.Errorf("a page trigger while one is in flight issues %v, want nothing", again.Kind)
	}

	// The answer overlaps the page before it: the shared walk's dedupe drops what it holds.
	page := searchPage("two", 2, "")
	page.Posts = append([]domain.Post{searchPosts("one", 3)[2]}, page.Posts...)
	page.Requested++
	r, _ = r.Update(fetched(page), searchNow)
	if got := r.Walk().Loaded(); got != 5 {
		t.Errorf("after an overlapping page the walk holds %d rows, want 5: no post appears twice", got)
	}
	if r.Walk().HasMore() {
		t.Errorf("the walk claims more after a page with no cursor: an empty cursor is the end of the walk")
	}
}

// TestTheResultsFrameStallsOnARepeatedCursor: a cursor that comes back a second time ends the
// walk, and the frame reports it rather than asking again.
func TestTheResultsFrameStallsOnARepeatedCursor(t *testing.T) {
	r, _ := NewPostResults("page cache", domain.Latest)
	r, _ = r.Update(fetched(searchPage("one", 3, "cursor-1")), searchNow)
	// The page trigger goes out at the walk's cursor, and the answer hands the same cursor back.
	r, _ = r.Update(ResultsRequested{}, searchNow)
	r, _ = r.Update(fetched(searchPage("two", 2, "cursor-1")), searchNow)
	if !r.Walk().Stalled() {
		t.Fatal("a page that hands back the cursor it was asked with did not end the walk")
	}
	if _, effect := r.Update(ResultsRequested{}, searchNow); effect.Kind != EffectNone {
		t.Errorf("a stalled walk answers a page trigger with %v, want nothing", effect.Kind)
	}
}

// TestTheResultsFrameRefreshPrependsAndCounts: `r` re-reads page one under the same query and
// product, prepends only the posts the walk did not hold, and announces how many those were.
func TestTheResultsFrameRefreshPrependsAndCounts(t *testing.T) {
	r, _ := NewPostResults("page cache", domain.Latest)
	r, _ = r.Update(fetched(searchPage("one", 3, "cursor-1")), searchNow)

	r, effect := r.Update(ResultsRefreshRequested{}, searchNow)
	if effect.Kind != EffectRefresh {
		t.Fatalf("the refresh's effect is %v, want a refresh", effect.Kind)
	}
	req := effect.Req.(SearchRequest)
	if req.Cursor != "" || req.Query != "page cache" || req.Product != domain.Latest {
		t.Errorf("the refresh request is %+v, want page one of the same query and product", req)
	}

	// The answer carries two new posts at the top and one the walk already holds.
	fresh := searchPage("fresh", 2, "").Posts
	answer := domain.PostPage{Posts: append(fresh, searchPosts("one", 3)[0]), Next: "cursor-1", Requested: 3}
	r, _ = r.Update(ResultsRefreshed[domain.Post, domain.PostPage]{Page: answer, Rows: answer.Posts, Next: answer.Next}, searchNow)
	if got := r.Walk().NewRows(); got != 2 {
		t.Errorf("the refresh reports %d new rows, want 2", got)
	}
	if got := r.Walk().Loaded(); got != 5 {
		t.Errorf("after the refresh the walk holds %d rows, want 5", got)
	}
	if got := r.Walk().Rows()[0].ID; got != fresh[0].ID {
		t.Errorf("the first row after the refresh is %q, want the answer's newest %q", got, fresh[0].ID)
	}
}

// TestTheResultsFrameFailureKeepsTheRows: a failed page changes nothing but the failure, and a
// successful read clears it.
func TestTheResultsFrameFailureKeepsTheRows(t *testing.T) {
	r, _ := NewPostResults("page cache", domain.Latest)
	r, _ = r.Update(fetched(searchPage("one", 3, "cursor-1")), searchNow)
	selected := r.Walk().Selected()

	failure := domain.Failure{Kind: domain.FailureRateLimited, Detail: "the classifier's sentence"}
	r, _ = r.Update(ResultsFailed{Failure: failure}, searchNow)
	if r.Walk().Failure() != failure {
		t.Errorf("the frame's failure is %+v, want the one the request reported", r.Walk().Failure())
	}
	if r.Walk().Loaded() != 3 || !r.Walk().Selected().Equal(selected) {
		t.Errorf("a failed page moved the frame: %d rows, selection %v", r.Walk().Loaded(), r.Walk().Selected())
	}

	r, _ = r.Update(fetched(searchPage("two", 2, "")), searchNow)
	if r.Walk().Failure().Kind != domain.FailureNone {
		t.Errorf("a successful page left the failure %v on the frame", r.Walk().Failure().Kind)
	}
}

// TestTheResultsFrameIgnoresWhatItDoesNotKnow: an event this frame does not carry is a no-op,
// including the other frames' own events.
func TestTheResultsFrameIgnoresWhatItDoesNotKnow(t *testing.T) {
	r, _ := NewPostResults("page cache", domain.Latest)
	r, _ = r.Update(fetched(searchPage("one", 2, "cursor-1")), searchNow)

	for _, ev := range []Event{nil, PageRequested{}, PageFailed{Failure: domain.Failure{Kind: domain.FailureRateLimited}}, ThreadRequested{}} {
		next, effect := r.Update(ev, searchNow)
		if effect.Kind != EffectNone {
			t.Errorf("the event %T produced effect %v, want none", ev, effect.Kind)
		}
		if next.Walk().Loaded() != 2 || next.Walk().Failure().Kind != domain.FailureNone {
			t.Errorf("the event %T changed the frame: %d rows, failure %v", ev, next.Walk().Loaded(), next.Walk().Failure().Kind)
		}
	}
}
