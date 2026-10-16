package cli

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/render"
	"github.com/LZD121/x-terminal/internal/store"
	"github.com/LZD121/x-terminal/internal/xweb"
	"github.com/LZD121/x-terminal/internal/xweb/volatile"
	_ "modernc.org/sqlite"
)

// syncFakeFetcher is the verb test's SavedFetcher: a slice of pages per kind
// plus a call counter. No network.
type syncFakeFetcher struct {
	pages map[domain.SavedKind][]domain.PostPage
	calls map[domain.SavedKind]int
	errAt map[domain.SavedKind]int
	errs  map[domain.SavedKind]error
}

func newSyncFake() *syncFakeFetcher {
	return &syncFakeFetcher{
		pages: map[domain.SavedKind][]domain.PostPage{},
		calls: map[domain.SavedKind]int{},
		errAt: map[domain.SavedKind]int{},
		errs:  map[domain.SavedKind]error{},
	}
}

func (f *syncFakeFetcher) Saved(_ context.Context, kind domain.SavedKind, _ domain.Cursor) (domain.PostPage, error) {
	f.calls[kind]++
	if at := f.errAt[kind]; at > 0 && f.calls[kind] == at {
		err := f.errs[kind]
		if err == nil {
			err = errors.New("fetch failed")
		}
		return domain.PostPage{}, err
	}
	pages := f.pages[kind]
	i := f.calls[kind] - 1
	if i >= len(pages) {
		return domain.PostPage{}, nil
	}
	return pages[i], nil
}

type syncFakeViewer struct {
	user  domain.User
	calls int
}

func (v *syncFakeViewer) Viewer(context.Context) (domain.User, error) {
	v.calls++
	return v.user, nil
}

func syncPost(id, text string) domain.Post {
	return domain.Post{
		ID:        domain.PostID(id),
		Author:    domain.User{ID: "u" + id, ScreenName: "author", Name: "Author"},
		Text:      text,
		CreatedAt: time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC),
	}
}

// withOpenMirror replaces openMirror for one test: a temp store driven by the
// given fetcher and viewer, no network.
func withOpenMirror(t *testing.T, fetch store.SavedFetcher, viewer store.Viewer) string {
	t.Helper()
	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)
	path := writeSession(t, t.TempDir(), "auth_token="+fixtureToken+"\nct0="+fixtureCTO+"\n")
	hermeticConfig(t, path)
	// hermeticConfig resets XDG_DATA_HOME; put ours back so SavedDBPath lands in data.
	t.Setenv("XDG_DATA_HOME", data)

	prev := openMirror
	openMirror = func(p string, _ store.SavedFetcher, _ store.Viewer) (*store.Mirror, io.Closer, error) {
		st, err := store.Open(p)
		if err != nil {
			return nil, nil, err
		}
		return store.NewMirror(st, fetch, viewer), st, nil
	}
	t.Cleanup(func() { openMirror = prev })
	return filepath.Join(data, "x-terminal", "saved.db")
}

func TestSyncIncrementalStopsEarly(t *testing.T) {
	fetch := newSyncFake()
	fetch.pages[domain.Bookmark] = []domain.PostPage{
		{Posts: []domain.Post{syncPost("1", "alpha body"), syncPost("2", "beta body")}, Next: "c1"},
		{Posts: []domain.Post{syncPost("1", "alpha body"), syncPost("2", "beta body")}, Next: "c2"},
	}
	// Likes empty end-of-list so both kinds succeed.
	fetch.pages[domain.Like] = []domain.PostPage{{}}
	viewer := &syncFakeViewer{user: domain.User{ScreenName: "aergon", ID: "1"}}
	wantPath := withOpenMirror(t, fetch, viewer)

	code, stdout, stderr := run(t, Sync, "--kind", "bookmarks")
	if code != ExitOK {
		t.Fatalf("sync exited %d (%s)", code, stderr)
	}
	if fetch.calls[domain.Bookmark] != 2 {
		t.Fatalf("bookmark calls = %d, want 2", fetch.calls[domain.Bookmark])
	}
	if !strings.Contains(stdout, "bookmarks") {
		t.Fatalf("stdout missing bookmarks: %q", stdout)
	}
	if !strings.Contains(stdout, "items 2") {
		t.Fatalf("stdout missing items 2: %q", stdout)
	}
	if !strings.Contains(stdout, domain.StopNoUnknownIDs.String()) {
		t.Fatalf("stdout missing stop reason: %q", stdout)
	}
	if !strings.Contains(stdout, wantPath) {
		t.Fatalf("stdout missing path %q: %q", wantPath, stdout)
	}
	if _, err := os.Stat(wantPath); err != nil {
		t.Fatalf("mirror file missing: %v", err)
	}

	// Second run adds nothing.
	callsBefore := fetch.calls[domain.Bookmark]
	code, stdout, stderr = run(t, Sync, "--kind", "bookmarks")
	if code != ExitOK {
		t.Fatalf("second sync exited %d (%s)", code, stderr)
	}
	if fetch.calls[domain.Bookmark]-callsBefore != 1 {
		t.Fatalf("second run calls = %d, want 1", fetch.calls[domain.Bookmark]-callsBefore)
	}
	if !strings.Contains(stdout, "items 0") {
		t.Fatalf("second run stdout %q, want items 0", stdout)
	}
}

func TestSyncFullMarksAbsent(t *testing.T) {
	fetch := newSyncFake()
	fetch.pages[domain.Bookmark] = []domain.PostPage{
		{Posts: []domain.Post{syncPost("1", "one")}, Next: "c1"},
		{Posts: []domain.Post{syncPost("2", "two")}, Next: "c2"},
		{Posts: []domain.Post{syncPost("3", "three")}},
	}
	viewer := &syncFakeViewer{user: domain.User{ScreenName: "aergon", ID: "1"}}
	path := withOpenMirror(t, fetch, viewer)

	// Seed a row the full walk will not see.
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	st, err := store.Open(path)
	if err != nil {
		t.Fatalf("Open seed: %v", err)
	}
	seedAt := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	if _, err := st.Upsert(context.Background(), []domain.SavedItem{{
		Kind: domain.Bookmark, Post: syncPost("gone", "seeded absent text"),
		SortIndex: "0", FirstSeen: seedAt, LastSeen: seedAt, Present: true,
	}}); err != nil {
		st.Close()
		t.Fatalf("seed: %v", err)
	}
	st.Close()

	code, stdout, stderr := run(t, Sync, "--full", "--kind", "bookmarks")
	if code != ExitOK {
		t.Fatalf("sync --full exited %d (%s)", code, stderr)
	}
	if !strings.Contains(stdout, "absent 1") {
		t.Fatalf("stdout %q, want absent 1", stdout)
	}

	// Seeded row still in the file.
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro")
	if err != nil {
		t.Fatalf("sql open: %v", err)
	}
	defer db.Close()
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM saved_items WHERE post_id = 'gone'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("gone row count = %d (%v), want 1", n, err)
	}
}

func TestSyncFailingFetchExitsOneWithoutPostText(t *testing.T) {
	const marker = "UNIQUE_POST_BODY_MUST_NOT_APPEAR"
	fetch := newSyncFake()
	fetch.pages[domain.Bookmark] = []domain.PostPage{
		{Posts: []domain.Post{syncPost("1", marker)}, Next: "c1"},
	}
	fetch.errAt[domain.Bookmark] = 2
	fetch.errs[domain.Bookmark] = &xweb.APIError{Outcome: xweb.OutcomeRateLimited, Status: 429}
	viewer := &syncFakeViewer{user: domain.User{ScreenName: "aergon", ID: "1"}}
	withOpenMirror(t, fetch, viewer)

	code, stdout, stderr := run(t, Sync, "--kind", "bookmarks")
	if code != ExitError {
		t.Fatalf("sync exited %d, want %d", code, ExitError)
	}
	if want := render.StateWord(domain.FailureRateLimited); !strings.Contains(stderr, want) {
		t.Fatalf("stderr %q, want named state %q", stderr, want)
	}
	if strings.Contains(stdout, marker) || strings.Contains(stderr, marker) {
		t.Fatalf("post text leaked into output:\nstdout=%q\nstderr=%q", stdout, stderr)
	}
}

func TestSyncOwnerRefusal(t *testing.T) {
	fetch := newSyncFake()
	fetch.pages[domain.Bookmark] = []domain.PostPage{
		{Posts: []domain.Post{syncPost("1", "one")}},
	}
	first := &syncFakeViewer{user: domain.User{ScreenName: "aergon", ID: "1"}}
	path := withOpenMirror(t, fetch, first)

	code, stdout, stderr := run(t, Sync, "--kind", "bookmarks")
	if code != ExitOK {
		t.Fatalf("first sync exited %d (%s)", code, stderr)
	}
	callsAfterFirst := fetch.calls[domain.Bookmark]

	// Re-open with a different viewer; fetcher must not move.
	foreign := &syncFakeViewer{user: domain.User{ScreenName: "kestrel", ID: "2"}}
	prev := openMirror
	openMirror = func(p string, _ store.SavedFetcher, _ store.Viewer) (*store.Mirror, io.Closer, error) {
		st, err := store.Open(p)
		if err != nil {
			return nil, nil, err
		}
		return store.NewMirror(st, fetch, foreign), st, nil
	}
	t.Cleanup(func() { openMirror = prev })

	code, stdout, stderr = run(t, Sync, "--kind", "bookmarks")
	if code != ExitError {
		t.Fatalf("foreign sync exited %d, want %d (stdout %q)", code, ExitError, stdout)
	}
	if !strings.Contains(stderr, "aergon") || !strings.Contains(stderr, "kestrel") {
		t.Fatalf("stderr %q must name both handles", stderr)
	}
	if fetch.calls[domain.Bookmark] != callsAfterFirst {
		t.Fatalf("fetcher moved after owner refusal: %d -> %d", callsAfterFirst, fetch.calls[domain.Bookmark])
	}
	_ = path
}

func TestSyncKindLikesUnknownOpLeavesBookmarks(t *testing.T) {
	fetch := newSyncFake()
	fetch.pages[domain.Bookmark] = []domain.PostPage{
		{Posts: []domain.Post{syncPost("b1", "bookmark stays")}},
	}
	fetch.errAt[domain.Like] = 1
	fetch.errs[domain.Like] = fmt.Errorf("%w: Likes", volatile.ErrUnknownOperation)
	viewer := &syncFakeViewer{user: domain.User{ScreenName: "aergon", ID: "1"}}
	path := withOpenMirror(t, fetch, viewer)

	// Seed bookmarks first so we can assert they survive a likes failure.
	code, stdout, stderr := run(t, Sync, "--kind", "bookmarks")
	if code != ExitOK {
		t.Fatalf("bookmarks seed exited %d (%s)", code, stderr)
	}
	var before int
	{
		db, err := sql.Open("sqlite", "file:"+path+"?mode=ro")
		if err != nil {
			t.Fatalf("sql: %v", err)
		}
		if err := db.QueryRow(`SELECT count(*) FROM saved_items WHERE kind = 'bookmark'`).Scan(&before); err != nil {
			db.Close()
			t.Fatalf("count: %v", err)
		}
		db.Close()
	}

	code, stdout, stderr = run(t, Sync, "--kind", "likes")
	if code != ExitError {
		t.Fatalf("likes sync exited %d, want %d", code, ExitError)
	}
	if want := render.StateWord(domain.FailureOperationStale); !strings.Contains(stderr, want) && !strings.Contains(stderr, "unknown") {
		// ErrUnknownOperation maps through failureFor as unclassified unless it's an *APIError.
		// Either the named state or the error's own words are acceptable presence facts.
		if !strings.Contains(stderr, "Likes") && !strings.Contains(stderr, "unknown") {
			t.Fatalf("stderr %q, want a named likes failure", stderr)
		}
	}
	var after int
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro")
	if err != nil {
		t.Fatalf("sql: %v", err)
	}
	defer db.Close()
	if err := db.QueryRow(`SELECT count(*) FROM saved_items WHERE kind = 'bookmark'`).Scan(&after); err != nil {
		t.Fatalf("count: %v", err)
	}
	if after != before {
		t.Fatalf("bookmarks rows changed %d -> %d after likes failure", before, after)
	}
	_ = stdout
}

func TestSyncQuietAndJSON(t *testing.T) {
	fetch := newSyncFake()
	fetch.pages[domain.Bookmark] = []domain.PostPage{
		{Posts: []domain.Post{syncPost("1", "one")}},
	}
	viewer := &syncFakeViewer{user: domain.User{ScreenName: "aergon", ID: "1"}}
	withOpenMirror(t, fetch, viewer)

	code, stdout, stderr := run(t, Sync, "--kind", "bookmarks", "--quiet")
	if code != ExitOK {
		t.Fatalf("quiet sync exited %d (%s)", code, stderr)
	}
	if stdout != "" {
		t.Fatalf("--quiet printed %q", stdout)
	}

	code, stdout, stderr = run(t, Sync, "--kind", "bookmarks", "--json")
	if code != ExitOK {
		t.Fatalf("json sync exited %d (%s)", code, stderr)
	}
	var obj syncJSON
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &obj); err != nil {
		t.Fatalf("json decode %q: %v", stdout, err)
	}
	if obj.Kind != "bookmarks" || obj.Pages < 1 || obj.Stop == "" || obj.Path == "" {
		t.Fatalf("json object = %+v, want kind/pages/stop/path", obj)
	}
}

func TestSyncUnknownKindIsUsage(t *testing.T) {
	hermeticConfig(t, filepath.Join(t.TempDir(), "session"))
	code, stdout, stderr := run(t, Sync, "--kind", "timeline")
	if code != ExitUsage {
		t.Fatalf("exited %d, want %d", code, ExitUsage)
	}
	if stdout != "" {
		t.Fatalf("usage path wrote stdout: %q", stdout)
	}
	if !strings.Contains(stderr, "usage: x-terminal") {
		t.Fatalf("stderr missing usage: %q", stderr)
	}
}
