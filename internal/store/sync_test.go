package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/LZD121/x-terminal/internal/domain"
)

func TestUpsertPreservesFirstSeenAndRestoresPresent(t *testing.T) {
	s := openTest(t)
	ctx := context.Background()
	first := time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC)
	second := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)

	item := domain.SavedItem{
		Kind:      domain.Bookmark,
		Post:      samplePost("1", "kestrel", "first body"),
		SortIndex: "a",
		FirstSeen: first,
		LastSeen:  first,
		Present:   true,
	}
	if added, err := s.Upsert(ctx, []domain.SavedItem{item}); err != nil || added != 1 {
		t.Fatalf("first Upsert = (%d, %v), want (1, nil)", added, err)
	}

	if _, err := s.MarkAbsent(ctx, domain.Bookmark, first.Add(time.Hour)); err != nil {
		t.Fatalf("MarkAbsent: %v", err)
	}

	item.Post.Text = "updated body"
	item.SortIndex = "b"
	item.LastSeen = second
	item.FirstSeen = second // must not overwrite the stored first_seen
	if added, err := s.Upsert(ctx, []domain.SavedItem{item}); err != nil || added != 0 {
		t.Fatalf("second Upsert = (%d, %v), want (0, nil)", added, err)
	}

	var (
		firstSeen, lastSeen int64
		sortIndex           string
		present             int
		body                string
	)
	if err := s.w.QueryRow(`SELECT first_seen, last_seen_at, sort_index, present, json FROM saved_items WHERE post_id = '1'`).
		Scan(&firstSeen, &lastSeen, &sortIndex, &present, &body); err != nil {
		t.Fatalf("select: %v", err)
	}
	if firstSeen != first.Unix() {
		t.Errorf("first_seen = %d, want %d (must survive upsert)", firstSeen, first.Unix())
	}
	if lastSeen != second.Unix() {
		t.Errorf("last_seen_at = %d, want %d", lastSeen, second.Unix())
	}
	if sortIndex != "b" {
		t.Errorf("sort_index = %q, want b", sortIndex)
	}
	if present != 1 {
		t.Errorf("present = %d, want 1 after re-seeing an absent row", present)
	}
	if !strings.Contains(body, "updated body") {
		t.Errorf("json body %q does not carry the updated text", body)
	}
}

func TestUpsertAddedCountsOnlyUnknownIDs(t *testing.T) {
	s := openTest(t)
	ctx := context.Background()
	seen := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	known := []domain.SavedItem{
		{Kind: domain.Like, Post: samplePost("a", "mira", "one"), SortIndex: "3", FirstSeen: seen, LastSeen: seen, Present: true},
		{Kind: domain.Like, Post: samplePost("b", "mira", "two"), SortIndex: "2", FirstSeen: seen, LastSeen: seen, Present: true},
	}
	if added, err := s.Upsert(ctx, known); err != nil || added != 2 {
		t.Fatalf("seed Upsert = (%d, %v), want (2, nil)", added, err)
	}
	page := []domain.SavedItem{
		{Kind: domain.Like, Post: samplePost("a", "mira", "one"), SortIndex: "3", FirstSeen: seen, LastSeen: seen, Present: true},
		{Kind: domain.Like, Post: samplePost("b", "mira", "two"), SortIndex: "2", FirstSeen: seen, LastSeen: seen, Present: true},
		{Kind: domain.Like, Post: samplePost("c", "mira", "three"), SortIndex: "1", FirstSeen: seen, LastSeen: seen, Present: true},
	}
	added, err := s.Upsert(ctx, page)
	if err != nil {
		t.Fatalf("page Upsert: %v", err)
	}
	if added != 1 {
		t.Fatalf("added = %d, want 1 (two known + one unknown)", added)
	}
}

func TestMarkAbsentNeverDeletes(t *testing.T) {
	s := openTest(t)
	ctx := context.Background()
	t0 := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	t1 := t0.Add(time.Hour)
	t2 := t0.Add(2 * time.Hour)
	runStart := t1 // after the third row's last_seen; at/before the other two

	items := []domain.SavedItem{
		{Kind: domain.Bookmark, Post: samplePost("1", "a", "one"), SortIndex: "c", FirstSeen: t0, LastSeen: t2, Present: true},
		{Kind: domain.Bookmark, Post: samplePost("2", "b", "two"), SortIndex: "b", FirstSeen: t0, LastSeen: t1, Present: true},
		{Kind: domain.Bookmark, Post: samplePost("3", "c", "three"), SortIndex: "a", FirstSeen: t0, LastSeen: t0, Present: true},
	}
	if _, err := s.Upsert(ctx, items); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	n, err := s.MarkAbsent(ctx, domain.Bookmark, runStart)
	if err != nil {
		t.Fatalf("MarkAbsent: %v", err)
	}
	if n != 1 {
		t.Fatalf("MarkAbsent marked %d, want 1 (only the third row is older than run start)", n)
	}

	page, err := s.Page(ctx, domain.Bookmark, "")
	if err != nil {
		t.Fatalf("Page: %v", err)
	}
	if len(page.Posts) != 2 {
		t.Fatalf("Page returned %d posts, want 2 (absent row omitted)", len(page.Posts))
	}
	for _, p := range page.Posts {
		if p.ID == "3" {
			t.Fatal("Page returned the absent row")
		}
	}

	st, err := s.Status(ctx, domain.Bookmark)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if st.Present != 2 || st.Absent != 1 {
		t.Fatalf("Status present/absent = %d/%d, want 2/1", st.Present, st.Absent)
	}

	var total int
	if err := s.r.QueryRow(`SELECT count(*) FROM saved_items WHERE kind = 'bookmark'`).Scan(&total); err != nil {
		t.Fatalf("count: %v", err)
	}
	if total != 3 {
		t.Fatalf("table still holds %d rows, want 3 (never-delete)", total)
	}
}

func TestRecordRunWritesLastSuccessOnlyOnSuccess(t *testing.T) {
	s := openTest(t)
	ctx := context.Background()
	started := time.Date(2026, 10, 14, 10, 0, 0, 0, time.UTC)
	finished := started.Add(time.Minute)

	if err := s.RecordRun(ctx, domain.SyncResult{
		Kind: domain.Bookmark, Full: false, Pages: 1, Added: 2, Absent: 0,
		Stop: domain.StopNoUnknownIDs, Started: started, Finished: finished,
	}); err != nil {
		t.Fatalf("successful RecordRun: %v", err)
	}

	st, err := s.Status(ctx, domain.Bookmark)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if !st.LastSuccess.Equal(finished) {
		t.Fatalf("LastSuccess = %v, want %v", st.LastSuccess, finished)
	}

	var runs int
	if err := s.r.QueryRow(`SELECT count(*) FROM sync_runs`).Scan(&runs); err != nil {
		t.Fatalf("count runs: %v", err)
	}
	if runs != 1 {
		t.Fatalf("sync_runs = %d, want 1", runs)
	}

	failedAt := finished.Add(time.Hour)
	if err := s.RecordRun(ctx, domain.SyncResult{
		Kind: domain.Bookmark, Full: false, Pages: 1, Added: 0, Absent: 0,
		Stop: domain.StopFailure, Started: failedAt, Finished: failedAt.Add(time.Second),
	}); err != nil {
		t.Fatalf("failure RecordRun: %v", err)
	}
	st, err = s.Status(ctx, domain.Bookmark)
	if err != nil {
		t.Fatalf("Status after failure: %v", err)
	}
	if !st.LastSuccess.Equal(finished) {
		t.Fatalf("LastSuccess after failure = %v, want the previous success %v", st.LastSuccess, finished)
	}
	if err := s.r.QueryRow(`SELECT count(*) FROM sync_runs`).Scan(&runs); err != nil {
		t.Fatalf("count runs: %v", err)
	}
	if runs != 2 {
		t.Fatalf("sync_runs after failure = %d, want 2", runs)
	}
}

func TestOwnerRefusesRepoint(t *testing.T) {
	s := openTest(t)
	ctx := context.Background()
	if err := s.SetOwner(ctx, "aergon"); err != nil {
		t.Fatalf("SetOwner(aergon): %v", err)
	}
	if err := s.SetOwner(ctx, "aergon"); err != nil {
		t.Fatalf("idempotent SetOwner(aergon): %v", err)
	}
	err := s.SetOwner(ctx, "kestrel")
	if !errors.Is(err, ErrOwnerConflict) {
		t.Fatalf("SetOwner(kestrel) = %v, want ErrOwnerConflict", err)
	}
	if err == nil || !strings.Contains(err.Error(), "aergon") || !strings.Contains(err.Error(), "kestrel") {
		t.Fatalf("conflict error %v must name both handles", err)
	}
	st, err := s.Status(ctx, domain.Bookmark)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if st.Owner != "aergon" {
		t.Fatalf("Owner = %q, want aergon", st.Owner)
	}
}

func TestSnapshotRoundTripsWithoutCounts(t *testing.T) {
	s := openTest(t)
	ctx := context.Background()
	seen := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	quote := samplePost("q1", "mira", "quoted text")
	reposter := domain.User{ID: "r1", ScreenName: "relay", Name: "Relay"}
	post := samplePost("p1", "aergon", "outer text")
	post.Counts = domain.Counts{Replies: 3, Likes: 9, Views: 100}
	post.Media = []domain.MediaRef{
		{Kind: domain.Image, URL: "https://example/img", AltText: "an image"},
		{Kind: domain.GIF, URL: "https://example/gif", AltText: ""},
	}
	post.Quote = &quote
	post.RepostedBy = &reposter
	post.ConversationID = "conv-1"
	post.ReplyTo = "parent-1"
	post.Visibility = domain.Limited

	if _, err := s.Upsert(ctx, []domain.SavedItem{{
		Kind: domain.Bookmark, Post: post, SortIndex: "z",
		FirstSeen: seen, LastSeen: seen, Present: true,
	}}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	page, err := s.Page(ctx, domain.Bookmark, "")
	if err != nil {
		t.Fatalf("Page: %v", err)
	}
	if len(page.Posts) != 1 {
		t.Fatalf("posts = %d", len(page.Posts))
	}
	got := page.Posts[0]
	if got.Text != "outer text" || got.Author.ID != "uid-p1" || got.Author.ScreenName != "aergon" {
		t.Fatalf("post identity broken: %+v", got)
	}
	if got.Quote == nil || got.Quote.Text != "quoted text" {
		t.Fatalf("quote missing: %+v", got.Quote)
	}
	if got.RepostedBy == nil || got.RepostedBy.ScreenName != "relay" {
		t.Fatalf("reposted_by missing: %+v", got.RepostedBy)
	}
	if len(got.Media) != 2 || got.Media[0].Kind != domain.Image || got.Media[1].Kind != domain.GIF {
		t.Fatalf("media = %+v", got.Media)
	}
	if got.Visibility != domain.Limited {
		t.Fatalf("visibility = %v", got.Visibility)
	}
	if got.Counts != (domain.Counts{}) {
		t.Fatalf("counts survived the round trip: %+v", got.Counts)
	}

	var raw string
	if err := s.r.QueryRow(`SELECT json FROM saved_items WHERE post_id = 'p1'`).Scan(&raw); err != nil {
		t.Fatalf("raw json: %v", err)
	}
	if strings.Contains(raw, "Counts") || strings.Contains(raw, "\"likes\"") || strings.Contains(raw, "\"replies\"") {
		t.Fatalf("stored JSON carries a count field: %s", raw)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}
	for _, banned := range []string{"Counts", "counts", "Replies", "Likes", "Views", "Bookmarks", "Quotes", "Reposts"} {
		if _, ok := m[banned]; ok {
			t.Errorf("stored JSON has key %q", banned)
		}
	}
	_ = sql.ErrNoRows // keep database/sql imported for clarity in helpers above
}

// fakeFetcher serves a fixed slice of pages and counts calls. Defined in this
// test file so the walk is driven with no socket and no adapter.
type fakeFetcher struct {
	pages    []domain.PostPage
	calls    int
	errAt    int // 1-based call index that returns err; 0 = never
	err      error
	infinite bool // always one new post + non-empty next (page-cap arm)
	n        int
}

func (f *fakeFetcher) Saved(_ context.Context, _ domain.SavedKind, c domain.Cursor) (domain.PostPage, error) {
	f.calls++
	if f.errAt > 0 && f.calls == f.errAt {
		return domain.PostPage{}, f.err
	}
	if f.infinite {
		f.n++
		id := domain.PostID(fmt.Sprintf("p%d", f.n))
		return domain.PostPage{
			Posts: []domain.Post{samplePost(string(id), "a", string(id))},
			Next:  domain.Cursor(fmt.Sprintf("c%d", f.n)),
		}, nil
	}
	if f.calls-1 >= len(f.pages) {
		return domain.PostPage{}, nil
	}
	return f.pages[f.calls-1], nil
}

type fakeViewer struct {
	user  domain.User
	err   error
	calls int
}

func (v *fakeViewer) Viewer(context.Context) (domain.User, error) {
	v.calls++
	if v.err != nil {
		return domain.User{}, v.err
	}
	return v.user, nil
}

func freezeNow(t *testing.T, at time.Time) {
	t.Helper()
	prev := now
	now = func() time.Time { return at }
	t.Cleanup(func() { now = prev })
}

func walkAll(t *testing.T, run domain.SyncRun) domain.SyncProgress {
	t.Helper()
	for {
		p, err := run.Step(context.Background())
		if err != nil {
			t.Fatalf("Step: %v", err)
		}
		if p.Done {
			return p
		}
	}
}

func TestWalkTwoPagesAddsThreeEndsAtList(t *testing.T) {
	s := openTest(t)
	ctx := context.Background()
	started := time.Date(2026, 10, 14, 15, 0, 0, 0, time.UTC)
	freezeNow(t, started)

	fetch := &fakeFetcher{pages: []domain.PostPage{
		{
			Posts: []domain.Post{samplePost("1", "a", "one"), samplePost("2", "b", "two")},
			Next:  "c1",
		},
		{
			// One known (2) and one unknown (3); empty Next ends the list. The page
			// added something, so the stop is EndOfList rather than NoUnknownIDs.
			Posts: []domain.Post{samplePost("2", "b", "two"), samplePost("3", "c", "three")},
		},
	}}
	viewer := &fakeViewer{user: domain.User{ScreenName: "aergon", ID: "1"}}
	m := NewMirror(s, fetch, viewer)

	run, err := m.Begin(ctx, domain.Bookmark, false)
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	prog := walkAll(t, run)
	if prog.Stop != domain.StopEndOfList {
		t.Fatalf("stop = %v (%s), want StopEndOfList; pages=%d added=%d calls=%d", prog.Stop, prog.Stop, prog.Pages, prog.Added, fetch.calls)
	}
	if prog.Added != 3 {
		t.Fatalf("added = %d, want 3", prog.Added)
	}
	if fetch.calls != 2 {
		t.Fatalf("calls = %d, want 2", fetch.calls)
	}

	page, err := m.Page(ctx, domain.Bookmark, "")
	if err != nil {
		t.Fatalf("Page: %v", err)
	}
	if len(page.Posts) != 3 {
		t.Fatalf("Page returned %d posts, want 3", len(page.Posts))
	}
	if page.Posts[0].ID != "1" || page.Posts[2].ID != "3" {
		t.Fatalf("order = %v, want newest-first 1,2,3", idsOf(page.Posts))
	}

	var runs int
	if err := s.r.QueryRow(`SELECT count(*) FROM sync_runs`).Scan(&runs); err != nil || runs != 1 {
		t.Fatalf("sync_runs = %d (%v), want 1", runs, err)
	}
	st, err := m.Status(ctx, domain.Bookmark)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if st.LastSuccess.IsZero() {
		t.Fatal("LastSuccess did not move after a successful walk")
	}
}

func idsOf(posts []domain.Post) []domain.PostID {
	out := make([]domain.PostID, len(posts))
	for i, p := range posts {
		out[i] = p.ID
	}
	return out
}

func TestWalkIncrementalFirstPageAddsNothing(t *testing.T) {
	s := openTest(t)
	ctx := context.Background()
	started := time.Date(2026, 10, 14, 16, 0, 0, 0, time.UTC)
	freezeNow(t, started)

	if _, err := s.Upsert(ctx, []domain.SavedItem{
		{Kind: domain.Bookmark, Post: samplePost("1", "a", "one"), SortIndex: "z", FirstSeen: started, LastSeen: started, Present: true},
		{Kind: domain.Bookmark, Post: samplePost("2", "b", "two"), SortIndex: "y", FirstSeen: started, LastSeen: started, Present: true},
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	fetch := &fakeFetcher{pages: []domain.PostPage{{
		Posts: []domain.Post{samplePost("1", "a", "one"), samplePost("2", "b", "two")},
		Next:  "c1",
	}}}
	m := NewMirror(s, fetch, &fakeViewer{user: domain.User{ScreenName: "aergon"}})
	run, err := m.Begin(ctx, domain.Bookmark, false)
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	prog, err := run.Step(ctx)
	if err != nil {
		t.Fatalf("Step: %v", err)
	}
	if !prog.Done || prog.Stop != domain.StopNoUnknownIDs {
		t.Fatalf("prog = %+v, want Done with StopNoUnknownIDs", prog)
	}
	if fetch.calls != 1 {
		t.Fatalf("calls = %d, want exactly 1", fetch.calls)
	}
	if prog.Added != 0 {
		t.Fatalf("added = %d, want 0", prog.Added)
	}
}

func TestWalkEndOfList(t *testing.T) {
	s := openTest(t)
	ctx := context.Background()
	freezeNow(t, time.Date(2026, 10, 14, 17, 0, 0, 0, time.UTC))

	fetch := &fakeFetcher{pages: []domain.PostPage{{
		Posts: []domain.Post{samplePost("9", "z", "end")},
		Next:  "",
	}}}
	m := NewMirror(s, fetch, &fakeViewer{user: domain.User{ScreenName: "aergon"}})
	run, err := m.Begin(ctx, domain.Bookmark, false)
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	prog := walkAll(t, run)
	if prog.Stop != domain.StopEndOfList {
		t.Fatalf("stop = %v, want StopEndOfList", prog.Stop)
	}
	page, err := m.Page(ctx, domain.Bookmark, "")
	if err != nil {
		t.Fatalf("Page: %v", err)
	}
	if page.Next != "" {
		t.Fatalf("reader Next = %q, want empty", page.Next)
	}
}

func TestWalkRecordsRunAndLastSuccess(t *testing.T) {
	s := openTest(t)
	ctx := context.Background()
	freezeNow(t, time.Date(2026, 10, 14, 18, 0, 0, 0, time.UTC))

	fetch := &fakeFetcher{pages: []domain.PostPage{{
		Posts: []domain.Post{samplePost("1", "a", "one")},
	}}}
	m := NewMirror(s, fetch, &fakeViewer{user: domain.User{ScreenName: "aergon"}})
	run, err := m.Begin(ctx, domain.Bookmark, false)
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	prog := walkAll(t, run)
	if !prog.Done {
		t.Fatal("walk not done")
	}

	var pages, items, absent int
	var stop string
	if err := s.r.QueryRow(`SELECT pages, items, absent, stop_reason FROM sync_runs ORDER BY id DESC LIMIT 1`).
		Scan(&pages, &items, &absent, &stop); err != nil {
		t.Fatalf("sync_runs: %v", err)
	}
	if pages != 1 || items != 1 || absent != 0 || stop != domain.StopEndOfList.String() {
		t.Fatalf("run row = pages=%d items=%d absent=%d stop=%q", pages, items, absent, stop)
	}
	st, err := m.Status(ctx, domain.Bookmark)
	if err != nil || st.LastSuccess.IsZero() {
		t.Fatalf("LastSuccess after success = %v (%v)", st.LastSuccess, err)
	}
	prev := st.LastSuccess

	// Failed walk: last-success untouched, run row still written.
	failFetch := &fakeFetcher{errAt: 1, err: errors.New("upstream refused")}
	m2 := NewMirror(s, failFetch, &fakeViewer{user: domain.User{ScreenName: "aergon"}})
	run2, err := m2.Begin(ctx, domain.Bookmark, false)
	if err != nil {
		t.Fatalf("Begin fail: %v", err)
	}
	_, err = run2.Step(ctx)
	if err == nil {
		t.Fatal("expected fetch error")
	}
	st, err = m2.Status(ctx, domain.Bookmark)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if !st.LastSuccess.Equal(prev) {
		t.Fatalf("LastSuccess moved on failure: %v -> %v", prev, st.LastSuccess)
	}
	var stops int
	if err := s.r.QueryRow(`SELECT count(*) FROM sync_runs WHERE stop_reason = ?`, domain.StopFailure.String()).Scan(&stops); err != nil || stops != 1 {
		t.Fatalf("failure runs = %d (%v), want 1", stops, err)
	}
}

func TestFullNeverDeletes(t *testing.T) {
	s := openTest(t)
	ctx := context.Background()
	seedAt := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	runAt := time.Date(2026, 10, 14, 19, 0, 0, 0, time.UTC)
	freezeNow(t, runAt)

	seed := []domain.SavedItem{
		{Kind: domain.Bookmark, Post: samplePost("1", "a", "one"), SortIndex: "c", FirstSeen: seedAt, LastSeen: seedAt, Present: true},
		{Kind: domain.Bookmark, Post: samplePost("2", "b", "two"), SortIndex: "b", FirstSeen: seedAt, LastSeen: seedAt, Present: true},
		{Kind: domain.Bookmark, Post: samplePost("3", "c", "three"), SortIndex: "a", FirstSeen: seedAt, LastSeen: seedAt, Present: true},
	}
	if _, err := s.Upsert(ctx, seed); err != nil {
		t.Fatalf("seed: %v", err)
	}
	var before int
	if err := s.r.QueryRow(`SELECT count(*) FROM saved_items WHERE kind = 'bookmark'`).Scan(&before); err != nil {
		t.Fatalf("count before: %v", err)
	}

	// Full walk returns only posts 1 and 2; 3 is gone upstream.
	fetch := &fakeFetcher{pages: []domain.PostPage{{
		Posts: []domain.Post{samplePost("1", "a", "one"), samplePost("2", "b", "two")},
	}}}
	m := NewMirror(s, fetch, &fakeViewer{user: domain.User{ScreenName: "aergon"}})
	run, err := m.Begin(ctx, domain.Bookmark, true)
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	prog := walkAll(t, run)
	if prog.Stop != domain.StopEndOfList {
		t.Fatalf("stop = %v, want EndOfList", prog.Stop)
	}
	if prog.Absent != 1 {
		t.Fatalf("Absent = %d, want 1", prog.Absent)
	}

	// (a) third row still in the file — count through the test's own read-only handle.
	var after int
	if err := s.r.QueryRow(`SELECT count(*) FROM saved_items WHERE kind = 'bookmark'`).Scan(&after); err != nil {
		t.Fatalf("count after: %v", err)
	}
	if after != before || after != 3 {
		t.Fatalf("rows before/after = %d/%d, want 3/3 (never-delete)", before, after)
	}

	// (b) Page no longer returns it; (c) Status.Absent is 1.
	page, err := m.Page(ctx, domain.Bookmark, "")
	if err != nil {
		t.Fatalf("Page: %v", err)
	}
	if len(page.Posts) != 2 {
		t.Fatalf("Page returned %d, want 2", len(page.Posts))
	}
	for _, p := range page.Posts {
		if p.ID == "3" {
			t.Fatal("Page still returns the absent post")
		}
	}
	st, err := m.Status(ctx, domain.Bookmark)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if st.Absent != 1 || st.Present != 2 {
		t.Fatalf("Status present/absent = %d/%d, want 2/1", st.Present, st.Absent)
	}
	if st.LastSuccess.IsZero() {
		t.Fatal("full walk did not move LastSuccess")
	}

	// (e) later incremental walk that re-sees post 3 restores present = 1.
	refetch := &fakeFetcher{pages: []domain.PostPage{{
		Posts: []domain.Post{samplePost("3", "c", "three"), samplePost("1", "a", "one")},
		Next:  "c1",
	}}}
	m3 := NewMirror(s, refetch, &fakeViewer{user: domain.User{ScreenName: "aergon"}})
	run3, err := m3.Begin(ctx, domain.Bookmark, false)
	if err != nil {
		t.Fatalf("Begin restore: %v", err)
	}
	_ = walkAll(t, run3)
	page, err = m3.Page(ctx, domain.Bookmark, "")
	if err != nil {
		t.Fatalf("Page after restore: %v", err)
	}
	found := false
	for _, p := range page.Posts {
		if p.ID == "3" {
			found = true
		}
	}
	if !found {
		t.Fatal("post 3 did not return to the reader after being re-seen")
	}
}

func TestWalkPageCap(t *testing.T) {
	s := openTest(t)
	ctx := context.Background()
	freezeNow(t, time.Date(2026, 10, 14, 20, 0, 0, 0, time.UTC))

	fetch := &fakeFetcher{infinite: true}
	m := NewMirror(s, fetch, &fakeViewer{user: domain.User{ScreenName: "aergon"}})
	run, err := m.Begin(ctx, domain.Bookmark, true)
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	prog := walkAll(t, run)
	if prog.Stop != domain.StopPageCap {
		t.Fatalf("stop = %v, want StopPageCap", prog.Stop)
	}
	if prog.Pages != pageCap || fetch.calls != pageCap {
		t.Fatalf("pages/calls = %d/%d, want %d", prog.Pages, fetch.calls, pageCap)
	}
	st, err := m.Status(ctx, domain.Bookmark)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if st.Present != pageCap {
		t.Fatalf("Present = %d, want %d", st.Present, pageCap)
	}
	if st.LastSuccess.IsZero() {
		t.Fatal("StopPageCap must move LastSuccess")
	}
}

func TestWalkFetchFailureMidWalk(t *testing.T) {
	s := openTest(t)
	ctx := context.Background()
	freezeNow(t, time.Date(2026, 10, 14, 21, 0, 0, 0, time.UTC))

	// Seed a prior success so we can assert LastSuccess is unchanged.
	if err := s.RecordRun(ctx, domain.SyncResult{
		Kind: domain.Bookmark, Stop: domain.StopEndOfList,
		Started:  time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
		Finished: time.Date(2026, 10, 1, 0, 1, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("seed run: %v", err)
	}
	prev, err := s.Status(ctx, domain.Bookmark)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}

	fetch := &fakeFetcher{
		pages: []domain.PostPage{{
			Posts: []domain.Post{samplePost("1", "a", "one")},
			Next:  "c1",
		}},
		errAt: 2,
		err:   errors.New("page two failed"),
	}
	m := NewMirror(s, fetch, &fakeViewer{user: domain.User{ScreenName: "aergon"}})
	run, err := m.Begin(ctx, domain.Bookmark, false)
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	p1, err := run.Step(ctx)
	if err != nil || p1.Done {
		t.Fatalf("page one = %+v err=%v", p1, err)
	}
	_, err = run.Step(ctx)
	if err == nil {
		t.Fatal("page two should fail")
	}
	page, err := m.Page(ctx, domain.Bookmark, "")
	if err != nil {
		t.Fatalf("Page: %v", err)
	}
	if len(page.Posts) != 1 || page.Posts[0].ID != "1" {
		t.Fatalf("Page after failure = %v, want [1]", idsOf(page.Posts))
	}
	st, err := m.Status(ctx, domain.Bookmark)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if !st.LastSuccess.Equal(prev.LastSuccess) {
		t.Fatalf("LastSuccess moved on mid-walk failure")
	}
	var stops int
	if err := s.r.QueryRow(`SELECT count(*) FROM sync_runs WHERE stop_reason = ?`, domain.StopFailure.String()).Scan(&stops); err != nil || stops != 1 {
		t.Fatalf("failure runs = %d (%v)", stops, err)
	}
}

func TestWalkAbandonMarksNothing(t *testing.T) {
	s := openTest(t)
	ctx := context.Background()
	seedAt := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	runAt := time.Date(2026, 10, 14, 22, 0, 0, 0, time.UTC)
	freezeNow(t, runAt)

	if _, err := s.Upsert(ctx, []domain.SavedItem{
		{Kind: domain.Bookmark, Post: samplePost("old", "x", "old"), SortIndex: "a", FirstSeen: seedAt, LastSeen: seedAt, Present: true},
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	fetch := &fakeFetcher{pages: []domain.PostPage{{
		Posts: []domain.Post{samplePost("1", "a", "one")},
		Next:  "c1",
	}}}
	m := NewMirror(s, fetch, &fakeViewer{user: domain.User{ScreenName: "aergon"}})
	run, err := m.Begin(ctx, domain.Bookmark, true)
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	if _, err := run.Step(ctx); err != nil {
		t.Fatalf("Step: %v", err)
	}
	if err := run.Abandon(ctx, domain.StopAbandoned); err != nil {
		t.Fatalf("Abandon: %v", err)
	}
	var stop string
	var absent int
	if err := s.r.QueryRow(`SELECT stop_reason, absent FROM sync_runs ORDER BY id DESC LIMIT 1`).Scan(&stop, &absent); err != nil {
		t.Fatalf("sync_runs: %v", err)
	}
	if stop != domain.StopAbandoned.String() || absent != 0 {
		t.Fatalf("abandon row stop=%q absent=%d", stop, absent)
	}
	st, err := m.Status(ctx, domain.Bookmark)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if st.Absent != 0 {
		t.Fatalf("Absent = %d after abandon, want 0", st.Absent)
	}
	page, err := m.Page(ctx, domain.Bookmark, "")
	if err != nil {
		t.Fatalf("Page: %v", err)
	}
	foundOld := false
	for _, p := range page.Posts {
		if p.ID == "old" {
			foundOld = true
		}
	}
	if !foundOld {
		t.Fatal("abandoned full walk marked the unseen row absent")
	}
}

func TestOwnerRules(t *testing.T) {
	s := openTest(t)
	ctx := context.Background()
	freezeNow(t, time.Date(2026, 10, 14, 23, 0, 0, 0, time.UTC))

	fetch := &fakeFetcher{pages: []domain.PostPage{{
		Posts: []domain.Post{samplePost("1", "a", "one")},
	}}}
	m := NewMirror(s, fetch, &fakeViewer{user: domain.User{ScreenName: "aergon"}})
	run, err := m.Begin(ctx, domain.Bookmark, false)
	if err != nil {
		t.Fatalf("Begin fresh: %v", err)
	}
	_ = walkAll(t, run)
	st, err := m.Status(ctx, domain.Bookmark)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if st.Owner != "aergon" {
		t.Fatalf("Owner = %q, want aergon", st.Owner)
	}

	callsBefore := fetch.calls
	run2, err := m.Begin(ctx, domain.Bookmark, false)
	if err != nil {
		t.Fatalf("Begin same owner: %v", err)
	}
	_ = walkAll(t, run2)

	foreign := &fakeFetcher{pages: fetch.pages}
	m3 := NewMirror(s, foreign, &fakeViewer{user: domain.User{ScreenName: "kestrel"}})
	_, err = m3.Begin(ctx, domain.Bookmark, false)
	if !errors.Is(err, ErrWrongOwner) {
		t.Fatalf("Begin foreign = %v, want ErrWrongOwner", err)
	}
	if err == nil || !strings.Contains(err.Error(), "aergon") || !strings.Contains(err.Error(), "kestrel") {
		t.Fatalf("wrong-owner error %v must name both handles", err)
	}
	if foreign.calls != 0 {
		t.Fatalf("fetcher calls = %d after owner refusal, want 0", foreign.calls)
	}
	_ = callsBefore
	st, err = m.Status(ctx, domain.Bookmark)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if st.Owner != "aergon" {
		t.Fatalf("Owner after refusal = %q", st.Owner)
	}
}

func TestWalkStepAfterDoneIsNoop(t *testing.T) {
	s := openTest(t)
	ctx := context.Background()
	freezeNow(t, time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC))
	fetch := &fakeFetcher{pages: []domain.PostPage{{
		Posts: []domain.Post{samplePost("1", "a", "one")},
	}}}
	m := NewMirror(s, fetch, &fakeViewer{user: domain.User{ScreenName: "aergon"}})
	run, err := m.Begin(ctx, domain.Bookmark, false)
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	first := walkAll(t, run)
	calls := fetch.calls
	second, err := run.Step(ctx)
	if err != nil {
		t.Fatalf("Step after Done: %v", err)
	}
	if fetch.calls != calls {
		t.Fatalf("Step after Done issued a request")
	}
	if second != first {
		t.Fatalf("progress after Done = %+v, want %+v", second, first)
	}
}

func TestWalkRepeatedCursorStopsStalled(t *testing.T) {
	s := openTest(t)
	ctx := context.Background()
	freezeNow(t, time.Date(2026, 10, 15, 1, 0, 0, 0, time.UTC))
	fetch := &fakeFetcher{pages: []domain.PostPage{
		{Posts: []domain.Post{samplePost("1", "a", "one")}, Next: "same"},
		{Posts: []domain.Post{samplePost("2", "b", "two")}, Next: "same"},
	}}
	m := NewMirror(s, fetch, &fakeViewer{user: domain.User{ScreenName: "aergon"}})
	run, err := m.Begin(ctx, domain.Bookmark, true)
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	prog := walkAll(t, run)
	if prog.Stop != domain.StopStalled {
		t.Fatalf("stop = %v, want StopStalled", prog.Stop)
	}
	if fetch.calls != 2 {
		t.Fatalf("calls = %d, want 2", fetch.calls)
	}
}

// pageSet cuts ids into pages of PageSize with the cursor chain a walk follows:
// every page but the last carries the cursor that fetches the next one.
func pageSet(ids []string) []domain.PostPage {
	pages := make([]domain.PostPage, 0, (len(ids)+PageSize-1)/PageSize)
	for start := 0; start < len(ids); start += PageSize {
		end := min(start+PageSize, len(ids))
		posts := make([]domain.Post, 0, end-start)
		for _, id := range ids[start:end] {
			posts = append(posts, samplePost(id, "a", "body "+id))
		}
		page := domain.PostPage{Posts: posts}
		if end < len(ids) {
			page.Next = domain.Cursor(fmt.Sprintf("c%d", end))
		}
		pages = append(pages, page)
	}
	return pages
}

// keyAudit counts a kind's rows and the distinct keys among them. Two rows of
// one kind sharing a key is the fault these tests exist for: Page keys its pages
// with `sort_index < cursor`, so one of the pair is never returned at all.
func keyAudit(t *testing.T, s *Store) (rows, distinct int) {
	t.Helper()
	if err := s.r.QueryRow(`SELECT count(*), count(DISTINCT sort_index) FROM saved_items`).
		Scan(&rows, &distinct); err != nil {
		t.Fatalf("key audit: %v", err)
	}
	return rows, distinct
}

// readAll walks the reader the way the pane does — PageSize pages, each keyed by
// the last one — and returns every id it was handed, once per id, with the page
// count. It fails on a repeat, which is what a page boundary landing inside a
// tie produces.
func readAll(t *testing.T, m *Mirror, kind domain.SavedKind) ([]domain.PostID, int) {
	t.Helper()
	var (
		ids  []domain.PostID
		seen = make(map[domain.PostID]bool)
		next domain.Cursor
	)
	for pages := 1; ; pages++ {
		if pages > 100 {
			t.Fatalf("reader walk did not end after %d pages", pages)
		}
		page, err := m.Page(context.Background(), kind, next)
		if err != nil {
			t.Fatalf("Page: %v", err)
		}
		for _, p := range page.Posts {
			if seen[p.ID] {
				t.Fatalf("reader returned post %q twice", p.ID)
			}
			seen[p.ID] = true
			ids = append(ids, p.ID)
		}
		if page.Next == "" {
			return ids, pages
		}
		next = page.Next
	}
}

// oldRuleKey is the key the pre-fix walk wrote at a given position: a counter
// that restarted at zero every run, so the newest pages of any later run were
// written with keys the full run had already given the rows just below them.
func oldRuleKey(pos int) string { return fmt.Sprintf("%010d", 1_000_000_000-pos) }

// ids runs p000..p(n-1), the list the full walks in these tests read.
func ids(n int) []string {
	out := make([]string, 0, n)
	for i := range n {
		out = append(out, fmt.Sprintf("p%03d", i))
	}
	return out
}

// movedList is the list after three posts were saved on top of it: the three new
// ids, then the rows the previous walk put at 0..36. Read as two pages, the
// second of which adds nothing the mirror does not hold, it is exactly the
// incremental walk that reproduced the audit's defect: 40 rows re-fetched over a
// mirror whose keys were handed out 40 to a band.
func movedList() []domain.PostPage {
	inc := append([]string{"n0", "n1", "n2"}, ids(800)[:37]...)
	return []domain.PostPage{
		{Posts: postsOf(inc[:PageSize]), Next: "c20"},
		{Posts: postsOf(inc[PageSize:]), Next: "c40"},
	}
}

func postsOf(ids []string) []domain.Post {
	posts := make([]domain.Post, 0, len(ids))
	for _, id := range ids {
		posts = append(posts, samplePost(id, "a", "body "+id))
	}
	return posts
}

// TestWalkBandsKeepSortIndexUnique is the audit's own case, end to end: a full
// walk of the widest list there is (pageCap pages of PageSize rows), then an
// incremental walk over the two newest pages of a list that moved. The rows the
// incremental run re-writes must take keys of their own — under the old rule
// they took the keys the full run had given rows 37..39, two rows shared a key,
// and the reader, which pages with `sort_index < cursor`, lost the row that fell
// just below a page boundary. Every present row must come back exactly once.
func TestWalkBandsKeepSortIndexUnique(t *testing.T) {
	s := openTest(t)
	ctx := context.Background()
	freezeNow(t, time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC))

	full := &fakeFetcher{pages: pageSet(ids(800))}
	m := NewMirror(s, full, &fakeViewer{user: domain.User{ScreenName: "aergon"}})
	run, err := m.Begin(ctx, domain.Bookmark, true)
	if err != nil {
		t.Fatalf("Begin full: %v", err)
	}
	prog := walkAll(t, run)
	if prog.Stop != domain.StopEndOfList || prog.Added != 800 {
		t.Fatalf("full walk = %v added %d, want end of list with 800", prog.Stop, prog.Added)
	}
	if rows, distinct := keyAudit(t, s); rows != 800 || distinct != 800 {
		t.Fatalf("after the full walk: %d rows, %d distinct keys, want 800/800", rows, distinct)
	}

	inc := &fakeFetcher{pages: movedList()}
	mi := NewMirror(s, inc, &fakeViewer{user: domain.User{ScreenName: "aergon"}})
	runI, err := mi.Begin(ctx, domain.Bookmark, false)
	if err != nil {
		t.Fatalf("Begin incremental: %v", err)
	}
	progI := walkAll(t, runI)
	if progI.Stop != domain.StopNoUnknownIDs || progI.Added != 3 || inc.calls != 2 {
		t.Fatalf("incremental walk = %v added %d in %d calls, want no unknown ids, 3, 2", progI.Stop, progI.Added, inc.calls)
	}

	// The fault, stated as the file sees it: 803 present rows, and no key handed
	// out twice.
	if rows, distinct := keyAudit(t, s); rows != 803 || distinct != 803 {
		t.Fatalf("after the incremental walk: %d rows, %d distinct keys, want 803/803", rows, distinct)
	}

	got, pages := readAll(t, mi, domain.Bookmark)
	if len(got) != 803 {
		t.Fatalf("reader returned %d of 803 present rows, want every one", len(got))
	}
	if pages != 41 {
		t.Fatalf("reader pages = %d, want 41 (40 of %d and a last of 3)", pages, PageSize)
	}
	if want := []domain.PostID{"n0", "n1", "n2", "p000"}; !slices.Equal(got[:4], want) {
		t.Fatalf("reader starts %v, want the re-fetched rows leading: %v", got[:4], want)
	}
}

// TestFileWrittenUnderTheOldRuleStillReads: a mirror a pre-fix build left behind
// holds keys from the old per-run counter, and nothing migrates them — they are
// already unique and already in order, so the reader reads them as they are, and
// the next run's band simply starts above them. What must not happen is the new
// walk tying with them, or rewriting rows it did not fetch.
func TestFileWrittenUnderTheOldRuleStillReads(t *testing.T) {
	s := openTest(t)
	ctx := context.Background()
	seen := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	freezeNow(t, time.Date(2026, 10, 14, 13, 0, 0, 0, time.UTC))

	legacy := make([]domain.SavedItem, 0, 800)
	for pos, id := range ids(800) {
		legacy = append(legacy, domain.SavedItem{
			Kind: domain.Bookmark, Post: samplePost(id, "a", "body "+id),
			SortIndex: oldRuleKey(pos), FirstSeen: seen, LastSeen: seen, Present: true,
		})
	}
	if _, err := s.Upsert(ctx, legacy); err != nil {
		t.Fatalf("seed: %v", err)
	}
	m := NewMirror(s, &fakeFetcher{}, &fakeViewer{user: domain.User{ScreenName: "aergon"}})
	got, pages := readAll(t, m, domain.Bookmark)
	if len(got) != 800 || pages != 40 {
		t.Fatalf("old-rule file: reader returned %d rows over %d pages, want 800/40", len(got), pages)
	}

	inc := &fakeFetcher{pages: movedList()}
	mi := NewMirror(s, inc, &fakeViewer{user: domain.User{ScreenName: "aergon"}})
	run, err := mi.Begin(ctx, domain.Bookmark, false)
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	if prog := walkAll(t, run); prog.Added != 3 {
		t.Fatalf("incremental added = %d, want 3", prog.Added)
	}
	if rows, distinct := keyAudit(t, s); rows != 803 || distinct != 803 {
		t.Fatalf("after the incremental walk: %d rows, %d distinct keys, want 803/803", rows, distinct)
	}

	// A row the incremental walk did not fetch keeps the key it had: nothing was
	// renumbered to make room.
	var key string
	if err := s.r.QueryRow(`SELECT sort_index FROM saved_items WHERE post_id = 'p500'`).Scan(&key); err != nil {
		t.Fatalf("select: %v", err)
	}
	if key != oldRuleKey(500) {
		t.Fatalf("untouched row's key = %q, want it left alone at %q", key, oldRuleKey(500))
	}

	got, _ = readAll(t, mi, domain.Bookmark)
	if len(got) != 803 {
		t.Fatalf("reader returned %d of 803 present rows, want every one", len(got))
	}
	if want := []domain.PostID{"n0", "n1", "n2", "p000"}; !slices.Equal(got[:4], want) {
		t.Fatalf("reader starts %v, want the re-fetched rows above the old keys: %v", got[:4], want)
	}
}
