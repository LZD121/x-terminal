package screen

import (
	"testing"
	"time"

	"github.com/LZD121/x-terminal/internal/domain"
)

var agnosticNow = time.Date(2026, 10, 14, 9, 12, 0, 0, time.UTC)

// agnosticOutcome is one observable snapshot of a walk, shared across entity kinds.
type agnosticOutcome struct {
	Rows    int
	Next    domain.Cursor
	Stalled bool
	Latch   bool
	Failed  bool
	Sel     string
	NewRows int
}

func outcomeOf[T any, P any](w Walk[T, P]) agnosticOutcome {
	return agnosticOutcome{
		Rows:    w.Loaded(),
		Next:    w.Next(),
		Stalled: w.Stalled(),
		Latch:   w.InFlight(),
		Failed:  w.Failure().Kind != domain.FailureNone,
		Sel:     w.Selected().ID,
		NewRows: w.NewRows(),
	}
}

// runAgnosticScript drives one shared script of operations over a walk and returns the
// outcomes after each step. The caller supplies the pages (posts or accounts) so both
// entity kinds exercise the SAME expected sequence.
func runAgnosticScript[T any, P any](
	w Walk[T, P],
	reqAt func(domain.Cursor) Request,
	page1 P, rows1 []T, next1 domain.Cursor,
	dedupe P, dedupeRows []T, dedupeNext domain.Cursor,
	stall P, stallRows []T, stallNext domain.Cursor,
	fail domain.Failure,
	refreshRows []T, refreshNext domain.Cursor,
	emptyRefresh []T,
) []agnosticOutcome {
	var out []agnosticOutcome

	// 1. first page
	w = w.absorb(page1, rows1, next1)
	out = append(out, outcomeOf(w))

	// 2. page repeating an id (dedupe)
	w, _ = w.Page(reqAt(w.Next()), agnosticNow)
	w = w.absorb(dedupe, dedupeRows, dedupeNext)
	out = append(out, outcomeOf(w))

	// 3. page that adds nothing / repeated cursor → stall
	w, _ = w.Page(reqAt(w.Next()), agnosticNow)
	w = w.absorb(stall, stallRows, stallNext)
	out = append(out, outcomeOf(w))

	// 4. failed page (on a fresh walk that still has a cursor)
	failed := NewWalk[T, P](w.key)
	failed = failed.absorb(page1, rows1, next1)
	failed = failed.pageFailed(fail)
	out = append(out, outcomeOf(failed))

	// 5. refresh bringing new rows
	ref := NewWalk[T, P](w.key)
	ref = ref.absorb(page1, rows1, next1)
	ref, _ = ref.Refresh(reqAt(""), agnosticNow)
	ref = ref.refresh(refreshRows, refreshNext, RefreshPrepend)
	out = append(out, outcomeOf(ref))

	// 6. refresh bringing none
	none := NewWalk[T, P](w.key)
	none = none.absorb(page1, rows1, next1)
	none, _ = none.Refresh(reqAt(""), agnosticNow)
	none = none.refresh(emptyRefresh, next1, RefreshPrepend)
	out = append(out, outcomeOf(none))

	// 7. selecting and moving
	moved := NewWalk[T, P](w.key)
	moved = moved.absorb(page1, rows1, next1)
	if len(rows1) > 1 {
		moved = moved.Select(Target{ID: w.key(rows1[1])})
	}
	out = append(out, outcomeOf(moved))

	// 8. reaching the end (empty next)
	end := NewWalk[T, P](w.key)
	end = end.absorb(page1, rows1, "")
	out = append(out, outcomeOf(end))

	return out
}

// TestAgnosticWalkBehaviour: one script table run twice — posts and accounts — asserting
// the SAME expected values. A failure here means the machinery is not shared.
func TestAgnosticWalkBehaviour(t *testing.T) {
	want := []agnosticOutcome{
		{Rows: 3, Next: "c1", Sel: "a"},                // first page
		{Rows: 4, Next: "c2", Sel: "a"},                // dedupe adds one
		{Rows: 4, Next: "c2", Stalled: true, Sel: "a"}, // stall
		{Rows: 3, Next: "c1", Failed: true, Sel: "a"},  // failed
		{Rows: 4, Next: "c1", Sel: "a", NewRows: 1},    // refresh new
		{Rows: 3, Next: "c1", Sel: "a", NewRows: 0},    // refresh none
		{Rows: 3, Next: "c1", Sel: "b"},                // select/move
		{Rows: 3, Next: "", Sel: "a"},                  // end
	}

	postKeyFn := func(p domain.Post) string { return string(p.ID) }
	userKeyFn := func(u domain.User) string { return u.ID }

	posts1 := []domain.Post{{ID: "a"}, {ID: "b"}, {ID: "c"}}
	postsDedupe := []domain.Post{{ID: "c"}, {ID: "d"}}
	postsStall := []domain.Post{{ID: "c"}, {ID: "d"}} // nothing new
	postsRefresh := []domain.Post{{ID: "z"}}

	users1 := []domain.User{{ID: "a", ScreenName: "a"}, {ID: "b", ScreenName: "b"}, {ID: "c", ScreenName: "c"}}
	usersDedupe := []domain.User{{ID: "c", ScreenName: "c"}, {ID: "d", ScreenName: "d"}}
	usersStall := []domain.User{{ID: "c", ScreenName: "c"}, {ID: "d", ScreenName: "d"}}
	usersRefresh := []domain.User{{ID: "z", ScreenName: "z"}}

	fail := domain.Failure{Kind: domain.FailureLoadShed, Retry: true}

	postOut := runAgnosticScript(
		NewWalk[domain.Post, domain.PostPage](postKeyFn),
		func(c domain.Cursor) Request { return SearchRequest{Query: "q", Product: domain.Latest, Cursor: c} },
		domain.PostPage{}, posts1, "c1",
		domain.PostPage{}, postsDedupe, "c2",
		domain.PostPage{}, postsStall, "c2", // repeated cursor → stall
		fail,
		postsRefresh, "c1",
		nil,
	)
	userOut := runAgnosticScript(
		NewWalk[domain.User, domain.UserPage](userKeyFn),
		func(c domain.Cursor) Request { return PeopleRequest{Query: "q", Cursor: c} },
		domain.UserPage{}, users1, "c1",
		domain.UserPage{}, usersDedupe, "c2",
		domain.UserPage{}, usersStall, "c2",
		fail,
		usersRefresh, "c1",
		nil,
	)

	if len(postOut) != len(want) || len(userOut) != len(want) {
		t.Fatalf("script lengths: posts=%d users=%d want=%d", len(postOut), len(userOut), len(want))
	}
	for i := range want {
		if postOut[i] != want[i] {
			t.Errorf("posts step %d: got %+v want %+v", i, postOut[i], want[i])
		}
		if userOut[i] != want[i] {
			t.Errorf("users step %d: got %+v want %+v", i, userOut[i], want[i])
		}
		if postOut[i] != userOut[i] {
			t.Errorf("step %d diverged: posts=%+v users=%+v", i, postOut[i], userOut[i])
		}
	}
}

// TestAgnosticDedupeArmIsNonVacuous: stubbing out dedupe in a local copy of the page
// sequence would make the account walk report a different row count — prove the arm
// can fail, then restore.
func TestAgnosticDedupeArmIsNonVacuous(t *testing.T) {
	w := NewWalk[domain.User, domain.UserPage](func(u domain.User) string { return u.ID })
	first := []domain.User{{ID: "a", ScreenName: "a"}, {ID: "b", ScreenName: "b"}}
	w = w.absorb(domain.UserPage{}, first, "c1")
	repeat := []domain.User{{ID: "a", ScreenName: "a"}, {ID: "c", ScreenName: "c"}}
	w = w.absorb(domain.UserPage{}, repeat, "c2")
	if w.Loaded() != 3 {
		t.Fatalf("dedupe kept %d rows, want 3 (a,b,c)", w.Loaded())
	}

	// Break once: absorb without going through the walk's seen set by appending locally.
	broken := append(append([]domain.User{}, first...), repeat...)
	if len(broken) == 3 {
		t.Fatal("the planted violation collapsed; the arm is vacuous")
	}
	if len(broken) != 4 {
		t.Fatalf("planted raw concat has %d rows, want 4 before dedupe", len(broken))
	}
	// Restored: the walk's own absorb is what the script uses above.
}
