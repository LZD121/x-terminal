package store

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/LZD121/x-terminal/internal/domain"
)

func TestReadFactsReportsWhatTheWriterPut(t *testing.T) {
	path := filepath.Join(t.TempDir(), "saved.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()
	ctx := context.Background()
	if err := s.SetOwner(ctx, "aergon"); err != nil {
		t.Fatalf("SetOwner: %v", err)
	}
	now := time.Date(2026, 10, 14, 9, 12, 0, 0, time.UTC)
	items := []domain.SavedItem{{
		Kind: domain.Bookmark, SortIndex: "10", FirstSeen: now, LastSeen: now, Present: true,
		Post: domain.Post{ID: "1", Text: "hello", Author: domain.User{ScreenName: "a", ID: "1"}, CreatedAt: now},
	}, {
		Kind: domain.Like, SortIndex: "11", FirstSeen: now, LastSeen: now, Present: true,
		Post: domain.Post{ID: "2", Text: "liked", Author: domain.User{ScreenName: "b", ID: "2"}, CreatedAt: now},
	}}
	if _, err := s.Upsert(ctx, items); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if err := s.RecordRun(ctx, domain.SyncResult{
		Kind: domain.Bookmark, Pages: 1, Added: 1, Stop: domain.StopNoUnknownIDs,
		Started: now, Finished: now,
	}); err != nil {
		t.Fatalf("RecordRun: %v", err)
	}

	facts, err := ReadFacts(path)
	if err != nil {
		t.Fatalf("ReadFacts: %v", err)
	}
	if !facts.Exists || facts.Newer || facts.SchemaVersion != 1 {
		t.Fatalf("facts = %+v, want exists schema 1", facts)
	}
	if facts.Owner != "aergon" {
		t.Fatalf("Owner = %q", facts.Owner)
	}
	if len(facts.Kinds) != 2 {
		t.Fatalf("Kinds = %v", facts.Kinds)
	}
	if facts.Kinds[0].Kind != domain.Bookmark || facts.Kinds[0].Present != 1 || facts.Kinds[0].Absent != 0 {
		t.Fatalf("bookmark counts = %+v", facts.Kinds[0])
	}
	if facts.Kinds[1].Kind != domain.Like || facts.Kinds[1].Present != 1 {
		t.Fatalf("like counts = %+v", facts.Kinds[1])
	}
	if !facts.LastRun.Equal(now) || facts.Stop != domain.StopNoUnknownIDs {
		t.Fatalf("LastRun=%s Stop=%s", facts.LastRun, facts.Stop)
	}
}

func TestReadFactsMissingIsPresenceNotFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "no-such.db")
	facts, err := ReadFacts(path)
	if err != nil {
		t.Fatalf("ReadFacts: %v", err)
	}
	if facts.Exists || facts.Path != path {
		t.Fatalf("facts = %+v", facts)
	}
}

func TestReadFactsNewerSchemaIsNotAnError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "saved.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", dsn(path, ""))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`UPDATE meta SET value = '2' WHERE key = 'schema_version'`); err != nil {
		t.Fatal(err)
	}

	facts, err := ReadFacts(path)
	if err != nil {
		t.Fatalf("ReadFacts: %v", err)
	}
	if !facts.Exists || !facts.Newer || facts.SchemaVersion != 2 {
		t.Fatalf("facts = %+v, want Newer with version 2", facts)
	}
}

func TestReadFactsCorruptIsNamedError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "saved.db")
	if err := os.WriteFile(path, []byte("not a sqlite file"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := ReadFacts(path)
	if err == nil {
		t.Fatal("ReadFacts accepted a corrupt file")
	}
	if got := err.Error(); len(got) > 200 || !strings.Contains(got, path) {
		t.Fatalf("error %q: want path and a bounded reason", got)
	}
}

func TestReadFactsLeavesFileUnchanged(t *testing.T) {
	path := filepath.Join(t.TempDir(), "saved.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := s.SetOwner(context.Background(), "aergon"); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	mtime := info.ModTime()

	if _, err := ReadFacts(path); err != nil {
		t.Fatalf("ReadFacts: %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("ReadFacts changed the mirror bytes")
	}
	info, err = os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(mtime) {
		t.Fatalf("mtime moved from %s to %s", mtime, info.ModTime())
	}
}
