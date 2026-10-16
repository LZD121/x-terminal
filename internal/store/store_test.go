package store

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/LZD121/x-terminal/internal/domain"
)

func openTest(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "saved.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open(%s): %v", path, err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func samplePost(id, handle, text string) domain.Post {
	return domain.Post{
		ID: domain.PostID(id),
		Author: domain.User{
			ID:         "uid-" + id,
			ScreenName: handle,
			Name:       handle,
		},
		Text:      text,
		CreatedAt: time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC),
		Lang:      "en",
	}
}

func TestRoundTripOneSavedItem(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "saved.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	if got := s.Path(); got != path {
		t.Errorf("Path() = %q, want %q", got, path)
	}

	var mode string
	if err := s.w.QueryRow(`PRAGMA journal_mode`).Scan(&mode); err != nil {
		t.Fatalf("journal_mode: %v", err)
	}
	if mode != "wal" {
		t.Fatalf("journal_mode = %q, want wal", mode)
	}

	ctx := context.Background()
	ver, err := schemaVersion(ctx, s.w)
	if err != nil {
		t.Fatalf("schemaVersion: %v", err)
	}
	if ver != 1 {
		t.Fatalf("schema version = %d, want 1", ver)
	}

	frozen := time.Date(2026, 10, 14, 15, 0, 0, 0, time.UTC)
	old := now
	now = func() time.Time { return frozen }
	defer func() { now = old }()

	seen := time.Date(2026, 10, 14, 14, 0, 0, 0, time.UTC)
	item := domain.SavedItem{
		Kind:      domain.Bookmark,
		Post:      samplePost("100", "aergon", "mirror holds the post body"),
		SortIndex: "sort-100",
		FirstSeen: seen,
		LastSeen:  seen,
		Present:   true,
	}
	added, err := s.Upsert(ctx, []domain.SavedItem{item})
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if added != 1 {
		t.Fatalf("added = %d, want 1", added)
	}

	page, err := s.Page(ctx, domain.Bookmark, "")
	if err != nil {
		t.Fatalf("Page: %v", err)
	}
	if len(page.Posts) != 1 {
		t.Fatalf("Page posts = %d, want 1", len(page.Posts))
	}
	got := page.Posts[0]
	if got.ID != "100" || got.Author.ID != "uid-100" || got.Text != "mirror holds the post body" {
		t.Fatalf("round-trip post = %+v", got)
	}
	if page.FetchedAt != frozen {
		t.Errorf("FetchedAt = %v, want %v", page.FetchedAt, frozen)
	}
	if page.Requested != 1 {
		t.Errorf("Requested = %d, want 1", page.Requested)
	}
	if page.Next != "" {
		t.Errorf("Next = %q, want empty on a short page", page.Next)
	}

	added, err = s.Upsert(ctx, []domain.SavedItem{item})
	if err != nil {
		t.Fatalf("second Upsert: %v", err)
	}
	if added != 0 {
		t.Fatalf("second Upsert added = %d, want 0", added)
	}

	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	names := map[string]bool{}
	for _, e := range entries {
		names[e.Name()] = true
	}
	if !names["saved.db"] {
		t.Fatalf("after Close, dir holds %v, want saved.db", names)
	}
	// WAL siblings may or may not remain after a clean close; either is fine as
	// long as the primary file is there and nothing else unexpected appears.
	for name := range names {
		switch name {
		case "saved.db", "saved.db-wal", "saved.db-shm":
		default:
			t.Errorf("unexpected file after Close: %s", name)
		}
	}
}

func TestOpenMissingDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "saved.db")
	s, err := Open(path)
	if err == nil {
		_ = s.Close()
		t.Fatal("Open into a missing directory succeeded; want a named error")
	}
	if s != nil {
		t.Fatal("Open returned a non-nil Store with an error")
	}
}

func TestOpenSetsQueryOnlyOnReader(t *testing.T) {
	s := openTest(t)
	_, err := s.r.Exec(`INSERT INTO meta(key, value) VALUES('probe', '1')`)
	if err == nil {
		t.Fatal("reader Exec succeeded; query_only must refuse writes")
	}
}

// TestTheFilesOwnReadFailuresCarryTheMark: Page and Status mark every failure of their own read as
// store.FileError, and the mark is what a caller maps onto "this file cannot be read". A read of a
// store that has been closed is a real failure of exactly that kind — the same shape a corrupt or
// unreadable file produces — and the arm also pins the other half of the mark's contract: the
// wrapped message is unchanged, so every surface that prints a store error prints what it printed
// before.
func TestTheFilesOwnReadFailuresCarryTheMark(t *testing.T) {
	path := filepath.Join(t.TempDir(), "saved.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open(%s): %v", path, err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	_, readErr := s.Page(context.Background(), domain.Bookmark, "")
	if readErr == nil {
		t.Fatal("a read of a closed store succeeded: this test has no read failure to check")
	}
	var file *FileError
	if !errors.As(readErr, &file) {
		t.Errorf("the read failure %v (%T) does not carry the file's mark", readErr, readErr)
	}
	if file != nil && file.Error() != file.Err.Error() {
		t.Errorf("the mark's message is %q, want the wrapped error's own %q: a surface that prints a store error must be unchanged", file.Error(), file.Err.Error())
	}
	if !strings.Contains(readErr.Error(), "store: page") {
		t.Errorf("the failure's message is %q, want the store's own text", readErr.Error())
	}
	// The mark is not the message: an error the walk fetched (a parse failure, say) is not the
	// file's, and errors.As over a plain error says so.
	if errors.As(errors.New("xsource: a shape this build cannot read"), &file) {
		t.Error("a plain error was read as a failure of the file")
	}
}

func rawOpen(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", dsn(path, ""))
	if err != nil {
		t.Fatalf("raw open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}
