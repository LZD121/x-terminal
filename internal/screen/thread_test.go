package screen

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/LZD121/x-terminal/internal/domain"
)

// threadNow is the instant every update in this file is called with: the frame has no clock, and a
// rate limit's reset is judged against this value.
var threadNow = time.Date(2026, 10, 14, 9, 30, 0, 0, time.UTC)

// threadSource is a page source with a call counter and hand-built pages: this file's only route to a
// request, and the reason "opening a post asks for nothing" is a claim about a count rather than about a
// struct field. Nothing in internal/screen can perform a request — the frames produce effects and the
// shell performs them — so serving an effect here is the shell's half of the loop, written out.
type threadSource struct {
	pages map[domain.Cursor]domain.ThreadPage
	calls int
	asked []domain.Cursor
}

func (s *threadSource) Thread(_ context.Context, _ domain.PostID, c domain.Cursor) (domain.ThreadPage, error) {
	s.calls++
	s.asked = append(s.asked, c)
	page, ok := s.pages[c]
	if !ok {
		return domain.ThreadPage{}, errors.New("no page at that cursor")
	}
	return page, nil
}

// serveThread performs one effect the way the shell does and folds the answer back into the frame.
func serveThread(t *testing.T, src *threadSource, frame Thread, effect Effect, now time.Time) Thread {
	t.Helper()
	req, ok := effect.Req.(ThreadRequest)
	if !ok {
		t.Fatalf("the effect carries %#v, want a thread request", effect.Req)
	}
	page, err := src.Thread(context.Background(), req.Focal, req.Cursor)
	if err != nil {
		t.Fatalf("the source serves no page at %q: %v", req.Cursor, err)
	}
	event := Event(ThreadFetched{Page: page})
	if effect.Kind == EffectRefresh {
		event = ThreadRefreshed{Page: page}
	}
	next, after := frame.Update(event, now)
	if after.Kind != EffectNone {
		t.Fatalf("absorbing an answer produced the effect %v, want none", after)
	}
	return next
}

// reply is a post of a conversation's reply stream.
func reply(id, parent, text string) domain.Post {
	return domain.Post{
		ID:        domain.PostID(id),
		ReplyTo:   domain.PostID(parent),
		Text:      text,
		Author:    domain.User{ID: "u-" + id, ScreenName: "user" + id},
		CreatedAt: threadNow,
	}
}

// repliesPage is an answer carrying reply groups' posts in order, with a cursor.
func repliesPage(cursor string, posts ...domain.Post) domain.ThreadPage {
	return domain.ThreadPage{Groups: []domain.Group{{Kind: domain.Replies, Posts: posts}}, Next: domain.Cursor(cursor)}
}

// threadTargets is a frame's row ids in row order, which is what most assertions here compare.
func threadTargets(t Thread) []string {
	out := make([]string, 0, len(t.Targets()))
	for _, target := range t.Targets() {
		out = append(out, target.String())
	}
	return out
}

// TestThreadOpeningCostsNoRequest is the plan's first claim, as a count: a frame opened on the copy the
// caller holds renders that copy — reading every accessor and walking the copy's own rows — and no page
// is asked for. The first request is born from the frame's own trigger, and it is page one: the copy the
// caller held is replaced by the copy the answer carries, in place.
func TestThreadOpeningCostsNoRequest(t *testing.T) {
	held := domain.Post{ID: "7", Text: "the copy the caller held", Author: domain.User{ScreenName: "lzd"}, Counts: domain.Counts{Replies: 2}}
	src := &threadSource{pages: map[domain.Cursor]domain.ThreadPage{
		"": {
			Focal:     domain.Post{ID: "7", Text: "the copy the answer carried", Author: domain.User{ScreenName: "lzd"}},
			Ancestors: []domain.Post{{ID: "1", Text: "the post above"}},
			Groups:    []domain.Group{{Kind: domain.Replies, Posts: []domain.Post{reply("2", "7", "a reply")}}},
			Next:      "c1",
		},
	}}

	frame := NewThread(held)
	if frame.Read() || frame.Loading() {
		t.Fatalf("a frame opened from a copy reports read=%v loading=%v, want neither: nothing has been asked for or answered yet", frame.Read(), frame.Loading())
	}
	if !reflect.DeepEqual(frame.Focal(), held) {
		t.Fatalf("the frame opened on %+v, want the copy the caller held", frame.Focal())
	}
	if got, want := threadTargets(frame), []string{"7"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("the frame's rows are %v, want the one row it holds a copy of", got)
	}

	// Everything the pane reads, and a walk through the copy's own rows: opening costs no request.
	if _, total := frame.Position(); total != 1 {
		t.Errorf("the position counts %d posts, want the one post the frame holds", total)
	}
	frame = frame.Move(1)
	frame = frame.Move(-1)
	if !frame.Selected().Equal(Target{ID: "7"}) {
		t.Errorf("the selection is %s, want the focal post it started on", frame.Selected())
	}
	if src.calls != 0 {
		t.Fatalf("opening a post and reading it asked the source for %d pages, want none: the frame renders from the copy in hand", src.calls)
	}

	// The frame's own trigger asks for page one, with no cursor: it is the first request, and it is the
	// only way one is born.
	next, effect := frame.Update(ThreadRequested{}, threadNow)
	if effect.Kind != EffectPage || effect.Cursor != "" {
		t.Fatalf("the first page request produced %v, want page one with the empty cursor", effect)
	}
	req, ok := effect.Req.(ThreadRequest)
	if !ok || req.Focal != "7" || req.Cursor != "" {
		t.Fatalf("the request is %#v, want page one of the post %q", effect.Req, "7")
	}
	if !next.Loading() {
		t.Error("the first page is out and the frame does not report it as loading")
	}
	if src.calls != 0 {
		t.Fatalf("asking for a page spent %d requests: the frame produces an effect and the shell performs it", src.calls)
	}

	frame = serveThread(t, src, next, effect, threadNow)
	if src.calls != 1 {
		t.Fatalf("the first page cost %d requests, want exactly one", src.calls)
	}
	frame = frame.Move(-1)
	if frame.Loading() || !frame.Read() {
		t.Errorf("after the answer the frame reports read=%v loading=%v, want read and not loading", frame.Read(), frame.Loading())
	}
	if got, want := frame.Focal().Text, "the copy the answer carried"; got != want {
		t.Errorf("the focal post's text is %q, want %q: page one's own copy replaces the one the caller held", got, want)
	}
	if got, want := threadTargets(frame), []string{"1", "7", "2"}; !reflect.DeepEqual(got, want) {
		t.Errorf("the frame's rows are %v, want the chain above, the focal post and the reply", got)
	}
}

// TestAFirstPageThatFailsKeepsTheCopy is the honesty rule for a conversation that could not be read:
// everything on screen stays exactly as it was — including the copy the frame was opened with — and the
// failure is the state the status line prints. A cold-start failure is this state, not an empty
// conversation.
func TestAFirstPageThatFailsKeepsTheCopy(t *testing.T) {
	held := domain.Post{ID: "7", Text: "the copy the caller held"}
	src := &threadSource{}
	frame := NewThread(held)
	_, effect := frame.Update(ThreadRequested{}, threadNow)
	if effect.Kind != EffectPage {
		t.Fatalf("the first page request produced %v, want a page", effect)
	}

	failure := domain.Failure{Kind: domain.FailureLoadShed, Detail: "backoff, then retry", Retry: true}
	frame, after := frame.Update(ThreadFailed{Failure: failure}, threadNow)
	if after.Kind != EffectNone {
		t.Fatalf("a failure produced the effect %v, want none: a failure is reported, never acted on", after)
	}
	if frame.Failure() != failure {
		t.Errorf("the frame records %+v, want the failure it was handed", frame.Failure())
	}
	if !reflect.DeepEqual(frame.Focal(), held) {
		t.Errorf("the failing page changed the focal post to %+v, want the copy the frame was opened with", frame.Focal())
	}
	if got, want := threadTargets(frame), []string{"7"}; !reflect.DeepEqual(got, want) {
		t.Errorf("the frame's rows are %v, want the one row it still holds", got)
	}
	if frame.Read() || frame.Loading() {
		t.Errorf("after a failed first page the frame reports read=%v loading=%v, want neither: no answer arrived, and nothing is still out", frame.Read(), frame.Loading())
	}

	// And the retry is one more trigger away, which is what a retryable failure means.
	_, again := frame.Update(ThreadRequested{}, threadNow)
	if again.Kind != EffectPage {
		t.Errorf("the retry produced %v, want the page one request again", again)
	}
	if src.calls != 0 {
		t.Fatalf("the frame performed %d requests of its own, want none: nothing in this package reaches a source", src.calls)
	}
}

// TestThreadRowsAreTheAnswersOrder is the frame's row order: the chain above root first, the focal post,
// the focal post's own continuations, then the replies — each block followed by the posts it quotes — and
// nothing re-sorted into a tree. A tombstone ancestor keeps its place among the blocks.
func TestThreadRowsAreTheAnswersOrder(t *testing.T) {
	page := domain.ThreadPage{
		Focal: domain.Post{ID: "7", Text: "the focal post", Quote: &domain.Post{ID: "q", Text: "the post it quotes"}},
		Ancestors: []domain.Post{
			{ID: "1", Text: "the root of the chain"},
			{ID: "4", Visibility: domain.Unavailable},
		},
		Groups: []domain.Group{
			{Kind: domain.SelfThread, Posts: []domain.Post{{ID: "8", Text: "the focal post continues"}}},
			{Kind: domain.Replies, Posts: []domain.Post{reply("2", "7", "a reply"), reply("3", "1", "a reply to the root")}},
		},
		Next: "c1",
	}
	frame := threadOf(t, page)

	want := []string{
		"1", // the chain above, root first
		"4", // the tombstone keeps its place in the chain
		"7", // the focal post
		"q (quoted by 7)",
		"8", // the focal post's own continuation
		"2", // the replies, in the answer's own order
		"3", // a reply whose parent is not in the frame keeps its place
	}
	if got := threadTargets(frame); !reflect.DeepEqual(got, want) {
		t.Errorf("the frame's rows are %v, want %v", got, want)
	}
	if got, want := frame.Ancestors(), 2; len(got) != want {
		t.Errorf("the frame holds %d ancestors, want %d", len(got), want)
	}
	if got := frame.Ancestors()[1].Visibility; got != domain.Unavailable {
		t.Errorf("the tombstone ancestor is %v, want it carried as itself", got)
	}
	if got, want := frame.Continued(), "8"; len(got) != 1 || string(got[0].ID) != want {
		t.Errorf("the continuations are %+v, want the self-thread post %q", got, want)
	}
	if got, want := len(frame.Replies()), 2; got != want {
		t.Errorf("the frame holds %d replies, want %d", got, want)
	}

	// Every row is selectable, the quoted row and the tombstone included: what a row is drawn as is the
	// renderer's business, and a row the cursor cannot land on is a row the reader cannot open.
	for _, target := range frame.Targets() {
		next := frame.Select(target)
		if !next.Selected().Equal(target) {
			t.Errorf("the row %s cannot be selected", target)
		}
	}
	if _, ok := frame.Post(Target{ID: "q", Path: []string{"7"}}); !ok {
		t.Error("the frame does not hold the copy of the quoted post: opening a quote costs no request only if the copy is here")
	}
	if _, ok := frame.Post(Target{ID: "1"}); !ok {
		t.Error("the frame does not hold the copy of an ancestor")
	}

	// A later page that carries no ancestors and no continuations is not evidence that they were erased.
	later := repliesPage("c2", reply("5", "7", "an older reply"))
	after, effect := frame.Update(ThreadFetched{Page: later}, threadNow)
	if effect.Kind != EffectNone {
		t.Fatalf("absorbing a later page produced the effect %v, want none", effect)
	}
	if got, want := len(after.Ancestors()), len(page.Ancestors); got != want {
		t.Errorf("a later page left %d ancestors, want the %d the frame already held", got, want)
	}
	if got, want := len(after.Continued()), 1; got != want {
		t.Errorf("a later page left %d continuations, want the %d the frame already held", got, want)
	}
	if got, want := len(after.Replies()), 3; got != want {
		t.Errorf("the frame holds %d replies after a later page, want %d", got, want)
	}
}

// TestThreadPaginationIsTheReplyWalk is the trigger, page by page: reaching the last loaded reply asks for
// exactly one page and a press while it is out is a no-op. Reaching a row above the reply walk — an
// ancestor, the focal post, one of its continuations — asks for nothing at all: those posts came with
// page one and are not a walk.
func TestThreadPaginationIsTheReplyWalk(t *testing.T) {
	src := &threadSource{pages: map[domain.Cursor]domain.ThreadPage{
		"":   repliesPage("c1", reply("2", "7", "the first reply")),
		"c1": repliesPage("c2", reply("3", "7", "the second reply")),
	}}
	frame := threadOf(t,
		domain.ThreadPage{
			Focal:     domain.Post{ID: "7"},
			Ancestors: []domain.Post{{ID: "1"}},
			Groups: []domain.Group{
				{Kind: domain.SelfThread, Posts: []domain.Post{{ID: "8"}}},
				{Kind: domain.Replies, Posts: []domain.Post{reply("2", "7", "the first reply")}},
			},
			Next: "c1",
		})

	// A row above the walk is not a trigger.
	for _, above := range []Target{{ID: "1"}, {ID: "7"}, {ID: "8"}} {
		frame = frame.Select(above)
		_, effect := frame.Update(ThreadRequested{}, threadNow)
		if effect.Kind != EffectNone {
			t.Fatalf("a press with the selection on %s produced %v, want nothing: a post above the reply walk is not a walk", above, effect)
		}
	}

	// The last loaded reply is.
	replyRow := Target{ID: "2"}
	frame = frame.Select(replyRow)
	asked, effect := frame.Update(ThreadRequested{}, threadNow)
	if effect.Kind != EffectPage || effect.Cursor != "c1" {
		t.Fatalf("reaching the last reply produced %v, want one page at the walk's cursor", effect)
	}
	if !asked.LoadingOlder() {
		t.Error("the reply page did not report itself as loading")
	}

	// A press while the page is out is a no-op rather than a queued request.
	for range 3 {
		again, second := asked.Update(ThreadRequested{}, threadNow)
		if second.Kind != EffectNone {
			t.Fatalf("a press while a reply page was out produced %v", second)
		}
		if !reflect.DeepEqual(stateOf(asked.walk), stateOf(again.walk)) {
			t.Error("a refused press changed the reply walk")
		}
	}
	if src.calls != 0 {
		t.Fatalf("the presses spent %d requests of their own", src.calls)
	}

	frame = serveThread(t, src, asked, effect, threadNow)
	if src.calls != 1 {
		t.Fatalf("the reply page cost %d requests, want one", src.calls)
	}
	if got, want := len(frame.Replies()), 2; got != want {
		t.Errorf("the frame holds %d replies after the second page, want %d", got, want)
	}
	if !frame.Selected().Equal(replyRow) {
		t.Errorf("the answer moved the selection to %s, want it left on %s: a late page never moves what the reader is reading", frame.Selected(), replyRow)
	}
	if !frame.HasMore() || frame.Stalled() {
		t.Errorf("the walk is at %q (stalled=%v), want the answer's cursor and no end", frame.walk.Next(), frame.Stalled())
	}
}

// TestThreadStallEndsTheReplyWalk is the first partial-thread condition, and the rule behind it: a cursor
// the upstream would not advance ends the walk, the frame says so, and nothing asks for that cursor again.
func TestThreadStallEndsTheReplyWalk(t *testing.T) {
	src := &threadSource{pages: map[domain.Cursor]domain.ThreadPage{
		"":   repliesPage("c1", reply("2", "7", "the first reply")),
		"c1": repliesPage("c1", reply("2", "7", "the first reply")),
	}}
	frame := threadOf(t, repliesPage("c1", reply("2", "7", "the first reply")))
	frame = frame.Select(Target{ID: "2"})

	asked, effect := frame.Update(ThreadRequested{}, threadNow)
	if effect.Kind != EffectPage {
		t.Fatalf("the page request produced %v, want a page", effect)
	}
	frame = serveThread(t, src, asked, effect, threadNow)

	if !frame.Stalled() {
		t.Fatal("a page that handed back the cursor it was asked with did not end the walk")
	}
	if !frame.PartialStopped() {
		t.Error("the frame does not report the reply walk as stopped: that is the first partial-thread condition")
	}
	if frame.HasMore() {
		t.Error("the walk still says there is more after it ended")
	}
	calls := src.calls
	for range 3 {
		again, effect := frame.Update(ThreadRequested{}, threadNow)
		if effect.Kind != EffectNone {
			t.Fatalf("the walk asked for another page after it ended: %v", effect)
		}
		if !again.Stalled() {
			t.Error("a refused request cleared the stall")
		}
	}
	if src.calls != calls {
		t.Errorf("the source was asked %d more times after the stall, want none", src.calls-calls)
	}
	if frame.Failure().Kind != domain.FailureNone {
		t.Error("a stalled walk raised a failure: a walk that ended is not an error, and the pane says only that it stopped here")
	}
}

// TestThreadPartialAbove is the second partial-thread condition: the focal post answers a post the frame
// does not hold, so the chain above it has a hole at its root.
func TestThreadPartialAbove(t *testing.T) {
	focal := domain.Post{ID: "7", ReplyTo: "1"}
	held := domain.ThreadPage{Focal: focal, Ancestors: []domain.Post{{ID: "1"}}}
	if frame := threadOf(t, held); frame.PartialAbove() {
		t.Error("the frame reports the post above as unreachable while it holds it")
	}

	hole := domain.ThreadPage{Focal: focal, Ancestors: []domain.Post{{ID: "3"}}}
	if frame := threadOf(t, hole); !frame.PartialAbove() {
		t.Error("the frame does not report the post above as unreachable: the focal post answers an id neither the ancestors nor the focal post hold")
	}

	root := domain.ThreadPage{Focal: domain.Post{ID: "7"}}
	if frame := threadOf(t, root); frame.PartialAbove() {
		t.Error("the frame reports a missing post above a post that answers nothing")
	}

	// The focal post answers itself: the chain is complete as far as the frame can see.
	self := domain.ThreadPage{Focal: domain.Post{ID: "7", ReplyTo: "7"}}
	if frame := threadOf(t, self); frame.PartialAbove() {
		t.Error("the frame reports a missing post above a post the frame holds")
	}
}

// TestThreadPartialHidden is the third partial-thread condition, and the count is evidence rather than
// content: the answer carried no replies and no cursor while the focal post's own reply count says there
// are replies. The count is never printed by this package — the frame exposes the question only.
func TestThreadPartialHidden(t *testing.T) {
	cases := []struct {
		name  string
		page  domain.ThreadPage
		count int
		want  bool
	}{
		{"no replies, no cursor, a count that says there are", domain.ThreadPage{Focal: domain.Post{ID: "7"}}, 3, true},
		{"no replies, no cursor, a count of zero", domain.ThreadPage{Focal: domain.Post{ID: "7"}}, 0, false},
		{"no replies but a cursor to walk", domain.ThreadPage{Focal: domain.Post{ID: "7"}, Next: "c1"}, 3, false},
		{"replies that arrived", repliesPage("", reply("2", "7", "a reply")), 3, false},
	}
	for _, tc := range cases {
		focal := tc.page.Focal
		focal.Counts = domain.Counts{Replies: tc.count}
		tc.page.Focal = focal
		frame := threadOf(t, tc.page)
		if got := frame.PartialHidden(); got != tc.want {
			t.Errorf("%s: PartialHidden is %v, want %v", tc.name, got, tc.want)
		}
	}

	// A frame that has read nothing says nothing: the three conditions are about a conversation that was
	// read, never a standing disclaimer over a frame that is still loading.
	held := domain.Post{ID: "7", Counts: domain.Counts{Replies: 4}}
	if frame := NewThread(held); frame.PartialHidden() || frame.PartialStopped() || frame.PartialAbove() {
		t.Error("a frame opened from a copy reports a partial-thread condition before anything has been read")
	}
}

// TestThreadPositionCountsPosts is the status line's position: it counts the posts the frame holds, and
// reports the block the selection is inside — so stepping into a quoted post does not move the number.
func TestThreadPositionCountsPosts(t *testing.T) {
	frame := threadOf(t, domain.ThreadPage{
		Focal:     domain.Post{ID: "7", Quote: &domain.Post{ID: "q"}},
		Ancestors: []domain.Post{{ID: "1"}, {ID: "4"}},
		Groups:    []domain.Group{{Kind: domain.SelfThread, Posts: []domain.Post{{ID: "8"}}}, {Kind: domain.Replies, Posts: []domain.Post{reply("2", "7", "a reply")}}},
	})

	if selected, total := frame.Position(); selected != 3 || total != 5 {
		t.Errorf("the position is %d/%d, want the focal post 3/5", selected, total)
	}
	inQuote := frame.Select(Target{ID: "q", Path: []string{"7"}})
	if selected, total := inQuote.Position(); selected != 3 || total != 5 {
		t.Errorf("the position inside a quoted post is %d/%d, want it unchanged at 3/5: the number answers which post the reader is in", selected, total)
	}
	last := frame.Select(Target{ID: "2"})
	if selected, total := last.Position(); selected != 5 || total != 5 {
		t.Errorf("the position on the last reply is %d/%d, want 5/5", selected, total)
	}
	if selected, _ := NewThread(domain.Post{ID: "7"}).Position(); selected != 1 {
		t.Errorf("a frame opened on a copy is at position %d, want the focal post at 1", selected)
	}
}

// TestThreadVisibilityStatesAreOrdinaryRows is the honesty rule for a conversation: a tombstone, a limited
// post and a withheld post are rows of whichever list they arrived in — they keep their place, they count
// toward the position, they are selectable — and nothing about one is a failure or an error state.
func TestThreadVisibilityStatesAreOrdinaryRows(t *testing.T) {
	tombstone := domain.Post{ID: "2", Visibility: domain.Unavailable}
	limited := domain.Post{ID: "3", Visibility: domain.Limited, Text: "still readable, with a marker"}
	withheld := domain.Post{ID: "4", Visibility: domain.Withheld}
	page := domain.ThreadPage{
		Focal: domain.Post{ID: "7"},
		Ancestors: []domain.Post{
			{ID: "1", Visibility: domain.Unavailable},
		},
		Groups: []domain.Group{{Kind: domain.Replies, Posts: []domain.Post{tombstone, limited, withheld}}},
		Next:   "c1",
	}
	frame := threadOf(t, page)

	got := make([]string, 0, len(frame.Replies()))
	for _, post := range frame.Replies() {
		got = append(got, string(post.ID))
	}
	if want := []string{"2", "3", "4"}; !reflect.DeepEqual(got, want) {
		t.Errorf("the replies are %v, want %v: a post the upstream will not show is a row like any other", got, want)
	}
	targets := threadTargets(frame)
	if want := []string{"1", "7", "2", "3", "4"}; !reflect.DeepEqual(targets, want) {
		t.Errorf("the frame's rows are %v, want %v", targets, want)
	}
	// The frame opens on the focal post, which is the second block here: the tombstone above it is the
	// first, and the position counts the posts the frame holds.
	if selected, total := frame.Position(); total != 5 || selected != 2 {
		t.Errorf("the position is %d/%d, want the focal post at 2/5", selected, total)
	}
	for _, id := range []string{"1", "2", "3", "4"} {
		target := Target{ID: id}
		if !frame.Select(target).Selected().Equal(target) {
			t.Errorf("the %s row is not selectable", id)
		}
	}
	if frame.Failure().Kind != domain.FailureNone {
		t.Errorf("a visibility state raised the failure %+v: none of them is an error", frame.Failure())
	}
	// And one arriving on a later page is still just a row.
	later := repliesPage("c2", domain.Post{ID: "5", Visibility: domain.Withheld})
	next := frame.Select(Target{ID: "4"})
	after, effect := next.Update(ThreadFetched{Page: later}, threadNow)
	if effect.Kind != EffectNone || after.Failure().Kind != domain.FailureNone {
		t.Errorf("a withheld reply on a later page produced %v with the failure %+v", effect, after.Failure())
	}
	if got, want := len(after.Replies()), 4; got != want {
		t.Errorf("the frame holds %d replies after the withheld one arrived, want %d", got, want)
	}
}

// TestThreadRefreshedReplacesTheReplies is `r` in a conversation: the answer's replies replace the frame's
// — this list is a ranking and re-reading it is the only honest way to make it current — the notice counts
// the reply ids the frame did not hold before, the selection is held by target and then falls back to the
// focal post, and the walk's cursor becomes the answer's.
func TestThreadRefreshedReplacesTheReplies(t *testing.T) {
	page := domain.ThreadPage{
		Focal:     domain.Post{ID: "7", Text: "the focal post"},
		Ancestors: []domain.Post{{ID: "1"}},
		Groups:    []domain.Group{{Kind: domain.Replies, Posts: []domain.Post{reply("2", "7", "a reply"), reply("3", "7", "another reply")}}},
		Next:      "c1",
	}
	frame := threadOf(t, page)
	frame = frame.Select(Target{ID: "3"})
	// The refresh is asked for by the shell like any other request; this frame's own refresh event is the
	// answer to it, and the trigger is the walk's.
	fresh := domain.ThreadPage{
		Focal:     domain.Post{ID: "7", Text: "the focal post, re-read"},
		Ancestors: []domain.Post{{ID: "1"}},
		Groups:    []domain.Group{{Kind: domain.Replies, Posts: []domain.Post{reply("3", "7", "another reply"), reply("9", "7", "a reply that is new")}}},
		Next:      "c9",
	}
	refreshed, effect := frame.Update(ThreadRefreshed{Page: fresh}, threadNow)
	if effect.Kind != EffectNone {
		t.Fatalf("the refresh answer produced %v, want none", effect)
	}
	if got, want := len(refreshed.Replies()), 2; got != want {
		t.Errorf("the frame holds %d replies after the refresh, want the answer's %d: a ranking is re-read, not merged into", got, want)
	}
	if got, want := string(refreshed.Replies()[0].ID), "3"; got != want {
		t.Errorf("the first reply is %q, want the answer's own first: nothing is re-sorted", got)
	}
	if got, want := refreshed.NewReplies(), 1; got != want {
		t.Errorf("the notice would read %d new replies, want the %d the frame did not hold before", got, want)
	}
	if !refreshed.Selected().Equal(Target{ID: "3"}) {
		t.Errorf("the selection is %s, want it held by target: the reply the reader was on is still there", refreshed.Selected())
	}
	if got, want := refreshed.walk.Next(), domain.Cursor("c9"); got != want {
		t.Errorf("the walk continues from %q, want the answer's own cursor %q", got, want)
	}
	if got, want := refreshed.Focal().Text, "the focal post, re-read"; got != want {
		t.Errorf("the focal post is %q, want page one's own copy: a refresh re-reads the conversation", got)
	}
	if !refreshed.Read() {
		t.Error("the frame does not report the refresh as read")
	}

	// A refresh whose answer dropped the selected reply leaves the selection on the focal post: the one
	// row this frame always has.
	dropped := domain.ThreadPage{Focal: domain.Post{ID: "7"}, Groups: []domain.Group{{Kind: domain.Replies, Posts: []domain.Post{reply("9", "7", "a new reply")}}}, Next: "c9"}
	after, _ := refreshed.Select(Target{ID: "3"}).Update(ThreadRefreshed{Page: dropped}, threadNow)
	if !after.Selected().Equal(Target{ID: "7"}) {
		t.Errorf("the selection is %s, want the focal post: the reply the reader was on is gone", after.Selected())
	}
}

// TestTheTwoFramesIgnoreEachOthersEvents is the sealed event set's rule: a frame ignores the events that
// are not its own in the same way it ignores one this build does not know. A state machine that guessed at
// an unknown transition is a state machine whose tests prove nothing.
func TestTheTwoFramesIgnoreEachOthersEvents(t *testing.T) {
	list := absorbs(t, List{Kind: domain.Following}, domain.PostPage{Posts: walkRows("1"), Next: "c1"})
	before := stateOf(list.walk)
	for _, ev := range []Event{ThreadRequested{}, ThreadFetched{}, ThreadFailed{}, ThreadRefreshed{}} {
		next, effect := list.Update(ev, threadNow)
		if effect.Kind != EffectNone || !reflect.DeepEqual(stateOf(next.walk), before) {
			t.Errorf("the home list reacted to %T: %v", ev, effect)
		}
	}

	frame := threadOf(t, domain.ThreadPage{Focal: domain.Post{ID: "7"}, Groups: []domain.Group{{Kind: domain.Replies, Posts: []domain.Post{reply("2", "7", "a reply")}}}, Next: "c1"})
	rows := threadTargets(frame)
	for _, ev := range []Event{PageRequested{}, PageFetched{}, PageFailed{}, RefreshRequested{}, Refreshed{}, Select{Target: Target{ID: "2"}}} {
		next, effect := frame.Update(ev, threadNow)
		if effect.Kind != EffectNone || !reflect.DeepEqual(threadTargets(next), rows) {
			t.Errorf("the thread frame reacted to %T: %v", ev, effect)
		}
	}
}

// TestThreadReadsTheReplyCountWithoutPrintingIt is T-04-06, and the rule `04-CONTEXT.md` records: the
// focal post's own reply count is evidence for one question and nothing else. Two frames whose answers
// differ only in that number are the same through every accessor a renderer or the shell reads — bar the
// one bool the number decides — and the frame's own vocabulary has no count in it at all, so a count can
// only travel inside a post value (which every shipped grammar prints no field of).
func TestThreadReadsTheReplyCountWithoutPrintingIt(t *testing.T) {
	answer := func(count int) domain.ThreadPage {
		return domain.ThreadPage{Focal: domain.Post{
			ID:     "7",
			Text:   "the focal post",
			Counts: domain.Counts{Replies: count, Likes: count * 2, Views: count * 10},
		}}
	}
	quiet, loud := threadOf(t, answer(0)), threadOf(t, answer(9))

	if quiet.PartialHidden() {
		t.Error("the frame reports replies as unreachable for a post whose own count says there are none")
	}
	if !loud.PartialHidden() {
		t.Error("the frame does not report replies as unreachable for a post whose own count says there are replies")
	}

	// Everything else is the same, including the number accessors the shell reads: the count moved no
	// number the frame hands out.
	quietSelected, quietTotal := quiet.Position()
	loudSelected, loudTotal := loud.Position()
	switch {
	case quiet.NewReplies() != loud.NewReplies():
		t.Errorf("the new-reply count differs: %d and %d", quiet.NewReplies(), loud.NewReplies())
	case len(quiet.Replies()) != len(loud.Replies()):
		t.Errorf("the reply counts differ: %d and %d", len(quiet.Replies()), len(loud.Replies()))
	case quietSelected != loudSelected || quietTotal != loudTotal:
		t.Errorf("the positions differ: %d/%d and %d/%d", quietSelected, quietTotal, loudSelected, loudTotal)
	case quiet.HasMore() != loud.HasMore() || quiet.Stalled() != loud.Stalled():
		t.Error("the walk's state differs between the two answers")
	case quiet.PartialStopped() != loud.PartialStopped() || quiet.PartialAbove() != loud.PartialAbove():
		t.Error("a partial-thread condition other than the third differs between the two answers")
	case !quiet.Selected().Equal(loud.Selected()):
		t.Errorf("the selections differ: %s and %s", quiet.Selected(), loud.Selected())
	case !reflect.DeepEqual(threadTargets(quiet), threadTargets(loud)):
		t.Error("the row orders differ between the two answers")
	case !reflect.DeepEqual(stripped(quiet.Replies()), stripped(loud.Replies())):
		t.Error("the replies differ between the two answers, beyond the counts a post carries")
	case !reflect.DeepEqual(stripped([]domain.Post{quiet.Focal()}), stripped([]domain.Post{loud.Focal()})):
		t.Error("the focal posts differ between the two answers, beyond the counts a post carries")
	}

	// And the vocabulary: no accessor names an engagement count, so a renderer cannot reach one except
	// through a post value.
	kind := reflect.TypeOf(Thread{})
	for i := range kind.NumMethod() {
		name := kind.Method(i).Name
		for _, banned := range []string{"Count", "Like", "Repost", "View", "Bookmark", "Score"} {
			if strings.Contains(name, banned) {
				t.Errorf("the frame exposes %s: a number a renderer could print as an engagement count", name)
			}
		}
	}
}

// stripped is posts with their engagement numbers zeroed: the two answers above differ only in that
// number, and this is what makes the comparison about everything else.
func stripped(posts []domain.Post) []domain.Post {
	out := slices.Clone(posts)
	for i := range out {
		out[i].Counts = domain.Counts{}
	}
	return out
}

// threadOf is a frame that has read one page, built from a hand-written answer: the thread tests need no
// decoder, no renderer and no terminal for this.
func threadOf(t *testing.T, page domain.ThreadPage) Thread {
	t.Helper()
	frame, effect := NewThread(page.Focal).Update(ThreadFetched{Page: page}, threadNow)
	if effect.Kind != EffectNone {
		t.Fatalf("absorbing an answer produced the effect %v, want none", effect)
	}
	return frame
}
